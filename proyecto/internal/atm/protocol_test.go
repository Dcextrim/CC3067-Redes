package atm

import "testing"

func TestParseRequest(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		wantErr bool
		want    Request
	}{
		{"login valido", "LOGIN|4111111111111111|1234", false, Request{Action: "LOGIN", Card: "4111111111111111", Pin: "1234"}},
		{"withdraw valido", "WITHDRAW|100.00", false, Request{Action: "WITHDRAW", Amount: 100.00}},
		{"logout valido", "LOGOUT", false, Request{Action: "LOGOUT"}},
		{"login con campos faltantes", "LOGIN|4111111111111111", true, Request{}},
		{"withdraw con monto no numerico", "WITHDRAW|mucho", true, Request{}},
		{"accion desconocida", "TRANSFER|100", true, Request{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseRequest(c.text)
			if c.wantErr {
				if err == nil {
					t.Fatalf("se esperaba error para %q", c.text)
				}
				return
			}
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestBuildCommands(t *testing.T) {
	if got := BuildLogin("4111111111111111", "1234"); got != "LOGIN|4111111111111111|1234" {
		t.Fatalf("BuildLogin: got %q", got)
	}
	if got := BuildWithdraw(100); got != "WITHDRAW|100.00" {
		t.Fatalf("BuildWithdraw: got %q", got)
	}
	if got := BuildLogout(); got != "LOGOUT" {
		t.Fatalf("BuildLogout: got %q", got)
	}
}

func TestResponseEncode(t *testing.T) {
	cases := []struct {
		name string
		resp Response
		want string
	}{
		{"login ok", Response{Action: "LOGIN_OK", Message: "Autenticacion exitosa"}, "LOGIN_OK|Autenticacion exitosa"},
		{"login denied", Response{Action: "LOGIN_DENIED", Message: "Tarjeta o PIN invalidos"}, "LOGIN_DENIED|Tarjeta o PIN invalidos"},
		{"withdraw ok", Response{Action: "WITHDRAW_OK", Amount: 100, Balance: 400}, "WITHDRAW_OK|100.00|400.00"},
		{"withdraw error", Response{Action: "WITHDRAW_ERROR", Message: "Fondos insuficientes"}, "WITHDRAW_ERROR|Fondos insuficientes"},
		{"logout ok", Response{Action: "LOGOUT_OK", Message: "Hasta luego"}, "LOGOUT_OK|Hasta luego"},
		{"error", Response{Action: "ERROR", Message: "Comando desconocido"}, "ERROR|Comando desconocido"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.resp.Encode(); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestParseResponseRoundTrip(t *testing.T) {
	cases := []Response{
		{Action: "LOGIN_OK", Message: "Autenticacion exitosa"},
		{Action: "LOGIN_DENIED", Message: "Tarjeta o PIN invalidos"},
		{Action: "WITHDRAW_OK", Amount: 100, Balance: 400},
		{Action: "WITHDRAW_ERROR", Message: "Fondos insuficientes"},
		{Action: "LOGOUT_OK", Message: "Hasta luego"},
		{Action: "ERROR", Message: "Comando desconocido"},
	}
	for _, want := range cases {
		t.Run(want.Action, func(t *testing.T) {
			got, err := ParseResponse(want.Encode())
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got != want {
				t.Fatalf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestParseResponseInvalid(t *testing.T) {
	if _, err := ParseResponse("WITHDRAW_OK|100.00"); err == nil {
		t.Fatal("se esperaba error para WITHDRAW_OK con campos faltantes")
	}
	if _, err := ParseResponse("SINPIPE"); err == nil {
		t.Fatal("se esperaba error para respuesta sin separador")
	}
}
