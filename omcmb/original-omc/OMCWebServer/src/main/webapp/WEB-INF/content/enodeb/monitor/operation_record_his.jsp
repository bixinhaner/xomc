<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
	#toolbar_operationResultForeNodeB .combo{
		margin-top:-3px;
	}
</style>

<%-- 基站监控-历史（基站状态等变化的历史记录）--%>
<table class="easyui-datagrid" id="operationRecordHisForeNodeB" fit="true"
        data-options="border:false,singleSelect:true,rownumbers:true,fitColumns:true,striped:true,onLoadError:datagridLoadError,
                 url:'${ctx}/cell/cpeinfos/getOperationResultForeNodeB.action',toolbar:'#toolbar_operationResultForeNodeB',
                 pagination:true,onLoadSuccess:datagridLoadSuccess">
     <thead>
	     <tr>
	         <th data-options="field:'SERIAL_NUMBER'" width="150"><%=rb.getString("XiaoZhanBianMa")%></th>
	         <th data-options="field:'HOST_NAME'" width="100"><%=rb.getString("HostName")%></th>
	         <th data-options="field:'OPERATION_TYPE'" width="150"><%=rb.getString("CaoZuoLeiXing")%></th>
	         <th data-options="field:'PROGRESS_DETAIL'" width="200"><%=rb.getString("JinDu")%></th>
	         <th data-options="field:'RUN_TIME'" width="200"><%=rb.getString("ShiJian")%></th>
	     </tr>
     </thead>
</table>

<div id="toolbar_operationResultForeNodeB" style="padding:20px 5px; height: auto;">
	<label><%=rb.getString("XiaoZhanBianMa")%></label>
	<input type="text" id="searchText_operationForeNodeB" class="border-box border" style="margin-left: 10px;"/>
	<label style="margin-left:16px;margin-right:6px;"><%=rb.getString("CaoZuoLeiXing")%></label>
	<!-- <input type="text" id="operationType" class="border-box border" style="margin-left: 5px;"/> -->
	<select id="operationType_enodeb" class="easyui-combobox border border-box" style="height:26px;" data-options="editable:false">
    	<option value=""> --<%=rb.getString("QingXuanZe")%>--</option>
        <option value="ChongQi"><%=rb.getString("ChongQi")%></option>
        <option value="ShiFouJiHuo"><%=rb.getString("ShiFouJiHuo")%></option>
        <c:if test="${isElfCell == '0'}">
        	 <option value="MMEZhuangTai"><%=rb.getString("MMEZhuangTai")%></option>
        </c:if>
        <option value="KPIShangBaoZhuangTai"><%=rb.getString("KPIShangBaoZhuangTai")%></option>
        <c:if test="${isElfCell == '0'}">
        	<option value="TongBuZhuangTai"><%=rb.getString("TongBuZhuangTai")%></option>
        </c:if>
        <option value="JiZhanShangXian"><%=rb.getString("JiZhanShangXian")%></option>
        <option value="JiZhanXiaXian"><%=rb.getString("JiZhanXiaXian")%></option>
    </select>
	<label style="margin-left:16px;margin-right:6px;"><%=rb.getString("ShiJian")%></label>
    <input id="TimeStart_enodeb" class="easyui-datetimebox border-box border" style="height:26px;" data-options="editable:false">
    <span style="margin:0 6px;">-</span>
    <input id="TimeStop_enodeb" class="easyui-datetimebox border-box border" style="height:26px;" data-options="editable:false">
	<a href="#" class="easyui-linkbutton" style="vertical-align: top; margin-left: 16px;" onclick="queryOperationResultForeNodeB()"><%=rb.getString("ChaXun")%></a>
	<a href="#" class="easyui-linkbutton" style="vertical-align: top; margin-left: 16px;" onclick="exportHistory()"><%=rb.getString("DaoChu")%></a>
</div>

<form id="operationResultForeNodeB" style="display:none" method="post" action=""></form>

<script type="text/javascript">
$(function () {
	$("#searchText_operationForeNodeB").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryOperationResultForeNodeB();
		}
	});
	
	$("#operationType_enodeb").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryOperationResultForeNodeB();
		}
	});
	
	$("#operationRecordHisForeNodeB").datagrid({
        queryParams : {timeZone : timeZone}
    });
	
	/* if (timer_operationResultForeNodeBReload) {
		clearInterval(timer_operationResultForeNodeBReload);
		timer_operationResultForeNodeBReload = undefined;
	}
	timer_operationResultForeNodeBReload = setInterval("refreshProgress()", 3000); */
	
	//查询当前运营商所拥有的基站和CPE数量
	$.post("${ctx}/cell/cpeinfos/getOperatoreNodeBAndCpeCounts.action", {}, function(data) {
		$("#eNodeBCount").text(data["eNodeBCounts"]);
		$("#cpeCount").text(data["cpeCounts"]);
		$("#maxOnlintCount").text(data["maxOnlineNumber"]);
		$("#maxActiveCount").text(data["maxActiveNumber"]);
		$("#offlineCount").text(data["offlineNumber"]);
		$("#inactiveCount").text(data["inactiveNumber"]);
	}, "json");
});

function refreshProgress() {
	if ($("#operationRecordHisForeNodeB").length == 0) {
		clearInterval(timer_operationResultForeNodeBReload);
		timer_operationResultForeNodeBReload = undefined;
	}
	
	var sn = $("#searchText_operationForeNodeB").val();
	var operationType = $("#operationType_enodeb").combobox('getValue');
	var timeStart = $("#TimeStart_enodeb").datetimebox('getValue');
	var timeStop = $("#TimeStop_enodeb").datetimebox('getValue');
	var params = {
		timeZone : timeZone,
		sn : sn,
		operationType : operationType,
		timeStart : timeStart,
		timeStop : timeStop
	};
	
	$.post("${ctx}/cell/cpeinfos/getOperationResultProgressForeNodeB.action", params, function(data) {
		$("#autoUpgradTaskProgress").datagrid("loadData", data);
	}, "json");
}

function queryOperationResultForeNodeB() {
	var sn = $("#searchText_operationForeNodeB").val();
	var operationType = $("#operationType_enodeb").combobox('getValue');
	var timeStart = $("#TimeStart_enodeb").datetimebox('getValue');
	var timeStop = $("#TimeStop_enodeb").datetimebox('getValue');
	var params = {
		timeZone : timeZone,
		sn : sn,
		operationType : operationType,
		timeStart : timeStart,
		timeStop : timeStop
	};
	$("#operationRecordHisForeNodeB").datagrid({
		url : '${ctx}/cell/cpeinfos/getOperationResultForeNodeB.action',
		queryParams : params,
		pageNumber : 1
	});
}

function exportHistory() {
	//var url = "${ctx}/cell/cpeinfos/exportOperationResultForeNodeB.action";
	var url = "${ctx}/cell/cpeinfos/exportOperationResultForeNodeBToCSV.action";
	/* $("#operationResultForeNodeB").form('submit', {
		url: url,
		onSubmit: function(param) {
			param.timeZone=timeZone,
			param.sn = $("#searchText_operationForeNodeB").val();
			param.operationType = $("#operationType_enodeb").combobox('getValue');
			param.timeStart = $("#TimeStart_enodeb").datetimebox('getValue');
			param.timeStop = $("#TimeStop_enodeb").datetimebox('getValue');
			var bool = checkParams(param)
			if(!bool) return false;
		}
	}); */
	exportByForm(url,{
		timeZone: timeZone,
		sn: $("#searchText_operationForeNodeB").val(),
		operationType: $("#operationType_enodeb").combobox('getValue'),
		timeStart: $("#TimeStart_enodeb").datetimebox('getValue'),
		timeStop: $("#TimeStop_enodeb").datetimebox('getValue')
	});
}
</script>