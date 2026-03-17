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
		<label for="TUNNEL_ENABLE_name">${TUNNEL_ENABLE_name }</label>
		<select id="TUNNEL_ENABLE_name" name="TUNNEL_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="TUNNEL_NAME_name">${TUNNEL_NAME_name }</label>
		<input id="TUNNEL_NAME_name" name="TUNNEL_NAME" title="${TUNNEL_NAME_title }" class="border border-box" 
			min_length="1" max_length="10" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^((\d*([a-z]|[A-Z])*)+)$/"/>
		<div id="TUNNEL_NAME_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_NAME_title }
		</div>
	</li>
	<li>
		<label for="TUNNEL_GATEWAY_name">${TUNNEL_GATEWAY_name }</label>
		<input id="TUNNEL_GATEWAY_name" name="TUNNEL_GATEWAY" title="${TUNNEL_GATEWAY_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="TUNNEL_GATEWAY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${TUNNEL_GATEWAY_title }
		</div>
	</li>
	
	<li>
		<label for="LEFT_IDENTIFIER_name">${LEFT_IDENTIFIER_name }</label>
		<input id="LEFT_IDENTIFIER_name" name="LEFT_IDENTIFIER" title="${LEFT_IDENTIFIER_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="LEFT_IDENTIFIER_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LEFT_IDENTIFIER_title }
		</div>
	</li>
	<li>
		<label for="RIGHT_IDENTIFIER_name">${RIGHT_IDENTIFIER_name }</label>
		<input id="RIGHT_IDENTIFIER_name" name="RIGHT_IDENTIFIER" title="${RIGHT_IDENTIFIER_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="RIGHT_IDENTIFIER_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RIGHT_IDENTIFIER_title }
		</div>
	</li>
	<li>
		<label for="AUTHBY_name">${AUTHBY_name }</label>
		<select id="AUTHBY_name" name="AUTHBY" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="psk">psk</option>
			<option value="cert">cert</option>
			<option value="aka_psk">aka_psk</option>
			<option value="aka_cert">aka_cert</option>
		</select>
	</li>
	<li>
		<label for="PRE_SHARED_KEY_name">${PRE_SHARED_KEY_name }</label>
		<input id="PRE_SHARED_KEY_name" name="PRE_SHARED_KEY" title="${PRE_SHARED_KEY_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="PRE_SHARED_KEY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${PRE_SHARED_KEY_title }
		</div>
	</li>
	<li>
		<label for="LEFTSOURCEIP_name">${LEFTSOURCEIP_name }</label>
		<input id="LEFTSOURCEIP_name" name="LEFTSOURCEIP" title="${LEFTSOURCEIP_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();"
			js_regex="/^(?:(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])|%config)$/"/>
		<div id="LEFTSOURCEIP_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LEFTSOURCEIP_title }
		</div>
	</li>
	<li>
		<label for="IKE_ENCRYPTION_name">${IKE_ENCRYPTION_name }</label>
		<select id="IKE_ENCRYPTION_name" name="IKE_ENCRYPTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="aes128">aes128</option>
			<option value="aes256">aes256</option>
			<option value="3des">3des</option>
		</select>
	</li>
	<li>
		<label for="ESP_ENCRYPTION_name">${ESP_ENCRYPTION_name }</label>
		<select id="ESP_ENCRYPTION_name" name="ESP_ENCRYPTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="aes128">aes128</option>
			<option value="aes256">aes256</option>
			<option value="3des">3des</option>
		</select>
	</li>
	<li>
		<label for="IKE_DH_GROUP_name">${IKE_DH_GROUP_name }</label>
		<select id="IKE_DH_GROUP_name" name="IKE_DH_GROUP" class="border border-box" onblur="createMML();">
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
		<label for="ESP_DH_GROUP_name">${ESP_DH_GROUP_name }</label>
		<select id="ESP_DH_GROUP_name" name="ESP_DH_GROUP" class="border border-box" onblur="createMML();">
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
		<label for="IKE_AUTHENTICATION_name">${IKE_AUTHENTICATION_name }</label>
		<select id="IKE_AUTHENTICATION_name" name="IKE_AUTHENTICATION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="sha1">sha1</option>
			<option value="sha512">sha512</option>
		</select>
	</li>
	<li>
		<label for="ESP_AUTHENTICATION_name">${ESP_AUTHENTICATION_name }</label>
		<select id="ESP_AUTHENTICATION_name" name="ESP_AUTHENTICATION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="sha1">sha1</option>
		</select>
	</li>
	
	<c:if test='${hardwareVersion == "EA4.0" || hardwareVersion == "EA4.0DUAL"}'>
		<li>
			<label for="KEYLIFE_name">${KEYLIFE_name }</label>
			<input id="KEYLIFE_name" name="KEYLIFE" title="${KEYLIFE_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateKeylife(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
			<div id="KEYLIFE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${KEYLIFE_title }
			</div>
		</li>
		<li>
			<label for="IKELIFETIME_name">${IKELIFETIME_name }</label>
			<input id="IKELIFETIME_name" name="IKELIFETIME" title="${IKELIFETIME_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);validateIKElifeTime(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
			<div id="IKELIFETIME_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${IKELIFETIME_title }
			</div>
		</li>
		<li>
			<label for="REKEYMARGIN_name">${REKEYMARGIN_name }</label>
			<input id="REKEYMARGIN_name" name="REKEYMARGIN" title="${REKEYMARGIN_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);ValidateMinTime(event);createMML();" js_regex="/^\d+[m|h|d]{1}$/"/>
			<div id="REKEYMARGIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${REKEYMARGIN_title }
			</div>
		</li>
	</c:if>
	
	<c:if test='${hardwareVersion != "EA4.0" && hardwareVersion != "EA4.0DUAL"}'>
		<li>
			<label for="KEYLIFE_name">${KEYLIFE_name }</label>
			<input id="KEYLIFE_name" name="KEYLIFE" title="${KEYLIFE_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d+[s|m|d]{1}$/"/>
			<div id="KEYLIFE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${KEYLIFE_title }
			</div>
		</li>
		<li>
			<label for="IKELIFETIME_name">${IKELIFETIME_name }</label>
			<input id="IKELIFETIME_name" name="IKELIFETIME" title="${IKELIFETIME_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d+[s|m|d]{1}$/"/>
			<div id="IKELIFETIME_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${IKELIFETIME_title }
			</div>
		</li>
		<li>
			<label for="REKEYMARGIN_name">${REKEYMARGIN_name }</label>
			<input id="REKEYMARGIN_name" name="REKEYMARGIN" title="${REKEYMARGIN_title }" class="border border-box" 
				min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d+[s|m|d]{1}$/"/>
			<div id="REKEYMARGIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
				${REKEYMARGIN_title }
			</div>
		</li>
	</c:if>
	
	<li>
		<label for="KEYINGTRIES_name">${KEYINGTRIES_name }</label>
		<input id="KEYINGTRIES_name" name="KEYINGTRIES" title="${KEYINGTRIES_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^(?:\d+|%forever)$/"/>
		<div id="KEYINGTRIES_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${KEYINGTRIES_title }
		</div>
	</li>
	<li>
		<label for="DPDACTION_name">${DPDACTION_name }</label>
		<select id="DPDACTION_name" name="DPDACTION" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="none">none</option>
			<option value="clear">clear</option>
			<option value="hold">hold</option>
			<option value="restart">restart</option>
		</select>
	</li>
	<li>
		<label for="DPDDELAY_name">${DPDDELAY_name }</label>
		<input id="DPDDELAY_name" name="DPDDELAY" title="${DPDDELAY_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="/^\d+[s|m|d]{1}$/"/>
		<div id="DPDDELAY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${DPDDELAY_title }
		</div>
	</li>
	<li>
		<label for="RIGHT_SUBNET_name">${RIGHT_SUBNET_name }</label>
		<input id="RIGHT_SUBNET_name" name="RIGHT_SUBNET" title="${RIGHT_SUBNET_title }" class="border border-box" 
			min_length="0" max_length="256" onblur="validateMaxAndMinLength(event);validateSpecialChar(event);createMML();" js_regex="no_zh"/>
		<div id="RIGHT_SUBNET_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${RIGHT_SUBNET_title }
		</div>
	</li>
</ul>
<script>
	function validateKeylife(e) {
		var ele = $(e["target"]);
		
		var mustVal = ele.attr("must") == undefined ? "" : ele.attr("must");
		//如果不是必填项，且输入的值为空，则不做下面的验证
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
			$('#IKELIFETIME_name').blur();
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
		var value = $('#IKELIFETIME_name').val(),
			tarVal = $('#KEYLIFE_name').val(),
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
			value = $('#REKEYMARGIN_name').val(),
			reg = /^\d+[s|m|h|d]{1}$/,
			minTypes = {
				'RTS&RTD': '5m'
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
		}
	}

	function is3Times() {
		var value = $('#KEYLIFE_name').val(),
			tarVal = $('#REKEYMARGIN_name').val(),
			reg = /^\d+[s|m|h|d]{1}$/,
			bool = false;

		if(reg.test(value) && getEnbType() != '') {
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