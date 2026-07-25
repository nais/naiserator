package frontend

import (
	"encoding/json"
	"path"
	"strings"
	"testing"
)

// Edge cases from the adversarial review of nais/naiserator#687.

func TestVersionFromImage(t *testing.T) {
	tests := []struct {
		image string
		want  string
	}{
		{"navikt/myapplication:1.2.3", "1.2.3"},
		{"registry.local:5000/navikt/app:1.2.3", "1.2.3"},   // registry port + tag
		{"registry.local:5000/navikt/app", ""},              // registry port, no tag
		{"ghcr.io/navikt/app", ""},                          // no tag
		{"ghcr.io/navikt/app@sha256:deadbeef", ""},          // digest ref: no tag, never the hex
		{"ghcr.io/navikt/app:v1@sha256:deadbeef", "v1"},     // tag + digest
		{"", ""},
	}
	for _, tt := range tests {
		if got := versionFromImage(tt.image); got != tt.want {
			t.Errorf("versionFromImage(%q) = %q, want %q", tt.image, got, tt.want)
		}
	}
}

// The ES module must be backed by the same escaped serialization as the JSON
// file: a hostile image tag (unvalidated spec.image) must never break the
// module or make the two formats disagree.
func TestNaisJsEscapesHostileValues(t *testing.T) {
	cfg := generatedConfig{
		SchemaVersion:         1,
		TelemetryCollectorURL: "http://collector",
		App: generatedConfigApp{
			Name:      "myapp",
			Namespace: "myteam",
			// legal-in-spec.image junk: quotes, statement injection, newline
			Version: "1.2.3'; globalThis.pwned=1; //\n",
		},
		Environment: "prod-gcp",
	}
	jsonContents, err := naisJson(cfg)
	if err != nil {
		t.Fatal(err)
	}
	js := naisJs(jsonContents)

	if !strings.HasPrefix(js, "export default {") || !strings.HasSuffix(js, ";\n") {
		t.Fatalf("nais.js is not a well-formed module wrapper: %q", js)
	}
	// The payload between the wrapper must be exactly the JSON document —
	// values escaped, no raw quote can terminate anything.
	body := strings.TrimSuffix(strings.TrimPrefix(js, "export default "), ";\n")
	var roundTrip generatedConfig
	if err := json.Unmarshal([]byte(body), &roundTrip); err != nil {
		t.Fatalf("nais.js body is not valid JSON (module would be broken): %v", err)
	}
	if roundTrip != cfg {
		t.Fatalf("nais.js and nais.json diverged: %+v != %+v", roundTrip, cfg)
	}
	if strings.Contains(js, "pwned=1; //\n};") {
		t.Fatal("raw injection survived into the module")
	}
}

// The JSON sibling is only mounted for the conventional file-named mountPath,
// on cleaned paths.
func TestJsonSiblingMountNarrowing(t *testing.T) {
	tests := []struct {
		mountPath string
		wantJSON  string // "" = no sibling mount
	}{
		{"/usr/share/nginx/html/nais.js", "/usr/share/nginx/html/nais.json"},
		{"/path/to//nais.js", "/path/to/nais.json"},  // uncleaned dup slips nothing past
		{"/path/to/nais.json", ""},                   // would shadow itself
		{"/webroot/", ""},                            // directory-ish: no surprise sibling
		{"/webroot", ""},                             // no filename convention
		{"/app/config.js", ""},                       // unconventional filename
		{"/nais.js", "/nais.json"},                   // root-dir file is fine
	}
	for _, tt := range tests {
		got := ""
		if cleaned := path.Clean(tt.mountPath); path.Base(cleaned) == configFileName {
			got = path.Join(path.Dir(cleaned), jsonFileName)
		}
		if got != tt.wantJSON {
			t.Errorf("sibling for mountPath %q = %q, want %q", tt.mountPath, got, tt.wantJSON)
		}
	}
}
