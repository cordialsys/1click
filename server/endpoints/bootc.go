package endpoints

import (
	"log/slog"
	"os/exec"

	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

// Exec a command and return the stdout
func execCmd(program string, execList []string) (string, error) {
	cmd := exec.Command(program, execList...)
	// stderrBuf := bytes.NewBuffer([]byte{})
	// cmd.Stderr = stderrBuf
	bz, err := cmd.CombinedOutput()
	// get stderr for logging only
	// stderr, _ := io.ReadAll(stderrBuf)
	slog.Info("exec", "binary", program, "cmd", cmd.String(), "output", string(bz))
	return string(bz), err
}

func (endpoints *Endpoints) BootcStatus(c *fiber.Ctx) error {
	formatMaybe := c.Query("format")
	args := []string{"status"}
	if formatMaybe != "" {
		args = append(args, "--format", formatMaybe)
	}
	output, err := execCmd("bootc", args)
	if err != nil {
		// return servererrors.InternalErrorf("failed to get bootc status: %v", err)
		c.Status(fiber.StatusInternalServerError)
	}

	_, _ = c.WriteString(string(output))
	return nil
}

func (endpoints *Endpoints) BootcCheck(c *fiber.Ctx) error {
	// Check if an update is available, but don't stage it.
	output, err := execCmd("bootc", []string{"upgrade", "--check"})
	if err != nil {
		return servererrors.InternalErrorf("failed to get bootc status: %v: %s", err, string(output))
	}

	return c.SendString(string(output))
}

func (endpoints *Endpoints) BootcStage(c *fiber.Ctx) error {
	// This stages the VM for an update on next boot
	// The new image will be downloaded and verified.
	output, err := execCmd("bootc", []string{"upgrade"})
	if err != nil {
		return servererrors.InternalErrorf("failed to get bootc status: %v: %s", err, string(output))
	}

	return c.SendString(string(output))
}

func (endpoints *Endpoints) BootcUpgradeApply(c *fiber.Ctx) error {
	// Same as with stage, but will reboot the VM.
	output, err := execCmd("bootc", []string{"upgrade", "--apply"})
	if err != nil {
		return servererrors.InternalErrorf("failed to get bootc status: %v: %s", err, string(output))
	}

	return c.SendString(string(output))
}

func (endpoints *Endpoints) BootcRollbackStage(c *fiber.Ctx) error {
	// This stages the VM to rollback to previous image.
	// It will switch on the next boot.
	output, err := execCmd("bootc", []string{"rollback"})
	if err != nil {
		return servererrors.InternalErrorf("failed to get bootc status: %v: %s", err, string(output))
	}

	return c.SendString(string(output))
}

func (endpoints *Endpoints) BootcRollbackApply(c *fiber.Ctx) error {
	// This stages the VM to rollback to previous image.
	// It will switch on the next boot.
	output, err := execCmd("bootc", []string{"rollback", "--apply"})
	if err != nil {
		return servererrors.InternalErrorf("failed to get bootc status: %v: %s", err, string(output))
	}

	return c.SendString(string(output))
}
