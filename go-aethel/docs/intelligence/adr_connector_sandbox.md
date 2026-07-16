# ADR: Sandboxed Connector Permissions & Polling Rate-Limits

## Status
Proposed

## Context
Aethel needs to fetch data from diverse public web sources (RSS, APIs). Untrusted connector code could lead to SSRF, data leakage, or paywall/abuse violations.

## Decision
* **Verification classification**: Connectors are classified as `BuiltInTrusted`, `LocallyApproved`, `CommunityUnverified`, or `Blocked`.
* **Resource Sandbox**: Every connector must define a manifest containing: URL patterns, permissions, required secrets, and rate-limits.
* **Dialer Restrictions**: Custom HTTP dialer enforces rate limits and blocks private IP ranges.

## Consequences
* Safe execution of third-party ingest connectors without risk of network compromise.
