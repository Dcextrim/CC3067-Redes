# Protocolo Link State — CC3067 Lab 3

Este documento formaliza el protocolo implementado en Go por
`internal/router`, la compatibilidad depende de este contrato de red y no del lenguaje usado por
cada grupo.

## 1. Parametros generales

| Campo | Valor |
|---|---|
| Transporte | TCP, una conexion corta por mensaje (conectar, enviar una linea, cerrar) |
| Formato | JSON serializado, UTF-8 |
| Delimitador | `\n` al final de cada mensaje |
| Campo clave | `"type"` obligatorio en todo mensaje: `HELLO`, `LSA`, `DATA` |
| **Identificador de nodo** | `ip:puerto` (ver seccion 2) |

### Por que `ip:puerto` y no solo IP

en `127.0.0.1` todos los nodos comparten la misma IP y solo se distinguen por el puerto. Por eso
el identificador canonico de cualquier nodo (router u host adjunto) es
`f"{ip}:{port}"`, tanto en pruebas locales como en Tailscale donde sigue
siendo unico trivialmente, ya que ahi la IP sola ya lo era.

## 2. Configuracion de nodo (`config.json`)

```json
{
  "name": "A",
  "ip": "100.x.x.x",
  "port": 5000,
  "noise_probability": 0,
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

`noise_probability` es local a cada router y admite un decimal entre 0 y 1.
Se aplica de forma independiente a cada bit de DATA que ese router envia. Si
se omite vale 0. En una terminal, si no se proporciona `-noise`, el programa
pregunta si se desea activar la simulacion; Enter o `n` selecciona el modo sin
ruido. `-noise 0` fuerza ese modo sin preguntar y `-noise 1/1000` activa la
simulacion. La opcion no forma parte de los mensajes del protocolo, por lo que
cada grupo puede elegirla sin romper interoperabilidad.

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
1. Mantener la secuencia maxima aceptada para cada `origin`.
2. Aceptar un LSA solo cuando `seq` sea mayor que la secuencia maxima de ese
   origen; guardar `links` en el grafo local y reenviar a todos los vecinos
   **excepto** al que lo envio (campo `from` recibido).
3. Antes de reenviar, actualizar `from` a la identidad propia.
4. Si `seq` es duplicado o menor que el maximo ya aceptado, descartar en
   silencio. Un paquete atrasado nunca puede revertir el grafo.
5. `seq` solo se incrementa al generar un LSA propio, nunca al reenviar.

### C. DATA (datos)

Transporta `{from, to, msg}` protegido con Hamming(7,4) sobre el frame
completo, no solo sobre `msg`. Los routers intermedios *deben* corregir errores sobre todos los bits recibidos antes
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

**Pipeline en cada router al recibir DATA**:

1. Recibir la cadena de bits (`bits`).
2. Corregir errores sobre **todos** los bits (Hamming, usando `len`).
3. Extraer los bits de datos ya corregidos.
4. Deserializar el JSON resultante y leer **solo** el campo `to`.
5. Leer `<nodo>_tabla_enrutamiento.csv` y consultar ese destino. El plano de
   datos abre el CSV para cada DATA; no usa un mapa privado como sustituto.
6. Obtener IP y puerto del siguiente salto.
7. Re-serializar el mismo frame (sin tocar `msg`).
8. Aplicar Hamming(7,4) de nuevo sobre el frame completo (nueva `bits`).
9. Aplicar ruido a la trama ya protegida: cada bit, incluidos los de paridad,
   se voltea independientemente con `noise_probability`.
10. Enviar por un socket TCP nuevo hacia el siguiente salto.

El re-empaquetado en los pasos 7-9 importa: si Hamming corrigio un bit en
el paso 2, el frame reenviado queda "limpio" en vez de arrastrar errores
previos — cada enlace obtiene su propia proteccion Hamming fresca y una nueva
aplicacion independiente de ruido.

Hamming y LSA/HELLO **no** comparten tratamiento: HELLO y LSA viajan en
texto plano JSON; Hamming y el ruido simulado se aplican unicamente al plano
de datos (DATA).

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

El archivo se genera primero con un nombre temporal y luego reemplaza la tabla
anterior de forma atomica. Asi, un forwarding concurrente nunca observa un CSV
parcialmente escrito.

## 5. Tiempos y comportamiento

| Evento | Regla |
|---|---|
| HELLO | Cada 10s. Primer envio inmediato al iniciar el nodo. |
| LSA (inicial) | Generar y floodear 5s despues del primer HELLO recibido de cualquier vecino (exigido por el enunciado). |
| LSA (por cambio de topologia) | Si un vecino inactivo vuelve a responder HELLO, o un vecino activo deja de responder (ver expiracion abajo), se reconstruye y floodea un LSA nuevo con `seq` incrementado. Con un debounce minimo de 3s para no inundar la red si varios cambios ocurren juntos. |
| Expiracion de vecino | Si no llega HELLO de un vecino activo en 3 ciclos de HELLO (30s), se marca como caido (se retira de `links` en el proximo LSA propio). Revisado cada 5s. |
| Convergencia inicial | Esperar 30s antes de calcular y escribir la primera tabla de ruteo (exigido por el enunciado). |
| Recalculo de rutas | Tras la convergencia inicial, la tabla se recalcula de inmediato cada vez que cambia el grafo (LSA nuevo aceptado, vecino caido/recuperado) y, como respaldo, cada 15s. El CSV solo se reescribe si el resultado realmente cambio. |
| Hamming | Solo en plano de datos (DATA). HELLO y LSA van en texto plano. |
| Ruido | Se aplica despues de Hamming, antes de cada envio DATA; 0 por defecto. |
| Concurrencia | El nodo Go corre routing (control) y forwarding (datos) en goroutines separadas, comunicadas por canales internos; otra goroutine acepta conexiones TCP, una vigila la expiracion de vecinos y otra recalcula rutas periodicamente. |

## 6. Puertos

Rango acordado entre las 2 parejas reales: **5000-5007**
para routers, un puerto TCP dedicado por nodo router (indispensable en la
fase de pruebas locales, donde todos comparten `127.0.0.1`; en Tailscale
cada nodo ademas tiene una IP unica, pero se mantiene 1 puerto por nodo para
no branchear logica entre las dos fases). Los hosts adjuntos (cliente/servidor)
usan un rango aparte, **6000+**, para no competir por los 8 puertos de router.

Como el equipo confirmo con el profesor que puede operar como 2 parejas
reales y simular la tercera corriendo nodos adicionales, se usa el siguiente
reparto, validado en una prueba basica de interoperabilidad por Tailscale:

| Bloque | Asignado a |
|---|---|
| 5000-5002 | Pareja 1 — Daniel Chet / Cristian Tunchez (hasta 3 routers) |
| 5003-5005 | Pareja 2 — Javier Linares / Dilary Cruz (hasta 3 routers) |
| 5006-5007 | Pareja 3 simulada (nodos extra, ejecutados por quien corra la prueba) |

## 7. Vectores canonicos de interoperabilidad

`internal/router/testdata/protocol_vectors.json` contiene ejemplos
canonicos y autocontenidos de:

- HELLO y LSA serializados como JSON UTF-8 compacto.
- El vector manual Hamming(7,4) `1011 -> 0110011`.
- Un payload `{from,to,msg}`, sus bits UTF-8, el envelope DATA protegido y su
  JSON exterior.
