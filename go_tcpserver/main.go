package main

import (
	"fmt"
	"net"
	"os"
	"sync"
)

const (
	connHost = "192.168.1.7"
	connPort = "6666"
	connType = "tcp4"
)

type user struct {
	conn net.Conn
}

func notify(user *user, b []byte) {
	_, e := user.conn.Write(b)
	if e != nil {
		fmt.Println(e.Error())
	}
}

type users struct {
	users []*user
	mtx   *sync.Mutex
}

func (users *users) connections() []byte {
	b := make([]byte, 4*len(users.users))
	for _, e := range users.users {
		var temp []byte = net.ParseIP(e.conn.RemoteAddr().String())
		b = append(b, temp...)
	}
	return b
}

type fun func(*user, []byte)
type upointer []*user

func (u upointer) each(f fun, b []byte) {
	for _, e := range u {
		f(e, b)
	}
}

func (u upointer) find(user *user) int {
	for i, e := range u {
		if e.conn == user.conn {
			return i
		}
	}
	return -1
}

func (users *users) addUser(user *user) {
	users.mtx.Lock()
	users.users = append(users.users, user)
	b := users.connections()
	(upointer(users.users)).each(notify, b)
	users.mtx.Unlock()
}

func (users *users) removeUser(user *user) {
	users.mtx.Lock()
	i := (upointer(users.users)).find(user)
	users.users = append(users.users[:i], users.users[i+1:]...)
	b := users.connections()
	(upointer(users.users)).each(notify, b)
	users.mtx.Unlock()
}

func (users *users) shutdown() {
	users.mtx.Lock()
	b := []byte("shutdown")
	(upointer(users.users)).each(notify, b)
	users.mtx.Unlock()
}

func main() {
	lsn, err := net.Listen(connType, connHost+":"+connPort)
	if err != nil {
		fmt.Println("Error while listening:", err.Error())
		os.Exit(-1)
	}

	defer lsn.Close()

	users := new(users)
	users.mtx = new(sync.Mutex)

	defer users.shutdown()

	fmt.Println("Listening on ", connHost, ":", connPort)

	for {

		conn, err := lsn.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			os.Exit(-1)
		}

		newUser := user{
			conn: conn,
		}

		users.addUser(&newUser)
	}
}
