package bleveadapter

import (
	"fmt"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/tour-of-go/k8s-event-sink/internal/core"
)

// doc is the Bleve document structure.
type doc struct {
	ID        string    `json:"id"`
	Namespace string    `json:"namespace"`
	Pod       string    `json:"pod"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Count     int       `json:"count"`
	LastSeen  time.Time `json:"last_seen"`
}

// Index implements core.SearchPort using Bleve.
type Index struct {
	idx bleve.Index
}

func New(path string) (*Index, error) {
	idx, err := bleve.Open(path)
	if err == bleve.ErrorIndexPathDoesNotExist {
		mapping := bleve.NewIndexMapping()
		idx, err = bleve.New(path, mapping)
	}
	if err != nil {
		return nil, fmt.Errorf("opening bleve index: %w", err)
	}
	return &Index{idx: idx}, nil
}

func (i *Index) Index(event core.Event) error {
	d := doc{
		ID:        event.ID,
		Namespace: event.Namespace,
		Pod:       event.Pod,
		Reason:    event.Reason,
		Message:   event.Message,
		Severity:  event.Severity,
		Count:     event.Count,
		LastSeen:  event.LastSeen,
	}
	return i.idx.Index(event.ID, d)
}

func (i *Index) Search(query string) ([]core.Event, error) {
	q := bleve.NewMatchQuery(query)
	req := bleve.NewSearchRequest(q)
	req.Size = 100
	req.Fields = []string{"*"}

	result, err := i.idx.Search(req)
	if err != nil {
		return nil, fmt.Errorf("bleve search: %w", err)
	}

	var events []core.Event
	for _, hit := range result.Hits {
		e := core.Event{
			ID:        hit.ID,
			Namespace: fieldStr(hit.Fields, "namespace"),
			Pod:       fieldStr(hit.Fields, "pod"),
			Reason:    fieldStr(hit.Fields, "reason"),
			Message:   fieldStr(hit.Fields, "message"),
			Severity:  fieldStr(hit.Fields, "severity"),
		}
		if c, ok := hit.Fields["count"].(float64); ok {
			e.Count = int(c)
		}
		events = append(events, e)
	}
	return events, nil
}

func (i *Index) Close() error { return i.idx.Close() }

func fieldStr(fields map[string]interface{}, key string) string {
	if v, ok := fields[key].(string); ok {
		return v
	}
	return ""
}
