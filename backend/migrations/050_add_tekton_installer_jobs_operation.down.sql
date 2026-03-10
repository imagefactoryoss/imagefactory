BEGIN;

ALTER TABLE tekton_installer_jobs
  DROP COLUMN IF EXISTS operation;

COMMIT;
