<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<script type="text/javascript" >
	var ctx = "${ctx}";
</script>

<style>
#cpeInfos .datagrid-header-rownumber,.datagrid-cell-rownumber{
   width:40px;
   text-align: center;
   margin:0;
   padding:0;
 }
</style>


<%-- 展示基站下CPE列表--%>
<div id="cpeInfos" class="easyui-layout" data-options="fit:true">
	<div class="easyui-layout" data-options="border:false,fit:true" >
		<div region="center" data-options="border:false" style="padding-top:15px">
				<%-- CPE列表 --%>
				<table class="easyui-datagrid" id="tableHomeCpeListSimple" fit="true" data-options="fitColumns:true,border:false,singleSelect:true,
                    rownumbers:true,url:'${ctx}/cell/CPE/queryCpeInfosList.action?TimeZone='+timeZone+'&cellID='+${cellID},pageSize:${pageSize},pageList:${pageList},striped:true,
                    pagination:true,pagePosition:'bottom',idField:'CPE_CODE',singleSelect:true,onLoadSuccess:datagridLoadSuccess">
				<thead>
					<tr>
						<th data-options="field:'CONNECTION_STATUS',sortable:true,fixed:true,formatter:connStatusFormatter" width="30"></th>
						<th data-options="field:'CPE_CODE',hidden:true"><%=rb.getString("XiaoZhanBianMa")%></th>
						<th data-options="field:'SERIAL_NUMBER',sortable:true" width="200"><%=rb.getString("CPEXuLieHao")%></th>
						<th data-options="field:'CPE_NAME',editor:'text'" width="100"><%=rb.getString("CPEName")%></th>
						<th data-options="field:'MACADDRESS',sortable:true" width="200"><%=rb.getString("CPEMacAddress")%></th>
						<th data-options="field:'IMSI',sortable:true" width="200">IMSI</th>
					</tr>
				</thead>
			</table>
		</div>
	</div>
</div>	

<input type="hidden" id="cellID" value="" />

<script type="text/javascript">
<%-- 加载完成事件 --%>
$(function() {
	closeLoading();
});
</script>
