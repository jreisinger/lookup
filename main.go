package main

import (
	"flag"
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

var n = flag.Int("n", 0, "lookup only against first n nameservers")

func main() {
	flag.Parse()

	log.SetPrefix(os.Args[0] + ": ")
	log.SetFlags(0) // no timestamp

	if len(flag.Args()) < 1 {
		log.Fatalln("missing FQDN to lookup")
	}

	var servers Nameservers

	servers.add("1.1.1.1", "1.0.0.1") // Cloudflare
	servers.add("8.8.8.8", "8.8.4.4") // Google
	if err := servers.getLocal(); err != nil {
		log.Printf("getting local nameservers: %v\n", err)
	}
	if err := servers.getPublic("https://public-dns.info/nameservers.txt"); err != nil {
		log.Printf("getting public nameservers: %v\n", err)
	}
	servers.dedup()
	if *n != 0 {
		servers = servers[:*n]
	}

	var wg sync.WaitGroup
	serversCh := make(chan string)

	var stats Stats
	fqdn := flag.Arg(0)

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

	fmt.Fprintf(os.Stderr, "%d failed nameservers out of %d (%.2f%%)\n",
		stats.failedServers, stats.totalServers(), stats.failedServersPercentage())

	fmt.Printf("%d ok responses out of %d (%.2f%%)\n",
		stats.okResponses, stats.totalResponses(), 100-stats.failedResponsesPercentage())

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

	myMsg := fmt.Sprintf("lookup at %-15s ", server)

	r, _, err := c.Exchange(m, net.JoinHostPort(server, "53"))
	if r == nil { // server issues
		stats.Lock()
		stats.failedServers++
		stats.Unlock()
		log.Println(myMsg + err.Error())
		return
	}

	stats.Lock()
	stats.okServers++
	stats.Unlock()

	if len(r.Answer) < 1 {
		stats.Lock()
		stats.failedResponses++
		stats.Unlock()
		fmt.Println(myMsg + "FAIL")
		return
	}

	stats.Lock()
	stats.okResponses++
	stats.Unlock()
	fmt.Println(myMsg + "OK")
}
