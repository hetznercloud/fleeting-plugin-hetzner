package hetzner

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLaxStringListUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    LaxStringList
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "success string",
			data:    `"foo"`,
			want:    LaxStringList{"foo"},
			wantErr: assert.NoError,
		},
		{
			name:    "failure number",
			data:    `1`,
			want:    LaxStringList{},
			wantErr: assert.Error,
		},
		{
			name:    "failure null",
			data:    `null`,
			want:    LaxStringList{},
			wantErr: assert.Error,
		},
		{
			name:    "success empty list",
			data:    `[]`,
			want:    LaxStringList{},
			wantErr: assert.NoError,
		},
		{
			name:    "success string list",
			data:    `["foo", "bar"]`,
			want:    LaxStringList{"foo", "bar"},
			wantErr: assert.NoError,
		},
		{
			name:    "failure non-string list",
			data:    `[7, false]`,
			want:    LaxStringList{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &LaxStringList{}

			tt.wantErr(t, result.UnmarshalJSON([]byte(tt.data)), fmt.Sprintf("UnmarshalJSON(%v)", tt.data))
			assert.Equal(t, tt.want, *result)
		})
	}
}
