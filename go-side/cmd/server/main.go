package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"

	"cc3067/lab2/go-side/internal/layers/application"
	"cc3067/lab2/go-side/internal/layers/link"
	"cc3067/lab2/go-side/internal/layers/noise"
	"cc3067/lab2/go-side/internal/layers/presentation"
	"cc3067/lab2/go-side/internal/layers/transmission"
)

func receiveLoop(layer *transmission.Layer, done chan<- struct{}) {
	defer close(done)
	for {
		received, err := layer.RecibirInformacion()
		if err != nil {
			application.MostrarMensaje("", "recepcion fallida: "+err.Error(), false)
			return
		}
		if received == nil {
			fmt.Println("\n[TRANSMISION] El cajero cerro la conexion.")
			return
		}
		integrity := link.VerificarIntegridad(received.FrameBits, received.Algorithm, received.MessageBitLength)
		if !integrity.OK {
			application.MostrarMensaje("", integrity.Error, false)
			continue
		}
		message, err := presentation.DecodificarMensaje(integrity.MessageBits)
		if err != nil {
			application.MostrarMensaje("", err.Error(), false)
			continue
		}
		application.MostrarMensaje(message, "", integrity.Corrected)
	}
}

func sendRequest(layer *transmission.Layer, request application.OutgoingRequest, source *rand.Rand) error {
	probability, err := noise.ParseProbability(request.ProbabilityText)
	if err != nil {
		return err
	}
	messageBits, err := presentation.CodificarMensaje(request.Message)
	if err != nil {
		return err
	}
	frameBits, err := link.CalcularIntegridad(messageBits, request.Algorithm)
	if err != nil {
		return err
	}
	noisyFrame, flips, err := noise.AplicarRuido(frameBits, probability, source)
	if err != nil {
		return err
	}
	if err := layer.EnviarInformacion(request.Algorithm, len(messageBits), noisyFrame); err != nil {
		return fmt.Errorf("envio fallido: %w", err)
	}
	fmt.Printf("[RUIDO] %d bit(s) volteado(s) de %d; redundancia=%d bits\n", flips, len(frameBits), len(frameBits)-len(messageBits))
	return nil
}

func consoleLoop(reader *bufio.Reader, requests chan<- application.OutgoingRequest) {
	defer close(requests)
	for {
		request, err := application.SolicitarMensaje(reader, os.Stdout)
		if err != nil {
			if err == io.EOF {
				return
			}
			application.MostrarMensaje("", err.Error(), false)
			continue
		}
		requests <- request
	}
}

func handleConnection(connection net.Conn, requests <-chan application.OutgoingRequest) {
	defer connection.Close()
	layer := transmission.New(connection)
	done := make(chan struct{})
	go receiveLoop(layer, done)
	source := rand.New(rand.NewSource(time.Now().UnixNano()))
	activeRequests := requests

	for {
		select {
		case <-done:
			return
		case request, ok := <-activeRequests:
			if !ok {
				activeRequests = nil
				continue
			}
			if err := sendRequest(layer, request, source); err != nil {
				application.MostrarMensaje("", err.Error(), false)
				var networkError *net.OpError
				if errors.As(err, &networkError) {
					return
				}
			}
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
	requests := make(chan application.OutgoingRequest)
	go consoleLoop(bufio.NewReader(os.Stdin), requests)
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, "[SERVIDOR] Error de accept:", err)
			continue
		}
		fmt.Println("[SERVIDOR] Conexion de", connection.RemoteAddr())
		handleConnection(connection, requests)
	}
}
