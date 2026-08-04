"""Punto de entrada interactivo del cajero (cliente TCP)."""

import argparse
import socket
import threading

from atm.layers import application, link, noise, presentation
from atm.layers.transmission import TransmissionLayer


def receive_loop(transport: TransmissionLayer) -> None:
    while True:
        try:
            received = transport.recibir_informacion()
            if received is None:
                print("\n[TRANSMISION] El servidor cerro la conexion.")
                return
            integrity = link.verificar_integridad(
                received.frame_bits, received.algorithm, received.message_bit_length
            )
            if not integrity.ok:
                application.mostrar_mensaje(error=integrity.error or "error de integridad")
                continue
            try:
                message = presentation.decodificar_mensaje(integrity.message_bits)
            except ValueError as exc:
                application.mostrar_mensaje(error=str(exc))
                continue
            application.mostrar_mensaje(message, corrected=integrity.corrected)
        except (ConnectionError, OSError, ValueError) as exc:
            application.mostrar_mensaje(error=f"recepcion fallida: {exc}")
            return


def run(host: str, port: int) -> None:
    with socket.create_connection((host, port)) as sock:
        print(f"[CAJERO] Conectado a {host}:{port}")
        transport = TransmissionLayer(sock)
        receiver = threading.Thread(target=receive_loop, args=(transport,), daemon=True)
        receiver.start()

        while receiver.is_alive():
            try:
                request = application.solicitar_mensaje()
                probability = noise.parse_probability(request.probability_text)
                message_bits = presentation.codificar_mensaje(request.message)
                frame_bits = link.calcular_integridad(message_bits, request.algorithm)
                noisy_frame, flips = noise.aplicar_ruido(frame_bits, probability)
                transport.enviar_informacion(request.algorithm, len(message_bits), noisy_frame)
                print(
                    f"[RUIDO] {flips} bit(s) volteado(s) de {len(frame_bits)}; "
                    f"redundancia={len(frame_bits) - len(message_bits)} bits"
                )
            except EOFError:
                print("[CAJERO] Fin de envio.")
                try:
                    sock.shutdown(socket.SHUT_WR)
                except OSError:
                    pass
                receiver.join(timeout=2)
                return
            except ValueError as exc:
                application.mostrar_mensaje(error=str(exc))
            except OSError as exc:
                application.mostrar_mensaje(error=f"envio fallido: {exc}")
                return


def main() -> None:
    parser = argparse.ArgumentParser(description="Cajero CC3067 - Laboratorio 2")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=9000)
    args = parser.parse_args()
    run(args.host, args.port)


if __name__ == "__main__":
    main()
