<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_GERAN_FREQ_BAND_INDICATOR_name">${LTE_GERAN_FREQ_BAND_INDICATOR_name }</label>
		<select id="LTE_GERAN_FREQ_BAND_INDICATOR_name" name="LTE_GERAN_FREQ_BAND_INDICATOR" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="GSM850">GSM850</option>
			<option value="GSM900">GSM900</option>
			<option value="DCS1800">DCS1800</option>
			<option value="PCS1900">PCS1900</option>
		</select>
	</li>
	<li>
		<label for="LTE_GERAN_FREQ_BCCH_ARFCN_name">${LTE_GERAN_FREQ_BCCH_ARFCN_name }</label>
		<input id="LTE_GERAN_FREQ_BCCH_ARFCN_name" name="LTE_GERAN_FREQ_BCCH_ARFCN" type="text" min_value="0" max_value="1023" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_GERAN_FREQ_BCCH_ARFCN_title }" must="1"/>
		<div id="LTE_GERAN_FREQ_BCCH_ARFCN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_FREQ_BCCH_ARFCN_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_FREQ_QRX_LEV_MIN_name">${LTE_GERAN_FREQ_QRX_LEV_MIN_name }</label>
		<input id="LTE_GERAN_FREQ_QRX_LEV_MIN_name" name="LTE_GERAN_FREQ_QRX_LEV_MIN" type="text" min_value="0" max_value="45" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_GERAN_FREQ_QRX_LEV_MIN_title }"/>
		<div id="LTE_GERAN_FREQ_QRX_LEV_MIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_FREQ_QRX_LEV_MIN_title }
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_FREQ_CELL_RESEL_PRIOR_name">${LTE_GERAN_FREQ_CELL_RESEL_PRIOR_name }</label>
		<input id="LTE_GERAN_FREQ_CELL_RESEL_PRIOR_name" name="LTE_GERAN_FREQ_CELL_RESEL_PRIOR" type="text" min_value="0" max_value="7" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_GERAN_FREQ_CELL_RESEL_PRIOR_title }"/>
		<div id="LTE_GERAN_FREQ_CELL_RESEL_PRIOR_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_FREQ_CELL_RESEL_PRIOR_title }
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_FREQ_THRESHX_HIGH_name">${LTE_GERAN_FREQ_THRESHX_HIGH_name }</label>
		<input id="LTE_GERAN_FREQ_THRESHX_HIGH_name" name="LTE_GERAN_FREQ_THRESHX_HIGH" type="text" min_value="0" max_value="31" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_GERAN_FREQ_THRESHX_HIGH_title }"/>
		<div id="LTE_GERAN_FREQ_THRESHX_HIGH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_FREQ_THRESHX_HIGH_title }
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_FREQ_THRESHX_LOW_name">${LTE_GERAN_FREQ_THRESHX_LOW_name }</label>
		<input id="LTE_GERAN_FREQ_THRESHX_LOW_name" name="LTE_GERAN_FREQ_THRESHX_LOW" type="text" min_value="0" max_value="31" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_GERAN_FREQ_THRESHX_LOW_title }"/>
		<div id="LTE_GERAN_FREQ_THRESHX_LOW_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_FREQ_THRESHX_LOW_title }
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_FREQ_ENABLE_name">${LTE_GERAN_FREQ_ENABLE_name }</label>
		<select id="LTE_GERAN_FREQ_ENABLE_name" name="LTE_GERAN_FREQ_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
</ul>
