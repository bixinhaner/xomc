<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="RRC_WIRESHARK_ENABLE_name">${RRC_WIRESHARK_ENABLE_name }</label>
		<select id="RRC_WIRESHARK_ENABLE_name" name="RRC_WIRESHARK_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	
	<li>
		<label for="RRC_WIRESHARK_IP_name">${RRC_WIRESHARK_IP_name }</label>
		<input id="RRC_WIRESHARK_IP_name" name="RRC_WIRESHARK_IP" type="text" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" title="${RRC_WIRESHARK_IP_title }" 
			min_length="0" max_length="256" class="border border-box"
			js_regex="/^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/"/>
		<div id="RRC_WIRESHARK_IP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RRC_WIRESHARK_IP_title }
		</div>
	</li>
	<li>
		<label for="RRC_WIRESHARK_PORT_name">${RRC_WIRESHARK_PORT_name }</label>
		<input id="RRC_WIRESHARK_PORT_name" name="RRC_WIRESHARK_PORT" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${RRC_WIRESHARK_PORT_title }" 
			min_value="0" max_value=65535 class="border border-box"/>
		<div id="RRC_WIRESHARK_PORT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RRC_WIRESHARK_PORT_title }
		</div>
	</li>
	
</ul>
