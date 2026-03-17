<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 基站-策略-升级策略 软件系统升级 升级任务进度（执行结果） --%>	
<div class="slideHeader" style="border:none;">
	<h3><%=rb.getString("ZhiXingJieGuo")%></h3>
	<ul class="iconText">
		<li><span class="hoverShow"><%=rb.getString("DaoChu")%></span><a style='font-size:18px;' class="el-icon el-icon-operation-export slideExport iconSize" style="margin-right:0px;vertical-align: middle;" onclick="exportUpgradeProgResult()"></a></li>
		<li><a style='font-size:18px;' class="el-icon el-icon-close" onclick="closeSlideDiv()"></a></li>
	</ul>
</div>
<div  style="padding: 0 20px;position:absolute;top:40px;bottom:1px;left:0px;right:0;overflow:auto;">
	<table class="easyui-datagrid" id="egwUpgradeTaskProTable" fit="true"
		   data-options="border:false,
		   singleSelect:true,
		   rownumbers:true,
		   fitColumns:true,
		   striped:true,
		   onLoadError:datagridLoadError,
		   onLoadSuccess:datagridLoadSuccess,
		   toolbar:'#toolbar_egwUpgradeTaskResult'">
		<thead>
			<tr>
				<th data-options="field:'gw_ip'" width="100"><%=rb.getString("eGWIP")%></th>
				<th data-options="field:'gw_name'" width="200"><%=rb.getString("EGWMingCheng")%></th>
				<th data-options="field:'original_version'" width="200"><%=rb.getString("ChuShiBanBen")%></th>
				<th data-options="field:'progress'" width="200"><%=rb.getString("JinDu")%></th>
				<th data-options="field:'time'" width="200"><%=rb.getString("ShiJian")%></th>
			</tr>
		</thead>
	</table>
</div>	

<form id="egwUpgradeTaskProResult" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="egwTaskId"/>
</form>

<!-- 升级任务结果工具栏 -->
<div id="toolbar_egwUpgradeTaskResult" class="toolbarContainer defaultQuery">
	<div class='queryGroup'>
		<input id="searchText_egwTaskResult"  style="margin-left:0;" placeholder="<%=rb.getString("eGWIP")%>/<%=rb.getString("EGWMingCheng")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		<b  class="el-icon el-icon-common-search" onclick="queryUpgradeTaskInfo()" ></b>
	</div>
</div>

<script type="text/javascript">
var selectTask = $("#egwUpgradeListTable").datagrid("getSelected")
var task_status = selectTask["task_status"];
var timer_egwUpgradeTaskProReload;

function exportUpgradeProgResult(){
	var selectedTask = $("#egwUpgradeListTable").datagrid("getSelected");
	var task_id =  selectedTask["id"];
	var url = "${ctx}/egw/softwareUpgrade/downloadSoftwareExeResList.action";
	/* $("#egwUpgradeTaskProResult").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone;
			param.taskId=task_id;
			param.searchText = $("#searchText_egwTaskResult").val();
			var bool = checkParams(param)
			if(!bool) return false;
		}
	}); */
	exportByForm(url,{
		egwTaskId: '${taskInfo.TASK_ID }',
		timeZone: timeZone,
		taskId: task_id,
		searchText: $("#searchText_egwTaskResult").val()
	});
}

$(function () {
	$("#searchText_egwTaskResult").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryUpgradeTaskInfo();
		}
	}); 
	
	refreshProgress();
	
	if (timer_egwUpgradeTaskProReload) {
		clearInterval(timer_egwUpgradeTaskProReload);
		timer_egwUpgradeTaskProReload = undefined;
	}
	if (task_status != 2) {// 未结束，则定时刷新
		clearInterval(timer_egwUpgradeTaskProReload);
		timer_egwUpgradeTaskProReload = setInterval("refreshProgress()", 3000);
	}
	
	$(".slideHeader .el-icon-operation-export").hover(function(){
		$(".hoverShow").fadeIn();
	},function (){
		$(".hoverShow").fadeOut();
	})
});

/* 关闭任务结果信息 */
function closeSlideDiv(){
	$("#egw_task_result").animate({bottom:'-350px'},400);
}

function refreshProgress() {
	if ($("#egwUpgradeTaskProTable").length == 0) {
		clearInterval(timer_egwUpgradeTaskProReload);
		timer_egwUpgradeTaskProReload = undefined;
	}
	var selectedTask = $("#egwUpgradeListTable").datagrid("getSelected");
	var taskId =  selectedTask["id"];
	var param = {};
	param.taskId=taskId;
	param.timeZone=timeZone;
	param.searchText = $("#searchText_egwTaskResult").val();
	$.post("${ctx}/egw/softwareUpgrade/querySoftwareExeResPageList.action", param, function(data) {
		if (data){
			$("#egwUpgradeTaskProTable").datagrid("loadData", data);
	        var status= selectedTask["task_status"];
	        if (2 == status) {
	            // task is end
	            // stop to refresh progress
	            if (timer_egwUpgradeTaskProReload) {
	                clearInterval(timer_egwUpgradeTaskProReload);
	                timer_egwUpgradeTaskProReload = undefined;
	                // refresh task list
	                $("#egwUpgradeListTable").datagrid("reload");
	            }
	        }
		} else {
			$("#egwUpgradeTaskProTable").datagrid("loadData", {total:0,rows:[]});
		}
	}, "json");
}

function queryUpgradeTaskInfo() {
	var selectedTask = $("#egwUpgradeListTable").datagrid("getSelected");
	var task_id =  selectedTask["id"];
	var param = {};
	param.timeZone=timeZone;
	param.taskId=task_id;
	param.searchText = $("#searchText_egwTaskResult").val();
	$.post("${ctx}/egw/softwareUpgrade/querySoftwareExeResPageList.action", param, function(data) {
		var gridData = {total:0,rows:[]};
		if (data){
			gridData = data;
		}
		$("#egwUpgradeTaskProTable").datagrid("loadData", gridData);
	}, "json");
}
</script>