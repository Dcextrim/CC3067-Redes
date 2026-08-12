// Package algorithms contiene Hamming, Dijkstra y el simulador de ruido.
package algorithms

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// ParseProbability acepta una tasa decimal o una fraccion como 1/100.
func ParseProbability(value string) (float64, error) {
	text := strings.TrimSpace(value)
	var probability float64
	var err error
	if strings.Contains(text, "/") {
		parts := strings.SplitN(text, "/", 2)
		var numerator, denominator float64
		numerator, err = strconv.ParseFloat(parts[0], 64)
		if err == nil {
			denominator, err = strconv.ParseFloat(parts[1], 64)
		}
		if err != nil || denominator == 0 {
			return 0, fmt.Errorf("use una probabilidad decimal o una fraccion como 1/100")
		}
		probability = numerator / denominator
	} else {
		probability, err = strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, fmt.Errorf("use una probabilidad decimal o una fraccion como 1/100")
		}
	}
	if probability < 0 || probability > 1 {
		return 0, fmt.Errorf("la probabilidad debe estar entre 0 y 1")
	}
	return probability, nil
}

// AplicarRuido evalua cada bit de forma independiente y cuenta los flips.
func AplicarRuido(bits string, probability float64, source *rand.Rand) (string, int, error) {
	if probability < 0 || probability > 1 {
		return "", 0, fmt.Errorf("la probabilidad debe estar entre 0 y 1")
	}
	if source == nil {
		return "", 0, fmt.Errorf("se requiere una fuente aleatoria")
	}
	output := []byte(bits)
	flips := 0
	// La fuente inyectable permite pruebas y experimentos deterministas.
	for index, bit := range output {
		if bit != '0' && bit != '1' {
			return "", 0, fmt.Errorf("la cadena solo puede contener bits 0 y 1")
		}
		if source.Float64() < probability {
			if bit == '0' {
				output[index] = '1'
			} else {
				output[index] = '0'
			}
			flips++
		}
	}
	return string(output), flips, nil
}
