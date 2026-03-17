<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c" %>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_TUNNEL2_ENABLE_name">${LTE_TUNNEL2_ENABLE_name }</label>
		<select id="LTE_TUNNEL2_ENABLE_name" name="LTE_TUNNEL2_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="true">Enable</option>
			<option value="false">Disable</option>
		</select>
	</li>
	<li>
		<label for="TEL_TUNNEL2_LEFT_AUTH_name">${TEL_TUNNEL2_LEFT_AUTH_name }</label>
		<select id="TEL_TUNNEL2_LEFT_AUTH_name" name="TEL_TUNNEL2_LEFT_AUTH" class="border border-box" onchange="validLeftAuth(this)" onblur="createMML();">
			<option value=""></option>
			<option value="psk">psk</option>
			<option value="pubkey">pubkey</option>
			<option value="eap-aka">eap-aka</option>
		</select>
	</li>
	<li>
		<label for="TEL_TUNNEL2_RIGHT_AUTH_name">${TEL_TUNNEL2_RIGHT_AUTH_name }</label>
		<select id="TEL_TUNNEL2_RIGHT_AUTH_name" name="TEL_TUNNEL2_RIGHT_AUTH" class="border border-box" onchange="validRightAuth(this)" onblur="createMML();">
			<option value=""></option>
			<option value="psk">psk</option>
			<option value="pubkey">pubkey</option>
			<option value="eap-aka">eap-aka</option>
		</select>
	</li>
	
	<li>
		<label for="TEL_TUNNEL2_RIGHT_name">${TEL_TUNNEL2_RIGHT_name }</label>
		<input id="TEL_TUNNEL2_RIGHT_name" name="TEL_TUNNEL2_RIGHT" title="${TEL_TUNNEL2_RIGHT_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_RIGHT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_RIGHT_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_RIGHT_SUBNET_name">${TEL_TUNNEL2_RIGHT_SUBNET_name }</label>
		<input id="TEL_TUNNEL2_RIGHT_SUBNET_name" name="TEL_TUNNEL2_RIGHT_SUBNET" title="${TEL_TUNNEL2_RIGHT_SUBNET_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_RIGHT_SUBNET_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_RIGHT_SUBNET_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_LEFT_ID_name">${TEL_TUNNEL2_LEFT_ID_name }</label>
		<input id="TEL_TUNNEL2_LEFT_ID_name" name="TEL_TUNNEL2_LEFT_ID" title="${TEL_TUNNEL2_LEFT_ID_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_LEFT_ID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_LEFT_ID_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_RIGHT_ID_name">${TEL_TUNNEL2_RIGHT_ID_name }</label>
		<input id="TEL_TUNNEL2_RIGHT_ID_name" name="TEL_TUNNEL2_RIGHT_ID" title="${TEL_TUNNEL2_RIGHT_ID_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_RIGHT_ID_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_RIGHT_ID_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_LEFT_CERT_name">${TEL_TUNNEL2_LEFT_CERT_name }</label>
		<input id="TEL_TUNNEL2_LEFT_CERT_name" name="TEL_TUNNEL2_LEFT_CERT" title="${TEL_TUNNEL2_LEFT_CERT_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_LEFT_CERT_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_LEFT_CERT_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_SECRET_KEY_name">${TEL_TUNNEL2_SECRET_KEY_name }</label>
		<input id="TEL_TUNNEL2_SECRET_KEY_name" name="TEL_TUNNEL2_SECRET_KEY" title="${TEL_TUNNEL2_SECRET_KEY_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_SECRET_KEY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_SECRET_KEY_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_RIGHT_SECRET_KEY_name">${TEL_TUNNEL2_RIGHT_SECRET_KEY_name }</label>
		<input id="TEL_TUNNEL2_RIGHT_SECRET_KEY_name" name="TEL_TUNNEL2_RIGHT_SECRET_KEY" title="${TEL_TUNNEL2_RIGHT_SECRET_KEY_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_RIGHT_SECRET_KEY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_RIGHT_SECRET_KEY_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_LEFT_SOURCEIP_name">${TEL_TUNNEL2_LEFT_SOURCEIP_name }</label>
		<input id="TEL_TUNNEL2_LEFT_SOURCEIP_name" name="TEL_TUNNEL2_LEFT_SOURCEIP" title="${TEL_TUNNEL2_LEFT_SOURCEIP_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_LEFT_SOURCEIP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_LEFT_SOURCEIP_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_LEFT_SUBNET_name">${TEL_TUNNEL2_LEFT_SUBNET_name }</label>
		<input id="TEL_TUNNEL2_LEFT_SUBNET_name" name="TEL_TUNNEL2_LEFT_SUBNET" title="${TEL_TUNNEL2_LEFT_SUBNET_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TEL_TUNNEL2_LEFT_SUBNET_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_LEFT_SUBNET_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_FRAGMENTATION_name">${TEL_TUNNEL2_FRAGMENTATION_name }</label>
		<select id="TEL_TUNNEL2_FRAGMENTATION_name" name="TEL_TUNNEL2_FRAGMENTATION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="yes">yes</option>
			<option value="accept">accept</option>
			<option value="force">force</option>
			<option value="no">no</option>
		</select>
	</li>

	<li>
		<label for="TEL_TUNNEL2_IKE_ENCRYPTION_name">${TEL_TUNNEL2_IKE_ENCRYPTION_name }</label>
		<select id="TEL_TUNNEL2_IKE_ENCRYPTION_name" name="TEL_TUNNEL2_IKE_ENCRYPTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="aes128">aes128</option>
			<option value="aes256">aes256</option>
			<option value="3des">3des</option>
		</select>
	</li>
	<li>
		<label for="TEL_TUNNEL2_IKE_DH_GROUP_name">${TEL_TUNNEL2_IKE_DH_GROUP_name }</label>
		<select id="TEL_TUNNEL2_IKE_DH_GROUP_name" name="TEL_TUNNEL2_IKE_DH_GROUP" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="modp768">modp768</option>
			<option value="modp1024">modp1024</option>
			<option value="modp1536">modp1536</option>
			<option value="modp2048">modp2048</option>
			<option value="modp3072">modp3072</option>
			<option value="modp4096">modp4096</option>
		</select>
	</li>
	<li>
		<label for="TEL_TUNNEL2_IKE_AUTHENTICATION_name">${TEL_TUNNEL2_IKE_AUTHENTICATION_name }</label>
		<select id="TEL_TUNNEL2_IKE_AUTHENTICATION_name" name="TEL_TUNNEL2_IKE_AUTHENTICATION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="sha1">sha1</option>
			<option value="sha1_160">sha1_160</option>
			<option value="sha256_96">sha256_96</option>
			<option value="sha256">sha256</option>
		</select>
	</li>
	<li>
		<label for="TEL_TUNNEL2_ESP_ENCRYPTION_name">${TEL_TUNNEL2_ESP_ENCRYPTION_name }</label>
		<select id="TEL_TUNNEL2_ESP_ENCRYPTION_name" name="TEL_TUNNEL2_ESP_ENCRYPTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="aes128">aes128</option>
			<option value="aes256">aes256</option>
			<option value="3des">3des</option>
		</select>
	</li>
	<li>
		<label for="TEL_TUNNEL2_ESP_DH_GROUP_name">${TEL_TUNNEL2_ESP_DH_GROUP_name }</label>
		<select id="TEL_TUNNEL2_ESP_DH_GROUP_name" name="TEL_TUNNEL2_ESP_DH_GROUP" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="null">null</option>
			<option value="modp768">modp768</option>
			<option value="modp1024">modp1024</option>
			<option value="modp1536">modp1536</option>
			<option value="modp2048">modp2048</option>
			<option value="modp4096">modp4096</option>
		</select>
	</li>
	<li>
		<label for="TEL_TUNNEL2_ESP_AUTHENTICATION_name">${TEL_TUNNEL2_ESP_AUTHENTICATION_name }</label>
		<select id="TEL_TUNNEL2_ESP_AUTHENTICATION_name" name="TEL_TUNNEL2_ESP_AUTHENTICATION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="sha1">sha1</option>
		</select>
	</li>
	
	<li>
		<label for="TEL_TUNNEL2_KEY_LIFE_name">${TEL_TUNNEL2_KEY_LIFE_name }</label>
		<input id="TEL_TUNNEL2_KEY_LIFE_name" name="TEL_TUNNEL2_KEY_LIFE" title="${TEL_TUNNEL2_KEY_LIFE_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateKeylife(event);validateTimeRange(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
		<div id="TEL_TUNNEL2_KEY_LIFE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_KEY_LIFE_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_IKE_LIFE_TIME_name">${TEL_TUNNEL2_IKE_LIFE_TIME_name }</label>
		<input id="TEL_TUNNEL2_IKE_LIFE_TIME_name" name="TEL_TUNNEL2_IKE_LIFE_TIME" title="${TEL_TUNNEL2_IKE_LIFE_TIME_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateIKElifeTime(event);validateTimeRange(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
		<div id="TEL_TUNNEL2_IKE_LIFE_TIME_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_IKE_LIFE_TIME_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_REKEY_MARGIN_name">${TEL_TUNNEL2_REKEY_MARGIN_name }</label>
		<input id="TEL_TUNNEL2_REKEY_MARGIN_name" name="TEL_TUNNEL2_REKEY_MARGIN" title="${TEL_TUNNEL2_REKEY_MARGIN_title }" class="border border-box" 
			min_length="0" max_length="64" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);ValidateMinTime(event);validateTimeRange(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
		<div id="TEL_TUNNEL2_REKEY_MARGIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_REKEY_MARGIN_title }
		</div>
	</li>
	
	<li>
		<label for="TEL_TUNNEL2_DPDACTION_name">${TEL_TUNNEL2_DPDACTION_name }</label>
		<select id="TEL_TUNNEL2_DPDACTION_name" name="TEL_TUNNEL2_DPDACTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="none">none</option>
			<option value="clear">clear</option>
			<option value="hold">hold</option>
			<option value="restart">restart</option>
		</select>
	</li>
	<li>
		<label for="TEL_TUNNEL2_DPDDELAY_name">${TEL_TUNNEL2_DPDDELAY_name }</label>
		<input id="TEL_TUNNEL2_DPDDELAY_name" name="TEL_TUNNEL2_DPDDELAY" title="${TEL_TUNNEL2_DPDDELAY_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d+[s|m|d]{1}$/"/>
		<div id="TUNNEL_CONFIG_DPDDELAY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TEL_TUNNEL2_DPDDELAY_title }
		</div>
	</li>
	<li>
		<label for="TEL_TUNNEL2_LEFT_INTERFACE_name">${TEL_TUNNEL2_LEFT_INTERFACE_name }</label>
		<select id="TEL_TUNNEL2_LEFT_INTERFACE_name" name="TEL_TUNNEL1_LEFT_INTERFACE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="none">none</option>
			<option value="wanConfig1">wanConfig1</option>
			<option value="wanConfig2">wanConfig2</option>
			<option value="wanConfig3">wanConfig3</option>
			<option value="wanConfig4">wanConfig4</option>
			<option value="wanConfig5">wanConfig5</option>
			<option value="wanConfig6">wanConfig6</option>
			<option value="wanConfig7">wanConfig7</option>
			<option value="wanConfig8">wanConfig8</option>
			<option value="wanConfig9">wanConfig9</option>
			<option value="wanConfig10">wanConfig10</option>
			<option value="wanConfig11">wanConfig11</option>
			<option value="wanConfig12">wanConfig12</option>
		</select>
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
			$('#TEL_TUNNEL2_IKE_LIFE_TIME_name').blur();
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
		var value = $('#TEL_TUNNEL2_IKE_LIFE_TIME_name').val(),
			tarVal = $('#TEL_TUNNEL2_KEY_LIFE_name').val(),
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
			value = $('#TEL_TUNNEL2_REKEY_MARGIN_name').val(),
			reg = /^\d+[s|m|h|d]{1}$/,
			minTypes = {
				'RTS&RTD': '5m',
				'QAFB': '3m'
			},
			type = getEnbType();

		if(minTypes[type] && reg.test(value)) {
			var prevTime = getSecondsTime(value),
				minTime = getSecondsTime(minTypes[type]),
				distance = prevTime - minTime;

			if(distance >= 0) {
				$("#" + ele.attr("id") + "_err").removeClass('redColor');
				ele.removeClass("err_border");
			}else {
				$("#" + ele.attr("id") + "_err").addClass('redColor');
				ele.addClass("err_border");
			}
			
			try{
				$('#TEL_TUNNEL2_KEY_LIFE_name').blur();
			}catch(e){}
		}
	}

	function is3Times() {
		var value = $('#TEL_TUNNEL2_KEY_LIFE_name').val(),
			tarVal = $('#TEL_TUNNEL2_REKEY_MARGIN_name').val(),
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
		
		if(['QC3.1'].includes(hardVersion)) {
			type = 'QAFB'
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

	
	function validLeftAuth(el) {
		var value = $(el).val(),
			tarDom = $('#TEL_TUNNEL2_RIGHT_AUTH_name'),
			cascade = {
				'psk-psk':         { show: ['TEL_TUNNEL2_SECRET_KEY_name'],  hide: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'psk-pubkey':      { show: ['TEL_TUNNEL2_SECRET_KEY_name'],  hide: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'pubkey-psk':      { show: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_SECRET_KEY_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'], hide: [] },
				'pubkey-pubkey':   { show: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_SECRET_KEY_name'], hide: ['TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'eap-aka-psk':     { show: ['TEL_TUNNEL2_SECRET_KEY_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'], hide: ['TEL_TUNNEL2_LEFT_CERT_name'] },
				'eap-aka-pubkey':  { show: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_SECRET_KEY_name'], hide: ['TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'eap-aka-eap-aka': { show: [], hide: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_SECRET_KEY_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'psk-eap-aka':     {show: [], hide: []},
				'pubkey-eap-aka':  {show: [], hide: []}
			};

		// 级联隐藏和显示
		if(tarDom && tarDom.length) {
			var tarVal = tarDom.val(),
				key = value + '-' + tarVal,
				showItems = cascade[key].show,
				hideItems = cascade[key].hide;

			showItems.map(function(code){
				$('#'+code).parent().show();
			});
			hideItems.map(function(code){
				$('#'+code).parent().hide();
			});
		}
	}
	function validRightAuth(el) {
		var value = $(el).val(),
			tarDom = $('#TEL_TUNNEL2_LEFT_AUTH_name'),
			cascade = {
				'psk-psk':         { show: ['TEL_TUNNEL2_SECRET_KEY_name'],  hide: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'psk-pubkey':      { show: ['TEL_TUNNEL2_SECRET_KEY_name'],  hide: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'pubkey-psk':      { show: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_SECRET_KEY_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'], hide: [] },
				'pubkey-pubkey':   { show: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_SECRET_KEY_name'], hide: ['TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'eap-aka-psk':     { show: ['TEL_TUNNEL2_SECRET_KEY_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'], hide: ['TEL_TUNNEL2_LEFT_CERT_name'] },
				'eap-aka-pubkey':  { show: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_SECRET_KEY_name'], hide: ['TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'eap-aka-eap-aka': { show: [], hide: ['TEL_TUNNEL2_LEFT_CERT_name','TEL_TUNNEL2_SECRET_KEY_name','TEL_TUNNEL2_RIGHT_SECRET_KEY_name'] },
				'psk-eap-aka':     {show: [], hide: []},
				'pubkey-eap-aka':  {show: [], hide: []}
			};

		// 级联隐藏和显示
		if(tarDom && tarDom.length) {
			var tarVal = tarDom.val(),
				key = tarVal + '-' + value,
				showItems = cascade[key].show,
				hideItems = cascade[key].hide;

			showItems.map(function(code){
				$('#'+code).parent().show();
			});
			hideItems.map(function(code){
				$('#'+code).parent().hide();
			});

			if(value == 'eap-aka') {
				tarDom.val(value);
				cascade['eap-aka-eap-aka'].hide.map(function(code){
					$('#'+code).parent().hide();
				});
			}
		}
	}
</script>