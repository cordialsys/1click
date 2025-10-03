package bak

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"strings"

	"filippo.io/age"
	"github.com/cosmos/btcutil/bech32"
	"github.com/tyler-smith/go-bip39"
)

type SecretKey struct {
	entropy  []byte
	identity *age.X25519Identity
}

func (sk *SecretKey) Words() []string {
	mnemonic, err := bip39.NewMnemonic(sk.entropy)
	if err != nil {
		panic(err)
	}
	return strings.Split(mnemonic, " ")
}

func (sk *SecretKey) Entropy() []byte {
	return sk.entropy
}

func (sk *SecretKey) Recipient() Recipient {
	recipient := sk.identity.Recipient()
	return Recipient{
		recipient,
	}
}

type Recipient struct {
	recipient *age.X25519Recipient
}

func (r *Recipient) String() string {
	return r.recipient.String()
}

func (r *Recipient) PublicKey() []byte {
	ageString := r.recipient.String()
	_, publicKey5bit, err := bech32.Decode(ageString, 64)
	if err != nil {
		panic(err)
	}
	publicKey, err := bech32.ConvertBits(publicKey5bit, 5, 8, true)
	if err != nil {
		panic(err)
	}
	// drop checksum
	return publicKey[:len(publicKey)-1]
}

func (r *Recipient) Encrypt(bz []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	encryptor, err := age.Encrypt(buf, r.recipient)
	if err != nil {
		return nil, err
	}
	encryptor.Write(bz)
	encryptor.Close()

	return buf.Bytes(), nil
}

func IdentityFromBytes(key []byte) (*age.X25519Identity, error) {
	ed255Key := key
	if len(key) == 16 {
		ed255Key = make([]byte, 32)
		copy(ed255Key[16:], key)
	}
	secret5Bit, err := bech32.ConvertBits(ed255Key, 8, 5, true)
	if err != nil {
		return nil, fmt.Errorf("error converting bits: %v", err)
	}
	ageSecret, err := bech32.Encode("AGE-SECRET-KEY-", secret5Bit)
	if err != nil {
		return nil, fmt.Errorf("error encoding: %v", err)
	}

	// Age wants uppercase bech32
	ageSecret = strings.ToUpper(ageSecret)

	identity, err := age.ParseX25519Identity(ageSecret)
	if err != nil {
		return nil, fmt.Errorf("error producing identity: %v", err)
	}
	return identity, nil
}

func NewEncryptionKeyFromEntropy(randomBz []byte) (*SecretKey, error) {
	entropy := randomBz
	identity, err := IdentityFromBytes(entropy)
	if err != nil {
		return nil, err
	}

	return &SecretKey{entropy, identity}, nil
}

// The upstream bip39 package does not actually produce a way to reverse the mnemonic,
// so we define our own here.
func MnemonicToEntropy(words []string) ([]byte, error) {
	// Provider enough space for the longest possible word list
	entropy := make([]byte, 33)
	cursor := 0
	offset := 0
	remainder := uint32(0)
	for _, word := range words {
		index, ok := bip39.GetWordIndex(word)
		if !ok {
			return nil, fmt.Errorf("invalid word: %s", word)
		}
		remainder |= (uint32(index) << (32 - 11)) >> offset
		offset += 11
		for offset >= 8 {
			entropy[cursor] = uint8(remainder >> 24)
			cursor += 1
			remainder <<= 8
			offset -= 8
		}
	}
	if offset != 0 {
		entropy[cursor] = uint8(remainder >> 24)
	}
	entropy_bytes := (len(words) / 3) * 4
	return entropy[:entropy_bytes], nil
}

func NewEncryptionKey(words []string) (*SecretKey, error) {
	if len(words) != 12 && len(words) != 24 {
		return nil, fmt.Errorf("mnemonic must be 12 or 24 words")
	}

	entropy, err := MnemonicToEntropy(words)
	if err != nil {
		return nil, err
	}

	return NewEncryptionKeyFromEntropy(entropy)
}

func GenerateEncryptionKey() *SecretKey {
	// 16 bytes == 128 bits == 12 words
	randomBz := make([]byte, 16)
	_, err := rand.Read(randomBz)
	if err != nil {
		panic(err)
	}
	words, err := bip39.NewMnemonic(randomBz)
	if err != nil {
		panic(err)
	}
	enc, err := NewEncryptionKey(strings.Split(words, " "))
	if err != nil {
		panic(err)
	}
	return enc
}
