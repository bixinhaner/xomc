<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 编辑Uboot任务 --%>
<div id="editUbootAutoTask" class="easyui-panel" data-options="border:false,fit:true">
	<%-- 第一步 --%>
	<div id="step_1" class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 20px;">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="west" class="padT20" data-options="width:370,border:true,collapsible:false" title="<%=rb.getString("ChuShiBanBen")%>">
					<table id="gridOriUbootVer"></table>
				</div>
				<div region="center" data-options="border:false,onResize:setArrowMargin">
					<a id="arrow_right_addUpgradeBiosTask" href="javascript:void(0);" class="arrow_right" onclick="addSelectedOriUbootVer()"></a>
					<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_addUpgradeBiosTask()"></a>
				</div>
				<div region="east" class="padT20" data-options="width:370,border:true,collapsible:false,title:'<%=rb.getString("YiXuanZeBanBen")%>'">
					<%-- 已选择 --%>
			    	<table class="easyui-datagrid" id="gridSelectedOriUbootVer"></table>
				</div>
			</div>
		</div>
		<div region="south" data-options="border:false,height:56" style="padding: 10px 20px 20px;">
			<a class="easyui-linkbutton" href="javascript:void(0)" onclick="stepDown_editUbootAutoTask()" style="float: right;"><%=rb.getString("XiaYiBu")%></a>
		</div>
	</div>
	<%-- 第二步 --%>
	<div id="step_2" class="easyui-panel" data-options="border:false,fit:true">
		<div class="easyui-layout" data-options="border:false,fit:true">
			<div region="center" data-options="border:false" style="padding:20px;" class="padT20">
				<table id="gridFiles_editUbootAutoTask"></table>
			</div>
			<div region="south" data-options="border:false,height:56" style="padding: 10px 20px 20px;">
				<div class="windowButtonGroup">
				<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="stepUp_editUbootAutoTask()" ><span><%=rb.getString("ShangYiBu")%></span></a>
				<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveUbootAutoTask()" ><span><%=rb.getString("QueDing")%></span></a>
				<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="closeWinEditUbootAutoTask()" ><span><%=rb.getString("QuXiao")%></span></a>
				
				
				</div>
				
			</div>
		</div>
	</div>
</div>

<script type="text/javascript">
$(function() {
	$("#step_2").hide();
	/* var detail = $("#winEditUbootTask").attr("detail"); */
	var detail = $(winDefaultSelector).attr("detail");
	var currVerData = new Array();
	var destVerId = "";
	if (detail) {
		detail = eval("(" + detail + ")");
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
	}
	
	// 声明原始uboot版本表格
	$("#gridOriUbootVer").datagrid({
		border: false,
		fitColumns: true,
        rownumbers: true,
        url: '${ctx}/cell/version/getOriUbootVer.action',
        striped: true,
        fit:true,
        idField: 'ver_name',
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'ver_name', width: 100, title: '<%=rb.getString("BanBen")%>'}
		]],
		onLoadSuccess: function () {
			$(this).datagrid("enableContextmenuAutoSize");
			if (!currVerData || currVerData.length == 0) {
				$("#gridSelectedOriUbootVer").datagrid({
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
	});

	$("#gridOriUbootVer").datagrid("getPager").pagination({
		layout : [ 'prev', 'manual', 'next', 'refresh' ]
	});

	// 声明已选择版本表格
	$("#gridSelectedOriUbootVer").datagrid({
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
	$("#gridFiles_editUbootAutoTask").datagrid({
		title: "<%=rb.getString("XuanZeShengJiWenJian")%>",
		singleSelect: true,
		rownumbers: true,
		pagination: true,
		pagePosition: 'bottom',
		url: '${ctx}/cell/version/queryfileInfosList.action?file_type=2',
		queryParams:{timeZone:timeZone},
		fitColumns: false,
		fit: true,
		idField: 'id',
		border: true,
		striped: true,
		columns: [[
			{field: 'id', hidden: true},
			{field:'file_name', title: '<%=rb.getString("WenJianMing")%>', width: 200},
			{field:'version',title: '<%=rb.getString("BanBen")%>', width: 100},
			{field:'manufacturer', title: '<%=rb.getString("ZhiZaoShang")%>', width: 80},
			{field:'product',title: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>', width: 100},
			{field:'size',title: '<%=rb.getString("WenJianDaXiao")%>', width: 100},
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

//保存当前已选择的基站对象数组
var delVerObjArr = new Array();

<%-- 下一步  --%>
function stepDown_editUbootAutoTask() {
	if (validateStep1()) {
		$("#editUbootAutoTask #step_1").hide();
		$("#editUbootAutoTask #step_2").show();
		$('#editUbootAutoTask #step_2').panel("doLayout");		
	}
}
<%-- 上一步 --%>
function stepUp_editUbootAutoTask() {
	$("#editUbootAutoTask #step_2").hide();
	$("#editUbootAutoTask #step_1").show();
}

<%-- 验证第一步输入 --%>
function validateStep1() {
	// 是否已选择版本
	var selVer = $("#gridSelectedOriUbootVer").datagrid("getRows");
	if (selVer.length == 1 && (selVer[0]["value"].length == 0 || selVer[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("QingXuanZeYuanShiBanBen")%>");
		return false;
	}

	return true;
}

<%-- 验证第二步输入 --%>
function validateStep2() {
	var selFile = $("#gridFiles_editUbootAutoTask").datagrid("getSelected");
	if (selFile) {
		return true;
	} else {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("QingXianXuanZeWenJian")%>");
		return false;
	}
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable() {
	if (document.getElementById("timing_addUpgradeBiosTask").checked) {
		$("#exeTime_addUpgradeBiosTask").datetimebox("enable");
	} else {
		$("#exeTime_addUpgradeBiosTask").datetimebox("disable");
	}
}

<%-- 删除已选中的小站-向左键头的点击事件 --%>
function delSelectedCell_addUpgradeBiosTask() {
	var dl = $("#gridSelectedOriUbootVer");
	var selVer = dl.datagrid("getSelections");
	var selVerLength = selVer.length;
	var needDelVerArr = new Array(); //需要删除-数组
	
	if (selVerLength > 0) {
		for (var selVerCount = 0; selVerCount < selVer.length; selVerCount++) {
			if (selVer[selVerCount]["value"]) {
    			//从列表中恢复
    			var find = false;
        		for(var count = 0; count < delVerObjArr.length; count++) {
        			if (selVer[selVerCount].value == delVerObjArr[count]["ver_name"]) {
            			var row = {
            				"ver_name": selVer[selVerCount].value
            			}
            			$("#gridOriUbootVer").datagrid("appendRow",row);
            			delVerObjArr.remove(count);
            			needDelVerArr.push(selVer[selVerCount].value);
            			find = true;
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
    			var rowIndex = $("#gridSelectedOriUbootVer").datagrid("getRowIndex", needDelVerArr[count]);
    			$("#gridSelectedOriUbootVer").datagrid("deleteRow", rowIndex);
    		}
    	}
    	var rows = dl.datagrid("getRows");
		if (rows.length == 0) {
			var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
			dl.datagrid("insertRow", row);
		}
		
		//将复选框取消选中
		$("#gridSelectedOriUbootVer").datagrid("uncheckAll");
	}
}
<%-- 选择小站-向右键头的点击事件 --%>
function addSelectedOriUbootVer() {
	var selVer = $("#gridOriUbootVer").datagrid("getSelections");
	var selVerLength = selVer.length;
	if (selVerLength == 0) {
		return;
	}
	
	var newSelectdArr = new Array();
	var dl = $("#gridSelectedOriUbootVer");
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
			var rowIndex = $("#gridOriUbootVer").datagrid("getRowIndex", newSelectdArr[count]["ver_name"]);
			$("#gridOriUbootVer").datagrid("deleteRow", rowIndex);
		}
	}
	
	$("#gridOriUbootVer").datagrid("uncheckAll");
}

<%-- 保存新建的任务 --%>
function saveUbootAutoTask() {
	if (!validateStep2()) {
		return;
	}
	var params = {};
	
	// 取得已选择的基站
	var verNames = "";
	var selVer = $("#gridSelectedOriUbootVer").datagrid("getRows");
	if (selVer.length == 1 && (selVer[0]["value"].length == 0 || selVer[0]["text"] == "<%=rb.getString("QingXuanZe")%>")) {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}

	for (var i = 0; i < selVer.length; i++) {
		verNames += selVer[i]["value"] + ",";
	}
	params["verNames"] = verNames;

	// 取得选择的文件
	var selFile = $("#gridFiles_editUbootAutoTask").datagrid("getSelected");
	params["file_id"] = selFile["id"];
	$.post("${ctx}/cell/selfstart/saveUbootAutoTask.action", params, function(data) {
		if (data["success"]) {
			closeWinEditUbootAutoTask();
			$("#autoTaskList").datagrid("reload");
			$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("ChengGong")%>");
		} else {
			$.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
		}
	}, "json");
}

// 关闭窗口-编辑uboot自动任务
function closeWinEditUbootAutoTask() {
	/* $("#winEditUbootTask").window("close"); */
	closeDefaultWindow();
}

</script>