/**
 * Auto-Update module for code modifications with Git integration
 */

import { query } from '@anthropic-ai/claude-agent-sdk';
import { homedir } from 'os';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { GitOps } from './git-ops.js';
import type {
  AutoUpdateConfig,
  AutoUpdateResult,
  ServerHealthResult,
  MCPProxyConfig
} from './types.js';

const __dirname = dirname(fileURLToPath(import.meta.url));

export class AutoUpdateAgent {
  private config: AutoUpdateConfig;
  private mcpConfig: MCPProxyConfig;
  private gitOps: GitOps;
  private workingDir: string;
  private backupDir: string;

  constructor(
    config: AutoUpdateConfig,
    mcpConfig: MCPProxyConfig,
    workingDir?: string
  ) {
    this.config = config;
    this.mcpConfig = mcpConfig;
    this.workingDir = workingDir || join(__dirname, '..');
    this.backupDir = join(this.workingDir, 'backups');
    this.gitOps = new GitOps(this.workingDir);
  }

  private expandPath(path: string): string {
    if (path.startsWith('~')) {
      return path.replace('~', homedir());
    }
    return path;
  }

  async performUpdate(
    server: ServerHealthResult,
    updateDescription: string
  ): Promise<AutoUpdateResult> {
    if (!this.config.enabled) {
      return {
        success: false,
        filesModified: [],
        error: 'Auto-update is disabled'
      };
    }

    const result: AutoUpdateResult = {
      success: false,
      filesModified: []
    };

    const mcpConfigPath = this.expandPath(this.mcpConfig.configPath);

    try {
      // Ensure we're in a git repo
      if (!(await this.gitOps.isGitRepo())) {
        await this.gitOps.initRepo();
      }

      // Step 1: Commit BEFORE changes (with config backup)
      if (this.config.gitCommitBeforeChange) {
        console.log('Creating pre-update commit...');
        result.commitBefore = await this.gitOps.commitWithConfigBackup(
          `backup: vor auto-update für ${server.name}`,
          mcpConfigPath,
          this.backupDir
        );

        if (!result.commitBefore.success) {
          console.warn('Pre-update commit failed:', result.commitBefore.error);
        }
      }

      // Step 2: Execute the update using Claude Agent SDK
      console.log(`Executing auto-update for ${server.name}...`);
      const updateResult = await this.executeUpdate(server, updateDescription);

      result.filesModified = updateResult.filesModified;
      result.success = updateResult.success;

      // Step 3: Commit AFTER changes (with config backup)
      if (this.config.gitCommitAfterChange && result.success) {
        console.log('Creating post-update commit...');
        result.commitAfter = await this.gitOps.commitWithConfigBackup(
          `fix: auto-update für ${server.name} - ${updateDescription}`,
          mcpConfigPath,
          this.backupDir
        );

        if (!result.commitAfter.success) {
          console.warn('Post-update commit failed:', result.commitAfter.error);
        }
      }

      // If update failed, try to revert
      if (!result.success && result.commitBefore?.success) {
        console.log('Update failed, attempting to revert...');
        const reverted = await this.gitOps.revertLastCommit();
        if (!reverted) {
          result.error = 'Update failed and revert also failed';
        }
      }

    } catch (error) {
      result.error = error instanceof Error ? error.message : String(error);
    }

    return result;
  }

  private async executeUpdate(
    server: ServerHealthResult,
    updateDescription: string
  ): Promise<{ success: boolean; filesModified: string[] }> {
    const filesModified: string[] = [];
    let success = false;

    try {
      for await (const message of query({
        prompt: `Führe folgende Auto-Update Aktion durch:

Server: ${server.name}
Status: ${server.status}
Fehler: ${server.lastError}
Beschreibung der Änderung: ${updateDescription}

MCP-Config-Pfad: ${this.expandPath(this.mcpConfig.configPath)}

Analysiere das Problem und führe die notwendigen Code-Änderungen durch.
Dokumentiere welche Dateien geändert wurden.`,
        options: {
          allowedTools: ['Read', 'Edit', 'Write', 'Bash', 'Glob', 'Grep'],
          maxTurns: 10,
          cwd: this.workingDir,
          systemPrompt: `Du bist ein erfahrener DevOps-Ingenieur der MCP-Server-Konfigurationen repariert.
Sei vorsichtig bei Änderungen und dokumentiere alles.
Nach jeder Dateiänderung, füge den Pfad zur Liste der geänderten Dateien hinzu.`
        }
      })) {
        // Track tool use for file modifications
        if ('tool_use' in message) {
          const toolUse = message.tool_use as any;
          if (['Edit', 'Write'].includes(toolUse.name)) {
            const filePath = toolUse.input?.file_path || toolUse.input?.path;
            if (filePath && !filesModified.includes(filePath)) {
              filesModified.push(filePath);
            }
          }
        }

        if ('result' in message) {
          console.log('Update result:', message.result);
          success = true;
        }
      }
    } catch (error) {
      console.error('Update execution failed:', error);
    }

    return { success, filesModified };
  }

  async batchUpdate(
    servers: ServerHealthResult[],
    updateDescriptions: Map<string, string>
  ): Promise<Map<string, AutoUpdateResult>> {
    const results = new Map<string, AutoUpdateResult>();

    for (const server of servers) {
      const description = updateDescriptions.get(server.name) || 'Automatische Fehlerbehebung';
      const result = await this.performUpdate(server, description);
      results.set(server.name, result);
    }

    return results;
  }

  async getUpdateStatus(): Promise<{
    enabled: boolean;
    gitRepoInitialized: boolean;
    hasUncommittedChanges: boolean;
    currentBranch: string | null;
    lastCommit: string | null;
  }> {
    return {
      enabled: this.config.enabled,
      gitRepoInitialized: await this.gitOps.isGitRepo(),
      hasUncommittedChanges: await this.gitOps.hasUncommittedChanges(),
      currentBranch: await this.gitOps.getCurrentBranch(),
      lastCommit: await this.gitOps.getLastCommitHash()
    };
  }
}
