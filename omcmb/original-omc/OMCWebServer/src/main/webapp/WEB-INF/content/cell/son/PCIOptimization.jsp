<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<script type="text/javascript">
var ctx = "${ctx}";
var TiShi = "<%=rb.getString("TiShi")%>";
var QueRen = "<%=rb.getString("QueRen")%>"
var cellNPCI;//存储基站所有邻区的PCI值，数据示例：{'cellid_1': [PCI_1,PCI_2,PCI_3,PCI_4]}

/*<%-- 加载完成事件 --%>*/
$(function() {
	<%-- $("#tablePCICollisionList").datagrid().datagrid("loading");
	/*请求PCI冲突数据*/
	$.post("${ctx}/cell/SON/queryDetectionData.action", {}, function(data) {
		$("#tablePCICollisionList").datagrid("loadData", data["gridData"]);
		$("#tablePCICollisionList").datagrid("loaded");
		cellNPCI = data["cellNPCI"];
	}, "json"); --%>
	$("#divWestList").css({
		width: document.body.clientWidth * 0.55
	});
	
 	$("#layout_PCIOptimization_progress").css({
		height: document.body.clientHeight * 0.4
	});
});

<%--右键点击表格中的一行，弹出菜单--%>
function popRowMenuTablePCIOptimization(e, rowIndex, rowData) {
	e.preventDefault();
	if(rowIndex < 0) {
		return;
	}
	$("#tablePCICollisionList").datagrid("clearSelections");
	$("#tablePCICollisionList").datagrid("selectRow", rowIndex);
	$("#rowMenu_TablePCIInfos").menu("show", {
		left:e.clientX,
		top:e.clientY
	});
}

<%--弹出PCI范围设置窗口--%>
function changePCIRange(){
	/* $("#winPCIRangeSetting").window("open");
	$("#winPCIRangeSetting").window("refresh", "${ctx}/cell/SON/goPCIRangeSetting.action") */
	var url = "${ctx}/cell/SON/goPCIRangeSetting.action";
	openDefaultWindow(url,{
		title: '<%=rb.getString("PCIFanWei")%>',
		width:350,height:200
	});
}

<%--打开“解决PCI冲突”对话框--%>
function resolveConfirm(){
	//选择一条冲突记录
	var selRecord = $("#tablePCICollisionList").datagrid("getSelected");
	
    var params = {
        cellCode : selRecord["SMALL_CELL_CODE"],
        type : selRecord["PCI_ERROR_TYPE"]
    };
    
    var msg = "eNodeB:" + selRecord["HOST_NAME"] + "," + "CellId:" + selRecord["CELL_ID"] + "," + "DLEARFCN:" + selRecord["DLEARFCN"]
            + "\n" + "Are you sure you want to optimize the Cell you've been chosen?";
    
    $.messager.confirm(QueRen, msg, function(r) {
    	if (r) {
    		$.post("${ctx}/cell/SON/resolvePCICollision.action", params, function(data) {
    	    	$.messager.alert(TiShi, data["message"]);
    	    }, "json");
    	}
    });
	
	//开始寻找一个新的PCI分配给邻区，原则新的PCI不能与当前小区PCI相同，也不能与当前小区的其他邻区PCI相同；并且新的PCI与其他邻区PCI模3后不能相等。
	//新的PCI选择的范围首先按照客户选择的范围选取，如果没哟合适的PCI，则自动在0-503范围选取。
	<%-- for (var i = parseInt(PCI_Start) ; i<= parseInt(PCI_Stop); i++) {
		var exist = false;
		if (PCI == i) {
			exist = true;
		} else {
			for (var j = 0; j < neighborPCIArray.length; j++) {
				if (neighborPCIArray[j] == i){
					exist = true;
				} else if (neighborPCIArray[j] % 3 == i % 3) {//判断当前小区所有邻区PCI与新的PCI是否存在摸3冲突
					exist = true;
				}
			}
		}
		if (!exist) {
			N_NewPCI = i;
			break;
		}
	}
	//若根据用户选择的PCI取值范围，无法找到合适的PCI，那么扩大范围搜索
	if (N_NewPCI) {
		$("#txt_cellId").val(cellId);
		$("#txt_N_cellId").val(N_CellId);
		$("#txt_NeighborIndex").val(neighborIndex);
		$("#txt_N_CurrentPCI").val(N_PCI);
		$("#txt_N_NewPCI").val(N_NewPCI);

		$("#winPCIConfirmResolve").window("open");
	} else {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCITiShi")%>");
		setTimeout(function(){
			for (var i = 0 ; i<= 503; i++) {
				var exist = false;
				if (PCI == i) {
					exist = true;
				} else {
					for (var j = PCI_Start; j < neighborPCIArray.length; j++) {
						if (neighborPCIArray[j] == i){
							exist = true;
						} else if (neighborPCIArray[j] % 3 == i % 3) {//判断当前小区所有邻区PCI与新的PCI是否存在摸3冲突
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
			$("#txt_N_CurrentPCI").val(N_PCI);
			$("#txt_N_NewPCI").val(N_NewPCI);

			$("#winPCIConfirmResolve").window("open");
		}, "3000");
	} --%>
}

function modifyNewPCIValue() {
	var selRecord = $("#tablePCIDetailedInfos").datagrid("getSelected");
    //var params = $("#modifyPCIForm").serializeJson();
    var params = {
        n_cellId : selRecord["co_CellID"],
        n_newPCI : document.getElementById("txt_N_NewPCI").value
    };
    $.post("${ctx}/cell/SON/modifyPCIValue.action", params, function(data) {
    	/* $("#winModifyPCIValue").window("close"); */
    	closeDefaultWindow();
    	$.messager.alert(TiShi, data["message"]);
    }, "json");
} 

function showPCIErrorDetail(index, row) {
	$("#tablePCIDetailedInfos").datagrid().datagrid("loading");
	<%--请求PCI冲突详细数据--%>
	var param = {
	    cellCode : row["SMALL_CELL_CODE"],
	    hostName : row["HOST_NAME"],
	    cellId : row["CELL_ID"],
	    PCI : row["PCI"],
	    dlErafcn : row["DLEARFCN"],
	    type : row["PCI_ERROR_TYPE"]
	};
	<%--向服务器发起请求，将PCI冲突的详细信息返回到页面--%>
	$.post("${ctx}/cell/SON/showPCIErrorDetailedInfo.action", param, function(data) {
		$("#tablePCIDetailedInfos").datagrid("loadData", data["gridData"]);
		$("#tablePCIDetailedInfos").datagrid("loaded");
		cellNPCI = data["cellNPCI"];
	}, "json");
}

function resolvePCIError() {
	var selCell = $("#tablePCICollisionList").datagrid("getSelected");
	if (!selCell) {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("QingXuanZeXiaoQu")%>");
	}
	var params = {
	    cellCode : selRecord["SMALL_CELL_CODE"],
	    type : selRecord["PCI_ERROR_TYPE"]
	};
	    
	$.post("${ctx}/cell/SON/resolvePCICollision.action", params, function(data) {
	    $.messager.alert(TiShi, data["message"]);
	}, "json");
}

<%--右键手动修改PCI值--%>
function winManuallyModifyPCIValue(e, rowIndex, rowData) {
	e.preventDefault();
	if(rowIndex < 0) {
		return;
	}
	$("#tablePCIDetailedInfos").datagrid("clearSelections");
	$("#tablePCIDetailedInfos").datagrid("selectRow", rowIndex);
	/* $("#winModifyPCIValue").window({
		left:e.clientX,
		top:e.clientY
	}).window("open"); */
	$("#rowMenu_TableChangePCIValue").menu("show", {
		left:e.clientX,
		top:e.clientY
	});
}

function closeWinPCIModification() {
	/* $("#winModifyPCIValue").window("close"); */
	closeDefaultWindow();
}

function tableOnSelect(row, index) {
	var rows = $("#tablePCICollisionList").datagrid("selectRow");
	if (rows) {
		return "background:#CFCFCF";
	}
}

function modifyPCIValue() {
	/* $("#winModifyPCIValue").window("open"); */
	var url = "${ctx}/cell/SON/toPCIPages.action?code=origin";
	openDefaultWindow(url,{
		title: '<%=rb.getString("JieJueFangAn")%>',
		width:350,height:200
	});
}
</script>

<%-- 右键菜单-PCI列表 --%>
<div id="rowMenu_TablePCIInfos" class="easyui-menu" style="width: 150px;">
	<div onclick="resolveConfirm()"><%=rb.getString("JieJue")%></div>
</div>

<%--右键菜单-修改PCI值--%>
<div id="rowMenu_TableChangePCIValue" class="easyui-menu" style="width: 150px;">
	<div onclick="modifyPCIValue()"><%=rb.getString("XiuGaiPCI")%></div>
</div>

<%--配置PCI取值范围窗口 --%>
<%-- <div id="winPCIRangeSetting" class="easyui-window" title="<%=rb.getString("PCIFanWei")%>"
    data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:350,height:200">
</div> --%>

<%-- <div id="winModifyPCIValue" class="easyui-window" title="<%=rb.getString("JieJueFangAn")%>"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:350,height:200">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false,fit:true" style="padding: 20px 10px;">
			<form id="modifyPCIForm">
			    <span><%=rb.getString("ShouDongXiuGaiPCI")%></span>
			    <div style="border-style: none;height: 40px;margin-top:10px">
				    <lable for="txt_N_NewPCI">Neighbor New PCI:</lable>
				    <input id="txt_N_NewPCI" type="text" name="suggested_newPCI" class="border border-box"/>
			    </div>
			</form>
		</div>
		<div region="south" data-options="border:true,height: 37" style="padding:5px 10px;border-width: 1px 0 0 0">
			<a class="easyui-linkbutton" onclick="closeWinPCIModification()" style="float:right;"><%=rb.getString("QuXiao")%></a>
			<a class="easyui-linkbutton" onclick="modifyNewPCIValue()" style="margin-right: 10px;float:right;"><%=rb.getString("QueDing")%></a>
		</div>
	</div>
</div> --%>
<%-- PCI列表 --%>
<div class="easyui-layout" data-options="fit:true,border:false">
    <div region="north" data-options="border:false,height: 38,collapsible:false" style="border-width: 0 0 1px 0;padding:5px 10px;">
        <a class="easyui-linkbutton" onclick="changePCIRange()" style="float: left;margin-left: 10px;display: inline-block;"><%=rb.getString("PCIFanWei")%></a>
        <a class="easyui-linkbutton" onclick="resolvePCIError()" style="float: left;margin-left: 10px;display: inline-block;"><%=rb.getString("JieJue")%></a>
    </div>
	<div region="center" data-options="border:false" style="border-width: 0 0 1px 1px;">
		<div class="easyui-layout" id="divCenterList" data-options="fit:true,border:true">
		    <div region="west" id="divWestList" style="border-width:0 1px 0px 0px" data-options="border:false,split:true,maxWidth:1100">
		        <table id="tablePCICollisionList" class="easyui-datagrid" data-options="
                            border:false,
                            url: '${ctx}/cell/SON/queryPCICollisionInfoList.action',
                            fit:true,
                            striped: true,
                            pagination: true,
                            pagePosition: 'top',
                            rownumbers: true,
                            fitColumns: true,
                            singleSelect: true,
                            onClickRow:showPCIErrorDetail,
                            onRowContextMenu:popRowMenuTablePCIOptimization,
                            onLoadSuccess:datagridLoadSuccess">              
                    <thead>
                    <tr>
          	            <th data-options="field:'SMALL_CELL_CODE',hidden:true"></th>
                        <th data-options="field:'HOST_NAME'" width="150">eNodeB_Name</th>
                        <th data-options="field:'CELL_ID'" width="100">CellID</th>
                        <th data-options="field:'PCI'" width="100">PCI</th>
                        <th data-options="field:'DLEARFCN'" width="100">DLEarfcn</th>
                        <th data-options="field:'NETWORK_MODEL'" width="100">Network_Model</th>
                        <th data-options="field:'PCI_ERROR_TYPE'" width="100">Type</th>
                    </tr>
                    </thead>
                </table>
		    </div>
		    <div region="center" data-options="border:true" style="border-width: 0 0 0 1px;">
		        <table class="easyui-datagrid" id="tablePCIDetailedInfos" fit="true" data-options="
		                      border:false,
		                      fitColumns:true,
                              rownumbers:true,
                              singleSelect:true,
                              striped:true,
                              onRowContextMenu:winManuallyModifyPCIValue,
                              onLoadSuccess:datagridLoadSuccess">
                    <thead>
                    <tr>
                        <th data-options="field:'co_eNodeB_Name'" width="150">N-eNodeB_Name</th>
                        <th data-options="field:'co_CellID'" width="100">N-CellId</th>
                        <th data-options="field:'co_PCI'" width="100">N-PCI</th>
                        <th data-options="field:'suggested_PCI'" width="100">Suggested-PCI</th>
				        <th data-options="field:'co_DLEarfcn'" width="100">N-DLEarfcn</th>
                    </tr>
                    </thead>
                </table>
		    </div>
		</div>
	</div>
	<div id="layout_PCIOptimization_progress" region="south" 
		data-options="border:true,href: '${ctx}/cell/SON/toPCIOptimizationProgress.action'" style="border-width: 1px 0 0 1px;">
	</div>
</div>