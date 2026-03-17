<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<div id="cpeFreqLockTask" class="easyui-panel" data-options="border:false,fit:true">
	<%-- 第一步 --%>
	<div id="step_1" class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 10px 20px;">
			<div class="easyui-panel" data-options="border:false,fit:true" style="padding: 0 15px;">
				<table class="easyui-datagrid" id="cpeFreqLockList" fit="true" fitColumns="true" 
		   				data-options="rownumbers:true,border:false,striped:true,onLoadError:datagridLoadError,toolbar:'#toolbar_GridCpe_freqLock',
		             	url:'${ctx}/cell/CPE/queryCpeFreqLockInfosList.action?cellId=${cellId}',pagination:true,pagePosition:'bottom',
		             	onBeforeLoad:beforeLoad_gridCpe_freqLock,onLoadSuccess:datagridLoadSuccess">
					<thead><tr>
					    <th data-options="field:'ck',checkbox: true"></th>
					    <th data-options="field:'CPE_CODE',hidden:true"><%=rb.getString("XiaoZhanBianMa")%></th>
						<th data-options="field:'CONNECTION_STATUS',sortable:true,fixed:true,formatter:connStatusFormatter" width="30"></th>
						<th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("CPEXuLieHao")%></th>
						<th data-options="field:'IMSI'" width=80>IMSI</th>
						<th data-options="field:'SCANMODE'" width=80><%=rb.getString("SaoMiaoFangShi")%></th>
						<th data-options="field:'FREQANDPCI'" width=80><%=rb.getString("SuoPinXinXi")%></th>
					</tr></thead>
				</table>
			</div>
		</div>
		<div region="south" data-options="border:false,height:36" style="padding: 0px 20px 10px 0px;">
			<a class="easyui-linkbutton" href="javascript:void(0)" onclick="stepDown_cpeFreqLock()" style="float: right;"><%=rb.getString("XiaYiBu")%></a>
		</div>
	</div>
	<%-- 第二步 --%>
	<div id="step_2" class="easyui-panel" data-options="border:false,fit:true">
		<div id="cpeFreqLockConfig" class="easyui-layout" data-options="border:false,fit:true">
			<div region="center" data-options="border:false" style="padding:18px 10px">
				<div class="itemDiv">
					<span style="width:80px;"><%=rb.getString("SaoMiaoFangShi")%></span>
					<select id="scanMode" name="CPE_scanMode" class="border border-box item">
						<option value="pcilock">PCI lock</option>
						<option value="fullband">Full Band</option>
					</select>
				</div>
				<div id="forPCILock" class="itemDiv">
					<span style="width:80px;"><%=rb.getString("PinLv")%></span>
					<input type="text" name="CPE_Frequency" class="border border-box item" 
						value="${earfcn}" onblur="validateRequired(event)" style="width:110px"/>
					<span>:</span>
					<span style="width:40px;"><%=rb.getString("PCIZhi")%></span>
					<input type="text" name="PCI_value" class="border border-box item" 
						value="${pci}" onblur="validateRequired(event)" style="width:108px"/>
					<a onclick="addMMEIpInputText(this)">
						<img src="${ctx}/js/jquery-easyui/themes/icons/edit_add.png"> 
					</a>
				</div>
			</div>
			<div region="south" data-options="border:false,height:36" style="padding: 0px 0px 10px 0;">
				<a class="easyui-linkbutton" href="javascript:void(0)" onclick="cancelCpeFreqLockTask()" style="float: right;margin-right: 20px;"><%=rb.getString("QuXiao")%></a>
				<a class="easyui-linkbutton" href="javascript:void(0)" onclick="saveCpeFreqLockTask()" style="float: right;margin-right: 15px;"><%=rb.getString("WanCheng")%></a>
				<a class="easyui-linkbutton" href="javascript:void(0)" onclick="stepUP_cpeFreqLockTask()" style="float: right;margin-right: 15px;"><%=rb.getString("ShangYiBu")%></a>
			</div>
		</div>
	</div>
</div>

<div id="toolbar_GridCpe_freqLock" class="admin_query_head" style="background-color:white;">
	<%-- <ul class="inputslist">
	 	<li>
			<select id = "cpeFreqLockListForSearch" name="type" class="border border-box" style="width:100px;margin-right:1px;height: 26px;"></select>
		</li> 
		<li class="serial_number input_li">
			<input name="value" type="text" class="border border-box">
		</li>
		<li>
			<a onclick="$('#gridCpe_cpeFreqLockList').datagrid('reload');" style="margin-left:15px;"
					class="easyui-linkbutton"><%=rb.getString("ChaXun")%></a>
		</li>
	</ul> --%>
	<input type="text" id="searchText_cpeFreqLock" class="border-box border"  placeholder="<%=rb.getString("QingShuRuCPEChaXunNeiRong")%>" style="margin-left: 25px;width:350px;"/>
	<a href="#" class="easyui-linkbutton" style="vertical-align: top; margin-left: 15px;" onclick="$('#gridCpe_cpeFreqLockList').datagrid('reload');"><%=rb.getString("ChaXun")%></a>
</div>

<%-- 窗口-右键设置进度条 --%>
<div id="winSettingPro" title="<%=rb.getString("CanShuPeiZhiJinDu")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:70,resizable:false,closable:false">
    <img src="${ctx}/js/jquery-easyui/themes/bootstrap/images/loading.gif"/>
    <span style="margin-left:75px;"><%=rb.getString("eNodeBZhengZaiSheZhi")%></span>
</div>

<script type="text/javascript">
var TiShi = "<%=rb.getString("TiShi")%>";
var QingXuanZeSheBei = "<%=rb.getString("QingXuanZeSheBei")%>";

$(function() {
	closeLoading();
	$("#step_2").hide();
	
	<%-- $("#gridCpe_cpeFreqLockList").datagrid({
		border: false,
		fitColumns: true,
		fit: true,
        rownumbers: true,
        url: '${ctx}/cell/CPE/queryCpeFreqLockInfosList.action?cellId=${cellId}',
        pageSize: 10,
        pageList: [10],
        striped: true,
        singleSelect: false,
        pagination: true,
        pagePosition: 'bottom',
        idField: 'CPE_CODE',
        onBeforeLoad: beforeLoad_gridCpe_freqLock,
        onLoadError: datagridLoadError,
        toolbar: '#toolbar_GridCpe_freqLock',
        columns: [[
			{field: 'ck', checkbox: true},
			{field: 'CPE_CODE', hidden: true},
			{field: 'CONNECTION_STATUS', sortable: true, fixed: true, width: 30, formatter: connStatusFormatter},
			{field: 'SERIAL_NUMBER', sortable: true, width: 100, title: '<%=rb.getString("CPEXuLieHao")%>'},
			{field: 'IMSI', sortable: true, width: 80, title: 'IMSI'},
			{field: 'SCANMODE', width: 80, title: '<%=rb.getString("SaoMiaoFangShi")%>'},
			{field: 'FREQANDPCI', width: 100, title: '<%=rb.getString("SuoPinXinXi")%>'}
		]]
	});
	
	$("#gridCpe_cpeFreqLockList").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	}); --%>
});

function beforeLoad_gridCpe_freqLock(param) {
	if ($("#searchText_cpeFreqLock").val()) {
		param.search_text = $("#searchText_cpeFreqLock").val();
	}
}

function stepDown_cpeFreqLock() {
	if (validateStep1()) {
		$("#cpeFreqLockTask #step_1").hide();
		/* $("#winCpeFreqLockConfig").window("resize", {
			height: 250,
			width: 410
		}).window("center"); */
		setDefaultWindow({
			height: 250,
			width: 410
		});
		$("#cpeFreqLockTask #step_2").show();
		$('#cpeFreqLockTask #step_2').panel("doLayout");	
	}
}

function validateStep1() {
	// 是否已选择CPE
	var selCpes = $("#cpeFreqLockList").datagrid("getSelections");
	if (selCpes.length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
		return false;
	}
	return true;
}

//添加一个MME地址输入框
function addMMEIpInputText(e) {
	//添加新的输入框
	var $cpeDiv = $("<div class='itemDiv addition'></div>");
	var $cpeSpan1 = $("<span style='width:80px;'><%=rb.getString("PinLv")%></span>");
	var $cpeInput1 = $("<input type='text' name='CPE_Frequency' class='border border-box item frequency' "
		 + "onblur='validateRequired(event)' value='' style='width:110px;margin-left:4px'></input>");
	var $cpeSpan2 = $("<span style='margin-left:3px'>:</span>");
	var $cpeSpan3 = $("<span style='width:80px;margin-left:4px'><%=rb.getString("PCIZhi")%></span>");
	var $cpeInput2 = $("<input type='text' name='PCI_value' class='border border-box item pci' "
			 + "onblur='validateRequired(event)' value='' style='width:108px;margin-left:3px'></input>");
	var $cpeButtn = $("<a onclick='removeMMEIpInputText(this)' style='margin-left:3px'>" 
			+ "<img src='${ctx}/js/jquery-easyui/themes/icons/edit_remove.png'></a>");
	
	$cpeDiv.append($cpeSpan1);
	$cpeDiv.append($cpeInput1);
	$cpeDiv.append($cpeSpan2);
	$cpeDiv.append($cpeSpan3);
	$cpeDiv.append($cpeInput2);
	$cpeDiv.append($cpeButtn);
	
	//将新增的输入框插入到plmn参数之前
	//var currentDiv = $("input[name='CPE_Frequency']").parent("div");
	var currentDiv = document.getElementById("bottom");
	$cpeDiv.insertAfter(currentDiv);
	
	//将当前输入框后面的图标改为删除图标，并重新绑定事件
	$(e).children("img").attr("src","${ctx}/js/jquery-easyui/themes/icons/edit_add.png");
	$(e).attr("onclick", "addMMEIpInputText(this)");
}

//删除选中的MME地址输入框
function removeMMEIpInputText(e) {
	$(e).parent("div").remove();
	
	var cpeDiv = $("input[name='CPE_Frequency']:first");
	var isVisible = cpeDiv.parent("div").children("span").css("visibility");
	if (isVisible == "hidden") {
		cpeDiv.parent("div").children("span").css("visibility","visible");
	}
}

function cancelCpeFreqLockTask() {
	/* $("#winCpeFreqLockConfig").window("close"); */
	closeDefaultWindow();
}

function stepUP_cpeFreqLockTask() {
	$("#cpeFreqLockTask #step_2").hide();
	/* $("#winCpeFreqLockConfig").window("resize", {
		height: 400,
		width: 600
	}).window("center"); */
	setDefaultWindow({
		height: 400,
		width: 600
	});
	$("#cpeFreqLockTask #step_1").show();
}

function saveCpeFreqLockTask() {
	var eNodeBEarfcn = "${earfcn}";
	var eNodeBPci = "${pci}";
	var scanMode = $("select[name='CPE_scanMode']").val();
	var cellCodeArr = "";
	/*<%--获取选择的小站--%>*/
    var selCpes = $("#cpeFreqLockList").datagrid("getSelections");
    if (0 == selCpes.length) {
        $.messager.alert(TiShi, QingXuanZeSheBei);
        return;
    }
    /*<%--拼接多个小站编码--%>*/
    for (var codeNum = 0; codeNum < selCpes.length; codeNum++) {
    	cellCodeArr += selCpes[codeNum]["CPE_CODE"] + ",";
    }
    
    cellCodeArr = cellCodeArr.substring(0, cellCodeArr.length - 1);
    
    var params = "";
    params["cpeCodes"] = cellCodeArr;
    params["scanMode"] = scanMode;
    
    if (scanMode == "pcilock") {
    	$("#cpeFreqLockConfig .item").each(function(){
			var paramName = $(this).attr("name");
			var value = $(this).val();
			if (paramName == "CPE_Frequency" || paramName == "PCI_value") {
				if (params[paramName] != null) {
					var cpeLockAdd = params[paramName];
					//确保不添加重复地址
					if (cpeLockAdd.indexOf(value) == -1) {
						params[paramName] =  params[paramName] + value + ","; 
					}
				} else {
					params[paramName] =  value + ","; 
				}
			}
		});
    }
    var msg = null;
    if (params["CPE_Frequency"] != eNodeBEarfcn || params["PCI_value"] != eNodeBPci) {
    	msg = "<%=rb.getString("CPESuoPinTiShi")%>";
    } else {
    	msg = "<%=rb.getString("QueDingXiaFaPeiZhi")%>";
    }
    
    $.messager.confirm("<%=rb.getString("QueRen")%>",msg , function(r) {
    	if (r) {
    		$("#winSettingPro").window("open");
    		$.post("${ctx}/cell/CPE/setParamGroupValues.action", params, function (data) {
    			$("#winSettingPro").window("close");
    	        if (!data["success"]) {
    	            $.messager.alert(TiShi, data["message"]);
    	        }
    	    }, "json");
    	}
    });
}
</script>