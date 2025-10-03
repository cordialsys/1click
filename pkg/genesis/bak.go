package genesis

import (
	"encoding/json"
)

type Bak struct {
	// this is the age string
	Key string `json:"key" toml:"key"`
	// Optional
	Id string `json:"id,omitempty" toml:"id,omitempty"`
}

type BakArray []Bak

func NewBakArrayFromStrings(bakKeys ...string) BakArray {
	baks := make([]Bak, len(bakKeys))
	for i := range bakKeys {
		baks[i] = Bak{Key: bakKeys[i]}
	}
	return baks
}

// Custom UnmarshalJSON to support both string and array deserialization
func (b *BakArray) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a string first
	var key string
	if err := json.Unmarshal(data, &key); err == nil {
		*b = BakArray{
			Bak{
				Key: key,
				Id:  "",
			},
		}
		return nil
	}

	// If it's not a string, try to unmarshal as an object
	var aux []Bak
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*b = aux
	return nil
}
