<%@ page contentType="text/html;charset=UTF-8"%>
<%
	response.setHeader("Pragma","No-cache");
	response.setHeader("Cache-Control","no-cache");
	response.setDateHeader("Expires", 0);
%>
<%@ include file="/common/taglibs.jsp"%>

<!DOCTYPE html>
<html>
    <head>
		<meta charset="UTF-8" />
        <title></title>
        <link href="${ctx}/favicon.ico"  rel="Shortcut Icon" type="image/x-icon"/>
        <link rel="stylesheet" type="text/css" href="${ctx}/css/iconfont.css?_=${omc_ver}" />
        <link rel="stylesheet" type="text/css" href="${ctx}/skin/${manufacturer}/skin.css?_=${omc_ver}" />
		<link rel="stylesheet" type="text/css" href="${ctx}/css/element.css?_=${omc_ver}"/>
		<link rel="stylesheet" type="text/css" href="${ctx}/css/app.css?_=${omc_ver}"/>
		<script type="text/javascript" src="${ctx}/js/element/vue.min.js?_=${omc_ver}"></script>
    	<script type="text/javascript" src="${ctx}/js/element/axios.min.js?_=${omc_ver}"></script>
		<script type="text/javascript" src="${ctx}/js/element/viewGrid.js?_=${omc_ver}"></script>
        <script type="text/javascript" src="${ctx}/js/gVerify.js?_=${omc_ver}"></script>
    	<script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery-3.5.1.min.js?_=${omc_ver}"></script>
    	<script type="text/javascript" src="${ctx}/js/element/element-en.js?_=${omc_ver}"></script>
    	<script type="text/javascript" src="${ctx}/js/jsencrypt.min.js?_=${omc_ver}"></script>
    	<script type="text/javascript" src="${ctx}/js/crypto-js.min.js?_=${omc_ver}"></script>
        <style type="text/css">
			.input_ {
				border-style: none;
				height: 30px;
				margin-top: 3px;
				outline: none;
			}
			.code {
	            font-family: Arial;
	            font-style: italic;
	            font-weight: bold;
	            border: 0;
	            letter-spacing: 3px;
	            color: blue;
	        }
	        form.readonly::before {
	        	content: '';
	        	display: inline-block;
	        	position: absolute;
	        	top: 0;
	        	right: 0;
	        	bottom: 0;
	        	left: 0;
	        }
	        input:-webkit-autofill,input:-webkit-autofill:hover,input:-webkit-autofill:focus,
	        input:-webkit-autofill:active {
				-webkit-transition-delay: 9999s;
				transition-delay: 9999s;
				-webkit-transition: color 9999s ease-out, background-color 9999s ease-out;
				transition: color 9999s ease-out, background-color 9999s ease-out;
				background: transparent !important;
			}
			input:-webkit-autofill{
				-webkit-box-shadow: 0 0 0px 1000px transparent inset;
				box-shadow: 0 0 0px 1000px transparent inset;
				border:1px solid #ccc !important;
			}
			#loginMain{
				width:100%;
				height:100vh;
			}
			.verifyMain {
				width:100%;
				height:100vh;
				background:#fff;
			}
			#checkCode #verifyCanvas {
				border-radius: 0 10px 10px 0;
			}
		</style>
    </head>
    <body>
		<div class="loginbody" id="noLogoLogin">
			<div class='lockIconItem'>
				<div style="margin:auto;display:flex">
					<p style='line-height:40px;'><span class="el-icon el-icon-circle-warning" style="margin-right:10px;"></span><span style="font-size:14px;"><%=rb.getString("JianRongTiShi")%>:</span></p>
					<div style="display:inline-block;margin-left:10px">
						<span class="browser_icon google_icon"></span>
						<span class="browser_icon safari_icon"></span>
						<span class="browser_icon firefox_icon"></span>
						<span class="browser_icon edge_icon"></span>
					</div>
				</div>
			</div>
			<div id="loginMain" class="loginformbox min-width-600">
				<form id="loginfrom" action="" method="post"> 
					<input type="hidden" name="userInfo.logintype" value="0" />
					<div class="login_bar_logo_div"></div>
					<div class="login_user_div">
						<div class="input_ico_user_div"></div>
						<input style="width: 80%;" type="text" id="uid" readonly="readonly" autocomplete="off" class="input_" placeholder="<%=rb.getString("QingShuRuYongHuMing")%>" value="${userInfo.usercode}"
								required onblur="this.setAttribute('readonly',true)" onfocus="this.removeAttribute('readonly')">
						<span style="color:red; font-size: 13px; display:block;text-align:left;margin-left:41px;margin-top:8px; "></span>

						<input type="hidde" style="display: none;" id="hide_uid" autocomplete="off" name="userInfo.usercode">
					</div>
					<div class="login_pwd_div">
						<div class="input_ico_pwd_div"></div>
						<div v-if="isBrowserAutoRecordPass" style="display: inline-block;width: 80%;">
							<el-password style="width: 100%;" v-model="password" size="mini" @input="passwordChange" placeholder='<%=rb.getString("QingShuRuMiMa")%>'></el-password>
							<input id="password" style="width: 100%;display:none;" type="text" class="input_" name="userInfo.password" placeholder="<%=rb.getString("QingShuRuMiMa")%>">
						</div>
						<input v-if="!isBrowserAutoRecordPass" style="width: 80%;" id="password" type="password" class="input_" name="userInfo.password" placeholder="<%=rb.getString("QingShuRuMiMa")%>"
						required oninvalid="setCustomValidity('<%=rb.getString("QingShuRuMiMa")%>')" oninput="setCustomValidity('')">
						<span style="color:red; font-size: 13px; display:block;text-align:left;margin-left:41px;margin-top:8px; "></span>
					</div>
					<div class="login_verifyCode_div">
						<input id="verifyCode" name="userInfo.loginCaptcha" type="text" class="input_" placeholder="<%=rb.getString("QingShuRuYanZhengMa")%>" style='width: calc(100% - 105px); padding: 0 0 0 20px;'/>
						<div id="checkCode" style="display:inline-block;width: 100px;height:5vh;vertical-align : top"></div>
						<span style="color:red; font-size: 13px; display:block;text-align:left;margin-top:8px;"></span>
					</div> 
					<!-- 获取的userinfo 返回的错误信息 -->
					<span id='errorInfoTip' style="color:red; font-size: 12px; display:block;text-align:left;margin-left:35px; width: 80%;">${userInfo.errinfo}</span>
					<!-- 验证码开关为关的时候，不传这个参数 -->
					<input type="hidden" id='verifyErrorFlag' name="userInfo.verifyFlag" value="1" style='display: none;'/>
					<div class="login_btn_div" onclick="submit_form()" id="splinter_omc_login_btn">
						<span style="line-height: 38px;	font-size: 20px; color: #FFFFFF;display: block; text-align: center;"><%=rb.getString("DengLu")%></span>
					</div>
					<input type="hidden" name="userInfo.publicKey" value="${publicKey}"/>
					<input type="hidden" name="userInfo.captchaId" value="${userInfo.captchaId}"/>
					<input id="btn_submit" type="submit" value="<%=rb.getString("DengLu")%>" style="display: none;"/>
				</form>
			</div>
			<!-- 双因子短信验证 -->
			<div id="verifyMain" class="verifyMain"></div>
		</div>
		<script>
			var noLogoLoginVue = new Vue({
					el:'#noLogoLogin',
					data(){
						return{
							password:'',
							isBrowserAutoRecordPass:false
						}
					},
					computed: {
						
					},
					methods:{
						// 初始化请求
						init(){
							var vm = this;
							$.ajax({
								type: "post",
								url: '${ctx}/sys/login/getIsBrowserAutoRecordPass',
								data: {},
								async: false,
								dataType:"json",
								success: function(data) {
									vm.isBrowserAutoRecordPass = data ? true : false;

									vm.$nextTick(function(){
										vm.bindEvent();
									})
								}
							});
						},
						// 密码改变事件
						passwordChange(val){
							var vm = this;
							$("#password").val(val)
						},
						// 密码回车事件
						passwordEnterClick(){
							$('#loginfrom input').blur();
						
							submit_form();
							e.preventDefault();
							e.stopPropagation();
							return false;
						},
						bindEvent() {
							/* 用户名、密码输入框，回车事件-提交表单 */
							$(".input_, .el-input__inner").bind("keypress", function(e) {
								if (e.keyCode == 13) {
									$('#loginfrom input').blur();
									
									submit_form();
									e.preventDefault();
									e.stopPropagation();
									return false;
								}
							});
							$("#uid").keyup(function(e){
								if (e.keyCode != 13) {
									if($(this).val().length>0){
										$(this).next().html("");
									}
								}
							})
							$("#password").keyup(function(e){
								if (e.keyCode != 13) {
									if($(this).val().length>0){
										$(this).next().html("");
									}
								}
							})
							$("#verifyCode").keyup(function(e){
								if (e.keyCode != 13) {
									
									if($(this).val().length>0){
										$("#checkCode").next().html("");
									}
								}
							})
						}
					},
					mounted(){
						this.init();
					}
				})
		</script>

		<script type="text/javascript">
			//验证码开关 0- 关闭， 1-打开
			var captchaEnable = '${userInfo.verificationCode}';
			//设置页面可配置 0 次，此时需要增加一个接口进行校验是否显示验证码
			var newCaptchaEnable = '';
			$.ajax({
				type: "post",
				url: '${ctx}/sys/login/getVerficationCode',
				data: {},
				async: false,
				dataType:"json",
				success: function(data) {
					if(data.verification == 'true'){
						//则需要显示验证码
						newCaptchaEnable = 'true';
					}else {
						newCaptchaEnable = 'false';
					}
				}
			});

			// RSA 加密
			function RsaEncrypt(data, newKey) {
				var publicKey = newKey || '${publicKey}';

				var encryptor = new JSEncrypt();
				encryptor.setPublicKey(publicKey);

				return encryptor.encrypt(data);
			}
			
			var encrypt_key = '';
			// AES 加密
			function AesEncrypt(word) {
				try{
					let srcs = CryptoJS.enc.Utf8.parse(word),
						encrypted = CryptoJS.AES.encrypt(srcs, CryptoJS.enc.Utf8.parse('BaiCellsSecurity'), {
							mode: CryptoJS.mode.ECB,
						}),
						base64Data = CryptoJS.enc.Base64.stringify(encrypted.ciphertext);
					
					return base64Data;
				}catch(e) {
					return word;
				}
			}
			// AES 解密
			function AesDecrypt(word, newKey) {
				try{
					var skey = newKey || encrypt_key, 
						encryptedHexStr = CryptoJS.enc.Base64.parse(word),
						srcs = CryptoJS.enc.Base64.stringify(encryptedHexStr),
						decrypt = CryptoJS.AES.decrypt(srcs, skey, {
							mode: CryptoJS.mode.ECB,
						}),
						decryptedStr = decrypt.toString(CryptoJS.enc.Utf8);
					
					return JSON.parse(decryptedStr.toString());
				}catch(e) {
					return word;
				}
			}

			$.ajaxSetup({
				beforeSend: function(){
					var paramstr = arguments[1].data,bool=true;
					if(paramstr){
						var paramArr = paramstr.split('&');
						paramArr.map(function(item){
							var codes = item.split('=');
							if(!validXSS(codes[1])) bool = false;
						});
					}
					if(!bool) {
						try{
							$('.messager-window:contains(tips)').panel('destroy');
						}catch(e){}
						$("#checkCode").next().html('illegal input: script、iframe、onclick、onfocus、onerror or onchange');
					}
					return bool;
				}
			});
			/* 检查是否含XSS脚本 */
			function validXSS(val){
				if(filterSpecialCharactersEnable != '1') return true;
				
				var words = ['script','iframe','onclick','onfocus','onerror','onchange'],bool=true;
				words.map(function(item){
					if(val && val.toLowerCase().indexOf(item)>=0) bool = false;
				});
				return bool;
			}
			function checkForm(form){
				if(filterSpecialCharactersEnable != '1') return true;
				
				var bool = true;
				try{
					$(':input',form).each(function(n,item){
						var namedItem = form[item.name];
						if(namedItem){
							if(!validXSS(namedItem.value)) bool = false;
						}
					})
				}catch(e){}
				if(!bool) {
					try{
						$('.messager-window:contains(tips)').panel('destroy');
					}catch(e){}
					$("#checkCode").next().html('illegal input: script、iframe、onclick、onfocus、onerror or onchange');
				}
				return bool;
			}
			
			/*<%--格式化时间--%>*/
			function dateformatter(date) {
				var y = date.getFullYear();
				var m = date.getMonth() + 1;
				var d = date.getDate();
				var h = date.getHours();
				var min = date.getMinutes();
				var s = date.getSeconds();
				return y + '-' + (m < 10 ? ('0' + m) : m) + '-' + (d < 10 ? ('0' + d) : d) + " " + (h < 10 ? ('0' + h) : h) + ":" + (min < 10 ? ('0' + min) : min) + ":" + (s < 10 ? ('0' + s) : s);
			}
			
			var verifyCode = "";
			
			//var captchaEnable = ${validatecode};  // long long ago
			<%-- 提交表单 --%>
			function submit_form() {
				var bool = checkForm(document.querySelector('#loginfrom'));
				if(!bool) return false;
				var uid = $.trim($('#uid').val());
				var password = $.trim($('#password').val());	

				window.sessionStorage.setItem('userName',uid);
				window.sessionStorage.setItem('userPassword',password);
				
				if($('#loginfrom').hasClass('readonly')) {
					return false;
				}

				//未输入密码-提示请输入密码；密码错误-根据 userinfo.errinfo 提示；再次点击登录时，需userinfo.errinfo 内容置空；
				$("#errorInfoTip").html('');
				if(uid ==""){
					$("#uid").next().html('<%=rb.getString("QingShuRuYongHuMing")%>');
					return false;
				}
				if($("#password").val()==""){
					$("#password").next().html('<%=rb.getString("QingShuRuMiMa")%>');
					return false;
				}
				//点击登录 判断是否支持验证码功能;  验证码由前端控制 是否输入正确；
				
				//验证码开关 0- 关闭， 1-打开
				if(captchaEnable == '1' || newCaptchaEnable == 'true'){
					$('#verifyErrorFlag').attr('disabled',false);
					
					if($("#verifyCode").val()==""){
						$("#checkCode").next().html('<%=rb.getString("QingShuRuYanZhengMa")%>');
						$("#errorInfoTip").html('');
						return false;
					}else{
						/*
						var res = verifyCode.validate($("#verifyCode").val());
						//true false 布尔型
						//userInfo.verifyFlag  告知后端当前验证码是正确还是错误；0-错误，1-正确（为了统计验证码的错误次数）； 验证码开关为关，不传此参数；
						if(res){
							$('#verifyErrorFlag').val('1');
						}else{
							$('#verifyErrorFlag').val('0');
							$("#verifyCode").focus();
						}
						*/
					}
				}else{
					//开关是关的时候，不传userInfo.verifyFlag；
					$('#verifyErrorFlag').attr('disabled',true);
				}
				//$('#loginfrom').addClass('readonly');
				sessionStorage.locked = '';
				//$("#btn_submit").click();
				
				var dateStr = dateformatter(new Date()).replace(' ','-').substr(3);
				//登录时，需判断是否是短信验证模式  
				$.post("${ctx}/system/sysuser/msgmod.action",{userCode: uid, rd: AesEncrypt(dateStr)}, function(data){
					var result = AesDecrypt(data.result||'4yNThc1APBpYgFK4s6OvOw==', CryptoJS.enc.Utf8.parse(dateStr));
					if (result.msgEnable == "false") {	
						$('#loginfrom').addClass('readonly');

						var canLogin = false;
						$.ajax({
							type: "post",
							url: '${ctx}/sys/login/isAlreadyLogIn.action',
							data: {userCode: $("#uid").val()},
							async: false,
							dataType:"text",
							success: function(data) {
								var isLogined = ['true', true].includes(data);

								if(isLogined) {
									noLogoLoginVue.$confirm('<%=rb.getString("YongHuQiangZhiDengChu")%>','<%=rb.getString("QueRen")%>',{
										confirmButtonText:'<%=rb.getString("QueDing")%>',
										cancelButtonText:'<%=rb.getString("QuXiao")%>',
										type:'warning',
									}).then(function(r){
										if(r == 'confirm') {
											// 强制下线
											$.ajax({
												type: "post",
												url: '${ctx}/sys/login/forceLogoutUser1.action',
												async: false,
												data: {user_names: $("#uid").val()},
												dataType:"json",
												success: function(data) {
													canLogin = true;
												}
											});

											if(canLogin) {
												$.ajax({
													type: "post",
													url: '${ctx}/sys/login/getPublicKey',
													async: false,
													dataType:"text",
													success: function(data) {
														var raskey = data;
												
														$("#hide_uid").val(RsaEncrypt($("#uid").val(), raskey));
														$("#password").val(RsaEncrypt($("#password").val(), raskey));
														$('input[name="userInfo.publicKey"]').val(raskey);
													}
												});

												$('#loginfrom').attr('action',"${ctx }/sys/login/userLoad.htm");
												$("#btn_submit").click();	
											}
										}
									}).catch(()=>{})
								}else {
									$.ajax({
										type: "post",
										url: '${ctx}/sys/login/getPublicKey',
										async: false,
										dataType:"text",
										success: function(data) {
											var raskey = data;
									
											$("#hide_uid").val(RsaEncrypt($("#uid").val(), raskey));
											$("#password").val(RsaEncrypt($("#password").val(), raskey));
											$('input[name="userInfo.publicKey"]').val(raskey);
										}
									});

									$('#loginfrom').attr('action',"${ctx }/sys/login/userLoad.htm");
									$("#btn_submit").click();	
								}
							}
						});						              	 
					} else {	                	
						//用户账号校验接口
						var paramsUser= {
								userCode: uid,
								password: AesEncrypt(password),
								rd: AesEncrypt(dateStr)
							};
						$.post("${ctx}/system/sysuser/validateUserInfo.action", paramsUser, function(data){
							var rst = AesDecrypt(data.result||'4yNThc1APBpYgFK4s6OvOw==', CryptoJS.enc.Utf8.parse(dateStr));
							
							if (rst.success) {			                	                			                	 
								//跳转到双因子短信验证页面(verification.jsp)
								$('#verifyMain').load("${ctx}/msg/toVerification.action",paramsUser, function(data){
									
								}); 			                	 
								$('#loginMain').hide();
								$('#verifyMain').show();		
							} else {
								//提示错误信息
								$("#password").next().html(rst.message);
							} 
						}, "json"); 
					}  
					
				}, "json"); 
			}
			function isChrome(){
				var agent = window.navigator.userAgent,
					reg = /^[\s\S]+Chrome\/[0-9\. ]+(\sMobile\s)?Safari\/[0-9\.]+$/;

				return reg.test(agent);
			}
			$(function(){
				var uiCustomOld = {
						"ui_color":"#FF4614",
						"ui_login_background":"./images/login/login_bg.png",
						"ui_menu_logo_up":"./images/login/nav_logo_collapse.png",
						"ui_menu_logo_down":"./images/login/login_logo.png",
						"ui_restore":"true",
						"ui_omc_name":"OMC"
				};
				var uiCustom = {
						"ui_color":"",
						"ui_login_background":"",
						"ui_menu_logo_up":"",
						"ui_menu_logo_down":"",
						"ui_restore":"",
						"ui_omc_name":"OMC"
				}
				
				
				Object.assign(uiCustom,uiCustomOld);
				
				$.post("${ctx}/ui/customization/getCustomizationInfo",{},function(data){
					if(data["ui_restore"] == "true"){
						Object.assign(uiCustom,uiCustomOld);
						document.title = uiCustom.ui_omc_name;
						document.documentElement.style.setProperty("--logo-big-white" ,'url('+ uiCustom.ui_menu_logo_down +')');
						document.documentElement.style.setProperty("--main-bg" ,'url('+ uiCustom.ui_login_background +')');
					}else {
						Object.assign(uiCustom,data);
						document.title = uiCustom.ui_omc_name;
						//document.documentElement.style.setProperty("--logo-big-white" ,'url('+ uiCustom.ui_menu_logo_down +')');
						//document.documentElement.style.setProperty("--main-bg" ,'url('+ uiCustom.ui_login_background +')');
						setStyleProp("--logo-big-white" , uiCustom.ui_menu_logo_down);
						setStyleProp("--main-bg" , uiCustom.ui_login_background);
					}
		 			var colorUIRgb = changeColor(uiCustom.ui_color);
					document.documentElement.style.setProperty("--main-color-rgba1" , colorUIRgb);
					updateFavicon(uiCustom.ui_color);
				},"json")
				
				
				
				
				//var captchaEnable = ${validatecode}; 
				
				if(captchaEnable == '1' || newCaptchaEnable == 'true'){
					$(".login_verifyCode_div").show();				
					$(".loginformbox").removeClass("lockformbox");

					$.post("${ctx}/sys/login/getVerifyCode").then(function(code){
						/*
						verifyCode = new GVerify({
							id: "checkCode", 
							preset: code,
							callback: function(p) {
								$.post("${ctx}/sys/login/getVerifyCode").then(function(data){
									p.options.preset = data;
									p.refresh();
								})
							}
						});
						*/
						
						$('#checkCode').html('<img src="' + code.captcha + '" style="width: 100%;height: 100%;" />');
						$('input[name="userInfo.captchaId"]').val(code.captchaId);

						$('#checkCode').on('click',function(){
							$.post("${ctx}/sys/login/getVerifyCode").then(function(data){
								$('#checkCode').html('<img src="' + data.captcha + '" style="width: 100%;height: 100%;" />');
								$('input[name="userInfo.captchaId"]').val(code.captchaId);
							})
						})
					})
				}else{
					$(".login_verifyCode_div").hide();
					$(".loginformbox").addClass("lockformbox");
				}
				
				
				try{
					sessionStorage.locked = '';
				}catch(e){}
				var explorer = window.navigator.userAgent;
				var explorFlag = false;
				if(explorer.indexOf("Firefox") > 0){//火狐浏览器
					explorFlag = true;
				}else if(explorer.indexOf("Edg") > 0){//Edge浏览器
					explorFlag = true;
				}else if(explorer.indexOf("Mac") > 0){//Safari浏览器
					explorFlag = true;
				}else if(isChrome()){//Google浏览器
					explorFlag = true;
					// 完全使用chromiun内核，且不做任务差异处理的
					try{
						var mimeTypes = Array.from(window.navigator.mimeTypes).map(function(item){ return item.type;}),
							excludes = ['application/360softmgrplugin','application/x-ppapi-widevine-cdm'];
						
						excludes.map(function(item){
							if(mimeTypes.includes(item)) explorFlag = false;
						})
					}catch(e){}
				}else{
					explorFlag = false;
				}
				if(explorFlag){//此浏览器是支持的浏览器
					$(".lockIconItem").hide();
				}else{
					$(".lockIconItem").show();
				}
			});

			function base64ToBlob(base64) {
				const base64Content = base64.split(',')[1];
				const byteString = atob(base64Content);
				const mimeType = base64.split(',')[0].split(':')[1].split(';')[0];
				const byteArray = new Uint8Array(byteString.length);
				for(let i = 0; i < byteString.length; i++) {
					byteArray[i] = byteString.charCodeAt(i);
				}

				return new Blob([byteArray], {type: mimeType});
			}
			function setStyleProp(prop, base64) {
				if(isValidBase64(base64)) {
					const regex = /^/;
					const blob = base64ToBlob(base64);
					const url = URL.createObjectURL(blob);
					document.documentElement.style.setProperty(prop ,'url(' + url + ')');
				}else {
					document.documentElement.style.setProperty(prop ,'url(' + base64 + ')');
				}
			}
			function isValidBase64(base64Str) {
				return (base64Str||'').length > 50;
			}
			//颜色值转换  十六进制转换为rgb 
			function changeColor(col){
				
				var newStr = (col.toLowerCase()).replace(/\#/g,'');
				var len = newStr.length;
				
				if(len == 3){
					let t = '';
					for (var i=0;i<len;i++){
						t += newStr.slice(i,i+1).concat(newStr.slice(i,i+1))
					}
					newStr = t;
				}
				
				let arr = [];
				for(var i=0;i<6;i=i+2){
					let s = newStr.slice(i,i+2)
					arr.push(parseInt("0x"+s))
				}
				return arr.join(",")
			}
			
			function createRoundedFavicon(color = '#FF4614', radius = 10) {
				// 1. 创建Canvas画布（推荐32x32尺寸）
				const canvas = document.createElement('canvas');
				const size = 32; // 标准favicon尺寸
				canvas.width = size;
				canvas.height = size;
				const ctx = canvas.getContext('2d');
				
				// 2. 绘制圆角矩形
				ctx.clearRect(0, 0, size, size);
				ctx.fillStyle = color;
				
				// 圆角矩形路径算法
				ctx.beginPath();
				ctx.moveTo(radius, 0);
				ctx.lineTo(size - radius, 0);
				ctx.arcTo(size, 0, size, radius, radius);
				ctx.lineTo(size, size);
				ctx.arcTo(size, size, size, size, 0);
				ctx.lineTo(radius, size);
				ctx.arcTo(0, size, 0, size - radius, radius);
				ctx.lineTo(0, 0);
				ctx.arcTo(0, 0, 0, 0, 0);
				ctx.closePath();
				ctx.fill();
				
				// 3. 转换为Data URL
				return canvas.toDataURL('image/png');
			}

			// 4. 更新页面favicon
			function updateFavicon(color) {
				const faviconUrl = createRoundedFavicon(color);
				let link = document.querySelector("link[rel*='icon']");
				
				if (!link) {
					link = document.createElement('link');
					link.rel = 'icon';
					document.head.appendChild(link);
				}
				
				link.href = faviconUrl;
				link.type = 'image/png';
			}
			// 示例：生成绿色圆角favicon
			// updateFavicon('#4CAF50');
		</script>
    </body>
</html>