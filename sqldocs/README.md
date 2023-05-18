# SQL Docs

Experimenting with a consistent format for documenting SQL code.

## Principles

- group by feature/domain
  - document a set of related tables, and their relationships
- describe business rules
  - formatting
  - constraints
  - trigger
  - validation

- queries
  - views and analytics
  - indices

## Structure


```markdown
# Domain Name

Some description about the domain

## Tables

<insert migration script here>

### Table 1

<insert description of table>

Columns
- column1
  - Rule: Rule A
  - Rule: Rule B
  - Rule: Rule Z
- column2
- columnn


### Table 2
### Table N

## ER Diagram

<insert mermaid erDiagram here>


## Queries

<insert queries here>

## Mutations

<insert mutations here>


## Business Rules

### Rule: Rule A

<insert rules here>

### Rule: Rule B
```
