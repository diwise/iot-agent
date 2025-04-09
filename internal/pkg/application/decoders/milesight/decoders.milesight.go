package milesight

import (
	"context"
	"encoding/binary"
	"log/slog"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type MilesightPayload struct {
	Battery      *int
	CO2          *int
	Distance     *float64
	Humidity     *float64
	Temperature  *float64
	Position     *string
	MagnetStatus *string
}

func Decoder(ctx context.Context, deviceID string, e types.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	p, err := decode(e.Data)
	if err != nil {
		return nil, err
	}

	return convertToLwm2mObjects(ctx, deviceID, p, e.Timestamp), nil
}

func Converter(ctx context.Context, deviceID string, payload any, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(MilesightPayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p MilesightPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	if p.Battery != nil {
		d := lwm2m.NewDevice(deviceID, ts)
		bat := int(*p.Battery)
		d.BatteryLevel = &bat
		objects = append(objects, d)
	}

	if p.CO2 != nil {
		co2 := float64(*p.CO2)
		objects = append(objects, lwm2m.NewAirQuality(deviceID, &co2, nil, nil, nil, ts))
	}

	if p.Distance != nil {
		objects = append(objects, lwm2m.NewDistance(deviceID, *p.Distance, ts))
	}

	if p.Humidity != nil {
		objects = append(objects, lwm2m.NewHumidity(deviceID, *p.Humidity, ts))
	}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, *p.Temperature, ts))
	}

	// TODO: use urn:oma:lwm2m:x:10351?
	if p.MagnetStatus != nil {
		objects = append(objects, lwm2m.NewDigitalInput(deviceID, *p.MagnetStatus == "open", ts))
	}

	//TODO: Position

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

func decode(bytes []byte) (MilesightPayload, error) {
	m := milesightdecoder(bytes)

	p := MilesightPayload{}

	if bat, ok := m["battery"]; ok {
		b := int(bat.(uint8))
		p.Battery = &b
	}

	if temp, ok := m["temperature"]; ok {
		t := temp.(float64)
		p.Temperature = &t
	}

	if distance, ok := m["distance"]; ok {
		d := float64(distance.(uint16)) / 1000 // meters
		p.Distance = &d
	}

	if position, ok := m["position"]; ok {
		pos := position.(string)
		p.Position = &pos
	}

	if humidity, ok := m["humidity"]; ok {
		h := humidity.(float64)
		p.Humidity = &h
	}

	if co2, ok := m["co2"]; ok {
		c := int(co2.(uint16))
		p.CO2 = &c
	}

	if magnetStatus, ok := m["magnet_status"]; ok {
		m := magnetStatus.(uint8)
		status := "open"
		if m == 0 {
			status = "close"
		}
		p.MagnetStatus = &status
	}

	return p, nil
}

func milesightdecoder(bytes []byte) map[string]any {
	var decoded = make(map[string]any)
	i := 0
	for i < len(bytes) {
		channelID := bytes[i]
		i++
		channelType := bytes[i]
		i++
		switch {
		case channelID == 0x01 && channelType == 0x75: // BATTERY
			decoded["battery"] = bytes[i]
			i += 1
		case channelID == 0x03 && channelType == 0x67: // TEMPERATURE
			decoded["temperature"] = float64(int16(binary.LittleEndian.Uint16(bytes[i:i+2]))) / 10.0
			i += 2
		case channelID == 0x03 && channelType == 0x82: // DISTANCE (EM500UDL)
			decoded["distance"] = binary.LittleEndian.Uint16(bytes[i : i+2])
			i += 2
		case channelID == 0x04 && channelType == 0x82: // DISTANCE
			decoded["distance"] = binary.LittleEndian.Uint16(bytes[i : i+2])
			i += 2
		case channelID == 0x05 && channelType == 0x00: // POSITION
			if bytes[i] == 0 {
				decoded["position"] = "normal"
			} else {
				decoded["position"] = "tilt"
			}
			i += 1
		case channelID == 0x83 && channelType == 0x67: // TEMPERATURE WITH ABNORMAL
			decoded["temperature"] = float64(int16(binary.LittleEndian.Uint16(bytes[i:i+2]))) / 10.0
			decoded["temperature_abnormal"] = bytes[i+2] != 0
			i += 3
		case channelID == 0x84 && channelType == 0x82: // DISTANCE WITH ALARMING
			decoded["distance"] = binary.LittleEndian.Uint16(bytes[i : i+2])
			decoded["distance_alarming"] = bytes[i+2] != 0
			i += 3
		case channelID == 0x04 && channelType == 0x68: // HUMIDITY
			decoded["humidity"] = float64(bytes[i]) / 2.0
			i++
		case channelID == 0x05 && channelType == 0x6a: // PIR (Activity)
			decoded["activity"] = binary.LittleEndian.Uint16(bytes[i : i+2])
			i += 2
		case channelID == 0x06 && channelType == 0x00: // MAGNET STATUS
			decoded["magnet_status"] = bytes[i]
			i += 1
		case channelID == 0x06 && channelType == 0x65: // LIGHT
			decoded["illumination"] = binary.LittleEndian.Uint16(bytes[i : i+2])
			decoded["infrared_and_visible"] = binary.LittleEndian.Uint16(bytes[i+2 : i+4])
			decoded["infrared"] = binary.LittleEndian.Uint16(bytes[i+4 : i+6])
			i += 6
		case channelID == 0x07 && channelType == 0x7d: // CO2
			decoded["co2"] = binary.LittleEndian.Uint16(bytes[i : i+2])
			i += 2
		default:
		}
	}
	return decoded
}
