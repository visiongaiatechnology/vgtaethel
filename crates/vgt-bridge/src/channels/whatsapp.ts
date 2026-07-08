import makeWASocket, { 
    useMultiFileAuthState, 
    DisconnectReason, 
    WASocket 
} from '@adiwajshing/baileys';
import { ChannelProvider, ChannelType, IncomingMessage } from '../core/bridge';
import { v4 as uuidv4 } from 'uuid';
import * as fs from 'fs';

export class WhatsAppProvider extends ChannelProvider {
    channelType = ChannelType.WhatsApp;
    private sock: WASocket | null = null;

    async initialize(): Promise<void> {
        console.log('Initializing WhatsApp (Baileys)...');
        const { state, saveCreds } = await useMultiFileAuthState('auth_info_baileys');

        this.sock = makeWASocket({
            auth: state,
            printQRInTerminal: true,
            syncFullHistory: false
        });

        this.sock.ev.on('creds.update', saveCreds);

        this.sock.ev.on('connection.update', (update) => {
            const { connection, lastDisconnect } = update;
            if (connection === 'close') {
                const shouldReconnect = (lastDisconnect?.error as any)?.output?.statusCode !== DisconnectReason.loggedOut;
                console.log('WA Connection closed. Reconnecting:', shouldReconnect);
                if (shouldReconnect) {
                    this.initialize();
                }
            } else if (connection === 'open') {
                console.log('✅ WhatsApp connected');
            }
        });

        this.sock.ev.on('messages.upsert', async (m) => {
            if (m.type === 'notify' || m.type === 'append') {
                for (const msg of m.messages) {
                    if (!msg.key.fromMe && msg.message) {
                        const content = msg.message.conversation || msg.message.extendedTextMessage?.text || "";
                        if (!content) continue; // Skip non-text for now

                        const payload: IncomingMessage = {
                            id: msg.key.id || uuidv4(),
                            channel: this.channelType,
                            source_id: msg.key.remoteJid || 'unknown',
                            author_name: msg.pushName || 'User',
                            content: content,
                            timestamp: new Date().toISOString(),
                            metadata: {}
                        };
                        this.emit('message', payload);
                    }
                }
            }
        });
    }

    async sendMessage(targetId: string, content: string, attachments: string[]): Promise<void> {
        if (!this.sock) throw new Error("WhatsApp not connected");
        await this.sock.sendMessage(targetId, { text: content });
    }
}