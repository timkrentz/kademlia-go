package kademlia_go

import (
	"bytes"
	"container/list"
	"fmt"
)

const (
	cmdAdd   = "ADD NODE"
	cmdRead  = "READ TABLE"
	cmdJoin  = "GYRE JOIN"
	cmdDump  = "DUMP TABLE"
	cmdRoute = "ROUTE KEY"
	//cmdUUID          = "UUID"
	//cmdName          = "NAME"
	//cmdSetName       = "SET NAME"
	//cmdSetHeader     = "SET HEADER"
	//cmdSetVerbose    = "SET VERBOSE"
	//cmdSetPort       = "SET PORT"
	//cmdSetInterval   = "SET INTERVAL"
	//cmdSetIface      = "SET INTERFACE"
	//cmdSetEndpoint   = "SET ENDPOINT"
	//cmdGossipBind    = "GOSSIP BIND"
	//cmdGossipPort    = "GOSSIP PORT"
	//cmdGossipConnect = "GOSSIP CONNECT"
	//cmdStart         = "START"
	//cmdStop          = "STOP"
	//cmdWhisper       = "WHISPER"
	//cmdShout         = "SHOUT"
	//cmdJoin          = "JOIN"
	//cmdLeave         = "LEAVE"
	//cmdDump          = "DUMP"
	//cmdTerm          = "$TERM"
	//
	//// Deprecated
	//cmdAddr    = "ADDR"
	//cmdHeader  = "HEADER"
	//cmdHeaders = "HEADERS"
)

type NodeID struct {
	IP string
	ID []byte
}

func XOR(a, b []byte) []byte {
	var res []byte = nil
	for i, v := range a {
		res = append(res, v^b[i])
	}
	return res
}

// True if A < B, False otherwise, assumes little-endian
func Less(a, b []byte) bool {
	var cmp byte
	for idx, _ := range a {
		for i := 7; i >= 0; i-- {
			cmp = 0x01 << uint(i)
			if (a[idx]&cmp) == 0 && (b[idx]&cmp) != 0 {
				return true
			} else if (a[idx]&cmp) != 0 && (b[idx]&cmp) == 0 {
				return false
			}
		}
	}
	return false
}

//Return index of first '1' bit in byte slice
func msb(x []byte) int {
	var cmp byte
	var j, i int
Outer:
	for j = 0; j < 20; j++ {
		for i = 7; i >= 0; i-- {
			cmp = 0x01 << uint(i)
			if (x[j] & cmp) != 0x00 {
				break Outer
			}
		}
	}
	retVal := (j*8 + (7 - i))
	if retVal > 159 || retVal < 0 {
		fmt.Errorf("MSB ERROR first '1' bit is: %d", retVal)
	}
	return retVal
}

func (self *NodeID) String() string {
	return self.IP + fmt.Sprintf(" %x", self.ID)
}

type MsgCmd struct {
	cmd     string
	rChan   chan MsgReply
	payload interface{}
}

type MsgReply struct {
	cmd     string
	payload interface{}
	err     error
}

// Always keep the 6 closest to you, and 1 for each 4 bit diffs above that\
//Decent for cluster size 24
type Table struct {
	//peers   [][]NodeID
	peers   *list.List
	cmds    chan MsgCmd
	replies chan MsgReply
	owner   *Node
}

func NewTable() *Table {
	//ptr := make([][]NodeID, 5)
	//for i := 0; i < 4; i++ {ptr[i] = make([]NodeID, 1)}
	//ptr[4] = make([]NodeID,6)
	return &Table{
		//peers:   ptr,
		peers:   list.New(),
		cmds:    make(chan MsgCmd),
		replies: make(chan MsgReply),
		owner:   nil,
	}
}

func (self *Table) Start(owner *Node) {
	self.owner = owner
	self.peers.PushFront(NodeID{self.owner.IP, self.owner.ID})
	for {
		select {
		case c := <-self.cmds:
			//self.owner.log.Print("Table received command:", c.cmd)
			switch c.cmd {
			case cmdAdd:
				if p, ok := c.payload.([]NodeID); ok {
					for _, newP := range p {
						self.insert(newP)
					}
				} else if p, ok := c.payload.(NodeID); ok {
					self.insert(p)
				}
			case cmdRead:
				c.rChan <- MsgReply{cmdRead, self.peers, nil}

			case cmdDump:
				var idSlice []NodeID
				for e := self.peers.Front(); e != nil; e = e.Next() {
					idSlice = append(idSlice, e.Value.(NodeID))
				}
				c.rChan <- MsgReply{cmdDump, idSlice, nil}
			case cmdRoute:
				//Find the 4 IDs most similar to given key
				distance := XOR(c.payload.([]byte), self.owner.ID)

				var rList []NodeID = nil
				sortList := list.New()
				sortList.PushFront(self.peers.Front().Value.(NodeID))
				for e := self.peers.Front().Next(); e != nil; e = e.Next() {
					insVal := e.Value.(NodeID)
				Inner:
					for f := sortList.Front(); f != nil; f = f.Next() {
						currVal := f.Value.(NodeID)
						//If distance from key to insVal < distance from key to currVal
						if Less(XOR(distance, XOR(self.owner.ID, insVal.ID)), XOR(distance, XOR(self.owner.ID, currVal.ID))) {
							if f != sortList.Back() {
								continue
							} else {
								sortList.InsertAfter(insVal, f)
								break Inner
							}
						} else {
							sortList.InsertBefore(insVal, f)
							break Inner
						}
					}
				}
				numAdded := 0
				for e := sortList.Back(); numAdded < replicationFactor && e != nil; e = e.Prev() {
					rList = append(rList, e.Value.(NodeID))
					numAdded++
				}
				c.rChan <- MsgReply{cmdRoute, rList, nil}
			}
		}
	}
}

//insert ID into table
// ret: 1 - success, 0 not inserted
func (self *Table) insert(id NodeID) int {
	//self.owner.log.Debug("Attempting to insert:",id)
	distance := XOR(id.ID, self.owner.ID)
	for e := self.peers.Front(); e != nil; e = e.Next() {
		curr := e.Value.(NodeID)
		if bytes.Equal(curr.ID, id.ID) {
			return 0
		} //ignore if id is already present
		if Less(distance, XOR(curr.ID, self.owner.ID)) {
			if e.Next() != nil {
				continue
			} else {
				self.peers.InsertAfter(id, e)
				return 1
			}
		} else {
			self.peers.InsertBefore(id, e)
			return 1
		}
	}
	//Here if peers.Front == nil (list empty)
	//Push to front
	self.peers.PushFront(id)
	return 1
}
