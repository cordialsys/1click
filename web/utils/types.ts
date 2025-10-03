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
  web_invite?: string;
  cli_invite?: string;
}

export interface PanelInfo {
  treasury_home: string;
  supervisor_home: string;
  binary_dir: string;
  panel_dir: string;
  backup_dir: string;
  treasury_user: string;
  api_key_id?: string;
  node_id?: number;
  treasury_id?: string;
  baks: Array<{
    id: string;
    key: string;
  }>;
  network: string;
  otel_enabled: boolean;
  treasury_size?: number;
  state: "inactive" | "generated" | "active" | "sealed" | "stopped";
  recipient: string;
  ear_secret: string;
  users?: User[];
  blueprint?: "production" | "demo";
}

export interface HealthInfo {
  status_code: number;
  json: any;
}
