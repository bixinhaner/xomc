<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<%-- 窗口-选择移动到的运营商(CELL)--%>
<div id="winSelectOperatorToMove" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding:0 20px 20px;">
			<table class="easyui-datagrid" id="gridSelectOperatorToMove"
		            data-options="singleSelect:true,fit:true,fitColumns:true,border:false,rownumbers:true,pagePosition:'bottom',toolbar:'#toolbar_gridSelectOperatorToMove',
		                 url: '${ctx}/system/operator/getOperatorListExceptself.action?no_built_in=1',pagination:true,idField:'operator_code',
		                 onBeforeLoad: tableBeforeLoadOperatorList_station,onLoadSuccess:datagridLoadSuccess">
				<thead>
					<tr>
						<th data-options="field:'operator_code',hidden:true"></th>
						<th data-options="field:'operator_name',width:100,sortable:true"><%=rb.getString("YunYingShangMingCheng")%></th>
						<th data-options="field:'cloud_key',width:100,sortable:true"><%=rb.getString("CLOUDKEY")%></th>
					</tr>
				</thead>
			</table>
		</div>
		<div region="south" data-options="border:false,height:68" style="padding: 10px 20px 20px 20px;">
			<div class="right">
				<span class="el-button el-button--primary" onclick="moveToOperator()"><%=rb.getString("QueDing")%></span>
				<span class="el-button" onclick="closeDefaultWindow();"><%=rb.getString("QuXiao")%></span>
			</div>
		</div>
		<div class="waiting"></div>
	</div>
</div>
<div id="toolbar_gridSelectOperatorToMove" class="toolbarContainer">
	<div class="queryGroup" style="margin-right:0;">
		<input id="operatorCode_station" style="width: 260px" placeholder="<%=rb.getString("QingShuRu")%><%=rb.getString("YunYingShangMingCheng")%>">
		<b onclick="javascript: $('#gridSelectOperatorToMove').datagrid('load');"></b>
	</div>
</div>
<script>
	function tableBeforeLoadOperatorList_station(param){
		var searchOperatorCode = $("#operatorCode_station").val();
		var selOperator = $("#tableOperatorList").datagrid("getSelected");
		if (selOperator) {
			param["operator_code_current"] = selOperator["operator_code"];
		}
		if (searchOperatorCode) {
			param["operator_code"] = searchOperatorCode;
		}
	}
</script>