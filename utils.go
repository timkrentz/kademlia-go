package kademlia_go

import (
	"log"
	"net"
)

// GetOutboundIP returns an internet facing IP for the local machine and a random port
func GetOutboundIP() string {

	conn, err := net.Dial("udp", "172.21.20.97:")
	if err != nil {
		log.Fatal(err)
	}

	ip := conn.LocalAddr().String()

	err = conn.Close()
	if err != nil {
		log.Fatal(err)
	}

	return ip
}
