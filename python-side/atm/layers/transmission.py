"""Capa de Transmision: framing binario y E/S TCP."""

from dataclasses import dataclass
import socket
import struct
import threading

MAGIC = b"CC67"
VERSION = 1
# magic, version, algoritmo, flags, longitud original y longitud de trama.
HEADER = struct.Struct("!4sBBHII")
MAX_FRAME_BITS = 8 * 1024 * 1024
ALGORITHM_IDS = {"hamming": 1, "crc32": 2}
ID_ALGORITHMS = {value: key for key, value in ALGORITHM_IDS.items()}


@dataclass(frozen=True)
class ReceivedFrame:
    """Trama reconstruida a partir del encabezado y cuerpo del stream TCP."""

    algorithm: str
    message_bit_length: int
    frame_bits: str


def pack_bits(bits: str) -> bytes:
    """Empaqueta bits MSB primero y completa el ultimo octeto con ceros."""
    if any(bit not in "01" for bit in bits):
        raise ValueError("la cadena solo puede contener bits 0 y 1")
    padded = bits + "0" * ((8 - len(bits) % 8) % 8)
    return bytes(int(padded[i : i + 8], 2) for i in range(0, len(padded), 8))


def unpack_bits(data: bytes, bit_length: int) -> str:
    """Recupera la longitud logica y descarta el padding del ultimo octeto."""
    bits = "".join(f"{byte:08b}" for byte in data)
    if bit_length > len(bits):
        raise ValueError("el cuerpo no contiene suficientes bits")
    return bits[:bit_length]


def _recv_exact(sock: socket.socket, length: int) -> bytes | None:
    """Lee exactamente length bytes porque una llamada recv puede ser parcial."""
    chunks = bytearray()
    while len(chunks) < length:
        chunk = sock.recv(length - len(chunks))
        if not chunk:
            if not chunks:
                return None
            raise ConnectionError("la conexion termino en medio de una trama")
        chunks.extend(chunk)
    return bytes(chunks)


class TransmissionLayer:
    """Encapsula framing y E/S TCP; no interpreta la integridad del cuerpo."""

    def __init__(self, sock: socket.socket):
        self._sock = sock
        self._send_lock = threading.Lock()

    def enviar_informacion(self, algorithm: str, message_bit_length: int, frame_bits: str) -> None:
        """Envia de forma contigua un encabezado CC67 y el cuerpo empaquetado."""
        if algorithm not in ALGORITHM_IDS:
            raise ValueError("algoritmo no soportado")
        if not 0 <= message_bit_length <= 0xFFFFFFFF:
            raise ValueError("longitud de mensaje fuera de rango")
        if len(frame_bits) > MAX_FRAME_BITS:
            raise ValueError("trama demasiado grande")
        header = HEADER.pack(
            MAGIC, VERSION, ALGORITHM_IDS[algorithm], 0,
            message_bit_length, len(frame_bits),
        )
        with self._send_lock:
            self._sock.sendall(header + pack_bits(frame_bits))

    def recibir_informacion(self) -> ReceivedFrame | None:
        """Delimita una trama del stream o retorna None ante un cierre limpio."""
        raw_header = _recv_exact(self._sock, HEADER.size)
        if raw_header is None:
            return None
        magic, version, algorithm_id, flags, message_length, frame_length = HEADER.unpack(raw_header)
        if magic != MAGIC or version != VERSION or flags != 0:
            raise ValueError("encabezado de transmision invalido")
        if algorithm_id not in ID_ALGORITHMS:
            raise ValueError("identificador de algoritmo desconocido")
        if frame_length > MAX_FRAME_BITS:
            raise ValueError("trama recibida demasiado grande")
        body_length = (frame_length + 7) // 8
        body = _recv_exact(self._sock, body_length)
        if body is None:
            raise ConnectionError("falta el cuerpo de la trama")
        return ReceivedFrame(ID_ALGORITHMS[algorithm_id], message_length, unpack_bits(body, frame_length))
