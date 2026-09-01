package main

import "testing"

// Shaped like `ip rule show` output on a machine that already has the rule,
// including an unrelated rule pointing at the same table further down.
const rulesWithTailnet = `0:	from all lookup local
50:	from all lookup main suppress_prefixlength 1
60:	from all to 100.64.0.0/10 lookup 52
5270:	from all lookup 52
32766:	from all lookup main
32767:	from all lookup default`

func TestRouteIsListed(t *testing.T) {
	if !routeIsListed(rulesWithTailnet) {
		t.Error("the tailnet rule was not recognised")
	}

	cases := map[string]string{
		"no rule at all": `0:	from all lookup local
32766:	from all lookup main`,
		// The table is used, but not for the tailnet.
		"table without the tailnet": `0:	from all lookup local
5270:	from all lookup 52`,
		// The tailnet is routed, but somewhere else.
		"tailnet to another table": `60:	from all to 100.64.0.0/10 lookup 99`,
		// "52" must not match "520" by prefix.
		"similar table number": `60:	from all to 100.64.0.0/10 lookup 520`,
	}
	for name, rules := range cases {
		if routeIsListed(rules) {
			t.Errorf("%s: reported the rule as present", name)
		}
	}
}
