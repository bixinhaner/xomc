<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li id="usim_enable_li">
		<label for="80159">${USIM_ENABLE_name }</label>
		<select id="80159" name="USIM_ENABLE" class="border border-box" onblur="createMML();" onchange="usimEnableChange(this)">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="80160">${IMSI_name }</label>
		<input id="80160" name="IMSI" title="${IMSI_title }" class="border border-box" 
			min_length="0" max_length="21" onblur="validateMaxAndMinLength(event);createMML();"/>
		<div id="80160_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${IMSI_title }
		</div>
	</li>
	<li>
		<label for="80161">${KEY_name }</label>
		<input id="80161" name="KEY" title="${KEY_title }" class="border border-box" 
			min_length="0" max_length="32" onblur="validateMaxAndMinLength(event);createMML();"/>
		<div id="80161_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${KEY_title }
		</div>
	</li>
	<li>
		<label for="80162">${OPc_name }</label>
		<input id="80162" name="OPc" title="${OPc_title }" class="border border-box" 
			min_length="0" max_length="32" onblur="validateMaxAndMinLength(event);createMML();"/>
		<div id="80162_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${OPc_title }
		</div>
	</li>
	<li>
		<label for="80163">${R_value_name }</label>
		<input id="80163" name="R_value" title="${R_value_title }" class="border border-box" 
			min_length="0" max_length="32" onblur="validateMaxAndMinLength(event);createMML();"/>
		<div id="80163_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${R_value_title }
		</div>
	</li>
</ul>

<script>
	function usimEnableChange(dom) {
		var value = dom.value,
			siblings = $('#usim_enable_li').siblings();
		
		if(value == '1') {
			siblings.show();
		}else {
			siblings.hide();
			//siblings.val('');
		}
	}
</script>