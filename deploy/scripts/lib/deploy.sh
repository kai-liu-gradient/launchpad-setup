#!/bin/bash
# Deployment orchestration, health checks, and lifecycle management

COMPOSE_CMD="docker compose -f ${DEPLOY_DIR}/generated/docker-compose.yml"

wait_for_healthy() {
    local service="$1" timeout="$2"
    local elapsed=0

    while [[ $elapsed -lt $timeout ]]; do
        local container_id
        container_id=$($COMPOSE_CMD ps -q "$service" 2>/dev/null | head -1)
        if [[ -n "$container_id" ]]; then
            local status
            status=$(docker inspect --format='{{.State.Health.Status}}' "$container_id" 2>/dev/null || echo "unknown")
            if [[ "$status" == "healthy" ]]; then
                return 0
            fi
        fi
        sleep 2
        elapsed=$((elapsed + 2))
        _draw_progress "$MSG_HEALTH_WAITING $service... ${elapsed}s"
    done
    echo ""
    log_error "$service $MSG_HEALTH_FAILED in ${timeout}s"
    $COMPOSE_CMD logs --tail=20 "$service"
    return 1
}

deploy_fail() {
    local phase="$1"
    log_error "$MSG_DEPLOY_FAILED at phase: $phase"
    echo ""
    echo "  $MSG_ERR_RESUME:"
    echo "    ./deploy/setup.sh --resume"
    echo ""
    echo "  $MSG_ERR_LOGS:"
    echo "    docker compose -f deploy/generated/docker-compose.yml logs <service>"
    echo ""
    echo "  $MSG_ERR_UNINSTALL:"
    echo "    ./deploy/setup.sh --uninstall"
    exit 1
}

setup_hosts() {
    local domains=("$LAUNCHPAD_DOMAIN" "$GITEA_DOMAIN")
    for domain in "${domains[@]}"; do
        grep -qF "$domain" /etc/hosts || echo "127.0.0.1  $domain" >> /etc/hosts || log_warn "Could not update /etc/hosts (run as root)"
    done
    log_ok "Hosts file configured"
}

deploy_services() {
    echo ""
    echo -e "${BOLD}  Deploying services...${NC}"
    echo ""
    # Total steps: infra(2) + db(2) + gitea(2) + services(4) + nginx(1) = 11
    progress_start 11

    # Phase 1: Infrastructure
    if [[ "$DB_MODE" == "builtin" ]]; then
        _draw_progress "Starting PostgreSQL & Redis..."
        $COMPOSE_CMD up -d postgres redis 2>&1 | verbose_filter
        wait_for_healthy postgres 30 || deploy_fail "infrastructure (postgres)"
        progress_update "PostgreSQL ready"
        wait_for_healthy redis 15 || deploy_fail "infrastructure (redis)"
        progress_update "Redis ready"
    else
        progress_update "External database"
        progress_update "External Redis"
    fi

    # Phase 2: Database init + schema migration
    _draw_progress "Initializing database schemas..."
    init_database || deploy_fail "database initialization"
    progress_update "Database schemas created"

    _draw_progress "Running schema migrations..."
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema.prisma 2>&1 | verbose_filter || deploy_fail "prisma db push (main)"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/monitoring.prisma 2>&1 | verbose_filter || deploy_fail "prisma db push (monitoring)"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-billing.prisma 2>&1 | verbose_filter || deploy_fail "prisma db push (billing)"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-events.prisma 2>&1 | verbose_filter || deploy_fail "prisma db push (events)"
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-stats.prisma 2>&1 | verbose_filter || deploy_fail "prisma db push (stats)"
    progress_update "Migrations complete"

    # Phase 3: Gitea
    _draw_progress "Starting Gitea..."
    $COMPOSE_CMD up -d gitea 2>&1 | verbose_filter
    wait_for_healthy gitea 90 || deploy_fail "gitea"
    progress_update "Gitea healthy"

    _draw_progress "Bootstrapping Gitea..."
    bootstrap_gitea || deploy_fail "gitea bootstrap"
    progress_update "Gitea bootstrapped"

    # Phase 4: Application services
    _draw_progress "Starting API, UI, Router, Gateway..."
    $COMPOSE_CMD up -d api ui router cron backup-worker gateway 2>&1 | verbose_filter
    wait_for_healthy api 90 || deploy_fail "application services (api)"
    progress_update "API ready"
    wait_for_healthy ui 60 || deploy_fail "application services (ui)"
    progress_update "UI ready"
    wait_for_healthy router 60 || deploy_fail "application services (router)"
    progress_update "Router ready"

    # Phase 5: Nginx
    _draw_progress "Starting Nginx..."
    $COMPOSE_CMD up -d nginx 2>&1 | verbose_filter
    wait_for_healthy nginx 30 || deploy_fail "nginx"

    setup_hosts

    progress_done
    echo ""
}

bootstrap_gitea() {
    log_info "$MSG_DEPLOY_GITEA"

    local gitea_url="http://localhost:3000"
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"

    # Check if already bootstrapped
    local status
    status=$(docker compose -f "$compose_file" exec -T gitea \
        curl -s -o /dev/null -w "%{http_code}" "http://localhost:3000/api/v1/orgs/launchpad" 2>/dev/null || echo "000")

    if [[ "$status" == "200" ]]; then
        # Ensure token is in .env (may have been lost by re-render)
        if ! grep -q '^GITEA_ACCESS_TOKEN=' "${DEPLOY_DIR}/generated/launchpad/.env" 2>/dev/null; then
            if [[ -n "${GITEA_ACCESS_TOKEN:-}" ]]; then
                echo "" >> "${DEPLOY_DIR}/generated/launchpad/.env"
                echo "GITEA_ACCESS_TOKEN=${GITEA_ACCESS_TOKEN}" >> "${DEPLOY_DIR}/generated/launchpad/.env"
                echo "GITEA_USER=${ADMIN_EMAIL%%@*}" >> "${DEPLOY_DIR}/generated/launchpad/.env"
            fi
        fi
        log_done "Gitea already bootstrapped"
        return 0
    fi

    # Wait for Gitea to be fully ready
    sleep 5

    local gitea_admin="${ADMIN_EMAIL%%@*}"  # username from email
    local gitea_pass="${ADMIN_PASSWORD}"

    # Create admin user via Gitea CLI
    docker compose -f "$compose_file" exec -T gitea \
        gitea admin user create \
        --username "$gitea_admin" \
        --password "$gitea_pass" \
        --email "$ADMIN_EMAIL" \
        --admin --must-change-password=false &>/dev/null || true

    # Generate API token
    local token_response
    token_response=$(docker compose -f "$compose_file" exec -T gitea \
        curl -s -X POST "http://localhost:3000/api/v1/users/${gitea_admin}/tokens" \
        -u "${gitea_admin}:${gitea_pass}" \
        -H "Content-Type: application/json" \
        -d '{"name":"launchpad-api","scopes":["all"]}')

    # Gitea returns token in "sha1" (older) or "token" (newer) field
    GITEA_ACCESS_TOKEN=$(echo "$token_response" | grep -oE '"(sha1|token)":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [[ -n "$GITEA_ACCESS_TOKEN" ]]; then
        # Create organization
        docker compose -f "$compose_file" exec -T gitea \
            curl -s -X POST "http://localhost:3000/api/v1/orgs" \
            -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
            -H "Content-Type: application/json" \
            -d '{"username":"launchpad","full_name":"Launchpad","visibility":"public"}' >/dev/null

        # Write token back to .env (API reads it on next start)
        echo "" >> "${DEPLOY_DIR}/generated/launchpad/.env"
        echo "GITEA_ACCESS_TOKEN=${GITEA_ACCESS_TOKEN}" >> "${DEPLOY_DIR}/generated/launchpad/.env"
        echo "GITEA_USER=${gitea_admin}" >> "${DEPLOY_DIR}/generated/launchpad/.env"

        # Persist token in .secrets so render_templates can restore it
        local secrets_file="${DEPLOY_DIR}/generated/.secrets"
        if [[ -f "$secrets_file" ]]; then
            # Remove old entries if any, then append
            sed_i '/^GITEA_ACCESS_TOKEN=/d; /^GITEA_USER=/d' "$secrets_file"
            echo "GITEA_ACCESS_TOKEN=\"${GITEA_ACCESS_TOKEN}\"" >> "$secrets_file"
            echo "GITEA_USER=\"${gitea_admin}\"" >> "$secrets_file"
        fi

        log_ok "Gitea bootstrapped (admin: $gitea_admin, org: launchpad)"
    else
        log_warn "Gitea token generation failed — manual setup may be required"
    fi
}

import_templates() {
    log_info "$MSG_DEPLOY_TEMPLATES"

    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    local files_dir="${PROJECT_DIR}/files"
    local gitea_admin="${ADMIN_EMAIL%%@*}"
    local gitea_pass="${ADMIN_PASSWORD}"
    local gitea_api="http://localhost:3000/api/v1"
    local api_base="http://localhost:6802"  # API container internal port, matches docker-compose config

    # Check prerequisites
    if [[ -z "${GITEA_ACCESS_TOKEN:-}" ]]; then
        log_warn "No Gitea token — skipping template import"
        return 1
    fi

    if [[ ! -d "$files_dir" ]]; then
        log_warn "No files/ directory found — skipping template import"
        return 0
    fi

    # Helper: run curl inside gitea container (defined early for health check)
    _gitea_curl() {
        docker compose -f "$compose_file" exec -T gitea curl -s "$@"
    }

    # Check Gitea is healthy
    local gitea_status
    gitea_status=$(_gitea_curl -o /dev/null -w "%{http_code}" "${gitea_api}/version" 2>/dev/null || echo "000")
    if [[ "$gitea_status" != "200" ]]; then
        log_warn "Gitea not healthy (HTTP $gitea_status) — skipping template import"
        return 1
    fi

    # Check API is healthy
    local api_status
    api_status=$(docker compose -f "$compose_file" exec -T api curl -s -o /dev/null -w "%{http_code}" "${api_base}/health" 2>/dev/null || echo "000")
    if [[ "$api_status" != "200" ]]; then
        log_warn "API not healthy (HTTP $api_status) — skipping template import"
        return 1
    fi

    # Phase 1: Push git repos from tar.gz files
    local tar_file repo_name
    for tar_file in "$files_dir"/*.tar.gz; do
        [[ -f "$tar_file" ]] || continue
        repo_name=$(basename "$tar_file" | sed 's/-main\.tar\.gz$//')

        # Check if repo exists
        local repo_status
        repo_status=$(_gitea_curl -o /dev/null -w "%{http_code}" \
            -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
            "${gitea_api}/repos/launchpad/${repo_name}")

        if [[ "$repo_status" == "200" ]]; then
            # Check if repo has commits (not empty)
            local commits
            commits=$(_gitea_curl \
                -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
                "${gitea_api}/repos/launchpad/${repo_name}/commits?limit=1")
            if echo "$commits" | grep -q '"sha"'; then
                log_ok "$MSG_DEPLOY_TEMPLATES_REPO ${repo_name} — $MSG_DEPLOY_TEMPLATES_SKIP"
                continue
            fi
        else
            # Create empty repo
            _gitea_curl -X POST "${gitea_api}/orgs/launchpad/repos" \
                -H "Authorization: token ${GITEA_ACCESS_TOKEN}" \
                -H "Content-Type: application/json" \
                -d "{\"name\":\"${repo_name}\",\"auto_init\":false,\"default_branch\":\"main\"}" >/dev/null
        fi

        # Extract and push
        local tmp_dir
        tmp_dir=$(mktemp -d)
        tar xzf "$tar_file" -C "$tmp_dir" --strip-components=1 2>/dev/null || tar xzf "$tar_file" -C "$tmp_dir"

        (
            cd "$tmp_dir"
            git init -b main >/dev/null 2>&1
            git add -A >/dev/null 2>&1
            git commit -m "Initial import" --author="Launchpad <launchpad@${DOMAIN}>" >/dev/null 2>&1

            local remote_url="https://${gitea_admin}:${gitea_pass}@${GITEA_DOMAIN}/launchpad/${repo_name}.git"
            if [[ "${SSL_MODE}" == "selfsigned" ]]; then
                GIT_SSL_NO_VERIFY=1 git push -f "$remote_url" main >/dev/null 2>&1
            else
                git push -f "$remote_url" main >/dev/null 2>&1
            fi
        )
        rm -rf "$tmp_dir"

        log_ok "$MSG_DEPLOY_TEMPLATES_REPO ${repo_name}"
    done

    # Phase 2: Import YAML templates via API
    local yaml_file yaml_content
    for yaml_file in "$files_dir"/*.yaml; do
        [[ -f "$yaml_file" ]] || continue
        local template_name
        template_name=$(basename "$yaml_file" .yaml)

        # Read and substitute $GITLAB_DOMAIN
        yaml_content=$(cat "$yaml_file" | sed "s|\\\$GITLAB_DOMAIN|https://${GITEA_DOMAIN}|g")

        # Escape for JSON (handle newlines, quotes, backslashes)
        local json_yaml
        json_yaml=$(printf '%s' "$yaml_content" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null \
            || printf '%s' "$yaml_content" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | awk '{printf "%s\\n", $0}' | sed 's/\\n$//')

        # Validate
        local validate_resp
        validate_resp=$(docker compose -f "$compose_file" exec -T api \
            curl -s -X POST "${api_base}/api/v1/templates/validate-yaml" \
            -H "Content-Type: application/json" \
            -d "{\"yaml\": ${json_yaml}}" 2>/dev/null)

        if echo "$validate_resp" | grep -qi "error\|invalid"; then
            log_warn "$MSG_DEPLOY_TEMPLATES_YAML ${template_name} — validation failed: $validate_resp"
            continue
        fi

        # Import
        local import_resp
        import_resp=$(docker compose -f "$compose_file" exec -T api \
            curl -s -X POST "${api_base}/api/v1/templates/from-yaml" \
            -H "Content-Type: application/json" \
            -d "{\"yaml\": ${json_yaml}}" 2>/dev/null)

        if echo "$import_resp" | grep -qi "error"; then
            log_warn "$MSG_DEPLOY_TEMPLATES_YAML ${template_name} — import failed: $import_resp"
        else
            log_ok "$MSG_DEPLOY_TEMPLATES_YAML ${template_name}"
        fi
    done

    log_done "$MSG_DEPLOY_TEMPLATES_DONE"
}

show_result() {
    local ip
    ip=$(detect_internal_ip 2>/dev/null || echo "x.x.x.x")

    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  ✓ $MSG_DEPLOY_COMPLETE${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════${NC}"
    echo ""
    echo "  $MSG_OUT_URLS:"
    echo "    $MSG_OUT_DASHBOARD:  https://${LAUNCHPAD_DOMAIN}"
    echo "    $MSG_OUT_GITEA:      https://${GITEA_DOMAIN}"
    echo "    $MSG_OUT_ADMIN:      https://${LAUNCHPAD_DOMAIN}/admin"
    echo ""
    echo "  $MSG_OUT_CREDENTIALS:"
    echo "    $MSG_OUT_EMAIL:      ${ADMIN_EMAIL}"
    echo "    $MSG_OUT_PASSWORD:   ${ADMIN_PASSWORD}"
    echo ""
    echo "  $MSG_OUT_K8S:"
    echo "    $MSG_OUT_MODE:       ${K8S_MODE}"
    if [[ "${CLUSTER_REGISTERED:-false}" == "true" ]]; then
        echo "    $MSG_OUT_STATUS:     ✓ Registered"
    else
        echo -e "    $MSG_OUT_STATUS:     ${YELLOW}⚠ Not registered${NC}"
        echo "    Register manually:   ./deploy/setup.sh --resume"
    fi
    echo ""
    echo "  $MSG_OUT_DNS:"
    echo "    $MSG_OUT_DNS_MSG ($ip):"
    echo "    ├─ ${LAUNCHPAD_DOMAIN}  → $ip"
    echo "    ├─ ${GITEA_DOMAIN}      → $ip"
    echo "    └─ *.${DOMAIN}          → $ip"
    echo ""
    echo "  $MSG_OUT_CONFIG: deploy/generated/"
    echo "  $MSG_OUT_LOGS: docker compose -f deploy/generated/docker-compose.yml logs -f"
    echo "  $MSG_OUT_RECONFIG: ./deploy/setup.sh --reconfigure"
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════${NC}"
}

show_status() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    if [[ ! -f "$compose_file" ]]; then
        echo "No deployment found. Run ./deploy/setup.sh to deploy."
        exit 1
    fi
    docker compose -f "$compose_file" ps
}

upgrade_services() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    # Reload versions, saved config, and secrets, then re-render compose
    source "${DEPLOY_DIR}/scripts/lib/common.sh"
    source "${DEPLOY_DIR}/scripts/lib/render.sh"
    source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
    source "${DEPLOY_DIR}/versions.conf"
    source "${DEPLOY_DIR}/generated/.setup.conf"
    load_secrets
    render_compose
    log_info "Pulling latest images..."
    docker compose -f "$compose_file" pull
    docker compose -f "$compose_file" up -d
    wait_for_healthy api 60
    wait_for_healthy ui 30
    wait_for_healthy nginx 15
    log_ok "Services upgraded"
}

uninstall_services() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    log_warn "This will stop all services, remove containers, AND DELETE ALL DATA (databases, Gitea repos, etc.)."
    echo ""
    if ask_confirm "Are you sure you want to delete everything? This cannot be undone." "N"; then
        docker compose -f "$compose_file" down -v
        log_ok "Services and volumes removed"
    fi
}

resume_deploy() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    if [[ ! -f "$compose_file" ]]; then
        log_error "No deployment configuration found. Run ./deploy/setup.sh first."
        exit 1
    fi

    # Load saved config, secrets, and all modules
    source "${DEPLOY_DIR}/scripts/lib/common.sh"
    source "${DEPLOY_DIR}/scripts/lib/detect.sh"
    source "${DEPLOY_DIR}/scripts/lib/secrets.sh"
    source "${DEPLOY_DIR}/scripts/lib/render.sh"
    source "${DEPLOY_DIR}/scripts/lib/database.sh"
    source "${DEPLOY_DIR}/scripts/lib/k3s.sh"
    source "${DEPLOY_DIR}/versions.conf"
    source "${DEPLOY_DIR}/generated/.setup.conf"
    source "${DEPLOY_DIR}/scripts/lang/${LANG_CHOICE:-en}.sh"
    load_secrets || { log_error "Cannot load secrets. Run ./deploy/setup.sh to redeploy."; exit 1; }

    log_info "Checking deployment state and resuming from failed phase..."

    # Helper to check if a service is already healthy
    is_healthy() {
        local cid
        cid=$(docker compose -f "$compose_file" ps -q "$1" 2>/dev/null | head -1)
        [[ -n "$cid" ]] && [[ "$(docker inspect --format='{{.State.Health.Status}}' "$cid" 2>/dev/null)" == "healthy" ]]
    }

    # Phase 1: Infrastructure
    if [[ "$DB_MODE" == "builtin" ]]; then
        if ! is_healthy postgres || ! is_healthy redis; then
            log_step "1/4" "$MSG_DEPLOY_INFRA"
            docker compose -f "$compose_file" up -d postgres redis
            wait_for_healthy postgres 30 || deploy_fail "infrastructure (postgres)"
            wait_for_healthy redis 15 || deploy_fail "infrastructure (redis)"
        else
            log_ok "Infrastructure already healthy — skipping"
        fi
    fi

    # Phase 2: Database init (idempotent, safe to re-run)
    init_database

    # Phase 3: Application services
    if ! is_healthy api || ! is_healthy ui || ! is_healthy router; then
        log_step "2/4" "$MSG_DEPLOY_SERVICES"
        docker compose -f "$compose_file" up -d api ui router cron backup-worker gateway gitea
        wait_for_healthy api 60 || deploy_fail "application services (api)"
        wait_for_healthy ui 30 || deploy_fail "application services (ui)"
        wait_for_healthy router 30 || deploy_fail "application services (router)"
        wait_for_healthy gitea 45 || deploy_fail "application services (gitea)"
    else
        log_ok "Application services already healthy — skipping"
    fi

    # Phase 4: Nginx
    if ! is_healthy nginx; then
        log_step "3/4" "$MSG_DEPLOY_NGINX"
        docker compose -f "$compose_file" up -d nginx
        wait_for_healthy nginx 30 || deploy_fail "nginx"
    else
        log_ok "Nginx already healthy — skipping"
    fi

    # Phase 5: Gitea bootstrap (idempotent)
    bootstrap_gitea

    setup_hosts

    # Phase 6: Cluster registration
    register_cluster || log_warn "Cluster registration failed — you can retry later"

    log_done "All services healthy"
    show_result
}
