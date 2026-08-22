# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Go standard library `net/http` server with embedded HTML, CSS, and vanilla JavaScript. No frontend framework, frontend build step, or additional runtime dependency.

## Users

Software developers and coding-agent operators working in a local project who need to understand current work, ownership, status, hierarchy, and execution order.

## Product Purpose

Saga is local task management for human-agent collaboration. The web interface provides a read-only visual overview so a user can quickly understand what work exists, what state it is in, and what dependencies determine what can happen next.

## Positioning

Saga keeps agent work coordination and progress records close to the codebase in simple local files, while making explicit task hierarchy, claims, and dependencies available to both CLI-driven agents and human operators.

## Operating Context

Users work from a project directory initialized with `.saga/`. Agents continue to create, claim, update, and log work through the CLI. Humans launch `saga web` from the same project and inspect the current local task data in a browser.

## Capabilities and Constraints

- The first web interface is read-only.
- It must visualize task status, hierarchy, and hard dependencies.
- It should make blocked and actionable work easy to distinguish.
- It runs as a local HTTP server launched by `saga web` and should open the browser automatically.
- It uses the existing Saga storage and domain types.
- It should add minimal dependencies: the implementation uses only Go's standard library and embedded static assets.
- Existing CLI workflows and storage formats remain authoritative and unchanged.

## Brand Commitments

The product name is Saga. Existing terminology—including saga, sub-saga, active, paused, done, wontdo, claims, dependencies, and related work—must remain recognizable.

## Evidence on Hand

The repository README, command reference, Go domain types, existing CLI output, and local `.saga/` records are the factual sources. No external customer claims, benchmarks, or brand assets are available and none should be fabricated.

## Product Principles

- Local project truth stays close to the code.
- Agent-written data must remain easy for humans to inspect.
- Explicit relationships and states should reveal execution order without inference.
- Visualization complements the CLI rather than replacing agent workflows.
- Readability and operational clarity take precedence over decoration.
