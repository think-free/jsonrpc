package jsonrpcreader

import (
	"bytes"
	"github.com/think-free/jsonrpc/common/clientsocket"
	"log"
	"regexp"
)

func ReadJsonPacket(messageReceivedChan *chan []byte, stopChan *chan bool, client *jsonrpcclientsocket.ClientSocket) {

	var clientBuffer []byte
	reg := regexp.MustCompile(":::0:::(.*?):::1:::")

	for {

		// Make a buffer to hold incoming data.
		messageReceived := make([]byte, 1024)

		// Read the incoming connection into the buffer.
		receivedBytes, err := client.Read(&messageReceived)

		if err != nil {
			if err.Error() == "EOF" {
				log.Println("Client disconnected")
			} else {
				log.Println("Error reading : ", err.Error(), receivedBytes)
			}

			*stopChan <- true

			return
		}

		clientBuffer = append(clientBuffer, messageReceived[:receivedBytes]...)

		for {
			regexMatch := reg.FindSubmatch(clientBuffer)

			if regexMatch == nil {
				break
			}

			messagePositionInBuffer := bytes.IndexAny(clientBuffer, string(regexMatch[0]))
			messageLength := len(regexMatch[0])

			clientBuffer = clientBuffer[messagePositionInBuffer+messageLength:]

			*messageReceivedChan <- regexMatch[1]
		}
	}
}
