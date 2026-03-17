<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style type="text/css">
	#inputMML_gnb {
		width: 99%;
		height: 63px;
		resize: none;
		color: #949494;
	}
	.cellParamWestNorth .layout-panel-north .panel-header {
		border-width: 0 1px 1px 1px;
	}
	.select_reset .combo input{
		width:98px !important;
	}
	.select_reset .combo{
		width:102px !important;
	}

	#cellParam_gnb #paramNodesUl label{
		margin-top:5px;
		word-break:break-all;
	}
	.eNBHighQuery-gnb{
		height:auto;
		position:absolute;
		top:27px;
		left:0px;
		box-shadow:0 5px 15px #c9c9c9;
		background:white;
		display:none;
	}
	.iconHighQuery{
		position:relative;
	}
	.operationAdd{
		margin-left:10px;
	}
	.operationSub{
		margin-left:14px;
	}
	.operationAdd img , .operationSub img{
		vertical-align:text-bottom;
		cursor:pointer;
	}
	.gnbElfcellConfigParamPanel .gnbMmlItemDiv{
		height:46px;
	}
	.gnbElfcellConfigParamPanel .gnbMmlItemDiv .prompt{
		height:20px;
		line-height:20px;
		color:red;
	}
	/* 参数面板 - flex布局：操作区可滚动，按钮固定底部 */
	.gnbElfcellConfigParamPanel {
		/* 保持 tabsContentDiv > div 的绝对定位，由 turnTabs 控制 display */
	}
	.gnb-panel-flex-wrapper {
		display: flex;
		flex-direction: column;
		position: absolute;
		top: 0;
		bottom: 0;
		left: 0;
		right: 0;
	}
	.gnb-panel-flex-wrapper #elfcellshowParamValues_gnb {
		flex: 1;
		overflow: auto;
		/* 覆盖 panelTableDiv 的 position:absolute，使其参与 flex 布局 */
		position: relative;
		top: auto;
		left: auto;
		bottom: auto;
		right: auto;
		padding: 20px;
	}
	.gnb-panel-bottom-bar {
		flex-shrink: 0;
		padding: 8px 20px;
		border-top: 1px solid #EDEDED;
		background: #fff;
		text-align: left;
	}
	.item{
		width:330px;
	}

	.enb-list-ctn-gnb {
		position:absolute;
		width: 100%;
		height: 100%;
		z-index: 100;
		top: 0;
		background-color: #fff;
		display: flex;
		flex-direction: column;
	}
	.list-title {
		font-size: 16px;
		color: #5A7B92;
		padding: 10px 0 10px 25px;
	}
	.list-body {
		width: calc(100% - 45px);
		padding-left: 25px;
		overflow: auto;
	}
	.list-body-title {
		background-color: #FBFBFB;
		color:#5A7B92;
		font-size: 12px;
		font-weight: bold;
		padding: 5px;
		border-bottom: 1px dashed silver;
	}
	.list-item-info {
		position: relative;
		padding: 6px 10px;
		border-bottom: 1px dashed silver;
	}
	.list-item-op {
		padding: 5px;
		display: inline-block;
		position: absolute;
		top: 0px;
		right: 5px;
		color: red;
		cursor: pointer
	}
	.paramValuesTips {
		position: absolute;
		/* height: calc(100% - 40px); */
		width: 100%;
		display: none;
		text-align: center;
	}
	.param-step {
		display: flex;
		flex-direction: column;
		flex: auto;
	}
	.param-step > div {
		padding: 2px 0;
		color: #6b6b6b;
	}
	.step-title {
		color: #5AB8FA;
		height: 30px;
		position: relative;
	}
	.step-title::after {
		content: '';
		display: inline-block;
		position: absolute;
		width: 50%;
		height: 2px;
		top: calc(50% - 2px);
		right: 0;
		background-color: #4D84FF;
	} 
	.step-title::before {
		content: '';
		display: inline-block;
		position: absolute;
		width: 50%;
		height: 2px;
		top: calc(50% - 2px);
		left: 0;
		background-color: #4D84FF;
	}
	.step-title.first::before, .step-title.last::after {
		width: 0;
	}
	.step-count {
		position: absolute;
		top: calc(50% - 12px);
		left: calc(50% - 10px);
		display: inline-block;
		width: 20px;
		height: 20px;
		line-height: 20px;
		border-radius: 20px;
		border: 2px solid #4D84FF;
		background: #fff;
		z-index: 1;
	}
	.param-values-oper {
		margin: 6px;
	}

	#showParamValues_gnb:empty + .paramValuesTips{
		display: flex;
		top: 40%;
	}

	.flex-row-item-gnb {
		display: flex;
		flex: 1 1 100%;
	}
	.flex-prev-item-gnb {
		flex: 1 1 37%;
	}
	.flex-suff-item-gnb {
		flex: 1 1 62%;
	}
	.horizontal-line-gnb {
		cursor: row-resize;
		padding: 10px;
	}
	.vertical-line-gnb {
		cursor: col-resize;
		padding: 10px;
	}
	.searchInputStyle{
		border-bottom:none !important;
	}
	.tabsTitle{
		border:none;
	}
	.taskListToorbarStyle{
		top:46px;
	}
	.splitPanel{
		box-shadow:none;
		border:1px solid #DEE2EC;
	}
	.singleTitle{
		color:#333;
		background:none;
		padding-left:20px;
		padding-top:5px;
	}
	.tabsTitle{
		color:#333;
	}
	#modeRebootTips {
		margin-left:200px;
		color:blue;
	}
	#toolbar_gridCell_cellParam_gnb .textbox.combo{
		height:32px !important;
	}
	#toolbar_gridCell_cellParam_gnb .textbox-icon.combo-arrow{
		height:33px !important;
	}
	#toolbar_gridCell_cellParam_gnb .textbox .textbox-text{
		padding-top:8px !important;
		padding-bottom:8px !important;
	}
	#cellParam_gnb .tabsContentDiv , #cellParam_gnb .contentDiv {
		border:none;
	}
	.el-icon-common-download:before{
		color:#4D84FF;
	}
	@keyframes shinning {
		0%{box-shadow: 0 0 5px 0 rgba(0,0,255,0.3);opacity:0.9}
		25%{box-shadow: 0 0 15px 0 rgba(0,0,255,0.6);opacity:1}
		50%{box-shadow: 0 0 5px 0 rgba(0,0,255,0.3);opacity:0.9}
		75%{box-shadow: 0 0 15px 0 rgba(0,0,255,0.6);opacity:1}
		100%{box-shadow: 0 0 5px 0 rgba(0,0,255,0.3);opacity:0.9}
	}

	#commonQueryText_gnb {
		width: 200px;
	}

	.editButton {
		padding: 0 10px;
		height: 22px;
		background: #F2F9FF;
		border-radius: 2px;
		line-height: 22px;
		cursor: pointer;
		margin-left: 10px;
		border: 1px solid #1DA3FC;
		display: inline-block;
		float: right;
	}
	.explanation-ctn .explanation-tips {
		position: absolute;
		left: 20px;
		padding: 5px;
		display: none;
		z-index: 1000;
		background: #fff;
		min-width: 180px;
		border: 1px solid #5AB8FA;
		border-radius: 3px;
	}
	.explanation-ctn:hover .explanation-tips {
		display: inline-block;
	}
	.tipText{
		color:#CFCFCF;
		padding-bottom:26px;
		padding-top:12px;
		display:flex;
	}
	.tipText .infoTip{		
		padding-top:2px;
	}
	.infoTip:before{
		color:#CFCFCF;
		font-size:14px;
		margin-right:6px;
	}
	.double1-item select , .double2-item select{
		width:100px;
	}
	.paramNodesUl li.double1-item{
		width:300px;
		min-width:auto; 
		display:inline-block;
	}
	.paramNodesUl li.double2-item{
		width:100px;
		min-width:auto;
		display:inline-block;
		margin-left:35px;
		position:relative;
	}
	.paramNodesUl li.double1-item label {
		visibility:hidden;
	}
	.paramNodesUl li.double2-item label {
		position:absolute;
		left:-335px;
	}
	li.double2-item:before{
		content:'X';
		font-size:14px;
		position:absolute;
		top:4px;
		left:-20px;
	}
	.overflow-auto {
		overflow: auto;
	}
	.helpDiv {
		display: flex;
		flex-direction: column;
		height: 100%;
	}
	.help-summary {
		height: fit-content;
		padding: 0 20px;
	}
	.help-summary .queryGroup input {
		width: inherit;
	}
	.help-content {
		flex: 1 1 100%;
		overflow: auto;
	}
	.param-title {
		width: 100%;
		color: rgba(0,0,0,0.8);
		font-weight: bold;
		margin-bottom: 10px;
	}
	.help-item {
		border: 1px solid #DEE2EC;
		border-radius: 4px;
		padding: 10px;
		display: flex;
		flex-wrap: wrap;
		margin-bottom:8px;
	}
	.help-item .el-form-item {
		margin-bottom: 6px;
		width: 23%;
		overflow: auto;
	}
	.help-item .el-form-item__content {
		min-height:28px;
	}
	.help-item .el-form-item .el-form-item__label {
		margin-bottom: 0;
		color: #7a7992;
	}
	.help-item-title {
		color: #7a7992;
		display: inline-block;
		width: 100px;
		margin-bottom: 10px;
	}

	#operGroupTree_gnb .tree-node {
		position: relative;
	}
	.hide-item-cls {
		display: none !important;
	}

	/* 自定义MML节点操作按钮组 - 默认隐藏 */
	.cus-btn-group {
		display: none;
	}
	/* 鼠标悬停或选中时显示操作按钮组 */
	.tree-node-selected .cus-btn-group,
	.tree-node-hover .cus-btn-group {
		display: inline-block;
	}
	/* 操作按钮悬停效果 */
	.cus-btn-icon:hover {
		color: #409EFF !important;
	}

	.cus-bt-cls {
		display: none;
		font-size: 12px;
	}
	.tree-node-selected .cus-bt-cls,
	.tree-node-hover .cus-bt-cls {
		display: inline-block;
	}
</style>

<script type="text/javascript">
	var ctx = "${ctx}";
	var isEnglish = <%=(rb.getLocale().equals(Locale.ENGLISH))%>;
	var QingXuanZeSheBei = "<%=rb.getString("QingXuanZeSheBei")%>";
	var TiShi = "<%=rb.getString("TiShi")%>";
	var CiPeiZhiChongQiHouShengXiao = "<%=rb.getString("CiPeiZhiChongQiHouShengXiao")%>";
	var QingShuRuJiaoBen = "<%=rb.getString("QingShuRuJiaoBen")%>";
</script>
<div class="panelDefault" id="cellParam_gnb" style="overflow:hidden" >
	<div class="tabsTitle" id="eNb_tabs_area">
		<span tabtit="MMLConfig_gnb" onclick="turnTabs(this)" class="active"><%=rb.getString("CanShuPeiZhi")%></span>	
		<span id="MMLScript_gnbTab" tabtit="MMLScript_gnb" onclick="turnTabs(this)"><%=rb.getString("MMLJiaoBen")%></span>			
	</div>
	<!-- <div style="height: 5px;background: #E3E7F0;"></div> -->
	<div class="omcTabsTablePage" style="left:0px;right:0px;top: 50px;bottom:0px;background:#fff;">
		<div class="MMLConfig_gnb" id="MML_Config_gnb" style="display: block;">
			<div style="width: 100%;height: 100%;min-height: 600px;min-width:1000px;" class="flex-ctn">
				<div class="flex-row-item-gnb" style="height:47%;">
					<div class="splitPanel cellParamWestNorth flex-prev-item-gnb" style="height:100%;width:37%;float:left;margin-left:20px;">
						<div class="singleTitle">
							gNB
							<p style='position:absolute;right:15px;top:7px;display:none;' class='showSelectNum'><%=rb.getString("YiXuan") %>(
								<span class="circle-num-tips" style='margin:0 3px;color:#4D84FF'></span>
							)
								<span class='el-icon el-icon-circle-down' style='font-size:16px;cursor:pointer;border-bottom:none;margin-left:5px;' onclick="slidedownlistGnb()"></span>
							</p>
							<div id="gnbMMLBatchInput" class="editButton" onclick="showBatchDL_gnb()" style="margin-right: 15px;">
								<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
								<span><%=rb.getString("PiLiangShuRu")%></span>
							</div>
						</div>
					    <div class="contentDiv">
							<table id="gridCell_cellParam_gnb"></table>
						</div>
						<div class="enb-list-ctn-gnb">
							<div class="list-title">
								<%=rb.getString("YiXuanJiZhan")%>
								<span style="float: right;padding-right: 15px;" onclick="slideuplistGnb()"><i class="el-icon el-icon-close"></i></span>
							</div>
							<div style="width: calc(100% - 44px);padding-left: 25px;">
								<div class="list-body-title">
									<%=rb.getString("HostName")%> + <%=rb.getString("XiaoZhanBianMa")%>
									<div class="operationDiv operation_delete" title="Delete All" style="margin-left:5px;float: right;" onclick="delAllDoStgRecordGnb()"></div>
								</div>
							</div>
							<div class="list-body"></div>
						</div>
					</div>
					<div class="vertical-line-gnb"></div>
					<div class="splitPanel flex-suff-item-gnb"  style="height:100%;width:62%;float:right;margin-right:20px;">
						<div class="singleTitle">
							{{resultTitle}}
							<div style="float: right;display:flex;margin-right:20px;">
								<el-radio-group v-model="resultType" size="small" @input="changeType">
									<el-radio-button label="res" ><%=rb.getString("PeiZhiJieGuo")%></el-radio-button><el-radio-button label="help"><%=rb.getString("BangZhu")%></el-radio-button>
								</el-radio-group>
								<div v-if="resultType == 'res'" class="param-values-oper">
									<a onclick="menuHandlerGnb({name:'clear'})" class="el-icon el-icon-operation-clear" title="<%=rb.getString("QingKong")%>" style="font-size:20px;margin-right:5px;cursor: pointer;"> </a>
									<a onclick="menuHandlerGnb({name:'download'})" class="el-icon el-icon-common-download" title="<%=rb.getString("BaoCun")%>" style="font-size:20px;margin:0px 5px;cursor: pointer;"> </a>
								</div>
								<div v-if="resultType == 'help'" class="param-values-oper">
									<a @click="exportEnbMml" class="el-icon el-icon-operation-export" title="<%=rb.getString("DaoChu")%>" style="font-size:20px;margin-right:5px;cursor: pointer;"> </a>
								</div>
								
							</div>
							
						</div>
						
						<div :class="resultType == 'help' ? 'contentDiv' : 'contentDiv overflow-auto'" style="right:20px;top:40px;">
							<div v-if="resultType == 'help'" class="helpDiv">
								<div class="help-summary">
									<div style="font-weight: bold;margin-bottom:10px;">{{helpData.param_name}}</div>

									<div>
										<span class="help-item-title"><%=rb.getString("CanShuLieBiaos")%></span>
										<div class="queryGroup" style="margin:-10px 10px 0;float:right;">
											<el-input v-model="paramData.searchText" @keyup.enter.native="queryEnbMML" placeholder='<%=rb.getString("BianMa")%> / <%=rb.getString("MingCheng")%>' style="width:260px;"></el-input>
											<i @click='queryEnbMML' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
										</div>
									</div>
								</div>
								<div class="help-content">
									<el-form label-position="top" style="padding:0 20px 20px;">
									<div class="help-item" v-for="item in helpData.groupList" >
										<span class="param-title">{{item.paramName}}</span>
										
										<el-form-item label='<%=rb.getString("ShuJuLeiXing")%>'>{{item.dataModel}}</el-form-item>
										<el-form-item label='<%=rb.getString("FanWei")%>'>{{item.dataRange}}</el-form-item>
										<el-form-item label='<%=rb.getString("MoRenZhi")%>'>{{item.dftValue}}</el-form-item>
										
										<el-form-item label='<%=rb.getString("ShiFouXuYaoChongQi")%>'>
											<template v-if="item.dynamic == '1'"><%=rb.getString("Shi")%></template>
											<template v-else> <%=rb.getString("Fou")%></template>
										</el-form-item>
										<el-form-item label='<%=rb.getString("ShuoMing")%>' style="width:100%;">{{item.explanation}}</el-form-item>
										<el-form-item label="MIB_DN" style="width:100%;">{{item.mib_dn}}</el-form-item>
										<el-form-item label="TR PATH" style="width:100%;">
											{{item.name_path}}
											<span class="el-icon el-icon-operation-copy" @click="copyPath(item.name_path)"></span>
										</el-form-item>
									</div>
								</el-form>
								</div>
							</div>
							
							<div v-show="resultType == 'res'" class="contentDiv" id="showParamValues_gnb" style="padding-top:20px;overflow:auto;right:20px;"></div>
							<div v-if="resultType == 'res' && isResult == false" class="paramValuesTips">
								<div class="param-step">
									<div class="step-title first"><span class="step-count">1</span></div>
									<div><%=rb.getString("DiYiBu")%></div>
									<div><%=rb.getString("XuanZheJiZhanSheBei")%></div>
								</div>
								<div class="param-step">
									<div class="step-title"><span class="step-count">2</span></div>
									<div><%=rb.getString("DiErBu")%></div>
									<div><%=rb.getString("XuanZheMingLin")%></div>
								</div>
								<div class="param-step">
									<div class="step-title"><span class="step-count">3</span></div>
									<div><%=rb.getString("DiSanBu")%></div>
									<div><%=rb.getString("PeiZhiJuTiCanShu")%></div>
								</div>
								<div class="param-step">
									<div class="step-title last"><span class="step-count">4</span></div>
									<div><%=rb.getString("DiSiBu")%></div>
									<div><%=rb.getString("ChaKanJieGuo")%></div>
								</div>
							</div>	
						</div>
					</div>
				</div>
				<div class="horizontal-line-gnb"> </div>
				<div class="flex-row-item-gnb" style="height:50%;margin-bottom:20px;">
					<div class="splitPanel flex-prev-item-gnb" id="mmlListCenter_gnb" style="height:100%;width:37%;float:left;margin-left:20px;">
						<div class="singleTitle">
							<%=rb.getString("MingLingLieBiao")%>
						</div>
						<div class="queryGroup" style="margin-left:15px;">
							<input id="groupQueryText_gnb" name="value" style="margin-left:0px;width:275px;" placeholder="<%=rb.getString("BianMa")%> / <%=rb.getString("MingCheng")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)" value="">
							<b class='el-icon el-icon-common-search' onclick="javascript: filterMMLTreeGnb();"></b>
						</div>
					    <div class="contentDiv" style="padding-top:20px; top: 70px;">
							<%-- MML 命令集-树形结构 --%>
							<ul id="operGroupTree_gnb" class="easyui-tree" data-options="border:false" style="height:calc(100% - 20px);overflow:auto;"></ul>
						</div>
					</div>
					<div class="vertical-line-gnb"></div>
					<div class="splitPanel flex-suff-item-gnb" id="operatorActor_gnb"  style="height:100%;width:62%;float:right;margin-right:20px;">
						<div class="tabsTitle omcLogLists" style='background:none;height:40px;line-height:40px;border-bottom:1px solid #EDEDED;'>				
							<span tabtit="gnbCellParamDetail" onclick="turnTabs(this)" class="active" style='margin-left:20px;'><%=rb.getString("JiaoBenYanShi")%></span>
							<span tabtit="gnbElfcellConfigParamPanel" onclick="turnTabs(this)"><%=rb.getString("CanShuMianBan")%></span>
						</div>
						<div class="tabsContentDiv" style='top:42px;'>
						    <div class="contentDiv gnbCellParamDetail" id="showParamValues_gnb" style="padding-top:20px;margin-left:20px;">
								<div>
									<div style="margin-bottom: 20px;">
										<textarea id="inputMML_gnb" name="inputMML" class="border border-box"  style="padding-left:10px;border-style: solid;" placeholder="<%=rb.getString("QingShuRuJiaoBen")%>"></textarea>
									</div>
									<div>
										<select id="textOperName_gnb" style="float:left;height:26px;"></select>
										<a class="linkbutton CODE_GNB_MML hidden visible" style='cursor:pointer;margin-left:20px;' onclick="clickGoGnb()"><span><%=rb.getString("GO")%></span></a>
									</div>
								</div>
								<div style="height:59%;overflow:auto;">
									<!-- 自动生成的操作表单 -->
									<div id="operValueForm_gnb">
										<ul id="paramNodesUl" class="paramNodesUl"></ul>
									</div>
								</div>
							</div>
							<!-- 参数面板 -->
							<div class="gnbElfcellConfigParamPanel">
							  <div class="gnb-panel-flex-wrapper">
								 <div class="panelTableDiv" id="elfcellshowParamValues_gnb">
									<span><%=rb.getString("CaoZuoLeiXing")%></span>
									<select id="elfcellConfigParamOperType_gnb" class="border border-box" style="height:26px;vertical-align:middle;margin-left: 10px">
										<option value="LST" selected>LST</option>
										<option value="MOD" class="CODE_GNB_MML hidden">MOD</option>
										<option value="ADD" class="CODE_GNB_MML hidden">ADD</option> 
										<option value="RMV" class="CODE_GNB_MML hidden">RMV</option>
									</select>
										
									<div style="margin-top: 20px;" id="paramForElfcellLSTDiv_gnb">
										<div class="gnbMmlItemDiv" style="margin-bottom: 5px">
											<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span><input type="text" name="gnbParamPathNameForElfcellLSTConfig" placeholder='<%=rb.getString("JiaoYanGuiZe")%>' class="showWholeVal border border-box item"/>
											<a onclick="addParampathLSTInputText_elfcellGnb(this)" id="addbtnForLstAndRMVPath" class="operationAdd CODE_GNB_MML hidden visible">
												<span class="img-suffix el-icon el-icon-plus"></span>
											</a>
											<p class="prompt"></p>
										</div>
									</div>
									
									<div style="margin-top: 20px;display: none" id="paramForElfcellADDAndMODDiv_gnb">
										<div class="gnbMmlItemDiv" style="margin-bottom: 5px">
											<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span><input type="text" name="gnbParamPathNameForElfcellMODConfig" placeholder='<%=rb.getString("JiaoYanGuiZe")%>'  class='showWholeVal border border-box item'/>
											<span style="margin:0 10px 0 15px;"><%=rb.getString("CanShuZhi")%></span><input type="text" name="gnbParamPathValueForElfcellMODConfig"  class='showWholeVal border border-box item' />
											<a onclick="addParampathMODInputText_elfcellGnb(this)" class="operationAdd CODE_GNB_MML hidden">
												<span class="img-suffix el-icon el-icon-plus"></span>
											</a>
											<p class="prompt"></p>
										</div>
									</div>
									
									<div style="margin-top: 20px;display: none" id="paramForElfcellRMVDiv_gnb">
										<div class="gnbMmlItemDiv" style="margin-bottom: 5px">
											<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span><input type="text" name="gnbParamPathNameForElfcellRMVConfig" placeholder='<%=rb.getString("JiaoYanGuiZe")%>' class="showWholeVal border border-box item"/>
										    <p class="prompt"></p>
										</div>
									</div>
									
								</div>
								<!-- 底部固定按钮栏 -->
								<div class="gnb-panel-bottom-bar">
									<a id="customMmlSaveBtnGnb" class="linkbutton CODE_GNB_MML hidden" @click="openCommandConfirm_gnb" style="margin-left:10px;display:none;"><span><%=rb.getString("BaoCun")%></span></a>
									<a class="linkbutton CODE_GNB_MML hidden visible" onclick="clickRussiaElfcellGoGnb()" style="margin-left:0px;"><span><%=rb.getString("GO")%></span></a>
								</div>
							  </div>
							</div>
						</div>
					</div>
				</div>	
			</div>
		</div>
		<div class="MMLScript_gnb" id="MMLScript_gnb"></div>
	</div>

	<el-dialog title="<%=rb.getString("QueRen")%>" :visible.sync="customDlShow" width="400">
		<div>
			<el-form ref="commandform" :model="commandForm" :rules="rules" label-position="top">
				<el-form-item label='<%=rb.getString("ZiDingYiCanShuMing")%>' required prop="commandName">
					<el-input v-model="commandForm.commandName" maxlength="50" style="width: 90%;"></el-input>
				</el-form-item>
				<el-form-item label='<%=rb.getString("Type")%>' required>
					<el-radio-group v-model="commandForm.isPublic">
						<el-radio label="0">Private</el-radio>
						<el-radio label="1">Public</el-radio>
					</el-radio-group>
				</el-form-item>
			</el-form>
		</div>
		<el-button-group style='margin-top:20px;'>
		   <el-button type='primary' size='small' @click="commandSave"><%=rb.getString("QueDing")%></el-button>
		   <el-button size='small' @click="customDlShow = false"><%=rb.getString("QuXiao")%></el-button>
	   </el-button-group>
	</el-dialog>
</div>

<%-- 工具栏-基站列表 --%>
<div id="toolbar_gridCell_cellParam_gnb" class="admin_query_head toolbarContainer" style="padding:10px 0 10px !important;">
	<div class="defaultQuery">
		<div>
			<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun")%>" 
				name="search_text"
				inputId="commonQueryText_gnb"
				targetId="eNBHighQueryDiv_gnb"
				placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>"
				data-options="query: queryGridCellParamGnb"></div>

            <!-- 高级查询 -->
			<div class="eNBHighQuery-gnb" id="eNBHighQueryDiv_gnb" style='top:35px;'>
				<ul class="inputslist">				
				 	<li style="margin:20px 20px 0;">
						<select id ="stationConfigDeviceGroupGnb" name="type" class="border border-box" style="width:260px;margin-right:1px;height: 26px;"></select>
					</li>
				</ul>
			</div>
		</div>
	</div>
</div>

<form id="configResultExportTxtFile" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="" name="result" id="result"/>
</form>

<div id="winCLIFile" class="easyui-window" title="<%=rb.getString("CLIFile")%>"
	data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:650,height:500">
</div>

<div id="mmlTiShiGnb" class="easyui-window" title="<%=rb.getString("TiShi") %>" data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:500,height:300,resizable:false">
    <div class="easyui-layout" data-options="border:false,fit:true">
		 <div id="mmlTiShiGnbContent" region="center"></div>
		 <div region="south" data-options="border:false,height:50" style="padding: 0px 0px 10px 0;">
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="closeMMLTiShiGnb()"><%=rb.getString("GuanBi")%></a>
		 </div>
    </div>
</div>

<div id="mml_device_dl_gnb" class="easyui-dialog" style="width: 600px;height: 400px;"
	data-options="closed: true, modal: true"
	title='<%=rb.getString("TianJia") %>'>
	<div style="padding: 20px;">
		<div style="padding-bottom: 10px;font-weight: bold;"><%=rb.getString("XiaoZhanBianMa") %></div>
		<textarea id="batch_sn_textarea_gnb" rows="10" cols="80"></textarea>
		<p id="batch_sn_tips_gnb" style="color: red;min-height: 18px;"></p>

		<div class="tipText">
			<span class="el-icon el-icon-circle-info infoTip"></span>
			<span><%=rb.getString("eNBZhuCeTiShiWenZi")%></span>
		</div>
		<div style="padding-top: 10px;">
			<a class="linkbutton" onclick="batchInputSN_gnb()"><span><%=rb.getString("QueDing") %></span></a>
			<a class="linkbutton" onclick="closeBatchDL_gnb()"><span><%=rb.getString("QuXiao") %></span></a>
		</div>
	</div>
</div>

<script type="text/javascript">
	var choosedGroupId = -1;
	var cellIndexLocation = "";
	var isShowRightClickMenu = false;
	var hardware_version_gnb = "BaiBNX1.0";
	var original_hardware_version_gnb = '',
		mmlCommandTreeSearchTxt = '';


	var gnbmmlVue = new Vue({
		el:'#cellParam_gnb',
		data(){
			return {
				resultType:'res',
				isResult:false,
				resultTitle:'<%=rb.getString("PeiZhiJieGuo")%>',
				resultContent:'',
				helpData:[],
				paramData:{
					operID : '',
					actionType : '',
					searchText:''
				},
				paramProduct:'',

				customDlShow: false,
				oldCommandName: '',
				commandForm: {
					commandName: '',
					isPublic: '0',
					groupId: '',
					nodeName: '',
					tempNodeId: null  // 用于存储临时节点ID
				},
				rules: {
					commandName: [
						{required: true, message: '<%=rb.getString("QingShuRuZhiLingMing")%>'},
						{pattern: /^[a-zA-Z0-9_]{1,50}$/, message: '<%=rb.getString("ZiDingYiZhiLingJiaoYan")%>'}
					]
				}
			}
		},
		methods:{
			changeType(val){
				var vm = this;
				if(val == 'help'){
					vm.resultTitle = '<%=rb.getString("BangZhu")%>';
					//$("#cellParam_gnb .command-tips").hide();
					//$("#cellParam_gnb .getSetParamValResult").hide();
				}else{
					vm.resultTitle = '<%=rb.getString("PeiZhiJieGuo")%>';
					//$("#cellParam_gnb .command-tips").show();
					//$("#cellParam_gnb .getSetParamValResult").show();
				}
			
			},
			init(operID, actionType, operName,cellNumber){	
				var vm = this;

				var param = {
					operID : operID,
					actionType : actionType,
					searchText:''
				};
				Object.assign(vm.paramData,param)
				$.post("${ctx}/cell/param/getParamGroupHelpInfos.action", param, function(data) {
					if (data) {
						vm.helpData = data;
						vm.paramProduct = '';
					}
				}, "json");

				/* vm.helpData = 
				{
					"param_name":"NTP SYNC(MOD NTP_SYNC)",
					
					"groupList":
						[
							{
								"paramName":"NTP interval",
								"dataModel":"unsignedInt",
								"dataRange":"1-1440",
								"dftValue":"60",
							
								"dynamic":"0",
								"explanation":"NTP interval",
								"mib_dn":"LTE_X_BAICELLS_NTP_SYNC_INTERVAL",
								"name_path":"Device.Time.LTE_X_BAICELLS_NTP_SYNC_INTERVAL",
							},
							{
								"paramName":"TimeZone",
								"dataModel":"unsignedInt",
								"dataRange":"{GMT0,EAT-3,CET-1}",
								"dftValue":"GMT0",
								
								"dynamic":"1",
								"explanation":"NTP interval",
								"mib_dn":"NTP_LOCAL_TIME_ZONE_NAME",
								"name_path":"Device.Time.LocalTimeZoneName",
							}
						]
				} */
			},
			//点击按钮复制Trpath值
			copyPath(path){
				//复制path值到剪切板
				var input = document.createElement('input');
				input.setAttribute('readonly', 'readonly');
				input.setAttribute('value', path);
				document.body.appendChild(input);
				input.select();
				input.setSelectionRange(0, input.value.length),
				document.execCommand('Copy');
				document.body.removeChild(input);
				this.$message({
					message: '<%=rb.getString("ChengGong")%>',
					type: 'success'
				});
				
			},
			//查询MML命令
			queryEnbMML(){
				var vm = this;

				$.post("${ctx}/cell/param/getParamGroupHelpInfos.action", vm.paramData, function(data) {
					if (data) {
						vm.helpData = data;
					}
				}, "json");
			},
			exportEnbMml(){
				var vm = this;
				var url = "${ctx}/cell/param/exportParamGroupInfo.action";
				var param = {
					timeZone:timeZone
				};
				param = Object.assign(param,vm.paramData);
				exportByForm(url,param)
			},

			openCommandConfirm_gnb() {
				var vm = this;

				vm.customDlShow = true;
				vm.commandForm.commandName = vm.oldCommandName;
			},
			commandSave() {
				var vm = this;

				vm.$refs.commandform.validate(function(v) {
					if(v) {
						saveCustomParamCommandGnb(vm.commandForm.commandName, vm.commandForm.groupId, vm.commandForm.nodeName, vm.commandForm.isPublic);
					}
				});
			}
		},
		mounted(){
			this.init()
		}
	})
	

	//俄罗斯版本，按参数路径修改参数，选择操作类型下拉框修改事件
	$("#elfcellConfigParamOperType_gnb").bind("change",function(){
		$(".showWholeVal").css("borderColor","#C9D1D6");
		$(".showWholeVal").siblings("p.prompt").text("");
		if ("MOD" == this.value) {
			$("#paramForElfcellADDAndMODDiv_gnb").css("display", "block");
			$("#paramForElfcellLSTDiv_gnb").css("display", "none");
			$("#paramForElfcellRMVDiv_gnb").css("display", "none");
			
			//清空参数路径和值
			$("input[name=gnbParamPathNameForElfcellMODConfig]").val("");
			$("input[name=gnbParamPathValueForElfcellMODConfig]").val("");

			//只留一个输入框
			$("#paramForElfcellADDAndMODDiv_gnb div:gt(0)").remove();
		}  else if ("RMV" == this.value){
			//RMV 操作只有一个输入框
			$("#paramForElfcellADDAndMODDiv_gnb").css("display", "none");
			$("#paramForElfcellLSTDiv_gnb").css("display", "none");
			$("#paramForElfcellRMVDiv_gnb").css("display", "block");
			
			//清空参数路径
			$("input[name=gnbParamPathNameForElfcellRMVConfig]").val("");
		} else {
			//LST 操作可以有多个输入框
			$("#paramForElfcellADDAndMODDiv_gnb").css("display", "none");
			$("#paramForElfcellRMVDiv_gnb").css("display", "none");
			$("#paramForElfcellLSTDiv_gnb").css("display", "block");
			
			//清空参数路径和值
			$("input[name=gnbParamPathNameForElfcellLSTConfig]").val("");

			if("ADD" == this.value) {
				$("input[name=paramPathNameForElfcellLSTConfig]").attr('placeholder', 'cannot contain {i} must contain . must end with .');
				$('#paramForElfcellLSTDiv_gnb #addbtnForLstAndRMVPath').addClass('hide-item-cls');
			}else {
				$('#paramForElfcellLSTDiv_gnb #addbtnForLstAndRMVPath').removeClass('hide-item-cls');
			}
			//只留一个输入框
			$("#paramForElfcellLSTDiv_gnb div:gt(0)").remove();
		}
	});
	slideuplistGnb();
	
	// 收起已选基站滑出页
	function slideuplistGnb(){
		$(".enb-list-ctn-gnb").slideUp();
	}
	// 滑出已选设备列表浮层
	function slidedownlistGnb(){
		$(".enb-list-ctn-gnb").slideDown();
		refreshEnbListGnb();
	}
	// 刷新已选设备列表
	function refreshEnbListGnb(){
		var tb = $('#gridCell_cellParam_gnb'),
			ctn = $(".enb-list-ctn-gnb .list-body"),
			rows = tb.datagrid('getSelections');
		ctn.html('');
		rows.map(function(row){
			var item = '<div class="list-item-info">' +row.host_name + ' (' + row.serial_number + ') <span class="list-item-op" onclick="clearDoStgRecordGnb(\''+row.small_cell_code+'\',this)">x</span></div>'
			ctn.append(item);
		})
	}
	// 删除所有已选设备记录
	function delAllDoStgRecordGnb(){
		$("#gridCell_cellParam_gnb").datagrid('clearSelections').datagrid('clearChecked');
		$(".enb-list-ctn-gnb .list-body").html('');
		setTimeout(function(){
			slideuplistGnb();
		},1000);
	}
	// 删除单个已选设备记录
	function clearDoStgRecordGnb(id,span){
		var tb = $("#gridCell_cellParam_gnb"),
			rows = tb.datagrid('getSelections'),
			crows = tb.datagrid('getChecked');
		
		var rIdx = -1;
		rows.map(function(item,idx){
			if(item.small_cell_code == id) rIdx = idx;
		});
		if(rIdx>=0) {
			rows.splice(rIdx,1);
			crows.splice(rIdx,1);
		}
		
		var index = tb.datagrid('getRowIndex',id);
		if(index>=0){
			tb.datagrid('unselectRow',index);
		}
		
		$(span).parents('.list-item-info:first').remove();
	}
	// 更新已选设备数量展示
	function refreshEnbNumGnb(){
		setTimeout(function(){
			var tb = $('#gridCell_cellParam_gnb'),
			ctn = $("#MML_Config_gnb .circle-num-tips"),
			rows = tb.datagrid('getSelections');
			if(rows.length){
				$(".showSelectNum").show();
				ctn.text(rows.length);
				
			}else{
				$(".showSelectNum").hide();
			}
		},10)
	}
	/**
	* 清除和下载操作
	* @param item{object}: 操作类型对象
	**/
	function menuHandlerGnb(item){
		var itemName = item.name;
		if(itemName == "copy"){
			
		}else if(itemName == "clear"){
			$("#showParamValues_gnb").children().remove();
			$("#showParamValues_gnb").unbind("contextmenu");
			isShowRightClickMenu = false;
			$('#MML_Config_gnb .param-values-oper').addClass('empty');
			gnbmmlVue.isResult = false;
		}else if(itemName == "download"){
			exportConfigResultTxtFileGnb();
		}else{
			showMsg('error_msg','<%=rb.getString("FeiFaCaoZuo")%>');
		}
	}
	
	/* 显示高级查询选项  */
	function moreQueryImgFun(){
		if($("#configEnbQueryImg").attr("flag")=="1"){
			$("#eNBHighQueryDiv_gnb").slideDown(500);
			$("#configEnbQueryImg").attr("flag","0");
			$("#configEnbQueryImg").addClass('el-icon-common-query-up').removeClass('el-icon-common-query-down');
		}else{
			$("#eNBHighQueryDiv_gnb").slideUp(500);
			$("#configEnbQueryImg").attr("flag","1");
			$("#configEnbQueryImg").addClass('el-icon-common-query-down').removeClass('el-icon-common-query-up');
		}
		event.stopPropagation();
	}
	
	/* 阻止冒泡  */
	$(".eNBHighQuery-gnb").click(function(event){
		event.stopPropagation();
	})

	
	// 下载指令执行结果
	function exportConfigResultTxtFileGnb(){	
		var rs = $("#showParamValues_gnb");
		var url = "${ctx}/cell/param/exportConfigResultTxtFile.action";
		//处理数据，拼接结果文本
		var result = '',
			chlidres = rs.children();
	
		$.each(chlidres, function(index, item){
			if($(item).is('.command-tips')) { // 提示
				result += $(item).text() +"@<H>";
			}else {// 返回的参数列表
				var titleInfos = $(item).find('>span'),
					paramList = $(item).find('>div');

				// 拼接命令头信息
				result += Array.from(titleInfos).map(function(span){ return $(span).text(); }).join('@<H>') + '@<H>';

				// 拼接参数详情信息
				paramList.each(function(idx, div){ 
					var cells= $('>.cell',div);

					// 拼接key、value键值对，多个以逗号连接
					var paramStr = Array.from(cells).map(function(cell){
						var paramISpans = $('span',cell),
							props = Array.from(paramISpans).map(function(span){ return $(span).text(); }).join(':');
							
						return props;
					}).join(',');

					result += paramStr + '@<H>';
				})
			}

			//result += '@<H>';
		});
		//数组拼接
		var arr = [ ];
		arr = result.split('@<H>');
		for(var i = 0 ;i<arr .length;i++){
		  if(arr[i] == "" || typeof(arr[i]) == "undefined"){
				arr.splice(i,1);
                i= i-1;
			}
		}
                                                                                                                                                                                                                                                                                         
		// 将文本提交后台以文件形式下载
		$("#result").val(arr );
		exportByForm(url,{
            timeZone: timeZone,
			result: JSON.stringify(arr)
		});
	}
	// 关闭提示浮层
	function closeMMLTiShiGnb(){
		$("#mmlTiShiGnb").window("close");
	}
	// 查询命令列表
	function queryGridCellParamGnb() {
		if($(".eNBHighQuery-gnb").css('display') == 'block'){
			$(".eNBHighQuery-gnb").slideToggle(300);
		}
		
		
		var search_text = $("#commonQueryText_gnb").val();
		
		$('#gridCell_cellParam_gnb').datagrid('reload',{
			search_text: search_text
		});
		hardwareVersionChangeGnb(hardware_version_gnb);
	}
	// 用于保存所有操作，以datagrid需要的形式
	var allOperNameForSearchGrid = {total: 0, rows: []};
	/**
	* 选择第一个叶子节点
	* @param data{array}: 子节点队列
	* @param status{object}: 父节点
	**/
	function selectFirstLeafGnb(data,status) {
		var map = status || {isLeaf: false,id: ''};
		if(map.isLeaf == true) return;
		if(data) {
			var treeN = $("#operGroupTree_gnb");
			data.map(function(item){
				if(map.isLeaf == true) return;
				
				if(item.children) {
					map = selectFirstLeafGnb(item.children,map);
				}else {
					var node = treeN.tree('find',item.id),
						isLeaf = treeN.tree('isLeaf',node.target);
					if(isLeaf == true) {
						map.isLeaf = true;
						map.node = node;
					}
				}
			});
		}
		return map;
	}
	/**
	* 加载完成事件
	*/
	$(function () {
		var singleArry = [false];
		if(!batchOperation){
			$("#MMLScript_gnbTab").hide();
			$("#gnbMMLBatchInput").hide();
			singleArry = [true]
		}
		closeLoading();
		$("#mmlListCenter_gnb").siblings(".panel-header").css("border-width", "1px 0 1px 0");
		// 加载操作集树形结构
		$("#operGroupTree_gnb").tree({
			animate: true,
			lines: true,
			border: true,
			onlyLeafCheck : true,
			filter: function(q, node) {
				if(q) {
					var lowcaseQ = q.toLowerCase(),
						lowcaseText = node.text.toLowerCase(),
						keywords = node.keyword||[],
						idx = lowcaseText.indexOf(lowcaseQ),
						inKeys = false;

					keywords.map(function(key){
						var lowcaseKey = key.toLowerCase();

						if(lowcaseKey.indexOf(lowcaseQ)>=0) {
							inKeys = true;
						}
					});
					
					return idx >= 0 || inKeys;
				}else {
					return true;
				}
			},
			formatter: function(node){
				var str = '<span code="'+node.id+'">' + node.text + '</span>';
				
				if(node.customized == 'true') {
					if(node.children) {
						if(['PrivateTemplateId', 'PublicTemplateId'].includes(node.id)) {
							str += '<i class="el-icon el-icon-plus" nodeid="'+node.id+'" style="position: absolute;right: 10px;top: 10px;color: #7A7992;" onclick="toADDCommandGnb(&quot;'+ node.text +'&quot;, event)"></i>';
						}else {
							str += '';
						}
					}else {
						// 自定义叶子节点 - 通过预处理标记 _isOwn 判断是否属于当前用户的组
						// _isOwn 在 loadData 之前由 _markOwnershipGnb 预处理设置
						var isOwn = (node._isOwn === true);
						// 临时节点始终属于自己
						if(node.id && String(node.id).indexOf('temp_') === 0) { isOwn = true; }

						if(isOwn) {
							str += '<span class="cus-btn-group" style="position: absolute;right: 10px;top: 10px;">';
							// 如果是 Untitled 指令，不显示复制和修改按钮
							if(node.text && !node.text.startsWith('Untitled')) {
								str += '<i class="cus-bt-cls el-icon el-icon-operation-copy" nodeid="'+node.id+'" style="margin-right: 8px;color: #7A7992;cursor:pointer;" onclick="copyCusNodeGnb(this,event)" title="复制"></i>';
								str += '<i class="cus-bt-cls el-icon el-icon-operation-edit" nodeid="'+node.id+'" style="margin-right: 8px;color: #7A7992;cursor:pointer;" onclick="editCusNodeGnb(this,event)" title="修改"></i>';
							}
							str += '<i class="cus-bt-cls el-icon el-icon-close" nodeid="'+node.id+'" style="color: #7A7992;cursor:pointer;" onclick="removeCusNodeGnb(this,event)" title="删除"></i>';
							str += '</span>';
						}
						// 其他人的指令组：不渲染操作按钮，只能查看
					}
				}

				return str;
			},
			onSelect: function(node){
				// 如果是叶子节点，构建右侧操作框；如果非叶子节点，则切换打开状态
				if ($("#operGroupTree_gnb").tree("isLeaf", node["target"])) {
					if(node.customized == 'true') {
						var tabNav = $('span[tabtit="gnbElfcellConfigParamPanel"]');
						turnTabs(tabNav);
						reviewCustomParamConmmandGnb(node);
					}else {
						resetCommandParamGnb();
						// 删除之前的元素
						$("#cellParam_gnb #paramNodesUl li").remove();
						$("#cellParam_gnb #paramNodesUl div").remove();
						doActionByOperIDGnb(node);
					}
				} else {
					$("#operGroupTree_gnb").tree("toggle", node["target"])
				}
			},
			onLoadSuccess: function(node,data){
				if(mmlCommandTreeSearchTxt){
					var map = selectFirstLeafGnb(data),
						_tree = $(this);
					if(map.node) {
						var target = map.node.target;
						setTimeout(function(){
							_tree.tree('expandTo',target).tree('select',target);
						},500)
					}
				}
			}
		});
		
		$("#gridCell_cellParam_gnb").datagrid({
			url: '${ctx}/cell/param/getCellListOfMML.action?forSelect=1',
			queryParams:{like_fields:"serial_number,host_name",isGnb: 1},
			singleSelect:singleArry[0],
			fit:true,
			fitColumns:true,
			border:false,
			rownumbers:true,
			pagePosition:'bottom',
			pageSize : 100,
			pageList : [100],
			idField:'small_cell_code',
			toolbar:'#toolbar_gridCell_cellParam_gnb',
			onLoadSuccess:gridCellParamDatagridLoadSuccessGnb,
			onLoadError:datagridLoadError,
			onBeforeLoad: beforeLoad_gridCell_cellParam_gnb,
			onCheck: checkDoStg,
			onUncheck: uncheckDoStg,
			onCheckAll: checkAllDoStg,
			onUncheckAll: uncheckAllDoStg,
			pagination : true,
			striped: true,
			columns: [[
				{field: 'ck', checkbox: true},
				{field: 'small_cell_code', hidden: true},
				{field: 'connection_status',sortable:true,fixed:true,width: 30,formatter:connStatusFormatter},
				{field: 'serial_number',sortable:true,width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
				{field: 'host_name',sortable:true,width: 100, title: '<%=rb.getString("HostName")%>'},
				{field: 'mme_enable',hidden:true, title: 'MME_ENABLE'}
			]]
		});
		
		function checkDoStg(index,row){
			var textOperName = $("#textOperName_gnb").combo('getText');
			if("MOD MME" == textOperName){
				showMMEPoolWayGnb(true,row);
			}
			refreshEnbNumGnb();
		}

		function uncheckDoStg(index,row){
			var textOperName = $("#textOperName_gnb").combo('getText');
			if("MOD MME" == textOperName){
				showMMEPoolWayGnb(false,row);
			}
			refreshEnbNumGnb();
		}
		
		function checkAllDoStg(rows){
			refreshEnbNumGnb();
		}
		
		function uncheckAllDoStg(rows){
			refreshEnbNumGnb();
		}
		// 操作名称输入框，初始化下拉面板
		$("#textOperName_gnb").combobox({
			width: 250,
			panelWidth: 400,
			panelHeight: 300,
			valueField: 'oper_id',
			textField: 'oper_name',
			onSelect: clickRowTableSearchOperNameGnb
		});
		
		$("#gridCell_cellParam_gnb").datagrid("getPager").pagination({
			layout:['prev','manual','next','refresh']
		});
		
		// 基站搜索-类型-change事件
		$("#toolbar_gridCell_cellParam_gnb select[name='type']").bind("change", function(e) {
			var type = e.target.value;
			$("#toolbar_gridCell_cellParam_gnb .input_li").hide();
			$("#toolbar_gridCell_cellParam_gnb ." + type).show();
			
		});
		
		// 基站搜索-地域树-下拉面板
		$("#regnTreeCombo_cellParam").combotree({
			url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
			panelWidth: 200
		});
		
		$("#stationConfigDeviceGroupGnb").combobox({
			url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action',
			width: 260,
			panelWidth: 260,
			panelHeight: 200,
			valueField: 'id',
			textField: 'group_name',
			onSelect: chooseStationConfigDeviceGroupGnb,
			onLoadSuccess : function(){
				
			}
		});
		
		$("#stationConfigDeviceGroupGnb").combobox('setValues',['-1','<%=rb.getString("QuanBuSheBeiZu")%>']); 
	
		// 样式修改
		$(".group_id input.textbox-text").css("padding", "0").css("margin", "0");
		
		$('#MMLScript_gnb').load('${ctx}/task/MMLScript/toGnbMMLScriptTaskList.action',function(data){
			$.parser.parse(this);
		})

		hardwareVersionChangeGnb(hardware_version_gnb);
	});
	/**
	* 设备组下拉选中事件
	* @param data{object}: 选中项数据
	**/
	function chooseStationConfigDeviceGroupGnb(data){
		choosedGroupId = data.id;
		$("#gridCell_cellParam_gnb").datagrid("reload");	
	}
	/**
	* 根据操作ID和操作类型，动态生成操作面板
	* @param operID{string}：命令树节点Id
	* @param actionType{string}：命令类型
	* @param operName{string}：命令名
	* @param cellNumber{string}：基站编码
	**/
	function  doActionByOperIDAndTypeGnb(operID, actionType, operName,cellNumber) {
		// 向后台请求该操作包含的子节点信息
		var param = {
			operID : operID,
			actionType : actionType
		};

		var regVersion = /^436Q/;
		var hardwareVer = regVersion.test(hardware_version_gnb );
		var reg430 = /^NEU430/;
		var hardware430 = reg430.test(hardware_version_gnb ),
			isCR4860 = /^CR_B4860_/.test(hardware_version_gnb);
		var isMLN = /^MLN_/.test(hardware_version_gnb);
		var matchCodes = [
				"NBIOT1.0",
				"CR4.0",
				"QC4.2",
				"QC4.2T",
				"QC4.2RELAY",
				"QC3.1",
				"CA2.0",
				"436Q_CA1.0"
			];
		
		if (hardware430 || matchCodes.includes(hardware_version_gnb) || hardwareVer || isCR4860 || isMLN || true) {
			if(isLoadJSPGnb(operID,actionType,hardware_version_gnb)) {
				$("#operValueForm_gnb").panel({
					border : false,
					queryParams: param,
					href: '${ctx}/eNodeB/config/toConfigFormPage.action?isGnb=1',
					onLoad: function(){
						createMMLGnb();
					}
				});
			}else {
				$.post("${ctx}/cell/param/getParamGroupTreeNodes.action?isGnb=1", param, function(data) {
					if($("#cellParam_gnb #paramNodesUl").length==0){
						var $paramNodesUl = '<ul id="paramNodesUl" class="paramNodesUl"></ul>'
						$("#operValueForm_gnb").append($paramNodesUl);
					}
					/*删除之前的元素*/
					$("#cellParam_gnb #paramNodesUl li").remove();
					$("#cellParam_gnb #paramNodesUl div").remove();
					allMibDn = "";
					
					var  innerVersionList = ["CA2.0", "436Q_CA1.0", "NEU430_CA1.0", "CR_B4860_AC4.0","BaiBNX1.0"];
					if(innerVersionList.includes(hardware_version_gnb)){
						var operNameList = ["NTP","SYNC","IPSEC","SAS","REBOOT","RESET","ACT","DEACT","HTTP","POWER_AMP","LMT_LOGIN"],
							excluded = operNameList.every(function(item){
								return operName.indexOf(item) == -1;
							});

						if (actionType != "v_lst") {
							if (excluded) {
								var $liForNumCells = createCellIndex(cellNumber);
								$liForNumCells.css('display','block');
								if(cellNumber>1) $("#cellParam_gnb #paramNodesUl").append($liForNumCells);
								
								$("#numCells").change(function() {
									var $1282obj = $('[name=i]'),
										$1330obj = $('[name=LTE_DL_EARFCN]'),
										$1331obj = $('[name=LTE_UL_EARFCN]'),
										$1332obj = $('[name=LTE_DL_BANDWIDTH]'),
										$1333obj = $('[name=LTE_UL_BANDWIDTH]'),
										$1334obj = $('[name=LTE_TDD_SUBFRAME_ASSIGNMENT]'),
										$1335obj = $('[name=LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS]'),
										$1336obj = $('[name=LTE_FREQ_BAND_INDICATOR]'),
										$1337obj = $('[name=LTE_BANDS_SUPPORTED]'),
										$1338obj = $('[name=LTE_OAM_PLMNID]'),
										$1377obj = $('[name=LTE_INTER_FREQ_DL_EARFCN]'),
										$1387obj = $('[name=LTE_NEIGH_LIST_LTE_CELL_PLMNID]'),
										$1388obj = $('[name=LTE_NEIGH_LIST_LTE_CELL_CID]'),
										$1389obj = $('[name=LTE_NEIGH_LIST_LTE_CELL_EARFCN]'),
										$1390obj = $('[name=LTE_NEIGH_LIST_LTE_CELL_PHY_CELLID]'),
										$1391obj = $('[name=LTE_X_ZTE_NEIGH_LIST_LTE_CELL_TAC]'),
										$1392obj = $('[name=LTE_NEIGH_LIST_LTE_CELL_QOFFSET]'),
										$1393obj = $('[name=LTE_NEIGH_LIST_LTE_CELL_CIO]');
									
									//如果SAS开关打开，则该参数不可配置
									if(false && SASEnble == "1" && operName == "MOD CELL"){
										[$1330obj, $1331obj, $1336obj,$1337obj].map(function(item){
											item.attr("disabled",true);
										});
										[$1332obj, $1333obj].map(function(item){
											item.attr("disabled",true).css("background","#EAF1F4");
										});
									}
									
									var operName = $("#operGroupTree_gnb").attr("operName");
									$("#inputMML_gnb").val(operName);
									selectedCellNum = $(this).val();
									
									if ( selectedCellNum >= 2) {
										$("#cellParam_gnb #paramNodesUl input").val("").blur();
										$("#cellParam_gnb #paramNodesUl select").val("");
										$(this).val(selectedCellNum);
										
										$1338obj.attr("disabled", "disabled");
										[$1332obj, $1333obj, $1334obj, $1335obj].map(function(item){
											item.attr("disabled", "disabled");
											item.attr("style", "background-color:#EAF1F4");
										});
										
										if( operName.indexOf("EUTRANNFREQ") > -1 || operName.indexOf("EUTRANNCELL") > -1){
											var selects = $("#cellParam_gnb #paramNodesUl select");
											var inputs = $("#cellParam_gnb #paramNodesUl input");
											for(var i=0;i<selects.length;i++){
												if($(selects[i]).attr("id")!= "numCells"){
													$(selects[i]).attr("disabled", "disabled").attr("style", "background-color:#EAF1F4");
													$(selects[i]).next().hide();
												}
											}
											for(var i = 0;i<inputs.length;i++){
												$(inputs[i]).attr("disabled", "disabled");
												$(inputs[i]).removeClass('err_border').next().hide();
											}
										}
									} else {
										$("#cellParam_gnb #paramNodesUl input").val("").blur();
										$("#cellParam_gnb #paramNodesUl select").val("");
										$(this).val(selectedCellNum);
										
										$1338obj.attr("disabled", false);
										[$1332obj, $1333obj, $1334obj, $1335obj].map(function(item){
											item.attr("disabled", false);
											item.attr("style", "background-color:#FFFFFF");
										});
										
										if( operName.indexOf("EUTRANNFREQ") > -1 || operName.indexOf("EUTRANNCELL") > -1){
											var selects = $("#cellParam_gnb #paramNodesUl select");
											var inputs = $("#cellParam_gnb #paramNodesUl input");
											for(var i=0;i<selects.length;i++){
												if($(selects[i]).attr("id")!= "numCells"){
													$(selects[i]).attr("disabled",false).attr("style", "background-color:#FFFFFF");
												}
											}
											for(var i = 0;i<inputs.length;i++){
												$(inputs[i]).attr("disabled", false);
												$(inputs[i]).removeClass('err_border').next().hide();
											}
											$1282obj.next().show();
											if(operName == 'ADD EUTRANNFREQ'){
												$1377obj.next().show();
											}
											if(operName == 'ADD EUTRANNCELL'){
												[$1387obj, $1388obj, $1389obj, $1390obj, $1391obj, $1392obj, $1393obj].map(function(item){
													item.next().show();
												});
											}
										}
									}
								});
							}
						}else if(actionType == "v_lst") {
							if (excluded) {
								var $checkBoxForNumCells = createCheckBoxIndex(cellNumber);
								if(cellNumber>1) $("#cellParam_gnb #paramNodesUl").append($checkBoxForNumCells);
							}
						}
					}
					for (var nodeNum=0; nodeNum < data.length; nodeNum++) {
						var $li = createEleByIDAndTypeGnb(data[nodeNum]);
						if ($li) {
							$("#cellParam_gnb #paramNodesUl").append($li);
						}
					}
					
					$.parser.parse('#cellParam_gnb #paramNodesUl');
					customEventFnc5G();
					var $1282obj = $('[name=i]'),
						$1330obj = $('[name=LTE_DL_EARFCN]'),
						$1331obj = $('[name=LTE_UL_EARFCN]'),
						$1332obj = $('[name=LTE_DL_BANDWIDTH]'),
						$1333obj = $('[name=LTE_UL_BANDWIDTH]'),
						$1336obj = $('[name=LTE_FREQ_BAND_INDICATOR]'),
						$1337obj = $('[name=LTE_BANDS_SUPPORTED]');
						$1399obj = $('[name=LTE_CELL_POWER_MODIFY]');
					
					//如果SAS开关打开，则该参数不可配置
					if(false && SASEnble == "1" && operName == "MOD CELL"){
						[$1330obj, $1331obj, $1336obj,$1337obj,$1399obj].map(function(item){
							item.attr("disabled",true);
						});
						[$1332obj, $1333obj].map(function(item){
							item.attr("disabled",true).css("background","#EAF1F4");
						});
					}
					
					$("select[name='LTE_TM_MODE']").blur(function(){
					
						if ($("#modeRebootTips").length>0) return;
						
						var tips = $("<div id='modeRebootTips'>" + "<%=rb.getString("ShiFouXuYaoChongQi")%>"+ "</div>");
						$(this).after(tips);
					})
					createMMLGnb();
				},"json"); 
			}
		} else {
			$("#operValueForm_gnb").panel({
				border : false,
				queryParams: param,
				href: '${ctx}/eNodeB/config/toConfigFormPage.action?isGnb=1',
				onLoad: function(){
					createMMLGnb();
				}
			});
		}

	}
	/**
	* 检测条件是否要走加载jsp逻辑
	* @param operID{string}：命令树节点Id
	* @param actionType{string}：命令类型
	* @param version{string}：版本
	**/
	function isLoadJSPGnb(operId,actionType,version) {
		var bool = false,
			list = [
				{id: '100018',type: 'v_mod',v: 'BaiBNX1.0'},
				{id: '100018',type: 'v_add',v: 'BaiBNX1.0'},
				{id: '100018',type: 'v_rmv',v: 'BaiBNX1.0'},
				{id: '100018',type: 'v_lst',v: 'BaiBNX1.0'}
			];

		list.map(function(item){
			if(item.id == operId && item.type == actionType && item.v == version) bool = true;
		})

		return bool;
	}
	/**
	* 对CELL_ID,即ECI做特殊处理，如果输入的是268435456，也可以成功
	* @param e{event}: 事件对象
	**/
	function validateECI(e){
		var ele = $(e["target"]);
		var name = ele.attr("name");
		var val = ele.val();
		//获取errorSpan元素的内容
		var $span = ele.parent().find("span");
		var spanVal = $($span[0]).text();
		//如果是268435456则将范围扩大到最大值为268435456，否则最大值为268435455
		if(val == "268435456"){
			ele.attr("max_value","268435456");
		}else{
			ele.attr("max_value","268435455");
		}
		validateMaxAndMinVal(e);//验证最大值最小值
		if($(e["target"]).hasClass("err_border")){
			//如果验证不通过，则必定是最大值最小值的验证，title和span将显示unsignedInt-[0:268435455]
			ele.attr("title",ECITitle.normal);
			$span.text(ECITitle.normal);
			$span.addClass("errSpan");
			$span.css("color","red");
			$span.show();
		}else{//如果通过了验证，则需要修改title和span的内容
			if(val == "268435456"){
				ele.attr("title",ECITitle.special);
				$span.text(ECITitle.special);
				$span.removeClass("errSpan");
				$span.css("color","green");
				$span.show();
			}else{
				ele.attr("title",ECITitle.normal);
				$span.text(ECITitle.normal);
			}
		}
	}

	var ECITitle = {};
	/**
	* 根据元素的数据，生成元素
	* @param eleData{object}: 元素数据
	**/
	function createEleByIDAndTypeGnb(eleData) {
		if(allMibDn!=""){
			allMibDn = allMibDn+","+eleData["mib_dn"];
		}else{
			allMibDn = eleData["mib_dn"];
		}
		var $li = $("<li></li>");
		var paramName = eleData["paramName"] || eleData["PARAM_NAME"];
		var defaultVal = eleData["dftValue"] || '';
		var $label = $("<label for='" + eleData["param_id"] + "'>" + paramName + ":</label>");
		var ele;
		var errSpan;
		var dataType = eleData["dataType"] || eleData["v_type"] || '';
		var must = eleData["must"];

		var type_1 = /string-(\d+)/;
		var type_2 = /string-\[(\d+)\:(\d+)\]/;
		var type_3 = /enum-\{(.*,.*)\}/;
		var type_3_1 = /bool-\{(.*,.*)\}/;
		var type_4 = /.*[Ii]nt-\[(-{0,1}\d+)\:\]/;
		var type_5 = /.*[Ii]nt-\[(-{0,1}\d+)\:(-{0,1}\d+)\]/;
		var type_5_1 = /.*[Ii]ntList-\[(-{0,1}\d+)\:(-{0,1}\d+)\]/;
		var type_6 = /.*[Ii]nt-\[\:(-{0,1}\d+)\]/;
		var type_7 = /enum-\{(.*)\}-\{(.*)\}/;
		var type_7_1 = /bool-\{(.*,.*)\}-\{(.*,.*)\}/;
		
		if (dataType == "string") {
			<%--没有其它限制条件--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal + "'>");
			ele.bind("blur", trimValue);
		}
		else if (dataType == "Ipv4AddrArr") {
			// ipv4 mult 校验 
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal + "'>");
			ele.bind("blur", validateMultIPV4Address_gnb);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>" +eleData["title"]+ "</div>");
		}
		else if (dataType == "Ipv4Addr") {
			// ipv6 校验 
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal + "'>");
			ele.bind("blur", validateIPV4Address_gnb);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>" +eleData["title"]+ "</div>");
		}else if (dataType == "Ipv6Addr") {
			// ipv6 校验 
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal + "'>");
			ele.bind("blur", validateIPV6Address_gnb);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>" +eleData["title"]+ "</div>");
		}  else if (type_1.test(dataType)) {
			<%--样式：string-64，最大长度为64个字符--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal + "' max_length='" + type_1.exec(dataType)[1] + "'>");
			ele.bind("blur", trimValue);
			ele.bind("blur", validateMaxAndMinLength);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>" +dataType+ "</div>");
		} else if (type_2.test(dataType)) {
			<%--样式：string-(3:4)，最小长度3,最大长度4--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal + "' min_length='"
			+ type_2.exec(dataType)[1] + "' max_length='" + type_2.exec(dataType)[2] + "'>");
			ele.bind("blur", trimValue);
			ele.bind("blur", validateMaxAndMinLength);
			if ((hardware_version_gnb =="QC3.1" || hardware_version_gnb =="QC4.2" || hardware_version_gnb=="QC4.2RELAY" || hardware_version_gnb=="QC4.2T") && eleData["mib_dn"] == "LTE_REFERENCE_SIG_POWER") {
				ele.bind("blur", validateRangeNumber);
			}
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>"+dataType+"</div>");
		} else if (type_7.test(dataType)) {
			<%-- 样式：enum-[n1,n2,n3,n4,n6,n8,n10]-[1,2,3,4, 6, 8, 10]，特殊枚举类型，显示值和下发的值不一致 --%>
			ele = $("<select id='" + eleData["param_id"] + "'></select>");
			var textItemArr = type_7.exec(dataType)[1].split(",");
			var valueItemArr = type_7.exec(dataType)[2].split(",");
			ele.append($("<option value=''></option>"));
			for (var i = 0; i < textItemArr.length; i++) {
				var optionValue = valueItemArr[i].trim().replace(/&/g,','),
					optionText = textItemArr[i].trim().replace(/&/g,',');
				var itemOption = $("<option value='" + optionValue + "'>" + optionText + "</option>");
				ele.append(itemOption);
			}
			<%--如果有默认值，则选中默认值--%>
			ele.val(defaultVal);
		}  else if (type_7_1.test(dataType)) {
			<%-- 样式：enum-[n1,n2,n3,n4,n6,n8,n10]-[1,2,3,4, 6, 8, 10]，特殊枚举类型，显示值和下发的值不一致 --%>
			ele = $("<select id='" + eleData["param_id"] + "'></select>");
			var textItemArr = type_7_1.exec(dataType)[1].split(",");
			var valueItemArr = type_7_1.exec(dataType)[2].split(",");
			ele.append($("<option value=''></option>"));
			for (var i = 0; i < textItemArr.length; i++) {
				var itemOption = $("<option value='" + valueItemArr[i].trim() + "'>" + textItemArr[i].trim() + "</option>");
				ele.append(itemOption);
			}
			<%--如果有默认值，则选中默认值--%>
			ele.val(defaultVal);
		} else if (type_3.test(dataType)) {
			<%--样式：string-[n6,n25,n110]或unsignedInt-[6, 15, 25, 50, 75, 100]，字符串枚举型或整型枚举型--%>
			ele = $("<select id='" + eleData["param_id"] + "'></select>");
			var enumItemArr = type_3.exec(dataType)[1].split(",");
			ele.append($("<option value=''></option>"));
			for (var i = 0; i < enumItemArr.length; i++) {
				var itemOption = $("<option value='" + enumItemArr[i].trim() + "'>" + enumItemArr[i].trim() + "</option>");
				ele.append(itemOption);
			}
			<%--如果有默认值，则选中默认值--%>
			ele.val(defaultVal);
		} else if (type_3_1.test(dataType)) {
			
			<%--样式：string-[n6,n25,n110]或unsignedInt-[6, 15, 25, 50, 75, 100]，字符串枚举型或整型枚举型--%>
			ele = $("<select id='" + eleData["param_id"] + "'></select>");
			var enumItemArr = type_3_1.exec(dataType)[1].split(",");
			ele.append($("<option value=''></option>"));
			for (var i = 0; i < enumItemArr.length; i++) {
				var itemOption = $("<option value='" + enumItemArr[i].trim() + "'>" + enumItemArr[i].trim() + "</option>");
				ele.append(itemOption);
			}
			<%--如果有默认值，则选中默认值--%>
			ele.val(defaultVal);
		} else if ("unsignedInt" == dataType || "int" == dataType || "unsignedLong" == dataType) {
			<%--样式：unsignedInt或int，无符号整型，没有其他限制条件--%>
			var ele_String = "<input id='" + eleData["param_id"] + "' value='" + defaultVal + "'";
			if (dataType.indexOf("unsigned") != -1) {
				<%--unsignedInt和int的唯一区别就是unsignedInt是不能小于零的--%>
				ele_String += " min_value='0'";
			}
			ele_String += " >";
			ele = $(ele_String);
			ele.bind("blur", validateMaxAndMinVal);
			var errSpan_String = "<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>int";
			if (dataType.indexOf("unsigned") != -1) {
				errSpan_String += ";minValue:0;";
			}
			errSpan_String += "</div>";
			errSpan = $(errSpan_String);
		} else if (type_4.test(dataType)) {
			<%--样式：unsignedInt-[1:]或int-[1:]，最小值1--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal + "' min_value='"
			+ type_4.exec(dataType)[1] + "'>");
			ele.bind("blur", trimValue);
			ele.bind("blur", validateMaxAndMinVal);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>"/* minValue:"
			+ type_4.exec(dataType)[1] + */+dataType+ "</div>");
		} else if (type_5.test(dataType)) {
			<%--样式：unsignedInt-[1000:65535]或int-[1000:65535]，最小值1000,最大值65535--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal
			+ "' min_value='" + type_5.exec(dataType)[1]
			+ "' max_value='" +  + type_5.exec(dataType)[2]  + "'>");
			ele.bind("blur", trimValue);
		<%--  if(eleData["mib_dn"] == "LTE_CELL_ECI"){
				ele.bind("blur",validateECI);
				ECITitle['normal'] = dataType;
				ECITitle['special'] = "<%=rb.getString("CellIdZiPeiZhi")%>";
			}else{
				ele.bind("blur", validateMaxAndMinVal);
			} --%>
			ele.bind("blur", validateMaxAndMinVal);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>"+dataType+"</div>");
		} else if (type_5_1.test(dataType)) {
			<%--样式：unsignedInt-[1000:65535]或int-[1000:65535]，最小值1000,最大值65535--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal
			+ "' min_value='" + type_5_1.exec(dataType)[1]
			+ "' max_value='" +  + type_5_1.exec(dataType)[2]  + "'>");
			ele.bind("blur", trimValue);
		<%--  if(eleData["mib_dn"] == "LTE_CELL_ECI"){
				ele.bind("blur",validateECI);
				ECITitle['normal'] = dataType;
				ECITitle['special'] = "<%=rb.getString("CellIdZiPeiZhi")%>";
			}else{
				ele.bind("blur", validateMaxAndMinVal);
			} --%>
			ele.bind("blur", validateMaxAndMinVal);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>"+dataType+"</div>");
		} else if (type_6.test(dataType)) {
			<%--样式：unsignedInt-[:65535]或int-[:65535]--%>
			var ele_String = "<input id='" + eleData["param_id"] + "' value='" + defaultVal
				+ "' max_value='" +  + type_6.exec(dataType)[1]  + "'";
			if (dataType.index("unsigned") != -1) {
				ele_String += " min_value='0' ";
			}
			ele_String += ">";
			ele = $(ele_String);
			ele.bind("blur", trimValue);
			ele.bind("blur", validateMaxAndMinVal);
			var errSpan_String = "<div id='" + eleData["param_id"] + "_err' class='errSpan' style='display:none;margin-top:5px;margin-left:200px;'>"/* int;";
			if (dataType.index("unsigned") != -1) {
				errSpan_String += "minValue:'0';";
			}
			errSpan_String += "maxValue:" + type_6.exec(dataType)[1] + */+dataType+ "</div>";
			errSpan = $(errSpan_String);
		} else if ("bool" == dataType) {
			ele = $("<select id='" + eleData["param_id"] + "'></select>");
			ele.append("<option value=''></option>");
			ele.append("<option value='1'>TRUE</option>");
			ele.append("<option value='0'>FALSE</option>");
		} else if ("dateTime" == dataType) {
			ele = $("<input id='" + eleData["param_id"] + "' class='easyui-datetimebox' data-options='editable:false,width:200'>");
		} else if (dataType.indexOf("List") >= 0 || dataType.indexOf("struct") >= 0) {
			ele = $("<input id='" + eleData["param_id"] + "' value='" + defaultVal + "' title='" + dataType + "'>");
			ele.bind("blur", trimValue);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>"+dataType+"</div>");
			ele.bind("blur", validateMaxAndMinLength);
			
			if (hardware_version_gnb =="QC3.1" || hardware_version_gnb =="QC4.2" || hardware_version_gnb=="QC4.2RELAY" || hardware_version_gnb=="QC4.2T"){
				if (eleData["mib_dn"] == "LTE_PHY_CELLID_LIST" || eleData["mib_dn"] == "LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST") {
					ele.bind("blur", validateRangeNumber);
				}
			}
		}
		
		if(eleData["js_regex"] != null && eleData["js_regex"] != ""){
			ele.attr("js_regex",eleData["js_regex"]);
			ele.bind("blur",validateSpecialChar);
		}
		
		if (ele) {
			ele.bind("blur", createMMLGnb);
			//ele.attr("name", "_" + eleData["param_id"]);
			ele.attr("name", eleData["mib_dn"]);//用mib_dn值作为元素name
			<%--添加dynamic属性，标识参数配置后是否是重启生效--%>
			ele.attr("dynamic", eleData["dynamic"]);
			ele.addClass("border border-box");
			
			if(eleData["title"] != null && eleData["title"] != ""){
				if(errSpan){
					errSpan.html(eleData["title"]);
				}
			}
			
			<%-- 判断是否为必填项，并添加验证事件 --%>
			if (must == "1") {
				<%-- 表示为必填项，新建一个错误信息 --%>
				if (errSpan) {
					errSpan.append('<%=rb.getString("DouHao")%><%=rb.getString("BiTian")%>');
				} else {
					<%-- 没有其它限制条件，添加错误信息、验证事件 --%>
					errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'><%=rb.getString("BiTian")%></div>");
					ele.bind("blur", validateRequired);
				}
				errSpan.show();
				ele.attr("must", "1");
			}
			
			$li.append($label);
			$li.append(ele);
			if (errSpan) {
				$li.append(errSpan);
				ele.attr("title", errSpan.text());
			}
			
			if(eleData["title"] != null && eleData["title"] != ""){
				ele.attr("title",eleData["title"]);
			}
			
			return $li;
		} else {
			return undefined;
		}
	}
	/**
	* 校验IPv4 多个以逗号分割
	* @param e{event}: 事件对象
	**/
	function validateMultIPV4Address_gnb(e){
		var ele = $(e["target"]);
		//如果在之前的操作中已经报错，则不再进行这一步判断
		/* if($("#"+ele.attr("id")+"_err").hasClass("redColor")){
			return ;
		} */
		var mustVal = ele.attr("must") == undefined ? "" : ele.attr("must");
		//如果不是必填项，且输入的值为空，则不做下面的验证
		if(mustVal != "1" && ele.val().trim() == ""){
			return ;
		}
		
		var val = ele.val().trim(),
			isValid = true,
			ipList = val?val.split(','):[];

		ipList.map(function(item){
			if(!isIPv4(item)) isValid = false;
		})

		if(isValid){
			//$("#" + ele.attr("id") + "_err").hide();
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
			ele.removeClass("err_border");
		}else{
			//$("#" + ele.attr("id") + "_err").show();
			$("#" + ele.attr("id") + "_err").addClass('redColor');
			ele.addClass("err_border");
		}
	}
	/**
	* 校验IPv4
	* @param e{event}: 事件对象
	**/
	function validateIPV4Address_gnb(e){
		var ele = $(e["target"]);
		//如果在之前的操作中已经报错，则不再进行这一步判断
		/* if($("#"+ele.attr("id")+"_err").hasClass("redColor")){
			return ;
		} */
		var mustVal = ele.attr("must") == undefined ? "" : ele.attr("must");
		//如果不是必填项，且输入的值为空，则不做下面的验证
		if(mustVal != "1" && ele.val().trim() == ""){
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
			ele.removeClass("err_border");
			return ;
		}
		
		var val = ele.val().trim();
		if(isIPv4(val)){
			//$("#" + ele.attr("id") + "_err").hide();
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
			ele.removeClass("err_border");
		}else{
			//$("#" + ele.attr("id") + "_err").show();
			$("#" + ele.attr("id") + "_err").addClass('redColor');
			ele.addClass("err_border");
		}
	}
	/**
	* 校验IPv6
	* @param e{event}: 事件对象
	**/
	function validateIPV6Address_gnb(e){
		var ele = $(e["target"]);
		//如果在之前的操作中已经报错，则不再进行这一步判断
		/* if($("#"+ele.attr("id")+"_err").hasClass("redColor")){
			return ;
		} */
		var mustVal = ele.attr("must") == undefined ? "" : ele.attr("must");
		//如果不是必填项，且输入的值为空，则不做下面的验证
		if(mustVal != "1" && ele.val().trim() == ""){
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
			ele.removeClass("err_border");
			return ;
		}
		
		var val = ele.val().trim();
		if(isIPv6(val)){
			//$("#" + ele.attr("id") + "_err").hide();
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
			ele.removeClass("err_border");
		}else{
			//$("#" + ele.attr("id") + "_err").show();
			$("#" + ele.attr("id") + "_err").addClass('redColor');
			ele.addClass("err_border");
		}
	}
	/**
	* 校验特殊字符
	* @param e{event}: 事件对象
	**/
	function validateSpecialChar(e){
		var ele = $(e["target"]);
		//如果在之前的操作中已经报错，则不再进行这一步判断
		/* if($("#"+ele.attr("id")+"_err").hasClass("redColor")){
			return ;
		} */
		var mustVal = ele.attr("must") == undefined ? "" : ele.attr("must");
		//如果不是必填项，且输入的值为空，则不做下面的验证
		if(mustVal != "1" && ele.val().trim() == ""){
			return ;
		}
		var js_regex;
		if(ele.attr("js_regex") == "no_zh"){
			/*  验证不允许输入中文，由于下面的正则表达式在用eval转换时出错，因此做特殊处理 **/
			js_regex = /^(?:(?![\u4E00-\u9FA5]|[\uFE30-\uFFA0]).)+$/;
		}else{
			js_regex = eval("(" + ele.attr("js_regex") + ")");
		}
		
		var val = ele.val().trim();
		if(js_regex.test(val)){
			//$("#" + ele.attr("id") + "_err").hide();
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
			ele.removeClass("err_border");
		}else{
			//$("#" + ele.attr("id") + "_err").show();
			$("#" + ele.attr("id") + "_err").addClass('redColor');
			ele.addClass("err_border");
		}
	}
	/*输入的命令赋值给表单*/
	function isLoadJSPGnb(){
		var mmlVar = $("#inputMML_gnb").val();
		var varArr = mmlVar.split(",");//拆分输入命令
		for(var i = 0;i<varArr.length;i++){//循环处理输入的命令
			var eleArr = varArr[i].split("=");//拆分元素，每个元素的格式应该为key=val
			if(eleArr.length>1){//拆分后长度大于1，说明key和val值都有
				//赋值
				$("[name='"+eleArr[0]+"']").val(eleArr[1]);
			}
		}
	}
	/*生成MML命令*/
	function createMMLGnb(){
		var allMibDn = "";
		var imsibind = "";
		//如果验证不通过，则不会生成脚本
		if($(this).hasClass("err_border")){
			return;
		}
		var operName = $("#operGroupTree_gnb").attr("operName");
		var mmlVar = operName;
		
		var selects = $("#cellParam_gnb #paramNodesUl select");
		var inputs = $("#cellParam_gnb #paramNodesUl input:not(.root-param)");
		
		for(var i=0;i<selects.length;i++){
			if($(selects[i]).attr("id")!= "numCells"){
				var midDnName = $(selects[i]).attr("name");
				var parentDisplayCss = $(selects[i]).parent().css("display");
				//当设置halob开关为关时，不需要下发HalobMode这个参数
				if (midDnName == "LTE_HALOB_MODE" && parentDisplayCss == "none" || parentDisplayCss == "none") {
					continue;
				} else {
					allMibDn = allMibDn + $(selects[i]).attr("name")+",";
				}
			}
		}
		
		for(var i = 0;i<inputs.length;i++){
			var midDnNameInputs = $(inputs[i]).attr("name");
			var parentDisplayCssIn = $(inputs[i]).parent().css("display");
			//
			if ( parentDisplayCssIn == "none") {
				continue;
			}else{
				allMibDn = allMibDn + $(inputs[i]).attr("name") +",";
			}
			
		}
		
		allMibDn = allMibDn.substring(0,allMibDn.length-1);
		
		//页面所有元素的name
		var mibdnary = allMibDn.split(",");
		var isAllEmpty = true;
		var isIPBegin = true;
		var j = 1;
		for(var i=0; i < mibdnary.length; i++){
			//获取元素的值
			var elVar = ($("[name='"+mibdnary[i]+"']",$("#cellParam_gnb #paramNodesUl")).val()||'').trim();
			if(mibdnary[i] == "IPSEC_LEFTINTERFACE"){
				text = $("#IPSEC_LEFTINTERFACE_name option:selected").text();
				if(!$("[name='"+mibdnary[i]+"']").hasClass("err_border")){
					if(text != ""){
						isAllEmpty = false;
						if(mmlVar != operName){
							mmlVar =  mmlVar+","+mibdnary[i]+"={"+elVar+"}";
						}else{
							mmlVar += ":" + mibdnary[i]+"={"+elVar + "}";
						}
					}
				}
			}else{
				if(elVar != undefined && elVar != ""){
					isAllEmpty = false;
					if(!$("[name='"+mibdnary[i]+"']").hasClass("err_border")){//如果有元素有不符合要求，则不允许生成脚本
						
						var lgwname = "LGW_IMSI" + j;
						var lgwnameIp = "LGW_IMSI_IP" + j;
						if(mmlVar != operName){
							if(mibdnary[i] == lgwname ){
								if(isIPBegin){
									imsibind = "LTE_LGW_IMSI_IP_LIST={" + elVar + "+"; 
									isIPBegin = false;
								}else{
									imsibind = imsibind + elVar + "+";
								}
								
							}else if(mibdnary[i] == lgwnameIp){
								imsibind = imsibind + elVar + ",";
								j++;
							}else{
								mmlVar =  mmlVar+","+mibdnary[i]+"={"+elVar+"}";
							}
							
						}else{
							mmlVar += ":" + mibdnary[i]+"={"+elVar + "}";
						}
						
							
					}
				}
			}
			
		}
			
			if(imsibind){
				imsibind = imsibind.substring(0,imsibind.length-1);
				imsibind = imsibind +"}";
				mmlVar = mmlVar + "," + imsibind ;
			}
			
		
		
		if (!isAllEmpty) {
			$("#inputMML_gnb").val(mmlVar + ";");
		} else {
			$("#inputMML_gnb").val(operName);
		}
	}
	// 提交之前，验证是否用户的所有输入都已合法，返回true表示没有错误，否则表示有错误存在
	function validateErrBeforeSubmitGnb() {
		$('#showParamValues_gnb input').blur();
		var allowSubmit = true;
		//var errLenth = $("#operValueForm_gnb .errSpan:visible").length;
		var errLenth = $("#operValueForm_gnb .errSpan.redColor").length;
		var errBorder = $("#operValueForm_gnb .err_border").length;
		if (errLenth > 0 && errBorder > 0) {
		<%--用户输入存在不合法的情况--%>
			allowSubmit = false;
			$("#operValueForm_gnb .err_border").fadeOut().fadeIn();
		}
		return allowSubmit;
	}
	/**
	* 点击操作集之后，进行该操作
	* @param node{dom}: 命令树节点
	**/
	function doActionByOperIDGnb(node){
		var nodeList = node.id.split("_");
		var operID = nodeList[0];
		var actionType = nodeList[1],
			cellNumber = node.cellNumber;
		cellIndexLocation = node.cellIndexLocation;
		
		if("LST" == actionType){
			actionType = "v_lst";
		}else if("MOD" == actionType){
			actionType = "v_mod";
		}else if("ADD" == actionType){
			actionType = "v_add";
		}else if("RMV" == actionType){
			actionType = "v_rmv";
		}
		var reg = /.*\((.*)\)/g;
		var operName = reg.exec(node.text)[1];
		$("#operGroupTree_gnb").attr("paramGroupId", operID);
		$("#operGroupTree_gnb").attr("operType", actionType);
		$("#operGroupTree_gnb").attr("operName", operName);
		$("#inputMML_gnb").val(operName);
		$("#textOperName_gnb").combobox("setValue", node.id);
		$("#textOperName_gnb").combo("setText", operName);
		doActionByOperIDAndTypeGnb(operID, actionType,operName,cellNumber);
		gnbmmlVue.init(operID, actionType,operName,cellNumber)
	}
	// 输入框回车事件
	function operInputEnterGnb() {
		var opers = $("#textOperName_gnb").combo("getText");
		var allNodes = $("#operGroupTree_gnb").tree("getChildren");
		var reg = /.*\((.*)\)/;
		for (var num = 0; num < allNodes.length; num++) {
			if (reg.test(allNodes[num].text.toUpperCase())) {
				var node_text = allNodes[num].text.toUpperCase();
				var nodeOper = reg.exec(node_text)[1];
				if (nodeOper == opers.toUpperCase()) {
					// 找到该节点，选中
					$("#operGroupTree_gnb").tree("select", allNodes[num].target);
					break;
				}
			}
		}
	}
	/**
	* 选中命令下拉后联动选中命令树对应节点
	* @param data{object}：选中项
	**/
	function clickRowTableSearchOperNameGnb(data) {
		// 将选择的内容上屏到输入框
		var oper_name = data["oper_name"];
	
		// 重置查询条件，载入全量树
		$('#groupQueryText_gnb').val('');
		$("#operGroupTree_gnb").tree("doFilter", '');
		
		/*<%-- 选中左侧操作集的节点 --%>*/
		var operNode = $("#operGroupTree_gnb").tree("find", data["oper_id"]);
		// 先打开其父节点
		var parentNode = $("#operGroupTree_gnb").tree("getParent", operNode["target"]);
		$("#operGroupTree_gnb").tree("expand", parentNode["target"]);
		$("#operGroupTree_gnb").tree("select", operNode["target"]);
	}
	// 执行命令
	function clickGoGnb() {
		if (!validateErrBeforeSubmitGnb()) {
			return;
		} 
		
		var ctner = $('#cellParam_gnb');
		var smallCells = "";
		var cellsText = "";
		var serial_numbers = "";
		
		var paramGroupId = $("#operGroupTree_gnb").attr("paramGroupId");
		
		var mmlstr = $("#inputMML_gnb").val();
		
		if (mmlstr == "" || mmlstr == undefined) {
			showMsg('prompt_msg',QingShuRuJiaoBen);
			return;
		}
		
		var operName;
		var operIndex = "";//记录要操作小区的index(绝大多数参数需要区分小区，有index；有少数参数不区分小区，index为null)
		if (mmlstr.indexOf(":") != -1) {// MOD CELL: XXX=xxx;
			operName = mmlstr.substring(0, mmlstr.indexOf(":"));
		} else {// LST CELL
			operName = mmlstr;
		}
		if($("#checkboxForIndex", ctner).length>0){
			var checkArray = $("#checkboxForIndex input[type='checkbox']", ctner);
			for (var i = 0; i < checkArray.length; i++) {
				if (checkArray[i].checked == true) {
					operIndex += checkArray[i].value + ",";
				}
			}
			operIndex = operIndex.substring(0, operIndex.length - 1);
		}
		if($("#numCells", ctner).length>0){
			operIndex = $("#numCells", ctner).val();
		}
	
		var ping_reg = /^ping .*/;
		if (ping_reg.test(mmlstr)) {
			goExecuteCmdGnb(mmlstr);
			return;
		}
		
		<%--获取选择的小站--%>
		var selCells = $("#gridCell_cellParam_gnb").datagrid("getSelections");
		if (0 == selCells.length) {
			showMsg('prompt_msg',QingXuanZeSheBei);
			return;
		}
		
		if(hardware_version_gnb == "CA2.0" || hardware_version_gnb == "436Q_CA1.0" || hardware_version_gnb == "NEU430_CA1.0"){
			if (operName.indexOf("NTP") == -1 && operName.indexOf("SYNC") == -1 && operName.indexOf("IPSEC") == -1 && operName.indexOf("SAS") == -1
					&& operName.indexOf("REBOOT") == -1 && operName.indexOf("RESET") == -1 && operName.indexOf("ACT") == -1 && operName.indexOf("DEACT") == -1
						&&operName.indexOf("HTTP") == -1 && operName.indexOf("POWER_AMP") == -1 && operName.indexOf("LMT_LOGIN") == -1) {
				if (operIndex == null || operIndex == "") {
					if( $("#checkboxForIndex, #numCells").length ) {// 含小区时才开启校验
						showMsg('prompt_msg','<%=rb.getString("QingXuanZeXiaoQuBianHao")%>');
						return;
					}
				}else if(operIndex == '1' && $("input[name='LTE_FREQ_BAND_INDICATOR']").val() == '43'){ 
					showMsg('prompt_msg','<%=rb.getString("PinDuanZhiShiTiShi")%>');
				}
			}
		}
		
		//特殊处理，打开halob开关时必须设置MODE
		if (mmlstr.indexOf("LTE_HALOB_ENABLE_STATE={1}") > -1) {
			var halobModeValue = $("[name='LTE_HALOB_MODE']").val().trim();
			if (halobModeValue == undefined || halobModeValue == ""){
				//如果验证不通过，则不会生成脚本
				$("#LTE_HALOB_MODE_name_err").show();
				$("#LTE_HALOB_MODE_name_err").addClass("err_border");
				return;
			}
		}
		<%-- 去掉空格 --%>
		var inputs = $("#cellParam_gnb #paramNodesUl input"); 
		$.each(inputs,function(index,value){
			$(value).val($(value).val().trim());
		});
		
		gnbmmlVue.resultType = 'res';
		gnbmmlVue.isResult = true;
		
				
		var textOperName = $("#textOperName_gnb").combo('getText');
		if("MOD MME" == textOperName){
			modMMEPoolGnb(selCells,mmlstr,hardware_version_gnb);
			return;
		}
		
		<%--拼接多个小站编码--%>
		for (var codeNum = 0; codeNum < selCells.length; codeNum++) {
			smallCells += selCells[codeNum]["small_cell_code"] + ",";
		// cellsText += selCells[codeNum]["host_name"] + ",";
			serial_numbers += selCells[codeNum]["serial_number"]+",";
		}

		smallCells = smallCells.substring(0, smallCells.length - 1);
		serial_numbers = serial_numbers.substring(0,serial_numbers.length-1);
		<%--表单序列化--%>
		var paramJson = $('#operValueForm_gnb').serializeJson();
		
		<%-- 某个 指标若为空，则表示不对该指标进行设置，从对象中去掉该指标 --%>
		var deleteKeyArr = [];
		
		var param = {};
		param["smallCells"] = smallCells;
		param["serial_numbers"] = serial_numbers;
		param["inputMML"] = encodeURIComponent(mmlstr);
		param["hardwareVersion"] = hardware_version_gnb;
		param["cellIndex"] = operIndex;
		
		if (cellIndexLocation) {
			param["cellIndexLocation"] = cellIndexLocation;
		}
		
		$("#cellParam_gnb #paramNodesUl input.root-param").each(function(idx, item){
			if(item.name) param[item.name] = item.value;
		})
		
		updateActionHistoryContentGnb(serial_numbers, operName,mmlstr);
		
		$.post("${ctx}/cell/param/checkIsNeedReboot.action", param, function (data) {
			if (data["success"]) {
				$('.success_msg').html(CiPeiZhiChongQiHouShengXiao);
				var width = $('.success_msg').width();
				$('.success_msg').css("left",'50%');
				var left = parseFloat($('.success_msg').css("left")) - width/2;
				$('.success_msg').css("left",left+'px');
				$('.success_msg').animate({top:'55px'},200,function(){
					setTimeout(function(){
						$('.success_msg').animate({top:'-40px'},function(){
							var specialValidate = specialVaildate(mmlstr);
							if(specialValidate != ""){
								showMsg('prompt_msg',specialValidate);
								return;
							}
							
							$.post("${ctx}/cell/param/operParamGroupValues.action", param, function (data) {}, "json");
						})
					},3000)
				})
			} else {
				var specialValidate = specialVaildate(mmlstr);
				if(specialValidate != ""){
					showMsg('prompt_msg',specialValidate);
					return;
				}
				
				$.post("${ctx}/cell/param/operParamGroupValues.action", param, function (data) {
					if (data["success"]) {
						
					} else {
						
					}
				}, "json");
			}
		},"json");
	}
	/**
	* 特殊处理某些参数
	* @param mmlStr{string}：命令
	**/
	function specialVaildate(mmlStr){
		var operName = mmlStr.substring(0, mmlStr.indexOf(":"));
		if(operName != null && operName.trim() == "MOD PCI_RANGE"){
			var startPCI = $("#cellParam_gnb #paramNodesUl input[name='LTE_SMALLCELL_START_PCI']").val();
			var pciRange = $("#cellParam_gnb #paramNodesUl input[name='LTE_SMALLCELL_PCI_RANGE']").val();
			var total = parseInt(startPCI) + parseInt(pciRange);
			if(total > 503){
				return "The sum of LTE_SMALLCELL_START_PCI and LTE_SMALLCELL_PCI_RANGE must be less than 503";
			}
		}
		
		return "";	
	}
	/**
	* 执行command命令
	* @param mmlStr{string}：命令
	*/
	function goExecuteCmdGnb(mmlstr) {
		var cmd = mmlstr;
		// 结果显示面板
		var divEle = $("<div class='getSetParamValResult cmd_result'></div>");
		divEle.append($("<span class='resultTitle'>" + cmd + ":</span><br/>"));
		showAtPromptOrPanel(divEle);
		
		if(!isShowRightClickMenu) {
		isShowRightClickMenu = true;
		}  
		
		$.post("${ctx}/runtime/executeCommand.action", {cmd: cmd}, function(data) {
			var line = $("<span style='white-space:pre'>" + data.result + "<span><br/>");
			divEle.append(line);
			/*将滚动条滚到最下方*/
			$("#showParamValues_gnb").animate({scrollTop: $("#showParamValues_gnb")[0].scrollHeight + 'px'}, 500);
			
			if(!isShowRightClickMenu){
				/* 配置结果右键事件 */
				isShowRightClickMenu = true;
			}
		}, "json");
	}
	/**
	* 停止正在执行的命令行命令
	*/
	function stopExecute() {
		$.post("${ctx}/runtime/stopExecute.action", {}, function(data) {
			if(data["success"]) {
				showMsg('success_msg',data["message"]);
			}
		}, "json");
	}
	/**
	* 更新到结果记录
	* @param content{string}: 结果内容
	**/
	function updateResultInfoGnb(content) {
		var resultDiv = $('#showParamValues_gnb'),
			last = resultDiv.children(':last');
		resultDiv.append(content);
	}
	/**
	* 更新动作履历记录内容
	* @param serial_numbers{string}: 基站序列号
	* @param operName{string}: 命令名
	* @param mmlstr{string}: 要执行的命令
	**/
	function updateActionHistoryContentGnb(serial_numbers, operName,mmlstr) {
		var date = dateformatter(new Date(gloableTime));//operName + ": " + smallCells
		var $li = $(("<li style='padding:5px 0;'></li>")).append($("<span>" + date + " " + mmlstr + "{" + serial_numbers + "}" + "</span><br/>"));
		var defaultWindowObj = $("#winDefault");

		defaultWindowObj.append($li);
		defaultWindowObj.animate({scrollTop: defaultWindowObj[0].scrollHeight + 'px'}, 500);
		
		var strContent = "<div class='command-tips'>" + date + " " + mmlstr + "{" + serial_numbers + "}" + "</div>";
		// 更新到结果记录方便知晓指令状况
		updateResultInfoGnb(strContent);
	}
	/**
	 * 预处理树数据：递归标记每个自定义节点的 _isOwn 属性
	 * Private Template 下：子分组名 === user_code 的分组及其下属叶子节点标记为 true
	 * Public Template 下：所有叶子节点标记为 true（公共指令所有人可操作）
	 * @param {Array} nodes 树节点数组
	 * @param {string|null} parentId 父节点ID
	 * @param {boolean|null} inheritOwn 从父级继承的归属标记
	 */
	function _markOwnershipGnb(nodes, parentId, inheritOwn) {
		if(!nodes || !nodes.length) return;
		for(var i = 0; i < nodes.length; i++) {
			var n = nodes[i];
			if(n.customized != 'true') {
				// 非自定义节点，递归子节点
				if(n.children) _markOwnershipGnb(n.children, n.id, null);
				continue;
			}
			if(parentId === 'PublicTemplateId') {
				// Public Template 下的所有节点：所有人可操作
				n._isOwn = true;
				if(n.children) _markOwnershipGnb(n.children, n.id, true);
			} else if(parentId === 'PrivateTemplateId') {
				// Private Template 的直接子节点 = 用户分组，判断组名是否匹配当前用户
				var own = (n.text === user_code);
				n._isOwn = own;
				if(n.children) _markOwnershipGnb(n.children, n.id, own);
			} else if(inheritOwn !== null) {
				// 已在某个用户分组内部，继承父级归属
				n._isOwn = inheritOwn;
				if(n.children) _markOwnershipGnb(n.children, n.id, inheritOwn);
			} else {
				// 其他层级（如 Customized 根节点、PrivateTemplateId/PublicTemplateId 本身）
				if(n.children) _markOwnershipGnb(n.children, n.id, null);
			}
		}
	}
	/** 
	* 选择的小站的软件版本发生了变化 ，newSoftwareVer:变化后的软件版本
	* @param newSoftwareVer{string}: 软件版本
	**/
	function hardwareVersionChangeGnb(newSoftwareVer) {
		$("#inputMML_gnb").val("");
		
		// 1. 刷新操作树
		$.post("${ctx}/cell/param/getOperGroupTree.action", { hardware_version: newSoftwareVer}, function(data) {
			_markOwnershipGnb(data, null, null);
			$("#operGroupTree_gnb").tree("loadData", data);
			if(data && data.length){
				<%-- 操作名称输入框，初始化下拉面板 --%>
				$.post("${ctx}/cell/param/getOperName4SearchGrid.action", {hardware_version: newSoftwareVer}, function(data) {
					$("#textOperName_gnb").combobox("loadData", data);
				},"json");
			}else {
				$("#textOperName_gnb").combobox("loadData", []);
			}
		},"json");
		
		// 2. 清空操作表单、操作输入框
		$("#cellParam_gnb #paramNodesUl li").remove();
		$("#cellParam_gnb #paramNodesUl div").remove();
		$("#textOperName_gnb").combo("clear");
	}
	/** 
	* 加载前事件-基站列表
	* @param param{object}: 查询参数
	**/
	function beforeLoad_gridCell_cellParam_gnb(param) {
		param["hardware_version"] = hardware_version_gnb;
		//查找当前选中的
		if(choosedGroupId > 0){
			param["group_id"] = choosedGroupId;
		}
	}
	// 事件处理-数据表格加载成功
	function gridCellParamDatagridLoadSuccessGnb() {
		$(this).datagrid("fixRownumber");
		$(this).datagrid("enableContextmenuAutoSize");
		var currData = $(this).datagrid("getData");
	}
	
	//添加一个参数路径输入框-用于LST和RMV方法
	function addParampathLSTInputText_elfcellGnb(e) {
		//添加新的输入框
		var $namePathDiv = $("<div class='gnbMmlItemDiv' style='margin-top: 5px'></div>");
		var $namePathText= $('<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span>');
		var $namePathInput = $("<input type='text' placeholder='<%=rb.getString("JiaoYanGuiZe")%>' name='gnbParamPathNameForElfcellLSTConfig' class='showWholeVal border border-box item'></input>");
		var $namePathButtn = $("<a onclick='rmvParampathLSTInputText_elfcellGnb(this)' class='operationSub'>" 
				+ "<span class='img-suffix el-icon el-icon-minus'></span></a>");
		var $namePathPrompt = $("<p class='prompt'></p>");
		
		$namePathDiv.append($namePathText);
		$namePathDiv.append($namePathInput);
		$namePathDiv.append($namePathButtn);
		$namePathDiv.append($namePathPrompt);
		
		var parentDiv = $("#paramForElfcellLSTDiv_gnb");
		$namePathDiv.appendTo(parentDiv);
		
		//将当前输入框后面的图标改为删除图标，并重新绑定事件
		$(e).children(".img-suffix").removeClass('el-icon-minus').addClass('el-icon-plus');
		$(e).attr("onclick", "addParampathLSTInputText_elfcellGnb(this)");
	}
	//删除LST下的参数路径和值输入框
	function rmvParampathLSTInputText_elfcellGnb(e) {
		$(e).parent("div").remove();
	}
	//添加一个参数路径输入框-用于MOD和ADD方法
	function addParampathMODInputText_elfcellGnb(e, field) {
		var $namePathDiv = $("<div class='gnbMmlItemDiv' style='margin-top: 5px'></div>");
		var $namePathText= $('<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span>');
		var $namePathInput = $("<input type='text' placeholder='<%=rb.getString("JiaoYanGuiZe")%>' name='gnbParamPathNameForElfcellMODConfig' class='showWholeVal border border-box item'></input>");
		var $paramValueText=$('<span style="margin:0 10px 0 19px;"><%=rb.getString("CanShuZhi")%></span>');
		var $paramValueInput = $("<input type='text' name='gnbParamPathValueForElfcellMODConfig' class='showWholeVal border border-box item'></input>");
		var $namePathButtn = $("<a onclick='rmvParampathMODInputText_elfcellGnb(this)' class='operationSub'>" 
				+ "<span class='img-suffix el-icon el-icon-minus'></span></a>");
		var $namePathPrompt = $("<p class='prompt'></p>");
		
		$namePathDiv.append($namePathText);
		$namePathDiv.append($namePathInput);
		$namePathDiv.append($paramValueText);
		$namePathDiv.append($paramValueInput);
		$namePathDiv.append($namePathButtn);
		$namePathDiv.append($namePathPrompt);
		
		var parentDiv = $("#paramForElfcellADDAndMODDiv_gnb");
		$namePathDiv.appendTo(parentDiv);
		
		//将当前输入框后面的图标改为删除图标，并重新绑定事件
		$(e).children(".img-suffix").removeClass('el-icon-minus').addClass('el-icon-plus');
		$(e).attr("onclick", "addParampathMODInputText_elfcellGnb(this)");
		
		if(field) {
			$namePathInput.val(field.namePath);
			$paramValueInput.val(field.value);
		}
	}
	//删除MOD下的参数路径和值输入框
	function rmvParampathMODInputText_elfcellGnb(e) {
		$(e).parent("div").remove();
	}
	//适用于俄罗斯高通平台
	function clickRussiaElfcellGoGnb() {
		var checkResultMsg = validateRussiaElfcellErrBeforeSubmitGnb();
		if (checkResultMsg.length > 0) {
			showMsg('prompt_msg',checkResultMsg);
			return;
		} 
		var smallCells = "";
		var serial_numbers = "";
		// 获取选择的小站
		var selCells = $("#gridCell_cellParam_gnb").datagrid("getSelections");
		if (0 == selCells.length) {
			showMsg('prompt_msg',QingXuanZeSheBei);
			return;
		}

		gnbmmlVue.resultType = 'res';
		gnbmmlVue.isResult = true;

		// 拼接多个小站编码
		for (var codeNum = 0; codeNum < selCells.length; codeNum++) {
			smallCells += selCells[codeNum]["small_cell_code"] + ",";
			serial_numbers += selCells[codeNum]["serial_number"]+",";
		}
		
		smallCells = smallCells.substring(0, smallCells.length - 1);
		serial_numbers = serial_numbers.substring(0,serial_numbers.length-1);
		
		var objParentObjArray = new Array();
		var paramArray = new Array();
		
		var operName=$("#elfcellConfigParamOperType_gnb").val();
		if ("MOD" == operName || "ADD" == operName) {
			var $namePathEle = $("input[name=gnbParamPathNameForElfcellMODConfig]");
			var $valueList = $("input[name=gnbParamPathValueForElfcellMODConfig]");
			
			if("ADD" == operName) {
				$namePathEle = $("input[name=gnbParamPathNameForElfcellLSTConfig]");
			}

			var namePathLength = $namePathEle.length;
			for (var i = 0; i< namePathLength; i++) {
				var obj = new Object();
				var namePath = $namePathEle[i].value;
				var pathValue = $valueList[i]?$valueList[i].value:'';
				// 去掉空格
				
				if("ADD" == operName) {
					obj.namePath = namePath.trim();
				}else {
					// 去掉空格
					obj.namePath = namePath.trim();
					obj.value = pathValue.trim();
				}

				paramArray.push(obj);
			}
		} else {
			var $namePathEle;
			if ("RMV" == operName) {
				$namePathEle = $("input[name=gnbParamPathNameForElfcellRMVConfig]");
			} else {
				$namePathEle = $("input[name=gnbParamPathNameForElfcellLSTConfig]");
			}
			var namePathLength = $namePathEle.length;
			
			for (var i = 0; i< namePathLength; i++) {
				var obj = new Object();
				var namePath = $namePathEle[i].value;
				obj.namePath = namePath.trim();
				paramArray.push(obj);
			}
		}
		
		var mmlstr = JSON.stringify(paramArray);
		var param = {};
		param["smallCells"] = smallCells;
		param["serial_numbers"] = serial_numbers;
		param["inputMML"] = mmlstr;
		param["hardwareVersion"] = hardware_version_gnb;
		param["operName"] = operName;

		updateActionHistoryContentGnb(serial_numbers, operName, mmlstr);
		
		//如果是add object验证参数路径是否只包含一个对象，路径是否合法
		if ("ADD" == operName) {
			$.post("${ctx}/cell/param/checkAddObjectPathRussiaElfcell.action", param, function (data) {
				if (data["success"]) {
					$.post("${ctx}/cell/param/operRussiaElfcellParamValues.action", param, function (data) {}, "json");
				} else {
					showMsg('error_msg',data["msg"]);
					return;
				}
			}, "json");
		} else {
			$.post("${ctx}/cell/param/operRussiaElfcellParamValues.action", param, function (data) {}, "json");
		}
	}

	// 自定义参数保存
	function saveCustomParamCommandGnb(commandName, commandId, nodeName, isPublic) {
		var checkResultMsg = validateRussiaElfcellErrBeforeSubmitGnb();
		if (checkResultMsg.length > 0) {
			showMsg('prompt_msg',checkResultMsg);
			return;
		} 
		
		var objParentObjArray = new Array();
		var paramArray = new Array();
		
		var operName=$("#elfcellConfigParamOperType_gnb").val();
		if ("MOD" == operName || "ADD" == operName) {
			var $namePathEle = $("input[name=gnbParamPathNameForElfcellMODConfig]");
			var $valueList = $("input[name=gnbParamPathValueForElfcellMODConfig]");
			
			if("ADD" == operName) {
				$namePathEle = $("input[name=gnbParamPathNameForElfcellLSTConfig]");
			}

			var namePathLength = $namePathEle.length;
			for (var i = 0; i< namePathLength; i++) {
				var obj = new Object();
				var namePath = $namePathEle[i].value;
				var pathValue = $valueList[i].value;
				// 去掉空格
				
				if("ADD" == operName) {
					obj.namePath = namePath.trim();
				}else {
					// 去掉空格
					obj.namePath = namePath.trim();
					obj.value = pathValue.trim();
				}

				paramArray.push(obj);
			}
		} else {
			var $namePathEle;
			if ("RMV" == operName) {
				$namePathEle = $("input[name=gnbParamPathNameForElfcellRMVConfig]");
			} else {
				$namePathEle = $("input[name=gnbParamPathNameForElfcellLSTConfig]");
			}
			var namePathLength = $namePathEle.length;
			
			for (var i = 0; i< namePathLength; i++) {
				var obj = new Object();
				var namePath = $namePathEle[i].value;
				obj.namePath = namePath.trim();
				paramArray.push(obj);
			}
		}
		
		var mmlstr = JSON.stringify(paramArray);
		var param = {};
		param["inputMML"] = mmlstr;
		param["hardwareVersion"] = hardware_version_gnb;
		param["operType"] = operName;
		param["commandName"] = commandName;
		param["groupId"] = commandId;
		param["nodeName"] = nodeName;
		param["isPublic"] = isPublic;
		
		$.post("${ctx}/cell/param/saveCustomizedMML.action", param, function (data) {
			if(data.success == true) {
				gnbmmlVue.customDlShow = false;
				gnbmmlVue.oldCommandName = gnbmmlVue.commandForm.commandName;
				gnbmmlVue.$message({
					message: '<%=rb.getString("TiShiChengGong")%>',
					type: 'success'
				});

				// 如果是从临时节点保存的，先删除临时节点
				if(gnbmmlVue.commandForm.tempNodeId) {
					var tree = $("#operGroupTree_gnb");
					var tempNode = tree.tree('find', gnbmmlVue.commandForm.tempNodeId);
					if(tempNode) {
						tree.tree('remove', tempNode.target);
					}
					gnbmmlVue.commandForm.tempNodeId = null;
				}

				// 保存成功后的节点ID：优先使用后端返回的ID（新建场景），否则使用当前编辑的ID
				var savedGroupId = data.flag || commandId;

				// 刷新指令树（复用初始化加载逻辑，包含测试数据和 _markOwnershipGnb）
				hardwareVersionChangeGnb(hardware_version_gnb);

				// 刷新后默认选中保存的指令
				if(savedGroupId) {
					var tree = $("#operGroupTree_gnb");
					var savedNode = tree.tree('find', savedGroupId);
					if(savedNode) {
						// 展开父节点链
						var parentNode = tree.tree('getParent', savedNode.target);
						while(parentNode) {
							tree.tree('expand', parentNode.target);
							parentNode = tree.tree('getParent', parentNode.target);
						}
						tree.tree('select', savedNode.target);
						// 触发节点点击，回显参数
						reviewCustomParamConmmandGnb(savedNode);
					}
				}
			}else {
				gnbmmlVue.$message.error(data["message"]);
			}
		}, "json");

	}
	
	function resetCommandParamGnb() {
		var tabNav = $('span[tabtit="gnbCellParamDetail"]');

		turnTabs(tabNav);

		$('#elfcellConfigParamOperType_gnb').trigger('change');
		Object.assign(gnbmmlVue.commandForm, {
			isPublic: '0',
			commandName: '',
			groupId: '',
			nodeName: ''
		});
		gnbmmlVue.oldCommandName = '';
	}
	// 自定义参数回显
	function reviewCustomParamConmmandGnb(node) {
		// 如果是 Untitled 指令，设置为编辑模式，显示保存按钮；否则为查看模式，隐藏保存按钮
		if(node.text && node.text.startsWith('Untitled')) {
			isEditModeGnb = true;
		} else {
			isEditModeGnb = false;
		}
		updateSaveButtonVisibilityGnb();
		
		// 检查是否为临时节点
		if(node.id && node.id.startsWith('temp_')) {
			// 临时节点，不需要从后台查询数据
			var tabNav = $('span[tabtit="gnbElfcellConfigParamPanel"]');
			turnTabs(tabNav);
			$('#elfcellConfigParamOperType_gnb').trigger('change');
			
			Object.assign(gnbmmlVue.commandForm, {
				commandName: node.text,
				groupId: '',
				nodeName: '',
				tempNodeId: node.id
			});
			gnbmmlVue.oldCommandName = '';
			return;
		}
		
		var params = {
				groupId: node.id
			};

		Object.assign(gnbmmlVue.commandForm, {
			commandName: node.text,
			groupId: node.id,
			nodeName: ''
		});
		gnbmmlVue.oldCommandName = node.text;

		$.post("${ctx}/cell/param/getCustomizedMMLFields.action", params, function (data) {
			var fieldList = data.pathList || [],
				operType = data.operType || 'LST';

			if(['LST', 'ADD', 'MOD', 'RMV'].includes(operType)) {
				$("#elfcellConfigParamOperType_gnb").val(operType).trigger('change');
			}else {
				$("#elfcellConfigParamOperType_gnb").val('LST').trigger('change');
			}

			fieldList.map(function(field, index){
				
				if(index == 0) {// 第一组数值映射
					//只留一个输入框 lst or add
					if(['LST', 'ADD'].includes(operType)) {
						var ctner = $("#paramForElfcellLSTDiv_gnb");
						$("div:gt(0)", ctner).remove();
						$("div:first input", ctner).val(field.namePath);
					}

					// mod
					if(operType == 'MOD') {
						var ctner = $("#paramForElfcellADDAndMODDiv_gnb");
						$("div:gt(0)", ctner).remove();
						$("div:first input:first", ctner).val(field.namePath);
						$("div:first input:last", ctner).val(field.value);
					}
					// rmv
					if(operType == 'RMV') {
						var ctner = $("#paramForElfcellRMVDiv_gnb");
						$("div:first input:first", ctner).val(field.namePath);
					}
				}else {// 其他组数值映射
					if(['LST', 'ADD'].includes(operType)) {
						var el = $('#paramForElfcellLSTDiv_gnb .gnbMmlItemDiv:last a');
						addParampathLSTInputText_elfcellGnb(el, field);
					}

					// mod
					if(operType == 'MOD') {
						var el = $('#paramForElfcellADDAndMODDiv_gnb .gnbMmlItemDiv:last a');
						addParampathMODInputText_elfcellGnb(el, field);
					}
				}
			});
		}, "json");
	}
	// 
	function toADDCommandGnb(nodeName, evt) {
		var tree = $("#operGroupTree_gnb");
		
		// 查找父节点（nodeName对应的节点，如 Private Template）
		var parentNode = findParentNodeByNameGnb(tree, nodeName);
		if(!parentNode) {
			gnbmmlVue.$message.error('Parent node not found');
			return;
		}
		
		// 获取父节点的子节点，找到当前登录用户对应的子组作为实际的目标节点
		var parentChildren = tree.tree('getChildren', parentNode.target);
		var targetNode = parentNode; // 默认目标节点为父节点本身
		
		// 遍历子节点，查找与当前登录用户匹配的用户分组
		if(parentChildren && parentChildren.length > 0) {
			var foundUserGroup = false;
			for(var i = 0; i < parentChildren.length; i++) {
				var child = parentChildren[i];
				var isFolder = !tree.tree('isLeaf', child.target);
				if(isFolder && child.text === user_code) {
					// 找到当前用户对应的分组
					targetNode = child;
					foundUserGroup = true;
					break;
				}
			}
			// 如果没找到当前用户的分组，检查第一个子节点是否是分组（兼容只有一个分组的情况）
			if(!foundUserGroup) {
				var firstChild = parentChildren[0];
				var isFolder = !tree.tree('isLeaf', firstChild.target);
				if(isFolder) {
					targetNode = firstChild;
				}
			}
		}
		
		// 检查目标节点下是否已经存在 'Untitled' 临时节点
		var children = tree.tree('getChildren', targetNode.target);
		var existingTempNode = null;
		if(children && children.length > 0) {
			for(var i = 0; i < children.length; i++) {
				var child = children[i];
				// 检查是否为临时节点且名称为 'Untitled'
				if(child.id && child.id.startsWith('temp_') && child.text === 'Untitled') {
					existingTempNode = child;
					break;
				}
			}
		}
		
		evt.stopPropagation();

		// 如果已存在，直接选中该节点
		if(existingTempNode) {
			// 先展开父节点和目标节点
			tree.tree('expand', parentNode.target);
			if(targetNode !== parentNode) {
				tree.tree('expand', targetNode.target);
			}
			
			tree.tree('select', existingTempNode.target);
			
			// 切换到配置面板
			var tabNav = $('span[tabtit="gnbElfcellConfigParamPanel"]');
			turnTabs(tabNav);
			$('#elfcellConfigParamOperType_gnb').trigger('change');
			
			// 设置表单数据
			Object.assign(gnbmmlVue.commandForm, {
				commandName: existingTempNode.text,
				groupId: '',
				nodeName: targetNode.text,
				tempNodeId: existingTempNode.id
			});
			gnbmmlVue.oldCommandName = '';
			return;
		}
		
		// 不存在才创建新的临时节点
		// 生成临时节点ID（使用特殊前缀标识临时节点）
		var tempNodeId = 'temp_' + new Date().getTime();
		
		// 创建临时节点数据
		var tempNode = {
			id: tempNodeId,
			text: 'Untitled',
			customized: 'true',
			isTemp: true  // 标记为临时节点
		};
		
		// 在目标节点下添加临时节点到最前面
		var targetChildren = tree.tree('getChildren', targetNode.target);
		if(targetChildren && targetChildren.length > 0) {
			// 有子节点时，插入到第一个子节点前面
			tree.tree('insert', {
				before: targetChildren[0].target,
				data: tempNode
			});
		} else {
			// 没有子节点时，直接append
			tree.tree('append', {
				parent: targetNode.target,
				data: [tempNode]
			});
		}
		
		// 自动展开父节点和目标节点（两级展开）
		tree.tree('expand', parentNode.target);
		if(targetNode !== parentNode) {
			tree.tree('expand', targetNode.target);
		}
		
		// 查找刚添加的节点并选中
		var newNode = tree.tree('find', tempNodeId);
		if(newNode) {
			tree.tree('select', newNode.target);
			
			// 切换到配置面板，准备编辑
			var tabNav = $('span[tabtit="gnbElfcellConfigParamPanel"]');
			turnTabs(tabNav);
			$('#elfcellConfigParamOperType_gnb').trigger('change');
			
			// 设置表单数据
			Object.assign(gnbmmlVue.commandForm, {
				commandName: 'Untitled',
				groupId: '', // 空表示新建
				nodeName: targetNode.text, // 使用实际目标节点的名称
				tempNodeId: tempNodeId // 记录临时节点ID，保存时用于删除
			});
			gnbmmlVue.oldCommandName = '';
		}
	}
	
	// 辅助函数：根据节点名称查找父节点
	function findParentNodeByNameGnb(tree, nodeName) {
		var allNodes = tree.tree('getRoots');
		return searchNodeByNameGnb(tree, allNodes, nodeName);
	}
	
	function searchNodeByNameGnb(tree, nodes, nodeName) {
		for(var i = 0; i < nodes.length; i++) {
			var node = tree.tree('find', nodes[i].id);
			if(node && node.text == nodeName) {
				return node;
			}
			var children = tree.tree('getChildren', nodes[i].target);
			if(children && children.length > 0) {
				var found = searchNodeByNameGnb(tree, children, nodeName);
				if(found) return found;
			}
		}
		return null;
	}
	function removeCusNodeGnb(el, evt) {
		var nodeId = $(el).attr('nodeid');
		
		// 检查是否为临时节点
		if(nodeId && nodeId.startsWith('temp_')) {
			// 直接删除临时节点，无需调用后台
			var tree = $("#operGroupTree_gnb");
			var tempNode = tree.tree('find', nodeId);
			if(tempNode) {
				tree.tree('remove', tempNode.target);
				gnbmmlVue.$message({
					message: '<%=rb.getString("ChengGong")%>',
					type: 'success'
				});
			}
			evt.stopPropagation();
			return;
		}
		
		var param = {
			groupId: nodeId
		};

		$.messager.confirm({
			width: 400,
			height: 200,
			title: '<%=rb.getString("QueRen")%>',
			msg: '<%=rb.getString("QueDingShanChuRenWu")%>',
			fn:function(r){
				if(r){
					$.post("${ctx}/cell/param/deleteCustomizedMML.action", param, function (data) {
						if(data.success == true) {
							gnbmmlVue.$message({
								message: '<%=rb.getString("TiShiChengGong")%>',
								type:'success'
							});

							hardwareVersionChangeGnb(hardware_version_gnb);

							$('#elfcellConfigParamOperType_gnb').trigger('change');
							Object.assign(gnbmmlVue.commandForm, {
								commandName: '',
								groupId: '',
								nodeName: ''
							});
							gnbmmlVue.oldCommandName = '';
						}else {
							gnbmmlVue.$message.error(data["message"]);
						}
					}, "json");
				}
			}
		}).addClass("normalConfirm reSetConfirm");

		evt.stopPropagation();
	}

	// 全局变量：标记当前是否为编辑模式（点击修改按钮进入）
	var isEditModeGnb = false;

	/**
	 * 复制自定义MML节点
	 * @param el 点击的元素
	 * @param evt 事件对象
	 */
	function copyCusNodeGnb(el, evt) {
		var nodeId = $(el).attr('nodeid');
		var tree = $("#operGroupTree_gnb");
		var node = tree.tree('find', nodeId);
		
		if(!node) {
			evt.stopPropagation();
			return;
		}

		// 临时节点不支持复制
		if(nodeId && nodeId.startsWith('temp_')) {
			gnbmmlVue.$message.warning('Unsaved command cannot be copied');
			evt.stopPropagation();
			return;
		}

		var param = { groupId: nodeId };
		$.post("${ctx}/cell/param/copyCustomizedMML.action", param, function (data) {
			if(data.success == true) {
				gnbmmlVue.$message({
					message: '<%=rb.getString("TiShiChengGong")%>',
					type: 'success'
				});

				// 记录复制后新节点的groupId，用于刷新后选中
				var newGroupId = data.flag;

				// 刷新指令树（复用初始化加载逻辑，包含测试数据和 _markOwnershipGnb）
				hardwareVersionChangeGnb(hardware_version_gnb);

				// 刷新后默认选中复制的节点
				if(newGroupId) {
					var newNode = tree.tree('find', newGroupId);
					if(newNode) {
						// 展开父节点链
						var parentNode = tree.tree('getParent', newNode.target);
						while(parentNode) {
							tree.tree('expand', parentNode.target);
							parentNode = tree.tree('getParent', parentNode.target);
						}
						tree.tree('select', newNode.target);
						// 触发节点点击，回显参数
						reviewCustomParamConmmandGnb(newNode);
					}
				}
			} else {
				gnbmmlVue.$message.error(data.message || 'Copy failed');
			}
		}, "json");

		evt.stopPropagation();
	}

	/* ==================== copyCusNodeGnb 旧逻辑（已注释） ====================
	function copyCusNodeGnb(el, evt) {
		var nodeId = $(el).attr('nodeid');
		var tree = $("#operGroupTree_gnb");
		var node = tree.tree('find', nodeId);
		
		if(node) {
			// 切换到参数面板
			var tabNav = $('span[tabtit="gnbElfcellConfigParamPanel"]');
			turnTabs(tabNav);
			
			// 设置为非编辑模式，隐藏保存按钮
			isEditModeGnb = false;
			updateSaveButtonVisibilityGnb();
			
			// 复制时重置commandForm，清空groupId表示新建
			Object.assign(gnbmmlVue.commandForm, {
				commandName: node.text + '_copy',
				groupId: '', // 清空groupId表示新建
				nodeName: ''
			});
			gnbmmlVue.oldCommandName = '';
			
			// 如果是临时节点，直接回显数据
			if(nodeId && nodeId.startsWith('temp_')) {
				$('#elfcellConfigParamOperType_gnb').trigger('change');
				evt.stopPropagation();
				return;
			}
			
			// 从后台获取数据并回显
			var params = { groupId: nodeId };
			$.post("${ctx}/cell/param/getCustomizedMMLFields.action", params, function (data) {
				var fieldList = data.pathList || [],
					operType = data.operType || 'LST';

				if(['LST', 'ADD', 'MOD', 'RMV'].includes(operType)) {
					$("#elfcellConfigParamOperType_gnb").val(operType).trigger('change');
				}else {
					$("#elfcellConfigParamOperType_gnb").val('LST').trigger('change');
				}

				// 回显字段数据
				fieldList.map(function(field, index){
					if(index == 0) {
						if(['LST', 'ADD'].includes(operType)) {
							var ctner = $("#paramForElfcellLSTDiv_gnb");
							$("div:gt(0)", ctner).remove();
							$("div:first input", ctner).val(field.namePath);
						}
						if(operType == 'MOD') {
							var ctner = $("#paramForElfcellADDAndMODDiv_gnb");
							$("div:gt(0)", ctner).remove();
							$("div:first input:first", ctner).val(field.namePath);
							$("div:first input:last", ctner).val(field.value);
						}
						if(operType == 'RMV') {
							var ctner = $("#paramForElfcellRMVDiv_gnb");
							$("div:first input:first", ctner).val(field.namePath);
						}
					}else {
						if(['LST', 'ADD'].includes(operType)) {
							var el = $('#paramForElfcellLSTDiv_gnb .gnbMmlItemDiv:first a');
							addParampathLSTInputText_elfcellGnb(el, field);
						}
						if(operType == 'MOD') {
							var el = $('#paramForElfcellADDAndMODDiv_gnb .gnbMmlItemDiv:first a');
							addParampathMODInputText_elfcellGnb(el, field);
						}
					}
				});
			}, "json");
		}
		
		evt.stopPropagation();
	}
	==================== copyCusNodeGnb 旧逻辑结束 ==================== */

	/**
	 * 修改自定义MML节点
	 * @param el 点击的元素
	 * @param evt 事件对象
	 */
	function editCusNodeGnb(el, evt) {
		var nodeId = $(el).attr('nodeid');
		var tree = $("#operGroupTree_gnb");
		var node = tree.tree('find', nodeId);
		
		if(node) {
			// 切换到参数面板
			var tabNav = $('span[tabtit="gnbElfcellConfigParamPanel"]');
			turnTabs(tabNav);
			
			// 设置为编辑模式，显示保存按钮
			isEditModeGnb = true;
			updateSaveButtonVisibilityGnb();
			
			// 设置commandForm
			Object.assign(gnbmmlVue.commandForm, {
				commandName: node.text,
				groupId: node.id,
				nodeName: ''
			});
			gnbmmlVue.oldCommandName = node.text;
			
			// 如果是临时节点，直接回显数据
			if(nodeId && nodeId.startsWith('temp_')) {
				$('#elfcellConfigParamOperType_gnb').trigger('change');
				evt.stopPropagation();
				return;
			}
			
			// 从后台获取数据并回显
			var params = { groupId: nodeId };
			$.post("${ctx}/cell/param/getCustomizedMMLFields.action", params, function (data) {
				var fieldList = data.pathList || [],
					operType = data.operType || 'LST';

				if(['LST', 'ADD', 'MOD', 'RMV'].includes(operType)) {
					$("#elfcellConfigParamOperType_gnb").val(operType).trigger('change');
				}else {
					$("#elfcellConfigParamOperType_gnb").val('LST').trigger('change');
				}

				// 回显字段数据
				fieldList.map(function(field, index){
					if(index == 0) {
						if(['LST', 'ADD'].includes(operType)) {
							var ctner = $("#paramForElfcellLSTDiv_gnb");
							$("div:gt(0)", ctner).remove();
							$("div:first input", ctner).val(field.namePath);
						}
						if(operType == 'MOD') {
							var ctner = $("#paramForElfcellADDAndMODDiv_gnb");
							$("div:gt(0)", ctner).remove();
							$("div:first input:first", ctner).val(field.namePath);
							$("div:first input:last", ctner).val(field.value);
						}
						if(operType == 'RMV') {
							var ctner = $("#paramForElfcellRMVDiv_gnb");
							$("div:first input:first", ctner).val(field.namePath);
						}
					}else {
						if(['LST', 'ADD'].includes(operType)) {
							var el = $('#paramForElfcellLSTDiv_gnb .gnbMmlItemDiv:first a');
							addParampathLSTInputText_elfcellGnb(el, field);
						}
						if(operType == 'MOD') {
							var el = $('#paramForElfcellADDAndMODDiv_gnb .gnbMmlItemDiv:first a');
							addParampathMODInputText_elfcellGnb(el, field);
						}
					}
				});
			}, "json");
		}
		
		evt.stopPropagation();
	}

	/**
	 * 更新保存按钮的显示状态
	 * 编辑模式显示，查看模式隐藏
	 */
	function updateSaveButtonVisibilityGnb() {
		var $saveBtn = $('#customMmlSaveBtnGnb');
		if(isEditModeGnb) {
			$saveBtn.show();
		} else {
			$saveBtn.hide();
		}
	}

	//LST,RMV 操作 参数路径不能为空，不能包含{i}，必须包含.，且RMV操作 参数路径必须以.结尾
	function validLstAndRmvNamePath(e){
		var ele = $(e["target"]);
		var currVal = ele.val();
		
		var operType = $("#elfcellConfigParamOperType_gnb").val();

		if (currVal.length == 0) {
			$(ele).css("borderColor","red");
			$(ele).siblings("p.prompt").text("NamePath cannot be empty!");
		} else if (currVal.indexOf('{i}') > -1) {
			$(ele).css("borderColor","red");
			$(ele).siblings("p.prompt").text("NamePath cannot contains {i}!");
		} else if (currVal.indexOf(".") == -1) {
			$(ele).css("borderColor","red");
			$(ele).siblings("p.prompt").text("NamePath must contains .!");
		} else{
			$(ele).css("borderColor","#C9D1D6");
			$(ele).siblings("p.prompt").text("");
		}
		
		if ("RMV" == operType) {
			var lastChar = currVal.substring(currVal.length-1, currVal.length);
			if (lastChar != ".") {
				$(ele).siblings("p.prompt").text("NamePath must end with .!");
				$(ele).css("borderColor","red");
			} else {
				$(ele).css("borderColor","#C9D1D6");
			}
		}
	}
	// 提交之前，验证是否用户的所有输入都已合法，返回true表示没有错误，否则表示有错误存在，用于俄罗斯高通版
	function validateRussiaElfcellErrBeforeSubmitGnb() {
		var operType = $("#elfcellConfigParamOperType_gnb").val();
		var msg = "";
		var allowSubmit = true;
		if ("MOD" == operType) {
			var $namePathEle = $("input[name=gnbParamPathNameForElfcellMODConfig]");
			var $valueList = $("input[name=gnbParamPathValueForElfcellMODConfig]");
			
			var namePathLength = $namePathEle.length;
			for (var i = 0; i< namePathLength; i++) {
				var namePath = $namePathEle[i].value;
				var pathValue = $valueList[i].value;
				if (namePath.length == 0 || pathValue.length == 0) {
					msg = "NamePath and value cannot be empty!";
					allowSubmit = false;
				} else if (namePath.indexOf('{i}') > -1) {
					msg = "NamePath cannot contains {i}!";
					allowSubmit = false;
				} else if (namePath.indexOf(".") == -1) {
					msg = "NamePath must contains .!"
					allowSubmit = false;
				} else if (namePath.substring(namePath.length-1, namePath.length) == '.') {
					msg = "NamePath cannot end with .!"
					allowSubmit = false;
				} 
				if (!allowSubmit) {
					break;
				}
			}
		} else {
			var $namePathEle = $("input[name=gnbParamPathNameForElfcellLSTConfig]");
			if ("RMV" == operType) {
				$namePathEle = $("input[name=gnbParamPathNameForElfcellRMVConfig]");
			}
			var namePathLength = $namePathEle.length;
			for (var i = 0; i< namePathLength; i++) {
				var namePath = $namePathEle[i].value;
				if (namePath.length == 0) {
					msg = "NamePath cannot be empty!";
					allowSubmit = false;
				} else if (namePath.indexOf('{i}') > -1) {
					msg = "NamePath cannot contains {i}!";
					allowSubmit = false;
				} else if (namePath.indexOf(".") == -1) {
					msg = "NamePath must contains .!"
					allowSubmit = false;
				} 
				
				if ("RMV" == operType || "ADD" == operType) {
					var lastChar = namePath.substring(namePath.length-1, namePath.length);
					if (lastChar != ".") {
						msg = "NamePath must end with .!";
						allowSubmit = false;
					}
				}
				
				if (!allowSubmit) {
					break;
				}
			}
		} 
		return msg;
	}
	/** 
	* 创建小区选择下拉
	* @param eleData{string}: 基站编码
	**/
	function createCellIndex(eleData) {
		var totalIndex = eleData;
		var selectId = "numCells";
		var $li = $("<li></li>");
		var $label = $("<label>" + "<%=rb.getString("XiaoQuXuanZe")%>" + ":</label>");
		var ele = $("<select id='" + selectId +"'></select>");
		ele.append($("<option value=''></option>"));
		for (var i = 1; i <= totalIndex; i++) {
			var itemOption = $("<option value='" + i + "'>" + "<%=rb.getString("XiaoQuBianHao")%>" + " " + i + "</option>");
			ele.append(itemOption);
		}
		ele.addClass("border border-box");
		$li.append($label);
		$li.append(ele);
		return $li;
	}
	/** 
	* 创建小区复选框式选项
	* @param eleData{string}: 基站编码
	**/
	function createCheckBoxIndex(eleData) {
		var totalIndex = eleData;
		var checkbox = "checkbox";
		var $div = $("<div id='checkboxForIndex'></div>");
		var $label;
		var ele;
		for (var i = 1; i <= totalIndex; i++) {
			ele = $("<input value='" + i + "' type='checkbox' style='width:20px; float:left; margin: 6px 0 5px 0'></input>");
			$label = $("<label style='margin-bottom:5px'>" + "<%=rb.getString("XiaoQuBianHao")%>" + i + "</label>");
			$div.append(ele);
			$div.append($label);
		}
		return $div;
	}
	// 过滤查询命令树
	function filterMMLTreeGnb(){
		var treeCtn = $('#operGroupTree_gnb');
		mmlCommandTreeSearchTxt = $('#groupQueryText_gnb').val().trim();
		//hardwareVersionChangeGnb(hardware_version_gnb);
		treeCtn.tree('doFilter',mmlCommandTreeSearchTxt);
	}
	// 初始化拖拽
	var gnbVLine = document.querySelectorAll('.vertical-line-gnb'),
		gnbHLine = document.querySelector('.horizontal-line-gnb');

	Array.from(gnbVLine).map(function(line){
	gnbAddListener(line,"mousedown",onmousedownHGNB);
	});
	gnbAddListener(gnbHLine,"mousedown",onmousedownVGNB);
	// 根据事件对象获取事件触发源对象
	function getTarget(evt){
		return evt.target || evt.srcElement;
	}
	/**
	* 绑定事件方法
	* @param element{dom}: 要绑定事件的对象
	* @param type{string}: 事件类型
	* @param listener{function}: 事件响应的方法
	* @param useCapture{boolean}: 是否在捕获阶段触发
	**/
	function gnbAddListener(element,type,listener,useCapture){
		element.addEventListener?element.addEventListener(type,listener,useCapture):element.attachEvent("on" + type,listener);
	}
	/* 鼠标点击事件 */
	function onmousedownHGNB(event) {
		var lastX = event.clientX, d = document,
		preItems = document.querySelectorAll('.flex-prev-item-gnb'),
		sufItems = document.querySelectorAll('.flex-suff-item-gnb');
		d.onmousemove = function(event){
			var evt = event || window.event;
			
			var direction = evt.clientX - lastX;
			
			if(direction > 0) {//right
			sufItems.forEach(function(item){
				setWidth(item,direction)
			})
			preItems.forEach(function(item){
				clearWidth(item)
			})
			} else if(direction < 0){//left
			preItems.forEach(function(item){
				setWidth(item,-direction);
			})
			sufItems.forEach(function(item){
				clearWidth(item)
			})
			}
			lastX = evt.clientX;
		};
		d.onmouseup = function(){
			d.onmousemove = null;
			d.onmouseup = null;
			$('#omc_app_ctn').resize();
		};

		function setWidth(item,offset) {
			var style = getComputedStyle(item),
				width = style.width.replace('px','') - offset;
			// maxWidth、minWidth 防止flex布局对宽度计算的影响
			if(width < 325) width = 325;
			item.style.width = width + 'px';
			item.style.maxWidth = width + 'px';
			item.style.minWidth = width + 'px';
		}
		function clearWidth(item) {
			item.style.width = '';
			item.style.maxWidth = '';
			item.style.minWidth = '';
		}
	}
	/* 鼠标点击事件 */
	function onmousedownVGNB(event) {
		var _self = this;
		var lastY = event.clientY, d = document,
		rowItems = document.querySelectorAll('.flex-row-item-gnb'),
		preItem = rowItems[0],
		sufItem = rowItems[1];
		_self.clickDown = true;
		d.onmousemove = function(event){
			var evt = event || window.event;
			if(_self.clickDown == true) {
				var direction = evt.clientY - lastY;
				if(direction > 0) {//up
				setHeight(sufItem,direction);
				clearHeight(preItem);
				} else if(direction < 0){//down
				setHeight(preItem,-direction);
				clearHeight(sufItem);
				}
				lastY = evt.clientY;
			}
		};
		d.onmouseup = function(){
			d.onmousemove = null;
			d.onmouseup = null;
			_self.clickDown = false;
			$('#omc_app_ctn').resize();
		};

		function setHeight(item,offset) {
			var style = getComputedStyle(item),
				height = style.height.replace('px','') - offset;
			// maxHeight、minHeight 防止flex布局对高度计算的影响
			if(height < 200) height = 200;
			item.style.height = height + 'px';
			item.style.maxHeight = height + 'px';
			item.style.minHeight = height + 'px';
		}
		function clearHeight(item) {
			item.style.height = '100%';
			item.style.maxHeight = '';
			item.style.minHeight = '';
		}
	}
	/**
	* 修改MME池
	* @param sels{string}: 选中的设备标识
	* @param mmlstr{string}: 命令
	* @param hardware_version_gnb{string}: 版本
	**/
	function modMMEPoolGnb(sels,mmlstr,hardware_version_gnb){
		
		if(sels.length == 1){
			doExecuteGnb(sels,mmlstr,hardware_version_gnb);
			return;
		}
		//当选中的站至少为2个时，触发下面操作
		if(sels.length >= 2){
			//选择第一个站的mme_enable作为基准
			var mmeEnableZero = sels[0].mme_enable;
			
			if(mmeEnableZero == null || mmeEnableZero.toString().trim() == ""){
				mmeEnableZero = 0;
			}
			var needPrompt = false;
			var oneMMEPoolEnb = new Array();
			
			for(var i = 0; i < sels.length; i ++){
				
				var mmeEnable = sels[i].mme_enable;
				if("1" != mmeEnable){
					oneMMEPoolEnb.push(sels[i].serial_number);
				}
				
				if(mmeEnable == null || mmeEnable.toString().trim() == ""){
					mmeEnable = 0;        		}
				
				if(mmeEnable != mmeEnableZero){
					needPrompt = true;
				}
			}
			
			if(needPrompt){
				var sns = "</br>";
				for(var i = 0;i < oneMMEPoolEnb.length;i ++){
					sns = sns + oneMMEPoolEnb[i]+",";
				}
				
				sns = sns.substring(0,sns.length-1);
				
				$.messager.confirm({
					width:450,
					height:220,
					title:'<%=rb.getString("QueRen")%>',
					msg:'<%=rb.getString("MMEPoolConfigConfirm")%>' + sns,
					fn:function(r){
						if(r){
							doExecuteGnb(sels,mmlstr,hardware_version_gnb);
						}
					}
				}).addClass("normalConfirm reSetConfirm");
			}
		}
	}
	/**
	* 执行命令
	* @param sels{string}: 选中的设备标识
	* @param mmlstr{string}: 命令
	* @param hardware_version_gnb{string}: 版本
	**/
	function doExecuteGnb(selCells,mmlstr,hardware_version_gnb){
		
		var smallCells = "";
		var cellsText = "";
		var serial_numbers = "";
		
		<%--拼接多个小站编码--%>
		for (var codeNum = 0; codeNum < selCells.length; codeNum++) {
			smallCells += selCells[codeNum]["small_cell_code"] + ",";
			serial_numbers += selCells[codeNum]["serial_number"]+",";
		}

		smallCells = smallCells.substring(0, smallCells.length - 1);
		serial_numbers = serial_numbers.substring(0,serial_numbers.length-1);
		<%--表单序列化--%>
		var paramJson = $('#operValueForm_gnb').serializeJson();
		
		<%-- 某个 指标若为空，则表示不对该指标进行设置，从对象中去掉该指标 --%>
		var deleteKeyArr = [];
	
		var param = {};
		param["smallCells"] = smallCells;
		param["serial_numbers"] = serial_numbers;
		param["inputMML"] = encodeURIComponent(mmlstr);
		param["hardwareVersion"] = hardware_version_gnb;
		
		updateActionHistoryContentGnb(serial_numbers, "MOD MME",mmlstr);
		
		$.post("${pageContext.request.contextPath}/cell/param/operParamGroupValues.action", param, function (data) {
			if (data["success"]) {
				closeDefaultWindow();
			} else {
				
			}
		}, "json");
		
	}
	/**
	* 展示MME池方式
	* @param flag{string}: 新选中标识
	* @param row{object}: 选中行数据
	**/
	function showMMEPoolWayGnb(flag, row) {
		
		var sels = $("#gridCell_cellParam_gnb").datagrid("getSelections");
		var isAllOneMMEPool = true;

		if(row){
			if(flag){
				//新选中
				sels.push(row);
			}else{
				var index = 0;
				for(var i=0; i<sels.length;i++){
					if(sels[i].small_cell_code == row.small_cell_code){
						index = i;
						break;
					}
				}
				sels.splice(index,1);
			}
		}
		
		for (var i = 0; i < sels.length; i++) {
			if (sels[i].mme_enable == "1") {
				isAllOneMMEPool = false;
				break;
			}
		}
		
		if (isAllOneMMEPool && sels.length > 0) {
			$("#oneMMEPool").show();
			$("#twoMMEPool").hide();
			
			$("#LTE_SIGLINK_SERVER_LIST_name").val("");
		}else{
			$("#oneMMEPool").hide();
			$("#twoMMEPool").show();
			$("#LTE_X_BAICELLS_POOL_MME_LIST1_name").val("");
			$("#LTE_X_BAICELLS_POOL_MME_LIST2_name").val("");
		}
		
	}
	// MML操作模板自定义方法
	function customEventFnc5G(){
		var portInput = document.getElementById('60113');
		if(portInput){
			$('#60113').on('blur',function(){
				var protocolVal = $("#60111 option:selected").text(),
					portVal = $("#60113").val();
				if(protocolVal == 'tcp' || protocolVal == 'udp'){
					if(portVal){
						if(isNumeric(portVal)&& parseInt(portVal)>=0 && parseInt(portVal)<=65535){
							$("#60113_err").html('');
							$("#60113").removeClass('err_border');
						}else{
							$("#60113").addClass('err_border');
							$("#60113_err").addClass('redColor');
							$("#60113_err").html('<%=rb.getString("LGWPortTiShi")%>');
						}
					}else{
						$("#60113").addClass('err_border');
						$("#60113_err").addClass('redColor');
						$("#60113_err").html('<%=rb.getString("BiTian")%>');
					}
				}else{
					$("#60113").removeClass('err_border');
					$("#60113_err").removeClass('redColor');
					$("#60113_err").html('<%=rb.getString("LGWPortTiShi")%>');

				}
			})
			
		}

		// ['100389','100390']
		Array.from($('#100389,#100390')).map(function(item) {
			item.addEventListener('blur',function(){
				var val = $(item).val();

				if(val) {
					var fmtVal = val.replaceAll(' ','').split('').reduce(function(n, m){ 
							if(n.length%3 == 2) n = n + ' ';
							return n + m;
						});

					$(item).val(fmtVal);
				}
			}, true);
		})

		// 100507
		$('#100507').on('change', function(evt) {
			var $doms = $('#100488,#100489,#100491'),
				val = $(this).val();

			if(val == 'DHCPv6') {
				$doms.val('').attr('disabled', true);
			}else {
				$doms.attr('disabled', false);
			}
		})

		// 100506
		$('#100506').on('change', function(evt) {
			var $doms = $('#100483,#100484,#100485'),
				val = $(this).val();

			if(val == 'DHCP') {
				$doms.val('').attr('disabled', true);
			}else {
				$doms.attr('disabled', false);
			}
		})
	}
	function isNumeric(str) {
			if(str.length==0){
				return false;
			}
			for(var i=0;i<str.length;i++){
				if(str.charAt(i)<"0" || str.charAt(i)>"9"){
					return false;
				}
			}
			return true;  
	}

	// ========== 5G gNB 批量输入 ==========
	function showBatchDL_gnb() {
		var dl = $('#mml_device_dl_gnb');
		dl.dialog('open');
	}

	function closeBatchDL_gnb() {
		var dl = $('#mml_device_dl_gnb');
		$('#batch_sn_textarea_gnb').val('');
		$('#batch_sn_tips_gnb').text('');
		dl.dialog('close');
	}

	function batchInputSN_gnb() {
		var serialNumber = $('#batch_sn_textarea_gnb').val(),
			list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;}),
			tips = $('#batch_sn_tips_gnb');

		var temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/;

		if (serialNumber == null || serialNumber.length == 0) {
			tips.text('<%=rb.getString("SNBuNengWeiKong")%>');
		}else{
			var nameFlag = list.every(function(item,index){
				return temp.test(item)
			})
			if(nameFlag){
				tips.text('');
			}else{
				tips.text('<%=rb.getString("QingShuRuZhengQueSn")%>');
			}
		}

		// 验证通过 执行批量选择
		if(tips.text() == '') {
			var tb = $('#gridCell_cellParam_gnb'),
				existedRows = tb.datagrid('getSelections'),
				existedSns = existedRows.map(function(row){
					return row.serial_number;
				});

			$.ajax({
				url: '${ctx}/cell/param/getMmlSelectedCellList.action',
				type: 'post',
				data: {
					serialNumbers: list.join(','),
					productType: hardware_version_gnb,
					isGnb: 1
				},
				dataType: 'json',
				success: function(data){
					var rows = data || [];

					rows.map(function(row){
						if(!existedSns.includes(row.serial_number)) {
							tb.data('datagrid').selectedRows.push(Object.assign({},row));
							tb.data('datagrid').checkedRows.push(Object.assign({},row));
						}
					});

					tb.datagrid('reload');
					refreshEnbNumGnb();
				}
			});

			closeBatchDL_gnb();
		}
	}
</script>