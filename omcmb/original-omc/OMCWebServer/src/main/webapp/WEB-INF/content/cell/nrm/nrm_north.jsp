<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%-- 系统设定页面 --%>
<style type="text/css">
.contentDiv {
	padding: 10px;
}
.contentDiv input {
	width: 305px;
	display: block;
}
</style>
<div class="easyui-layout" data-options="border:false,fit:true">
    <div region="center" data-options="border:false" style="padding:10px;">
        <div class="contentDiv">
			<span for="ItfnHeartbeatInterval"><%=rb.getString("XinTiaoJianGe")%></span>
			<input id="ItfnHeartbeatInterval" class="border-box border field" value="${ItfnHeartbeatInterval}"
				onblur="validateNumber(this, 1, 180)" title="Minimum: 1; Maximum: 180"/>
			<br/>
			<span for="ItfnPMFileDaysToHold"><%=rb.getString("PMWenJianBaoLiuTianShu")%></span>
			<input id="ItfnPMFileDaysToHold" class="border-box border field" value="${ItfnPMFileDaysToHold}"
				onblur="validateNumber(this, 7)" title="Minimum: 7"/>
			<br/>
			<span for="ItfnCMFileDaysToHold"><%=rb.getString("CMWenJianBaoLiuTianShu")%></span>
			<input id="ItfnCMFileDaysToHold" class="border-box border field" value="${ItfnCMFileDaysToHold}"
				onblur="validateNumber(this, 7)" title="Minimum: 7"/>
	    </div>
    </div>
    <div region="south" data-options="border:true"  style="height:51px;border-width: 0px 0 0 0;">
    	<div class="windowButtonGroup" style="margin-right:20px;">
    		<a class="linkbutton linkbutton_trend" onclick="itfnSettingCommit()"><span><%=rb.getString("QueDing")%></span></a>
        	<a class="linkbutton linkbutton_nowanna" onclick="closeItfnWin()" ><span><%=rb.getString("QuXiao")%></span></a>     
    	</div>
    </div>
</div>

<script type="text/javascript">
function closeItfnWin() {
	$("#winNRMConfig").window("close");
}

function itfnSettingCommit() {
	var params = {};
	if ($(".err_border").length > 0) {
		$.messager.alert(TiShi, "<%=rb.getString("ShuJuYanZhengBuTongGuo")%>");
		return;
	}
	$(".field").each(function () {
		params[$(this).attr("id")] = $(this).val();
	});
	$.post("${ctx}/cell/collect/nrm/doItfnUpdate.action", params, function (data) {
		if (data["success"]) {
			$("#winNRMConfig").window("close");
		}
		$.messager.alert(TiShi, data["message"]);
	}, "json");
}

/**
 * 验证某元素的值是否为正整数
 */
function validateNumber(ele, min, max) {
	if (!ele) {
		return;
	}
	var reg = /^\d+$/;
	if (reg.test(ele.value)) {// 是数字
		$(ele).removeClass("err_border");
		ele.value = parseInt(ele.value, 10);
	} else {
		$(ele).addClass("err_border");
	}
	
	if (min) {// 检验最小值
		if (ele.value < min) {
			$(ele).addClass("err_border");
			return;
		}
	}
	
	if (max) {
		if (ele.value > max) {
			$(ele).addClass("err_border");
			return;
		}
	}
}
</script>