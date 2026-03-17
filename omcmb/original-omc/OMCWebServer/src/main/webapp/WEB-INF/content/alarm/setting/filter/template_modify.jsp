<%@ page import="java.util.Map" %>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>

<style>
.itemUl{
	height:auto;
	margin:20px;
}
.itemUl .itemLi{
	display:inline-block;
	width:230px;
	margin:10px;
}
</style>

<div id="modifyFilterTemplate" class="easyui-layout" fit="true" data-options="border:false">
   	<div region="center" class="easyui-tabs" data-options="border:false">
		<div id="add_tab1" title="<%=rb.getString("ShengXiaoJiZhan")%>" data-options="border:false" style="padding: 10px 15px;">
	    	<%-- 生效基站 --%>
	    	<div class="easyui-layout" data-options="fit:true,border:false">
				<div region="north" data-options="border:false,height:46" style="padding: 15px 10px 5px 20px;">
					<span><%=rb.getString("GuoLvMoBan")%></span>
					<input id="filterName" class="border border-box" style="width:600px;margin-left:5px;height:26px;"
							placeholder="<%=rb.getString("QingShuRuMuBanMingCheng")%>">
				</div>
				<div region="center" data-options="border:false,width:600" style="padding: 10px 20px;">
					<div class="easyui-layout" data-options="border:false,fit:true">
						<div region="west" data-options="width:370,border:true,collapsible:false" title="<%=rb.getString("JiZhanLieBiao")%>" class="padT20">
							<table id="gridCell_filter_modify">
							</table>
						</div>
						<div region="center" data-options="border:false,onResize:setArrowMargin">
							<a href="javascript:void(0);" class="arrow_right" onclick="addSelectedCell_filter()"></a>
							<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_filter()"></a>
						</div>
						<div region="east" data-options="width:340,border:false,collapsible:false">
							<%-- 已选择基站 --%>
							<ul id="selectedCell_filter_modify" class="easyui-datalist" data-options="singleSelect:false,border:true,fit:true,title:'<%=rb.getString("YiXuanZeJiZhan")%>'">
								<li value=""><%=rb.getString("QingXuanZe")%></li>
							</ul>
						</div>
					</div>
				</div>		
			</div>
	    </div>
	    <div id="add_tab2" title="<%=rb.getString("GaoJingLeiXing")%>" data-options="border:false" style="padding: 10px 15px;">
	      	<%-- 告警类型列表 --%>
	      	<div class="easyui-panel" data-options="border:true,fit:true,title:'<%=rb.getString("GaoJingLeiXing")%>'">
				<ul class="itemUl">
					<li class="itemLi">
						<input id="_01" type="checkbox" class="itemInput alarmType" value="30000">
						<label for="_01" class="itemLabel"><%=rb.getString("TongXinGaoJing")%></label>
					</li>
					<li class="itemLi">
						<input id="_02" type="checkbox" class="itemInput alarmType" value="30001">
						<label for="_02" class="itemLabel"><%=rb.getString("FuWuZhiLiangGaoJing")%></label>
					</li>
					<li class="itemLi">
						<input id="_03" type="checkbox" class="itemInput alarmType" value="30002">
						<label for="_03" class="itemLabel"><%=rb.getString("ChuLiShiBaiGaoJing")%></label>
					</li>
					<li class="itemLi">
						<input id="_04" type="checkbox" class="itemInput alarmType" value="30003">
						<label for="_04" class="itemLabel"><%=rb.getString("SheBeiGaoJing")%></label>
					</li>
					<li class="itemLi">
						<input id="_05" type="checkbox" class="itemInput alarmType" value="30004">
						<label for="_05" class="itemLabel"><%=rb.getString("HuanJingGaoJing")%></label>
					</li>
					<!-- <li class="itemLi">
						<input id="_06" type="checkbox" class="itemInput alarmType" value="30007">
						<label for="_06" class="itemLabel">Event Alarm</label>
					</li> -->
					<li class="itemLi">
						<input id="_07" type="checkbox" class="itemInput alarmType" value="30006">
						<label for="_07" class="itemLabel"><%=rb.getString("XingNengYiChuGaoJing")%></label>
					</li>
				</ul>
			</div>
	    </div>
	    <div id="add_tab3" title="<%=rb.getString("GaoJingJiBie")%>" data-options="border:false" style="padding: 10px 15px;">
	      	<%-- 告警级别列表 --%>
			<div class="easyui-panel" data-options="border:true,fit:true,title:'<%=rb.getString("GaoJingJiBie")%>'">
				<ul class="itemUl">
					<li class="itemLi">
						<input id="_07" type="checkbox" class="itemInput alarmLevel" value="31001" >
						<label for="_07" class="itemLabel"><%=rb.getString("JinJiGaoJing")%></label>
					</li class="itemLi">
					<li class="itemLi">
						<input id="_08" type="checkbox" class="itemInput alarmLevel" value="31002" >
						<label for="_08" class="itemLabel"><%=rb.getString("ZhuYaoGaoJing")%></label>
					</li>
					<li class="itemLi">
						<input id="_09" type="checkbox" class="itemInput alarmLevel" value="31003" >
						<label for="_09" class="itemLabel"><%=rb.getString("CiYaoGaoJing")%></label>
					</li>
					<li class="itemLi">
						<input id="_10" type="checkbox" class="itemInput alarmLevel" value="31004" >
						<label for="_10" class="itemLabel"><%=rb.getString("JingGaoGaoJing")%></label>
					</li>
				</ul>
			</div>
	    </div>
	    <div id="add_tab4" title="<%=rb.getString("GuZhangShiJian")%>" data-options="border:false" style="padding: 10px 15px;">
	      	<%-- 故障时间列表 --%>
	      	<div class="easyui-panel" data-options="border:true,fit:true,title:'<%=rb.getString("GuZhangShiJian")%>'" style="padding:20px;">
				<input id="start_time" class="easyui-datetimebox border-box border" style="height:26px;width:200px;">
				<span> - </span>
				<input id="end_time" class="easyui-datetimebox border-box border" style="height:26px;width:200px;">
			</div>
	    </div>
	</div>
    <div region="south" data-options="border:false,height:71" style="padding:10px 20px 20px;">
    	<div class="windowButtonGroup">
			<a onclick="modifyFilter()" class="linkbutton linkbutton_trend"><span><%=rb.getString("QueDing")%></span></a>
			<a onclick="closeDefaultWindow()" class="linkbutton linkbutton_nowanna"><span><%=rb.getString("QuXiao")%></span></a>
    	</div>
		
	</div>
</div>

<%-- 工具栏-基站列表 --%>
<div id="toolbar_gridCell_filter_modify" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery">
		<li style="margin-left:8px;margin-right:20px">
			<select id="toolbar_gridCell_filter_type_modify" name="type" class="easyui-combobox border border-box" style="width:100px;margin-right:20px;height:26px;">
				<option value="serial_number"><%=rb.getString("XiaoZhanBianMa")%></option>
				<option value="host_name"><%=rb.getString("HostName")%></option>n> --%>
			</select>
		</li>
		<li class="serial_number input_li">
			<input name="value" style="width:180px;margin-left:0px;" placeholder="<%=rb.getString("XiaoZhanBianMa")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
			<!-- <input name="value" type="text" class="border border-box" style="width:143px;"> -->
		</li>
		<li class="host_name input_li" style="display: none;">
			<input name="value" style="width:180px;margin-left:0px;" placeholder="<%=rb.getString("HostName")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
			<!-- <input name="value" type="text" class="border border-box" style="width:143px;"> -->
		</li>
		<li>
			<b class="searchResultImgChangeStyle" onclick="$('#gridCell_filter_modify').datagrid('reload');"></b>	

		</li>
	</ul>
</div>

<script type="text/javascript">
var templateId = "${templateId}";
var delCellCodeObjArr = new Array();//存储基站列表中需要删除的基站信息

$(function() {
	closeLoading();
	
	$("#gridCell_filter_modify").datagrid({
		fit: true,
		border: false,
		fitColumns: true,
		rownumbers: true,
		url: '${ctx}/cell/cpeinfos/queryCpeInfosList.action?forSelect=1',
		queryParams:{like_fields:"serial_number"},
		pageSize: 100,
		pageList: [100],
		striped: true,
		pagination: true,
		pagePosition: 'bottom',
		idField: 'small_cell_code',
		onBeforeCheck: beforeCheck_gridCell_filter,
		onCheckAll: checkAll_gridCell_filter,
		onBeforeLoad: beforeLoad_gridCell_filter,
		onLoadSuccess: loadSuccess_gridCell_filter,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_gridCell_filter_modify',
		columns: [[
					{field: 'ck', checkbox:true},
					{field: 'small_cell_code', hidden: true},
					{field: 'connection_status',sortable:true,fixed:true,width: 25,formatter:connStatusFormatter},
					{field: 'serial_number',sortable:true,width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
					{field: 'host_name',sortable:true,width: 100, title: '<%=rb.getString("HostName")%>'}
		]]
	});
	
	$("#gridCell_filter_modify").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
	
	setTimeout(function(){
		loadFilterById(templateId);
	},100);
	
	initCellGrid_gridCell_filter();
});
function beforeCheck_gridCell_filter(index,row){
	return !$(this).datagrid('checkUnable',row);
}
function checkAll_gridCell_filter(rows){
	$(this).datagrid('initRow').datagrid('getPanel').find('.datagrid-htable .datagrid-header-check>input').prop('checked',true);
}
//初始化基站列表工具栏
function initCellGrid_gridCell_filter() {
	<%-- 基站搜索-回车事件 --%>
    $("#toolbar_gridCell_filter_modify input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_filter_modify").datagrid("reload");
        }
    });
    
    <%-- 基站搜索-软件版本-下拉面板 --%>
	$("#softwareVersionCombo_filter_modify").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	});
	
	<%-- 基站搜索-类型-change事件 --%>
	/* $("#toolbar_gridCell_filter_modify select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_gridCell_filter_modify .input_li").hide();
		$("#toolbar_gridCell_filter_modify ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_filter_modify").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	}); */
	$("#toolbar_gridCell_filter_modify select[name='type']").combobox({
		onSelect:function(e) {
			var selected = $("#toolbar_gridCell_filter_type_modify").combobox('getValue');
			$("#toolbar_gridCell_filter_modify .input_li").hide();
			$("#toolbar_gridCell_filter_modify ." + selected).show();
		}
	});
}

//按模板ID加载过滤模板信息
function loadFilterById(idVal) {
	if (!idVal) {
		return;
	}
	delCellCodeObjArr.splice(0, delCellCodeObjArr.length);
	$.post("${ctx}/cell/fault/queryFilterInfoById.action", {"tem_id" : idVal,"timezone" : timeZone}, function(data) {
		//模板名称
		$("#modifyFilterTemplate #filterName").val(data["template_name"]);
		// 已选择基站
		var cells = data["cells"];
		if (cells.length == 0) {
			cells[0] = {"value": "", "text": "<%=rb.getString("QingXuanZe")%>"};
		}
		$("#selectedCell_filter_modify").datalist("loadData", cells);
		
		var cellsLength = cells.length;
		for(var cellCount = 0; cellCount < cellsLength; cellCount++) {
    		var rowIndex = $("#gridCell_filter_modify").datagrid("getRowIndex", cells[cellCount].value);
    		if (rowIndex != -1) {
    			//$("#gridCell_filter_modify").datagrid("deleteRow",rowIndex);
    			$("#gridCell_filter_modify").datagrid("setRow",{small_cell_code:cells[cellCount].value, usable:false});
    		}
    		var selectedCellObj = new Object();
    		selectedCellObj.small_cell_code = cells[cellCount].value;
    		selectedCellObj.serial_number = cells[cellCount].serialNumber;
   			selectedCellObj.host_name = cells[cellCount].hostName;
   			selectedCellObj.connection_status = cells[cellCount].connectionStatus;
    		delCellCodeObjArr.push(selectedCellObj);
    	}
		
		$("#gridCell_filter").datagrid("reload");
		
		// 告警类型
		var alarmTypeCheckboxArr = $("#modifyFilterTemplate .alarmType");
		for (var i = 0; i < alarmTypeCheckboxArr.length; i++) {
			alarmTypeCheckboxArr[i].checked = false;
		}
		
		var alarmTypes = data["alarmType"];
		for (var i = 0; i < alarmTypes.length; i++) {
			$("#modifyFilterTemplate .alarmType[value=" + alarmTypes[i] + "]")[0].checked = true;
		}
		
		// 告警级别
		var alarmLevelCheckboxArr = $("#modifyFilterTemplate .alarmLevel");
		for (var i = 0; i < alarmLevelCheckboxArr.length; i++) {
			alarmLevelCheckboxArr[i].checked = false;
		}
		
		var alarmLevels = data["alarmLevel"];
		for (var i = 0; i < alarmLevels.length; i++) {
			$("#modifyFilterTemplate .alarmLevel[value=" + alarmLevels[i] + "]")[0].checked = true;
		}
		
		// 故障时间：开始时间、结束时间
		$("#modifyFilterTemplate #start_time").datetimebox("setValue", data["start_time"]);
		$("#modifyFilterTemplate #end_time").datetimebox("setValue", data["end_time"]);
	}, "json")
}

function modifyFilter() {
	var filterName = $("#modifyFilterTemplate #filterName").val();
	if (!filterName) {
		showMsg('prompt_msg','<%=rb.getString("MuBanMingChengBuNengWeiKong")%>')
		return;
	}
	if(filterName.length>50){
		showMsg('prompt_msg','<%=rb.getString("MuBanMingChengChangDuXianZhi")%> 50')
		return;
	}
	if(validateFilterName(filterName,templateId)){
		// 基站
		var cellCodes = "";
		var selCells = $("#modifyFilterTemplate #selectedCell_filter_modify").datalist("getRows");
		for (var i = 0; i < selCells.length; i++) {
			cellCodes += selCells[i]["value"] + ",";
		}
		if (cellCodes == ",") {
			showMsg('prompt_msg','<%=rb.getString("QingXuanZeSheBei")%>')
			return;
		}
		
		// 告警类型
		var alarmTypes = "";
		$("#modifyFilterTemplate .alarmType").each(function() {
			if (this.checked) {
				alarmTypes += this.value + ",";
			}
		});
		
		// 告警级别
		var alarmLevels = "";
		$("#modifyFilterTemplate .alarmLevel").each(function() {
			if (this.checked) {
				alarmLevels += this.value + ",";
			}
		});
		
		if (alarmTypes == "" && alarmLevels == "") {
			showMsg('prompt_msg','<%=rb.getString("GaoJingSheZhiTiShi")%>')
			return;
		}
		
		var params = {
			timeZone:timeZone,
			"name": filterName,
			"cellCodes": cellCodes,
			"alarmTypes": alarmTypes,
			"alarmLevels": alarmLevels,
			"startTime": $("#modifyFilterTemplate #start_time").datetimebox("getValue"),
			"endTime": $("#modifyFilterTemplate #end_time").datetimebox("getValue"),
			"tem_id": templateId
			//"enable": $("#addFilterTemplate input[name='radioEnable_filter'][filter_enable='1']")[0].checked + ""
		};
		$.post("${ctx}/cell/fault/modifyAlarmFilterTemplate.action", params, function(data) {
			if (data["success"]) {
				//$("#fault_filter #filterName").combobox("reload", "${ctx}/cell/fault/queryFilterName.action");
				showMsg('success_msg','<%=rb.getString("CaoZuoChengGong")%>')
				//$("#winModifyFilterTemp").window("close");
				closeDefaultWindow();
				$("#gridAlarm").datagrid("reload");
				$("#gridAlarmHis").datagrid("reload");
				$("#tableAlarmFilter").datagrid("reload");
				refreshAliveAlarmCount();
			} else {
				showMsg('error_msg',data["message"])
			}
		}, "json");
	}
}

//加载前事件-基站列表
function beforeLoad_gridCell_filter(param) {
	try {
		var type = $("#toolbar_gridCell_filter_type_modify").combobox("getValue");
		var val = $("#toolbar_gridCell_filter_modify ." + type + " input[name='value']").val();
		//var val = $("#toolbar_gridCell_filter_modify .serial_number input[name='value']").val();
		param[type] = val;
		
	} catch (e) {}
}

function loadSuccess_gridCell_filter() {
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	$(this).datagrid('initRow');
	/* if (delCellCodeObjArr.length > 0) {
    	for(var cellCount = 0; cellCount < delCellCodeObjArr.length; cellCount++) {
    		var rowIndex = $("#gridCell_filter_modify").datagrid("getRowIndex", delCellCodeObjArr[cellCount]["small_cell_code"]);
    		if (rowIndex != -1) {
    			$("#gridCell_filter_modify").datagrid("deleteRow",rowIndex);
    		}
    	}
    } */
}

//选择小站-向右键头的点击事件
function addSelectedCell_filter() {
	var selCell = $("#gridCell_filter_modify").datagrid("getSelections");
	var selCellLength = selCell.length;
	if (selCellLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#selectedCell_filter_modify");
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
   			//selectedCellObj.groupId = choosedGroupId;
   			delCellCodeObjArr.push(selectedCellObj);
   			newSelectdArr.push(selectedCellObj);
		}
	}
	
	if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		dl.datagrid("deleteRow", 0);
	}
		
	if (newSelectdArr.length > 0) {
		//从基站列表中删除已选择的基站
		for(var cellCount = 0; cellCount < newSelectdArr.length; cellCount++) {
			var rowIndex = $("#gridCell_filter_modify").datagrid("getRowIndex", newSelectdArr[cellCount]["small_cell_code"]);
			//$("#gridCell_filter_modify").datagrid("deleteRow",rowIndex);
			// 置灰，使数据不可选
			var smcode = newSelectdArr[cellCount]["small_cell_code"];
			$("#gridCell_filter_modify").datagrid("setRow",{small_cell_code: smcode,usable: false});
		}
	}
	
	$("#gridCell_filter_modify").datagrid("uncheckAll");
}

//删除已选中的小站-向左键头的点击事件
function delSelectedCell_filter() {
	var dl = $("#selectedCell_filter_modify");
	var selCell = dl.datalist("getSelections");
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
            			//$("#gridCell_filter_modify").datagrid("appendRow",row);
            			// 恢复数据可选状态
            			$("#gridCell_filter_modify").datagrid("setRow",{small_cell_code:row.small_cell_code,usable: true});
            			delCellCodeObjArr.splice(cellCount,1);
            			needDelCellArr.push(selCell[selCellCount]);
            			break;
        			}
        		}
    		}
		}
		
    	if (needDelCellArr.length > 0) {
    		//从已选基站列表中删除基站
    		for(var cellCount = 0; cellCount < needDelCellArr.length; cellCount++) {
    			var rowIndex = $("#selectedCell_filter_modify").datalist("getRowIndex", needDelCellArr[cellCount]);
    			$("#selectedCell_filter_modify").datalist("deleteRow",rowIndex);
    		}
    	}
    	
		var rows = dl.datalist("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datalist("insertRow", row);
		}
		//将复选框取消选中
		$("#selectedCell_filter_modify").datalist("uncheckAll");
	}
}

//验证名称是否可用
function validateFilterName(filterName,templateId) {
	if (!filterName) {
		showMsg('prompt_msg','<%=rb.getString("QingShuRuMuBanMingCheng")%>')
		return false;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/cell/fault/filterNameExist.action", 
			data: {"name": filterName,
				   "templateId":templateId},
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
			showMsg('prompt_msg','<%=rb.getString("MuBanMingChengYiCunZai")%>')
			return;
		}
	}
	return true;
}
</script>