DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'rdm_app') THEN
        CREATE ROLE rdm_app WITH LOGIN PASSWORD 'rdm_app_password';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'rdm_readonly') THEN
        CREATE ROLE rdm_readonly WITH LOGIN PASSWORD 'rdm_readonly_password';
    END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO rdm_app, rdm_readonly;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO rdm_app;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO rdm_readonly;
