package application

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"cc3067/lab2/go-side/internal/layers/link"
)

type OutgoingRequest struct {
	Message         string
	Algorithm       string
	ProbabilityText string
}

func readLine(reader *bufio.Reader, writer io.Writer, prompt string) (string, error) {
	fmt.Fprint(writer, prompt)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

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
