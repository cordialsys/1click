package panel_test

import (
	"bytes"
	"testing"

	"github.com/cordialsys/panel/server/panel"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
)

// BackupConfig represents the backup configuration with Bak entries
type BackupConfig struct {
	Backup struct {
		Bak []panel.Bak `toml:"bak"`
	} `toml:"backup"`
}

func TestBakTomlSerialization(t *testing.T) {
	// Create test data with backup keys
	config := BackupConfig{}
	config.Backup.Bak = []panel.Bak{
		{
			Id:  "hot",
			Key: "age15sr33c4jrdm367u7hmdekkz4xlmehalcqcflzrpr9ndp9mv6tyhs92qgw6",
		},
		{
			Id:  "cold",
			Key: "age1y6cj8x6934lckm3ljzyg3c3z9kldvv2af6sk7u0szew3r7a03c3szzwk33",
		},
	}

	// Serialize to TOML
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	err := encoder.Encode(config)
	assert.NoError(t, err)

	// Check the expected TOML format is compatible with treasury backup config
	// https://docs.cordialsystems.com/reference/config#backup
	expected := `[backup]
[[backup.bak]]
id = 'hot'
key = 'age15sr33c4jrdm367u7hmdekkz4xlmehalcqcflzrpr9ndp9mv6tyhs92qgw6'

[[backup.bak]]
id = 'cold'
key = 'age1y6cj8x6934lckm3ljzyg3c3z9kldvv2af6sk7u0szew3r7a03c3szzwk33'
`

	actual := buf.String()
	assert.Equal(t, expected, actual, "TOML serialization should match expected format")

	// Also test deserialization to ensure round-trip works
	var decoded BackupConfig
	err = toml.Unmarshal([]byte(expected), &decoded)
	assert.NoError(t, err)
	assert.Equal(t, config.Backup.Bak, decoded.Backup.Bak, "Deserialized data should match original")
}
