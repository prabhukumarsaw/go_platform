-- 018_standardize_role_names.up.sql
-- Standardize system role names to clean professional Title Case

UPDATE roles SET name = 'Super Administrator' WHERE LOWER(name) IN ('super_admin', 'super administrator');
UPDATE roles SET name = 'Editor' WHERE LOWER(name) IN ('editor');
UPDATE roles SET name = 'Sub-Editor' WHERE LOWER(name) IN ('sub_editor', 'sub editor', 'sub-editor');
UPDATE roles SET name = 'Reporter' WHERE LOWER(name) IN ('reporter');
UPDATE roles SET name = 'Community Moderator' WHERE LOWER(name) IN ('moderator', 'community moderator');
UPDATE roles SET name = 'Fact Checker' WHERE LOWER(name) IN ('fact checker', 'fact_checker');
UPDATE roles SET name = 'Multimedia Producer' WHERE LOWER(name) IN ('multimedia producer', 'multimedia_producer');
