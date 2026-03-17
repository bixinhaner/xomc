<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_ENB_ID_RANGE_ENABLE_name">${LTE_ENB_ID_RANGE_ENABLE_name}</label>
		<select id="LTE_ENB_ID_RANGE_ENABLE_name" name="LTE_ENB_ID_RANGE_ENABLE" dynamic="0" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="true">true</option>
			<option value="false">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_ENB_ID_TYPE_name">${LTE_ENB_ID_TYPE_name }</label>
		<select id="LTE_ENB_ID_TYPE_name" name="LTE_ENB_ID_TYPE" dynamic="0" class="border border-box" 
			onblur="createMML();" onchange="lteEnbIdTypeChange(this)">
			<option value=""></option>
			<option value="segment">segment</option>
			<option value="normal">normal</option>
		</select>
	</li>
	<li>
		<label for="LTE_ENB_ID_RANGE_name">${LTE_ENB_ID_RANGE_name }</label>
		<input id="LTE_ENB_ID_RANGE_name" name="LTE_ENB_ID_RANGE" type="text" min_length="0" max_length="512" 
			onblur="validateMaxAndMinVal(event);validateEnbIdRange(event);createMML();" class="border border-box" title="${LTE_ENB_ID_RANGE_title }"/>
		<div id="LTE_ENB_ID_RANGE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_ENB_ID_RANGE_title }
		</div>
	</li>
</ul>

<script>
	function lteEnbIdTypeChange(el) {
		validateEnbIdRange();
	}
	
	function validateEnbIdRange(ev) {
		var reg = /^(,?\[\d+-\d+\]|,?\d+-\d+|,?\d+)*$/,
			type = $('#LTE_ENB_ID_TYPE_name').val(),
			error = $("#LTE_ENB_ID_RANGE_name_err"),
			ele = $('#LTE_ENB_ID_RANGE_name'),
			val = ele.val(),
			msgTip = '${LTE_ENB_ID_RANGE_title }';
		
		if(type == 'normal') {
			reg = /^(,?\d+-\d+|,?\d+)*$/;
			msgTip = '${LTE_ENB_ID_RANGE_title }';
		}
		
		if(val) {
			if(reg.test(val)) {
				error.removeClass('redColor').text(msgTip);
				ele.removeClass("err_border");
			}else {
				error.addClass('redColor').text(msgTip);
				ele.addClass("err_border");
			}
		}
	}
</script>