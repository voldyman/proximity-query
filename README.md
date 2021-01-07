# Proximity Based Scoring for Bluge

**This package is WIP**

The default composite scorer in [Bluge](https://github.com/blugelabs/bluge) only sums the score of the matches. This can cause less relevant matches to bubble up to the top since the term scoring only looks at frequency for ranking (tf-idf but still frequencies).

In this package I've created a composite scorer that modifies the score based on how close the terms per field are to each other. And there is a `ProximityQuery` since the [`BooleanQuery`](https://godoc.org/github.com/blugelabs/bluge#BooleanQuery) in bluge doesn't expose the scorer (yet).

## Usage

Usage is very similar to boolean query.

```go
subQuery := bluge.NewMatchQuery("...").SetField("...")

query := proximity.NewProximityQuery().AddSubquery(subQuery)

req := bluge.NewTopNSearch(numResults, query)
// ...
```

## Performance

This package is more of a proof of concept than high performance code, there are benchmarks nonetheless.

```
BenchmarkDefaultCompositeScorer-8   	   42183	     27577 ns/op
BenchmarkProximityCompositeScorer-8   	   40339	     28816 ns/op
```
