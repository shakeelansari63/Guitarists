//go:build !windows

package platform

import (
	"path/filepath"
	"testing"
)

func TestParseUserDirs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contents string
		want     string
		wantErr  bool
	}{
		{
			name:     "plain quoted",
			contents: "XDG_DOCUMENTS_DIR=\"$HOME/Documents\"\n",
			want:     filepath.Join("/home/player", "Documents"),
		},
		{
			name:     "braced home",
			contents: "XDG_DOCUMENTS_DIR=\"${HOME}/My Docs\"\n",
			want:     filepath.Join("/home/player", "My Docs"),
		},
		{
			name:     "comments and unrelated keys",
			contents: "# comment\nXDG_DESKTOP_DIR=\"$HOME/Desktop\"\nXDG_DOCUMENTS_DIR=\"$HOME/Docs\"\n",
			want:     filepath.Join("/home/player", "Docs"),
		},
		{
			name:     "absolute path",
			contents: "XDG_DOCUMENTS_DIR=\"/data/documents\"\n",
			want:     "/data/documents",
		},
		{
			name:     "missing key",
			contents: "XDG_DESKTOP_DIR=\"$HOME/Desktop\"\n",
			wantErr:  true,
		},
		{
			name:     "empty value",
			contents: "XDG_DOCUMENTS_DIR=\"\"\n",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseUserDirs(tt.contents, "XDG_DOCUMENTS_DIR", "/home/player")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
