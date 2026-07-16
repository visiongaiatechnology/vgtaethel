# ADR: Versioned, Deterministic Risk Scoring

## Status
Proposed

## Context
Risk values in typical intelligence platforms are often vague averages or generated dynamically by LLMs, leading to non-reproducible and hallucinated alerts.

## Decision
* **Deterministic Logic**: Risk score calculation is implemented as Go code, using explicit math formulas.
* **Inputs**: Weights assigned to event categories, modified by freshness decay and confidence factors.
* **Explainability**: The system returns a structured breakdown of inputs alongside any calculated score, rendering it fully auditable by the operator.

## Consequences
* High reliability. No random fluctuations in risk scores between runs.
* Full testability via deterministic unit tests.
