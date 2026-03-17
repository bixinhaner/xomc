<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LOCAL_WEB_LOCK_TIMEOUT_name">${LOCAL_WEB_LOCK_TIMEOUT_name }</label>
		<input id="LOCAL_WEB_LOCK_TIMEOUT_name" name="LOCAL_WEB_LOCK_TIMEOUT" title="${LOCAL_WEB_LOCK_TIMEOUT_title }" class="border border-box" 
			min_value="0" max_value="60" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LOCAL_WEB_LOCK_TIMEOUT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LOCAL_WEB_LOCK_TIMEOUT_title }
		</div>
	</li>
</ul>
