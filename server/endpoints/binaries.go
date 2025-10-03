package endpoints

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

type BinaryVersions struct {
	Cord        string `json:"cord"`
	Signer      string `json:"signer"`
	TreasuryCLI string `json:"treasury_cli"`
}

func getVersion(params *panel.Panel, binaryName string, unzippedName string) (string, error) {
	binaryDir := params.BinaryDir

	if unzippedName == "" {
		unzippedName = binaryName
	}

	cmd := exec.Command(filepath.Join(binaryDir, unzippedName), "version")
	bz, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to read %s version: %v", binaryName, err)
	}
	return strings.TrimSpace(string(bz)), nil
}

func (endpoints *Endpoints) GetBinaryVersion(c *fiber.Ctx) error {
	binaryName := c.Params("binary")
	unzippedName := ""
	switch binaryName {
	case "signer":
		// ok
	case "cord":
		// ok
	case "treasury-cli", "treasury_cli":
		// ok
		unzippedName = "treasury"
	default:
		return servererrors.BadRequestf("unknown binary: %s", binaryName)
	}

	version, err := getVersion(endpoints.panel, binaryName, unzippedName)
	if err != nil {
		return servererrors.InternalErrorf("failed to get %s version: %v", binaryName, err)
	}

	return c.JSON(version)
}

func (endpoints *Endpoints) GetBinaryVersions(c *fiber.Ctx) error {
	// cord
	signerVersion, err := getVersion(endpoints.panel, "signer", "")
	if err != nil {
		return servererrors.InternalErrorf("failed to get signer version: %v", err)
	}

	cordVersion, err := getVersion(endpoints.panel, "cord", "")
	if err != nil {
		return servererrors.InternalErrorf("failed to get cord version: %v", err)
	}

	treasuryCLI, err := getVersion(endpoints.panel, "treasury", "")
	if err != nil {
		return servererrors.InternalErrorf("failed to get treasury-cli version: %v", err)
	}

	return c.JSON((&BinaryVersions{
		Cord:        cordVersion,
		Signer:      signerVersion,
		TreasuryCLI: treasuryCLI,
	}))
}
