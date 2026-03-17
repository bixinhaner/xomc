<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="RU_RSSI_ENABLE_name">${RU_RSSI_ENABLE_name }</label>
		<select id="RU_RSSI_ENABLE_name" name="RU_RSSI_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<%-- <li>
		<label for="RU_RSSI0_name">${RU_RSSI0_name }</label>
		<input id="RU_RSSI0_name" name="RU_RSSI0" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${RU_RSSI0_title }" 
			min_value="-65535" max_value="65535" class="border border-box"/>
		<div id="RU_RSSI0_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RU_RSSI0_title }
		</div>
	</li>
	<li>
		<label for="RU_RSSI1_name">${RU_RSSI1_name }</label>
		<input id="RU_RSSI1_name" name="RU_RSSI1" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${RU_RSSI1_title }" 
			min_value="-65535" max_value="65535"  class="border border-box"/>
		<div id="RU_RSSI1_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RU_RSSI1_title }
		</div>
	</li>
	<li>
		<label for="RU_TX0_POWER_name">${RU_TX0_POWER_name }</label>
		<input id="RU_TX0_POWER_name" name="RU_TX0_POWER" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${RU_TX0_POWER_title }" 
			min_value="-65535" max_value="65535"class="border border-box"/>
		<div id="RU_TX0_POWER_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RU_TX0_POWER_title }
		</div>
	</li>
	<li>
		<label for="RU_TX1_POWER_name">${RU_TX1_POWER_name }</label>
		<input id="RU_TX1_POWER_name" name="RU_TX1_POWER" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${RU_TX1_POWER_title }" 
			class="border border-box"/>
		<div id="RU_TX1_POWER_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RU_TX1_POWER_title }
		</div>
	</li>
	<li>
		<label for="RU_TX0_VSWR_name">${RU_TX0_VSWR_name }</label>
		<input id="RU_TX0_VSWR_name" name="RU_TX0_VSWR" type="text" onblur="validateMaxAndMinLength(event);createMML();" title="${RU_TX0_VSWR_title }" 
			min_value="-65535" max_value="65535" class="border border-box"/>
		<div id="RU_TX0_VSWR_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RU_TX0_VSWR_title }
		</div>
	</li>
	<li>
		<label for="RU_TX1_VSWR_name">${RU_TX1_VSWR_name }</label>
		<input id="RU_TX1_VSWR_name" name="RU_TX1_VSWR" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${RU_TX1_VSWR_title }" 
			min_value="-65535" max_value="65535" class="border border-box"/>
		<div id="RU_TX1_VSWR_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RU_TX1_VSWR_title }
		</div>
	</li> --%>
	
</ul>
