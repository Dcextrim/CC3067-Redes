# Cajero-Banco: mensajeria resiliente a errores

Simulacion de un cajero automatico (Python) y un servidor bancario (Go) que intercambian
mensajes ASCII por TCP a traves de un canal no confiable. Ambos extremos pueden enviar y
recibir, elegir entre Hamming o CRC-32 para verificar la integridad, y especificar una
probabilidad de bit flip distinta en cada envio.

Los archivos `legacy/client.py` y `legacy/server.py` son un cliente/servidor base anterior,
sin capas ni deteccion de errores, conservados sin cambios como referencia. La implementacion
nueva esta separada por capas en `python-side/` y `go-side/`.

## Estructura

```text
python-side/atm/          cajero Python y sus cinco capas
go-side/                  servidor Go y sus cinco capas
tests/                    unitarias Python e interoperabilidad TCP
docs/protocol.md          framing y convenciones bit a bit
scripts/                  experimentos y automatizacion de pruebas
data/                     resultados CSV reproducibles
figures/                  graficas generadas
legacy/                   cliente/servidor base sin capas, como referencia
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
