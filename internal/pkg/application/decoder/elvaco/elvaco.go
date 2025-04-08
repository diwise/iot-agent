package elvaco

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

// VifMapping definierar en struktur för mätvärdesinformation
type VifMapping struct {
	Measure string
	Unit    string
	Decimal int
}

// UplinkInput representerar den inkommande datan med fPort och byte-array.
type UplinkInput struct {
	FPort int
	Bytes []byte
}

// DecodeResult innehåller dekodade data samt eventuella felmeddelanden.
type DecodeResult struct {
	Data   map[string]interface{}
	Errors []string
}

// difVifMapping definierar mappningen från DIF till VIF, där alla nycklar anges som strängar.
var difVifMapping = map[string]map[string]VifMapping{
	"34": {
		"03": {Measure: "", Unit: "", Decimal: 0},
		"13": {Measure: "", Unit: "", Decimal: 0},
	},
	"04": {
		"00":   {Measure: "energy", Unit: "kWh", Decimal: 6},
		"01":   {Measure: "energy", Unit: "kWh", Decimal: 5},
		"02":   {Measure: "energy", Unit: "kWh", Decimal: 4},
		"03":   {Measure: "energy", Unit: "kWh", Decimal: 3},
		"04":   {Measure: "energy", Unit: "kWh", Decimal: 2},
		"05":   {Measure: "energy", Unit: "kWh", Decimal: 1},
		"06":   {Measure: "energy", Unit: "kWh", Decimal: 0},
		"07":   {Measure: "energy", Unit: "kWh", Decimal: -1},
		"10":   {Measure: "volume", Unit: "m3", Decimal: 6},
		"11":   {Measure: "volume", Unit: "m3", Decimal: 5},
		"12":   {Measure: "volume", Unit: "m3", Decimal: 4},
		"13":   {Measure: "volume", Unit: "m3", Decimal: 3},
		"14":   {Measure: "volume", Unit: "m3", Decimal: 2},
		"15":   {Measure: "volume", Unit: "m3", Decimal: 1},
		"16":   {Measure: "volume", Unit: "m3", Decimal: 0},
		"17":   {Measure: "volume", Unit: "m3", Decimal: -1},
		"fd17": {Measure: "error_flag", Unit: "", Decimal: 0},
		"6d":   {Measure: "datetime_heat_meter", Unit: "", Decimal: 0},
	},
	"02": {
		"29":   {Measure: "power", Unit: "kW", Decimal: 5},
		"2a":   {Measure: "power", Unit: "kW", Decimal: 4},
		"2b":   {Measure: "power", Unit: "kW", Decimal: 3},
		"2c":   {Measure: "power", Unit: "kW", Decimal: 2},
		"2d":   {Measure: "power", Unit: "kW", Decimal: 1},
		"2e":   {Measure: "power", Unit: "kW", Decimal: 0},
		"2f":   {Measure: "power", Unit: "kW", Decimal: -1},
		"39":   {Measure: "flow", Unit: "m3/h", Decimal: 5},
		"3a":   {Measure: "flow", Unit: "m3/h", Decimal: 4},
		"3b":   {Measure: "flow", Unit: "m3/h", Decimal: 3},
		"3c":   {Measure: "flow", Unit: "m3/h", Decimal: 2},
		"3d":   {Measure: "flow", Unit: "m3/h", Decimal: 1},
		"3e":   {Measure: "flow", Unit: "m3/h", Decimal: 0},
		"3f":   {Measure: "flow", Unit: "m3/h", Decimal: -1},
		"58":   {Measure: "flow_temperature", Unit: "°C", Decimal: 3},
		"59":   {Measure: "flow_temperature", Unit: "°C", Decimal: 2},
		"5a":   {Measure: "flow_temperature", Unit: "°C", Decimal: 1},
		"5b":   {Measure: "flow_temperature", Unit: "°C", Decimal: 0},
		"5c":   {Measure: "return_temperature", Unit: "°C", Decimal: 3},
		"5d":   {Measure: "return_temperature", Unit: "°C", Decimal: 2},
		"5e":   {Measure: "return_temperature", Unit: "°C", Decimal: 1},
		"5f":   {Measure: "return_temperature", Unit: "°C", Decimal: 0},
		"fd17": {Measure: "error_flag", Unit: "", Decimal: 0},
	},
	"0c": {
		"78": {Measure: "serial_from_message", Unit: "", Decimal: 0},
	},
	"84": {
		"0201": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 5},
		"0202": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 4},
		"0203": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 3},
		"0204": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 2},
		"0205": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 1},
		"0206": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 0},
		"0207": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: -1},
		"2001": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 5},
		"2002": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 4},
		"2003": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 3},
		"2004": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 2},
		"2005": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 1},
		"2006": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: 0},
		"2007": {Measure: "energy_tariff_2", Unit: "kWh", Decimal: -1},
		"0301": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 5},
		"0302": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 4},
		"0303": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 3},
		"0304": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 2},
		"0305": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 1},
		"0306": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 0},
		"0307": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: -1},
		"3001": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 5},
		"3002": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 4},
		"3003": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 3},
		"3004": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 2},
		"3005": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 1},
		"3006": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: 0},
		"3007": {Measure: "energy_tariff_3", Unit: "kWh", Decimal: -1},
		"fd17": {Measure: "error_flag", Unit: "", Decimal: 0},
	},
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	// Exempel på payload
	bytesInput := []byte{
		5, 4, 6, 90, 38, 0, 0, 4, 20, 240, 20, 10,
		0, 2, 45, 11, 0, 2, 59, 38, 0, 2, 90, 123,
		2, 2, 94, 124, 1, 12, 120, 113, 53, 73, 105,
		4, 253, 23, 0, 0, 8, 0,
	}

	input := UplinkInput{
		FPort: 2,
		Bytes: bytesInput,
	}
	result := decodeUplink(input)
	fmt.Printf("Decoded Data: %+v\n", result.Data)
	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range result.Errors {
			fmt.Println(err)
		}
	}

	return nil, errors.New("not implemented")
}

// decodeUplink väljer avkodningsmetod beroende på fPort.
func decodeUplink(input UplinkInput) DecodeResult {
	if input.FPort == 2 {
		hexArray := bytesToHexArray(input.Bytes)
		fmt.Println("hex_array:", hexArray)
		if len(hexArray) < 40 {
			return DecodeResult{
				Data:   map[string]any{},
				Errors: []string{"payload length < 40"},
			}
		}
		if hexArray[0] != "05" {
			return DecodeResult{
				Data:   map[string]any{},
				Errors: []string{"Payload type unknown, currently standard format supported"},
			}
		}
		data, err := decodeCMI4111Standard(hexArray)
		if err != nil {
			return DecodeResult{
				Data:   map[string]interface{}{},
				Errors: []string{err.Error()},
			}
		}
		return DecodeResult{
			Data:   data,
			Errors: nil,
		}
	}
	return DecodeResult{
		Data:   map[string]interface{}{},
		Errors: []string{"unknown FPort"},
	}
}

// decodeCMI4111Standard avkodar en payload enligt standarden.
// Den returnerar en mapp med nycklar (mätarens namn) och deras värden.
func decodeCMI4111Standard(payloadArr []string) (map[string]interface{}, error) {
	decodedDictionary := make(map[string]any)
	i := 1

	for i < len(payloadArr) {
		dif := strings.ToLower(payloadArr[i])
		if i+1 >= len(payloadArr) {
			return nil, fmt.Errorf("unexpected end of payload at index %d", i)
		}
		vif := strings.ToLower(payloadArr[i+1])
		difInt, err := strconv.ParseInt(dif, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DIF at index %d: %v", i, err)
		}
		i += 2

		if len(payloadArr[i:]) <= 5 && vif == "fd" {
			if i >= len(payloadArr) {
				return nil, fmt.Errorf("unexpected end of payload when processing fd at index %d", i)
			}
			vif += strings.ToLower(payloadArr[i])
			i++
		}

		bcdLen := 4
		if difInt >= 2 && difInt <= 4 {
			bcdLen = int(difInt)
		}

		mapping, ok := difVifMapping[dif]
		if !ok {
			return nil, fmt.Errorf("unknown DIF %s at index %d", dif, i)
		}
		unitInfo, ok := mapping[vif]
		if !ok {
			return nil, fmt.Errorf("unknown VIF %s for DIF %s at index %d", vif, dif, i)
		}

		if dif == "34" {
			count := 0
			for _, v := range payloadArr {
				if v == "00" {
					count++
				}
			}
			if count > 20 {
				return nil, errors.New("empty payload, value during error state")
			}
			return nil, fmt.Errorf("unknown DIF %s and VIF %s", dif, vif)
		}

		if i+bcdLen > len(payloadArr) {
			return nil, fmt.Errorf("not enough data for BCD value at index %d", i)
		}

		slice := make([]string, bcdLen)
		copy(slice, payloadArr[i:i+bcdLen])
		for j, k := 0, len(slice)-1; j < k; j, k = j+1, k-1 {
			slice[j], slice[k] = slice[k], slice[j]
		}
		reversedValues := strings.Join(slice, "")
		var valueInt int64

		if strings.HasPrefix(reversedValues, "fff") && (unitInfo.Measure == "power" || unitInfo.Measure == "flow") {
			newStr := strings.Replace(reversedValues, "fff", "-", 1)
			valueInt, err = strconv.ParseInt(newStr, 16, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse negative value at index %d: %v", i, err)
			}
		} else {
			valueInt, err = strconv.ParseInt(reversedValues, 16, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value at index %d: %v", i, err)
			}
		}
		i += bcdLen

		var value any
		if unitInfo.Measure == "date_heat_meter" {
			return nil, errors.New("date_heat_meter is not supported yet")
		} else if unitInfo.Measure == "serial_from_message" {
			value, err = strconv.ParseInt(reversedValues, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse serial number at index %d: %v", i, err)
			}
		} else if unitInfo.Unit != "" {
			value = float64(valueInt) / math.Pow(10, float64(unitInfo.Decimal))
		} else {
			val, err := strconv.ParseInt(reversedValues, 16, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse hex value at index %d: %v", i, err)
			}
			value = val
		}
		decodedDictionary[unitInfo.Measure] = value
	}

	return decodedDictionary, nil
}

// bytesToHexArray konverterar en byte-array till en array av hex-strängar.
func bytesToHexArray(bytes []byte) []string {
	hexArray := make([]string, len(bytes))
	for i, b := range bytes {
		hexArray[i] = fmt.Sprintf("%02x", b)
	}
	return hexArray
}
