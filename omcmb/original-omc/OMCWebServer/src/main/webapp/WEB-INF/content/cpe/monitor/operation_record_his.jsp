<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
	#toolbar_operationResultForCpe .combo{
		margin-top:-3px;
	}
</style>

<%-- CPE监控-历史（CPE状态等变化的历史记录--废弃）--%>

<table class="easyui-datagrid" id="operationRecordHisForCpe"
       data-options="fit:true,singleSelect:true,rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,
                url:'${ctx}/cell/CPE/getOperationResultForCpe.action',toolbar:'#toolbar_operationResultForCpe',
                fitColumns:true,pagination:true,onLoadSuccess:datagridLoadSuccess">
    <thead>
	    <tr>
	        <th data-options="field:'SERIAL_NUMBER'" width="150"><%=rb.getString("CPEBianMa")%></th>
	        <th data-options="field:'CPE_NAME'" width="100"><%=rb.getString("CpeName")%></th>
	        <th data-options="field:'OPERATION_TYPE'" width="150"><%=rb.getString("CaoZuoLeiXing")%></th>
	        <th data-options="field:'PROGRESS_DETAIL'" width="200"><%=rb.getString("JinDu")%></th>
	        <th data-options="field:'RUN_TIME'" width="200"><%=rb.getString("ShiJian")%></th>
	    </tr>
    </thead>
</table>

<div id="toolbar_operationResultForCpe" style="padding:20px 5px; height: auto;">
	<label><%=rb.getString("CPEBianMa")%></label>
	<input type="text" id="searchText_operationForCpe" class="border-box border" style="margin-left: 10px;"/>
	<label style="margin-left:16px;margin-right:6px;"><%=rb.getString("CaoZuoLeiXing")%></label>
	<!-- <input type="text" id="operationType" class="border-box border" style="margin-left: 5px;"/> -->
	<select id="operationType_cpe" class="easyui-combobox border border-box" style="height:26px;" data-options="editable:false">
    	<option value=""> --<%=rb.getString("QingXuanZe")%>--</option>
        <option value="ChongQi"><%=rb.getString("ChongQi")%></option>
        <option value="WanKouSheZhi"><%=rb.getString("WanKouSheZhi")%></option>
        <option value="LianJieZhuangTai"><%=rb.getString("LianJieZhuangTai")%></option>
       <%--  <option value="MMEZhuangTai"><%=rb.getString("MMEZhuangTai")%></option>
        <option value="KPIShangBaoZhuangTai"><%=rb.getString("KPIShangBaoZhuangTai")%></option>
        <option value="TongBuZhuangTai"><%=rb.getString("TongBuZhuangTai")%></option>
        <option value="JiZhanShangXian"><%=rb.getString("JiZhanShangXian")%></option>
        <option value="JiZhanXiaXian"><%=rb.getString("JiZhanXiaXian")%></option> --%>
    </select>
	<label style="margin-left:16px;margin-right:6px;"><%=rb.getString("ShiJian")%></label>
    <input id="TimeStart_cpe" class="easyui-datetimebox border-box border" style="height:26px;" data-options="editable:false">
    <span style="margin:0 6px;">-</span>
    <input id="TimeStop_cpe" class="easyui-datetimebox border-box border" style="height:26px;" data-options="editable:false">
	<a href="#" class="easyui-linkbutton" style="vertical-align: top; margin-left: 16px;" onclick="queryOperationResultForCpe()"><%=rb.getString("ChaXun")%></a>
	<a href="#" class="easyui-linkbutton" style="vertical-align: top; margin-left: 16px;" onclick="exportHistoryForCpe()"><%=rb.getString("DaoChu")%></a>
	<%-- <span><%=rb.getString("XiaoZhan")%>:</span><span style="margin-left:5px;" id="eNodeBCount">0</span>
	<c:if test="${ is_broadband == 1 }">	
		<span style="margin-left:20px;"><%=rb.getString("CPE")%>:</span><span style="margin-left:5px;" id="cpeCount">0</span>
	</c:if>	
	<span style="margin-left:20px;"><%=rb.getString("ZuiDaZaiXianShu")%>:</span><span style="margin-left:5px;" id="maxOnlintCount">0</span>
	<span style="margin-left:20px;"><%=rb.getString("DiaoXianCiShu")%>:</span><span style="margin-left:5px;" id="offlineCount">0</span>
	<span style="margin-left:20px;"><%=rb.getString("ZuiDaJiHuoShu")%>:</span><span style="margin-left:5px;" id="maxActiveCount">0</span>
	<span style="margin-left:20px;"><%=rb.getString("QuJiHuoShu")%>:</span><span style="margin-left:5px;" id="inactiveCount">0</span> --%>
</div>
<form id="exportHistoryForCpe" style="display:none" method="post" action=""></form>

<script type="text/javascript">
$(function () {
	$("#searchText_operationForCpe").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryOperationResultForCpe();
		}
	});
	
	$("#operationType_cpe").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryOperationResultForCpe();
		}
	});
	$("#operationRecordHisForCpe").datagrid({
        queryParams : {timeZone : timeZone}
    });
	
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



function queryOperationResultForCpe() {
	var sn = $("#searchText_operationForCpe").val();
	var operationType = $("#operationType_cpe").combobox('getValue');
	var timeStart = $("#TimeStart_cpe").datetimebox('getValue');
	var timeStop = $("#TimeStop_cpe").datetimebox('getValue');
	var params = {
		timeZone : timeZone,
		sn : sn,
		operationType : operationType,
		timeStart : timeStart,
		timeStop : timeStop
	};
	$("#operationRecordHisForCpe").datagrid({
		url : "${ctx}/cell/CPE/getOperationResultForCpe.action",
		queryParams : params,
		pageNumber : 1
	});
}

function exportHistoryForCpe() {
	/* $("#exportHistoryForCpe").form('submit', {
		url: "${ctx}/cell/CPE/exportOperationResultForCpe.action",
		onSubmit: function(param) {
			param.timeZone = timeZone;
			param.sn = $("#searchText_operationForCpe").val();
			param.operationType = $("#operationType_cpe").combobox('getValue');
			param.timeStart = $("#TimeStart_cpe").datetimebox('getValue');
			param.timeStop = $("#TimeStop_cpe").datetimebox('getValue');
			var bool = checkParams(param)
			if(!bool) return false;
		}
	}); */
	exportByForm("${ctx}/cell/CPE/exportOperationResultForCpe.action",{
		timeZone: timeZone,
		sn: $("#searchText_operationForCpe").val(),
		operationType: $("#operationType_cpe").combobox('getValue'),
		timeStart: $("#TimeStart_cpe").datetimebox('getValue'),
		timeStop: $("#TimeStop_cpe").datetimebox('getValue')
	});
}
</script>