<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_PHY_PPSsyncGpsAdjSamp_name">${LTE_PHY_PPSsyncGpsAdjSamp_name }</label>
		<input id="LTE_PHY_PPSsyncGpsAdjSamp_name" name="LTE_PHY_PPSsyncGpsAdjSamp" title="${LTE_PHY_PPSsyncGpsAdjSamp_title }" class="border border-box" 
			min_value="-65535" max_value="65535" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_PHY_PPSsyncGpsAdjSamp_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_PHY_PPSsyncGpsAdjSamp_title }
		</div>
	</li>
	<li>
		<label for="LTE_PHY_ICTAadjSamp_name">${LTE_PHY_ICTAadjSamp_name }</label>
		<input id="LTE_PHY_ICTAadjSamp_name" name="LTE_PHY_ICTAadjSamp" title="${LTE_PHY_ICTAadjSamp_title }" class="border border-box" 
			min_value="-65535" max_value="65535" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_PHY_ICTAadjSamp_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_PHY_ICTAadjSamp_title }
		</div>
	</li>
</ul>
