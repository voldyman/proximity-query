package proximity

import (
	"sort"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"github.com/blugelabs/bluge/search/searcher"
)

var _ bluge.Query = &Query{}

// Query can be used to run sub queries where the tearms should be close to each other
type Query struct {
	subQs  []bluge.Query
	scorer search.CompositeScorer
}

// NewProximityQuery creates a new query
func NewProximityQuery() *Query {
	return &Query{
		scorer: NewCompositeScorer(),
	}
}

// AddSubquery adds a sub query, all added subqueries would be run. this should be though of as calling `AddShould` on a `BooleanQuery`
func (q *Query) AddSubquery(sub bluge.Query) *Query {
	q.subQs = append(q.subQs, sub)
	return q
}

// SetScorer changes the scorer used to compute score of the matches
func (q *Query) SetScorer(scorer search.CompositeScorer) *Query {
	q.scorer = scorer
	return q
}

// Searcher is used by bluge to search matches
func (q *Query) Searcher(i search.Reader, options search.SearcherOptions) (rv search.Searcher, err error) {
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

var _ search.CompositeScorer = NewCompositeScorer()

// CompositeScorer scores matches with terms in the field closer together high
type CompositeScorer struct{}

// NewCompositeScorer creates a new scorer
func NewCompositeScorer() *CompositeScorer {
	return &CompositeScorer{}
}

func (p *CompositeScorer) ScoreComposite(constituents []*search.DocumentMatch) float64 {
	var rv float64
	for _, constituent := range constituents {
		rv += constituent.Score + calculateProximateDensityScore(constituent.FieldTermLocations)
	}
	return rv
}

func (p *CompositeScorer) ExplainComposite(constituents []*search.DocumentMatch) *search.Explanation {
	var sum float64
	var children []*search.Explanation
	for _, constituent := range constituents {
		proximityScore := calculateProximateDensityScore(constituent.FieldTermLocations)
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

var _ sort.Interface = termLocs{}

type termLocs []search.FieldTermLocation

func (t termLocs) Len() int      { return len(t) }
func (t termLocs) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

func (t termLocs) Less(i, j int) bool {
	return t[i].Location.Pos < t[j].Location.Pos
}

func insertSorted(locs termLocs, l search.FieldTermLocation) termLocs {
	// hack: didn't feel like writing sort.Search to find the correct location to insert at
	// change in the future
	res := append(locs, l)
	sort.Sort(res)
	return res
}

func calculateProximateDensityScore(locs []search.FieldTermLocation) float64 {
	fieldTerms := map[string]termLocs{}

	for _, l := range locs {
		if fieldLocations, ok := fieldTerms[l.Field]; ok {
			fieldTerms[l.Field] = insertSorted(fieldLocations, l)
		} else {
			fieldTerms[l.Field] = []search.FieldTermLocation{l}
		}
	}

	score := float64(0)
	for _, termLocs := range fieldTerms {
		if len(termLocs) == 0 {
			continue
		}

		prevPos := termLocs[0]

		for i := 1; i < len(termLocs); i++ {
			curPos := termLocs[i]
			dist := curPos.Location.Pos - prevPos.Location.Pos - 1 // prev term at 4, current term at 5, dist 0
			similarity := 1
			if prevPos.Term == curPos.Term {
				similarity = 0
			}

			prevPos = curPos
			posScore := dist + similarity

			score += 1.0 / float64(zeroOrGreater(posScore))
		}

	}

	return score
}

func zeroOrGreater(val int) int {
	if val >= 0 {
		return val
	}
	return 0
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
