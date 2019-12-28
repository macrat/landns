package landns

import (
	"testing"
)

func TestDynamicRecord_unmarshalAnnotation(t *testing.T) {
	tests := []struct {
		Text  string
		NilID bool
		ID    int
	}{
		{"ID:1", false, 1},
		{" ID:2 ", false, 2},
		{"\tID:3 \t ", false, 3},
		{" ", true, 0},
		{"", true, 0},
		{"hello id:4 world", false, 4},
	}

	r := new(DynamicRecord)

	for _, tt := range tests {
		if err := r.unmarshalAnnotation([]byte(tt.Text)); err != nil {
			t.Errorf("failed to unmarshal annotation: %s", err)
		}

		if !tt.NilID {
			if r.ID == nil || *r.ID != tt.ID {
				t.Errorf("failed to unmarshal ID: expected %v but got %v", tt.ID, r.ID)
			}
		} else if r.ID != nil {
			t.Errorf("failed to unmarshal ID: expected nil but got %v", r.ID)
		}
	}
}
