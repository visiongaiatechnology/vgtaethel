import { Client, GatewayIntentBits, Partials, ChannelType as DChannelType } from 'discord.js';
import { ChannelProvider, ChannelType, IncomingMessage } from '../core/bridge';
import { v4 as uuidv4 } from 'uuid';

export class DiscordProvider extends ChannelProvider {
    channelType = ChannelType.Discord;
    private client: Client;

    constructor(private token: string) {
        super();
        this.client = new Client({
            intents: [
                GatewayIntentBits.Guilds,
                GatewayIntentBits.GuildMessages,
                GatewayIntentBits.MessageContent,
                GatewayIntentBits.DirectMessages
            ],
            partials: [Partials.Channel] // Required for DMs
        });
    }

    async initialize(): Promise<void> {
        console.log('Initializing Discord...');

        this.client.on('ready', () => {
            console.log(`✅ Discord connected as ${this.client.user?.tag}`);
        });

        this.client.on('messageCreate', (message) => {
            if (message.author.bot) return;

            const payload: IncomingMessage = {
                id: message.id,
                channel: this.channelType,
                source_id: message.channelId,
                group_id: message.guildId || undefined,
                author_name: message.author.username,
                content: message.content,
                timestamp: message.createdAt.toISOString(),
                metadata: {
                    isDM: (message.channel.type === DChannelType.DM).toString()
                }
            };
            this.emit('message', payload);
        });

        await this.client.login(this.token);
    }

    async sendMessage(targetId: string, content: string, attachments: string[]): Promise<void> {
        const channel = await this.client.channels.fetch(targetId);
        if (channel && channel.isTextBased()) {
            await channel.send(content);
        } else {
            console.error(`Discord channel ${targetId} not found or not text-based`);
        }
    }
}