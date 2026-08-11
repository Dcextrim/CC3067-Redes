// Command client envia un mensaje DATA a traves de su router-gateway.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	router "cc3067/lab3/go-side/internal/router"
)

func main() {
	selfIP := flag.String("ip", "127.0.0.1", "IP del cliente")
	selfPort := flag.Int("port", 6000, "puerto que identifica al cliente")
	gatewayIP := flag.String("gateway-ip", "127.0.0.1", "IP del router-gateway")
	gatewayPort := flag.Int("gateway-port", 5000, "puerto del router-gateway")
	destination := flag.String("to", "", "destino canonico ip:puerto")
	message := flag.String("message", "", "texto a enviar; si se omite, se solicita por consola")
	flag.Parse()

	if *destination == "" {
		fmt.Fprintln(os.Stderr, "falta -to ip:puerto")
		flag.Usage()
		os.Exit(2)
	}
	if *message == "" {
		if rest := strings.TrimSpace(strings.Join(flag.Args(), " ")); rest != "" {
			*message = rest
		} else {
			fmt.Print("Mensaje: ")
			text, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil && text == "" {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			*message = strings.TrimSpace(text)
		}
	}

	selfID := router.NodeID(*selfIP, *selfPort)
	if err := router.SendViaGateway(selfID, *gatewayIP, *gatewayPort, *destination, *message); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Mensaje enviado desde %s hacia %s via %s:%d\n", selfID, *destination, *gatewayIP, *gatewayPort)
}
