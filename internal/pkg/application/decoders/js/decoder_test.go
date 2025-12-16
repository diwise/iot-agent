package js

import (
	"fmt"
	"strings"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/matryer/is"
)

func TestTalkpool(t *testing.T) {
	is := is.New(t)

	ctx := t.Context()
	ue, err := facades.New("netmore")(nil, "payload", fmt.Appendf(nil, message, testPayloads[0]))
	is.NoErr(err)

	is.Equal(ue.DevEUI, "00138e0000007608")
	x, err := Decode(ctx, strings.NewReader(talkpoolDecoder), ue)
	is.NoErr(err)

	is.Equal(x["CO2"].(int64), int64(826))
	is.Equal(x["Humidity"].(float64), 43.8)
	is.Equal(x["Temperature"].(float64), 24.1)
}

const talkpoolDecoder = ` 
function toHexString(byteArray) {
    return Array.prototype.map.call(byteArray, function (byte) {
        return ('0' + (byte & 0xFF).toString(16)).slice(-2);
    }).join('');
}

function DecodeOy1210Payload(bytes, port) {
    if (port===2) {
        if(bytes.length !== 5){
            return {
				"error": "oy1210: payload must contain exactly 5 bytes"
				};
        }

        bytes = toHexString(bytes);

        var OY1210Data = {}
        OY1210Data.Temperature =  parseFloat(((parseInt(bytes.substring(0,2)+bytes.substring(4,5),16)/10)-80).toFixed(1));
        OY1210Data.Humidity    =  parseFloat(((parseInt(bytes.substring(2,4)+bytes.substring(5,6),16)/10)-25).toFixed(1))
        OY1210Data.CO2         =  parseInt(bytes.substring(6,10),16)
        return OY1210Data;
    }

    return {
		"error": "oy1210: unsupported fPort, expected 2"
		};
}

function decodeUplink(input) {
    return {
        "data": DecodeOy1210Payload(input.bytes, input.fPort)
    }
}`

var testPayloads = []string{"412B10033A"}

const message string = `[{
	"devEui":"00138e0000007608",
	"deviceName":"other",
	"sensorType":"other",
	"fPort":"2",
	"payload": "%s",
	"timestamp":"2024-02-28T11:21:59.626943Z",
	"rxInfo":{
		"gatewayId":"274",
		"rssi":-107,"snr":4
	},
	"txInfo":{},
	"error":{}
}]`
