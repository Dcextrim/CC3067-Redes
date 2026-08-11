from pathlib import Path
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from router.control.routing_table import Route  # noqa: E402
from router.forwarding import network  # noqa: E402


class DataPipelineTests(unittest.TestCase):
    """Pipeline de 9 pasos del enunciado (docs/protocol.md, seccion 3.C)."""

    def test_encode_then_decode_round_trip(self):
        payload = {"from": "A", "to": "B", "msg": "hola"}
        envelope = network.encode_envelope(payload)
        self.assertEqual(envelope["type"], "DATA")
        self.assertEqual(network.decode_payload(envelope), payload)

    def test_forward_only_reads_to_and_reencodes_full_frame(self):
        envelope = network.encode_envelope({"from": "A", "to": "C", "msg": "secreto"})
        routes = {"C": Route("C", "10.0.0.2", 5001, 2)}
        sent: list[tuple[str, int, dict]] = []

        network.forward(envelope, routes, lambda ip, port, message: sent.append((ip, port, message)))

        self.assertEqual(len(sent), 1)
        ip, port, forwarded = sent[0]
        self.assertEqual((ip, port), ("10.0.0.2", 5001))
        # El router reenvia el MISMO payload logico (msg intacto), solo cambia
        # la codificacion Hamming del frame (fresca en cada salto).
        self.assertEqual(network.decode_payload(forwarded), {"from": "A", "to": "C", "msg": "secreto"})

    def test_forward_drops_when_no_route(self):
        envelope = network.encode_envelope({"from": "A", "to": "unknown", "msg": "x"})
        sent = []
        network.forward(envelope, routes={}, send_message=lambda ip, port, m: sent.append(m))
        self.assertEqual(sent, [])

    def test_single_bit_error_is_still_corrected_before_routing(self):
        envelope = network.encode_envelope({"from": "A", "to": "C", "msg": "y"})
        corrupted = dict(envelope)
        bits = list(corrupted["bits"])
        bits[3] = "1" if bits[3] == "0" else "0"
        corrupted["bits"] = "".join(bits)

        routes = {"C": Route("C", "10.0.0.2", 5001, 1)}
        sent = []
        network.forward(corrupted, routes, lambda ip, port, message: sent.append(message))

        self.assertEqual(len(sent), 1)
        self.assertEqual(network.decode_payload(sent[0])["msg"], "y")


if __name__ == "__main__":
    unittest.main()
