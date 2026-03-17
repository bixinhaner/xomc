<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_WAN_MTU_name">${LTE_WAN_MTU_name }</label>
		<input id="LTE_WAN_MTU_name" name="LTE_WAN_MTU" type="text" min_value="1200" max_value="1600" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_WAN_MTU_title }"/>
		<div id="LTE_WAN_MTU_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_WAN_MTU_title }
		</div>
	</li>
	<li>
		<label for="LTE_WAN_CASCADE_name">${LTE_WAN_CASCADE_name }</label>
		<select id="LTE_WAN_CASCADE_name" name="LTE_WAN_CASCADE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">Enable</option>
			<option value="0">Disable</option>
		</select>
	</li>
</ul>