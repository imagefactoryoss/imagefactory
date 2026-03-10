BEGIN;

-- 1) Add nullable `operation` column to installer jobs
ALTER TABLE tekton_installer_jobs
  ADD COLUMN operation varchar(32);

-- 2) Backfill `operation` from installer job events where available (requested events include operation in details)
UPDATE tekton_installer_jobs t
SET operation = COALESCE(
  (SELECT (details->>'operation')::text
   FROM tekton_installer_job_events e
   WHERE e.job_id = t.id AND e.event_type LIKE '%requested'
   ORDER BY e.created_at ASC
   LIMIT 1),
  'install'
);

-- 3) Prevent nulls going forward (safe after backfill)
ALTER TABLE tekton_installer_jobs
  ALTER COLUMN operation SET NOT NULL;

COMMIT;
