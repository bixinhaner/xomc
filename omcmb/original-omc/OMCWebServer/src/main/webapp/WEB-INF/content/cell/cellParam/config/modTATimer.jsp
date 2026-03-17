<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_TIME_ALIGNMENT_TIMER_COMMON_name">${LTE_TIME_ALIGNMENT_TIMER_COMMON_name }</label>
		<select id="LTE_TIME_ALIGNMENT_TIMER_COMMON_name" name="LTE_TIME_ALIGNMENT_TIMER_COMMON" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">SF500</option>
			<option value="1">SF750</option>
			<option value="2">SF1280</option>
			<option value="3">SF1920</option>
			<option value="4">SF2560</option>
			<option value="5">SF5120</option>
			<option value="6">SF10240</option>
			<option value="7">INFINITY</option>
		</select>
	</li>
	<li>
		<label for="LTE_TIME_ALIGNMENT_TIMER_DEDICATED_name">${LTE_TIME_ALIGNMENT_TIMER_DEDICATED_name }</label>
		<select id="LTE_TIME_ALIGNMENT_TIMER_DEDICATED_name" name="LTE_TIME_ALIGNMENT_TIMER_DEDICATED" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">SF500</option>
			<option value="1">SF750</option>
			<option value="2">SF1280</option>
			<option value="3">SF1920</option>
			<option value="4">SF2560</option>
			<option value="5">SF5120</option>
			<option value="6">SF10240</option>
			<option value="7">INFINITY</option>
		</select>
	</li>
</ul>
