<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_B2_THRESHOLD1_UTRA_RSRP_HO_name">${LTE_B2_THRESHOLD1_UTRA_RSRP_HO_name }</label>
		<input id="LTE_B2_THRESHOLD1_UTRA_RSRP_HO_name" name="LTE_B2_THRESHOLD1_UTRA_RSRP_HO" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_B2_THRESHOLD1_UTRA_RSRP_HO_title }" 
			min_value="0" max_value="97" class="border border-box"/>
		<div id="LTE_B2_THRESHOLD1_UTRA_RSRP_HO_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_B2_THRESHOLD1_UTRA_RSRP_HO_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_B2_THRSH2_UTRATDD_RSCP_name">${LTE_TDS_CDMA_B2_THRSH2_UTRATDD_RSCP_name }</label>
		<input id="LTE_TDS_CDMA_B2_THRSH2_UTRATDD_RSCP_name" name="LTE_TDS_CDMA_B2_THRSH2_UTRATDD_RSCP" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_TDS_CDMA_B2_THRSH2_UTRATDD_RSCP_title }" 
			min_value="-5" max_value="91" class="border border-box"/>
		<div id="LTE_TDS_CDMA_B2_THRSH2_UTRATDD_RSCP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_B2_THRSH2_UTRATDD_RSCP_title }
		</div>
	</li>
	<li>
		<label for="LTE_B2_THRESHOLD1_GERAN_RSRP_IRAT_name">${LTE_B2_THRESHOLD1_GERAN_RSRP_IRAT_name }</label>
		<input id="LTE_B2_THRESHOLD1_GERAN_RSRP_IRAT_name" name="LTE_B2_THRESHOLD1_GERAN_RSRP_IRAT" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_B2_THRESHOLD1_GERAN_RSRP_IRAT_title }" 
			min_value="0" max_value="97" class="border border-box"/>
		<div id="LTE_B2_THRESHOLD1_GERAN_RSRP_IRAT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_B2_THRESHOLD1_GERAN_RSRP_IRAT_title }
		</div>
	</li>
	<li>
		<label for="LTE_B2_THRESHOLD2_GERAN_IRAT_name">${LTE_B2_THRESHOLD2_GERAN_IRAT_name }</label>
		<input id="LTE_B2_THRESHOLD2_GERAN_IRAT_name" name="LTE_B2_THRESHOLD2_GERAN_IRAT" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_B2_THRESHOLD2_GERAN_IRAT_title }" 
			min_value="0" max_value="63" class="border border-box"/>
		<div id="LTE_B2_THRESHOLD2_GERAN_IRAT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_B2_THRESHOLD2_GERAN_IRAT_title }
		</div>
	</li>
</ul>
