use crate::models::{ModelRegistry, AIModel};
use vgt_skills::SkillRegistry;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::env;
use std::sync::RwLock; // Interior Mutability
use anyhow::{Result, Context};
use futures_util::{Stream, StreamExt};
use std::pin::Pin;
use serde_json::Value;
use std::path::Path;

const GROQ_API_URL: &str = "https://api.groq.com/openai/v1/chat/completions";
const CONFIG_FILE: &str = "./vgt_workspace/vgt_config.json";

// ... existing structs (InferenceRequest, GroqPayload etc.) ...
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct InferenceRequest {
    pub model_id: String,
    pub prompt: String, 
    pub messages: Option<Vec<Value>>, 
    pub system_prompt: Option<String>,
    pub temperature: f32,
    pub use_tools: bool,
}

#[derive(Serialize)]
struct GroqPayload {
    model: String,
    messages: Vec<GroqMessage>,
    temperature: f32,
    stream: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    tools: Option<Vec<Value>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    tool_choice: Option<String>,
}

#[derive(Serialize, Deserialize, Clone)]
struct GroqMessage {
    role: String,
    content: String,
}

#[derive(Deserialize, Debug)]
struct GroqChunk {
    choices: Vec<GroqChunkChoice>,
}

#[derive(Deserialize, Debug)]
struct GroqChunkChoice {
    delta: GroqChunkDelta,
    finish_reason: Option<String>,
}

#[derive(Deserialize, Debug)]
struct GroqChunkDelta {
    content: Option<String>,
    tool_calls: Option<Vec<GroqToolCallDelta>>,
}

#[derive(Deserialize, Debug, Serialize)]
struct GroqToolCallDelta {
    index: u32,
    id: Option<String>,
    #[serde(rename = "type")]
    type_: Option<String>,
    function: Option<GroqFunctionDelta>,
}

#[derive(Deserialize, Debug, Serialize)]
struct GroqFunctionDelta {
    name: Option<String>,
    arguments: Option<String>,
}

#[derive(Serialize, Deserialize)]
struct VgtConfig {
    api_key: String,
}

pub struct InferenceEngine {
    client: Client,
    model_registry: ModelRegistry,
    skill_registry: SkillRegistry,
    // RwLock erlaubt das Ändern des Keys zur Laufzeit bei gleichzeitiger Thread-Safety
    api_key: RwLock<String>, 
}

impl InferenceEngine {
    pub fn new() -> Result<Self> {
        // 1. Priorität: Environment Variable
        let mut key = env::var("GROQ_API_KEY").unwrap_or_default();

        // 2. Priorität: Config File im Workspace
        if key.is_empty() {
            if Path::new(CONFIG_FILE).exists() {
                if let Ok(content) = std::fs::read_to_string(CONFIG_FILE) {
                    if let Ok(config) = serde_json::from_str::<VgtConfig>(&content) {
                        key = config.api_key;
                    }
                }
            }
        }
        
        // Wir returnen OK auch wenn Key leer ist -> App startet im "Setup Mode"
        Ok(Self {
            client: Client::new(),
            model_registry: ModelRegistry::new(),
            skill_registry: SkillRegistry::new(),
            api_key: RwLock::new(key),
        })
    }

    pub fn is_configured(&self) -> bool {
        let key = self.api_key.read().unwrap();
        !key.is_empty() && key.starts_with("gsk_")
    }

    pub fn set_api_key(&self, new_key: String) -> Result<()> {
        // 1. Update In-Memory
        {
            let mut w = self.api_key.write().unwrap();
            *w = new_key.clone();
        }

        // 2. Persist to Disk (Workspace)
        // Ensure workspace exists
        if let Some(parent) = Path::new(CONFIG_FILE).parent() {
            std::fs::create_dir_all(parent)?;
        }
        
        let config = VgtConfig { api_key: new_key };
        let json = serde_json::to_string_pretty(&config)?;
        std::fs::write(CONFIG_FILE, json)?;

        Ok(())
    }

    pub fn get_model_registry(&self) -> &ModelRegistry {
        &self.model_registry
    }
    
    pub fn get_skill_registry(&self) -> &SkillRegistry {
        &self.skill_registry
    }

    pub fn get_skill_registry_mut(&mut self) -> &mut SkillRegistry {
        &mut self.skill_registry
    }

    pub async fn process_stream(
        &self, 
        req: InferenceRequest
    ) -> Result<Pin<Box<dyn Stream<Item = Result<String, anyhow::Error>> + Send>>> {
        
        // CHECK CONFIG
        let current_key = self.api_key.read().unwrap().clone();
        if current_key.is_empty() {
            anyhow::bail!("SYSTEM_HALT: API Key not configured. Initiation required.");
        }

        let model = self.model_registry.get(&req.model_id)
            .context("Modell nicht gefunden")?;

        let mut messages = Vec::new();
        if let Some(sys) = &req.system_prompt {
            messages.push(GroqMessage { role: "system".to_string(), content: sys.clone() });
        }

        if let Some(history) = req.messages {
            for msg in history {
                let role = msg.get("role").and_then(|v| v.as_str()).unwrap_or("user").to_string();
                let content = msg.get("content").and_then(|v| v.as_str()).unwrap_or("").to_string();
                messages.push(GroqMessage { role, content });
            }
        } else {
            messages.push(GroqMessage { role: "user".to_string(), content: req.prompt.clone() });
        }

        let tools = if req.use_tools {
            Some(self.skill_registry.to_tool_definitions())
        } else {
            None
        };
        
        let tool_choice = if tools.is_some() { Some("auto".to_string()) } else { None };

        let payload = GroqPayload {
            model: model.id.clone(),
            messages,
            temperature: req.temperature,
            stream: true,
            tools,
            tool_choice,
        };

        let request = self.client.post(GROQ_API_URL)
            .header("Authorization", format!("Bearer {}", current_key))
            .header("Content-Type", "application/json")
            .json(&payload);

        let response = request.send().await.context("Failed to connect to Groq API")?;

        if !response.status().is_success() {
            let err_text = response.text().await?;
            anyhow::bail!("Groq API Error: {}", err_text);
        }

        let stream = async_stream::try_stream! {
            let mut byte_stream = response.bytes_stream();

            while let Some(chunk_result) = byte_stream.next().await {
                let chunk = chunk_result.context("Error reading byte stream")?;
                let s = String::from_utf8_lossy(&chunk);

                for line in s.lines() {
                    let line = line.trim();
                    if line.is_empty() { continue; }
                    if line == "data: [DONE]" { break; }

                    if let Some(json_str) = line.strip_prefix("data: ") {
                        if let Ok(parsed) = serde_json::from_str::<GroqChunk>(json_str) {
                            if let Some(choice) = parsed.choices.first() {
                                if let Some(content) = &choice.delta.content {
                                    yield content.clone();
                                }
                                if let Some(tool_calls) = &choice.delta.tool_calls {
                                    let json_fragment = serde_json::to_string(&tool_calls)?;
                                    yield format!("[[TOOL_DELTA]]:{}", json_fragment);
                                }
                                if let Some(reason) = &choice.finish_reason {
                                    if reason == "tool_calls" {
                                        yield "[[TOOL_COMMIT]]".to_string();
                                    }
                                }
                            }
                        }
                    }
                }
            }
        };

        Ok(Box::pin(stream))
    }
}