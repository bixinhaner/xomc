<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%-- patch升级任务进度 --%>
<div style="background:#fff;box-sizing:border-box;height:100%;" class="borderChange">	
	<div class="header-title" style="border:none;">
		<h3><%=rb.getString("ZhiXingJieGuo")%></h3>
		<ul class="iconText">
			<li><a class="iconExport iconSize" onclick="exportUpgradeCaProgResult()"><%=rb.getString("DaoChu")%></a></li>
		</ul>
	</div>
	<div style="padding:20px;position:absolute;top:61px;bottom:1px;left:0px;right:0;">
		<table class="easyui-datagrid" id="upgradeCaTaskProg" fit="true" fitColumns="true"
			   data-options="singleSelect:true,idField : 'SERIAL_NUMBER',rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_upgradeCaTask'">
			<thead><tr>
				<th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
				<th data-options="field:'HOST_NAME'" width="100"><%=rb.getString("HostName")%></th>
				<th data-options="field:'PROGRESS_DETAIL'" width="500"><%=rb.getString("JinDu")%></th>
				<th data-options="field:'RUN_TIME'" width="100"><%=rb.getString("ShiJian")%></th>
			</tr></thead>
		</table>
	</div>
</div>
<div id="upgradeCaTaskProgToolsBar">
	<a href="javascript:void(0)" onclick="exportUpgradeCaProgResult()" class="icon-export"></a>
</div>

<form id="upgradeCaProgResult" style="display:none" method="post"
	  action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId"/>
</form>

<div id="toolbar_upgradeCaTask" style=" height: 46px;">
	<input type="text" id="searchText_upgradeCa" class="border-box border"  placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>" style="margin-left: 27px;width:400px;"/>
	<a href="#" class="easyui-linkbutton" style="vertical-align: top; margin-left: 15px;" onclick="queryUpgradeCaTaskInfo()"><%=rb.getString("ChaXun")%></a>
</div>

<script type="text/javascript">
var task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_upgradeCaTaskProgReload;

function exportUpgradeCaProgResult(){
	var url = "${ctx}/task/upgradeCa/exportUpgradeCaProgResult.action";
	/* $("#upgradeCaProgResult").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone;
			param.searchText = $("#searchText_upgradeCa").val();
			var bool = checkParams(param)
			if(!bool) return false;
		}
	}); */
	exportByForm(url,{
		taskId: '${taskInfo.TASK_ID }',
		timeZone: timeZone,
		searchText: $("#searchText_upgradeCa").val()
	});
}

$(function () {
	$("#searchText_upgradeCa").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryUpgradeCaTaskInfo();
		}
	}); 
	
	refreshCaProgress();
	if (timer_upgradeCaTaskProgReload) {
		clearInterval(timer_upgradeCaTaskProgReload);
		timer_upgradeCaTaskProgReload = undefined;
	}
	if (task_progress != '2') {// 未结束，则定时刷新
		clearInterval(timer_upgradeCaTaskProgReload);
		timer_upgradeCaTaskProgReload = setInterval("refreshCaProgress()", 3000);
	}
});
function refreshCaProgress() {
	if ($("#upgradeCaTaskProg").length == 0) {
		clearInterval(timer_upgradeCaTaskProgReload);
		timer_upgradeCaTaskProgReload = undefined;
	}
	
	var param = {};
	param.timeZone=timeZone;
	if ($("#searchText_upgradeCa").val()) {
		param.searchText = $("#searchText_upgradeCa").val();
	}
	
	$.post("${ctx}/task/upgradeCa/getUpgradeCaTaskProgress.action?task_id=${taskInfo.TASK_ID}", param, function(data) {
		$("#upgradeCaTaskProg").datagrid("loadData", data.grid);
		if ("2" == data.pro) {
			if (timer_upgradeCaTaskProgReload) {
				clearInterval(timer_upgradeCaTaskProgReload);
				timer_upgradeCaTaskProgReload = undefined;
				$("#upgradeCaTaskList").datagrid("reload");
			}
		}
	}, "json");
}

function queryUpgradeCaTaskInfo() {
	var param = {};
	param.timeZone=timeZone;
	if ($("#searchText_upgradeCa").val()) {
		param.searchText = $("#searchText_upgradeCa").val();
	}
	$.post("${ctx}/task/upgradeCa/getUpgradeCaTaskProgress.action?task_id=${taskInfo.TASK_ID}", param, function(data) {
		$("#upgradeCaTaskProg").datagrid("loadData", data.grid);
	}, "json");
}
</script>