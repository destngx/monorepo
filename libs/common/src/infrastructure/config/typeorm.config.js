"use strict";
// typeorm.config.ts
Object.defineProperty(exports, "__esModule", { value: true });
var entity_1 = require("../../domain/entity");
var typeorm_1 = require("typeorm");
exports.default = new typeorm_1.DataSource({
    type: 'postgres',
    host: 'localhost',
    port: 3306,
    username: 'user',
    password: 'mauFJcuf5dhRMQrjj',
    database: 'monorepo',
    entities: [entity_1.BaseEntity],
});
