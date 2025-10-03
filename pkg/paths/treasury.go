package paths

import (
	"path/filepath"
)

// Paths are relative to TREASURY_HOME
const GenesisRelPath = "config/genesis.json"
const ImportGenesisRelPath = "config/import.json"
const ImportGenesisStateRelPath = "config/state.jsonl"

const DataRelPath = "data"
const DataBakRelPath = "data.bak"
const DataPrivValidatorStateRelPath = "data/priv_validator_state.json"

type TreasuryHome string

func (home TreasuryHome) Genesis() string {
	return filepath.Join(string(home), GenesisRelPath)
}
func (home TreasuryHome) ImportGenesis() string {
	return filepath.Join(string(home), ImportGenesisRelPath)
}

func (home TreasuryHome) ImportGenesisState() string {
	return filepath.Join(string(home), ImportGenesisStateRelPath)
}

func (home TreasuryHome) Data() string {
	return filepath.Join(string(home), DataRelPath)
}

func (home TreasuryHome) DataBackup() string {
	return filepath.Join(string(home), DataBakRelPath)
}

func (home TreasuryHome) SnapshotDir() string {
	return filepath.Join(string(home), DataRelPath, "snapshots")
}

func (home TreasuryHome) PrivValidatorState() string {
	return filepath.Join(string(home), DataPrivValidatorStateRelPath)
}

func (home TreasuryHome) SignerDb() string {
	return filepath.Join(string(home), "signer.db")
}

func (home TreasuryHome) CometConfig() string {
	return filepath.Join(string(home), "config", "config.toml")
}

func (home TreasuryHome) PrivValidatorKey() string {
	return filepath.Join(string(home), "config", "priv_validator_key.json")
}

func (home TreasuryHome) NodeKey() string {
	return filepath.Join(string(home), "config", "node_key.json")
}

func (home TreasuryHome) TriplesDir() string {
	return filepath.Join(string(home), "triples")
}

func (home TreasuryHome) DefaultBackupDir() string {
	return filepath.Join(string(home), "backups")
}

func (home TreasuryHome) ApplicationDb() string {
	return filepath.Join(string(home), DataRelPath, "application.db")
}

func (home TreasuryHome) StateDb() string {
	return filepath.Join(string(home), DataRelPath, "state.db")
}

func (home TreasuryHome) BootedVersionFile() string {
	return filepath.Join(string(home), "config", "VERSION")
}

func (home TreasuryHome) UpdateStateFile() string {
	return filepath.Join(string(home), DataRelPath, "UPDATE-STATUS")
}

func (home TreasuryHome) TreasuryConfig() string {
	return filepath.Join(string(home), "treasury.toml")
}
