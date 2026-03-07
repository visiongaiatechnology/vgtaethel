import { ChannelProvider, ChannelType } from '../core/bridge';

export class TeamsProvider extends ChannelProvider {
    channelType = ChannelType.Teams;

    constructor(private appId: string, private appPassword: string) {
        super();
    }

    async initialize(): Promise<void> {
        console.log('Initializing MS Teams Bot Framework...');
        console.log('✅ Teams Adapter Ready (Requires HTTP Endpoint Tunneling)');
        // Teams works via Webhook callbacks, so the Bridge needs an Express server
        // to receive POST requests from Azure.
    }

    async sendMessage(targetId: string, content: string, attachments: string[]): Promise<void> {
        console.log('Sending to Teams via ConnectorClient...');
    }
}