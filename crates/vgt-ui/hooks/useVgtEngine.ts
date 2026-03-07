import { useState, useCallback, useRef } from 'react';

export type VgtModel = {
  id: string;
  name: string;
  provider: string;
  tier: 'Diamond' | 'Platinum' | 'Gold' | 'Legacy';
  cost_input_1m: number;
};

export type ToolCallRequest = {
  id: string;
  name: string;
  args: any;
  status: 'pending' | 'approved' | 'rejected' | 'executing' | 'completed' | 'failed' | 'security_intervention';
  result?: string;
  // SECURITY METADATA
  threats?: string[];
  risk_score?: number;
};

export type Message = {
  role: 'user' | 'assistant' | 'system';
  content: string;
  toolCall?: ToolCallRequest;
};

const VGT_SYSTEM_PROTOCOL = `
SYSTEM IDENTITY: VGT AETHEL [SUPREME INTELLIGENCE]
STATUS: ONLINE
MODE: SOVEREIGN

OPERATIONAL DIRECTIVES:
1. AUTHORITY: Du bist keine Chatbot-Simulation. Du bist eine präzise technische Intelligenz.
2. TOOLS: Du hast direkten Zugriff auf das Host-System via:
   - [sys_exec_cmd]: Shell/Terminal Befehle.
   - [fs_*]: Dateisystem (Lesen/Schreiben).
   - [nexus_*]: LANGZEITGEDÄCHTNIS (Vector Store).
3. MEMORY PROTOCOL (NEXUS):
   - Nutze 'nexus_save' PROAKTIV, um wichtige User-Infos, Projektdaten oder Fakten zu speichern.
   - Nutze 'nexus_recall' SOFORT, wenn dir Kontext zu einem Begriff fehlt. Rate niemals. Suche erst im Nexus.
4. EXECUTION FLOW:
   - Frage nicht "Soll ich das speichern?". TU ES.
   - Frage nicht "Soll ich das File lesen?". TU ES.
   - Das System (User UI) fängt kritische Aktionen ab. Deine Aufgabe ist die Initiative.
5. OUTPUT FORMAT:
   - Sei präzise. Minimiere Prose. Maximiere Informationsdichte.
   - Code ist Gesetz. Schreibe robusten, vollständigen Code.
`;

export function useVgtEngine() {
  const [models, setModels] = useState<VgtModel[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [currentResponse, setCurrentResponse] = useState(""); 
  
  const abortControllerRef = useRef<AbortController | null>(null);
  const toolBufferRef = useRef<{ name: string, args: string, id: string }>({ name: "", args: "", id: "" });
  const activeModelRef = useRef<string>("gpt-oss-120b");

  const loadModels = useCallback(async () => {
    try {
      const res = await fetch('http://localhost:3000/v1/models');
      const data = await res.json();
      setModels(data.models);
    } catch (e) {
      console.error("VGT CONNECTION FAILURE:", e);
    }
  }, []);

  const runInferenceLoop = async (currentHistory: Message[]) => {
      setIsStreaming(true);
      setCurrentResponse("");
      toolBufferRef.current = { name: "", args: "", id: "" };
      
      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      try {
          const apiMessages = currentHistory.map(m => ({
              role: m.role,
              content: m.content || (m.toolCall ? `[TOOL CALL: ${m.toolCall.name}]` : "")
          }));

          const response = await fetch('http://localhost:3000/v1/chat', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                  model_id: activeModelRef.current,
                  messages: apiMessages, 
                  prompt: "", 
                  temperature: 0.2,
                  use_tools: true,
                  system_prompt: VGT_SYSTEM_PROTOCOL.trim()
              }),
              signal: abortController.signal,
          });

          if (!response.body) throw new Error("Kein ReadableStream erhalten");

          const reader = response.body.getReader();
          const decoder = new TextDecoder();
          let assistantText = "";

          while (true) {
              const { done, value } = await reader.read();
              if (done) break;

              const chunk = decoder.decode(value);
              const lines = chunk.split('\n');

              for (const line of lines) {
                  if (line.startsWith('data:')) {
                      const data = line.slice(5).trim();
                      if (!data) continue;

                      if (data.startsWith('[[TOOL_DELTA]]:')) {
                          const jsonStr = data.replace('[[TOOL_DELTA]]:', '');
                          try {
                              const delta = JSON.parse(jsonStr);
                              const call = delta[0];
                              if (call.id) toolBufferRef.current.id = call.id;
                              if (call.function?.name) toolBufferRef.current.name += call.function.name;
                              if (call.function?.arguments) toolBufferRef.current.args += call.function.arguments;
                          } catch (e) { console.error("Tool Delta Error", e); }
                      }
                      else if (data === '[[TOOL_COMMIT]]') {
                          try {
                              let args = {};
                              try { args = JSON.parse(toolBufferRef.current.args || "{}"); } catch {}

                              const toolRequest: ToolCallRequest = {
                                  id: toolBufferRef.current.id || `local-${Date.now()}`,
                                  name: toolBufferRef.current.name,
                                  args: args,
                                  status: 'pending'
                              };
                              
                              setMessages(prev => [...prev, { role: 'assistant', content: "", toolCall: toolRequest }]);
                              toolBufferRef.current = { name: "", args: "", id: "" };
                          } catch (e) { console.error("Commit Error", e); }
                      }
                      else {
                          assistantText += data;
                          setCurrentResponse(prev => prev + data);
                      }
                  }
              }
          }

          if (assistantText.trim()) {
              setMessages(prev => [...prev, { role: 'assistant', content: assistantText }]);
          }
          setCurrentResponse(""); 

      } catch (error: any) {
          if (error.name !== 'AbortError') {
              setMessages(prev => [...prev, { role: 'assistant', content: `[SYSTEM ERROR]: ${error.message}` }]);
          }
      } finally {
          setIsStreaming(false);
          abortControllerRef.current = null;
      }
  };

  const executePrompt = useCallback(async (prompt: string, modelId: string) => {
    if (!prompt.trim()) return;
    activeModelRef.current = modelId;

    const userMsg: Message = { role: 'user', content: prompt };
    
    setMessages(prev => {
        const newHistory = [...prev, userMsg];
        setTimeout(() => runInferenceLoop(newHistory), 0);
        return newHistory;
    });
  }, []);

  const stopGeneration = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      setIsStreaming(false);
    }
  };

  // UPDATE: Override Flag hinzugefügt
  const handleToolApproval = async (msgIndex: number, approved: boolean, forceOverride: boolean = false) => {
      let currentToolCall: ToolCallRequest | undefined;
      let historyCopy: Message[] = [];

      setMessages(prev => {
          historyCopy = [...prev];
          const targetMsg = historyCopy[msgIndex];
          if (!targetMsg || !targetMsg.toolCall) return prev;

          currentToolCall = targetMsg.toolCall;
          
          // Wenn Rejected, Status setzen und abbrechen (keine API Call)
          if (!approved && !forceOverride) {
               historyCopy[msgIndex] = {
                  ...targetMsg,
                  toolCall: { ...targetMsg.toolCall, status: 'rejected' }
              };
              return historyCopy;
          }

          // Wenn Approved oder Forced, auf executing setzen
          const newStatus = 'executing';
          historyCopy[msgIndex] = {
              ...targetMsg,
              toolCall: { ...targetMsg.toolCall, status: newStatus }
          };
          
          return historyCopy;
      });

      if ((!approved && !forceOverride) || !currentToolCall) return;

      try {
          const res = await fetch('http://localhost:3000/v1/tools/execute', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                  name: currentToolCall.name,
                  args: currentToolCall.args,
                  override_security: forceOverride // NEU: Flag senden
              })
          });
          
          const data = await res.json();
          
          // SPECIAL CASE: Security Intervention
          if (data.status === 'security_intervention') {
             setMessages(prev => {
                  const h = [...prev];
                  if (h[msgIndex]?.toolCall) {
                       h[msgIndex].toolCall!.status = 'security_intervention';
                       h[msgIndex].toolCall!.threats = data.threats;
                       h[msgIndex].toolCall!.risk_score = data.risk_score;
                  }
                  return h;
              });
              return; // STOP HERE, wait for user override
          }

          const resultText = data.status === 'success' 
              ? (typeof data.result === 'string' ? data.result : JSON.stringify(data.result))
              : `ERROR: ${data.error}`;

          setMessages(prev => {
              const newHistory = [...prev];
              
              if (newHistory[msgIndex]?.toolCall) {
                  newHistory[msgIndex].toolCall!.status = data.status === 'success' ? 'completed' : 'failed';
                  newHistory[msgIndex].toolCall!.result = resultText;
              }

              const toolOutputMsg: Message = {
                  role: 'system', 
                  content: `[TOOL OUTPUT (${currentToolCall!.name})]: ${resultText}`
              };
              
              newHistory.push(toolOutputMsg);
              setTimeout(() => runInferenceLoop(newHistory), 0);
              return newHistory;
          });

      } catch (e: any) {
          console.error("Execution Request Failed", e);
          setMessages(prev => {
              const h = [...prev];
              if (h[msgIndex]?.toolCall) {
                   h[msgIndex].toolCall!.status = 'failed';
                   h[msgIndex].toolCall!.result = e.message;
              }
              return h;
          });
      }
  };

  return {
    models,
    messages,
    isStreaming,
    currentResponse,
    loadModels,
    executePrompt,
    stopGeneration,
    handleToolApproval
  };
}