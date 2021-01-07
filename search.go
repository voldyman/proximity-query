package proximity

import (
	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"github.com/blugelabs/bluge/search/searcher"
)

var _ bluge.Query = &ProximityQuery{}

// ProximityQuery can be used to run sub queries where the tearms should be close to each other
type ProximityQuery struct {
	subQs  []bluge.Query
	scorer search.CompositeScorer
}

// NewProximityQuery creates a new query
func NewProximityQuery() *ProximityQuery {
	return &ProximityQuery{
		scorer: NewProximityCompositeScorer(),
	}
}

// AddSubquery adds a sub query, all added subqueries would be run. this should be though of as calling `AddShould` on a `BooleanQuery`
func (q *ProximityQuery) AddSubquery(sub bluge.Query) *ProximityQuery {
	q.subQs = append(q.subQs, sub)
	return q
}

// SetScorer changes the scorer used to compute score of the matches
func (q *ProximityQuery) SetScorer(scorer search.CompositeScorer) *ProximityQuery {
	q.scorer = scorer
	return q
}

// Searcher is used by bluge to search matches
func (q *ProximityQuery) Searcher(i search.Reader, options search.SearcherOptions) (rv search.Searcher, err error) {
	if len(q.subQs) == 0 {
		return searcher.NewMatchNoneSearcher(i, options)
	}
	subQSearchers := []search.Searcher{}
	for _, subQ := range q.subQs {
		s, err := subQ.Searcher(i, options)
		if err != nil {
			closeAll(subQSearchers)
			return nil, err
		}
		subQSearchers = append(subQSearchers, s)
	}
	disjunctionSearcher, err := searcher.NewDisjunctionSearcher(i, subQSearchers, 1, q.scorer, options)
	if err != nil {
		closeAll(subQSearchers)
		return nil, err
	}
	return disjunctionSearcher, nil
}

func closeAll(els []search.Searcher) {
	for _, el := range els {
		el.Close()
	}
}

var _ search.CompositeScorer = NewProximityCompositeScorer()

// ProximityCompositeScorer scores matches with terms in the field closer together high
type ProximityCompositeScorer struct{}

// NewProximityCompositeScorer creates a new scorer
func NewProximityCompositeScorer() *ProximityCompositeScorer {
	return &ProximityCompositeScorer{}
}

func (p *ProximityCompositeScorer) ScoreComposite(constituents []*search.DocumentMatch) float64 {
	var rv float64
	for _, constituent := range constituents {
		rv += constituent.Score + calculateProximityScore(constituent.FieldTermLocations)
	}
	return rv
}

func (p *ProximityCompositeScorer) ExplainComposite(constituents []*search.DocumentMatch) *search.Explanation {
	var sum float64
	var children []*search.Explanation
	for _, constituent := range constituents {
		proximityScore := calculateProximityScore(constituent.FieldTermLocations)
		sum += constituent.Score + proximityScore

		explanation := constituent.Explanation
		if proximityScore != 0 {
			explanation = search.NewExplanation(proximityScore, "additional boost added for proximity", constituent.Explanation)
		}
		children = append(children, explanation)
	}
	return search.NewExplanation(sum,
		"sum of:",
		children...)
}

type termPositions map[string][]int

func calculateProximityScore(locs []search.FieldTermLocation) float64 {
	fieldTermPosMap := map[string]termPositions{}

	for _, l := range locs {
		if termPosMap, ok := fieldTermPosMap[l.Field]; ok {
			if freqs, ok := termPosMap[l.Term]; ok {
				freqs = append(freqs, l.Location.Pos)
			} else {
				termPosMap[l.Term] = []int{l.Location.Pos}
			}
		} else {
			fieldTermPosMap[l.Field] = termPositions{l.Term: []int{l.Location.Pos}}
		}
	}
	// todo: this is broken, needs to be fixed
	sum := float64(0)
	for _, termPos := range fieldTermPosMap {
		for _, locs := range termPos {
			sum += calculateScore(locs)
		}
	}

	return sum
}

func calculateScore(locs []int) float64 {
	minPos := min(locs)
	maxPos := max(locs)

	if minPos == maxPos {
		return 0
	}
	return 1.0 / float64(maxPos-minPos)
}

func min(i []int) int {
	if len(i) == 0 {
		return 0
	}
	m := i[0]
	for n := range i {
		if n < m {
			m = n
		}
	}
	return m
}

func max(i []int) int {
	if len(i) == 0 {
		return 0
	}
	m := i[0]
	for n := range i {
		if n > m {
			m = n
		}
	}
	return m
}
