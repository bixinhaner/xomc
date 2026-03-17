<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.highQueryArrow span{
		vertical-align:super;
	}
	.showMMLOp {
		position:fixed;
	}
</style>
<%-- MML脚本任务 --%>
<div class="panelDefault" style="border:none;">
	<!-- 右上角添加按钮 -->
	<div class="circleIcon placeholder-bt CODE_ENB_MML hidden" placeholder="<%=rb.getString("TianJia")%>" onclick="openWinAddMMLScriptTask()" style='top:0px;'>
		<span class="el-icon el-icon-circle-add"></span>
    </div>

	<div class="singleContentDiv MMLTable" style='top:-20px;'>	
		<table class="easyui-datagrid" id="MMLScriptTaskList" fit="true" fitColumns="true"
				data-options="singleSelect:true,rownumbers:true,pagination:true,border:false,striped:true,pagePosition:'bottom',toolbar:'#toolbar_MMLScriptTaskList'">
            <thead>
	            <tr>
	            	<th data-options="field:'operation',formatter : MMLTaskFormatter,styler:setMMLStyle,fixed:true" width="30"></th>
	                <th data-options="field:'TASK_ID',hidden:true"></th>
	                <th data-options="field:'TASK_NAME'" width="100"><%=rb.getString("RenWuMingCheng")%></th>
	                <th data-options="field:'CREATE_USER'" width="100"><%=rb.getString("ChuangJianZhe")%></th>
	                <th data-options="field:'CREATE_TIME'" width="110"><%=rb.getString("ChuangJianShiJian")%></th>
	                <th data-options="field:'CREATE_STATUS',formatter: mmlStatusFmt" width="80"><%=rb.getString("Type")%></th>
	                <th data-options="field:'TASK_STATUS',formatter:taskTableStatus" width="70"><%=rb.getString("ZhuangTai")%></th>
	                <th data-options="field:'TASK_PROGRESS'" width="70"><%=rb.getString("JinDu")%></th>
					<th data-options="field:'TASK_RESULT',formatter:taskTableResult" width="70"><%=rb.getString("JieGuo")%></th>
	                <th data-options="field:'START_TIME'" width="110"><%=rb.getString("KaiShiShiJian")%></th>
	                <th data-options="field:'END_TIME'" width="110"><%=rb.getString("JieShuShiJian")%></th>
	               
	            </tr>
            </thead>
        </table>
	</div>
	<!-- 查看任务结果信息 -->
	<div id="MMLScriptTaskProgress" style="display:none;"></div>
</div>

<%-- 工具栏 - MML脚本任务 --%>
<div id="toolbar_MMLScriptTaskList" class="toolbarContainer">
	<div class="easyui-query" name="operator_code" 
    		inputId="mmlScriptSearch" targetId="mmlScriptQueryDiv" 
    		tips="<%=rb.getString("GaoJiChaXun") %>" 
    		placeholder="<%=rb.getString("RenWu")%><%=rb.getString("MingCheng")%>" 
    		data-options="query: function(){ vagueTaskSearchFun('MMLScriptTaskList','mmlScriptSearch','mmlScript_start_time','mmlScript_end_time'); }"></div>
	<div id="mmlScriptQueryDiv" class="taskListToorBarStyle" style="">
		    <ul class="inputslist" style="">
	          <li>
	              <label ><%=rb.getString("KaiShiShiJian")%><%=rb.getString("MaoHao")%></label><br>
	              <input name="" id="mmlScript_start_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
	          </li>
	          <li>
	              <label ><%=rb.getString("JieShuShiJian")%><%=rb.getString("MaoHao")%></label><br>
	              <input name="" id="mmlScript_end_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
	          </li>
		    </ul>
		    <div class="linkbuttonGroup" style="margin-bottom:20px">
		    	  <a href="#" class="linkbutton linkbutton_trend" onclick="accurateTaskSearchFun('MMLScriptTaskList','mmlScriptSearch','mmlScript_start_time','mmlScript_end_time');mmlScriptEnterEvent();"><span><%=rb.getString("ChaXun")%></span></a>
		    	  <a href="#" class="linkbutton linkbutton_nowanna" onclick="taskResetQueryInput('mmlScriptSearch','mmlScript_start_time','mmlScript_end_time')"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
	     	</div>
	</div>
	</div>

<!-- 菜单生成 -->
<div class="wrap">
    <div id="mmlScriptMenu" class="showMMLOp"></div>
</div>

<%-- 窗口-新建任务--%>
<div id='winAddMMLScriptTask' class="slidebarPanel" style="overflow: auto;"></div>

<script type="text/javascript">

var timer_rebootTaskProgReload;
taskSearchText = '';
queryStartTime = '';
queryEndTime = '';
var curInfoDataTaskId = '';

function mmlScriptEnterEvent(){
	$("#mmlScriptQueryDiv").slideUp(400);
	$("#mmlScriptMoreQueryImg").removeClass('expanded');
    $("#mmlScriptMoreQueryImg").attr("flag","1"); 
    $("#mmlScriptHiddenSpan").hide();
    $("#mmlScriptSearch").blur();
}

function mmlStatusFmt(value, row, index) {
	var status = {
			active: '<%=rb.getString("LiJiZhiXing")%>',
			suspend: '<%=rb.getString("GuaQi")%>',
			timing: '<%=rb.getString("DingShiZhiXing")%>',
			period: '<%=rb.getString("ZhouQiRenWu")%>'
		};

	return status[value];
}

//定时刷新任务列表
function refreshMMLTasklist(taskListId, url){
	var tableTaskList = $("#"+taskListId);
	if (tableTaskList.length > 0) {
		var pageNumber = $("#"+taskListId).datagrid('options').pageNumber;
		var pageSize = $("#"+taskListId).datagrid('options').pageSize;
		var param = {};

		param ={
			timeZone:timeZone,
			page:pageNumber,
			rows:pageSize,
			likeFields :"task_name",
			searchText: taskSearchText,
			startTime : queryStartTime,
			endTime : queryEndTime
		}

		var selectBefore = "";
		try{
			selectBefore = $("#"+taskListId).datagrid('getSelected');
		}catch(e){}

		var selectRowTaskId = "";
		if(selectBefore!=null){
			selectRowTaskId = selectBefore.TASK_ID;
		}

		$.ajax({
			url: url,
			data: param,
			dataType: 'json',
			success: function(data) {
				if(window.enbMmlScriptListTimer) {
					clearTimeout(window.enbMmlScriptListTimer);
				}
				window.enbMmlScriptListTimer = setTimeout(function(){
					refreshMMLTasklist(taskListId, url);
				}, 6000);

				$.each(data.rows,function(idx,item){
					var index = $("#"+taskListId).datagrid('getRowIndex',item.TASK_ID);
					
					if(item.TASK_ID == selectRowTaskId && index > -1){
						$("#"+taskListId).datagrid('selectRow', index);
					}

					if(index > -1) {
						$("#"+taskListId).datagrid('updateRow',{
							index:index,
							row:item
						})
					}
				});
			},
			error: function(e) {
				if(window.enbMmlScriptListTimer) {
					clearTimeout(window.enbMmlScriptListTimer);
				}
				window.enbMmlScriptListTimer = setTimeout(function(){
					refreshMMLTasklist(taskListId, url);
				}, 6000);
			}
		});
		/*
		$.post(url, param,function (data) {
			$.each(data.rows,function(index,item){
				if(item.TASK_ID == selectRowTaskId){
					$("#"+taskListId).datagrid('selectRow',index);
				}
				var index = $("#"+taskListId).datagrid('getRowIndex',item.TASK_ID);
				if(index > -1) {
					$("#"+taskListId).datagrid('updateRow',{
						index:index,
						row:item
					})
				}
			})
		}, "json");
		*/
	}
}

var taskListUrl = '${ctx}/task/MMLScript/getMMLScriptTaskList.action';
$(function() {
	closeLoading();
	
	//周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_rebootTaskProgReload);
	timer_rebootTaskProgReload = undefined;
	/*
	timer_rebootTaskProgReload = setInterval(function(){
		refreshTasklist('MMLScriptTaskList',taskListUrl,'');
		var dom = $("#MMLScriptTaskList");
		if(dom.length == 0) clearInterval(timer_rebootTaskProgReload);
	},6000);
	*/
	
	$("#MMLScriptTaskList").datagrid({
        queryParams:{timeZone:timeZone},
		url:'${ctx}/task/MMLScript/getMMLScriptTaskList.action',
		onBeforeLoad: beforeLoad_MMLScriptTaskList,
		onLoadError: datagridLoadError,
		onLoadSuccess: loadSuccessMMLScriptTaskList,
		idField:'TASK_ID'
    });

	//refreshMMLTasklist('MMLScriptTaskList', taskListUrl);

	//键盘回车事件  --- 根据任务名称查询  
	$("#mmlScriptSearch").bind("keyup", function(e){
		if (e.keyCode == 13){
			taskSearchText = $('#mmlScriptSearch').val();
			queryStartTime = "";
			queryEndTime = "";
			$("#mmlScript_start_time").datetimebox('setValue', null);
			$("#mmlScript_end_time").datetimebox('setValue', null);
			mmlScriptEnterEvent();
			$('#MMLScriptTaskList').datagrid('reload');
		}
	}); 
	
    //点击页面其他位置，隐藏操作下拉选项菜单
	$(document).click(function(e){
        var e = e || window.event;
        var elem = e.target || e.srcElement;
        while(elem){
            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'showMMLOp' || elem.className == 'slideDiv'){
                return
            }
            elem = elem.parentNode;
        }
        //$(".showMMLOp").css('display','none');
    	$('#mmlScriptMenu').hide();
    })
  
});

// 显示任务详情
function showMMLScriptTaskDetail() {
	var selectedTask = $("#MMLScriptTaskList").datagrid("getSelected");
	if (!selectedTask) {
		return;
	}
	var task_id = selectedTask["TASK_ID"];
	
	if($("#MMLScriptTaskProgress").css('display') == 'block'){
	}else{		
		$("#MMLScriptTaskProgress").show(400).fadeIn(400);
	}
	
	//$(".showMMLOp").slideUp(100);
    
    $("#MMLScriptTaskProgress").panel({
        href: '${ctx}/task/MMLScript/toMMLScriptTaskProgress.action?task_id=' + task_id
    });
}

/**
 * 格式化操作
 */
function MMLTaskFormatter(value, rowData, rowIndex){	
	var task_progress = rowData.TASK_PROGRESS;
	var task_status = rowData.TASK_STATUS;
	var task_id = rowData.TASK_ID;
	
	value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbMmlscriptOp("+ task_id + ",\""+task_status+"\",this)'></div>";
	return value;
}

function enbMmlscriptOp(taskId,task_status,el){
	var XinXi = '<%=rb.getString("XinXi")%>',
	    JieGuo = '<%=rb.getString("JieGuo")%>',
	 	KaiShi = '<%=rb.getString("KaiShi")%>',
	 	ZanTing = '<%=rb.getString("ZanTing")%>',
	 	ZhongZhi = '<%=rb.getString("ZhongZhi")%>',
	 	ShanChu = '<%=rb.getString("ShanChu")%>';
	var data = [
	        {taskId:taskId, code: 'info', text: XinXi},
			{taskId:taskId, code: 'view', text: JieGuo},
			{taskId:taskId, code: 'start', text: KaiShi,cls:'CODE_ENB_MML hidden'},
			{taskId:taskId, code: 'wait', text: ZanTing,cls:'CODE_ENB_MML hidden'},
			{taskId:taskId, code: 'end', text: ZhongZhi,cls:'CODE_ENB_MML hidden'},
			{taskId:taskId, code: 'del', text: ShanChu,cls:'CODE_ENB_MML hidden'}
		];
	initTaskStatus(task_status,data);
	var menuCnt = $('#mmlScriptMenu');
	menuCnt.cmenu({data: data, click: mmlScriptOpClick}); 
	/* 菜单位置 */
	var allHeight = $(document).height(),
		thisLeft = $(el).offset().left,
		thisTop = $(el).offset().top;
	if((allHeight - thisTop) <200){
		menuCnt.css({
			"bottom":allHeight - thisTop ,
			"left":thisLeft + 20,
			"top":"unset"
		});
		
	}else{
		menuCnt.css({
			"top":thisTop + 30,
			"left":thisLeft + 20,
			"bottom":"unset"

		});
	}
	
	menuCnt.show();
}
function mmlScriptOpClick(row){
    curInfoDataTaskId = row.taskId;
	var codes = {
	        info: openWinInfoMMLScriptTask,
			view: showMMLScriptTaskDetail,
			start: activeMMLScriptTask,
			wait: suspendMMLScriptTask,
			end: terminateMMLScriptTask,
			del: delMMLScriptTask
		};
	
	if(codes[row.code]) codes[row.code](row.taskId);
	$('#mmlScriptMenu').hide();
}
//设置操作列单元格样式 
function setMMLStyle(){
	return 'position:relative';
}

//点击行内【更多】按钮，下拉显示操作选项 
function choseMMLOp(idVal,e){
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	var indexRow = $("#MMLScriptTaskList").datagrid("getRowIndex",idVal);
	var rowHeight = $(".MMLTable .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
	
	if((allHeight - thisTop) < 240){
		$(e).next(".showMMLOp").css("bottom",rowHeight+"px");
	}else{
		$(e).next().css("top",rowHeight+"px");
	}
	$(".showMMLOp").hide();
	$("#MMLScriptTaskProgress").hide(400);
	$(e).next().fadeToggle(300);
}

//激活任务
function activeMMLScriptTask(idVal) {
	//$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingLiJiZhiXingRenWu")%>", function(r) {
        //if (r) {
        	var params = {};
        	params["taskId"] = idVal;

        	$.post("${ctx}/task/MMLScript/activeTask.action", params, function(data) {
        		if (data["success"]) {
        			$("#MMLScriptTaskList").datagrid("reload");
        		} else {
        			showMsg('error_msg',data["message"]);
        		}
        	}, "json");
        //}
 	//});
}

// 挂起任务
function suspendMMLScriptTask(idVal) {
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/MMLScript/suspendTask.action", params, function(data) {
		if (data["success"]) {
			$("#MMLScriptTaskList").datagrid("reload");
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

//终止任务
function terminateMMLScriptTask(idVal) {
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/MMLScript/terminateMMLScriptTask.action", params, function(data) {
		if (data["success"]) {
			$("#MMLScriptTaskList").datagrid("reload");
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

//删除任务
function delMMLScriptTask(idVal){
	var params = {};
	params["taskId"] = idVal;

	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
        if (r) {
            $.post('${ctx}/task/MMLScript/delMMLScriptTask.action', params, function(data) {
                if (data["success"]) {
                	$("#MMLScriptTaskList").datagrid("reload");
                } else {
                    showMsg('error_msg',data["message"]);
                }
            }, "json");
        }
    }).addClass("seriousConfirm");
}

// 任务列表加载完成，默认选中第一条数据
function loadSuccessMMLScriptTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#MMLScriptTaskList").datagrid("selectRow", 0);
	}

	if(window.enbMmlScriptListTimer) {
		clearTimeout(window.enbMmlScriptListTimer);
	}
	window.enbMmlScriptListTimer = setTimeout(function(){
		refreshMMLTasklist('MMLScriptTaskList', taskListUrl);
	}, 6000);
}
//详情
function openWinInfoMMLScriptTask() {
	$("#MMLScriptTaskProgress").hide(400);
    var slider = $("#winAddMMLScriptTask");
    slider.html('');
    var paramArr = ['info', curInfoDataTaskId];
    sessionStorage.setItem('addOrInfoFlag', paramArr);

    slider.addClass('loading').slideDown(function(){
    	slider.load("${ctx}/task/MMLScript/goAddTask.action",function(html){
    		slider.removeClass('loading');
    		$.parser.parse(slider);
    	})
    })
}
// 打开添加任务窗口
function openWinAddMMLScriptTask() {
	$("#MMLScriptTaskProgress").hide(400);

    var slider = $("#winAddMMLScriptTask");
    slider.html('');
    var paramArr = ['add', {}]
    sessionStorage.setItem('addOrInfoFlag', paramArr);

    slider.addClass('loading').slideDown(function(){
    	slider.load("${ctx}/task/MMLScript/goAddTask.action",function(html){
    		slider.removeClass('loading');
    		$.parser.parse(slider);
    	})
    })
}

//任务状态格式化：激活/挂起
function taskStatusFmtMML(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("KaiShi")%>";
	} else if (value == "1") {
		return "<%=rb.getString("ZanTing")%>";
	}
}

//表格加载前事件
function beforeLoad_MMLScriptTaskList(param) {
	//添加查询条件
	param["timeZone"] = timeZone;
	param["likeFields"] = "task_name";
	param["searchText"] = taskSearchText;
	param["startTime"] = queryStartTime;
	param["endTime"] = queryEndTime;
}
</script>