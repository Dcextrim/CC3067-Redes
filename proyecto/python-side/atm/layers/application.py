"""Servicios de la capa de Aplicacion: login, retiro y logout del cajero."""

from dataclasses import dataclass
from typing import Callable

from atm.layers.link import normalize_algorithm


@dataclass(frozen=True)
class Credenciales:
    """Tarjeta y PIN capturados antes de intentar iniciar sesion."""

    card: str
    pin: str


@dataclass(frozen=True)
class ParametrosEnvio:
    """Algoritmo y ruido pedidos justo antes de cada envio individual."""

    algorithm: str
    probability_text: str


@dataclass(frozen=True)
class Respuesta:
    """Respuesta del servidor ya separada en accion y campos."""

    action: str
    message: str = ""
    amount: float | None = None
    balance: float | None = None


class TransmisionCorrupta(Exception):
    """La respuesta llego con la integridad rota; el resultado es ambiguo."""


def solicitar_credenciales(input_fn: Callable[[str], str] = input) -> Credenciales:
    """Pide numero de tarjeta y PIN para un intento de inicio de sesion."""
    card = input_fn("Numero de tarjeta: ").strip()
    pin = input_fn("PIN: ").strip()
    return Credenciales(card, pin)


def solicitar_monto(input_fn: Callable[[str], str] = input) -> float:
    """Pide el monto a retirar; la validacion de negocio queda del lado del banco."""
    return float(input_fn("Monto a retirar: ").strip())


def solicitar_opcion_menu(input_fn: Callable[[str], str] = input) -> str:
    """Muestra el menu del cajero y devuelve la opcion elegida."""
    print("\n--- MENU ---")
    print("1) Retirar dinero")
    print("2) Salir")
    return input_fn("Elija una opcion: ").strip()


def solicitar_parametros_envio(input_fn: Callable[[str], str] = input) -> ParametrosEnvio:
    """Pide el algoritmo y la tasa de ruido para el envio que esta por salir."""
    algorithm = normalize_algorithm(input_fn("Algoritmo [1=hamming, 2=crc32]: "))
    probability = input_fn("Probabilidad de error [decimal o fraccion, ej. 1/100]: ")
    return ParametrosEnvio(algorithm, probability)


def comando_login(card: str, pin: str) -> str:
    """Construye el texto de peticion para un intento de inicio de sesion."""
    return f"LOGIN|{card}|{pin}"


def comando_retiro(amount: float) -> str:
    """Construye el texto de peticion para un retiro."""
    return f"WITHDRAW|{amount:.2f}"


def comando_logout() -> str:
    """Construye el texto de peticion para cerrar sesion."""
    return "LOGOUT"


def parse_respuesta(text: str) -> Respuesta:
    """Interpreta el texto plano recibido del banco como una respuesta tipada."""
    parts = text.split("|")
    action = parts[0]
    if action in ("LOGIN_OK", "LOGIN_DENIED", "LOGOUT_OK", "ERROR", "WITHDRAW_ERROR"):
        if len(parts) != 2:
            raise ValueError(f"respuesta {action} mal formada: {text!r}")
        return Respuesta(action, message=parts[1])
    if action == "WITHDRAW_OK":
        if len(parts) != 3:
            raise ValueError(f"respuesta WITHDRAW_OK mal formada: {text!r}")
        try:
            amount = float(parts[1])
            balance = float(parts[2])
        except ValueError as exc:
            raise ValueError(f"montos invalidos en WITHDRAW_OK: {text!r}") from exc
        return Respuesta(action, amount=amount, balance=balance)
    raise ValueError(f"respuesta desconocida: {text!r}")


def mostrar_respuesta(respuesta: Respuesta) -> None:
    """Presenta al usuario el resultado de la ultima peticion enviada."""
    if respuesta.action == "WITHDRAW_OK":
        print(f"\n[APLICACION] Retire su dinero: ${respuesta.amount:.2f}")
        print(f"[APLICACION] Saldo restante: ${respuesta.balance:.2f}")
    elif respuesta.action == "ERROR":
        print(f"\n[APLICACION] ERROR: {respuesta.message}")
    else:
        print(f"\n[APLICACION] {respuesta.message}")
