import { Controller, Get } from '@nestjs/common';

import { AppService } from './app.service';

@Controller()
export class AppController {
  constructor(private readonly _appService: AppService) {}

  @Get()
  getData() {
    throw Error('Not implemented');
    return this._appService.getData();
  }
}
