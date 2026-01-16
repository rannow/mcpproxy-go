/**
 * Diagnostics module using Claude Agent SDK for intelligent analysis
 */

import { query } from '@anthropic-ai/claude-agent-sdk';
import { appendFile, mkdir, stat, rename, unlink, readFile } from 'fs/promises';
import { existsSync } from 'fs';
import { dirname, join } from 'path';
import type {
  ServerHealthResult,
  DiagnosticEntry,
  DiagnosticsConfig,
  HealthCheckResult
} from './types.js';

export class DiagnosticsAgent {
  private config: DiagnosticsConfig;
  private logPath: string;

  constructor(config: DiagnosticsConfig, baseDir: string) {
    this.config = config;
    this.logPath = config.logPath.startsWith('./')
      ? join(baseDir, config.logPath.slice(2))
      : config.logPath;
  }

  async analyzeProblem(server: ServerHealthResult): Promise<DiagnosticEntry> {
    const entry: DiagnosticEntry = {
      timestamp: new Date(),
      serverName: server.name,
      status: server.status,
      error: server.lastError,
      diagnosis: '',
      recommendation: ''
    };

    // Use Claude Agent SDK for intelligent diagnosis
    try {
      let result = '';

      for await (const message of query({
        prompt: `Analysiere diesen MCP-Server-Fehler und gib eine kurze Diagnose und Empfehlung:

Server: ${server.name}
Status: ${server.status}
Fehler: ${server.lastError || 'Unbekannt'}
Startup-Mode: ${server.startupMode || 'Unbekannt'}

Antworte im Format:
DIAGNOSE: [Eine Zeile Diagnose]
EMPFEHLUNG: [Eine Zeile Empfehlung zur Behebung]`,
        options: {
          allowedTools: [],
          maxTurns: 1,
          systemPrompt: 'Du bist ein MCP-Server-Diagnose-Experte. Antworte kurz und präzise auf Deutsch.'
        }
      })) {
        if ('result' in message && typeof message.result === 'string') {
          result = message.result;
        }
      }

      // Parse response
      const diagMatch = result.match(/DIAGNOSE:\s*(.+)/i);
      const recMatch = result.match(/EMPFEHLUNG:\s*(.+)/i);

      entry.diagnosis = diagMatch?.[1]?.trim() || this.getFallbackDiagnosis(server);
      entry.recommendation = recMatch?.[1]?.trim() || this.getFallbackRecommendation(server);
    } catch (error) {
      // Fallback to rule-based diagnosis if SDK fails
      entry.diagnosis = this.getFallbackDiagnosis(server);
      entry.recommendation = this.getFallbackRecommendation(server);
    }

    return entry;
  }

  private getFallbackDiagnosis(server: ServerHealthResult): string {
    const error = server.lastError?.toLowerCase() || '';

    if (error.includes('connection refused')) {
      return 'Server-Prozess läuft nicht oder Port ist nicht erreichbar';
    }
    if (error.includes('timeout')) {
      return 'Server antwortet zu langsam oder ist überlastet';
    }
    if (error.includes('not found') || error.includes('missing')) {
      return 'Konfiguration unvollständig oder Abhängigkeit fehlt';
    }
    if (error.includes('permission') || error.includes('access denied')) {
      return 'Berechtigungsproblem beim Zugriff auf Ressourcen';
    }
    if (server.status === 'auto_disabled') {
      return `Server wurde nach wiederholten Fehlern automatisch deaktiviert`;
    }
    if (server.status === 'quarantined') {
      return 'Server wurde aus Sicherheitsgründen isoliert';
    }

    return 'Verbindungsfehler - Server nicht erreichbar';
  }

  private getFallbackRecommendation(server: ServerHealthResult): string {
    const error = server.lastError?.toLowerCase() || '';

    if (error.includes('connection refused')) {
      return 'Starte den Server-Prozess oder prüfe die Port-Konfiguration';
    }
    if (error.includes('timeout')) {
      return 'Erhöhe Timeout-Werte oder prüfe Server-Last';
    }
    if (error.includes('not found') || error.includes('missing')) {
      return 'Prüfe mcp_config.json auf fehlende Einträge';
    }
    if (error.includes('permission')) {
      return 'Prüfe Dateiberechtigungen und Benutzerrechte';
    }
    if (server.status === 'auto_disabled') {
      return 'Behebe den Fehler und aktiviere den Server manuell neu';
    }
    if (server.status === 'quarantined') {
      return 'Prüfe Server auf Sicherheitsprobleme vor Reaktivierung';
    }

    return 'Prüfe Server-Logs und Konfiguration';
  }

  async writeLog(entries: DiagnosticEntry[]): Promise<void> {
    if (entries.length === 0) return;

    const logDir = dirname(this.logPath);
    if (!existsSync(logDir)) {
      await mkdir(logDir, { recursive: true });
    }

    const timestamp = new Date().toISOString();
    let logContent = `\n[${timestamp}] === MCP Server Diagnose ===\n`;

    for (const entry of entries) {
      logContent += `
Server: ${entry.serverName}
Status: ${entry.status.toUpperCase()}
${entry.error ? `Fehler: ${entry.error}` : ''}
Diagnose: ${entry.diagnosis}
Empfehlung: ${entry.recommendation}
${entry.autoCorrectAttempted ? `Auto-Correct: ${entry.autoCorrectSuccess ? 'Erfolgreich' : 'Fehlgeschlagen'}` : ''}
---
`;
    }

    await appendFile(this.logPath, logContent);
    await this.checkRotation();
  }

  private async checkRotation(): Promise<void> {
    try {
      if (!existsSync(this.logPath)) return;

      const stats = await stat(this.logPath);
      const sizeMb = stats.size / (1024 * 1024);

      if (sizeMb >= this.config.maxLogSizeMb) {
        await this.rotate();
      }
    } catch {
      // Ignore rotation errors
    }
  }

  private async rotate(): Promise<void> {
    const oldest = `${this.logPath}.${this.config.rotateCount}`;
    if (existsSync(oldest)) {
      await unlink(oldest);
    }

    for (let i = this.config.rotateCount - 1; i >= 1; i--) {
      const from = `${this.logPath}.${i}`;
      const to = `${this.logPath}.${i + 1}`;
      if (existsSync(from)) {
        await rename(from, to);
      }
    }

    if (existsSync(this.logPath)) {
      await rename(this.logPath, `${this.logPath}.1`);
    }
  }

  async generateReport(healthResult: HealthCheckResult): Promise<string> {
    const unhealthy = healthResult.unhealthyServers;

    if (unhealthy.length === 0) {
      return `Alle ${healthResult.total} Server sind gesund.`;
    }

    let report = `MCP Server Status Report
========================
Zeitpunkt: ${healthResult.timestamp.toISOString()}
Gesamt: ${healthResult.total} Server
Gesund: ${healthResult.healthyCount}
Problematisch: ${healthResult.unhealthyCount}

Problematische Server:
`;

    for (const server of unhealthy) {
      const diagnosis = await this.analyzeProblem(server);
      report += `
- ${server.name}
  Status: ${server.status}
  Fehler: ${server.lastError || 'Unbekannt'}
  Diagnose: ${diagnosis.diagnosis}
  Empfehlung: ${diagnosis.recommendation}
`;
    }

    return report;
  }
}
