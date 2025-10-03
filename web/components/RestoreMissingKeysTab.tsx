import { useState, useEffect, useRef } from "react";
import { getApiHost } from "../utils/api";
import * as age from "age-encryption";
import { PanelInfo } from "../utils/types";
import {
  panelApiClient,
  RestoreMissingKeysResponse,
} from "../utils/panel-client";

interface Props {
  panelInfo: PanelInfo | null;
}

export default function RestoreMissingKeysTab({ panelInfo }: Props) {
  const [selectedBakKey, setSelectedBakKey] = useState<string>("");
  const [manualBakKey, setManualBakKey] = useState<string>("");
  const [useManualInput, setUseManualInput] = useState<boolean>(false);
  const [keyFiles, setKeyFiles] = useState<string[]>([]);
  const [keyFilesLoading, setKeyFilesLoading] = useState(false);
  const [keyFilesCount, setKeyFilesCount] = useState(0);
  const [restoring, setRestoring] = useState(false);
  const [mnemonic, setMnemonic] = useState<string>("");
  const [showMnemonicInput, setShowMnemonicInput] = useState(false);
  const [restoreResult, setRestoreResult] =
    useState<RestoreMissingKeysResponse | null>(null);
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });

  const debounceTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const mnemonicInputRef = useRef<HTMLDivElement>(null);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }
    };
  }, []);

  // Handle manual input changes
  const handleManualInputChange = (value: string) => {
    setManualBakKey(value);
    fetchKeyFiles(value);
  };

  // Fetch key files from S3
  const fetchKeyFiles = async (bakKey?: string) => {
    if (!panelInfo?.node_id) {
      setStatus({
        type: "error",
        message: "Node ID not available. Panel info may not be loaded.",
      });
      return;
    }

    const currentBakKey =
      bakKey || (useManualInput ? manualBakKey : selectedBakKey);
    if (!currentBakKey.trim()) {
      setKeyFiles([]);
      setKeyFilesCount(0);
      return;
    }

    if (!currentBakKey.trim().startsWith("age1")) {
      setStatus({
        type: "error",
        message: "Backup key must start with 'age1'",
      });
      setKeyFiles([]);
      setKeyFilesCount(0);
      return;
    }

    setKeyFilesLoading(true);
    setStatus({ type: null, message: "" });
    setKeyFiles([]);
    setKeyFilesCount(0);

    try {
      // Use first 32 characters of the backup key for the prefix
      const truncatedBakKey = currentBakKey.substring(0, 32);
      const nodeId = panelInfo.node_id.toString();
      const prefix = `nodes/${nodeId}/keys/nodes/${nodeId}/${truncatedBakKey}`;

      let allKeyFiles: string[] = [];
      let marker = "";
      let hasMore = true;
      let totalCount = 0;

      // Paginate through all results
      while (hasMore) {
        const queryParams = new URLSearchParams({ prefix });
        if (marker) {
          queryParams.append("marker", marker);
        }

        const response = await fetch(
          `${getApiHost()}/v1/s3/objects?${queryParams}`
        );

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText || `HTTP ${response.status}`);
        }

        const data = await response.json();

        if (!data.Contents || !Array.isArray(data.Contents)) {
          hasMore = data.IsTruncated || false;
          marker = data.NextMarker || "";
          if (hasMore && !marker) {
            hasMore = false;
          }
          continue;
        }

        // Filter for JSON key files
        const jsonFiles = data.Contents.filter((obj: any) =>
          obj.Key.toLowerCase().endsWith(".json")
        );

        totalCount += jsonFiles.length;
        setKeyFilesCount(totalCount);

        // Add to preview (limit to first 50 for performance)
        if (allKeyFiles.length < 50) {
          const newFiles = jsonFiles
            .slice(0, 50 - allKeyFiles.length)
            .map((obj: any) => obj.Key.split("/").pop() || obj.Key);
          allKeyFiles.push(...newFiles);
          setKeyFiles([...allKeyFiles]);
        }

        // Check if we need to paginate
        hasMore = data.IsTruncated || false;
        marker = data.NextMarker || "";

        if (hasMore && !marker) {
          hasMore = false;
        }
      }

      if (totalCount === 0) {
        setStatus({
          type: "info",
          message: "No key files found for this backup key.",
        });
      }
    } catch (error) {
      console.error("Failed to fetch key files:", error);
      setStatus({
        type: "error",
        message: `Failed to fetch key files: ${error}`,
      });
    } finally {
      setKeyFilesLoading(false);
    }
  };

  // Handle restore missing keys
  const handleRestoreMissingKeys = async () => {
    if (!panelInfo?.recipient) {
      setStatus({
        type: "error",
        message: "Panel recipient not available.",
      });
      return;
    }

    if (!mnemonic.trim()) {
      setStatus({
        type: "error",
        message: "Please enter your mnemonic phrase.",
      });
      return;
    }

    setRestoring(true);
    setStatus({ type: null, message: "" });
    setRestoreResult(null);

    try {
      // Encrypt mnemonic using age encryption
      const encrypter = new age.Encrypter();
      encrypter.addRecipient(panelInfo.recipient);
      const encryptedBytes = await encrypter.encrypt(mnemonic);

      // Convert to base64 string for API transmission
      const encryptedMnemonic = btoa(
        String.fromCharCode.apply(null, Array.from(encryptedBytes))
      );

      const result = await panelApiClient.restoreMissingKeys({
        encrypted_secret_phrase: encryptedMnemonic,
      });

      setRestoreResult(result);
      setStatus({
        type: "success",
        message: `Key restoration completed! ${result.imported_keys} keys imported.`,
      });

      // Clear form
      setMnemonic("");
      setShowMnemonicInput(false);

      // Restart treasury service
      await panelApiClient.restartService("treasury.service");
    } catch (error) {
      console.error("Failed to restore missing keys:", error);
      setStatus({
        type: "error",
        message: `Failed to restore missing keys: ${error}`,
      });
    } finally {
      setRestoring(false);
    }
  };

  // Set default backup key when panelInfo loads
  useEffect(() => {
    if (panelInfo?.baks && panelInfo.baks.length > 0 && !selectedBakKey) {
      setSelectedBakKey(panelInfo.baks[0].key);
    }
  }, [panelInfo?.baks, selectedBakKey]);

  // Load key files when backup key changes
  useEffect(() => {
    if (panelInfo?.node_id && selectedBakKey && !useManualInput) {
      fetchKeyFiles(selectedBakKey);
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
    <div className="card">
      <h3>Restore Keys</h3>
      <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}>
        Restore keys from backup that are not already present in the local
        treasury. This is already done by default during the normal restore
        process.
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
              }}
            />
            Enter manually
          </label>
        </div>

        {/* Conditional display based on selection */}
        {!useManualInput ? (
          <div>
            <select
              value={selectedBakKey}
              onChange={(e) => {
                setSelectedBakKey(e.target.value);
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
              id="advancedBakKeyInput"
              type="text"
              placeholder="Enter backup key (age1...)"
              value={manualBakKey}
              onChange={(e) => {
                handleManualInputChange(e.target.value);
              }}
              onPaste={() => {
                setTimeout(() => {
                  const input = document.getElementById(
                    "advancedBakKeyInput"
                  ) as HTMLInputElement;
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
              Enter the backup key you want to search for keys.
            </p>
          </div>
        )}
      </div>

      {/* Key Files Display */}
      {keyFilesLoading && (
        <div style={{ textAlign: "center", padding: "1rem" }}>
          <span className="loading"></span>
          Searching for key files...
        </div>
      )}

      {keyFilesCount > 0 && (
        <div style={{ marginBottom: "1rem" }}>
          <h4>
            Found {keyFilesCount} key file{keyFilesCount !== 1 ? "s" : ""}
          </h4>

          {keyFiles.length > 0 && (
            <div
              style={{
                maxHeight: "200px",
                overflowY: "auto",
                border: "1px solid #ddd",
                borderRadius: "4px",
                backgroundColor: "#f8f9fa",
                padding: "0.5rem",
              }}
            >
              <p
                style={{
                  fontSize: "0.85rem",
                  color: "#666",
                  margin: "0 0 0.5rem 0",
                }}
              >
                Preview ({Math.min(keyFiles.length, 50)} of {keyFilesCount}{" "}
                files):
              </p>
              {keyFiles.map((file, index) => (
                <div
                  key={index}
                  style={{
                    fontSize: "0.8rem",
                    fontFamily: "monospace",
                    padding: "0.25rem",
                    backgroundColor: index % 2 === 0 ? "#ffffff" : "#f1f3f4",
                    wordBreak: "break-all",
                  }}
                >
                  {file}
                </div>
              ))}
              {keyFilesCount > 50 && (
                <div
                  style={{
                    fontSize: "0.8rem",
                    color: "#666",
                    fontStyle: "italic",
                    textAlign: "center",
                    padding: "0.5rem",
                  }}
                >
                  ... and {keyFilesCount - 50} more files
                </div>
              )}
            </div>
          )}

          <button
            className="btn"
            onClick={() => setShowMnemonicInput(true)}
            disabled={restoring}
            style={{
              backgroundColor: "#28a745",
              borderColor: "#28a745",
              color: "white",
              marginTop: "1rem",
            }}
          >
            Restore Keys
          </button>
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
            Enter your mnemonic phrase to decrypt and import any missing keys
            from the backup. Keys already present in the treasury will be
            skipped.
          </p>

          <div style={{ marginBottom: "1rem" }}>
            <label
              htmlFor="mnemonicInput"
              style={{ display: "block", marginBottom: "0.5rem" }}
            >
              Mnemonic Phrase:
            </label>
            <textarea
              id="mnemonicInput"
              value={mnemonic}
              onChange={(e) => setMnemonic(e.target.value)}
              placeholder="Enter your mnemonic phrase..."
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
              onClick={handleRestoreMissingKeys}
              disabled={restoring || !mnemonic.trim()}
              style={{
                backgroundColor: "#28a745",
                borderColor: "#28a745",
                color: "white",
              }}
            >
              {restoring && <span className="loading"></span>}
              {restoring ? "Restoring..." : "🔑 Import Missing Keys"}
            </button>
            <button
              className="btn secondary"
              onClick={() => {
                setShowMnemonicInput(false);
                setMnemonic("");
                setRestoreResult(null);
              }}
              disabled={restoring}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Restore Result */}
      {restoreResult && (
        <div
          style={{
            marginTop: "1rem",
            padding: "1rem",
            backgroundColor: "#d4edda",
            border: "1px solid #c3e6cb",
            borderRadius: "4px",
          }}
        >
          <h4 style={{ margin: "0 0 0.5rem 0", color: "#155724" }}>
            ✅ Key Restoration Complete
          </h4>
          <div style={{ fontSize: "0.9rem", color: "#155724" }}>
            <div>• Active keys in treasury: {restoreResult.active_keys}</div>
            <div>• Keys found in backup: {restoreResult.backed_up_keys}</div>
            <div>• Keys imported: {restoreResult.imported_keys}</div>
          </div>
          {restoreResult.imported_keys === 0 && (
            <p
              style={{
                fontSize: "0.85rem",
                color: "#856404",
                marginTop: "0.5rem",
                marginBottom: 0,
              }}
            >
              No missing keys were found. All backed up keys are already present
              in the treasury.
            </p>
          )}
        </div>
      )}
    </div>
  );
}
