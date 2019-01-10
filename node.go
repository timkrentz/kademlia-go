package kademlia_go

type Node struct {
	IP   string
	Port uint16
}

func NewNode(ip string, port uint16) *Node {
	return &Node{
		IP:   ip,
		Port: port,
	}
}

func (self Node) Ping(ip string, port uint16) int {
	return 0
}
