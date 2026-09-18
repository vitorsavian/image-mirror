package sortstrategy_test

import (
	"testing"

	"github.com/k3s-io/image-mirror/internal/sortstrategy"
	"github.com/stretchr/testify/assert"
)

func TestNatural(t *testing.T) {
	tests := []struct {
		name    string
		t       []string
		want    []string
		wantErr bool
	}{
		{
			name: "Sorting revisions",
			t: []string{
				"0",
				"0.27.0",
				"0.27.0-1.1",
				"0.27.0-1.10",
			},
			want: []string{
				"0",
				"0.27.0",
				"0.27.0-1.1",
				"0.27.0-1.10",
			},
			wantErr: false,
		},
		{
			name: "Sorting non-tag formats",
			t: []string{
				"a1",
				"a10",
				"a11",
				"a2",
				"a3",
				"a9",
			},
			want: []string{
				"a1",
				"a2",
				"a3",
				"a9",
				"a10",
				"a11",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := sortstrategy.Natural(tt.t)
			if tt.wantErr {
				assert.Error(t, gotErr, "error should not be nil")
			} else {
				assert.NoError(t, gotErr, "error should be nil")
			}
			assert.Equal(t, tt.want, got, "The two slices should be the same")
		})
	}
}

func TestSemver(t *testing.T) {
	tests := []struct {
		name    string
		t       []string
		want    []string
		wantErr bool
	}{
		{
			name: "Sorting pre-releases",
			t: []string{
				"0",
				"0.27.0",
				"0.27.0-1.1",
				"0.27.0-1.10",
			},
			want: []string{
				"0.0.0",
				"0.27.0-1.1",
				"0.27.0-1.10",
				"0.27.0",
			},
			wantErr: false,
		},
		{
			name: "Sorting non-tag formats",
			t: []string{
				"a1",
				"a10",
				"a11",
				"a2",
				"a3",
				"a9",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := sortstrategy.Semver(tt.t)
			if tt.wantErr {
				assert.Error(t, gotErr, "error should not be nil")
			} else {
				assert.NoError(t, gotErr, "error should be nil")
			}
			assert.Equal(t, tt.want, got, "The two slices should be the same")
		})
	}
}
