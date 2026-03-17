<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c" %>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_HSS_HALOBENABLE_name">${LTE_HSS_HALOBENABLE_name}</label>
		<select id="LTE_HSS_HALOBENABLE_name" name="LTE_HSS_HALOBENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">false</option>
			<option value="1">true</option>
		</select>
	</li>
	<li>
		<label for="LTE_EMBEDDED_EPC_BREAKOUT_TIME_name">${LTE_EMBEDDED_EPC_BREAKOUT_TIME_name }</label>
		<select id="LTE_EMBEDDED_EPC_BREAKOUT_TIME_name" name="LTE_EMBEDDED_EPC_BREAKOUT_TIME" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">1Day</option>
			<option value="2">2Day</option>
			<option value="3">3Day</option>
			<option value="5">5Day</option>
			<option value="0">7Day</option>
			<option value="10">10Day</option>
		</select>
	</li>
	
	<li id="halobModeLi">
		<label for="LTE_HALOB_MODE_name">${LTE_HALOB_MODE_name }</label>
		<select id="LTE_HALOB_MODE_name" name="LTE_HALOB_MODE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">Centralized</option>
			<option value="2">Single</option>
		</select>
		<div id="LTE_HALOB_MODE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_HALOB_MODE_title }
		</div>
	</li>
</ul>

<script type="text/javascript"> 
$(function(){
	$('#LTE_HALOB_ENABLE_STATE_name').bind('change',function(){
		if (this.value == "1") {
			$("#halobModeLi").css("display", "inline-block");
		} else {
			$("#halobModeLi").css("display", "none");
		}
	});
});
</script>