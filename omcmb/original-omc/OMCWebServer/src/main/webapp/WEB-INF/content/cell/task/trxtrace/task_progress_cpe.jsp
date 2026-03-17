<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%-- 重启任务进度 --%>
<div class="slideDiv" style="height:300px;left:15px;right:15px;bottom:15px" >
	<div class="slideHeader" style="border:none;">
		<h3><%=rb.getString("ZhiXingJieGuo")%></h3>
		<ul class="iconText">
			<li><span class="hoverShow"><%=rb.getString("DaoChu")%></span><a class="titleIcon_export iconSize" onclick="exportrTraceProgResultCpe()"></a></li>
			<li><a class="titleIcon_close iconSize" onclick="closeSlideDiv()"></a></li>
		</ul>
	</div>
	<div style="padding:0 20px 20px;position:absolute;top:40px;bottom:1px;left:0px;right:0;overflow:auto;">
		<table class="easyui-datagrid" id="traceTaskProgCpe" fit="true" fitColumns="true"
			   data-options="singleSelect:true,idField : 'SERIAL_NUMBER',rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_traceTaskCpe'">
			<thead><tr>
				<th data-options="field:'TASK_ID'" width="5" hidden="true"></th>
				<th data-options="field:'SMALL_CELL_CODE'" width="100" hidden="true"></th>
				<th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
				<th data-options="field:'HOST_NAME'" width="100"><%=rb.getString("HostName")%></th>
				<th data-options="field:'PROGRESS_DETAIL'" width="500"><%=rb.getString("JinDu")%></th>
				<th data-options="field:'RUN_TIME'" width="100"><%=rb.getString("ShiJian")%></th>
				<th data-options="field:'GRAPHY',sortable:false,formatter:trxGraph" width="50"><%=rb.getString("TuBiao")%></th>		
			</tr></thead>
		</table>
	</div>
</div>
<div id="traceTaskProgToolsBarCpe">
	<a href="javascript:void(0)" onclick="exportrTraceProgResultCpe()" class="icon-export"></a>
</div>

<form id="traceProgResultCpe" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId" id="taskId"/>
</form>

<div id="toolbar_traceTaskCpe" class="toolbarContainer defaultQuery">
	<input id="searchText_trace_cpe"  style="margin-left:0px" placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
	<b class="searchResultImgChangeStyle" onclick="queryTraceTaskInfo()"></b>	
</div>
<%-- 窗口-cell trx trace数据折线图--%>
<%-- <div id="winTrxGraph" class="easyui-window" title="<%=rb.getString("TuBiao")%>"
     data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:1150,height:600,resizable:false,inline:false">
</div> --%>
<script type="text/javascript">
var trace_task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_traceTaskProgReload;
var flag = false;

function exportrTraceProgResultCpe(){
	var url = "${ctx}/task/trxtrace/exportTraceProgResultForCpe.action";
	/* $("#traceProgResultCpe").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone;
			param.searchText = $("#searchText_trace_cpe").val();
			var bool = checkParams(param)
			if(!bool) return false;
		}
	}); */
	exportByForm(url,{
		taskId: '${taskInfo.TASK_ID }',
		timeZone: timeZone,
		searchText: $("#searchText_trace_cpe").val()
	});
}

$(function () {
	$("#searchText_trace_cpe").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryTraceTaskInfo();
		}
	}); 
	
	refreshTraceProgress();
	if (timer_traceTaskProgReload) {
		clearInterval(timer_traceTaskProgReload);
		timer_traceTaskProgReload = undefined;
	}
	if (trace_task_progress != '2') {// 未结束，则定时刷新
		clearInterval(timer_traceTaskProgReload);
		timer_traceTaskProgReload = setInterval("refreshTraceProgress()", 6000);
	}
	$(".slideHeader .iconExport").hover(function(){
		$(".hoverShow").fadeIn();
	},function (){
		$(".hoverShow").fadeOut();
	})
});

function closeSlideDiv(){
	$(".slideDiv").animate({bottom:'-400px'},400);
	$("#traceTaskProgressCpe").hide(400);
}

function refreshTraceProgress() {
	if ($("#traceTaskProgCpe").length == 0) {
		clearInterval(timer_traceTaskProgReload);
		timer_traceTaskProgReload = undefined;
	}
	
	var param = {};
	param.timeZone=timeZone;
	if ($("#searchText_trace_cpe").val()) {
		param.searchText = $("#searchText_trace_cpe").val();
	}
	var taskId = $("#taskId").val();
	$.post("${ctx}/task/trxtrace/getTraceTaskProgressForCpe.action?task_id=" + taskId, param, function(data) {
		$("#traceTaskProgCpe").datagrid("loadData", data.grid);
		if("1" == data.rflag){
			$("#traceTaskListCpe").datagrid("reload");
		}
		if ("2" == data.pro) {
			if (timer_traceTaskProgReload) {
				clearInterval(timer_traceTaskProgReload);
				timer_traceTaskProgReload = undefined;
				$("#traceTaskListCpe").datagrid("reload");
			}
		}
	}, "json");
}

function queryTraceTaskInfo() {
	var param = {};
	param.timeZone=timeZone;
	if ($("#searchText_trace_cpe").val()) {
		param.searchText = $("#searchText_trace_cpe").val();
	}
	var taskId = $("#taskId").val();
	$.post("${ctx}/task/trxtrace/getTraceTaskProgressForCpe.action?task_id=" + taskId, param, function(data) {
		$("#traceTaskProgCpe").datagrid("loadData", data.grid);
	}, "json");
}


function trxGraph(value, rowData, rowIndex){
	var imgL = "<span class='historyGraph' onclick=showTrxGraph('" + rowData.SMALL_CELL_CODE + "','" + rowData.TASK_ID + "','" + rowData.SERIAL_NUMBER + "')>" + "</span>" ;
	return imgL;
}

function showTrxGraph(cellCode,taskId,sn){
	/* $("#winTrxGraph").window("open");
	$("#winTrxGraph").window("refresh", "${ctx}/task/trxtrace/toCellTrxGraph.action?cellCode=" + cellCode +"&TimeZone=" + timeZone+"&taskId=" + taskId +"&sn=" + sn); */
	var url = "${ctx}/task/trxtrace/toCellTrxGraph.action?cellCode=" + cellCode +"&TimeZone=" + timeZone+"&taskId=" + taskId +"&sn=" + sn;
	openDefaultWindow(url,{
		title: '<%=rb.getString("TuBiao")%>',
		width:1150,height:600
	});
}

</script>