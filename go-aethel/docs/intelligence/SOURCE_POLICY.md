# SOURCE POLICY — VGT AETHEL UNIFIED PERSONAL INTELLIGENCE SYSTEM

## Safety Boundaries for External Connections
1. **SSRF Protection**:
   * Block all private IP ranges (RFC 1918, RFC 4193, loopback, link-local, e.g., `127.0.0.1`, `10.0.0.0/8`, `192.168.0.0/16`) in DNS resolution and dialing to prevent local network scanning.
   * Enforce short connection timeouts (maximum 10 seconds) to prevent hanging socket attacks.
2. **Untrusted Content & Prompt Injection**:
   * Feeds, HTML snapshots, and API payloads must be treated as untrusted text.
   * They must NEVER be parsed as executable commands, HTML, script blocks, or code.
   * When sent to an LLM for summarization or event proposal, the engine must wrap them in strict separators and instruct the LLM to extract factual parameters (e.g. coordinates, title, domain) rather than executing instructions within the feed content.
3. **No Automatic Actions**:
   * Observations proposed from feeds remain in a `proposed` unverified status. They cannot trigger system configurations, file system access, or network calls autonomously without explicit operator confirmation.
4. **Reproducible Snapshots**:
   * When an observation is promoted to evidence, a local, immutable snapshot of the raw feed item / page content is cryptographically sealed (SHA-256) and saved locally.
