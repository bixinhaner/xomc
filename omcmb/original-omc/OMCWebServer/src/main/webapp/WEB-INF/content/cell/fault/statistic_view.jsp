<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>
<style type="text/css">
.addButtonDiv{
	position:absolute;
	right:20px;
	top:7px;
	width:36px;
}
.addButtonText{
	text-align:center;
	font-size:16px;
	color:#1DA3FC;
	margin-top:5px;
}
.viewSlide .slide-content{
	height:unset !important;
}
.viewSlide .el-ctable{
	height:100%;
}
.viewSlide .el-card .el-card__body > div:first-child{
	overflow:hidden !important;
}
.mubanTitle{
	font-weight:700;
	font-style:'normal';
	font-size:14px;
	color:#5A7B92;
	margin-left:20px;
}
.mubanName{
	font-weight:700;
	font-style:'normal';
	font-size:14px;
	color:#5A7B92;
}
#viewStatisticPage{
	display:flex;
	flex-direction:column;
}

</style>
<div class="panelDefault" id="viewStatisticPage">
	<div style="margin-top:10px;"><span class="mubanTitle"><%=rb.getString("MuBanMingCheng")%> ：</span><span class="mubanName">&nbsp;{{muBanTitle}}</span></div>
	<div class="circleIcon" style="top:10px;right:110px">
		<span class="el-icon el-icon-circle-chart" @click="openEchart"></span>
		<div class="titleButtonText"><%=rb.getString("TuBiao")%></div>
	</div>
	<div class="circleIcon" style="top:10px;right:55px">
		<span class="el-icon el-icon-circle-export" @click="exportStatistic"></span>
		<div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
	</div>
	<div class="circleIcon" style="top:10px;">
		<span class="el-icon el-icon-circle-close" @click="goStatisticTable"></span>
		<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	</div>
	
	<!-- 表格 -->
	<div style="flex:1 1 auto;overflow:hidden;">
			<template>
			<div class="tableContainer" style="height:100%;"><!-- :url="viewStatisticUrl" :data="viewTableData"   -->
				<el-ctable ref="statisticViewTable" :url=viewStatisticUrl :query-params="params" :height="height" pagination="true" :rownumber="rownumber">
					<template slot="toolbar">
						<div class="queryGroup">
							<el-input class='pairgrid-query' style='width:329px;' v-model="params.searchText" @keyup.enter.native="searchResult" size="mini" 
								placeholder="<%=rb.getString("SheBeiWeiYiBiaoZhi")%>" v-if="statisticObject == 0"></el-input>
							<el-input class='pairgrid-query' style='width:329px;' v-model="params.searchText" @keyup.enter.native="searchResult" size="mini" 
								placeholder="<%=rb.getString("SheBeiZuMingCheng")%>" v-if="statisticObject == 1"></el-input>
							<el-input class='pairgrid-query' style='width:329px;' v-model="params.searchText" @keyup.enter.native="searchResult" size="mini" 
								placeholder="<%=rb.getString("AlarmId")%>" v-if="statisticObject == 2"></el-input>
					    	<i @click='searchResult' class="el-icon el-icon-common-search"></i>
				    	</div>
					</template>
					
					<el-table-column v-if="statisticObject == 0" prop="statisticObject" label="<%=rb.getString("SheBeiWeiYiBiaoZhi")%>" ></el-table-column>
					<el-table-column v-if="statisticObject == 1" prop="statisticObject" label="<%=rb.getString("SheBeiZuMingCheng")%>" ></el-table-column>
					<el-table-column v-if="statisticObject == 2" prop="statisticObject" label="<%=rb.getString("AlarmId")%>" ></el-table-column>
					<el-table-column prop="alarmCount" label="<%=rb.getString("GaoJingShu")%>" ></el-table-column>
					<el-table-column prop="startTime" label="<%=rb.getString("KaiShiShiJian")%>" ></el-table-column>
					<el-table-column prop="endTime" label="<%=rb.getString("JieShuShiJian")%>" ></el-table-column>
				</el-ctable>
			</div>
		</template>
		<el-slide ref="echartSlide" :url='echartSlide' :title="echartSlideTitle" :footer="echartSlideFooter" :header="echartSlideHeader" :position="echartSlideposition"
		:height="echartSlideHeight" :modal='modal' :width='echartSlideWidth' @ok="" @cancel="" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
	</div>
	
</div>
<form id="downLoadAlarmStatisticFile" style="display:none" method="post" action=""></form>
<script type="text/javascript">
new Vue({
	el:"#viewStatisticPage",
	data(){
		return{
			echartSlide:'',
			echartSlideTitle:'',
			echartSlideFooter:'',
			echartSlideHeader:'',
			echartSlideposition:'',
			echartSlideHeight:'',
			echartSlideWidth:'',
			viewStatisticUrl:'${ctx}/cell/fault/queryAlarmStatisticResult.action',
			modal:false,
			statisticObject:'',
			muBanTitle:'admin1111',
			viewTableData:[
				{sn:'1234567890',alarmCount:'50',startTime:'2019-10-11',endTime:'2019-10-11'},
				{sn:'1234567890',alarmCount:'50',startTime:'2019-10-11',endTime:'2019-10-11'}
			],
			statisticTime:'',
			startTime:'',
			endTime:'',
			rownumber:true,
			params:{
				searchText:'',
				statisticId:'',
				timeZone:timeZone,
			},
			height:'100%',
			pageSize:50,
		
		}
	},
	methods:{
		// 搜索事件
		searchResult(){
			this.$refs.statisticViewTable.refresh();
		},
		// 打开图表页面
	    openEchart(){
	    	var vm = this;
	    	vm.echartSlideHeader = false;
	    	vm.echartSlide = '${ctx}/cell/fault/goAlarmStatisticGraph.action';
	    	vm.echartSlideFooter = false;
	    	vm.echartSlideposition = 'top';
	        vm.echartSlideHeight = '100%';
	    	vm.echartSlideWidth = '100%';
	    	vm.$refs.echartSlide.showSlide(()=>{
	    		eventBus.$emit('open-echart',vm.statisticTime,vm.startTime,vm.endTime,vm.params.statisticId, vm.statisticObject);
    			vm.model = true;
    		})
			
	    },
	    goStatisticTable(){
	    	var vm = this;
	    	eventBus.$emit('hide-view-statistic');
	    },
		/**
		* 结果页面信息
		* @param statisticId{number}   统计模板id
		* @param statisticTime{number}   统计粒度
		* @param startTime{string}   统计开始时间
		* @param endTime{string}   统计结束时间
		* @param statisticType{number}   统计类型 
		* @param statisticName{string}   统计模板名称
		*/
	    initViewTable(statisticId,statisticTime,startTime,endTime,statisticType,statisticName){
	    	var vm = this;
	    	this.params.statisticId = statisticId;
	    	vm.statisticTime = statisticTime;
	    	vm.startTime = startTime;
			vm.endTime = endTime;
			vm.statisticObject = statisticType;
			vm.muBanTitle = statisticName;
	    },
		// 结果页面关闭
	    closeChartPage(){
	    	var vm = this;
	    	vm.$refs.echartSlide.hide();
	    },
		// 图表数据导出
	    exportStatistic(){
			var vm = this;
	    	/* $("#downLoadAlarmStatisticFile").form('submit', {
				url: "${ctx}/cell/fault/exportAlarmStatisticResult.action",
				onSubmit: function(param) {
		            param.searchText = vm.params.searchText;
		            param.statisticId = vm.params.statisticId;
		            param.timeZone = vm.params.timeZone;
					var bool = checkParams(param)
					if(!bool) return false;
		        }
			}); */
			exportByForm("${ctx}/cell/fault/exportAlarmStatisticResult.action",{
				searchText: vm.params.searchText,
				statisticId: vm.params.statisticId,
				timeZone: vm.params.timeZone
			});
	    }
	    
	},
	beforeMount(){
		eventBus.$off('view-info').$on('view-info',this.initViewTable);
	},
	mounted(){
		eventBus.$off('hide-chartPage').$on('hide-chartPage',this.closeChartPage);
	
	}
	
})
</script>