<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.el-card__body{
		padding-top:0px;
	}
	.queryGroup input{
		width:300px;
	}
</style>
<div id='viewEnbResult'>
	<el-tabs v-model='activeName' style='height:100%;'>
		<el-tab-pane label='eNB' name='enb'>
			<el-ctable id="pciEnbProgressTable" ref="ctable" :url="enbUrl" time=6 :query-params="enbParams" :height="height" page-size=20 pagination="true">
				<template slot="toolbar">
					<div class='queryGroup'>
						<el-input @keyup.enter.native="query" v-model='enbParams.searchText' class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>' style='width:320px;'></el-input>
						<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"  prop="SERIAL_NUMBER"></el-table-column>
				<el-table-column label='<%=rb.getString("HostName")%>' width="200"  prop="HOST_NAME"></el-table-column>
				<el-table-column label='<%=rb.getString("FrequencyXiuGaiQian")%>' width="200" prop="OLD_EARFCN"></el-table-column>
				<el-table-column label='<%=rb.getString("FrequencyXiuGaiHou")%>' width="200" prop="EARFCN"></el-table-column>
				<el-table-column label='<%=rb.getString("PCIXiuGaiQian")%>' width=200" prop="OLD_PCI" ></el-table-column>
				<el-table-column label='<%=rb.getString("PCIXiuGaiHou")%>' width="200" prop="PCI"></el-table-column>
				<el-table-column label='<%=rb.getString("XiuGaiBangDingCPE")%>' width="180" prop="CPE_FLAG" :formatter='cpeFlagFmt'></el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="PROGRESS_STATUS">
					<template slot-scope="scope">
						<div v-html="resultTableStatus(scope.row.PROGRESS_STATUS)"></div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("JieGuo")%>' prop="PROGRESS_RESULT" :formatter='resultFmt'></el-table-column>
				<el-table-column label='<%=rb.getString("PCILOCKShiBaiYuanYin")%>' prop="FAILURE_REASON" width='200'></el-table-column>
			</el-ctable>
		</el-tab-pane>
		<el-tab-pane label='CPE' name='cpe'>
			<el-ctable id='pciCpeProgressTable' ref="ctableCpe" time=6  :url='cpeUrl' :query-params="cpeParams" :height="height" page-size=20 pagination="true">
				<template slot="toolbar">
					<div class='queryGroup'>
						<el-input @keyup.enter.native="queryCpe" v-model='cpeParams.searchText' class='pairgrid-query' placeholder='<%=rb.getString("CPEBianMa")%>' style='width:320px;'></el-input>
						<i @click='queryCpe' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column label='<%=rb.getString("CPEBianMa")%>' width="200"  prop="serial_number"></el-table-column>
				<el-table-column label='<%=rb.getString("CPEName")%>' width="200"  prop="cpe_name"></el-table-column>
				<el-table-column label='IMSI' width="200"  prop="imsi"></el-table-column>
				<el-table-column label='<%=rb.getString("FrequencyXiuGaiQian")%>' width="200" prop="cpe_earfcn_before"></el-table-column>
				<el-table-column label='<%=rb.getString("FrequencyXiuGaiHou")%>' width="200" prop="cpe_earfcn_after"></el-table-column>
				<el-table-column label='<%=rb.getString("PCIXiuGaiQian")%>' width=200" prop="cpe_pci_before" ></el-table-column>
				<el-table-column label='<%=rb.getString("PCIXiuGaiHou")%>' width="200" prop="cpe_pci_after"></el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="progress_status">
					<template slot-scope="scope">
						<div v-html="resultTableStatus(scope.row.progress_status)"></div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("JieGuo")%>' prop="progress_result" :formatter='resultFmt'></el-table-column>
				<el-table-column label='<%=rb.getString("PCILOCKShiBaiYuanYin")%>' prop="failure_reason" width='200'></el-table-column>
			</el-ctable>
		</el-tab-pane>
	</el-tabs>
</div>
<form id="enbProgResult" v-show='false' method="post" action=""></form>
<form id="cpeProgResult" v-show='false' method="post" action=""></form>
<script>
	new Vue({
		el:'#viewEnbResult',
		data(){
			return{
				activeName:'enb',
				enbParams:{
					timeZone:timeZone,
					searchText:''
				},
				cpeParams:{
					timeZone:timeZone,
					searchText:''
				},
				cpeUrl:'${ctx}/task/pcilock/getPciLockCPETaskListProgress.action?taskId=${taskInfo.TASK_ID}',
				height:'100%',
				enbUrl:'${ctx}/task/pcilock/getPciLockTaskProgress.action?task_id=${taskInfo.TASK_ID}'
			}
		},
		methods:{
			// 搜索eNb 
			query(){
				this.$refs.ctable.refresh();
			},
			// 搜索Cpe
			queryCpe(){
				this.$refs.ctableCpe.refresh();
			},
			// 导出执行 结果的表格数据
			exportResult(){
		    	var vm = this;
		    	if(vm.activeName == 'enb'){
		    		exportByForm("${ctx}/task/pcilock/exportPciLockProgResultToCSV.action",{
		    			timeZone: timeZone,
		    			taskId: "${taskInfo.TASK_ID}",
		    			searchText: vm.enbParams.searchText
		    		});
		    	}else{
		    		exportByForm("${ctx}/task/pcilock/exportPciLockCPETaskListProgress.action",{
		    			timeZone: timeZone,
		    			taskId: "${taskInfo.TASK_ID}",
		    			searchText: vm.cpeParams.searchText
		    		});
		    	}
			},
			/**
			 * 数据结果的修改转换
			 * @parame row:表格数据
			 * @parame column:表格dom元素
			 * @parame value: 结果数据 进行转换
			 * @parame index: 下标
			*/
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
			/**
			 * 数据cpe状态的修改转换
			 * @parame row:表格数据
			 * @parame column:表格dom元素
			 * @parame value: cpe状态数据 进行转换
			 * @parame index: 下标
			*/
			cpeFlagFmt(row,column,value,index){
				if (value == "0") {
					return "<%=rb.getString("BuKeYong")%>";
				} else if (value == "1") {
					return "<%=rb.getString("KeYong")%>";
				}
			}
		},
		mounted(){
			eventBus.$off('export-enb').$on('export-enb',this.exportResult);
		}
	})
</script>