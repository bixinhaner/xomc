//面板标题栏右上角操作按钮，鼠标悬停事件，显示操作按钮提示文字信息
function showTipText(ele){
	$(ele).prev().fadeToggle();
}
//鼠标悬停事件，显示操作按钮提示文字信息 --元素后面文字
function showTipTextNext(ele){
	$(ele).next().fadeToggle();
}
function sortTable(tb) {
	var mrTb = tb;
	if (mrTb && mrTb.length) {
	    var opts = mrTb.datagrid('options'),
	    	params = {
		        page: opts.pageNumber,
		        rows: opts.pageSize,
		        order: opts.sortOrder,
		        sort: opts.sortName
		    },
		idField = opts.idField;
	    if (opts.sortName) {
	    	$.extend(params, opts.queryParams);
	    	if (opts.onBeforeLoad) opts.onBeforeLoad(params);
			
	    	$.post(opts.url, params, function(data) {
	    		if (data && data.rows) {
	    			var rows = mrTb.datagrid('getRows'),
		            	srows = mrTb.datagrid('getSelections').map(function(item) {
		            		return item
		            	});
	    			var rowLength = rows.length;
		          
	    			/* start */
	    			var newLength = data.rows.length;
	    			if(rowLength <= newLength) {// 新数据多于历史数据，重叠部分直接更新，多出部分append
	    				for(var i=0;i<newLength;i++) {
	    					if(i<=rowLength) {
	    						mrTb.datagrid('updateRow', {index: i, row: data.rows[i]});
	    					}else {
	    						mrTb.datagrid('appendRow', data.rows[i]);
	    					}
	    				}
	    			}else { // 新数据少于历史数据，重叠部分更新，多出部分直接移除
	    				for(var i=0;i<rowLength;i++) {
	    					if(i<=newLength) {
	    						mrTb.datagrid('updateRow', {index: i, row: data.rows[i]});
	    					}else {
	    						mrTb.datagrid('deleteRow', i);
	    					}
	    				}
	    			}
	    			/* end */
		          
	    			srows.map(function(row) {
	    				mrTb.datagrid('selectRecord', row[idField]);
	    			});
	    			mrTb.datagrid('resize');
	    		}
	    	}, 'json');
	    }
	}
}
function setIntervalUpdate(id, p) {
	if(window[id+'_timer']) {
		clearInterval(window[id+'_timer']);
	}

	window[id+'_timer'] = setInterval(function() {
		var $tb = $('#'+id);
		if($tb.length) {
			updateTable($tb, p);
		}else {
			clearInterval(window[id+'_timer']);
		}
	},6000);
}
// update 方式更新table
function updateTable(tb,p) {
	var mrTb = tb;
	if (mrTb && mrTb.length) {
		var opts = mrTb.datagrid('options'),
	    	params = {
		        page: opts.pageNumber,
		        rows: opts.pageSize,
	      	},
	      	idField = opts.idField;
		if(opts.sortName){
			params.order = opts.sortOrder
			params.sort = opts.sortName
		}
	    // 合并所以查询条件
	    if(p) {
	      	$.extend(params, opts.queryParams, p);
	    }else {
	      	$.extend(params, opts.queryParams);
	    }

		if (opts.onBeforeLoad) opts.onBeforeLoad(params);

	    // 抓取数据更新table
	    $.post(opts.url, params, function(data) {
	      	if (data && data.rows) {
	      		var rows = mrTb.datagrid('getRows'),
	      			srows = mrTb.datagrid('getSelections').map(function(item) {
	      				return item
	      			});
	      		var rowLength = rows.length;
	          
	      		/* start */
	      		var newLength = data.rows.length;
	      		if(rowLength <= newLength) {// 新数据多于历史数据，重叠部分直接更新，多出部分append
	      			for(var i=0;i<newLength;i++) {
	      				if(i<rowLength) {
	      					mrTb.datagrid('updateRow', {index: i, row: data.rows[i]});
	      				}else {
	      					mrTb.datagrid('appendRow', data.rows[i]);
	      				}
	      			}
	      		}else { // 新数据少于历史数据，重叠部分更新，多出部分直接移除
	      			for(var i=0;i<rowLength;i++) {
	      				if(i<newLength) {
	      					mrTb.datagrid('updateRow', {index: i, row: data.rows[i]});
	      				}else {
	      					mrTb.datagrid('deleteRow', newLength);
	      				}
	      			}
	      		}
	      		/* end */
	          
	      		srows.map(function(row) {
	      			mrTb.datagrid('selectRecord', row[idField]);
	      		});
	      		mrTb.datagrid('resize');
	      	}
	    }, 'json');
	}
}
//update 方式更新table
function updateTableRows(tb,dataRows) {
	var mrTb = tb;
	if (mrTb && mrTb.length) {
	    var opts = mrTb.datagrid('options'),
	        idField = opts.idField;
	    
        if (dataRows) {
          var rows = mrTb.datagrid('getRows'),
	          srows = mrTb.datagrid('getSelections').map(function(item) {
	              return item
	          });
          
          /* start */
          dataRows.map(function(item){
        	  rows.map(function(row,idx){
        		  if(item[idField] == row[idField]) {
        			  mrTb.datagrid("updateRow", {
        				  index: idx,
        				  row: Object.assign(row,item)
					  });
        		  }
        	  })
          })
          /* end */
          
          srows.map(function(row) {
        	  mrTb.datagrid('selectRecord', row[idField]);
          });
          mrTb.datagrid('resize');
        }
	}
}

//update 方式更新table
function updateTableRows(tb,dataRows) {
	var mrTb = tb;
	if (mrTb && mrTb.length) {
	    var opts = mrTb.datagrid('options'),
	        idField = opts.idField;
	    
        if (dataRows) {
          var rows = mrTb.datagrid('getRows'),
	          srows = mrTb.datagrid('getSelections').map(function(item) {
	              return item
	          });
          
          /* start */
          dataRows.map(function(item){
        	  rows.map(function(row,idx){
        		  if(item[idField] == row[idField]) {
        			  mrTb.datagrid("updateRow", {
        				  index: idx,
        				  row: Object.assign(row,item)
					  });
        		  }
        	  })
          })
          /* end */
          
          srows.map(function(row) {
        	  mrTb.datagrid('selectRecord', row[idField]);
          });
          mrTb.datagrid('resize');
        }
	}
}
/* <%-- 客户端主动刷新告警数量，要根据现在在用的告警定制规则 --%> */
function refreshAliveAlarmCount() {

	// 实时更新告警查询数据
	if ($("#gridAlarm").length != 0) {
		var p = $("#gridAlarm").datagrid("options");
		var queryParamObj = p.queryParams;//$('#queryform').serializeJson();
		queryParamObj.timeZone = timeZone;
		queryParamObj["like_fields"] = 'alarm_serverity_value,alarm_identifier,alarm_name,ne_type,equip_info,event_type_value,deal_state,specific_problem';
		queryParamObj.rows = p['pageSize'];
		queryParamObj.page = p['pageNumber'];
		queryParamObj.order = p['sortOrder'];
		queryParamObj.sort = p['sortName'];
		$.post(webRootPath + '/cell/fault/queryFaultPageList.action',queryParamObj, function(data) {
			if(data){
				var addAlarmDate = data["rows"];// 待更新行数
				if (addAlarmDate) {
					var alarmGridData = $("#gridAlarm").datagrid("getData").rows;// 当前页行数
					if (alarmGridData.length == addAlarmDate.length) {
						// 如果当前页行数等于更新行数，Update更新
						for (var i = 0; i < addAlarmDate.length; i++) {
							$("#gridAlarm").datagrid("updateRow", {
								index : i,
								row : addAlarmDate[i]
							});
						}
					} else if (alarmGridData.length > addAlarmDate.length) {
						// 如果当前页行数大于待更新行数，Insert插入，并判断总行数是否大于默认行数
						for (var i = 0; i < addAlarmDate.length; i++) {
							$("#gridAlarm").datagrid("updateRow", {
								index : i,
								row : addAlarmDate[i]
							});
						}
						for (var j = alarmGridData.length - 1; j >= addAlarmDate.length; j--) {
							$("#gridAlarm").datagrid("deleteRow", j);
						}
					} else if (alarmGridData.length < addAlarmDate.length) {
						// 如果当前页行数小于待更新行数，Insert插入，并判断总行数是否大于默认行数
						for (var i = 0; i < addAlarmDate.length; i++) {
							$("#gridAlarm").datagrid("insertRow", {
								index : i,
								row : addAlarmDate[i]
							});
							if (alarmGridData.length > addAlarmDate.length) {
								$("#gridAlarm").datagrid("deleteRow",
									alarmGridData.length - 1);
								}
						}
					}
					var alarmsPageTotal = data["total"];
					var p1 = $("#gridAlarm").datagrid("getPager");
						p1.pagination("refresh", {
							total : alarmsPageTotal,
							pageSize : p['pageSize'],
							pageNumber : p['pageNumber']
						});
				}
			}
		}, "json");
	}
}

/*<%-- 显示“北向接口通信异常告警”的记录 --%>*/
function showItfnAlarm() {
    $("#winItfnAlarm").window("open");
    $("#winItfnAlarm").window("refresh", webRootPath + "/cell/fault/goItfnFault.action");
}
/*<%-- 刷新Log文件列表 --%>*/
function refreshLOGFileList() {
	if ($("#immeLogFileTaskDatagrid") != null) {
		//$("#immeLogFileTaskDatagrid").datagrid("reload");
	}
}
/*<%-- 刷新周期上报日志列表 --%>*/
function refreshPeriodReportLOGFileList() {
	if ($("#periodLogFileTaskDatagrid") != null) {
		$("#periodLogFileTaskDatagrid").datagrid("reload");  
	}
}
/*<%-- 刷新NRM文件列表 --%>*/
function refreshNRMFileList() {
	if ($("#ConfigFileListDatagrid") != null) {
		$("#winCollectionNRMFilePro").window("close");
		$("#ConfigFileListDatagrid").datagrid("reload");
	}
}
/*<%--刷新Uplink Interference Detection列表--%>*/
function refreshPRBDataList() {
	if ($("#prbDatagrid") != null) {
		$("#prbDatagrid").datagrid("reload");
	}
}
/*<%-- 刷新MR上报进度列表 --%>*/
function refreshMrReportTaskList() {
	if ($("#currentCustomizeMrReportTaskDatagrid") != null) {
		$("#currentCustomizeMrReportTaskDatagrid").datagrid("reload");  
	}
}
/*<%-- 刷新死机日志文件列表 --%>*/
function refreshRLFileList() {
	if ($("#deviceErrorLogFileDatagrid") != null) {
		$("#deviceErrorLogFileDatagrid").datagrid("reload");
	}
}
/*<%-- 刷新PCILOCK添加列表 --%>*/
function refreshPCIList() {
	if ($("#PCITaskList") != null) {
		$("#PCITaskList").datagrid("reload");
	}
}

/*<%-- 刷新CPEPCILOCK添加列表 --%>*/
function refreshCPEPCIList() {
	if ($("#CPEPCITaskList") != null) {
		$("#CPEPCITaskList").datagrid("reload");
	}
}
/*<%-- 刷新密码重置任务列表 --%>*/
function refreshPwdResetList() {
	if ($("#resetPwdTaskDatagrid") != null) {
		$("#resetPwdTaskDatagrid").datagrid("reload");
	}
}
/*<%-- 刷新自配置任务及查看详情列表 --%>*/
function refreshSelfConfigTaskList(task,record,nextStep) {
	if(task){
		if ($("#taskManagement_selfConfig").length != 0) {
			$("#taskManagement_selfConfig").datagrid("reload");
		}
	}
	if(record){
		if ($("#taskManagement_selfConfig_view").length != 0) {
			$("#taskManagement_selfConfig_view").datagrid('reload');
		}
	}
}
/*<%--提示资源超出阈值   已经存在专门的监控程序 弹出预警信息多余（不在使用）--%>*/
function showResourceOverRangeTip(overRangeInfo) {
	if (overRangeInfo != "") {
		if (isWinResourceOverrangeOpen == false) {
			isWinResourceOverrangeOpen = true;
			$.messager.alert('Prompt', overRangeInfo, "", function() {
				isWinResourceOverrangeOpen = false;
			});
		}
	}
}

/*<%-- 弹出窗口显示当前活动告警，只显示匹配用户正在使用的定制规则的告警，只显示某一严重程度的告警 --%>*/
function showCurrAliveAlarm(alarm_severity) {
	var title = "";
	if (alarm_severity == '31001') {
		title = "Critical Alarms";
	} else if (alarm_severity == '31002') {
		title = "Major Alarms";
	} else if (alarm_severity == '31003') {
		title = "Minor Alarms";
	} else if (alarm_severity == '31004') {
		title = "Warning Alarms";
	}
	
	try{
		eventAllBus.$emit("gomenupage","2002","",'2002',{alarm_severity:alarm_severity});
	}catch(e){}
}
/*<%-- 弹出窗口显示当前SAS告警 --%>*/
function showSASAlarm(){
	if($("#SAS_alarm").hasClass("sas_wrong")){
		$("#SAS_alarm").removeClass("sas_wrong");
		$("#SAS_alarm").addClass("sas_alarm");
	}
	$("#winSasAlarm").window("open");
	$("#winSasAlarm").window("refresh",webRootPath+"/cell/SAS/goSasInterface.action");
}
/*<%-- 如果用户当前正处于小站信息页面，则刷新小站信息列表 --%>*/
function refreshCellInfo(alarmDatas) {
	var tableHomeCellList = $("#tableHomeCellList");

	if(alarmDatas){
		var options = tableHomeCellList.datagrid('options'),
			tbDatas = tableHomeCellList.datagrid('getRows'),
			code = 'SMALL_CELL_CODE';
		$.each(alarmDatas,function(index,item){
			$.each(tbDatas,function(n,row){
				if(item[code] == row.small_cell_code){
					tableHomeCellList.datagrid("updateRow", {
						index: n,
						row: {
							alarm_count: item.ALARM_COUNT,
							alarm_serverity: item.ALARM_SERVERITY
						}
					});
				}
			});
		})
		sortTable(tableHomeCellList);
	}
}
function refreshSignalingTraceTable(traceDatas){
	var tableTraceList = $("#tableTraceList");
	if(traceDatas){
		var tbDatas = tableTraceList.datagrid('getRows'),
		    code = 'trace_id';
		$.each(traceDatas,function(index,item){
			$.each(tbDatas,function(n,row){
				if(item[code] == row.trace_id){
					tableTraceList.datagrid("updateRow",{
						index:n,
						row:{
							task_status:item.task_status,
							end_time:item.end_time
						}
					})
				}
			})
		})
	}
}
function refreshSignalingTraceInfoTable(traceId){
	if(traceId == traceInfoListId){
		if ($("#tableTraceViewList") != null) {
			$("#tableTraceViewList").datagrid("reload");  
		}
	}
	
}
/*<%--判断会话是否超时--%>*/
function checkSessionTime() {
	if (userSessionExpireTime != "null" && userSessionExpireTime != "") {
		//上次访问到现在的分钟数
		var sessionAlivePeriod = ((new Date(gloableTime)).getTime() - lastVisitedTime.getTime()) / 1000 / 60;
		//会话超时，退出登录
		if (sessionAlivePeriod > userSessionExpireTime && userSessionExpireTime != "0") {
			screenLock();
		}
	}
}

/* 判断未操作时间是否超时 */
function checkNoActionTime() {
	//计算未操作时间间隔
	var noActionTimePeriod = ((new Date(gloableTime)).getTime() - noActionTime.getTime()) / 1000 / 60;
	//超过设定时间，退出登录
	if (noActionTimePeriod > sessionTimeOutMin && userSessionExpireTime != "0") {
		if (noActionTimer) {
			window.clearInterval(noActionTimer); //清除定时执行函数
		}
		//退出登录
		logout(ctx);
	}
}
//接收后台的刷新用户会话超时时间请求
function refreshSessionExpireTime(qryMap) {
	userSessionExpireTime = qryMap.now_session_expire_time;
}

/*<%-- SAS HTTP通信错误通知 --%>*/
function refreshSASMatter(){
	if($("#SAS_alarm").hasClass("sas_alarm")){
		$("#SAS_alarm").removeClass("sas_alarm");
		$("#SAS_alarm").addClass("sas_wrong");
	}else{
		return;
	}
}

//refresh customize kpi report progress
function refreshCustomizeKPIReportProgress() {
	if ($("#customiz_progress_grid")) {
		//刷新进度列表
        $('#customiz_progress_grid').datagrid("reload");
	}
}

//信令追踪查看面板打开标识  信令追踪查看/关闭会改变此值
/*var isMonitorSignalingInfoPage = 0;*/
var iSInpprogresstraceId;

//定时刷新，查询当前未读告警以及告警声音
function getAlarmAlert(){
	$.ajax({
		url: webRootPath + "/fault/viewConfig/getUnreadAlarmAlter.action",
		dataType: 'json',
		type: 'post',
		success: function(data){
			if(data){
				// 更新首页未读告警状态
				if(data.isRinger == 'true'){
					$("#alarmMusic")[0].play();
				}
				if(data.isUnread == 'true'){
					sysMain.unreadFlag = false;
				}else{
					sysMain.unreadFlag = true;
				}
				
			}
			if(alarmAlertInterval){
	        	clearTimeout(alarmAlertInterval);
	        }
			alarmAlertInterval = setTimeout("getAlarmAlert()", 6000);
		},
		error: function(){
			if(alarmAlertInterval){
	        	clearTimeout(alarmAlertInterval);
	        }
			alarmAlertInterval = setTimeout("getAlarmAlert()", 6000);
		}
	});
}


//定时刷新 获取告警数量
function getAlarmCount(){
	$.ajax({
		url: webRootPath + "/msgPush/getAlarmMsg.action",
		dataType: 'json',
		type: 'post',
		success: function(data){
			if(data && data["alarmCount"]){
				// 更新首页告警数量
				var alarm_count = data["alarmCount"],
					critical = alarm_count['critical'],
					major = alarm_count['major'],
					minor = alarm_count['minor'],
					warning = alarm_count['warning'];
				
		        $("#spanCriticalCount,#criticalsNum").text(critical);
		        $("#spanMajorCount,#majorsNum").text(major);
		        $("#spanMinorCount,#minorsNum").text(minor);
		        $("#spanWarningCount,#warningsNum").text(warning);
			}
			if(alarmInterval){
	        	clearTimeout(alarmInterval);
	        }
			alarmInterval = setTimeout("getAlarmCount()", getMsgTimeAlarm);
		},
		error: function(res) {
			if(alarmInterval){
	        	clearTimeout(alarmInterval);
	        }
			alarmInterval = setTimeout("getAlarmCount()", getMsgTimeAlarm);
		}
	});
}
//定时执行，取回服务器端消息，并做相应处理
var traceInfoListId = "";//用于判断是否刷新信令追踪列表信息
function getMsg() {
	var param = {};
	param.timeZone=timeZone;
	var isMonitorPage = "0";
	if ($("#tableHomeCellList").length != 0 && isVisible(document.querySelector("#tableHomeCellList"))) {
		isMonitorPage = "1";
		param.isMonitorPage = isMonitorPage;
	}
	if ($("#gsmTableHomeCellList").length != 0 && isVisible(document.querySelector("#gsmTableHomeCellList"))) {
		isMonitorPage = "1";
		param.isMonitorPage = isMonitorPage;
	}
	if ($("#gnb_monitor_tb").length != 0 && isVisible(document.querySelector("#gnb_monitor_tb"))) {
		isMonitorPage = "0";
		param.isMonitorPage = isMonitorPage;
	}
	
	var isMonitorCpePage = "0";
	if ($("#tableHomeCpeList").length != 0 && isVisible(document.querySelector("#tableHomeCpeList"))) {
		isMonitorCpePage = "1";
		param.isMonitorCpePage = isMonitorCpePage;
	}
	var isMonitorLicensePage = "0";
	if($("#tableLicense").length!=0 && isVisible(document.querySelector("#tableLicense"))){
		isMonitorLicensePage = "1";
		param.isMonitorLicensePage = isMonitorLicensePage;
	}
	//基站-监控-设置-license
	var isMonitorSettingLicensePage = "0";
	if($("#featureListTable").length!=0 && isVisible(document.querySelector("#featureListTable"))){
		isMonitorSettingLicensePage = "1";
		param.isMonitorSettingLicensePage = isMonitorSettingLicensePage;
	}
	//gnb-监控-设置-license
    var gnbIsMonitorSettingLicensePage = "0";
    if($("#gnbLicenseTable").length!=0 && isVisible(document.querySelector("#gnbLicenseTable"))){
        gnbIsMonitorSettingLicensePage = "1";
        param.gnbIsMonitorSettingLicensePage = gnbIsMonitorSettingLicensePage;
    }
	//信令追踪任务列表
	var isMonitorSignalingTaskPage = "0";
	if($("#tableTraceList").length!=0 && isVisible(document.querySelector("#tableTraceList"))){
		isMonitorSignalingTaskPage = "1";
		param.isMonitorSignalingTaskPage = isMonitorSignalingTaskPage;
	}
	//信令追踪查看列表
	var isMonitorSignalingInfoPage = "0";
	if($("#tableTraceViewList").length!=0 && isVisible(document.querySelector("#tableTraceViewList"))){
		isMonitorSignalingInfoPage = "1";
		param.isMonitorSignalingInfoPage = isMonitorSignalingInfoPage;
		param.signalingTraceTaskId = iSInpprogresstraceId;
	}
	//自开站任务列表
	var isMonitorHalobAutoTaskPage = "0";
	if($("#taskManagement_selfConfig").length!=0 && isVisible(document.querySelector("#taskManagement_selfConfig"))){
		//console.log('lockFreshStr = ' + lockFreshFlag);
		//console.log('floatFreshStr = ' + floatFreshFlag);
		isMonitorHalobAutoTaskPage = "1";
		param.isMonitorHalobAutoTaskPage = isMonitorHalobAutoTaskPage;
	}
	//UPS监控
	var isMonitorUpsPage = "0";
	if($("#upsMonitorTable").length!=0 && isVisible(document.querySelector("#upsMonitorTable"))){
		isMonitorUpsPage = "1";
		param.isMonitorUpsPage = isMonitorUpsPage;
	}
	
	//测量维护页面
	var isMonitorCustomizePMPage = "0";
	if($("#kpi_customize_grid").length!=0 && isVisible(document.querySelector("#kpi_customize_grid"))){
		isMonitorCustomizePMPage = "1";
		param.isMonitorCustomizePMPage = isMonitorCustomizePMPage;
	}
	//分布式列表
	var isMonitorEUPage = "0";
	var isMonitorRUPage = "0";
	if(distributedVue != ''){
		if(distributedVue.$refs.euTable.getData().length != 0){
			isMonitorEUPage = "1";
			param.isMonitorEUPage = isMonitorEUPage;
		}
		if(distributedVue.$refs.ruTable.getData().length != 0){
			isMonitorRUPage = "1";
			param.isMonitorRUPage = isMonitorRUPage;
		}
	}
	// sas 
	var isMonitorSasENB = "0";
	var isMonitorSasCPE = "0";
	if($("#sasMonitorTable").length!=0 && isVisible(document.querySelector("#sasMonitorTable"))){
		var type = $("#searchDeViceType").combobox("getValue");
		if(type == 'eNB'){
			isMonitorSasENB = "1"
			param.isMonitorSasENB = isMonitorSasENB
		}else{
			isMonitorSasCPE = "1"
			param.isMonitorSasCPE = isMonitorSasCPE
		}
	}
	
	$.post(webRootPath + "/msgPush/getMsg.action", param, function(data) {
		var boradcastMsgList = data["broadcastMsgList"];
		var unicastMsgList = data["unicastMsgList"];
		var cellStatusChange = data["cellStatusChange"];
		var cpeStatusChange = data["cpeStatusChange"];
		var sasCPEStatusChange = data["sasCPEStatusChange"];
		var sasENBStatusChange = data["sasENBStatusChange"];
		var shuangJiBiaoZhi = data["dual_switch"];
		var operatorTime = data["operatorTime"]||'';
		var licenseCellStatusChange = data["licenseCellStatusChange"];
		var selfConfigTasklist_CellStatus = data["halobAutoStartCellStatusChange"];
	    var signalingTraceMonitor = data["signalingTaskFieldChange"];
		var signalingTraceInfoList = data["refreshSignalingInfoList"];
		var upsStatusChange = data["upsStatusChange"];
		var customizeStatusChange = data["customizeStatusChange"];
		var euStatusChange = data["euStatusChange"];
		var ruStatusChange = data["ruStatusChange"];
		var collectLogInfo = data["collectTr069LogInfo"];
		//基站-监控-设置-license
		var licenseStatusChange = data["licenseStatusChange"];
		
		
		gloableTime = operatorTime;
		// 广播消息
		if (boradcastMsgList) {
			if (boradcastMsgList.length != 0) {
				for (var i = 0; i < boradcastMsgList.length; i++) {
					onData(boradcastMsgList[i]);
				}
			}
		}
		
		// 广播消息
		if (unicastMsgList) {
			if (unicastMsgList.length != 0) {
				for (var i = 0; i < unicastMsgList.length; i++) {
					onData(unicastMsgList[i]);
				}
			}
		}
		
		if (operatorTime) {
			$("#curr_operator_div #curr_time_span").text(getTimeZoneStr(timeZone)+operatorTime.substring(0,16));
		}
          	//信令追踪的同步刷新
		if(signalingTraceMonitor){
			refreshSignalingTraceTable(signalingTraceMonitor)
		}
		if(signalingTraceInfoList){
			refreshSignalingTraceInfoTable(signalingTraceInfoList.signalingTraceTaskId);
		}
		// 基站状态变化信息
		if (cellStatusChange) {
			var tbdom = $("#tableHomeCellList");
			var gsmTbdom = $("#gsmTableHomeCellList");
			if(tbdom.length && isVisible(document.querySelector("#tableHomeCellList"))) {
				
				var gridRows = enbvm.allTableData;

				cellStatusChange.map(function(item){
					gridRows.map(function(row){
						if(item.small_cell_code == row.small_cell_code) {
							Object.assign(row, item);
						}
					});
				});

				if( !queryVue.timeLocked && !isOverlapped(document.querySelector('#tableHomeCellList')) ) enbvm.refreshList();
				//刷新基站监控 更新连接状态统计信息 填充状态栏
				if(isMonitorPage == "1"){
					try{
						refresh_cellStatusStatistics();
					}catch(e){ }
				}
			}
			if(gsmTbdom.length && isVisible(document.querySelector("#gsmTableHomeCellList"))) {
				
				var gridRows = gsmvm.tableData;

				cellStatusChange.map(function(item){
					gridRows.map(function(row){
						if(item.small_cell_code == row.small_cell_code) {
							Object.assign(row, item);
						}
					});
				});

				if( !queryGsmVue.timeLocked && !isOverlapped(document.querySelector('#gsmTableHomeCellList')) ) gsmvm.refreshList();
				//刷新基站监控 更新连接状态统计信息 填充状态栏
				if(isMonitorPage == "1"){
					try{
						gsmvm.refresh_cellStatusStatistics();
					}catch(e){
						
					}
					
				}
			}
		}
		// 自开站任务列表状态变化信息
		if (selfConfigTasklist_CellStatus) {
			var tbdom = $("#taskManagement_selfConfig");
			if(tbdom.length && isVisible(document.querySelector("#taskManagement_selfConfig"))) {
				var gridData = tbdom.datagrid("getData").rows;
				for (var i = 0; i < selfConfigTasklist_CellStatus.length; i++) {
					for (var j = 0; j < gridData.length; j++) {
						if (gridData[j].task_id == selfConfigTasklist_CellStatus[i].task_id) {
							tbdom.datagrid("updateRow", {
								index: j,
								row: selfConfigTasklist_CellStatus[i]
							});
						}
					}
				}
			}
		}
		
		//license文件列表更新
		if (licenseCellStatusChange && isMonitorLicensePage == "1") {
			var gridData = $("#tableLicense").datagrid("getData").rows;
			for (var i = 0; i < licenseCellStatusChange.length; i++) {
				for (var j = 0; j < gridData.length; j++) {
					if (gridData[j].serial_number == licenseCellStatusChange[i].serial_number) {
						if(gridData[j].connection_status != licenseCellStatusChange[i].connection_status && gridData[j].connection_status == "Exception"){
							$("#getCellInfoErrorTips").fadeOut(200);
						}
						delete licenseCellStatusChange[i].small_cell_code;
						$("#tableLicense").datagrid("updateRow", {
							index: j,
							row: licenseCellStatusChange[i]
						});
					}
				}
			}
		}
		
		//基站-监控-设置-license 列表更新
		if (licenseStatusChange && isMonitorSettingLicensePage == "1") {
			enbSettingLicenseVue.getLicenceInfo();
		}
		//5g license
		if (licenseStatusChange && gnbIsMonitorSettingLicensePage == "1") {
            gnbSetLicenseVue.licenseInit();
        }
		
		// CPE数据更新
		if (cpeStatusChange) {
			var tbdom = $("#tableHomeCpeList");
			if(tbdom.length && isVisible(document.querySelector("#tableHomeCpeList"))) {

				var gridRows = cpevm.$refs.list.getData();

				cpeStatusChange.map(function(item){
					gridRows.map(function(row){
						if(item.CPE_CODE == row.CPE_CODE) {
							Object.assign(row, item);
						}
					});
				});

				if( !queryCpeVue.timeLocked && !isOverlapped(document.querySelector('#tableHomeCpeList')) ) cpevm.refreshList();
				refresh_cpeStatusStatistics();
			}
		}
		// SAS CPE 数据更新 
		if(sasCPEStatusChange){
			updateTableRows($("#sasMonitorTable"),sasCPEStatusChange);
		}
		// SAS ENB 数据更新
		if(sasENBStatusChange){
			updateTableRows($("#sasMonitorTable"),sasENBStatusChange);
		}
		if (shuangJiBiaoZhi) {
			$("#dual_switch").html("主备服务："+shuangJiBiaoZhi);
		}
		if(isQb == 'true'){
			newAlarmRing();
		}
		// UPS数据更新
		if (upsStatusChange) {
			if(isMonitorUpsPage == "1"){
				var gridData = $("#upsMonitorTable").datagrid("getData").rows;
				for (var i = 0; i < upsStatusChange.length; i++) {
					for (var j = 0; j < gridData.length; j++) {
						if (gridData[j].ups_code == upsStatusChange[i].ups_code) {
							$("#upsMonitorTable").datagrid("updateRow", {
								index: j,
								row: upsStatusChange[i]
							});
						}
					}
				}			
			}
		}
		//分布式EU列表更新
		if(euStatusChange){
			distributedVue.$refs.euTable.refresh();
		}
		//分布式RU列表更新
		if(ruStatusChange){
			distributedVue.$refs.ruTable.refresh();
		}
		if(euStatusChange || ruStatusChange){
			distributedVue.getEuAndRuData();
		}

		// 报文收集
		if(collectLogInfo){
			collectLogInfo.map(function(item){
				if(item.operatorCode == operatorCodeGloab) {
					if(item.type == 'enb' && queryVue) {
						queryVue.startInterval(item.remainTime);
					}
					if(item.type == 'cpe' && queryCpeVue) {
						queryCpeVue.startInterval(item.remainTime);
					}
				}
			});
		}
	}, "json");
}

/*<%-- 接收推送过来的消息 --%>*/
function onData(msg) {
	var msg = eval("(" + msg + ")");
	if(msg['cellAlarmCountChange']){
		refreshCellInfo(msg['cellAlarmCountChange'])//新上告警基站列表刷新
	}else if (msg["getParamVal"]) {
		/*<%-- 查询参数的结果 --%>*/
		showGetParamVal(msg["data"], msg["operName"], msg["objectParam"], msg['neType']);
	} else if (msg["getParamValForRussiaElfcell"]) {
		/*<%-- 查询参数的结果 --%>*/
		showGetParamValForRussiaElfcell(msg["data"], msg['neType']);
	} else if (msg["setParamVal"]) {
		/*<%-- 设置参数的结果 --%>*/
		showSetParamVal(msg["data"], msg["operName"], msg["objectParam"], msg["index"], msg['neType']);
	} else if (msg["setParamValForRussiaElfcell"]) {
		/*<%-- 设置参数的结果 --%>*/
		showSetParamValForRussiaElfcell(msg["data"], msg['neType']);
	} else if (msg["AddObject"]) {
		showAddObjectCompleteTip(msg["data"], msg["operName"], msg['neType']);
	} else if (msg["DeleteObject"]) {
		showDeleteObjectCompleteTip(msg["data"], msg["operName"], msg['neType']);
	} else if (msg["kickOut"]) {
		/*<%-- 踢出已登录用户 --%>*/
		if (isCloudCore == "false") {
			kickOutUser(msg["user_code"]);
		}
	} else if (msg["logoutByGroupDestroyed"]) {
		logoutUserByGroupDestroyed(msg["user_code"]);
	} else if (msg["ItfnAlarm"]) {
		refreshItfnAlarm();
	} else if (msg["refreshPRBDataList"]) {
		refreshPRBDataList();
	} 
	/* 提示资源超出阈值   已经存在专门的监控程序 弹出预警信息多余（不在使用）
	else if (msg["resourceOverRange"]) {
		showResourceOverRangeTip(msg["resourceOverRange"]);
	}*/
	else if (msg["commandResult"]) {
		showCommandResult(msg["data"]);
	} else if(msg["refreshSessionExpireTime"]){
		refreshSessionExpireTime(msg["qryMap"]);
	} else if(msg["refreshSASAlarm"]){
        refreshSASMatter();
    } else if(msg["refreshCustomizeKPIReportProgress"]) {
    	//定制KPI上报任务完成
    	refreshCustomizeKPIReportProgress();
    } else if(msg["logoutStopUser"]) {
    	/*<%-- 踢出被锁定用户 --%>*/
    	if (isCloudCore == "false") {
    		logoutStopUserByUserId(msg["qryMap"]);
    	}
    } else if (msg["refreshLOGFileList"]) {
		refreshLOGFileList();
	} else if (msg["refreshNRMFileList"]) {
		refreshNRMFileList();
    }else if(msg["CellConfigReboot"]){
    	showRebootInfo(msg["data"],msg["operName"], msg['neType']);
    }else if(msg["CellConfigReset"]){
    	showResetInfo(msg["data"],msg["operName"], msg['neType']);
    } else if (msg["refreshPeriodReportLOGFileList"]) {
    	refreshPeriodReportLOGFileList();
    } else if (msg["refreshMrReportTaskList"]) {
    	refreshMrReportTaskList();
    }else if(msg["errorMML"]){
    	showErrorMML(msg["data"], msg["operName"], msg["objectParam"]);
    } else if (msg["refreshRLFileList"]) {
    	refreshRLFileList();
    } else if (msg["refreshPciLockTaskList"]) {//刷新pcilock列表
    	refreshPCIList();
    }else if (msg["refreshCPEPciLockTaskList"]) {//刷新cpepcilock列表
    	refreshCPEPCIList();
    }else if (msg["refresh_signaling_trace_detail"]){
    	refreshSignalingTraceDetail(msg["signaling_idx"]);
    }else if(msg["refresh_signalling_trace_state"]){
    	refreshSignallingStatus(msg["signalling_state_uuid"]);
    }else if(msg["refreshLicenseFileList"]){
    	refreshLicense();
    }else if (msg["refreshCellResetPwd"]) {//刷新密码重置列表
    	refreshPwdResetList();
    }else if (msg["refreshHalobAutoConfigTaskList"]) {//自启动任务
    	refreshSelfConfigTaskList(msg['task'],msg['record'],msg['next_step']);
    }else if(msg["eNBTrxTraceFlag"]){//刷新基站策略管理 功率追踪列表
    	refreshTRXTraceTable();
    }else if(msg["refresh_egw_upgrade_list"]){
    	refreshEgwUpgradeTask();
    }else if(msg["refresh_egw_monitor_list"]){ //eGW监控的同步刷新
    	refreshEgwMonitorTable();
    }else if(msg["refresh_egw_reboot_list"]){ //eGW重启的同步刷新
    	refreshRebootTable();
    }else if(msg["refresh_self_task"]){
    	refreshSelfTask(msg['update_data'])//自配置->接入控制监控
    }else if(msg["refresh_access_task"]){
    	refreshAccessTask(msg['update_data']);//自配置任务刷新
    }
	if(msg["refresh_task_record"]){
    	refreshTaskRecord()//自配置详情刷新
    }
	if(msg["SyncDeviceTR069Params"]) {
		if(msg['success'] == true || msg['success'] == 'true') {
			refreshRenderData2Dom({groupId: msg['groupId'], smallCellCode: msg['smallCellCode']});
		}else {
			
		}
	}
}
function refreshSelfTask(data){
	var $tb = $("#configTaskList");
	
	if($tb.length == 0) return;
	
	var rowData = $("#configTaskList").datagrid("getData").rows;
	var rowDataArr = rowData.map(function(item,index){
		return item.task_id;
	})
	data.map(function(item,index){
		if(rowDataArr.indexOf(item.task_id) >= 0){
			var index1 = $("#configTaskList").datagrid('getRowIndex',item.task_id);
			$("#configTaskList").datagrid("updateRow", {
				index: index1,
				row: item
			});
		}else{
			$("#configTaskList").datagrid("insertRow",{
				index:0,
				row:item
			})
		}
	})
}
function refreshAccessTask(data){
	var rowData = $("#accessTaskList").datagrid("getData").rows;
	var rowDataArr = rowData.map(function(item,index){
		return item.key_id;
	})
	data.map(function(item,index){
		if(rowDataArr.indexOf(item.key_id) >= 0){
			var index1 = $("#accessTaskList").datagrid('getRowIndex',item.key_id);
			$("#accessTaskList").datagrid("updateRow", {
				index: index1,
				row: item
			});
		}else{
			$("#accessTaskList").datagrid("insertRow",{
				index:0,
				row:item
			})
		}
	})
}
function refreshTaskRecord(){
	$("#viewConfigTaskList").datagrid("reload");
}
function refreshEgwMonitorTable(){
	if($("#eGW_monitor_datagrid") != null){
		$('#eGW_monitor_datagrid').datagrid('reload');
	}
}
function refreshRebootTable(){
	if($("#eGWRebootTable") != null){
		$("#eGWRebootTable").datagrid("reload");
	}
}
function refreshEgwUpgradeTask(){
	if($("#egwUpgradeListTable") != null){
		$("#egwUpgradeListTable").datagrid("reload");
	}
}
function refreshTRXTraceTable(){
	if ($("#trxTraceTaskListCpe") != null) {
		$("#trxTraceTaskListCpe").datagrid("reload");
	}
	if ($("#signaling_trace_enb") != null) {
		$("#signaling_trace_enb").datagrid("reload");
	}
}

function refreshLicense(){
	var tableLicense = $("#tableLicense");
	if (tableLicense.length > 0) {
		var pageNumber = $('#tableLicense').datagrid('options').pageNumber;
		var pageSize = $('#tableLicense').datagrid('options').pageSize;
		var param ={
				timeZone:timeZone,
				search_text: $('#serialNumberLicense').val(),
				page:pageNumber,
				rows:pageSize
		}
		var checkBefore = $('#tableLicense').datagrid('getChecked');
		var checkBeforeArr =[];
		$.each(checkBefore,function(index,item){
			checkBeforeArr.push(item.serial_number);
		})
		$.post(webRootPath + "/cell/license/getAllLicenseInfoData.action", param,function (data) {
			$('#tableLicense').datagrid('loadData',data.rows);
			$.each(data.rows,function(index,item){
				if(checkBeforeArr.indexOf(item.serial_number) !=-1){
					$('#tableLicense').datagrid('checkRow',index);
				}
			} )
	    }, "json");
	}
}
//踢出被锁定用户
function logoutStopUserByUserId(userIdMap) {
	var logoutUserId = userIdMap["user_id"];
	//如果是被锁定的用户则退出系统，返回到登录页面
	if (global_user_id == logoutUserId) {
        $.messager.alert(TiShi, SuoDingYongHuTiShi);
        setTimeout("kickOut()", 3000);
	}
	
}

// 基站连接状态-格式化
function connStatusFormatterSyn(value, rowData, rowIndex) {
	var synTime  = rowData.lastsyntime;
	if("initializing" == value) {
		return "<div class='el-icon el-loading status-init' onmouseover='showTips(\"initializing\",this,\" " +synTime + " \" )' onmouseout='hideTips(\"initializing\")'></div>";
	}else if("syncSourceInSync" == value) {
		return "<div class='el-icon el-loading status-sync' onmouseover='showTips(\"syncSourceInSync\",this,\" " +synTime + " \" )' onmouseout='hideTips(\"syncSourceInSync\")'></div>";
	}else if("syncSourceInSynced" == value) {
		return "<div class='el-icon el-loading status-synced' onmouseover='showTips(\"syncSourceInSynced\",this,\" " +synTime + " \" )' onmouseout='hideTips(\"syncSourceInSynced\")'></div>";
	}else if ("On" == value || value == 1) {
		return "<div class='el-icon el-icon-status-conn-on' style='font-size:20px;' onmouseover='showTips(\"on\",this,\" " +synTime + " \" )' onmouseout='hideTips(\"on\")'><div>";
	}else if("updating" == value){
		return "<div class='status_syncing' onmouseover='showTips(\"update\",this,\" " +synTime + " \" )' onmouseout='hideTips(\"update\")'></div>";
	}else if ("Exception" == value) {
		return "<div class='conn_exc' onmouseover='showTips(\"exception\",this,\" " +synTime + " \" )' onmouseout='hideTips(\"exception\")'><div>";
	}else {
		if(rowData["have_connected"] == "2"){
			return "";
		}
		return "<div class='el-icon el-icon-status-conn-off' style='font-size:20px;' onmouseover='showTips(\"off\",this,\" " +synTime + " \" )' onmouseout='hideTips(\"off\")'><div>";
	}
	return value;
}
function connStatusFormatter(value, rowData, rowIndex) {
	var synTime  = rowData.lastsyntime;
	if(rowData.have_connected == 2) return '';
	if ("On" == value || "updating" == value || 1 == value || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(value)) {
		return "<div class='el-icon el-icon-status-conn-on' style='font-size:22px;'><div>";
	} else if ("Exception" == value) {
		return "<div class='conn_exc' onmouseover='showTips(\"exception\",this,\" " +synTime + " \" )' onmouseout='hideTips(\"exception\")'><div>";
	} else {
		return "<div class='el-icon el-icon-status-conn-off' style='font-size:22px;'><div>";
	}
}
function showTips(oper,ele,time){
	var pTd = $(ele).parents("td");
	var pTr = $(ele).parents("tr");
	var tbBody = $(ele).parents(".datagrid-body:first"),
		visableWidth = tbBody.width(),
		tableWidth = $(ele).parents("table").width(),
		minWidth = Math.min(visableWidth,tableWidth);

	if(!tbBody.length) {
		visableWidth = $(ele).parents(".el-table__body:first").width();
		minWidth = Math.min(visableWidth,tableWidth);
	}

	switch(oper){
		case "off":
			controlShow("getCellInfoOffTips",BuZaiXian,pTd,pTr,minWidth);
			break;
		case "on":
			var mes1=TongBuChengGongShiJian+time;
			controlShow("getCellInfoOnTips",mes1,pTd,pTr,minWidth);
			break;
		case "update":
			controlShow("getCellInfoUpdateTips",TongBuZhong,pTd,pTr,minWidth);
			break;
		case "exception":
			var mes=TongBuShiBaiShiJian+time;
			controlShow("getCellInfoErrorTips",mes,pTd,pTr,minWidth);
			break;
		case "initializing":
			controlShow("getCellInfoInitTips",ChuShiHuaZhong,pTd,pTr,minWidth);
			break;
		case "syncSourceInSync":
			controlShow("getCellInfoSyncTips",YuanTongBuZhong,pTd,pTr,minWidth);
			break;
		case "syncSourceInSynced":
			controlShow("getCellInfoSyncedTips",YuanTongBuWanCheng,pTd,pTr,minWidth);
			break;
	}
}

function controlShow(tar,mes,pTd,pTr,minWidth){
	$("#"+tar).css({
		"top" : (pTd.offset().top) + "px",
		"left" : (pTd.offset().left + pTd.width()) + "px",
		"width" : (minWidth -pTd.width()- 10) + "px",
		"height" : pTr.height() + "px",
		"line-height" : pTr.height() + "px",
		"z-index": 8888
	}).stop().fadeIn(200);
	$($("#"+tar).children()).html(mes);
}
function hideTips(oper) {
	$("#getCellInfoOffTips").stop().fadeOut(200);
	$("#getCellInfoOnTips").stop().fadeOut(200);
	$("#getCellInfoUpdateTips").stop().fadeOut(200);
	$("#getCellInfoErrorTips").stop().fadeOut(200);
	$("#getCellInfoInitTips").stop().fadeOut(200);
	$("#getCellInfoSyncTips").stop().fadeOut(200);
	$("#getCellInfoSyncedTips").stop().fadeOut(200);
}


function showRebootTips(oper,ele){
	var pTd = $(ele).parents("td");
	var pTr = $(ele).parents("tr");
	var visableWidth = $(ele).parents(".datagrid-body:first").width(),tableWidth = $(ele).parents("table").width(),
	minWidth = Math.min(visableWidth,tableWidth);
	if(oper == "on"){
		var mes=ChongQiShengXiao;
		controlShow("getCellneedRebootTips",mes,pTd,pTr,minWidth);
	}
}

function hideRebootTips(oper) {
	if(oper == "on"){
		$("#getCellneedRebootTips").stop().fadeOut(200);
	}
}



// 动态设置向右键头的margin-top样式
function setArrowMargin(width, height) {
	$(this).children(".arrow_right").css("margin-top", (height-68)/2);
}

// 确定修改密码
function sureModifyPass(){
    $("#winPasswdRulePrompt").window("close");
    $("#passwd").window("open");
    $("#passwd").window("refresh", webRootPath + "/system/sysuser/goModifyPwd.action");
}
// 取消修改密码
function cancelModifyPass(){
    $("#winPasswdRulePrompt").window("close");
}
// 修改密码成功，退出登录 --%>
function modPwdSuccess() {
	logout(webRootPath);
}
// 锁屏
function screenLock() {
	sessionStorage.locked = true;
	$("#winLockScreen").css("display","block");
	try {
		if(baiCellsLockVue){
			baiCellsLockVue.init();
		}
	} catch (error) {}
	try {
		if(noLogoLockVue){
			noLogoLockVue.init();
		}
	} catch (error) {}
}

// 解锁入口
function cancelLock() {
	if($("#lockPassword").val() == ""){
		$('#lockPassResult').text(QingShuRuMiMa);
	}
	if ($("#lockPassword").val() != "") {
		var dateStr = dateformatter(new Date()).replace(' ','-').substr(3);
		
        $.post(webRootPath + "/system/sysuser/doCancelLockScreen.action", {lockPassword: AesEncrypt($("#lockPassword").val()), rd: AesEncrypt(dateStr)}, function (data) {
        	var result = AesDecrypt(data.result||'4yNThc1APBpYgFK4s6OvOw==', CryptoJS.enc.Utf8.parse(dateStr));
        	
        	if(result.success == true) {
        		lastVisitedTime = new Date(gloableTime);
                $("#lockPassword").val("");
                $('#lockPassResult').text("");
                $("#winLockScreen").css("display","none");
				try {
					if(baiCellsLockVue){
						baiCellsLockVue.password = '';
					}
				} catch (error) {}
				try {
					if(noLogoLockVue){
						noLogoLockVue.password = '';
					}
				} catch (error) {}
                sessionStorage.locked = '';
        	}else {
        		$('#lockPassResult').text(result["message"]);
                $("#lockPassword").val("");
        	}
        	return;
        	
            if (data["success"]) {
                lastVisitedTime = new Date(gloableTime);
                $("#lockPassword").val("");
                $('#lockPassResult').text("");
                $("#winLockScreen").css("display","none");
                sessionStorage.locked = '';
            } else {
                $('#lockPassResult').text(data["message"]);
                $("#lockPassword").val("");
            }
        }, "json");
    }
}
// RSA 加密
function RsaEncrypt(data, newKey) {
	var publicKey = newKey;

	var encryptor = new JSEncrypt();
	encryptor.setPublicKey(publicKey);

	return encryptor.encrypt(data);
}

try{
	var securityKey = 'BaiCellsSecurity';
	var crypto_key = CryptoJS.enc.Utf8.parse(securityKey);
}catch(e){}
// AES 加密
function AesEncrypt(word) {
	try{
		let srcs = CryptoJS.enc.Utf8.parse(word),
			encrypted = CryptoJS.AES.encrypt(srcs, crypto_key, {
				mode: CryptoJS.mode.ECB,
			}),
			base64Data = CryptoJS.enc.Base64.stringify(encrypted.ciphertext);
		
		return base64Data;
	}catch(e) {
		return word;
	}
}
// AES 解密
function AesDecrypt(word, newKey) {
	try{
		var skey = newKey || crypto_key, 
			encryptedHexStr = CryptoJS.enc.Base64.parse(word),
			srcs = CryptoJS.enc.Base64.stringify(encryptedHexStr),
			decrypt = CryptoJS.AES.decrypt(srcs, skey, {
				mode: CryptoJS.mode.ECB
			}),
			decryptedStr = decrypt.toString(CryptoJS.enc.Utf8);
		
		return JSON.parse(decryptedStr.toString());
	}catch(e) {
		return word;
	}
}

// AES 解密
function AesDecryptSingle(word, newKey) {
	try{
		var skey = newKey || crypto_key, 
			encryptedHexStr = CryptoJS.enc.Base64.parse(word),
			srcs = CryptoJS.enc.Base64.stringify(encryptedHexStr),
			decrypt = CryptoJS.AES.decrypt(srcs, skey, {
				mode: CryptoJS.mode.ECB
			}),
			decryptedStr = decrypt.toString(CryptoJS.enc.Utf8);
		
		return decryptedStr.toString();
	}catch(e) {
		return word;
	}
}

function documentClick(){
	if ($("#nav_bar_user_pop").is(":visible")) {
		$("#nav_bar_user_pop").slideUp(); 
		$("#arrow").removeClass("arrowUp").addClass("arrow");
	}	
	if ($("#nav_language_pop").is(":visible")) {
		$("#nav_language_pop").slideUp(); 
		$("#languageicon").removeClass("language_arrowUp").addClass("language_arrow")
	}
}
// 设置当前运营商
function setCurrOperator(operator_code) {
	try{
		$.ajaxSettings.async = false;
		$.post(webRootPath + "/system/operator/selectOperator.action", {"operator_code": operator_code}, function(data) {
					if (data) {
						timeZone = -1 * new Date().getTimezoneOffset();
						if (typeof(data.time_zone) != "undifined"
								&& data.time_zone != null && data.time_zone != "") {
							// 设置用户显示的运营商时区
							timeZone = data.time_zone;
						}
						if (isCloud == 'true') { 
							$("#curr_operator_span").text(data.operator_name);
						}
					}
				}, "json");	
	}finally{
		$.ajaxSettings.async = true;
	}
	if (isCloud == 'true') { 
		// 非移动版
		refreshAliveAlarmCount();
        //$("#curr_operator_div").css("display", "inline-block");
	}
}
// 拼装时区显示
function getTimeZoneStr(timeZone) {
	var zone = "(UTC)";
	if (timeZone > 0) {
		var hour = Math.floor(timeZone / 60);
		hour = (hour < 10 ? "0" + hour : hour);
		var minute = (timeZone % 60);
		minute = (minute < 10 ? "0" + minute : minute);
		zone = "(UTC+" + hour + ":" + minute + ")";
	} else if (timeZone < 0) {
		var hour = Math.floor(-1 * timeZone / 60);
		hour = (hour < 10 ? "0" + hour : hour);
		var minute = (-1 * timeZone % 60);
		minute = (minute < 10 ? "0" + minute : minute);
		zone = "(UTC-" + hour + ":" + minute + ")";
	}
	return zone;
}

// easyui 控件扩展
function extendEasyui() {
	// 扩展easyui datagrid rownumber列样式 解决行数位数过大显示不全问题
	$.extend($.fn.datagrid.methods, {
				fixRownumber : function(dg) {
					var panel = dg.datagrid("getPanel");
					// 获取最后一行的number容器,并拷贝一份
					var clone = $(".datagrid-cell-rownumber", panel).last().clone();
					// 由于在某些浏览器里面,是不支持获取隐藏元素的宽度
					clone.css({"position" : "absolute", left : -1000 }).appendTo("body");
					var width = clone.width("auto").width();
					// 默认宽度是30,所以只有大于30的时候才进行fix
					if (width > 30) {
						// 多加5个像素,保持一点边距
						$(".datagrid-header-rownumber,.datagrid-cell-rownumber", panel).width(width + 5);
						// 修改了宽度之后,需要对容器进行重新计算,所以调用resize
						dg.datagrid("resize");
						// 一些清理工作
						clone.remove();
						clone = null;
					} else {
						// 还原成默认状态
						$(".datagrid-header-rownumber,.datagrid-cell-rownumber", panel).removeAttr("style");
					}
					return;
				},
				enableContextmenuAutoSize : function (dg) {
					dg.siblings("div.datagrid-view2").find("tr.datagrid-header-row").find("div.datagrid-cell").bind("contextmenu", function (event) {
						event.preventDefault();
						event.stopPropagation();
						var field = $(this).closest("td").attr("field");
						var id = $(this).closest("div.datagrid-view2").siblings(".datagrid-f").attr("id");
						$("#" + id).datagrid("autoSizeColumn", field);
					}).attr("title", YouJianZiShiYingKuanDu);
				}
			});
}

function autoSizeColumn(ele) {
	event.preventDefault();
	event.stopPropagation();
	var field = $(ele).closest("td").attr("field");
	var id = $(ele).closest("div.datagrid-view2").siblings(".datagrid-f").attr("id");
	$("#" + id).datagrid("autoSizeColumn", field);
}

function gridOnLoadSuccessForAutoSize() {
	$(this).datagrid("enableContextmenuAutoSize");
	$(this).datagrid("fixRownumber");
}
// 公共窗口开启、关闭方法
var winDefaultSelector = "#winDefault";
function openDefaultWindow(url,options) {
	var op = {width: 500,height: 400},
		windowItem = $(winDefaultSelector);
	if(options) {
		$.extend(op,options);
		try{
			setDefaultWindow(op);
		}catch(e){}
	}
	try{
		if(url) {
			windowItem.window("open").window("refresh",url);
			windowItem.attr("clearFlag","yes");
		}else {
			var clearFlag = windowItem.attr("clearFlag");
			if(clearFlag && clearFlag=="yes") {
				windowItem.window("clear");
				windowItem.attr("clearFlag","no");
			}
			windowItem.window("open");
		}
	}catch(e){}
}
function setDefaultWindow(options) {
	var windowItem = $(winDefaultSelector);
	try{/* call back */
		var winOptions = windowItem.window("options");
		if(options.onLoad) {
			windowItem.window({
				onLoad: function(){
					options.onLoad();
				}
			});
		}
		else winOptions.onLoad = function(){};
	}catch(e){}
	if(['get','post','GET','POST'].includes(options.type)) {
		windowItem.panel("options").method = options.type;
	}
	try{
		if(options.title) windowItem.window("setTitle",options.title);
	}catch(e){}
	try{
		if(options && (options.width || options.height || options.left || options.top)) {
			return windowItem.window("resize",options).window("center").window("resize",options);
		}
	}catch(e){}
}
function closeDefaultWindow(selector) {
	try{
		if(selector) $(selector).window("clear").window("close");
		else $(winDefaultSelector).window("clear").window("close");
	}catch(e){}
}
//两个日期比较   用于不能选择当前日期之前日期
function differ(pre,after){
	  function initDate(date){
		  var d = new Date(date.getFullYear(), (date.getMonth()+1), date.getDate())
		  return d;
	  }
    var preDate = initDate(new Date(pre)), afterDate = initDate(new Date(after));
    var differTimes = preDate.valueOf() - afterDate.valueOf();
    return Math.round(differTimes/(24 * 60 * 60 * 1000));
}
//时间选择表  不能选择当前之前的日期
function disableSelectEarlyTime(ele){
	var now = new Date(gloableTime);
	var d1 = new Date(now.getFullYear(), now.getMonth(), now.getDate());
	now = new Date(d1);
	ele.datetimebox({}).datetimebox('calendar').calendar({
		validator: function(date) {
			if(differ(date,now)<0){
				return false;
			}else{
				return true;
			}
		}
	});
}

//更新信令等待图标 zhengss
function refreshSignalingTraceDetail(uuid){
	if ($("#signaling_trace_dg").length != 0){
		var rows=$("#signaling_trace_dg").datagrid("getRows");
		if(rows.length>0){
			for(var i=0;i<rows.length;i++){
				if(rows[i].UUID==uuid){
					//监控等待图标,信令上报后改为可展开图标"+"
					if(rows[i].SUBGRID=="0"){
						//$.messager.alert(TiShi, "begin report...");
						$("#toolbar_dg").next().find(".datagrid-view1 .datagrid-body .datagrid-body-inner table")
						.find("tr[datagrid-row-index="+i+"]").find("td").eq(1).find("span")
						.removeClass("datagrid-row-waiting").addClass("datagrid-row-expand"); 
						var upd=$('#signaling_trace_dg').datagrid('selectRow',i).datagrid('getSelected');
						upd.SUBGRID='1';
					}
					//如果页面有已展开任务,图标"-",并且还在追踪时长范围内,则动态推送信令消息
					if($("#toolbar_dg").next().find(".datagrid-view1 .datagrid-body .datagrid-body-inner table")
							.find("tr[datagrid-row-index="+i+"]").find("td").eq(1).find("span").hasClass("datagrid-row-collapse")){
						 //$.messager.alert(TiShi, "begin push...");
						$("#signaling_trace_dg").datagrid('getRowDetail',i).find('table.ddv').datagrid('reload');
						
					}
					
				}
				
			}
		}
	}
	
}

/*<%-- 刷新“北向接口通信异常告警”--%>*/
function refreshItfnAlarm() {
	$.post(webRootPath + '/cell/fault/queryIfItfnAlarm.action', {}, function (data) {
    	if (data["success"]) {
    		$(".itfn_alarm").show();
    	} else {
    		$(".itfn_alarm").hide();
    	}
    }, 'json');
}

function refreshSignallingStatus(uuid){
	
	if ($("#signaling_trace_dg").length != 0){
		var rows=$("#signaling_trace_dg").datagrid("getRows");
		if(rows.length>0){
			for(var i=0;i<rows.length;i++){
				if(rows[i].UUID==uuid){
					$("#signaling_trace_dg").datagrid('getData').rows[i].ACTION="OFF";
					//$("#signaling_trace_dg").datagrid('getData').rows[i].SUBGRID="1";
					if(rows[i].SUBGRID=="0"){
						$("#toolbar_dg").next().find(".datagrid-view1 .datagrid-body .datagrid-body-inner table")
						.find("tr[datagrid-row-index="+i+"]").find("td").eq(1).find("span")
						.removeClass("datagrid-row-waiting").addClass("datagrid-row-expand"); 
						var upd=$('#signaling_trace_dg').datagrid('selectRow',i).datagrid('getSelected');
						upd.SUBGRID='1';
					}
					$("#signaling_trace_dg").datagrid('refreshRow',i);
				}
				
			}
		}
	}
	
}
function productFormatter(value,row,index){
	//var type_1 = /^FAP\/\w+|\*\/DC$/;
	if(row.have_connected == 2) return '';
	if("--" == value) return "--";
	return value;
}
/**
* json数据映射到html
* @param obj: json数据
* @param type: 映射类型（text：值映射为innerHTML；其他：值映射为value）
* @param path: 根路径，name属性的映射前缀
**/
function parseJson2Html(obj,type,path){
	var props = {};
	for (var key in obj) {
		  var propPath = path;
    	if (path) propPath = path + '.' + key;
  		else propPath = key;

  		if(typeof obj[key] === 'object') arguments.callee(obj[key], type, propPath);
  		else {
   			propPath = propPath.replace(/\.(\d+)\./g, '[+$1+].').replace(/[+]/g,'');
   			props[propPath] = obj[key];
  		}
 	}
 	/* 数据映射到 html */
 	for (var key in props) {
  		var doms = document.querySelectorAll("[name='"+key+"']");
  		if(doms.length==0) continue;
  		for(var domIndex in doms) setValue(doms[domIndex],props[key]);
 	}
 	function setValue(domObj,value){
 		if(domObj.type == 'radio' || domObj.type == 'checkbox'){// 扩展radio、checkbox
 			if(domObj.type == 'radio'){
 				if(domObj.value == value) domObj.setAttribute('checked',true);
 			}else{
 				var checkboxs = value.split(',');
 				if(checkboxs.indexOf(domObj.value)>=0) domObj.setAttribute('checked',true);
 			}
 		}else if((domObj.value+'') != (value+'') && type != 'text') domObj.value = value;
 		else if(domObj.innerText != (value+'') ) domObj.innerHTML = value;
 		// 支持easyui组件赋值
 		try{parseEasyUI(domObj,value);}catch(e){}
 	}
 	function parseEasyUI(obj,value){// 扩展对easyui组件的支持
 		var dClass = obj.className;
 		if(dClass && dClass.includes('-value') && obj.type == 'hidden'){
 			var ctn = $(obj).parent().prev(), classArr = ctn.prop('class').split(' ');
 			$.each(classArr,function(index,item){
 				if(item.includes('easyui-')) ctn[item.replace('easyui-','')]('setValue',value);
 			});
 		}
 	}
}


$.extend(jQuery.fn,{
	ready: function(fn){
		// Add the callback
		var fun = function(){
				try{
					fn.apply(this,arguments);
				}catch(e){console.trace(e); }
			};
		jQuery.ready.then(fun);
		return this;
	},
	size: function(){
		return this.length;
	}
});
Date.prototype.format = function () {
	var s = '';
	var mouth = (this.getMonth() + 1) >= 10 ? (this.getMonth() + 1) : ('0' + (this.getMonth() + 1));
	var day = this.getDate() >= 10 ? this.getDate() : ('0' + this.getDate());
	s += this.getFullYear() + '-'; 
	s += mouth + "-"; 
	s += day; 
	return (s);   
};

function getAll(begin, end) {
	var arr = [];
	var ab = begin.split("-");
	var ae = end.split("-");
	var db = new Date();
	db.setUTCFullYear(ab[0], ab[1] - 1, ab[2]);
	var de = new Date();
	de.setUTCFullYear(ae[0], ae[1] - 1, ae[2]);
	var unixDb = db.getTime() - 24 * 60 * 60 * 1000;
	var unixDe = de.getTime() - 24 * 60 * 60 * 1000;
	for (var k = unixDb; k <= unixDe;) {
		//console.log((new Date(parseInt(k))).format());
		k = k + 24 * 60 * 60 * 1000;
		arr.push((new Date(parseInt(k))).format());
	}
	return arr;
}
function timestampToTime(timestamp) {
    var date = new Date(timestamp);
    var Y = date.getFullYear() + '-';
    var M = (date.getMonth() + 1 < 10 ? '0' + (date.getMonth() + 1) : date.getMonth() + 1) + '-';
    var D = date.getDate() + ' ';
    var h = (date.getHours() < 10 ? '0' + (date.getHours()) : date.getHours()) + ':';
    var m =(date.getMinutes() < 10 ? '0' + (date.getMinutes()) : date.getMinutes()) + ':';
    var s =(date.getSeconds() < 10 ? '0' + (date.getSeconds()) : date.getSeconds());
    return Y + M + D + h + m + s;
}