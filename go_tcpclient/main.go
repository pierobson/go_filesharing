package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const (
	connHost = "192.168.1.7"
	connPort = "6666"
	connType = "tcp4"
)

func GetIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		fmt.Println("Couldn't get IP:", err.Error())
		os.Exit(-1)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

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
var nextPort int = 6665
var ip string

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// This is not thread safe. Must synchronize outside of this function.
func (peers *peers) getPeers(b []byte) {
	n := 0
	for n < len(b) {
		ip := net.IP(b[n*4 : n*4+4])
		if !peers.hasPeer(ip) {
			// Clients should listen on port 666 for new connections which negotiate port to actually connect on.
			conn, e := net.Dial(connType, ip.String()+":6667")
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

func send(conn *net.Conn, msg string) {
	fmt.Println("Sharing", msg, "with", (*conn).RemoteAddr().String())
	(*conn).Write([]byte(msg))
}

type peerPointer []*net.Conn
type fun func(*net.Conn, string)

func (peers peerPointer) each(f fun, msg string) {
	for _, e := range peers {
		f(e, msg)
	}
}

func (peers *peers) share(path string) {
	(peerPointer(peers.peers)).each(send, path)
}

func main() {
	clearScreen()

	ip = GetIP().String()

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
		if string(buf) == "shutdown" {
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
	reader := bufio.NewReader(os.Stdin)
	for {
		cmd, e := reader.ReadString('\n')
		if e != nil {
			fmt.Println("Failed to read message:", e.Error())
		}
		cmd = cmd[:len(cmd)-1]
		if len(cmd) > 0 {
			args := strings.Split(cmd, " ")
			switch args[0] {
			case "exit":
				fmt.Println("Disconnecting...")
				os.Exit(1)
			case "clear":
				clearScreen()
			case "share":
				if len(args) == 2 && args[1] != "" {
					peers.mtx.Lock()
					fmt.Println(peers.peers)
					peers.share(args[1])
					peers.mtx.Unlock()
				} else {
					fmt.Println("Usage: `share 'path'")
				}
			default:
				fmt.Println("Command does not exist:", cmd)
			}
		}
	}
}

// Listens for updates to the peer list from the server.
func handleNewPeers(peers *peers) {
	fmt.Println("Attempting to listen on port 6667")
	lsn, err := net.Listen(connType, connHost+":6667")
	for err != nil {
		fmt.Println("Attempting to listen on port", nextPort)
		lsn, err = net.Listen(connType, connHost+":"+strconv.Itoa(nextPort))
		nextPort--
	}
	defer lsn.Close()
	fmt.Println("Listening...")

	for {
		// Wait for people to connect
		conn, err := lsn.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			os.Exit(-1)
		}

		fmt.Println("Here")

		// Somebody connected
		newLsn, e := net.Listen(connType, ip+":"+string(nextPort))
		nextPort--
		for e != nil {
			newLsn, e = net.Listen(connType, ip+":"+string(nextPort))
			nextPort--
		}
		_, e = conn.Write([]byte(strconv.Itoa(nextPort)))
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
	for {
		n := 0
		buf := make([]byte, 10)
		for n <= 0 {
			n, _ = (*conn).Read(buf)
		}
		fmt.Println(string(buf))
	}
}
