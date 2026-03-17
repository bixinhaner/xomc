<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<%-- 新建配置备份任务 --%>
<div id="addConfigBackupTaskSteps" class="easyui-panel" data-options="border:false" style="width: 100%;height: 100%;">
	<%-- 第一步 --%>
	<div id="step_1" class="easyui-layout" data-options="border:false" style="width: 100%;height: 100%;">
	    <div region="center" data-options="border:false">
            <div class="easyui-layout" style="height: 100%;min-height: 750px;">
				<div region="north" data-options="border:false,height:66" style="padding:20px;">
					<%-- 任务名称 --%>
					<label style=" display: inline-block;width:72px;"><%=rb.getString("RenWuMingCheng")%></label>
					<input id="taskName_addConfigBackupTask" type="text" class="border border-box" style="width:600px;margin-left:10px;"
						maxlength=100 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
				</div>
				<div region="center" data-options="border:false" style="padding: 0px 20px 20px;">
					<div class="easyui-layout" data-options="border:false,fit:true">
						<div region="west" data-options="width:420,border:true,collapsible:false" title="<%=rb.getString("JiZhanLieBiao")%>" style="padding-top:20px;">
							<table id="gridCell_addTask_configBackup"></table>
						</div>
						<div region="center" data-options="border:false,onResize:setArrowMargin">
							<a id="arrow_right_addConfigBackupTask" href="javascript:void(0);" class="arrow_right" onclick="addSelectedCell_addConfigBackupTask()"></a>
							<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_addConfigBackupTask()"></a>
						</div>
						<div region="east" data-options="width:420,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeJiZhan")%>'" style="padding-top:20px;">
							<%-- 已选择基站 --%>
					    	<table class="easyui-datagrid" id="selectedCell_addConfigBackupTask"
									style="border-width: 0px;"
									data-options="fit:true,
									singleSelect: false,
									striped: true,
									border:false,
									pagination:false, 
									idField:'value',
									rownumbers:true,
									fitColumns:false,
									onLoadSuccess:datagridLoadSuccess">
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
							<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding: 20px;">
								<div class="verM">
									<input type="radio" id="active_addConfigBackupTask" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable_addConfigBackupTask()"/>
									<label for="active_addConfigBackupTask" style="display:inline-block;width:440px;"><%=rb.getString("LiJiZhiXing")%></label>
									<input type="radio" id="suspend_addConfigBackupTask" status="suspend" style="margin-left: 40px;" name="taskStatus" onchange="setExeTimerEnable_addConfigBackupTask()"/>
									<label for="suspend_addConfigBackupTask"><%=rb.getString("GuaQi")%></label>
									
								</div>
								<div class="dateRe" style="margin-top: 10px;">
									<input type="radio" id="timing_addConfigBackupTask" status="timing" name="taskStatus" onchange="setExeTimerEnable_addConfigBackupTask()"/>
									<label for="timing_addConfigBackupTask" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
									<input id="exeTime_addConfigBackupTask" class="easyui-datetimebox border-box border" style="height:26px;"
										data-options="disabled:true,editable:false">
								</div>
							</div>
						</div>
					</div>
				</div>
		     </div>
		</div>
		<div region="south" data-options="border:false,height:71" style="padding: 10px 20px 20px;">
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="stepDown_addConfigBackupTask()" style="float: right;margin-right:18px"><span><%=rb.getString("XiaYiBu")%></span></a>
		</div>
	</div>
	<%-- 第二步 --%>
	<div id="step_2" class="easyui-panel" data-options="border:false,fit:true">
		<div style="height:28px;line-height:28px;margin:20px auto;text-align:center;">
			<div class="question-mark"></div>
			<div style="display:inline-block;margin-left:10px"><%=rb.getString("QueDingBeiFenJiZhanPeiZhi")%></div>
		</div>
		<div style="height:28px;text-align:center;">
			<div class="windowButtonGroup" style="float:none">
				<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUp_addConfigBackupTask()"><span><%=rb.getString("ShangYiBu")%></span></a>
				<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="saveConfigBackupTask(this)"><span><%=rb.getString("QueDing")%></span></a>
				<a class="easyui_linkbutton linkbutton linkbutton_trend" href="javascript:void(0)" onclick="cancelCreateConfigBackupTask()"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
		</div>
	</div>
</div>

<%-- 工具栏-基站列表 --%>
<div id="toolbar_GridCell_addTask_configBackup" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery">
	 	<li class="select_reset" style="margin-left:27px;margin-right:5px;">
			<select id = "configBackupDeviceGroup" name="type" class="border border-box" style="width:100px;margin-right:1px;height: 26px;"></select>
		</li>
		<li class="serial_number input_li">
			<input name="value" type="text" style="width:154px;" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)" />
		</li>
		<li >
			<b class="searchResultImgChangeStyle" onclick="$('#gridCell_addTask_configBackup').datagrid('reload');" ></b>	
		</li>
	</ul>
	
</div>

<script type="text/javascript">

var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在基站列表加载前

$(function() {
	closeLoading();
	var ele = $("#exeTime_addConfigBackupTask");
	disableSelectEarlyTime(ele);
	
	$("#taskName_addConfigBackupTask").val("${addTaskName}");
	$("#addConfigBackupTaskSteps #step_2").hide();
	initCellGrid_addConfigBackupTask();
	$("#cellSearch_addConfigBackupTask").bind('keyup', function(e) {
		if (e["keyCode"] == 13) {
			$("#cellTree_addConfigBackupTask").tree("reload");
		}
	});
	
	$("#gridCell_addTask_configBackup").datagrid({
		url: '${ctx}/cell/cpeinfos/queryCpeInfosList.action?forSelect=1',
		queryParams:{like_fields:'serial_number,host_name'},
		singleSelect:false,
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		pagePosition:'bottom',
		pageSize : 100,
		pageList : [100],
		idField:'small_cell_code',
		toolbar:'#toolbar_GridCell_addTask_configBackup',
		pagination : true,
		striped: true,
		onLoadError:datagridLoadError,
		onLoadSuccess:loadSuccess_addTask_configBackup,
		onBeforeLoad:beforeLoad_gridCell_addTask_configBackup,
		columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status',sortable:true,fixed:true,width: 30,formatter:connStatusFormatter},
			{field: 'serial_number',sortable:true,width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name',sortable:true,width: 100, title: '<%=rb.getString("HostName")%>'}
		]]
	});
	
	$("#gridCell_addTask_configBackup").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
	
	$("#configBackupDeviceGroup").combobox({
    	url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action',
        width: 100,
        panelWidth: 150,
        panelHeight: 200,
        valueField: 'id',
        textField: 'group_name',
        onSelect: chooseConfigBackupDeviceGroup
  });
	
});
//保存当前已选择的基站对象数组
var delCellCodeObjArr = new Array();

$("#configBackupDeviceGroup").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']);

function chooseConfigBackupDeviceGroup(data){
	choosedGroupId = data.id;
    $("#gridCell_addTask_configBackup").datagrid("reload");
}
<%-- 基站列表加载完成事件--%>
function loadSuccess_addTask_configBackup() {
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	
	//初始添加-行操作提示记录
    $("#selectedCell_addConfigBackupTask").datagrid({
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
			var rowIndex = $("#gridCell_addTask_configBackup").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["small_cell_code"]);
			if (rowIndex != -1) {
				$("#gridCell_addTask_configBackup").datagrid("deleteRow",rowIndex);
			}
		}
		//向右侧列表添加已经选中的基站
		for (var i = 0; i < delCellCodeObjArr.length; i++) {
			var row = {value: delCellCodeObjArr[i]["small_cell_code"], text: delCellCodeObjArr[i]["serial_number"] + "(" + delCellCodeObjArr[i]["host_name"] + ")"};
			$("#selectedCell_addConfigBackupTask").datagrid("appendRow", row);
		}
		 
		var rows = $("#selectedCell_addConfigBackupTask").datagrid("getRows");
		if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
			$("#selectedCell_addConfigBackupTask").datagrid("deleteRow", 0);
		}
	}
}
<%-- 初始化基站树 --%>
function initCellGrid_addConfigBackupTask() {
	<%-- 基站搜索-回车事件 --%>
    $("#toolbar_GridCell_addTask_configBackup input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_addTask_configBackup").datagrid("reload");
        }
    });
    
    <%-- 基站搜索-软件版本-下拉面板 --%>
	$("#softwareVersionCombo_addTask_configBackup").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	});
	
	<%-- 基站搜索-类型-change事件 --%>
	$("#toolbar_GridCell_addTask_configBackup select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_addTask_configBackup .input_li").hide();
		$("#toolbar_GridCell_addTask_configBackup ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_addTask_configBackup").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	});
	
	<%-- 基站搜索-设备组树-下拉面板 --%>
	$("#regnTreeCombo_addTask_configBackup").combotree({
		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
		panelWidth: 200
	});
	
	$(".group_id input.textbox-text").css("padding", "0 10px").css("margin", "0");
}
<%-- 下一步  --%>
function stepDown_addConfigBackupTask() {
	if (validateStep1_addConfigBackupTask()) {
		// 改变窗口大小
		/* $("#winAddConfigBackupTask").window("resize", {
			height: 200,
			width: 500
		}).window("center"); */
		setDefaultWindow({
			height: 200,
			width: 500
		});
		
		 var dateTimeVal = $("#exeTime_addConfigBackupTask").datetimebox("getValue");
		$("#addConfigBackupTaskSteps #step_1").hide();
        $("#addConfigBackupTaskSteps #step_2").show();
        $("#addConfigBackupTaskSteps").css("width", "360").css("height", "130");
        /* $.parser.parse($("#winAddConfigBackupTask")); */
        $.parser.parse($(winDefaultSelector)); 
        $("#exeTime_addConfigBackupTask").datetimebox("setValue",dateTimeVal)
	}
}
<%-- 上一步 --%>
function stepUp_addConfigBackupTask() {
	// 改变窗口大小
	/* $("#winAddConfigBackupTask").window("resize", {
        height: document.body.clientHeight * 0.9,
        width: 960
    }).window("center"); */
    setDefaultWindow({
    	height: document.body.clientHeight * 0.9,
        width: 960
	});
	
    var dateTimeVal = $("#exeTime_addConfigBackupTask").datetimebox("getValue");
    
	$("#addConfigBackupTaskSteps #step_1").show();
    $("#addConfigBackupTaskSteps #step_2").hide();
    $("#addConfigBackupTaskSteps").css("width", "100%").css("height", "100%");
    /* $.parser.parse($("#winAddConfigBackupTask")); */
    $.parser.parse($(winDefaultSelector));
    $("#exeTime_addConfigBackupTask").datetimebox("setValue",dateTimeVal)
}

<%-- 验证第一步输入 --%>
function validateStep1_addConfigBackupTask() {
	// 验证任务名称是否填写
	if ($("#taskName_addConfigBackupTask").val().trim().length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>");
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/configBackup/taskNameExist.action", 
			data: {"taskName": $("#taskName_addConfigBackupTask").val().trim()},
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
	var selCells = $("#selectedCell_addConfigBackupTask").datagrid("getRows");
	if (selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	// 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	if (document.getElementById("timing_addConfigBackupTask").checked
			&& $("#exeTime_addConfigBackupTask").datetimebox("getValue").length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeShiJian")%>");
		return false;
	}
	return true;
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable_addConfigBackupTask() {
	if (document.getElementById("timing_addConfigBackupTask").checked) {
		$("#exeTime_addConfigBackupTask").datetimebox("enable");
	} else {
		$("#exeTime_addConfigBackupTask").datetimebox("disable");
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_addConfigBackupTask() {
	var dl = $("#selectedCell_addConfigBackupTask");
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
            				$("#gridCell_addTask_configBackup").datagrid("appendRow",row);
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
    			var rowIndex = $("#selectedCell_addConfigBackupTask").datagrid("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_addConfigBackupTask").datagrid("deleteRow",rowIndex);
    		}
    	}
    	var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		
		//将复选框取消选中
		$("#selectedCell_addConfigBackupTask").datagrid("uncheckAll");
	}
}
<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedCell_addConfigBackupTask() {
	var selCell = $("#gridCell_addTask_configBackup").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_addConfigBackupTask");
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
			var rowIndex = $("#gridCell_addTask_configBackup").datagrid("getRowIndex", newSelectdArr[cellCount]["small_cell_code"]);
			$("#gridCell_addTask_configBackup").datagrid("deleteRow",rowIndex);
		}
	}
	
	$("#gridCell_addTask_configBackup").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveConfigBackupTask(ele) {
	var params = {};
	params.timeZone=timeZone;
	params["taskName"] = $("#taskName_addConfigBackupTask").val().trim();
	
	// 取得已选择的基站
	var cellCodes = "";
	var selCells = $("#selectedCell_addConfigBackupTask").datagrid("getRows");
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
		params["time"] = $("#exeTime_addConfigBackupTask").datetimebox("getValue");
	}
	//已经点击了确认按钮，禁用“确认按钮”，以防止服务器响应慢而导致重复点击按钮
	$(ele).linkbutton('disable');
	
	$.post("${ctx}/task/configBackup/addTask.action", params, function(data) {
		if (data["success"]) {
			/* $("#winAddConfigBackupTask").window("close"); */
			closeDefaultWindow();
			$("#configBackupTaskList").datagrid("load");
		} else {
			$(ele).linkbutton('enable');
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

// 取消新建任务
function cancelCreateConfigBackupTask() {
	/* $("#winAddConfigBackupTask").window("close"); */
	closeDefaultWindow();
}

//加载前事件-基站列表
function beforeLoad_gridCell_addTask_configBackup(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $(".inputslist li.serial_number>input").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	
	/* var type = $("#toolbar_GridCell_addTask_configBackup select[name='type']").val();
	var val = $("#toolbar_GridCell_addTask_configBackup ." + type + " input[name='value']").val();
	param[type] = val; */
}
</script>