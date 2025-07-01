-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user (
	id INT UNSIGNED AUTO_INCREMENT,
	
	-- Email.
	email VARCHAR(255) NOT NULL UNIQUE,
	email_verified BOOLEAN NOT NULL DEFAULT 0,
	password VARCHAR(255) NOT NULL,

	-- Profile.
	first_name VARCHAR(64) NOT NULL DEFAULT '',
	last_name VARCHAR(64) NOT NULL DEFAULT '',
	birth_date DATE NOT NULL DEFAULT '9999-12-31',
	gender CHAR(1) NOT NULL DEFAULT '' COMMENT '(m)ale, (f)emale or empty string if not set', 
	spouse BOOLEAN NOT NULL DEFAULT 0,
	children TINYINT NOT NULL DEFAULT 0 COMMENT 'Number of children',
	phone_number VARCHAR(32) NOT NULL DEFAULT '',
	phone_number_verified BOOLEAN NOT NULL DEFAULT 0,
	picture VARCHAR(2083) NOT NULL DEFAULT '',

	-- Address.
	address VARCHAR(255) NOT NULL DEFAULT 'The full address of the user',
	prefecture_id INT UNSIGNED NOT NULL DEFAULT 1,
	postal_code VARCHAR(16) NOT NULL DEFAULT '',

	-- Datetime.
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	deleted_at DATETIME NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (prefecture_id) REFERENCES prefecture(id)
) ENGINE=InnoDB CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user;
-- +goose StatementEnd
