<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_1588_SYNC_ENABLE_name">${LTE_1588_SYNC_ENABLE_name }</label>
		<select id="LTE_1588_SYNC_ENABLE_name" name="LTE_1588_SYNC_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li class="paramFor1588">
		<label for="LTE_1588_DOMAIN_NUM_name">${LTE_1588_DOMAIN_NUM_name }</label>
		<input id="LTE_1588_DOMAIN_NUM_name" name="LTE_1588_DOMAIN_NUM" title="${LTE_1588_DOMAIN_NUM_title }" class="border border-box" 
		 min_value="0" max_value="255" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_1588_DOMAIN_NUM_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_1588_DOMAIN_NUM_title }
		</div>
	</li>
	<li class="paramFor1588">
		<label for="LTE_1588_UNICAST_SWITCH_name">${LTE_1588_UNICAST_SWITCH_name }</label>
		<select id="LTE_1588_UNICAST_SWITCH_name" name="LTE_1588_UNICAST_SWITCH" class="border border-box" onblur="createMML();" onchange="unicastChange(this)">
			<option value=""></option>
			<option value="1">Unicast</option>
			<option value="0">Multicast</option>
		</select>
	</li>
	<li class="paramFor1588">
		<label for="LTE_1588_ASYMMETRY_VALUE_name">${LTE_1588_ASYMMETRY_VALUE_name }</label>
		<input id="LTE_1588_ASYMMETRY_VALUE_name" name="LTE_1588_ASYMMETRY_VALUE" title="${LTE_1588_ASYMMETRY_VALUE_title }" class="border border-box" 
		 min_value="-65535" max_value="65535" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_1588_ASYMMETRY_VALUE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_1588_ASYMMETRY_VALUE_title }
		</div>
	</li>
	<li class="paramFor1588">
		<label for="LTE_1588_INTERFACE_BINDING_name">${LTE_1588_INTERFACE_BINDING_name }</label>
		<select id="LTE_1588_INTERFACE_BINDING_name" name="LTE_1588_INTERFACE_BINDING" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="eth2">WAN</option>
		</select>
	</li>
	<li class="paramFor1588 enableForUnicast" style="display: none;">
		<label for="LTE_1588_UNICAST_IP_name">${LTE_1588_UNICAST_IP_name }</label>
		<input id="LTE_1588_UNICAST_IP_name" name="LTE_1588_UNICAST_IP" type="text" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" title="${LTE_1588_UNICAST_IP_title }" 
			js_regex="/^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-4]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/" min_length="0" max_length="256" class="border border-box"/>
		<div id="LTE_1588_UNICAST_IP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_1588_UNICAST_IP_title }
		</div>
	</li>
	<li class="paramFor1588 enableForUnicast" style="display: none;">
		<label for="LTE_1588_SYNC_MESSAGE_INTERVAL_name">${LTE_1588_SYNC_MESSAGE_INTERVAL_name }</label>
		<input id="LTE_1588_SYNC_MESSAGE_INTERVAL_name" name="LTE_1588_SYNC_MESSAGE_INTERVAL" title="${LTE_1588_SYNC_MESSAGE_INTERVAL_title }" class="border border-box" 
		 min_value="-7" max_value="0" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_1588_SYNC_MESSAGE_INTERVAL_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_1588_SYNC_MESSAGE_INTERVAL_title }
		</div>
	</li>
	<li class="paramFor1588 enableForUnicast" style="display: none;">
		<label for="LTE_1588_DELAY_REQUEST_MESSAGE_INTERVAL_name">${LTE_1588_DELAY_REQUEST_MESSAGE_INTERVAL_name }</label>
		<input id="LTE_1588_DELAY_REQUEST_MESSAGE_INTERVAL_name" name="LTE_1588_DELAY_REQUEST_MESSAGE_INTERVAL" title="${LTE_1588_DELAY_REQUEST_MESSAGE_INTERVAL_title }" class="border border-box" 
		 min_value="-7" max_value="0" onblur="validateMaxAndMinVal(event);createMML();"/>
		<div id="LTE_1588_DELAY_REQUEST_MESSAGE_INTERVAL_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_1588_DELAY_REQUEST_MESSAGE_INTERVAL_title }
		</div>
	</li>
</ul>

<script type ="text/javascript">
LTE_1588_UNICAST_SWITCH_name.value = '0';
LTE_1588_INTERFACE_BINDING_name.value = 'eth2';

function changeVisibilityOf1588Param(e) {
	<%-- 同步开关打开，则显示1588相关参数，反之，隐藏这些参数 --%>
	var lte_1588_sync_enable_value = $(e.target).val();
	if ("1" == lte_1588_sync_enable_value) {
		$(".paramFor1588").show();
	} else {
		$(".paramFor1588").hide();
	}
}

function unicastChange(dom) {
	var val = $(dom).val();
	
	if(val === '1') {
		$('.enableForUnicast').show();
	}else {
		$('.enableForUnicast').hide();
	}
}

<%-- 加载完成，触发1588同步开关的onchange事件  --%>
$("#LTE_1588_SYNC_ENABLE_name").change();
</script>