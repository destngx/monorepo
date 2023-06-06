import { Module } from '@nestjs/common';

import { AppController } from './app.controller';
import { AppService } from './app.service';
import { ReadDatabaseModule, WriteDatabaseModule } from '@nx-pnpm-monorepo/common';

@Module({
  imports: [ReadDatabaseModule, WriteDatabaseModule],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
