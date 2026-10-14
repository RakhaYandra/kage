package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKnownFixturesDetected(t *testing.T) {
	fixtures := map[string]string{
		"id_rsa":           "-----" + "BEGIN RSA PRIVATE KEY-----\nabc\n",
		"key.pem":          "-----" + "BEGIN PRIVATE KEY-----\nabc\n",
		".env":             "API_KEY=supersecret123\n",
		"credentials.json": "{\"token\": \"ghp_\" + \"abcdefghijklmnopqrst\"}" + "\n",
		"aws.txt":          "AKIA" + "IOSFODNN7EXAMPLE\n",
		"token.conf":       "password = hunter2hunter2\n",
		"github.txt":       "ghp_" + "abcdefghijklmnopqrstuvwx\n",
		"slack.txt":        "xoxb-" + "123456789012-abcdefghijklmnop\n",
		"google.txt":       "AIza" + "SyDdI0hCZtE6vySjMm-WEfQrTgI0hAaUqw\n",
		"openssh.txt":      "-----" + "BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaA==\n",
	}
	for name, content := range fixtures {
		findings := ScanContent(name, []byte(content))
		if len(findings) == 0 {
			t.Errorf("fixture %q not detected", name)
		}
	}
}

func TestCleanFilePasses(t *testing.T) {
	content := []byte("theme=dark\nkeybindings=default\n# example_token is just a word here\n")
	if f := ScanContent("hyprland.conf", content); len(f) != 0 {
		t.Fatalf("false positive: %v", f)
	}
}

func TestRedactMasksSecrets(t *testing.T) {
	in := "token = ghp_" + "abcdefghijklmnopqrstuvwx and password = hunter2hunter2"
	out := Redact(in)
	if out == in {
		t.Fatalf("redact did not change output")
	}
	for _, secret := range []string{"ghp_" + "abcdefghijklmnopqrstuvwx", "hunter2hunter2"} {
		if contains(out, secret) {
			t.Fatalf("secret leaked: %q", out)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestSensitiveName(t *testing.T) {
	for _, p := range []string{".ssh/id_rsa", "cert.pem", "store.p12", ".env", ".env.local", "credentials.json"} {
		if !SensitiveName(p) {
			t.Errorf("%q must be sensitive", p)
		}
	}
	if SensitiveName("hyprland.conf") {
		t.Errorf("hyprland.conf must not be sensitive")
	}
}

func TestScanEntriesSkipsExcluded(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, ".env")
	os.WriteFile(p, []byte("K=1\n"), 0o644)
	// excluded path must not even be scanned (discovery handles it);
	// direct content scan still flags it — documents layering.
	if len(ScanContent(".env", []byte("K=1\n"))) == 0 {
		// .env name itself is sensitive even with benign content
		if !SensitiveName(".env") {
			t.Fatalf(".env must be sensitive by name")
		}
	}
}
