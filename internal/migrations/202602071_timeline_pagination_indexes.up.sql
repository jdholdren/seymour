-- Remove old partial indexes that are now redundant
DROP INDEX IF EXISTS idx_timeline_entries_user_visible;
DROP INDEX IF EXISTS idx_timeline_entries_user_id_requires_judgement;

-- Primary index for timeline pagination queries
-- Covers: user_id + status filtering with created_at sorting
CREATE INDEX idx_timeline_user_status_created
ON timeline_entries(user_id, status, created_at DESC);

-- Index for feed-filtered timeline queries
-- Covers: user_id + feed_id + status filtering with created_at sorting
CREATE INDEX idx_timeline_user_feed_status_created
ON timeline_entries(user_id, feed_id, status, created_at DESC);

-- Keep the basic user_id index for other queries
-- idx_timeline_entries_user_id already exists and is still useful
