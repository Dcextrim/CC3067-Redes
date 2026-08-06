// Package application interpreta y responde los comandos del cajero: login, retiro y logout.
package application

import (
	"fmt"
	"strconv"
	"strings"
)

// Request representa un comando de cajero ya separado en campos.
type Request struct {
	Action string
	Card   string
	Pin    string
	Amount float64
}

// ParseRequest interpreta el texto plano recibido de Aplicacion en el cajero.
func ParseRequest(text string) (Request, error) {
	parts := strings.Split(text, "|")
	switch parts[0] {
	case "LOGIN":
		if len(parts) != 3 {
			return Request{}, fmt.Errorf("comando LOGIN invalido")
		}
		return Request{Action: "LOGIN", Card: parts[1], Pin: parts[2]}, nil
	case "WITHDRAW":
		if len(parts) != 2 {
			return Request{}, fmt.Errorf("comando WITHDRAW invalido")
		}
		amount, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return Request{}, fmt.Errorf("monto invalido")
		}
		return Request{Action: "WITHDRAW", Amount: amount}, nil
	case "LOGOUT":
		return Request{Action: "LOGOUT"}, nil
	default:
		return Request{}, fmt.Errorf("comando desconocido: %s", parts[0])
	}
}

// Response representa una respuesta automatica ya lista para codificar.
type Response struct {
	Action  string
	Message string
	Amount  float64
	Balance float64
}

// Encode produce el texto plano que atraviesa Presentacion.
func (r Response) Encode() string {
	if r.Action == "WITHDRAW_OK" {
		return fmt.Sprintf("WITHDRAW_OK|%.2f|%.2f", r.Amount, r.Balance)
	}
	return fmt.Sprintf("%s|%s", r.Action, r.Message)
}

// MostrarEvento registra en consola cada peticion atendida automaticamente.
func MostrarEvento(remote, requestText, responseText string) {
	fmt.Printf("[APLICACION] %s -> %s | respuesta: %s\n", remote, requestText, responseText)
}
