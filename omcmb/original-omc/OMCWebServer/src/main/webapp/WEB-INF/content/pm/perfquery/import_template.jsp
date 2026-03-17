<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.inportBg { vertical-align: middle; position: absolute; z-index: 900; margin: 1px 1px 0 -29px; border-left: 1px solid #ddd; display: inline-block; width: 20px; height: 20px; background-color: #fff; margin-left: 223px; padding-top: 3px; padding-left: 4px; }
	.templateErrorTip { color: red; display: none; }
	#importTemplateDiv .itemDiv span { width: 150px; }
	.tipText { color: #999; }
	.exportLink { color: #363B4E; font-weight: bold; text-decoration: underline; cursor: pointer; margin-left: 7px; }
</style>

<%-- 导入模板列表文件（Excel） --%>
<div id="importTemplateDiv"  class="flex-ctn" style="height: 100%;">
	<div class="cardHeader">
		<%=rb.getString("DaoRu")%>
		<a class="el-icon el-icon-close slideIcon" onclick="closeImportAddEnb()"></a>
	</div>
	<div style="padding: 20px;" class="cardBody">
		<div style="height:65px;">
			<div style="margin-bottom:7px;"><%=rb.getString("MuBanMingCheng")%></div>
			<input id="templateName" maxlength="200" class="inputDivCss border border-box" style="width:250px;"/>
			<div id="templateName_err" class="templateErrorTip"><%=rb.getString("QingShuRuMuBanMingCheng")%></div>
		</div>
		<div style="height:65px;">
			<div style="margin-bottom:7px;"><%=rb.getString("DaoRuWenJian")%></div>
			<a class="el-icon el-icon-operation-import inportBg" title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="selectFileButton()"></a>
			<input type="text" name="uploadFilePath" id="templateImportInput" style="vertical-align:middle;" readonly="readonly" class="border border-box item" >
			<div id="templateImportInput_err" class="templateErrorTip"><%=rb.getString("QingXianXuanZeWenJian")%></div>
    	</div>
    	<div class="tipText"><%=rb.getString("DaoRuWenJianTiShi")%> <span class="exportLink" onclick="downloadKpiTemplate()"><i class="el-icon el-icon-common-download"></i><%=rb.getString("DaoChuMuBan")%></span></div>
    	<div style="height:100px;margin-top:10px;">
			<div style="margin-bottom:7px;"><%=rb.getString("MiaoShu")%></div>
			<textarea id="templateDescription" maxlength="500" class="inputDivCss border border-box" style="width:400px;height:80px;vertical-align:top;resize:none;"></textarea>
		</div>
	</div>
	<div class="cardFooter">
		<span class="el-button el-button--primary templateImportButton"  onclick="uploadKpiTemplate()"><%=rb.getString("QueDing")%></span>
		<span class="el-button" onclick="closeImportAddEnb();"><%=rb.getString("QuXiao")%></span>
	</div>
</div>

<%-- 表单-上传模板列表文件 --%>
<form enctype="multipart/form-data" method="post" id="importTemplateForm" style="display: none;"
	action="${ctx}/pm/template/uploadQueryTemplate.action">
    <input name="uploadFile" type="file">
    <input name="tempName" value="">
    <input name="description" value="">
    <input name="timeZone" value="">
</form>
<%-- 表单-用于下载模板列表文件模板 --%>
<form id="downLoadKpiTemplateForm" style="display:none" method="post"
      action="${ctx}/pm/template/downloadKPIQueryTemplate.action">
</form>

<script type="text/javascript">
	var curImportEnbGnbEgwType = sessionStorage.getItem('importTemplateEnbGnbOrEgw');
	
	$(function() {
		$("#importTemplateForm input[name='uploadFile']").bind("change", function() {
			$("#importTemplateDiv input[name='uploadFilePath']").val(this.value);
			
			if (this.value == "") {
				return;
			}
			//校验文件格式，只支持 .csv 格式
			if(fileFormatMatch(this.value,"csv")){
				$("#templateImportInput_err").hide();
			}else{
				$("#templateImportInput_err").html("<%=rb.getString("ZhiZhiChiCSVFile")%>");
				$("#templateImportInput_err").show();
			}
		});
		$("#templateName").blur(function(){
			if($(this).val().trim().length > 0){
				$("#templateName_err").hide();
				$("#importTemplateForm input[name='tempName']").val(this.value);
				$("#importTemplateForm input[name='timeZone']").val(timeZone);
			}else{
				$("#templateName_err").show();
			}
		})
	});
	
	//关闭添加基站窗口
	function closeImportAddEnb(){
		$("#importTemplateBlock").slideUp(500,function(){$("#importTemplateBlock").html("")});
	}
	
	// 打开文件选择窗口
	function selectFileButton() {
		$("#importTemplateForm input[name='uploadFile']").click();
	}
	//上传文件
	function uploadKpiTemplate() {
		var errTipShow = false, curSaveUrl = '';
		
		if ($("#importTemplateForm input[name='uploadFile']")[0].files.length == 0) {
			$("#templateImportInput_err").show();
			errTipShow = true;
		}
		
		if($("#templateImportInput_err").is(':visible')){
			errTipShow = true;
		}
		
		if($("#templateName").val().trim().length == 0 ){
			$("#templateName_err").show();
			errTipShow = true;
		}

		if(errTipShow){
			return;
		}
		
		$("#importTemplateDiv").addClass("loading");
		if(!$(".templateImportButton").hasClass("forbidden")){
			$(".templateImportButton").addClass("forbidden");
			
			if(curImportEnbGnbEgwType == '0'){
				curSaveUrl = $("#importTemplateForm").attr("action");
			}else if(curImportEnbGnbEgwType == '1'){
				curSaveUrl = '${ctx}/gnb/pm/template/uploadQueryTemplate.action';
			}else{
				curSaveUrl = '${ctx}/egw/pm/template/uploadQueryTemplate.action';
			}
			
			uploadWithProgress({
		    	url: curSaveUrl,
		    	form: document.querySelector("#importTemplateForm"),
		    	progress: function(ev){},
		    	success: function(data){
		    		if(typeof data == 'string') data = eval("(" + data + ")");
					$("#importTemplateDiv").removeClass("loading");
					closeImportAddEnb();
					
					if (data["success"]) {
						showMsg('success_msg',data.msg)
						$("#kpiTemplateDatagrid").datagrid("reload");
		        	}else {
	        			showMsg('error_msg',data.msg)
	        		}
		    	}
		    });
		}	
	}

	// 下载模板列表文件模板
	function downloadKpiTemplate() {
		var url = '';
		if(curImportEnbGnbEgwType == '0'){
			url = $("#downLoadKpiTemplateForm").attr('action');
		}else if(curImportEnbGnbEgwType == '1'){
			url = '${ctx}/gnb/pm/template/downloadKPIQueryTemplate.action';
		}else{
			url = '${ctx}/egw/pm/template/downloadKPIQueryTemplate.action';
		}
		
		exportByForm(url, {});
	}
</script>