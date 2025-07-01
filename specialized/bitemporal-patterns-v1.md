# Bitemporal with single columns


```sql
-- Can bitemporal be represented purely using two timestamptz?
-- Yes.
-- Bitemporal can be represented using two tstzrange
-- One is valid_period: The date when the fact is valid. It can be the past, present, not necessarily now.
-- The other is the system_period/transaction_period/version_period. It is the date when the entry is created/deleted. It MUST use the present time.
-- Strictly continuous unitemporal can be represented using a single tstzrange column. It simplifies design a lot as compared to using tstzrange, but requires a view to construct the tstzrange equivalent.

-- The valid_at needs to be specified when creating entries.
-- Any update is equivalent to creating a new version - that is, deleted all existing non-deleted entries by primary key, then inserting new entries.
-- For new continuous valid_at entry, just insert a new row.
-- Deleting entries means ending the system period - aka setting deleted_at.
-- Unfortunately this system does not have a way to "end" the valid_at. That means the final entry must be always valid till the end of time.
-- Note that this is not the same as terminating the system period - for e.g. when someone is deceased, there should be a time of death. For this scenario, the time of death/date of birth should be modelled as data and not used as valid_at period.
create table user_vaccination_status (
	id int generated always as identity,
	user_id int not null,
	status text not null,
	vaccine text not null,

	valid_at timestamptz,
	created_at timestamptz not null default now(),
	deleted_at timestamptz,

	primary key (id)
);


-- Update the vaccination status today, user vaccinated on 1st January 2022.
insert into user_vaccination_status (user_id, status, vaccine, valid_at) values
(1, 'VACCINATED', 'Moderna', '2022-01-01 00:00:00');


-- The vaccinate is actually Pfizer ... delete the old record and reinsert;
begin;
update user_vaccination_status set deleted_at = now();
insert into user_vaccination_status (user_id, status, vaccine, valid_at) values
(1, 'VACCINATED', 'Pfizer', '2022-01-01 00:00:00');
commit;

-- The date is actually 1st February ...
begin;
update user_vaccination_status set deleted_at = now() where deleted_at is null;
insert into user_vaccination_status (user_id, status, vaccine, valid_at) values
(1, 'VACCINATED', 'Pfizer', '2022-02-01 00:00:00');
commit;

-- Also received second dose 5 months later in July
insert into user_vaccination_status (user_id, status, vaccine, valid_at) values
(1, 'VACCINATED', 'Pfizer', '2022-07-01 00:00:00');

-- Turns out both is Astra Zeneca.
begin;
update user_vaccination_status set deleted_at = now() where deleted_at is null;
insert into user_vaccination_status (user_id, status, vaccine, valid_at) values
(1, 'VACCINATED', 'Astra Zeneca', '2022-02-01 00:00:00'),
(1, 'VACCINATED', 'Astra Zeneca', '2022-07-01 00:00:00');
commit;

table user_vaccination_status;


with user_vaccination_status_version as (
	select *,
		-- Dense rank has no gap if the values are the same.
		-- Rows deleted at the same time belongs to the same version.
		dense_rank() over (order by deleted_at) as version
	from user_vaccination_status
)
select *,
	tstzrange(
		valid_at,
		lead(valid_at, 1) over (order by valid_at, created_at),
		'[)'
	) as valid_period,
	tstzrange (
		created_at,
		deleted_at,
		'[)'
	) as system_period
from user_vaccination_status_version
where version = (select max(version) from user_vaccination_status_version) - 3
order by valid_at, created_at;

-- What is the vaccination status as of time?
select *,
	tstzrange(created_at, deleted_at, '[)') as system_period
from user_vaccination_status
where tstzrange(created_at, deleted_at, '[)') @> '2022-07-23 15:32:00'::timestamptz;

select *,
	tstzrange(created_at, deleted_at, '[)') as system_period
from user_vaccination_status
where tstzrange(created_at, deleted_at, '[)') @> now();

select *,
	tstzrange(created_at, deleted_at, '[)') as system_period
from user_vaccination_status
where valid_at = '2022-01-01 00:00:00+00';
```
