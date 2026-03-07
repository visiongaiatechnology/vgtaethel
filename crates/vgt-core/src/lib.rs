// VGT-AETHEL CORE LIBRARY
// Status: PLATINUM
// Architektur: Modular Export

// Das Modell-Register (Preise, Specs)
pub mod models;

// Die Inferenz-Maschine (Groq Connector)
pub mod engine;

// Re-Export für einfacheren Zugriff in anderen Crates
pub use models::{ModelRegistry, AIModel, VgtModelTier};
pub use engine::{InferenceEngine, InferenceRequest, InferenceResponse};

/// Initialisiert das Core-System (Environment, Logging)
pub fn init_system() {
    // Lädt .env Datei falls vorhanden
    let _ = dotenvy::dotenv();
    
    // Initialisiert Tracing Subscriber für strukturiertes Logging
    tracing_subscriber::fmt::init();
}