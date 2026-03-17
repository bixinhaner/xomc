<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style type="text/css">
.question-mark {
	background-image: url(${ctx}/css/images/global/question_mark.png);
	background-position: center center;
	display: inline-block;
	width: 40px;
	height: 40px;
	vertical-align: top;
	background-repeat: no-repeat;
}
.highQueryArrow span{
	vertical-align:super;
}
.taskListToorBarStyle{
	top:46px;
}
</style>

<%-- UPS 软件系统升级 升级任务列表 --%>
<div class="panelDefault">
	<!-- 右上角添加按钮 -->
	<div class="circleIcon" style="display: none;">
		<span class="import_circle circleBg" onclick="openWinAddTask()"></span>
		<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
	</div>
	
	<div class="circleIcon">
		<span class="el-icon el-icon-circle-close" onclick="closeUPSSlide()"></span>
		<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	</div>
	
	<div class="singleContentDiv" id="UPStaskTable">	
		<table class="easyui-datagrid" id="UPSupgradTaskList" fit="true"
				data-options="singleSelect:true,rownumbers:true,pagination:true,border:false,striped:true,pagePosition:'bottom',fitColumns:true,
				toolbar:'#toolbar_UPSupgradTaskList',url:'${ctx}/task/upgrade/ups/getUpgradeTaskList.action',onBeforeLoad:tableUpgradTaskListBeforeLoad,onLoadError:datagridLoadError,onLoadSuccess:loadSuccessUpgradTaskList,idField:'TASK_ID'">
            <thead>
	            <tr>
	             	<th data-options="field:'operation',formatter : upgradeTaskFormatter,styler:setCPEUpgradeStyle,fixed:true" width="30"></th>
	                <th data-options="field:'TASK_ID',hidden:true"></th>
	                <th data-options="field:'TYPE',hidden:true"></th>
	                <th data-options="field:'TASK_NAME'" width="200"><%=rb.getString("RenWuMingCheng")%></th>
	                <th data-options="field:'CREATE_USER'" width="100"><%=rb.getString("ChuangJianZhe")%></th>
	            	<th data-options="field:'CREATE_TIME'" width="150"><%=rb.getString("ChuangJianShiJian")%></th>
	                <th data-options="field:'FILE_NAME'" width="270"><%=rb.getString("WenJianMing")%></th>
	                <th data-options="field:'VERSION'" width="100"><%=rb.getString("BanBen")%></th>
	            	<th data-options="field:'TASK_STATUS',formatter:taskTableStatus" width="90"><%=rb.getString("ZhuangTai")%></th>
	                <th data-options="field:'TASK_PROGRESS'" width="70"><%=rb.getString("JinDu")%></th>
					<th data-options="field:'TASK_RESULT',formatter:taskTableResult" width="70"><%=rb.getString("JieGuo")%></th>
	                <th data-options="field:'START_TIME'" width="150"><%=rb.getString("KaiShiShiJian")%></th>
	                <th data-options="field:'END_TIME'" width="150"><%=rb.getString("JieShuShiJian")%></th>
	            </tr>
            </thead>
        </table>
	</div>	
	<!-- 查看任务结果信息 -->
	<div id="layout_center_progress_UPS" style="display:none;"></div>			
</div>

<%-- 升级任务工具栏 --%>
<div id="toolbar_UPSupgradTaskList" class="toolbarContainer">
	<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
    		name="operator_code" 
    		inputId="txtSearchUPSTaskName" targetId="upgradeUPSQueryDiv" 
    		placeholder="<%=rb.getString("RenWu")%><%=rb.getString("MingCheng")%>" 
    		data-options="query: function(){ vagueTaskSearchFun('UPSupgradTaskList','txtSearchUPSTaskName','upgrade_UPS_start_time','upgrade_UPS_end_time'); }"></div>
	<div id="upgradeUPSQueryDiv" class="taskListToorBarStyle" style="">
		    <ul class="inputslist" style="">
	          <li>
	              <label ><%=rb.getString("KaiShiShiJian")%><%=rb.getString("MaoHao")%></label><br>
	              <input name="" id="upgrade_UPS_start_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
	          </li>
	          <li>
	              <label ><%=rb.getString("JieShuShiJian")%><%=rb.getString("MaoHao")%></label><br>
	              <input name="" id="upgrade_UPS_end_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
	          </li>
		    </ul>
		    <div class="linkbuttonGroup" style="margin-bottom:20px">
		    	  <a href="#" class="linkbutton linkbutton_trend" onclick="accurateTaskSearchFun('UPSupgradTaskList','txtSearchUPSTaskName','upgrade_UPS_start_time','upgrade_UPS_end_time');upgradeCpeEnterEvent();"><span><%=rb.getString("ChaXun")%></span></a>
		    	  <a href="#" class="linkbutton linkbutton_nowanna" onclick="taskResetQueryInput('txtSearchUPSTaskName','upgrade_UPS_start_time','upgrade_UPS_end_time')"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
	     	</div>
	</div>
</div>

<!-- 菜单生成 -->
<div class="wrap">
    <div id="UPSSoftwareMenu" class="showOpUPS"></div>
</div>

<script type="text/javascript">

var timer_rebootTaskProgReload;
taskSearchText = '';
queryStartTime = '';
queryEndTime = '';

function closeUPSSlide() {
	try {
		slideUPSDiv();
	}catch(e){}
}

/* 显示高级查询选项  */
function upgradeUPSMoreQueryImgFun(){
	if($("#upgradeUPSMoreQueryImg").attr("flag")=="1"){
		$("#upgradeUPSQueryDiv").slideDown(500);
		$("#upgradeUPSMoreQueryImg").attr("flag","0");
		$("#upgradeUPSMoreQueryImg").addClass('expanded');
	}else{
		$("#upgradeUPSQueryDiv").slideUp(500);
		$("#upgradeUPSMoreQueryImg").attr("flag","1");
		$("#upgradeUPSMoreQueryImg").removeClass('expanded');
	}
	event.stopPropagation();
}
function upgradeCpeEnterEvent(){
	$("#upgradeUPSQueryDiv").slideUp(400);
	$("#upgradeUPSMoreQueryImg").removeClass('expanded');
    $("#upgradeUPSMoreQueryImg").attr("flag","1"); 
    $("#upgradeUPSHiddenSpan").hide();
    $("#txtSearchUPSTaskName").blur();
}

$(function() {
	closeLoading();
	//周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_rebootTaskProgReload);
	timer_rebootTaskProgReload = undefined;
	var taskListUrl = '${ctx}/task/upgrade/ups/getUpgradeTaskList.action';
	timer_rebootTaskProgReload = setInterval(function(){
		refreshTasklist('UPSupgradTaskList',taskListUrl);
		var dom = $("#UPSupgradTaskList");
		if(dom.length == 0) clearInterval(timer_rebootTaskProgReload);
	}, 6000);

	//点击页面其他位置，隐藏操作下拉选项菜单
	$(document).bind('click',function(e){
		var e = e || window.event;
		var elem = e.target || e.srcElement;
		while(elem){
			if($(elem).hasClass('el-icon-operation-more') || elem.className == 'add_circle circleBg' || elem.className == 'showOpUPS' || elem.className == 'slideDiv'){
				return;
			}
			elem = elem.parentNode;
		}
		//$(".showOpUPS").css('display','none');
		$("#choseListCPE").css('display','none');
		$("#layout_center_progress_UPS").hide(400);
		$('#UPSSoftwareMenu').hide();
	})
	$("#txtSearchUPSTaskName").bind("keyup", function(e){
		if (e.keyCode == 13){
			taskSearchText = $('#txtSearchUPSTaskName').val();
			queryStartTime = "";
			queryEndTime = "";
			$("#upgrade_UPS_start_time").datetimebox('setValue', null);
			$("#upgrade_UPS_end_time").datetimebox('setValue', null);
			upgradeCpeEnterEvent();
			$('#UPSupgradTaskList').datagrid('load');
		}
	});
	
	
	$("#UPSupgradTaskList").datagrid({
        queryParams:{timeZone:timeZone}
    });
});

//设置操作列单元格样式 
function setCPEUpgradeStyle(){
	return "position:relative";
}

/**
 * 点击行内【更多】按钮，下拉显示操作选项 
 * @param idval:传入值 生成标签
*/
function choseOp(idVal,e){
	$("#layout_center_progress_UPS").hide(400);
	$("#choseListCPE").css('display','none');
	$(".showOpUPS").hide();
	$(e).next().fadeToggle(300);
	var rows = $("#UPSupgradTaskList").datagrid("getData").rows;
	var length = rows.length;
	var indexRow;
	for(var i=0;i<length;i++){
		if(rows[i]['TASK_ID']==idVal){
			indexRow = i;
			break;
		}
	}
	
	var thisRow = $("#UPStaskTable .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]");
	var rowHeight = $("#UPStaskTable .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	
	if((allHeight - thisTop) < 240){
		$(e).next(".showOpUPS").css("bottom",rowHeight+"px");
	}else{
		$(e).next().css("top",rowHeight+"px");
	}
}

/**
 * 格式化操作
 * @return value:生成的dom
 * @param rowData:传入data值
 */
function upgradeTaskFormatter(value, rowData, rowIndex){
	
	var task_progress = rowData.TASK_PROGRESS;
	var task_status = rowData.TASK_STATUS;
	var task_id = rowData.TASK_ID;
	var task_type = rowData.TYPE;
	
	/* 加入升级文件类型判断 */
	var typeCls = {
			1:'eNbSoftwareUpdrade',
			2:'eNbSoftwareRoolback',
			3:'eNbUbootUpdrade',
			4:'eNbPatchUpdrade'
		},
		softwareCls = typeCls[rowData.TYPE];
	
	value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='cpeSoftwareOp("+ task_id +",\""+task_status+"\",\""+softwareCls+"\",\""+task_type+"\",this)'></div>";
	return value;
}

/**
 * 菜单生成DOm
 * @param 参数是当前数据的参数
*/
function cpeSoftwareOp(taskId,task_status,cls,type,el){
	var JieGuo = '<%=rb.getString("JieGuo")%>',
	 	KaiShi = '<%=rb.getString("KaiShi")%>',
	 	ZanTing = '<%=rb.getString("ZanTing")%>',
	 	ZhongZhi = '<%=rb.getString("ZhongZhi")%>',
	 	ShanChu = '<%=rb.getString("ShanChu")%>';
	var data = [
		{taskId:taskId, type: type, code: 'view', text: JieGuo},
		{taskId:taskId, type: type, code: 'start', text: KaiShi,cls:'UPS hidden'},
		{taskId:taskId, type: type, code: 'wait', text: ZanTing,cls:'UPS hidden'},
		{taskId:taskId, type: type, code: 'end', text: ZhongZhi,cls:'UPS hidden'},
		{taskId:taskId, type: type, code: 'del', text: ShanChu,cls:'UPS hidden'}
		];
	initTaskStatus(task_status,data);
	$('#UPSSoftwareMenu').cmenu({data: data, click: enbSoftwareOpClick}); 
	/* 菜单位置 */
	var allHeight = $(document).height(),
		isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
		tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0,
		thisTop = $(el).offset().top;
	if((allHeight - thisTop) <200){
		$('#UPSSoftwareMenu').css({
			"top":thisTop - 179 - tabsHeight,
			"left":70,
		});
		if((allHeight - thisTop) <184) $('.item-child ').css({"top":"-54px",});
	}else{
		$('#UPSSoftwareMenu').css({
			"top":thisTop - 20 - tabsHeight,
			"left":30,
		});
	}
	
	$('#UPSSoftwareMenu').show();
}
/**
 * 点击菜单按钮
 * @param row:当前传入的 点击哪个菜单
*/
function enbSoftwareOpClick(row){
	var codes = {
			view: showUpgradeTaskDetail,
			start: activeUpgradeTask,
			wait: suspendUpgradeTask,
			end: terminateTask,
			del: delUpgradeTask
		};
	    
	if(codes[row.code]) codes[row.code](row.taskId,row.type);
	$('#UPSSoftwareMenu').hide();
}
// 显示任务进度
function showUpgradeTaskDetail() {
	var selectedTask = $("#UPSupgradTaskList").datagrid("getSelected");
	var  task_id = selectedTask["TASK_ID"];
	if("undefined"==typeof(task_id)){
		return;
	} 
	
	if($("#layout_center_progress_UPS").css('display') == 'block'){
	
	}else{		
		$("#layout_center_progress_UPS").show(400).fadeIn(400);
	}
	//$(".showOpUPS").slideUp(100);
	
    $("#layout_center_progress_UPS").panel({
         href: '${ctx}/task/upgrade/ups/toUpgradeTaskProgress.action?task_id=' + task_id
    });
}

/**
 * 删除任务
 * @param idVal:当前数据ID
 * @param type:类型
*/
function delUpgradeTask(idVal,type){
	var params = {};
	params["taskId"] = idVal;
	params["type"] = type;
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
        if (r) {
            $.post('${ctx}/task/upgrade/ups/delUpgradeTask.action', params, function(data) {
                if (data["success"]) {
                	 showMsg('success_msg','<%=rb.getString("ChengGong")%>');
                    $("#UPSupgradTaskList").datagrid("reload");
                } else {
                    showMsg('error_msg',data["message"]);
                }
            }, "json");
        }
    }).addClass('seriousConfirm');
}

// 任务列表加载完成事件，如果有数据，则默认选中第一条数据 
function loadSuccessUpgradTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#UPSupgradTaskList").datagrid("selectRow", 0);
	}
}

// 打开新建任务窗口
function openWinAddTask() {
	var width = "${width}";
	var urlOduSoftWare = "${ctx}/task/upgrade/ups/goAddTask.action";
	var titleOduSoftWare =  "<%=rb.getString("XinJianShengJiRenWu")%>";
	
   	 url = urlOduSoftWare; 
   	/* title = titleOduSoftWare;
     var options = {title: title,
 		    		width: width,
 		        	height: document.body.clientHeight * 0.9}
     openDefaultWindow(url,options); */
     
     var slider = $("#winAddUpgradTask");
     slider.html('');
     slider.addClass('loading').slideDown(function(){
     	slider.load(url,function(html){
     		slider.removeClass('loading');
     		$.parser.parse(slider);
     	})
     })
}

// 添加任务成功后执行
function callbackForAddTask() {
	/* $("#winAddUpgradTask").window("close"); */
	closeDefaultWindow();
    $("#UPSupgradTaskList").datagrid("reload");
}

/**
 * 终止任务
 * @param idVal:当前数据ID
 * @param type:类型
*/
function terminateTask(idVal,type) {
	/* RenWuYiJieShu */
	var params = {};
	params["taskId"] = idVal;
	params["type"] = type;
	
	$.post("${ctx}/task/upgrade/ups/terminateUpgradeTask.action", params, function(data) {
		if (data["success"]) {
			$("#UPSupgradTaskList").datagrid("reload");
		}else{
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

/**
 * 激活任务
 * @param idVal:当前数据ID
 * @param type:类型
*/
function activeUpgradeTask(idVal,type) {
	//$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingLiJiZhiXingRenWu")%>", function(r) {
        //if (r) {
        	var params = {};
        	params["taskId"] = idVal;
        	params["type"] = type;
        	$.post("${ctx}/task/upgrade/ups/activeTask.action", params, function(data) {
        		if (data["success"]) {
        			$("#UPSupgradTaskList").datagrid("reload");
        		} else {
        			showMsg('error_msg',data["message"]);
        		}
        	}, "json");
        //}
 	//}).addClass('normalConfirm');
}

/**
 * 挂起任务
 * @param idVal:当前数据ID
 * @param type:类型
*/
function suspendUpgradeTask(idVal,type) {
	var params = {};
	params["taskId"] = idVal;
	params["type"] = type;
	$.post("${ctx}/task/upgrade/ups/suspendTask.action", params, function(data) {
		if (data["success"]) {
			$("#UPSupgradTaskList").datagrid("reload");
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

// 任务类型树，节点点击事件
function typeTreeOnClick(node) {
	$("#centerPanel_versionMgr").panel({
		href: "${ctx}" + node.attr("url")
	});
}
/**
 * 提示信息
 * param:传入参数判断是否需要提示信息
*/
function tableUpgradTaskListBeforeLoad(param) {
	var taskName = taskSearchText;
	if (taskName) {
		param["taskName"] = taskName;
	}
	param["startTime"] = queryStartTime;
	param["endTime"] = queryEndTime;
	
}
// 没发现用到的地方
function queryUpgradeTaskList(){	
	$("#UPSupgradTaskList").datagrid('load');
}

</script>