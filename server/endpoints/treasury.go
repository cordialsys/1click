package endpoints

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cordialsys/panel/pkg/admin"
	"github.com/cordialsys/panel/pkg/api"
	"github.com/cordialsys/panel/pkg/client"
	"github.com/cordialsys/panel/pkg/genesis"
	"github.com/cordialsys/panel/pkg/names"
	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/gofiber/fiber/v2"
	"github.com/pelletier/go-toml/v2"
	"github.com/sirupsen/logrus"
)

// pass through to the treasury API
func (endpoints *Endpoints) GetTreasury(c *fiber.Ctx) error {
	req, err := http.NewRequest("GET", "http://127.0.0.1:8777/v1/treasury", nil)
	if err != nil {
		return servererrors.InternalErrorf("failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return servererrors.InternalErrorf("failed to send request: %v", err)
	}
	c.Status(resp.StatusCode)
	c.Response().Header.Set("Content-Type", resp.Header.Get("Content-Type"))
	defer resp.Body.Close()
	io.Copy(c, resp.Body)
	return nil
}

type TreasuryGenerate struct {
	// Required: list of backup keys
	Baks        []string `json:"baks"`
	Participant int      `json:"participant,omitempty"`

	// Optional:
	Overwrite bool `json:"overwrite,omitempty"`
	// id of the treasury (generated if not provided)
	Id string `json:"id,omitempty"`
}

type Treasury struct {
	Baks        []string `json:"baks"`
	Participant int      `json:"participant"`
}

type ExecType string

const (
	IncludeEar  ExecType = "cord"
	NoEarNeeded ExecType = "signer"
)

// Executes a cord command, adding the --home flag to the command
func (endpoints *Endpoints) execCordWithHome(cmd []string, execType ExecType, envs ...string) error {
	params := endpoints.panel
	binaryDir := params.BinaryDir
	cord := filepath.Join(binaryDir, "cord")

	execList := append(cmd, "--home", string(params.TreasuryHome))

	execCmd := exec.Command(cord, execList...)
	execCmd.Env = append(os.Environ(), envs...)

	if execType == IncludeEar {
		err := endpoints.attachEarSecretToCmd(execCmd)
		if err != nil {
			return err
		}
	}

	bz, err := execCmd.CombinedOutput()
	slog.Info("exec", "binary", cord, "cmd", execCmd.String(), "output", string(bz))

	if err != nil {
		return fmt.Errorf("failed to run `%s`: %v", execCmd.String(), string(bz))
	}
	return nil
}

func unsafeDeleteTreasury(panel *panel.Panel) error {
	if panel.TreasuryHome != "" {
		if err := os.RemoveAll(string(panel.TreasuryHome)); err != nil {
			slog.Error("failed to reset treasury", "error", err)
			return err
		}
	}
	return nil
}

func (endpoints *Endpoints) GenerateTreasury(c *fiber.Ctx) error {
	if len(endpoints.panel.Baks) == 0 {
		return servererrors.BadRequestf("no backup keys configured, should activate a backup key first")
	}

	client, err := endpoints.AdminClient()
	if err != nil {
		return err
	}
	node, err := client.GetNode(endpoints.panel.NodeName())
	if err != nil {
		return servererrors.InternalErrorf("failed to get node: %v", err)
	}
	treasury, err := client.GetTreasuryById(endpoints.panel.TreasuryId)
	if err != nil {
		return servererrors.InternalErrorf("failed to get treasury: %v", err)
	}

	// target the image for the initial version if not already set
	// If there is already a version set, we don't want to change it.  As we could be restoring from a backup.
	image, err := readSupervisorImage(endpoints.panel)
	if err != nil || image == "" {
		initialVersion := api.DerefOrZero(treasury.InitialVersion)
		initialVersion = strings.TrimPrefix(initialVersion, "v")
		slog.Info("no initial image found, using treasury initial version", "error", err, "initial_version", initialVersion)
		if initialVersion == "" {
			return servererrors.InternalErrorf("treasury initial version is not set - contact Cordial Systems")
		}
		image = "us-docker.pkg.dev/cordialsys/containers/treasury:" + initialVersion
		args := []string{
			"supervise",
			"use-image",
			image,
		}
		err := execSupervisorWithHome(endpoints.panel, args)
		if err != nil {
			return servererrors.InternalErrorf("failed to use initial image: %v", err)
		}
	}

	baks := []string{}
	for _, bak := range endpoints.panel.Baks {
		baks = append(baks, bak.Key)
	}

	args := []string{
		"genesis",
		"init",
		fmt.Sprintf("--participant=%d", endpoints.panel.NodeId),
		fmt.Sprintf("--bak=%s", strings.Join(baks, ",")),
		"--upload-backups",
	}

	err = endpoints.execCordWithHome(args, IncludeEar)
	if err != nil {
		// Perhaps there is already a treasury initialized, in which case the user should
		// manually call `DELETE /treasury`
		return servererrors.InternalErrorf("failed to generate treasury: %v", err)
	}

	initFileName := newInitFileName(endpoints.panel.NodeId)
	initFileBz, err := os.ReadFile(filepath.Join(string(endpoints.panel.TreasuryHome), initFileName))
	if err != nil {
		unsafeDeleteTreasury(endpoints.panel)
		if os.IsNotExist(err) {
			return servererrors.NotFoundf("treasury init file not found")
		}
		return servererrors.InternalErrorf("failed to read treasury init file: %v", err)
	}

	var initFile genesis.TreasuryInitConfig
	if err := json.Unmarshal(initFileBz, &initFile); err != nil {
		unsafeDeleteTreasury(endpoints.panel)
		return servererrors.InternalErrorf("failed to parse treasury init file: %v", err)
	}

	// now we update the admin resource
	node.Keys = &admin.Keys{
		Engine: struct {
			Identity string "json:\"identity\""
		}{
			Identity: initFile.Validator.PublicKey,
		},
		Node: struct {
			Identity string "json:\"identity\""
		}{
			// confusingly, nodeId here is the cosmos-sdk node-id
			Identity: initFile.NodeId,
		},
		Signer: struct {
			Identity  string "json:\"identity\""
			Recipient string "json:\"recipient\""
		}{
			Identity:  initFile.Signer.VerifyingKey,
			Recipient: initFile.Signer.Recipient,
		},
	}
	node.Baks = &[]admin.Bak{}
	for _, bak := range endpoints.panel.Baks {
		*node.Baks = append(*node.Baks, admin.Bak{
			Id:  api.IfValueNotZero(bak.Id),
			Bak: bak.Key,
		})
	}
	// update the node with the new key info
	updated, err := client.UpdateNode(endpoints.panel.NodeName(), node)
	if err != nil {
		// rollback the treasury data in case of API rejection
		unsafeDeleteTreasury(endpoints.panel)
		return servererrors.InternalErrorf("failed to update node: %v", err)
	}

	return c.JSON(updated)
}

func (endpoints *Endpoints) DeleteTreasury(c *fiber.Ctx) error {
	// stop services
	_, err := updateSystemdService(c.Context(), ServiceTreasury, "stop")
	if err != nil {
		logrus.WithError(err).Error("failed to stop start-treasury service")
	}
	_, err = updateSystemdService(c.Context(), ServiceStartTreasury, "stop")
	if err != nil {
		logrus.WithError(err).Error("failed to stop start-treasury service")
	}
	_, err = updateSystemdService(c.Context(), ServiceBlueprint, "stop")
	if err != nil {
		logrus.WithError(err).Error("failed to stop blueprint service")
	}

	err = unsafeDeleteTreasury(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to delete treasury: %v", err)
	}
	// Sync the node info on admin API so that no one uses deleted keys
	client, err := endpoints.AdminClient()
	if err != nil {
		return err
	}
	nodeMaybe, err := client.GetNode(endpoints.panel.NodeName())
	if err != nil {
		// ignore
		return c.SendStatus(http.StatusOK)
	}
	if nodeMaybe.Keys != nil {
		nodeMaybe.Keys = nil
		_, err = client.UpdateNode(endpoints.panel.NodeName(), nodeMaybe)
		if err != nil {
			return servererrors.InternalErrorf("failed to delete treasury on remote API: %v", err)
		}
	}

	_, deleteSupervisor := c.Queries()["supervisor"]
	if deleteSupervisor {
		err = os.RemoveAll(endpoints.panel.SupervisorHome.ConfigFile())
		if err != nil {
			return servererrors.InternalErrorf("failed to delete supervisor config: %v", err)
		}
		endpoints.panel.EarSecret = ""
	}

	// Reset panel settings relating to backups + blueprint
	endpoints.panel.Users = []panel.UserWithInvite{}
	endpoints.panel.Baks = []panel.Bak{}
	endpoints.panel.Blueprint = ""
	_ = panel.Save(endpoints.panel)

	return c.SendStatus(http.StatusOK)
}

func (endpoints *Endpoints) SyncTreasuryPeers(c *fiber.Ctx) error {
	var req client.RequestSyncTreasuryPeers
	if len(c.Body()) > 0 {
		if err := json.Unmarshal(c.Body(), &req); err != nil {
			return servererrors.BadRequestf("failed to parse request: %v", err)
		}
	}
	if len(req.Peers) == 0 {
		// lookup from admin API
		admin, err := endpoints.AdminClient()
		if err != nil {
			return err
		}
		nodePage, err := admin.ListNodes(endpoints.panel.TreasuryId)
		if err != nil {
			return servererrors.InternalErrorf("failed to list nodes: %v", err)
		}
		if len(*nodePage.Nodes) == 0 {
			return servererrors.InternalErrorf("no nodes found")
		}
		req.Peers = []client.Peer{}
		for _, node := range *nodePage.Nodes {
			if node.Name != endpoints.panel.NodeName() {
				req.Peers = append(req.Peers, client.Peer{
					Socket:      node.Host,
					NodeId:      node.Keys.Node.Identity,
					Participant: int(names.NodeName(node.Name).Participant()),
				})
			}
		}
		conn, err := dbus.NewSystemConnectionContext(c.Context())
		if err != nil {
			return servererrors.InternalErrorf("failed to connect to systemd: %v", err)
		}
		const serviceName = "treasury.service"
		units, err := conn.ListUnitsByNamesContext(c.Context(), []string{serviceName})
		if err != nil {
			return servererrors.InternalErrorf("failed to get units: %v", err)
		}
		// If treasury service is running, we should stop + restart it to take the new peer changes
		if units[0].ActiveState == "active" {
			slog.Info("stopping treasury service")
			_, err = conn.StopUnitContext(c.Context(), serviceName, "replace", nil)
			if err != nil {
				return servererrors.InternalErrorf("failed to stop treasury service: %v", err)
			}
			defer func() {
				slog.Info("restarting treasury service")
				_, err = conn.RestartUnitContext(c.Context(), serviceName, "replace", nil)
				if err != nil {
					slog.Error("failed to start treasury service", "error", err)
				}
			}()
		}
	}

	// default force, as we manually filter out ourselves from the peer list
	force := true
	if req.Force != nil {
		force = *req.Force
	}
	peers := []string{}
	nodeIds := []string{}
	participantIds := []string{}
	signers := []string{}
	for _, peer := range req.Peers {
		peers = append(peers, peer.Socket)
		nodeIds = append(nodeIds, peer.NodeId)
		participantIds = append(participantIds, fmt.Sprintf("%d", peer.Participant))
		if peer.SignerSocket != "" {
			signers = append(signers, peer.SignerSocket)
		}
	}

	args := []string{
		"genesis",
		"config-peers",
		"--peers", strings.Join(peers, ","),
		"--node-ids", strings.Join(nodeIds, ","),
		"--participant-ids", strings.Join(participantIds, ","),
	}
	if len(signers) > 0 {
		args = append(args, "--signers", strings.Join(signers, ","))
	}
	if force {
		args = append(args, "--force")
	}
	if req.Listen != "" && req.ListenSigner == "" {
		args = append(args, "--listen", req.Listen)
	} else {
		if req.Listen != "" {
			args = append(args, "--listen-engine", req.Listen)
		}
		if req.ListenSigner != "" {
			args = append(args, "--listen-signer", req.ListenSigner)
		}
	}
	err := endpoints.execCordWithHome(args, NoEarNeeded)
	if err != nil {
		return servererrors.InternalErrorf("failed to generate treasury: %v", err)
	}
	return c.JSON(nil)
}

func (endpoints *Endpoints) GetTreasuryConfig(c *fiber.Ctx) error {
	treasuryConfig, err := os.ReadFile(endpoints.panel.TreasuryHome.TreasuryConfig())
	if err != nil {
		if os.IsNotExist(err) {
			return servererrors.NotFoundf("treasury config not found")
		}
		return servererrors.InternalErrorf("failed to read treasury config: %v", err)
	}
	// application/toml wins
	// https://github.com/toml-lang/toml/issues/465
	// https://www.iana.org/assignments/media-types/application/toml
	c.Set("Content-Type", "application/toml")
	c.Write(treasuryConfig)
	return nil
}

func readTreasuryParticipant(panel *panel.Panel) (string, error) {
	treasuryConfig, err := os.ReadFile(panel.TreasuryHome.TreasuryConfig())
	if err != nil {
		return "", err
	}
	var participant string
	// support reading it as int or string
	var config1 struct {
		InitializedAs int `toml:"initialized_as"`
	}
	var config2 struct {
		InitializedAs string `toml:"initialized_as"`
	}
	err1 := toml.Unmarshal(treasuryConfig, &config1)
	if err1 == nil {
		participant = fmt.Sprintf("%d", config1.InitializedAs)
	} else {
		err2 := toml.Unmarshal(treasuryConfig, &config2)
		if err2 != nil {
			return "", errors.Join(err1, err2)
		}
		participant = config2.InitializedAs
	}
	return participant, nil
}

func newInitFileName(participant uint64) string {
	return fmt.Sprintf("init-%d.json", participant)
}

func (endpoints *Endpoints) GetTreasuryParticipant(c *fiber.Ctx) error {
	participant, err := readTreasuryParticipant(endpoints.panel)
	if err != nil {
		return servererrors.InternalErrorf("failed to read treasury participant: %v", err)
	}
	return c.JSON(participant)
}

func asTemporaryJsonFiles(initFiles []genesis.TreasuryInitConfig) (paths []string, closer func(), err error) {
	tmpDir, err := os.MkdirTemp("", "treasury-install")
	if err != nil {
		return nil, nil, servererrors.InternalErrorf("failed to create tempdir: %v", err)
	}
	closer = func() {
		os.RemoveAll(tmpDir)
	}
	for i, initFile := range initFiles {
		contents, err := json.MarshalIndent(initFile, "", "  ")
		if err != nil {
			closer()
			return nil, nil, servererrors.InternalErrorf("failed to marshal init file: %v", err)
		}
		path := filepath.Join(tmpDir, fmt.Sprintf("tmp-%d.json", i))
		err = os.WriteFile(path, contents, 0644)
		if err != nil {
			closer()
			return nil, nil, servererrors.InternalErrorf("failed to write init file: %v", err)
		}
		paths = append(paths, path)
	}
	return paths, closer, nil
}

func (endpoints *Endpoints) PostTreasuryCompleteAndStart(c *fiber.Ctx) error {
	err := endpoints.PostTreasuryComplete(c)
	if err != nil {
		// not all nodes are ready, so we start the start-treasury service instead.
		if apiErr, ok := err.(*servererrors.ErrorResponse); ok && apiErr.Code == servererrors.CodeFailedPrecondition {
			logrus.WithError(err).Info("Starting start-treasury service since it is not yet complete")
			_, err = updateSystemdService(c.Context(), ServiceStartTreasury, "start")
			if err != nil {
				return servererrors.InternalErrorf("failed to start start-treasury service: %v", err)
			}
			return c.SendStatus(http.StatusOK)
		}
		return err
	}

	// Enable + start the treasury service
	_, err = updateSystemdService(c.Context(), ServiceTreasury, "enable")
	if err != nil {
		return servererrors.InternalErrorf("failed to enable treasury service: %v", err)
	}
	_, err = updateSystemdService(c.Context(), ServiceTreasury, "start")
	if err != nil {
		return servererrors.InternalErrorf("failed to start treasury service: %v", err)
	}

	return c.SendStatus(http.StatusOK)
}

func (endpoints *Endpoints) PostTreasuryComplete(c *fiber.Ctx) error {
	client, err := endpoints.AdminClient()
	if err != nil {
		return err
	}
	treasury, err := client.GetTreasuryById(endpoints.panel.TreasuryId)
	if err != nil {
		return servererrors.InternalErrorf("failed to get treasury: %v", err)
	}
	size := api.DerefOrZero(treasury.Size)
	if size == 0 {
		return servererrors.InternalErrorf("treasury size is not set - contact Cordial Systems")
	}
	nodePage, err := client.ListNodes(endpoints.panel.TreasuryId)
	if err != nil {
		return servererrors.InternalErrorf("failed to list nodes: %v", err)
	}
	if len(*nodePage.Nodes) != int(size) {
		return servererrors.InternalErrorf("treasury size (%d) does not match the number of nodes (%d)", size, len(*nodePage.Nodes))
	}
	// check if all of the nodes are ready
	unreadyNodes := []string{}
	for _, node := range *nodePage.Nodes {
		if node.Keys == nil || !node.Keys.IsReady() {
			unreadyNodes = append(unreadyNodes, node.Name)
		}
	}
	if len(unreadyNodes) > 0 {
		return servererrors.FailedPreconditionf("not all nodes are ready, waiting on: %s", strings.Join(unreadyNodes, ", "))
	}

	// sort by participant id, starting from 1 (ascending)
	// This is required for the input to `cord genesis config-peers`
	nodes := *nodePage.Nodes
	sort.Slice(nodes, func(i, j int) bool {
		return names.NodeName(nodes[i].Name).Participant() < names.NodeName(nodes[j].Name).Participant()
	})

	initFiles := []genesis.TreasuryInitConfig{}
	// peerInitFiles := []genesis.TreasuryInitConfig{}
	peers := []string{}
	peerNodeIds := []string{}
	participantIds := []string{}
	for _, node := range nodes {
		initFile, err := admin.NewInitFile(treasury, &node)
		if err != nil {
			return servererrors.InternalErrorf("failed to create init file for %s: %v", node.Name, err)
		}
		initFiles = append(initFiles, initFile)
		contents, _ := json.MarshalIndent(initFile, "", "  ")
		slog.Info("init file", "participant", initFile.Participant, "contents", string(contents))

		// Do not count ourselves as a peer
		if node.Name != endpoints.panel.NodeName() {
			peer := node.Host
			if port := api.DerefOrZero(node.Port); port != 0 {
				peer = fmt.Sprintf("%s:%d", peer, port)
			}
			peers = append(peers, peer)
			peerNodeIds = append(peerNodeIds, initFile.NodeId)
			participantIds = append(participantIds, fmt.Sprintf("%d", initFile.Participant))
		}
	}

	initFilePaths, close1, err := asTemporaryJsonFiles(initFiles)
	if err != nil {
		return servererrors.InternalErrorf("failed to create init files: %v", err)
	}
	defer close1()

	// sanity check
	if len(initFilePaths) == 0 {
		return servererrors.InternalErrorf("no init files found")
	}
	// sanity check
	if len(peerNodeIds) == 0 {
		return servererrors.InternalErrorf("no peers found")
	}

	configPeersArgs := []string{
		"genesis",
		"config-peers",
		"-vv",
		// use --force since we are manually filtering out ourselves from the peer list, so the tool doesn't
		// need to try to figure it out.
		"--force",
		"--peers", strings.Join(peers, ","),
		"--node-ids", strings.Join(peerNodeIds, ","),
		"--participant-ids", strings.Join(participantIds, ","),
	}
	err = endpoints.execCordWithHome(configPeersArgs, NoEarNeeded)
	if err != nil {
		return servererrors.InternalErrorf("failed to configure peers: %v", err)
	}

	completeArgs := []string{
		"genesis",
		"complete",
	}
	completeArgs = append(completeArgs, initFilePaths...)
	err = endpoints.execCordWithHome(completeArgs, IncludeEar)
	if err != nil {
		return servererrors.InternalErrorf("failed to complete treasury: %v", err)
	}
	// We don't want the treasury home permissions to be owned by root/panel user.
	// so we hand it all over to the treasury user at this point.
	dirs := []string{string(endpoints.panel.TreasuryHome), string(endpoints.panel.BackupDir), string(endpoints.panel.SupervisorHome)}
	for _, dir := range dirs {
		_ = os.MkdirAll(dir, 0755)
		execCmd := exec.Command("chown", "-R", endpoints.panel.TreasuryUser, dir)
		bz, err := execCmd.CombinedOutput()
		if err != nil {
			return servererrors.InternalErrorf("failed to change ownership of %s: %v", dir, err)
		}
		slog.Info("changed ownership of %s", "output", string(bz))
	}

	return c.SendStatus(http.StatusOK)
}

// Pass through to the treasury health endpoint
func (endpoints *Endpoints) GetTreasuryHealth(c *fiber.Ctx) error {
	rawQuery := c.Request().URI().QueryString()
	req, err := http.NewRequest("GET", "http://127.0.0.1:8777/healthy?"+string(rawQuery), nil)
	if err != nil {
		return servererrors.InternalErrorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return servererrors.InternalErrorf("failed to send request: %v", err)
	}
	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Response().Header.Set(k, v)
		}
	}
	c.Status(resp.StatusCode)
	defer resp.Body.Close()
	io.Copy(c, resp.Body)
	return nil
}

// misc endpoint for debugging
func (endpoints *Endpoints) GetTreasuryInit(c *fiber.Ctx) error {
	if !endpoints.panel.HasNodeSet() {
		return servererrors.BadRequestf("the API key has not yet been activated")
	}

	initFileName := newInitFileName(endpoints.panel.NodeId)
	initFile, err := os.ReadFile(filepath.Join(string(endpoints.panel.TreasuryHome), initFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return servererrors.NotFoundf("treasury init file not found")
		}
		return servererrors.InternalErrorf("failed to read treasury init file: %v", err)
	}
	c.Set("Content-Type", "application/json")
	c.Write(initFile)
	return nil
}
