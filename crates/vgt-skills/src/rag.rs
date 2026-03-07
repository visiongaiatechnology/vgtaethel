use crate::{VgtSkill, RiskLevel};
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::sync::{Arc, RwLock};
use std::path::PathBuf;
use anyhow::{Result, Context};
use fastembed::{TextEmbedding, InitOptions, EmbeddingModel};
use chrono::{DateTime, Utc};
use uuid::Uuid;

// VGT MEMORY CONFIG
const MEMORY_FILE: &str = "./vgt_workspace/nexus_memory.json";
const MAX_RESULTS: usize = 5;
const SIMILARITY_THRESHOLD: f32 = 0.75; // Strengerer Filter für Qualität

// --- DATA STRUCTURES ---

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct MemoryEntry {
    pub id: String,
    pub content: String,
    pub timestamp: DateTime<Utc>,
    // Vector wird nicht serialisiert um JSON lesbar zu halten? 
    // NEIN. Für Persistence brauchen wir ihn. 
    // Optimierung: In separater Binärdatei speichern wäre besser, 
    // aber für "Light" Version ist JSON okay.
    pub vector: Vec<f32>, 
    pub metadata: Value,
}

/// Der VGT Nexus Vector Store
/// Thread-Safe Wrapper um Embedding Model und Daten
pub struct NexusStore {
    embedding_model: TextEmbedding,
    entries: Arc<RwLock<Vec<MemoryEntry>>>,
}

impl NexusStore {
    pub fn new() -> Result<Self> {
        // 1. Initialisiere Embedding Model (Lokal, Quantisiert)
        let model = TextEmbedding::try_new(InitOptions {
            model_name: EmbeddingModel::AllMiniLML6V2, // Sehr schnell, gut für RAG
            show_download_progress: true,
            ..Default::default()
        })?;

        // 2. Lade existierende Erinnerungen
        let entries = Self::load_from_disk().unwrap_or_else(|_| Vec::new());

        Ok(Self {
            embedding_model: model,
            entries: Arc::new(RwLock::new(entries)),
        })
    }

    fn load_from_disk() -> Result<Vec<MemoryEntry>> {
        if !std::path::Path::new(MEMORY_FILE).exists() {
            return Ok(Vec::new());
        }
        let data = std::fs::read_to_string(MEMORY_FILE)?;
        let entries: Vec<MemoryEntry> = serde_json::from_str(&data)?;
        Ok(entries)
    }

    fn save_to_disk(&self) -> Result<()> {
        let entries = self.entries.read().unwrap();
        // Stelle sicher, dass Directory existiert
        if let Some(parent) = std::path::Path::new(MEMORY_FILE).parent() {
            std::fs::create_dir_all(parent)?;
        }
        let data = serde_json::to_string_pretty(&*entries)?;
        std::fs::write(MEMORY_FILE, data)?;
        Ok(())
    }

    /// Generiert Embeddings und speichert Eintrag
    pub fn add(&self, content: String, metadata: Value) -> Result<String> {
        // Embeddings generieren (Batch size 1)
        let documents = vec![content.clone()];
        let embeddings = self.embedding_model.embed(documents, None)?;
        let vector = embeddings.first().context("Kein Embedding generiert")?.clone();

        let id = Uuid::new_v4().to_string();
        let entry = MemoryEntry {
            id: id.clone(),
            content,
            timestamp: Utc::now(),
            vector,
            metadata,
        };

        {
            let mut write_guard = self.entries.write().unwrap();
            write_guard.push(entry);
        } // Mutex freigeben

        // Async persistieren (hier synchron der Einfachheit halber)
        self.save_to_disk()?;
        
        Ok(id)
    }

    /// Semantische Suche (Cosine Similarity)
    pub fn search(&self, query: &str) -> Result<Vec<(MemoryEntry, f32)>> {
        let query_vec = self.embedding_model.embed(vec![query.to_string()], None)?
            .first()
            .context("Query Embedding Failed")?
            .clone();

        let read_guard = self.entries.read().unwrap();
        
        let mut results: Vec<(MemoryEntry, f32)> = read_guard.iter().map(|entry| {
            let score = cosine_similarity(&query_vec, &entry.vector);
            (entry.clone(), score)
        })
        .filter(|(_, score)| *score >= SIMILARITY_THRESHOLD)
        .collect();

        // Sortieren nach Score (Absteigend)
        results.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));
        
        Ok(results.into_iter().take(MAX_RESULTS).collect())
    }
}

/// Berechnet Cosine Similarity zwischen zwei Vektoren
/// Mathe-Kern: A . B / (|A| * |B|)
fn cosine_similarity(a: &[f32], b: &[f32]) -> f32 {
    let dot_product: f32 = a.iter().zip(b).map(|(x, y)| x * y).sum();
    let norm_a: f32 = a.iter().map(|x| x * x).sum::<f32>().sqrt();
    let norm_b: f32 = b.iter().map(|x| x * x).sum::<f32>().sqrt();
    
    if norm_a == 0.0 || norm_b == 0.0 {
        return 0.0;
    }
    
    dot_product / (norm_a * norm_b)
}


// --- SKILL: MEMORY SAVE ---

pub struct MemorySaveSkill {
    store: Arc<NexusStore>,
}

impl MemorySaveSkill {
    pub fn new(store: Arc<NexusStore>) -> Self {
        Self { store }
    }
}

#[derive(Deserialize)]
struct SaveArgs {
    content: String,
    category: Option<String>,
}

#[async_trait]
impl VgtSkill for MemorySaveSkill {
    fn name(&self) -> &str { "nexus_save" }
    
    fn description(&self) -> &str {
        "Speichert Information im Langzeitgedächtnis (Vektor-DB). Nutze dies für wichtige Fakten, User-Präferenzen oder Projekt-Details."
    }
    
    fn parameters(&self) -> Value {
        serde_json::json!({
            "type": "object",
            "properties": {
                "content": { "type": "string", "description": "Die Information, die gespeichert werden soll." },
                "category": { "type": "string", "description": "Optionaler Tag (z.B. 'user_bio', 'project_x')" }
            },
            "required": ["content"]
        })
    }
    
    fn risk_level(&self) -> RiskLevel { RiskLevel::Safe } // Speichern ist sicher

    async fn execute(&self, args: Value) -> Result<String> {
        let input: SaveArgs = serde_json::from_value(args)?;
        let meta = serde_json::json!({ "category": input.category.unwrap_or("general".into()) });
        
        let id = self.store.add(input.content, meta)?;
        Ok(format!("NEXUS: Information gespeichert. ID: {}", id))
    }
}


// --- SKILL: MEMORY SEARCH ---

pub struct MemorySearchSkill {
    store: Arc<NexusStore>,
}

impl MemorySearchSkill {
    pub fn new(store: Arc<NexusStore>) -> Self {
        Self { store }
    }
}

#[derive(Deserialize)]
struct SearchArgs {
    query: String,
}

#[async_trait]
impl VgtSkill for MemorySearchSkill {
    fn name(&self) -> &str { "nexus_recall" }
    
    fn description(&self) -> &str {
        "Durchsucht das Langzeitgedächtnis nach relevanten Informationen basierend auf einer semantischen Frage."
    }
    
    fn parameters(&self) -> Value {
        serde_json::json!({
            "type": "object",
            "properties": {
                "query": { "type": "string", "description": "Die Suchanfrage (z.B. 'Was weißt du über Projekt Alpha?')" }
            },
            "required": ["query"]
        })
    }
    
    fn risk_level(&self) -> RiskLevel { RiskLevel::Safe }

    async fn execute(&self, args: Value) -> Result<String> {
        let input: SearchArgs = serde_json::from_value(args)?;
        let results = self.store.search(&input.query)?;
        
        if results.is_empty() {
            return Ok("NEXUS: Keine relevanten Einträge gefunden.".to_string());
        }

        let output: Vec<String> = results.iter().map(|(entry, score)| {
            format!("[Score: {:.2}] {} (Datum: {})", score, entry.content, entry.timestamp.format("%Y-%m-%d"))
        }).collect();

        Ok(output.join("\n\n"))
    }
}