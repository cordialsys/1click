package genesis_test

import (
	"encoding/json"
	"testing"

	"github.com/cordialsys/panel/pkg/genesis"
	"github.com/stretchr/testify/require"
)

func TestBakArrayUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected genesis.BakArray
		err      string
	}{
		{
			name:     "Deserialize from string",
			input:    `"age1example_key"`,
			expected: genesis.BakArray{{Key: "age1example_key", Id: ""}},
		},
		{
			name:  "Deserialize from array of objects",
			input: `[{"key": "key1", "id": "id1"}, {"key": "key2", "id": "id2"}]`,
			expected: genesis.BakArray{
				{Key: "key1", Id: "id1"},
				{Key: "key2", Id: "id2"},
			},
		},
		{
			name:     "Invalid JSON",
			input:    `{"key": "key1", "id": "id1"}`, // This is not an array or string
			expected: nil,
			err:      "cannot unmarshal object",
		},
		{
			name:     "Empty array",
			input:    `[]`,
			expected: genesis.BakArray{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result genesis.BakArray
			err := json.Unmarshal([]byte(tt.input), &result)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, tt.expected, result)
			}

		})
	}
}
