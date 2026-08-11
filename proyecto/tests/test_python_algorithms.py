from pathlib import Path
import random
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from router import codec  # noqa: E402
from router.algorithms import dijkstra, hamming, noise  # noqa: E402


class HammingTests(unittest.TestCase):
    """Vectores manuales, longitudes genericas y correccion de un bit por bloque."""

    def test_manual_hamming_7_4_vector(self):
        self.assertEqual(hamming.required_parity_bits(4), 3)
        self.assertEqual(hamming.encode("1011"), "0110011")

    def test_every_single_bit_error_is_corrected_within_one_block(self):
        message = "1011"  # un solo bloque de 4 bits
        encoded = hamming.encode(message)
        for index in range(len(encoded)):
            corrupted = list(encoded)
            corrupted[index] = "1" if corrupted[index] == "0" else "0"
            result = hamming.decode("".join(corrupted), len(message))
            self.assertTrue(result.valid, index)
            self.assertTrue(result.corrected, index)
            self.assertEqual(result.syndrome, index + 1)
            self.assertEqual(result.data_bits, message)

    def test_one_error_per_block_is_independently_corrected(self):
        # Un SEC generico sobre todo el frame solo tolera UN bit volteado en
        # todo el mensaje; por bloques, cada bloque de 7 tolera el suyo.
        message = "10110010"  # dos bloques de 4 bits
        encoded = hamming.encode(message)
        corrupted = list(encoded)
        corrupted[2] = "1" if corrupted[2] == "0" else "0"  # bloque 0
        corrupted[9] = "1" if corrupted[9] == "0" else "0"  # bloque 1
        result = hamming.decode("".join(corrupted), len(message))
        self.assertTrue(result.valid)
        self.assertTrue(result.corrected)
        self.assertEqual(result.data_bits, message)

    def test_generic_lengths_round_trip(self):
        source = random.Random(3067)
        for length in (1, 2, 4, 8, 31, 32, 1000):
            message = "".join(source.choice("01") for _ in range(length))
            encoded = hamming.encode(message)
            blocks = -(-length // hamming.BLOCK_DATA_BITS) if length else 0
            self.assertEqual(len(encoded), blocks * hamming.BLOCK_CODE_BITS)
            self.assertEqual(hamming.decode(encoded, length).data_bits, message)


class CodecAndNoiseTests(unittest.TestCase):
    """Contratos entre el codec ASCII<->bits y el simulador de ruido."""

    def test_codec_round_trip(self):
        bits = codec.codificar_mensaje("A router message")
        self.assertEqual(bits[:8], "01000001")
        self.assertEqual(codec.decodificar_mensaje(bits), "A router message")

    def test_hamming_over_full_frame_after_noise(self):
        # Simula el plano de datos: se codifica el frame completo, se le aplica
        # ruido y un solo bit volteado debe seguir siendo corregible.
        data = codec.codificar_mensaje('{"from":"A","to":"B","msg":"hi"}')
        frame = list(hamming.encode(data))
        frame[4] = "1" if frame[4] == "0" else "0"
        result = hamming.decode("".join(frame), len(data))
        self.assertTrue(result.valid)
        self.assertTrue(result.corrected)
        self.assertEqual(result.data_bits, data)

    def test_noise_rate_one_flips_all_bits(self):
        result, flips = noise.aplicar_ruido("001101", 1.0, random.Random(1))
        self.assertEqual(result, "110010")
        self.assertEqual(flips, 6)

    def test_fraction_probability(self):
        self.assertAlmostEqual(noise.parse_probability("1/100"), 0.01)


class DijkstraTests(unittest.TestCase):
    """Grafo de ejemplo tomado del ejemplo del profesor (nodo U)."""

    GRAPH = {
        "U": {"X": 1, "Y": 1, "C": 1},
        "X": {"U": 1, "Y": 1},
        "Y": {"U": 1, "X": 1, "Z": 1},
        "Z": {"Y": 1, "S": 1},
        "C": {"U": 1},
        "S": {"Z": 1},
    }

    def test_shortest_paths_from_u(self):
        result = dijkstra.shortest_paths(self.GRAPH, "U")
        self.assertEqual(result["C"].cost, 1)
        self.assertEqual(result["X"].cost, 1)
        self.assertEqual(result["Y"].cost, 1)
        self.assertEqual(result["Z"].cost, 2)
        self.assertEqual(result["S"].cost, 3)
        self.assertEqual(result["Z"].next_hop, "Y")
        self.assertEqual(result["S"].next_hop, "Y")

    def test_unreachable_node_is_excluded(self):
        graph = {"A": {"B": 1}, "B": {"A": 1}, "C": {}}
        result = dijkstra.shortest_paths(graph, "A")
        self.assertNotIn("C", result)


if __name__ == "__main__":
    unittest.main()
