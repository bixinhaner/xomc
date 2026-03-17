<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
.el-card__body{
	padding:0px;
	/* padding:20px 20px 0px 20px; */
}
.queryGroup input{
	width:300px;
}
.slide-content{
	border:none;
}
</style>
<div id='reboot_progress'>
	<el-ctable id="rebootProgressTable" ref="ctable" :url="url" :query-params="params" :height="height" time="6">
		<template slot="toolbar">
			<div class='queryGroup'>
				<el-input @keyup.enter.native="query" v-model='params.searchText' class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>' style='width:320px;'></el-input>
				<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;cursor:pointer;"></i>
			</div>
		</template>
		<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' width="220"  prop="SERIAL_NUMBER"></el-table-column>
		<el-table-column label='<%=rb.getString("HostName")%>' width="220" prop="HOST_NAME"></el-table-column>
		<el-table-column v-if='showEu' label='<%=rb.getString("LuYouSuoYin") %>' width="150" prop="ROUTE_INDEX"></el-table-column>
		<el-table-column v-if='showRu' label='<%=rb.getString("XuLieHao") %>' width="150" prop="INDEX"></el-table-column>
		<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="150" prop="PROGRESS_STATUS">
			<template slot-scope="scope">
				<div v-html="resultTableStatus(scope.row.PROGRESS_STATUS)"></div>
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("JieGuo")%>' width="150" prop="PROGRESS_RESULT" :formatter='resultFmt'></el-table-column>
		<el-table-column v-if='showReason' label='<%=rb.getString("PCILOCKShiBaiYuanYin")%>'  prop="FAILURE_REASON" width="200"></el-table-column>
		<el-table-column label='<%=rb.getString("ShiJian")%>' prop="RUN_TIME"></el-table-column>
	</el-ctable>
</div>
<form id="rebootTaskProgResult" v-show='false' method="post" action=""></form>
<script>
var timer;
new Vue({
	el:'#reboot_progress',
	data(){
		return{
			url:'${ctx}/task/reboot/getRebootTaskProgress.action?task_id=${taskInfo.TASK_ID}',
			params:{
				searchText:'',
				timeZone:timeZone,
				productType:vmReboot.rowData.PRODUCT_TYPE!=undefined&&vmReboot.rowData.PRODUCT_TYPE.indexOf('CR-B4860') > -1?vmReboot.rowData.PRODUCT_TYPE:''
			},
			height:'99%',
			showEu:false,
			showRu:false,
			showReason:true
		}
	},
	methods:{
		init(){
			var product = vmReboot.rowData.PRODUCT_TYPE;
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
	    	/* $("#rebootTaskProgResult").form('submit', {
	    		url: '${ctx}/task/reboot/exportRebootProgResultToCSV.action',
	    		onSubmit: function(param) {
	    			param.timeZone=timeZone;
	    			param.taskId=task_id;
	    			param.searchText = vm.params.searchText;
	    			if(vmReboot.rowData.PRODUCT_TYPE!=undefined && vmReboot.rowData.PRODUCT_TYPE.indexOf('CR-B4860') > -1){
	    				param.productType = vmReboot.rowData.PRODUCT_TYPE
	    			}
	    		}
	    	}); */
	    	exportByForm('${ctx}/task/reboot/exportRebootProgResultToCSV.action',{
	    		taskId: task_id,
	    		timeZone: timeZone,
	    		searchText: vm.params.searchText
	    	});
	    }
	},
	mounted(){
		this.init();
		eventBus.$off('export-result').$on('export-result',this.exportResult);
	}
	
})
</script>