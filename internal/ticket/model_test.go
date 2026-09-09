package ticket

import "testing"

func TestStatus_Valid(t *testing.T) {
	cases := []struct {
		status Status
		want   bool
	}{
		{StatusOpen, true},
		{StatusInProgress, true},
		{StatusClosed, true},
		{Status("bogus"), false},
		{Status(""), false},
	}

	for _, tc := range cases {
		if got := tc.status.Valid(); got != tc.want {
			t.Errorf("Status(%q).Valid() = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestCanTransition(t *testing.T) {
	cases := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{"open to in_progress", StatusOpen, StatusInProgress, true},
		{"in_progress to closed", StatusInProgress, StatusClosed, true},
		{"open to closed directly", StatusOpen, StatusClosed, false},
		{"in_progress back to open", StatusInProgress, StatusOpen, false},
		{"closed to open", StatusClosed, StatusOpen, false},
		{"closed to in_progress", StatusClosed, StatusInProgress, false},
		{"closed to closed", StatusClosed, StatusClosed, false},
		{"open to open", StatusOpen, StatusOpen, false},
		{"unknown from status", Status("bogus"), StatusOpen, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransition(tc.from, tc.to); got != tc.want {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}
