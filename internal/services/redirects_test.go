package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseRedirects(t *testing.T) {
	svcs, err := Parse("fleet-data=http://a,fleet-analysis=http://a")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRedirects(" archive = fleet-data , analysis=fleet-analysis ,", svcs)
	if err != nil {
		t.Fatal(err)
	}
	want := []Redirect{{From: "archive", To: "fleet-data"}, {From: "analysis", To: "fleet-analysis"}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if got, err := ParseRedirects("", svcs); err != nil || len(got) != 0 {
		t.Fatalf("empty: got %+v, %v", got, err)
	}
}

func TestParseRedirectsRejects(t *testing.T) {
	svcs, err := Parse("fleet-data=http://a,led=http://b")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"archive",            // no target
		"Archive=fleet-data", // key pattern
		"ui=fleet-data",      // reserved
		"led=fleet-data",     // shadows a service
		"archive=nope",       // target is no service
		"a=fleet-data,a=led", // twice
	} {
		if _, err := ParseRedirects(raw, svcs); err == nil {
			t.Errorf("%q: want an error", raw)
		}
	}
}

func TestRedirectHandler(t *testing.T) {
	h := Redirect{From: "archive", To: "fleet-data"}.Handler()
	for in, want := range map[string]string{
		"/archive/":                  "/fleet-data/",
		"/archive/?tab=analyses":     "/fleet-data/?tab=analyses",
		"/archive/ui/analysis?key=x": "/fleet-data/ui/analysis?key=x",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, in, http.NoBody))
		if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != want {
			t.Errorf("%s: got %d %q, want 301 %q", in, rec.Code, rec.Header().Get("Location"), want)
		}
	}
}
