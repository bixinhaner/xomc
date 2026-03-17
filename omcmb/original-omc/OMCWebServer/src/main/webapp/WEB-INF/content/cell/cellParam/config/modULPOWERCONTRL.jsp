<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_IS_UL_GRP_PWR_CONTROL_PUSCH_ENABLE_name">${LTE_IS_UL_GRP_PWR_CONTROL_PUSCH_ENABLE_name }</label>
		<select id="LTE_IS_UL_GRP_PWR_CONTROL_PUSCH_ENABLE_name" name="LTE_IS_UL_GRP_PWR_CONTROL_PUSCH_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_IS_UL_GRP_PWR_CONTROL_PUCCH_ENABLE_name">${LTE_IS_UL_GRP_PWR_CONTROL_PUCCH_ENABLE_name }</label>
		<select id="LTE_IS_UL_GRP_PWR_CONTROL_PUCCH_ENABLE_name" name="LTE_IS_UL_GRP_PWR_CONTROL_PUCCH_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_PUSCH_POWER_FORMAT_3_SIZE_name">${LTE_PUSCH_POWER_FORMAT_3_SIZE_name }</label>
		<input id="LTE_PUSCH_POWER_FORMAT_3_SIZE_name" name="LTE_PUSCH_POWER_FORMAT_3_SIZE" title="${LTE_PUSCH_POWER_FORMAT_3_SIZE_title }" class="border border-box" 
			min_value="1" max_value="100" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_PUSCH_POWER_FORMAT_3_SIZE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_PUSCH_POWER_FORMAT_3_SIZE_title }
		</div>
	</li>
	<li>
		<label for="LTE_PUSCH_POWER_FORMAT_3_RNTI_name">${LTE_PUSCH_POWER_FORMAT_3_RNTI_name }</label>
		<input id="LTE_PUSCH_POWER_FORMAT_3_RNTI_name" name="LTE_PUSCH_POWER_FORMAT_3_RNTI" title="${LTE_PUSCH_POWER_FORMAT_3_RNTI_title }" class="border border-box" 
			min_value="1" max_value="65523" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_PUSCH_POWER_FORMAT_3_RNTI_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_PUSCH_POWER_FORMAT_3_RNTI_title }
		</div>
	</li>
	<li>
		<label for="LTE_ACCUMULATION_ENABLE_name">${LTE_ACCUMULATION_ENABLE_name }</label>
		<select id="LTE_ACCUMULATION_ENABLE_name" name="LTE_ACCUMULATION_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
</ul>
