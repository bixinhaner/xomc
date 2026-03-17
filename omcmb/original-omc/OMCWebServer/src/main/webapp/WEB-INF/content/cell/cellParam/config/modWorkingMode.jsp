<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_96_UE_ENABLE_name">${LTE_96_UE_ENABLE_name }</label>
		<select id="LTE_96_UE_ENABLE_name" name="LTE_96_UE_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
            <option value="1">Multi-user Mode(96UE)</option>
            <option value="0">Low Delay Mode(32UE)</option>
		</select>
	</li>
</ul>
