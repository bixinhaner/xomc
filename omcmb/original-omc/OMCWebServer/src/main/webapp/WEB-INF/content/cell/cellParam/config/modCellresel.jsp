<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_SINTRA_SEARCH_name">${LTE_SINTRA_SEARCH_name }</label>
		<input id="LTE_SINTRA_SEARCH_name" name="LTE_SINTRA_SEARCH" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_SINTRA_SEARCH_title }" 
			min_value="0" max_value="31" class="border border-box"/>
		<div id="LTE_SINTRA_SEARCH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SINTRA_SEARCH_title }
		</div>
	</li>
	<li>
		<label for="LTE_S_NON_INTRA_SEARCH_name">${LTE_S_NON_INTRA_SEARCH_name }</label>
		<input id="LTE_S_NON_INTRA_SEARCH_name" name="LTE_S_NON_INTRA_SEARCH" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_S_NON_INTRA_SEARCH_title }" 
			min_value="0" max_value="31" class="border border-box"/>
		<div id="LTE_S_NON_INTRA_SEARCH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_S_NON_INTRA_SEARCH_title }
		</div>
	</li>
	<li>
		<label for="LTE_QRX_LEVEL_MIN_SIB3_name">${LTE_QRX_LEVEL_MIN_SIB3_name }</label>
		<input id="LTE_QRX_LEVEL_MIN_SIB3_name" name="LTE_QRX_LEVEL_MIN_SIB3" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_QRX_LEVEL_MIN_SIB3_title }" 
			min_value="-70" max_value="-22" class="border border-box"/>
		<div id="LTE_QRX_LEVEL_MIN_SIB3_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_QRX_LEVEL_MIN_SIB3_title }
		</div>
	</li>
	<li>
		<label for="LTE_QHYST_name">${LTE_QHYST_name }</label>
		<select id="LTE_QHYST_name" name="LTE_QHYST" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">dB0</option>
			<option value="1">dB1</option>
			<option value="2">dB2</option>
			<option value="3">dB3</option>
			<option value="4">dB4</option>
			<option value="5">dB5</option>
			<option value="6">dB6</option>
			<option value="8">dB8</option>
			<option value="10">dB10</option>
			<option value="12">dB12</option>
			<option value="14">dB14</option>
			<option value="16">dB16</option>
			<option value="18">dB18</option>
			<option value="20">dB20</option>
			<option value="22">dB22</option>
			<option value="24">dB24</option>
		</select>
	</li>
	<li>
		<label for="LTE_CELL_RESELECTION_PRIORITY_name">${LTE_CELL_RESELECTION_PRIORITY_name }</label>
		<input id="LTE_CELL_RESELECTION_PRIORITY_name" name="LTE_CELL_RESELECTION_PRIORITY" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_CELL_RESELECTION_PRIORITY_title }" 
			min_value="0" max_value="7" class="border border-box"/>
		<div id="LTE_CELL_RESELECTION_PRIORITY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_CELL_RESELECTION_PRIORITY_title }
		</div>
	</li>
	<li>
		<label for="LTE_THRESH_SERVING_LOW_name">${LTE_THRESH_SERVING_LOW_name }</label>
		<input id="LTE_THRESH_SERVING_LOW_name" name="LTE_THRESH_SERVING_LOW" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_THRESH_SERVING_LOW_title }" 
			min_value="0" max_value="31" class="border border-box"/>
		<div id="LTE_THRESH_SERVING_LOW_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_THRESH_SERVING_LOW_title }
		</div>
	</li>
	<li>
		<label for="LTE_OAM_NEIGHBOUR_DL_BANDWIDTH_SIB3_name">${LTE_OAM_NEIGHBOUR_DL_BANDWIDTH_SIB3_name }</label>
		<select id="LTE_OAM_NEIGHBOUR_DL_BANDWIDTH_SIB3_name" name="LTE_OAM_NEIGHBOUR_DL_BANDWIDTH_SIB3" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="n6">CELL_BW_N6(1.4M)</option>
			<option value="n15">CELL_BW_N15(3M)</option>
			<option value="n25">CELL_BW_N25(5M)</option>
			<option value="n50">CELL_BW_N50(10M)</option>
			<option value="n75">CELL_BW_N75(15M)</option>
			<option value="n100">CELL_BW_N100(20M)</option>
		</select>
	</li>
</ul>
