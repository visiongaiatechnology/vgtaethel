import { Cpu, Activity, ShieldCheck } from "lucide-react";

interface SystemStatusProps {
  modelCount: number;
  isStreaming: boolean;
}

export function SystemStatus({ modelCount, isStreaming }: SystemStatusProps) {
  return (
    <div className="flex gap-4">
      <StatusBadge 
        icon={<Cpu size={14}/>} 
        label="ENGINE" 
        value={isStreaming ? "PROCESSING" : "IDLE"} 
        active={isStreaming}
      />
      <StatusBadge 
        icon={<Activity size={14}/>} 
        label="MODELS" 
        value={modelCount.toString()} 
      />
      <StatusBadge 
        icon={<ShieldCheck size={14}/>} 
        label="SECURITY" 
        value="PLATINUM" 
      />
    </div>
  );
}

function StatusBadge({ icon, label, value, active = false }: { icon: React.ReactNode, label: string, value: string, active?: boolean }) {
  return (
      <div className={`flex items-center gap-2 px-3 py-1 rounded-full border text-xs font-mono transition-all duration-300
        ${active 
          ? 'bg-vgt-cyan/10 border-vgt-cyan/50 text-vgt-cyan shadow-[0_0_10px_rgba(0,240,255,0.2)]' 
          : 'bg-white/5 border-white/10 text-vgt-text-dim'
        }`}>
          <span className={active ? "text-vgt-cyan animate-pulse" : "text-vgt-cyan"}>{icon}</span>
          <span className="text-vgt-text-dim">{label}</span>
          <span className={`font-bold ${active ? "text-white" : "text-white"}`}>{value}</span>
      </div>
  )
}