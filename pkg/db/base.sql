CREATE TABLE IF NOT EXISTS status (
	id INTEGER PRIMARY KEY AUTOINCREMENT, 
	name VARCHAR(20) UNIQUE NOT NULL
);

INSERT OR IGNORE INTO status (id, name) VALUES
	(1, 'open'),
	(2, 'closed'),
	(3, 'on_hold'),
	(4, 'done'),
	(5, 'abandoned')
;

CREATE TABLE IF NOT EXISTS todo (
	id INTEGER PRIMARY KEY AUTOINCREMENT, 
	title VARCHAR(255) NOT NULL,
	description VARCHAR(1023),
	status_id SMALLINT NOT NULL,
	rank INT NOT NULL,
	created_datetime DATETIME NOT NULL,
	updated_datetime DATETIME,
	FOREIGN KEY (status_id) REFERENCES status(id)
);

CREATE TABLE IF NOT EXISTS label (
	id INTEGER PRIMARY KEY AUTOINCREMENT, 
	name VARCHAR(20) UNIQUE NOT NULL
);

INSERT OR IGNORE INTO label (id, name) VALUES
	(1, 'task'),
	(2, 'learning'),
	(3, 'human_interaction'),
	(4, 'urgent'),
	(5, 'platform_learning'),
	(6, 'personal_growth'),
	(7, 'environment_setup'),
	(8, 'planning/design'),
	(9, 'onboarding')
;

CREATE TABLE IF NOT EXISTS todo_label (
	id INTEGER PRIMARY KEY AUTOINCREMENT, 
	todo_id INTEGER NOT NULL,
	label_id INTEGER NOT NULL,
	FOREIGN KEY (todo_id) REFERENCES todo(id),
	FOREIGN KEY (label_id) REFERENCES label(id),
	UNIQUE(todo_id, label_id)
);
