<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<div id="selfConfigTaskDiv">
	<el-ctable id="exeProgressTable" ref="ctableProgress" :url="taskUrl" :query-params="params_progress" time="6"
			:row-key="'id'" :height="height" pagination="false" rownumber="true">
				
		<el-table-column label="<%=rb.getString("JinDu")%>" prop="progress" :formatter="progressFmt"></el-table-column>
		<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
			<template slot-scope="scope">
				<div v-if="scope.row.status == '0'">
					<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span><%=rb.getString("DengDai")%>
				</div>
				<div v-if="scope.row.status == '1'">
					<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span><%=rb.getString("JinXingZhong")%>
				</div>
				<div v-if="scope.row.status == '2'">
					<span class='el-icon el-icon-status-terminate' style='margin-right:5px;'></span><%=rb.getString("YiJieShu")%>
				</div>
			</template>
		</el-table-column>
		<el-table-column label="<%=rb.getString("JieGuo")%>" prop="result" :formatter="resultViewFmt"></el-table-column>
		<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="start_time"></el-table-column>
		<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="end_time"></el-table-column>
		<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failure_reason"></el-table-column>
	</el-ctable>
</div>
<script>
	var selfTaskVue = new Vue({
		el:"#selfConfigTaskDiv",
		data(){
			return{
				taskUrl:"${ctx}/SON/SelfConfigurationTask/getSelfConfigTaskRecordPageList.action",
				params_progress:{
					timeZone:timeZone,
					task_id:selfVue.rowDataExe.task_id
				},
				height:"100%"
			}
		},
		methods:{
			progressFmt(row,column,value,index){
				if(value == 1){
					return "<%=rb.getString("RuanJianShengJi")%>"
				}else if(value == 2){
					return "<%=rb.getString("LicenseXiaFa")%>"
				}else if(value == 3){
					return "<%=rb.getString("CanShuZiPeiZhi")%>"
				}else if(value == 4){
					return "Cell active"
				}
			},
			resultViewFmt(row,column,value,index){
				if(value == 0){
					return "<%=rb.getString("ChengGong")%>"
				}else if(value == 1){
					return "<%=rb.getString("ShiBai")%>"
				}
			}
		}
	})
</script>
