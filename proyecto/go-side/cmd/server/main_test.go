package main

import (
	"testing"

	"cc3067/lab2/go-side/internal/layers/application"
)

func TestHandleRequestLoginWrongPin(t *testing.T) {
	response, card := handleRequest(application.Request{Action: "LOGIN", Card: "4111111111111111", Pin: "0000"}, "")
	if response.Action != "LOGIN_DENIED" || card != "" {
		t.Fatalf("resultado inesperado: %+v card=%q", response, card)
	}
}

func TestHandleRequestLoginUnknownCard(t *testing.T) {
	response, card := handleRequest(application.Request{Action: "LOGIN", Card: "0000000000000000", Pin: "1234"}, "")
	if response.Action != "LOGIN_DENIED" || card != "" {
		t.Fatalf("resultado inesperado: %+v card=%q", response, card)
	}
}

func TestHandleRequestWithdrawWithoutAuthentication(t *testing.T) {
	response, card := handleRequest(application.Request{Action: "WITHDRAW", Amount: 10}, "")
	if response.Action != "ERROR" || card != "" {
		t.Fatalf("resultado inesperado: %+v card=%q", response, card)
	}
}

func TestHandleRequestFullSession(t *testing.T) {
	accountsMutex.Lock()
	accounts["9999999999999999"] = &account{pin: "5555", balance: 100.00}
	accountsMutex.Unlock()

	response, card := handleRequest(application.Request{Action: "LOGIN", Card: "9999999999999999", Pin: "5555"}, "")
	if response.Action != "LOGIN_OK" || card != "9999999999999999" {
		t.Fatalf("login: resultado inesperado: %+v card=%q", response, card)
	}

	response, card = handleRequest(application.Request{Action: "WITHDRAW", Amount: -5}, card)
	if response.Action != "WITHDRAW_ERROR" {
		t.Fatalf("monto negativo: resultado inesperado: %+v", response)
	}

	response, card = handleRequest(application.Request{Action: "WITHDRAW", Amount: 1000}, card)
	if response.Action != "WITHDRAW_ERROR" {
		t.Fatalf("fondos insuficientes: resultado inesperado: %+v", response)
	}

	response, card = handleRequest(application.Request{Action: "WITHDRAW", Amount: 40}, card)
	if response.Action != "WITHDRAW_OK" || response.Amount != 40 || response.Balance != 60 {
		t.Fatalf("retiro valido: resultado inesperado: %+v", response)
	}

	response, card = handleRequest(application.Request{Action: "LOGOUT"}, card)
	if response.Action != "LOGOUT_OK" || card != "" {
		t.Fatalf("logout: resultado inesperado: %+v card=%q", response, card)
	}
}
