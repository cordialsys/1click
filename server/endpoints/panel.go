package endpoints

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cordialsys/panel/pkg/admin"
	"github.com/cordialsys/panel/pkg/nonce"
	"github.com/cordialsys/panel/pkg/resource"
	"github.com/cordialsys/panel/pkg/secret"
	"github.com/cordialsys/panel/pkg/treasury"
	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

func (endpoints *Endpoints) GetPanel(c *fiber.Ctx) error {
	panelData := *endpoints.panel
	// hide the secret
	panelData.ApiKeyId = strings.Split(panelData.ApiKey, ":")[0]
	panelData.ApiKey = ""

	// hide the ear secret if it's "raw:"
	if endpoints.panel.EarSecret.IsType(secret.Raw) {
		panelData.EarSecret = "raw:<hidden>"
	}

	var hasGenesis = false
	_, err := os.Stat(endpoints.panel.TreasuryHome.Genesis())
	if err == nil {
		hasGenesis = true
	}
	var hasValidatorKey = false
	_, err = os.Stat(endpoints.panel.TreasuryHome.PrivValidatorKey())
	if err == nil {
		hasValidatorKey = true
	}

	if hasGenesis {
		panelData.State = panel.StateActive
		treasuryClient := treasury.NewClient()

		if panelData.Blueprint == panel.BlueprintDemo {
			// demos enable sso_self_link feature but leave root user in place
			// so instead we just check for the sso_self_link feature
			feature, err := treasuryClient.GetFeature("sso_self_link")
			if err != nil {
				// assume sealed
				panelData.State = panel.StateSealed
			} else {
				if feature.State == resource.FeatureStateActive {
					panelData.State = panel.StateSealed
				}
			}

		} else {
			// Check for existance of root user to determine if we sealed blueprint or not
			resp, err := treasuryClient.GetUser("root")
			if err != nil {
				// don't know
				slog.Debug("failed to query for root user", "error", err)
				// fmt.Println(err)
				panelData.State = panel.StateStopped
				svc, _ := getSystemdService(c.Context(), ServiceTreasury)
				if svc.ActiveState == "active" || svc.ActiveState == "activating" {
					if len(panelData.Users) == 0 {
						panelData.State = panel.StateActive
					} else {
						panelData.State = panel.StateSealed
					}
				}
				fmt.Println(panelData.State)
			} else {
				if resp.StatusCode == http.StatusNotFound {
					panelData.State = panel.StateSealed
				}
			}
		}

	} else if hasValidatorKey {
		panelData.State = panel.StateGenerated
	} else {
		panelData.State = panel.StateInactive
	}

	panelData.Recipient = endpoints.identity.Recipient().String()

	return c.JSON(panelData)
}

type IsActivatedResponse struct {
	Activated bool   `json:"activated"`
	Message   string `json:"message"`
}

func validateAPIKey(apiKey string) (string, error) {
	if len(apiKey) > 128 {
		return "", servererrors.BadRequestf("API key cannot be longer than 128 characters")
	}

	if !strings.Contains(apiKey, ":") {
		decodedKey, err := base64.StdEncoding.DecodeString(apiKey)
		if err != nil {
			return "", servererrors.BadRequestf("invalid API key; expected format: <secret>:<key> or base64(<secret>:<key>)")
		}
		apiKey = string(decodedKey)
	}
	if !utf8.ValidString(apiKey) {
		return "", servererrors.BadRequestf("invalid API key; not valid UTF-8")
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return "", servererrors.BadRequestf("API key cannot be empty")
	}

	return apiKey, nil
}

type SealPanelRequest struct {
	Blueprint panel.Blueprint       `json:"blueprint"`
	Users     []panel.UserWithRoles `json:"users"`
	Roles     []string              `json:"roles"`
}

func (endpoints *Endpoints) SealPanel(c *fiber.Ctx) error {
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	var req SealPanelRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return servererrors.BadRequestf("failed to parse request: %v", err)
	}
	binaryDir := endpoints.panel.BinaryDir
	treasuryBin := filepath.Join(binaryDir, "treasury")

	var blueprint panel.Blueprint
	switch req.Blueprint {
	case panel.BlueprintProduction, "deployment":
		// ok, alias
		blueprint = panel.BlueprintProduction
	case panel.BlueprintDemo:
		// ok
		blueprint = panel.BlueprintDemo
	default:
		return servererrors.BadRequestf("invalid blueprint: %s", req.Blueprint)
	}

	invitedUsers := []panel.UserWithInvite{}
	args := []string{"blueprint", string(blueprint), fmt.Sprint(endpoints.panel.TreasurySize)}

	if blueprint == panel.BlueprintProduction {
		if len(req.Users) == 0 {
			return servererrors.BadRequestf("no users provided for production blueprint")
		}

		if len(req.Roles) > 0 {
			args = append(args, "--roles", strings.Join(req.Roles, ","))
		}

		// Construct the list of users to invite
		for _, user := range req.Users {
			if len(*user.Emails) == 0 && user.PrimaryEmail == nil {
			}
			var email = admin.DerefOrZero(user.PrimaryEmail)
			if email == "" {
				if user.Emails != nil && len(*user.Emails) > 0 {
					email = (*user.Emails)[0]
				}
			}
			if email == "" {
				return servererrors.BadRequestf("user %s (%s) has no email", user.Name, admin.DerefOrZero(user.DisplayName))
			}
			webInvite := nonce.NewString()
			cliInvite := nonce.NewString()
			userCode := fmt.Sprintf("%s;cli=%s;web=%s", email, cliInvite, webInvite)
			args = append(args, "--user", userCode)

			invitedUsers = append(invitedUsers, panel.UserWithInvite{
				UserWithRoles: user,
				WebInvite:     webInvite,
				CliInvite:     cliInvite,
			})
		}
	} else {
		// Nothing to do for demo blueprint
	}
	slog.Info("generating blueprint", "args", args)

	cmd := exec.Command(treasuryBin, args...)
	// Include API key so that treasury can lookup user info, etc
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", panel.ENV_API_KEY, endpoints.panel.ApiKey))
	outputBz, err := cmd.CombinedOutput()
	if err != nil {
		return servererrors.BadRequestf("failed to generate blueprint: %v: %s", err, string(outputBz))
	}
	slog.Info("generated blueprint", "output", string(outputBz))

	err = os.WriteFile(endpoints.panel.PanelDir.BlueprintFile(), []byte(outputBz), 0644)
	if err != nil {
		return servererrors.InternalErrorf("failed to write blueprint: %v", err)
	}

	endpoints.panel.Users = invitedUsers
	endpoints.panel.Blueprint = blueprint
	err = panel.Save(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to save panel: %v", err)
	}

	_, _ = updateSystemdService(c.Context(), ServiceBlueprint, "start")

	// wait for it to stop on it's own (UI can watch logs)
	err = waitSystemdServiceToStop(c.Context(), ServiceBlueprint, 120*time.Second)
	if err != nil {
		return servererrors.InternalErrorf("failed to wait for blueprint to be applied: %v", err)
	}

	return c.JSON(nil)
}
