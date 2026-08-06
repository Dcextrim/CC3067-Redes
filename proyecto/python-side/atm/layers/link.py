"""Capa de Enlace: calculo, verificacion y correccion de integridad."""

from dataclasses import dataclass

from atm.algorithms import crc32, hamming

HAMMING = "hamming"
CRC32 = "crc32"
ALGORITHMS = (HAMMING, CRC32)


@dataclass(frozen=True)
class IntegrityResult:
    """Respuesta uniforme de Enlace para algoritmos de deteccion o correccion."""

    ok: bool
    message_bits: str = ""
    corrected: bool = False
    error: str | None = None
    syndrome: int = 0


def normalize_algorithm(algorithm: str) -> str:
    """Convierte las opciones de consola al identificador canonico del protocolo."""
    value = algorithm.strip().lower().replace("-", "")
    aliases = {"1": HAMMING, "hamming": HAMMING, "2": CRC32, "crc": CRC32, "crc32": CRC32}
    if value not in aliases:
        raise ValueError("algoritmo invalido; use hamming o crc32")
    return aliases[value]


def calcular_integridad(message_bits: str, algorithm: str) -> str:
    """Agrega la redundancia definida por el algoritmo seleccionado."""
    selected = normalize_algorithm(algorithm)
    if selected == HAMMING:
        return hamming.encode(message_bits)
    return crc32.encode(message_bits)


def verificar_integridad(frame_bits: str, algorithm: str, message_length: int) -> IntegrityResult:
    """Verifica una trama; delega en corregir_mensaje cuando el algoritmo puede reparar errores."""
    selected = normalize_algorithm(algorithm)
    if selected == HAMMING:
        return corregir_mensaje(frame_bits, message_length)

    ok, data, error = crc32.verify(frame_bits, message_length)
    return IntegrityResult(ok, data, False, error)


def corregir_mensaje(frame_bits: str, message_length: int) -> IntegrityResult:
    """Servicio de Enlace: verifica Hamming y corrige el bit senalado por el sindrome."""
    result = hamming.decode(frame_bits, message_length)
    return IntegrityResult(result.valid, result.data_bits, result.corrected, result.error, result.syndrome)


def redundancy_bits(message_length: int, algorithm: str) -> int:
    """Calcula el overhead teorico sin construir la trama completa."""
    selected = normalize_algorithm(algorithm)
    return hamming.required_parity_bits(message_length) if selected == HAMMING else 32
