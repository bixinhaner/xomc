<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.detailMesDiv div{
		display:inline-block;
	}
	#egwEditUpgradeFileInfo label{
		margin-bottom:5px;
		display:block;
	}
	#egwEditUpgradeFileInfo p{
		font-size:12px;
		color:#CC0000;
		margin-top:5px;
		margin-bottom:7px;
		visibility:hidden;
	}
</style>
<div id="egwEditUpgradeFileInfo" style="display:flex;flex-direction:column;height:100%;">
		<div class='el-card__header'>
			<span><%=rb.getString("WenJianXinXi")%></span>
			<span class='el-icon el-icon-close' onclick='closeEditegwFileWindow()'></span>
		</div>
		<div class='el-card__body' style='display:flex;flex-direction:column;flex:1 1 auto;height:100%;overflow:auto;'>
			<div class='detailMesDiv'  style="background:#fff;border:1px solid #EEE">
			    <div style='margin:20px;'>
				    <input name="fileType" type="hidden"/>
				    <div>
						<label for="egwFilePathEdit" style="width: 120px;"><%=rb.getString("WenJianMing")%><%=rb.getString("MaoHao")%></label>
						<input value='${fileName }' id="egwFilePathEdit" type="text" class="border border-box file_info" readonly style="width: 350px;"/>
					</div>
				    <div style='margin-left:40px;'>
				    	<label for="egwProductEdit" style="width: 120px;"><%=rb.getString("ChanPinLeiXingBiaoZhi")%><%=rb.getString("MaoHao")%></label>
						<input value='${productType }' id="egwProductEdit" readonly type="text" class="border border-box file_info required" style="width: 350px;height:26px;" />
					</div>
					<div style='margin-top:20px;'>
						<label for="egwVersionEdit" style="width: 120px;"><%=rb.getString("BanBen")%><%=rb.getString("MaoHao")%></label>
						<input value='${version }' onblur='checkVersion()' id="egwVersionEdit" type="text" class="border border-box file_info required" maxlength=45 style="width: 350px;height:26px;"/>
						<p><%=rb.getString("ShuRuBiTianXiang")%></p>
					</div>
					<div style="margin-right:35px;margin-top:20px;" id="descEditBox">
						<label for="egwDescEdit" style="width: 120px; vertical-align: top;"><%=rb.getString("MiaoShu")%><%=rb.getString("MaoHao")%></label>
						<textarea id="egwDescEdit" cols="20" style="padding-top:5px;font-size: 12px;width: 766px; height: 377px; resize: none;" rows="5"
							maxlength=500 class="border-box border file_info">${description }</textarea>
					</div>
				</div>
			</div>
		</div>
		<div class='el-card__footer'>
			<div class="windowButtonGroup" style='float:left'>
				<a class="linkbutton linkbutton_trend" onclick="submitegwEdit()"><span><%=rb.getString("QueDing")%></span></a>
				<a class="linkbutton linkbutton_nowanna " onclick="closeEditegwFileWindow();"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
		</div>
		
</div>
<script>
	function checkVersion(){
		if($("#egwVersionEdit").val() == ""){
			$("#egwVersionEdit").siblings("p").css("visibility","visible");
			return false;
		}else{
			$("#egwVersionEdit").siblings("p").css("visibility","hidden");
			return true;
		}
	}
	function submitegwEdit(){
		if($("#egwVersionEdit").val() == ""){
			$("#egwVersionEdit").siblings("p").css("visibility","visible");
		}else{
			$("#egwVersionEdit").siblings("p").css("visibility","hidden");
			var select = $("#egwFileUpgradeTable").datagrid("getSelected");
			var id = select["id"];
			var param = {
					id:id,
					file_name:'${fileName}',
					oldVersion:'${version}',
					newVersion:$("#egwVersionEdit").val(),
					oldDescription:'${descriptioin}',
					newDescription:$("#egwDescEdit").val()
			}
			$.post("${ctx}/egw/softwareFile/editSoftwareFileInfos.action",param,function(data){
				if(data["success"]){
					showMsg('success_msg','<%=rb.getString("ChengGong")%>');
						var  time = setTimeout(function(){
						$('.editegwUpgradeFileDiv').animate({right:'-1000px'},400,function(){
							$('.editegwUpgradeFileDiv').html("");
							$("#egwFileUpgradeTable").datagrid("reload");
						});
					})
				}else{
					showMsg('error_msg',data["message"]);
				}
			},"json")
		}
	}
</script>