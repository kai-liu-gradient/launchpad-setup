package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (e *Engine) startPostgres(ctx context.Context) error {
	// Idempotency: check if already running and healthy
	if out, err := RunWithOutput(ctx, "check-postgres", 10*time.Second,
		"docker", "compose", "-f", e.output+"/docker-compose.yml",
		"ps", "--format", "{{.Health}}", "postgres"); err == nil {
		if strings.TrimSpace(out) == "healthy" {
			return nil
		}
	}

	if err := RunWithTimeout(ctx, "start-postgres", 30*time.Second,
		"docker", "compose", "-f", e.output+"/docker-compose.yml",
		"up", "-d", "postgres"); err != nil {
		return fmt.Errorf("starting postgres: %w", err)
	}

	return WaitForDocker(ctx, e.output, "postgres", 30*time.Second)
}

func (e *Engine) startRedis(ctx context.Context) error {
	// Idempotency: check if already running and healthy
	if out, err := RunWithOutput(ctx, "check-redis", 10*time.Second,
		"docker", "compose", "-f", e.output+"/docker-compose.yml",
		"ps", "--format", "{{.Health}}", "redis"); err == nil {
		if strings.TrimSpace(out) == "healthy" {
			return nil
		}
	}

	if err := RunWithTimeout(ctx, "start-redis", 30*time.Second,
		"docker", "compose", "-f", e.output+"/docker-compose.yml",
		"up", "-d", "redis"); err != nil {
		return fmt.Errorf("starting redis: %w", err)
	}

	return WaitForDocker(ctx, e.output, "redis", 15*time.Second)
}

func (e *Engine) initDatabases(ctx context.Context) error {
	if e.cfg.Database.Mode != "builtin" {
		return nil // external mode: user manages DB
	}

	// Idempotency: check if schemas already exist
	out, err := DockerExec(ctx, e.output, "postgres",
		"psql", "-U", "postgres", "-d", "launchpad", "-t", "-c",
		"SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'launchpad_%'")
	if err == nil {
		count := strings.TrimSpace(out)
		if count == "6" {
			return nil // already initialized
		}
	}

	// Create launchpad database if not exists
	DockerExec(ctx, e.output, "postgres", //nolint:errcheck
		"psql", "-U", "postgres", "-d", "postgres", "-c",
		"SELECT 'CREATE DATABASE launchpad' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'launchpad')\\gexec")

	// Create schemas and users
	type schemaInfo struct {
		name     string
		password string
	}
	schemas := []schemaInfo{
		{"main", e.sec.DBPasswordMain},
		{"monitoring", e.sec.DBPasswordMonitoring},
		{"events", e.sec.DBPasswordEvents},
		{"billing", e.sec.DBPasswordBilling},
		{"stats", e.sec.DBPasswordStats},
		{"gateway", e.sec.DBPasswordGateway},
	}

	for _, s := range schemas {
		schema := "launchpad_" + s.name
		user := "launchpad_" + s.name + "user"
		sql := fmt.Sprintf(`
CREATE SCHEMA IF NOT EXISTS %s;
DO $$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '%s') THEN
        CREATE ROLE %s LOGIN PASSWORD '%s';
    END IF;
END $$;
GRANT USAGE ON SCHEMA %s TO %s;
GRANT CREATE ON SCHEMA %s TO %s;
ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON TABLES TO %s;
ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON SEQUENCES TO %s;
ALTER ROLE %s SET search_path TO %s, public;`,
			schema, user, user, s.password,
			schema, user, schema, user,
			schema, user, schema, user,
			user, schema)
		DockerExec(ctx, e.output, "postgres", //nolint:errcheck
			"psql", "-U", "postgres", "-d", "launchpad", "-c", sql)
		e.send(StepEvent{Step: "Initializing databases", Status: Running, Detail: "Schema: " + schema})
	}

	// Create Gitea database
	giteaSQL := fmt.Sprintf(`
SELECT 'CREATE DATABASE gitea' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'gitea')\gexec
DO $$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'gitea') THEN
        CREATE ROLE gitea LOGIN PASSWORD '%s';
    END IF;
END $$;
GRANT ALL PRIVILEGES ON DATABASE gitea TO gitea;
ALTER DATABASE gitea OWNER TO gitea;`, e.sec.DBPasswordGitea)
	DockerExec(ctx, e.output, "postgres", //nolint:errcheck
		"psql", "-U", "postgres", "-c", giteaSQL)

	// Grant schema permissions in gitea database
	DockerExec(ctx, e.output, "postgres", //nolint:errcheck
		"psql", "-U", "postgres", "-d", "gitea", "-c",
		"GRANT ALL ON SCHEMA public TO gitea;")

	e.send(StepEvent{Step: "Initializing databases", Status: Running, Detail: "Gitea database"})

	// Run Prisma migrations
	prismaSchemas := []struct {
		schema  string
		service string
	}{
		{"prisma/schema.prisma", "api"},
		{"prisma/monitoring.prisma", "api"},
		{"prisma/schema-billing.prisma", "api"},
		{"prisma/schema-events.prisma", "api"},
		{"prisma/schema-stats.prisma", "api"},
	}
	for _, ps := range prismaSchemas {
		e.send(StepEvent{Step: "Initializing databases", Status: Running, Detail: "Prisma push: " + ps.schema})
		if _, err := DockerRun(ctx, e.output, ps.service,
			"npx", "prisma", "db", "push", "--schema", ps.schema); err != nil {
			return fmt.Errorf("prisma db push %s: %w", ps.schema, err)
		}
	}

	// Gateway schema
	e.send(StepEvent{Step: "Initializing databases", Status: Running, Detail: "Prisma push: gateway"})
	if _, err := DockerRun(ctx, e.output, "gateway", "npm", "run", "db:push"); err != nil {
		return fmt.Errorf("gateway db push: %w", err)
	}

	return nil
}
