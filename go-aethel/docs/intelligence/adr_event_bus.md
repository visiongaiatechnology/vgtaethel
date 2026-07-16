# ADR: Decoupled Intelligence Event Bus

## Status
Proposed

## Context
Aethel consists of multiple modules: RSS Ingestion, 3D Globe canvas, Chat Interface, Alert Rules, and Risk Scoring. If these components call each other directly, the system will become rigid and hard to test.

## Decision
We implement a central, asynchronous, type-safe Event Bus in Go.
* **Mechanism**: Publishers post events of a registered `Type` (e.g., `source.fetched`, `alert.created`). Subscribers register a non-blocking channel to receive events.
* **Streaming**: Events of high importance (like `global_watch.command` or `alert.created`) are pushed to the frontend via Server-Sent Events (SSE).
* **Guarantees**: Delivery is asynchronous (non-blocking). If a subscriber channel is full, the event is dropped (or logged) to prevent blocking main operation.

## Consequences
* High modularity and clear testability.
* Real-time reactive updates on the map and chat UI.
