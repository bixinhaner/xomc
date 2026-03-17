<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_INTER_FREQ_MEAS_GAP_name">${LTE_INTER_FREQ_MEAS_GAP_name }</label>
		<input id="LTE_INTER_FREQ_MEAS_GAP_name" name="LTE_INTER_FREQ_MEAS_GAP" title="${LTE_INTER_FREQ_MEAS_GAP_title }" class="border border-box" 
			min_value="1" max_value="2" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_INTER_FREQ_MEAS_GAP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_FREQ_MEAS_GAP_title }
		</div>
	</li>
</ul>
