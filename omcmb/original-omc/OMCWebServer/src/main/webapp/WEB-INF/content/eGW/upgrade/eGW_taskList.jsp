<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style type="text/css">
.textbox.combo{
	vertical-align:top;
}
.deviceLogContainer{
	width:90%;
	margin-left:60px;
	margin-top:20px;
}
.deviceListContainer,.selectedDevicesList{
	width:45%;
	height:100%;
	
}
.selectButtonContainer{
	width:6%;
	height:100%;
}
.deviceTitlt{
	color:#85A8BF;
	font-size:13px;
}
.flex-ctn{
	display: -webkit-flex;
	display: flex;
	flex-direction: column;
}
.flex-item {
	flex: auto;
	overflow: auto;
}
.titleButtonText {
	transition: opacity 0.5s ease-in;
}
.inputformat label{
	display:block;
	font-size:12px;
	color:#85A8BF;
}
.inputformat input{
	width:400px;
	height:25px;
	border:1px solid #85A8BF;
	display:block;
	margin-top:6px;
	margin-bottom:6px;
}
.inputformat span{
	color:#CC0000;
}
.tree-title{
	cursor:pointer;
}
.tree-icon{
	cursor:pointer;
}
.tree-checkbox:{
	cursor:pointer;
}
.showegwTaskOp{
	position:absolute;
	z-index:950;
	left:10px;
	width:160px;
	background:#fff;
	border:1px solid rgba(188,188,188,0.1);
	display:none;
	box-shadow:0 5px 15px #d8d8d8;
}
.showegwTaskOp div{
	height:40px;
	line-height:40px;
	padding-left:14px;
	display:block;
	width:104px;
	margin:0 !important;
	border-bottom:1px solid #e5f0f6;
	color:#000;
	background-position-x:15px !important;
	padding-right:42px;
}
.showegwTaskOp div:hover{
	background:#EDF6FF;
}
.showegwTaskOp  span{
	font-size:12px;
	margin-left:14px;
}
.showegwTaskOp div:last-of-type {
	border:none;
}
#egw_task_result{
	width:100%;
	height:300px;
	position:absolute;
	bottom:-350px;
	box-shadow:0 -8px 15px rgba(155,200,222,0.35);
	background:#fff;
	z-index:100;
}
</style>

<%--网关-升级--软件升级界面   --%>
<div class="panelDefault" style="overflow:hidden">
	 <!-- 新建任务按钮 -->
    <div class="addConfig circleIcon" id="op_add_div" style="display: none;"> 
    	<span class="circleBg add_circle" onclick="addegwUpgradeTask()"></span>
    	<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
    </div>
    <div class="addConfig circleIcon" id="op_close_div" style='right:15px;'> 
    	<span class="el-icon el-icon-circle-close" onclick="cancelAddegwTask()"></span>
    	<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
    </div>
    <div class="singleTitle">
		<span><%=rb.getString("RenWuLieBiao")%></span>
	</div>
	<div class="singleContentDiv" id='egwUpgradeTaskListDiv'>
		<table id="egwUpgradeListTable"></table>
	</div>
	<!-- 新建网关软件升级任务 -->
    <div id="addegwUpgradeTaskDiv" style="position:absolute;width:100%;;background:#FFFFFF;top:0px;z-index:50;left:0px;bottom:0;display:none;overflow:auto;"></div>   
	<!-- 查看任务结果信息 -->
	<div id="egw_task_result"></div>
</div>  
<%-- 升级任务工具栏 --%>
<div id="toolbar_egwUpgradeTaskList" class="toolbarContainer">
    <div class="queryGroup" style="margin-left:15px">
    	<input id="egwSearchTaskName" style="" placeholder="<%=rb.getString("RenWuMingCheng")%>">
		<b class='el-icon el-icon-common-search' onclick="queryegwUpgradeTaskList()"></b>
    </div>
</div>
<script type="text/javascript">
function hiddenEgwTable(){
	$('#egwUpgradeTaskListDiv').hide();
	addegwUpgradeTask();
}
$(function() {
    closeLoading();
    $("#egwUpgradeListTable").datagrid({
    	url:'${ctx}/egw/softwareUpgrade/querySoftwarePageList.action',
    	queryParams:{
    		taskName:$("#egwSearchTaskName").val(),
    		timeZone:timeZone},
    	fit:true,
    	fitColumns:true,
    	border:false,
    	singleSelect:true,
    	rownumbers:true,
    	striped:true,
    	pagination:true,
    	pagePosition:'bottom',
    	idField:'id',
    	toolbar:'#toolbar_egwUpgradeTaskList',
    	onBeforeLoad:egwUpgradeTaskListBeforeLoad,
    	onLoadSuccess:egwloadSuccessTaskList,
    	onLoadError:datagridLoadError,
    	columns: [[
    				{field: 'operation',width:30,fixed:true,styler:setStyle,formatter:egwUpgradeTaskFmt,title:''},
    				{field: 'id',hidden:true},
    				{field: 'task_name',width:200,title:'<%=rb.getString("RenWuMingCheng") %>'},
    				{field: 'file_name',width:270,title:'<%=rb.getString("WenJianMing") %>'},
    				{field: 'version',width:200,title:'<%=rb.getString("BanBen") %>'},
    				{field: 'product_type',width:100,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi") %>'},
    				{field: 'task_status',width:120,formatter:egwTaskStatusFmt,title:'<%=rb.getString("ZhuangTai") %>'},
    				{field: 'task_progress',width:130,title:'<%=rb.getString("JinDu") %>'},
    				{field: 'task_result',width:130,formatter:egwTaskResultFmt,title:'<%=rb.getString("JieGuo") %>'},
    				{field: 'start_time',width:170,title:'<%=rb.getString("KaiShiShiJian") %>'},
    				{field: 'stop_time',width:170,title:'<%=rb.getString("JieShuShiJian") %>'},
    			]]
    })
    $("#egwSearchTaskName").bind("keyup", function(e){
		if (e.keyCode == 13){
			$('#egwUpgradeListTable').datagrid('load');
		}
	});
    $(document).click(function(e){
        var e = e || window.event;
        var elem = e.target || e.srcElement;
        while(elem){
            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'circleBg add_circle' || elem.className == 'showegwTaskOp' || elem.className == 'slideDiv'){
                return
            } 
            elem = elem.parentNode;
        }
        $(".showegwTaskOp").css('display','none');
       /*  $("#egw_task_result").animate({bottom:'-350px'},400); */
    })
});

//新建网关升级任务
var addegwTaskFlag = true;
function addegwUpgradeTask(){
	if(addegwTaskFlag){
		var ctn = $('#op_add_div');
		$(".titleButtonText",ctn).html("<%=rb.getString("GuanBi")%>");
		$(".circleBg",ctn).removeClass("add_circle");
		$(".circleBg",ctn).addClass("close_circle");
		$("#egw_task_result").animate({bottom:'-350px'},400);
		var params = {
				timeZone:timeZone
		}
		$("#addegwUpgradeTaskDiv").slideDown(500,function(){
			$('#addegwUpgradeTaskDiv').load("${ctx}/egw/softwareUpgrade/toSoftwareUpgradeAdd.action",params,function(){
				$.parser.parse(this);
				closeLoading();
			})
		});
		addegwTaskFlag = false;
	}else{
		cancelAddegwTask();
	}
}
function cancelAddegwTask(){
	var ctn = $('#op_add_div');
	$(".titleButtonText",ctn).html("<%=rb.getString("TianJia")%>");
	$(".circleBg",ctn).addClass("add_circle");
	$(".circleBg",ctn).removeClass("close_circle");
	$("#addegwUpgradeTaskDiv").slideUp(500,function(){
		$("#addegwUpgradeTaskDiv").html("");
	});
	addegwTaskFlag = true;
	try{
		slideEGWDiv();
	}catch(e){}
}

function showTipCircle(ele){
		$(ele).next().css('opacity','1');
}
function hideTipCircle(ele){
		$(ele).next().css('opacity','0');
}
//设置操作列单元格样式 
function setStyle(){
	return 'position:relative';
}
function egwUpgradeTaskListBeforeLoad(param) {
	param["taskName"] = $("#egwSearchTaskName").val();
}
// 任务列表加载完成事件，如果有数据，则默认选中第一条数据 
function egwloadSuccessTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#egwUpgradeListTable").datagrid("selectRow", 0);
	}
}
/**
* 更多 数据格式化
* @param value{string} 绑定值
* @param rowData{object}  行数据
* @param rowIndex{number}  下标
*/
function egwUpgradeTaskFmt(value, rowData, rowIndex){
	//0  等待          1 进行中          2 已结束       3 终止中
	var task_id = rowData.id;
    var task_status = rowData.task_status;
	var JieGuo = '<%=rb.getString("JieGuo")%>';
	var JiHuo = '<%=rb.getString("JiHuo")%>';
	var GuaQi = '<%=rb.getString("GuaQi")%>';
	var ZhongZhi = '<%=rb.getString("ZhongZhi")%>';
	var ShanChu = '<%=rb.getString("ShanChu")%>';
	value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='choseOp("+ task_id +",this)'></div>";
	
	var opt="";
	opt = opt + "<div class='el-icon el-icon-operation-result' title='"+JieGuo+"' onclick='showUpgradeTaskDetail()'><span>"+JieGuo+"</span></div>";
	
	if(task_status == 0){//当前不是处于激活状态，且未结束，激活图标可用
		opt = opt + "<div class='el-icon el-icon-operation-start CODE_EGW hidden' title='"+JiHuo+"' onclick='activeUpgradeTask(\"" + task_id + "\")'><span>"+JiHuo+"</span></div>";
	}else if(task_status == 1){//当前不是处于挂起状态，且未结束，挂起图标可用
		opt = opt + "<div class='el-icon el-icon-operation-awaiting CODE_EGW hidden' title='"+GuaQi+"' onclick='suspendUpgradeTask(\"" + task_id + "\")'><span>"+GuaQi+"</span></div>";
	}else{
		opt = opt + "<div class='el-icon el-icon-operation-start disabled' title='"+JiHuo+"'><span>"+JiHuo+"</span></div>";
	}
 	if(task_status == 1 || task_status == 0){//当前任务没有结束，终止图标可用
		opt = opt + "<div class='el-icon el-icon-operation-terminate CODE_EGW hidden' title='"+ZhongZhi+"' onclick='terminateTask(\"" + task_id + "\")'><span>"+ZhongZhi+"</span></div>";
	}else{
		opt = opt + "<div class='el-icon el-icon-operation-terminate disabled' title='"+ZhongZhi+"'><span>"+ZhongZhi+"</span></div>";
	}
	
	if(task_status == 2||task_status == 0){//当前任务不在进行中，删除图标可用
		opt = opt + "<div class='el-icon el-icon-operation-delete CODE_EGW hidden' title='"+ShanChu+"' onclick='delUpgradeTask(\"" + task_id + "\")'><span>"+ShanChu+"</span></div>";
	}else{  
		opt = opt + "<div class='el-icon el-icon-operation-delete disabled' title='"+ShanChu+"'><span>"+ShanChu+"</span></div>";
	}
	
	value = value + "<div class='showegwTaskOp'>"+ opt +"</div>";
	return value;
}
//点击行内【更多】按钮，下拉显示操作选项 
function choseOp(idVal,e){
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	var indexRow = $("#egwUpgradeListTable").datagrid("getRowIndex",idVal);
	var rowHeight = $("#egwUpgradeTaskListDiv .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
	
	if((allHeight - thisTop) < 240){
		$(e).next(".showegwTaskOp").css("bottom",rowHeight+"px");
	}else{
		$(e).next().css("top",rowHeight+"px");
	}
	$(".showegwTaskOp").hide();
	$("#egw_task_result").animate({bottom:'-350px'},400);
	$(e).next().fadeToggle(300);
}
//任务状态格式化：激活/挂起
function egwTaskStatusFmt(value, rowData, rowIndex) {
	if (value == "0") {
		return "<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span>" + "<%=rb.getString("DengDai")%>";
	} else if (value == "1") {
		return "<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span>" + "<%=rb.getString("JinXingZhong")%>";
	} else if (value == "2") {
		return "<span class='el-icon el-icon-status-terminate' style='margin-right:5px;'></span>" + "<%=rb.getString("JieShu")%>";
	} else if (value == "3") {
		return "<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span>" + "<%=rb.getString("JinXingZhong")%>";
	}
}
//任务执行结果格式化
function egwTaskResultFmt(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("ChengGong")%>";
	} else if (value == "1") {
		return "<%=rb.getString("BuFenChengGong")%>";
	} else if (value == "2") {
		return "<%=rb.getString("ShiBai")%>";
	} else if (value == "3") {
		return "";
	} else {
		return "";
	}
}
function queryegwUpgradeTaskList(){
	$("#egwUpgradeListTable").datagrid('reload');
}
//显示任务进度
function showUpgradeTaskDetail() {
	$("#egw_task_result").animate({bottom:'0px'},400);
	$(".showegwTaskOp").slideUp(100);
    $("#egw_task_result").panel({
        href: "${ctx}/egw/softwareUpgrade/toSoftwareUpgradeView.action" 
    });
}
//删除任务
function delUpgradeTask(idVal){
	var params = {};
	params["taskId"] = idVal;
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
        if (r) {
            $.post("${ctx}/egw/softwareUpgrade/deleteTaskSoftware.action", params, function(data) {
                if (data["success"]) {
                    $("#egwUpgradeListTable").datagrid("reload");
                } else {
                	showMsg('error_msg',data["message"]);
                }
            }, "json");
        }
    }).addClass("seriousConfirm");
}
//终止任务
function terminateTask(idVal) {
	/* RenWuYiJieShu */
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/egw/softwareUpgrade/exeStopSoftware.action", params, function(data) {
		if (data["success"]) {
			$("#egwUpgradeListTable").datagrid("reload");
		}else{
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}
//激活任务
function activeUpgradeTask(idVal) {
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingLiJiZhiXingRenWu")%>", function(r) {
        if (r) {
        	var params = {};
        	params["taskId"] = idVal;
        	$.post("${ctx}/egw/softwareUpgrade/exeActiveSoftware.action", params, function(data) {
        		if (data["success"]) {
        			$("#egwUpgradeListTable").datagrid("reload");
        		} else {
        			showMsg('error_msg',data["message"]);
        		}
        	}, "json");
        }
 	}).addClass("normalConfirm");
}
//挂起任务
function suspendUpgradeTask(idVal) {
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/egw/softwareUpgrade/exeAwaitingStartSoftware.action", params, function(data) {
		if (data["success"]) {
			$("#egwUpgradeListTable").datagrid("reload");
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}
</script>