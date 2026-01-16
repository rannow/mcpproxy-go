#!/usr/bin/env node
/**
 * MCP Server Monitor - Entry Point
 *
 * A service using Claude Agent SDK that:
 * - Regularly checks MCP server status
 * - Creates diagnostic logs for disconnected/disabled servers
 * - Auto-correct: Attempts to fix configuration issues
 * - Auto-update: Can modify app code with git commits before/after changes
 */

import { readFile } from 'fs/promises';
import { existsSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { homedir } from 'os';
import { MCPMonitor } from './monitor.js';
import type { MonitorConfig } from './types.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const BASE_DIR = join(__dirname, '..');

// Parse CLI arguments
const args = process.argv.slice(2);
const checkOnce = args.includes('--check-once');
const diagnoseOnly = args.includes('--diagnose');
const showHelp = args.includes('--help') || args.includes('-h');

function printHelp(): void {
  console.log(`
MCP Server Monitor - Health Check & Auto-Correction Service

Usage:
  npm start              Start continuous monitoring
  npm run check          Run single health check
  npm run diagnose       Run diagnostics only

Options:
  --check-once    Run a single health check and exit
  --diagnose      Run diagnostics only and exit
  --help, -h      Show this help message

Configuration:
  Edit config/monitor.config.json to configure:

  autoCheck.enabled      Enable/disable automatic health checks
  autoCheck.intervalMs   Interval between checks (default: 60000ms)

  autoCorrect.enabled    Enable/disable automatic error correction
  autoCorrect.maxRetries Number of retry attempts

  autoUpdate.enabled              Enable/disable code modifications
  autoUpdate.gitCommitBeforeChange  Commit before changes
  autoUpdate.gitCommitAfterChange   Commit after changes
  autoUpdate.backupConfig           Backup config in commits

Log output:
  logs/diagnostic.log    Diagnostic entries for unhealthy servers
`);
}

async function loadConfig(): Promise<MonitorConfig> {
  const configPath = join(BASE_DIR, 'config', 'monitor.config.json');

  if (!existsSync(configPath)) {
    throw new Error(`Config file not found: ${configPath}`);
  }

  const content = await readFile(configPath, 'utf-8');
  const config = JSON.parse(content) as MonitorConfig;

  // Expand paths
  config.mcpProxy.configPath = expandPath(config.mcpProxy.configPath);
  config.diagnostics.logPath = config.diagnostics.logPath.startsWith('./')
    ? join(BASE_DIR, config.diagnostics.logPath.slice(2))
    : config.diagnostics.logPath;

  return config;
}

function expandPath(path: string): string {
  if (path.startsWith('~')) {
    return path.replace('~', homedir());
  }
  return path;
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function main(): Promise<void> {
  if (showHelp) {
    printHelp();
    process.exit(0);
  }

  console.log('MCP Server Monitor starting...');
  console.log(`Base directory: ${BASE_DIR}`);

  try {
    const config = await loadConfig();
    console.log('Configuration loaded');

    const monitor = new MCPMonitor(config, BASE_DIR);

    // Handle different modes
    if (diagnoseOnly) {
      console.log('\n=== Running Diagnostics ===\n');
      await monitor.runDiagnostics();
      process.exit(0);
    }

    if (checkOnce) {
      console.log('\n=== Running Single Health Check ===\n');
      const results = await monitor.checkHealth();

      console.log(`Total servers: ${results.total}`);
      console.log(`Healthy: ${results.healthyCount}`);
      console.log(`Unhealthy: ${results.unhealthyCount}`);

      if (results.unhealthyServers.length > 0) {
        console.log('\nUnhealthy servers:');
        for (const server of results.unhealthyServers) {
          console.log(`  - ${server.name}: ${server.status}`);
          if (server.lastError) {
            console.log(`    Error: ${server.lastError}`);
          }
        }
        await monitor.writeDiagnosticLog(results);
        console.log(`\nDiagnostic log written to: ${config.diagnostics.logPath}`);
      }

      process.exit(results.unhealthyServers.length > 0 ? 1 : 0);
    }

    // Continuous monitoring mode
    if (!config.autoCheck.enabled) {
      console.warn('\nAuto-check is disabled in config.');
      console.log('Enable with: autoCheck.enabled: true');
      console.log('\nYou can still run manual checks:');
      console.log('  npm run check     - Single health check');
      console.log('  npm run diagnose  - Full diagnostics');
      process.exit(0);
    }

    console.log(`\nStarting continuous monitoring (interval: ${config.autoCheck.intervalMs}ms)`);
    console.log(`Auto-correct: ${config.autoCorrect.enabled ? 'enabled' : 'disabled'}`);
    console.log(`Auto-update: ${config.autoUpdate.enabled ? 'enabled' : 'disabled'}`);
    console.log('\nPress Ctrl+C to stop\n');

    // Initial delay before first check
    if (config.autoCheck.startupDelayMs > 0) {
      console.log(`Waiting ${config.autoCheck.startupDelayMs}ms before first check...`);
      await sleep(config.autoCheck.startupDelayMs);
    }

    // Start monitoring loop
    await monitor.startMonitoring();

  } catch (error) {
    console.error('Fatal error:', error instanceof Error ? error.message : error);
    process.exit(1);
  }
}

// Handle graceful shutdown
process.on('SIGINT', () => {
  console.log('\nReceived SIGINT, shutting down...');
  process.exit(0);
});

process.on('SIGTERM', () => {
  console.log('\nReceived SIGTERM, shutting down...');
  process.exit(0);
});

// Run
main();
