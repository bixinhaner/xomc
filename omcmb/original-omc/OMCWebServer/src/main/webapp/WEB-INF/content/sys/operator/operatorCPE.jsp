<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<%-- 窗口-选择移动到的运营商(CPE)--%>
<div id="winSelectCpeOperatorToMove" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding:0 20px 20px;">
			<table class="easyui-datagrid" id="gridSelectCpeOperatorToMove" data-options="singleSelect:true,fit:true,fitColumns:true,border:false,rownumbers:true,pagePosition:'bottom',
                 url: '${ctx}/system/operator/getOperatorListExceptself.action?no_built_in=1',pagination:true,idField:'operator_code',toolbar:'#toolbar_gridSelectCpeOperatorToMove',
                 onBeforeLoad: tableBeforeLoadOperatorList_cpe,onLoadSuccess:datagridLoadSuccess">
				<thead>
					<tr>
						<th data-options="field:'operator_code',hidden:true"></th>
						<th data-options="field:'operator_name',width:100,sortable:true"><%=rb.getString("YunYingShangMingCheng")%></th>
						<th data-options="field:'cloud_key',width:100,sortable:true"><%=rb.getString("CLOUDKEY")%></th>
					</tr>
				</thead>
			</table>
		</div>
		<div region="south" data-options="border:false,height:68" style="padding:10px 20px 20px;">
			<div class="right">
				<span class="el-button el-button--primary" onclick="moveCpeToOperator()"><%=rb.getString("QueDing")%></span>
				<span class="el-button" onclick="closeDefaultWindow();"><%=rb.getString("QuXiao")%></span>
			</div>
		</div>
		<div class="waiting"></div>
	</div>	
</div>
<div id="toolbar_gridSelectCpeOperatorToMove" class="toolbarContainer">
	<div class="queryGroup" style="margin-right:0;">
		<input id="operatorCode_cpe" style="width: 260px" placeholder="<%=rb.getString("QingShuRu")%><%=rb.getString("YunYingShangMingCheng")%>">
		<b onclick="javascript: $('#gridSelectCpeOperatorToMove').datagrid('load');"></b>
	</div>
</div>
<script>
	//CPE移动到运营商窗口  ，运营商名称列表加载前事件 
	function tableBeforeLoadOperatorList_cpe(param){
		var selOperator = $("#tableOperatorList").datagrid("getSelected");
		if (selOperator) {
			param["operator_code_current"] = selOperator["operator_code"];
		}
		
		var searchOperatorCode = $("#operatorCode_cpe").val();
		if (searchOperatorCode) {
			param["operator_code"] = searchOperatorCode;
		}
	}
</script>