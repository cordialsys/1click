import { useState, useEffect, useRef } from "react";
import * as age from "age-encryption";

import { PanelInfo } from "../utils/types";
import {
  panelApiClient,
  RestoreMissingKeysResponse,
  S3ListResponse,
} from "../utils/panel-client";
import RestoreMissingKeysTab from "./RestoreMissingKeysTab";

interface SnapshotOption {
  key: string;
  basename: string;
  createTime: Date;
  displayName: string;
}

interface Props {
  panelInfo: PanelInfo | null;
}

export default function BackupRestoreTab({ panelInfo }: Props) {
  const [snapshots, setSnapshots] = useState<SnapshotOption[]>([]);
  const [filteredSnapshots, setFilteredSnapshots] = useState<SnapshotOption[]>(
    []
  );
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedSnapshot, setSelectedSnapshot] = useState<string | null>(null);
  const [selectedBakKey, setSelectedBakKey] = useState<string>("");
  const [manualBakKey, setManualBakKey] = useState<string>("");
  const [useManualInput, setUseManualInput] = useState<boolean>(false);
  const [mnemonic, setMnemonic] = useState<string>("");
  const [showMnemonicInput, setShowMnemonicInput] = useState(false);
  const [restoring, setRestoring] = useState(false);
  const [restoreProgress, setRestoreProgress] = useState<string[]>([]);
  const [missingKeysResult, setMissingKeysResult] =
    useState<RestoreMissingKeysResponse | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });
  const mnemonicInputRef = useRef<HTMLDivElement>(null);

  // Handle manual input changes (typing or pasting)
  const handleManualInputChange = (value: string) => {
    console.log("value: ", value);
    setManualBakKey(value);
    setSelectedSnapshot(null); // Clear selection when key changes

    // Validate and handle the input
    if (!value.trim()) {
      // Clear snapshots if input is empty
      setSnapshots([]);
      setStatus({ type: null, message: "" });
    } else if (!value.trim().startsWith("age1")) {
      // Show validation error for invalid format
      setStatus({
        type: "error",
        message: "Backup key must start with 'age1'",
      });
      setSnapshots([]);
    } else {
      // Valid format - fetch immediately
      console.log("fetching snapshots");
      setStatus({ type: null, message: "" });
      fetchSnapshots(value);
    }
  };

  // Fetch snapshots from S3
  const fetchSnapshots = async (currentBakKey?: string) => {
    if (!panelInfo?.node_id) {
      setStatus({
        type: "error",
        message: "Node ID not available. Panel info may not be loaded.",
      });
      return;
    }

    if (!currentBakKey) {
      currentBakKey = useManualInput ? manualBakKey : selectedBakKey;
    }
    if (!currentBakKey.trim()) {
      setStatus({
        type: "error",
        message: "Please select or enter a backup key first.",
      });
      return;
    }

    setLoading(true);
    setStatus({ type: null, message: "" });
    try {
      // Use first 32 characters of the backup key for the prefix
      const truncatedBakKey = currentBakKey.substring(0, 32);
      const prefix = `nodes/${panelInfo.node_id}/snapshots/${truncatedBakKey}`;
      let allSnapshots: SnapshotOption[] = [];
      let marker = "";
      let hasMore = true;

      // Paginate through all results
      while (hasMore) {
        const data: S3ListResponse = await panelApiClient.listS3Objects({
          prefix,
          marker: marker || undefined,
        });

        // Check if Contents exists and is an array
        if (!data.Contents || !Array.isArray(data.Contents)) {
          // No contents found, continue to next iteration or break
          hasMore = data.IsTruncated || false;
          marker = data.NextMarker || "";
          if (hasMore && !marker) {
            hasMore = false;
          }
          continue;
        }

        // Filter and process only snapshot files (.tar files in snapshots directory)
        const snapshotObjects = data.Contents.filter(
          (obj) => obj.Key.includes("/snapshots/") && obj.Key.endsWith(".tar")
        );

        // Convert to snapshot options
        const snapshotOptions: SnapshotOption[] = snapshotObjects.map((obj) => {
          const basename = obj.Key.split("/").pop() || obj.Key;
          const createTime = new Date(obj.LastModified);

          return {
            key: obj.Key,
            basename,
            createTime,
            displayName: `${basename} (${createTime.toLocaleString()})`,
          };
        });

        allSnapshots.push(...snapshotOptions);

        // Check if we need to paginate
        hasMore = data.IsTruncated || false;
        marker = data.NextMarker || "";

        // Break if no marker for next page
        if (hasMore && !marker) {
          hasMore = false;
        }
      }

      // Sort by create time in descending order (newest first)
      allSnapshots.sort(
        (a, b) => b.createTime.getTime() - a.createTime.getTime()
      );

      setSnapshots(allSnapshots);
      setFilteredSnapshots(allSnapshots);

      if (allSnapshots.length === 0) {
        setStatus({
          type: "info",
          message: "No snapshots found for this node.",
        });
      }
    } catch (error) {
      console.error("Failed to fetch snapshots:", error);
      setStatus({
        type: "error",
        message: `Failed to fetch snapshots: ${error}`,
      });
    } finally {
      setLoading(false);
    }
  };

  // Download snapshot file
  const handleDownloadSnapshot = async (fileKey: string, filename: string) => {
    try {
      const response = await panelApiClient.downloadS3Object(fileKey);

      if (!response.ok) {
        throw new Error(`Download failed: ${response.statusText}`);
      }

      // Create blob and download file
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);

      setStatus({
        type: "success",
        message: `Successfully downloaded ${filename}`,
      });
    } catch (error) {
      console.error("Failed to download snapshot:", error);
      setStatus({
        type: "error",
        message: `Failed to download snapshot: ${error}`,
      });
    }
  };

  // Restore from selected snapshot
  const handleRestore = async () => {
    if (!selectedSnapshot || !panelInfo?.recipient) {
      setStatus({
        type: "error",
        message: "Missing required information for restore.",
      });
      return;
    }

    if (!mnemonic.trim()) {
      setStatus({
        type: "error",
        message: "Please enter your 12-word mnemonic phrase.",
      });
      return;
    }

    setRestoring(true);
    setStatus({ type: null, message: "" });
    setRestoreProgress([]);
    setMissingKeysResult(null);

    try {
      // Encrypt mnemonic using age encryption
      const encrypter = new age.Encrypter();
      encrypter.addRecipient(panelInfo.recipient);
      const encryptedBytes = await encrypter.encrypt(mnemonic);

      // Convert to base64 string for API transmission
      const encryptedMnemonic = btoa(
        String.fromCharCode.apply(null, Array.from(encryptedBytes))
      );

      // Step 1: Restore from snapshot
      setRestoreProgress((prev) => [...prev, "🔄 Restoring from snapshot..."]);
      await panelApiClient.restoreFromSnapshot({
        s3_key: selectedSnapshot,
        encrypted_secret_phrase: encryptedMnemonic,
      });
      setRestoreProgress((prev) => [
        ...prev,
        "✅ Snapshot restored successfully",
      ]);

      // Step 2: Restore missing keys
      setRestoreProgress((prev) => [
        ...prev,
        "🔄 Checking for missing keys...",
      ]);
      const missingKeysResponse = await panelApiClient.restoreMissingKeys({
        encrypted_secret_phrase: encryptedMnemonic,
      });
      setMissingKeysResult(missingKeysResponse);
      setRestoreProgress((prev) => [
        ...prev,
        `✅ Key restoration complete:`,
        `   • Active keys: ${missingKeysResponse.active_keys}`,
        `   • Backed up keys: ${missingKeysResponse.backed_up_keys}`,
        `   • Imported keys: ${missingKeysResponse.imported_keys}`,
      ]);

      // Step 3: Restart treasury service
      setRestoreProgress((prev) => [
        ...prev,
        "🔄 Restarting treasury service...",
      ]);
      await panelApiClient.restartService("treasury.service");
      setRestoreProgress((prev) => [...prev, "✅ Treasury service restarted"]);

      setStatus({
        type: "success",
        message:
          "Restore completed successfully! Treasury service has been restarted.",
      });

      // Clear form
      setMnemonic("");
      setShowMnemonicInput(false);
      setSelectedSnapshot(null);
    } catch (error) {
      console.error("Failed to restore:", error);
      setRestoreProgress((prev) => [...prev, `❌ Error: ${error}`]);
      setStatus({
        type: "error",
        message: `Failed to restore: ${error}`,
      });
    } finally {
      setRestoring(false);
    }
  };

  // Filter snapshots based on search term
  useEffect(() => {
    if (!searchTerm.trim()) {
      setFilteredSnapshots(snapshots);
    } else {
      const filtered = snapshots.filter(
        (snapshot) =>
          snapshot.displayName
            .toLowerCase()
            .includes(searchTerm.toLowerCase()) ||
          snapshot.basename.toLowerCase().includes(searchTerm.toLowerCase())
      );
      setFilteredSnapshots(filtered);
    }
  }, [searchTerm, snapshots]);

  // Set default backup key when panelInfo loads
  useEffect(() => {
    if (panelInfo?.baks && panelInfo.baks.length > 0 && !selectedBakKey) {
      setSelectedBakKey(panelInfo.baks[0].key);
    }
  }, [panelInfo?.baks, selectedBakKey]);

  // Load snapshots when panelInfo or selectedBakKey changes
  useEffect(() => {
    if (panelInfo?.node_id && selectedBakKey && !useManualInput) {
      fetchSnapshots(selectedBakKey);
    }
  }, [panelInfo?.node_id, selectedBakKey, useManualInput]);

  // Scroll to mnemonic input when it appears
  useEffect(() => {
    if (showMnemonicInput && mnemonicInputRef.current) {
      mnemonicInputRef.current.scrollIntoView({
        behavior: 'smooth',
        block: 'start'
      });
    }
  }, [showMnemonicInput]);

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
        <h3>Restore from Snapshot</h3>
        <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}>
          Select a backup key and snapshot to restore from backup. This will
          stop the treasury, restore the selected snapshot, and restart the
          treasury.
        </p>

        {/* Backup Key Selector */}
        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.5rem" }}>
            Backup Key:
          </label>

          {/* Toggle between select and manual input */}
          <div style={{ marginBottom: "1rem" }}>
            <label
              style={{
                display: "flex",
                alignItems: "center",
                gap: "0.5rem",
                fontSize: "0.9rem",
              }}
            >
              <input
                type="radio"
                name="bakKeyMethod"
                checked={!useManualInput}
                onChange={() => {
                  setUseManualInput(false);
                  setManualBakKey("");
                  setSelectedSnapshot(null);
                }}
              />
              Select from current backup keys:
            </label>
            <label
              style={{
                display: "flex",
                alignItems: "center",
                gap: "0.5rem",
                fontSize: "0.9rem",
                marginTop: "0.5rem",
              }}
            >
              <input
                type="radio"
                name="bakKeyMethod"
                checked={useManualInput}
                onChange={() => {
                  setUseManualInput(true);
                  setSelectedBakKey("");
                  setSelectedSnapshot(null);
                }}
              />
              Enter manually
            </label>
          </div>

          {/* Conditional display based on selection */}
          {!useManualInput ? (
            <div>
              <select
                id="bakKeySelect"
                value={selectedBakKey}
                onChange={(e) => {
                  setSelectedBakKey(e.target.value);
                  setSelectedSnapshot(null); // Clear selection when key changes
                }}
                style={{
                  width: "100%",
                  padding: "0.5rem",
                  border: "1px solid #ddd",
                  borderRadius: "4px",
                  fontSize: "0.9rem",
                  fontFamily: "monospace",
                }}
              >
                <option value="">Select a backup key...</option>
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
          ) : (
            <div>
              <input
                id="manualBakKeyInput"
                type="text"
                placeholder="Enter backup key (age1...)"
                value={manualBakKey}
                onChange={(e) => {
                  handleManualInputChange(e.target.value);
                }}
                onPaste={() => {
                  console.log("onPaste");
                  setTimeout(() => {
                    const input = document.getElementById(
                      "manualBakKeyInput"
                    ) as HTMLInputElement;
                    console.log("onPaste", input);
                    if (input) {
                      handleManualInputChange(input.value);
                    }
                  }, 10);
                }}
                onInput={(e) => {
                  handleManualInputChange(e.currentTarget.value);
                }}
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
                  fontSize: "0.85rem",
                  color: "#666",
                  marginTop: "0.5rem",
                  marginBottom: 0,
                }}
              >
                Enter the backup key you want to use for decryption.
              </p>
            </div>
          )}
        </div>

        {/* Search/Filter Input */}
        <div style={{ marginBottom: "1rem" }}>
          {/* <label
            htmlFor="snapshotSearch"
            style={{ display: "block", marginBottom: "0.5rem" }}
          >
            Search Snapshots:
          </label> */}
          <input
            id="snapshotSearch"
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Filter snapshots by name or date..."
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ddd",
              borderRadius: "4px",
              fontSize: "0.9rem",
            }}
          />
        </div>

        {/* Loading State */}
        {loading && (
          <div style={{ textAlign: "center", padding: "2rem" }}>
            <span className="loading"></span>
            Loading snapshots...
          </div>
        )}

        {/* Snapshots List */}
        {!loading && filteredSnapshots.length > 0 && (
          <div style={{ marginBottom: "1rem" }}>
            <label style={{ display: "block", marginBottom: "0.5rem" }}>
              Select Snapshot:
            </label>
            <div
              style={{
                maxHeight: "300px",
                overflowY: "auto",
                border: "1px solid #ddd",
                borderRadius: "4px",
                backgroundColor: "#f8f9fa",
              }}
            >
              {filteredSnapshots.map((snapshot) => (
                <div
                  key={snapshot.key}
                  onClick={() => setSelectedSnapshot(snapshot.key)}
                  style={{
                    padding: "0.75rem",
                    borderBottom: "1px solid #e9ecef",
                    cursor: "pointer",
                    backgroundColor:
                      selectedSnapshot === snapshot.key ? "#007bff" : "white",
                    color:
                      selectedSnapshot === snapshot.key ? "white" : "#212529",
                    transition: "all 0.2s",
                  }}
                  onMouseEnter={(e) => {
                    if (selectedSnapshot !== snapshot.key) {
                      e.currentTarget.style.backgroundColor = "#e9ecef";
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (selectedSnapshot !== snapshot.key) {
                      e.currentTarget.style.backgroundColor = "white";
                    }
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                    }}
                  >
                    <div style={{ flex: 1 }}>
                      <div style={{ fontWeight: "bold", fontSize: "0.95rem" }}>
                        {snapshot.basename}
                      </div>
                      <div
                        style={{
                          fontSize: "0.85rem",
                          opacity: 0.8,
                          marginTop: "0.25rem",
                        }}
                      >
                        {snapshot.createTime.toLocaleString()}
                      </div>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation(); // Prevent selecting the snapshot when clicking download
                        handleDownloadSnapshot(snapshot.key, snapshot.basename);
                      }}
                      style={{
                        padding: "0.25rem 0.5rem",
                        border: "1px solid",
                        borderColor:
                          selectedSnapshot === snapshot.key
                            ? "rgba(255,255,255,0.5)"
                            : "#007bff",
                        borderRadius: "4px",
                        background:
                          selectedSnapshot === snapshot.key
                            ? "rgba(255,255,255,0.1)"
                            : "white",
                        color:
                          selectedSnapshot === snapshot.key
                            ? "white"
                            : "#007bff",
                        cursor: "pointer",
                        fontSize: "0.75rem",
                        transition: "all 0.2s",
                        marginLeft: "0.5rem",
                      }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.backgroundColor =
                          selectedSnapshot === snapshot.key
                            ? "rgba(255,255,255,0.2)"
                            : "#007bff";
                        e.currentTarget.style.color = "white";
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.backgroundColor =
                          selectedSnapshot === snapshot.key
                            ? "rgba(255,255,255,0.1)"
                            : "white";
                        e.currentTarget.style.color =
                          selectedSnapshot === snapshot.key
                            ? "white"
                            : "#007bff";
                      }}
                    >
                      📥 Download
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Empty State */}
        {!loading && filteredSnapshots.length === 0 && snapshots.length > 0 && (
          <div
            style={{
              textAlign: "center",
              padding: "2rem",
              color: "#666",
              fontStyle: "italic",
            }}
          >
            No snapshots match your search criteria.
          </div>
        )}

        {/* Refresh Button */}
        <div style={{ marginTop: "1rem" }}>
          <button
            className="btn secondary"
            onClick={async () => await fetchSnapshots(selectedBakKey)}
            disabled={loading || !selectedBakKey}
            style={{ marginRight: "0.5rem" }}
          >
            {loading ? "Refreshing..." : "Refresh Snapshots"}
          </button>

          {/* Restore Button */}
          {selectedSnapshot && (
            <button
              className="btn"
              onClick={() => setShowMnemonicInput(true)}
              disabled={restoring}
              style={{
                backgroundColor: "#28a745",
                borderColor: "#28a745",
                color: "white",
              }}
            >
              Restore Selected Snapshot
            </button>
          )}
        </div>

        {/* Selected Snapshot Info */}
        {selectedSnapshot && (
          <div
            style={{
              marginTop: "1rem",
              padding: "1rem",
              backgroundColor: "#e7f3ff",
              border: "1px solid #b8daff",
              borderRadius: "4px",
            }}
          >
            <h4 style={{ margin: "0 0 0.5rem 0", color: "#004085" }}>
              Selected Snapshot:
            </h4>
            <p
              style={{
                margin: 0,
                fontFamily: "monospace",
                fontSize: "0.75rem",
                wordBreak: "break-all",
                lineHeight: "1.4",
                color: "#0056b3",
              }}
            >
              {selectedSnapshot}
            </p>
          </div>
        )}

        {/* Mnemonic Input Modal */}
        {showMnemonicInput && (
          <div
            ref={mnemonicInputRef}
            style={{
              marginTop: "1rem",
              padding: "1.5rem",
              backgroundColor: "#fff3cd",
              border: "2px solid #ffc107",
              borderRadius: "4px",
            }}
          >
            <h4 style={{ margin: "0 0 1rem 0", color: "#856404" }}>
              ⚠️ Enter Recovery Mnemonic
            </h4>
            <p
              style={{
                fontSize: "0.9rem",
                color: "#856404",
                marginBottom: "1rem",
              }}
            >
              This will <strong>permanently replace</strong> your current
              treasury node with the selected snapshot. Enter your 12-word
              mnemonic phrase to decrypt and restore the backup.
            </p>

            <div style={{ marginBottom: "1rem" }}>
              <label
                htmlFor="mnemonicInput"
                style={{ display: "block", marginBottom: "0.5rem" }}
              >
                12-Word Mnemonic Phrase:
              </label>
              <textarea
                id="mnemonicInput"
                value={mnemonic}
                onChange={(e) => setMnemonic(e.target.value)}
                placeholder="Enter your 12-word mnemonic phrase separated by spaces..."
                disabled={restoring}
                rows={3}
                style={{
                  width: "100%",
                  padding: "0.75rem",
                  border: "1px solid #ddd",
                  borderRadius: "4px",
                  fontSize: "0.9rem",
                  fontFamily: "monospace",
                  resize: "vertical",
                }}
              />
            </div>

            <div style={{ display: "flex", gap: "0.5rem" }}>
              <button
                className="btn"
                onClick={handleRestore}
                disabled={restoring || !mnemonic.trim()}
                style={{
                  backgroundColor: "#dc3545",
                  borderColor: "#dc3545",
                  color: "white",
                }}
              >
                {restoring && <span className="loading"></span>}
                {restoring ? "Restoring..." : "🔄 Restore Treasury"}
              </button>
              <button
                className="btn secondary"
                onClick={() => {
                  setShowMnemonicInput(false);
                  setMnemonic("");
                  setRestoreProgress([]);
                  setMissingKeysResult(null);
                }}
                disabled={restoring}
              >
                Cancel
              </button>
            </div>

            {/* Restore Progress Log */}
            {restoreProgress.length > 0 && (
              <div
                style={{
                  marginTop: "1rem",
                  padding: "1rem",
                  backgroundColor: "#f8f9fa",
                  border: "1px solid #dee2e6",
                  borderRadius: "4px",
                  maxHeight: "300px",
                  overflowY: "auto",
                }}
              >
                <h5 style={{ margin: "0 0 0.5rem 0", color: "#495057" }}>
                  Restore Progress:
                </h5>
                <div
                  style={{
                    fontFamily: "monospace",
                    fontSize: "0.85rem",
                    lineHeight: "1.4",
                    whiteSpace: "pre-line",
                  }}
                >
                  {restoreProgress.map((step, index) => (
                    <div key={index} style={{ marginBottom: "0.25rem" }}>
                      {step}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Advanced Options */}
      <div style={{ marginTop: "2rem" }}>
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          style={{
            background: "none",
            border: "none",
            color: "#0070f3",
            cursor: "pointer",
            fontSize: "1rem",
            textDecoration: "underline",
            marginBottom: "1rem",
          }}
        >
          {showAdvanced ? "▼" : "▶"} Advanced
        </button>

        {showAdvanced && <RestoreMissingKeysTab panelInfo={panelInfo} />}
      </div>

      {/* Footer - Panel Identity Info */}
      <div
        style={{
          marginTop: "2rem",
          padding: "0.75rem",
          backgroundColor: "#f8f9fa",
          border: "1px solid #dee2e6",
          borderRadius: "4px",
          fontSize: "0.85rem",
        }}
      >
        <div style={{ marginBottom: "0.5rem" }}>
          <strong>Panel Identity</strong>
        </div>
        <div
          style={{
            fontFamily: "monospace",
            fontSize: "0.8rem",
            color: "#495057",
            wordBreak: "break-all",
          }}
        >
          {panelInfo?.recipient || "Not available"}
        </div>
        <div
          style={{ marginTop: "0.5rem", fontSize: "0.8rem", color: "#6c757d" }}
        >
          The secret phrase will be transmitted securely to the Panel's identity
          for restoration, but never saved.
        </div>
      </div>
    </div>
  );
}
