package provision

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchTemplate(t *testing.T) {
	templates := []template.ConfigTemplate{
		{
			ID:           uuid.New(),
			Name:         "cmcc_lte_default",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{}`),
			Priority:     0,
			Active:       true,
		},
		{
			ID:           uuid.New(),
			Name:         "cmcc_lte_huawei",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			ProductClass: "SmallCell-LTE",
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{}`),
			Priority:     10,
			Active:       true,
		},
		{
			ID:           uuid.New(),
			Name:         "ctcc_nr_default",
			Carrier:      model.CarrierCTCC,
			Technology:   model.TechNR,
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{}`),
			Priority:     0,
			Active:       true,
		},
		{
			ID:           uuid.New(),
			Name:         "inactive_template",
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			TemplateType: template.TemplateProvisioning,
			Parameters:   json.RawMessage(`{}`),
			Priority:     100,
			Active:       false,
		},
	}

	t.Run("match by carrier+tech+productClass", func(t *testing.T) {
		device := &model.Device{
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			ProductClass: "SmallCell-LTE",
		}
		result := MatchTemplate(templates, device)
		require.NotNil(t, result)
		assert.Equal(t, "cmcc_lte_huawei", result.Name)
	})

	t.Run("fallback to carrier+tech when productClass not matched", func(t *testing.T) {
		device := &model.Device{
			Carrier:      model.CarrierCMCC,
			Technology:   model.TechLTE,
			ProductClass: "Unknown-Product",
		}
		result := MatchTemplate(templates, device)
		require.NotNil(t, result)
		assert.Equal(t, "cmcc_lte_default", result.Name)
	})

	t.Run("match different carrier", func(t *testing.T) {
		device := &model.Device{
			Carrier:    model.CarrierCTCC,
			Technology: model.TechNR,
		}
		result := MatchTemplate(templates, device)
		require.NotNil(t, result)
		assert.Equal(t, "ctcc_nr_default", result.Name)
	})

	t.Run("no match for unknown carrier", func(t *testing.T) {
		device := &model.Device{
			Carrier:    model.CarrierCUCC,
			Technology: model.TechLTE,
		}
		result := MatchTemplate(templates, device)
		assert.Nil(t, result)
	})

	t.Run("nil device returns nil", func(t *testing.T) {
		result := MatchTemplate(templates, nil)
		assert.Nil(t, result)
	})

	t.Run("empty templates returns nil", func(t *testing.T) {
		device := &model.Device{Carrier: model.CarrierCMCC, Technology: model.TechLTE}
		result := MatchTemplate(nil, device)
		assert.Nil(t, result)
	})

	t.Run("inactive templates are skipped", func(t *testing.T) {
		device := &model.Device{
			Carrier:    model.CarrierCMCC,
			Technology: model.TechLTE,
		}
		// The inactive template has highest priority but should be skipped.
		result := MatchTemplate(templates, device)
		require.NotNil(t, result)
		assert.True(t, result.Active)
	})
}
