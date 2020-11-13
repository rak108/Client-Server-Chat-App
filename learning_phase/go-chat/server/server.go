/**
This file contains the definition for the server struct and the functions it implements.

There are two main classes of goroutines that exist in the server process:
- The first class is a single goroutine that listens for connection requests from clients.
  Whenever it receives a connection request, it creates a new goroutine that would come under
  the second class.
- The second class contains goroutines that handles the already established connections between
  the clients. These goroutines allow the server and the client to send messages to each other.
  Whenever a message is received from a particular client, the recipient's address is determined
  using the map containing the current connections. The message is then sent to the relevant client
  through the goroutine corresponding to that client.
*/

package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rak108/Distributed-DNS/learning_phase/go-chat/shared"
)

// This struct characterizes the server process
type server struct {
	ServerHost      string // ip:port or :port, by default, it is "0.0.0.0:4545"
	Password        string // The server password specified by the user
	mutex           sync.RWMutex
	ClientConnected map[string]net.Conn
}

func Server(pass string, address string) *server {

	/*
		An instance of the 'server' struct is created, initialized with given
		or the default data(if the user hasn't specified the data).
	*/
	var password string
	var serveraddress string
	if pass == "-" {
		password = "1234"
	} else {
		password = pass
	}
	if address == "-" {
		serveraddress = "4545"
	} else {
		serveraddress = address
	}
	return &server{
		ClientConnected: make(map[string]net.Conn),
		Password:        password,
		ServerHost:      serveraddress,
	}

}

func (ser *server) unicasting(fromusername string, tousername string, message string) string {
	checker := true
	ser.mutex.RLock()
	for name, connection := range ser.ClientConnected {
		if tousername == name {
			message = string("Unicast received from " + fromusername + ": " + message)
			_, err := connection.Write([]byte(message + "\n"))
			if err != nil {
				//shared.CheckError(err)
				fmt.Println("Unable to send message from " + fromusername + "to " + tousername)
				checker = false
				break
			}
		}
	}
	ser.mutex.RUnlock()
	if checker {
		return string("Message sent!")
	} else {
		return string("Unable to send message to " + tousername)
	}

}

func (ser *server) broadcasting(fromusername string, message string) string {
	checker := true
	ser.mutex.RLock()
	for name, connection := range ser.ClientConnected {
		if fromusername != name {
			printmessage := string("Broadcast received from " + fromusername + ": " + message)
			_, err := connection.Write([]byte(printmessage + "\n"))
			if err != nil {
				//shared.CheckError(err)
				fmt.Println("\n Unable to broadcast message from " + fromusername + "to " + name)
				checker = false
				continue
			}
		}
	}
	ser.mutex.RUnlock()
	if checker {
		return string("Message sent!")
	} else {
		return string("Unable to broadcast to all.")
	}
}

func (ser *server) listenForMessages(ctx context.Context, conn net.Conn, username string, term chan bool) {

	/**
	Method parameter description:
	1. ctx - cancellable context
	2. conn - represents the socket connection to the client
	2. username - client username
	3. term - write to this channel on receiving termination request from client
	*/

	/**
	Spawned by handleClient(). Listens for messages sent by the given client, and appropriately unicasts/broadcast/
	prints error message, etc.
	*/
	timeoutDuration := 300 * time.Second
	for {
		messagebytes := make([]byte, 256)
		conn.SetReadDeadline(time.Now().Add(timeoutDuration))
		_, err := io.ReadFull(conn, messagebytes)
		if err != nil && err == io.EOF {
			//shared.CheckError(err)
			fmt.Println("\nLost connection to " + username + ", EOF received")
			term <- true
			return
		}
		message := string(messagebytes[:])
		splitmessage := strings.Split(message, "~")
		mtype := splitmessage[0]
		switch mtype {
		case "1":
			msg := ser.unicasting(username, splitmessage[1], splitmessage[2])
			conn.Write([]byte(shared.Paddingto256(msg)))
		case "2":
			msg := ser.broadcasting(username, splitmessage[1])
			conn.Write([]byte(shared.Paddingto256(msg)))
		case "3":
			{
				term <- true
				return
			}
		default:
		}

	}
}

func (ser *server) handleClient(ctx context.Context, conn net.Conn) {

	/**
	Spawned by Run() when a client connection is received. Performs authentication and username checking, responds
	appropriately and after that, uses listenForMessages to handle incoming messages. Should handle cancellation of context
	and messages written to the term channel in the above function
	*/
	term_chan := make(chan bool)
	var password string
	var username string
	messagebytes := make([]byte, 256)
	_, err := io.ReadFull(conn, messagebytes)
	if err != nil {
		if err != io.EOF {
			shared.CheckError(err)
		} else {
			fmt.Println("\nConnection termiated on client end before authentication")
		}
	}
	message := string(messagebytes)
	message = strings.Trim(message, "\r\n")
	splitmessage := strings.Split(message, "~")

	if splitmessage[0] == "0" {
		password = splitmessage[2]
		username = splitmessage[1]
		if password == ser.Password {
			ser.mutex.RLock()
			_, check := ser.ClientConnected[username]
			ser.mutex.RUnlock()
			if check == true {
				message := "0~Password correct, but Username already taken in this server. Please change username and try again."
				conn.Write([]byte(shared.Paddingto256(message)))
				fmt.Println("\nConnection unsuccessful as user with same username was already connected to server")
				return
			} else {
				message := "---" + username + " Authenticated---"
				conn.Write([]byte(shared.Paddingto256(message)))
				fmt.Println("User authenticated: " + username)
				formatmessage := string("Messaging format after authentication:\n\nPrivate messaging other users on server:-\n\t\tUnicast~receiverusername~message\n\nBroadcast to others on server:-\n\t\tBroadcast~message\n\nTerminate from server:\n\t\tTerminate\n\n")
				conn.Write([]byte(shared.Paddingto256(formatmessage)))
				ser.mutex.Lock()
				ser.ClientConnected[username] = conn
				ser.mutex.Unlock()
				go ser.listenForMessages(ctx, conn, username, term_chan)
				select {
				case <-ctx.Done():
					{
						message := "0~You have been disconnected! Server disconnected and exited program."
						conn.Write([]byte(message + "\n"))
						conn.Close()
						return
					}

				case check := <-term_chan:
					if check == true {
						message = ("0~User " + username + " terminated. Bye!")
						conn.Write([]byte(shared.Paddingto256(message)))
						fmt.Println("\n" + username + " terminated from server.")
						ser.mutex.Lock()
						delete(ser.ClientConnected, username)
						ser.mutex.Unlock()
						conn.Close()
						return
					}

				}
			}

		} else {
			message := "0~Wrong Password, Connection rejected. Try again.\n\n"
			conn.Write([]byte(shared.Paddingto256(message)))
			fmt.Println("Password Incorrect from Received Connection, Connection rejected.")
			return
		}
	}

}

func (ser *server) listenForConnections(ctx context.Context, newConn chan net.Conn, listener *net.TCPListener) {

	// Called from Run()
	// Accept incoming connections from clients and write it to the newConn channel
	for {
		conn, err := listener.Accept()
		if err != nil {
			shared.CheckError(err)
			return
		}
		newConn <- conn
		fmt.Println("\n Connection received. Waiting for authentication...")
	}

}

func (ser *server) Run(ctx context.Context) {

	// Bind a socket to a port and start listening on it. Use listenForConnections
	// and listen for new connections written to the channel, and appropriately spawn
	// handleClient. Also handle cancellation of context sent from main.
	ser.ServerHost = ":" + ser.ServerHost
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ser.ServerHost)
	if err != nil {
		shared.CheckError(err)
		return
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		shared.CheckError(err)
		return
	}
	fmt.Println("\nServer at " + ser.ServerHost + " started!\nServer password: " + ser.Password + "\n")
	defer listener.Close()
	newConn := make(chan net.Conn)
	go ser.listenForConnections(ctx, newConn, listener)
	for {
		select {
		case <-ctx.Done():
			return
		case conn := <-newConn:
			go ser.handleClient(ctx, conn)
		}
	}

}
