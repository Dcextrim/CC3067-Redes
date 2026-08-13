package atm

import "sync"

// Account es el estado hardcodeado de una cuenta bancaria de demostracion,
// las mismas dos cuentas usadas en el Lab 2.
type Account struct {
	PIN     string
	Balance float64
}

// Bank contiene el estado del servidor bancario: cuentas y sesiones activas
// por cajero. El Lab 2 guardaba la sesion en la variable local de una
// conexion TCP persistente; aqui cada DATA es independiente, asi que la
// tarjeta autenticada se guarda explicitamente por remitente (atmID).
type Bank struct {
	mu       sync.Mutex
	accounts map[string]*Account
	sessions map[string]string // atmID -> tarjeta autenticada
}

// NewBank crea un banco con las cuentas de demostracion del Lab 2.
func NewBank() *Bank {
	return &Bank{
		accounts: map[string]*Account{
			"4111111111111111": {PIN: "1234", Balance: 500.00},
			"5500005555555559": {PIN: "0000", Balance: 1200.50},
		},
		sessions: make(map[string]string),
	}
}

// Handle aplica la logica de negocio para el cajero atmID; equivalente al
// handleRequest del Lab 2, resolviendo la sesion autenticada por atmID en
// vez de por conexion TCP.
func (b *Bank) Handle(atmID string, request Request) Response {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch request.Action {
	case "LOGIN":
		acc, ok := b.accounts[request.Card]
		if ok && acc.PIN == request.Pin {
			b.sessions[atmID] = request.Card
			return Response{Action: "LOGIN_OK", Message: "Autenticacion exitosa"}
		}
		return Response{Action: "LOGIN_DENIED", Message: "Tarjeta o PIN invalidos"}

	case "WITHDRAW":
		card, authenticated := b.sessions[atmID]
		if !authenticated {
			return Response{Action: "ERROR", Message: "No ha iniciado sesion"}
		}
		if request.Amount <= 0 {
			return Response{Action: "WITHDRAW_ERROR", Message: "El monto debe ser mayor que cero"}
		}
		acc := b.accounts[card]
		if request.Amount > acc.Balance {
			return Response{Action: "WITHDRAW_ERROR", Message: "Fondos insuficientes"}
		}
		acc.Balance -= request.Amount
		return Response{Action: "WITHDRAW_OK", Amount: request.Amount, Balance: acc.Balance}

	case "LOGOUT":
		delete(b.sessions, atmID)
		return Response{Action: "LOGOUT_OK", Message: "Hasta luego"}

	default:
		return Response{Action: "ERROR", Message: "Comando desconocido"}
	}
}
