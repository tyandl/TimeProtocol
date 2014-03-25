package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"time"
)

/**
 * An implementation of a Time Protocol Server (RFC 868)
 * http://tools.ietf.org/html/rfc868
 * @Author: Timothy Yandl (University of Portland)
 * @Date: 8 October 2013
**/

func main() {
	//log.Println("Starting...")
	go func() {
		if err := handleUDP(); err != nil {
			log.Fatal("could not start UDP server: ", err)
		}
	}()
	if err := handleTCP(); err != nil {
		log.Fatal("could not start TCP server: ", err)
	}
}

// function to handle UDP time requests. The server listens on port 37 for time requests 
// which will be empty datagram packets. When a packet is recieved, the time should be
// sent back as a 32 bit binary number. If the time can not be determined, 
// ignore the request.
func handleUDP() error {
	addr, err := net.ResolveUDPAddr("udp", ":37")
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err != nil {
		return err
	}
	buf := make([]byte, 1024)
	// wait for time requests
	for {
		if _, r_addr, err := conn.ReadFromUDP(buf); err == nil {
			if t, err := getTime(); err == nil {
				conn.WriteToUDP(t, r_addr)
			}
		}
	}
}

// function to handle TCP time requests. The server listens on port 37 for time requests.
// once connection is established, reply with the time as a 32 bit binary number and
// immediately close the connection. If the time can not be determined, 
// ignore the request and close the connection.
func handleTCP() error {
	addr, err := net.ResolveTCPAddr("tcp", ":37")
	if err != nil {
		return err
	}
	
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	
	for {
		conn, err := listener.AcceptTCP()
		if err == nil {
			go func(conn *net.TCPConn){
				if t, err := getTime(); err == nil {
					conn.Write(t)
				}
				conn.Close()
			}(conn)
		}
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



