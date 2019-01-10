package kademlia_go

import (
	"log"
	"net"
)

// GetOutboundIP returns an internet facing IP for the local machine and a random port
func GetOutboundIP() (string, uint16) {

	conn, err := net.Dial("udp", "8.8.8.8:0")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	localPort := conn.LocalAddr().(*net.UDPAddr).Port

	ip := localAddr.IP.String()
	port := uint16(localPort)

	return ip, port
}
