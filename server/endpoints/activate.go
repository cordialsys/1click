package endpoints

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/cordialsys/panel/pkg/admin"
	"github.com/cordialsys/panel/pkg/api"
	"github.com/cordialsys/panel/pkg/client"
	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc/codes"
)

// Validates the API key, as well as the associated node + treasury.
// If the API key is valid, it will get saved in the panel params.
func (endpoints *Endpoints) ActivateApiKey(c *fiber.Ctx) error {
	var request client.RequestActivateApiKey
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return servererrors.BadRequestf("failed to parse request: %v", err)
	}

	if request.ApiKey == "" {
		return servererrors.BadRequestf("api_key is required")
	}
	client := admin.NewClient(request.ApiKey)

	apiKeyId := ""
	if !strings.Contains(request.ApiKey, ":") {
		decoded, err := base64.StdEncoding.DecodeString(request.ApiKey)
		if err != nil {
			return servererrors.BadRequestf("failed to parse API key ID: %v", err)
		}
		apiKeyId = strings.Split(string(decoded), ":")[0]
	} else {
		apiKeyId = strings.Split(request.ApiKey, ":")[0]
	}

	// Test that the API key authorizes
	apiKeyResource, err := client.GetApiKey(apiKeyId)
	if err != nil {
		return servererrors.BadRequestf("could not get api-key: %v", err)
	}

	nodeName := api.DerefOrZero(apiKeyResource.Node)
	if nodeName == "" {
		return servererrors.BadRequestf("api-key is not associated with a node")
	}

	endpoints.panel.ApiKey = request.ApiKey

	parts := strings.Split(nodeName, "/")
	if len(parts) != 4 {
		return servererrors.BadRequestf("invalid node name: %s", nodeName)
	}
	treasuryId := parts[1]
	nodeId := parts[3]
	nodeIdInt, err := strconv.ParseUint(nodeId, 10, 64)
	if err != nil {
		return servererrors.BadRequestf("invalid node id: %s", nodeId)
	}

	// test that both the node + treasury exist
	node, err := client.GetNode(nodeName)
	if err != nil {
		return servererrors.BadRequestf("could not get node permissioned by API key: %v", err)
	}

	treas, err := client.GetTreasuryById(treasuryId)
	if err != nil {
		return servererrors.BadRequestf("could not get treasury permissioned by API key: %v", err)
	}
	if api.DerefOrZero(treas.Size) == 0 {
		return servererrors.BadRequestf("treasury size is not set - contact Cordial Systems")
	}
	if api.DerefOrZero(treas.InitialVersion) == "" {
		return servererrors.InternalErrorf("treasury initial version is not set - contact Cordial Systems")
	}

	endpoints.panel.NodeId = nodeIdInt
	endpoints.panel.TreasuryId = treasuryId
	endpoints.panel.Connector = node.Connector
	if request.Connector != nil {
		endpoints.panel.Connector = *request.Connector
	}
	if treas.Network != nil {
		// default is mainnet
		endpoints.panel.Network = *treas.Network
	}
	// override the network if it is set in the request
	if request.Network != nil {
		endpoints.panel.Network = *request.Network
	}
	endpoints.panel.TreasurySize = uint64(api.DerefOrZero(treas.Size))

	err = panel.Save(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to save panel: %v", err)
	}

	slog.Info("updated panel params", "panel", endpoints.panel, "path", endpoints.panel.PanelDir.PanelFile())
	return c.JSON(endpoints.panel)
}

func (endpoints *Endpoints) ActivateBinaries(c *fiber.Ctx) error {
	version := c.Query("version")
	if version == "" {
		version = "latest"
	}
	ctx := c.Context()

	treasury, err := getSystemdService(ctx, ServiceTreasury)
	if err != nil {
		slog.Warn("failed to get treasury service", "error", err)
	} else {
		// if treasury is running, stop it
		if treasury.ActiveState == "active" {
			stopSystemdServiceAndWait(ctx, ServiceTreasury)
			defer func() {
				// start it again
				updateSystemdService(ctx, ServiceTreasury, "start")
			}()
		}
	}

	// just install the latest of cord, signer, treasury-cli
	for _, binaryName := range []string{"cord", "signer", "treasury-cli"} {
		remote := formatDownloadUrl(version, binaryName)
		slog.Info("downloading", "remote", remote)
		err := DownloadAndUntar(endpoints.panel, remote, endpoints.panel.BinaryDir, true)
		if err != nil {
			return servererrors.InternalErrorf("failed to download %s: %v", binaryName, err)
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

func (endpoints *Endpoints) ActivateNetwork(c *fiber.Ctx) error {
	if !endpoints.panel.HasNodeSet() {
		// /activate/api-key must be called first
		return servererrors.BadRequestf("the API key has not yet been activated")
	}
	client := admin.NewClient(endpoints.panel.ApiKey)

	networkKey, err := client.GetNetworkKey(endpoints.panel.NodeName())
	if err != nil {
		if apiErr, ok := err.(*admin.Error); ok {
			if apiErr.Code == int(codes.NotFound) {
				return servererrors.BadRequestf("the network key has not yet been setup - contact Cordial Systems")
			}
		}
		return servererrors.InternalErrorf("failed to get network key: %v", err)
	}
	if networkKey == "" {
		return servererrors.BadRequestf("the network key has not yet been setup - contact Cordial Systems")
	}
	node, err := client.GetNode(endpoints.panel.NodeName())
	if err != nil {
		return servererrors.InternalErrorf("failed to get node: %v", err)
	}

	if node.Host == "" {
		return servererrors.BadRequestf("the node host has not yet been setup - contact Cordial Systems")
	}

	// netbird up --setup-key xxx --hostname example.com
	execCmd := exec.Command("netbird", []string{"up", "--setup-key", networkKey, "--hostname", node.Host}...)
	bz, err := execCmd.CombinedOutput()
	slog.Info("exec", "binary", "netbird", "cmd", execCmd.String(), "output", string(bz))
	if err != nil {
		return servererrors.InternalErrorf("failed to setup network: %v: %s", err, string(bz))
	}

	return c.SendStatus(fiber.StatusOK)
}

// Set a backup phrases for the node.
func (endpoints *Endpoints) ActivateBackup(c *fiber.Ctx) error {
	// Load from file, in case the user manually reset
	panelInfo, err := panel.Load(endpoints.panel.PanelDir)
	if err != nil {
		return servererrors.InternalErrorf("failed to load panel: %v", err)
	}
	if !slices.Equal(panelInfo.Baks, endpoints.panel.Baks) {
		endpoints.panel.Baks = panelInfo.Baks
	}

	var request client.RequestActivateBackup
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return servererrors.BadRequestf("failed to parse request: %v", err)
	}

	if len(request.Baks) == 0 {
		return servererrors.BadRequestf("need at least one backup key configured")
	}

	for _, bak := range request.Baks {
		if bak.Key == "" {
			return servererrors.BadRequestf("bak is required")
		}
		if !strings.HasPrefix(bak.Key, "age1") {
			return servererrors.BadRequestf("backup key must be an age key")
		}
	}
	// Do not allow changing the backup keys, as this could be a backdoor.
	if len(endpoints.panel.Baks) > 0 {
		return servererrors.BadRequestf("backup keys already set -- cannot change without resetting the treasury")
	}

	endpoints.panel.Baks = request.Baks
	err = panel.Save(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to save panel: %v", err)
	}

	return c.SendStatus(fiber.StatusOK)
}

func (endpoints *Endpoints) ActivateOtel(c *fiber.Ctx) error {
	// Load from file, in case the user manually reset
	panelInfo, err := panel.Load(endpoints.panel.PanelDir)
	if err != nil {
		return servererrors.InternalErrorf("failed to load panel: %v", err)
	}
	if !slices.Equal(panelInfo.Baks, endpoints.panel.Baks) {
		endpoints.panel.Baks = panelInfo.Baks
	}

	var request client.RequestActivateOtel
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return servererrors.BadRequestf("failed to parse request: %v", err)
	}

	endpoints.panel.OtelEnabled = request.Enabled
	err = panel.Save(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to save panel: %v", err)
	}

	return c.SendStatus(fiber.StatusOK)
}
