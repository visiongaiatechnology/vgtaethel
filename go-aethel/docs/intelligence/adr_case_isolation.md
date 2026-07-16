# ADR: Case Isolation (OSINT Data Compartmentalization)

## Status
Proposed

## Context
OSINT target investigations could collect sensitive data (e.g. usernames, metadata). This information must not bleed into other cases or the assistant's standard background memories.

## Decision
* **Case Compartments**: Every Case contains its own list of entities, relations, and evidence.
* **No global cross-contamination**: The assistant's chat memory does not have automatic read access to Case files unless the Case is explicitly loaded or referenced in the query.
* **Pseudonymized Targets**: Person entities inside cases are hashed with Case-specific secrets.

## Consequences
* Leakage in one Case does not compromise the security or data of other Cases.
