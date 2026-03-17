<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 窗口  -- 修改密码 --%>
<div id="modifyPass" class="easyui-layout" data-options="fit:true,border: false">
	<div region="center" data-options="border: false" style="padding: 20px;">
		<div class="itemDiv" style="height: 50px;">
			<span style="width: 140px;"><%=rb.getString("YuanMiMa")%></span>
			<input id="opassword" autocomplete="off" name="opassword" type="password" onblur="validateRequired(event)" oninput="hideErrorTips()" value="" class="border border-box item" style="width: 210px" />
			<i id="opassword_air" class="el-icon el-icon-operation-hide" onclick="oldPwdClick()" style="margin-left: -25px;z-index: 1;"></i>
		</div>
		<div class="itemDiv" style="height: 50px;">
			<span style="width: 140px;"><%=rb.getString("XinMiMa")%></span>
			<input id="npassword" name="npassword" onblur="validateRequired(event)" oninput="hideErrorTips()" class="border border-box item" style="width: 210px" type="password" />
			<i id="npassword_air" class="el-icon el-icon-operation-hide" onclick="newPwdClick()" style="margin-left: -25px;z-index: 1;"></i>
		</div>
		<div class="itemDiv" style="height: 50px;">
			<span style="width: 140px;"><%=rb.getString("QueRenXinMiMa")%></span>
			<input id="renpassword" name="renpassword" type="password" onblur="validateRequired(event)" oninput="hideErrorTips()" class="border border-box item" style="width: 210px" />
			<i id="renpassword_air" class="el-icon el-icon-operation-hide" onclick="cfmPwdClick()" style="margin-left: -25px;z-index: 1;"></i>
		</div>
		<div id="password_err_text" class="redColor" style="margin-bottom:10px;">

		</div>
		<div class="itemDiv" style="width: 398px;height: 30px;background-color: #F2F2F2;">
			<span style="line-height: 30px;width: 140px;"><%=rb.getString("MiMaChangDu")%><%=rb.getString("MaoHao")%></span>
	       	<span style="color:#AEAEAE" id="pwd_length_tip">6-20</span>
	       	<!--<span style="color:#AEAEAE"><%=rb.getString("ZiFuFuShu")%></span>-->
	       	<span style="color:#AEAEAE"><%=rb.getString("DouHao")%></span>
	        
	        <span style="color:#AEAEAE"><%=rb.getString("ZhangHuYouXiaoTianShu")%><%=rb.getString("MaoHao")%></span>
	        <span style="color:#AEAEAE">${pwdLength}</span>
	        <span style="color:#AEAEAE"><%=rb.getString("TianFuShu")%></span>
	        <span style="color:#AEAEAE"><%=rb.getString("JuHao")%></span>
        </div>
	</div>
	<div region="south" data-options="border:false,height:61" style="padding:10px 20px;border-width:0;">
		<div class="windowButtonGroup">
			<a class="linkbutton linkbutton_trend" onclick="savePwd()"><span><%=rb.getString("QueDing")%></span></a>
			<a class="linkbutton linkbutton_nowanna" onclick="closeDefaultWindow();"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
	</div>
</div>

<script type="text/javascript">
function getPwdLength() {
	var tips = $('#pwd_length_tip');

	$.post('${ctx}/system/sysuser/getPwdLength.action', {}, function(data) {
		var minLength = data.minLength,
			maxLength = data.maxLength;
		
		tips.html(minLength + '-' + maxLength);
	});
}

getPwdLength();

function hideErrorTips() {
	$('#password_err_text').text('');
}

function oldPwdClick() {
	var icon = $('#opassword_air'),
		input = $('#opassword'),
		isClose = icon.hasClass('el-icon-operation-hide');

	if(isClose) {
		icon.removeClass('el-icon-operation-hide');
		icon.addClass('el-icon-operation-view');
		input.attr('type','text');
	}else {
		icon.removeClass('el-icon-operation-view');
		icon.addClass('el-icon-operation-hide');
		input.attr('type','password');
	}
}
function newPwdClick() {
	var icon = $('#npassword_air'),
		input = $('#npassword'),
		isClose = icon.hasClass('el-icon-operation-hide');
	
	if(isClose) {
		icon.removeClass('el-icon-operation-hide');
		icon.addClass('el-icon-operation-view');
		input.attr('type','text');
	}else {
		icon.removeClass('el-icon-operation-view');
		icon.addClass('el-icon-operation-hide');
		input.attr('type','password');
	}
}
function cfmPwdClick() {
	var icon = $('#renpassword_air'),
		input = $('#renpassword'),
		isClose = icon.hasClass('el-icon-operation-hide');

	if(isClose) {
		icon.removeClass('el-icon-operation-hide');
		icon.addClass('el-icon-operation-view');
		input.attr('type','text');
	}else {
		icon.removeClass('el-icon-operation-view');
		icon.addClass('el-icon-operation-hide');
		input.attr('type','password');
	}
}

// 保存新密码
function savePwd(){
	var opassword = $("#opassword").val();
	var npassword = $("#npassword").val();
	$("#password_err_text").html('')
	
	if ($("#modifyPass .err_border").length != 0) {
		return;
	}
	//新密码不能和旧密码相同
	if (opassword == npassword) {
		$("#password_err_text").html('<%=rb.getString("XinJiuMiMaBuNengXiangTong")%>')
		return;
	}
	
	var renpassword = $("#renpassword").val();
	//两次输入的新密码不一致
	if (renpassword != npassword) {
		$("#password_err_text").html('<%=rb.getString("LiangMiMaBuYiZhi")%>')
		return;
	}

	$.post('${ctx}/system/sysuser/getPwdLength.action', {}, function(data) {
		var minLength = data.minLength,
			maxLength = data.maxLength,
			reg = /^[a-zA-Z0-9_!@#$%^&*?]+$/;
		
		var msg = "<%=rb.getString("MiMaChangDu")%> " + minLength + "-" + maxLength+ " <%=rb.getString("ZiFuFuShu")%>" 
					+ ", <%=rb.getString("FanWei")%>: [a-zA-Z0-9]、<%=rb.getString("TeSHuFuHao")%>(_!@#$%^&*?)";
		
		if (npassword.length < parseInt(minLength) || npassword.length > parseInt(maxLength) || !reg.test(npassword)) {
			$("#password_err_text").html(msg)
		}else {

			//密码规则
			$.post("${ctx}/system/sysuser/queryPasswordRule.action", {}, function (data) {
				var passwordContent = data.password_content;
				/*
				// 经与后端协商，密码长度均统一为6-20位（10.2.2版本）
				var minLength = 6;
				var maxLength = 20;
				//密码长度
				if (!(renpassword.length >= minLength && renpassword.length <= maxLength)) {
					$("#password_err_text").html('<%=rb.getString("MiMaChangDu")%> ' + minLength + "-" + maxLength+ "<%=rb.getString("ZiFuFuShu")%>")
					return;
				}
				*/
				//密码内容 passwordContent -- 2:弱密码，只校验长度 ；0：强密码，数字、大写字母、小写字母、特殊字符（_!@#$%^&*?）
				if (passwordContent == "0") {
					if (checkPasswdStrength(renpassword) == "1") {
						$("#password_err_text").html('<%=rb.getString("MiMaBiXuLiangZhongLeiXing")%>')
						return;
					}
				}else {
					$("#password_err_text").html('')
				}
				$.messager.confirm('<%=rb.getString("QueRen")%>', '<%=rb.getString("QueRenXiuGaiMiMa")%>', function (r) {
					if (r) {
						var params = {
							"opassword": RsaEncrypt(opassword, '${publicKey}'),
							"npassword": RsaEncrypt(npassword, '${publicKey}')
						};
						$.post("${ctx}/system/sysuser/modifyPassword.action", params, function (data) {
							if (data["success"]) {
								logout('${ctx}');
							} else{
								$("#password_err_text").html(data["message"])
								//showMsg('error_msg',data["message"]);
							}
						}, "json");
					}
				}).addClass('normalConfirm');
			}, "json"); 
		}
	});
	
	
}
	
</script>