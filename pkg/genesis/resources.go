package genesis

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// API representation of resources needed during initialization
// some fields that are not needed are omitted

type Treasury struct {
	Name     string `json:"name,omitempty"`
	Network  string `json:"network,omitempty"`
	Software string `json:"software,omitempty"`
}

var _ json.Unmarshaler = (*Validator)(nil)

func (_s *Treasury) UnmarshalJSON(data []byte) error {
	type Alias Treasury
	type legacyFields struct {
		*Alias
		Id string `json:"id,omitempty"`
	}
	var withLegacy = legacyFields{Alias: &Alias{}}
	if err := json.Unmarshal(data, &withLegacy); err != nil {
		return err
	}

	if withLegacy.Id != "" && withLegacy.Name == "" {
		withLegacy.Name = "treasuries/" + strings.TrimPrefix(withLegacy.Id, "treasuries/")
	}

	*_s = Treasury(*withLegacy.Alias)

	return nil
}

type Validator struct {
	// should be validators/{id}
	Name string `json:"name"`
	// Public key is in hex
	PublicKey string `json:"public_key"`
}

var _ json.Unmarshaler = (*Validator)(nil)

func (_s *Validator) UnmarshalJSON(data []byte) error {
	type Alias Validator
	type legacyFields struct {
		*Alias
		Id string `json:"id,omitempty"`
	}
	var withLegacy = legacyFields{Alias: &Alias{}}
	if err := json.Unmarshal(data, &withLegacy); err != nil {
		return err
	}

	if withLegacy.Id != "" && withLegacy.Name == "" {
		withLegacy.Name = "validators/" + strings.TrimPrefix(withLegacy.Id, "validators/")
	}

	// convert base64 to hex
	if _, err := hex.DecodeString(withLegacy.PublicKey); err != nil {
		if fromb64, err := base64.StdEncoding.DecodeString(withLegacy.PublicKey); err == nil {
			withLegacy.PublicKey = hex.EncodeToString(fromb64)
		}
	}
	*_s = Validator(*withLegacy.Alias)

	return nil
}

// Signer defines model for Signer.
type Signer struct {
	Name string `json:"name,omitempty"`

	// Hex encoded.
	Recipient string `json:"recipient,omitempty"`
	// Hex encoded.
	VerifyingKey string `json:"verifying_key,omitempty"`
	Socket       string `json:"socket,omitempty"`

	State string `json:"state,omitempty"`

	User string `json:"user,omitempty"`
}

var _ json.Unmarshaler = (*Signer)(nil)

func (_s *Signer) UnmarshalJSON(data []byte) error {
	type Alias Signer
	type legacyFields struct {
		*Alias
		ReceivingKey string `json:"receiving_key,omitempty"`
		Id           string `json:"id,omitempty"`
	}
	var withLegacy = legacyFields{Alias: &Alias{}}
	if err := json.Unmarshal(data, &withLegacy); err != nil {
		return err
	}

	if withLegacy.Id != "" && withLegacy.Name == "" {
		withLegacy.Name = "signers/" + strings.TrimPrefix(withLegacy.Id, "signers/")
	}

	if withLegacy.ReceivingKey != "" && withLegacy.Recipient == "" {
		withLegacy.Recipient = withLegacy.ReceivingKey
	}

	// convert base64 to hex
	if _, err := hex.DecodeString(withLegacy.Recipient); err != nil {
		if fromb64, err := base64.StdEncoding.DecodeString(withLegacy.Recipient); err == nil {
			withLegacy.Recipient = hex.EncodeToString(fromb64)
		}
	}

	// convert base64 to hex
	if _, err := hex.DecodeString(withLegacy.VerifyingKey); err != nil {
		if fromb64, err := base64.StdEncoding.DecodeString(withLegacy.VerifyingKey); err == nil {
			withLegacy.VerifyingKey = hex.EncodeToString(fromb64)
		}
	}

	*_s = Signer(*withLegacy.Alias)

	return nil
}
