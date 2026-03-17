<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_QRX_LEVEL_MIN_SIB1_name">${LTE_QRX_LEVEL_MIN_SIB1_name }</label>
		<input id="LTE_QRX_LEVEL_MIN_SIB1_name" name="LTE_QRX_LEVEL_MIN_SIB1" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_QRX_LEVEL_MIN_SIB1_title }" 
			min_value="-70" max_value="-22" class="border border-box"/>
		<div id="LTE_QRX_LEVEL_MIN_SIB1_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_QRX_LEVEL_MIN_SIB1_title }
		</div>
	</li>
	<li>
		<label for="LTE_QRX_LEVEL_MIN_OFFSET_name">${LTE_QRX_LEVEL_MIN_OFFSET_name }</label>
		<input id="LTE_QRX_LEVEL_MIN_OFFSET_name" name="LTE_QRX_LEVEL_MIN_OFFSET" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_QRX_LEVEL_MIN_OFFSET_title }" 
			min_value="1" max_value="8" class="border border-box"/>
		<div id="LTE_QRX_LEVEL_MIN_OFFSET_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_QRX_LEVEL_MIN_OFFSET_title }
		</div>
	</li>
</ul>
