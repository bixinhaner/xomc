<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="PM_UPLOAD_ENABLE_name">${PM_UPLOAD_ENABLE_name}</label>
		<select id="PM_UPLOAD_ENABLE_name" name="PM_UPLOAD_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
   <%--  <li>
        <label for="ENB_VENDOR_UNIT_FAMILY_TYPE_name">${ENB_VENDOR_UNIT_FAMILY_TYPE_name }</label>
        <input id="ENB_VENDOR_UNIT_FAMILY_TYPE_name" name="ENB_VENDOR_UNIT_FAMILY_TYPE" type="text" min_value="1" max_value="256" 
            onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${ENB_VENDOR_UNIT_FAMILY_TYPE_title}"/>
        <div id="ENB_VENDOR_UNIT_FAMILY_TYPE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${ENB_VENDOR_UNIT_FAMILY_TYPE_title}
        </div>
    </li>
    
     <li>
        <label for="LBT_STATUS_name">${LBT_STATUS_name}</label>
        <input id="LBT_STATUS_name" name="LBT_STATUS" type="text" min_value="0" max_value="2" 
            onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LBT_STATUS_title}"/>
        <div id="LBT_STATUS_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LBT_STATUS_title}
        </div>
    </li> --%>
     
</ul>
