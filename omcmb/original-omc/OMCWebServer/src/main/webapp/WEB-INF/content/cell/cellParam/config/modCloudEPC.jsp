<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="TOGGLE_SWITCH_name">${TOGGLE_SWITCH_name }</label>
		<select id="TOGGLE_SWITCH_name" name="TOGGLE_SWITCH" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">enable</option>
			<option value="0">disable</option>
		</select>
	</li>
</ul>
