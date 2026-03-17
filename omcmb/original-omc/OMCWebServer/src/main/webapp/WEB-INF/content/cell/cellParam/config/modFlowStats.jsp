<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_STATS_FLOW_SWITCH_name">${LTE_STATS_FLOW_SWITCH_name}</label>
		<select id="LTE_STATS_FLOW_SWITCH_name" name="LTE_STATS_FLOW_SWITCH" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">false</option>
			<option value="1">true</option>
		</select>
	</li>
	<li>
		<label for="LTE_STATS_FLOW_UPLOAD_INTERVAL_name">${LTE_STATS_FLOW_UPLOAD_INTERVAL_name }</label>
		<select id="LTE_STATS_FLOW_UPLOAD_INTERVAL_name" name="LTE_STATS_FLOW_UPLOAD_INTERVAL" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="15">15min</option>
			<option value="30">30min</option>
			<option value="60">60min</option>
			<option value="120">120min</option>
		</select>
	</li>
</ul>