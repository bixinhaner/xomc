package collector

import (
	"strings"

	"github.com/omcgo/omcgo/internal/core/model"
)

const (
	minTechnologyFamilies = 2
	maxTechnologySignals  = 20
)

var technologyFamilies = map[string][]string{
	"gsm": {"BTS.", "SDCCH.", "TCH.", "CALL."},
	// RRC/PDCP/MAC/RLC/RRU/PHY are intentionally absent: current LTE and NR
	// vendor libraries share those prefixes, so treating them as evidence
	// would quarantine valid NR files as LTE.
	"lte": {"ERAB.", "S1SIG.", "S1.", "X2SIG."},
	"nr":  {"NGSIG.", "NR.", "GNBCU.", "GNBDU.", "NRCELL."},
}

func classifyManagedElementIdentity(identity string) string {
	upper := strings.ToUpper(strings.TrimSpace(identity))
	switch {
	case strings.Contains(upper, "GNB"):
		return "nr"
	case strings.Contains(upper, "ENB"):
		return "lte"
	case strings.Contains(upper, "BSC"), strings.Contains(upper, "BTS"):
		return "gsm"
	default:
		return ""
	}
}

// ClassifyTechnologyWithIdentity prefers unambiguous counter-family evidence.
// If a sparse file has insufficient counters, the managedElement identity is
// used only when it contains an explicit eNB/gNB/BSC/BTS technology marker.
func ClassifyTechnologyWithIdentity(identity string, counters []model.PMCounter) TechnologyEvidence {
	evidence := ClassifyTechnology(counters)
	if evidence.Technology != "" {
		return evidence
	}
	technology := classifyManagedElementIdentity(identity)
	if technology == "" {
		return TechnologyEvidence{}
	}
	return TechnologyEvidence{
		Technology: technology,
		Confidence: 1,
		Signals:    []string{"managedElement:" + identity},
	}
}

// TechnologyEvidence is a bounded explanation of the technology inferred
// from vendor PM counter families. Empty Technology means the payload did not
// contain enough unambiguous evidence to classify safely.
type TechnologyEvidence struct {
	Technology string
	Confidence float64
	Signals    []string
}

// ClassifyTechnology infers GSM/LTE/NR from independent counter-name
// families. It deliberately ignores common OTHER.* counters and returns
// unknown for ties or single-family evidence so collection stays fail-open
// when a vendor payload is sparse.
func ClassifyTechnology(counters []model.PMCounter) TechnologyEvidence {
	type score struct {
		families map[string]struct{}
		signals  []string
	}
	scores := make(map[string]*score, len(technologyFamilies))
	for technology := range technologyFamilies {
		scores[technology] = &score{families: make(map[string]struct{})}
	}

	for _, counter := range counters {
		name := strings.ToUpper(strings.TrimSpace(counter.CounterName))
		for technology, families := range technologyFamilies {
			for _, family := range families {
				if !strings.HasPrefix(name, family) {
					continue
				}
				current := scores[technology]
				current.families[family] = struct{}{}
				if len(current.signals) < maxTechnologySignals {
					current.signals = append(current.signals, counter.CounterName)
				}
				break
			}
		}
	}

	topTechnology := ""
	topFamilies := 0
	secondFamilies := 0
	totalFamilies := 0
	for technology, current := range scores {
		count := len(current.families)
		totalFamilies += count
		if count > topFamilies {
			secondFamilies = topFamilies
			topFamilies = count
			topTechnology = technology
		} else if count > secondFamilies {
			secondFamilies = count
		}
	}
	if topFamilies < minTechnologyFamilies || topFamilies == secondFamilies {
		return TechnologyEvidence{}
	}

	return TechnologyEvidence{
		Technology: topTechnology,
		Confidence: float64(topFamilies) / float64(totalFamilies),
		Signals:    scores[topTechnology].signals,
	}
}
