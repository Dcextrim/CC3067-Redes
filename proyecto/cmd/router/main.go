// Punto de entrada CLI de un nodo router (ver docs/protocol.md).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"cc3067/lab3/internal/commandui"
	router "cc3067/lab3/internal/router"
	"cc3067/lab3/internal/router/algorithms"
)

func main() {
	configPath := flag.String("config", "", "Ruta a un config.json; si se omite, se pide interactivamente")
	noiseOverride := flag.String("noise", "", "probabilidad DATA: decimal o fraccion; reemplaza el valor del config")
	statusPort := flag.Int("status-port", -1, "puerto HTTP opcional para /status y /send, fuera del protocolo (monitoreo externo); -1 = puerto+2000 automatico, 0 = desactivado")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)
	var config router.NodeConfig
	var err error
	if *configPath != "" {
		config, err = router.LoadConfig(*configPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		config = promptConfig(reader)
	}
	if *noiseOverride != "" {
		config.NoiseProbability, err = algorithms.ParseProbability(*noiseOverride)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else if commandui.IsInteractive(os.Stdin) {
		config.NoiseProbability, err = commandui.PromptNoiseProbability(reader, os.Stdout, config.NoiseProbability)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	commandui.PrintNoiseMode(os.Stdout, config.NoiseProbability)

	effectiveStatusPort := *statusPort
	if effectiveStatusPort == -1 {
		effectiveStatusPort = config.Port + 2000
	}

	node := router.NewNode(config, "")
	if effectiveStatusPort != 0 {
		node.StartStatusServer(effectiveStatusPort)
		fmt.Printf("[STATUS] http://%s:%d/status (sin autenticacion, fuera del protocolo)\n", config.IP, effectiveStatusPort)
	}
	node.RunForever()
}

func promptConfig(reader *bufio.Reader) router.NodeConfig {
	prompt := func(label string) string {
		fmt.Print(label)
		text, _ := reader.ReadString('\n')
		return strings.TrimSpace(text)
	}

	name := prompt("Node name (a single letter, e.g. A): ")
	fmt.Printf("[ROUTER %s] No routing table found (%s_tabla_enrutamiento.csv). Let's configure this node.\n", name, name)
	ip := prompt("This router's IP: ")
	if ip == "" {
		ip = "127.0.0.1"
	}
	port, _ := strconv.Atoi(prompt("This router's listen port: "))

	var neighbors []router.Neighbor
	fmt.Println("Enter neighbors (blank IP to finish):")
	for {
		neighborIP := prompt("  Neighbor IP: ")
		if neighborIP == "" {
			break
		}
		neighborPort, _ := strconv.Atoi(prompt("  Neighbor port: "))
		costText := prompt("  Link cost [1]: ")
		cost := 1
		if costText != "" {
			cost, _ = strconv.Atoi(costText)
		}
		neighbors = append(neighbors, router.Neighbor{IP: neighborIP, Port: neighborPort, Cost: cost})
	}

	var attachedHost *router.AttachedHost
	role := strings.ToUpper(prompt("Attached host? (C=Client, S=Server, N=None): "))
	if role == "C" || role == "S" {
		hostIP := prompt(fmt.Sprintf("  %s host IP: ", role))
		hostPort, _ := strconv.Atoi(prompt(fmt.Sprintf("  %s host port: ", role)))
		roleName := "client"
		if role == "S" {
			roleName = "server"
		}
		attachedHost = &router.AttachedHost{Role: roleName, IP: hostIP, Port: hostPort, Cost: 1}
	}
	return router.NodeConfig{
		Name: name, IP: ip, Port: port, Neighbors: neighbors, AttachedHost: attachedHost,
	}
}
