<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<%-- CQI周期 --%>
	<li>
		<label for="LTE_RRM_CQI_PRDCTY_name">${LTE_RRM_CQI_PRDCTY_name }</label>
		<input id="LTE_RRM_CQI_PRDCTY_name" name="LTE_RRM_CQI_PRDCTY" title="${LTE_RRM_CQI_PRDCTY_title }" class="border border-box" 
			min_value="0" max_value="9" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_RRM_CQI_PRDCTY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_RRM_CQI_PRDCTY_title }
		</div>
	</li>
	<%-- 每TTI内的CQI数目 --%>
	<li>
		<label for="LTE_RRM_NUM_CQI_PER_TTI_name">${LTE_RRM_NUM_CQI_PER_TTI_name }</label>
		<input id="LTE_RRM_NUM_CQI_PER_TTI_name" name="LTE_RRM_NUM_CQI_PER_TTI" title="${LTE_RRM_NUM_CQI_PER_TTI_title }" class="border border-box" 
			min_value="0" max_value="1176" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_RRM_NUM_CQI_PER_TTI_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_RRM_NUM_CQI_PER_TTI_title }
		</div>
	</li>
	<%-- SR周期 --%>
	<li>
		<label for="LTE_RRM_SR_PRDCTY_name">${LTE_RRM_SR_PRDCTY_name }</label>
		<input id="LTE_RRM_SR_PRDCTY_name" name="LTE_RRM_SR_PRDCTY" title="${LTE_RRM_SR_PRDCTY_title }" class="border border-box" 
			min_value="1" max_value="4" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_RRM_SR_PRDCTY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_RRM_SR_PRDCTY_title }
		</div>
	</li>
	<%-- MAC RNTI数目 --%>
	<li>
		<label for="LTE_MAX_MAC_RNTIS_name">${LTE_MAX_MAC_RNTIS_name }</label>
		<input id="LTE_MAX_MAC_RNTIS_name" name="LTE_MAX_MAC_RNTIS" title="${LTE_MAX_MAC_RNTIS_title }" class="border border-box" 
			min_value="0" max_value="381" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_MAX_MAC_RNTIS_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_MAX_MAC_RNTIS_title }
		</div>
	</li>
</ul>
