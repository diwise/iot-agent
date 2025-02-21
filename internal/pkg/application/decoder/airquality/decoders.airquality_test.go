package airquality

import (
	"context"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/matryer/is"
)

func TestAirQuality(t *testing.T) {
	is := is.New(t)
	ue, _ := application.ChirpStack([]byte(testData))

	objects, err := Decoder(context.Background(), "id", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "id")
}

const testData string = ``
