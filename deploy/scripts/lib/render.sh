#!/bin/bash
# Template rendering

render_templates() {
    log_info "Rendering configuration templates..."

    # Create output directories
    mkdir -p "${DEPLOY_DIR}/generated/launchpad/config"
    mkdir -p "${DEPLOY_DIR}/generated/launchpad/cron"
    mkdir -p "${DEPLOY_DIR}/generated/gateway"
    mkdir -p "${DEPLOY_DIR}/generated/nginx/certs"

    # Compute derived variables
    export EXTERNAL_DOMAIN="https://${LAUNCHPAD_DOMAIN}"
    export PUBLIC_DOMAIN="${LAUNCHPAD_DOMAIN}"
    export PUBLIC_URL="https://${LAUNCHPAD_DOMAIN}"
    export EXTERNAL_API_URL="https://${LAUNCHPAD_DOMAIN}/api"
    export INTERNAL_API_URL="https://${LAUNCHPAD_DOMAIN}/api"
    export INTERNAL_ADMIN_URL="https://${LAUNCHPAD_DOMAIN}/admin/adminapi"
    export CORS_ORIGINS="https://${LAUNCHPAD_DOMAIN}"
    export INGRESS_DOMAIN="${DOMAIN}"
    export BASE_DOMAIN="${DOMAIN}"
    export ANI_CODE_RELEASE_URL="https://${GITEA_DOMAIN}/launchpad/ani-code/archive/main.tar.gz"
    export GITEA_ROOT_URL="https://${GITEA_DOMAIN}"
    export ENTRA_REDIRECT_URI="https://${LAUNCHPAD_DOMAIN}/api/v1/auth/microsoft/callback"
    export GATEWAY_PUBLIC_URL="https://${LAUNCHPAD_DOMAIN}/gatewayproxy"

    # Internal service URLs: use Docker DNS for compose-local services, external URLs otherwise
    export ROUTER_LOCAL_URL="${ROUTER_LOCAL_URL:-http://router:6580}"
    export ANI_CODE_GATEWAY_URL="${ANI_CODE_GATEWAY_URL:-${GATEWAY_PUBLIC_URL}}"
    # k8s pods are the SSH clients — must use resolvable domain, not Docker DNS
    export GITEA_GIT_SSH="${GITEA_GIT_SSH:-git@${GITEA_DOMAIN}:2222}"

    # Resolve image versions
    export IMAGE_VERSION_API="${IMAGE_VERSION_API:-$IMAGE_VERSION}"
    export IMAGE_VERSION_UI="${IMAGE_VERSION_UI:-$IMAGE_VERSION}"
    export IMAGE_VERSION_ROUTER="${IMAGE_VERSION_ROUTER:-$IMAGE_VERSION}"
    export IMAGE_VERSION_GATEWAY="${IMAGE_VERSION_GATEWAY:-$IMAGE_VERSION}"
    export IMAGE_VERSION_GITEA="${IMAGE_VERSION_GITEA:-$IMAGE_VERSION}"

    # Override DEFAULT_BACKEND for builtin mode — interact.sh sets "localhost"
    # but builtin k3s needs ingress-nginx NodePort
    if [[ "${K8S_MODE:-}" == "builtin" ]]; then
        local host_ip
        host_ip=$(detect_internal_ip 2>/dev/null || echo "127.0.0.1")
        export DEFAULT_BACKEND="http://${host_ip}:30080"
    fi

    # Build database URLs
    if [[ "$DB_MODE" == "builtin" ]]; then
        export DATABASE_URL="postgresql://launchpad_mainuser:${DB_PASSWORD_MAIN}@postgres:5432/launchpad?schema=launchpad_main"
        export MONITORING_DATABASE_URL="postgresql://launchpad_monitoringuser:${DB_PASSWORD_MONITORING}@postgres:5432/launchpad?schema=launchpad_monitoring"
        export EVENTS_DATABASE_URL="postgresql://launchpad_eventsuser:${DB_PASSWORD_EVENTS}@postgres:5432/launchpad?schema=launchpad_events"
        export BILLING_DATABASE_URL="postgresql://launchpad_billinguser:${DB_PASSWORD_BILLING}@postgres:5432/launchpad?schema=launchpad_billing"
        export STATS_DATABASE_URL="postgresql://launchpad_statsuser:${DB_PASSWORD_STATS}@postgres:5432/launchpad?schema=launchpad_stats"
        export GATEWAY_DATABASE_URL="postgresql://launchpad_gatewayuser:${DB_PASSWORD_GATEWAY}@postgres:5432/launchpad?schema=launchpad_gateway"
    else
        # External mode: user provided full DATABASE_URLs during interaction
        export DATABASE_URL MONITORING_DATABASE_URL EVENTS_DATABASE_URL
        export BILLING_DATABASE_URL STATS_DATABASE_URL GATEWAY_DATABASE_URL

        # Parse Gitea DB URL into individual fields for Gitea env vars
        if [[ -n "${GITEA_DATABASE_URL:-}" ]]; then
            local gitea_url_body="${GITEA_DATABASE_URL#postgresql://}"
            GITEA_DB_USER="${gitea_url_body%%:*}"
            local rest="${gitea_url_body#*:}"
            GITEA_DB_PASSWORD="${rest%%@*}"
            rest="${rest#*@}"
            GITEA_DB_HOST="${rest%%:*}"
            rest="${rest#*:}"
            GITEA_DB_PORT="${rest%%/*}"
            GITEA_DB_NAME="${rest#*/}"
            GITEA_DB_NAME="${GITEA_DB_NAME%%\?*}"
        fi
    fi

    # Export all variables for envsubst
    export DOMAIN SUBDOMAIN LAUNCHPAD_DOMAIN GITEA_DOMAIN
    export JWT_SECRET JWT_REFRESH_SECRET SESSION_SECRET
    export ENCRYPTION_KEY CLAUDE_CREDENTIALS_ENCRYPTION_KEY SSH_KEY_ENCRYPTION_SECRET
    export ZT_PRIVATE_KEY ZT_PUBLIC_KEY
    export REDIS_HOST REDIS_PORT REDIS_PASSWORD
    export DEFAULT_BACKEND STORAGE_CLASS
    export ANI_CODE_GATEWAY_API_KEY ANI_CODE_GATEWAY_URL LAUNCHPAD_INTERNAL_SECRET
    export ROUTER_LOCAL_URL GITEA_GIT_SSH
    export ADMIN_EMAIL IMAGE_REGISTRY
    export IMAGE_VERSION_API IMAGE_VERSION_UI IMAGE_VERSION_ROUTER IMAGE_VERSION_GATEWAY IMAGE_VERSION_GITEA

    # Render .env files
    envsubst < "${DEPLOY_DIR}/templates/env.template" > "${DEPLOY_DIR}/generated/launchpad/.env"
    envsubst < "${DEPLOY_DIR}/templates/env.gateway.template" > "${DEPLOY_DIR}/generated/gateway/.env"
    envsubst '${DOMAIN} ${LAUNCHPAD_DOMAIN} ${GITEA_DOMAIN}' < "${DEPLOY_DIR}/templates/nginx.conf.template" > "${DEPLOY_DIR}/generated/nginx/nginx.conf"
    envsubst < "${DEPLOY_DIR}/templates/settings.yml.template" > "${DEPLOY_DIR}/generated/launchpad/config/settings.yml"
    cp "${DEPLOY_DIR}/templates/crontab" "${DEPLOY_DIR}/generated/launchpad/cron/crontab"

    # Render docker-compose.yml (conditional logic)
    render_compose

    log_ok "Templates rendered to generated/"
}

render_compose() {
    local compose_file="${DEPLOY_DIR}/generated/docker-compose.yml"
    cat > "$compose_file" <<COMPOSE_EOF
# Auto-generated by AniLaunchpad setup — do not edit manually
# Re-generate with: ./deploy/setup.sh --reconfigure

services:
COMPOSE_EOF

    # Conditionally add infrastructure services
    if [[ "$DB_MODE" == "builtin" ]]; then
        cat >> "$compose_file" <<'INFRA_EOF'
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: __POSTGRES_SUPERUSER_PASSWORD__
      POSTGRES_DB: launchpad
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d launchpad"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: >
      sh -c "redis-server
      --appendonly yes
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru
      --requirepass __REDIS_PASSWORD__"
    volumes:
      - redis-data:/data
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a __REDIS_PASSWORD__ ping"]
      interval: 10s
      timeout: 5s
      retries: 3

INFRA_EOF
    fi

    # Core services (single-quoted heredoc to prevent shell expansion)
    cat >> "$compose_file" <<'SERVICES_EOF'
  api:
    image: __IMAGE_REGISTRY__/launchpad-api:__IMAGE_VERSION_API__
    platform: linux/amd64
    restart: unless-stopped
    deploy:
      replicas: __API_REPLICAS__
      update_config:
        order: start-first
        parallelism: 1
        failure_action: rollback
      rollback_config:
        order: start-first
    env_file: ./launchpad/.env
    environment:
      - NODE_ENV=production
      - USE_REDIS=true
      - MULTI_INSTANCE=true
      - KUBERNETES_ENABLED=true
      - ANI_CODE_GATEWAY_ENABLED=true
      - GRACEFUL_SHUTDOWN_TIMEOUT_MS=30000
      - NODE_TLS_REJECT_UNAUTHORIZED=__NODE_TLS_REJECT__
    stop_grace_period: 2m
    stop_signal: SIGTERM
    volumes:
      - ./launchpad/config/settings.yml:/app/config/settings.yml:ro
      - api-logs:/app/logs
      - api-data:/app/data
      - __KUBECONFIG_PATH__:/home/nodejs/.kube/config:ro
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "node -e \"require('http').get('http://127.0.0.1:6802/api/health',r=>{process.exit(r.statusCode<500?0:1)}).on('error',()=>process.exit(1))\""]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s

  ui:
    image: __IMAGE_REGISTRY__/launchpad-ui:__IMAGE_VERSION_UI__
    platform: linux/amd64
    restart: unless-stopped
    env_file: ./launchpad/.env
    environment:
      - UI_PORT=6801
      - API_URL=http://api:6802
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "node -e \"require('net').connect(6801,'127.0.0.1',()=>process.exit(0)).on('error',()=>process.exit(1))\""]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 15s

  router:
    image: __IMAGE_REGISTRY__/launchpad-router:__IMAGE_VERSION_ROUTER__
    platform: linux/amd64
    restart: unless-stopped
    deploy:
      replicas: __ROUTER_REPLICAS__
      update_config:
        order: start-first
        parallelism: 1
        failure_action: rollback
      rollback_config:
        order: start-first
    env_file: ./launchpad/.env
    environment:
      - WORKER_PORT=6580
      - WORKER_HOST=0.0.0.0
      - LAUNCHPAD_API_URL=http://api:6802/api/v1/public
      - LAUNCHPAD_DASHBOARD_URL=__EXTERNAL_DOMAIN__
      - BASE_DOMAIN=__BASE_DOMAIN__
      - DEFAULT_BACKEND=__DEFAULT_BACKEND__
      - ZT_PUBLIC_KEY=__ZT_PUBLIC_KEY__
    depends_on:
      - api
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "node -e \"require('net').connect(6580,'127.0.0.1',()=>process.exit(0)).on('error',()=>process.exit(1))\""]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 15s

  cron:
    image: __IMAGE_REGISTRY__/launchpad-api:__IMAGE_VERSION_API__
    platform: linux/amd64
    restart: unless-stopped
    env_file: ./launchpad/.env
    user: root
    command: >
      sh -c "echo '=== Loaded crontab ===' &&
             cat /etc/crontabs/root &&
             echo '=== Starting crond ===' &&
             crond -f -l 0"
    cap_add:
      - SETGID
      - SETUID
    security_opt:
      - no-new-privileges:false
    environment:
      - NODE_ENV=production
    volumes:
      - ./launchpad/cron/crontab:/etc/crontabs/root:ro
      - cron-logs:/var/log/cron
    networks:
      - launchpad-network

  backup-worker:
    image: __IMAGE_REGISTRY__/launchpad-api:__IMAGE_VERSION_API__
    platform: linux/amd64
    restart: unless-stopped
    env_file: ./launchpad/.env
    command: ["node", "src/workers/backupWorker.js"]
    volumes:
      - worker-logs:/app/logs
    networks:
      - launchpad-network

  gateway:
    image: __IMAGE_REGISTRY__/ani-code-gateway:__IMAGE_VERSION_GATEWAY__
    platform: linux/amd64
    restart: unless-stopped
    deploy:
      replicas: __GATEWAY_REPLICAS__
    env_file: ./gateway/.env
    volumes:
      - gateway-data:/app/data
      - gateway-logs:/app/logs
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "node -e \"require('net').connect(6555,'127.0.0.1',()=>process.exit(0)).on('error',()=>process.exit(1))\""]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 15s

  gitea:
    image: gitea/gitea:__IMAGE_VERSION_GITEA__
    restart: unless-stopped
    environment:
      - GITEA__database__DB_TYPE=postgres
      - GITEA__database__HOST=__GITEA_DB_HOST__:__GITEA_DB_PORT__
      - GITEA__database__NAME=__GITEA_DB_NAME__
      - GITEA__database__USER=__GITEA_DB_USER__
      - GITEA__database__PASSWD=__GITEA_DB_PASSWORD__
      - GITEA__security__INSTALL_LOCK=true
      - GITEA__server__ROOT_URL=__GITEA_ROOT_URL__
      - GITEA__server__SSH_PORT=2222
      - GITEA__server__SSH_LISTEN_PORT=2222
      - GITEA__server__START_SSH_SERVER=true
      - GITEA__service__DISABLE_REGISTRATION=true
    volumes:
      - gitea-data:/var/lib/gitea
      - gitea-config:/etc/gitea
    ports:
      - "2222:2222"
    networks:
      - launchpad-network
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:3000/ -o /dev/null || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 15s

  heartbeat:
    image: bitnami/kubectl:latest
    restart: unless-stopped
    user: root
    entrypoint: ["/bin/bash", "-c"]
    command:
      - |
        echo "Waiting for heartbeat config..."
        while [ ! -f /heartbeat/heartbeat.env ]; do sleep 5; done
        echo "Starting heartbeat service..."
        set -a; source /heartbeat/heartbeat.env; set +a
        export RUN_ONCE=false
        exec bash /heartbeat/k3s-heartbeat.sh
    environment:
      - NODE_TLS_REJECT_UNAUTHORIZED=__NODE_TLS_REJECT__
    volumes:
      - ./heartbeat:/heartbeat:ro
      - ./heartbeat/kubeconfig:/.kube/config:ro
    networks:
      - launchpad-network

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/certs:/etc/nginx/certs:ro
    networks:
      launchpad-network:
        aliases:
          - __LAUNCHPAD_DOMAIN__
          - __GITEA_DOMAIN__
          - example.__BASE_DOMAIN__
    depends_on:
      - api
      - ui
      - router
      - gateway
      - gitea
    healthcheck:
      test: ["CMD-SHELL", "nginx -t && wget -q --spider http://127.0.0.1:80/nginx-health || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 5s

SERVICES_EOF

    # Replace __PLACEHOLDER__ tokens with actual variable values
    sed_i \
        -e "s|__IMAGE_REGISTRY__|${IMAGE_REGISTRY}|g" \
        -e "s|__IMAGE_VERSION_API__|${IMAGE_VERSION_API}|g" \
        -e "s|__IMAGE_VERSION_UI__|${IMAGE_VERSION_UI}|g" \
        -e "s|__IMAGE_VERSION_ROUTER__|${IMAGE_VERSION_ROUTER}|g" \
        -e "s|__IMAGE_VERSION_GATEWAY__|${IMAGE_VERSION_GATEWAY}|g" \
        -e "s|__IMAGE_VERSION_GITEA__|${IMAGE_VERSION_GITEA}|g" \
        -e "s|__GITEA_DB_HOST__|${GITEA_DB_HOST:-${DB_HOST:-postgres}}|g" \
        -e "s|__GITEA_DB_PORT__|${GITEA_DB_PORT:-${DB_PORT:-5432}}|g" \
        -e "s|__GITEA_DB_NAME__|${GITEA_DB_NAME:-gitea}|g" \
        -e "s|__GITEA_DB_USER__|${GITEA_DB_USER:-gitea}|g" \
        -e "s|__GITEA_DB_PASSWORD__|${GITEA_DB_PASSWORD}|g" \
        -e "s|__GITEA_ROOT_URL__|${GITEA_ROOT_URL:-https://${GITEA_DOMAIN}}|g" \
        -e "s|__REDIS_PASSWORD__|${REDIS_PASSWORD}|g" \
        -e "s|__POSTGRES_SUPERUSER_PASSWORD__|${POSTGRES_SUPERUSER_PASSWORD}|g" \
        -e "s|__KUBECONFIG_PATH__|${K8S_KUBECONFIG_PATH:-/etc/rancher/k3s/k3s.yaml}|g" \
        -e "s|__EXTERNAL_DOMAIN__|${EXTERNAL_DOMAIN}|g" \
        -e "s|__BASE_DOMAIN__|${BASE_DOMAIN}|g" \
        -e "s|__DEFAULT_BACKEND__|${DEFAULT_BACKEND}|g" \
        -e "s|__ZT_PUBLIC_KEY__|${ZT_PUBLIC_KEY}|g" \
        -e "s|__LAUNCHPAD_DOMAIN__|${LAUNCHPAD_DOMAIN}|g" \
        -e "s|__GITEA_DOMAIN__|${GITEA_DOMAIN}|g" \
        -e "s|__NODE_TLS_REJECT__|$( [[ "${SSL_MODE}" == "selfsigned" ]] && echo 0 || echo 1 )|g" \
        -e "s|__API_REPLICAS__|${API_REPLICAS:-1}|g" \
        -e "s|__ROUTER_REPLICAS__|${ROUTER_REPLICAS:-1}|g" \
        -e "s|__GATEWAY_REPLICAS__|${GATEWAY_REPLICAS:-1}|g" \
        "$compose_file"

    # Networks and volumes
    cat >> "$compose_file" <<'NET_EOF'

networks:
  launchpad-network:
    driver: bridge

volumes:
  postgres-data:
  redis-data:
  api-logs:
  api-data:
  cron-logs:
  worker-logs:
  gateway-data:
  gateway-logs:
  gitea-data:
  gitea-config:
NET_EOF

}
