
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>
<style>
.menu-list {
	color: #000;
	width:160px;
	border:1px solid rgba(188,188,188,0.1);
	background:#fff;
	position:absolute;
	top:60px;
	right:70px;
	display:none;
	box-shadow:0 5px 15px #d8d8d8;
}
.menu-list>div, .sub-choselist div{
	position: relative;
	height:40px;
	line-height:40px;
	padding-left:30px;
	border-bottom:1px solid #e5f0f6; 
}
.menu-list div a{
	color:#000;
}
.menu-list div:last-of-type, .sub-choselist div:last-of-type{
	border:none;
}
.menu-list>div:hover, .sub-choselist div:hover{
	background:#e1f2fa;
}
.menu-list div:active{
	background:#c4e6f5;
}
/* 新建 enb 恢复配置样式 */
.addRestContainer{
	position:absolute;
	left:0px;
	right:0px;
	top:0px;
	bottom:0px;
	z-index:97;
	background:#FFFFFF;
}
.restmainContainer{
	display:flex;
	flex-direction:column;
}
.resetExecutionModeCon{
	
}
.resetExecutionModeWay{
	height:50px;
	border:1px solid #85A8BF;
	flex:1 1 auto; 
}
.sub-wrap {
	display: none;
	position: absolute !important;
	top: 0px;
	padding: 0px !important;
	right: 162px;
}
.sub-choselist {
	height: auto !important;
	width:160px;
	height: auto;
	border:1px solid rgba(188,188,188,0.1);
	background:#fff;
	box-shadow:0 5px 15px #d8d8d8;
}
.tabsTitle{
	border:none;
}
.highQueryArrow span{
	vertical-align:super;
}
</style>

<%-- 配置恢复任务 --%>
<div class="panelDefault">
	<!-- 右上角添加按钮 -->
	<div class="circleIcon CODE_ENB_RESET_CONFIG hidden" style="top:0;">
		<span id="reset_add_bt" class="el-icon el-icon-circle-add" onclick="rebootMenuClick()"></span>
		<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
	</div>
	<div class=" rebootTable resetConfig" style="height:100%;">	
		<table class="easyui-datagrid" id="resetTaskList" fit="true" fitColumns="true"
				data-options="singleSelect:true,rownumbers:true,pagination:true,border:false,striped:true,pagePosition:'bottom',onBeforeLoad: beforeLoadResetTaskList,toolbar:'#toolbar_resetTaskList',
				url:'${ctx}/task/factoryReset/getFactoryResetTaskList.action',onLoadError:datagridLoadError,onLoadSuccess:loadSuccessResetTaskList,idField:'TASK_ID'">
            <thead>
	            <tr>
	            	<th data-options="field:'operation',formatter : resetTaskFormatter,fixed:true,styler:setResetStyler,fixed:true" width="30"></th>	  
	                <th data-options="field:'TASK_ID',hidden:true"></th>
	                <th data-options="field:'TASK_NAME'" width="120"><%=rb.getString("RenWuMingCheng")%></th>
	                <th data-options="field:'CREATE_USER'" width="100"><%=rb.getString("ChuangJianZhe")%></th>
	                <th data-options="field:'CREATE_TIME'" width="110"><%=rb.getString("ChuangJianShiJian")%></th>
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
	<div id="resetTaskProgress" style="display:none;"></div>
</div>

<%-- 工具栏 - 配置恢复任务  --%>
<div id="toolbar_resetTaskList" class="toolbarContainer">
	<div class="easyui-query" name="task_name" 
    		inputId="resetTaskListSearch" targetId="resetQueryDiv" 
    		tips="<%=rb.getString("GaoJiChaXun") %>" 
    		placeholder="<%=rb.getString("RenWu")%><%=rb.getString("MingCheng")%>" 
    		data-options="query: function(){ vagueTaskSearchFun('resetTaskList','resetTaskListSearch','reset_start_time','reset_end_time'); }"></div>
	<div id="resetQueryDiv" class="advanceQuery_content" style="">
	    <ul class="inputslist" style="">
	        <li>
                <label ><%=rb.getString("KaiShiShiJian")%><%=rb.getString("MaoHao")%></label><br>
                <input name="" id="reset_start_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
            </li>
            <li>
                <label ><%=rb.getString("JieShuShiJian")%><%=rb.getString("MaoHao")%></label><br>
                <input name="" id="reset_end_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
            </li>
	    </ul>
	    <div class="linkbuttonGroup" >
	    	<a href="#" class="linkbutton linkbutton_trend" onclick="accurateTaskSearchFun('resetTaskList','resetTaskListSearch','reset_start_time','reset_end_time');resetEnterEvent();"><span><%=rb.getString("ChaXun")%></span></a>
	    	<a href="#" class="linkbutton linkbutton_nowanna" onclick="taskResetQueryInput('resetTaskListSearch','reset_start_time','reset_end_time')"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
     	</div>
     </div>
</div>

<!-- 菜单生成 -->
<div class="wrap">
    <div id="enbResetMenu"></div>
</div>
<%-- 双载波选择重启任务类型 --%>
<div id="dualResetChoseList" class="menu-list" >
	<input type="hidden" id="rebootCarrierModel"/>
	<input type="hidden" id="rebootProductValue"/>
</div>

<%-- 窗口-新建任务--%>
<div id='winAddRebootTask' class="slidebarPanel"></div>

<script type="text/javascript">

var timer_rebootTaskProgReload;
taskSearchText = '';
queryStartTime = '';
queryEndTime = '';

/* 显示高级查询选项  */
function resetMoreQueryImgFun(){
	if($("#resetMoreQueryImg").attr("flag")=="1"){
		$("#resetQueryDiv").slideDown(500);
		$("#resetMoreQueryImg").attr("flag","0");
		$("#resetMoreQueryImg").addClass('expanded');
	}else{
		$("#resetQueryDiv").slideUp(500);
		$("#resetMoreQueryImg").attr("flag","1");
		$("#resetMoreQueryImg").removeClass('expanded');
	}
	event.stopPropagation();
}
function resetEnterEvent(){
	$("#resetQueryDiv").slideUp(400);
	$("#resetMoreQueryImg").removeClass('expanded');
    $("#resetMoreQueryImg").attr("flag","1"); 
    $("#resetHiddenSpan").hide();
    $("#resetTaskListSearch").blur();
}
try{
	$.getJSON('${ctx}/cell/version/getProductType.action',function(data){
		if(data){			
			var ctn = $('#dualResetChoseList');
			var gMap = {};
			data.map(function(item){
				if(gMap[item.device_type]) gMap[item.device_type].push(item);
				else gMap[item.device_type] = [item];
			});
			for(var key in gMap){
				var sub = $('<div>'+key+'<div class="sub-wrap"><div class="sub-choselist"></div></div></div>'),
					subCtn = sub.find('.sub-choselist');
				ctn.append(sub);
				var items = gMap[key];
				items.map(function(item){
					subCtn.append('<div class="single" valueFlag="'+item.value+'" onclick="rebootMenuClick(this)"><a href="javascript:void(0)" >'+item.name+'</a></div>');
				});
			}
			ctn.find('>div').on('mouseenter',function(){
				var wrap = $(this);
				wrap.find('.sub-wrap').show();
				wrap.siblings().find('.sub-wrap').hide();
				$('#choseList').hide();
			});
		}
	});
}catch(e){}
$(function() {
	closeLoading();
	
	//周期性刷新，同时刷新任务列表和结果列表，先清空定时器，只有任务列表时只刷新任务列，定时器传一个函数
	clearInterval(timer_rebootTaskProgReload);
	timer_rebootTaskProgReload = undefined;
	var taskListUrl = '${ctx}/task/factoryReset/getFactoryResetTaskList.action';
	timer_rebootTaskProgReload = setInterval(function(){
		refreshTasklist('resetTaskList',taskListUrl,'');
		var dom = $("#resetTaskList");
		if(dom.length == 0) clearInterval(timer_rebootTaskProgReload);
	},6000);
	
	$("#resetTaskList").datagrid({
        queryParams:{timeZone:timeZone}
    });
	//键盘回车事件  --- 根据任务名称查询  
	$("#resetTaskListSearch").bind("keyup", function(e){
		if (e.keyCode == 13){
			taskSearchText = $('#resetTaskListSearch').val();
			queryStartTime = "";
			queryEndTime = "";
			$("#reset_start_timeme").datetimebox('setValue', null);
			$("#reset_end_time").datetimebox('setValue', null);
			resetEnterEvent();
			$('#resetTaskList').datagrid('reload');
		}
	});
  /**
   * 点击页面其他位置，隐藏操作下拉选项菜单
  */
	$(document).click(function(e){
        var e = e || window.event;
        var elem = e.target || e.srcElement;
        while(elem){
            if($(elem).hasClass('el-icon-operation-more')|| elem.className == 'showRebootOp' || elem.className == 'slideDiv'){
                return
            }
            elem = elem.parentNode;
        }
        //$(".showRebootOp").css('display','none');
        $("#dualResetChoseList").fadeOut(400);
       // $("#resetTaskProgress").hide(400);  
        $('#enbResetMenu').hide();
    })
   });

//设置操作列单元格样式 
function setResetStyler(){
	return "position:relative";
}

/**
 * 点击行内【更多】按钮，下拉显示操作选项 
 * @param idval:点击的参数
*/
function choseResetOp(idVal,e){
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	var indexRow = $("#resetTaskList").datagrid("getRowIndex",idVal);
	var rowHeight = $(".rebootTable .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();

	if((allHeight - thisTop) < 240){
		$(e).next(".showRebootOp").css("bottom",rowHeight+"px");
	}else{
		$(e).next().css("top",rowHeight+"px");
	}
	
	$(".showRebootOp").hide();
	$("#resetTaskProgress").hide(400);
	$(e).next().fadeToggle(300);
}

// 显示任务详情
function showResetTaskDetail() {
	var selectedTask = $("#resetTaskList").datagrid("getSelected");
	if (!selectedTask) {
		return;
	}
	var task_id = selectedTask["TASK_ID"];
	
	if($("#resetTaskProgress").css('display') == 'block'){	
	}else{		
		$("#resetTaskProgress").show(400).fadeIn(400);
	}
	//$(".showRebootOp").slideUp(100);
	
    $("#resetTaskProgress").panel({
        href: '${ctx}/task/factoryReset/toFactoryResetTaskProgress.action?task_id=' + task_id
    });
}



/**
 * 激活任务
 * @param idval:传递参数
*/
function activeResetTask(idVal) {
	//$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingLiJiZhiXingRenWu")%>", function(r) {
        //if (r) {
        	var grid = $("#resetTaskList");
        	var params = {};
        	params["taskId"] = idVal;
        	
        	$.post("${ctx}/task/factoryReset/activeTask.action", params, function(data) {
        		if (data["success"]) {
        			grid.datagrid("reload");
        		} else {
        			showMsg('error_msg',data["message"]);
        		}
        	}, "json");
        //}
 	//}).addClass("normalConfirm");
}

/**
 * 挂起任务
 * @param idval:传递参数
*/
function suspendResetTask(idVal) {
	var grid = $("#resetTaskList");
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/factoryReset/suspendTask.action", params, function(data) {
		if (data["success"]) {
			grid.datagrid("reload");
		} else {
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

/**
 * 终止任务
 * @param idval:传递参数
*/
function terminateResetTask(idVal) {
	var params = {};
	params["taskId"] = idVal;
	$.post("${ctx}/task/factoryReset/terminateFactoryResetTask.action", params, function(data) {
		if (data["success"]) {
			$("#resetTaskList").datagrid("reload");
		}else{
			showMsg('error_msg',data["message"]);
		}
	}, "json");
}

/**
 * 删除任务
 * @param idval:传递参数
*/
function delResetTask(idVal){
	var params = {};
	params["taskId"] = idVal;
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
        if (r) {
            $.post('${ctx}/task/factoryReset/delFactoryResetTask.action', params, function(data) {
                if (data["success"]) {
                    $("#resetTaskList").datagrid("reload");
                } else {
                	showMsg('error_msg',data["message"]);
                }
            }, "json");
        }
    }).addClass("seriousConfirm");
}

/**
 * 任务列表加载完成，默认选中第一条数据
 * @param data：表格数据
*/
function loadSuccessResetTaskList(data) {
	$(this).datagrid("enableContextmenuAutoSize");
	if (data["rows"].length > 0) {
		$("#resetTaskList").datagrid("selectRow", 0);
	}
}


// 打开新建任务页面 
function rebootMenuClick(obj){
	var url = "${ctx}/task/factoryReset/goAddTask.action";
	
	$("#resetTaskProgress").hide(400);
	$(".showRebootOp").css('display','none');
	<%-- var options = {
    		title: '<%=rb.getString("XinJianChongZhiRenWu")%>',
			width: 860,
	    	height: document.body.clientHeight * 0.9
   		};
    openDefaultWindow(url,options); --%>
    
    var slider = $("#winAddRebootTask");
    slider.html('');
    slider.addClass('loading').slideDown(function(){
    	slider.load(url,function(html){
    		slider.removeClass('loading');
    		$.parser.parse(slider);
    	})
    })
}

/**
 * 格式化操作
 */
function resetTaskFormatter(value, rowData, rowIndex){
	var task_progress = rowData.TASK_PROGRESS;
	var task_status = rowData.TASK_STATUS;
	var task_id = rowData.TASK_ID;
	
	value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbResetConfigOp("+ task_id + ",\""+task_status+"\",this)'></div>";
	return value;
}
function enbResetConfigOp(taskId,task_status,el){
	var JieGuo = '<%=rb.getString("JieGuo")%>',
	 	KaiShi = '<%=rb.getString("KaiShi")%>',
	 	ZanTing = '<%=rb.getString("ZanTing")%>',
	 	ZhongZhi = '<%=rb.getString("ZhongZhi")%>',
	 	ShanChu = '<%=rb.getString("ShanChu")%>';
	var data = [
			{taskId:taskId, code: 'view', text: JieGuo,},
			{taskId:taskId, code: 'start', text: KaiShi,cls:'CODE_ENB_RESET_CONFIG hidden'},
			{taskId:taskId, code: 'wait', text: ZanTing,cls:'CODE_ENB_RESET_CONFIG hidden'},
			{taskId:taskId, code: 'end', text: ZhongZhi,cls:'CODE_ENB_RESET_CONFIG hidden'},
			{taskId:taskId, code: 'del', text: ShanChu,cls:'CODE_ENB_RESET_CONFIG hidden'}
		];
	initTaskStatus(task_status,data);
	$('#enbResetMenu').cmenu({data: data, click: enbResetOpClick}); 
	/* 菜单位置 */
	var allHeight = $(document).height(),
		isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
		tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0,
		thisTop = $(el).offset().top;
	if((allHeight - thisTop) <200){
		$('#enbResetMenu').css({
			"top":thisTop - 189 - tabsHeight,
			"left":30,
		});
		if((allHeight - thisTop) <184) $('.item-child ').css({"top":"-54px",});
	}else{
		$('#enbResetMenu').css({
			"top":thisTop - 20 - tabsHeight,
			"left":30,
		});
	}
	
	$('#enbResetMenu').show();
}
/**
 * 点击操作的具体项
 * @param row:传递的具体参数
*/
function enbResetOpClick(row){
	var codes = {
			view: showResetTaskDetail,
			start: activeResetTask,
			wait: suspendResetTask,
			end: terminateResetTask,
			del: delResetTask
		};
	
	if(codes[row.code]) codes[row.code](row.taskId);
	$('#enbResetMenu').hide();
}
//加载前事件 - 选择升级文件
function beforeLoadResetTaskList(param){ 
	// 双载波
	/* var carrierType = '${carrierType}';
	if(carrierType == '2') param["isDual"] = true; */
	param["timeZone"] = timeZone;
	param["likeFields"] = "task_name";
	param["searchText"] = taskSearchText;
	param["startTime"] = queryStartTime;
	param["endTime"] = queryEndTime;
}
</script>