<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_GERAN_CELL_PLMNID_name">${LTE_GERAN_CELL_PLMNID_name }</label>
		<input id="LTE_GERAN_CELL_PLMNID_name" name="LTE_GERAN_CELL_PLMNID" title="${LTE_GERAN_CELL_PLMNID_title }" class="border border-box" 
			min_length="5" max_length="6" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d{5,6}$/" must="1"/>
		<div id="LTE_GERAN_CELL_PLMNID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_CELL_PLMNID_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_CELL_LAC_name">${LTE_GERAN_CELL_LAC_name }</label>
		<input id="LTE_GERAN_CELL_LAC_name" name="LTE_GERAN_CELL_LAC" title="${LTE_GERAN_CELL_LAC_title }" class="border border-box" 
			min_value="1" max_value="65533" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_GERAN_CELL_LAC_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_CELL_LAC_title }
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_CELL_BSIC_name">${LTE_GERAN_CELL_BSIC_name }</label>
		<input id="LTE_GERAN_CELL_BSIC_name" name="LTE_GERAN_CELL_BSIC" title="${LTE_GERAN_CELL_BSIC_title }" class="border border-box" 
			min_value="0" max_value="255" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_GERAN_CELL_BSIC_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_CELL_BSIC_title }
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_CELL_CI_name">${LTE_GERAN_CELL_CI_name }</label>
		<input id="LTE_GERAN_CELL_CI_name" name="LTE_GERAN_CELL_CI" title="${LTE_GERAN_CELL_CI_title }" class="border border-box" 
			min_value="0" max_value="65535" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_GERAN_CELL_CI_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_CELL_CI_title }
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_CELL_BAND_INDICATOR_name">${LTE_GERAN_CELL_BAND_INDICATOR_name }</label>
		<select id="LTE_GERAN_CELL_BAND_INDICATOR_name" name="LTE_GERAN_CELL_BAND_INDICATOR" class="border border-box" onblur="validateRequired(event);createMML();">
			<option value=""></option>
			<option value="GSM850">GSM850</option>
			<option value="GSM900">GSM900</option>
			<option value="DCS1800">DCS1800</option>
			<option value="PCS1900">PCS1900</option>
		</select>
		<div id="LTE_GERAN_CELL_BAND_INDICATOR_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			<%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_CELL_BCCH_ARFCN_name">${LTE_GERAN_CELL_BCCH_ARFCN_name }</label>
		<input id="LTE_GERAN_CELL_BCCH_ARFCN_name" name="LTE_GERAN_CELL_BCCH_ARFCN" title="${LTE_GERAN_CELL_BCCH_ARFCN_title }" class="border border-box" 
			min_value="0" max_value="1023" onblur="validateMaxAndMinVal(event);createMML();" must="1"/>
		<div id="LTE_GERAN_CELL_BCCH_ARFCN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_GERAN_CELL_BCCH_ARFCN_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_GERAN_CELL_ENABLE_name">${LTE_GERAN_CELL_ENABLE_name }</label>
		<select id="LTE_GERAN_CELL_ENABLE_name" name="LTE_GERAN_CELL_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
</ul>
