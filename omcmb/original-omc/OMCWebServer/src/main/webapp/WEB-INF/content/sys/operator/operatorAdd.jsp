<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 添加运营商 --%>
<style type="text/css">
#addOperatorForm{
	margin-left:40px;
}
#addOperatorForm label{
	display:block;
	color:#85A8BF;
}
#addOperatorForm input{
	width:400px;
	height:26px;
	border:1px solid #85A8BF;
	margin-top:10px;
}
#addOperatorForm p{
	color:#CC0000;
}
#addOperatorForm div{
	margin-top:15px;
}
#addOperatorForm p{
	margin-top:5px;
	width:100%;
	height:16px;
}
#addOperatorsuccess{
	background:#E9FBFF url("${ctx}/images/success.png") no-repeat 8px center;
	height:38px;
	float:left;
	margin-left:10px;
	margin-top:25px;
	color:#508D9B;
	font-size:15px;
	font-weight:bold;
	padding:0 5px 0 10px;
	line-height:38px;
	text-indent:25px;
	display:none;
}
</style>
<div style='height:50px;width:100%;margin-top:10px;'>
	<div style="display:inline-block;margin-left:30px;line-height:50px;color:#7993B6;font-size:16px;"><%=rb.getString("TianJiaYunYingShang")%></div>
	<a class="titleIcon_close iconSize" style="float:right;margin-top:15px;margin-right:10px;" onclick="cancelAddOperator()"></a>
</div>
<form id="addOperatorForm">
	<div>
		<label><%=rb.getString("YunYingShangMingCheng")%><%=rb.getString("MaoHao")%></label>
		<input id='operatorCode' type="text" name="operatorCode" onBlur="fillDefaultAdmin(this);" />
		<span class='operationDiv operation_getFocus'></span>
		<p></p>
	</div>
	<div>
		<label><%=rb.getString("CLOUDKEY")%><%=rb.getString("MaoHao")%></label>
		<input id='cloudKey'  type="text" name="cloudKey" onfocus="fillDefaultCloudKey(this);"  onBlur ="checkCloudKey(this)" />
		<span class='operationDiv operation_getFocus'></span>
		<p></p>
	</div>
	<div>
		<label><%=rb.getString("MoRenGuanLiYuan")%><%=rb.getString("MaoHao")%></label>
		<input id='adminUserCode' type="text" name="adminUserCode" onfocus="checkAdminUserCode()"/>
		<span class='operationDiv operation_getFocus'></span>
		<p></p>
	</div>
</form>
<div style='float:left;margin:25px 0px 0px 40px'>
  	<span class="el-button el-button--primary" onclick="addOperator()"><%=rb.getString("QueDing")%></span>
  	<span class="el-button" onclick="cancelAddOperator()"><%=rb.getString("QuXiao")%></span>
</div>
<div id='addOperatorsuccess'><%=rb.getString("QuanXianTianJiaYunYingShangChengGong")%></div>

<script type="text/javascript">
//自定义cloud_key正则表达式验证
<%-- $.extend($.fn.validatebox.defaults.rules,{
	cloudkey:{
	
		validator:function(value){
			return /^[0-9A-Z]{6}$/.test(value);
		},
		message:"<%=rb.getString("CLOUDKEYYOUXIAOXING")%>"+"6"
	},
	regOper:{
		validator:function (value){
			var reg = /^[a-zA-Z0-9\_\-\@\.]+$/;
			return reg.test(value);
		},
		message:'<%=rb.getString("YunYingShangChengBuHeFa")%>'
	}
}) --%>
// 填充默认管理员
function fillDefaultAdmin(ele) {
	var reg = /^[a-zA-Z0-9\_\-\@\.]+$/;
	if($(ele).val() == ""){
		$(ele).siblings('p').html('<%=rb.getString("QingShuRuYunYingShangMingCheng")%>');
		var opCode = $("#addOperatorForm input[name='operatorCode']").val();
		$("#addOperatorForm input[name='adminUserCode']").val(opCode + "Admin");
		checkAdminUserCode();
	}else{
		if(!reg.test($(ele).val())){
			$(ele).siblings('p').html('<%=rb.getString("YunYingShangChengBuHeFa")%>');
		}else{
			$(ele).siblings('p').html('');
			var opCode = $("#addOperatorForm input[name='operatorCode']").val();
			$("#addOperatorForm input[name='adminUserCode']").val(opCode + "Admin");
			checkAdminUserCode();
		}
	}
}
//随机生成CLOUDKEY
function fillDefaultCloudKey(ele){
	var cloudKey = $("#addOperatorForm input[name='cloudKey']").val();
	if(cloudKey == ""){
		cloudKey = Math.random().toString(30).substring(5).slice(0,6);
		$("#addOperatorForm input[name='cloudKey']").val(cloudKey.toUpperCase());
	}
}
function checkCloudKey(ele){
	var reg = /^[0-9A-Z]{6}$/;
	if(!reg.test($('#cloudKey').val())){
		$(ele).siblings('p').html("<%=rb.getString("CLOUDKEYYOUXIAOXING")%>"+"6");
	}else{
		$(ele).siblings('p').html('');
	}
}
function checkAdminUserCode(){
	if($('#adminUserCode').val() != ""){
		$('#adminUserCode').siblings('p').html('');
	}
}
// 保存
function addOperator() {
	var reg = /^[a-zA-Z0-9\_\-\@\.]+$/;
	var regCloud = /^[0-9A-Z]{6}$/;
	var isPass = true;
	if($('#operatorCode').val() == ""){
		$('#operatorCode').siblings('p').html('<%=rb.getString("QingShuRuYunYingShangMingCheng")%>');
		isPass = false;
	}else{
		if(!reg.test($('#operatorCode').val())){
			$('#operatorCode').siblings('p').html('<%=rb.getString("YunYingShangChengBuHeFa")%>');
			isPass = false;
		}	
	}
	if($('#cloudKey').val() == ""){
		$('#cloudKey').siblings('p').html('<%=rb.getString("QingShuRuCloudKey")%>');
		isPass = false;
	}else{
		if(!regCloud.test($('#cloudKey').val())){
			$('#cloudKey').siblings('p').html("<%=rb.getString("CLOUDKEYYOUXIAOXING")%>"+"6");
			isPass = false;
		}
	}
	if($('#adminUserCode').val() == ""){
		$('#adminUserCode').siblings('p').html('<%=rb.getString("TianXieMoRenGuanLiYuan")%>');
		isPass = false;
	}
	// 验证必填项
	 <%-- if(!$("#addOperatorForm").form("validate")){
		$.messager.alert(TiShi, "<%=rb.getString("ShuJuYanZhengBuTongGuo")%>");
		return;
	} --%>
	if(isPass){
		var operatorCode = $("#addOperatorForm input[name='operatorCode']").val();
		var adminUserCode = $("#addOperatorForm input[name='adminUserCode']").val();
		var cloudKey = $("#addOperatorForm input[name='cloudKey']").val();
		var params = {
			 operatorCode : operatorCode,
			 operatorName : "",
			 adminUserCode : adminUserCode,
			 cloudKey : cloudKey,
		};
		$.post("${ctx}/system/operator/addOperator.action", params, function(data) {
			if (data["success"]) {
			
				
				$('#addOperatorsuccess').fadeIn(300,function(){
					var  time = setTimeout(function(){
					$('#operAddOperatorDiv').animate({height:'0px'},400,function(){
						$('#operAddOperatorDiv').css('border','none');
						$("#tableOperatorList").datagrid("reload");
					});
					},1000);
				})
			} else {
				showMsg('error_msg',data["message"]);
			}
		}, "json");
	}
}
function cancelAddOperator(){
	$('#operAddOperatorDiv').animate({height:'0px'},400,function(){
		$('#operAddOperatorDiv').css('border','none');
	});
	
}
</script>