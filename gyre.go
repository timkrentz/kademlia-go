package kademlia_go

import (
	"errors"
	"fmt"
	"github.com/zeromq/gyre"
	"log"
)

type GyreNode struct {
	g     *gyre.Gyre
	done  chan bool
	log   *log.Logger
	pNode *Node
}

func NewGyreNode(owner *Node) *GyreNode {
	gptr, err := gyre.New()
	if err != nil {
		_ = fmt.Errorf(fmt.Sprintf("Gyre.New error: %s", err))
	}
	return &GyreNode{
		g:     gptr,
		done:  make(chan bool),
		log:   nil,
		pNode: owner,
	}
}

func (self *GyreNode) Start(logger *log.Logger) error {
	var err error

	self.log = logger

	err = self.g.SetHeader("microgrid_test", "%s", self.pNode.IP)
	if err != nil {
		self.log.Fatal(err)
	}

	//err = self.g.SetVerbose()
	//if err != nil {
	//	self.log.Fatal(err)
	//}

	self.g.SetInterface("eth0")
	if err != nil {
		self.log.Fatal(err)
	}

	err = self.g.Start()
	if err != nil {
		return errors.New(fmt.Sprintf("Gyre.Start error: %s", err))
	}
	self.log.Print("Gyre Node Started!")

	err = self.g.Join("MICROGRID_GROUP")
	if err != nil {
		return errors.New(fmt.Sprintf("Gyre.Join error: %s", err))
	}

	self.log.Print("Gyre Node listening...")

	//Loop:
	for {
		select {
		case e := <-self.g.Events():
			switch e.Type() {
			case gyre.EventJoin:
				//self.log.Debugf("GYRE EVENT JOIN: %s", e.Headers())
			case gyre.EventLeave:
				//self.log.Debugf("GYRE EVENT LEAVE: %s", e.Headers())
				//attempt remove node from routing table
			case gyre.EventEnter:
				//self.log.Debugf("GYRE EVENT ENTER: %s", e.Headers())
				payload, ok := e.Header("microgrid_test")
				if ok {
					self.pNode.internalCmds <- MsgCmd{cmdJoin, nil, payload}
				}
			case gyre.EventExit:
				//self.log.Debugf("GYRE EVENT EXIT: %s", e.Headers())
			}
		case <-self.done:
			err = self.g.Stop()
			self.done <- true
			if err != nil {
				return errors.New(fmt.Sprintf("Gyre.Stop error: %s", err))
			}
			//break Loop
		}
	}
	return nil
}

func (self *GyreNode) Stop() error {
	self.done <- true
	<-self.done
	self.log.Print("Gyre Node stopping...")
	return nil
}
