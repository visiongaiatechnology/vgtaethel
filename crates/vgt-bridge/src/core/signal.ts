import { exec, spawn } from 'child_process';
import { ChannelProvider, ChannelType, IncomingMessage } from '../core/bridge';
import { v4 as uuidv4 } from 'uuid';

// Requirement: signal-cli installed and registered on host system
export class SignalProvider extends ChannelProvider {
    channelType = ChannelType.Signal;
    private process: any;

    constructor(private user: string) {
        super();
    }

    async initialize(): Promise<void> {
        console.log('Initializing Signal (JSON RPC)...');
        
        // Start signal-cli in daemon mode JSON-RPC
        this.process = spawn('signal-cli', ['-u', this.user, '--output=json', 'daemon']);

        this.process.stdout.on('data', (data: Buffer) => {
            const lines = data.toString().split('\n');
            for (const line of lines) {
                if (!line.trim()) continue;
                try {
                    const json = JSON.parse(line);
                    if (json.envelope && json.envelope.dataMessage) {
                        const msg = json.envelope.dataMessage;
                        const payload: IncomingMessage = {
                            id: msg.timestamp.toString(),
                            channel: this.channelType,
                            source_id: json.envelope.sourceNumber || json.envelope.source,
                            author_name: 'Signal User', // Signal doesn't always send profiles
                            content: msg.message,
                            timestamp: new Date(msg.timestamp).toISOString(),
                            metadata: {}
                        };
                        this.emit('message', payload);
                    }
                } catch (e) {
                    // Ignore non-json stdout
                }
            }
        });

        this.process.stderr.on('data', (data: any) => {
            console.error(`Signal Error: ${data}`);
        });
        
        console.log('✅ Signal-cli daemon attached');
    }

    async sendMessage(targetId: string, content: string, attachments: string[]): Promise<void> {
        // Use exec for sending to avoid complexity of JSON-RPC ID tracking for now
        // In Platinum production: Use proper stdin write with request ID
        const cmd = `signal-cli -u ${this.user} send -m "${content.replace(/"/g, '\\"')}" ${targetId}`;
        exec(cmd, (error, stdout, stderr) => {
            if (error) console.error(`Signal send failed: ${stderr}`);
        });
    }
}