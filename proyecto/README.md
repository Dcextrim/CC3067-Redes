# Cajero-Banco: mensajeria resiliente a errores

Simulacion de un cajero automatico (Python) que hace login con tarjeta+PIN y retira dinero
de un servidor bancario (Go), sobre TCP y a traves de un canal no confiable. Cada envio
(login, retiro, logout) elige entre Hamming o CRC-32 para verificar la integridad y una
probabilidad de bit flip propia. Las respuestas automaticas del banco viajan por el mismo
pipeline de capas, sin ruido.

Los archivos `legacy/client.py` y `legacy/server.py` son el cliente/servidor base original
(login+retiro sobre JSON plano, sin capas ni deteccion de errores) que se uso como punto de
partida, conservados sin cambios como referencia. La implementacion nueva envuelve ese mismo
flujo transaccional con las cinco capas, separadas en `python-side/` y `go-side/`.

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

El cajero pide numero de tarjeta y PIN (reintenta hasta autenticar); las cuentas de prueba
son `4111111111111111`/`1234` y `5500005555555559`/`0000`. Ya autenticado, el menu permite
retirar dinero o salir. Cada envio pide el algoritmo (`1`=hamming, `2`=crc32) y una tasa de
ruido como `0.01` o `1/100`. Ambos procesos muestran cuantos bits fueron volteados y si el
receptor corrigio o rechazo la trama.

## Pruebas y artefactos

En PowerShell, el flujo completo se ejecuta con:

```powershell
.\scripts\run_all.ps1
```

Tambien se incluye un `Makefile`: `make test`, `make experiments` y `make all`.
Las pruebas incluyen los vectores manuales documentados, errores de un bit en todas las posiciones,
longitudes genericas, framing TCP y una sesion real Python-Go extremo a extremo (login, retiro
exitoso y con fondos insuficientes, logout) sobre el algoritmo elegido.

La especificacion completa del formato esta en [docs/protocol.md](docs/protocol.md).
