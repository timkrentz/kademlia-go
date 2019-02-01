package kademlia_go

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	timeDelta         = 10
	replicationFactor = 1
)

type Node struct {
	IP            string
	GyreNode      *GyreNode
	RpcServer     *RPCNode
	ID            []byte
	log           *log.Logger
	table         *Table
	internalCmds  chan MsgCmd
	storage       map[string][]Measurement
	storageMx     *sync.Mutex
	routeCache    map[string][]NodeID
	clientCache   map[string]*rpc.Client
	clientCacheMx map[string]*sync.Mutex
}

func NewNode(ip string) *Node {
	return &Node{
		IP:            ip,
		GyreNode:      nil,
		RpcServer:     nil,
		ID:            make([]byte, 20),
		log:           nil,
		internalCmds:  make(chan MsgCmd),
		storage:       make(map[string][]Measurement),
		storageMx:     &sync.Mutex{},
		routeCache:    make(map[string][]NodeID),
		clientCache:   make(map[string]*rpc.Client),
		clientCacheMx: make(map[string]*sync.Mutex),
	}
}

func (self *Node) Start() error {
	if len(self.IP) == 0 {
		return errors.New(fmt.Sprintf("Attempted to start Node with empty IP: %s", self.IP))
	}

	//log.Print() takes approx 0.8 ms
	self.log = log.New(os.Stdout, self.IP, log.Ltime|log.Lmicroseconds|log.Lshortfile)

	h := sha1.New()
	h.Write([]byte(self.IP))
	self.ID = h.Sum(nil)

	self.table = NewTable()
	go self.table.Start(self)

	syncChan := make(chan *RPCNode)
	go StartRPCServer(self.IP, self, syncChan)
	//wait until RPC Server is actually started
	self.RpcServer = <-syncChan

	self.GyreNode = NewGyreNode(self)
	go self.GyreNode.Start(self.log)

	self.log.Print("Kademlia node started on IP:", self.IP)

	//TODO: The following should be put in some kind of actor class

	for {
		select {
		case intMsg := <-self.internalCmds:
			//self.log.Printf("Received internal command: %s",intMsg.cmd)
			switch intMsg.cmd {
			case cmdJoin:
				self.sendRoutingTable(intMsg.payload.(string))
			}
		}
	}

	self.log.Print("Kademlia node stopping...")

	return nil
}

func (self *Node) Stop() int {
	for k, v := range self.clientCache {
		err := v.Close()
		if err != nil {
			self.log.Printf("Failed to close client to %s: %s", k, err)
		}
	}
	StopRPCServer(self)
	err := self.GyreNode.Stop()
	if err != nil {
		self.log.Fatalf("Gyre.Stop failed with error: %s", err)
	}
	return 0
}

func (self *Node) Print() {
	repChan := make(chan MsgReply)
	self.table.cmds <- MsgCmd{cmdDump, repChan, nil}
	repMsg := <-repChan
	idSlice := repMsg.payload.([]NodeID)
	var ips []string
	for _, id := range idSlice {
		ips = append(ips, id.IP+fmt.Sprintf(" %X", XOR(id.ID, self.ID)))
	}
	self.log.Print("Routing Table Size:", strconv.Itoa(len(ips)))
	self.log.Print("Routing Table:", ips, "\n")

	self.log.Printf("Storage keys: ")
	for k, _ := range self.storage {
		self.log.Printf("%s ", k)
	}
}

func (self *Node) ping(ip string) int {
	var reply MsgPing
	msg := MsgPing{NodeID{self.IP, self.ID}}

	//TODO: Catch timeout error, use to remove from routing table
	client, err := rpc.DialHTTP("tcp", ip)
	defer client.Close()
	if err != nil {
		self.log.Fatal("Ping Connection error: ", err)
	}

	//self.log.Debug("Sending ping to ",ip)

	err = client.Call("RPCNode.RPCPing", msg, &reply)
	if err != nil {
		self.log.Fatal("client.Call error: ", err)
	}
	return 0
}

func (self *Node) sendRoutingTable(endpoint string) error {
	var reply MsgPing
	msg := MsgRTTransfer{}
	msg.Sender = NodeID{self.IP, self.ID}

	repChan := make(chan MsgReply)
	self.table.cmds <- MsgCmd{cmdDump, repChan, nil}
	repMsg := <-repChan

	if repMsg.cmd != cmdDump {
		self.log.Fatal("RT Send reply message of wrong type!")
	}

	msg.NodeList = repMsg.payload.([]NodeID)

	//self.log.Print("Sending routing table to ",endpoint)

	client, err := rpc.DialHTTP("tcp", endpoint)
	defer client.Close()
	if err != nil {
		self.log.Fatal("SendRT Connection error: ", err)
	}
	err = client.Call("RPCNode.RTTransfer", msg, &reply)
	if err != nil {
		self.log.Fatal("client.Call error: ", err)
	}
	return nil
}

func (self *Node) route(key []byte) ([]NodeID, error) {
	//tic := time.Now()
	keyString := fmt.Sprintf("%x", key)

	var rList []NodeID = nil
	if self.routeCache[keyString] != nil {
		rList = append(rList, self.routeCache[keyString]...)
	} else {
		repChan := make(chan MsgReply)
		self.table.cmds <- MsgCmd{cmdRoute, repChan, key}
		repMsg := <-repChan

		tempList, ok := repMsg.payload.([]NodeID)
		if !ok {
			return tempList, errors.New("route(): Malformed Table Read")
		} else {
			self.routeCache[keyString] = append(self.routeCache[keyString], tempList...)
			rList = append(rList, tempList...)
		}
	}
	//self.log.Printf("Single route cmd took: %d",time.Since(tic).Nanoseconds())
	return rList, nil
}

func (self *Node) Get(key string, times ...unix.Timespec) ([]Measurement, error) {
	tic := time.Now()
	var startTime, endTime unix.Timespec
	var numRequests int

	if len(times) > 2 {
		return nil, errors.New("More than 2 timespecs passed to Get()")
	} else if len(times) > 1 {
		startTime = times[0]
		endTime = times[1]
		numRequests = int(endTime.Sec/timeDelta) - int(startTime.Sec/timeDelta) + 1
	} else {
		startTime = times[0]
		numRequests = 1
	}

	var idList = make([][]byte, numRequests, numRequests)
	for j := 0; j < numRequests; j++ {
		ts := startTime
		ts.Sec += int32(j * timeDelta)
		dstId := generateKey(key, ts)
		idList[j] = dstId
	}

	var retrievedMeasurements []Measurement = nil
	retDataChannel := make(chan []Measurement)

	for _, id := range idList {
		ownerNodeList, err := self.route(id)
		if err != nil {
			self.log.Fatal("Error when routing:", err)
		}

		currentNode := ownerNodeList[0]

		go func(n NodeID, id []byte, mChan chan []Measurement) {

			var repMsg MsgReadRep

			reqMsg := MsgReadReq{
				Sender: NodeID{self.IP, self.ID},
				IDs:    []string{fmt.Sprintf("%x", id)},
			}

			if self.clientCache[n.IP] == nil {
				self.clientCache[n.IP], err = rpc.DialHTTP("tcp", n.IP)
				if err != nil {
					self.log.Fatal("Connection error: ", err)
				}
			}

			if self.clientCacheMx[n.IP] == nil {
				self.clientCacheMx[n.IP] = &sync.Mutex{}
			}
			self.clientCacheMx[n.IP].Lock()

			err = self.clientCache[n.IP].Call("RPCNode.Read", reqMsg, &repMsg)
			if err != nil {
				self.log.Fatal("client.Call Read error: ", err)
			}

			self.clientCacheMx[n.IP].Unlock()

			mChan <- repMsg.Value

		}(currentNode, id, retDataChannel)
	}

	for i := 0; i < numRequests; i++ {
		retrievedMeasurements = append(retrievedMeasurements, <-retDataChannel...)
	}

	self.log.Printf("GET %d entries took %d nanoseconds\n", numRequests, time.Since(tic).Nanoseconds())

	return retrievedMeasurements, nil

	//for _, id := range idList {
	//	reqMsg.IDs = append(reqMsg.IDs, fmt.Sprintf("%x",id))
	//}
	//
	//
	//nodeList, err := self.route(idList[0])
	//if err != nil {
	//	self.log.Fatal("Error when routing:",err)
	//}
	//self.Print()
	////self.log.Print("Routed Node List:",nodeList)
	//
	//var repMsg MsgReadRep
	//
	////self.log.Print("Sending Read RPC to ",nodeList[0].IP)
	//
	//client, err := rpc.DialHTTP("tcp", nodeList[0].IP)
	//defer client.Close()
	//if err != nil {
	//	self.log.Fatal("Get Connection error: ", err)
	//}
	//err = client.Call("RPCNode.Read", reqMsg, &repMsg)
	//if err != nil {
	//	self.log.Fatal("client.Call error: ", err)
	//}

	//return repMsg.Value, nil
}

func (self *Node) Put(key string, measurement int, ts unix.Timespec) error {
	//toc := time.Now()
	//self.log.Printf("Sending put for key %s\n",key)

	dstId := generateKey(key, ts)

	nodeList, err := self.route(dstId)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	//self.log.Printf("Setup PUT RPC took: %d",time.Since(toc).Nanoseconds())
	//self.log.Print("Destination nodes are:",nodeList)

	for _, n := range nodeList {
		wg.Add(1)
		go func(n NodeID, id []byte, m int) {
			defer wg.Done()

			msg := MsgStore{
				Sender:  NodeID{self.IP, self.ID},
				Key:     fmt.Sprintf("%x", id),
				Payload: Measurement{measurement, ts},
			}
			var reply MsgPing

			if self.clientCache[n.IP] == nil {
				self.clientCache[n.IP], err = rpc.DialHTTP("tcp", n.IP)
				if err != nil {
					self.log.Fatal("Connection error: ", err)
				}
			}

			if self.clientCacheMx[n.IP] == nil {
				self.clientCacheMx[n.IP] = &sync.Mutex{}
			}
			self.clientCacheMx[n.IP].Lock()

			err = self.clientCache[n.IP].Call("RPCNode.Store", msg, &reply)
			if err != nil {
				self.log.Fatal("client.Call Store error: ", err)
			}

			self.clientCacheMx[n.IP].Unlock()

		}(n, dstId, measurement)
	}
	wg.Wait()
	return nil
}

func generateKey(value string, ts unix.Timespec) []byte {

	h := sha1.New()
	h.Write([]byte(value))
	hashedValue := h.Sum(nil)

	//Set modulo here to define system-wide time bucket
	var window = ts.Sec / timeDelta
	timeSlice := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeSlice, uint64(window))
	h.Write([]byte(timeSlice))
	hashedTime := h.Sum(nil)

	/*======================================================
	==================   KEY FIRST ID   ====================
	 =====================================================*/
	//retval := append(hashedValue[0:10],hashedTime[0:10]...)

	/*======================================================
		=================   QUANTA FIRST ID   ==================
	 	======================================================*/
	retval := append(hashedTime[0:10], hashedValue[0:10]...)

	return retval
}
