<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
#importFileDiv{
	position:absolute;
	width:900px;
	/* height:92%; */
	right:-1000px;
	background:#fff;
	top:0px;
	bottom:0px;
	z-index:100;
	box-shadow:2px 3px 16px rgba(158,200,222,0.5);
	overflow:auto;
	/* border:1px solid #4AB3FF; */
}
.forbidden > span{
   	background: #B0CBDD !important;
}
.editFileUpgradeDiv,.viewFileUpgradeDiv{
	position:absolute;
	width:900px;
	/* height:94%; */
	right:-1000px;
	background:#fff;
	top:0px;
	bottom:0px;
	z-index:100;
	box-shadow:2px 3px 16px rgba(158,200,222,0.5);
	overflow:auto;
	/* border:1px solid #4AB3FF; */
}
.slideDownAllDiv{
	position:absolute;
	top: 0;
	left: 0;
	bottom: 0;
	width:100%;
	height:100%;
	background:#fff;
	z-index:100;
	border:none;
	display:none;
}

#enbSoftContent .el-card {
	border: none;
}
.panelDefault .upgradeTabsMainPage{
	left:0px;
}
.tabsTitle{
	border:none;
	color:#333;
}
.showFileOp div{
	padding-left:14px;
	border-bottom:none;
	width:auto;
}
.el-badge{
	position:relative;
}
.el-badge__content{
	position:absolute;
	top:6px;
	right:-4px;
	transform:translateY(-50%) translateX(100%);
	background-color:transparent;
	border-radius:10px;
	color:#fff;
	display:inline-block;
	font-size:10px;
	height:12px;
	line-height:11px;
	padding:0 6px;
	text-align:center;
	white-space:nowrap;
	cursor:default;
	border:1px solid transparent;
}
.el-icon-star-badge:before{
	color:#F3916C;
}
.linkStyle {
	margin-left:5px;
	color:#4D84FF;
	cursor:pointer;
}
.versionDetails{
		position:absolute;
		background:white;
		padding:10px 20px;
		box-shadow:4px 4px 19px 0 rgba(0,0,0,0.15);
		border:1px solid #d1ecf5;
		display:none;
		left:0;
		z-index:99999;
	}
</style>
<div class='panelDefault' style='overflow:hidden' id="software_file_app">
	<div id="enbSoftOperateDiv" class="slideDownAllDiv">
		<div id="enbSoftContent"></div>
	</div>
	<!-- 右上角导入按钮 -->
	<div id='upgradeImportDiv' class="circleIcon hidden placeholder-bt CODE_CPE_UPGRADE_FILE" style="right: 115px;" placeholder="<%=rb.getString("DaoRuWenJian")%>">		
		<span class="el-icon el-icon-circle-import" onclick="importUpgradeFile()"></span>
	</div>
	<template v-if="isImageTab">
		<div id='upgradeAddDiv' class="circleIcon placeholder-bt CODE_CPE_UPGRADE_IMAGE hidden" style="right: 60px;" placeholder="<%=rb.getString("XinJianRenWu")%>">		
			<span class="el-icon el-icon-circle-addTask" @click="addTask"></span>
		</div>
		<div id='upgradeViewDiv' class="circleIcon placeholder-bt CODE_CPE_UPGRADE_IMAGE hidden visible" placeholder="<%=rb.getString("RenWuLieBiao")%>">		
			<span class="el-icon el-icon-circle-taskList" @click="viewTask"></span>
		</div>
	</template>
	<template v-if="!isImageTab">
		<div id='upgradeAddDiv' class="circleIcon placeholder-bt" style="right: 60px;" placeholder="<%=rb.getString("XinJianRenWu")%>">		
			<span class="el-icon el-icon-circle-addTask" @click="addTask"></span>
		</div>
		<div id='upgradeViewDiv' class="circleIcon placeholder-bt" placeholder="<%=rb.getString("RenWuLieBiao")%>">		
			<span class="el-icon el-icon-circle-taskList" @click="viewTask"></span>
		</div>
	</template>
	<div class="tabsTitle upgradeLists" id="omcLogTabsDiv">
		<c:if test="${ isCpeFileUpload ==null}">
			<span tabtit="imageUpgradeDiv" onclick='closeWindow();changeClass(this)' class="CODE_ENB_UPGRADE_IMAGE hidden visible active" logtype="upgrade"><%=rb.getString("IMAGE")%></span>
			<span tabtit="ubootUpgradeDiv" onclick='closeWindow();changeClass(this)' class="eNbUbootUpdrade hidden visible" logtype="bios"><%=rb.getString("UBOOTBanBen")%></span>
			<span tabtit="patchUgradeDiv" onclick='closeWindow();changeClass(this)' class="CODE_ENB_UPGRADE_PATCH hidden visible" logtype="ca"><%=rb.getString("CABanBen")%></span>
			<span tabtit="fpgaUgradeDiv" onclick='closeWindow();changeClass(this)' class="CODE_ENB_UPGRADE_FPGA hidden visible" logtype="fpga"><%=rb.getString("FPGAShengJiWenJian")%></span>
		</c:if>
		<c:if test="${ isCpeFileUpload ==1}">
			<span tabtit="oducpeUpgradeDiv" @click="imageTabClick" class="CODE_CPE_UPGRADE_FILE CODE_CPE_UPGRADE_IMAGE hidden visible active" logtype="upgradecpe"><%=rb.getString("IMAGE")%></span>
			<c:if test="${isSuperAdmin == 1}">
				<span tabtit="cpeMidVersionDiv" @click="isImageTab = false" logtype="midVersion"><%=rb.getString("ZhongJianBanBen")%></span>
				<span tabtit="cpeModuleVersionDiv" @click="isImageTab = false" logtype="moduleVersion"><%=rb.getString("MoKuaiBanBen")%></span>
			</c:if>
		</c:if>		
	</div>
	<div class="tabsContentDiv upgradeTabsMainPage">
		<c:if test="${ isCpeFileUpload ==null}">
			<div class='imageUpgradeDiv'>
				<table id='fileinfo_upgrade'></table>
			</div>
			<div class='ubootUpgradeDiv'>
				<table id='fileinfo_bios'></table>
			</div>
			<div class='patchUgradeDiv'>
				<table id='fileinfo_ca'></table>
			</div>
			<div class='fpgaUgradeDiv'>
				<table id='fileinfo_fpga'></table>
			</div>
		</c:if>
		<c:if test="${ isCpeFileUpload ==1}">
			<div class='oducpeUpgradeDiv'>
				<table id='fileinfo_upgradecpe'></table>
			</div>
			<div class='cpeMidVersionDiv'>
				<table id='fileinfo_cpemidversion'></table>
			</div>
			<div class='cpeModuleVersionDiv'>
				<table id='fileinfo_cpemodule'></table>
			</div>
			<!-- <div class='iducpeUgradeDiv'>
				<table id='fileinfo_upgradeiducpe'></table>
			</div> -->
		</c:if>
	</div>
	<!-- 导入文件模块 -->
	<div id='importFileDiv' class="slidebarPanel"></div>
	<!-- 修改文件模块 -->
	<div class='editFileUpgradeDiv slidebarPanel'></div>
	<!-- 查看文件模块 -->
	<div class='viewFileUpgradeDiv slidebarPanel'></div>
</div>
<%-- 下载文件的用的表单 --%>
<form method="post" style="display: none"  id="downloadFileForm_filelib"></form>

<%-- 上传文件的用的表单 --%>
<form enctype="multipart/form-data" method="post" id="uploadFileForm_filelib">
    <input name="newFileName" id="newFileName" value="" hidden="true">
    <input name="fileSize" id="fileSize" value="" hidden="true">
    <input name="fileType" id="fileType" value="" type="hidden"/>
    <input name="deviceType" value="" type="hidden"/>
    <input name="product" value="" type="hidden"/>
    <input name="version" value="" type="hidden"/>
    <input name="to_who" value="" type="hidden"/>
	<input name="model_name" value="" type="hidden"/>
    <input name="desc" value="" type="hidden"/>
    <input name="uploadFile" id="uploadFile_filelib"  type="file" style="display: none;">
    <input name="md5" value="" type="hidden"/>
    <input name="recommend" value="" type="hidden"/>
</form>
<!-- 工具栏--按查询  -->
<c:if test="${ isCpeFileUpload ==null}">
<div id="toolbar_imageUpgrade" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='upgradeInput'   placeholder="<%=rb.getString("BanBen")%>">
		<b class="el-icon el-icon-common-search" onclick='$("#fileinfo_upgrade").datagrid("reload")'></b>
     </div>
</div>
<div id="toolbar_ubootUpgrade" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='biosInput'  placeholder="<%=rb.getString("BanBen")%>">
		<b class="searchResultImgChangeStyle" onclick='$("#fileinfo_bios").datagrid("reload")'></b>
     </div>
</div>
<div id="toolbar_patchUpgrade" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='caInput'  placeholder="<%=rb.getString("BanBen")%>">
		<b class="el-icon el-icon-common-search" onclick='$("#fileinfo_ca").datagrid("reload")'></b>
     </div>
</div>
<div id="toolbar_fpgaUpgrade" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='fpgaInput'  placeholder="<%=rb.getString("BanBen")%>">
		<b class="el-icon el-icon-common-search" onclick='$("#fileinfo_fpga").datagrid("reload")'></b>
     </div>
</div>
</c:if>
<c:if test="${ isCpeFileUpload ==1}">
<div id="toolbar_oducpeUpgrade" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='oducpeInput'  placeholder="<%=rb.getString("BanBen")%>">
		<b class="el-icon el-icon-common-search" onclick='$("#fileinfo_upgradecpe").datagrid("reload")'></b>
     </div>
</div>
<div id="toolbar_cpemidversion" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id="midVersionInput" placeholder="<%=rb.getString("BanBen")%>">
		<b class="el-icon el-icon-common-search" onclick='$("#fileinfo_cpemidversion").datagrid("reload")'></b>
     </div>
</div>
<div id="toolbar_cpemodule" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id="modelVersionInput" placeholder="<%=rb.getString("BanBen")%>">
		<b class="el-icon el-icon-common-search" onclick='$("#fileinfo_cpemodule").datagrid("reload")'></b>
     </div>
</div>

</c:if>

<!-- 菜单生成 -->
<div class="wrap">
    <div id="enbSoftwareFileMenu" class="showFileOp"></div>
</div>
<script>
var deviceTypeEditArr = [];
var operateType = 'enb';
var eventBus = new Vue();
	new Vue({
		el: '#software_file_app',
		data(){
			var url = '${ctx}/cell/version/toVersionMgr.action',
				isCPE = '${isCpeFileUpload}' == '1';
			if(isCPE) url = '${ctx}/cell/version/toCpeVersionMgr.action';
			return {
				slideTitle: '',
				slideUrl: url,
				isCPE: isCPE,
				isImageTab: true
			};
		},
		computed: {
			addBtClass(){
				return {
					'CODE_CPE_UPGRADE_IMAGE hidden': '${isSuperAdmin}' != '1',
					'circleIcon': true,
					'placeholder-bt': true
				}
			}
		},
		methods: {
			imageTabClick() {
				this.isImageTab = true;
				setTimeout(function(){
					document.body.click();
				},10);
			},
			addTask: function(){ // 新建任务
				var type = $('#omcLogTabsDiv span.active').attr('logType');
				if(this.isCPE) {// CPE的升级任务
					cpeSoftUpgradeVM.addCpeUpdadeTask();
					cpeSoftUpgradeVM.tableShow = false;
				}else { // eNb的升级任务
					if(type == 'upgrade') {
						enbSoftUpgradeVM.addUpgradeTask();
					}else if(type == 'bios') {
						
					}else if(type == 'ca') {
						enbSoftUpgradeVM.addPatchTask();
					}else if(type == 'fpga') {
						enbSoftUpgradeVM.addFpgaTask();
					}
					enbSoftUpgradeVM.tabsShow = false;
				}
				$("#enbSoftOperateDiv").slideDown(500);
			},
			viewTask: function(){ // 任务列表
				var type = $('#omcLogTabsDiv span.active').attr('logType'),
					activeName = 'first';
				if(this.isCPE) {
					//cpeSoftUpgradeVM.tableShow = true;
				}else {
					if(type == 'upgrade') {
						activeName = 'first';
					}else if(type == 'bios') {
						activeName = 'second';
					}else if(type == 'ca') {
						activeName = 'third';
					}else if(type == 'fpga') {
						activeName = 'fourth';
					}
				}
				var vm = this;
				$("#enbSoftContent").load(this.slideUrl,function(data){
					$.parser.parse(vm);
					if(vm.isCPE) {
						cpeSoftUpgradeVM.tableShow = true;
					}else{
						enbSoftUpgradeVM.activeName = activeName;
						enbSoftUpgradeVM.tabsShow = true;
					}
				});
				$("#enbSoftOperateDiv").slideDown(500);
			},
			closeSlide: function(){ // 关闭任务
				$("#enbSoftOperateDiv").slideUp(500);
			},
			triggerCancel: function(){ // 暂无用
				enbSoftUpgradeVM.cancelSlide();
			}
		},
		mounted(){
			$("#enbSoftContent").load(this.slideUrl,function(data){
				$.parser.parse(this);
			});
			eventBus.$off('enb-slide-up').$on('enb-slide-up',this.closeSlide);
			eventBus.$off('to-task-view').$on('to-task-view',this.viewTask);
		}
	});

	var isCpe = '${ isCpeFileUpload}';
	$(function(){
		closeLoading();
		setTimeout(function(){
			var code = isJumpToPage?isJumpToPage.code:'';
			var vid = isJumpToPage?isJumpToPage.vid:'';
			if(!code){
				 $(".upgradeLists span:visible:first").click();
			}else if(code == 'enodeb' && $(".upgradeLists span[logtype=upgrade]").is(":visible")){
				$(".upgradeLists span[logtype=upgrade]").click();
			}else if(code == 'enodeb_uboot' && $(".upgradeLists span[logtype=bios]").is(":visible")){
				$(".upgradeLists span[logtype=bios]").click();
			}else if(code == 'enodeb_patch' && $(".upgradeLists span[logtype=ca]").is(":visible")){
				$(".upgradeLists span[logtype=ca]").click();
			}else if(code == 'cpe_odu'&& $(".upgradeLists span[logtype=upgradecpe]").is(":visible")){
				$(".upgradeLists span[logtype=upgradecpe]").click();
			}else if(code == 'cpe_idu'&& $(".upgradeLists span[logtype=upgradecpe]").is(":visible")){
				$(".upgradeLists span[logtype=upgradecpe]").click();
			}
			var upgradeType = $("#omcLogTabsDiv span.active").attr("logtype");
			/* if(upgradeType == "upgrade"){
				$("#upgradeImportDiv").addClass("CODE_ENB_UPGRADE_IMAGE");
			}else if(upgradeType == "bios"){
				$("#upgradeImportDiv").addClass("eNbUbootUpdrade");
			}else if(upgradeType == "ca"){
				$("#upgradeImportDiv").addClass("CODE_ENB_UPGRADE_PATCH");
			}else if(upgradeType == "fpga"){
				$("#upgradeImportDiv").addClass("CODE_ENB_UPGRADE_PATCH");
			}else if(upgradeType == "upgradecpe"){
				$("#upgradeImportDiv").addClass("CODE_CPE_UPGRADE_IMAGE");
			}else if(upgradeType == "upgradeiducpe"){
				$("#upgradeImportDiv").addClass("CODE_CPE_UPGRADE_IMAGE");
			} */
		},20)
		$("#fileinfo_upgrade").datagrid({  // 初始化列表 软件系统升级文件
			url:'${ctx}/cell/version/queryfileInfosList.action?file_type=0',
			queryParams:{timeZone:timeZone},
			idField:'id',
			fit:true,
			fitColumns:true,
			nowrap:false,
			border:false,
			striped:true,
			singleSelect:true,
			rownumbers:true,
			pagination:true,
			pagePosition:'bottom',
			pageSize:100,
			pageList:[50,100,200,500],
			toolbar:'#toolbar_imageUpgrade',
			onBeforeLoad:function(param){
				param.searchText = $("#upgradeInput").val()
			},
			onLoadSuccess:function(){
				var code = isJumpToPage?isJumpToPage.code:'';
				var vid = isJumpToPage?isJumpToPage.vid:'';
				if(!code){
				}else if(code == 'enodeb' && $(".upgradeLists span[logtype=upgrade]").is(":visible")){
					if(vid){
						var index = $("#fileinfo_upgrade").datagrid("getRowIndex",vid);
						$("#fileinfo_upgrade").datagrid("selectRow",index);
					}
				}
				$(this).datagrid("fixRownumber");
				$(this).datagrid("enableContextmenuAutoSize");
			},
			columns:[[
				{field:'id',hidden:true},
				{field:'operation',formatter:moreFileFormatter,styler:setStyle,fixed:true, width:30,title:''},
				{field:'version', formatter:versionFmt,width:250,title:'<%=rb.getString("BanBen")%>'},
				{field:'product', width:150,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'},
				{field:'size', width:100,title:'<%=rb.getString("WenJianDaXiao")%>'},
				{field:'toWho',hidden:isCloudCore=='true'?false:true,formatter:towhoFormat, width:100,title:'<%=rb.getString("BanBenLeiXing")%>'},
				{field:'upload_time', width:140,title:'<%=rb.getString("ShangChuanShiJian")%>'}
			        ]]
		})
		$("#fileinfo_bios").datagrid({  // 初始化列表 PATCH  FPGA升级文件
		url:'${ctx}/cell/version/queryfileInfosList.action?file_type=2',
		queryParams:{timeZone:timeZone},
		idField:'id',
		fit:true,
		fitColumns:true,
		nowrap:false,
		border:false,
		striped:true,
		singleSelect:true,
		rownumbers:true,
		pagination:true,
		pagePosition:'bottom',
		pageSize:100,
		pageList:[50,100,200,500],
		toolbar:'#toolbar_ubootUpgrade',
		onBeforeLoad:function(param){
			param.searchText = $("#biosInput").val()
		},
		onLoadSuccess:function(){
			var code = isJumpToPage?isJumpToPage.code:'';
			var vid = isJumpToPage?isJumpToPage.vid:'';
			if(!code){
			}else if(code == 'enodeb_uboot' && $(".upgradeLists span[logtype=bios]").is(":visible")){
				if(vid){
					var index = $("#fileinfo_bios").datagrid("getRowIndex",vid);
					$("#fileinfo_bios").datagrid("selectRow",index);
				}
			}
			$(this).datagrid("fixRownumber");
			$(this).datagrid("enableContextmenuAutoSize");
		},
		columns:[[
			{field:'id',hidden:true},
			{field:'operation',formatter:moreFileFormatterUboot,styler:setStyle,fixed:true, width:30,title:''},
			{field:'version',formatter:versionFmt, width:250,title:'<%=rb.getString("BanBen")%>'},
			{field:'product', width:150,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'},
			{field:'size', width:100,title:'<%=rb.getString("WenJianDaXiao")%>'},
			{field:'toWho',hidden:isCloudCore=='true'?false:true,formatter:towhoFormat, width:100,title:'<%=rb.getString("BanBenLeiXing")%>'},
			{field:'upload_time', width:140,title:'<%=rb.getString("ShangChuanShiJian")%>'}
		        ]]
		})
		$("#fileinfo_ca").datagrid({ // 初始化列表 PATCH 版本升级文件
		url:'${ctx}/cell/version/queryfileInfosList.action?file_type=1',
		queryParams:{timeZone:timeZone},
		idField:'id',
		fit:true,
		fitColumns:true,
		nowrap:false,
		border:false,
		striped:true,
		singleSelect:true,
		rownumbers:true,
		pagination:true,
		pagePosition:'bottom',
		pageSize:100,
		pageList:[50,100,200,500],
		toolbar:'#toolbar_patchUpgrade',
		onBeforeLoad:function(param){
			param.searchText = $("#caInput").val()
		},
		onLoadSuccess:function(){
			var code = isJumpToPage?isJumpToPage.code:'';
			var vid = isJumpToPage?isJumpToPage.vid:'';
			if(!code){
			}else if(code == 'enodeb_patch' && $(".upgradeLists span[logtype=ca]").is(":visible")){
				if(vid){
					var index = $("#fileinfo_ca").datagrid("getRowIndex",vid);
					$("#fileinfo_ca").datagrid("selectRow",index);
				}
			}
			$(this).datagrid("fixRownumber");
			$(this).datagrid("enableContextmenuAutoSize");
		},
		columns:[[
			{field:'id',hidden:true},
			{field:'operation',formatter:moreFileFormatterPatch,styler:setStyle,fixed:true, width:30,title:''},
			{field:'version',formatter:versionFmt, width:250,title:'<%=rb.getString("BanBen")%>'},
			{field:'product', width:150,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'},
			{field:'size', width:100,title:'<%=rb.getString("WenJianDaXiao")%>'},
			{field:'toWho',hidden:isCloudCore=='true'?false:true,formatter:towhoFormat, width:100,title:'<%=rb.getString("BanBenLeiXing")%>'},
			{field:'upload_time', width:140,title:'<%=rb.getString("ShangChuanShiJian")%>'}
		        ]]
		});
		
		$("#fileinfo_fpga").datagrid({
			url:'${ctx}/cell/version/queryfileInfosList.action?file_type=6',
			queryParams:{timeZone:timeZone},
			idField:'id',
			fit:true,
			fitColumns:true,
			nowrap:false,
			border:false,
			striped:true,
			singleSelect:true,
			rownumbers:true,
			pagination:true,
			pagePosition:'bottom',
			pageSize:100,
			pageList:[50,100,200,500],
			toolbar:'#toolbar_fpgaUpgrade',
			onBeforeLoad:function(param){
				param.searchText = $("#fpgaInput").val()
			},
			onLoadSuccess:function(){
				var code = isJumpToPage?isJumpToPage.code:'';
				var vid = isJumpToPage?isJumpToPage.vid:'';
				if(!code){
				}else if(code == 'enodeb_fpga' && $(".upgradeLists span[logtype=fpga]").is(":visible")){
					if(vid){
						var index = $("#fileinfo_fpga").datagrid("getRowIndex",vid);
						$("#fileinfo_fpga").datagrid("selectRow",index);
					}
				}
				$(this).datagrid("fixRownumber");
				$(this).datagrid("enableContextmenuAutoSize");
			},
			columns:[[
				{field:'id',hidden:true},
				{field:'operation',formatter:moreFileFormatterFpga,styler:setStyle,fixed:true, width:30,title:''},
				{field:'version',formatter:versionFmt, width:250,title:'<%=rb.getString("BanBen")%>'},
				{field:'product', width:150,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'},
				{field:'size', width:100,title:'<%=rb.getString("WenJianDaXiao")%>'},
				{field:'toWho',hidden:isCloudCore=='true'?false:true,formatter:towhoFormat, width:100,title:'<%=rb.getString("BanBenLeiXing")%>'},
				{field:'upload_time', width:140,title:'<%=rb.getString("ShangChuanShiJian")%>'}
			        ]]
			})
		
		$("#fileinfo_upgradecpe").datagrid({
			url:'${ctx}/cell/version/queryfileInfosList.action?file_type=9',
			queryParams:{timeZone:timeZone},
			idField:'id',
			fit:true,
			fitColumns:true,
			nowrap:false,
			border:false,
			striped:true,
			singleSelect:true,
			rownumbers:true,
			pagination:true,
			pagePosition:'bottom',
			pageSize:100,
			pageList:[50,100,200,500],
			toolbar:'#toolbar_oducpeUpgrade',
			onBeforeLoad:function(param){
				param.searchText = $("#oducpeInput").val()
			},
			onLoadSuccess:function(){
				var code = isJumpToPage?isJumpToPage.code:'';
				var vid = isJumpToPage?isJumpToPage.vid:'';
				if(!code){
				}else if((code == 'cpe_odu' || code == 'cpe_idu') && $(".upgradeLists span[logtype=upgradecpe]").is(":visible")){
					if(vid){
						var index = $("#fileinfo_upgradecpe").datagrid("getRowIndex",vid);
						$("#fileinfo_upgradecpe").datagrid("selectRow",index);
						isJumpToPage = {};
					}
				}
				$(this).datagrid("fixRownumber");
				$(this).datagrid("enableContextmenuAutoSize");
			},
			columns:[[
				{field:'id',hidden:true},
				{field:'operation',formatter:moreCpeFileFormatter,styler:setStyle,fixed:true, width:30,title:''},
				{field:'file_name', width:250,title:'<%=rb.getString("WenJianMing")%>'},
				{field:'version',formatter:versionFmt, width:250,title:'<%=rb.getString("BanBen")%>'},
				{field:'model_name',formatter:moreCpeOriginalFileFormatter,styler:mmeSetStyle, width:250,title:'<%=rb.getString("ChanPinXingHao")%>'},
				{field:'product',formatter:productFmt, width:150,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'},
				{field:'size', width:100,title:'<%=rb.getString("WenJianDaXiao")%>'},
				{field:'toWho',hidden:isCloudCore=='true'?false:true,formatter:towhoFormat, width:100,title:'<%=rb.getString("BanBenLeiXing")%>'},
				{field:'upload_time', width:140,title:'<%=rb.getString("ShangChuanShiJian")%>'}
			        ]]
		})
		/* CPE中间版本文件列表 */
		$("#fileinfo_cpemidversion").datagrid({
			url:'${ctx}/cell/version/queryMidFileInfos.action',
			queryParams:{timeZone:timeZone},
			idField:'id',
			fit:true,
			fitColumns:true,
			nowrap:false,
			border:false,
			striped:true,
			singleSelect:true,
			rownumbers:true,
			pagination:true,
			pagePosition:'bottom',
			pageSize:100,
			pageList:[50,100,200,500],
			toolbar:'#toolbar_cpemidversion',
			onBeforeLoad:function(param){
				param.version = $("#midVersionInput").val()
			},
			columns:[[
				{field:'id',hidden:true},
				{field:'operation',formatter:moreCpeMidFileFormatter,styler:setStyle,fixed:true, width:30,title:''},
				{field:'version', width:250,title:'<%=rb.getString("BanBen")%>'},
				{field:'modelName', width:100,title:'<%=rb.getString("ChanPinXingHao")%>'},
				{field:'oriVersion',formatter:moreCpeOriginalFileFormatter,styler:mmeSetStyle,width:200,title:'<%=rb.getString("ChuShiBanBen")%>'},
				{field:'fileSize', width:100,title:'<%=rb.getString("WenJianDaXiao")%>'},
				{field:'uploadTime', width:140,title:'<%=rb.getString("ShangChuanShiJian")%>'}
			        ]]
		})
		/* CPE模块版本文件列表 */
		$("#fileinfo_cpemodule").datagrid({
			url:'${ctx}/cell/version/queryModuleFileInfos.action',
			queryParams:{timeZone:timeZone},
			idField:'id',
			fit:true,
			fitColumns:true,
			nowrap:false,
			border:false,
			striped:true,
			singleSelect:true,
			rownumbers:true,
			pagination:true,
			pagePosition:'bottom',
			pageSize:100,
			pageList:[50,100,200,500],
			toolbar:'#toolbar_cpemodule',
			onBeforeLoad:function(param){
				param.version = $("#modelVersionInput").val()
			},
			columns:[[
				{field:'id',hidden:true},
				{field:'operation',formatter:moreCpeMidFileFormatter,styler:setStyle,fixed:true, width:30,title:''},
				{field:'version', width:250,title:'<%=rb.getString("BanBen")%>'},
				{field:'destVersion',formatter:moreCpeOriginalFileFormatter,styler:mmeSetStyle, width:250,title:'<%=rb.getString("MuBiaoBanBen")%>'},
				{field:'moduleName',formatter:moreCpeOriginalFileFormatter,styler:mmeSetStyle, width:200,title:'<%=rb.getString("MoKuaiMingCheng")%>'},
				{field:'fileSize', width:100,title:'<%=rb.getString("WenJianDaXiao")%>'},
				{field:'uploadTime', width:140,title:'<%=rb.getString("ShangChuanShiJian")%>'}
			        ]]
		})
		
		$("#upgradeInput").bind("keyup",function(e){
			if(e.keyCode == 13){
				$("#fileinfo_upgrade").datagrid("reload");
			}
		})
		$("#biosInput").bind("keyup",function(e){
			if(e.keyCode == 13){
				$("#fileinfo_bios").datagrid("reload");
			}
		})
		$("#caInput").bind("keyup",function(e){
			if(e.keyCode == 13){
				$("#fileinfo_ca").datagrid("reload");
			}
		})
		$("#oducpeInput").bind("keyup",function(e){
			if(e.keyCode == 13){
				$("#fileinfo_upgradecpe").datagrid("reload");
			}
		})
		
		$(document).click(function(e){
		 	var e = e || window.event;
	        var elem = e.target || e.srcElement;
	        while(elem){
	            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'showFileOp'|| elem.className == 'showFileUbootOp'
	            		|| elem.className == 'showFilePatchOp'|| elem.className == 'showCpeOduFileOp'|| 
	            		elem.className == 'showCpeIduFileOp' || elem.className == 'linkStyle'){
	                return
	            }
	            elem = elem.parentNode;
	        }
	        //$(".showFileOp").css('display','none');
	        $(".showFileUbootOp").css('display','none');
	        $(".showFilePatchOp").css('display','none');
	        $(".showCpeOduFileOp").css('display','none');
	        //$(".showCpeIduFileOp").css('display','none');
			$('#enbSoftwareFileMenu').hide();
			$('.versionDetails').hide();
		});
	})
	/**
	 * 下拉按钮
	 * @param  
	*/
	function enbSoftwareFileOp(rowId,fileName,type,recommend,el){ // 下拉按钮
		var XiaZai = "<%=rb.getString("XiaZai")%>",
			XinXi = "<%=rb.getString("XinXi")%>",
			XiuGai = "<%=rb.getString("XiuGai")%>",
			ShanChu = "<%=rb.getString("ShanChu")%>";
		var recommendIcon = '';
			
		if(recommend == '1'){//说明此文件是推荐文件
			var TuiJian = '<%=rb.getString("QuXiaoTuiJian")%>';
			recommendStr = '0';
			recommendIcon = 'el-icon el-icon-operation-cancel-recommend';
		}else{
			var TuiJian = '<%=rb.getString("TuiJian")%>';
			recommendStr = '1';
			recommendIcon = 'el-icon el-icon-operation-recommend';
		}
		/* 菜单显示控制 */
		var classes = {
				fileinfo_upgrade: ' CODE_ENB_UPGRADE_IMAGE hidden',
				fileinfo_bios: ' eNbUbootUpdrade hidden',
				fileinfo_ca: ' CODE_ENB_UPGRADE_PATCH hidden',
				fileinfo_fpga: ' CODE_ENB_UPGRADE_FPGA hidden',
				fileinfo_upgradecpe: ' CODE_CPE_UPGRADE_FILE hidden'
				//fileinfo_upgradeiducpe: 'CODE_CPE_UPGRADE_IMAGE hidden'
			},
			cls = classes[type],
			modifyShow = false;
			deleteShow = true;
		
		if( is_super_user == 'true' ){
			modifyShow = true;
		}else{
			deleteShow = false;
		}
				
		var data = [
				{rowId: rowId, type: type, fileName: fileName, text: XinXi, code: 'view', cls: 'el-icon el-icon-operation-info'},
				{rowId: rowId, type: type, fileName: fileName, text: XiaZai, code: 'download', cls: 'el-icon el-icon-operation-download'},
				{rowId: rowId, type: type, fileName: fileName, text: XiuGai, code: 'modify', cls: 'el-icon el-icon-operation-edit '+cls},
				{rowId: rowId, type: type, fileName: fileName, text: ShanChu, code: 'remove', cls: 'el-icon el-icon-operation-delete '+cls},
				{rowId: rowId, type: type, recommend: recommendStr, text: TuiJian, code: 'recommend', cls: recommendIcon +cls}
			];
		$('#enbSoftwareFileMenu').cmenu({data: data, click: enbSoftwareFileOpClick}); 
		/* 菜单位置 */
		var allHeight = $(document).height(),
			isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
			tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0,
			thisTop = $(el).offset().top;
		if((allHeight - thisTop) <200){
			$('#enbSoftwareFileMenu').css({
				"top":thisTop - 179 - tabsHeight,
				"left":70,
			});
			if((allHeight - thisTop) <184) $('.item-child ').css({"top":"-54px",});
		}else{
			$('#enbSoftwareFileMenu').css({
				"top":thisTop - 20 - tabsHeight,
				"left":30,
			});
		}
		$('#enbSoftwareFileMenu').show();
	}
	/**
	 * 点击操作具体项
	 * param row:点击的具体项
	*/
	function enbSoftwareFileOpClick(row){
		var codes = {
				view: viewFile,
				download: fileLibraryDownload,
				modify: editFile,
				remove: delFile,
				recommend : recommendFile
			},
			fileTypes = {
				fileinfo_upgrade: 1,
				fileinfo_bios: 2,
				fileinfo_ca: 3,
				fileinfo_fpga: 6,
				fileinfo_upgradecpe: 4,
				fileinfo_upgradeiducpe: 4
			},
			fileType = fileTypes[row.type],
			fileName = row.fileName;
		    
		if(codes[row.code]) {
			if(row.code == 'download'){
				codes[row.code](fileName,fileType);
			}else if(row.code == 'recommend'){
				codes[row.code](row.rowId,row.recommend,row.type)
			}else{
				codes[row.code](row.rowId,row.type);
			}
		}
		$('#enbSoftwareFileMenu').hide();
	}
	/**
	 * 生成更多操作按钮
	*/
	function moreFileFormatter(value,rowData,rowIndex){
		var row_id = rowData.id;
		var fileName = rowData.file_name;
		var recommend = rowData.recommend;
		if(value == null){
			value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbSoftwareFileOp("+ row_id +",\""+fileName+"\",\"fileinfo_upgrade\",\""+recommend+"\",this)' ></div>";
		}
		return value; 
	}
	/**
	 * select 操作
	*/
	function moreFileFormatterUboot(value,rowData,rowIndex){
		var fileName = rowData.file_name;
		var row_id = rowData.id;
		var recommend = rowData.recommend;
		if(value == null){
			value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbSoftwareFileOp("+ row_id +",\""+fileName+"\",\"fileinfo_bios\",\""+recommend+"\",this)' ></div>";
		}
		return value; 
	}
	/**
	 * select操作
	*/
	function moreFileFormatterPatch(value,rowData,rowIndex){
		var fileName = rowData.file_name;
		var row_id = rowData.id;
		var recommend = rowData.recommend;
		if(value == null){
			value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbSoftwareFileOp("+ row_id +",\""+fileName+"\",\"fileinfo_ca\",\""+recommend+"\",this)' ></div>";
		}
		return value; 
	}
	/**
	 * select操作
	*/
	function moreFileFormatterFpga(value,rowData,rowIndex){
		var fileName = rowData.file_name;
		var row_id = rowData.id;
		var recommend = rowData.recommend;
		if(value == null){
			value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbSoftwareFileOp("+ row_id +",\""+fileName+"\",\"fileinfo_fpga\",\""+recommend+"\",this)' ></div>";
		}
		return value; 
	}
	/**
	 * select操作
	*/
	function moreCpeFileFormatter(value,rowData,rowIndex){
		var row_id = rowData.id;
		var fileName = rowData.file_name;
		var recommend = rowData.recommend;
		if(value == null){
			value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbSoftwareFileOp("+ row_id +",\""+fileName+"\",\"fileinfo_upgradecpe\",\""+recommend+"\",this)' ></div>";
		}
		return value;
	}
	/* function moreCpeIduFileFormatter(value,rowData,rowIndex){
		var row_id = rowData.id;
		var fileName = rowData.file_name;
		var recommend = rowData.recommend;
		if(value == null){
			value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbSoftwareFileOp("+ row_id +",\""+fileName+"\",\"fileinfo_upgradeiducpe\",\""+recommend+"\",this)' ></div>";
		}
		return value; 
	} */
	/**
	 * 版本类型转换
	 * @param value：表格的数据
	*/
	function towhoFormat(value,rowData,rowIndex){
		if(value =='all'){
			return "GA"
		}else if(value == 'beta'){
			return "Beta"
		}else if(value == 'none'){
			return "Test"
		}
	}
	/**
	 * 版本号转化
	 * @param rowData:表格数据
	*/
	function versionFmt(value,rowData,rowIndex){
		if(rowData.recommend == '1'){
			return "<div class='el-badge'><span>"+value+"</span><span class='el-badge__content el-icon el-icon-star-badge'></span></div>"
		}else{
			return value;
		}
	}
	/**
	 *  产品类型标志 转化
	 * @param value:表格数据
	*/
	function productFmt(value,rowData,rowIndex){
		if(value == 'CPE_VERSION' || value == "ODU"){
			return "ODU";
		}else if(value == 'CPE_IDU_VERSION' || value == "IDU"){
			return "IDU";
		}
	} 
	// 没找到用到地方
	function openMoreOperation(idVal,e,tableId,div,showOp){
		var thisTop = $(e).offset().top;
		var allHeight = $(document).height();
		var indexRow = $("#"+tableId).datagrid("getRowIndex",idVal);
		var rowHeight = $("."+div+" .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
		if((allHeight - thisTop) < 240){
			$(e).next("."+showOp).css("bottom",rowHeight+"px");                                                                                                                                                                                                                                                             
		}else{
			$(e).next().css("top",(rowHeight)+"px");
		}
		var current = $('.'+showOp,$(e).parent());
		$('.'+showOp).not(current).hide();
		current.fadeToggle(100);
	}
	//设置操作列单元格样式 
	function setStyle(){
		return 'position:relative';
	}
	function importUpgradeFile(){ // 导入文件
		closeFileUpgradeWindow();
		closeViewFileUpradeWindow();
		$(".upgradeLists > span").each(function(index,ele){
	        if($(this).hasClass("active")){
	            choseType = $(this).attr("logtype");
	        }
	    })
	    var pageType = '';
		pageType = choseType == 'midVersion' || 'moduleVersion' ? choseType : '';
		
		document.getElementById("uploadFile_filelib").value = "";// 置空文件组件
	    $("#uploadFileForm_filelib [name=fileType]").val(choseType);
	    $("#winUpgradeFileInfo .file_info").val("");
		$("#importFileDiv").animate({right:"0px"},400,function(){
			$("#importFileDiv").panel({
				href:'${ctx}/cell/version/loadPage.action?code='+pageType+'&type=add',
				width:900
			})
		});
	}
	/**
	 * 取消推荐函数
	*/
	function recommendFile(idVal,recommend,gridId){
		$.post("${ctx}/cell/version/updateRecommendStatus.action", {id: idVal,recommend:recommend}, function(data) {
            if (data["success"]) {
                 $("#" + gridId).datagrid("reload");
            } else {
                $.messager.alert(TiShi, data["message"]);
            }
        }, "json");
	}
	// 删除已上传的升级文件 
	function delFile(idVal, gridId) {
		closeWindow();
	    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuWenJian")%>", function(r) {
	        if (r) {
	            $.post("${ctx}/cell/version/deleteVersionFile.action", {fileID: idVal}, function(data) {
	                if (data["success"]) {
	                     $("#" + gridId).datagrid("reload");
	                } else {
	                    $.messager.alert(TiShi, data["message"]);
	                }
	            }, "json");
	        }
	    }).addClass("seriousConfirm");
	}
	/* 查看文件 */
	function viewFile(idVal){
		//$(".showFileOp").hide();
		var productNew;
		$(".showCpeOduFileOp").hide();
		$(".showCpeIduFileOp").hide();
		$(".showFileUbootOp").hide();
		$(".showFilePatchOp").hide();
		closeFileUpgradeWindow();
		closeImportFileWindow();
		$(".viewFileUpgradeDiv").animate({right:'0px'},400,function(){
			$(".viewFileUpgradeDiv").panel({
				width:900,
				href:'${ctx}/cell/version/loadPage.action?code=view',
				onLoad:function(){
					$.post("${ctx}/cell/version/getDeviceVersionFileInfo.action", { versionId : idVal }, function(data) {
						if (data) {
							$("#winViewUpgradeFileInfo input[name='fileType']").val(data.file_type);
							$("#winViewUpgradeFileInfo #filePathView").val(data.file_name);
							var pdvDom = $("#winViewUpgradeFileInfo #productView"),
								pdvDatas = pdvDom.combobox('getData');
							if(data.product == "ODU"){
								productNew = "CPE_VERSION";
							}else if(data.product == "IDU"){
								productNew = "CPE_IDU_VERSION";
							}else{
								productNew = data.product;
							}
							pdvDom.combobox('setValue',productNew);
								/* if(pdvDatas){
									pdvDatas.map(function(item){
										if(item.value == data.product) $("#winViewUpgradeFileInfo #deviceView").combobox('setValue',item.device_type);
									});
								} */
							$("#winViewUpgradeFileInfo #versionView").val(data.version);
							$("#toWhoView").combobox("setValue",data.toWho);
							$("#recommendView").combobox("setValue",data.recommend);
							$("#winViewUpgradeFileInfo #descView").val(data.desc);	
							
							var destModel = data.model_name.split(',');
							for(var i=0;i<destModel.length;i++){
								var selDiv;
								selDiv = "<div class='selectedDiv'>" + destModel[i] + "</div>";
								//selDiv = "<div class='selectedDiv'>" + destModel[i] + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
								 
								$(".selectedModuleView").append(selDiv);
							}
							
						}
					}, "json");
				}
			})
		});
	}
	// 修改已上传的升级文件 
	function editFile(idVal, gridId) {
		//加载设备软件版本文件信息
		//$(".showFileOp").hide();
		var productNew;
		$(".showCpeOduFileOp").hide();
		$(".showCpeIduFileOp").hide();
		closeViewFileUpradeWindow();
		closeImportFileWindow();
		$(".editFileUpgradeDiv").animate({right:'0px'},400,function(){
			$(".editFileUpgradeDiv").panel({
				width:900,
				href:'${ctx}/cell/version/loadPage.action?code=edit',
				onLoad:function(){
					$.post("${ctx}/cell/version/getDeviceVersionFileInfo.action", { versionId : idVal }, function(data) {
						if (data) {
							setTimeout(function(){
								$("#winEditUpgradeFileInfo input[name='fileType']").val(data.file_type);
								$("#winEditUpgradeFileInfo #filePathEdit").val(data.file_name);
								var pdtDom = $("#winEditUpgradeFileInfo #productEdit"),
									pdtDatas = pdtDom.combobox('getData');
								if(data.product == "ODU"){
									productNew = "CPE_VERSION";
								}else if(data.product == "IDU"){
									productNew = "CPE_IDU_VERSION";
								}else{
									productNew = data.product;
								}
								pdtDom.combobox('setValue',productNew);
								$("#winEditUpgradeFileInfo #versionEdit").val(data.version);
								$("#toWhoEdit").combobox("setValue",data.toWho);
								$("#recommendEdit").combobox("setValue",data.recommend);
								$("#winEditUpgradeFileInfo #descEdit").val(data.desc);
								
								var destModel = data.model_name.split(',');
								$("#modelNameEdit").combobox("setValues",data.model_name)
								for(var i=0;i<destModel.length;i++){
									var selDiv;
									selDiv = "<div class='selectedDiv'>" + destModel[i] + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
									 
									$(".selectedModuleEdit").append(selDiv);
								}
							},0)
						}
					}, "json");
					$("#submitUploadEdit").unbind("click");
					$("#submitUploadEdit").click(function(){
						var required_err = false;
						var productEditDom = $("#winEditUpgradeFileInfo #productEdit"),
							productEditValue = productEditDom.combobox('getValue');
						if(isCpe == 1){//cpe
							deviceEditValue = ''
						}else{//非cpe
							deviceTypeEditArr.map(function(item,index){
								if(item.value == productEditValue){
									deviceEditValue = item.device_type;
								}
							})
						}
							
						var selectedName = '';
						if ($("#winEditUpgradeFileInfo .selectedDiv").length > 0){
							$("#winEditUpgradeFileInfo .selectedDiv").each(function(){
								selectedName = selectedName + $(this).text() +',';
							})
							selectedName = selectedName ? selectedName.substring(0,selectedName.length-1):selectedName;
						}else {
							$(".errorText_modelName").css("visibility","visible");
						}	
							
							
						var modelNameEdit = $("#modelNameEdit").combobox("getValue");
						$("#winEditUpgradeFileInfo #productEdit").val(productEditValue);
							if(checkVersion()){
								if($("#productEdit").combobox('getValue') == ""){
									$("#selectType").css("visibility","visible");
									return false;
								}else{
									$("#selectType").css("visibility","hidden");
								}
								//修改设备软件版本文件信息
								var pram={ 
										versionId : idVal,
										fileType:$("#winEditUpgradeFileInfo input[name='fileType']").val(),
										fileName:$("#winEditUpgradeFileInfo #filePathEdit").val(),
										product:productEditValue,
										deviceType:deviceEditValue,
										version:$("#winEditUpgradeFileInfo #versionEdit").val(), 
										model_name:selectedName,
										toWho:$("#toWhoEdit").combobox("getValue"),
										recommend:$("#recommendEdit").combobox("getValue"),
										desc:$("#winEditUpgradeFileInfo #descEdit").val()
								};
							    $.post("${ctx}/cell/version/goModifyDeviceVersionFileInfo.action", pram, function(data) {
							        if (data["success"]) {
										showMsg("success_msg",'<%=rb.getString("ChengGong")%>')
										$('.editFileUpgradeDiv').animate({right:'-1000px'},400,function(){
											$('.editFileUpgradeDiv').html("");
											$("#" + gridId).datagrid("reload");
										});
							        }
							    }, "json");
							}
					});
				}
			})
		});
	}
	/**
	 * 下载
	 * @param fileName:传入的name数据
	 * @param fileType:type类型
	*/
	function fileLibraryDownload(fileName, fileType){
	    $.post("${ctx}/omc/version/file/fileIsExist.action", { fileName : fileName, fileType : fileType}, function (data) {
	        if(data["success"]){
	        	/* $("#downloadFileForm_filelib").form('submit', {
	                url: "${ctx}/omc/version/file/downLoadFile.action",
	                onSubmit: function(param){
	                    param.fileName = fileName;
	                    param.fileType = fileType;
	                }
	            }); */
	            exportByForm("${ctx}/omc/version/file/downLoadFile.action",{
	            	fileName: fileName,
	            	fileType: fileType
	            });
	        }else{
	        	$.messager.alert(TiShi, data["message"]);
	        }
	    },"json");
	}

	// 关闭 修改文件模块
	function closeFileUpgradeWindow(){
		$(".editFileUpgradeDiv").animate({right:'-1000px'},400,function(){
			$('.editFileUpgradeDiv').html("");
		});
	}
	// 关闭查看文件模块
	function closeViewFileUpradeWindow(){
		$(".viewFileUpgradeDiv").animate({right:'-1000px'},400,function(){
			$('.viewFileUpgradeDiv').html("");
		});
	}
	// 关闭导入文件模块
	function closeImportFileWindow(){
		$("#importFileDiv").animate({right:"-1000px"},400,function(){
			$("#importFileDiv").html("");
		});
		$(".editFileUpgradeDiv").animate({right:"-1000px"},400,function(){
			$(".editFileUpgradeDiv").html("");
		});
	}
	// 关闭所有模块
	function closeWindow(){
		closeFileUpgradeWindow();
		closeViewFileUpradeWindow();
		closeImportFileWindow();
	}
	/**
	 * 监听样式添加
	 * @param ele:监听到的数据类型 进行判断添加
	*/
	function changeClass(ele){
		$("#upgradeImportDiv").attr("class","circleIcon hidden placeholder-bt");
		var upgradeType = $(ele).attr("logtype");
		if(upgradeType == "upgrade"){
			$("#upgradeImportDiv").addClass("CODE_ENB_UPGRADE_IMAGE");
		}else if(upgradeType == "bios"){
			$("#upgradeImportDiv").addClass("eNbUbootUpdrade");
		}else if(upgradeType == "ca"){
			$("#upgradeImportDiv").addClass("CODE_ENB_UPGRADE_PATCH");
		}else if(upgradeType == "fpga"){
			$("#upgradeImportDiv").addClass("CODE_ENB_UPGRADE_PATCH");
		}else if(upgradeType == "upgradecpe"){
			$("#upgradeImportDiv").addClass("CODE_CPE_UPGRADE_IMAGE");
		}else if(upgradeType == "upgradeiducpe"){
			$("#upgradeImportDiv").addClass("CODE_CPE_UPGRADE_IMAGE");
		}
		$('#mainpage').click();
	}
	/* 不确定代码 */
	// 任务执行结果格式化
	function taskResultFmt(value, rowData, rowIndex) {
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

	/**
	 * 任务进度格式化
	 * @param value:传入的转换数据
	*/
	function taskProgressFmt(value, rowData, rowIndex) {
		if (value == "0") {
			return "<%=rb.getString("WeiKaiShi")%>";
		} else if (value == "1") {
			return "<%=rb.getString("JinXingZhong")%>";
		} else if (value == "2") {
			return "<%=rb.getString("YiJieShu")%>";
		}
	}
	
	// 显示md5值
	function showMd5Val(okHandler, cancelHandler, md5) {
	    var url = '${ctx}/cell/version/loadPage.action?code=md5';;
		openDefaultWindow(url,{
			title: '<%=rb.getString("WenJianXinXi")%>',
			width:410,height:200,
			onLoad: function(){
				$("#cancelBtn_md5").unbind("click");
				$("#okBtn_md5").unbind("click");
				$("#cancelBtn_md5").bind("click", cancelHandler);
				$("#okBtn_md5").bind("click", okHandler);
			    $("#MD5Value").html(md5);
			}
		}); 
	}
	// 客户校验MD5值之后，确认上传文档
	function saveInputFile() {
		var params = {};
		params = {
			fileName_temp : retMap["fileName_temp"],
			fileSize : retMap["fileSize"],
			fileType : retMap["fileType"],
			product : retMap["product"],
			version : retMap["version"],
			desc : retMap["desc"],
			MD5 : retMap["MD5"],
			destination : retMap["destination"]
		};

		var grid;

		var fileType = $("#uploaFileForm_filelib [name=fileType]").val();

		if (fileType == "upgrade") {
			grid = $("#fileinfo_upgrade");
		} else if (fileType == "bios") {
			grid = $("#fileinfo_bios");
		} else if (fileType == "ca") {
			grid = $("#fileinfo_ca");
		} else if (fileType == "upgradecpe") {
			grid = $("#fileinfo_upgradecpe");
		} else if (fileType == "upgradeiducpe") {
			grid = $("#fileinfo_upgradeiducpe");
		}

		$.post("${ctx}/cell/version/saveInputFileForSure.action", params,
				function(data) {
					if (data["success"]) {
						/* $("#inputFileMd5").window("close"); */
						closeDefaultWindow();
						$.messager.alert(TiShi, data["message"]);
						grid.datagrid("reload");
					}
				}, "json");
	}

	// 客户校验MD5值之后，认为不需要上传该文件
	function cancelInputFile() {
		var param = {};
		param = {
			destination : retMap["destination"]
		};
		$.post("${ctx}/cell/version/cancelInputFileForSure.action", param,
				function(data) {
					if (data["success"]) {
						/* $("#inputFileMd5").window("close"); */
						closeDefaultWindow();
					}
				}, "json");
	}

	function mmeSetStyle(value,row,index){
		return 'position:relative'; 
	}
	
	//中间版本文件列表 -- 初始版本显示格式化 
	function moreCpeOriginalFileFormatter(value,rowData,rowIndex){
		var versionList;
		if ( value ){
			versionList = value.split(",")
		}else {
			return ''
		}
		
		if (versionList.length == 1){
			value = versionList[0]
		}else{
			value = versionList[0] + '...' + '<span class="linkStyle" onclick="showDetailVersion(this)">[' +versionList.length+']</span>'
				+ "<div class='versionDetails'> "+value + "</div>";
		}
		
		return value; 
	}
	
	function showDetailVersion(ele){
		var thisTop = $(ele).offset().top;
		var allHeight = $(document).height();
		var thisLeft = $(ele).offset().left;
		var allWidth = $(document).width();
		$(".versionDetails").hide();
		if((allHeight - thisTop) < 200){
			$(ele).siblings(".versionDetails").css("top","-55px");
		}
		if((allWidth - thisLeft) < 200){
			$(ele).siblings(".versionDetails").css("left","-90px");
		}
			$(ele).siblings(".versionDetails").fadeToggle();
	}
	
	function moreCpeMidFileFormatter(value,rowData,rowIndex){
		var version = rowData.version;
		
		if(value == null){
			value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='cpeVersionOp(\""+version+"\",\""+rowData.modelName+"\",this)' ></div>";
		}
		return value;
	}
	
	function cpeVersionOp(version,model,el){
		var XiaZai = "<%=rb.getString("XiaZai")%>",
			XinXi = "<%=rb.getString("XinXi")%>",
			XiuGai = "<%=rb.getString("XiuGai")%>",
			ShanChu = "<%=rb.getString("ShanChu")%>";
				
		var data = [
				{ version: version,modelName:model, text: XinXi, code: 'view', cls: 'el-icon el-icon-operation-info'},
				{ version: version,modelName:model, text: XiuGai, code: 'modify', cls: 'el-icon el-icon-operation-edit '},
				{ version: version,modelName:model, text: ShanChu, code: 'remove', cls: 'el-icon el-icon-operation-delete '},
			];
		$('#enbSoftwareFileMenu').cmenu({data: data, click: cpeVersionFileOpClick}); 
		/* 菜单位置 */
		var allHeight = $(document).height(),
			isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
			tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0,
			thisTop = $(el).offset().top;
		if((allHeight - thisTop) <200){
			$('#enbSoftwareFileMenu').css({
				"top":thisTop - 179 - tabsHeight,
				"left":70,
			});
			if((allHeight - thisTop) <184) $('.item-child ').css({"top":"-54px",});
		}else{
			$('#enbSoftwareFileMenu').css({
				"top":thisTop - 20 - tabsHeight,
				"left":30,
			});
		}
		$('#enbSoftwareFileMenu').show();
	}
	function cpeVersionFileOpClick(row){
		var codes = {
				view: viewVersionFile,
				modify: editVersionFile,
				remove: delVersionFile,
			};
		    
			codes[row.code](row.version,row.modelName);
		
		$('#enbSoftwareFileMenu').hide();
	}
	
	<%-- 修改已上传的升级文件 --%>
	function viewVersionFile(idVal,modelName) {
		//加载设备软件版本文件信息
		$(".showCpeOduFileOp").hide();
		$(".showCpeIduFileOp").hide();
		closeViewFileUpradeWindow();
		closeImportFileWindow();
		var upgradeType = $('#omcLogTabsDiv span.active').attr('logType');
		
		if(upgradeType == "midVersion"){
			editMidVersionFile(idVal,'view',modelName);
		}else if(upgradeType == "moduleVersion"){
			editModuleFile(idVal,'view');
		}
	}
	
	<%-- 修改已上传的升级文件 --%>
	function editVersionFile(idVal,modelName) {
		//加载设备软件版本文件信息
		$(".showCpeOduFileOp").hide();
		$(".showCpeIduFileOp").hide();
		closeViewFileUpradeWindow();
		closeImportFileWindow();
		var upgradeType = $('#omcLogTabsDiv span.active').attr('logType');
		
		if(upgradeType == "midVersion"){
			editMidVersionFile(idVal,'edit',modelName);
		}else if(upgradeType == "moduleVersion"){
			editModuleFile(idVal,'edit');
		}
	}
	
	function editMidVersionFile(idVal,type,modelName) {
		//加载设备软件版本文件信息
		var url = '${ctx}/cell/version/loadPage.action?code=midVersion&type='+type,
				gridId = 'fileinfo_cpemidversion';
		
		$(".editFileUpgradeDiv").animate({right:'0px'},400,function(){
			$(".editFileUpgradeDiv").panel({
				width:900,
				href:url,
				onLoad:function(){
					var params = {
							midVersion:idVal,
							modelName:modelName?modelName :""
					}
					
					$.post("${ctx}/cell/version/queryMidFileInfo.action", params, function(data) {
						if (data) {
							$("#winMidUpgradeFileInfo #midFilePath").val(data.fileName);
							$("#winMidUpgradeFileInfo #midVersion").val(data.version);
							$("#winMidUpgradeFileInfo #midDesc").val(data.description);	
							$("#winMidUpgradeFileInfo #modelName").combobox("setValue",data.modelName);
							
							var destVer = data.oriVersion.split(',') ,selectedStr='', selectedVer ={} ,selectArr =[];
							for(var i=0;i<destVer.length;i++){
								selectedStr = "{" + "'software_version':'" +destVer[i]+"'}";
								selectedStr = eval( '(' + selectedStr +')');
								selectArr.push(selectedStr)
							}
							
							selectedVer ={
									"total":selectArr.length,
									"rows":selectArr
								};
							
							$('#oriVersionList_datagrid').pairgrid('loadRight',selectArr);
							
							//initInputs("#winMidUpgradeFileInfo");
							
						}
					}, "json");
					
				}
			})
		});
	}
	
	
	function editModuleFile(idVal,type) {
		var url = '${ctx}/cell/version/loadPage.action?code=moduleVersion&type='+type,
			gridId = 'fileinfo_cpemodule';
		
		$(".editFileUpgradeDiv").animate({right:'0px'},400,function(){
			$(".editFileUpgradeDiv").panel({
				width:900,
				href:url,
				onLoad:function(){
					
					$.post("${ctx}/cell/version/queryModuleFileInfo.action", { moduleVersion : idVal }, function(data) {
						if (data) {
							$("#winModuleUpgradeFileInfo #moduleFilePath").val(data.fileName);
							$("#winModuleUpgradeFileInfo #moduleVersion").val(data.version);
							$("#winModuleUpgradeFileInfo #moduleDesc").val(data.description);	
							
							var destModel = data.moduleName.split(',');
							for(var i=0;i<destModel.length;i++){
								var selDiv;
								if (type == 'view'){
									selDiv = "<div class='selectedDiv'>" + destModel[i] + "</div>";
								}else if (type == 'edit'){
									$("#moduleName").combobox("setValues",data.moduleName);
									selDiv = "<div class='selectedDiv'>" + destModel[i] + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
								}
								 
								$(".selectedModule").append(selDiv);
							}
							
							var destVer = data.destVersion.split(',') ,selectedStr='', selectedVer ={} ,selectArr =[];
							for(var i=0;i<destVer.length;i++){
								selectedStr = "{" + "'version':'" +destVer[i]+"'}";
								selectedStr = eval( '(' + selectedStr +')');
								selectArr.push(selectedStr)
							}
							
							selectedVer ={
									"total":selectArr.length,
									"rows":selectArr
								};
							
							$('#destVersionList_datagrid').pairgrid('loadRight',selectArr)
							
						}
					}, "json");
					
				}
			})
		});
	}
	
	function delVersionFile(version,model) {
		closeWindow();
		var url='',params,gridId;
		var upgradeType = $('#omcLogTabsDiv span.active').attr('logType');
		
		if(upgradeType == "midVersion"){
			url = "${ctx}/cell/version/delMidVersionInfo.action";
			params={
					midVersion: version,
					modelName : model?model:""
			};
			gridId = 'fileinfo_cpemidversion';
			
		}else if(upgradeType == "moduleVersion"){
			url = "${ctx}/cell/version/delModuleVersionInfo.action";
			params={
					moduleVersion: version
			};
			gridId = 'fileinfo_cpemodule';
		}
		
	    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuWenJian")%>", function(r) {
	        if (r) {
	            $.post(url, params, function(data) {
	                if (data["success"]) {
	                     $("#" + gridId).datagrid("reload");
	                } else {
	                    $.messager.alert(TiShi, data["message"]);
	                }
	            }, "json");
	        }
	    }).addClass("seriousConfirm");
	}
</script>