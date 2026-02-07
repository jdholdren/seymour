-- Restore old partial indexes that were removed
CREATE INDEX idx_timeline_entries_user_visible ON timeline_entries(user_id) WHERE status = 'approved';
CREATE INDEX idx_timeline_entries_user_id_requires_judgement ON timeline_entries(user_id) WHERE status = 'requires_judgement';

-- Remove the new indexes
DROP INDEX IF EXISTS idx_timeline_user_status_created;
DROP INDEX IF EXISTS idx_timeline_user_feed_status_created;
