import { useState } from "react";
import { PanelInfo } from "../utils/types";
import { panelApiClient } from "../utils/panel-client";
import UploadSnapshotTab from "./UploadSnapshotTab";

interface Props {
  panelInfo: PanelInfo | null;
}

export default function BackupTab({ panelInfo }: Props) {
  const [selectedBakKey, setSelectedBakKey] = useState<string>("");
  const [customSnapshotId, setCustomSnapshotId] = useState<string>("");
  const [downloadSnapshot, setDownloadSnapshot] = useState<boolean>(false);
  const [creating, setCreating] = useState(false);
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });
  const [showUploadSection, setShowUploadSection] = useState(false);

  const createSnapshot = async () => {
    // selectedBakKey can be empty string for "all backup keys"
    // Only show error if no backup keys exist at all
    if (panelInfo?.baks && panelInfo.baks.length === 0) {
      setStatus({
        type: "error",
        message:
          "No backup keys available. Please generate backup keys in the Activation tab first.",
      });
      return;
    }

    setCreating(true);
    setStatus({ type: null, message: "" });

    try {
      const snapshotId =
        customSnapshotId.trim() || `manual-snapshot-${Date.now()}`;

      const response = await panelApiClient.takeSnapshot(
        snapshotId,
        selectedBakKey,
        downloadSnapshot
      );

      if (downloadSnapshot) {
        // Handle file download
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${snapshotId}.tar`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);

        setStatus({
          type: "success",
          message: `Snapshot "${snapshotId}" created and downloaded successfully!`,
        });
      } else {
        setStatus({
          type: "success",
          message: `Snapshot "${snapshotId}" created successfully and uploaded to backup storage!`,
        });
      }

      // Clear custom snapshot ID after successful creation
      setCustomSnapshotId("");
    } catch (error) {
      console.error("Failed to create snapshot:", error);
      setStatus({
        type: "error",
        message: `Failed to create snapshot: ${error}`,
      });
    } finally {
      setCreating(false);
    }
  };

  return (
    <div>
      <div className="card">
        <h3>Create Backup Snapshot</h3>
        <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}>
          Create a backup snapshot of your treasury. The snapshot will be
          encrypted with your selected backup key and can be used to restore
          your treasury later.
        </p>

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

        {/* Backup Key Selector */}
        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.5rem" }}>
            <strong>Backup Key:</strong>
          </label>
          <select
            value={selectedBakKey}
            onChange={(e) => setSelectedBakKey(e.target.value)}
            disabled={creating}
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ddd",
              borderRadius: "4px",
              fontSize: "0.9rem",
              fontFamily: "monospace",
            }}
          >
            <option value="">All backup keys</option>
            {panelInfo?.baks?.map((bak) => (
              <option key={bak.id} value={bak.key}>
                {bak.id ? `${bak.id} - ${bak.key}` : bak.key}
              </option>
            ))}
          </select>
          {panelInfo?.baks && panelInfo.baks.length === 0 && (
            <p
              style={{
                fontSize: "0.85rem",
                color: "#dc3545",
                marginTop: "0.5rem",
                marginBottom: 0,
              }}
            >
              No backup keys available. Please generate backup keys in the
              Activation tab first.
            </p>
          )}
        </div>

        {/* Custom Snapshot ID */}
        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.5rem" }}>
            <strong>Snapshot ID (optional):</strong>
          </label>
          <input
            type="text"
            value={customSnapshotId}
            onChange={(e) => setCustomSnapshotId(e.target.value)}
            placeholder="manual-snapshot-12345..."
            disabled={creating}
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ddd",
              borderRadius: "4px",
              fontSize: "0.9rem",
            }}
          />
        </div>

        {/* Download Option */}
        <div style={{ marginBottom: "1rem" }}>
          <label
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.5rem",
              cursor: creating ? "not-allowed" : "pointer",
              fontSize: "0.9rem",
            }}
          >
            <input
              type="checkbox"
              checked={downloadSnapshot}
              onChange={(e) => setDownloadSnapshot(e.target.checked)}
              disabled={creating}
            />
            <strong>Download snapshot after creation</strong>
          </label>
          <p
            style={{
              fontSize: "0.8rem",
              color: "#666",
              marginTop: "0.25rem",
              marginBottom: 0,
              marginLeft: "1.5rem",
            }}
          >
            {downloadSnapshot
              ? "Snapshot will be created, uploaded to backup storage, and downloaded to your device."
              : "Snapshot will only be created and uploaded to backup storage."}
          </p>
        </div>

        {/* Action Button */}
        <div style={{ marginTop: "1.5rem" }}>
          <button
            className="btn"
            onClick={createSnapshot}
            disabled={
              creating || !panelInfo?.baks || panelInfo.baks.length === 0
            }
            style={{
              backgroundColor: "#0070f3",
              borderColor: "#0070f3",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!creating && panelInfo?.baks && panelInfo.baks.length > 0) {
                e.currentTarget.style.backgroundColor = "#0051a0";
                e.currentTarget.style.borderColor = "#0051a0";
              }
            }}
            onMouseLeave={(e) => {
              if (!creating && panelInfo?.baks && panelInfo.baks.length > 0) {
                e.currentTarget.style.backgroundColor = "#0070f3";
                e.currentTarget.style.borderColor = "#0070f3";
              }
            }}
          >
            {creating && <span className="loading"></span>}
            {creating ? "Creating..." : "📤 Create Snapshot"}
          </button>
        </div>
      </div>

      {/* Upload Section */}
      <div className="card" style={{ marginTop: "1rem" }}>
        <h3
          onClick={() => setShowUploadSection(!showUploadSection)}
          style={{
            cursor: "pointer",
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
            margin: 0,
            padding: 0,
          }}
        >
          <span style={{ transform: showUploadSection ? "rotate(90deg)" : "rotate(0deg)", transition: "transform 0.2s" }}>
            ▶
          </span>
          Upload
        </h3>
        <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: showUploadSection ? "1rem" : 0, marginTop: "0.5rem" }}>
          Upload a previously downloaded snapshot file to backup storage
        </p>

        {showUploadSection && (
          <div style={{ marginTop: "1rem" }}>
            <UploadSnapshotTab panelInfo={panelInfo} />
          </div>
        )}
      </div>
    </div>
  );
}
