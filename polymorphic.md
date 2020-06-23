## Using Check Constraint to guard against polymorphic association

```sql
create table test_subscription (
	type text check (type in ('question', 'answer', 'comment')),
	answer_id uuid references answer(id),
	question_id uuid references question(id),
	comment_id uuid references comment(id),
	check ((type = 'question' and question_id is not null) or (type = 'answer' and answer_id is not null) or (type = 'comment' and comment_id is not null))
);

-- Modifying check constraint.
alter table test_subscription 
drop constraint test_subscription_check, 
add constraint test_subscription_check 
check ((type = 'question' and question_id is not null) or (type = 'answer' and answer_id is not null) or (type = 'comment' and comment_id is not null));
```
