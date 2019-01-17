package kademlia_go

import (
	"log"
	"net"
)

// GetOutboundIP returns an internet facing IP for the local machine and a random port
func GetOutboundIP() string {

	conn, err := net.Dial("udp", "8.8.8.8:0")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ip := conn.LocalAddr().String()

	return ip
}
