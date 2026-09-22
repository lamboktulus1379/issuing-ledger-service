# 🚀 Go + Copilot Microservice Blueprint

This blueprint defines the Day Zero setup for any new Go microservice. It enforces a strict 4-layer Clean Architecture, integrates our standard infrastructure (MySQL, Redis, Kafka, OTLP), and pre-configures the Copilot AI agent to automate development while adhering to our ENGINEERING_PLAYBOOK.md.

## 1. Initialize the Standard Directory Tree
Run these commands in your terminal to instantly generate the compliant folder structure:

    # 1. Initialize Go module (replace with your module name)
    go mod init github.com/your-org/new-service
    
    # 2. Create Clean Architecture Layers
    mkdir -p domain/{model,dto,repository}
    mkdir -p usecase
    mkdir -p infrastructure/{persistence,cache,message}
    mkdir -p interfaces/{http,middleware}
    
    # 3. Create Human Documentation Folders
    mkdir -p docs/{rfd,feature-tasks,architecture}
    
    # 4. Create AI Agent (Copilot) Steering Folders
    mkdir -p .Copilot/steering

## 2. The Golden Files Checklist
Before writing any code, ensure these files exist in your repository to enforce governance.

### 📝 Human Documentation (docs/)
1. docs/ENGINEERING_PLAYBOOK.md: The master ruleset for humans. Defines the tech stack (MySQL FOR UPDATE, Kafka, Redis TTLs, OpenTelemetry) and the execution lifecycle.
2. docs/feature-tasks/_feature-task-template.md: The template used to plan every new feature. Enforces the Gate Checks: Requirements -> POC -> Implementation.

### 🤖 AI Agent Configuration (.github/)
These files act as the automated enforcers of your playbook. 
1. project.md (inclusion: always): Defines the business context (e.g., "High-concurrency B2B Fintech App"). Instructs the AI to read the Engineering Playbook for lifecycle rules.
2. tech.md (inclusion: always): Forces the AI to use Go 1.26, shopspring/decimal (no float64), and OpenTelemetry.
3. structure.md (inclusion: always): Maps the Clean Architecture boundaries so the AI never hallucinates folder names.
4. database.md (inclusion: fileMatch: "infrastructure/persistence/**/*.go"): Forces the AI to use *sql.Tx and MySQL SELECT ... FOR UPDATE row-locking when modifying database files.
5. planning.md (inclusion: manual): Used only when asking the AI to break down tasks. Enforces the I-XX micro-task breakdown and Clean Architecture separation.

## 3. The Execution Workflow
Whenever a new feature is requested, execute this loop:

1. Plan: Create docs/feature-tasks/your-feature.md from the template.
2. Gate Check: Manually approve G-01 (Requirements) and G-02 (POC).
3. AI Task Breakdown: Instruct Copilot: "Load #planning and break down the Implementation tasks for this feature into I-XX steps."
4. AI Execution: Instruct Copilot: "Execute task I-01. When done, output the exact Git commit command."
5. Commit: Run the provided micro-commit (e.g., feat(domain): [F-FEAT-001][I-01] define core models).
