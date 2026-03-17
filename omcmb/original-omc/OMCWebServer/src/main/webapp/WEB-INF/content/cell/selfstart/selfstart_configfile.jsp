<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
.fileinfo {
	height: 35px;
	line-height: 35px;
	display: inline-block;
	width: 300px;
}
</style>

<%-- 自启动下发配置文件 --%>
<div class="easyui-layout" data-options="border:false,fit:true">
	<div region="north" data-options="height:250,border:true,title:'<%=rb.getString("WenJianXinXi")%>',collapsible:false" style="padding: 20px;">
		<input type="hidden" id="switch_selfstart_configfile" value="${switch_}" />
		<div style="height: 30px;">
			<span><%=rb.getString("ShiFouQiYong")%><%=rb.getString("MaoHao")%></span>
			<input type="radio" id="open_selfstart_configfile" name="selfstart_configfile" value="1" style="vertical-align: bottom;" onclick="statusChange(this)">
			<label for="open_selfstart_configfile"><%=rb.getString("Shi")%></label>
			<input type="radio" id="close_selfstart_configfile" name="selfstart_configfile" value="0" style="vertical-align: bottom;" onclick="statusChange(this)">
			<label for="close_selfstart_configfile"><%=rb.getString("Fou")%></label>
		</div>
		
		<a class="easyui-linkbutton" onclick="openWinUploadConfigFile()"><%=rb.getString("ShangChuanWenJian")%></a>
		<div>
			<div class="fileinfo">
				<span><%=rb.getString("WenJianMing")%><%=rb.getString("MaoHao")%></span><span id="filename">${fileInfo.filename}</span>
			</div>
			<div class="fileinfo">
				<span><%=rb.getString("WenJianDaXiao")%><%=rb.getString("MaoHao")%></span><span id="filesize">${fileInfo.filesize}</span>
			</div>
		</div>
		<div>
			<div class="fileinfo">
				<span>MD5<%=rb.getString("MaoHao")%></span><span id="md5">${fileInfo.md5}</span>
			</div>
			<div class="fileinfo">
				<span><%=rb.getString("ShangChuanShiJian")%><%=rb.getString("MaoHao")%></span><span id="upload_time">${fileInfo.upload_time}</span>
			</div>
		</div>
	</div>
	<div region="center" data-options="border:false" style="padding-top: 15px;background-color: #F3F3F4;">
		<div class="easyui-panel" data-options="border:true,fit:true,title:'<%=rb.getString("PeiZhiShengJiJiLu")%>'" style="padding: 0 15px;">
			<table id="gridProgress_selfstart"></table>
		</div>
	</div>
</div>

<%-- 表单 - 上传文件 --%>
<form enctype="multipart/form-data" method="post" id="formUploadConfigFile_selfstart" style="display: none;">
	<input name="uploadFile" type="file" style="display: none;">
	<input name="cellCodes" type="hidden">
</form>

<script type="text/javascript">
$(function() {
	$("#formUploadConfigFile_selfstart input[name=uploadFile]").bind("change", function() {
		var tmpPath = this.value;
		if (tmpPath) {
			$("#formUploadConfigFile_selfstart").form("submit", {
				url: '${ctx}/cell/selfstart/uploadConfigFile.action',
				dataType: 'json',
				success: function(data) {
					if(typeof data == 'string') data = eval("(" + data + ")");
					$.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
					//$("#formUploadConfigFile_selfstart input[name=uploadFile]")[0].value = "";
					$('#mainpage').panel({
				        border: false,
				        href: '${ctx}/cell/selfstart/toSelfStartConfigFile.action'
				    });
				},
				onSubmit: function(param){
					var bool = checkParams(param);
					if(!bool) return false;
		        }
			});
		}
	});
	
	// 配置文件下发记录表格
	$("#gridProgress_selfstart").datagrid({
		url: '${ctx}/cell/selfstart/getConfigFileProgress.action',
		singleSelect : true,
		fit : true,
		fitColumns : true,
		border : false,
		striped: true,
		idField: 'small_cell_code',
		pagination: true,
		onLoadSuccess:datagridLoadSuccess,
		columns: [[
			{field: 'small_cell_code', width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'update_time', width: 100, title: '<%=rb.getString("GengXinShiJian")%>'},
			{field: 'detail', width: 200, title: '<%=rb.getString("JinDu")%>'}
		]]
	});
	
	// 开关赋值
	var switch_ = $("#switch_selfstart_configfile").val();
	if (switch_ == "1") {
		document.getElementById("open_selfstart_configfile").checked = true;
	} else {
		document.getElementById("close_selfstart_configfile").checked = true;
	}
})

// 打开上传文件窗口
function openWinUploadConfigFile() {
	$("#formUploadConfigFile_selfstart input[name=uploadFile]").click();
}

// 启用状态点击事件
function statusChange(e) {
	$.post("${ctx}/cell/selfstart/changeSelfstartConfigStatus.action", {status: e.value}, function(data) {
		$.messager.alert("<%=rb.getString("TiShi")%>", data["message"]);
	}, "json");
}
</script>