# Generating ordered UUID v1


## Deprecated, don't follow
Random produces very fragmented inserts that destroy tables. Use `uuid_generate_v1mc()` [instead] ... the keys are seq because they're time based. So all inserts go to the same data page without random io.

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS "uuid-ossp";

SELECT uuid_generate_v1();
SELECT uuid_generate_v4();

-- Recommended!
SELECT uuid_generate_v1mc();

-- Not recommended!
CREATE EXTENSION pgcrypto;
select gen_random_uuid();
```

## New


- [Postgres 13](https://www.postgresql.org/docs/13/functions-uuid.html) introduces a native uuid function called `gen_random_uuid` that generates v4 uuid. Use this instead.
- uuid column in Postgres is native is stored as 16 bytes - it's not the same as mysql that stores as 32 bytes
- casing is not important, so the example below is true
-
```sql
select upper('afe05298-4bf1-4ea2-ae6f-752aa005895a')::uuid = 'afe05298-4bf1-4ea2-ae6f-752aa005895a'::uuid;
```

- there's no claims that using uuidv1 is better than uuidv4, see [here](https://www.postgresql.org/message-id/20151222124018.bee10b60b3d9b58d7b3a1839%40potentialtech.com)
- uuid performance is definitely better than mysql, see [here](https://blog.josephscott.org/2005/12/08/mysql-to-postgresql-and-uuidguids/)
- you can cast int to uuid, but this is a [poor example](https://cleanspeak.com/blog/2015/09/23/postgresql-int-to-uuid)
- instead, use uuid v5

```sql
create extension "uuid-ossp";

-- ns_base
select uuid_generate_v5(uuid_ns_dns(), 'stackoverflow.com'); -- cd84c40a-6019-50c7-87f7-178668ab9c8b

-- ns_user
select uuid_generate_v5('cd84c40a-6019-50c7-87f7-178668ab9c8b', 'user'); -- f9bc56cc-8023-5a98-9d89-3986d5a1e5e1
select uuid_generate_v5('f9bc56cc-8023-5a98-9d89-3986d5a1e5e1', '1');


-- ns_question
select uuid_generate_v5('cd84c40a-6019-50c7-87f7-178668ab9c8b', 'question'); -- 7315c970-1b65-5924-ad72-6b966025477c

-- ns_answer
select uuid_generate_v5('cd84c40a-6019-50c7-87f7-178668ab9c8b', 'answer'); -- db1e4c3f-8ff7-5bfe-a508-90c950e23ff3

-- ns_comment
select uuid_generate_v5('cd84c40a-6019-50c7-87f7-178668ab9c8b', 'comment'); -- c6c70eea-d74f-5777-8e9d-223484338c96
```


The implementation should produce similar results to golang code:


```go
// You can edit this code!
// Click here and start typing.
package main

import (
	"fmt"

	"github.com/google/uuid"
)

func main() {
	nsBase := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("stackoverflow.com"))
	nsUser := uuid.NewSHA1(nsBase, []byte("user"))
	nsQuestion := uuid.NewSHA1(nsBase, []byte("question"))
	nsAnswer := uuid.NewSHA1(nsBase, []byte("answer"))
	fmt.Println(nsBase)     // cd84c40a-6019-50c7-87f7-178668ab9c8b
	fmt.Println(nsQuestion) // 7315c970-1b65-5924-ad72-6b966025477c
	fmt.Println(nsAnswer)   // db1e4c3f-8ff7-5bfe-a508-90c950e23ff3

	nsUser1 := uuid.NewSHA1(nsUser, []byte("1"))
	fmt.Println(nsUser1) // cb890694-4c46-5e64-be6d-09ce05f01664
}
```





## References

- https://www.2ndquadrant.com/en/blog/sequential-uuid-generators/
- https://stackoverflow.com/questions/34230208/uuid-primary-key-in-postgres-what-insert-performance-impact
- https://dzone.com/articles/store-uuid-optimized-way
- https://starkandwayne.com/blog/uuid-primary-keys-in-postgresql/
