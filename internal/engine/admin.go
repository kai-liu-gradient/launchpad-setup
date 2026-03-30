package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (e *Engine) registerAdmin(ctx context.Context) error {
	email := e.cfg.AdminEmail
	password := e.sec.AdminPassword
	username := strings.Split(email, "@")[0]

	// Register admin user via Node.js (API container has node but not curl)
	nodeScript := fmt.Sprintf(`
const http = require('http');
const data = JSON.stringify({email:'%s', password:'%s', name:'Admin', username:'%s'});
const req = http.request({hostname:'127.0.0.1', port:6802, path:'/api/v1/auth/register', method:'POST', headers:{'Content-Type':'application/json','Content-Length':data.length}}, res => {
  let body = '';
  res.on('data', c => body += c);
  res.on('end', () => { console.log(res.statusCode); process.exit(0); });
});
req.on('error', e => { console.log('error:' + e.message); process.exit(1); });
req.write(data);
req.end();
`, email, password, username)

	out, err := DockerExec(ctx, e.output, "api", "node", "-e", nodeScript)

	if err != nil {
		e.send(StepEvent{Step: "Registering admin user", Status: Running, Detail: "Warning: API call failed, skipping"})
	} else {
		httpCode := strings.TrimSpace(out)
		switch {
		case httpCode == "200" || httpCode == "201":
			e.send(StepEvent{Step: "Registering admin user", Status: Running, Detail: "Admin registered"})
		case httpCode == "409" || httpCode == "422" || httpCode == "400":
			// User already exists — skip silently
			e.send(StepEvent{Step: "Registering admin user", Status: Running, Detail: "Admin already exists, skipping"})
			return nil
		default:
			e.send(StepEvent{Step: "Registering admin user", Status: Running, Detail: fmt.Sprintf("Warning: register returned %s, continuing", httpCode)})
		}
	}

	// Auto-verify email via SQL
	verifySQL := fmt.Sprintf(`UPDATE launchpad_main.users SET email_verified = true WHERE email = '%s'`, email)

	if e.cfg.Database.Mode == "builtin" {
		DockerExec(ctx, e.output, "postgres", //nolint:errcheck
			"psql", "-U", "postgres", "-d", "launchpad", "-c", verifySQL)
	} else if len(e.cfg.Database.URLs) > 0 {
		if mainURL, ok := e.cfg.Database.URLs["main"]; ok {
			RunWithTimeout(ctx, "verify-admin-email", 10*time.Second,
				"psql", mainURL, "-c", verifySQL)
		}
	}

	return nil
}
