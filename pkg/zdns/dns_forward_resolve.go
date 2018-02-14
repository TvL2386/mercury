package zdns

import (
	"fmt"
	"math/rand"
	"time"

	dnssrv "github.com/miekg/dns"
)

// Resolve resolves a request at a remote host
func Resolve(ns []string, dnsHost string, dnsDomain string, dnsQuery uint16) ([]Record, error) {
	if len(ns) == 0 {
		return []Record{}, fmt.Errorf("No NS found to query")
	}
	var question string
	if dnsHost == "" {
		question = dnsDomain
	} else {
		question = fmt.Sprintf("%s.%s", dnsHost, dnsDomain)
	}

	// limit number of dns servers we do the request to maxNameservers
	// and randomize them

	if len(ns) > maxNameservers {
		dest := make([]string, len(ns))
		perm := rand.Perm(len(ns))
		for i, v := range perm {
			dest[v] = ns[i]
		}
		ns = dest[:maxNameservers]
	}

	/* parallel lookups on all servers */
	resultChan := make(chan *dnssrv.Msg)
	for _, nsSrv := range ns {
		go ResolveSingle(fmt.Sprintf("%s:%d", nsSrv, 53), question, dnsQuery, resultChan)
	}

	var zone *dnssrv.Msg
	/* wait for first answer or timeout */
	timeout := time.NewTimer(5 * time.Second)
gotresult:
	for {
		select {
		case zone = <-resultChan:
			break gotresult
		case <-timeout.C:
			return Records{}, fmt.Errorf("lookup resulted in timeout")
		}
	}

	records := forwardCache.importZone(zone.String())
	return records, nil
}

// ResolveSingle resolves a single request at a single DNS server, and returns its result to chan
func ResolveSingle(ns string, question string, dnsQuery uint16, channel chan *dnssrv.Msg) {
	c := new(dnssrv.Client)
	m := new(dnssrv.Msg)
	m.SetEdns0(4096, true)
	m.SetQuestion(question, dnsQuery)
	zone, _, err := c.Exchange(m, ns)
	if err == nil {
		select {
		case channel <- zone:
		default:
		}
	}
}
