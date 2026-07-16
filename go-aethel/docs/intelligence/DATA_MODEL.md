# DATA MODEL — VGT AETHEL UNIFIED PERSONAL INTELLIGENCE SYSTEM

## Core Intelligence Entities

### 1. Source
Describes the publisher or endpoint origin:
* `ID`: string (e.g. `src_spiegel_feed`)
* `Name`: string
* `URL`: string (HTTPS only)
* `SourceType`: string (rss, api, local, database)
* `Publisher`: string
* `TrustTier`: integer (1=verified, 2=community, 3=unverified)
* `FetchedAt`: timestamp
* `PublishedAt`: timestamp
* `ParserVersion`: string
* `ContentHash`: string

### 2. Observation
Raw observations as ingested from a Source, containing zero model interpretations:
* `ID`: string
* `SourceID`: string
* `RawText`: string
* `ObservedAt`: timestamp
* `Latitude`: float64
* `Longitude`: float64
* `ContentHash`: string

### 3. Evidence
Immutable sealed content with secure chain of custody:
* `ID`: string
* `CaseID`: string
* `SourceID`: string
* `Excerpt`: string
* `SHA256`: string
* `CollectedAt`: timestamp
* `Sealed`: boolean
* `ValidationStatus`: string (pending, verified, disputed, rejected)
* `ChainOfCustodyID`: string

### 4. Entity
A mapped actor, location, asset, or event (person names are pseudonymized):
* `ID`: string (e.g. `GX-PER-HASH` for pseudonymized persons)
* `Label`: string
* `Kind`: string (person, organisation, location, asset, event)
* `Confidence`: integer (0-100)

### 5. Relation
A semantic connection between two entities, backed by Evidence:
* `FromEntity`: string
* `ToEntity`: string
* `RelationType`: string
* `EvidenceIDs`: array of strings
* `Confidence`: integer (0-100)
* `ValidFrom`: timestamp
* `ValidUntil`: timestamp

### 6. Event
A classified happening with location and severity:
* `ID`: string
* `Title`: string
* `Summary`: string
* `Domain`: string (geo, cyber, economic, humanitarian, general)
* `Latitude`: float64
* `Longitude`: float64
* `Severity`: string (low, medium, high)
* `Confidence`: integer (0-100)
* `ObservedAt`: timestamp

### 7. Region
A geographic area defined by name, polygon, or radius:
* `ID`: string
* `Name`: string
* `Type`: string (state, city, zone, economic, custom)
* `Polygons`: array of coordinate rings

### 8. Assessment
An interpretation of observations and events by the model or operator:
* `ID`: string
* `Statement`: string
* `Confidence`: integer (0-100)
* `EvidenceIDs`: array of strings
* `Status`: string (hypothesis, unverified, corroborated, verified, disputed, disputed, rejected)
* `CreatedAt`: timestamp
* `GeneratedBy`: string (model, operator)

### 9. RiskScore
Evaluated risk scores across domains:
* `OverallRisk`: float64
* `GeopoliticalRisk`: float64
* `ConflictRisk`: float64
* `CyberRisk`: float64
* `InfrastructureRisk`: float64
* `EconomicRisk`: float64
* `Confidence`: integer (0-100)
* `Trend`: string (up, stable, down)
* `PrimaryDrivers`: array of strings
* `LastUpdated`: timestamp

### 10. Alert
A high-severity notification generated under rate limits and cooldown thresholds:
* `ID`: string
* `Severity`: string
* `Confidence`: integer
* `Region`: string
* `AffectedDomains`: array of strings
* `EvidenceIDs`: array of strings
* `Reason`: string
* `CreatedAt`: timestamp
* `Acknowledged`: boolean
* `EscalationState`: string

### 11. Case
An isolated investigation containing context, evidence, and audit logs:
* `ID`: string
* `Title`: string
* `Purpose`: string
* `Classification`: string
* `Status`: string (open, closed, archived)
* `Evidence`: array of Evidence
* `Entities`: array of Entity
* `Relations`: array of Relation
* `Audit`: array of AuditEvent
