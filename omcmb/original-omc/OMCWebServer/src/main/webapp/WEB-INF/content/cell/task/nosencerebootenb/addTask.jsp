<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
	<style>
		.errorTitle{
			margin-top:5px;
			color:red !important;
		}
		.errorInput{
			border-color:red ;
		}
		.labelContainer + .textbox{
			height:27px;
		}
	</style>
		<!-- 固定头部 -->
		<div class="addNoChangeHeader">
			<span>定时重启任务</span>
			<span class="el-icon el-icon-close" onclick="closeAddTask()"></span>
		</div> 
		<div class="bodycontainer" style="overflow:scroll;padding:20px;">
			<div>
				<div class="group-title not-extend">
					<span class="title-icon"></span>
					<span class="title-text">设备重启条件</span>
				</div>
				
				<div class="viewDatagrid">
					<p style="font-size:14px;color:#000000;margin-bottom:10px;">排除设备<span style="margin-left:5px;color:#1DA3FC;font-size:12px;">(  指定设备本次任务不执行  )</span></p>
					<div id="add_temp_device_datagrid"></div>
				</div>
				<p></p>
				<p id="choseDeviceTitle" style="visibility:hidden;margin-top:40px;margin-left:51px" class="errorTitle">请选择设备</p><!-- 提示 -->
				<p style="font-size:14px;color:#000000;margin-bottom:10px;margin:20px 0px 5px 55px;">版本号指定</p>
				<div class="versionContainer" style="height:300px;padding:0px 45px 0px 55px">
					<table id="versionTable"></table>
				</div>
				<p id="choseVersion" style="visibility:hidden;margin-left:51px" class="errorTitle">请选择版本号</p>
				<div class="labelContainer">
				 	<div class="">
				 		<p>运行时长最小限制</p>
				 		<div>
				 			<input id="minRunTime" name="minRunTime" value="" type="text" /><span style="margin-left:8px">/小时</span>
				 			<p style="visibility:hidden" class="errorTitle">请输入正整数</p>
				 		</div>
				 		
				 	</div>
				 	<!--  #30707 V4基站无感知重启功能优化 去掉用户数限制，由基站自己来判断是否重启
				 	<div class="" style="margin-left:90px;">
				 		<p>在线用户数最大限制</p>
				 		<div>
				 			<input id="maxOnlineUserNum" name="maxOnlineUserNum" value="" type="text" /><span style="margin-left:8px">/人</span>
				 			<p style="visibility:hidden" class="errorTitle">请输入正整数</p>
				 		</div>
				 	</div>
				 	-->
				</div>
				<div class="cutOffLine"></div>
			</div>
			<!-- 运行时间设置 -->
			
			<div>
				<div class="group-title not-extend">
					<span class="title-icon"></span>
					<span class="title-text">运行时间设置(开始时间中的分钟数要避开基站上报kpi的时间点，比如03,18,33,48)</span>
				</div>
				<div class="item_container">
					<p style="font-size:14px;color:#000000;margin-bottom:10px;margin:50px 0px 5px 55px;">执行日期范围</p>
					<div class="labelContainer" style="border:1px solid #D1ECF5;padding:20px 13px 13px 20px">
				 	<div class="" style="height:75px;">
				 		<p>开始日期</p>
				 		<div>
				 			<input id="nosenceStartTime" data-options="height:30" type="text" class="easyui-datebox"/>
				 		</div>
				 		<p id="choseStartDay" style="visibility:hidden;" class="errorTitle">请选择开始日期</p>
				 	</div>
				 	<div class="" style="margin-left:90px;height:75px;">
				 		<p>结束日期</p>
				 		<div>
				 			<input id="nosenceEndTime" width=270 data-options="height:30" type="text" class="easyui-datebox"/>
				 		</div>
				 		<p id="choseEndDay" style="visibility:hidden;" class="errorTitle">请选择结束日期</p>
				 	</div>
				 	<div style="height:21px;padding-top:31px;margin-left:25px;">
				 		<input id="singSelect" type="checkbox" name="checkitem" value="" style="width:16px;height:16px;vertical-align:middle" onclick="checkedSingSelect(this)" />
				 		<span style="display:inline-block;height:16px;line-height:16px;">单次任务</span>
				 	</div>
				</div>
				</div>
				
				<div class="item_container">
					<p style="font-size:14px;color:#000000;margin-bottom:10px;margin:50px 0px 5px 55px;">执行时间段</p>
					<div class="labelContainer" style="border:1px solid #D1ECF5;padding:20px 13px 13px 20px">
				 	<div class="">
				 		<p>开始时间</p>
				 		<div>
				 			<input id="startTime" name="startTime" value="" type="text" placeholder = "时间格式 XX:XX:XX" />
				 			<p style="visibility:hidden" class="errorTitle">请输入开始时间</p>
				 		
				 		</div>
				 	</div>
				 	<div class="" style="margin-left:69px;">
				 		<p>结束时间</p>
				 		<div>
				 			<input id="endTime" name="endTime" value="" type="text" placeholder = "时间格式 XX:XX:XX"/>
				 			<p style="visibility:hidden" class="errorTitle">请输入结束时间</p>
				 		</div>
				 	</div>
				 	
				</div>
				</div>
				
				<div class="cutOffLine"></div>
			</div>
			<!-- 重启数量设置 -->
			<div>
				<div class="group-title not-extend">
					<span class="title-icon"></span>
					<span class="title-text">重启数量限制</span>
				</div>
				<div class="labelContainer">
				 	<div class="">
				 		<p>同时重启数量最大限制</p>
				 		<div>
				 			<input id="maxRebootNum" name="maxRebootNum" value="" type="text" />
				 			<p style="margin-top:5px">安全范围最大值20</p>
				 		</div>
				 		
				 	</div>
				 	<div class="" style="margin-left:90px;">
				 		<p>全部重启数量最大限制</p>
				 		<div>
				 			<input id="allRebootNum" name="allRebootNum" value="" type="text" />
				 			<p style="margin-top:5px">安全范围最大值200</p>
				 		</div>
				 		
				 	</div>
				</div>
			</div>
			
			
		</div>
		<div id="footercontainer" class="footercontainer" style="">
			 <div class="linkbuttonGroup">
	        	<a href="#" class="linkbutton linkbutton_trend" onclick="submitRebootInfo()"><span><%=rb.getString("QueDing")%></span></a>
	        	<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeAddTask()"><span><%=rb.getString("QuXiao")%></span></a>
	        </div>
		</div>
		
		<!-- 设备选择列表 -->
<div id="toolbar_add_temp_device_datagrid" style="padding:5px 10px;">
	<div class="queryGroup" style="margin:0 0 0 20px;">
		<input name="search_text" id="tempAddDeviceQuery" style="width:260px;" placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>">
		<b class="el-icon el-icon-common-search" onclick='javascript: $("#add_temp_device_datagrid").pairgrid("reload");'></b>
	</div>
</div>
		<script>
		var mockData = {}
		var nosenDisabledFlag = ""
		$(function(){
			closeLoading();
		
			//页面加载完成  请求当前的状态 根据当前状态判断是查看还是编辑
			$.post("${ctx}/task/secretReboot/getSwitch.action",{operatorCode:operatorCode},function(data){
				nosenDisabledFlag = data.startFlag;
				 if(data.startFlag == "1"){
					$("#footercontainer").hide();
					//禁用所有组件
					$("#minRunTime").prop("disabled",true);
					//#30707 V4基站无感知重启功能优化 去掉用户数限制，由基站自己来判断是否重启
					//$("#maxOnlineUserNum").prop("disabled",true);
					$("#startTime").prop("disabled",true);
					$("#endTime").prop("disabled",true);
					$("#singSelect").prop("disabled",true);
					$("#maxRebootNum").prop("disabled",true);
					$("#allRebootNum").prop("disabled",true);
					$("#nosenceEndTime").datebox('disable',true);
					$("#nosenceStartTime").datebox('disable',true);
					$("#add_temp_device_datagrid").addClass("readonly");
					$("#add_temp_device_datagrid").pairgrid({"readonly":true});
					//$(".pairgrid-op").hide();
				}else{
					$("#footercontainer").show();
				} 
			},"json")
			//页面加载完成 获取初始化参数
			$.post("${ctx}/task/secretReboot/getConfigInfo.action",{operatorCode:operatorCode,timeZone:timeZone},function(data){
				mockData = data;
				$("#minRunTime").val(mockData.minRunTime);
				//#30707 V4基站无感知重启功能优化 去掉用户数限制，由基站自己来判断是否重启
				//$("#maxOnlineUserNum").val(mockData.maxOnlineUserNum);
				$("#startTime").val(mockData.startTime);
				$("#endTime").val(mockData.endTime); 
			  /*   $("#temp_time_start").combobox("setValue");
		    	$("#temp_time_end").combobox("setValue"); */
				$("#maxRebootNum").val(mockData.maxRebootNum);
				$("#allRebootNum").val(mockData.allRebootNum);
		    	$("#nosenceStartTime").datebox('setValue',mockData.startDay);
		    	
		    	if(mockData.onlyOnceFlag == '1'){
		    		$("#singSelect").prop("checked",true)
		    		$("#nosenceEndTime").datebox('setValue',mockData.startDay);
		    		$("#nosenceEndTime").datebox('disable',true);
		    	}else{
		    		$("#singSelect").prop("checked",false)
		    		$("#nosenceEndTime").datebox('setValue',mockData.endDay);
		    	}
			},"json")
			
			var versionData = [{"software_version":'omc4.6.1',checked:false},{"software_version":'omc4.6.2',checked:true}]
			 $("#add_temp_device_datagrid").pairgrid({
					idField : 'smallCellCode',
			    	leftUrl : '${ctx}/task/secretReboot/getEnbListPageData.action',
			    	rightUrl : '${ctx}/task/secretReboot/getNotSelectedEnbPageData.action',
					border : false,
					fit : true,
					fitColumns : true,
					rownumbers : true,
					striped : true,
					singleSelect : true,
					pageList : [50, 100, 150, 200, 250, 300 ],
					pagination : true,
					pagePosition : 'bottom',
					toolBar : "#toolbar_add_temp_device_datagrid",
					zone : [50,50],
					queryName : 'serialNumber,hostName',
					messages:{queryName:'<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> '},
					onCheck: addDeviceDatagridCheck,
			        leftBeforeLoad : beforeLoad_add_temp_left_device_datagrid,
					leftColumns : [ {
						field : 'ck',
						checkbox:true,
					}, 
					{field: 'connection_status',fixed:true,width: 40,formatter: connStatusFormatter},
					{
						field : 'smallCellCode',
						hidden : true
					}, {
						field : 'serialNumber',
						sortable : true,
						width : 100,
						title : '<%=rb.getString("XiaoZhanBianMa")%>'
					}, {
						field: 'hostName',
						sortable: true,
						width: 120,
						title: '<%=rb.getString("HostName")%>'
					}, {
						field: 'groupName',
						sortable: true,
						hidden : true
					} ],
					rightColumns : [ {
						field : 'smallCellCode',
						hidden : true
					}, {
						field : 'serialNumber',
						width : 110,
						title : '<%=rb.getString("XiaoZhanBianMa")%>(<%=rb.getString("HostName")%>)',
						formatter: selectedDeviceTableFun
					}, {
						field: 'hostName',
						sortable: true,
						hidden : true
					}]
				});
		    
		    $("#versionTable").datagrid({
		    	border : true,
		        fit : true,
		        url : '${ctx}/task/secretReboot/getSoftwareVersionList.action',  
		       /*  data:versionData, */
		        rownumbers : true,
		        striped : true,
		        checkOnselect:false,
		        singleSelect : false,
		        selectOncheck:false,
		        fitColumns : true,
		        /* pagination : true,
		        pageSize : 20,
		        pagePosition : 'bottom', */
		        idField : 'software_version',
		        onCheck:checkVersion,
		        columns: [[
						   {field:'ck',width:120,checkbox:true},
		                   {field:'software_version',title:'版本号',width:120}
		               ]],
		        onLoadSuccess: getcheckedRow,
		        onBeforeSelect:function(){
		        	return false;
		        }
		       
		    });
		})
	

		//表格加载前事件
		function beforeLoad_MMLScriptTaskList(param) {
			//添加查询条件
			param["timeZone"] = timeZone;
			param["likeFields"] = "task_name";
			param["searchText"] = taskSearchText;
			param["startTime"] = queryStartTime;
			param["endTime"] = queryEndTime;
		}
		//任务列表加载完成，默认选中第一条数据
		function loadSuccessMMLScriptTaskList(data) {
			$(this).datagrid("enableContextmenuAutoSize");
			if (data["rows"].length > 0) {
				$("#noSenceTaskList").datagrid("selectRow", 0);
			}
		}
		function MMLTaskFormatter(value, rowData, rowIndex){	
			var task_progress = rowData.TASK_PROGRESS;
			var task_status = rowData.TASK_STATUS;
			var task_id = rowData.TASK_ID;
			
			value = "<div class='operation_more' title='"+ CaoZuo+"' onclick='enbMmlscriptOp("+ task_id + ",\""+task_status+"\",this)'></div>";
			return value;
		}
		//设置操作列单元格样式 
		function setMMLStyle(){
			return 'position:relative'
		}
		function beforeLoad_add_temp_left_device_datagrid(param){
			param.searchText = $("#tempAddDeviceQuery").val();
		}
		function beforeLoad_add_temp_right_device_datagrid(param){
		}
		function addDeviceDatagridCheck(){
			$("#choseDeviceTitle").css("visibility","hidden");
    		$("#choseDeviceTitle").removeClass("errorInput");
			/* $("#add_temp_device_datagrid").parent().next().html(""); */
		}
		function checkVersion(){
			$("#choseVersion").css("visibility","hidden");
    		$("#choseVersion").removeClass("errorInput");
		}
		function selectedDeviceTableFun(value, rowData, rowIndex){
		   	return rowData.serialNumber+"("+rowData.hostName+")"
		}
	    function submitRebootInfo(){
	    	var params = {};
	    	params.timeZone=timeZone;
	    	//选中的基站
	    	var selectedEnbsArr = []; 
	    	var selectedEnbs = $("#add_temp_device_datagrid").pairgrid("getData");
	    	if(selectedEnbs.length>0){
	    		$.each(selectedEnbs,function(index,ele){
	    			selectedEnbsArr.push(ele.smallCellCode);
	    		})
	    		selectedEnbsArr = JSON.stringify(selectedEnbsArr);
	    		params.noSelectedENB = selectedEnbsArr;
	    		$("#choseDeviceTitle").css("visibility","hidden");
	    		$("#choseDeviceTitle").removeClass("errorInput");
	    	}else{
	    		/* $(".bodycontainer").animate({scrollTop:'50px'},800)
	    		$("#choseDeviceTitle").css("visibility","visible");
	    		$("#choseDeviceTitle").addClass("errorInput");
	    		return false; */
	    		selectedEnbsArr = JSON.stringify(selectedEnbsArr);
	    		params.noSelectedENB = selectedEnbsArr;
	    	}
	    	
	    	//验证版本号是否获取
	    	var versionDataArr = $("#versionTable").datagrid("getChecked");
	    	var versionarr = []
	    	if(versionDataArr.length == 0){
	    		//没有选中提示并return
	    		$(".bodycontainer").animate({scrollTop:'250px'},800)
	    		$("#choseVersion").css("visibility","visible");
	    		$("#choseVersion").addClass("errorInput");
	    		/* $.messager.alert('提示', '请选择版本号'); */
	    		return false;
	    	}else{
	    		$("#choseVersion").css("visibility","hidden");
	    		$("#choseVersion").removeClass("errorInput");
	    		$.each(versionDataArr,function(index,ele){
	    			versionarr.push(ele.software_version);
	    		})
	    		versionarr = JSON.stringify(versionarr)
	    		params.selectedVersion=versionarr
	    	}
	    	
	    	var minRunTime = $("#minRunTime").val();
	    	//校验运行时长
	    	if(!checkRait(minRunTime)){
	    		 $("#minRunTime").siblings('p').css("visibility","visible");
	    		 $("#minRunTime").addClass("errorInput")
	    		 return false;
	    	}else{
	    		 params.minRunTime = minRunTime
	    		 $("#minRunTime").siblings('p').css("visibility","hidden");
	    		 $("#minRunTime").removeClass("errorInput");
	    	}
	    	//#30707 V4基站无感知重启功能优化 去掉用户数限制，由基站自己来判断是否重启
	    	//校验在线用户数量最大限制
	    	/* var maxOnlineUserNum = $("#maxOnlineUserNum").val()
	    	if(!checkRait(maxOnlineUserNum)){
	    		 $("#maxOnlineUserNum").siblings('p').css("visibility","visible");
	    		 $("#maxOnlineUserNum").addClass("errorInput")
	    		 return false;
	    	}else{
	    		 params.maxOnlineUserNum = maxOnlineUserNum
	    		 $("#maxOnlineUserNum").siblings('p').css("visibility","hidden");
	    		 $("#maxOnlineUserNum").removeClass("errorInput")
	    	} */
	    	//开始日期校验
	    	var startDay = $("#nosenceStartTime").datebox('getValue');
	    	var endDay = $("#nosenceEndTime").datebox('getValue');
	    	
	    	if(startDay == ""){
	    		
	    		
	    		$("#choseStartDay").css("visibility","visible");
	    		$("#choseStartDay").addClass("errorInput")
	    		return false;
	    	}else{
	    		
	    		$("#choseStartDay").css("visibility","hidden");
	    		$("#choseStartDay").removeClass("errorInput")
	    	}
	    	//判断是不是选择了单次任务 选择了单次任务结束时间传空
	    	if($("#singSelect").prop("checked")) {
	    		params.onlyOnceFlag=1;
	    		params.startDay = startDay;
	    		params.endDay = endDay;
	    	}else{
	    		params.onlyOnceFlag=2;
	    		if(endDay == ""){
	    			$("#choseEndDay").css("visibility","visible");
		    		$("#choseEndDay").addClass("errorInput")
		    		return false;
	    		}else{
	    			
		    		$("#choseEndDay").css("visibility","hidden");
		    		$("#choseEndDay").removeClass("errorInput")
	    			params.endDay = endDay;
	    			params.onlyOnceFlag=0;
		    		params.startDay = startDay;
	    		}
	    	}
	    	//判断开始结束时间
	    	var startTime = $("#startTime").val();
	    	var endTime = $("#endTime").val();
	    	if(!isTime(startTime)){
	    		$("#startTime").siblings('p').css("visibility","visible");
	    		$("#startTime").addClass("errorInput")
	    		return false;
	    	}else{
	    		$("#startTime").siblings('p').css("visibility","hidden");
	    		$("#startTime").removeClass("errorInput")
	    	}
	    	if(!isTime(endTime)){
	    		$("#endTime").siblings('p').css("visibility","visible");
	    		$("#endTime").addClass("errorInput")
	    		return false;
	    	}else{
	    		$("#endTime").siblings('p').css("visibility","hidden");
	    		$("#endTime").removeClass("errorInput")
	    	}
	    	//结束时间不能小于结束时间
	    	var reg = new RegExp(':',"g")
	    	var testStartTime = startTime;
	    	var testEndTime = endTime;
	    	testStartTime = parseInt(testStartTime.replace(reg,""));
	    	testEndTime = parseInt(testEndTime.replace(reg,""));
	    	if(testStartTime>testEndTime || parseInt(startTime) == testEndTime){
	    		$.messager.alert('提示', '结束时间不能早于开始时间');
	    		return false;
	    	}else{
	    		params.startTime = startTime;
	    		params.endTime = endTime;
	    	}
	    	// 同时重启最大数量限制
	    	var maxRebootNum = $("#maxRebootNum").val()
	    	if(!checkRait(maxRebootNum)){
	    		 $("#maxRebootNum").siblings('p').text("请输入正整数");
	    		 $("#maxRebootNum").siblings('p').addClass("errorTitle");
	    		 $("#maxRebootNum").addClass("errorInput")
	    		 return false;
	    	}else{
	    		
	    		 if(maxRebootNum > 20 || maxRebootNum < 0){
	    			 $("#maxRebootNum").siblings('p').text("安全范围最大值20");
		    		 $("#maxRebootNum").siblings('p').addClass("errorTitle");
		    		 $("#maxRebootNum").addClass("errorInput")
		    		 return false;
	    		 }else{
	    			 $("#maxRebootNum").siblings('p').text("安全范围最大值20");
		    		 $("#maxRebootNum").siblings('p').removeClass("errorTitle");
		    		 $("#maxRebootNum").removeClass("errorInput")
		    		 params.maxRebootNum = maxRebootNum
	    		 }
	    		 
	    	}
	    	//全部重启数量限制
	    	var allRebootNum = $("#allRebootNum").val()
	    	if(!checkRait(allRebootNum)){
	    		 $("#allRebootNum").siblings('p').text("请输入正整数");
	    		 $("#allRebootNum").siblings('p').addClass("errorTitle")
	    		 $("#allRebootNum").addClass("errorInput")
	    		 return false;
	    	}else{
	    		if(allRebootNum > 200 || allRebootNum < 0 ){
	    			$("#allRebootNum").siblings('p').text("安全范围最大值200");
	    			$("#allRebootNum").siblings('p').addClass("errorTitle")
		    		$("#allRebootNum").addClass("errorInput")
		    		return false;
	    		}else{
	    			params.allRebootNum = allRebootNum;
	    			$("#allRebootNum").siblings('p').text("安全范围最大值200").removeClass("errorTitle");
	    			$("#allRebootNum").siblings('p').removeClass("errorTitle");
		    		$("#allRebootNum").removeClass("errorInput")
	    		}
	    		 
	    	}
	    	$.post("${ctx}/task/secretReboot/updateConfigInfo.action",params,function(data){11
				if(data["success"]){
					toast("提交成功",$('#rightContainerPanel'),'success');
					setTimeout(function(){
						$("#rightContainerPanel").animate({
				    		right:"-2000px"
				    	});
						$("#noSenceTaskList").datagrid("reload")
					},2000)
				}else{
					toast("提交失败",$('#rightContainerPanel'),3000);
				}
			},"json") 
	    	
	    	
	    }
	    //正则 验证输入的是否是数字
	    function checkRait(val){
	    	var reg = /^\+?[0-9][0-9]*$/;
	    	if(reg.test(val)){
	    		return true;
	    	}else{
	    		return false;
	    	}
	    }
	    function closeAddTask(){
	    	$("#rightContainerPanel").animate({
	    		right:"-2000px"
	    	});
	    }
	    function checkedSingSelect(e){
	    	if($(e).prop("checked")) {
	    		$("#nosenceEndTime").datebox({disabled:true});
	    		var newTime = $("#nosenceStartTime").datebox('getValue');
	    		$("#nosenceEndTime").datebox('setValue',newTime);
	    	}else{
	    		$("#nosenceEndTime").datebox({disabled:false});
	    		
	    	}
	    }
	    function isTime(str) { 
	    var reg = /^(0\d{1}|1\d{1}|2[0-3]):[0-5]\d{1}:([0-5]\d{1})$/;
	    
	    if (!reg.test(str)) 
	    { 
	    return false 
	    }  
	    return true; 
	    } 
	    function getcheckedRow(data){
	    	var checkedrow = data.rows;
	    	for(var i = 0;i<checkedrow.length;i++){
	    		if(checkedrow[i].checked == true){
	    			$("#versionTable").datagrid("checkRow",i)
	    		}
	    	}
	    	if(nosenDisabledFlag == "1"){
	    		for(var i = 0;i<checkedrow.length;i++){
	    			$(".versionContainer .datagrid-body .datagrid-cell-check input[type='checkbox']")[i].disabled = true;
	    			$(".versionContainer .datagrid-header-row .datagrid-header-check input[type='checkbox']")[0].disabled = true;
		    	}
	    		
	    	}else{
	    		
	    	}
	    	
	    }

		</script>