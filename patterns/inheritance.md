# Postgres Inheritance

```sql
CREATE TABLE IF NOT EXISTS invoices (
	invoice_number int NOT NULL PRIMARY KEY,
	issued_on date NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS government_invoices (
	department_id text NOT NULL
) INHERITS (invoices);

INSERT INTO invoices (invoice_number) VALUES (100);
INSERT INTO government_invoices(invoice_number, department_id) VALUES (101, 'DOD');

SELECT * FROM invoices;
SELECT * FROM government_invoices;
```
