package migrations

import (
	"testing"
)

func TestFixDbUrl(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"postgres://user:password@localhost:5432/dbname", "pgx5://user:password@localhost:5432/dbname"},
		{"pgx5://user:password@localhost:5432/dbname", "pgx5://user:password@localhost:5432/dbname"},
		{"mysql://user:password@localhost:3306/dbname", "pgx5://user:password@localhost:3306/dbname"},
	}

	for _, test := range tests {
		result := fixDbUrl(test.input)
		if result != test.expected {
			t.Errorf("fixDbUrl(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}
