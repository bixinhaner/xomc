<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.inportBg{
		vertical-align:middle;
		position:absolute;
		z-index:900;
		margin:1px 1px 0 -29px;
		border-left:1px solid #ddd;
		display:inline-block;
		width:24px;
		height:24px;
		background-color:#fff;
		margin-left:223px;
	}
</style>

<%-- 导入基站列表文件（Excel） --%>
<div id="ImportHalobEnodebDiv"  class="flex-ctn" style="height: 100%;">
	<div region="center" data-options="border:false" style="padding: 20px 30px;position:relative;" class="floatR flex-item">
		<div class="itemDiv" style="height:30px;font-size: 16px;margin-left:0px;width:395px;">
			<span style="display:block"><%=rb.getString("Title_SheBeiDaoRu")%></span>
		</div>
		<div class="itemDiv" style="margin-top:10px;height:50px;margin-left: 20px;">
			<span style="display:block"><%=rb.getString("DaoRuLeiXing")%></span>
			<select name="importType" class="border border-box item easyui-combobox" style="height:26px;" disabled="disabled">
				<option value="append"><%=rb.getString("ZhuiJia")%></option>
				<option value="cover"><%=rb.getString("FuGai")%></option>
			</select>
		</div>
		<div class="itemDiv" style="margin-top:20px;height:50px;margin-left: 20px;">
			<span style="display:block"><%=rb.getString("DaoRuWenJian")%></span>
			<a class="titleIcon_import inportBg" title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="selectFileButton()">
			</a>
			<input type="text" name="uploadFilePath" id="importHalobEnodebInput" style="vertical-align:middle;" readonly="readonly" class="border border-box item" >
			<div id="importHalobEnodebTitle" style="color:#797979;margin-top:8px;"><%=rb.getString("QingXianXuanZeWenJian")%></div>
    	</div>
	</div>
	<div region="south" data-options="border:false,height:57" style="min-height: 57px;">
	<div class="windowButtonGroup" style="margin-right:20px;">
	    <a href="#" class="linkbutton linkbutton_nowanna" onclick="downloadTemplateHalobeNB()" ><span style="min-width:auto;padding:0 20px"><%=rb.getString("DaoChuMuBan")%></span></a>
		<a href="#" class="linkbutton linkbutton_trend"  onclick="uploadTemplateHalobeNB()"><span><%=rb.getString("QueDing")%></span></a>
		<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeImportAddEnb();"><span><%=rb.getString("QuXiao")%></span></a>
	</div>
	</div>
</div>

<%-- 表单-上传基站列表文件 --%>
<form enctype="multipart/form-data" method="post" id="importHalobEnodebForm" style="display: none;"
	action="${ctx}/cell/halobSelfConfig/addDevicesParamConfig.action">
    <input name="uploadFile" type="file">
    <input name="importType" value="append">
</form>
<%-- 表单-用于下载基站列表文件模板 --%>
<form id="downLoadHalobEnodebForm" style="display:none" method="post"
      action="${ctx}/system/deviceGroup/downloadImportCellTemplate.action">
</form>
<script type="text/javascript">
//关闭添加基站窗口
function closeImportAddEnb(){
	$("#importHalobEnbBlock").slideUp(500,function(){$("#importHalobEnbBlock").html("")});
}

$(function() {
	$("#importHalobEnodebForm input[name='uploadFile']").bind("change", function() {
		$("#ImportHalobEnodebDiv input[name='uploadFilePath']").val(this.value);
		if (this.value) {
			$("#ImportHalobEnodebDiv input[name='uploadFilePath']").removeClass("err_border");
		}
	});
});

// 打开文件选择窗口
function selectFileButton() {
	$("#importHalobEnodebForm input[name='uploadFile']").click();
}

//上传文件
function uploadTemplateHalobeNB() {
	if ($("#importHalobEnodebForm input[name='uploadFile']")[0].files.length == 0) {
		return;
	}
	$("#importHalobEnodebForm").form("submit", {
		dataType: 'json',
		success: function (data) {
			if(typeof data == 'string') data = eval("(" + data + ")");
			closeImportAddEnb();
			
			// 清空
			$("#unsuc_count_importResult").html("");
			$("#suc_count_importResult").html("");
			$("#unsuc_info").html("");
			if (data["success"]) {
        		$("#singleDeviceConfig").datagrid('reload');
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
	 	            	if (data.unvalidsn) {
	 	            		unsuc_count = parseInt(data.unsuc.length) + parseInt(data.unvalidsn.length);
	 	            	}
	 	            	
	 	            	if (data.unExist) {
	 	            		unsuc_count += parseInt(data.unExist.length);
	 	            	}
	 	            	$("#unsuc_count_importResult").html("<%=rb.getString("DaoRuShiBaiShu")%><%=rb.getString("MaoHao")%> " + unsuc_count);
	 	            	$("#unsuc_info").append("<div style='font-weight: bold;'><%=rb.getString("JiZhanBuCunZai")%></div>");
	 	            	for (var i = 0; i < data.unsuc.length; i++) {
	 	            		$("#unsuc_info").append("<div>" + data.unsuc[i] + "</div>");
	 	            	}
	 	            }
	 	     		
	 	     		if (data.unExist) {
	 	     			$("#unsuc_info").append("<div style='font-weight: bold;'><%=rb.getString("YiCunZai")%></div>");
	 	            	for (var i = 0; i < data.unExist.length; i++) {
	 	            		$("#unsuc_info").append("<div>" + data.unExist[i] + "</div>");
	 	            	}
	 	     		}
	 	            if (data.unvalidsn) {
	 	            	$("#unsuc_info").append("<div style='font-weight: bold;'><%=rb.getString("XunLieHaoFeiFa")%></div>");
	 	            	for (var i = 0; i < data.unvalidsn.length; i++) {
	 	            		$("#unsuc_info").append("<div>" + data.unvalidsn[i] + "</div>");
	 	            	}
	 	            }
	 	     	  }
	 	       });
        	} else {
        		if (data.msg == "File format is wrong") {
        			$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("DaoRuWenJianGeShiTiShi")%>");
        		} else {
        			$.messager.alert("<%=rb.getString("TiShi")%>", data.msg);
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
function downloadTemplateHalobeNB() {
	//$("#downLoadHalobEnodebForm").form("submit");
	var url = $("#downLoadHalobEnodebForm").attr('action');
	exportByForm(url, {});
}
</script>