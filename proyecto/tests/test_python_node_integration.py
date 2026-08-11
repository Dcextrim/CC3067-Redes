"""Prueba de integracion real: nodos router de verdad (sockets TCP en
127.0.0.1, sin mocks) intercambiando HELLO/LSA para converger a las rutas
mas cortas correctas. Usa temporizadores acelerados (los de produccion,
10s/30s, harian la prueba demasiado lenta)."""

from pathlib import Path
import sys
import time
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from router.config import Neighbor, NodeConfig  # noqa: E402
from router.node import Node  # noqa: E402


def _accelerated(config: NodeConfig, csv_path: str) -> Node:
    return Node(
        config,
        csv_path,
        hello_interval_s=0.1,
        lsa_delay_s=0.2,
        convergence_wait_s=0.8,
        neighbor_timeout_s=1.0,
        neighbor_check_interval_s=0.15,
        route_recompute_interval_s=0.3,
        lsa_rebuild_debounce_s=0.1,
    )


class MultiRouterConvergenceTests(unittest.TestCase):
    """Topologia de linea A-B-C (sin enlace directo A-C)."""

    def setUp(self):
        self.config_a = NodeConfig(
            name="A", ip="127.0.0.1", port=15920,
            neighbors=(Neighbor("127.0.0.1", 15921, 2),),
        )
        self.config_b = NodeConfig(
            name="B", ip="127.0.0.1", port=15921,
            neighbors=(Neighbor("127.0.0.1", 15920, 2), Neighbor("127.0.0.1", 15922, 3)),
        )
        self.config_c = NodeConfig(
            name="C", ip="127.0.0.1", port=15922,
            neighbors=(Neighbor("127.0.0.1", 15921, 3),),
        )
        tmp = self._tmp_dir()
        self.node_a = _accelerated(self.config_a, f"{tmp}/A_tabla_enrutamiento.csv")
        self.node_b = _accelerated(self.config_b, f"{tmp}/B_tabla_enrutamiento.csv")
        self.node_c = _accelerated(self.config_c, f"{tmp}/C_tabla_enrutamiento.csv")
        self.node_a.start()
        self.node_b.start()
        self.node_c.start()
        self.addCleanup(self._stop_all)

    def _stop_all(self):
        self.node_a._stop.set()
        self.node_b._stop.set()
        self.node_c._stop.set()

    def _tmp_dir(self) -> str:
        import tempfile
        return tempfile.mkdtemp()

    def _wait_for_route(self, node: Node, destination: str, timeout: float = 5.0):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            with node._state_lock:
                route = node.routes.get(destination)
            if route is not None:
                return route
            time.sleep(0.05)
        self.fail(f"timeout esperando ruta hacia {destination} en {node.config.name}")

    def test_converges_to_correct_shortest_paths(self):
        route_a_to_c = self._wait_for_route(self.node_a, self.config_c.id)
        self.assertEqual(route_a_to_c.cost, 5)
        self.assertEqual(route_a_to_c.next_hop_port, self.config_b.port)

        route_c_to_a = self._wait_for_route(self.node_c, self.config_a.id)
        self.assertEqual(route_c_to_a.cost, 5)
        self.assertEqual(route_c_to_a.next_hop_port, self.config_b.port)

        route_a_to_b = self._wait_for_route(self.node_a, self.config_b.id)
        self.assertEqual(route_a_to_b.cost, 2)

        route_b_to_c = self._wait_for_route(self.node_b, self.config_c.id)
        self.assertEqual(route_b_to_c.cost, 3)


if __name__ == "__main__":
    unittest.main()
