<%--
  User: Yujianfei
  Date: 2017/2/17
  Time: 11:37
--%>
<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
  #selfconfigSetting .itemDiv {
      height: 45px;
      padding: 10px;
  }
  #selfconfigSetting .itemDiv span {
    display: inline-block;
    width: 130px;
  }
  #selfconfigSetting .item {
    width: 200px;
  }
</style>

<div class="easyui-layout" data-options="border:false,fit:true">
    <div id="selfconfigSetting" region="center" data-options="border:false">
        <div class="itemDiv" style="margin-top:20px;">
            <span style="vertical-align:top; margin:5px 0 0 15px;"><%=rb.getString("XIAOQUID")%></span>
            <div style="height:40px; width:200px;display: inline-block;">
            	<input type="text" name="CELL_ID" class="border border-box item" value="${CELL_ID}"/>
            	<span style="display: inline-block;"></span>
            </div>
            <span style="vertical-align:top; margin:5px 0 0 15px;"><%=rb.getString("PLMN")%></span>
            <div style="height:40px; width:200px;display: inline-block;">
            	<input type="text" name="PLMN" class="border border-box item" value="${PLMN}"/>
            	<span style="display: inline-block; width:200px; color:#A9A9A9;"><%=rb.getString("QuZhiFanWei")%><%=rb.getString("MaoHao")%>5-6 <%=rb.getString("WeiShu")%></span>
            </div>
        </div>
        <div class="itemDiv">
            <span style="vertical-align:top; margin:5px 0 0 15px;">PCI</span>
            <div style="height:40px; width:200px;display: inline-block;">
            	<input type="text" name="PCI" class="border border-box item" value="${PCI}"/>
            	<span style="display: inline-block; color:#A9A9A9;width:200px; ">0-503,'23'or'1,2,'or'300..500'</span>
            </div>
            <span style="vertical-align:top; margin:5px 0 0 15px;"><%=rb.getString("XiaXingPinDian")%></span>
            <div style="height:40px; width:200px;display: inline-block;">
            	<input type="text" name="DL_EARFCN" class="border border-box item" value="${DL_EARFCN}"/>
            	<span style="display: inline-block;"></span>
            </div>
        </div>
        <div class="itemDiv">
            <span style="vertical-align:top; margin:5px 0 0 15px;"><%=rb.getString("TAC")%></span>
            <div style="height:40px; width:200px;display: inline-block;">
            	<input type="text" name="TAC" class="border border-box item" value="${TAC}"/>
            	<span style="display: inline-block; color:#A9A9A9;"><%=rb.getString("QuZhiFanWei")%><%=rb.getString("MaoHao")%> 0-65535</span>
            </div>
            <span style="vertical-align:top; margin:5px 0 0 15px;"><%=rb.getString("MMEDIZHI")%></span>
            <div style="height:40px; width:200px;display: inline-block;">
            	<input type="text" name="MME_ADDRESS" class="border border-box item" value="${MME_ADDRESS}"/>
            	<span style="display: inline-block;"></span>
            </div>
        </div>
       <%--  <div class="itemDiv">
        	<span style="vertical-align:top; margin:5px 0 0 15px;"><%=rb.getString("DaiKuan")%></span>
            <div style="height:40px; width:200px;display: inline-block;">
            	<input type="text" name="BANDWIDTH" class="border border-box item" value="${BANDWIDTH}"/>
            	<span style="display: inline-block; color:#A9A9A9;"><%=rb.getString("QuZhiFanWei")%><%=rb.getString("MaoHao")%> 5/10/15/20</span>
            </div>
        </div> --%>
    </div>
    <div region="south" data-options="border:false,height:71" style="padding: 10px 20px 20px;">
        <div class="windowButtonGroup">
        	 <a class="linkbutton linkbutton_trend" onclick="selfSettingCommit()"><span><%=rb.getString("QueDing")%></span></a>
        	 <a class="linkbutton linkbutton_nowanna" onclick="closeSelfWin()"><span><%=rb.getString("QuXiao")%></span></a>
        </div>
    </div>
</div>

<script type="text/javascript">
function closeSelfWin() {
	closeDefaultWindow();
}

function selfSettingCommit() {
	var params={};
    params["serialNumber"] = "<%=request.getParameter("serialNumber") %>";
    $("#selfconfigSetting .item").each(function() {
        if ($(this).val()) {
            params[$(this).attr("name")] = $(this).val();
        } 
    });
    //判断cellid合法性
    var cellId = /^(0|[1-9][0-9]*)$/;
    if (!cellId.test(params["CELL_ID"])) {
    	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("XiaoQuIdGeShiCuoWu")%>");
    	return;
    }
    if (parseInt(params["CELL_ID"]) < 0 || parseInt(params["CELL_ID"]) > 268435455) {
    	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("XiaoQuIDChaoChuFanWei")%>");
    	return;
    }
    
    
<%--     var picRegex=/^(?:(?:(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]),)*(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]))$|^(?:(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3])\.\.(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]))$/;
    if (!picRegex.test(params["PCI"])) {
    	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIGeShiCuoWu")%>");
    	return;
    } --%>
    
    if (params["PCI"].length == 0) {
    	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIGeShiCuoWu")%>");
    	return;
    }
    $.post("${ctx}/cell/selfstart/isElfcellPciInValid.action", {"pciStr" : params["PCI"]}, function(data) {
        if (data["success"]) {
        	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIGeShiCuoWu")%>");
        	return;
        } else {
        	 //判断频点合法性
            var earfcn = /^(0|[1-9][0-9]*)$/;
            if (!earfcn.test(params["DL_EARFCN"])) {
            	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PinDianGeShiCuoWu")%>");
            	return;
            }
            if ((parseInt(params["DL_EARFCN"]) < 0 || parseInt(params["DL_EARFCN"]) > 65535)) {
            	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PinDianChaoChuFanWei")%>");
            	return;
            }
            //判断TAC的合法性
            var tac = /^(0|[1-9][0-9]*)$/;
            if (!tac.test(params["TAC"])) {
            	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("TACGeShiCuoWu")%>");
            	return;
            }
            if (parseInt(params["TAC"]) < 0 || parseInt(params["TAC"]) > 65535) {
            	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("TacChaoChuFanWei")%>");
            	return;
            }
            //判断PLMN的合法性
            var plmn = /^(0|[1-9][0-9]*)$/;
            if (!plmn.test(params["PLMN"])) {
            	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PLMNGeShiCuoWu")%>");
            	return;
            }
            if (params["PLMN"].toString().length < 5 || params["PLMN"].toString().length > 6) {
            	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PLMNChaoChuFanWei")%>");
            	return;
            }
            //验证核心网ip地址合法性
            var re =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
            if (!isValidIP(params["MME_ADDRESS"])) {
            	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("HeXinWangIPDiZhiFeiFa")%>");
            	return;
            }
            
           <%--  if (parseInt(params["BANDWIDTH"]) != 5 && parseInt(params["BANDWIDTH"]) != 10 && parseInt(params["BANDWIDTH"]) != 15 && parseInt(params["BANDWIDTH"]) != 20) {
            	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("DaiKuanChaoChuFanWei")%>");
            	return;
            } --%>
            
            $.post("${ctx}/cell/selfstart/selfstartConfigSetting.action", params, function(data) {
                if (data["success"]) {
                    closeSelfWin();
                    $("#tableSelfStartParams").datagrid('reload');
                    $.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("CaoZuoChengGong")%>");
                } else {
                    $.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
                }
            }, "json");
        }
    }, "json");
    
<%--     //判断PCI合法性
    var pci = /^(0|[1-9][0-9]*)$/;
    if (!pci.test(params["PCI"])) {
    	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIGeShiCuoWu")%>");
    	return;
    }
    if (parseInt(params["PCI"]) < 0 || parseInt(params["PCI"]) > 503) {
    	$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("PCIChaoChuFanWei")%>");
    	return;
    } --%>
   
}
</script>