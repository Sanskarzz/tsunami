// Package scoring implements only the CVSS 3.1 base equations. The reviewer
// chooses metrics; the formula cannot establish their real-world applicability.
package scoring

import (
	"errors"
	"math"
	"regexp"
	"strings"
)

var vectorPattern = regexp.MustCompile(`^CVSS:3\.1/AV:[NALP]/AC:[LH]/PR:[NLH]/UI:[NR]/S:[UC]/C:[NLH]/I:[NLH]/A:[NLH]$`)

func Base(vector string) (float64, string, error) {
	if !vectorPattern.MatchString(vector) {
		return 0, "", errors.New("canonical CVSS 3.1 base vector required")
	}
	m := map[string]string{}
	for _, part := range strings.Split(vector, "/")[1:] {
		kv := strings.SplitN(part, ":", 2)
		m[kv[0]] = kv[1]
	}
	av := map[string]float64{"N": .85, "A": .62, "L": .55, "P": .2}[m["AV"]]
	ac := map[string]float64{"L": .77, "H": .44}[m["AC"]]
	pr := map[string]float64{"N": .85, "L": .62, "H": .27}[m["PR"]]
	changed := m["S"] == "C"
	if changed {
		pr = map[string]float64{"N": .85, "L": .68, "H": .5}[m["PR"]]
	}
	ui := map[string]float64{"N": .85, "R": .62}[m["UI"]]
	weight := map[string]float64{"N": 0, "L": .22, "H": .56}
	iss := 1 - (1-weight[m["C"]])*(1-weight[m["I"]])*(1-weight[m["A"]])
	impact := 6.42 * iss
	if changed {
		impact = 7.52*(iss-.029) - 3.25*math.Pow(iss-.02, 15)
	}
	if impact <= 0 {
		return 0, "Informational", nil
	}
	x := impact + 8.22*av*ac*pr*ui
	if changed {
		x *= 1.08
	}
	x = math.Min(x, 10)
	// FIRST Appendix A: convert to five decimal places before rounding up.
	n := int64(math.Round(x * 100000))
	score := float64(n) / 100000
	if n%10000 != 0 {
		score = float64(n/10000+1) / 10
	}
	severity := "Low"
	if score >= 9 {
		severity = "Critical"
	} else if score >= 7 {
		severity = "High"
	} else if score >= 4 {
		severity = "Medium"
	}
	return score, severity, nil
}
