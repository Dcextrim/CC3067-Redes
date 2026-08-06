"""Capa de Presentacion: texto ASCII <-> bits."""


def codificar_mensaje(message: str) -> str:
    """Codifica cada caracter ASCII como un octeto binario MSB primero."""
    try:
        raw = message.encode("ascii")
    except UnicodeEncodeError as exc:
        raise ValueError("el mensaje debe contener unicamente caracteres ASCII") from exc
    return "".join(f"{byte:08b}" for byte in raw)


def decodificar_mensaje(bits: str) -> str:
    """Reconstruye texto ASCII solo cuando la longitud y los valores son validos."""
    if any(bit not in "01" for bit in bits):
        raise ValueError("la cadena solo puede contener bits 0 y 1")
    if len(bits) % 8:
        raise ValueError("la cantidad de bits no es multiplo de 8")
    raw = bytes(int(bits[i : i + 8], 2) for i in range(0, len(bits), 8))
    try:
        return raw.decode("ascii")
    except UnicodeDecodeError as exc:
        raise ValueError("los bits recibidos no representan ASCII valido") from exc
