<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 编辑Software任务 --%>
<div id="editSoftwareAutoTask" class="easyui-panel" data-options="border:false,fit:true">
	<%-- 第一步 --%>
	<div id="step_1" class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 20px;">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="west" class="padT20" data-options="width:370,border:true,collapsible:false" title="<%=rb.getString("ChuShiBanBen")%>">
					<table id="gridOriSoftwareVer"></table>
				</div>
				<div region="center" data-options="border:false,onResize:setArrowMargin">
					<a id="arrow_right_addUpgradeBiosTask" href="javascript:void(0);" class="arrow_right" onclick="addSelectedOriSoftwareVer()"></a>
					<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_addUpgradeBiosTask()"></a>
				</div>
				<div region="east" class="padT20" data-options="width:370,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeBanBen")%>'">
					<%-- 已选择 --%>
			    	<table class="easyui-datagrid" id="gridSelectedOriSoftwareVer"></table>
				</div>
				<div region="south" id="editAutoSoftUpgradeNorth" data-options="height:145,border:false,collapsible:false" style="padding-top: 10px;">
					<div class="easyui-panel" data-options="fit:true,border:true,title:'<%=rb.getString("XuanZeZhiXingFangShi")%>'" style="padding: 15px 25px;">
						<div>
							<label style="margin-right:16px;"><%=rb.getString("BaoLiuPeiZhi")%></label>
							<select id="upgradeRawModeFlag" class="border border-box"
								style="height:26px;width: 180px">
								<option value="false" selected="selected"><%=rb.getString("Shi")%></option>
								<option value="true"><%=rb.getString("Fou")%></option>
							</select>
						</div>
						<div class="dateRe" style="margin-top: 10px;">
							<label style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
							<input id="autoTaskExecuteStartTime" class="easyui-datetimebox border-box border" style="height:26px;">
							<label style="margin-right:5px;margin-left:5px;">-</label>
							<input id="autoTaskExecuteEndTime" class="easyui-datetimebox border-box border" style="height:26px; margin-left:5px;">
						</div>
				    </div>
				  </div>
			</div>
		</div>
		<div region="south" data-options="border:false,height:56">
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="stepDown_editSoftwareAutoTask()" style="float: right;margin-right:20px"><span><%=rb.getString("XiaYiBu")%></span></a>
		</div>
	</div>
	<%-- 第二步 --%>
	<div id="step_2" class="easyui-panel" data-options="border:false,fit:true">
		<div class="easyui-layout" data-options="border:false,fit:true">
			<div region="center" data-options="border:false" style="padding:20px;" class="padT20">
				<table id="gridFiles_editSoftwareAutoTask"></table>
			</div>
			<div region="south" data-options="border:false,height:56">
				<div class="windowButtonGroup">
					<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUp_editSoftwareAutoTask()"><span><%=rb.getString("ShangYiBu")%></span></a>
					<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveSoftwareAutoTask()" ><span><%=rb.getString("QueDing")%></span></a>
					<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="closeWinEditSoftwareAutoTask()" ><span><%=rb.getString("QuXiao")%></span></a>
				</div>
			</div>
		</div>
	</div>
</div>

<script type="text/javascript">
var currVerData = new Array();
$(function() {
	$("#step_2").hide();
	// 当前任务信息
	var detail = ${detail};
	
	var destVerId = "";
	var rawMode = "";
	
 	if (detail.destVerId) {
		// 赋值已选版本
		var oriVerList = detail.originalVerList;
		if (oriVerList) {
			if (oriVerList.length != 0) {
				for (var i = 0; i < oriVerList.length; i++) {
					var row = {"value": oriVerList[i], "text": oriVerList[i]};
					currVerData.push(row);
				}
			}
		}
		
		// 赋值已选文件
		destVerId = detail.destVerId;
		
		// 赋值是否保存配置
		rawMode = detail.rawMode;
		$("#upgradeRawModeFlag").val(rawMode);
		

		var startTime = detail.startTime;
		var endTime = detail.endTime;
		
		$("#autoTaskExecuteStartTime").val(startTime);
		$("#autoTaskExecuteEndTime").val(endTime);
	} 
	
	// 声明原始software版本表格
	$("#gridOriSoftwareVer").datagrid({
		border: false,
		fitColumns: true,
        rownumbers: true,
        url: '${ctx}/cell/version/getOriSoftwareVer.action',
        striped: true,
        fit:true,
        idField: 'ver_name',
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'ver_name', width: 100, title: '<%=rb.getString("BanBen")%>'}
		]],
		onLoadSuccess: loadSucc_oriSoftwareVerDatagrid
	});

	$("#gridOriSoftwareVer").datagrid("getPager").pagination({
		layout : [ 'prev', 'manual', 'next', 'refresh' ]
	});

	// 声明已选择版本表格
	$("#gridSelectedOriSoftwareVer").datagrid({
		fit: true,
		striped: true,
		border: false,
		idField: 'value',
		fitColumns: true,
		onLoadSuccess:datagridLoadSuccess,
		columns: [[
			{field: 'ck', checkbox: true},
			{field: 'value', hidden: true},
			{field: 'text', width: 100, title: "<%=rb.getString("BanBen")%>"}
		]],
		data: currVerData
	});
	
	// 声明升级文件表格
	$("#gridFiles_editSoftwareAutoTask").datagrid({
		title: "<%=rb.getString("XuanZeShengJiWenJian")%>",
		singleSelect: true,
		rownumbers: true,
		pagination: true,
		pagePosition: 'bottom',
		url: '${ctx}/cell/version/queryfileInfosList.action?file_type=0',
		queryParams:{timeZone:timeZone},
		fitColumns: false,
		fit: true,
		idField: 'id',
		border: true,
		striped: true,
		columns: [[
			{field: 'id', hidden: true},
			{field:'file_name', title: '<%=rb.getString("WenJianMing")%>', width: 200},
			{field:'product',title: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>', width: 100},
			{field:'size',title: '<%=rb.getString("WenJianDaXiao")%>', width: 100},
			{field:'version',title: '<%=rb.getString("BanBen")%>', width: 100},
			{field:'upload_time',title: '<%=rb.getString("ShangChuanShiJian")%>', width: 140},
			{field:'uploader',title: '<%=rb.getString("ShangChuanZhe")%>', width: 100},
			{field:'desc',title: '<%=rb.getString("MiaoShu")%>', width: 100}
		]],
		onLoadSuccess: function(data) {
			$(this).datagrid("enableContextmenuAutoSize");
			if (data.rows.length != 0) {
				$(this).datagrid("selectRecord", destVerId);
			}
		}
	});
	

	
});

function loadSucc_oriSoftwareVerDatagrid(){
	$(this).datagrid("enableContextmenuAutoSize");
	//获取其他软件升级任务的所有初始版本
 	var task = $("#autoTaskList").datagrid("getData").rows;
	for (var count = 0; count < task.length; count++) {
		var taskName = task[count].name;
		if (taskName.indexOf("SoftwareZiDongShengJi") > -1 && taskName != "${softwareTaskName}") {
			var detail = task[count].detail;
			if (detail) {
				detail = eval("(" + detail + ")");
				// 赋值已选版本
				var oriVerList = detail.originalVerList;
				if (oriVerList) {
					if (oriVerList.length != 0) {
						for (var i = 0; i < oriVerList.length; i++) {
							var rowIndex = $(this).datagrid("getRowIndex", oriVerList[i]);
							if (rowIndex >= 0) {
								$(this).datagrid("deleteRow", rowIndex);
							}
						}
					}
				}
			}
		}
	}
	if (!currVerData || currVerData.length == 0) {
		$("#gridSelectedOriSoftwareVer").datagrid({
	    	data: {
	    		total: 1,
	    		rows: [{value: "", text: "<%=rb.getString("QingXuanZe")%>"}]
	    	}
	    });
	} else {
		for (var i = 0; i < currVerData.length; i++) {
			var rowIndex = $(this).datagrid("getRowIndex", currVerData[i].value);
			if (rowIndex >= 0) {
				$(this).datagrid("deleteRow", rowIndex);
    			var row = {"ver_name": currVerData[i].value};
    			delVerObjArr.push(row);
			}
		}
	}
	
}
//保存当前已选择的基站对象数组
var delVerObjArr = new Array();

<%-- 下一步  --%>
function stepDown_editSoftwareAutoTask() {
	if (validateStep1()) {
		$("#editSoftwareAutoTask #step_1").hide();
		$("#editSoftwareAutoTask #step_2").show();
		$('#editSoftwareAutoTask #step_2').panel("doLayout");		
	}
}
<%-- 上一步 --%>
function stepUp_editSoftwareAutoTask() {
	$("#editSoftwareAutoTask #step_2").hide();
	$("#editSoftwareAutoTask #step_1").show();
}

<%-- 验证第一步输入 --%>
function validateStep1() {
	// 是否已选择版本
	var selVer = $("#gridSelectedOriSoftwareVer").datagrid("getRows");
	if (selVer.length == 1 && (selVer[0]["value"].length == 0 || selVer[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("QingXuanZeYuanShiBanBen")%>");
		return false;
	}


	var startTime = $("#autoTaskExecuteStartTime").datetimebox("getValue");
    var endTime = $("#autoTaskExecuteEndTime").datetimebox("getValue");
    
  	 //开始时间不能为空
    if (startTime.length == 0) {
   	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("KaiShiShiJianBuNengKong")%>");
		return false;
    }
  
    //结束时间不能为空
    if (endTime.length == 0) {
   	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("JieShuShiJianBuNengKong")%>");
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
    
	
	return true;
}

<%-- 验证第二步输入 --%>
function validateStep2() {
	var selFile = $("#gridFiles_editSoftwareAutoTask").datagrid("getSelected");
	if (selFile) {
		return true;
	} else {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("QingXianXuanZeWenJian")%>");
		return false;
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_addUpgradeBiosTask() {
	var dl = $("#gridSelectedOriSoftwareVer");
	var selVer = dl.datagrid("getSelections");
	var selVerLength = selVer.length;
	var needDelVerArr = new Array(); //需要删除-数组
	
	if (selVerLength > 0) {
		for (var selVerCount = 0; selVerCount < selVer.length; selVerCount++) {
			if (selVer[selVerCount]["value"]) {
				var find = false;
    			//从列表中恢复
        		for(var count = 0; count < delVerObjArr.length; count++) {
        			if (selVer[selVerCount].value == delVerObjArr[count]["ver_name"]) {
        				find = true;
            			var row = {
            				"ver_name": selVer[selVerCount].value
            			}
            			$("#gridOriSoftwareVer").datagrid("appendRow",row);
            			delVerObjArr.remove(count);
            			needDelVerArr.push(selVer[selVerCount].value);
            			break;
        			}
        		}
        		if (!find) {
    				needDelVerArr.push(selVer[selVerCount].value);
    			}
    		}
		}
		
    	if (needDelVerArr.length > 0) {
    		//从已选列表中删除
    		for(var count = 0; count < needDelVerArr.length; count++) {
    			var rowIndex = $("#gridSelectedOriSoftwareVer").datagrid("getRowIndex", needDelVerArr[count]);
    			$("#gridSelectedOriSoftwareVer").datagrid("deleteRow", rowIndex);
    		}
    	}
    	var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		
		//将复选框取消选中
		$("#gridSelectedOriSoftwareVer").datagrid("uncheckAll");
	}
}
<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedOriSoftwareVer() {
	var selVer = $("#gridOriSoftwareVer").datagrid("getSelections");
	var selVerLength = selVer.length;
	if (selVerLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#gridSelectedOriSoftwareVer");
	var rows = dl.datagrid("getRows");


	for (var selVerCount = 0; selVerCount < selVerLength; selVerCount++) {
		var i = 0
		var ver_name = selVer[selVerCount]["ver_name"];
		for (; i < rows.length; i++) {
			//如果当前已经选择了该小站，跳出循环
    		if (rows[i]["value"] == ver_name) {
    			break;
    		}
    	}

		// i == rows.length表示当前没有选择该小站，需要加到右侧列表中
		if (i == rows.length) {
			var row = {value: selVer[selVerCount]["ver_name"], text: selVer[selVerCount]["ver_name"]};
   			dl.datagrid("appendRow", row);
   			var selectedVerObj = new Object();
   			selectedVerObj.ver_name = selVer[selVerCount]["ver_name"];
   			
   			delVerObjArr.push(selectedVerObj);
   			newSelectdArr.push(selectedVerObj)
		}
	}
	
	if (rows.length >= 2 && (rows[0]["value"].length == 0 || rows[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		dl.datagrid("deleteRow", 0);
	}
		
	if (newSelectdArr.length > 0) {
		//从基站列表中删除已选择的基站
		for(var count = 0; count < newSelectdArr.length; count++) {
			var rowIndex = $("#gridOriSoftwareVer").datagrid("getRowIndex", newSelectdArr[count]["ver_name"]);
			$("#gridOriSoftwareVer").datagrid("deleteRow", rowIndex);
		}
	}
	
	$("#gridOriSoftwareVer").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveSoftwareAutoTask() {
	if (!validateStep2()) {
		return;
	}
	var params = {};
	
	// 取得已选择的基站
	var verNames = "";
	var selVer = $("#gridSelectedOriSoftwareVer").datagrid("getRows");
	if (selVer.length == 1 && (selVer[0]["value"].length == 0 || selVer[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}

	for (var i = 0; i < selVer.length; i++) {
		verNames += selVer[i]["value"] + ",";
	}
	params["verNames"] = verNames;

	// 是否保留配置
	params["rawMode"] = $("#upgradeRawModeFlag").val();
	params["taskName"] = "${softwareTaskName}";
	params["timeZone"] = timeZone;

	var startTime = $("#autoTaskExecuteStartTime").datetimebox("getValue");
    var endTime = $("#autoTaskExecuteEndTime").datetimebox("getValue");
    
  	 //开始时间不能为空
    if (startTime.length == 0) {
   	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("KaiShiShiJianBuNengKong")%>");
		return false;
    }
  
    //结束时间不能为空
    if (endTime.length == 0) {
   	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("JieShuShiJianBuNengKong")%>");
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
    
	params["startTime"] = startTime;
	params["endTime"] = endTime;
	params["taskId"] = "${softwareTaskId}";
	params["timeZone"] = timeZone;
	
	// 取得选择的文件
	var selFile = $("#gridFiles_editSoftwareAutoTask").datagrid("getSelected");
	params["file_id"] = selFile["id"];
	$.post("${ctx}/cell/selfstart/saveSoftwareAutoTask.action", params, function(data) {
		if (data["success"]) {
			closeWinEditSoftwareAutoTask();
			$("#autoTaskList").datagrid("reload");
			$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("ChengGong")%>");
		} else {
			$.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
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
// 关闭窗口-编辑software自动任务
function closeWinEditSoftwareAutoTask() {
	/* $("#winEditSoftwareTask").window("close"); */
	closeDefaultWindow();
}

</script>