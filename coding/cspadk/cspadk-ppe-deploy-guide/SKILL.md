---
name: cspadk-ppe-deploy-guide
description: Guide PPE deployment preparation, rollout validation, and rollback planning for CSPADK changes.
version: 1.2.0
metadata:
  patterns:
    - generator
    - tool-wrapper
  domain: cspadk
  i18n_level: 0
  prompt_version: "1.2.0"
  agent_support:
    - claude-code
    - trae
    - trae-cli
    - coco
  language:
    - en
---

# cspadk-ppe-deploy-guide

## Overview

This skill guides users through deploying a change to PPE and validating that the environment is ready, the rollout is safe, and the result is verifiable. It integrates with HiWorks/BITS for development task creation and uses `csp-config.json` for configuration.

## Trigger Scenarios

- When the user asks to deploy to PPE or prepare a PPE verification plan.
- When a change needs a pre-production rollout checklist.
- When the team needs deployment validation steps before release approval.
- When rollback preparation and smoke checks must be documented clearly.
- When creating BITS development tasks for PPE deployment.

## Inputs

- The service, module, or change being deployed.
- The deployment target, environment information, and any required release identifiers.
- The expected post-deploy behavior, metrics, or smoke-test scenarios.

## Outputs

- A PPE deployment checklist.
- A rollout validation plan covering pre-checks, deployment checks, and post-deploy verification.
- Rollback and contingency guidance.
- A concise deployment readiness summary.
- BITS development task creation command with parameters.

## Configuration

This skill reads configuration from `.ttadk/csp-config.json` (fields align with `hiworks bits-task-create` command):

```json
{
  "bits": {
    // Task Creation
    "title": "",
    "services": "",
    "change": [],
    "lane": "test",
    "enableLanes": "both",

    // SCM Configuration
    "scmMode": "branch",
    "scmBranch": "",

    // Template & Workflow
    "fromDevId": "",
    "devTaskTemplateId": "",
    "workflowSnapshotId": "",
    "teamFlowId": "",
    "devTaskMode": "",

    // Service Configuration
    "serviceType": "PROJECT_TYPE_TCE",
    "idcs": ["lf", "lq"],

    // Team & Tracking
    "developer": "dev@bytedance.com",
    "qa": "qa@bytedance.com",
    "meego": "",

    // Custom Variables
    "vars": {},

    // Advanced
    "envSettingMapJson": "",
    "cloudSite": "prod",
    "apiBaseUrl": "",
    "jwtToken": ""
  }
}
```

## Core Principles

- Safety comes before speed; confirm prerequisites before rollout.
- Verify both deployment success and expected behavior after deployment.
- Prefer observable checks over assumptions.
- Make rollback readiness explicit before deployment begins.
- Tailor the checklist to the actual service and environment, not a generic release script.

## Execution Steps

### Step 1: Gather Deployment Context

- **Action**: Identify the target service, environment, version, and deployment mechanism.
- **Action**: Confirm whether the change affects code, configuration, data, or infrastructure.
- **Action**: Identify any dependencies, sequencing requirements, or approval gates.
- **Action**: Read `.ttadk/csp-config.json` for BITS configuration defaults.

### Step 2: Create BITS Development Task

Use the `hiworks-deploy-skill` to create a BITS development task:

```bash
# Read current branch for SCM branch
git branch --show-current

# Create BITS dev task using csp-config.json defaults
cspadk hiworks bits-task-create \
  --title "<task-title>" \
  --services "<service-psm>" \
  --lane "<lane>" \
  --scm-branch "<current-branch>" \
  --meego "<meego-url-or-id>" \
  --team-flow-id <from-csp-config>
```

**Parameter Priority** (from hiworks-deploy-skill):
1. User input (highest priority)
2. Change context from `openspec/changes/<changeName>`
3. Repository docs (AGENTS.md, deploy.md)
4. `.ttadk/csp-config.json` defaults
5. Local runtime data (current branch, etc.)

### Step 3: Prepare the Pre-Deployment Checklist

- **Action**: List the prerequisites required before rollout.
- **Action**: Confirm that build artifacts, config changes, and environment variables are ready.
- **Action**: Confirm the expected smoke-test scenarios and observability signals.

### Step 4: Define Deployment-Time Checks

- **Action**: Describe the checks to perform during rollout, such as instance health, release progress, and configuration application.
- **Action**: Call out any canary, staged, or manual confirmation points when relevant.
- **Action**: Identify stop conditions that should abort or pause the rollout.

### Step 5: Define Post-Deployment Verification

- **Action**: List functional smoke checks, log checks, and key metrics to inspect after deploy.
- **Action**: Include critical user paths and service dependencies that must remain healthy.
- **Action**: Distinguish immediate verification from longer-tail monitoring.

### Step 6: Prepare Rollback Guidance

- **Action**: Describe what rollback path is available for the change.
- **Action**: Note what evidence should trigger rollback.
- **Action**: Summarize any extra checks needed after rollback completes.

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
