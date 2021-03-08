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

var n = flag.Int("n", 0, "look up only against first `n` nameservers")
var t = flag.String("t", "A", "DNS `type` to look up")

func main() {
	flag.Parse()

	log.SetPrefix(os.Args[0] + ": ")
	log.SetFlags(0) // no timestamp

	if len(flag.Args()) < 1 {
		log.Fatalln("missing FQDN to lookup")
	}

	fqdn := flag.Arg(0)
	var stats Stats
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
	stats.totalServers = len(servers)

	var wg sync.WaitGroup
	serversCh := make(chan string)

	t := strings.ToUpper(*t)
	dnsType := dns.StringToType[t]
	if dnsType == 0 {
		log.Fatalf("unsupported DNS type: %s", t)
	}

	// Spin up 100 workers to make lookups.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range serversCh {
				lookup(fqdn, s, &stats, dnsType)
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
	stats.printSummary()
}

// Stats holds statistics about DNS responses.
type Stats struct {
	sync.Mutex
	totalServers    int
	failedServers   int
	totalResponses  int
	failedResponses int
	emptyResponses  int
}

func (s *Stats) printSummary() {
	fmt.Printf("%s\n", strings.Repeat("-", 35))
	format := "%-24s %2.0f%% (%d/%d)\n"
	fmt.Printf(format, "Failed nameservers",
		s.failedServersPercentage(), s.failedServers, s.totalServers)
	fmt.Printf(format, "Failed responses",
		s.failedResponsesPercentage(), s.failedResponses, s.totalResponses)
	fmt.Printf(format, "Empty responses",
		s.emptyResponsesPercentage(), s.emptyResponses, s.totalResponses-s.failedResponses)
}

func (s *Stats) failedServersPercentage() float64 {
	return float64(s.failedServers) / float64(s.totalServers) * 100
}

func (s *Stats) failedResponsesPercentage() float64 {
	return float64(s.failedResponses) / float64(s.totalResponses) * 100
}

func (s *Stats) emptyResponsesPercentage() float64 {
	return float64(s.emptyResponses) / float64(s.totalResponses-s.failedResponses) * 100
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

func lookup(fqdn, server string, stats *Stats, dnsType uint16) {
	c := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(fqdn), dnsType)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, net.JoinHostPort(server, "53"))
	if r == nil { // server issues
		stats.Lock()
		stats.failedServers++
		stats.Unlock()
		fmt.Printf("querying %-15s %v\n", server, err.Error())
		return
	}

	stats.Lock()
	stats.totalResponses++
	stats.Unlock()

	nRRs := len(r.Answer)
	rCode := dns.RcodeToString[r.Rcode]

	if rCode != "NOERROR" {
		stats.Lock()
		stats.failedResponses++
		stats.Unlock()
	}

	format := "querying %-15s %d RR"
	if nRRs == 0 || nRRs > 1 {
		format += "s"
	}
	format += " (%s)\n"
	fmt.Printf(format, server, nRRs, rCode)

	if nRRs < 1 {
		stats.Lock()
		stats.emptyResponses++
		stats.Unlock()
		return
	}
}
