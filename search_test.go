package proximity

import (
	"os"
	"testing"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"github.com/blugelabs/bluge/search/similarity"
	"github.com/voldyman/proximity-query/util"
)

type failureReporter interface {
	Error(args ...interface{})
	Fatal(args ...interface{})
}
type searchResult struct {
	id   string
	text string
	m    *search.DocumentMatch
}

const indexPath = "temp.index"

func TestProximityOrdering(t *testing.T) {
	r, err := util.CreateIndex(indexPath)
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	defer os.RemoveAll(indexPath)

	// Search using proximity scorer, words closer together should be ranked higher
	query := createSearchQuery().SetScorer(NewProximityCompositeScorer())
	id0Doc, id1Doc := searchDocMatches(t, r, query)
	if id1Doc.Score <= id0Doc.Score {
		// doc 0: "_quick_ brown scared _fox_"
		// doc 1: "_quick black _fox_"
		// doc 1 should score higher because words appear closer together
		t.Fatalf("document id 1's score was lower than document id 0\nid 0: %f\nid 1: %f", id0Doc.Score, id1Doc.Score)
	}

	// Search using default scorer, doesn't care about proximity, results where matches are distant rank higher
	query = createSearchQuery().SetScorer(similarity.NewCompositeSumScorer())
	id0Doc, id1Doc = searchDocMatches(t, r, query)
	if id0Doc.Score != id1Doc.Score {
		t.Fatalf("score(doc0) != score(doc1) \nid 0: %f\nid 1: %f", id0Doc.Score, id1Doc.Score)
	}
}

func searchDocMatches(t failureReporter, r *bluge.Reader, q bluge.Query) (search.DocumentMatch, search.DocumentMatch) {
	res, err := util.ExecuteSearch(r, q)
	if err != nil {
		t.Error(err)
	}

	results := map[string]searchResult{}
	err = util.VisitResults(res, func(id, text string, match *search.DocumentMatch) {
		results[id] = searchResult{id, text, match}
	})

	if err != nil {
		t.Error(err)
	}

	id0Doc, ok := results["0"]
	if !ok {
		t.Fatal("document with id 0 not found in results", results)
	}
	id1Doc, ok := results["1"]
	if !ok {
		t.Fatal("document with id 1 not found in results", results)
	}
	return *id0Doc.m, *id1Doc.m
}
func createSearchQuery() *ProximityQuery {
	query := bluge.NewMatchQuery("quick fox").SetField("text")

	return NewProximityQuery().AddSubquery(query)
}

func BenchmarkDefaultCompositeScorer(b *testing.B) {
	r, err := util.CreateIndex(indexPath)
	if err != nil {
		b.Error(err)
	}
	defer r.Close()
	defer os.RemoveAll(indexPath)

	query := createSearchQuery().SetScorer(similarity.NewCompositeSumScorer())
	for i := 0; i < b.N; i++ {
		searchDocMatches(b, r, query)
	}
}

func BenchmarkProximityCompositeScorer(b *testing.B) {
	r, err := util.CreateIndex(indexPath)
	if err != nil {
		b.Error(err)
	}
	defer r.Close()
	defer os.RemoveAll(indexPath)

	query := createSearchQuery().SetScorer(NewProximityCompositeScorer())
	for i := 0; i < b.N; i++ {
		searchDocMatches(b, r, query)
	}
}
