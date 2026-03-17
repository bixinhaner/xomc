<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<%-- 收集页面：收集NRM文件 --%>
<div class="easyui-layout" data-options="border:false,fit:true">
    <div region="west" data-options="border:false,width:430,split:true,maxWidth:430,minWidth:250">
    	<div class="easyui-panel" data-options="fit:true,border:true" style="padding: 0 10px;">
    	<div data-options="border:false,height: 40,collapsible:false" >
		        <div class="omcPageTitleDiv" style="padding:0 10px 0 10px">
					<ul class="omcPageTitleContainer">
						<li class="default"><%=rb.getString("JiZhanSheBei")%></li>
					</ul>
				</div>
	    </div>
		<table id="gridCell_nrmCollection"></table>
		</div>
    </div>
    <div region="center" data-options="border:false" style="padding-left: 15px;background-color: #F3F3F4;">
    	<div class="easyui-panel" data-options="border:true,fit:true">
    		<div data-options="border:false,height: 40,collapsible:false" >
		        <div class="omcPageTitleDiv">
					<ul class="omcPageTitleContainer">
						<li class="default"><%=rb.getString("NRMXinXi")%></li>
					</ul>
				</div>
		    </div>
	        <div class="easyui-layout" data-options="border:false,fit:true">
	            <div region="north" data-options="border:false,height:90" style="padding:20px;">
	                <a onclick="showConfigWindow()" id="cfgClear"  style="margin-left:20px;" class="linkbutton"><span><%=rb.getString("SheZhi")%></span></a>
	            </div>
	            <div region="center" data-options="border:false,fit:true">
	                <div class="" data-options="fit:true,border:false">
	                    <div data-options="border:false">
	                    	<div data-options="region:'north',border:false,height: 40,collapsible:false" >
						        <div class="omcPageTitleDiv">
									<ul class="omcPageTitleContainer">
										<li class="default"><%=rb.getString("PeiZhiWenJian")%></li>
									</ul>
								</div>
						    </div>
	                        <div class="easyui-layout" data-options="border:false,fit:true">
	                            <div region="north" data-options="border:false,height:66" style="padding:20px 0px 0px 0px;">
	                                <div class="linkbuttonGroup" style="float:right;margin-right:20px;">
	                                	<a onclick="collectFile()" id="cfgCollect" class="linkbutton"><span><%=rb.getString("KaiShiShouJi")%></span></a>
	                                	<a onclick="downloadFiles()" id="cfgDownload" class="linkbutton"><span><%=rb.getString("XiaZai")%></span></a>
	                                	<a onclick="clearConifgFile()" id="cfgClear" class="linkbutton"><span><%=rb.getString("QingChuWenJian")%></span></a>
	                                </div>  
	                            </div>
	                            <div region="center" data-options="border:false,fit:true" style="padding: 0 15px;">
	                                <table id="ConfigFileListDatagrid" class="easyui-datagrid"
	                                       data-options="border:false,url:'${ctx}/cell/collect/nrm/getNRMFileList.action',fit:true,checkbox:true,
								                striped: true,pagination: true,pagePosition: 'bottom',rownumbers: true,fitColumns: true,onLoadSuccess:datagridLoadSuccess">
	                                    <thead>
	                                    <tr>
	                                    <th data-options="field:'ck',checkbox:true"></th>
	                                    <th data-options="field:'fileName'" width="100"><%=rb.getString("WenJianMing")%></th>
	                                    <th data-options="field:'filePath'" hidden=""></th>
	                                    <th data-options="field:'modifyTime'" width="100"><%=rb.getString("XiuGaiShiJian")%></th>
	                                    </tr>
	                                    </thead>
	                                </table>
	                            </div>
	                        </div> 
	                    </div>
	                </div>
	            </div>
	        </div>
		</div>
    </div>
</div>
<%-- 窗口-提示正在收集 --%>
<div id="winCollectionNRMFilePro" title="<%=rb.getString("TiShi")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:70,resizable:false,closable:false">
    <img src="${ctx}/js/jquery-easyui/themes/bootstrap/images/loading.gif"/>
    <span style="margin-left:75px;"><%=rb.getString("ZhengZaiShouJi")%></span>
</div>
<%-- 表单-导出小站配置文件和日志文件 --%>
<form id="formExportConfigFile" style="display:none" method="post"
      action="${ctx}/cell/collect/nrm/downloadNRMFiles.action">
    <%-- //已选中的小站的编码，提交表单之前为此值赋值 --%>
    <input id="filePathStr" name="filePathStr" type="hidden" value="">
</form>
<%-- 工具栏-基站列表 --%>
<div id="toolbar_GridCell_nrm" class="admin_query_head" style="padding: 20px 0px;margin-left:20px;">
	<ul class="" style="display:inline-block;margin-right:20px;">
		<li>
			<select id="toolbar_GridCell_nrm_type" name="type" class="easyui-combobox border border-box" style="height:26px;width:100px;margin-left:5px;">
				<option value="serial_number"><%=rb.getString("XiaoZhanBianMa")%></option>
				<option value="host_name"><%=rb.getString("HostName")%></option>
				<option value="software_version"><%=rb.getString("SoftwareVersion")%></option>
				<option value="group_id"><%=rb.getString("SheBeiZu")%></option>
			</select>
		</li>
		<!-- <li class="serial_number input_li">
			<input name="value" type="text" class="border border-box" style="height:26px;margin-left:5px;">
		</li> -->
		<li class="host_name input_li" style="display: none;">
			<input name="value" type="text" class="border border-box" style="height:26px;margin-left:5px;">
		</li>
		<li class="software_version input_li" style="display: none;">
			<input id="softwareVersionCombo_nrm" name="value" class="border border-box" style="height:26px;margin-left:5px;">
		</li>
		<li class="group_id input_li" style="display: none;">
			<input id="regnTreeCombo_nrm" name="value" class="border border-box" style="height:26px;margin-left:5px;">
		</li>
		<%-- <li>
			<a onclick="$('#gridCell_nrmCollection').datagrid('reload');" style="float:left;margin-left:15px;"
					class="easyui-linkbutton"><%=rb.getString("SouSuo")%></a>
		</li> --%>
	</ul>
	<div class="queryGroup serial_number">
		<input name="value" type="text" style="width:200px;"/>
		<b onclick="$('#gridCell_nrmCollection').datagrid('reload');"></b>
	</div>
</div>

<%-- 窗口-北向接口设置 --%>
<div id="winNRMConfig" class="easyui-window" title="<%=rb.getString("BeiXiangJieKou")%>"
	 data-options="modal:true,closed:true,minimizable:false,maximizable:false,collapsible:false,width:350,height:300,inline:false">
</div>

<script type="text/javascript">
$(function(){
	closeLoading();
	<%-- 基站搜索-回车事件 --%>
    $("#toolbar_GridCell_nrm input[name='value']").bind("keyup", function (e) {
        if (e.keyCode == 13) {
            $("#gridCell_nrmCollection").datagrid("reload");
        }
    });
    
    <%-- 基站搜索-软件版本-下拉面板 --%>
	$("#softwareVersionCombo_nrm").combobox({
		valueField: "value",
		textField: "text",
		panelWidth: 200
	});
	
	<%-- 基站搜索-类型-change事件 --%>
	$("#toolbar_GridCell_nrm select[name='type']").bind("change", function(e) {
		var type = e.target.value;
		$("#toolbar_GridCell_nrm .input_li").hide();
		$("#toolbar_GridCell_nrm ." + type).show();
		
		if (type == "software_version") {
			$("#softwareVersionCombo_nrm").combobox({
				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
			});
		}
	});
	
	<%-- 基站搜索-设备分组-下拉面板 --%>
   	$("#regnTreeCombo_nrm").combotree({
   		url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
   		panelWidth: 200
   	});
   	
   	$(".group_id input.textbox-text").css("padding", "0 10px").css("margin", "0");
   	
   	$("#gridCell_nrmCollection").datagrid({
   		url: '${ctx}/cell/cpeinfos/queryCpeInfosList.action?forSelect=1',
   		queryParams:{like_fields:"serial_number,host_name"},
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		pagePosition:'bottom',
		idField:'small_cell_code',
		toolbar:'#toolbar_GridCell_nrm',
		onBeforeLoad: beforeLoad_gridCell_nrmCollection,
		onLoadSuccess:datagridLoadSuccess,
		onLoadError:datagridLoadError,
		pagination : true,
		striped: true,
		columns: [[
			{field: 'ck', checkbox: true},
			{field: 'small_cell_code', hidden: true},
			{field: 'connection_status',sortable:true,fixed:true,width: 30,formatter:connStatusFormatter},
			{field: 'serial_number',sortable:true,width: 80, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
			{field: 'host_name',sortable:true,width: 100, title: '<%=rb.getString("HostName")%>'}
		]]
   	});
   	
   	$("#gridCell_nrmCollection").datagrid("getPager").pagination({
		layout:['prev','manual','next','refresh']
	});
   	
   	$("#ConfigFileListDatagrid").datagrid({
        queryParams : {
            timeZone : timeZone
        }
    });
});

function beforeLoad_gridCell_nrmCollection(param) {
	try {
		var type = $("#toolbar_GridCell_nrm_type").combobox("getValue");
		var val = $("#toolbar_GridCell_nrm ." + type + " input[name='value']").val();
		param[type] = val;
		
	} catch (e) {}
}

<%--点击Clear按钮，清除选中文件--%>
function clearConifgFile() {
    var ConfigFileSelections = $("#ConfigFileListDatagrid").datagrid("getSelections");
    if ((0 == ConfigFileSelections.length || null == ConfigFileSelections)) {
        $.messager.alert(TiShi, "<%=rb.getString("QingXianXuanZeWenJian")%>");
        return;
    }
    var filePathStr = "";
    for (var LogFileCount = 0; LogFileCount < ConfigFileSelections.length; LogFileCount++) {
        filePathStr += ConfigFileSelections[LogFileCount].fileName + ",";
    }
    filePathStr = filePathStr.substring(0, filePathStr.length - 1);

    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuWenJian")%>", function (r) {
        if (r) {
            var params = {
                "filePathStr": filePathStr
            };
            $.post("${ctx}/cell/collect/nrm/doClearConfigFilesTableNodes.action", params, function (data) {
                if (data["success"]) {
                    $("#ConfigFileListDatagrid").datagrid("clearSelections");
                    $("#ConfigFileListDatagrid").datagrid("reload");
                }
            }, "json");
        }
    });
}

<%--点击Download按钮，下载文件到本地--%>
function downloadFiles() {
	var filePathStr = "";
	var configFileChecked = $("#ConfigFileListDatagrid").datagrid("getSelections");
    if (0 == configFileChecked.length || null == configFileChecked) {
        $.messager.alert(TiShi, "<%=rb.getString("QingXianXuanZeWenJian")%>");
        return;
    }
    for (var cfFileCount = 0; cfFileCount < configFileChecked.length; cfFileCount++) {
        filePathStr += configFileChecked[cfFileCount].filePath + ",";
    }
  
    filePathStr = filePathStr.substring(0, filePathStr.length - 1);
    $("#filePathStr").val(filePathStr);
    /* $("#formExportConfigFile").form('submit', {
        onSubmit: function(param) {
			var bool = checkParams(param);
			if(!bool) return false;
            param.timeZone=timeZone;
        }
    }); */
    exportByForm('${ctx}/cell/collect/nrm/downloadNRMFiles.action',{
    	filePathStr: filePathStr
    });
}

<%--点击Collect按钮--%>
function collectFile() {
	$("#gridCell_nrmCollection").datagrid("reload");
	// 获取选择的基站
    var selCells = $("#gridCell_nrmCollection").datagrid("getSelections");
    if (selCells.length == 0 || null == selCells) {
        $.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
        return;
    }
    var cellCodes = "";
    for (var cellCount = 0; cellCount < selCells.length; cellCount++) {
    	var connectionStatus = selCells[cellCount]["connection_status"];
    	if (connectionStatus == 'On'||connectionStatus == 'updating' ||connectionStatus == 'Exception') {
			cellCodes += selCells[cellCount]["small_cell_code"] + ",";
    	}
    }
    if (cellCodes.length > 0) {
    	cellCodes = cellCodes.substring(0, cellCodes.length - 1);
        var param = {
            "cellCodes": cellCodes
        };

        $.post("${ctx}/cell/collect/nrm/goCollectNRMFile.action", param, function (data) {
        	 $("#winCollectionNRMFilePro").window("open");
        }, "json");
    } else {
    	$.messager.alert(TiShi, "<%=rb.getString("JiZhanWeiLianJie")%>");
		return;
    }
}

function showConfigWindow() {
	$("#winNRMConfig").window("open");
	$("#winNRMConfig").window("refresh", "${ctx}/cell/collect/nrm/goNRMConfiguration.action");
}

//加载前事件-基站列表
function beforeLoad_gridCell_nrm(param) {
	var type = $("#toolbar_GridCell_nrm select[name='type']").val();
	var val = $("#toolbar_GridCell_nrm ." + type + " input[name='value']").val();
	param[type] = val;
}
</script>