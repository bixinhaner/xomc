<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 自动任务界面 --%>
<div id="autoTask" style="width: 100%;height: 100%;min-height: 700px;">
	<div class="splitPanel" style="height:49%;">
		<div class="tabsTitle">
			<span tabtit="autoTaskDiv" class="active eNbAutoSoftwareUpgrade hidden visible" logtype="autoTask"><%=rb.getString("ZiDongRenWu")%></span>
			<span tabtit="verUpgradeDiv" class='eNbAutoSoftwareUpgrade hidden visible' logtype="verUpgrade"><%=rb.getString("RuanJianShengJi")%><%=rb.getString("KongGe")%><%=rb.getString("ChuShiBanBen")%></span>
		</div>
		<div class="tabsContentDiv">
			<div class='autoTaskDiv'> 
				<table id="autoTaskList"></table>
			</div>
			<div class='verUpgradeDiv'>
				<table id="oriSoftwareVerTable"></table>
			</div>
		</div>
	</div>
	<div class="splitPanel"  style="height:49%;margin-top: 15px;">
		<div id="autoTask_detail" style="height:100%;position:relative;"></div>
	</div>
</div>
<%-- 初始软件版本工具栏 --%>
<div id="toolbar_oriSoftwareVerTable" class="toolbarContainer eNbAutoSoftwareUpgrade hidden">
	<label><%=rb.getString("TianJia")%><%=rb.getString("KongGe")%><%=rb.getString("ChuShiBanBen") %></label>
	<input id="txtVerName_addOriSoftwareVer" class="border-box border" style="width:400px;margin-left:6px;"
		placeholder="<%=rb.getString("QingShuRuXuYaoTianJiaDeBanBen")%>" />
	<a class="linkbutton" onclick="addOriSoftwareVer()" style="vertical-align:middle;margin-left:16px;"><span><%=rb.getString("TianJia")%></span></a>
</div>
 <%-- 自动任务界面工具栏 --%>
<%--<div id="toolbar_autoTaskList" class="omcTableTool"></div> --%>
<script>
$(function () {
	// 定义任务列表
	$("#autoTaskList").datagrid({
		url : "${ctx}/cell/selfstart/queryAutoTaskList.action",
		queryParams:{timeZone:timeZone},
		singleSelect : true,
		fit : true,
		rownumbers:true,
		fitColumns : true,
		/* toolbar:'#toolbar_autoTaskList', */
		border : false,
		striped: true,
		idField: 'task_id',
		columns: [[
			{field: 'task_id', hidden: true},
			{field: 'name_s', width: 150, title: '<%=rb.getString("MingCheng")%>'},
			{field: 'switch', width: 150, title: '<%=rb.getString("ZhuangTai")%>', formatter: statusFmt_autoTask},
			{field: 'update_time', width: 150, title: '<%=rb.getString("GengXinShiJian")%>'},
			{field: 'detail', hidden: true},
			{field: 'd', width: 150, fixed: true, title: '<%=rb.getString("CaoZuo")%>', formatter: operFmt_autoTask},
		]],
		onLoadSuccess: loadSuc_autoTaskList,
		onSelect: onSelect_autoTaskList
	});
	// 加载初始software版本表格
	$("#oriSoftwareVerTable").datagrid({
		url: "${ctx}/cell/version/getOriSoftwareVer.action",
		queryParams:{timeZone:timeZone},
		singleSelect: true,
		fit: true,
		rownumbers:true,
		fitColumns: true,
		toolbar:'#toolbar_oriSoftwareVerTable',
		border: false,
		striped: true,
		idField: 'ver_name',
		onLoadSuccess: oriSoftwareVer_load_success,
		columns: [[
			{field: 'ver_name', width: 100, title: '<%=rb.getString("ChuShiBanBen")%><%=rb.getString("KongGe")%><%=rb.getString("MingCheng")%>'},
			{field: 'add_time', width: 100, title: '<%=rb.getString("TianJia")%><%=rb.getString("KongGe")%><%=rb.getString("ShiJian")%>'},
			{field: 'd', width: 100, title: '<%=rb.getString("CaoZuo")%>', fixed: true, formatter: function(value, row, index) {
				var ver_name = row.ver_name;
				if(ver_name=="'"){
					ver_name="&#39";
				}
				var b = "<div class='operationDiv operation_delete eNbAutoSoftwareUpgrade hidden' style='margin-left:25px' title='<%=rb.getString("ShanChu")%>' onclick='delOriSoftwareVer(\"" + encodeURI(ver_name) + "\")'></div>";
				return b;
			}}
		]]
	});
})

// 自动任务表格 - 状态格式化
function statusFmt_autoTask(value, row, index) {
	if (value == "1") {
		return "<%=rb.getString("JinXingZhong")%>";
	} else {
		return "<%=rb.getString("GuanBi")%>";
	}
}

// 自动任务表格 - 操作格式化
function operFmt_autoTask(value, row, index) {
	var _switch = row["switch"];
	var b = "";
	if ("1" == _switch) {// 打开状态，显示停止按钮
		b += "<div class='operationDiv operation_terminate eNbAutoSoftwareUpgrade hidden' title='<%=rb.getString("TingZhi")%>' onclick='disableTask_autoTask(\"" + row.task_id + "\")'></div>";
	} else {
		b += "<div class='operationDiv operation_active eNbAutoSoftwareUpgrade hidden' title='<%=rb.getString("KaiShi")%>' onclick='enableTask_autoTask(\"" + row.task_id + "\")'></div>";
	}

	if (row.name == "ParamZiDongPeiZhi") {
		if ("${paramTaskEditable}" == "1") {
			b += "<div class='operationDiv operation_edit eNbAutoSoftwareUpgrade hidden' style='margin-left: 15px' title='<%=rb.getString("XiuGai")%>' onclick='openWinEditTask(\"" + index + "\")'></div>";
		}
	} else {
		b += "<div class='operationDiv operation_edit eNbAutoSoftwareUpgrade hidden' style='margin-left: 15px' title='<%=rb.getString("XiuGai")%>' onclick='openWinEditTask(\"" + index + "\")'></div>";
	}

	return b;
}

// 启用任务
function enableTask_autoTask(taskId) {
	//自启动任务为空，不让启用任务
	$.post("${ctx}/cell/selfstart/isAutoTaskEmpity.action", {"taskId": taskId}, function(data) {
		if (data.success) {
			$.messager.alert(TiShi, "<%=rb.getString("RenWuWeiKongBuNengQiDong") %>");
		} else {
			$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenKaiShiRenWu")%>", function(r) {
				if (r) {
					$.post("${ctx}/cell/selfstart/enableTask.action", {"taskId": taskId, "timeZone": timeZone}, function(data) {
						if (data.success) {
							$("#autoTaskList").datagrid("reload");
						} else {
							$.messager.alert(TiShi, "<%=rb.getString("RenWuWeiKongBuNengQiDong") %>");
						}
					}, "json");
				}
			}).addClass("normalConfirm");
		}
	}, "json");
	

}

// 停用任务
function disableTask_autoTask(taskId) {
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenTingZhiRenWu")%>", function(r) {
		if (r) {
			$.post("${ctx}/cell/selfstart/disableTask.action", {"taskId": taskId, "timeZone": timeZone}, function(data) {
				if (data.success) {
					$("#autoTaskList").datagrid("reload");
				} else {
					$.messager.alert(TiShi, data.message);
				}
			}, "json");
		}
	}).addClass("normalConfirm");
}

// 自动任务表格-加载完成事件
function loadSuc_autoTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	var selRow = $("#autoTaskList").datagrid("getSelected");
	if (!selRow) {// 无选中项，选中第一行
		$("#autoTaskList").datagrid("selectRow", 0);
	} else {
		loadProgress(selRow["task_id"]);
	}
}

// 自动任务表格-选择行事件-显示任务进度
function onSelect_autoTaskList(index, row) {
	loadProgress(row.task_id);
}

// 加载进度信息
function loadProgress(task_id) {
	var url = "${ctx}/cell/selfstart/toSelfStartPro.action?taskId=" + task_id;
	$("#autoTask_detail").panel({
    	border:false,
        href: url
    });
}

//打开窗口 - 新建升级任务
<%-- function openWinAddSoftwareTask() {
	var url = "${ctx}/cell/selfstart/toAddSoftwareTask.action";
	openDefaultWindow(url,{
		title: '<%=rb.getString("XinJianZiDongShengJiRenWu")%>',
		width: 860,
		height: document.body.clientHeight * 0.75
	});
} --%>

// 打开窗口 - 编辑任务
function openWinEditTask(index) {
	var task = $("#autoTaskList").datagrid("getData").rows[index];
	if (task.name == "UbootZiDongShengJi") {// uboot
		/* $("#winEditUbootTask").window({
			width: 860,
			height: document.body.clientHeight * 0.75

		}).window("center").window("open");
		$("#winEditUbootTask").attr("detail", task.detail);
		$("#winEditUbootTask").window("refresh", "${ctx}/cell/selfstart/toEditUbootTask.action"); */
		$(winDefaultSelector).attr("detail", task.detail);
		var url = "${ctx}/cell/selfstart/toEditUbootTask.action";
		openDefaultWindow(url,{
			title: '<%=rb.getString("UbootZiDongShengJi")%>',
			width:860,height:600
		});
	} /* else if (task.name == "SoftwareZiDongShengJi") {// software
		$("#winEditSoftwareTask").window({
			width: 860,

			height: document.body.clientHeight * 0.75

		}).window("center").window("open");
		$("#winEditSoftwareTask").attr("detail", task.detail);
		$("#winEditSoftwareTask").window("refresh", "${ctx}/cell/selfstart/toEditSoftwareTask.action");
	} */ else if (task.name.indexOf("SoftwareZiDongShengJi") > -1) {// software
		var tempTitleName = "<%= rb.getString("SoftwareZiDongShengJi") %>";
		var taskNameLength = task.name.length;
		/* $("#winEditSoftwareTask").window({
			width: 860,
			height: document.body.clientHeight * 0.75,
			title: tempTitleName + task.name.substring(21, taskNameLength)

		}).window("center").window("open");
		$("#winEditSoftwareTask").attr("detail", task.detail);
		$("#winEditSoftwareTask").window("refresh", "${ctx}/cell/selfstart/toEditSoftwareTask.action?taskId=" + task.task_id+"&timeZone=" + timeZone); */
		var url = "${ctx}/cell/selfstart/toEditSoftwareTask.action?taskId=" + task.task_id+"&timeZone=" + timeZone,
			options = {
				width: 860,
				height: document.body.clientHeight * 0.75,
				title: tempTitleName + task.name.substring(21, taskNameLength)
			};
		$(winDefaultSelector).attr("detail", task.detail);
		openDefaultWindow(url,options);
	}else if (task.name == "ParamZiDongPeiZhi") {// param
		/* $("#winEditParamTask").window({
			width: 520,
			height: 570
		}).window("center").window("open");
		$("#winEditParamTask").attr("detail", task.detail);
		$("#winEditParamTask").window("refresh", "${ctx}/cell/selfstart/toEditParamTask.action"); */
		$(winDefaultSelector).attr("detail", task.detail);
		var url = "${ctx}/cell/selfstart/toEditParamTask.action";
		openDefaultWindow(url,{
			title: '<%=rb.getString("ParamZiDongPeiZhi")%>',
			width:600,height:570
		});
	} else if (task.name == "ConfigFileZiDongShengJi") {// config file
		/* $("#winEditConfigFileTask").window("open").window("center");
		$("#winEditConfigFileTask").window("refresh", "${ctx}/cell/selfstart/toEditConfigFileTask.action"); */
		var url = "${ctx}/cell/selfstart/toEditConfigFileTask.action";
		openDefaultWindow(url,{
			title: '<%=rb.getString("ConfigFileZiDongShengJi")%>',
			width:600,height:570
		});
	}
}

//重新加载自动任务列表
function reloadAutoTaskList() {
	$("#autoTaskList").datagrid("reload");
}
function oriSoftwareVer_load_success() {
	$(this).datagrid("enableContextmenuAutoSize");
}

// 删除原始版本
function delOriSoftwareVer(ver_name) {
	if (!ver_name) {
		return;
	}
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChu")%>", function(r) {
		if (r) {
			$.post("${ctx}/cell/version/delOriSoftwareVer.action", {"ver_name": decodeURI(ver_name)}, function(data){
				if (data.success) {
					$("#oriSoftwareVerTable").datagrid("reload");
				} else {
					$.messager.alert(TiShi, data.message);
				}
			}, "json");
		}
	}).addClass("seriousConfirm");
}

// 添加版本
function addOriSoftwareVer() {
	var ver_name = $("#txtVerName_addOriSoftwareVer").val();
	if (!ver_name) {
		return;
	}

	$.post("${ctx}/cell/version/addOriSoftwareVer.action", {"ver_name": ver_name}, function(data) {
		if (data.success) {
			$("#oriSoftwareVerTable").datagrid("reload");
		} else {
			$.messager.alert(TiShi, data.message);
		}
	}, "json");
}
</script>