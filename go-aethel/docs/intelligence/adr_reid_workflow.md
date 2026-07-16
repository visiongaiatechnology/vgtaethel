# ADR: Re-ID Workflow & Approval Gate

## Status
Proposed

## Context
Revealing the cleartext name behind a case-scoped pseudonym (Re-identification) is a high-risk security action that should be strictly governed to prevent unauthorized tracking or disclosure.

## Decision
* **Gate Mechanism**:
  1. Operator submits a Re-ID request via the UI.
  2. The system checks the `Identity Profile` permissions and prompts the operator to confirm the purpose.
  3. Upon confirmation, the backend generates an `AuditEvent` with the timestamp, actor, case ID, and reason.
  4. The cleartext identifier is returned to the UI session context in memory only. It is not saved to persistent logs or written to disk.
  5. The UI automatically clears the cleartext view after a short timeout (e.g. 5 minutes) or on session logout.

## Consequences
* Immutable audit trail for all target identification actions.
* Strict protection of target privacy.
