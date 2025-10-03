package names

import (
	"strconv"
	"strings"
)

type NodeName string

func (n NodeName) Id() string {
	parts := strings.Split(string(n), "/")
	return parts[len(parts)-1]
}

func (n NodeName) Participant() uint64 {
	id, err := strconv.ParseUint(n.Id(), 10, 64)
	_ = err
	return id
}
