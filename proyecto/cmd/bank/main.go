// Command bank es el servidor bancario del laboratorio: un host adjunto a
// un router-gateway que atiende el protocolo de aplicacion Cajero/Banco del
// Lab 2 (login, retiro, logout) sobre el plano de datos Link State en vez
// de una conexion TCP directa. Cada respuesta se envia de vuelta al cajero
// a traves del gateway propio del banco.
package main

import (
	"flag"
	"fmt"
	"os"

	"cc3067/lab3/internal/atm"
	router "cc3067/lab3/internal/router"
	"cc3067/lab3/internal/router/algorithms"
	"cc3067/lab3/internal/router/control"
)

func main() {
	listenIP := flag.String("ip", "127.0.0.1", "IP donde escucha el banco")
	listenPort := flag.Int("port", 6001, "puerto donde escucha el banco")
	gatewayIP := flag.String("gateway-ip", "127.0.0.1", "IP del router-gateway propio (para responder)")
	gatewayPort := flag.Int("gateway-port", 5005, "puerto del router-gateway propio (para responder)")
	noiseText := flag.String("noise", "", "probabilidad de flip por bit en las respuestas")
	flag.Parse()

	noiseProbability := 0.0
	if *noiseText != "" {
		var err error
		noiseProbability, err = algorithms.ParseProbability(*noiseText)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	selfID := router.NodeID(*listenIP, *listenPort)
	bank := atm.NewBank()
	fmt.Printf("[BANCO] Escuchando en %s, respondiendo via gateway %s:%d\n", selfID, *gatewayIP, *gatewayPort)

	onMessage := func(payload control.Payload) {
		request, err := atm.ParseRequest(payload.Msg)
		if err != nil {
			fmt.Printf("[BANCO] %s -> comando invalido (%v)\n", payload.From, err)
			return
		}
		response := bank.Handle(payload.From, request)
		fmt.Printf("[BANCO] %s -> %s | respuesta: %s\n", payload.From, payload.Msg, response.Encode())
		if _, err := router.SendViaGatewayWithNoise(selfID, *gatewayIP, *gatewayPort, payload.From, response.Encode(), noiseProbability); err != nil {
			fmt.Fprintf(os.Stderr, "[BANCO] no se pudo responder a %s: %v\n", payload.From, err)
		}
	}
	if err := router.RunServer(*listenIP, *listenPort, onMessage); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
