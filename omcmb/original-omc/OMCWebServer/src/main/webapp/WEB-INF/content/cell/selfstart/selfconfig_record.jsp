<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<div class="easyui-layout" data-options="fit:true,border:false">
	<div region="north" data-options="border:false,height:73,border:false,collapsible:false" style="padding:20px;">
		<div class="queryGroup">
			<input id="txtSearchEnb" placeholder="<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>" />
			<b onclick="javascript: $('#tableSelfStartRecordList').datagrid('load');"></b>
		</div>
	</div>
	<div region="center" data-options="border:false">
		<table class="easyui-datagrid" id="tableSelfStartRecordList" fit="true" data-options="border:false,fitColumns:true,
	                  rownumbers:true,url:'${ctx}/cell/selfstart/querySelfstartInfoAndProgress.action',queryParams:{timeZone:timeZone},
	                  striped:true,pagination:true,pagePosition:'bottom',onLoadError:datagridLoadError,onBeforeLoad:selfstartRecordBeforeLoad,onLoadSuccess:datagridLoadSuccess">
            <thead>
	            <tr>
	            	<th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
	            	<th data-options="field:'HOST_NAME'" width="100"><%=rb.getString("HostName")%></th>
	                <th data-options="field:'CONFIG_STATUS',formatter:configFormatter" width="100"><%=rb.getString("ZiPeiZhiJinDu")%></th>
	                <th data-options="field:'CONFIG_TIME'" width="100"><%=rb.getString("WanChengShiJian")%></th>
	            </tr>
            </thead>
        </table>
	</div>
</div>

<script type="text/javascript">
$(function () {
	<%--回车事件--%>
	$("#txtSearchEnb").bind("keyup", function(e){
		if (e.keyCode == 13){
			$('#tableSelfStartRecordList').datagrid('load');
		}
	});
})

function configFormatter(value, rowData, rowIndex) {
	if(value == "1") {
		return "<div><%=rb.getString("PeiZhiChengGong")%><div>";
	} else if(value == "2") {
		return "<div><%=rb.getString("PeiZhiShiBai")%><div>";
	} else {
		return "<div><%=rb.getString("PeiZhiWeiXiaFa")%><div>";
	}
}

function selfstartRecordBeforeLoad(param) {
	var searchText = $("#txtSearchEnb").val();
	if(searchText) {
		param["search_text"] = searchText;
	}
}
</script>