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
    # Detect if we have a tty for fancy Style C display
    local _has_tty=false
    if [ -t 1 ] || (echo -n "" >/dev/tty 2>/dev/null); then
        _has_tty=true
    fi

    # ── Build dynamic step list ──
    _ds_steps=()
    _ds_idx=0
    # Associative array: step key → array index
    declare -A _ds_step_map

    _add_step() {
        local key="$1" label="$2"
        _ds_steps+=("pending:0:${label}")
        _ds_step_map[$key]=$_ds_idx
        _ds_idx=$((_ds_idx + 1))
    }

    # Ensure KUBECONFIG is set for helm/kubectl (install_k3s exports this,
    # but if k3s is already running we skip that function)
    export KUBECONFIG="${K8S_KUBECONFIG_PATH:-/etc/rancher/k3s/k3s.yaml}"

    # k3s installation (only if builtin AND not already running)
    local _k3s_needed=false
    if [[ "${K8S_MODE:-builtin}" == "builtin" ]] && ! kubectl get nodes &>/dev/null 2>&1; then
        _k3s_needed=true
        _add_step "k3s" "$MSG_DEPLOY_PROGRESS_K3S"
    fi

    # SSL certificates (always)
    _add_step "ssl" "$MSG_DEPLOY_PROGRESS_SSL"

    # k8s components (only if builtin)
    if [[ "${K8S_MODE:-builtin}" == "builtin" ]]; then
        _add_step "ingress" "$MSG_DEPLOY_PROGRESS_INGRESS"
        _add_step "coredns" "$MSG_DEPLOY_PROGRESS_COREDNS"
        if [[ "${SSL_MODE:-}" == "selfsigned" ]]; then
            _add_step "kyverno" "$MSG_DEPLOY_PROGRESS_KYVERNO"
            _add_step "certdist" "$MSG_DEPLOY_PROGRESS_CERTDIST"
        fi
    fi

    # Docker services (always)
    _add_step "infra" "$MSG_DEPLOY_PROGRESS_INFRA"
    _add_step "pg" "$MSG_DEPLOY_PROGRESS_PG"
    _add_step "redis" "$MSG_DEPLOY_PROGRESS_REDIS"
    _add_step "db" "$MSG_DEPLOY_PROGRESS_DB"
    _add_step "gitea" "$MSG_DEPLOY_PROGRESS_GITEA"
    _add_step "gitea_boot" "$MSG_DEPLOY_PROGRESS_GITEA_BOOT"
    _add_step "api" "$MSG_DEPLOY_PROGRESS_API"
    _add_step "ui" "$MSG_DEPLOY_PROGRESS_UI"
    _add_step "router" "$MSG_DEPLOY_PROGRESS_ROUTER"
    _add_step "nginx" "$MSG_DEPLOY_PROGRESS_NGINX"
    _add_step "templates" "$MSG_DEPLOY_PROGRESS_TEMPLATES"
    _add_step "cluster" "$MSG_DEPLOY_PROGRESS_CLUSTER"

    _ds_total=${#_ds_steps[@]}
    _ds_current=0

    # Helper to look up step index by key
    _step() { echo "${_ds_step_map[$1]}"; }

    if [[ "$_has_tty" == "true" ]]; then
        # ── Style C: fancy progress display with checkmarks/spinners ──
        clear >/dev/tty 2>/dev/null || true
        print_brand_header "$MSG_DEPLOY_DEPLOYING"

        _deploy_area_top=6
        printf "\033[?25l" >/dev/tty

        _update_step() {
            local idx="$1" status="$2" pct="${3:-0}"
            local label="${_ds_steps[$idx]}"
            # Extract label (third field)
            label="${label#*:}"   # remove status:
            label="${label#*:}"   # remove pct:
            _ds_steps[$idx]="${status}:${pct}:${label}"
        }
        _redraw() { _draw_deploy_status "$_ds_current" "$_ds_total" _ds_steps; }
        _complete_step() {
            local idx="$1"
            _update_step "$idx" "done" 100
            _ds_current=$((_ds_current + 1))
            local next=$((idx + 1))
            [[ $next -lt $_ds_total ]] && _update_step "$next" "active" 0
            _redraw
        }
        _fail_with_cursor() { printf "\033[?25h" >/dev/tty; deploy_fail "$1"; }

        # Style C aware health wait — updates per-step progress bar with spinning animation
        _wait_healthy_styled() {
            local service="$1" timeout="$2" step_idx="$3"
            local elapsed=0 tick=0
            while [[ $elapsed -lt $timeout ]]; do
                # Check health every 2s (every 6 ticks of 0.3s)
                if (( tick % 6 == 0 )) && [[ $tick -gt 0 ]]; then
                    elapsed=$((elapsed + 2))
                fi
                if (( tick % 6 == 0 )); then
                    local container_id
                    container_id=$($COMPOSE_CMD ps -q "$service" 2>/dev/null | head -1)
                    if [[ -n "$container_id" ]]; then
                        local hstatus
                        hstatus=$(docker inspect --format='{{.State.Health.Status}}' "$container_id" 2>/dev/null || echo "unknown")
                        if [[ "$hstatus" == "healthy" ]]; then
                            return 0
                        fi
                    fi
                fi
                local step_pct=$(( elapsed * 100 / timeout ))
                [[ $step_pct -gt 95 ]] && step_pct=95
                _update_step "$step_idx" "active" "$step_pct"
                _redraw
                sleep 0.3
                tick=$((tick + 1))
            done
            echo ""
            log_error "$service $MSG_HEALTH_FAILED in ${timeout}s"
            $COMPOSE_CMD logs --tail=20 "$service"
            return 1
        }

        # Style C aware generic wait — updates progress bar while a command runs
        _wait_cmd_styled() {
            local step_idx="$1" timeout="$2"
            shift 2
            local elapsed=0 tick=0
            # Run command in background
            "$@" &
            local _cmd_pid=$!
            while kill -0 "$_cmd_pid" 2>/dev/null; do
                if (( tick % 6 == 0 )) && [[ $tick -gt 0 ]]; then
                    elapsed=$((elapsed + 2))
                fi
                local step_pct=$(( elapsed * 100 / timeout ))
                [[ $step_pct -gt 95 ]] && step_pct=95
                _update_step "$step_idx" "active" "$step_pct"
                _redraw
                sleep 0.3
                tick=$((tick + 1))
                if [[ $elapsed -ge $timeout ]]; then
                    break
                fi
            done
            wait "$_cmd_pid"
        }

        _update_step 0 "active" 0
        _redraw
    else
        # ── Fallback: simple progress bar (no tty) ──
        echo ""
        echo -e "${BOLD}  Deploying services...${NC}"
        echo ""
        progress_start "$_ds_total"

        _complete_step() { progress_update "${1:-}"; }
        _fail_with_cursor() { deploy_fail "$1"; }
    fi

    # Helper: wait for healthy — uses styled version in tty mode
    _do_wait() {
        local service="$1" timeout="$2" step_idx="$3"
        if [[ "$_has_tty" == "true" ]]; then
            _wait_healthy_styled "$service" "$timeout" "$step_idx"
        else
            wait_for_healthy "$service" "$timeout"
        fi
    }

    # ── Phase: K3s installation (conditional) ──
    if [[ "$_k3s_needed" == "true" ]]; then
        local _si=$(_step k3s)
        [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 0; _redraw; }
        # Tell install_k3s to use Style C progress instead of its own progress bar
        [[ "$_has_tty" == "true" ]] && _DS_K3S_STEP=$_si
        install_k3s || _fail_with_cursor "k3s installation"
        unset _DS_K3S_STEP
        _complete_step "$_si"
    fi

    # ── Phase: SSL certificates ──
    local _si=$(_step ssl)
    [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 0; _redraw; }
    setup_certificates || _fail_with_cursor "SSL certificates"
    _complete_step "$_si"

    # ── Phase: K8s components (conditional on builtin) ──
    if [[ "${K8S_MODE:-builtin}" == "builtin" ]]; then
        # Ingress-Nginx
        _si=$(_step ingress)
        [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 0; _redraw; }
        install_ingress_nginx || _fail_with_cursor "ingress-nginx"
        _complete_step "$_si"

        # CoreDNS
        _si=$(_step coredns)
        [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 0; _redraw; }
        configure_coredns || _fail_with_cursor "coredns"
        _complete_step "$_si"

        # Kyverno + cert distribution (selfsigned only)
        if [[ "${SSL_MODE:-}" == "selfsigned" ]]; then
            _si=$(_step kyverno)
            [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 0; _redraw; }
            install_kyverno || _fail_with_cursor "kyverno"
            _complete_step "$_si"

            _si=$(_step certdist)
            [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 0; _redraw; }
            setup_cert_distribution || _fail_with_cursor "cert distribution"
            _complete_step "$_si"
        fi
    fi

    # ── Phase: Infrastructure (PostgreSQL & Redis) ──
    if [[ "$DB_MODE" == "builtin" ]]; then
        $COMPOSE_CMD up -d postgres redis 2>&1 | verbose_filter
        _complete_step $(_step infra)
        _do_wait postgres 30 $(_step pg) || _fail_with_cursor "infrastructure (postgres)"
        _complete_step $(_step pg)
        _do_wait redis 15 $(_step redis) || _fail_with_cursor "infrastructure (redis)"
        _complete_step $(_step redis)
    else
        _complete_step $(_step infra)
        _complete_step $(_step pg)
        _complete_step $(_step redis)
    fi

    # ── Phase: Database init + migration ──
    _si=$(_step db)
    init_database || _fail_with_cursor "database initialization"
    [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 20; _redraw; }
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (main)"
    [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 40; _redraw; }
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/monitoring.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (monitoring)"
    [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 55; _redraw; }
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-billing.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (billing)"
    [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 70; _redraw; }
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-events.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (events)"
    [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 85; _redraw; }
    $COMPOSE_CMD run --rm api npx prisma db push --schema prisma/schema-stats.prisma 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (stats)"
    [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 95; _redraw; }
    $COMPOSE_CMD run --rm gateway npm run db:push 2>&1 | verbose_filter || _fail_with_cursor "prisma db push (gateway)"
    _complete_step "$_si"

    # ── Phase: Gitea ──
    $COMPOSE_CMD up -d gitea 2>&1 | verbose_filter
    _do_wait gitea 90 $(_step gitea) || _fail_with_cursor "gitea"
    _complete_step $(_step gitea)

    bootstrap_gitea || _fail_with_cursor "gitea bootstrap"
    _complete_step $(_step gitea_boot)

    # ── Phase: Application services ──
    $COMPOSE_CMD up -d api ui router cron backup-worker gateway 2>&1 | verbose_filter
    _do_wait api 90 $(_step api) || _fail_with_cursor "application services (api)"
    _complete_step $(_step api)
    _do_wait ui 60 $(_step ui) || _fail_with_cursor "application services (ui)"
    _complete_step $(_step ui)
    _do_wait router 60 $(_step router) || _fail_with_cursor "application services (router)"
    _complete_step $(_step router)

    # ── Phase: Nginx (must be before templates — git push needs HTTPS domain) ──
    $COMPOSE_CMD up -d nginx 2>&1 | verbose_filter
    _do_wait nginx 30 $(_step nginx) || _fail_with_cursor "nginx"
    setup_hosts
    sleep 5  # Let nginx fully initialize HTTPS proxy before git push
    _complete_step $(_step nginx)

    # ── Phase: Templates ──
    import_templates || log_warn "Template import failed — you can retry with: ./setup.sh --import-templates"
    _complete_step $(_step templates)

    # ── Phase: Cluster registration (moved inline) ──
    _si=$(_step cluster)
    [[ "$_has_tty" == "true" ]] && { _update_step "$_si" "active" 0; _redraw; }
    register_cluster || log_warn "Cluster registration failed — you can retry later with: ./deploy/setup.sh --resume"
    _complete_step "$_si"

    if [[ "$_has_tty" == "true" ]]; then
        printf "\033[?25h" >/dev/tty  # Show cursor
        echo "" >/dev/tty
    else
        progress_done
        echo ""
    fi
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
    # files/ can be in PROJECT_DIR (when running from deploy/setup.sh) or DEPLOY_DIR (root setup.sh)
    local files_dir="${PROJECT_DIR}/files"
    [[ ! -d "$files_dir" ]] && files_dir="${DEPLOY_DIR}/files"
    local gitea_admin="${ADMIN_EMAIL%%@*}"
    local gitea_pass="${ADMIN_PASSWORD}"
    local gitea_api="http://localhost:3000/api/v1"
    local api_base="http://localhost:6802"   # API container internal port
    local admin_base="http://localhost:6804" # Admin port (no auth required)

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

    # Check API is healthy (API container has wget, not curl)
    local api_health
    api_health=$(docker compose -f "$compose_file" exec -T api wget -q -O- "${api_base}/health" 2>/dev/null || echo "")
    if ! echo "$api_health" | grep -q '"healthy"'; then
        log_warn "API not healthy — skipping template import"
        return 1
    fi

    # Pre-flight: verify Gitea is reachable via HTTPS domain (DNS + TLS chain)
    local _gitea_https_ok=false
    local _preflight_attempts=0
    while [[ $_preflight_attempts -lt 15 ]]; do
        if curl -sfk -o /dev/null --max-time 5 "https://${GITEA_DOMAIN}/api/v1/version" 2>/dev/null; then
            _gitea_https_ok=true
            break
        fi
        _preflight_attempts=$((_preflight_attempts + 1))
        log_info "Waiting for Gitea HTTPS to be reachable (${_preflight_attempts}/15)..."
        sleep 2
    done
    if [[ "$_gitea_https_ok" != "true" ]]; then
        log_warn "Gitea HTTPS not reachable via ${GITEA_DOMAIN} — git push may fail"
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

        # Extract and push (with retry)
        local tmp_dir
        tmp_dir=$(mktemp -d)
        tar xzf "$tar_file" -C "$tmp_dir" --strip-components=1 2>/dev/null || tar xzf "$tar_file" -C "$tmp_dir"

        local _push_ok=false
        local _push_attempt=0
        while [[ $_push_attempt -lt 3 ]]; do
            _push_attempt=$((_push_attempt + 1))
            (
                trap - EXIT  # Clear inherited /dev/tty trap from common.sh
                cd "$tmp_dir"
                export GIT_AUTHOR_NAME="Launchpad" GIT_AUTHOR_EMAIL="launchpad@${DOMAIN}"
                export GIT_COMMITTER_NAME="Launchpad" GIT_COMMITTER_EMAIL="launchpad@${DOMAIN}"
                # Only init on first attempt
                if [[ ! -d .git ]]; then
                    git init -b main >/dev/null 2>&1
                    git add -A >/dev/null 2>&1
                    git commit -m "Initial import" >/dev/null 2>&1
                fi

                remote_url="https://${gitea_admin}:${gitea_pass}@${GITEA_DOMAIN}/launchpad/${repo_name}.git"
                if [[ "${SSL_MODE}" == "selfsigned" ]]; then
                    GIT_SSL_NO_VERIFY=1 git push -f "$remote_url" main >/dev/null 2>&1
                else
                    git push -f "$remote_url" main >/dev/null 2>&1
                fi
            ) && _push_ok=true
            [[ "$_push_ok" == "true" ]] && break
            log_warn "Git push failed for ${repo_name} (attempt ${_push_attempt}/3), retrying in 3s..."
            sleep 3
        done
        rm -rf "$tmp_dir"

        if [[ "$_push_ok" == "true" ]]; then
            log_ok "$MSG_DEPLOY_TEMPLATES_REPO ${repo_name}"
        else
            log_warn "$MSG_DEPLOY_TEMPLATES_REPO ${repo_name} — push failed after 3 attempts"
        fi
    done

    # Phase 2: Import YAML templates via admin API (port 6804, no auth required)
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

        # Validate via admin API (no auth needed)
        local validate_resp
        validate_resp=$(docker compose -f "$compose_file" exec -T api \
            wget -q -O- --post-data="{\"yaml\": ${json_yaml}}" \
            --header="Content-Type: application/json" \
            "${admin_base}/api/templates/yaml/validate" 2>&1 || echo "WGET_FAILED")

        if [[ "$validate_resp" == "WGET_FAILED" ]]; then
            log_warn "$MSG_DEPLOY_TEMPLATES_YAML ${template_name} — validation request failed"
            continue
        fi
        # Check if valid:false or errors array is non-empty
        if echo "$validate_resp" | grep -q '"valid":false'; then
            log_warn "$MSG_DEPLOY_TEMPLATES_YAML ${template_name} — validation failed: $validate_resp"
            continue
        fi

        # Import via admin API
        local import_resp
        import_resp=$(docker compose -f "$compose_file" exec -T api \
            wget -q -O- --post-data="{\"yaml\": ${json_yaml}, \"isOfficial\": true}" \
            --header="Content-Type: application/json" \
            "${admin_base}/api/templates/yaml/create" 2>&1 || echo "WGET_FAILED")

        if [[ "$import_resp" == "WGET_FAILED" ]] || echo "$import_resp" | grep -q '"success":false'; then
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

    # Detect tty — use Style C if available, otherwise plain stdout
    local _has_tty=false
    if [ -t 1 ] || (echo -n "" >/dev/tty 2>/dev/null); then
        _has_tty=true
    fi

    if [[ "$_has_tty" == "true" ]]; then
        clear >/dev/tty 2>/dev/null || true
        print_brand_header "${GREEN}${BOLD}✓ $MSG_DEPLOY_COMPLETE${NC}"

        printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty

        print_accent_block green "$MSG_OUT_URLS" \
            "${DIM}$MSG_OUT_DASHBOARD  https://${LAUNCHPAD_DOMAIN}${NC}" \
            "${DIM}$MSG_OUT_GITEA      https://${GITEA_DOMAIN}${NC}" \
            "${DIM}$MSG_OUT_ADMIN      https://${LAUNCHPAD_DOMAIN}/admin${NC}"

        print_accent_block red "$MSG_OUT_CREDENTIALS" \
            "${DIM}$MSG_OUT_EMAIL      ${ADMIN_EMAIL}${NC}" \
            "${DIM}$MSG_OUT_PASSWORD   ${ADMIN_PASSWORD}${NC}"

        local k8s_status
        if [[ "${CLUSTER_REGISTERED:-false}" == "true" ]]; then
            k8s_status="${GREEN}✓${NC} ${DIM}Registered${NC}"
        else
            k8s_status="${YELLOW}⚠ Not registered${NC}"
        fi
        print_accent_block blue "$MSG_OUT_K8S" \
            "${DIM}$MSG_OUT_MODE       ${K8S_MODE}${NC}" \
            "$k8s_status"

        print_accent_block yellow "$MSG_OUT_DNS" \
            "${DIM}$MSG_OUT_DNS_MSG ($ip):${NC}" \
            "${DIM}├─ ${LAUNCHPAD_DOMAIN}${NC}" \
            "${DIM}├─ ${GITEA_DOMAIN}${NC}" \
            "${DIM}└─ *.${DOMAIN}${NC}"

        echo "" >/dev/tty
        printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
        printf "  ${DIM}$MSG_OUT_CONFIG:  generated/${NC}\n" >/dev/tty
        printf "  ${DIM}$MSG_OUT_LOGS:    docker compose -f generated/docker-compose.yml logs -f${NC}\n" >/dev/tty
        printf "  ${DIM}$MSG_OUT_RECONFIG:  ./setup.sh --reconfigure${NC}\n" >/dev/tty
        printf "  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n" >/dev/tty
    else
        # Fallback: plain stdout (for non-interactive/SSH/--config mode)
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
    fi
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
    docker compose -f "$compose_file" up -d --force-recreate
    docker compose -f "$compose_file" restart nginx
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

uninstall_all() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"

    # Load config to know K8S_MODE
    if [[ -f "${DEPLOY_DIR}/generated/.setup.conf" ]]; then
        source "${DEPLOY_DIR}/generated/.setup.conf"
    fi

    log_warn "$MSG_UNINSTALL_ALL_WARN"
    echo ""
    # Skip confirmation if no tty (non-interactive / piped input)
    if [ -t 0 ] || [ -t 1 ]; then
        if ! ask_confirm "$MSG_UNINSTALL_ALL_CONFIRM" "N"; then
            return 0
        fi
    else
        # Non-interactive: require "Y" on stdin
        local answer
        read -r answer 2>/dev/null || answer=""
        if [[ ! "$answer" =~ ^[Yy] ]]; then
            log_warn "Non-interactive mode: pass 'Y' on stdin to confirm"
            return 0
        fi
    fi

    # Step 1: Stop services and remove volumes
    if [[ -f "$compose_file" ]]; then
        log_info "Stopping services..."
        docker compose -f "$compose_file" down -v 2>/dev/null || true
        log_ok "Services and volumes removed"
    fi

    # Step 2: Remove heartbeat cron, script, env
    if crontab -l 2>/dev/null | grep -q 'k3s-heartbeat'; then
        crontab -l 2>/dev/null | grep -v 'k3s-heartbeat' | crontab - 2>/dev/null || true
        log_ok "Heartbeat cron removed"
    fi
    rm -f /usr/local/bin/k3s-heartbeat.sh
    rm -f /etc/default/k3s-heartbeat

    # Step 3: Uninstall k3s if builtin
    if [[ -f /usr/local/bin/k3s-uninstall.sh ]]; then
        if [[ "${K8S_MODE:-builtin}" == "builtin" ]]; then
            log_info "Uninstalling k3s..."
            /usr/local/bin/k3s-uninstall.sh 2>/dev/null || true
            log_ok "k3s uninstalled"
        fi
    fi

    # Step 3b: Remove dnsmasq launchpad config
    if [[ -f /etc/dnsmasq.d/launchpad.conf ]]; then
        rm -f /etc/dnsmasq.d/launchpad.conf
        systemctl restart dnsmasq 2>/dev/null || true
        log_ok "dnsmasq config removed"
    fi

    # Step 4: Remove generated directory
    if [[ -d "${DEPLOY_DIR}/generated" ]]; then
        rm -rf "${DEPLOY_DIR}/generated"
        log_ok "Generated config removed"
    fi

    log_done "$MSG_UNINSTALL_ALL_DONE"
}

restart_service() {
    local target="$1"
    local valid_targets="api ui router gateway nginx gitea all"
    if [[ ! " $valid_targets " =~ " $target " ]]; then
        log_error "Unknown service: $target (valid: $valid_targets)"
        exit 1
    fi
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    if [[ ! -f "$compose_file" ]]; then
        log_error "No deployment found. Run ./setup.sh first."
        exit 1
    fi
    if [[ "$target" == "all" ]]; then
        docker compose -f "$compose_file" restart
    else
        docker compose -f "$compose_file" restart "$target"
    fi
    log_done "Restarted: $target"
}

# Recreate services to reload .env files, then restart nginx to refresh DNS cache.
# Usage: recreate_services service1 [service2 ...]
recreate_services() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    if [[ ! -f "$compose_file" ]]; then
        log_error "No deployment found. Run ./setup.sh first."
        exit 1
    fi
    docker compose -f "$compose_file" up -d --force-recreate "$@"
    docker compose -f "$compose_file" restart nginx
    for svc in "$@"; do
        wait_for_healthy "$svc" 60 || true
    done
    log_done "Recreated: $* (nginx restarted)"
}

