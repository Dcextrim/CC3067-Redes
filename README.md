# Laboratorio 2 - Esquemas de deteccion y correccion de errores

Implementacion de CC3067 (UVG) con cajero en Python y servidor bancario en Go. Los dos extremos
pueden enviar y recibir mensajes ASCII, elegir Hamming o CRC-32 y especificar una probabilidad de
flip distinta en cada envio.

Los archivos `client.py` y `server.py` de la raiz son el material base del laboratorio anterior y
se conservaron sin cambios. La implementacion nueva esta separada por capas en `python-side/` y
`go-side/`.

## Estructura

```text
python-side/atm/          cajero Python y sus cinco capas
go-side/                  servidor Go y sus cinco capas
tests/                    unitarias Python e interoperabilidad TCP
docs/protocol.md          framing y convenciones bit a bit
scripts/                  experimentos y automatizacion de pruebas
data/                     resultados CSV reproducibles
figures/                  graficas generadas
```

## Ejecucion interactiva

Terminal 1, servidor Go (queda escuchando en el puerto elegido):

```powershell
Set-Location .\go-side
go run .\cmd\server --host 127.0.0.1 --port 9000
```

Terminal 2, cajero Python:

```powershell
$env:PYTHONPATH = ".\python-side"
python -m atm.main --host 127.0.0.1 --port 9000
```

Escriba un mensaje ASCII, seleccione `1`/`hamming` o `2`/`crc32` y una tasa como `0.01` o
`1/100`. Use `/salir` para terminar el envio. Ambos procesos muestran cuantos bits fueron
volteados y si el receptor corrigio o rechazo la trama.

## Pruebas y artefactos

En PowerShell, el flujo completo se ejecuta con:

```powershell
.\scripts\run_all.ps1
```

Tambien se incluye un `Makefile`: `make test`, `make experiments` y `make all`.
Las pruebas incluyen los vectores manuales documentados, errores de un bit en todas las posiciones,
longitudes genericas, framing TCP y comunicacion real Python-Go en ambas direcciones.

La especificacion completa del formato esta en [docs/protocol.md](docs/protocol.md).
