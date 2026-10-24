package platform

import "testing"

func TestTabsDirName(t *testing.T) {
	t.Parallel()
	if TabsDirName != "guitarists-tabs" {
		t.Fatalf("unexpected tabs dir name %q", TabsDirName)
	}
}
