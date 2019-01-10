package kademlia_go

import (
	"fmt"
	"log"
	"net/rpc"
	"testing"
)

func TestNode(t *testing.T) {
	ip, port := GetOutboundIP()
	myNode := NewNode(ip, port)
	t.Log("Node A is:", myNode)

	syncChan := make(chan bool)
	go StartRPCServer(ip, port, myNode, syncChan)
	//wait until RPC Server is actually started
	<-syncChan

	var reply MsgPing
	msg := MsgPing{ip, port}

	t.Log("Reply:", reply)
	t.Log("Ping Msg:", msg)

	client, err := rpc.DialHTTP("tcp", ip+":"+fmt.Sprint(port))
	if err != nil {
		log.Fatal("Connection error: ", err)
	}

	client.Call("RPCNode.RPCPing", msg, &reply)

	t.Log("Reply:", reply)
	t.Log("Ping Msg:", msg)

}
