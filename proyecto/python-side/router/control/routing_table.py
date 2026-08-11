"""Lectura y escritura de <nodo>_tabla_enrutamiento.csv (plano de control -> plano de datos)."""

from __future__ import annotations

import csv
from dataclasses import dataclass
from pathlib import Path

FIELDNAMES = ["destination", "next_hop_ip", "next_hop_port", "cost"]


@dataclass(frozen=True)
class Route:
    destination: str
    next_hop_ip: str
    next_hop_port: int
    cost: int


def write(path: str | Path, routes: dict[str, Route]) -> None:
    with Path(path).open("w", newline="", encoding="utf-8") as handle:
        writer = csv.writer(handle)
        writer.writerow(FIELDNAMES)
        for route in routes.values():
            writer.writerow([route.destination, route.next_hop_ip, route.next_hop_port, route.cost])


def read(path: str | Path) -> dict[str, Route]:
    routes: dict[str, Route] = {}
    with Path(path).open(newline="", encoding="utf-8") as handle:
        for row in csv.DictReader(handle):
            routes[row["destination"]] = Route(
                row["destination"], row["next_hop_ip"], int(row["next_hop_port"]), int(row["cost"])
            )
    return routes
