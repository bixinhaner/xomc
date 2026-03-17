<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 重启任务进度 --%>
<div class="slideDiv" style="height:300px;" >
	<div class="slideHeader" style="border:none;">
		<h3><%=rb.getString("ZhiXingJieGuo")%></h3>
		<ul class="iconText">
			<li><span class="hoverShow"><%=rb.getString("DaoChu")%></span><a class="slideExport titleIcon_export iconSize" style="margin-right:0px;vertical-align: middle;" onclick="exportrTraceProgResultCpe()"></a></li>
			<li><a class="titleIcon_close iconSize" onclick="closeSlideDiv()"></a></li>
		</ul>
	</div>
	<div style="position:absolute;top:40px;bottom:1px;left:20px;right:20px;overflow:auto;">
		<table class="easyui-datagrid" id="traceTaskProgCpe" fit="true" fitColumns="true"
			   data-options="singleSelect:true,idField : 'SERIAL_NUMBER',rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_traceTaskCpe'">
			<thead>
				<tr>
					<th data-options="field:'CPE_NAME',editor:'text'" width="90"><%=rb.getString("CPEName")%></th>
					<th data-options="field:'IMSI',sortable:true" width="100">IMSI</th>
					<th data-options="field:'UL_MCS',sortable:false" width="75">UL_MCS</th>
					<th data-options="field:'DL_MCS',sortable:false" width="75">DL_MCS</th>
					<th data-options="field:'RSRP0',sortable:false" width="70">RSRP1</th>
					<th data-options="field:'RSRP1',sortable:false" width="70">RSRP2</th>
					<th data-options="field:'CINR0',sortable:false" width="70">CINR1</th>
					<th data-options="field:'CINR1',sortable:false" width="70">CINR2</th>
					<th data-options="field:'CPE_SINR'" width="60">SINR</th>
					<th data-options="field:'DL_CURRENT_DATARATE',sortable:false" width="110"><%=rb.getString("CPEXiaXingTunTuLiang")%> (Mbps)</th>
					<th data-options="field:'UL_CURRENT_DATARATE',sortable:false" width="110"><%=rb.getString("CPEShangXingTunTuLiang")%> (Mbps)</th>
					<th data-options="field:'SERIAL_NUMBER',sortable:false" width="120"><%=rb.getString("CPEXuLieHao")%></th>
					<th data-options="field:'MACADDRESS',sortable:false" width="120"><%=rb.getString("CPEMacAddress")%></th>
					<th data-options="field:'TX_POWER'" width="60">TX_POWER</th>
					<th data-options="field:'MCC'" width="60">MCC</th>
					<th data-options="field:'MNC'" width="60">MNC</th>
					<th data-options="field:'BAK_TIME'" width="140">TRACE_TIME</th>
				</tr>
			</thead>
		</table>
	</div>
</div>

<!-- <div id="traceTaskProgToolsBarCpe">
	<a href="javascript:void(0)" onclick="exportrTraceProgResultCpe()" class="icon-export"></a>
</div> -->

<form id="traceProgResultCpe" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId" id="taskId"/>
</form>

<div id="toolbar_traceTaskCpe" class="toolbarContainer">
	<div class="queryGroup">	
		<input id="searchText_trace_cpe" placeholder="<%=rb.getString("QingShuRuCpeChaXunNeiRong")%>"/>
		<b onclick="queryTraceTaskInfo()"></b>	
	</div>
</div>

<script type="text/javascript">
var trace_task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_traceTaskProgReload;
var flag = false;

function exportrTraceProgResultCpe(){
	var url = "${ctx}/task/trace/exportTraceProgResultForCpe.action";
	/* $("#traceProgResultCpe").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone;
			param.searchText = $("#searchText_trace_cpe").val();
			var bool = checkParams(param)
			if(!bool) return false;
		}
	});	 */
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
	$(".slideHeader .titleIcon_export").hover(function(){
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
	$.post("${ctx}/task/trace/getTraceTaskProgressForCpe.action?task_id=" + taskId, param, function(data) {
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
	$.post("${ctx}/task/trace/getTraceTaskProgressForCpe.action?task_id=" + taskId, param, function(data) {
		$("#traceTaskProgCpe").datagrid("loadData", data.grid);
	}, "json");
}
</script>