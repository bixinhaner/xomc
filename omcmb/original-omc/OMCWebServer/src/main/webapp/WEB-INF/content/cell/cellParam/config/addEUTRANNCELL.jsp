<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_NEIGH_LIST_LTE_CELL_PLMNID_name">${LTE_NEIGH_LIST_LTE_CELL_PLMNID_name }</label>
		<input id="LTE_NEIGH_LIST_LTE_CELL_PLMNID_name" name="LTE_NEIGH_LIST_LTE_CELL_PLMNID" title="${LTE_NEIGH_LIST_LTE_CELL_PLMNID_title }" class="border border-box" 
			min_length="5" max_length="6" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d{5,6}$/" must="1"/>
		<div id="LTE_NEIGH_LIST_LTE_CELL_PLMNID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_NEIGH_LIST_LTE_CELL_PLMNID_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_NEIGH_LIST_LTE_CELL_CID_name">${LTE_NEIGH_LIST_LTE_CELL_CID_name }</label>
		<input id="LTE_NEIGH_LIST_LTE_CELL_CID_name" name="LTE_NEIGH_LIST_LTE_CELL_CID" type="text" min_value="0" max_value="268435455" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_NEIGH_LIST_LTE_CELL_CID_title }" must="1"/>
		<div id="LTE_NEIGH_LIST_LTE_CELL_CID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_NEIGH_LIST_LTE_CELL_CID_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_NEIGH_LIST_LTE_CELL_EARFCN_name">${LTE_NEIGH_LIST_LTE_CELL_EARFCN_name }</label>
		<input id="LTE_NEIGH_LIST_LTE_CELL_EARFCN_name" name="LTE_NEIGH_LIST_LTE_CELL_EARFCN" type="text" min_value="0" max_value="65535" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_NEIGH_LIST_LTE_CELL_EARFCN_title }" must="1"/>
		<div id="LTE_NEIGH_LIST_LTE_CELL_EARFCN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_NEIGH_LIST_LTE_CELL_EARFCN_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_NEIGH_LIST_LTE_CELL_PHY_CELLID_name">${LTE_NEIGH_LIST_LTE_CELL_PHY_CELLID_name }</label>
		<input id="LTE_NEIGH_LIST_LTE_CELL_PHY_CELLID_name" name="LTE_NEIGH_LIST_LTE_CELL_PHY_CELLID" type="text" min_value="0" max_value="503" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_NEIGH_LIST_LTE_CELL_PHY_CELLID_title }" must="1"/>
		<div id="LTE_NEIGH_LIST_LTE_CELL_PHY_CELLID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_NEIGH_LIST_LTE_CELL_PHY_CELLID_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_X_BAICELLS_NEIGH_LIST_LTE_CELL_TAC_name">${LTE_X_BAICELLS_NEIGH_LIST_LTE_CELL_TAC_name }</label>
		<input id="LTE_X_BAICELLS_NEIGH_LIST_LTE_CELL_TAC_name" name="LTE_X_BAICELLS_NEIGH_LIST_LTE_CELL_TAC" type="text" min_value="0" max_value="65535" 
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_X_BAICELLS_NEIGH_LIST_LTE_CELL_TAC_title }" must="1"/>
		<div id="LTE_X_BAICELLS_NEIGH_LIST_LTE_CELL_TAC_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_X_BAICELLS_NEIGH_LIST_LTE_CELL_TAC_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li> 
		<label for="LTE_NEIGH_LIST_LTE_CELL_QOFFSET_name">${LTE_NEIGH_LIST_LTE_CELL_QOFFSET_name }</label>
		<select id="LTE_NEIGH_LIST_LTE_CELL_QOFFSET_name" name="LTE_NEIGH_LIST_LTE_CELL_QOFFSET" class="border border-box" onblur="validateRequired(event);createMML();">
			<option value=""></option>
			<option value="-24">-24</option>
			<option value="-22">-22</option>
			<option value="-20">-20</option>
			<option value="-18">-18</option>
			<option value="-16">-16</option>
			<option value="-14">-14</option>
			<option value="-12">-12</option>
			<option value="-10">-10</option>
			<option value="-8">-8</option>
			<option value="-6">-6</option>
			<option value="-5">-5</option>
			<option value="-4">-4</option>
			<option value="-3">-3</option>
			<option value="-2">-2</option>
			<option value="-1">-1</option>
			<option value="0">0</option>
			<option value="1">1</option>
			<option value="2">2</option>
			<option value="3">3</option>
			<option value="4">4</option>
			<option value="5">5</option>
			<option value="6">6</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
		</select>
		<div id="LTE_NEIGH_LIST_LTE_CELL_QOFFSET_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			<%=rb.getString("BiTian") %>
		</div>
	</li>
	<li> 
		<label for="LTE_NEIGH_LIST_LTE_CELL_CIO_name">${LTE_NEIGH_LIST_LTE_CELL_CIO_name }</label>
		<select id="LTE_NEIGH_LIST_LTE_CELL_CIO_name" name="LTE_NEIGH_LIST_LTE_CELL_CIO" class="border border-box" onblur="validateRequired(event);createMML();">
			<option value=""></option>
			<option value="-24">-24</option>
			<option value="-22">-22</option>
			<option value="-20">-20</option>
			<option value="-18">-18</option>
			<option value="-16">-16</option>
			<option value="-14">-14</option>
			<option value="-12">-12</option>
			<option value="-10">-10</option>
			<option value="-8">-8</option>
			<option value="-6">-6</option>
			<option value="-5">-5</option>
			<option value="-4">-4</option>
			<option value="-3">-3</option>
			<option value="-2">-2</option>
			<option value="-1">-1</option>
			<option value="0">0</option>
			<option value="1">1</option>
			<option value="2">2</option>
			<option value="3">3</option>
			<option value="4">4</option>
			<option value="5">5</option>
			<option value="6">6</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
		</select>
		<div id="LTE_NEIGH_LIST_LTE_CELL_CIO_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			<%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_NEIGH_LIST_LTE_CELL_ENABLE_name">${LTE_NEIGH_LIST_LTE_CELL_ENABLE_name }</label>
		<select id="LTE_NEIGH_LIST_LTE_CELL_ENABLE_name" name="LTE_NEIGH_LIST_LTE_CELL_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_NEIGH_LIST_LTE_CELL_ENB_TYPE_name">${LTE_NEIGH_LIST_LTE_CELL_ENB_TYPE_name}</label>
		<select id="LTE_NEIGH_LIST_LTE_CELL_ENB_TYPE_name" name="LTE_NEIGH_LIST_LTE_CELL_ENB_TYPE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">Home</option>
			<option value="0">Macro</option>
		</select>
	</li>
</ul>
