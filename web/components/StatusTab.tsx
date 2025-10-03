import { useEffect, useRef, useState } from "react";

import { PanelInfo, HealthInfo } from "../utils/types";
import { panelApiClient } from "../utils/panel-client";
import PanelStatus from "./PanelStatus";
import TreasuryLogsTab from "./TreasuryLogsTab";

interface Props {
  panelInfo: PanelInfo | null;
  loadingPanelInfo: boolean;
  treasuryService: any;
  startTreasuryService: any;
  treasuryHealth: HealthInfo | null;
  loadingServices: boolean;
  fetchServiceStatus: () => Promise<void>;
}

export default function StatusTab({
  panelInfo,
  loadingPanelInfo,
  treasuryService,
  startTreasuryService,
  treasuryHealth,
  loadingServices,
  fetchServiceStatus,
}: Props) {
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const [isManualRefreshing, setIsManualRefreshing] = useState(false);
  const [treasuryData, setTreasuryData] = useState<any>(null);
  const [copiedItems, setCopiedItems] = useState<Set<string>>(new Set());

  // Fetch treasury data
  const fetchTreasuryData = async () => {
    try {
      const data = await panelApiClient.getTreasuryInfo();
      setTreasuryData(data);
    } catch (error) {
      console.error("Failed to fetch treasury data:", error);
      setTreasuryData(null);
    }
  };

  // Set up automatic refresh every 1 second
  useEffect(() => {
    // Initial fetch
    fetchServiceStatus();
    fetchTreasuryData();

    // Set up interval
    intervalRef.current = setInterval(() => {
      fetchServiceStatus();
      fetchTreasuryData();
    }, 1000);

    // Cleanup on unmount
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Empty dependency array to run only once

  // Determine health status for color coding
  const getHealthBackgroundColor = () => {
    if (!treasuryHealth) return "#f8f9fa"; // default light gray

    // Check if health response indicates success
    const isHealthy = treasuryHealth.status_code === 200;

    return isHealthy ? "#d4edda" : "#f8d7da"; // light green : light red
  };

  // Handle manual refresh
  const handleManualRefresh = async () => {
    setIsManualRefreshing(true);
    try {
      await Promise.all([fetchServiceStatus(), fetchTreasuryData()]);
    } finally {
      setIsManualRefreshing(false);
    }
  };

  // Handle copying JSON data
  const handleCopy = async (data: any, itemName: string) => {
    try {
      const jsonString = JSON.stringify(data, null, 2);
      await navigator.clipboard.writeText(jsonString);
      setCopiedItems((prev) => new Set(prev).add(itemName));
      setTimeout(() => {
        setCopiedItems((prev) => {
          const newSet = new Set(prev);
          newSet.delete(itemName);
          return newSet;
        });
      }, 2000);
    } catch (err) {
      console.error("Failed to copy to clipboard:", err);
    }
  };
  return (
    <div>
      {loadingPanelInfo ? (
        <div className="card">
          <div style={{ textAlign: "center", padding: "2rem" }}>
            <span className="loading"></span>
            Loading panel information...
          </div>
        </div>
      ) : (
        <>
          <PanelStatus panelInfo={panelInfo} />

          <div className="card">
            <h3>Service Status</h3>
            <div
              style={{
                display: "grid",
                gap: "1rem",
                width: "100%",
                minWidth: 0,
              }}
            >
              {treasuryService && (
                <div>
                  <h4>Treasury Service</h4>
                  <p>
                    <strong>Status:</strong> {treasuryService.active_state}
                  </p>
                  {treasuryService.Description && (
                    <p>
                      <strong>Description:</strong>{" "}
                      {treasuryService.Description}
                    </p>
                  )}
                </div>
              )}

              {startTreasuryService && (
                <div>
                  <h4>Start Treasury Service</h4>
                  <p>
                    <strong>Status:</strong> {startTreasuryService.active_state}
                  </p>
                  {startTreasuryService.Description && (
                    <p>
                      <strong>Description:</strong>{" "}
                      {startTreasuryService.Description}
                    </p>
                  )}
                </div>
              )}

              {treasuryService?.active_state === "active" && treasuryHealth && (
                <div style={{ minWidth: 0, width: "100%" }}>
                  <h4>Treasury Health</h4>
                  <div style={{ position: "relative" }}>
                    <button
                      onClick={() =>
                        handleCopy(treasuryHealth.json, "treasury-health")
                      }
                      style={{
                        position: "absolute",
                        top: "0.5rem",
                        right: "0.5rem",
                        padding: "0.25rem 0.5rem",
                        border: `1px solid ${
                          copiedItems.has("treasury-health")
                            ? "#28a745"
                            : "#0070f3"
                        }`,
                        borderRadius: "4px",
                        background: copiedItems.has("treasury-health")
                          ? "#d4edda"
                          : "rgba(255, 255, 255, 0.9)",
                        color: copiedItems.has("treasury-health")
                          ? "#155724"
                          : "#0070f3",
                        cursor: "pointer",
                        fontSize: "0.75rem",
                        fontWeight: "500",
                        transition: "all 0.2s ease",
                        minWidth: "70px",
                        zIndex: 10,
                        boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
                      }}
                    >
                      {copiedItems.has("treasury-health")
                        ? "✓ Copied!"
                        : "Copy"}
                    </button>
                    <pre
                      style={{
                        background: getHealthBackgroundColor(),
                        padding: "1rem",
                        borderRadius: "4px",
                        overflow: "auto",
                        fontSize: "0.9rem",
                        width: "100%",
                        minWidth: 0,
                        maxWidth: "100%",
                        boxSizing: "border-box",
                        whiteSpace: "pre-wrap",
                        wordWrap: "break-word",
                        overflowWrap: "break-word",
                        margin: 0,
                      }}
                    >
                      {JSON.stringify(treasuryHealth.json, null, 2)}
                    </pre>
                  </div>
                </div>
              )}

              {treasuryData && (
                <div style={{ minWidth: 0, width: "100%" }}>
                  <h4>Treasury Data</h4>
                  <div style={{ position: "relative" }}>
                    <button
                      onClick={() => handleCopy(treasuryData, "treasury-data")}
                      style={{
                        position: "absolute",
                        top: "0.5rem",
                        right: "0.5rem",
                        padding: "0.25rem 0.5rem",
                        border: `1px solid ${
                          copiedItems.has("treasury-data")
                            ? "#28a745"
                            : "#0070f3"
                        }`,
                        borderRadius: "4px",
                        background: copiedItems.has("treasury-data")
                          ? "#d4edda"
                          : "rgba(255, 255, 255, 0.9)",
                        color: copiedItems.has("treasury-data")
                          ? "#155724"
                          : "#0070f3",
                        cursor: "pointer",
                        fontSize: "0.75rem",
                        fontWeight: "500",
                        transition: "all 0.2s ease",
                        minWidth: "70px",
                        zIndex: 10,
                        boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
                      }}
                    >
                      {copiedItems.has("treasury-data") ? "✓ Copied!" : "Copy"}
                    </button>
                    <pre
                      style={{
                        background: "#f8f9fa",
                        padding: "1rem",
                        borderRadius: "4px",
                        overflow: "auto",
                        fontSize: "0.9rem",
                        width: "100%",
                        minWidth: 0,
                        maxWidth: "100%",
                        boxSizing: "border-box",
                        whiteSpace: "pre-wrap",
                        wordWrap: "break-word",
                        overflowWrap: "break-word",
                        margin: 0,
                      }}
                    >
                      {JSON.stringify(treasuryData, null, 2)}
                    </pre>
                  </div>
                </div>
              )}
            </div>

            <button
              className="btn secondary"
              onClick={handleManualRefresh}
              disabled={isManualRefreshing}
              style={{ marginTop: "1rem" }}
            >
              {isManualRefreshing ? "Refreshing..." : "Refresh Status"}
            </button>
          </div>

          {/* Treasury Logs Section */}
          <TreasuryLogsTab
            active={treasuryService?.active_state === "active"}
            endpointType="containers"
            resourceName="treasury"
            displayName="Treasury Logs"
          />
          <TreasuryLogsTab
            active={treasuryService?.active_state === "active"}
            endpointType="services"
            resourceName="treasury.service"
            displayName="Service Logs"
          />
        </>
      )}
    </div>
  );
}
