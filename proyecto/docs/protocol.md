# Protocolo Link State — CC3067 Lab 3

Este documento formaliza el protocolo implementado por `python-side/router` y
`go-side/internal/router`. Debe acordarse **sin cambios** con las demas
parejas de la topologia (interoperabilidad exigida por el enunciado, seccion
3.3): cualquier modificacion aqui rompe la compatibilidad con los routers de
otros grupos.

## 1. Parametros generales

| Campo | Valor |
|---|---|
| Transporte | TCP, una conexion corta por mensaje (conectar, enviar una linea, cerrar) |
| Formato | JSON serializado, UTF-8 |
| Delimitador | `\n` al final de cada mensaje |
| Campo clave | `"type"` obligatorio en todo mensaje: `HELLO`, `LSA`, `DATA` |
| **Identificador de nodo** | `ip:puerto` (ver seccion 2) |

### Por que `ip:puerto` y no solo IP

La propuesta original identificaba nodos solo por IP de Tailscale. Eso
funciona en la fase de red real (cada nodo tiene una IP de Tailscale unica),
pero **rompe la fase de pruebas locales** que exige el enunciado ("nuestro
medio de desarrollo sera nuestra computadora local"): en `127.0.0.1` todos
los nodos comparten la misma IP y solo se distinguen por el puerto. Por eso
el identificador canonico de cualquier nodo (router u host adjunto) es
`f"{ip}:{port}"`, tanto en pruebas locales como en Tailscale (donde sigue
siendo unico trivialmente, ya que ahi la IP sola ya lo era).

## 2. Configuracion de nodo (`config.json`)

```json
{
  "name": "A",
  "ip": "100.x.x.x",
  "port": 5000,
  "neighbors": [
    { "ip": "100.y.y.y", "port": 5001, "cost": 2 },
    { "ip": "100.z.z.z", "port": 5002, "cost": 5 }
  ],
  "attached_host": { "role": "client", "ip": "100.w.w.w", "port": 6000, "cost": 1 }
}
```

`attached_host` es opcional: identifica al cliente o servidor (no-router)
que usa a este nodo como puerta de enlace predeterminada (seccion 3.2 del
enunciado). El host adjunto se agrega como un vecino mas al grafo de
enrutamiento (con el costo indicado, 1 por defecto) — exactamente igual a
como aparece un router vecino.

## 3. Tipos de mensaje

### A. HELLO (control)

Enviado a todos los vecinos configurados. Confirma que el vecino esta vivo;
no requiere respuesta explicita. Cadencia: cada 10s, primer envio inmediato.

```json
{ "type": "HELLO", "from": "100.x.x.x:5000" }
```

### B. LSA — Link State Advertisement (control)

Se construye 5s despues de recibir el primer HELLO de un vecino, y se
inunda a toda la red por flooding.

```json
{
  "type": "LSA",
  "origin": "100.x.x.x:5000",
  "seq": 3,
  "links": { "100.y.y.y:5001": 2, "100.z.z.z:5002": 5 },
  "from": "100.x.x.x:5000"
}
```

- `origin`: nodo que genero el LSA. No cambia en ningun salto.
- `from`: nodo que reenvia. Se actualiza en cada salto a la IP:puerto propios.
- `links`: vecinos activos (que respondieron HELLO) + host adjunto si existe.

**Reglas de flooding:**
1. Mantener el conjunto de pares `(origin, seq)` ya procesados.
2. Si `(origin, seq)` es nuevo: guardar `links` en el grafo local y reenviar
   a todos los vecinos **excepto** al que lo envio (campo `from` recibido).
3. Antes de reenviar, actualizar `from` a la identidad propia.
4. Si ya se vio: descartar en silencio.
5. `seq` solo se incrementa al generar un LSA propio, nunca al reenviar.

### C. DATA (datos)

Transporta `{from, to, msg}` protegido con **Hamming(7,4) sobre el frame
completo**, no solo sobre `msg`. Este es el punto que se corrigio respecto
a la propuesta original tras aclaracion del profesor: los routers
intermedios *deben* corregir errores sobre todos los bits recibidos antes
de poder leer nada — pero solo **leen** el campo `to` para decidir el
siguiente salto; nunca interpretan ni actuan sobre `msg`. Unicamente el
destino final usa el contenido de `msg`.

Hamming(7,4) es un codigo **por bloques**: el frame completo se parte en
bloques de 4 bits de datos, cada uno se codifica a 7 bits (4 datos + 3
paridad), y el ultimo bloque se rellena con ceros si el frame no es
multiplo de 4 (el relleno se descarta al decodificar usando `len`). Esto
corrige **hasta un error de un bit por cada bloque de 7** — a diferencia de
un unico codigo Hamming generico sobre todo el frame, que solo tolera un
bit volteado en todo el mensaje sin importar su longitud.

```json
{ "type": "DATA", "len": 128, "bits": "0110011010..." }
```

- `bits`: `Hamming7_4( UTF8_bits( json.dumps({"from","to","msg"}) ) )`, con el
  frame partido en bloques de 4 bits antes de codificar cada uno a 7.
- `len`: longitud en bits del frame *sin* redundancia ni relleno (necesaria
  para que Hamming sepa cuantos bloques esperar al decodificar).

**Pipeline en cada router al recibir DATA** (igual al enunciado, seccion 3.2):

1. Recibir la cadena de bits (`bits`).
2. Corregir errores sobre **todos** los bits (Hamming, usando `len`).
3. Extraer los bits de datos ya corregidos.
4. Deserializar el JSON resultante y leer **solo** el campo `to`.
5. Consultar `<nodo>_tabla_enrutamiento.csv` con ese valor.
6. Obtener IP y puerto del siguiente salto.
7. Re-serializar el mismo frame (sin tocar `msg`).
8. Aplicar Hamming(7,4) de nuevo sobre el frame completo (nueva `bits`).
9. Enviar por un socket TCP nuevo hacia el siguiente salto.

El re-empaquetado en los pasos 7-8 importa: si Hamming corrigio un bit en
el paso 2, el frame reenviado queda "limpio" en vez de arrastrar errores
previos — cada enlace obtiene su propia proteccion Hamming fresca.

Hamming y LSA/HELLO **no** comparten tratamiento: HELLO y LSA viajan en
texto plano JSON; Hamming se aplica unicamente al plano de datos (DATA).

## 4. Tabla de ruteo generada

`<nombre_nodo>_tabla_enrutamiento.csv`, escrita por el plano de control una
vez transcurrida la espera de convergencia:

```csv
destination,next_hop_ip,next_hop_port,cost
100.y.y.y:5001,100.y.y.y,5001,2
100.z.z.z:5002,100.y.y.y,5001,7
```

`destination` es el identificador `ip:puerto` del nodo (router u host
adjunto); `next_hop_ip`/`next_hop_port` son la direccion real de socket del
primer salto (siempre un vecino directo).

## 5. Tiempos y comportamiento

| Evento | Regla |
|---|---|
| HELLO | Cada 10s. Primer envio inmediato al iniciar el nodo. |
| LSA | Generar y floodear 5s despues del primer HELLO recibido. |
| Convergencia | Esperar 30s antes de asumir que la tabla de ruteo es estable. |
| Hamming | Solo en plano de datos (DATA). HELLO y LSA van en texto plano. |
| Hilos | El nodo corre routing (control) y forwarding (datos) en hilos/goroutines separados, comunicados por colas internas; un tercer hilo de escucha acepta conexiones TCP y despacha cada mensaje a la cola que corresponda segun su `"type"`. |

## 6. Puertos

**Pendiente de fijar** hasta confirmar el numero final de nodos de la
topologia (ver nota del equipo sobre parejas incompletas). Cada nodo
(router u host) usa un puerto TCP propio — indispensable en la fase de
pruebas locales, donde todos comparten `127.0.0.1`. En Tailscale cada nodo
tiene ademas una IP unica, por lo que los puertos podrian repetirse entre
nodos sin ambiguedad, pero se recomienda mantener la asignacion 1 puerto
por nodo para no tener que branchear logica entre las dos fases de prueba.
