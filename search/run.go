package search

import (
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/nicksellen/pictures/index"
)

// Run a very specify query
func Run() error {
	index, err := index.OpenIndexReadOnly("db.bleve")

	if err != nil {
		return err
	}

	field := "EXIF:Orientation"

	min := float64(8)
	max := float64(8)
	trueVal := true
	query := bleve.NewNumericRangeInclusiveQuery(&min, &max, &trueVal, &trueVal)
	query.FieldVal = field

	request := bleve.NewSearchRequestOptions(query, 100, 0, false)
	request.Fields = []string{field}
	results, err := index.Search(request)
	if err != nil {
		return err
	}

	for _, hit := range results.Hits {
		fmt.Printf("%s\n", hit.ID)
	}

	return nil

}
