"""Prueba TCP real end-to-end: cajero Python contra servidor bancario Go."""

from pathlib import Path
import os
import socket
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
PYTHON_SIDE = ROOT / "python-side"
GO_SIDE = ROOT / "go-side"
sys.path.insert(0, str(PYTHON_SIDE))

from atm.layers import application, link, presentation  # noqa: E402
from atm.layers.transmission import TransmissionLayer  # noqa: E402

CARD = "4111111111111111"
PIN = "1234"
OTHER_CARD = "5500005555555559"
OTHER_PIN = "0000"


def free_port() -> int:
    """Reserva temporalmente un puerto local y devuelve el numero asignado."""
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def connect_with_retry(port: int, timeout: float = 15.0) -> socket.socket:
    """Espera a que el proceso Go termine de compilar e inicie la escucha."""
    deadline = time.monotonic() + timeout
    last_error: OSError | None = None
    while time.monotonic() < deadline:
        try:
            return socket.create_connection(("127.0.0.1", port), timeout=1)
        except OSError as exc:
            last_error = exc
            time.sleep(0.1)
    raise RuntimeError(f"el servidor Go no abrio el puerto {port}: {last_error}")


def enviar_comando(transport: TransmissionLayer, algorithm: str, command_text: str) -> application.Respuesta:
    """Recorre Presentacion/Enlace/Transmision sin ruido y parsea la respuesta real."""
    message_bits = presentation.codificar_mensaje(command_text)
    frame_bits = link.calcular_integridad(message_bits, algorithm)
    transport.enviar_informacion(algorithm, len(message_bits), frame_bits)

    received = transport.recibir_informacion()
    if received is None:
        raise AssertionError("el servidor cerro la conexion antes de responder")
    integrity = link.verificar_integridad(received.frame_bits, received.algorithm, received.message_bit_length)
    if not integrity.ok:
        raise AssertionError(f"el cliente rechazo la respuesta: {integrity.error}")
    text = presentation.decodificar_mensaje(integrity.message_bits)
    return application.parse_respuesta(text)


def run_case(server_binary: Path, algorithm: str) -> None:
    """Recorre login exitoso, retiro exitoso, fondos insuficientes y logout."""
    port = free_port()
    process = subprocess.Popen(
        [str(server_binary), "--host", "127.0.0.1", "--port", str(port)],
        cwd=GO_SIDE,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        encoding="utf-8",
    )
    client: socket.socket | None = None
    try:
        client = connect_with_retry(port)
        transport = TransmissionLayer(client)

        respuesta = enviar_comando(transport, algorithm, application.comando_login("0000000000000000", "9999"))
        if respuesta.action != "LOGIN_DENIED":
            raise AssertionError(f"se esperaba LOGIN_DENIED, se obtuvo: {respuesta}")

        respuesta = enviar_comando(transport, algorithm, application.comando_login(CARD, PIN))
        if respuesta.action != "LOGIN_OK":
            raise AssertionError(f"se esperaba LOGIN_OK, se obtuvo: {respuesta}")

        respuesta = enviar_comando(transport, algorithm, application.comando_retiro(1_000_000))
        if respuesta.action != "WITHDRAW_ERROR":
            raise AssertionError(f"se esperaba WITHDRAW_ERROR, se obtuvo: {respuesta}")

        respuesta = enviar_comando(transport, algorithm, application.comando_retiro(50))
        if respuesta.action != "WITHDRAW_OK" or respuesta.amount != 50.00:
            raise AssertionError(f"se esperaba WITHDRAW_OK de 50.00, se obtuvo: {respuesta}")

        respuesta = enviar_comando(transport, algorithm, application.comando_logout())
        if respuesta.action != "LOGOUT_OK":
            raise AssertionError(f"se esperaba LOGOUT_OK, se obtuvo: {respuesta}")
    finally:
        if client is not None:
            client.close()
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
        if process.stdout and not process.stdout.closed:
            process.stdout.close()


def run_reaccept_case(server_binary: Path) -> None:
    """Comprueba que el servidor acepta clientes sucesivos, cada uno con su sesion."""
    port = free_port()
    process = subprocess.Popen(
        [str(server_binary), "--host", "127.0.0.1", "--port", str(port)],
        cwd=GO_SIDE,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        encoding="utf-8",
    )
    output = ""
    try:
        for card, pin in ((CARD, PIN), (OTHER_CARD, OTHER_PIN)):
            client = connect_with_retry(port)
            try:
                transport = TransmissionLayer(client)
                respuesta = enviar_comando(transport, "crc32", application.comando_login(card, pin))
                if respuesta.action != "LOGIN_OK":
                    raise AssertionError(f"login fallo para {card}: {respuesta}")
                respuesta = enviar_comando(transport, "crc32", application.comando_logout())
                if respuesta.action != "LOGOUT_OK":
                    raise AssertionError(f"logout fallo para {card}: {respuesta}")
            finally:
                client.close()
        time.sleep(0.3)
        process.terminate()
        output, _ = process.communicate(timeout=5)
        if output.count("LOGOUT_OK") < 2:
            raise AssertionError(f"el servidor no atendio dos sesiones sucesivas. Salida:\n{output}")
    finally:
        if process.poll() is None:
            process.kill()
        if process.stdout and not process.stdout.closed:
            process.stdout.close()


def main() -> None:
    """Compila una vez el servidor y ejecuta todos los escenarios cruzados."""
    with tempfile.TemporaryDirectory(prefix="cc3067-lab2-") as temp_directory:
        suffix = ".exe" if os.name == "nt" else ""
        server_binary = Path(temp_directory) / f"lab2-server{suffix}"
        subprocess.run(
            ["go", "build", "-o", str(server_binary), "./cmd/server"],
            cwd=GO_SIDE,
            check=True,
        )
        for algorithm in ("hamming", "crc32"):
            run_case(server_binary, algorithm)
            print(f"OK transaccion completa (login/retiro/logout): {algorithm}")
        run_reaccept_case(server_binary)
        print("OK servidor en escucha para sesiones sucesivas")


if __name__ == "__main__":
    main()
