"""Codigo de Hamming(7,4) por bloques: paridad par, correccion de un bit por
bloque de 7 (4 bits de datos + 3 de paridad). El ultimo bloque se rellena
con ceros hasta completar 4 bits; el llamador es responsable de recordar la
longitud original (`len` en el envoltorio DATA) para descartar el relleno
al decodificar."""

from dataclasses import dataclass

BLOCK_DATA_BITS = 4
BLOCK_PARITY_BITS = 3
BLOCK_CODE_BITS = BLOCK_DATA_BITS + BLOCK_PARITY_BITS
_PARITY_POSITIONS = (1, 2, 4)


@dataclass(frozen=True)
class DecodeResult:
    """Resultado inmutable de verificar y decodificar un frame Hamming(7,4)."""

    data_bits: str
    corrected: bool
    syndrome: int
    valid: bool
    error: str | None = None


def _validate_bits(bits: str) -> None:
    """Valida la representacion textual usada por todos los algoritmos."""
    if any(bit not in "01" for bit in bits):
        raise ValueError("la cadena solo puede contener bits 0 y 1")


def _blocks_needed(message_bits: int) -> int:
    return (message_bits + BLOCK_DATA_BITS - 1) // BLOCK_DATA_BITS


def required_parity_bits(message_bits: int) -> int:
    """Bits de paridad totales para proteger message_bits en bloques de 4."""
    if message_bits < 0:
        raise ValueError("message_bits no puede ser negativo")
    return _blocks_needed(message_bits) * BLOCK_PARITY_BITS


def _encode_block(data_bits: str) -> str:
    """Inserta paridad par en las posiciones 1, 2, 4 de un bloque de 4 bits."""
    codeword = [0] * (BLOCK_CODE_BITS + 1)  # indice cero no usado
    data_index = 0
    for position in range(1, len(codeword)):
        if position & (position - 1):
            codeword[position] = int(data_bits[data_index])
            data_index += 1
    for parity_position in _PARITY_POSITIONS:
        parity = 0
        for position in range(1, len(codeword)):
            if position & parity_position:
                parity ^= codeword[position]
        codeword[parity_position] = parity
    return "".join(str(bit) for bit in codeword[1:])


def encode(data_bits: str) -> str:
    """Aplica Hamming(7,4) a data_bits en bloques de 4, con relleno de ceros."""
    _validate_bits(data_bits)
    if not data_bits:
        return ""
    padded = data_bits.ljust(_blocks_needed(len(data_bits)) * BLOCK_DATA_BITS, "0")
    blocks = (
        padded[start : start + BLOCK_DATA_BITS]
        for start in range(0, len(padded), BLOCK_DATA_BITS)
    )
    return "".join(_encode_block(block) for block in blocks)


def _block_syndrome(codeword: list[int]) -> int:
    syndrome = 0
    for parity_position in _PARITY_POSITIONS:
        parity = 0
        for position in range(1, len(codeword)):
            if position & parity_position:
                parity ^= codeword[position]
        if parity:
            syndrome += parity_position
    return syndrome


def _decode_block(encoded_block: str) -> tuple[str, bool, int, bool, str | None]:
    """Verifica y corrige un unico bloque de 7 bits."""
    codeword = [0] + [int(bit) for bit in encoded_block]
    syndrome = _block_syndrome(codeword)
    corrected = False
    if syndrome:
        # En SEC el sindrome es la posicion, numerada desde uno, que se voltea.
        if syndrome >= len(codeword):
            return "", False, syndrome, False, "el sindrome apunta fuera del bloque; error no corregible"
        codeword[syndrome] ^= 1
        corrected = True
        if _block_syndrome(codeword):
            return "", False, syndrome, False, "el bloque conserva paridad invalida despues de corregir"
    data = "".join(
        str(codeword[position])
        for position in range(1, len(codeword))
        if position & (position - 1)
    )
    return data, corrected, syndrome, True, None


def decode(encoded_bits: str, message_length: int) -> DecodeResult:
    """Verifica/corrige cada bloque de 7 bits y extrae message_length bits de datos."""
    _validate_bits(encoded_bits)
    if message_length < 0:
        raise ValueError("message_length no puede ser negativo")

    blocks_needed = _blocks_needed(message_length)
    expected_length = blocks_needed * BLOCK_CODE_BITS
    if len(encoded_bits) != expected_length:
        return DecodeResult(
            "", False, 0, False,
            f"longitud Hamming invalida: se esperaban {expected_length} bits",
        )
    if not encoded_bits:
        return DecodeResult("", False, 0, True)

    data_chunks = []
    corrected_any = False
    last_syndrome = 0
    for block_index in range(blocks_needed):
        start = block_index * BLOCK_CODE_BITS
        block = encoded_bits[start : start + BLOCK_CODE_BITS]
        data, corrected, syndrome, valid, error = _decode_block(block)
        if not valid:
            return DecodeResult("", False, syndrome, False, f"bloque {block_index}: {error}")
        data_chunks.append(data)
        if corrected:
            corrected_any = True
            last_syndrome = syndrome

    data = "".join(data_chunks)[:message_length]
    return DecodeResult(data, corrected_any, last_syndrome, True)
