CREATE TABLE timeline_entries (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	feed_entry_id TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	status TEXT NOT NULL
);

-- Index for fast lookups by user
CREATE INDEX idx_timeline_entries_user_id ON timeline_entries(user_id);

-- Indexes for querying approved/unapproved entries by user
CREATE INDEX idx_timeline_entries_user_visible ON timeline_entries(user_id) WHERE status = 'approved';
CREATE INDEX idx_timeline_entries_user_id_requires_judgement ON timeline_entries(user_id) WHERE status = 'requires_judgement';
