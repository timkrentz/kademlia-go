package kademlia_go

import (
	"crypto/sha1"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	IP        string
	Port      uint16
	GyreNode  *GyreNode
	RpcServer *RPCNode
	ID        []byte
	log       *log.Logger
}

func NewNode(ip string, port uint16) *Node {
	return &Node{
		IP:        ip,
		Port:      port,
		GyreNode:  nil,
		RpcServer: nil,
		ID:        make([]byte, 20),
		log:       nil,
	}
}

func (self *Node) Start() error {
	if len(self.IP) == 0 || self.Port == 0 {
		return errors.New(fmt.Sprintf("Attempted to start Node with empty IP or Port - ip: %s\tport:%d", self.IP, self.Port))
	}

	self.log = log.New()
	self.log.SetLevel(log.DebugLevel)

	h := sha1.New()
	h.Write([]byte(self.IP + ":" + fmt.Sprint(self.Port)))
	self.ID = h.Sum(nil)

	self.GyreNode = NewGyreNode()
	go self.GyreNode.Start(self.log)

	syncChan := make(chan *RPCNode)
	go StartRPCServer(self.IP, self.Port, self, syncChan)
	//wait until RPC Server is actually started
	self.RpcServer = <-syncChan
	self.log.Debug("Kademlia node started on IP:", self.IP, " Port:", self.Port)
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

func (self *Node) Ping(ip string, port uint16) int {
	return 0
}
