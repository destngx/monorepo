// typeorm.config.ts

import { BaseEntity } from '../../domain/entity';
import { DataSource } from 'typeorm';

export default new DataSource({
  type: 'postgres',
  host: 'localhost',
  port: 3306,
  username: 'user',
  password: 'mauFJcuf5dhRMQrjj',
  database: 'monorepo',
  entities: [BaseEntity],
});
