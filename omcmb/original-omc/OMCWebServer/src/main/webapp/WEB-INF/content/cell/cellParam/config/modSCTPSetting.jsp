<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_S1SIG_LINK_PORT_name">${LTE_S1SIG_LINK_PORT_name }</label>
		<input id="LTE_S1SIG_LINK_PORT_name" name="LTE_S1SIG_LINK_PORT" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_S1SIG_LINK_PORT_title }" 
			min_value="0" max_value="65535" class="border border-box"/>
		<div id="LTE_S1SIG_LINK_PORT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_S1SIG_LINK_PORT_title }
		</div>
	</li>
</ul>
