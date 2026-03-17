<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_WAN_CHECK_ENABLE_name">${LTE_WAN_CHECK_ENABLE_name }</label>
		<select id="LTE_WAN_CHECK_ENABLE_name" name="LTE_WAN_CHECK_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<%-- <label for="LTE_WAN_CHECK_TIMER_LEN_name">${LTE_WAN_CHECK_TIMER_LEN_name }</label>
		<input id="LTE_WAN_CHECK_TIMER_LEN_name" type="text" name = "LTE_WAN_CHECK_TIMER_LEN" min_value="1" max_value="3" title="${LTE_WAN_CHECK_TIMER_LEN_title }"
		onblur="validateMaxAndMinVal(event);createMML();" class="border border-box"/>
		<div id="LTE_WAN_CHECK_TIMER_LEN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">${LTE_WAN_CHECK_TIMER_LEN_title }</div>
	    --%>
		
		<label for="LTE_WAN_CHECK_TIMER_LEN_name">${LTE_WAN_CHECK_TIMER_LEN_name }</label>
		<select id="LTE_WAN_CHECK_TIMER_LEN_name" name="LTE_WAN_CHECK_TIMER_LEN" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">5</option>
			<option value="2">10</option>
			<option value="3">15</option>
		</select>
	</li>
</ul>
