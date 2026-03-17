<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style>		
.wirelessSetting{
	position : absolute;
	top : 50px;
	bottom : 10px;
	left : 20px;
	overflow : auto;
}
.switch {
	background-color: #66CC66;
	margin-left:0px;
}
.wirelessSetting select:disabled{
	background:#EAF1F4;
}
.itemDiv{
		width:375px;
		height:92px;
		float:left;
		/* margin-right:30px; */
		margin-left:40px;
} 
.clearBoth{
	float : none !important;
	clear : both;
	margin-right : 400px !important;
	/* height:50px !important; */
}
.selfConfigSucTip{
	display:inline-block;
	min-width:200px;
	height:38px;
	line-height:38px;
	padding : 0 15px 0 50px;
	margin-left:35px;
	color:#508D9B;
	font-size:16px;
	font-weight:bold;
	background:#E9FBFF url("${ctx}/images/success.png") no-repeat 16px center;
}
.errorTitle{
	height:26px;
	line-height:26px;
	min-width:50px;
	font-size:12px;
	color:red;
	display : none;
}
.saveTip{
	height:26px;
	line-height:26px;
	min-width:50px;
	font-size:12px;
	color:#4c6778;
	display:none;
}
</style>
<div class="slidebarTitleDiv">
	<ul class="slidebarTitleContainer" style="margin-left:0;padding-left:0;">
		<li class="default" id="commonHalobConfigTitle"></li>
	</ul>
	<div class="tableDiv titleIcon_close" onclick="closeCommonHalobConfig();" style="position:absolute;right:25px;top:15px;"></div>
</div>
<div class="wirelessSetting" style="padding: 20px 0 0 20px;display:block">
	<div style="height:560px;">
		<div class="itemDiv" style="">
			<span><%=rb.getString("ZhiChiPinDuan")%></span>
			<input type="text" name="halob_BANDS_SUPPORTED" id="halob_LTE_BANDS_SUPPORTED" class="inputDivCss border border-box item" 
		 		onblur="validateMaxAndMinVal(event)" min_value="1" max_value="62" must="1" 
		 		title="<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 1<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 62" />					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_LTE_BANDS_SUPPORTED_err"><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 1<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 62</div>	
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("DaiKuan")%></span>
			<select name="halob_UL_BANDWIDTH" id="halob_LTE_UL_BANDWIDTH" class="inputDivCss border border-box item">
				<option value="n25">5MHz</option>
				<option value="n50">10MHz</option>
				<option value="n75">15MHz</option>
				<option value="n100">20MHz</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="basic_LTE_BANDS_SUPPORTED_err"></div>	
		</div>
		<div class="itemDiv pciisLock">
			<span><%=rb.getString("PinLv")%>(MHz)</span>
			<input type="text" name="halob_UL_DL_EARFCN" id="halob_LTE_UL_DL_EARFCN" class="inputDivCss border border-box item" 
					onblur="validateMaxAndMinVal(event)" min_value="0" max_value="3800" must="1" 
					title="<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 3800"/>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_LTE_UL_DL_EARFCN_err" ><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 3800</div>
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("ZiZhenPeiBi")%></span>
			<select name="halob_TDD_SUBFRAME_ASSIGNMENT" id="halob_LTE_TDD_SUBFRAME_ASSIGNMENT" class="inputDivCss border border-box item">
				<option value="1">1(DL:UL = 2:2)</option>
				<option value="2">2(DL:UL = 3:1)</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_LTE_TDD_SUBFRAME_ASSIGNMENT_err"></div>	
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("TeShuZiZhenPeiBi")%></span>
			<select name="halob_TDD_SPECIAL_SUB_FRAME_PATTERNS" id="halob_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS" class="inputDivCss border border-box item">
				<option value="5">5</option>
				<option value="7">7</option>
			</select>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_err"></div>	
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("PLMN")%></span>
			<input type="text" name="halob_OAM_PLMNID" id="halob_LTE_OAM_PLMNID" class="inputDivCss border border-box item"
					onblur="validateMaxAndMinVal(event)"  min_value="10000" max_value="999999" must="1" 
					title="<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 10000<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 999999"/>					
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_LTE_OAM_PLMNID_err" ><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 10000<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 999999</div>			
		</div>
		<div class="itemDiv" style="">
			<span><%=rb.getString("TAC")%></span>
			<input type="text" name="halob_TAC"  id="halob_TAC"class="inputDivCss border border-box item" oldValue="${TAC}" value="${TAC}"
					onblur="validateByRegexAndRange(event);changeTipShowOrHide(event)" min_value="0" max_value="65535" must="1" 
					vali-regex="/^(\d+\.\.){0,1}(\d+)$/" 
					title="<%=rb.getString("LiRu")%>:'12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 65535"/>
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_TAC_err"><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 65535</div>	
			<div class="saveTip" id="halob_TAC_tip"><%=rb.getString("DanZhanPeiZhiBuGaiBian")%></div>
		</div>
		<div class="itemDiv" style="">
			<span><%=rb.getString("JiZhanID")%></span>
			<input type="text" name="halob_CELL_IDENTITY"  id="halob_CELL_IDENTITY"class="inputDivCss border border-box item" oldValue="${TAC}" value="${TAC}"
					onblur="validateByRegexAndRange(event);changeTipShowOrHide(event)" min_value="0" max_value="268435455" must="1" 
					vali-regex="/^(\d+\.\.){0,1}(\d+)$/" 
					title="<%=rb.getString("LiRu")%>:'12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 268435455"/>
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_CELL_IDENTITY_err"><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 268435455</div>	
			<div class="saveTip" id="halob_CELL_IDENTITY_tip"><%=rb.getString("DanZhanPeiZhiBuGaiBian")%></div>
		</div>
		<div class="itemDiv" style="">
			<span><%=rb.getString("PCI2")%></span>
			<input type="text" name="halob_PHY_CELLID_LIST" id="halob_PHY_CELLID_LIST" class="inputDivCss border border-box item" oldValue="${PHYCELLID}" value="${PHYCELLID}"
				onblur="validateByRegexAndRange(event);changeTipShowOrHide(event)" min_value="0" max_value="503" must="1" 
				vali-regex="/^(\d+\.\.){0,1}(\d+)$/"
				title="<%=rb.getString("LiRu")%>:'12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 503"/>
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_PHY_CELLID_LIST_err"><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 503</div>	
			<div class="saveTip" id="halob_PHY_CELLID_LIST_tip"><%=rb.getString("DanZhanPeiZhiBuGaiBian")%></div>
		</div>
		<div class="itemDiv" style="">
			<span><%=rb.getString("GenXuLieSuoYin")%></span>
			<input type="text" name="halob_ROOT_SEQ_INDEX"  id="halob_ROOT_SEQ_INDEX" class="inputDivCss border border-box item" 
				oldValue="${ROOT_SEQUENCE_INDEX}" value="${ROOT_SEQUENCE_INDEX}"
					onblur="validateByRegexAndRange(event);changeTipShowOrHide(event)" min_value="0" max_value="837" must="1" 
					vali-regex="/^(\d+\.\.){0,1}(\d+)$/"
					title="<%=rb.getString("LiRu")%>:'12..34'<%=rb.getString("DouHao")%><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 837"/>
			<img src="${ctx}/basic/images/newIcon/operatorIcon/lose_Focus.png" style="vertical-align:top;"/>
			<div class="errorTitle" id="halob_ROOT_SEQ_INDEX_err"><%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 837</div>	
			<div class="saveTip" id="halob_ROOT_SEQ_INDEX_tip"><%=rb.getString("DanZhanPeiZhiBuGaiBian")%></div>	
		</div>
		<div class="itemDiv" id="halobSwitchInput">
			<span><%=rb.getString("HaloBJiChuPeiZhi")%><%=rb.getString("KaiGuan")%></span>
			<input type="text" name="halob_HALOB_ENABLE_STATE" id="halob_LTE_HALOB_ENABLE_STATE_input" class="inputDivCss border border-box item"/>
			<div class="errorTitle" id="halob_LTE_HALOB_ENABLE_STATE_input_err"><%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 1<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 62</div>	
		</div>
		<div class="itemDiv" id="halobSwitchDiv">
			<span><%=rb.getString("HaloBJiChuPeiZhi")%></span>
			<div class="controlManaContent">
				<div class='switch' onclick="openHalobConfig(this)">
					<div isopen='true' id="halob_config_switch"  class='btnn' style='left:24px;'></div>
				</div>
			</div>
		</div>		
	</div>
	<div class="linkbuttonGroup" id="buttonGroupView_halob" style="margin-left:40px;margin-bottom:20px;"">
	    <a id="halobConfigSub" href="#" class="linkbutton linkbutton_trend" onclick="halobConfigSubmit(this)" ><span><%=rb.getString("QueDing")%></span></a>
	    <a href="#" class="linkbutton linkbutton_nowanna" onclick="closeCommonHalobConfig()"><span><%=rb.getString("QuXiao")%></span></a>
	</div>
	<a style="position:absolute;left:230px;display:none;" class="selfConfigSucTip" ><%=rb.getString("HaloBJiChuPeiZhiXiuGaiWanCheng")%></a>
</div>
<script type="text/javascript">

$(function() {
	if(isView == "true"){
		$("#halobSwitchInput").show();
		$("#halobSwitchDiv").hide();
		$("#buttonGroupView_halob").hide();
		$(".wirelessSetting img").hide();
		$(".wirelessSetting input,select").prop("disabled",true);
	}else{
		$("#halobSwitchInput").hide();
		$("#halobSwitchDiv").show();
	}
	closeLoading();
	//如果SAS开关打开，则该参数不可配置
	if(SASEnble == "1"){
		$("#halob_LTE_UL_BANDWIDTH,#halob_LTE_UL_DL_EARFCN").prop("disabled",true);
	}
	var selectedRow = $("#publicConfig").datagrid("getSelected");
	var params={};
	params.config_type = "2";  // config_type 1-基础配置，2-halob基础配置
	$.post("${ctx}/cell/halobSelfConfig/getCurrConfigParamContent.action", params, function(data){
		if(data){
			$("input[name='halob_BANDS_SUPPORTED']").val(data.bands_support);
			$("select[name='halob_UL_BANDWIDTH']").val(data.band_width);
			$("input[name='halob_UL_DL_EARFCN']").val(data.frequency).attr('oldValue',data.frequency);
			$("select[name='halob_TDD_SUBFRAME_ASSIGNMENT']").val(data.subframe_assignment);
			$("select[name='halob_TDD_SPECIAL_SUB_FRAME_PATTERNS']").val(data.special_subframe_patterns);
			$("input[name='halob_OAM_PLMNID']").val(data.plmn_id);
			$("input[name='halob_TAC']").val(data.tac).attr('oldvalue',data.tac);
			$("input[name='halob_CELL_IDENTITY']").val(data.cell_identity).attr('oldvalue',data.cell_identity);
			$("input[name='halob_PHY_CELLID_LIST']").val(data.phycellid).attr('oldvalue',data.phycellid);
			$("input[name='halob_ROOT_SEQ_INDEX']").val(data.root_sequence_index).attr('oldvalue',data.root_sequence_index);
			var switchStr = data.halob_enable == 0?'<%=rb.getString("GuanBi")%>':'<%=rb.getString("KaiQi")%>';
			$("input[name='halob_HALOB_ENABLE_STATE']").attr('sign',data.halob_enable).val(switchStr);
			if(data.halob_enable == "0"){
				openHalobConfig($("#halob_config_switch").parent());
			}
		}
	},"json");
})

function halob_commonChangeTip_support(e){
    validateMaxAndMinVal(e);
	if ($("#halob_LTE_BANDS_SUPPORTED").val().trim()=="") {
		$("#halob_LTE_BANDS_SUPPORTED").addClass("err_border");
	}
}
function halob_commonChangeTip_plmn(e){
    validateMaxAndMinVal(e);
	if ($("#halob_LTE_OAM_PLMNID").val().trim()=="") {
		$("#halob_LTE_OAM_PLMNID").addClass("err_border");
	}
}
function halob_validateMaxAndMinVal1(e) {
    validateMaxAndMinVal(e);
    if ($("#halob_LTE_UL_DL_EARFCN").val().trim()=="") {
		$("#halob_LTE_UL_DL_EARFCN").addClass("err_border");
	}
}

//halob基础开关控制
function openHalobConfig(ele){
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
$("#commonHalobConfigViewOrModify input").blur(function(){
	if($(this).val()==""){
		$(this).addClass('err_border');
	}
})
//halob基础配置修改提交
function halobConfigSubmit(){
	var allInput = $("#commonHalobConfigViewOrModify input");
	for(var i=0;i<allInput.length;i++){
		if($(allInput[i]).val()==""){
			$(allInput[i]).addClass('err_border');
		};
	}
	if ($("#commonHalobConfigViewOrModify .item.err_border").length > 0) {
		var positionTop =parseInt($($("#commonHalobConfigViewOrModify .item.err_border")[0]).offset().top-0);
 		$("#commonHalobConfigViewOrModify").animate({scrollTop:positionTop},200);
		return;
	}
	var paramMap = {};
	paramMap.LTE_BANDS_SUPPORTED = $("#halob_LTE_BANDS_SUPPORTED").val().trim();
	paramMap.LTE_DL_BANDWIDTH = $("select[name='halob_UL_BANDWIDTH']").val();
	paramMap.LTE_DL_EARFCN = $("#halob_LTE_UL_DL_EARFCN").val().trim();
	paramMap.LTE_TDD_SUBFRAME_ASSIGNMENT = $("select[name='halob_TDD_SUBFRAME_ASSIGNMENT']").val();
	paramMap.LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS = $("select[name='halob_TDD_SPECIAL_SUB_FRAME_PATTERNS']").val();
	paramMap.LTE_OAM_PLMNID = $("#halob_LTE_OAM_PLMNID").val().trim();
	paramMap.LTE_HALOB_ENABLE_STATE = $("#halob_config_switch").attr("isopen")== "true" ?1:0;;
	paramMap.LTE_TAC = $("#halob_TAC").val() ;
	paramMap.LTE_CELL_IDENTITY = $("#halob_CELL_IDENTITY").val() ;
	paramMap.LTE_PHY_CELLID_LIST = $("#halob_PHY_CELLID_LIST").val() ;
	paramMap.LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST = $("#halob_ROOT_SEQ_INDEX").val() ;
	paramMap=JSON.stringify(paramMap);
	var params={};
	params.paramMap = paramMap;
	params.config_type = "2";
	$.post("${ctx}/cell/halobSelfConfig/saveAllConfigParamValue.action", params, function(data){
		if(data.success){
			$(".selfConfigSucTip").show();
			setTimeout('$(".selfConfigSucTip").fadeOut()',1000);
			$("#publicConfig").datagrid("reload");
			setTimeout('closeCommonHalobConfig()',1000);
		}else{
			$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
      	    return;
		}
	},"json");
}
//变化时提示单站配置需要手工修改
function changeTipShowOrHide(e) {
	var ele = $(e["target"]);
	if(!ele.hasClass("err_border") && ele.val() != ele.attr('oldvalue')){
        $("#" + ele.attr("id") + "_tip").show();
	}else{
        $("#" + ele.attr("id") + "_tip").hide();
	}
}
</script>