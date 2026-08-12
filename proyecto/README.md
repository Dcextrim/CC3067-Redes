# Router Link State en Go

Cada router ejecuta en paralelo el plano de control —HELLO, LSA, flooding, expiracion de vecinos y Dijkstra— y el
plano de datos —lectura del CSV, Hamming(7,4) y forwarding por TCP—.

## Estructura

- `cmd/router`: proceso router configurable.
- `cmd/client`: host cliente que envia DATA a su gateway.
- `cmd/server`: host servidor que recibe DATA desde su gateway.
- `cmd/experiment`: simulacion reproducible del canal ruidoso y Hamming.
- `internal/router`: protocolo, algoritmos y pruebas Go.
- `configs/local`: topologia reproducible de seis routers A-F.
- `configs/examples`: configuraciones sanitizadas para adaptar a Tailscale.
- `configs/demo`: configuraciones reales locales, ignoradas por Git.
- `docs/protocol.md`: contrato JSON que deben compartir las otras parejas.
- `internal/router/testdata/protocol_vectors.json`: vectores canonicos
  independientes del lenguaje.
- `experiments/data` y `experiments/figures`: resultados reproducibles y
  graficas SVG generadas con Go.

## Requisitos

- Go 1.22 o posterior.
- PowerShell 7 recomendado para los scripts de automatizacion en Windows.

No se requiere instalar dependencias externas.

## Compilar y probar

Desde `proyecto/`:

```powershell
go test -count=1 ./...
go build ./...
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
go run ./cmd/router
```

Con configuracion JSON:

```powershell
go run ./cmd/router -config configs/local/A.json
```

Al iniciar desde una terminal aparece esta eleccion:

```text
¿Activar simulacion de ruido en DATA? [s/N]:
```

Presione Enter o escriba `n` para usar la version sin ruido que ya fue probada
con la otra pareja. Esa es la opcion predeterminada. Si escribe `s`, puede
aceptar la probabilidad recomendada `1/1000` o ingresar otra.

Tambien se puede escoger el modo directamente, sin pregunta. Para la prueba de
interoperabilidad estable:

```powershell
go run ./cmd/router -config configs/local/A.json -noise 0
```

Para activar la simulacion opcional:

```powershell
go run ./cmd/router -config configs/local/A.json -noise 1/1000
```

La misma opcion puede guardarse como `"noise_probability": 0.001` en el JSON.
El ruido se aplica despues de Hamming en cada enlace DATA; HELLO y LSA no se
alteran.

Al converger escribe `<nombre>_tabla_enrutamiento.csv`. Cuando cambia la
topologia vuelve a anunciar su LSA, recalcula Dijkstra y reemplaza el CSV de
forma atomica. Cada trama DATA consulta literalmente ese archivo.

## Cliente y servidor

Servidor adjunto al router F:

```powershell
go run ./cmd/server -ip 127.0.0.1 -port 6001
```

Cliente adjunto al router A:

```powershell
go run ./cmd/client -ip 127.0.0.1 -port 6000 `
  -gateway-ip 127.0.0.1 -gateway-port 5000 `
  -to 127.0.0.1:6001 -message "hola"
```

El cliente hace la misma pregunta. Presione Enter para enviar sin ruido. El
cliente tambien representa un enlace y solo aplica ruido antes de enviar al
gateway cuando se elige esa version. Use `-noise 0` para automatizar una prueba
determinista sin preguntas.

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
internal/router/testdata/protocol_vectors.json
```

El archivo contiene JSON canonico para HELLO, LSA y DATA, un vector manual de
Hamming(7,4), los bits UTF-8 y el envelope completo. Se regenera con:

```powershell
go run ./cmd/vector-gen
```

## Experimento de ruido

Los CSV y SVG usados para analizar Hamming se regeneran con el mismo codigo Go
del proyecto:

```powershell
go run ./cmd/experiment
```

La corrida usa una semilla fija, 25 condiciones y 7500 transmisiones. Consulte
`experiments/README.md` para interpretar las columnas y la limitacion ante
varios flips dentro del mismo bloque de siete bits.

La interoperabilidad basica ya fue validada por Tailscale con un router remoto
y su servidor adjunto: HELLO, LSA, Dijkstra, CSV, Hamming y DATA funcionaron
entre ambas implementaciones. Para repetir una prueba, copie
`configs/examples/A.tailscale.example.json` a `configs/demo/A.json` y sustituya
las IP de ejemplo; `configs/demo/` no se versiona.
