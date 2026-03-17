<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 手动添加基站到设备组 --%>
<div id="addDevice" class="flex-ctn" style="height: 100%;">
    <div region="center" data-options="border:false" style="padding: 20px 30px;" class="flex-item">
        <div class="itemDiv" style="height:30px;font-size: 16px;margin-left:0px;width:395px;">
			<span style="display:block"><%=rb.getString("TianJiaJiZhan")%></span>
		</div>
        <div class="itemDiv" style="margin-top:10px;height:50px;margin-left: 20px;">
            <span style="display:block"><%=rb.getString("XiaoZhanBianMa")%></span>
            <input id="addSerialNumInput" type="text" name="serialNumber" class="border border-box item" must=1 minLength=2 maxlength=30 title=""
            	onblur="validateJudgeByRegex(event)" must="1" 
				vali-regex="/^(\d|[a-zA-Z]|-){2,30}$/" />
        </div>
    	<div id="addSerialNumInput_err" style="margin-top:8px;margin-left: 20px;"><%=rb.getString("ZuiXiaoChangDu") %> : 2 <%=rb.getString("ZiFu")%><%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaChangDu")%> : 30 <%=rb.getString("ZiFu")%></div>
    </div>
    <div region="south" data-options="border:false,height:57" style="min-height: 57px;">
    	<div class="windowButtonGroup" style="margin-right:34px">
   			<a onclick="addDevice()" class="linkbutton linkbutton_trend" ><span><%=rb.getString("TianJia")%></span></a>
    		<a onclick="closeWinAddDevice()" class="linkbutton linkbutton_nowanna" ><span><%=rb.getString("QuXiao")%></span></a>
    	</div>
		
	</div>
</div>

<script type="text/javascript">
// 关闭添加基站窗口
function closeWinAddDevice(){
	$("#addHalobEnbBlock").slideUp(500,function(){$("#addHalobEnbBlock").html("")});
}

// 添加基站
function addDevice() {
	$("#addDevice input.item").blur();
	if ($("#addDevice input.item.err_border").length > 0) {
		return;
	}
	var serialNumber = $("#addDevice input[name='serialNumber']").val();
	var temp = /^(\d|[a-zA-Z]|-){2,30}$/;
	if (serialNumber != null && serialNumber.length != 0 && !temp.test(serialNumber)) {
		$("#addSerialNumInput_err").html('<%=rb.getString("QingShuRuZhengQueSn")%>');
		return;
	}
	if (serialNumber == null || serialNumber.length == 0) {
		$("#addSerialNumInput_err").html('<%=rb.getString("SNBuNengWeiKong")%>');
		return;
	}
	var params = {
		"serialNumber": serialNumber
    };
	$.post("${ctx}/cell/halobSelfConfig/addOneDeviceParamConfig.action", params, function(data) {
		if (data["success"]) {
        	$("#singleDeviceConfig").datagrid("reload");
			closeWinAddDevice();
			$.messager.alert(TiShi, data["message"]);
		} else {
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

function validateJudgeByRegex(e){
	var ele = $(e["target"]);
    var reg = eval(ele.attr("vali-regex"));
    var currVal = ele.val();
    var must = ele.attr("must");
    
    var currValLength = currVal.length;
    if (currValLength == 0) {
    	if (must == 1) {
    		$("#" + ele.attr("id") + "_err").css('color','red');
    		ele.addClass("err_border");
    		$("#addSerialNumInput_err").html('<%=rb.getString("SNBuNengWeiKong")%>');
    	} else {
    		$("#" + ele.attr("id") + "_err").css('color','#444');
    		ele.removeClass("err_border");
    		$("#addSerialNumInput_err").html('<%=rb.getString("ZuiXiaoChangDu") %> : 2 <%=rb.getString("ZiFu")%><%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaChangDu")%> : 30 <%=rb.getString("ZiFu")%>');
    	}
    	return;
    }
    var showFlag = false;
    if (!reg.test(currVal)) {
    	showFlag = true;
    }
    
    if (showFlag) {
        $("#" + ele.attr("id") + "_err").css('color','red');
		$("#addSerialNumInput_err").html('<%=rb.getString("QingShuRuZhengQueSn")%>');
        ele.addClass("err_border");
    } else {
        $("#" + ele.attr("id") + "_err").css('color','#444');
        $("#addSerialNumInput_err").html('<%=rb.getString("ZuiXiaoChangDu") %> : 2 <%=rb.getString("ZiFu")%><%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaChangDu")%> : 30 <%=rb.getString("ZiFu")%>');
    	ele.removeClass("err_border");
    }
}
</script>