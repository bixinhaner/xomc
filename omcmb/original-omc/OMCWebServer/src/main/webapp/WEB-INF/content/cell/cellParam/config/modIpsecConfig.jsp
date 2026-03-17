<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
    <li>
        <label for="LTE_IPSEC_ONBOOT_name">${LTE_IPSEC_ONBOOT_name }</label>
        <select id="LTE_IPSEC_ONBOOT_name" name="LTE_IPSEC_ONBOOT" class="border border-box" onblur="createMML();" value"false">
            <option value=""></option>
            <option value="true">Enable</option>
            <option value="false">Disable</option>
        </select>
    </li>
    
    <li class="hidden">
        <label for="LTE_SWAN_CONFIG_MTU_name">${LTE_SWAN_CONFIG_MTU_name }</label>
        <input id="LTE_SWAN_CONFIG_MTU_name" name="LTE_SWAN_CONFIG_MTU" title="${LTE_SWAN_CONFIG_MTU_title }" class="border border-box" 
            min_value="0" max_value="9194" onblur="validateMaxAndMinLength(event);validateByRegexAndRange(event);createMML();" vali-regex="/^(\d*)$/"/>
        <div id="LTE_SWAN_CONFIG_MTU_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_SWAN_CONFIG_MTU_title }
        </div>
    </li>
    
	<li class="hidden">
		<label for="LTE_IPSEC_SOFT_USIM_name">${LTE_IPSEC_SOFT_USIM_name }</label>
        <select id="LTE_IPSEC_SOFT_USIM_name" name="LTE_IPSEC_SOFT_USIM" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="true">Enable</option>
            <option value="false">Disable</option>
        </select>
	</li>

	<li class="hidden">
		<label for="LTE_USIM_IMSI_VALUE_name">${LTE_USIM_IMSI_VALUE_name }</label>
		<input id="LTE_USIM_IMSI_VALUE_name" name="LTE_USIM_IMSI_VALUE" title="${LTE_USIM_IMSI_VALUE_title }" class="border border-box" 
			min_length="1" max_length="1024" onblur="validateMaxAndMinLength(event);createMML();"/>
		<div id="LTE_USIM_IMSI_VALUE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_USIM_IMSI_VALUE_title }
		</div>
	</li>
	<li class="hidden">
		<label for="LTE_USIM_KEY_name">${LTE_USIM_KEY_name }</label>
		<input id="LTE_USIM_KEY_name" name=LTE_USIM_KEY title="${LTE_USIM_KEY_title }" class="border border-box" 
			min_length="1" max_length="1024" onblur="validateMaxAndMinLength(event);createMML();"/>
		<div id="LTE_USIM_KEY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_USIM_KEY_title }
		</div>
	</li>
	
	<li class="hidden">
		<label for="LTE_USIM_OPC_name">${LTE_USIM_OPC_name }</label>
		<input id="LTE_USIM_OPC_name" name="LTE_USIM_OPC" title="${LTE_USIM_OPC_title }" class="border border-box" 
			min_length="1" max_length="1024" onblur="validateMaxAndMinLength(event);createMML();"/>
		<div id="LTE_USIM_OPC_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_USIM_OPC_title }
		</div>
	</li>
	
</ul>
<script type="text/javascript"> 
$(function(){
	// Ipsec enable 联动
	$('#LTE_IPSEC_ONBOOT_name').on('change',function(){
		var siblings = $(this).parent().siblings();

		if (this.value == "true") {
			var three = $('#LTE_USIM_IMSI_VALUE_name, #LTE_USIM_KEY_name, #LTE_USIM_OPC_name'),
				mtu = $('#LTE_SWAN_CONFIG_MTU_name').parent(),
				soft = $('#LTE_IPSEC_SOFT_USIM_name');
			if(soft.val()=='true') {
				siblings.removeClass('hidden');
			}else {
				mtu.removeClass('hidden');
				soft.parent().removeClass('hidden');
			}
		} else {
			siblings.addClass('hidden');
		}
	});
	// softUsim 联动
	$('#LTE_IPSEC_SOFT_USIM_name').on('change',function(){
		var siblings = $('#LTE_USIM_IMSI_VALUE_name, #LTE_USIM_KEY_name, #LTE_USIM_OPC_name');

		if (this.value == "true") {
			Array.from(siblings).map(function(item){
				$(item).parent().removeClass('hidden')
			})
		} else {
			Array.from(siblings).map(function(item){
				$(item).parent().addClass('hidden')
			})
		}
	});
});
</script>