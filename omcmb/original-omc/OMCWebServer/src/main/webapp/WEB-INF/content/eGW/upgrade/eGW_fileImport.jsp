<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#egwUpgradeFileInfoContent label{
		display:block;
		margin-bottom:10px;
		color:#4C6778;
	}
	#egwUpgradeFileInfoContent input{
		width:350px;
		height:26px;
	}
	.errorMes{
		color:#CC0000;
		display:block;
		margin-top:5px;
		visibility:hidden
	}
	.operationDiv{
		margin-left:5px;
	}
	.errorborder{
		border:1px solid #CC0000;
	}
</style>
<div id="egwUpgradeFileInfoContent">
	<div style='height:50px;width:100%;margin-top:10px;'>
		<div style="display:inline-block;margin-left:30px;line-height:50px;color:#7993B6;font-size:16px;"><%=rb.getString("WenJianXinXi")%></div>
		<a class="el-icon el-icon-close" style="font-size:18px;float:right;margin-top:0px;margin-right:20px;" onclick="cancelImportegwFile()"></a>
	</div>
	<div style='margin-left:40px;margin-top:10px;'>
		<div>
			<label for="filePath" style="width: 120px;"><%=rb.getString("WenJian")%><%=rb.getString("MaoHao")%></label>
			<input id="egwfilePath" type="text" class="border border-box file_info" readonly="readonly" style="vertical-align:middle;padding-right:27px;"/>
			<a class="el-icon el-icon-operation-import" title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="egwscanClick()" 
			    style="vertical-align:middle; margin:0 2px 0 -29px;">
			</a>
			<span class="operationDiv operation_getFocus"></span>
			<span class='errorMes'  id="egwFileCheckMes"><%=rb.getString("QingXianXuanZeWenJian")%></span>
		</div>
		<div style='margin-top:20px;'>
			<label for="product" style="width: 120px;"><%=rb.getString("ChanPinLeiXingBiaoZhi")%><%=rb.getString("MaoHao")%></label>
			<input style='background:#EAF1F4' value='eGW' readonly id="product" type="text" class="border border-box file_info required" maxlength=100 />
			<span class="operationDiv operation_getFocus"></span>
		</div>
		<div style='margin-top:40px;'>
			<label for="version" style="width: 120px;"><%=rb.getString("BanBen")%><%=rb.getString("MaoHao")%></label>
			<input onblur='checkVersion()' id="egwversion" type="text" class="border border-box file_info required" maxlength=45/>
			<span class="operationDiv operation_getFocus"></span>
			<span class='errorMes' id='checkVersionMes'><%=rb.getString("ShuRuBiTianXiang")%></span>
		</div>
		<div style="margin-top:20px;" id="descBox">
			<label for="desc" style="width: 120px; vertical-align: top;"><%=rb.getString("MiaoShu")%><%=rb.getString("MaoHao")%></label>
			<textarea onblur='checkTextarea()' id="egwdesc" cols="20" style="font-size: 12px;width: 350px; height: 130px; resize: none;" rows="5"
				 class="border-box border file_info"></textarea>
			<span class='errorMes' id='checkTextareaMes'>长度超出显示，最大长度500</span>
		</div>
		<div class="windowButtonGroup" style='margin-top:20px;float:left'>
			<a class="linkbutton linkbutton_trend" onclick="egwsubmitUploadForm()"><span><%=rb.getString("QueDing")%></span></a>
			<a class="linkbutton linkbutton_nowanna" onclick="cancelImportegwFile();"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
	</div>
</div>
<%-- 上传文件的用的表单 --%>
<form enctype="multipart/form-data" method="post" id="egwuploadFileForm">
    <!-- <input name="uploadFile" value="" hidden="true"> -->
    <input name="fileSize"  value="" hidden="true">
    <input name="productType" value="" hidden="true">
    <input name="version" value="" type="hidden"/>
    <input name="description" value="" type="hidden"/>
    <input name="uploadFile"  id="egwuploadFile_filelib"  type="file" style="display: none;">
</form>
<script>
	$(function(){
		$("#egwuploadFile_filelib").bind("change", function() {
	    	checkFile();
	    	$("#egwUpgradeFileInfoContent #egwfilePath").val(this.value);
		});
	})
	function egwscanClick(){
		$('#egwuploadFile_filelib').click();
	}
	function egwsubmitUploadForm(){
		var files = document.getElementById("egwuploadFile_filelib").files;
	    var filePath = $("#egwuploadFile_filelib").val();
	    var pathSplit = filePath.split(/\\/);
	    var filename = pathSplit[pathSplit.length - 1];
	    checkFile();
	    checkVersion();
	    checkTextarea();
	    var egwImportPass = true;
	    $(".border").each(function(index,item){
	    	if($(item).hasClass("errorborder")){
	    		egwImportPass = false;	
	    	}
	    })
	    if(egwImportPass){
	        $("#egwuploadFileForm [name=fileSize]").val(files[0].size);
		    $("#egwuploadFileForm [name=productType]").val("eGW");
		    $("#egwuploadFileForm [name=version]").val($("#egwversion").val());
		    $("#egwuploadFileForm [name=description]").val($("#egwdesc").val());
		    /* $("#egwuploadFileForm").form('submit', {
		        url: "${ctx}/egw/softwareFile/uploadSoftwareFile.action",
		        success: function (data) {
		        	if(typeof data == 'string') {
		        		var data = JSON.parse(data);
		        	}
		        	if(data["success"]){
		        		cancelImportegwFile();
			        	$("#egwFileUpgradeTable").datagrid("reload");
		        	}else{
		        		showMsg('error_msg',data["message"]);
		        	}
		        	
		        },
		        onSubmit: function(param){
					var bool = checkParams(param)
					if(!bool) return false;
		        }
		    }); */
		    uploadWithProgress({
		    	url: "${ctx}/egw/softwareFile/uploadSoftwareFile.action",
		    	form: document.querySelector("#egwuploadFileForm"),
		    	success: function (data) {
		        	if(data["success"]){
		        		cancelImportegwFile();
			        	$("#egwFileUpgradeTable").datagrid("reload");
		        	}else{
		        		showMsg('error_msg',data["message"]);
		        	}
		        	
		        }
		    });
	    }
	}
	function checkFile(){
		var files = document.getElementById("egwuploadFile_filelib").files;
	    var filePath = $("#egwuploadFile_filelib").val();
	    if (filePath.length < 1) {
	    	$("#egwFileCheckMes").css("visibility","visible");
	    	$("#egwFileCheckMes").html("<%=rb.getString("QingXianXuanZeWenJian")%>");
	    	$("#egwfilePath").addClass("errorborder");
	        return;
	    }
	    var pathSplit = filePath.split(/\\/);
	    var filename = pathSplit[pathSplit.length - 1];
	    if (filename.length > 100) {
	    	$("#egwFileCheckMes").css("visibility","visible");
	    	$("#egwfilePath").addClass("errorborder");
	    	$("#egwFileCheckMes").html("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
	    	return;
	    }
	    if(filePath.length>=1 && filename.length<=100){
			$("#egwFileCheckMes").css("visibility","hidden");
	    	$("#egwfilePath").removeClass("errorborder");
	    }
	}
	function checkVersion(){
		var value = $("#egwversion").val();
		if(value == ""){
			$("#checkVersionMes").css("visibility","visible");
			$("#egwversion").addClass("errorborder");
		}else{
			$("#checkVersionMes").css("visibility","hidden");
			$("#egwversion").removeClass("errorborder");
		}
	}
	function checkTextarea(){
		var value = $("#egwdesc").val();
		if(value.length > 500){
			$("#checkTextareaMes").css("visibility","visible");
			$("#egwdesc").addClass("errorborder");
		}else{
			$("#checkTextareaMes").css("visibility","hidden");
			$("#egwdesc").removeClass("errorborder");
		}
	}
</script>