package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
)

const (
	connHost = "192.168.1.9"
	connPort = "6666"
	connType = "tcp"
)

type peers struct {
	peers []*net.Conn
}

func (peers *peers) hasPeer(ip net.IP) bool {
	for _, e := range peers.peers {
		if (*e).RemoteAddr().String() == ip.String() {
			return true
		}
	}
	return false
}

var output string = ""
var next_port uint32 = 667

func updateScreen(msg string) {
	output += string(msg)
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
	fmt.Printf("%s\n", output)
}

func (peers *peers) getPeers(b []byte) {
	n := 0
	for n < len(b) {
		ip := net.IP(b[n*4 : n*4+4])
		if !peers.hasPeer(ip) {
			// Clients should listen on port 666 for new connections which negotiate port to actually connect on.
			conn, e := net.Dial("tcp4", ip.String()+":666")
			if e == nil {
				_, e = conn.Write([]byte(string(next_port)))
				if e == nil {
					var a int = 0
					var er bool = true
					d := make([]byte, 4)
					for a <= 0 && er {
						a, e = conn.Read(d)
						if e != nil {
							er = false
						} else {
							conn.Close()
							conn, e = net.Dial("tcp4", ip.String()+string(d))
							peers.peers = append(peers.peers, &conn)
						}
					}
				}
			}
		}
		n += 4
	}
}

func main() {
	updateScreen("")

	conn, err := net.Dial(connType, connHost+":"+connPort)
	if err != nil {
		fmt.Println("Error connecting:", err.Error())
		os.Exit(-1)
	}
	defer conn.Close()

	// Get the peer list
	peers := new(peers)
	n := 0
	var e error
	buf := make([]byte, 4)
	for n <= 0 {
		n, e = conn.Read(buf)
		if e != nil {
			fmt.Println(e.Error())
			os.Exit(-1)
		}
	}
	peers.getPeers(buf)

	// Handle user input
	go handleUser(peers)
	// Handle new peers connecting
	go handleNewPeers(peers)
	// Listen for updates to the peer list
	for {

	}
}

func handleUser(peers *peers) {
}

func handleNewPeers(peers *peers) {
}
