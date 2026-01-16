/**
 * Main Monitor class - orchestrates health checks, diagnostics, and corrections
 */

import { query } from '@anthropic-ai/claude-agent-sdk';
import type {
  MonitorConfig,
  HealthCheckResult,
  ServerHealthResult,
  DiagnosticEntry
} from './types.js';
import { DiagnosticsAgent } from './diagnostics.js';
import { AutoCorrectAgent } from './auto-correct.js';
import { AutoUpdateAgent } from './auto-update.js';

export class MCPMonitor {
  private config: MonitorConfig;
  private baseDir: string;
  private diagnostics: DiagnosticsAgent;
  private autoCorrect: AutoCorrectAgent;
  private autoUpdate: AutoUpdateAgent;
  private intervalId: NodeJS.Timeout | null = null;
  private isRunning = false;

  constructor(config: MonitorConfig, baseDir: string) {
    this.config = config;
    this.baseDir = baseDir;
    this.diagnostics = new DiagnosticsAgent(config.diagnostics, baseDir);
    this.autoCorrect = new AutoCorrectAgent(config.autoCorrect, config.mcpProxy);
    this.autoUpdate = new AutoUpdateAgent(config.autoUpdate, config.mcpProxy, baseDir);
  }

  async checkHealth(): Promise<HealthCheckResult> {
    const result: HealthCheckResult = {
      timestamp: new Date(),
      total: 0,
      healthyCount: 0,
      unhealthyCount: 0,
      servers: [],
      unhealthyServers: []
    };

    try {
      // Use Claude Agent SDK to call the MCP proxy health check
      for await (const message of query({
        prompt: `Führe einen Health-Check für alle MCP-Server durch.
Rufe das mcpproxy Tool 'upstream_servers' mit operation 'list' auf um alle Server zu sehen.
Dann analysiere welche Server 'disconnected', 'auto_disabled' oder andere Probleme haben.

Gib das Ergebnis als JSON zurück mit diesem Format:
{
  "servers": [
    {"name": "server-name", "status": "connected|disconnected|disabled|auto_disabled", "healthy": true|false, "lastError": "optional error", "toolCount": 0}
  ]
}`,
        options: {
          allowedTools: ['Bash'],
          maxTurns: 5,
          systemPrompt: 'Du bist ein MCP-Server-Monitor. Nutze curl oder mcpproxy CLI um Server-Status abzufragen.'
        }
      })) {
        if ('result' in message && typeof message.result === 'string') {
          try {
            // Try to parse JSON from result
            const jsonMatch = message.result.match(/\{[\s\S]*\}/);
            if (jsonMatch) {
              const parsed = JSON.parse(jsonMatch[0]);
              if (parsed.servers && Array.isArray(parsed.servers)) {
                result.servers = parsed.servers;
              }
            }
          } catch {
            // If parsing fails, try alternative approach
            console.warn('Could not parse health check result as JSON');
          }
        }
      }

      // Calculate statistics
      result.total = result.servers.length;
      result.healthyCount = result.servers.filter(s => s.healthy).length;
      result.unhealthyCount = result.total - result.healthyCount;
      result.unhealthyServers = result.servers.filter(s => !s.healthy);

    } catch (error) {
      console.error('Health check failed:', error);
      // Try fallback method using direct HTTP
      await this.fallbackHealthCheck(result);
    }

    return result;
  }

  private async fallbackHealthCheck(result: HealthCheckResult): Promise<void> {
    try {
      // Direct HTTP call to mcpproxy
      const response = await fetch(`${this.config.mcpProxy.endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 1,
          method: 'tools/call',
          params: {
            name: 'mcpproxy_health_check',
            arguments: {}
          }
        })
      });

      if (response.ok) {
        const data = await response.json() as any;
        if (data.result?.content?.[0]?.text) {
          const healthData = JSON.parse(data.result.content[0].text);
          result.total = healthData.total || 0;
          result.healthyCount = healthData.healthy_count || 0;
          result.unhealthyCount = healthData.unhealthy_count || 0;

          if (healthData.servers) {
            result.servers = Object.entries(healthData.servers).map(([name, info]: [string, any]) => ({
              name,
              status: info.status || 'unknown',
              healthy: info.healthy || false,
              lastError: info.last_error,
              toolCount: info.tool_count
            }));
            result.unhealthyServers = result.servers.filter(s => !s.healthy);
          }
        }
      }
    } catch (error) {
      console.error('Fallback health check failed:', error);
    }
  }

  async runDiagnostics(): Promise<DiagnosticEntry[]> {
    console.log('Running diagnostics...');
    const healthResult = await this.checkHealth();
    const entries: DiagnosticEntry[] = [];

    for (const server of healthResult.unhealthyServers) {
      const entry = await this.diagnostics.analyzeProblem(server);
      entries.push(entry);
      console.log(`\n${server.name}: ${entry.diagnosis}`);
      console.log(`  Empfehlung: ${entry.recommendation}`);
    }

    if (entries.length === 0) {
      console.log('Alle Server sind gesund!');
    } else {
      await this.diagnostics.writeLog(entries);
      console.log(`\nDiagnose-Log geschrieben: ${this.config.diagnostics.logPath}`);
    }

    return entries;
  }

  async writeDiagnosticLog(healthResult: HealthCheckResult): Promise<void> {
    const entries: DiagnosticEntry[] = [];

    for (const server of healthResult.unhealthyServers) {
      const entry = await this.diagnostics.analyzeProblem(server);
      entries.push(entry);
    }

    await this.diagnostics.writeLog(entries);
  }

  async startMonitoring(): Promise<void> {
    if (this.isRunning) {
      console.warn('Monitor is already running');
      return;
    }

    this.isRunning = true;
    console.log('Starting continuous monitoring...');

    // Run initial check
    await this.runMonitoringCycle();

    // Set up interval
    this.intervalId = setInterval(
      () => this.runMonitoringCycle(),
      this.config.autoCheck.intervalMs
    );

    // Keep process running
    await new Promise<void>((resolve) => {
      process.on('SIGINT', () => {
        this.stop();
        resolve();
      });
    });
  }

  private async runMonitoringCycle(): Promise<void> {
    const timestamp = new Date().toISOString();
    console.log(`\n[${timestamp}] Running health check...`);

    try {
      const healthResult = await this.checkHealth();

      console.log(`Total: ${healthResult.total}, Healthy: ${healthResult.healthyCount}, Unhealthy: ${healthResult.unhealthyCount}`);

      if (healthResult.unhealthyServers.length > 0) {
        console.log('Unhealthy servers:');
        for (const server of healthResult.unhealthyServers) {
          console.log(`  - ${server.name}: ${server.status} (${server.lastError || 'no error details'})`);
        }

        // Write diagnostic log
        await this.writeDiagnosticLog(healthResult);

        // Attempt auto-correction if enabled
        if (this.config.autoCorrect.enabled) {
          console.log('Attempting auto-correction...');
          const corrections = await this.autoCorrect.batchCorrect(healthResult.unhealthyServers);

          for (const correction of corrections) {
            if (correction.success) {
              console.log(`  ✓ ${correction.serverName}: ${correction.action}`);
            } else {
              console.log(`  ✗ ${correction.serverName}: ${correction.error}`);
            }
          }
        }

        // Attempt auto-update if enabled
        if (this.config.autoUpdate.enabled) {
          console.log('Auto-update is enabled - checking for code fixes...');
          // Auto-update would be triggered here for specific scenarios
        }
      }
    } catch (error) {
      console.error('Monitoring cycle failed:', error);
    }
  }

  async stop(): Promise<void> {
    console.log('Stopping monitor...');
    this.isRunning = false;

    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
  }

  getConfig(): MonitorConfig {
    return this.config;
  }

  isMonitoring(): boolean {
    return this.isRunning;
  }
}
