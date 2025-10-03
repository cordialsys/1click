import { useState } from "react";
import { PanelInfo } from "../utils/types";

interface Props {
  panelInfo: PanelInfo | null;
}

export default function PanelStatus({ panelInfo }: Props) {
  const [copiedItems, setCopiedItems] = useState<Record<string, boolean>>({});

  const copyToClipboard = async (text: string, itemId: string) => {
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(text);
      setCopiedItems((prev) => ({ ...prev, [itemId]: true }));
      setTimeout(() => {
        setCopiedItems((prev) => ({ ...prev, [itemId]: false }));
      }, 500);
    }
  };

  if (!panelInfo) {
    return null;
  }

  return (
    <div className="card">
      <h3>Panel Status</h3>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
          gap: "1rem",
        }}
      >
        <div>
          <strong>Network:</strong> {panelInfo.network}
        </div>
        <div>
          <strong>Treasury Size:</strong> {panelInfo.treasury_size || "Not set"}
        </div>
        <div>
          <strong>Node ID:</strong> {panelInfo.node_id || "Not set"}
        </div>
        <div>
          <strong>Treasury ID:</strong>
          {panelInfo.treasury_id ? (
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "0.5rem",
                marginTop: "0.25rem",
              }}
            >
              <span style={{ fontFamily: "monospace", fontSize: "0.85rem" }}>
                {panelInfo.treasury_id}
              </span>
              <button
                type="button"
                onClick={() =>
                  copyToClipboard(panelInfo.treasury_id!, `treasury-id`)
                }
                style={{
                  padding: "0.25rem",
                  border: "1px solid #ddd",
                  borderRadius: "4px",
                  background: copiedItems[`treasury-id`] ? "#d4edda" : "white",
                  cursor: "pointer",
                  transition: "background-color 0.3s",
                  fontSize: "0.75rem",
                }}
              >
                {copiedItems[`treasury-id`] ? "✅" : "📋"}
              </button>
            </div>
          ) : (
            " Not set"
          )}
        </div>
        <div>
          <strong>State:</strong> {panelInfo.state}
        </div>
        <div style={{ gridColumn: "1 / -1" }}>
          <strong>Panel Identity:</strong>
          <div
            style={{
              fontFamily: "monospace",
              fontSize: "0.75rem",
              color: "#6c757d",
              wordBreak: "break-all",
              marginTop: "0.25rem",
            }}
          >
            {panelInfo.recipient || "Not available"}
          </div>
        </div>
        {panelInfo.baks && panelInfo.baks.length > 0 && (
          <div style={{ gridColumn: "1 / -1" }}>
            <strong>Backup Keys</strong>
            <div style={{ marginTop: "0.5rem" }}>
              {panelInfo.baks.map((bak, index) => (
                <div
                  key={index}
                  style={{
                    padding: "0.5rem",
                    border: "1px solid #ddd",
                    borderRadius: "4px",
                    marginBottom: "0.5rem",
                    backgroundColor: "#f8f9fa",
                    fontSize: "0.85rem",
                  }}
                >
                  {bak.id && (
                    <div
                      style={{ fontWeight: "bold", marginBottom: "0.25rem" }}
                    >
                      {bak.id}
                    </div>
                  )}
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "0.5rem",
                    }}
                  >
                    <div
                      style={{
                        fontFamily: "monospace",
                        wordBreak: "break-all",
                        color: "#495057",
                        flex: 1,
                      }}
                    >
                      {bak.key}
                    </div>
                    <button
                      type="button"
                      onClick={() => copyToClipboard(bak.key, `bak-${index}`)}
                      style={{
                        padding: "0.25rem",
                        border: "1px solid #ddd",
                        borderRadius: "4px",
                        background: copiedItems[`bak-${index}`]
                          ? "#d4edda"
                          : "white",
                        cursor: "pointer",
                        transition: "background-color 0.3s",
                        fontSize: "0.75rem",
                        flexShrink: 0,
                      }}
                    >
                      {copiedItems[`bak-${index}`] ? "✅" : "📋"}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
