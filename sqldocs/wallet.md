# Wallet Domain

How to design the schema for a virtual wallet that can do withdrawal and deposits?

Deposit is when money is added to the wallet.

Withdrawal is when money is deducted from the wallet.

This is reverse-engineering the database schema from [laravel-wallet](https://bavix.github.io/laravel-wallet/#/README).

## Tables

TODO: Reference external source.

### Wallets

Wallets stores the User wallet balances.

**Columns**
- id
- holder_id
- holder_type
- name
- slug
- uuid
- description
- meta
- balance
- decimal places

Rules
- Rule: Unique holder id, holder type and slug


### Transactions


**Columns**
- id
- payable_id
- payable_type
- wallet id
- type: withdraw/deposit
- amount
- confirmed
- meta
- uuid
- created_at
- updated_at

### Transfers

Transfer takes place between wallets, when a Deposit and Withdraw operations are performed in one database transaction.

**Columns**
- id
- from_id
- from_type
- to_id
- to_type
- status: exchange|transfer|paid|refund|gift
- status_last
- deposit_id fk transactions.id
- withdraw_id fk transactions.id
- discount
- fee
- created_at
- updated_at

## ER Diagram

<insert mermaid erDiagram here>


## Queries

### Get confirmed balance by user
```sql
select sum(amount), payable_id, payable_type
from transactions
where confirmed
group by payable_id, payable_type
```

### Get transactions by user

```sql
select sum(amount), payable_id, payable_type, confirmed
from transactions
group by payable_id, payable_type, confirmed
```

## Mutations

<insert mutations here>

### Create New Wallet

```sql
INSERT INTO wallets (
  holder_id,
  holder_type,
  name,
  slug,
  uuid,
  description,
  meta,
  balance,
  decimal_places
) VALUES (
  1, -- holder_id,
  'App\Models\User', -- holder_type,
  'Default Wallet', -- name,
  'default', -- slug,
  'f66f4392-6b3b-414e-be08-ea0b1d7ce781', -- uuid,
  '', -- description,
  '{}', -- meta,
  0, -- balance,
  2, -- decimal_places
)
```

```psql
-[ RECORD 2 ]--+-------------------------------------
id             | 1
holder_type    | App\Models\User
holder_id      | 1
name           | Default Wallet
slug           | default
uuid           | f66f4392-6b3b-414e-be08-ea0b1d7ce781
description    |
meta           | []
balance        | 0
decimal_places | 2
created_at     | 2023-05-18 16:42:55
updated_at     | 2023-05-18 16:43:27
```

### Create Deposit

Deposit `10` cents to User with `id=1`.
```sql
-- Update wallet balance.
UPDATE wallets
SET balance = balance + 10
WHERE
  holder_type = 'App\Models\User' AND
  holder_id = 1;

-- Create transactions.
INSERT INTO transactions (
  payable_type,
  payable_id,
  wallet_id,
  type,
  amount,
  confirmed,
  meta,
  uuid
) VALUES (
  'App\Models\User', -- payable_type,
  1, -- payable_id,
  1, -- wallet_id,
  'deposit', -- type,
  10, -- amount,
  t, -- confirmed,
  '{}', -- meta,
  '30bee122-ca1b-45e6-a822-863a22ce6c0a', -- uuid
)
```

```psql
laravel=# select * from transactions;
-[ RECORD 1 ]+-------------------------------------
id           | 1
payable_type | App\Models\User
payable_id   | 1
wallet_id    | 1
type         | deposit
amount       | 10
confirmed    | t
meta         |
uuid         | 30bee122-ca1b-45e6-a822-863a22ce6c0a
created_at   | 2023-05-18 16:43:27
updated_at   | 2023-05-18 16:43:27
```

### Withdraw

User withdraws `10` cents from the wallet:

```psql
laravel=# table transactions;
-[ RECORD 2 ]+-------------------------------------
id           | 2
payable_type | App\Models\User
payable_id   | 1
wallet_id    | 1
type         | withdraw
amount       | -10
confirmed    | t
meta         |
uuid         | 7080ed88-475e-434f-840a-6868641394fe
created_at   | 2023-05-18 17:02:25
updated_at   | 2023-05-18 17:02:25
```

The balance was previously 10, it is now 0.

```psql
laravel=# table wallets;
-[ RECORD 2 ]--+-------------------------------------
id             | 1
holder_type    | App\Models\User
holder_id      | 1
name           | Default Wallet
slug           | default
uuid           | f66f4392-6b3b-414e-be08-ea0b1d7ce781
description    |
meta           | []
balance        | 0
decimal_places | 2
created_at     | 2023-05-18 16:42:55
updated_at     | 2023-05-18 17:02:25
```

### Transfer

Deposit 10 cents to user 1, then transfer from user 1 to user 3.

```sql
laravel=# table transactions;
-[ RECORD 3 ]+-------------------------------------
id           | 3
payable_type | App\Models\User
payable_id   | 1
wallet_id    | 1
type         | deposit
amount       | 10
confirmed    | t
meta         |
uuid         | 0b2be398-83c4-4166-a249-92f833742476
created_at   | 2023-05-18 17:11:04
updated_at   | 2023-05-18 17:11:04
-[ RECORD 4 ]+-------------------------------------
id           | 4
payable_type | App\Models\User
payable_id   | 1
wallet_id    | 1
type         | withdraw
amount       | -10
confirmed    | t
meta         |
uuid         | 9b31008e-6dfe-4a79-ba36-3914aeeb1422
created_at   | 2023-05-18 17:11:04
updated_at   | 2023-05-18 17:11:04
-[ RECORD 5 ]+-------------------------------------
id           | 5
payable_type | App\Models\User
payable_id   | 3
wallet_id    | 2
type         | deposit
amount       | 10
confirmed    | t
meta         |
uuid         | 462c6982-70e3-4365-86a0-82e65d39cb16
created_at   | 2023-05-18 17:11:04
updated_at   | 2023-05-18 17:11:04
```

`transfers` table:

```sql
laravel=# table transfers;
-[ RECORD 1 ]-------------------------------------
id          | 1
from_type   | Bavix\Wallet\Models\Wallet
from_id     | 1
to_type     | Bavix\Wallet\Models\Wallet
to_id       | 2
status      | transfer
status_last |
deposit_id  | 5
withdraw_id | 4
discount    | 0
fee         | 0
uuid        | f8883d64-fecd-4758-a39a-920588bf7cdc
created_at  | 2023-05-18 17:11:04
updated_at  | 2023-05-18 17:11:04
```

`wallets` table:

```sql
laravel=# table wallets;
-[ RECORD 1 ]--+-------------------------------------
id             | 2
holder_type    | App\Models\User
holder_id      | 3
name           | Default Wallet
slug           | default
uuid           | 5a5586ee-8ce7-4555-80d2-f3cbd350ff6e
description    |
meta           | []
balance        | 10
decimal_places | 2
created_at     | 2023-05-18 16:42:55
updated_at     | 2023-05-18 17:11:04
-[ RECORD 2 ]--+-------------------------------------
id             | 1
holder_type    | App\Models\User
holder_id      | 1
name           | Default Wallet
slug           | default
uuid           | f66f4392-6b3b-414e-be08-ea0b1d7ce781
description    |
meta           | []
balance        | 0
decimal_places | 2
created_at     | 2023-05-18 16:42:55
updated_at     | 2023-05-18 17:11:04
```


## Business Rules

### Rule: Rule A

<insert rules here>

### Rule: Rule B
