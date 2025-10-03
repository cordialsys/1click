package nonce_test

import (
	"testing"

	"github.com/cordialsys/panel/pkg/nonce"
	"github.com/stretchr/testify/require"
)

func TestId(t *testing.T) {
	dups := map[string]struct{}{}
	for i := 0; i < 512; i++ {
		id := nonce.Random()
		require.EqualValues(t, nonce.NonceLength, len(id))

		_, ok := dups[string(id)]
		require.False(t, ok)
		dups[string(id)] = struct{}{}
	}
	id := nonce.NewFromSeed([]byte{1, 2, 34})
	require.EqualValues(t, nonce.NonceLength, len(id))

	_, ok := dups[string(id)]
	require.False(t, ok)
}
