<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="/common/alltaglibs.jsp" %>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_ANR_REPORT_CFG_VAL_name">${LTE_ANR_REPORT_CFG_VAL_name}</label>
		<select id="LTE_ANR_REPORT_CFG_VAL_name" name="LTE_ANR_REPORT_CFG_VAL" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">No ANR</option>
			<option value="1">EVENT A3</option>
			<option value="2">EVENT A5</option>
			<option value="3">STRONG CELL</option>
			<option value="4">EVENT A5 INTER</option>
		</select>
	</li>
	<li>
		<label for="LTE_REPORT_AMOUNT_name">${LTE_REPORT_AMOUNT_name}</label>
		<select id="LTE_REPORT_AMOUNT_name" name="LTE_REPORT_AMOUNT" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="r1">r1</option>
			<option value="r2">r2</option>
			<option value="r4">r4</option>
			<option value="r8">r8</option>
			<option value="r16">r16</option>
			<option value="r32">r32</option>
			<option value="r64">r64</option>
			<option value="infinity">infinity</option>
		</select>
	</li>
	
</ul>