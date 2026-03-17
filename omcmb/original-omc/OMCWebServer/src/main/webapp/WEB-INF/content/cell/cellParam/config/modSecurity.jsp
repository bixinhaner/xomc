<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_CIPHERING_ALGO_LIST_name">${LTE_CIPHERING_ALGO_LIST_name }</label>
		<select id="LTE_CIPHERING_ALGO_LIST_name" name="LTE_CIPHERING_ALGO_LIST" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="128-EEA1">128-EEA1</option>
			<option value="128-EEA2">128-EEA2</option>
			<option value="128-EEA3">128-EEA3</option>
			<option value="EEA0">EEA0</option>
		</select>
	</li>
	<li>
		<label for="LTE_INTEGRITY_ALGO_LIST_name">${LTE_INTEGRITY_ALGO_LIST_name }</label>
		<select id="LTE_INTEGRITY_ALGO_LIST_name" name="LTE_INTEGRITY_ALGO_LIST" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="128-EIA1">128-EIA1</option>
			<option value="128-EIA2">128-EIA2</option>
			<option value="128-EIA3">128-EIA3</option>
		</select>
	</li>
</ul>
