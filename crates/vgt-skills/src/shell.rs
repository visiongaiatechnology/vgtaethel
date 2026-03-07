use crate::{VgtSkill, RiskLevel};
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::process::Stdio;
use tokio::process::Command;

#[derive(Serialize, Deserialize)]
struct ShellArgs {
    command: String,
    args: Vec<String>,
}

pub struct ExecuteCommandSkill;

#[async_trait]
impl VgtSkill for ExecuteCommandSkill {
    fn name(&self) -> &str {
        "sys_exec_cmd"
    }

    fn description(&self) -> &str {
        "Führt einen Systembefehl aus. NUR verwenden, wenn absolut notwendig. Verbotene Befehle: rm, dd, mkfs."
    }

    fn parameters(&self) -> Value {
        serde_json::json!({
            "type": "object",
            "properties": {
                "command": { "type": "string", "description": "Der Befehl (z.B. 'ls', 'git')" },
                "args": { "type": "array", "items": { "type": "string" }, "description": "Argumente für den Befehl" }
            },
            "required": ["command", "args"]
        })
    }

    fn risk_level(&self) -> RiskLevel {
        RiskLevel::Critical // Erfordert User-Bestätigung im UI
    }

    async fn execute(&self, args: Value) -> anyhow::Result<String> {
        let input: ShellArgs = serde_json::from_value(args)?;

        // VGT SECURITY POLICY: BLACKLIST CHECK
        let blacklist = vec!["rm", "dd", ":(){:|:&};:", "shutdown", "reboot"];
        if blacklist.contains(&input.command.as_str()) {
             return Err(anyhow::anyhow!("VGT SECURITY INTERVENTION: Befehl '{}' ist blockiert.", input.command));
        }

        // Execution
        let output = Command::new(&input.command)
            .args(&input.args)
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()?
            .wait_with_output()
            .await?;

        if output.status.success() {
            Ok(String::from_utf8_lossy(&output.stdout).to_string())
        } else {
            let err = String::from_utf8_lossy(&output.stderr);
            Err(anyhow::anyhow!("Command Failed: {}", err))
        }
    }
}