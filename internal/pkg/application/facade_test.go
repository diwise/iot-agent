package application

import (
	"encoding/json"
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
		se, err := Netmore([]byte(ue))
		is.NoErr(err)

		_, err = json.Marshal(se)
		is.NoErr(err)
	}
}

func TestPayloadWithError(t *testing.T) {
	is := is.New(t)

	e, err := ChirpStack([]byte(payloadWithError))

	is.NoErr(err)
	is.Equal(e.Error.Type, "UPLINK_FCNT_RETRANSMISSION")
	is.Equal(e.Error.Message, "frame-counter did not increment")
}

var uplinkChirpStack = []string{
	`{"applicationID":"ttt","applicationName":"Soraker","deviceName":"05343464","deviceProfileName":"Axioma_Universal_Codec","deviceProfileID":"d45461aa-e877-4c09-8b52-0b41e670359f","devEUI":"lkfjslkdfu39w0woejf","rxInfo":[{"gatewayID":"dflgj34209rtues","uplinkID":"f4b3f1df-9ca2-4a7c-a84c-ac522bddebb2","name":"SN-LGW-017","time":"2022-10-18T15:39:39.088645922Z","rssi":-113,"loRaSNR":-5.5,"location":{"latitude":62.504757783826896,"longitude":17.51152038574219,"altitude":0}}],"txInfo":{"frequency":868100000,"dr":3},"adr":true,"fCnt":4107,"fPort":100,"data":"AebITmMAAKoCAAAAAKAAIAAGQADwBPgABsAAUAAAAABAFEAEAAAEQAJQACAAB4AAUAAA","object":{"curDateTime":"2022-10-18 17:40:22","curVol":174592,"deltaVol":{"id1":0,"id10":3,"id11":5,"id12":0,"id13":0,"id14":81,"id15":68,"id16":0,"id17":4,"id18":9,"id19":5,"id2":0,"id20":8,"id21":7,"id22":2,"id23":5,"id3":10,"id4":8,"id5":6,"id6":1,"id7":79,"id8":62,"id9":6},"frameVersion":1,"statusCode":0},"tags":{"SerialNo":"05343464"}}`,
	`{"deviceName":"sn-elt-livboj-01","devEUI":"a81758fffe04d855","data":"Bw4ADQA=","object":{"present":false}}`,
	`{"deviceName":"sk-elt-temp-31","deviceProfileName":"Elsys_Codec","deviceProfileID":"f113c342-4048-4df5-8e83-a2642d66990d","devEUI":"df98ge4rth495345","data":"Bw5BDADk","object":{"externalTemperature":22.8,"vdd":3649},"tags":{"Location":"Norrhassel"}}`,
	`{"applicationID":"tttt","applicationName":"Air-Temperature","deviceName":"AIR-SENS-02","deviceProfileName":"Elsys_Codec","deviceProfileID":"f113c342-4048-4df5-8e83-a2642d66990d","devEUI":"dfge5t634tr4545","rxInfo":[{"gatewayID":"435rgdf4343534tre","uplinkID":"b761e459-84da-4d03-9f63-06a9653f4f1e","name":"SN-LGW-047","time":"2022-10-18T15:43:12.577940261Z","rssi":-118,"loRaSNR":-9,"location":{"latitude":62.36956091265246,"longitude":17.319844410529534,"altitude":0}}],"txInfo":{"frequency":867700000,"dr":4},"adr":true,"fCnt":92871,"fPort":5,"data":"AQAzAlQUAA9q7Q==","object":{"humidity":84,"pressure":1010.413,"temperature":5.1},"tags":{"Location":"Norraberget (Ryggis)"}}`,
	`{"applicationID":"tttttt","applicationName":"Bergsaker","deviceName":"05343338","deviceProfileName":"Axioma_Universal_Codec","deviceProfileID":"d45461aa-e877-4c09-8b52-0b41e670359f","devEUI":"345dfg34ttg435t43t4","rxInfo":[{"gatewayID":"dfg34t4g43t43t","uplinkID":"4f3b871f-66d3-4b7f-8543-74d30a0e97a3","name":"SN-LGW-105","time":"2022-10-18T15:27:18.948345Z","rssi":-119,"loRaSNR":-0.5,"location":{"latitude":62.42065486301624,"longitude":17.201843261718754,"altitude":0}}],"txInfo":{"frequency":867500000,"dr":5},"adr":true,"fCnt":703,"fPort":100,"data":"Ac3FTmMACcUDAAAAAVABcAAQAAAAAAAABUAAgAA0AAKAAIAAFAAXgACQAFgAAwAAAAAA","object":{"curDateTime":"2022-10-18 17:27:9","curVol":247049,"deltaVol":{"id1":0,"id10":1,"id11":8,"id12":13,"id13":2,"id14":2,"id15":8,"id16":5,"id17":23,"id18":2,"id19":9,"id2":4,"id20":22,"id21":3,"id22":0,"id23":0,"id3":21,"id4":28,"id5":16,"id6":0,"id7":0,"id8":0,"id9":5},"frameVersion":1,"statusCode":0},"tags":{"SerialNo":"05343338"}}`,
	`{"applicationID":"ttt","applicationName":"Sensorer-mot-diwise","deviceName":"Airquality_4","deviceProfileName":"Airquality","deviceProfileID":"4e85d145-5bd8-4bfa-a581-625e168ca001","devEUI":"cmckamckslamcksa","rxInfo":[{"gatewayID":"7276ff003904082b","uplinkID":"563d681c-66b9-4f6a-931d-eab6a467f436","name":"SN-LGW-037","time":"2025-02-28T14:55:05.137765Z","rssi":-115,"loRaSNR":-8,"location":{"latitude":62.38928050228606,"longitude":17.301567601153035,"altitude":0}},{"gatewayID":"fcc23dfffe20bbad","uplinkID":"733218c1-f4ec-47e2-b959-40f1ede54cc7","name":"SN-LGW-041","rssi":-116,"loRaSNR":-3,"location":{"latitude":62.38823,"longitude":17.30878,"altitude":40}}],"txInfo":{"frequency":868100000,"dr":0},"adr":false,"fCnt":121,"fPort":2,"data":"CgAJAAUL3wEEABQn","object":{"humidity":47.9,"no2":2.0200000000000062,"pm10":4.9307,"pm10raw":1,"pm25":4.2304529408349545,"pm25raw":0.9,"sensorerror":4,"temperature":8.9}}`,
}

var uplinkNetmore = []string{
	`[{"devEui":"fdget435345345","sensorType":"cube02","timestamp":"2022-10-18T13:32:46.361551Z","payload":"b006b8000130094598000001c1a8000099000001f4a9000008416be864","spreadingFactor":"12","rssi":"-106","snr":"4.2","gatewayIdentifier":"126","fPort":"2"}]`,
	`[{"devEui":"fgert43t34t34t43t4","sensorType":"other","timestamp":"2022-10-18T13:43:04.584377Z","payload":"110a00520000410c0000000000003db600000000","spreadingFactor":"12","rssi":"-97","snr":"0.5","gatewayIdentifier":"187","fPort":"125"}]`,
	`[{"devEui":"fgh45t6435435345","sensorType":"strips_lora_ms_h","timestamp":"2022-10-18T11:50:04.138012Z","payload":"ffff0a01","spreadingFactor":"8","rssi":"-112","snr":"-3","gatewayIdentifier":"824","fPort":"1"}]`,
	`[{"devEui":"dfg34g34reg3453454","sensorType":"strips_lora_ms_wl","timestamp":"2022-10-18T11:57:33.587149Z","payload":"ffff01580200c80400c807001508001509000a010d000e00","spreadingFactor":"8","rssi":"-95","snr":"5.5","gatewayIdentifier":"187","fPort":"1"}]`,
	`[{"devEui":"43gfrdg34tgb445h","sensorType":"tem_lab_14ns","timestamp":"2022-10-18T12:12:25.509216Z","payload":"01ef82359c1000c0","spreadingFactor":"11","rssi":"-112","snr":"-6.8","gatewayIdentifier":"187","fPort":"3","latitude":57.687844,"longitude":12.036078}]`,
	`[{"devEui":"rg34g34gerg3454trg","sensorType":"qalcosonic_w1e","messageType":"payload","timestamp":"2022-10-18T12:17:03.166298Z","payload":"8e974e6300227a060050c24d63cb790600000000000000000000000000000016003300000000000000000000000e00","fCntUp":6647,"toa":null,"freq":867300000,"batteryLevel":"255","ack":false,"spreadingFactor":"8","rssi":"-108","snr":"3.5","gatewayIdentifier":"126","fPort":"100","tags":{"application":["1_kretsloppvatten_w1e_1"],"customer":["kretsloppvatten"],"deviceType":["w1e"],"facilityID":[],"municipality":[],"serial":["05014191"]},"gateways":[{"rssi":"-108","snr":"3.5","gatewayIdentifier":"126","antenna":0}]}]`,
	`[{"devEui":"a81758fffe09ec03","sensorType":"elt_2_hp","timestamp":"2023-10-30T13:57:37.868543Z","payload":"01006f0245070e270d0014000f3d221a00","spreadingFactor":"12","dr":0,"rssi":"-117","snr":"-17","gatewayIdentifier":"881","fPort":"5"}]`,
	`[{"devEui":"c4ac590000ccc60d","sensorType":"vegapuls_air_41","timestamp":"2024-02-28T11:21:59.626943Z","payload":"02003fbf3faf2d61002403","spreadingFactor":"7","dr":5,"rssi":"-107","snr":"4","gatewayIdentifier":"274","fPort":"1","tags":{"application":["thingboard_KoV"],"customer":["kretsloppvatten"],"deviceType":["vega"],"serial":["211800"]}}]`,
	`[{"devEui":"363536305b398e11","sensorType":"other","messageType":"payload","timestamp":"2024-04-05T08:16:28.321140Z","payload":"80cf36","fCntUp":2543,"toa":null,"freq":868500000,"batteryLevel":"0","ack":false,"spreadingFactor":"12","dr":0,"rssi":"-119","snr":"2.5","gatewayIdentifier":"508","fPort":"2","tags":{"GBG_KOV":[]},"gateways":[{"rssi":"-122","snr":"-2.2","gatewayIdentifier":"640","gwEui":"7076ff005605080f","mac":"7076ff030a3a","antenna":0},{"rssi":"-119","snr":"2.5","gatewayIdentifier":"508","gwEui":"7076ff0056050503","mac":"7076ff030776","antenna":0}]}]`,
}

var payloadWithError string = `{"applicationID":"1","applicationName":"Watermetering","deviceName":"yyyyyyyyyy","devEUI":"xxxxxxxxxxxxxx","type":"UPLINK_FCNT_RETRANSMISSION","error":"frame-counter did not increment","fCnt":456,"tags":{"Location":"UnSet","SerialNo":"zzzzzzzzzzz"}}`
