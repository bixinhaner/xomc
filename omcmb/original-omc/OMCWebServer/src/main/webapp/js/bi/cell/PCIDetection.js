/*<%-- 存储各小站的所有邻区PCI值，数据示例：{'cellid_1': [PCI_1,PCI_2,PCI_3,PCI_4]} --%>*/
var cellNPCI;
/*<%-- 加载完成事件 --%>*/
$(function() {
	$("#tablePCIDetection").datagrid().datagrid("loading");
	/*<%-- 请求探测数据 --%>*/
	$.post(ctx + "/SON/getDetectionData.action", {}, function(data) {
		$("#tablePCIDetection").datagrid("loadData", data["gridData"]);
		$("#tablePCIDetection").datagrid("loaded");
		cellNPCI = data["cellNPCI"];
	}, "json");
});
/*<%-- 右键点击表格中的行，弹出菜单 --%>*/
function popRowMenuTablePCIDetection(e, rowIndex, rowData) {
	e.preventDefault();
	$("#tablePCIDetection").datagrid("clearSelections");
	$("#tablePCIDetection").datagrid("selectRow", rowIndex);
	$("#rowMenu_TablePCIDetection").menu("show", {
		left: e.clientX,
		top: e.clientY
	});
}
/*<%-- 打开“解决冲突”确认框 --%>*/
function resolveConfirm() {
	var selRecord = $("#tablePCIDetection").datagrid("getSelected");
	/*<%-- 该小站cellid --%>*/
	var cellId = selRecord["CellID"];
	/*<%-- 该小站PCI --%>*/
	var PCI = selRecord["PCI"];
	/*<%-- 邻区cellid --%>*/
	var N_CellId = selRecord["N_CellId"];
	/*<%-- 邻区索引 --%>*/
	var neighborIndex = selRecord["neighbor_index"];
	/*<%-- 邻区当前PCI --%>*/
	var N_CurrPCI = selRecord["N_PCI"];
	/*<%-- 新分配的邻区PCI --%>*/
	var N_NewPCI;
	var neighborPCIArr = cellNPCI[cellId];
	for (var i = 0; i < 504; i++) {
		var exist = false;
		if (PCI == i) {
			exist = true;
		} else {
			for (var j = 0; j < neighborPCIArr.length; j++) {
				if (neighborPCIArr[j] == i) {
					exist = true;
				}
			}
		}
		if (!exist) {
			N_NewPCI = i;
			break;
		}
	}
	$("#txt_cellId").val(cellId);
	$("#txt_N_cellId").val(N_CellId);
	$("#txt_NeighborIndex").val(neighborIndex);
	$("#txt_N_CurrentPCI").val(N_CurrPCI);
	$("#txt_N_NewPCI").val(N_NewPCI);

	$("#winConfirmResolve").window("open");
}
/*<%-- 发回服务器，重设PCI值，解决冲突 --%>*/
function resolveConflict() {
	var param = $('#resolveForm').serializeJson();
	$.post(ctx + "/SON/resolveConflict.action", param, function(data) {
		$.messager.alert(TiShi, data["message"]);
	}, "json");
}