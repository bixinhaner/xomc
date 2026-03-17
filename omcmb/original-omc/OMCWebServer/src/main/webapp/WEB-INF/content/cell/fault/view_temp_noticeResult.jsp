<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<style>
	.stateClose [class*=" el-icon-circle"], .stateSuccess [class*=" el-icon-circle"] {
		font-size: 16px!important;
	}
	.stateClose .el-icon:before{
		font-size: 16px;
		color: #E88282;
	}
	.stateSuccess .el-icon:before{
		font-size: 16px;
		color: #67D972;
	}
</style>

<div id="noticeResultInfoPage" >
	<!--告警通知结果  -->
	<el-ctable id="noticeResultInfo" ref="noticeResultInfo" :url="noticeResultURL" :page-size="pageSize" :page-list="pageList" :rownumber="false" :pagination="true" :query-params="queryParams">
		<!--搜索工具栏 -->
		<template slot="toolbar">
			<div class="queryGroup">
				<input v-model="queryForm.searchText" placeholder="<%=rb.getString("YouJianBiaoTi")%>" />
				<b class="el-icon el-icon-common-search" @click="query"></b>
			</div>
		</template>
		
		<el-table-column label='<%=rb.getString("YouJianBiaoTi")%>'   prop="subject"></el-table-column>
		<el-table-column label='<%=rb.getString("YouJianJiShouRen")%>'  prop="email_address"></el-table-column>
		<el-table-column label='<%=rb.getString("YouJianFaSongShiJian")%>' prop="send_time"></el-table-column>
		<el-table-column label='<%=rb.getString("JieGuo")%>'>
			<template slot-scope="scope">
				<div v-if="scope.row.result == '0'" class="stateClose">
					<span class="el-icon el-icon-circle-close" style="padding-right:5px;"></span><%=rb.getString("ShiBai")%>
				</div>
				<div v-else-if="scope.row.result == '1'" class="stateSuccess">
					<span class="el-icon el-icon-circle-success" style="padding-right:5px;"></span><%=rb.getString("ChengGong")%>
				</div>
				<div v-else-if="scope.row.result == '2'" class="stateSuccess">
					<span class="el-icon el-icon-circle-success" style="padding-right:5px;"></span><%=rb.getString("BuFenChengGong")%>
				</div>
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>'  prop="failure_reason" show-overflow-tooltip></el-table-column>
	</el-ctable>
</div>
<script type="text/javascript">
new Vue({
	el:"#noticeResultInfoPage",
	data(){
		return{
			noticeResultURL:'',
			queryForm: {
				searchText:'',
			},
			pageSize:50,
			pageList:[50,100,200],
			queryParams: {
				searchText:'',
				timeZone: timeZone
			},
		}
	},
	methods:{
		init(templateId){
			var vm = this;
			vm.noticeResultURL = '${ctx}/cell/fault/queryEmailAlarmRecordsPageList.action?timeZone='+timeZone+"&templateId="+templateId
		},
		// 搜索功能
		query(){
			var vm = this;
			// es6 Object.assign(),用于对象的合并，将源对象的所有可枚举属性，复制到目标对象，第一个参数是目标对象，后面参数为源对象，注意：模板对象与源对象有同名属性，后面属性覆盖前面的属性
			Object.assign(vm.queryParams, vm.queryForm);
		},
	},
	mounted(){
		eventBus.$off('result-info').$on('result-info',this.init);
	
	}
	
})
</script>