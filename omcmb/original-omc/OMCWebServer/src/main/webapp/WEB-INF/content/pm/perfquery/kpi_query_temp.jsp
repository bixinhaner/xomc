<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style type="text/css">
	.viewDatagrid { width: 900px; height: 380px; margin: 30px 0 0 54px; }
	.partDicStrle { margin-bottom: 35px; }
	.divTitleStyle { font-size: 14px; color: #0F344D; font-weight: 700; margin: 40px 0 10px 30px; }
	.divLineStyle { background: #DCECF7; width: 900px; height: 2px; margin-left: 52px; margin-bottom: 30px; }
	.kpiTimeSlotDiv { display: inline-block; }
	.kpiTimeSlotDiv .selectRegion { display: inline-block; margin-left: 20px; vertical-align: top; }
	.kpiTimeSlotDiv p { display: inline-block; width: 33px; height: 33px; line-height: 33px; text-align: center; border: 1px solid #CCE1EF; cursor: pointer; font-size: 8px; margin: 6px; border-radius: 3px; }
	.kpiTimeSlotDiv p.unSelected { color: 1DA3FC; background: #fff; }
	.kpiTimeSlotDiv ul { display: block; }
	.kpiTimeSlotDiv li { display: inline-block; width: 30px; height: 30px; line-height: 30px; text-align: center; border-radius: 30px; cursor: pointer; font-size: 10px; margin: 6px; }
	.kpiTimeSlotDiv li.unSelected { color: 1DA3FC; background: #e6f5fa; }
	.kpiTimeSlotDiv .selected { background: #1da3fc; color: #fff; }
	.kpiTimeSlotDiv input { vertical-align: top; margin-right: 2px; }
	.kpiTimeSlotDiv label { margin-right: 30px; }
	.timeSlotTitle { display: inline-block; vertical-align: top; margin-top: 10px; width: 50px; font-weight: bold; color: #333333; }
	.errorTitle { height: 26px; line-height: 26px; min-width: 50px; font-size: 12px; color: red; display: none; }
	#importTemplateBlock { width: 530px; height: 450px; position: absolute; right: 20px; top: 52px; z-index: 100; display: none; }		
	.el-icon-status-disable:before { color: #C2C2C2; }
</style>
<!-- 查询模板页面 -->
<div>
	<!-- 0-4G-->
	<div class="eNBShow" style="display: none;">
		<div class="circleIcon CODE_PERFORMANCE_VIEW hidden" style="right:103px;">
			<span class="el-icon el-icon-circle-add" onclick="newTemplate()"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
		</div>
		<div class="circleIcon CODE_PERFORMANCE_VIEW hidden" style="right:55px;">
			<span class="el-icon el-icon-circle-import" onclick="inportTemplate()"></span>
			<div class="titleButtonText"><%=rb.getString("DaoRu")%></div>
		</div>
	</div>
	<!-- 1-5G-->
	<div class="gNBShow" style="display: none;">
		<div class="circleIcon CODE_GNB hidden" style="right:103px;">
			<span class="el-icon el-icon-circle-add" onclick="newTemplate()"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
		</div>
		<div class="circleIcon CODE_GNB hidden" style="right:55px;">
			<span class="el-icon el-icon-circle-import" onclick="inportTemplate()"></span>
			<div class="titleButtonText"><%=rb.getString("DaoRu")%></div>
		</div>
	</div>
	<!-- 2-eGW-->
	<div class="eGWShow" style="display: none;">
		<div class="circleIcon CODE_EGW hidden" style="right:103px;">
			<span class="el-icon el-icon-circle-add" onclick="newTemplate()"></span>
			<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
		</div>
		<div class="circleIcon CODE_EGW hidden" style="right:55px;">
			<span class="el-icon el-icon-circle-import" onclick="inportTemplate()"></span>
			<div class="titleButtonText"><%=rb.getString("DaoRu")%></div>
		</div>
	</div>
	
	<div class="circleIcon">
		<span class="el-icon el-icon-circle-close" onclick="closeTempOperateDiv()"></span>
		<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	</div>
</div>

<div class="el-card__header">
	<span><%=rb.getString("KPI_XingNengChaXunMoBan")%></span>
</div>

<div class="tabsContentDiv" style="border:none;top:49px;">
	<div style="height:100%;width:100%" >
		<table id="kpiTemplateDatagrid"></table>
	</div>
</div>

<div id='importTemplateBlock' class="cardPage"></div>

<!-- KPI查询模板toolbar -->
<div id="toolbar_kpiTemplateDatagrid" style="padding:10px 0;">
	<div class="queryGroup">
		<input name="searchText" id="kpi_tempDatagrid_query"
			placeholder="<%=rb.getString("MuBanMingCheng")%> / <%=rb.getString("GengXinRen")%>">
		<b class="el-icon el-icon-common-search" onclick="kpiTemplateDatagridQuery();"></b>
	</div>
</div>

<!-- 菜单生成 -->
<div class="wrap">
    <div id="operateTempPopDiv" class="z-index:100"></div>
</div>

<!-- 导出模板文件 -->
<form id="formTemplateDownload" style="display: none" method="post"></form>

<%-- 表单-用于下载错误模板列表文件 --%>
<form id="downloadFailureTemplate" style="display:none" method="post" action="${ctx}/pm/template/downloadFailureTemplate.action"></form>

<script type="text/javascript">
	var ShanChu = '<%=rb.getString("ShanChu")%>';
	var ShiBai = '<%=rb.getString("ShiBai")%>';
	var XiuGai = '<%=rb.getString("XiuGai")%>';
	var XinXi = '<%=rb.getString("XinXi")%>';
	var data = "";
	var kpiTempSearchText = "";
	var globleTempId = '';
	var curAddEnbGnbEgwType = sessionStorage.getItem('showTemplateEnbGnbOrEgw');

	$(function(){
		var curKpiTemplateTableUrl = '';
		
		//搜索回车事件
		$("#kpi_tempDatagrid_query").bind("keyup", function (event) {
		    if (event.keyCode == 13) {
		    	kpiTemplateDatagridQuery();
		    }
		});
		
		if(curAddEnbGnbEgwType == '0'){
			$('.eNBShow').show();
			$('.gNBShow').hide();
			$('.eGWShow').hide();
			curKpiTemplateTableUrl = '${ctx}/pm/template/getTemplateListPageData.action';
		}else if(curAddEnbGnbEgwType == '1'){
			$('.eNBShow').hide();
			$('.gNBShow').show();
			$('.eGWShow').hide();
			curKpiTemplateTableUrl = '${ctx}/gnb/pm/template/getTemplateListPageData.action';
		}else{
			$('.eNBShow').hide();
			$('.gNBShow').hide();
			$('.eGWShow').show();
			curKpiTemplateTableUrl = '${ctx}/egw/pm/template/getTemplateListPageData.action';
		}
		
		//kpi Template 表格数据加载
		$("#kpiTemplateDatagrid").datagrid({
			border:false,
			fit:true,
			fitColumns:true,
			url:curKpiTemplateTableUrl,
			queryParams:{
				timeZone:timeZone
			},
			toolbar:'#toolbar_kpiTemplateDatagrid',
			singleSelect:true,
			rownumbers: false,
			pagination:true,
			pagePosition:'bottom',
			striped:true,
			singleSelect:true,
			idField:'tempId',
			onBeforeLoad : beforeLoad_kpiTemplateDatagrid,
			onLoadSuccess: datagridLoadSuccess,
			columns: [[
	            {field: 'idx',fixed:true,width: 70,formatter: indexTempFormatrer},
	            {field: 'tempId', hidden: true},
				{field: 'operate',fixed:true,styler:setStyle,width: 30,title: '',formatter: operateTempGrid},
				{field: 'tempName',sortable:true,width: 130, title: '<%=rb.getString("MuBanMingCheng")%>'},
				{field: 'reportSwitch',sortable:true,width: 70, title: '<%=rb.getString("DingShiBaoBiaoZhuangTai")%>',formatter:reportStatusFmt},
				{field: 'updater',sortable:true,width: 80,title: '<%=rb.getString("GengXinRen")%>'},
				{field: 'updateTime',sortable:true,width: 100, title: '<%=rb.getString("GengXinShiJian")%>'},
				{field: 'description',sortable:true,width: 150, title: '<%=rb.getString("MiaoShu")%>'},
			]],
		})
		
	    closeLoading();
		
		//点击页面其他位置，隐藏操作下拉选项菜单
	    $(document).click(function(e){
	        var e = e || window.event;
	        var elem = e.target || e.srcElement;
	        while(elem){
	            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'showEnodebOp' || elem.className == 'slideDiv'){
	                return
	            } 
	            elem = elem.parentNode;
	        }
	    	$("#operateTempPopDiv").hide();
	    })
	})
	
	//关闭
	function closeTempOperateDiv(){
		//清除session
		sessionStorage.removeItem('newTemplateEnbGnbOrEgw');
		sessionStorage.removeItem('showTemplateEnbGnbOrEgw');
		$("#templateOperateDiv").slideUp(500,function(){
			$("#templateOperateDiv").html("");
		});
	}
	
	//条件搜索
	function kpiTemplateDatagridQuery(){
		kpiTempSearchText = $("#kpi_tempDatagrid_query").val();
		$("#kpiTemplateDatagrid").datagrid("reload");
	}
	
	//发送加载数据的请求前触发
	function beforeLoad_kpiTemplateDatagrid(param){
		param["timeZone"] = timeZone;
		var searchText = $("#kpi_tempDatagrid_query").val();
		
	    if (searchText != "") {
	        param["searchText"] = searchText;
	    }
	}
	
	//导入模板按钮点击
	function inportTemplate(){
		//此接口不区分 4G,5G,eGW
		var url = "${ctx}/pm/template/goImportTemplatePage.action"; // 导入模板页面
		sessionStorage.setItem('importTemplateEnbGnbOrEgw',curAddEnbGnbEgwType);
		$("#importTemplateBlock").slideDown(300,function(){
			//.load() 加载页面内容
			$('#importTemplateBlock').load( url,function(){
					
			});
	    });
	}

	/**
	* 是否为默认模板状态
	* @param value{string}: 值
	* @param rowData{object}: 表格行数据
	* @param rowIndex{number}: 表格行数据对应索引
	**/
	function indexTempFormatrer(value, rowData, rowIndex){
		var content = '',
			opts = $("#kpiTemplateDatagrid").datagrid('options'),
			page = opts.pageNumber,
			size = opts.pageSize;
		
		if(rowData.is_default === "true") {
			content = '<span class="status_checked selected-status" style="padding: 5px 10px; margin-top: 5px; display: inline-block;"></span>';
		}
		
		return content + '<div style="padding-right: 10px;float: right;">'+( (page-1)*size + rowIndex + 1 )+'</div>';
	}
	
	/**
	* 操作列格式化 
	* @param value{string}: 值
	* @param rowData{object}: 表格行数据
	* @param rowIndex{number}: 表格行数据对应索引
	**/
	function operateTempGrid(value, rowData, rowIndex){	
		var isCustomize = rowData.isCustomize,
			tempId = rowData.tempId,
			isDefault = rowData.is_default == "true" ? "true" : "false",
			isAllOperator = rowData.isAllOperator == 1;
		
		value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='choseTempOp(&quot;" + isCustomize +"&quot;,this,&quot;" + tempId +"&quot;,"+ isDefault +","+ isAllOperator +")'></div>";
		return value;
	}
	
	//点击行内【更多】按钮，下拉显示操作选项 
	function choseTempOp(isCustomize,e,tempId,isDefault,isAllOperator){
		var thisTop = $(e).offset().top;
		var thisLeft = $(e).offset().left;
		var XinXi = '<%=rb.getString("XinXi")%>';
		var DaoChu = '<%=rb.getString("DaoChu")%>';
		var JiChuPeiZhi = '<%=rb.getString("JiChuPeiZhi")%>';
		var HaloBJiChuPeiZhi = '<%=rb.getString("HaloBJiChuPeiZhi")%>';		
		var DingShiBaoBiao = '<%=rb.getString("DingShiBaoBiao")%>',
			YiWeiMoRen = '<%=rb.getString("YiWeiMoRen")%>',
			SheWeiMoRen = '<%=rb.getString("SheWeiMoRen")%>',
			deleteTempFlag = "",
			defaultFlag = false,
			opdisable = false,
			data = [];
			
		if(isCustomize == '0'){
			deleteTempFlag = false;
		}else{
			deleteTempFlag = true;
		}
		if(isDefault === true) {
			defaultFlag = true;
		}

		if(isAllOperator) {
			opdisable = true;
		}

		if(curAddEnbGnbEgwType == '0'){
			data = [
				{text: YiWeiMoRen,id:6,cls:'el-icon el-icon-operation-default CODE_PERFORMANCE_VIEW hidden',show: defaultFlag, tempId:tempId,disable: opdisable},
				{text: SheWeiMoRen,id:7,cls:'el-icon el-icon-operation-setDefault CODE_PERFORMANCE_VIEW hidden',show: !defaultFlag, tempId:tempId,disable: opdisable},
				{text: XinXi,id:1,cls:'el-icon el-icon-operation-info',show:true, tempId:tempId},
				{text: XiuGai,id:2,cls:'el-icon el-icon-operation-edit CODE_PERFORMANCE_VIEW hidden',show:true, tempId:tempId},
				{text: ShanChu,id:4,cls:'el-icon el-icon-operation-delete CODE_PERFORMANCE_VIEW hidden',show: deleteTempFlag, tempId:tempId,disable: opdisable},
		        {text: DaoChu,id:3,cls:'el-icon el-icon-operation-export',show:true, tempId:tempId},
				{text: DingShiBaoBiao,id:5,cls:'el-icon el-icon-operation-report CODE_PERFORMANCE_VIEW hidden', tempId:tempId},
	         ]
		}else if(curAddEnbGnbEgwType == '1'){
			data = [
				{text: YiWeiMoRen,id:6,cls:'el-icon el-icon-operation-default CODE_GNB hidden',show: defaultFlag, tempId:tempId,disable: opdisable},
				{text: SheWeiMoRen,id:7,cls:'el-icon el-icon-operation-setDefault CODE_GNB hidden',show: !defaultFlag, tempId:tempId,disable: opdisable},
				{text: XinXi,id:1,cls:'el-icon el-icon-operation-info',show:true, tempId:tempId},
				{text: XiuGai,id:2,cls:'el-icon el-icon-operation-edit CODE_GNB hidden',show:true, tempId:tempId},
				{text: ShanChu,id:4,cls:'el-icon el-icon-operation-delete CODE_GNB hidden',show: deleteTempFlag, tempId:tempId,disable: opdisable},
		        {text: DaoChu,id:3,cls:'el-icon el-icon-operation-export',show:true, tempId:tempId},
				{text: DingShiBaoBiao,id:5,cls:'el-icon el-icon-operation-report CODE_GNB hidden', tempId:tempId},
	        ]
		}else{
			data = [
				{text: YiWeiMoRen,id:6,cls:'el-icon el-icon-operation-default CODE_EGW hidden',show: defaultFlag, tempId:tempId,disable: opdisable},
				{text: SheWeiMoRen,id:7,cls:'el-icon el-icon-operation-setDefault CODE_EGW hidden',show: !defaultFlag, tempId:tempId,disable: opdisable},
				{text: XinXi,id:1,cls:'el-icon el-icon-operation-info',show:true, tempId:tempId},
				{text: XiuGai,id:2,cls:'el-icon el-icon-operation-edit CODE_EGW hidden',show:true, tempId:tempId},
				{text: ShanChu,id:4,cls:'el-icon el-icon-operation-delete CODE_EGW hidden',show: deleteTempFlag, tempId:tempId,disable: opdisable},
		        {text: DaoChu,id:3,cls:'el-icon el-icon-operation-export',show:true, tempId:tempId},
				{text: DingShiBaoBiao,id:5,cls:'el-icon el-icon-operation-report CODE_EGW hidden', tempId:tempId},
	         ]
		}
		
		showMenu(data);
		
		//判断菜单的位置
		var allHeight = $(document).height(),
			mHeight = $('#operateTempPopDiv > div:not(.menu-hidden)').length * 39;
		
		if((allHeight - 20 - thisTop) < mHeight){
			$('#operateTempPopDiv').css({
				"top": allHeight - mHeight - 110,
				"left":80,
			});
		}else{
			$('#operateTempPopDiv').css({
				"top":thisTop - 26,
				"left":80,
			});
		}
		$('#operateTempPopDiv').show();
	}
	
	/**
	* 操作-显示菜单
	* @param data{array}: 操作-菜单数据
	**/
	function showMenu(data){
		$('#operateTempPopDiv').cmenu({data:data,click:clickEvent});           
	}
	
	/**
	* 操作-菜单点击事件
	* @param row{object}: 被点击的菜单行数据
	**/
	function clickEvent(row){
	    switch(row.id){
		    case 1: //查看
		    	viewTemplate();
		    	 $('#operateTempPopDiv').hide();
		    	break;
		    case 2://修改
		    	modifyTemplate();
		    	 $('#operateTempPopDiv').hide();
		    	break;
		    case 3://导出
		    	exportTemplate();
		    	break;
		    case 4://删除
		    	deleteTemplate();
		    	 $('#operateTempPopDiv').hide();
			   	break;
			case 5://定时报表
				templateReport(row.tempId);
				 $('#operateTempPopDiv').hide();
			   	break;
			case 6://已为默认时
				 $('#operateTempPopDiv').hide();
			   	break;
			case 7://设置为默认时
				templateDefault(row.tempId);
				 $('#operateTempPopDiv').hide();
			   	break;
		} 
	}
	
	/**
	* 设置为默认模板
	* @param tempId{string}: 模板 id
	**/
	function templateDefault(tempId){
		var curSetDefaultUrl = '';
		
		if(curAddEnbGnbEgwType == '0'){
			curSetDefaultUrl = '${ctx}/pm/template/setDefaultTemplate.action';
		}else if(curAddEnbGnbEgwType == '1'){
			curSetDefaultUrl = '${ctx}/gnb/pm/template/setDefaultTemplate.action';
		}else{
			curSetDefaultUrl = '${ctx}/egw/pm/template/setDefaultTemplate.action';
		}
		
		$.post(curSetDefaultUrl, {tempId: tempId, is_default: true}, function(data) {
			if (data["success"]) {
				$("#kpiTemplateDatagrid").datagrid("reload");
	        } else {
	        	showMsg('error_msg',data["message"])
	        }
	    }, "json");
	}
	
	/**
	* 定时报表
	* @param tempId{string}: 模板 id
	**/
	function templateReport(tempId) {
		globleTempId = tempId;
		closeViewTemplateDiv();
		closeModifyTemplateDiv();
		closeAddTemplateDiv();
		
		sessionStorage.setItem('reportTemplateEnbGnbOrEgw',curAddEnbGnbEgwType);
		//此接口不区分 4G,5G,eGW
		$("#reportTemplateDiv").animate({right:'0px'},500,function(){
			$("#reportTemplateDiv").panel({
				 href:'${ctx}/pm/template/goRegularReport.action',
			})
		});
	}
	
	//查看模板详情
	function viewTemplate(){
		closeReportTemplateDiv();
		closeModifyTemplateDiv();
		closeAddTemplateDiv();
		sessionStorage.setItem('viewTemplateEnbGnbOrEgw',curAddEnbGnbEgwType);
		//此接口不区分 4G,5G,eGW
		$("#viewTemplateDiv").animate({right:'0px'},500,function(){
			$('#viewTemplateDiv').load( '${ctx}/pm/template/queryTemplateInfo.action',function(){
				
			});
		});
	}
	
	//修改模板
	function modifyTemplate(){
		closeReportTemplateDiv();
		closeViewTemplateDiv();
		closeAddTemplateDiv();
		//此接口不区分 4G,5G,eGW
		sessionStorage.setItem('modifyTemplateEnbGnbOrEgw',curAddEnbGnbEgwType);
		$("#modifyTemplateDiv").animate({right:'0px'},500,function(){
			$('#modifyTemplateDiv').load( '${ctx}/cell/perfmgmt/kpitemp/goUpdCustomQueryTemplatePage.action',function(){
				
			});
		});
	
	}
	
	//导出模板
	function exportTemplate(){
		var curExportTemplateUrl = '', datagrid = $("#kpiTemplateDatagrid").datagrid("getSelected"); 
		
		if(curAddEnbGnbEgwType == '0'){
			curExportTemplateUrl = '${ctx}/pm/template/exportTemplateIndicators.action';
		}else if(curAddEnbGnbEgwType == '1'){
			curExportTemplateUrl = '${ctx}/gnb/pm/template/exportTemplateIndicators.action';
		}else{
			curExportTemplateUrl = '${ctx}/egw/pm/template/exportTemplateIndicators.action';
		}
		
	    exportByForm(curExportTemplateUrl,{
	    	tempId: datagrid.tempId
	    });
	}
	
	//删除模板
	function deleteTemplate(){
		var curDeleteTemplateUrl = '', datagrid = $("#kpiTemplateDatagrid").datagrid("getSelected"); 
		
		if(curAddEnbGnbEgwType == '0'){
			curDeleteTemplateUrl = '${ctx}/pm/template/delTemplate.action';
		}else if(curAddEnbGnbEgwType == '1'){
			curDeleteTemplateUrl = '${ctx}/gnb/pm/template/delTemplate.action';
		}else{
			curDeleteTemplateUrl = '${ctx}/egw/pm/template/delTemplate.action';
		}
		
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuMuBan")%>", function (r) {
	        if (r) {
	           var params = {}
	           params.tempId = datagrid.tempId;
	           
	           $.post(curDeleteTemplateUrl, params, function(data) {
	    			if (data["success"]) {
	    				$("#kpiTempList").datagrid("reload");
	    				$("#kpiTemplateDatagrid").datagrid("reload");
	                } else {
	                	showMsg('error_msg',data["message"])
	                    return;
	                }
	            }, "json");
	        }
	    }).addClass('seriousConfirm');
	}
	
	//设置操作列单元格样式 
	function setStyle(){
		return 'position:relative';
	}
	
	//关闭定时报表弹出
	function closeReportTemplateDiv(){
		$("#reportTemplateDiv").animate({right:'-2000px'},500,function(){
			$(this).html("");
		});
	}
	
	//关闭查看模板弹窗
	function closeViewTemplateDiv(){
		$("#viewTemplateDiv").animate({right:'-2000px'},500,function(){
			$(this).html("");
		});
	}
	//关闭新建模板弹窗
	function closeModifyTemplateDiv(){
		$("#modifyTemplateDiv").animate({right:'-2000px'},500,function(){
			$(this).html("");
		});
	}
	
	//下载错误模板信息 
	function dowloadFailureFileConfig(){
		//未确定此方法的作用,未做4G,5G区分处理
		var url = $("#downloadFailureTemplate").attr('action');
		exportByForm(url, {});
		$('#winDowloadFailureFileConfig').window('close');
	}
	 
	/**
	* 格式化 定时报表状态
	* @param value{string}: 值
	* @param rowData{object}: 表格行数据
	* @param rowIndex{number}: 表格行数据对应索引
	**/
	function reportStatusFmt(value, rowData, rowIndex){
		var val ,
			QiYong = '<%=rb.getString("QiYong")%>',
			JinYong = '<%=rb.getString("JinYong")%>';
			
		if(value == '1'){
			val = '<span class="el-icon el-icon-status-enable" style="margin-right:5px;"></span>'+ QiYong;
		}else{
			val = '<span class="el-icon el-icon-status-disable"></span><span style="margin-left:5px;color:#C2C2C2">'+JinYong+'</span>'
		}
		return val; 
	}
</script>