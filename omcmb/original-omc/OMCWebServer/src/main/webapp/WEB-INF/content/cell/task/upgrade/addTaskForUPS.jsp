<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<%-- 新建升级任务 --%><div class="slidebarTitleDiv">
	<div style="display: inline-block;"><%=rb.getString("XinJianShengJiRenWu")%></div>
</div>
<div class="slideBody">
	<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("GuanBi")%>" onclick="cancelCreateUpgradeTask_ups_common()">
		<span class="el-icon el-icon-circle-close"></span>
	</div>
	<div id="addUpgradeTaskSteps_ups_common" class="easyui-panel" data-options="border:false" style="width: 100%;height: 98%;">
		<%-- 第一步 --%>
		<div id="step_1" class="easyui-layout" data-options="border:false" style="width: 100%;height: 100%;">
		   <div region="center" data-options="border:false">
	            <div class="easyui-layout" style="width: 98.5%;height: 100%;min-height: 720px;">
					<div region="north" data-options="border:false,height:66" style="padding:20px;">
						<%-- 任务名称 --%>
						<label style="display: inline-block;width: 72px;"><%=rb.getString("RenWuMingCheng")%></label>
						<input id="taskName_addUpgradeTask_ups_common" type="text" class="border border-box" style="width:600px;margin-left:8px;"
							maxlength=100 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
					</div>
					<div region="center" data-options="border:false" style="padding:0  20px 20px;">
						<div class="easyui-layout" data-options="border:false,fit:true">
							<div class="special" region="west" data-options="width:'48%',border:true,collapsible:false" title="<%=rb.getString("UPSLieBiao")%>" style="padding-top:20px;">
								<table id="gridCell_addTask_upgrade_ups_common"></table>
							</div>
							<div region="center" data-options="width: '5',border:false,onResize:setArrowMargin">
								<a id="arrow_right_addUpgradeTask" href="javascript:void(0);" class="el-icon el-icon-common-arrow-right" style='font-size:30px;margin-top:233px;margin-left:15px;' onclick="addSelectedCell_addUpgradTask_common_ups()"></a>
								<a href="javascript:void(0);" class="el-icon el-icon-common-arrow-left" style="font-size:30px;margin-left:15px;margin-top: 20px;" onclick="delSelectedCell_addUpgradTask_common_ups()"></a>
							</div>
							<div class="special2" region="east" data-options="width:'47%',border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeUPS")%>'" style="padding-top:20px;">
								<%-- 已选择CPE --%>
						    	<table class="easyui-datagrid" id="selectedCell_addUpgradTask_ups"
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
												<%=rb.getString("DianYuanBianMa")%>
											</th>
										</tr>
									</thead>
						  		</table>
							</div>
							<div region="south" data-options="height:170,border:false,collapsible:false" style="padding-top: 20px;">
								<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding: 20px;">
									<div class="easyui-layout" data-options="fit:true,border:false">
										<div region="west" data-options="border:false">
											<div  class="verM">
												<input type="radio" id="active_addUpgradeTask_ups_common" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable()"/>
												<label for="active_addUpgradeTask_ups_common" style="display:inline-block;width:387px;"><%=rb.getString("LiJiZhiXing")%></label>
												<input type="radio" id="suspend_addUpgradeTask_ups_common" status="suspend"  name="taskStatus" onchange="setExeTimerEnable()"/>
												<label for="suspend_addUpgradeTask_ups_common"><%=rb.getString("GuaQi")%></label>
										
											</div>
											<div style="margin-top: 10px;" class="dateRe">
												<input type="radio" id="timing_addUpgradeTask_ups_common" status="timing" name="taskStatus" onchange="setExeTimerEnable()"/>
												<label for="timing_addUpgradeTask_ups_common" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
												<input id="exeTime_addUpgradeTask_ups_common" class="easyui-datetimebox border-box border" style="height:26px;"
													data-options="disabled:true,editable:false">
											</div>
										</div>
										<div region="center" data-options="border:false">
											<label hidden="true"><%=rb.getString("BaoLiuPeiZhi")%></label>
											<select id="upgradeRawModeFlagupsCommon" class="border border-box" style="margin-left:15px" hidden="true">
												<option value="false" selected="selected"><%=rb.getString("Shi")%></option>
												<option value="true"><%=rb.getString("Fou")%></option>
											</select>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div region="south" data-options="border:false,height:69" style="padding: 30px 20px 0px;">
				<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="stepDown_addUpgradeTask_ups_common()" style="float: right;">
					<span><%=rb.getString("XiaYiBu")%></span>
				</a>
			</div>
		</div>
		<%-- 第二步 --%>
		<div id="step_2" class="easyui-panel" data-options="border:false,fit:true">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="center" data-options="border:false" style="padding: 10px 20px;">
					<table class="easyui-datagrid" id="gridFiles_addUpgradeFiles_common_ups" title="<%=rb.getString("XuanZeShengJiWenJian")%>"
						data-options="singleSelect:true,rownumbers:true,pagination:true,pagePosition:'bottom',onLoadSuccess:datagridLoadSuccess,
							url: '${ctx}/cell/version/queryfileInfosList.action?file_type=5',queryParams:{timeZone:timeZone},fitColumns:false,fit:true,
							onSelect : onSelect_softwareFile_cpe ,
							border:true,striped:true">
			            <thead>
			            <tr>
			                <th data-options="field:'id',hidden:true"><%=rb.getString("WenJianBiaoShi")%></th>
			                <th data-options="field:'operate',formatter: radioMatterCPE" width="50" align='center' halign='center'></th>
			                <th data-options="field:'file_name'" width="200"><%=rb.getString("WenJianMing")%></th>
			                <%-- <th data-options="field:'manufacturer'" width="80"><%=rb.getString("ZhiZaoShang")%></th> --%>
			                <th data-options="field:'product'" width="60"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></th>
			                <th data-options="field:'size'" width="100"><%=rb.getString("WenJianDaXiao")%></th>
							<th data-options="field:'version'" width="80"><%=rb.getString("BanBen")%></th>
			                <th data-options="field:'upload_time'" width="140"><%=rb.getString("ShangChuanShiJian")%></th>
			                <th data-options="field:'uploader'" width="80"><%=rb.getString("ShangChuanZhe")%></th>
					        <th data-options="field:'desc'" width="100"><%=rb.getString("MiaoShu")%></th>
					        <%-- <th data-options="field:'md5_error'" width="100"><%=rb.getString("WenJianJiaoYanCuoWu")%></th> --%>
			            </tr>
			            </thead>
			        </table>
				</div>
				<div region="south" data-options="border:false,height:59" style="padding: 20px 0px 0px 0px;">
					<div class="windowButtonGroup" style="margin-right:20px;">				
						<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUP_addUpgradeTask_ups_common()" ><span><%=rb.getString("ShangYiBu")%></span></a>
						<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveUpgradeTask_ups_common(this)"><span><%=rb.getString("WanCheng")%></span></a>
						<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelCreateUpgradeTask_ups_common()"><span><%=rb.getString("QuXiao")%></span></a>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

<%-- 工具栏-CPE列表 --%>
<div id="toolbar_GridCell_addTask_upgrade_ups_common" class="admin_query_head" style="background-color:white;">
	<%-- <ul class="inputslist">
		<li>
			<select name="type" class="border border-box" style="width:100px;margin-right:1px;">
				<option value="serial_number"><%=rb.getString("XiaoZhanBianMa")%></option>
				<option value="host_name"><%=rb.getString("HostName")%></option>
				<option value="software_version"><%=rb.getString("SoftwareVersion")%></option>
				<option value="group_id"><%=rb.getString("SheBeiZu")%></option>
			</select>
		</li>
		<li class="serial_number input_li">
			<input name="value" type="text" class="border border-box">
		</li>
		<li class="host_name input_li" style="display: none;">
			<input name="value" type="text" class="border border-box">
		</li>
		<li class="software_version input_li" style="display: none;">
			<input id="softwareVersionCombo_addTask_upgrade" name="value" class="border border-box" style="height:26px;">
		</li>
		<li class="group_id input_li" style="display: none;">
			<input id="regnTreeCombo_addTask_upgrade" name="value" class="border border-box" style="height:26px;">
		</li>
		<li>
			<a onclick="$('#gridCell_addTask_upgrade').datagrid('reload');" style="float:left;margin-left:15px;"
					class="easyui-linkbutton"><%=rb.getString("SouSuo")%></a>
		</li>
	</ul> --%>
	<div style="margin-left:10px;">	
		<ul class="inputslist">
		 	<li class="select_reset">
				<select id = "upgradeDeviceGroup" name="type" class="border border-box" style="width:100px;margin-right:1px;height: 26px;"></select>
			</li>
			<div class='queryGroup'>
				<input name="value" style="width:140px;" placeholder="<%=rb.getString("QingShuRuUPSBianMaJinXingChaXun")%>"/>
				<b onclick="$('#gridCell_addTask_upgrade_ups_common').datagrid('reload');"></b>	
			</div>
		</ul>
		</div>
</div>

<script type="text/javascript">

var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在CPE列表加载前

$(function() {
 	var objele = $("#exeTime_addUpgradeTask_ups_common");
	disableSelectEarlyTime(objele); 
	closeLoading();
	
	$("#taskName_addUpgradeTask_ups_common").val("${addTaskName}");
	
	$("#step_2").hide();
	
	// 声明CPE列表
	$("#gridCell_addTask_upgrade_ups_common").datagrid({
		border: false,
		fitColumns: false,
		fit: true,
        rownumbers: true,
        url: '${ctx}/ups/queryUpsInfosListByGroupId.action',
        pageSize: 100,
        pageList: [100],
        striped: true,
        singleSelect: false,
        pagination: true,
        pagePosition: 'bottom',
        idField: 'ups_code',
        onBeforeLoad: beforeLoad_gridCell_addTask_upgrade_cpe_common,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_GridCell_addTask_upgrade_ups_common',
        onCheckAll:checkAll_upgrade_cpe,
        onBeforeCheck:beforeCheck_ugrade_cpe,
        onLoadSuccess: loadSuccess_addTask_upgrade,
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'ups_code', hidden: true},
			{field: 'connection_status', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'serial_number', sortable: true, width: 200, title: '<%=rb.getString("DianYuanBianMa")%>'},
			<%-- {field: 'host_name', sortable: true, width: 200, title: '<%=rb.getString("CpeName")%>'} --%>
		]]
	});
	
	$("#gridCell_addTask_upgrade_ups_common").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
	
	initCellGrid_addUpgradeTask();
	
	$("#upgradeDeviceGroup").combobox({
	    	url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action',
	        width: 100,
	        panelWidth: 150,
	        panelHeight: 200,
	        valueField: 'id',
	        textField: 'group_name',
	        onSelect: chooseUpgradeDeviceGroup
	 });
});
//保存当前已选择的CPE对象数组
var delCellCodeObjArr = new Array();

$("#upgradeDeviceGroup").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']); 

function checkAll_upgrade_cpe(rows){
	$(this).datagrid("initRow").datagrid("getPanel").find(".datagrid-htable .datagrid-header-check>input").prop("checked",true);
}
function beforeCheck_ugrade_cpe(index,row){
	var bool = $(this).datagrid("checkUnable",row);
	if(bool){
		return !bool;
	}
}
function chooseUpgradeDeviceGroup(data){
	choosedGroupId = data.id;
    $("#gridCell_addTask_upgrade_ups_common").datagrid("reload");
}

<%-- CPE列表加载完成事件--%>
function loadSuccess_addTask_upgrade() {
	$(this).datagrid("initRow");
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	
	//初始添加-行操作提示记录
    $("#selectedCell_addUpgradTask_ups").datagrid({
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
			var rowIndex = $("#gridCell_addTask_upgrade_ups_common").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["ups_code"]);
			if (rowIndex != -1) {
				//$("#gridCell_addTask_upgrade_ups_common").datagrid("deleteRow",rowIndex);
				$("#gridCell_addTask_upgrade_ups_common").datagrid("setRow",{ups_code:delCellCodeObjArr[cellCount]["ups_code"]});
			}
		}
		//向右侧列表添加已经选中的CPE
		for (var i = 0; i < delCellCodeObjArr.length; i++) {
			var row = {value: delCellCodeObjArr[i]["ups_code"], text: delCellCodeObjArr[i]["serial_number"] + "(" + delCellCodeObjArr[i]["host_name"] + ")"};
			$("#selectedCell_addUpgradTask_ups").datagrid("appendRow", row);
		}
		 
		var rows = $("#selectedCell_addUpgradTask_ups").datagrid("getRows");
		if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
			$("#selectedCell_addUpgradTask_ups").datagrid("deleteRow", 0);
		}
	}
}
<%-- 初始化CPE列表--%>
function initCellGrid_addUpgradeTask() {
	<%-- CPE搜索-回车事件 --%>
    $("#toolbar_GridCell_addTask_upgrade_ups_common input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_addTask_upgrade_ups_common").datagrid("reload");
        }
    });
    
    <%-- CPE搜索-软件版本-下拉面板 --%>
	$("#softwareVersionCombo_addTask_upgrade").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	});
	
	<%-- CPE搜索-类型-change事件 --%>
	$("#toolbar_GridCell_addTask_upgrade_ups_common select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_addTask_upgrade_ups_common .input_li").hide();
		$("#toolbar_GridCell_addTask_upgrade_ups_common ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_addTask_upgrade").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	});
	
	<%-- CPE搜索-地域树-下拉面板 --%>
	$("#regnTreeCombo_addTask_upgrade").combotree({
		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
		panelWidth: 200
	});
	
	$(".group_id input.textbox-text").css("padding", "0 10px").css("margin", "0");
}
<%-- 下一步  --%>
function stepDown_addUpgradeTask_ups_common() {
	if (validateStep1()) {
		$("#addUpgradeTaskSteps_ups_common #step_1").hide();
		$("#addUpgradeTaskSteps_ups_common #step_2").show();
		$('#addUpgradeTaskSteps_ups_common #step_2').panel("doLayout");		
	}
}
<%-- 上一步 --%>
function stepUP_addUpgradeTask_ups_common() {
	$("#addUpgradeTaskSteps_ups_common #step_2").hide();
	$("#addUpgradeTaskSteps_ups_common #step_1").show();
}

<%-- 验证第一步输入 --%>
function validateStep1() {
	// 验证任务名称是否填写
	if ($("#taskName_addUpgradeTask_ups_common").val().trim().length == 0) {
		showMsg('prompt_msg',"<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>");
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/upgrade/ups/taskNameExist.action", 
			data: {"taskName": $("#taskName_addUpgradeTask_ups_common").val().trim(),"taskType":"1"},
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
	var selCells = $("#selectedCell_addUpgradTask_ups").datagrid("getRows");
	if (selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		showMsg('prompt_msg',"<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	// 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	if (document.getElementById("timing_addUpgradeTask_ups_common").checked
			&& $("#exeTime_addUpgradeTask_ups_common").datetimebox("getValue").length == 0) {
		showMsg('prompt_msg',"<%=rb.getString("QingXuanZeShiJian")%>");
		return false;
	}
	return true;
}

<%-- 验证第二步输入 --%>
function validateStep2() {
	var selFile = $("#gridFiles_addUpgradeFiles_common_ups").datagrid("getSelected");
	if (selFile) {
		return true;
	} else {
		showMsg('prompt_msg',"<%=rb.getString("QingXianXuanZeWenJian")%>");
		return false;
	}
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable() {
	if (document.getElementById("timing_addUpgradeTask_ups_common").checked) {
		$("#exeTime_addUpgradeTask_ups_common").datetimebox("enable");
	} else {
		$("#exeTime_addUpgradeTask_ups_common").datetimebox("disable");
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_addUpgradTask_common_ups() {
	var dl = $("#selectedCell_addUpgradTask_ups");
	var selCell = dl.datagrid("getSelections");
	var selCellLength = selCell.length;
	var needDelCellArr = new Array(); //需要删除的小站编码数组
	
	if (selCellLength > 0) {
		for (var selCellCount = 0; selCellCount < selCell.length; selCellCount++) {
			if (selCell[selCellCount]["value"]) {
    			//从CPE列表中恢复CPE
        		for(var cellCount = 0; cellCount < delCellCodeObjArr.length; cellCount++) {
        			if (selCell[selCellCount].value == delCellCodeObjArr[cellCount]["ups_code"]) {
            			var row = {
            				ups_code: delCellCodeObjArr[cellCount]["ups_code"], 
            				serial_number: delCellCodeObjArr[cellCount]["serial_number"],
            				host_name: delCellCodeObjArr[cellCount]["host_name"],
            				connection_status: delCellCodeObjArr[cellCount]["connection_status"]
            			}
            			if(choosedGroupId == delCellCodeObjArr[cellCount]["groupId"]){//如果是当前组被选中的站，则可以在右侧添加，否则不添加
            				//$("#gridCell_addTask_upgrade_ups_common").datagrid("appendRow",row);
            				$("#gridCell_addTask_upgrade_ups_common").datagrid("setRow",{ups_code:row["ups_code"],usable:true});
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
    			var rowIndex = $("#selectedCell_addUpgradTask_ups").datagrid("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_addUpgradTask_ups").datagrid("deleteRow",rowIndex);
    		}
    	}
    	
		var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		//将复选框取消选中
		$("#selectedCell_addUpgradTask_ups").datagrid("uncheckAll");
	}
}
<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedCell_addUpgradTask_common_ups() {
	var selCell = $("#gridCell_addTask_upgrade_ups_common").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_addUpgradTask_ups");
	var rows = dl.datagrid("getRows");


	for (var selCellCount = 0; selCellCount < selCellLength; selCellCount++) {
		var i = 0;
		var cellName = selCell[selCellCount]["ups_code"];
		for (; i < rows.length; i++) {
			//如果当前已经选择了该小站，跳出循环
    		if (rows[i]["value"] == cellName) {
    			break;
    		}
    	}
		//i == rows.length表示当前没有选择该小站，需要加到右侧列表中
		if (i == rows.length) {
			var row = {value: selCell[selCellCount]["ups_code"], text: selCell[selCellCount]["serial_number"] + "(" + selCell[selCellCount]["host_name"] + ")",groupId:choosedGroupId};
   			dl.datagrid("appendRow", row); 
   			var selectedCellObj = new Object();
   			selectedCellObj.ups_code = selCell[selCellCount]["ups_code"];
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
			var rowIndex = $("#gridCell_addTask_upgrade_ups_common").datagrid("getRowIndex", newSelectdArr[cellCount]["ups_code"]);
			//$("#gridCell_addTask_upgrade_ups_common").datagrid("deleteRow",rowIndex);
			$("#gridCell_addTask_upgrade_ups_common").datagrid("setRow",{ups_code:newSelectdArr[cellCount]["ups_code"],usable:false});
		}
	}
	
	$("#gridCell_addTask_upgrade_ups_common").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveUpgradeTask_ups_common(ele) {
	if (!validateStep2()) {
		return;
	}
	var params = {};
	params["timeZone"] = timeZone;
	params["productValue"] = "1";
	params["taskType"] = "1";
	params["taskName"] = $("#taskName_addUpgradeTask_ups_common").val().trim();
	
	// 取得已选择的CPE
	var cellCodes = "";
	var selCells = $("#selectedCell_addUpgradTask_ups").datagrid("getRows");
	if (selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		showMsg('prompt_msg',"<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}

	for (var i = 0; i < selCells.length; i++) {
		cellCodes += selCells[i]["value"] + ",";
	}
	params["upsCodes"] = cellCodes;
	// 取得任务状态
	var radios = document.getElementsByName("taskStatus");
	for (var i = 0; i < radios.length; i++) {
		if (radios[i].checked == true) {
			params["status"] = $(radios[i]).attr("status");
		}
	}
	// 如果是定时执行的，取得时间
	if (params["status"] == "timing") {
		params["time"] = $("#exeTime_addUpgradeTask_ups_common").datetimebox("getValue");
	}
	// 取得选择的文件
	var selFile = $("#gridFiles_addUpgradeFiles_common_ups").datagrid("getSelected");
	params["file_id"] = selFile["id"];
    params["file_name"] = selFile["file_name"];
    //params["task_type"] = "ODU";
	params["rawMode"] = $("#upgradeRawModeFlagupsCommon").val();
	//已经点击了确认按钮，禁用“确认按钮”，以防止服务器响应慢而导致重复点击按钮
	//$(ele).linkbutton('disable');
	
	$.post("${ctx}/task/upgrade/ups/addTask.action", params, function(data) {
		if (data["success"]) {
			showMsg('success_msg','<%=rb.getString("ChengGong")%>');
			$("#winAddUpgradTask").slideUp();
			toViewUPS();//调整到任务列表
		} else {
			$(ele).linkbutton('enable');
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

// 取消新建升级任务
function cancelCreateUpgradeTask_ups_common() {
	$("#winAddUpgradTask").slideUp();
	/* closeDefaultWindow(); */
}

//加载前事件-CPE列表
function beforeLoad_gridCell_addTask_upgrade_cpe_common(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $(".inputslist li.serial_number>input").val();
	if(search_text != ""){
		param["searchText"] = search_text;
	}
}
function radioMatterCPE(value, rowData, rowIndex){
   	value = '<input type="radio" name="productType"/>';
    return value;
}
function onSelect_softwareFile_cpe(param){
	if($("#gridFiles_addUpgradeFiles_common_ups").datagrid("getSelected")){
		$($("input[name=productType]")[param]).prop("checked",true);
	}
}
</script>