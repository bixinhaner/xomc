<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
    .itemDiv {
        height: 30px;
        padding: 3px;
    }

    .itemDiv span {
        display: inline-block;
        width: 150px;
    }

    .item {
        width: 200px;
    }
</style>

<div class="easyui-layout" data-options="border:false,fit:true">
    <div region="center" data-options="border:false" style="padding:20px;">
        <span><%=rb.getString("PCIInstruction")%></span>
        <div class="itemDiv">
            <span style="width:150px; padding-top:10px;"><%=rb.getString("PCIStart")%></span>
            <input type="text" name="PCI_Start" class="border border-box item start" value="${PCI_Start}"/>
        </div>
        <div class="itemDiv">
            <span style="width:150px;"><%=rb.getString("PCIEnd")%></span>
            <input type="text" name="PCI_End" class="border border-box item stop" value="${PCI_End}"/>
        </div>
    </div>
    <div region="south" data-options="border:true" style="border-width:0px 0 0 0;height:61px;padding:10px 20px 10px 0;">
        <div class="windowButtonGroup" >
        	<a class="linkbutton linkbutton_trend" onclick="savePCIRange()" ><span><%=rb.getString("QueDing")%></span></a>
        	<a class="linkbutton linkbutton_nowanna" onclick="closePCIRangeWin()"><span><%=rb.getString("QuXiao")%></span></a>
        </div>
    </div>
</div>

<script type="text/javascript">
<%--关闭窗口--%>
function closePCIRangeWin(){
	/* $("#winPCIRangeSetting").window("close"); */
	closeDefaultWindow();
}

function savePCIRange(){
	if (parseInt($(".start").val()) < 0 || parseInt($(".stop").val()) > 503){
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIRangeUnreasonable")%>");
		return;
	} else if (parseInt($(".start").val()) > parseInt($(".stop").val())) {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIRangeValueError")%>");
		return;
	} else if (!isPositiveNum($(".start").val()) || !isPositiveNum($(".stop").val())){
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIRangeTypeError")%>")
		return;
	}
	var params = {};
	$(".item").each(function(){
		if($(this).val()){
			params[$(this).attr("name")] = $(this).val();
		}
	});
	$.post("${ctx}/cell/SON/updatePCIRange.action", params, function (data) {
        if(data["success"]){
        	closePCIRangeWin();
        	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("CaoZuoChengGong")%>");
        } else {
        	$.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
        }
    }, "json");
}

function isPositiveNum(s){
	var re = /^(0|[0-9]*[1-9][0-9]*)$/;
	return re.test(s);
}
</script>