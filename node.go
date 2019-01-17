package kademlia_go

import (
	"crypto/sha1"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/rpc"
)

type Node struct {
	IP        string
	GyreNode  *GyreNode
	RpcServer *RPCNode
	ID        []byte
	log       *log.Logger
	table     *Table
}

func NewNode(ip string) *Node {
	return &Node{
		IP:        ip,
		GyreNode:  nil,
		RpcServer: nil,
		ID:        make([]byte, 20),
		log:       nil,
	}
}

func (self *Node) Start() error {
	if len(self.IP) == 0 {
		return errors.New(fmt.Sprintf("Attempted to start Node with empty IP: %s", self.IP))
	}

	self.log = log.New()
	self.log.SetLevel(log.DebugLevel)

	h := sha1.New()
	h.Write([]byte(self.IP))
	self.ID = h.Sum(nil)

	self.table = NewTable()
	go self.table.Start()

	self.GyreNode = NewGyreNode(self)
	go self.GyreNode.Start(self.log)

	syncChan := make(chan *RPCNode)
	go StartRPCServer(self.IP, self, syncChan)
	//wait until RPC Server is actually started
	self.RpcServer = <-syncChan
	self.log.Debug("Kademlia node started on IP:", self.IP)
	return nil
}

func (self *Node) Stop() int {
	StopRPCServer(self)
	err := self.GyreNode.Stop()
	if err != nil {
		self.log.Errorf("Gyre.Stop failed with error: %s", err)
	}
	return 0
}

func (self *Node) ping(ip string, port uint16) int {
	var reply MsgPing
	msg := MsgPing{self.IP, self.ID}

	//TODO: Catch timeout error, use to remove from routing table
	client, err := rpc.DialHTTP("tcp", ip+":"+fmt.Sprint(port))
	if err != nil {
		log.Fatal("Connection error: ", err)
	}

	client.Call("RPCNode.RPCPing", msg, &reply)
	return 0
}
