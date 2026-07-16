# ADR: Immutable Evidence Sealing & Encryption-at-Rest

## Status
Proposed

## Context
In OSINT investigations, evidence integrity and chain of custody are paramount. We must guarantee that once evidence is collected, it cannot be silently modified by the LLM or an attacker.

## Decision
* **Immutability**: Every piece of sealed evidence is hashed using SHA-256. The hash, retrieval timestamp, operator identifier, and raw content excerpt are saved.
* **Locking**: Once marked as `sealed`, the evidence struct is read-only.
* **Encryption-at-Rest**: The JSON databases are encrypted using AES-256-GCM via Aethel's Guard Kernel before writing to the local file system.

## Consequences
* Cryptographically verifiable chain of custody for all OSINT cases.
* Leak protection if the workstation's hard drive is accessed by unauthorized third parties.
