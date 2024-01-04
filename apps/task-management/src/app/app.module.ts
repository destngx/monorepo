import { Module } from '@nestjs/common';

import { AppController } from './app.controller';
import { AppService } from './app.service';
import { ConfigModule } from '@nestjs/config';
import { baseConfig, redisConfig } from '@task-management/config';
import { QueueModule } from '@task-management/queue';

@Module({
  // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      load: [baseConfig, redisConfig],
    }),
    QueueModule,
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
