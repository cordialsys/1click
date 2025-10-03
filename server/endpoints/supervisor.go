package endpoints

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
	"github.com/pelletier/go-toml/v2"
)

type SupervisorImage struct {
	Image     string `json:"image"`
	Overwrite bool   `json:"overwrite,omitempty"`
}

// Executes a cord command, adding the --home flag to the command
func execSupervisorWithHome(params *panel.Panel, cmd []string) error {
	binaryDir := params.BinaryDir
	cord := filepath.Join(binaryDir, "cord")

	execList := append(cmd, "--supervisor-home", string(params.SupervisorHome))

	execCmd := exec.Command(cord, execList...)
	bz, err := execCmd.CombinedOutput()
	slog.Info("exec", "binary", cord, "cmd", execCmd.String(), "output", string(bz))

	if err != nil {
		return fmt.Errorf("failed to run `%s`: %v", execCmd.String(), string(bz))
	}
	return nil
}

func readSupervisorImage(params *panel.Panel) (string, error) {
	supervisorHome := params.SupervisorHome
	imageFile := filepath.Join(supervisorHome.ConfigFile())
	supervisorConfigBz, err := os.ReadFile(imageFile)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %v", err)
	}
	var config struct {
		Image string `toml:"image"`
	}
	err1 := toml.Unmarshal(supervisorConfigBz, &config)
	if err1 != nil {
		return "", fmt.Errorf("failed to unmarshal supervisor config: %v", err1)
	}
	return config.Image, nil
}

func (endpoints *Endpoints) PostSupervisorImage(c *fiber.Ctx) error {
	var req SupervisorImage
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return servererrors.BadRequestf("failed to parse request: %v", err)
	}
	if req.Image == "" {
		return servererrors.BadRequestf("image is required")
	}

	args := []string{
		"supervise",
		"use-image",
		req.Image,
	}
	if req.Overwrite {
		args = append(args, "--overwrite")
	}

	err := execSupervisorWithHome(endpoints.panel, args)
	if err != nil {
		return servererrors.InternalErrorf("failed to use image: %v", err)
	}
	image, err := readSupervisorImage(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to read supervisor image: %v", err)
	}
	return c.JSON(image)
}

func (endpoints *Endpoints) GetSupervisorImage(c *fiber.Ctx) error {
	image, err := readSupervisorImage(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to read supervisor image: %v", err)
	}
	return c.JSON(image)
}

func (endpoints *Endpoints) DeleteSupervisorImage(c *fiber.Ctx) error {
	imageFile := filepath.Join(endpoints.panel.SupervisorHome.ConfigFile())
	_ = os.Remove(imageFile)
	return c.SendStatus(fiber.StatusOK)
}
