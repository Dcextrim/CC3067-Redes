# CC3067 - Laboratorio 3: Protocolos de Enrutamiento

Simulador de una red de routers Link State que descubre vecinos con HELLO,
distribuye LSA mediante flooding, calcula rutas con Dijkstra y transporta
mensajes DATA protegidos con Hamming(7,4).

La implementacion oficial utiliza exclusivamente **Go**. La interoperabilidad
con otros grupos se define por el protocolo JSON y los vectores canonicos, no
por compartir un lenguaje de programacion.

- [`proyecto/`](proyecto/) - codigo Go, configuraciones, pruebas, protocolo y
  una topologia local reproducible de seis routers.
- [`informe/`](informe/) - ubicacion reservada para el reporte final en PDF.

Ver [`proyecto/README.md`](proyecto/README.md) para compilar, ejecutar y probar
el sistema, y [`proyecto/docs/protocol.md`](proyecto/docs/protocol.md) para el
contrato de interoperabilidad.
