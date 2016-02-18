package jsontools

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/think-free/jsonrpc/common/messages"
)

func GenerateRpcMessage(sendChannel *chan []byte, module string, fct string, params interface{}, src string, dst string) {

	body := &jsonrpcmessage.RpcBody{Module: module, Fct: fct, Params: params}
	mes := jsonrpcmessage.NewRoutedMessage("rpc", body, src, dst)
	js, _ := json.Marshal(mes)

	*sendChannel <- js
}

func GetUrl(url string) []byte {

	response, err := http.Get(url)
	if err != nil {

		fmt.Printf("%s", err)

		return nil

	} else {

		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
		}

		return contents
	}
}
