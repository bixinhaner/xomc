<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c" %>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="i_name">${i_name }</label>
		<input id="i_name" name="i" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${i_title }" 
			min_value="1" max_value="2" class="border border-box" must="1"/>
		<div id="i_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${i_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_TUNNELENABLE_name">${TUNNEL_CONFIG_TUNNELENABLE_name }</label>
		<select id="TUNNEL_CONFIG_TUNNELENABLE_name" name="TUNNEL_CONFIG_TUNNELENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_LEFTAUTH_name">${TUNNEL_CONFIG_LEFTAUTH_name }</label>
		<select id="TUNNEL_CONFIG_LEFTAUTH_name" name="TUNNEL_CONFIG_LEFTAUTH" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">psk</option>
			<option value="1">pubkey</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_RIGHTAUTH_name">${TUNNEL_CONFIG_RIGHTAUTH_name }</label>
		<select id="TUNNEL_CONFIG_RIGHTAUTH_name" name="TUNNEL_CONFIG_RIGHTAUTH" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">psk</option>
			<option value="1">pubkey</option>
		</select>
	</li>
	
	<li>
		<label for="TUNNEL_CONFIG_RIGHT_name">${TUNNEL_CONFIG_RIGHT_name }</label>
		<input id="TUNNEL_CONFIG_RIGHT_name" name="TUNNEL_CONFIG_RIGHT" title="${TUNNEL_CONFIG_RIGHT_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_CONFIG_RIGHT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_RIGHT_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_RIGHTSUBNET_name">${TUNNEL_CONFIG_RIGHTSUBNET_name }</label>
		<input id="TUNNEL_CONFIG_RIGHTSUBNET_name" name="TUNNEL_CONFIG_RIGHTSUBNET" title="${TUNNEL_CONFIG_RIGHTSUBNET_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_CONFIG_RIGHTSUBNET_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_RIGHTSUBNET_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_LEFTID_name">${TUNNEL_CONFIG_LEFTID_name }</label>
		<input id="TUNNEL_CONFIG_LEFTID_name" name="TUNNEL_CONFIG_LEFTID" title="${TUNNEL_CONFIG_LEFTID_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_CONFIG_LEFTID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_LEFTID_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_RIGHTID_name">${TUNNEL_CONFIG_RIGHTID_name }</label>
		<input id="TUNNEL_CONFIG_RIGHTID_name" name="TUNNEL_CONFIG_RIGHTID" title="${TUNNEL_CONFIG_RIGHTID_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_CONFIG_RIGHTID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_RIGHTID_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_LEFTCERT_name">${TUNNEL_CONFIG_LEFTCERT_name }</label>
		<input id="TUNNEL_CONFIG_LEFTCERT_name" name="TUNNEL_CONFIG_LEFTCERT" title="${TUNNEL_CONFIG_LEFTCERT_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_CONFIG_LEFTCERT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_LEFTCERT_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_SECRETKEY_name">${TUNNEL_CONFIG_SECRETKEY_name }</label>
		<input id="TUNNEL_CONFIG_SECRETKEY_name" name="TUNNEL_CONFIG_SECRETKEY" title="${TUNNEL_CONFIG_SECRETKEY_title }" class="border border-box" min_length="0" 
			max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_CONFIG_SECRETKEY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_SECRETKEY_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_LEFTSOURCEIP_name">${TUNNEL_CONFIG_LEFTSOURCEIP_name }</label>
		<input id="TUNNEL_CONFIG_LEFTSOURCEIP_name" name="TUNNEL_CONFIG_LEFTSOURCEIP" title="${TUNNEL_CONFIG_LEFTSOURCEIP_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_CONFIG_LEFTSOURCEIP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_LEFTSOURCEIP_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_LEFTSUBNET_name">${TUNNEL_CONFIG_LEFTSUBNET_name }</label>
		<input id="TUNNEL_CONFIG_LEFTSUBNET_name" name="TUNNEL_CONFIG_LEFTSUBNET" title="${TUNNEL_CONFIG_LEFTSUBNET_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_CONFIG_LEFTSUBNET_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_LEFTSUBNET_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_FRAGMENTATION_name">${TUNNEL_CONFIG_FRAGMENTATION_name }</label>
		<select id="TUNNEL_CONFIG_FRAGMENTATION_name" name="TUNNEL_CONFIG_FRAGMENTATION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">yes</option>
			<option value="1">accept</option>
			<option value="2">force</option>
			<option value="3">no</option>
		</select>
	</li>

	<li>
		<label for="TUNNEL_CONFIG_IKEENCRYPTION_name">${TUNNEL_CONFIG_IKEENCRYPTION_name }</label>
		<select id="TUNNEL_CONFIG_IKEENCRYPTION_name" name="TUNNEL_CONFIG_IKEENCRYPTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">aes128</option>
			<option value="1">aes256</option>
			<option value="2">3des</option>
			<option value="3">des</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_IKEDHGROUP_name">${TUNNEL_CONFIG_IKEDHGROUP_name }</label>
		<select id="TUNNEL_CONFIG_IKEDHGROUP_name" name="TUNNEL_CONFIG_IKEDHGROUP" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">modp768</option>
			<option value="1">modp1024</option>
			<option value="2">modp1536</option>
			<option value="3">modp2048</option>
			<option value="4">modp4096</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_IKEAUTHENTICATION_name">${TUNNEL_CONFIG_IKEAUTHENTICATION_name }</label>
		<select id="TUNNEL_CONFIG_IKEAUTHENTICATION_name" name="TUNNEL_CONFIG_IKEAUTHENTICATION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">sha1</option>
			<option value="1">sha1_160</option>
			<option value="2">sha256_96</option>
			<option value="3">sha256</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_ESPENCRYPTION_name">${TUNNEL_CONFIG_ESPENCRYPTION_name }</label>
		<select id="TUNNEL_CONFIG_ESPENCRYPTION_name" name="TUNNEL_CONFIG_ESPENCRYPTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">aes128</option>
			<option value="1">aes256</option>
			<option value="2">3des</option>
			<option value="3">des</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_ESPDHGROUP_name">${TUNNEL_CONFIG_ESPDHGROUP_name }</label>
		<select id="TUNNEL_CONFIG_ESPDHGROUP_name" name="TUNNEL_CONFIG_ESPDHGROUP" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">modp768</option>
			<option value="1">modp1024</option>
			<option value="2">modp1536</option>
			<option value="3">modp2048</option>
			<option value="4">modp4096</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_ESPAUTHENTICATION_name">${TUNNEL_CONFIG_ESPAUTHENTICATION_name }</label>
		<select id="TUNNEL_CONFIG_ESPAUTHENTICATION_name" name="TUNNEL_CONFIG_ESPAUTHENTICATION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">sha1</option>
			<option value="1">sha1_160</option>
			<option value="2">sha256_96</option>
			<option value="3">sha256</option>
		</select>
	</li>
	
	<li>
		<label for="TUNNEL_CONFIG_KEYLIFE_name">${TUNNEL_CONFIG_KEYLIFE_name }</label>
		<input id="TUNNEL_CONFIG_KEYLIFE_name" name="TUNNEL_CONFIG_KEYLIFE" title="${TUNNEL_CONFIG_KEYLIFE_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateKeylife(event);validateTimeRange(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
		<div id="TUNNEL_CONFIG_KEYLIFE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_KEYLIFE_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_IKELIFETIME_name">${TUNNEL_CONFIG_IKELIFETIME_name }</label>
		<input id="TUNNEL_CONFIG_IKELIFETIME_name" name="TUNNEL_CONFIG_IKELIFETIME" title="${TUNNEL_CONFIG_IKELIFETIME_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateIKElifeTime(event);validateTimeRange(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
		<div id="TUNNEL_CONFIG_IKELIFETIME_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_IKELIFETIME_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_REKEYMARGIN_name">${TUNNEL_CONFIG_REKEYMARGIN_name }</label>
		<input id="TUNNEL_CONFIG_REKEYMARGIN_name" name="TUNNEL_CONFIG_REKEYMARGIN" title="${TUNNEL_CONFIG_REKEYMARGIN_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);ValidateMinTime(event);validateTimeRange(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
		<div id="TUNNEL_CONFIG_REKEYMARGIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_REKEYMARGIN_title }
		</div>
	</li>
	
	<li>
		<label for="TUNNEL_CONFIG_DPDACTION_name">${TUNNEL_CONFIG_DPDACTION_name }</label>
		<select id="TUNNEL_CONFIG_DPDACTION_name" name="TUNNEL_CONFIG_DPDACTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">none</option>
			<option value="1">clear</option>
			<option value="2">hold</option>
			<option value="3">restart</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_CONFIG_DPDDELAY_name">${TUNNEL_CONFIG_DPDDELAY_name }</label>
		<input id="TUNNEL_CONFIG_DPDDELAY_name" name="TUNNEL_CONFIG_DPDDELAY" title="${TUNNEL_CONFIG_DPDDELAY_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d+[s|m|d]{1}$/"/>
		<div id="TUNNEL_CONFIG_DPDDELAY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_CONFIG_DPDDELAY_title }
		</div>
	</li>
</ul>
<script>
	function validateTimeRange(e) {
		var ele = $(e["target"]),
			units = {
				s: 31536000,
				m: 525600,
				h: 8760,
				d: 365
			},
			time = ele.val();
		
		if($('#'+ele.attr('id')+'_err').hasClass('redColor')) {
			return;
		}
		
		if(time) {
			var key = time.match(/[smhd]/)[0],
				value = time.match(/\d*/)[0],
				timeUnit = units[key];
			
			if(value - timeUnit <= 0) {
				$("#" + ele.attr("id") + "_err").removeClass('redColor');
				ele.removeClass("err_border");
			}else {
				$("#" + ele.attr("id") + "_err").addClass('redColor');
				ele.addClass("err_border");
			}
		}
	}

	function validateKeylife(e) {
		var ele = $(e["target"]);
		
		var mustVal = ele.attr("must") == undefined ? "" : ele.attr("must");
		
		if(mustVal != "1" && ele.val().trim() == ""){
			return ;
		}
		
		if(is3Times()){
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
			ele.removeClass("err_border");
		}else{
			$("#" + ele.attr("id") + "_err").addClass('redColor');
			ele.addClass("err_border");
		}

		try{
			$('#TUNNEL_CONFIG_IKELIFETIME_name').blur();
		}catch(e){}
	}

	function validateIKElifeTime(e) {
		var ele = $(e["target"]);
		
		var mustVal = ele.attr("must") == undefined ? "" : ele.attr("must");
		
		if(mustVal != "1" && ele.val().trim() == ""){
			return ;
		}
		
		if(isLargeThan()){
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
			ele.removeClass("err_border");
		}else{
			$("#" + ele.attr("id") + "_err").addClass('redColor');
			ele.addClass("err_border");
		}
	}

	function isLargeThan() {
		var value = $('#TUNNEL_CONFIG_IKELIFETIME_name').val(),
			tarVal = $('#TUNNEL_CONFIG_KEYLIFE_name').val(),
			reg = /^\d+[s|m|h|d]{1}$/,
			bool = false;

		if(reg.test(value)) {
			if(reg.test(tarVal)) {
				var ikelifeTime = getSecondsTime(value),
					keyTime = getSecondsTime(tarVal),
					distance = ikelifeTime - keyTime;

				if(distance >= 0) {
    				bool = true;
				}
			}
		}

		return bool;
	}

	function ValidateMinTime(e) {
		var ele = $(e["target"]),
			value = $('#TUNNEL_CONFIG_REKEYMARGIN_name').val(),
			reg = /^\d+[s|m|h|d]{1}$/,
			min = '5m';

		if(reg.test(value)) {
			var prevTime = getSecondsTime(value),
				minTime = getSecondsTime(min),
				distance = prevTime - minTime;

			if(distance >= 0) {
				$("#" + ele.attr("id") + "_err").removeClass('redColor');
				ele.removeClass("err_border");
			}else {
				$("#" + ele.attr("id") + "_err").addClass('redColor');
				ele.addClass("err_border");
			}
			
			try{
				$('#TUNNEL_CONFIG_KEYLIFE_name').blur();
			}catch(e){}
		}
	}

	function is3Times() {
		var value = $('#TUNNEL_CONFIG_KEYLIFE_name').val(),
			tarVal = $('#TUNNEL_CONFIG_REKEYMARGIN_name').val(),
			reg = /^\d+[s|m|h|d]{1}$/,
			bool = false;

		if(reg.test(value)) {
			if(reg.test(tarVal)) {
				var keylifeTime = getSecondsTime(value),
					rekeyTime = getSecondsTime(tarVal),
					distance = keylifeTime - rekeyTime*3;

				if(distance >= 0) {
    				bool = true;
				}
			}
		}

		return bool;
	}

	function getEnbType() {
		var type = '',
			hardVersion = hardware_version;

		if(['EA4.0','EA4.0DUAL'].includes(hardVersion)) {
			type = 'RTS&RTD';
		}

		return type;
	}

	function getSecondsTime(time) {
		var units = {
				s: 1,
				m: 60,
				h: 3600,
				d: 86400
			},
			key = time.match(/[smhd]/)[0],
			value = time.match(/\d*/)[0],
			timeUnit = units[key],
			sTime = value*timeUnit;

		return sTime;
	}
</script>