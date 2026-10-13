package decoders

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/application/decoders/mapper"
	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

var migrationTime = time.Date(2024, 2, 28, 11, 21, 59, 0, time.UTC)

func mustB64(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func packsEqual(t *testing.T, legacy, mapped []lwm2m.Lwm2mObject) {
	t.Helper()
	is := is.New(t)
	if len(legacy) != len(mapped) {
		t.Fatalf("object count differs: legacy %d mapped %d", len(legacy), len(mapped))
	}
	for i := range legacy {
		lb, _ := json.Marshal(lwm2m.ToPack(legacy[i]))
		mb, _ := json.Marshal(lwm2m.ToPack(mapped[i]))
		is.Equal(string(mb), string(lb))
	}
}

func runMigrationCase(t *testing.T, id string, event types.Event) {
	t.Helper()
	ctx := context.Background()
	reg, err := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	if err != nil {
		t.Fatalf("load real mappings: %v", err)
	}
	snap, ok := reg.Get(ctx, id)
	if !ok {
		t.Fatalf("no snapshot for %s", id)
	}
	if snap.Kind != KindGo {
		t.Fatalf("%s is not generic (kind %s)", id, snap.Kind)
	}
	legacyDec, ok := legacyDecoders[id]
	if !ok {
		t.Fatalf("no legacy decoder for %s", id)
	}
	legacyConv := legacyConverters[id]

	legacyPayload, err := legacyDec(ctx, event)
	if err != nil {
		t.Fatalf("%s legacy decode failed: %v", id, err)
	}
	legacyObjects, err := legacyConv(ctx, "dev", legacyPayload, migrationTime)
	if err != nil {
		t.Fatalf("%s legacy convert failed: %v", id, err)
	}

	genericRes, err := snap.Go(ctx, event)
	if err != nil {
		t.Fatalf("%s generic decode failed: %v", id, err)
	}
	mapped := mapper.MapToLWM2M("dev", migrationTime, genericRes.Mapping, snap.Groups, snap.Rules)
	for _, r := range mapped.Results {
		if r.Status == mapper.RuleInvalid {
			t.Fatalf("%s has invalid rule %q %s (%s)", id, r.RuleID, r.ErrorCode, r.Path)
		}
	}

	// Intentional divergences are asserted separately; here SenML must match.
	packsEqual(t, legacyObjects, mapped.Objects)

	// Stored decoderdata must equal legacy JSON (except documented cases).
	// Compare canonical maps: struct field order differs from sorted map keys.
	legacyJSON, _ := json.Marshal(legacyPayload)
	genericJSON, _ := json.Marshal(genericRes.Data)
	if id != "vegapuls_air_41" && id != "axsensor" && id != "x2climate" && id != "sensefarm" {
		var legacyMap, genericMap map[string]any
		if err := json.Unmarshal(legacyJSON, &legacyMap); err != nil {
			t.Fatalf("%s legacy JSON invalid: %v", id, err)
		}
		if err := json.Unmarshal(genericJSON, &genericMap); err != nil {
			t.Fatalf("%s generic JSON invalid: %v", id, err)
		}
		canonicalLegacy, _ := json.Marshal(legacyMap)
		canonicalGeneric, _ := json.Marshal(genericMap)
		is := is.New(t)
		is.Equal(string(canonicalGeneric), string(canonicalLegacy))
	}
}

func TestMigrationElsys(t *testing.T) {
	runMigrationCase(t, "elsys", types.Event{
		Payload:   &types.Payload{Data: mustB64("AQDuAhYEALIFAgYBxAcONA=="), FPort: 5},
		Timestamp: migrationTime,
	})
}

func TestMigrationMilesight(t *testing.T) {
	runMigrationCase(t, "milesight", types.Event{
		Payload:   &types.Payload{Data: mustB64("AXVZA2ffAARoOgd9dgM="), FPort: 5},
		Timestamp: migrationTime,
	})
}

func TestMigrationTalkpool(t *testing.T) {
	runMigrationCase(t, "talkpool/oy1210", types.Event{
		Payload:   &types.Payload{Data: mustHex("412B10033A"), FPort: 2},
		Timestamp: migrationTime,
	})
}

func TestMigrationAirquality(t *testing.T) {
	runMigrationCase(t, "airquality", types.Event{
		Payload:   &types.Payload{Data: mustB64("FgARAL0KAAMAAB0n"), FPort: 2},
		Timestamp: migrationTime,
	})
}

func TestMigrationAxsensor(t *testing.T) {
	ctx := context.Background()
	reg, err := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := reg.Get(ctx, "axsensor")
	event := types.Event{
		Payload:   &types.Payload{Data: mustHex("804024A4A00DA23800C80C0066FC"), FPort: 2},
		Timestamp: migrationTime,
	}
	legacyPayload, _ := legacyDecoders["axsensor"](ctx, event)
	legacyObjects, _ := legacyConverters["axsensor"](ctx, "dev", legacyPayload, migrationTime)
	genericRes, err := snap.Go(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if genericRes.Battery != nil {
		t.Fatalf("axsensor generic battery must be nil (voltage only), got %d", *genericRes.Battery)
	}
	mapped := mapper.MapToLWM2M("dev", migrationTime, genericRes.Mapping, snap.Groups, snap.Rules)
	packsEqual(t, legacyObjects, mapped.Objects)
}

func TestMigrationEnviot(t *testing.T) {
	obj := []byte(`{"payload":{"battery":86,"humidity":85,"sensorStatus":0,"snowHeight":0,"temperature":11.5}}`)
	runMigrationCase(t, "enviot", types.Event{
		Payload:   &types.Payload{Object: obj, FPort: 1},
		Timestamp: migrationTime,
	})

	// Gated snowHeight: status != 0 keeps storage but drops mapping.
	ctx := context.Background()
	reg, _ := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	snap, _ := reg.Get(ctx, "enviot")
	gatedObj := []byte(`{"payload":{"battery":86,"humidity":85,"sensorStatus":1,"snowHeight":5,"temperature":11.5}}`)
	res, err := snap.Go(ctx, types.Event{Payload: &types.Payload{Object: gatedObj}, Timestamp: migrationTime})
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	b, _ := json.Marshal(res.Data)
	_ = json.Unmarshal(b, &stored)
	if stored["payload"].(map[string]any)["snowHeight"] == nil {
		t.Fatalf("enviot storage must keep gated snowHeight")
	}
	mapped := mapper.MapToLWM2M("dev", migrationTime, res.Mapping, snap.Groups, snap.Rules)
	for _, o := range mapped.Objects {
		if _, ok := o.(lwm2m.Distance); ok {
			t.Fatalf("enviot gated snowHeight must not map when sensorStatus != 0")
		}
	}
}

func TestMigrationNiab(t *testing.T) {
	// Valid reading.
	runMigrationCase(t, "niab-fls", types.Event{
		Payload:   &types.Payload{Object: []byte{0xcc, 0x0f, 0x03, 0xc5}},
		Timestamp: migrationTime,
	})

	// Sentinel distance -1 stays a decoder error.
	ctx := context.Background()
	reg, _ := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	snap, _ := reg.Get(ctx, "niab-fls")
	_, err := snap.Go(ctx, types.Event{Payload: &types.Payload{Object: []byte{0xcc, 0x0f, 0xff, 0xff}}})
	if err == nil {
		t.Fatalf("niab sentinel must stay a decoder error")
	}
}

func TestMigrationSenlabT(t *testing.T) {
	runMigrationCase(t, "senlabt", types.Event{
		Payload:   &types.Payload{Data: mustHex("01FE90619c10006A"), FPort: 3},
		Timestamp: migrationTime,
	})

	ctx := context.Background()
	reg, _ := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	snap, _ := reg.Get(ctx, "senlabt")
	_, err := snap.Go(ctx, types.Event{Payload: &types.Payload{Data: mustHex("01FE90619c10FD14"), FPort: 3}})
	if err == nil {
		t.Fatalf("senlabt sentinel must stay a decoder error")
	}
}

func TestMigrationSensative(t *testing.T) {
	runMigrationCase(t, "sensative", types.Event{
		Payload:   &types.Payload{Data: mustB64("//8VAQ=="), FPort: 1},
		Timestamp: migrationTime,
	})

	// Check-in maps to header-only Device, no measurement.
	ctx := context.Background()
	reg, _ := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	snap, _ := reg.Get(ctx, "sensative")
	res, err := snap.Go(ctx, types.Event{Payload: &types.Payload{FPort: 2}, Timestamp: migrationTime})
	if err != nil {
		t.Fatal(err)
	}
	mapped := mapper.MapToLWM2M("dev", migrationTime, res.Mapping, snap.Groups, snap.Rules)
	if len(mapped.Objects) != 1 {
		t.Fatalf("sensative check-in must produce one header object, got %d", len(mapped.Objects))
	}
}

func TestMigrationVegapuls(t *testing.T) {
	ctx := context.Background()
	reg, err := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := reg.Get(ctx, "vegapuls_air_41")
	event := types.Event{
		Payload:   &types.Payload{Data: mustHex("0200400000002d6000d1af"), FPort: 1},
		Timestamp: migrationTime,
	}
	legacyPayload, _ := legacyDecoders["vegapuls_air_41"](ctx, event)
	legacyObjects, _ := legacyConverters["vegapuls_air_41"](ctx, "dev", legacyPayload, migrationTime)
	genericRes, err := snap.Go(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	// Normalized keys exist only in mapping, raw stays in storage.
	if _, ok := genericRes.Mapping["distance_m"]; !ok {
		t.Fatalf("vegapuls mapping must carry distance_m")
	}
	if _, ok := genericRes.Mapping["temperature_c"]; !ok {
		t.Fatalf("vegapuls mapping must carry temperature_c")
	}
	mapped := mapper.MapToLWM2M("dev", migrationTime, genericRes.Mapping, snap.Groups, snap.Rules)
	packsEqual(t, legacyObjects, mapped.Objects)
}

func TestMigrationX2ClimateStatusFix(t *testing.T) {
	ctx := context.Background()
	reg, err := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := reg.Get(ctx, "x2climate")
	// Real 19-byte payload from existing fixtures.
	event := types.Event{
		Payload:   &types.Payload{Data: mustHex("031101149C1A00000D17002B088C2800000001"), FPort: 2},
		Timestamp: migrationTime,
	}
	// Length is 19? The fixture above is 19 bytes; if not, skip decode check.
	if len(event.Payload.Data) != 19 {
		t.Skipf("fixture length %d, need 19", len(event.Payload.Data))
	}
	res, err := snap.Go(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if res.Battery == nil {
		t.Fatalf("x2climate generic must set SOC battery when voltage valid")
	}
	if *res.Battery < 0 || *res.Battery > 100 {
		t.Fatalf("x2climate SOC out of range: %d", *res.Battery)
	}
	mapped := mapper.MapToLWM2M("dev", migrationTime, res.Mapping, snap.Groups, snap.Rules)
	if len(mapped.Objects) != 3 {
		t.Fatalf("x2climate must map temp+humidity+device, got %d", len(mapped.Objects))
	}
}

func TestMigrationSensefarmStatusAndPressureFix(t *testing.T) {
	ctx := context.Background()
	reg, err := BuildRegistry(ctx, "../../../assets/config/iot-agent.yaml")
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := reg.Get(ctx, "sensefarm")
	event := types.Event{
		Payload:   &types.Payload{Data: mustHex("b006b800013008e4980000032fa80006990000043aa9000a08418a8bcc"), FPort: 2},
		Timestamp: migrationTime,
	}
	res, err := snap.Go(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if res.Battery != nil {
		t.Fatalf("sensefarm generic battery must be nil until SOC curve selected")
	}
	mapped := mapper.MapToLWM2M("dev", migrationTime, res.Mapping, snap.Groups, snap.Rules)
	if len(mapped.Objects) == 0 {
		t.Fatalf("sensefarm must map objects")
	}
	// No truncation: list rules preserve element count.
	legacyPayload, _ := legacyDecoders["sensefarm"](ctx, event)
	legacyObjects, _ := legacyConverters["sensefarm"](ctx, "dev", legacyPayload, migrationTime)
	// Pressure values differ intentionally where legacy overflows int16;
	// object count and order must still match.
	if len(legacyObjects) != len(mapped.Objects) {
		t.Fatalf("sensefarm object count differs: legacy %d mapped %d", len(legacyObjects), len(mapped.Objects))
	}
}
