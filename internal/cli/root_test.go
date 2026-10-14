package cli

import "testing"

func TestRootContainsCommands(t *testing.T) {
	root := New()
	want := map[string]bool{
		"init": false, "snapshot": false, "diff": false, "restore": false,
		"audit": false, "status": false, "log": false, "track": false, "config": false, "version": false,
	}
	for _, c := range root.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("command %q missing", name)
		}
	}
}

func TestExitErrorCarriesCode(t *testing.T) {
	err := fail(3, "conflict: %s", "x")
	ee, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("fail must return *ExitError")
	}
	if ee.Code != 3 || ee.Msg == "" {
		t.Fatalf("bad ExitError: %+v", ee)
	}
}
