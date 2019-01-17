package kademlia_go

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/zeromq/gyre"
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

	err = self.g.SetHeader("riaps@"+self.pNode.IP, "%s", "microgrid_test")
	if err != nil {
		self.log.Error(err)
	}

	err = self.g.SetVerbose()
	if err != nil {
		self.log.Error(err)
	}
	self.g.SetInterface("eth0")
	if err != nil {
		self.log.Error(err)
	}

	err = self.g.Start()
	if err != nil {
		return errors.New(fmt.Sprintf("Gyre.Start error: %s", err))
	}
	self.log.Debug("Gyre Node Started!")

	err = self.g.Join("MICROGRID_GROUP")
	if err != nil {
		return errors.New(fmt.Sprintf("Gyre.Join error: %s", err))
	}

	self.log.Debug("Gyre Node listening...")

	//Loop:
	for {
		select {
		case e := <-self.g.Events():
			self.log.Debugf("Gyre event type: %s", e.Type())
			switch e.Type() {
			case gyre.EventJoin:
				self.log.Debugf("GYRE EVENT JOIN: %s", e.Addr())
				//self.pNode.table.cmds <- cmd{cmdAdd, make(chan reply),}
			case gyre.EventLeave:
				self.log.Debugf("GYRE EVENT LEAVE: %s", e.Addr())
				//attempt remove node from routing table
			case gyre.EventEnter:
				self.log.Debugf("GYRE EVENT ENTER: %s", e.Addr())
			case gyre.EventExit:
				self.log.Debugf("GYRE EVENT EXIT: %s", e.Addr())
			}
		case <-self.done:
			if err != nil {
				self.log.Error(err)
			}
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
	self.log.Debug("Gyre Node stopping...")
	return nil
}
