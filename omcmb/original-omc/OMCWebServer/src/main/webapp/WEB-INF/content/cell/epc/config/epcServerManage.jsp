<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>
<div class="panelDefault">	
	<div class="singleContentDiv" style="top:0;">
		<table id="gridAuthenticationCounter" class="easyui-datagrid" fit="true"
			data-options="border:false,rownumbers:true,pagePosition:'bottom',striped:true,fitColumns:true,
			toolbar:'#toolbar_gridAuthenticationCounter',url:'${ctx}/epc/counter/getAuthenticationInfo.action?TimeZone='+timeZone,
			pagination:true,onLoadError:datagridLoadError,onBeforeLoad:beforeload_epcmonitor,onLoadSuccess: datagridLoadSuccess">
			<thead>
				<tr>
				    <th data-options="field:'epc_name'" width="50"><%=rb.getString("EPCMingChen")%></th>
				    <th data-options="field:'ip'" width="80"><%=rb.getString("IPDiZhi")%></th>
					<th data-options="field:'success'" width="50"><%=rb.getString("JianQuanChengGongShuLiang")%></th>
					<th data-options="field:'failure'" width="50"><%=rb.getString("JianQuanShiBaiShuLiang")%></th>
					<th data-options="field:'count_time'" width="100"><%=rb.getString("TongJiShiJian")%></th>
				</tr>
			</thead>
		</table>
	</div>
</div>

<%-- 工具栏 - EPC监控查询 --%>
<div id="toolbar_gridAuthenticationCounter" class="toolbarContainer">
	<div class="queryGroup">
		<input id="EPCMonitorSearch" name="task_name" placeholder="<%=rb.getString("EPCMingChen")%>&nbsp;/&nbsp;<%=rb.getString("IPDiZhi")%>" />
		<b class="el-icon el-icon-common-search" onclick="$('#gridAuthenticationCounter').datagrid('reload')"></b>
	</div>
</div>


<script type="text/javascript">
$(function() {
	closeLoading();
	$(function(){
    	$('#EPCMonitorSearch').bind('keyup',function(e){
    		if(e.keyCode == 13){
    			$('#gridAuthenticationCounter').datagrid('reload');
    		}
    	})
    });
});

//查询epc register
function beforeload_epcmonitor(params){
	params["searchText"] = $('#EPCMonitorSearch').val();
	params["timeZone"] = timeZone;
}
</script>