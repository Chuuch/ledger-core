#!/bin/bash
set -euo pipefail

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
  SELECT 'CREATE DATABASE account_db'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'account_db')\gexec
  SELECT 'CREATE DATABASE ledger_db'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'ledger_db')\gexec
  SELECT 'CREATE DATABASE transfer_db'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'transfer_db')\gexec

  GRANT ALL PRIVILEGES ON DATABASE account_db TO ledger_user;
  GRANT ALL PRIVILEGES ON DATABASE ledger_db TO ledger_user;
  GRANT ALL PRIVILEGES ON DATABASE transfer_db TO ledger_user;
EOSQL
