package atm

import "testing"

func TestBankLoginWrongPin(t *testing.T) {
	bank := NewBank()
	response := bank.Handle("atm1", Request{Action: "LOGIN", Card: "4111111111111111", Pin: "0000"})
	if response.Action != "LOGIN_DENIED" {
		t.Fatalf("resultado inesperado: %+v", response)
	}
}

func TestBankLoginUnknownCard(t *testing.T) {
	bank := NewBank()
	response := bank.Handle("atm1", Request{Action: "LOGIN", Card: "0000000000000000", Pin: "1234"})
	if response.Action != "LOGIN_DENIED" {
		t.Fatalf("resultado inesperado: %+v", response)
	}
}

func TestBankWithdrawWithoutAuthentication(t *testing.T) {
	bank := NewBank()
	response := bank.Handle("atm1", Request{Action: "WITHDRAW", Amount: 10})
	if response.Action != "ERROR" {
		t.Fatalf("resultado inesperado: %+v", response)
	}
}

func TestBankFullSession(t *testing.T) {
	bank := NewBank()
	atmID := "127.0.0.1:6000"

	response := bank.Handle(atmID, Request{Action: "LOGIN", Card: "4111111111111111", Pin: "1234"})
	if response.Action != "LOGIN_OK" {
		t.Fatalf("login: resultado inesperado: %+v", response)
	}

	response = bank.Handle(atmID, Request{Action: "WITHDRAW", Amount: -5})
	if response.Action != "WITHDRAW_ERROR" {
		t.Fatalf("monto negativo: resultado inesperado: %+v", response)
	}

	response = bank.Handle(atmID, Request{Action: "WITHDRAW", Amount: 10000})
	if response.Action != "WITHDRAW_ERROR" {
		t.Fatalf("fondos insuficientes: resultado inesperado: %+v", response)
	}

	response = bank.Handle(atmID, Request{Action: "WITHDRAW", Amount: 100})
	if response.Action != "WITHDRAW_OK" || response.Amount != 100 || response.Balance != 400 {
		t.Fatalf("retiro valido: resultado inesperado: %+v", response)
	}

	response = bank.Handle(atmID, Request{Action: "LOGOUT"})
	if response.Action != "LOGOUT_OK" {
		t.Fatalf("logout: resultado inesperado: %+v", response)
	}

	// tras el logout, la sesion ya no esta autenticada
	response = bank.Handle(atmID, Request{Action: "WITHDRAW", Amount: 10})
	if response.Action != "ERROR" {
		t.Fatalf("retiro post-logout: resultado inesperado: %+v", response)
	}
}

func TestBankSessionsAreIndependentPerATM(t *testing.T) {
	bank := NewBank()

	response := bank.Handle("atm-A", Request{Action: "LOGIN", Card: "4111111111111111", Pin: "1234"})
	if response.Action != "LOGIN_OK" {
		t.Fatalf("login atm-A: resultado inesperado: %+v", response)
	}

	// atm-B nunca hizo login: debe ser rechazado aunque atm-A si tenga sesion.
	response = bank.Handle("atm-B", Request{Action: "WITHDRAW", Amount: 10})
	if response.Action != "ERROR" {
		t.Fatalf("retiro atm-B sin sesion: resultado inesperado: %+v", response)
	}
}
