# Hexagonal Architecture (Ports and Adapters)

## Purpose
This document explains what Hexagonal Architecture is, and why Image Factory uses it. It is meant to give a practical mental model for contributors and reviewers.

## What It Is
Hexagonal Architecture (also called Ports and Adapters) organizes the system around the domain model and use cases, and treats all external inputs and outputs as replaceable adapters.

The key idea is that the core business logic does not depend on frameworks, databases, or transports. Instead, it depends on interfaces (ports). Adapters implement those ports to connect the core to the outside world.

## Core Concepts
- Domain core: the business rules and entities. It should be stable and framework-agnostic.
- Ports: interfaces that describe what the domain needs from the outside world, or what the outside world can call into.
- Adapters: concrete implementations of ports. They translate from a specific technology or protocol into the domain’s language.
- Direction of dependency: adapters depend on the domain, never the other way around.

## How It Maps to This Repo
The backend is organized around these boundaries.

- `backend/internal/domain`: domain entities, value objects, and services.
- `backend/internal/adapters/primary`: driving adapters (HTTP handlers, REST, WebSocket).
- `backend/internal/adapters/secondary`: driven adapters (Postgres repositories, email, external integrations).
- Ports are expressed as Go interfaces in the domain and repository layers.

Example flow:
- An HTTP request hits a REST handler in `adapters/primary`.
- The handler calls a domain service or use-case.
- The domain service calls a repository port to persist or query data.
- A Postgres adapter implements that port in `adapters/secondary`.

## Why We Chose It
Hexagonal Architecture aligns with how Image Factory needs to evolve.

- Multiple interfaces: the product must support REST APIs, background workers, and future integration points.
- Replaceable infrastructure: database choices, queues, and external services can evolve without rewriting domain logic.
- Testability: domain logic can be unit-tested with in-memory fakes instead of real infrastructure.
- Security and compliance: clear boundaries reduce accidental data exposure and help auditing.
- Long-term maintainability: core rules remain stable even as tools, frameworks, and transports change.

## Practical Benefits You Will See
- Handlers are thin and map HTTP to domain calls.
- Services are reusable across REST endpoints and background jobs.
- Repositories are interfaces, so swapping Postgres for another store is contained.
- Integration tests can focus on adapter behavior without touching business rules.

## Tradeoffs
This pattern adds a little more upfront structure.

- More interfaces to define and maintain.
- Slightly more boilerplate when adding new adapters.
- Requires discipline to avoid leaking framework-specific types into the domain.

## Guidance for Contributors
- Keep domain logic free of HTTP, SQL, and framework dependencies.
- Add new integrations as adapters, not inside the domain.
- If you need a new dependency, add it in an adapter layer and define a port.
- Prefer dependency injection over global access to infrastructure.

## Related Docs
- `docs/architecture/REFERENCE_ARCHITECTURE.md`
- `docs/planning/02_OVERALL_ARCHITECTURE.md`
- `docs/phases/PHASE_STRUCTURE_AND_SEPARATION.md`
