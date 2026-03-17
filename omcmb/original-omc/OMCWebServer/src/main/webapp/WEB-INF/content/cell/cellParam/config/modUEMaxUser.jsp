<%@ page language="java" contentType="text/html; charset=UTF-8" pageEncoding="UTF-8"%>

<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_MAX_UE_SERVED_name">${LTE_MAX_UE_SERVED_name }</label>
		<input id="LTE_MAX_UE_SERVED_name" name="LTE_MAX_UE_SERVED" type="text" min_value="0" max_value="512"
			onblur="validateMaxAndMinVal(event);createMML();" class="border border-box" title="${LTE_MAX_UE_SERVED_title }"/>
		<div id="LTE_MAX_UE_SERVED_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_MAX_UE_SERVED_title }
		</div>
	</li>
	<li>
		<label for="LTE_MAX_UE_ACTIVE_name">${LTE_MAX_UE_ACTIVE_name }</label>
		<input id="LTE_MAX_UE_ACTIVE_name" name="LTE_MAX_UE_ACTIVE" type="text" title="${LTE_MAX_UE_ACTIVE_title }"  min_value="0" max_value="512"
			class="border border-box" onblur="validateMaxAndMinLength(event);validateUEActive(event);createMML();"/>
		<div id="LTE_MAX_UE_ACTIVE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_MAX_UE_ACTIVE_title }
		</div>
	</li>
</ul>
<script>
	function validateUEActive(e) {
		var ele = $(e["target"]),
			sval = LTE_MAX_UE_SERVED_name.value,
			aval = LTE_MAX_UE_ACTIVE_name.value;

		if(aval - sval > 0 || aval - 512 > 0) {
			$("#LTE_MAX_UE_ACTIVE_name_err").addClass('redColor');
			ele.addClass("err_border");
		}else{
			$("#LTE_MAX_UE_ACTIVE_name_err").removeClass('redColor');
			ele.removeClass("err_border");
		}
	}
</script>