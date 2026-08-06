from pathlib import Path
import random
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from atm.algorithms import crc32, hamming  # noqa: E402
from atm.layers import link, noise, presentation  # noqa: E402


class HammingTests(unittest.TestCase):
    """Vectores manuales, longitudes genericas y correccion de un bit."""

    def test_manual_hamming_7_4_vector(self):
        self.assertEqual(hamming.required_parity_bits(4), 3)
        self.assertEqual(hamming.encode("1011"), "0110011")

    def test_every_single_bit_error_is_corrected(self):
        message = "10110010"
        encoded = hamming.encode(message)
        for index in range(len(encoded)):
            corrupted = list(encoded)
            corrupted[index] = "1" if corrupted[index] == "0" else "0"
            result = hamming.decode("".join(corrupted), len(message))
            self.assertTrue(result.valid, index)
            self.assertTrue(result.corrected, index)
            self.assertEqual(result.syndrome, index + 1)
            self.assertEqual(result.data_bits, message)

    def test_generic_lengths_round_trip(self):
        source = random.Random(3067)
        for length in (1, 2, 4, 8, 31, 32, 1000):
            message = "".join(source.choice("01") for _ in range(length))
            encoded = hamming.encode(message)
            self.assertEqual(len(encoded), length + hamming.required_parity_bits(length))
            self.assertEqual(hamming.decode(encoded, length).data_bits, message)


class CRC32Tests(unittest.TestCase):
    """Vectores IEEE, padding corto y deteccion en datos/redundancia."""

    def test_ieee_known_vector(self):
        bits = presentation.codificar_mensaje("123456789")
        self.assertEqual(crc32.calculate(bits), 0xCBF43926)
        self.assertEqual(crc32.checksum_bits(bits), "11001011111101000011100100100110")

    def test_short_input_is_padded_to_32_bits(self):
        # Un bit 1 se representa como 0x80 00 00 00 despues del padding.
        self.assertEqual(crc32.calculate("1"), 0xCC1D6927)

    def test_detects_data_and_checksum_errors(self):
        data = presentation.codificar_mensaje("ATM")
        frame = crc32.encode(data)
        for index in (0, len(data) - 1, len(frame) - 1):
            corrupted = list(frame)
            corrupted[index] = "1" if corrupted[index] == "0" else "0"
            ok, _, _ = crc32.verify("".join(corrupted), len(data))
            self.assertFalse(ok)


class LayerTests(unittest.TestCase):
    """Contratos entre Presentacion, Enlace y Ruido."""

    def test_presentation_round_trip(self):
        bits = presentation.codificar_mensaje("A bank message")
        self.assertEqual(bits[:8], "01000001")
        self.assertEqual(presentation.decodificar_mensaje(bits), "A bank message")

    def test_link_corrects_hamming(self):
        data = presentation.codificar_mensaje("A")
        frame = list(link.calcular_integridad(data, "hamming"))
        frame[4] = "1" if frame[4] == "0" else "0"
        result = link.verificar_integridad("".join(frame), "hamming", len(data))
        self.assertTrue(result.ok)
        self.assertTrue(result.corrected)
        self.assertEqual(result.message_bits, data)

    def test_noise_rate_one_flips_all_bits(self):
        result, flips = noise.aplicar_ruido("001101", 1.0, random.Random(1))
        self.assertEqual(result, "110010")
        self.assertEqual(flips, 6)

    def test_fraction_probability(self):
        self.assertAlmostEqual(noise.parse_probability("1/100"), 0.01)


if __name__ == "__main__":
    unittest.main()
