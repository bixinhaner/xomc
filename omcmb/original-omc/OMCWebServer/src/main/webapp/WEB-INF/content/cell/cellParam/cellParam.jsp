<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style type="text/css">
	.length-adaptation {
		width: 60% !important;
	}
	#inputMML {
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

	#paramNodesUl label{
		margin-top:5px;
		word-break:break-all;
	}
	.eNBHighQuery{
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
	.elfcellConfigParamPanel .enbMmlItemDiv{
		height:46px;
	}
	.elfcellConfigParamPanel .enbMmlItemDiv .prompt{
		height:20px;
		line-height:20px;
		color:red;
	}
	/* 参数面板 - flex布局：操作区可滚动，按钮固定底部 */
	.enb-panel-flex-wrapper {
		display: flex;
		flex-direction: column;
		position: absolute;
		top: 0;
		bottom: 0;
		left: 0;
		right: 0;
	}
	.enb-panel-flex-wrapper #elfcellShowParamValues {
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
	.enb-panel-bottom-bar {
		flex-shrink: 0;
		padding: 8px 20px;
		border-top: 1px solid #EDEDED;
		background: #fff;
		text-align: left;
	}
	.item{
		width:330px;
	}

	.enb-list-ctn {
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
	#showParamValues:empty + .paramValuesTips{
		display: flex;
		top: 40%;
	}

	.flex-row-item {
		display: flex;
		flex: 1 1 100%;
	}
	.flex-prev-item {
		flex: 1 1 37%;
	}
	.flex-suff-item {
		flex: 1 1 62%;
	}
	.horizontal-line {
		cursor: row-resize;
		padding: 10px;
	}
	.vertical-line {
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
	#toolbar_GridCell_cellParam .textbox.combo{
		height:32px !important;
	}
	#toolbar_GridCell_cellParam .textbox-icon.combo-arrow{
		height:33px !important;
	}
	#toolbar_GridCell_cellParam .textbox .textbox-text{
		padding-top:8px !important;
		padding-bottom:8px !important;
	}
	#cellParam .tabsContentDiv , #cellParam .contentDiv {
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

	#commonQueryText {
		width: 200px;
	}

	#cellParam .editButton {
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
		word-break: normal;
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

	#operGroupTree .tree-node {
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
    .mmlDoubleConfirmIconCls::before{
        font-size: 60px;
        color: #FF4614;
    }
    .mml-second-confirm-container {
        margin-bottom: 20px;
    }
    .mml-second-confirm-icon {
        text-align: center;
        margin-bottom: 20px;
    }
    .mml-second-confirm-content {
        text-align: left;
    }
    .mml-second-confirm-item {
        line-height: 22px;
    }
    .mml-second-bullet {
        display: inline-block;
        margin-right: 8px;
    }
    .mml-second-bullet::before {
        content: '';
        display: inline-block;
        width: 6px;
        height: 6px;
        background-color: #333;
        border-radius: 50%;
        vertical-align: middle;
    }
    .mml-second-reset-input-container {
        margin-top: 15px;
    }
    .mml-second-reset-input-label {
        display: block;
        margin-bottom: 8px;
        font-weight: bold;
    }
    .mml-second-reset-input {
        width: 100%;
        padding: 8px;
        border: 1px solid #dcdfe6;
        border-radius: 4px;
        font-size: 14px;
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
<div class="overflow-cls">
	<div class="panelDefault" id="cellParam" style="min-width: 1000px;height: 100%;" >
		<div class="tabsTitle" id="eNb_tabs_area">
			<span tabtit="MMLConfig" onclick="turnTabs(this)" class="active"><%=rb.getString("CanShuPeiZhi")%></span>	
			<span id="eNBMMLScript" tabtit="MMLScript" onclick="turnTabs(this)"><%=rb.getString("MMLJiaoBen")%></span>			
		</div>
		<!-- <div style="height: 5px;background: #E3E7F0;"></div> -->
		<div class="omcTabsTablePage" style="left:0px;right:0px;top: 50px;bottom:0px;background:#fff;">
			<div class="MMLConfig" id="MML_Config" style="display: block;">
				<div style="width: 100%;height: 100%;min-height: 600px;min-width:1000px;" class="flex-ctn">
					<div class="flex-row-item" style="height:47%;">
						<div class="splitPanel cellParamWestNorth flex-prev-item" style="height:100%;width:37%;float:left;margin-left:20px;">
							<div class="singleTitle">
								<%=rb.getString("JiZhanSheBei")%>
								<p style='float:right; margin-right:15px; top:0px;display:none;' class='showSelectNum'>
									<%=rb.getString("YiXuan") %>(<span class="circle-num-tips" style='margin:0 3px;color:#4D84FF'></span>)
									<span class='el-icon el-icon-circle-down' style='font-size:16px;cursor:pointer;border-bottom:none;margin-left:5px;' onclick="slidedownlist()"></span>
								</p>
								<div id="enbMMLBatchInput" class="editButton" onclick="showBatchDL()" style="margin-right: 15px;">
									<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
									<span><%=rb.getString("PiLiangShuRu")%></span>
								</div>
							</div>
							<div class="contentDiv">
								<table id="gridCell_cellParam"></table>
							</div>
							<div class="enb-list-ctn">
								<div class="list-title">
									<%=rb.getString("YiXuanJiZhan")%>
									<span style="float: right;padding-right: 15px;" onclick="slideuplist()"><i class="el-icon el-icon-close"></i></span>
								</div>
								<div style="width: calc(100% - 44px);padding-left: 25px;">
									<div class="list-body-title">
										<%=rb.getString("HostName")%> + <%=rb.getString("XiaoZhanBianMa")%>
										<div class="operationDiv operation_delete" title="Delete All" style="margin-left:5px;float: right;" onclick="delAllDoStgRecord()"></div>
									</div>
								</div>
								<div class="list-body"></div>
							</div>
						</div>
						<div class="vertical-line"></div>
						<div id="enbmmlResult" class="splitPanel flex-suff-item"  style="height:100%;width:62%;float:right;margin-right:20px;">
							<div class="singleTitle">
								{{resultTitle}}
								<div style="float: right;display:flex;margin-right:20px;">
									<el-radio-group v-model="resultType" size="small" @input="changeType">
										<el-radio-button label="res" ><%=rb.getString("PeiZhiJieGuo")%></el-radio-button><el-radio-button label="help"><%=rb.getString("BangZhu")%></el-radio-button>
									</el-radio-group>
									<div v-if="resultType == 'res'" class="param-values-oper">
										<a onclick="menuHandler({name:'clear'})" class="el-icon el-icon-operation-clear" title="<%=rb.getString("QingKong")%>" style="font-size:20px;margin-right:5px;cursor: pointer;"> </a>
										<a onclick="menuHandler({name:'download'})" class="el-icon el-icon-common-download" title="<%=rb.getString("BaoCun")%>" style="font-size:20px;margin:0px 5px;cursor: pointer;"> </a>
									</div>
									<div v-if="resultType == 'help'" class="param-values-oper">
										<a @click="exportEnbMml" class="el-icon el-icon-operation-export" title="<%=rb.getString("DaoChu")%>" style="font-size:20px;margin-right:5px;cursor: pointer;"> </a>
									</div>
									
								</div>
								
							</div>
							<div :class="resultType == 'help' ? 'contentDiv' : 'contentDiv overflow-auto'"  style="right:20px;top:40px;">
								<div v-if="resultType == 'help'" class="helpDiv">
									<div class="help-summary">
										<div style="font-weight: bold;margin-bottom:10px;">{{helpData.param_name}}</div>
										<div>
											<span class="help-item-title"><%=rb.getString("ChanPinLeiXing")%></span><span>{{paramProduct}}</span>
										</div>
									
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
								
								<div v-show="resultType == 'res'" class="contentDiv" id="showParamValues" style="padding-top:10px;overflow:auto;right:20px;top:0px;"></div>
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
					<div class="horizontal-line"> </div>
					<div class="flex-row-item" style="height:50%;margin-bottom:20px;">
						<div class="splitPanel flex-prev-item" id="mmlListCenter" style="height:100%;width:37%;float:left;margin-left:20px;">
							<div class="singleTitle">
								<%=rb.getString("MingLingLieBiao")%>
							</div>
							<div class="queryGroup" style="margin-left:15px;">
								<input id="groupQueryText" name="value" style="margin-left:0px;width:275px;" placeholder="<%=rb.getString("BianMa")%> / <%=rb.getString("MingCheng")%>" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)" value="">
								<b class='el-icon el-icon-common-search' onclick="javascript: filterMMLTree();"></b>
							</div>
							<div class="contentDiv" style="padding-top:20px; top: 70px;">
								<%-- MML 命令集-树形结构 --%>
								<ul id="operGroupTree" class="easyui-tree" data-options="border:false" style="height:calc(100% - 20px);overflow:auto;"></ul>
							</div>
						</div>
						<div class="vertical-line"></div>
						<div class="splitPanel flex-suff-item" id="operatorActor"  style="height:100%;width:62%;float:right;margin-right:20px;">
							<div class="tabsTitle omcLogLists" style='background:none;height:40px;line-height:40px;border-bottom:1px solid #EDEDED;'>				
								<span tabtit="cellParamDetail" onclick="turnTabs(this)" class="active" style='margin-left:20px;'><%=rb.getString("JiaoBenYanShi")%></span>
								<span tabtit="elfcellConfigParamPanel" onclick="turnTabs(this)"><%=rb.getString("CanShuMianBan")%></span>
							</div>
							<div class="tabsContentDiv" style='top:42px;'>
								<div class="contentDiv cellParamDetail" id="showParamValues" style="padding-top:20px;margin-left:20px;">
									<div>
										<div style="margin-bottom: 20px;">
											<textarea id="inputMML" name="inputMML" class="border border-box" oninput="customCommand(this)" style="padding-left:10px;border-style: solid;" placeholder="<%=rb.getString("QingShuRuJiaoBen")%>"></textarea>
										</div>
										<div>
											<select id="textOperName" style="float:left;height:26px;"></select>
										<a class="linkbutton CODE_ENB_MML hidden visible" style='cursor:pointer;margin-left:20px;' onclick="clickGo()"><span><%=rb.getString("GO")%></span></a>
										</div>
									</div>
									<div style="height:59%;overflow:auto;">
										<!-- 自动生成的操作表单 -->
										<div id="operValueForm">
											<ul id="paramNodesUl" class="paramNodesUl"></ul>
										</div>
									</div>
								</div>
								<!-- 参数面板 -->
								<div class="elfcellConfigParamPanel">
								  <div class="enb-panel-flex-wrapper">
									<div class="panelTableDiv" id="elfcellShowParamValues">
										<span><%=rb.getString("CaoZuoLeiXing")%></span>
										<select id="elfcellConfigParamOperType" class="border border-box" style="height:26px;vertical-align:middle;margin-left: 10px">
											<option value="LST" selected>LST</option>
											<option value="MOD" class="CODE_ENB_MML hidden">MOD</option>
											<option value="ADD" class="CODE_ENB_MML hidden">ADD</option> 
											<option value="RMV" class="CODE_ENB_MML hidden">RMV</option>
										</select>
										
										<div style="margin-top: 20px;" id="paramForElfcellLSTDiv">
											<div class="enbMmlItemDiv" style="margin-bottom: 5px">
												<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span>
												<input type="text" name="paramPathNameForElfcellLSTConfig" placeholder='<%=rb.getString("JiaoYanGuiZe")%>' class="length-adaptation showWholeVal border border-box item"/>
												<a onclick="addParampathLSTInputText_elfcell(this)" id="addbtnForLstAndRMVPath" class="operationAdd CODE_ENB_MML hidden visible">
													<span class="img-suffix el-icon el-icon-plus"></span>
												</a>
												<p class="prompt"></p>
											</div>
										</div>
										
										<div style="margin-top: 20px;display: none" id="paramForElfcellADDAndMODDiv">
											<div class="enbMmlItemDiv" style="margin-bottom: 5px">
												<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span>
												<input type="text" name="paramPathNameForElfcellMODConfig" placeholder='<%=rb.getString("JiaoYanGuiZe")%>'  class='showWholeVal border border-box item'/>
												<span style="margin:0 10px 0 15px;"><%=rb.getString("CanShuZhi")%></span>
												<input type="text" name="paramPathValueForElfcellMODConfig"  class='showWholeVal border border-box item' />
												<a onclick="addParampathMODInputText_elfcell(this)" class="operationAdd CODE_ENB_MML hidden">
													<span class="img-suffix el-icon el-icon-plus"></span>
												</a>
												<p class="prompt"></p>
											</div>
										</div>
										
										<div style="margin-top: 20px;display: none" id="paramForElfcellRMVDiv">
											<div class="enbMmlItemDiv" style="margin-bottom: 5px">
												<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span>
												<input type="text" name="paramPathNameForElfcellRMVConfig" placeholder='<%=rb.getString("JiaoYanGuiZe")%>' class="length-adaptation showWholeVal border border-box item"/>
												<p class="prompt"></p>
											</div>
										</div>
										
									</div>
									<!-- 底部固定按钮栏 -->
									<div class="enb-panel-bottom-bar">
										<a id="customMmlSaveBtn" class="linkbutton CODE_ENB_MML hidden" @click="openCommandConfirm" style="margin-left:10px;display:none;"><span><%=rb.getString("BaoCun")%></span></a>
										<a class="linkbutton CODE_ENB_MML hidden visible" onclick="clickRussiaElfcellGo()" style="margin-left:0px;"><span><%=rb.getString("GO")%></span></a>
									</div>
								  </div>
								</div>
							</div>
						</div>
					</div>	
				</div>
			</div>
			<div class="MMLScript" id="MMLScript"></div>
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
					</el-form-item>
				</el-form>
			</div>
			<el-button-group style='margin-top:20px;'>
			   <el-button type='primary' size='small' @click="commandSave"><%=rb.getString("QueDing")%></el-button>
			   <el-button size='small' @click="customDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		   </el-button-group>
		</el-dialog>

		<el-slide
			ref="scriptAdd"
			:title="scriptAddTitle"
			:url="scriptAddUrl"
			:footer="false"
			@cancel="closeTaskAddPage"
		></el-slide>
	</div>
</div>
<%-- 工具栏-基站列表 --%>
<div id="toolbar_GridCell_cellParam" class="admin_query_head toolbarContainer" style="padding:15px 0 20px !important;">
	<div class="defaultQuery" style='margin-left:20px;'>
		<li style="display:inline-block;margin-right:15px;">
			<select id = "softwareVersionCombo_cellParam" name="software_version" class="border border-box software_version" style="width:180px;margin-right:1px;height: 26px;"></select>
		</li>
		<div>
			<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun")%>" 
				name="search_text"
				inputId="commonQueryText"
				targetId="eNBHighQueryDiv"
				placeholder="<%=rb.getString("XiaoZhanBianMaHUOMINGCHENG")%>"
				data-options="query: queryGridCellParam"></div>

            <!-- 高级查询 -->
			<div class="eNBHighQuery" id="eNBHighQueryDiv" style='top:35px;'>
				<ul class="inputslist">				
				 	<li style="margin:20px 20px 0;">
						<select id ="stationConfigDeviceGroup" name="type" class="border border-box" style="width:260px;margin-right:1px;height: 26px;"></select>
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

<div id="mmlTiShi" class="easyui-window" title="<%=rb.getString("TiShi") %>" data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:500,height:300,resizable:false">
    <div class="easyui-layout" data-options="border:false,fit:true">
		 <div id="mmlTiShiContent" region="center"></div>
		 <div region="south" data-options="border:false,height:50" style="padding: 0px 0px 10px 0;">
			<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="closeMMLTiShi()"><%=rb.getString("GuanBi")%></a>
		 </div>
    </div>
</div>

<div id="mml_device_dl" class="easyui-dialog" style="width: 600px;height: 400px;"
	data-options="closed: true, modal: true" 
	title='<%=rb.getString("TianJia") %>'>
	<div style="padding: 20px;">
		<div style="padding-bottom: 10px;font-weight: bold;"><%=rb.getString("XiaoZhanBianMa") %></div>
		<textarea id="batch_sn_textarea" rows="10" cols="80"></textarea>
		<p id="batch_sn_tips" style="color: red;min-height: 18px;"></p>
		
		<div class="tipText">
			<span class="el-icon el-icon-circle-info infoTip"></span>
			<span><%=rb.getString("eNBZhuCeTiShiWenZi")%></span>
		</div>
		<div style="padding-top: 10px;">
			<a class="linkbutton" onclick="batchInputSN()"><span><%=rb.getString("QueDing") %></span></a>
			<a class="linkbutton" onclick="closeBatchDL()"><span><%=rb.getString("QuXiao") %></span></a>
		</div>
	</div>
</div>

<script type="text/javascript">
	var choosedGroupId = -1;
	var isShowRightClickMenu = false;
	var hardware_version = "EA4.0";
	var original_hardware_version = '',
		mmlCommandTreeSearchTxt = '',
		otherDeviceCodes = [];
	var sel_product;

	var enbmmlVue = new Vue({
		el:'#cellParam',
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
					hardwareVersion: '',
					searchText:''
				},
				enbMmlParams:'',
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
				},

				scriptAddUrl: '',
				scriptAddTitle: '<%=rb.getString("TianJia")%>'
			}
		},
		methods:{
			changeType(val){
				var vm = this;
				if(val == 'help'){
					vm.resultTitle = '<%=rb.getString("BangZhu")%>';
					//$("#cellParam .command-tips").hide();
					//$("#cellParam .getSetParamValResult").hide();

				}else{
					vm.resultTitle = '<%=rb.getString("PeiZhiJieGuo")%>';
					//$("#cellParam .command-tips").show();
					//$("#cellParam .getSetParamValResult").show();
				}
			
			},
			init(operID, actionType, operName,cellNumber){	
				var vm = this;
				
				var param = {
					operID : operID,
					actionType : actionType,
					hardwareVersion: hardware_version,
					searchText:''
				};
				Object.assign(vm.paramData,param)
				$.post("${ctx}/cell/param/getParamGroupHelpInfos.action", param, function(data) {
					if (data) {
						vm.helpData = data;
						vm.paramProduct = sel_product;
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
					timeZone: timeZone
				}
				param = Object.assign(param,vm.paramData);
				exportByForm(url,param)
			},
			openCommandConfirm() {
				var vm = this;

				vm.customDlShow = true;
				vm.commandForm.commandName = vm.oldCommandName;
			},
			commandSave() {
				var vm = this;

				vm.$refs.commandform.validate(function(v) {
					if(v) {
						saveCustomParamCommand(vm.commandForm.commandName, vm.commandForm.groupId, vm.commandForm.nodeName, vm.commandForm.isPublic);
					}
				});
			},

			toAddTask(type) {
				var vm = this,
					isView = type == 'view';

				vm.scriptAddUrl = '${ctx}/task/MMLScript/goAddTask.action';
				vm.scriptAddTitle = isView? '<%=rb.getString("XinXi")%>':'<%=rb.getString("TianJia")%>';
				vm.$refs.scriptAdd.showSlide(function(){

				});
			},
			closeTaskAddPage() {
				var vm = this;

				vm.$refs.scriptAdd.hide();
				var taskVm = Vue.getInstance('#mml_task_list_ctn');
				if(taskVm) {
					taskVm.$refs.taskList.refresh();
				}
			}
		},
		mounted(){
			//this.init()
		}
	})

	//俄罗斯版本，按参数路径修改参数，选择操作类型下拉框修改事件
	$("#elfcellConfigParamOperType").bind("change",function(){
		$(".showWholeVal").css("borderColor","#C9D1D6");
		$(".showWholeVal").siblings("p.prompt").text("");
		if ("MOD" == this.value) {
			$("#paramForElfcellADDAndMODDiv").css("display", "block");
			$("#paramForElfcellLSTDiv").css("display", "none");
			$("#paramForElfcellRMVDiv").css("display", "none");
			
			//清空参数路径和值
			$("input[name=paramPathNameForElfcellMODConfig]").val("");
			$("input[name=paramPathValueForElfcellMODConfig]").val("");

			//只留一个输入框
			$("#paramForElfcellADDAndMODDiv div:gt(0)").remove();
		}  else if ("RMV" == this.value){
			//RMV 操作只有一个输入框
			$("#paramForElfcellADDAndMODDiv").css("display", "none");
			$("#paramForElfcellLSTDiv").css("display", "none");
			$("#paramForElfcellRMVDiv").css("display", "block");
			
			//清空参数路径
			$("input[name=paramPathNameForElfcellRMVConfig]").val("");
		} else {
			//LST 操作可以有多个输入框
			$("#paramForElfcellADDAndMODDiv").css("display", "none");
			$("#paramForElfcellRMVDiv").css("display", "none");
			$("#paramForElfcellLSTDiv").css("display", "block");
			
			//清空参数路径和值
			$("input[name=paramPathNameForElfcellLSTConfig]").val("");
			$("input[name=paramPathNameForElfcellLSTConfig]").attr('placeholder', 'cannot contain {i} must contain . must not end with .');

			if("ADD" == this.value) {
				$("input[name=paramPathNameForElfcellLSTConfig]").attr('placeholder', 'cannot contain {i} must contain . must end with .');
				$('#paramForElfcellLSTDiv #addbtnForLstAndRMVPath').addClass('hide-item-cls');
			}else {
				$('#paramForElfcellLSTDiv #addbtnForLstAndRMVPath').removeClass('hide-item-cls');
			}
			//只留一个输入框
			$("#paramForElfcellLSTDiv div:gt(0)").remove();
		}
	});
	slideuplist();

	getOtherDeviceCodes();

	// 获取其他设备版本类型
	function getOtherDeviceCodes() {
		$.ajax({
			url: '${ctx}/cell/param/getDefaultProductTypeList.action',
			type: 'get',
			dataType: 'json',
			success: function(data) {
				if(data) {
					otherDeviceCodes = data;
				}
			}
		});
	}

	// 收起已选基站滑出页
	function slideuplist(){
		$(".enb-list-ctn").slideUp();
	}
	// 滑出已选设备列表浮层
	function slidedownlist(){
		$(".enb-list-ctn").slideDown();
		refreshEnbList();
	}
	// 刷新已选设备列表
	function refreshEnbList(){
		var tb = $('#gridCell_cellParam'),
			ctn = $(".enb-list-ctn .list-body"),
			rows = tb.datagrid('getSelections');
		ctn.html('');
		rows.map(function(row){
			var item = '<div class="list-item-info">' +row.host_name + ' (' + row.serial_number + ') <span class="list-item-op" onclick="clearDoStgRecord(\''+row.small_cell_code+'\',this)">x</span></div>'
			ctn.append(item);
		})
	}
	// 删除所有已选设备记录
	function delAllDoStgRecord(){
		$("#gridCell_cellParam").datagrid('clearSelections').datagrid('clearChecked');
		$(".enb-list-ctn .list-body").html('');
		setTimeout(function(){
			slideuplist();
		},1000);
	}
	// 删除单个已选设备记录
	function clearDoStgRecord(id,span){
		var tb = $("#gridCell_cellParam"),
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
		refreshEnbNum();
	}
	// 更新已选设备数量展示
	function refreshEnbNum(){
		setTimeout(function(){
			var tb = $('#gridCell_cellParam'),
			ctn = $("#MML_Config .circle-num-tips"),
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
	function menuHandler(item){
		var itemName = item.name;
		if(itemName == "copy"){
			
		}else if(itemName == "clear"){
			$("#showParamValues").children().remove();
			$("#showParamValues").unbind("contextmenu");
			isShowRightClickMenu = false;
			$('#enbmmlResult .param-values-oper').addClass('empty');
			enbmmlVue.isResult = false;
		}else if(itemName == "download"){
			exportConfigResultTxtFile();
		}else{
			showMsg('error_msg','<%=rb.getString("FeiFaCaoZuo")%>');
		}
	}
	
	/* 显示高级查询选项  */
	function moreQueryImgFun(){
		if($("#configEnbQueryImg").attr("flag")=="1"){
			$("#eNBHighQueryDiv").slideDown(500);
			$("#configEnbQueryImg").attr("flag","0");
			$("#configEnbQueryImg").addClass('el-icon-common-query-up').removeClass('el-icon-common-query-down');
		}else{
			$("#eNBHighQueryDiv").slideUp(500);
			$("#configEnbQueryImg").attr("flag","1");
			$("#configEnbQueryImg").addClass('el-icon-common-query-down').removeClass('el-icon-common-query-up');
		}
		event.stopPropagation();
	}
	
	/* 阻止冒泡  */
	$(".eNBHighQuery").click(function(event){
		event.stopPropagation();
	})

	
	// 下载指令执行结果
	function exportConfigResultTxtFile(){	
		var rs = $("#showParamValues");
		var url = "${ctx}/cell/param/exportConfigResultTxtFile.action";
		//处理数据，拼接结果文本
		var result = '',
			chlidres = rs.children();
	
		$.each(chlidres, function(index, item){
			if($(item).is('.command-tips')) { // 提示
				result += $(item).text() +"@<H>";
			}else {// 返回的参数列表
				var subList = $(item).children(),
					titleInfos = $(item).find('>span'),
					paramList = $(item).find('>div');
				
				$.each(subList, function(index, sub){
					if(sub.tagName == 'SPAN') { // span
						result += $(sub).text() +"@<H>";
					}else if(sub.tagName == 'DIV'){ // div
						var cells= $('>.cell',sub);

						// 拼接key、value键值对，多个以逗号连接
						var paramStr = Array.from(cells).map(function(cell){
							var paramISpans = $('span',cell),
								props = Array.from(paramISpans).map(function(span){ return $(span).text(); }).join(':');
								
							return props;
						}).join(',');

						result += paramStr + '@<H>';
					}
				});
				// 拼接命令头信息
				/* result += Array.from(titleInfos).map(function(span){ return $(span).text(); }).join('@<H>') + '@<H>';

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
				}) */
			}

			//result += '@<H>';
		});
		
		// 替换空格
		result = result.replace(/&nbsp;/g,'  ');
		
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
	function closeMMLTiShi(){
		$("#mmlTiShi").window("close");
	}
	// 查询命令列表
	function queryGridCellParam() {
		if($(".eNBHighQuery").css('display') == 'block'){
			$(".eNBHighQuery").slideToggle(300);
		}
		
		
		var search_text = $("#commonQueryText").val();
		
		$('#gridCell_cellParam').datagrid('reload',{
			search_text: search_text
		});
		hardwareVersionChange(hardware_version);
	}
	// 用于保存所有操作，以datagrid需要的形式
	var allOperNameForSearchGrid = {total: 0, rows: []};
	/**
	* 选择第一个叶子节点
	* @param data{array}: 子节点队列
	* @param status{object}: 父节点
	**/
	function selectFirstLeaf(data,status) {
		var map = status || {isLeaf: false,id: ''};
		if(map.isLeaf == true) return;
		if(data) {
			var treeN = $("#operGroupTree");
			data.map(function(item){
				if(map.isLeaf == true) return;
				
				if(item.children) {
					map = selectFirstLeaf(item.children,map);
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
			$("#eNBMMLScript").hide();
			$("#enbMMLBatchInput").hide();
			singleArry = [true]
		}
		closeLoading();
		$("#mmlListCenter").siblings(".panel-header").css("border-width", "1px 0 1px 0");
		// 加载操作集树形结构
		$("#operGroupTree").tree({
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
						/*
						if(node.text == 'Customized') {
							str += '';
						}else {
							str += '<i class="el-icon el-icon-plus" nodeid="'+node.id+'" style="position: absolute;right: 10px;top: 10px;color: #7A7992;" onclick="toADDCommand(&quot;'+ node.text +'&quot;)"></i>';
						}*/
						if(['PrivateTemplateId', 'PublicTemplateId'].includes(node.id)) {
							str += '<i class="el-icon el-icon-plus" nodeid="'+node.id+'" style="position: absolute;right: 10px;top: 10px;color: #7A7992;" onclick="toADDCommand(&quot;'+ node.text +'&quot;, event)"></i>';
						}else {
							str += '';
						}
					}else {
						// 自定义叶子节点 - 通过预处理标记 _isOwn 判断是否属于当前用户的组
						// _isOwn 在 loadData 之前由 _markOwnership 预处理设置
						var isOwn = (node._isOwn === true);
						// 临时节点始终属于自己
						if(node.id && String(node.id).indexOf('temp_') === 0) { isOwn = true; }

						if(isOwn) {
							str += '<span class="cus-btn-group" style="position: absolute;right: 10px;top: 10px;">';
							// 如果是 Untitled 指令，不显示复制和修改按钮
							if(node.text && !node.text.startsWith('Untitled')) {
								str += '<i class="cus-bt-cls el-icon el-icon-operation-copy" nodeid="'+node.id+'" style="margin-right: 8px;color: #7A7992;cursor:pointer;" onclick="copyCusNode(this,event)" title="复制"></i>';
								str += '<i class="cus-bt-cls el-icon el-icon-operation-edit" nodeid="'+node.id+'" style="margin-right: 8px;color: #7A7992;cursor:pointer;" onclick="editCusNode(this,event)" title="修改"></i>';
							}
							str += '<i class="cus-bt-cls el-icon el-icon-close" nodeid="'+node.id+'" style="color: #7A7992;cursor:pointer;" onclick="removeCusNode(this,event)" title="删除"></i>';
							str += '</span>';
						}
						// 其他人的指令组：不渲染操作按钮，只能查看
					}
				}

				return str;
			},
			onSelect: function(node){
				// 如果是叶子节点，构建右侧操作框；如果非叶子节点，则切换打开状态
				if ($("#operGroupTree").tree("isLeaf", node["target"])) {
					if(node.customized == 'true') {
						var tabNav = $('span[tabtit="elfcellConfigParamPanel"]');
						turnTabs(tabNav);
						reviewCustomParamConmmand(node);
					}else {
						resetCommandParam();
						// 删除之前的元素 
						$("#cellParam #paramNodesUl li").remove();
						$("#cellParam #paramNodesUl div").remove();
						doActionByOperID(node);
					}
				} else {
					$("#operGroupTree").tree("toggle", node["target"])
				}
			},
			onLoadSuccess: function(node,data){
				if(mmlCommandTreeSearchTxt){
					var map = selectFirstLeaf(data),
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
		
		// 基站搜索-软件版本-下拉面板
		$("#softwareVersionCombo_cellParam").combobox({
			valueField: "value",
			textField: "text",
			panelWidth: 180
		});
		
		$("#gridCell_cellParam").datagrid({
			url: '${ctx}/cell/param/getCellListOfMML.action?forSelect=1',
			queryParams:{like_fields:"serial_number,host_name"},
			singleSelect:singleArry[0],
			fit:true,
			fitColumns:true,
			border:false,
			rownumbers:true,
			pagePosition:'bottom',
			pageSize : 100,
			pageList : [100],
			idField:'small_cell_code',
			toolbar:'#toolbar_GridCell_cellParam',
			onLoadSuccess:gridCellParamDatagridLoadSuccess,
			onLoadError:datagridLoadError,
			onBeforeLoad: beforeLoad_gridCell_cellParam,
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
			var textOperName = $("#textOperName").combo('getText');
			// if("MOD MME" == textOperName){
			// 	showMMEPoolWay(true);
			// }
			refreshEnbNum();
		}

		function uncheckDoStg(index,row){
			var textOperName = $("#textOperName").combo('getText');
			// if("MOD MME" == textOperName){
			// 	showMMEPoolWay(false);
			// }
			refreshEnbNum();
		}
		
		function checkAllDoStg(rows){
			refreshEnbNum();
		}
		
		function uncheckAllDoStg(rows){
			refreshEnbNum();
		}
		// 操作名称输入框，初始化下拉面板
		$("#textOperName").combobox({
			width: 250,
			panelWidth: 400,
			panelHeight: 300,
			valueField: 'oper_id',
			textField: 'oper_name',
			onSelect: clickRowTableSearchOperName
		});
		$("#softwareVersionCombo_cellParam").combobox({
			url: "${ctx}/cell/cpeinfos/getHardwareVersionList.action?timeZone=" + timeZone,
			onLoadSuccess: function () {
				//默认选中第一条数据
				var firstRecordValue = $(this).combobox("getData")[0]? $(this).combobox("getData")[0].value : '';
				original_hardware_version = firstRecordValue;
				var groupNum = $("#softwareVersionCombo_cellParam").combobox('getData').length;
				$(this).combobox("select", firstRecordValue);
				if(groupNum == 1){
					$("#softwareVersionCombo_cellParam").combobox('disable')
				}
			},
			onSelect: function (record) {
				hardware_version = record.value;
				sel_product = record.text;
				$("#gridCell_cellParam").datagrid("clearSelections");
				$("#gridCell_cellParam").datagrid("reload");
				hardwareVersionChange(hardware_version);
				$("#textOperName").combobox("reload");
				resetCommandParam();
			}
			
		});
		
		$("#gridCell_cellParam").datagrid("getPager").pagination({
			layout:['prev','manual','next','refresh']
		});
		
		// 基站搜索-类型-change事件
		$("#toolbar_GridCell_cellParam select[name='type']").bind("change", function(e) {
			var type = e.target.value;
			$("#toolbar_GridCell_cellParam .input_li").hide();
			$("#toolbar_GridCell_cellParam ." + type).show();
			
		});
		
		// 基站搜索-地域树-下拉面板
		$("#regnTreeCombo_cellParam").combotree({
			url: "${ctx}/system/deviceGroup/getDeviceGroupTreeData.action",
			panelWidth: 200
		});
		
		$("#stationConfigDeviceGroup").combobox({
			url: '${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action',
			width: 260,
			panelWidth: 260,
			panelHeight: 200,
			valueField: 'id',
			textField: 'group_name',
			onSelect: chooseStationConfigDeviceGroup,
			onLoadSuccess : function(){
				
			}
		});
		
		$("#stationConfigDeviceGroup").combobox('setValues',['-1','<%=rb.getString("QuanBuSheBeiZu")%>']); 
	
		// 样式修改
		$(".group_id input.textbox-text").css("padding", "0").css("margin", "0");
		
		$('#MMLScript').load('${ctx}/task/MMLScript/toMMLScriptTaskList.action',function(data){
			$.parser.parse(this);
		})
		
	});
	/**
	* 设备组下拉选中事件
	* @param data{object}: 选中项数据
	**/
	function chooseStationConfigDeviceGroup(data){
		choosedGroupId = data.id;
		$("#gridCell_cellParam").datagrid("reload");	
	}
	/**
	* 根据操作ID和操作类型，动态生成操作面板
	* @param operID{string}：命令树节点Id
	* @param actionType{string}：命令类型
	* @param operName{string}：命令名
	* @param cellNumber{string}：基站编码
	**/
	function  doActionByOperIDAndType(operID, actionType, operName,cellNumber) {
		// 向后台请求该操作包含的子节点信息
		var param = {
			operID : operID,
			actionType : actionType,
			hardwareVersion: hardware_version
		};

		var regVersion = /^436Q/;
		var hardwareVer = regVersion.test(hardware_version );
		var reg430 = /^NEU430/;
		var hardware430 = reg430.test(hardware_version ),
			isCR4860 = /^CR_B4860_/.test(hardware_version);
		var isMLN = /^MLN_/.test(hardware_version);
		var matchCodes = [
				"NBIOT1.0",
				"CR4.0",
				"QC4.2",
				"QC4.2T",
				"QC4.2RELAY",
				"QC3.1",
				"CA2.0",
				"436Q_CA1.0",
				"DXDF1.0",
				"EA4.0",
				'BAIBLQ1.0',
				'BLX1.0',
				'MLQ1.0',
				'BSC1.0',
				'BTS1.0',
				'BM1.0'
			],
			otherCodes = otherDeviceCodes,
			lstMatch = actionType == "v_lst" && ['EA4.0DUAL','BAIBLQ1.0','BLX1.0','MLQ1.0','BSC1.0','BTS1.0','BM1.0'].includes(hardware_version);
		
		if (hardware430 || matchCodes.includes(hardware_version) || hardwareVer || isCR4860 || otherCodes.includes(hardware_version) || lstMatch || isMLN) {
			if(isLoadJSP(operID,actionType,hardware_version)) {
				$("#operValueForm").panel({
					border : false,
					queryParams: param,
					href: '${ctx}/eNodeB/config/toConfigFormPage.action',
					onLoad: function(){
						createMML();
					}
				});
			}else {
				$.post("${ctx}/cell/param/getParamGroupTreeNodes.action", param, function(data) {
					if($("#cellParam #paramNodesUl").length==0){
						var $paramNodesUl = '<ul id="paramNodesUl" class="paramNodesUl"></ul>'
						$("#operValueForm").append($paramNodesUl);
					}
					/*删除之前的元素*/
					$("#cellParam #paramNodesUl li").remove();
					$("#cellParam #paramNodesUl div").remove();
					allMibDn = "";
					
					var  innerVersionList = ["CA2.0", "436Q_CA1.0", "NEU430_CA1.0", "CR_B4860_AC4.0", "CR_B4860_CA4.0","Nova430i_CA", "Nova430_CA", "Nova430i_DC", "Nova430_DC", "Nova430i_SC", "436Q_DC1.0", "436Q_SC1.0","MLN_CA1.0"];
					if(innerVersionList.includes(hardware_version)){
						var operNameList = ["NTP","SYNC","IPSEC","SAS","REBOOT","RESET","ACT","DEACT","HTTP","POWER_AMP","LMT_LOGIN"],
							excluded = operNameList.every(function(item){
								return operName.indexOf(item) == -1;
							});
						if((hardware_version == "CR_B4860_CA4.0" || hardware_version == "MLN_CA1.0") && (operName =="MOD CELL" ||operName =="LST CELL")){
							cellNumber = 2;
						}
						
						if (actionType != "v_lst") {
							if (excluded) {
								var $liForNumCells = createCellIndex(cellNumber);
								$liForNumCells.css('display','block');
								if(cellNumber>1) $("#cellParam #paramNodesUl").append($liForNumCells);
								
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
									
									var operName = $("#operGroupTree").attr("operName");
									$("#inputMML").val(operName);
									selectedCellNum = $(this).val();
									
									if ( selectedCellNum >= 2) {
										$("#cellParam #paramNodesUl input").val("").blur();
										$("#cellParam #paramNodesUl select").val("");
										$(this).val("2");
										
										$1338obj.attr("disabled", "disabled");
										[$1332obj, $1333obj, $1334obj, $1335obj].map(function(item){
											item.attr("disabled", "disabled");
											item.attr("style", "background-color:#EAF1F4");
										});
										
										if( operName.indexOf("EUTRANNFREQ") > -1 || operName.indexOf("EUTRANNCELL") > -1){
											var selects = $("#cellParam #paramNodesUl select");
											var inputs = $("#cellParam #paramNodesUl input");
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
										$("#cellParam #paramNodesUl input").val("").blur();
										$("#cellParam #paramNodesUl select").val("");
										$(this).val(selectedCellNum);
										
										$1338obj.attr("disabled", false);
										[$1332obj, $1333obj, $1334obj, $1335obj].map(function(item){
											item.attr("disabled", false);
											item.attr("style", "background-color:#FFFFFF");
										});
										
										if( operName.indexOf("EUTRANNFREQ") > -1 || operName.indexOf("EUTRANNCELL") > -1){
											var selects = $("#cellParam #paramNodesUl select");
											var inputs = $("#cellParam #paramNodesUl input");
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
							if (excluded || (hardware_version == "CR_B4860_CA4.0" || hardware_version == "MLN_CA1.0")) {
								var $checkBoxForNumCells = createCheckBoxIndex(cellNumber);
								if(cellNumber>1) $("#cellParam #paramNodesUl").append($checkBoxForNumCells);
							}
						}
					}
					
					if(actionType != "v_lst") {
						for (var nodeNum=0; nodeNum < data.length; nodeNum++) {
							var $li = createEleByIDAndType(data[nodeNum]);
							if ($li) {
								$("#cellParam #paramNodesUl").append($li);
							}
						}
					}
					
					if(actionType == "v_lst") {
						var lstList = data,
	                        hiddenInput = $('<li style="height: 0px;overflow: hidden;width: 0px;min-width: 0px;display: block;"><input name="lstId" type="hidden"/></li>'),
	                        ids = [];
						
	                    $("#cellParam #paramNodesUl").empty();
	                    $("#cellParam #paramNodesUl").append(hiddenInput);
	                 	//QRTB-CA 站 支持LST Cell 命令选择小区; QRTB-CA 对应hardware_version：436Q_CA1.0
						//if (hardware_version == "CR_B4860_CA4.0") {
						if(['CR_B4860_CA4.0', '436Q_CA1.0','MLN_CA1.0'].includes(hardware_version)) {
							var $checkBoxForNumCells = createCheckBoxIndex(cellNumber);
							if(cellNumber>1) $("#cellParam #paramNodesUl").append($checkBoxForNumCells);
						}

	                    lstList.map(function(item){
	                        var tips = createExplanation(item.software_version),
								ckdom = $(['<li style="width: 95%;height: 25px;">',
	                                        '<div style="display: flex;align-items: end;">',
	                                        '<input type="checkbox" checked id="'+ item.mib_dn +'" onchange="lstCkChange(this.checked)" style="width: 40px;"/>',
	                                        '<label for="'+ item.mib_dn +'" style="min-width: fit-content;">' + item.PARAM_NAME + (item.software_version?tips:'') + '</label>',
											'<span style="padding-left: 10px;word-break: break-all;">' + item.name_path + '</span>',
	                                        '</div>',
	                                        '</li>'].join(' '));
							if(item.v_type) {
								var matched = item.v_type.match(/\d+:\d+/)[0].split(':'),
									minVal = matched[0],
									maxVal = matched[1];
								ckdom = $(['<li id="cell_index_li">',
												'<label for="'+ item.mib_dn +'">'+ item.PARAM_NAME + (item.software_version?tips:'') +'</label>',
												'<input id="'+ item.mib_dn +'" name="'+ item.mib_dn +'" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="Integer,range:1-8" ',
													'min_value="'+minVal+'" max_value="'+maxVal+'" class="border border-box" must="0"/>',
												'<div id="i_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">',
													'Integer,range:'+minVal+'-'+maxVal,
												'</div>',
											'</li>'].join(' '));
								$("#cellParam #paramNodesUl").append(ckdom);
							}else {
								$("#cellParam #paramNodesUl").append(ckdom);
								ids.push(item.mib_dn);
							}
	                    });
	                    
	                    hiddenInput.find('input').val(ids.join(','));
					}
					
					$.parser.parse('#cellParam #paramNodesUl');
					
					if(isCR4860 || isMLN){
						
						$('[name^=LTE_ANTENNA_PORTS_COUNT]').parent("li").addClass("double1-item");
						$('[name^=LTE_CELL_POWER_MODIFY]').parent("li").addClass("double2-item");
						
						var doubleli = $("<li class='appendLi'></li>")
						
						if(operName == "MOD POWER_LEVEL"){
							$("#cellParam #paramNodesUl").append(doubleli);
							var li1 = $(".double1-item") , li2 = $(".double2-item");
							$(".appendLi").append(li1);
							$(".appendLi").append(li2);
							
						}
						
						
						
						var isTC = hardware_version.indexOf("TC")
						if (isTC != -1){
							$('[name^=LTE_ANTENNA_PORTS_COUNT]').find("option[value='4']").remove();
							$('[name=LTE_DL_256QAM_ENABLE]').parent("li").remove();
						}else {
							//$('[name=LTE_DL_256QAM_ENABLE]').val('0');
						}
					}
					
					
					
					customEventFnc();
					var $1282obj = $('[name=i]'),
						$1330obj = $('[name=LTE_DL_EARFCN]'),
						$1331obj = $('[name=LTE_UL_EARFCN]'),
						$1332obj = $('[name=LTE_DL_BANDWIDTH]'),
						$1333obj = $('[name=LTE_UL_BANDWIDTH]'),
						$1336obj = $('[name=LTE_FREQ_BAND_INDICATOR]'),
						$1337obj = $('[name=LTE_BANDS_SUPPORTED]'),
						$1399obj = $('[name=LTE_CELL_POWER_MODIFY]'),

						$1336obj = $('[name=LTE_FREQ_BAND_INDICATOR_CELL1]'),
						$1332obj = $('[name=LTE_DL_BANDWIDTH_CELL1]'),
						$60015obj = $('[name=LTE_EARFCNDL_LIST_CELL1]'),
						$60304obj = $('[name=LTE_EARFCNDL_LIST_CELL2]'),
						$60357obj = $('[name=LTE_EARFCNDL_LIST_CELL3]');
					
					//如果SAS开关打开，则该参数不可配置
					if(false && SASEnble == "1" && ["MOD CELL","MOD CELL1","MOD CELL2","MOD CELL3"].includes(operName)){
						[$1330obj, $1331obj, $1336obj,$1337obj,$1399obj,$60015obj,$60304obj,$60357obj].map(function(item){
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
					
					var $4339 = $('#4339');
					if($4339.length) {
						$4339.on('change', function(){
							var val = $4339.val(),
								$4343 = $('#4343'),
								$4344 = $('#4344');
							
							if(val == '2') {
								$4343.parent().show();
								$4344.parent().show();
							}else {
								$4343.parent().hide();
								$4344.parent().hide();
							}
						});
					}

					addQAFACascade();
					
					createMML();
				},"json"); 
			}
		} else {
			$("#operValueForm").panel({
				border : false,
				queryParams: param,
				href: '${ctx}/eNodeB/config/toConfigFormPage.action',
				onLoad: function(){
					createMML();
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
	function isLoadJSP(operId,actionType,version) {
		var bool = false,
			list = [
				{id: '300068',type: 'v_mod',v: 'QC3.1'},
				{id: '1062',type: 'v_mod',v: hardware_version},
				{id: '1048',type: 'v_mod',v: hardware_version},
				{id: '60017',type: 'v_mod',v: hardware_version},
				{id: '1058',type: 'v_mod',v: hardware_version},
				{id: '300164',type: 'v_mod',v: hardware_version},
				{id: '300069',type: 'v_mod',v: hardware_version},
				{id: '300070',type: 'v_mod',v: hardware_version},
				{id: '1010',type: 'v_mod',v: hardware_version},
				{id: '60067',type: 'v_mod',v: hardware_version},
				{id: '60088',type: 'v_mod',v: hardware_version},
				{id: '60089',type: 'v_mod',v: hardware_version},
				{id: '401512',type: 'v_mod',v: hardware_version},
				{id: '301512',type: 'v_mod',v: hardware_version},
				{id: '80053',type: 'v_mod',v: 'NBIOT1.0'},
				{id: '60019',type: 'v_mod',v: 'CR_B4860_DC4.0'},
				{id: '60019',type: 'v_mod',v: 'CR_B4860_SC4.0'},
				{id: '60019',type: 'v_mod',v: 'CR_B4860_TC4.0'},
				{id: '60019',type: 'v_mod',v: 'CR_B4860_CA4.0'},
				{id: '2160019',type: 'v_mod',v: 'MLN_SC1.0'},
				{id: '2160019',type: 'v_mod',v: 'MLN_CA1.0'},
				{id: '2160019',type: 'v_mod',v: 'MLN_DC1.0'},
				{id: '301617',type: 'v_mod',v: hardware_version},
				{id: '300084',type: 'v_mod',v: hardware_version},
				{id: '60059',type: 'v_mod',v: 'CR_B4860_DC4.0'},
				{id: '60059',type: 'v_mod',v: 'CR_B4860_SC4.0'},
				{id: '60059',type: 'v_mod',v: 'CR_B4860_TC4.0'},
				{id: '60059',type: 'v_mod',v: 'CR_B4860_CA4.0'},
				{id: '2160059',type: 'v_mod',v: 'MLN_SC1.0'},
                {id: '2160059',type: 'v_mod',v: 'MLN_CA1.0'},
                {id: '2160059',type: 'v_mod',v: 'MLN_DC1.0'},
				{id: '300140',type: 'v_mod',v: 'QC3.1'},
				{id: '60017',type: 'v_mod',v: 'CR_B4860_DC4.0'},
				{id: '60017',type: 'v_mod',v: 'CR_B4860_SC4.0'},
				{id: '60017',type: 'v_mod',v: 'CR_B4860_TC4.0'},
				{id: '60017',type: 'v_mod',v: 'CR_B4860_CA4.0'},
				{id: '2160017',type: 'v_mod',v: 'MLN_SC1.0'},
                {id: '2160017',type: 'v_mod',v: 'MLN_CA1.0'},
                {id: '2160017',type: 'v_mod',v: 'MLN_DC1.0'},
				{id: '2160088',type: 'v_mod',v: 'MLN_SC1.0'},
                {id: '2160088',type: 'v_mod',v: 'MLN_CA1.0'},
                {id: '2160088',type: 'v_mod',v: 'MLN_DC1.0'},
				{id: '2160089',type: 'v_mod',v: 'MLN_SC1.0'},
                {id: '2160089',type: 'v_mod',v: 'MLN_CA1.0'},
                {id: '2160089',type: 'v_mod',v: 'MLN_DC1.0'},
				
				{id: '301146',type: actionType,v: 'EA4.0'},
				{id: '301147',type: actionType,v: 'EA4.0'},
				{id: '301209',type: actionType,v: 'EA4.0'},
				{id: '301208',type: actionType,v: 'EA4.0'},
				{id: '301201',type: actionType,v: 'EA4.0'},
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

	function createExplanation(text) {
		var explanation = text || '',
			tips = [" <i class='el-icon el-icon-circle-info explanation-ctn' style='font-size: 12px;position: relative;'>",
					"<div class='explanation-tips'>"+explanation+"</div>",
					"</i> "].join(" ");

		return tips;
	}

	var ECITitle = {};
	/**
	* 根据元素的数据，生成元素
	* @param eleData{object}: 元素数据
	**/
	function createEleByIDAndType(eleData) {
		var explanation = eleData["explanation"],
			tips = createExplanation(explanation);

		if(allMibDn!=""){
			allMibDn = allMibDn+","+eleData["mib_dn"];
		}else{
			allMibDn = eleData["mib_dn"];
		}
		var $li = $("<li></li>");
		var $label = $("<label for='" + eleData["param_id"] + "'>" + eleData["paramName"] + (explanation?tips:"") + ":</label>");
		var ele;
		var errSpan;
		var dataType = eleData["dataType"];
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
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"] + "'>");
			ele.bind("blur", trimValue);
		}
		else if (dataType == "Ipv4Addr") {
			// ipv6 校验 
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"] + "'>");
			ele.bind("blur", validateIPV4Address);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>" +eleData["title"]+ "</div>");
		}else if (dataType == "Ipv6Addr") {
			// ipv6 校验 
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"] + "'>");
			ele.bind("blur", validateIPV6Address);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>" +eleData["title"]+ "</div>");
		} else if (type_1.test(dataType)) {
			<%--样式：string-64，最大长度为64个字符--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"] + "' max_length='" + type_1.exec(dataType)[1] + "'>");
			ele.bind("blur", trimValue);
			ele.bind("blur", validateMaxAndMinLength);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>" +dataType+ "</div>");
		} else if (type_2.test(dataType)) {
			<%--样式：string-(3:4)，最小长度3,最大长度4--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"] + "' min_length='"
			+ type_2.exec(dataType)[1] + "' max_length='" + type_2.exec(dataType)[2] + "'>");
			ele.bind("blur", trimValue);
			ele.bind("blur", validateMaxAndMinLength);
			if ((hardware_version =="QC3.1" || hardware_version =="QC4.2" || hardware_version=="QC4.2RELAY" || hardware_version=="QC4.2T") && eleData["mib_dn"] == "LTE_REFERENCE_SIG_POWER") {
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
			ele.val(eleData["dftValue"]);

			if(eleData["title"]) {
				errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>" +eleData["title"]+ "</div>");
			}
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
			ele.val(eleData["dftValue"]);
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
			ele.val(eleData["dftValue"]);
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
			ele.val(eleData["dftValue"]);
		} else if ("unsignedInt" == dataType || "int" == dataType || "unsignedLong" == dataType) {
			<%--样式：unsignedInt或int，无符号整型，没有其他限制条件--%>
			var ele_String = "<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"] + "'";
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
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"] + "' min_value='"
			+ type_4.exec(dataType)[1] + "'>");
			ele.bind("blur", trimValue);
			ele.bind("blur", validateMaxAndMinVal);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>"/* minValue:"
			+ type_4.exec(dataType)[1] + */+dataType+ "</div>");
		} else if (type_5.test(dataType)) {
			<%--样式：unsignedInt-[1000:65535]或int-[1000:65535]，最小值1000,最大值65535--%>
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"]
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
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"]
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
			var ele_String = "<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"]
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
		} else if (dataType.indexOf("List") || dataType.indexOf("struct")) {
			ele = $("<input id='" + eleData["param_id"] + "' value='" + eleData["dftValue"] + "' title='" + dataType + "'>");
			ele.bind("blur", trimValue);
			errSpan = $("<div id='" + eleData["param_id"] + "_err' class='errSpan' style='margin-top:5px;margin-left:200px;'>"+dataType+"</div>");
			ele.bind("blur", validateMaxAndMinLength);
			
			if (hardware_version =="QC3.1" || hardware_version =="QC4.2" || hardware_version=="QC4.2RELAY" || hardware_version=="QC4.2T"){
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
			ele.bind("blur", createMML);
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
	* 校验IPv4
	* @param e{event}: 事件对象
	**/
	function validateIPV4Address(e){
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
	function validateIPV6Address(e){
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
	function mmlSetEleVal(){
		var mmlVar = $("#inputMML").val();
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
	function createMML(){
		// 清空自定义输入标识
		$('#inputMML').attr('iscustom','');
		
		var allMibDn = "";
		var imsibind = "";
		//如果验证不通过，则不会生成脚本
		if($(this).hasClass("err_border")){
			return;
		}
		var operName = $("#operGroupTree").attr("operName");
		var mmlVar = operName;
		
		var selects = $("#cellParam #paramNodesUl select");
		var inputs = $("#cellParam #paramNodesUl input:not(.root-param)");
		
		for(var i=0;i<selects.length;i++){
			if($(selects[i]).attr("id")!= "numCells" && !$(selects[i]).hasClass('ignore')){
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
			}else if(midDnNameInputs){
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
			var $sdom = $("[name='"+mibdnary[i]+"']",$("#cellParam #paramNodesUl")),
				elVar = ($sdom.val()||'').trim();
			if(mibdnary[i] == "IPSEC_LEFTINTERFACE"){
				text = $("#IPSEC_LEFTINTERFACE_name option:selected").text();
				if(!$sdom.hasClass("err_border")){
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
				if((elVar != undefined && elVar != "") || $sdom.hasClass('none-valid')){
					isAllEmpty = false;
					if(!$sdom.hasClass("err_border")){//如果有元素有不符合要求，则不允许生成脚本
						
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
			$("#inputMML").val(mmlVar + ";");
		} else {
			$("#inputMML").val(operName);
		}
	}
	// 提交之前，验证是否用户的所有输入都已合法，返回true表示没有错误，否则表示有错误存在
	function validateErrBeforeSubmit() {
		var isNotCustom = $('#inputMML').attr('iscustom') != '1';
		
		if(isNotCustom) $('#showParamValues input').blur();
		
		var allowSubmit = true;
		//var errLenth = $("#operValueForm .errSpan:visible").length;
		var errLenth = $("#operValueForm .errSpan.redColor").length;
		var errBorder = $("#operValueForm .err_border").length;
		if (errLenth > 0 || errBorder > 0) {
		<%--用户输入存在不合法的情况--%>
			allowSubmit = false;
			$("#operValueForm .err_border").fadeOut().fadeIn();
		}
		return allowSubmit;
	}
	/**
	* 点击操作集之后，进行该操作
	* @param node{dom}: 命令树节点
	**/
	function doActionByOperID(node){
		var nodeList = node.id.split("_");
		var operID = nodeList[0];
		var actionType = nodeList[1],
			cellNumber = node.cellNumber;
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
		$("#operGroupTree").attr("paramGroupId", operID);
		$("#operGroupTree").attr("operType", actionType);
		$("#operGroupTree").attr("operName", operName);
		$("#inputMML").val(operName);
		$("#textOperName").combobox("setValue", node.id);
		$("#textOperName").combo("setText", operName);
		doActionByOperIDAndType(operID, actionType,operName,cellNumber);
		enbmmlVue.init(operID, actionType,operName,cellNumber)
	}
	// 输入框回车事件
	function operInputEnter() {
		var opers = $("#textOperName").combo("getText");
		var allNodes = $("#operGroupTree").tree("getChildren");
		var reg = /.*\((.*)\)/;
		for (var num = 0; num < allNodes.length; num++) {
			if (reg.test(allNodes[num].text.toUpperCase())) {
				var node_text = allNodes[num].text.toUpperCase();
				var nodeOper = reg.exec(node_text)[1];
				if (nodeOper == opers.toUpperCase()) {
					// 找到该节点，选中
					$("#operGroupTree").tree("select", allNodes[num].target);
					break;
				}
			}
		}
	}
	/**
	* 选中命令下拉后联动选中命令树对应节点
	* @param data{object}：选中项
	**/
	function clickRowTableSearchOperName(data) {
		// 将选择的内容上屏到输入框
		var oper_name = data["oper_name"];
	
		// 重置查询条件，载入全量树
		$('#groupQueryText').val('');
		$("#operGroupTree").tree("doFilter", '');
		
		/*<%-- 选中左侧操作集的节点 --%>*/
		var operNode = $("#operGroupTree").tree("find", data["oper_id"]);
		// 先打开其父节点
		var parentNode = $("#operGroupTree").tree("getParent", operNode["target"]);
		$("#operGroupTree").tree("expand", parentNode["target"]);
		$("#operGroupTree").tree("select", operNode["target"]);
	}
	// 执行命令
	function clickGo() {
		enbmmlVue.resultType = 'res';
		enbmmlVue.isResult = true;
		if (!validateErrBeforeSubmit()) {
			return;
		} 
		
		var ctner = $('#cellParam');
		var smallCells = "";
		var cellsText = "";
		var serial_numbers = "";
		
		var paramGroupId = $("#operGroupTree").attr("paramGroupId");
		
		var mmlstr = $("#inputMML").val();
		
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
			goExecuteCmd(mmlstr);
			return;
		}
		
		<%--获取选择的小站--%>
		var selCells = $("#gridCell_cellParam").datagrid("getSelections");
		if (0 == selCells.length) {
			showMsg('prompt_msg',QingXuanZeSheBei);
			return;
		}
		
		if(hardware_version == "CA2.0" || hardware_version == "436Q_CA1.0" || hardware_version == "NEU430_CA1.0"){
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
		var inputs = $("#cellParam #paramNodesUl input"); 
		$.each(inputs,function(index,value){
			$(value).val($(value).val().trim());
		});
		
		var textOperName = $("#textOperName").combo('getText');
		if("MOD MME" == textOperName){
			modMMEPool(selCells,mmlstr,hardware_version);
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
		var paramJson = $('#operValueForm').serializeJson();
		
		<%-- 某个 指标若为空，则表示不对该指标进行设置，从对象中去掉该指标 --%>
		var deleteKeyArr = [];
		
		enbmmlVue.resultType = 'res';
		enbmmlVue.isResult = true;
		
		var param = {};
		param["smallCells"] = smallCells;
		param["serial_numbers"] = serial_numbers;
		param["inputMML"] = encodeURIComponent(mmlstr);
		param["hardwareVersion"] = hardware_version;
		param["cellIndex"] = operIndex;
		
		$.post("${ctx}/cell/param/checkIsNeedReboot.action", param, function (data) {
			
			// 新的后端返回结构: {reboot: boolean, list: [{confirmMsg: string, keyword: string}]}
			
			// 处理reboot提示
			if (data && data.reboot) {
				// reboot为true时,显示重启提示
				$('.success_msg').html(CiPeiZhiChongQiHouShengXiao);
				var width = $('.success_msg').width();
				$('.success_msg').css("left",'50%');
				var left = parseFloat($('.success_msg').css("left")) - width/2;
				$('.success_msg').css("left",left+'px');
				$('.success_msg').animate({top:'55px'},200,function(){
					setTimeout(function(){
						$('.success_msg').animate({top:'-40px'},function(){})
					},3000)
				})
			} 
            handleParamSubmitWithConfirm(param, mmlstr, data, serial_numbers, operName);
		},"json");
	}
	
	/**
	 * 处理参数提交(带二次确认)
	 * @param param 参数对象
	 * @param mmlstr MML命令字符串
	 * @param checkData checkIsNeedReboot返回的数据
	 * @param serial_numbers 序列号
	 * @param operName 操作名称
	 */
	function handleParamSubmitWithConfirm(param, mmlstr, checkData, serial_numbers, operName) {
		var specialValidate = specialVaildate(mmlstr);
		if(specialValidate != ""){
			showMsg('prompt_msg',specialValidate);
			return;
		}
		
		// 检查是否需要二次确认
		if (checkData && checkData.list && checkData.list.length > 0) {
			// 检查是否包含 RESET 关键字
			var hasReset = false;
			for(var i = 0; i < checkData.list.length; i++) {
				if(checkData.list[i].keyword === 'RESET') {
					hasReset = true;
					break;
				}
			}
			
			// 构建确认消息 - 图标只显示一次，内容循环生成
			var confirmMsg = '<div class="mml-second-confirm-container">';
			// 图标只显示一个
			confirmMsg += '<div class="mml-second-confirm-icon"><i class="el-icon el-icon-status-alarm mmlDoubleConfirmIconCls"></i></div>';
			// 内容区域，循环显示所有确认消息
			confirmMsg += '<div class="mml-second-confirm-content">';
			for(var i = 0; i < checkData.list.length; i++) {
				var item = checkData.list[i];
				confirmMsg += '<div class="mml-second-confirm-item"><span class="mml-second-bullet"></span>' + item.confirmMsg + '</div>';
				if(i < checkData.list.length - 1) {
					confirmMsg += '<br/>';
				}
			}
			confirmMsg += '</div>';
			confirmMsg += '</div>';
			
			// 如果包含 RESET，添加输入框
			if(hasReset) {
				confirmMsg += '<div class="mml-second-reset-input-container">';
				confirmMsg += '<label class="mml-second-reset-input-label"><%=rb.getString("ErCiQueRenShuRuBiaoTi")%></label>';
				confirmMsg += '<input type="text" class="mml-second-reset-input" id="mml-second-reset-confirm-input" placeholder="RESET" />';
				confirmMsg += '</div>';
			}
			
			// 显示二次确认对话框
			var resetValidated = false; // 标志变量：RESET是否已验证通过
			var confirmDialog = $.messager.confirm({
				width: 500,
				title: '<%=rb.getString("QueRen")%>',
				msg: confirmMsg,
				fn: function(r){
					if (r) {
						// 如果包含 RESET，检查是否已验证通过
						if(hasReset && !resetValidated) {
							return false;
						}
						// 没有 RESET 验证或已验证通过，执行操作
						updateActionHistoryContent(serial_numbers, operName, mmlstr);
						executeOperParamGroupValues(param);
					}
				}
			});
			
			// 如果有 RESET 输入框，需要监听输入并控制确认按钮状态
			if(hasReset) {
				setTimeout(function() {
					var $input = $('#mml-second-reset-confirm-input');
					var $dialog = $input.closest('.panel-body');
					var $dialogWindow = $dialog.parent();
					var $confirmBtn = $dialogWindow.find('.messager-button .l-btn:first');
					
					// 初始状态：禁用确认按钮
					$confirmBtn.addClass('l-btn-disabled').attr('disabled', 'disabled');
					
					// 完全重写按钮的点击事件，使用捕获阶段拦截
					$confirmBtn.off('click').on('click', function(e) {
						var inputValue = $input.val();
						if(inputValue !== 'RESET') {
							// 输入不正确，阻止所有操作，不关闭对话框
							e.preventDefault();
							e.stopPropagation();
							e.stopImmediatePropagation();
							return false;
						}
						// 输入正确，设置验证标志
						resetValidated = true;
						
						// 阻止默认的确认框关闭行为
						e.preventDefault();
						e.stopPropagation();
						e.stopImmediatePropagation();
						
						// 直接关闭对话框 - 使用最简单的DOM操作
						// 查找所有可能的对话框容器
						var $messagerBody = $input.closest('.messager-body');
						var $windowWrapper = $messagerBody ? $messagerBody.closest('.window-wrapper') : null;
						
						if($windowWrapper && $windowWrapper.length > 0) {
							// 找到完整的window容器，直接移除
							$windowWrapper.remove();
						} else {
							// 备用方案：移除父容器
							$dialogWindow.remove();
						}
						
						// 移除所有相关的遮罩和阴影元素
						$('.window-mask').remove();
						$('.window-shadow').remove();
						$('.messager-mask').remove();
						
						// 执行操作
						updateActionHistoryContent(serial_numbers, operName, mmlstr);
						executeOperParamGroupValues(param);
						
						return false;
					});
					
					// 监听输入变化
					$input.on('input', function() {
						var inputValue = $(this).val();
						if(inputValue === 'RESET') {
							// 启用确认按钮
							$confirmBtn.removeClass('l-btn-disabled').removeAttr('disabled');
						} else {
							// 禁用确认按钮
							$confirmBtn.addClass('l-btn-disabled').attr('disabled', 'disabled');
						}
					});
				}, 100);
			}
		} else {
			// list为空,先更新历史记录，再直接下发
			updateActionHistoryContent(serial_numbers, operName, mmlstr);
			executeOperParamGroupValues(param);
		}
	}
	
	/**
	 * 执行参数配置操作
	 * @param param 参数对象
	 */
	function executeOperParamGroupValues(param) {
		$.post("${ctx}/cell/param/operParamGroupValues.action", param, function (data) {
			if (data["success"]) {
				// 成功处理
			} else {
				showMsg('error_msg',data["message"]);
			}
		}, "json");
	}
	
	/**
	* 特殊处理某些参数
	* @param mmlStr{string}：命令
	**/
	function specialVaildate(mmlStr){
		var operName = mmlStr.substring(0, mmlStr.indexOf(":"));
		if(operName != null && operName.trim() == "MOD PCI_RANGE"){
			var startPCI = $("#cellParam #paramNodesUl input[name='LTE_SMALLCELL_START_PCI']").val();
			var pciRange = $("#cellParam #paramNodesUl input[name='LTE_SMALLCELL_PCI_RANGE']").val();
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
	function goExecuteCmd(mmlstr) {
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
			$("#showParamValues").animate({scrollTop: $("#showParamValues")[0].scrollHeight + 'px'}, 500);
			
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
	function updateResultInfo(content) {
		var resultDiv = $('#showParamValues'),
			last = resultDiv.children(':last');
		resultDiv.append(content);
	}
	/**
	* 更新动作履历记录内容
	* @param serial_numbers{string}: 基站序列号
	* @param operName{string}: 命令名
	* @param mmlstr{string}: 要执行的命令
	**/
	function updateActionHistoryContent(serial_numbers, operName,mmlstr) {
		var date = dateformatter(new Date(gloableTime));//operName + ": " + smallCells
		var $li = $(("<li style='padding:5px 0;'></li>")).append($("<span>" + date + " " + mmlstr + "{" + serial_numbers + "}" + "</span><br/>"));
		var defaultWindowObj = $("#winDefault");

		defaultWindowObj.append($li);
		defaultWindowObj.animate({scrollTop: defaultWindowObj[0].scrollHeight + 'px'}, 500);
		
		var strContent = "<div class='command-tips'>" + date + " " + mmlstr + "{" + serial_numbers + "}" + "</div>";
		// 更新到结果记录方便知晓指令状况
		updateResultInfo(strContent);
	}
	/**
	 * 预处理树数据：递归标记每个自定义节点的 _isOwn 属性
	 * Private Template 下：子分组名 === user_code 的分组及其下属叶子节点标记为 true
	 * Public Template 下：所有叶子节点标记为 true（公共指令所有人可操作）
	 * @param {Array} nodes 树节点数组
	 * @param {string|null} parentId 父节点ID
	 * @param {boolean|null} inheritOwn 从父级继承的归属标记
	 */
	function _markOwnership(nodes, parentId, inheritOwn) {
		if(!nodes || !nodes.length) return;
		for(var i = 0; i < nodes.length; i++) {
			var n = nodes[i];
			if(n.customized != 'true') {
				// 非自定义节点，递归子节点
				if(n.children) _markOwnership(n.children, n.id, null);
				continue;
			}
			if(parentId === 'PublicTemplateId') {
				// Public Template 下的所有节点：所有人可操作
				n._isOwn = true;
				if(n.children) _markOwnership(n.children, n.id, true);
			} else if(parentId === 'PrivateTemplateId') {
				// Private Template 的直接子节点 = 用户分组，判断组名是否匹配当前用户
				var own = (n.text === user_code);
				n._isOwn = own;
				if(n.children) _markOwnership(n.children, n.id, own);
			} else if(inheritOwn !== null) {
				// 已在某个用户分组内部，继承父级归属
				n._isOwn = inheritOwn;
				if(n.children) _markOwnership(n.children, n.id, inheritOwn);
			} else {
				// 其他层级（如 Customized 根节点、PrivateTemplateId/PublicTemplateId 本身）
				if(n.children) _markOwnership(n.children, n.id, null);
			}
		}
	}
	/** 
	* 选择的小站的软件版本发生了变化 ，newSoftwareVer:变化后的软件版本
	* @param newSoftwareVer{string}: 软件版本
	**/
	function hardwareVersionChange(newSoftwareVer) {
		$("#inputMML").val("");
		
		// 1. 刷新操作树
		$.post("${ctx}/cell/param/getOperGroupTree.action", { hardware_version: newSoftwareVer}, function(data) {
			_markOwnership(data, null, null);
			$("#operGroupTree").tree("loadData", data);
			if(data && data.length){
				<%-- 操作名称输入框，初始化下拉面板 --%>
				$.post("${ctx}/cell/param/getOperName4SearchGrid.action", {hardware_version: newSoftwareVer}, function(data) {
					$("#textOperName").combobox("loadData", data);
				},"json");
			}else {
				$("#textOperName").combobox("loadData", []);
			}
		},"json");
		
		// 2. 清空操作表单、操作输入框
		$("#cellParam #paramNodesUl li").remove();
		$("#cellParam #paramNodesUl div").remove();
		$("#textOperName").combo("clear");
	}
	/** 
	* 加载前事件-基站列表
	* @param param{object}: 查询参数
	**/
	function beforeLoad_gridCell_cellParam(param) {
		param["hardware_version"] = hardware_version;
		//查找当前选中的
		if(choosedGroupId > 0){
			param["group_id"] = choosedGroupId;
		}
	}
	// 事件处理-数据表格加载成功
	function gridCellParamDatagridLoadSuccess() {
		$(this).datagrid("fixRownumber");
		$(this).datagrid("enableContextmenuAutoSize");
		var currData = $(this).datagrid("getData");
	}
	
	//添加一个参数路径输入框-用于LST和RMV方法
	function addParampathLSTInputText_elfcell(e, field) {
		//添加新的输入框
		var $namePathDiv = $("<div class='enbMmlItemDiv' style='margin-top: 5px'></div>");
		var $namePathText= $('<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span>');
		var $namePathInput = $("<input type='text' placeholder='<%=rb.getString("JiaoYanGuiZe")%>' name='paramPathNameForElfcellLSTConfig' class='length-adaptation showWholeVal border border-box item'></input>");
		var $namePathButtn = $("<a onclick='rmvParampathLSTInputText_elfcell(this)' class='operationSub'>" 
				+ "<span class='img-suffix el-icon el-icon-minus'></span></a>");
		var $namePathPrompt = $("<p class='prompt'></p>");
		
		$namePathDiv.append($namePathText);
		$namePathDiv.append($namePathInput);
		$namePathDiv.append($namePathButtn);
		$namePathDiv.append($namePathPrompt);
		
		var parentDiv = $("#paramForElfcellLSTDiv");
		$namePathDiv.appendTo(parentDiv);
		
		//将当前输入框后面的图标改为删除图标，并重新绑定事件
		$(e).children(".img-suffix").removeClass('el-icon-minus').addClass('el-icon-plus');
		$(e).attr("onclick", "addParampathLSTInputText_elfcell(this)");

		if(field) {
			$namePathInput.val(field.namePath);
		}
	}
	//删除LST下的参数路径和值输入框
	function rmvParampathLSTInputText_elfcell(e) {
		$(e).parent("div").remove();
	}
	//添加一个参数路径输入框-用于MOD和ADD方法
	function addParampathMODInputText_elfcell(e, field) {
		var $namePathDiv = $("<div class='enbMmlItemDiv' style='margin-top: 5px'></div>");
		var $namePathText= $('<span style="margin-right:10px;width:auto;"><%=rb.getString("CanShuLuJing")%></span>');
		var $namePathInput = $("<input type='text' placeholder='<%=rb.getString("JiaoYanGuiZe")%>' name='paramPathNameForElfcellMODConfig' class='showWholeVal border border-box item'></input>");
		var $paramValueText=$('<span style="margin:0 10px 0 19px;"><%=rb.getString("CanShuZhi")%></span>');
		var $paramValueInput = $("<input type='text' name='paramPathValueForElfcellMODConfig' class='showWholeVal border border-box item'></input>");
		var $namePathButtn = $("<a onclick='rmvParampathMODInputText_elfcell(this)' class='operationSub'>" 
				+ "<span class='img-suffix el-icon el-icon-minus'></span></a>");
		var $namePathPrompt = $("<p class='prompt'></p>");
		
		$namePathDiv.append($namePathText);
		$namePathDiv.append($namePathInput);
		$namePathDiv.append($paramValueText);
		$namePathDiv.append($paramValueInput);
		$namePathDiv.append($namePathButtn);
		$namePathDiv.append($namePathPrompt);
		
		var parentDiv = $("#paramForElfcellADDAndMODDiv");
		$namePathDiv.appendTo(parentDiv);
		
		//将当前输入框后面的图标改为删除图标，并重新绑定事件
		$(e).children(".img-suffix").removeClass('el-icon-minus').addClass('el-icon-plus');
		$(e).attr("onclick", "addParampathMODInputText_elfcell(this)");
		
		if(field) {
			$namePathInput.val(field.namePath);
			$paramValueInput.val(field.value);
		}
	}
	//删除MOD下的参数路径和值输入框
	function rmvParampathMODInputText_elfcell(e) {
		$(e).parent("div").remove();
	}
	//适用于俄罗斯高通平台
	function clickRussiaElfcellGo() {
		var checkResultMsg = validateRussiaElfcellErrBeforeSubmit();
		if (checkResultMsg.length > 0) {
			showMsg('prompt_msg',checkResultMsg);
			return;
		} 
		var smallCells = "";
		var serial_numbers = "";
		// 获取选择的小站
		var selCells = $("#gridCell_cellParam").datagrid("getSelections");
		if (0 == selCells.length) {
			showMsg('prompt_msg',QingXuanZeSheBei);
			return;
		}

		enbmmlVue.resultType = 'res';
		enbmmlVue.isResult = true;	
		// 拼接多个小站编码
		for (var codeNum = 0; codeNum < selCells.length; codeNum++) {
			smallCells += selCells[codeNum]["small_cell_code"] + ",";
			serial_numbers += selCells[codeNum]["serial_number"]+",";
		}
		
		smallCells = smallCells.substring(0, smallCells.length - 1);
		serial_numbers = serial_numbers.substring(0,serial_numbers.length-1);
		
		var objParentObjArray = new Array();
		var paramArray = new Array();
		
		var operName=$("#elfcellConfigParamOperType").val();
		if ("MOD" == operName || "ADD" == operName) {
			var $namePathEle = $("input[name=paramPathNameForElfcellMODConfig]");
			var $valueList = $("input[name=paramPathValueForElfcellMODConfig]");
			
			if("ADD" == operName) {
				$namePathEle = $("input[name=paramPathNameForElfcellLSTConfig]");
			}

			var namePathLength = $namePathEle.length;
			for (var i = 0; i< namePathLength; i++) {
				var obj = new Object();
				var namePath = $namePathEle[i].value;
				var pathValue = $valueList[i]?$valueList[i].value:'';

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
				$namePathEle = $("input[name=paramPathNameForElfcellRMVConfig]");
			} else {
				$namePathEle = $("input[name=paramPathNameForElfcellLSTConfig]");
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
		param["hardwareVersion"] = hardware_version;
		param["operName"] = operName;

		updateActionHistoryContent(serial_numbers, operName, mmlstr);
		
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
	function saveCustomParamCommand(commandName, commandId, nodeName, isPublic) {
		var checkResultMsg = validateRussiaElfcellErrBeforeSubmit();
		if (checkResultMsg.length > 0) {
			showMsg('prompt_msg',checkResultMsg);
			return;
		} 

		var objParentObjArray = new Array();
		var paramArray = new Array();
		
		var operName=$("#elfcellConfigParamOperType").val();
		if ("MOD" == operName || "ADD" == operName) {
			var $namePathEle = $("input[name=paramPathNameForElfcellMODConfig]");
			var $valueList = $("input[name=paramPathValueForElfcellMODConfig]");
			
			if("ADD" == operName) {
				$namePathEle = $("input[name=paramPathNameForElfcellLSTConfig]");
			}

			var namePathLength = $namePathEle.length;
			for (var i = 0; i< namePathLength; i++) {
				var obj = new Object();
				var namePath = $namePathEle[i].value;
				var pathValue = $valueList[i].value;

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
				$namePathEle = $("input[name=paramPathNameForElfcellRMVConfig]");
			} else {
				$namePathEle = $("input[name=paramPathNameForElfcellLSTConfig]");
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
		param["hardwareVersion"] = hardware_version;
		param["operType"] = operName;
		param["commandName"] = commandName;
		param["groupId"] = commandId;
		param["nodeName"] = nodeName;
		param["isPublic"] = isPublic;
		
		$.post("${ctx}/cell/param/saveCustomizedMML.action", param, function (data) {
			if(data.success == true) {
				enbmmlVue.customDlShow = false;
				enbmmlVue.oldCommandName = enbmmlVue.commandForm.commandName;
				enbmmlVue.$message({
					message: '<%=rb.getString("TiShiChengGong")%>',
					type: 'success'
				});

				// 如果是从临时节点保存的，先删除临时节点
				if(enbmmlVue.commandForm.tempNodeId) {
					var tree = $("#operGroupTree");
					var tempNode = tree.tree('find', enbmmlVue.commandForm.tempNodeId);
					if(tempNode) {
						tree.tree('remove', tempNode.target);
					}
					enbmmlVue.commandForm.tempNodeId = null;
				}

				// 保存成功后的节点ID：优先使用后端返回的ID（新建场景），否则使用当前编辑的ID
				var savedGroupId = data.flag || commandId;

				// 刷新指令树（复用初始化加载逻辑，包含测试数据和 _markOwnership）
				hardwareVersionChange(hardware_version);

				// 刷新后默认选中保存的指令
				if(savedGroupId) {
					var tree = $("#operGroupTree");
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
						reviewCustomParamConmmand(savedNode);
					}
				}
			}else {
				enbmmlVue.$message.error(data["message"]);
			}
		}, "json");

	}
	function resetCommandParam() {
		var tabNav = $('span[tabtit="cellParamDetail"]');

		turnTabs(tabNav);
		$('#elfcellConfigParamOperType').trigger('change');
		Object.assign(enbmmlVue.commandForm, {
			isPublic: '0',
			commandName: '',
			groupId: '',
			nodeName: ''
		});
		enbmmlVue.oldCommandName = '';
	}
	// 
	function toADDCommand(nodeName, evt) {
		var tree = $("#operGroupTree");
		
		// 查找父节点（nodeName对应的节点，如 Private Template）
		var parentNode = findParentNodeByName(tree, nodeName);
		if(!parentNode) {
			enbmmlVue.$message.error('Parent node not found');
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
			var tabNav = $('span[tabtit="elfcellConfigParamPanel"]');
			turnTabs(tabNav);
			$('#elfcellConfigParamOperType').trigger('change');
			
			// 设置表单数据
			Object.assign(enbmmlVue.commandForm, {
				commandName: existingTempNode.text,
				groupId: '',
				nodeName: targetNode.text, // 使用实际目标节点的名称
				tempNodeId: existingTempNode.id
			});
			enbmmlVue.oldCommandName = '';
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
			var tabNav = $('span[tabtit="elfcellConfigParamPanel"]');
			turnTabs(tabNav);
			$('#elfcellConfigParamOperType').trigger('change');
			
			// 设置表单数据
			Object.assign(enbmmlVue.commandForm, {
				commandName: 'Untitled',
				groupId: '', // 空表示新建
				nodeName: targetNode.text, // 使用实际目标节点的名称
				tempNodeId: tempNodeId // 记录临时节点ID，保存时用于删除
			});
			enbmmlVue.oldCommandName = '';
		}
	}
	
	// 辅助函数：根据节点名称查找父节点
	function findParentNodeByName(tree, nodeName) {
		var allNodes = tree.tree('getRoots');
		return searchNodeByName(tree, allNodes, nodeName);
	}
	
	function searchNodeByName(tree, nodes, nodeName) {
		for(var i = 0; i < nodes.length; i++) {
			var node = tree.tree('find', nodes[i].id);
			if(node && node.text == nodeName) {
				return node;
			}
			var children = tree.tree('getChildren', nodes[i].target);
			if(children && children.length > 0) {
				var found = searchNodeByName(tree, children, nodeName);
				if(found) return found;
			}
		}
		return null;
	}
	// 自定义参数回显
	function reviewCustomParamConmmand(node) {
		// 如果是 Untitled 指令，设置为编辑模式，显示保存按钮；否则为查看模式，隐藏保存按钮
		if(node.text && node.text.startsWith('Untitled')) {
			isEditMode = true;
		} else {
			isEditMode = false;
		}
		updateSaveButtonVisibility();
		
		// 检查是否为临时节点
		if(node.id && node.id.startsWith('temp_')) {
			// 临时节点，不需要从后台查询数据
			var tabNav = $('span[tabtit="elfcellConfigParamPanel"]');
			turnTabs(tabNav);
			$('#elfcellConfigParamOperType').trigger('change');
			
			Object.assign(enbmmlVue.commandForm, {
				commandName: node.text,
				groupId: '',
				nodeName: '', // 临时节点的父节点名称已经记录在commandForm中
				tempNodeId: node.id
			});
			enbmmlVue.oldCommandName = '';
			return;
		}
		
		var params = {
				groupId: node.id
			};
		
		Object.assign(enbmmlVue.commandForm, {
			commandName: node.text,
			groupId: node.id,
			nodeName: ''
		});
		enbmmlVue.oldCommandName = node.text;

		$.post("${ctx}/cell/param/getCustomizedMMLFields.action", params, function (data) {
			var fieldList = data.pathList || [],
				operType = data.operType || 'LST';

			if(['LST', 'ADD', 'MOD', 'RMV'].includes(operType)) {
				$("#elfcellConfigParamOperType").val(operType).trigger('change');
			}else {
				$("#elfcellConfigParamOperType").val('LST').trigger('change');
			}

			fieldList.map(function(field, index){
				
				if(index == 0) {// 第一组数值映射
					//只留一个输入框 lst or add
					if(['LST', 'ADD'].includes(operType)) {
						var ctner = $("#paramForElfcellLSTDiv");
						$("div:gt(0)", ctner).remove();
						$("div:first input", ctner).val(field.namePath);
					}

					// mod
					if(operType == 'MOD') {
						var ctner = $("#paramForElfcellADDAndMODDiv");
						$("div:gt(0)", ctner).remove();
						$("div:first input:first", ctner).val(field.namePath);
						$("div:first input:last", ctner).val(field.value);
					}
					// rmv
					if(operType == 'RMV') {
						var ctner = $("#paramForElfcellRMVDiv");
						$("div:first input:first", ctner).val(field.namePath);
					}
				}else {// 其他组数值映射
					if(['LST', 'ADD'].includes(operType)) {
						var el = $('#paramForElfcellLSTDiv .enbMmlItemDiv:first a');
						addParampathLSTInputText_elfcell(el, field);
					}

					// mod
					if(operType == 'MOD') {
						var el = $('#paramForElfcellADDAndMODDiv .enbMmlItemDiv:first a');
						addParampathMODInputText_elfcell(el, field);
					}
				}
			});
		}, "json");
	}

	// 全局变量：标记当前是否为编辑模式（点击修改按钮进入）
	var isEditMode = false;

	/**
	 * 复制自定义MML节点
	 * @param el 点击的元素
	 * @param evt 事件对象
	 */
	function copyCusNode(el, evt) {
		var nodeId = $(el).attr('nodeid');
		var tree = $("#operGroupTree");
		var node = tree.tree('find', nodeId);
		
		if(!node) {
			evt.stopPropagation();
			return;
		}

		// 临时节点不支持复制
		if(nodeId && nodeId.startsWith('temp_')) {
			enbmmlVue.$message.warning('Unsaved command cannot be copied');
			evt.stopPropagation();
			return;
		}

		var param = { groupId: nodeId };
		$.post("${ctx}/cell/param/copyCustomizedMML.action", param, function (data) {
			if(data.success == true) {
				enbmmlVue.$message({
					message: '<%=rb.getString("TiShiChengGong")%>',
					type: 'success'
				});

				// 记录复制后新节点的groupId，用于刷新后选中
				var newGroupId = data.flag;

				// 刷新指令树（复用初始化加载逻辑，包含测试数据和 _markOwnership）
				hardwareVersionChange(hardware_version);

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
						reviewCustomParamConmmand(newNode);
					}
				}
			} else {
				enbmmlVue.$message.error(data.message || 'Copy failed');
			}
		}, "json");

		evt.stopPropagation();
	}

	/* ==================== copyCusNode 旧逻辑（已注释） ====================
	function copyCusNode(el, evt) {
		var nodeId = $(el).attr('nodeid');
		var tree = $("#operGroupTree");
		var node = tree.tree('find', nodeId);
		
		if(node) {
			// 切换到参数面板
			var tabNav = $('span[tabtit="elfcellConfigParamPanel"]');
			turnTabs(tabNav);
			
			// 设置为非编辑模式，隐藏保存按钮
			isEditMode = false;
			updateSaveButtonVisibility();
			
			// 复制时重置commandForm，清空groupId表示新建
			Object.assign(enbmmlVue.commandForm, {
				commandName: node.text + '_copy',
				groupId: '', // 清空groupId表示新建
				nodeName: ''
			});
			enbmmlVue.oldCommandName = '';
			
			// 如果是临时节点，直接回显数据
			if(nodeId && nodeId.startsWith('temp_')) {
				$('#elfcellConfigParamOperType').trigger('change');
				evt.stopPropagation();
				return;
			}
			
			// 从后台获取数据并回显
			var params = { groupId: nodeId };
			$.post("${ctx}/cell/param/getCustomizedMMLFields.action", params, function (data) {
				var fieldList = data.pathList || [],
					operType = data.operType || 'LST';

				if(['LST', 'ADD', 'MOD', 'RMV'].includes(operType)) {
					$("#elfcellConfigParamOperType").val(operType).trigger('change');
				}else {
					$("#elfcellConfigParamOperType").val('LST').trigger('change');
				}

				// 回显字段数据
				fieldList.map(function(field, index){
					if(index == 0) {
						if(['LST', 'ADD'].includes(operType)) {
							var ctner = $("#paramForElfcellLSTDiv");
							$("div:gt(0)", ctner).remove();
							$("div:first input", ctner).val(field.namePath);
						}
						if(operType == 'MOD') {
							var ctner = $("#paramForElfcellADDAndMODDiv");
							$("div:gt(0)", ctner).remove();
							$("div:first input:first", ctner).val(field.namePath);
							$("div:first input:last", ctner).val(field.value);
						}
						if(operType == 'RMV') {
							var ctner = $("#paramForElfcellRMVDiv");
							$("div:first input:first", ctner).val(field.namePath);
						}
					}else {
						if(['LST', 'ADD'].includes(operType)) {
							var el = $('#paramForElfcellLSTDiv .enbMmlItemDiv:first a');
							addParampathLSTInputText_elfcell(el, field);
						}
						if(operType == 'MOD') {
							var el = $('#paramForElfcellADDAndMODDiv .enbMmlItemDiv:first a');
							addParampathMODInputText_elfcell(el, field);
						}
					}
				});
			}, "json");
		}
		
		evt.stopPropagation();
	}
	==================== copyCusNode 旧逻辑结束 ==================== */

	/**
	 * 修改自定义MML节点
	 * @param el 点击的元素
	 * @param evt 事件对象
	 */
	function editCusNode(el, evt) {
		var nodeId = $(el).attr('nodeid');
		var tree = $("#operGroupTree");
		var node = tree.tree('find', nodeId);
		
		if(node) {
			// 切换到参数面板
			var tabNav = $('span[tabtit="elfcellConfigParamPanel"]');
			turnTabs(tabNav);
			
			// 设置为编辑模式，显示保存按钮
			isEditMode = true;
			updateSaveButtonVisibility();
			
			// 设置commandForm
			Object.assign(enbmmlVue.commandForm, {
				commandName: node.text,
				groupId: node.id,
				nodeName: ''
			});
			enbmmlVue.oldCommandName = node.text;
			
			// 如果是临时节点，直接回显数据
			if(nodeId && nodeId.startsWith('temp_')) {
				$('#elfcellConfigParamOperType').trigger('change');
				evt.stopPropagation();
				return;
			}
			
			// 从后台获取数据并回显
			var params = { groupId: nodeId };
			$.post("${ctx}/cell/param/getCustomizedMMLFields.action", params, function (data) {
				var fieldList = data.pathList || [],
					operType = data.operType || 'LST';

				if(['LST', 'ADD', 'MOD', 'RMV'].includes(operType)) {
					$("#elfcellConfigParamOperType").val(operType).trigger('change');
				}else {
					$("#elfcellConfigParamOperType").val('LST').trigger('change');
				}

				// 回显字段数据
				fieldList.map(function(field, index){
					if(index == 0) {
						if(['LST', 'ADD'].includes(operType)) {
							var ctner = $("#paramForElfcellLSTDiv");
							$("div:gt(0)", ctner).remove();
							$("div:first input", ctner).val(field.namePath);
						}
						if(operType == 'MOD') {
							var ctner = $("#paramForElfcellADDAndMODDiv");
							$("div:gt(0)", ctner).remove();
							$("div:first input:first", ctner).val(field.namePath);
							$("div:first input:last", ctner).val(field.value);
						}
						if(operType == 'RMV') {
							var ctner = $("#paramForElfcellRMVDiv");
							$("div:first input:first", ctner).val(field.namePath);
						}
					}else {
						if(['LST', 'ADD'].includes(operType)) {
							var el = $('#paramForElfcellLSTDiv .enbMmlItemDiv:first a');
							addParampathLSTInputText_elfcell(el, field);
						}
						if(operType == 'MOD') {
							var el = $('#paramForElfcellADDAndMODDiv .enbMmlItemDiv:first a');
							addParampathMODInputText_elfcell(el, field);
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
	function updateSaveButtonVisibility() {
		var $saveBtn = $('#customMmlSaveBtn');
		if(isEditMode) {
			$saveBtn.show();
		} else {
			$saveBtn.hide();
		}
	}

	function removeCusNode(el, evt) {
		var nodeId = $(el).attr('nodeid');
		
		// 检查是否为临时节点
		if(nodeId && nodeId.startsWith('temp_')) {
			// 直接删除临时节点，无需调用后台
			var tree = $("#operGroupTree");
			var tempNode = tree.tree('find', nodeId);
			if(tempNode) {
				tree.tree('remove', tempNode.target);
				enbmmlVue.$message({
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
							enbmmlVue.$message({
								message: '<%=rb.getString("TiShiChengGong")%>',
								type:'success'
							});

							hardwareVersionChange(hardware_version);

							$('#elfcellConfigParamOperType').trigger('change');
							Object.assign(enbmmlVue.commandForm, {
								commandName: '',
								groupId: '',
								nodeName: ''
							});
							enbmmlVue.oldCommandName = '';
						}else {
							enbmmlVue.$message.error(data["message"]);
						}
					}, "json");
				}
			}
		}).addClass("normalConfirm reSetConfirm");

		evt.stopPropagation();
	}

	//LST,RMV 操作 参数路径不能为空，不能包含{i}，必须包含.，且RMV操作 参数路径必须以.结尾
	function validLstAndRmvNamePath(e){
		var ele = $(e["target"]);
		var currVal = ele.val();
		
		var operType = $("#elfcellConfigParamOperType").val();

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
	function validateRussiaElfcellErrBeforeSubmit() {
		var operType = $("#elfcellConfigParamOperType").val();
		var msg = "";
		var allowSubmit = true;
		if ("MOD" == operType) {
			var $namePathEle = $("input[name=paramPathNameForElfcellMODConfig]");
			var $valueList = $("input[name=paramPathValueForElfcellMODConfig]");
			
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
			var $namePathEle = $("input[name=paramPathNameForElfcellLSTConfig]");
			if ("RMV" == operType) {
				$namePathEle = $("input[name=paramPathNameForElfcellRMVConfig]");
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
	function filterMMLTree(){
		var treeCtn = $('#operGroupTree');
		mmlCommandTreeSearchTxt = $('#groupQueryText').val().trim();
		//hardwareVersionChange(hardware_version);
		treeCtn.tree('doFilter',mmlCommandTreeSearchTxt);
	}
	// 初始化拖拽
	var vLine = document.querySelectorAll('.vertical-line'),
		hLine = document.querySelector('.horizontal-line');

	Array.from(vLine).map(function(line){
	addListener(line,"mousedown",onmousedownH);
	});
	addListener(hLine,"mousedown",onmousedownV);
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
	function addListener(element,type,listener,useCapture){
		element.addEventListener?element.addEventListener(type,listener,useCapture):element.attachEvent("on" + type,listener);
	}
	/* 鼠标点击事件 */
	function onmousedownH(event) {
		var lastX = event.clientX, d = document,
		preItems = document.querySelectorAll('.flex-prev-item'),
		sufItems = document.querySelectorAll('.flex-suff-item');
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
	function onmousedownV(event) {
		var _self = this;
		var lastY = event.clientY, d = document,
		rowItems = document.querySelectorAll('.flex-row-item'),
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
	* @param hardware_version{string}: 版本
	**/
	function modMMEPool(sels,mmlstr,hardware_version){
		
		if(sels.length == 1){
			doExecute(sels,mmlstr,hardware_version);
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
							doExecute(sels,mmlstr,hardware_version);
						}
					}
				}).addClass("normalConfirm reSetConfirm");
			}else {
				doExecute(sels,mmlstr,hardware_version);
			}
		}
	}
	/**
	* 执行命令
	* @param sels{string}: 选中的设备标识
	* @param mmlstr{string}: 命令
	* @param hardware_version{string}: 版本
	**/
	function doExecute(selCells,mmlstr,hardware_version){
		
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
		var paramJson = $('#operValueForm').serializeJson();
		
		<%-- 某个 指标若为空，则表示不对该指标进行设置，从对象中去掉该指标 --%>
		var deleteKeyArr = [];
	
		var param = {};
		param["smallCells"] = smallCells;
		param["serial_numbers"] = serial_numbers;
		param["inputMML"] = encodeURIComponent(mmlstr);
		param["hardwareVersion"] = hardware_version;
		
		updateActionHistoryContent(serial_numbers, "MOD MME",mmlstr);
		
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
	function showMMEPoolWay(flag, row) {
		var sels = [],
			isAllOneMMEPool = true;

		if(row){
			sels = $("#gridCell_cellParam").datagrid("getSelections");
			
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
	function customEventFnc(){
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
		// ['305566','60375','405566','303143'] 
		var msgErTips = ['<%=rb.getString("MiMa3LeiTiShi")%>',
						'<%=rb.getString("MiMaZiFuLianXuTiShi")%>',
						'<%=rb.getString("MiMaZiFuChongFuTiShi")%>',
						'<%=rb.getString("MiMaNianFenTiShi")%>'].join('; ');

		['305566','60375','405566','303143','303142'].map(function(iCode){
			var errItem = $('#' + iCode + '_err');

			if(errItem.length) errItem.text(msgErTips);
		});

		$('#305566,#60375,#405566,#303143').on('blur',function(){
			var val = $(this).val();

			if(val) {
				var boolMap = checkPasswordSpecialRule(val),
					domId = this.id;

				if(boolMap.threeValid && boolMap.orderValid && boolMap.repeatValid && boolMap.yearValid) {
					$(this).removeClass('err_border');
					$('#' + domId + '_err').removeClass('redColor');
				}else{
					$(this).addClass('err_border');
					$('#' + domId + '_err').addClass('redColor');
				}
			}
		})

		// BaiBLQ X2 Flag 4387130 - 4387131
		$('#4387131').parent().hide();
		$('#4387130').on('change',function(){
			var val = $(this).val(),
				$x2IP = $('#4387131'),
				isMod = $('#textOperName').combobox('getValue') == '1078000_MOD' && false;

			if(val == '1') {
				$x2IP.parent().show();

				if(isMod) {
					$(this).attr('disabled', true);
					$x2IP.parent().hide();
				}
			}else {
				$x2IP.val('');
				$x2IP.parent().hide();
			}
		})
		$('#4387131').on('blur',function(){
			var val = $(this).val(),
				flag = $('#4387130').val();

			var domId = this.id;
			if(flag == '1') {
				if(val == '') {
					$(this).addClass('err_border');
					$('#' + domId + '_err').addClass('redColor');
				}
			}else {
				$(this).removeClass('err_border');
				$('#' + domId + '_err').removeClass('redColor');
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
	
	function customCommand(dom) {
		$(dom).attr('iscustom','1');
	}

	function showBatchDL() {
		var dl = $('#mml_device_dl');

		dl.dialog('open');
	}
	function closeBatchDL() {
		var dl = $('#mml_device_dl');

		$('#batch_sn_textarea').val('');
		$('#batch_sn_tips').text('');
		dl.dialog('close');
	}

	function batchInputSN() {
		var serialNumber = $('#batch_sn_textarea').val(),
			//snArr =  serialNumber.split(/[(\r\n)\r\n]+/g),
			list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;}),
			tips = $('#batch_sn_tips');

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
			var tb = $('#gridCell_cellParam'),
				existedRows = tb.datagrid('getSelections'),
				existedSns = existedRows.map(function(row){
					return row.serial_number;
				});

			$.ajax({
				url: '${ctx}/cell/param/getMmlSelectedCellList.action',
				type: 'post',
				data: {
					serialNumbers: list.join(','),
					productType: hardware_version
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
					refreshEnbNum();
				}
			});

			closeBatchDL();
		}
	}

	function lstCkChange(checked) {
		var ids = [];
		
		$('#cellParam #paramNodesUl input[type=checkbox]').each(function(idx, item){
			if(item.checked) ids.push(item.id);
		});
		$('#cellParam #paramNodesUl [name="lstId"]').val(ids.join(','));

		createMML();
	}

	function addQAFACascade() {
		var $model = $('#406176'),
			$switch = $('#406177');

		$model.on('change', function(){
			var value = this.value,
				cas = {
					'MODE1': {show: ['406175','406177','406180','406181','406182','406183'], hide: []},
					'MODE2': {show: ['406178'], hide: ['406175','406177','406180','406181','406182','406183']},
					'MODE3': {show: ['406178'], hide: ['406175','406177','406180','406181','406182','406183']}
				};

			if(value) {
				cas[value].show.map(function(id){
					$('#' + id).trigger('change');

					$('#' + id).parent().show();
				});

				cas[value].hide.map(function(id){
					$('#' + id).parent().hide();
				});
			}
		});

		$switch.on('change', function(){
			var value = this.value,
				cas = {
					'multicast': {show: [], hide: ['406178']},
					'unicast': {show: ['406178'], hide: []}
				};

			if(value) {
				cas[value].show.map(function(id){
					$('#' + id).parent().show();

					$('#' + id).trigger('change');
				});

				cas[value].hide.map(function(id){
					$('#' + id).parent().hide();
				});
			}
		});
	}
</script>