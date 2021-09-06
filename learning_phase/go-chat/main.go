package main

/**

Protocol for sending/receiving messages:
- Each message will be of length 256 bytes.
- There will be 6 types of messages:

1. authenticate~<client_username>~<server_password>~\n  --> sent from client to server
2. authenticated~\n  --> sent from server to client
3. unicast~<username>~<message>~\n  --> from client to server, unicast message
4. broadcast~<message>~\n  --> from client to server, broadcast message
5. message~<username>~<type>~<message>~\n  --> from server to client(s). Type indicates unicast/broadcast.
6. terminate\n  --> from server to client, for graceful shutdown

Possible extensions:
dockerize the application
use gorilla websockets instead of raw sockets, get a web-based frontend
use RPCs for communication instead of raw sockets
persist messages
write tests
add security
deploy to heroku

*/

import (
	"context"
	"fmt"

	"github.com/rak108/Client-Server-Chat-App/learning_phase/go-chat/client"
	"github.com/rak108/Client-Server-Chat-App/learning_phase/go-chat/server"
)

func main() {

	/**
	- Create a cancellable context to be passed to client/server functions
	- Listen for appropriate termination signals (refer to https://gobyexample.com/signals) in a separate goroutine
	- The above goroutine should call the context's cancel function on receiving a termination signal,
	  and wait for the client/server function called below to complete its execution (how will this goroutine know
	  when the client/server function called has completed execution?). Then it can finally terminate the program.
	- The main function should then decide whether the execution will be in client mode or in server mode
	  (by reading the command line parameters). On deciding, it should create & initialize a new struct of
	  client/server type (which will be defined in client.go and server.go), and call the respective Run() function
	*/
	var choice int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fmt.Println("\n If you are server adming setting server up, press 1.\n If you are client wishing to connect, enter 2. ")
	fmt.Scanln(&choice)
	if choice == 1 {
		var port string
		var pwd string
		fmt.Println("\n Enter port you want the server to listen on (For default 4545, enter ''): ")
		fmt.Scanln(&port)
		fmt.Println("\n Enter server password (For default '1234', enter '' ): ")
		fmt.Scanln(&pwd)
		ser := server.Server(pwd, port)
		go ser.Run(ctx)
		for {
		}
	} else if choice == 2 {
		fmt.Println("Launching client...\nWelcome! Enter your custom username (max 8 characters), or for a random one, enter '': ")
		var username string
		fmt.Scanln(&username)
		cli := client.Client("", username)
		main_term_chan := make(chan bool)
		go cli.Run(ctx, main_term_chan)
		select {
		case <-ctx.Done():
			break
		case <-main_term_chan:
			break
		}
	}

}
