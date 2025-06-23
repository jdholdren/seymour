CREATE TABLE subscriptions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	feed_id TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast lookups by user
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
-- Ensure a user can't subscribe to a feed twice
CREATE UNIQUE INDEX idx_subscriptions_user_id_feed_id ON subscriptions(user_id, feed_id);
