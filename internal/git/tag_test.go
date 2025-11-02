package git

import (
	"fmt"
	"testing"

	"github.com/alexjoedt/forge/internal/version"
)

func TestTagger(t *testing.T) {

	tgg := NewTagger(".", "teatapp/v", true)
	s, err := tgg.LatestTag(t.Context())
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	fmt.Println(s)

	v, err := tgg.ParseLatestVersion(t.Context(), version.SchemeSemVer)
	must(t, err)

	fmt.Printf("%v\n", v)

	got, err := tgg.IsTagOnCurrentCommit(t.Context(), s)
	must(t, err)
	fmt.Println(got)
}

func must(t *testing.T, err error) {
	if err != nil {
		t.FailNow()
	}
}
