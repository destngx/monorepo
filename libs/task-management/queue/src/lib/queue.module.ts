import { Module } from '@nestjs/common';
import { BullModule } from '@nestjs/bull';
import { ConfigModule, ConfigService } from '@nestjs/config';

export const QUEUE_DEFAULT = 'default';
@Module({
  imports: [
    BullModule.forRootAsync({
      imports: [ConfigModule],
      // eslint-disable-next-line @typescript-eslint/require-await
      useFactory: async (configService: ConfigService) => ({
        redis: {
          host: configService.get<string>('socket.host') || '127.0.0.1',
          port: configService.get<number>('socket.port') || 6379,
          password: configService.get<string>('password') || undefined,
        },
      }),
      inject: [ConfigService],
    }),
    BullModule.registerQueue({ name: QUEUE_DEFAULT }),
  ],
})
export class QueueModule {}
