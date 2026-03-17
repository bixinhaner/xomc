<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>
<%-- 基站-策略-升级策略 软件系统升级 升级任务列表 --%>
<div class="easyui-layout" data-options="border:false,fit:true">
	<div region="north" data-options="height:305,split:false" class="borderChange">
		<%-- 升级任务列表 --%>
			<div class="header-title" style="border:none;">
				<h3><%=rb.getString("ShengJi")%> <%=rb.getString("RenWu") %></h3>
				<ul class="iconText">
					<li><a class="iconAdd iconSize" onclick="openWinAddTask()"><%=rb.getString("TianJia")%></a></li>
				</ul>
			</div>
			<div style="padding: 20px 20px 0px;height:240px;">
				<table class="easyui-datagrid" id="upgradTaskList"
						data-options="fit:true,fitColumns:true,singleSelect:true,rownumbers:true,pagination:true,border:false,striped:true,pagePosition:'bottom',onSelect:showUpgradeTaskDetail,
						url:'${ctx}/task/upgrade/getUpgradeTaskList.action',onLoadError:datagridLoadError,onLoadSuccess:loadSuccessUpgradTaskList,idField:'TASK_ID'">
		            <thead><tr>
		                <th data-options="field:'TASK_ID',hidden:true"></th>
		                <th data-options="field:'TASK_NAME'" width="100"><%=rb.getString("RenWu")%> <%=rb.getString("MingCheng")%></th>
		                <th data-options="field:'FILE_NAME'" width="170"><%=rb.getString("ShengJi")%> <%=rb.getString("WenJian")%></th>
		                <th data-options="field:'VERSION'" width="100"><%=rb.getString("WenJian")%> <%=rb.getString("BanBen")%></th>
		                <th data-options="field:'TASK_STATUS',formatter:taskStatusFmt" width="70"><%=rb.getString("ZhuangTai")%></th>
		                <th data-options="field:'TASK_PROGRESS',formatter:taskProgressFmt" width="70"><%=rb.getString("JinDu")%></th>
						<th data-options="field:'TASK_RESULT',formatter:taskResultFmt" width="70"><%=rb.getString("JieGuo")%></th>
		                <th data-options="field:'START_TIME'" width="110"><%=rb.getString("KaiShiShiJian")%></th>
		                <th data-options="field:'STOP_TIME'" width="110"><%=rb.getString("JieShuShiJian")%></th>
		                <th data-options="field:'operation',formatter : upgradeTaskFormatter,fixed:true" width="150"><%=rb.getString("CaoZuo")%></th>
		            </tr></thead>
		        </table>
			</div>	
	</div>
	<div id="layout_center_progress" region="center" data-options="border:false,split:true" style="padding-top: 20px;background-color: #F3F3F4;"></div>
</div>

<%-- 窗口-新建任务--%>
<%-- <div id="winAddUpgradTask" class="easyui-window" title="<%=rb.getString("XinJianShengJiRenWu")%>"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:860,height:600,resizable:false">
</div> --%>

<div id="upgradeTaskToolsBar">
	<a href="javascript:void(0)" onclick="openWinAddTask()" class="icon-add"></a>
</div>

<script type="text/javascript">
$(function() {
	closeLoading();
	$("#upgradTaskList").datagrid({
		queryParams:{timeZone:timeZone}
	});
});

/**
 * 格式化操作
 */
function upgradeTaskFormatter(value, rowData, rowIndex){
	
	var task_progress = rowData.TASK_PROGRESS;
	var task_status = rowData.TASK_STATUS;
	var task_id = rowData.TASK_ID;
	
	var JiHuo = '<%=rb.getString("JiHuo")%>';
	var GuaQi = '<%=rb.getString("GuaQi")%>';
	var ZhongZhi = '<%=rb.getString("ZhongZhi")%>';
	var ShanChu = '<%=rb.getString("ShanChu")%>';
	
	value = "";
	if(task_status != 0 && task_progress != 2){//当前不是处于激活状态，且未结束，激活图标可用
		value = value + "<div class='icon-activation' title='"+JiHuo+"' onclick='activeUpgradeTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='icon-activation-disabled' title='"+JiHuo+"'></div>";
	}
	
	if(task_status != 1 && task_progress != 2){//当前不是处于挂起状态，且未结束，挂起图标可用
		value = value + "<div class='icon-suspend' title='"+GuaQi+"' style='margin-left:15px;' onclick='suspendUpgradeTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='icon-suspend-disabled' title='"+GuaQi+"' style='margin-left:15px;'></div>";
	}
	
	if(task_progress != 2){//当前任务没有结束，终止图标可用
		value = value + "<div class='icon-termination' title='"+ZhongZhi+"' style='margin-left:15px;' onclick='terminateTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='icon-termination-disabled' title='"+ZhongZhi+"' style='margin-left:15px;'></div>";
	}
	
	if(task_progress != 1){//当前任务不在进行中，删除图标可用
		value = value + "<div class='grid-del-btn-div' title='"+ShanChu+"' style='margin-left:15px;' onclick='delUpgradeTask(\"" + task_id + "\")'></div>";
	}else{  
		value = value + "<div class='icon-remove-disabled' title='"+ShanChu+"' style='margin-left:15px;'></div>";
	}
	return value;
}

// 显示任务进度
function showUpgradeTaskDetail() {
	var selectedTask = $("#upgradTaskList").datagrid("getSelected");
	if (!selectedTask) {
		return;
	}
	var task_id = selectedTask["TASK_ID"];
    $("#layout_center_progress").panel({
        href: '${ctx}/task/upgrade/toUpgradeTaskProgress.action?task_id=' + task_id
    });
}

// 删除任务
function delUpgradeTask(idVal){
	var params = {};
	params["taskId"] = idVal;
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
        if (r) {
            $.post('${ctx}/task/upgrade/delUpgradeTask.action', params, function(data) {
                if (data["success"]) {
                    $("#upgradTaskList").datagrid("reload");
                } else {
                    $.messager.alert(TiShi, data["message"]);
                }
            }, "json");
        }
    });
}

<%-- 任务列表加载完成事件，如果有数据，则默认选中第一条数据 --%>
function loadSuccessUpgradTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#upgradTaskList").datagrid("selectRow", 0);
	}
}

// 打开新建任务窗口
function openWinAddTask() {
	var width = "${width}";
    /* $("#winAddUpgradTask").window({
    	width: width,
    	height: document.body.clientHeight * 0.9
    }).window("center").window("open");
    
    $("#winAddUpgradTask").window("refresh", "${ctx}/task/upgrade/goAddTask.action"); */
    var url = '${ctx}/task/upgrade/goAddTask.action',
    	options = {
    		title: '<%=rb.getString("XinJianShengJiRenWu")%>',
        	width: width,
        	height: document.body.clientHeight * 0.9
    	};
	openDefaultWindow(url,options);
}

// 添加任务成功后执行
function callbackForAddTask() {
	/* $("#winAddUpgradTask").window("close"); */
	closeDefaultWindow();
    $("#upgradTaskList").datagrid("reload");
}

// 终止任务
function terminateTask(idVal) {
	/* RenWuYiJieShu */
	var params = {};
	params["taskId"] = idVal;
	
	$.post("${ctx}/task/upgrade/beforeTerminateUpgradeTaskCheck.action", params, function(data) {
		if (data["success"]) {
			$.post("${ctx}/task/upgrade/terminateUpgradeTask.action", params, function(data) {
				if (data["success"]) {
					$("#upgradTaskList").datagrid("reload");
				}else{
					$.messager.alert(TiShi, data["message"]);
				}
			}, "json");
		} else{
			//有正在执行的任务
			if (data["message"].indexOf("some device is in progress") > -1) {
				$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("BuFenSheBeiJieShuHouZhongZhi")%>", function(r) {
			    	if (r) {
			    		$.post("${ctx}/task/upgrade/terminateUpgradeTask.action", params, function(data) {
			    			if (data["success"]) {
			    				$("#upgradTaskList").datagrid("reload");
			    			}else{
			    				$.messager.alert(TiShi, data["message"]);
			    			}
			    		}, "json");
			    	}
			    });
			} else {
				$.messager.alert(TiShi, data["message"]);
			}
		}
	}, "json");
}
// 激活任务
function activeUpgradeTask(idVal) {
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingLiJiZhiXingRenWu")%>", function(r) {
        if (r) {
        	var params = {};
        	params["taskId"] = idVal;
        	
        	$.post("${ctx}/task/upgrade/activeTask.action", params, function(data) {
        		if (data["success"]) {
        			$("#upgradTaskList").datagrid("reload");
        		} else {
        			$.messager.alert(TiShi, data["message"]);
        		}
        	}, "json");
        }
    });
	
}

// 挂起任务
function suspendUpgradeTask(idVal) {
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/upgrade/suspendTask.action", params, function(data) {
		if (data["success"]) {
			$("#upgradTaskList").datagrid("reload");
		} else {
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}
</script>