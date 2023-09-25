// typeorm.config.ts

import { DataSource } from 'typeorm';
import { UserEntity } from '../../domain/entity/user.entity';

export default new DataSource({
  type: 'postgres',
  host: 'localhost',
  port: 3306,
  username: 'user',
  password: 'mauFJcuf5dhRMQrjj',
  entities: [UserEntity],
});
