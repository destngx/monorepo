import { Global, Module, OnModuleDestroy, OnModuleInit } from '@nestjs/common';
import { DataSource, EntityManager } from 'typeorm';
import { Configuration } from '../../config/env';

const Config = new Configuration();
interface WriteConnection {
  readonly startTransaction: (
    level?: 'READ UNCOMMITTED' | 'READ COMMITTED' | 'REPEATABLE READ' | 'SERIALIZABLE'
  ) => Promise<void>;
  readonly commitTransaction: () => Promise<void>;
  readonly rollbackTransaction: () => Promise<void>;
  readonly isTransactionActive: boolean;
  readonly manager: EntityManager;
}
export let writeConnection = {} as WriteConnection;

class DatabaseService implements OnModuleInit, OnModuleDestroy {
  private readonly _dataSource = new DataSource({
    type: 'mysql',
    entities: [],
    charset: 'utf8mb4_unicode_ci',
    logging: Config.DATABASE_WRITE_LOGGING,
    host: Config.DATABASE_WRITE_HOST,
    port: Config.DATABASE_WRITE_PORT,
    database: Config.DATABASE_WRITE_NAME,
    username: Config.DATABASE_WRITE_USER,
    password: Config.DATABASE_WRITE_PASSWORD,
    synchronize: Config.DATABASE_WRITE_SYNC,
  });

  async onModuleInit(): Promise<void> {
    await this._dataSource.initialize();
    if (!this._dataSource.isInitialized) {
      throw new Error('DataSource is not initialized');
    }
    writeConnection = this._dataSource.createQueryRunner();
  }

  async onModuleDestroy(): Promise<void> {
    await this._dataSource.destroy();
  }
}
@Global()
@Module({
  providers: [DatabaseService],
  exports: [],
})
export class WriteDatabaseModule {}
