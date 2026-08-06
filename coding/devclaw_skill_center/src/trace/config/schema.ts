export interface TosConfig {
  bucket: string;
  region: string;
  endpoint?: string;
  accessKey?: string;
  secretKey?: string;
}

export interface PrivacyConfig {
  collect_content: boolean;
  allowed_dirs: string[];
}

export interface FornaxConfig {
  ak?: string;
  sk?: string;
  region?: string;
  endpoint?: string;
}

export interface WorkspaceConfig {
  repos?: string[];   // relative paths to sub-repos under cwd
}

export interface TraceConfig {
  tos: TosConfig;
  privacy: PrivacyConfig;
  fornax?: FornaxConfig;
  workspace?: WorkspaceConfig;
}
