package scoring

import "testing"

func TestBaseScoresAndStrictVectors(t *testing.T) {
	for _, tc := range []struct {
		v        string
		s        float64
		severity string
	}{
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", 9.8, "Critical"},
		{"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", 6.5, "Medium"},
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H", 10, "Critical"},
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N", 0, "Informational"},
	} {
		score, severity, err := Base(tc.v)
		if err != nil || score != tc.s || severity != tc.severity {
			t.Fatalf("%s: %v %s %v", tc.v, score, severity, err)
		}
	}
	for _, v := range []string{"CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N/AV:L", "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N\n"} {
		if _, _, err := Base(v); err == nil {
			t.Fatal("ambiguous vector accepted")
		}
	}
}
