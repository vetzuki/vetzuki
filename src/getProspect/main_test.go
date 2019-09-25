package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestRender(t *testing.T) {
	html := getTemplate()

	page := Page{
		Name:     "name",
		Employer: "employerName",
		Role:     "role",
		SSHURL:   "sshURL",
		Links:    []Link{{Name: "link", HREF: "href"}},
	}
	s, err := render(html, &page)
	if err != nil {
		t.Fatalf("expected to render, got %s", err)
	}
	if len(s) == 1 {
		t.Fatalf("expected a string, got nothing")
	}
	ioutil.WriteFile("index.html", []byte(s), os.FileMode(int(0755)))
}
