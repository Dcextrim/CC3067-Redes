from pathlib import Path
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from router.control import messages  # noqa: E402
from router.control.flooding import LinkStateStore  # noqa: E402


class MessageBuilderTests(unittest.TestCase):
    def test_hello_shape(self):
        hello = messages.build_hello("100.0.0.1:5000")
        self.assertEqual(hello, {"type": "HELLO", "from": "100.0.0.1:5000"})

    def test_lsa_shape_matches_protocol_example(self):
        lsa = messages.build_lsa("A", 3, {"B": 2, "C": 5}, "A")
        self.assertEqual(lsa["type"], "LSA")
        self.assertEqual(lsa["origin"], "A")
        self.assertEqual(lsa["seq"], 3)
        self.assertEqual(lsa["links"], {"B": 2, "C": 5})

    def test_round_trip_serialization_has_newline_delimiter(self):
        line = messages.dumps(messages.build_hello("A"))
        self.assertTrue(line.endswith("\n"))
        self.assertEqual(messages.loads(line.strip()), {"type": "HELLO", "from": "A"})


class FloodingRulesTests(unittest.TestCase):
    """Reglas de la seccion B de docs/protocol.md."""

    def test_new_origin_seq_is_recorded(self):
        store = LinkStateStore(self_id="A")
        self.assertTrue(store.record("B", 1, {"A": 1, "C": 2}))
        self.assertEqual(store.snapshot(), {"B": {"A": 1, "C": 2}})

    def test_duplicate_origin_seq_is_ignored(self):
        store = LinkStateStore(self_id="A")
        store.record("B", 1, {"A": 1})
        self.assertFalse(store.record("B", 1, {"A": 1}))

    def test_same_origin_new_seq_updates_graph(self):
        store = LinkStateStore(self_id="A")
        store.record("B", 1, {"A": 1})
        self.assertTrue(store.record("B", 2, {"A": 1, "C": 4}))
        self.assertEqual(store.snapshot()["B"], {"A": 1, "C": 4})

    def test_seq_increments_only_for_own_lsa(self):
        store = LinkStateStore(self_id="A")
        self.assertEqual(store.next_seq(), 1)
        self.assertEqual(store.next_seq(), 2)


if __name__ == "__main__":
    unittest.main()
