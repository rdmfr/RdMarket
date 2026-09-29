#!/bin/sh
set -eu

: "${FORECAST_DB_USER:=rdm_forecast}"
: "${FORECAST_DB_PASSWORD:?FORECAST_DB_PASSWORD must be set}"

psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  --set=forecast_user="$FORECAST_DB_USER" \
  --set=forecast_password="$FORECAST_DB_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'forecast_user', :'forecast_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'forecast_user')
\gexec
SELECT format('ALTER ROLE %I LOGIN PASSWORD %L', :'forecast_user', :'forecast_password')
\gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO %I', current_database(), :'forecast_user')
\gexec
SQL
