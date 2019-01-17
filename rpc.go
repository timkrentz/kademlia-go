package kademlia_go

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/rpc"
	"strings"
)

// This is not a true RPC object, unfortunately, that has Start/Stop methods (as I think it should).
// Start/Stop methods violate the net/rpc interface, so the RPC methods simply receive a reference
// to the kademlia node so that they may access local data.
type RPCNode struct {
	pNode    *Node
	listener net.Listener
	log      *log.Logger
}

func NewRPCNode(self *Node) *RPCNode {
	return &RPCNode{
		pNode:    self,
		listener: nil,
		log:      nil,
	}
}

func StartRPCServer(ip string, kademliaNode *Node, syncChan chan<- *RPCNode) {
	var err error
	rpcNode := NewRPCNode(kademliaNode)
	rpcNode.log = kademliaNode.log
	err = rpc.Register(rpcNode)
	if err != nil {
		log.Fatal("Format of service rpcNode isn't correct. ", err)
	}
	rpc.HandleHTTP()
	rpcNode.listener, err = net.Listen("tcp", ip)
	if err != nil {
		log.Fatal("Listen error: ", err)
	}
	rpcNode.log.Debugf("Serving RPC server on endpoint %s", ip)

	syncChan <- rpcNode

	err = http.Serve(rpcNode.listener, nil)
	if err != nil && !strings.Contains(fmt.Sprint(err), "use of closed network connection") {
		log.Warn("HTTP ended on: ", err)
	}
	rpcNode.log.Debug("RPC Server closed")
}

func StopRPCServer(kademliaNode *Node) {
	kademliaNode.RpcServer.listener.Close()
}

type MsgPing struct {
	IP string
	ID []byte
}

func (self *RPCNode) RPCPing(arg MsgPing, reply *MsgPing) error {
	*reply = MsgPing{self.pNode.IP, self.pNode.ID}
	log.Debug("Received PING from ", arg.IP)
	return nil
}
