import { BaseEntity } from '@nx-pnpm-monorepo/common';
import { Entity, Column, PrimaryColumn } from 'typeorm';

@Entity()
export class UserEntity extends BaseEntity {
  @PrimaryColumn({ type: 'binary', length: 16 })
  id: Buffer;

  @Column()
  name: string;

  @Column()
  email: string;

  @Column()
  password: string;

  @Column({ type: 'datetime', precision: 6, nullable: true })
  lockedAt: Date | null;
}
