package main

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	tailscaleCIDR         = "100.64.0.0/10"
	tailscaleRouteTable   = "52"
	tailscaleRulePriority = "60"
)

// EnsureTailscaleRoute adds the policy routing rule that sends tailnet traffic
// through tailscale's own routing table:
//
//	sudo ip rule add to 100.64.0.0/10 lookup 52 priority 60
//
// Without it the sync requests leave through the default route and never reach
// the partners. A missing rule is not fatal — the daemon says so and carries on,
// because a machine may already have the routing set up another way.
func EnsureTailscaleRoute() {
	if hasTailscaleRoute() {
		fmt.Println("ip rule for " + tailscaleCIDR + " is already in place")
		return
	}

	output, err := exec.Command("sudo", "ip", "rule", "add",
		"to", tailscaleCIDR,
		"lookup", tailscaleRouteTable,
		"priority", tailscaleRulePriority,
	).CombinedOutput()
	if err != nil {
		fmt.Println("could not add the ip rule for " + tailscaleCIDR +
			", sync may not reach the partners")
		fmt.Println(err)
		if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
			fmt.Println(trimmed)
		}
		return
	}

	fmt.Println("added ip rule: to " + tailscaleCIDR +
		" lookup " + tailscaleRouteTable +
		" priority " + tailscaleRulePriority)
}

// hasTailscaleRoute keeps the daemon from stacking another identical rule on
// every restart — `ip rule add` happily adds duplicates.
func hasTailscaleRoute() bool {
	output, err := exec.Command("ip", "rule", "show").CombinedOutput()
	if err != nil {
		fmt.Println("could not read the existing ip rules")
		fmt.Println(err)
		return false
	}
	return routeIsListed(string(output))
}

// routeIsListed looks for a rule sending the tailnet to our routing table in
// the output of `ip rule show`, whose lines read like
//
//	60:	from all to 100.64.0.0/10 lookup 52
//
// Matching whole words matters: plenty of unrelated rules mention the same
// table, and "lookup 52" is a prefix of "lookup 520".
func routeIsListed(rules string) bool {
	for line := range strings.SplitSeq(rules, "\n") {
		fields := strings.Fields(line)
		routesTailnet := false
		for index, field := range fields {
			if field == "to" && index+1 < len(fields) && fields[index+1] == tailscaleCIDR {
				routesTailnet = true
			}
			if field == "lookup" && index+1 < len(fields) &&
				fields[index+1] == tailscaleRouteTable && routesTailnet {
				return true
			}
		}
	}
	return false
}
