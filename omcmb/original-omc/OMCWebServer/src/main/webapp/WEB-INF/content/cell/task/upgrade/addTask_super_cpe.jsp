<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<%-- 新建升级任务 --%>
<div id="addUpgradeTaskSteps_cpe_super" class="easyui-panel" data-options="border:false,fit:true">
	<%-- 第一步 --%>
	<div id="step_1" class="easyui-layout" data-options="border:false,fit:true">
		<div region="north" data-options="border:false,height:66" style="padding:20px;">
			<%-- 任务名称 --%>
			<label style="width: 72px; display: inline-block"><%=rb.getString("RenWuMingCheng")%></label>
			<input id="taskName_addUpgradeTask_cpe_super" type="text" class="border border-box" style="width:600px;margin-left:6px;"
				maxlength=100 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
		</div>
		<div region="center" data-options="border:false" style="padding:0 20px 20px;">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="west" data-options="width:310,collapsible:false" title="<%=rb.getString("QingXuanZeYunYingShang")%>" style="padding-top:20px;">
					<table class="easyui-datagrid" id="tableOperatorList_upgrade_cpe_super"
				           data-options="singleSelect:false,fit:true,fitColumns:true,border:false,rownumbers:true,pagePosition:'bottom',
				            url: '${ctx}/system/operator/getOperatorListByPage.action',pagination:true,idField:'operator_code',pageSize: 100,pageList: [100],
				            onBeforeLoad: tableUpgradeBeforeLoadOperatorList,onSelect: querySelectedOperatorStation,onUnselect: querySelectedOperatorStation,
				             onSelectAll: querySelectedOperatorStation,onUnselectAll: querySelectedOperatorStation,onLoadSuccess: upgradeOperatorLoadSucc,toolbar:'#toolbar_OperatorList_addTask_upgrade_cpe_super'">
						<thead>
							<tr>
								<th data-options="field:'ck',checkbox:true"></th>
								<th data-options="field:'operator_code',hidden:true"></th>
								<th data-options="field:'operator_name',width:100,sortable:false"><%=rb.getString("YunYingShangMingCheng")%></th>
							</tr>
						</thead>
					</table>
				</div>
				<div region="center" data-options="border:false,onResize:setArrowMargin">
				</div>
				<div region="east" data-options="border:false" style="width:802px;">
					<div class="easyui-layout" data-options="border:false,fit:true">
						<div class="special" region="west" data-options="width:370,border:true,collapsible:false" title="<%=rb.getString("CPELieBiao")%>" style="padding-top:20px;">
							<table id="gridCell_addTask_upgrade_cpe_super"></table>
						</div>
						<div region="center" data-options="border:false,onResize:setArrowMargin">
							<a id="arrow_right_addUpgradeTask" href="javascript:void(0);" class="arrow_right" onclick="addSelectedCell_addUpgradTask_cpe_super()"></a>
							<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_addUpgradTask_cpe_super()"></a>
						</div>
						<div class="special2" region="east" data-options="width:370,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeCPE")%>'" style="padding-top:20px;">
							<%-- 已选择CPE站 --%>
					    	<table class="easyui-datagrid" id="selectedCell_addUpgradTask_cpe_super"
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
			</div>
		</div>
		<div region="south" data-options="border:false,height:69" style="padding: 10px 20px 20px;">
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="stepDown_addUpgradeTask_cpe_super()" style="float:right;"><span><%=rb.getString("XiaYiBu")%></span></a>
		</div>
	</div>
	<%-- 第二步 --%>
	<div id="step_2" class="easyui-panel" data-options="border:false,fit:true">
		<div class="easyui-layout" data-options="border:false,fit:true">
			<div region="center" data-options="border:false">
				<div class="easyui-layout" data-options="border:false,fit:true">
					<div region="center" data-options="border:false" style="padding:20px;" class="padT20">
						<table class="easyui-datagrid" id="gridFiles_addUpgradeFiles_cpe_super" title="<%=rb.getString("XuanZeShengJiWenJian")%>"
							data-options="singleSelect:true,rownumbers:true,pagination:true,pagePosition:'bottom',onLoadSuccess:datagridLoadSuccess,
								url: '${ctx}/cell/version/queryfileInfosList.action?file_type=3',queryParams:{timeZone:timeZone},fitColumns:false,fit:true,
								onSelect : onSelect_softwareFile_super_cpe ,
								border:true,striped:true">
				            <thead>
				            <tr>
				                <th data-options="field:'id',hidden:true"><%=rb.getString("WenJianBiaoShi")%></th>
				                <th data-options="field:'operate',formatter: radioMatterSupperCPE" width="50" align='center'></th>
		                		<th data-options="field:'file_name',formatter:gridCellTooltipFormatter" width="260"><%=rb.getString("WenJianMing")%></th>
				               <%--  <th data-options="field:'manufacturer'" width="100"><%=rb.getString("ZhiZaoShang")%></th> --%>
				                <th data-options="field:'product'" width="78"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></th>
				                <th data-options="field:'size'" width="100"><%=rb.getString("WenJianDaXiao")%></th>
								<th data-options="field:'version'" width="80"><%=rb.getString("BanBen")%></th>
				                <th data-options="field:'upload_time'" width="140"><%=rb.getString("ShangChuanShiJian")%></th>
				                <th data-options="field:'uploader'" width="80"><%=rb.getString("ShangChuanZhe")%></th>
						        <th data-options="field:'desc'" width="260"><%=rb.getString("MiaoShu")%></th>
						        <%-- <th data-options="field:'md5_error'" width="100"><%=rb.getString("WenJianJiaoYanCuoWu")%></th> --%>
				            </tr>
				            </thead>
				        </table>
					</div>
					<div region="south" style="padding: 0 20px 20px 20px;" data-options="border:false,height:200">
						<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding:20px;">
							<div class="easyui-layout" data-options="fit:true,border:true">
								<div region="north" data-options="border:false,height:60">
									<div style="line-height:55px;" class="verM">
										<input type="radio" id="active_addUpgradeTask_cpe_super" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable()" />
										<label for="active_addUpgradeTask_cpe_super" style="display:inline-block;width:550px;" ><%=rb.getString("LiJiZhiXing")%></label>
										<input type="radio" id="suspend_addUpgradeTask_cpe_super" status="suspend"  name="taskStatus" onchange="setExeTimerEnable()"/>
										<label for="suspend_addUpgradeTask_cpe_super"><%=rb.getString("GuaQi")%></label>
									</div>
								</div>
								<div region="center" data-options="border:false">
									<div style="margin-top: 10px;" class="dateRe">
										<input type="radio" id="timing_addUpgradeTask_cpe_super" status="timing" name="taskStatus" onchange="setExeTimerEnable()"/>
										<label for="timing_addUpgradeTask_cpe_super" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
										<input id="exeTime_addUpgradeTask_cpe_super" class="easyui-datetimebox border-box border" style="height:26px;"
											data-options="disabled:true,editable:false">
																				
										<label style="" hidden="true"><%=rb.getString("BaoLiuPeiZhi")%></label>
										<select id="upgradeRawModeFlagCpeSuper" class="border border-box" style="margin-left:15px" hidden="true">
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
			<div region="south" data-options="border:false,height:69" style="padding: 10px 20px 20px;">
				<div class="windowButtonGroup">				
					<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUP_addUpgradeTask_cpe_super()"><span><%=rb.getString("ShangYiBu")%></span></a>
					<a class="easyui-linkbutton linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveUpgradeTask_cpe_super(this)"><span><%=rb.getString("WanCheng")%></span></a>
					<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelCreateUpgradeTask_cpe_super()"><span><%=rb.getString("QuXiao")%></span></a>
				</div>
			</div>
		</div>
	</div>
</div>

<div id="toolbar_OperatorList_addTask_upgrade_cpe_super" class="admin_query_head" style="background-color:white;">
	<div class="queryGroup" style="margin-left:20px;">	
	    <ul class="inputslist">
	        <li class="operator_code input_li">
		        <input id="upgradeOperatorCode_cpe_super" style="width:225px;" placeholder="<%=rb.getString("YunYingShangMingCheng")%>"/>
	        </li>
	        <li>
				<b onclick="$('#tableOperatorList_upgrade_cpe_super').datagrid('reload');"></b>	
	        </li>
	    </ul>
	</div>
</div>

<%-- 工具栏-CPE列表 --%>
<div id="toolbar_GridCell_addTask_upgrade" class="admin_query_head" style="background-color:white;">
	<div class="queryGroup" style="margin-left:10px;">	
		<ul class="inputslist defaultQuery">
		 	<li class="select_reset">
				<select id = "upgradeDeviceGroup" name="type" class="easyui-combobox border border-box" data-options="editable:false" style="height:26px;width:100px;margin-right:1px;height: 26px;"></select>
			</li>
			<li class="serial_number input_li">
				<input id="serarch_text" name="value" style="width:165px;margin-left:10px;" placeholder="<%=rb.getString("CpeBianMaHUOMINGCHENG")%>"/>
			</li>
			<li>
				<b onclick="$('#gridCell_addTask_upgrade_cpe_super').datagrid('reload');"></b>	
			</li>
		</ul>
	</div>
</div>

<script type="text/javascript">

var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在cpe列表加载前
var operator_codes = "";

$(function() {
    var ele = $("#exeTime_addUpgradeTask_cpe_super");
	disableSelectEarlyTime(ele);  
	closeLoading();
	
	$("#taskName_addUpgradeTask_cpe_super").val("${addTaskName}");
	
	$("#step_2").hide();
	
	// 声明cpe列表
	$("#gridCell_addTask_upgrade_cpe_super").datagrid({
		border: false,
		fitColumns: false,
		fit: true,
        rownumbers: true,
        pageSize: 100,
        pageList: [100],
        striped: true,
        singleSelect: false,
        pagination: true,
        pagePosition: 'bottom',
        idField: 'small_cell_code',
        onBeforeLoad: beforeLoad_gridCell_addTask_upgrade_cpe_super,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_GridCell_addTask_upgrade',
        onCheckAll:checkAll_upgrade_cpe_super,
        onBeforeCheck:beforeCheck_upgrade_cpe_super,
        onLoadSuccess: loadSuccess_addTask_upgrade,
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'serial_number', sortable: true, width: 200, title: '<%=rb.getString("CPEBianMa")%>'},
			{field: 'host_name', sortable: true, width: 200, title: '<%=rb.getString("CpeName")%>'}
		]]
	});
	
	initCellGrid_addUpgradeTask();
	
	$("#upgradeDeviceGroup").combobox({
	        width: 100,
	        panelWidth: 150,
	        panelHeight: 200,
	        editable:false,
	        valueField: 'id',
	        textField: 'group_name',
	        onSelect: chooseUpgradeDeviceGroup,
	        onBeforeLoad: upgradeDeviceGroupBeforeLoad
	 });
});
//保存当前已选择的cpe对象数组
var delCellCodeObjArr = new Array();

$("#upgradeDeviceGroup").combobox('setValues',['<%=rb.getString("SheBeiZu")%>']); 
function checkAll_upgrade_cpe_super(rows){
	$(this).datagrid("initRow").datagrid("getPanel").find(".datagrid-htable .datagrid-header-check>input").prop("checked",true);
}
function beforeCheck_upgrade_cpe_super(index,row){
	var bool = $(this).datagrid("checkUnable",row);
	if(bool){
		return !bool;
	}
}
function chooseUpgradeDeviceGroup(data){
	choosedGroupId = data.id;
    $("#gridCell_addTask_upgrade_cpe_super").datagrid("reload");
}
var opers = "";
function tableUpgradeBeforeLoadOperatorList(param){
	$("#tableOperatorList_upgrade_cpe_super").datagrid("getSelections");
	var searchOperatorCode = $("#upgradeOperatorCode_cpe_super").val();
	if (searchOperatorCode) {
		param["operator_code"] = searchOperatorCode;
	}
	
	//清空所有的已选运营商
	/* var sels = $("#tableOperatorList_upgrade_cpe_super").datagrid("getSelections");
	if(sels.length>0 && typeof(sels[0]) != "undefined"){
		$("#tableOperatorList_upgrade_cpe_super").datagrid("unselectAll");
	} */
	
	$("#tableOperatorList_upgrade_cpe_super").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
}
function upgradeDeviceGroupBeforeLoad(param){
	param["operator_codes"] = operator_codes;
}
function loadSuc_operatorList_upgrade(){
	$("#tableOperatorList_upgrade_cpe_super").datagrid("selectRecord", "default");
}
function upgradeOperatorLoadSucc(){
	$(this).datagrid("enableContextmenuAutoSize");
	//var sels = $("#tableOperatorList_upgrade_cpe_super").datagrid("getRows");
	if(opers == ""){
		$("#tableOperatorList_upgrade_cpe_super").datagrid("selectRow",0);
	}
}
function querySelectedOperatorStation(index,row){
	operator_codes = "";
	choosedGroupId = -1;
	var selCells = $("#tableOperatorList_upgrade_cpe_super").datagrid("getSelections");
	if(selCells.length > 0){
		for(var i=0;i<selCells.length;i++){
			if("undefined" == typeof(selCells[0])){
				return;//当未查询出运营商时，直接结果当前方法
			}
			operator_codes = operator_codes + selCells[i]["operator_code"]+";";
		}
		if(";" == operator_codes.charAt(operator_codes.length - 1)){
			operator_codes = operator_codes.substring(0,operator_codes.length-1);
		}
	}else{
		operator_codes = "-1";
	}
	$("#upgradeDeviceGroup").combobox({
		url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action'
	});
	$("#gridCell_addTask_upgrade_cpe_super").datagrid({
		url: '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=1'
	});
	$("#upgradeDeviceGroup").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']); 
	
}
<%-- cpe列表加载完成事件--%>
function loadSuccess_addTask_upgrade() {
	$(this).datagrid("initRow");
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	
	//初始添加-行操作提示记录
    $("#selectedCell_addUpgradTask_cpe_super").datagrid({
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
	//去掉已经选中的cpe
	if (delCellCodeObjArr.length > 0) {
	 	for(var cellCount = 0; cellCount < delCellCodeObjArr.length; cellCount++) {
			var rowIndex = $("#gridCell_addTask_upgrade").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["small_cell_code"]);
			if (rowIndex != -1) {
				//$("#gridCell_addTask_upgrade").datagrid("deleteRow",rowIndex);
				$("#gridCell_addTask_upgrade_cpe_super").datagrid("setRow",{small_cell_code:delCellCodeObjArr[cellCount]["small_cell_code"],usable:false});
			}
		}
		//向右侧列表添加已经选中的cpe
		for (var i = 0; i < delCellCodeObjArr.length; i++) {
			var row = {value: delCellCodeObjArr[i]["small_cell_code"], text: delCellCodeObjArr[i]["serial_number"] + "(" + delCellCodeObjArr[i]["host_name"] + ")"};
			$("#selectedCell_addUpgradTask_cpe_super").datagrid("appendRow", row);
		}
		 
		var rows = $("#selectedCell_addUpgradTask_cpe_super").datagrid("getRows");
		if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
			$("#selectedCell_addUpgradTask_cpe_super").datagrid("deleteRow", 0);
		}
	}
}
<%-- 初始化cpe列表--%>
function initCellGrid_addUpgradeTask() {
	<%-- cpe搜索-回车事件 --%>
    $("#upgradeOperatorCode_cpe_super").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#tableOperatorList_upgrade_cpe_super").datagrid("reload");
        }
    });
    
    <%-- cpe搜索-软件版本-下拉面板 --%>
	<%-- $("#softwareVersionCombo_addTask_upgrade").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	});
	
	cpe搜索-类型-change事件
	$("#toolbar_GridCell_addTask_upgrade select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_addTask_upgrade .input_li").hide();
		$("#toolbar_GridCell_addTask_upgrade ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_addTask_upgrade").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	}); --%>
	
	<%-- cpe搜索-地域树-下拉面板 --%>
	/* $("#regnTreeCombo_addTask_upgrade").combotree({
		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
		panelWidth: 200
	});
	
	$(".group_id input.textbox-text").css("padding", "0 10px").css("margin", "0"); */
}
<%-- 下一步  --%>
function stepDown_addUpgradeTask_cpe_super() {
	if (validateStep1()) {
		$("#addUpgradeTaskSteps_cpe_super #step_1").hide();
		$("#addUpgradeTaskSteps_cpe_super #step_2").show();
		$('#addUpgradeTaskSteps_cpe_super #step_2').panel("doLayout");		
	}
}
<%-- 上一步 --%>
function stepUP_addUpgradeTask_cpe_super() {
	$("#addUpgradeTaskSteps_cpe_super #step_2").hide();
	$("#addUpgradeTaskSteps_cpe_super #step_1").show();
}

<%-- 验证第一步输入 --%>
function validateStep1() {
	// 验证任务名称是否填写
	if ($("#taskName_addUpgradeTask_cpe_super").val().trim().length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>");
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/upgrade/cpe/taskNameExist.action", 
			data: {"taskName": $("#taskName_addUpgradeTask_cpe_super").val().trim(),"taskType":"1"},
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

	// 是否已选择CPE
	var selCells = $("#selectedCell_addUpgradTask_cpe_super").datagrid("getRows");
	if (selCells.length == 0 || selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	// 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	return true;
}

<%-- 验证第二步输入 --%>
function validateStep2() {
	var selFile = $("#gridFiles_addUpgradeFiles_cpe_super").datagrid("getSelected");
	if (selFile) {
		return true;
	} else {
		$.messager.alert(TiShi, "<%=rb.getString("QingXianXuanZeWenJian")%>");
		return false;
	}
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable() {
	if (document.getElementById("timing_addUpgradeTask_cpe_super").checked) {
		$("#exeTime_addUpgradeTask_cpe_super").datetimebox("enable");
	} else {
		$("#exeTime_addUpgradeTask_cpe_super").datetimebox("disable");
	}
}

<%-- 删除已选中的cpe-向左键头的点击事件 --%>
function delSelectedCell_addUpgradTask_cpe_super() {
	var dl = $("#selectedCell_addUpgradTask_cpe_super");
	var selCell = dl.datagrid("getSelections");
	var selCellLength = selCell.length;
	var needDelCellArr = new Array(); //需要删除的cpe编码数组
	
	if (selCellLength > 0) {
		for (var selCellCount = 0; selCellCount < selCell.length; selCellCount++) {
			if (selCell[selCellCount]["value"]) {
    			//从cpe列表中恢复cpe
        		for(var cellCount = 0; cellCount < delCellCodeObjArr.length; cellCount++) {
        			if (selCell[selCellCount].value == delCellCodeObjArr[cellCount]["small_cell_code"]) {
            			var row = {
            				small_cell_code: delCellCodeObjArr[cellCount]["small_cell_code"], 
            				serial_number: delCellCodeObjArr[cellCount]["serial_number"],
            				host_name: delCellCodeObjArr[cellCount]["host_name"],
            				connection_status: delCellCodeObjArr[cellCount]["connection_status"]
            			}
            			 if(choosedGroupId == delCellCodeObjArr[cellCount]["groupId"]){//如果是当前组被选中的cpe，则可以在右侧添加，否则不添加
            				//$("#gridCell_addTask_upgrade").datagrid("appendRow",row);
            				 $("#gridCell_addTask_upgrade_cpe_super").datagrid("setRow",{small_cell_code:row["small_cell_code"],usable:true});
            			}
            			delCellCodeObjArr.splice(cellCount,1);
            			needDelCellArr.push(selCell[selCellCount].value);
            			break;
        			}
        		}
    		}
		}
		
    	if (needDelCellArr.length > 0) {
    		//从已选cpe列表中删除cpe
    		for(var cellCount = 0; cellCount < needDelCellArr.length; cellCount++) {
    			var rowIndex = $("#selectedCell_addUpgradTask_cpe_super").datagrid("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_addUpgradTask_cpe_super").datagrid("deleteRow",rowIndex);
    		}
    	}
    	
		var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		//将复选框取消选中
		$("#selectedCell_addUpgradTask_cpe_super").datagrid("uncheckAll");
	}
}
<%-- 选择CPE-向右键头的点击事件 --%>
function addSelectedCell_addUpgradTask_cpe_super() {
	var selCell = $("#gridCell_addTask_upgrade_cpe_super").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_addUpgradTask_cpe_super");
	var rows = dl.datagrid("getRows");


	for (var selCellCount = 0; selCellCount < selCellLength; selCellCount++) {
		var i = 0;
		var cellName = selCell[selCellCount]["small_cell_code"];
		for (; i < rows.length; i++) {
			//如果当前已经选择了该CPE，跳出循环
    		if (rows[i]["value"] == cellName) {
    			break;
    		}
    	}
		//i == rows.length表示当前没有选择该CPE，需要加到右侧列表中
		if (i == rows.length) {
			var row = {value: selCell[selCellCount]["small_cell_code"], text: selCell[selCellCount]["serial_number"] + "(" + selCell[selCellCount]["host_name"] + ")",groupId:choosedGroupId};
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
		//从cpe列表中删除已选择的cpe
		for(var cellCount = 0; cellCount < newSelectdArr.length; cellCount++) {
			var rowIndex = $("#gridCell_addTask_upgrade_cpe_super").datagrid("getRowIndex", newSelectdArr[cellCount]["small_cell_code"]);
			//$("#gridCell_addTask_upgrade_cpe_super").datagrid("deleteRow",rowIndex);
			$("#gridCell_addTask_upgrade_cpe_super").datagrid("setRow",{small_cell_code:newSelectdArr[cellCount]["small_cell_code"],usable:false});
		}
	}
	
	$("#gridCell_addTask_upgrade_cpe_super").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveUpgradeTask_cpe_super(ele) {
	if (!validateStep2()) {
		return;
	}
	var params = {};
	params.timeZone = timeZone;
	params["taskName"] = $("#taskName_addUpgradeTask_cpe_super").val().trim();
	params["productValue"] = "1";
	params["taskType"] = "1";
	
	// 取得已选择的cpe
	var cellCodes = "";
	var selCells = $("#selectedCell_addUpgradTask_cpe_super").datagrid("getRows");
	if (selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}

	for (var i = 0; i < selCells.length; i++) {
		cellCodes += selCells[i]["value"] + ",";
	}
	params["cellCodes"] = cellCodes;
	// 取得任务状态
	var radios = document.getElementsByName("taskStatus");
	for (var i = 0; i < radios.length; i++) {
		if (radios[i].checked == true) {
			params["status"] = $(radios[i]).attr("status");
		}
	}
	// 如果是定时执行的，取得时间
	if (params["status"] == "timing") {
		params["time"] = $("#exeTime_addUpgradeTask_cpe_super").datetimebox("getValue");
	}
	// 取得选择的文件
	var selFile = $("#gridFiles_addUpgradeFiles_cpe_super").datagrid("getSelected");
	params["file_id"] = selFile["id"];
    params["file_name"] = selFile["file_name"];
   // params["task_type"] = "ODU";
	params["rawMode"] = $("#upgradeRawModeFlagCpeSuper").val();
	//已经点击了确认按钮，禁用“确认按钮”，以防止服务器响应慢而导致重复点击按钮
	$(ele).linkbutton('disable');
	
	$.post("${ctx}/task/upgrade/cpe/addTask.action", params, function(data) {
		if (data["success"]) {
			/* $("#winAddUpgradTask").window("close"); */
			closeDefaultWindow();
			$("#upgradTaskList").datagrid("load");
		} else {
			$(ele).linkbutton('enable');
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

// 取消新建升级任务
function cancelCreateUpgradeTask_cpe_super() {
	/* $("#winAddUpgradTask").window("close"); */
	closeDefaultWindow();
}

//加载前事件-cpe列表
function beforeLoad_gridCell_addTask_upgrade_cpe_super(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $("#serarch_text").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	if(operator_codes != ""){
		param["operator_codes"] = operator_codes;
	}else{
		param["operator_codes"] = "-1";//用于标识，为-1，则表明未选中任务运营商
	}
	$("#gridCell_addTask_upgrade_cpe_super").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
}
function radioMatterSupperCPE(value, rowData, rowIndex){
   	value = '<input type="radio" name="productType"/>';
    return value;
}
function onSelect_softwareFile_super_cpe(param){
	if($("#gridFiles_addUpgradeFiles_cpe_super").datagrid("getSelected")){
		$($("input[name=productType]")[param]).prop("checked",true);
	}
}
</script>