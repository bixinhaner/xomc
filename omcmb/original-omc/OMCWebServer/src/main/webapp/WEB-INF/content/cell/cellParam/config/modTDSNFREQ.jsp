<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="i_name">${i_name }</label>
		<input id="i_name" name="i" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${i_title }" 
			min_value="1" max_value="16" class="border border-box" must="1"/>
		<div id="i_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${i_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_UTRA_ARFCN_name">${LTE_TDS_CDMA_UTRA_ARFCN_name }</label>
		<input id="LTE_TDS_CDMA_UTRA_ARFCN_name" name="LTE_TDS_CDMA_UTRA_ARFCN" type="text" min_value="0" max_value="65535" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_UTRA_ARFCN_title }"/>
		<div id="LTE_TDS_CDMA_UTRA_ARFCN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_UTRA_ARFCN_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_QRX_LEV_MIN_name">${LTE_TDS_CDMA_QRX_LEV_MIN_name }</label>
		<input id="LTE_TDS_CDMA_QRX_LEV_MIN_name" name="LTE_TDS_CDMA_QRX_LEV_MIN" type="text" min_value="-60" max_value="-13" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_QRX_LEV_MIN_title }"/>
		<div id="LTE_TDS_CDMA_QRX_LEV_MIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_QRX_LEV_MIN_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_CELL_RESEL_PRIOR_name">${LTE_TDS_CDMA_CELL_RESEL_PRIOR_name }</label>
		<input id="LTE_TDS_CDMA_CELL_RESEL_PRIOR_name" name="LTE_TDS_CDMA_CELL_RESEL_PRIOR" type="text" min_value="0" max_value="7" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_CELL_RESEL_PRIOR_title }"/>
		<div id="LTE_TDS_CDMA_CELL_RESEL_PRIOR_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_CELL_RESEL_PRIOR_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_THRESH_X_HIGH_name">${LTE_TDS_CDMA_THRESH_X_HIGH_name }</label>
		<input id="LTE_TDS_CDMA_THRESH_X_HIGH_name" name="LTE_TDS_CDMA_THRESH_X_HIGH" type="text" min_value="0" max_value="31" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_THRESH_X_HIGH_title }"/>
		<div id="LTE_TDS_CDMA_THRESH_X_HIGH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_THRESH_X_HIGH_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_THRESH_X_LOW_name">${LTE_TDS_CDMA_THRESH_X_LOW_name }</label>
		<input id="LTE_TDS_CDMA_THRESH_X_LOW_name" name="LTE_TDS_CDMA_THRESH_X_LOW" type="text" min_value="0" max_value="31" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_THRESH_X_LOW_title }"/>
		<div id="LTE_TDS_CDMA_THRESH_X_LOW_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_THRESH_X_LOW_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_PMAX_UTRA_name">${LTE_TDS_CDMA_PMAX_UTRA_name }</label>
		<input id="LTE_TDS_CDMA_PMAX_UTRA_name" name="LTE_TDS_CDMA_PMAX_UTRA" type="text" min_value="-50" max_value="33" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_PMAX_UTRA_title }"/>
		<div id="LTE_TDS_CDMA_PMAX_UTRA_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_PMAX_UTRA_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_QOFFSET_UTRATDD_name">${LTE_TDS_CDMA_QOFFSET_UTRATDD_name }</label>
		<input id="LTE_TDS_CDMA_QOFFSET_UTRATDD_name" name="LTE_TDS_CDMA_QOFFSET_UTRATDD" type="text" min_value="-15" max_value="15" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_QOFFSET_UTRATDD_title }"/>
		<div id="LTE_TDS_CDMA_QOFFSET_UTRATDD_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_QOFFSET_UTRATDD_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_UTRATDD_BAND_INDICATOR_name">${LTE_TDS_CDMA_UTRATDD_BAND_INDICATOR_name }</label>
		<select id="LTE_TDS_CDMA_UTRATDD_BAND_INDICATOR_name" name="LTE_TDS_CDMA_UTRATDD_BAND_INDICATOR" value="1" class="border border-box" onblur="createMML();" must="1">
			<option value=""></option>
			<option value="1">A</option>
			<option value="2">B</option>
			<option value="3">C</option>
			<option value="4">D</option>
			<option value="5">E</option>
			<option value="6">F</option>
		</select>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_UTRATDD_MODE_name">${LTE_TDS_CDMA_UTRATDD_MODE_name }</label>
		<select id="LTE_TDS_CDMA_UTRATDD_MODE_name" name="LTE_TDS_CDMA_UTRATDD_MODE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="UTRA_TDD_128">UTRA_TDD_128</option>
			<option value="UTRA_TDD_384">UTRA_TDD_384</option>
			<option value="UTRA_TDD_768">UTRA_TDD_768</option>
		</select>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_FREQ_ENABLE_name">${LTE_TDS_CDMA_FREQ_ENABLE_name }</label>
		<select id="LTE_TDS_CDMA_FREQ_ENABLE_name" name="LTE_TDS_CDMA_FREQ_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
</ul>
