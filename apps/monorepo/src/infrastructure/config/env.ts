import { Configuration } from '@nx-pnpm-monorepo/common';
import { IsInt, validateSync } from 'class-validator';

class AppConfiguration extends Configuration {
  @IsInt()
  readonly PORT = Number(3000);

  constructor() {
    super();
    const error = validateSync(this);
    if (!error.length) return;
    console.error(`Config validation error: ${JSON.stringify(error)}`);
    process.exit(1);
  }
}

export const AppConfig = new AppConfiguration();
