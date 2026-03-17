<%@ page import="java.util.Locale" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>
<style>
	.containerBox{
		margin-left:50px;
	}
	.slidebarPanel .group-title{
		margin-left:10px;
	}
	.taskTableContainer{
		width: 600px;
	}
	.sureBtn{
		position:absolute;
		width:40px;
		height:25px;
		background:#1DA3FC;
		display:inlne-block;
		top:15px;
		z-index: 600;
		right:20px;
		text-align:center;
		line-height:25px;
		color:#FFFFFF;
		box-shadow:2px 4px 15px 0 rgba(130,193,234,0.45);
		cursor:pointer;
	}
</style>

<div style='display:flex;flex-direction:column;height:100%;'>
	<div class='el-card__header'>
		<span><%=rb.getString("EPCGenZongRenWuChuangJian")%></span>
		<span class="el-icon el-icon-close slideIcon" onclick='closeNewEPCTrace()'>
		</span>
	</div>
	<!-- <空白填充区域> -->
	<!-- <div style="width:924px;height:50px;"></div> -->
	<div class='el-card__body'>
		<!-- 信息填写部分 -->
		<div class="taskStatusContainer" style="position:relative;overflow:auto;background:#fff;margin-top:20px;margin-left:20px;">
			<!-- 基本信息 -->
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
			</div>
			<div class="containerBox defaultInput">
				<p style="height:82px">
					<label><%=rb.getString("GenZongMingCheng")%></label>
					<input id="epcTaskTraceName" type="text" name="traceName" onblur="testTraceName(this)"/>
					<span style="display:none;height:20px;line-height:20px;color:red;"><%=rb.getString("GenZongMingChengBuNengWeiKong")%></span>
				</p>
				<p style="margin-right:0px">
					<label><%=rb.getString("GenZongCanKaoHao")%></label>
					<input id="epcTaskTraceId" type="text" name="traceId" disabled="disabled"/>
				</p>
				<p style="margin-right:0px">
					<label><%=rb.getString("BeiZhuXinXi")%></label>
					<textarea id="epcTaskRemarks" style="resize:none;width:350px;height:90px;border-color:#DEDFE6;"></textarea>
				</p>
			</div>
			
			<!-- 跟踪配置 -->
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("GenZongPeiZhi")%></span>
			</div>
			
			<div class="containerBox defaultInput">
				<div style="margin-bottom:25px;">
					<p>
						<label><%=rb.getString("GenZongSheBei")%></label>
						<input class="showEPCTraceName" style="background:#F5F7FA;" id="showEPCTraceName" type="text" disabled="disabled" name="traceName" />
						<span class="showDevice" style="cursor:pointer;" onclick="openChoseEPCTable()"></span>
					</p>
					
					<div class="taskTableContainer" id="EPCtaskTableContainer">
						<table id="traceTaskEPCTable"></table>
						<span class="sureBtn" onclick="choseEPCDevice()"><%=rb.getString("QueDing")%></span>
					
					</div>
				</div>
					
				<p style="margin-right:0px;margin-bottom:35px;height:82px">
					<label>IMSI</label>
					<input id="EPCTraceIMSI" type="text" name="traceIMSI" onblur="testIMSI(this)"/>
					<span style="display:none;height:20px;line-height:20px;color:red;"><%=rb.getString("LGWImsiChangDuCuoWu")%></span>
				</p>
				<!-- <div style="display:block;margin-bottom:35px;">
					<label style="display:block;line-height:36px;font-size:12px;">协议类型</label>
					<div style="width:350px;height:21px;padding-top:5px;border:1px solid #DEDFE6;">
						<input class="alignCenter" id="s1protocol" type="checkbox" name="protocolType" value="S1" style="margin-left:7px;">
						<label style="margin-right:55px;" for="s1protocol">s1</label>
						<input class="alignCenter" id="s6aprotocol" type="checkbox" name="protocolType" value="S6a">
						<label for="s6aprotocol">s6a</label>
					</div>
				</div> -->
				<!-- <div style="display:block;">
					<label style="display:block;line-height:36px;font-size:12px;">网元类型</label>
					<div style="width:350px;height:21px;padding-top:5px;border:1px solid #DEDFE6;">
						<input class="alignCenter" id="mmeNEType" type="checkbox" name="NEType" value="MME" style="margin-left:7px;">
						<label style="margin-right:55px;" for="mmeNEType">MME</label>
						<input class="alignCenter" id="hssNEType" type="checkbox" name="NEType" value="HSS">
						<label for="hssNEType">HSS</label>
					</div>
				</div> -->
					
				<div class="containerBox defaultInput" style="border-bottom:none;margin-bottom:0px;margin-left:0px;padding-bottom:0px">
					<div style="display:block;margin-bottom:0px;">
						<label style="display:block;line-height:36px;font-size:12px;"><%=rb.getString("JieKouLeiXing")%></label>
						<div class="EPCInterface" style="width:600px;height:105px;padding-top:10px;border:1px solid #DEDFE6;">
							<div class="EPCChoseMME" style="width:350px;margin-bottom:30px;height:21px;padding-top:5px;">
								<label style="margin-right:20px;margin-left:10px" for="mmeNEType">MME:</label>
								<input class="alignCenter" id="EPCS1" type="checkbox" name="MMEType" value="S1" onclick="testEpcInterBox(this)">
								<label style="margin-right:55px;" for="EPCS1">S1</label>
								<input class="alignCenter" id="EPCS6a" type="checkbox" name="MMEType" value="HSS" onclick="testEpcInterBox(this)">
								<label for="EPCS6a">S6a</label>
							</div>
							<div class="EPCChoseHSS" style="width:350px;height:21px;padding-top:5px;">
								<label style="margin-right:26px;margin-left:10px" for="mmeNEType">HSS:</label>
								<input class="alignCenter" id="HssEPCS6a" type="checkbox" name="HSSType" value="S6a" onclick="testEpcInterBox(this)">
								<label style="margin-right:55px;" for="HssEPCS6a">S6a</label>
							</div>
						</div>
					</div>
				</div>
			</div>
			
			<!-- 执行方式 -->
			<div class="group-title not-extend">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
			</div>
			<div class="containerBox defaultInput" style="border-bottom:none;margin-bottom:0px;padding-bottom:0px">
				<div style="display:block;margin-bottom:0px;">
					<label style="display:block;line-height:36px;font-size:12px;"><%=rb.getString("ZhiXingFangShi")%></label>
					<div id="epcStartWay" style="width:675px;height:50px;padding-top:10px;border:1px solid #DEDFE6;">
						<input class="alignCenter" id="immediatelyWay"  checked="true" type="radio" name="doWay" value="immediately" style="margin-left:22px;" onclick="testradioInputTimeBox(this)">
						<label style="margin-right:55px;" for="immediatelyWay"><%=rb.getString("LiJiZhiXing")%></label>
						<input class="alignCenter" id="hangUpWay" type="radio" name="doWay" value="hangUp" onclick="testradioInputTimeBox(this)">
						<label style="margin-right:55px;" for="hangUpWay"><%=rb.getString("GuaQi")%></label>
						<input class="alignCenter" id="setTimeWay" type="radio" name="doWay" value="schetime" onclick="testInputBox(this)">
						<label for="setTimeWay"><%=rb.getString("DingShiZhiXing")%></label>
						<div style="margin-top:10px;display:inline-block;margin-left:22px;height:26px;width:200px;">	    
		               		<input id="EPCSetTime" class="easyui-datetimebox border-box border"  data-options="require:true,editable:false" name="scheduleStart" style="height:26px;width:200px;line-height:26px;"/>   
		                </div>
					</div>
				</div>
					
				<p style="margin-right:0px;margin-bottom:35px;">
					<label><%=rb.getString("ChiXuShiChang")%></label>
					<input id="EPCDuration" type="text" name="traceIMSI" onblur="testTraceTime(this)"/>
					<span style="display:block;height:20px;line-height:20px;color:red;display:none"><%=rb.getString("XinLingZhuiZongZuiDaShiChang")%></span>
				</p>
			</div>
		</div>
	</div>
	
	<!-- 按钮 -->
	<div class='slideFooter'>
        	<a href="#" class="linkbutton linkbutton_trend" onclick="saveNewEPCTraceTask()"><span><%=rb.getString("QueDing")%></span></a>
        	<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeNewEPCTrace()"><span><%=rb.getString("QuXiao")%></span></a>
       	<div id="saveNewTraceEPCBtn" style="display:none" class="newTraceTaskBtn"><%=rb.getString("XinJianEPCGenZongRenWuChengGong")%></div>
	</div>
</div>
<!-- 表格工具栏 -->
<div id="toolbar_EPCtableTraceList" class="toolbarContainer defaultQuery" style="position:relative">
	<div class='queryGroup'>
		<input style="margin-left:0px;width:220px" id="EPCtraceSearchText" placeholder='<%=rb.getString("EPCMingChen")%> / IP' name="searchText" class="searchInputStyle faultListInput"/>
		<b class="searchResultImgChangeStyle el-icon el-icon-common-search" onclick="$('#traceTaskEPCTable').datagrid('reload')"></b>
	</div>
</div>
<script type="text/javascript">
var epcTraceID = "${traceId}";
var epcTaskName = "${taskName}";
var EPCTraceDevice = "";
var EPCTraceDeviceId = "";
var signalingAddr = "";
$(function(){
	$("#epcTaskTraceId").val(epcTraceID);
	$("#epcTaskTraceName").val(epcTaskName);
	$(document).click(function(evt){
		 var e = evt || window.event;
	     var elem = e.target || e.srcElement;
         while(elem){
             if(elem.className == 'searchInputStyle faultListInput' || elem.className == 'searchResultImgChangeStyle' || elem.className == 'taskTableContainer' || elem.className == 'showDevice' || elem.className == 'showEPCTraceName' || elem.className == 'datagrid-row' || elem.className == 'datagrid-row' || elem.className == 'datagrid-body' || elem.className == 'datagrid-view' || elem.className == 'l-btn-left l-btn-icon-left'){
                 return
             }
             elem = elem.parentNode;
            
         }
         if($(e.target).closest(".taskTableContainer").length==0 && $(e.target).closest(".window-mask").length==0
      			 &&$(e.target).closest(".messager-window").length==0){
        	 $("#EPCtaskTableContainer").slideUp(300);
         }
         
	});
	setTimeout(function(){
		$("#EPCSetTime").datebox({
			onChange:function(){
				$("#EPCSetTime").next().css({"border-color":"#DEDFE6"});
			}
		})
	},0);
	$("#traceTaskEPCTable").datagrid({
		url:'${ctx}/signaling/queryEPCTraceDevice.action',
		border:true,
		fit:true,
		queryParams:{
			timeZone:timeZone,
			likeFields:'NAME,IP' 
		},
		rownumbers:true,
		fitColumns:true,
		striped:true,
		singleSelect:true,
		idField:'EPC_ID',
		toolbar:'#toolbar_EPCtableTraceList',
		pagination:true,
		pagePosition:'bottom', 
		columns:[[
			   {field:'radio',formatter:radioFormatter,fixed:true,width:50, title: ''},
	    	   {field:'EPC_ID',fixed:true,hidden:true,width:120, title: '设备ID'},
	    	   {field:'NAME',fixed:false,sortable:true,width:120, title: '设备名称'},
	    	   {field:'IP',fixed:true,sortable:true,width:150, title: 'IP'},
	    	   {field:'PORT',fixed:true,hidden:true,width:150, title: 'IP'},
	    	   
		]],
	   onBeforeLoad:beforeLoad_addEPCTask, 
	   onClickRow:clickEPCRadio, 
	   onLoadSuccess:loadSuccessNewEpcTask 
	});
})

function saveNewEPCTraceTask(){
	var saveEpcParams = {};
	//traceName
	var traceNameval = $("#epcTaskTraceName").val();
	var traceNameEle = $("#epcTaskTraceName");
	if(testTraceName(traceNameEle)){
		$("#epcTaskTraceName").css("borderColor","#DEDFE6");
		saveEpcParams.traceName = traceNameval
	}else{
		$("#epcTaskTraceName").css("borderColor","red");
		return;
	};
	//traceId
	saveEpcParams.traceId = $("#epcTaskTraceId").val();
	saveEpcParams.timeZone = timeZone;
	saveEpcParams.remarks = $("#epcTaskRemarks").val()
	saveEpcParams.signalingAddr = signalingAddr;
	saveEpcParams.traceNetworkElement = "EPC";
	var traceDevice = $("#showEPCTraceName").val() ;
	if(traceDevice == ""){
		$("#showEPCTraceName").css("borderColor","red");
		return;
	}else{
		$("#showEPCTraceName").css("borderColor","#DEDFE6");
		saveEpcParams.traceDevice = traceDevice;
	}
	var EPCIMSI = $("#EPCTraceIMSI");
	
	if(testIMSI(EPCIMSI)){
		saveEpcParams.imsi = EPCIMSI.val();
		$("#EPCTraceIMSI").css("borderColor","#DEDFE6");
		$("#EPCTraceIMSI").next().css("display","none");
	}else{
		$("#EPCTraceIMSI").css("borderColor","red");
		$("#EPCTraceIMSI").next().css("display","block");
		return ;
	}
	//NEInterface
	var HSSVal = "";
	var NEInterfaceStr = "";
	var MMEVal = "";
	var hasCheckedLen = $(".EPCInterface input[type='checkbox']:checked").length;
	if(hasCheckedLen == 0){
		$(".EPCInterface").css("borderColor","red")
		return ;
	}else{
		$(".EPCInterface").css("borderColor","#DEDFE6");
		if($("#EPCS1").prop("checked") || $("#EPCS6a").prop("checked")){
			MMEVal = "MME";
			if($("#EPCS1").prop("checked") && !$("#EPCS6a").prop("checked")){
				MMEVal = MMEVal + ":" + "S1"
			}else if(!$("#EPCS1").prop("checked") && $("#EPCS6a").prop("checked")){
				MMEVal = MMEVal + ":" + "S6a"
			}else if($("#EPCS1").prop("checked") && $("#EPCS6a").prop("checked")){
				MMEVal = MMEVal + ":" + "S1" + "," + "S6a" 
			}else{
				MMEVal = "";
			}
		}
		if($("#HssEPCS6a").prop("checked")){
			HSSVal = "HSS" + ":" + "S6a";
		}else{
			HSSVal = "";
		};
		if(MMEVal != "" && HSSVal !== ""){
			saveEpcParams.NEInterface = MMEVal + "|" +HSSVal;
		}else if(MMEVal == "" && HSSVal !== ""){
			saveEpcParams.NEInterface = HSSVal;
		}else if(MMEVal != "" && HSSVal == ""){
			saveEpcParams.NEInterface = MMEVal;
		}else{
			$(".EPCInterface").css("borderColor","red")
			return ;
		}
	}
	//执行方式
	var strStartTime = $("#EPCSetTime").combobox('getValue');
	var selectVal = $("#epcStartWay input:radio[name='doWay']:checked").val();
	var selectValLen = $("#epcStartWay input:radio[name='doWay']:checked").length;
	if(selectValLen == ""){
		$("#epcStartWay").css("borderColor","red");
		return;
	}else{
		$("#epcStartWay").css("borderColor","#DEDFE6");
		if(selectVal == "immediately"){
			saveEpcParams.executeMode = "1";
		}else if(selectVal == "hangUp"){
			saveEpcParams.executeMode = "2";
		}else if(selectVal == "schetime"){
			saveEpcParams.executeMode = "3";
			if(strStartTime == ""){
				$("#EPCSetTime").next().css({"border-color":"red"});
				return ;
			}else{
				$("#EPCSetTime").next().css({"border-color":"#DEDFE6"});
				saveEpcParams.startTime = strStartTime;
			}
		}
	}
	//标识
	saveEpcParams.identificationInfo = "IMSI"+"="+EPCIMSI.val();
	saveEpcParams.epcId = EPCTraceDeviceId;
	//持续时长
	var durationTimeVal = $("#EPCDuration");
	if(testTraceTime(durationTimeVal)){
		saveEpcParams.duration = durationTimeVal.val();
		$("#EPCDuration").css("borderColor","#DEDFE6");
	}else{
		$("#EPCDuration").css("borderColor","red");
		return ;
	}
	$.post("${ctx}/signaling/addEPCSignalingTraceTask.action",saveEpcParams,function(data){
		if(data["success"]){
			$("#saveNewTraceEPCBtn").text('<%=rb.getString("XinJianRenWuChengGong")%>');
			$("#saveNewTraceEPCBtn").fadeIn(300,function(){
				setTimeout(function(){
					$("#saveNewTraceEPCBtn").fadeOut(300,function(){
						$("#newEPCTraceContainer").animate({"right":"-2000px"},function(){
							$("#tableTraceList").datagrid("reload");
						})
					})
				},500)
			})
		}else{
			showMsg('error_msg',data["message"]);
		}
	},"json")
}
//选择EPC 设备
function openChoseEPCTable(){
	$("#EPCtaskTableContainer").slideDown(300,function(){
		$("#traceTaskEPCTable").datagrid({
			url:'${ctx}/signaling/queryEPCTraceDevice.action',
			border:true,
			fit:true,
			queryParams:{
				timeZone:timeZone,
				likeFields:'NAME,IP' 
			},
			rownumbers:true,
			fitColumns:true,
			striped:true,
			singleSelect:true,
			idField:'EPC_ID',
			toolbar:'#toolbar_EPCtableTraceList',
			pagination:true,
			pagePosition:'bottom', 
			columns:[[
				   {field:'radio',formatter:radioFormatter,fixed:true,width:50, title: ''},
		    	   {field:'EPC_ID',fixed:true,hidden:true,width:120, title: '设备ID'},
		    	   {field:'NAME',fixed:false,sortable:true,width:120, title: '设备名称'},
		    	   {field:'IP',fixed:true,sortable:true,width:150, title: 'IP'},
		    	   {field:'PORT',fixed:true,hidden:true,width:150, title: 'IP'},
		    	   
			]],
		   onBeforeLoad:beforeLoad_addEPCTask, 
		   onClickRow:clickEPCRadio, 
		   onLoadSuccess:loadSuccessNewEpcTask 
		});
	});
	
	
}
function clickEPCRadio(rowIndex,rowData){
	var deviceName = rowData.NAME?rowData.NAME:'';
	var ipAddress = rowData.IP;
	var epcId = rowData.EPC_ID;
	var port = rowData.PORT;
	var showVal = deviceName+'('+ipAddress+')';
	var radio=$(".taskTableContainer input[type='radio']")[rowIndex].disabled;
	if(radio != true){
		$("#EPCtaskTableContainer input[type='radio']")[rowIndex].checked = true;
	}else{
		$("#EPCtaskTableContainer input[type='radio']")[rowIndex].checked = false;
	}
	EPCTraceDeviceId = epcId;
	EPCTraceDevice = showVal;
	signalingAddr = ipAddress + ":" + port;
}
function choseEPCDevice(){
	if(EPCTraceDevice == ""){
		$("#showEPCTraceName").css("borderColor","red");
	}else{
		$("#showEPCTraceName").css("borderColor","#DEDFE6");
		$("#showEPCTraceName").val(EPCTraceDevice);
		$("#EPCtaskTableContainer").slideUp(300);
	}
	
}
function loadSuccessNewEpcTask(data){
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	var allRows = data.rows;
	var hasChoseEpc = EPCTraceDevice;
	if(hasChoseEpc == ""){
		
	}else{
		/* hasChoseEpc = hasChoseEpc.substring(0,hasChoseEpc.length-1); */
		var hasChoseEpcArr = []; 
		hasChoseEpcArr = hasChoseEpc.split("(");
		var hasChoseEpcName = hasChoseEpcArr[0];
		for(var i = 0;i<allRows.length;i++){
			if(hasChoseEpcName == allRows[i].NAME){
				$("#EPCtaskTableContainer input[type='radio']")[i].checked = true;
			}
		}
	}
}

function beforeLoad_addEPCTask(params){
	var searchCellCode = $("#EPCtraceSearchText").val();
	if (searchCellCode) {
		params["searchText"] = searchCellCode;
	}
}
function testEpcInterBox(e){
	$(e).parents(".EPCInterface").css("borderColor","#DEDFE6");
}
</script>
