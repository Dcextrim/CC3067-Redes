"""Prueba TCP real en ambos sentidos entre el cliente Python y el servidor Go."""

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

from atm.layers import link, presentation  # noqa: E402
from atm.layers.transmission import TransmissionLayer  # noqa: E402


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


def run_case(server_binary: Path, algorithm: str) -> None:
    """Verifica Go->Python y Python->Go con un algoritmo sobre TCP real."""
    port = free_port()
    environment = os.environ.copy()
    environment["GODEBUG"] = ""
    process = subprocess.Popen(
        [str(server_binary), "--host", "127.0.0.1", "--port", str(port)],
        cwd=GO_SIDE,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        encoding="utf-8",
        env=environment,
    )
    output = ""
    client: socket.socket | None = None
    try:
        client = connect_with_retry(port)
        transport = TransmissionLayer(client)

        assert process.stdin is not None
        process.stdin.write(f"DESDE GO\n{algorithm}\n0\n")
        process.stdin.flush()

        received = transport.recibir_informacion()
        if received is None:
            raise AssertionError("Go cerro la conexion antes de enviar")
        checked = link.verificar_integridad(
            received.frame_bits, received.algorithm, received.message_bit_length
        )
        if not checked.ok:
            raise AssertionError(f"Python rechazo la trama de Go: {checked.error}")
        if presentation.decodificar_mensaje(checked.message_bits) != "DESDE GO":
            raise AssertionError("Python no decodifico el texto enviado por Go")

        python_bits = presentation.codificar_mensaje("DESDE PYTHON")
        python_frame = link.calcular_integridad(python_bits, algorithm)
        transport.enviar_informacion(algorithm, len(python_bits), python_frame)
        client.shutdown(socket.SHUT_WR)

        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            time.sleep(0.1)
            if process.poll() is not None:
                break
        process.terminate()
        output, _ = process.communicate(timeout=5)
        if "Mensaje recibido" not in output or "DESDE PYTHON" not in output:
            raise AssertionError(f"Go no mostro el mensaje de Python. Salida:\n{output}")
    finally:
        if client is not None:
            client.close()
        if process.poll() is None:
            process.kill()
        if process.stdin and not process.stdin.closed:
            process.stdin.close()
        if process.stdout and not process.stdout.closed:
            process.stdout.close()


def run_reaccept_case(server_binary: Path) -> None:
    """Comprueba que el servidor vuelve a accept despues de una desconexion."""
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
        for index, algorithm in enumerate(("hamming", "crc32"), start=1):
            client = connect_with_retry(port)
            try:
                transport = TransmissionLayer(client)
                message_bits = presentation.codificar_mensaje(f"CLIENTE {index}")
                frame = link.calcular_integridad(message_bits, algorithm)
                transport.enviar_informacion(algorithm, len(message_bits), frame)
                client.shutdown(socket.SHUT_WR)
                time.sleep(0.3)
            finally:
                client.close()
        time.sleep(0.5)
        process.terminate()
        output, _ = process.communicate(timeout=5)
        if "CLIENTE 1" not in output or "CLIENTE 2" not in output:
            raise AssertionError(f"el servidor no acepto dos conexiones sucesivas. Salida:\n{output}")
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
            print(f"OK interoperabilidad bidireccional: {algorithm}")
        run_reaccept_case(server_binary)
        print("OK servidor en escucha para conexiones sucesivas")


if __name__ == "__main__":
    main()
