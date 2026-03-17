<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LOCAL_WEB_HTTP_PORT_name">${LOCAL_WEB_HTTP_PORT_name }</label>
		<input id="LOCAL_WEB_HTTP_PORT_name" name="LOCAL_WEB_HTTP_PORT" type="text" onblur="validateByRegex(event);createMML();" title="${LOCAL_WEB_HTTP_PORT_title }" 
			vali-regex="/^20\d{3}$|^21000$|^80$/" class="border border-box"/>
		<div id="LOCAL_WEB_HTTP_PORT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LOCAL_WEB_HTTP_PORT_title }
		</div>
	</li>
</ul>
