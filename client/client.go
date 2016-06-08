/*
The package jsonrpclient implement the jsonprotocol v2.1

You just have to call RunServer to create a tcp and websocket server.
The 3 channels are of type chan jsonrpcmessage.RoutedMessage -> var myChan = make(chan jsonrpcmessage.RoutedMessage)

(C) Meurice Christophe 2015
*/
package jsonrpclient

import (
	"encoding/json"
<<<<<<< HEAD
	"github.com/think-free/jsonrpc/common/clientsocket"
	"github.com/think-free/jsonrpc/common/messagereader"
	"github.com/think-free/jsonrpc/common/messages"
	"log"
	"net"
	"time"
=======
	"log"
	"net"
	"time"

	"github.com/think-free/jsonrpc/common/clientsocket"
	"github.com/think-free/jsonrpc/common/messagereader"
	"github.com/think-free/jsonrpc/common/messages"
>>>>>>> master
)

/* Constants */
/* ************************************************************** */

const (
	hB_TIMEOUT        = 2 // In seconds
<<<<<<< HEAD
	con_TIMEOUT       = 5
	protocolVersion   = "2.1.0"
	reconnect_TIMEOUT = 5
=======
	con_TIMEOUT       = 15
	protocolVersion   = "2.1.0"
	reconnect_TIMEOUT = 15
>>>>>>> master
)

/* The client */
/* ************************************************************** */

type Client struct {
	socket                  *jsonrpcclientsocket.ClientSocket
	hostname                string
	ip                      string
	port                    string
	state                   *jsonrpcmessage.StateBody
	sendChannel             chan []byte
	stateChannel            chan *jsonrpcmessage.StateBody
	rpcMessageChannel       chan *jsonrpcmessage.RpcMessage
	routedMessageChannel    chan *jsonrpcmessage.RoutedMessage
	nonroutedMessageChannel chan *jsonrpcmessage.Message
}

func New(hostname, ip, port string,
	sendChannel chan []byte,
	stateChannel chan *jsonrpcmessage.StateBody,
	rpcMessageChannel chan *jsonrpcmessage.RpcMessage,
	routedMessageChannel chan *jsonrpcmessage.RoutedMessage,
	nonroutedMessageChannel chan *jsonrpcmessage.Message) *Client {

	return &Client{
		socket:                  nil,
		hostname:                hostname,
		ip:                      ip,
		port:                    port,
		state:                   &jsonrpcmessage.StateBody{Domain: "", Tld: false, Logged: false, Ssid: ""},
		sendChannel:             sendChannel,
		stateChannel:            stateChannel,
		rpcMessageChannel:       rpcMessageChannel,
		routedMessageChannel:    routedMessageChannel,
		nonroutedMessageChannel: nonroutedMessageChannel,
	}
}

/* Public */
/* ************************************************************** */

/* Run a jsonrpc client
   Params :
   - hostname : the name of the client
   - ip : the ip of the server
   - port : the port of the server
*/
func (client *Client) Run() {

	for {
		// Connecting to server
		serverAddr, err := net.ResolveTCPAddr("tcp", client.ip+":"+client.port)
		if err != nil {

			log.Println("Can't resolve tcp address :" + client.ip + ":" + client.port)
			log.Println(err)
			time.Sleep(reconnect_TIMEOUT * time.Second)
			continue
		}
		c, err := net.DialTCP("tcp", nil, serverAddr)
		if err != nil {

			log.Println("Error connecting to server")
			log.Println(err)
			time.Sleep(reconnect_TIMEOUT * time.Second)
			continue
		}

		// Creating client and sending hello packet
		client.socket = jsonrpcclientsocket.NewClientSocket(c, nil, "tcp")
		client.socket.Write([]byte("{\"type\" : \"hello\", \"body\" : {\"name\" : \"" + client.hostname + "\", \"ssid\" : \"\", \"version\" : \"" + protocolVersion + "\"}}"))

		// Starting heartbeat manager and connection timeout
		go client.runHeartbeatManager()
		contimer := time.NewTimer(time.Second * con_TIMEOUT)

		// Reading from socket
		messageReceivedChan := make(chan []byte)
		stopChan := make(chan bool)
		go jsonrpcreader.ReadJsonPacket(&messageReceivedChan, &stopChan, client.socket)

		// Processing message from channels
		for {

			shouldstop := false
			select {

			// Message received from server
			case mes := <-messageReceivedChan:
				contimer.Reset(time.Second * con_TIMEOUT)
				go client.parseJson(mes)

			// Connection timer
			case <-contimer.C:
				client.socket.Close()
				log.Println("Closing client")
<<<<<<< HEAD
				return
=======
				break
>>>>>>> master

			// Send message to the server
			case mes := <-client.sendChannel:
				client.socket.Write(mes)

			// Client disconnected
			case <-stopChan:

				shouldstop = true
				break
			}

			if shouldstop {
				break
			}
		}
		time.Sleep(reconnect_TIMEOUT * time.Second)
	}
}

/* Private */
/* ************************************************************** */

// The heartbeater timer
func (client *Client) runHeartbeatManager() {

	hbtimer := time.NewTimer(time.Second * hB_TIMEOUT)

	go func() {
		for {
			<-hbtimer.C
			client.socket.Write([]byte("{\"type\": \"hb\", \"body\": \"Go!\"}"))
			hbtimer.Reset(time.Second * hB_TIMEOUT)
		}
	}()
}

func (client *Client) parseJson(message []byte) {

	var body json.RawMessage
	env := jsonrpcmessage.Message{
		Body: &body,
	}
	if err := json.Unmarshal(message, &env); err != nil {
		log.Println("Error parsing json from server")
		log.Println(err)
	}

	switch env.Type {

	// Handling heartbeat
	case "hbAck":
		return
	case "state":
		var state jsonrpcmessage.StateBody
		if err := json.Unmarshal(body, &state); err != nil {
			log.Fatal(err)
		}
		client.state = &state
		client.stateChannel <- &state
	default:
		var mes jsonrpcmessage.RoutedMessage
		if err := json.Unmarshal(message, &mes); err != nil {

			log.Println("Internal message received for unknown type : ", env.Type)
			// TODO : Handle internal messages (externaly, send to channel ?)
		}

		if mes.Dst == client.state.Domain || client.state.Domain == "" {

			if mes.Type == "rpc" {

				var rpc jsonrpcmessage.RpcMessage
				if err := json.Unmarshal(message, &rpc); err != nil {

					log.Println("Invalid rpc received :", message)
					return
				}
				client.rpcMessageChannel <- &rpc
			} else {

				client.routedMessageChannel <- &mes
			}
		} else {

			log.Println("Unknow destination for message", mes.Dst+" - ", client.state.Domain, string(message))
		}
	}
}
