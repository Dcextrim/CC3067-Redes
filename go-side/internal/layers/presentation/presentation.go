package presentation

import (
	"fmt"
	"strings"
)

// CodificarMensaje convierte cada byte ASCII en ocho bits, MSB primero.
func CodificarMensaje(message string) (string, error) {
	var builder strings.Builder
	builder.Grow(len(message) * 8)
	for _, value := range []byte(message) {
		if value > 127 {
			return "", fmt.Errorf("el mensaje debe contener unicamente caracteres ASCII")
		}
		builder.WriteString(fmt.Sprintf("%08b", value))
	}
	return builder.String(), nil
}

// DecodificarMensaje convierte grupos de ocho bits en texto ASCII.
func DecodificarMensaje(bits string) (string, error) {
	for _, bit := range bits {
		if bit != '0' && bit != '1' {
			return "", fmt.Errorf("la cadena solo puede contener bits 0 y 1")
		}
	}
	if len(bits)%8 != 0 {
		return "", fmt.Errorf("la cantidad de bits no es multiplo de 8")
	}
	result := make([]byte, len(bits)/8)
	for index := range result {
		var value byte
		for _, bit := range bits[index*8 : index*8+8] {
			value <<= 1
			value |= byte(bit - '0')
		}
		if value > 127 {
			return "", fmt.Errorf("los bits recibidos no representan ASCII valido")
		}
		result[index] = value
	}
	return string(result), nil
}
