import { useState, useEffect, useRef } from "react";
import { PanelInfo } from "../utils/types";
import { panelApiClient } from "../utils/panel-client";

interface Props {
  panelInfo: PanelInfo | null;
}

export default function EncryptionAtRestTab({ panelInfo }: Props) {
  const [secretReference, setSecretReference] = useState<string>("");
  const [configuring, setConfiguring] = useState(false);
  const [showDisableConfirmation, setShowDisableConfirmation] = useState(false);
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });
  const confirmationRef = useRef<HTMLDivElement>(null);

  const hasEarSecret = panelInfo?.ear_secret && panelInfo.ear_secret !== "";

  // Pre-populate the form with current ear_secret value
  useEffect(() => {
    if (panelInfo?.ear_secret && panelInfo.ear_secret !== "") {
      setSecretReference(panelInfo.ear_secret);
    }
  }, [panelInfo?.ear_secret]);

  // Scroll to confirmation when it appears
  useEffect(() => {
    if (showDisableConfirmation && confirmationRef.current) {
      confirmationRef.current.scrollIntoView({
        behavior: 'smooth',
        block: 'start'
      });
    }
  }, [showDisableConfirmation]);

  const handleSetEncryption = async () => {
    if (!secretReference.trim()) {
      setStatus({
        type: "error",
        message: "Please enter a secret manager reference.",
      });
      return;
    }

    setConfiguring(true);
    setStatus({ type: null, message: "" });

    try {
      await panelApiClient.setEncryptionAtRest({
        ear_secret: secretReference.trim(),
      });

      setStatus({
        type: "success",
        message:
          "Encryption at rest configured successfully! Treasury service will restart.",
      });

      // Clear the input after successful configuration
      setSecretReference("");
    } catch (error) {
      console.error("Failed to configure encryption at rest:", error);
      setStatus({
        type: "error",
        message: `Failed to configure encryption at rest: ${error}`,
      });
    } finally {
      setConfiguring(false);
    }
  };

  const handleDeleteEncryption = async () => {
    setConfiguring(true);
    setStatus({ type: null, message: "" });
    setShowDisableConfirmation(false);

    try {
      await panelApiClient.deleteEncryptionAtRest();

      setStatus({
        type: "success",
        message:
          "Encryption at rest removed successfully! Treasury service will restart.",
      });
    } catch (error) {
      console.error("Failed to remove encryption at rest:", error);
      setStatus({
        type: "error",
        message: `Failed to remove encryption at rest: ${error}`,
      });
    } finally {
      setConfiguring(false);
    }
  };

  return (
    <div>
      <div className="card">
        <h3>Encryption at Rest Configuration</h3>
        <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}>
          Configure encryption for this node's key shares using{" "}
          <a
            href="https://docs.cordialsystems.com/guides/deployment/securing#encryption-at-rest"
            target="_blank"
            rel="noopener noreferrer"
          >
            a secret manager reference
          </a>
          . This adds an additional layer of security to protect your treasury
          keys when stored on disk.
        </p>

        {/* Current Status */}
        {hasEarSecret && (
          <div
            style={{
              padding: "0.75rem",
              marginBottom: "1rem",
              borderRadius: "4px",
              border: "2px solid #28a745",
              backgroundColor: "#d4edda",
              color: "#155724",
            }}
          >
            🔐 Encryption at rest is currently <strong>enabled</strong> for this
            node.
          </div>
        )}

        {!hasEarSecret && (
          <div
            style={{
              padding: "0.75rem",
              marginBottom: "1rem",
              borderRadius: "4px",
              border: "2px solid #ffc107",
              backgroundColor: "#fff3cd",
              color: "#856404",
            }}
          >
            ⚠️ Encryption at rest is currently <strong>disabled</strong> for
            this node.
          </div>
        )}

        {/* Status Messages */}
        {status.type && (
          <div
            style={{
              padding: "0.75rem",
              marginBottom: "1rem",
              borderRadius: "4px",
              border: `2px solid ${
                status.type === "success"
                  ? "#28a745"
                  : status.type === "error"
                  ? "#dc3545"
                  : "#17a2b8"
              }`,
              backgroundColor:
                status.type === "success"
                  ? "#d4edda"
                  : status.type === "error"
                  ? "#f8d7da"
                  : "#d1ecf1",
              color:
                status.type === "success"
                  ? "#155724"
                  : status.type === "error"
                  ? "#721c24"
                  : "#0c5460",
            }}
          >
            {status.message}
          </div>
        )}

        {/* Configuration Section */}
        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.5rem" }}>
            <strong>Secret Manager Reference:</strong>
          </label>
          <input
            type="text"
            value={secretReference}
            onChange={(e) => setSecretReference(e.target.value)}
            placeholder="e.g., aws:<name[:key]>[,region][,version] or gcp:<project>,<name>[,version]"
            disabled={configuring}
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ddd",
              borderRadius: "4px",
              fontSize: "0.9rem",
              fontFamily: "monospace",
            }}
          />
          <p
            style={{
              fontSize: "0.8rem",
              color: "#666",
              marginTop: "0.25rem",
              marginBottom: 0,
            }}
          >
            Enter a secret manager reference that contains a 12-word BIP39
            mnemonic phrase.
          </p>
        </div>

        {/* Action Buttons */}
        <div style={{ display: "flex", gap: "0.5rem", marginTop: "1.5rem" }}>
          <button
            className="btn"
            onClick={handleSetEncryption}
            disabled={configuring || !secretReference.trim()}
            style={{
              backgroundColor: hasEarSecret ? "#fd7e14" : "#28a745",
              borderColor: hasEarSecret ? "#fd7e14" : "#28a745",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!configuring && secretReference.trim()) {
                const hoverColor = hasEarSecret ? "#e8650e" : "#218838";
                e.currentTarget.style.backgroundColor = hoverColor;
                e.currentTarget.style.borderColor = hoverColor;
              }
            }}
            onMouseLeave={(e) => {
              if (!configuring && secretReference.trim()) {
                const normalColor = hasEarSecret ? "#fd7e14" : "#28a745";
                e.currentTarget.style.backgroundColor = normalColor;
                e.currentTarget.style.borderColor = normalColor;
              }
            }}
          >
            {configuring && <span className="loading"></span>}
            {configuring
              ? "Configuring..."
              : hasEarSecret
              ? "🔄 Update Encryption"
              : "🔐 Enable Encryption"}
          </button>

          {hasEarSecret && (
            <button
              className="btn"
              onClick={() => setShowDisableConfirmation(true)}
              disabled={configuring}
              style={{
                backgroundColor: "#dc3545",
                borderColor: "#dc3545",
                color: "white",
                transition: "all 0.2s ease",
              }}
              onMouseEnter={(e) => {
                if (!configuring) {
                  e.currentTarget.style.backgroundColor = "#c82333";
                  e.currentTarget.style.borderColor = "#c82333";
                }
              }}
              onMouseLeave={(e) => {
                if (!configuring) {
                  e.currentTarget.style.backgroundColor = "#dc3545";
                  e.currentTarget.style.borderColor = "#dc3545";
                }
              }}
            >
              🗑️ Disable Encryption
            </button>
          )}
        </div>

        <div
          style={{
            marginTop: "1rem",
            padding: "0.75rem",
            backgroundColor: "#e7f3ff",
            border: "1px solid #b8daff",
            borderRadius: "4px",
            fontSize: "0.85rem",
            color: "#004085",
          }}
        >
          <strong>💡 Important Notes:</strong>
          <ul style={{ margin: "0.5rem 0 0 1rem", paddingLeft: "1rem" }}>
            <li>
              The secret must contain a valid 12-word BIP39 mnemonic phrase.
            </li>
            <li>
              Treasury service will be temporarily stopped and restarted during
              configuration. Any existing EAR secret will be replaced.
            </li>
            <li>
              This encrypts key shares stored on disk for additional security.
            </li>
            <li>
              Make sure your secret manager reference remains accessible from
              this node, or treasury will not be able to use the keys.
            </li>
            <li>
              This does not affect any of the backups, as they remain encrypted
              against their own keys.
            </li>
          </ul>
        </div>
      </div>

      {/* Disable Confirmation Dialog */}
      {showDisableConfirmation && (
        <div
          ref={confirmationRef}
          className="card"
          style={{ border: "2px solid #dc3545", marginTop: "1rem" }}
        >
          <h3 style={{ color: "#dc3545" }}>⚠️ Confirm Disable Encryption</h3>
          <p>Are you sure you want to disable encryption at rest? This will:</p>
          <ul style={{ marginLeft: "1rem", color: "#721c24" }}>
            <li>Remove encryption protection for key shares stored on disk</li>
            <li>Restart the treasury service (temporary downtime)</li>
            <li>Reduce the security level of your treasury node</li>
          </ul>

          <div style={{ marginTop: "1.5rem", display: "flex", gap: "0.5rem" }}>
            <button
              className="btn"
              onClick={handleDeleteEncryption}
              disabled={configuring}
              style={{
                backgroundColor: "#dc3545",
                borderColor: "#dc3545",
                color: "white",
                transition: "all 0.2s ease",
              }}
              onMouseEnter={(e) => {
                if (!configuring) {
                  e.currentTarget.style.backgroundColor = "#c82333";
                  e.currentTarget.style.borderColor = "#c82333";
                }
              }}
              onMouseLeave={(e) => {
                if (!configuring) {
                  e.currentTarget.style.backgroundColor = "#dc3545";
                  e.currentTarget.style.borderColor = "#dc3545";
                }
              }}
            >
              {configuring && <span className="loading"></span>}
              {configuring ? "Disabling..." : "Yes, Disable Encryption"}
            </button>
            <button
              className="btn secondary"
              onClick={() => setShowDisableConfirmation(false)}
              disabled={configuring}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
