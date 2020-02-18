# principles

- use singular noun
- ~~use uuid if possible~~
- use ordered UUID v1 (only v1 can be ordered sequentially), stored as BINARY(16) when the data is dynamically generated. For reference table, stick to auto-incremented primary keys since the values are static and won't change that much either. If the number of items can be less than [a-z0-9], then use char(1) as primary key, since they can be more verbose than just int (`m` for male, `f` for female, `o` for others etc) 
- some issues with auto-incremented id is that it needs to be converted to the correct type (int64/uint64) in the application to prevent users from submitting string alphabets. The same complexity lies with UUID, which needs validation too. This is only necessary if we want to avoid the call to the db. Casting the type to `int8` means only up to `127` ids are supported!
- use soft delete
- no null fields, except date

# Notes

- inner joins is faster that subquery most of the time
- apply logic in the database if you are going to have many different applications
- use optimized uuid for faster querying
- include the default auto incremented id

## Styleguides

Yes, there are styleguide (and perhaps linters) for everything.

https://www.sqlstyle.guide/

# Migration file naming convention

Reference the naming convention from active record:

https://edgeguides.rubyonrails.org/active_record_migrations.html#using-the-change-method


## Useful Statements


```sql
-- Sets a default created date
created_at datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP

-- Sets a default updated date, and updates it whenever a row is updated
updated_at datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

-- Adds a new constraint that checks if the first user id is smaller than the second.
CONSTRAINT check_one_way CHECK (user_id1 < user_id2)

-- Adds a constraint that checks if the combination of both columns is unique.
CONSTRAINT uq_user_id_1_user_id_2 UNIQUE (user_id1, user_id2)

-- Sets a foreign key constraint, and updates the foreign key when the primary key changes, or delete the row when it is deleted.
FOREIGN KEY (user_id1) REFERENCES user (id) ON UPDATE CASCADE ON DELETE CASCADE

-- Sets a foreign key constraint, and updates the foreign key when the primary key changes, or set the foreign key to null when it is deleted.
FOREIGN KEY (relationship) REFERENCES ref_relationship (status) ON UPDATE CASCADE ON DELETE SET NULL

-- Create a composite primary key from two columns.
PRIMARY KEY (user_id1, user_id2)
```


Sorting:

```sql
-- Sorting integer string in the correct order
SELECT * FROM <table> ORDER BY CAST(<column> AS unsigned)
```
## Data Type: IPV4 and IPV6

You have two possibilities (for an IPv4 address) :

- a `varchar(15)`, if your want to store the IP address as a string. e.g. `192.128.0.15` for instance
- an `integer (4 bytes)`, if you convert the IP address to an integer. e.g. `3229614095` for the IP I used before

```sql
`ipv4` INT UNSIGNED
INSERT INTO `table` (`ipv4`) VALUES (INET_ATON("127.0.0.1"));
SELECT INET_NTOA(`ipv4`) FROM `table`;
```

```sql
`ipv6` VARBINARY(16)
INSERT INTO `table` (`ipv6`) VALUES (INET6_ATON("127.0.0.1"));
SELECT INET6_NTOA(`ipv6`) FROM `table`;
```

To use a single column for both IPV4 and IPV6:

```sql
CREATE TABLE `sensor` (
  `ip` varbinary(16) NOT NULL DEFAULT '0x'
)
-- Insert IPv6.
insert into sensor (ip) values (INET6_ATON("2001:0db8:85a3:0000:0000:8a2e:0370:7334"));

-- Insert IPv4.
insert into sensor (ip) values (INET6_ATON("255.255.255.0"));

select INET6_NTOA(ip) from sensor;
+------------------------------+
| INET6_NTOA(ip)               |
+------------------------------+
| 2001:db8:85a3::8a2e:370:7334 |
| 255.255.255.0                |
+------------------------------+
2 rows in set (0.00 sec)
```

Sample query from `ipnation`:

```mysql
SELECT c.country 
FROM ip2nationCountries c, ip2nation i 
WHERE i.ip < INET_ATON('your_ip_address') 
AND c.code = i.country 
ORDER BY i.ip DESC 
LIMIT 0,1;
```

References:
- http://www.ip2nation.com/

## Data Type: Country
Al Jumahiriyah al Arabiyah al Libiyah ash Shabiyah al Ishtirakiyah al Uzma also known as Libya is the world's longest country name at 74 characters with spaces and 63 characters without.

```sql
country varchar(74) NOT NULL DEFAULT ''
```

## Data Type: Address

```sql
address_line_1 VARCHAR(255) NOT NULL DEFAULT '',
address_line_2 VARCHAR(255) NOT NULL DEFAULT '',

-- Longest city name: Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch (58 chars.)
city VARCHAR(58) NOT NULL DEFAULT '',

-- Longest state name: The State of Rhode Island and Providence Plantations (52 chars.)
state VARCHAR(56) NOT NULL DEFAULT '',
-- There is no maximum size for a postcode. Currently, the longest postal code is 10 char. Iran has 10 diguts and the US have 4 and 5 seperated by a hyphen. Brazil is 9 and Canada is 7, I think.
postal_code VARCHAR(16) NOT NULL DEFAULT '',
country VARCHAR(74) NOT NULL DEFAULT '',
```

Alternative is this, based on [OpenID AddressClaim](https://openid.net/specs/openid-connect-core-1_0.html#AddressClaim)
```sql
street_address VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Full street address component, which MAY include house number, street name, Post Office Box, and multi-line extended street address information. This field MAY contain multiple lines, separated by newlines. Newlines can be represented either as a carriage return/line feed pair ("\r\n") or as a single line feed character ("\n").'
locality VARCHAR(58) NOT NULL DEFAULT '' COMMENT 'City or locality component.'
region VARCHAR(56) NOT NULL DEFAULT '' COMMENT 'State, province, prefecture or region component.'
postal_code VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'Zip code or postal code component.'
country VARCHAR(74) NOT NULL DEFAULT '' COMMENT 'Country name component.'
-- latitude (See below)
-- longitude
```


References:
- postal code length https://stackoverflow.com/questions/325041/i-need-to-store-postal-codes-in-a-database-how-big-should-the-column-be

## Data Type: URL

```sql
url varchar(2083) NOT NULL DEFAULT '';
```

Checking can be done at the application side. If the length exceeded that of 2083, just warn the client or suggest them to use a url shortener.

References: https://stackoverflow.com/questions/219569/best-database-field-type-for-a-url


## Data Type: Email

https://stackoverflow.com/questions/8242567/acceptable-field-type-and-size-for-email-address

```sql
email VARCHAR(255) NOT NULL UNIQUE
```
## Data Type: Geolocation

With `MySQL <8.0`:
```sql
CREATE TABLE locations (
  lat DECIMAL(10,8) NOT NULL, 
  lng DECIMAL(11,8) NOT NULL
);
```
with `MySQL >8.0`:
```sql
CREATE TABLE locations (
    location POINT SRID 4326 NOT NULL,
    SPATIAL INDEX (location)
);
```

To insert:
```sql
INSERT INTO locations (location) VALUES (ST_PointFromText('Point(1 1)', 4326));
```

To select:
```sql
SELECT ST_AsText(location) FROM locations
```


References:
- https://medium.com/maatwebsite/the-best-way-to-locate-in-mysql-8-e47a59892443

## Data Type: TZ

Max length of 32, longest is `America/Argentina/ComodRivadavia`:

```sql
zoneinfo VARCHAR(32) COMMENT "String from zoneinfo [zoneinfo] time zone database representing the End-User's time zone. For example, Europe/Paris or America/Los_Angeles"
```
References: 
- https://stackoverflow.com/questions/12546312/max-length-of-tzname-field-timezone-identifier-name
- https://www.iana.org/time-zones

## Data Type: Locale

BCP47/RFC5646 section 4.4.1 recommends a 35 characters tag length:

```sql
locale VARCHAR(35) NOT NULL DEFAULT '' COMMENT "End-User's locale, represented as a BCP47 [RFC5646] language tag. This is typically an ISO 639-1 Alpha-2 [ISO639?1] language code in lowercase and an ISO 3166-1 Alpha-2 [ISO3166?1] country code in uppercase, separated by a dash. For example, en-US or fr-CA. As a compatibility note, some implementations have used an underscore as the separator rather than a dash, for example, en_US",
```

References:
- https://stackoverflow.com/questions/17848070/what-data-type-should-i-use-for-ietf-language-codes
- https://openid.net/specs/openid-connect-core-1_0.html#zoneinfo

## Data Type: Phone number

```sql
phone_number VARCHAR(32) NOT NULL DEFAULT '',
phone_number_verified BOOLEAN NOT NULL DEFAULT 0,
```
References:
- https://en.wikipedia.org/wiki/Telephone_numbering_plan
- https://boards.straightdope.com/sdmb/showthread.php?t=417024
https://stackoverflow.com/questions/723587/whats-the-longest-possible-worldwide-phone-number-i-should-consider-in-sql-varc

## Data Type: Name
Longest name (225 characters)
```
Barnaby Marmaduke Aloysius Benjy Cobweb Dartagnan Egbert Felix Gaspar Humbert Ignatius Jayden Kasper Leroy Maximilian Neddy Obiajulu Pepin Quilliam Rosencrantz Sexton Teddy Upwood Vivatma Wayland Xylon Yardley Zachary Usansky
```

References:
- http://www.worldrecordacademy.com/society/longest_name_Barnaby_Marmaduke_sets_world_record_112063.html

## Data Type: Gender

Column naming can be `sex`, or alternatively `gender`:

```sql
-- Probably the best bet, but needs to be validated. When in doubt, use this.
sex char(1) 
insert into table (gender) values (IF(LEFT(?,1) in ('m', 'f', 'x', 'o'), LOWER(LEFT(?,1)), ''));
-- We can also just take the first character of the string with the left function.
insert into table(gender) values (LEFT('female', 1));

-- With enum. Allows only 'm', 'f', 'M', or 'F'. Don't use enum - it will rebuild the whole database when we update it.
sex enum('m','f') DEFAULT 'm' 

-- With set.
sex set('m', 'f') // Allows '', 'm', 'M', 'f', 'F', or 'm,f'
```


References:
- http://komlenic.com/244/8-reasons-why-mysqls-enum-data-type-is-evil/
- http://download.nust.na/pub6/mysql/tech-resources/articles/mysql-set-datatype.html
- https://ocelot.ca/blog/blog/2013/09/16/representing-sex-in-databases/

Note: We could have used check constraint, but it is ignored by MySQL.

## Data Type: Currency

compliant with Generally Accepted Accounting Principles (GAAP):

```sql
currency DECIMAL(13,4)
```

For percentage:

```sql
-- For your case (0.00% to 100.00%) you'd want decimal(5,4).
gst DECIMAL(5, 4)

-- For the most common case (0% to 100%) you'd want decimal(3,2).
discount DECIMAL(3, 2)
```


## Marital Status

| Code | Description | Definition | 
| ---- | ------------| -----------|
| 1 | Single | This refers to a person who has never been married. |
| 2 | Married | This refers to a person who is recognised as married under the marriage laws in Singapore. It includes a person who has remarried. |
| 3 | Widowed | This refers to a person whose spouse(s) is/are deceased and who has not remarried. |
| 4 | Separated | This refers to a person who has been legally separated or estranged from his/her spouse(s) and who has not remarried. |
| 5 | Divorced | This refers to a person whose marriage(s) has/have been legally dissolved and who has not remarried. |
| x | Not Reported | This includes instances where the marital status is unknown, not reported or where there is no/insufficient information available on the marital status |

```sql
marital_status ENUM('single', 'married', 'widowed', 'separated', 'divorced', 'not reported');
```
References:
- https://www.singstat.gov.sg/-/media/files/standards_and_classifications/scms.pdf

## References
- Gender X: https://www.lifesitenews.com/news/generation-x-germany-to-allow-third-blank-gender-for-birth-certificates

## One-to-One Relationship

Example of 1-to-1 relationship between `user` and `preference` table:

```
CREATE TABLE IF NOT EXISTS user (
  name VARCHAR(255),
  id INT UNSIGNED AUTO_INCREMENT,
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS preference (
  user_id INT UNSIGNED AUTO_INCREMENT,
  interest TEXT,
  -- ...other fields
  PRIMARY KEY (id),
  FOREIGN KEY (id) REFERENCES user(id)
);
```



## Country Table

https://www.ip2location.com/free/country-multilingual


## Issues

There was a scenario where User A is logged in User B account (bug), the reason is very simple. User A previous user `id` is 2, and the JWT token is not expired. When the db was cleared, and User B recreated two new users, which has user id `1` and `2`. So now, the JWT token only store the user id `2`, when User A calls the API, since the previous id is `2`, User A can view User B profile. Pitfalls of integer id.

## Thoughts

- Should I create a differen table for user profile and password? No.
https://stackoverflow.com/questions/17683571/should-i-create-2-tables-first-for-usernames-and-passwords-and-other-for-user
https://www.quora.com/Should-we-keep-the-user-name-and-password-in-the-same-table-where-the-other-personal-information-is
https://dba.stackexchange.com/questions/148909/is-it-a-good-practice-to-isolate-login-information-username-password-in-a-sep


## Useful features for postgres

- lateral join
- sum conditional
- filter option
- RANK, DENSE_RANK, ROW_NUMBER
- partial index

```sql
select id, name, 
	review_count_rank,
	recent_review_count_rank,
	rating_rank,
	word_count_rank,
	review_count_rank + recent_review_count_rank + rating_rank + word_count_rank as total
from (
	select 
		product_items.id as id,
		product_items.name,
		
	--	product_items.cached_reviews_count,
		DENSE_RANK() OVER (order by product_items.cached_reviews_count desc) review_count_rank,
		
	--	COALESCE(tmp.review_count, 0) as recent_review_count,
		DENSE_RANK() OVER (order by COALESCE(tmp.review_count, 0) desc) recent_review_count_rank,
		
	--	product_items.cached_rating,
		DENSE_RANK() OVER (order by product_items.cached_rating desc) rating_rank,
		
	--	COALESCE(tmp.word_count, 0) as word_count,
		DENSE_RANK() OVER (order by COALESCE(tmp.word_count,0) desc) word_count_rank
	from product_items 
	left join (
		select 
			pir.item_id as item_id, 
			count(*) as review_count,
			sum(array_length(regexp_split_to_array(pir.text, '\s'),1)) as word_count
		from product_items pi
		left join product_item_reviews pir
			on (pi.id = pir.item_id)
		where pir.deleted_at is null
			and pi.deleted_at is null
			and pir.created_at >= current_timestamp - interval '30 day'
		group by pir.item_id
	) tmp on (tmp.item_id = product_items.id)
	where category_id = 6
		and product_items.deleted_at is null
	order by review_count_rank, 
		rating_rank,
		recent_review_count_rank,
		word_count_rank
	) tmp
order by total;
```
