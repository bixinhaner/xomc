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
        <title>sunsea aiot</title>
        <link href="${ctx}/favicon.ico"  rel="Shortcut Icon" type="image/x-icon"/>
        <link rel="stylesheet" type="text/css" href="${ctx}/css/iconfont.css?_=${omc_ver}" />
        <link rel="stylesheet" type="text/css" href="${ctx}/skin/${manufacturer}/skin.css?_=${omc_ver}" />
        <script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery-3.5.1.min.js?_=${omc_ver}"></script>
    	<script type="text/javascript" src="${ctx}/js/crypto-js.min.js?_=${omc_ver}"></script>
    	<style>
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
				border:1px solid #ccc !importnat;
			}
    	</style>
		<script type="text/javascript">
			try{
				var securityKey = 'BaiCellsSecurity';
				var key = CryptoJS.enc.Utf8.parse(securityKey);
			}catch(e){}
			// AES 加密
			function AesEncrypt(word) {
				try{
					let srcs = CryptoJS.enc.Utf8.parse(word),
						encrypted = CryptoJS.AES.encrypt(srcs, key, {
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
					var skey = newKey || key, 
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
			<%-- 提交表单 --%>
			function submit_form() {
				var bool = checkForm(document.querySelector('#loginfrom'));
				if(!bool) return false;
				
				$("#password").val(AesEncrypt($("#password").val()));
				
				$('#loginfrom').addClass('readonly');
		        sessionStorage.locked = '';
				$("#btn_submit").click();
			}
			function isChrome(){
				var agent = window.navigator.userAgent,
					reg = /^[\s\S]+Chrome\/[0-9\. ]+Safari\/[0-9\.]+$/;
	
				return reg.test(agent);
			}
			$(function(){
				var uiCustomOld = {
						"ui_color":"#1913BB",
						"ui_login_background":"./images/login/login_bg.png",
						"ui_menu_logo_up":"./images/login/nav_logo_collapse.png",
						"ui_menu_logo_down":"./images/login/login_logo.png",
						"ui_restore":"true",
				};
				var uiCustom = {
						"ui_color":"",
						"ui_login_background":"",
						"ui_menu_logo_up":"",
						"ui_menu_logo_down":"",
						"ui_restore":""
				}
				
				
				Object.assign(uiCustom,uiCustomOld)
				
				$.post("${ctx}/ui/customization/getCustomizationInfo",{},function(data){
			 	   if(data["ui_restore"] == "true"){
			 		  Object.assign(uiCustom,uiCustomOld);
					  document.documentElement.style.setProperty("--logo-big-white" ,'url('+ uiCustom.ui_menu_logo_down +')');
					document.documentElement.style.setProperty("--main-bg" ,'url('+ uiCustom.ui_login_background +')');
			 	   }else {
			 		  Object.assign(uiCustom,data);
					  document.documentElement.style.setProperty("--logo-big-white" ,'url('+ uiCustom.ui_menu_logo_down +')');
					document.documentElement.style.setProperty("--main-bg" ,'url('+ uiCustom.ui_login_background +')');
			 	   }
			    },"json")
				
				
				
				
				var captchaEnable = ${validatecode}; 
				
				if(captchaEnable){				
					$(".login_verifyCode_div").show();				
					$(".loginformbox").removeClass("lockformbox");
					verifyCode = new GVerify("checkCode");
				}else{
					$(".login_verifyCode_div").hide();
					$(".loginformbox").addClass("lockformbox");
				}
				
				
				<%-- 用户名、密码输入框，回车事件-提交表单 --%>
				$(".input_").bind("keyup", function(e) {
					if (e.keyCode == 13) {
						$('#loginfrom input').blur();

						submit_form();
						e.preventDefault();
						e.stopPropagation();
						return false;
					}
				});
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
		</script>
    </head>
    <body class="loginbody">
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
		<form id="loginfrom" action="${ctx }/sys/login/userLoad.htm" method="post">  
			<input type="hidden" name="userInfo.logintype" value="0" />
			<div class="login_bar_logo_div"></div>
			<div class="login_user_div">
				<div class="input_ico_user_div"></div>
				<input type="text" id="uid" class="input_" name="userInfo.usercode" placeholder="<%=rb.getString("QingShuRuYongHuMing")%>" value="${userInfo.usercode}"
					required oninvalid="setCustomValidity('<%=rb.getString("QingShuRuYongHuMing")%>')" oninput="setCustomValidity('')">
			</div>
			<div class="login_pwd_div">
				<div class="input_ico_pwd_div"></div>
				<input id="password" type="password" class="input_" name="userInfo.password" placeholder="<%=rb.getString("QingShuRuMiMa")%>"
				 	required oninvalid="setCustomValidity('<%=rb.getString("QingShuRuMiMa")%>')" oninput="setCustomValidity('')">
				<span style="color:red; font-size: 13px; display:block;text-align:left;margin-left:41px;margin-top:8px;">${userInfo.errinfo}</span>
			</div>
			<div class="login_btn_div" onclick="submit_form()">
				<span style="line-height: 38px;	font-size: 20px; color: #FFFFFF;"><%=rb.getString("DengLu")%></span>
			</div>
			<input id="btn_submit" type="submit" value="<%=rb.getString("DengLu")%>" style="display: none;"/>
		</form>
    </body>
</html>