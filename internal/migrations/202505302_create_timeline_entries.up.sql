CREATE TABLE timeline_entries (
	id TEXT PRIMARY KEY,
	subscription_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	feed_id TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	needs_approval BOOLEAN NOT NULL DEFAULT FALSE,
	visible BOOLEAN NOT NULL DEFAULT FALSE
);

-- Index for fast lookups by user
CREATE INDEX idx_timeline_entries_user_id ON timeline_entries(user_id);

-- Indexes for querying approved/unapproved entries by user
CREATE INDEX idx_timeline_entries_user_visible ON timeline_entries(user_id) WHERE visible = 1;
CREATE INDEX idx_timeline_entries_user_id_needs_approval ON timeline_entries(user_id) WHERE needs_approval = 1;
