package main

import (
	"fmt"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"github.com/voldyman/proximity-query"
	"github.com/voldyman/proximity-query/util"
)

const indexPath = "test.index"

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	r, err := util.CreateIndex(indexPath)
	if err != nil {
		return err
	}
	defer r.Close()

	res, err := util.ExecuteSearch(r, createSearchQuery())
	if err != nil {
		return err
	}

	return util.VisitResults(res, func(id, text string, match *search.DocumentMatch) {
		fmt.Printf("found: %s -> %s\nScore: %f\nExplanation: %s\n\n", id, text, match.Score, "") //match.Explanation.String())
	})
}

func createSearchQuery() *proximity.ProximityQuery {
	query := bluge.NewMatchQuery("quick fox").SetField("text")

	return proximity.NewProximityQuery().AddSubquery(query)
}
