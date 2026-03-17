<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="i_name">${i_name }</label>
		<input id="i_name" name="i" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${i_title }" 
			min_value="1" max_value="32" class="border border-box" must="1"/>
		<div id="i_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${i_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_CELL_PARAM_ID_name">${LTE_TDS_CDMA_CELL_PARAM_ID_name }</label>
		<input id="LTE_TDS_CDMA_CELL_PARAM_ID_name" name="LTE_TDS_CDMA_CELL_PARAM_ID" type="text" min_value="1" max_value="127" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_CELL_PARAM_ID_title }"/>
		<div id="LTE_TDS_CDMA_CELL_PARAM_ID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_CELL_PARAM_ID_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_PLMNID_name">${LTE_TDS_CDMA_PLMNID_name }</label>
		<input id="LTE_TDS_CDMA_PLMNID_name" name="LTE_TDS_CDMA_PLMNID" title="${LTE_TDS_CDMA_PLMNID_title }" class="border border-box" 
			min_length="5" max_length="6" onblur="validateMaxAndMinLength(event);createMML();" js_regex="/^\d{5,6}$/"/>
		<div id="LTE_TDS_CDMA_PLMNID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_PLMNID_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_UARFCN_name">${LTE_TDS_CDMA_UARFCN_name }</label>
		<input id="LTE_TDS_CDMA_UARFCN_name" name="LTE_TDS_CDMA_UARFCN" title="${LTE_TDS_CDMA_UARFCN_title }" class="border border-box" 
			min_value="1" max_value="16383" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_TDS_CDMA_UARFCN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_UARFCN_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_CID_name">${LTE_TDS_CDMA_CID_name }</label>
		<input id="LTE_TDS_CDMA_CID_name" name="LTE_TDS_CDMA_CID" title="${LTE_TDS_CDMA_CID_title }" class="border border-box" 
			min_value="1" max_value=65535 onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_TDS_CDMA_CID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_CID_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_LAC_name">${LTE_TDS_CDMA_LAC_name }</label>
		<input id="LTE_TDS_CDMA_LAC_name" name="LTE_TDS_CDMA_LAC" title="${LTE_TDS_CDMA_LAC_title }" class="border border-box" 
			min_value="0" max_value=65535 onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_TDS_CDMA_LAC_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_LAC_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_RNCID_name">${LTE_TDS_CDMA_RNCID_name }</label>
		<input id="LTE_TDS_CDMA_RNCID_name" name="LTE_TDS_CDMA_RNCID" title="${LTE_TDS_CDMA_RNCID_title }" class="border border-box" 
			min_value="0" max_value=65535 onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_TDS_CDMA_RNCID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_RNCID_title }
		</div>
	</li>
	<li>
		<label for="LTE_TDS_CDMA_CELL_ENABLE_name">${LTE_TDS_CDMA_CELL_ENABLE_name }</label>
		<select id="LTE_TDS_CDMA_CELL_ENABLE_name" name="LTE_TDS_CDMA_CELL_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
</ul>
