<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<div id="compactParamChange" class="easyui-layout" data-options="border:false,fit:true">
	<div region="center" data-options="border:false">
		<%-- <div class="itemDiv">
			<span><%=rb.getString("PinDuan")%></span>
			<input type="text" name="LTE_FREQ_BAND_INDICATOR" class="border border-box item" 
			 oldValue="${FREQBAND_INDICATOR}" value="${FREQBAND_INDICATOR}"
					onblur="validateMaxAndMinVal(event)" min_value="1" max_value="65"
					title="int, min value: 1, max value: 65"/>
		</div> --%>
		<div class="itemDiv" style="padding:20px 0 0 10px;">
			<span><%=rb.getString("PinDian")%></span>
			<input type="text" name="LTE_UL_DL_EARFCN" class="border border-box item" oldValue="${EARFCNULINUSE}" value="${EARFCNULINUSE}"
					onblur="validateMaxAndMinVal(event)" min_value="0" max_value="65535"
					title="int, min value: 0, max value: 65535"/>
		</div>
		<div class="itemDiv" style="padding:20px 0 0 10px;">
			<span><%=rb.getString("PCI2")%></span>
			<input type="text" name="LTE_PHY_CELLID_LIST" class="border border-box item" oldValue="${PHYCELLID}" value="${PHYCELLID}"
					onblur="validateMaxAndMinVal(event)" min_value="0" max_value="503"
					title="int, min value: 0, max value: 503"/>
		</div>
	</div>
	<div region="south" data-options="border:true" style="border-width:1px 0 0 0;height:47px;padding:10px 0;">
		<div class="windowButtonGroup">
			<a class="linkbutton" onclick="closeWin()" style="float:right;margin-right:20px;"><span><%=rb.getString("QuXiao")%></span></a>
			<a class="linkbutton" onclick="cellFreqLockCommit()" style="float:right;margin-right:15px;"><span><%=rb.getString("QueDing")%></span></a>
		</div>
	</div>
</div>

<%-- 窗口-右键设置进度条 --%>
<div id="winSettingPro" title="<%=rb.getString("CanShuPeiZhiJinDu")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:70,resizable:false,closable:false">
    <img src="${ctx}/js/jquery-easyui/themes/bootstrap/images/loading.gif"/>
    <span style="margin-left:75px;"><%=rb.getString("eNodeBZhengZaiSheZhi")%></span>
</div>

<script type="text/javascript">

$(function() {
	closeLoading();
	var disableFreqAndPci = "${disableFreqAndPci}";
	if (disableFreqAndPci == 1) {
		$("input[name='LTE_UL_DL_EARFCN']").attr("disabled", true);
		$("input[name='LTE_PHY_CELLID_LIST']").attr("disabled", true);
	}
});

function closeWin() {
	/* $("#winCpeFreqLockChange").window('close'); */
	closeDefaultWindow();
}

function cellFreqLockCommit() {
	var params = {};
	var cpeCodeArr = "${cpeCodeArr}";
	var cellCode = "${cellCode}";
	if (cpeCodeArr == null && cpeCodeArr.length == 0) {
		$.messager.alert(TiShi, "<%=rb.getString("BuNengGengGaiCpeSuoPinPeiZhi")%>");
		return
	}
	var earfcn = $("input[name='LTE_UL_DL_EARFCN']").val();
	var pci = $("input[name='LTE_PHY_CELLID_LIST']").val();
	
	if (earfcn != null && earfcn.length != 0) {
		params["earfcn"] = earfcn;
	} else if (pci != null && pci.length != 0) {
		params["pci"] = pci;
	}
	
	params["cpeCodeArr"] = cpeCodeArr;
	params["cellCode"] = cellCode;
	
	var msg = "<%=rb.getString("WenHao")%>" + "<%=rb.getString("GaiCaoZuoHuiDaoZhiCPEChongXinSuoPin")%>";
	
	$.messager.confirm("<%=rb.getString("QueRen")%>", msg , function(r) {
		if (r) {
			$("#winSettingPro").window("open");
			$.post("${ctx}/cell/CPE/updateNodeBSettringsAndCpeFreqLockParams.action", params, function(data){
				$("#winSettingPro").window("close");
				if (data["success"]) {
					/* $("#winCpeFreqLockChange").window("close"); */
					closeDefaultWindow();
					cpevm.refreshList();
				} else {
					/* $("#winCpeFreqLockChange").window("close"); */
					closeDefaultWindow();
					$.messager.alert(TiShi, data["message"]);
				}
			}, "json");
		}
	});
}
</script>