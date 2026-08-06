# CSPADK Workflow Principles

This document defines the core principles and constraints that apply to all process skills in the CSPADK SDD (Spec-Driven Development) pipeline. Every process skill MUST enforce these principles during execution.

## 1. No Automatic Commits

AI agents are strictly prohibited from automatically executing `git commit`. All changes MUST be left in the working directory (or staged via `git add`) for the user to review and manually commit.

## 2. No Automatic Code Generation (Apply)

During the specification phase, AI agents SHALL NOT automatically apply changes or generate code unless the user explicitly requests it via `openspec-apply-change` (or equivalent).

## 3. Top-Down Module Breakdown with Zero Overlap

When breaking down requirements, use a top-down approach. Split the work into modules based on effort. **Crucially, ensure that the split modules have no overlapping code or directories.** This is the highest priority constraint to allow multiple tasks to be implemented in parallel by different developers or agent sessions.

## 4. Iterative OpenSpec Execution

Break down requirement points and implement their OpenSpec artifacts (proposal, design, tasks) step-by-step. Do not try to implement all artifacts for an entire epic in a single shot. You may suggest using multiple parallel chat windows for designing and implementing different modules.

## 5. Interactive Updates

If a proposal needs adjustment, use the `AskUserQuestion` tool to clarify the exact adjustment points first, and then update the proposal. Do not guess.

## 6. Sub-Agent Delegation

During the workflow, delegate the actual application of changes (`openspec apply`) to sub-agents (e.g., the `openspec-apply-change` skill) to minimize context window bloat in the main session.

## 7. Change vs. Requirement Relationship

A single requirement can involve multiple OpenSpec changes. If details need adjustment during implementation, create a new change. The goal of a change is to formalize a specific modification, not to strictly map 1:1 to a high-level requirement.

## 8. Breakpoint Resumption

The workflow supports being interrupted and resumed. When resumed, identify the current branch and its relationship with the requirement to determine the correct context and current phase.

## 9. Requirement Status Sync

**After EVERY step completes**, you MUST update `.ttadk/requirements-status/<requirement-name>.json`. This includes updating the current status, paths to all generated documents (`docs` directory, breakdown, openspec, etc.), and execution progress.
