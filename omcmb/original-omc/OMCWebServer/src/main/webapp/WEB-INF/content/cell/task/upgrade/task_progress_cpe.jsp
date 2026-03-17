<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
#cpeUpgradeResult .pairgrid-query .el-input__inner{
	width:400px;
}
</style>
<div id="cpeUpgradeResult">
	<el-ctable ref="ctable" :url="url" :query-params="params" height="255">
		<template slot="toolbar">
			<div class='queryGroup'>
				<el-input class='pairgrid-query' style='width:400px;' v-model="params.searchText" @keyup.enter.native="searchResult"
				placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%>" size="mini" ></el-input>
	    		<i @click='searchResult' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
			</div>
		</template>
		<el-table-column prop="ID" v-if="false"></el-table-column>
		<el-table-column prop="SERIAL_NUMBER" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" width="180"></el-table-column>
		<el-table-column prop="HOST_NAME" show-overflow-tooltip label="<%=rb.getString("CPEName")%>"  width="200"></el-table-column>
		<el-table-column prop="ORI_VERSION" show-overflow-tooltip label="<%=rb.getString("ChuShiBanBen")%>" width="160"></el-table-column>
		<el-table-column prop="PROGRESS_STATUS" label="<%=rb.getString("ZhuangTai")%>" width="90">
			<template slot-scope="scope">
				<div v-html="getHtml(scope.row.PROGRESS_STATUS)"></div>
			</template>
		</el-table-column>
		<el-table-column prop="PROGRESS_RESULT" label="<%=rb.getString("JieGuo")%>" width="90" :formatter="resultTableResult"></el-table-column>
		<el-table-column prop="FAILURE_REASON" show-overflow-tooltip label="<%=rb.getString("PCILOCKShiBaiYuanYin")%>"></el-table-column>
		<el-table-column prop="RUN_TIME" label="<%=rb.getString("ShiJian")%>" width="180"></el-table-column>
	</el-ctable>
</div>
<form id="upgradTaskProgResult" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId"/>
</form>
<script>

$(function(){
	closeLoading();
	new Vue({
		el:"#cpeUpgradeResult",
		data(){
			return {
				url:"${ctx}/task/upgrade/cpe/getUpgradeTaskProgressForCpe.action?task_id=${taskInfo.TASK_ID}",
				params:{
					timeZone: timeZone,
					searchText:""
				},
			}
		},
		created(){
			eventBus.$off('hander-cancel').$on('hander-cancel',this.cancelSubmit);
			eventBus.$off('export-result').$on('export-result',this.exportResult);
			eventBus.$off('setinterval-fresh').$on('setinterval-fresh',this.searchResult);
		},
		methods:{
			getHtml(status){
				return resultTableStatus(status)
			},
			searchResult(){
				this.$refs.ctable.refresh();
			},
			resultTableStatus(row,column,cellValue,index){
				var statusObj = {
					'1':"<%=rb.getString("DengDai")%>",	
					'2':"<%=rb.getString("JinXingZhong")%>",	
					'3':"<%=rb.getString("ZanTing")%>",	
					'4':"<%=rb.getString("YiJieShu")%>",	
					'5':"<%=rb.getString("JinXingZhong")%>",
					'6':"<%=rb.getString("ZhongJianBanBenShengJiZhong")%>",
					'7':"<%=rb.getString("MoKuaiBanBenShengJiZhong")%>",
					'8':"<%=rb.getString("DiBanBanBenShengJiZhong")%>",
					'9':"<%=rb.getString("CanShuPeiZhiZhong")%>",
					'':"",	
				}
				return statusObj[cellValue];
			},
			resultTableResult(row,column,cellValue,index){
				var resultObj = {
					"1" : "<%=rb.getString("ChengGong")%>",
					"2" : "<%=rb.getString("ZhongZhi")%>",
					"3" : "<%=rb.getString("ShiBai")%>",
					"" : "",
				}
				return resultObj[cellValue];
			},
			cancelSubmit(){
				eventBus.$emit('closeRightDiv','success');
			},
			exportResult(){
				var vm = this;
				/* $("#upgradTaskProgResult").form('submit', {
					url: "${ctx}/task/upgrade/cpe/exportUpgradeProgResultForCpe.action",
					onSubmit: function(param) {
			            param.timeZone=timeZone;
			            param.searchText = vm.params.searchText
						var bool = checkParams(param)
						if(!bool) return false;
			        }
				}); */
				exportByForm("${ctx}/task/upgrade/cpe/exportUpgradeProgResultForCpe.action",{
					taskId: '${taskInfo.TASK_ID }',
					timeZone: timeZone,
					searchText: vm.params.searchText
				});
			}
		}
	})
})
</script>	