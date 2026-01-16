/**
 * Git Operations for Auto-Update functionality
 */

import { exec } from 'child_process';
import { promisify } from 'util';
import { writeFile, readFile, mkdir } from 'fs/promises';
import { existsSync } from 'fs';
import { dirname, join } from 'path';
import type { GitCommitResult } from './types.js';

const execAsync = promisify(exec);

export class GitOps {
  private workingDir: string;

  constructor(workingDir: string) {
    this.workingDir = workingDir;
  }

  async isGitRepo(): Promise<boolean> {
    try {
      await execAsync('git rev-parse --git-dir', { cwd: this.workingDir });
      return true;
    } catch {
      return false;
    }
  }

  async initRepo(): Promise<boolean> {
    try {
      await execAsync('git init', { cwd: this.workingDir });
      return true;
    } catch {
      return false;
    }
  }

  async getStatus(): Promise<string> {
    try {
      const { stdout } = await execAsync('git status --porcelain', { cwd: this.workingDir });
      return stdout.trim();
    } catch (error) {
      return '';
    }
  }

  async hasUncommittedChanges(): Promise<boolean> {
    const status = await this.getStatus();
    return status.length > 0;
  }

  async createConfigBackup(configPath: string, backupDir: string): Promise<string | null> {
    try {
      if (!existsSync(configPath)) {
        return null;
      }

      const content = await readFile(configPath, 'utf-8');
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
      const backupPath = join(backupDir, `mcp_config_backup_${timestamp}.json`);

      if (!existsSync(backupDir)) {
        await mkdir(backupDir, { recursive: true });
      }

      await writeFile(backupPath, content, 'utf-8');
      return backupPath;
    } catch (error) {
      console.error('Failed to create config backup:', error);
      return null;
    }
  }

  async commit(message: string, files?: string[]): Promise<GitCommitResult> {
    try {
      // Stage files
      if (files && files.length > 0) {
        for (const file of files) {
          await execAsync(`git add "${file}"`, { cwd: this.workingDir });
        }
      } else {
        await execAsync('git add -A', { cwd: this.workingDir });
      }

      // Check if there's anything to commit
      const status = await this.getStatus();
      if (!status) {
        return { success: true, commitHash: 'no-changes' };
      }

      // Commit with message
      const { stdout } = await execAsync(
        `git commit -m "${message.replace(/"/g, '\\"')}"`,
        { cwd: this.workingDir }
      );

      // Extract commit hash
      const hashMatch = stdout.match(/\[[\w-]+ ([a-f0-9]+)\]/);
      const commitHash = hashMatch ? hashMatch[1] : undefined;

      return { success: true, commitHash };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : String(error)
      };
    }
  }

  async commitWithConfigBackup(
    message: string,
    mcpConfigPath: string,
    backupDir: string
  ): Promise<GitCommitResult> {
    // Create backup first
    const backupPath = await this.createConfigBackup(mcpConfigPath, backupDir);

    // Add backup to commit if created
    const files = backupPath ? [backupPath] : undefined;

    return this.commit(message, files);
  }

  async createBranch(branchName: string): Promise<boolean> {
    try {
      await execAsync(`git checkout -b ${branchName}`, { cwd: this.workingDir });
      return true;
    } catch {
      return false;
    }
  }

  async getCurrentBranch(): Promise<string | null> {
    try {
      const { stdout } = await execAsync('git branch --show-current', { cwd: this.workingDir });
      return stdout.trim() || null;
    } catch {
      return null;
    }
  }

  async getLastCommitHash(): Promise<string | null> {
    try {
      const { stdout } = await execAsync('git rev-parse HEAD', { cwd: this.workingDir });
      return stdout.trim() || null;
    } catch {
      return null;
    }
  }

  async revertLastCommit(): Promise<boolean> {
    try {
      await execAsync('git revert HEAD --no-edit', { cwd: this.workingDir });
      return true;
    } catch {
      return false;
    }
  }
}
