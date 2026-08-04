package transmission

import (
	"net"
	"testing"
)

func TestBinaryFrameAcrossStream(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	sender := New(left)
	receiver := New(right)
	errChannel := make(chan error, 1)
	go func() {
		errChannel <- sender.EnviarInformacion("hamming", 8, "10100101101")
	}()
	frame, err := receiver.RecibirInformacion()
	if err != nil {
		t.Fatal(err)
	}
	if sendErr := <-errChannel; sendErr != nil {
		t.Fatal(sendErr)
	}
	if frame.Algorithm != "hamming" || frame.MessageBitLength != 8 || frame.FrameBits != "10100101101" {
		t.Fatalf("trama inesperada: %+v", frame)
	}
}
