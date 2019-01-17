package kademlia_go

const (
	cmdAdd  = "ADD NODE"
	cmdRead = "READ TABLE"
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
	IP   string
	Port uint16
	ID   []byte
}

type cmd struct {
	cmd     string
	rChan   chan<- *reply
	payload interface{}
}

type reply struct {
	cmd     string
	payload interface{}
	err     error
}

type Table struct {
	peers   []NodeID
	cmds    chan cmd
	replies chan interface{}
}

func NewTable() *Table {
	return &Table{
		peers:   make([]NodeID, 1),
		cmds:    make(chan cmd),
		replies: make(chan interface{}),
	}
}

func (self *Table) Start() {
	for {
		select {
		case c := <-self.cmds:
			switch c.cmd {
			case cmdAdd:
				self.peers = append(self.peers, c.payload.(NodeID))

			case cmdRead:
				c.rChan <- &reply{cmdRead, self.peers, nil}
			}
		}
	}
}
