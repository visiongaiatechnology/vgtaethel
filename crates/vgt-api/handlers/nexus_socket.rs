use axum::{
    extract::ws::{Message, WebSocket, WebSocketUpgrade},
    extract::State,
    response::IntoResponse,
};
use futures::{sink::SinkExt, stream::StreamExt};
use std::sync::Arc;
use vgt_core::nexus::{NexusHub, NexusMessage, NexusResponse, ChannelType};
use tokio::sync::mpsc;

// State Wrapper muss in main.rs definiert sein, hier abstrahiert
pub struct AppState {
    pub nexus: Arc<NexusHub>,
}

pub async fn nexus_ws_handler(
    ws: WebSocketUpgrade,
    State(state): State<Arc<AppState>>,
) -> impl IntoResponse {
    ws.on_upgrade(|socket| handle_bridge_socket(socket, state))
}

async fn handle_bridge_socket(socket: WebSocket, state: Arc<AppState>) {
    let (mut sender, mut receiver) = socket.split();
    
    // Channel für ausgehende Nachrichten vom Core -> Bridge
    let (tx, mut rx) = mpsc::unbounded_channel::<NexusResponse>();

    // Wir spawnen einen Task, der Nachrichten vom Core an den WebSocket sendet
    let mut send_task = tokio::spawn(async move {
        while let Some(msg) = rx.recv().await {
            if let Ok(json) = serde_json::to_string(&msg) {
                if sender.send(Message::Text(json)).await.is_err() {
                    break;
                }
            }
        }
    });

    // Loop für eingehende Nachrichten von der Bridge -> Core
    while let Some(Ok(msg)) = receiver.next().await {
        if let Message::Text(text) = msg {
            // Handshake oder Message?
            // Annahme: Message Format { "type": "register", "channel": "WhatsApp" } oder Payload
            
            // Simplifizierter Flow: Wir parsen direkt NexusMessage
            if let Ok(nexus_msg) = serde_json::from_str::<NexusMessage>(&text) {
                // Registrierung des Output Channels beim ersten Kontakt für diesen Channel
                // In Prod: Ein separater Handshake wäre sauberer. 
                // Hier: Wir registrieren den Sender für den Channel der eingehenden Nachricht
                state.nexus.register_output(nexus_msg.channel.clone(), tx.clone()).await;
                
                if let Err(e) = state.nexus.ingest_message(nexus_msg).await {
                    tracing::error!("Failed to ingest message: {}", e);
                }
            } else {
                // Check if it's a handshake
                 #[derive(serde::Deserialize)]
                 struct Handshake { command: String, channels: Vec<ChannelType> }
                 if let Ok(hs) = serde_json::from_str::<Handshake>(&text) {
                     if hs.command == "REGISTER" {
                         for ch in hs.channels {
                             state.nexus.register_output(ch, tx.clone()).await;
                         }
                         tracing::info!("Bridge registered channels via Handshake");
                     }
                 }
            }
        }
    }

    send_task.abort();
}