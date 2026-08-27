'use client';

import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { JSX, ReactNode } from 'react';

// ---------------------------------------------------------------------------
// VGT MARKDOWN RENDERER
// Parsed Markdown (Tables, Code, Headings, Bold, etc.) into styled React nodes.
// Integrates with the Cyberpunk glass aesthetic via Tailwind classes.
// ---------------------------------------------------------------------------

interface MarkdownRendererProps {
    content: string;
    /** If true, renders the streaming cursor at the end */
    isStreaming?: boolean;
}

/** Maps <h1> – <h6> to VGT-appropriate Tailwind classes */
function heading(level: number, children: ReactNode) {
    const Tag = `h${level}` as keyof JSX.IntrinsicElements;
    const sizes: Record<number, string> = {
        1: 'text-2xl font-bold border-b border-vgt-cyan/30 pb-2',
        2: 'text-xl font-bold border-b border-vgt-cyan/20 pb-1',
        3: 'text-lg font-semibold',
        4: 'text-base font-semibold',
        5: 'text-sm font-semibold',
        6: 'text-xs font-semibold',
    };
    return <Tag className={`${sizes[level] || 'text-sm'} text-white mt-4 mb-2 tracking-wide`}>{children}</Tag>;
}

export function MarkdownRenderer({ content, isStreaming }: MarkdownRendererProps) {
    return (
        <div className="vgt-markdown text-sm leading-relaxed space-y-1">
            <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                    // --- Headings ---
                    h1: ({ children }) => heading(1, children),
                    h2: ({ children }) => heading(2, children),
                    h3: ({ children }) => heading(3, children),
                    h4: ({ children }) => heading(4, children),
                    h5: ({ children }) => heading(5, children),
                    h6: ({ children }) => heading(6, children),

                    // --- Tables (the core fix) ---
                    table: ({ children }) => (
                        <div className="overflow-x-auto my-3 rounded-lg border border-vgt-border/40">
                            <table className="min-w-full text-left text-xs font-mono border-collapse">
                                {children}
                            </table>
                        </div>
                    ),
                    thead: ({ children }) => (
                        <thead className="bg-vgt-cyan/10 text-vgt-cyan uppercase tracking-wider border-b border-vgt-cyan/30">
                            {children}
                        </thead>
                    ),
                    tbody: ({ children }) => (
                        <tbody className="divide-y divide-white/5">{children}</tbody>
                    ),
                    th: ({ children }) => (
                        <th className="px-4 py-2 font-semibold whitespace-nowrap">{children}</th>
                    ),
                    td: ({ children }) => (
                        <td className="px-4 py-2 text-white/80 whitespace-nowrap">{children}</td>
                    ),
                    tr: ({ children }) => (
                        <tr className="hover:bg-white/5 transition-colors">{children}</tr>
                    ),

                    // --- Code ---
                    code: ({ children, className }) => {
                        const isBlock = className?.includes('language-');
                        if (isBlock) {
                            return (
                                <div className="relative my-3 rounded-lg border border-vgt-cyan/20 bg-black/40 overflow-hidden">
                                    <div className="flex items-center justify-between px-4 py-2 bg-black/60 border-b border-vgt-cyan/20">
                                        <span className="text-[10px] text-vgt-cyan uppercase tracking-widest font-mono">
                                            {className?.replace('language-', '') || 'CODE'}
                                        </span>
                                    </div>
                                    <pre className="p-4 overflow-x-auto">
                                        <code className="text-xs font-mono text-green-400">{children}</code>
                                    </pre>
                                </div>
                            );
                        }
                        return (
                            <code className="px-1.5 py-0.5 rounded bg-vgt-cyan/10 text-vgt-cyan text-xs font-mono border border-vgt-cyan/20">
                                {children}
                            </code>
                        );
                    },

                    // --- Blockquote ---
                    blockquote: ({ children }) => (
                        <blockquote className="border-l-2 border-vgt-purple/50 pl-4 italic text-white/60 my-2">
                            {children}
                        </blockquote>
                    ),

                    // --- Lists ---
                    ul: ({ children }) => (
                        <ul className="list-disc list-inside space-y-1 my-2 text-white/80">{children}</ul>
                    ),
                    ol: ({ children }) => (
                        <ol className="list-decimal list-inside space-y-1 my-2 text-white/80">{children}</ol>
                    ),
                    li: ({ children }) => (
                        <li className="pl-1">{children}</li>
                    ),

                    // --- Emphasis ---
                    strong: ({ children }) => (
                        <strong className="font-bold text-white">{children}</strong>
                    ),
                    em: ({ children }) => (
                        <em className="italic text-vgt-cyan">{children}</em>
                    ),

                    // --- Links ---
                    a: ({ children, href }) => (
                        <a
                            href={href}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-vgt-cyan underline decoration-vgt-cyan/30 hover:decoration-vgt-cyan transition-all"
                        >
                            {children}
                        </a>
                    ),

                    // --- Horizontal Rule ---
                    hr: () => (
                        <hr className="border-vgt-border/30 my-4" />
                    ),

                    // --- Paragraph ---
                    p: ({ children }) => (
                        <p className="text-white/85 my-1">{children}</p>
                    ),
                }}
            >
                {content}
            </ReactMarkdown>
            {isStreaming && (
                <span className="inline-block w-2 h-4 bg-vgt-cyan ml-1 animate-pulse align-middle" />
            )}
        </div>
    );
}
