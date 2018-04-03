CREATE TABLE configs (
	hostname TEXT NOT NULL,
	config JSON NOT NULL,

  CONSTRAINT hostname_unique UNIQUE (hostname)
);