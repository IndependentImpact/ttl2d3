package config_test

import (
	"testing"

	"github.com/IndependentImpact/ttl2d3/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	c := config.DefaultConfig()

	if c.Output != config.OutputHTML {
		t.Errorf("default Output = %q, want %q", c.Output, config.OutputHTML)
	}
	if c.LinkDistance != 80 {
		t.Errorf("default LinkDistance = %g, want 80", c.LinkDistance)
	}
	if c.ChargeStrength != -300 {
		t.Errorf("default ChargeStrength = %g, want -300", c.ChargeStrength)
	}
	if c.CollideRadius != 20 {
		t.Errorf("default CollideRadius = %g, want 20", c.CollideRadius)
	}
	if c.Layout != config.LayoutForce {
		t.Errorf("default Layout = %q, want %q", c.Layout, config.LayoutForce)
	}
	if c.LayoutDirection != config.LayoutDirectionLR {
		t.Errorf("default LayoutDirection = %q, want %q", c.LayoutDirection, config.LayoutDirectionLR)
	}
	if c.RankSeparation <= 0 {
		t.Errorf("default RankSeparation = %g, want positive", c.RankSeparation)
	}
	if c.NodeSeparation <= 0 {
		t.Errorf("default NodeSeparation = %g, want positive", c.NodeSeparation)
	}
}

func TestValidate(t *testing.T) {
	// validBase returns a minimally valid Config; tests override individual fields.
	validBase := func() config.Config {
		return config.Config{
			Input:           "file.ttl",
			Output:          config.OutputHTML,
			LinkDistance:    80,
			ChargeStrength:  -300,
			CollideRadius:   20,
			Layout:          config.LayoutForce,
			LayoutDirection: config.LayoutDirectionLR,
			RankSeparation:  180,
			NodeSeparation:  80,
		}
	}

	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name:    "valid html",
			cfg:     validBase(),
			wantErr: false,
		},
		{
			name: "valid json",
			cfg: func() config.Config {
				c := validBase()
				c.Output = config.OutputJSON
				return c
			}(),
			wantErr: false,
		},
		{
			name: "missing input",
			cfg: func() config.Config {
				c := validBase()
				c.Input = ""
				return c
			}(),
			wantErr: true,
		},
		{
			name: "bad output format",
			cfg: func() config.Config {
				c := validBase()
				c.Output = config.OutputFormat("svg")
				return c
			}(),
			wantErr: true,
		},
		{
			name: "bad input format",
			cfg: func() config.Config {
				c := validBase()
				c.Format = config.InputFormat("n3")
				return c
			}(),
			wantErr: true,
		},
		{
			name: "zero link distance",
			cfg: func() config.Config {
				c := validBase()
				c.LinkDistance = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "zero collide radius",
			cfg: func() config.Config {
				c := validBase()
				c.CollideRadius = 0
				return c
			}(),
			wantErr: true,
		},
		// Layout-specific tests.
		{
			name: "valid layered layout",
			cfg: func() config.Config {
				c := validBase()
				c.Layout = config.LayoutLayered
				return c
			}(),
			wantErr: false,
		},
		{
			name: "valid swimlane layout",
			cfg: func() config.Config {
				c := validBase()
				c.Layout = config.LayoutSwimlane
				return c
			}(),
			wantErr: false,
		},
		{
			name: "invalid layout mode",
			cfg: func() config.Config {
				c := validBase()
				c.Layout = config.LayoutMode("bpmn")
				return c
			}(),
			wantErr: true,
		},
		{
			name: "layered layout rejected for json output",
			cfg: func() config.Config {
				c := validBase()
				c.Output = config.OutputJSON
				c.Layout = config.LayoutLayered
				return c
			}(),
			wantErr: true,
		},
		{
			name: "swimlane layout rejected for json output",
			cfg: func() config.Config {
				c := validBase()
				c.Output = config.OutputJSON
				c.Layout = config.LayoutSwimlane
				return c
			}(),
			wantErr: true,
		},
		{
			name: "valid tb direction",
			cfg: func() config.Config {
				c := validBase()
				c.Layout = config.LayoutLayered
				c.LayoutDirection = config.LayoutDirectionTB
				return c
			}(),
			wantErr: false,
		},
		{
			name: "invalid layout direction",
			cfg: func() config.Config {
				c := validBase()
				c.Layout = config.LayoutLayered
				c.LayoutDirection = config.LayoutDirection("rl")
				return c
			}(),
			wantErr: true,
		},
		{
			name: "zero rank separation",
			cfg: func() config.Config {
				c := validBase()
				c.Layout = config.LayoutLayered
				c.RankSeparation = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "zero node separation",
			cfg: func() config.Config {
				c := validBase()
				c.Layout = config.LayoutLayered
				c.NodeSeparation = 0
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
