package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"filippo.io/age"
	"github.com/cordialsys/panel/pkg/paths"
	_ "github.com/cordialsys/panel/pkg/plog"
	"github.com/cordialsys/panel/server/endpoints"
	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// Server represents the panel server
type Server struct {
	app      *fiber.App
	params   *panel.Panel
	identity *age.X25519Identity
	Options
}

// Options holds server configuration
type Options struct {
	ListenAddr     string
	TreasuryHome   string
	BinaryDir      string
	PanelDir       string
	ApiKey         string
	SupervisorHome string
	Triples        uint64
	BackupDir      string
	Connector      bool
	ApiNode        bool
	TreasuryUser   string
	WebDir         string
}

func loadPanel(panelDir paths.PanelHome) (*panel.Panel, bool, error) {
	var existingPanel panel.Panel
	if _, err := os.Stat(panelDir.PanelFile()); err == nil {
		panelBz, err := os.ReadFile(panelDir.PanelFile())
		if err != nil {
			return nil, false, err
		}
		err = json.Unmarshal(panelBz, &existingPanel)
		if err != nil {
			return nil, false, err
		}
		return &existingPanel, true, nil
	}
	return nil, false, nil
}

func loadIdentity(panelDir paths.PanelHome) (*age.X25519Identity, bool, error) {
	if _, err := os.Stat(panelDir.IdentityFile()); err == nil {
		panelBz, err := os.ReadFile(panelDir.IdentityFile())
		if err != nil {
			return nil, false, err
		}
		identity, err := age.ParseX25519Identity(string(panelBz))
		if err != nil {
			return nil, false, err
		}
		return identity, true, nil
	}
	return nil, false, nil
}

func saveIdentity(panelDir paths.PanelHome, identity *age.X25519Identity) error {
	identityBz := identity.String()
	return os.WriteFile(panelDir.IdentityFile(), []byte(identityBz), 0644)
}

// New creates a new server instance
func New(args Options) *Server {
	app := fiber.New(fiber.Config{
		AppName: "Panel",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if apiErr, ok := err.(*servererrors.ErrorResponse); ok {
				return apiErr.Send(c)
			} else {
				return servererrors.InternalErrorf("%v", err)
			}

		},
	})
	params := panel.New()
	if args.TreasuryHome != "" {
		params.TreasuryHome = paths.TreasuryHome(args.TreasuryHome)
	}
	if args.BinaryDir != "" {
		params.BinaryDir = args.BinaryDir
	}
	if args.PanelDir != "" {
		params.PanelDir = paths.PanelHome(args.PanelDir)
	}
	if args.ApiKey != "" {
		params.ApiKey = args.ApiKey
	}
	if args.SupervisorHome != "" {
		params.SupervisorHome = paths.SupervisorHome(args.SupervisorHome)
	}
	if args.BackupDir != "" {
		params.BackupDir = args.BackupDir
	}
	if args.Connector {
		params.Connector = args.Connector
	}
	if args.ApiNode {
		params.ApiNode = args.ApiNode
	}
	if args.TreasuryUser != "" {
		params.TreasuryUser = args.TreasuryUser
	}
	// default otel collection to true
	params.OtelEnabled = true

	existingPanel, exists, err := loadPanel(params.PanelDir)
	if err != nil {
		slog.Error("failed to load panel", "error", err)
	}
	if exists {
		// load latest panel configuration
		slog.Info("loading existing panel", "path", params.PanelDir.PanelFile())

		// paranoid checks :)
		if existingPanel.SupervisorHome == "" {
			slog.Warn("supervisor home is not set in existing panel, using previous value for it")
			existingPanel.SupervisorHome = params.SupervisorHome
		}
		if existingPanel.TreasuryHome == "" {
			slog.Warn("treasury home is not set in existing panel, using previous value for it")
			existingPanel.TreasuryHome = params.TreasuryHome
		}
		if existingPanel.BinaryDir == "" {
			slog.Warn("binary directory is not set in existing panel, using previous value for it")
			existingPanel.BinaryDir = params.BinaryDir
		}
		if existingPanel.BackupDir == "" {
			slog.Warn("backup directory is not set in existing panel, using previous value for it")
			existingPanel.BackupDir = params.BackupDir
		}
		if existingPanel.TreasuryUser == "" {
			slog.Warn("treasury user is not set in existing panel, using previous value for it")
			existingPanel.TreasuryUser = params.TreasuryUser
		}
		params = existingPanel

	} else {
		// write out so information is saved + env is accessible by systemd services
		slog.Info("saving new panel", "path", params.PanelDir.PanelFile())
		err = panel.Save(params)
		if err != nil {
			slog.Error("failed to save panel", "error", err)
		}
	}

	identity, exists, err := loadIdentity(params.PanelDir)
	if err != nil {
		slog.Error("failed to load identity", "error", err)
	}
	if !exists {
		identity, err = age.GenerateX25519Identity()
		if err != nil {
			panic(err)
		}
		err = saveIdentity(params.PanelDir, identity)
		if err != nil {
			slog.Error("failed to save identity", "error", err)
		}
	}

	// Add the binary path to the PATH environment variable, so any `exec`'d processes
	// will also have the right path to find the binaries (e.g. `cord` exec'ing to `signer`).
	path := os.Getenv("PATH")
	if path != "" {
		path = path + ":"
	}
	path += args.BinaryDir
	os.Setenv("PATH", path)

	return &Server{
		app,
		params,
		identity,
		args,
	}
}

// setupRoutes configures all the routes for the server
func (s *Server) setupRoutes() {
	// Add CORS middleware for development
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000,https://localhost:3000",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Add middleware
	s.app.Use(func(c *fiber.Ctx) error {
		fmt.Printf("[%s] %s\n", c.Method(), c.Path())
		return c.Next()
	})

	// Add panic recovery middleware
	s.app.Use(func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				debug.PrintStack()
				c.Status(500).SendString("Internal Server Error")
				err = servererrors.InternalErrorf("internal server error, please contact Cordial Systems developers")
			}
		}()
		return c.Next()
	})

	// Health check
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	api := s.app.Group("/v1")

	// Root endpoint
	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to the Panel Server",
			"status":  "running",
		})
	})
	endpointHandler := endpoints.NewEndpoints(s.params, s.identity)

	// POST /activate/api-key {api-key}
	// - test API key, then store it
	// - indicate if this treasury has connector enabled or not.
	api.Post("/activate/api-key", endpointHandler.ActivateApiKey)

	// POST /activate/binaries
	// - download binaries needed (treasury, cord, signer)
	api.Post("/activate/binaries", endpointHandler.ActivateBinaries)

	// POST /activate/network
	// - Enrolls the VPN
	api.Post("/activate/network", endpointHandler.ActivateNetwork)

	// - Configure the backup keys (warning: this may only be done once, otherwise have to start over.)
	api.Post("/activate/backup", endpointHandler.ActivateBackup)

	// - Configure OTEL collection (optional, defaults to true)
	api.Post("/activate/otel", endpointHandler.ActivateOtel)

	// - Check if the node is activated
	// api.Post("/activate/check", endpointHandler.IsActivated)

	// POST /treasury
	// - Download treasury image, if there isn't an image set already.
	// - Generate a treasury, if not yet generated (error if there is a treasury already).
	// - Update the admin resource
	// - may use backup API to download a snapshot
	api.Post("/treasury", endpointHandler.GenerateTreasury)

	// POST /treasury/snapshot
	// - generate a snapshot on demand (treasury must be completed)
	// - upload using backup API
	// api.Delete("/treasury/snapshot", endpointHandler.SnapshotTreasury)

	// Also have DELETE /treasury
	// - Deletes locally
	api.Delete("/treasury", endpointHandler.DeleteTreasury)

	// POST /treasury/complete
	// - Try to complete the treasury, returning `FAILED_PRECONDITION` if not all nodes are updated yet.
	// - Run peer setup user Node.host fields
	// - Run `cord genesis complete`
	// - Run `cord use-image ...`
	api.Post("/treasury/complete", endpointHandler.PostTreasuryComplete)
	api.Post("/treasury/complete-and-start", endpointHandler.PostTreasuryCompleteAndStart)

	// Get the treasury init file produced (not really needed, but for debugging)
	api.Get("/treasury/init", endpointHandler.GetTreasuryInit)

	// Get the panel settings.  Some of these are set from activation endpoints, others are set by VM/cli-args.
	api.Get("/panel", endpointHandler.GetPanel)
	// EAR management
	api.Put("/panel/ear", endpointHandler.SetEncryptionAtRest)
	api.Delete("/panel/ear", endpointHandler.DeleteEncryptionAtRest)

	/// USER MANAGEMENT
	// - Available users can be seen from the admin API
	// - The user can select users by posting them to the Panel resource
	// Transparently forwards to the admin API
	api.Get("/admin/users", endpointHandler.AdminUsers)

	// This activates the target blueprint
	// - Generates /etc/panel/blueprint.csl
	//   - {"blueprint": "demo" | "deployment"}
	//   	- "demo" -> demo blueprint, no users configured.
	//   	- "deployment" -> deployment blueprint, users configured.
	// - Starts blueprint.service (runs `treasury script -f /etc/panel/blueprint.csl`)
	// - Waiting for blueprint.service to finish
	// (can see output from /services/blueprint.service/logs)
	api.Post("/panel/seal", endpointHandler.SealPanel)

	// Get treasury resource from the node's treasury API
	api.Get("/treasury", endpointHandler.GetTreasury)
	// Get treasury health endpoint
	api.Get("/treasury/healthy", endpointHandler.GetTreasuryHealth)
	// Get configured treasury image
	api.Get("/treasury/image", endpointHandler.GetSupervisorImage)
	// Set treasury image, possibly overriding the existing image
	api.Post("/treasury/image", endpointHandler.PostSupervisorImage)
	// Delete the current treasury image (possibly reseting to the treasury initial version if regenerating the treasury.)
	api.Delete("/treasury/image", endpointHandler.DeleteSupervisorImage)
	// Get treasury config ($TREASURY_HOME/treasury.toml)
	api.Get("/treasury/config", endpointHandler.GetTreasuryConfig)
	// Useful for re-syncing the peers based on the admin API, or manual input.
	api.Post("/treasury/peers/sync", endpointHandler.SyncTreasuryPeers)

	// Not needed, but may be useful for debugging.
	api.Get("/exists", endpointHandler.Stat)
	api.Get("/ls", endpointHandler.Ls)

	// Download binaries + verify signatures
	api.Get("/binaries/:binary", endpointHandler.GetBinaryVersion)
	api.Get("/binaries", endpointHandler.GetBinaryVersions)
	api.Post("/binaries/:binary/versions/:version/install", endpointHandler.Install)

	// Generate start/stop/restart service endpoints
	// Available services:
	// - treasury.service
	// - start-treasury.service (keeps trying to complete install, waiting on all peers to activate).
	// - docker.service (for troubleshooting)
	// - panel.service (readonly)
	api.Post("/services/:service/:action", endpointHandler.UpdateService)
	api.Get("/services/:service", endpointHandler.GetService)
	api.Get("/services/:service/logs", endpointHandler.GetServiceLogs)
	api.Get("/containers/:container/logs", endpointHandler.GetContainerLogs)
	api.Get("/services", endpointHandler.ListServices)
	api.Get("/logs", endpointHandler.Logs) // redundant with service logs if panel is running via systemd

	///// BOOTC - for managing VM updates
	// Get bootc status (`bootc status --format json`)
	api.Get("/bootc/status", endpointHandler.BootcStatus)
	// Check if an update is available, but don't stage it (`bootc upgrade --check`)
	api.Post("/bootc/upgrade/check", endpointHandler.BootcCheck)
	// Stage an update for next boot (`bootc upgrade`)
	api.Post("/bootc/upgrade/stage", endpointHandler.BootcStage)
	// Apply an update, rebooting the VM (`bootc upgrade --apply`)
	api.Post("/bootc/upgrade/apply", endpointHandler.BootcUpgradeApply)
	// Stage a rollback for next boot (`bootc rollback`)
	api.Post("/bootc/rollback/stage", endpointHandler.BootcRollbackStage)
	// Apply a rollback, rebooting the VM (`bootc rollback --apply`)
	api.Post("/bootc/rollback/apply", endpointHandler.BootcRollbackApply)

	// Backup / recovery
	api.Get("/s3/objects", endpointHandler.ListObjects)
	api.Get("/s3/object", endpointHandler.DownloadObject)

	// import a snapshot
	api.Put("/backup/snapshot/:id", endpointHandler.UploadSnapshot)
	// generate a snapshot
	api.Post("/backup/snapshot/:id", endpointHandler.TakeSnapshot)
	// restore from a (uploaded) snapshot
	api.Post("/backup/restore", endpointHandler.RestoreFromSnapshot)
	// restore missing keys
	api.Post("/backup/restore-missing-keys", endpointHandler.RestoreMissingKeys)

	// TODO:
	// - More endpoints to manage vpn/netbird?

	api.Get("/test-panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})

	// Serve static files from web/out directory
	s.app.Static("/", s.Options.WebDir)

	s.app.Use(func(c *fiber.Ctx) error {
		return servererrors.NotFoundf("endpoint for %s %s not found", c.Method(), c.Path())
	})

}

// Start begins listening for requests
func (s *Server) Start() error {
	s.setupRoutes()

	fmt.Printf("Starting server on %s\n", s.ListenAddr)
	return s.app.Listen(s.ListenAddr)
}
