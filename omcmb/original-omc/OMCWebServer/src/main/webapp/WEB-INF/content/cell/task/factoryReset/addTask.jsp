<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<%-- 新建重启任务 --%>
<style>
	.dateRe .datebox{
		background:#F5F7FA
	}
	.dateRe .datebox .textbox-prompt{
			background:#F5F7FA
	}
	.bgF5{
	
	}
</style>
<div class="slidebarTitleDiv">
	<div style="display: inline-block;"><%=rb.getString("XinJianChongZhiRenWu")%></div>
</div>
<div class="slideBody">
	<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("GuanBi")%>" onclick="cancelCreateRebootTask()">
		<span class="el-icon el-icon-circle-close"></span>
	</div>
	<%-- 第一步 --%>
	<div id="eNodeB_resettask_step_1" class="easyui-layout" data-options="border:false" style="width: 100%;height: 100%;">
		<div region="center" data-options="border:false">
			<div class="easyui-layout" style="width: 98.5%;height: 100%;min-width: 800px;">
			   <div region="north" data-options="border:false,height:66" style="padding: 20px;">
			       <%-- 任务名称 --%>
			       <span style="display:inline-block;width:72px;"><%=rb.getString("RenWuMingCheng")%></span>
			       <input id="taskName_addResetTask" type="text" class="border border-box" style="width:600px;margin-left:6px;"
			           maxlength=50 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
			   </div>
			   <div region="center" data-options="border:false" style="padding: 0 20px;">
			       <div class="easyui-layout" data-options="border:false,fit:true">
			           <div region="west" data-options="width: '48%',border:true,collapsible:false" title="<%=rb.getString("JiZhanLieBiao")%>" style="padding:20px 0 0">
			               <table id="gridCell_addTask_reset"></table>
			           </div>
			           <div region="center" data-options="width: '5%',border:false,onResize:setArrowMargin">
			               <a id="arrow_right_addResetTask" href="javascript:void(0);" class="el-icon el-icon-common-arrow-right" style='margin-top:233px;font-size:30px;margin-left:15px;' onclick="addSelectedCell_addRebootTask()"></a>
			               <a href="javascript:void(0);" class="el-icon el-icon-common-arrow-left" style="margin-top: 20px;font-size:30px;margin-left:15px;" onclick="delSelectedCell_addRebootTask()"></a>
			           </div>
			           <div class="special2" region="east" data-options="width:'47%',border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeJiZhan")%>'" style="padding-top:20px;" >
			               <%-- 已选择基站 --%>
			               <table class="easyui-datagrid" id="selectedCell_addResetTask"
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
			           <div region="south" data-options="height:150,border:false,collapsible:false" style="margin-top:20px">
			               <div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding: 20px;">
			                   <div class="verM">
			                       <input type="radio" id="active_addResetTask" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable_addRebootTask()"/>
			                       <label for="active_addResetTask" style="display:inline-block;width:390px;"><%=rb.getString("LiJiZhiXing")%></label>
			                       <input type="radio" id="suspend_addResetTask" status="suspend" style="margin-left: 40px;" name="taskStatus" onchange="setExeTimerEnable_addRebootTask()"/>
			                       <label for="suspend_addResetTask"><%=rb.getString("GuaQi")%></label>
			                       
			                   </div>
			                   <div  class="dateRe" style="margin-top: 10px;">
			                       <input type="radio" id="timing_addResetTask" status="timing" name="taskStatus" onchange="setExeTimerEnable_addRebootTask()"/>
			                       <label for="timing_addResetTask" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
			                       <input id="exeTime_addResetTask" class="easyui-datetimebox border-box border bgF5" style="height:26px;"
			                           data-options="disabled:true,editable:false">
			                   </div>
			               </div>
			           </div>
			       </div>
			   </div>
			</div>
	    </div>
	    <div region="south" data-options="border:false,height:71" style="padding: 20px 20px 10px; ">
	      <%--  <a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveRsetTask(this)" style="float: right;margin-right:15px;"><span><%=rb.getString("XiaYiBu")%></span></a> --%>
	       <div class="windowButtonGroup" style="float:none">
	   	   	    <%-- <a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUp_addRebootTask()"><span><%=rb.getString("ShangYiBu")%></span></a> --%>
		        <a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveRsetTask(this)"><span><%=rb.getString("QueDing")%></span></a>
		        <a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelCreateRebootTask()"><span><%=rb.getString("QuXiao")%></span></a>
	   	   </div>
	    </div>
	</div>
	
	<%-- 第二步 --%>
	<div id="eNodeB_resettask_step_2" data-options="border:false">
	   <div style="height:30px;line-height:28px;margin:20px;text-align:center;">
	       <div class="question-mark" style="float:left;"></div>
	       <div id="resetprompt" style="display:float:left;margin-left:10px"><%=rb.getString("QueDingHuiFuMoRenPeiZhi")%></div>
	   </div>
	   <div style="height:30px;text-align:center;">
	   	   <div class="windowButtonGroup" style="float:none">
	   	   	    <a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUp_addRebootTask()"><span><%=rb.getString("ShangYiBu")%></span></a>
		        <a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveRsetTask(this)"><span><%=rb.getString("QueDing")%></span></a>
		        <a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelCreateRebootTask()"><span><%=rb.getString("QuXiao")%></span></a>
	   	   </div>
	    </div>
	</div>
</div>
<%-- 工具栏-基站列表 --%>
<div id="toolbar_GridCell_addTask_reset" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery">
	 	<li  class="select_reset" style="margin:0 10px;">
			<select id = "resetDeviceGroup" name="type" class="border border-box easyui-combobox" data-options="editable:false,value:-1" style="width:103px;margin-right:1px;height: 26px;"></select>
		</li>
		<div class='queryGroup'>
			<input name="value" style="width:145px;margin-left:0px;" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>" class="searchInputStyle">
			<b class="el-icon el-icon-common-search" onclick="$('#gridCell_addTask_reset').datagrid('reload');" ></b>	
		</div>
		<%-- <li class="serial_number input_li">
			<input name="value" type="text" class="border border-box" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>">
			<input name="value" style="width:180px;margin-left:0px" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		</li>
		<li>
			<b class="searchResultImgChangeStyle" onclick="$('#gridCell_addTask_reset').datagrid('reload');" ></b>	
		</li> --%>
	</ul>
</div>

<script type="text/javascript">
var lang = "${language}";
var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在基站列表加载前

$(function() {
	var objele = $("#exeTime_addResetTask")
	disableSelectEarlyTime(objele);
	closeLoading();
	
	$("#taskName_addResetTask").val("${addTaskName}");
	$("#eNodeB_resettask_step_2").hide();
	initCellGrid_addRebootTask();
	$("#cellSearch_addRebootTask").bind('keyup', function(e) {
		if (e["keyCode"] == 13) {
			$("#cellTree_addRebootTask").tree("reload");
		}
	});
	
	$("#gridCell_addTask_reset").datagrid({ // 初始化列表数据
		border: false,
		fitColumns: true,
		fit: true,
        rownumbers: true,
        url: '${ctx}/cell/cpeinfos/queryCpeInfosList.action?forSelect=1',
        queryParams:{like_fields:'serial_number,host_name'},
        pageSize: 100,
        pageList: [100],
        striped: true,
        singleSelect: false,
        pagination: true,
        pagePosition: 'bottom',
        idField: 'small_cell_code',
        onBeforeLoad: beforeLoad_gridCell_addTask_reboot,
        onLoadSuccess: loadSuccess_addTask_reboot,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_GridCell_addTask_reset',
        onCheckAll:checkAll_reboot,
        onBeforeCheck:beforeCheck_reboot,
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'serial_number', sortable: true, width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name', sortable: true, width: 100, title: '<%=rb.getString("HostName")%>'}
		]]
	});
	
	$("#gridCell_addTask_reset").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
	
	$("#resetDeviceGroup").combobox({
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
/**
 * 全选
 * @param rows:选择的数据
*/
function checkAll_reboot(rows){ 
	$(this).datagrid("initRow").datagrid("getPanel").find(".datagrid-htable .datagrid-header-check>input").prop("checked",true);
}
/**
 * 复选框操作
 * @param index:下标
 * @param row:选择的数据
*/
function beforeCheck_reboot(index,row){
	var bool = $(this).datagrid("checkUnable",row);
	if(bool){
		return !bool;
	}
}
/**
 * 单选操作
 * @param data:选择的数据
*/
function chooseRebootDeviceGroup(data){
	choosedGroupId = data.id;
    $("#gridCell_addTask_reset").datagrid("reload");
}
// 加载成功时触发的函数
function loadSuccess_addTask_reboot() {
	$(this).datagrid("initRow");
	$(this).datagrid("fixRownumber");
	 $(this).datagrid("enableContextmenuAutoSize");
	
	//初始添加-行操作提示记录
    $("#selectedCell_addResetTask").datagrid({
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
    		var rowIndex = $("#gridCell_addTask_reset").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["small_cell_code"]);
    		if (rowIndex != -1) {
    			//$("#gridCell_addTask_reset").datagrid("deleteRow",rowIndex);
    			$("#gridCell_addTask_reset").datagrid("setRow",{small_cell_code:delCellCodeObjArr[cellCount]["small_cell_code"],usable:false});
    		}
    	}
    	//向右侧列表添加已经选中的基站
    	for (var i = 0; i < delCellCodeObjArr.length; i++) {
    		var row = {value: delCellCodeObjArr[i]["small_cell_code"], text: delCellCodeObjArr[i]["serial_number"] + "(" + delCellCodeObjArr[i]["host_name"] + ")"};
    		$("#selectedCell_addResetTask").datagrid("appendRow", row);
    	}
    	 
    	var rows = $("#selectedCell_addResetTask").datagrid("getRows");
    	if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
    		$("#selectedCell_addResetTask").datagrid("deleteRow", 0);
    	}
    }
}
// 初始化基站树 
function initCellGrid_addRebootTask() {
	<%-- 基站搜索-回车事件 --%>
    $("#toolbar_GridCell_addTask_reset input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_addTask_reset").datagrid("reload");
        }
    });
    
    <%-- 基站搜索-软件版本-下拉面板 --%>
	$("#softwareVersionCombo_addTask_reboot").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	});
	
	<%-- 基站搜索-类型-change事件 --%>
	$("#toolbar_GridCell_addTask_reset select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_addTask_reset .input_li").hide();
		$("#toolbar_GridCell_addTask_reset ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_addTask_reboot").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	});
	
	<%-- 基站搜索-地域树-下拉面板 --%>
	$("#regnTreeCombo_addTask_reboot").combotree({
		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
		panelWidth: 200
	});
	
	$(".group_id input.textbox-text").css("padding", "0 10px").css("margin", "0");
}
// 下一步  
function stepDown_addRebootTask() {
	var height = 177;
	var radios = document.getElementsByName("taskStatus");
	var reg = /^en.*$/;
	if(reg.test(lang)){
		var $div0 = $($("#eNodeB_resettask_step_2 div").get(0));
		$div0.css({"height":"55px","line-height":"28px"});
		var $div1 = $($("#eNodeB_resettask_step_2 div").get(1));
		$div1.css({"height":"28px","line-height":"28px"});
		var height = 200;
		var $image = $($div0.find("div").get(0));
		$image.css({"height":"38px"});
	}
	if (eNodeB_resettask_step_1lidateStep1_addRebootTask()) {
		$("#eNodeB_resettask_step_1").hide();
		// 改变窗口大小
		/* $("#winAddRebootTask").window("resize", {
			height: height,
			width: 400
		}).window("center"); */
		setDefaultWindow({
			height: height,
			width: 400
		});
		$("#eNodeB_resettask_step_2").show();
		$("#eNodeB_resettask_step_2").panel("doLayout");
	}
	for (var i = 0; i < radios.length; i++) {
		if (radios[i].checked == true && $(radios[i]).attr("status") == "suspend") {
			$("#resetprompt").html("<%=rb.getString("QueDingXinJianHuiFuRenWuErBuLiJiChongQiJiZhan")%>");			
		} else if(radios[i].checked == true && $(radios[i]).attr("status") == "active") {
			$("#resetprompt").html("<%=rb.getString("QueDingHuiFuMoRenPeiZhi")%>");
		} else if (radios[i].checked == true && $(radios[i]).attr("status") == "timing") {
			$("#resetprompt").html("<%=rb.getString("QueDingXinJianHuiFuRenWuErBuLiJiChongQiJiZhan")%>");
		}
	}
}
// 上一步 
function stepUp_addRebootTask() {
	$("#eNodeB_resettask_step_2").hide();
	// 改变窗口大小
	/* $("#winAddRebootTask").window("resize", {
		height: document.body.clientHeight * 0.9,
		width: 860
	}).window("center"); */
	$("#eNodeB_resettask_step_1").show();
	setDefaultWindow({
		height: document.body.clientHeight * 0.9,
		width: 860
	});
	
}

// 验证第一步输入 
function eNodeB_resettask_step_1lidateStep1_addRebootTask() {
	// 验证任务名称是否填写
	if ($("#taskName_addResetTask").val().trim().length == 0) {
		showMsg('prompt_msg','<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>');
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/factoryReset/taskNameExist.action", 
			data: {"taskName": $("#taskName_addResetTask").val().trim()},
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
			showMsg('prompt_msg','<%=rb.getString("RenWuMingChengYiCunZai")%>');
			return false;
		}
	}

	// 是否已选择小站
	var selCells = $("#selectedCell_addResetTask").datalist("getRows");
	if (selCells.length == 1 && selCells[0]["value"].length == 0) {
		showMsg('prompt_msg','<%=rb.getString("QingXuanZeSheBei")%>');
		return false;
	}
	// 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	if (document.getElementById("timing_addResetTask").checked
			&& $("#exeTime_addResetTask").datetimebox("getValue").length == 0) {
		showMsg('prompt_msg','<%=rb.getString("QingXuanZeShiJian")%>');
		return false;
	}
	return true;
}

// 控制定时执行的时间输入框启用/禁用 
function setExeTimerEnable_addRebootTask() {
	
	if (document.getElementById("timing_addResetTask").checked) {
		$("#exeTime_addResetTask").datetimebox("enable");
		$('.dateRe .datebox').css('background','#FFFFFF')
		$('.dateRe .datebox .textbox-prompt').css('background','#FFFFFF')
	} else {
		$("#exeTime_addResetTask").datetimebox("disable");
		$('.dateRe .datebox').css('background','#F5F4FA')
		$('.dateRe .datebox .textbox-prompt').css('background','#F5F4FA')
	}
}

//删除已选中的小站-向左键头的点击事件 
function delSelectedCell_addRebootTask() {
	var dl = $("#selectedCell_addResetTask");
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
            				//$("#gridCell_addTask_reset").datagrid("appendRow",row);
            				$("#gridCell_addTask_reset").datagrid("setRow",{small_cell_code:row["small_cell_code"],usable:true});
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
    			var rowIndex = $("#selectedCell_addResetTask").datagrid("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_addResetTask").datagrid("deleteRow",rowIndex);
    		}
    	}
    	
		var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		//将复选框取消选中
		$("#selectedCell_addResetTask").datagrid("uncheckAll");
	}
}

// 选择小站-向右键头的点击事件
function addSelectedCell_addRebootTask() {
	var selCell = $("#gridCell_addTask_reset").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_addResetTask");
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
			var rowIndex = $("#gridCell_addTask_reset").datagrid("getRowIndex", newSelectdArr[cellCount]["small_cell_code"]);
			//$("#gridCell_addTask_reset").datagrid("deleteRow",rowIndex);
			$("#gridCell_addTask_reset").datagrid("setRow",{small_cell_code:newSelectdArr[cellCount]["small_cell_code"],usable:false});
		}
	}
	
	$("#gridCell_addTask_reset").datagrid("uncheckAll");
}

// 保存新建的任务 
function saveRsetTask(ele) {
	if(!eNodeB_resettask_step_1lidateStep1_addRebootTask()){
		return false;
	}
	var params = {};
	params.timeZone=timeZone;
	params["taskName"] = $("#taskName_addResetTask").val().trim();
	
	// 取得已选择的基站
	var cellCodes = "";
	var selCells = $("#selectedCell_addResetTask").datalist("getRows");
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
		params["time"] = $("#exeTime_addResetTask").datetimebox("getValue");
	}
	//已经点击了确认按钮，禁用“确认按钮”，以防止服务器响应慢而导致重复点击按钮
	//$(ele).linkbutton('disable');
	
	$.post("${ctx}/task/factoryReset/addTask.action", params, function(data) {
		if (data["success"]) {
			/* $("#winAddRebootTask").window("close"); */
			//closeDefaultWindow();
			showMsg('success_msg','<%=rb.getString("ChengGong")%>');
			cancelCreateRebootTask();
			$("#resetTaskList").datagrid("load");
		} else {
			//$(ele).linkbutton('enable');
			cancelCreateRebootTask();
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

// 取消新建任务
function cancelCreateRebootTask() {
	$("#winAddRebootTask").slideUp();
	/* closeDefaultWindow(); */
}

//加载前事件-基站列表
function beforeLoad_gridCell_addTask_reboot(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $("#toolbar_GridCell_addTask_reset .inputslist .queryGroup>input").val();
	if(search_text != ""){
		param["searchText"] = search_text;
	}
	param["isShowSlave"] = false;
}

/**
 * 比较两个时间
 * @param pre:开始时间
 * @param after:结束时间
*/
function differ(pre,after){
	  function initDate(date){
		  var d = new Date(date.getFullYear(), (date.getMonth()+1), date.getDate())
		  return d;
	  }
      var preDate = initDate(new Date(pre)), afterDate = initDate(new Date(after));
      var differTimes = preDate.valueOf() - afterDate.valueOf();
      return Math.round(differTimes/(24 * 60 * 60 * 1000));
}
//时间选择表  不能选择当前之前的日期
function disableSelectEarlyTime(ele){
	var now = new Date(gloableTime);
	var d1 = new Date(now.getFullYear(), now.getMonth(), now.getDate());
	now = new Date(d1);
	ele.datetimebox({}).datetimebox('calendar').calendar({
		validator: function(date) {
			if(differ(date,now)<0){
				return false;
			}else{
				return true;
			}
		}
	});
}
</script>