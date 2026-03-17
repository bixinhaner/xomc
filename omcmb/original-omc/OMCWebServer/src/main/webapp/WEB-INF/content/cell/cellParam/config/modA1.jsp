<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_A1_THRESHOLD_RSRP_name">${LTE_A1_THRESHOLD_RSRP_name }</label>
		<input id="LTE_A1_THRESHOLD_RSRP_name" name="LTE_A1_THRESHOLD_RSRP" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_A1_THRESHOLD_RSRP_title }" 
			min_value="0" max_value="97" class="border border-box"/>
		<div id="LTE_A1_THRESHOLD_RSRP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_A1_THRESHOLD_RSRP_title }
		</div>
	</li>
	<c:if test='${hardwareVersion != "BAIBLX1.0"}'>
		<li>
			<label for="LTE_A1_HYSTERESIS_name">${LTE_A1_HYSTERESIS_name }</label>
			<input id="LTE_A1_HYSTERESIS_name" name="LTE_A1_HYSTERESIS" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_A1_HYSTERESIS_title }" 
				min_value="0" max_value="30" class="border border-box"/>
			<div id="LTE_A1_HYSTERESIS_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${LTE_A1_HYSTERESIS_title }
			</div>
		</li>
		<li>
			<label for="LTE_A1_TIME_TO_TRIGGER_name">${LTE_A1_TIME_TO_TRIGGER_name }</label>
			<select id="LTE_A1_TIME_TO_TRIGGER_name" name="LTE_A1_TIME_TO_TRIGGER" class="border border-box" onblur="createMML();">
				<option value=""></option>
				<option value="0">ms0</option>
				<option value="40">ms40</option>
				<option value="64">ms64</option>
				<option value="80">ms80</option>
				<option value="100">ms100</option>
				<option value="128">ms128</option>
				<option value="160">ms160</option>
				<option value="256">ms256</option>
				<option value="320">ms320</option>
				<option value="480">ms480</option>
				<option value="512">ms512</option>
				<option value="640">ms640</option>
				<option value="1024">ms1024</option>
				<option value="1280">ms1280</option>
				<option value="2560">ms2560</option>
				<option value="5120">ms5120</option>
			</select>
		</li>
	</c:if>
</ul>
