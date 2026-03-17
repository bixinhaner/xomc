<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<div id='license_result'>
	<el-ctable ref="ctable" time=6  :url="url" :query-params="params" :height="height" pagination="true">
		<template slot="toolbar" style="position: relative;display:flex;align-items: center;"> 
            <span style="font-size:14px;;margin: 0px 20px;font-weight:bold;"><%=rb.getString("JieGuo")%></span>
			<div class="queryGroup">
				<el-input v-model="params.search_text" @keyup.enter.native="query" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
				<i @click="query" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
			</div>
            <div class="newIconBoxCls-bt" @click="exportTaskResult" style="right:60px;top:10px;" tip="<%=rb.getString("DaoChu")%>">
                <span class='el-icon el-icon-operation-export'></span>
            </div>
            <div class="newIconBoxCls-bt" @click="close1588LicenseTaskResultSlide" style="right:20px;top:10px;" tip="<%=rb.getString("GuanBi")%>">
                <span class='el-icon el-icon-close'></span>
            </div>
		</template>
		<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"  prop="serial_number"></el-table-column>
		<el-table-column label='<%=rb.getString("HostName")%>' width="150" prop="host_name"></el-table-column>
		<el-table-column label='<%=rb.getString("MACDiZhi")%>' width="200" prop="mac_address"></el-table-column>
		<el-table-column label='<%=rb.getString("ZhuangTai")%>' width=200" prop="progress_status">
			<template slot-scope="scope">
				<div v-if="scope.row.progress_status == 'Waiting'">
					<span class='el-icon el-icon-status-waiting1'></span><%=rb.getString("DengDai")%>
				</div>
				<div v-if="scope.row.progress_status == 'In Progress'">
					<span class='el-icon el-icon-status-inProgress'></span><%=rb.getString("JinXingZhong")%>
				</div>
				<div v-if="scope.row.progress_status == 'Suspend'">
					<span class='el-icon el-icon-status-suspend'></span><%=rb.getString("ZanTing")%>
				</div>
				<div v-if="scope.row.progress_status == 'End'">
					<span class='el-icon el-icon-status-terminate'></span><%=rb.getString("YiJieShu")%>
				</div>
				<div v-if="scope.row.progress_status == 'Termination'">
					<span class='el-icon el-icon-status-terminate'></span><%=rb.getString("ZhongZhi")%>
				</div>
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("JieGuo")%>' width="200" prop="progress_result" :formatter='resultFmt'></el-table-column>
		<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' width="180" prop="failure_reason"></el-table-column>
		<el-table-column label='<%=rb.getString("ShiJian")%>' prop="run_time"></el-table-column>
	</el-ctable>
</div>
<form id="licensetTaskResult" v-show='false' method="post" action=""></form>
<script>
	new Vue({
		el:'#license_result',
		data(){
			return{
				url:'',
				height:'99%',
				params:{
					search_text:'',
					time_zone:timeZone
				},
				task_id:''
			}
		},
		methods:{
			query(){
				this.$refs.ctable.refresh();
			},
			getResult(task_id){
				this.task_id = task_id;
				this.url = '${ctx}/cell/1588License/getCellRecordList.action?task_id=' + task_id;
				this.$refs.ctable.refresh();
			},
			exportTaskResult(){
				var vm = this;
		    	/* $("#licensetTaskResult").form('submit', {
		    		url: '${ctx}/cell/1588License/exportRecord.action',
		    		onSubmit: function(param) {
		    			param.time_zone=timeZone;
		    			param.task_id=vm.task_id;
		    			param.search_text = vm.params.search_text
		    		}
		    	}); */
		    	exportByForm('${ctx}/cell/1588License/exportRecord.action',{
		    		time_zone: timeZone,
		    		task_id: vm.task_id,
		    		search_text: vm.params.search_text
		    	});
			},
			// 格式化
			resultFmt(row,column,cellValue,index){
				var resultObj = {
    					"Success" : "<%=rb.getString("ChengGong")%>",
    					"Partial Success" : "<%=rb.getString("BuFenChengGong")%>",
    					"Fail" : "<%=rb.getString("ShiBai")%>",
    					"Unexecuted" : "<%=rb.getString("WeiZhiXing")%>",
						"In Progress" : "<%=rb.getString("JinXingZhong")%>",
    					"" : ""
    			}
    			return resultObj[cellValue];
			},
            close1588LicenseTaskResultSlide(){
                eventBus.$emit('hide-edit')
            }
		},
		mounted(){
			eventBus.$off('view-result').$on('view-result',this.getResult)
			eventBus.$off('export-result').$on('export-result',this.exportTaskResult)
		}
	})
</script>