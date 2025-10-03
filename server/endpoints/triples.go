package endpoints

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

// This endpoint is a bit hacky has `signer count-triples` is not intended to be machine-readable.
func (endpoints *Endpoints) GetTriplesCount(c *fiber.Ctx) error {
	signerDb := endpoints.panel.TreasuryHome.SignerDb()
	signer := filepath.Join(endpoints.panel.BinaryDir, "signer")

	_wantsJson := c.Query("json", "none")
	wantsJson := _wantsJson != "none"

	cmd := exec.Command(signer, "count-triples", "--db", signerDb)
	bz, err := cmd.CombinedOutput()
	if err != nil {
		return servererrors.InternalErrorf("failed to count triples (%v): %s", err, string(bz))
	}

	if wantsJson {
		// output looks like this:
		// curve k256 threshold 1: 1000 of 1000 triples available (500 signatures)
		everything := string(bz)
		lines := strings.Split(everything, "\n")
		output := map[string]interface{}{}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			parts := strings.Split(line, " ")
			// try to future proof
			for i, part := range parts {
				parts[i] = strings.TrimSpace(part)
				parts[i] = strings.Trim(part, ":")
				parts[i] = strings.Trim(part, "=")
				parts[i] = strings.TrimSpace(part)
				parts[i] = strings.ToLower(part)
			}

			var curve string
			var countStr string
			for i, part := range parts {
				if part == "curve" {
					if i+1 < len(parts) {
						curve = parts[i+1]
					}
				}
			}
			for i, part := range parts {
				if part == "alg" || part == "algorithm" {
					if i+1 < len(parts) {
						curve = parts[i+1]
					}
				}
			}
			for i, part := range parts {
				if part == "of" {
					if i > 0 {
						_, err = strconv.Atoi(parts[i-1])
						if err != nil {
							continue
						}
						countStr = parts[i-1]
					}
				}
			}
			if curve == "" || countStr == "" {
				return servererrors.BadRequestf("failed to parse count-triples output, try without json query parameter")
			}
			count, _ := strconv.Atoi(countStr)
			output[curve] = count
		}
		return c.JSON(output)
	} else {
		return c.SendString(string(bz))
	}
}

type InstallTriplesInput struct {
	Overwrite   bool   `json:"overwrite"`
	Participant string `json:"participant"`
}

// This is not really needed as this is available as one of the one-shot services (triples-generate.service).
// But unlike the service, this exposes the `--overwrite` capability.  Could be useful for backup/recovery scenarios.
func (endpoints *Endpoints) InstallTriples(c *fiber.Ctx) error {
	input := InstallTriplesInput{}
	if err := json.Unmarshal(c.Body(), &input); err != nil {
		return servererrors.BadRequestf("failed to parse body: %v", err)
	}

	args := []string{"genesis", "install-triples"}
	if input.Overwrite {
		args = append(args, "--overwrite")
	}
	if input.Participant != "" {
		args = append(args, "--participant", input.Participant)
	}
	err := endpoints.execCordWithHome(args, IncludeEar)
	if err != nil {
		return servererrors.InternalErrorf("failed to install triples: %v", err)
	}
	return c.JSON(nil)
}
