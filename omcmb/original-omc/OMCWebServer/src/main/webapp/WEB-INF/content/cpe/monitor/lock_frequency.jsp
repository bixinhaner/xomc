<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<div id="cpeFreqLockConfigSingle" class="easyui-layout" data-options="border:false,fit:true">
	<div region="center" data-options="border:false" style="padding:20px">
		<div class="itemDiv forPCILock" >
			<span style="width:110px; color:red;"><%=rb.getString("DangQianPinDian")%></span>
			<span id="freqForAttention" style="width:40px; color:red">${freqForAttention}</span>
			<span style="color:red;">, </span>
			<span style="width:100px; color:red;"><%=rb.getString("DangQianPci")%></span>
			<span id="pciForAttention" style="width:40px; margin-left:10px; color:red;">${pciForAttention}</span>
		</div>
		<!-- frequency 提示消息 -->
		<div class="itemDiv forFrequency" >
			<span style="width:110px; color:red;"><%=rb.getString("DangQianPinDian")%></span>
			<span id="singlefreqForAttention" style="width:40px; color:red">${freqForAttention}</span>
		</div>
		
		<div class="itemDiv">
			<span style="width:80px;"><%=rb.getString("SaoMiaoFangShi")%></span>
			<select id="scanMode" name="CPE_scanMode" class="border border-box item" data-options="editable:false" style="height:26px;">
				<option value="fullband">Full Band</option>
				<option id="cpeFreqTypeOpt" value="freqpreferred">Band/Frequency Preferred</option>
				<option value="pcilock">PCI lock</option>
			</select>
		</div>
		<div class="itemDiv forPCILock">
			<span style="width:80px;"><%=rb.getString("PinDian")%></span>
			<input type="text" name="CPE_Frequency" class="border border-box item" 
				value="" onblur="validateRequired(event)" style="width:110px" title="0 - 65535" id="firstFreq"/>
			<span>:</span>
			<span style="width:40px;"><%=rb.getString("PCIZhi")%></span>
			<input type="text" name="PCI_value" class="border border-box item" 
				value="" onblur="validateRequired(event)" style="width:108px" title="0 - 503" id="firstPCI"/>
			<a onclick="addPCILockInputText(this)">
				<img src="${ctx}/js/jquery-easyui/themes/icons/edit_add.png"> 
			</a>
		</div>
				<!-- 选择frequency 对应的选项 -->
			<div class="itemDiv forFrequency">
				<span style="width:80px;"><%=rb.getString("PinDian")%></span>
				<input type="text" name="CPE_singleFrequ" class="border border-box itemBand" 
					value="" onblur="validateRequired(event)" style="width:110px" title="0 - 65535" id="singleFreq"/>
				<a onclick="addFrequencyLockInputText(this)">
					<img src="${ctx}/js/jquery-easyui/themes/icons/edit_add.png"> 
				</a>
			</div>
			
		<div id="bottom"></div>
	</div>
	<div region="south" data-options="border:false,height:58" style="padding:0px 20px 20px;">
		<div class="windowButtonGroup">		
			<a class="linkbutton linkbutton_trend" onclick="cpeFreqLockCommit()"><span><%=rb.getString("QueDing")%></span></a>
			<a class="linkbutton linkbutton_nowanna" onclick="closeCpeFreqLockWin()"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
	</div>
</div>

<%-- 窗口-右键设置进度条 --%>
<div id="winSettingProFreLock" title="<%=rb.getString("CanShuPeiZhiJinDu")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:70,resizable:false,closable:false">
    <img src="${ctx}/js/jquery-easyui/themes/bootstrap/images/loading.gif"/>
    <span style="margin-left:75px;"><%=rb.getString("CpeZhengZaiSheZhi")%></span>
   
</div>

<script type="text/javascript">
var frequency = "${frequency}";
var pci = "${PCI}";
var PciLockCount = 1;
var frequencyCount = 1;
var cpeFreqType = "${cpeFreqType}";
$(function(){
	if(cpeFreqType=="ODUZM" || cpeFreqType=="BAIYI"){
		$("#cpeFreqTypeOpt").show();
	}else{
		$("#cpeFreqTypeOpt").hide();
	} 
	var scanMode = "${scanMode}";
	if (!scanMode) {
		scanMode = $("select[name='CPE_scanMode']").val();
	}
	if (scanMode == "fullband") {
		$(".forPCILock").hide();
		$(".forFrequency").hide();
	}else if(scanMode == "freqpreferred"){
		$("select[name='CPE_scanMode']").val(scanMode);
		$(".forPCILock").hide();
		var arrayFreq = frequency.split(',');
		for (var i = 0; i < arrayFreq.length; i++) {
			if (i == 0) {
				$("input[name='CPE_singleFrequ']").val(arrayFreq[i]);
				
			} else {
				addFrequencyLockInputText($(".addition a"));
				$(".addition").removeClass("addition");
				$(".frequencyBand").val(arrayFreq[i]);
				$(".frequencyBand").removeClass("frequencyBand");
			}
		}
	}else {
		/* if(frequency || pci){
			if("" != frequency || "" != pci){
				$("#firstFreq").prop("disabled","disabled");
				$("#firstPCI").prop("disabled","disabled");
			}
			if("" == frequency && "" == pci){
				$("#firstFreq").prop("disabled","");
				$("#firstPCI").prop("disabled","");
			}
		} */
		$("select[name='CPE_scanMode']").val(scanMode);
		$(".forFrequency").hide();
		if (frequency.indexOf(",") > -1) {
			var arrayFreq = frequency.split(',');
			var arrayPci = pci.split(',');
			for (var i = 0; i < arrayFreq.length; i++) {
				if (i == 0) {
					$("input[name='CPE_Frequency']").val(arrayFreq[i]);
					$("input[name='PCI_value']").val(arrayPci[i]);
				} else {
					addPCILockInputText($(".addition a"));
					$(".addition").removeClass("addition");
					$(".frequency").val(arrayFreq[i]);
					$(".frequency").removeClass("frequency");
					$(".pci").val(arrayPci[i]);
					$(".pci").removeClass("pci");
				}
			}
		} else {
			$("input[name='CPE_Frequency']").val(frequency);
			$("input[name='PCI_value']").val(pci);
		}
	}
	$("#scanMode").change(function() {
		var scanMode = $("select[name='CPE_scanMode']").val();
		if (scanMode == "pcilock") {
			$(".forPCILock").show();
			$(".forFrequency").hide();
			/* if(frequency || pci){
				if("" != frequency || "" != pci){
					$("#firstFreq").prop("disabled","disabled");
					$("#firstPCI").prop("disabled","disabled");
					$("#firstFreq").val(freq);
					$("#firstPCI").val(pci_first);
				}
				if("" == frequency && "" == pci){
					$("#firstFreq").prop("disabled","");
					$("#firstPCI").prop("disabled","");
				}
			} */
		} else if(scanMode == "freqpreferred"){
			$(".forPCILock").hide();
			$(".forPCILock:first").hide();
			$(".forFrequency").show();
		}else {
			$(".forPCILock").hide();
			$(".forFrequency").hide();
		}
	});
});

function cpeFreqLockCommit() {
	var freqArr = new Array();//记录频点重复次数
	var pciArr = new Array();//记录PCI重复次数
	var cpeVersion = null;//当前CPE软件版本
	var selCpe = cpevm.selectedRow;
	var scanMode = $("select[name='CPE_scanMode']").val();
	$.post("${ctx}/cell/cpeinfos/getCpeConnStatus.action", {"small_cell_code": selCpe["CPE_CODE"]}, function(data){
		// 基站未连接，不允许修改参数
		if(data["connStatus"] == "false") {
			$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("CpeWeiLianJie")%>");
			return;
		} else {
			//先判断一下当前CPE版本，如果版本为：MT-22151-1.2.2-R5-Standard,则12支持锁频配置，那么点击错频，则不下发配置，改为弹出相应提示
			$.ajax({
				type: "post",
				url: "${ctx}/cell/CPE/queryCpeVersion.action", 
				data: {cpeCode:selCpe["CPE_CODE"]},
				async: false,
				dataType: 'json',
				success: function(data) {
					cpeVersion = data["cpeVersion"];
				}
			});
			if (cpeVersion == "MT-22151-1.2.2-R5-Standard") {
				$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("CpeFreqLockTiShi")%>");
				return;
			}
			
			var params = {};
			params["cpeCode"] = selCpe["CPE_CODE"];
			params["scanMode"] = scanMode;
			
			if (scanMode == "pcilock") {
				$("#cpeFreqLockConfigSingle .item").each(function(){
					var paramName = $(this).attr("name");
					var value = $(this).val();
					if (paramName == "CPE_Frequency") {
						freqArr.push(value);
						if (!params["CPE_Frequency"]) {
							params["CPE_Frequency"] = value + ",";
						} else {
							params["CPE_Frequency"] = params["CPE_Frequency"] + value + ",";
						}
					} else if (paramName == "PCI_value") {
						pciArr.push(value);
						if (!params["PCI_value"]) {
							params["PCI_value"] = value + ",";
						} else {
							params["PCI_value"] = params["PCI_value"] + value + ",";
						}
					}
				});
				
				
				if(params["CPE_Frequency"].replace(/(^\s*)|(\s*$)/g, '').length == 1){
					$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PinDianWeiKong")%>");
					return;
				}
				
				if(params["PCI_value"].replace(/(^\s*)|(\s*$)/g, '').length == 1){
					$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIWeiKong")%>");
					return;
				}
			}else if(scanMode == "freqpreferred"){
				//itemBand
				$("#cpeFreqLockConfigSingle .itemBand").each(function(){
					var paramName = $(this).attr("name");
					var value = $(this).val();
					if (paramName == "CPE_singleFrequ") {
						freqArr.push(value);
						if (!params["CPE_Frequency"]) {
							params["CPE_Frequency"] = value + ",";
						} else {
							params["CPE_Frequency"] = params["CPE_Frequency"] + value + ",";
						}
					}
					
				});
				params["CPE_Frequency"] = params["CPE_Frequency"].substring(0,params["CPE_Frequency"].length-1); 
				<%-- if(params["CPE_Frequency"].replace(/(^\s*)|(\s*$)/g, '').length == 1){
					$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PinDianWeiKong")%>");
					return;
				}--%>
				var paramCpe_freArr = params["CPE_Frequency"].split(",");
				for(var i = 0;i<paramCpe_freArr.length;i++){
					if(paramCpe_freArr[i] == ""){
						$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PinDianWeiKong")%>");
						return;
					}
				}
			}
			for (var i = 0; i < freqArr.length; i++) {
				var form = freqArr[i] + pciArr[i];
				for (var j = 0; j < freqArr.length; j++) {
					if (j == i) {
						continue;
					}
					
					var latter = freqArr[j] + pciArr[j];
					if(freqArr[j] == ""){
						$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PinDianWeiKong")%>");
						return;
					}
					if(pciArr[j] == ""){
						$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIWeiKong")%>");
						return;
					}
					if (form == latter) {
						$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("YouChongFuShuJu")%>");
						return;
					}
				}
			}
			
			var earfcn = /^(0|[1-9][0-9]*)$/;
			for (var i = 0; i < freqArr.length; i++) {
				if (!earfcn.test(freqArr[i])) {
					$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PinDianGeShiCuoWu")%>");
					return;
				}
				if (parseInt(freqArr[i]) < 0 || parseInt(freqArr[i]) > 65535) {
					$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PinDianChaoChuFanWei")%>");
					return;
				}
			}
			
			var pci = /^(0|[1-9][0-9]*)$/;
			for (var i = 0; i < pciArr.length; i++) {
				if (!pci.test(pciArr[i])) {
					$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIGeShiCuoWu")%>");
					return;
				}
				if (parseInt(pciArr[i]) < 0 || parseInt(pciArr[i]) > 503) {
					$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIChaoChuFanWei")%>");
					return;
				}
			}
			
			$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingSuoPin")%>" , function (r) {
				if (r) {
					$("#winSettingProFreLock").window("open");
					$.post("${ctx}/cell/CPE/updateCPEPciLockParams.action", params, function(data){
						$("#winSettingProFreLock").window("close");
						if (data["success"]) {
							$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PeiZhiChengGong")%>");
							/* $("#winCpeFreqLock").window("close"); */
							closeDefaultWindow();
							cpevm.refreshList();
						} else {
							/* $("#winCpeFreqLock").window("close"); */
							closeDefaultWindow();
							$.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
						}
					}, "json");
				}
			}).addClass('normalConfirm');
		 } 
	},"json");
}

//添加一个MME地址输入框
function addPCILockInputText(e) {
	if (PciLockCount == 10) {
		return;
	}
	//添加新的输入框
	var $cpeDiv = $("<div class='itemDiv addition forPCILock'></div>");
	var $cpeSpan1 = $("<span style='width:80px;'><%=rb.getString("PinDian")%></span>");
	var $cpeInput1 = $("<input type='text' name='CPE_Frequency' class='border border-box item frequency' "
		           + "onblur='validateRequired(event)' value='' style='width:110px;margin-left:4px' title='0 - 65535'></input>");
	var $cpeSpan2 = $("<span style='margin-left:3px'>:</span>");
	var $cpeSpan3 = $("<span style='width:80px;margin-left:4px'><%=rb.getString("PCIZhi")%></span>");
	var $cpeInput2 = $("<input type='text' name='PCI_value' class='border border-box item pci' "
			       + "onblur='validateRequired(event)' value='' style='width:108px;margin-left:3px' title='0 - 503'></input>");
	var $cpeButtn = $("<a onclick='removePCILockInputText(this)' style='margin-left:3px'>" 
			      + "<img src='${ctx}/js/jquery-easyui/themes/icons/edit_remove.png'></a>");
	
	$cpeDiv.append($cpeSpan1);
	$cpeDiv.append($cpeInput1);
	$cpeDiv.append($cpeSpan2);
	$cpeDiv.append($cpeSpan3);
	$cpeDiv.append($cpeInput2);
	$cpeDiv.append($cpeButtn);
	
	//将新增的输入框插入到plmn参数之前
	//var currentDiv = $("input[name='CPE_Frequency']").parent("div");
	var currentDiv = $(".forPCILock:last");
	$cpeDiv.insertAfter(currentDiv);
	
	//将当前输入框后面的图标改为删除图标，并重新绑定事件
	$(e).children("img").attr("src","${ctx}/js/jquery-easyui/themes/icons/edit_add.png");
	$(e).attr("onclick", "addPCILockInputText(this)");
	//计数，最多锁频暂时定位6组
	PciLockCount ++;
}

//添加一个单独的singlefrequency地址输入框
function addFrequencyLockInputText(e) {
	if (frequencyCount == 10) {
		return;
	}
	//添加新的输入框
	var $cpeDiv = $("<div class='itemDiv addition forFrequency'></div>");
	var $cpeSpan1 = $("<span style='width:80px;'><%=rb.getString("PinDian")%></span>");
	var $cpeInput1 = $("<input type='text' name='CPE_singleFrequ' class='border border-box itemBand frequencyBand' "
		           + "onblur='validateRequired(event)' value='' style='width:110px;margin-left:4px' title='0 - 65535'></input>");
	var $cpeButtn = $("<a onclick='removeBandFreInputText(this)' style='margin-left:3px'>" 
			      + "<img src='${ctx}/js/jquery-easyui/themes/icons/edit_remove.png'></a>");
	
	$cpeDiv.append($cpeSpan1);
	$cpeDiv.append($cpeInput1);
	$cpeDiv.append($cpeButtn);
	
	//将新增的输入框插入到plmn参数之前
	//var currentDiv = $("input[name='CPE_Frequency']").parent("div");
	var currentDiv = $(".forFrequency:last");
	$cpeDiv.insertAfter(currentDiv);
	
	//将当前输入框后面的图标改为删除图标，并重新绑定事件
	$(e).children("img").attr("src","${ctx}/js/jquery-easyui/themes/icons/edit_add.png");
	$(e).attr("onclick", "addFrequencyLockInputText(this)");
	//计数，最多锁频暂时定位6组
	frequencyCount ++;
}


//删除选中的MME地址输入框
function removePCILockInputText(e) {
	$(e).parent("div").remove();
	
	var cpeDiv = $("input[name='CPE_Frequency']:first");
	var isVisible = cpeDiv.parent("div").children("span").css("visibility");
	if (isVisible == "hidden") {
		cpeDiv.parent("div").children("span").css("visibility","visible");
	}
	PciLockCount --;
}

function removeBandFreInputText(e) {
	$(e).parent("div").remove();
	
	var cpeDiv = $("input[name='CPE_singleFrequ']:first");
	var isVisible = cpeDiv.parent("div").children("span").css("visibility");
	if (isVisible == "hidden") {
		cpeDiv.parent("div").children("span").css("visibility","visible");
	}
	frequencyCount --;
}

function closeCpeFreqLockWin() {
	/* $("#winCpeFreqLock").window("close"); */
	closeDefaultWindow();
}
</script>