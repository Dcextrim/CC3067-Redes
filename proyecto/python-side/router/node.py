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

# Si no llega HELLO de un vecino activo en este lapso (3 ciclos de HELLO), se
# marca como caido y se regenera el LSA propio.
NEIGHBOR_TIMEOUT_S = 3 * HELLO_INTERVAL_S
NEIGHBOR_CHECK_INTERVAL_S = 5

# Ademas de recalcular de inmediato cuando cambia el grafo (nuevo LSA, vecino
# caido/recuperado), se revisa con esta cadencia por si algun evento se perdio.
ROUTE_RECOMPUTE_INTERVAL_S = 15

# Evita floodear un LSA nuevo por cada cambio de vecino si varios ocurren juntos.
LSA_REBUILD_DEBOUNCE_S = 3


class Node:
    """Un router: hilo de escucha + hilo de routing + hilo de forwarding."""

    def __init__(
        self,
        config: NodeConfig,
        csv_path: str | None = None,
        *,
        hello_interval_s: float = HELLO_INTERVAL_S,
        lsa_delay_s: float = LSA_DELAY_AFTER_FIRST_HELLO_S,
        convergence_wait_s: float = CONVERGENCE_WAIT_S,
        neighbor_timeout_s: float = NEIGHBOR_TIMEOUT_S,
        neighbor_check_interval_s: float = NEIGHBOR_CHECK_INTERVAL_S,
        route_recompute_interval_s: float = ROUTE_RECOMPUTE_INTERVAL_S,
        lsa_rebuild_debounce_s: float = LSA_REBUILD_DEBOUNCE_S,
    ):
        self.config = config
        self.csv_path = csv_path or f"{config.name}_tabla_enrutamiento.csv"
        self.hello_interval_s = hello_interval_s
        self.lsa_delay_s = lsa_delay_s
        self.convergence_wait_s = convergence_wait_s
        self.neighbor_timeout_s = neighbor_timeout_s
        self.neighbor_check_interval_s = neighbor_check_interval_s
        self.route_recompute_interval_s = route_recompute_interval_s
        self.lsa_rebuild_debounce_s = lsa_rebuild_debounce_s

        self.store = LinkStateStore(self_id=config.id)
        self.active_neighbors: set[str] = set()
        self.last_hello_at: dict[str, float] = {}
        self.routes: dict[str, routing_table.Route] = {}
        self._routing_queue: queue.Queue = queue.Queue()
        self._forwarding_queue: queue.Queue = queue.Queue()
        self._first_hello_event = threading.Event()
        self._converged = False
        self._last_lsa_rebuild = 0.0
        # Protege active_neighbors/last_hello_at/routes/_converged/_last_lsa_rebuild:
        # se leen/escriben desde hilos distintos (routing, watchdog de vecinos,
        # convergencia y forwarding corren en paralelo).
        self._state_lock = threading.Lock()
        self._stop = threading.Event()

    def start(self) -> None:
        threading.Thread(target=self._listen_loop, daemon=True, name="listener").start()
        threading.Thread(target=self._hello_loop, daemon=True, name="hello").start()
        threading.Thread(target=self._routing_loop, daemon=True, name="routing").start()
        threading.Thread(target=self._forwarding_loop, daemon=True, name="forwarding").start()
        threading.Thread(target=self._neighbor_watchdog, daemon=True, name="neighbor-watchdog").start()
        threading.Thread(target=self._convergence_loop, daemon=True, name="convergence").start()

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
            time.sleep(self.hello_interval_s)

    def _routing_loop(self) -> None:
        while not self._stop.is_set():
            message = self._routing_queue.get()
            if message["type"] == messages.HELLO:
                self._on_hello(message)
            elif message["type"] == messages.LSA:
                self._on_lsa(message)

    def _on_hello(self, message: dict) -> None:
        sender = message["from"]
        with self._state_lock:
            self.last_hello_at[sender] = time.monotonic()
            first_time = sender not in self.active_neighbors
            self.active_neighbors.add(sender)
            is_very_first_hello = not self._first_hello_event.is_set()
        logger.info("[NETWORK %s:%s] HELLO reply from %s", self.config.ip, self.config.port, sender)
        if not first_time:
            return
        if is_very_first_hello:
            self._first_hello_event.set()
            logger.info("[ROUTER %s] Waiting %ss before building the LSA...",
                        self.config.name, self.lsa_delay_s)
            threading.Timer(self.lsa_delay_s, self._build_and_flood_own_lsa).start()
            return
        # Un vecino que ya habia caido volvio a responder: no es el primer
        # HELLO del nodo, pero cambia la topologia -> hay que avisar con un LSA nuevo.
        self._request_lsa_rebuild()

    def _neighbor_watchdog(self) -> None:
        """Vigila que los vecinos activos sigan enviando HELLO; si uno deja de
        hacerlo por neighbor_timeout_s, se marca caido y se pide un LSA nuevo."""
        while not self._stop.is_set():
            time.sleep(self.neighbor_check_interval_s)
            changed = False
            now = time.monotonic()
            with self._state_lock:
                for neighbor in self.config.neighbors:
                    if neighbor.id not in self.active_neighbors:
                        continue
                    last = self.last_hello_at.get(neighbor.id)
                    if last is None or now - last > self.neighbor_timeout_s:
                        self.active_neighbors.discard(neighbor.id)
                        changed = True
                        logger.info("[ROUTER %s] Neighbor %s expired (sin HELLO en %ss)",
                                    self.config.name, neighbor.id, self.neighbor_timeout_s)
            if changed:
                self._request_lsa_rebuild()

    def _request_lsa_rebuild(self) -> None:
        """Reconstruye y floodea el LSA propio, con un debounce minimo para no
        inundar la red si varios vecinos cambian a la vez."""
        with self._state_lock:
            elapsed = time.monotonic() - self._last_lsa_rebuild
        if elapsed < self.lsa_rebuild_debounce_s:
            threading.Timer(self.lsa_rebuild_debounce_s - elapsed, self._build_and_flood_own_lsa).start()
        else:
            self._build_and_flood_own_lsa()

    def _build_and_flood_own_lsa(self) -> None:
        with self._state_lock:
            self._last_lsa_rebuild = time.monotonic()
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
        self._maybe_recompute_routes()

    def _on_lsa(self, message: dict) -> None:
        origin, seq, links, sender = message["origin"], message["seq"], message["links"], message["from"]
        if self.store.record(origin, seq, links):
            logger.info("[NETWORK %s:%s] LSA from %s stored -> flooding onward",
                        self.config.ip, self.config.port, origin)
            self._flood(message, exclude_id=sender)
            self._maybe_recompute_routes()
        else:
            logger.info("[NETWORK %s:%s] Ignoring LSA from %s (own or already known)",
                        self.config.ip, self.config.port, origin)

    def _flood(self, lsa: dict, exclude_id: str | None) -> None:
        for neighbor in self.config.neighbors:
            if neighbor.id != exclude_id:
                self.send_message(neighbor.ip, neighbor.port, {**lsa, "from": self.config.id})

    # -- convergencia ------------------------------------------------------
    def _convergence_loop(self) -> None:
        """Espera la convergencia inicial exigida por el enunciado (30s) y
        calcula la primera tabla de ruteo; despues sigue recalculando cada vez
        que cambia el grafo (_maybe_recompute_routes) y, como respaldo, en
        cada tick de route_recompute_interval_s por si algun evento no
        disparo el recalculo."""
        time.sleep(self.convergence_wait_s)
        with self._state_lock:
            self._converged = True
        self._recompute_routes()
        logger.info("[ROUTER %s] Converged. Recomputando rutas ante cada cambio de topologia.", self.config.name)

        while not self._stop.is_set():
            time.sleep(self.route_recompute_interval_s)
            self._recompute_routes()

    def _maybe_recompute_routes(self) -> None:
        with self._state_lock:
            converged = self._converged
        if converged:
            self._recompute_routes()

    def _recompute_routes(self) -> None:
        graph = self.store.snapshot()
        paths = dijkstra.shortest_paths(graph, self.config.id)
        routes = {
            destination: routing_table.Route(
                destination, self._ip_for(result.next_hop), self._port_for(result.next_hop), result.cost
            )
            for destination, result in paths.items()
        }
        with self._state_lock:
            changed = routes != self.routes
            self.routes = routes
        if not changed:
            return

        logger.info("[ROUTER %s] Shortest paths (Dijkstra):", self.config.name)
        for destination, result in paths.items():
            logger.info("  %s: cost %s via %s", destination, result.cost, result.next_hop)
        routing_table.write(self.csv_path, routes)
        logger.info("[NETWORK %s:%s] Routing table written to %s",
                    self.config.ip, self.config.port, self.csv_path)

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
            with self._state_lock:
                routes = self.routes
            network.forward(message, routes, self.send_message)


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
