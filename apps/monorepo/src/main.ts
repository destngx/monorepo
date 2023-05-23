import { Logger } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';

import { AppConfig } from './infrastructure/config/env';
import { AppModule } from './application/app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  const globalPrefix = 'api';
  app.setGlobalPrefix(globalPrefix);
  const port = AppConfig.PORT;
  await app.listen(port);
  Logger.log(`🚀 Application is running on: http://localhost:${port}/${globalPrefix}`);
}

bootstrap();
