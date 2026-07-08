import { ShieldAlert, Check, X, Terminal, FileCode, AlertTriangle, Skull } from "lucide-react";
import { ToolCallRequest } from "@/hooks/useVgtEngine";

interface ToolRequestCardProps {
    tool: ToolCallRequest;
    onApprove: (override?: boolean) => void;
    onReject: () => void;
}

export function ToolRequestCard({ tool, onApprove, onReject }: ToolRequestCardProps) {
    const isCritical = tool.name === 'sys_exec_cmd' || tool.name === 'fs_write_file';
    const isSecurityIntervention = tool.status === 'security_intervention';
    
    // 1. DONE STATES
    if (tool.status === 'approved' || tool.status === 'executing' || tool.status === 'completed') {
        return (
            <div className="border border-green-500/30 bg-green-500/5 rounded p-3 flex items-center gap-3 text-xs font-mono">
                <div className="p-1 bg-green-500/20 rounded-full text-green-500"><Check size={12}/></div>
                <span className="text-green-400 tracking-wider">ACCESS GRANTED :: EXECUTING SEQUENCE</span>
            </div>
        );
    }
    
    if (tool.status === 'rejected') {
        return (
             <div className="border border-red-500/30 bg-red-500/5 rounded p-3 flex items-center gap-3 text-xs font-mono opacity-60">
                <div className="p-1 bg-red-500/20 rounded-full text-red-500"><X size={12}/></div>
                <span className="text-red-400 tracking-wider">ACCESS DENIED :: INTERVENTION LOGGED</span>
            </div>
        );
    }

    if (tool.status === 'failed') {
        return (
             <div className="border border-red-500/30 bg-red-500/5 rounded p-3 flex items-center gap-3 text-xs font-mono">
                <div className="p-1 bg-red-500/20 rounded-full text-red-500"><AlertTriangle size={12}/></div>
                <span className="text-red-400 tracking-wider">EXECUTION FAILED :: {tool.result}</span>
            </div>
        );
    }

    // 2. SECURITY INTERVENTION STATE (THE RED CARD)
    if (isSecurityIntervention) {
        return (
            <div className="my-4 border border-red-500 bg-[#1a0000] rounded-lg overflow-hidden shadow-[0_0_30px_rgba(255,0,0,0.2)] max-w-2xl animate-in fade-in slide-in-from-bottom-2">
                <div className="p-3 flex items-center justify-between border-b border-red-500/50 bg-red-950/50">
                    <div className="flex items-center gap-2">
                        <Skull size={16} className="text-red-500 animate-pulse"/>
                        <span className="text-xs font-bold font-mono tracking-widest text-red-500 uppercase">
                            SECURITY LOCKDOWN ACTIVE
                        </span>
                    </div>
                    <div className="flex items-center gap-2">
                         <span className="text-[10px] text-red-400 font-mono">RISK SCORE:</span>
                         <span className="text-xs font-bold text-red-500 font-mono bg-red-500/10 px-2 py-0.5 rounded border border-red-500/30">
                            {tool.risk_score}/100
                         </span>
                    </div>
                </div>

                <div className="p-4 font-mono text-xs text-red-300">
                    <div className="mb-3 font-bold tracking-wider">DETECTED THREATS:</div>
                    <div className="space-y-2 mb-4">
                        {tool.threats?.map((threat, idx) => (
                            <div key={idx} className="flex gap-2 items-start bg-red-500/10 p-2 rounded border border-red-500/20">
                                <AlertTriangle size={12} className="mt-0.5 shrink-0"/>
                                <span>{threat}</span>
                            </div>
                        ))}
                    </div>
                    
                    <div className="text-[10px] opacity-70 border-t border-red-500/20 pt-2">
                        System has blocked this action to protect integrity. Override requires manual authorization.
                    </div>
                </div>

                <div className="p-3 bg-red-950/30 border-t border-red-500/30 flex justify-end gap-3">
                    <button 
                        onClick={onReject}
                        className="px-4 py-2 rounded text-xs font-bold font-mono tracking-wider border border-white/10 text-vgt-text-dim hover:bg-white/5 transition-all"
                    >
                        ABORT
                    </button>
                    <button 
                        onClick={() => onApprove(true)} // FORCE OVERRIDE
                        className="px-4 py-2 rounded text-xs font-bold font-mono tracking-wider bg-red-600 hover:bg-red-500 text-white shadow-[0_0_15px_rgba(220,38,38,0.5)] transition-all flex items-center gap-2 border border-red-400"
                    >
                        <ShieldAlert size={12} /> FORCE EXECUTE
                    </button>
                </div>
            </div>
        )
    }

    // 3. STANDARD PENDING STATE
    return (
        <div className="my-4 border border-vgt-cyan/30 bg-[#030014]/80 backdrop-blur-md rounded-lg overflow-hidden shadow-[0_0_20px_rgba(0,240,255,0.05)] max-w-2xl animate-in fade-in slide-in-from-bottom-2">
            <div className={`p-3 flex items-center justify-between border-b ${isCritical ? 'bg-red-950/30 border-red-500/30' : 'bg-vgt-cyan/5 border-vgt-cyan/20'}`}>
                <div className="flex items-center gap-2">
                    {isCritical ? <ShieldAlert size={16} className="text-red-500 animate-pulse"/> : <FileCode size={16} className="text-vgt-cyan"/>}
                    <span className={`text-xs font-bold font-mono tracking-widest ${isCritical ? 'text-red-400' : 'text-vgt-cyan'}`}>
                        {isCritical ? 'CRITICAL SYSTEM REQUEST' : 'STANDARD OPERATION'}
                    </span>
                </div>
                <span className="text-[10px] text-vgt-text-dim font-mono">{tool.name}</span>
            </div>

            <div className="p-4 font-mono text-xs">
                <div className="text-vgt-text-dim mb-2 uppercase tracking-wider text-[10px]">Payload Parameters</div>
                <div className="bg-black/50 p-3 rounded border border-white/5 text-gray-300 overflow-x-auto custom-scrollbar">
                    <pre>{JSON.stringify(tool.args, null, 2)}</pre>
                </div>
            </div>

            <div className="p-3 bg-white/5 border-t border-white/5 flex justify-end gap-3">
                <button 
                    onClick={onReject}
                    className="px-4 py-2 rounded text-xs font-bold font-mono tracking-wider border border-white/10 text-vgt-text-dim hover:bg-red-500/10 hover:text-red-400 hover:border-red-500/50 transition-all"
                >
                    DENY
                </button>
                <button 
                    onClick={() => onApprove(false)} // Normal Approve
                    className="px-4 py-2 rounded text-xs font-bold font-mono tracking-wider bg-vgt-cyan/10 border border-vgt-cyan/50 text-vgt-cyan hover:bg-vgt-cyan/20 hover:shadow-[0_0_15px_rgba(0,240,255,0.3)] transition-all flex items-center gap-2"
                >
                    <Terminal size={12} /> APPROVE EXECUTION
                </button>
            </div>
        </div>
    );
}