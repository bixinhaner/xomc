<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<%-- 新建配置还原任务 --%>
<div id="addConfigRecoverTaskSteps" class="easyui-panel" data-options="border:false" style="width: 100%;height: 100%;">
	<%-- 第一步 --%>
	<div id="step_1" class="easyui-layout" data-options="border:false" style="width: 100%;height: 100%;">
		<div region="center" data-options="border:false">
            <div class="easyui-layout" style="height: 100%;min-height: 750px;">
				<div region="north" data-options="border:false,height:66" style="padding:20px;">
					<%-- 任务名称 --%>
					<label style="width:72px; display: inline-block"><%=rb.getString("RenWuMingCheng")%></label>
					<input id="taskName_addConfigRecoverTask" type="text" class="border border-box" style="width:600px;margin-left:10px;"
						maxlength=100 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
				</div>
				<div region="center" data-options="border:false" style="padding: 0 20px 20px;">
					<div class="easyui-layout" data-options="border:false,fit:true">
						<div region="west" data-options="width:420,border:true,collapsible:false" title="<%=rb.getString("JiZhanLieBiao")%>" style="padding-top:20px;">
							<table id="gridCell_addTask_configRecover"></table>
						</div>
						<div region="center" data-options="border:false,onResize:setArrowMargin">
							<a id="arrow_right_addConfigRecoverTask" href="javascript:void(0);" class="arrow_right" onclick="addSelectedCell_addConfigRecoverTask()"></a>
							<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_addConfigRecoverTask()"></a>
						</div>
						<div region="east" data-options="width:420,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeJiZhan")%>'" style="padding-top:20px;">
							<%-- 已选择基站 --%>
					    	<table class="easyui-datagrid" id="selectedCell_addConfigRecoverTask"
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
											<%=rb.getString("XiaoZhanBianMa")%><%=rb.getString("ZuoKuoHao")%><%=rb.getString("HostName")%><%=rb.getString("YouKuoHao")%>
										</th>
									</tr>
								</thead>
					  		</table>
						</div>
						<div region="south" data-options="height:160,border:false,collapsible:false" style="padding-top: 20px;">
							<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding: 15px 25px;">
								<div class="verM">
									<input type="radio" id="active_addConfigRecoverTask" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable_addConfigRecoverTask()"/>
									<label for="active_addConfigRecoverTask" style="display:inline-block;width:440px;"><%=rb.getString("LiJiZhiXing")%></label>
									<input type="radio" id="suspend_addConfigRecoverTask" status="suspend" style="margin-left: 40px;" name="taskStatus" onchange="setExeTimerEnable_addConfigRecoverTask()"/>
									<label for="suspend_addConfigRecoverTask"><%=rb.getString("GuaQi")%></label>
									
								</div>
								<div class="dateRe" style="margin-top: 10px;">
									<input type="radio" id="timing_addConfigRecoverTask" status="timing" name="taskStatus" onchange="setExeTimerEnable_addConfigRecoverTask()"/>
									<label for="timing_addConfigRecoverTask" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
									<input id="exeTime_addConfigRecoverTask" class="easyui-datetimebox border-box border" style="height:26px;"
										data-options="disabled:true,editable:false">
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div region="south" data-options="border:false,height:71" style="padding: 10px 20px 20px;">
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="stepDown_addConfigRecoverTask()" style="float: right;margin-right:18px"><span><%=rb.getString("XiaYiBu")%></span></a>
		</div>
	</div>
	<%-- 第二步 --%>
	<div id="step_2" class="easyui-panel" data-options="border:false,fit:true">
		<div style="height:28px;line-height:28px;margin:20px auto;text-align:center;">
			<div class="question-mark"></div>
			<div style="display:inline-block;margin-left:10px"><%=rb.getString("QueDingHuanYuanJiZhanPeiZhi")%></div>
		</div>
		<div style="height:28px;text-align:center;">
			<div class="windowButtonGroup" style="float:none">
				<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUp_addConfigRecoverTask()"><span><%=rb.getString("ShangYiBu")%></span></a>
				<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="saveConfigRecoverTask(this)"><span><%=rb.getString("QueDing")%></span></a>
				<a class="easyui-linkbutton linkbutton linkbutton_trend" href="javascript:void(0)" onclick="cancelCreateConfigRecoverTask()"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
		</div>
	</div>
</div>

<%-- 工具栏-基站列表 --%>
<div id="toolbar_GridCell_addTask_configRecover" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery">
	 	<li class="select_reset" style="margin:0 10px;">
			<select id = "configRecoverDeviceGroup" name="type" class="border border-box" style="width:100px;margin-right:1px;height: 26px;"></select>
		</li>
		<li class="serial_number input_li">
			<%-- <input name="value" type="text" class="border border-box" style="width:154px;" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>"> --%>
			<input name="value" style="width:180px;margin-left:0px" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		</li>
		<li>
			<b class="searchResultImgChangeStyle" onclick="$('#gridCell_addTask_configRecover').datagrid('reload');"></b>	
		</li>
	</ul>
	
</div>

<script type="text/javascript">

var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在基站列表加载前

$(function() {
	closeLoading();
	var ele = $("#exeTime_addConfigRecoverTask");
	disableSelectEarlyTime(ele);
	$("#taskName_addConfigRecoverTask").val("${addTaskName}");
	$("#addConfigRecoverTaskSteps #step_2").hide();
	initCellGrid_addConfigRecoverTask();
	$("#cellSearch_addConfigRecoverTask").bind('keyup', function(e) {
		if (e["keyCode"] == 13) {
			$("#cellTree_addConfigRecoverTask").tree("reload");
		}
	});
	
	$("#gridCell_addTask_configRecover").datagrid({
		url: '${ctx}/cell/nvfile/getExistBackupCellList.action?forSelect=1',
		queryParams:{timeZone:timeZone},
		singleSelect:false,
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		pagePosition:'bottom',
		pageSize : 100,
		pageList : [100],
		idField:'small_cell_code',
		toolbar:'#toolbar_GridCell_addTask_configRecover',
		pagination : true,
		striped: true,
		onLoadError:datagridLoadError,
		onLoadSuccess:loadSuccess_addTask_configRecover,
		onBeforeLoad:beforeLoad_gridCell_addTask_configRecover,
		columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status',sortable:true,fixed:true,width: 30,formatter:connStatusFormatter},
			{field: 'serial_number',sortable:true,width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name',sortable:true,width: 100, title: '<%=rb.getString("HostName")%>'}
		]]
	});
	
	$("#gridCell_addTask_configRecover").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
	
	$("#configRecoverDeviceGroup").combobox({
    	url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action',
        width: 100,
        panelWidth: 150,
        panelHeight: 200,
        valueField: 'id',
        textField: 'group_name',
        onSelect: chooseConfigRecoverDeviceGroup
  });
	
});
//保存当前已选择的基站对象数组
var delCellCodeObjArr = new Array();

$("#configRecoverDeviceGroup").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']);

function chooseConfigRecoverDeviceGroup(data){
	choosedGroupId = data.id;
    $("#gridCell_addTask_configRecover").datagrid("reload");
}
<%-- 基站列表加载完成事件--%>
function loadSuccess_addTask_configRecover() {
	$(this).datagrid("enableContextmenuAutoSize");
	//初始添加-行操作提示记录
    $("#selectedCell_addConfigRecoverTask").datagrid({
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
    //去掉已经选中的基站
	if (delCellCodeObjArr.length > 0) {
		for(var cellCount = 0; cellCount < delCellCodeObjArr.length; cellCount++) {
			var rowIndex = $("#gridCell_addTask_configRecover").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["small_cell_code"]);
			if (rowIndex != -1) {
				$("#gridCell_addTask_configRecover").datagrid("deleteRow",rowIndex);
			}
		}
		//向右侧列表添加已经选中的基站
		for (var i = 0; i < delCellCodeObjArr.length; i++) {
			var row = {value: delCellCodeObjArr[i]["small_cell_code"], text: delCellCodeObjArr[i]["serial_number"] + "(" + delCellCodeObjArr[i]["host_name"] + ")"};
			$("#selectedCell_addConfigRecoverTask").datagrid("appendRow", row);
		}
		 
		var rows = $("#selectedCell_addConfigRecoverTask").datagrid("getRows");
		if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
			$("#selectedCell_addConfigRecoverTask").datagrid("deleteRow", 0);
		}
	}
}
<%-- 初始化基站树 --%>
function initCellGrid_addConfigRecoverTask() {
	<%-- 基站搜索-回车事件 --%>
    $("#toolbar_GridCell_addTask_configRecover input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_addTask_configRecover").datagrid("reload");
        }
    });
    
    <%-- 基站搜索-软件版本-下拉面板 --%>
	$("#softwareVersionCombo_addTask_configRecover").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	});
	
	<%-- 基站搜索-类型-change事件 --%>
	$("#toolbar_GridCell_addTask_configRecover select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_addTask_configRecover .input_li").hide();
		$("#toolbar_GridCell_addTask_configRecover ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_addTask_configRecover").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	});
	
	<%-- 基站搜索-地域树-下拉面板 --%>
	$("#regnTreeCombo_addTask_configRecover").combotree({
		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
		panelWidth: 200
	});
	
	$(".group_id input.textbox-text").css("padding", "0").css("margin", "0");
}
<%-- 下一步  --%>
function stepDown_addConfigRecoverTask() {
	if (validateStep1_addConfigRecoverTask()) {
		// 改变窗口大小
		/* $("#winAddConfigRecoverTask").window("resize", {
			height: 200,
			width: 400
		}).window("center"); */
		setDefaultWindow({
			height: 200,
			width: 400
		});
		$("#addConfigRecoverTaskSteps #step_1").hide();
		$("#addConfigRecoverTaskSteps #step_2").show();
		$("#addConfigRecoverTaskSteps").css("width", "360").css("height", "130");
        /* $.parser.parse($("#winAddConfigRecoverTask")); */
		//$.parser.parse($(winDefaultSelector));
	}
}
<%-- 上一步 --%>
function stepUp_addConfigRecoverTask() {
	// 改变窗口大小
	/* $("#winAddConfigRecoverTask").window("resize", {
        height: document.body.clientHeight * 0.9,
        width: 960
    }).window("center"); */
    setDefaultWindow({
    	height: document.body.clientHeight * 0.9,
        width: 960
	});
	$("#addConfigRecoverTaskSteps #step_1").show();
	$("#addConfigRecoverTaskSteps #step_2").hide();
    $("#addConfigRecoverTaskSteps").css("width", "100%").css("height", "100%");
    /* $.parser.parse($("#winAddConfigRecoverTask")); */
    $.parser.parse($(winDefaultSelector));
}

<%-- 验证第一步输入 --%>
function validateStep1_addConfigRecoverTask() {
	// 验证任务名称是否填写
	if ($("#taskName_addConfigRecoverTask").val().trim().length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>");
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/configRecover/taskNameExist.action", 
			data: {"taskName": $("#taskName_addConfigRecoverTask").val().trim()},
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
			$.messager.alert(TiShi, "<%=rb.getString("RenWuMingChengYiCunZai")%>");
			return false;
		}
	}

	// 是否已选择小站
	var selCells = $("#selectedCell_addConfigRecoverTask").datagrid("getRows");
	if (selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	// 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	if (document.getElementById("timing_addConfigRecoverTask").checked
			&& $("#exeTime_addConfigRecoverTask").datetimebox("getValue").length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeShiJian")%>");
		return false;
	}
	return true;
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable_addConfigRecoverTask() {
	if (document.getElementById("timing_addConfigRecoverTask").checked) {
		$("#exeTime_addConfigRecoverTask").datetimebox("enable");
	} else {
		$("#exeTime_addConfigRecoverTask").datetimebox("disable");
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_addConfigRecoverTask() {
	var dl = $("#selectedCell_addConfigRecoverTask");
	var selCell = dl.datagrid("getSelections");
	var selCellLength = selCell.length;
	var needDelCellArr = new Array(); //需要删除的小站编码数组
	
	if (selCellLength > 0) {
		for (var selCellCount = 0; selCellCount < selCell.length; selCellCount++) {
			if (selCell[selCellCount]["value"]) {
    			//从基站列表中恢复基站
        		for(var cellCount = 0; cellCount < delCellCodeObjArr.length; cellCount++) {
        			if (selCell[selCellCount].value == delCellCodeObjArr[cellCount]["small_cell_code"]) {
            			var row = {
            				small_cell_code: delCellCodeObjArr[cellCount]["small_cell_code"], 
            				serial_number: delCellCodeObjArr[cellCount]["serial_number"],
            				host_name: delCellCodeObjArr[cellCount]["host_name"],
            				connection_status: delCellCodeObjArr[cellCount]["connection_status"]
            			}
            			if(choosedGroupId == delCellCodeObjArr[cellCount]["groupId"]){//如果是当前组被选中的站，则可以在右侧添加，否则不添加
            				$("#gridCell_addTask_configRecover").datagrid("appendRow",row);
            			}
            			delCellCodeObjArr.splice(cellCount,1);
            			needDelCellArr.push(selCell[selCellCount].value);
            			break;
        			}
        		}
    		}
		}
		
    	if (needDelCellArr.length > 0) {
    		//从已选基站列表中删除基站
    		for(var cellCount = 0; cellCount < needDelCellArr.length; cellCount++) {
    			var rowIndex = $("#selectedCell_addConfigRecoverTask").datagrid("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_addConfigRecoverTask").datagrid("deleteRow",rowIndex);
    		}
    	}
    	
    	var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		
		//将复选框取消选中
		$("#selectedCell_addConfigRecoverTask").datagrid("uncheckAll");
	}
}
<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedCell_addConfigRecoverTask() {
	var selCell = $("#gridCell_addTask_configRecover").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_addConfigRecoverTask");
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
		//从基站列表中删除已选择的基站
		for(var cellCount = 0; cellCount < newSelectdArr.length; cellCount++) {
			var rowIndex = $("#gridCell_addTask_configRecover").datagrid("getRowIndex", newSelectdArr[cellCount]["small_cell_code"]);
			$("#gridCell_addTask_configRecover").datagrid("deleteRow",rowIndex);
		}
	}
	
	$("#gridCell_addTask_configRecover").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveConfigRecoverTask(ele) {
	var params = {};
	params.timeZone=timeZone;
	params["taskName"] = $("#taskName_addConfigRecoverTask").val().trim();
	
	// 取得已选择的基站
	var cellCodes = "";
	var selCells = $("#selectedCell_addConfigRecoverTask").datagrid("getRows");
	if (selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}

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
		params["time"] = $("#exeTime_addConfigRecoverTask").datetimebox("getValue");
	}
	//已经点击了确认按钮，禁用“确认按钮”，以防止服务器响应慢而导致重复点击按钮
	$(ele).linkbutton('disable');
	
	$.post("${ctx}/task/configRecover/addTask.action", params, function(data) {
		if (data["success"]) {
			/* $("#winAddConfigRecoverTask").window("close"); */
			closeDefaultWindow();
			$("#configRecoverTaskList").datagrid("load");
		} else {
			$(ele).linkbutton('enable');
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

// 取消新建任务
function cancelCreateConfigRecoverTask() {
	/* $("#winAddConfigRecoverTask").window("close"); */
	closeDefaultWindow();
}

//加载前事件-基站列表
function beforeLoad_gridCell_addTask_configRecover(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $(".inputslist li.serial_number>input").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	
	/* var type = $("#toolbar_GridCell_addTask_configRecover select[name='type']").val();
	var val = $("#toolbar_GridCell_addTask_configRecover ." + type + " input[name='value']").val();
	param[type] = val; */
}
</script>