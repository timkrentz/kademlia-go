package main

import (
	"fmt"
	k "github.com/timkrentz/kademlia-go"
	"time"
)

func main() {
	ip, port := k.GetOutboundIP()
	node := k.NewNode(ip, port)
	err := node.Start()
	if err != nil {
		fmt.Errorf("Node start failure:", err)
	}
	defer node.Stop()

	time.Sleep(10 * time.Second)
	fmt.Println("Experiment Done!")
}
