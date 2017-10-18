package xconf

import (
	"math/rand"
	"net"
	"sync"
)

// xHostProvider is the xConf HostProvider. It resolving
// hosts from xConf server once during the call to Init.
type xHostProvider struct {
	mu         sync.Mutex
	servers    []string
	curr       int
	last       int
	lookupHost func(string) ([]string, error)
}

// Init is called first, with the servers specified in the connection
// string.
// TODO It uses xConf server to look up addresses for each server, then shuffles them all together.
func (hp *xHostProvider) Init(servers []string) error {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	lookupHost := hp.lookupHost
	if lookupHost == nil {
		lookupHost = net.LookupHost
	}

	found := []string{}
	for _, server := range servers {
		host, port, err := net.SplitHostPort(server)
		if err != nil {
			return err
		}
		addrs, err := lookupHost(host)
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			found = append(found, net.JoinHostPort(addr, port))
		}
	}

	// Randomize the order of the servers to avoid creating hotspots
	stringShuffle(found)

	hp.servers = found
	hp.curr = -1
	hp.last = -1

	return nil
}

// Len returns the number of servers available
func (hp *xHostProvider) Len() int {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	return len(hp.servers)
}

// Next returns the next server to connect to. retryStart will be true
// if we've looped through all known servers without Connected() being
// called.
func (hp *xHostProvider) Next() (server string, retryStart bool) {

	hp.mu.Lock()
	defer hp.mu.Unlock()
	hp.curr = (hp.curr + 1) % len(hp.servers)
	retryStart = hp.curr == hp.last
	if hp.last == -1 {
		hp.last = 0
	}
	return hp.servers[hp.curr], retryStart
}

// Connected notifies the HostProvider of a successful connection.
func (hp *xHostProvider) Connected() {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	hp.last = hp.curr
}

// stringShuffle performs a Fisher-Yates shuffle on a slice of strings
func stringShuffle(s []string) {
	for i := len(s) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}
