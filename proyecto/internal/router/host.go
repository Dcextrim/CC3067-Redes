package router

import (
	"bufio"
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	"time"

	"cc3067/lab3/internal/router/control"
	"cc3067/lab3/internal/router/forwarding"
)

// SendViaGateway arma el DATA inicial y lo entrega al router-gateway local.
// selfID / toID son identificadores "ip:puerto" (ver NodeID), iguales a los
// que el router uso para este host en su LSA.
func SendViaGateway(selfID, gatewayIP string, gatewayPort int, toID, text string) error {
	_, err := SendViaGatewayWithNoise(selfID, gatewayIP, gatewayPort, toID, text, 0)
	return err
}

// SendViaGatewayWithNoise protege el DATA y despues simula ruido en el enlace
// cliente-gateway. Retorna cuantos bits fueron alterados antes del socket.
func SendViaGatewayWithNoise(selfID, gatewayIP string, gatewayPort int, toID, text string, noiseProbability float64) (int, error) {
	envelope, err := forwarding.EncodeEnvelope(control.Payload{From: selfID, To: toID, Msg: text})
	if err != nil {
		return 0, err
	}
	envelope, flips, err := forwarding.ApplyNoise(
		envelope,
		noiseProbability,
		rand.New(rand.NewSource(time.Now().UnixNano())),
	)
	if err != nil {
		return 0, err
	}
	address := NodeID(gatewayIP, gatewayPort)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	raw, err := json.Marshal(envelope)
	if err != nil {
		return 0, err
	}
	_, err = conn.Write(append(raw, '\n'))
	return flips, err
}

// RunServer escucha DATA que el router-gateway local ya resolvio hasta este host.
func RunServer(listenIP string, listenPort int, onMessage func(control.Payload)) error {
	address := NodeID(listenIP, listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	return ServeHost(listener, onMessage)
}

// ServeHost atiende DATA sobre un listener ya creado. Separar el listener de
// la logica permite cerrar limpiamente servidores en pruebas end-to-end.
func ServeHost(listener net.Listener, onMessage func(control.Payload)) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go func(c net.Conn) {
			defer c.Close()
			line, err := bufio.NewReader(c).ReadString('\n')
			if err != nil && line == "" {
				return
			}
			var envelope control.DataEnvelope
			if err := json.Unmarshal([]byte(line), &envelope); err != nil {
				return
			}
			payload, err := forwarding.DecodePayload(envelope)
			if err == nil {
				onMessage(payload)
			}
		}(conn)
	}
}
