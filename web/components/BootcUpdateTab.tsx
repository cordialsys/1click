import { useState, useEffect, useRef } from "react";
import { panelApiClient, BootcStatusResponse } from "../utils/panel-client";

export default function BootcUpdateTab() {
  const [bootcStatus, setBootcStatus] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [currentAction, setCurrentAction] = useState("");
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });

  const statusIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const fetchBootcStatus = async () => {
    try {
      const data = await panelApiClient.getBootcStatus();
      setBootcStatus(data);
      if (loading) setLoading(false);
    } catch (error) {
      console.error("Failed to fetch bootc status:", error);
      if (loading) {
        setStatus({
          type: "error",
          message: `Failed to fetch bootc status: ${error}`,
        });
        setLoading(false);
      }
    }
  };

  // Auto-refresh every 30 seconds
  useEffect(() => {
    fetchBootcStatus();

    statusIntervalRef.current = setInterval(() => {
      fetchBootcStatus();
    }, 30000); // 30 seconds

    return () => {
      if (statusIntervalRef.current) {
        clearInterval(statusIntervalRef.current);
      }
    };
  }, []);

  const handleAction = async (
    action: string,
    actionFn: () => Promise<void>
  ) => {
    setActionLoading(true);
    setCurrentAction(action);
    setStatus({ type: null, message: "" });

    try {
      await actionFn();
      setStatus({
        type: "success",
        message: `${action} completed successfully`,
      });
      // Refresh status after action
      setTimeout(() => {
        fetchBootcStatus();
      }, 1000);
    } catch (error) {
      setStatus({
        type: "error",
        message: `${action} failed: ${error}`,
      });
    } finally {
      setActionLoading(false);
      setCurrentAction("");
    }
  };

  const renderStatusValue = (value: any): string => {
    if (value === null || value === undefined) return "null";
    if (typeof value === "object") return JSON.stringify(value, null, 2);
    return String(value);
  };

  if (loading) {
    return (
      <div style={{ textAlign: "center", padding: "2rem" }}>
        <span className="loading"></span>
        Loading bootc status...
      </div>
    );
  }

  return (
    <div>
      {/* Status Messages */}
      {status.type && (
        <div
          className={`status ${status.type}`}
          style={{ marginBottom: "1rem" }}
        >
          {status.message}
        </div>
      )}

      {currentAction && (
        <div className="status info" style={{ marginBottom: "1rem" }}>
          <span className="loading"></span>
          {currentAction}
        </div>
      )}

      {/* Current Bootc Status */}
      <div style={{ marginBottom: "2rem" }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: "1rem",
          }}
        >
          <h4 style={{ margin: 0 }}>Current System Status</h4>
          <button
            onClick={fetchBootcStatus}
            disabled={actionLoading}
            style={{
              padding: "0.5rem 1rem",
              border: "1px solid #ddd",
              borderRadius: "4px",
              background: "#f8f9fa",
              cursor: actionLoading ? "not-allowed" : "pointer",
              fontSize: "0.85rem",
            }}
          >
            🔄 Refresh
          </button>
        </div>

        {bootcStatus && (
          <div
            style={{
              background: "#f8f9fa",
              border: "1px solid #e9ecef",
              borderRadius: "4px",
              padding: "1rem",
              fontSize: "0.9rem",
              maxHeight: "300px",
              overflowY: "auto",
            }}
          >
            <pre
              style={{
                margin: 0,
                whiteSpace: "pre-wrap",
                wordBreak: "break-word",
                fontFamily: "monospace",
              }}
            >
              {bootcStatus}
            </pre>
          </div>
        )}
      </div>

      {/* Update Actions */}
      <div style={{ marginBottom: "2rem" }}>
        <h4>System Updates</h4>
        <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}>
          Progressive update process: Check → Stage → Apply (with reboot)
        </p>

        <div style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap" }}>
          <button
            className="btn"
            onClick={() =>
              handleAction(
                "Checking for updates",
                () => panelApiClient.checkBootcUpdate()
              )
            }
            disabled={actionLoading}
            style={{
              backgroundColor: "#17a2b8",
              borderColor: "#17a2b8",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#138496";
                e.currentTarget.style.borderColor = "#138496";
              }
            }}
            onMouseLeave={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#17a2b8";
                e.currentTarget.style.borderColor = "#17a2b8";
              }
            }}
          >
            🔍 Check Updates
          </button>

          <button
            className="btn"
            onClick={() =>
              handleAction("Staging update", () => panelApiClient.stageBootcUpdate())
            }
            disabled={actionLoading}
            style={{
              backgroundColor: "#ffc107",
              borderColor: "#ffc107",
              color: "#212529",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#e0a800";
                e.currentTarget.style.borderColor = "#e0a800";
              }
            }}
            onMouseLeave={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#ffc107";
                e.currentTarget.style.borderColor = "#ffc107";
              }
            }}
          >
            📦 Stage Update
          </button>

          <button
            className="btn"
            onClick={() =>
              handleAction(
                "Applying update (will reboot)",
                () => panelApiClient.applyBootcUpdate()
              )
            }
            disabled={actionLoading}
            style={{
              backgroundColor: "#28a745",
              borderColor: "#28a745",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#218838";
                e.currentTarget.style.borderColor = "#218838";
              }
            }}
            onMouseLeave={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#28a745";
                e.currentTarget.style.borderColor = "#28a745";
              }
            }}
          >
            🚀 Apply Update
          </button>
        </div>
      </div>

      {/* Rollback Actions */}
      <div>
        <h4>System Rollback</h4>
        <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}>
          Progressive rollback process: Stage → Apply (with reboot)
        </p>

        <div style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap" }}>
          <button
            className="btn"
            onClick={() =>
              handleAction(
                "Staging rollback",
                () => panelApiClient.stageBootcRollback()
              )
            }
            disabled={actionLoading}
            style={{
              backgroundColor: "#fd7e14",
              borderColor: "#fd7e14",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#e8650e";
                e.currentTarget.style.borderColor = "#e8650e";
              }
            }}
            onMouseLeave={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#fd7e14";
                e.currentTarget.style.borderColor = "#fd7e14";
              }
            }}
          >
            ⏪ Stage Rollback
          </button>

          <button
            className="btn"
            onClick={() =>
              handleAction(
                "Applying rollback (will reboot)",
                () => panelApiClient.applyBootcRollback()
              )
            }
            disabled={actionLoading}
            style={{
              backgroundColor: "#dc3545",
              borderColor: "#dc3545",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#c82333";
                e.currentTarget.style.borderColor = "#c82333";
              }
            }}
            onMouseLeave={(e) => {
              if (!actionLoading) {
                e.currentTarget.style.backgroundColor = "#dc3545";
                e.currentTarget.style.borderColor = "#dc3545";
              }
            }}
          >
            🔄 Apply Rollback
          </button>
        </div>
      </div>
    </div>
  );
}
