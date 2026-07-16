# OSINT BOUNDARIES — VGT AETHEL UNIFIED PERSONAL INTELLIGENCE SYSTEM

## Principles
1. **Local-First Privacy**: Aethel does not publish, share, or leak personal identifier information (PII) to external models or servers.
2. **Case-Scoped Pseudonymization**:
   * All person entities are automatically assigned a stable pseudonym derived via HMAC-SHA256, scoped to the specific Case:
     \[\text{Pseudonym} = \text{HMAC-SHA256}(\text{CaseSecret}, \text{Normalized Name})\]
   * The Master Secret used to derive the `CaseSecret` is platform-protected and never written to logs or public files.
3. **Re-ID Gate (Re-identification)**:
   * Re-identification is a critical security action.
   * It requires an explicit operator request, listing a clear purpose.
   * Every Re-ID request generates a permanent entry in the system audit trail.
   * The cleartext identity is revealed only temporarily in the session context and is never written to disk or sent to the LLM.
4. **Surveillance Prohibitions**:
   * Aethel strictly prohibits:
     * Automatic facial recognition.
     * Invasive target tracking or doxxing.
     * Accessing paywalled, credential-protected, or private targets without authorization.
     * Storing cleartext biometrics.
