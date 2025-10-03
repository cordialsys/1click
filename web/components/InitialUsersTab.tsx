import { useState, useEffect } from "react";
import { panelApiClient, User, UsersResponse } from "../utils/panel-client";
import { PanelInfo } from "../utils/types";
import TreasuryLogsTab from "./TreasuryLogsTab";

interface Props {
  panelInfo: PanelInfo | null;
}

export default function InitialUsersTab({ panelInfo }: Props) {
  const panelState = panelInfo?.state;
  const [users, setUsers] = useState<User[]>([]);
  const [selectedUsers, setSelectedUsers] = useState<string[]>([]);
  const [selectedRoles, setSelectedRoles] = useState<{
    root: boolean;
    "co-root": boolean;
    operator: boolean;
  }>({
    root: true,
    "co-root": true,
    operator: false,
  });
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [sealing, setSealing] = useState(false);
  const [policyBlueprint, setPolicyBlueprint] = useState<"production" | "demo">(
    "production"
  );
  const [nextPageToken, setNextPageToken] = useState<string | undefined>();
  const [hasMorePages, setHasMorePages] = useState(false);
  const [status, setStatus] = useState<{
    type: "success" | "error" | "info" | null;
    message: string;
  }>({ type: null, message: "" });
  const [showLogs, setShowLogs] = useState(false);
  const [copiedInvites, setCopiedInvites] = useState<Set<string>>(new Set());

  const fetchUsers = async (pageToken?: string, append = false) => {
    if (!append) setLoading(true);
    else setLoadingMore(true);

    try {
      const response: UsersResponse = await panelApiClient.getAdminUsers(
        pageToken
      );

      if (append) {
        setUsers((prev) => [...prev, ...response.users]);
      } else {
        setUsers(response.users);
      }

      setNextPageToken(response.next_page_token);
      setHasMorePages(!!response.next_page_token);
    } catch (error) {
      console.error("Failed to fetch users:", error);
      setStatus({
        type: "error",
        message: `Failed to fetch users: ${error}`,
      });
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  const loadMoreUsers = () => {
    if (nextPageToken && !loadingMore) {
      fetchUsers(nextPageToken, true);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleUserToggle = (userName: string) => {
    setSelectedUsers((prev) => {
      const newSelection = prev.includes(userName)
        ? prev.filter((u) => u !== userName)
        : [...prev, userName];

      return newSelection;
    });
  };

  const startLogMonitoring = () => {
    setShowLogs(true);
  };

  const stopLogMonitoring = () => {
    setShowLogs(false);
  };

  const handleSeal = async () => {
    if (policyBlueprint === "production" && selectedUsers.length === 0) {
      setStatus({
        type: "error",
        message:
          "Please select at least one user for production blueprint, or choose demo blueprint.",
      });
      return;
    }

    // Validate role requirements for production blueprint
    if (policyBlueprint === "production") {
      const hasSelectedRoles =
        selectedRoles.root ||
        selectedRoles["co-root"] ||
        selectedRoles.operator;
      if (!hasSelectedRoles) {
        setStatus({
          type: "error",
          message: "Please select at least one role for the initial users.",
        });
        return;
      }

      // If only co-root is selected (no root), need at least 2 users for approval workflows
      if (
        !selectedRoles.root &&
        selectedRoles["co-root"] &&
        !selectedRoles.operator &&
        selectedUsers.length < 2
      ) {
        setStatus({
          type: "error",
          message:
            "Co-root only configuration requires at least 2 users for approval workflows.",
        });
        return;
      }
    }

    setSealing(true);
    setStatus({ type: null, message: "" });

    // Start log monitoring
    startLogMonitoring();

    try {
      const selectedUsersData =
        policyBlueprint === "production"
          ? selectedUsers.map(
              (userName) => users.find((u) => u.name === userName)!
            )
          : undefined;

      const roles = [];
      if (selectedRoles.root) roles.push("root");
      if (selectedRoles["co-root"]) roles.push("co-root");
      if (selectedRoles.operator) roles.push("operator");

      await panelApiClient.sealPanel({
        blueprint: policyBlueprint,
        users: selectedUsersData,
        roles: policyBlueprint === "production" ? roles : undefined,
      });
      setStatus({
        type: "success",
        message: `Panel sealed successfully with ${policyBlueprint} blueprint!`,
      });

      // Continue monitoring logs for a bit after success to show completion
      setTimeout(() => {
        stopLogMonitoring();
      }, 10000); // Stop monitoring after 10 seconds
    } catch (error) {
      console.error("Failed to seal panel:", error);
      setStatus({
        type: "error",
        message: `Failed to seal panel: ${error}`,
      });
      stopLogMonitoring();
    } finally {
      setSealing(false);
    }
  };

  const getUserDisplayName = (user: User): string => {
    if (user.display_name) return user.display_name;
    if (user.first_name && user.last_name)
      return `${user.first_name} ${user.last_name}`;
    if (user.first_name) return user.first_name;
    return user.primary_email;
  };

  const getUserRole = (user: User): string => {
    const orgRoles = Object.values(user.organizations);
    if (orgRoles.includes("admin")) return "admin";
    if (orgRoles.includes("member")) return "member";
    return "unknown";
  };

  // Don't show loading if we have configured users for sealed/stopped states
  const hasConfiguredUsers =
    (panelState === "sealed" || panelState === "stopped") && panelInfo?.users;

  if (loading && !hasConfiguredUsers) {
    return (
      <div style={{ textAlign: "center", padding: "2rem" }}>
        <span className="loading"></span>
        Loading users...
      </div>
    );
  }

  // Show configured users when panel is sealed or stopped
  if (
    (panelState === "sealed" || panelState === "stopped") &&
    (panelInfo?.users || panelInfo?.blueprint === "demo")
  ) {
    return (
      <div>
        <div
          style={{
            padding: "1rem",
            backgroundColor: "#d4edda",
            border: `1px solid ${"#c3e6cb"}`,
            borderRadius: "4px",
            marginBottom: "1.5rem",
          }}
        >
          <h4
            style={{
              margin: "0 0 0.5rem 0",
              color: "#155724",
            }}
          >
            {"🔒 Panel Sealed"}
          </h4>
          <p
            style={{
              margin: 0,
              fontSize: "0.9rem",
              color: "#155724",
            }}
          >
            Treasury configured with a blueprint and initial users.
          </p>
        </div>

        {panelInfo?.users && panelInfo.users.length > 0 && (
          <div>
            <h4 style={{ marginBottom: "1rem" }}>Initial Users</h4>
            <div
              style={{
                border: "1px solid #e0e0e0",
                borderRadius: "4px",
                overflow: "hidden",
              }}
            >
              {panelInfo.users.map((user, index) => (
                <div
                  key={user.name}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    padding: "1rem",
                    borderBottom:
                      index < panelInfo.users!.length - 1
                        ? "1px solid #f0f0f0"
                        : "none",
                    backgroundColor: "#f8f9fa",
                  }}
                >
                  <div style={{ flex: 1 }}>
                    <div style={{ fontWeight: "500", fontSize: "1rem" }}>
                      {getUserDisplayName(user)}
                    </div>
                    <div style={{ fontSize: "0.9rem", color: "#666" }}>
                      {user.primary_email}
                    </div>
                    {user.web_invite && panelInfo.treasury_id && (
                      <div
                        style={{
                          fontSize: "0.85rem",
                          // color: "#0070f3",
                          marginTop: "0.5rem",
                        }}
                      >
                        <div
                          style={{ marginBottom: "0.25rem", fontWeight: "600" }}
                        >
                          Invite Link
                        </div>
                        <div
                          style={{
                            display: "flex",
                            alignItems: "center",
                            gap: "0.5rem",
                          }}
                        >
                          <code
                            style={{
                              flex: 1,
                              padding: "0.25rem 0.5rem",
                              backgroundColor: "#f8f9fa",
                              border: "1px solid #e9ecef",
                              borderRadius: "4px",
                              fontSize: "0.8rem",
                              wordBreak: "break-all",
                              color: "#495057",
                            }}
                          >
                            {`https://treasury.cordial.systems/treasury?invite=${user.web_invite}&treasury=${panelInfo.treasury_id}`}
                          </code>
                          <button
                            onClick={async () => {
                              const inviteUrl = `https://treasury.cordial.systems/treasury?invite=${user.web_invite}&treasury=${panelInfo.treasury_id}`;
                              try {
                                await navigator.clipboard.writeText(inviteUrl);
                                setCopiedInvites((prev) =>
                                  new Set(prev).add(user.name)
                                );
                                setTimeout(() => {
                                  setCopiedInvites((prev) => {
                                    const newSet = new Set(prev);
                                    newSet.delete(user.name);
                                    return newSet;
                                  });
                                }, 2000);
                              } catch (err) {
                                console.error(
                                  "Failed to copy to clipboard:",
                                  err
                                );
                              }
                            }}
                            style={{
                              padding: "0.25rem 0.5rem",
                              border: `1px solid ${
                                copiedInvites.has(user.name)
                                  ? "#28a745"
                                  : "#0070f3"
                              }`,
                              borderRadius: "4px",
                              background: copiedInvites.has(user.name)
                                ? "#d4edda"
                                : "white",
                              color: copiedInvites.has(user.name)
                                ? "#155724"
                                : "#0070f3",
                              cursor: "pointer",
                              fontSize: "0.75rem",
                              fontWeight: "500",
                              transition: "all 0.2s ease",
                              minWidth: "70px",
                            }}
                          >
                            {copiedInvites.has(user.name)
                              ? "✓ Copied!"
                              : "Copy"}
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {panelInfo?.blueprint === "demo" &&
          (!panelInfo?.users || panelInfo.users.length === 0) && (
            <div>
              <h4 style={{ marginBottom: "1rem" }}>Demo Blueprint</h4>
              <div
                style={{
                  padding: "1rem",
                  backgroundColor: "#fff3cd",
                  border: "1px solid #ffeaa7",
                  borderRadius: "4px",
                  fontSize: "0.9rem",
                  color: "#856404",
                }}
              >
                ℹ️ Demo blueprint configured - anyone in your organization can
                enroll without pre-selected users.
              </div>
            </div>
          )}

        {/* Log Toggle Button */}
        <div style={{ marginTop: "1.5rem", textAlign: "center" }}>
          <button
            onClick={() => setShowLogs(!showLogs)}
            style={{
              padding: "0.5rem 1rem",
              border: "1px solid #6c757d",
              borderRadius: "4px",
              background: showLogs ? "#6c757d" : "white",
              color: showLogs ? "white" : "#6c757d",
              cursor: "pointer",
              fontSize: "0.85rem",
              fontWeight: "500",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!showLogs) {
                e.currentTarget.style.backgroundColor = "#f8f9fa";
              }
            }}
            onMouseLeave={(e) => {
              if (!showLogs) {
                e.currentTarget.style.backgroundColor = "white";
              }
            }}
          >
            {showLogs ? "Hide Log" : "Show Blueprint Log"}
          </button>
        </div>

        {/* Blueprint Service Logs */}
        {showLogs && (
          // <div style={{ marginTop: "1.5rem" }}>
          <TreasuryLogsTab
            active={true}
            endpointType="services"
            resourceName="blueprint.service"
            displayName="Blueprint Log"
            subTitle=""
          />
          // </div>
        )}
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

      {/* Policy Blueprint Selection */}
      <div style={{ marginBottom: "2rem" }}>
        <h4>Policy Blueprint</h4>
        <div style={{ display: "flex", gap: "1rem", marginBottom: "1rem" }}>
          <label
            style={{ display: "flex", alignItems: "center", cursor: "pointer" }}
          >
            <input
              type="radio"
              name="policyBlueprint"
              value="production"
              checked={policyBlueprint === "production"}
              onChange={(e) =>
                setPolicyBlueprint(e.target.value as "production")
              }
              disabled={sealing}
              style={{ marginRight: "0.5rem" }}
            />
            <span>Production Blueprint</span>
          </label>
          <label
            style={{ display: "flex", alignItems: "center", cursor: "pointer" }}
          >
            <input
              type="radio"
              name="policyBlueprint"
              value="demo"
              checked={policyBlueprint === "demo"}
              onChange={(e) => setPolicyBlueprint(e.target.value as "demo")}
              disabled={sealing}
              style={{ marginRight: "0.5rem" }}
            />
            <span>Demo Blueprint</span>
          </label>
        </div>

        {policyBlueprint === "demo" && (
          <div
            style={{
              padding: "0.75rem",
              backgroundColor: "#fff3cd",
              border: "1px solid #ffeaa7",
              borderRadius: "4px",
              fontSize: "0.9rem",
              color: "#856404",
            }}
          >
            ℹ️ Demo blueprint allows anyone in your organization to enroll. No
            initial users need to be selected.
          </div>
        )}

        {policyBlueprint === "production" && (
          <div>
            <div
              style={{
                padding: "0.75rem",
                backgroundColor: "#e7f3ff",
                border: "1px solid #b8daff",
                borderRadius: "4px",
                fontSize: "0.9rem",
                color: "#004085",
              }}
            >
              ℹ️ Production blueprint requires selecting initial users who can
              manage the treasury.
              <div style={{ marginTop: "0.5rem", fontSize: "0.85rem" }}>
                <strong>Roles:</strong>
                <br />•{" "}
                <span style={{ color: "#d63384", fontWeight: "500" }}>
                  root
                </span>
                : Can perform any operation without approval
                <br />•{" "}
                <span style={{ color: "#fd7e14", fontWeight: "500" }}>
                  co-root
                </span>
                : Requires approval for sensitive operations
                <br />•{" "}
                <span style={{ color: "#198754", fontWeight: "500" }}>
                  operator
                </span>
                : Regular user with limited treasury access
              </div>
            </div>

            {/* Global Role Selection */}
            <div
              style={{
                marginTop: "1rem",
                padding: "0.75rem",
                backgroundColor: "#f8f9fa",
                border: "1px solid #e9ecef",
                borderRadius: "4px",
              }}
            >
              <h5 style={{ margin: "0 0 0.75rem 0", fontSize: "0.95rem" }}>
                Initial User Roles
              </h5>
              <p
                style={{
                  fontSize: "0.85rem",
                  color: "#666",
                  margin: "0 0 0.75rem 0",
                }}
              >
                Select which roles to start with:
              </p>
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: "0.5rem",
                }}
              >
                <label
                  style={{
                    display: "flex",
                    alignItems: "center",
                    cursor: "pointer",
                  }}
                >
                  <input
                    type="checkbox"
                    checked={selectedRoles.root}
                    onChange={(e) =>
                      setSelectedRoles((prev) => ({
                        ...prev,
                        root: e.target.checked,
                      }))
                    }
                    disabled={sealing}
                    style={{ marginRight: "0.5rem" }}
                  />
                  <span style={{ fontSize: "0.9rem" }}>
                    <span style={{ color: "#d63384", fontWeight: "500" }}>
                      root
                    </span>{" "}
                    role
                  </span>
                </label>
                <label
                  style={{
                    display: "flex",
                    alignItems: "center",
                    cursor: "pointer",
                  }}
                >
                  <input
                    type="checkbox"
                    checked={selectedRoles["co-root"]}
                    onChange={(e) =>
                      setSelectedRoles((prev) => ({
                        ...prev,
                        "co-root": e.target.checked,
                      }))
                    }
                    disabled={sealing}
                    style={{ marginRight: "0.5rem" }}
                  />
                  <span style={{ fontSize: "0.9rem" }}>
                    <span style={{ color: "#fd7e14", fontWeight: "500" }}>
                      co-root
                    </span>{" "}
                    role
                  </span>
                </label>
                <label
                  style={{
                    display: "flex",
                    alignItems: "center",
                    cursor: "pointer",
                  }}
                >
                  <input
                    type="checkbox"
                    checked={selectedRoles.operator}
                    onChange={(e) =>
                      setSelectedRoles((prev) => ({
                        ...prev,
                        operator: e.target.checked,
                      }))
                    }
                    disabled={sealing}
                    style={{ marginRight: "0.5rem" }}
                  />
                  <span style={{ fontSize: "0.9rem" }}>
                    <span style={{ color: "#198754", fontWeight: "500" }}>
                      operator
                    </span>{" "}
                    role
                  </span>
                </label>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* User Selection */}
      {policyBlueprint === "production" && (
        <div style={{ marginBottom: "2rem" }}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: "1rem",
            }}
          >
            <h4 style={{ margin: 0 }}>
              Select initial root users ({selectedUsers.length} selected)
            </h4>
            <button
              onClick={() => fetchUsers()}
              disabled={loading || sealing}
              style={{
                padding: "0.5rem 1rem",
                border: "1px solid #ddd",
                borderRadius: "4px",
                background: "#f8f9fa",
                cursor: loading || sealing ? "not-allowed" : "pointer",
                fontSize: "0.85rem",
              }}
            >
              🔄 Refresh
            </button>
          </div>

          <div
            style={{
              border: "1px solid #e0e0e0",
              borderRadius: "4px",
              maxHeight: "400px",
              overflowY: "auto",
            }}
          >
            {users.map((user, index) => (
              <div
                key={user.name}
                style={{
                  display: "flex",
                  alignItems: "center",
                  padding: "0.75rem 1rem",
                  borderBottom:
                    index < users.length - 1 ? "1px solid #f0f0f0" : "none",
                  cursor: "pointer",
                  backgroundColor: selectedUsers.includes(user.name)
                    ? "#f0f8ff"
                    : "transparent",
                  transition: "background-color 0.2s ease",
                }}
                onClick={() => !sealing && handleUserToggle(user.name)}
              >
                <input
                  type="checkbox"
                  checked={selectedUsers.includes(user.name)}
                  onChange={() => {}} // Handled by div onClick
                  disabled={sealing}
                  style={{ marginRight: "1rem" }}
                />

                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: "500", fontSize: "0.95rem" }}>
                    {getUserDisplayName(user)}
                  </div>
                  <div style={{ fontSize: "0.85rem", color: "#666" }}>
                    {user.primary_email}
                  </div>
                </div>

                <div
                  style={{
                    padding: "0.25rem 0.5rem",
                    fontSize: "0.8rem",
                    borderRadius: "12px",
                    backgroundColor:
                      getUserRole(user) === "admin" ? "#d4edda" : "#e2e3e5",
                    color:
                      getUserRole(user) === "admin" ? "#155724" : "#495057",
                    fontWeight: "500",
                  }}
                >
                  {getUserRole(user)}
                </div>
              </div>
            ))}

            {hasMorePages && (
              <div
                style={{
                  padding: "1rem",
                  textAlign: "center",
                  borderTop: "1px solid #f0f0f0",
                }}
              >
                <button
                  onClick={loadMoreUsers}
                  disabled={loadingMore || sealing}
                  style={{
                    padding: "0.5rem 1rem",
                    border: "1px solid #ddd",
                    borderRadius: "4px",
                    background: "#f8f9fa",
                    cursor: loadingMore || sealing ? "not-allowed" : "pointer",
                    fontSize: "0.9rem",
                  }}
                >
                  {loadingMore ? (
                    <>
                      <span className="loading"></span>
                      Loading...
                    </>
                  ) : (
                    "Load More Users"
                  )}
                </button>
              </div>
            )}
          </div>

          {/* Portal Callout */}
          <div
            style={{
              marginTop: "1rem",
              padding: "0.75rem",
              backgroundColor: "#f8f9fa",
              border: "1px solid #e9ecef",
              borderRadius: "4px",
              fontSize: "0.9rem",
              color: "#495057",
            }}
          >
            💡 Not seeing a user?{" "}
            <a
              href="https://portal.cordial.systems/"
              target="_blank"
              rel="noopener noreferrer"
              style={{
                color: "#0070f3",
                textDecoration: "none",
                fontWeight: "500",
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.textDecoration = "underline";
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.textDecoration = "none";
              }}
            >
              Add them on Portal first
            </a>
            .
          </div>
        </div>
      )}

      {/* Seal Button */}
      <div
        style={{
          marginTop: "2rem",
          paddingTop: "2rem",
          borderTop: "2px solid #e0e0e0",
        }}
      >
        <h4>Finalize blueprint</h4>
        <p style={{ fontSize: "0.9rem", color: "#666", marginBottom: "1rem" }}>
          This action cannot be undone, but you can always edit users and policy
          later on the Treasury app.
        </p>

        <button
          className="btn"
          onClick={handleSeal}
          disabled={
            sealing ||
            (policyBlueprint === "production" &&
              (selectedUsers.length === 0 ||
                (!selectedRoles.root &&
                  !selectedRoles["co-root"] &&
                  !selectedRoles.operator))) ||
            panelState === "generated" ||
            panelState === "stopped"
          }
          style={{
            backgroundColor: "#28a745",
            borderColor: "#28a745",
            color: "white",
            fontSize: "1rem",
            padding: "0.75rem 2rem",
            transition: "all 0.2s ease",
          }}
          onMouseEnter={(e) => {
            if (
              !sealing &&
              !(
                policyBlueprint === "production" &&
                (selectedUsers.length === 0 ||
                  (!selectedRoles.root &&
                    !selectedRoles["co-root"] &&
                    !selectedRoles.operator))
              ) &&
              panelState !== "generated" &&
              panelState !== "stopped"
            ) {
              e.currentTarget.style.backgroundColor = "#218838";
              e.currentTarget.style.borderColor = "#218838";
            }
          }}
          onMouseLeave={(e) => {
            if (
              !sealing &&
              !(
                policyBlueprint === "production" &&
                (selectedUsers.length === 0 ||
                  (!selectedRoles.root &&
                    !selectedRoles["co-root"] &&
                    !selectedRoles.operator))
              ) &&
              panelState !== "generated" &&
              panelState !== "stopped"
            ) {
              e.currentTarget.style.backgroundColor = "#28a745";
              e.currentTarget.style.borderColor = "#28a745";
            }
          }}
        >
          {sealing && <span className="loading"></span>}
          {sealing ? "Sealing Panel..." : "🔒 Seal Blueprint"}
        </button>

        {policyBlueprint === "production" && selectedUsers.length === 0 && (
          <p
            style={{
              fontSize: "0.85rem",
              color: "#dc3545",
              marginTop: "0.5rem",
            }}
          >
            Please select at least one user to seal with production blueprint.
          </p>
        )}

        {policyBlueprint === "production" &&
          selectedUsers.length > 0 &&
          !selectedRoles.root &&
          !selectedRoles["co-root"] &&
          !selectedRoles.operator && (
            <p
              style={{
                fontSize: "0.85rem",
                color: "#dc3545",
                marginTop: "0.5rem",
              }}
            >
              Please select at least one role for the initial users.
            </p>
          )}

        {panelState === "generated" && (
          <p
            style={{
              fontSize: "0.85rem",
              color: "#856404",
              marginTop: "0.5rem",
            }}
          >
            Treasury is not yet active. Blueprint sealing will be available once
            the treasury becomes active.
          </p>
        )}

        {panelState === "stopped" && (
          <p
            style={{
              fontSize: "0.85rem",
              color: "#856404",
              marginTop: "0.5rem",
            }}
          >
            Treasury is currently stopped. Blueprint sealing requires the
            treasury to be active.
          </p>
        )}

        {/* Log Toggle Button */}
        <div style={{ marginTop: "1rem", textAlign: "center" }}>
          <button
            onClick={() => setShowLogs(!showLogs)}
            style={{
              padding: "0.5rem 1rem",
              border: "1px solid #6c757d",
              borderRadius: "4px",
              background: showLogs ? "#6c757d" : "white",
              color: showLogs ? "white" : "#6c757d",
              cursor: "pointer",
              fontSize: "0.85rem",
              fontWeight: "500",
              transition: "all 0.2s ease",
            }}
            onMouseEnter={(e) => {
              if (!showLogs) {
                e.currentTarget.style.backgroundColor = "#f8f9fa";
              }
            }}
            onMouseLeave={(e) => {
              if (!showLogs) {
                e.currentTarget.style.backgroundColor = "white";
              }
            }}
          >
            {showLogs ? "Hide Logs" : "Show Blueprint Progress"}
          </button>
        </div>

        {/* Blueprint Service Logs */}
        {showLogs && (
          <div style={{ marginTop: "1.5rem" }}>
            <TreasuryLogsTab
              active={true}
              endpointType="services"
              resourceName="blueprint.service"
              displayName="Blueprint Log"
              subTitle=""
            />
          </div>
        )}
      </div>
    </div>
  );
}
