<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#autoTask .queryItem {
	display: inline-block;
	vertical-align: bottom;
	padding: 10px 15px 0 0;
}
</style>

<%-- 自动任务详情 --%>
<div class="panelDefault">
	<div class="singleTitle">
		<%=rb.getString("XiangQing")%>
	</div>
	<div class="contentDiv">
		<table id="autoTaskProgressList"></table>
	</div>
</div>

<%-- 自动任务详情工具栏  --%>
<div id="toolbar_autoTaskProgressList" class="toolbarContainer">
	<div class="queryItem" style="display: none;">
		<label>开始时间</label>
		<input id="startTime_autoTask" class="easyui-datetimebox border-box border" style="height:26px;">
	</div>
	<div class="queryItem" style="display: none;">
		<label>结束时间</label>
		<input id="stopTime_autoTask" class="easyui-datetimebox border-box border" style="height:26px;">
	</div>
	<div class="queryGroup">
		<input id="searchText_autoTask" class="border-box border" placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>">
		<b onclick="queryPro_autoTask()"></b>
	</div>
	<div class="linkbuttonGroup">
		<%-- <a class="easyui-linkbutton" onclick="queryPro_autoTask()"><%=rb.getString("SouSuo")%></a> --%>
		<a class="linkbutton"  onclick="exportAutoTaskPro()"><span><%=rb.getString("DaoChu")%></span></a>
		<a class="linkbutton"  onclick="clearAutoTaskPro()"><span><%=rb.getString("QingKong")%></span></a>
	</div>
</div>

<%-- 表单-导出进度数据 --%>
<form id="formExportAutoTaskPro" style="display:none" method="post"
	action="${ctx}/cell/selfstart/exportAutoTaskProCsv.action?taskName=${taskName}">
</form>

<script>
var intervalAutoUpgradeTaskProgress; //升级进度
$(function () {
	// 定义任务详情列表
	$("#autoTaskProgressList").datagrid({
		url: "${ctx}/cell/selfstart/getAutoTaskPro.action?taskName=${taskName}",
		queryParams:{timeZone:timeZone},
		singleSelect : true,
		rownumbers:true,
		fit : true,
		fitColumns : true,
		border : false,
		toolbar:'#toolbar_autoTaskProgressList',
		striped: true,
		idField: 'id',
		pagination: true,
		onBeforeLoad: beforeLoad_autoTaskProgressList,
		columns: [[
			{field: 'id', hidden: true},
			{field: 'task_id', hidden: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'serial_number', width: 150, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name', width: 100, title: '<%=rb.getString("HostName")%>'},
			{field: 'ori_version', width: 180, title: '<%=rb.getString("ChuShiBanBen")%>'},
			{field: 'dest_version', width: 150, title: '<%=rb.getString("ShengJiBanBen")%>',hidden: true},
			{field: 'progress', width: 150, title: '<%=rb.getString("JinDu")%>'},
			{field: 'start_time', width: 150, title: '<%=rb.getString("KaiShiShiJian")%>'},
			{field: 'stop_time', width: 150, title: '<%=rb.getString("JieShuShiJian")%>'}
		]]
	});
	
	
	// 搜索框，添加回车事件
	$("#searchText_autoTask").bind("keyup", function(e) {
		if (e.keyCode == 13) {
			$("#autoTaskProgressList").datagrid("load");
		}
	});
	
	if (intervalAutoUpgradeTaskProgress) {
		clearInterval(intervalAutoUpgradeTaskProgress);
		intervalAutoUpgradeTaskProgress = undefined;
	}

	if ("${_switch}" == '1') {// 未关闭，则定时刷新
		clearInterval(intervalAutoUpgradeTaskProgress);
		intervalAutoUpgradeTaskProgress = setInterval("refreshAutoUpgradeTaskProgress()", 3000);
	}
})

// 查询
function queryPro_autoTask() {
	$("#autoTaskProgressList").datagrid("reload");
}

// 进度表-加载完成事件
function beforeLoad_autoTaskProgressList(param) {
	$(this).datagrid("enableContextmenuAutoSize");
	try {
		$(this).datagrid("enableContextmenuAutoSize");
		if ($("#startTime_autoTask").datetimebox("getValue")) {
			param.startTime = $("#startTime_autoTask").datetimebox("getValue");
		}
		if ($("#stopTime_autoTask").datetimebox("getValue")) {
			param.stopTime = $("#stopTime_autoTask").datetimebox("getValue");
		}
		if ($("#searchText_autoTask").val()) {
			param.searchText = $("#searchText_autoTask").val();
		}
	} catch (e) {}
}

// 导出进度数据
function exportAutoTaskPro() {
	var param = {
		timeZone:timeZone,
		"taskName" : "${taskName}"
	};
	if ($("#startTime_autoTask").datetimebox("getValue")) {
		param.startTime = $("#startTime_autoTask").datetimebox("getValue");
	}
	if ($("#stopTime_autoTask").datetimebox("getValue")) {
		param.stopTime = $("#stopTime_autoTask").datetimebox("getValue");
	}
	if ($("#searchText_autoTask").val()) {
		param.searchText = $("#searchText_autoTask").val();
	}

	$("#formExportAutoTaskPro").form({
		"queryParams": param
	}).form("submit",{
		onSubmit: function(param){
			var bool = checkParams(param);
			if(!bool) return false;
        }
	});
}

// 清空自动任务进度
function clearAutoTaskPro() {
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenQingKongJinDu")%><%=rb.getString("WenHao")%>", function(r) {
        if (r) {
        	var param = {"taskName" : "${taskName}"};
        	$.post("${ctx}/cell/selfstart/clearAutoTaskPro.action", param, function(data) {
                if (data.success) {
                    $("#autoTaskProgressList").datagrid("reload");
                } else {
                    $.messager.alert(TiShi, data.message);
                }
            }, "json");
        }
    }).addClass("seriousConfirm");
}

function refreshAutoUpgradeTaskProgress() {
	if ($("#autoTaskProgressList").length == 0) {
		clearInterval(intervalAutoUpgradeTaskProgress);
		intervalAutoUpgradeTaskProgress = undefined;
	} else {
		var selectedTask = $("#autoTaskList").datagrid("getSelected");
		var param = {
				taskId: "${taskId}",
				timeZone: timeZone,
				taskName: "${taskName}",
				searchText: $("#searchText_autoTask").val()
		};
		
		$.post("${ctx}/cell/selfstart/getAutoUpgradeTaskProgress.action", param, function(data) {
	 		 if (data && data.grid){
				$("#autoTaskProgressList").datagrid("loadData", data.grid);
		        var taskSwitch = data.taskSwitch;
		        if ('0' == taskSwitch) {
		            // task is end
		            // stop to refresh progress
		            if (intervalAutoUpgradeTaskProgress) {
		                clearInterval(intervalAutoUpgradeTaskProgress);
		                intervalAutoUpgradeTaskProgress = undefined;
		                // refresh task list
		                $("#autoTaskList").datagrid("reload");
		            }
		        }
			} else {
				$("#autoTaskProgressList").datagrid("loadData", {total:0,rows:[]});
			}  
		}, "json");
	}
	
}
</script>