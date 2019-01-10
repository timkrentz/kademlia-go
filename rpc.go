package kademlia_go

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type RPCNode struct {
	pNode *Node
}

func NewRPCNode(self *Node) *RPCNode {
	return &RPCNode{self}
}

func StartRPCServer(ip string, port uint16, kademliaNode *Node, syncChan chan bool) {
	var err error
	rpcNode := NewRPCNode(kademliaNode)
	err = rpc.Register(rpcNode)
	if err != nil {
		log.Fatal("Format of service rpcNode isn't correct. ", err)
	}
	rpc.HandleHTTP()
	listener, e := net.Listen("tcp", ip+":"+fmt.Sprint(port))
	if e != nil {
		log.Fatal("Listen error: ", e)
	}
	log.Printf("Serving RPC server on port %d", port)

	syncChan <- true

	err = http.Serve(listener, nil)
	if err != nil {
		log.Fatal("Error serving: ", err)
	}
	return
}

type MsgPing struct {
	IP   string
	Port uint16
}

func (self *RPCNode) RPCPing(arg MsgPing, reply *MsgPing) error {
	*reply = MsgPing{self.pNode.IP, self.pNode.Port}
	return nil
}
