<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<script type="text/javascript">
	var ctx = "${ctx}";
	var TiShi = "<%=rb.getString("TiShi")%>";
</script>
<script type="text/javascript" src="${ctx}/js/bi/cell/PCIDetection.js"></script>
<%-- 右键菜单-PCI列表 --%>
<div id="rowMenu_TablePCIDetection" class="easyui-menu" style="width: 150px;">
	<div onclick="resolveConfirm()"><%=rb.getString("JieJue")%></div>
</div>
<div id="winConfirmResolve" class="easyui-window" title="<%=rb.getString("JieJueFangAn")%>"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:500,height:400"
		style="padding: 10px;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:true,fit:true" style="padding: 10px;">
			<form id="resolveForm">
			<div style="border-style: none;height: 30px;">
				<lable for="txt_cellId">CellId:</lable>
				<input id="txt_cellId" type="text" name="cellId" readonly="readonly" class="border border-box"/>
			</div><div style="border-style: none;height: 30px;">
				<lable for="txt_N_cellId">Neighbor CellId:</lable>
				<input id="txt_N_cellId" type="text" name="n_cellId" readonly="readonly" class="border border-box"/>
			</div><div style="border-style: none;height: 30px;">
				<lable for="txt_NeighborIndex">Neighbor Index:</lable>
				<input id="txt_NeighborIndex" type="text" name="neighborIndex" readonly="readonly" class="border border-box"/>
			</div><div style="border-style: none;height: 30px;">
				<lable for="txt_N_CurrentPCI">Neighbor Current PCI:</lable>
				<input id="txt_N_CurrentPCI" type="text" name="n_currentPCI" readonly="readonly" class="border border-box"/>
			</div><div style="border-style: none;height: 30px;">
				<lable for="txt_N_NewPCI">Neighbor New PCI:</lable>
				<input id="txt_N_NewPCI" type="text" name="n_newPCI" readonly="readonly" class="border border-box"/>
			</div>
			</form>
		</div>
		<div region="south" data-options="border:true,height: 45" style="text-align: right;padding: 10px;border-width: 1px 0 0 0">
			<a id="btnHomeEditCell" class="easyui-linkbutton" onclick="resolveConflict()" style="width: 60px;"><%=rb.getString("QueDing")%></a>
		</div>
	</div>
</div>
<%-- PCI列表 --%>
<div class="easyui-layout" data-options="fit:true,border:false">
	<div region="center" data-options="border:false">
		<table class="easyui-datagrid" id="tablePCIDetection" fit="true" data-options="border:false,fitColumns:true,
                    rownumbers:true,
                    singleSelect:true,
                    striped:true,
                    onRowContextMenu: popRowMenuTablePCIDetection,onLoadSuccess:datagridLoadSuccess">
            <thead>
            <tr>
            	<th data-options="field:'neighbor_index',hidden:true"></th>
                <!-- <th data-options="field:'eNodeB_Name'" width="150">eNodeB_Name</th> -->
                <th data-options="field:'eNodeB_Name'" width="150">基站名称</th>
                   <th data-options="field:'CellID'" width="150">CellID</th>
                <th data-options="field:'PCI'" width="100">PCI</th>
                <th data-options="field:'N_PLMN'" width="100">N-PLMN</th>
                <th data-options="field:'N_CellId'" width="100">N-CellId</th>
                <th data-options="field:'N_DLEarfcn'" width="100">N-DLEarfcn</th>
                <th data-options="field:'N_PCI'" width="100">N-PCI</th>
				<th data-options="field:'N_TAC'" width="100" >N-TAC</th>
            </tr>
            </thead>
        </table>
	</div>
</div>