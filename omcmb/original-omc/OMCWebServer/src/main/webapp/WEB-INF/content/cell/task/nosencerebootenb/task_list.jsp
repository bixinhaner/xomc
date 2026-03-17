<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>
<style>
	.singleContentBox{
		position: absolute;
		top: 0px;
		right: 0px;
		bottom: 0px;
		left: 0px;
	}
	.boxHeader{
		height:40px;
		padding-top:15px;
	}
	.switch {
	    width:50px;
	    height:20px;
	    padding:2px;
	    border-radius: 30px;
	    -webkit-border-radius:30px;
	    -moz-border-radius:30px;
	    background-color: #838383;
	    position: relative;
	    display:inline-block;
	    margin-left: 10px;
	    float:right;
	}	

	.btnn {
	    width:20px;
	    height:20px;
	    -webkit-border-radius:30px;
	    -moz-border-radius:30px;
	    border-radius:30px;
	    background-color: #fff;
	    position: absolute;
	}
	.rightContainer{
		position:absolute;
		width:900px;
		top:0px;
		bottom:0px;
		right:-2000px;
		background:#ffffff;
		
		z-index:100;
		display:flex;
		flex-direction:column;
	}
	.addNoChangeHeader{
		flex-shrink:0;
		height:40px;
		line-height:40px;
		border:1px solid #D2D2D2;
		display:flex;
		justify-content:space-between;
		border-left:none;
		border-right:none;
		align-items:center;
		padding-left:20px;
		padding-right:15px;
	}
	.footercontainer{
		height:55px;
		line-height:40px;
		border:1px solid #D2D2D2;
		display:flex;
		justify-content:space-between;
		border-left:none;
		border-right:none;
		align-items:center;
		padding-left:20px;
		padding-right:15px;
		flex-shrink:0;
	}
.addNoChangeHeader span{
	font-size:14px;
	font-weight:bolder;
	color:#5A7B92;
	
}
	.propertiesDiv{
		margin:0px 40px 0px 40px;
		border-bottom:1px solid #DCECF7;
		padding-bottom:40px;
		
	}
	.propertiesDiv > p input{
		width:310px !important;
	}
	.viewDatagrid{
		width:800px;
		height:380px;
		margin:30px 0 0 54px;
	}
	.labelContainer{
		margin:20px 25px 20px 55px;
		display:flex;
		flex-direction:row;
		
	}
	.labelContainer > div{
		width:350px;
	}
	.labelContainer p{
		font-size:14px;
		color:#5A7B92;
		margin-bottom:5px;
	}
	.labelContainer input:not(.easyui-datebox){
		width:295px;
		height:30px;
		line-height:30px;
		font-size:14px;
		padding-left:5px;
		/* border-color:#85A8BF; */
	}
	.labelContainer input.easyui-datebox{
		width:270px;
		
	}
	.cutOffLine{
		height:1px;
		background:#E9E9E9;
		margin-bottom:30px;
	}
	.noSenseRebootIcon{
		cursor:pointer;
		vertical-align:middle;
		display: inline-block;
		height: 32px;
		width: 30px;
		background: url('${ctx}/css/images/newIcon/circleIcon/icon-silence-reboot.png') no-repeat center 4px;
	}
	 .el-icon-operation-result.center {
	 	text-align:center;
	 	width:inherit;
	 }
	.profileAddDiv .el-icon-status-timeOut:before { color: #19D5F3; }
</style>
<div>
	<div class="boxHeader" id="noSenseRebootTasl">
		<div style="display:inline-block;float:right">
			<span id="isUseTitle" style="display:inline-block;height:24px;line-height:24px;">启用中</span>
			<div class='switch' style='background-color:#66CC66;margin-right:60px;' onclick="setIsUseSwitch(this)">
				<div isopen='true' class='btnn' style='left:33px;'></div>
			</div>
	
			<div class="circleIcon placeholder-bt profileAddDiv" style="right: 20px;top:13px;cursor:pointer" placeholder="<%=rb.getString("XiuGai")%>">		
				<span class="el-icon el-icon-operation-reboot" onclick="openViewConfig()"></span>
				<span class='el-icon el-icon-status-timeOut' style='position: absolute; top: 8px; right: 0; font-size: 12px; '></span>
			</div>
		</div>
	</div>
	<div class="singleContentBox MMLTable">	
		<table class="easyui-datagrid" id="noSenceTaskList" fit="true" fitColumns="true" style="height:700px"
				data-options="singleSelect:true,rownumbers:true,pagination:true,border:false,striped:true,pagePosition:'bottom',toolbar:'#noSenseRebootTasl',
				url:'${ctx}/task/secretReboot/getRebootTaskList.action',onBeforeLoad:beforeLoad_nosenceTaskList,onLoadError:datagridLoadError,onLoadSuccess:loadSuccessMMLScriptTaskList,idField:'taskId'">
            <thead>
	            <tr>
	            	<th data-options="field:'operation',formatter : senceTaskFormatter,styler:setMMLStyle,fixed:true,fixed:true" width="70"><%=rb.getString("CaoZuo")%></th>
	                <th data-options="field:'taskId',hidden:true"></th>
	                <th data-options="field:'taskName'" width="120"><%=rb.getString("RenWuMingCheng")%></th>
	                <th data-options="field:'createUser'" width="100"><%=rb.getString("ChuangJianZhe")%></th>
	                <th data-options="field:'createTime'" width="110"><%=rb.getString("ChuangJianShiJian")%></th>
	                <th data-options="field:'taskStatus',formatter:taskTableStatus" width="70"><%=rb.getString("ZhuangTai")%></th>
	                <th data-options="field:'taskProgress'" width="70"><%=rb.getString("JinDu")%></th>
					<th data-options="field:'taskResult',formatter:taskTableResult" width="70"><%=rb.getString("JieGuo")%></th>
	                <th data-options="field:'startTime'" width="110"><%=rb.getString("KaiShiShiJian")%></th>
	                <th data-options="field:'endTime'" width="110"><%=rb.getString("JieShuShiJian")%></th>
	            </tr>
            </thead>
        </table>
	</div>
	<!-- 重启界面 -->
	<div id="rightContainerPanel" class="rightContainer"></div>
	<!-- 查看界面 -->
	<div id="bottomContainerPanel" class="bottomContainer"></div>
	<!-- 菜单生成 -->
<div class="wrap">
    <div id="mmlScriptMenu" class="showMMLOp"></div>
</div>
</div>

<script type="text/javascript">

var operatorCode = operator_code;
var timer_rebootTaskProgReload;
$(function(){
	//mockData为假设请求成功的值
	closeLoading();
	//周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_rebootTaskProgReload);
	timer_rebootTaskProgReload = undefined;
	var taskListUrl = '${ctx}/task/secretReboot/getRebootTaskList.action';
	timer_rebootTaskProgReload = setInterval("refreshnosemceTasklist('noSenceTaskList','"+taskListUrl+"','')", 6000);
	//页面加载完成  请求当前的状态 根据当前状态判断是查看还是编辑
	$.post("${ctx}/task/secretReboot/getSwitch.action",{operatorCode:operatorCode},function(data){
		 if(data.startFlag == "1"){
			
		}else{
			$(".switch").children().attr('isopen','false').animate({left:'1px'});
	        $(".switch").css('background-color','#838383');
	        $("#isUseTitle").text("停用中")
		} 
	},"json")
	
	$(document).click(function(e){
        var e = e || window.event;
        var elem = e.target || e.srcElement;
        while(elem){
            if(elem.className == 'operation_more' || elem.className == 'slideDiv'){
                return
            }
            elem = elem.parentNode;
        }
    	$('#mmlScriptMenu').hide();
    })
})
//设置是否开启停用
function setIsUseSwitch(e) {
	var params = {};
    if ($(e).children().attr('isopen') == 'false') {
        $(e).children().attr('isopen','true').animate({left:'33px'});
        $(e).css('background-color','#66CC66');
        $("#isUseTitle").text("启用中");
        params = {
        		operatorCode: operatorCode,
        		startFlag:1
        };
        $.post("${ctx}/task/secretReboot/setSwitch.action",params,function(data){
        },"json")
    } else {
    	$.messager.confirm('提示', '您确定要停用本次任务吗',function(r){
			if(!r) return;
			$(e).children().attr('isopen','false').animate({left:'1px'});
	        $(e).css('background-color','#838383');
	        $("#isUseTitle").text("停用中");
	        toast("停用成功",$('#mainpage'),'success');
	        params = {
	        		operatorCode: operatorCode,
	        		startFlag:0
	        };
	        $.post("${ctx}/task/secretReboot/setSwitch.action",params,function(data){
	    	 },"json")			
		}).addClass("seriousConfirm");
    	 
    }
   
   
}


//表格加载前事件
function beforeLoad_nosenceTaskList(param) {
	//添加查询条件
	param["timeZone"] = timeZone;
	param["operatorCode"] = operatorCode;
	
}
//任务列表加载完成，默认选中第一条数据
function loadSuccessMMLScriptTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#noSenceTaskList").datagrid("selectRow", 0);
	}
}
function senceTaskFormatter(value, rowData, rowIndex){	
	var task_progress = rowData.taskProgress;
	var task_status = rowData.taskStatus;
	var task_id = rowData.taskId;
	
	value = "<div class='el-icon el-icon-operation-result center' title='"+ CaoZuo+"' onclick='showMMLScriptTaskDetail("+ task_id + ")'></div>";
	return value;
}
function enbMmlscriptOp(taskId,task_status,el){
	var JieGuo = '<%=rb.getString("JieGuo")%>';
	var data = [
			{taskId:taskId, code: 'view', text: JieGuo,cls:' el-icon el-icon-operation-result'},
		];
	var menuCnt = $('#mmlScriptMenu');
	menuCnt.cmenu({data: data, click: mmlScriptOpClick}); 
	/* 菜单位置 */
	var allHeight = $(document).height(),
		isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
		tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0,
		thisTop = $(el).offset().top;
	if((allHeight - thisTop) <80){
		menuCnt.css({
			"top":thisTop - 142 - tabsHeight,
			"left":70,
		});
		if((allHeight - thisTop) <184) $('.item-child ').css({"top":"-54px",});
	}else{
		menuCnt.css({
			"top":thisTop - 60 - tabsHeight,
			"left":70,
		});
	}
	
	menuCnt.show();
}
function mmlScriptOpClick(row){
	var codes = {
			view: showMMLScriptTaskDetail,
		};
	
	if(codes[row.code]) codes[row.code](row.taskId);
	$('#mmlScriptMenu').hide();
}
function showMMLScriptTaskDetail(taskId) {
	var selectedTask = $("#noSenceTaskList").datagrid("getSelected");
	if (!selectedTask) {
		return;
	}
	var task_id = taskId;
	
	if($("#bottomContainerPanel").css('display') == 'block'){
	}else{		
		$("#bottomContainerPanel").show(400).fadeIn(400);
	}
	
	//$(".showMMLOp").slideUp(100);
    $("#rightContainerPanel").animate({
	    		right:"-2000px"
	 });
    $("#bottomContainerPanel").panel({
        href: '${ctx}/task/secretReboot/toRebootTaskProgress.action?taskId=' + task_id
    });
}
//设置操作列单元格样式 
function setMMLStyle(){
	return 'position:relative'
}
function openViewConfig (){
	$("#bottomContainerPanel").hide(400);
	$("#rightContainerPanel").animate({
		right:"0px"
	},function(){
		$("#rightContainerPanel").panel({
			width:900,
			href:"${ctx}/task/secretReboot/toAddTaskPage.action"
		});
	})
}
//定时刷新任务列表
function refreshnosemceTasklist(taskListId,url,typeId){
	if(typeId){
		var taskType = "";
			try{
				taskType = $("#"+typeId).combobox("getValue");
			} catch(e){
		}
	}
	var tableTaskList = $("#"+taskListId);
	if (tableTaskList.length > 0) {
		var pageNumber = $("#"+taskListId).datagrid('options').pageNumber;
		var pageSize = $("#"+taskListId).datagrid('options').pageSize;
		var param = {};
		param["timeZone"] = timeZone;
		param["operatorCode"] = operatorCode;
		param["page"]=pageNumber;
		param["rows"]=pageSize;
		var selectBefore = "";
		try{
			selectBefore = $("#"+taskListId).datagrid('getSelected');
		}catch(e){
			
		}
		var selectRowTaskId = "";
		if(selectBefore!=null){
			selectRowTaskId = selectBefore.taskId;
		}
		$.post(url, param,function (data) {
			$.each(data.rows,function(index,item){
				if(item.taskId == selectRowTaskId){
					$("#"+taskListId).datagrid('selectRow',index);
				}
				var index = $("#"+taskListId).datagrid('getRowIndex',item.taskId);
				$("#"+taskListId).datagrid('updateRow',{
					index:index,
					row:item
				})
			})
	    }, "json");
	}else{
		clearInterval(timer_rebootTaskProgReload);
	}
}
</script>