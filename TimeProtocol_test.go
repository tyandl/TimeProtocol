package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// decodeRFC868 mirrors the decoding logic in Client/Client.go so tests can
// verify what actually goes out on the wire.
func decodeRFC868(buf []byte) (time.Time, error) {
	var t uint32
	if err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &t); err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(t)-2208988800, 0), nil
}

func assertCloseToNow(t *testing.T, got time.Time) {
	t.Helper()
	if diff := time.Since(got); diff < -2*time.Second || diff > 2*time.Second {
		t.Fatalf("decoded time %v too far from now (diff %v)", got, diff)
	}
}

func TestGetTime(t *testing.T) {
	out, err := getTime()
	if err != nil {
		t.Fatalf("getTime returned error: %v", err)
	}
	if len(out) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(out))
	}
	got, err := decodeRFC868(out)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	assertCloseToNow(t, got)
}

func TestServeUDP(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveUDP(ctx, conn) }()

	client, err := net.Dial("udp", conn.LocalAddr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	if _, err := client.Write(make([]byte, 0)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4)
	if n, err := client.Read(buf); err != nil {
		t.Fatalf("read response: %v", err)
	} else if n != 4 {
		t.Fatalf("expected 4 bytes, got %d", n)
	}

	got, err := decodeRFC868(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertCloseToNow(t, got)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveUDP returned error after cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveUDP did not shut down after context cancellation")
	}
}

func TestServeTCP(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveTCP(ctx, listener) }()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read response: %v", err)
	}

	got, err := decodeRFC868(buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertCloseToNow(t, got)

	// The server closes the connection right after replying.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected EOF after response, got %v", err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveTCP returned error after cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveTCP did not shut down after context cancellation")
	}
}
