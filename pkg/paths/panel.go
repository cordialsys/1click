package paths

import (
	"os"
	"path/filepath"
)

type PanelHome string

func (p PanelHome) String() string {
	return string(p)
}

func (p PanelHome) PanelFile() string {
	return filepath.Join(string(p), "panel.json")
}

func (p PanelHome) IdentityFile() string {
	return filepath.Join(string(p), "identity.txt")
}

func (p PanelHome) PanelFileExists() bool {
	_, err := os.Stat(p.PanelFile())
	return err == nil
}

func (p PanelHome) EnvFile() string {
	return filepath.Join(string(p), "env")
}

func (p PanelHome) BlueprintFile() string {
	return filepath.Join(string(p), "blueprint.csl")
}

func PanelDir(home string) string {
	return filepath.Join(home, "panel")
}
