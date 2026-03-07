mod state;
mod handlers;

use axum::{
    routing::{get, post},
    Router,
    http::Method,
};
use std::net::SocketAddr;
use tower_http::cors::{CorsLayer, Any};
use tower_http::trace::TraceLayer;
use tracing::info;
use crate::state::AppState;
use anyhow::Context;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    vgt_core::init_system();
    info!("VGT API GATEWAY: Bootvorgang (Streaming Mode)...");

    // Init AppState (startet Engine, ggf. unkonfiguriert)
    let state = AppState::new()
        .context("Konnte AppState nicht initialisieren.")?;

    let cors = CorsLayer::new()
        .allow_origin(Any) // Im Container Context vereinfacht, für Production spezifizieren
        .allow_methods([Method::GET, Method::POST])
        .allow_headers(Any);

    let app = Router::new()
        .route("/health", get(handlers::health_check))
        .route("/v1/setup", post(handlers::setup_system)) // NEU: Setup Route
        .route("/v1/models", get(handlers::list_models))
        .route("/v1/chat", post(handlers::chat_inference))
        .route("/v1/tools/execute", post(handlers::execute_tool))
        .layer(TraceLayer::new_for_http())
        .layer(cors)
        .with_state(state);

    let addr = SocketAddr::from(([0, 0, 0, 0], 3000)); // 0.0.0.0 für Docker wichtig
    info!("VGT STREAMING SERVER online unter http://{}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}