<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
    <li>
		<label for="ALD_SET_TILT_DevNo_name">${ALD_SET_TILT_DevNo_name}</label>
		<input id="ALD_SET_TILT_DevNo_name" name="ALD_SET_TILT_DevNo" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${ALD_SET_TILT_DevNo_title}" 
			min_value="0" max_value="255" class="border border-box" must="1"/>
		<div id="ALD_SET_TILT_DevNo_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${ALD_SET_TILT_DevNo_title}<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="ALD_SET_TILT_Tilt10_name">${ALD_SET_TILT_Tilt10_name }</label>
		<input id="ALD_SET_TILT_Tilt10_name" name="ALD_SET_TILT_Tilt10" title="${ALD_SET_TILT_Tilt10_title }" class="border border-box" 
			onblur="validateSpecialChar(event);createMML();"
			js_regex="/^((-)?\d+(\.\d)?)$/"  must="1" />
		<div id="ALD_SET_TILT_Tilt10_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${ALD_SET_TILT_Tilt10_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
</ul>
