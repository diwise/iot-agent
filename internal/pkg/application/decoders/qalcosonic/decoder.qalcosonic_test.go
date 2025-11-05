package qalcosonic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"

	"github.com/matryer/is"
)

func TestQalcosonic_w1t(t *testing.T) {
	is, _ := testSetup(t)

	p, err := w1t(bytes.NewReader(qalcosonic("55cb585f7cf29d0400120ae0fe575f8a570400cd04cb04cc04cd04ca04c404c504c404f004e604dc04d604b9057905")))

	is.NoErr(err)

	is.Equal(*p.Temperature, uint16(2578))
	is.Equal(15, len(p.Volumes))
}

func TestQalcosonic_w1t_lwm2m(t *testing.T) {
	is, _ := testSetup(t)

	p, ap, err := decode(qalcosonic_ue("55cb585f7cf29d0400120ae0fe575f8a570400cd04cb04cc04cd04ca04c404c504c404f004e604dc04d604b9057905"))

	is.NoErr(err)
	is.True(ap == nil)

	is.Equal(float64(302578), p.Current)

	objects := convertToLwm2mObjects(context.Background(), "", p, nil)
	is.Equal(16, len(objects))

	is.Equal(float64(25.78), objects[15].(lwm2m.Temperature).SensorValue)
}

func TestQalcosonic_w1h(t *testing.T) {
	is, _ := testSetup(t)

	p, err := w1h(bytes.NewReader(qalcosonic("011fbfd05e30cd0f0800d4879e41865c1b42470d7283b8201608fec181981dd007f3919460218247b631784c1c9e87b8e17600")))

	is.NoErr(err)

	is.Equal(24, len(p.Volumes))
	is.Equal(uint8(48), p.StatusCode)
}

func TestQalcosonic_w1h_lwm2m(t *testing.T) {
	is, _ := testSetup(t)

	p, ap, err := decode(qalcosonic_ue("011fbfd05e30cd0f0800d4879e41865c1b42470d7283b8201608fec181981dd007f3919460218247b631784c1c9e87b8e17600"))

	is.NoErr(err)
	is.True(ap == nil)

	is.Equal(float64(573662), p.Current)

	objects := convertToLwm2mObjects(context.Background(), "", p, nil)
	is.Equal(24, len(objects))

	is.Equal(float64(560.639), objects[16].(lwm2m.WaterMeter).CumulatedWaterVolume)
}

func TestQalcosonic_w1e(t *testing.T) {
	is, _ := testSetup(t)

	p, err := w1e(bytes.NewReader(qalcosonic("0ea0355d302935000054c0345de7290000b800b900b800b800b800b900b800b800b800b800b800b800b900b900b900")))

	is.NoErr(err)

	is.Equal(16, len(p.Volumes))
	is.Equal(uint8(48), p.StatusCode)
}

func TestQalcosonic_w1e_lwm2m(t *testing.T) {
	is, _ := testSetup(t)

	p, err := w1e(bytes.NewReader(qalcosonic("0ea0355d302935000054c0345de7290000b800b900b800b800b800b900b800b800b800b800b800b800b900b900b900")))

	is.NoErr(err)
	is.Equal(16, len(p.Volumes))

	is.Equal(float64(13609), p.Current)

	objects := convertToLwm2mObjects(context.Background(), "", &p, nil)
	is.Equal(16, len(objects))

	is.Equal(float64(13.492), objects[15].(lwm2m.WaterMeter).CumulatedWaterVolume)
}

func TestQalcosonicAlarmMessage(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(qalcosonic_alarmpacket))
	p, ap, err := decode(context.Background(), ue)

	is.NoErr(err)

	is.True(p == nil)
	is.True(ap.StatusCode == uint8(136))
}

type volume struct {
	Timestamp time.Time
	Volume    float64
	Temp      *uint16
}

func TestDecodeW1t(t *testing.T) {
	is, _ := testSetup(t)

	hex := []string{
		"3883B268007BB06B008D0560B2B1681F806B00A50465067E06B706BB03A6028501AC00620053006300CF014E03A604", // 9/30/2025, 8:35:29 AM
		"F94AB26800F8A16B002006207AB16871716B00370325036103F104A50465067E06B706BB03A6028501AC0062005300",
		"B912B2680019A06B00AF05E041B16805646B00C202A10272039704370325036103F104A50465067E06B706BB03A602",
		"78DAB16800B9906B00AE05A009B1682F526B007003B104C805ED03C202A10272039704370325036103F104A5046506",
		"38A2B16800A57A6B00940560D1B068CD4F6B0058005900610050017003B104C805ED03C202A1027203970437032503",
		"F869B16800296C6B009C052099B0687A466B003704EF029701960058005900610050017003B104C805ED03C202A102",
		"B731B16800285F6B009905E060B068BC2B6B001A0649072E082D053704EF029701960058005900610050017003B104",
		"76F9B06800D4506B001706A028B06866176B005604EC046204B2061A0649072E082D053704EF029701960058005900",
		"37C1B06800194F6B00CC0560F0AF68BBFE6A009105F9065D07C4045604EC046204B2061A0649072E082D053704EF02",
		"F788B06800BC3F6B00C70520B8AF683BF26A00DF0036024F031C069105F9065D07C4045604EC046204B2061A064907", // 9/28/2025, 8:35:28 PM
		"B2BC9E680034CF6600B506E0EB9D68689D6600CE05A007AA050B052B04FF0222017D0098007E0065004A010304BD05", // 9/15/2025, 8:35:15 AM
		"70849E6800BDBF6600FF06A0B39D68EC8766003504C30500068405CE05A007AA050B052B04FF0222017D0098007E00",
		"304C9E6800C5BD6600DB06607B9D681C74660003053805DF04B6043504C30500068405CE05A007AA050B052B04FF02",
		"EF139E6800DAAF6600C20620439D68006A66000601B4016003020403053805DF04B6043504C30500068405CE05A007",
		"AFDB9D680020976600C706E00A9D68AE676600EA00910058007F000601B4016003020403053805DF04B6043504C305",
		"6EA39D68005A826600CA06A0D29C686C5E66007B039402C4016F01EA00910058007F000601B4016003020403053805",
		"2F6B9D6800916F6600D006609A9C681E4E6600C504DF03ED04BD027B039402C4016F01EA00910058007F000601B401", // 9/14/2025, 8:35:11 AM
		"0681E8680094760D008305D0B1E768DC6B0D00AA00F3001B017801CF017D011801390146002A004300160005000800", // 10/10/2025, 5:43:10 AM
		"41D8E768003A6E0D0056051009E7687F640D000400040035002F0184017301BE009B00500063004400AA00AA00F300",
		"BE67E7680074690D0062059098E668235E0D007B016501E801F100710027000A0001000400040035002F0184017301",
		"7D2FE7680096640D0076055060E6681D590D00AA000301510108027B016501E801F100710027000A00010004000400",
		"F9BEE668007D620D006005D0EFE56809530D003602E3007C00CD008700490086005C00AA000301510108027B016501", // 10/8/2025, 9:42:58 PM
	}

	vol := []volume{}

	for _, h := range slices.Backward(hex) {
		p, err := w1t(bytes.NewReader(qalcosonic(h)))
		is.NoErr(err)

		vol = append(vol, volume{
			Timestamp: p.Timestamp,
			Volume:    p.Current,
			Temp:      p.Temperature,
		})

		is.True(p.Temperature != nil)
		is.Equal(15, len(p.Volumes))
		// fmt.Printf("ts: %s, vol: %f, temp: %d\n", p.Timestamp.Format(time.RFC3339), p.CurrentVolume, *p.Temperature)

		obj := convertToLwm2mObjects(context.Background(), "", &p, nil)
		is.Equal(16, len(obj))

		for _, d := range p.Volumes {
			vol = append(vol, volume{
				Timestamp: d.Timestamp,
				Volume:    d.Volume,
			})
		}
	}

	slices.SortFunc(vol, func(a, b volume) int {
		if a.Timestamp.Before(b.Timestamp) {
			return -1
		} else if a.Timestamp.After(b.Timestamp) {
			return 1
		}
		return 0
	})

	between := func(v volume, start, end time.Time) bool {
		return (v.Timestamp.Equal(start) || v.Timestamp.After(start)) && v.Timestamp.Before(end)
	}

	vol1, vol2, vol3 := 0.0, 0.0, 0.0

	for _, v := range vol {
		// payloads from sensor 1
		if between(v, time.Date(2025, 8, 13, 14, 0, 0, 0, time.UTC), time.Date(2025, 8, 28, 2, 0, 0, 0, time.UTC)) {
			is.True(v.Volume >= vol1)
			vol1 = v.Volume
		}
		// payloads from sensor 2
		if between(v, time.Date(2025, 8, 28, 2, 0, 0, 0, time.UTC), time.Date(2025, 10, 8, 5, 0, 0, 0, time.UTC)) {
			is.True(v.Volume >= vol2)
			vol2 = v.Volume
		}
		// payloads from sensor 3
		if between(v, time.Date(2025, 10, 8, 5, 0, 0, 0, time.UTC), time.Date(2025, 10, 10, 5, 0, 0, 0, time.UTC)) {
			is.True(v.Volume >= vol3)
			vol3 = v.Volume
		}
	}
}

func TestDecode(t *testing.T) {
	is, _ := testSetup(t)

	hex := []string{
		"3883B268007BB06B008D0560B2B1681F806B00A50465067E06B706BB03A6028501AC00620053006300CF014E03A604", // 9/30/2025, 8:35:29 AM
		"F94AB26800F8A16B002006207AB16871716B00370325036103F104A50465067E06B706BB03A6028501AC0062005300",
		"B912B2680019A06B00AF05E041B16805646B00C202A10272039704370325036103F104A50465067E06B706BB03A602",
		"78DAB16800B9906B00AE05A009B1682F526B007003B104C805ED03C202A10272039704370325036103F104A5046506",
		"38A2B16800A57A6B00940560D1B068CD4F6B0058005900610050017003B104C805ED03C202A1027203970437032503",
		"F869B16800296C6B009C052099B0687A466B003704EF029701960058005900610050017003B104C805ED03C202A102",
		"B731B16800285F6B009905E060B068BC2B6B001A0649072E082D053704EF029701960058005900610050017003B104",
		"76F9B06800D4506B001706A028B06866176B005604EC046204B2061A0649072E082D053704EF029701960058005900",
		"37C1B06800194F6B00CC0560F0AF68BBFE6A009105F9065D07C4045604EC046204B2061A0649072E082D053704EF02",
		"F788B06800BC3F6B00C70520B8AF683BF26A00DF0036024F031C069105F9065D07C4045604EC046204B2061A064907", // 9/28/2025, 8:35:28 PM
		"B2BC9E680034CF6600B506E0EB9D68689D6600CE05A007AA050B052B04FF0222017D0098007E0065004A010304BD05", // 9/15/2025, 8:35:15 AM
		"70849E6800BDBF6600FF06A0B39D68EC8766003504C30500068405CE05A007AA050B052B04FF0222017D0098007E00",
		"304C9E6800C5BD6600DB06607B9D681C74660003053805DF04B6043504C30500068405CE05A007AA050B052B04FF02",
		"EF139E6800DAAF6600C20620439D68006A66000601B4016003020403053805DF04B6043504C30500068405CE05A007",
		"AFDB9D680020976600C706E00A9D68AE676600EA00910058007F000601B4016003020403053805DF04B6043504C305",
		"6EA39D68005A826600CA06A0D29C686C5E66007B039402C4016F01EA00910058007F000601B4016003020403053805",
		"2F6B9D6800916F6600D006609A9C681E4E6600C504DF03ED04BD027B039402C4016F01EA00910058007F000601B401", // 9/14/2025, 8:35:11 AM
		"0681E8680094760D008305D0B1E768DC6B0D00AA00F3001B017801CF017D011801390146002A004300160005000800", // 10/10/2025, 5:43:10 AM
		"41D8E768003A6E0D0056051009E7687F640D000400040035002F0184017301BE009B00500063004400AA00AA00F300",
		"BE67E7680074690D0062059098E668235E0D007B016501E801F100710027000A0001000400040035002F0184017301",
		"7D2FE7680096640D0076055060E6681D590D00AA000301510108027B016501E801F100710027000A00010004000400",
		"F9BEE668007D620D006005D0EFE56809530D003602E3007C00CD008700490086005C00AA000301510108027B016501", // 10/8/2025, 9:42:58 PM
	}

	vol := []volume{}

	for _, h := range slices.Backward(hex) {
		// Test that decode uses correct decoder (should use w1t for all payloads here)
		p, _, err := decode(qalcosonic_ue(h))
		is.NoErr(err)

		vol = append(vol, volume{
			Timestamp: p.Timestamp,
			Volume:    p.Current,
			Temp:      p.Temperature,
		})

		is.True(p.Temperature != nil)
		is.Equal(15, len(p.Volumes))
		// fmt.Printf("ts: %s, vol: %f, temp: %d\n", p.Timestamp.Format(time.RFC3339), p.CurrentVolume, *p.Temperature)

		obj := convertToLwm2mObjects(context.Background(), "", p, nil)
		is.Equal(16, len(obj))

		for _, d := range p.Volumes {
			vol = append(vol, volume{
				Timestamp: d.Timestamp,
				Volume:    d.Volume,
			})
		}
	}

	slices.SortFunc(vol, func(a, b volume) int {
		if a.Timestamp.Before(b.Timestamp) {
			return -1
		} else if a.Timestamp.After(b.Timestamp) {
			return 1
		}
		return 0
	})

	between := func(v volume, start, end time.Time) bool {
		return (v.Timestamp.Equal(start) || v.Timestamp.After(start)) && v.Timestamp.Before(end)
	}

	vol1, vol2, vol3 := 0.0, 0.0, 0.0

	for _, v := range vol {
		// payloads from sensor 1
		if between(v, time.Date(2025, 8, 13, 14, 0, 0, 0, time.UTC), time.Date(2025, 8, 28, 2, 0, 0, 0, time.UTC)) {
			is.True(v.Volume >= vol1)
			vol1 = v.Volume
		}
		// payloads from sensor 2
		if between(v, time.Date(2025, 8, 28, 2, 0, 0, 0, time.UTC), time.Date(2025, 10, 8, 5, 0, 0, 0, time.UTC)) {
			is.True(v.Volume >= vol2)
			vol2 = v.Volume
		}
		// payloads from sensor 3
		if between(v, time.Date(2025, 10, 8, 5, 0, 0, 0, time.UTC), time.Date(2025, 10, 10, 5, 0, 0, 0, time.UTC)) {
			is.True(v.Volume >= vol3)
			vol3 = v.Volume
		}
	}
}

func TestQalcosonicStatusCodes(t *testing.T) {
	is, _ := testSetup(t)

	is.Equal("No error", getStatusMessage(0)[0])
	is.Equal("Power low", getStatusMessage(0x04)[0])
	is.Equal("Permanent error", getStatusMessage(0x08)[0])
	is.Equal("Temporary error", getStatusMessage(0x10)[0])
	is.Equal("Empty spool", getStatusMessage(0x10)[1])
	is.Equal("Leak", getStatusMessage(0x20)[0])
	is.Equal("Burst", getStatusMessage(0xA0)[0])
	is.Equal("Backflow", getStatusMessage(0x60)[0])
	is.Equal("Freeze", getStatusMessage(0x80)[0])

	is.Equal("Power low", getStatusMessage(0x0C)[0])
	is.Equal("Permanent error", getStatusMessage(0x0C)[1])

	is.Equal("Temporary error", getStatusMessage(0x10)[0])
	is.Equal("Empty spool", getStatusMessage(0x10)[1])

	is.Equal("Power low", getStatusMessage(0x14)[0])
	is.Equal("Temporary error", getStatusMessage(0x14)[1])
	is.Equal("Empty spool", getStatusMessage(0x14)[2])

	// ...

	is.Equal("Permanent error", getStatusMessage(0x18)[0])
	is.Equal("Temporary error", getStatusMessage(0x18)[1])
	is.Equal("Empty spool", getStatusMessage(0x18)[2])

	// ...

	is.Equal("Power low", getStatusMessage(0x3C)[0])
	is.Equal("Permanent error", getStatusMessage(0x3C)[1])
	is.Equal("Temporary error", getStatusMessage(0x3C)[2])
	is.Equal("Leak", getStatusMessage(0x3C)[3])

	// ...

	is.Equal("Power low", getStatusMessage(0xBC)[0])
	is.Equal("Permanent error", getStatusMessage(0xBC)[1])
	is.Equal("Temporary error", getStatusMessage(0xBC)[2])
	is.Equal("Burst", getStatusMessage(0xBC)[3])

	is.Equal("Permanent error", getStatusMessage(0x88)[0])
	is.Equal("Freeze", getStatusMessage(0x88)[1])

	is.Equal("Unknown", getStatusMessage(0x02)[0])
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

func qalcosonic_ue(hex string) (context.Context, types.Event) {
	s := fmt.Sprintf(qalcosonic_payload, hex)
	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(s))
	return context.Background(), ue
}

func qalcosonic(hex string) []byte {
	_, ue := qalcosonic_ue(hex)
	return ue.Payload.Data
}

const qalcosonic_payload string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1t",
  "messageType": "payload.Payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "Payload": "%s",
  "fCntUp": 1490,
  "toa": null,
  "freq": 867900000,
  "batteryLevel": "255",
  "ack": false,
  "spreadingFactor": "8",
  "rssi": "-115",
  "snr": "-1.8",
  "gatewayIdentifier": "000",
  "fPort": "100",
  "tags": {
    "application": ["ambiductor_test"]
  },
  "gateways": [
    {
      "rssi": "-115",
      "snr": "-1.8",
      "gatewayIdentifier": "000",
      "antenna": 0
    }
  ]
}]
`

const qalcosonic_alarmpacket string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1t",
  "messageType": "payload.Payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "Payload": "43b1315d88",
  "fCntUp": 1490,
  "toa": null,
  "freq": 867900000,
  "batteryLevel": "255",
  "ack": false,
  "spreadingFactor": "8",
  "rssi": "-115",
  "snr": "-1.8",
  "gatewayIdentifier": "000",
  "fPort": "103",
  "tags": {
    "application": ["ambiductor_test"]
  },
  "gateways": [
    {
      "rssi": "-115",
      "snr": "-1.8",
      "gatewayIdentifier": "000",
      "antenna": 0
    }
  ]
}]

`
