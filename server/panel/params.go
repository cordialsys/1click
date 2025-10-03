package panel

import (
	"crypto"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/cordialsys/panel/pkg/admin"
	"github.com/cordialsys/panel/pkg/paths"
	"github.com/cordialsys/panel/pkg/secret"
	"github.com/sigstore/cosign/v2/pkg/signature"
	sigstore "github.com/sigstore/sigstore/pkg/signature"
)

const ENV_TREASURY_HOME = "TREASURY_HOME"
const ENV_API_KEY = "TREASURY_API_KEY"
const ENV_SUPERVISOR_HOME = "SUPERVISOR_HOME"
const ENV_TRIPLES_COUNT = "TRIPLES_COUNT"
const ENV_TREASURY_BACKUP_DIR = "TREASURY_BACKUP_DIR"
const ENV_TREASURY_API_NODE = "TREASURY_API_NODE"
const ENV_TREASURY_ENABLE_CONNECTOR = "TREASURY_ENABLE_CONNECTOR"
const ENV_TREASURY_OTEL_ENABLED = "TREASURY_OTEL_ENABLED"

const ENV_PANEL_TREASURY_SIZE = "PANEL_TREASURY_SIZE"
const ENV_PANEL_NODE_ID = "PANEL_NODE_ID"

// This should match the indivdual treasury Bak configuration
type Bak struct {
	Id  string `json:"id" toml:"id"`
	Key string `json:"key" toml:"key"`
}

type State string

const (
	// Activation is not complete
	StateInactive State = "inactive"
	// Treasury is generated
	StateGenerated State = "generated"
	// Treasury is completed
	StateActive State = "active"
	// Policy has been applied
	StateSealed State = "sealed"
	// probably treasury node has stopped
	StateStopped State = "stopped"
)

type UserWithRoles struct {
	admin.User
}
type UserWithInvite struct {
	UserWithRoles
	WebInvite string `json:"web_invite"`
	CliInvite string `json:"cli_invite"`
}

type Blueprint string

const (
	BlueprintProduction Blueprint = "production"
	BlueprintDemo       Blueprint = "demo"
)

type Panel struct {
	//// These are configured by CLI / host machine:
	TreasuryHome   paths.TreasuryHome   `json:"treasury_home"`
	SupervisorHome paths.SupervisorHome `json:"supervisor_home"`
	BinaryDir      string               `json:"binary_dir"`
	PanelDir       paths.PanelHome      `json:"panel_dir"`
	BackupDir      string               `json:"backup_dir"`
	TreasuryUser   string               `json:"treasury_user"`
	binaryVerifier sigstore.Verifier
	////

	//// These are updated via panel API endpoints:
	// Updates via POST /activate
	ApiKey   string `json:"api_key,omitempty"`
	ApiKeyId string `json:"api_key_id,omitempty"`
	// Updates via POST /activate
	NodeId     uint64 `json:"node_id,omitempty"`
	TreasuryId string `json:"treasury_id,omitempty"`
	// Updates via POST /activate/backup
	Baks []Bak `json:"baks,omitempty"`

	// Updates via POST /activate
	Connector bool `json:"connector,omitempty"`
	// Updates via POST /activate
	ApiNode bool `json:"api_node,omitempty"`
	// Updates via POST /activate
	Network string `json:"network,omitempty"`
	// Updates via POST /activate
	OtelEnabled  bool   `json:"otel_enabled,omitempty"`
	TreasurySize uint64 `json:"treasury_size,omitempty"`
	// Updates via PUT/DELETE of /panel/ear
	EarSecret secret.Secret `json:"ear_secret,omitempty"`
	// Updates via POST /panel/seal
	Users     []UserWithInvite `json:"users,omitempty"`
	Blueprint Blueprint        `json:"blueprint,omitempty"`
	////

	//// Calculated at query time
	// State figured out based on current environment
	State State `json:"state"`
	// Age recipient for sending encrypted backups to the panel server / "identity"
	Recipient string `json:"recipient,omitempty"`
}

func (p *Panel) SetBinaryVerifier(verifier sigstore.Verifier) {
	p.binaryVerifier = verifier
}

const CordialBinaryPublicKeyRaw = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEAmz48XmSfB8sn6h03RoERbdplk9K
fZj0k94OzoqMxwOaT8AvL/iZ7kumiH+IkO7X+ekl4lbbhhaSwkKZHR8wPA==
-----END PUBLIC KEY-----`

var defaultBinaryVerifier sigstore.Verifier

func init() {
	var err error
	defaultBinaryVerifier, err = signature.LoadPublicKeyRaw([]byte(CordialBinaryPublicKeyRaw), crypto.SHA256)
	if err != nil {
		panic(err)
	}
}

func (p *Panel) GetBinaryVerifierOrDefault() sigstore.Verifier {
	if p.binaryVerifier != nil {
		return p.binaryVerifier
	}
	// sanity check
	if defaultBinaryVerifier == nil {
		panic("default binary verifier is nil")
	}
	return defaultBinaryVerifier
}

func (p *Panel) NodeName() string {
	return fmt.Sprintf("treasuries/%s/nodes/%d", p.TreasuryId, p.NodeId)
}

func (p *Panel) TreasuryName() string {
	return fmt.Sprintf("treasuries/%s", p.TreasuryId)
}

func (p *Panel) HasNodeSet() bool {
	return p.NodeId != 0 && p.TreasuryId != ""
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func New() *Panel {
	return &Panel{
		TreasuryHome: paths.TreasuryHome(
			envOrDefault(
				ENV_TREASURY_HOME,
				// important to store in /var/ as these are persisted
				// between updating bootable containers, and writable by the treasury/supervisor user
				"/var/treasury",
			),
		),
		SupervisorHome: paths.SupervisorHome(
			envOrDefault(
				ENV_SUPERVISOR_HOME,
				// important to store in /var/ as these are persisted
				// between updating bootable containers, and writable by the treasury/supervisor user
				"/var/supervisor",
			),
		),
		BackupDir: envOrDefault(
			ENV_TREASURY_BACKUP_DIR,
			"/var/backup",
		),
		BinaryDir: "/usr/bin",
		// /etc is also persisted, but will be writeable only by root or the panel user
		PanelDir:     "/etc/panel",
		TreasuryUser: "cordial",

		ApiKey: envOrDefault(ENV_API_KEY, ""),
	}
}

func Save(panel *Panel) error {
	if panel.PanelDir == "" {
		slog.Error("panel directory is not set, using default dir")
		panel.PanelDir = New().PanelDir
	}
	panelBz, err := json.Marshal(panel)
	if err != nil {
		return err
	}
	err = os.MkdirAll(panel.PanelDir.String(), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(panel.PanelDir.PanelFile(), panelBz, 0644)
	if err != nil {
		return err
	}

	// writeout a env file that can be used by systemd services running on the host
	envContents := fmt.Sprintf(
		ENV_TREASURY_HOME+"=%s\n"+
			ENV_SUPERVISOR_HOME+"=%s\n"+
			ENV_API_KEY+"=%s\n"+
			ENV_TREASURY_BACKUP_DIR+"=%s\n",
		panel.TreasuryHome, panel.SupervisorHome, panel.ApiKey, panel.BackupDir,
	)

	if panel.Connector {
		envContents += fmt.Sprintf("%s=1\n", ENV_TREASURY_ENABLE_CONNECTOR)
	}
	if panel.ApiNode {
		envContents += fmt.Sprintf("%s=1\n", ENV_TREASURY_API_NODE)
	}
	if panel.OtelEnabled {
		envContents += fmt.Sprintf("%s=true\n", ENV_TREASURY_OTEL_ENABLED)
	}
	size := 1
	if panel.TreasurySize > 0 {
		size = int(panel.TreasurySize)
	}
	envContents += fmt.Sprintf("%s=%d\n", ENV_PANEL_TREASURY_SIZE, size)

	err = os.WriteFile(panel.PanelDir.EnvFile(), []byte(envContents), 0644)
	if err != nil {
		return err
	}
	return nil
}

func Load(panelDir paths.PanelHome) (*Panel, error) {
	panelBz, err := os.ReadFile(panelDir.PanelFile())
	if err != nil {
		return nil, err
	}
	var panel Panel
	err = json.Unmarshal(panelBz, &panel)
	if err != nil {
		return nil, err
	}
	return &panel, nil
}
