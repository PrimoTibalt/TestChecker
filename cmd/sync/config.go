package main

import (
	"errors"
	"os"
	"strings"
)

const syncPort = "8081"

// LocalAddr is the address this instance's sync server listens on.
func LocalAddr() (string, error) {
	ip := os.Getenv("TAILSCALE_IP")
	if ip == "" {
		return "", errors.New("no ip in TAILSCALE_IP was provided")
	}
	return ip + ":" + syncPort, nil
}

// PartnerURLs are the base urls of every partner instance. TAILSCALE_PARTNER_IP
// holds one ip per partner, separated by commas, so the same daemon can keep
// any number of machines in sync rather than just a pair.
func PartnerURLs() ([]string, error) {
	raw := os.Getenv("TAILSCALE_PARTNER_IP")
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("no ip in TAILSCALE_PARTNER_IP was provided")
	}

	urls := []string{}
	seen := map[string]bool{}
	for _, ip := range strings.Split(raw, ",") {
		ip = strings.TrimSpace(ip)
		if ip == "" || seen[ip] {
			continue
		}
		seen[ip] = true
		urls = append(urls, "http://"+ip+":"+syncPort)
	}

	if len(urls) == 0 {
		return nil, errors.New("no usable ip in TAILSCALE_PARTNER_IP")
	}
	return urls, nil
}
