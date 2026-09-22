---
inclusion: always
---
## 1. The Business Domain
We are building the core backend infrastructure for Tabodo Donuts. Specifically, we are migrating from a standard POS API into a StraitsX-style Omnichannel B2B Card Issuing platform. 

## 2. System Characteristics
- **High Concurrency:** The system handles simultaneous requests from multiple POS devices against shared corporate wallets.
- **Financial Strictness:** This is a financial ledger. Mathematical accuracy (Zero-Sum double-entry) and auditability are non-negotiable. 

## 3. Governance Reference
As an AI agent, you operate under strict organizational governance. If you are asked to plan a feature or define architecture, you MUST refer to the master `docs/ENGINEERING_PLAYBOOK.md` for our complete lifecycle rules.
