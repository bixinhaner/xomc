<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>

<%-- 配置还原任务 --%>
<div style="width: 100%;height: 100%;background:#F5F7FA;">
	<%-- 任务列表 --%>
	<div class="splitPanel" style="height:49%;">
		<div class="omcPageTitleDiv">
			<ul class="omcPageTitleContainer">
				<li class="default"><%=rb.getString("HuanYuanRenWuXiangQing")%></li>
			</ul>
		</div>
		<!-- 右上角添加按钮 -->
		<div class="omcTitleButton">
			<span class="titleButtonText"><%=rb.getString("TianJia")%></span><span class="circleBg add_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="openWinAddConfigRecoverTask()"></span>
		</div>
		<div class="panelTableDiv"  style="padding-top:20px;">
			<table class="easyui-datagrid" id="configRecoverTaskList" fit="true" fitColumns="true"
					data-options="singleSelect:true,rownumbers:true,pagination:true,border:false,striped:true,pagePosition:'bottom',onSelect:showConfigRecoverTaskDetail,
					url:'${ctx}/task/configRecover/getConfigRecoverTaskList.action',onLoadError:datagridLoadError,onLoadSuccess:loadSuccessConfigRecoverTaskList,idField:'TASK_ID'">
	            <thead>
		            <tr>
		                <th data-options="field:'TASK_ID',hidden:true"></th>
		                <th data-options="field:'TASK_NAME'" width="110"><%=rb.getString("RenWu")%> <%=rb.getString("MingCheng")%></th>
		                <th data-options="field:'TASK_STATUS',formatter:taskStatusFmt" width="70"><%=rb.getString("ZhuangTai")%></th>
		                <th data-options="field:'TASK_PROGRESS',formatter:taskProgressFmt" width="70"><%=rb.getString("JinDu")%></th>
						<th data-options="field:'TASK_RESULT',formatter:taskResultFmt" width="70"><%=rb.getString("JieGuo")%></th>
		                <th data-options="field:'START_TIME'" width="110"><%=rb.getString("KaiShiShiJian")%></th>
		                <th data-options="field:'STOP_TIME'" width="110"><%=rb.getString("JieShuShiJian")%></th>
		                <th data-options="field:'operation',formatter : configRecoverTaskFormatter,fixed:true" width="160"><%=rb.getString("CaoZuo")%></th>
		            </tr>
	            </thead>
	        </table>
		</div>
	</div>
	<div class="splitPanel"  style="height:49%;margin-top: 15px;">
		<div id="configRecoverTaskProgress" style="height:100%;position:relative;"></div>
	</div>
</div>

<%-- 窗口-新建任务--%>
<%-- <div id="winAddConfigRecoverTask" class="easyui-window" title="<%=rb.getString("XinJianHuanYuanRenWu")%>"
	data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:860,height:600,resizable:false">
</div> --%>

<script type="text/javascript">
$(function() {
	closeLoading();
	$("#configRecoverTaskList").datagrid({
        queryParams:{timeZone:timeZone}
    });
});

/**
 * 格式化操作
 */
function configRecoverTaskFormatter(value, rowData, rowIndex){
	var task_progress = rowData.TASK_PROGRESS;
	var task_status = rowData.TASK_STATUS;
	var task_id = rowData.TASK_ID;
	
	var JiHuo = '<%=rb.getString("JiHuo")%>';
	var GuaQi = '<%=rb.getString("GuaQi")%>';
	var ZhongZhi = '<%=rb.getString("ZhongZhiRenWu")%>';
	var ShanChu = '<%=rb.getString("ShanChu")%>';
	
	value = "";
	if(task_status != 0 && task_progress != 2){//当前不是处于激活状态，且未结束，激活图标可用
		value = value + "<div class='operationDiv operation_active' title='"+JiHuo+"' onclick='activeConfigRecoverTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='operationDiv operation_active_disabled' title='"+JiHuo+"'></div>";
	}
	
	if(task_status != 1 && task_progress != 2){//当前不是处于挂起状态，且未结束，挂起图标可用
		value = value + "<div class='operationDiv operation_awaiting' title='"+GuaQi+"' style='margin-left:15px;' onclick='suspendConfigRecoverTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='operationDiv operation_awaiting_disabled' title='"+GuaQi+"' style='margin-left:15px;'></div>";
	}
	
	if(task_progress != 2){//当前任务没有结束，终止图标可用
		value = value + "<div class='operationDiv operation_terminate' title='"+ZhongZhi+"' style='margin-left:15px;' onclick='terminateConfigRecoverTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='operationDiv operation_terminate_disabled' title='"+ZhongZhi+"' style='margin-left:15px;'></div>";
	}
	
	if(task_progress != 1){//当前任务不在进行中，删除图标可用
		value = value + "<div class='operationDiv operation_delete' title='"+ShanChu+"' style='margin-left:15px;' onclick='delConfigRecoverTask(\"" + task_id + "\")'></div>";
	}else{  
		value = value + "<div class='operationDiv operation_delete_disabled' title='"+ShanChu+"' style='margin-left:15px;'></div>";
	}
	return value;
}

// 显示任务详情
function showConfigRecoverTaskDetail() {
	var selectedTask = $("#configRecoverTaskList").datagrid("getSelected");
	if (!selectedTask) {
		return;
	}
	var task_id = selectedTask["TASK_ID"];
    $("#configRecoverTaskProgress").panel({
        href: '${ctx}/task/configRecover/toConfigRecoverTaskProgress.action?task_id=' + task_id
    });
}

//激活任务
function activeConfigRecoverTask() {
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingLiJiZhiXingRenWu")%>", function(r) {
        if (r) {
        	var grid = $("#configRecoverTaskList");
        	var selTask = grid.datagrid("getSelected");
        	var params = {};
        	params["taskId"] = selTask["TASK_ID"];
        	
        	$.post("${ctx}/task/configRecover/activeTask.action", params, function(data) {
        		if (data["success"]) {
        			grid.datagrid("load");
        		} else {
        			$.messager.alert(TiShi, data["message"]);
        		}
        	}, "json");
        }
 	}).addClass("normalConfirm");
}

// 挂起任务
function suspendConfigRecoverTask() {
	var grid = $("#configRecoverTaskList");
	var selTask = grid.datagrid("getSelected");
	$.post("${ctx}/task/configRecover/suspendTask.action", {"taskId": selTask["TASK_ID"]}, function(data) {
		if (data["success"]) {
			grid.datagrid("load");
		} else {
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

//终止任务
function terminateConfigRecoverTask() {
	var selectedTask = $("#configRecoverTaskList").datagrid("getSelected");
	if (!selectedTask) {
		$.messager.alert(TiShi, "<%= rb.getString("QingXuanZeRenWu")%>");
		return;
	}
	if (selectedTask["TASK_PROGRESS"] == 2) {
		$.messager.alert(TiShi, "<%= rb.getString("RenWuYiJieShu")%>");
		return;
	}
	var params = {};
	params["taskId"] = selectedTask["TASK_ID"];
	$.post("${ctx}/task/configRecover/terminateConfigRecoverTask.action", params, function(data) {
		if (data["success"]) {
			$("#configRecoverTaskList").datagrid("reload");
		} else {
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

//删除任务
function delConfigRecoverTask(){
	var selectedTask = $("#configRecoverTaskList").datagrid("getSelected");
	if (!selectedTask) {
		$.messager.alert(TiShi, "<%= rb.getString("QingXuanZeRenWu")%>");
		return;
	}
	
	var taskId = selectedTask["TASK_ID"];
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
        if (r) {
            $.post('${ctx}/task/configRecover/delConfigRecoverTask.action', {"taskId": taskId}, function(data) {
                if (data["success"]) {
                    $("#configRecoverTaskList").datagrid("reload");
                } else {
                    $.messager.alert(TiShi, data["message"]);
                }
            }, "json");
        }
    }).addClass("seriousConfirm");
}

// 任务列表加载完成，默认选中第一条数据
function loadSuccessConfigRecoverTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#configRecoverTaskList").datagrid("selectRow", 0);
	}
}

// 打开添加任务窗口
function openWinAddConfigRecoverTask() {
	/* $("#winAddConfigRecoverTask").window({
		width: 960,
    	height: document.body.clientHeight * 0.9
    }).window("center").window("open");
    
    $("#winAddConfigRecoverTask").window("refresh", "${ctx}/task/configRecover/goAddTask.action"); */
	
	var url = "${ctx}/task/configRecover/goAddTask.action",
		options = {
    		title: '<%=rb.getString("XinJianHuanYuanRenWu")%>',
    		width: 960,
        	height: document.body.clientHeight * 0.9
		};
	openDefaultWindow(url,options);
}
</script>