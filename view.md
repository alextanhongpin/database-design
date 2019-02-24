## Using view for multiple tables

```sql
create table test_user (
	id int auto_increment,
	name varchar(255),
	primary key (id)
);

create table test_address (
	id int auto_increment,
	country varchar(255),
	test_user_id int not null,
	foreign key (test_user_id) references test_user(id),
	primary key (id)
);


create view test_user_address AS
	select test_user.id as user_id, name, ta.id as address_id, ta.country as country from test_user inner join test_address ta on (test_user.id = ta.test_user_id);

insert into test_user (name) values ('John');

insert into test_address(country, test_user_id) values ("Country", 1);

select * from test_user_address;

-- Can not modify more than one base table through a join view 'employee_benefit.test_user_address'
update test_user_address set name = 'Johny', country = "New Country";
update test_user_address set country = "New Country";
update test_user_address set name = "Johny";

drop table test_address, test_user;
drop view test_user_address;
```
