// Package application implementa la interfaz de usuario del servidor bancario.
package application

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"cc3067/lab2/go-side/internal/layers/link"
)

// OutgoingRequest agrupa los valores solicitados antes de recorrer las capas.
type OutgoingRequest struct {
	Message         string
	Algorithm       string
	ProbabilityText string
}

func readLine(reader *bufio.Reader, writer io.Writer, prompt string) (string, error) {
	// Reader se inyecta para desacoplar la consola de las pruebas y del transporte.
	fmt.Fprint(writer, prompt)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// SolicitarMensaje obtiene texto, algoritmo y tasa sin codificar ni transmitir.
func SolicitarMensaje(reader *bufio.Reader, writer io.Writer) (OutgoingRequest, error) {
	message, err := readLine(reader, writer, "Mensaje ASCII (o /salir): ")
	if err != nil {
		return OutgoingRequest{}, err
	}
	if message == "/salir" {
		return OutgoingRequest{}, io.EOF
	}
	algorithmText, err := readLine(reader, writer, "Algoritmo [1=hamming, 2=crc32]: ")
	if err != nil {
		return OutgoingRequest{}, err
	}
	algorithm, err := link.NormalizeAlgorithm(algorithmText)
	if err != nil {
		return OutgoingRequest{}, err
	}
	probability, err := readLine(reader, writer, "Probabilidad de error [decimal o fraccion, ej. 1/100]: ")
	if err != nil {
		return OutgoingRequest{}, err
	}
	return OutgoingRequest{message, algorithm, probability}, nil
}

// MostrarMensaje presenta una entrega valida o un error de las capas inferiores.
func MostrarMensaje(message, errorMessage string, corrected bool) {
	if errorMessage != "" {
		fmt.Printf("\n[APLICACION] ERROR: %s\n", errorMessage)
		return
	}
	suffix := ""
	if corrected {
		suffix = " (Hamming corrigio un bit)"
	}
	fmt.Printf("\n[APLICACION] Mensaje recibido%s: %s\n", suffix, message)
}
