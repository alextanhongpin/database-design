import { faker } from "@faker-js/faker";

class Service {
  constructor(db) {
    this.db = db;
  }

  async count() {
    const [rows] = await this.db.query("select 1 + 1 as solution");
    return rows;
  }

  async migrate() {
    return this.db.execute(
      `create table if not exists users (
        id int auto_increment,
        name varchar(80) not null,
        email varchar(80) not null,
        primary key (id),
        unique(email)
      )`
    );
  }

  async seed(n = 1_000) {
    const sizes = batch(n, 1000);
    return Promise.all(sizes.flatMap((n) => this._seed(n)));
  }

  async _seed(n) {
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
}
function batch(n, size) {
  const result = [];
  while (n > 0) {
    result.push(Math.min(size, n));
    n -= size;
  }
  return result;
}

export default function createService(conn) {
  return new Service(conn);
}
