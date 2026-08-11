"""Estado del grafo de enlaces y reglas de flooding de LSA."""

from __future__ import annotations

from dataclasses import dataclass, field
import threading


@dataclass
class LinkStateStore:
    """Estado compartido entre el hilo de routing y las conexiones entrantes."""

    self_id: str
    graph: dict[str, dict[str, int]] = field(default_factory=dict)
    seen: set[tuple[str, int]] = field(default_factory=set)
    seq: int = 0
    _lock: threading.Lock = field(default_factory=threading.Lock)

    def next_seq(self) -> int:
        with self._lock:
            self.seq += 1
            return self.seq

    def record(self, origin: str, seq: int, links: dict[str, int]) -> bool:
        """Registra un LSA si (origin, seq) es nuevo. Retorna False si ya se vio."""
        key = (origin, seq)
        with self._lock:
            if key in self.seen:
                return False
            self.seen.add(key)
            self.graph[origin] = dict(links)
            return True

    def snapshot(self) -> dict[str, dict[str, int]]:
        with self._lock:
            return {node: dict(links) for node, links in self.graph.items()}
