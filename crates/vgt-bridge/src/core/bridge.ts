import WebSocket from 'ws';
import { v4 as uuidv4 } from 'uuid';
import { EventEmitter } from 'events';

export enum ChannelType {
    WhatsApp = "WhatsApp",
    Telegram = "Telegram",
    Discord = "Discord",
    Signal = "Signal",
    BlueBubbles = "BlueBubbles",
    Teams = "Teams",
    Matrix = "Matrix"
}

export interface IncomingMessage {
    id: string;
    channel: ChannelType;
    source_id: string;
    group_id?: string;
    author_name?: string;
    content: string;
    media_url?: string;
    timestamp: string; // ISO String
    metadata: Record<string, string>;
}

export interface OutgoingMessage {
    target_channel: ChannelType;
    target_id: string;
    content: string;
    attachments: string[];
}

export abstract class ChannelProvider extends EventEmitter {
    abstract channelType: ChannelType;
    abstract initialize(): Promise<void>;
    abstract sendMessage(targetId: string, content: string, attachments: string[]): Promise<void>;
}

export class NexusBridge {
    private ws: WebSocket;
    private providers: Map<ChannelType, ChannelProvider> = new Map();
    private reconnectInterval: NodeJS.Timeout | null = null;

    constructor(private coreUrl: string) {
        this.connect();
    }

    private connect() {
        this.ws = new WebSocket(this.coreUrl);

        this.ws.on('open', () => {
            console.log('⚡ VGT NEXUS BRIDGE CONNECTED TO CORE');
            this.registerProviders();
            if (this.reconnectInterval) clearInterval(this.reconnectInterval);
        });

        this.ws.on('message', (data: string) => {
            try {
                const msg: OutgoingMessage = JSON.parse(data.toString());
                this.routeMessage(msg);
            } catch (e) {
                console.error('Failed to parse core message', e);
            }
        });

        this.ws.on('close', () => {
            console.warn('VGT Core disconnected. Retrying...');
            this.reconnectInterval = setInterval(() => this.connect(), 5000);
        });
    }

    public registerProvider(provider: ChannelProvider) {
        this.providers.set(provider.channelType, provider);
        
        // Listen to incoming messages from provider
        provider.on('message', (msg: IncomingMessage) => {
            this.sendToCore(msg);
        });
    }

    private registerProviders() {
        const channels = Array.from(this.providers.keys());
        this.ws.send(JSON.stringify({
            command: "REGISTER",
            channels: channels
        }));
    }

    private sendToCore(msg: IncomingMessage) {
        if (this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(msg));
        } else {
            console.error('Queueing not implemented - Message dropped (Platinum Status requires Queue, TODO)');
        }
    }

    private async routeMessage(msg: OutgoingMessage) {
        const provider = this.providers.get(msg.target_channel);
        if (provider) {
            try {
                await provider.sendMessage(msg.target_id, msg.content, msg.attachments);
                console.log(`[${msg.target_channel}] Sent message to ${msg.target_id}`);
            } catch (e) {
                console.error(`[${msg.target_channel}] Failed to send:`, e);
            }
        }
    }
}