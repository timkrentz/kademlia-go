package kademlia_go

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/zeromq/gyre"
)

type GyreNode struct {
	g    *gyre.Gyre
	done chan bool
	log  *log.Logger
}

func NewGyreNode() *GyreNode {
	gptr, err := gyre.New()
	if err != nil {
		_ = fmt.Errorf(fmt.Sprintf("Gyre.New error: %s", err))
	}
	return &GyreNode{
		g:    gptr,
		done: make(chan bool),
		log:  nil,
	}
}

func (self *GyreNode) Start(logger *log.Logger) error {
	var err error

	self.log = logger

	err = self.g.Start()
	if err != nil {
		return errors.New(fmt.Sprintf("Gyre.Start error: %s", err))
	}

	err = self.g.Join("DISCOVERY_GROUP")
	if err != nil {
		return errors.New(fmt.Sprintf("Gyre.Join error: %s", err))
	}

	for {
		select {
		case e := <-self.g.Events():
			switch e.Type() {
			case gyre.EventJoin:
				log.Debug("GYRE EVENT JOIN: ", e.Sender())
			case gyre.EventLeave:
				log.Debug("GYRE EVENT LEAVE:", e.Sender())
				//attempt remove node from routing table
			}
		case <-self.done:
			err = self.g.Stop()
			self.done <- true
			if err != nil {
				return errors.New(fmt.Sprintf("Gyre.Stop error: %s", err))
			}
		}
	}
	return nil
}

func (self *GyreNode) Stop() error {
	self.done <- true
	<-self.done
	return nil
}
