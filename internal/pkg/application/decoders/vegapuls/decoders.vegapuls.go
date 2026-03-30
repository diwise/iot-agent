package vegapuls

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type VegapulsPayload struct {
	PacketIdentifier     uint8             `json:"packetIdentifier"`
	NamurState           *uint8            `json:"namurState,omitempty"`
	Distance             *float64          `json:"distance,omitempty"`
	Unit                 *uint8            `json:"unit,omitempty"`
	RemainingPower       *int              `json:"remainingPower,omitempty"`
	Temperature          *float64          `json:"temperature,omitempty"`
	TemperatureUnit      *uint8            `json:"temperatureUnit,omitempty"`
	InclinationDegree    *uint8            `json:"inclinationDegree,omitempty"`
	GNSSLatitude         *float64          `json:"gnssLatitude,omitempty"`
	GNSSLongitude        *float64          `json:"gnssLongitude,omitempty"`
	DetailState          *uint32           `json:"detailState,omitempty"`
	PercentValue         *float64          `json:"percentValue,omitempty"`
	LinPercentValue      *float64          `json:"linPercentValue,omitempty"`
	ScaledValue          *float64          `json:"scaledValue,omitempty"`
	ScaledValueUnit      *uint8            `json:"scaledValueUnit,omitempty"`
	Information          *uint8            `json:"information,omitempty"`
	DtmID                *uint32           `json:"dtmId,omitempty"`
	ManufacturerID       *uint32           `json:"manufacturerId,omitempty"`
	DeviceType           *uint32           `json:"deviceType,omitempty"`
	SoftwareVersionASCII *string           `json:"softwareVersionAscii,omitempty"`
	Schedule             *VegapulsSchedule `json:"schedule,omitempty"`
	ChangeCounter        *uint16           `json:"changeCounter,omitempty"`
	ScaledMin            *float64          `json:"scaledMin,omitempty"`
	ScaledMax            *float64          `json:"scaledMax,omitempty"`
	DeviceName           *string           `json:"deviceName,omitempty"`
	DeviceTag            *string           `json:"deviceTag,omitempty"`
}

type VegapulsSchedule struct {
	Days                        []string `json:"days,omitempty"`
	StartTimeMinutes            uint16   `json:"startTimeMinutes"`
	EndTimeMinutes              uint16   `json:"endTimeMinutes"`
	MeasureIntervalMinutes      uint16   `json:"measureIntervalMinutes"`
	TransmissionIntervalMinutes uint16   `json:"transmissionIntervalMinutes"`
}

func (a VegapulsPayload) BatteryLevel() *int {
	return a.RemainingPower
}

func (a VegapulsPayload) Error() (string, []string) {
	return "", []string{}
}

func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	return decode(e.Payload.Data)
}

func Converter(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(VegapulsPayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p VegapulsPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := make([]lwm2m.Lwm2mObject, 0, 3)

	if battery := p.BatteryLevel(); battery != nil {
		d := lwm2m.NewDevice(deviceID, ts)
		bat := int(*battery)
		d.BatteryLevel = &bat
		objects = append(objects, d)
	}

	if distance, ok := normalizedDistance(p); ok && !math.IsNaN(distance) && !math.IsInf(distance, 0) {
		dist := roundFloat(distance, 5)
		objects = append(objects, lwm2m.NewDistance(deviceID, dist, ts))
	}

	if temperature, ok := normalizedTemperature(p); ok && !math.IsNaN(temperature) && !math.IsInf(temperature, 0) {
		objects = append(objects, lwm2m.NewTemperature(deviceID, temperature, ts))
	}

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

const (
	distanceUnitFeet       uint8 = 44
	distanceUnitMeters     uint8 = 45
	distanceUnitInches     uint8 = 47
	distanceUnitMillimeter uint8 = 49

	temperatureUnitCelsius    uint8 = 32
	temperatureUnitFahrenheit uint8 = 33
)

func decode(b []byte) (VegapulsPayload, error) {
	p := VegapulsPayload{}
	if len(b) == 0 {
		return p, types.ErrPayloadEmpty
	}

	packetIdentifier := b[0]
	p.PacketIdentifier = packetIdentifier
	decoder := payloadDecoder{payload: b, position: 1}

	switch {
	case packetIdentifier >= 2 && packetIdentifier <= 5:
		if err := decodePacket2To5(&decoder, packetIdentifier, &p); err != nil {
			return p, err
		}
	case packetIdentifier == 6:
		if err := decodePacket6(&decoder, &p); err != nil {
			return p, err
		}
	case packetIdentifier == 7:
		if err := decodePacket7(&decoder, &p); err != nil {
			return p, err
		}
	case packetIdentifier >= 8 && packetIdentifier <= 15:
		if err := decodePacket8To22(&decoder, packetIdentifier, &p); err != nil {
			return p, err
		}
	case packetIdentifier >= 16 && packetIdentifier <= 17:
		if err := decodePacket16To26(&decoder, packetIdentifier, &p); err != nil {
			return p, err
		}
	case packetIdentifier >= 18 && packetIdentifier <= 22:
		if err := decodePacket8To22(&decoder, packetIdentifier, &p); err != nil {
			return p, err
		}
	case packetIdentifier >= 23 && packetIdentifier <= 26:
		if err := decodePacket16To26(&decoder, packetIdentifier, &p); err != nil {
			return p, err
		}
	default:
		return p, fmt.Errorf("unknown packet identifier: %d", packetIdentifier)
	}

	return p, nil
}

type payloadDecoder struct {
	payload  []byte
	position int
}

func (d *payloadDecoder) take(count int) ([]byte, error) {
	if len(d.payload[d.position:]) < count {
		return nil, types.ErrUnsupportedPayloadLength
	}

	start := d.position
	d.position += count
	return d.payload[start:d.position], nil
}

func (d *payloadDecoder) readUint8() (uint8, error) {
	buf, err := d.take(1)
	if err != nil {
		return 0, err
	}

	return buf[0], nil
}

func (d *payloadDecoder) readInt16() (int16, error) {
	buf, err := d.take(2)
	if err != nil {
		return 0, err
	}

	return int16(binary.BigEndian.Uint16(buf)), nil
}

func (d *payloadDecoder) readUint16() (uint16, error) {
	buf, err := d.take(2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(buf), nil
}

func (d *payloadDecoder) readUint32() (uint32, error) {
	buf, err := d.take(4)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(buf), nil
}

func (d *payloadDecoder) readFloat32() (float64, error) {
	buf, err := d.take(4)
	if err != nil {
		return 0, err
	}

	return bytesToFloat(buf), nil
}

func (d *payloadDecoder) readFloat32WithRaw() ([]byte, float64, error) {
	buf, err := d.take(4)
	if err != nil {
		return nil, 0, err
	}

	return buf, bytesToFloat(buf), nil
}

func (d *payloadDecoder) readString(count int) (string, error) {
	buf, err := d.take(count)
	if err != nil {
		return "", err
	}

	return bytesToString(buf), nil
}

func decodePacket2To5(d *payloadDecoder, packetIdentifier uint8, p *VegapulsPayload) error {
	namurState, err := d.readUint8()
	if err != nil {
		return err
	}
	p.NamurState = new(namurState)

	distance, err := d.readFloat32()
	if err != nil {
		return err
	}
	p.Distance = new(distance)

	unit, err := d.readUint8()
	if err != nil {
		return err
	}
	p.Unit = new(unit)

	remainingPower, err := d.readUint8()
	if err != nil {
		return err
	}
	p.RemainingPower = new(int(remainingPower))

	temperature, err := d.readInt16()
	if err != nil {
		return err
	}
	p.Temperature = new(float64(temperature) / 10)
	p.TemperatureUnit = new(temperatureUnitCelsius)

	switch packetIdentifier {
	case 2:
		inclinationDegree, err := d.readUint8()
		if err != nil {
			return err
		}
		p.InclinationDegree = new(inclinationDegree)
	case 3:
		if err := decodeGPS(d, &p.GNSSLatitude, &p.GNSSLongitude); err != nil {
			return err
		}

		inclinationDegree, err := d.readUint8()
		if err != nil {
			return err
		}
		p.InclinationDegree = new(inclinationDegree)
	case 4:
		detailState, err := d.readUint32()
		if err != nil {
			return err
		}
		p.DetailState = new(detailState)

		inclinationDegree, err := d.readUint8()
		if err != nil {
			return err
		}
		p.InclinationDegree = new(inclinationDegree)
	case 5:
		if err := decodeGPS(d, &p.GNSSLatitude, &p.GNSSLongitude); err != nil {
			return err
		}

		detailState, err := d.readUint32()
		if err != nil {
			return err
		}
		p.DetailState = new(detailState)

		inclinationDegree, err := d.readUint8()
		if err != nil {
			return err
		}
		p.InclinationDegree = new(inclinationDegree)
	}

	return nil
}

func decodePacket6(d *payloadDecoder, p *VegapulsPayload) error {
	namurState, err := d.readUint8()
	if err != nil {
		return err
	}
	p.NamurState = new(namurState)

	return decodeGPS(d, &p.GNSSLatitude, &p.GNSSLongitude)
}

func decodePacket7(d *payloadDecoder, p *VegapulsPayload) error {
	namurState, err := d.readUint8()
	if err != nil {
		return err
	}
	p.NamurState = new(namurState)

	detailState, err := d.readUint32()
	if err != nil {
		return err
	}
	p.DetailState = new(detailState)

	return nil
}

func decodePacket8To22(d *payloadDecoder, packetIdentifier uint8, p *VegapulsPayload) error {
	if hasNamurStateInVariablePacket(packetIdentifier) {
		namurState, err := d.readUint8()
		if err != nil {
			return err
		}
		p.NamurState = new(namurState)
	} else {
		p.NamurState = new(uint8(0))
	}

	if hasDistance(packetIdentifier) {
		distance, err := d.readFloat32()
		if err != nil {
			return err
		}
		p.Distance = new(distance)

		unit, err := d.readUint8()
		if err != nil {
			return err
		}
		p.Unit = new(unit)
	}

	if hasPercentValues(packetIdentifier) {
		percentValue, err := d.readNaNableHundredths()
		if err != nil {
			return err
		}
		p.PercentValue = new(percentValue)

		linPercentValue, err := d.readNaNableHundredths()
		if err != nil {
			return err
		}
		p.LinPercentValue = new(linPercentValue)

		scaledValue, err := d.readFloat32()
		if err != nil {
			return err
		}
		p.ScaledValue = new(scaledValue)

		scaledValueUnit, err := d.readUint8()
		if err != nil {
			return err
		}
		p.ScaledValueUnit = new(scaledValueUnit)
	}

	if hasRemainingPower(packetIdentifier) {
		remainingPower, err := d.readUint8()
		if err != nil {
			return err
		}
		p.RemainingPower = new(int(remainingPower))
	}

	if hasGPS(packetIdentifier) {
		if err := decodeGPS(d, &p.GNSSLatitude, &p.GNSSLongitude); err != nil {
			return err
		}
	}

	if hasDetailState(packetIdentifier) {
		detailState, err := d.readUint32()
		if err != nil {
			return err
		}
		p.DetailState = new(detailState)
	}

	if hasTemperature(packetIdentifier) {
		temperature, err := d.readNaNableTenths()
		if err != nil {
			return err
		}
		p.Temperature = new(temperature)

		temperatureUnit, err := d.readUint8()
		if err != nil {
			return err
		}
		p.TemperatureUnit = new(temperatureUnit)
	}

	if hasInclination(packetIdentifier) {
		inclinationDegree, err := d.readUint8()
		if err != nil {
			return err
		}
		p.InclinationDegree = new(inclinationDegree)
	}

	return nil
}

func decodePacket16To26(d *payloadDecoder, packetIdentifier uint8, p *VegapulsPayload) error {
	if hasNamurStateInInfoPacket(packetIdentifier) {
		namurState, err := d.readUint8()
		if err != nil {
			return err
		}
		p.NamurState = new(namurState)
	}

	if hasInformation(packetIdentifier) {
		information, err := d.readUint8()
		if err != nil {
			return err
		}
		p.Information = new(information)
	}

	if hasDtmMetadata(packetIdentifier) {
		dtmID, err := d.readUint32()
		if err != nil {
			return err
		}
		p.DtmID = new(dtmID)

		manufacturerID, err := d.readUint32()
		if err != nil {
			return err
		}
		p.ManufacturerID = new(manufacturerID)
	}

	if hasDeviceMetadata(packetIdentifier) {
		deviceType, err := d.readUint32()
		if err != nil {
			return err
		}
		p.DeviceType = new(deviceType)

		versionBytes, err := d.take(4)
		if err != nil {
			return err
		}
		softwareVersion := fmt.Sprintf("%d.%d.%d.%d", versionBytes[0], versionBytes[1], versionBytes[2], versionBytes[3])
		p.SoftwareVersionASCII = &softwareVersion
	}

	if hasSchedule(packetIdentifier) {
		scheduleBytes, err := d.take(7)
		if err != nil {
			return err
		}
		schedule := decodeSchedule(scheduleBytes)
		p.Schedule = &schedule

		changeCounter, err := d.readUint16()
		if err != nil {
			return err
		}
		p.ChangeCounter = new(changeCounter)
	}

	if hasScaledRange(packetIdentifier) {
		scaledMin, err := d.readFloat32()
		if err != nil {
			return err
		}
		p.ScaledMin = new(scaledMin)

		scaledMax, err := d.readFloat32()
		if err != nil {
			return err
		}
		p.ScaledMax = new(scaledMax)
	}

	if packetIdentifier == 17 {
		deviceName, err := d.readString(19)
		if err != nil {
			return err
		}
		p.DeviceName = &deviceName

		deviceTag, err := d.readString(19)
		if err != nil {
			return err
		}
		p.DeviceTag = &deviceTag
	}

	return nil
}

func decodeGPS(d *payloadDecoder, latitude **float64, longitude **float64) error {
	rawLatitude, lat, err := d.readFloat32WithRaw()
	if err != nil {
		return err
	}
	assignGPSCoordinate(latitude, rawLatitude, lat)

	rawLongitude, lon, err := d.readFloat32WithRaw()
	if err != nil {
		return err
	}
	assignGPSCoordinate(longitude, rawLongitude, lon)

	return nil
}

func assignGPSCoordinate(target **float64, raw []byte, value float64) {
	if math.IsNaN(value) && raw[0]&0x80 != 0 {
		return
	}

	*target = new(value)
}

func (d *payloadDecoder) readNaNableHundredths() (float64, error) {
	raw, err := d.take(2)
	if err != nil {
		return 0, err
	}

	if raw[0] == 0x80 && raw[1] == 0x00 {
		return math.NaN(), nil
	}

	return float64(int16(binary.BigEndian.Uint16(raw))) / 100, nil
}

func (d *payloadDecoder) readNaNableTenths() (float64, error) {
	raw, err := d.take(2)
	if err != nil {
		return 0, err
	}

	if raw[0] == 0x80 && raw[1] == 0x00 {
		return math.NaN(), nil
	}

	return float64(int16(binary.BigEndian.Uint16(raw))) / 10, nil
}

func bytesToFloat(buf []byte) float64 {
	return float64(math.Float32frombits(binary.BigEndian.Uint32(buf)))
}

func bytesToString(buf []byte) string {
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}

	return string(buf)
}

func decodeSchedule(buf []byte) VegapulsSchedule {
	daysRaw := buf[0] >> 1 & 0x7f
	dayNames := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	days := make([]string, 0, len(dayNames))
	for i, dayName := range dayNames {
		if daysRaw&(1<<i) != 0 {
			days = append(days, dayName)
		}
	}

	return VegapulsSchedule{
		Days:                        days,
		StartTimeMinutes:            uint16(buf[0]&0x01)<<10 | uint16(buf[1])<<2 | uint16((buf[2]&0xC0)>>6),
		EndTimeMinutes:              uint16(buf[2]&0x3F)<<5 | uint16((buf[3]&0xF8)>>3),
		MeasureIntervalMinutes:      uint16(buf[3]&0x07)<<8 | uint16(buf[4]),
		TransmissionIntervalMinutes: uint16(buf[5]&0x07)<<8 | uint16(buf[6]),
	}
}

func hasNamurStateInVariablePacket(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 10, 11, 14, 15, 19, 20, 21, 22:
		return true
	default:
		return false
	}
}

func hasDistance(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 8, 9, 10, 11, 12, 13, 14, 15, 18, 19:
		return true
	default:
		return false
	}
}

func hasPercentValues(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 12, 13, 14, 15, 22:
		return true
	default:
		return false
	}
}

func hasRemainingPower(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 8, 9, 10, 11, 12, 13, 14, 15, 18, 19:
		return true
	default:
		return false
	}
}

func hasGPS(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 9, 11, 13, 15, 21:
		return true
	default:
		return false
	}
}

func hasDetailState(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 10, 11, 14, 15, 20:
		return true
	default:
		return false
	}
}

func hasTemperature(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 8, 9, 10, 11, 12, 13, 14, 15, 18, 20:
		return true
	default:
		return false
	}
}

func hasInclination(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 8, 9, 10, 11, 12, 13, 14, 15, 18, 19:
		return true
	default:
		return false
	}
}

func hasNamurStateInInfoPacket(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 23, 24, 25, 26:
		return true
	default:
		return false
	}
}

func hasInformation(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 16, 23:
		return true
	default:
		return false
	}
}

func hasDtmMetadata(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 16, 23:
		return true
	default:
		return false
	}
}

func hasDeviceMetadata(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 16, 24:
		return true
	default:
		return false
	}
}

func hasSchedule(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 16, 25:
		return true
	default:
		return false
	}
}

func hasScaledRange(packetIdentifier uint8) bool {
	switch packetIdentifier {
	case 16, 26:
		return true
	default:
		return false
	}
}

func normalizedDistance(p VegapulsPayload) (float64, bool) {
	if p.Distance == nil {
		return 0, false
	}

	distance := *p.Distance
	if p.Unit == nil {
		return distance, true
	}

	switch *p.Unit {
	case distanceUnitFeet:
		distance *= 0.3048
	case distanceUnitInches:
		distance *= 0.0254
	case distanceUnitMillimeter:
		distance /= 1000
	case distanceUnitMeters:
	}

	return distance, true
}

func normalizedTemperature(p VegapulsPayload) (float64, bool) {
	if p.Temperature == nil {
		return 0, false
	}

	temperature := *p.Temperature
	if p.TemperatureUnit != nil && *p.TemperatureUnit == temperatureUnitFahrenheit {
		temperature = (temperature - 32) * 5 / 9
	}

	return temperature, true
}

//go:fix inline
func ptr[T any](value T) *T {
	return new(value)
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
