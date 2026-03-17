<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 导入CPE列表文件（Excel） --%>
<style>
	.floatR>div input{
		float:right;
	}
	.floatR>div select{
		float:right;
	}
	.floatR .itemDiv .combo{
		float:right;
	}
	.cpeImportErrorTip{
		color:red;
		margin-left:216px;
		margin-top:5px;
		display:none;
	}
</style>

<div id="ImportDeviceGroupCpeFile" class="easyui-layout" data-options="border:false,fit:true">
	<div region="center" data-options="border:false" style="padding: 20px;" class="floatR">
		<div class="itemDiv hidden">
			<span><%=rb.getString("DaoRuLeiXing")%></span>
			<select name="importType" class="border border-box item easyui-combobox" disabled="disabled">
				<option value="append"><%=rb.getString("ZhuiJia")%></option>
				<option value="cover"><%=rb.getString("FuGai")%></option>
			</select>
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("DaoRuWenJian")%></span>
			<a class="titleIcon_import " title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClick_deviceGroup()" 
			   style="vertical-align:middle;float:right; position:relative;z-index:900;margin:1px 1px 0 -29px;border-left:1px solid #ddd;display:inline-block;width:23px;height:24px;background-color:#fff;">
			</a>
			<input type="text"  name="uploadFilePath" style="vertical-align:middle;" readonly="readonly" class="border border-box item" >
			<div id="cpeImportInput_err" class="cpeImportErrorTip"><%=rb.getString("QingXianXuanZeWenJian")%></div>
		</div>
	</div> 
	<div region="south" data-options="border:false,height:57">
		<div class="windowButtonGroup" style="margin-right:20px">
			<a href="#" class="linkbutton linkbutton_nowanna" onclick="downloadTemplate_deviceGroupCpe()"><span style="min-width:auto;padding:0 20px"><%=rb.getString("DaoChuMuBan")%></span></a>
			<a href="#" class="linkbutton linkbutton_trend"  onclick="uploadFile_deviceGroupCpe()"><span><%=rb.getString("QueDing")%></span></a>
			<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeDefaultWindow();"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
	</div>
</div>

<%-- 表单-上传CPE列表文件 --%>
<form enctype="multipart/form-data" method="post" id="uploadForm_deviceGroupCpe" style="display: none;"
	action="${ctx}/cell/CPE/uploadFile.action">
    <input name="uploadFile" type="file">
    <input name="importType" value="append">
    <input name="group_id" value="append">
</form>

<script type="text/javascript">
$(function() {
	$("#uploadForm_deviceGroupCpe input[name='uploadFile']").bind("change", function() {
		$("#ImportDeviceGroupCpeFile input[name='uploadFilePath']").val(this.value);
		if (this.value =="") {
			return;
		}
		if(fileFormatMatch(this.value,"xlsx,xls,csv")){
			$("#cpeImportInput_err").html("<%=rb.getString("QingXianXuanZeWenJian")%>");
			$("#cpeImportInput_err").hide();
		}else{
			$("#cpeImportInput_err").html("<%=rb.getString("DaoRuWenJianGeShi")%>");
			$("#cpeImportInput_err").show();
		}
	});
});

// 打开文件选择窗口
function scanClick_deviceGroup() {
	$("#uploadForm_deviceGroupCpe input[name='uploadFile']").click();
}

//上传文件
function uploadFile_deviceGroupCpe() {
	//$("#uploadForm_deviceGroupCpe input[name='importType']").val($("#ImportDeviceGroupCpeFile select[name='importType']").val());
	$("#uploadForm_deviceGroupCpe input[name='group_id']").val($("#gridDeviceGroup").datagrid("getSelected")["id"]);
	if ($("#uploadForm_deviceGroupCpe input[name='uploadFile']")[0].files.length == 0) {
		$("#cpeImportInput_err").html("<%=rb.getString("QingXianXuanZeWenJian")%>").show();
		return;
	}
	if($("#cpeImportInput_err").is(':visible')){
		return;
	}
	$("#uploadForm_deviceGroupCpe").form("submit", {
		dataType: 'json',
		success: function (data) {
			if(typeof data == 'string') data = eval("(" + data + ")");
			/* $("#winImportDevice").window("close"); */
			closeDefaultWindow();
			
			// 清空
			$("#unsuc_count_importResult").html("");
			$("#suc_count_importResult").html("");
			$("#unsuc_info").html("");

			if (data["success"]) {
        		$("#tableDeviceGroupCpe").datagrid('reload');
        		
 	           var url = '${ctx}/system/deviceGroup/toDiviceGroupPage.action?code=result';
	 	       openDefaultWindow(url,{
	 	     	   title: '<%=rb.getString("TiShi")%>',
	 	     	   width: 380,
				   height: 280,
	 	     	   onLoad: function(){
	 	     		if (data.repeatFromExcel) {
	  	            	$("#suc_count_importResult").html("<%=rb.getString("BiaoGeYouChongFuShuJu")%>" + "<br/>" + "<%=rb.getString("DaoRuChengGongShu")%><%=rb.getString("MaoHao")%> " + data.suc_count);
	  	            	$("#unsuc_info").html("<div style='font-weight: bold;'><%=rb.getString("ChongFuShuJu")%></div>");
	  	            	for (var i = 0; i < data.repeatFromExcel.length; i++) {
	  	            		$("#unsuc_info").append("<div>" + data.repeatFromExcel[i] + "</div>");
	  	            	}
	  	            } else {
	  	            	$("#suc_count_importResult").html("<%=rb.getString("DaoRuChengGongShu")%><%=rb.getString("MaoHao")%> " + data.suc_count);
	  	            }
	 	     		if (data.unsuc) {
	 	            	var unsuc_count = parseInt(data.unsuc.length);
	 	            	if (data.unValidMac) {
	 	            		unsuc_count = parseInt(data.unsuc.length) + parseInt(data.unValidMac.length);
	 	            	}
	 	            	$("#unsuc_count_importResult").html("<%=rb.getString("DaoRuShiBaiShu")%><%=rb.getString("MaoHao")%> " + unsuc_count);
	 	            	$("#unsuc_info").append("<div style='font-weight: bold;'><%=rb.getString("YiCunZai")%></div>");
	 	            	for (var i = 0; i < data.unsuc.length; i++) {
	 	            		$("#unsuc_info").append("<div>" + data.unsuc[i] + "</div>");
	 	            	}
	 	            }
	 	            if (data.unValidMac) {
	 	            	$("#unsuc_info").append("<div style='font-weight: bold;'><%=rb.getString("MACDiZhiFeiFa")%></div>");
	 	            	for (var i = 0; i < data.unValidMac.length; i++) {
	 	            		$("#unsuc_info").append("<div>" + data.unValidMac[i] + "</div>");
	 	            	}
	 	            }
	 	     	  }
	 	       }); 
        	} else {
        		if (data.msg == "File format is wrong") {
        			showMsg('prompt_msg',"<%=rb.getString("DaoRuWenJianGeShiTiShi")%>");
        		} else {
        			showMsg('error_msg',data.msg);
        		}
        	}
        },
        onSubmit: function(param){
			var bool = checkParams(param)
			if(!bool) return false;
		}
	});
}

// 下载基站列表文件模板
function downloadTemplate_deviceGroupCpe() {
	$("#formDownloadTemplate_deviceGroupCpe").form("submit",{
		onSubmit: function(param){
			var bool = checkParams(param)
			if(!bool) return false;
		}
	});
}
</script>