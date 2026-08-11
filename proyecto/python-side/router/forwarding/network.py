"""Plano de datos: decodifica DATA, consulta la tabla de ruteo y reenvia.

Implementa el pipeline de 9 pasos del enunciado: Hamming(7,4) por bloques
protege el frame {from, to, msg} completo en cada salto (el frame se parte
en bloques de 4 bits, cada uno codificado a 7). Un router intermedio corrige
y deserializa el frame entero (asi lo exige Hamming), pero solo LEE el
campo "to" para decidir hacia donde reenviar -- nunca actua sobre "msg".
Unicamente el destino final interpreta el contenido de "msg".
"""

from __future__ import annotations

import json
import logging

from router import codec
from router.algorithms import hamming
from router.control.routing_table import Route

logger = logging.getLogger("router.forwarding")


def decode_frame(message: dict) -> str | None:
    """Pasos 1-2: corrige errores sobre TODOS los bits del frame recibido."""
    result = hamming.decode(message["bits"], message["len"])
    if not result.valid:
        logger.warning("frame DATA invalido: %s", result.error)
        return None
    return result.data_bits


def decode_payload(message: dict) -> dict | None:
    """Frame ya corregido -> dict {from, to, msg}. Usado por routers y hosts."""
    data_bits = decode_frame(message)
    if data_bits is None:
        return None
    return json.loads(codec.decodificar_mensaje(data_bits))


def encode_envelope(payload: dict) -> dict:
    """Serializa {from, to, msg} y aplica Hamming(7,4) sobre el frame completo."""
    data_bits = codec.codificar_mensaje(json.dumps(payload))
    return {"type": "DATA", "len": len(data_bits), "bits": hamming.encode(data_bits)}


def forward(message: dict, routes: dict[str, Route], send_message) -> None:
    """Pasos 3-9: deserializa solo 'to', consulta tabla, re-codifica y reenvia."""
    data_bits = decode_frame(message)
    if data_bits is None:
        return
    destination = json.loads(codec.decodificar_mensaje(data_bits))["to"]

    route = routes.get(destination)
    if route is None:
        logger.warning("sin ruta hacia %s; descartando DATA", destination)
        return

    envelope = {"type": "DATA", "len": len(data_bits), "bits": hamming.encode(data_bits)}
    send_message(route.next_hop_ip, route.next_hop_port, envelope)
