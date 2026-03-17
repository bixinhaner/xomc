<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<%-- 周期性上报基站日志文件 --%>
<div style="width: 100%;height: 100%;min-width:1000px; min-height: 700px;">
	<div class="easyui-layout" data-options="border:false,fit:true">
	    <div region="west" data-options="border:true,split:true,collapsible:false,maxWidth:500,minWidth:350" title="<%=rb.getString("JiZhanLieBiao")%>">
	        <div class="easyui-layout" data-options="border:false,fit:true">
		        <div region="north" data-options="border:false,height:45" id="toolbar_gridCell_period_collection">
				    <ul class="inputslist" style="padding: 10px 20px;">
				        <li>
				            <select id="stationConfigDeviceGroup" name="type" class="border border-box" style="width:100px;height: 26px;"></select>
				        </li>
				        <li class="serial_number input_li">
				            <input name='value' type="text" class="border border-box" style="margin-left:5px;" placeholder="<%=rb.getString("XiaoZhanBianMa")%>">
				        </li>
				        <li>
				            <a onclick="$('#gridCell_period_collection').datagrid('load');" style="margin-left:15px;" class="easyui-linkbutton"><%=rb.getString("SouSuo")%></a>
				        </li>
				        <li>
				            <a onclick="openConfirmPeriodCollectLogWin();" style="margin-left:15px;" class="easyui-linkbutton"><%=rb.getString("ZhouQiShangBao")%></a>
				        </li>
				    </ul>
				</div>
				<div region="center" data-options="border:false">
		            <table id="gridCell_period_collection" style="padding-bottom: 10px"></table>
		        </div>
	        </div>
	    </div>
	    <div region="center" data-options="border:false" style="margin-left: 10px;">
	    	<div class="easyui-layout" data-options="border:false,fit:true">
		        <div region="center" data-options="border:true" title="<%=rb.getString("ZhouQiShangBaoRiZhi")%>" >
	                <div class="easyui-layout" data-options="border:false,fit:true" style="padding-top: 10px">
	                    <div region="north" data-options="border:false,height:46" style="line-height:26px;padding:20px 20px 0px 0px;overflow: hidden;">
	                    	<input id="searchCellTex" type="text" class="border border-box" style="margin-left:10px;" placeholder="<%=rb.getString("XiaoZhanBianMa")%>">
	                    	<a onclick="$('#periodLogFileTaskDatagrid').datagrid('reload');" class="easyui-linkbutton" style="vertical-align:middle;margin-left: 15px;"><%=rb.getString("ChaXun")%></a>
							<a onclick="delPeriodLogFile()" class="easyui-linkbutton" style="float:right;" id="periodLogPatchDelBtn">
								<%=rb.getString("PiLiangQingChu")%>
							</a>
	                        <a onclick="downloadPeriodLogFile()" class="easyui-linkbutton" style="margin-right: 15px;float:right;" id="periodLogPatchDownloadBtn">
	                        	<%=rb.getString("PiLiangXiaZai")%>
	                        </a>
	                        <a onclick="stopPeriodReportLog()" class="easyui-linkbutton" style="margin-right: 15px;float:right;" id="periodLogStopReportBtn">
	                        	<%=rb.getString("TingZhiShangBao")%>
	                        </a>
	                        <input type="hidden" id="periodReportLogTaskCellCode">
	                    </div>
	                    <div region="center" data-options="border:false" style="padding: 20px 15px 0px 15px;">
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
		        <div region="south" data-options="height:300,border:false,split:false" style="background-color: #F3F3F4;padding-top:15px">
					<div class="easyui-panel" title="<%=rb.getString("ZhouQiShangBaoWenJian")%>" 
					   data-options="border:true,fit:true">
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
		                                <th data-options="field:'operation',formatter: periodCollectLogFileOperFormatter,fixed:true" width="160"><%=rb.getString("CaoZuo")%></th>
		                            </tr>
	                            </thead>
	                        </table>
					</div>
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

<%-- 窗口-日志-确认收集 --%>
<%-- <div id="winConfirmPeriodCollect" class="easyui-window" title="<%=rb.getString("RiZhiZhouQiShangBao")%>"
	 data-options="modal:true,closed:true,minimizable:false,maximizable:false,
	 	collapsible:false,width: 600,height:750,inline:true,onBeforeClose: winconfirmPeriodCollectCloseFn">
	 <div class="easyui-layout" data-options="border:false, fit:true">
	 	<div region="north" data-options="border:false,height:500" style="padding: 15px 20px;">
	    	<table id="confirmPeriodCollectLogDg" class="easyui-datagrid"
	 			data-options="fit:true,
							singleSelect: false,
							striped: true,
							border:true,
							pagination:false, 
							idField:'small_cell_code',
							title:'<%=rb.getString("JiZhanLieBiao")%>',
							rownumbers:true,
							fitColumns:true,onLoadSuccess:datagridLoadSuccess">
				<thead>
					<tr>
						<th data-options="field:'small_cell_code',hidden:true"></th>
						<th data-options="field:'serial_number'" width="450"><%=rb.getString("XiaoZhanBianMa")%></th>
						<th data-options="field:'host_name'" width="450"><%=rb.getString("HostName")%></th>
					</tr>
				</thead>
	 		</table>
	 	</div>
	 	<div region="center" data-options="border:false">
			<div class="easyui-panel" data-options="fit:true,border:false" style="padding: 0px 20px">
				<div class="easyui-layout" data-options="fit:true,border:false">
					<div region="center" data-options="border:true,title:'<%=rb.getString("ShangBaoZhouQi")%>'">
						<div style="margin-top: 20px;margin-left: 20px;">
							<label style="width: 90px; display: inline-block" class="borderBoxClass"><%=rb.getString("FenZhongZhouQi")%><%=rb.getString("MaoHao")%></label>
							<select id="periodReportLogMin" class="border border-box borderBoxClass">
								<option value="900" selected="selected">15</option>
								<option value="1800">30</option>
								<option value="2700">45</option>
								<option value="3600">60</option>
							</select>
						</div>
						<div style="margin-top: 20px;margin-left: 20px;">
							<label style="width: 90px; display: inline-block" class="borderBoxClass"> <%=rb.getString("KaiShiShiJian")%><%=rb.getString("MaoHao")%></label>
							<input id="periodReportLogStartTime" class="easyui-datetimebox border border-box borderBoxClass" style="vertical-align: middle; height: 26px">
							<label style="width: 90px; display: inline-block; margin-left: 10px" class="borderBoxClass"><%=rb.getString("JieShuShiJian")%><%=rb.getString("MaoHao")%></label>
							<input id="periodReportLogEndTime" class="easyui-datetimebox border border-box borderBoxClass" style="vertical-align: middle; height: 26px">
						</div>
					</div>
				</div>
		   </div>
	 	</div>
	 	<div region="south" data-options="border:false,height:46" style="padding: 10px 20px 10px 0px;">
	    	<a onclick="cancelPeriodReportLogFile();" style="float:right;" class="easyui-linkbutton"><%=rb.getString("QuXiao")%></a>
	    	<a onclick="confirmPeriodReportLogFile();" style="float:right;margin-right:15px;" class="easyui-linkbutton"><%=rb.getString("QueDing")%></a>
	 	</div>
	 </div>
</div> --%>

<%-- 窗口-日志文件内容查看 --%>
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


<script type="text/javascript">
	var choosedGroupId = -1;
    $(function(){
    	closeLoading();
    	//初始化基站列表筛选条件
    	initCellGrid_QueryTemplateCondition();
    	
    	$("#gridCell_period_collection").datagrid({
    		url: "${ctx}/system/device/enodeb/queryENBInfoPageList.action",
    		queryParams : {like_fields:"serial_number"},
    		singleSelect:false,
    		fit:true,
    		fitColumns:true,
    		border:false,
    		rownumbers:true,
    		pagePosition:'bottom',
    		pageSize : 100,
    		pageList : [100],
    		idField:'small_cell_code',
    		pagination : true,
    		striped: true,
    		onLoadSuccess:datagridLoadSuccess,
    		onLoadError:datagridLoadError,
    		onBeforeLoad:beforeLoad_gridCell_period_collection,
    		columns: [[
    			{field: 'ck', checkbox: true},
    			{field: 'small_cell_code', hidden: true},
    			{field: 'connection_status',sortable:true,fixed:true,width: 30,formatter:connStatusFormatter},
    			{field: 'serial_number',sortable:true,width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
    			{field: 'host_name',sortable:true,width: 100, title: '<%=rb.getString("HostName")%>'}
    		]]
    	});
    	
    	$("#gridCell_period_collection").datagrid("getPager").pagination({
    		layout:['prev','manual','next','refresh']
    	});
    });
    
    <%-- 初始化基站列表--%>
    function initCellGrid_QueryTemplateCondition() {
    	<%-- 基站搜索-回车事件 --%>
        $("#toolbar_gridCell_period_collection input[name='value']").bind("keyup", function (e) {
            if (e.keyCode == 13) {
                $("#gridCell_period_collection").datagrid("reload");
            }
        });
        
        <%-- 基站搜索-软件版本-下拉面板 --%>
    	/* $("#softwareVersionCombo_period_collection").combobox({
    		valueField: "value",
    		textField: "text",
    		panelWidth: 200
    	}); */
    	
    	<%-- 基站搜索-类型-change事件 --%>
    	/* $("#toolbar_gridCell_period_collection select[name='type']").bind("change", function(e) {
    		var type = e.target.value;
    		$("#toolbar_gridCell_period_collection .input_li").hide();
    		$("#toolbar_gridCell_period_collection ." + type).show();
    		
    		if (type == "software_version") {
    			$("#softwareVersionCombo_period_collection").combobox({
    				url: "${ctx}/cell/cpeinfos/getSoftwareVersionList.action"
    			});
    		}
    	}); */
    	
        /* 基站搜索-设备分组-下拉面板  zss*/
	   	 $("#stationConfigDeviceGroup").combobox({
	   	    	url: "${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action",
	   	        width: 100,
	   	        panelWidth: 150,
	   	        panelHeight: 200,
	   	        valueField: 'id',
	   	        textField: 'group_name',
	   	        editable: false,
	   	     	onSelect: function(data){
	   	     		choosedGroupId = data.id;
	   	     	 	$("#gridCell_period_collection").datagrid("reload");	
	   	     	}
	   	    });
	   	$("#stationConfigDeviceGroup").combobox('setValues',['<%=rb.getString("SheBeiZu")%>','<%=rb.getString("SheBeiZu")%>']);
    }
    
    // 加载前事件-基站列表
    function beforeLoad_gridCell_period_collection(param) {
    	if(choosedGroupId >0 ){
    		param["group_id"] = choosedGroupId;
    	}
    	var search_text = $(".inputslist li.serial_number>input").val();
    	if (search_text != ""){
    		param["search_text"] = search_text;
    	}
    	/* var type = $("#toolbar_gridCell_period_collection select[name='type']").val();
    	var val = $("#toolbar_gridCell_period_collection ." + type + " input[name='value']").val();
    	param[type] = val; */
    }
    
    <%--点击收集按钮--%>
    function openConfirmPeriodCollectLogWin() {
    	$("#gridCell_period_collection").datagrid("reload");
    	// 获取选择的基站
        var selCells = $("#gridCell_period_collection").datagrid("getSelections");
        if (selCells.length == 0 || null == selCells) {
            $.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
            return;
        }
        
	    //刷新设置日志列表数据
	    $("#periodLogFileTaskDatagrid").datagrid("reload");
	    
        var jsonstr = "[";
        var cellCodes = "";
        for (var cellCount = 0; cellCount < selCells.length; cellCount++) {
        	var smallCellCode = selCells[cellCount]["small_cell_code"];
        	var cellTaskRowIndex = $("#periodLogFileTaskDatagrid").datagrid("getRowIndex", smallCellCode);
        	//判断要设置周期上报的站是否正在设置上报操作中
        	if (cellTaskRowIndex > -1) {
        		var cellTaskRow = $("#periodLogFileTaskDatagrid").datagrid("getData").rows[cellTaskRowIndex];
        		var taskStatus = cellTaskRow.task_status;
    	    	
        		//根据中英文浏览器设置不同的提示内容
    	    	var promptMessage = smallCellCode;
    	    	if ("${localeIsZh} == 0") {
    	    		promptMessage += " ";
    	    	}
    	    	  //查看基站是否正在设置周期上报操作
    	    	if ( taskStatus == 0) {
    	    		$.messager.alert(TiShi, promptMessage + "<%=rb.getString("ZhouQiShangBaoSheZhiMeiWanChengTiShi")%>");
    		        return;
    	    	}
    	    	//查看基站是否正在设置停止上报操作
    	    	if ( taskStatus == 4) {
    	    		$.messager.alert(TiShi, promptMessage + "<%=rb.getString("TingZhiShangBaoSheZhiMeiWanChengTiShi")%>");
    		        return;
    	    	}
        	}
        	cellCodes += selCells[cellCount]["small_cell_code"] + ",";
        	jsonstr += "{'small_cell_code': '" + selCells[cellCount]["small_cell_code"] 
        	        + "','serial_number': '" + selCells[cellCount]["serial_number"]
        	        + "','host_name': '" + selCells[cellCount]["host_name"] + "'},";
        }
        
        jsonstr = jsonstr.substring(0, jsonstr.length -1);
        jsonstr += "]";
	    
        /* $("#winConfirmPeriodCollect").window("center").window("open");
        $("#confirmPeriodCollectLogDg").datagrid('loadData', eval('(' + jsonstr + ')')); */
        var url = "${ctx}/cell/collect/toLogWinPage.action?code=confirm_pre";
       	openDefaultWindow(url,{
       		title: '<%=rb.getString("RiZhiZhouQiShangBao")%>',
       		width: 600,height:750,
     		onLoad: function(){
     			$("#confirmPeriodCollectLogDg").datagrid('loadData', eval('(' + jsonstr + ')'));
     		}
       	}); 
    }
    
    //点击确认收集页面的确定按钮
    function confirmPeriodReportLogFile() {
    	 var selCells = $("#gridCell_period_collection").datagrid("getSelections");
    	 var cellCodes = "";
    	 var serialNumbers = "";
         for (var cellCount = 0; cellCount < selCells.length; cellCount++) {
        	 cellCodes += selCells[cellCount]["small_cell_code"] + ",";
        	 serialNumbers += selCells[cellCount]["serial_number"] + ",";
         }
         cellCodes = cellCodes.substring(0, cellCodes.length - 1);
         serialNumbers = serialNumbers.substring(0, serialNumbers.length - 1);
         
         var periodMin = $("#periodReportLogMin").val();
         var startTime = $("#periodReportLogStartTime").datetimebox("getValue");
         var endTime = $("#periodReportLogEndTime").datetimebox("getValue");
       	 //开始时间不能为空
         if (startTime.length == 0) {
        	$.messager.alert(TiShi, "<%=rb.getString("KaiShiShiJianBuNengKong")%>");
     		return false;
         }
       
         //结束时间不能为空
         if (endTime.length == 0) {
        	$.messager.alert(TiShi, "<%=rb.getString("JieShuShiJianBuNengKong")%>");
     		return false;
         }
         //结束时间不能早于当前时间
         if (new Date(gloableTime) > dateParser(endTime).getTime()) {
        	 $.messager.alert(TiShi, "<%=rb.getString("JieShuZaoYuDangQianShiJian")%>");
             return false;
         }
         //开始时间不能晚于结束时间
         var validTimeResult = validateStartAndStopTime(startTime, endTime);
         if ("false" == validTimeResult) {
             $.messager.alert(TiShi, "<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>");
             return false;
         }
         
         var param = {
        		 timeZone:timeZone,
        		 cellCodes:cellCodes,
        		 serialNumbers:serialNumbers,
        		 reportPeriod:periodMin,
        		 startTime:startTime,
        		 endTime:endTime
         };
       	 $.post("${ctx}/cell/collect/customizePeriodReportLogFile.action", param, function (data) {
       		/* $("#winConfirmPeriodCollect").window("close"); */
       		
       		if (data["success"]) {
       			closeDefaultWindow();
           		$("#periodLogFileTaskDatagrid").datagrid("reload");
       		}else{
       			if(data["responseCode"] == '901'){
    				$.messager.alert(TiShi, '<%=rb.getString("CiPanKongJianBuZu")%>');
    			}
       		}
         }, "json"); 
    }
    
    <%--验证开始时间和结束时间是否合法--%>
    function validateStartAndStopTime(startTimeStr, endTimeStr){
        if (null != startTimeStr && null != endTimeStr) {
            var startDate = dateParser(startTimeStr);
            var endDate = dateParser(endTimeStr);
            if (startDate.getTime() < endDate.getTime()) {
                return "true";
            }
        }
        return "false";
    }
    
    //点击确认收集页面的确定按钮
    function cancelPeriodReportLogFile() {
    	/* $("#winConfirmPeriodCollect").window("close"); */
    	closeDefaultWindow();
    	return;
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
            		periodCollectLogFileName: 
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
            	});
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
            	});
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
	   });
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
	    });
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
	    });
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
    	value = value + "<div class='el-icon el-icon-operation-download' title='"+XiaZai+"' onclick='downloadPeriodLogFileByTask(\"" + small_cell_code + "\",\"" + rowData.file_num + "\")'></div>";
    	value = value + "<div class='grid-del-btn-div' title='"+ShanChu+"' style='margin-left:15px;' onclick='delPeriodLogFileByTask(\"" + small_cell_code + "\",\"" + rowData.file_num + "\")'></div>";
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
    	value = value + "<div class='el-icon el-icon-operation-view' title='"+ChaKan+"' style='margin-left:15px;' onclick='viewPeriodLogFileByFile(" + gridData + ")'></div>";
    	value = value + "<div class='el-icon el-icon-operation-download' title='"+XiaZai+"' style='margin-left:15px;' onclick='downloadPeriodLogFileByFile(\"" + small_cell_code + "\",\"" + file_name + "\")'></div>";
    	value = value + "<div class='grid-del-btn-div' title='"+ShanChu+"' style='margin-left:15px;' onclick='delPeriodLogFileByFile(\"" + small_cell_code + "\",\"" + file_name + "\")'></div>";
    	return value;
    }
    
    //确认周期性上报窗口关闭事件
    function winconfirmPeriodCollectCloseFn() {
    	$("#periodReportLogEndTime").datetimebox("setValue","");
    	$("#periodReportLogStartTime").datetimebox("setValue","");
    	$("#periodReportLogMin").val("900");
    }
    
    //点击查看上报的日志文件
    function viewPeriodLogFileByFile(rowData) {
    	$.messager.progress({
    		title : "<%=rb.getString("QingDengDai")%>",
    		text : "<%=rb.getString("WenJian")%>" + "<%=rb.getString("JieXiZhong")%>"
    	});
    	
    	var params={"smallCellCode": rowData.small_cell_code,"fileName":rowData.file_name};
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
</script>

