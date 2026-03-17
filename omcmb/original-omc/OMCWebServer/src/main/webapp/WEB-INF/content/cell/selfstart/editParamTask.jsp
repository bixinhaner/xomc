<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 编辑参数配置自动任务-只用于普通基站参数配置自动任务 --%>
<style type="text/css">
.edit-param-task {
	width: 250px;
}

#editParamTask .itemDiv {
	height: 60px;
}

#editParamTask .itemDiv > span:nth-child(1) {
	width: 100px;
}
</style>

<div id="editParamTask" class="easyui-layout" data-options="fit:true">
	<div region="center" data-options="border:false" style="padding: 20px">
		<div class="itemDiv">
			<span>SN</span>
			<input type="text" class="border border-box edit-param-task" value="ALL" disabled="disabled"/>
		</div>
		<div class="itemDiv">
			<span>eNodeB ID</span>
			<input id="enbId_min" type="text" class="easyui-numberspinner border border-box item" value="${taskInfo.enbId_min}" data-options="min:1" style="height: 26px;width:150px;"/>
			----
			<input id="enbId_max" type="text" class="easyui-numberspinner border border-box item" value="${taskInfo.enbId_max}" data-options="min:1" style="height: 26px;width:150px;"/>
		</div>
		<div class="itemDiv">
			<span>MME IP</span>
			<input id="mmeIp" type="text" class="border border-box item edit-param-task" value="${taskInfo.mmeIp}"/>
		</div>
		<div class="itemDiv">
			<span>Band</span>
			<input id="_band" type="text" class="border border-box item edit-param-task" value="${taskInfo._band}"/>
		</div>
		<div class="itemDiv">
			<span>EARFCN</span>
			<input id="earfcn" type="text" class="border border-box item edit-param-task" value="${taskInfo.earfcn}"/>
		</div>
		<div class="itemDiv">
			<span>Rem Band</span>
			<input id="remBand" type="text" class="border border-box item edit-param-task" value="${taskInfo.remBand}"/>
		</div>
		<div class="itemDiv">
			<span>Rem EARFCN</span>
			<input id="remEarfcn" type="text" class="border border-box item edit-param-task" value="${taskInfo.remEarfcn}"/>
		</div>
	</div>
	<div region="south" data-options="border:false,height:47" style="padding: 10px 20px;">
		<a class="easyui-linkbutton" style="float:right;" onclick="closeWinEditParamAutoTask()"><%=rb.getString("QuXiao")%></a>
		<a class="easyui-linkbutton" style="float:right;margin-right: 15px;" onclick="saveParamAutoTask()"><%=rb.getString("QueDing")%></a>
	</div>
</div>

<script type="text/javascript">
$(function() {
});

// 关闭窗口
function closeWinEditParamAutoTask() {
	/* $("#winEditParamTask").window("close"); */
	closeDefaultWindow();
}

// 保存任务信息
function saveParamAutoTask() {
	var param = {};
	param["enbId_min"] = $("#enbId_min").numberspinner("getValue");
	param["enbId_max"] = $("#enbId_max").numberspinner("getValue");
	param["mmeIp"] = $("#mmeIp").val();
	param["_band"] = $("#_band").val();
	param["earfcn"] = $("#earfcn").val();
	param["remBand"] = $("#remBand").val();
	param["remEarfcn"] = $("#remEarfcn").val();

	$.post("${ctx}/cell/selfstart/saveParamAutoTask.action", param, function(data) {
		if (data.success) {
			closeWinEditParamAutoTask();
			$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("ChengGong")%>");
		} else {
			$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
		}
	}, "json");
}
</script>