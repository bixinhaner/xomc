package export

import (
	"bytes"
	"encoding/json"
	"fmt"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func withExportLocale(raw []byte, loc appcontext.Locale) ([]byte, error) {
	params := make(map[string]json.RawMessage)
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, fmt.Errorf("export params must be a JSON object: %w", err)
		}
		if params == nil {
			return nil, fmt.Errorf("export params must be a JSON object")
		}
	}
	loc = normalizeExportLocale(loc)
	encoded, err := json.Marshal(loc)
	if err != nil {
		return nil, fmt.Errorf("marshal export locale: %w", err)
	}
	params["locale"] = encoded
	out, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal export params: %w", err)
	}
	return out, nil
}

func exportLocale(raw []byte) appcontext.Locale {
	var params struct {
		Locale appcontext.Locale `json:"locale"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return appcontext.LocaleZH
	}
	return normalizeExportLocale(params.Locale)
}

func normalizeExportLocale(loc appcontext.Locale) appcontext.Locale {
	if loc == appcontext.LocaleEN {
		return appcontext.LocaleEN
	}
	return appcontext.LocaleZH
}
