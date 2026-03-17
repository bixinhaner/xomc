<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.addButtonDiv{
		position: absolute;
		right: 20px;
		top: 10px;
		width: 36px;
	}
	.tableDiv{
		height: 20px
	}
	.alarmSpan{
		display:inline-block;
		height:21px;
		line-height:24px;
		margin-left:8px;
	}
	.tableIcon{
		vertical-align:middle;
	}
	.minor .el-icon-status-alarm:before{
		color:#D0D53B;
	}
	.major .el-icon-status-alarm:before{
		color:#FB9F50;
	}
	.critical .el-icon-status-alarm:before{
		color:#FF7B7B;
	}
	.warning .el-icon-status-alarm:before{
		color:#67DFF8;
	}
</style>

<!-- KPI 告警-表格-操作-结果页面 -->
<div id="kpi_alarm_record">
	<el-ctable :rownumber="true" :height="'99%'" :url="tbURL" :pagination="true">
		<el-table-column label='<%=rb.getString("XuHao")%>' width="80"  prop="alarm_index"></el-table-column>
		<el-table-column label='<%=rb.getString("SheBeiWeiYiBiaoZhi")%>' width="180"  prop="serial_number"></el-table-column>
		<el-table-column label='<%=rb.getString("GaoJingJiBie")%>' width="100"  prop="event_type">
			<template slot-scope="scope">
				<div class="tableTdContainer minor" v-if="scope.row.event_type == 'Minor'">
					<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("CiYaoGaoJing")%></span>
				</div>
				<div class="tableTdContainer major" v-else-if="scope.row.event_type == 'Major'">
					<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("ZhuYaoGaoJing")%></span>
				</div>
				<div class="tableTdContainer critical" v-else-if="scope.row.event_type == 'Critical'">
					<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan" ><%=rb.getString("JinJiGaoJing")%></span>
				</div>
				<div class="tableTdContainer warning" v-else-if="scope.row.event_type == 'Warning'">
					<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("JingGaoGaoJing")%></span>
				</div>
				
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("BiaoShi")%>' width="150" prop="alarm_identifier"></el-table-column>
		<el-table-column label='<%=rb.getString("KeNengYuanYin")%>' prop="probable_cause"></el-table-column>
		<el-table-column label='<%=rb.getString("JuTiGuZhang")%>' prop="specific_problem"></el-table-column>
		<el-table-column label='<%=rb.getString("GaoJingFaShengShiJian")%>' prop="alarm_time"></el-table-column>
		<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status"></el-table-column>
	</el-ctable>
</div>

<script type="text/javascript">
	new Vue({
		el: '#kpi_alarm_record',
		data(){
			
			return {
				tbURL: '',
				menus:[],
				styleObj:{
			    	zIndex:10
			    },
			    addText: false,
			    buttonIcon:'el-icon-plus',
			    buttonText:'<%=rb.getString("TianJia") %>'
    	    }
		},
		methods: {
			init(id){
				var vm = this;
				vm.tbURL = '${ctx}/pm/alarm/getKpiAlarmInfoListPageData.action?tempId='+id+'&timeZone='+timeZone;
			}
		},
		mounted(){
			eventBus.$off('action-record').$on('action-record',this.init);
		}
	});
</script>