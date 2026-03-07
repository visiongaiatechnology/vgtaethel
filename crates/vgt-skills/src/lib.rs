pub mod fs;
pub mod shell;
pub mod rag;
pub mod guard; // NEU: Security Guard Export

use async_trait::async_trait;
use serde_json::Value;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum RiskLevel {
    Safe,       // Read-Only, Memory Ops
    Moderate,   // Netzwerkintern
    Critical,   // System-Mutation (Shell, FS Write)
}

/// Das VGT Skill Interface.
#[async_trait]
pub trait VgtSkill: Send + Sync {
    fn name(&self) -> &str;
    fn description(&self) -> &str;
    fn parameters(&self) -> Value;
    fn risk_level(&self) -> RiskLevel;
    async fn execute(&self, args: Value) -> anyhow::Result<String>;
}

/// Die Skill-Registry (Waffenkammer)
pub struct SkillRegistry {
    skills: std::collections::HashMap<String, Box<dyn VgtSkill>>,
}

impl SkillRegistry {
    pub fn new() -> Self {
        Self { skills: std::collections::HashMap::new() }
    }

    pub fn register(&mut self, skill: Box<dyn VgtSkill>) {
        self.skills.insert(skill.name().to_string(), skill);
    }

    pub fn get(&self, name: &str) -> Option<&dyn VgtSkill> {
        self.skills.get(name).map(|b| b.as_ref())
    }
    
    pub fn to_tool_definitions(&self) -> Vec<Value> {
        self.skills.values().map(|s| {
            serde_json::json!({
                "type": "function",
                "function": {
                    "name": s.name(),
                    "description": s.description(),
                    "parameters": s.parameters()
                }
            })
        }).collect()
    }
}