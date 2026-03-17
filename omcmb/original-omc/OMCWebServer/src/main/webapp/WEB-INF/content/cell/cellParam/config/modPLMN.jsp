<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="i_name">${i_name }</label>
		<input id="i_name" name="i" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${i_title }" 
			min_value="1" max_value="6" class="border border-box" must="1"/>
		<div id="i_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${i_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_OAM_PLMNID_name">${LTE_OAM_PLMNID_name }</label>
		<input id="LTE_OAM_PLMNID_name" name="LTE_OAM_PLMNID" title="${LTE_OAM_PLMNID_title }" class="border border-box" 
			min_length="5" max_length="6" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d{5,6}$/"/>
		<div id="LTE_OAM_PLMNID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_OAM_PLMNID_title }
		</div>
	</li>
	<li>
		<label for="LTE_OAM_PRIMARY_PLMN_name">${LTE_OAM_PRIMARY_PLMN_name }</label>
		<select id="LTE_OAM_PRIMARY_PLMN_name" name="LTE_OAM_PRIMARY_PLMN" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="true">true</option>
			<option value="false">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_EPC_PLMN_ENABLE_name">${LTE_EPC_PLMN_ENABLE_name }</label>
		<select id="LTE_EPC_PLMN_ENABLE_name" name="LTE_EPC_PLMN_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="true">true</option>
			<option value="false">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_OAM_CELL_RESERVE_FOR_OPERATOR_name">${LTE_OAM_CELL_RESERVE_FOR_OPERATOR_name }</label>
		<select id="LTE_OAM_CELL_RESERVE_FOR_OPERATOR_name" name="LTE_OAM_CELL_RESERVE_FOR_OPERATOR" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="true">true</option>
			<option value="false">false</option>
		</select>
	</li>
</ul>
