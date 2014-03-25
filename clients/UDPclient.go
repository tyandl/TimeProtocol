package main

import (
	"net"
	"fmt"
	"bytes"
	"encoding/binary"
	"time"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf( "Usage: %s <IP Address> <Port>\n", os.Args[0] )
		return
	}
	
	addr, err := net.ResolveUDPAddr("udp", os.Args[1] + ":" + os.Args[2])
	if err != nil {
		fmt.Println("Could not resolve address")
		return
	}
	
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("an error occured setting up the connection")
		return
	}
	
	// send a blank packet.
	_, err = conn.Write([]byte(""))
	if err != nil {
		fmt.Println("Unable to send packet: ", err)
	}
	
	// read response
	buf := make([]byte, 4)
	conn.Read(buf)
	var t uint32
	err = binary.Read(bytes.NewReader(buf), binary.BigEndian, &t)
	if err != nil {
		fmt.Println("Unable to read packet: ", err)
	}
	fmt.Printf("Received: %v\n", time.Unix(int64(t)-2208988800, 0))
	conn.Close()
}




