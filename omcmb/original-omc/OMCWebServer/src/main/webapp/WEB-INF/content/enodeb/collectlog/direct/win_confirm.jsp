<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<%-- 窗口-日志-确认收集 --%>
<div id="winConfirmImmediateCollect" style="width: 100%;height: 100%">
	 <div class="easyui-layout" data-options="border:false, fit:true">
	 	<div region="center" data-options="border:false" style="padding: 20px">
	 		<table id="confirmImCollectCellDg" class="easyui-datagrid"
	 			data-options="border:false,fit:true,singleSelect: false,striped: true,idField:'small_cell_code',
							rownumbers:true,fitColumns:true,pagination:false,onLoadSuccess:datagridLoadSuccess">
				<thead>
					<tr>
						<th data-options="field:'small_cell_code',hidden:true"></th>
						<th data-options="field:'serial_number'" width="450"><%=rb.getString("XiaoZhanBianMa")%></th>
						<th data-options="field:'host_name'" width="450"><%=rb.getString("HostName")%></th>
					</tr>
				</thead>
	 		</table>
	 	</div>
	 	<div region="south" data-options="border:false,height:56" style="padding: 10px 20px 20px;">
	    	<a onclick="cancelImmediateCollectLogFile();" style="float:right;" class="easyui-linkbutton"><%=rb.getString("QuXiao")%></a>
	    	<a onclick="confirmImmediateCollectLogFile();" style="float:right;margin-right:20px;" class="easyui-linkbutton"><%=rb.getString("QueDing")%></a>
	 	</div>
	 </div>
</div>