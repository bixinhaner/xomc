<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<!-- 右上角导出按钮 -->
<div class="omcTitleButton">
	<span class="titleButtonText"><%=rb.getString("DaoChu")%></span><span class="circleBg export_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="exportConfigBackupProgResult()"></span>
</div>

<%-- 配置备份任务进度 --%>
<div class="panelDefault">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer">
			<li class="default">${taskInfo.TASK_NAME} <%=rb.getString("ZhiXingJieGuo")%></li>
		</ul>
	</div>
	<div class="panelTableDiv" style="padding-bottom:15px;">	
		<table class="easyui-datagrid" id="configBackupTaskProg" fit="true" fitColumns="true"
			   data-options="singleSelect:true,idField : 'SERIAL_NUMBER',rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,toolbar:'#toolbar_configBackupTask'">
			<thead>
				<tr>
					<th data-options="field:'SMALL_CELL_CODE',hidden:true"></th>
					<th data-options="field:'SERIAL_NUMBER'" width="110"><%=rb.getString("XiaoZhanBianMa")%></th>
					<th data-options="field:'HOST_NAME'" width="110"><%=rb.getString("HostName")%></th>
					<th data-options="field:'ORI_VERSION'" width="150"><%=rb.getString("ChuShiBanBen")%></th>
					<th data-options="field:'PROGRESS_DETAIL'" width="350"><%=rb.getString("JinDu")%></th>
					<th data-options="field:'RUN_TIME'" width="120"><%=rb.getString("ShiJian")%></th>
					<th data-options="field:'operation',formatter: backupTaskOperFormatter,fixed:true" width="70"><%=rb.getString("CaoZuo")%></th> 
				</tr>
			</thead>
		</table>
	</div>
</div>

<form id="configBackupTaskProgResult" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="${taskInfo.TASK_ID }" name="taskId"/>
</form>

<%-- 配置备份任务进度工具栏 --%>
<div id="toolbar_configBackupTask" class="omcTableTool defaultQuery">
	<input type="text" id="searchText_configBackup" class="searchInputStyle"  placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>" style="margin-left:27px;width:400px;" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)"/>
	<b class="searchResultImgChangeStyle" onclick="queryRebootTaskInfo()"></b>
</div>

<%-- 表单-下载单个小站周期性上报日志文件--%>
<form id="formExportNvFileByCellCode" style="display:none" method="post"
      action="${ctx}/task/configBackup/doDownloadBackUpFileByCell.action">
    <%-- //已选中的小站的编码，提交表单之前为此值赋值 --%>
    <input id="backUpNVFilecellCode" name="backUpNVFilecellCode" type="hidden" value="">
</form>

<script type="text/javascript">
var configBackup_task_progress = "${taskInfo.TASK_PROGRESS}";
var timer_configBackupTaskProgReload;
$(function () {
	$("#searchText_configBackup").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryRebootTaskInfo();
		}
	}); 
	
	refreshConfigBackupProgress();
	if (timer_configBackupTaskProgReload) {
		clearInterval(timer_configBackupTaskProgReload);
		timer_configBackupTaskProgReload = undefined;
	}
	if (configBackup_task_progress != '2') {// 未结束，则定时刷新
		clearInterval(timer_configBackupTaskProgReload);
		timer_configBackupTaskProgReload = setInterval("refreshConfigBackupProgress()", 3000);
	}
});

function refreshConfigBackupProgress() {
	if ($("#configBackupTaskProg").length == 0) {
		clearInterval(timer_configBackupTaskProgReload);
		timer_configBackupTaskProgReload = undefined;
	}
	
	var param = {};
	param.timeZone=timeZone;
	if ($("#searchText_configBackup").val()) {
		param.searchText = $("#searchText_configBackup").val();
	}
	
	$.post("${ctx}/task/configBackup/getConfigBackupTaskProgress.action?task_id=${taskInfo.TASK_ID}", param, function(data) {
		$("#configBackupTaskProg").datagrid("loadData", data.grid);
		if ("2" == data.pro) {
			if (timer_configBackupTaskProgReload) {
				clearInterval(timer_configBackupTaskProgReload);
				timer_configBackupTaskProgReload = undefined;
				$("#configBackupTaskList").datagrid("reload");
			}
		}
	}, "json");
}

function exportConfigBackupProgResult() {
	var url = "${ctx}/task/configBackup/exportConfigBackupProgResult.action";
	/* $("#configBackupTaskProgResult").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone;
			param.searchText = $("#searchText_configBackup").val();
			var bool = checkParams(param)
			if(!bool) return false;
		}
	}); */
	exportByForm(url,{
		taskId: '${taskInfo.TASK_ID }',
		timeZone: timeZone,
		searchText: $("#searchText_configBackup").val()
	});
}

function queryRebootTaskInfo() {
	var param = {};
	param.timeZone=timeZone;
	if ($("#searchText_configBackup").val()) {
		param.searchText = $("#searchText_configBackup").val();
	}
	$.post("${ctx}/task/configBackup/getConfigBackupTaskProgress.action?task_id=${taskInfo.TASK_ID}", param, function(data) {
		$("#configBackupTaskProg").datagrid("loadData", data.grid);
	}, "json");
}


/**
 * 格式化设备备份进度操作列
 */
function backupTaskOperFormatter(value, rowData, rowIndex){
	var small_cell_code = rowData.SMALL_CELL_CODE;
	var progress_detail = rowData.PROGRESS_DETAIL;
	
	var isDownloadEnable = false;
	if (progress_detail == "Backup success"|| progress_detail == "备份完成") {
		isDownloadEnable = true;
	}
	
	var XiaZai = '<%=rb.getString("XiaZai")%>';
	
	var value = "<div style='margin-left: 10px' title='"+XiaZai+"'";
	
	if (!isDownloadEnable) {
		value += " class='operationDiv operation_download_disabled'";
	} else {
		value += " class='operationDiv operation_download' onclick='downloadBackUpFileByCellCode(\"" + small_cell_code + "\")'";
	}
	value += "></div>";
	return value;
}


//下载nv文件
function downloadBackUpFileByCellCode(smallCellCode) {
	$("#configBackupTaskProg").datagrid("clearSelections");
   
    $.post("${ctx}/task/configBackup/isDownloadNvFileExist.action", {"cellCode":smallCellCode}, function (data) {
        if (data["success"]) {
        	$("#backUpNVFilecellCode").val(smallCellCode);
		    /* $("#formExportNvFileByCellCode").form('submit',{
		    	onSubmit: function(param){
					var bool = checkParams(param)
					if(!bool) return false;
		    	}
		    }); */
        	exportByForm('${ctx}/task/configBackup/doDownloadBackUpFileByCell.action',{
        		backUpNVFilecellCode: smallCellCode
        	});
        } else {
       	 	$.messager.alert(TiShi, "<%=rb.getString("WenJianBuCunZai")%>");
            return;
        }
    }, "json");
}
</script>