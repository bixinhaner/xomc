<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style>
	.tabsTitle{
	border:none;
	}
</style>
<%-- 周期性上报基站日志文件 --%>
<div style="width: 100%;height: 100%;min-width min-height: 700px;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="north" data-options="height:50,border:false,split:false">
			<div class="tabsTitle" style='height:40px;line-height:40px;'>
				<span tabtit='resetConfig' class="active"><%=rb.getString("ZhouQiShangBaoRiZhi")%></span>	
			</div>
		</div>

		<div region="center" data-options="border:false" >
	                <div class="easyui-layout" data-options="border:false,fit:true" style="padding-top: 10px">
	                    <div region="north" data-options="border:false,height:45" style="line-height:26px;overflow: hidden;">
	                    	<div class="queryGroup">
	                    		<input id="searchCellTex" type="text" class="border border-box"  placeholder="<%=rb.getString("XiaoZhanBianMa")%>">
	                    		<b class='el-icon el-icon-common-search' onclick="$('#periodLogFileTaskDatagrid').datagrid('reload');"></b>
	                    	</div>
	                    	<div class="linkbuttonGroup" style="float:right;margin-right:13px">
	                    		<a onclick="openPeriodCollectWin()" class="linkbutton">
	                        		<span><%=rb.getString("ZhouQiShangBao")%></span>
	                       		 </a>
	                       		 <a onclick="stopPeriodReportLog()" class="linkbutton">
	                        		<span><%=rb.getString("TingZhiShangBao")%></span>
	                        	</a>
	                        	<a onclick="downloadPeriodLogFile()" class="linkbutton" id="periodLogPatchDownloadBtn">
	                        		<span><%=rb.getString("PiLiangXiaZai")%></span>
	                        	</a>
	                    		<a onclick="delPeriodLogFile()" class="linkbutton"  id="periodLogPatchDelBtn">
									<span><%=rb.getString("PiLiangQingChu")%></span>
								</a>                               
	                    	</div>
							
	                        <input type="hidden" id="periodReportLogTaskCellCode">
	                    </div>
	                    <div region="center" data-options="border:false">
	                        <table id="periodLogFileTaskDatagrid" class="easyui-datagrid"
	                           data-options="
	                           border:false,
	                           url: '${ctx}/cell/collect/getPeriodReportLogTaskList.action',
	                           fit:true,
	                           checkbox:true,
	                           striped: true,
	                           pagination: true,
	                           pagePosition: 'bottom',
	                           onBeforeLoad: beforeLoadPeriodLogFileTaskFn,
	                           onSelectAll: changeQueryCollectLogListFn,
	                           onUnselectAll: changeQueryCollectLogListFn,
	                           onSelect: queryPeriodCollectLogListFn,
	                           onUnselect: changeQueryCollectLogListFn,
	                           onLoadSuccess:datagridLoadSuccess,
	                           idField: 'small_cell_code',
	                           rownumbers: true,
	                           fitColumns: true">
	                            <thead>
		                            <tr>
		                                <th data-options="field:'ck',checkbox:true"></th>
		                                <th data-options="field:'small_cell_code',hidden:true" ><%=rb.getString("XiaoZhanBianMa")%></th>
		                                <th data-options="field:'serial_number'" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
		                                <th data-options="field:'file_num'" width="50"><%=rb.getString("WenJianShuLiang")%></th>
		                                <th data-options="field:'report_period'" width="50"><%=rb.getString("FenZhongZhouQi")%></th>
		                                <th data-options="field:'task_status',formatter: periodTaskStatusFmt" width="100"><%=rb.getString("JinDu")%></th>
		                                <th data-options="field:'update_time'" width="60"><%=rb.getString("GengXinShiJian")%></th>
		                                <th data-options="field:'operation',formatter: periodCollectLogTaskOperFormatter,fixed:true" width="70"><%=rb.getString("CaoZuo")%></th>
		                            </tr>
	                            </thead>
	                        </table>
	                    </div>
	                </div>
		        </div>
		        <div region="south" data-options="height:300,border:false,split:false" style="background-color: #F3F3F4;margin-top:15px">
					<div class="easyui-layout" 
					   data-options="border:false,fit:true">
		       		 	<div region="north" style="height: 55px" data-options="border:false" >
						   <div class="tabsTitle" style='height:40px;line-height:40px;'>
								<span tabtit='resetConfig' class="active"><%=rb.getString("ZhouQiShangBaoWenJian")%></span>	
							</div>
					  	</div>
			        	<div region="center" data-options="border:false" >
						   	  <table id="periodLogFileListDataGrid" class="easyui-datagrid"
		                          data-options="border:false,
		                           fit:true,
		                           url:'${ctx}/cell/collect/getPeriodReportLogFileByCellCode.action',
		                           striped: true,
		                           rownumbers: true,
		                           singleSelect: true,
		                           fitColumns: true,
		                           onBeforeLoad: beforeTaskLoadPeriodLogFileFn,onLoadSuccess:datagridLoadSuccess">
		                            <thead>
			                            <tr>
			                                <th data-options="field:'small_cell_code'" hidden="true"><%=rb.getString("XiaoZhanBianMa")%></th>
			                                <th data-options="field:'file_name'" width="200"><%=rb.getString("WenJianMing")%></th>
			                                <th data-options="field:'upload_time'" width="100"><%=rb.getString("ShangBaoShiJian")%></th>
			                                <th data-options="field:'operation',formatter: periodCollectLogFileOperFormatter,fixed:true" width="150"><%=rb.getString("CaoZuo")%></th>
			                            </tr>
		                            </thead>
		                        </table>
		                  </div>
					</div>
		       </div>
			
	    
	</div>
</div>
<%-- 窗口-提示正在收集 --%>
<%--no open <div id="winCollectionLogPro" title="<%=rb.getString("TiShi")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:70,resizable:false,closable:false">
    <img src="${ctx}/js/jquery-easyui/themes/bootstrap/images/loading.gif"/>
    <span style="margin-left:75px;"><%=rb.getString("ZhengZaiShouJi")%></span>
</div> --%>

<%-- 窗口-周期上报日志文件--%>
<%-- <div id="winPeriodCollect" class="easyui-window" title="<%=rb.getString("ZhouQiShangBao")%>"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:860,height:600,resizable:false">
</div> --%>

<%-- 表单-下载单个小站周期性上报日志文件--%>
<form id="formExportPeriodReportLogFileByTask" style="display:none" method="post"
      action="${ctx}/cell/collect/doDownloadPeriodReportLogFileByTask.action">
    <%-- //已选中的小站的编码，提交表单之前为此值赋值 --%>
    <input id="periodCollectLogTimeZone" name="timeZone" type="hidden" value="">
    <input id="periodCollectLogCellCode" name="periodCollectLogCellCode" type="hidden" value="">
</form>

<%-- 表单-下载小站周期性上报单个日志文件 --%>
<form id="formExportPeriodReportLogFile" style="display:none" method="post" action="${ctx}/cell/collect/doDownloadPeriodReportLogFile.action">
    <%-- //已选中的小站的编码，提交表单之前为此值赋值 --%>
    <input id="periodCollectLogFileName" name="periodCollectLogFileName" type="hidden" value="">
</form>

<%-- 窗口-日志文件查看 --%>
<%-- <div id="winViewPeriodCollectLogFile" class="easyui-window" title="<%=rb.getString("WenJianXinXi")%>"
	 data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,resizable:false">
	 <div class="easyui-layout" data-options="border:false, fit:true">
	 	<div region="west" data-options="border:false,width:260,split:true,collapsible:false,maxWidth:500,minWidth:150">
	        <div class="easyui-panel" data-options="border:true,fit:true" style="padding: 0 15px;">
				<table id="periodCollectLogFileList"></table>
			</div>
	    </div>
	 	<div region="center" data-options="border:false" style="padding: 10px">
	 		<textarea id="periodCollectLogFileContent" class="border border-box" style="padding-left:10px;border-style:solid;width: 99%;height: 99%;resize: none;"></textarea>
	 	</div>
	 </div>
</div> --%>

<script type="text/javascript">
    $(function(){
    	closeLoading();
    });
    
    <%--点击周期上报按钮--%>
    function openPeriodCollectWin() {
      	/* $("#winPeriodCollect").window({
    		width: 860,
        	height: document.body.clientHeight * 0.9
        }).window("center").window("open");
    	$("#winPeriodCollect").window("refresh", "${ctx}/cell/collect/toSetCellPeriodLogPage.action"); */
    	var url = "${ctx}/cell/collect/toSetCellPeriodLogPage.action";
    	openDefaultWindow(url,{
    		title: '<%=rb.getString("ZhouQiShangBao")%>',
    		width: 860,
        	height: document.body.clientHeight * 0.9
    	});
    }
    
    //下载周期上报文件列表中的单个文件
    function downloadPeriodLogFileByFile(smallCellCode, fileName) {
    	//判断文件是否存在
        $.post("${ctx}/cell/collect/isPeriodLogFileExist.action", {"fileName" : fileName}, function (data) {
            if (data["success"]) {
            	$("#periodCollectLogFileName").val(fileName);
            	/* $("#formExportPeriodReportLogFile").form('submit',{
            		onSubmit: function(param){
            			var bool = checkParams(param)
            			if(!bool) return false;
            		}
            	}); */
            	exportByForm($("#formExportPeriodReportLogFile").attr("action"),{
            		periodCollectLogFileName: fileName
            	});
            } else {
           	 	$.messager.alert(TiShi, "<%=rb.getString("WenJianBuCunZai")%>");
                return;
            }
        }, "json");
    }
  
    //下载一个设备上报的多个日志文件
    function downloadPeriodLogFileByTask(smallCellCode, fileNum) {
    	$("#periodLogFileTaskDatagrid").datagrid("clearSelections");
	   
	    //如果没有日志文件，提示没有文件
	    if (fileNum == 0) {
	   	 	$.messager.alert(TiShi, "<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>");
	        return;
	    }

        $.post("${ctx}/cell/collect/isTaskPeriodLogFileExist.action", {"cellCodes": smallCellCode}, function (data) {
            if (data["success"]) {
            	$("#periodCollectLogTimeZone").val(timeZone);
            	$("#periodCollectLogCellCode").val(smallCellCode);
            	/* $("#formExportPeriodReportLogFileByTask").form('submit',{
            		onSubmit: function(param){
            			var bool = checkParams(param)
            			if(!bool) return false;
            		}
            	}); */
            	exportByForm($("#formExportPeriodReportLogFileByTask").attr("action"),{
            		timeZone: timeZone,
            		periodCollectLogCellCode: smallCellCode
            	})
            } else {
           	 	$.messager.alert(TiShi, "<%=rb.getString("WenJianBuCunZai")%>");
                return;
            }
        }, "json");
    }
    
    //点击批量下载按钮-批量下载多个设备上报的日志文件
    function downloadPeriodLogFile() {
    	var logFileSelect = $("#periodLogFileTaskDatagrid").datagrid("getSelections");
	    if ( 0 == logFileSelect.length) {
	        $.messager.alert(TiShi, "<%=rb.getString("QingXuanZeRenWu")%>");
	        return;
	    }
	   
	    var selecteTaskLength = logFileSelect.length;
	    var smallCellStr = "";
	    for (var count = 0; count < selecteTaskLength; count++) {
	    	var logTask = logFileSelect[count];
	    	 //如果没有日志文件，提示没有文件
		    var fileNum = logTask.file_num;
		    if (fileNum != 0) {
		    	smallCellStr += logTask.small_cell_code + ",";
		    }
	    }
	   
	    if (smallCellStr.length == 0) {
	    	 $.messager.alert(TiShi, "<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>");
		     return;
	    }
	    smallCellStr = smallCellStr.substring(0, smallCellStr.length - 1);
	    
        $.post("${ctx}/cell/collect/isTaskPeriodLogFileExist.action", {"cellCodes": smallCellStr}, function (data) {
        	 if (data["success"]) {
        		 $("#periodCollectLogTimeZone").val(timeZone);
            	$("#periodCollectLogCellCode").val(smallCellStr);
            	/* $("#formExportPeriodReportLogFileByTask").form('submit',{
            		onSubmit: function(param){
            			var bool = checkParams(param)
            			if(!bool) return false;
            		}
            	}); */

            	exportByForm($("#formExportPeriodReportLogFileByTask").attr("action"),{
            		timeZone: timeZone,
            		periodCollectLogCellCode: smallCellStr
            	})
            } else {
           	 	$.messager.alert(TiShi, "<%=rb.getString("WenJianBuCunZai")%>");
                return;
            }
        }, "json");
    }
  
    //删除周期上报文件列表中的单个文件
    function delPeriodLogFileByFile(smallCellCode, fileName) {
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuWenJian")%>", function (r) {
	       if (r) {
	           var params = {
	               "fileName" : fileName,
	               "cellCode" : smallCellCode
	           };
	           $.post("${ctx}/cell/collect/doClearPeriodReportLogFile.action", params, function (data) {
	               if (data["success"]) {
	               	   $("#periodLogFileTaskDatagrid").datagrid("reload");
	                   $("#periodLogFileTaskDatagrid").datagrid("selectRecord", smallCellCode);
	               } else {
	              	   $.messager.alert(TiShi, "<%=rb.getString("WenJianBuCunZai")%>");
	              	   $("#periodLogFileTaskDatagrid").datagrid("reload");
	                   $("#periodLogFileTaskDatagrid").datagrid("selectRecord", smallCellCode);
	                   return;
	               }
	           }, "json");
	       }
	   }).addClass("seriousConfirm");
    }
    
    //删除一个设备上报的日志文件
    function delPeriodLogFileByTask(smallCellCode, fileNum) {
    	$("#periodLogFileTaskDatagrid").datagrid("clearSelections");
	    //如果没有日志文件，提示没有文件
	    if (fileNum == 0) {
	   	 $.messager.alert(TiShi, "<%=rb.getString("MeiYouYaoShanChuDeWenJian")%>");
	        return;
	    }
	    
	    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuWenJian")%>", function (r) {
	        if (r) {
	            var params = {
	                "cellCodes": smallCellCode
	            };
	            $.post("${ctx}/cell/collect/doClearPeriodLogFileByTask.action", params, function (data) {
	                if (data["success"]) {
	                	$("#periodLogFileTaskDatagrid").datagrid("reload");
	                    $("#periodLogFileTaskDatagrid").datagrid("selectRecord", smallCellCode);
	                } 
	            }, "json");
	        }
	    }).addClass("seriousConfirm");
    }
    //点击批量清除按钮-批量清除多个设备上报的日志文件
    function delPeriodLogFile() {
	  	var logFileSelections = $("#periodLogFileTaskDatagrid").datagrid("getSelections");
	    if (logFileSelections.length == 0) {
	        $.messager.alert(TiShi, "<%=rb.getString("QingXuanZeRenWu")%>");
	        return;
	    }
	    
	    var selecteTaskLength = logFileSelections.length;
	    var smallCellStr = "";
	    for (var count = 0; count < selecteTaskLength; count++) {
	    	var logTask = logFileSelections[count];
	    	 //如果没有日志文件，提示没有文件
		    var fileNum = logTask.file_num;
		    if (fileNum != 0) {
		    	smallCellStr += logTask.small_cell_code + ",";
		    }
	    }
	   
	    if (smallCellStr.length == 0) {
	    	 $.messager.alert(TiShi, "<%=rb.getString("MeiYouYaoShanChuDeWenJian")%>");
		     return;
	    }
	    smallCellStr = smallCellStr.substring(0, smallCellStr.length - 1);
	   
	    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuWenJian")%>", function (r) {
	       if (r) {
	    	   var params = {
	    		   "cellCodes" : smallCellStr
	    	   }
	           $.post("${ctx}/cell/collect/doClearPeriodLogFileByTask.action", params, function (data) {
	                if (data["success"]) {
	                	$("#periodLogFileTaskDatagrid").datagrid("reload");
	                	$("#periodLogFileTaskDatagrid").datagrid("selectRecord", $("#periodReportLogTaskCellCode").val());
	                }
	           }, "json");
	        }
	    }).addClass("seriousConfirm");
    }

    //点击停止上报按钮
    function stopPeriodReportLog() {
    	
    	var logFileSelections = $("#periodLogFileTaskDatagrid").datagrid("getSelections");
	    if (logFileSelections.length == 0) {
	        $.messager.alert(TiShi, "<%=rb.getString("QingXuanZeRenWu")%>");
	        return;
	    }
	    
	    //刷新前台数据
	    $("#periodLogFileTaskDatagrid").datagrid("reload");
	  
	    var selecteTaskLength = logFileSelections.length;
	    var cellCodes = "";
	    for (var count = 0; count < selecteTaskLength; count++) {
	    	var logTask = logFileSelections[count];
	    	var taskStatus = logTask.task_status;
	    	var smallCellCode = logTask.small_cell_code;
	    	var serialNumber = logTask.serial_number;
	    	
	    	var promptMessage = serialNumber;
	    	if ("${localeIsZh} == 0") {
	    		promptMessage += " ";
	    	}
	    	  //查看基站是否正在设置周期上报操作
	    	if (taskStatus == 0) {
	    		$.messager.alert(TiShi, promptMessage + "<%=rb.getString("ZhouQiShangBaoSheZhiMeiWanChengTiShi")%>");
		        return;
	    	}
	    	
	    	//查看基站是否正在设置停止上报操作
	    	if (taskStatus == 4) {
	    		$.messager.alert(TiShi, promptMessage + "<%=rb.getString("TingZhiShangBaoSheZhiMeiWanChengTiShi")%>");
		        return;
	    		
	    	}
	    	 //如果没有日志文件，提示没有文件
		    cellCodes += smallCellCode + ",";
	    }
	    cellCodes = cellCodes.substring(0, cellCodes.length - 1);
	    
	   $.post("${ctx}/cell/collect/terminatePeriodReportLogFile.action", {"cellCodes" : cellCodes}, function (data) {
            if (data["success"]) {
            	$("#periodLogFileTaskDatagrid").datagrid("reload");
            }
       }, "json");
    }
    
    // 加载前事件-基站周期上报日志任务列表
    function beforeLoadPeriodLogFileTaskFn(param) {
    	param["timeZone"] = timeZone;
    	var search_text = $("#searchCellTex").val();
    	if(search_text != ""){
    		param["search_text"] = search_text;
    	}
    }
    
    // 加载前事件-基站 周期 收集日志任务关联的文件列表
    function beforeTaskLoadPeriodLogFileFn(param) {
        param["timeZone"] = timeZone;
    }
    
    // 全选、取消全选、取消选中  任务，获取第一个收集文件列表
    function changeQueryCollectLogListFn() {
        var selectDevice = $("#periodLogFileTaskDatagrid").datagrid('getSelections');
        if (selectDevice.length >= 1) {
            var param = {
                timeZone : timeZone,
                cellCode : selectDevice[0]["small_cell_code"]
            }
            $("#periodLogFileListDataGrid").datagrid('load', param);
        } else {
            $('#periodLogFileListDataGrid').datagrid('loadData', {total : 0, rows : []});
        }
    }
  
    //选中设备日志事件，获取关联的周期上报文件列表
    function queryPeriodCollectLogListFn(index, row) {
    	var cellCode = row.small_cell_code;
    	$("#periodReportLogTaskCellCode").val(cellCode);
    	var param = {
                timeZone : timeZone,
                cellCode : cellCode
            }
        $("#periodLogFileListDataGrid").datagrid('load', param);
    }
    
    // 任务进度格式化
    function periodTaskStatusFmt(value, rowData, rowIndex) {
    	//任务状态0-定制未开始，1-正在定制上报，2-定制上报成功，3-定制上报失败，4-停止上报未开始，5-正在停止上报，6-停止上报成功，7-停止上报失败
    	if (value == "0") { 
    		return "<%=rb.getString("ZhouQiShangBaoSheZhiWeiKaiShi")%>";
    	} else if (value == "1") {
    		return "<%=rb.getString("ZhouQiShangBaoZhengZaiSheZhi")%>";
    	} else if (value == "2") {
    		return "<%=rb.getString("ZhouQiShangBaoSheZhiChengGong")%>";
    	} else if (value == "3") {
    		return "<%=rb.getString("ZhouQiShangBaoSheZhiShiBai")%>";
    	} else if (value == "4") {
    		return "<%=rb.getString("TingZhiShangBaoSheZhiWeiKaiShi")%>";
    	} else if (value == "5") {
    		return "<%=rb.getString("TingZhiShangBaoZhengZaiSheZhi")%>";
    	} else if (value == "6") {
    		return "<%=rb.getString("TingZhiShangBaoSheZhiChengGong")%>";
    	} else if (value == "7") {
    		return "<%=rb.getString("TingZhiShangBaoSheZhiShiBai")%>";
    	} else if (value == "8") {
    		return "<%=rb.getString("ZhouQiShangBaoSheZhiChengGongCQSX")%>";
    	} 
    }
 
    /**
     * 格式化设备日志列表操作列
     */
    function periodCollectLogTaskOperFormatter(value, rowData, rowIndex){
    	var task_progress = rowData.progress_detail;
    	var small_cell_code = rowData.small_cell_code;
    	
    	var XiaZai = '<%=rb.getString("XiaZai")%>';
    	var ShanChu = '<%=rb.getString("ShanChu")%>';
    	
    	value = "";
    	value = value + "<div class='operationDiv operation_download' title='"+XiaZai+"' onclick='downloadPeriodLogFileByTask(\"" + small_cell_code + "\",\"" + rowData.file_num + "\")'></div>";
    	value = value + "<div class='operationDiv operation_delete' title='"+ShanChu+"' style='margin-left:15px;' onclick='delPeriodLogFileByTask(\"" + small_cell_code + "\",\"" + rowData.file_num + "\")'></div>";
    	return value;
    }
    
    /**
     * 格式化周期上报文件列表操作列
     */
    function periodCollectLogFileOperFormatter(value, rowData, rowIndex){
    	var task_progress = rowData.progress_detail;
    	var small_cell_code = rowData.small_cell_code;
    	var file_name = rowData.file_name;
    	
    	var XiaZai = '<%=rb.getString("XiaZai")%>';
    	var ShanChu = '<%=rb.getString("ShanChu")%>';
    	var ChaKan = '<%=rb.getString("ChaKan")%>';
    	
    	var gridData = JSON.stringify(rowData);
    	
    	value = "";
    	value = value + "<div class='operationDiv operation_view' title='"+ChaKan+"'  onclick='viewPeriodLogFileByFile(" + gridData + ")'></div>";
    	value = value + "<div class='operationDiv operation_download' title='"+XiaZai+"' style='margin-left:15px;' onclick='downloadPeriodLogFileByFile(\"" + small_cell_code + "\",\"" + file_name + "\")'></div>";
    	value = value + "<div class='operationDiv operation_delete' title='"+ShanChu+"' style='margin-left:15px;' onclick='delPeriodLogFileByFile(\"" + small_cell_code + "\",\"" + file_name + "\")'></div>";
    	return value;
    }
    
    

  //点击查看上报的日志文件
  function viewPeriodLogFileByFile(rowData) {
  	$.messager.progress({
  		title : "<%=rb.getString("QingDengDai")%>",
  		text : "<%=rb.getString("WenJian")%>" + "<%=rb.getString("JieXiZhong")%>"
  	});
  	
  	var params={
  			"smallCellCode": rowData.small_cell_code,
  			"fileName": rowData.file_name
  		};
  	$.post("${ctx}/cell/collect/doUnZipPeriodLogFile.action", params, function (data) {
  		$.messager.progress("close");
           if (data.length > 0) {
          	 <%-- $("#periodCollectLogFileList").datagrid({
        			singleSelect : true,
        			fit : true,
        			fitColumns : true,
        			border : false,
        			striped: false,
        			columns: [[
        				{field: 'small_cell_code', hidden: true},
        				{field: 'file_name', hidden: true},
        				{field: 'un_file_path', hidden: true},
        				{field: 'un_file_name', width: 200, title: '<%=rb.getString("WenJianLieBiao")%>'}
        			]],
        			onSelect: function(index,row) {
        				viewPeriodLogFileContent(row);
        			},
        			onLoadSuccess:datagridLoadSuccess
        		});
          	$("#periodCollectLogFileList").datagrid('loadData', data);
          	
          	$("#winViewPeriodCollectLogFile").window({
          		 width: 900,
          		 height: 600,
          	});
          	$("#periodCollectLogFileContent").val("");
           	$("#winViewPeriodCollectLogFile").window("center").window("open"); --%>
           	var url = "${ctx}/cell/collect/toLogWinPage.action?code=view_pre";
           	openDefaultWindow(url,{
           		title: '<%=rb.getString("WenJianXinXi")%>',
           		width: 900,
         		height: 600,
         		onLoad: function(){
         			$("#periodCollectLogFileList").datagrid({
            			singleSelect : true,
            			fit : true,
            			fitColumns : true,
            			border : false,
            			striped: false,
            			columns: [[
            				{field: 'small_cell_code', hidden: true},
            				{field: 'file_name', hidden: true},
            				{field: 'un_file_path', hidden: true},
            				{field: 'un_file_name', width: 200, title: '<%=rb.getString("WenJianLieBiao")%>'}
            			]],
            			onSelect: function(index,row) {
            				viewPeriodLogFileContent(row);
            			},
            			onLoadSuccess:datagridLoadSuccess
            		});
              		$("#periodCollectLogFileList").datagrid('loadData', data);
         			$("#periodCollectLogFileContent").val("");
         		}
           	}); 
           } else {
          	$.messager.alert(TiShi, "<%=rb.getString("WenJianBuCunZai")%>");
              return;
           }
       }, "json");
  }
  	//点击查看上报的日志文件内容信息
  function viewPeriodLogFileContent(rowData) {
  	$.messager.progress({
  			title : "<%=rb.getString("QingDengDai")%>",
  			text : "<%=rb.getString("WenJian")%>" + "<%=rb.getString("JieXiZhong")%>"
  		});
  	$("#periodCollectLogFileContent").val("");
  	
  	var params={fileName:rowData.file_name,unFilePath:rowData.un_file_path};
  		$.post("${ctx}/cell/collect/viewUnZipImmedLogFile.action", params, function (data) {
  			$.messager.progress("close");
  			if (data.success) {
  				if(data.message==""){
  					$.messager.alert(TiShi, "<%=rb.getString("WenJianNeiRongWeiKong")%>");
  				}else{
  					$("#periodCollectLogFileContent").val(data.message);
  				}
          } else {
         		$.messager.alert(TiShi, "<%=rb.getString("WenJianBuCunZai")%>");
          	return;
          }
      }, "json");
  }
  	
  function queryPeriodLogTaskByCellCode () {
	var param = {};
  	param["timeZone"] = timeZone;
	var search_text = $("#searchCellTex").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
	  $('#periodLogFileTaskDatagrid').datagrid('load', param);
  }
</script>

