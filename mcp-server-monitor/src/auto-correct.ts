/**
 * Auto-Correct module for automatically fixing MCP server issues
 */

import { query } from '@anthropic-ai/claude-agent-sdk';
import { readFile, writeFile } from 'fs/promises';
import { existsSync } from 'fs';
import { homedir } from 'os';
import type {
  ServerHealthResult,
  AutoCorrectConfig,
  AutoCorrectResult,
  MCPProxyConfig
} from './types.js';

export class AutoCorrectAgent {
  private config: AutoCorrectConfig;
  private mcpConfig: MCPProxyConfig;

  constructor(config: AutoCorrectConfig, mcpConfig: MCPProxyConfig) {
    this.config = config;
    this.mcpConfig = mcpConfig;
  }

  private expandPath(path: string): string {
    if (path.startsWith('~')) {
      return path.replace('~', homedir());
    }
    return path;
  }

  async attemptCorrection(server: ServerHealthResult): Promise<AutoCorrectResult> {
    if (!this.config.enabled) {
      return {
        serverName: server.name,
        attempted: false,
        success: false,
        error: 'Auto-correct is disabled'
      };
    }

    const result: AutoCorrectResult = {
      serverName: server.name,
      attempted: true,
      success: false
    };

    try {
      // Try different correction strategies based on the error
      const error = server.lastError?.toLowerCase() || '';

      if (this.canAutoCorrect(server)) {
        const correctionAction = await this.determineAction(server);

        if (correctionAction) {
          result.action = correctionAction;
          const success = await this.executeCorrection(server, correctionAction);
          result.success = success;
        } else {
          result.error = 'No suitable correction action found';
        }
      } else {
        result.error = 'Server issue cannot be auto-corrected';
      }
    } catch (error) {
      result.error = error instanceof Error ? error.message : String(error);
    }

    return result;
  }

  private canAutoCorrect(server: ServerHealthResult): boolean {
    const error = server.lastError?.toLowerCase() || '';

    // Issues that can potentially be auto-corrected
    const correctablePatterns = [
      'missing',
      'not found',
      'connection refused',
      'auto_disabled'
    ];

    return correctablePatterns.some(pattern =>
      error.includes(pattern) || server.status === 'auto_disabled'
    );
  }

  private async determineAction(server: ServerHealthResult): Promise<string | null> {
    const error = server.lastError?.toLowerCase() || '';

    if (server.status === 'auto_disabled') {
      return 'reconnect';
    }

    if (error.includes('connection refused')) {
      return 'restart_check';
    }

    if (error.includes('missing') || error.includes('not found')) {
      return 'config_check';
    }

    // Use Claude for intelligent decision
    try {
      let action: string | null = null;

      for await (const message of query({
        prompt: `Welche Auto-Correct Aktion würdest du für diesen MCP-Server-Fehler empfehlen?

Server: ${server.name}
Status: ${server.status}
Fehler: ${server.lastError}

Antworte mit EINER der folgenden Aktionen oder "none":
- reconnect: Server neu verbinden
- restart_check: Prüfen ob Server-Prozess läuft
- config_check: Konfiguration prüfen und korrigieren
- none: Keine automatische Korrektur möglich

Antwort (nur das Aktionswort):`,
        options: {
          allowedTools: [],
          maxTurns: 1
        }
      })) {
        if ('result' in message && typeof message.result === 'string') {
          const result = message.result.toLowerCase().trim();
          if (['reconnect', 'restart_check', 'config_check'].includes(result)) {
            action = result;
          }
        }
      }

      return action;
    } catch {
      return null;
    }
  }

  private async executeCorrection(server: ServerHealthResult, action: string): Promise<boolean> {
    switch (action) {
      case 'reconnect':
        return this.attemptReconnect(server);

      case 'restart_check':
        return this.checkAndRestartService(server);

      case 'config_check':
        return this.checkAndFixConfig(server);

      default:
        return false;
    }
  }

  private async attemptReconnect(server: ServerHealthResult): Promise<boolean> {
    try {
      // Use the MCP proxy's health_check tool with fix_issues flag
      for await (const message of query({
        prompt: `Verwende das mcpproxy_health_check Tool um den Server "${server.name}" zu reparieren.`,
        options: {
          allowedTools: ['Bash'],
          maxTurns: 3,
          systemPrompt: 'Du hast Zugriff auf mcpproxy CLI. Führe health-check mit --fix aus.'
        }
      })) {
        if ('result' in message) {
          console.log(`Reconnect attempt for ${server.name}:`, message.result);
        }
      }
      return true;
    } catch {
      return false;
    }
  }

  private async checkAndRestartService(server: ServerHealthResult): Promise<boolean> {
    try {
      for await (const message of query({
        prompt: `Prüfe ob der Prozess für MCP-Server "${server.name}" läuft und starte ihn ggf. neu.
Verwende 'ps aux | grep' um den Prozess zu finden.
Falls nicht gefunden, versuche ihn zu starten.`,
        options: {
          allowedTools: ['Bash', 'Read'],
          maxTurns: 5
        }
      })) {
        if ('result' in message) {
          console.log(`Service check for ${server.name}:`, message.result);
        }
      }
      return true;
    } catch {
      return false;
    }
  }

  private async checkAndFixConfig(server: ServerHealthResult): Promise<boolean> {
    const configPath = this.expandPath(this.mcpConfig.configPath);

    if (!existsSync(configPath)) {
      console.error(`MCP config not found at ${configPath}`);
      return false;
    }

    try {
      const configContent = await readFile(configPath, 'utf-8');
      const config = JSON.parse(configContent);

      // Find the server in config
      const servers = config.mcpServers || config.servers || [];
      const serverConfig = servers.find((s: any) => s.name === server.name);

      if (!serverConfig) {
        console.error(`Server ${server.name} not found in config`);
        return false;
      }

      // Use Claude to analyze and suggest fixes
      let fixed = false;

      for await (const message of query({
        prompt: `Analysiere diese MCP-Server-Konfiguration und schlage Korrekturen vor:

Server-Name: ${server.name}
Fehler: ${server.lastError}
Aktuelle Konfiguration:
${JSON.stringify(serverConfig, null, 2)}

Falls Korrekturen möglich sind, beschreibe sie. Falls nicht, sage "KEINE_KORREKTUR".`,
        options: {
          allowedTools: [],
          maxTurns: 1
        }
      })) {
        if ('result' in message && typeof message.result === 'string') {
          if (!message.result.includes('KEINE_KORREKTUR')) {
            console.log(`Config suggestion for ${server.name}:`, message.result);
            // Note: Actual config modification would need careful implementation
            // For safety, we just log the suggestion here
          }
        }
      }

      return fixed;
    } catch (error) {
      console.error(`Config check failed for ${server.name}:`, error);
      return false;
    }
  }

  async batchCorrect(servers: ServerHealthResult[]): Promise<AutoCorrectResult[]> {
    const results: AutoCorrectResult[] = [];

    for (const server of servers) {
      let retries = 0;
      let result: AutoCorrectResult;

      do {
        result = await this.attemptCorrection(server);
        retries++;

        if (!result.success && retries < this.config.maxRetries) {
          await this.sleep(this.config.retryDelayMs);
        }
      } while (!result.success && retries < this.config.maxRetries);

      results.push(result);
    }

    return results;
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
