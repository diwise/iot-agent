package application

import (
	"testing"

	"github.com/matryer/is"
)

func TestChirpStack(t *testing.T) {
	is := is.New(t)
	for _, ue := range uplinkChirpStack {
		_, err := ChirpStack([]byte(ue))
		is.NoErr(err)
	}
}

func TestNetmore(t *testing.T) {
	is := is.New(t)

	for _, ue := range uplinkNetmore {
		_, err := Netmore([]byte(ue))
		is.NoErr(err)
	}
}

var uplinkChirpStack = []string{
	`{"applicationID":"21","applicationName":"Soraker","deviceName":"05343464","deviceProfileName":"Axioma_Universal_Codec","deviceProfileID":"d45461aa-e877-4c09-8b52-0b41e670359f","devEUI":"00070900005188e8","rxInfo":[{"gatewayID":"fcc23dfffe0b6b58","uplinkID":"f4b3f1df-9ca2-4a7c-a84c-ac522bddebb2","name":"SN-LGW-017","time":"2022-10-18T15:39:39.088645922Z","rssi":-113,"loRaSNR":-5.5,"location":{"latitude":62.504757783826896,"longitude":17.51152038574219,"altitude":0}}],"txInfo":{"frequency":868100000,"dr":3},"adr":true,"fCnt":4107,"fPort":100,"data":"AebITmMAAKoCAAAAAKAAIAAGQADwBPgABsAAUAAAAABAFEAEAAAEQAJQACAAB4AAUAAA","object":{"curDateTime":"2022-10-18 17:40:22","curVol":174592,"deltaVol":{"id1":0,"id10":3,"id11":5,"id12":0,"id13":0,"id14":81,"id15":68,"id16":0,"id17":4,"id18":9,"id19":5,"id2":0,"id20":8,"id21":7,"id22":2,"id23":5,"id3":10,"id4":8,"id5":6,"id6":1,"id7":79,"id8":62,"id9":6},"frameVersion":1,"statusCode":0},"tags":{"SerialNo":"05343464"}}`,
	`{"deviceName":"sn-elt-livboj-01","devEUI":"a81758fffe04d855","data":"Bw4ADQA=","object":{"present":false}}`,
	`{"deviceName":"sk-elt-temp-31","deviceProfileName":"Elsys_Codec","deviceProfileID":"f113c342-4048-4df5-8e83-a2642d66990d","devEUI":"a81758fffe04d824","data":"Bw5BDADk","object":{"externalTemperature":22.8,"vdd":3649},"tags":{"Location":"Norrhassel"}}`,
	`{"applicationID":"24","applicationName":"Air-Temperature","deviceName":"AIR-SENS-02","deviceProfileName":"Elsys_Codec","deviceProfileID":"f113c342-4048-4df5-8e83-a2642d66990d","devEUI":"a81758fffe0524f3","rxInfo":[{"gatewayID":"fcc23dfffe2ee936","uplinkID":"b761e459-84da-4d03-9f63-06a9653f4f1e","name":"SN-LGW-047","time":"2022-10-18T15:43:12.577940261Z","rssi":-118,"loRaSNR":-9,"location":{"latitude":62.36956091265246,"longitude":17.319844410529534,"altitude":0}}],"txInfo":{"frequency":867700000,"dr":4},"adr":true,"fCnt":92871,"fPort":5,"data":"AQAzAlQUAA9q7Q==","object":{"humidity":84,"pressure":1010.413,"temperature":5.1},"tags":{"Location":"Norraberget (Ryggis)"}}`,
	`{"applicationID":"22","applicationName":"Bergsaker","deviceName":"05343338","deviceProfileName":"Axioma_Universal_Codec","deviceProfileID":"d45461aa-e877-4c09-8b52-0b41e670359f","devEUI":"000709000051886a","rxInfo":[{"gatewayID":"7276ff0039040436","uplinkID":"4f3b871f-66d3-4b7f-8543-74d30a0e97a3","name":"SN-LGW-105","time":"2022-10-18T15:27:18.948345Z","rssi":-119,"loRaSNR":-0.5,"location":{"latitude":62.42065486301624,"longitude":17.201843261718754,"altitude":0}}],"txInfo":{"frequency":867500000,"dr":5},"adr":true,"fCnt":703,"fPort":100,"data":"Ac3FTmMACcUDAAAAAVABcAAQAAAAAAAABUAAgAA0AAKAAIAAFAAXgACQAFgAAwAAAAAA","object":{"curDateTime":"2022-10-18 17:27:9","curVol":247049,"deltaVol":{"id1":0,"id10":1,"id11":8,"id12":13,"id13":2,"id14":2,"id15":8,"id16":5,"id17":23,"id18":2,"id19":9,"id2":4,"id20":22,"id21":3,"id22":0,"id23":0,"id3":21,"id4":28,"id5":16,"id6":0,"id7":0,"id8":0,"id9":5},"frameVersion":1,"statusCode":0},"tags":{"SerialNo":"05343338"}}`,
}

var uplinkNetmore = []string{
	`[{"devEui":"70b3d554600002b6","sensorType":"cube02","timestamp":"2022-10-18T13:32:46.361551Z","payload":"b006b8000130094598000001c1a8000099000001f4a9000008416be864","spreadingFactor":"12","rssi":"-106","snr":"4.2","gatewayIdentifier":"126","fPort":"2"}]`,
	`[{"devEui":"70b3d5e75e000efe","sensorType":"other","timestamp":"2022-10-18T13:43:04.584377Z","payload":"110a00520000410c0000000000003db600000000","spreadingFactor":"12","rssi":"-97","snr":"0.5","gatewayIdentifier":"187","fPort":"125"}]`,
	`[{"devEui":"70b3d52c0001918b","sensorType":"strips_lora_ms_h","timestamp":"2022-10-18T11:50:04.138012Z","payload":"ffff0a01","spreadingFactor":"8","rssi":"-112","snr":"-3","gatewayIdentifier":"824","fPort":"1"}]`,
	`[{"devEui":"70b3d52c00018778","sensorType":"strips_lora_ms_wl","timestamp":"2022-10-18T11:57:33.587149Z","payload":"ffff01580200c80400c807001508001509000a010d000e00","spreadingFactor":"8","rssi":"-95","snr":"5.5","gatewayIdentifier":"187","fPort":"1"}]`,
	`[{"devEui":"70b3d580a0110608","sensorType":"tem_lab_14ns","timestamp":"2022-10-18T12:12:25.509216Z","payload":"01ef82359c1000c0","spreadingFactor":"11","rssi":"-112","snr":"-6.8","gatewayIdentifier":"187","fPort":"3","latitude":57.687844,"longitude":12.036078}]`,
	`[{"devEui":"00070900004c82af","sensorType":"qalcosonic_w1e","messageType":"payload","timestamp":"2022-10-18T12:17:03.166298Z","payload":"8e974e6300227a060050c24d63cb790600000000000000000000000000000016003300000000000000000000000e00","fCntUp":6647,"toa":null,"freq":867300000,"batteryLevel":"255","ack":false,"spreadingFactor":"8","rssi":"-108","snr":"3.5","gatewayIdentifier":"126","fPort":"100","tags":{"application":["1_kretsloppvatten_w1e_1"],"customer":["kretsloppvatten"],"deviceType":["w1e"],"facilityID":[],"municipality":[],"serial":["05014191"]},"gateways":[{"rssi":"-108","snr":"3.5","gatewayIdentifier":"126","antenna":0}]}]`,
}
