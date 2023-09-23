import { faker } from "@faker-js/faker";

const USERS = "users";
const ACCOUNTS = "accounts";

class Service {
  MAX_BATCH_SIZE = 1000;

  sql = {
    [USERS]: `
      create table if not exists users (
        id int auto_increment,
        name varchar(80) not null,
        email varchar(80) not null,
        primary key (id),
        unique(email)
      )
    `,
    [ACCOUNTS]: `
      create table if not exists accounts (
        id int auto_increment,
        name varchar(80) not null,
        email varchar(80) not null,
        created_at timestamp not null default now(),
        updated_at timestamp not null default now(),
        primary key (id)
      )
    `,
  };

  constructor(db, ns) {
    if (!this.sql[ns]) {
      throw new Error(`unknown namespace: ${ns}`);
    }
    this.db = db;
    this.ns = ns;
    this.seeder = {
      [USERS]: (n) => this.seedUsers(n),
      [ACCOUNTS]: (n) => this.seedAccounts(n),
    };
  }

  async migrate() {
    const sql = this.sql[this.ns];
    return this.db.execute(sql);
  }

  async seed(n = 1_000) {
    const fn = this.seeder[this.ns];
    const sizes = batch(n, this.MAX_BATCH_SIZE);
    return Promise.all(sizes.flatMap((n) => fn(n)));
  }

  async seedUsers(n) {
    const names = Array.from({ length: n }).map(() => [
      faker.person.fullName(),
      faker.internet.email(),
    ]);
    return this.db.query(
      `
      insert into users(name, email) values ?
    `,
      [names]
    );
  }

  async seedAccounts(n) {
    const names = Array.from({ length: n }).map(() => [
      faker.person.fullName(),
      faker.internet.email(),
    ]);
    return this.db.query(
      `
      insert into accounts(name, email) values ?
    `,
      [names]
    );
  }
}
function batch(n, size) {
  const result = [];
  while (n > 0) {
    result.push(Math.min(size, n));
    n -= size;
  }
  return result;
}

export default function createService(conn, ns) {
  return new Service(conn, ns);
}
