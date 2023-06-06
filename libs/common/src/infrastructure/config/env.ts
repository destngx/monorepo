import { Logger } from '@nestjs/common';
import { IsBoolean, IsInt, IsString, validateSync } from 'class-validator';

export class Configuration {
  private readonly _logger = new Logger(Configuration.name);

  @IsBoolean()
  readonly DATABASE_READ_LOGGING = true;

  @IsString()
  readonly DATABASE_READ_HOST = 'localhost';

  @IsInt()
  readonly DATABASE_READ_PORT = Number(3306);

  @IsString()
  readonly DATABASE_READ_NAME = '';

  @IsString()
  readonly DATABASE_READ_USER = 'root';

  @IsString()
  readonly DATABASE_READ_PASSWORD = 'mauFJcuf5dhRMQrjj';

  @IsBoolean()
  readonly DATABASE_READ_SYNC = true;

  @IsBoolean()
  readonly DATABASE_WRITE_LOGGING = true;

  @IsString()
  readonly DATABASE_WRITE_HOST = 'localhost';

  @IsInt()
  readonly DATABASE_WRITE_PORT = Number(3306);

  @IsString()
  readonly DATABASE_WRITE_NAME = '';

  @IsString()
  readonly DATABASE_WRITE_USER = 'root';

  @IsString()
  readonly DATABASE_WRITE_PASSWORD = 'mauFJcuf5dhRMQrjj';

  @IsBoolean()
  readonly DATABASE_WRITE_SYNC = true;

  constructor() {
    const error = validateSync(this);
    if (!error.length) return;
    this._logger.error(`Config validation error: ${JSON.stringify(error)}`);
    process.exit(1);
  }
}
