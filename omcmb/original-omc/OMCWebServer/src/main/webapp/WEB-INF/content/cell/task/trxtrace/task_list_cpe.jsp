<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>

<%-- 重启任务 --%>
<div class="panelDefault" id="cellInfo" style="overflow:hidden" >
	<!-- 右上角添加按钮 -->
	<div class="circleIcon eNbTrace hidden">
		<span class="circleBg add_circle" onclick="openWinAddTrxTraceTaskCell()"></span>
		<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
	</div>

	<div class="CPEtraceTable singleContentDiv">
		<table class="easyui-datagrid" id="trxTraceTaskListCpe" fit="true" fitColumns="true"
				data-options="singleSelect:true,rownumbers:true,pagination:true,border:false,striped:true,pagePosition:'bottom',toolbar:'#toolbar_trxTraceTaskListCpe',
				url:'${ctx}/task/trxtrace/getTraceTaskListForCpe.action',onBeforeLoad: beforeLoadtrxTraceTaskListCpe,onLoadError:datagridLoadError,onLoadSuccess:loadSuccessTraceTaskListCpe,idField:'TASK_ID'">
            <thead><tr>
                <th data-options="field:'TASK_ID',hidden:true"></th>
                <th data-options="field:'operation',formatter : traceTaskFormatterCpe,fixed:true" width="70"><%=rb.getString("CaoZuo")%></th>
                <th data-options="field:'TASK_NAME'" width="120"><%=rb.getString("RenWu")%> <%=rb.getString("MingCheng")%></th>
                <th data-options="field:'TASK_STATUS',formatter:taskStatusFmtTraceCpe" width="70"><%=rb.getString("ZhuangTai")%></th>
                <th data-options="field:'TASK_PROGRESS',formatter:taskProgressFmtTraceCpe" width="70"><%=rb.getString("JinDu")%></th>
				<th data-options="field:'TASK_RESULT',formatter:taskResultFmtTraceCpe" width="70"><%=rb.getString("JieGuo")%></th>
                <th data-options="field:'START_TIME'" width="110"><%=rb.getString("KaiShiShiJian")%></th>
                <th data-options="field:'STOP_TIME'" width="110"><%=rb.getString("JieShuShiJian")%></th>
                
            </tr></thead>
        </table>
	</div>	
</div>
		
<%-- 工具栏 - trxtrace --%>
<div id="toolbar_trxTraceTaskListCpe" class="toolbarContainer">
	<div class="queryGroup">
		<input id="trxTraceSearch" name="task_name" placeholder="<%=rb.getString("RenWu")%><%=rb.getString("MingCheng")%>" />
		<b onclick="$('#trxTraceTaskListCpe').datagrid('reload')"></b>
	</div>
</div>


<div id="trxTraceTaskProgressCpe" style="display:none;"></div>

<div id="traceTaskToolsBarCpe">
	<a href="javascript:void(0)" onclick="openWinAddTrxTraceTaskCell()" class="icon-add"></a>
</div>
<%-- 右键菜单 --%>
<div id="rowMenu_traceTaskListCpe" class="easyui-menu">
	<div act="active" onclick="activeTraceTaskCpe()"><%=rb.getString("JiHuo")%></div>
	<div act="suspend" onclick="suspendTraceTaskCpe()"><%=rb.getString("GuaQi")%></div>
</div>

<%-- 窗口-新建任务--%>
<%-- <div id="winAddTrxTraceTaskCpe" class="easyui-window" title="<%=rb.getString("XinJianZhuiZongRenWu")%>"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:860,height:600,resizable:false">
</div> --%>

<script type="text/javascript">
$(function() {
	closeLoading();
	$("#trxTraceTaskListCpe").datagrid({
        queryParams:{timeZone:timeZone}
    });
	
	//鼠标滚动事件，当有鼠标滚动时，操作下拉选项隐藏
	var scrollFunc = function(e){
		e = e || window.event;
		if(e.wheelDelta){  //IE/OPera/Chrome
			$(".showTraceOpCPE").hide();
		}else if(e.detail){  //Firefox
			$(".showTraceOpCPE").hide();
		}
	}
	/* 注册事件 */
	if(document.addEventListener){
		document.addEventListener('DOMMouseScroll',scrollFunc,false);
	}
	window.onmousewheel = document.onmousewheel = scrollFunc;
	//键盘回车事件  --- 根据任务名称查询  
	$("#trxTraceSearch").bind("keyup", function(e){
		if (e.keyCode == 13){
			$('#trxTraceTaskListCpe').datagrid('reload');
		}
	});
	//点击页面其他位置，隐藏操作下拉选项菜单
    $(document).bind('click',function(e){
        var e = e || window.event;
        var elem = e.target || e.srcElement;
        while(elem){
            if( elem.className == 'iconAdd iconSize' || elem.className == 'showTraceOpCPE' || elem.className == 'slideDiv'){ 
                return
            }
            elem = elem.parentNode;
        }
        $(".showTraceOpCPE").css('display','none');
        $("#trxTraceTaskProgressCpe").hide(400);
    })
    //周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_reLoad);
	timer_reLoad = undefined;
	timer_reLoad = setInterval(function(){
		trxTraceTaskListCpeReload();
		var dom = $("#trxTraceTaskListCpe");
		if(dom.length == 0) clearInterval(timer_reLoad);
	}, 6000);
});

function ChoseAddUpgradeTask(){
	$(".showTraceOpCPE").css('display','none');
}

function choseOp(idVal,e){
	$("#trxTraceTaskProgressCpe").hide(400);
	$(".showTraceOpCPE").fadeIn(300);
	$(e).next().fadeToggle(300);
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	var indexRow = $("#trxTraceTaskListCpe").datagrid("getRowIndex",idVal);
	var rowHeight = $(".CPEtraceTable .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
	//var thisRow = $(".CPEtraceTable .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]");
	
	//var thisTop = thisRow.offset().top;
	if((allHeight - thisTop) < 240){
		$(e).next(".showTraceOpCPE").css("bottom",rowHeight+"px");
	}else{
		$(e).next(".showTraceOpCPE").css("top",(rowHeight*(indexRow+2))+"px");
	}
	//$(e).next().css("top",(thisTop+rowHeight)+"px");
	//$(e).next().css("right","70px");
}
/* function choseMMLOp(idVal,e){
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	var indexRow = $("#MMLScriptTaskList").datagrid("getRowIndex",idVal);
	var rowHeight = $(".MMLTable .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
	
	if((allHeight - thisTop) < 240){
		$(e).next(".showMMLOp").css("bottom",rowHeight+"px");
	}else{
		$(e).next().css("top",(rowHeight*(indexRow+1))+"px");
	}
	$(".showMMLOp").hide();
	$("#MMLScriptTaskProgress").hide(400);
	$(e).next().fadeToggle(300);
} */

// 显示任务详情
function showTraceTaskDetailCpe() {
	var selectedTask = $("#trxTraceTaskListCpe").datagrid("getSelected");
	if (!selectedTask) {
		return;
	}
	var task_id = selectedTask["TASK_ID"];
	
	if($("#trxTraceTaskProgressCpe").css('display') == 'block'){
	}else{
		
		$("#trxTraceTaskProgressCpe").show(300).fadeIn(300);
	}
	
	$(".showTraceOpCPE").slideUp(100);
	
    $("#trxTraceTaskProgressCpe").panel({
        href: '${ctx}/task/trxtrace/toTraceTaskProgressForCpe.action?task_id=' + task_id
    });
}



//激活任务
function activeTraceTaskCpe(idVal) {
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingLiJiZhiXingRenWu")%>", function(r) {
        if (r) {
        	var grid = $("#trxTraceTaskListCpe");
        	var params = {};
        	params["taskId"] = idVal;
        	
        	$.post("${ctx}/task/trxtrace/activeTaskForCpe.action", params, function(data) {
        		if (data["success"]) {
        			grid.datagrid("reload");
        		} else {
        			showMsg('error_msg',data["message"]);
        		}
        	}, "json");
        }
 	});
}

// 挂起任务
function suspendTraceTaskCpe(idVal) {
	var grid = $("#trxTraceTaskListCpe");
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/trxtrace/suspendTaskForCpe.action", params, function(data) {
		if (data["success"]) {
			grid.datagrid("reload");
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}
//任务状态格式化：激活/挂起
function taskStatusFmtTraceCpe(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("JiHuo")%>";
	} else if (value == "1") {
		return "<%=rb.getString("GuaQi")%>";
	}
}
//任务执行结果格式化
function taskResultFmtTraceCpe(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("ChengGong")%>";
	} else if (value == "1") {
		return "<%=rb.getString("BuFenChengGong")%>";
	} else if (value == "2") {
		return "<%=rb.getString("ShiBai")%>";
	} else if (value == "3") {
		return "<%=rb.getString("ZhongZhi")%>";
	} else {
		return "";
	}
}
//任务进度格式化
function taskProgressFmtTraceCpe(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("WeiKaiShi")%>";
	} else if (value == "1") {
		return "<%=rb.getString("JinXingZhong")%>";
	} else if (value == "2") {
		return "<%=rb.getString("YiJieShu")%>";
	}
}
//终止任务
function terminateTraceTaskCpe(idVal) {
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/trxtrace/terminateTraceTaskForCpe.action", params, function(data) {
		if (data["success"]) {
			$("#trxTraceTaskListCpe").datagrid("reload");
		}else{
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

//删除任务
function delTraceTaskCpe(idVal){
	var params = {};
	params["taskId"] = idVal;
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
        if (r) {
            $.post('${ctx}/task/trxtrace/delTraceTaskForCpe.action', params, function(data) {
                if (data["success"]) {
                    $("#trxTraceTaskListCpe").datagrid("reload");
                } else {
                	showMsg('error_msg',data["message"]);
                }
            }, "json");
        }
    }).addClass("seriousConfirm");
}

// 任务列表加载完成，默认选中第一条数据
function loadSuccessTraceTaskListCpe(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#trxTraceTaskListCpe").datagrid("selectRow", 0);
	}
}

// 打开添加任务窗口
function openWinAddTrxTraceTaskCell() {
	$("#trxTraceTaskProgressCpe").hide(400);
	/* $("#winAddTrxTraceTaskCpe").window({
		width: 860,
    	height: document.body.clientHeight * 0.9
    }).window("center").window("open");
    
    $("#winAddTrxTraceTaskCpe").window("refresh", "${ctx}/task/trxtrace/goAddTaskForCpe.action"); */
    var url = "${ctx}/task/trxtrace/goAddTaskForCpe.action";
    openDefaultWindow(url,{
    	title: '<%=rb.getString("XinJianZhuiZongRenWu")%>',
    	width: 860,
    	height: document.body.clientHeight * 0.9
    });
}
function traceTaskFormatterCpe(value, rowData, rowIndex){
	var task_progress = rowData.TASK_PROGRESS;
	var task_status = rowData.TASK_STATUS;
	var task_id = rowData.TASK_ID;
	
	var JieGuo = '<%=rb.getString("JieGuo")%>';
	var JiHuo = '<%=rb.getString("JiHuo")%>';
	var GuaQi = '<%=rb.getString("GuaQi")%>';
	var ZhongZhi = '<%=rb.getString("ZhongZhi")%>';
	var ShanChu = '<%=rb.getString("ShanChu")%>';
	value = "<div class='operation_more' title='"+ CaoZuo+"' onclick='choseOp("+ task_id +",this)'></div>";
	
	var opt="";
	opt = opt + "<div class='titleDiv operation_result' title='"+JieGuo+"' onclick='showTraceTaskDetailCpe()' style='margin-left:15px;'>"+JieGuo+"</div>";
	if(task_status != 0 && task_progress != 2){//当前不是处于激活状态，且未结束，激活图标可用
		opt = opt + "<div class='titleDiv titleIcon_active_click eNbTrace hidden' title='"+JiHuo+"' onclick='activeTraceTaskCpe(\"" + task_id + "\")'>"+JiHuo+"</div>";
	}else if(task_status != 1 && task_progress != 2){//当前不是处于挂起状态，且未结束，挂起图标可用
		opt = opt + "<div class='titleDiv titleIcon_awaiting_click eNbTrace hidden' title='"+GuaQi+"' style='margin-left:15px;' onclick='suspendTraceTaskCpe(\"" + task_id + "\")'>"+GuaQi+"</div>";
	}else{
		opt = opt + "<div class='titleDiv titleIcon_active_disabled eNbTrace hidden' title='"+JiHuo+"'>"+JiHuo+"</div>";
	}
	
	if(task_progress != 2){//当前任务没有结束，终止图标可用
		opt = opt + "<div class='titleDiv titleIcon_terminate_click eNbTrace hidden' title='"+ZhongZhi+"' style='margin-left:15px;' onclick='terminateTraceTaskCpe(\"" + task_id + "\")'>"+ZhongZhi+"</div>";
	}else{
		opt = opt + "<div class='titleDiv titleIcon_terminate_disabled eNbTrace hidden' title='"+ZhongZhi+"' style='margin-left:15px;'>"+ZhongZhi+"</div>";
	}
	
	if(task_progress != 1){//当前任务不在进行中，删除图标可用
		opt = opt + "<div class='titleDiv titleIcon_delete_click eNbTrace hidden' title='"+ShanChu+"' style='margin-left:15px;' onclick='delTraceTaskCpe(\"" + task_id + "\")'>"+ShanChu+"</div>";
	}else{  
		opt = opt + "<div class='titleDiv titleIcon_delete_disabled eNbTrace hidden' title='"+ShanChu+"' style='margin-left:15px;'>"+ShanChu+"</div>";
	}
	value = value + "<div class='showTraceOpCPE'>"+ opt +"</div>";
	return value;
}
function beforeLoadtrxTraceTaskListCpe(param) {
	//添加查询条件
	param["search_text"] = $("#toolbar_trxTraceTaskListCpe input[name='task_name']").val();
	param["like_fields"] = "task_name";
	param["timeZone"] = timeZone;
}
function trxTraceTaskListCpeReload(){
	var pageNumber = $("#trxTraceTaskListCpe").datagrid('options').pageNumber;
	var pageSize = $("#trxTraceTaskListCpe").datagrid('options').pageSize;
	var param ={};
   	param["timeZone"] = timeZone;
   	param["page"] = pageNumber;
   	param["rows"] = pageSize;
   	param["search_text"] = $("#toolbar_trxTraceTaskListCpe input[name='task_name']").val();
   	param["like_fields"] = "task_name";
   	var url = '${ctx}/task/trxtrace/getTraceTaskListForCpe.action';
   	var selectBefore = "";
	try{
		selectBefore = $("#trxTraceTaskListCpe").datagrid('getSelected');
	}catch(e){
		
	}
	var selectRowTaskId = "";
	if(selectBefore!=null){
		selectRowTaskId = selectBefore.TASK_ID;
	}
	$.post(url, param,function (data) {
		$.each(data.rows,function(index,item){
			$("#trxTraceTaskListCpe").datagrid('updateRow',{
				index:index,
				row:item
			})
			if(item.TASK_ID == selectRowTaskId){
				$("#trxTraceTaskListCpe").datagrid('selectRow',index);
			}
		})
    }, "json");
}
</script>