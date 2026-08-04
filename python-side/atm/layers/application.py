"""Servicios de la capa de Aplicacion."""

from dataclasses import dataclass
from typing import Callable

from atm.layers.link import normalize_algorithm


@dataclass(frozen=True)
class OutgoingRequest:
    """Datos capturados por Aplicacion antes de atravesar las otras capas."""

    message: str
    algorithm: str
    probability_text: str


def solicitar_mensaje(input_fn: Callable[[str], str] = input) -> OutgoingRequest:
    """Solicita texto, algoritmo y ruido sin codificar ni transmitir datos."""
    message = input_fn("Mensaje ASCII (o /salir): ")
    if message == "/salir":
        raise EOFError
    algorithm = normalize_algorithm(input_fn("Algoritmo [1=hamming, 2=crc32]: "))
    probability = input_fn("Probabilidad de error [decimal o fraccion, ej. 1/100]: ")
    return OutgoingRequest(message, algorithm, probability)


def mostrar_mensaje(message: str | None = None, error: str | None = None, corrected: bool = False) -> None:
    """Presenta al usuario una entrega valida o el error reportado por las capas."""
    if error:
        print(f"\n[APLICACION] ERROR: {error}")
        return
    suffix = " (Hamming corrigio un bit)" if corrected else ""
    print(f"\n[APLICACION] Mensaje recibido{suffix}: {message}")
