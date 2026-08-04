from pathlib import Path
import socket
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from atm.layers.transmission import TransmissionLayer  # noqa: E402


class TransmissionTests(unittest.TestCase):
    def test_binary_frame_survives_tcp_stream(self):
        left, right = socket.socketpair()
        try:
            sender = TransmissionLayer(left)
            receiver = TransmissionLayer(right)
            sender.enviar_informacion("hamming", 8, "10100101101")
            frame = receiver.recibir_informacion()
            self.assertIsNotNone(frame)
            self.assertEqual(frame.algorithm, "hamming")
            self.assertEqual(frame.message_bit_length, 8)
            self.assertEqual(frame.frame_bits, "10100101101")
        finally:
            left.close()
            right.close()


if __name__ == "__main__":
    unittest.main()
