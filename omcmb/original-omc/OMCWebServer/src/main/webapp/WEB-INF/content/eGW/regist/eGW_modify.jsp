<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
.itemDiv{
	width:450px;
	height:100px;
	float:left;
} 
.success{
	height:38px;
	margin-left:30px;
	color:#508D9B;
	font-size:15px;
	font-weight:bold;
	padding:0 20px 0 20px;
	line-height:38px;
	text-indent:25px;
	display:none;
}
.bgPanelbarDiv{
	width:90%;
	margin-left:80px;
	clear:both;
}
.errorTitle{
	height:26px;
	line-height:18px;
	width:320px;
	font-size:12px;
	color:red;
	display : none;
}
</style>
<!-- 名称目前只做了32的长度限制 onblur="validateEGWName(event)"  -->
<div class="pageDefault slide-position-top">
	<div class="slidebarTitleDiv">
	   	<span><%=rb.getString("XiuGai")%></span>
	</div>
	<div class="slideBody">
		<div class="bgPanelbarDiv" style="margin-top:40px;">
			<div class="itemDiv">
				<span><%=rb.getString("EGWMingCheng")%><%=rb.getString("MaoHao")%></span>
				<input type="text" id="eGWModifyName"class="inputDivCss border border-box item" title="<%=rb.getString("QingShuRuEGWMingCheng")%>"
						onblur="validateEGWName(event);" min_length="1" maxlength="32" must="1" />
				<div class="errorTitle" id="eGWModifyName_err"><%=rb.getString("QingShuRuEGWMingCheng")%></div>	
			</div>
			<div class="itemDiv" style="">
				<span><%=rb.getString("EGWIP")%><%=rb.getString("MaoHao")%></span>
				<input type="text" id="eGWModifyIP"class="inputDivCss border border-box item" 
						onblur="validateMaxAndMinVal(event)" min_value="0" max_value="65535" must="1" disabled/>
				<div class="errorTitle" id="eGWModifyIP_err"><%=rb.getString("IPDiZhi")%></div>	
			</div>
		</div>
		<div class="bgPanelbarDiv">
			<div class="itemDiv" style="">
				<span><%=rb.getString("EGWDuanKou")%> (0~65535)<%=rb.getString("MaoHao")%></span>
				<input type="text" id="eGWModifyPort"class="inputDivCss border border-box item"
						onblur="validateMaxAndMinVal(event)" min_value="0" max_value="65535" must="1" disabled/>
				<div class="errorTitle" id="eGWModifyPort_err"><%=rb.getString("ZhengXing")%><%=rb.getString("MaoHao")%> 0~65535</div>	
			</div>
		</div>
		
	</div>
	<div class="slideFooter" style="display:flex;align-items:center">
		<a href="#" class="linkbutton"  onclick="eGWRegistModifysave()"><span><%=rb.getString("QueDing")%></span></a>
		<div class='success'><%=rb.getString("XiuGaiWanCheng")%></div>
	</div>
</div>
<script>
//原网关名称
var gwNameOld = '';
//新网关名称   gwNameNew = '';
var eGWId = "";
$(function(){
	closeLoading();
	var datagridRow = $("#eGWRegistDatagrid").datagrid("getSelected");
	eGWId = datagridRow.gw_id;
	gwNameOld = datagridRow.gw_name;
	$("#eGWModifyName").val(gwNameOld);
	$("#eGWModifyIP").val(datagridRow.gw_ip);
	$("#eGWModifyPort").val(datagridRow.gw_port);
})
// 修改确定  进行验证数据是否填写准确
function eGWRegistModifysave(){
	var allInput = $("#eGWModifyDiv input");
	for(var i=0;i<allInput.length;i++){
		if($(allInput[i]).val()==""){
			$(allInput[i]).addClass('err_border');
			$(allInput[i]).next().next().show();
		};
	}
	if ($("#eGWModifyDiv .item.err_border").length > 0) {
		return;
	}
	var gwNameNew = $("#eGWModifyName").val();
	//参数未发生变化时提示没有变化 
	if(gwNameNew == gwNameOld){
		showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
	}else{
		if(validateFilterName(gwNameNew,eGWId)){
			var params = {
					"gwId" : eGWId,
					"gwNameOld" : gwNameOld,
					"gwNameNew" : $("#eGWModifyName").val()
			}
			$.post("${ctx}/egw/register/editEgw.action",params,function(data){
		         if (data["success"]) {
		        	$("#eGWRegistDatagrid").datagrid("reload");
		        	$(".success").show();
		        	setTimeout('cancelSysModifyRole()',3000);
		        } else {
		            showMsg('error_msg',data["message"]);
		            return;
		        } 
		    }, "json");
		}
	}
}
//eGW名称校验
function validateEGWName(e){
	var ele = $(e["target"]);
    var currVal = ele.val();
    var reg = /^[A-Za-z]{1}[A-Za-z0-9_\-]{0,31}$/;
    if(!reg.test(currVal)){
    	 $("#" + ele.attr("id") + "_err").show();
         ele.addClass("err_border");
    }else{
    	 $("#" + ele.attr("id") + "_err").hide();
         ele.removeClass("err_border");
    }
}

//验证名称是否可用
function validateFilterName(gwName,eGWId) {
	var exist = false;
	$.ajax({
		type: "post",
		url: "${ctx}/egw/register/checkeGWName.action", 
		data: {"gwName": gwName,
			   "eGWId":eGWId},
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
		showMsg('prompt_msg','<%=rb.getString("WangGuanMingChengYiCunZai")%>');
		return;
	}
	return true;
}
</script>