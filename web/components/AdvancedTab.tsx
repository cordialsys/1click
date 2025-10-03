import { useState, useRef, useEffect } from "react";
import { panelApiClient } from "../utils/panel-client";
import BootcUpdateTab from "./BootcUpdateTab";

interface Props {
  loading: boolean;
  showResetConfirmation: boolean;
  setShowResetConfirmation: (show: boolean) => void;
  resetTreasury: () => Promise<void>;
}

export default function AdvancedTab({
  loading,
  showResetConfirmation,
  setShowResetConfirmation,
  resetTreasury,
}: Props) {
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });
  const [binaryVersion, setBinaryVersion] = useState("latest");
  const [loadingAction, setLoadingAction] = useState<string | null>(null);
  const [showVmUpdate, setShowVmUpdate] = useState(false);
  const confirmationRef = useRef<HTMLDivElement>(null);

  // Scroll to confirmation when it appears
  useEffect(() => {
    if (showResetConfirmation && confirmationRef.current) {
      confirmationRef.current.scrollIntoView({
        behavior: 'smooth',
        block: 'start'
      });
    }
  }, [showResetConfirmation]);
  return (
    <div>
      {/* Status Messages */}
      {status.type && (
        <div
          className="card"
          style={{
            border: `2px solid ${
              status.type === "success"
                ? "#28a745"
                : status.type === "error"
                ? "#dc3545"
                : "#17a2b8"
            }`,
            marginBottom: "1rem",
          }}
        >
          <p
            style={{
              color:
                status.type === "success"
                  ? "#155724"
                  : status.type === "error"
                  ? "#721c24"
                  : "#0c5460",
              margin: 0,
            }}
          >
            {status.message}
          </p>
        </div>
      )}

      <div className="card">
        <h3>Advanced Operations</h3>

        <div style={{ marginBottom: "2rem" }}>
          <h4>Treasury Service Control</h4>
          <p
            style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}
          >
            Restart or stop the treasury service.
          </p>
          <div style={{ display: "flex", gap: "0.5rem" }}>
            <button
              className="btn"
              onClick={async () => {
                setLoadingAction("restart-treasury");
                try {
                  await panelApiClient.restartService("treasury.service");

                  setStatus({
                    type: "success",
                    message:
                      "Treasury service restarting. Track status page for progress.",
                  });
                } catch (error) {
                  console.error("Failed to restart treasury service:", error);
                  setStatus({
                    type: "error",
                    message: `Failed to restart treasury service: ${error}`,
                  });
                } finally {
                  setLoadingAction(null);
                }
              }}
              disabled={loading || loadingAction !== null}
              style={{
                backgroundColor: "#0070f3",
                borderColor: "#0070f3",
                color: "white",
                transition: "all 0.2s ease",
              }}
              onMouseEnter={(e) => {
                if (!loading && loadingAction === null) {
                  e.currentTarget.style.backgroundColor = "#0056b3";
                  e.currentTarget.style.borderColor = "#0056b3";
                }
              }}
              onMouseLeave={(e) => {
                if (!loading && loadingAction === null) {
                  e.currentTarget.style.backgroundColor = "#0070f3";
                  e.currentTarget.style.borderColor = "#0070f3";
                }
              }}
            >
              {loadingAction === "restart-treasury" && (
                <span className="loading"></span>
              )}
              🔄 Restart Treasury
            </button>
            <button
              className="btn"
              onClick={async () => {
                setLoadingAction("stop-treasury");
                try {
                  await panelApiClient.stopService("treasury.service");

                  setStatus({
                    type: "success",
                    message: "Treasury service stopped successfully.",
                  });
                } catch (error) {
                  console.error("Failed to stop treasury service:", error);
                  setStatus({
                    type: "error",
                    message: `Failed to stop treasury service: ${error}`,
                  });
                } finally {
                  setLoadingAction(null);
                }
              }}
              disabled={loading || loadingAction !== null}
              style={{
                backgroundColor: "#6c757d",
                borderColor: "#6c757d",
                color: "white",
                transition: "all 0.2s ease",
              }}
              onMouseEnter={(e) => {
                if (!loading && loadingAction === null) {
                  e.currentTarget.style.backgroundColor = "#5a6268";
                  e.currentTarget.style.borderColor = "#5a6268";
                }
              }}
              onMouseLeave={(e) => {
                if (!loading && loadingAction === null) {
                  e.currentTarget.style.backgroundColor = "#6c757d";
                  e.currentTarget.style.borderColor = "#6c757d";
                }
              }}
            >
              {loadingAction === "stop-treasury" && (
                <span className="loading"></span>
              )}
              ⏹️ Stop Treasury
            </button>
          </div>
        </div>

        <div style={{ marginBottom: "2rem" }}>
          <h4>Download Binaries</h4>
          <p
            style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}
          >
            Securely update treasury binaries. This is not the same as updating
            treasury itself. Treasury will be restarted if it is running.
          </p>
          <div
            style={{
              display: "flex",
              gap: "0.5rem",
              alignItems: "center",
              marginBottom: "1rem",
            }}
          >
            <label
              htmlFor="binaryVersion"
              style={{ fontSize: "0.9rem", minWidth: "60px" }}
            >
              Version:
            </label>
            <input
              id="binaryVersion"
              type="text"
              value={binaryVersion}
              onChange={(e) => setBinaryVersion(e.target.value)}
              placeholder="latest"
              disabled={loading}
              style={{
                flex: 1,
                padding: "0.5rem",
                border: "1px solid #ddd",
                borderRadius: "4px",
                fontSize: "0.9rem",
              }}
            />
          </div>
          <button
            className="btn"
            onClick={async () => {
              setLoadingAction("download-binaries");
              try {
                const binariesResponse = await panelApiClient.downloadBinaries(
                  binaryVersion
                );
                if (!binariesResponse.ok) {
                  const errorText = await binariesResponse.text();
                  throw new Error(`Failed to download binaries: ${errorText}`);
                }

                setStatus({
                  type: "success",
                  message: `Binaries downloaded successfully (version: ${binaryVersion}).`,
                });
              } catch (error) {
                console.error("Failed to download binaries:", error);
                setStatus({
                  type: "error",
                  message: `Failed to download binaries: ${error}`,
                });
              } finally {
                setLoadingAction(null);
              }
            }}
            disabled={loading || loadingAction !== null}
            style={{
              backgroundColor: "#6c757d",
              borderColor: "#6c757d",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!loading && loadingAction === null) {
                e.currentTarget.style.backgroundColor = "#5a6268";
                e.currentTarget.style.borderColor = "#5a6268";
              }
            }}
            onMouseLeave={(e) => {
              if (!loading && loadingAction === null) {
                e.currentTarget.style.backgroundColor = "#6c757d";
                e.currentTarget.style.borderColor = "#6c757d";
              }
            }}
          >
            {loadingAction === "download-binaries" && (
              <span className="loading"></span>
            )}
            📥 Download Binaries
          </button>
        </div>

        <div style={{ marginBottom: "2rem" }}>
          <h4>Sync Peers</h4>
          <p
            style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}
          >
            This will sync the DNS information for all of the peers. Any changes
            will require treasury to be restarted to take effect.
          </p>
          <button
            className="btn"
            onClick={async () => {
              setLoadingAction("sync-peers");
              try {
                await panelApiClient.syncPeers();

                setStatus({
                  type: "success",
                  message:
                    "Peers synced.  Any changes will reflect after treasury is restarted.",
                });
              } catch (error) {
                console.error("Failed to sync peers:", error);
                setStatus({
                  type: "error",
                  message: `Failed to sync peers: ${error}`,
                });
              } finally {
                setLoadingAction(null);
              }
            }}
            disabled={loading || loadingAction !== null}
            style={{
              backgroundColor: "#6c757d",
              borderColor: "#6c757d",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!loading && loadingAction === null) {
                e.currentTarget.style.backgroundColor = "#5a6268";
                e.currentTarget.style.borderColor = "#5a6268";
              }
            }}
            onMouseLeave={(e) => {
              if (!loading && loadingAction === null) {
                e.currentTarget.style.backgroundColor = "#6c757d";
                e.currentTarget.style.borderColor = "#6c757d";
              }
            }}
          >
            {loadingAction === "sync-peers" && (
              <span className="loading"></span>
            )}
            🔄 Sync Peers
          </button>
        </div>

        <div style={{ marginBottom: "2rem" }}>
          <h4>Delete Node</h4>
          <p
            style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}
          >
            Permanently delete the treasury node and all its data regardless of
            current state. This action cannot be undone, except by restoring
            from an encrypted backup.
          </p>
          <button
            className="btn"
            onClick={() => setShowResetConfirmation(true)}
            disabled={loading}
            style={{
              backgroundColor: "#dc3545",
              borderColor: "#dc3545",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!loading) {
                e.currentTarget.style.backgroundColor = "#c82333";
                e.currentTarget.style.borderColor = "#c82333";
              }
            }}
            onMouseLeave={(e) => {
              if (!loading) {
                e.currentTarget.style.backgroundColor = "#dc3545";
                e.currentTarget.style.borderColor = "#dc3545";
              }
            }}
          >
            🗑️ Delete Node
          </button>
        </div>
      </div>

      {showResetConfirmation && (
        <div
          ref={confirmationRef}
          className="card"
          style={{ border: "2px solid #dc3545" }}>
          <h3 style={{ color: "#dc3545" }}>⚠️ Confirm Node Deletion</h3>
          <p>
            This will permanently delete the treasury node and all its data.
            This action cannot be undone, except via backup recovery.
          </p>
          <p>
            <strong>Are you sure you want to delete the treasury node?</strong>
          </p>
          <div style={{ marginTop: "1rem" }}>
            <button
              className="btn"
              onClick={async () => {
                await resetTreasury();
                setStatus({
                  type: "success",
                  message: "Treasury deleted.",
                });
              }}
              disabled={loading}
              style={{
                backgroundColor: "#dc3545",
                borderColor: "#dc3545",
                color: "white",
                marginRight: "0.5rem",
                transition: "all 0.2s ease",
              }}
              onMouseEnter={(e) => {
                if (!loading) {
                  e.currentTarget.style.backgroundColor = "#c82333";
                  e.currentTarget.style.borderColor = "#c82333";
                }
              }}
              onMouseLeave={(e) => {
                if (!loading) {
                  e.currentTarget.style.backgroundColor = "#dc3545";
                  e.currentTarget.style.borderColor = "#dc3545";
                }
              }}
            >
              {loading ? "Deleting..." : "Yes, Delete Node"}
            </button>
            <button
              className="btn secondary"
              onClick={() => setShowResetConfirmation(false)}
              disabled={loading}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* VM Update Section */}
      <div className="card" style={{ marginTop: "1rem" }}>
        <h3
          onClick={() => setShowVmUpdate(!showVmUpdate)}
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
              transform: showVmUpdate ? "rotate(90deg)" : "rotate(0deg)",
              transition: "transform 0.2s",
            }}
          >
            ▶
          </span>
          Update VM
        </h3>
        <p
          style={{
            fontSize: "0.9rem",
            color: "#666",
            marginBottom: showVmUpdate ? "1rem" : 0,
            marginTop: "0.5rem",
          }}
        >
          Manage VM system updates
        </p>

        {showVmUpdate && (
          <div style={{ marginTop: "1rem" }}>
            <BootcUpdateTab />
          </div>
        )}
      </div>
    </div>
  );
}
