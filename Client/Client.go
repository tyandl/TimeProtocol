package main

import (
	"net"
	"log"
	"fmt"
	"flag"
	"bytes"
	"encoding/binary"
	"time"
)

var net_type string
var ip string
var size int

func init() {
	flag.StringVar(&net_type, "net", "udp", "Set udp or tcp (default udp)")
	flag.StringVar(&ip, "ip", "localhost", "The host's ip address (default localhost)")
	flag.IntVar(&size, "size", 0, "set a packet size for testing malformed requests")
	flag.Parse()
}

func main() {
	fmt.Printf("\nSending %d bytes to %s in %s format\n", size, ip, net_type)
	
	conn, err := net.Dial(net_type, ip + ":37")
	if err != nil {
		log.Fatal("an error occured setting up the connection: ", err)
	}
	
	// send a blank packet.
	out := make([]byte, size)
	_, err = conn.Write(out)
	if err != nil {
		conn.Close()
		log.Fatal("Unable to send packet: ", err)
	}
	
	// read response
	buf := make([]byte, 4)
	conn.Read(buf)
	var t uint32
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &t)
	if err != nil {
		conn.Close()
		log.Fatal("Unable to read packet: ", err)
	}
	// print the result
	fmt.Printf("Received: %v\n\n", time.Unix(int64(t)-2208988800, 0))
	conn.Close()
}




