# ADR: Polygon Region Model & Geo-Fencing

## Status
Proposed

## Context
Intelligence events must be resolved to geographic regions (e.g. states, cities, radius zones) to compute localized risks and filter map displays.

## Decision
* **Polygon Representation**: Mapped regions are defined by names and boundary coordinates.
* **Point-in-Polygon (PIP)**: Implement a ray-casting algorithm to resolve event coordinates (`lat, lon`) to specific active regions.
* **Hierarchical Scoring**: Region risks cascade from cities to provinces to states, with weights adjusted based on containment.

## Consequences
* Enables localized regional security briefs and risk assessments.
