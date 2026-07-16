# ADR: Memory Separation (Personal vs World State)

## Status
Proposed

## Context
Aethel acts as both a personal assistant and a global intelligence platform. Mixing raw public RSS streams with the operator's personal private journals, keys, or project configs would pollute the context window and risk leakage of private data.

## Decision
* **Logical Isolation**:
  * **Personal Context**: Stored in `nexus_memory.json` (encrypted). Contains journals, routines, and user preferences.
  * **World State**: Stored in `intelligence_core.json`. Contains public events, sources, and cases.
* **Separation of Concerns**: LLM context prompts are constructed dynamically. Personal details are only attached to queries when the user specifically asks for correlation or personal impact.

## Consequences
* Enhanced security. Public feeds cannot access or leak personal context.
