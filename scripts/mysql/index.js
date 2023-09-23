"use strict";
exports.__esModule = true;
var promise_1 = require("mysql2/promise");
var service_1 = require("./service");
var connection = await promise_1["default"].createConnection({
    host: "127.0.0.1",
    user: "root",
    password: "",
    database: "test"
});
connection.connect();
var service = (0, service_1["default"])(connection);
console.log(await service.count());
connection.end();
