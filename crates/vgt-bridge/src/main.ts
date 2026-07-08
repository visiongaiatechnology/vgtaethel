import { NexusBridge } from './core/bridge';
import { WhatsAppProvider } from './channels/whatsapp';
import { TelegramProvider } from './channels/telegram';
import { DiscordProvider } from './channels/discord';
import { SignalProvider } from './channels/signal';
import { MatrixProvider } from './channels/matrix';
import dotenv from 'dotenv';

dotenv.config();

const CORE_URL = process.env.VGT_CORE_URL || 'ws://localhost:3000/api/nexus/socket';

async function bootstrap() {
    console.log('🚀 STARTING VGT NEXUS BRIDGE (SUPREME TIER)');
    
    const bridge = new NexusBridge(CORE_URL);

    // --- WHATSAPP ---
    // if (process.env.ENABLE_WHATSAPP === 'true') {
        const wa = new WhatsAppProvider();
        await wa.initialize();
        bridge.registerProvider(wa);
    // }

    // --- TELEGRAM ---
    if (process.env.TELEGRAM_TOKEN) {
        const tg = new TelegramProvider(process.env.TELEGRAM_TOKEN);
        await tg.initialize();
        bridge.registerProvider(tg);
    }

    // --- DISCORD ---
    if (process.env.DISCORD_TOKEN) {
        const discord = new DiscordProvider(process.env.DISCORD_TOKEN);
        await discord.initialize();
        bridge.registerProvider(discord);
    }
    
    // --- SIGNAL ---
    if (process.env.SIGNAL_USER) {
        const signal = new SignalProvider(process.env.SIGNAL_USER);
        await signal.initialize();
        bridge.registerProvider(signal);
    }

    // --- MATRIX ---
    if (process.env.MATRIX_ACCESS_TOKEN) {
        const matrix = new MatrixProvider(
            process.env.MATRIX_URL || 'https://matrix.org', 
            process.env.MATRIX_ACCESS_TOKEN, 
            process.env.MATRIX_USER_ID || ''
        );
        await matrix.initialize();
        bridge.registerProvider(matrix);
    }

    console.log('🌐 VGT BRIDGE ACTIVE AND LISTENING');
}

bootstrap().catch(console.error);