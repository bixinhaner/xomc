<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_ENABLE_X2_name">${LTE_ENABLE_X2_name }</label>
		<select id="LTE_ENABLE_X2_name" name="LTE_ENABLE_X2" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
</ul>
