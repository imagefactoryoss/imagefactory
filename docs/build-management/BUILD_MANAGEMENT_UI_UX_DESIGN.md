# Build Management UI/UX Design

This document describes the current UI and UX patterns for build creation and monitoring.

## Primary Screens

- **Build List**: status, type, duration, created time, and actions.
- **Build Details**: configuration summary, execution status, logs, and artifacts.
- **Build Wizard**: step-by-step configuration with validation.

---

## UX Principles

- Validate inputs on save/submit, not per keystroke.
- Preserve user inputs across steps.
- Provide explicit feedback for queueing and dispatcher activity.

---

## Build Wizard Flow

1. **Basics**: name, type, repository context.
2. **Configuration**: method-specific fields (dockerfile, build context, registry).
3. **Infrastructure**: choose execution environment and provider.
4. **Review & Submit**: validation summary and submit.
