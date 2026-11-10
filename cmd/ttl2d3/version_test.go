package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name                string
		ver, com, date      string
		info                *debug.BuildInfo
		wantV, wantC, wantD string
	}{
		{
			name: "ldflags from release build win",
			ver:  "0.1.1", com: "abc123", date: "2026-11-10T00:00:00Z",
			info:  &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}},
			wantV: "0.1.1", wantC: "abc123", wantD: "2026-11-10T00:00:00Z",
		},
		{
			name: "go install of a tagged module",
			ver:  "dev", com: "none", date: "unknown",
			info:  &debug.BuildInfo{Main: debug.Module{Version: "v0.1.1"}},
			wantV: "v0.1.1", wantC: "none", wantD: "unknown",
		},
		{
			name: "local build with VCS stamping",
			ver:  "dev", com: "none", date: "unknown",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1612d6d"},
					{Key: "vcs.time", Value: "2026-11-10T14:50:00Z"},
				},
			},
			wantV: "dev", wantC: "1612d6d", wantD: "2026-11-10T14:50:00Z",
		},
		{
			name: "no build info",
			ver:  "dev", com: "none", date: "unknown",
			info:  nil,
			wantV: "dev", wantC: "none", wantD: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, c, d := resolveVersion(tt.ver, tt.com, tt.date, tt.info)
			if v != tt.wantV || c != tt.wantC || d != tt.wantD {
				t.Errorf("resolveVersion() = (%q, %q, %q), want (%q, %q, %q)", v, c, d, tt.wantV, tt.wantC, tt.wantD)
			}
		})
	}
}
