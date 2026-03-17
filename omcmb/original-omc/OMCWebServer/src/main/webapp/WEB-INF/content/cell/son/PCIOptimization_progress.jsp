<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%-- PCI优化任务进度 --%>
<div class="easyui-panel" data-options="border:false,fit:true" style="padding: 0 15px;">
	<div data-options="region:'north',border:false,height: 40,collapsible:false" style="margin-bottom:20px;">
		        <div class="omcPageTitleDiv">
					<ul class="omcPageTitleContainer">
						<li class="default"><%=rb.getString("PCIYouHuaXiangQing")%></li>
					</ul>
				</div>
	</div>
	<table class="easyui-datagrid" id="PCIOptimizationProgress" fit="true" fitColumns="true"
		   data-options="singleSelect:true,rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess,
		   			url:'${ctx}/cell/SON/getPCIOptimizationProgress.action',queryParams:{timeZone:timeZone}">
		<thead>
		<tr>
		    <th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("Serial_Number") %></th>
			<th data-options="field:'HOST_NAME'" width="100"><%=rb.getString("HostName")%></th>
			<th data-options="field:'CELL_ID'" width="100"><%=rb.getString("XIAOQUID")%></th>
			<th data-options="field:'CURRENT_PCI'" width="100"><%=rb.getString("DangQianPCI")%></th>
			<th data-options="field:'PCI_LIST'" width="100">pciList</th>
			<th data-options="field:'SUGGESTED_PCI'" width="100"><%=rb.getString("JianYiPCI")%></th>
			<th data-options="field:'PROGRESS_DETAIL'" width="400"><%=rb.getString("JinDu")%></th>
			<th data-options="field:'OPTIMIZE_TIME'" width="100"><%=rb.getString("KaiShiShiJian")%></th>
		</tr>
		</thead>
	</table>
</div>

<script type="text/javascript">
var timer_autoPCIOptimizationReload;

$(function () {
	timer_autoPCIOptimizationReload = setInterval("refreshProgress()", 1000);
});

function refreshProgress(){
	if ($("#PCIOptimizationProgress").length == 0) {
		clearInterval(timer_autoPCIOptimizationReload);
	}
	
	$.post("${ctx}/cell/SON/getPCIOptimizationProgress.action", {timeZone:timeZone}, function(data) {
		$("#PCIOptimizationProgress").datagrid("loadData", data);
	}, "json");
}
</script>