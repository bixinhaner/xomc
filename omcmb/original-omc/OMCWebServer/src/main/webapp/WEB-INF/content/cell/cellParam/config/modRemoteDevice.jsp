<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>

<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="i_name">RU Index</label>
		<input id="i_name" name="i" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="Integer,range:1-32" 
			min_value="1" max_value="32" class="border border-box" must="1"/>
		<div id="i_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Integer,range:1-32,Required
		</div>
	</li>
	
	<li>
		<label for="CRAN_EU_RU_RFTxStatus_name">${CRAN_EU_RU_RFTxStatus_name }</label>
		<select id="CRAN_EU_RU_RFTxStatus_name" name="CRAN_EU_RU_RFTxStatus" class="border border-box" onblur="validateStatus();createMML();">
			<option value=""></option>
			<option value="true">true</option>
			<option value="false">false</option>
		</select>
		<div id="i_status_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Required
		</div>
	</li>
</ul>
<script>
	function validateStatus(){
		var val = $("#CRAN_EU_RU_RFTxStatus_name option:selected").val();
		if(val == ""){
			$("#i_status_err").show()
		}else{
			$("#i_status_err").hide()
		}
	}
</script>