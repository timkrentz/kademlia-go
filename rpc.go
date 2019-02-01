package kademlia_go

import (
	"fmt"
	"golang.org/x/sys/unix"
	"log"
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
	rpcNode.log.Printf("Serving RPC server on endpoint %s", ip)

	syncChan <- rpcNode

	err = http.Serve(rpcNode.listener, nil)
	if err != nil && !strings.Contains(fmt.Sprint(err), "use of closed network connection") {
		rpcNode.log.Print("HTTP ended on: ", err)
	}
	rpcNode.log.Println("RPC Server closed")
}

func StopRPCServer(kademliaNode *Node) {
	kademliaNode.RpcServer.listener.Close()
}

type MsgPing struct {
	Sender NodeID
}

func (self *RPCNode) RPCPing(arg MsgPing, reply *MsgPing) error {
	//self.log.Print("Received PING from ", arg.Sender.IP)
	*reply = MsgPing{NodeID{self.pNode.IP, self.pNode.ID}}
	return nil
}

type MsgRTTransfer struct {
	Sender   NodeID
	NodeList []NodeID
}

func (self *RPCNode) RTTransfer(arg MsgRTTransfer, reply *MsgPing) error {
	//self.log.Print("Received RT from ",arg.Sender.IP)
	RTandSender := []NodeID{arg.Sender}
	RTandSender = append(RTandSender, arg.NodeList...)
	self.pNode.table.cmds <- MsgCmd{cmdAdd, nil, RTandSender}
	*reply = MsgPing{NodeID{self.pNode.IP, self.pNode.ID}}
	return nil
}

type MsgStore struct {
	Sender  NodeID
	Key     string
	Payload Measurement
}

type Measurement struct {
	Value     int
	Timestamp unix.Timespec
}

// 15us sans debug statements
func (self *RPCNode) Store(arg MsgStore, reply *MsgPing) error {
	//self.log.Print("Received Store from ",arg.Sender.IP)
	//self.log.Print("Attempting to store ",arg)
	self.pNode.storageMx.Lock()
	self.pNode.storage[arg.Key] = append(self.pNode.storage[arg.Key], arg.Payload)
	self.pNode.storageMx.Unlock()
	//self.log.Print("Stored",arg.Payload.Value)

	*reply = MsgPing{NodeID{self.pNode.IP, self.pNode.ID}}
	return nil
}

type MsgReadReq struct {
	Sender NodeID
	IDs    []string
}

type MsgReadRep struct {
	Sender NodeID
	Value  []Measurement
}

func (self *RPCNode) Read(arg MsgReadReq, reply *MsgReadRep) error {
	*reply = MsgReadRep{}
	self.pNode.storageMx.Lock()
	for _, id := range arg.IDs {
		reply.Value = append(reply.Value, self.pNode.storage[id]...)
	}
	self.pNode.storageMx.Unlock()

	reply.Sender = NodeID{self.pNode.IP, self.pNode.ID}
	self.log.Print("RPC Read:", arg.IDs)

	return nil
}
