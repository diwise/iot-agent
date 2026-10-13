package js

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/matryer/is"
)

const ttnHappy = `
function decodeUplink(input) {
    return {
        "data": {"temperature": 21.5, "error": "just data"},
        "warnings": ["w1"]
    };
}`

const ttnErrorEnvelope = `
function decodeUplink(input) {
    return {
        "data": {},
        "errors": ["boom"]
    };
}`

const ttnTime = `
function decodeUplink(input) {
    return {
        "data": {"millis": input.recvTime.getTime()}
    };
}`

const ttnStateful = `
var count = 0;
function decodeUplink(input) {
    count = count + 1;
    return {"data": {"count": count}};
}`

const ttnMutate = `
function decodeUplink(input) {
    input.bytes.push(99);
    return {"data": {"len": input.bytes.length}};
}`

const ttnLoop = `
function decodeUplink(input) {
    while (true) {}
}`

const chirpstackFlat = `
function Decode(fPort, bytes) {
    return {"temperature": bytes[0]};
}`

func TestCompileRejects(t *testing.T) {
	is := is.New(t)

	_, err := Compile("s", "ref", "var x = ;", EntryTTN)
	is.True(err != nil)

	_, err = Compile("s", "ref", ttnHappy, "bogus-style")
	is.True(err != nil)

	_, err = Compile("s", "ref", "function nope() { return {}; }", EntryTTN)
	is.True(err != nil)
}

func TestDecodeTTN(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnHappy, EntryTTN)
	is.NoErr(err)

	payload, warnings, err := d.Decode(t.Context(), JSInput{Bytes: []byte{1, 2}, FPort: 5, RecvTime: time.Now().UTC()})
	is.NoErr(err)
	is.Equal(payload["temperature"], 21.5)
	// A raw error field is plain data.
	is.Equal(payload["error"], "just data")
	is.Equal(warnings, []string{"w1"})
}

func TestDecodeTTNErrors(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnErrorEnvelope, EntryTTN)
	is.NoErr(err)

	_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "boom"))
}

func TestDecodeRecvTime(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnTime, EntryTTN)
	is.NoErr(err)

	ts := time.Date(2024, 2, 28, 11, 21, 59, 0, time.UTC)
	payload, _, err := d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: ts})
	is.NoErr(err)
	millis, err := types.AsFloat(payload["millis"])
	is.NoErr(err)
	is.Equal(millis, float64(ts.UnixMilli()))
}

func TestDecodeIsolatesState(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnStateful, EntryTTN)
	is.NoErr(err)
	ctx := t.Context()
	in := JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()}

	first, _, err := d.Decode(ctx, in)
	is.NoErr(err)
	second, _, err := d.Decode(ctx, in)
	is.NoErr(err)
	// Fresh runtimes never share globals.
	firstCount, err := types.AsFloat(first["count"])
	is.NoErr(err)
	secondCount, err := types.AsFloat(second["count"])
	is.NoErr(err)
	is.Equal(firstCount, 1.0)
	is.Equal(secondCount, 1.0)
}

func TestDecodeCopiesInput(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnMutate, EntryTTN)
	is.NoErr(err)

	raw := []byte{1, 2}
	payload, _, err := d.Decode(t.Context(), JSInput{Bytes: raw, FPort: 5, RecvTime: time.Now().UTC()})
	is.NoErr(err)
	length, err := types.AsFloat(payload["len"])
	is.NoErr(err)
	is.Equal(length, 3.0)
	is.Equal(len(raw), 2)
}

func TestDecodeTimeout(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnLoop, EntryTTN)
	is.NoErr(err)
	d.Timeout = 200 * time.Millisecond

	_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "timeout"))
}

func TestDecodeCancellation(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnLoop, EntryTTN)
	is.NoErr(err)
	d.Timeout = 30 * time.Second

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err = d.Decode(ctx, JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
	is.True(errors.Is(err, context.Canceled))
}

func TestDecodeFPortRange(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnHappy, EntryTTN)
	is.NoErr(err)

	for _, fport := range []int{-1, 256, 1000} {
		_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: fport, RecvTime: time.Now().UTC()})
		is.True(err != nil)
		is.True(errors.Is(err, types.ErrInvalidFPort))
	}
}

func TestDecodeChirpStack(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", chirpstackFlat, EntryChirpStack)
	is.NoErr(err)

	payload, _, err := d.Decode(t.Context(), JSInput{Bytes: []byte{42}, FPort: 5, RecvTime: time.Now().UTC()})
	is.NoErr(err)
	temp, err := types.AsFloat(payload["temperature"])
	is.NoErr(err)
	is.Equal(temp, 42.0)
}

func TestDecodeBadResults(t *testing.T) {
	is := is.New(t)

	for _, script := range []string{
		`function decodeUplink(input) { return {data: null}; }`,
		`function decodeUplink(input) { return {data: [1, 2]}; }`,
		`function decodeUplink(input) { return {data: 42}; }`,
		`function decodeUplink(input) { return {}; }`,
	} {
		d, err := Compile("s", "ref", script, EntryTTN)
		is.NoErr(err)
		_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
		is.True(err != nil)
	}

	d, err := Compile("s", "ref", `function Decode(fPort, bytes) { return [1]; }`, EntryChirpStack)
	is.NoErr(err)
	_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
}

func TestCompileRejectsOversizeScript(t *testing.T) {
	is := is.New(t)

	big := make([]byte, MaxScriptBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	_, err := Compile("s", "ref", string(big), EntryTTN)
	is.True(err != nil)
}

func TestDecodeRejectsOversizeInput(t *testing.T) {
	is := is.New(t)

	d, err := Compile("s", "ref", ttnHappy, EntryTTN)
	is.NoErr(err)

	big := make([]byte, MaxInputBytes+1)
	_, _, err = d.Decode(t.Context(), JSInput{Bytes: big, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
}

func TestDecodeRejectsDeepExport(t *testing.T) {
	is := is.New(t)

	// 12 nested maps exceed MaxExportDepth 10.
	script := `function decodeUplink(input) { var o = {a: 1}; for (var i = 0; i < 12; i++) { o = {nest: o}; } return {data: o}; }`
	d, err := Compile("s", "ref", script, EntryTTN)
	is.NoErr(err)
	_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
}

func TestDecodeRejectsLongList(t *testing.T) {
	is := is.New(t)

	script := `function decodeUplink(input) { var a = []; for (var i = 0; i < 300; i++) { a.push(i); } return {data: {list: a}}; }`
	d, err := Compile("s", "ref", script, EntryTTN)
	is.NoErr(err)
	_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
}

func TestDecodeRejectsNonFinite(t *testing.T) {
	is := is.New(t)

	for _, script := range []string{
		`function decodeUplink(input) { return {data: {v: NaN}}; }`,
		`function decodeUplink(input) { return {data: {v: Infinity}}; }`,
	} {
		d, err := Compile("s", "ref", script, EntryTTN)
		is.NoErr(err)
		_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
		is.True(err != nil)
	}
}

func TestDecodeRejectsNonStringTTNErrors(t *testing.T) {
	is := is.New(t)

	script := `function decodeUplink(input) { return {data: {}, errors: [42]}; }`
	d, err := Compile("s", "ref", script, EntryTTN)
	is.NoErr(err)
	_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)

	badEnvelope := `function decodeUplink(input) { return {data: {}, errors: "boom"}; }`
	d, err = Compile("s", "ref", badEnvelope, EntryTTN)
	is.NoErr(err)
	_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
}

func TestDecodeQueueFullFailsFast(t *testing.T) {
	is := is.New(t)

	script := `function decodeUplink(input) { var x = 0; for (var i = 0; i < 1000000; i++) { x += i; } return {data: {v: x}}; }`
	d, err := Compile("s", "ref", script, EntryTTN)
	is.NoErr(err)
	d.MaxConcurrent = 1
	d.sem = make(chan struct{}, 1)
	d.MaxQueued = 0

	_, _, err = d.Decode(t.Context(), JSInput{Bytes: []byte{1}, FPort: 5, RecvTime: time.Now().UTC()})
	is.True(err != nil)
}
