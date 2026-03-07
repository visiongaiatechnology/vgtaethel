use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use tokio::sync::mpsc;
use uuid::Uuid;
use chrono::{DateTime, Utc};
use std::sync::Arc;
use tokio::sync::RwLock;

// --- DATENSTRUKTUREN ---

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum ChannelType {
    WhatsApp,
    Telegram,
    Discord,
    Signal,
    BlueBubbles,
    Teams,
    Matrix,
    WebTerminal, // Default Internal
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NexusMessage {
    pub id: Uuid,
    pub channel: ChannelType,
    pub source_id: String, // z.B. Telefonnummer oder Discord User ID
    pub group_id: Option<String>, // Falls Gruppenchat
    pub author_name: Option<String>,
    pub content: String,
    pub media_url: Option<String>,
    pub timestamp: DateTime<Utc>,
    pub metadata: HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NexusResponse {
    pub target_channel: ChannelType,
    pub target_id: String, // Antwort an User/Gruppe
    pub content: String,
    pub attachments: Vec<String>,
}

// --- NEXUS HUB ---

/// Der Hub verwaltet alle eingehenden Verbindungen von der VGT Bridge
pub struct NexusHub {
    // Sender um Nachrichten IN die Engine zu leiten
    engine_tx: mpsc::UnboundedSender<NexusMessage>,
    // Map von Channel-Typ zu Sender (um Antworten an die Bridge zu schicken)
    outbound_channels: Arc<RwLock<HashMap<ChannelType, mpsc::UnboundedSender<NexusResponse>>>>,
}

impl NexusHub {
    pub fn new(engine_tx: mpsc::UnboundedSender<NexusMessage>) -> Self {
        Self {
            engine_tx,
            outbound_channels: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Registriert einen Ausgabekanal (wird von der API/Websocket Verbindung aufgerufen)
    pub async fn register_output(&self, channel: ChannelType, tx: mpsc::UnboundedSender<NexusResponse>) {
        let mut map = self.outbound_channels.write().await;
        map.insert(channel, tx);
        tracing::info!("NEXUS: Output channel registered: {:?}", channel);
    }

    /// Empfängt eine Nachricht von Extern (Bridge)
    pub async fn ingest_message(&self, msg: NexusMessage) -> Result<(), String> {
        tracing::info!("NEXUS: Ingesting message from {:?} // Author: {:?}", msg.channel, msg.source_id);
        
        // Hier könnte Middleware laufen (Spam-Check, Blacklist, etc.)
        
        // Weiterleitung an die AI Engine
        self.engine_tx.send(msg).map_err(|e| format!("Engine channel closed: {}", e))?;
        Ok(())
    }

    /// Sendet eine Antwort zurück an die Bridge
    pub async fn dispatch_response(&self, response: NexusResponse) -> Result<(), String> {
        let map = self.outbound_channels.read().await;
        
        if let Some(tx) = map.get(&response.target_channel) {
            tx.send(response).map_err(|e| format!("Outbound channel closed: {}", e))?;
            Ok(())
        } else {
            tracing::warn!("NEXUS: No outbound connection for channel {:?}", response.target_channel);
            Err("Channel not connected".to_string())
        }
    }
}