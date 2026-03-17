<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="/common/alltaglibs.jsp" %>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_ANR_REPORT_CFG_VAL_name">${LTE_ANR_REPORT_CFG_VAL_name}</label>
		<select id="LTE_ANR_REPORT_CFG_VAL_name" name="LTE_ANR_REPORT_CFG_VAL" class="border border-box" onblur="createMML();" onchange="ANRReportChange(this)">
			<option value=""></option>
			<option value="0">No ANR</option>
			<option value="3">PERIOD ANR</option>
			<option value="4">EVENT ANR</option>
		</select>
	</li>
	<li>
		<label for="LTE_INTER_ANR_VALID_AGE_name">${LTE_INTER_ANR_VALID_AGE_name}</label>
		<input id="LTE_INTER_ANR_VALID_AGE_name" name="LTE_INTER_ANR_VALID_AGE" type="text" onblur="validateMaxAndMinVal(event);createMML();" 
			min_value="0" max_value="65535" title="${LTE_INTER_ANR_VALID_AGE_title }"  class="border border-box"/>
		<div id="LTE_INTER_ANR_VALID_AGE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_ANR_VALID_AGE_title }
		</div>
	</li>
	<li name="ANR">
		<label for="LTE_INTER_ANR_A5_THRESHOLD_1_RSRP_name">${LTE_INTER_ANR_A5_THRESHOLD_1_RSRP_name }</label>
		<input id="LTE_INTER_ANR_A5_THRESHOLD_1_RSRP_name" name="LTE_INTER_ANR_A5_THRESHOLD_1_RSRP" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_ANR_A3_OFFSET_title }" 
			min_value="0" max_value="97" class="border border-box"/>
		<div id="LTE_INTER_ANR_A5_THRESHOLD_1_RSRP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_ANR_A5_THRESHOLD_1_RSRP_title }
		</div>
	</li>
	<li name="ANR">
		<label for="LTE_INTER_ANR_A5_THRESHOLD_2_RSRP_name">${LTE_INTER_ANR_A5_THRESHOLD_2_RSRP_name }</label>
		<input id="LTE_INTER_ANR_A5_THRESHOLD_2_RSRP_name" name="LTE_INTER_ANR_A5_THRESHOLD_2_RSRP" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_ANR_A3_OFFSET_title }" 
			min_value="0" max_value="97" class="border border-box"/>
		<div id="LTE_INTER_ANR_A5_THRESHOLD_2_RSRP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_ANR_A5_THRESHOLD_2_RSRP_title }
		</div>
	</li>
	<li name="ANR">
		<label for="LTE_INTER_ANR_A3_OFFSET_name">${LTE_INTER_ANR_A3_OFFSET_name }</label>
		<input id="LTE_INTER_ANR_A3_OFFSET_name" name="LTE_INTER_ANR_A3_OFFSET" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_ANR_A3_OFFSET_title }" 
			min_value="-30" max_value="30" class="border border-box"/>
		<div id="LTE_INTER_ANR_A3_OFFSET_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_ANR_A3_OFFSET_title }
		</div>
	</li>
</ul>
<script>
	$(function(){
		ANRReportChange(document.querySelector('#LTE_ANR_REPORT_CFG_VAL_name'));
	    createMML();
	});
	
	function ANRReportChange(ele) {
		var code = $(ele).val();

	    if("4" == code){
	        $("li[name='ANR']").css("display","inline-block");
	    } else {
	        $("li[name='ANR']").hide();
	    }
	}
</script>