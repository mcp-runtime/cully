-- This runs in a separate transaction so the exclusive lock from migration
-- 004 has been released before PostgreSQL scans existing entries.
ALTER TABLE cully_entries VALIDATE CONSTRAINT cully_entries_entry_type_check;
