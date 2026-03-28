-- Migration: 003_seed_initial_data.down.sql
-- Rollback: Seed data

DELETE FROM user_role_assignments WHERE user_id IN (
    SELECT id FROM users WHERE email IN ('admin@imgfactory.com', 'system@imgfactory.com')
);

DELETE FROM roles WHERE is_system_role = true;

DELETE FROM users WHERE email IN ('admin@imgfactory.com', 'system@imgfactory.com');

DELETE FROM tenants WHERE slug IN ('platform-engineering', 'linux-engineering', 'windows-engineering', 'container-platform', 'aws-tower');

DELETE FROM companies WHERE name = 'ImageFactory';
