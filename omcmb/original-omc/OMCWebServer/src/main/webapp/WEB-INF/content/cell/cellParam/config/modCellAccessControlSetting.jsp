<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_ARP_PREEMPT_ENABLE_name">${LTE_ARP_PREEMPT_ENABLE_name }</label>
		<select id="LTE_ARP_PREEMPT_ENABLE_name" name="LTE_ARP_PREEMPT_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	
	<li>
		<label for="LTE_US_MAX_UL_BROAD_BAND_BW_name">${LTE_US_MAX_UL_BROAD_BAND_BW_name }</label>
		<input id="LTE_US_MAX_UL_BROAD_BAND_BW_name" name="LTE_US_MAX_UL_BROAD_BAND_BW" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_US_MAX_UL_BROAD_BAND_BW_title }" 
			min_value="0" max_value="10000" class="border border-box"/>
		<div id="LTE_US_MAX_UL_BROAD_BAND_BW_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_US_MAX_UL_BROAD_BAND_BW_title }
		</div>
	</li>
	
	<li>
		<label for="LTE_US_MAX_DL_BROAD_BAND_BW_name">${LTE_US_MAX_DL_BROAD_BAND_BW_name }</label>
		<input id="LTE_US_MAX_DL_BROAD_BAND_BW_name" name="LTE_US_MAX_DL_BROAD_BAND_BW" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_US_MAX_DL_BROAD_BAND_BW_title }" 
			min_value="0" max_value="10000" class="border border-box"/>
		<div id="LTE_US_MAX_DL_BROAD_BAND_BW_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_US_MAX_DL_BROAD_BAND_BW_title }
		</div>
	</li>
	
	<li>
		<label for="LTE_MAX_NUM_GBR_BEARERS_name">${LTE_MAX_NUM_GBR_BEARERS_name }</label>
		<input id="LTE_MAX_NUM_GBR_BEARERS_name" name="LTE_MAX_NUM_GBR_BEARERS" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_MAX_NUM_GBR_BEARERS_title }" 
			min_value="0" max_value="255" class="border border-box"/>
		<div id="LTE_MAX_NUM_GBR_BEARERS_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_MAX_NUM_GBR_BEARERS_title }
		</div>
	</li>
</ul>
