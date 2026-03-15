#!/bin/bash
# Database initialization

init_database() {
    if [[ "$DB_MODE" != "builtin" ]]; then
        log_info "External database mode — skipping initialization"
        return 0
    fi

    log_info "$MSG_DEPLOY_DB_INIT"

    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local psql_base="docker compose -f $compose_file exec -T postgres psql -U postgres"

    # Create the launchpad database if it doesn't exist
    $psql_base -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'launchpad'" | grep -q 1 \
        || $psql_base -d postgres -c "CREATE DATABASE launchpad"

    local psql_cmd="$psql_base -d launchpad"

    # Check if already initialized
    local schema_count
    schema_count=$($psql_cmd -t -c "SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'launchpad_%'" 2>/dev/null | tr -d ' ')
    if [[ "${schema_count:-0}" -ge 6 ]]; then
        log_ok "Database schemas already exist — skipping"
        return 0
    fi

    # Create schemas and users
    local schemas=("main" "monitoring" "events" "billing" "stats" "gateway")
    local passwords=(
        "$DB_PASSWORD_MAIN" "$DB_PASSWORD_MONITORING" "$DB_PASSWORD_EVENTS"
        "$DB_PASSWORD_BILLING" "$DB_PASSWORD_STATS" "$DB_PASSWORD_GATEWAY"
    )

    for i in "${!schemas[@]}"; do
        local schema="launchpad_${schemas[$i]}"
        local user="launchpad_${schemas[$i]}user"
        local pass="${passwords[$i]}"

        $psql_cmd <<SQL
CREATE SCHEMA IF NOT EXISTS ${schema};
DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '${user}') THEN
        CREATE ROLE ${user} LOGIN PASSWORD '${pass}';
    END IF;
END \$\$;
GRANT USAGE ON SCHEMA ${schema} TO ${user};
GRANT CREATE ON SCHEMA ${schema} TO ${user};
ALTER DEFAULT PRIVILEGES IN SCHEMA ${schema} GRANT ALL ON TABLES TO ${user};
ALTER DEFAULT PRIVILEGES IN SCHEMA ${schema} GRANT ALL ON SEQUENCES TO ${user};
SQL
        log_ok "Schema: $schema"
    done

    # Create Gitea database
    docker compose -f "$compose_file" exec -T postgres psql -U postgres <<SQL
SELECT 'CREATE DATABASE gitea' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'gitea')\gexec
DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'gitea') THEN
        CREATE ROLE gitea LOGIN PASSWORD '${GITEA_DB_PASSWORD}';
    END IF;
END \$\$;
GRANT ALL PRIVILEGES ON DATABASE gitea TO gitea;
ALTER DATABASE gitea OWNER TO gitea;
SQL
    # Grant schema-level permissions within gitea database
    docker compose -f "$compose_file" exec -T postgres psql -U postgres -d gitea <<SQL
GRANT ALL ON SCHEMA public TO gitea;
SQL
    log_ok "Gitea database"

    log_ok "Database initialization complete"
}
