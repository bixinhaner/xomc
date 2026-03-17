<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_PHY_RXGAIN_name">${LTE_PHY_RXGAIN_name }</label>
		<input id="LTE_PHY_RXGAIN_name" name="LTE_PHY_RXGAIN" title="${LTE_PHY_RXGAIN_title }" class="border border-box" 
			min_value="-48" max_value="76" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_PHY_RXGAIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_PHY_RXGAIN_title }
		</div>
	</li>
</ul>
