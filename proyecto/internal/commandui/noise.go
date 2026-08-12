// Package commandui contiene interacciones compartidas por los ejecutables.
package commandui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"cc3067/lab3/internal/router/algorithms"
)

const defaultNoiseProbability = 0.001

// IsInteractive indica si el programa esta conectado directamente a una
// terminal. Los scripts con entrada redirigida no deben quedar esperando.
func IsInteractive(input *os.File) bool {
	info, err := input.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// PromptNoiseProbability permite escoger entre el modo compatible sin ruido y
// la simulacion opcional. La respuesta vacia siempre conserva el modo probado
// sin ruido.
func PromptNoiseProbability(reader *bufio.Reader, writer io.Writer, configured float64) (float64, error) {
	for {
		fmt.Fprint(writer, "¿Activar simulacion de ruido en DATA? [s/N]: ")
		answer, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return 0, err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "", "n", "no":
			return 0, nil
		case "s", "si", "sí", "y", "yes":
			return promptProbability(reader, writer, configured)
		default:
			fmt.Fprintln(writer, "Responda s para activar ruido o n para usar la version sin ruido.")
		}
		if err == io.EOF {
			return 0, nil
		}
	}
}

func promptProbability(reader *bufio.Reader, writer io.Writer, configured float64) (float64, error) {
	defaultProbability := configured
	defaultText := strconv.FormatFloat(configured, 'g', -1, 64)
	if configured <= 0 {
		defaultProbability = defaultNoiseProbability
		defaultText = "1/1000"
	}
	for {
		fmt.Fprintf(writer, "Probabilidad de flip por bit [%s]: ", defaultText)
		value, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return 0, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return defaultProbability, nil
		}
		probability, parseErr := algorithms.ParseProbability(value)
		if parseErr == nil {
			return probability, nil
		}
		fmt.Fprintf(writer, "Valor invalido: %v\n", parseErr)
		if err == io.EOF {
			return 0, parseErr
		}
	}
}

// PrintNoiseMode deja explicito el modo elegido antes de abrir sockets.
func PrintNoiseMode(writer io.Writer, probability float64) {
	if probability == 0 {
		fmt.Fprintln(writer, "Modo seleccionado: sin ruido (compatible con la prueba anterior).")
		return
	}
	fmt.Fprintf(writer, "Modo seleccionado: simulacion con ruido DATA (p=%g por bit).\n", probability)
}
