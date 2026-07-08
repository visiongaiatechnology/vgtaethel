import { VgtModel } from "@/hooks/useVgtEngine";

interface ModelSelectorProps {
  models: VgtModel[];
  activeModelId: string;
  onSelectModel: (id: string) => void;
}

export function ModelSelector({ models, activeModelId, onSelectModel }: ModelSelectorProps) {
  return (
    <div className="md:col-span-3 vgt-glass rounded-xl p-4 flex flex-col gap-4 overflow-y-auto h-full max-h-[75vh]">
        <h2 className="text-sm font-mono text-vgt-text-dim uppercase border-b border-vgt-border pb-2 sticky top-0 bg-[#030014]/90 backdrop-blur-md z-10">
            Neural Configuration
        </h2>
        
        <div className="flex flex-col gap-2">
            {models.length === 0 ? (
                <div className="text-xs text-vgt-text-dim animate-pulse p-2">
                    > CONNECTING TO VGT CORE...
                </div>
            ) : (
                models.map((m) => (
                    <ModelCard 
                        key={m.id}
                        name={m.name} 
                        tier={m.tier} 
                        active={activeModelId === m.id}
                        onClick={() => onSelectModel(m.id)}
                    />
                ))
            )}
        </div>
    </div>
  );
}

// Sub-Component für interne Kapselung
function ModelCard({ name, tier, active, onClick }: { name: string, tier: string, active: boolean, onClick: () => void }) {
  return (
      <div 
          onClick={onClick}
          className={`p-3 rounded border transition-all cursor-pointer flex justify-between items-center group relative overflow-hidden
          ${active 
              ? 'bg-vgt-cyan/10 border-vgt-cyan text-white shadow-[inset_0_0_20px_rgba(0,240,255,0.05)]' 
              : 'bg-transparent border-white/5 text-vgt-text-dim hover:bg-white/5 hover:border-white/10'
          }`}>
          <div className="flex flex-col relative z-10">
              <span className={`text-xs font-bold ${active ? 'text-glow' : ''}`}>{name}</span>
              <span className="text-[10px] tracking-widest opacity-60 font-mono">{tier}</span>
          </div>
          {active && (
              <>
                  <div className="w-1.5 h-1.5 rounded-full bg-vgt-cyan shadow-[0_0_8px_#00f0ff] relative z-10" />
                  {/* Scanline Effect */}
                  <div className="absolute inset-0 bg-gradient-to-r from-transparent via-vgt-cyan/5 to-transparent skew-x-12 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000" />
              </>
          )}
      </div>
  )
}