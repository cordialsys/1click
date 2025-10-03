import { useState, useEffect, useRef } from "react";
import { panelApiClient } from "../utils/panel-client";
import { AnsiUp } from "ansi_up";

interface Props {
  active: boolean;
  endpointType: "services" | "containers";
  resourceName: string;
  displayName?: string;
  subTitle?: string;
}

export default function TreasuryLogsTab({
  active,
  endpointType,
  resourceName,
  displayName,
  subTitle,
}: Props) {
  const [showLogs, setShowLogs] = useState(active);
  const [logs, setLogs] = useState<string>("");
  const [htmlLogs, setHtmlLogs] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [copiedLogs, setCopiedLogs] = useState(false);
  const [shouldAutoScroll, setShouldAutoScroll] = useState(true);
  const logsIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);
  const ansiUpRef = useRef<AnsiUp | null>(null);

  // Initialize AnsiUp instance
  if (!ansiUpRef.current) {
    ansiUpRef.current = new AnsiUp();
  }

  // Escape HTML characters to prevent injection
  const escapeHtml = (text: string): string => {
    const div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  };

  // Check if user has scrolled away from bottom
  const handleScroll = () => {
    const container = logsContainerRef.current;
    if (!container) return;

    const isAtBottom =
      container.scrollTop + container.clientHeight >=
      container.scrollHeight - 5; // 5px tolerance
    setShouldAutoScroll(isAtBottom);
  };

  // Auto-scroll to bottom if enabled
  const scrollToBottom = () => {
    const container = logsContainerRef.current;
    if (!container || !shouldAutoScroll) return;

    container.scrollTop = container.scrollHeight;
  };

  // Fetch logs from the configured endpoint
  const fetchLogs = async () => {
    try {
      let logData: string;
      if (endpointType === "services") {
        logData = await panelApiClient.getServiceLogs(resourceName);
      } else {
        logData = await panelApiClient.getContainerLogs(resourceName);
      }
      setLogs(logData);

      // Escape HTML characters first, then convert ANSI escape codes to HTML
      const escapedLogs = escapeHtml(logData);
      const convertedHtml =
        ansiUpRef.current?.ansi_to_html(escapedLogs) || escapedLogs;
      setHtmlLogs(convertedHtml);
    } catch (error) {
      console.error(`Failed to fetch ${endpointType} logs:`, error);
      const errorMsg = `Error fetching logs: ${error}`;
      setLogs(errorMsg);
      const escapedError = escapeHtml(errorMsg);
      setHtmlLogs(escapedError);
    }
  };

  // Handle copy logs
  const handleCopyLogs = async () => {
    try {
      const container = logsContainerRef.current;
      if (container) {
        // Get the plain text content from the container, excluding HTML tags
        const textContent = container.textContent || container.innerText || "";
        await navigator.clipboard.writeText(textContent);
      } else {
        // Fallback to original logs if container not available
        await navigator.clipboard.writeText(logs);
      }
      setCopiedLogs(true);
      setTimeout(() => {
        setCopiedLogs(false);
      }, 2000);
    } catch (err) {
      console.error("Failed to copy logs to clipboard:", err);
    }
  };

  // Set up/cleanup logs refresh interval
  useEffect(() => {
    if (showLogs && active) {
      setLoading(true);
      setShouldAutoScroll(true); // Reset auto-scroll when opening logs
      // Initial fetch
      fetchLogs().finally(() => setLoading(false));

      // Set up 1-second refresh interval
      logsIntervalRef.current = setInterval(() => {
        fetchLogs();
      }, 1000);
    } else {
      // Clear interval when logs are hidden or service inactive
      if (logsIntervalRef.current) {
        clearInterval(logsIntervalRef.current);
        logsIntervalRef.current = null;
      }
    }

    // Cleanup on unmount or when effect re-runs
    return () => {
      if (logsIntervalRef.current) {
        clearInterval(logsIntervalRef.current);
        logsIntervalRef.current = null;
      }
    };
  }, [showLogs, active]);

  // Scroll to bottom when logs are first loaded or updated
  useEffect(() => {
    if (htmlLogs && shouldAutoScroll) {
      setTimeout(scrollToBottom, 10); // Small delay to ensure DOM is rendered
    }
  }, [htmlLogs, shouldAutoScroll]);

  // Reset auto-scroll when logs section is toggled open
  useEffect(() => {
    if (showLogs) {
      setShouldAutoScroll(true);
      // Scroll to bottom after a short delay to ensure content is rendered
      setTimeout(() => {
        scrollToBottom();
      }, 50);
    }
  }, [showLogs]);

  // Don't render if not active
  if (!active) {
    return null;
  }

  return (
    <div className="card" style={{ marginTop: "1rem" }}>
      <h3
        onClick={() => setShowLogs(!showLogs)}
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
            transform: showLogs ? "rotate(90deg)" : "rotate(0deg)",
            transition: "transform 0.2s",
          }}
        >
          ▶
        </span>
        {displayName || `${resourceName} Logs`}
      </h3>
      <p
        style={{
          fontSize: "0.9rem",
          color: "#666",
          marginBottom: showLogs ? "1rem" : 0,
          marginTop: "0.5rem",
        }}
      >
        {subTitle === undefined ? "Refreshes every second" : subTitle}
      </p>

      {showLogs && (
        <div style={{ marginTop: "1rem" }}>
          {loading && (
            <div style={{ textAlign: "center", padding: "1rem" }}>
              <span className="loading"></span>
              Loading logs...
            </div>
          )}

          <div style={{ position: "relative" }}>
            <button
              onClick={handleCopyLogs}
              style={{
                position: "absolute",
                top: "0.5rem",
                right: "0.5rem",
                padding: "0.25rem 0.5rem",
                border: `1px solid ${copiedLogs ? "#28a745" : "#0070f3"}`,
                borderRadius: "4px",
                background: copiedLogs ? "#d4edda" : "rgba(255, 255, 255, 0.9)",
                color: copiedLogs ? "#155724" : "#0070f3",
                cursor: "pointer",
                fontSize: "0.75rem",
                fontWeight: "500",
                transition: "all 0.2s ease",
                minWidth: "70px",
                zIndex: 10,
                boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
              }}
            >
              {copiedLogs ? "✓ Copied!" : "Copy"}
            </button>
            <div
              ref={logsContainerRef}
              className="treasury-logs-container"
              onScroll={handleScroll}
              style={{
                background: "#000000",
                color: "#ffffff",
                padding: "1rem",
                borderRadius: "4px",
                overflow: "auto",
                fontSize: "0.85rem",
                width: "100%",
                minWidth: 0,
                maxWidth: "100%",
                maxHeight: "480px",
                boxSizing: "border-box",
                margin: 0,
                fontFamily: "monospace",
                lineHeight: "1.4",
                border: "1px solid #e9ecef",
                whiteSpace: "pre-wrap",
                wordWrap: "break-word",
                overflowWrap: "break-word",
              }}
              dangerouslySetInnerHTML={{
                __html: htmlLogs || "No logs available",
              }}
            />
            <style />
          </div>
        </div>
      )}
    </div>
  );
}
