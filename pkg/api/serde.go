package api

import (
	"encoding/base64"
	"encoding/hex"
)

func AsStdBase64(bz []byte) *string {
	if len(bz) > 0 {
		s := base64.StdEncoding.EncodeToString(bz)
		return &s
	}
	return nil
}

func AsHex(bz []byte) *string {
	if len(bz) > 0 {
		s := hex.EncodeToString(bz)
		return &s
	}
	return nil
}
func As[T any](a T) *T {
	return &a
}

func IfNotEmpty[T any](a []T) *[]T {
	if len(a) > 0 {
		return &a
	}
	return nil
}

func DerefOrZero[T any](a *T) T {
	if a == nil {
		var aZero T
		return aZero
	}
	return *a
}

func IfNotZero[T comparable](a *T) *T {
	var aZero T
	if a != nil && *a != aZero {
		return a
	}
	return nil
}

func IfValueNotZero[T comparable](a T) *T {
	return IfNotZero(&a)
}
