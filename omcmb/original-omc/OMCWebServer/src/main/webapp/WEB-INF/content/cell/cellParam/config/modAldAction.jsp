<%@ page language="java" contentType="text/html; charset=UTF-8"
	pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li><label for="ALD_OOK_ENABLE_name">${ALD_OOK_ENABLE_name }</label>
		<select id="ALD_OOK_ENABLE_name" name="ALD_OOK_ENABLE"
		class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">ON</option>
			<option value="2">OFF</option>
	</select></li>
	<li><label for="ALD_RS485_ENABLE_name">${ALD_RS485_ENABLE_name }</label>
		<select id="ALD_RS485_ENABLE_name" name="ALD_RS485_ENABLE"
		class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">ON</option>
			<option value="2">OFF</option>
	</select></li>
</ul>
