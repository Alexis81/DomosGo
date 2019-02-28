package main

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type JMessage struct {
	S    string   `json:"s"`
	M    string   `json:"m"`
	V    Temp     `json:"v"`
	Tags []string `json:"tags"`
}

type Temp struct {
	Temperature string `json:"temperature"`
}

func PushLiveObject(temperatureEauString string) {

	connOpts := mqtt.NewClientOptions().AddBroker("tcp://" + urlLiveObjects).SetClientID("urn:lo:nsid:Sonde_Piscine_Eau:SondeEau")
	connOpts.SetKeepAlive(50 * time.Second)
	connOpts.SetUsername("json+device")
	connOpts.SetPassword(cleApiLiveObjects)
	connOpts.SetDefaultPublishHandler(f)
	connOpts.SetPingTimeout(10 * time.Second)

	clientMqtt := mqtt.NewClient(connOpts)
	if token := clientMqtt.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	jmessage := JMessage{
		S: "Temperature_eau",
		M: "temperatureDevice_eau",
		V: Temp{
			Temperature: temperatureEauString,
		},
		Tags: []string{"Poolhouse"},
	}

	message, err := json.Marshal(jmessage)

	if err != nil {
		fmt.Println(err)
	}

	//message := "{\"s\" : \"Temperature_eau\",\"v\" : {\"temperature\" : " + temperatureEauString + "},\"tags\" : [\"Poolhouse\"]}"

	//fmt.Println(message)

	clientMqtt.Publish("dev/data", 0, false, message)

	clientMqtt.Disconnect(1)
}

//blood on the dance floor
