ALTER TABLE epr_registration_requests
    DROP CONSTRAINT IF EXISTS chk_epr_registration_requests_status;

ALTER TABLE epr_registration_requests
    ADD CONSTRAINT chk_epr_registration_requests_status
        CHECK (status IN ('pending', 'approved', 'rejected'));
