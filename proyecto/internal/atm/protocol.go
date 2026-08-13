// Package atm implementa el protocolo de aplicacion Cajero/Banco heredado
// del Lab 2. A diferencia de ese laboratorio (conexion TCP directa), aqui
// los comandos viajan como texto plano en el campo "msg" de un DATA del
// plano de datos Link State (ver internal/router/control): los routers
// intermedios solo leen "to" para enrutar, nunca interpretan "msg".
package atm

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

// ParseRequest interpreta el texto plano recibido del cajero.
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

// BuildLogin arma el comando LOGIN que el cajero envia al banco.
func BuildLogin(card, pin string) string { return fmt.Sprintf("LOGIN|%s|%s", card, pin) }

// BuildWithdraw arma el comando WITHDRAW que el cajero envia al banco.
func BuildWithdraw(amount float64) string { return fmt.Sprintf("WITHDRAW|%.2f", amount) }

// BuildLogout arma el comando LOGOUT que el cajero envia al banco.
func BuildLogout() string { return "LOGOUT" }

// Response representa una respuesta del banco ya lista para codificar.
type Response struct {
	Action  string
	Message string
	Amount  float64
	Balance float64
}

// Encode produce el texto plano que viaja de vuelta al cajero.
func (r Response) Encode() string {
	if r.Action == "WITHDRAW_OK" {
		return fmt.Sprintf("WITHDRAW_OK|%.2f|%.2f", r.Amount, r.Balance)
	}
	return fmt.Sprintf("%s|%s", r.Action, r.Message)
}

// ParseResponse interpreta el texto plano recibido del banco.
func ParseResponse(text string) (Response, error) {
	action := text
	if idx := strings.IndexByte(text, '|'); idx >= 0 {
		action = text[:idx]
	}
	if action == "WITHDRAW_OK" {
		fields := strings.Split(text, "|")
		if len(fields) != 3 {
			return Response{}, fmt.Errorf("respuesta WITHDRAW_OK invalida: %s", text)
		}
		amount, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return Response{}, fmt.Errorf("monto invalido en respuesta: %s", text)
		}
		balance, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			return Response{}, fmt.Errorf("saldo invalido en respuesta: %s", text)
		}
		return Response{Action: action, Amount: amount, Balance: balance}, nil
	}
	parts := strings.SplitN(text, "|", 2)
	if len(parts) != 2 {
		return Response{}, fmt.Errorf("respuesta invalida: %s", text)
	}
	return Response{Action: parts[0], Message: parts[1]}, nil
}
