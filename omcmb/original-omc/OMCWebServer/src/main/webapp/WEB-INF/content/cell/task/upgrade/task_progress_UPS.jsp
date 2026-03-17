<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- UPS -策略-升级策略 软件系统升级 升级任务进度（执行结果） --%>
<div class="slideDiv" style="height:300px;" >	
	<div class="slideHeader" style="border:none;">
		<h3><%=rb.getString("ZhiXingJieGuo")%></h3>
		<ul class="iconText">
			<li><span class="hoverShow"><%=rb.getString("DaoChu")%></span><a style='font-size:18px;' class="slideExport el-icon el-icon-operation-export iconSize" style="margin-right:0px;" onclick="exportUpgradeProgResult()"></a></li>
			<li><a class="el-icon el-icon-close" style='font-size:18px;' onclick="closeSlideDiv()"></a></li>
		</ul>
	</div>
	<div style="position:absolute;top:61px;bottom:1px;left:0px;right:0;overflow:auto;">
		<table class="easyui-datagrid" id="upgradTaskProg" fit="true" fitColumns="true" 
			   data-options="singleSelect:true,idField : 'SERIAL_NUMBER',rownumbers:true,border:false,striped:true,pagination:true,onBeforeLoad:beforeLoadUPS,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_upgradeTask',url:'${ctx}/task/upgrade/ups/getUpgradeTaskProgress.action?task_id=${taskInfo.TASK_ID}'">
			<thead>
				<tr>
					<th data-options="field:'ID'" width="5" hidden="true"></th>
					<th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("DianYuanBianMa")%></th>
					<th data-options="field:'ORI_VERSION'" width="150"><%=rb.getString("ChuShiBanBen")%></th>
					<th data-options="field:'PROGRESS_STATUS',formatter:resultTableStatus" width="120"><%=rb.getString("ZhuangTai")%></th>
					<th data-options="field:'PROGRESS_RESULT',formatter:resultTableResult" width="120"><%=rb.getString("JieGuo")%></th>
					<th data-options="field:'FAILURE_REASON'" width="120"><%=rb.getString("PCILOCKShiBaiYuanYin")%></th>
					<th data-options="field:'RUN_TIME'" width="100"><%=rb.getString("ShiJian")%></th>
				</tr>
			</thead>
		</table>
	</div>
</div>

<form id="upgradTaskProgResult" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId"/>
</form>

<%-- CPE软件系统升级 升级任务进度工具栏 --%>
<div id="toolbar_upgradeTask" style="height: 46px;padding:0px 20px 0px 0px;">
	<div class="queryGroup">	
		<input id="searchText_upgradeUPS" name ="searchText_upgrade" placeholder="<%=rb.getString("DianYuanBianMa")%>"/>
		<b class="el-icon el-icon-common-search" onclick="queryUpgradeTaskInfo()"></b>	
	</div>
</div>

<input type="hidden" id="upsType" value="" />

<script type="text/javascript">
var resultSearchText = "";
var ssu = $("#searchText_upgradeUPS").val();
var task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_upgradeUPS;

function exportUpgradeProgResult(){
	/* $("#upgradTaskProgResult").form('submit', {
		url: "${ctx}/task/upgrade/ups/exportUpgradeProgResult.action",
		onSubmit: function(param) {
            param.timeZone=timeZone;
            param.searchText_upgrade = $("#searchText_upgradeUPS").val();
			var bool = checkParams(param)
			if(!bool) return false;
        }
	}); */
	exportByForm(url,{
		taskId: '${taskInfo.TASK_ID }',
		timeZone: timeZone,
		searchText_upgrade: $("#searchText_upgradeUPS").val()
	});
}

$(function () {
	//周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_upgradeUPS);
	timer_upgradeUPS = undefined;
	timer_upgradeUPS = setInterval(function(){
		updateTable($("#upgradTaskProg"));
		var dom = $("#upgradTaskProg");
		if(dom.length == 0) clearInterval(timer_upgradeUPS);
	},6000)
	//结果列表的搜索框回车事件
	$("#searchText_upgradeUPS").bind("keyup", function(e){
		if (e.keyCode == 13){
			$("#upgradTaskProg").datagrid("reload");
		}
	}); 
	
	$("#upsType").val("${upsType}");
	
	$(".slideHeader .el-icon-operation-export").hover(function(){
		$(".hoverShow").fadeIn();
	},function (){
		$(".hoverShow").fadeOut();
	})
});

function closeSlideDiv(){
	$("#layout_center_progress_UPS").hide(400);
	clearInterval(timer_upgradeUPS);
}
function beforeLoadUPS(param){
	param.timeZone = timeZone;
	param.searchText = $("#searchText_upgradeUPS").val();
}
</script>