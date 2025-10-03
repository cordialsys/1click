import { PanelInfo, HealthInfo } from "./types";

export interface S3Object {
  ChecksumAlgorithm: string | null;
  ETag: string;
  Key: string;
  LastModified: string;
  Owner: any | null;
  RestoreStatus: any | null;
  Size: number;
  StorageClass: string;
}

export interface S3ListResponse {
  CommonPrefixes: any[] | null;
  Contents: S3Object[] | null;
  Delimiter: string;
  EncodingType: string;
  IsTruncated: boolean;
  Marker: string | null;
  MaxKeys: number | null;
  Name: string;
  NextMarker: string | null;
  Prefix: string;
  RequestCharged: string;
  ResultMetadata: any;
}

export interface BackupKey {
  key: string;
  id?: string;
}

export interface ActivationApiKeyRequest {
  api_key: string;
  demo_policy?: boolean;
}

export interface ActivationBackupRequest {
  baks: BackupKey[];
}

export interface ActivationOtelRequest {
  enabled: boolean;
}

export interface SetEncryptionAtRestRequest {
  ear_secret: string;
}

export interface RestoreSnapshotRequest {
  s3_key: string;
  encrypted_secret_phrase: string;
}

export interface RestoreMissingKeysRequest {
  encrypted_secret_phrase: string;
}

export interface RestoreMissingKeysResponse {
  active_keys: number;
  backed_up_keys: number;
  imported_keys: number;
}

export interface BootcStatusResponse {
  [key: string]: any;
}

export interface User {
  create_time: string;
  creator: string;
  display_name?: string;
  emails: string[];
  first_name: string;
  last_name?: string;
  name: string;
  organizations: Record<string, string>;
  primary_email: string;
  update_time: string;
}

export interface UsersResponse {
  users: User[];
  next_page_token?: string;
}

export interface SealRequest {
  blueprint: "production" | "demo";
  users?: User[];
  roles?: string[];
}

export interface ServiceInfo {
  active_state: string;
  Description?: string;
}

export type ServiceName = "treasury.service" | "start-treasury.service";

export class PanelApiClient {
  private apiHost: string;

  constructor(apiHost?: string) {
    this.apiHost = apiHost || this.getApiHost();
  }

  private getApiHost(): string {
    if (typeof window !== "undefined") {
      // Client-side: use environment variable or same host
      return process.env.NEXT_PUBLIC_API_HOST || window.location.origin;
    }
    // Server-side: use environment variable or empty (relative URLs)
    return process.env.NEXT_PUBLIC_API_HOST || "";
  }

  private async makeRequest<T = any>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.apiHost}${endpoint}`;
    const response = await fetch(url, {
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(error || `HTTP ${response.status}`);
    }

    const contentType = response.headers.get("content-type");
    if (contentType && contentType.includes("application/json")) {
      return response.json();
    }

    return response as any;
  }

  private async makeRequestText(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<string> {
    const url = `${this.apiHost}${endpoint}`;
    const response = await fetch(url, {
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
      ...options,
    });

    // if (!response.ok) {
    //   const error = await response.text();
    //   throw new Error(error || `HTTP ${response.status}`);
    // }

    return response.text();
  }

  // Panel Info
  async getPanelInfo(): Promise<PanelInfo> {
    return this.makeRequest<PanelInfo>("/v1/panel");
  }

  // Treasury Operations
  async getTreasuryInfo(): Promise<any> {
    return this.makeRequest("/v1/treasury");
  }

  async generateTreasury(): Promise<void> {
    await this.makeRequest("/v1/treasury", { method: "POST" });
  }

  async deleteTreasury(): Promise<void> {
    await this.makeRequest("/v1/treasury", { method: "DELETE" });
  }

  async completeTreasury(): Promise<void> {
    await this.makeRequest("/v1/treasury/complete-and-start", {
      method: "POST",
    });
  }

  async getTreasuryHealth(verbose: boolean = true): Promise<HealthInfo> {
    const query = verbose ? "?verbose" : "";
    const response = await fetch(`${this.apiHost}/v1/treasury/healthy${query}`);
    return {
      status_code: response.status,
      json: await response.json(),
    };
  }

  async syncPeers(): Promise<void> {
    await this.makeRequest("/v1/treasury/peers/sync", { method: "POST" });
  }

  // Encryption at Rest Endpoints
  async setEncryptionAtRest(
    request: SetEncryptionAtRestRequest
  ): Promise<void> {
    await this.makeRequest("/v1/panel/ear", {
      method: "PUT",
      body: JSON.stringify(request),
    });
  }

  async deleteEncryptionAtRest(): Promise<void> {
    await this.makeRequest("/v1/panel/ear", { method: "DELETE" });
  }

  // Activation Endpoints
  async activateApiKey(request: ActivationApiKeyRequest): Promise<void> {
    await this.makeRequest("/v1/activate/api-key", {
      method: "POST",
      body: JSON.stringify(request),
    });
  }

  async downloadBinaries(version?: string): Promise<Response> {
    const query = version && version !== "latest" ? `?version=${version}` : "";
    return this.makeRequest(`/v1/activate/binaries${query}`, {
      method: "POST",
    });
  }

  async setupNetwork(): Promise<void> {
    await this.makeRequest("/v1/activate/network", { method: "POST" });
  }

  async configureBackup(request: ActivationBackupRequest): Promise<void> {
    await this.makeRequest("/v1/activate/backup", {
      method: "POST",
      body: JSON.stringify(request),
    });
  }

  async configureOtel(request: ActivationOtelRequest): Promise<void> {
    await this.makeRequest("/v1/activate/otel", {
      method: "POST",
      body: JSON.stringify(request),
    });
  }

  // Service Management
  async getService(serviceName: ServiceName): Promise<ServiceInfo> {
    return this.makeRequest<ServiceInfo>(`/v1/services/${serviceName}`);
  }

  async restartService(serviceName: ServiceName): Promise<void> {
    await this.makeRequest(`/v1/services/${serviceName}/restart`, {
      method: "POST",
    });
  }

  async stopService(serviceName: ServiceName): Promise<void> {
    await this.makeRequest(`/v1/services/${serviceName}/stop`, {
      method: "POST",
    });
  }

  async startService(serviceName: ServiceName): Promise<void> {
    await this.makeRequest(`/v1/services/${serviceName}/start`, {
      method: "POST",
    });
  }

  // S3 & Backup Operations
  async listS3Objects(
    options: { prefix?: string; marker?: string } = {}
  ): Promise<S3ListResponse> {
    const params = new URLSearchParams();
    if (options.prefix) params.append("prefix", options.prefix);
    if (options.marker) params.append("marker", options.marker);

    const query = params.toString() ? `?${params.toString()}` : "";
    return this.makeRequest<S3ListResponse>(`/v1/s3/objects${query}`);
  }

  async downloadS3Object(fileKey: string): Promise<Response> {
    const params = new URLSearchParams();
    params.append("key", fileKey);

    return this.makeRequest(`/v1/s3/object?${params.toString()}`, {
      method: "GET",
    });
  }

  async restoreFromSnapshot(request: RestoreSnapshotRequest): Promise<void> {
    await this.makeRequest("/v1/backup/restore", {
      method: "POST",
      body: JSON.stringify(request),
    });
  }

  async restoreMissingKeys(
    request: RestoreMissingKeysRequest
  ): Promise<RestoreMissingKeysResponse> {
    return this.makeRequest<RestoreMissingKeysResponse>(
      "/v1/backup/restore-missing-keys",
      {
        method: "POST",
        body: JSON.stringify(request),
      }
    );
  }

  async takeSnapshot(
    snapshotId: string,
    bakKey: string,
    download: boolean = false
  ): Promise<Response> {
    const params = new URLSearchParams();
    if (bakKey) {
      params.append("bak", bakKey);
    }
    if (download) {
      params.append("download", "");
    }

    return this.makeRequest(
      `/v1/backup/snapshot/${encodeURIComponent(
        snapshotId
      )}?${params.toString()}`,
      {
        method: "POST",
      }
    );
  }

  async uploadSnapshot(snapshotId: string, file: File): Promise<void> {
    const url = `${this.apiHost}/v1/backup/snapshot/${encodeURIComponent(
      snapshotId
    )}`;

    const response = await fetch(url, {
      method: "PUT",
      body: file,
      headers: {
        "Content-Type": "application/octet-stream",
      },
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(error || `HTTP ${response.status}`);
    }
  }

  // Bootc Management
  async getBootcStatus(): Promise<string> {
    return this.makeRequestText("/v1/bootc/status");
  }

  async checkBootcUpdate(): Promise<void> {
    await this.makeRequest("/v1/bootc/upgrade/check", { method: "POST" });
  }

  async stageBootcUpdate(): Promise<void> {
    await this.makeRequest("/v1/bootc/upgrade/stage", { method: "POST" });
  }

  async applyBootcUpdate(): Promise<void> {
    await this.makeRequest("/v1/bootc/upgrade/apply", { method: "POST" });
  }

  async stageBootcRollback(): Promise<void> {
    await this.makeRequest("/v1/bootc/rollback/stage", { method: "POST" });
  }

  async applyBootcRollback(): Promise<void> {
    await this.makeRequest("/v1/bootc/rollback/apply", { method: "POST" });
  }

  // Admin/Users Management
  async getAdminUsers(pageToken?: string): Promise<UsersResponse> {
    const params = new URLSearchParams();
    if (pageToken) params.append("page_token", pageToken);
    const query = params.toString() ? `?${params.toString()}` : "";
    return this.makeRequest<UsersResponse>(`/v1/admin/users${query}`);
  }

  async sealPanel(request: SealRequest): Promise<void> {
    await this.makeRequest("/v1/panel/seal", {
      method: "POST",
      body: JSON.stringify(request),
    });
  }

  async getServiceLogs(serviceName: string): Promise<string> {
    return this.makeRequestText(`/v1/services/${serviceName}/logs`);
  }

  async getContainerLogs(containerName: string): Promise<string> {
    return this.makeRequestText(`/v1/containers/${containerName}/logs`);
  }
}

// Export a default instance
export const panelApiClient = new PanelApiClient();
