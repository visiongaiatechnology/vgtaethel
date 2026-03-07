use std::sync::Arc;
use vgt_core::InferenceEngine;
use vgt_skills::{
    SkillRegistry,
    shell::ExecuteCommandSkill,
    fs::{ReadFileSkill, WriteFileSkill},
    rag::{NexusStore, MemorySaveSkill, MemorySearchSkill}
};
use anyhow::{Result, Context};
use tracing::info;

/// Der geteilte Zustand der gesamten Applikation.
#[derive(Clone)]
pub struct AppState {
    pub engine: Arc<InferenceEngine>,
    // Nexus Store könnte hier auch exposed werden, 
    // ist aber via Skills gekapselt.
}

impl AppState {
    /// Initialisiert den App-State, die Inferenz-Engine und lädt Skills.
    pub async fn new() -> Result<Self> {
        // 1. Engine starten
        let mut engine = InferenceEngine::new()?;
        
        // 2. NEXUS RAG SYSTEM INITIALISIEREN
        info!("VGT NEXUS: Initialisiere Neural Memory...");
        let nexus_store = Arc::new(NexusStore::new()
            .context("Fehler beim Starten des Nexus Vector Stores")?);
        info!("VGT NEXUS: Online.");

        // 3. Skills registrieren (Waffenkammer füllen)
        {
            let registry = engine.get_skill_registry_mut();
            
            // SYSTEM (Critical)
            registry.register(Box::new(ExecuteCommandSkill));
            
            // FILESYSTEM (Mixed Risk)
            registry.register(Box::new(ReadFileSkill));
            registry.register(Box::new(WriteFileSkill));

            // RAG / MEMORY (Safe)
            registry.register(Box::new(MemorySaveSkill::new(nexus_store.clone())));
            registry.register(Box::new(MemorySearchSkill::new(nexus_store.clone())));
        }

        Ok(Self {
            engine: Arc::new(engine),
        })
    }
}