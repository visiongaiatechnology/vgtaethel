use regex::Regex;
use lazy_static::lazy_static;
use serde_json::Value;
use anyhow::{Result, anyhow};
use tracing::warn;

// VGT THREAT DEFINTIONS
// Diese Regexes sind aggressiv. Sie fangen lieber zu viel als zu wenig.

lazy_static! {
    // 1. SHELL INJECTION: Erkennt Versuche, Befehle zu verketten oder Subshells zu öffnen.
    // Blockiert: &&, ||, ;, ``, $(), |, >, <
    static ref RE_SHELL_INJECTION: Regex = Regex::new(r"([;&|><`$]|\$\(|\)\))").unwrap();

    // 2. DESTRUCTIVE COMMANDS: Befehle, die das System löschen oder formatieren können.
    static ref RE_DESTRUCTIVE: Regex = Regex::new(r"(?i)\b(rm\s+-[rf]+|mkfs|dd|shred|wipe|format|:[\(\)\{\}\|:&;]+)\b").unwrap();

    // 3. PATH TRAVERSAL: Erkennt Versuche, aus der Sandbox auszubrechen.
    // Blockiert: ../, ..\, /etc/, C:\Windows
    static ref RE_PATH_TRAVERSAL: Regex = Regex::new(r"(\.\./|\.\.\\|/etc/|C:\\Windows|/var/|/usr/)").unwrap();
    
    // 4. NETWORK EXFILTRATION: Verdächtige Netzwerk-Tools (optional, je nach Need)
    static ref RE_NET_EXFIL: Regex = Regex::new(r"(?i)\b(nc|netcat|ncat|curl|wget|ssh)\b").unwrap();
}

#[derive(Debug, Clone)]
pub struct ThreatReport {
    pub is_safe: bool,
    pub threats: Vec<String>,
    pub risk_score: u8, // 0-100
}

pub struct SecurityGuard;

impl SecurityGuard {
    /// Die Haupt-Analyse-Funktion. Scannt Argumente basierend auf dem Tool-Typ.
    pub fn scan(tool_name: &str, args: &Value) -> ThreatReport {
        let mut threats = Vec::new();
        let mut risk_score = 0;

        // Argumente in String konvertieren für Regex-Scan
        let args_str = args.to_string();

        // GLOBAL CHECK: Path Traversal (Gilt für ALLE Tools)
        if RE_PATH_TRAVERSAL.is_match(&args_str) {
            threats.push("PATH_TRAVERSAL_DETECTED: Versuch, die Sandbox zu verlassen.".into());
            risk_score += 50;
        }

        // TOOL SPECIFIC CHECKS
        match tool_name {
            "sys_exec_cmd" => {
                // Shell Injection Check
                if let Some(caps) = RE_SHELL_INJECTION.captures(&args_str) {
                    threats.push(format!("SHELL_INJECTION_DETECTED: Illegales Zeichen '{}'", caps.get(0).unwrap().as_str()));
                    risk_score += 80;
                }

                // Destructive Command Check
                if let Some(caps) = RE_DESTRUCTIVE.captures(&args_str) {
                    threats.push(format!("DESTRUCTIVE_COMMAND_DETECTED: '{}'", caps.get(0).unwrap().as_str()));
                    risk_score += 100; // Kritisch
                }
                
                // Exfiltration Check
                if let Some(caps) = RE_NET_EXFIL.captures(&args_str) {
                    threats.push(format!("NETWORK_TOOL_DETECTED: '{}'", caps.get(0).unwrap().as_str()));
                    risk_score += 30; // Warnung
                }
            },
            "fs_write_file" => {
                // Bei Write Files prüfen wir, ob ausführbare Dateien geschrieben werden sollen
                if args_str.contains(".sh") || args_str.contains(".exe") || args_str.contains(".bat") {
                    threats.push("EXECUTABLE_WRITE_ATTEMPT: Schreiben von Skripten/Binaries.".into());
                    risk_score += 60;
                }
            },
            _ => {}
        }

        // Normalisierung
        if risk_score > 100 { risk_score = 100; }

        let is_safe = threats.is_empty();
        
        if !is_safe {
            warn!("VGT SECURITY INTERVENTION: Threats identified: {:?}", threats);
        }

        ThreatReport {
            is_safe,
            threats,
            risk_score,
        }
    }
}