<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 编辑配置文件任务 --%>
<div id="editConfigFileAutoTask" class="easyui-panel" data-options="border:false,fit:true">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false">
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="north" data-options="border:false,height: 66" style="padding:20px;">
					<input id="autoTaskConfigFileInput" type="text" readonly="readonly" class="border border-box" style="width:235px;float:left;"/>
					<a class="easyui-linkbutton" href="javascript:void(0)" onclick="scanClick_editConfigFileAutoTask()" style="float:left;margin-left:20px;"><%=rb.getString("XuanZeWenJian")%></a>
				</div>
				<div region="center" data-options="border:false">
					<div class="easyui-layout" data-options="border:false,fit:true">
						<div region="north" data-options="border:false,height:35" style="padding: 10px 20px;">
							<pre><%=rb.getString("WenJianDaXiao")%> <%=rb.getString("MaoHao")%> ${filesize} Byte<%=rb.getString("DouHao")%> MD5 <%=rb.getString("MaoHao")%> ${md5}</pre>
						</div>
						<div region="center" data-options="border:false" style="padding: 0 20px 10px 20px;">
							<pre style="background-color: #EAF1F4;border-radius: 10px;padding: 10px;overflow: auto;height: 330px;"><c:out value="${file_content}" escapeXml="true"/></pre>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div region="south" data-options="border:false,height:57" style="padding: 10px 20px 20px;">
			<a class="easyui-linkbutton" onclick="closeWinEditConfigFileAutoTask()"
				style="float: right;"><%=rb.getString("QuXiao")%></a>
			<a class="easyui-linkbutton" onclick="saveConfigFileAutoTask()"
				style="float: right;margin-right: 20px;"><%=rb.getString("QueDing")%></a>
		</div>
	</div>
</div>

<%-- 表单，上传文件 --%>
<form enctype="multipart/form-data" method="post" id="formUploadAutoTaskConfigFile">
	<input name="uploadFile" type="file" style="display: none;">
</form>

<script type="text/javascript">
$(function() {
	$("#formUploadAutoTaskConfigFile input[name='uploadFile']").bind("change", function(e) {
		$("#autoTaskConfigFileInput").val(e.target.value);
	});
});

// 关闭窗口-编辑uboot自动任务
function closeWinEditConfigFileAutoTask() {
	/* $("#winEditConfigFileTask").window("close"); */
	closeDefaultWindow();
}

function scanClick_editConfigFileAutoTask() {
	$("#formUploadAutoTaskConfigFile input[name='uploadFile']").click();
}

// 保存配置文件自动任务
function saveConfigFileAutoTask() {
	if($("#formUploadAutoTaskConfigFile input[name='uploadFile']")[0].files.length == 0) {
		$.messager.alert("<%=rb.getString("TiShi")%>", "<%=rb.getString("QingXianXuanZeWenJian")%>");
		return false;
	}
	
	<%-- $("#formUploadAutoTaskConfigFile").form('submit', {
        url: "${ctx}/cell/selfstart/saveConfigFileTask.action",
		dataType: 'json',
        onSubmit: function(param){
			var bool = checkParams(param);
			if(!bool) return false;
        },
        success: function (data) {
	     	if(typeof data == 'string') {
	     		var data = JSON.parse(data);
	     	}
            if (data["success"]) {
            	closeWinEditConfigFileAutoTask();
            	reloadAutoTaskList();
            } else {
            	$.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
            }
        }
    }); --%>
    uploadWithProgress({
    	url: "${ctx}/cell/selfstart/saveConfigFileTask.action",
    	form: document.querySelector("#formUploadAutoTaskConfigFile"),
    	successfunction (data) {
            if (data["success"]) {
            	closeWinEditConfigFileAutoTask();
            	reloadAutoTaskList();
            } else {
            	$.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
            }
        }
    });
}
</script>