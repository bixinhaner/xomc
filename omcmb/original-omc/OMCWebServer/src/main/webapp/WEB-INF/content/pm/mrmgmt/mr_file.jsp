<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>


<style type="text/css">
    .downKpiMeasFileDiv{
    	width:300px;
    	height:300px;
    	position:absolute;
		z-index:950;
		right:0px;
		background:#fff;
		border:1px solid rgba(188,188,188,0.1);
		display:none;
		box-shadow:0 5px 15px #d8d8d8;
    }
    .timeSelectInput{
		height:50x;
		margin:20px 40px;    
    }
    .timeSelectTitle{
    	color:red;
    	margin-left:40px;
    	height:40px;
    	width:200px;
    	white-space:initial;
    }
    .mr-query label {
    	margin: 0px 8px 0px 0px;
    	padding: 0px;
    }
</style>

<div class="panelDefault" style="overflow:hidden;">
	<div class="singleTitle"><span><%=rb.getString("MRWenJianGuanLi")%></span></div>
	<div class="contentDiv" style="display:flex;">
		<table id="mr_fileMgmt_grid"></table>
	</div>
</div>

<!-- 工具栏 - MR文件管理  -->
<div id="toolbar_mr_fileMgmt_grid" class="toolbarContainer">
    <div class="queryGroup">
   	   <input id="searchFileList" name="search_text" placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>">
	   <b class="el-icon el-icon-common-search" onclick="mrFileListQuery()"></b>
    </div>
</div>  

<!-- 生成操作菜单列表  -->
<div class="wrap">
    <div id="mrFileOp"></div>
</div>

<!-- 下载任务界面  -->
<div id="downloadFilePage" class="rightHideDiv slidebarPanel flex-ctn" style="width:920px;position:absolute;right:-1000px;top:0px;bottom:0px;z-index:100;">
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

<%-- 下载文件的用的表单 --%>
<form method="post" style="display: none"  id="downloadMRFileForm"></form>
<script type="text/javascript">
var mrMeasSearchText = "";
var StartLessEnd = "<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>"

$(function() {
	closeLoading();
	
	//设备列表初始化加载 
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
})

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
	
	showMenu(data);
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
function showMenu(data){
	$('#mrFileOp').cmenu({data:data,click:clickEvent});           
}

//绑定操作菜单方法 
function clickEvent(row){
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
		toast(StartLessEnd,$('#downloadFilePage'),'',false);
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