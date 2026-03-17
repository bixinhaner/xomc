<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<div id="toolbar_OperatorList_addTask_upgrade_super" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery" style="margin-top:20px;">
		<li class="operator_code input_li" style="margin-left:27px;">
			<!-- <input id="upgradeOperatorCode_super" style="width:168px;height:26px;float:left;" class="border border-box"> -->
			<input id="upgradeOperatorCode_super"  style="width:220px;margin-left:0px;" placeholder="<%=rb.getString("YunYingShangMingCheng")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		</li>
		<li>
			<b class="searchResultImgChangeStyle" onclick="$('#tableOperatorList_upgrade_super').datagrid('reload');"></b>	
		</li>
	</ul>
</div>
<%-- 新建升级任务 --%>
<div id="addUpgradeTaskSteps_super" class="easyui-panel" data-options="border:false,fit:true">
	<%-- 第一步 --%>
	<div id="step_1_super" class="easyui-layout" data-options="border:false,fit:true">
		<div region="north" data-options="border:false,height:66" style="padding:20px;">
			<%-- 任务名称 --%>
			<label style="display: inline-block;width:72px;"><%=rb.getString("RenWuMingCheng")%></label>
			<input id="taskName_addUpgradeTask_super" type="text" class="border border-box" style="width:613px;margin-left:10px;"
				maxlength=100 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
		</div>
		<div region="center" data-options="border:false" style="padding: 0px 20px 20px;">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="west" data-options="width:300,collapsible:false" title="<%=rb.getString("QingXuanZeYunYingShang")%>">
					<table class="easyui-datagrid" id="tableOperatorList_upgrade_super"
				           data-options="singleSelect:false,fit:true,fitColumns:true,border:false,rownumbers:true,pagePosition:'bottom',
				            url: '${ctx}/system/operator/getOperatorListByPage.action',pagination:true,idField:'operator_code', pageSize: 100,pageList: [100],
				            onBeforeLoad: tableUpgradeBeforeLoadOperatorList,onSelect: querySelectedOperatorStation,onUnselect: querySelectedOperatorStation,
				             onSelectAll: querySelectedOperatorStation,onUnselectAll: querySelectedOperatorStation,onLoadSuccess: upgradeOperatorLoadSucc,toolbar:'#toolbar_OperatorList_addTask_upgrade_super'">
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
				<div region="east" data-options="border:false,onResize:setArrowMargin,width:802">
					<div class="easyui-layout" data-options="border:false,fit:true">
                        <div class="special" region="west" data-options="width:370,border:true,collapsible:false" title="<%=rb.getString("JiZhanLieBiao")%>" style="padding-top:20px;">
                            <table id="gridCell_addTask_upgrade_super"></table>
                        </div>
                        <div region="center" data-options="border:false,onResize:setArrowMargin">
                            <a id="arrow_right_addUpgradeTask" href="javascript:void(0);" class="arrow_right" onclick="addSelectedCell_addUpgradTask()"></a>
                            <a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_addUpgradTask()"></a>
                        </div>
                        <div class="special2" region="east" data-options="width:370,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeJiZhan")%>'" style="padding-top:20px;">
                            <%-- 已选择基站 --%>
                            <table class="easyui-datagrid" id="selectedCell_addUpgradTask_super"
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
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="stepDown_addUpgradeTask()" style="float:right"><span><%=rb.getString("XiaYiBu")%></span></a>
		</div>
	</div>
	<%-- 第二步 --%>
	<div id="step_2_super" class="easyui-panel" data-options="border:false,fit:true">
		<div class="easyui-layout" data-options="border:false,fit:true">
			<div region="center" data-options="border:false" style="padding:20px;">
				<div class="easyui-layout" data-options="border:false,fit:true">
					<div region="center" data-options="border:false" class="padT20">
						<table class="easyui-datagrid" id="gridFiles_addUpgradeFiles" title="<%=rb.getString("XuanZeShengJiWenJian")%>"
							data-options="singleSelect:true,rownumbers:true,pagination:true,pagePosition:'bottom',onBeforeLoad: beforeLoadAddUpgradeFiles ,onLoadSuccess:datagridLoadSuccess,
								url: '${ctx}/cell/version/queryfileInfosList.action?file_type=0',queryParams:{timeZone:timeZone,productValue:'${productValue}'},fitColumns:true,fit:true,
								onSelect : onSelect_softwareFile_supper ,
								border:true,striped:true" style="margin-top:20px;">
				            <thead>
				            <tr>
				                <th data-options="field:'id',hidden:true"><%=rb.getString("WenJianBiaoShi")%></th>
				                <th data-options="field:'operate',formatter: radioMatterSupper" width="50" align='center' halign='center'></th>
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
					<div region="south" style="margin-top:20px;" data-options="border:false,height:200">
						<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding: 20px;">
							<div class="easyui-layout" data-options="fit:true,border:true">
								<div region="north" data-options="border:false,height:60">
									<div style="line-height:55px;">
										<input type="radio" id="active_addUpgradeTask_super" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable()" style="vertical-align:middle;margin-right:6px;"/>
										<label for="active_addUpgradeTask_super" style="display:inline-block;width:530px;vertical-align:middle;"><%=rb.getString("LiJiZhiXing")%></label>
										<input type="radio" id="suspend_addUpgradeTask_super" status="suspend" style="vertical-align:middle;margin-right:6px;" name="taskStatus" onchange="setExeTimerEnable()"/>
										<label for="suspend_addUpgradeTask_super" style="vertical-align:middle;"><%=rb.getString("GuaQi")%></label>
									</div>
								</div>
								<div region="center" data-options="border:false">
									<div class="dateRe" style="line-height:54px;">
										<input type="radio" id="timing_addUpgradeTask_super" status="timing" name="taskStatus" onchange="setExeTimerEnable()" style="vertical-align:middle;margin-right:6px;"/>
										<span style="display:inline-block;width:530px;vertical-align:middle;"><label for="timing_addUpgradeTask_super" style="margin-right:10px;"><%=rb.getString("DingShiZhiXing")%></label>
										<input id="exeTime_addUpgradeTask_super" class="easyui-datetimebox border-box border" style="height:26px;"
											data-options="disabled:true,editable:false"></span>
																				
										<label style="margin-right:20px;"><%=rb.getString("BaoLiuPeiZhi")%></label>
										<select id="upgradeRawModeFlag_super" class="border border-box" style="height:26px;" data-options="editable:false">
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
			<div region="south" data-options="border:false,height:71" style="padding: 10px 20px 20px;">
				<div class="windowButtonGroup">
					<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUP_addUpgradeTask()"><span><%=rb.getString("ShangYiBu")%></span></a>
					<a class="easyui-linkbutton linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveUpgradeTask(this)"><span><%=rb.getString("WanCheng")%></span></a>
					<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelCreateUpgradeTask()"><span><%=rb.getString("QuXiao")%></span></a>
				</div>
			</div>
		</div>
	</div>
</div>
<div ></div>
<%-- 工具栏-基站列表 --%>
<div id="toolbar_GridCell_addTask_upgrade_super" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery">
	 	<li class="select_reset">
			<select id = "upgradeDeviceGroup_super" name="type" class="border border-box" style="width:103px;margin-right:1px;height: 26px;"></select>
		</li>
		<li class="serial_number input_li">
			<%-- <input name="value" type="text" class="border border-box" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>"> --%>
			<input name="value"  style="width:190px;margin-left:0px;" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		</li>
		<li>
			<b class="searchResultImgChangeStyle" onclick="$('#gridCell_addTask_upgrade_super').datagrid('reload');" ></b>	
		</li>
	</ul>
</div>

<script type="text/javascript">

var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在基站列表加载前
var operator_codes = "";


$(function() {
	var ele = $("#exeTime_addUpgradeTask_super");
	disableSelectEarlyTime(ele);
	closeLoading();
	
	$("#taskName_addUpgradeTask_super").val("${addTaskName}");
	$("#addUpgradeTaskSteps_super step_2_super").hide();
	
	// 声明基站列表
	$("#gridCell_addTask_upgrade_super").datagrid({
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
        onBeforeLoad: beforeLoad_gridCell_addTask_upgrade_super,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_GridCell_addTask_upgrade_super',
        onCheckAll:checkAll_upgrade_super,
        onBeforeCheck:beforeCheck_ugrade_super,
        onLoadSuccess: loadSuccess_addTask_upgrade,
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'serial_number', sortable: true, width: 200, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name', sortable: true, width: 200, title: '<%=rb.getString("HostName")%>'}
		]]
	});
	
	initCellGrid_addUpgradeTask();
	
	$("#upgradeDeviceGroup_super").combobox({
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



//保存当前已选择的基站对象数组
var delCellCodeObjArr = new Array();

$("#upgradeDeviceGroup_super").combobox('setValues',['<%=rb.getString("SheBeiZu")%>']); 

function checkAll_upgrade_super(rows){
	$(this).datagrid("initRow").datagrid("getPanel").find(".datagrid-htable .datagrid-header-check>input").prop("checked",true);
}
function beforeCheck_ugrade_super(index,row){
	var bool = $(this).datagrid("checkUnable",row);
	if(bool){
		return !bool;
	}
}
function chooseUpgradeDeviceGroup(data){
	choosedGroupId = data.id;
    $("#gridCell_addTask_upgrade_super").datagrid("reload");
}
var opers = "";
function tableUpgradeBeforeLoadOperatorList(param){
	opers = $("#tableOperatorList_upgrade_super").datagrid("getSelections");
	var searchOperatorCode = $("#upgradeOperatorCode_super").val();
	if (searchOperatorCode) {
		param["operator_code"] = searchOperatorCode;
	}
	// 兼容intel 和 qc
	param["isShowSlave"] = false;
	
	//清空所有的已选运营商
	/* var sels = $("#tableOperatorList_upgrade_super").datagrid("getSelections");
	if(sels.length>0 && typeof(sels[0]) != "undefined"){
		$("#tableOperatorList_upgrade_super").datagrid("unselectAll");
	} */
	
	$("#tableOperatorList_upgrade_super").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
}
function upgradeDeviceGroupBeforeLoad(param){
	param["operator_codes"] = operator_codes;
}
function loadSuc_operatorList_upgrade(){
	$("#tableOperatorList_upgrade_super").datagrid("selectRecord", "default");
}
function upgradeOperatorLoadSucc(){
	$(this).datagrid("enableContextmenuAutoSize");
	//var sels = $("#tableOperatorList_upgrade_super").datagrid("getRows");
	if(opers == ""){
		$("#tableOperatorList_upgrade_super").datagrid("selectRow",0);
	}
}
function querySelectedOperatorStation(index,row){
	operator_codes = "";
	choosedGroupId = -1;
	var selCells = $("#tableOperatorList_upgrade_super").datagrid("getSelections");
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
	$("#upgradeDeviceGroup_super").combobox({
		url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action'
	});
	$("#gridCell_addTask_upgrade_super").datagrid({
		url: '${ctx}/cell/cpeinfos/queryCpeInfosList.action?forSelect=1',
		queryParams:{like_fields:'serial_number,host_name',productValue:'${productValue}'}
	});
	$("#upgradeDeviceGroup_super").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']); 
	
}
<%-- 基站列表加载完成事件--%>
function loadSuccess_addTask_upgrade() {
	$(this).datagrid("initRow");
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	
	//初始添加-行操作提示记录
    $("#selectedCell_addUpgradTask_super").datagrid({
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
			var rowIndex = $("#gridCell_addTask_upgrade_super").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["small_cell_code"]);
			if (rowIndex != -1) {
				//$("#gridCell_addTask_upgrade_super").datagrid("deleteRow",rowIndex);
				$("#gridCell_addTask_upgrade_super").datagrid("setRow",{small_cell_code:delCellCodeObjArr[cellCount]["small_cell_code"],usable:false});
			}
		}
		//向右侧列表添加已经选中的基站
		for (var i = 0; i < delCellCodeObjArr.length; i++) {
			var row = {value: delCellCodeObjArr[i]["small_cell_code"], text: delCellCodeObjArr[i]["serial_number"] + "(" + delCellCodeObjArr[i]["host_name"] + ")"};
			$("#selectedCell_addUpgradTask_super").datagrid("appendRow", row);
		}
		 
		var rows = $("#selectedCell_addUpgradTask_super").datagrid("getRows");
		if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
			$("#selectedCell_addUpgradTask_super").datagrid("deleteRow", 0);
		}
	} 
	 layoutOnComplete();
}
<%-- 初始化基站列表--%>
function initCellGrid_addUpgradeTask() {
	<%-- 运营商搜索回车事件 --%>
	$("#upgradeOperatorCode_super").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#tableOperatorList_upgrade_super").datagrid("reload");
        }
    });
	
	<%-- 基站搜索-回车事件 --%>
    $("#toolbar_GridCell_addTask_upgrade_super input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_addTask_upgrade_super").datagrid("reload");
        }
    });
    
    <%-- 基站搜索-软件版本-下拉面板 --%>
	/* $("#softwareVersionCombo_addTask_upgrade").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	}); */
	
	<%-- 基站搜索-类型-change事件 --%>
	/* $("#toolbar_GridCell_addTask_upgrade_super select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_addTask_upgrade_super .input_li").hide();
		$("#toolbar_GridCell_addTask_upgrade_super ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_addTask_upgrade").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	}); */
	
	<%-- 基站搜索-地域树-下拉面板 --%>
	/* $("#regnTreeCombo_addTask_upgrade").combotree({
		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
		panelWidth: 200
	});
	
	$(".group_id input.textbox-text").css("padding", "0 10px").css("margin", "0"); */
}
<%-- 下一步  --%>
function stepDown_addUpgradeTask() {
	if (validateStep1()) {
		$("#addUpgradeTaskSteps_super #step_1_super").hide();
		$("#addUpgradeTaskSteps_super #step_2_super").show();
		$('#addUpgradeTaskSteps_super #step_2_super').panel("doLayout");		
	}
}
<%-- 上一步 --%>
function stepUP_addUpgradeTask() {
	$("#addUpgradeTaskSteps_super #step_2_super").hide();
	$("#addUpgradeTaskSteps_super #step_1_super").show();
}

<%-- 验证第一步输入 --%>
function validateStep1() {
	// 验证任务名称是否填写
	if ($("#taskName_addUpgradeTask_super").val().trim().length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>");
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/upgrade/taskNameExist.action", 
			data: {"taskName": $("#taskName_addUpgradeTask_super").val().trim(),"taskType":"0"},
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
	var selCells = $("#selectedCell_addUpgradTask_super").datagrid("getRows");
	if (selCells.length == 0 || selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	// 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	return true;
}

<%-- 验证第二步输入 --%>
function validateStep2() {
	var selFile = $("#gridFiles_addUpgradeFiles").datagrid("getSelected");
	if (selFile) {
		return true;
	} else {
		$.messager.alert(TiShi, "<%=rb.getString("QingXianXuanZeWenJian")%>");
		return false;
	}
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable() {
	if (document.getElementById("timing_addUpgradeTask_super").checked) {
		$("#exeTime_addUpgradeTask_super").datetimebox("enable");
	} else {
		$("#exeTime_addUpgradeTask_super").datetimebox("disable");
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_addUpgradTask() {
	var dl = $("#selectedCell_addUpgradTask_super");
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
            				//$("#gridCell_addTask_upgrade_super").datagrid("appendRow",row);
            				 $("#gridCell_addTask_upgrade_super").datagrid("setRow",{small_cell_code:row["small_cell_code"],usable:true});
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
    			var rowIndex = $("#selectedCell_addUpgradTask_super").datagrid("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_addUpgradTask_super").datagrid("deleteRow",rowIndex);
    		}
    	}
    	
		var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		//将复选框取消选中
		$("#selectedCell_addUpgradTask_super").datagrid("uncheckAll");
	}
}
<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedCell_addUpgradTask() {
	var selCell = $("#gridCell_addTask_upgrade_super").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_addUpgradTask_super");
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
		//从基站列表中删除已选择的基站
		for(var cellCount = 0; cellCount < newSelectdArr.length; cellCount++) {
			var rowIndex = $("#gridCell_addTask_upgrade_super").datagrid("getRowIndex", newSelectdArr[cellCount]["small_cell_code"]);
			var small_cell_code = newSelectdArr[cellCount]["small_cell_code"];
			//$("#gridCell_addTask_upgrade_super").datagrid("deleteRow",rowIndex);
			$("#gridCell_addTask_upgrade_super").datagrid("setRow",{small_cell_code:small_cell_code,usable:false});
		}
	}
	
	$("#gridCell_addTask_upgrade_super").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveUpgradeTask(ele) {
	if (!validateStep2()) {
		return;
	}
	var params = {};
	params["timeZone"] = timeZone;
	params["taskName"] = $("#taskName_addUpgradeTask_super").val().trim();
	
	// 取得已选择的基站
	var cellCodes = "";
	var selCells = $("#selectedCell_addUpgradTask_super").datagrid("getRows");
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
		params["time"] = $("#exeTime_addUpgradeTask_super").datetimebox("getValue");
	}
	// 取得选择的文件
	var selFile = $("#gridFiles_addUpgradeFiles").datagrid("getSelected");
	params["fileId"] = selFile["id"];
    params["fileName"] = selFile["file_name"];
    params["taskType"] = "1";
	params["rawMode"] = $("#upgradeRawModeFlag_super").val();
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
function cancelCreateUpgradeTask() {
	/* $("#winAddUpgradTask").window("close"); */
	closeDefaultWindow();
}

//加载前事件-基站列表
function beforeLoad_gridCell_addTask_upgrade_super(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $("#toolbar_GridCell_addTask_upgrade_super .inputslist li.serial_number>input").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	if(operator_codes != ""){
		param["operator_codes"] = operator_codes;
	}else{
		param["operator_codes"] = "-1";//用于标识，为-1，则表明未选中任务运营商
	}
	// 兼容intel 和 qc
	param["isShowSlave"] = false;
	
	$("#gridCell_addTask_upgrade_super").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
	
}
// 加载前事件 - 选择升级文件
function beforeLoadAddUpgradeFiles(param){
	// 兼容intel 和 qc
	param["isShowSlave"] = false;
}
function radioMatterSupper(value, rowData, rowIndex){
   	value = '<input type="radio" name="productType"/>';
    return value;
}
function onSelect_softwareFile_supper(param){
	if($("#gridFiles_addUpgradeFiles").datagrid("getSelected")){
		$($("input[name=productType]")[param]).prop("checked",true);
	}
}
</script>