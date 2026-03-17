<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for=LTE_INTER_FREQ_DL_EARFCN_name>${LTE_INTER_FREQ_DL_EARFCN_name }</label>
		<input id="LTE_INTER_FREQ_DL_EARFCN_name" name="LTE_INTER_FREQ_DL_EARFCN" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_FREQ_DL_EARFCN_title }" 
			min_value="0" max_value="65535" class="border border-box" must="1"/>
		<div id="LTE_INTER_FREQ_DL_EARFCN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_FREQ_DL_EARFCN_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for=LTE_INTER_FREQ_QRX_LEVEL_MIN_SIB5_name>${LTE_INTER_FREQ_QRX_LEVEL_MIN_SIB5_name }</label>
		<input id="LTE_INTER_FREQ_QRX_LEVEL_MIN_SIB5_name" name="LTE_INTER_FREQ_QRX_LEVEL_MIN_SIB5" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_FREQ_QRX_LEVEL_MIN_SIB5_title }" 
			min_value="-70" max_value="-22" class="border border-box"/>
		<div id="LTE_INTER_FREQ_QRX_LEVEL_MIN_SIB5_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_FREQ_QRX_LEVEL_MIN_SIB5_title }
		</div>
	</li>
	<li> 
		<label for="LTE_INTER_FREQ_QOFFSETFREQ_name">${LTE_INTER_FREQ_QOFFSETFREQ_name }</label>
		<select id="LTE_INTER_FREQ_QOFFSETFREQ_name" name="LTE_INTER_FREQ_QOFFSETFREQ" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="-24">dB-24</option>
			<option value="-22">dB-22</option>
			<option value="-20">dB-20</option>
			<option value="-18">dB-18</option>
			<option value="-16">dB-16</option>
			<option value="-14">dB-14</option>
			<option value="-12">dB-12</option>
			<option value="-10">dB-10</option>
			<option value="-8">dB-8</option>
			<option value="-6">dB-6</option>
			<option value="-5">dB-5</option>
			<option value="-4">dB-4</option>
			<option value="-3">dB-3</option>
			<option value="-2">dB-2</option>
			<option value="-1">dB-1</option>
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
		<label for=LTE_INTER_FREQ_TRESELECTION_EUTRA_name>${LTE_INTER_FREQ_TRESELECTION_EUTRA_name }</label>
		<input id="LTE_INTER_FREQ_TRESELECTION_EUTRA_name" name="LTE_INTER_FREQ_TRESELECTION_EUTRA" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_FREQ_TRESELECTION_EUTRA_title }" 
			min_value="0" max_value="7" class="border border-box"/>
		<div id="LTE_INTER_FREQ_TRESELECTION_EUTRA_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_FREQ_TRESELECTION_EUTRA_title }
		</div>
	</li>
	<li>
		<label for=LTE_INTER_FREQ_CELL_RESELECT_PRIORITY_name>${LTE_INTER_FREQ_CELL_RESELECT_PRIORITY_name }</label>
		<input id="LTE_INTER_FREQ_CELL_RESELECT_PRIORITY_name" name="LTE_INTER_FREQ_CELL_RESELECT_PRIORITY" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_FREQ_CELL_RESELECT_PRIORITY_title }" 
			min_value="0" max_value="7" class="border border-box"/>
		<div id="LTE_INTER_FREQ_CELL_RESELECT_PRIORITY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_FREQ_CELL_RESELECT_PRIORITY_title }
		</div>
	</li>
	<li>
		<label for=LTE_INTER_FREQ_THRESHOLD_XHIGH_name>${LTE_INTER_FREQ_THRESHOLD_XHIGH_name }</label>
		<input id="LTE_INTER_FREQ_THRESHOLD_XHIGH_name" name="LTE_INTER_FREQ_THRESHOLD_XHIGH" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_FREQ_THRESHOLD_XHIGH_title }" 
			min_value="0" max_value="31" class="border border-box"/>
		<div id="LTE_INTER_FREQ_THRESHOLD_XHIGH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_FREQ_THRESHOLD_XHIGH_title }
		</div>
	</li>
	<li>
		<label for=LTE_INTER_FREQ_THRESHOLD_XLOW_name>${LTE_INTER_FREQ_THRESHOLD_XLOW_name }</label>
		<input id="LTE_INTER_FREQ_THRESHOLD_XLOW_name" name="LTE_INTER_FREQ_THRESHOLD_XLOW" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_FREQ_THRESHOLD_XLOW_title }" 
			min_value="0" max_value="31" class="border border-box"/>
		<div id="LTE_INTER_FREQ_THRESHOLD_XLOW_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_FREQ_THRESHOLD_XLOW_title }
		</div>
	</li>
	<li>
		<label for=LTE_INTER_FREQ_PMAX_name>${LTE_INTER_FREQ_PMAX_name }</label>
		<input id="LTE_INTER_FREQ_PMAX_name" name="LTE_INTER_FREQ_PMAX" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_INTER_FREQ_PMAX_title }" 
			min_value="-30" max_value="33" class="border border-box"/>
		<div id="LTE_INTER_FREQ_PMAX_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_INTER_FREQ_PMAX_title }
		</div>
	</li>
	<li>
		<label for="LTE_INTER_FREQ_ENABLE_name">${LTE_INTER_FREQ_ENABLE_name }</label>
		<select id="LTE_INTER_FREQ_ENABLE_name" name="LTE_INTER_FREQ_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
</ul>
