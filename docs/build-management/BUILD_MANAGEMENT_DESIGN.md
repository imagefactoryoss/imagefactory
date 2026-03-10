# Build Management Design

This document describes the build management model and user flow used by Image Factory.

## Goals

- Provide a consistent build creation experience across methods.
- Validate configuration before queueing.
- Enable dispatcher-based execution with clear visibility.

---

## Core Flow

1. User selects build type and provides configuration.
2. System validates `BuildConfig` and tool availability.
3. Build is created with `status = queued`.
4. Dispatcher claims and executes the build.

---

## Key Components

- **BuildConfig**: Method-specific details (kaniko, buildx, packer, etc).
- **Dispatcher**: Claims queued builds and starts execution.
- **Executors**: Local or Tekton-based.

---

## System Guarantees

- One build config per build.
- Config validation before queueing.
- Dispatcher is the only component that starts execution.
