<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<div class="easyui-panel" data-options="border:true,fit:true" style="padding: 0 15px;">
	<table class="easyui-datagrid" id="cpeFreqLockList" fit="true" fitColumns="true" 
		   data-options="rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,toolbar:'#toolbar_cpeFreqLock',
		             url:'${ctx}/cell/CPE/queryCpeFreqLockInfosList.action?cellId=${cellId}',pagination:true,pagePosition:'bottom',
		             onBeforeLoad:cpeFreqLockListBeforeLoad,onLoadSuccess:datagridLoadSuccess">
		<thead><tr>
			<th data-options="field:'CONNECTION_STATUS',sortable:true,fixed:true,formatter:connStatusFormatter" width="30"></th>
			<th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("CPEXuLieHao")%></th>
			<th data-options="field:'MACADDRESS',sortable:true" width="80"><%=rb.getString("CPEMacAddress")%></th>
		</tr></thead>
	</table>
</div>

<div id="toolbar_cpeFreqLock" style="padding: 20px 20px 0 20px; height: 46px;">
	<input type="text" id="searchText_cpeFreqLock" class="border-box border"  placeholder="<%=rb.getString("QingShuRuCPEChaXunNeiRong")%>" style="margin-left: 25px;width:400px;"/>
	<a href="#" class="easyui-linkbutton" style="vertical-align: top; margin-left: 15px;" onclick="javascript: $('#cpeFreqLockList').datagrid('load');"><%=rb.getString("ChaXun")%></a>
</div>

<script type="text/javascript">
function cpeFreqLockListBeforeLoad() {
	if ($("#searchText_cpeFreqLock").val()) {
		param.searchText = $("#searchText_cpeFreqLock").val();
	}
}
</script>