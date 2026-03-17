<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<!-- 右上角导出按钮 -->
<div class="omcTitleButton">
	<span class="titleButtonText"><%=rb.getString("DaoChu")%></span><span class="circleBg export_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="exportConfigRecoverProgResult()"></span>
</div>

<%-- 配置还原任务进度 --%>
<div class="panelDefault">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer">
			<li class="default">${taskInfo.TASK_NAME} <%=rb.getString("ZhiXingJieGuo")%></li>
		</ul>
	</div>
	<div class="panelTableDiv">		
		<table class="easyui-datagrid" id="configRecoverTaskProg" fit="true" fitColumns="true"
			   data-options="singleSelect:true,idField : 'SERIAL_NUMBER',rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_configRecoverTask'">
			<thead>
				<tr>
					<th data-options="field:'SERIAL_NUMBER'" width="110"><%=rb.getString("XiaoZhanBianMa")%></th>
					<th data-options="field:'HOST_NAME'" width="110"><%=rb.getString("HostName")%></th>
					<th data-options="field:'ORI_VERSION'" width="150"><%=rb.getString("ChuShiBanBen")%></th>
					<th data-options="field:'PROGRESS_DETAIL'" width="350"><%=rb.getString("JinDu")%></th>
					<th data-options="field:'RUN_TIME'" width="120"><%=rb.getString("ShiJian")%></th>
				</tr>
			</thead>
		</table>
	</div>
</div>

<form id="configRecoverTaskProgResult" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId"/>
</form>

<%-- 配置还原任务进度工具栏 --%>
<div id="toolbar_configRecoverTask" class="omcTableTool defaultQuery">
	<input type="text" id="searchText_configRecover" class="searchInputStyle"  placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>" style="width:400px;margin-left:27px;" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)"/>
	<b class="searchResultImgChangeStyle" onclick="queryRecoverTaskInfo()"></b>
</div>

<script type="text/javascript">

var configRecover_task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_configRecoverTaskProgReload;
$(function () {
	$("#searchText_configRecover").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryRecoverTaskInfo();
		}
	}); 
	
	refreshConfigRecoverProgress();
	if (timer_configRecoverTaskProgReload) {
		clearInterval(timer_configRecoverTaskProgReload);
		timer_configRecoverTaskProgReload = undefined;
	}
	if (configRecover_task_progress != '2') {// 未结束，则定时刷新
		clearInterval(timer_configRecoverTaskProgReload);
		timer_configRecoverTaskProgReload = setInterval("refreshConfigRecoverProgress()", 3000);
	}
});

function refreshConfigRecoverProgress() {
	if ($("#configRecoverTaskProg").length == 0) {
		clearInterval(timer_configRecoverTaskProgReload);
		timer_configRecoverTaskProgReload = undefined;
	}
	
	var param = {};
	param.timeZone=timeZone;
	if ($("#searchText_configRecover").val()) {
		param.searchText = $("#searchText_configRecover").val();
	}
	
	$.post("${ctx}/task/configRecover/getConfigRecoverTaskProgress.action?task_id=${taskInfo.TASK_ID}", param, function(data) {
		$("#configRecoverTaskProg").datagrid("loadData", data.grid);
		if ("2" == data.pro) {
			if (timer_configRecoverTaskProgReload) {
				clearInterval(timer_configRecoverTaskProgReload);
				timer_configRecoverTaskProgReload = undefined;
				$("#configRecoverTaskList").datagrid("reload");
			}
		}
	}, "json");
}

function exportConfigRecoverProgResult() {
	var url = "${ctx}/task/configRecover/exportConfigRecoverProgResult.action";
	/* $("#configRecoverTaskProgResult").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone;
			param.searchText = $("#searchText_configRecover").val();
			var bool = checkParams(param)
			if(!bool) return false;
		}
	}); */
	exportByForm(url, {
		taskId: '${taskInfo.TASK_ID }',
		timeZone: timeZone,
		searchText: $("#searchText_configRecover").val()
	});
}

function queryRecoverTaskInfo() {
	var param = {};
	param.timeZone=timeZone;
	if ($("#searchText_configRecover").val()) {
		param.searchText = $("#searchText_configRecover").val();
	}
	$.post("${ctx}/task/configRecover/getConfigRecoverTaskProgress.action?task_id=${taskInfo.TASK_ID}", param, function(data) {
		$("#configRecoverTaskProg").datagrid("loadData", data.grid);
	}, "json");
}
</script>