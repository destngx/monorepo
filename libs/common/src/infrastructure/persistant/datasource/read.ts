import { Global, Module, OnModuleDestroy, OnModuleInit } from '@nestjs/common';
import { DataSource, EntityTarget, ObjectLiteral, QueryRunner, Repository, SelectQueryBuilder } from 'typeorm';
import { Configuration } from '../../config/env';
import { BaseEntity } from '../../../index';

const Config = new Configuration();
interface ReadConnection {
  readonly getRepository: <T extends ObjectLiteral>(target: EntityTarget<T>) => Repository<T>;
  readonly query: (query: string) => Promise<void>;
  readonly createQueryBuilder: <TEntity extends ObjectLiteral>(
    entityClass: EntityTarget<TEntity>,
    alias: string,
    queryRunner?: QueryRunner
  ) => SelectQueryBuilder<TEntity>;
}

export let readConnection = {} as ReadConnection;

class DatabaseService implements OnModuleInit, OnModuleDestroy {
  private readonly _dataSource = new DataSource({
    type: 'mysql',
    entities: [BaseEntity],
    charset: 'utf8mb4_unicode_ci',
    logging: Config.DATABASE_READ_LOGGING,
    host: Config.DATABASE_READ_HOST,
    port: Config.DATABASE_READ_PORT,
    database: Config.DATABASE_READ_NAME,
    username: Config.DATABASE_READ_USER,
    password: Config.DATABASE_READ_PASSWORD,
    synchronize: Config.DATABASE_READ_SYNC,
  });

  async onModuleInit(): Promise<void> {
    await this._dataSource.initialize();
    if (!this._dataSource.isInitialized) {
      throw new Error('DataSource is not initialized');
    }
    readConnection = this._dataSource.manager;
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
export class ReadDatabaseModule {}
