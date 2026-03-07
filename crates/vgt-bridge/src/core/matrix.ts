import * as sdk from 'matrix-js-sdk';
import { ChannelProvider, ChannelType, IncomingMessage } from '../core/bridge';

export class MatrixProvider extends ChannelProvider {
    channelType = ChannelType.Matrix;
    private client: any;

    constructor(private baseUrl: string, private accessToken: string, private userId: string) {
        super();
        this.client = sdk.createClient({
            baseUrl,
            accessToken,
            userId
        });
    }

    async initialize(): Promise<void> {
        console.log('Initializing Matrix...');
        
        this.client.on("Room.timeline", (event: any, room: any, toStartOfTimeline: boolean) => {
            if (event.getType() !== "m.room.message") return;
            if (toStartOfTimeline) return; // Don't process old messages
            if (event.getSender() === this.userId) return;

            const payload: IncomingMessage = {
                id: event.getId(),
                channel: this.channelType,
                source_id: room.roomId,
                author_name: event.getSender(),
                content: event.getContent().body,
                timestamp: new Date().toISOString(),
                metadata: {}
            };
            this.emit('message', payload);
        });

        await this.client.startClient();
        console.log('✅ Matrix Client Started');
    }

    async sendMessage(targetId: string, content: string, attachments: string[]): Promise<void> {
        await this.client.sendEvent(targetId, "m.room.message", {
            "msgtype": "m.text",
            "body": content
        }, "");
    }
}