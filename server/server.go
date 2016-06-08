/*
The package jsonrpcserver implement the jsonprotocol v2.1

You just have to call RunServer to create a tcp and websocket server.
The 3 channels are of type chan jsonrpcmessage.RoutedMessage -> var myChan = make(chan jsonrpcmessage.RoutedMessage)

(C) Meurice Christophe 2015
*/
package jsonrpcserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/think-free/jsonrpc/common/clientsocket"
	"github.com/think-free/jsonrpc/common/messagereader"
	"github.com/think-free/jsonrpc/common/messages"
	"golang.org/x/net/websocket"
)

/* Constants */
/* ************************************************************** */

const (
	hB_TIMEOUT = 100 // In seconds
)

/* The server */
/* ************************************************************** */

type Server struct {
	host                    string
	port                    string
	wsPort                  string
	hostname                string
	completeDomain          string
	state                   jsonrpcmessage.StateBody
	channels2m              chan jsonrpcmessage.RoutedMessage
	channelm2s              chan jsonrpcmessage.RoutedMessage
	channelinternalmessages chan jsonrpcmessage.RoutedMessage
	channelstate            chan jsonrpcmessage.StateBody
	channelClientState      chan ClientState
	mutex                   *sync.Mutex
	clientsByName           map[string]*jsonrpcclientsocket.ClientSocket
	clientsByValue          map[*jsonrpcclientsocket.ClientSocket]string
}

func New(host string, port string, wsPort string, hostname string, state jsonrpcmessage.StateBody,
	channels2m chan jsonrpcmessage.RoutedMessage,
	channelm2s chan jsonrpcmessage.RoutedMessage,
	channelinternalmessages chan jsonrpcmessage.RoutedMessage,
	channelstate chan jsonrpcmessage.StateBody,
	channelClientState chan ClientState) *Server {

	var completeD = ""
	if state.Domain != "" {
		completeD = hostname + "." + state.Domain
	} else {
		completeD = hostname
	}

	return &Server{
		host:                    host,
		port:                    port,
		wsPort:                  wsPort,
		hostname:                hostname,
		completeDomain:          completeD,
		state:                   state,
		channels2m:              channels2m,
		channelm2s:              channelm2s,
		channelinternalmessages: channelinternalmessages,
		channelstate:            channelstate,
		channelClientState:      channelClientState,
		mutex:                   &sync.Mutex{},
		clientsByName:           make(map[string]*jsonrpcclientsocket.ClientSocket),
		clientsByValue:          make(map[*jsonrpcclientsocket.ClientSocket]string),
	}
}

/* Public */
/* ************************************************************** */

/* Run a jsonrpc server
   Params :
   - host : interface on which to listen
   - port : tcp port to listen
   - wsPort : websocket port to listen
   - domain : domain for this server (see protocol for more details)
   - channels2m : this channel is used to send message to the application
   - channelm2s : this channel is used to send message from the application
   - channelinternalmessages : this channel is used to send internal messages received here to the application
   - channelstate : this channel is used to send the current state to the server
*/
func (server *Server) Run() {

	// Handle messages coming from the channel

	go server.runChannelHandler()

	// Listen for incoming tcp connections.

	l, err := net.Listen("tcp", server.host+":"+server.port)
	if err != nil {
		log.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	// Close the listener when the application closes.

	defer l.Close()

	// Websocket

	go func() {

		log.Println("Websocket server '" + server.hostname + "' is listening on " + server.host + ":" + server.wsPort)

		s := &http.Server{
			Addr: server.host + ":" + server.wsPort,
			Handler: websocket.Server{Handler: func(ws *websocket.Conn) {

				client := jsonrpcclientsocket.NewClientSocket(nil, ws, "ws")
				server.runNetHandler(client)
			}},
		}
		err = s.ListenAndServe()
		if err != nil {
			panic("Web socket server error: " + err.Error())
		}
	}()

	// TCP

	log.Println("Tcp server '" + server.hostname + "' is listening on " + server.host + ":" + server.port)

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		client := jsonrpcclientsocket.NewClientSocket(conn, nil, "tcp")

		// Handle connections in a new goroutine.
		go server.runNetHandler(client)
	}
}

/* Private */
/* ************************************************************** */

// Handle message from channel (route them to the client)
func (server *Server) runChannelHandler() {

	for {

		select {

		case mes := <-server.channelm2s:
			//log.Println("Message received from channel in server :", server.hostname, " type :", mes.Type, " dst : ", mes.Dst)
			server.parseRoutedMessage(&mes)

		case state := <-server.channelstate:
			log.Println("State changed in server :", server.hostname)
			server.processStateChange(&state)
		}
	}
}

// Handles incoming requests from network
func (server *Server) runNetHandler(client *jsonrpcclientsocket.ClientSocket) {

	log.Println("New", client.ClientType, "client connected to", server.hostname)

	hbtimer := time.NewTimer(time.Second * hB_TIMEOUT)

	go func() {
		<-hbtimer.C
		log.Println("Client heartbeat not received")
		client.Close()

		return
	}()

	messageReceivedChan := make(chan []byte)
	stopChan := make(chan bool)

	go jsonrpcreader.ReadJsonPacket(&messageReceivedChan, &stopChan, client)

	for {
		select {

		case mes := <-messageReceivedChan:
			go server.parseJson(hbtimer, client, mes)

		case <-stopChan:
			server.mutex.Lock()
			if server.clientsByValue[client] != "" {

				name := server.clientsByValue[client]
				server.channelClientState <- ClientState{name, false}

				log.Println("Removing client from client list : ", name)

				delete(server.clientsByName, name)
				delete(server.clientsByValue, client)
			}
			server.mutex.Unlock()
			hbtimer.Reset(time.Nanosecond)
			runtime.Goexit()

			return
		}
	}
}

// Json parser for messages
func (server *Server) parseJson(hbtimer *time.Timer, client *jsonrpcclientsocket.ClientSocket, message []byte) {

	var body json.RawMessage
	env := jsonrpcmessage.Message{
		Body: &body,
	}
	if err := json.Unmarshal(message, &env); err != nil {
		log.Println("Error parsing json from client")
		log.Println(err)
		client.Close()

		server.mutex.Lock()
		if server.clientsByValue[client] != "" {

			server.channelClientState <- ClientState{server.clientsByValue[client], false}
		}
		server.mutex.Unlock()
	}

	switch env.Type {

	// Handling heartbeat
	case "hb":

		var hb string
		if err := json.Unmarshal(body, &hb); err != nil {

			log.Println(err)
		}

		hbtimer.Reset(time.Second * hB_TIMEOUT)
		client.Write([]byte("{\"type\" : \"hbAck\", \"body\" : \"" + hb + "\"}"))

	// Handling hello
	case "hello":

		var hello jsonrpcmessage.HelloBody
		if err := json.Unmarshal(body, &hello); err != nil {

			log.Println(err)
		}

		server.mutex.Lock()

		if server.clientsByName[hello.Name] != nil {

			log.Println("WARNING ! Client '" + hello.Name + "' already send hello") //, skipping")
		}

		server.clientsByName[hello.Name] = client
		server.clientsByValue[client] = hello.Name

		tld := strconv.FormatBool(server.state.Tld)
		logged := strconv.FormatBool(server.state.Logged)
		domain := hello.Name + "." + server.completeDomain

		hostn := server.hostname

		server.mutex.Unlock()

		client.Write([]byte("{\"type\" : \"state\", \"body\" : {\"tld\": " + tld + ", \"logged\" : " + logged + ", \"domain\" : \"" + domain + "\" , \"ssid\" : \"not implemented\"}}"))

		log.Println("Client '" + hello.Name + "' (" + client.ClientType + ") in server '" + hostn + "' allowed to communicate (" + domain + ")")

		server.channelClientState <- ClientState{hello.Name, true}

		// TODO : Handle loggin

	// Handling routing and internal messages
	default:

		if server.clientsByValue[client] != "" {

			var mes jsonrpcmessage.RoutedMessage
			if err := json.Unmarshal(message, &mes); err != nil {

				log.Println("Internal message received for unknown type : ", env.Type)
				// TODO : Handle internal messages (externaly, send to channel ?)
			}

			server.parseRoutedMessage(&mes)

		} else {

			log.Println("Client forgot to send a hello message")
			// TODO : Send notification to the client

			time.Sleep(10 * time.Millisecond)
			client.Close()
		}
	}
}

// Routed json message (from socket or from channel)
func (server *Server) parseRoutedMessage(mes *jsonrpcmessage.RoutedMessage) {

	//log.Println("Routed message received of type :'" + mes.Type + "' -> routing from : " + mes.Src + " to : " + mes.Dst)

	if mes.Dst == server.completeDomain || mes.Dst == "" {

		//log.Println("This is an internal message of type", mes.Type)

		if server.channelinternalmessages != nil {

			server.channelinternalmessages <- *mes
		} else {

			log.Println("Internal message received in " + server.completeDomain + "but handler is not defined")
		}

	} else if strings.HasSuffix(mes.Dst, server.completeDomain) {

		// Internal routing to other host on this server

		// Getting the client name from the complete dst domain
		dstSlice := strings.Split(strings.TrimSuffix(mes.Dst, "."+server.completeDomain), ".")
		dst := dstSlice[len(dstSlice)-1]

		server.mutex.Lock()

		if dst == "*" {

			for destinationClientName, destinationClient := range server.clientsByName {

				mes.Dst = destinationClientName + "." + server.completeDomain
				//log.Println("Routing message to internal client : ", destinationClientName)
				json, _ := json.Marshal(mes)
				destinationClient.Write(json)
			}

		} else {

			if destinationClient, ok := server.clientsByName[dst]; ok {

				//log.Println("Routing message to internal client : ", dst)

				// Sending to this host
				json, _ := json.Marshal(mes)

				//log.Println(string(json))

				destinationClient.Write(json)
			} else {

				log.Println("Can't route message to internal client, client doesn't exists : " + dst)
			}
		}

		server.mutex.Unlock()

	} else {

		domainSlice := strings.Split(server.completeDomain, ".")

		if strings.HasSuffix(mes.Dst, domainSlice[len(domainSlice)-1]) {

			//log.Println("It's for an other server on this domain")
			mes := jsonrpcmessage.NewRoutedMessage(mes.Type, mes.Body, mes.Src, mes.Dst)
			server.channels2m <- *mes

		} else {

			log.Println("Unknow dst domain, aborting")
		}
	}
}

// Process the server state change

func (server *Server) processStateChange(state *jsonrpcmessage.StateBody) {

	server.mutex.Lock()

	server.state.Domain = state.Domain
	server.state.Logged = state.Logged
	server.state.Tld = state.Tld

	if server.state.Domain != "" {
		server.completeDomain = server.hostname + "." + server.state.Domain
	} else {
		server.completeDomain = server.hostname
	}

	// Send the state to all clients

	for name, client := range server.clientsByName {

		log.Println("Sending new state to client", name)

		tld := strconv.FormatBool(server.state.Tld)
		logged := strconv.FormatBool(server.state.Logged)
		domain := name + "." + server.completeDomain

		client.Write([]byte("{\"type\" : \"state\", \"body\" : {\"tld\": " + tld + ", \"logged\" : " + logged + ", \"domain\" : \"" + domain + "\" , \"ssid\" : \"not implemented\" }}"))
	}

	server.mutex.Unlock()
}
