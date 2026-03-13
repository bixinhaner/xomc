package cmcc

import (
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

func kpiDefinitions(tech model.Technology) []*carrier.KPIDefinition {
	switch tech {
	case model.TechLTE:
		return lteKPIDefinitions()
	case model.TechNR:
		return nrKPIDefinitions()
	default:
		return nil
	}
}

func lteKPIDefinitions() []*carrier.KPIDefinition {
	return []*carrier.KPIDefinition{
		// Accessibility KPIs
		{
			Name:        "lte_rrc_setup_success_rate",
			DisplayName: "RRC 连接建立成功率",
			Formula:     "rrc_conn_setup_succ / rrc_conn_setup_att * 100",
			Unit:        "%",
			Counters:    []string{"rrc_conn_setup_succ", "rrc_conn_setup_att"},
			Category:    "accessibility",
		},
		{
			Name:        "lte_erab_setup_success_rate",
			DisplayName: "E-RAB 建立成功率",
			Formula:     "erab_setup_succ / erab_setup_att * 100",
			Unit:        "%",
			Counters:    []string{"erab_setup_succ", "erab_setup_att"},
			Category:    "accessibility",
		},
		{
			Name:        "lte_s1_signaling_setup_success_rate",
			DisplayName: "S1 信令连接建立成功率",
			Formula:     "s1_sig_conn_setup_succ / s1_sig_conn_setup_att * 100",
			Unit:        "%",
			Counters:    []string{"s1_sig_conn_setup_succ", "s1_sig_conn_setup_att"},
			Category:    "accessibility",
		},

		// Retainability KPIs
		{
			Name:        "lte_erab_drop_rate",
			DisplayName: "E-RAB 掉话率",
			Formula:     "erab_abnormal_release / (erab_abnormal_release + erab_normal_release) * 100",
			Unit:        "%",
			Counters:    []string{"erab_abnormal_release", "erab_normal_release"},
			Category:    "retainability",
		},

		// Mobility KPIs
		{
			Name:        "lte_intra_freq_ho_success_rate",
			DisplayName: "同频切换成功率",
			Formula:     "intra_freq_ho_succ / intra_freq_ho_att * 100",
			Unit:        "%",
			Counters:    []string{"intra_freq_ho_succ", "intra_freq_ho_att"},
			Category:    "mobility",
		},
		{
			Name:        "lte_inter_freq_ho_success_rate",
			DisplayName: "异频切换成功率",
			Formula:     "inter_freq_ho_succ / inter_freq_ho_att * 100",
			Unit:        "%",
			Counters:    []string{"inter_freq_ho_succ", "inter_freq_ho_att"},
			Category:    "mobility",
		},

		// Utilization KPIs
		{
			Name:        "lte_dl_prb_utilization",
			DisplayName: "下行 PRB 利用率",
			Formula:     "dl_prb_used_avg / dl_prb_total * 100",
			Unit:        "%",
			Counters:    []string{"dl_prb_used_avg", "dl_prb_total"},
			Category:    "utilization",
		},
		{
			Name:        "lte_ul_prb_utilization",
			DisplayName: "上行 PRB 利用率",
			Formula:     "ul_prb_used_avg / ul_prb_total * 100",
			Unit:        "%",
			Counters:    []string{"ul_prb_used_avg", "ul_prb_total"},
			Category:    "utilization",
		},

		// Throughput KPIs
		{
			Name:        "lte_dl_throughput",
			DisplayName: "下行吞吐量",
			Formula:     "pdcp_sdu_dl_volume * 8 / period_seconds / 1000000",
			Unit:        "Mbps",
			Counters:    []string{"pdcp_sdu_dl_volume"},
			Category:    "throughput",
		},
		{
			Name:        "lte_ul_throughput",
			DisplayName: "上行吞吐量",
			Formula:     "pdcp_sdu_ul_volume * 8 / period_seconds / 1000000",
			Unit:        "Mbps",
			Counters:    []string{"pdcp_sdu_ul_volume"},
			Category:    "throughput",
		},
	}
}

func nrKPIDefinitions() []*carrier.KPIDefinition {
	return []*carrier.KPIDefinition{
		// 5G Accessibility
		{
			Name:        "nr_rrc_setup_success_rate",
			DisplayName: "5G RRC 连接建立成功率",
			Formula:     "nr_rrc_conn_setup_succ / nr_rrc_conn_setup_att * 100",
			Unit:        "%",
			Counters:    []string{"nr_rrc_conn_setup_succ", "nr_rrc_conn_setup_att"},
			Category:    "accessibility",
		},
		{
			Name:        "nr_ng_signaling_setup_success_rate",
			DisplayName: "NG 信令连接建立成功率",
			Formula:     "ng_sig_conn_setup_succ / ng_sig_conn_setup_att * 100",
			Unit:        "%",
			Counters:    []string{"ng_sig_conn_setup_succ", "ng_sig_conn_setup_att"},
			Category:    "accessibility",
		},

		// 5G Retainability
		{
			Name:        "nr_session_drop_rate",
			DisplayName: "5G 会话掉线率",
			Formula:     "nr_session_abnormal_release / (nr_session_abnormal_release + nr_session_normal_release) * 100",
			Unit:        "%",
			Counters:    []string{"nr_session_abnormal_release", "nr_session_normal_release"},
			Category:    "retainability",
		},

		// 5G Throughput
		{
			Name:        "nr_dl_user_throughput",
			DisplayName: "5G 下行用户吞吐量",
			Formula:     "nr_pdcp_sdu_dl_volume * 8 / nr_active_ue_dl_time / 1000000",
			Unit:        "Mbps",
			Counters:    []string{"nr_pdcp_sdu_dl_volume", "nr_active_ue_dl_time"},
			Category:    "throughput",
		},
		{
			Name:        "nr_ul_user_throughput",
			DisplayName: "5G 上行用户吞吐量",
			Formula:     "nr_pdcp_sdu_ul_volume * 8 / nr_active_ue_ul_time / 1000000",
			Unit:        "Mbps",
			Counters:    []string{"nr_pdcp_sdu_ul_volume", "nr_active_ue_ul_time"},
			Category:    "throughput",
		},

		// 5G Latency
		{
			Name:        "nr_dl_average_latency",
			DisplayName: "5G 下行平均时延",
			Formula:     "nr_dl_total_delay / nr_dl_total_packets",
			Unit:        "ms",
			Counters:    []string{"nr_dl_total_delay", "nr_dl_total_packets"},
			Category:    "latency",
		},
	}
}
