package services

import (
	"fmt"
	"net/http"
	"strings"
)

// Redirect sends an old service path on to the key it was renamed to:
// /archive/x?y -> /fleet-data/x?y. Links already handed out -- in tickets,
// chat cards, bookmarks -- keep working after a service changes its key.
type Redirect struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ParseRedirects reads PORTAL_REDIRECTS: comma-separated old=new pairs of
// path keys, "archive=fleet-data". The target must be a key of svcs; the old
// key must be none of them (it would shadow the service) and not one of the
// portal's own paths. Malformed input is an error, like in Parse.
func ParseRedirects(raw string, svcs []Service) ([]Redirect, error) {
	keys := map[string]bool{}
	for _, s := range svcs {
		keys[s.Key] = true
	}
	var out []Redirect
	seen := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		from, to, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("PORTAL_REDIRECTS entry %q: want old=new", entry)
		}
		from, to = strings.TrimSpace(from), strings.TrimSpace(to)
		if !keyPattern.MatchString(from) {
			return nil, fmt.Errorf("PORTAL_REDIRECTS key %q: lowercase letters, digits and - only", from)
		}
		if reserved[from] {
			return nil, fmt.Errorf("PORTAL_REDIRECTS key %q: reserved for the portal itself", from)
		}
		if keys[from] {
			return nil, fmt.Errorf("PORTAL_REDIRECTS key %q: is a service in PORTAL_SERVICES", from)
		}
		if !keys[to] {
			return nil, fmt.Errorf("PORTAL_REDIRECTS %s: target %q is not a service in PORTAL_SERVICES", from, to)
		}
		if seen[from] {
			return nil, fmt.Errorf("PORTAL_REDIRECTS key %q: listed twice", from)
		}
		seen[from] = true
		out = append(out, Redirect{From: from, To: to})
	}
	return out, nil
}

// Handler answers /<from>/... with a permanent redirect to /<to>/..., the
// rest of the path and the query kept.
func (r Redirect) Handler() http.Handler {
	prefix := "/" + r.From
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		target := "/" + r.To + strings.TrimPrefix(req.URL.Path, prefix)
		if req.URL.RawQuery != "" {
			target += "?" + req.URL.RawQuery
		}
		http.Redirect(w, req, target, http.StatusMovedPermanently)
	})
}
