"""Punto de entrada interactivo del cajero (cliente TCP)."""

import argparse
import socket

from atm.layers import application, link, noise, presentation
from atm.layers.transmission import TransmissionLayer


def enviar_comando(transport: TransmissionLayer, command_text: str) -> application.Respuesta:
    """Pide algoritmo/ruido, envia un comando y bloquea hasta recibir la respuesta."""
    params = application.solicitar_parametros_envio()
    probability = noise.parse_probability(params.probability_text)
    message_bits = presentation.codificar_mensaje(command_text)
    frame_bits = link.calcular_integridad(message_bits, params.algorithm)
    noisy_frame, flips = noise.aplicar_ruido(frame_bits, probability)
    transport.enviar_informacion(params.algorithm, len(message_bits), noisy_frame)
    print(
        f"[RUIDO] {flips} bit(s) volteado(s) de {len(frame_bits)}; "
        f"redundancia={len(frame_bits) - len(message_bits)} bits"
    )

    received = transport.recibir_informacion()
    if received is None:
        raise ConnectionError("el servidor cerro la conexion")
    integrity = link.verificar_integridad(
        received.frame_bits, received.algorithm, received.message_bit_length
    )
    if not integrity.ok:
        raise application.TransmisionCorrupta(integrity.error or "error de integridad")
    try:
        text = presentation.decodificar_mensaje(integrity.message_bits)
    except ValueError as exc:
        raise application.TransmisionCorrupta(str(exc)) from exc
    if integrity.corrected:
        print("[APLICACION] Hamming corrigio un bit en la respuesta")
    try:
        return application.parse_respuesta(text)
    except ValueError as exc:
        raise application.TransmisionCorrupta(str(exc)) from exc


def flujo_login(transport: TransmissionLayer) -> None:
    """Reintenta el login hasta autenticarse; un LOGIN no tiene efectos secundarios."""
    while True:
        credenciales = application.solicitar_credenciales()
        try:
            respuesta = enviar_comando(
                transport, application.comando_login(credenciales.card, credenciales.pin)
            )
        except application.TransmisionCorrupta as exc:
            print(f"\n[APLICACION] ERROR: {exc}")
            print("[APLICACION] El LOGIN no tiene efectos secundarios; reintentando es seguro.")
            continue
        application.mostrar_respuesta(respuesta)
        if respuesta.action == "LOGIN_OK":
            return
        print("[APLICACION] Intente de nuevo.")


def flujo_menu(transport: TransmissionLayer) -> None:
    """Atiende el menu de retiro/salida hasta que el usuario cierra sesion."""
    while True:
        opcion = application.solicitar_opcion_menu()
        if opcion == "1":
            try:
                amount = application.solicitar_monto()
            except ValueError as exc:
                print(f"\n[APLICACION] ERROR: {exc}")
                continue
            try:
                respuesta = enviar_comando(transport, application.comando_retiro(amount))
            except application.TransmisionCorrupta as exc:
                print(f"\n[APLICACION] ERROR: {exc}")
                print(
                    "[APLICACION] No se pudo confirmar el retiro; "
                    "verifique su saldo antes de reintentar."
                )
                continue
            application.mostrar_respuesta(respuesta)
        elif opcion == "2":
            try:
                respuesta = enviar_comando(transport, application.comando_logout())
            except application.TransmisionCorrupta as exc:
                print(f"\n[APLICACION] ERROR: {exc}")
                print("[APLICACION] Se cerrara la sesion localmente de todas formas.")
                return
            application.mostrar_respuesta(respuesta)
            return
        else:
            print("[APLICACION] Opcion invalida.")


def run(host: str, port: int) -> None:
    """Conecta el cajero y recorre login -> menu sobre la misma conexion TCP."""
    with socket.create_connection((host, port)) as sock:
        print(f"[CAJERO] Conectado a {host}:{port}")
        transport = TransmissionLayer(sock)
        try:
            flujo_login(transport)
            flujo_menu(transport)
        except (ConnectionError, OSError) as exc:
            print(f"\n[APLICACION] ERROR: conexion perdida: {exc}")


def main() -> None:
    """Procesa host/puerto configurables e inicia la aplicacion interactiva."""
    parser = argparse.ArgumentParser(description="Cajero CC3067 - Laboratorio 2")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=9000)
    args = parser.parse_args()
    run(args.host, args.port)


if __name__ == "__main__":
    main()
