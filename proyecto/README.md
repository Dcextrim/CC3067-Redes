# Router Link State en Go

Implementacion del Laboratorio 3 de CC3067. Cada router ejecuta en paralelo el
plano de control —HELLO, LSA, flooding, expiracion de vecinos y Dijkstra— y el
plano de datos —lectura del CSV, Hamming(7,4) y forwarding por TCP—.

## Estructura

- `go-side/cmd/router`: proceso router configurable.
- `go-side/cmd/client`: host cliente que envia DATA a su gateway.
- `go-side/cmd/server`: host servidor que recibe DATA desde su gateway.
- `go-side/internal/router`: protocolo, algoritmos y pruebas Go.
- `configs/local`: topologia reproducible de seis routers A-F.
- `docs/protocol.md`: contrato JSON que deben compartir las otras parejas.
- `go-side/internal/router/testdata/protocol_vectors.json`: vectores canonicos
  independientes del lenguaje.
- `data/` y `figures/`: resultados experimentales conservados para el reporte.

## Requisitos

- Go 1.22 o posterior.
- PowerShell 7 recomendado para los scripts de automatizacion en Windows.

No se requiere instalar dependencias externas.

## Compilar y probar

Desde `proyecto/`:

```powershell
cd go-side
go test -count=1 ./...
go build ./cmd/router ./cmd/client ./cmd/server
```

O ejecutar toda la validacion corta:

```powershell
.\scripts\run_all.ps1
```

Las pruebas incluyen algoritmos, flooding con secuencias, lectura del CSV,
convergencia real de routers por sockets, cliente-router-servidor end-to-end y
los vectores canonicos de interoperabilidad.

## Ejecutar un router

Sin archivo, el programa solicita nombre, IP, puerto, vecinos y host adjunto:

```powershell
cd go-side
go run ./cmd/router
```

Con configuracion JSON:

```powershell
cd go-side
go run ./cmd/router -config ../configs/local/A.json
```

Al converger escribe `<nombre>_tabla_enrutamiento.csv`. Cuando cambia la
topologia vuelve a anunciar su LSA, recalcula Dijkstra y reemplaza el CSV de
forma atomica. Cada trama DATA consulta literalmente ese archivo.

## Cliente y servidor

Servidor adjunto al router F:

```powershell
cd go-side
go run ./cmd/server -ip 127.0.0.1 -port 6001
```

Cliente adjunto al router A:

```powershell
cd go-side
go run ./cmd/client -ip 127.0.0.1 -port 6000 `
  -gateway-ip 127.0.0.1 -gateway-port 5000 `
  -to 127.0.0.1:6001 -message "hola"
```

## Topologia local completa

El siguiente comando compila los ejecutables, levanta seis routers y los dos
hosts, espera la convergencia con los tiempos de produccion, envia un mensaje y
comprueba que el servidor lo recibio:

```powershell
.\scripts\run_local_topology.ps1
```

La ruta optima configurada de A hacia F es A-B-D-E-F, con costo de routers 7.
Los logs y CSV de cada ejecucion quedan bajo `tmp/local_topology/`.

## Interoperabilidad con otros grupos

El protocolo de red esta especificado en `docs/protocol.md`. Otros equipos
pueden validar su implementacion con:

```text
go-side/internal/router/testdata/protocol_vectors.json
```

El archivo contiene JSON canonico para HELLO, LSA y DATA, un vector manual de
Hamming(7,4), los bits UTF-8 y el envelope completo. Se regenera con:

```powershell
cd go-side
go run ./cmd/vector-gen
```

## Pendiente externo

- Acordar el protocolo final con las otras parejas antes de la prueba en clase.
- Sustituir `127.0.0.1` por las IP de Tailscale en configuraciones de despliegue.
- Incorporar el reporte final PDF en `informe/`.
