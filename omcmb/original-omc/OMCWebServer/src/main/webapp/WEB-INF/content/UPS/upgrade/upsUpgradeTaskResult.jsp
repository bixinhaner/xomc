<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.container .cmenu{
		z-index: 361!important;
	}
	#upsUpgradeSlide{
		z-index: 2002!important;
	}
	#upsUpgradeResuitSlide .el-icon-close{
		top:0px!important;
		font-size: 18px!important;
	}
</style>
<div class="pageDefault" id='upsUpgradeTaskResult'>
	<div class="container">
        <!-- 表格组件 -->
		<el-ctable
			:url="upsUpgradeTableUrl" 
			:query-params="params" 
			ref="upsUpgradeTable" 
			id="upsUpgradeTable"
			:height="height" 
			time="6" 
			:page-size="pageSize" 
			:page-list="pageList" 
			pagination="true">
				<!-- 列表toolbar -->
			<template slot="toolbar">
				<el-query type="normal" @query="queryUpsUpgradeFile" placeholder="<%=rb.getString("BanBen")%>"></el-query>
			</template>
				<!-- 列表columns -->
			<el-table-column label='<%=rb.getString("DianYuanBianMa")%>' min-width="100"  prop="SERIAL_NUMBER"></el-table-column>
			<el-table-column label='<%=rb.getString("ChuShiBanBen")%>' min-width="150" prop="ORI_VERSION"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="120" prop="PROGRESS_STATUS">
				<template slot-scope="scope">
					<div v-html="resultTableStatus(scope.row.PROGRESS_STATUS)"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("JieGuo")%>' min-width="120" prop="PROGRESS_RESULT">
				<template slot-scope="scope">
					<div v-html="resultTableResult(scope.row.PROGRESS_RESULT)"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("PCILOCKShiBaiYuanYin")%>' min-width="120" prop="FAILURE_REASON"></el-table-column>
			<el-table-column label='<%=rb.getString("ShiJian")%>' min-width="100" prop="RUN_TIME"></el-table-column>
		</el-ctable>
    </div>
</div>
<form id="upsUpgradeTaskResultExport" v-show='false' method="post" action=""></form>
<script type="text/javascript">
new Vue({
	el:'#upsUpgradeTaskResult',
	data(){
		return {
            params:{
                timeZone:timeZone,
                searchText:'',
            },
			
            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
            upsUpgradeTableUrl:'',

		}
	},
	methods:{
		init(id){
			var vm = this;
			vm.upsUpgradeTableUrl = '${ctx}/task/upgrade/ups/getUpgradeTaskProgress.action?task_id='+id;
		},
        // UPS升级文件搜索事件
        queryUpsUpgradeFile(val){
            var vm = this;
            vm.params.searchText = val;
        },
		// 导出升级任务结果
		exportResult(task_id,type){
	    	var vm = this;
	    	
	    	exportByForm('${ctx}/task/upgrade/ups/exportUpgradeProgResult.action',{
	    		timeZone: timeZone,
	    		type: type,
	    		taskId: task_id,
	    		searchText_upgrade: vm.params.searchText,
	    	})
	    }
	},
	mounted(){
		eventBus.$off('ups-upgradeTaskResult-init').$on('ups-upgradeTaskResult-init',this.init);
		eventBus.$off('export-upsUpgrade-taskResult').$on('export-upsUpgrade-taskResult',this.exportResult);
	}
	
})

</script> 
