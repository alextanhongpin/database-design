# Partition by date range

```sql
CREATE TABLE invoices (
	invoice_number int NOT NULL,
	issued_on date NOT NULL DEFAULT now()
) PARTITION BY RANGE(issued_on);

-- Table for the month of May 2018.
CREATE TABLE invoices_2018_05 
PARTITION OF invoices
FOR VALUES FROM ('2018-05-01') TO ('2018-06-01');

-- Table for the month of June 2018.
CREATE TABLE invoices_2018_06
PARTITION OF invoices
FOR VALUES FROM ('2018-06-01') TO ('2018-07-01');

INSERT INTO invoices 
VALUES 
	(10042, '2018-05-15'),
	(43029, '2018-06-15');
```
