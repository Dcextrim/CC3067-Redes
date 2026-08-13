// Command atm es el cajero (ATM) del laboratorio: un host adjunto a un
// router-gateway que habla el protocolo de aplicacion Cajero/Banco del
// Lab 2 sobre el plano de datos Link State (DATA) en vez de una conexion
// TCP directa. Cada comando se envia por separado a traves del gateway y
// la respuesta llega de forma independiente al listener propio del cajero.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cc3067/lab3/internal/atm"
	"cc3067/lab3/internal/commandui"
	router "cc3067/lab3/internal/router"
	"cc3067/lab3/internal/router/algorithms"
	"cc3067/lab3/internal/router/control"
)

const responseTimeout = 5 * time.Second

func main() {
	selfIP := flag.String("ip", "127.0.0.1", "IP del cajero")
	selfPort := flag.Int("port", 6000, "puerto que identifica al cajero")
	gatewayIP := flag.String("gateway-ip", "127.0.0.1", "IP del router-gateway")
	gatewayPort := flag.Int("gateway-port", 5000, "puerto del router-gateway")
	bankID := flag.String("bank", "127.0.0.1:6001", "ip:puerto canonico del banco")
	noiseText := flag.String("noise", "", "probabilidad de flip por bit; si se omite, pregunta en modo interactivo")
	flag.Parse()

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

	selfID := router.NodeID(*selfIP, *selfPort)
	responses := make(chan control.Payload, 8)
	go func() {
		err := router.RunServer(*selfIP, *selfPort, func(payload control.Payload) {
			responses <- payload
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "[CAJERO] no se pudo escuchar respuestas:", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("[CAJERO] %s listo, banco en %s via gateway %s:%d\n", selfID, *bankID, *gatewayIP, *gatewayPort)

	c := &cajero{
		reader:           reader,
		selfID:           selfID,
		bankID:           *bankID,
		gatewayIP:        *gatewayIP,
		gatewayPort:      *gatewayPort,
		noiseProbability: noiseProbability,
		responses:        responses,
	}
	c.flujoLogin()
	c.flujoMenu()
}

// cajero agrupa el estado necesario para conversar con el banco: identidad
// propia, gateway, y el canal donde el listener en segundo plano deposita
// las respuestas que van llegando.
type cajero struct {
	reader           *bufio.Reader
	selfID           string
	bankID           string
	gatewayIP        string
	gatewayPort      int
	noiseProbability float64
	responses        chan control.Payload
}

// enviarComando manda un comando al banco y bloquea hasta recibir la
// respuesta correlacionada (o hasta agotar responseTimeout).
func (c *cajero) enviarComando(text string) (atm.Response, error) {
	flips, err := router.SendViaGatewayWithNoise(c.selfID, c.gatewayIP, c.gatewayPort, c.bankID, text, c.noiseProbability)
	if err != nil {
		return atm.Response{}, err
	}
	if flips > 0 {
		fmt.Printf("[RUIDO] %d bit(s) volteado(s) antes de llegar al gateway\n", flips)
	}
	select {
	case payload := <-c.responses:
		return atm.ParseResponse(payload.Msg)
	case <-time.After(responseTimeout):
		return atm.Response{}, fmt.Errorf("sin respuesta del banco tras %s", responseTimeout)
	}
}

func (c *cajero) prompt(label string) string {
	fmt.Print(label)
	text, _ := c.reader.ReadString('\n')
	return strings.TrimSpace(text)
}

// flujoLogin reintenta el login hasta autenticarse; un LOGIN no tiene
// efectos secundarios en el banco, asi que reintentar es seguro.
func (c *cajero) flujoLogin() {
	for {
		card := c.prompt("Numero de tarjeta: ")
		pin := c.prompt("PIN: ")
		response, err := c.enviarComando(atm.BuildLogin(card, pin))
		if err != nil {
			fmt.Printf("[APLICACION] ERROR: %v\n", err)
			continue
		}
		mostrarRespuesta(response)
		if response.Action == "LOGIN_OK" {
			return
		}
		fmt.Println("[APLICACION] Intente de nuevo.")
	}
}

// flujoMenu atiende el menu de retiro/salida hasta que el usuario cierra
// sesion.
func (c *cajero) flujoMenu() {
	for {
		fmt.Println("\n1) Retirar\n2) Salir")
		switch c.prompt("Opcion: ") {
		case "1":
			amountText := c.prompt("Monto a retirar: ")
			amount, err := strconv.ParseFloat(amountText, 64)
			if err != nil || amount <= 0 {
				fmt.Println("[APLICACION] ERROR: monto invalido")
				continue
			}
			response, err := c.enviarComando(atm.BuildWithdraw(amount))
			if err != nil {
				fmt.Printf("[APLICACION] ERROR: %v\n", err)
				continue
			}
			mostrarRespuesta(response)
		case "2":
			response, err := c.enviarComando(atm.BuildLogout())
			if err != nil {
				fmt.Printf("[APLICACION] ERROR: %v\n", err)
				return
			}
			mostrarRespuesta(response)
			return
		default:
			fmt.Println("[APLICACION] Opcion invalida.")
		}
	}
}

func mostrarRespuesta(r atm.Response) {
	if r.Action == "WITHDRAW_OK" {
		fmt.Printf("[APLICACION] Retiro exitoso: $%.2f (saldo: $%.2f)\n", r.Amount, r.Balance)
		return
	}
	fmt.Printf("[APLICACION] %s: %s\n", r.Action, r.Message)
}
