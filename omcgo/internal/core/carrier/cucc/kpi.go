package cucc

import (
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/model"
)

func kpiDefinitions(tech model.Technology) []*carrier.KPIDefinition {
	if tech == model.TechNR {
		return nrKPIDefinitions()
	}
	return nil
}

func nrKPIDefinitions() []*carrier.KPIDefinition {
	return []*carrier.KPIDefinition{
		// 5G Accessibility KPIs
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
		{
			Name:        "nr_pdu_session_setup_success_rate",
			DisplayName: "PDU 会话建立成功率",
			Formula:     "nr_pdu_session_setup_succ / nr_pdu_session_setup_att * 100",
			Unit:        "%",
			Counters:    []string{"nr_pdu_session_setup_succ", "nr_pdu_session_setup_att"},
			Category:    "accessibility",
		},

		// 5G Retainability KPIs
		{
			Name:        "nr_session_drop_rate",
			DisplayName: "5G 会话掉线率",
			Formula:     "nr_session_abnormal_release / (nr_session_abnormal_release + nr_session_normal_release) * 100",
			Unit:        "%",
			Counters:    []string{"nr_session_abnormal_release", "nr_session_normal_release"},
			Category:    "retainability",
		},
		{
			Name:        "nr_rrc_connection_drop_rate",
			DisplayName: "5G RRC 连接掉线率",
			Formula:     "nr_rrc_conn_abnormal_release / (nr_rrc_conn_abnormal_release + nr_rrc_conn_normal_release) * 100",
			Unit:        "%",
			Counters:    []string{"nr_rrc_conn_abnormal_release", "nr_rrc_conn_normal_release"},
			Category:    "retainability",
		},

		// 5G Mobility KPIs
		{
			Name:        "nr_intra_freq_ho_success_rate",
			DisplayName: "5G 同频切换成功率",
			Formula:     "nr_intra_freq_ho_succ / nr_intra_freq_ho_att * 100",
			Unit:        "%",
			Counters:    []string{"nr_intra_freq_ho_succ", "nr_intra_freq_ho_att"},
			Category:    "mobility",
		},
		{
			Name:        "nr_inter_freq_ho_success_rate",
			DisplayName: "5G 异频切换成功率",
			Formula:     "nr_inter_freq_ho_succ / nr_inter_freq_ho_att * 100",
			Unit:        "%",
			Counters:    []string{"nr_inter_freq_ho_succ", "nr_inter_freq_ho_att"},
			Category:    "mobility",
		},
		{
			Name:        "nr_xn_ho_success_rate",
			DisplayName: "5G Xn 切换成功率",
			Formula:     "nr_xn_ho_succ / nr_xn_ho_att * 100",
			Unit:        "%",
			Counters:    []string{"nr_xn_ho_succ", "nr_xn_ho_att"},
			Category:    "mobility",
		},

		// 5G Throughput KPIs
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
		{
			Name:        "nr_dl_cell_throughput",
			DisplayName: "5G 下行小区吞吐量",
			Formula:     "nr_pdcp_sdu_dl_volume * 8 / period_seconds / 1000000",
			Unit:        "Mbps",
			Counters:    []string{"nr_pdcp_sdu_dl_volume"},
			Category:    "throughput",
		},
		{
			Name:        "nr_ul_cell_throughput",
			DisplayName: "5G 上行小区吞吐量",
			Formula:     "nr_pdcp_sdu_ul_volume * 8 / period_seconds / 1000000",
			Unit:        "Mbps",
			Counters:    []string{"nr_pdcp_sdu_ul_volume"},
			Category:    "throughput",
		},

		// 5G Latency KPIs
		{
			Name:        "nr_dl_average_latency",
			DisplayName: "5G 下行平均时延",
			Formula:     "nr_dl_total_delay / nr_dl_total_packets",
			Unit:        "ms",
			Counters:    []string{"nr_dl_total_delay", "nr_dl_total_packets"},
			Category:    "latency",
		},
		{
			Name:        "nr_ul_average_latency",
			DisplayName: "5G 上行平均时延",
			Formula:     "nr_ul_total_delay / nr_ul_total_packets",
			Unit:        "ms",
			Counters:    []string{"nr_ul_total_delay", "nr_ul_total_packets"},
			Category:    "latency",
		},

		// 5G Utilization KPIs
		{
			Name:        "nr_dl_prb_utilization",
			DisplayName: "5G 下行 PRB 利用率",
			Formula:     "nr_dl_prb_used_avg / nr_dl_prb_total * 100",
			Unit:        "%",
			Counters:    []string{"nr_dl_prb_used_avg", "nr_dl_prb_total"},
			Category:    "utilization",
		},
		{
			Name:        "nr_ul_prb_utilization",
			DisplayName: "5G 上行 PRB 利用率",
			Formula:     "nr_ul_prb_used_avg / nr_ul_prb_total * 100",
			Unit:        "%",
			Counters:    []string{"nr_ul_prb_used_avg", "nr_ul_prb_total"},
			Category:    "utilization",
		},
		{
			Name:        "nr_active_ue_avg",
			DisplayName: "5G 平均激活用户数",
			Formula:     "nr_active_ue_dl_sum / nr_active_ue_sample_count",
			Unit:        "",
			Counters:    []string{"nr_active_ue_dl_sum", "nr_active_ue_sample_count"},
			Category:    "utilization",
		},
	}
}
