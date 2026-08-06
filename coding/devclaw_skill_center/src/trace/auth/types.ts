export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresAt: number; // Unix timestamp (seconds)
  userInfo?: UserInfo; // cached user info from SSO userinfo endpoint
}

export interface UserInfo {
  userId?: string;       // email prefix (e.g. "yaoqiyu")
  email?: string;        // full email
  nickname?: string;     // real name from SSO (e.g. "钟智伟")
  department?: string;   // department path from SSO (e.g. "产品研发和工程架构-Dev Infra-AI Platform-X")
}

export interface DeviceAuthResponse {
  deviceCode: string;
  userCode: string;
  verificationUri: string;
  verificationUriComplete?: string; // RFC 8628: URI with user_code pre-embedded
  expiresIn: number; // seconds
  interval: number; // polling interval in seconds
}

export interface TokenResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number; // seconds
}
