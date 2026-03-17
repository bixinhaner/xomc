<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
.basicInfo{
	display:inline-block;
	margin-bottom:40px;
	height:85px;
	float:left;
}
.basicInfo label{
	display:block;
	margin-bottom:8px;
	color:#4C6778;
}
.basicInfo input{
	width:350px;
}
.basicInfo p.errorMes{
	display:none;
	color:#CC0000;
	margin-top:5px;
	width:350px;
}
.rightSlide{
	margin-left:70px;
}
.errorborder{
	border:1px solid #CC0000;
}
.cpeProSuccess{
	background:#E9FBFF url("${ctx}/images/success.png") no-repeat 16px center;
	height:38px;
	float:left;
	margin-left:20px;
	margin-top:20px;
	color:#508D9B;
	font-size:15px;
	font-weight:bold;
	padding:0 20px 0 20px;
	line-height:38px;
	text-indent:25px;
	display:none;
}
.forbidden > span{
    background: #B0CBDD !important;
}
.el-card__body{
	display:flex;
	flex-direction:column;
	flex:1 1 auto;
	height:100%;
	overflow:auto;
}
</style>
<div style='height:100%;display:flex;flex-direction:column;'>
	<div class='el-card__header'>
		<span id='cpeProductHead'></span>
		<span class="el-icon el-icon-close slideIcon" onclick='cancelOperCpeProType()'></span>
	</div>
	<div class='el-card__body'>
		<div style='padding:10px;background:#fff;flex:1 1 auto;display:flex;flex-direction:column'>
			<div id='parameterContent' style='height:85px;'>
				<div class='basicInfo' style='margin-bottom:0px;'>
					<label><%=rb.getString("ChanPinLeiXingBiaoZhi") %></label>
					<input onblur='checkProductName()' id='proNameInput' maxLength="200" class='border border-box' />
					<p class='errorMes productErr'>Product tyep can not be empty!</p>
				</div>
				<div class='basicInfo rightSlide' style='margin-bottom:0px;'>
					<label><%=rb.getString("CanShuMoXing") %></label>
					<select style='width:350px;height:26px;' id='modelInput' class='border border-box' data-options="editable:false"></select>
					<p class='errorMes'>Parameter Model can not be empty</p>
				</div>
			</div>
			<div style='flex:1 1 auto;margin-right:30px;'>
				<table id='cpeProTypeTableList'></table>
			</div>
		</div>
	</div>
	<c:if test="${type == 'add'}">
		<div class='el-card__footer'>
			<div class='windowButtonGroup' style='float:left;'>
				<a class='linkbutton linkbutton_trend cpeProButton' onclick='saveCpeProType()'><span><%=rb.getString("QueDing")%></span></a>
				<a class='linkbutton linkbutton_nowanna' onclick='cancelOperCpeProType()'><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			<div class='cpeProSuccess'><%=rb.getString("XinJianChanPinLeiXingChengGong") %></div>
		</div>
	</c:if>
</div>
	
	
	<script>
		$(function(){
			if("${type}" == "view"){
				var selected = $("#cpeProductTypeTable").datagrid("getSelected");
				var productName = selected.PRODUCT_NAME;
				var model_id = selected.MODEL_ID;
				$("#proNameInput").val(productName);
				$("#proNameInput").attr("disabled",true);
				$("#cpeProductHead").html("<%=rb.getString("XinXi") %>")
			}else{
				$("#cpeProductHead").html("<%=rb.getString("XinJianChanPinLeiXing") %>")
			}
			// 表格数据
			$("#cpeProTypeTableList").datagrid({
				singleSelect:true,
				fit:true,
				fitColumns:true,
				border:false,
				rownumbers:true,
				pagePosition:'bottom',
				pagination: true,
				striped: true,
				onLoadSuccess:datagridLoadSuccess,
				columns: [[
					{field: 'PARAM_NAME',width:150,title:'<%=rb.getString("CanShuMingCheng") %>'},
					{field: 'PARAM_PATH',width:300,title:'<%=rb.getString("CPECanShuLuJing") %>'},
				]]
			});
			// 参数模型
			$("#modelInput").combobox({
				valueField:'id',
				textField:'model_name',
				url:'${ctx}/cell/CPE/getModelNameForCommbox.action',
				onLoadSuccess:function(data){
					if("${type}" == "add"){
						if(data.length > 0){
							$("#modelInput").combobox("setValue",data[0].id);
						}
					}
					if("${type}" == "view"){
						$("#modelInput").combobox("setValue",model_id);
						$("#modelInput").combobox("disable");
					}
					$("#cpeProTypeTableList").datagrid({
						url:'${ctx}/cell/CPE/getModelParamPathList.action',
						queryParams:{
							modelId : $("#modelInput").combobox("getValue")
						}
					});
				},
				onChange:function(){
					$("#cpeProTypeTableList").datagrid({
						url:'${ctx}/cell/CPE/getModelParamPathList.action',
						queryParams:{
							modelId : $("#modelInput").combobox("getValue")
						}
					});
				}
			})
		})
		// 产品类型名称验证
		function checkProductName(){
			var value = $("#proNameInput").val();
			var flag;
			var param ={
					productName : value.trim()
			}
			var reg = /^[a-zA-Z][a-zA-Z0-9\_\-\/\. ]*$/
			if(value == ""){
				$("#proNameInput").next().show().html("<%=rb.getString("ChanPinLeiXingBuNengWeiKong") %>");
				$("#proNameInput").addClass("errorborder");
			}else{
				if(value == "IDU-Standard" || value == "ODU-Standard"){
					$("#proNameInput").next().show().html("<%=rb.getString("NeiZhiChanPinLeiXing") %>");
					$("#proNameInput").addClass("errorborder");
				}else{
					if(reg.test(value)){
						$.post("${ctx}/cell/CPE/productNameExist.action",param,function(data){
							flag = data.message
							if(flag == "true"){
								$("#proNameInput").next().show().html("<%=rb.getString("ChanPinLeiXingYiCunZai") %>");
								$("#proNameInput").addClass("errorborder");
							}else{
								$("#proNameInput").next().hide().html("<%=rb.getString("ChanPinLeiXingBuNengWeiKong") %>");
								$("#proNameInput").removeClass("errorborder");
							}
						},"json")
					}else{
						$("#proNameInput").next().show().html("<%=rb.getString("BuNengShuRuZhongWen") %>");
						$("#proNameInput").addClass("errorborder");
					}
				}
			}
		}
		// 新建保存
		function saveCpeProType(){
			checkProductName();
			var ifPass = true;
			if($(".productErr").is(":visible")){
				ifPass = false;
			}
			if(ifPass){
				if(!$(".cpeProButton").hasClass("forbidden")){
					var productName = $("#proNameInput").val();
					var modelId = $("#modelInput").combobox("getValue");
					var params = {
						productName : productName,
						modelId : modelId,
						type:"save"
					}
					$(".cpeProButton").addClass("forbidden");
					$.post("${ctx}/cell/CPE/saveOrUpdateProductAndModelInfo.action",params,function(data){
						if(data["success"]){
							$('.cpeProSuccess').fadeIn(300,function(){
								var  time = setTimeout(function(){
									$(".titleButtonText").html("<%=rb.getString("TianJia")%>");
									$(".circleBg").addClass("add_circle").removeClass("close_circle addCpeProTypeFlag viewCpeProTypeFlag");
									$('#operCpeProTypeDiv').animate({right:"-900px"},300,function(){
										$("#cpeProductTypeTable").datagrid("reload");
									});
								},1500);
							})
						}else{
							showMsg('error_msg',data["message"]);
						}
					},"json")
				 }
			}
		}
	</script>
