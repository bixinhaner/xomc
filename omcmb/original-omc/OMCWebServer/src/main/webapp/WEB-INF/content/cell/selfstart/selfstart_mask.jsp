<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<div class="easyui-layout" data-options="fit:true,border:false">
    <div region="north" data-options="border:false,height:48,border:false,collapsible:false" style="border-width: 0 0 1px 0;padding:10px;">
        <div style="border-width: 0px;display: inline-block;float: left">
            <form id="formDownloadSelfstartTemplate" style="display:none;" method="post" action="${ctx}/cell/selfstart/downloadExcelTemplate.action"></form>
            <form enctype="multipart/form-data" id="formUploadSelfstartTemplate" style="display:none;" method="post" action="${ctx}/cell/selfstart/uploadExcelTemplate.action">
              <input id="uploadFile" name="uploadFile" type="file"/>
            </form>
            <a onclick="downloadTemplate()" style="float:left;display: inline-block;"
               class="easyui-linkbutton"><%=rb.getString("GuiHuaDaoChu")%>
            </a>
            <a onclick="uploadTemplate()" style="float:left;margin-left:10px;display: inline-block;"
               class="easyui-linkbutton"><%=rb.getString("GuiHuaDaoRu")%>
            </a>
        </div>
    </div>
    <div region="center" data-options="border:false">
        <table class="easyui-datagrid" id="tableSelfStartParamList" fit="true" data-options="border:false,fitColumns:true,
	                  rownumbers:true,url:'${ctx}/cell/selfstart/querySelfstartParam.action',striped:true,
	                  pagination:true,pagePosition:'bottom',onRowContextMenu:showRowMenutableSelfstartList,onLoadError:datagridLoadError,onLoadSuccess:datagridLoadSuccess">
            <thead>
            <tr>
                <th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
                <th data-options="field:'CELL_ID'" width="50"><%=rb.getString("XIAOQUID")%></th>
                <th data-options="field:'PCI'" width="50"><%=rb.getString("PCIValue")%></th>
                <th data-options="field:'DL_EARFCN'" width="80"><%=rb.getString("XiaXingPinDian")%></th>
                <th data-options="field:'UL_EARFCN'" width="80"><%=rb.getString("ShangXingPinDian")%></th>
                <th data-options="field:'TAC'" width="50"><%=rb.getString("TAC")%></th>
                <th data-options="field:'PLMN'" width="100"><%=rb.getString("PLMN")%></th>
                <th data-options="field:'PA'" width="100"><%=rb.getString("PA")%></th>
                <th data-options="field:'PB'" width="50"><%=rb.getString("PB")%></th>
                <th data-options="field:'REFERENCESIGPOWER'" width="100"><%=rb.getString("CanKaoXinHaoQiangDu")%></th>
                <th data-options="field:'MME_ADDRESS'" width="80"><%=rb.getString("MMEDIZHI")%></th>
            </tr>
            </thead>
        </table>
    </div>
</div>

<%--右键菜单-小站列表（自配置参数）--%>
<div id="rowSelfstartParam" class="easyui-menu" style="width: 200px">
    <div onclick="clearParams()"><%=rb.getString("ShanChuCanShu")%></div>
</div>

<script type="text/javascript">
$(function () {
	<%--上传模板过程中，选取文件后确定后，上传表单--%>
	$("#uploadFile").bind("change",function(){
		if(document.getElementById("uploadFile").value==""){
			return;
		}

		$("#formUploadSelfstartTemplate").form("submit",{
			dataType: 'json',
			success:function(data){
				if(typeof data == 'string') data = eval("(" + data + ")");
				$.messager.alert("<%=rb.getString("ShangChuanMoBan")%>",data["message"]);
				<%--提交表单后，要清空表单中的excel表，以便以后再次提交--%>
				document.getElementById("uploadFile").value = "";
				$("#tableSelfStartParamList").datagrid("reload");
			},
			onSubmit: function(param){
				var bool = checkParams(param);
				if(!bool) return false;
	        }
		})
	})
})

<%--下载模板--%>
function downloadTemplate() {
	$("#formDownloadSelfstartTemplate").form("submit",{
		onSubmit: function(param){
			var bool = checkParams(param)
			if(!bool) return false;
        }
	});
}

<%--上传模板,弹出选择模板窗口--%>
function uploadTemplate() {
	$("#uploadFile").click();
}

<%--右键-小站列表（自配置参数）--%>
function showRowMenutableSelfstartList(e, rowIndex, rowData){
	e.preventDefault();
	$("#tableSelfStartParamList").datagrid("clearSelections");
	$("#tableSelfStartParamList").datagrid("selectRow", rowIndex);
	$("#rowSelfstartParam").menu("show",{
		left: e.clientX,
		top: e.clientY
	});
}

function clearParams(){
	var selCell = $("#tableSelfStartParamList").datagrid("getSelected");
	var param = {serial_number:selCell["SERIAL_NUMBER"]};
	$.post("${ctx}/cell/selfstart/clearParam.action", param, function (data) {
        $("#tableSelfStartParamList").datagrid("reload");
    }, "json");
}
</script>