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
        <label for="LTE_ENABLE_LGW_name">${LTE_ENABLE_LGW_name }</label>
        <select id="LTE_ENABLE_LGW_name" name="LTE_ENABLE_LGW" class="border border-box" onblur="createMML();" onchange="LGWEnableChange(this)">
            <option value="0">false</option>
            <option value="1">true</option>         
        </select>
    </li>
    <li name="LGW">
        <label for="LTE_LGW_MODE_name">${LTE_LGW_MODE_name }</label>
        <select id="LTE_LGW_MODE_name" name="LTE_LGW_MODE" class="border border-box" onblur="createMML();" onchange="LGWModeChange(this)">
            <option value="NAT">NAT</option>
            <option value="Router">Router</option>
            <option value="Bridge">Bridge</option>
        </select>
    </li>
    <li name="LGW">
        <label for="LTE_LGW_IP_POOL_name">${LTE_LGW_IP_POOL_name }</label>
        <input id="LTE_LGW_IP_POOL_name" name="LTE_LGW_IP_POOL" type="text" onblur="ippoolonblur(event);createMML();" 
            title="${LTE_LGW_IP_POOL_title }"  class="border border-box" />
        <div id="LTE_LGW_IP_POOL_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_LGW_IP_POOL_title }
        </div>
    </li>
    <li name="LGW">
        <label for="LTE_LGW_POOL_NETMASK_name">${LTE_LGW_POOL_NETMASK_name }</label>
        <input id="LTE_LGW_POOL_NETMASK_name" name="LTE_LGW_POOL_NETMASK" type="text" onblur="ipmaskonblur(event);createMML();" 
            title="${LTE_LGW_POOL_NETMASK_title }"  class="border border-box" />
        <div id="LTE_LGW_POOL_NETMASK_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_LGW_POOL_NETMASK_title }
        </div>
    </li>
</ul>

<script>

$(function(){
    LGWEnableChange(document.querySelector('#LTE_ENABLE_LGW_name'));
    createMML();
})

function LGWEnableChange(ele){
    var enaleFlag = $(ele).val();

    if("0" == enaleFlag){
        $("#LTE_LGW_MODE_name").val("NAT")
        $("li[name='LGW']").hide();
        $(".staticDiv").hide();
        $("#routerModeConfig").hide();
    } else {
        $("li[name='LGW']").css("display","inline-block");
        $("#LTE_LGW_IP_POOL_name").focus();
        LGWModeChange(document.querySelector('#LTE_ENABLE_LGW_name'));
    }
}


/* Mode 为Router时显示 IMSI to IP Binding  */ 
function LGWModeChange(ele){
    var lgwmode = $(ele).val();
    
    if(lgwmode == 'Router'){          // router
        $("#LTE_LGW_STATIC_ADDRESS_SWITCH_name").val("0");      
        $("li[name='LGW']").css("display","inline-block");
        $("#LTE_LGW_IP_POOL_name").focus();
    }else{     
        $(".paramNodesUl li").hide();
        $("#LTE_ENABLE_LGW_name").parent().css("display","inline-block");
        $("#LTE_LGW_MODE_name").parent().css("display","inline-block");
    }
}


var i =2;
var imsiIP_tdd = 0;


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
    var lwgMode = $('#LTE_LGW_MODE_name').val(),
    	lgwSwitch = $('#LTE_ENABLE_LGW_name').val();
    var ele = $(e["target"]);
    if(!retflag && lwgMode == 'Router' && lgwSwitch == '1'){
        $("#LTE_LGW_IP_POOL_name_err").show();
         ele.addClass("err_border");
        return;
    }else{
        $("#LTE_LGW_IP_POOL_name_err").hide();
        ele.removeClass("err_border");
    }
}

function ipmaskonblur(e){
    var retflag = validateLgwIp(e);
    var lwgMode = $('#LTE_LGW_MODE_name').val(),
    	lgwSwitch = $('#LTE_ENABLE_LGW_name').val();
    var ele = $(e["target"]);
    if(!retflag && lwgMode == 'Router' && lgwSwitch == '1'){
        $("#LTE_LGW_POOL_NETMASK_name_err").show();
         ele.addClass("err_border");
        return;
    }else{
        $("#LTE_LGW_POOL_NETMASK_name_err").hide();
        ele.removeClass("err_border");
    }
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

</script>