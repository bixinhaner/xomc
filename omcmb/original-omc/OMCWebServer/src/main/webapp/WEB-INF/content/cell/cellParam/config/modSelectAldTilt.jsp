<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
    <li>
		<label for="ALD_GET_TILT_name">${ALD_GET_TILT_name}</label>
		<input id="ALD_GET_TILT_name" name="ALD_GET_TILT" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${ALD_GET_TILT_title}" 
			min_value="0" max_value="255" class="border border-box" must="1"/>
		<div id="ALD_GET_TILT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${ALD_GET_TILT_title}<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
</ul>
