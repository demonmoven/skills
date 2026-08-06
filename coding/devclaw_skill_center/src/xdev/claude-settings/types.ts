export interface Hook {
  type: string;
  command: string;
  [key: string]: unknown;
}

export interface HookEntry {
  matcher: string;
  hooks: Hook[];
}

export interface PermissionsBlock {
  allow?: string[];
  deny?: string[];
  ask?: string[];
  defaultMode?: string;
  [key: string]: unknown;
}

export interface ClaudeSettings {
  env?: Record<string, string>;
  permissions?: PermissionsBlock;
  model?: string;
  hooks?: Record<string, HookEntry[]>;
  enabledPlugins?: Record<string, boolean>;
  extraKnownMarketplaces?: Record<string, unknown>;
  language?: string;
  alwaysThinkingEnabled?: boolean;
  autoUpdatesChannel?: string;
  skipDangerousModePermissionPrompt?: boolean;
  [key: string]: unknown;
}

export interface MergeOptions {
  /** When true, scalar/object conflicts resolve to template's value. Default: false (user-wins). */
  preferTemplate?: boolean;
}

export interface MergeReport {
  changes: string[];
  changed: boolean;
}
