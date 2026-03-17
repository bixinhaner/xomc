<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<div id="toolbar_OperatorList_addTask_upgradeCa_super" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery">
		<li class="operator_code input_li" style="margin-left:27px;">
			<!-- <input id="upgradeCaOperatorCode_super" style="width:168px;height:26px;float:left;" class="border border-box"> -->
			<input id="upgradeCaOperatorCode_super"  style="width:220px;margin-left:0px;" placeholder="<%=rb.getString("YunYingShangMingCheng")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		</li>
		<li>
			<b class="searchResultImgChangeStyle" onclick="$('#tableOperatorList_upgradeCa_super').datagrid('reload');" ></b>	
		</li>
	</ul>
</div>
<%-- 新建CA证书升级任务 --%>
<div id="addUpgradeCaTaskSteps_super" class="easyui-panel" data-options="border:false,fit:true">
	<%-- 第一步 --%>
	<div id="step_1" class="easyui-layout" data-options="border:false,fit:true">
		<div region="north" data-options="border:false,height:66" style="padding:20px;">
			<%-- 任务名称 --%>
			<label style="display: inline-block;width:72px;"><%=rb.getString("RenWuMingCheng")%></label>
			<input id="taskName_addUpgradeCaTask_super" type="text" class="border border-box" style="width:617px;margin-left:6px;"
				maxlength=100 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
		</div>
		<div region="center" data-options="border:false" style="padding: 0 20px 20px;">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="west" data-options="width:310,collapsible:false" title="<%=rb.getString("QingXuanZeYunYingShang")%>" style="padding-top:20px;">
					 <table class="easyui-datagrid" id="tableOperatorList_upgradeCa_super"
				            data-options="singleSelect:false,fit:true,fitColumns:true,border:false,rownumbers:true,pagePosition:'bottom',
				            url: '${ctx}/system/operator/getOperatorListByPage.action',pagination:true,idField:'operator_code',pageSize: 100,pageList: [100],
				            onBeforeLoad: tableUpgradeCaBeforeLoadOperatorList,onSelect: queryUpgradeCaSelectedOperatorStation,onUnselect: queryUpgradeCaSelectedOperatorStation,
				            onSelectAll: queryUpgradeCaSelectedOperatorStation,onUnselectAll: queryUpgradeCaSelectedOperatorStation,onLoadSuccess: upgradeCaOperatorLoadSuc,
				            toolbar:'#toolbar_OperatorList_addTask_upgradeCa_super'">
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
						<div class="special" region="west" data-options="width:370,border:true,collapsible:false" title="<%=rb.getString("JiZhanLieBiao")%>" style="padding-top:20px;">
							<table id="gridCell_addTask_upgradeCa_super"></table>
						</div>
						<div region="center" data-options="border:false,onResize:setArrowMargin">
							<a id="arrow_right_addUpgradeCaTask_super" href="javascript:void(0);" class="arrow_right" onclick="addSelectedCell_addUpgradeCaTask()"></a>
							<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_addUpgradeCaTask()"></a>
						</div>
						<div class="special2" region="east" data-options="width:370,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeJiZhan")%>'" style="padding-top:20px;">
							<%-- 已选择基站 --%>
					    	<table class="easyui-datagrid" id="selectedCell_addUpgradeCaTask_super"
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
					</div>					
				</div>
			</div>
		</div>
		<div region="south" data-options="border:false,height:71" style="padding: 10px 20px 20px;">
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="stepDown_addUpgradeCaTask()" style="float: right;"><span><%=rb.getString("XiaYiBu")%></span></a>
		</div>
	</div>
	
	<%-- 第二步 --%>
	<div id="step_2" class="easyui-panel" data-options="border:false,fit:true">
		<div class="easyui-layout" data-options="border:false,fit:true">
			<div region="center" data-options="border:false" style="padding:20px;">
				<div class="easyui-layout" data-options="border:false,fit:true">
					<div region="center" data-options="border:false" class="padT20">
						<table class="easyui-datagrid" id="gridFiles_addUpgradeCaFiles_super" title="<%=rb.getString("XuanZeShengJiWenJian")%>"
							data-options="singleSelect:true,rownumbers:true,pagination:true,pagePosition:'bottom',onBeforeLoad: beforeLoadAddUpgradeCaFiles,onLoadSuccess:datagridLoadSuccess,
								url: '${ctx}/cell/version/queryfileInfosList.action?file_type=1',queryParams:{timeZone:timeZone,productValue:'${productValue}'},fitColumns:true,fit:true,
								onSelect : onSelect_softwareFile_Ca ,
								border:true,striped:true">
				            <thead>
				            <tr>
				                <th data-options="field:'id',hidden:true"><%=rb.getString("WenJianBiaoShi")%></th>
				                <th data-options="field:'operate',formatter: radioMatterCa" width="50" align='center' halign='center'></th>
		                		<th data-options="field:'file_name'" width="260"><%=rb.getString("WenJianMing")%></th>
				                <%-- <th data-options="field:'manufacturer'" width="100"><%=rb.getString("ZhiZaoShang")%></th> --%>
				                <th data-options="field:'product'" width="78"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></th>
				                <th data-options="field:'size'" width="100"><%=rb.getString("WenJianDaXiao")%></th>
								<th data-options="field:'version'" width="80"><%=rb.getString("BanBen")%></th>
				                <th data-options="field:'upload_time'" width="140"><%=rb.getString("ShangChuanShiJian")%></th>
				                <th data-options="field:'uploader'" width="80"><%=rb.getString("ShangChuanZhe")%></th>
						        <th data-options="field:'desc'" width="260"><%=rb.getString("MiaoShu")%></th>
						       <%--  <th data-options="field:'md5_error'" width="100"><%=rb.getString("WenJianJiaoYanCuoWu")%></th> --%>
				            </tr>
				            </thead>
				        </table>
					</div>
					<div region = "south" style="margin-top:20px;" data-options="border:false,height:200">
						<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding: 20px;">
								<div class="easyui-layout" data-options="fit:true,border:true">
									<div region="north" data-options="border:false,height:60">
										<div class="verM"  style="line-height:55px;">
											<input type="radio" id="active_addUpgradeCaTask_super" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable()"/>
											<label for="active_addUpgradeCaTask_super" style="width:250px;"><%=rb.getString("LiJiZhiXing")%></label>
											<input type="radio" id="suspend_addUpgradeCaTask_super" status="suspend" style="margin-left: 465px;" name="taskStatus" onchange="setExeTimerEnable()"/>
											<label for="suspend_addUpgradeCaTask_super"><%=rb.getString("GuaQi")%></label>
										</div>
									</div>
									<div region="center" data-options="border:false">
										<div  class="dateRe" style="line-height:55px;">
											<input type="radio" id="timing_addUpgradeCaTask_super" status="timing" name="taskStatus" onchange="setExeTimerEnable()"/>
											<label for="timing_addUpgradeCaTask_super" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
											<input id="exeTime_addUpgradeCaTask_super" class="easyui-datetimebox border-box border" style="height:26px;"
												data-options="disabled:true,editable:false">
										</div>
									</div>
								</div>
							</div>
					</div>
				</div>
			</div>
			<div region="south" data-options="border:false,height:71" style="padding: 10px 20px 20px;">
				<div class="windowButtonGroup">
					<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUP_addUpgradeCaTask()"><span><%=rb.getString("ShangYiBu")%></span></a>
					<a class="easyui-linkbutton linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveUpgradeCaTask(this)"><span><%=rb.getString("WanCheng")%></span></a>
					<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelCreateUpgradeCaTask()"><span><%=rb.getString("QuXiao")%></span></a>
				</div>
			</div>
		</div>
	</div>
</div>


<%-- 工具栏-基站列表 --%>
<div id="toolbar_GridCell_addTask_upgradeCa_super" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery">
	 	<li class="select_reset" >
			<select id = "caUpgradeDeviceGroup_super" name="type" class="border border-box easyui-combobox" data-options="editable:false" style="width:103px;margin-right:1px;height: 26px;"></select>
		</li>
		<li class="serial_number input_li">
			<%-- <input name="value" type="text" class="border border-box" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>"> --%>
			<input name="value" style="width:190px;margin-left:0px;" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		</li>
		<li>
			<b class="searchResultImgChangeStyle" onclick="$('#gridCell_addTask_upgradeCa_super').datagrid('reload');" ></b>	
		</li>
	</ul>
</div>

<script type="text/javascript">

var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在基站列表加载前
var operator_codes = "";

$(function() {
    var ele = $("#exeTime_addUpgradeCaTask_super");
	disableSelectEarlyTime(ele);  
	closeLoading();
	
	$("#taskName_addUpgradeCaTask_super").val("${addTaskName}");
	$("#addUpgradeCaTaskSteps_super step_2").hide();
	
	// 声明基站列表
	$("#gridCell_addTask_upgradeCa_super").datagrid({
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
        onBeforeLoad: beforeLoad_gridCell_addTask_upgradeCa,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_GridCell_addTask_upgradeCa_super',
        onCheckAll:checkAll_upgrade_ca_super,
        onBeforeCheck:beforeCheck_upgrade_ca_super,
        onLoadSuccess: loadSuccess_addTask_upgradeCa,
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'serial_number', sortable: true, width: 200, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name', sortable: true, width: 200, title: '<%=rb.getString("HostName")%>'}
		]]
	});
	
	initCellGrid_addUpgradeCaTask();
	
	$("#caUpgradeDeviceGroup_super").combobox({
        width: 100,
        panelWidth: 150,
        panelHeight: 200,
        valueField: 'id',
        textField: 'group_name',
        onSelect: chooseCaUpgradeDeviceGroup,
        onBeforeLoad: upgradeCaDeviceGroupBeforeLoad
    });
});
//保存当前已选择的基站对象数组
var delCellCodeObjArr = new Array();

$("#caUpgradeDeviceGroup_super").combobox('setValues',['<%=rb.getString("SheBeiZu")%>']);
function checkAll_upgrade_ca_super(rows){
	$(this).datagrid("initRow").datagrid("getPanel").find(".datagrid-htable .datagrid-header-check>input").prop("checked",true);
}
function beforeCheck_upgrade_ca_super(index,row){
	var bool = $(this).datagrid("checkUnable",row);
	if(bool){
		return !bool;
	}
}
function chooseCaUpgradeDeviceGroup(data){
	choosedGroupId = data.id;
    $("#gridCell_addTask_upgradeCa_super").datagrid("reload");
}
var opers = "";
function tableUpgradeCaBeforeLoadOperatorList(param){
	opers = $("#tableOperatorList_upgradeCa_super").datagrid("getSelections");
	var searchOperatorCode = $("#upgradeCaOperatorCode_super").val();
	if (searchOperatorCode) {
		param["operator_code"] = searchOperatorCode;
	}


	// 兼容intel 和 qc
	param["isShowSlave"] = false;
	
	//清空所有的已选运营商
	/* var sels = $("#tableOperatorList_upgradeCa_super").datagrid("getSelections");
	if(sels.length>0 && typeof(sels[0]) != "undefined"){
		$("#tableOperatorList_upgradeCa_super").datagrid("unselectAll");
	}
	 */
	$("#tableOperatorList_upgradeCa_super").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
}
function upgradeCaDeviceGroupBeforeLoad(param){
	param["operator_codes"] = operator_codes;
}
function upgradeCaOperatorLoadSuc(){
	$(this).datagrid("enableContextmenuAutoSize");
	if(opers == ""){
		$("#tableOperatorList_upgradeCa_super").datagrid("selectRow",0);
	}
}
function queryUpgradeCaSelectedOperatorStation(){
	operator_codes = "";
	choosedGroupId = -1;
	var selCells = $("#tableOperatorList_upgradeCa_super").datagrid("getSelections");
	if(selCells.length > 0){
		for(var i=0;i<selCells.length;i++){
			operator_codes = operator_codes + selCells[i]["operator_code"]+";";
		}
		if(";" == operator_codes.charAt(operator_codes.length - 1)){
			operator_codes = operator_codes.substring(0,operator_codes.length-1);
		}
	}else{
		operator_codes = "-1";
	}
	$("#caUpgradeDeviceGroup_super").combobox({
		url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action'
	});
	$("#gridCell_addTask_upgradeCa_super").datagrid({
		 url: '${ctx}/cell/cpeinfos/queryCpeInfosList.action?forSelect=1',
		 queryParams:{like_fields:'serial_number,host_name',productValue:'${productValue}'}
	});
	$("#caUpgradeDeviceGroup_super").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']);
}
<%-- 基站列表加载完成事件--%>
function loadSuccess_addTask_upgradeCa() {
	$(this).datagrid("initRow");
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	
	//初始添加-行操作提示记录
    $("#selectedCell_addUpgradeCaTask_super").datagrid({
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
    		var rowIndex = $("#gridCell_addTask_upgradeCa_super").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["small_cell_code"]);
    		if (rowIndex != -1) {
    			//$("#gridCell_addTask_upgradeCa_super").datagrid("deleteRow",rowIndex);
    			$("#gridCell_addTask_upgradeCa_super").datagrid("setRow",{small_cell_code:delCellCodeObjArr[cellCount]["small_cell_code"],usable:false});
    		}
    	}
    	//向右侧列表添加已经选中的基站
    	for (var i = 0; i < delCellCodeObjArr.length; i++) {
    		var row = {value: delCellCodeObjArr[i]["small_cell_code"], text: delCellCodeObjArr[i]["serial_number"] + "(" + delCellCodeObjArr[i]["host_name"] + ")"};
    		$("#selectedCell_addUpgradeCaTask_super").datagrid("appendRow", row);
    	}
    	 
    	var rows = $("#selectedCell_addUpgradeCaTask_super").datagrid("getRows");
    	if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
    		$("#selectedCell_addUpgradeCaTask_super").datagrid("deleteRow", 0);
    	}
    }
}
<%-- 初始化基站列表--%>
function initCellGrid_addUpgradeCaTask() {
	<%-- 运营商搜索-回车事件 --%>
	$("#upgradeCaOperatorCode_super").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#tableOperatorList_upgradeCa_super").datagrid("reload");
        }
    });
	
	<%-- 基站搜索-回车事件 --%>
    $("#toolbar_GridCell_addTask_upgradeCa_super input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_addTask_upgradeCa_super").datagrid("reload");
        }
    });
    
    <%-- 基站搜索-软件版本-下拉面板 --%>
	/* $("#softwareVersionCombo_addTask_upgradeCa").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	}); */
	
	<%-- 基站搜索-类型-change事件 --%>
	/* $("#toolbar_GridCell_addTask_upgradeCa_super select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_addTask_upgradeCa_super .input_li").hide();
		$("#toolbar_GridCell_addTask_upgradeCa_super ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_addTask_upgradeCa").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	}); */
	
	<%-- 基站搜索-地域树-下拉面板 --%>
	/* $("#regnTreeCombo_addTask_upgradeCa").combotree({
		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
		panelWidth: 200
	});
	
	$(".group_id input.textbox-text").css("padding", "0 10px").css("margin", "0"); */
}
<%-- 下一步  --%>
function stepDown_addUpgradeCaTask() {
	if (validateStep1()) {
		$("#addUpgradeCaTaskSteps_super #step_1").hide();
		$("#addUpgradeCaTaskSteps_super #step_2").show();
		$('#addUpgradeCaTaskSteps_super #step_2').panel("doLayout");		
	}
}
<%-- 上一步 --%>
function stepUP_addUpgradeCaTask() {
	$("#addUpgradeCaTaskSteps_super #step_2").hide();
	$("#addUpgradeCaTaskSteps_super #step_1").show();
}

<%-- 验证第一步输入 --%>
function validateStep1() {
	// 验证任务名称是否填写
	if ($("#taskName_addUpgradeCaTask_super").val().trim().length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>");
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/upgradeCa/taskNameExist.action", 
			data: {"taskName": $("#taskName_addUpgradeCaTask_super").val().trim()},
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
	var selCells = $("#selectedCell_addUpgradeCaTask_super").datagrid("getRows");
	if (selCells.length == 0 || selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	// 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	if (document.getElementById("timing_addUpgradeCaTask_super").checked
			&& $("#exeTime_addUpgradeCaTask_super").datetimebox("getValue").length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeShiJian")%>");
		return false;
	}
	return true;
}

<%-- 验证第二步输入 --%>
function validateStep2() {
	var selFile = $("#gridFiles_addUpgradeCaFiles_super").datagrid("getSelected");
	if (selFile) {
		return true;
	} else {
		$.messager.alert(TiShi, "<%=rb.getString("QingXianXuanZeWenJian")%>");
		return false;
	}
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable() {
	if (document.getElementById("timing_addUpgradeCaTask_super").checked) {
		$("#exeTime_addUpgradeCaTask_super").datetimebox("enable");
	} else {
		$("#exeTime_addUpgradeCaTask_super").datetimebox("disable");
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_addUpgradeCaTask() {
	var dl = $("#selectedCell_addUpgradeCaTask_super");
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
            				//$("#gridCell_addTask_upgradeCa_super").datagrid("appendRow",row);
            				 $("#gridCell_addTask_upgradeCa_super").datagrid("setRow",{small_cell_code:row["small_cell_code"],usable:true});
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
    			var rowIndex = $("#selectedCell_addUpgradeCaTask_super").datagrid("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_addUpgradeCaTask_super").datagrid("deleteRow",rowIndex);
    		}
    	}
    	
		var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		
		//将复选框取消选中
		$("#selectedCell_addUpgradeCaTask_super").datagrid("uncheckAll");
	}
}
<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedCell_addUpgradeCaTask() {
	var selCell = $("#gridCell_addTask_upgradeCa_super").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_addUpgradeCaTask_super");
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
			var rowIndex = $("#gridCell_addTask_upgradeCa_super").datagrid("getRowIndex", newSelectdArr[cellCount]["small_cell_code"]);
			//$("#gridCell_addTask_upgradeCa_super").datagrid("deleteRow",rowIndex);
			$("#gridCell_addTask_upgradeCa_super").datagrid("setRow",{small_cell_code:newSelectdArr[cellCount]["small_cell_code"],usable:false});
		}
	}
	
	$("#gridCell_addTask_upgradeCa_super").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveUpgradeCaTask(ele) {
	if (!validateStep2()) {
		return;
	}
	var params = {};
	params.timeZone=timeZone;
	params["taskName"] = $("#taskName_addUpgradeCaTask_super").val().trim();
	params["taskType"] = "4";
	
	// 取得已选择的基站
	var cellCodes = "";
	var selCells = $("#selectedCell_addUpgradeCaTask_super").datagrid("getRows");
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
		params["time"] = $("#exeTime_addUpgradeCaTask_super").datetimebox("getValue");
	}
	// 取得选择的文件
	var selFile = $("#gridFiles_addUpgradeCaFiles_super").datagrid("getSelected");
	params["fileId"] = selFile["id"];
	params["fileName"] = selFile["file_name"];
	//已经点击了确认按钮，禁用“确认按钮”，以防止服务器响应慢而导致重复点击按钮
    $(ele).linkbutton('disable');
	
	params["productValue"] = "${productValue}";
	$.post("${ctx}/task/upgrade/addTask.action", params, function(data) {
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
function cancelCreateUpgradeCaTask() {
	/* $("#winAddUpgradTask").window("close"); */
	closeDefaultWindow();
}

//加载前事件-基站列表
function beforeLoad_gridCell_addTask_upgradeCa(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $("#toolbar_GridCell_addTask_upgradeCa_super .inputslist li.serial_number>input").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	if(operator_codes != ""){
		param["operator_codes"] = operator_codes;
	}else{
		param["operator_codes"] = "-1";
	}

	// 兼容intel 和 qc
	param["isShowSlave"] = false;
	
	/* var type = $("#toolbar_GridCell_addTask_upgradeCa_super select[name='type']").val();
	var val = $("#toolbar_GridCell_addTask_upgradeCa_super ." + type + " input[name='value']").val();
	param[type] = val; */
	
	$("#gridCell_addTask_upgradeCa_super").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
}
//加载前事件 - 选择升级文件
function beforeLoadAddUpgradeCaFiles(param){

	// 兼容intel 和 qc
	param["isShowSlave"] = false;
}
function radioMatterCa(value, rowData, rowIndex){
   	value = '<input type="radio" name="productType"/>';
    return value;
}
function onSelect_softwareFile_Ca(param){
	if($("#gridFiles_addUpgradeCaFiles_super").datagrid("getSelected")){
		$($("input[name=productType]")[param]).prop("checked",true);
	}
}
</script>