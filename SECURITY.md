# Security policy

## Supported version

Security fixes are developed for AETHEL Beta V3 (`1.0.0-beta.3`). Older alpha and beta
builds are unsupported.

## Reporting a vulnerability

Use GitHub Private Vulnerability Reporting for the repository. If that facility is not
available, contact VisionGaia Technology through its official website and request a private
security channel before sharing technical details.

Include the affected version, reproduction prerequisites, impact and a minimal proof. Never
include provider credentials, personal workspaces, chat history, memory stores, audit data,
private documents or unrelated system information.

Do not open a public issue until a coordinated disclosure date has been agreed.

## Security model

AETHEL applies capability routing, input validation, path jails, policy evaluation,
one-time approvals, effect verification and tamper-evident local audit records. These controls
reduce agent risk but do not turn the desktop process into an OS sandbox. Approved host-control
skills run with the permissions of the current Windows user.

Operators should use least-privilege accounts, retain backups, review every high-risk approval
and install only signed builds from a verified release.
