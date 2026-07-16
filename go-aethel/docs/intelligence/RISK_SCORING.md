# RISK SCORING — VGT AETHEL UNIFIED PERSONAL INTELLIGENCE SYSTEM

## Principles
1. **No Intransparent Averages**: The Overall Risk Score must not be a simple average. The drivers, weights, and inputs must be fully explainable.
2. **Determinism**: The actual score calculation must be implemented as versioned, unit-tested code. An LLM may suggest parameters or label drivers, but it must not generate the scores dynamically in an unstructured manner.
3. **Traceability**: The operator can request *"Why is this score so high?"* and receive a breakdown of active events, weight factors, data freshness penalty, and missing data gaps.

## Scoring Formula
Each dimension (Geopolitical, Conflict, Cyber, Infrastructure, Economic, Financial, Energy, SupplyChain, Climate, PublicSafety) is scored between 0 and 100:

\[S_{dim} = \min\left(100, \sum_{i} W_i \times E_i \times F_i \times C_i\right)\]

Where:
* \(W_i\): Weight of event type \(i\) (e.g., Conflict event has weight 15, Infrastructure outage weight 10).
* \(E_i\): Event count or severity multiplier (e.g. Low = 1.0, Medium = 1.5, High = 2.5).
* \(F_i\): Freshness decay factor. Decreases over time since the last observation:
  \[F_i = e^{-\lambda \Delta t}\]
  Where \(\Delta t\) is the hours elapsed, and \(\lambda\) is the decay constant (e.g., \(\lambda = 0.02\), meaning score half-life is ~35 hours).
* \(C_i\): Confidence multiplier (0.0 to 1.0) of the source observations.

## Overall Risk Score
The Overall Risk Score (\(S_{overall}\)) is a weighted combination of the individual dimension scores, heavily biased towards the maximum dimension risk to prevent hiding a catastrophic single-point failure (e.g. a complete local cyber attack or power grid collapse) in a benign average:

\[S_{overall} = \alpha \times \max(S_{dim}) + (1 - \alpha) \times \frac{1}{N} \sum_{dim} S_{dim}\]

Where:
* \(\alpha\) is the bias factor (typically \(0.6\)).
* \(\max(S_{dim})\) is the highest risk dimension.
* The second term represents the average background risk.
* If any dimension risk is above a critical threshold (e.g. > 80), it triggers an escalation warning alert.
