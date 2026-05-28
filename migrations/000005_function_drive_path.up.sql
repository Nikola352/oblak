ALTER TABLE functions
    DROP COLUMN bucket;

ALTER TABLE functions
    RENAME COLUMN path TO archive_path;

ALTER TABLE functions
    ADD COLUMN drive_path TEXT DEFAULT NULL;
