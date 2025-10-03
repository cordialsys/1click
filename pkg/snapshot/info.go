package snapshot

import (
	"strconv"
	"strings"
	"time"
)

type Int int

func (i *Int) UnmarshalJSON(data []byte) error {
	var asStr string = string(data)
	asStr = strings.Trim(asStr, "\"")
	asInt, err := strconv.Atoi(asStr)
	*i = Int(asInt)
	return err
}

type Info struct {
	Height           uint64    `json:"height"`
	NodeId           string    `json:"node_id"`
	ValidatorAddress string    `json:"validator_address"`
	Validator        bool      `json:"validator"`
	Bak              string    `json:"bak"`
	Participant      Int       `json:"participant"`
	CreateTime       time.Time `json:"create_time"`
	// Command          []string  `json:"command"`
	// Keyring          []string  `json:"keyring"`
}
