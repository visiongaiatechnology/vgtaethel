import { Terminal, Activity } from "lucide-react";
import { useEffect, useRef } from "react";
import { Message, ToolCallRequest } from "@/hooks/useVgtEngine";
import { ToolRequestCard } from "./ToolRequestCard";

interface TerminalOutputProps {
    messages: Message[];
    isStreaming: boolean;
    currentResponse: string;
    onToolAction?: (msgIndex: number, approved: boolean, force?: boolean) => void;
}

export function TerminalOutput({ messages, isStreaming, currentResponse, onToolAction }: TerminalOutputProps) {
    const scrollRef = useRef<HTMLDivElement>(null);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [messages, currentResponse, isStreaming]);

    return (
        <div ref={scrollRef} className="flex-1 p-6 font-mono text-sm space-y-6 overflow-y-auto scroll-smooth custom-scrollbar">
            {messages.length === 0 && (
                <div className="flex gap-4 opacity-50 h-full items-center justify-center">
                    <div className="text-center">
                        <div className="w-12 h-12 rounded-full bg-vgt-cyan/10 flex items-center justify-center text-vgt-cyan mx-auto mb-4 border border-vgt-cyan/20">
                            <Terminal size={24} />
                        </div>
                        <p className="text-vgt-cyan text-xs mb-1 tracking-widest">SYSTEM READY</p>
                        <p className="text-vgt-text-dim text-xs">
                            Awaiting Neural Directive.<br/>
                            Select Protocol and Execute.
                        </p>
                    </div>
                </div>
            )}

            {messages.map((msg, idx) => (
                <div key={idx}>
                    <MessageBubble message={msg} />
                    {/* Render Tool Card if present */}
                    {msg.toolCall && onToolAction && (
                        <div className="ml-12">
                            <ToolRequestCard 
                                tool={msg.toolCall} 
                                onApprove={(force) => onToolAction(idx, true, force)}
                                onReject={() => onToolAction(idx, false)}
                            />
                        </div>
                    )}
                </div>
            ))}

            {isStreaming && currentResponse && (
                <div className="flex gap-4">
                    <div className="w-8 h-8 rounded-full bg-vgt-cyan/10 border border-vgt-cyan/30 flex items-center justify-center text-vgt-cyan animate-pulse shrink-0">
                        <Terminal size={14} />
                    </div>
                    <div className="flex-1">
                        <p className="text-vgt-cyan text-[10px] mb-1 tracking-wider opacity-70">GENERATING SEQUENCE...</p>
                        <div className="opacity-90 leading-relaxed whitespace-pre-wrap border-l-2 border-vgt-cyan pl-3 text-vgt-text-primary">
                            {currentResponse}
                            <span className="inline-block w-2 h-4 bg-vgt-cyan ml-1 animate-pulse align-middle"/>
                        </div>
                    </div>
                </div>
            )}
            
            <div ref={messagesEndRef} />
        </div>
    );
}

function MessageBubble({ message }: { message: Message }) {
    if (!message.content && message.toolCall) return null;

    const isUser = message.role === 'user';
    
    return (
        <div className={`flex gap-4 ${isUser ? 'flex-row-reverse' : ''} group`}>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 border backdrop-blur-sm
                ${isUser 
                    ? 'bg-vgt-purple/10 border-vgt-purple/30 text-vgt-purple' 
                    : 'bg-vgt-cyan/10 border-vgt-cyan/30 text-vgt-cyan'}`}>
                {isUser ? <Activity size={14} /> : <Terminal size={14} />}
            </div>

            <div className={`flex-1 max-w-[85%] ${isUser ? 'text-right' : ''}`}>
                <p className={`text-[10px] mb-1 font-bold tracking-wider opacity-70 ${isUser ? 'text-vgt-purple' : 'text-vgt-cyan'}`}>
                    {isUser ? 'OPERATOR :: ID_01' : 'AETHEL :: CORE'}
                </p>
                <div className={`opacity-90 leading-relaxed whitespace-pre-wrap transition-all duration-300
                    ${isUser ? 'text-vgt-text-dim group-hover:text-white' : 'text-white'}`}>
                    {message.content}
                </div>
            </div>
        </div>
    );
}