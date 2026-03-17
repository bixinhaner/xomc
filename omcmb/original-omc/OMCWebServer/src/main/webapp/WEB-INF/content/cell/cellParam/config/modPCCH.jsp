<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_DEFAULT_PAGING_CYCLE_name">${LTE_DEFAULT_PAGING_CYCLE_name }</label>
		<select id="LTE_DEFAULT_PAGING_CYCLE_name" name="LTE_DEFAULT_PAGING_CYCLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">rf32</option>
			<option value="1">rf64</option>
			<option value="2">rf128</option>
			<option value="3">rf256</option>
		</select>
	</li>
</ul>
