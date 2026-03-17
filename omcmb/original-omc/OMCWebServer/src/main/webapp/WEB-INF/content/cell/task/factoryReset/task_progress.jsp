<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 重启任务进度 --%>
<div class="slideDiv" style="height:300px;box-shadow:0px -5px 10px rgba(0,0,0,0.1)" >
	<div class="slideHeader" style="border:none;margin-bottom:0;">
		<h3><%=rb.getString("JieGuo")%></h3>
		<ul class="iconText">
			<li><span class="hoverShow" style='color:#4D84FF'><%=rb.getString("DaoChu")%></span><a class="el-icon el-icon-operation-export" style="font-size:18px;margin-right:15px;" onclick="exportrRebootProgResult()"></a></li>
			<li><a class="el-icon el-icon-close" style='font-size:18px;' onclick="closeSlideDiv()"></a></li>
		</ul>
	</div>
	<div style="position:absolute;top:51px;bottom:10px;left:0px;right:0;overflow:auto;">
		<table class="easyui-datagrid" id="resetTaskProg" fit="true" fitColumns="true"
			   data-options="singleSelect:true,idField : 'SERIAL_NUMBER',rownumbers:true,border:false,striped:true,pagination:true,onBeforeLoad:beforeLoadFactoryReset,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_resetTask',url:'${ctx}/task/factoryReset/getFactoryResetTaskProgress.action?task_id=${taskInfo.TASK_ID}'">
			<thead>
				<tr>
					<th data-options="field:'ID'" width="5" hidden="true"></th>
					<th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
					<th data-options="field:'HOST_NAME'" width="120"><%=rb.getString("HostName")%></th>
					<th data-options="field:'PROGRESS_STATUS',formatter:resultTableStatus" width="120"><%=rb.getString("ZhuangTai")%></th>
					<th data-options="field:'PROGRESS_RESULT',formatter:resultTableResult" width="120"><%=rb.getString("JieGuo")%></th>
					<th data-options="field:'FAILURE_REASON'" width="120"><%=rb.getString("PCILOCKShiBaiYuanYin")%></th>
					<th data-options="field:'RUN_TIME'" width="120"><%=rb.getString("ShiJian")%></th>
				</tr>
			</thead>
		</table>
	</div>
</div>

<form id="resetProgResult" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId"/>
</form>

<%-- 重启任务进度工具栏 --%>
<div id="toolbar_resetTask" style="padding-bottom:15px;">
	<div class='queryGroup'>
		<input id="searchText_reset" placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>" />
		<b class='el-icon el-icon-common-search' onclick="queryRebootTaskInfo()"></b>	
	</div>
</div>

<script type="text/javascript">
var resultSearchText = "";
var reboot_task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_factoryreset;

function exportrRebootProgResult(){
	var url = "${ctx}/task/factoryReset/exportFactoryResetProgResultToCSV.action";
	/* $("#resetProgResult").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone;
			param.searchText = $("#searchText_reset").val();
		}
	}); */
	exportByForm(url,{
		taskId: '${taskInfo.TASK_ID }',
		timeZone: timeZone,
		searchText: $("#searchText_reset").val()
	});
}

$(function () {
	//周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_factoryreset);
	timer_factoryreset = undefined;
	timer_factoryreset = setInterval(function(){
		updateTable($("#resetTaskProg"));
		var dom = $("#resetTaskProg");
		if(dom.length == 0) clearInterval(timer_factoryreset);
	},6000)
	
	//结果列表的搜索框回车事件
	$("#searchText_reset").bind("keyup", function(e){
		if (e.keyCode == 13){
			$("#resetTaskProg").datagrid("reload");
		}
	});
	
	$(".slideHeader .el-icon-operation-export").hover(function(){
		$(".hoverShow").fadeIn();
	},function (){
		$(".hoverShow").fadeOut();
	})
});

function closeSlideDiv(){
	//$(".slideDiv").animate({bottom:'-400px'},400);
	$("#resetTaskProgress").hide(400);
	clearInterval(timer_factoryreset);
}
function beforeLoadFactoryReset(param){
	param.timeZone = timeZone;
	param.searchText = $("#searchText_reset").val()
}
</script>