"""Nodo no-router (cliente o servidor): usa a su router adjunto como gateway.

Un host no participa en HELLO/LSA/Dijkstra; solo abre socket hacia su
router-gateway para enviar DATA, y escucha DATA que el router ya resolvio
hasta su IP/puerto (ver seccion 3.2 del enunciado: "puerta de enlace
predeterminada").
"""

from __future__ import annotations

import socket

from router.control import messages
from router.forwarding.network import decode_payload, encode_envelope


def send_via_gateway(self_id: str, gateway_ip: str, gateway_port: int, to_id: str, text: str) -> None:
    """Arma el DATA inicial y lo entrega al router-gateway local.

    self_id / to_id son identificadores "ip:puerto" (ver config.node_id),
    iguales a los que el router uso para este host en su LSA.
    """
    envelope = encode_envelope({"from": self_id, "to": to_id, "msg": text})
    with socket.create_connection((gateway_ip, gateway_port)) as sock:
        sock.sendall(messages.dumps(envelope).encode("utf-8"))


def run_server(listen_ip: str, listen_port: int, on_message) -> None:
    """Escucha DATA que el router-gateway local ya resolvio hasta este host."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server:
        server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server.bind((listen_ip, listen_port))
        server.listen()
        while True:
            conn, _ = server.accept()
            with conn:
                line = _read_line(conn)
                if not line:
                    continue
                payload = decode_payload(messages.loads(line))
                if payload is not None:
                    on_message(payload)


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
