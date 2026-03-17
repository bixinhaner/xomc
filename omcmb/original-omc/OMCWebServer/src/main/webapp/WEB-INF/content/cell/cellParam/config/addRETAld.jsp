<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
    <li>
		<label for="ALD_ADD_DevNo_name">${ALD_ADD_DevNo_name}</label>
		<input id="ALD_ADD_DevNo_name" name="ALD_ADD_DevNo" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${ALD_ADD_DevNo_title}" 
			min_value="0" max_value="255" class="border border-box" must="1"/>
		<div id="ALD_ADD_DevNo_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${ALD_ADD_DevNo_title}<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>   
    <li>
        <label for="ALD_ADD_VendorCode_name">${ALD_ADD_VendorCode_name}</label>
        <input id="ALD_ADD_VendorCode_name" name="ALD_ADD_VendorCode" type="text" min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);createMML();" title="${ALD_ADD_VendorCode_title}"  class="border border-box" must="1"/>
        <div id="ALD_ADD_VendorCode_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${ALD_ADD_VendorCode_title}<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
        </div>
    </li>    
    <li>
        <label for="ALD_ADD_SerialNo_name">${ALD_ADD_SerialNo_name}</label>
        <input id="ALD_ADD_SerialNo_name" name="ALD_ADD_SerialNo" type="text" min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);createMML();" title="${ALD_ADD_SerialNo_title}"  class="border border-box" must="1"/>
        <div id="ALD_ADD_SerialNo_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${ALD_ADD_SerialNo_title}<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
        </div>
    </li>    
</ul>
