# Protocolo binario CC67 v1

## Decisiones de diseño

TCP entrega un flujo de octetos, no mensajes. Por ello, cada envio usa un encabezado fijo de
Transmision y un cuerpo de longitud explicita. El simulador de ruido se aplica antes de
Transmision, exclusivamente a la trama que Enlace entrega (datos y redundancia). El encabezado
no pertenece a esa trama y no recibe ruido; de otro modo, un flip en la longitud impediria
delimitar la siguiente trama.

Todos los enteros multiocteto usan orden de red (big-endian). No se envia texto `0`/`1`: el
cuerpo se empaqueta en octetos, con el primer bit en el MSB. Si la longitud no es multiplo de
ocho, los bits finales del ultimo octeto se rellenan con ceros y se descartan al recibir.

## Encabezado de Transmision (16 octetos)

| Offset | Tamaño | Campo | Valor |
|---:|---:|---|---|
| 0 | 4 | magic | ASCII `CC67` (`43 43 36 37`) |
| 4 | 1 | version | `01` |
| 5 | 1 | algorithm | `01` Hamming, `02` CRC-32 |
| 6 | 2 | flags | reservado, debe ser `0000` |
| 8 | 4 | message_bit_length | bits originales, sin redundancia |
| 12 | 4 | frame_bit_length | bits en el cuerpo, con redundancia |

Inmediatamente despues se envian `ceil(frame_bit_length / 8)` octetos.

## Presentacion

Cada caracter debe pertenecer a ASCII (0 a 127) y se codifica en ocho bits, MSB primero. Por
ejemplo, `A` se convierte en `01000001`. El receptor rechaza longitudes que no sean multiplos de
ocho y valores que no formen ASCII valido.

## Hamming generico SEC

Para `m` bits se elige el menor `r` que cumple `m + r + 1 <= 2^r`. Las posiciones de la palabra
codificada se numeran desde 1, de izquierda a derecha. Las potencias de dos (`1, 2, 4, ...`) son
bits de paridad par; las demas contienen los datos en orden. Enlace envia directamente los
`m + r` bits Hamming. El sindrome identifica y corrige cualquier error de exactamente un bit.

Caso verificable a mano, Hamming(7,4):

```text
datos:       1 0 1 1
posiciones:  1 2 3 4 5 6 7
contenido:  p1 p2 1 p4 0 1 1
p1 cubre 1,3,5,7 -> p1=0
p2 cubre 2,3,6,7 -> p2=1
p4 cubre 4,5,6,7 -> p4=0
trama:       0 1 1 0 0 1 1
```

Si se altera la posicion 5, el sindrome es `101b = 5`; al voltearla se recupera `1011`.
Hamming SEC no garantiza detectar dos o mas flips dentro de una misma palabra: puede producir
una correccion equivocada. Esa limitacion se mide como corrupcion silenciosa en los experimentos.

## CRC-32

Se implementa CRC-32/ISO-HDLC con la forma reflejada del polinomio IEEE
`0xEDB88320` (forma normal `0x04C11DB7`), valor inicial `0xFFFFFFFF` y XOR final
`0xFFFFFFFF`. Los datos se empaquetan MSB primero en octetos; el bucle reflejado procesa cada
octeto con el bit menos significativo primero. Si `n <= 32`, se agregan ceros a la derecha hasta
32 bits solo para calcular el CRC. Para `n > 32` no multiplo de ocho, se completa el ultimo
octeto con ceros solo durante el calculo. La longitud original no cambia.

El checksum se serializa como 32 bits, MSB primero, y se concatena al mensaje:

```text
trama_enlace = message_bits || crc_bits
```

Vector de control: ASCII `123456789` produce `0xCBF43926`, es decir,
`11001011111101000011100100100110`. Un mensaje de un bit `1` se completa como
`80 00 00 00` y produce `0xCC1D6927`.

El receptor separa los ultimos 32 bits, recalcula el checksum sobre los primeros
`message_bit_length` bits y compara. Si difieren, informa el error a Aplicacion y no decodifica.

## Flujo por capas

```text
Aplicacion -> Presentacion -> Enlace -> Ruido -> Transmision -> TCP
TCP -> Transmision -> Enlace -> Presentacion -> Aplicacion
```

Ambos extremos ejecutan los dos flujos: el cajero Python inicia la conexion y el servidor Go
permanece escuchando. En cada extremo un hilo/goroutine recibe mientras la consola permite enviar.
