use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Definiert die Qualitätsstufe eines Modells nach VGT Standards
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
pub enum VgtModelTier {
    Diamond, // GPT OSS 120B
    Platinum, // GPT OSS 20B Safeguard
    Gold,    // Llama 8B
    Legacy,  // Teure US-Modelle
}

/// Repräsentiert ein einzelnes KI-Modell mit allen Metadaten
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AIModel {
    pub id: String,
    pub name: String,
    pub provider: String, // "Groq", "Local", "OpenAI"
    pub tier: VgtModelTier,
    pub context_window: u32,
    pub tps_avg: u32,       // Tokens per Second
    pub cost_input_1m: f64, // Preis in USD
    pub cost_output_1m: f64,
}

/// Das zentrale Register für alle verfügbaren Modelle
pub struct ModelRegistry {
    models: HashMap<String, AIModel>,
}

impl ModelRegistry {
    /// Erstellt das Register mit den VGT-Standard-Modellen
    pub fn new() -> Self {
        let mut models = HashMap::new();

        // 1. DIAMANT STATUS: GPT OSS 120B (High Reasoning)
        models.insert(
            "gpt-oss-120b".to_string(),
            AIModel {
                id: "openai/gpt-oss-120b".to_string(), // Mapping auf Groq ID
                name: "GPT OSS 120B (VGT Logic Core)".to_string(),
                provider: "Groq".to_string(),
                tier: VgtModelTier::Diamond,
                context_window: 128_000,
                tps_avg: 500,
                cost_input_1m: 0.15,
                cost_output_1m: 0.60,
            },
        );

        // 2. PLATIN STATUS: GPT OSS Safeguard 20B (Speed & Safety)
        models.insert(
            "gpt-oss-20b".to_string(),
            AIModel {
                id: "openai/gpt-oss-safeguard-20b".to_string(), // Mapping auf effizientes MoE
                name: "GPT OSS Safeguard 20B".to_string(),
                provider: "Groq".to_string(),
                tier: VgtModelTier::Platinum,
                context_window: 32_768,
                tps_avg: 1000,
                cost_input_1m: 0.075,
                cost_output_1m: 0.30,
            },
        );

        Self { models }
    }

    /// Holt ein Modell anhand der ID
    pub fn get(&self, id: &str) -> Option<&AIModel> {
        self.models.get(id)
    }

    /// Berechnet die geschätzten Kosten für eine Transaktion
    pub fn estimate_cost(&self, model_id: &str, input_tokens: u32, estimated_output: u32) -> f64 {
        if let Some(model) = self.get(model_id) {
            let input_cost = (input_tokens as f64 / 1_000_000.0) * model.cost_input_1m;
            let output_cost = (estimated_output as f64 / 1_000_000.0) * model.cost_output_1m;
            return input_cost + output_cost;
        }
        0.0
    }
}