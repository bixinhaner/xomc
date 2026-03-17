<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.slide-content{
		border:none;
	}
	.queryGroup input{
		width:auto;
	}
</style>
<div id='viewCpeResult'>
	<el-ctable id="pciProgressTableCpe" ref="ctable" time=6 :url="url" :query-params="cpeParams" :height="height" page-size=20 pagination="true">
		<template slot="toolbar">
			<div class='queryGroup'>
				<el-input @keyup.enter.native="query" style='width:320px;' v-model='cpeParams.searchText' class='pairgrid-query' placeholder='<%=rb.getString("CPEBianMa")%>'></el-input>
				<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
			</div>
		</template>
		<el-table-column label='<%=rb.getString("CPEBianMa")%>' width="200"  prop="SERIAL_NUMBER"></el-table-column>
		<el-table-column label='<%=rb.getString("CpeName")%>' width="200" prop="CPE_NAME"></el-table-column>
		<el-table-column label='IMSI' width="200" prop="IMSI"></el-table-column>
		<el-table-column label='<%=rb.getString("PinDian")%>' width="200" prop="EARFCN" :formatter='earfcnFmt'></el-table-column>
		<el-table-column label='PCI' width="200" prop="PCI" :formatter='pciFmt'></el-table-column>
		<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="PROGRESS_STATUS">
			<template slot-scope="scope">
				<div v-html="taskTableStatus(scope.row.PROGRESS_STATUS)"></div>
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("JieGuo")%>' prop="PROGRESS_RESULT" :formatter='resultFmt'></el-table-column>
		<el-table-column label='<%=rb.getString("PCILOCKShiBaiYuanYin")%>' prop="FAILURE_REASON" width='200'></el-table-column>
	</el-ctable>
</div>
<form id="progResultForm" v-show='false' method="post" action=""></form>
<script>
	new Vue({
		el:'#viewCpeResult',
		data(){
			return{
				cpeParams:{
					timeZone:timeZone,
					searchText:''
				},
				height:'100%',
				url:'${ctx}/cpe/strategy/getPciLockTaskProgress.action?task_id=${taskInfo.TASK_ID}'
			}
		},
		methods:{
			query(){
				this.$refs.ctable.refresh();
			},
			exportResult(){
		    	var vm = this;
	    		exportByForm("${ctx}/cpe/strategy/exportPciLockProgResultToCSV.action",{
	    			timeZone: timeZone,
	    			taskId: "${taskInfo.TASK_ID}",
	    			searchText: vm.cpeParams.searchText
	    		});
			},
			resultFmt(row,column,value,index){
				if (value == "3") {
					return ShiBai;
				} else if (value == "1") {
					return ChengGong;
				} else if (value == "2") {
					return ZhongZhi;
				}else {
					return "";
				}
			},
			earfcnFmt(row,column,value,index){
				if(row.EARFCN == null){
					return "--";
				}else{
					return row.EARFCN;
				} 
			},
			pciFmt(row,column,value,index){
				if(row.PCI == null){
					return "--";
				}else{
					return row.PCI;
				}
			}
		},
		mounted(){
			eventBus.$off('export-cpe').$on('export-cpe',this.exportResult);
		}
	})
</script>