import { useState, useEffect } from "react";
import {
  generateBackupKey,
  validateAgeRecipient,
  BackupKey,
} from "../utils/backupKey";

interface Props {
  onKeysGenerated: (
    keys: { mnemonic: string; ageRecipient: string; nickname: string }[],
    hasUnsavedKeys: boolean,
    hasOpenImportForms: boolean
  ) => void;
  disabled?: boolean;
}

interface GeneratedKey extends BackupKey {
  id: string;
  saved: boolean;
  nickname: string;
}

export default function BackupKeyGenerator({
  onKeysGenerated,
  disabled = false,
}: Props) {
  const [generatedKeys, setGeneratedKeys] = useState<GeneratedKey[]>([]);
  const [showMnemonics, setShowMnemonics] = useState<Record<string, boolean>>(
    {}
  );
  const [copiedItems, setCopiedItems] = useState<Record<string, boolean>>({});
  const [showImportForm, setShowImportForm] = useState(false);
  const [importedAgeKey, setImportedAgeKey] = useState("");
  const [importNickname, setImportNickname] = useState("");
  const [importError, setImportError] = useState("");

  const updateNickname = (id: string, nickname: string) => {
    // Remove whitespace from nickname
    const cleanedNickname = nickname.replace(/\s+/g, "");
    const updatedKeys = generatedKeys.map((k) =>
      k.id === id ? { ...k, nickname: cleanedNickname } : k
    );
    setGeneratedKeys(updatedKeys);
    updateParent(updatedKeys);
  };

  // Update parent when import form state changes
  useEffect(() => {
    updateParent(generatedKeys);
  }, [showImportForm]);

  const generateNewKey = async () => {
    try {
      const key = await generateBackupKey();
      const newKey: GeneratedKey = {
        ...key,
        id: Date.now().toString(),
        saved: false,
        nickname: "",
      };

      setGeneratedKeys((prev) => [...prev, newKey]);
      updateParent([...generatedKeys, newKey]);
    } catch (error) {
      console.error("Failed to generate backup key:", error);
    }
  };

  const importBackupKey = () => {
    if (!importedAgeKey.trim()) {
      setImportError("Please enter an age key.");
      return;
    }

    // Validate age key format
    if (!validateAgeRecipient(importedAgeKey.trim())) {
      setImportError(
        "Invalid age key format. Please enter a valid age1... key."
      );
      return;
    }

    // Clear any previous errors
    setImportError("");

    // Create imported key (no mnemonic, but marked as saved)
    const importedKey: GeneratedKey = {
      mnemonic: [], // Empty mnemonic for imported keys
      ageRecipient: importedAgeKey.trim(),
      id: Date.now().toString(),
      saved: true, // Imported keys are automatically considered "saved"
      nickname: importNickname.trim(),
    };

    setGeneratedKeys((prev) => [...prev, importedKey]);
    updateParent([...generatedKeys, importedKey]);
    setImportedAgeKey("");
    setImportNickname("");
    setShowImportForm(false);
    setImportError("");
  };

  const markAsSaved = (id: string) => {
    const key = generatedKeys.find((k) => k.id === id);
    if (!key) return;

    const updatedKeys = generatedKeys.map((k) =>
      k.id === id ? { ...k, saved: !k.saved } : k
    );
    setGeneratedKeys(updatedKeys);
    updateParent(updatedKeys);

    // Toggle visibility based on saved state
    setShowMnemonics((prev) => ({
      ...prev,
      [id]: key.saved, // If it was saved, show it (unsave). If it wasn't saved, hide it (save)
    }));
  };

  const removeKey = (id: string) => {
    const updatedKeys = generatedKeys.filter((key) => key.id !== id);
    setGeneratedKeys(updatedKeys);
    updateParent(updatedKeys);
  };

  const updateParent = (keys: GeneratedKey[]) => {
    const savedKeys = keys
      .filter((key) => key.saved)
      .map((key) => ({
        mnemonic: key.mnemonic.join(" "),
        ageRecipient: key.ageRecipient,
        nickname: key.nickname,
      }));
    const hasUnsavedKeys = keys.some((key) => !key.saved);
    onKeysGenerated(savedKeys, hasUnsavedKeys, showImportForm);
  };

  const copyToClipboard = async (text: string, itemId: string) => {
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(text);
      setCopiedItems((prev) => ({ ...prev, [itemId]: true }));
      setTimeout(() => {
        setCopiedItems((prev) => ({ ...prev, [itemId]: false }));
      }, 500);
    }
  };

  return (
    <div>
      <div style={{ marginBottom: "1rem" }}>
        <h4>Backup Key Generator</h4>
        <p style={{ color: "#666", fontSize: "0.9rem", marginBottom: "1rem" }}>
          Generate secure backup keys for treasury recovery. You must save at
          least one backup key.
        </p>
      </div>

      {generatedKeys.length > 0 && (
        <div>
          {generatedKeys.map((key) => (
            <div
              key={key.id}
              style={{
                border: "1px solid #ddd",
                borderRadius: "6px",
                padding: "1rem",
                marginBottom: "1rem",
                backgroundColor: key.saved ? "#f0f8f0" : "#fff8f0",
              }}
            >
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  marginBottom: "0.5rem",
                }}
              >
                <strong>
                  Backup Key #{generatedKeys.indexOf(key) + 1}
                  {key.mnemonic.length === 0 ? " (imported)" : ""}
                </strong>
                <button
                  type="button"
                  onClick={() => removeKey(key.id)}
                  style={{
                    background: "none",
                    border: "none",
                    color: "#dc3545",
                    cursor: "pointer",
                    fontSize: "1.2rem",
                  }}
                >
                  ×
                </button>
              </div>

              {key.mnemonic.length > 0 && (
                <div style={{ marginBottom: "1rem" }}>
                  <label
                    style={{
                      display: "block",
                      marginBottom: "0.5rem",
                      fontWeight: "bold",
                    }}
                  >
                    Secret Recovery Phrase:
                  </label>
                  <div
                    style={{
                      display: "flex",
                      alignItems: "flex-start",
                      gap: "0.5rem",
                    }}
                  >
                    {showMnemonics[key.id] === false ? (
                      <input
                        type="password"
                        value={key.mnemonic.join(" ")}
                        readOnly
                        style={{
                          flex: 1,
                          padding: "0.5rem",
                          border: "1px solid #ddd",
                          borderRadius: "4px",
                          fontFamily: "monospace",
                          fontSize: "0.9rem",
                        }}
                      />
                    ) : (
                      <div
                        style={{
                          flex: 1,
                          display: "grid",
                          gridTemplateColumns: "repeat(4, 1fr)",
                          gap: "0.5rem",
                          padding: "0.5rem",
                          border: "1px solid #ddd",
                          borderRadius: "4px",
                          backgroundColor: "#f8f9fa",
                        }}
                      >
                        {key.mnemonic.map((word, index) => (
                          <div
                            key={index}
                            style={{
                              padding: "0.25rem 0.5rem",
                              backgroundColor: "white",
                              border: "1px solid #e9ecef",
                              borderRadius: "3px",
                              fontSize: "0.85rem",
                              fontFamily: "monospace",
                              textAlign: "center",
                              color: "#495057",
                            }}
                          >
                            {word}
                          </div>
                        ))}
                      </div>
                    )}
                    <button
                      type="button"
                      onClick={() =>
                        copyToClipboard(
                          key.mnemonic.join(" "),
                          `mnemonic-${key.id}`
                        )
                      }
                      style={{
                        padding: "0.5rem",
                        border: "1px solid #ddd",
                        borderRadius: "4px",
                        background: copiedItems[`mnemonic-${key.id}`]
                          ? "#d4edda"
                          : "white",
                        cursor: "pointer",
                        transition: "background-color 0.3s",
                      }}
                    >
                      {copiedItems[`mnemonic-${key.id}`] ? "✅" : "📋"}
                    </button>
                  </div>
                </div>
              )}

              <div style={{ marginBottom: "1rem" }}>
                <label
                  style={{
                    display: "block",
                    marginBottom: "0.5rem",
                    fontWeight: "bold",
                  }}
                >
                  Public Key:
                </label>
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "0.5rem",
                  }}
                >
                  <input
                    type="text"
                    value={key.ageRecipient}
                    readOnly
                    style={{
                      flex: 1,
                      padding: "0.5rem",
                      border: "1px solid #ddd",
                      borderRadius: "4px",
                      fontFamily: "monospace",
                      fontSize: "0.9rem",
                    }}
                  />
                  <button
                    type="button"
                    onClick={() =>
                      copyToClipboard(key.ageRecipient, `age-${key.id}`)
                    }
                    style={{
                      padding: "0.5rem",
                      border: "1px solid #ddd",
                      borderRadius: "4px",
                      background: copiedItems[`age-${key.id}`]
                        ? "#d4edda"
                        : "white",
                      cursor: "pointer",
                      transition: "background-color 0.3s",
                    }}
                  >
                    {copiedItems[`age-${key.id}`] ? "✅" : "📋"}
                  </button>
                </div>
              </div>
              <div style={{ marginBottom: "1rem" }}>
                <input
                  type="text"
                  value={key.nickname}
                  onChange={(e) => updateNickname(key.id, e.target.value)}
                  placeholder="Optional nickname"
                  disabled={disabled}
                  style={{
                    width: "100%",
                    padding: "0.5rem",
                    border: "1px solid #ddd",
                    borderRadius: "4px",
                    fontSize: "0.9rem",
                    marginBottom: "1rem",
                  }}
                />
              </div>

              <button
                type="button"
                onClick={() => markAsSaved(key.id)}
                disabled={disabled || key.mnemonic.length === 0}
                className="btn"
                style={{
                  backgroundColor: key.saved ? "#28a745" : "#0070f3",
                  opacity: disabled || key.mnemonic.length === 0 ? 0.7 : 1,
                  cursor:
                    disabled || key.mnemonic.length === 0
                      ? "not-allowed"
                      : "pointer",
                }}
              >
                {key.saved ? "✓ Saved" : "Mark as Saved"}
              </button>
              {/* {!key.saved && (
                <div
                  style={{
                    padding: "0.75rem",
                    backgroundColor: "#fff3cd",
                    border: "1px solid #ffeaa7",
                    borderRadius: "4px",
                    marginBottom: "1rem",
                  }}
                >
                  <strong>⚠️ Important:</strong> Save your recovery phrase in a
                  secure location.
                </div>
              )} */}
            </div>
          ))}
        </div>
      )}

      {generatedKeys.length === 0 && (
        <div
          style={{
            padding: "2rem",
            textAlign: "center",
            color: "#666",
            border: "2px dashed #ddd",
            borderRadius: "6px",
          }}
        >
          <p>No backup keys generated yet.</p>
          <p style={{ fontSize: "0.9rem" }}>
            Click "Generate New Backup Key" to create your first backup key.
          </p>
        </div>
      )}

      <div style={{ display: "flex", gap: "0.5rem", marginBottom: "1rem" }}>
        <button
          type="button"
          onClick={generateNewKey}
          className="btn secondary"
          disabled={disabled}
          style={{
            opacity: disabled ? 0.6 : 1,
            cursor: disabled ? "not-allowed" : "pointer",
          }}
        >
          + Generate New Backup Key
        </button>
        <button
          type="button"
          onClick={() => setShowImportForm(true)}
          className="btn secondary"
          disabled={disabled}
          style={{
            opacity: disabled ? 0.6 : 1,
            cursor: disabled ? "not-allowed" : "pointer",
          }}
        >
          Import Backup Key
        </button>
      </div>

      {/* Import Form */}
      {showImportForm && (
        <div
          style={{
            marginBottom: "1rem",
            padding: "1rem",
            border: "1px solid #ddd",
            borderRadius: "4px",
            backgroundColor: "#f9f9f9",
          }}
        >
          <h4 style={{ margin: "0 0 1rem 0" }}>Import Backup Key</h4>
          <div style={{ marginBottom: "1rem" }}>
            <label
              htmlFor="importAgeKey"
              style={{ display: "block", marginBottom: "0.5rem" }}
            >
              Age Key (age1...):
            </label>
            <input
              id="importAgeKey"
              type="text"
              value={importedAgeKey}
              onChange={(e) => {
                setImportedAgeKey(e.target.value);
                setImportError(""); // Clear error when user types
              }}
              placeholder="age1..."
              disabled={disabled}
              style={{
                width: "100%",
                padding: "0.5rem",
                border: importError ? "1px solid #dc3545" : "1px solid #ddd",
                borderRadius: "4px",
                fontFamily: "monospace",
                fontSize: "0.9rem",
                marginBottom: "1rem",
              }}
            />

            <input
              id="importNickname"
              type="text"
              value={importNickname}
              onChange={(e) =>
                setImportNickname(e.target.value.replace(/\s+/g, ""))
              }
              placeholder="Optional nickname"
              disabled={disabled}
              style={{
                width: "100%",
                padding: "0.5rem",
                border: "1px solid #ddd",
                borderRadius: "4px",
                fontSize: "0.9rem",
              }}
            />
            {importError && (
              <div
                style={{
                  color: "#dc3545",
                  fontSize: "0.85rem",
                  marginTop: "0.5rem",
                }}
              >
                {importError}
              </div>
            )}
          </div>
          <div style={{ display: "flex", gap: "0.5rem" }}>
            <button
              type="button"
              className="btn"
              onClick={importBackupKey}
              disabled={disabled || !importedAgeKey.trim()}
            >
              Import Key
            </button>
            <button
              type="button"
              className="btn secondary"
              onClick={() => {
                setShowImportForm(false);
                setImportedAgeKey("");
                setImportNickname("");
                setImportError("");
              }}
              disabled={disabled}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {generatedKeys.filter((key) => key.saved).length > 0 && (
        <div
          style={{
            padding: "1rem",
            backgroundColor: "#d4edda",
            border: "1px solid #c3e6cb",
            borderRadius: "4px",
            marginTop: "1rem",
          }}
        >
          <strong>
            ✓ {generatedKeys.filter((key) => key.saved).length} backup key(s)
            saved
          </strong>
          <p style={{ fontSize: "0.9rem", margin: "0.5rem 0 0 0" }}>
            These keys will be used for encrypted backups. It's recommended to
            create at least 2 backup keys.
          </p>
        </div>
      )}
    </div>
  );
}
