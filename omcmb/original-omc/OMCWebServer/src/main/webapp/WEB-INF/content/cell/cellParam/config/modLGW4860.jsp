<%@ page language="java" contentType="text/html; charset=UTF-8" pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<%@ include file="/common/alltaglibs.jsp" %>

 <style>

     #paramNodesUl li[name="LGW"]{
        display: block;
    }
     #paramNodesUl .staticDiv{
        display: block;
        width:auto;
    }
    #paramNodesUl .IMSIIP{
        width:auto;
        height:auto;
        margin-left:0;
        margin-top:10px;
    }
    #paramNodesUl #insertIP{
        height:auto;
        margin-bottom:25px;
    }
    .errTextGroup{
        display:inline-block;
    }
 </style>   
  
<ul id="paramNodesUl" class="paramNodesUl">
    <li>
        <label for="LTE_LGW_SWITCH_name">${LTE_LGW_SWITCH_name }</label>
        <select id="LTE_LGW_SWITCH_name" name="LTE_LGW_SWITCH" class="border border-box" onblur="createMML();" onchange="LGWEnableChange(this)">
            <option value="0">OFF</option>
            <option value="1">ON</option>         
        </select>
    </li>
    <li name="LGW">
        <label for="LTE_LGW_TRANSFER_MODE_name">${LTE_LGW_TRANSFER_MODE_name }</label>
        <select id="LTE_LGW_TRANSFER_MODE_name" name="LTE_LGW_TRANSFER_MODE" class="border border-box" onblur="createMML();" onchange="LGWModeChange(this)">
            <option value="0">NAT</option>
            <option value="1">Router</option>
            <option value="2">Bridge</option>
        </select>
    </li>
    <li name="LGW">
        <label for="LTE_LGW_START_UE_ADDR_name">${LTE_LGW_START_UE_ADDR_name }</label>
        <input id="LTE_LGW_START_UE_ADDR_name" name="LTE_LGW_START_UE_ADDR" type="text" onblur="ippoolonblur(event);createMML();" 
            title="${LTE_LGW_START_UE_ADDR_title }"  class="border border-box" />
        <div id="LTE_LGW_START_UE_ADDR_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_LGW_START_UE_ADDR_title }
        </div>
    </li>
    <li name="LGW">
        <label for="LTE_LGW_NET_MASK_name">${LTE_LGW_NET_MASK_name }</label>
        <select id="LTE_LGW_NET_MASK_name" name="LTE_LGW_NET_MASK" title="${LTE_LGW_NET_MASK_title }" class="border border-box item"  onblur="createMML();">            
            <option value="255.255.255.0" seleted>255.255.255.0</option>
            <option value="255.255.255.128" >255.255.255.128</option>
            <option value="255.255.255.192" >255.255.255.192</option>
            <option value="255.255.255.224" >255.255.255.224</option>
            <option value="255.255.255.240" >255.255.255.240</option>
            <option value="255.255.248.0" >255.255.248.0</option>
            <option value="255.255.240.0" >255.255.240.0</option>
        </select>
    </li>
    <li id="routerModeConfig" >
        <label for="LTE_LGW_STATIC_ADDRESS_SWITCH_name">${LTE_LGW_STATIC_IP_ADDR_SWITCH_name }</label>
        <select id="LTE_LGW_STATIC_ADDRESS_SWITCH_name" name="LTE_LGW_STATIC_IP_ADDR_SWITCH" class="border border-box" onblur="createMML();" onchange="LGWStaticAddrEnableChange(this)">
            <option value="0">OFF</option>
            <option value="1">ON</option> 
        </select>
    </li>
    <li class="staticDiv">
        <label >${LTE_LGW_FIRST_STATIC_IP_ADDRESS_name }</label>
        <input type="text" name="LTE_LGW_FIRST_STATIC_IP_ADDRESS" id="intelTdd_LGW_STATIC_IP_CONFIG_BEGIN" class="border border-box item" style="width:169px;height:26px;padding-left:10px"
            onblur="validateIPAddress(event);validateipandrange(event);createMML();" must="1" />
        -
        <input type="text" name="LTE_LGW_LAST_STATIC_IP_ADDRESS" id="intelTdd_LGW_STATIC_IP_CONFIG_END" class="border border-box item" style="width:169px;height:26px;padding-left:10px"  
            onblur="validateIPAddress(event);validateipandrange(event);createMML();" must="1"/>
         <div  class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;" id="intelTdd_LGW_STATIC_IP_CONFIG_err"></div>
    </li>
    <li class="staticDiv" id="insertIP">
        <label for="LTE_LGW_IMSI_IP_">${LTE_LGW_IMSI_IP_LIST_name }</label>
        <input type="text"  name="LGW_IMSI1"  id="intelTdd_LGW_IMSI_1" class="border border-box item" style="width:169px;height:26px;padding-left:10px" 
               onblur="imsionblur(event);createMML();" />
        -
        <input type="text"  name="LGW_IMSI_IP1"  id="intelTdd_LGW_IMSI_IP_1" class="border border-box item" style="width:169px;height:26px;padding-left:10px"
            onblur="validateIPAddress(event);validateipandrangeforbinding(event);createMML();" />
        <a onclick="intelTdd_addISMIAndIPInputText(this)" class="addIcon"></a>  
        <div  class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            <div class="errSpan errTextGroup" style="visibility:hidden;margin-top: 5px;width:180px;" id="intelTdd_LGW_IMSI_1_err"><%=rb.getString("LGWImsiTiShi")%></div>
            <div class="errSpan errTextGroup" style="visibility:hidden;margin-top: 5px;" id="intelTdd_LGW_IMSI_IP_1_err"><%=rb.getString("LGWIpTiShi")%></div>
        </div>  
    </li> 
</ul>

<script>

$(function(){
    LGWEnableChange(document.querySelector('#LTE_LGW_SWITCH_name'));
    createMML();
})

function LGWEnableChange(ele){
    var enaleFlag = $(ele).val();

    if("0" == enaleFlag){
        $("#LTE_LGW_TRANSFER_MODE_name").val("0")
        $("li[name='LGW']").hide();
        $(".staticDiv").hide();
        $("#routerModeConfig").hide();
    } else {
        $("li[name='LGW']").css("display","inline-block");
        $("#LTE_LGW_START_UE_ADDR_name").focus();
    }
}


/* Mode 为Router时显示 IMSI to IP Binding  */ 
function LGWModeChange(ele){
    var lgwmode = $(ele).val();
    
    if(lgwmode == 1){          // router
        $("#LTE_LGW_STATIC_ADDRESS_SWITCH_name").val("0");      
        $("li[name='LGW']").css("display","inline-block");
        $("#routerModeConfig").show();  
        $("#LTE_LGW_START_UE_ADDR_name").focus();
    }else if(lgwmode == 2){     
        $(".paramNodesUl li").hide();
        $("#LTE_LGW_SWITCH_name").parent().css("display","inline-block");
        $("#LTE_LGW_TRANSFER_MODE_name").parent().css("display","inline-block");
        $("#LTE_LGW_IFNAME_name").parent().css("display","inline-block");
    }else{        // NAT 
        $("#insertIP > li").remove();
        $("#LTE_LGW_START_UE_ADDR_name").focus();
        $(".staticDiv").hide();
        $("li[name='LGW']").css("display","inline-block");
        $("#routerModeConfig").hide();      
    }   
}

function LGWStaticAddrEnableChange(ele){
    var enaleFlag = $(ele).val();
    
    if("0" == enaleFlag){
        $(".staticDiv").hide();
        $(".staticDiv input").addClass('ignore');
        $("#insertIP > li").remove();
    } else {
        $(".staticDiv").show();
        $(".staticDiv input").removeClass('ignore');
    }   
}

var i =2;
var imsiIP_tdd = 0;
//添加一个MME地址输入框
function intelTdd_addISMIAndIPInputText(e) {

    var focusEle ;
    var flag = false;
    $("#insertIP input").each(function(){
        if($(this).val() == '' || $(this).hasClass("err_border")){
            $(this).focus();
            flag = true;
            return;
        }
    })
    
    if(flag){
        return;
    }
    
        
    if(i == 10){
        return;
    }
    imsiIP_tdd += 1;
    //添加新的输入框
    var $mmeSpan = $("<label style='display:inline-block;visibility:hidden'><%=rb.getString("IMSIBangDing")%></label>"); 
    var $mmeDiv = $("<li class='IMSIIP' id='"+i+"'></li>");
    var $mmeInput = $("<input type='text' name='LGW_IMSI"+i+"' class='border border-box item' style='width:169px;height:26px;margin-left:0px;padding-left:10px' "  + " title='<%=rb.getString("LGWImsiTiShi")%>'"
         + "id='intelTdd_LGW_IMSI_"+ i +"' onblur='imsionblur(event);createMML();' />"
         +" - "
         + "<input type='text' name='LGW_IMSI_IP"+i+"' class='border border-box item' style='width:169px;height:26px;padding-left:10px' "  + " title='<%=rb.getString("LGWIpTiShi")%>'"
         + "id='intelTdd_LGW_IMSI_IP_"+ i +"' onblur='validateIPAddress(event);validateipandrangeforbinding(event);createMML();' />"
    );
    var $mmeButtn = $("<a onclick='intelTdd_removeIMSIAndIPInputText(this);createMML();' style='margin-left:3px' class='subIcon' >" 
            + "</a>");
    
    var $errText1 = $("<div class='errSpan errTextGroup' style='visibility:hidden;margin-top: 5px;width:180px;' id='intelTdd_LGW_IMSI_"+ i +"_err'><%=rb.getString("LGWImsiTiShi")%></div>") ;
    var $errText2 = $("<div class='errSpan errTextGroup' style='visibility:hidden;margin-top: 5px;width:180px;' id='intelTdd_LGW_IMSI_IP_"+ i +"_err'><%=rb.getString("LGWIpTiShi")%></div>") ;
    
    var $errText = $("<div class='errSpan' style='margin-top: 5px;margin-left: 200px;'></div>");
    var $contan = $("#insertIP");
    
    $mmeDiv.append($mmeSpan);
    $mmeDiv.addClass("clearBoth");
    $mmeDiv.append($mmeInput);
    $mmeDiv.append($mmeButtn);
    $errText.append($errText1);
    $errText.append($errText2);
    $mmeDiv.append($errText);
    
    $contan.append($mmeDiv);
    
    i++;
}

//删除选中的MME地址输入框
function intelTdd_removeIMSIAndIPInputText(e) {
    $(e).parent("li").remove();
    i--;
}

function validateipandrangeforbinding(e){
    var ele = $(e["target"]),
        mode = $('#LTE_LGW_TRANSFER_MODE_name').val(),
        enable = $('#LTE_LGW_STATIC_ADDRESS_SWITCH_name').val();
    var ipvalue = $(ele).val();
    
    var ipstart = $("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").val();
    var ipend = $("#intelTdd_LGW_STATIC_IP_CONFIG_END").val();
    var ipstartValid =  $("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").hasClass("err_border");
    var ipendValid =  $("#intelTdd_LGW_STATIC_IP_CONFIG_END").hasClass("err_border");
    
    if(ipvalue && mode == '1' && enable == '1') {
        var retflag = validateLgwIp(e);
        if(!retflag){
            $("#" + ele.attr("id") + "_err").text(ipstart + "-" + ipend).css("visibility","visible");
            ele.addClass("err_border");
            return;
        }
        
        var compareFalg = compareIp(ipvalue,ipstart,ipend);
        if(!compareFalg){
            $("#" + ele.attr("id") + "_err").text(ipstart + "-" + ipend).css("visibility","visible");
            ele.addClass("err_border");
        } else {
            $("#" + ele.attr("id") + "_err").css("visibility","hidden");
            ele.removeClass("err_border");
        }

        if(ipstart == '' || ipstartValid){
            $("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").focus();
            return;
        }
        
        if(ipend == '' || ipendValid){
            $("#intelTdd_LGW_STATIC_IP_CONFIG_END").focus();    
            return;
        }
    }else {
    	$("#" + ele.attr("id") + "_err").css("visibility","hidden");
        ele.removeClass("err_border");
    }
}

function imsionblur(e){
    var ele = $(e["target"]),
    	imsivalue = $(ele).val(),
        mode = $('#LTE_LGW_TRANSFER_MODE_name').val(),
        enable = $('#LTE_LGW_STATIC_ADDRESS_SWITCH_name').val();
    
    if(!imsivalue){
        imsivalue = "";
    }

    if(mode == '1' && enable == '1') {
        if(imsivalue && imsivalue.length != 15){
            $("#" + ele.attr("id") + "_err").css("visibility","visible").show();
            ele.addClass("err_border");
        } else {
            $("#" + ele.attr("id") + "_err").css("visibility","hidden");
            ele.removeClass("err_border");
        }
    }else{
        $("#" + ele.attr("id") + "_err").css("visibility","hidden");
        ele.removeClass("err_border");
    }
}

//校验静态IP范围  static ip config  
function validateipandrange(e){
    var ele = $(e["target"]);
    var visible = isVisible(ele[0]);
    //静态IP的范围由 LGW IP以及Netmask计算得出 
    //若IP或 Netmask为空，则提示先输入IP或 Netmask
    var ip = $("#LTE_LGW_START_UE_ADDR_name").val();
    var netmask = $("#LTE_LGW_NET_MASK_name").val();
    var ipErr = $("#LTE_LGW_START_UE_ADDR_name").hasClass("err_border");
    var ipRangeTip ;
    if((ip == '' || netmask == '' || ipErr) && visible){
        ele.addClass("err_border");
        $("#intelTdd_LGW_STATIC_IP_CONFIG_err").text("<%=rb.getString("QingXianShuRuLGWIp")%>");
        $("#LTE_LGW_START_UE_ADDR_name").focus();
        return;
    }else{
        var ipstart = getLowAddr(ip,netmask);
        var ipend = getHighAddr(ip,netmask);
        
        ipRangeTip = '<%=rb.getString("IPBangDingFanWei")%>' + ipstart + '-' + ipend;
    }
    
    // 校验输入的IP格式
    var retflag = validateLgwIp(e);    
    if(!retflag && visible){
        $("#intelTdd_LGW_STATIC_IP_CONFIG_err").css("color","black").text(ipRangeTip).show();  //显示错误提示 
        ele.addClass("err_border");
        return;
    }
    
    //校验输入的IP是否在可配置的范围内 
    var ipvalue = $(ele).val();
    var compareFalg = compareIp(ipvalue,ipstart,ipend);     
    if(!compareFalg && visible){
        $("#intelTdd_LGW_STATIC_IP_CONFIG_err").css("color","black").text(ipRangeTip).show();  //显示错误提示 
        ele.addClass("err_border");
    } else {
        $("#staticIPRange").text(ipstart + "-" + ipend);
        ele.removeClass("err_border");
    }
    
    $('#insertIP input').blur();
}

//计算静态IP范围的起始值 
function getLowAddr(ip, netMask){
    var lowAddr = "";
    var ipArray = new Array();
    var netMaskArray = new Array();
    
    if (4 != ip.split(".").length || netMask == ""){
        return "";
    }
    for (var i = 0; i < 4; i++){
        ipArray[i] = ip.split(".")[i];
        netMaskArray[i] = netMask.split(".")[i];
        if ((ipArray[i] > 255) || (ipArray[i] < 0) || (netMaskArray[i] > 255) && (netMaskArray[i] < 0)){
            return "";
        }
        ipArray[i] = ipArray[i] & netMaskArray[i];
    }
    
    for (var i = 0; i < 4; i++){
        if(i == 3){
            ipArray[i] = ipArray[i] + 1;
        }
        if (lowAddr == ""){
            lowAddr +=ipArray[i];
        } else{
            lowAddr += "." + ipArray[i];
        }
    }
    return lowAddr;
}

//计算静态IP范围的终止值 
function getHighAddr(ip,netMask){
    var lowAddr = getLowAddr(ip,netMask);
    var hostNumber = getHostNumber(netMask);
    if(lowAddr == "" || hostNumber == 0){
        return "";
    }
    
    var lowAddrArray = new Array();
    for(var i = 0; i < 4; i++){
        lowAddrArray[i] = lowAddr.split(".")[i];
        if(i == 3){
            lowAddrArray[i] = Number(lowAddrArray[i] - 1);
        }
    }
    lowAddrArray[3] = lowAddrArray[3] + Number(hostNumber - 1);
   
    if(lowAddrArray[3] > 255){
        var k = parseInt(lowAddrArray[3] / 256);       
        
        lowAddrArray[3] = lowAddrArray[3] % 256;
       
        lowAddrArray[2] = Number(lowAddrArray[2]) + Number(k);       
       
        if(lowAddrArray[2] > 255){
            k = parseInt(lowAddrArray[2] / 256);
            lowAddrArray[2] = lowAddrArray[2] % 256;
            lowAddrArray[1] = Number(lowAddrArray[1]) + Number(k);
            if(lowAddrArray[1] > 255){
                k = parseInt(lowAddrArray[1] / 256);
                lowAddrArray[1] = lowAddrArray[1] % 256;
                lowAddrArray[0] = Number(lowAddrArray[0]) + Number(k);
            }
        }
    }

    var highAddr = "";
    for(var i = 0; i < 4; i++){
        if(i == 3){
          lowAddrArray[i] = lowAddrArray[i] - 1;
        }if(highAddr == ""){
            highAddr = lowAddrArray[i];
        }else{
            highAddr += "." + lowAddrArray[i];
        }
    }
    
    return highAddr;
}

function getHostNumber(netMask){
    var hostNumber = 0;
    var netMaskArray = new Array();
    for(var i = 0; i < 4; i++)
  {
        netMaskArray[i] = netMask.split(".")[i];
        if(netMaskArray[i] < 255)
    {
            hostNumber = Math.pow(256,3-i) * (256 - netMaskArray[i]);
            break;
        }
    }

    return hostNumber;
}

//校验输入的静态IP是否在允许范围内 
function compareIp(ipvalue,startip,endip) {
    var ipNum = changeIpToNum(ipvalue),
        startNum = changeIpToNum(startip),
        endNum = changeIpToNum(endip);

    if(isLessThan(ipNum, endNum) && isLessThan(startNum, ipNum)) {
        return true;
    }else {
        return false;
    }

    return true;
}

function changeIpToNum(ipStr) {
    var list = ipStr.split('.');

    list = list.map(function(item){
        if(item.length < 3) {
            var dis = 3 - item.length;
            for(var i = 0; i < dis; i++) item = '0' + item;
        }

        return item;
    });

    return list.join('');
}

//校验IP Pool输入格式 
function ippoolonblur(e){
    var retflag = validateLgwIp(e);
    var lwgMode = $('#LTE_LGW_TRANSFER_MODE_name').val(),
		lgwSwitch = $('#LTE_LGW_SWITCH_name').val();
    var ele = $(e["target"]);
    if(!retflag && lwgMode!='2' && lgwSwitch == '1'){
        $("#LTE_LGW_START_UE_ADDR_name_err").show();
         ele.addClass("err_border");
        $("#intelTdd_LGW_STATIC_IP_CONFIG_err").text("<%=rb.getString("QingXianShuRuLGWIp")%>").show();
        return;
    }else{
        $("#LTE_LGW_START_UE_ADDR_name_err").hide();
        ele.removeClass("err_border");
        var ipvalue = $(ele).val();
        var ipnetmask = $("#LTE_LGW_NET_MASK_name").val();
        var ipstart = getLowAddr(ipvalue,ipnetmask);
        var ipend = getHighAddr(ipvalue,ipnetmask);
        
        ipRangeTip = '<%=rb.getString("IPBangDingFanWei")%>' + ipstart + '-' + ipend;
        $("#intelTdd_LGW_STATIC_IP_CONFIG_err").css("color","black").text(ipRangeTip).show();
        $("#intelTdd_LGW_STATIC_IP_CONFIG_BEGIN").focus();
    }

    $('#insertIP input, .staticDiv input').blur();
    // 有效范围192.168.x.x 10.0.x.x 172.16.x.x
    
    <%-- var ipvalue = $(ele).val();
    var ipnetmask = $("#LTE_LGW_NET_MASK_name").val();
    var reg = new RegExp("^" + "(192.168.|10.0.|172.16.)");
    if(!reg.test(ipvalue)){
        $("#LTE_LGW_START_UE_ADDR_name_err").show();
        ele.addClass("err_border");
    } else {
        $("#LTE_LGW_START_UE_ADDR_name_err").hide();
        ele.removeClass("err_border");
        
        var ipstart = getLowAddr(ipvalue,ipnetmask);
        var ipend = getHighAddr(ipvalue,ipnetmask);
        
        ipRangeTip = '<%=rb.getString("IPBangDingFanWei")%>' + ipstart + '-' + ipend;
        $("#intelTdd_LGW_STATIC_IP_CONFIG_err").css("color","black").text(ipRangeTip).show();
    } --%>
}

//校验IP格式 
function validateLgwIp(e){   
    var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-4])$/;
    
    var ele = $(e["target"]);
    
    if (ele.val().length == 0) {        
        return false;
    }
        
    if (isValidIP(ele.val())) { // 格式正确
        return true;
    } else {
        return false;
    }
}

$("#LTE_LGW_NET_MASK_name").change(function(){
    lgwStaticIPRange();
});

//计算 static ip 范围 
function lgwStaticIPRange(){
    var ip = $("#LTE_LGW_START_UE_ADDR_name").val();
    var netmask = $("#LTE_LGW_NET_MASK_name").val(); 
    
    var ipstart = getLowAddr(ip,netmask);
    var ipend = getHighAddr(ip,netmask);
        
    var ipRangeTip = '<%=rb.getString("IPBangDingFanWei")%>' + ipstart + '-' + ipend;
    $("#intelTdd_LGW_STATIC_IP_CONFIG_err").css("color","black").text(ipRangeTip).show();
    
}
</script>