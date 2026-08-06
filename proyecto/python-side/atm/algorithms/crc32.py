"""CRC-32/ISO-HDLC sobre una cadena de bits.

Se usa la forma reflejada del polinomio IEEE 802.3, 0xEDB88320,
init=0xFFFFFFFF y xorout=0xFFFFFFFF. Los bits se empaquetan MSB primero
en cada octeto antes de ejecutar el algoritmo reflejado.
"""

POLYNOMIAL_REFLECTED = 0xEDB88320


def _validate_bits(bits: str) -> None:
    """Rechaza entradas distintas de una cadena binaria."""
    if any(bit not in "01" for bit in bits):
        raise ValueError("la cadena solo puede contener bits 0 y 1")


def _padded_bytes(bits: str) -> bytes:
    """Aplica el padding acordado y empaqueta MSB primero en octetos."""
    _validate_bits(bits)
    minimum_padding = max(0, 32 - len(bits)) if len(bits) <= 32 else 0
    padded = bits + ("0" * minimum_padding)
    padded += "0" * ((8 - len(padded) % 8) % 8)
    return bytes(int(padded[i : i + 8], 2) for i in range(0, len(padded), 8))


def calculate(bits: str) -> int:
    """Calcula el CRC como entero de 32 bits."""
    crc = 0xFFFFFFFF
    for byte in _padded_bytes(bits):
        crc ^= byte
        for _ in range(8):
            # El desplazamiento a la derecha corresponde al polinomio reflejado.
            crc = (crc >> 1) ^ POLYNOMIAL_REFLECTED if crc & 1 else crc >> 1
    return crc ^ 0xFFFFFFFF


def checksum_bits(bits: str) -> str:
    """Serializa el CRC como 32 bits, con el bit mas significativo primero."""
    return f"{calculate(bits):032b}"


def encode(data_bits: str) -> str:
    """Concatena datos y checksum para formar la trama de Enlace."""
    _validate_bits(data_bits)
    return data_bits + checksum_bits(data_bits)


def verify(frame_bits: str, message_length: int) -> tuple[bool, str, str | None]:
    """Separa el checksum recibido y lo compara contra el valor recalculado."""
    _validate_bits(frame_bits)
    if message_length < 0 or len(frame_bits) != message_length + 32:
        return False, "", "longitud CRC-32 invalida"
    data = frame_bits[:message_length]
    received = frame_bits[message_length:]
    expected = checksum_bits(data)
    if received != expected:
        return False, "", f"CRC-32 no coincide: recibido {received}, esperado {expected}"
    return True, data, None
