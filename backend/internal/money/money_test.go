package money

import "testing"

func TestFormatMXN(t *testing.T) {
	cases := map[int64]string{
		0:         "$0 MXN",
		44900:     "$449 MXN",
		89900:     "$899 MXN",
		129900:    "$1,299 MXN",
		100000000: "$1,000,000 MXN",
		-44900:    "-$449 MXN",
	}
	for minor, want := range cases {
		if got := FormatMXN(minor); got != want {
			t.Errorf("FormatMXN(%d) = %q, want %q", minor, got, want)
		}
	}
}
