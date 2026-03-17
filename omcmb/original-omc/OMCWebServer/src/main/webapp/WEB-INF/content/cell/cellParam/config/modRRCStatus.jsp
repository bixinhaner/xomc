<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<%-- UE非激活状态定时器 --%>
	<li>
		<label for="LTE_UE_INACTIVITY_TIMER_VAL_name">${LTE_UE_INACTIVITY_TIMER_VAL_name }</label>
		<input id="LTE_UE_INACTIVITY_TIMER_VAL_name" name="LTE_UE_INACTIVITY_TIMER_VAL" title="${LTE_UE_INACTIVITY_TIMER_VAL_title }" class="border border-box" 
			min_value="0" max_value="4294967" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_UE_INACTIVITY_TIMER_VAL_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_UE_INACTIVITY_TIMER_VAL_title }
		</div>
	</li>
	<%-- 状态定时器最大超时次数 --%>
	<li>
		<label for="LTE_MAX_EXPIRY_COUNT_name">${LTE_MAX_EXPIRY_COUNT_name }</label>
		<input id="LTE_MAX_EXPIRY_COUNT_name" name="LTE_MAX_EXPIRY_COUNT" title="${LTE_MAX_EXPIRY_COUNT_title }" class="border border-box" 
			min_value="1" max_value="65535" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_MAX_EXPIRY_COUNT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_MAX_EXPIRY_COUNT_title }
		</div>
	</li>
</ul>
