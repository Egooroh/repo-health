package domain

import "testing"

func TestParseRef(t *testing.T) {
	for input, want := range map[string]string{" Org/Repo ": "org/repo", "https://sourcecraft.dev/org/repo": "org/repo", "https://git.sourcecraft.dev/org/repo.git": "org/repo"} {
		got, err := ParseRef(input)
		if err != nil || got != want {
			t.Errorf("%q: %q %v", input, got, err)
		}
	}
	for _, input := range []string{"", "https://evil.test/a/b", "http://sourcecraft.dev/a/b", "https://sourcecraft.dev:443/a/b", "https://user:secret@sourcecraft.dev/a/b", "https://sourcecraft.dev/a/b?token=x", "https://sourcecraft.dev/a/b/issues", "a/../b", "a/%2e%2e", "file:///etc/passwd", "a/b#x", "https://sourcecraft.dev/a%2Fb/c"} {
		if _, err := ParseRef(input); err == nil {
			t.Errorf("accepted unsafe ref %q", input)
		}
	}
	if RepositoryID("a/b", true) == RepositoryID("a/b", false) {
		t.Fatal("demo and real ids overlap")
	}
}
