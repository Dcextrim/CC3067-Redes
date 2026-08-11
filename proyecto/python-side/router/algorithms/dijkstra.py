"""Rutas mas cortas sobre el grafo construido a partir de los LSA."""

from __future__ import annotations

from dataclasses import dataclass
import heapq


@dataclass(frozen=True)
class PathResult:
    """Costo total y primer salto desde el origen hacia un destino."""

    cost: int
    next_hop: str


def shortest_paths(graph: dict[str, dict[str, int]], source: str) -> dict[str, PathResult]:
    """Calcula costo y primer salto de source hacia cada nodo alcanzable."""
    costs: dict[str, float] = {source: 0}
    first_hop: dict[str, str] = {}
    visited: set[str] = set()
    queue: list[tuple[int, str]] = [(0, source)]

    while queue:
        cost, node = heapq.heappop(queue)
        if node in visited:
            continue
        visited.add(node)
        for neighbor, weight in graph.get(node, {}).items():
            new_cost = cost + weight
            if new_cost < costs.get(neighbor, float("inf")):
                costs[neighbor] = new_cost
                # El primer salto se hereda del nodo previo, salvo el caso base.
                first_hop[neighbor] = neighbor if node == source else first_hop[node]
                heapq.heappush(queue, (new_cost, neighbor))

    return {
        node: PathResult(int(cost), first_hop[node])
        for node, cost in costs.items()
        if node != source
    }
