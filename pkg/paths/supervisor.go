package paths

import (
	"os"
	"path/filepath"
)

type SupervisorHome string

func (p SupervisorHome) String() string {
	return string(p)
}

func (p SupervisorHome) ConfigFile() string {
	return filepath.Join(string(p), "supervisor.toml")
}

func (p SupervisorHome) ConfigExists() bool {
	_, err := os.Stat(p.ConfigFile())
	return err == nil
}
