"""Orquestacion de un nodo router: hilo de routing e hilo de forwarding en paralelo."""

from __future__ import annotations

import json
import logging
import queue
import socket
import threading
import time

from router.algorithms import dijkstra
from router.config import NodeConfig
from router.control import messages, routing_table
from router.control.flooding import LinkStateStore
from router.forwarding import network

logger = logging.getLogger("router.node")

HELLO_INTERVAL_S = 10
LSA_DELAY_AFTER_FIRST_HELLO_S = 5
CONVERGENCE_WAIT_S = 30


class Node:
    """Un router: hilo de escucha + hilo de routing + hilo de forwarding."""

    def __init__(self, config: NodeConfig, csv_path: str | None = None):
        self.config = config
        self.csv_path = csv_path or f"{config.name}_routing_table.csv"
        self.store = LinkStateStore(self_id=config.id)
        self.active_neighbors: set[str] = set()
        self.routes: dict[str, routing_table.Route] = {}
        self._routing_queue: queue.Queue = queue.Queue()
        self._forwarding_queue: queue.Queue = queue.Queue()
        self._first_hello_event = threading.Event()
        self._stop = threading.Event()

    def start(self) -> None:
        threading.Thread(target=self._listen_loop, daemon=True, name="listener").start()
        threading.Thread(target=self._hello_loop, daemon=True, name="hello").start()
        threading.Thread(target=self._routing_loop, daemon=True, name="routing").start()
        threading.Thread(target=self._forwarding_loop, daemon=True, name="forwarding").start()
        threading.Thread(target=self._convergence_timer, daemon=True, name="convergence").start()

    def run_forever(self) -> None:
        self.start()
        try:
            while not self._stop.is_set():
                time.sleep(0.5)
        except KeyboardInterrupt:
            logger.info("[ROUTER %s] Detenido (Ctrl-C).", self.config.name)
            self._stop.set()

    # -- red -----------------------------------------------------------
    def send_message(self, ip: str, port: int, message: dict) -> None:
        try:
            with socket.create_connection((ip, port), timeout=5) as sock:
                sock.sendall(messages.dumps(message).encode("utf-8"))
        except OSError as exc:
            logger.warning(
                "[NETWORK %s:%s] no se pudo enviar a %s:%s (%s)",
                self.config.ip, self.config.port, ip, port, exc,
            )

    def _listen_loop(self) -> None:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server:
            server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            server.bind((self.config.ip, self.config.port))
            server.listen()
            logger.info("[NETWORK %s:%s] Listening", self.config.ip, self.config.port)
            while not self._stop.is_set():
                conn, _ = server.accept()
                threading.Thread(target=self._handle_connection, args=(conn,), daemon=True).start()

    def _handle_connection(self, conn: socket.socket) -> None:
        with conn:
            line = _read_line(conn)
            if not line:
                return
            try:
                message = messages.loads(line)
            except json.JSONDecodeError:
                logger.warning("mensaje invalido descartado")
                return
            # El nodo tiene dos hilos en paralelo (routing / forwarding); esta
            # cola es lo unico que los conecta con el hilo que acepta sockets.
            if message.get("type") == messages.DATA:
                self._forwarding_queue.put(message)
            else:
                self._routing_queue.put(message)

    # -- plano de control (hilo de routing) -----------------------------
    def _hello_loop(self) -> None:
        while not self._stop.is_set():
            for neighbor in self.config.neighbors:
                self.send_message(neighbor.ip, neighbor.port, messages.build_hello(self.config.id))
            time.sleep(HELLO_INTERVAL_S)

    def _routing_loop(self) -> None:
        while not self._stop.is_set():
            message = self._routing_queue.get()
            if message["type"] == messages.HELLO:
                self._on_hello(message)
            elif message["type"] == messages.LSA:
                self._on_lsa(message)

    def _on_hello(self, message: dict) -> None:
        sender = message["from"]
        first_time = sender not in self.active_neighbors
        self.active_neighbors.add(sender)
        logger.info("[NETWORK %s:%s] HELLO reply from %s", self.config.ip, self.config.port, sender)
        if first_time and not self._first_hello_event.is_set():
            self._first_hello_event.set()
            logger.info("[ROUTER %s] Waiting %ss before building the LSA...",
                        self.config.name, LSA_DELAY_AFTER_FIRST_HELLO_S)
            threading.Timer(LSA_DELAY_AFTER_FIRST_HELLO_S, self._build_and_flood_own_lsa).start()

    def _build_and_flood_own_lsa(self) -> None:
        links = {
            neighbor.id: neighbor.cost
            for neighbor in self.config.neighbors
            if neighbor.id in self.active_neighbors
        }
        if self.config.attached_host is not None:
            links[self.config.attached_host.id] = self.config.attached_host.cost
        seq = self.store.next_seq()
        lsa = messages.build_lsa(self.config.id, seq, links, self.config.id)
        logger.info("[ROUTER %s] LSA: %s", self.config.name, lsa)
        self.store.record(self.config.id, seq, links)
        self._flood(lsa, exclude_id=None)

    def _on_lsa(self, message: dict) -> None:
        origin, seq, links, sender = message["origin"], message["seq"], message["links"], message["from"]
        if self.store.record(origin, seq, links):
            logger.info("[NETWORK %s:%s] LSA from %s stored -> flooding onward",
                        self.config.ip, self.config.port, origin)
            self._flood(message, exclude_id=sender)
        else:
            logger.info("[NETWORK %s:%s] Ignoring LSA from %s (own or already known)",
                        self.config.ip, self.config.port, origin)

    def _flood(self, lsa: dict, exclude_id: str | None) -> None:
        for neighbor in self.config.neighbors:
            if neighbor.id != exclude_id:
                self.send_message(neighbor.ip, neighbor.port, {**lsa, "from": self.config.id})

    # -- convergencia ------------------------------------------------------
    def _convergence_timer(self) -> None:
        time.sleep(CONVERGENCE_WAIT_S)
        graph = self.store.snapshot()
        logger.info("[ROUTER %s] Converged. Network graph: %s", self.config.name, graph)
        paths = dijkstra.shortest_paths(graph, self.config.id)
        logger.info("[ROUTER %s] Shortest paths (Dijkstra):", self.config.name)
        for destination, result in paths.items():
            logger.info("  %s: cost %s via %s", destination, result.cost, result.next_hop)
        self.routes = {
            destination: routing_table.Route(
                destination, self._ip_for(result.next_hop), self._port_for(result.next_hop), result.cost
            )
            for destination, result in paths.items()
        }
        routing_table.write(self.csv_path, self.routes)
        logger.info("[NETWORK %s:%s] Routing table written to %s",
                    self.config.ip, self.config.port, self.csv_path)
        logger.info("[ROUTER %s] Done (Ctrl-C to stop).", self.config.name)

    def _port_for(self, node_identity: str) -> int:
        """El primer salto siempre es un vecino directo (o el host adjunto)."""
        for neighbor in self.config.neighbors:
            if neighbor.id == node_identity:
                return neighbor.port
        if self.config.attached_host is not None and self.config.attached_host.id == node_identity:
            return self.config.attached_host.port
        return self.config.port

    def _ip_for(self, node_identity: str) -> str:
        for neighbor in self.config.neighbors:
            if neighbor.id == node_identity:
                return neighbor.ip
        if self.config.attached_host is not None and self.config.attached_host.id == node_identity:
            return self.config.attached_host.ip
        return self.config.ip

    # -- plano de datos (hilo de forwarding) ---------------------------------
    def _forwarding_loop(self) -> None:
        while not self._stop.is_set():
            message = self._forwarding_queue.get()
            network.forward(message, self.routes, self.send_message)


def _read_line(conn: socket.socket) -> str:
    chunks = bytearray()
    while True:
        chunk = conn.recv(4096)
        if not chunk:
            break
        chunks.extend(chunk)
        if b"\n" in chunk:
            break
    return chunks.decode("utf-8").strip()
