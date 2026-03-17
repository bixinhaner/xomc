<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
    
<style>
	.ck-cls {
		display: inline-block;
		margin-right: 10px;
		min-width: 70px;
	}
	.ck-cls input {
		width: 20px;
	}
</style>

<ul id="paramNodesUl" class="paramNodesUl">
	<li style="display: flex; align-items: baseline;height:auto;">
		<label for="STK_MD_ALL_CK" style="margin-bottom: 5px;">${LTE_SLOG_MASK_name }: </label>
		<input id="STK_MD_ALL_CK" type="checkbox" class="ignore" style="width: 20px;"/> All
		<div id="fpalog_ck_list" style="display: flex; flex-wrap: wrap; margin-left: 10px;">
			<span class="ck-cls"> <input id="STK_MD_APP" type="checkbox" class="ignore" onblur="createMML();" /> APP </span>
			<span class="ck-cls"> <input id="STK_MD_EPR" type="checkbox" class="ignore" onblur="createMML();" /> EPR </span>
			<span class="ck-cls"> <input id="STK_MD_MAC" type="checkbox" class="ignore" onblur="createMML();" /> MAC </span>
			<span class="ck-cls"> <input id="STK_MD_YS"  type="checkbox" class="ignore" onblur="createMML();" /> YS </span>
			<span class="ck-cls"> <input id="STK_MD_RAM" type="checkbox" class="ignore" onblur="createMML();" /> RAM </span>
			<span class="ck-cls"> <input id="STK_MD_LGW" type="checkbox" class="ignore" onblur="createMML();" /> LGW </span>
			<span class="ck-cls"> <input id="STK_MD_UE"  type="checkbox" class="ignore" onblur="createMML();" /> UE </span>
		</div>
	</li>
	<li style="height: 0; overflow: hidden;">
		<input id="LTE_SLOG_MASK_STRING_name" name="LTE_SLOG_MASK" type="hidden"/>
	</li>
</ul>

<script>
	var maskString = [];
	
	createSlogStr();
	
	$('#fpalog_ck_list input').on('change',function(){
		var bools = Array.from($('#fpalog_ck_list input')).map(function(item){
			return item.checked;
		});
		
		STK_MD_ALL_CK.checked = !bools.includes(false);
		
		createSlogStr();
	});
	
	$('#STK_MD_ALL_CK').on('change',function(val){
		var checked = this.checked;
		
		$('#fpalog_ck_list input').each(function(idx, item) {
			item.checked = checked;
		});
		
		createSlogStr();
	});
	
	function createSlogStr() {
		maskString = [];
		
		$('#fpalog_ck_list input').each(function(idx, item) {
			var key = item.id,
				value = [true, 'true'].includes(item.checked)? 1:0;
			
			maskString.push(key + ':' + value);
		});
		
		LTE_SLOG_MASK_STRING_name.value = maskString.join(',');
		
		createMML();
	}
</script>