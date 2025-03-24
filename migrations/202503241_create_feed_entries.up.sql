CREATE TABLE feed_entries (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	publish_date DATE NOT NULL,
	description TEXT NOT NULL,
	guid TEXT NOT NULL, -- The link to the original article
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
