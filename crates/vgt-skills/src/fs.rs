use crate::{VgtSkill, RiskLevel};
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::path::{Path, PathBuf};
use tokio::fs;
use anyhow::{Context, Result};

// VGT SECURITY POLICY: SANDBOX ROOT
const WORKSPACE_DIR: &str = "./vgt_workspace";

/// Helper: Validiert, ob der Pfad sicher ist (innerhalb der Sandbox)
fn validate_path(path_str: &str) -> Result<PathBuf> {
    let root = Path::new(WORKSPACE_DIR).canonicalize()
        .unwrap_or_else(|_| PathBuf::from(".")); // Fallback wenn dir noch nicht existiert
    
    // Simuliere Pfad-Resolution
    let target = Path::new(WORKSPACE_DIR).join(path_str);
    
    // Normalisiere Pfad (entferne ..)
    // Hinweis: In Production muss hier eine robustere Path-Traversal-Prevention hin
    if target.to_string_lossy().contains("..") {
        anyhow::bail!("SECURITY VIOLATION: Path Traversal detected.");
    }

    Ok(target)
}

// --- SKILL: READ FILE ---

pub struct ReadFileSkill;

#[derive(Deserialize)]
struct ReadFileArgs {
    path: String,
}

#[async_trait]
impl VgtSkill for ReadFileSkill {
    fn name(&self) -> &str { "fs_read_file" }
    
    fn description(&self) -> &str {
        "Liest den Inhalt einer Datei aus dem VGT Workspace. Erlaubt nur Textdateien."
    }
    
    fn parameters(&self) -> Value {
        serde_json::json!({
            "type": "object",
            "properties": {
                "path": { "type": "string", "description": "Relativer Pfad zur Datei (z.B. 'docs/readme.md')" }
            },
            "required": ["path"]
        })
    }
    
    fn risk_level(&self) -> RiskLevel { RiskLevel::Safe }

    async fn execute(&self, args: Value) -> Result<String> {
        let input: ReadFileArgs = serde_json::from_value(args)?;
        let target_path = validate_path(&input.path)?;

        let content = fs::read_to_string(&target_path).await
            .context(format!("Konnte Datei nicht lesen: {:?}", target_path))?;
            
        Ok(content)
    }
}

// --- SKILL: WRITE FILE ---

pub struct WriteFileSkill;

#[derive(Deserialize)]
struct WriteFileArgs {
    path: String,
    content: String,
}

#[async_trait]
impl VgtSkill for WriteFileSkill {
    fn name(&self) -> &str { "fs_write_file" }
    
    fn description(&self) -> &str {
        "Schreibt Text in eine Datei. Erstellt Verzeichnisse automatisch. ÜBERSCHREIBT existierende Dateien."
    }
    
    fn parameters(&self) -> Value {
        serde_json::json!({
            "type": "object",
            "properties": {
                "path": { "type": "string", "description": "Zielpfad" },
                "content": { "type": "string", "description": "Der vollständige Inhalt der Datei" }
            },
            "required": ["path", "content"]
        })
    }
    
    fn risk_level(&self) -> RiskLevel { RiskLevel::Critical } // User Approval Required

    async fn execute(&self, args: Value) -> Result<String> {
        let input: WriteFileArgs = serde_json::from_value(args)?;
        let target_path = validate_path(&input.path)?;

        // Sicherstellen, dass Parent Directory existiert
        if let Some(parent) = target_path.parent() {
            fs::create_dir_all(parent).await?;
        }

        fs::write(&target_path, input.content).await
            .context("Fehler beim Schreiben der Datei")?;
            
        Ok(format!("Datei erfolgreich geschrieben: {}", input.path))
    }
}