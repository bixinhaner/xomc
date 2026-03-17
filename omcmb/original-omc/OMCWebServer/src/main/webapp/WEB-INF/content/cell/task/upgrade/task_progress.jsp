<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
.el-card__body{
	padding:0px;
	padding:20px 20px 0px 20px;
}
.queryGroup input{
	width:320px;
}
.el-card__body{
	padding:0px;
}
.slide-content{
	border:none;
}
</style>
<div id='upgrade_progress'>
	<el-ctable ref="ctable" id="upgradeProgressTable" time="6" :url="progressUrl" :query-params="params" :height="height">
		<template slot="toolbar">
			<div class='queryGroup'>
				<el-input @keyup.enter.native="query" v-model='params.searchText' class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>' style='width:320px;'></el-input>
				<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
			</div>
		</template>
		<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"  prop="SERIAL_NUMBER"></el-table-column>
		<el-table-column label='<%=rb.getString("HostName")%>' width="150" prop="HOST_NAME"></el-table-column>
		<el-table-column v-if='showEu' label='<%=rb.getString("LuYouSuoYin") %>' width="150" prop="ROUTE_INDEX"></el-table-column>
		<el-table-column v-if='showRu' label='<%=rb.getString("XuLieHao") %>' width="150" prop="INDEX"></el-table-column>
		<el-table-column label='<%=rb.getString("ChuShiBanBen")%>' width="200" prop="ORI_VERSION"></el-table-column>
		<el-table-column label='<%=rb.getString("ZhuangTai")%>' width=200" prop="PROGRESS_STATUS">
			<template slot-scope="scope">
				<div v-html="resultTableStatus(scope.row.PROGRESS_STATUS)"></div>
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("JieGuo")%>' width="200" prop="PROGRESS_RESULT" :formatter='resultFmt'></el-table-column>
		<el-table-column v-if='showReason' label='<%=rb.getString("PCILOCKShiBaiYuanYin")%>' width="180" prop="FAILURE_REASON"></el-table-column>
		<el-table-column label='<%=rb.getString("ShiJian")%>' prop="RUN_TIME"></el-table-column>
	</el-ctable>
</div>
<form id="upgradTaskProgResult" v-show='false' method="post" action=""></form>
<script>
var timer;
new Vue({
	el:'#upgrade_progress',
	data(){
		return{
			progressUrl:'${ctx}/task/upgrade/getUpgradeTaskProgress.action?taskId=${taskInfo.TASK_ID}',
			params:{
				searchText:'',
				timeZone:timeZone,
				productType:enbSoftUpgradeVM.rowData.PRODUCT.indexOf('CR-B4860') > -1?enbSoftUpgradeVM.rowData.PRODUCT:''
			},
			height:'100%',
			showEu:false,
			showRu:false,
			showReason:true
		}
	},
	methods:{
		init(){
			var product = enbSoftUpgradeVM.rowData.PRODUCT;
			if(product == 'CR-B4860/EU'){
				this.showEu = true;
				this.showReason = false;
			}else if(product == 'CR-B4860/RU'){
				this.showEu = true;
				this.showRu = true;
				this.showReason = false;
			}else{
				this.showEu = false;
				this.showRu = false;
				this.showReason = true;
			}
		},
	    resultFmt(row,column,cellValue,index){
	    	var code = {
	    			1 : '<%=rb.getString("ChengGong")%>',
	    			2 : '<%=rb.getString("ZhongZhi")%>',
	    			3 : '<%=rb.getString("ShiBai")%>'
	    	}
	    	if(code[cellValue] != undefined){
	    		return code[cellValue]
	    	}else{
	    		return ""
	    	}
	    },
	    cancel(){
	    	eventBus.$emit('cancel-view')
	    },
	    query(){
	    	this.$refs.ctable.refresh();
	    },
	    exportResult(task_id,type){
	    	var vm = this;
	    	/* $("#upgradTaskProgResult").form('submit', {
	    		url: '${ctx}/task/upgrade/exportUpgradeProgResultaToCSV.action',
	    		onSubmit: function(param) {
	    			param.timeZone=timeZone;
	    			param.type=type;
	    			param.taskId=task_id;
	    			param.searchText = vm.params.searchText
	    			if(enbSoftUpgradeVM.rowData.PRODUCT.indexOf('CR-B4860') > -1){
	    				param.product = enbSoftUpgradeVM.rowData.PRODUCT
	    			}
	    		}
	    	}); */
	    	exportByForm('${ctx}/task/upgrade/exportUpgradeProgResultaToCSV.action',{
	    		timeZone: timeZone,
	    		type: type,
	    		taskId: task_id,
	    		searchText: vm.params.searchText,
	    		product: enbSoftUpgradeVM.rowData.PRODUCT.indexOf('CR-B4860') > -1 ? enbSoftUpgradeVM.rowData.PRODUCT:''
	    	})
	    }
	},
	mounted(){
		this.init();
		eventBus.$off('export-result').$on('export-result',this.exportResult);
	}
})
</script>