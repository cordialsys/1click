import { useState } from "react";
import { PanelInfo } from "../utils/types";
import { panelApiClient } from "../utils/panel-client";

interface Props {
  panelInfo: PanelInfo | null;
}

export default function UploadSnapshotTab({ panelInfo }: Props) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });

  const getSnapshotIdFromFile = (file: File): string => {
    // Get basename without extension
    const filename = file.name;
    const lastDotIndex = filename.lastIndexOf(".");
    return lastDotIndex > 0 ? filename.substring(0, lastDotIndex) : filename;
  };

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      setStatus({ type: null, message: "" });
    }
  };

  const uploadSnapshot = async () => {
    if (!selectedFile) {
      setStatus({
        type: "error",
        message: "Please select a snapshot file to upload.",
      });
      return;
    }

    const snapshotId = getSnapshotIdFromFile(selectedFile);

    setUploading(true);
    setStatus({ type: null, message: "" });

    try {
      await panelApiClient.uploadSnapshot(snapshotId, selectedFile);

      setStatus({
        type: "success",
        message: `Snapshot "${snapshotId}" uploaded successfully!`,
      });

      // Clear form after successful upload
      setSelectedFile(null);
      // Reset file input
      const fileInput = document.getElementById(
        "snapshotFile"
      ) as HTMLInputElement;
      if (fileInput) {
        fileInput.value = "";
      }
    } catch (error) {
      console.error("Failed to upload snapshot:", error);
      setStatus({
        type: "error",
        message: `Failed to upload snapshot: ${error}`,
      });
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      <div className="card">
        <h3>Upload Backup Snapshot</h3>
        <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}>
          Only snapshots for treasury node {panelInfo?.node_id} can be uploaded.
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

        {/* File Upload */}
        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.5rem" }}>
            <strong>Snapshot File:</strong>
          </label>
          <input
            id="snapshotFile"
            type="file"
            accept=".tar"
            onChange={handleFileChange}
            disabled={uploading}
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ddd",
              borderRadius: "4px",
              fontSize: "0.9rem",
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
            Select a .tar snapshot file that was previously downloaded from this
            treasury system.
          </p>
          {selectedFile && (
            <div
              style={{
                fontSize: "0.85rem",
                color: "#28a745",
                marginTop: "0.5rem",
                padding: "0.5rem",
                backgroundColor: "#d4edda",
                borderRadius: "4px",
                border: "1px solid #c3e6cb",
              }}
            >
              <div>
                📄 {selectedFile.name} ({Math.round(selectedFile.size / 1024)}{" "}
                KB)
              </div>
            </div>
          )}
        </div>

        {/* Upload Button */}
        <div style={{ marginTop: "1.5rem" }}>
          <button
            className="btn"
            onClick={uploadSnapshot}
            disabled={uploading || !selectedFile}
            style={{
              backgroundColor: "#28a745",
              borderColor: "#28a745",
              color: "white",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!uploading && selectedFile) {
                e.currentTarget.style.backgroundColor = "#218838";
                e.currentTarget.style.borderColor = "#218838";
              }
            }}
            onMouseLeave={(e) => {
              if (!uploading && selectedFile) {
                e.currentTarget.style.backgroundColor = "#28a745";
                e.currentTarget.style.borderColor = "#28a745";
              }
            }}
          >
            {uploading && <span className="loading"></span>}
            {uploading ? "Uploading..." : "📤 Upload Snapshot"}
          </button>
        </div>
      </div>
    </div>
  );
}
