// Command server recibe mensajes DATA entregados por su router-gateway.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	router "cc3067/lab3/go-side/internal/router"
	"cc3067/lab3/go-side/internal/router/control"
)

func main() {
	listenIP := flag.String("ip", "127.0.0.1", "IP donde escucha el servidor")
	listenPort := flag.Int("port", 6001, "puerto donde escucha el servidor")
	flag.Parse()

	address := router.NodeID(*listenIP, *listenPort)
	fmt.Printf("Servidor escuchando DATA en %s\n", address)
	onMessage := func(payload control.Payload) {
		raw, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Println(string(raw))
	}
	if err := router.RunServer(*listenIP, *listenPort, onMessage); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
