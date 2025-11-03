# Failed MCP Servers - Detailed Diagnostic Report

**Generated**: 2025-10-31 16:43:47
**Total Failed Servers**: 99
**Analysis Method**: Individual server testing with log analysis

---

## Executive Summary

This report provides detailed diagnostics for each MCP server that failed to connect.

### Category Breakdown

- **⏱️ Timeout/Slow**: 80 servers (80 quick-fixable)
- **📦 Package Issue**: 18 servers (18 quick-fixable)
- **🔧 Unknown Error**: 1 servers

---

## ⏱️ Timeout/Slow (80 servers)

### MCP-Analyzer

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `/Users/hrannow/mcp-analyzer-wrapper.sh --key {"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"claude-ai","version":"0.1.0"}},"jsonrpc":"2.0","id":0} --config "{\"page\":1,\"lines\":100,\"filter\":\"\",\"fileLimit\":5,\"customPath\":\"\"}"`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # Manual installation required for /Users/hrannow/mcp-analyzer-wrapper.sh
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### MCP_DOCKER

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `docker mcp gateway run --verbose`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # Manual installation required for docker
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### airtable-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y airtable-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### auto-mcp

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y auto-mcp`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### aws-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `docker run -i --rm -v /Users/hrannow/.aws:/home/appuser/.aws:ro ghcr.io/alexei-led/aws-mcp-server:latest`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # Manual installation required for docker
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### bigquery-ergut

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @ergut/mcp-bigquery-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### browserless-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y browserless-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### crawl4ai-rag

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `docker run --rm -i -e TRANSPORT -e OPENAI_API_KEY -e SUPABASE_URL -e SUPABASE_SERVICE_KEY mcp/crawl4ai`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # Manual installation required for docker
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### crawl4ai-rag

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `docker run --rm -i -e TRANSPORT -e OPENAI_API_KEY -e SUPABASE_URL -e SUPABASE_SERVICE_KEY mcp/crawl4ai`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # Manual installation required for docker
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### dbhub-universal

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @bytebase/dbhub`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### docs-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y docs-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### documents-vector-search

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y documents-vector-search`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### elasticsearch-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y elasticsearch-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### email-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y email-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### gdrive

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @modelcontextprotocol/server-gdrive`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### gitlab

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y gitlab-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### grafana-extern

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y grafana-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### infinity-swiss

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `node /Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/MCP-infinity/dist/server.js`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # Manual installation required for node
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### k8s-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `docker run -i --rm -v /Users/hrannow/.kube:/home/appuser/.kube:ro -e K8S_CONTEXT=my-cluster -e K8S_NAMESPACE=my-namespace ghcr.io/alexei-led/k8s-mcp-server:latest`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # Manual installation required for docker
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### markdownify-mcp

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y markdownify-mcp`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### maven-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y maven-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-anthropic-claude

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-anthropic-claude`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-browser-tools

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-browserbase`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-cloudflare

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-cloudflare`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-code-executor

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `node /Users/hrannow/OneDrive/workspace/mcp-server/mcp_code_executor/build/index.js`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # Manual installation required for node
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-communicator-telegram

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-communicator-telegram`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-computer-use

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-computer-use`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-datetime

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-time`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-docker

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-docker`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-docker-compose

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-docker-compose`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-k8s

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-k8s`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-linear

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-linear`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-linkedin

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-linkedin`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-openai

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-openai`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-pandoc

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-pandoc`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-perplexity

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-perplexity`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-pinecone

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-pinecone`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-ragdocs

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-ragdocs`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-reddit

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-reddit`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-s3

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-s3`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-apache-airflow

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-apache-airflow`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-browserbase

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-browserbase`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-buildkite

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-buildkite`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-git

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @modelcontextprotocol/server-git`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-kibana

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-kibana`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-notion

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @modelcontextprotocol/server-notion`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-odoo

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-odoo`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-raygun

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-raygun`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-redis

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @modelcontextprotocol/server-redis`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-shell

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @modelcontextprotocol/server-shell`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-todoist

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-todoist`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-twitter

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-twitter`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-server-youtube-transcript

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-server-youtube-transcript`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-servers-kagi

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-servers-kagi`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-teams-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-teams-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-webresearch

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-webresearch`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-webscraper

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-webscraper`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp-xmind

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp-xmind`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp_code_analyzer

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mcp_code_analyzer`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp_server_mysql_preprod

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @benborla29/mcp-server-mysql`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mcp_server_mysql_staging

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @benborla29/mcp-server-mysql`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mem0-mcp

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mem0-mcp`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### memory-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y memory-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### mindmap-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y mindmap-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### n8n-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y n8n-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### neon-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y neon-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### obsidian

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @mcp-obsidian/server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### obsidian-mcp-tools

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/Obsidian/My Obsidian Vault/.obsidian/plugins/mcp-tools/bin/mcp-server None`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   # No args specified
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### opencode

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y opencode-mcp-tool -- --model google/gemini-2.5-pro`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### ragie-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y ragie-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### search-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y search-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### spotify-mcp

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y spotify-mcp`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### test-weather-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx @modelcontextprotocol/server-weather`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g @modelcontextprotocol/server-weather
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### time-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx --yes @modelcontextprotocol/server-time`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g --yes
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### travel-planner-mcp-server

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y travel-planner-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### video-editing-mcp

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y video-editing-mcp`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### whatsapp-mcp-lharries

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y whatsapp-mcp`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### youtube

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y youtube-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### youtube-mcp-server-zubeid

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx -y @zubeidhendricks/youtube-mcp-server`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g -y
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

### zapier-mcp

**Category**: ⏱️ Timeout/Slow
**Protocol**: `stdio`
**Command**: `npx mcp-remote https://actions.zapier.com/mcp/sk-ak-cOdRD4hkevP7gQZ8fvEWvL2pE6/sse`

**Issue**: Server takes >60s to initialize (NPX download or slow startup)

**Fix**:
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   npm install -g mcp-remote
   ```


**Quick Fix**: ✅ Can be fixed automatically

---

## 📦 Package Issue (18 servers)

### Container User

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `container-use stdio`

**Issue**: Command `container-use` not found in PATH

**Fix**:Install `container-use` command

**Quick Fix**: ✅ Can be fixed automatically

---

### awslabs.amazon-rekognition-mcp-server

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx awslabs.amazon-rekognition-mcp-server@latest`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### awslabs.cloudwatch-logs-mcp-server

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx awslabs.cloudwatch-logs-mcp-server@latest`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### awslabs.cost-analysis-mcp-server

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx awslabs.cost-analysis-mcp-server@latest`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### awslabs.ecs-mcp-server

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx awslabs.ecs-mcp-server@latest`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### basic-memory

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx basic-memory mcp`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### bigquery-lucashild

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `pipx run mcp-server-bigquery`

**Issue**: Command `pipx` not found in PATH

**Fix**:
**Install pipx**:
```bash
pip install pipx
pipx ensurepath
```


**Quick Fix**: ✅ Can be fixed automatically

---

### calculator

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx mcp-server-calculator`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### cipher

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `cipher -mode mcp`

**Issue**: Command `cipher` not found in PATH

**Fix**:Install `cipher` command

**Quick Fix**: ✅ Can be fixed automatically

---

### cognee

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uv --directory /Users/hrannow/OneDrive/workspace/cognee/cognee/cognee-mcp run cognee`

**Issue**: Command `uv` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### docker-mcp

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx docker-mcp`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### duckdb-ktanaka

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `pipx run mcp-server-duckdb`

**Issue**: Command `pipx` not found in PATH

**Fix**:
**Install pipx**:
```bash
pip install pipx
pipx ensurepath
```


**Quick Fix**: ✅ Can be fixed automatically

---

### everything-search

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx mcp-server-everything-search`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### fetch

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx mcp-server-fetch`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### motherduck-duckdb

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `pipx run mcp-server-motherduck`

**Issue**: Command `pipx` not found in PATH

**Fix**:
**Install pipx**:
```bash
pip install pipx
pipx ensurepath
```


**Quick Fix**: ✅ Can be fixed automatically

---

### serena

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uvx --from git+https://github.com/oraios/serena serena start-mcp-server`

**Issue**: Command `uvx` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

### toolfront-database

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `pipx run toolfront`

**Issue**: Command `pipx` not found in PATH

**Fix**:
**Install pipx**:
```bash
pip install pipx
pipx ensurepath
```


**Quick Fix**: ✅ Can be fixed automatically

---

### whatsapp

**Category**: 📦 Package Issue
**Protocol**: `stdio`
**Command**: `uv --directory /Users/hrannow/OneDrive/workspace/mcp-server/whatsapp-mcp/whatsapp-mcp-server run main.py`

**Issue**: Command `uv` not found in PATH

**Fix**:
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```


**Quick Fix**: ✅ Can be fixed automatically

---

## 🔧 Unknown Error (1 servers)

### Framelink Figma MCP

**Category**: 🔧 Unknown Error
**Protocol**: `stdio`
**Command**: `npx -y figma-developer-mcp --figma-api-key=REDACTED --stdio`

**Issue**: See log file for details

**Fix**:Check `/Users/hrannow/Library/Logs/mcpproxy/server-Framelink Figma MCP.log` for error messages

---

