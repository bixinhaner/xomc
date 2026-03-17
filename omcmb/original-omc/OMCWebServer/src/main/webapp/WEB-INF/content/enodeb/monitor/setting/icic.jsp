<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.itemDiv{
		width:375px;
		height:92px;
		float:left;
		margin-right:45px;
	} 
	.promptTitle{
		height:24px;
		line-height:24px;
		min-width:50px;
		font-size:12px;
		color:#9FB318;
		display : none;
	}
	.errorTitle{
		height:24px;
		line-height:24px;
		min-width:50px;
		font-size:12px;
		color:red;
		display : none;
	}
	#intelTdd_compactParamConfig .itemDiv input,#intelTdd_compactParamConfig .itemDiv select{
		width:350px;
	}
	#intelTdd_compactParamConfig .itemDiv img{
		position:relative;
		top:4px;
		left:5px;
	}  
</style>
<script type="text/javascript">
$(function(){
	var sels = $("#icicConfig select");
	sels.bind("change",changeColor_select);
	
	var ipts = $("#icicConfig input[type='text']");
	ipts.bind("change",changeColor_input);
	
	$("#LTE_ICIC_SFR_ENABLE").val("${LTE_ICIC_SFR_ENABLE}");
	$("#LTE_CEU_REPORT_INTERVAL").val("${LTE_CEU_REPORT_INTERVAL}");
	$("#LTE_CEU_REPORT_AMOUNT").val("${LTE_CEU_REPORT_AMOUNT}");
	$("#LTE_ICIC_SFR_RB_RANGE").val("${LTE_ICIC_SFR_RB_RANGE}");
	$("#LTE_ICIC_SFR_POWER_DIV").val("${LTE_ICIC_SFR_POWER_DIV}");
	$("#LTE_ICIC_SFR_POWER_CEU").val("${LTE_ICIC_SFR_POWER_CEU}");
	
	var paAdjust = "${LTE_ICIC_SFR_POWER_DIV}";
	if("1" == paAdjust){
		$("#LTE_ICIC_SFR_POWER_CCU").parent().show();
		$("#LTE_ICIC_SFR_POWER_CEU").parent().show();
	}else{
		$("#LTE_ICIC_SFR_POWER_CCU").parent().hide();
		$("#LTE_ICIC_SFR_POWER_CEU").parent().hide();
	}
	
});

function changePAAdjust(e){
	var ele = $(e["target"]);
	var opts = ele.find("option");
	if("1" == ele.val()){
		$("#LTE_ICIC_SFR_POWER_CCU").parent().show();
		$("#LTE_ICIC_SFR_POWER_CEU").parent().show();
	}else{
		$("#LTE_ICIC_SFR_POWER_CCU").parent().hide();
		$("#LTE_ICIC_SFR_POWER_CEU").parent().hide();
	}
	
	var oldValue = ele.attr("oldValue");
	var oldValueText = "";
	var currVal = ele.find("option:selected").val();
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == oldValue){
			oldValueText = $(opts[i]).text();
		}
	}
    if(currVal!=oldValue){
    	$("#" +ele.attr("id")).css({"color":"blue"});
    	ele.siblings(".promptTitle").show();
    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValueText);
    }else{
    	$("#" +ele.attr("id")).css({"color":"black"});
    	ele.siblings(".promptTitle").hide();
    }
	
}

//下拉框选择内容发生变化
function icicSelectChange(e){
	var ele = $(e["target"]);
	var opts = ele.find("option");
	var oldValue = ele.attr("oldValue");
	var oldValueText = "";
	var currVal = ele.find("option:selected").val();
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == oldValue){
			oldValueText = $(opts[i]).text();
		}
	}
    if(currVal!=oldValue){
    	$("#" +ele.attr("id")).css({"color":"blue"});
    	ele.siblings(".promptTitle").show();
    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValueText);
    }else{
    	$("#" +ele.attr("id")).css({"color":"black"});
    	ele.siblings(".promptTitle").hide();
    }
}
//输入框内容发生变化
function icicInputChange(e){
	var ele = $(e["target"]);
    ele.siblings(".promptTitle").hide();
    validateMaxAndMinVal(e)
    if(!$("#" + ele.attr("id") + "_err").is(':visible')){
	    var currVal = ele.val();
	    var oldValue = ele.attr("oldValue");
	    if(currVal!=oldValue){
	    	$("#" +ele.attr("id")).css({"color":"blue"});
	    	ele.siblings(".promptTitle").show();
	    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
	    }else{
	    	$("#" +ele.attr("id")).css({"color":"black"});
	    	ele.siblings(".promptTitle").hide();
	    }
    }
}
</script>

	<div class="itemDiv">
		<span>SFR</span>
		<select name="LTE_ICIC_SFR_ENABLE"  id="LTE_ICIC_SFR_ENABLE" class="border border-box item" 
		        onchange="isICICConfigEnable(event);" oldValue="${LTE_ICIC_SFR_ENABLE }">
			<option value=""></option>
			<option value="1"><%=rb.getString("KeYong") %></option>
			<option value="0"><%=rb.getString("BuKeYong") %></option>
		</select>
		<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_ENABLE">
	</div>
	<c:if test="${LTE_ICIC_SFR_ENABLE == '1'}">
		<div id="icicDetailConfig" style="display: block;">
	</c:if>
	<c:if test="${LTE_ICIC_SFR_ENABLE != '1'}">
		<div id="icicDetailConfig" style="display: none;">
	</c:if>
	<div class="itemDiv">
		<span><%=rb.getString("XiaoQuBianYuanYongHuPanJuePianYiLiang")%></span>
		<input type="text" name="LTE_ICIC_CEU_PM_OFFSET" id="LTE_ICIC_CEU_PM_OFFSET" class="border border-box item" oldValue="${LTE_ICIC_CEU_PM_OFFSET}" value="${LTE_ICIC_CEU_PM_OFFSET}"
			onblur="icicInputChange(event)" min_value="-30" max_value="30"
			title="int, min value: -30 max value: 30"/>
		<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		<div class="errorTitle" id="LTE_ICIC_CEU_PM_OFFSET_err">int, min value: -30 max value: 30</div>
	</div>
	<div class="itemDiv">
		<span><%=rb.getString("XiaoQuBianYuanYongHuPanJueChiZhiZhi")%></span>
		<input type="text" name="LTE_ICIC_CEU_PM_HYSTERESIS" id="LTE_ICIC_CEU_PM_HYSTERESIS" class="border border-box item" oldValue="${LTE_ICIC_CEU_PM_HYSTERESIS}" value="${LTE_ICIC_CEU_PM_HYSTERESIS}"
			onblur="icicInputChange(event)" min_value="0" max_value="30"
			title="int, min value: 0 max value: 30"/>
		<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		<div class="errorTitle" id="LTE_ICIC_CEU_PM_HYSTERESIS_err">int, min value: 0 max value: 30</div>
	</div>
	<div class="itemDiv">
		<span><%=rb.getString("XiaoQuBainYuanYongHuCeLiangJianGe")%></span>
		<select name="LTE_CEU_REPORT_INTERVAL" id="LTE_CEU_REPORT_INTERVAL"  class="border border-box item" oldValue="${LTE_CEU_REPORT_INTERVAL }" onchange="icicSelectChange(event)">
			<option value=""></option>
			<option value="120">120ms</option>
			<option value="240">240ms</option>
			<option value="480">480ms</option>
			<option value="640">640ms</option>
			<option value="1024">1024ms</option>
			<option value="2048">2048ms</option>
			<option value="5120">5120ms</option>
			<option value="10240">10240ms</option>
			<option value="60000">1min</option>
			<option value="360000">6min</option>
			<option value="720000">12min</option>
			<option value="1800000">30min</option>
			<option value="3600000">60min</option>
		</select>
		<input type="hidden" value="" id="OLD_LTE_CEU_REPORT_INTERVAL">
		<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
	</div>
	<div class="itemDiv">
		<span><%=rb.getString("XiaoQuBianYuanYongHuCeLiangShuLiang")%></span>
		<select name="LTE_CEU_REPORT_AMOUNT" id="LTE_CEU_REPORT_AMOUNT" class="border border-box item" oldValue="${LTE_CEU_REPORT_AMOUNT }" onchange="icicSelectChange(event)">
			<option value=""></option>
			<option value="r1">r1</option>
			<option value="r2">r2</option>
			<option value="r4">r4</option>
			<option value="r8">r8</option>
			<option value="r16">r16</option>
			<option value="r32">r32</option>
			<option value="r64">r64</option>
			<option value="infinity">infinity</option>
		</select>
		<input type="hidden" value="" id="OLD_LTE_CEU_REPORT_AMOUNT">
		<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
	</div>
	<%-- 带宽20M --%>
	<c:if test="${bandWidth == 'n100' }">
		<div class="itemDiv">
			<span><%=rb.getString("BianYuanPinDaiFanWei")%></span>
			<select name="LTE_ICIC_SFR_RB_RANGE" id="LTE_ICIC_SFR_RB_RANGE" class="border border-box item" oldValue="${LTE_ICIC_SFR_RB_RANGE }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="0">0-31</option>
				<option value="1">32-63</option>
				<option value="2">64-99</option>
				<option value="3">0-23</option>
				<option value="4">24-46</option>
				<option value="5">47-71</option>
				<option value="6">72-99</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_RB_RANGE">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<%-- 带宽10M --%>
	<c:if test="${bandWidth == 'n50' }">
		<div class="itemDiv">
			<span><%=rb.getString("BianYuanPinDaiFanWei")%></span>
			<select name="LTE_ICIC_SFR_RB_RANGE" id="LTE_ICIC_SFR_RB_RANGE" class="border border-box item" oldValue="${LTE_ICIC_SFR_RB_RANGE }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="0">0-14</option>
				<option value="1">15-29</option>
				<option value="2">30-47</option>
				<option value="3">0-11</option>
				<option value="4">12-21</option>
				<option value="5">22-35</option>
				<option value="6">36-47</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_RB_RANGE">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<%-- PA可调 --%>
	<div class="itemDiv">
		<span><%=rb.getString("PAKeTiao")%></span>
		<select name="LTE_ICIC_SFR_POWER_DIV"  id="LTE_ICIC_SFR_POWER_DIV" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_DIV }" onchange="changePAAdjust(event);">
			<option value=""></option>
			<option value="1"><%=rb.getString("Kai") %></option>
			<option value="0"><%=rb.getString("Guan") %></option>
		</select>
		<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_DIV">
		<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
	</div>
	<%-- 小区中心用户PA --%>
	<div class="itemDiv">
		<span><%=rb.getString("XiaoQuZhongXinYongHuPA")%></span>
		<input type="text" name="LTE_ICIC_SFR_POWER_CCU" id="LTE_ICIC_SFR_POWER_CCU" class="border border-box item" 
			          oldValue="${LTE_ICIC_SFR_POWER_CCU}" value="${LTE_ICIC_SFR_POWER_CCU}" disabled="disabled"/>
	</div>
	<%-- 小区边缘用户PA --%>
	<c:if test="${LTE_ICIC_SFR_POWER_CCU == '0'}">
		<div class="itemDiv">
			<span><%=rb.getString("XiaoQuBianYuanYongHuPA")%></span>
			<select name="LTE_ICIC_SFR_POWER_CEU" id="LTE_ICIC_SFR_POWER_CEU" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_CEU }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="0">-6</option>
				<option value="1">-4.77</option>
				<option value="2">-3</option>
				<option value="3">-1.77</option>
				<option value="4">0</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_CEU">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<c:if test="${LTE_ICIC_SFR_POWER_CCU == '1'}">
		<div class="itemDiv">
			<span><%=rb.getString("XiaoQuBianYuanYongHuPA")%></span>
			<select name="LTE_ICIC_SFR_POWER_CEU" id="LTE_ICIC_SFR_POWER_CEU" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_CEU }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="1">-4.77</option>
				<option value="2">-3</option>
				<option value="3">-1.77</option>
				<option value="4">0</option>
				<option value="5">1</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_CEU">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<c:if test="${LTE_ICIC_SFR_POWER_CCU == '2'}">
		<div class="itemDiv">
			<span><%=rb.getString("XiaoQuBianYuanYongHuPA")%></span>
			<select name="LTE_ICIC_SFR_POWER_CEU" id="LTE_ICIC_SFR_POWER_CEU" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_CEU }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="2">-3</option>
				<option value="3">-1.77</option>
				<option value="4">0</option>
				<option value="5">1</option>
				<option value="6">2</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_CEU">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<c:if test="${LTE_ICIC_SFR_POWER_CCU == '3'}">
		<div class="itemDiv">
			<span><%=rb.getString("XiaoQuBianYuanYongHuPA")%></span>
			<select name="LTE_ICIC_SFR_POWER_CEU" id="LTE_ICIC_SFR_POWER_CEU" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_CEU }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="3">-1.77</option>
				<option value="4">0</option>
				<option value="5">1</option>
				<option value="6">2</option>
				<option value="7">3</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_CEU">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<c:if test="${LTE_ICIC_SFR_POWER_CCU == '4'}">
		<div class="itemDiv">
			<span><%=rb.getString("XiaoQuBianYuanYongHuPA")%></span>
			<select name="LTE_ICIC_SFR_POWER_CEU" id="LTE_ICIC_SFR_POWER_CEU" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_CEU }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="4">0</option>
				<option value="5">1</option>
				<option value="6">2</option>
				<option value="7">3</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_CEU">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<c:if test="${LTE_ICIC_SFR_POWER_CCU == '5'}">
		<div class="itemDiv">
			<span><%=rb.getString("XiaoQuBianYuanYongHuPA")%></span>
			<select name="LTE_ICIC_SFR_POWER_CEU" id="LTE_ICIC_SFR_POWER_CEU" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_CEU }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="5">1</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_CEU">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<c:if test="${LTE_ICIC_SFR_POWER_CCU == '6'}">
		<div class="itemDiv">
			<span><%=rb.getString("XiaoQuBianYuanYongHuPA")%></span>
			<select name="LTE_ICIC_SFR_POWER_CEU" id="LTE_ICIC_SFR_POWER_CEU" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_CEU }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="6">2</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_CEU">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	<c:if test="${LTE_ICIC_SFR_POWER_CCU == '7'}">
		<div class="itemDiv">
			<span><%=rb.getString("XiaoQuBianYuanYongHuPA")%></span>
			<select name="LTE_ICIC_SFR_POWER_CEU" id="LTE_ICIC_SFR_POWER_CEU" class="border border-box item" oldValue="${LTE_ICIC_SFR_POWER_CEU }" onchange="icicSelectChange(event)">
				<option value=""></option>
				<option value="7">3</option>
			</select>
			<input type="hidden" value="" id="OLD_LTE_ICIC_SFR_POWER_CEU">
			<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
		</div>
	</c:if>
	</div>