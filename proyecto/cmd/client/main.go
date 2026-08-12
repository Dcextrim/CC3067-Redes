// Command client envia un mensaje DATA a traves de su router-gateway.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"cc3067/lab3/internal/commandui"
	router "cc3067/lab3/internal/router"
	"cc3067/lab3/internal/router/algorithms"
)

func main() {
	selfIP := flag.String("ip", "127.0.0.1", "IP del cliente")
	selfPort := flag.Int("port", 6000, "puerto que identifica al cliente")
	gatewayIP := flag.String("gateway-ip", "127.0.0.1", "IP del router-gateway")
	gatewayPort := flag.Int("gateway-port", 5000, "puerto del router-gateway")
	destination := flag.String("to", "", "destino canonico ip:puerto")
	message := flag.String("message", "", "texto a enviar; si se omite, se solicita por consola")
	noiseText := flag.String("noise", "", "probabilidad de flip por bit; si se omite, pregunta en modo interactivo")
	flag.Parse()

	if *destination == "" {
		fmt.Fprintln(os.Stderr, "falta -to ip:puerto")
		flag.Usage()
		os.Exit(2)
	}
	reader := bufio.NewReader(os.Stdin)
	noiseProbability := 0.0
	var err error
	if *noiseText != "" {
		noiseProbability, err = algorithms.ParseProbability(*noiseText)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else if commandui.IsInteractive(os.Stdin) {
		noiseProbability, err = commandui.PromptNoiseProbability(reader, os.Stdout, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	commandui.PrintNoiseMode(os.Stdout, noiseProbability)

	if *message == "" {
		if rest := strings.TrimSpace(strings.Join(flag.Args(), " ")); rest != "" {
			*message = rest
		} else {
			fmt.Print("Mensaje: ")
			text, err := reader.ReadString('\n')
			if err != nil && text == "" {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			*message = strings.TrimSpace(text)
		}
	}
	selfID := router.NodeID(*selfIP, *selfPort)
	flips, err := router.SendViaGatewayWithNoise(selfID, *gatewayIP, *gatewayPort, *destination, *message, noiseProbability)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Mensaje enviado desde %s hacia %s via %s:%d (ruido=%g, flips=%d)\n", selfID, *destination, *gatewayIP, *gatewayPort, noiseProbability, flips)
}
