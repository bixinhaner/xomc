<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- MML脚本任务进度 --%>
<div class="slideDiv" style="height:300px;box-shadow: 0 0 10px rgba(0,0,0,0.1)">	
	<div class="slideHeader" style="border:none;">
		<h3><%=rb.getString("JieGuo")%></h3>
		<ul class="iconText">
			<li><a class="titleIcon_close iconSize" onclick="closeMMLSlideDiv()"></a></li>			
		</ul>
	</div>
	<div style="padding: 0 20px;position:absolute;top:50px;bottom:1px;left:0px;right:0;overflow:auto;">
		<table class="easyui-datagrid" id="noSenceTaskProg" fit="true" fitColumns="true"
			data-options="singleSelect:true,idField : 'serialNumber',rownumbers:true,border:false,striped:true,pagination:true,onBeforeLoad:beforeLoadSecretReboot,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_mmlScriptTask',url:'${ctx}/task/secretReboot/getRebootTaskProgress.action?taskId=${taskInfo.taskId}'">
			<thead>
				<tr>
					<th data-options="field:'ID'" width="5" hidden="true"></th>
					<th data-options="field:'serialNumber'" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
					<th data-options="field:'hostName'" width="100"><%=rb.getString("HostName")%></th>
					<th data-options="field:'progressStatus',formatter:resultTableStatus" width="120"><%=rb.getString("ZhuangTai")%></th>
					<th data-options="field:'progressResult',formatter:resultTableResult" width="120"><%=rb.getString("JieGuo")%></th>
					<th data-options="field:'failureReason'" width="120"><%=rb.getString("PCILOCKShiBaiYuanYin")%></th>
					<th data-options="field:'runTime'" width="120"><%=rb.getString("ShiJian")%></th>
				</tr>
			</thead>
		</table>
	</div>	
</div>

<form id="mmlProgResult" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input id="protaskId" type="hidden" value="${taskInfo.taskId }" name="taskId"/>
</form>

<%-- MML脚本任务进度工具栏 --%>
<div id="toolbar_mmlScriptTask" class="defaultQuery queryGroup">
	<input id="searchText_mmlScript" placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>" />
	<b class="el-icon el-icon-common-search" onclick="queryMMLTaskInfo()"/></b>
</div>

<script type="text/javascript">
var resultSearchText = "";
var MMLScript_task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_secretReboot;
var aataskId = "${taskInfo.taskId }";
$(function () {
	//周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_secretReboot);
	timer_secretReboot = undefined;
	timer_secretReboot = setInterval(function(){
		updateTable($("#noSenceTaskProg"));
		var dom = $("#noSenceTaskProg");
		if(dom.length == 0) clearInterval(timer_secretReboot);
	},6000)
	//结果列表的搜索框回车事件
	$("#searchText_mmlScript").bind("keyup", function(e){
		if (e.keyCode == 13){
			$("#noSenceTaskProg").datagrid("reload");
		}
	}); 
	
	$(".slideHeader .titleIcon_export").hover(function(){
		$(".hoverShow").fadeIn();
	},function (){
		$(".hoverShow").fadeOut();
	})
});
function closeMMLSlideDiv(){
	$("#bottomContainerPanel").hide(400);
	clearInterval(timer_secretReboot);
}
function beforeLoadSecretReboot(param){
	param.timeZone = timeZone;
	param.searchText =  $("#searchText_mmlScript").val();
}
</script>