"""Construccion y parseo de los mensajes JSON del protocolo (docs/protocol.md)."""

from __future__ import annotations

import json

HELLO = "HELLO"
LSA = "LSA"
DATA = "DATA"


def build_hello(from_ip: str) -> dict:
    return {"type": HELLO, "from": from_ip}


def build_lsa(origin: str, seq: int, links: dict[str, int], from_ip: str) -> dict:
    return {"type": LSA, "origin": origin, "seq": seq, "links": links, "from": from_ip}


def loads(line: str) -> dict:
    """Parsea una linea recibida por el socket (delimitada por \\n)."""
    return json.loads(line)


def dumps(message: dict) -> str:
    """Serializa un mensaje agregando el delimitador de linea del protocolo."""
    return json.dumps(message) + "\n"
