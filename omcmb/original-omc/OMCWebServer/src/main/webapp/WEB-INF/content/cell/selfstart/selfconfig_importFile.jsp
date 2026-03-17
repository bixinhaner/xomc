<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

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
</style>

<%-- 导入自配置数据（Excel） --%>
<div id="ImportSelfstartFile" class="easyui-layout" data-options="border:false,fit:true">
	<div region="center" data-options="border:false" style="padding: 20px;" class="floatR">
		<div class="itemDiv">
			<span><%=rb.getString("MuBanDaoChu")%></span>
			<a class="titleDiv titleIcon_export" title="<%=rb.getString("DaoChuWenJian")%>" href="javascript: void(0)" onclick="scanClick_downloadTemplate()" style="vertical-align:middle;float:right; position:relative;z-index:900;margin:1px 1px 0 -29px;border-left:1px solid #ddd;display:inline-block;width:23px;height:24px;background-color:#fff;"></a>
			<input type="text" name="exportFilePath" style="vertical-align:middle;" class="border border-box item" value="<%=rb.getString("DaoChuZiPeiZhiMuBan")%>" disabled="true">
		</div>
		<div class="itemDiv">
			<span><%=rb.getString("DaoRuWenJian")%></span>
			<a class="titleDiv titleIcon_import" title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClick_importFile()" style="vertical-align:middle;float:right; position:relative;z-index:900;margin:1px 1px 0 -29px;border-left:1px solid #ddd;display:inline-block;width:23px;height:24px;background-color:#fff;"></a>
			<input type="text" name="uploadFilePath" style="vertical-align:middle;" class="border border-box item" value="<%=rb.getString("DaoRuZiPeiZhiMuBan")%>" disabled="true">
		</div>
	</div>
	<div region="south" data-options="border:false,height:71" style="padding: 10px 20px 20px;">
		<div class="windowButtonGroup">
			<a href="#" class="linkbutton linkbutton_trend" onclick="uploadFile_importFile()"><span><%=rb.getString("QueDing")%></span></a>
			<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeDefaultWindow()"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
	</div>
</div>

<%-- 表单-上传自配置参数文件 --%>
<%-- <form enctype="multipart/form-data" id="formUploadSelfstartTemplate" style="display:none;" method="post" action="${ctx}/cell/selfstart/uploadExcelTemplate.action">
	<input id="uploadFile" name="uploadFile" type="file"/>
</form> --%>

<c:if test="${isRussiaType == '1'}">
	<form enctype="multipart/form-data" id="formUploadSelfstartTemplate" style="display:none;" method="post" action="${ctx}/cell/selfstart/uploadRussiaExcelTemplate.action">
		<input id="uploadFile" name="uploadFile" type="file"/>
	</form>
</c:if>

<c:if test="${isRussiaType == '0'}">
	<form enctype="multipart/form-data" id="formUploadSelfstartTemplate" style="display:none;" method="post" action="${ctx}/cell/selfstart/uploadExcelTemplate.action">
		<input id="uploadFile" name="uploadFile" type="file"/>
	</form>
</c:if>

<%-- 表单-用于下载自配置参数模板 --%>
<form id="formDownloadSelfstartTemplate" style="display:none;" method="post" action="${ctx}/cell/selfstart/downloadExcelTemplate.action"></form>

<script type="text/javascript">
$(function() {
	$("#formUploadSelfstartTemplate input[name='uploadFile']").bind("change", function() {
		$("#ImportSelfstartFile input[name='uploadFilePath']").val(this.value);
		if (this.value) {
			$("#winUploadSelfstartFile input[name='uploadFilePath']").removeClass("err_border");
		}
	});
});

function scanClick_downloadTemplate() {
	$("#formDownloadSelfstartTemplate").form("submit",{
		onSubmit: function(param){
			var bool = checkParams(param);
			if(!bool) return false;
        }
	});
}

function scanClick_importFile() {
	$("#uploadFile").click();
}

function uploadFile_importFile() {
	if ($("#formUploadSelfstartTemplate input[name='uploadFile']")[0].files.length == 0) {
		$("#winUploadSelfstartFile input[name='uploadFilePath']").addClass("err_border").fadeOut().fadeIn();
		return;
	}
	
	$("#formUploadSelfstartTemplate").form("submit", {
		onSubmit: function(param){
			var bool = checkParams(param);
			if(!bool) return false;
        },
		dataType: 'json',
		success: function (data) {

			if(typeof data == 'string') data = eval("(" + data + ")");

			if (data["success"]) {
        		<%-- $("#tableSelfStartParams").datagrid('reload');
 	            
 	            $("#suc_count_importSelfstartResult").html("<%=rb.getString("DaoRuChengGongShu")%><%=rb.getString("MaoHao")%> " + data.suc_count);
 	            
 	            var unsuc_count = 0;
 	            if (data.invalidSn) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("XunLieHaoFeiFa")%></div>");
 	            	for (var i = 0; i < data.invalidSn.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidSn[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidSn.length;
 	            }
 	            if (data.invalidBandwidth) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("DaiKuanBuZhiChi")%></div>");
 	            	for (var i = 0; i < data.invalidBandwidth.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidBandwidth[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidBandwidth.length;
 	            }
 	            if (data.invalidBand) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PinDuanCuoWu")%></div>");
 	            	for (var i = 0; i < data.invalidBand.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidBand[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidBand.length;
 	            }
 	            if (data.invalidCellid) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("XiaoQuIDChaoChuFanWei")%></div>");
 	            	for (var i = 0; i < data.invalidCellid.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidCellid[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidCellid.length;
 	            }
 	            if (data.invalidDlEarfcn) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PinDianSheZhiCuoWu")%></div>");
 	            	for (var i = 0; i < data.invalidDlEarfcn.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidDlEarfcn[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidDlEarfcn.length;
 	            }
 	            if (data.invalidDlUlDiff) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("ShangXiaXingPinDianBuYiZhi")%></div>");
 	            	for (var i = 0; i < data.invalidDlUlDiff.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidDlUlDiff[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidDlUlDiff.length;
 	            }
 	            if (data.invalidPci) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PCIChaoChuFanWei")%></div>");
 	            	for (var i = 0; i < data.invalidPci.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidPci[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidPci.length;
 	            }
 	            if (data.invalidPa) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PAChaoChuFanWei")%></div>");
 	            	for (var i = 0; i < data.invalidPa.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidPa[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidPa.length;
 	            }
 	            if (data.invalidPb) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PBChaoChuFanWei")%></div>");
 	            	for (var i = 0; i < data.invalidPb.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidPb[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidPb.length;
 	            }
 	            if (data.invalidTac) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("TacChaoChuFanWei")%></div>");
 	            	for (var i = 0; i < data.invalidTac.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidTac[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidTac.length;
 	            }
 	            if (data.invalidRefPow) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("ChaoKaoXinHaoGongLvChaoChuFanWei")%></div>");
 	            	for (var i = 0; i < data.invalidRefPow.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidRefPow[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidRefPow.length;
 	            }
 	            if (data.invalidMME) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("HeXinWangIPDiZhiFeiFa")%></div>");
 	            	for (var i = 0; i < data.invalidMME.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidMME[i] + "</div>");
 	            	}
 	            	unsuc_count += data.invalidMME.length;
 	            }
 	            if (data.existCell) {
 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("YiCunZai")%></div>");
 	            	for (var i = 0; i < data.existCell.length; i++) {
 	            		$("#unsucSelfstart_info").append("<div>" + data.existCell[i] + "</div>");
 	            	}
 	            	unsuc_count += data.existCell.length;
 	            }
 	            
 	           if (unsuc_count > 0) {
 	        	  $("#unsuc_count_importSelfstartResult").html("<%=rb.getString("DaoRuShiBaiShu")%><%=rb.getString("MaoHao")%> " + unsuc_count);
 	           }
 	           $("#winImportSelfstartResult").window("open"); --%>
 	           var url = '${ctx}/cell/selfstart/toSelfStartPages.action';
	 	       openDefaultWindow(url,{
	 	      		title: '<%=rb.getString("TiShi")%>',
	 	      		width:380,height:280,
	 	     		onLoad: function(){
		 	     		// 清空
		 	   			$("#suc_count_importSelfstartResult").html("");
		 	   			$("#unsuc_count_importSelfstartResult").html("");
		 	   			$("#unsucSelfstart_info").html("");
		 	   			
	 	     			$("#tableSelfStartParams").datagrid('reload');
	 	 	            
	 	 	            $("#suc_count_importSelfstartResult").html("<%=rb.getString("DaoRuChengGongShu")%><%=rb.getString("MaoHao")%> " + data.suc_count);
	 	 	            
	 	 	            var unsuc_count = 0;
	 	 	            if (data.invalidSn) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("XunLieHaoFeiFa")%></div>");
	 	 	            	for (var i = 0; i < data.invalidSn.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidSn[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidSn.length;
	 	 	            }
	 	 	            if (data.invalidBandwidth) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("DaiKuanBuZhiChi")%></div>");
	 	 	            	for (var i = 0; i < data.invalidBandwidth.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidBandwidth[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidBandwidth.length;
	 	 	            }
	 	 	            if (data.invalidBand) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PinDuanCuoWu")%></div>");
	 	 	            	for (var i = 0; i < data.invalidBand.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidBand[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidBand.length;
	 	 	            }
	 	 	            if (data.invalidCellid) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("XiaoQuIDChaoChuFanWei")%></div>");
	 	 	            	for (var i = 0; i < data.invalidCellid.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidCellid[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidCellid.length;
	 	 	            }
	 	 	            if (data.invalidDlEarfcn) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PinDianSheZhiCuoWu")%></div>");
	 	 	            	for (var i = 0; i < data.invalidDlEarfcn.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidDlEarfcn[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidDlEarfcn.length;
	 	 	            }
	 	 	            if (data.invalidDlUlDiff) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("ShangXiaXingPinDianBuYiZhi")%></div>");
	 	 	            	for (var i = 0; i < data.invalidDlUlDiff.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidDlUlDiff[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidDlUlDiff.length;
	 	 	            }
	 	 	            if (data.invalidPci) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PCIChaoChuFanWei")%></div>");
	 	 	            	for (var i = 0; i < data.invalidPci.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidPci[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidPci.length;
	 	 	            }
	 	 	            if (data.invalidPa) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PAChaoChuFanWei")%></div>");
	 	 	            	for (var i = 0; i < data.invalidPa.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidPa[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidPa.length;
	 	 	            }
	 	 	            if (data.invalidPb) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("PBChaoChuFanWei")%></div>");
	 	 	            	for (var i = 0; i < data.invalidPb.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidPb[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidPb.length;
	 	 	            }
	 	 	            if (data.invalidTac) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("TacChaoChuFanWei")%></div>");
	 	 	            	for (var i = 0; i < data.invalidTac.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidTac[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidTac.length;
	 	 	            }
	 	 	            if (data.invalidRefPow) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("ChaoKaoXinHaoGongLvChaoChuFanWei")%></div>");
	 	 	            	for (var i = 0; i < data.invalidRefPow.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidRefPow[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidRefPow.length;
	 	 	            }
	 	 	            if (data.invalidMME) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("HeXinWangIPDiZhiFeiFa")%></div>");
	 	 	            	for (var i = 0; i < data.invalidMME.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.invalidMME[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.invalidMME.length;
	 	 	            }
	 	 	            if (data.existCell) {
	 	 	            	$("#unsucSelfstart_info").append("<div style='font-weight: bold;'><%=rb.getString("YiCunZai")%></div>");
	 	 	            	for (var i = 0; i < data.existCell.length; i++) {
	 	 	            		$("#unsucSelfstart_info").append("<div>" + data.existCell[i] + "</div>");
	 	 	            	}
	 	 	            	unsuc_count += data.existCell.length;
	 	 	            }
	 	 	            
	 	 	           if (unsuc_count > 0) {
	 	 	        	  $("#unsuc_count_importSelfstartResult").html("<%=rb.getString("DaoRuShiBaiShu")%><%=rb.getString("MaoHao")%> " + unsuc_count);
	 	 	           }
	 	     		}
	 	       }); 
        	} else {
        		if (data.emptyCell) {
        			$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("ShuJuBuNengWeiKong")%>");
        		} else {
        			$.messager.alert("<%=rb.getString("TiShi")%>", data.msg);
        		}
        	}
        }
	});
}
</script>