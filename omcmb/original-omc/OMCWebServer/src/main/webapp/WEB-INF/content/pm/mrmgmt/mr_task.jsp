<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<style>
	.mrRightHideDiv {
		position:absolute;
		width:900px;
		height:100%;
		background:#FFFFFF;
		right:-2000px;	
		top:10px;
		z-index:100;
	}
	
	.mrPromptTitle {
		height:26px;
		line-height:26px;
		min-width:50px;
		font-size:12px;
		color: #FF0000;
		display : none;
		padding-left:7px;
	}
	.mr-form-item-wrap input[type='text'],
	.mr-form-item-wrap select {
		width:300px;
		height:25px;
	}
	.mrSingleSelectItem input {
		margin:0 8px 0 30px;
	}
	#task_view_slider {
		height: 310px;
		position:absolute;
		bottom: -350px;
		z-index:100;
		transition: bottom 0.5s ease;
	}
	#task_view_slider.show {
		bottom: 0px;
	}
	.mrReadonly .form-group:not(.last)::before {
		position: absolute;
		display: inline-block;
		content: '';
		width: 100%;
		height: 100%;
		z-index: 1000;
	} 
	.mrReadonly .form-group:not(.last) input, .mrReadonly .form-group:not(.last) .textbox {
		background-color: #E8EEF2;
	}
	.mrReadonly .form-operations {
		display:none;
	}
	#mrTaskListInfo.loading::before {
		background-color: #FFF;
	}
	.mr-form-item-wrap > input[readonly] {
		background-color: #F6F6F6;
	}
	#gsmConfigForm .group-title {
		padding: 10px 0 0 20px;
	}
	.mrInputslist .searchbox {
		height: 26px;
		width: 200px;
	}
	.mrInputslist {
		overflow: hidden;
		margin-bottom: 10px;
	}
	.mrInputslist li{
		float: none;
		display: inline-block;
		margin: 0 50px 15px 0;
	}
	.mrInputslist label{
		line-height: 24px;
	}
	#mr_task_manager_div .contentDivBox{
		height: calc(100% - 30px);
		width:100%;
	}
</style>
<div class="overflow-cls">
<!-- MR 任务管理 界面 -->
	<div class="panelDefault" id="mr_task_manager_div" style="min-width: 900px;">
		
		<div style="display:flex;">
			<div class="singleTitle" style="cursor: pointer" id="mrTaskBtn" onclick="mrHeadBtnClick('task')"><span><%=rb.getString("MRRenWuGuanLi")%></span></div>
			<div class="singleTitle" style="cursor: pointer" id="mrFileBtn" onclick="mrHeadBtnClick('file')"><span><%=rb.getString("MRWenJianGuanLi")%></span></div>
		</div>
		<div class="contentDivBox" id="mrTaskTableBox">	
			<!-- 右上角新建任务按钮 -->			
			<div class="newIconBoxCls-bt CODE_PERFORMANCE_MR hidden" style="right:10px;top:10px;" onclick="addMRTaskPage('add')" tip="<%=rb.getString("TianJia")%>">
				<i class="el-icon el-icon-plus" ></i>
			</div>
			<table id="mrTaskListDatagrid" style="height:100%;"></table>
		</div>
		<div class="contentDivBox" id="mrFileTableBox" style="display:none">	
			<table id="mr_fileMgmt_grid" style="height:100%;"></table>
		</div>
	</div>
</div>
<!-- 任务查询toolbar -->
<div id="toolbar_mrTaskListDatagrid" class="toolbarContainer"> 
	<form id="">
		<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
			name="searchText" 
			inputId="mrTaskListQueryInput" targetId="mrListAdvanceQuery" 
			placeholder="<%=rb.getString("RenWuMingCheng")%>" 
			data-options="query: mrListQuery">
		</div>
		<div id="mrListAdvanceQuery" class="advanceQuery_content" style="display:none;">         
			<ul class="mrInputslist">
				<li>
					<label><%=rb.getString("RenWuMingCheng")%></label><br>	                
					<input name="taskName" id="serialNum" class="border border-box searchbox">
				</li>
				<li>
					<label><%=rb.getString("ZhuangTai")%></label><br>
					<input name="taskStatus" id="taskStatus" class="easyui-combobox searchbox"
						data-options="
							editable: false,
							data:[
								{value:'',text: QuanXuan},
								{value:'waitting',text: DengDaiZhiXing},
								{value:'on',text: KaiQi},
								{value:'off',text: GuanBi},
								{value:'suspend',text: ZanTingStatus},
								{value:'termination',text: YiZhongZhi}
							],
							value:''
						">
				</li>
				<li>
					<label><%=rb.getString("ChuangJianZhe")%></label><br>
					<input name="creator" id="creatorUser" class="border border-box searchbox">
				</li>
				<li>
					<label for="sel_exception_type"><%=rb.getString("JieGuo")%></label><br>
					<input name="taskResult" id="taskResult" class="easyui-combobox searchbox"
						data-options="
							editable: false,
							data:[
								{value:'',text: QuanXuan},
								{value:'success',text: ChengGong},
								{value:'partialSuccess',text: BuFenChengGong},
								{value:'failure',text: ShiBai},
								{value:'unExecuted',text: WeiZhiXing},
								{value:'inExecuted',text: ZhengZaiZhiXing}
							],
							value: ''
						">
				</li>
				<li style="position: relative;">
					<label ><%=rb.getString("KaiShiShiJian")%></label><br>
					<input name="queryStartTime" id="mr_query_start_time" class="easyui-datetimebox border border-box searchbox" data-options="editable:false">
					--
					<label style="position: absolute;z-index: 100; top: 0px;"><%=rb.getString("JieShuShiJian")%></label>
					<input name="queryEndTime" id="mr_query_end_time" class="easyui-datetimebox border border-box searchbox" data-options="editable:false">
				</li>
			</ul>
			<div>
				<a href="#" class="linkbutton linkbutton_trend" onclick="mraAdvanceQueryTaskDatagrid()"><span><%=rb.getString("ChaXun")%></span></a>
				<a href="#" class="linkbutton linkbutton_nowanna" onclick="resetkpiAdvanceQuery()"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
			</div>
		</div>
	</form>    
</div>
<!-- 工具栏 - MR文件管理  -->
<div id="toolbar_mr_fileMgmt_grid" class="toolbarContainer">
    <div class="queryGroup">
   	   <input id="searchFileList" name="search_text" placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>">
	   <b class="el-icon el-icon-common-search" onclick="mrFileListQuery()"></b>
    </div>
</div> 
	<!-- 菜单生成 -->
<div class="wrap">
    <div id="mn"></div>
</div>
<!-- 生成操作菜单列表  -->
<div class="wrap">
    <div id="mrFileOp"></div>
</div>

<!-- 下载任务界面  -->
<div id="downloadFilePage" class="mrRightHideDiv slidebarPanel flex-ctn" style="width:920px;position:absolute;right:-1000px;top:0px;bottom:0px;z-index:100;">
	<div class="slidebarTitleDiv" style="padding-left:0px;">
		<ul class="slidebarTitleContainer">
			<li class="default"><%=rb.getString("WenJianXiaZai")%></li>
		</ul>
		<div class="slideIcon el-icon el-icon-close" onclick="closeMrDownloadPanel()"></div>
	</div>
	<div class="slideBody" style="flex:1 auto;overflow:auto;">
		<div class="slideCont" style="height:100%;">
			<table id="mr_fileList_grid"></table>
		</div>
	</div>	
</div>

<!-- 工具栏 - MR文件下载  -->
<div id="toolbar_mr_fileList_grid" class="toolbarContainer">
    <form>
		<label style="padding-left:15px;"></label>
		<input id="startTime_mrFileList" type="text" class="easyui-datetimebox" style="width:160px;height:26px;" />
		<span> -- </span>
		<input id="endTime_mrFileList" type="text" class="easyui-datetimebox" style="width:160px;height:26px;margin-right:30px;" />
		
		<div style="display:inline-block;margin-left:20px;">
	   		<span class="el-button el-button--primary" onclick="queryMrFiles()"><%=rb.getString("ChaXun")%></span>
	   		<span class="el-button el-button--primary" onclick="resetMrFiles()"><%=rb.getString("ChaXunChongZhi")%></span>
	   		<span class="el-button el-button--primary" onclick="downloadMrMeasFile('batch')"><%=rb.getString("XiaZai")%></span>
		</div>
    </form>
    <div id="timeErrText"></div>
</div> 
<!-- 新建任务、查看任务属性 -->
<div id="mrTaskListInfo" class="slidebarPanel flex-ctn" style="width:920px; background: #FFFFFF;">
	<div class=" slidebarTitleContainer" style="font-size: 16px;font-weight: bold; padding: 20px;">
		<span></span>
		<div class="newIconBoxCls-bt" placeholder="<%=rb.getString("GuanBi")%>" style='top: 10px; right: 10px;'>
			<span class="el-icon el-icon-circle-close" onclick="closeMrTaskPanel()"></span>
		</div>
	</div>
	<div class="overflow-cls" style="flex-direction: column;">
		<div class="slideBody" style="min-width:900px;">
			<form id="gsmConfigForm" class="slideCont">
				<div class="form-group">
					<div class="group-title">
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
					</div>
					<div class="form-wrap">
						<div class="form-item">
							<label class="form-title"><%=rb.getString("RenWuMingCheng")%></label>
							<div class="mr-form-item-wrap">
								<input type="text" class="border border-box item readonlyClass" name="taskName" id="mr_taskName" mrReadonly value="${GSM_BSIC}" onblur="mrValidator.validMrTaskName(this)" maxlength="80"/>
								<span id="task_name_span" style="display: none"></span>
								<input type="hidden" name="taskId" id="mr_taskId">
								<input type="hidden" id="mr_taskStatus">
								<input type="hidden" id="mr_taskStartTime">
							</div>
							<span class="mrPromptTitle" id="GSM_BSIC_err"><%=rb.getString("QingShuRu")%><%=rb.getString("RenWuMingCheng")%></span>
						</div>
					</div>
				</div>
				<div class="form-group measureSet">
					<div class="group-title">
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("CeLiangSheZhi")%></span>
					</div>
					<div class="form-wrap">
						<div class="form-item" style="flex: 1 1 80%;">
							<label class="form-title"><%=rb.getString("CeLiangLeiXing")%></label>
							<div class="mr-form-item-wrap mrSingleSelectItem">
								<input type="checkbox" name="type" id="type_MRS"  checked value="MRS" style="margin-left:0;"/>MRS
								<input type="checkbox" name="type" id="type_MRE"  checked value="MRE"/>MRE
								<input type="checkbox" name="type" id="type_MRO"  checked value="MRO"/>MRO
							</div>
							<span class="mrPromptTitle" id="GSMSIG_LEVEL_THRESH_err"><%=rb.getString("QingShuRu")%><%=rb.getString("CeLiangLeiXing")%></span>
						</div>
						<div class="form-item">
							<label class="form-title"><%=rb.getString("CeLiangZhouQi")%></label>
							<div class="mr-form-item-wrap">
								<input  name="measurePeriod"  id="mr_measurePeriod"  style="width:140px;height:26px;" />		
							</div>
						</div>
						<div class="form-item">
							<label class="form-title"><%=rb.getString("ShangBaoZhouQi")%></label>
							<div class="mr-form-item-wrap">
								<input  name="reportPeriod"  id="mr_reportPeriod"  style="width:140px;height:26px;" />
							</div>
						</div>
						<div class="form-item">
							<label class="form-title"><%=rb.getString("KaiShiShiJian")%></label>
							<div class="mr-form-item-wrap">
								<input  name="startDate" id="startDateBox"  style="width:140px;height:26px;" /> - 
								<input name="startTime" id="startTimeSpinner"  style="width:140px;height:26px;" />				
							</div>
							<span class="mrPromptTitle" id="GSM_Neighbor_BCCHARFCN_err"><%=rb.getString("QingShuRu")%><%=rb.getString("KaiShiShiJian")%></span>
						</div>
						<div class="form-item">
							<label class="form-title" style="position:relative"><%=rb.getString("JieShuShiJian")%>
								<input type="checkbox" name="unlimitedTime" id="unlimitedTime" style="margin-left:10px;opacity: 0.8;"/>
								<span style="color: #aaa;"><%=rb.getString("BuXianZhiShiJian")%></span>
							</label>
							<div class="mr-form-item-wrap">
								<input name="endDate" id="endDateBox" style="width:140px;height:26px;" /> - 
								<input name="endTime" id="endTimeSpinner" style="width:140px;height:26px;" />		
							</div>
							<span class="mrPromptTitle"></span>
						</div>
					</div>
				</div>
				<div class="form-group last">
					<div class="group-title">
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
					</div>
					<div class="form-wrap" style="height:450px;">
						<div id="mrTaskList_device_datagrid"></div>
					</div>
				</div>
			</form>
		</div>
		<div id="saveMrTaskBox">
			<div  class="form-operations slideFooter">
				<a class="linkbutton" onclick="saveMrTask()"><span><%=rb.getString("QueDing")%></span></a>
				<a class="linkbutton linkbutton_nowanna" onclick="closeMrTaskPanel()"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
		</div>
	</div>
	<!-- 设备选择toolbar -->
	<div id="toolbar_add_temp_device_datagrid" style="padding:5px;">
		<input name="deviceGroup" id="tempAddDeviceGroup" class="easyui-combobox"
			data-options="
				height: 26,
				url: '${ctx }/cell/cpeinfos/getDeviceGroupListByCell.action',
				textField: 'group_name',
				valueField: 'id',
				editable: false,
				loadFilter: function(rows){
					if(rows){
						rows.splice(0,0,{group_name:'All',id:''});
					}
					return rows;
				},
				value:'',
				onChange:changeDevice
			">
		<div class="queryGroup" style="margin:0 0 0 20px;">
			<input name="searchText" id="tempAddDeviceQuery" style="width:150px;" placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>">
			<b class="el-icon el-icon-common-search" onclick='queryMrDevice()'></b>
		</div>
	</div>
</div>
<!-- 查看任务详情 -->
<div id="task_view_slider" class="slideDiv flex-ctn slide-position-bottom">
	<div class="slidebarTitleDiv">
		<span><%=rb.getString("ZhiXingJieGuo")%></span>
		<div class="slideIcon el-icon el-icon-close" onclick="$('#task_view_slider').removeClass('show')"></div>
	</div>
	<div class="flex-item flex-ctn" style="overflow: hidden">
		<table id="task_view_details_list"></table>
	</div>
</div>

<script type="text/javascript">
	// 国际化处理
	var XiaZai = '<%=rb.getString("XiaZai")%>';
	var ShanChu = '<%=rb.getString("ShanChu")%>';
	var ShiBai = '<%=rb.getString("ShiBai")%>';
	var XiuGai = '<%=rb.getString("XiuGai")%>';
	var JieGuo = '<%=rb.getString("JieGuo")%>';
	var XinXi = '<%=rb.getString("XinXi")%>';
	var KaiShi = '<%=rb.getString("KaiShi")%>';
	var ZanTing = '<%=rb.getString("ZanTing")%>';
	var ZhongZhi = '<%=rb.getString("ZhongZhi")%>';
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
	var ZhengZaiZhiXing = '<%=rb.getString("ZhengZaiZhiXing")%>';
	var ShiJian = '<%=rb.getString("ShiJian")%>';
	var QueRen = '<%=rb.getString("QueRen")%>';
	var QueRenShanChuRenWu = '<%=rb.getString("QueRenShanChuRenWu")%>';
	var QingXuanZeSheBei = '<%=rb.getString("QingXuanZeSheBei")%>';
	var mrMeasSearchText = "";
	/* 校验器 */
	var mrValidator = {
		validMrTaskName: function(el){
			var requiredMsg = '<%=rb.getString("QingShuRu")%><%=rb.getString("RenWuMingCheng")%>',
				existMsg = '<%=rb.getString("RenWuMingChengYiCunZai")%>',
				errorDom = $('#GSM_BSIC_err'),
				inputVal = $(el).val()||'',
				orgVal = '';
			if(el.attributes.value) orgVal = el.attributes.value.value;
				
			if(inputVal.trim()){
				if(orgVal.trim() != inputVal.trim()){
					$.post('${ctx}/cell/perfmgmt/mrcustomize/taskNameExist.action',{
						taskName: inputVal.trim()
					},function(data){
						if(data.message == 'true') {
							errorDom.show().text(existMsg);
						}else {
							errorDom.hide();
						}
					},'json');
				}else errorDom.hide();
				
			}else{
				errorDom.show().text(requiredMsg);
			}
		},
		validDate: function(newVal, oldVal){
			var requiredMsg = '<%=rb.getString("QingShuRu")%><%=rb.getString("KaiShiShiJian")%>',
				notLessMsg = '<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>',
				endRequiredMsg = '<%=rb.getString("QingShuRu")%><%=rb.getString("JieShuShiJian")%>',
				errorDom = $('#GSM_Neighbor_BCCHARFCN_err'),
				startTime = $('#startDateBox').datebox('getValue')+' '+$('#startTimeSpinner').datetimespinner('getValue'),
				endTime = $('#endDateBox').datebox('getValue')+' '+$('#endTimeSpinner').datetimespinner('getValue'),
				islimited = $('#unlimitedTime').prop('checked');
			
			if(islimited){
				errorDom.hide();
			}else {
				if(endTime.trim()){
					if(new Date(startTime).getTime()-new Date(endTime).getTime()>=0) errorDom.show().text(notLessMsg);
					else errorDom.hide();
				}else errorDom.show().text(endRequiredMsg);
			}
		}
	};

	//创建定时器
	var timer_mrtaskreload; 
	$(function () {
		//关闭 loading 加载
		closeLoading(); 
		$("#mr_fileList_grid").datagrid({"toolbar":""});
		$("#mrFileBtn span").css("border-top", "none");
		//MR任务列表初始化加载
		$("#mrTaskListDatagrid").datagrid({
			//任务管理表格请求接口
			url : '${ctx}/cell/perfmgmt/mrcustomize/getCustomizeMRTask.action',
			border:false,
			fit:true,
			toolbar:'#toolbar_mrTaskListDatagrid',
			rownumbers:true,
			fitColumns:true,
			pagination:true,
			pagePosition:'bottom',
			striped:true,
			singleSelect:true,
			idField:'task_id',
			queryParams: {timeZone: timeZone},
			columns:[[
					{ field:'task_id',hidden:true},
					{ field:'op',title:'',width:30,fixed:true,formatter:operationsCont },
					{ field:'task_name',title:'<%=rb.getString("RenWuMingCheng")%>',width:150, formatter: mrTaskNameFmt },
					{ field:'creator',title:'<%=rb.getString("ChuangJianZhe")%>',width:80}, 
					{ field:'create_time',title:'<%=rb.getString("ChuangJianShiJian")%>',width:100},
					{ field:'task_status',formatter: mrTaskStatusFormatter,title:'<%=rb.getString("ZhuangTai")%>',width:60},
					{ field:'task_result',formatter: mrTaskResultFormatter,title:'<%=rb.getString("JieGuo")%>',width:60},
					{ field:'start_time',title:'<%=rb.getString("KaiShiShiJian")%>',width:100}, 
					{ field:'end_time',title:'<%=rb.getString("JieShuShiJian")%>',width:100 }	          
				]],
			onLoadSuccess:datagridLoadSuccess
		})
		
		//取消 由 setInterval（） 设置的 定时器
		clearInterval(timer_mrtaskreload);
		
		/** 
		* 参数信息
		* @param page 当前页码
		* @param rows 显示记录行数
		* @param timeZone 时区
		* @return row[idField] 当前行的 task_id
		* @param row{object}: 表格行数据
		**/
		
		//创建定时器
		timer_mrtaskreload = setInterval(function(){ 
			var mrTb = $("#mrTaskListDatagrid"); 
			
			if(mrTb && mrTb.length){
				var opts = mrTb.datagrid('options'),
					params = {page: opts.pageNumber, rows: opts.pageSize}, 
					idField = opts.idField;
				
				//将参数合并
				$.extend(params,opts.queryParams); 
				
				$.post(opts.url,params,function(data){
					if(data){
						var ids = data.rows.map(function(row){
							return row[idField] 
						});
						
						mrTb.datagrid('getRows').map(function(row){
							if(!ids.includes(row[idField])) {
								var idx = mrTb.datagrid('getRowIndex',row[idField]);	
								//删除行
								mrTb.datagrid('deleteRow',idx); 
							}
						});
						
						data.rows.map(function(row){
							var index = mrTb.datagrid('getRowIndex',row[idField]);
							if(index>=0) mrTb.datagrid('updateRow',{index: index,row: row});
							else mrTb.datagrid('appendRow',row);						
						});
					}
				},'json');
				
				try{
					$('#task_view_details_list').datagrid('reload');
				}catch(e){}
			}else {
				clearInterval(timer_mrtaskreload);
			}
		},6000);
		
		//点击页面其他位置，隐藏操作下拉选项菜单
		$(document).click(function(e){
			var e = e || window.event;
			var elem = e.target || e.srcElement;
			
			while(elem){
				if($(elem).hasClass('el-icon-operation-more') || elem.className == 'circleBg add_circle' || elem.className == 'toolbarContainer datagrid-toolbar'){
					return
				} 
				elem = elem.parentNode;
			}
			
			$("#mn").css('display','none');
			$("#choseList").css('display','none');
		});
		
		//新建任务-基本信息点击事件，控制内容的显示或隐藏
		$('#gsmConfigForm .group-title .title-text').click(function(){
			var ctn = $(this).parents('.form-group'),
				title = $(this).parents('.group-title');
			if(ctn.hasClass('extend')){
				title.next().slideDown();
				ctn.removeClass('extend');
			}else{
				title.next().slideUp();
				ctn.addClass('extend')
			}
		});

		//文件列表初始化加载 
		$("#mr_fileMgmt_grid").datagrid({
			url:"${ctx}/cell/perfmgmt/mrreport/getCustomizeListPageData.action",
			border : false,
			fit : true,
			fitColumns : true,
			rownumbers : true,
			singleSelect : true,
			pagination : true,
			striped : true,
			idField : 'smallCellCode',
			toolbar: '#toolbar_mr_fileMgmt_grid',
			onBeforeLoad : beforeLoad_customize_grid,
			onLoadSuccess : loadSuccess_customize_grid,
			columns : [ [ 
					{field : 'smallCellCode',hidden : true}, 
					{field : 'operation',width: 30,fixed: true,title: '',formatter: mrFileMgmtOp},
					{field : 'status',sortable : true,title : '<%=rb.getString("ZhuangTai")%>',formatter : mrMeasuserStatus,fixed: true,width: 80},
					{field : 'serialNumber',sortable : true,width : 100,title : '<%=rb.getString("XiaoZhanBianMa")%>'}, 
					{field : 'hostName',sortable: true,width: 100,title: '<%=rb.getString("HostName")%>'} , 
					{field : 'cellId',sortable: true,width: 100,title: 'ECI'} , 
					{field : 'reportPeriod',width: 100,title: '<%=rb.getString("CeLiangZhouQi")%>(<%=rb.getString("FenZhongDaXie")%>)'}, 
					{field : 'startTime',sortable: true,width: 100,title: '<%=rb.getString("KaiShiShiJian")%>'} , 
					{field : 'endTime',sortable: true,width: 100,title: '<%=rb.getString("JieShuShiJian")%>'}, 
				]]
		});
		//查询绑定回车事件 
		$("#searchFileList").bind("keyup", function(e){
			if (e.keyCode == 13){
				mrFileListQuery();
			}
		});
		//点击页面其他位置，隐藏操作下拉选项菜单
		$(document).bind('click',function(e){
			var e = e || window.event;
			var elem = e.target || e.srcElement;
			while(elem){
				if($(elem).hasClass('el-icon-operation-more') ){
					return
				}
				elem = elem.parentNode;
			}
			$("#mrFileOp").fadeOut(200);
		})
	});

	/**
	* 任务管理表格-状态字段格式化
	* @param value{string}: 状态字段值
	* @param row{object}: 表格行数据
	* @param index{number}: 表格行数据对应索引
	**/
	function mrTaskStatusFormatter(value, row, index){
		/* var codes = {
				'waitting': DengDaiZhiXing,
				'on': KaiQi,//JinXingZhong
				'off': GuanBi,//YiJieShu
				'suspend': ZanTingStatus,//暂停
				'termination': YiZhongZhi,
				'unspport': '--'
			};
		return codes[value]; */
		if(value == 'waitting'){
			return "<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span>" + DengDaiZhiXing;
		}else if(value == 'on'){
			return "<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span>" + KaiQi;
		}else if(value == 'off'){
			return "<span class='el-icon el-icon-status-terminate' style='margin-right:5px;'></span>" + GuanBi;
		}else if(value == 'suspend'){
			return "<span class='el-icon el-icon-status-suspend' style='margin-right:5px;'></span>" + ZanTingStatus;
		}else if(value == 'termination'){
			return "<span class='el-icon el-icon-status-terminate' style='margin-right:5px;'></span>" + YiZhongZhi;
		}else if(value == 'unspport'){
			return "--";
		}
	}
	
	/**
	* 任务管理表格-结果字段格式化
	* @param value{string}: 结果字段值
	* @param row{object}: 表格行数据
	* @param index{number}: 表格行数据对应索引
	**/
	function mrTaskResultFormatter(value, row, index){
		var codes = {
				'success': ChengGong,
				'partialSuccess': BuFenChengGong,
				'failure': ShiBai,
				'unExecuted': WeiZhiXing,
				inExecuted: ZhengZaiZhiXing
			};
		return codes[value];
	}
	
	//模糊查询-任务名称 
	function mrListQuery(){
		var searchTxt = $("#mrTaskListQueryInput").val();
		$("#mrTaskListDatagrid").datagrid('load',{
			timeZone: timeZone,
			searchText: searchTxt
		});
	}

	//显示高级查询浮层 ------无用代码待删除
	/* function mrAdvanceQuerySlideFun(){
		judmentOtherChange("mrListAdvanceQuery");
		
		if($("#mrAdvanceQueryImg").attr("flag")=="1"){
			$("#mrListAdvanceQuery").slideDown(500);
			$("#mrAdvanceQueryImg").attr("flag","0");
			$("#mrAdvanceQueryImg").addClass('expanded');
		}else{
			$("#mrListAdvanceQuery").slideUp(400);
			$("#mrAdvanceQueryImg").attr("flag","1");
			$("#mrAdvanceQueryImg").removeClass('expanded');
		}	
	} */ 

	//高级查询 
	function mraAdvanceQueryTaskDatagrid(){
		$("#mrTaskListQueryInput").val("");
		var search_text_query = "",
			serial_number_query = $("#serialNum").val(),
			device_name_query = $("#cellName").val(),
			taskStatus = $('#taskStatus').combobox('getValue'),
			creator = $('#creatorUser').val(),
			taskResult = $('#taskResult').combobox('getValue'),
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

		$("#mrTaskListDatagrid").datagrid({
			queryParams : {
				searchText : search_text_query,
				taskName : serial_number_query,
				taskStatus: taskStatus,
				creator: creator,
				taskResult: taskResult,
				timeZone :timeZone,
				queryStartTime : start_time_query,
				queryEndTime : end_time_query
			}
		});
		$("#mrListAdvanceQuery").slideUp(100);	
		
		//无用代码待删除
		//$("#mrAdvanceQueryImg").removeClass('expanded');
		//$("#mrAdvanceQueryImg").attr("flag","1"); 
		//$("#mrAdvanceQueryTip").hide();
	}

	//高级查询条件重置
	function resetkpiAdvanceQuery(){	
		$("#serialNum").val("");
		$("#cellName").val("");
		$('#taskStatus').combobox('setValue','');
		$('#creatorUser').val('');
		$('#taskResult').combobox('setValue','');
		$("#deviceGroup").combobox('setValue', '');
		$("#mr_query_start_time").datetimebox('setValue', '');
		$("#mr_query_end_time").datetimebox('setValue', ''); 
		
	}

	/**
	* 任务管理表格-操作格式化
	* @param value{string}: 操作
	* @param rowData{object}: 表格行数据
	* @param rowIndex{number}: 表格行数据对应索引
	**/
	function operationsCont(value, rowData, rowIndex){	
		var rowDatas = rowData;
		value = "<div class='el-icon el-icon-operation-more' title='<%=rb.getString("CaoZuo")%>' onclick='choseMrOp(&quot;"+rowDatas.task_status+"&quot;,this)'></div>";
		return value;
	}
	/**
	* 任务管理表格-任务名称格式化
	* @param value{string}: 任务名称
	* @param rowData{object}: 表格行数据
	* @param rowIndex{number}: 表格行数据对应索引
	**/
	function mrTaskNameFmt(value, rowData, rowIndex) {
		return '<script type="text/html" style="display: block">'+value+'<\/script>';
	}
	/**
	* 点击行内【更多】按钮，下拉显示操作选项 
	* @param taskState{object}: 表格行数据
	* @param e{event}: 鼠标事件对象
	**/
	function choseMrOp(taskState,e){
		var thisTop = $(e).offset().top,
			isLicencesShow = true;
		
		/* 菜单状态 */
		var startShow = false,
			startFlag = false,
			waitFlag = false,
			endFlag = false,
			delFlag = false,
			editFlag = false,
			startShow = false;
		
		if(taskState == 'waitting'){
			waitFlag = true;
			endFlag = true;
		}else if(taskState == 'suspend'){
			startFlag = true;
			endFlag = true;
		}else if(taskState == 'on'){
			endFlag = true;
		}else if(taskState == 'off' || taskState == 'termination'){
			delFlag = true;
		}
		if(startFlag){
			startShow = true;
			editFlag = true;
		}
		
		var data = [
					{text: JieGuo, id: 1, cls: 'el-icon el-icon-operation-result', show: true},
					{text: KaiShi, id: 3, cls: 'el-icon el-icon-operation-start CODE_PERFORMANCE_MR hidden', show: startShow, disable: !startFlag},
					{text: ZanTing, id: 4, cls: 'el-icon el-icon-operation-awaiting CODE_PERFORMANCE_MR hidden', show: !startShow, disable: !waitFlag},
					{text: ZhongZhi, id: 5, cls: 'el-icon el-icon-operation-terminate CODE_PERFORMANCE_MR hidden', show: isLicencesShow, disable: !endFlag},
					{text: XinXi, id: 2, cls: 'el-icon el-icon-operation-info', show: !editFlag},
/* 					{text: XiuGai, id: 2, cls: 'el-icon el-icon-operation-edit CODE_PERFORMANCE_MR hidden', show: editFlag},
 */					{text: ShanChu, id: 6, cls: 'el-icon el-icon-operation-delete CODE_PERFORMANCE_MR hidden', show: isLicencesShow, disable: !delFlag}
					]
		
		showMenu(data);
		
		var allHeight = $(document).height();
		var isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
			tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0;
			
		if((allHeight - thisTop) <280){
			$('#mn').css({
				"top":thisTop - 300 - tabsHeight,
				"left":60,
			});
			if((allHeight - thisTop) <184){
				$('.item-child ').css({
					"top":"-54px",
				});
			}
		}else{
			$('#mn').css({
				"top":thisTop - 110 - tabsHeight,
				"left":60,
			});
		}
		
		$('#mn').show();	
	}
	
	/**
	* 操作-显示菜单
	* @param data{array}: 操作-菜单数据
	**/
	function showMenu(data){
		$('#mn').cmenu({data:data,click:clickEvent});           
	}
	
	/**
	* 操作-菜单点击事件
	* @param row{object}: 被点击的菜单行数据
	**/
	function clickEvent(row){
		$('#task_view_slider').removeClass('show');
		switch(row.id){
			case 1: //信息
				viewTaskDetails("enbStatistics");
				$('#mn').hide();
				break;
			case 2://属性 
				addMRTaskPage('view')
				$('#mn').hide();
				break;
			case 3://开始
				mrTaskProcess('start');
				$('#mn').hide();
				break;
			case 4://暂停
				mrTaskProcess('stop');
				$('#mn').hide();
				break;
			case 5://终止
				mrTaskProcess('terminate');
				$('#mn').hide();
				break;
			case 6://删除
				mrTaskProcess('del');
				$('#mn').hide();
				break;
		}
	}

	/**
	* 开始，暂停，终止，删除点击事件
	* @param type{string}: 对应状态
	**/
	function mrTaskProcess(type){
		var codes = {
				start: '${ctx}/cell/perfmgmt/mrcustomize/activeMRTask.action',
				stop: '${ctx}/cell/perfmgmt/mrcustomize/suspendMRTask.action',
				terminate: '${ctx}/cell/perfmgmt/mrcustomize/terminateMRTask.action',
				del: '${ctx}/cell/perfmgmt/mrcustomize/clearMRTask.action'
			},
			srow = $('#mrTaskListDatagrid').datagrid('getSelected'),
			taskId = srow?srow.task_id:'',
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
								$('#mrTaskListDatagrid').datagrid('reload');
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
						$('#mrTaskListDatagrid').datagrid('reload');
						toast(ChengGong,$('#omc_app_ctn'),'success');
					}else{
						toast(data.message,$('#omc_app_ctn'));
					}
				},'json');
			}
		}
	}
	
	//将新建和查看属性界面单独出jsp页面，以下方法需要转移和拆分 

	/**
	* 打开新建任务界面
	* @param type{string}: 对应状态
	**/
	function addMRTaskPage(type){
		var disableFlag = false;
		if(type=='view'){
			disableFlag = true;
			$("#saveMrTaskBox").addClass('mrReadonly');
		}else{
			$("#saveMrTaskBox").removeClass('mrReadonly');
		}
		$('#tempAddDeviceGroup').combobox();
		//执行结果内容隐藏
		$('#task_view_slider').removeClass('show');
		var formDom = document.querySelector('#gsmConfigForm');
		formDom.scrollTop = 0;
		formDom.reset();
		$("#startTimeSpinner").timespinner({
			disabled:disableFlag
		});
		$("#endTimeSpinner").timespinner({
			disabled:disableFlag
		});
		$("#startDateBox").datebox({
			disabled:disableFlag
		});
		$("#endDateBox").datebox({
			disabled:disableFlag
		});
		$('#gsmConfigForm .mrPromptTitle').hide();
		try{
			$("#mrTaskList_device_datagrid").pairgrid('clear');
			$("#mrTaskList_device_datagrid").pairgrid("reload",{searchText:''});
		}catch(e){}
		
		$("#mrTaskListInfo").addClass('loading').animate({right:'0px'},500,function(){ 

			var rightUrl = '${ctx}/pm/template/getSelectedEnbListPageData.action',
				srow = $('#mrTaskListDatagrid').datagrid('getSelected'),
				taskId = '',
				taskStatus = '',
				taskName = 'MR'+MRRenWuMoRen+'_'+user_code+'_'+formatDate(new Date(gloableTime)),
				startDate = getCurentDateStr(),
				endDate = getCurentDateStr(),
				mPeriod = '5120',
				rPeriod = '15',
				isReadonly = false;
			//测量类型
			var typeChecks = $('#type_MRS,#type_MRE,#type_MRO');
			typeChecks.on('click',function(){
				typeChecks.prop('checked',true);
			});
			//上报周期初始化 
			//id为num形式，涉及到下一步计算开始时间/结束时间的函数  
			$("#mr_reportPeriod").combobox({
				valueField:'id',
				textField:'text',
				editable: false,
				disabled: disableFlag,
				onSelect:changePeriod,
				onLoadSuccess:function(){
					$("#mr_reportPeriod").combobox("select",rPeriod);
				},
				data:[
					{
						'id':'15',
						'text':'15min'
					},
					{
						'id':'30',
						'text':'30min'
					},
					{
						'id':'60',
						'text':'60min'
					}
				]
			})
			$('.readonlyClass').attr('readonly',disableFlag);
			$('#unlimitedTime').prop('disabled',disableFlag)
			if(type=='view') {
				taskId = srow?srow.task_id:'';
				taskStatus = srow?srow.task_status:'';
				rightUrl = '${ctx}/cell/perfmgmt/mrcustomize/getMRTaskproperty.action?taskId='+taskId;
				startDate = srow.start_time||'';
				endDate = srow.end_time||'';
				mPeriod = srow.statis_period;
				rPeriod = srow.report_period;
				$("#mr_reportPeriod").combobox("select",rPeriod);
				var mrTypes = srow.mr_type;
				
				$('#task_name_span').text(srow.task_name);
				taskName = $('#task_name_span').text();
				isReadonly = true;
				if(mrTypes) {
					mrTypes.split(',').map(function(item){
						$('#type_'+item.toUpperCase()).prop('checked',true);
					});
				}
				
				$("#startTimeSpinner").timespinner({
					value: startDate.substring(11)
				});
				$("#endTimeSpinner").timespinner({
					value: endDate.substring(11)
				});
				if(!endDate){
					$('#unlimitedTime').prop('checked',true).click();
				}
			}
			$('#mr_taskName').val(taskName);
			$('#mr_taskId').attr('value',taskId);
			$('#mr_taskStatus').attr('value',taskStatus);
			$('#mr_taskStartTime').attr('value',startDate);
			
			//获取日历对象
			$("#startDateBox").datebox({editable: false,value: startDate,disabled: disableFlag, onChange: mrValidator.validDate}).datebox('calendar').calendar({
				validator:noMoreThanToday
			});
				
			$("#endDateBox").datebox({editable: false,value: endDate,disabled: disableFlag, onChange: mrValidator.validDate}).datebox('calendar').calendar({
				validator:noMoreThanToday
			});
			
			//测量周期
			$("#mr_measurePeriod").combobox({
				valueField:'id',
				textField:'text',
				disabled: disableFlag,
				value: mPeriod,
				editable: false,
				data:[
					{
						'id':'2048',
						'text':'ms2048'
					},
					{
						'id':'5120',
						'text':'ms5120'
					},
					{
						'id':'10240',
						'text':'ms10240'
					},
					{
						'id':'1',
						'text':'min1'
					},
					{
						'id':'6',
						'text':'min6'
					},
					{
						'id':'12',
						'text':'min12'
					},
					{
						'id':'30',
						'text':'min30'
					},
					{
						'id':'60',
						'text':'min60'
					},
				]
			})
			
			//设备选择
			$("#mrTaskList_device_datagrid").pairgrid({
				readonly: isReadonly,
				idField : 'serialNumber',
				leftUrl : '${ctx}/pm/template/getEnbListPageData.action',
				rightUrl : rightUrl,
				border : false,
				fit : true,
				fitColumns : true,
				rownumbers : true,
				striped : true,
				singleSelect : true,
				pageSize : 100,
				pageList : [100, 150, 200, 250, 300],
				pagination : true,
				pagePosition : 'bottom',
				toolBar : "#toolbar_add_temp_device_datagrid",
				zone : [50,50],
				queryName : 'serialNumber,hostName,groupName',
				messages:{queryName:'<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / <%=rb.getString("SheBeiZu")%>'},
				leftColumns : [{
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
				rightColumns : [{
					field : 'smallCellCode',
					hidden : true
				}, {
					field : 'serialNumber',
					width : 100,
					title : '<%=rb.getString("XiaoZhanBianMa")%>(<%=rb.getString("HostName")%>)',
					formatter: function(value,row,idx){
						var cellName = row.hostName? row.hostName:'unknown name';
						return value+'('+cellName+')';
					}
				}, {
					field: 'hostName',
					sortable: true,
					hidden : true
				}, {
					field : 'groupName',
					width : 60,
					title: '<%=rb.getString("SheBeiZu")%>'
				} ],
				
				leftBeforeLoad: function(param){
					var str = Math.random().toString();
					param.randomCode = str;
					param.timeZone = timeZone;
					return param;
				},
				
				rightBeforeLoad: function(param){
					param.timeZone = timeZone;
					return param;
				}
			});
		});
		
		
		if(type == 'add'){
			$(".slidebarTitleContainer > span",$('#mrTaskListInfo')).text('<%=rb.getString("XinJianRenWu")%>');
			$("#mrTaskListInfo").removeClass('loading');
		}
		
		if(type == 'view'){
			$(".slidebarTitleContainer > span",$('#mrTaskListInfo')).text('<%=rb.getString("XinXi")%>');
			
			setTimeout(function(){
				//公共方法
				initInputs($('#gsmConfigForm'));
				$("#mrTaskListInfo").removeClass('loading');
			},500);
		}
	}
	
	//设备选择-表格搜索
	function queryMrDevice(){
		var params = {
				searchText: $('#tempAddDeviceQuery').val(),
				like_fields: 'serial_number,host_name',
				'groupId': $('#tempAddDeviceGroup').combobox('getValue')||''
			};
		$("#mrTaskList_device_datagrid").pairgrid("reload",params);
	}
	
	/**
	* 表格-操作-信息
	* @param taskId{string}：行数据的唯一标识
	**/
	function viewTaskDetails(taskId){
		$('#task_view_slider').addClass('show');
		
		//执行结果表格
		var tb = $('#task_view_details_list'),
			columns = [
				{field:'serial_number',title: JiZhangBianMa,width: 100},
				{field:'host_name',title: JiZhangMingChen,width: 150},
				{field:'progress_status', formatter: viewMRStatusFormatter,title: ZhuangTai,width: 100},
				{field:'progress_result', formatter: viewMRResultFormatter,title: JieGuo,width: 100},
				{field:'failure_reason', formatter: viewMRReasionFormatter,title: ShiBaiYuanYin,width: 100},
				{field:'run_time',title: ShiJian,width: 100}
			];
		
		var srow = $('#mrTaskListDatagrid').datagrid('getSelected'),
			url = '${ctx}/cell/perfmgmt/mrcustomize/getMRCellPage.action';
		
		tb.datagrid({
			fit: true,
			striped: true,
			border: false,
			fitColumns: true,
			rownumbers: true,
			pagination: true,
			queryParams: {timeZone: timeZone, task_id: srow.task_id},
			columns: [columns],
			url: url
		});
	}

	/**
	* 表格-状态字段格式化
	* @param value{string}: 状态字段值
	* @param row{object}: 表格行数据
	* @param idx{number}: 表格行数据对应索引
	**/
	function viewMRStatusFormatter(value,row,idx){
		var codes = {
				'waitting': DengDaiZhiXing,
				'on': '<%=rb.getString("Kai")%>',
				'off': '<%=rb.getString("Guan")%>',
				'suspend': ZanTingStatus,
				'termination': YiZhongZhi
			};
		return codes[value];
	}
	
	/**
	* 表格-结果字段格式化
	* @param value{string}: 状态字段值
	* @param row{object}: 表格行数据
	* @param idx{number}: 表格行数据对应索引
	**/
	function viewMRResultFormatter(value,row,idx) {
		var codes = {
				unExecuted: WeiZhiXing,
				openSuccess: '<%=rb.getString("KaiQi")%><%=rb.getString("ChengGong")%>',
				openFailure: '<%=rb.getString("KaiQi")%><%=rb.getString("ShiBai")%>',
				closeSuccess: '<%=rb.getString("GuanBi")%><%=rb.getString("ChengGong")%>',
				closeFailure: '<%=rb.getString("GuanBi")%><%=rb.getString("ShiBai")%>',
				termination: YiZhongZhi,
				reboot: '<%=rb.getString("ChongQi")%>',
				rebootSuccess: '<%=rb.getString("ChongQi")%><%=rb.getString("ChengGong")%>',
				rebootFailure: '<%=rb.getString("ChongQi")%><%=rb.getString("ShiBai")%>'
			};
		
		return codes[value];
	}
	
	/**
	* 表格-失败原因字段格式化
	* @param value{string}: 状态字段值
	* @param row{object}: 表格行数据
	* @param idx{number}: 表格行数据对应索引
	**/
	function viewMRReasionFormatter(value,row,idx){
		var codes = {
				rebootTimeOut: '<%=rb.getString("ChongQiChaoShi")%>',
				timeOut: '<%=rb.getString("MingLingXiaFaChaoShi")%>'
			};
		if(codes[value]) return codes[value];
		else return value;
	}
	
	/**
	* 开始时间，结束时间格式化
	* @param day{object}日期
	**/
	function noMoreThanToday(day){
		var timestamp = Date.parse(new Date(gloableTime));
		var lasttime = timestamp - 86400000;
		var last = new Date(lasttime);
		return day > last;
	}

	//拼接年-月-日，时：分 做补零
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

	//修改上报周期对应的开始时间变化规则 
	function changePeriod(){
		var periodVal = $("#mr_reportPeriod").combobox("getValue");
		var currDate = new Date(gloableTime);
		var h = currDate.getHours();
		var m = currDate.getMinutes();
		var startMins = periodVal * ( parseInt( m / periodVal ) +1 );
		
		if( startMins == 60){
			h++;
			m = 00;
		}else{
			m = startMins
		}
		
		$("#startTimeSpinner").timespinner({
			step:periodVal,
			editable:false,
			value:h+':'+m, 
			onChange: mrValidator.validDate
		})
		
		$("#endTimeSpinner").timespinner({
			step:periodVal,
			editable:false,
			value:(h+1)+':'+m, 
			onChange: mrValidator.validDate
		})
		
	}

	//不限制时间点击事件
	$("#unlimitedTime").click(function(){
		if($(this).is(':checked')){
			$("#endTimeSpinner").timespinner({
				disabled:true
			});
			$("#endDateBox").datebox({
				disabled:true
			});
		}else{
			$("#endTimeSpinner").timespinner({
				disabled:false
			});
			$("#endDateBox").datebox({
				disabled:false
			});
		}
		mrValidator.validDate();
	})

	//新建任务-点击确定事件
	function saveMrTask() {
		var validFlag = true;
		/**
		* 提示内容
		* @param idx{number} 索引
		* @param item{object} 提示内容
		**/
		$('#gsmConfigForm .mrPromptTitle').each(function(idx,item){
			if($(item).is(':visible')) validFlag = false;
		});
		
		if (validFlag == false) {
			return;
		}
		
		var params = $('#gsmConfigForm').serializeJSON(),
			type = '';
		
		$('#gsmConfigForm [name=type]:checked').each(function(idx,item){
			if(type) type += ',' + item.value;
			else type = item.value;
		});
		
		var rtb = $("#mrTaskList_device_datagrid_right"),
			datas = rtb.datagrid("getRows"),
			codestr = '',
			namestr = '',
			snstr = '',
			delCodeStr = '',
			oList = rtb.data('originalCheckList');
		if(datas && datas.length){
			datas.map(function(item,idx){
				if(!oList.includes(item.serialNumber)){
					if(codestr || namestr || snstr){
						codestr += ','+item.smallCellCode;
						namestr += ','+item.hostName;
						snstr += ','+item.serialNumber;
					}else {
						codestr += item.smallCellCode;
						namestr += item.hostName;
						snstr += item.serialNumber;
					}
				}
			});
			
			var snList = datas.map(function(item,idx){return item.serialNumber; });
			oList.map(function(item,idx){
				if(!snList.includes(item)){
					if(delCodeStr) delCodeStr += ',';
					
					delCodeStr += item;
				}
			});
		}else{
			toast(QingXuanZeSheBei,$('#omc_app_ctn'));
			return false;
		}
		
		params['serialNumberDelStr'] = delCodeStr;
		params['timeZone'] = timeZone;
		params['smallCellStr'] = codestr;
		params['hostNameStr'] = namestr;
		params['serialNumberStr'] = snstr;
		params['type'] = type;
		params['task_status'] = 'on';
		params['startDate'] += ' '+params['startTime']+':00';
		
		if(!$("#unlimitedTime").is(':checked')){
			params['endDate'] += ' '+params['endTime']+':00';
		}
		
		var url = "${ctx}/cell/perfmgmt/mrcustomize/addMRTask.action";
		if(params.taskId) {/* 编辑 */
			url = "${ctx}/cell/perfmgmt/mrcustomize/updateMRTask.action";
			var mrStime = $('#mr_taskStartTime').val();
			if(new Date(mrStime).getTime()-new Date(gloableTime).getTime()<=0) {
				toast('<%=rb.getString("RenWuWuFaBianGeng")%>',$('#omc_app_ctn'));
				return ;
			}
			var originalList = $('#mrTaskList_device_datagrid_right').data('originalCheckList'),
				checkDatas = $('#mrTaskList_device_datagrid').pairgrid('getData'),
				checkIds = [],
				isChanged = false;
			if(checkDatas){
				checkIds = checkDatas.map(function(item){
					return item.serialNumber;
				})
			}
			if(originalList.sort().join(',') != checkIds.sort().join(',')) isChanged = true;
			
			if(isValueChanged('#gsmConfigForm') || isChanged) {
				params['task_status'] = $('#mr_taskStatus').val();
			}else {
				toast('<%=rb.getString("CanShuZhiMeiYouBianHua")%>',$('#omc_app_ctn'), 'success');
				return;
			}
		}
		/* 基站重复 */
		var isCellSelected = false;
		$.ajax({
			url: '${ctx}/cell/perfmgmt/mrcustomize/taskCellExist.action',
			type: 'post',
			dataType: 'json',
			async: false,
			data: {smallCellStr: params['smallCellStr']},
			success: function(data){
				if(data && data.rows.length){
					var isUsed = false,
						tNames = [];
					data.rows.map(function(row){
						if(row.progress_status!='off'){
							isUsed = true;
							if(!tNames.includes(row.task_name)) tNames.push(row.task_name);
						}
					});
					
					if(isUsed){
						var msgInfo = '<%=rb.getString("RenWuSheBeiJianCePrev")%>'+tNames.join('、')+'<%=rb.getString("RenWuSheBeiJianCeSuffix")%>';
						toast(msgInfo,$('#omc_app_ctn'));
					}else{
						$.messager.confirm(QueRen,'<%=rb.getString("SheBeiChongFuXuanZhe")%>',function(r){
							if(r){
								//请求修改参数
								$.post(url, params, function(data) {
									if (data["success"]) {
										closeMrTaskPanel();
										toast(ChengGong,$('#omc_app_ctn'), 'success');
										$("#mrTaskListDatagrid").datagrid("reload");
									} else {
										toast(data['message'],$('#omc_app_ctn'));
									}
								}, "json");
							}
						}).addClass('seriousConfirm');
					}
				}else{
					//请求修改参数
					$.post(url, params, function(data) {
						if (data["success"]) {
							closeMrTaskPanel();
							toast(ChengGong,$('#omc_app_ctn'), 'success');
							$("#mrTaskListDatagrid").datagrid("reload");
						} else {
							toast(data['message'],$('#omc_app_ctn'));
						}
					}, "json");
				}
			}
		})
	}

	//关闭新建任务 、 查看属性界面 
	function closeMrTaskPanel(){
		$('#mrTaskListInfo').animate({right:'-1000px'},500,function(){
			$('#gsmConfigForm .form-group').each(function(idx,item){
				var ctn = $(item),
					title = ctn.find('.group-title');
				if(ctn.hasClass('extend')) {
					title.next().slideDown();
					ctn.removeClass('extend');
				}
			});
		});
	}
	function changeDevice(newVal){
			var params = {
				'groupId': newVal || '',
				like_fields: 'serial_number,host_name'
			};
		$("#mrTaskList_device_datagrid").pairgrid("reload",params);
	}
	function mrHeadBtnClick(val){
		if(val == 'task'){
			$("#mrFileBtn span").css("border-top", "none");
			$("#mrTaskBtn span").css("border-top", "2px solid var(--main-color)");
			$("#mrFileTableBox").css("display", "none");
			$("#mrTaskTableBox").css("display", "block");
			$("#mrTaskListDatagrid").datagrid("resize");

		}else{
			$('#task_view_slider').removeClass('show')
			$("#mrTaskBtn span").css("border-top", "none");
			$("#mrFileBtn span").css("border-top", "2px solid var(--main-color)");
			$("#mrFileTableBox").css("display", "block");
			$("#mrTaskTableBox").css("display", "none");
			$("#mr_fileMgmt_grid").datagrid("resize");
		}
	}
	// 设备定制列表加载前事件
	function beforeLoad_customize_grid(param) {
		param.timeZone = timeZone;
		param.searchText = mrMeasSearchText;
	}

	// 设备定制列表加载成功事件
	function loadSuccess_customize_grid(data) {
		$(this).datagrid("enableContextmenuAutoSize");
		$(this).datagrid("fixRownumber");
	}

	//查询功能
	function mrFileListQuery(){
		mrMeasSearchText = $("#searchFileList").val();
		$("#mr_fileMgmt_grid").datagrid("reload");
	}

	//状态列图标展示 
	function mrMeasuserStatus(value, row, rowIndex) {
		//
		var imgText = "<img style='margin:5px 11px 0px 0;float:left' src ='${ctx}/css/images/main/monitor-ico/monitor-kpi-brolenomc.png' title='<%=rb.getString("ChuXianYiChang")%>'/>";
		if("0" == value){
			imgText = "<img style='margin:5px 11px 0px 0;float:left' src ='${ctx}/css/images/main/monitor-ico/monitor-kpi-offomc.png' title='<%=rb.getString("Guan")%>'/>"		
		} else if ("1" == value) {
			imgText = "<img style='margin:5px 11px 0px 0;float:left' src ='${ctx}/css/images/main/monitor-ico/monitor-kpi-normalomc.png' title='<%=rb.getString("ZhengChang")%>'/>"
		} else if ("2" == value) {
			imgText = "<img style='margin:5px 11px 0px 0;float:left' src ='${ctx}/css/images/main/monitor-ico/monitor-kpi-brolenomc.png' title='<%=rb.getString("ChuXianYiChang")%>'/>"
		}
		return imgText;
	}	

	// 设备定制列表操作列
	function mrFileMgmtOp(value, rowData, rowIndex){
		// 是否根据状态判断？？？ 
		var smallCellCode = rowData.smallCellCode;
		var rowDatas = rowData;
		
		value = "<div class='el-icon el-icon-operation-more' title='<%=rb.getString("CaoZuo")%>' onclick='choseMrFileOp("+rowDatas.status+",this)'></div>";
		return value; 
	}

	//点击行内【更多】按钮，下拉显示操作选项 
	function choseMrFileOp(taskState,e){
		var XiaZai = '<%=rb.getString("XiaZai")%>';
		var thisTop = $(e).offset().top;
		var data = [
					{text:XiaZai,id:1,cls:'el-icon el-icon-operation-download',show:true},
					]
		
		showFileMenu(data);
		var allHeight = $(document).height();
		var isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
			tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0;
		if((allHeight - thisTop) <280){
			$('#mrFileOp').css({
				"top":thisTop - 157 - tabsHeight,
				"left":50,
			});
			if((allHeight - thisTop) <184){
				$('.item-child ').css({
					"top":"-54px",
				});
			}
		}else{
			$('#mrFileOp').css({
				"top":thisTop - 110 - tabsHeight,
				"left": 60,
			});
		}
		
		$('#mrFileOp').show();	
	}

	//生成菜单 
	function showFileMenu(data){
		$('#mrFileOp').cmenu({data:data,click:clickFileEvent});           
	}

	//绑定操作菜单方法 
	function clickFileEvent(row){
		var srow = $('#mr_fileMgmt_grid').datagrid('getSelected'),
			rowCode = srow.smallCellCode,
			startTime = srow.startTime,
			endTime = srow.endTime;
		switch(row.id){
		case 1: //下载 
			goMrFileDownloadPage(rowCode,startTime,endTime);
			$('#mrFileOp').hide();
			break;
		}
	}

	//下载文件界面 
	function goMrFileDownloadPage(code,startTime,endTime){
		$('#downloadFilePage .top-tips').remove();
		$("#startTime_mrFileList").datetimebox("setValue",startTime);
		$("#endTime_mrFileList").datetimebox("setValue",endTime);
		try{
			$("#mr_fileList_grid").datagrid('clearSelections').datagrid('clearChecked');
		}catch(e){}
		$("#downloadFilePage").animate({right:'0px'},500,function(){ 
			$("#mr_fileList_grid").datagrid({
				url:"${ctx}/cell/perfmgmt/mrreport/getMRFilesTreeNodes.action",
				border : false,
				fit : true,
				fitColumns : true,
				rownumbers : true,
				//singleSelect : false,
				pagination : true,
				striped : true,
				idField : 'id',
				toolbar: '#toolbar_mr_fileList_grid',
				//selectOnCheck: true,
				//checkOnSelect:false,
				queryParams: {
					startTime: startTime,
					endTime: endTime,
					smallCellCode: code,
					timeZone: timeZone
				},
				columns: [[ 
					{field : 'smallCellCode',hidden : true}, 
					{field : 'id',hidden : true}, 
					{field : 'ck',checkbox:true},
					{field : 'fileName',sortable: true,width: 200,title: '<%=rb.getString("WenJianMing")%>'} , 
					{field : 'reportTime',sortable: true,width: 100,title: '<%=rb.getString("ShangBaoShiJian")%>'}, 
					{field : 'operation',width: 80,fixed: true,title: '<%=rb.getString("CaoZuo")%>',formatter: mrFileListOp}
				]]
			});
		})
	}

	//文件下载列表操作列
	function mrFileListOp(value, rowData, rowIndex){

		var smallCellCode = rowData.smallCellCode;
		var rowDatas = rowData;
		
		value = "<div class='el-icon el-icon-operation-download' title='<%=rb.getString("XiaZai")%>' style='width:100%;text-align:center;' onclick='downloadMrMeasFile(\"single\","+rowIndex+")'></div>";
		return value; 
	}

	//下载
	function downloadMrMeasFile(type,idx){
		var srow = $("#mr_fileMgmt_grid").datagrid('getSelected'),
			smallCellCode = srow?srow.smallCellCode:'';
		var startTime = $("#startTime_mrFileList").datetimebox("getValue"),
			endTime = $("#endTime_mrFileList").datetimebox("getValue"),
			fileRows = $("#mr_fileList_grid").datagrid('getSelections'),
			mrFilesPath = '',
			mrFilesName = '',
			params = {
				smallCellCode: smallCellCode,
				timeZone: timeZone
			};

		if( type == 'single'){
			//接口如何定义，是否根据类型传递参数
			var rows = $("#mr_fileList_grid").datagrid('getRows');
			mrFilesPath = rows[idx].id;
			mrFilesName = rows[idx].fileName;
		}else{
			// 批量下载时 处理时间范围与选中的文件之间的关系   
			
			<%-- if(startTime && endTime && startTime>endTime){
				//$("#timeErrText").html("<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>");
				toast('<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>',$('#downloadFilePage'));
				return;
			}
			if(new Date(endTime).getTime()-new Date(startTime).getTime()>31*24*3600000){
				//$("#timeErrText").html("<%=rb.getString("ZuiDuoKeXiaZai31TIanShuJu")%>");
				toast('<%=rb.getString("ZuiDuoKeXiaZai31TIanShuJu")%>',$('#downloadFilePage'));
				return;
			} --%>
			
			params.startTime = startTime;
			params.endTime = endTime;
			
			if(fileRows){
				fileRows.map(function(item,idx){
					if(idx){
						mrFilesPath += ','+item.id
						mrFilesName +=  ','+item.fileName
					}else{
						mrFilesPath += item.id
						mrFilesName += item.fileName
					}
				});
			}
		}
		params.mrFilesPath = mrFilesPath;
		params.mrFilesName = mrFilesName;
		
		if(params.mrFilesPath){
			/* $("#downloadMRFileForm").form('submit', {
				url: "${ctx}/cell/perfmgmt/mrreport/downloadMRFiles.action",
				onSubmit: function(param){
					$.extend(param,params);
				}
			}); */
			exportByForm("${ctx}/cell/perfmgmt/mrreport/downloadMRFiles.action", params);
			// closeMrDownloadPanel();
		}else {
			toast('<%=rb.getString("MeiYouXuanZeWenJian")%>',$('#downloadFilePage'));
		}
	}

	//关闭下载文件界面 
	function closeMrDownloadPanel(){
		$('#downloadFilePage').animate({right:'-1000px'},500);
		$('#startTime_mrFileList').datetimebox('setValue','');
		$('#endTime_mrFileList').datetimebox('setValue','');
		$("#mr_fileList_grid").datagrid('clearSelections').datagrid('clearChecked');
		$('#mr_fileMgmt_grid').datagrid('toolbar','');
	}

	function queryMrFiles(){
		var srow = $('#mr_fileMgmt_grid').datagrid('getSelected'),
			startTime = $('#startTime_mrFileList').datetimebox('getValue'),
			endTime = $('#endTime_mrFileList').datetimebox('getValue'),
			params = {
				startTime: srow.startTime,
				endTime: srow.endTime,
				smallCellCode: srow?srow.smallCellCode:'',
				timeZone: timeZone
			};
		params.startTime = startTime;
		params.endTime = endTime;
		
		if(startTime && endTime && new Date(startTime).getTime() - new Date(endTime).getTime()>0){
			toast('<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>',$('#downloadFilePage'),'',false);
		}else{
			$('#downloadFilePage .top-tips').remove();
			$("#mr_fileList_grid").datagrid('load',params);
		}
	}
	function resetMrFiles(){
		$('#startTime_mrFileList').datetimebox('setValue','');
		$('#endTime_mrFileList').datetimebox('setValue','');
	}
</script>