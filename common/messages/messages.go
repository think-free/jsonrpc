package jsonrpcmessage

import (
	"encoding/json"
	"errors"
)

/* Base message */
/* ************************************************************** */

type Message struct {
	Type string      `json:"type"`
	Body interface{} `json:"body"`
}

/* Routed message */
/* ************************************************************** */

type RoutingInfo struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type RoutedMessage struct {
	Message
	RoutingInfo
}

func NewRoutedMessage(mesType string, body interface{}, src string, dst string) *RoutedMessage {

	return &RoutedMessage{
		Message{
			Type: mesType,
			Body: body,
		},
		RoutingInfo{
			Src: src,
			Dst: dst,
		},
	}
}

/* Hello body */
/* ************************************************************** */

type HelloBody struct {
	Name    string `json:"name"`
	Ssid    string `json:"ssid"`
	Version string `json:"version"`
}

/* State body */
/* ************************************************************** */

type StateBody struct {
	Domain string `json:"domain"`
	Tld    bool   `json:"tld"`
	Logged bool   `json:"logged"`
	Ssid   string `json:"ssid"`
}

/* Rpc */
/* ************************************************************** */

type RpcMessage struct {
	Type string `json:"type"`
	RoutingInfo
	Body RpcBody `json:"body"`
}

type RpcBody struct {
	Module string      `json:"module"`
	Fct    string      `json:"fct"`
	Params interface{} `json:"params"`
}

func GerRpcBodyFromMesBody(mes interface{}) (*RpcBody, error) {

	body, ok := mes.(map[string]interface{})

	if !ok {

		js, _ := json.Marshal(body)

		return nil, errors.New("Body error in rpc received : " + string(js))
	}

	return &RpcBody{
		Module: string(body["module"].(string)),
		Fct:    string(body["fct"].(string)),
		Params: body["params"],
	}, nil
}
