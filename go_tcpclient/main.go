package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

const (
	connHost = "192.168.1.9"
	connPort = "6666"
	connType = "tcp4"
)

type peers struct {
	peers []*net.Conn
	mtx   *sync.Mutex
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
var next_port int = 667

func updateScreen(msg string) {
	output += string(msg)
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
	fmt.Printf("%s\n", output)
}

// This is not thread safe. Must synchronize outside of this function.
func (peers *peers) getPeers(b []byte) {
	n := 0
	for n < len(b) {
		ip := net.IP(b[n*4 : n*4+4])
		if !peers.hasPeer(ip) {
			// Clients should listen on port 666 for new connections which negotiate port to actually connect on.
			conn, e := net.Dial(connType, ip.String()+":666")
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
						conn, e = net.Dial(connType, ip.String()+string(d))
						peers.peers = append(peers.peers, &conn)
						go handlePeer(&conn, peers)
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
	peers.mtx = new(sync.Mutex)
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
	// Doesn't need synchronization bc only 1 thread atm
	peers.getPeers(buf)

	// Handle user input
	go handleUser(peers)
	// Handle new peers connecting
	go handleNewPeers(peers)
	// Listen for updates to the peer list
	for {
		n = 0
		e = nil
		for n <= 0 {
			n, e = conn.Read(buf)
			if e != nil {
				fmt.Println(e.Error())
				os.Exit(-1)
			}
		}
		if string(buf) == "\x00\x00\x00\x00" {
			fmt.Println("Server Disconnected...")
			os.Exit(-2)
		}
		peers.mtx.Lock()
		peers.getPeers(buf)
		peers.mtx.Unlock()
	}
}

// Reads input from stdin and parses commands (disconnect, share, etc.)
func handleUser(peers *peers) {
}

// Listens for updates to the peer list from the server.
func handleNewPeers(peers *peers) {
	lsn, err := net.Listen(connType, connHost+":666")
	if err != nil {
		fmt.Println("Error while listening:", err.Error())
		os.Exit(-1)
	}
	defer lsn.Close()

	for {
		// Wait for people to connect
		conn, err := lsn.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			os.Exit(-1)
		}

		// Somebody connected
		newLsn, _ := net.Listen(connType, connHost+":"+string(next_port))
		_, e := conn.Write([]byte(strconv.Itoa(next_port)))
		next_port++
		if e != nil {
			fmt.Println(e.Error())
			break
		}

		newConn, _ := newLsn.Accept()
		peers.mtx.Lock()
		peers.peers = append(peers.peers, &newConn)
		peers.mtx.Unlock()
	}
}

/*
* 	Listens to a peer for sharing requests.
*
*	`peers` is not currently used but I'm not sure if it will be
*	needed in the future so I'm just going to keep it for the moment.
 */
func handlePeer(conn *net.Conn, peers *peers) {

}
