/**
 * Type definitions for MCP Server Monitor
 */

export interface MonitorConfig {
  autoCheck: AutoCheckConfig;
  autoCorrect: AutoCorrectConfig;
  autoUpdate: AutoUpdateConfig;
  diagnostics: DiagnosticsConfig;
  mcpProxy: MCPProxyConfig;
}

export interface AutoCheckConfig {
  enabled: boolean;
  intervalMs: number;
  startupDelayMs: number;
}

export interface AutoCorrectConfig {
  enabled: boolean;
  maxRetries: number;
  retryDelayMs: number;
}

export interface AutoUpdateConfig {
  enabled: boolean;
  gitCommitBeforeChange: boolean;
  gitCommitAfterChange: boolean;
  backupConfig: boolean;
}

export interface DiagnosticsConfig {
  logPath: string;
  maxLogSizeMb: number;
  rotateCount: number;
}

export interface MCPProxyConfig {
  configPath: string;
  endpoint: string;
}

export type ServerStatus = 'connected' | 'disconnected' | 'disabled' | 'auto_disabled' | 'quarantined';

export interface ServerHealthResult {
  name: string;
  status: ServerStatus;
  healthy: boolean;
  lastError?: string;
  startupMode?: string;
  toolCount?: number;
}

export interface HealthCheckResult {
  timestamp: Date;
  total: number;
  healthyCount: number;
  unhealthyCount: number;
  servers: ServerHealthResult[];
  unhealthyServers: ServerHealthResult[];
}

export interface DiagnosticEntry {
  timestamp: Date;
  serverName: string;
  status: ServerStatus;
  error?: string;
  diagnosis: string;
  recommendation: string;
  autoCorrectAttempted?: boolean;
  autoCorrectSuccess?: boolean;
}

export interface GitCommitResult {
  success: boolean;
  commitHash?: string;
  error?: string;
}

export interface AutoCorrectResult {
  serverName: string;
  attempted: boolean;
  success: boolean;
  action?: string;
  error?: string;
}

export interface AutoUpdateResult {
  success: boolean;
  filesModified: string[];
  commitBefore?: GitCommitResult;
  commitAfter?: GitCommitResult;
  error?: string;
}
