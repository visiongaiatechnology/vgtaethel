import { Bot } from "grammy";
import { ChannelProvider, ChannelType, IncomingMessage } from '../core/bridge';
import { v4 as uuidv4 } from 'uuid';

export class TelegramProvider extends ChannelProvider {
    channelType = ChannelType.Telegram;
    private bot: Bot;

    constructor(private token: string) {
        super();
        this.bot = new Bot(token);
    }

    async initialize(): Promise<void> {
        console.log('Initializing Telegram (grammY)...');

        this.bot.on("message:text", (ctx) => {
            const payload: IncomingMessage = {
                id: ctx.msg.message_id.toString(),
                channel: this.channelType,
                source_id: ctx.chat.id.toString(),
                author_name: ctx.from?.first_name || 'User',
                content: ctx.message.text,
                timestamp: new Date().toISOString(),
                metadata: {
                    username: ctx.from?.username || ''
                }
            };
            this.emit('message', payload);
        });

        this.bot.start({
            onStart: (botInfo) => {
                console.log(`✅ Telegram connected as ${botInfo.username}`);
            }
        });
    }

    async sendMessage(targetId: string, content: string, attachments: string[]): Promise<void> {
        await this.bot.api.sendMessage(targetId, content);
    }
}