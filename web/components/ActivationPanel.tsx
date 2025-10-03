import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/router";
import ActivationTab from "./ActivationTab";
import StatusTab from "./StatusTab";
import AdvancedTab from "./AdvancedTab";
import BackupTab from "./BackupTab";
import BackupRestoreTab from "./BackupRestoreTab";
import { PanelInfo, HealthInfo } from "../utils/types";
import { panelApiClient } from "../utils/panel-client";

export default function ActivationPanel() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<
    "activation" | "status" | "backup" | "backup-restore" | "advanced"
  >("activation");
  const [activationComplete, setActivationComplete] = useState(false);
  const [panelInfo, setPanelInfo] = useState<PanelInfo | null>(null);
  const [loadingPanelInfo, setLoadingPanelInfo] = useState(true);

  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });
  const [currentAction, setCurrentAction] = useState("");
  const [autoRunning, setAutoRunning] = useState(false);
  const [showResetConfirmation, setShowResetConfirmation] = useState(false);

  // Status tab state
  const [treasuryService, setTreasuryService] = useState<any>(null);
  const [startTreasuryService, setStartTreasuryService] = useState<any>(null);
  const [treasuryHealth, setTreasuryHealth] = useState<HealthInfo | null>(null);
  const [loadingServices, setLoadingServices] = useState(false);
  const panelInfoIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const fetchServiceStatus = async () => {
    setLoadingServices(true);
    try {
      // Fetch treasury service status
      try {
        const treasuryResponse = await panelApiClient.getService(
          "treasury.service"
        );
        setTreasuryService(treasuryResponse);
      } catch (error) {
        console.error("Failed to fetch treasury service:", error);
      }

      // Fetch start-treasury service status
      try {
        const startTreasuryResponse = await panelApiClient.getService(
          "start-treasury.service"
        );
        setStartTreasuryService(startTreasuryResponse);
      } catch (error) {
        console.error("Failed to fetch start-treasury service:", error);
      }

      // Fetch treasury health if treasury service is active
      try {
        const treasuryHealth = await panelApiClient.getTreasuryHealth(true);
        setTreasuryHealth(treasuryHealth);
      } catch (error) {
        console.error("Failed to fetch treasury health:", error);
      }
    } finally {
      setLoadingServices(false);
    }
  };

  const fetchPanelInfo = async () => {
    try {
      const data = await panelApiClient.getPanelInfo();
      setPanelInfo(data);

      // Update UI state based on panel info
      if (data.otel_enabled !== undefined) {
        // This would need to be passed to ActivationTab if needed
      }

      // Check activation status based on panel state
      if (
        data.state === "generated" ||
        data.state === "active" ||
        data.state === "sealed" ||
        data.state === "stopped"
      ) {
        setActivationComplete(true);
      } else {
        setActivationComplete(false);
      }
    } catch (error) {
      console.error("Failed to fetch panel info:", error);
      setActivationComplete(false);
    } finally {
      setLoadingPanelInfo(false);
    }
  };

  const resetTreasury = async () => {
    setLoading(true);
    setCurrentAction("Resetting treasury...");
    try {
      await panelApiClient.deleteTreasury();
      setStatus({
        type: "success",
        message: "Treasury reset successfully. You can now activate again.",
      });
      setActivationComplete(false);
      setPanelInfo((prev) =>
        prev
          ? {
              ...prev,
              baks: [],
              api_key_id: undefined,
              node_id: undefined,
              treasury_id: undefined,
              state: "inactive",
            }
          : null
      );
    } catch (error) {
      setStatus({
        type: "error",
        message: `Failed to reset treasury: ${error}`,
      });
    } finally {
      setLoading(false);
      setCurrentAction("");
      setShowResetConfirmation(false);
    }
  };

  // Sync activeTab with URL
  useEffect(() => {
    const tab = router.query.tab as string;
    if (
      tab &&
      ["activation", "status", "backup", "backup-restore", "advanced"].includes(
        tab
      )
    ) {
      setActiveTab(
        tab as
          | "activation"
          | "status"
          | "backup"
          | "backup-restore"
          | "advanced"
      );
    } else if (router.isReady && !tab) {
      // Set default tab if no tab specified
      router.replace("?tab=activation", undefined, { shallow: true });
    }
  }, [router.query.tab, router.isReady]);

  useEffect(() => {
    fetchPanelInfo();
  }, [activationComplete, status]);

  // Set up automatic panel info refresh every 1 second
  useEffect(() => {
    // Initial fetch
    fetchPanelInfo();

    // Set up interval
    panelInfoIntervalRef.current = setInterval(() => {
      fetchPanelInfo();
    }, 1000);

    // Cleanup on unmount
    return () => {
      if (panelInfoIntervalRef.current) {
        clearInterval(panelInfoIntervalRef.current);
      }
    };
  }, []); // Empty dependency array to run only once

  const handleApiKeyActivation = async (
    apiKey: string,
    generatedBackupKeys: {
      mnemonic: string;
      ageRecipient: string;
      nickname: string;
    }[],
    hasUnsavedBackupKeys: boolean,
    otelEnabled: boolean,
    binaryVersion: string,
    useDemoPolicy: boolean,
    hasOpenImportForms: boolean
  ) => {
    // Check for open import forms first
    if (hasOpenImportForms) {
      setStatus({
        type: "error",
        message:
          "Please complete or cancel any open Import Backup Key forms before activating.",
      });
      return;
    }

    if (!apiKey.trim()) {
      setStatus({ type: "error", message: "Please enter an API key" });
      return;
    }

    // If state is inactive, backup keys are required
    const backupKeysLocked = panelInfo?.state !== "inactive";
    if (!backupKeysLocked && generatedBackupKeys.length === 0) {
      setStatus({
        type: "error",
        message:
          "Please generate and save at least one backup key before continuing",
      });
      return;
    }

    // Check if there are any unsaved backup keys when keys are not locked
    if (!backupKeysLocked && hasUnsavedBackupKeys) {
      setStatus({
        type: "error",
        message:
          "Please save all generated backup keys before continuing, or remove unsaved keys",
      });
      return;
    }

    await runAutomaticActivation(
      apiKey,
      generatedBackupKeys,
      otelEnabled,
      binaryVersion
    );
  };

  const runAutomaticActivation = async (
    apiKey: string,
    generatedBackupKeys: {
      mnemonic: string;
      ageRecipient: string;
      nickname: string;
    }[],
    otelEnabled: boolean,
    binaryVersion: string
  ) => {
    setAutoRunning(true);
    setLoading(true);

    try {
      // Step 1: API Key
      setCurrentAction("Activating API key...");
      const apiKeyBody: any = { api_key: apiKey };
      await panelApiClient.activateApiKey(apiKeyBody);
      setStatus({ type: "success", message: "API key activated successfully" });

      await new Promise((resolve) => setTimeout(resolve, 500));

      // Step 2: Download Binaries
      setCurrentAction("Downloading binaries...");
      const binariesResponse = await panelApiClient.downloadBinaries(
        binaryVersion
      );
      if (!binariesResponse.ok) {
        const errorText = await binariesResponse.text();
        throw new Error(`Failed to download binaries: ${errorText}`);
      }
      setStatus({
        type: "success",
        message: "Binaries downloaded successfully",
      });

      await new Promise((resolve) => setTimeout(resolve, 500));

      // Step 3: Network Setup
      setCurrentAction("Setting up network connection...");
      await panelApiClient.setupNetwork();
      setStatus({
        type: "success",
        message: "Network configured successfully",
      });

      await new Promise((resolve) => setTimeout(resolve, 500));

      const backupKeysLocked = panelInfo?.state !== "inactive";

      // Step 4: Backup Keys (if new ones provided and keys are not locked)
      if (generatedBackupKeys.length > 0 && !backupKeysLocked) {
        setCurrentAction("Configuring backup keys...");
        const keys = generatedBackupKeys.map((key) => ({
          key: key.ageRecipient,
          id: key.nickname || undefined,
        }));
        await panelApiClient.configureBackup({
          baks: keys,
        });
        setStatus({
          type: "success",
          message: "Backup keys configured successfully",
        });

        await new Promise((resolve) => setTimeout(resolve, 500));
      } else if (backupKeysLocked) {
        setStatus({ type: "success", message: "Using existing backup keys" });
        await new Promise((resolve) => setTimeout(resolve, 500));
      }

      // Step 5: OTEL Setup
      setCurrentAction("Configuring observability...");
      await panelApiClient.configureOtel({
        enabled: otelEnabled,
      });
      setStatus({
        type: "success",
        message: "Observability configured successfully",
      });

      await new Promise((resolve) => setTimeout(resolve, 500));

      // Step 6: Check treasury service status and generate if needed
      setCurrentAction("Checking treasury status...");
      try {
        const treasuryService = await panelApiClient.getService(
          "treasury.service"
        );

        if (treasuryService.active_state !== "active") {
          // Generate treasury
          setCurrentAction("Generating treasury...");
          await panelApiClient.generateTreasury();
          setStatus({
            type: "success",
            message: "Treasury generated successfully",
          });

          await new Promise((resolve) => setTimeout(resolve, 500));

          // Try to complete the treasury
          setCurrentAction("Completing treasury with peer information...");
          await panelApiClient.completeTreasury();

          setStatus({
            type: "success",
            message: "Treasury started successfully!",
          });
        } else {
          setStatus({
            type: "success",
            message: "Treasury already running",
          });
        }

        setStatus({
          type: "success",
          message: "All activation steps completed successfully!",
        });
        setCurrentAction("");
        setActivationComplete(true);
      } catch (treasuryError) {
        setStatus({
          type: "error",
          message: `Treasury setup failed: ${treasuryError}`,
        });
        setCurrentAction("");
      }
    } catch (error) {
      setStatus({ type: "error", message: `Activation failed: ${error}` });
      setCurrentAction("");
    } finally {
      setLoading(false);
      setAutoRunning(false);
    }
  };

  const renderTabContent = () => {
    if (activeTab === "activation") {
      return (
        <ActivationTab
          panelInfo={panelInfo}
          activationComplete={activationComplete}
          status={status}
          currentAction={currentAction}
          loading={loading}
          autoRunning={autoRunning}
          handleApiKeyActivation={handleApiKeyActivation}
        />
      );
    } else if (activeTab === "status") {
      return (
        <StatusTab
          panelInfo={panelInfo}
          loadingPanelInfo={loadingPanelInfo}
          treasuryService={treasuryService}
          startTreasuryService={startTreasuryService}
          treasuryHealth={treasuryHealth}
          loadingServices={loadingServices}
          fetchServiceStatus={fetchServiceStatus}
        />
      );
    } else if (activeTab === "backup") {
      return <BackupTab panelInfo={panelInfo} />;
    } else if (activeTab === "backup-restore") {
      return <BackupRestoreTab panelInfo={panelInfo} />;
    } else if (activeTab === "advanced") {
      return (
        <AdvancedTab
          loading={loading}
          showResetConfirmation={showResetConfirmation}
          setShowResetConfirmation={setShowResetConfirmation}
          resetTreasury={resetTreasury}
        />
      );
    }
  };

  if (loadingPanelInfo) {
    return (
      <div className="card">
        <div style={{ textAlign: "center", padding: "2rem" }}>
          <span className="loading"></span>
          Loading panel information...
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* Tab Navigation */}
      <div
        style={{
          display: "flex",
          borderBottom: "2px solid #e0e0e0",
          marginBottom: "2rem",
        }}
      >
        {[
          { key: "activation", label: "Activation" },
          { key: "status", label: "Status" },
          { key: "backup", label: "Backup" },
          { key: "backup-restore", label: "Restore" },
          { key: "advanced", label: "Advanced" },
        ].map((tab) => (
          <button
            key={tab.key}
            onClick={() => {
              router.push(`?tab=${tab.key}`, undefined, { shallow: true });
              if (tab.key === "status") {
                fetchServiceStatus();
              }
            }}
            style={{
              padding: "1rem 2rem",
              border: "none",
              background: activeTab === tab.key ? "#0070f3" : "transparent",
              color: activeTab === tab.key ? "white" : "#666",
              cursor: "pointer",
              borderRadius: "4px 4px 0 0",
              fontWeight: activeTab === tab.key ? "bold" : "normal",
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {renderTabContent()}
    </div>
  );
}
