'use client';

import { useEffect, useState } from "react";
import { useVgtEngine } from "@/hooks/useVgtEngine";

import { SystemStatus } from "@/components/dashboard/SystemStatus";
import { ModelSelector } from "@/components/dashboard/ModelSelector";
import { TerminalOutput } from "@/components/chat/TerminalOutput";
import { InputConsole } from "@/components/chat/InputConsole";
import { SetupWizard } from "@/components/onboarding/SetupWizard"; // Import Wizard

export default function Home() {
  const { 
    models, 
    messages, 
    isStreaming, 
    currentResponse, 
    loadModels, 
    executePrompt, 
    stopGeneration,
    handleToolApproval
  } = useVgtEngine();

  const [activeModel, setActiveModel] = useState<string>("gpt-oss-120b");
  const [systemStatus, setSystemStatus] = useState<'LOADING' | 'READY' | 'SETUP_REQUIRED'>('LOADING');

  // Check Health & Config Status on Mount
  useEffect(() => {
    const checkSystem = async () => {
        try {
            const res = await fetch('http://localhost:3000/health');
            const data = await res.json();
            
            if (data.status === "SETUP_REQUIRED") {
                setSystemStatus('SETUP_REQUIRED');
            } else {
                setSystemStatus('READY');
                loadModels(); // Load models only if ready
            }
        } catch (e) {
            console.error("System Check Failed", e);
            // Retry logic could be added here
        }
    };
    checkSystem();
  }, [loadModels]);

  const handleExecution = (prompt: string) => {
    executePrompt(prompt, activeModel);
  };

  if (systemStatus === 'LOADING') {
      return <div className="min-h-screen flex items-center justify-center text-vgt-cyan font-mono animate-pulse">SYSTEM BOOT SEQUENCE...</div>;
  }

  if (systemStatus === 'SETUP_REQUIRED') {
      return <SetupWizard onComplete={() => {
          setSystemStatus('READY');
          loadModels();
      }} />;
  }

  return (
    <div className="w-full max-w-7xl flex flex-col gap-6 h-[90vh]">
      <header className="w-full vgt-glass rounded-xl p-6 flex justify-between items-center shrink-0 border-b border-white/5 relative overflow-hidden">
        <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-vgt-cyan/50 to-transparent opacity-50"></div>
        
        <div className="z-10">
          <h1 className="text-3xl font-bold tracking-widest font-mono text-transparent bg-clip-text bg-gradient-to-r from-white via-vgt-cyan to-white drop-shadow-[0_0_10px_rgba(0,240,255,0.3)]">
            VGT <span className="font-light text-white">AETHEL</span>
          </h1>
          <p className="text-[10px] text-vgt-text-dim font-mono mt-1 tracking-[0.2em] uppercase">
            Sovereign Intelligence Interface :: <span className="text-green-400">Online</span>
          </p>
        </div>
        
        <SystemStatus 
          modelCount={models.length} 
          isStreaming={isStreaming} 
        />
      </header>

      <div className="grid grid-cols-1 md:grid-cols-12 gap-6 flex-1 min-h-0">
        <ModelSelector 
          models={models} 
          activeModelId={activeModel} 
          onSelectModel={setActiveModel} 
        />

        <div className="md:col-span-9 vgt-glass rounded-xl p-0 flex flex-col overflow-hidden relative shadow-2xl border-vgt-border/50">
            <TerminalOutput 
                messages={messages}
                isStreaming={isStreaming}
                currentResponse={currentResponse}
                onToolAction={handleToolApproval}
            />

            <InputConsole 
                onExecute={handleExecution}
                onStop={stopGeneration}
                isStreaming={isStreaming}
                disabled={models.length === 0}
            />
        </div>
      </div>
    </div>
  );
}