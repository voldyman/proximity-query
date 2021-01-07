package util

import (
	"context"
	"os"
	"strconv"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"github.com/blugelabs/bluge/search/highlight"
	"github.com/pkg/errors"
)

// CreateIndex creates an index with demo data at given path
func CreateIndex(indexPath string) (*bluge.Reader, error) {
	os.RemoveAll(indexPath)

	cfg := bluge.DefaultConfig(indexPath)
	w, err := bluge.OpenWriter(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open write")
	}

	batch := bluge.NewBatch()

	for _, doc := range createDocs() {
		batch.Update(doc.ID(), doc)
	}

	err = w.Batch(batch)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert documents batch")
	}

	w.Close()
	r, err := bluge.OpenReader(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open reader after building index")
	}
	return r, nil
}

// ExecuteSearch runs the query on the reader
func ExecuteSearch(r *bluge.Reader, query bluge.Query) (search.DocumentMatchIterator, error) {
	req := bluge.NewTopNSearch(10, query).IncludeLocations().ExplainScores()
	res, err := r.Search(context.Background(), req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to execute search")
	}
	return res, nil
}

// VisitResults can be used to extract values from the results
func VisitResults(res search.DocumentMatchIterator, fn func(id, text string, m *search.DocumentMatch)) error {
	highligher := highlight.NewANSIHighlighterColor(highlight.FgCyan)
	match, err := res.Next()
	for match != nil && err == nil {
		id := ""
		text := ""
		match.VisitStoredFields(func(field string, value []byte) bool {
			val := string(value)
			if loc, ok := match.Locations[field]; ok {
				frag := highligher.BestFragment(loc, value)
				if len(frag) > 0 {
					val = frag
				}
			}
			if field == "_id" {
				id = val
			}
			if field == "text" {
				text = val
			}
			return true
		})
		fn(id, text, match)

		match, err = res.Next()
	}
	if err != nil {
		return errors.Wrap(err, "unable to read results")
	}
	return nil
}

func createDocs() []*bluge.Document {
	id := &Seq{}
	docs := []struct {
		id   string
		text string
	}{
		{id: id.Next(), text: "The quick brown scared fox being chased by an animal ran"},
		{id: id.Next(), text: "The quick black fox chased a mouse though the bushes in"},
		{id: id.Next(), text: "rabbit ate the prying fox"},
		{id: id.Next(), text: "all you need is a database and a beer"},
	}

	res := []*bluge.Document{}
	for _, doc := range docs {
		d := bluge.NewDocument(doc.id).
			AddField(bluge.NewTextField("text", doc.text).
				StoreValue().
				HighlightMatches().
				SearchTermPositions())

		res = append(res, d)
	}

	return res
}

type Seq struct {
	cur int
}

func (s *Seq) Next() string {
	c := s.cur
	s.cur++
	return strconv.Itoa(c)
}
