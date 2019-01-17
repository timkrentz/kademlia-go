package kademlia_go

import (
	"bytes"
	"fmt"
	"log"
	"net/rpc"
	"testing"
)

func TestNode(t *testing.T) {
	ip := GetOutboundIP()
	myNode := NewNode(ip)
	t.Log("Node A is:", myNode)

	err := myNode.Start()
	if err != nil {
		t.Fatal("Node start failure:", err)
	}
	defer myNode.Stop()

	t.Log("Node ID:", fmt.Sprintf("%x", myNode.ID))

	var reply MsgPing
	msg := MsgPing{ip, myNode.ID}

	t.Log("Reply:", reply)
	t.Log("Ping Msg:", msg)

	client, err := rpc.DialHTTP("tcp", ip)
	if err != nil {
		log.Fatal("Connection error: ", err)
	}

	client.Call("RPCNode.RPCPing", msg, &reply)

	if bytes.Compare(reply.ID, msg.ID) != 0 {
		t.Fatal("PING reply doesn't match request - \n\nREQ ", msg, "\n\nREP", reply)
	}

	t.Log("Reply:", reply)
	t.Log("Ping Msg:", msg)
}
