package config

import "reflect"

// Change records a single field-level difference between two configs.
type Change struct {
	Field string
	Old   interface{}
	New   interface{}
}

// ConfigDiff holds all changes between two Config values.
type ConfigDiff struct {
	Changes []Change
}

// Diff compares two Config structs and returns all field-level differences.
// Top-level scalar fields and nested struct fields are compared using reflect.DeepEqual.
func Diff(old, new *Config) *ConfigDiff {
	d := &ConfigDiff{}
	diffStruct(reflect.ValueOf(*old), reflect.ValueOf(*new), "", &d.Changes)
	return d
}

// diffStruct walks two struct values recursively, recording changed fields.
func diffStruct(a, b reflect.Value, prefix string, changes *[]Change) {
	t := a.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fa := a.Field(i)
		fb := b.Field(i)

		name := field.Name
		if prefix != "" {
			name = prefix + "." + name
		}

		if fa.Kind() == reflect.Struct {
			diffStruct(fa, fb, name, changes)
			continue
		}

		if !reflect.DeepEqual(fa.Interface(), fb.Interface()) {
			*changes = append(*changes, Change{
				Field: name,
				Old:   fa.Interface(),
				New:   fb.Interface(),
			})
		}
	}
}

// AffectedServices returns a deduplicated list of service names that need to
// be restarted based on which config sections changed.
//
// Mapping rules:
//   - SSL.*        → nginx, api, ui, router, gateway
//   - SMTP.*       → api
//   - SSO.*        → api
//   - Storage.*    → api
//   - Telegram.*   → api
//   - AI.*         → api
//   - Stripe.*     → api
//   - Performance.*→ api
//   - Database.*   → all services (nginx, api, ui, router, gateway, gitea, postgresql, redis)
//   - Images.API   → api
//   - Images.UI    → ui
//   - Images.Router→ router
//   - Images.Gateway→ gateway
//   - Images.Gitea → gitea
//   - Images.*     → (other image fields: registry/version) → all services
func (d *ConfigDiff) AffectedServices() []string {
	all := []string{"nginx", "api", "ui", "router", "gateway", "gitea", "postgresql", "redis"}
	seen := map[string]bool{}

	add := func(services ...string) {
		for _, s := range services {
			if !seen[s] {
				seen[s] = true
			}
		}
	}

	for _, c := range d.Changes {
		switch prefixOf(c.Field) {
		case "SSL":
			add("nginx", "api", "ui", "router", "gateway")
		case "SMTP":
			add("api")
		case "SSO":
			add("api")
		case "Storage":
			add("api")
		case "Telegram":
			add("api")
		case "AI":
			add("api")
		case "Stripe":
			add("api")
		case "Performance":
			add("api")
		case "Database":
			add(all...)
		case "Images":
			switch c.Field {
			case "Images.API":
				add("api")
			case "Images.APIVersion":
				add("api")
			case "Images.UI":
				add("ui")
			case "Images.UIVersion":
				add("ui")
			case "Images.Router":
				add("router")
			case "Images.RouterVersion":
				add("router")
			case "Images.Gateway":
				add("gateway")
			case "Images.GatewayVersion":
				add("gateway")
			case "Images.Gitea":
				add("gitea")
			case "Images.GiteaVersion":
				add("gitea")
			default:
				// Registry change affects all image-based services.
				add("nginx", "api", "ui", "router", "gateway", "gitea")
			}
		}
	}

	result := make([]string, 0, len(seen))
	// Preserve a stable order using the canonical all-services list.
	for _, s := range append(all, "nginx", "api", "ui", "router", "gateway") {
		if seen[s] {
			result = append(result, s)
			delete(seen, s)
		}
	}
	return result
}

// prefixOf returns the top-level section name from a dotted field path,
// e.g. "SSL.Mode" → "SSL".
func prefixOf(field string) string {
	for i := 0; i < len(field); i++ {
		if field[i] == '.' {
			return field[:i]
		}
	}
	return field
}
