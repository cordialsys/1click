package resource

import (
	"strings"
	"unicode"
)

const KeyNamePrefix = "keys"
const RoleNamePrefix = "roles"

type KeyName string
type RoleName string

func NewKeyName(id string) KeyName {
	id = strings.TrimPrefix(id, KeyNamePrefix+"/")
	return KeyName(KeyNamePrefix + "/" + id)
}

func (n KeyName) Id() string {
	return strings.TrimPrefix(string(n), KeyNamePrefix+"/")
}

func NewRoleName(id string) RoleName {
	id = strings.TrimPrefix(id, RoleNamePrefix+"/")
	return RoleName(RoleNamePrefix + "/" + id)
}

type Key struct {
	Name      KeyName `json:"name"`
	Algorithm string  `json:"algorithm"`
	// public key hex
	Key    string `json:"key"`
	Format string `json:"format"`
	State  string `json:"state"`
}

func NormalizeId(id string) string {
	// remove leading + trailing whitespace
	id = strings.TrimSpace(id)

	// drop everything else that is not valid
	var sb strings.Builder
	for _, c := range id {
		switch c {
		case '.', ':', ',', ';', '!', '?', '-':
			sb.WriteRune('-')
		case ' ', '\n', '\r', '\t':
			sb.WriteRune('_')
		default:
			if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' {
				sb.WriteRune(c)
			} else {
				sb.WriteRune('-')
			}
		}

	}

	return sb.String()
}
