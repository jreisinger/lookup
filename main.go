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

	log.SetPrefix(progName + ": ")
	log.SetFlags(0) // no timestamp

	if len(os.Args[1:]) != 1 {
		log.Fatalln("missing FQDN to lookup")
	}

	fqdn := os.Args[1]
	var stats Stats
	var servers Nameservers

	if err := servers.getLocal(); err != nil {
		log.Printf("getting local nameservers: %v\n", err)
	}

	if err := servers.getPublic("https://public-dns.info/nameservers.txt"); err != nil {
		log.Printf("getting public nameservers: %v\n", err)
	}

	// Add couple of reliable public nameservers.
	servers.add("1.1.1.1", "8.8.8.8", "8.8.4.4")
	servers.dedup()

	var wg sync.WaitGroup
	serversCh := make(chan string)

	// Spin up 100 workers to make lookups.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range serversCh {
				lookup(fqdn, s, &stats)
			}
		}()
	}

	// Send workers servers to make lookups at.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(serversCh)
		for _, s := range servers {
			serversCh <- s
		}
	}()

	wg.Wait()

	log.Printf("%d failed nameservers out of %d (%.2f%%)\n",
		stats.failedServers, stats.totalServers(), stats.failedServersPercentage())

	fmt.Printf("%d failed responses out of %d nameservers (%.2f%%)\n",
		stats.failedResponses, stats.totalResponses(), stats.failedResponsesPercentage())

	if stats.failedResponsesPercentage() > 10 {
		os.Exit(1)
	}
}

// Stats holds statistics about DNS responses.
type Stats struct {
	sync.Mutex
	okResponses     int
	failedResponses int
	okServers       int
	failedServers   int
}

func (s *Stats) failedResponsesPercentage() float64 {
	return float64(s.failedResponses) / float64(s.totalResponses()) * 100
}

func (s *Stats) totalResponses() int {
	return s.failedResponses + s.okResponses
}

func (s *Stats) failedServersPercentage() float64 {
	return float64(s.failedServers) / float64(s.totalServers()) * 100
}

func (s *Stats) totalServers() int {
	return s.failedServers + s.okServers
}

// Nameservers to make lookups against.
type Nameservers []string

func (n *Nameservers) add(servers ...string) {
	*n = append(*n, servers...)
}

func (n *Nameservers) dedup() {
	orig := *n
	*n = []string{} // empty the slice
	seen := make(map[string]struct{})
	for _, s := range orig {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		*n = append(*n, s)
	}
}

func (n *Nameservers) getLocal() error {
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return err
	}
	*n = append(*n, config.Servers...)
	return nil
}

func (n *Nameservers) getPublic(url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Select only IPv4 nameservers.
	for _, server := range strings.Split(string(b), "\n") {
		ipv4 := net.ParseIP(server).To4()
		if ipv4 != nil {
			*n = append(*n, ipv4.String())
		}
	}

	return nil
}

func lookup(fqdn, server string, stats *Stats) {
	c := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
	m.RecursionDesired = true

	msg := fmt.Sprintf("lookup at %-15s ", server)

	r, _, err := c.Exchange(m, net.JoinHostPort(server, "53"))
	if r == nil { // server issues
		stats.Lock()
		stats.failedServers++
		stats.Unlock()
		log.Println(msg + err.Error())
		return
	}

	stats.Lock()
	stats.okServers++
	stats.Unlock()

	if r.Rcode != dns.RcodeSuccess {
		// server issues
		switch r.Rcode {
		case dns.RcodeRefused:
			stats.Lock()
			stats.failedServers++
			stats.Unlock()
			log.Println(msg + "REFUSED")
			return
		case dns.RcodeServerFailure:
			stats.Lock()
			stats.failedServers++
			stats.Unlock()
			log.Println(msg + "SRVFAIL")
			return
		}

		stats.Lock()
		stats.failedResponses++
		stats.Unlock()
		fmt.Println(msg + "FAIL")
		return
	}

	stats.Lock()
	stats.okResponses++
	stats.Unlock()
	fmt.Println(msg + "OK")
}
