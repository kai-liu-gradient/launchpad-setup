# Init Logic Refactor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the initialization flow to add k8s component setup (ingress-nginx, CoreDNS, Kyverno), fix URLs for k8s pod access, and eliminate iptables dependency.

**Architecture:** New `setup_k8s_components()` phase inserted between `setup_certificates()` and `deploy_services()`. Helm installs ingress-nginx and Kyverno in builtin k3s mode. CoreDNS configured dynamically with ingress-nginx ClusterIP. Only `GITEA_GIT_SSH` and `DEFAULT_BACKEND` URL values change.

**Tech Stack:** Bash, Helm 3, kubectl, k3s, Kyverno, CoreDNS, Docker Compose

**Spec:** `docs/superpowers/specs/2026-03-17-init-logic-refactor-design.md`

---

## Chunk 1: Static Template Files

### Task 1: Create values-builtin.yml

**Files:**
- Create: `deploy/templates/values-builtin.yml`

- [ ] **Step 1: Create the file**

```yaml
controller:
  replicaCount: 1

  ingressClass: nginx

  ingressClassResource:
    name: nginx
    default: true

  config:
    enable-real-ip: "true"
    use-forwarded-headers: "true"
    compute-full-forwarded-for: "true"

    worker-processes: "auto"
    max-worker-connections: "30000"

    proxy-body-size: 50m
    keep-alive: "300"
    keep-alive-requests: "65535"

    ssl-redirect: "false"

    log-format-escape-json: "true"
    log-format-upstream: |
      { "time": "$time_iso8601",
        "remote_addr": "$remote_addr",
        "x-forward-for": "$proxy_add_x_forwarded_for",
        "request_id": "$request_id",
        "status": $status,
        "host": "$host",
        "path": "$uri",
        "method": "$request_method",
        "request_time": $request_time,
        "upstream_addr": "$upstream_addr"
      }

  service:
    type: NodePort
    nodePorts:
      http: 30080
      https: 30443
```

- [ ] **Step 2: Commit**

```bash
git add deploy/templates/values-builtin.yml
git commit -m "feat: add ingress-nginx Helm values for builtin k3s mode"
```

---

### Task 2: Create CoreDNS custom ConfigMap template

**Files:**
- Create: `deploy/templates/coredns-custom.yaml.template`

- [ ] **Step 1: Create the file**

Note: `envsubst` will replace `${HOST_IP}`, `${INGRESS_NGINX_CLUSTER_IP}`, `${DOMAIN}`, `${DOMAIN_ESCAPED}`, `${LAUNCHPAD_DOMAIN}`, `${GITEA_DOMAIN}`. The CoreDNS template syntax `{{ .Name }}` must NOT be touched by envsubst — this is ensured by limiting the variable list in the `envsubst` call (done in Task 6).

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns-custom
  namespace: kube-system
data:
  # Specific domains → host IP (Docker nginx)
  # These server blocks take priority over the wildcard because CoreDNS
  # selects the most specific zone match first.
  launchpad-specific.server: |
    ${LAUNCHPAD_DOMAIN}:53 ${GITEA_DOMAIN}:53 {
        hosts {
            ${HOST_IP} ${LAUNCHPAD_DOMAIN}
            ${HOST_IP} ${GITEA_DOMAIN}
            ttl 60
        }
    }

  # Wildcard subdomains → ingress-nginx ClusterIP (project pods)
  wildcard.server: |
    ${DOMAIN}:53 {
        hosts {
            ${INGRESS_NGINX_CLUSTER_IP} ${DOMAIN}
            ttl 60
            fallthrough
        }
        template IN A {
            match ".*\.${DOMAIN_ESCAPED}\.$"
            answer "{{ .Name }} 60 IN A ${INGRESS_NGINX_CLUSTER_IP}"
        }
    }
```

- [ ] **Step 2: Commit**

```bash
git add deploy/templates/coredns-custom.yaml.template
git commit -m "feat: add CoreDNS custom ConfigMap template with dynamic ClusterIP"
```

---

### Task 3: Create Kyverno ClusterPolicy manifests

**Files:**
- Create: `deploy/templates/kyverno-sync-ca.yaml`
- Create: `deploy/templates/kyverno-inject-ca.yaml`

- [ ] **Step 1: Create kyverno-sync-ca.yaml**

This policy watches Namespace creation events and clones the `launchpad-ca-cert` ConfigMap from `kube-system` into each new namespace.

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: sync-launchpad-ca-cert
spec:
  generateExisting: true
  rules:
    - name: sync-ca-configmap
      match:
        any:
          - resources:
              kinds:
                - Namespace
      exclude:
        any:
          - resources:
              namespaces:
                - kube-system
                - kyverno
      generate:
        synchronize: true
        apiVersion: v1
        kind: ConfigMap
        name: launchpad-ca-cert
        namespace: "{{request.object.metadata.name}}"
        clone:
          namespace: kube-system
          name: launchpad-ca-cert
```

- [ ] **Step 2: Create kyverno-inject-ca.yaml**

This policy mutates every Pod (outside system namespaces) to mount the CA cert and set `NODE_EXTRA_CA_CERTS`.

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: inject-launchpad-ca-cert
spec:
  rules:
    - name: inject-ca-cert
      match:
        any:
          - resources:
              kinds:
                - Pod
      exclude:
        any:
          - resources:
              namespaces:
                - kube-system
                - kyverno
                - ingress-nginx
      mutate:
        patchStrategicMerge:
          spec:
            volumes:
              - name: launchpad-ca-cert
                configMap:
                  name: launchpad-ca-cert
            containers:
              - (name): "*"
                volumeMounts:
                  - name: launchpad-ca-cert
                    mountPath: /etc/ssl/certs/launchpad-ca.pem
                    subPath: ca.pem
                    readOnly: true
                env:
                  - name: NODE_EXTRA_CA_CERTS
                    value: /etc/ssl/certs/launchpad-ca.pem
```

- [ ] **Step 3: Commit**

```bash
git add deploy/templates/kyverno-sync-ca.yaml deploy/templates/kyverno-inject-ca.yaml
git commit -m "feat: add Kyverno policies for CA cert distribution and pod injection"
```

---

## Chunk 2: k8s-components.sh Module

### Task 4: Create k8s-components.sh with install_ingress_nginx()

**Files:**
- Create: `deploy/scripts/lib/k8s-components.sh`

- [ ] **Step 1: Create the file with install_ingress_nginx()**

```bash
#!/bin/bash
# Kubernetes component setup (ingress-nginx, CoreDNS, Kyverno)
# Only runs in K8S_MODE=builtin

install_ingress_nginx() {
    if helm status ingress-nginx -n ingress-nginx &>/dev/null; then
        log_ok "ingress-nginx already installed"
        return 0
    fi

    log_info "Installing ingress-nginx via Helm..."
    helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
    helm repo update

    helm install ingress-nginx ingress-nginx/ingress-nginx \
        -n ingress-nginx --create-namespace \
        -f "${DEPLOY_DIR}/templates/values-builtin.yml"

    log_info "Waiting for ingress-nginx controller to be ready..."
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=120s

    log_ok "ingress-nginx installed"
}
```

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/k8s-components.sh
git commit -m "feat: add k8s-components.sh with install_ingress_nginx()"
```

---

### Task 5: Add install_kyverno() and setup_cert_distribution()

**Files:**
- Modify: `deploy/scripts/lib/k8s-components.sh`

- [ ] **Step 1: Append install_kyverno()**

Add after `install_ingress_nginx()`:

```bash
install_kyverno() {
    if helm status kyverno -n kyverno &>/dev/null; then
        log_ok "Kyverno already installed"
        return 0
    fi

    log_info "Installing Kyverno via Helm..."
    helm repo add kyverno https://kyverno.github.io/kyverno/
    helm install kyverno kyverno/kyverno \
        -n kyverno --create-namespace

    log_info "Waiting for Kyverno admission controller to be ready..."
    kubectl wait --namespace kyverno \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=admission-controller \
        --timeout=120s

    log_ok "Kyverno installed"
}
```

Note: `helm repo update` is NOT called again here — it was already called in `install_ingress_nginx()` which runs first.

- [ ] **Step 2: Append setup_cert_distribution()**

Add after `install_kyverno()`:

```bash
setup_cert_distribution() {
    local ca_cert="${DEPLOY_DIR}/generated/nginx/certs/ca.pem"
    if [[ ! -f "$ca_cert" ]]; then
        log_warn "CA certificate not found at $ca_cert — skipping cert distribution"
        return 0
    fi

    log_info "Distributing CA certificate via Kyverno..."

    # Create source ConfigMap in kube-system
    kubectl create configmap launchpad-ca-cert \
        --from-file=ca.pem="$ca_cert" \
        -n kube-system --dry-run=client -o yaml | kubectl apply -f -

    # Apply Kyverno policies
    kubectl apply -f "${DEPLOY_DIR}/templates/kyverno-sync-ca.yaml"
    kubectl apply -f "${DEPLOY_DIR}/templates/kyverno-inject-ca.yaml"

    log_ok "CA certificate distribution configured"
}
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/k8s-components.sh
git commit -m "feat: add install_kyverno() and setup_cert_distribution()"
```

---

### Task 6: Add configure_coredns() and setup_k8s_components() orchestrator

**Files:**
- Modify: `deploy/scripts/lib/k8s-components.sh`

- [ ] **Step 1: Append configure_coredns()**

Add after `setup_cert_distribution()`:

```bash
configure_coredns() {
    local host_ip
    host_ip=$(detect_internal_ip) || { log_error "Cannot detect internal IP for CoreDNS config"; return 1; }

    local cluster_ip
    cluster_ip=$(kubectl get svc ingress-nginx-controller \
        -n ingress-nginx -o jsonpath='{.spec.clusterIP}')

    if [[ -z "$cluster_ip" ]]; then
        log_error "Cannot get ingress-nginx ClusterIP"
        return 1
    fi

    log_info "Configuring CoreDNS (host=$host_ip, ingress=$cluster_ip)..."

    local domain_escaped="${DOMAIN//./\\.}"

    # Render template — limit envsubst variables to avoid clobbering CoreDNS {{ .Name }} syntax
    export HOST_IP="$host_ip"
    export INGRESS_NGINX_CLUSTER_IP="$cluster_ip"
    export DOMAIN_ESCAPED="$domain_escaped"

    envsubst '${HOST_IP} ${INGRESS_NGINX_CLUSTER_IP} ${DOMAIN} ${DOMAIN_ESCAPED} ${LAUNCHPAD_DOMAIN} ${GITEA_DOMAIN}' \
        < "${DEPLOY_DIR}/templates/coredns-custom.yaml.template" \
        > /tmp/coredns-custom.yaml

    kubectl apply -f /tmp/coredns-custom.yaml

    # Patch default CoreDNS ConfigMap: disable loop plugin, use public DNS forwarders
    # Idempotent sed: only comments out uncommented 'loop' lines
    kubectl get cm coredns -n kube-system -o yaml \
        | sed '/^[^#]*loop$/s/loop/# loop/' \
        | sed 's|forward \. /etc/resolv\.conf|forward . 223.5.5.5 8.8.8.8|' \
        | kubectl apply -f -

    # k3s natively supports coredns-custom ConfigMap convention (auto-imports)
    kubectl rollout restart deploy/coredns -n kube-system
    rm -f /tmp/coredns-custom.yaml

    log_ok "CoreDNS configured"
}
```

- [ ] **Step 2: Append setup_k8s_components() orchestrator**

Add at the end of the file:

```bash
setup_k8s_components() {
    [[ "$K8S_MODE" != "builtin" ]] && return 0

    log_info "Setting up k8s components..."

    install_ingress_nginx
    configure_coredns

    if [[ "$SSL_MODE" == "selfsigned" ]]; then
        install_kyverno
        setup_cert_distribution
    fi

    log_ok "k8s components ready"
}
```

- [ ] **Step 3: Commit**

```bash
git add deploy/scripts/lib/k8s-components.sh
git commit -m "feat: add configure_coredns() and setup_k8s_components() orchestrator"
```

---

## Chunk 3: Modify Existing Files

### Task 7: Modify render.sh — URL changes

**Files:**
- Modify: `deploy/scripts/lib/render.sh:30-31` (GITEA_GIT_SSH)
- Modify: `deploy/scripts/lib/render.sh` (DEFAULT_BACKEND, after DB URL section)

- [ ] **Step 1: Change GITEA_GIT_SSH default value**

In `deploy/scripts/lib/render.sh`, find line 31:

```bash
# Before
export GITEA_GIT_SSH="${GITEA_GIT_SSH:-git@gitea:2222}"
```

Replace with:

```bash
# After — k8s pods are the SSH clients, must resolve via domain
export GITEA_GIT_SSH="${GITEA_GIT_SSH:-git@${GITEA_DOMAIN}:2222}"
```

- [ ] **Step 2: Add DEFAULT_BACKEND override for builtin mode**

In `deploy/scripts/lib/render.sh`, find the section after the image version exports (around line 38, after `export IMAGE_VERSION_GITEA=...`). Add:

```bash
    # Override DEFAULT_BACKEND for builtin mode — interact.sh sets "localhost"
    # but builtin k3s needs ingress-nginx NodePort
    if [[ "${K8S_MODE:-}" == "builtin" ]]; then
        local host_ip
        host_ip=$(detect_internal_ip 2>/dev/null || echo "127.0.0.1")
        export DEFAULT_BACKEND="http://${host_ip}:30080"
    fi
```

- [ ] **Step 3: Verify no other references to `git@gitea:2222` exist**

Run: `grep -r 'git@gitea:2222' deploy/`

Expected: no results (only the render.sh line was changed).

- [ ] **Step 4: Commit**

```bash
git add deploy/scripts/lib/render.sh
git commit -m "fix: GITEA_GIT_SSH uses domain for k8s pod access, DEFAULT_BACKEND points to ingress-nginx"
```

---

### Task 8: Modify detect.sh — add helm check

**Files:**
- Modify: `deploy/scripts/lib/detect.sh:48` (before the error count check)

- [ ] **Step 1: Add helm check**

In `deploy/scripts/lib/detect.sh`, find the block just before `if [[ "$errors" -gt 0 ]]; then` (line 50). Insert before it:

```bash
    # helm (required for builtin k8s mode — ingress-nginx, kyverno)
    if [[ "${K8S_MODE:-builtin}" == "builtin" ]]; then
        if ! command -v helm &>/dev/null; then
            log_error "helm is not installed (required for builtin k8s mode)"
            ((errors++))
        else
            log_ok "helm: $(helm version --short 2>/dev/null)"
        fi
    fi
```

Note: At `check_environment()` time, `K8S_MODE` may not be set yet (it's set in `interact.sh`). The default `builtin` ensures helm is checked unless the user has already set external mode via `--config`.

- [ ] **Step 2: Commit**

```bash
git add deploy/scripts/lib/detect.sh
git commit -m "feat: check_environment() verifies helm is installed for builtin mode"
```

---

### Task 9: Modify setup.sh — integrate new phase

**Files:**
- Modify: `deploy/setup.sh:56` (add source for k8s-components.sh)
- Modify: `deploy/setup.sh:87-88` (insert setup_k8s_components between setup_certificates and deploy_services)

- [ ] **Step 1: Add source for k8s-components.sh**

In `deploy/setup.sh`, find line 56 (`source "${DEPLOY_DIR}/scripts/lib/k3s.sh"`). Add after it:

```bash
        source "${DEPLOY_DIR}/scripts/lib/k8s-components.sh"
```

- [ ] **Step 2: Insert setup_k8s_components() call**

In `deploy/setup.sh`, find line 88-89:

```bash
        setup_kubernetes
        setup_certificates
        deploy_services
```

Replace with:

```bash
        setup_kubernetes
        setup_certificates
        setup_k8s_components
        deploy_services
```

- [ ] **Step 3: Verify the full flow in setup.sh**

Read `deploy/setup.sh` and confirm the install/reconfigure case now runs:
1. `setup_kubernetes`
2. `setup_certificates`
3. `setup_k8s_components` (new)
4. `deploy_services`
5. `register_cluster`

- [ ] **Step 4: Commit**

```bash
git add deploy/setup.sh
git commit -m "feat: integrate setup_k8s_components() into main deployment flow"
```

---

### Task 10: Final review and integration commit

**Files:**
- All files from Tasks 1-9

- [ ] **Step 1: Verify all new files exist**

Run:
```bash
ls -la deploy/templates/values-builtin.yml \
       deploy/templates/coredns-custom.yaml.template \
       deploy/templates/kyverno-sync-ca.yaml \
       deploy/templates/kyverno-inject-ca.yaml \
       deploy/scripts/lib/k8s-components.sh
```

Expected: all 5 files listed.

- [ ] **Step 2: Verify setup.sh sources k8s-components.sh**

Run: `grep 'k8s-components' deploy/setup.sh`

Expected: `source "${DEPLOY_DIR}/scripts/lib/k8s-components.sh"`

- [ ] **Step 3: Verify render.sh changes**

Run: `grep 'GITEA_GIT_SSH' deploy/scripts/lib/render.sh`

Expected: contains `git@${GITEA_DOMAIN}:2222` (not `git@gitea:2222`)

Run: `grep 'DEFAULT_BACKEND' deploy/scripts/lib/render.sh`

Expected: contains `http://${host_ip}:30080` in builtin conditional

- [ ] **Step 4: Verify detect.sh has helm check**

Run: `grep 'helm' deploy/scripts/lib/detect.sh`

Expected: contains helm version check

- [ ] **Step 5: Verify setup.sh flow order**

Run: `grep -A5 'setup_kubernetes' deploy/setup.sh`

Expected: `setup_kubernetes` → `setup_certificates` → `setup_k8s_components` → `deploy_services`

- [ ] **Step 6: Shellcheck (if available)**

Run: `shellcheck deploy/scripts/lib/k8s-components.sh 2>/dev/null || echo "shellcheck not installed, skipping"`

Expected: no errors (warnings OK)
