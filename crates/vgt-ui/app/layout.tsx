import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";

const inter = Inter({ subsets: ["latin"], variable: '--font-inter' });
const mono = JetBrains_Mono({ subsets: ["latin"], variable: '--font-mono' });

export const metadata: Metadata = {
  title: "VGT AETHEL | SUPREME INTELLIGENCE",
  description: "Sovereign AI Interface powered by Rust & Groq",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className={`${inter.variable} ${mono.variable} antialiased bg-vgt-void text-vgt-text-primary selection:bg-vgt-cyan selection:text-black`}>
        {/* Background Grid Layer */}
        <div className="fixed inset-0 z-[-1] bg-grid-pattern pointer-events-none" />
        
        {/* Main Content */}
        <main className="min-h-screen flex flex-col items-center justify-center p-4 md:p-8">
          {children}
        </main>
      </body>
    </html>
  );
}