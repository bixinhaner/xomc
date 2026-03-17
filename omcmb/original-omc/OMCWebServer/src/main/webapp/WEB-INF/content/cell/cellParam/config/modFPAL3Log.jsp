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
		<label for="LTE_ENBLOG_MASK_STRING_name" style="margin-bottom: 5px;">${LTE_ENBLOG_MASK_name }: </label>
		<input id="STK_MD_L3_ALL_CK" type="checkbox" class="ignore" style="width: 60px;"/> All
		<div id="fpal3log_ck_list" style="display: flex; flex-wrap: wrap; margin-left: 10px;">
			<span class="ck-cls"> <input id="RRC" type="checkbox" class="ignore" onblur="createMML();" /> RRC </span>
			<span class="ck-cls"> <input id="RLC" type="checkbox" class="ignore" onblur="createMML();" /> RLC </span>
			<span class="ck-cls"> <input id="EMM+MSM"  type="checkbox" class="ignore" onblur="createMML();" /> EMM+MSM </span>
			<span class="ck-cls"> <input id="SMM" type="checkbox" class="ignore" onblur="createMML();" /> SMM </span>
			<span class="ck-cls"> <input id="IFM"  type="checkbox" class="ignore" onblur="createMML();" /> IFM </span>
			<span class="ck-cls"> <input id="EGTP+LGW"  type="checkbox" class="ignore" onblur="createMML();" /> EGTP+LGW </span>
			<span class="ck-cls"> <input id="DAM"  type="checkbox" class="ignore" onblur="createMML();" /> DAM </span>
			<span class="ck-cls"> <input id="UMM"  type="checkbox" class="ignore" onblur="createMML();" /> UMM </span>
			<span class="ck-cls"> <input id="ENBAPP"  type="checkbox" class="ignore" onblur="createMML();" /> ENBAPP </span>
			<span class="ck-cls"> <input id="RRM"  type="checkbox" class="ignore" onblur="createMML();" /> RRM </span>
			<span class="ck-cls"> <input id="SON"  type="checkbox" class="ignore" onblur="createMML();" /> SON </span>
			<span class="ck-cls"> <input id="PDCP"  type="checkbox" class="ignore" onblur="createMML();" /> PDCP </span>
			<span class="ck-cls"> <input id="MAC"  type="checkbox" class="ignore" onblur="createMML();" /> MAC </span>
			<span class="ck-cls"> <input id="CL"  type="checkbox" class="ignore" onblur="createMML();" /> CL </span>
			<span class="ck-cls"> <input id="S1AP"  type="checkbox" class="ignore" onblur="createMML();" /> S1AP </span>
			<span class="ck-cls"> <input id="SCTP"  type="checkbox" class="ignore" onblur="createMML();" /> SCTP </span>
			<span class="ck-cls"> <input id="UDX"  type="checkbox" class="ignore" onblur="createMML();" /> UDX </span>
		</div>
	</li>
	<li style="height: 0; overflow: hidden;">
		<input id="LTE_ENBLOG_MASK_STRING_name" name="LTE_ENBLOG_MASK" type="hidden"/>
	</li>
</ul>

<script>
	var maskString = [];
	
	createSlogStr();
	
	$('#fpal3log_ck_list input').on('change',function(){
		var bools = Array.from($('#fpal3log_ck_list input')).map(function(item){
			return item.checked;
		});
		
		STK_MD_L3_ALL_CK.checked = !bools.includes(false);
		
		createSlogStr();
	});
	
	$('#STK_MD_L3_ALL_CK').on('change',function(val){
		var checked = this.checked;
		
		$('#fpal3log_ck_list input').each(function(idx, item) {
			item.checked = checked;
		});
		
		createSlogStr();
	});
	
	function createSlogStr() {
		maskString = [];
		
		$('#fpal3log_ck_list input').each(function(idx, item) {
			var key = item.id,
				value = [true, 'true'].includes(item.checked)? 1:0;
			
			maskString.push(key + ':' + value);
		});
		
		LTE_ENBLOG_MASK_STRING_name.value = maskString.join(',');
		
		createMML();
	}
</script>