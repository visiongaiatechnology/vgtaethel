import { useState } from "react";
import { Key, ShieldCheck, ChevronRight, Cpu } from "lucide-react";

interface SetupWizardProps {
    onComplete: () => void;
}

export function SetupWizard({ onComplete }: SetupWizardProps) {
    const [apiKey, setApiKey] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    const handleSetup = async () => {
        if (!apiKey.startsWith("gsk_")) {
            setError("Ungültiges Format. VGT erfordert einen Groq API Key (gsk_...).");
            return;
        }

        setLoading(true);
        try {
            const res = await fetch('http://localhost:3000/v1/setup', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ api_key: apiKey })
            });
            const data = await res.json();

            if (data.status === 'success') {
                // Künstlicher Delay für "System Boot" Effekt
                setTimeout(() => {
                    onComplete();
                }, 1500);
            } else {
                setError(data.message);
                setLoading(false);
            }
        } catch (e) {
            setError("Verbindung zum Neural Core fehlgeschlagen.");
            setLoading(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#030014]">
            <div className="absolute inset-0 bg-grid-pattern opacity-20" />
            
            <div className="relative z-10 w-full max-w-md p-8 vgt-glass rounded-2xl border border-vgt-cyan/30 shadow-[0_0_50px_rgba(0,240,255,0.1)]">
                <div className="flex justify-center mb-8">
                     <div className="w-16 h-16 rounded-full bg-vgt-cyan/10 border border-vgt-cyan/50 flex items-center justify-center animate-pulse">
                        <Cpu size={32} className="text-vgt-cyan" />
                     </div>
                </div>

                <div className="text-center mb-8">
                    <h1 className="text-2xl font-bold font-mono text-transparent bg-clip-text bg-gradient-to-r from-white via-vgt-cyan to-white tracking-widest mb-2">
                        VGT GENESIS
                    </h1>
                    <p className="text-xs text-vgt-text-dim font-mono uppercase tracking-[0.2em]">
                        Initial System Configuration
                    </p>
                </div>

                <div className="space-y-6">
                    <div className="space-y-2">
                        <label className="text-xs font-mono text-vgt-cyan flex items-center gap-2">
                            <Key size={12} /> NEURAL UPLINK KEY (GROQ)
                        </label>
                        <input 
                            type="password"
                            value={apiKey}
                            onChange={(e) => { setApiKey(e.target.value); setError(""); }}
                            placeholder="gsk_..."
                            className="w-full bg-black/50 border border-white/10 rounded px-4 py-3 text-white font-mono text-sm outline-none focus:border-vgt-cyan/50 focus:shadow-[0_0_15px_rgba(0,240,255,0.1)] transition-all"
                        />
                        {error && (
                            <div className="text-[10px] text-red-500 font-mono flex items-center gap-2 mt-2">
                                <span className="w-1.5 h-1.5 bg-red-500 rounded-full animate-pulse"/>
                                {error}
                            </div>
                        )}
                    </div>

                    <button 
                        onClick={handleSetup}
                        disabled={loading || !apiKey}
                        className="w-full group relative overflow-hidden rounded bg-vgt-cyan text-black font-bold font-mono text-sm py-3 transition-all hover:shadow-[0_0_20px_rgba(0,240,255,0.4)] disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        <span className="relative z-10 flex items-center justify-center gap-2">
                            {loading ? "INITIALIZING..." : "INITIATE SYSTEM"} 
                            {!loading && <ChevronRight size={16} />}
                        </span>
                        {/* Scanline Effect */}
                        <div className="absolute inset-0 bg-white/30 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-500 z-0" />
                    </button>
                    
                    <div className="text-center mt-6">
                         <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-green-500/5 border border-green-500/20 text-[10px] text-green-400 font-mono">
                            <ShieldCheck size={10} />
                            LOCAL ENCRYPTION ACTIVE
                        </div>
                        <p className="text-[10px] text-vgt-text-dim mt-2 opacity-50">
                            Key is stored locally in ./vgt_workspace/vgt_config.json
                        </p>
                    </div>
                </div>
            </div>
        </div>
    );
}