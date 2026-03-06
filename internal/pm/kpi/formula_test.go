package kpi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormula_ParseAndEvaluate(t *testing.T) {
	tests := []struct {
		name       string
		formula    string
		values     map[string]float64
		want       float64
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "simple division percentage",
			formula: "a / b * 100",
			values:  map[string]float64{"a": 95, "b": 100},
			want:    95.0,
		},
		{
			name:    "RRC setup success rate",
			formula: "rrc_conn_setup_succ / rrc_conn_setup_att * 100",
			values:  map[string]float64{"rrc_conn_setup_succ": 950, "rrc_conn_setup_att": 1000},
			want:    95.0,
		},
		{
			name:    "throughput calculation",
			formula: "a * 8 / b / 1000000",
			values:  map[string]float64{"a": 125000000, "b": 900},
			want:    125000000.0 * 8 / 900.0 / 1000000.0,
		},
		{
			name:    "parentheses grouping",
			formula: "(a + b) / c * 100",
			values:  map[string]float64{"a": 10, "b": 90, "c": 200},
			want:    50.0,
		},
		{
			name:    "ERAB drop rate with parentheses",
			formula: "erab_abnormal_release / (erab_abnormal_release + erab_normal_release) * 100",
			values:  map[string]float64{"erab_abnormal_release": 5, "erab_normal_release": 95},
			want:    5.0,
		},
		{
			name:    "addition and subtraction",
			formula: "a + b - c",
			values:  map[string]float64{"a": 100, "b": 50, "c": 30},
			want:    120.0,
		},
		{
			name:    "multiplication precedence over addition",
			formula: "a + b * c",
			values:  map[string]float64{"a": 10, "b": 5, "c": 3},
			want:    25.0, // 10 + (5*3)
		},
		{
			name:    "float constants",
			formula: "a * 1.5",
			values:  map[string]float64{"a": 100},
			want:    150.0,
		},
		{
			name:       "division by zero",
			formula:    "a / b",
			values:     map[string]float64{"a": 100, "b": 0},
			wantErr:    true,
			wantErrMsg: "division by zero",
		},
		{
			name:       "unknown counter",
			formula:    "a / b * 100",
			values:     map[string]float64{"a": 95},
			wantErr:    true,
			wantErrMsg: "counter not found: b",
		},
		{
			name:       "invalid expression - double operator",
			formula:    "a ** b",
			wantErr:    true,
			wantErrMsg: "parse formula",
		},
		{
			name:       "empty formula",
			formula:    "",
			wantErr:    true,
			wantErrMsg: "unexpected end",
		},
		{
			name:       "unmatched parenthesis",
			formula:    "(a + b",
			wantErr:    true,
			wantErrMsg: "missing closing parenthesis",
		},
		{
			name:    "nested parentheses",
			formula: "((a + b) * c) / d",
			values:  map[string]float64{"a": 10, "b": 20, "c": 3, "d": 5},
			want:    18.0,
		},
		{
			name:    "single counter",
			formula: "counter_x",
			values:  map[string]float64{"counter_x": 42.5},
			want:    42.5,
		},
		{
			name:    "single number",
			formula: "100",
			values:  map[string]float64{},
			want:    100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formula, parseErr := ParseFormula(tt.formula)
			if tt.wantErr && parseErr != nil {
				// Parse error is acceptable for invalid expression tests
				if tt.wantErrMsg != "" {
					assert.Contains(t, parseErr.Error(), tt.wantErrMsg)
				}
				return
			}
			if parseErr != nil && !tt.wantErr {
				t.Fatalf("unexpected parse error: %v", parseErr)
			}

			require.NotNil(t, formula)

			result, evalErr := formula.Evaluate(tt.values)
			if tt.wantErr {
				require.Error(t, evalErr)
				if tt.wantErrMsg != "" {
					assert.Contains(t, evalErr.Error(), tt.wantErrMsg)
				}
				return
			}

			require.NoError(t, evalErr)
			assert.InDelta(t, tt.want, result, 0.0001, "expected %.4f, got %.4f", tt.want, result)
		})
	}
}

func TestParseAllCarrierFormulas(t *testing.T) {
	// Test that all the actual carrier KPI formulas parse successfully
	formulas := []string{
		"rrc_conn_setup_succ / rrc_conn_setup_att * 100",
		"erab_setup_succ / erab_setup_att * 100",
		"s1_sig_conn_setup_succ / s1_sig_conn_setup_att * 100",
		"erab_abnormal_release / (erab_abnormal_release + erab_normal_release) * 100",
		"intra_freq_ho_succ / intra_freq_ho_att * 100",
		"inter_freq_ho_succ / inter_freq_ho_att * 100",
		"dl_prb_used_avg / dl_prb_total * 100",
		"ul_prb_used_avg / ul_prb_total * 100",
		"pdcp_sdu_dl_volume * 8 / period_seconds / 1000000",
		"pdcp_sdu_ul_volume * 8 / period_seconds / 1000000",
		"nr_rrc_conn_setup_succ / nr_rrc_conn_setup_att * 100",
		"ng_sig_conn_setup_succ / ng_sig_conn_setup_att * 100",
		"nr_session_abnormal_release / (nr_session_abnormal_release + nr_session_normal_release) * 100",
		"nr_pdcp_sdu_dl_volume * 8 / nr_active_ue_dl_time / 1000000",
		"nr_pdcp_sdu_ul_volume * 8 / nr_active_ue_ul_time / 1000000",
		"nr_dl_total_delay / nr_dl_total_packets",
	}

	for _, f := range formulas {
		t.Run(f, func(t *testing.T) {
			formula, err := ParseFormula(f)
			require.NoError(t, err, "failed to parse formula: %s", f)
			require.NotNil(t, formula)
		})
	}
}
