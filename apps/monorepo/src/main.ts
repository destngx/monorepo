import { Logger } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';

import { AppConfig } from './infrastructure/config/env';
import { AppModule } from './application/app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  const globalPrefix = 'api';
  app.setGlobalPrefix(globalPrefix);
  const port = AppConfig.PORT;
  Logger.log(`🚀 Application is starting on: http://localhost:${port}/${globalPrefix}`);
  await app.listen(port);
}
// eslint-disable-next-line
bootstrap().then(r => Logger.log(`🚀 Application is running ${r}`));
