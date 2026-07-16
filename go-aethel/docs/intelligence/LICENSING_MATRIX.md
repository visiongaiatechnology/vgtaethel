# LICENSING MATRIX — VGT AETHEL UNIFIED PERSONAL INTELLIGENCE SYSTEM

## Reference Projects License Assessment

Aethel maintains absolute sovereignty and prevents direct licensing contamination by treating reference code exclusively as conceptual inspiration rather than direct copy-paste.

| Project | License | Usage Mode in Aethel | Contamination Risk & Mitigation |
| :--- | :--- | :--- | :--- |
| **WorldWideView** | Elastic License 2.0 | **Architecture & UX inspiration only.** No source code copied. Analysed how the real-time event bus and the 3D globe visualization work. | **High (Commercial restrictions)**: Elastic 2.0 prohibits offering it as a managed service. *Mitigation*: The entire map projection and drawing system in Aethel is rewritten from scratch in pure JS canvas code. No WWV code or assets are integrated. |
| **WorldMonitor** | AGPL-3.0 | **Scoring & API architecture inspiration.** Mapped how multi-dimensional risk scores and region engines correlate. No code copied. | **Critical (AGPL viral contamination)**: Code reuse would trigger source-disclosure requirements. *Mitigation*: The Region Engine, multi-dimensional risk formulas, and report generation in Aethel are clean-room implementations written from scratch in Go. |
| **GaiasEye** | Proprietary / Custom | **Evidence, case isolation, and pseudonymization kernel.** Translated GaiasEye's HMAC alias structure and evidence sealing principles. | **Low (Internal sovereign domain)**: Designed to align with our local-first security boundaries. *Mitigation*: We reuse the existing Aethel Guard Kernel cryptographic primitives to implement the HMAC-SHA256 alias generation. |
