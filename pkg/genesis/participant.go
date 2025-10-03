package genesis

import (
	"strconv"
	"strings"
)

type Uint64 uint

func (b *Uint64) UnmarshalJSON(data []byte) error {
	var asStr string = string(data)

	// drop quotes
	asStr = strings.Trim(asStr, "\"")

	asInt, err := strconv.Atoi(asStr)
	*b = Uint64(asInt)
	return err
}
func (b *Uint64) UnmarshalText(data []byte) error {
	var asStr string = string(data)

	// drop quotes
	asStr = strings.Trim(asStr, "\"")

	asInt, err := strconv.Atoi(asStr)
	*b = Uint64(asInt)
	return err
}
