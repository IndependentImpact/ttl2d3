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
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "valid html",
			cfg: config.Config{
				Input:          "file.ttl",
				Output:         config.OutputHTML,
				LinkDistance:   80,
				ChargeStrength: -300,
				CollideRadius:  20,
			},
			wantErr: false,
		},
		{
			name: "valid json",
			cfg: config.Config{
				Input:          "file.ttl",
				Output:         config.OutputJSON,
				LinkDistance:   80,
				ChargeStrength: -300,
				CollideRadius:  20,
			},
			wantErr: false,
		},
		{
			name: "missing input",
			cfg: config.Config{
				Output:         config.OutputHTML,
				LinkDistance:   80,
				ChargeStrength: -300,
				CollideRadius:  20,
			},
			wantErr: true,
		},
		{
			name: "bad output format",
			cfg: config.Config{
				Input:          "file.ttl",
				Output:         config.OutputFormat("svg"),
				LinkDistance:   80,
				ChargeStrength: -300,
				CollideRadius:  20,
			},
			wantErr: true,
		},
		{
			name: "bad input format",
			cfg: config.Config{
				Input:          "file.ttl",
				Output:         config.OutputHTML,
				Format:         config.InputFormat("n3"),
				LinkDistance:   80,
				ChargeStrength: -300,
				CollideRadius:  20,
			},
			wantErr: true,
		},
		{
			name: "zero link distance",
			cfg: config.Config{
				Input:          "file.ttl",
				Output:         config.OutputHTML,
				LinkDistance:   0,
				ChargeStrength: -300,
				CollideRadius:  20,
			},
			wantErr: true,
		},
		{
			name: "zero collide radius",
			cfg: config.Config{
				Input:          "file.ttl",
				Output:         config.OutputHTML,
				LinkDistance:   80,
				ChargeStrength: -300,
				CollideRadius:  0,
			},
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
