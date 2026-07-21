package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

/**
 * An implementation of a Time Protocol Server (RFC 868)
 * http://tools.ietf.org/html/rfc868
 * @Author: Timothy Yandl (University of Portland)
 * @Date: 8 October 2013
**/

// writeTimeout bounds how long a client has to receive its reply, so a
// connection that never reads can't pin a goroutine open indefinitely.
// Var (not const) so tests can shrink it.
var writeTimeout = 5 * time.Second

func main() {
	port := 37
	if p := os.Getenv("TIMEPROTOCOL_PORT"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil {
			log.Fatalf("invalid TIMEPROTOCOL_PORT %q: %v", p, err)
		}
		port = v
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 2)
	go func() { errc <- handleUDP(ctx, port) }()
	go func() { errc <- handleTCP(ctx, port) }()

	select {
	case <-ctx.Done():
		log.Println("shutting down...")
	case err := <-errc:
		if err != nil {
			log.Fatal(err)
		}
	}
}

// function to handle UDP time requests. The server listens for time requests
// which will be empty datagram packets. When a packet is recieved, the time should be
// sent back as a 32 bit binary number. If the time can not be determined,
// ignore the request.
func handleUDP(ctx context.Context, port int) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	return serveUDP(ctx, conn)
}

// serveUDP runs the UDP request loop against an already-bound connection.
// Split out from handleUDP so tests can bind an ephemeral port (":0") and
// exercise the loop directly.
func serveUDP(ctx context.Context, conn *net.UDPConn) error {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	// max theoretical UDP packet size: 65535
	// minus headers: 65507 (this is the max I could send on localhost)
	// Ethernet MTU: 1500 (unless jumbo frames are enabled)
	// If problems occure on Windows due to malformed time requests, set this higher.
	buf := make([]byte, 1500)
	// wait for time requests
	for {
		_, r_addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		if t, err := getTime(); err == nil {
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			conn.WriteToUDP(t, r_addr)
		}
	}
}

// function to handle TCP time requests. The server listens for time requests.
// once connection is established, reply with the time as a 32 bit binary number and
// immediately close the connection. If the time can not be determined,
// ignore the request and close the connection.
func handleTCP(ctx context.Context, port int) error {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	return serveTCP(ctx, listener)
}

// serveTCP runs the TCP accept loop against an already-bound listener. Split
// out from handleTCP so tests can bind an ephemeral port (":0") and exercise
// the loop directly.
func serveTCP(ctx context.Context, listener *net.TCPListener) error {
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		go func(conn *net.TCPConn) {
			defer conn.Close()
			// Bound the write so a client that opens a connection and never
			// reads can't hold this goroutine open forever (Slowloris-style).
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if t, err := getTime(); err == nil {
				conn.Write(t)
			}
		}(conn)
	}
}

func getTime() (out []byte, err error) {
	// get the current time and add the number of seconds from jan 1 1900
	// to jan 1 1970 (RFC868 uses windows epoch)
	buf := new(bytes.Buffer)
	t := int32((time.Now().Unix())+2208988800)
	if err := binary.Write(buf, binary.BigEndian, t); err != nil {
		return nil, err
	}
	out = buf.Bytes()
	buf.Reset()
	return out, err
}
