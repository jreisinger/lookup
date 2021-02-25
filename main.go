package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

func main() {
	progName := os.Args[0]

	if len(os.Args[1:]) != 1 {
		help := fmt.Sprintf(`Lookup FQDN at many public DNS servers.

Usage:
  %s FQDN`, progName)
		fmt.Println(help)
		os.Exit(0)
	}

	log.SetPrefix(progName + ": ")
	log.SetFlags(0) // no timestamp

	fqdn := os.Args[1]
	servers := getNameservers()
	var stats Stats

	// Run lookups concurrently.
	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		go func(s string) {
			lookupAt(s, fqdn, &stats)
			wg.Done()
		}(server)
	}
	wg.Wait()

	printSummaryAndExit(&stats)
}

// Stats holds statistics about DNS responses.
type Stats struct {
	sync.Mutex
	okResponses         int
	failedResponsesFrom []string
}

func getNameservers() []string {
	// Let's hardcode couple of reliable DNS servers.
	servers := []string{"1.1.1.1", "8.8.8.8", "8.8.4.4"}

	// Add more DNS servers.
	publicServers, err := fetchPublicNameservers("https://public-dns.info/nameservers.txt")
	if err != nil {
		log.Printf("getting public nameservers: %v\n", err)
	}
	servers = append(servers, publicServers...)

	// Add DNS servers from local config file if any.
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		log.Printf("getting local nameservers: %v\n", err)
	}
	servers = append(servers, config.Servers...)

	return dedup(servers)
}

func dedup(in []string) []string {
	out := []string{}
	seen := make(map[string]struct{})
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		out = append(out, s)
		seen[s] = struct{}{}
	}
	return out
}

func printSummaryAndExit(stats *Stats) {
	failed := len(stats.failedResponsesFrom)
	total := failed + stats.okResponses
	failedPercentage := float64(len(stats.failedResponsesFrom)) / float64(total) * 100
	log.Printf("failed response from %d out of %d servers (%.2f%%) %s\n",
		failed, total, failedPercentage, strings.Join(stats.failedResponsesFrom, ", "))
	if failedPercentage > 10 {
		os.Exit(1)
	}
}

func lookupAt(server, fqdn string, stats *Stats) {
	c := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
	m.RecursionDesired = true

	msg := fmt.Sprintf("lookup at %-15s ", server)

	r, _, _ := c.Exchange(m, net.JoinHostPort(server, "53"))
	if r == nil { // no response from the server
		return
	}

	if r.Rcode != dns.RcodeSuccess {
		// ignore server issues
		if r.Rcode == dns.RcodeRefused || r.Rcode == dns.RcodeServerFailure {
			return
		}
		stats.Lock()
		stats.failedResponsesFrom = append(stats.failedResponsesFrom, server)
		stats.Unlock()
		log.Println(msg + "ERR")
		return
	}

	stats.Lock()
	stats.okResponses++
	stats.Unlock()
	log.Println(msg + "OK")
}

func fetchPublicNameservers(url string) ([]string, error) {
	var servers []string

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return servers, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return servers, err
	}

	// Select only IPv4 nameservers.
	for _, server := range strings.Split(string(b), "\n") {
		ipv4 := net.ParseIP(server).To4()
		if ipv4 != nil {
			servers = append(servers, ipv4.String())
		}
	}

	return servers, nil
}
