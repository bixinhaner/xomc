<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>
<%-- patch升级任务列表 --%>
<div class="easyui-layout" data-options="border:false,fit:true">
	<div region="north" data-options="height:300,border:false,split:false" style="background-color: #fff;" class="borderChange">
		<%-- 升级任务列表 --%>				
		<div class="header-title" style="border:none;">
			<h3><%=rb.getString("CAZhengShuShengJi") %> <%=rb.getString("RenWu") %></h3>
			<ul class="iconText">
				<li><a class="iconAdd iconSize" onclick="openWinAddUpgradeCaTask()"><%=rb.getString("TianJia")%></a></li>
			</ul>
		</div>
		<div style="padding: 20px 20px 0;height:237px;">
			<table class="easyui-datagrid" id="upgradeCaTaskList" fit="true" fitColumns="true"
					data-options="singleSelect:true,rownumbers:true,pagination:true,border:false,striped:true,pagePosition:'bottom',onSelect:showUpgradeCaTaskDetail,
					url:'${ctx}/task/upgradeCa/getUpgradeCaTaskList.action',onLoadError:datagridLoadError,onLoadSuccess:loadSuccessUpgradeCaTaskList,idField:'TASK_ID'">
	            <thead>
	                <tr>
		                <th data-options="field:'TASK_ID',hidden:true"></th>
		                <th data-options="field:'TASK_NAME'" width="100"><%=rb.getString("RenWu")%> <%=rb.getString("MingCheng")%></th>
		                <th data-options="field:'FILE_NAME'" width="150"><%=rb.getString("ShengJi")%> <%=rb.getString("WenJian")%></th>
		                <th data-options="field:'VERSION'" width="100"><%=rb.getString("WenJian")%> <%=rb.getString("BanBen")%></th>
		                <th data-options="field:'TASK_STATUS',formatter:taskStatusFmt" width="70"><%=rb.getString("ZhuangTai")%></th>
		                <th data-options="field:'TASK_PROGRESS',formatter:taskProgressFmt" width="70"><%=rb.getString("JinDu")%></th>
						<th data-options="field:'TASK_RESULT',formatter:taskResultFmt" width="70"><%=rb.getString("JieGuo")%></th>
		                <th data-options="field:'START_TIME'" width="110"><%=rb.getString("KaiShiShiJian")%></th>
		                <th data-options="field:'STOP_TIME'" width="110"><%=rb.getString("JieShuShiJian")%></th>
		           		<th data-options="field:'operation',formatter : caTaskFormatter,fixed:true" width="160"><%=rb.getString("CaoZuo")%></th>
	            	</tr>
	            </thead>
	        </table>
		</div>
	</div>
	<div id="layout_center_progress" region="center" data-options="border:false, split:true" style="padding-top: 20px;background-color: #F3F3F4;"></div>
</div>

<%-- 窗口-新建任务--%>
<%-- <div id="winAddUpgradeCaTask" class="easyui-window" title="<%=rb.getString("XinJianShengJiRenWu")%>"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:860,height:600,resizable:false">
</div> --%>

<div id="caTaskToolsBar">
	<a href="javascript:void(0)" onclick="openWinAddUpgradeCaTask()" class="icon-add"></a>
</div>

<script type="text/javascript">
$(function() {
	closeLoading();
	$("#upgradeCaTaskList").datagrid({
        queryParams:{timeZone:timeZone}
    });
});
/**
 * 格式化操作
 */
function caTaskFormatter(value, rowData, rowIndex){
	var task_progress = rowData.TASK_PROGRESS;
	var task_status = rowData.TASK_STATUS;
	var task_id = rowData.TASK_ID;
	
	var JiHuo = '<%=rb.getString("JiHuo")%>';
	var GuaQi = '<%=rb.getString("GuaQi")%>';
	var ZhongZhi = '<%=rb.getString("ZhongZhi")%>';
	var ShanChu = '<%=rb.getString("ShanChu")%>';
	
	value = "";
	if(task_status != 0 && task_progress != 2){//当前不是处于激活状态，且未结束，激活图标可用
		value = value + "<div class='icon-activation' title='"+JiHuo+"' onclick='activeUpgradeCaTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='icon-activation-disabled' title='"+JiHuo+"'></div>";
	}
	
	if(task_status != 1 && task_progress != 2){//当前不是处于挂起状态，且未结束，挂起图标可用
		value = value + "<div class='icon-suspend' title='"+GuaQi+"' style='margin-left:15px;' onclick='suspendUpgradeCaTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='icon-suspend-disabled' title='"+GuaQi+"' style='margin-left:15px;'></div>";
	}
	
	if(task_progress != 2){//当前任务没有结束，终止图标可用
		value = value + "<div class='icon-termination' title='"+ZhongZhi+"' style='margin-left:15px;' onclick='terminateUpgradeCaTask(\"" + task_id + "\")'></div>";
	}else{
		value = value + "<div class='icon-termination-disabled' title='"+ZhongZhi+"' style='margin-left:15px;'></div>";
	}
	
	if(task_progress != 1){//当前任务不在进行中，删除图标可用
		value = value + "<div class='grid-del-btn-div' title='"+ShanChu+"' style='margin-left:15px;' onclick='delUpgradeCaTask(\"" + task_id + "\")'></div>";
	}else{  
		value = value + "<div class='icon-remove-disabled' title='"+ShanChu+"' style='margin-left:15px;'></div>";
	}
	return value;
}
// 显示任务进度
function showUpgradeCaTaskDetail() {
	var selectedTask = $("#upgradeCaTaskList").datagrid("getSelected");
	if (!selectedTask) {
		return;
	}
	var task_id = selectedTask["TASK_ID"];
    $("#layout_center_progress").panel({
        href: '${ctx}/task/upgradeCa/toUpgradeCaTaskProgress.action?task_id=' + task_id
    });
}

<%-- 任务列表加载完成事件，如果有数据，则默认选中第一条数据 --%>
function loadSuccessUpgradeCaTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#upgradeCaTaskList").datagrid("selectRow", 0);
	}
}

// 打开新建任务窗口
function openWinAddUpgradeCaTask() {
	var width = "${width}";
	
    /* $("#winAddUpgradeCaTask").window({
    	width: width,
    	height: document.body.clientHeight * 0.9
    }).window("center").window("open");
    
    $("#winAddUpgradeCaTask").window("refresh", "${ctx}/task/upgradeCa/goAddTask.action"); */
    var url = "${ctx}/task/upgradeCa/goAddTask.action";
    openDefaultWindow(url,{
    	title: '<%=rb.getString("XinJianShengJiRenWu")%>',
    	width: width,
    	height: document.body.clientHeight * 0.9
    });
}

// 添加任务成功后执行
function callbackForAddTask() {
	/* $("#winAddUpgradeCaTask").window("close"); */
	closeDefaultWindow();
    $("#upgradeCaTaskList").datagrid("reload");
}



// 弹出右键菜单-升级任务列表
function popRowMenu_upgradeCaTaskList(e, index, row) {
	e.preventDefault();
	if (index < 0) {
		return;
	}
	// 判断当前行是否为选中状态
	if ($("#upgradeCaTaskList").datagrid("getSelected")["TASK_ID"] != row["TASK_ID"]) {
		$("#upgradeCaTaskList").datagrid("clearSelections");
		$("#upgradeCaTaskList").datagrid("selectRow", index);
	}
	
	if (row["TASK_STATUS"] == '0') {// 当前状态为激活
		$("#rowMenu_upgradeCaTaskList").menu("disableItem", $("#rowMenu_upgradeCaTaskList [act='active']")[0]);
		$("#rowMenu_upgradeCaTaskList").menu("enableItem", $("#rowMenu_upgradeCaTaskList [act='suspend']")[0]);
	} else {// 当前状态为挂起
		$("#rowMenu_upgradeCaTaskList").menu("enableItem", $("#rowMenu_upgradeCaTaskList [act='active']")[0]);
		$("#rowMenu_upgradeCaTaskList").menu("disableItem", $("#rowMenu_upgradeCaTaskList [act='suspend']")[0]);
	}
	
	// 如果任务已结束，激活、挂起菜单均禁用
	if (row["TASK_PROGRESS"] == "2") {
		$("#rowMenu_upgradeCaTaskList").menu("disableItem", $("#rowMenu_upgradeCaTaskList [act='active']")[0]);
		$("#rowMenu_upgradeCaTaskList").menu("disableItem", $("#rowMenu_upgradeCaTaskList [act='suspend']")[0]);
	}
	
	$("#rowMenu_upgradeCaTaskList").menu("show", {
		left: e.clientX,
		top: e.clientY
	});
}

// 激活任务
function activeUpgradeCaTask(idVal) {
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingLiJiZhiXingRenWu")%>", function(r) {
        if (r) {
        	var ca_datagrid = $("#upgradeCaTaskList");
        	var params = {};
        	params["taskId"] = idVal;
        	$.post("${ctx}/task/upgradeCa/activeTask.action", params, function(data) {
        		if (data["success"]) {
        			ca_datagrid.datagrid("reload");
        		} else {
        			$.messager.alert(TiShi, data["message"]);
        		}
        	}, "json");
        }
 	});
}

// 挂起任务
function suspendUpgradeCaTask(idVal) {
	var ca_datagrid = $("#upgradeCaTaskList");
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/upgradeCa/suspendTask.action", params, function(data) {
		if (data["success"]) {
			ca_datagrid.datagrid("reload");
		} else {
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

//终止任务
function terminateUpgradeCaTask(idVal) {
	var ca_datagrid = $("#upgradeCaTaskList");
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/upgradeCa/terminateUpgradeCaTask.action", params, function(data) {
		if (data["success"]) {
			ca_datagrid.datagrid("reload");
		}else{
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

//删除任务
function delUpgradeCaTask(idVal){
	var ca_datagrid = $("#upgradeCaTaskList");
	var params = {};
	params["taskId"] = idVal;
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
        if (r) {
            $.post('${ctx}/task/upgradeCa/delUpgradeCaTask.action', params, function(data) {
                if (data["success"]) {
                	ca_datagrid.datagrid("reload");
                } else {
                    $.messager.alert(TiShi, data["message"]);
                }
            }, "json");
        }
    });
}
</script>