<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.panel-body{
		color:#000;
	}
	.upgrade_clear{
		float:right;
		display:inline-block;
		width:20px;
		height:20px;
		margin-right:5px;
		margin-top:8px;
	}
	#selectedCellDiv_uboot .datagrid-body .datagrid-cell{
		position:relative;
		cursor:pointer;
	}
	#selectedCellDiv_uboot .datagrid-view2 .datagrid-body td{
		cursor:pointer;
	}
	#selectedCellDiv_uboot td{
		border:none;
	}
	.upgrade_delete{
		visibility:hidden;
		transation: visibility 0.5s ease 0.2s;
		cursor:pointer;
		position:absolute;
		top:0px;
		width:20px;
		height:20px;
	}
	.visible .upgrade_delete {
		visibility:visible;
	}
	.upgradeContent_uboot{
		width:70%;
		margin:30px auto 0px;
		overflow:auto;
	}
	@media screen and (max-width:1280px){
		.upgradeContent_uboot{
			width:90%;
			margin:30px auto 0px;
			overflow:auto;
		}
	}
	#addTasksuccessUboot{
	background:#E9FBFF url("${ctx}/images/success.png") no-repeat 8px center;
	height:38px;
	float:left;
	margin-left:50px;
	margin-top:41px;
	color:#508D9B;
	font-size:15px;
	font-weight:bold;
	padding:0 5px 0 10px;
	line-height:38px;
	text-indent:25px;
	display:none;
}
</style>
<div class="omcPageTitleDiv">
	<ul class="omcPageTitleContainer">
		<li class="default"><%=rb.getString("XinJianShengJiRenWu")%></li>
	</ul>
</div>
<!-- 右上角关闭按钮 -->
<div class="omcTitleButton" style="top:21px">
	<span class="titleButtonText"><%=rb.getString("GuanBi")%></span>
	<span class="circleBg close_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="closeUpgradeUboot()"></span>
</div>
<div class='upgradeContent_uboot'>
	<div>
		<label style='display:block;color:#85A8BF;margin-bottom:10px;font-size:14px;'><%=rb.getString("RenWuMingCheng")%><%=rb.getString("MaoHao")%></label>
		<input onblur='taskNameBlurUboot()' value="${taskName }" id='newUpgradeTaskName_uboot' type="text" class="border border-box" style='width:350px;height:26px;'
			maxlength=100 placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>"/>
		<p style='visibility:hidden;color:#CC0000;margin-top:7px;font-size:12px;'><%=rb.getString("QingShuRuXinJianRenWuMingCheng")%></p>
	</div>
	<div style='margin-top:7px;'>
		<label style='color:#85A8BF;font-size:14px;margin-bottom:10px;'><%=rb.getString("ShengJiWenJian")%><%=rb.getString("MaoHao")%></label>
		<p style='font-size:12px;margin-top:10px;'><span><%=rb.getString("BanBenHao")%><%=rb.getString("MaoHao")%></span><span id='versionMesUboot'></span></p>
	</div>
	<div style='margin-top:30px;display:flex'>
		<div  style='display:inline-block;flex:3 3 65%;overflow-x:auto;margin-right:30px; '>
			<label style='color:#85A8BF;font-size:14px;'><%=rb.getString("JiZhanLieBiao")%><%=rb.getString("MaoHao")%></label>
			<div  style='height:433px;border:1px solid #CCE1EF;margin-top:10px;'>
				<table id="upgrade_Cell_list_uboot"></table>
			</div>
		</div>
		<div id='selectedCellDiv_uboot' style='display:inline-block;overflow-x:auto;;flex:1 1 35%'>
			<label style='color:#85A8BF;font-size:14px;'><%=rb.getString("YiXuanZeJiZhan")%><%=rb.getString("MaoHao")%></label>
			<div style='height:433px;border:1px solid #CCE1EF;margin-top:10px;'>
				<table id='upgrade_selectedCell_list_uboot'></table>
			</div>
		</div>
	</div>
	<div id='noDeviceDivUboot' style='font-size:12px;color:#CC0000;margin-top:6px;visibility:hidden'><%=rb.getString("QingXuanZeSheBei")%></div>
	<div style='margin-top:10px;'>
		<label style='display:block;margin-bottom:10px;font-size:14px;color:#85A8BF'><%=rb.getString("XuanZeZhiXingFangShi")%><%=rb.getString("MaoHao")%></label>
		<input type="radio" status="active" checked="true"  name="taskStatusUboot" onchange="setDateTimeBoxEnableUboot()" style="vertical-align:middle;margin-right:4px;"/>
		<label><%=rb.getString("LiJiZhiXing")%></label>
		<input type="radio" status="suspend"  name="taskStatusUboot" onchange="setDateTimeBoxEnableUboot()" style="margin-left:50px;vertical-align:middle;margin-right:4px;"/>
		<label><%=rb.getString("GuaQi")%></label>
		<input type="radio" id="upgrade_schedule_input_uboot" status="timingUboot" name="taskStatusUboot" onchange="setDateTimeBoxEnableUboot()" style="margin-left:50px;vertical-align:middle;margin-right:4px;"/>
		<label style="margin-right:10px;"><%=rb.getString("DingShiZhiXing")%></label>
		<input id="upgrade_schedule_timebox_uboot" class="easyui-datetimebox border-box border" style="height:26px;"
			   data-options="disabled:true,editable:false">
	</div>
	<div id='datetimeMesUboot' style='font-size:12px;color:#CC0000;margin-top:1px;visibility:hidden'><%=rb.getString("QingXuanZeShiJian")%></div>
	<div class="windowButtonGroup" style='float:left;margin-top:40px;margin-bottom:20px;'>
		<a class="easyui-linkbutton linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveNewUpgradeTaskUboot(this)"><span><%=rb.getString("QueDing")%></span></a>
		<a class="easyui-linkbutton linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="closeUpgradeUboot()"><span><%=rb.getString("QuXiao")%></span></a>
	</div>
	<div id='addTasksuccessUboot'><%=rb.getString("XinJianShengJiRenWuWanCheng")%></div>
</div>
<%-- 工具栏-基站列表 --%>
<div id="toolbar_upgrade_Cell_list_uboot" class="admin_query_head" style="background-color:white;">
	<ul class="inputslist defaultQuery" style='margin:15px 0px 0px 12px;'>
	 	<li class="select_reset">
			<select id = "upgrade_deviceList_select_uboot" name="type" class="border border-box" style="width:103px;margin-right:1px;height: 26px;"></select>
		</li>
		<li class="serial_number input_li">
			<input id='upgrade_device_input_uboot' name="value"  style="width:190px;margin-left:0px;" placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		</li>
		<li style='margin-left:10px;'>
			<b class="searchResultImgChangeStyle" onclick="$('#upgrade_Cell_list_uboot').datagrid('reload');" ></b>	
		</li>
	</ul>
</div>
<script>
var choosedGroupId = -1;//记录当前选中的设备组的id，该参数必须放在基站列表加载前
$(function(){
	var ele = $("#upgrade_schedule_timebox_uboot");
	disableSelectEarlyTime(ele);
	$('#versionMesUboot').html("${version}");
	// 声明基站列表
	$("#upgrade_Cell_list_uboot").datagrid({
		border: false,
		fitColumns: true,
		fit: false,
		width: '99%',
		height:'100%',
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
	    checkOnSelect:true,
        selectOnCheck:true,
	    onBeforeLoad: beforeLoad_upgrade_cellList_uboot,
	    onLoadError: datagridLoadError,
	    toolbar: '#toolbar_upgrade_Cell_list_uboot',
	    onLoadSuccess: loadSuccess_upgrade_cell_uboot,
	    onSelect:addSelectedCell_addTask_upgrade_uboot,
        onUnselect:deleteSelectedCell_task_upgrade_uboot,
        onSelectAll:addSelectedCell_addTask_upgrade_all_uboot,
        onUnselectAll:deleteSelectedCell_task_upgrade_all_uboot,
	    columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'serial_number', sortable: true, width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name', sortable: true, width: 100, title: '<%=rb.getString("HostName")%>'},
			{field: 'software_version', sortable: true, width: 100, title: '<%=rb.getString("BanBen")%>'}
		]]
	});
		//已经选择列表
		$("#upgrade_selectedCell_list_uboot").datagrid({
			border: false,
	        rownumbers: true,
	        striped: true,
	        singleSelect: true,
	        fit:true,
	        fitColumns:true,
	        onLoadSuccess:function(){
	        	 $("#selectedCellDiv_uboot .datagrid-view2 .datagrid-body").bind("mouseover",function(){
	        		var td = event.target
	        		if(event.target.tagName=='SPAN'||event.target.tagName=='TD'||(event.target.tagName=='DIV'&& $(event.target).hasClass('datagrid-cell'))){
	        			var td = event.target;
	        			if(event.target.tagName=='TD'){
	        				
	        			}else{
	        				td = $(event.target).parents('td')[0];
	        			}
	        			td.classList.add('visible');
	        			var grid = $("#upgrade_selectedCell_list_uboot");
	        			var cwidth = grid.datagrid('getColumnOption','text').width;
	        			var pwidth = $("#selectedCellDiv_uboot .datagrid-view2").width();
	        			$($(td).find("span")).css("left",pwidth-50);
	        		}
	        	}).bind("mouseout",function(){
	        		event.target.classList.remove('visible');
	        	});
	        	$("#selectedCellDiv_uboot").bind("mouseout",function(){
	        		$('#selectedCellDiv_uboot .visible').removeClass('visible');
	        	});
	        },
	        data:[],
	        columns: [[
				{field: 'value', hidden: true},
				{field: 'text',width:300,title:titleClearUboot,formatter:deleteSelectRowUboot}
			]]
		});
		$(window).resize();
		$("#upgrade_deviceList_select_uboot").combobox({
	    	url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action',
	        width: 100,
	        panelWidth: 150,
	        panelHeight: 200,
	        valueField: 'id',
	        textField: 'group_name',
	        onSelect: chooseDeviceGroup_upgrade_uboot
	 	});
		$("#upgrade_device_input_uboot").bind("keyup", function(e){
			if (e.keyCode == 13){
				$('#upgrade_Cell_list_uboot').datagrid('load');
			}
		});
})

$("#upgrade_deviceList_select_uboot").combobox('setValues',['<%=rb.getString("SheBeiZu")%>']);
/* 设备组选择 */
function chooseDeviceGroup_upgrade_uboot(data){
	choosedGroupId = data.id;
    $("#upgrade_Cell_list_uboot").datagrid("reload");
}
//加载前事件-基站列表
function beforeLoad_upgrade_cellList_uboot(param) {
	//查找当前选中的
	if(choosedGroupId > 0){
		param["group_id"] = choosedGroupId;
	}
	var search_text = $("#toolbar_upgrade_Cell_list_uboot .inputslist li.serial_number>input").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	// 兼容intel 和 qc
	param["isShowSlave"] = false;
	
	$("#upgrade_Cell_list_uboot").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});

}
<%-- 基站列表加载完成事件--%>
function loadSuccess_upgrade_cell_uboot() {
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
}
<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setDateTimeBoxEnableUboot() {
	if (document.getElementById("upgrade_schedule_input_uboot").checked) {
		$("#upgrade_schedule_timebox_uboot").datetimebox("enable");
	} else {
		$("#upgrade_schedule_timebox_uboot").datetimebox("disable");
		$("#datetimeMesUboot").css("visibility","hidden");
	}
}
<%-- 基站列表列表选择事件 --%>
function addSelectedCell_addTask_upgrade_uboot(index,row) {
	var grid = $("#upgrade_selectedCell_list_uboot");
	var cwidth = grid.datagrid('getColumnOption','text').width;
	var pwidth = $("#selectedCellDiv_uboot .datagrid-view2").width();
	$("#noDeviceDivUboot").css("visibility","hidden");
	var row = {
			value:row["small_cell_code"],
			text:row["serial_number"]+"("+row["host_name"]+")",
			groupId:choosedGroupId
	}
	grid.datagrid("appendRow", row);
	var cData = grid.datagrid("getRows");
	if(pwidth - cwidth >10){
		var cl = $("#upgrade_selectedCell_list_uboot").datagrid('getColumnOption','text');
		cl.width = pwidth;
		$("#upgrade_selectedCell_list_uboot").datagrid({fitColumns:true,data:cData});
	}else{
		$("#upgrade_selectedCell_list_uboot").datagrid({fitColumns:false,data:cData});
	}
	
}
function deleteSelectedCell_task_upgrade_uboot(index,row){
	var small_cell_code = row["small_cell_code"];
	var allRows = $("#upgrade_selectedCell_list_uboot").datagrid("getRows");
	allRows.map(function(item,index){
		if(item.value ==small_cell_code ){
			$("#upgrade_selectedCell_list_uboot").datagrid("deleteRow",index);
		}
	})
	$("#upgrade_selectedCell_list_uboot").datagrid("autoSizeColumn","text");
}
function addSelectedCell_addTask_upgrade_all_uboot(){
	var selCell = $("#upgrade_Cell_list_uboot").datagrid("getSelections");
	var del = $("#upgrade_selectedCell_list_uboot").datagrid("getRows");
	var selArr = [];
	del.map(function(item,index){
		selArr.push(item.value);
	})
	selCell.map(function(item,index){
		if(selArr.indexOf(item["small_cell_code"])!=-1){
			return;
		}else{
			var row = {
					value:item["small_cell_code"],
					text:item["serial_number"]+"("+item["host_name"]+")",
					groupId:choosedGroupId
			}
			$("#upgrade_selectedCell_list_uboot").datagrid("appendRow", row);
		}
	})
}
function deleteSelectedCell_task_upgrade_all_uboot(){
	$("#upgrade_selectedCell_list_uboot").datagrid("loadData", {total:0,rows:[]});
}
function titleClearUboot(){
	var span = $('<p onclick="clearAllSelectUboot()" style="position:absolute;top:0px;left:75%;line-height:38px;cursor:pointer;"><span style="float:right;color:#1DA3FC;margin-right:10px;"><%=rb.getString("QingKong")%></span><span class="upgrade_clear"></span></p>');
	var title = $(this);
	title.after(span);
	return "<%=rb.getString("XiaoZhanBianMa")%><%=rb.getString("ZuoKuoHao")%><%=rb.getString("HostName")%><%=rb.getString("YouKuoHao")%>"
}
function clearAllSelectUboot(){
	$("#upgrade_selectedCell_list_uboot").datagrid("loadData", {total:0,rows:[]});
	$("#upgrade_Cell_list_uboot").datagrid("clearSelections");
}
function deleteSelectRowUboot(value,rowData,rowIndex){
	var delStr = "<span onclick='deleteRowUboot(\""+rowData.value+"\")' class='upgrade_delete'></span>"
	return value+delStr;
}
function deleteRowUboot(value){
	var grid = $.data($("#upgrade_Cell_list_uboot")[0],'datagrid');
	grid.selectedRows = grid.selectedRows.filter(function(item){
		return item["small_cell_code"]!=value;
	})
	grid.checkedRows = grid.checkedRows.filter(function(item){
		return item["small_cell_code"]!=value;
	})
	var rows = $("#upgrade_selectedCell_list_uboot").datagrid("getRows");
	rows.map(function(item,index){
		if(value == item.value){
			$("#upgrade_selectedCell_list_uboot").datagrid("deleteRow",index);
		}
	})
	var allRows = $("#upgrade_Cell_list_uboot").datagrid("getRows");
	allRows.map(function(item,index){
		if(value == item["small_cell_code"]){
			 $("#upgrade_Cell_list_uboot").datagrid("unselectRow",index);
		}
	})
}
function closeUpgradeUboot(){
	$("#upgrade_container").slideUp(500,function(){
		$("#upgrade_container").html("");
	});
}
function saveNewUpgradeTaskUboot(ele){
	taskNameBlurUboot();
	if(isRightName){
		var selCells = $("#upgrade_selectedCell_list_uboot").datagrid("getRows");
		if (selCells.length == 0) {
			$("#noDeviceDivUboot").css("visibility","visible");
			return false;
		}
		var timebox = $("#upgrade_schedule_timebox_uboot").datetimebox("getValue");
		var isCheck = $("#upgrade_schedule_input_uboot")[0].checked;
		if(isCheck&&timebox==""){
			$("#datetimeMesUboot").css("visibility","visible");
			return false;
		}else{
			$("#datetimeMesUboot").css("visibility","hidden");
		}
		var params = {};
		params["timeZone"] = timeZone;
		params["taskName"] = $("#newUpgradeTaskName_uboot").val();
		
		// 取得已选择的基站
		var cellCodes = "";
		var selCells = $("#upgrade_selectedCell_list_uboot").datagrid("getRows");

		for (var i = 0; i < selCells.length; i++) {
			cellCodes += selCells[i]["value"] + ",";
		}
		params["cellCodes"] = cellCodes;
		// 取得任务状态
		var radios = document.getElementsByName("taskStatusUboot");
		for (var i = 0; i < radios.length; i++) {
			if (radios[i].checked == true) {
				params["status"] = $(radios[i]).attr("status");
			}
		}
		// 如果是定时执行的，取得时间
		if (params["status"] == "timingUboot") {
			params["time"] = $("#upgrade_schedule_timebox_uboot").datetimebox("getValue");
		}
		// 取得选择的文件
		params["fileId"] = ${versionId};
		params["productValue"] = "${productValue}";
		params["taskType"] = "3";
		//已经点击了确认按钮，禁用“确认按钮”，以防止服务器响应慢而导致重复点击按钮
		/* $(ele).linkbutton('disable'); */
		$.post("${ctx}/task/upgrade/addTask.action", params, function(data) {
			if (data["success"]) {
				$('#addTasksuccessUboot').fadeIn(300,function(){
					var  time = setTimeout(function(){
						closeUpgradeUboot();
						var thisVersionId = "${versionId}";
						fileDetail.map(function(item,index){
							if(item.versionId == thisVersionId){
								var param = {
										upgrade_type:item.upgradeType,
										version:item.version
								}
								$.post("${ctx}/system/sysuser/goCloseUpgradePrompt.action", param, function(data){
						        }, "json");
								fileDetail.splice(index,1)
							}
						})
						$(".upgradeNumber").html(fileDetail.length);
						$(".upgrade_file_title").map(function(index,item){
							if($(item).attr("versionId") == thisVersionId){
								$(item).remove();
								if(fileDetail.length == 0){
									$(".newFileInfoAlert").hide();
								}
							}
						})
					},1000);
				})
				
				/* goMenuPage('1004',false,function(){
					$('#omcLogTabsDiv span[tabtit=ubootUpgradeDiv]').addClass('active').click();
					eventBus.$emit('to-task-view');
				}); */
				try{
					eventAllBus.$emit("gomenupage","1004","","1004",false,function(){
						$('#omcLogTabsDiv span[tabtit=ubootUpgradeDiv]').addClass('active').click();
						eventBus.$emit('to-task-view');
					});
				}catch(e){}
			} else {
				/* $(ele).linkbutton('enable'); */
				$.messager.alert(TiShi, data["message"]);
			}
		}, "json");
	}
	
	
}
var isRightName = true;
function taskNameBlurUboot(){
	if($("#newUpgradeTaskName_uboot").val() == ""){
		$("#newUpgradeTaskName_uboot").next().html("<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>");
		$("#newUpgradeTaskName_uboot").css("border","1px solid #CC0000");
		$("#newUpgradeTaskName_uboot").next().css("visibility","visible");
		isRightName = false
	}else{
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/upgrade/taskNameExist.action", 
			data: {"taskName": $("#newUpgradeTaskName_uboot").val(),"taskType":"0"},
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
			$("#newUpgradeTaskName_uboot").next().html("<%=rb.getString("RenWuMingChengYiCunZai")%>");
			$("#newUpgradeTaskName_uboot").css("border","1px solid #CC0000");
			$("#newUpgradeTaskName_uboot").next().css("visibility","visible");
			isRightName = false;
		}else{
			$("#newUpgradeTaskName_uboot").next().html("<%=rb.getString("RenWuMingChengYiCunZai")%>");
			$("#newUpgradeTaskName_uboot").css("border","1px solid #c9d1d6");
			$("#newUpgradeTaskName_uboot").next().css("visibility","hidden");
			isRightName = true;
		}
	}
}

</script>