package endpoints

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cordialsys/panel/pkg/secret"
	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

type SetEncryptionAtRestRequest struct {
	EarSecret secret.Secret `json:"ear_secret"`
}

func (endpoints *Endpoints) attachEarSecretToCmd(cmd *exec.Cmd) error {
	if endpoints.panel.EarSecret == "" {
		return nil
	}
	secretValue, err := endpoints.panel.EarSecret.Load()
	if err != nil {
		return servererrors.BadRequestf("failed to load ear secret: %v", err)
	}
	secretValue = FormatMnemonic(secretValue)
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", ENV_SIGNER_EAR_PHRASE, secretValue))
	return nil
}

// PUT /v1/panel/ear
func (endpoints *Endpoints) SetEncryptionAtRest(c *fiber.Ctx) error {
	var err error
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	ctx := c.Context()

	var req SetEncryptionAtRestRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return servererrors.BadRequestf("failed to parse request: %v", err)
	}

	if req.EarSecret == "" {
		return servererrors.BadRequestf("missing ear_secret")
	}
	if secretType, _ := req.EarSecret.Type(); secretType == secret.File {
		return servererrors.BadRequestf("file type secret is not allowed")
	}

	secret, err := req.EarSecret.Load()
	if err != nil {
		return servererrors.BadRequestf("failed to load ear_secret: %v", err)
	}
	if secret == "" {
		return servererrors.BadRequestf("ear_secret value is empty")
	}

	secret = FormatMnemonic(secret)
	if len(strings.Split(secret, " ")) != 12 {
		return servererrors.BadRequestf("ear_secret must be a valid 12-word bip39 phrase (e.g. from `cord backup bak`)")
	}

	// check if we have an existing ear secret
	existingSecret := ""
	if endpoints.panel.EarSecret != "" {
		existingSecret, err = endpoints.panel.EarSecret.Load()
		if err != nil {
			return servererrors.BadRequestf("failed to load existing ear_secret: %v", err)
		}
		existingSecret = FormatMnemonic(existingSecret)
		if existingSecret == "" {
			return servererrors.BadRequestf("existing ear_secret loaded an empty value")
		}
	}

	signerBin := filepath.Join(endpoints.panel.BinaryDir, "signer")
	if existingSecret == secret {
		// okay, nothing to do
	} else {
		// Stop treasury
		didIssueStop, err := stopSystemdServiceAndWait(ctx, ServiceTreasury)
		if err != nil {
			return err
		}

		localEnv := os.Environ()

		var cmd *exec.Cmd
		if existingSecret != "" {
			cmd = exec.Command(signerBin, "recrypt-in-place", "--db", endpoints.panel.TreasuryHome.SignerDb())
			cmd.Env = localEnv
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", ENV_SIGNER_EAR_PHRASE, existingSecret))
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", ENV_SIGNER_NEW_EAR_PHRASE, secret))
		} else {
			cmd = exec.Command(signerBin, "encrypt-in-place", "--db", endpoints.panel.TreasuryHome.SignerDb())
			cmd.Env = localEnv
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", ENV_SIGNER_NEW_EAR_PHRASE, secret))
		}

		// Run the command
		outputBz, err := cmd.CombinedOutput()
		if err != nil {
			return servererrors.InternalErrorf("failed to run `%s`: %v", cmd.String(), string(outputBz))
		}

		// Save the new ear secret
		endpoints.panel.EarSecret = req.EarSecret
		err = panel.Save(endpoints.panel)
		if err != nil {
			return servererrors.InternalErrorf("failed to save panel: %v", err)
		}

		// restart treasury
		if didIssueStop {
			updateSystemdService(ctx, ServiceTreasury, ServiceActionStart)
		}
	}

	return c.JSON(nil)
}

// DELETE /v1/panel/ear
func (endpoints *Endpoints) DeleteEncryptionAtRest(c *fiber.Ctx) error {
	var err error
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	ctx := c.Context()

	if endpoints.panel.EarSecret == "" {
		return servererrors.BadRequestf("no ear secret to delete")
	}

	existingSecret, err := endpoints.panel.EarSecret.Load()
	if err != nil {
		return servererrors.BadRequestf("failed to load current ear secret: %v", err)
	}
	if existingSecret == "" {
		return servererrors.BadRequestf("current ear secret value is empty")
	}

	signerBin := filepath.Join(endpoints.panel.BinaryDir, "signer")

	// Stop treasury
	didIssueStop, err := stopSystemdServiceAndWait(ctx, ServiceTreasury)
	if err != nil {
		return err
	}

	localEnv := os.Environ()

	var cmd *exec.Cmd
	cmd = exec.Command(signerBin, "decrypt-in-place", "--db", endpoints.panel.TreasuryHome.SignerDb())
	cmd.Env = localEnv
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", ENV_SIGNER_EAR_PHRASE, existingSecret))

	// Run the command
	outputBz, err := cmd.CombinedOutput()
	if err != nil {
		return servererrors.InternalErrorf("failed to run `%s`: %v", cmd.String(), string(outputBz))
	}

	// Remove the ear secret
	endpoints.panel.EarSecret = ""
	err = panel.Save(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to save panel: %v", err)
	}

	// restart treasury
	if didIssueStop {
		updateSystemdService(ctx, ServiceTreasury, ServiceActionStart)
	}

	return c.JSON(nil)
}
