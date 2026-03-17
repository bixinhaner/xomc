<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="TR069_MANAGEMENT_SERVER_PORT_name">${TR069_MANAGEMENT_SERVER_PORT_name }</label>
		<input id="TR069_MANAGEMENT_SERVER_PORT_name" name="TR069_MANAGEMENT_SERVER_PORT" type="text" min_value="0" max_value="65535" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${TR069_MANAGEMENT_SERVER_PORT_title }"/>
		<div id="TR069_MANAGEMENT_SERVER_PORT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TR069_MANAGEMENT_SERVER_PORT_title }
		</div>
	</li>
	<li>
		<label for="TR069_SSL_ENABLE_name">${TR069_SSL_ENABLE_name }</label>
		<select id="TR069_SSL_ENABLE_name" name="TR069_SSL_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="TR069_INFORM_INTERVAL_name">${TR069_INFORM_INTERVAL_name }</label>
		<input id="TR069_INFORM_INTERVAL_name" name="TR069_INFORM_INTERVAL" type="text" min_value="0" max_value="65535"
			   onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${TR069_INFORM_INTERVAL_title }"/>
		<div id="TR069_INFORM_INTERVAL_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TR069_INFORM_INTERVAL_title }
		</div>
	</li>
</ul>
