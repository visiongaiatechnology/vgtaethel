import { Send, Square } from "lucide-react";
import { useState, KeyboardEvent } from "react";

interface InputConsoleProps {
    onExecute: (prompt: string) => void;
    onStop: () => void;
    isStreaming: boolean;
    disabled?: boolean;
}

export function InputConsole({ onExecute, onStop, isStreaming, disabled }: InputConsoleProps) {
    const [input, setInput] = useState("");

    const handleExecute = () => {
        if (!input.trim() || isStreaming || disabled) return;
        onExecute(input);
        setInput("");
    };

    const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleExecute();
        }
    };

    return (
        <div className="p-4 border-t border-vgt-border bg-[#030014]/50 backdrop-blur-xl rounded-b-xl">
            <div className="flex gap-3 relative">
                <input 
                    type="text" 
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    disabled={disabled}
                    placeholder={disabled ? "INITIALIZING UPLINK..." : "Enter command directive..."}
                    className="flex-1 bg-white/5 border border-white/10 rounded px-4 py-3 outline-none text-white font-mono text-sm 
                        placeholder:text-vgt-text-dim focus:border-vgt-cyan/50 focus:bg-white/10 transition-all shadow-inner"
                    autoFocus
                />
                
                {isStreaming ? (
                        <button 
                        onClick={onStop}
                        className="px-6 py-2 bg-red-500/10 text-red-500 border border-red-500/50 rounded hover:bg-red-500/20 
                            transition-all flex items-center gap-2 font-mono text-xs font-bold tracking-wider shadow-[0_0_15px_rgba(239,68,68,0.2)]"
                        >
                        <Square size={14} fill="currentColor" /> ABORT
                        </button>
                ) : (
                    <button 
                        onClick={handleExecute}
                        disabled={!input.trim() || disabled}
                        className="px-6 py-2 bg-vgt-cyan/10 text-vgt-cyan border border-vgt-cyan/50 rounded hover:bg-vgt-cyan/20 hover:shadow-[0_0_15px_rgba(0,240,255,0.3)]
                            transition-all disabled:opacity-30 disabled:cursor-not-allowed flex items-center gap-2 font-mono text-xs font-bold tracking-wider group"
                    >
                        <Send size={14} className="group-hover:translate-x-1 transition-transform" /> EXECUTE
                    </button>
                )}
            </div>
            
            {/* Input Decoration */}
            <div className="absolute bottom-1 right-2 flex gap-1">
                 <div className="w-1 h-1 bg-vgt-text-dim rounded-full opacity-20"></div>
                 <div className="w-1 h-1 bg-vgt-text-dim rounded-full opacity-20"></div>
            </div>
        </div>
    );
}