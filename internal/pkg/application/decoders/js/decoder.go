package js

import (
	"context"
	"fmt"
	"io"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/dop251/goja"
)

// This is a way to execute JavaScript decoders for payloads from The Things Network
// as of now only for test purposes.

func Decode(ctx context.Context, js io.Reader, e types.Event) (map[string]any, error) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	b, err := io.ReadAll(js)
	if err != nil {
		return nil, err
	}

	_, err = vm.RunScript("script", wrapScript(string(b)))
	if err != nil {
		return nil, err
	}

	entrypoint, ok := goja.AssertFunction(vm.Get("main"))
	if !ok {
		return nil, fmt.Errorf("entrypoint not found")
	}

	input := struct {
		Bytes    []uint8 `json:"bytes"`
		FPort    uint8   `json:"fPort"`
		RecvTime int64   `json:"recvTime"`
	}{
		Bytes:    e.Payload.Data,
		FPort:    uint8(e.Payload.FPort),
		RecvTime: e.Timestamp.Unix(),
	}

	res, err := entrypoint(goja.Undefined(), vm.ToValue(input))
	if err != nil {
		return nil, err
	}
	export := res.Export()
	output := export.(map[string]any)["decoded"].(map[string]any)["data"].(map[string]any)

	return output, nil
}

func wrapScript(script string) string {
	return fmt.Sprintf(`
		%s

		function main(input) {
			const bytes = input.bytes.slice();
			const { fPort, recvTime } = input;

			const jsDate = new Date(Number(BigInt(recvTime) / 1000000n));
			const decoded = decodeUplink({ bytes, fPort, recvTime: jsDate });
			return {
				decoded
			};
		}
	`, script)
}
