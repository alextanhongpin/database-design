## Workflow Pattern

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

## Workflow Pattern, with String ID

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
	id VARCHAR(255),
	alt_sequence INT UNSIGNED NOT NULL DEFAULT 0,
	description TEXT,
	effective_period_from DATE NOT NULL DEFAULT '1000-01-01',
	effective_period_to DATE NOT NULL DEFAULT '9999-12-31',
	pretty_name VARCHAR(255) NOT NULL DEFAULT '',
	PRIMARY KEY (id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO workflow_level_type 
(id, description) VALUES
('process', 'High level workflow process.'),
('state', 'A state in the process.'),
('outcome', 'How a state ends, its outcome.'),
('qualifier', 'An optional, more detailed qualifier for an outcome.');

CREATE TABLE IF NOT EXISTS workflow_state_type (
	id VARCHAR(255),
	alt_sequence INT UNSIGNED NOT NULL DEFAULT 0,
	description TEXT,
	effective_period_from DATE NOT NULL DEFAULT '1000-01-01',
	effective_period_to DATE NOT NULL DEFAULT '9999-12-31',
	pretty_name VARCHAR(255) NOT NULL DEFAULT '',
	type_key VARCHAR(32) NOT NULL DEFAULT '',
	workflow_level_type_id VARCHAR(255) NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (workflow_level_type_id) REFERENCES workflow_level_type(id),
	UNIQUE (id, workflow_level_type_id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO workflow_state_type
(workflow_level_type_id, id) VALUES 
-- 3: Outcomes
('outcome', 'passed'),
('outcome', 'failed'),
('outcome', 'accepted'),
('outcome', 'declined'),
('outcome', 'candidate_cancelled'),
('outcome', 'employer_cancelled'),
('outcome', 'rejected'),
('outcome', 'employer_withdrawn'),
('outcome', 'no_show'),
('outcome', 'hired'),
('outcome', 'not_hired'),
-- 2: State
('state', 'application_received'),
('state', 'application_review'),
('state', 'invited_to_interview'),
('state', 'interview'),
('state', 'test_aptitude'),
('state', 'seek_references'),
('state', 'make_offer'),
('state', 'application_closed'),
('state', 'end'),
-- 1: Process
('process', 'standard_job_application'),
('process', 'technical_job_application');

CREATE TABLE IF NOT EXISTS workflow_state_hierachy (
	id VARCHAR(255),
	alt_sequence INT UNSIGNED NOT NULL DEFAULT 0,
	wf_state_type_parent_id VARCHAR(255) NOT NULL,
	wf_state_type_child_id VARCHAR(255) NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (wf_state_type_parent_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (wf_state_type_child_id) REFERENCES workflow_state_type(id),
	UNIQUE KEY (wf_state_type_parent_id, wf_state_type_child_id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;


-- Each state must end with an outcome.
INSERT INTO workflow_state_hierachy 
(id, wf_state_type_parent_id, wf_state_type_child_id) VALUES
('application_received.accepted', 'application_received', 'accepted'),
('application_received.rejected', 'application_received', 'rejected'),
('application_review.passed', 'application_review', 'passed'),
('application_review.failed', 'application_review', 'failed'), 
('invited_to_interview.accepted', 'invited_to_interview', 'accepted'),
('invited_to_interview.declined', 'invited_to_interview', 'declined'),
('interview.passed', 'interview', 'passed'),
('interview.failed', 'interview', 'failed'),
('interview.candidate_cancelled', 'interview', 'candidate_cancelled'),
('interview.no_show', 'interview', 'no_show'),
('make_offer.accepted', 'make_offer', 'accepted'),
('make_offer.declined', 'make_offer', 'declined'),
('seek_references.passed', 'seek_references', 'passed'),
('seek_references.failed', 'seek_references', 'failed'),
('application_closed.hired', 'application_closed', 'hired'),
('application_closed.not_hired', 'application_closed', 'not_hired'),
('test_aptitude.passed', 'test_aptitude', 'passed'),
('test_aptitude.failed', 'test_aptitude', 'failed'),
('standard_job_application.application_received', 'standard_job_application', 'application_received'),
('standard_job_application.application_review', 'standard_job_application', 'application_review'),
('standard_job_application.invited_to_interview', 'standard_job_application', 'invited_to_interview'),
('standard_job_application.interview', 'standard_job_application', 'interview'),
('standard_job_application.make_offer', 'standard_job_application', 'make_offer'),
('standard_job_application.seek_references', 'standard_job_application', 'seek_references'),
('standard_job_application.application_closed', 'standard_job_application', 'application_closed'),
('technical_job_application.application_received', 'technical_job_application', 'application_received'),
('technical_job_application.application_review', 'technical_job_application', 'application_review'),
('technical_job_application.invited_to_interview', 'technical_job_application', 'invited_to_interview'),
('technical_job_application.interview', 'technical_job_application', 'interview'),
('technical_job_application.make_offer', 'technical_job_application', 'make_offer'),
('technical_job_application.seek_references', 'technical_job_application', 'seek_references'),
('technical_job_application.application_closed', 'technical_job_application', 'application_closed');




CREATE TABLE IF NOT EXISTS managed_entity_state (
	id INT UNSIGNED AUTO_INCREMENT,
	due_date DATE NOT NULL DEFAULT '1000-01-01',
	effective_period_from DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	effective_period_to DATETIME NOT NULL DEFAULT '9999-12-31',
	notes TEXT,
	managed_entity_id INT UNSIGNED NOT NULL,
	wf_state_type_state_id INT VARCHAR(255) NOT NULL,
	wf_state_type_outcome_id VARCHAR(255) NOT NULL,
	wf_state_type_qual_id VARCHAR(255) NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (wf_state_type_state_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (wf_state_type_outcome_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (wf_state_type_qual_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (managed_entity_id) REFERENCES your_entity_to_manage(id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS workflow_state_context (
	id VARCHAR(255),
	child_disabled BOOLEAN NOT NULL DEFAULT 0,
	workflow_state_type_id VARCHAR(255) NOT NULL,
	workflow_state_hierachy_id VARCHAR(255) NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (workflow_state_type_id) REFERENCES workflow_state_type(id),
	FOREIGN KEY (workflow_state_hierachy_id) REFERENCES workflow_state_hierachy(id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO workflow_state_context
(id, workflow_state_type_id, workflow_state_hierachy_id) VALUES 
('standard_job_application.application_received.accepted', 'standard_job_application', 'application_received.accepted'),
('standard_job_application.application_received.rejected', 'standard_job_application', 'application_received.rejected'),
('standard_job_application.application_review.passed', 'standard_job_application', 'application_review.passed'),
('standard_job_application.application_review.failed', 'standard_job_application', 'application_review.failed'),
('standard_job_application.invited_to_interview.accepted', 'standard_job_application', 'invited_to_interview.accepted'),
('standard_job_application.invited_to_interview.declined', 'standard_job_application', 'invited_to_interview.declined'),
('standard_job_application.interview.passed', 'standard_job_application', 'interview.passed'),
('standard_job_application.interview.failed', 'standard_job_application', 'interview.failed'),
('standard_job_application.interview.candidate_cancelled', 'standard_job_application', 'interview.candidate_cancelled'),
('standard_job_application.interview.no_show', 'standard_job_application', 'interview.no_show'),
('standard_job_application.make_offer.accepted', 'standard_job_application', 'make_offer.accepted'),
('standard_job_application.make_offer.declined', 'standard_job_application', 'make_offer.declined'),
('standard_job_application.seek_references.passed', 'standard_job_application', 'seek_references.passed'),
('standard_job_application.seek_references.failed', 'standard_job_application', 'seek_references.failed'),
('standard_job_application.application_closed.hired', 'standard_job_application', 'application_closed.hired'),
('standard_job_application.application_closed.not_hired', 'standard_job_application', 'application_closed.not_hired');

CREATE TABLE IF NOT EXISTS workflow_state_option (
	id INT UNSIGNED AUTO_INCREMENT,
	alt_sequence INT UNSIGNED NOT NULL DEFAULT 0,
	workflow_state_context_id VARCHAR(255) NOT NULL,
	workflow_state_type_id VARCHAR(255) NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (workflow_state_context_id) REFERENCES workflow_state_context(id),
	FOREIGN KEY (workflow_state_type_id) REFERENCES workflow_state_type(id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO workflow_state_option 
(workflow_state_context_id, workflow_state_type_id) VALUES
('standard_job_application.application_received.accepted', 'application_review'),
('standard_job_application.application_received.rejected', 'not_hired'),
('standard_job_application.application_review.passed', 'invited_to_interview'),
('standard_job_application.application_review.failed', 'not_hired'),
('standard_job_application.invited_to_interview.accepted', 'interview'),
('standard_job_application.invited_to_interview.declined', 'not_hired'),
('standard_job_application.interview.passed', 'make_offer'),
('standard_job_application.interview.passed', 'seek_references'),
('standard_job_application.interview.failed', 'application_closed'),
('standard_job_application.interview.candidate_cancelled', 'application_closed'),
('standard_job_application.interview.candidate_cancelled', 'invited_to_interview'),
('standard_job_application.interview.no_show', 'application_closed'),
('standard_job_application.make_offer.accepted', 'seek_references'),
('standard_job_application.make_offer.declined', 'application_closed'),
('standard_job_application.seek_references.passed', 'hired'),
('standard_job_application.seek_references.failed', 'application_closed'),
('standard_job_application.application_closed.hired', 'end'),
('standard_job_application.application_closed.not_hired', 'end')
;
```


## Modelling it in Golang

```go
package main

import (
	"fmt"
)

type Status string

type Transition func() Status

type Statuses map[Status][]Status

func main() {
	var statuses = Statuses{
		ApplicationReceived: []Status{ApplicationReview, ApplicationClosed},
		ApplicationReview:   []Status{InvitedToInterview, ApplicationClosed},
		InvitedToInterview:  []Status{Interview, ApplicationClosed},
		Interview:           []Status{MakeOffer, SeekReferences, ApplicationClosed, InvitedToInterview},
		MakeOffer:           []Status{SeekReferences, ApplicationClosed},
		SeekReferences:      []Status{Hired, ApplicationClosed},
		Hired:               []Status{},
		ApplicationClosed:   []Status{},
	}
}
```

REFERENCES: 
- https://www.vertabelo.com/blog/technical-articles/the-workflow-pattern-part-1-using-workflow-patterns-to-manage-the-state-of-any-entity
- https://www.vertabelo.com/blog/technical-articles/the-workflow-pattern-part-2-using-configuration-tables-to-define-the-actual-workflow


