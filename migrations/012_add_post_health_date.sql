-- health_date links a post to one day of health_records (YYYY-MM-DD, local
-- date). NULL = no link. Stored as TEXT on purpose: dates stay strings
-- through the whole stack (see health_record_repo.go measuredAtFormat).
ALTER TABLE posts ADD COLUMN health_date TEXT;
