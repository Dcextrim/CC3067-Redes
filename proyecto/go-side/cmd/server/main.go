package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"cc3067/lab2/go-side/internal/layers/application"
	"cc3067/lab2/go-side/internal/layers/link"
	"cc3067/lab2/go-side/internal/layers/noise"
	"cc3067/lab2/go-side/internal/layers/presentation"
	"cc3067/lab2/go-side/internal/layers/transmission"
)

// account replica el estado hardcodeado del servidor base del laboratorio anterior.
type account struct {
	pin     string
	balance float64
}

var accounts = map[string]*account{
	"4111111111111111": {pin: "1234", balance: 500.00},
	"5500005555555559": {pin: "0000", balance: 1200.50},
}
var accountsMutex sync.Mutex

// handleRequest aplica la logica de negocio; es pura para poder probarla sin sockets.
func handleRequest(request application.Request, authenticatedCard string) (application.Response, string) {
	switch request.Action {
	case "LOGIN":
		accountsMutex.Lock()
		acc, ok := accounts[request.Card]
		accountsMutex.Unlock()
		if ok && acc.pin == request.Pin {
			return application.Response{Action: "LOGIN_OK", Message: "Autenticacion exitosa"}, request.Card
		}
		return application.Response{Action: "LOGIN_DENIED", Message: "Tarjeta o PIN invalidos"}, authenticatedCard

	case "WITHDRAW":
		if authenticatedCard == "" {
			return application.Response{Action: "ERROR", Message: "No ha iniciado sesion"}, authenticatedCard
		}
		if request.Amount <= 0 {
			return application.Response{Action: "WITHDRAW_ERROR", Message: "El monto debe ser mayor que cero"}, authenticatedCard
		}
		accountsMutex.Lock()
		defer accountsMutex.Unlock()
		acc := accounts[authenticatedCard]
		if request.Amount > acc.balance {
			return application.Response{Action: "WITHDRAW_ERROR", Message: "Fondos insuficientes"}, authenticatedCard
		}
		acc.balance -= request.Amount
		return application.Response{Action: "WITHDRAW_OK", Amount: request.Amount, Balance: acc.balance}, authenticatedCard

	case "LOGOUT":
		return application.Response{Action: "LOGOUT_OK", Message: "Hasta luego"}, ""

	default:
		return application.Response{Action: "ERROR", Message: "Comando desconocido"}, authenticatedCard
	}
}

// sendAutoReply codifica y envia una respuesta automatica sin exponerla a ruido.
func sendAutoReply(layer *transmission.Layer, algorithm string, response application.Response, source *rand.Rand) error {
	messageBits, err := presentation.CodificarMensaje(response.Encode())
	if err != nil {
		return err
	}
	frameBits, err := link.CalcularIntegridad(messageBits, algorithm)
	if err != nil {
		return err
	}
	noisyFrame, _, err := noise.AplicarRuido(frameBits, 0.0, source)
	if err != nil {
		return err
	}
	return layer.EnviarInformacion(algorithm, len(messageBits), noisyFrame)
}

// handleConnection atiende una conexion de cajero hasta LOGOUT o desconexion.
func handleConnection(connection net.Conn) {
	defer connection.Close()
	layer := transmission.New(connection)
	source := rand.New(rand.NewSource(time.Now().UnixNano()))
	authenticatedCard := ""
	remote := connection.RemoteAddr().String()

	for {
		received, err := layer.RecibirInformacion()
		if err != nil {
			fmt.Println("[SERVIDOR] Error de recepcion:", err)
			return
		}
		if received == nil {
			fmt.Println("[SERVIDOR] El cajero cerro la conexion:", remote)
			return
		}

		var response application.Response
		requestText := ""
		integrity := link.VerificarIntegridad(received.FrameBits, received.Algorithm, received.MessageBitLength)
		if !integrity.OK {
			response = application.Response{Action: "ERROR", Message: "Transmision corrupta, reintente"}
		} else if text, decodeErr := presentation.DecodificarMensaje(integrity.MessageBits); decodeErr != nil {
			response = application.Response{Action: "ERROR", Message: "Transmision corrupta, reintente"}
		} else if request, parseErr := application.ParseRequest(text); parseErr != nil {
			requestText = text
			response = application.Response{Action: "ERROR", Message: "Comando invalido"}
		} else {
			requestText = text
			response, authenticatedCard = handleRequest(request, authenticatedCard)
		}
		application.MostrarEvento(remote, requestText, response.Encode())

		if err := sendAutoReply(layer, received.Algorithm, response, source); err != nil {
			fmt.Println("[SERVIDOR] Error de envio:", err)
			return
		}
		if response.Action == "LOGOUT_OK" {
			return
		}
	}
}

func main() {
	host := flag.String("host", "127.0.0.1", "direccion en la cual escuchar")
	port := flag.Int("port", 9000, "puerto TCP")
	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "[SERVIDOR] No se pudo escuchar:", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Printf("[SERVIDOR] Escuchando en %s:%d\n", *host, *port)
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, "[SERVIDOR] Error de accept:", err)
			continue
		}
		fmt.Println("[SERVIDOR] Conexion de", connection.RemoteAddr())
		go handleConnection(connection)
	}
}
