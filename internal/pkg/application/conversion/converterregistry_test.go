package conversion

import (
	"testing"

	"github.com/matryer/is"
)

func TestThatConverterRegistryReturnsOnlyConvertersThatMatchType(t *testing.T) {
	is, conReg := testSetup(t)

	mcs := conReg.DesignateConverters(nil, []string{"temperature", "humidity"})
	is.Equal(len(mcs), 1)
}

func TestThatConverterRegistryReturnsEmptyIfNoTypesMatch(t *testing.T) {
	is, conReg := testSetup(t)

	mcs := conReg.DesignateConverters(nil, []string{"co2", "humidity"})
	is.Equal(len(mcs), 0)
}

func testSetup(t *testing.T) (*is.I, ConverterRegistry) {
	is := is.New(t)

	conReg := NewConverterRegistry()

	return is, conReg
}