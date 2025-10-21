import { useState } from "react";
import BackupKeyGenerator from "./BackupKeyGenerator";
import PanelStatus from "./PanelStatus";
import EncryptionAtRestTab from "./EncryptionAtRestTab";
import InitialUsersTab from "./InitialUsersTab";
import { PanelInfo } from "../utils/types";

interface Props {
  panelInfo: PanelInfo | null;
  activationComplete: boolean;
  status: {
    type: "success" | "error" | "info" | null;
    message: string;
  };
  currentAction: string;
  loading: boolean;
  autoRunning: boolean;
  handleApiKeyActivation: (
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
    hasOpenImportForms: boolean,
    skipNetwork: boolean
  ) => Promise<void>;
}

export default function ActivationTab({
  panelInfo,
  activationComplete,
  status,
  currentAction,
  loading,
  autoRunning,
  handleApiKeyActivation,
}: Props) {
  const [apiKey, setApiKey] = useState("");
  const [generatedBackupKeys, setGeneratedBackupKeys] = useState<
    { mnemonic: string; ageRecipient: string; nickname: string }[]
  >([]);
  const [hasUnsavedBackupKeys, setHasUnsavedBackupKeys] = useState(false);
  const [hasOpenImportForms, setHasOpenImportForms] = useState(false);
  const [otelEnabled, setOtelEnabled] = useState(true);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [binaryVersion, setBinaryVersion] = useState("latest");
  const [useDemoPolicy, setUseDemoPolicy] = useState(false);
  const [showEncryptionAtRest, setShowEncryptionAtRest] = useState(false);
  const [showInitialUsers, setShowInitialUsers] = useState(true);
  const [showApiKey, setShowApiKey] = useState(false);
  const [skipNetwork, setSkipNetwork] = useState(false);

  const handleActivation = async () => {
    await handleApiKeyActivation(
      apiKey,
      generatedBackupKeys,
      hasUnsavedBackupKeys,
      otelEnabled,
      binaryVersion,
      useDemoPolicy,
      hasOpenImportForms,
      skipNetwork
    );
  };

  return (
    <div>
      {/* Status Messages for Activation */}
      {status.type && (
        <div className={`status ${status.type}`}>{status.message}</div>
      )}

      {currentAction && (
        <div className="status info">
          <span className="loading"></span>
          {currentAction}
        </div>
      )}

      {/* <PanelStatus panelInfo={panelInfo} /> */}

      {!activationComplete && (
        <div className="card">
          <h3>Treasury Activation</h3>

          <div>
            <p>
              Enter your API key and configure backup keys to begin the
              automatic activation process.
            </p>

            <div className="form-group">
              <label htmlFor="apiKey">API Key:</label>
              <div
                style={{
                  position: "relative",
                  display: "inline-block",
                  width: "100%",
                }}
              >
                <input
                  id="apiKey"
                  type={showApiKey ? "text" : "password"}
                  value={apiKey}
                  onChange={(e) => {
                    setApiKey(e.target.value);
                  }}
                  placeholder="Enter your API key"
                  disabled={autoRunning}
                  style={{ paddingRight: "3rem" }}
                />
                <button
                  type="button"
                  onClick={(e) => {
                    setShowApiKey(!showApiKey);
                  }}
                  disabled={autoRunning}
                  style={{
                    position: "absolute",
                    right: "0.5rem",
                    top: "50%",
                    transform: "translateY(-50%)",
                    background: "none",
                    border: "none",
                    padding: "0.25rem",
                    fontSize: "0.875rem",
                    color: autoRunning ? "#999" : "#666",
                    cursor: autoRunning ? "not-allowed" : "pointer",
                    transition: "all 0.2s ease",
                    borderRadius: "4px",
                  }}
                  onMouseEnter={(e) => {
                    if (!autoRunning) {
                      e.currentTarget.style.backgroundColor = "#f8f9fa";
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (!autoRunning) {
                      e.currentTarget.style.backgroundColor = "transparent";
                    }
                  }}
                >
                  {showApiKey ? "Hide" : "Show"}
                </button>
              </div>
            </div>

            {/* Show existing backup keys if any and state is not inactive */}
            {panelInfo?.baks &&
            panelInfo.baks.length > 0 &&
            panelInfo.state !== "inactive" ? (
              <div className="form-group">
                <h4>Existing Backup Keys</h4>
                <p
                  style={{
                    color: "#666",
                    fontSize: "0.9rem",
                    marginBottom: "1rem",
                  }}
                >
                  Backup keys are already configured. These cannot be changed
                  without resetting the treasury.
                </p>
                {panelInfo.baks.map((key, index) => (
                  <div
                    key={index}
                    style={{
                      padding: "0.75rem",
                      border: "1px solid #28a745",
                      borderRadius: "4px",
                      marginBottom: "0.5rem",
                      backgroundColor: "#f0f8f0",
                      fontFamily: "monospace",
                      fontSize: "0.9rem",
                    }}
                  >
                    ✓ {key.key}
                  </div>
                ))}
              </div>
            ) : (
              <div className="form-group">
                <BackupKeyGenerator
                  onKeysGenerated={(keys, hasUnsaved, hasOpenForms) => {
                    setGeneratedBackupKeys(keys);
                    setHasUnsavedBackupKeys(hasUnsaved);
                    setHasOpenImportForms(hasOpenForms);
                  }}
                  disabled={autoRunning}
                />
              </div>
            )}

            {/* Advanced Options */}
            <div style={{ marginTop: "1.5rem" }}>
              <button
                type="button"
                onClick={() => setShowAdvanced(!showAdvanced)}
                disabled={autoRunning}
                style={{
                  background: "none",
                  border: "none",
                  color: autoRunning ? "#999" : "#0070f3",
                  cursor: autoRunning ? "not-allowed" : "pointer",
                  fontSize: "0.9rem",
                  textDecoration: "underline",
                }}
              >
                {showAdvanced ? "▼" : "▶"} Advanced Options
              </button>

              {showAdvanced && (
                <div
                  style={{
                    marginTop: "1rem",
                    padding: "1rem",
                    border: "1px solid #e0e0e0",
                    borderRadius: "4px",
                    backgroundColor: "#f8f9fa",
                  }}
                >
                  <div className="form-group">
                    <label htmlFor="binaryVersion">Binary Version:</label>
                    <input
                      id="binaryVersion"
                      type="text"
                      value={binaryVersion}
                      onChange={(e) => setBinaryVersion(e.target.value)}
                      placeholder="latest"
                      disabled={autoRunning}
                      style={{
                        width: "100%",
                        padding: "0.5rem",
                        border: "1px solid #ddd",
                        borderRadius: "4px",
                        opacity: autoRunning ? 0.6 : 1,
                      }}
                    />
                  </div>

                  <div style={{ marginBottom: "1rem" }}>
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                      }}
                    >
                      <input
                        type="checkbox"
                        id="otelEnabled"
                        checked={otelEnabled}
                        onChange={(e) => setOtelEnabled(e.target.checked)}
                        disabled={autoRunning}
                        style={{ marginRight: "8px" }}
                      />
                      <label
                        htmlFor="otelEnabled"
                        style={{
                          cursor: autoRunning ? "not-allowed" : "pointer",
                          fontSize: "0.95rem",
                          userSelect: "none",
                        }}
                      >
                        Enable OTEL (Observability)
                      </label>
                    </div>
                  </div>

                  <div style={{ marginBottom: "1rem" }}>
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                      }}
                    >
                      <input
                        type="checkbox"
                        id="skipNetwork"
                        checked={skipNetwork}
                        onChange={(e) => setSkipNetwork(e.target.checked)}
                        disabled={autoRunning}
                        style={{ marginRight: "8px" }}
                      />
                      <label
                        htmlFor="skipNetwork"
                        style={{
                          cursor: autoRunning ? "not-allowed" : "pointer",
                          fontSize: "0.95rem",
                          userSelect: "none",
                        }}
                      >
                        Skip Network
                      </label>
                    </div>
                  </div>
                </div>
              )}
            </div>

            <button
              className="btn"
              onClick={handleActivation}
              disabled={loading}
              style={{ marginTop: "1rem" }}
            >
              {loading && <span className="loading"></span>}
              {autoRunning ? "Activating..." : "Start Activation"}
            </button>
          </div>
        </div>
      )}

      {activationComplete && (
        <div className="card text-center">
          {panelInfo?.state === "active" ? (
            <>
              <h2>🎉 Activation Complete!</h2>
              <p>
                Your treasury panel has been successfully activated and is
                running.
              </p>
            </>
          ) : panelInfo?.state === "stopped" ? (
            <>
              <h2>⏸️ Treasury Stopped</h2>
              <p>
                Your treasury panel has been activated but is currently stopped.
              </p>
              <p style={{ fontSize: "0.9rem", color: "#666" }}>
                You can restart the treasury service from the Advanced tab.
              </p>
            </>
          ) : panelInfo?.state === "generated" ? (
            <>
              <h2>⏳ Treasury Generated</h2>
              <p>
                Your treasury has been generated and is waiting for peers to
                complete their activation.
              </p>
              <p style={{ fontSize: "0.9rem", color: "#666" }}>
                The treasury will start automatically once all peers are ready.
              </p>
            </>
          ) : (
            <>
              <h2>🎉 Activation Complete!</h2>
              <p>Your treasury panel has been successfully activated.</p>
            </>
          )}
          <div className="mt-4">
            <a
              href={`https://treasury.cordial.systems/${
                panelInfo?.treasury_id
                  ? `?treasury=${panelInfo.treasury_id}`
                  : ""
              }`}
              target="_blank"
              rel="noopener noreferrer"
              className="btn"
              style={{ marginRight: "0.5rem" }}
            >
              🚀 Launch App
            </a>
          </div>
        </div>
      )}

      {/* Initial Users Section - Show when panel is in active, generated, sealed, or stopped state */}
      {(panelInfo?.state === "active" ||
        panelInfo?.state === "generated" ||
        panelInfo?.state === "sealed" ||
        panelInfo?.state === "stopped") && (
        <div className="card" style={{ marginTop: "1rem" }}>
          <h3
            onClick={() => setShowInitialUsers(!showInitialUsers)}
            style={{
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: "0.5rem",
              margin: 0,
              padding: 0,
            }}
          >
            <span
              style={{
                transform: showInitialUsers ? "rotate(90deg)" : "rotate(0deg)",
                transition: "transform 0.2s",
              }}
            >
              ▶
            </span>
            Initial Users
          </h3>
          <p
            style={{
              fontSize: "0.9rem",
              color: "#666",
              marginBottom: showInitialUsers ? "1rem" : 0,
              marginTop: "0.5rem",
            }}
          >
            Configure initial root users and an initial policy blueprint.
          </p>

          {showInitialUsers && (
            <div style={{ marginTop: "1rem" }}>
              <InitialUsersTab panelInfo={panelInfo} />
            </div>
          )}
        </div>
      )}

      {/* Encryption at Rest Section - Always visible */}
      <div className="card" style={{ marginTop: "1rem" }}>
        <h3
          onClick={() => setShowEncryptionAtRest(!showEncryptionAtRest)}
          style={{
            cursor: "pointer",
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
            margin: 0,
            padding: 0,
          }}
        >
          <span
            style={{
              transform: showEncryptionAtRest
                ? "rotate(90deg)"
                : "rotate(0deg)",
              transition: "transform 0.2s",
            }}
          >
            ▶
          </span>
          Encryption at Rest
        </h3>
        <p
          style={{
            fontSize: "0.9rem",
            color: "#666",
            marginBottom: showEncryptionAtRest ? "1rem" : 0,
            marginTop: "0.5rem",
          }}
        >
          Configure encryption for this node's key shares using a secret
          manager.
        </p>

        {showEncryptionAtRest && (
          <div style={{ marginTop: "1rem" }}>
            <EncryptionAtRestTab panelInfo={panelInfo} />
          </div>
        )}
      </div>
    </div>
  );
}
