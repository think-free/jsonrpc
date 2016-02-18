package jsonrpcclientsocket

import (
	"errors"
	"golang.org/x/net/websocket"
	"net"
)

/*
 * This is a client socket wich encapsulate
 * - websocket
 * - tcp socket
 ************************************************************** */

type ClientSocket struct {
	TcpClient       net.Conn
	WebsocketClient *websocket.Conn
	ClientType      string
}

// Create a new socket
func NewClientSocket(tcpClient net.Conn, websocketClient *websocket.Conn, clientType string) *ClientSocket {

	return &ClientSocket{
		TcpClient:       tcpClient,
		WebsocketClient: websocketClient,
		ClientType:      clientType,
	}
}

// Read a message from the socket
func (client *ClientSocket) Read(message *[]byte) (n int, err error) {

	if client.ClientType == "tcp" {

		return client.TcpClient.Read(*message)

	} else if client.ClientType == "ws" {

		var v string
		err = websocket.Message.Receive(client.WebsocketClient, &v)

		*message = []byte(v)
		return len(*message), err

	} else {

		return 0, errors.New("Unknown client type for read")
	}
}

// Write a message to the socket
func (client *ClientSocket) Write(message []byte) (err error) {

	var pack = append([]byte(":::0:::"), message...)
	pack = append(pack, []byte(":::1:::")...)

	if client.ClientType == "tcp" {

		if client.TcpClient == nil {

			return errors.New("Tcp client is null")
		}
		client.TcpClient.Write(pack)
	} else if client.ClientType == "ws" {

		if client.WebsocketClient == nil {

			return errors.New("Websocket client is null")
		}
		client.WebsocketClient.Write(pack)
	} else {

		return errors.New("Unknow client type, can't send the message")
	}

	return nil
}

// Close the socket
func (client *ClientSocket) Close() (err error) {

	if client.ClientType == "tcp" {

		return client.TcpClient.Close()
	} else if client.ClientType == "ws" {

		return client.WebsocketClient.Close()
	} else {

		return errors.New("Unknown client type for close")
	}
}
