import mysql from "mysql2/promise";
import createService from "./service";

const connection = await mysql.createConnection({
  host: "127.0.0.1",
  user: "root",
  password: "",
  database: "test",
});

connection.connect();

const service = createService(connection, "accounts");
await service.migrate();
console.log(await service.seed(1_000_000));

connection.end();
