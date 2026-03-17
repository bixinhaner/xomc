<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_SLOG_LVL_name">${LTE_SLOG_LVL_name }</label>
		<input id="LTE_SLOG_LVL_name" name="LTE_SLOG_LVL" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_SLOG_LVL_title }" 
			min_value="0" max_value="5" class="border border-box"/>
		<div id="LTE_SLOG_LVL_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SLOG_LVL_title }
		</div>
	</li>
</ul>
