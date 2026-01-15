# MCPProxy - Übersicht aller MCP Server

**Erstellt:** 2025-11-28
**MCPProxy Version:** Latest
**Anzahl Server:** 26
**Anzahl Tools:** 500+ (geschätzt)

## Zusammenfassung

MCPProxy fungiert als zentrale Verbindungsschicht zu 26 verschiedenen MCP Servern, die insgesamt über 500 Tools bereitstellen. Die Server decken ein breites Spektrum an Funktionalitäten ab - von AWS-Cloud-Services über Kubernetes-Management bis hin zu Entwicklungstools und Monitoring-Lösungen.

---

## 🎯 Server-Kategorien

### 1. AWS & Cloud Services (9 Server)

#### **athena**
- **Zweck**: AWS Athena SQL-Abfragen auf CloudWatch Metrics
- **Hauptfunktionen**: SQL-basierte Metrik-Abfragen, CloudWatch Integration
- **Wichtige Tools**: `cloudwatch_metrics`

#### **aws-mcp-server**
- **Zweck**: Umfassende AWS CLI Integration mit Unix-Pipeline-Unterstützung
- **Hauptfunktionen**: AWS CLI Befehle mit Pipes, flexibles Command Execution
- **Wichtige Tools**: `aws_cli_pipeline`
- **Besonderheit**: Unterstützt Unix-Pipes für Datenverarbeitung

#### **awslabs.aws-diagram-mcp-server**
- **Zweck**: AWS Architektur-Diagramm-Generierung
- **Hauptfunktionen**: Visuelle AWS Infrastruktur-Darstellung
- **Wichtige Tools**: `generate_diagram`, `list_icons`
- **Besonderheit**: Python Diagrams Package Integration

#### **awslabs.aws-documentation-mcp-server**
- **Zweck**: AWS Dokumentationssuche und -zugriff
- **Hauptfunktionen**: Dokumentation durchsuchen, Seiten lesen, Empfehlungen
- **Wichtige Tools**:
  - `search_documentation` - AWS Docs durchsuchen
  - `read_documentation` - Markdown-konvertierte Inhalte
  - `recommend` - Verwandte Dokumentation finden

#### **awslabs.aws-serverless-mcp-server**
- **Zweck**: AWS Lambda & Serverless Application Model (SAM) Lifecycle
- **Hauptfunktionen**: SAM init/build/deploy, Lambda local invoke, Event schemas
- **Wichtige Tools**:
  - `sam_init` - Projekt initialisieren
  - `sam_build` - Application bauen
  - `sam_deploy` - Auf AWS deployen
  - `sam_local_invoke` - Lokal testen
  - `get_serverless_templates` - Beispiel-Templates
  - `deploy_webapp` - Web-Apps deployen
  - `get_lambda_event_schemas` - Event-Schemas abrufen
- **Besonderheit**: Kompletter SAM Development Lifecycle

#### **awslabs.bedrock-kb-retrieval-mcp-server**
- **Zweck**: Amazon Bedrock Knowledge Base Zugriff
- **Hauptfunktionen**: Knowledge Base Listen, Abfragen
- **Wichtige Tools**: `ListKnowledgeBases`, `QueryKnowledgeBases`

#### **awslabs.eks-mcp-server**
- **Zweck**: Amazon EKS (Elastic Kubernetes Service) Management
- **Hauptfunktionen**:
  - Kubernetes Ressourcen verwalten (CRUD)
  - Pod Logs und Events abrufen
  - CloudWatch Logs/Metrics für Container Insights
  - Kubernetes Manifests generieren
  - CloudFormation Stack Management
- **Wichtige Tools**:
  - `list_k8s_resources` - Ressourcen auflisten
  - `manage_k8s_resource` - CRUD Operations
  - `get_pod_logs` - Pod Logs
  - `get_k8s_events` - Kubernetes Events
  - `get_cloudwatch_logs` - Container Logs
  - `get_cloudwatch_metrics` - Metriken
  - `generate_app_manifest` - Deployment Manifests
  - `manage_eks_stacks` - CloudFormation Stacks
- **Besonderheit**: Umfassende EKS-Integration mit CloudWatch

#### **awslabs.iam-mcp-server**
- **Zweck**: AWS IAM (Identity and Access Management)
- **Hauptfunktionen**: User Management, Policy Verwaltung
- **Wichtige Tools**: `list_users`

#### **awslabs.terraform-mcp-server**
- **Zweck**: Terraform für AWS Infrastructure as Code
- **Hauptfunktionen**: AWS-IA Module suchen, Terraform Provider Info
- **Wichtige Tools**:
  - `SearchSpecificAwsIaModules` - AWS-IA Module
  - `get_provider_capabilities` - Provider Features

---

### 2. Container & Kubernetes (6 Server)

#### **docker-mcp**
- **Zweck**: Docker Container Management
- **Hauptfunktionen**: Container erstellen, listen, logs abrufen
- **Wichtige Tools**:
  - `list-containers`
  - `create-container`
  - `get-logs`
  - `deploy-compose`

#### **k8s-mcp-server**
- **Zweck**: Kubernetes Helm Operations
- **Hauptfunktionen**: Helm Befehle, Kubernetes Hilfe
- **Wichtige Tools**: `describe_helm`

#### **mcp-k8s-go**
- **Zweck**: Kubernetes Go Client
- **Hauptfunktionen**: Context Management, Resource Operationen, Pod Logs
- **Wichtige Tools**:
  - `list-k8s-contexts`
  - `get-k8s-resource`
  - `get-k8s-pod-logs`
  - `k8s-pod-exec`

#### **mcp-server-kubernetes**
- **Zweck**: Kubernetes API Ressourcen-Management
- **Hauptfunktionen**: kubectl Kommandos abstrahieren
- **Wichtige Tools**:
  - `kubectl_get`
  - `kubectl_create`
  - `kubectl_logs`
  - `exec_in_pod`
  - `node_management`

#### **Container User**
- **Zweck**: Environment & Container Lifecycle Management
- **Hauptfunktionen**: Environments erstellen, konfigurieren, Dateien verwalten
- **Wichtige Tools**:
  - `environment_list`
  - `environment_config`
  - `environment_file_read`
  - `environment_file_write`

#### **code-sandbox-mcp**
- **Zweck**: Code Execution Sandboxes
- **Hauptfunktionen**: Container-basierte Code-Ausführung
- **Wichtige Tools**:
  - `sandbox_initialize`
  - `write_file_sandbox`
  - `copy_file_from_sandbox`
  - `sandbox_stop`

---

### 3. Development & Tools (6 Server)

#### **archon**
- **Zweck**: RAG (Retrieval Augmented Generation) Knowledge Base & Code Search
- **Hauptfunktionen**: Knowledge Base durchsuchen, Code-Beispiele finden
- **Wichtige Tools**:
  - `rag_search_knowledge_base`
  - `rag_search_code_examples`
  - `rag_get_available_sources`

#### **filesystem**
- **Zweck**: Dateisystem-Zugriff (sicherer, eingeschränkter Zugriff)
- **Hauptfunktionen**: Dateien lesen, schreiben, Verzeichnisse listen
- **Wichtige Tools**:
  - `read_file`
  - `list_directory`
  - `directory_tree`
  - `get_file_info`
  - `list_allowed_directories`

#### **mcp-compass**
- **Zweck**: MCP Server Discovery & Recommendations
- **Hauptfunktionen**: Externe MCP Server finden und empfehlen
- **Wichtige Tools**: `recommend-mcp-servers`
- **Besonderheit**: Hilft neue MCP Server zu finden

#### **mcp-graphql**
- **Zweck**: GraphQL Query Execution
- **Hauptfunktionen**: GraphQL Schema introspection, Queries
- **Wichtige Tools**: `introspect-schema`

#### **mcp-knowledge-graph**
- **Zweck**: Knowledge Graph & AIM Datenbanken
- **Hauptfunktionen**: Multi-location Datenbanken verwalten
- **Wichtige Tools**: `aim_list_databases`

#### **mcp-neurolora**
- **Zweck**: MCP Server Installation & Code Collection
- **Hauptfunktionen**: Base Server installieren, Code sammeln
- **Wichtige Tools**:
  - `install_base_servers`
  - `collect_code`

#### **swagger-mcp**
- **Zweck**: Swagger/OpenAPI Dokumentation
- **Hauptfunktionen**: API Endpoints aus Swagger Docs
- **Wichtige Tools**: `list_endpoints`

#### **taskmaster**
- **Zweck**: Task/Projekt-Management mit AI
- **Hauptfunktionen**: Tasks erstellen, verwalten, AI-unterstützte Analyse
- **Wichtige Tools**:
  - `add_task` - Tasks hinzufügen
  - `get_task` - Task Details
  - `update_task` - Tasks aktualisieren
  - `next_task` - Nächste Aufgabe
  - `analyze_project_complexity` - Komplexitätsanalyse
  - `initialize_project` - Projekt initialisieren

---

### 4. Monitoring & Observability (2 Server)

#### **MCP_DOCKER**
- **Zweck**: Umfassende Docker/Grafana/Jira/Pyroscope Integration
- **Hauptfunktionen**:
  - Grafana: Dashboards, Alerts, Datasources, OnCall, Incidents
  - Jira: Issue Management, Projekte, Workflows
  - Pyroscope: Profiling, Label Management
  - Loki: Log Label Management
  - Prometheus: Metrics, Labels
- **Wichtige Tools** (Auswahl):
  - Grafana: `list_datasources`, `search_dashboards`, `list_contact_points`, `get_alert_rule_by_uid`
  - Jira: `jira_create_issue`, `jira_get_issue`, `jira_update_issue`, `jira_search_fields`
  - Prometheus: `query_prometheus`, `list_prometheus_metric_names`
  - Loki: `list_loki_label_names`
  - Docker: `docker`, `sandbox_initialize`, `run_js`, `run_js_ephemeral`
- **Besonderheit**: Sehr umfangreich, kombiniert mehrere Tools

#### **prometheus-mcp-server**
- **Zweck**: Prometheus Metriken-Zugriff
- **Hauptfunktionen**: Metriken entdecken
- **Wichtige Tools**: `prom_discover`

---

### 5. Databases & Storage (1 Server)

#### **supabase**
- **Zweck**: Supabase Dokumentation & GraphQL
- **Hauptfunktionen**: Supabase Docs durchsuchen
- **Wichtige Tools**: `search_docs`

---

### 6. Weitere spezialisierte Server (2 Server)

#### **wcgw**
- **Zweck**: Shell Command Execution & File Operations
- **Hauptfunktionen**: Bash Commands, File Read/Write
- **Wichtige Tools**:
  - `BashCommand` - Shell Commands
  - `ReadFiles` - Dateien lesen
  - `FileWriteOrEdit` - Dateien schreiben/editieren

#### **applescript_execute**
- **Zweck**: macOS AppleScript Automation
- **Hauptfunktionen**: Mac Apps steuern (Notes, Calendar, Contacts, Mail, Finder)
- **Wichtige Tools**: `applescript_execute`
- **Besonderheit**: macOS-spezifisch

---

## 📊 Statistiken nach Kategorie

| Kategorie | Anzahl Server | Prozent |
|-----------|---------------|---------|
| AWS & Cloud | 9 | 34.6% |
| Container & Kubernetes | 6 | 23.1% |
| Development & Tools | 6 | 23.1% |
| Monitoring & Observability | 2 | 7.7% |
| Databases & Storage | 1 | 3.8% |
| Spezialisierte Tools | 2 | 7.7% |
| **Gesamt** | **26** | **100%** |

---

## 🔥 Top 10 MCP Server nach Funktionsumfang

1. **MCP_DOCKER** - Multi-Tool Platform (Grafana, Jira, Prometheus, Docker)
2. **awslabs.aws-serverless-mcp-server** - Kompletter SAM Lifecycle
3. **awslabs.eks-mcp-server** - Umfassendes EKS Management
4. **mcp-server-kubernetes** - Kubernetes API Abstraction
5. **filesystem** - Sicherer Dateisystem-Zugriff
6. **taskmaster** - AI-unterstütztes Task Management
7. **archon** - RAG Knowledge Base
8. **awslabs.terraform-mcp-server** - IaC Management
9. **wcgw** - Shell & File Operations
10. **mcp-k8s-go** - Kubernetes Go Client

---

## 🎯 Empfohlene Server für häufige Use Cases

### Cloud Infrastructure
- `awslabs.aws-serverless-mcp-server` - Serverless Apps
- `awslabs.eks-mcp-server` - Kubernetes auf AWS
- `awslabs.terraform-mcp-server` - Infrastructure as Code

### Container Orchestration
- `mcp-server-kubernetes` - Kubernetes Management
- `docker-mcp` - Docker Container
- `mcp-k8s-go` - Advanced K8s Operations

### Monitoring & Debugging
- `MCP_DOCKER` - Grafana/Prometheus/Loki
- `prometheus-mcp-server` - Prometheus Metrics
- `awslabs.eks-mcp-server` - CloudWatch Integration

### Development
- `filesystem` - File Operations
- `code-sandbox-mcp` - Code Execution
- `taskmaster` - Project Management
- `archon` - Code Search & Knowledge

### Documentation
- `awslabs.aws-documentation-mcp-server` - AWS Docs
- `swagger-mcp` - API Documentation
- `supabase` - Supabase Docs

---

## ⚠️ Wichtige Hinweise

1. **Quarantine System**: Neue Server werden automatisch in Quarantäne gesetzt zur Verhinderung von Tool Poisoning Attacks (TPAs)

2. **Read-Only vs Write**: Einige Server erfordern spezielle Flags:
   - `--allow-write` für schreibende Operationen
   - `--allow-sensitive-data-access` für sensible Daten

3. **MCP Server Discovery**: Nutze `mcp-compass` um neue MCP Server zu finden

4. **Tool Retrieval**: Verwende `retrieve_tools` mit spezifischen Queries für gezielte Tool-Suche

---

## 📝 Nächste Schritte

1. Testplan für kritische Tools erstellen
2. Automatisierte Tests implementieren
3. Performance-Benchmarks durchführen
4. Integration Tests zwischen Servern

---

**Ende der Übersicht**
