<%@ page language="java" contentType="text/html; charset=UTF-8" pageEncoding="UTF-8"%>

<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="MOD_ANTENNA_TILT1_name">B1(°)</label>
		<input id="MOD_ANTENNA_TILT1_name" title="Range: 0-12" class="border border-box" 
			min_value="0" max_value="12" js_regex="/^\d+$/" onblur="validateMaxAndMinVal(event);updateAntennalTilt();"/>
		<div id="MOD_ANTENNA_TILT1_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Range: 0-12
		</div>
	</li>
	<li>
		<label for="MOD_ANTENNA_TILT2_name">B2(°)</label>
		<input id="MOD_ANTENNA_TILT2_name" title="Range: 0-12" class="border border-box" 
			min_value="0" max_value="12" js_regex="/^\d+$/" onblur="validateMaxAndMinVal(event);updateAntennalTilt();"/>
		<div id="MOD_ANTENNA_TILT2_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Range: 0-12
		</div>
	</li>
	<li>
		<label for="MOD_ANTENNA_TILT3_name">B3(°)</label>
		<input id="MOD_ANTENNA_TILT3_name" title="Range: 0-12" class="border border-box" 
			min_value="0" max_value="12" js_regex="/^\d+$/" onblur="validateMaxAndMinVal(event);updateAntennalTilt();"/>
		<div id="MOD_ANTENNA_TILT3_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Range: 0-12
		</div>
	</li>
	<li>
		<label for="MOD_ANTENNA_TILT4_name">B4(°)</label>
		<input id="MOD_ANTENNA_TILT4_name" title="Range: 0-12" class="border border-box" 
			min_value="0" max_value="12" js_regex="/^\d+$/" onblur="validateMaxAndMinVal(event);updateAntennalTilt();"/>
		<div id="MOD_ANTENNA_TILT4_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Range: 0-12
		</div>
	</li>
	<li>
		<label for="MOD_ANTENNA_TILT5_name">B5(°)</label>
		<input id="MOD_ANTENNA_TILT5_name" title="Range: 0-12" class="border border-box" 
			min_value="0" max_value="12" js_regex="/^\d+$/" onblur="validateMaxAndMinVal(event);updateAntennalTilt();"/>
		<div id="MOD_ANTENNA_TILT5_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Range: 0-12
		</div>
	</li>
	<li>
		<label for="MOD_ANTENNA_TILT6_name">B6(°)</label>
		<input id="MOD_ANTENNA_TILT6_name" title="Range: 0-12" class="border border-box" 
			min_value="0" max_value="12" js_regex="/^\d+$/" onblur="validateMaxAndMinVal(event);updateAntennalTilt();"/>
		<div id="MOD_ANTENNA_TILT6_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Range: 0-12
		</div>
	</li>
	<li>
		<input id="MOD_ANTENNA_TILT_name" type="hidden" name="MOD_ANTENNA_TILT"/>
	</li>
</ul>

<script>
	function updateAntennalTilt() {
		var b1 = $('#MOD_ANTENNA_TILT1_name').val(),
			b2 = $('#MOD_ANTENNA_TILT2_name').val(),
			b3 = $('#MOD_ANTENNA_TILT3_name').val(),
			b4 = $('#MOD_ANTENNA_TILT4_name').val(),
			b5 = $('#MOD_ANTENNA_TILT5_name').val(),
			b6 = $('#MOD_ANTENNA_TILT6_name').val(),
			value = b1 + ',' + b2 + ',' + b3 + ',' + b4 + ',' + b5 + ',' + b6;
		
		$('#MOD_ANTENNA_TILT_name').val(value);
		createMML();
	}
</script>