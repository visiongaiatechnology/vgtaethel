# PRIVACY MODEL — VGT AETHEL UNIFIED PERSONAL INTELLIGENCE SYSTEM

## Principles
1. **Local Storage First**: No analytics or operational data leaves the user's workstation.
2. **Encrypted-at-Rest**:
   * Storage blocks (like `intelligence_core.json` and `nexus_memory.json`) are locally encrypted using the AES-GCM store powered by the AETHEL Guard Kernel.
   * Keys are derived using PBKDF2 from a master password or system-protected keychain.
3. **Data Isolation Boundaries**:
   * **Case Context**: Stays strictly isolated in the designated Case container. It is not fed into the assistant's general background memory automatically.
   * **Personal Memory**: Contains operator-approved interests, preferences, and routines. It is loaded only when answering personal chat questions or doing correlation checks.
   * **Transient Observations**: Proposed RSS events remain in memory or unencrypted temp storage only long enough for correlation, and are deleted when superseded. No raw feed history is stored long-term without explicit Case sealing.
