<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<%-- 新建重启任务 --%>

<div id="addTraceTaskStepsCpe" class="easyui-panel" data-options="border:false,fit:true">
	<%-- 第一步 --%>
	<div id="step_1" class="easyui-layout" data-options="border:false,fit:true">
		<div region="north" data-options="border:false,height:66" style="padding: 20px;">
			<%-- 任务名称 --%>
			<span style="display:inline-block;width:72px;"><%=rb.getString("RenWuMingCheng")%></span>
			<input id="taskName_addTraceTaskCpe" type="text" class="border border-box" style="width:725px;margin-left:6px;"
				maxlength=100 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
		</div>
		<div region="center" data-options="border:false" style="padding: 0 20px 20px;">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="west" data-options="width:370,border:true,collapsible:false" title="<%=rb.getString("CPELieBiao")%>" style="padding-top:20px;">
					<table id="gridCell_addTask_trace_cpe"></table>
				</div>
				<div region="center" data-options="border:false,onResize:setArrowMargin">
					<a id="arrow_right_addTraceTask" href="javascript:void(0);" class="arrow_right" onclick="addSelectedCell_addTraceTaskCpe()"></a>
					<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_addTraceTaskCpe()"></a>
				</div>
				<div class="special2" region="east" data-options="width:370,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeCPE")%>'" style="padding-top:20px;">
					<%-- 已选择CPE --%>
			    	<table class="easyui-datagrid" id="selectedCell_addTraceTaskCpe"
							style="border-width: 0px;"
							data-options="fit:true,
							singleSelect: false,
							striped: true,
							border:false,
							pagination:false, 
							idField:'value',
							rownumbers:true,
							fitColumns:false,onLoadSuccess:datagridLoadSuccess">
						<thead>
							<tr>
								<th data-options="field:'ck',checkbox:true"></th>
								<th data-options="field:'value',hidden:true"></th>
								<th data-options="field:'text'" width="450">
									<%=rb.getString("CPEBianMa")%><%=rb.getString("ZuoKuoHao")%><%=rb.getString("CpeName")%><%=rb.getString("YouKuoHao")%>
								</th>
								
							</tr>
						</thead>
			  		</table>
				</div>
			</div>
		</div>
		<div region="south" data-options="border:false,height:69" style="padding:10px 20px 20px;">
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="stepDown_addTraceTaskCpe()" style="float: right;"><span><%=rb.getString("XiaYiBu")%></span></a>
		</div>
	</div>
	<%-- 第二步 --%>
	<div id="step_2" class="easyui-panel" data-options="border:false,fit:true" >
		<div class="easyui-layout" data-options="border:false,fit:true">
			<div region="center" data-options="border:false">
				<div class="easyui-layout" data-options="border:false,fit:true">
					<div region="center" data-options="border:false,height:200" style="padding: 20px 20px 20px 20px;">
						<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("ZhuiZongTiaoJian")%>'" style="padding:20px;">
							<div class="easyui-layout" data-options="fit:true,border:true">
								<div region="center" data-options="border:false,height:60">
									<div style="padding: 10px 0px">
											<label style="display: inline-block;" class="borderBoxClass"><%=rb.getString("ZhuiZongZhouQi")%>：</label>
							                <select id="tPeriod" class="easyui-combobox border border-box" data-options="editable:false" name="EVENT_TYPE" style="height:26px;">
							                    <option value="5" >5 seconds</option>
							                    <option value="10">10 seconds</option>
							                    <option value="20" selected="true">20 seconds</option>
							                    <option value="30">30 seconds</option>
							                </select>
									</div>
									<div style="line-height:55px;" class="verM">
										<label style="display: inline-block;" class="borderBoxClass"><%=rb.getString("KaiShiShiJian")%><%=rb.getString("MaoHao")%></label>
										<input id="traceStartTime" class="easyui-datetimebox border border-box borderBoxClass" 
									 	data-options="disabled:false,editable:true" style="vertical-align: middle; height: 26px">
									 	<label style="margin-right:20px;display: inline-block; margin-left: 20px" class="borderBoxClass"><%=rb.getString("JieShuShiJian")%><%=rb.getString("MaoHao")%></label>
										<input id="traceEndTime" class="easyui-datetimebox border border-box borderBoxClass" data-options="disabled:false,editable:true" style="vertical-align: middle; height: 26px">
									</div>
									
								</div>
							</div>
						</div>
					</div>
					<div region="south" style="padding: 0 20px 20px 20px;" data-options="border:false,height:200">
						<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding:20px;">
							<div class="easyui-layout" data-options="fit:true,border:true">
								<div region="north" data-options="border:false,height:60">
									<div style="line-height:55px;" class="verM">
										<input type="radio" id="active_addTraceTask_cpe_super" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable()" />
										<label for="active_addTraceTask_cpe_super" style="display:inline-block;width:550px;" ><%=rb.getString("LiJiZhiXing")%></label>
										<input type="radio" id="suspend_addTraceTask_cpe_super" status="suspend"  name="taskStatus" onchange="setExeTimerEnable()"/>
										<label for="suspend_addTraceTask_cpe_super"><%=rb.getString("GuaQi")%></label>
									</div>
								</div>
								<div region="center" data-options="border:false" style="display:none">
									<div style="margin-top: 10px;" class="dateRe">
										<input type="radio" id="timing_addTraceTask_cpe_super" status="timing" name="taskStatus" onchange="setExeTimerEnable()"/>
										<label for="timing_addTraceTask_cpe_super" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
										<input id="exeTime_addTraceTask_cpe_super" class="easyui-datetimebox border-box border" style="height:26px;"
											data-options="disabled:true,editable:false">
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div region="south" data-options="border:false,height:69" style="padding: 10px 20px 20px;">
				<div class="windowButtonGroup">				
					<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUp_addTraceTaskCpe()"><span><%=rb.getString("ShangYiBu")%></span></a>
					<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveTraceTaskCpe()"><span><%=rb.getString("WanCheng")%></span></a>
					<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelCreateTraceTaskCpe()"><span><%=rb.getString("QuXiao")%></span></a>
				</div>
			</div>
		</div>
	</div>
</div>

<%-- 工具栏-CPE列表 --%>
<div id="toolbar_GridCell_addTask_trace_cpe" class="admin_query_head" style="background-color:white;">
	<div class="queryGroup">	
		<ul class="inputslist">
		 	<li class="select_reset">
				<select id = "traceDeviceGroupCpe" name="type" class="border border-box" style="width:100px;margin-right:1px;height: 26px;"></select>
			</li>
			<li class="serial_number input_li">
				<input name="value" style="width:175px;margin-left:10px;" placeholder="<%=rb.getString("CpeBianMaHUOMINGCHENG")%>"/>
			</li>
			<li>
				<b onclick="$('#gridCell_addTask_trace_cpe').datagrid('reload');"></b>	
			</li>
		</ul>
	</div>
</div>

<script type="text/javascript">
var lang = "${language}";
var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在CPE列表加载前

$(function() {
	closeLoading();
	var objele = $("#traceStartTime");
	disableSelectEarlyTime(objele); 
	var objele = $("#traceEndTime");
	disableSelectEarlyTime(objele); 
	
	$("#taskName_addTraceTaskCpe").val("${addTaskName}");
	$("#addTraceTaskStepsCpe #step_2").hide();
	initCellGrid_addTraceTask();
	/* $("#cellSearch_addRebootTask").bind('keyup', function(e) {
		if (e["keyCode"] == 13) {
			$("#cellTree_addRebootTask").tree("reload");
		}
	}); */
	
	$("#gridCell_addTask_trace_cpe").datagrid({
		border: false,
		fitColumns: true,
		fit: true,
        rownumbers: true,
        url: '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3',
        queryParams:{like_fields:'serial_number,host_name'},
        pageSize: 100,
        pageList: [100],
        striped: true,
        singleSelect: false,
        pagination: true,
        pagePosition: 'bottom',
        idField: 'small_cell_code',
        onBeforeLoad: beforeLoad_gridCell_addTask_trace_cpe,
        onLoadSuccess: loadSuccess_addTask_trace_cpe,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_GridCell_addTask_trace_cpe',
        onCheckAll:checkAll_trace,
        onBeforeCheck:beforeCheck_trace,
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'serial_number', sortable: true, width: 100, title: '<%=rb.getString("CPEBianMa")%>'},
			{field: 'host_name', sortable: true, width: 100, title: '<%=rb.getString("CpeName")%>'}
		]]
	});
	
	$("#gridCell_addTask_trace_cpe").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
	
	$("#traceDeviceGroupCpe").combobox({
    	url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action',
        width: 100,
        panelWidth: 150,
        panelHeight: 200,
        valueField: 'id',
        textField: 'group_name',
        onSelect: chooseRebootDeviceGroup
  });
	
});
var delCellCodeObjArr = new Array();

$("#traceDeviceGroupCpe").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']);

function checkAll_trace(rows){
	$(this).datagrid("initRow").datagrid("getPanel").find(".datagrid-htable .datagrid-header-check>input").prop("checked",true);
}
function beforeCheck_trace(index,row){
	var bool = $(this).datagrid("checkUnable",row);
	if(bool){
		return !bool;
	}
}
function chooseRebootDeviceGroup(data){
	choosedGroupId = data.id;
    $("#gridCell_addTask_trace_cpe").datagrid("reload");
}

function loadSuccess_addTask_trace_cpe() {
	$(this).datagrid("initRow");
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	
	//初始添加-行操作提示记录
    $("#selectedCell_addTraceTaskCpe").datagrid({
    	url : "",
    	data: {
    		total: 1,
    		rows: [
    			{
    				value: "", text: "<%=rb.getString("QingXuanZe")%>"
    			}
    		]
    	}
    });
    //去掉已经选中的CPE
    if (delCellCodeObjArr.length > 0) {
    	for(var cellCount = 0; cellCount < delCellCodeObjArr.length; cellCount++) {
    		var rowIndex = $("#gridCell_addTask_trace_cpe").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["small_cell_code"]);
    		if (rowIndex != -1) {
    			//$("#gridCell_addTask_trace_cpe").datagrid("deleteRow",rowIndex);
    			$("#gridCell_addTask_trace_cpe").datagrid("setRow",{small_cell_code:delCellCodeObjArr[cellCount]["small_cell_code"],usable:false});
    		}
    	}
    	//向右侧列表添加已经选中的CPE
    	for (var i = 0; i < delCellCodeObjArr.length; i++) {
    		var row = {value: delCellCodeObjArr[i]["small_cell_code"], text: delCellCodeObjArr[i]["serial_number"] + "(" + delCellCodeObjArr[i]["host_name"] + ")"};
    		$("#selectedCell_addTraceTaskCpe").datagrid("appendRow", row);
    	}
    	 
    	var rows = $("#selectedCell_addTraceTaskCpe").datagrid("getRows");
    	if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
    		$("#selectedCell_addTraceTaskCpe").datagrid("deleteRow", 0);
    	}
    }
}
<%-- 初始化CPE树 --%>
function initCellGrid_addTraceTask() {
	<%-- CPE搜索-回车事件 --%>
    $("#toolbar_GridCell_addTask_trace_cpe input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_addTask_trace_cpe").datagrid("reload");
        }
    });
    
    <%-- CPE搜索-软件版本-下拉面板 --%>
	$("#softwareVersionCombo_addTask_reboot").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	});
	
	<%-- CPE搜索-类型-change事件 --%>
	$("#toolbar_GridCell_addTask_trace_cpe select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_addTask_trace_cpe .input_li").hide();
		$("#toolbar_GridCell_addTask_trace_cpe ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_addTask_reboot").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	});
	
	<%-- CPE搜索-地域树-下拉面板 --%>
	$("#regnTreeCombo_addTask_reboot").combotree({
		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
		panelWidth: 200
	});
	
	$(".group_id input.textbox-text").css("padding", "0 10px").css("margin", "0");
}
<%-- 下一步  --%>
function stepDown_addTraceTaskCpe() {
	var rows = $("#selectedCell_addTraceTaskCpe").datagrid("getRows");
	if(rows.length > 5){
		showMsg('prompt_msg',"<%=rb.getString("ZuiDuoXuanZeSheBei5")%>");
		return;	
	}
	if (validateStep1_addTraceTaskCpe()) {
		$("#addTraceTaskStepsCpe #step_1").hide();
		// 改变窗口大小
		/* $("#winAddTraceTaskCpe").window("resize", {
			height: 545,
			width: 860
		}).window("center"); */
		setDefaultWindow({
			height: 545,
			width: 860
		});
		$("#addTraceTaskStepsCpe #step_1").hide();
		$("#addTraceTaskStepsCpe #step_2").show();
		$('#addTraceTaskStepsCpe #step_2').panel("doLayout");
	}
}
<%-- 上一步 --%>
function stepUp_addTraceTaskCpe() {
	$("#addTraceTaskStepsCpe #step_2").hide();
	// 改变窗口大小
	/* $("#winAddTraceTaskCpe").window("resize", {
		height: document.body.clientHeight * 0.9,
		width: 860
	}).window("center"); */
	setDefaultWindow({
		height: document.body.clientHeight * 0.9,
		width: 860
	});
	$("#addTraceTaskStepsCpe #step_1").show();
}

<%-- 验证第一步输入 --%>
function validateStep1_addTraceTaskCpe() {
	// 验证任务名称是否填写
	if ($("#taskName_addTraceTaskCpe").val().trim().length == 0) {
		showMsg('prompt_msg',"<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>");
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/trace/taskNameExistForCpe.action", 
			data: {"taskName": $("#taskName_addTraceTaskCpe").val().trim()},
			async: false,
			dataType: 'json',
			success: function(data) {
				if (data["success"]) {
					if (data["message"] == "true") {// 任务名称已存在
						exist = true;
					}
				}
			}
		});
		if (exist) {
			showMsg('prompt_msg',"<%=rb.getString("RenWuMingChengYiCunZai")%>");
			return false;
		}
	}

	// 是否已选择小站
	var selCells = $("#selectedCell_addTraceTaskCpe").datalist("getRows");
	if (selCells.length == 1 && selCells[0]["value"].length == 0) {
		showMsg('prompt_msg',"<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	<%-- // 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	if (document.getElementById("timing_addTraceTaskCpe").checked
			&& $("#exeTime_addTraceTaskCpe").datetimebox("getValue").length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeShiJian")%>");
		return false;
	} --%>
	return true;
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable_addTraceTaskCpe() {
	if (document.getElementById("timing_addTraceTaskCpe").checked) {
		$("#exeTime_addTraceTaskCpe").datetimebox("enable");
	} else {
		$("#exeTime_addTraceTaskCpe").datetimebox("disable");
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_addTraceTaskCpe() {
	var dl = $("#selectedCell_addTraceTaskCpe");
	var selCell = dl.datagrid("getSelections");
	var selCellLength = selCell.length;
	var needDelCellArr = new Array(); //需要删除的小站编码数组
	
	if (selCellLength > 0) {
		for (var selCellCount = 0; selCellCount < selCell.length; selCellCount++) {
			if (selCell[selCellCount]["value"]) {
    			//从CPE列表中恢复CPE
        		for(var cellCount = 0; cellCount < delCellCodeObjArr.length; cellCount++) {
        			if (selCell[selCellCount].value == delCellCodeObjArr[cellCount]["small_cell_code"]) {
            			var row = {
            				small_cell_code: delCellCodeObjArr[cellCount]["small_cell_code"], 
            				serial_number: delCellCodeObjArr[cellCount]["serial_number"],
            				host_name: delCellCodeObjArr[cellCount]["host_name"],
            				connection_status: delCellCodeObjArr[cellCount]["connection_status"]
            			}
            			if(choosedGroupId == delCellCodeObjArr[cellCount]["groupId"]){//如果是当前组被选中的站，则可以在右侧添加，否则不添加
            				//$("#gridCell_addTask_trace_cpe").datagrid("appendRow",row);
            				$("#gridCell_addTask_trace_cpe").datagrid("setRow",{small_cell_code:row["small_cell_code"],usable:true});
            			}
            			delCellCodeObjArr.splice(cellCount,1);
            			needDelCellArr.push(selCell[selCellCount].value);
            			break;
        			}
        		}
    		}
		}
		
    	if (needDelCellArr.length > 0) {
    		//从已选CPE列表中删除CPE
    		for(var cellCount = 0; cellCount < needDelCellArr.length; cellCount++) {
    			var rowIndex = $("#selectedCell_addTraceTaskCpe").datagrid("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_addTraceTaskCpe").datagrid("deleteRow",rowIndex);
    		}
    	}
    	
		var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		//将复选框取消选中
		$("#selectedCell_addTraceTaskCpe").datagrid("uncheckAll");
	}
}

<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedCell_addTraceTaskCpe() {
	var selCell = $("#gridCell_addTask_trace_cpe").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_addTraceTaskCpe");
	var rows = dl.datagrid("getRows");


	for (var selCellCount = 0; selCellCount < selCellLength; selCellCount++) {
		var i = 0;
		var cellName = selCell[selCellCount]["small_cell_code"];
		for (; i < rows.length; i++) {
			//如果当前已经选择了该小站，跳出循环
    		if (rows[i]["value"] == cellName) {
    			break;
    		}
    	}
		//i == rows.length表示当前没有选择该小站，需要加到右侧列表中
		if (i == rows.length) {
			var row = {value: selCell[selCellCount]["small_cell_code"], text: selCell[selCellCount]["serial_number"] + "(" + selCell[selCellCount]["host_name"] + ")"};
   			dl.datagrid("appendRow", row); 
   			var selectedCellObj = new Object();
   			selectedCellObj.small_cell_code = selCell[selCellCount]["small_cell_code"];
   			selectedCellObj.serial_number = selCell[selCellCount]["serial_number"];
   			selectedCellObj.host_name = selCell[selCellCount]["host_name"];
   			selectedCellObj.connection_status = selCell[selCellCount]["connection_status"];
   			selectedCellObj.groupId = choosedGroupId;
   			delCellCodeObjArr.push(selectedCellObj);
   			newSelectdArr.push(selectedCellObj)
		}
	}
	
	if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		dl.datagrid("deleteRow", 0);
	}
		
	if (newSelectdArr.length > 0) {
		//从CPE列表中删除已选择的CPE
		for(var cellCount = 0; cellCount < newSelectdArr.length; cellCount++) {
			var rowIndex = $("#gridCell_addTask_trace_cpe").datagrid("getRowIndex", newSelectdArr[cellCount]["small_cell_code"]);
			//$("#gridCell_addTask_trace_cpe").datagrid("deleteRow",rowIndex);
			$("#gridCell_addTask_trace_cpe").datagrid("setRow",{small_cell_code:newSelectdArr[cellCount]["small_cell_code"],usable:false});
		}
	}
	
	$("#gridCell_addTask_trace_cpe").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveTraceTaskCpe() {
	var tStartTime = $("#traceStartTime").datetimebox("getValue");
	var tEndTime = $("#traceEndTime").datetimebox("getValue");
	if(!tStartTime || !tEndTime){
		showMsg('prompt_msg',"<%=rb.getString("QingXuanZeShiJian")%>");
		return;
	}
    //开始时间不能晚于结束时间
    var validTimeResult = validateStartAndStopTime(tStartTime, tEndTime);
    if ("false" == validTimeResult) {
        showMsg('prompt_msg',"<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>");
        return;
    }
    if ("out" == validTimeResult) {
        showMsg('prompt_msg',"<%=rb.getString("ZhuiZongShiJianTiShi")%>");
        return;
    }
	var params = {};
	params.timeZone=timeZone;
	params["tStartTime"] = tStartTime;
	params["tEndTime"] = tEndTime;
	params["taskName"] = $("#taskName_addTraceTaskCpe").val().trim();
	
	// 取得已选择的CPE
	var cellCodes = "";
	var selCells = $("#selectedCell_addTraceTaskCpe").datalist("getRows");
	for (var i = 0; i < selCells.length; i++) {
		cellCodes += selCells[i]["value"] + ",";
	}
	params["cellCodes"] = cellCodes;
	// 取得执行方式：激活、挂起、定时执行
	var radios = document.getElementsByName("taskStatus");
	for (var i = 0; i < radios.length; i++) {
		if (radios[i].checked == true) {
			params["status"] = $(radios[i]).attr("status");
		}
	}
	// 如果是定时执行的，取得时间
	if (params["status"] == "timing") {
		params["time"] = $("#exeTime_addTraceTaskCpe").datetimebox("getValue");
	}
	var tperiod = $("#tPeriod").combobox('getValue');
	params["tPeriod"] = tperiod;
	$.post("${ctx}/task/trace/addTaskForCpe.action", params, function(data) {
		/* $("#winAddTraceTaskCpe").window("close"); */
		closeDefaultWindow();
		if (data["success"]) {
			$("#traceTaskListCpe").datagrid("load");
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json"); 
}

// 取消新建任务
function cancelCreateTraceTaskCpe() {
	/* $("#winAddTraceTaskCpe").window("close"); */
	closeDefaultWindow();
}

//加载前事件-CPE列表
function beforeLoad_gridCell_addTask_trace_cpe(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $("#toolbar_GridCell_addTask_trace_cpe .inputslist li.serial_number>input").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	
}

<%--验证开始时间和结束时间是否合法--%>
function validateStartAndStopTime(startTimeStr, endTimeStr){
    if (null != startTimeStr && null != endTimeStr) {
        var startDate = dateParser(startTimeStr);
        var endDate = dateParser(endTimeStr);
        if (startDate.getTime() < endDate.getTime()) {
        	var substract = endDate.getTime() - startDate.getTime();
       		if(substract > 60*60*1000){
	           	return "out";
            }
            return "true";
        }
    }
    if(""== startTimeStr&& "" == endTimeStr){
    	return "true";
    }
    return "false";
}
</script>