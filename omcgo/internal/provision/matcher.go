package provision

import (
	"github.com/omcgo/omcgo/internal/model"
	"github.com/omcgo/omcgo/internal/config/template"
)

// MatchTemplate selects the best-matching template from a list of candidates.
// Priority: carrier + tech + product_class > carrier + tech (no product_class).
func MatchTemplate(templates []template.ConfigTemplate, device *model.Device) *template.ConfigTemplate {
	if device == nil || len(templates) == 0 {
		return nil
	}

	var bestMatch *template.ConfigTemplate
	bestScore := -1

	for i := range templates {
		t := &templates[i]
		if !t.Active {
			continue
		}
		if t.Carrier != device.Carrier || t.Technology != device.Technology {
			continue
		}

		score := 0

		// Score 2: carrier + tech + product_class match.
		if t.ProductClass != "" && t.ProductClass == device.ProductClass {
			score = 2
		} else if t.ProductClass == "" {
			// Score 1: carrier + tech match (no product_class constraint).
			score = 1
		} else {
			// product_class specified but doesn't match.
			continue
		}

		// Higher priority breaks ties within the same score.
		if score > bestScore || (score == bestScore && bestMatch != nil && t.Priority > bestMatch.Priority) {
			bestMatch = t
			bestScore = score
		}
	}

	return bestMatch
}
