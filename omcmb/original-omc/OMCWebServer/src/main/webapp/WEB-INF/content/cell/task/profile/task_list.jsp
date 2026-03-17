<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<style>
#profileListAdvanceQuery{
	width:100%;
	padding:20px 20px 10px 40px;
	position:absolute;
	top:41px;
	left:0;
	z-index:100;
	background:white;
	box-shadow:rgba(158,200,222,0.35) 0px 10px 32px;
	display:none;
}
#profileListAdvanceQuery ul.inputslist li{
	float: left;
    height: 35px;
    margin-right:60px;
    margin-bottom:30px;
    margin-top:10px;
}
#profileListAdvanceQuery ul.inputslist li label{
    margin:0px 8px 0px 0px;   
}
.rightHideDiv{
	position:absolute;
	width:900px;
	height:96%;
	background:#FFFFFF;
	right:-2000px;	
	top:10px;
	z-index:100;
}
.itemDiv{
	width:375px;
	height:92px;
	float:left;
	margin-right:45px;
} 
.promptTitle{
	height:26px;
	line-height:26px;
	min-width:50px;
	font-size:12px;
	color:#9FB318;
	display : none;
}
.promptTitle{
	height:26px;
	line-height:26px;
	min-width:50px;
	font-size:12px;
	color:red;
	display : none;
	padding-left:7px;
}
.form-item-wrap input[type='text'] , .form-item-wrap select{
	width:300px;
	height:25px;
}
.singleSelectItem input{
	margin:0 8px 0 30px;
}
#task_view_slider {
	width: 100%;
	height: 280px;
	position:absolute;
	bottom: -315px;
	z-index:100;
	transition: bottom 0.5s ease;
}
#task_view_slider.show {
	bottom: 0px;
}
.readonly .form-group:not(.last)::before {
	position: absolute;
	display: inline-block;
	content: '';
	width: 100%;
	height: 100%;
	z-index: 1000;
} 
.readonly .form-group:not(.last) input, .readonly .form-group:not(.last) .textbox,.creator-readonly {
	background-color: #E8EEF2;
}
.readonly .form-operations {
	visibility: hidden;
}
#profileTaskListInfo.loading::before {
	background-color: #fff;
}
.execute-item {
	flex: 1 1 30%;
	padding: 10px 0px;  
	display: flex;
    align-items: center;
}
.highQueryArrow span{
	vertical-align:super;
}
</style>

<!-- profile 任务管理 界面 -->
<div class="panelDefault" id="mr_task_manager_div">
	<!-- 右上角新建任务按钮 -->
	<div class="circleIcon" style="display: none;">
		<span class="circleBg add_circle" onclick="chooseTaskType()"></span>
		<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
	</div>
	<div class="circleIcon">
		<span class="el-icon el-icon-circle-close" onclick="closeProfileSlide()"></span>
		<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	</div>
	<div class="singleContentDiv">
		<table id="backupRestoreTaskListDatagrid"></table>
	</div>
</div>

<!-- profile查询toolbar -->
 <div id="toolbar_backupRestoreListDatagrid" class="toolbarContainer"> 
 	<form id="">
 		<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
    		name="searchText" 
    		inputId="profileQueryInput" targetId="profileListAdvanceQuery" 
    		placeholder="<%=rb.getString("RenWuMingCheng")%>" 
    		data-options="query: mrListQuery"></div>
	     <div id="profileListAdvanceQuery" >         
        	  <ul class="inputslist" style="overflow: hidden;margin-bottom:10px;">
	            <li>
	                <label><%=rb.getString("RenWuMingCheng")%></label><br>	                
	                <input name="taskName" id="serialNum" class="border border-box" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label><%=rb.getString("ChuangJianZhe")%></label><br>
	                <input name="createUser" id="creatorUser" class="border border-box" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label for="sel_exception_type"><%=rb.getString("Type")%></label><br>
	                <input name="taskType" id="task_type" class="easyui-combobox" style="height: 26px;width:200px;"
	                	data-options="
	                		editable: false,
	                		data:[
	                			{value:'',text: QuanXuan},
	                			{value:'1',text: BeiFen},
	                			{value:'2',text: HuiFu}
	                		],
	                		value: ''
	                	">
	            </li>
	            <li>
	                <label><%=rb.getString("ZhuangTai")%></label><br>
	                <input name="taskStatus" id="taskStatus" class="easyui-combobox" style="height: 26px;width:200px;"
	                	data-options="
	                		editable: false,
	                		data:[
	                			{value:'',text: QuanXuan},
	                			{value:'1',text: DengDaiZhiXing},
	                			{value:'2',text: KaiQi},
	                			{value:'3',text: ZanTingStatus},
	                			{value:'4',text: GuanBi}
	                		],
	                		value:''
	                	">
	            </li>
	            <li>
	                <label ><%=rb.getString("KaiShiShiJian")%></label><br>
	                <input name="startTime" id="mr_query_start_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label ><%=rb.getString("JieShuShiJian")%></label><br>
	                <input name="endTime" id="mr_query_end_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	        </ul>
	        <div class="linkbuttonGroup" style="margin-bottom:20px">
	        	<a href="#" class="linkbutton linkbutton_trend" onclick="mraAdvanceQueryTaskDatagrid()"><span><%=rb.getString("ChaXun")%></span></a>
	        	<a href="#" class="linkbutton linkbutton_nowanna" onclick="resetkpiAdvanceQuery()"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
	        </div>
        </div>
    </form>    
</div>
	<!-- 菜单生成 -->
<div class="wrap">
    <div id="profileMenu"></div>
</div>


<!-- 查看任务详情 -->
<div id="task_view_slider" class="slideDiv flex-ctn" style='box-shadow:0px -5px 10px rgba(0,0,0,0.1)'>
	<div class="slidebarTitleDiv" style="padding-left:0px;border-bottom:none;">
		<ul class="slidebarTitleContainer">
			<li class="default"><%=rb.getString("ZhiXingJieGuo")%></li>
		</ul>
		<div class="el-icon el-icon-close" style="font-size:18px;position:absolute;right:20px;top:10px;" onclick="$('#task_view_slider').removeClass('show')"></div>
	</div>
	<div class="flex-item">
		<table id="task_view_details_list"></table>
	</div>
</div>

<%-- 选择升级任务类型 --%>
<%-- <div id="choseList" style="top: 60px; right: 60px;">
	<div onclick="addProfileTaskPage('backup');" ><a href="javascript:void(0)" ><%=rb.getString("WenJianBeiFen")%></a></div>
	<div onclick="addProfileTaskPage('restore');" ><a href="javascript:void(0)" ><%=rb.getString("WenJianHuiFu")%></a></div>
</div> --%>
<script type="text/javascript">
function closeProfileSlide(){
	try{
		slideProfileDiv();
	}catch(e){}
}
var XiaZai = '<%=rb.getString("XiaZai")%>';
var ShanChu = '<%=rb.getString("ShanChu")%>';
var ShiBai = '<%=rb.getString("ShiBai")%>';
var XiuGai = '<%=rb.getString("XiuGai")%>';
var JieGuo = '<%=rb.getString("JieGuo")%>';
var KaiShi = '<%=rb.getString("KaiShi")%>';
var ZanTing = '<%=rb.getString("ZanTing")%>';
var ZhongZhi = '<%=rb.getString("ZhongZhiRenWu")%>';
var JiZhangBianMa = '<%=rb.getString("XiaoZhanBianMa")%>';
var JiZhangMingChen = '<%=rb.getString("HostName")%>';
var ZhuangTai = '<%=rb.getString("ZhuangTai")%>';
var JieGuo = '<%=rb.getString("JieGuo")%>';
var ShiBaiYuanYin = '<%=rb.getString("ShiBaiYuanYin")%>';
var MRRenWuMoRen = '<%=rb.getString("RenWu")%>';
var DengDaiZhiXing = '<%=rb.getString("DengDaiZhiXing")%>';
var KaiQi = '<%=rb.getString("JinXingZhong")%>';
var GuanBi = '<%=rb.getString("YiJieShu")%>';
var ZanTingStatus = '<%=rb.getString("ZanTingStatus")%>';
var YiZhongZhi = '<%=rb.getString("YiZhongZhi")%>';
var ChengGong = '<%=rb.getString("ChengGong")%>';
var ShiBai = '<%=rb.getString("ShiBai")%>';
var BuFenChengGong = '<%=rb.getString("BuFenChengGong")%>'; 
var QuanXuan = '<%=rb.getString("QuanXuan")%>';
var WeiZhiXing = '<%=rb.getString("WeiZhiXing")%>';
var ShiJian = '<%=rb.getString("ShiJian")%>';
var QueRen = '<%=rb.getString("QueRen")%>';
var QueRenShanChuRenWu = '<%=rb.getString("QueRenShanChuRenWu")%>';

var BeiFen = '<%=rb.getString("BeiFen")%>';
var HuiFu = '<%=rb.getString("HuiFu")%>';


var timer_restoretaskreload;
$(function () {
	closeLoading();
	
	//MR任务列表初始化加载
	$("#backupRestoreTaskListDatagrid").datagrid({
		url : '${ctx}/task/profile/backupRestore/queryMainTaskList.action',
		border:false,
		fit:true,
		toolbar:'#toolbar_backupRestoreListDatagrid',
		rownumbers:true,
		fitColumns:true,
		pagination:true,
		pagePosition:'bottom',
		striped:true,
		singleSelect:true,
		idField:'TASK_ID',
		queryParams: {timeZone: timeZone},
		columns:[[
	          	{ field:'TASK_ID',hidden:true},
	          	{ field:'op',title:'',width:30,fixed:true,formatter: operationsCont }, 
	          	{ field:'TASK_NAME',title:'<%=rb.getString("RenWuMingCheng")%>',width:150 },
			    { field:'TASK_TYPE',title:'<%=rb.getString("Type")%>',width:60}, 
			    { field:'TASK_STATUS',formatter: taskTableStatus,title:'<%=rb.getString("ZhuangTai")%>',width:60}, 
			    { field:'TASK_PROGRESS',title:'<%=rb.getString("RenWuJinDu")%>',width:80}, 
			    { field:'TASK_RESULT',formatter: taskTableResult,title:'<%=rb.getString("JieGuo")%>',width:60}, 
			    { field:'START_TIME',title:'<%=rb.getString("KaiShiShiJian")%>',width:100}, 
				{ field:'END_TIME',title:'<%=rb.getString("JieShuShiJian")%>',width:100 },
			    { field:'CREATE_USER',title:'<%=rb.getString("ChuangJianZhe")%>',width:80}, 
			    { field:'CREATE_TIME',title:'<%=rb.getString("ChuangJianShiJian")%>',width:100}   
			]],
		/* data: [
			{task_id: 'SN001231',task_name:'备份配置-2018-10-28',task_type: 'restore',task_status: 'on',creator:'adimin',create_time:'2018-10-28'}
		], */
		onLoadSuccess:datagridLoadSuccess
	})
	
	clearInterval(timer_restoretaskreload);
	timer_restoretaskreload = setInterval(function(){
		var tb = $("#backupRestoreTaskListDatagrid");
		if(tb && tb.length){
			var opts = tb.datagrid('options'),
				params = {page: opts.pageNumber, rows: opts.pageSize};
			$.extend(params,opts.queryParams);
			$.post(opts.url,params,function(data){
				var sRow = tb.datagrid('getSelected');
				tb.datagrid('loadData',[]);
				if(data){
					data.rows.map(function(row){
						tb.datagrid('appendRow',row);
						if(sRow && sRow.TASK_ID == row.TASK_ID) tb.datagrid('selectRecord',row.TASK_ID);
					});
				}
			},'json');
			try{
				$('#task_view_details_list').datagrid('reload');
			}catch(e){}
		}else {
			clearInterval(timer_restoretaskreload);
		}
	},6000);
	
	//点击页面其他位置，隐藏操作下拉选项菜单
    $(document).click(function(e){
        var e = e || window.event;
        var elem = e.target || e.srcElement;
        while(elem){
            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'circleBg add_circle'){
                return
            } 
            elem = elem.parentNode;
        }
        
        $("#profileMenu").css('display','none');
        $("#choseList").css('display','none');
    })
});

//模糊查询 
function mrListQuery(){
	var searchTxt = $("#profileQueryInput").val();
	if($("#profileListAdvanceQuery").is(':visible')) mrAdvanceQuerySlideFun();
	
	$("#backupRestoreTaskListDatagrid").datagrid('load',{
        timeZone: timeZone,
        searchText: searchTxt,
        likeFields: 'task_name'
	});
}

//显示高级查询浮层 
function mrAdvanceQuerySlideFun(){
	judmentOtherChange("profileListAdvanceQuery");
	
	if($("#profileAdvanceQueryImg").attr("flag")=="1"){
		$("#profileListAdvanceQuery").slideDown(500);
		$("#profileAdvanceQueryImg").attr("flag","0");
		$("#profileAdvanceQueryImg").addClass('expanded');
	}else{
		$("#profileListAdvanceQuery").slideUp(400);
		$("#profileAdvanceQueryImg").attr("flag","1");
		$("#profileAdvanceQueryImg").removeClass('expanded');
	}	
}

//高级查询 
function mraAdvanceQueryTaskDatagrid(){
	$("#profileQueryInput").val("");
	var search_text_query = "",
		serial_number_query = $("#serialNum").val(),
		device_name_query = $("#cellName").val(),
        taskStatus = $('#taskStatus').combobox('getValue'),
        creator = $('#creatorUser').val(),
        taskType = $('#task_type').combobox('getValue'),
		start_time_query = $("#mr_query_start_time").datetimebox("getValue"),
		end_time_query = $("#mr_query_end_time").datetimebox("getValue");
    //开始时间不能晚于结束时间
    var validTimeResult = validateStartAndStopTime(start_time_query, end_time_query);
    if ("false" == validTimeResult) {
        showMsg('prompt_msg',"<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>");
        return;
    }
    var validTimeRange =  validateTimePeriod(start_time_query, end_time_query);
    if("false" == validTimeRange){
    	showMsg('prompt_msg',"<%=rb.getString("ZuiDuoKeXiaZai31TIanShuJu")%>");
        return;
    }
    $("#backupRestoreTaskListDatagrid").datagrid({
        queryParams : {
			searchText : search_text_query,
            taskName : serial_number_query,
            taskStatus: taskStatus,
            createUser: creator,
            taskType: taskType,
            timeZone :timeZone,
            startTime : start_time_query,
            endTime : end_time_query
        }
	});
	$("#profileListAdvanceQuery").slideUp(100);	
	$("#profileAdvanceQueryImg").removeClass('expanded');
	$("#profileAdvanceQueryImg").attr("flag","1"); 
	$("#profileAdvanceQueryTip").hide();
}

//高级查询条件重置
function resetkpiAdvanceQuery(){	
	$("#serialNum").val("");
	$("#cellName").val("");
    $('#taskStatus').combobox('setValue','');
    $('#creatorUser').val('');
    $('#task_type').combobox('setValue','');
	$("#deviceGroup").combobox('setValue', '');
	$("#mr_query_start_time").datetimebox('setValue', '');
	$("#mr_query_end_time").datetimebox('setValue', ''); 
	
}

//表格操作列初始化 
function operationsCont(value, rowData, rowIndex){	
	var rowDatas = rowData,
		taskStatus = rowDatas.TASK_STATUS,
		taskId = rowDatas.TASK_ID;
	value = "<div class='el-icon el-icon-operation-more' title='<%=rb.getString("CaoZuo")%>' onclick='choseProfileOp("+ taskId + ",\""+taskStatus+"\",this)'></div>";
	return value;
}

//点击行内【更多】按钮，下拉显示操作选项 
function choseProfileOp(taskId,task_status,el){
	var JieGuo = '<%=rb.getString("JieGuo")%>',
	 	KaiShi = '<%=rb.getString("KaiShi")%>',
	 	ZanTing = '<%=rb.getString("ZanTing")%>',
	 	ZhongZhi = '<%=rb.getString("ZhongZhi")%>',
	 	ShanChu = '<%=rb.getString("ShanChu")%>';
	var data = [
			{taskId:taskId, code: 'view', text: JieGuo},
			{taskId:taskId, code: 'start', text: KaiShi,cls:'CODE_ENB_BACKUP_RESTORE hidden'},
			{taskId:taskId, code: 'wait', text: ZanTing,cls:'CODE_ENB_BACKUP_RESTORE hidden'},
			{taskId:taskId, code: 'end', text: ZhongZhi,cls:'CODE_ENB_BACKUP_RESTORE hidden'},
			{taskId:taskId, code: 'del', text: ShanChu,cls:'CODE_ENB_BACKUP_RESTORE hidden'}
		];
	initTaskStatus(task_status,data);
	var menuCnt = $('#profileMenu');
	menuCnt.cmenu({data: data, click: clickEvent}); 
	/* 菜单位置 */
	var allHeight = $(document).height(),
		isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
		tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0,
		thisTop = $(el).offset().top;
	if((allHeight - thisTop) <200){
		menuCnt.css({
			"top":thisTop - 179 - tabsHeight,
			"left":30,
		});
		if((allHeight - thisTop) <184) $('.item-child ').css({"top":"-54px",});
	}else{
		menuCnt.css({
			"top":thisTop - 20 - tabsHeight,
			"left":30,
		});
	}
	
	menuCnt.show();
	
}
function showMenu(data){
	$('#profileMenu').cmenu({data:data,click:clickEvent});           
}
/**
* 菜单点击事件
* @param row{object}   菜单点击列表的信息
*/
function clickEvent(row){
    $('#task_view_slider').removeClass('show');
    var codes = {
			view: viewTaskDetails,
			start: profileTaskProcess,
			wait: profileTaskProcess,
			end: profileTaskProcess,
			del: profileTaskProcess
		};
	
	if(codes[row.code]) codes[row.code](row.taskId,row.code);
	$('#profileMenu').hide();
}
/**
* 根据不同参数执行不同方法
* @param taskId{number}  任务id
* @param type{string}   任务类型
*/
function profileTaskProcess(taskId,type){
	var codes = {
			start: '${ctx}/task/profile/backupRestore/activeTask.action',
			wait: '${ctx}/task/profile/backupRestore/suspendTask.action',
			end: '${ctx}/task/profile/backupRestore/terminateTask.action',
			del: '${ctx}/task/profile/backupRestore/delTask.action'
		},
		params = {
			timeZone: timeZone,
			taskId: taskId
		};
	if(codes[type]) {
		if(type == 'del'){
			$.messager.confirm(QueRen,QueRenShanChuRenWu,function(r){
				if(r) {
					$.post(codes[type],params,function(data){
						if(data['success']) {
							$('#backupRestoreTaskListDatagrid').datagrid('reload');
							toast(ChengGong,$('#omc_app_ctn'),'success');
						}else{
							toast(data.message,$('#omc_app_ctn'));
						}
					},'json');
				}
			}).addClass('seriousConfirm');
		}else{
			$.post(codes[type],params,function(data){
				if(data['success']) {
					$('#backupRestoreTaskListDatagrid').datagrid('reload');
					toast(ChengGong,$('#omc_app_ctn'),'success');
				}else{
					toast(data.message,$('#omc_app_ctn'));
				}
			},'json');
		}
	}
}
//打开新建任务界面 
function addProfileTaskPage(type){
	$('#task_view_slider').removeClass('show');
	
	$("#profileTaskListInfo").addClass('loading').animate({right:'0px'},500,function(){
		var url = '${ctx}/task/profile/backupRestore/goAddBackupRestoreTaskPage.action';
		
		$('#profileConfigForm').attr('taskType',type).load(url,{taskType:type},function(){
			$('#profileTaskType').val(type);
			$.parser.parse(this);
			$("#profileTaskListInfo").removeClass('loading');
		});
	});
	
	$(".slidebarTitleContainer > li",$('#profileTaskListInfo')).text('<%=rb.getString("XinJianRenWu")%>');
}
function queryMrDevice(){
	var params = {
			searchText: $('#tempAddDeviceQuery').val(),
			like_fields: 'serial_number,host_name',
			'groupId': $('#tempAddDeviceGroup').combobox('getValue')||''
		};
	$("#mrTaskList_device_datagrid").pairgrid("reload",params);
}
/**
* 查看任务详情
* @param taskId{number}  任务id
*/
function viewTaskDetails(taskId){
	closeProfileTaskPanel();
	$('#task_view_slider').addClass('show');
	var tb = $('#task_view_details_list'),
		columns = [
			{field:'SERIAL_NUMBER',title: JiZhangBianMa,width: 100},
			{field:'HOST_NAME',title: JiZhangMingChen,width: 150},
			{field:'PROGRESS_STATUS', formatter: resultTableStatus,title: ZhuangTai,width: 100},
			{field:'PROGRESS_RESULT', formatter: resultTableResult,title: JieGuo,width: 100},
			{field:'FAILURE_REASON',title: ShiBaiYuanYin,width: 100},
			{field:'EXECUTE_TIME',title: ShiJian,width: 100}
		];
	var url = '${ctx}/task/profile/backupRestore/querySubTask.action';
	
	tb.datagrid({
		fit: true,
		striped: true,
		border: false,
		fitColumns: true,
		rownumbers: true,
		pagination: true,
		queryParams: {timeZone: timeZone, taskId: taskId},
		columns: [columns],
		url: url
	});
}
// 判断日期 方法
function noMoreThanToday(day){
	var timestamp = Date.parse(new Date(gloableTime));
	var lasttime = timestamp - 86400000;
	var last = new Date(lasttime);
	return day > last;
}

function getCurentDateStr(){
	var now = new Date(gloableTime);
	var year = now.getFullYear();
	var month = now.getMonth()+1;
	var day = now.getDate();
	var clock = year + "-";
	if(month < 10) clock += "0";
	clock += month + "-";
	if(day <10) clock += "0";
	clock += day;
	
	return clock;
}

//关闭新建任务 、 查看属性界面 
function closeProfileTaskPanel(){
	$('#profileTaskListInfo').animate({right:'-1000px'},500);
}

function chooseTaskType(){
	var slide = $('#choseList');
	if(slide.is(':visible')) slide.hide();
	else slide.show();
}
</script>