<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<script type="text/javascript">
var ctx = "${ctx}";
var QueRen = "<%=rb.getString("QueRen")%>"

/*<%-- 加载完成事件 --%>*/
$(function() {
	$("#divWestList").css({
		width: document.body.clientWidth * 0.45
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
		width:450,height:210
	});
}

<%--打开“解决PCI冲突”对话框--%>
function resolveConfirm() {
    var selRecord = $("#tablePCICollisionList").datagrid("getSelected");
	
    /* var params = {
        cellCode : selRecord["SMALL_CELL_CODE"],
        cellId : selRecord["CELL_ID"],
        currentPCI : selRecord["PCI"],
        type : selRecord["PCI_ERROR_TYPE"]
    }; */
    var params = null;
    
    if (selRecord["PCI_ERROR_TYPE"] == "SelfConfig Failure") {
    	params = {
    		cellCode : selRecord["SMALL_CELL_CODE"],
    	    cellId : selRecord["CELL_ID"],
    	    pciList : selRecord["PCI_LIST"],
    	    type : selRecord["PCI_ERROR_TYPE"]
    	};
    } else {
    	params = {
    		cellCode : selRecord["SMALL_CELL_CODE"],
    	    cellId : selRecord["CELL_ID"],
    	    currentPCI : selRecord["PCI"],
    	    type : selRecord["PCI_ERROR_TYPE"]
    	};
    }
    
    var msg = "Are you sure you want to do the optimization of PCI?";
    
    $.messager.confirm(QueRen, msg, function(r) {
    	if (r) {
    		$.post("${ctx}/cell/SON/resolvePCICollision_v2.action", params, function(data) {
    	    	$.messager.alert(TiShi, data["message"]);
    	    }, "json");
    	}
    }).addClass("normalConfirm");
}

function modifyNewPCIValue() {
	var selRecord = $("#tablePCIDetailedInfos").datagrid("getSelected");
    //var params = $("#modifyPCIForm").serializeJson();
    var params = {
        cellId : selRecord["co_CellID"],
        newPCI : document.getElementById("txt_N_NewPCI").value
    };
    $.post("${ctx}/cell/SON/modifyPCIValue_v2.action", params, function(data) {
    	/* $("#winModifyPCIValue").window("close"); */
    	closeDefaultWindow();
    	$.messager.alert(TiShi, data["message"]);
    }, "json");
} 

function showPCIErrorDetail(index, row) {
	$("#tablePCIDetailedInfos").datagrid().datagrid("loading");
	<%--请求PCI冲突详细数据--%>
	var params = {
	    cellCode : row["SMALL_CELL_CODE"],
	    serialNumber : row["SERIAL_NUMBER"],
	    hostName : row["HOST_NAME"],
	    cellId : row["CELL_ID"],
	    PCI : row["PCI"],
	    dlErafcn : row["DLEARFCN"],
	    type : row["PCI_ERROR_TYPE"]
	};
	<%--向服务器发起请求，将PCI冲突的详细信息返回到页面--%>
	$.post("${ctx}/cell/SON/showPCIErrorDetailedInfo_v2.action", params, function(data) {
		$("#tablePCIDetailedInfos").datagrid("loadData", data["gridData"]);
		$("#tablePCIDetailedInfos").datagrid("loaded");
		cellNPCI = data["cellNPCI"];
	}, "json");
}

function resolvePCIError() {
	var selCell = $("#tablePCICollisionList").datagrid("getSelected");
	if (!selCell) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeXiaoQu")%>");
		return;
	}
	var params = {
	    cellCode : selCell["SMALL_CELL_CODE"],
	    cellId : selCell["CELL_ID"],
	    currentPCI : selCell["PCI"],
	    type : selCell["PCI_ERROR_TYPE"]
	};
	    
	var msg = "Are you sure you want to do the optimization of PCI?";

    $.messager.confirm(QueRen, msg, function(r) {
        if (r) {
	        $.post("${ctx}/cell/SON/resolvePCICollision_v2.action", params, function(data) {
    	        $.messager.alert(TiShi, data["message"]);
            }, "json");
        }
    }).addClass("normalConfirm");
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
	var url = "${ctx}/cell/SON/toPCIPages.action";
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
    data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:450,height:200">
</div> --%>

<%-- <div id="winModifyPCIValue" class="easyui-window" title="<%=rb.getString("JieJueFangAn")%>"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:350,height:200">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false,fit:true" style="padding: 20px 10px;">
			<form id="modifyPCIForm">
			    <span><%=rb.getString("ShouDongXiuGaiPCI")%></span>
			    <div style="border-style: none;height: 40px;margin-top:10px">
				    <lable for="txt_N_NewPCI">New PCI:</lable>
				    <input id="txt_N_NewPCI" type="text" name="suggested_newPCI" class="border border-box"/>
			    </div>
			</form>
		</div>
		<div region="south" data-options="border:true,height: 47" style="padding:10px 10px;border-width: 1px 0 0 0">
			<a class="easyui-linkbutton" onclick="closeWinPCIModification()" style="float:right margin-right: 15px;"><%=rb.getString("QuXiao")%></a>
			<a class="easyui-linkbutton" onclick="modifyNewPCIValue()" style="margin-right: 20px;float:right;"><%=rb.getString("QueDing")%></a>
		</div>
	</div>
</div> --%>
<%-- PCI列表 --%>
<div class="easyui-layout" data-options="fit:true,border:false" >
    <div region="north" data-options="border:false,height: 110,collapsible:false" style="padding:0 15px;">
	    <div data-options="region:'north',border:false,height: 40,collapsible:false" >
	        <div class="omcPageTitleDiv">
				<ul class="omcPageTitleContainer">
					<li class="default"><%=rb.getString("PCIYouHua")%></li>
				</ul>
			</div>
		</div> 
		<div class="linkbuttonGroup" style="margin:15px 0 0 29px;">
			 <a class="linkbutton" onclick="changePCIRange()"><span><%=rb.getString("PCIFanWei")%></span></a>
       		 <a class="linkbutton" onclick="resolvePCIError()"><span><%=rb.getString("JieJue")%></span></a>
		</div>
       
    </div>
	<div region="center" data-options="border:false" style="">
		<div class="easyui-layout" id="divCenterList" data-options="fit:true,border:false">
		    <div region="west" id="divWestList" style="padding-right: 15px;background-color: #F3F3F4;" data-options="border:false,split:false">
		    	<div class="easyui-panel" data-options="fit:true,border:false" style="padding: 0 15px;">
			        <table id="tablePCICollisionList" class="easyui-datagrid" data-options="
	                            border:false,
	                            url: '${ctx}/cell/SON/queryPCICollisionInfoList.action',
	                            fit:true,
	                            striped: true,
	                            pagination: true,
	                            pagePosition: 'bottom',
	                            rownumbers: true,
	                            fitColumns: true,
	                            singleSelect: true,
	                            onClickRow:showPCIErrorDetail,
	                            onRowContextMenu:popRowMenuTablePCIOptimization,
	                            onLoadSuccess:datagridLoadSuccess">              
	                    <thead>
	                    <tr>
	          	            <th data-options="field:'SMALL_CELL_CODE',hidden:true"></th>
	          	            <th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("Serial_Number") %></th>
	                        <th data-options="field:'HOST_NAME'" width="80"><%=rb.getString("eNodeB_Name") %></th>
	                        <th data-options="field:'CELL_ID'" width="80"><%=rb.getString("CellID") %></th>
	                        <th data-options="field:'PCI'" width="50">PCI</th>
	                        <th data-options="field:'PCI_LIST'" width="80">PCI_List</th>
	                        <th data-options="field:'DLEARFCN'" width="80"><%=rb.getString("DLEarfcn") %></th>
	                        <th data-options="field:'PCI_ERROR_TYPE'" width="90"><%=rb.getString("Type") %></th>
	                    </tr>
	                    </thead>
	                </table>
                </div>
		    </div>
		    <div region="center" data-options="border:false">
		    	<div class="easyui-panel" data-options="fit:true,border:false" style="padding: 0 15px;">
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
	                        <th data-options="field:'co_serial_number'" width="100"><%=rb.getString("N-Serial_Number") %></th>
	                        <th data-options="field:'co_eNodeB_Name'" width="100"><%=rb.getString("N-eNodeB_Name") %></th>
	                        <th data-options="field:'co_CellID'" width="60"><%=rb.getString("N-CellID") %></th>
	                        <th data-options="field:'co_PCI'" width="70"><%=rb.getString("N-PCI") %></th>
	                        <th data-options="field:'suggested_PCI'" width="80"><%=rb.getString("Suggested-PCI") %></th>
					        <th data-options="field:'co_DLEarfcn'" width="80"><%=rb.getString("N-DLEarfcn") %></th>
	                    </tr>
	                    </thead>
	                </table>
                </div>
		    </div>
		</div>
	</div>
	<div id="layout_PCIOptimization_progress" region="south" 
		data-options="border:false,href: '${ctx}/cell/SON/toPCIOptimizationProgress.action'" style="padding-top: 15px;background-color: #F3F3F4;">
	</div>
</div>