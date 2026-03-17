<%@ page contentType="text/html;charset=UTF-8"%>
<%
	response.setHeader("Pragma","No-cache");
	response.setHeader("Cache-Control","no-cache");
	response.setDateHeader("Expires", 0);
%>
<%@ include file="/common/taglibs.jsp"%>

<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <title>BaiOMC</title>
    <link href="${ctx}/favicon.ico"  rel="Shortcut Icon" type="image/x-icon"/>
    <link rel="stylesheet" type="text/css" href="${ctx}/css/iconfont.css?_=${omc_ver}" />
    <link rel="stylesheet" type="text/css" href="${ctx}/skin/${manufacturer}/skin.css?_=${omc_ver}" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
   	<style type="text/css">
		/*短信验证*/
		input,
		button {
		    background: none;
		    outline: none;		   
		}
	
		input:disabled,
		button:disabled {
			background-color: #F5F7FA;
			outline:none;
		}

		input:focus,
		button:focus {
		    border:none;
			outline:none;
		}
		input::-ms-clear{display:none;}
		input::-ms-reveal{display:none;}
		
		input:-webkit-autofill,
		textarea:-webkit-autofill,
		select:-webkit-autofill {
		    -webkit-box-shadow: 0 0 0 1000px white inset
		}
		
		.verification-warp {
		    width: 100%;
		    height: 100vh;
		    background:#fff;
		}
		
		.verification-warp .verification-header {
		    width: 100%;
		    height: 60px;
		    background-color: #4D84FF;
		}
		
		.verfication-logo {
			width:190px;
			height: 60px;
    		background: #4D84FF;
    		float:left;
		}
		.verfication-logo .logo-tip::before {
			color:#fff;
			margin-left:46px;
			font-size: 94px;
		    height: 20px; 
		    margin-top: -20px;
		    float: left;
		}
		.verfication-logo .verification-version {		   
		    width: 40px;
		    display: inline-block;
		    vertical-align: top;
		    margin-top: 26px;
		    text-align:center;
		    border-radius: 10px;
		    color: #fff;
		    float:right;
		    font-size:12px;
		    background-color: rgba(255,255,255,.19);		    
		}
		
		.verification-warp .verification-header .verify-back {
		    float: right;
		    color: #fff;
		    font-size: 14px;
		    margin-right: 56px;
		    line-height: 60px;
		    cursor: pointer;
		}
		.verification-warp .verification-header .verify-back i {
			margin-left:6px;
			font-size:12px;
			color:#fff;
		}
		
		.verification-main {
		    width: 768px;
		    height: 392px;
		    border: 1px solid #DEDFE6;
		    background-color: #fff;
		    position: absolute;
		    top: 50%;
		    left: 50%;
		    transform: translate(-50%, -50%);
		}
		
		.verification-main .verification-title {
		    width: 100%;
		    border-bottom: 1px solid #DEDFE6;
		}
		
		.verification-main .verification-title p {
		    font-size: 16px;
		    color: #1A1A38;
		    margin: 15px 0 15px 20px;
		}
		
		.verification-main .verification-box {
		    width: 392px;
		    margin: 74px auto;
		}
		
		.verification-box .common-box {
		    width: 100%;
		    height: 38px;
		    line-height: 38px;
		    border: 1px solid #DEDFE6;
		}
		
		.verification-box .common-box .common-text {
		    font-size: 14px;
		    height: 38px;
		    line-height: 38px;
		}
		
		.verification-box .phone-box {
		    width: 100%;
		    height: 38px;
		    line-height: 38px;
		    border: 1px solid #DEDFE6;
		}
		
		.verification-box .phone-box span {
		    width: 134px;
		    height: 38px;
		    color: #333;
		    display: block;
		    margin-left: 12px;
		    border-right: 1px solid #DEDFE6;
		    float: left;
		}
		
		.verification-box .phone-box input {
			width:242px;
		    border: none;
		    font-size: 14px;
		    color: #707070;
		    background-color: #fff;
		    padding-left: 12px;
		    height: 35px;
		    line-height: 35px;
		    float:left;
		    
		}
		
		.verification-box .message-box {
		    margin-top: 23px;
		    float: left;
		}
		
		.verification-box .message-box input {
		    width: 255px;
		    display: block;
		    text-indent: 16px;
		    border: none;
		    color: #707070;
		    border-right: 1px solid #DEDFE6;
		    float: left;
		    background-color: #fff;
		    font-size: 14px;
		    height: 20px;
		    margin-top: 10px;
		}
		
		.verification-box .message-box button {
		    width: 136px;
		    color: #4D84FF;
		    border: none;
		    background-color: #fff;
		    float: left;
		    cursor: pointer;
		}
		
		.verification-box .verify-submiit {
		    width: 392px;
		    height: 40px;
		    line-height: 40px;
		    text-align: center;
		    border: none;
		    font-size: 14px;
		    color: #fff;
		    background-color: #4D84FF;
		    margin-top: 40px;
		    cursor: pointer;
		}
		.verification-box .verify-submiit span{
			line-height: 40px;				
			font-size: 20px;
			color: #FFFFFF;
		}
		.verification-box .verify-btn {
			display: none;
			width: 392px;
			height: 40px;
			text-align: center;
			border: none;
			font-size: 14px;
			color: #fff;
			background-color: #4D84FF;
			margin-top: -40px;
		}
		button[disabled] {
		    color: #fff;
		    background: #ccc;
		}
		
		a {
		    text-decoration: none;
		    color: #fff;
		}
		
		#maskShow {
		    width: 278px;
		    height: 286px;
		    position: fixed;
		    top: 50%;
		    left: 50%;
		    margin: -169px 0 0 -143px;
		    background-color: #fff;
		    z-index: 3;
		}
		
		#shadowBox {
		    background-color: #000;
		    position: absolute;
		    top: 0;
		    left: 0;
		    display: none;
		    z-index: 2;
		}
		
		#getVerCodeBtn {
		    z-index: 1;
		    color: #4D84FF;
		}
		
		.time-second {
		    color: #FF1930;
		}
		
		.mask-pop {
		    position: fixed;
		    left: 0;
		    top: 0;
		    height: 100%;
		    width: 100%;
		    background-color: rgba(0, 0, 0, 0.5);
		    opacity: 0;
		    visibility: hidden;
		    -webkit-transition: opacity 0.3s 0s, visibility 0s 1s;
		    -moz-transition: opacity 0.3s 0s, visibility 0s 1s;
		    transition: opacity 0.3s 0s, visibility 0s 1s;
		    z-index: 9999;
		}
		
		.mask-pop.is-visible {
		    opacity: 1;
		    visibility: visible;
		    -webkit-transition: opacity 0.3s 0s, visibility 0s 0s;
		    -moz-transition: opacity 0.3s 0s, visibility 0s 0s;
		    transition: opacity 0.3s 0s, visibility 0s 0s;
		}
		
		/*遮罩滑块完成拼图*/
		.mask-verification {
		    width: 276px;
		    height: 284px;
		    position: absolute;
		    top: 50%;
		    left: 50%;
		    margin-left: -188px;
		    margin-top: -142px;
		    background-color: #fff;
		    box-shadow: 0 1px 8px rgba(128,128,128,0.3);
		    border-radius: 2px;
		    overflow: hidden;
		    -webkit-backface-visibility: hidden;
		    -webkit-transition-property: -webkit-transform;
		    -moz-transition-property: -moz-transform;
		    transition-property: transform;
		    -webkit-transition-duration: 0.5s;
		    -moz-transition-duration: 0.5s;
		    -ms-transition-duration: 0.5s;
		    -o-transition-duration: 0.5s;
		    transition-duration: 0.5s;
		}
		
		.mask-box {
		    width: 260px;
		    height: 268px;
		    padding: 8px 8px;
		}
		
		.mask-warp {
		    position: relative;
		    width: 100%;
		    height: 100%;
		}
		
		.mask-warp .mask-imgBox {
		    position: relative;
		    overflow: hidden;
		    width: 100%;
		    height: auto;
		}
		
		.mask-warp .mask-imgBox .mask-img {
		    position: relative;
		}
		
		.mask-warp .mask-imgBox .mask-img .mask-pic {
		    position: absolute;
		    left: 0;
		    top: 0;
		    z-index: 22;
		}
		
		.mask-warp .mask-imgBox .mask-block {
		    position: absolute;
		    z-index: 111;
		}
		
		.mask-warp .mask-imgBox .mask-block .maskBlock-one {
		    position: absolute;
		    left: 0;
		    top: 0;
		    z-index: 22;
		}
		
		.mask-warp .mask-imgBox .mask-block .maskBlock-two {
		    position: absolute;
		    left: 0;
		    top: 0;
		    z-index: 33;
		}
		
		.mask-tips {
		    position: absolute;
		    left: 0;
		    bottom: -24px;
		    width: 100%;
		    height: 24px;
		    line-height: 24px;
		    background-color: #de715b;
		    font-size: 14px;
		    margin: 0;
		    padding-left: 18px;
		    text-align: left;
		    transition: bottom .3s ease;
		}
		
		.slider-tips {
		    bottom: 0;
		}
		
		.mask-tipOk {
		    bottom: 0;
		    background-color: #5ebf70;
		}
		
		.mask-tips span {
		    display: inline-block;
		    vertical-align: top;
		    line-height: 24px;
		    color: #fff;
		}
		
		.active-tips {
		    display: block
		}
		
		.mask-warp .mask-slider {
		    position: relative;
		    width: 100%;
		    height: 50px;
		    margin-top: 20px;
		    border-bottom: 1px solid #DEDFE6;
		}
		
		.mask-warp .mask-slider .mask-sliderBox {
		    width: 100%;
		    height: 38px;
		    display: inline-block;
		    background: url("${ctx}/images/verification/picBj.png") no-repeat 0 0;
		    cursor: pointer;
		}
		
		.mask-warp .mask-slider .mask-sliderTip {
		    font-size: 14px;
		    color: #88949d;
		    line-height: 36px;
		    margin: 0;
		    padding-left: 30%;
		}
		
		.slider-btn {
		    position: absolute;
		    width: 65px;
		    height: 58px;
		    left: -6px;
		    top: -12px;
		    z-index: 12;
		    cursor: pointer;
		    background: url("${ctx}/images/verification/picBj.png") no-repeat 0 -47px;
		}
		
		.hidden {
		    display: none
		}
		
		.mask-warp .mask-operation {
		    margin-top: 10px;
		}
		
		.mask-warp .mask-operation .close {
		    display: inline-block;
		    width: 20px;
		    height: 20px;
		    margin: 0 14px;
		    background: url("${ctx}/images/verification/picBj.png") no-repeat 0 -188px;
		    cursor: pointer;
		    float: left;
		    -webkit-backface-visibility: hidden;
		    -webkit-transition-property: -webkit-transform;
		    -moz-transition-property: -moz-transform;
		    transition-property: transform;
		    -webkit-transition-duration: 0.3s;
		    -moz-transition-duration: 0.3s;
		    -ms-transition-duration: 0.3s;
		    -o-transition-duration: 0.3s;
		    transition-duration: 0.3s;
		}
		
		.mask-warp .mask-operation .refresh {
		    display: inline-block;
		    width: 20px;
		    height: 20px;
		    background: url("${ctx}/images/verification/picBj.png") no-repeat 0  -341px;
		    cursor: pointer;
		    float: left;
		}
		
		.mask-warp .mask-operation .close:hover {
		    background-position: 0 -214px
		}
		
		.mask-warp .mask-operation .refresh:hover {
		    background-position: 0 -367px
		}
		.verification-box .verify-code {
			width:100%;
			height:auto;
		}
		.verification-box .verify-error {
			color: #ff6060;
		    font-size: 12px;
		    margin-top: 6px;
		    display: inline-block;
		    margin: 6 10px 0 0;
		}
		
   </style> 
</head>
<body>
	<div class="verification-warp">		
	    <div class="verification-header">
			<div class="verfication-logo">
				<div class="el-icon-logo-baiomc el-icon logo-tip"></div>
				<span class="verification-version">${omc_ver}</span>
			</div>
	        <a class="verify-back" href="javascript:void(0);" onclick="javascript:logout('${ctx}')" ><%=rb.getString("FanHuiDengLuYe")%><i class="el-icon-right el-icon"></i> </a>
	    </div>
	    <div class="verification-main">
	        <div class="verification-title">
	            <p><%=rb.getString("DuanXinYanZheng")%></p>
	        </div>
	        
	        <div class="verification-box">
	             <form role="form" action="" id="verifyForm" method="post" >
		            <input id='verifyName' type="hidden" name="userInfo.usercode" value="${userInfo.usercode}" />
		            <input id="verifyPassword" type="hidden" name="userInfo.password" value="${userInfo.password}" /> 

		            <input id="logintype" type="hidden" name="userInfo.logintype" value=""/>
					<input id="loginCaptcha" type="hidden" name="userInfo.loginCaptcha" value=""/>
					<input id="publicKey" type="hidden" name="userInfo.publicKey" value=""/>
					<input id="captchaId" type="hidden" name="userInfo.captchaId" value=""/>

	            	<div class="verify-code">
		            	<div class="phone-box">
		                    <span class="common-text"><%=rb.getString("ZhongGuo")%> +86</span>
		                    <input id="phone" type="text" class="verify-input" name="userInfo.phoneNum" value="${userInfo.phoneNum}"
		                    	required oninvalid="setCustomValidity('<%=rb.getString("QingShuRuShouJiHao")%>')" oninput="setCustomValidity('')" 
		                    	placeholder="<%=rb.getString("QingShuRuShouJiHao")%>" maxlength="14">	                  
		                </div>
		                <p class="verify-error" id="verifyPhoneError"></p>
	            	</div>
	                
	                <div class="verify-code">
	                	<div class="message-box common-box">
	                    	<input id="verificationCode" class="verify-input" type="text" name="userInfo.verifycode" value="${userInfo.verifycode}"
	                    		required oninvalid="setCustomValidity('<%=rb.getString("QingShuRuYanZhengma")%>')" oninput="setCustomValidity('')"
	                    		placeholder="<%=rb.getString("DuanXinYanZhengMa")%>" maxlength="64">
	                    	<button id="getVerCodeBtn" type="button" class="common-text"><%=rb.getString("HuoQuDuanXinYanZhengMa")%></button>
	                	</div>
	                	<p class="verify-error" id="verifyCodeError">${userInfo.errinfo}</p>
	                </div>
	                 
	                <div class="verify-submiit" onclick="submit_verify()">
						<span><%=rb.getString("LiJiYanZheng")%></span>
					</div>
					
					<input id="verifyBtn" class="verify-btn" type="submit" value="<%=rb.getString("LiJiYanZheng")%>"/> 			                
	            </form> 
	            
	        </div>
	    </div>
	</div>

	<!--滑块验证-->
	<div class="mask-pop">
	    <div class="mask-verification" id="maskShow">
	        <div id="maskBox" class="mask-box"></div>
	    </div>
	</div>
	
	<script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery.min.js"></script>
	<script type="text/javascript">
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
		
		//返回登录
		function logout(ctx) {
			var params = {};
			$.ajax({
				url: ctx+'/login/logout.htm',
			    type: 'POST',
			    dataType: 'text',
			    data :params,
			    ontentType: "application/x-www-form-urlencoded; charset=utf-8",
			    error: function(request){
				    window.location.href = ctx+"/";
			    },
			    success: function(request){
			    }
			});			
		}
		
        //获取短信验证码点击事件
        $('#getVerCodeBtn').on('click', function(event) {
            var $this = $(this);
            $this.attr('disabled','disabled');
            if (!checkPhone()) {
                $this.removeAttr('disabled');
                return false;
            }else{
                event.preventDefault();
                $('.mask-pop').addClass('is-visible');
            }
            $("#verifyCodeError").html("");
        });
         
        if(window.sessionStorage){
        	var userName = window.sessionStorage.getItem('userName');
        	var userPassword = window.sessionStorage.getItem('userPassword');
			//new params
			var logintype = window.sessionStorage.getItem('logintype');
			var loginCaptcha = window.sessionStorage.getItem('loginCaptcha');
			//var publicKey = window.sessionStorage.getItem('publicKey'); // 通过接口来获取动态的publickey
			var captchaId = window.sessionStorage.getItem('captchaId');
        }
       
        //滑块拼图验证
        maskBox({
            el:'#maskBox',
            width:'260',
            height:'160',
            img:[ 
            	'${ctx}/images/verification/one.jpg',
                '${ctx}/images/verification/two.jpg',
                '${ctx}/images/verification/three.jpg',
                '${ctx}/images/verification/four.jpg'
            ],
            success:function () {           
                var phone = $.trim($('#phone').val());               
                var paramsMsg = {
                		userCode:userName,
                		password:AesEncrypt(userPassword),
    	                phoneNum:phone    	                
    	        };
                
                $.post("${ctx}/msg/sendMsg.action",paramsMsg,function(data){               	
                	 if (data.success) {
                		clearInterval(timer);
	                    getVerCode('#getVerCodeBtn','<%=rb.getString("HuoQuDuanXinYanZhengMa")%>');
	                    //$('#getVerCodeBtn').attr('disabled','disabled');	                    
	                } else {
	                	
	                	$('#getVerCodeBtn').removeAttr('disabled','disabled');
	                	$("#verifyCodeError").html(data.message);		               													
	                }  
                    
                },'json'); 
            },
            error:function () {

            }
        });
       	    
	    /*立即验证*/
	    function submit_verify() {       	
			var bool = checkForm(document.querySelector('#verifyForm'));
			if(!bool) return false;

			$.ajax({
				type: "post",
				url: '${ctx}/sys/login/getPublicKey',
				async: false,
				dataType:"text",
				success: function(data) {
					var raskey = data;
			
					$("#verifyName").val(RsaEncrypt(userName, raskey));
					$("#verifyPassword").val(RsaEncrypt(userPassword, raskey));
					$("#publicKey").val(raskey);
				}
			});
			

			//$('#verifyName').val(RsaEncrypt(userName));
			//$('#verifyPassword').val(RsaEncrypt(userPassword)); 
			//new params
			$('#logintype').val(logintype);
			$('#loginCaptcha').val(loginCaptcha);
			//$('#publicKey').val(publicKey);
			$('#captchaId').val(captchaId);
			

            var verificationCode = $.trim($('#verificationCode').val());
            
            if (!checkPhone()) {
                return false;
            }
            if (!verificationCode) {
                $("#verifyCodeError").html('<%=rb.getString("QingShuRuYanZhengma")%>');
                return false;
            }

			//start new	
			//点击立即验证按钮，先验证输入的短信验证码是否正确
			// 参数为  verifyForm 中的所有参数
			var params = {
				"userInfo.logintype": logintype, //new param
				"userInfo.usercode": $.trim($("#verifyName").val()),
				"userInfo.password": $.trim($("#verifyPassword").val()),
				//new params start
				"userInfo.loginCaptcha": loginCaptcha,
				"userInfo.publicKey": $("#publicKey").val(),
				"userInfo.captchaId": captchaId,
				//new params end
				"userInfo.phoneNum": $.trim($('#phone').val()),
				"userInfo.verifycode": $.trim($('#verificationCode').val())
			};
	
			//resulr: 1 验证成功 0 验证失败
			$.post("${ctx}/system/sysuser/validateVerifyCode.action",params,function(data){
				if(data.result == '1'){
					$('#verifyForm').addClass('readonly');
					$("#verifyCodeError").html("");
					$('#verifyForm').attr('action',"${ctx }/sys/login/userLoad.htm");
					$("#verifyBtn").click();				
				}else{
					$("#verifyCodeError").html(data.info);
				}
			},'json');
		} 
	    
	 	// 校验手机号
        function checkPhone () {
            var validatePhoneone = /^1\d{10}$/;
            var $phone = $("#phone").val().trim();
            if(!$phone){               
                $("#verifyPhoneError").html('<%=rb.getString("QingShuRuShouJiHao")%>');
                return false;
            }else{
                if (validatePhoneone.test($phone)) {
                    return true;
                } else {            
                    $("#verifyPhoneError").html('<%=rb.getString("ShouJiHaoMaGeShiBuZhengQue")%>');
                    return false;
                }
            }
        }
	 
	    /*倒计时  */
        var timer;
        function getVerCode(id,desc) {
            var _this = $(id);
            _this.attr('disabled','disabled');
            var i = 60;
            _this.html('<span class="time-second">'+ i +'s </span>' + ' <%=rb.getString("HouChongXinFaSong")%>');
            timer = setInterval(function(){
                i = i-1;
                _this.html('<span class="time-second">'+ i +'s </span>' + ' <%=rb.getString("HouChongXinFaSong")%>');
                if(i == 0){
                    clearInterval(timer);
                    _this.html(desc);
                    _this.removeAttr('disabled','disabled');
                }
            },1000);
        }
		    
	    
      	//关闭遮罩滑块验证
        $('.mask-pop').on('click', function(event){
            if( $(event.target).is('#closeMask') || $(event.target).is('.mask-pop') ) {
                event.preventDefault();
                $(this).removeClass('is-visible');
                $('#getVerCodeBtn').removeAttr('disabled');
            }
        });

        //ESC关闭遮罩滑块验证
        $(document).keyup(function(event){
            if(event.which=='27'){
                $('.mask-pop').removeClass('is-visible');
            }
        });
	          
              
        $(function(){	

	        //手机号、短信验证码，回车事件-提交表单
			$(".verify-input").bind("keyup", function(e) {
				if (e.keyCode == 13) {
					$('#verifyForm input').blur();
					if(!isAutoSubmit()) submitform();
				}
			});
			$("#phone").keyup(function(e){
				if (e.keyCode != 13) {
					if($(this).val().length>0){
						$("#verifyPhoneError").html("");
					}
				}
			})
			
			$("#verificationCode").keyup(function(e){
				if (e.keyCode != 13) {
					if($(this).val().length>0){
						$("#verifyCodeError").html("");
					}
				}
			})	
			 
	    });
        
        /*滑块验证  */
        function maskBox(Config) {
		    var el = $(Config.el);
		    var w = Config.width;
		    var h = Config.height;
		    var imgLibrary = Config.img;
		    var PL_Size = 48;
		    var padding = 20;
		    var MinN_X = padding + PL_Size;
		    var MaxN_X = w - padding - PL_Size - PL_Size / 6;
		    var MaxN_Y = padding;
		    var MinN_Y = h - padding - PL_Size - PL_Size / 6;

		    function RandomNum(Min, Max) {
		        var Range = Max - Min;
		        var Rand = Math.random();
		        if (Math.round(Rand * Range) == 0) {
		            return Min + 1;
		        } else if (Math.round(Rand * Max) == Max) {
		            return Max - 1;
		        } else {
		            var num = Min + Math.round(Rand * Range) - 1;
		            return num;
		        }
		    }
		    var imgRandom = RandomNum(1, imgLibrary.length);
		    var imgSrc = imgLibrary[imgRandom];
		    var X = RandomNum(MinN_X, MaxN_X);
		    var Y = RandomNum(MinN_Y, MaxN_Y);
		    var left_Num = -X + 10;
		    var nodeList=[];
		    nodeList.push(
		        '<div class="mask-warp">',
		            '<div class="mask-imgBox">',
		                '<div class="mask-img" style="width:' + w + 'px;height:' + h + 'px;">',
		                    '<img id="scream" src="' + imgSrc + '" style="width:' + w + 'px;height:' + h + 'px;">',
		                    '<canvas id="maskPic" class="mask-pic" width="' + w + '" height="' + h + '"></canvas>',
		                '</div>',
		                '<div class="mask-block" style="width:' + w + 'px;height:' + h + 'px;top:0;left:' + left_Num + 'px;">',
		                    '<canvas id="maskShadow" class="maskBlock-one" width="' + w + '" height="' + h + '"></canvas>',
		                    '<canvas id="maskLost" class="maskBlock-two" width="' + w + '" height="' + h + '"></canvas>',
		                '</div>',
		                '<p class="mask-tips"></p>',
		            '</div>',

		            '<div class="mask-slider">',
		                '<div class="mask-sliderBox">',
		                    '<p id="dragSlider" class="mask-sliderTip"><%=rb.getString("TuoDongHuaKuaiWanChengPinTu")%></p>',
		                '</div>',
		                '<div class="slider-btn"></div>',
		            '</div>',
		            '<div class="mask-operation">',
		                '<a class="close" id="closeMask"></a>',
		                '<a class="refresh" id="refreshBtn"></a>',
		            '</div>',
		        '</div>'
		    );
		    el.html(nodeList.join(''));

		    var d = PL_Size / 3;
		    var c = document.getElementById("maskPic");
		    var ctx = c.getContext("2d");
		    ctx.globalCompositeOperation = "xor";
		    ctx.shadowBlur = 10;
		    ctx.shadowColor = "#fff";
		    ctx.shadowOffsetX = 3;
		    ctx.shadowOffsetY = 3;
		    ctx.fillStyle = "rgba(0,0,0,0.7)";
		    ctx.beginPath();
		    ctx.lineWidth = "1";
		    ctx.strokeStyle = "rgba(0,0,0,0)";
		    ctx.moveTo(X, Y);
		    ctx.lineTo(X + d, Y);
		    ctx.bezierCurveTo(X + d, Y - d, X + 2 * d, Y - d, X + 2 * d, Y);
		    ctx.lineTo(X + 3 * d, Y);
		    ctx.lineTo(X + 3 * d, Y + d);
		    ctx.bezierCurveTo(X + 2 * d, Y + d, X + 2 * d, Y + 2 * d, X + 3 * d, Y + 2 * d);
		    ctx.lineTo(X + 3 * d, Y + 3 * d);
		    ctx.lineTo(X, Y + 3 * d);
		    ctx.closePath();
		    ctx.stroke();
		    ctx.fill();
		    var c_l = document.getElementById("maskLost");
		    var c_s = document.getElementById("maskShadow");
		    var ctx_l = c_l.getContext("2d");
		    var ctx_s = c_s.getContext("2d");
		    var img = new Image();
		    img.src = imgSrc;
		    img.onload = function() {
		        ctx_l.drawImage(img, 0, 0, w, h);
		    };
		    ctx_l.beginPath();
		    ctx_l.strokeStyle = "rgba(0,0,0,0)";
		    ctx_l.moveTo(X, Y);
		    ctx_l.lineTo(X + d, Y);
		    ctx_l.bezierCurveTo(X + d, Y - d, X + 2 * d, Y - d, X + 2 * d, Y);
		    ctx_l.lineTo(X + 3 * d, Y);
		    ctx_l.lineTo(X + 3 * d, Y + d);
		    ctx_l.bezierCurveTo(X + 2 * d, Y + d, X + 2 * d, Y + 2 * d, X + 3 * d, Y + 2 * d);
		    ctx_l.lineTo(X + 3 * d, Y + 3 * d);
		    ctx_l.lineTo(X, Y + 3 * d);
		    ctx_l.closePath();
		    ctx_l.stroke();
		    ctx_l.shadowBlur = 10;
		    ctx_l.shadowColor = "black";
		    ctx_l.clip();
		    ctx_s.beginPath();
		    ctx_s.lineWidth = "1";
		    ctx_s.strokeStyle = "rgba(0,0,0,0)";
		    ctx_s.moveTo(X, Y);
		    ctx_s.lineTo(X + d, Y);
		    ctx_s.bezierCurveTo(X + d, Y - d, X + 2 * d, Y - d, X + 2 * d, Y);
		    ctx_s.lineTo(X + 3 * d, Y);
		    ctx_s.lineTo(X + 3 * d, Y + d);
		    ctx_s.bezierCurveTo(X + 2 * d, Y + d, X + 2 * d, Y + 2 * d, X + 3 * d, Y + 2 * d);
		    ctx_s.lineTo(X + 3 * d, Y + 3 * d);
		    ctx_s.lineTo(X, Y + 3 * d);
		    ctx_s.closePath();
		    ctx_s.stroke();
		    ctx_s.shadowBlur = 20;
		    ctx_s.shadowColor = "black";
		    ctx_s.fill();
		    var moveStart = '';
		    $(".slider-btn").mousedown(function(e) {
		        e = e || window.event;
		        $(this).css({
		            "background-position": "0 -118px"
		        });
		        $('#dragSlider').css({
		            "opacity": '0'
		        });
		        moveStart = e.pageX;
		    });
		    onmousemove = function(e) {
		        e = e || window.event;
		        var moveX = e.pageX;
		        var d = moveX - moveStart;
		        if (moveStart == '') {} else {
		            if (d < 0 || d > (w - padding - PL_Size)) {} else {
		                $(".slider-btn").css({
		                    "left": d + 'px',
		                    "transition": "inherit"
		                });
		                $("#maskLost").css({
		                    "left": d + 'px',
		                    "transition": "inherit"
		                });
		                $("#maskShadow").css({
		                    "left": d + 'px',
		                    "transition": "inherit"
		                });
		            }
		        }
		    };
		    onmouseup = function(e) {
		        e = e || window.event;
		        var moveEnd_X = e.pageX - moveStart;
		        var ver_Num = X - 10;
		        var deviation = 4;
		        var Min_left = ver_Num - deviation;
		        var Max_left = ver_Num + deviation;
		        if (moveStart == '') {} else {
		            if (Max_left > moveEnd_X && moveEnd_X > Min_left) {
		                /*验证通过提示*/
		                $(".mask-tips").html('<span><%=rb.getString("YanZhengChengGong")%></span>');		                
		                $(".mask-tips").addClass("mask-tipOk");
		                $(".mask-block").addClass("hidden");
		                $("#maskPic").addClass("hidden");

		                setTimeout(function() {
		                    $(".mask-tips").removeClass("mask-tipOk");
		                    maskBox(Config);
		                    $('.mask-pop').removeClass('is-visible');
		                                                        
		                }, 1000);
		                Config.success();
		                              
		            } else {
		                $(".mask-tips").html('<span><%=rb.getString("QingZhengQuePinHeTuXiang")%></span>');		            	
		                $(".mask-tips").addClass("slider-tips");
		                setTimeout(function() {
		                    $(".mask-tips").removeClass("slider-tips");
		                }, 1000);
		                Config.error();
		            }
		        }
		        setTimeout(function() {
		            $(".slider-btn").css({
		                "left": '-6px',
		                "transition": "left 0.5s"
		            });
		            $("#maskLost").css({
		                "left": '0',
		                "transition": "left 0.5s"
		            });
		            $("#maskShadow").css({
		                "left": '0',
		                "transition": "left 0.5s"
		            });
		            $('#dragSlider').css({
		                "opacity": '1'
		            });

		        }, 1000);
		        $(".slider-btn").css({
		            "background-position": "0 -47px"
		        });
		        moveStart = '';
		        $("#refreshBtn").on("click", function() {
		            maskBox(Config);
		        });

		    }
		} 
	</script>
</body>
</html>