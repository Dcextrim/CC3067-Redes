"""Carga la configuracion de un nodo (ver docs/protocol.md, seccion Configuracion)."""

from __future__ import annotations

from dataclasses import dataclass
import json
from pathlib import Path


@dataclass(frozen=True)
class Neighbor:
    ip: str
    port: int
    cost: int

    @property
    def id(self) -> str:
        return node_id(self.ip, self.port)


@dataclass(frozen=True)
class AttachedHost:
    """Cliente o servidor no-router que usa a este nodo como gateway."""

    role: str  # "client" o "server"
    ip: str
    port: int
    cost: int = 1

    @property
    def id(self) -> str:
        return node_id(self.ip, self.port)


def node_id(ip: str, port: int) -> str:
    """Identificador unico de un nodo: IP sola no alcanza en pruebas locales
    (127.0.0.1 se comparte entre todos los nodos), asi que se usa ip:puerto."""
    return f"{ip}:{port}"


@dataclass(frozen=True)
class NodeConfig:
    name: str
    ip: str
    port: int
    neighbors: tuple[Neighbor, ...]
    attached_host: AttachedHost | None = None

    @property
    def id(self) -> str:
        return node_id(self.ip, self.port)


def load(path: str | Path) -> NodeConfig:
    """Lee y valida el config.json de un nodo."""
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    neighbors = tuple(
        Neighbor(n["ip"], int(n["port"]), int(n["cost"])) for n in data.get("neighbors", [])
    )
    host_data = data.get("attached_host")
    attached_host = (
        AttachedHost(host_data["role"], host_data["ip"], int(host_data["port"]), int(host_data.get("cost", 1)))
        if host_data
        else None
    )
    return NodeConfig(
        name=data["name"],
        ip=data["ip"],
        port=int(data["port"]),
        neighbors=neighbors,
        attached_host=attached_host,
    )
