<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- MML脚本任务进度 --%>
<div class="slideDiv" style="height:350px;box-shadow:0 0 10px rgba(0,0,0,0.15)">	
	<div class="el-card__header">
		<span><%=rb.getString("JieGuo")%></span>
		<ul class="iconText">
			<li><span class="hoverShow"><%=rb.getString("DaoChu")%></span><a class="el-icon el-icon-operation-export" style="margin-right:10px;right: 35px;" onclick="exportMMLProgResult()"></a></li>
			<li><a class="el-icon el-icon-close" onclick="closeMMLSlideDiv()"></a></li>			
		</ul>
	</div>
	<div style="position:absolute;top:40px;bottom:1px;left:0px;right:0;overflow:auto;">
		<table class="easyui-datagrid" id="MMLScriptTaskProg" fit="true"
			data-options="fitColumns: true,singleSelect:true,idField : 'SERIAL_NUMBER',rownumbers:true,border:false,striped:true,pagination:true,onBeforeLoad:beforeLoadMML,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_mmlScriptTask',url:'${ctx}/task/MMLScript/getMMLScriptTaskProgress.action?task_id=${taskInfo.TASK_ID}'">
			<thead>
				<tr>
					<th data-options="field:'ID'" width="5" hidden="true"></th>
					<th data-options="field:'SERIAL_NUMBER'" width="150"><%=rb.getString("XiaoZhanBianMa")%></th>
					<th data-options="field:'HOST_NAME'" width="100"><%=rb.getString("HostName")%></th>
					<th data-options="field:'MML'" width="400"><%=rb.getString("CLIFile")%></th>
					<%-- <th data-options="field:'PROGRESS_DETAIL'" width="500"><%=rb.getString("JinDu")%></th> --%>
					<th data-options="field:'PROGRESS_STATUS',formatter:resultTableStatus" width="120"><%=rb.getString("ZhuangTai")%></th>
					<th data-options="field:'PROGRESS_RESULT',formatter:resultTableResult" width="120"><%=rb.getString("JieGuo")%></th>
					<th data-options="field:'FAILURE_REASON'" width="160"><%=rb.getString("PCILOCKShiBaiYuanYin")%></th>
					<th data-options="field:'DETAIL',formatter: lstDetailFmt" width="160"><%=rb.getString("XiangQing")%></th>
					<th data-options="field:'RUN_TIME'" width="120"><%=rb.getString("KaiShiShiJian")%></th>
					<th data-options="field:'END_TIME'" width="120"><%=rb.getString("JieShuShiJian")%></th>
				</tr>
			</thead>
		</table>
	</div>	
</div>

<form id="mmlProgResult" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId"/>
</form>

<%-- MML脚本任务进度工具栏 --%>
<div id="toolbar_mmlScriptTask" style="padding:10px 0px;">
	<div class="queryGroup">
		<input id="searchText_mmlScript" placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>" />
		<b class="el-icon el-icon-common-search" onclick="queryMMLTaskInfo()"></b>
	</div>
</div>

<div id="task_details" class="easyui-dialog" data-options="closed:true" title="<%=rb.getString("XiangQing")%>"></div>

<script type="text/javascript">
var resultSearchText = "";
var MMLScript_task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_mmlProgress;

function exportMMLProgResult(){
	//var url = "${ctx}/task/MMLScript/exportMMLProgResult.action";
	var url = "${ctx}/task/MMLScript/exportMMLProgResultToCSV.action";
	/* $("#mmlProgResult").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone;
			param.searchText = $("#searchText_mmlScript").val();
			var bool = checkParams(param)
			if(!bool) return false;
		}
	}); */
	exportByForm(url, {
		taskId: '${taskInfo.TASK_ID }',
		timeZone: timeZone,
		searchText: $("#searchText_mmlScript").val()
	});
}

function lstDetailFmt(value, row, index) {
	var details = row.DETAIL,
		content = '',
		titleStr = '',
		strList = [];
	
	if(details) {
		if(details.substr(0,2).indexOf('[') < 0) {
			details = '['+ details +']';
		}
		
		var list = eval('('+details+')');
		
		titleStr = '[<br>' + getProps(list, 0) + '<br>]';

		content = details.length > 25? details.substr(0, 20)+'...':details;
	}
	
	var str = [
			'<span href="#" class="lst-note" msg="'+titleStr+'" onclick="getDetail(this)" style="cursor: pointer;color: blue;">',
				content,
			'</span>'
		].join(' ');
	
	return str;
}

function getDetail(el) {
	let titleStr = $(el).attr('msg');
	
	$('#task_details').dialog({
		width: 600,
		height: 500,
		top: 100,
		content: titleStr,
		modal: true,
		closed: false
	});
}

function getProps(list, n) {
	var prepad = '&nbsp;&nbsp;&nbsp;&nbsp;',
		propList = [],
		nextN = n +1;
	
	for(var i = 0; i < n; i++) {
		prepad += '&nbsp;&nbsp;&nbsp;&nbsp;';
	}

	list.map(function(item, idx){
		if(idx) propList.push(prepad + '{');
		else propList.push('&nbsp;&nbsp;&nbsp;&nbsp;' + '{');
		
		for(key in item) {
			
			if(Array.isArray(item[key])) {
				propList.push(prepad +'&nbsp;&nbsp;&nbsp;&nbsp;'+ key + ': [');
				
				propList.push(prepad + getProps(item[key], nextN));
				
				propList.push(prepad +'&nbsp;&nbsp;&nbsp;&nbsp;'+ '],');
			}else if(typeof(item[key]) == 'object') {
				propList.push(prepad + prepad + key + ': {');
				for(mkey in item[key]) {
					propList.push(prepad + prepad +'&nbsp;&nbsp;&nbsp;&nbsp;'+ mkey + ': &quot;' + item[key][mkey] + '&quot;');
				}
				propList.push(prepad + prepad + '},');
			}else {
				propList.push(prepad +'&nbsp;&nbsp;&nbsp;&nbsp;'+ key + ': &quot;' + item[key] + '&quot;');
			}
		}
		
		propList.push(prepad + '},');
	});
	
	return propList.join('<br>');
}

$(function () {
	//周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_mmlProgress);
	timer_mmlProgress = undefined;
	timer_mmlProgress = setInterval(function(){
		updateTable($("#MMLScriptTaskProg"));
		var dom = $("#MMLScriptTaskProg");
		if(dom.length == 0) clearInterval(timer_mmlProgress);
	},6000)
	//结果列表的搜索框回车事件
	$("#searchText_mmlScript").bind("keyup", function(e){
		if (e.keyCode == 13){
			$("#MMLScriptTaskProg").datagrid("reload");
		}
	}); 
	
	$(".slideHeader .titleIcon_export").hover(function(){
		$(".hoverShow").fadeIn();
	},function (){
		$(".hoverShow").fadeOut();
	})
});
function closeMMLSlideDiv(){
	$("#MMLScriptTaskProgress").hide(400);
	$("#MMLScriptTaskProgress_gnb").hide(400);
	clearInterval(timer_mmlProgress);
}
function beforeLoadMML(param){
	param.timeZone = timeZone;
	param.searchText = $("#searchText_mmlScript").val();
}
</script>