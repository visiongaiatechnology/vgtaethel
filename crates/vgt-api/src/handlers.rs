use axum::{
    extract::State,
    response::{IntoResponse, Sse, sse::Event},
    Json,
};
use serde_json::{json, Value};
use serde::Deserialize;
use vgt_core::InferenceRequest;
use vgt_skills::guard::SecurityGuard; // Import Guard
use crate::state::AppState;
use tracing::{info, error, warn};
use futures_util::{Stream, StreamExt};

pub async fn health_check() -> impl IntoResponse {
    Json(json!({ "status": "VGT OMEGA ACTIVE", "mode": "STREAMING" }))
}

pub async fn list_models(
    State(state): State<AppState>,
) -> impl IntoResponse {
    let registry = state.engine.get_model_registry();
    let model_ids = vec!["gpt-oss-120b", "gpt-oss-20b"]; 
    let mut models = Vec::new();

    for id in model_ids {
        if let Some(m) = registry.get(id) {
            models.push(m);
        }
    }
    Json(json!({ "models": models }))
}

// UPDATE: Request enthält nun 'override_security' Flag
#[derive(Deserialize)]
pub struct ToolExecRequest {
    pub name: String,
    pub args: Value,
    #[serde(default)] // Default ist false
    pub override_security: bool,
}

pub async fn execute_tool(
    State(state): State<AppState>,
    Json(payload): Json<ToolExecRequest>,
) -> impl IntoResponse {
    info!("EXECUTE TOOL: {} (Override: {})", payload.name, payload.override_security);

    let registry = state.engine.get_skill_registry();
    
    if let Some(skill) = registry.get(&payload.name) {
        
        // 1. VGT SECURITY GUARD SCAN
        // Scan wird IMMER ausgeführt, egal ob Override true ist oder nicht (für Logging)
        let report = SecurityGuard::scan(&payload.name, &payload.args);

        // 2. BLOCKIERUNG (Wenn unsicher UND kein Override)
        if !report.is_safe && !payload.override_security {
            warn!("VGT SECURITY BLOCK: Tool {} blocked due to risk score {}", payload.name, report.risk_score);
            return Json(json!({ 
                "status": "security_intervention", 
                "risk_score": report.risk_score,
                "threats": report.threats,
                "message": "Security Guard hat Bedrohungen erkannt. Bestätigen Sie die Ausführung explizit (Override)."
            }));
        }

        // 3. AUSFÜHRUNG (Entweder Safe oder Override aktiv)
        if !report.is_safe && payload.override_security {
            warn!("VGT SECURITY OVERRIDE: Executing risky command by USER DECREE.");
        }

        match skill.execute(payload.args).await {
            Ok(result) => {
                info!("TOOL SUCCESS: {}", payload.name);
                Json(json!({ "status": "success", "result": result }))
            },
            Err(e) => {
                error!("TOOL FAILURE {}: {:?}", payload.name, e);
                Json(json!({ "status": "error", "error": e.to_string() }))
            }
        }
    } else {
        warn!("TOOL NOT FOUND: {}", payload.name);
        Json(json!({ "status": "error", "error": "Tool not found in registry" }))
    }
}

pub async fn chat_inference(
    State(state): State<AppState>,
    Json(payload): Json<InferenceRequest>,
) -> impl IntoResponse {
    info!("Stream Request für Modell: {}", payload.model_id);

    let core_stream_result = state.engine.process_stream(payload).await;

    match core_stream_result {
        Ok(stream) => {
            let sse_stream = stream.map(|result| {
                match result {
                    Ok(text_token) => Event::default().data(text_token),
                    Err(e) => {
                        error!("Stream Error: {:?}", e);
                        Event::default().event("error").data(e.to_string())
                    }
                }
            });

            Sse::new(sse_stream)
                .keep_alive(axum::response::sse::KeepAlive::default())
                .into_response()
        }
        Err(e) => {
            error!("Setup Failure: {:?}", e);
            Json(json!({ "error": e.to_string() })).into_response()
        }
    }
}