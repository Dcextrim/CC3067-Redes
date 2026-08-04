"""Codigo de Hamming generico con paridad par y correccion de un bit."""

from dataclasses import dataclass


@dataclass(frozen=True)
class DecodeResult:
    data_bits: str
    corrected: bool
    syndrome: int
    valid: bool
    error: str | None = None


def _validate_bits(bits: str) -> None:
    if any(bit not in "01" for bit in bits):
        raise ValueError("la cadena solo puede contener bits 0 y 1")


def required_parity_bits(message_bits: int) -> int:
    """Retorna el menor r tal que m + r + 1 <= 2**r."""
    if message_bits < 0:
        raise ValueError("message_bits no puede ser negativo")
    r = 0
    while message_bits + r + 1 > (1 << r):
        r += 1
    return r


def encode(data_bits: str) -> str:
    """Inserta bits de paridad en posiciones 1, 2, 4, 8, ..."""
    _validate_bits(data_bits)
    if not data_bits:
        return ""

    parity_count = required_parity_bits(len(data_bits))
    codeword = [0] * (len(data_bits) + parity_count + 1)  # indice cero no usado
    data_index = 0

    for position in range(1, len(codeword)):
        if position & (position - 1):
            codeword[position] = int(data_bits[data_index])
            data_index += 1

    for parity_position in (1 << i for i in range(parity_count)):
        parity = 0
        for position in range(1, len(codeword)):
            if position & parity_position:
                parity ^= codeword[position]
        codeword[parity_position] = parity

    return "".join(str(bit) for bit in codeword[1:])


def _syndrome(codeword: list[int], parity_count: int) -> int:
    syndrome = 0
    for parity_position in (1 << i for i in range(parity_count)):
        parity = 0
        for position in range(1, len(codeword)):
            if position & parity_position:
                parity ^= codeword[position]
        if parity:
            syndrome += parity_position
    return syndrome


def decode(encoded_bits: str, message_length: int) -> DecodeResult:
    """Verifica, corrige un error y extrae exactamente message_length bits."""
    _validate_bits(encoded_bits)
    if message_length < 0:
        raise ValueError("message_length no puede ser negativo")

    parity_count = required_parity_bits(message_length)
    expected_length = message_length + parity_count
    if len(encoded_bits) != expected_length:
        return DecodeResult(
            "", False, 0, False,
            f"longitud Hamming invalida: se esperaban {expected_length} bits",
        )
    if not encoded_bits:
        return DecodeResult("", False, 0, True)

    codeword = [0] + [int(bit) for bit in encoded_bits]
    syndrome = _syndrome(codeword, parity_count)
    corrected = False

    if syndrome:
        if syndrome >= len(codeword):
            return DecodeResult(
                "", False, syndrome, False,
                "el sindrome apunta fuera de la trama; error no corregible",
            )
        codeword[syndrome] ^= 1
        corrected = True
        if _syndrome(codeword, parity_count):
            return DecodeResult(
                "", False, syndrome, False,
                "la trama conserva paridad invalida despues de corregir",
            )

    data = "".join(
        str(codeword[position])
        for position in range(1, len(codeword))
        if position & (position - 1)
    )
    return DecodeResult(data[:message_length], corrected, syndrome, True)
