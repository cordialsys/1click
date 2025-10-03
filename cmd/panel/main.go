package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/cordialsys/panel/pkg/bak"
	"github.com/cordialsys/panel/pkg/client"
	"github.com/cordialsys/panel/pkg/paths"
	"github.com/cordialsys/panel/pkg/plog"
	"github.com/cordialsys/panel/pkg/secret"
	"github.com/cordialsys/panel/server"
	"github.com/cordialsys/panel/server/panel"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

func StartCmd() *cobra.Command {
	var listenAddr string
	var apiKeyRef string

	var treasuryHome string
	var binaryDir string
	var panelDir string
	var supervisorHome string
	var triples uint64
	var backupDir string

	var connector bool
	var apiNode bool
	var treasuryUser string
	var webDir string

	var cmd = &cobra.Command{
		Use:          "start",
		Short:        "Start the panel server",
		SilenceUsage: true,

		RunE: func(cmd *cobra.Command, args []string) error {
			apiSecret := secret.Secret(apiKeyRef)
			apiKey, err := apiSecret.Load()
			if err != nil {
				return fmt.Errorf("failed to load API key: %v", err)
			}

			srv := server.New(server.Options{
				ListenAddr:     listenAddr,
				TreasuryHome:   treasuryHome,
				BinaryDir:      binaryDir,
				PanelDir:       panelDir,
				ApiKey:         apiKey,
				SupervisorHome: supervisorHome,
				Triples:        triples,
				BackupDir:      backupDir,
				Connector:      connector,
				ApiNode:        apiNode,
				TreasuryUser:   treasuryUser,
				WebDir:         webDir,
			})
			return srv.Start()
		},
	}

	// Add the listen flag to the start command
	cmd.Flags().StringVarP(&listenAddr, "listen", "l", ":7666", "Address to listen on")
	cmd.Flags().StringVar(&apiKeyRef, "api-key", "env:TREASURY_API_KEY", "API key secret reference")

	cmd.Flags().StringVar(&panelDir, "panel-dir", "", "Panel directory override")
	cmd.Flags().StringVar(&treasuryHome, "treasury-home", "", "Treasury home directory override")
	cmd.Flags().StringVar(&binaryDir, "binary-dir", "", "Binary directory override")
	cmd.Flags().StringVar(&supervisorHome, "supervisor-home", "", "Supervisor home directory override")
	cmd.Flags().StringVar(&backupDir, "backup-dir", "", "Backup directory override")
	cmd.Flags().StringVar(&treasuryUser, "treasury-user", "cordial", "Treasury user to run treasury service as")

	cmd.Flags().Uint64Var(&triples, "triples", 0, "Number of triples to generate")

	cmd.Flags().BoolVar(&connector, "connector", false, "Enable connector")
	cmd.Flags().BoolVar(&apiNode, "api-node", false, "Run as an API node")
	cmd.Flags().StringVar(&webDir, "web-dir", "./web/out", "Web directory override")

	return cmd
}

func ActivateCmd() *cobra.Command {
	var apiKeyRef string
	var connector bool
	// var apiNode bool
	var remote string
	var baks []string
	var version string
	var noOtel bool

	var cmd = &cobra.Command{
		Use:          "activate",
		Short:        "Activate the panel server",
		SilenceUsage: true,

		RunE: func(cmd *cobra.Command, args []string) error {

			var apiKey string
			var err error

			if apiKeyRef == "" {
				var input string
				for input == "" {
					fmt.Print("Enter Activation API key: ")
					fmt.Scanln(&input)
					input = strings.TrimSpace(input)
					apiKey = input
				}
			} else {
				secretMaybe := secret.Secret(apiKeyRef)
				if _, ok := secretMaybe.Type(); !ok {
					// treat as literal
					apiKey = apiKeyRef
				} else {
					apiKey, err = secretMaybe.Load()
				}
				if err != nil {
					return fmt.Errorf("failed to load API key: %v", err)
				}
				if apiKey == "" {
					return fmt.Errorf("API key reference resolved to an empty value")
				}
			}

			// 1. activate the API key
			fmt.Println("Activating API key...")
			var connectorInput *bool
			if cmd.Flags().Lookup("connector").Changed {
				// only pass if specified on CLI, so panel will otherwise default to the admin API.
				connectorInput = &connector
			}
			err = panelClient.ActivateApiKey(apiKey, connectorInput)
			if err != nil {
				return err
			}
			fmt.Println("API key activated.")

			panelInfo, err := panelClient.GetPanel()
			if err != nil {
				return err
			}

			// 2. activate the backup
			if len(panelInfo.Baks) == 0 {
				if len(baks) <= 0 {
					sk := bak.GenerateEncryptionKey()
					recipient := sk.Recipient()

					fmt.Println("# Generating new backup key...")
					fmt.Println("# You must save this somewhere safe")
					fmt.Println("------- SECRET BACKUP PHRASE -------")
					fmt.Println(strings.Join(sk.Words(), " "))
					fmt.Println("------------------------------------")
					fmt.Println("Public key:", recipient.String())
					fmt.Println()
					fmt.Print("Confirm (y/n): ")
					var confirm string
					fmt.Scanln(&confirm)
					if strings.ToLower(confirm) != "y" {
						return fmt.Errorf("cancelled")
					}
					fmt.Println()
					baks = append(baks, recipient.String())
				}
				fmt.Println("Activating backup...")
				bakObjs := make([]panel.Bak, len(baks))
				for i := range baks {
					bakObjs[i] = panel.Bak{
						Key: baks[i],
					}
				}
				err = panelClient.ActivateBackup(bakObjs)
				if err != nil {
					return err
				}
				fmt.Println("Backup activated.")
			} else {
				fmt.Println("Backup already activated.")
				baks := []string{}
				for _, bak := range panelInfo.Baks {
					baks = append(baks, bak.Key)
				}
				fmt.Println("Backup keys: ", strings.Join(baks, ", "))
				fmt.Println("To reset, run: `panel reset --force` and re-run activation.")
			}

			if noOtel {
				// 3. activate the otel
				fmt.Println("Disabling OTEL...")
				err = panelClient.ActivateOtel(false)
				if err != nil {
					return err
				}
			}

			// 3. activate the binaries
			fmt.Println("Installing production binaries...")
			err = panelClient.ActivateBinaries(client.ActivateBinariesOptions{
				Version: version,
			})
			if err != nil {
				return err
			}
			fmt.Println("Binaries installed.")

			// 4. activate the network
			fmt.Println("Activating network...")
			err = panelClient.ActivateNetwork()
			if err != nil {
				return err
			}
			fmt.Println("Network activated.")

			treasuryService, err := panelClient.GetService("treasury.service")
			if err != nil {
				return err
			}
			if treasuryService.ActiveState != "active" {
				// 5. Generate the treasury node
				err = panelClient.GenerateTreasury()
				if err != nil {
					return err
				}
				fmt.Println("Treasury generated.")

				// 6. Try to complete the treasury node
				fmt.Println("Completing treasury node with peer information...")
				err = panelClient.CompleteTreasury()
				if err != nil {
					// 6.a. Start the 'start-treasury' service, which keeps retrying to complete the treasury node.
					// This allows the user to not have to SSH in again (this is 1-click setup...).
					fmt.Printf("Treasury not yet complete (%v), will try continue trying...\n", err)
					err = panelClient.UpdateService("start-treasury.service", "start")
					if err != nil {
						return err
					}
					fmt.Println("Once peers have updated, this treasury node will start automatically.")
				} else {
					// 6.b start the treasury service directly.
					// Ideally all of the nodes should be setup by the time customer runs this command.
					// So we can just start the treasury service directly.
					fmt.Println("Starting treasury....")
					err = panelClient.UpdateService("treasury.service", "enable")
					if err != nil {
						return err
					}
					err = panelClient.UpdateService("treasury.service", "restart")
					if err != nil {
						return err
					}
					fmt.Println("Treasury started.")
				}
			} else {
				fmt.Println("Treasury already running.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&apiKeyRef, "api-key", "", "API key secret reference")
	cmd.Flags().StringVar(&remote, "url", "http://localhost:7666", "URL of the panel server")
	cmd.Flags().BoolVar(&connector, "connector", false, "Enable connector")
	cmd.Flags().StringSliceVar(&baks, "bak", []string{}, "Backup key(s)")
	cmd.Flags().StringVar(&version, "version", "latest", "Version of production binaries to install")
	cmd.Flags().BoolVar(&noOtel, "no-otel", false, "Disable OTEL collection")

	return cmd
}

func ResetCmd() *cobra.Command {
	var force bool
	var panelHome string
	var cmd = &cobra.Command{
		Use:          "reset",
		Short:        "Reset the panel server",
		Long:         "This is required if you want to change the backup keys for this node.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Resetting panel server...")
			if !force {
				fmt.Println("This will delete the panel server backup keys and API keys, do you want to continue? (y/n)")
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
					return fmt.Errorf("cancelled")
				}
			}
			err := os.RemoveAll(panelHome)
			if err != nil {
				return err
			}
			fmt.Println("Panel server reset.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force reset")
	cmd.Flags().StringVar(&panelHome, "panel-home", "/etc/panel", "Panel home directory")
	return cmd
}

func DeleteTreasuryCmd() *cobra.Command {
	var remote string
	var force bool
	var includeSupervisor bool
	var cmd = &cobra.Command{
		Use:          "delete-treasury",
		Short:        "Delete the treasury node",
		Long:         "This is required to recreate the treasury node.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Println("This will permanently delete the treasury node locally, do you want to continue? (y/n)")
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
					return fmt.Errorf("cancelled")
				}
			}
			fmt.Println("Stopping treasury...")
			_ = panelClient.UpdateService("treasury.service", "stop")
			_ = panelClient.UpdateService("start-treasury.service", "stop")

			fmt.Println("Deleting treasury...")
			err := panelClient.DeleteTreasury(includeSupervisor)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&remote, "url", "http://localhost:7666", "URL of the panel server")
	cmd.Flags().BoolVar(&force, "force", false, "Skip any confirmation")
	cmd.Flags().BoolVar(&includeSupervisor, "supervisor", false, "Include deleting supervisor config (reset to using initial image, no EAR)")
	return cmd
}

func SyncTreasuryPeersCmd() *cobra.Command {
	var remote string
	var cmd = &cobra.Command{
		Use:          "sync-peers",
		Short:        "Sync the treasury peers",
		Long:         "In case DNS names change, you can re-sync the configured peers.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := panelClient.SyncTreasuryPeers()
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&remote, "url", "http://localhost:7666", "URL of the panel server")
	return cmd
}

func DoTreasuryConfigSync(panelInfo *panel.Panel, treasuryHome paths.TreasuryHome) error {
	treasuryConfig := map[string]interface{}{}

	treasuryConfigFile, err := os.Open(treasuryHome.TreasuryConfig())
	if err != nil {
		return fmt.Errorf("failed to open treasury config: %v", err)
	}
	defer treasuryConfigFile.Close()

	treasuryConfigBz, err := io.ReadAll(treasuryConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read treasury config: %v", err)
	}
	treasuryConfigFileStat, err := treasuryConfigFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat treasury config: %v", err)
	}
	_ = treasuryConfigFile.Close()

	err = toml.Unmarshal(treasuryConfigBz, &treasuryConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal treasury config: %v", err)
	}

	// Modify only the backup.bak entry
	backupConfig, ok := treasuryConfig["backup"].(map[string]interface{})
	if !ok {
		// add backup section if not present
		treasuryConfig["backup"] = map[string]interface{}{}
	}
	backupConfig = treasuryConfig["backup"].(map[string]interface{})

	backupConfig["bak"] = panelInfo.Baks
	treasuryConfigBz, err = toml.Marshal(treasuryConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal treasury config: %v", err)
	}
	err = os.WriteFile(treasuryHome.TreasuryConfig(), treasuryConfigBz, treasuryConfigFileStat.Mode())
	if err != nil {
		return fmt.Errorf("failed to write treasury config: %v", err)
	}
	return nil
}

func DoSupervisorConfigSync(panelInfo *panel.Panel, supervisorHome paths.SupervisorHome) error {
	supervisorConfig := map[string]interface{}{}

	// create file if it doesn't exist
	if _, err := os.Stat(supervisorHome.ConfigFile()); os.IsNotExist(err) {
		f, err := os.Create(supervisorHome.ConfigFile())
		if err != nil {
			return fmt.Errorf("failed to create supervisor config: %v", err)
		}
		_ = f.Close()
	}

	supervisorConfigFile, err := os.Open(supervisorHome.ConfigFile())
	if err != nil {
		return fmt.Errorf("failed to open supervisor config: %v", err)
	}
	defer supervisorConfigFile.Close()

	supervisorConfigBz, err := io.ReadAll(supervisorConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read supervisor config: %v", err)
	}
	supervisorConfigFileStat, err := supervisorConfigFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat treasury config: %v", err)
	}
	err = toml.Unmarshal(supervisorConfigBz, &supervisorConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal supervisor config: %v", err)
	}

	supervisorConfig["ear_secret"] = panelInfo.EarSecret
	supervisorConfigBz, err = toml.Marshal(supervisorConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal supervisor config: %v", err)
	}
	err = os.WriteFile(supervisorHome.ConfigFile(), supervisorConfigBz, supervisorConfigFileStat.Mode())
	if err != nil {
		return fmt.Errorf("failed to write supervisor config: %v", err)
	}
	return nil
}

func SyncConfigCmd() *cobra.Command {
	var _panelDir string
	var _treasuryHome string
	var _supervisorHome string
	var cmd = &cobra.Command{
		Use:          "sync-config",
		Aliases:      []string{"sync-backup-keys", "sync-config"},
		Short:        "Write the configured backup keys in the panel to the current Treasury config",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			panelDir := paths.PanelHome(_panelDir)
			treasuryHome := paths.TreasuryHome(_treasuryHome)
			supervisorHome := paths.SupervisorHome(_supervisorHome)
			panelInfo, err := panel.Load(panelDir)
			if err != nil {
				return fmt.Errorf("failed to load panel data: %v", err)
			}
			if len(panelInfo.Baks) == 0 {
				return fmt.Errorf("no backup keys configured on panel")
			}

			err = DoTreasuryConfigSync(panelInfo, treasuryHome)
			if err != nil {
				return fmt.Errorf("failed to sync treasury config: %v", err)
			}

			err = DoSupervisorConfigSync(panelInfo, supervisorHome)
			if err != nil {
				return fmt.Errorf("failed to sync supervisor config: %v", err)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&_panelDir, "panel-dir", string(panel.New().PanelDir), "Panel directory override")
	cmd.Flags().StringVar(&_treasuryHome, "treasury-home", string(panel.New().TreasuryHome), "Treasury home directory override")
	cmd.Flags().StringVar(&_supervisorHome, "supervisor-home", string(panel.New().SupervisorHome), "Supervisor home directory override")
	return cmd
}

func GenerateCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:          "bak",
		Short:        "Generate a backup key",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("# You must save this somewhere safe")
			sk := bak.GenerateEncryptionKey()
			recipient := sk.Recipient()

			fmt.Println(strings.Join(sk.Words(), " "))
			fmt.Println(recipient.String())

			return nil
		},
	}
	return cmd
}

func HealthyCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:          "health",
		Aliases:      []string{"healthy"},
		Short:        "Check if the treasury node is healthy",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			health, err := panelClient.TreasuryHealth()
			if err != nil {
				return err
			}
			fmt.Println(string(health))
			return nil
		},
	}
	return cmd
}

var panelClient *client.Client

func main() {
	var verbose int
	var quiet bool
	var remote string
	var rootCmd = &cobra.Command{
		Use:   "panel",
		Short: "Panel server application",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			level := slog.LevelDebug
			switch verbose {
			case 0:
				level = slog.LevelInfo
			case 1:
				level = slog.LevelDebug
			default:
				level = slog.LevelDebug
			}
			if quiet {
				level = slog.LevelError
			}
			slog.SetLogLoggerLevel(level)
			plog.Init(level)
			remoteUrl, err := url.Parse(remote)
			if err != nil {
				return fmt.Errorf("failed to parse --url: %v", err)
			}
			panelClient = client.NewClient(remoteUrl)
			return nil
		},
	}
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Verbosity level")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Quiet mode")
	rootCmd.Flags().StringVar(&remote, "url", "http://localhost:7666", "URL of the panel server")

	startCmd := StartCmd()

	// Add commands to root
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(ActivateCmd())
	rootCmd.AddCommand(ResetCmd())
	rootCmd.AddCommand(GenerateCmd())
	rootCmd.AddCommand(DeleteTreasuryCmd())
	rootCmd.AddCommand(SyncTreasuryPeersCmd())
	rootCmd.AddCommand(SyncConfigCmd())
	rootCmd.AddCommand(HealthyCmd())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
