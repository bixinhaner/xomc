<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
 input:focus{
	border:none;
	border-bottom:1px solid #209FFF;
	outline:none;
}
</style>
<%-- 设置基站周期上报日志页面 --%>
<div id="setPeriodReportLogSteps" class="easyui-panel" data-options="border:false,fit:true" style="padding-top: 20px">
	<%-- 第一步 --%>
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 0 20px 20px;">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="west" data-options="width:370,border:true,collapsible:false" title="<%=rb.getString("JiZhanLieBiao")%>" style="padding:20px 0 0">
					<table id="gridCell_period_collection"></table>
				</div>
				<div region="center" data-options="border:false,onResize:setArrowMargin">
					<a href="javascript:void(0);" class="el-icon el-icon-common-arrow-right" style='margin-top:233px;font-size:30px;margin-left:25px;' onclick="addSelectedCell_setPeriodReportLog()"></a>
					<a href="javascript:void(0);" class="el-icon el-icon-common-arrow-left" style="margin-top: 20px;font-size:30px;margin-left:25px;" onclick="delSelectedCell_setPeriodReportLog()"></a>
				</div>
				<div class="special2" region="east" data-options="width:370,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeJiZhan")%>'" style="padding-top:20px;" >
					<%-- 已选择基站 --%>
			    	<table class="easyui-datagrid" id="selectedCell_setPeriodReportLog"
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
								<th data-options="field:'connection_status',hidden:true"></th>
								<th data-options="field:'small_cell_code', hidden: true"></th>
								<th data-options="field:'software_version', hidden: true"></th>
								<th data-options="field:'serial_number',hidden:true"></th>
								<th data-options="field:'host_name',hidden:true"></th>
								<th data-options="field:'value',hidden:true"></th>
								<th data-options="field:'text'" width="300">
									<%=rb.getString("XiaoZhanBianMa")%><%=rb.getString("ZuoKuoHao")%><%=rb.getString("HostName")%><%=rb.getString("YouKuoHao")%>
								</th>
							</tr>
						</thead>
			  		</table>
				</div>
				<div region="south" data-options="height:160,border:false,collapsible:false" style="margin-top:10px">
					<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding: 10px 20px;">
						<div class="verM">
							<input type="radio" id="active_setPeriodReportLog" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable_setPeriodReportLog()"/>
							<label for="active_setPeriodReportLog" style="display:inline-block;width:390px;"><%=rb.getString("LiJiZhiXing")%></label>
						</div>
						<div class="dateRe" style="margin-top: 10px;">
							<input type="radio" id="timing_setPeriodReportLog" status="timing" name="taskStatus" onchange="setExeTimerEnable_setPeriodReportLog()"/>
							<label for="timing_setPeriodReportLog" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
							<input id="periodReportLogStartTime" class="easyui-datetimebox border-box border" style="height:26px;"
								data-options="disabled:true,editable:false">
							<label style="margin-right:5px;margin-left:5px;">-</label>
							<input id="periodReportLogEndTime" class="easyui-datetimebox border-box border" style="height:26px; margin-left:5px;"
								data-options="disabled:true,editable:false">
						</div>
						<div style="margin-top: 10px;">
							<%--如果是中文界面 --%>
						    <c:if test="${localeIsZh == '1'}">
						    	<label style="display: inline-block; width: 85px" class="borderBoxClass"><%=rb.getString("FenZhongZhouQi")%><%=rb.getString("MaoHao")%></label>
   							</c:if>
   							<%--如果是英文界面 --%>
   							 <c:if test="${localeIsZh == '0'}">
						    	<label style="display: inline-block; width: 120px" class="borderBoxClass"><%=rb.getString("FenZhongZhouQi")%><%=rb.getString("MaoHao")%></label>
   							</c:if>
							<select id="periodReportLogMin" class="border border-box borderBoxClass" style="width: 180px; margin-left: 2px">
								<option value="900" selected="selected">15</option>
								<option value="1800">30</option>
								<option value="2700">45</option>
								<option value="3600">60</option>
							</select>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div region="south" data-options="border:false,height:56" >
			<div class="windowButtonGroup" style="margin-right:20px">
				<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="confirmPeriodReportLogFile()"><span><%=rb.getString("QueDing")%></span></a>
				<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelPeriodReportLogFile()" ><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			
		</div>
	</div>
</div>

<%-- 工具栏-基站列表 --%>
<div id="toolbar_gridCell_period_collection" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery">
		<li class="select_reset">
			<select id="periodLogDeviceGroup" name="type" class="border border-box" style="width:100px;margin-right:1px;height: 26px;"></select>
		</li>
		<div class='queryGroup'>
			<input style="width:140px;border:none;"  name="value" type="text" class="border border-box" placeholder="<%=rb.getString("XiaoZhanBianMa")%>">
			<b class='el-icon el-icon-common-search' onclick="$('#gridCell_period_collection').datagrid('reload');"></b>
		</div>
		<%-- <li class="serial_number input_li">
			<input style="width:202px;border:none;border-bottom:1px solid #D0D9DE;"  name="value" type="text" class="border border-box" placeholder="<%=rb.getString("XiaoZhanBianMa")%>">
		</li>
		<li>
			<b onclick="$('#gridCell_period_collection').datagrid('reload');"></b>
		</li> --%>
	</ul>
</div>

<script type="text/javascript">
var lang = "${language}";

$(function() {
	closeLoading();
	
	//初始添加-行操作提示记录
    $("#selectedCell_setPeriodReportLog").datagrid({
    	url : "",
    	data: {
    		total: 1,
    		rows: [{ value: "", text: "<%=rb.getString("QingXuanZe")%>" }]
        }
    });
	
	initCellGrid_setPeriodReportLog();	
	$("#gridCell_period_collection").datagrid({
		border: false,
		fitColumns: true,
		fit: true,
        rownumbers: true,
        url: "${ctx}/system/device/enodeb/queryENBInfoPageListForSelect.action",
        pageSize: 100,
        pageList: [100],
        queryParams:{
        	like_fields : 'serial_number'
        },
        striped: true,
        singleSelect: false,
        pagination: true,
        pagePosition: 'bottom',
        idField: 'small_cell_code',
        onBeforeLoad: beforeLoad_gridCell_periodReport_log,
        onLoadSuccess: loadSuccess_periodReport_log,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_gridCell_period_collection',
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'software_version',hidden: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'serial_number', sortable: true, width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name', sortable: true, width: 100, title: '<%=rb.getString("HostName")%>'}
		]]
	});
	
	$("#gridCell_period_collection").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
});

function loadSuccess_periodReport_log() {
	$(this).datagrid("enableContextmenuAutoSize");
	$(this).datagrid("fixRownumber");
}
<%-- 初始化基站树 --%>
function initCellGrid_setPeriodReportLog() {
	<%-- 基站搜索-回车事件 --%>
    $("#toolbar_gridCell_period_collection input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_period_collection").datagrid("reload");
        }
    });
	
    $("#periodLogDeviceGroup").combobox({
    	url: "${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action",
        width: 100,
        panelWidth: 150,
        panelHeight: 200,
        valueField: 'id',
        textField: 'group_name',
        onSelect: function(data){
	     	$("#gridCell_period_collection").datagrid("reload");	
	    }
    });
	$("#periodLogDeviceGroup").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']);
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable_setPeriodReportLog() {
	if (document.getElementById("timing_setPeriodReportLog").checked) {
		$("#periodReportLogStartTime").datetimebox("enable");
		$("#periodReportLogEndTime").datetimebox("enable");
	} else {
		$("#periodReportLogStartTime").datetimebox("disable");
		$("#periodReportLogEndTime").datetimebox("disable");
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_setPeriodReportLog() {
	var dg_right = $("#selectedCell_setPeriodReportLog");
	var selectRecordsRight = dg_right.datagrid("getSelections");
    if (selectRecordsRight.length == 0) {
        return;
    }
    
    var dg_left = $("#gridCell_period_collection");
    var selectRecordsLeft = dg_left.datagrid("getRows");
    
    var gridPageLeft = dg_left.datagrid("getPager").data("pagination").options;
    var pageSizeLeft = gridPageLeft.pageSize;
    
    var recordsRight = selectRecordsRight.slice();
    $.each(recordsRight,function(index_right,obj_right){
    	var cell_code_right = obj_right.small_cell_code;
        //删除右侧选中的基站
        var rowIndex=dg_right.datagrid("getRowIndex", cell_code_right);
        dg_right.datagrid("deleteRow",rowIndex);
    })
    var rows = dg_right.datagrid("getRows");
    if (rows.length == 0) {
        var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
        dg_right.datagrid("insertRow", row);
    }
    dg_right.datagrid("uncheckAll");
    dg_left.datagrid("reload");
}

<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedCell_setPeriodReportLog() {
	var dg_left = $("#gridCell_period_collection");
	var selectRecordsLeft = dg_left.datagrid("getSelections");
	if (selectRecordsLeft.length == 0) {
		return;
	}
	
	var dg_right = $("#selectedCell_setPeriodReportLog");
    var selectRecordsRight = dg_right.datagrid("getRows");
    if(selectRecordsRight[0]["value"]==""){
    	dg_right.datagrid("deleteRow", 0);
    }
    var recordsLeft = selectRecordsLeft.slice();
	$.each(recordsLeft,function(index_left,obj_left){
		var cell_code_left = obj_left.small_cell_code;
		var is_exit=false;
        $.each(selectRecordsRight,function(index_right,obj_right){
            var cell_code_right = obj_right.small_cell_code;
            if(cell_code_left==cell_code_right){
            	is_exit=true;
                return;
            }
        })
        
        if(!is_exit){
        	var row={};
        	$.each(obj_left,function(key,value){
        		row[key]=value;
        	})
        	row.value=obj_left.small_cell_code;
        	row.text=obj_left.serial_number + "(" + obj_left.host_name + ")";
        	dg_right.datagrid("appendRow", row);
        }
    })
    dg_left.datagrid("uncheckAll");
	dg_left.datagrid("reload");
}

<%-- 保存设置内容 --%>
function confirmPeriodReportLogFile() {
	
	var isSelectQCCell = false; //标识是否选择高通的站
	var params = {};
	
	// 取得已选择的基站
	var cellCodes = "";
	var serialNumbers = "";
	var selCells = $("#selectedCell_setPeriodReportLog").datalist("getRows");
	
	// 是否已选择小站
	if (selCells.length == 1 && (selCells[0]["value"].length == 0 || selCells[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	
	for (var i = 0; i < selCells.length; i++) {
		cellCodes += selCells[i]["value"] + ",";
		
		var indexNum = selCells[i]["text"].indexOf("(", 0);
		serialNumbers += selCells[i]["text"].substring(0, indexNum) + ",";
		
		var softwareVersion = selCells[i]["software_version"];
		if (softwareVersion != null) {
			if (softwareVersion.indexOf("FDD_V") == 0 || softwareVersion.indexOf("BaiBS_QA") == 0) {
				isSelectQCCell = true;
			}
		}
	}
	cellCodes = cellCodes.substring(0, cellCodes.length - 1);
    serialNumbers = serialNumbers.substring(0, serialNumbers.length - 1);
    
    params["serialNumbers"] = serialNumbers;
	params["cellCodes"] = cellCodes;
	// 取得执行方式：激活、挂起、定时执行
	var radios = document.getElementsByName("taskStatus");
	for (var i = 0; i < radios.length; i++) {
		if (radios[i].checked == true) {
			params["status"] = $(radios[i]).attr("status");
		}
	}
	
    var periodMin = $("#periodReportLogMin").val();
    params["reportPeriod"] = periodMin;
    
	// 如果是定时执行的，取得时间
	if (params["status"] == "timing") {
	    var startTime = $("#periodReportLogStartTime").datetimebox("getValue");
	    var endTime = $("#periodReportLogEndTime").datetimebox("getValue");
	    
	  	 //开始时间不能为空
	    if (startTime.length == 0) {
	   	$.messager.alert(TiShi, "<%=rb.getString("KaiShiShiJianBuNengKong")%>");
			return false;
	    }
	  
	    //结束时间不能为空
	    if (endTime.length == 0) {
	   	$.messager.alert(TiShi, "<%=rb.getString("JieShuShiJianBuNengKong")%>");
			return false;
	    }
	    //结束时间不能早于当前时间
	    if (new Date(gloableTime) > dateParser(endTime).getTime()) {
	   	 $.messager.alert(TiShi, "<%=rb.getString("JieShuZaoYuDangQianShiJian")%>");
	        return false;
	    }
	    //开始时间不能晚于结束时间
	    var validTimeResult = validateStartAndStopTime(startTime, endTime);
	    if ("false" == validTimeResult) {
	        $.messager.alert(TiShi, "<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>");
	        return false;
	    }
	    
		params["startTime"] = $("#periodReportLogStartTime").datetimebox("getValue");
		params["endTime"] = $("#periodReportLogEndTime").datetimebox("getValue");
	}
	
	params["timeZone"] = timeZone;
	
	<%-- 高通平台需要重启基站使参数生效--%>
	if (isSelectQCCell) {
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("CaoZuoWanChengHouChongQi")%>", function(r) {
			if (r) {
				params["isReboot"] = "false";
				postSetCellPeriodLog(params);
			}
		}).addClass("normalConfirm");
	} else {
		params["isReboot"] = "false";
		postSetCellPeriodLog(params);
	}
}

<%--设置周期上报日志--%>
function postSetCellPeriodLog(params) {
	$.post("${ctx}/cell/collect/customizePeriodReportLogFile.action", params, function(data) {
		/* $("#winPeriodCollect").window("close"); */
		closeDefaultWindow();
		if (data["success"]) {
			$("#periodLogFileTaskDatagrid").datagrid("load");
		} else {
			$.messager.alert(TiShi, data["message"]);
		}
}, "json"); 
}
<%--验证开始时间和结束时间是否合法--%>
function validateStartAndStopTime(startTimeStr, endTimeStr){
    if (null != startTimeStr && null != endTimeStr) {
        var startDate = dateParser(startTimeStr);
        var endDate = dateParser(endTimeStr);
        if (startDate.getTime() < endDate.getTime()) {
            return "true";
        }
    }
    return "false";
}

// 取消设置周期上报
 function cancelPeriodReportLogFile() {
 	/* $("#winPeriodCollect").window("close"); */
 	closeDefaultWindow();
 	return;
 }

//加载前事件-基站列表
function beforeLoad_gridCell_periodReport_log(param) {
 	var group_id = $("#periodLogDeviceGroup").combobox("getValue");
	if(group_id>0){
		param["group_id"] = group_id;
	}
	var search_text = $("#toolbar_gridCell_period_collection .queryGroup>input").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	
	if ("${isElfCell}" == "1") {
		param["onlySupportKPIAndLog"] = "1";
	} 
	
	var dg_right = $("#selectedCell_setPeriodReportLog");
    var selectRecordsRight = dg_right.datagrid("getRows");
    if(selectRecordsRight[0]["value"] != ""){
    	var enode_codes = "";
    	$.each(selectRecordsRight,function(index_right,obj_right){
            var cell_code_right = obj_right.small_cell_code;
            enode_codes += cell_code_right + ",";
        })
        if(enode_codes!=""){
        	enode_codes = enode_codes.substring(0, enode_codes.length - 1);
        }
    	param["selected_enodecodes"] = enode_codes;
    }
}
</script>