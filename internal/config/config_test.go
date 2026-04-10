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
	if c.WorkflowPlan != false {
		t.Error("default WorkflowPlan should be false")
	}
	if c.NodeSpacing != 180 {
		t.Errorf("default NodeSpacing = %g, want 180", c.NodeSpacing)
	}
	if c.VerticalNodeSpacing != 20 {
		t.Errorf("default VerticalNodeSpacing = %g, want 20", c.VerticalNodeSpacing)
	}
}

func TestValidate(t *testing.T) {
	// validBase returns a minimally valid Config; tests override individual fields.
	validBase := func() config.Config {
		return config.Config{
			Input:               "file.ttl",
			Output:              config.OutputHTML,
			LinkDistance:        80,
			ChargeStrength:      -300,
			CollideRadius:       20,
			NodeSpacing:         180,
			VerticalNodeSpacing: 20,
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
		// WorkflowPlan-specific tests.
		{
			name: "workflowplan valid html",
			cfg: func() config.Config {
				c := validBase()
				c.WorkflowPlan = true
				return c
			}(),
			wantErr: false,
		},
		{
			name: "workflowplan rejected for json output",
			cfg: func() config.Config {
				c := validBase()
				c.Output = config.OutputJSON
				c.WorkflowPlan = true
				return c
			}(),
			wantErr: true,
		},
		// NodeSpacing-specific tests.
		{
			name: "zero node spacing rejected",
			cfg: func() config.Config {
				c := validBase()
				c.NodeSpacing = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "negative node spacing rejected",
			cfg: func() config.Config {
				c := validBase()
				c.NodeSpacing = -10
				return c
			}(),
			wantErr: true,
		},
		{
			name: "custom node spacing accepted",
			cfg: func() config.Config {
				c := validBase()
				c.NodeSpacing = 300
				return c
			}(),
			wantErr: false,
		},
		// VerticalNodeSpacing-specific tests.
		{
			name: "zero vertical node spacing rejected",
			cfg: func() config.Config {
				c := validBase()
				c.VerticalNodeSpacing = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "negative vertical node spacing rejected",
			cfg: func() config.Config {
				c := validBase()
				c.VerticalNodeSpacing = -5
				return c
			}(),
			wantErr: true,
		},
		{
			name: "custom vertical node spacing accepted",
			cfg: func() config.Config {
				c := validBase()
				c.VerticalNodeSpacing = 60
				return c
			}(),
			wantErr: false,
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
