<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_DLFSS_ENABLE_name">${LTE_DLFSS_ENABLE_name }</label>
		<select id="LTE_DLFSS_ENABLE_name" name="LTE_DLFSS_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_DL_SCHD_TYPE_name">${LTE_DL_SCHD_TYPE_name }</label>
		<select id="LTE_DL_SCHD_TYPE_name" name="LTE_DL_SCHD_TYPE" class="border border-box" onchange="dlSchdChange(this)" onblur="createMML();">
			<option value=""></option>
			<option value="1">PFS</option>
			<option value="2">RR</option>
		</select>
	</li>
	<li style="display: none;">
		<label for="LTE_PFS_CQI_FACTOR_name">${LTE_PFS_CQI_FACTOR_name }</label>
		<input id="LTE_PFS_CQI_FACTOR_name" name="LTE_PFS_CQI_FACTOR" class="border border-box" min_value="0" max_value="100" 
			onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_PFS_CQI_FACTOR_title }"/>
		<div id="LTE_PFS_CQI_FACTOR_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_PFS_CQI_FACTOR_title }
		</div>
	</li>
	<li>
		<label for="LTE_UL_SCHD_TYPE_name">${LTE_UL_SCHD_TYPE_name }</label>
		<select id="LTE_UL_SCHD_TYPE_name" name="LTE_UL_SCHD_TYPE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="2">RR</option>
		</select>
	</li>
	<li>
		<label for="LTE_ENA_64QAM_name">${LTE_ENA_64QAM_name }</label>
		<select id="LTE_ENA_64QAM_name" name="LTE_ENA_64QAM" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_PEAK_THP_ENABLE_name">${LTE_PEAK_THP_ENABLE_name }</label>
		<select id="LTE_PEAK_THP_ENABLE_name" name="LTE_PEAK_THP_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
</ul>
<script>
	function dlSchdChange(el) {
		var $dom = $(el),
			val = $dom.val();
		try{
			if(val == '1') $('#LTE_PFS_CQI_FACTOR_name').parent().show();
			else $('#LTE_PFS_CQI_FACTOR_name').parent().hide();
		}catch(e){}
	}
</script>