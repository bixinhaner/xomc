<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style>		
.switch {
	background-color: #66CC66;
	margin-left:0px;
}
.controlManaItem{
	color : #85A8BF;
	margin-top:40px;
}
.singleContentDiv div{
	display: inline-block;
}
.singleContentDiv > div{
	display: block;
}
.controlManaLabel{
	width:150px;
	font-size:13px;
}
.controlManaContent{
	display:inline-block;
}
.controlManaContent input,.controlManaContent label{
	cursor:pointer;
	vertical-align : middle;
	margin-right:5px;
	/* vertical-align : -webkit-baseline-middle; */
}
.controlManaContent label{
	margin-right:30px;
}
.selfConfigSucTip{
	display:inline-block;
	min-width:200px;
	height:34px;
	line-height:36px;
	padding : 0 15px 0 50px;
	margin-left:35px;
	color:#508D9B;
	font-size:16px;
	font-weight:bold;
	background:#E9FBFF url("${ctx}/images/success.png") no-repeat 16px center;
}
</style>
<div class="panelDefault">
	<div class="singleContentDiv" style="top:40px;left:80px;">
		<div  id="" class = "controlManaItem" style="margin-top:0">
			<div class="controlManaLabel" style="vertical-align:middle;"><%=rb.getString("ZiQiDongKaiGuan")%><%=rb.getString("MaoHao")%></div>
			<div class="controlManaContent" style="vertical-align:middle;">
				<div class='switch' onclick="OpenSelfConfig(this)">
					<div isopen='true' id="self_config_switch" oldValue="" class='btnn' style='left:24px;'></div>
				</div>
			</div>
		</div>
		<div id="" class = "controlManaItem" >
			<div class="controlManaLabel"><%=rb.getString("ZiKaiZhanZhiXingFangShi")%><%=rb.getString("MaoHao")%></div>
			<div class="controlManaContent">
				<input type="radio" name="carryOutMode" id="autoMode" checked value="0"><label for="autoMode"><%=rb.getString("ZiDongZhiXing")%></label>
				<input type="radio" name="carryOutMode" id="manualMode" value="1"><label for="manualMode"><%=rb.getString("ShouDongZhiXing")%></label>
			</div>
		</div>
		<div id="" class = "controlManaItem" >
			<div class="controlManaLabel"><%=rb.getString("LiuChengKongZhi")%><%=rb.getString("MaoHao")%></div>
			<div class="controlManaContent">
				<input type="checkbox" name="processMode" id="softUpgrade" checked value="1"><label for="softUpgrade"><%=rb.getString("RuanJianShengJi")%></label>
				<input type="checkbox" name="processMode" id="licenseIssued" checked value="2"><label for="licenseIssued"><%=rb.getString("LicenseXiaFa")%></label>
				<input type="checkbox" name="processMode" id="selfConfig" checked value="3"><label for="selfConfig"><%=rb.getString("ZiQiDongCanShuPeiZhi")%></label>
			</div>
		</div>
		<div id="controlManaTip" style="height:15px;margin-left:153px;margin-top:15px;color:#FF0000"></div>
		<div style="margin-top:60px">
		    <a href="#" class="linkbutton eNbSelfConfiguration hidden" onclick="saveSeleConfig()"><span><%=rb.getString("QueDing")%></span></a>
		    <a style="display:none" class="selfConfigSucTip"><%=rb.getString("ZiPeiZhiCanShuYiTiJiao")%></a>
		</div>
	</div>
</div>
<script type="text/javascript">
$(function() {
	closeLoading();
	//勾选复选框之后“至少选一个”的提示消失
	$("input[type=checkbox]").click(function(){
		if($(this).prop("checked")){
			$("#controlManaTip").html("");
		}
	})
	//获取数据设置页面参数
	$.post("${ctx}/cell/halobSelfConfig/getCurrSelfConfigRule.action", {}, function(data){
		if (data) {
			if(data.config_switch == "0"){
				OpenSelfConfig($("#self_config_switch").parent());
			}
			if(data.execute_type=="0"){
				$("#autoMode").prop("checked",true);
			}else{
				$("#manualMode").prop("checked",true);
			}
			var procedure = data.execute_procedure.split(",");
			if(procedure.includes("1")){
				$("#softUpgrade").prop("checked",true);
			}else{
				$("#softUpgrade").prop("checked",false);
			}
			if(procedure.includes("2")){
				$("#licenseIssued").prop("checked",true);
			}else{
				$("#licenseIssued").prop("checked",false);
			}
			if(procedure.includes("3")){
				$("#selfConfig").prop("checked",true);
			}else{
				$("#selfConfig").prop("checked",false);
			}
       } else {
			$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
      	    return;
       }
	}, "json")
})
try{
	if(writableMap && !writableMap['eNbSelfConfiguration']){
		$(':input').prop('disabled',true);
	}
}catch(e){}
//自开站开关控制
function OpenSelfConfig(ele){
	if(writableMap && !writableMap['eNbSelfConfiguration']){
		return false;
	}
	if ($(ele).children().attr('isopen') == 'false') {
		$(ele).children().attr('isopen','true').animate({left:'24px'},100);
		$(ele).css('background-color','#66CC66');
		$(".controlManaContent input,.controlManaContent label").attr("disabled",false);
	} else {
		$(".controlManaContent input,.controlManaContent label").attr("disabled","disabled");
		$(ele).children().attr('isopen','false').animate({left:'1px'},100);
        $(ele).css('background-color','#838383');
		$("#controlManaTip").html("");
	}	
}
//确定按钮
function saveSeleConfig(){
	var config_switch  = $("#self_config_switch").attr("isopen")=="true"?1:0;
	if(config_switch == 1){
		var selectedProcessNum = $("input[type=checkbox]").filter(':checked').length;
		if(selectedProcessNum>0){
			$("#controlManaTip").html("");
		}else{
			$("#controlManaTip").html("<%=rb.getString("ZhiShaoXuanZeYiXiang")%>");
			return;
		}
	}
	var execute_type = $("input:radio:checked").val();
	var execute_procedure = [];
	var procedure = $("input:checkbox:checked");
	for(var i=0;i<procedure.length;i++){
		execute_procedure.push($(procedure[i]).val());
	}
	execute_procedure = execute_procedure.join(",");
	var params={};
	params.config_switch = config_switch;
	params.execute_type = execute_type;
	params.execute_procedure = execute_procedure;
	$.post("${ctx}/cell/halobSelfConfig/saveSelfConfigRule.action", params, function(data){
		if (data.success) {
			$(".selfConfigSucTip").show();
			setTimeout('$(".selfConfigSucTip").fadeOut()',1000);
       } else {
			$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
      	    return;
       }
	}, "json")
}
</script>