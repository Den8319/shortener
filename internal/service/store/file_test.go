package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFileStore(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "creates new file if not exists",
			path:    "store.json",
			wantErr: false},
		{name: "error on inaccessible path",
			path:    "/nonexistent/dir/store.json",
			wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if !tt.wantErr {
				path = t.TempDir() + "/" + tt.path
			}
			_, err := NewFileStore(path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
