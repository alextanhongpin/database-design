Workflow Pattern

```sql
DROP DATABASE test;
CREATE DATABASE test;
USE test;

CREATE TABLE IF NOT EXISTS your_entity_to_manage (
	id INT UNSIGNED AUTO_INCREMENT,
	your_col_1 INT NOT NULL DEFAULT 0,
	your_col_2 VARCHAR(255) NOT NULL DEFAULT "",
	your_col_3_etc BOOLEAN NOT NULL DEFAULT 0,
	-- This is not strictly a foreign key.
	wf_state_type_process_id INT UNSIGNED,
	PRIMARY KEY (id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS workflow_level_type (
	id INT UNSIGNED AUTO_INCREMENT,
	alt_sequence INT UNSIGNED NOT NULL DEFAULT 0,
	description TEXT,
	effective_period_from DATE NOT NULL DEFAULT '1000-01-01',
	effective_period_to DATE NOT NULL DEFAULT '9999-12-31',
	pretty_name VARCHAR(255) NOT NULL DEFAULT '',
	-- This should be unique.
	type_key VARCHAR(32) NOT NULL DEFAULT '',
	PRIMARY KEY (id),
	UNIQUE (type_key)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO workflow_level_type 
(type_key, description) VALUES
('process', 'High level workflow process.'),
('state', 'A state in the process.'),
('outcome', 'How a state ends, its outcome.'),
('qualifier', 'An optional, more detailed qualifier for an outcome.');

CREATE TABLE IF NOT EXISTS workflow_state_type (
	id INT UNSIGNED AUTO_INCREMENT,
	alt_sequence INT UNSIGNED NOT NULL DEFAULT 0,
	description TEXT,
	effective_period_from DATE NOT NULL DEFAULT '1000-01-01',
	effective_period_to DATE NOT NULL DEFAULT '9999-12-31',
	pretty_name VARCHAR(255) NOT NULL DEFAULT '',
	type_key VARCHAR(32) NOT NULL DEFAULT '',
	workflow_level_type_id INT UNSIGNED NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (workflow_level_type_id) REFERENCES workflow_level_type(id),
	UNIQUE (workflow_level_type_id, type_key)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO workflow_state_type
(workflow_level_type_id, type_key) VALUES 
-- 3: Outcomes
(3, 'passed'),
(3, 'failed'),
(3, 'accepted'),
(3, 'declined'),
(3, 'candidate_cancelled'),
(3, 'employer_cancelled'),
(3, 'rejected'),
(3, 'employer_withdrawn'),
(3, 'no_show'),
(3, 'hired'),
(3, 'not hired'),
-- 2: State
(2, 'application_received'),
(2, 'application_review'),
(2, 'invited_to_interview'),
(2, 'interview'),
(2, 'test_aptitude'),
(2, 'seek_references'),
(2, 'make_offer'),
(2, 'application_closed'),
-- 1: Process
(1, 'standard_job_application'),
(1, 'technical_job_application');

CREATE TABLE IF NOT EXISTS workflow_state_hierachy (
	id INT UNSIGNED AUTO_INCREMENT,
	alt_sequence INT UNSIGNED NOT NULL DEFAULT 0,
	wf_state_type_parent_id INT UNSIGNED NOT NULL,
	wf_state_type_child_id INT UNSIGNED NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (wf_state_type_parent_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (wf_state_type_child_id) REFERENCES workflow_state_type(id),
	UNIQUE KEY (wf_state_type_parent_id, wf_state_type_child_id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Each state must end with an outcome.
INSERT INTO workflow_state_hierachy 
(wf_state_type_parent_id, wf_state_type_child_id) VALUES
(12, 3), -- application_received, accepted
(12, 7), -- application_received, rejected
(13, 1), -- application_review, passed
(13, 2), -- application_review, failed,
(14, 3), -- invited_to_interview, accepted
(14, 4), -- invited_to_interview, declined
(15, 1), -- interview, passed
(15, 2), -- interview, failed
(15, 5), -- interview, candidate_cancelled
(15, 9), -- interview, no show
(18, 3), -- make_offer, accepted
(18, 4), -- make_offer, declined
(17, 1), -- seek_references, passed
(17, 2), -- seek_references, failed
(19, 10), -- application_closed, hired
(19, 11), -- application_closed, not_hired
(16, 1), -- test_aptitude, passed
(16, 2), -- test_aptitude, failed
(20, 12), -- standard_job_application, application_received
(20, 13), -- standard_job_application, application_review
(20, 14), -- standard_job_application, invited_to_interview
(20, 15), -- standard_job_application, interview
(20, 18), -- standard_job_application, make offer
(20, 17), -- standard_job_application, seek references
(20, 19), -- standard_job_application, application closed
(21, 12), -- technical_job_application, application_received
(21, 13), -- technical_job_application, application_review
(21, 14), -- technical_job_application, invited_to_interview
(21, 15), -- technical_job_application, interview
(21, 18), -- technical_job_application, make offer
(21, 17), -- technical_job_application, seek references
(21, 19) -- technical_job_application, application closed
;

CREATE TABLE IF NOT EXISTS managed_entity_state (
	id INT UNSIGNED AUTO_INCREMENT,
	due_date DATE NOT NULL DEFAULT '1000-01-01',
	effective_period_from DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	effective_period_to DATETIME NOT NULL DEFAULT '9999-12-31',
	notes TEXT,
	managed_entity_id INT UNSIGNED NOT NULL,
	wf_state_type_state_id INT UNSIGNED NOT NULL,
	wf_state_type_outcome_id INT UNSIGNED NOT NULL,
	wf_state_type_qual_id INT UNSIGNED NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (wf_state_type_state_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (wf_state_type_outcome_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (wf_state_type_qual_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (managed_entity_id) REFERENCES your_entity_to_manage(id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;



CREATE TABLE IF NOT EXISTS workflow_state_context (
	id INT UNSIGNED AUTO_INCREMENT,
	child_disabled BOOLEAN NOT NULL DEFAULT 0,
	workflow_state_type_id INT UNSIGNED NOT NULL,
	workflow_state_hierachy_id INT UNSIGNED NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (workflow_state_type_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (workflow_state_hierachy_id) REFERENCES workflow_state_hierachy(id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO workflow_state_context
(workflow_state_type_id, workflow_state_hierachy_id) VALUES 
(20, 1), -- standard_application, application_received.accepted
(20, 2), -- standard_application, application_received.rejected

(20, 3), -- standard_application, application_review.passed
(20, 4), -- standard_application, application_review.failed
(20, 5), -- standard_application, invited_to_interview.accepted
(20, 6), -- standard_application, invited_to_interview.declined
(20, 7), -- standard_application, interview.passed
(20, 8), -- standard_application, interview.failed
(20, 9), -- standard_application, interview.candidate_cancelled
(20, 10), -- standard_application, interview.no_show
(20, 11), -- standard_application, make_offer.accepted
(20, 12), -- standard_application, make_offer.declined
(20, 13), -- standard_application, seek_references.passed
(20, 14), -- standard_application, seek_references.failed
(20, 15), -- standard_application, application_closed.hired
(20, 16) -- standard_application, application_closed.not_hired
;

CREATE TABLE IF NOT EXISTS workflow_state_option (
	id INT UNSIGNED AUTO_INCREMENT,
	alt_sequence INT UNSIGNED NOT NULL DEFAULT 0,
	workflow_state_context_id INT UNSIGNED NOT NULL,
	workflow_state_type_id INT UNSIGNED NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (workflow_state_context_id) REFERENCES workflow_state_context(id),
	FOREIGN KEY (workflow_state_type_id) REFERENCES workflow_state_type(id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO workflow_state_option 
(workflow_state_context_id, workflow_state_type_id)
(1, 13), -- standard_application, application_received.accepted, option: application_review
(2, 16), -- standard_application, application_received.rejected, option: application_closed.not_hired

(3, 14), -- standard_application, application_review.passed, option: invited_to_interview
(4, 16), -- standard_application, application_review.failed, option: application_closed.not_hired
(5, 15), -- standard_application, invited_to_interview.accepted, option: interview
(6, 16), -- standard_application, invited_to_interview.declined, option: application_closed.not_hired
(7, 18), -- standard_application, interview.passed, option: make_offer
(7, 17), -- standard_application, interview.passed, option: seek_references
(8, 19), -- standard_application, interview.failed, option: application_closed
(9, 19), -- standard_application, interview.candidate_cancelled, option: application_closed
(9, 14), -- standard_application, interview.candidate_cancelled, option: invited_to_interview
(10, 19), -- standard_application, interview.no_show, option: application_closed
(11, 17), -- standard_application, make_offer.accepted, option: seek_references
(12, 19), -- standard_application, make_offer.declined, option: application_closed
(13, 15), -- standard_application, seek_references.passed, option: application_closed.hired
(14, 19), -- standard_application, seek_references.failed, option: application_closed
(15, 15), -- standard_application, application_closed.hired
(16, 16) -- standard_application, application_closed.not_hired
;
```


REFERENCES: 
- https://www.vertabelo.com/blog/technical-articles/the-workflow-pattern-part-1-using-workflow-patterns-to-manage-the-state-of-any-entity
- https://www.vertabelo.com/blog/technical-articles/the-workflow-pattern-part-2-using-configuration-tables-to-define-the-actual-workflow
