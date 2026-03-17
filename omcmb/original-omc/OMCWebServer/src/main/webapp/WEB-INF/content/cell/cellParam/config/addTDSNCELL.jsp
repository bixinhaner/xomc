<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<%-- 小区扰码 --%>
	<li>
		<label for="LTE_TDS_CDMA_CELL_PARAM_ID_name">${LTE_TDS_CDMA_CELL_PARAM_ID_name }</label>
		<input id="LTE_TDS_CDMA_CELL_PARAM_ID_name" name="LTE_TDS_CDMA_CELL_PARAM_ID" type="text" min_value="1" max_value="127" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_TDS_CDMA_CELL_PARAM_ID_title }"/>
		<div id="LTE_TDS_CDMA_CELL_PARAM_ID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_CELL_PARAM_ID_title }
		</div>
	</li>
	<%-- 目标PLMN --%>
	<li>
		<label for="LTE_TDS_CDMA_PLMNID_name">${LTE_TDS_CDMA_PLMNID_name }</label>
		<input id="LTE_TDS_CDMA_PLMNID_name" name="LTE_TDS_CDMA_PLMNID" title="${LTE_TDS_CDMA_PLMNID_title }" class="border border-box" 
			min_length="5" max_length="6" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d{5,6}$/"/>
		<div id="LTE_TDS_CDMA_PLMNID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_PLMNID_title }
		</div>
	</li>
	<%-- 目标频点 --%>
	<li>
		<label for="LTE_TDS_CDMA_UARFCN_name">${LTE_TDS_CDMA_UARFCN_name }</label>
		<input id="LTE_TDS_CDMA_UARFCN_name" name="LTE_TDS_CDMA_UARFCN" title="${LTE_TDS_CDMA_UARFCN_title }" class="border border-box" 
			min_value="1" max_value="16383" onblur="validateMaxAndMinVal(event);createMML();" must="1"/>
		<div id="LTE_TDS_CDMA_UARFCN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_UARFCN_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<%-- 目标小区ID --%>
	<li>
		<label for="LTE_TDS_CDMA_CID_name">${LTE_TDS_CDMA_CID_name }</label>
		<input id="LTE_TDS_CDMA_CID_name" name="LTE_TDS_CDMA_CID" title="${LTE_TDS_CDMA_CID_title }" class="border border-box" 
			min_value="1" max_value=65535 onblur="validateMaxAndMinVal(event);createMML();" must="1"/>
		<div id="LTE_TDS_CDMA_CID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_CID_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<%-- 位置区编码 --%>
	<li>
		<label for="LTE_TDS_CDMA_LAC_name">${LTE_TDS_CDMA_LAC_name }</label>
		<input id="LTE_TDS_CDMA_LAC_name" name="LTE_TDS_CDMA_LAC" title="${LTE_TDS_CDMA_LAC_title }" class="border border-box" 
			min_value="0" max_value=65535 onblur="validateMaxAndMinVal(event);createMML();" must="1"/>
		<div id="LTE_TDS_CDMA_LAC_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_LAC_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<%-- RNC标识 --%>
	<li>
		<label for="LTE_TDS_CDMA_RNCID_name">${LTE_TDS_CDMA_RNCID_name }</label>
		<input id="LTE_TDS_CDMA_RNCID_name" name="LTE_TDS_CDMA_RNCID" title="${LTE_TDS_CDMA_RNCID_title }" class="border border-box" 
			min_value="0" max_value=65535 onblur="validateMaxAndMinVal(event);createMML();" js_regex="/^\d+$/"/>
		<div id="LTE_TDS_CDMA_RNCID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_TDS_CDMA_RNCID_title }
		</div>
	</li>
	<%-- 开关 --%>
	<li>
		<label for="LTE_TDS_CDMA_CELL_ENABLE_name">${LTE_TDS_CDMA_CELL_ENABLE_name }</label>
		<select id="LTE_TDS_CDMA_CELL_ENABLE_name" name="LTE_TDS_CDMA_CELL_ENABLE" class="border border-box" onblur="validateRequired(event);createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
		<div id="LTE_TDS_CDMA_CELL_ENABLE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			<%=rb.getString("BiTian") %>
		</div>
	</li>
</ul>
