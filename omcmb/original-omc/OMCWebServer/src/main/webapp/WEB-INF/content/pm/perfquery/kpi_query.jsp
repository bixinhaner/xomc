<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style type="text/css">
	.kpiRightHideDiv { position: absolute; width: 900px; height: 100% !important; background: #FFFFFF; right: -2000px;	top: 10px; z-index: 101; }	
	.slideDownAllDiv { position: absolute; top: 0; left: 0; bottom: 0; width: 100%; height: 100%; background: #fff; z-index: 100; display: none; }
	#kpiQuery .KpiperiodSelect { display: inline-block; color: #363B4E; cursor: pointer; width: 80px; height: 30px; line-height: 30px; text-align: center; }
	#kpiQuery .periodSelected {  z-index:10; position: relative; border-color: var(--main-color) !important;
    	background-color: rgba(var(--main-color-rgba1),0.08) !important; color: var(--main-color);border-radius: 100px;}
	#kpiQuery .slide_right{ flex-grow: 1; background: #FFFFFF; position: relative; }
	#kpiQuery .slideTitle { font-size: 14px; font-weight: bold; color: rgba(0, 0, 0, 0.8); padding: 0 20px; height: 46px; line-height: 46px; border-bottom: 1px solid #DFE2EE; }
	#kpiQuery .slide_left { width: 260px; height: 100%; border-right: 1px solid #E9E9E9; }
	#kpiQuery .slide_left .datagrid-header,
	#kpiQuery .slide_left .datagrid-pager { display: block; }
	#kpiQuery .slide_left .datagrid-header { display: none;}
	#kpiQuery .slide_left .datagrid-btable table td { width: 260px; }
	#kpiQuery .slide_left .pagination-info { display: none!important; }
	#kpiQuery .slide_left .datagrid-cell,
	#kpiQuery .slide_left .datagrid-cell-group,
	#kpiQuery .slide_left .datagrid-header-rownumber,
	#kpiQuery .slide_left .datagrid-cell-rownumber { text-overflow: ellipsis; }
	#kpiQuery .datagrid-body,
	#kpiQuery .datagrid-btable {
		width: 260px;
	}
	#kpiQuery .datagrid-mask {
		background: #FFFFFF;
	}
	#kpiQuery .queryHeader{
		width: 100%;
    	height: 36px;
    	line-height: 36px;
    	padding: 0 20px;
    	position: relative;
    	border-bottom: 1px solid #DFE2EE;
	}
	#kpiQuery .templateBox .el-table td {
		border-bottom: 1px solid #DFE2EE;
		padding: 7px 0;
	}
	#kpiQuery .reportFormBox .el-form-item {
		margin-bottom: 20px;
	}
	#kpiQuery .commonSwitch .el-form-item__content{
		margin-left: 20px;
	}
	#kpiQuery .reportFormBox .el-checkbox.is-bordered {
		padding: 4px 28px 0 10px;
		height: 30px;
	}
	#kpiQuery .reportFormBox .el-checkbox__label {
		font-size: 12px;
		color: rgba(0, 0, 0, 0.8);
	}
	#kpiQuery .reportFormBox .el-textarea__inner { height: 100px; resize: none; border-radius: 4px; }
	#kpiQuery .reportTimeBox .el-select>.el-input,
	#kpiQuery .reportTimeBox .el-input,
	#kpiQuery .reportTimeBox .el-input__inner {
		width: 320px;
	}
	#kpiQuery .queryDatePicker {
		width: 200px;
		border-radius: 4px;
	}
	#kpiQuery .queryDatePicker .el-range__close-icon {
		display: none;
		line-height: 20px;
	}
	#kpiQuery .queryDatePicker input[readonly] {
		background-color: #FFFFFF !important;
	}

	#kpiQuery .timeFrameWarp .el-radio-button__inner{
		border: 0 !important;
		padding: 10px 0;
		margin: 0 20px;
		font-size: 12px;
		color: rgba(0, 0, 0, 0.8);
		background: #FFFFFF !important;
	}
	#kpiQuery .timeFrameWarp .el-radio-button__orig-radio:checked+.el-radio-button__inner{
		border-bottom: 2px solid var(--main-color) !important;
		padding: 10px 0;
		background: #FFFFFF !important;
		color: var(--main-color) !important;
	}
	#kpiQuery .el-icon-common-changeOperator:hover,
	#kpiQuery .el-icon-common-changeOperator:active,
	#kpiQuery .el-icon-operation-more-circle:hover,
	#kpiQuery .el-icon-operation-more-circle:active {
		color: var(--main-color) !important;
	}

	#kpiQuery .groupTreeBox .oneGroupEditCls{
		position:absolute;
		right:5px;
		top:5px;
	}
	#kpiQuery .groupTreeBox .oneGroupEditCls .el-icon::before{
		font-size: 14px;
		color: #000000;
	}
	#kpiQuery .templateNameQuery {
		padding: 10px 0;
		border-bottom:1px solid #DFE2EE
	}
	#kpiQuery .templateNameQuery .advanceQuery {
		margin: 0 10px;
		padding: 0 10px;
		height: 24px;
	}
	#kpiQuery .templateNameQuery .advanceQuery .el-input .el-input__inner {
		height: 24px;
		padding: 0;
		line-height: 24px;
	}
	#kpiQuery .queryGroup .el-icon,
	#kpiQuery .templateNameQuery .advanceQuery .el-icon {
		font-size: 14px;
		margin-left: 0 !important;
		line-height: 23px;
	}
	#kpiQuery .templateNameQuery .el-input.el-input--small{
		width: 170px;
	}
	#kpiQuery .mainPageTree .el-tree .el-tree-node .is-leaf + .el-checkbox .el-checkbox__inner {
		display: inline-block;
	}
	#kpiQuery .mainPageTree .el-tree .el-tree-node .el-checkbox .el-checkbox__inner {
		display: none;
	}
	#kpiQuery .mainPageTree .el-tree .el-tree-node__content {
		width: 100%;
		height: 34px;
		line-height: 34px;
		border-bottom: 1px solid;
		border-color: rgba(0, 0, 0, 0.06);
	}

	#kpiQuery .groupTreeBox .oneGroupEditCls{
		position:absolute;
		right:5px;
		top:5px;
	}
	#kpiQuery .groupTreeBox .oneGroupEditCls .el-icon::before{
		font-size: 14px;
		color: #000000;
	}
	#kpiQuery .treeItemBoxCls{
		width: 100%;
	}

	#kpiQuery .kpiItemLabelCls{
		display: inline-block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;

	}
	#kpiQuery .mainPageTree .el-tree-node__children .el-tree-node__content {
		padding-left: 18px !important;
	}
	#kpiQuery .queryGroup {
		height: 24px;
		border-radius: 4px;
		margin: 0 10px;
		padding: 0 10px;}
	#kpiQuery .queryGroup .el-input { width: 200px; }
	#kpiQuery .queryGroup .el-input__inner { width: 200px; height: 24px; line-height: 24px; padding: 0; }
	#kpiQuery .queryGroup .el-icon-common-search { font-size: 14px; }
	#kpiQuery .queryGroup .el-icon-common-search:before { color: #7A7992; }
	.limit-width-80  .el-tabs__nav {
		border: 1px solid #DFE2EE;
	}
	.limit-width-80  .el-tabs--top .el-tabs__item.is-top:last-child {
		border-left: 1 px solid #DFE2EE;
	}

	.slide_right .limit-width-80 .el-tabs {
		width: 100%;
		height: 100%;
	}
	.slide_right > .el-tabs--card {
		position: relative;
		height: 100%;
	}
	.limit-width-80 .el-tabs__nav-wrap {
		max-width: 80%;
	}
	#kpiQuery .pop-filter-wrap {
		height: 24px;
	}
	.slide_right .el-tabs--card>.el-tabs__header .el-tabs__item {
		height: 36px;
		line-height: 36px;
	}

</style>
<div class="overflow-cls">
	<!--性能查询页面  -->
	<div style="height:100%;overflow:hidden; position: relative;min-width: 1300px;width: 100%;" id="kpiQuery" class="kpiQueryPage">
		<!-- 主内容区域-->
		<div class='commonFlex' style='width: 100%; height: 100%;'>
			<div class="commonWarp" style="width: 100%; height: 100%; display: flex; flex:1; position: relative; background: #fff;">
				<div class="slide_left" style='width: 260px; flex: 260px 0 0;'>
					<div class="slideTitle" style="position: relative;">
						<div><%=rb.getString("KPI_XingNengChaXunMoBan")%></div>
						<div>
							<div class="circleIcon CODE_PERFORMANCE_VIEW hidden" style="top:10px;right: 4px; background: #FFFFFF; color: rgba(0, 0, 0, 0.8);">
								<span class="el-icon el-icon-operation-add" @click="newTemplateEnbGnbEgw"></span>
							</div>
						</div>
					</div>
					<div style="height:calc( 100% - 46px); width:260px;" class='groupTreeBox'>
						<!--使用 el-tree 实现模板展示 写5条模拟数据 :default-expanded-keys="defaultexpandedKeys" :default-checked-keys="defaultCheckedKeys"  @node-click="templateNodeClick" -->
						<div class="templateNameQuery">
							<div class='queryGroup '>
								<el-input v-model.trim='queryGroupSearchText' @keyup.enter.native="templateNameQuery" class='pairgrid-query' placeholder='<%=rb.getString("MuBanMingCheng") %>' style='width: 200px;'></el-input>
								<i @click='templateNameQuery' class="el-icon el-icon-common-search"></i>
							</div>
						</div>
						<div style='height:calc( 100% - 48px); width: 260px;overflow: auto;' class="groupTreeBox mainPageTree">
							<el-tree
								ref="groupTemplateTree"
								:data="templateTreeData"
								node-key="tempId"
								:default-expanded-keys="defaultexpandedKeys"
								:default-checked-keys="defaultCheckedKeys"
								:props= "{label:'group_name'}"
								:highlight-current="true">
								<div class="treeItemBoxCls" slot-scope="{ node,data }">
									<div v-if="data.children">
										<span class="kpiItemLabelCls commonSize14" :title="node.label" style='width: 214px;'>{{node.label}}</span>
									</div>
									<div v-else style='height: 34px; line-height: 34px; display: flex;'>
										<div style='width: 150px; margin-right: 10px;display: flex;'>
											<span :title='node.label' style="min-width: 115px;margin-right: 5px;" class="kpiItemLabelCls commonGeneral12">{{node.label}}</span>
											<span title='<%=rb.getString("MoRenMoBan")%>' class='el-icon el-icon-operation-default commonTipSize12' v-if='data.is_default == "true"' style='color:#FF8F1F; line-height: 34px;margin-right:5px;'></span>
											<span title='<%=rb.getString("DingShiBaoBiao")%>' class='el-icon el-icon-status-timeOut commonTipSize12' v-if='data.reportSwitch == "1"' style="line-height: 34px;"></span>
										</div>
										<span v-show="node.parent.data.isPublic == '1' || (node.parent.data.isPublic != '1' && data.isOneSelf == '1') || data.isAdmin == '1'" @click='templateNodeClick(node,data,event)' class='selectDataIcon el-icon el-icon-common-changeOperator'
											style='cursor: pointer;margin-top: 5px; position: absolute;right: 32px;'></span>
										<span class="el-icon el-icon-operation-more-circle" @click="templateOptClick(node,data,event)" v-clickoutside="handerClose" style="cursor: pointer; margin-left: 8px; color: rgba(0, 0, 0, 0.8);margin-top: 8px;position: absolute;right: 8px;"></span>
									</div>
								</div>
							</el-tree>
							<el-cmenu ref="menusTemplate" :data="menusTemplate" @click="clickTemplateMenu"></el-cmenu>
						</div>
					</div>
				</div>
				<!-- table 界面 -->
				<div class="slide_right" id="kpiQueryPanel" style="overflow-x: hidden;flex: 1;">
					<template v-for="(item, index) in templateNameTabs">
						<!--常规图表-->
						<div v-show="item.tempId == templateTabsValue && !item.operatorShow" :id="'chart_circle_bt_' + item.tempId" :key="'chart_' + item.tempId" 
							class="newIconBoxCls-bt" :tip='tableChartTip' :style="{'right': (currentNetworkType == 'egw' ? '60px' : '105px')}" style="top:6px;">
							<span :id="'chart_bt_' + item.tempId" class="el-icon el-icon-circle-chart" @click="openChartWin(this, item.tempId)"></span>
						</div>
						<!--自定义图形图表-->
						<div v-show="item.tempId == templateTabsValue && !item.operatorShow && currentNetworkType != 'egw'" :id="'customCharts_' + item.tempId" :key="'chart_' + item.tempId"
							class="newIconBoxCls-bt" :tip='customTableChartTip' style="top:6px;right:60px;">
							<span :id="'customCharts_bt_' + item.tempId" class="el-icon el-chart_template" @click="openCustomChartsWin(this, item.tempId)"></span>
						</div>
						<!--导出按钮-->
						<div v-show="item.tempId == templateTabsValue" :id="'export_bt_' + item.tempId" :key="'export_' + item.tempId"
							class="kpiQueryExportClass newIconBoxCls-bt" tip='<%=rb.getString("DaoChu")%>' style="top:6px; right: 15px;">
							<span class="el-icon el-icon-operation-export" @click="exportKpiData(item.tempId)"></span>
						</div>
					</template>

					<el-tabs v-model="templateTabsValue" class="limit-width-80" type="card" closable @tab-remove="templateRemoveTab" @tab-click="templateClickTab">
						<el-tab-pane v-for="(item, index) in templateNameTabs"
							:key="item.tempId"
							:label="item.title"
							:name="item.tempId">
							<div :id="'tab_content_' + item.tempId" style='height: 100%; width: 100%; position: absolute; overflow: hidden; top: 0;	left: 0;'>
								<div style="width:100%;height: 100%;">
									<!-- KPI查询toolbar -->
									<div :id="'toolbar_kpiTaskPerfDatagrid_' + item.tempId" class="toolbarContainer">
										<div class='commonFlex' style='border-bottom: 1px solid #DFE2EE; padding-bottom: 10px;'>
											<!--enb 增加设备组 和 设备的切换-->
											<div v-if='currentNetworkType == "enb"'>
												<!--All Operator 只显示设备-->
												<el-radio-group v-if="item.operatorShow"
													:key="'sel_device_type_' + item.tempId"
													v-model='tplTabForms[index].selDeviceType'
													@change='deviceOrGroupChange'
													class="commonRadioButton" style='margin-left: 10px;'>
													<el-radio-button label="2"><%=rb.getString("KPISheBei")%></el-radio-button>
												</el-radio-group>
												<el-radio-group v-else
													:key="'sel_device_type_' + item.tempId"
													v-model='tplTabForms[index].selDeviceType'
													@change='deviceOrGroupChange'
													class="commonRadioButton" style='margin-left: 10px;'>
													<el-radio-button label="2"><%=rb.getString("KPISheBei")%></el-radio-button>
													<el-radio-button label="1"><%=rb.getString("SheBeiZu")%></el-radio-button>
												</el-radio-group>
											</div>

											<div class="queryGroup" style='height: 24px;'>
												<input v-model.trim="tplTabForms[index].searchText" v-if='currentNetworkType == "enb" && "${modelType}" == "S0009"' name="search_text" style="width: 230px; height: 24px;" placeholder='<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%> / <%=rb.getString("ZhanZhiMingCheng")%>'>
												<input v-model.trim="tplTabForms[index].searchText" v-else-if='currentNetworkType == "egw"' name="search_text" style="width: 200px; height: 24px;" placeholder='<%=rb.getString("eGWBianMa")%>/<%=rb.getString("eGWMingCheng")%>'>
												<input v-model.trim="tplTabForms[index].searchText" v-else-if='currentNetworkType == "gnb"' name="search_text" style="width: 300px; height: 24px;"
													placeholder='<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%> / <%=rb.getString("GNBPLMNBiaoShi")%> / NrCGI'>
												<input v-model.trim="tplTabForms[index].searchText" v-else name="search_text" style="width: 230px; height: 24px;" :placeholder='enbPlaceholder'>
												<b class="el-icon el-icon-common-search" @click="vagueQueryKPITaskPerfDatagrid(index)"></b>
											</div>
											<div class='commonFlex'>
												<el-date-picker style='margin-right: 10px;' class="queryDatePicker"
													:key="'time_range_' + item.tempId"
													v-model="tplTabForms[index].dateValue"
													type="daterange"
													range-separator="——"
													@change="dateChange"
													value-format="yyyy-MM-dd"
													:clearable = "false"
													:editable="false"
													:picker-options="pickerOptions"
													start-placeholder='<%=rb.getString("KaiShiShiJian")%>'
													end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
												</el-date-picker>

												<el-popfilter v-if='item.operatorShow'
													type="single"
													label='<%=rb.getString("YunYingShang")%>'
													v-model="tplTabForms[index].operatorParam"
													:list="operatorOptions.map(function(item){ return {label: item.text, value: item.id}})"
													@check-change="operatorChange">
												</el-popfilter>
												<!--设备组都是二级的数据-->
												<el-popfilter v-if='!item.operatorShow && tplTabForms[index].selDeviceType == "2"'
													type="single"
													label='<%=rb.getString("SheBeiZu")%>'
													v-model="tplTabForms[index].deviceGroupParam"
													:list="groupOptions.map(function(item){ return {label: item.text, value: item.id}})"
													@check-change="deviceGroupChange">
												</el-popfilter>

												<el-radio-group
													:key="'time_type_' + item.tempId"
													v-model='tplTabForms[index].periodActive'
													@change='periodActiveChange'
													class="commonRadioButton" style='margin-left: 10px;'>
													<el-radio-button label="15">15Min</el-radio-button>
													<el-radio-button label="60">60Min</el-radio-button>
													<el-radio-button label="1440">24Hour</el-radio-button>
													<el-radio-button label="10080" v-show='enableWeekShow == true && currentNetworkType == "enb"'><%=rb.getString("Zhou")%></el-radio-button>
													<el-radio-button label="43200" v-show='enableMonthShow == true && currentNetworkType == "enb"'><%=rb.getString("Yue")%></el-radio-button>
												</el-radio-group>
											</div>
										</div>
										<div style='height: 36px; margin-bottom: -10px; ' class='commonFlex' v-if='(tplTabForms[index].periodActive == "15" || tplTabForms[index].periodActive == "60") && currentNetworkType == "enb"'>
											<el-radio-group :key="'date_range_' + item.tempId" v-model="tplTabForms[index].curTimeFrame" class='timeFrameWarp' @change='curTimeFrameChange'>
												<el-radio-button v-for="(item,index) in tplTabForms[index].timeFrame" :label="item">{{item}}</el-radio-button>
											</el-radio-group>
										</div>
									</div>

									<!-- table div -->
									<div :id="'winQueryResult_' + item.tempId" style="width:100%;height: 100%;">
										<table :id="'kpiTaskPerfDatagrid_' + item.tempId" class="easyui-datagrid"></table>
									</div>
									<!-- chart图div -->
									<div :id="'winChartCondition_' + item.tempId" style="display:none;width:100%;height:100%;overflow:auto;"></div>
									<!-- 自定义图 表chart -->
									<div :id="'winCustomChartsCondition_' + item.tempId" style="display:none;width:100%;height:100%;overflow:auto;"></div>
								</div>
							</div>
						</el-tab-pane>
					</el-tabs>
					<div id="kpiDrillDiv" class="kpiRightHideDiv slidebarPanel"></div>
				</div>
			</div>
			<div class='rightWarp' style='position: relative; flex: 0 1 360px;' v-show='reportTemplateShow'>
				<div class='rightWarpLayer'>
					<div class='rightBoxHeaderHasTip'>
						<div class='headerText'>
							<span><%=rb.getString("DingShiBaoBiao")%></span>
							<span class='closeIconBox' @click='reportTemplateCancel'><i class='el-icon el-icon-close'></i></span>
						</div>
					</div>
					<div class='rightWarpLayerContent'>
						<el-form ref="reportForm" label-position="top" :model="reportForm" :rules="reportRules" class='reportFormBox' style='padding: 30px 20px;'>
							<el-form-item label='<%=rb.getString("SheZhiKaiGuan") %>' prop='reportStatus' class='commonFlex commonSwitch'>
								<el-switch v-model="reportForm.reportStatus" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
							</el-form-item>
							<el-form-item label='<%=rb.getString("FaSongShiJian")%>' prop="reportTime" class='reportTimeBox'>
								<el-select v-model="reportForm.reportTime" >
									<el-option v-for="item in reportTimeOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
								</el-select>
							</el-form-item>

							<el-form-item label='<%=rb.getString("ZhouQi") %>' prop="reportPeriod">
								<el-checkbox-group v-model="reportForm.reportPeriod">
									<el-checkbox label="day" border><%=rb.getString("Tian") %></el-checkbox>
									<el-checkbox label="houre" border style='margin-left: 20px;'><%=rb.getString("XiaoShi") %></el-checkbox>
									<el-checkbox label="15Min" border style='margin-left: 20px;' v-show='currentNetworkType == "enb"'>15Min</el-checkbox>
								</el-checkbox-group>
							</el-form-item>

							<el-form-item label='<%=rb.getString("XinFaSong") %>' style="margin-bottom: 0;">
								<div :style="{borderColor: reportForm.mailStatus == '1' ? 'red' : '#DFE2EE'}" style="border: 1px solid; padding: 10px; margin-top: 4px; border-radius: 4px">
									<el-checkbox label='<%=rb.getString("YouXiang")%>' v-model='reportForm.mailStatus' true-label="1" false-label="0" @change="mailStatusChange"></el-checkbox>

									<div v-if='reportForm.mailStatus == "1"' style="padding-top: 10px;">
										<el-form-item label='' prop="mailAddress" placeholder="<%=rb.getString("YouXiangShuRuYaoQiu")%>" style='margin-bottom: 15px;'>
											<el-input type="textarea" v-model="reportForm.mailAddress"></el-input>
										</el-form-item>
										<div class='commonNotes12' style="padding-top: 6px;line-height: 18px;"><%=rb.getString("YouXiangDiZhiTiShi")%></div>
									</div>
								</div>
							</el-form-item>

							<div v-if='currentNetworkType != "egw"' :style="{borderColor: reportForm.ftpSwitch == '1' ? 'red' : '#DFE2EE'}" style="border: 1px solid; padding: 10px; margin-top: 10px; border-radius: 4px">
								<el-checkbox label='FTP' v-model='reportForm.ftpSwitch' true-label="1" false-label="0"></el-checkbox>

								<div v-if='reportForm.ftpSwitch == "1"' style="padding-top: 10px;">
									<el-form-item label='<%=rb.getString("FTPXieYi")%>' prop="ftpProtocol" class='reportTimeBox'>
										<el-select v-model="reportForm.ftpProtocol">
											<el-option label='SFTP' value="sftp"></el-option>
											<el-option label='FTP' value="ftp"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ShangChuanLuJing")%>' prop="ftpPath" class='reportTimeBox'>
										<el-input v-model="reportForm.ftpPath"></el-input>
									</el-form-item>

									<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop="ftpIp" class='reportTimeBox'>
										<el-input v-model="reportForm.ftpIp"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("DuanKou")%>' prop="ftpPort" class='reportTimeBox'>
										<el-input v-model="reportForm.ftpPort"></el-input>
									</el-form-item>

									<el-form-item label='<%=rb.getString("YongHuMingCheng")%>' prop="ftpUser" class='reportTimeBox'>
										<el-input v-model="reportForm.ftpUser" maxlength="60"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("MiMa")%>' prop="ftpPassword" class='reportTimeBox'>
										<el-password v-model="reportForm.ftpPassword" show-password placeholder=""></el-password>
										<el-input v-model="reportForm.ftpPassword" style="display: none;"></el-input>
									</el-form-item>
								</div>
							</div>
						</el-form>
					</div>
					<div class='commonFlex commonBorderTop commonFormFotter'>
						<div>
							<el-button type="primary" @click="reportTemplateSubmit"><%=rb.getString("QueDing")%></el-button>
							<el-button @click="reportTemplateCancel"><%=rb.getString("QuXiao")%></el-button>
						</div>
					</div>
				</div>
			</div>
		</div>

		<el-slide ref="templateSlider" :class="contClass" :url="templateSlideUrl" :title="templateSlideTitle" :footer="templateFooterShow" :header='templateHeaderShow'
			:position="templateSlidePosition" :modal="false" :height="templateSliderHeight" :width="templateSliderWidth" @cancel='templateCancelSlider' :cancel-text="'<%=rb.getString("QuXiao")%>'">
		</el-slide>
	</div>
</div>
<!-- kpi导出页面 -->
<div id="exportKpiDiv" class="slideDownAllDiv" style='width: calc(100% - 0px); height: calc(100% - 0px)'></div>

<script>
	/*问题描述： 当前功能仅支持 eNB、gNB、eGW 网元的性能查询
	BUG现象：
	从支持的网元切换到不支持的网元后，之前选中的页签内容不刷新，依然显示之前网元的内容
	1、当前网元 eNB; 打开 Performance 菜单； 页签选中 eNB - KPI View; 
	2、点击菜单 Upgrade， 此时页签选中 eNB - Upgrade；
	3、切换网元CPE, 页签选中 CPE - Upgrade; 
	4、之前选中页签 eNB - KPI View，页面的左侧查询模板数据还在，右侧的 el-tabs 内容模块是空白
	*/

	//内置模板: basic, all operator (标识:isCustomize == '0')
	var tempId = "",//模板ID
		endTime = formatDate(new Date(gloableTime)), //当前时间
		startTimeQuery = formatDate(addDate(Date.getNow(),-6)).substr(0,10), //时间范围：开始时间
		endTimeQuery = formatDate(Date.getNow()).substr(0,10), //时间范围：结束时间
		device_group_id_query = "", // 查询：设备组参数
		perfOperatorCode = "", //查询：运营商参数
		isAllOperatorTemp = false,
		reportCycle = '15', //模板定制粒度,单位（分钟/小时）
		newPeriod,
		periodUnit = '(<%=rb.getString("FenZhong")%>)',
	 	strRandom = Math.random().toString().replace('0.','');
	//new
	if(window.kpiQueryVue) {
		try {
			window.kpiQueryVue.$destroy();
		}catch(e){}
	}
	window.kpiQueryVue = new Vue({
		el:"#kpiQuery",
		data() {
			var vm = this,
				validReportPeriod = function(rule,value,callback){
					if(vm.reportForm.reportStatus == '1' && vm.reportForm.reportPeriod.length == 0){
						callback(new Error('<%=rb.getString("QingXuanZeZhouQi")%>'));
					}else{
						callback();
					}
				},
				validMailAddress = function(rule,value,callback){
					if(vm.reportForm.mailAddress){
						var addresses = vm.reportForm.mailAddress.replace(/\s/g,"").trim();
					}else{
						var addresses = '';
					}

					var reg = /^([a-zA-Z0-9_\.\-])+@([a-zA-Z0-9_-])+(\.[a-zA-Z0-9_-]+)+$/,
						addrArr = addresses.split(";"),
						addrList = [];

					if(addrArr.length > 1){
						addrArr.map(function(str){
							var item = str.trim(),
								lastIdx = item.lastIndexOf(';'),
								length = item.length-1;

							if(lastIdx>=0 && lastIdx == length) {
								addrList.push(item.substring(0,lastIdx));
							}else if(item) {
								addrList.push(item);
							}
						});
					}else{
						//单行
						addrList = addresses.split(';');
					}
					//判断最后一项是否为空 为空删除
					if(addrList[addrList.length-1] == ""){
						addrList.splice(addrList.length-1)
					}

					if(vm.reportForm.reportStatus == '1' && vm.reportForm.mailStatus == '1' && addrList.length == 0){
						callback(new Error('<%=rb.getString("QingShuRuYouXiang")%>'));
					}else if(addresses != null && addresses.length != 0){
						var nameFlag = addrList.every(function(item,index){
							return reg.test(item)
						})
						if(nameFlag){
							callback()
						}else{
							callback(new Error('<%=rb.getString("YouXiangGeShiCuoWu")%>'));
						}
					}else{
						callback();
					}
				},
				// FTP字段条件校验：仅当ftpSwitch开启时必填
				validateFtpPath = (rule,value,callback) => {
					if(vm.reportForm.reportStatus == '1' && vm.reportForm.ftpSwitch == '1'){
						if(!value || value == ''){
							callback(new Error('<%=rb.getString("QingShuRuShangChuanLuJing")%>'))
						}else{
							callback();
						}
					}else{
						callback();
					}
				},
				validateFtpIp = (rule,value,callback) => {
					var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;

					// 开关打开时，校验必填
					if(vm.reportForm.reportStatus == '1' && vm.reportForm.ftpSwitch == '1'){
						if(!value || value == ''){
							callback(new Error('<%=rb.getString("XinIPShuRuTiShi")%>'))
							return;
						}
					}

					// 无论开关状态，如果有值则校验IP格式
					if(value && value != ''){
						if(!reg.test(value)){
							callback(new Error('<%=rb.getString("XinIPGeShiCuoWu")%>'))
							return;
						}
					}

					callback();
				},
				validateFtpPort = (rule,value,callback) => {
					var regNum = /^\d+$/;

					// 开关打开时，校验必填
					if(vm.reportForm.reportStatus == '1' && vm.reportForm.ftpSwitch == '1'){
						if(!value || value == ''){
							callback(new Error('<%=rb.getString("QingShuRuDuanKou")%>'))
							return;
						}
					}

					// 无论开关状态，如果有值则校验端口格式和范围（1-65535）
					if(value && value != ''){
						var portNum = parseInt(value);
						if(!regNum.test(value) || (value < 0 || value > 65535)){
							callback(new Error('<%=rb.getString("ChangDuChaoChuFanWei")%>'))
							return;
						}
					}

					callback();
				},
				validateFtpUser = (rule,value,callback) => {
					if(vm.reportForm.reportStatus == '1' && vm.reportForm.ftpSwitch == '1'){
						if(!value || value == ''){
							callback(new Error('<%=rb.getString("QingShuRuYongHuMing")%>'))
						}else{
							callback();
						}
					}else{
						callback();
					}
				},
				validateFtpPassword = (rule,value,callback) => {
					if(vm.reportForm.reportStatus == '1' && vm.reportForm.ftpSwitch == '1'){
						if(!value || value == ''){
							callback(new Error('<%=rb.getString("QingShuRuMiMa")%>'))
						}else{
							callback();
						}
					}else{
						callback();
					}
				}
		    return {
				lastNetworkType: '',  // 记录上一次的网元类型，用于判断网元是否真正改变
				tplTabForms: [],
				templateNameTabs: [],
				templateTabsValue: '',
				templateTreeData:[],
				defaultexpandedKeys:[],
				defaultCheckedKeys:[],
				queryGroupSearchText:'',
		    	slidePosition: 'top',
			    slideTitle: '',
			    slideUrl: '',
			    sliderHeight: '100%',
			    sliderWidth: '100%',
			    footerShow: true,
			    headerShow: true,
			    contClass:'',
			    periodSwitch:'0',
			    reportCycle: '15',
			    selectTemplateName: '',
			    periodActive: '15',
			    enableWeekShow: false,
			    enableMonthShow: false,
				//数据保留天数限制
				holdTimeFor15m: 30,
				holdTimeFor60m: 30,
				holdTimeFor24h: 365,
				holdTimeForWeek: 365,
				holdTimeForMonth: 365,
			    templateUrl: '',
			    menusTemplate: [],// 左侧菜单数据
			    rowDataTemplate: {},
				rowDataTemplateNode: {},
			    reportForm: {
			    	reportStatus: '0',
			    	reportPeriod: [],
			    	mailStatus: '0',
			    	mailAddress: '',
			    	reportTime: 0,

					ftpSwitch: '0',
					ftpUser: '',
					ftpIp: '',
					ftpPort: '',
					ftpPath: '',
					ftpPassword: '',
					ftpProtocol: 'sftp',
			    },
			    reportRules: {
					reportTime: [ { required: true, message: '<%=rb.getString("QingXuanZeShiJian")%>', trigger: 'change' } ],
			    	reportPeriod: [ { required: true, validator: validReportPeriod }],
			    	mailAddress: [{ validator: validMailAddress }],
					ftpProtocol: [ { required: true, message: '<%=rb.getString("QingXuanZeFTPXieYi")%>', trigger: 'change' } ],
					ftpPath: [
						{ required: true, validator: validateFtpPath }
					],
					ftpIp: [
						{ required: true, validator: validateFtpIp }
					],
					ftpPort: [
						{ required: true, validator: validateFtpPort }
					],
					ftpUser: [
						{ required: true, validator: validateFtpUser }
					],
					ftpPassword: [
						{ required: true, validator: validateFtpPassword }
					],
			    },
			    reportTimeOptions: [
					{value:0,label:"00:00"},
					{value:1,label:"01:00"},
					{value:2,label:"02:00"},
					{value:3,label:"03:00"},
					{value:4,label:"04:00"},
					{value:5,label:"05:00"},
					{value:6,label:"06:00"},
					{value:7,label:"07:00"},
					{value:8,label:"08:00"},
					{value:9,label:"09:00"},
					{value:10,label:"10:00"},
					{value:11,label:"11:00"},
					{value:12,label:"12:00"},
					{value:13,label:"13:00"},
					{value:14,label:"14:00"},
					{value:15,label:"15:00"},
					{value:16,label:"16:00"},
					{value:17,label:"17:00"},
					{value:18,label:"18:00"},
					{value:19,label:"19:00"},
					{value:20,label:"20:00"},
					{value:21,label:"21:00"},
					{value:22,label:"22:00"},
					{value:23,label:"23:00"}
				],
				mailAddress: '',
				reportTemplateShow: false,
				templateSlidePosition: 'top',
				templateSlideTitle: '',
				templateSlideUrl: '',
				templateSliderHeight: '100%',
				templateSliderWidth: '100%',
				templateFooterShow: true,
				templateHeaderShow: true,
				commonTemplateSelectRow: [],
				groupOptions:[],
				operatorOptions: [],
				deviceGroupParam: '',
				operatorParam: '',
				dateValue:[startTimeQuery,endTimeQuery],
				operatorShow: false,
				deviceGroupShow: false,
				timeOptionRange: '',
				// 日期选择限制 当前日期开始向前7天
				pickerOptions:{
					disabledDate(time){
						// 禁用未来日期
						var today = new Date();
						today.setHours(23, 59, 59, 999);
						if(time.getTime() > today.getTime()){
							return true;
						}

						var timeOptionRange = vm.timeOptionRange;
						var periodActive = kpiQueryVue.periodActive;

						if(periodActive == '15' || periodActive == '60'){
							// 15/60分钟粒度：可选范围限制在 holdTimeFor 天数内，但在此范围内可选择任意7天
							var maxRangeDays = 60 * 60 * 24 * 7 * 1000 - 1; // 可选择的最大跨度：7天
							var holdTimeDays = periodActive == '15'
								? 60 * 60 * 24 * vm.holdTimeFor15m * 1000
								: 60 * 60 * 24 * vm.holdTimeFor60m * 1000;

							// 限制整体可选范围不超过数据保留天数
							if(time.getTime() < today.getTime() - holdTimeDays){
								return true;
							}

							// 当已选择第一个日期时，限制第二个日期的范围为7天内
							if(timeOptionRange){
								return time.getTime() > timeOptionRange.getTime() + maxRangeDays ||
								       time.getTime() < timeOptionRange.getTime() - maxRangeDays;
							}

							// 未选择第一个日期时，在数据保留范围内的日期都可选
							return false;
						}else{
							// 24小时/周/月粒度：根据各自的数据保留天数进行限制
							var holdTimeDays, maxRangeDays;

							if(periodActive == '1440'){ // 24小时粒度
								holdTimeDays = 60 * 60 * 24 * vm.holdTimeFor24h * 1000;
								maxRangeDays = 60 * 60 * 24 * vm.holdTimeFor24h * 1000 - 1;
							}else if(periodActive == '10080'){ // 周粒度
								holdTimeDays = 60 * 60 * 24 * vm.holdTimeForWeek * 1000;
								maxRangeDays = 60 * 60 * 24 * vm.holdTimeForWeek * 1000 - 1;
							}else if(periodActive == '43200'){ // 月粒度
								holdTimeDays = 60 * 60 * 24 * vm.holdTimeForMonth * 1000;
								maxRangeDays = 60 * 60 * 24 * vm.holdTimeForMonth * 1000 - 1;
							}

							// 限制整体可选范围不超过数据保留天数
							if(time.getTime() < today.getTime() - holdTimeDays){
								return true;
							}

							// 当已选择第一个日期时，限制第二个日期的范围
							if(timeOptionRange){
								return time.getTime() > timeOptionRange.getTime() + maxRangeDays ||
								       time.getTime() < timeOptionRange.getTime() - maxRangeDays;
							}

							// 未选择第一个日期时，在数据保留范围内的日期都可选
							return false;
						}
				 	},
				 	//选中日期会执行的回调
				 	onPick(time){
				 		//当第一时间选中才设置禁用
			 			if(time.minDate && !time.maxDate){
				 			vm.timeOptionRange = time.minDate;
				 		}
				 		if(time.maxDate){
				 			vm.timeOptionRange = null;
				 		}
				 	}
				},
				timeFrame: [],
				curTimeFrame: '',
				firstFlag: true,
				enbPlaceholder: '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>',
				tableChartTip: '<%=rb.getString("TuBiao")%>',
				customTableChartTip: '<%=rb.getString("ZiDingYiTuBiao")%>',
				levelType: ''
		    }
		 },
		computed:{
			// 合并后的唯一网元标识计算属性
			currentNetworkType() {
				let fullValue = '';
				//遍历 sysMain.$refs.nav.editableTabs 获取当前所有打开的tab
				if(sysMain.$refs.nav && sysMain.$refs.nav.editableTabs && sysMain.$refs.nav.editableTabs.length > 0){
					sysMain.$refs.nav.editableTabs.map(function(item,index){
						//根据 id 判断是否是网元标识相关的 tab enb、gnb、egw
						if(item.id == '900001' || item.id == '900005' || item.id == '900007'){	
												
							fullValue = item.netType;
						}					
					});
				}
				//如果fullValue 和 sysMain.headType 都有值，优先返回 fullValue
				return fullValue && fullValue != '' ? fullValue : sysMain.headType;
			},
			isAdmin() {
				return is_super_user == 'true';
			}
		},
		watch:{
			// 统一监听 currentNetworkType 变化
			currentNetworkType(newType, oldType) {
				var vm = this;
				// 当前功能仅支持 eNB、gNB、eGW 网元的性能查询
				if(newType !== 'enb' && newType !== 'gnb' && newType !== 'egw') {
					// 不支持的网元：什么都不做，保持页面当前状态
					return;
				}

				// 如果网元类型没有真正改变（如 eNB → CPE → eNB），不需要重新初始化
				if(vm.lastNetworkType === newType) {
					
					return;
				}

				// 支持的网元且类型确实改变了：重新初始化
				vm.lastNetworkType = newType;
				
				vm.queryGroupSearchText = '';
				vm.reportTemplateShow = false;
				
				// 安全地调用组件方法，避免组件未加载时出错
				if(vm.$refs.templateSlider) {
					vm.$refs.templateSlider.hide();
				}
				
				// 安全地调用全局函数
				try {
					closeKpiDrillDiv();
					closeKpiExportDiv();
				} catch(e) {
					
				}
				
				vm.firstFlag = true; //网元切换时，重新获取设备与设备组选中值
				
				// 重新初始化页面（init方法内部会清空并重新加载数据）
				vm.init();
			},
			"reportForm.reportPeriod":function(newVal){
				if(newVal.length > 0){
					this.$refs.reportForm.clearValidate('reportPeriod');
				}else{
					this.$refs.reportForm.validateField('reportPeriod');
				}
			}
		},
		methods:{
			// 辅助方法：只 resize 当前页面的 datagrid，避免影响其他页面
			resizeDatagrid() {
				try {
					$('#kpiQuery table.datagrid-f').datagrid('resize');
				} catch(e) {
				}
			},
			templateRemoveTab(targetName){
				var vm = this,
					nameTabs = vm.templateNameTabs;
					activeName = vm.templateTabsValue;
				if(nameTabs.length == 1){
					//保留一个模板不可手动删除
					vm.templateTabsValue = activeName;
					return
				}else{
					var switchToOtherTab = false; //标记是否需要切换到其他Tab
					if(activeName == targetName) {
						switchToOtherTab = true;
						nameTabs.forEach((tab, index) => {
							if(tab.tempId == targetName) {
						    	let nextTab = nameTabs[index+1] || nameTabs[index-1];
						   		if(nextTab) {
							   		activeName = nextTab.tempId;
						   		}
							}
					  	});
					}

					vm.templateTabsValue = activeName;
					vm.templateNameTabs = nameTabs.filter(tab => tab.tempId != targetName);

					vm.tplTabForms = vm.tplTabForms.filter(item => item.tempId != targetName);
					
					//关闭当前Tab后，如果切换到了其他Tab，需要触发Tab点击事件以更新状态
					if(switchToOtherTab){
						vm.$nextTick(() => {
							//查找切换后的Tab信息
							var switchedTab = vm.templateNameTabs.filter(tab => tab.tempId == activeName)[0];
							if(switchedTab){
								//手动触发Tab点击事件，模拟用户点击Tab的行为
								vm.templateClickTab({
									name: switchedTab.tempId,
									label: switchedTab.title
								});
							}
						});
					}else{
						//关闭的不是当前Tab，只需更新左侧树选中状态
						vm.$refs.groupTemplateTree.setCurrentKey(activeName);
						try{
							//$('table.datagrid-f').datagrid('resize');
							vm.resizeDatagrid()
							window.dispatchEvent(new Event('resize'));
						}	catch(e){}
					}
				}
			},
			//点击tab 不重新调用接口
			templateClickTab(tabItem){
				var vm = this;
				if(tabItem && tabItem.name){
					vm.$refs.groupTemplateTree.setCurrentKey(tabItem.name);
					//tab 切换，重新赋值
					tempId = tabItem.name;
					vm.selectTemplateName = tabItem.label;
				}

				//查询一下性能模板的详情
				vm.getKPITempInfo();
				vm.$nextTick(() => {
					try{
						//$('table.datagrid-f').datagrid('resize');
						vm.resizeDatagrid()
						window.dispatchEvent(new Event('resize'));

					}	catch(e){}
				})
			},
			//获取模板详情，设置对应的图表周期 如果模板上报周期为"24h"，则取消"天"周期,并获取图表数据
			getKPITempInfo(){
				var vm = this,
					curTemplateInfoUrl = '',
					curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
					params = {
						tempId: curForm.tempId,
						timeZone: timeZone
					};

				if(vm.currentNetworkType == 'enb'){
					curTemplateInfoUrl = '${ctx}/pm/template/getTemplateInfo.action';
				}else if(vm.currentNetworkType == 'gnb'){
					curTemplateInfoUrl = '${ctx}/gnb/pm/template/getTemplateInfo.action';
				}else if(vm.currentNetworkType == 'egw'){
					curTemplateInfoUrl = '${ctx}/egw/pm/template/getTemplateInfo.action';
				}

				axios.post(curTemplateInfoUrl, stringify(params)).then(function(response){
					var data = response.data;

					//0-天  1-周  2-月
					if(data){
						//当前表格粒度与data.reportPeriod 相同，则将天粒度隐藏，默认选中周粒度
						reportCycle = data.reportPeriod;
						if(vm.currentNetworkType == 'enb'){
							//模板关联的层级类型, 控制 plmn id 的显示和隐藏
							vm.levelType = data.indicatorLevel;
							if(data.selDeviceType == '2'){
								if(vm.levelType == 'device'){
									vm.enbPlaceholder = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>';						
								}else{
									vm.enbPlaceholder = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%> / <%=rb.getString("GNBPLMNBiaoShi")%>';
								}
							}else{
								kpiQueryVue.enbPlaceholder = '<%=rb.getString("SheBeiZuMingCheng")%>'
							}
							
						}
					}
				})
			},
			// 初始化
			init(){
				var vm = this;
				// 清空所有相关状态数据，防止网元切换或菜单切换后数据残留
				vm.templateTreeData = [];
				vm.templateNameTabs = [];
				vm.tplTabForms = [];
				vm.templateTabsValue = '';
				vm.defaultexpandedKeys = [];
				vm.defaultCheckedKeys = [];
				vm.timeFrame = [];
				
				//初始化获取模板数据
				vm.queryTemplateList();

				//获取设备组
				axios.post('${ctx}/system/deviceGroup/getDeviceGroupForCombobox.action').then(function(response){
					let data = response.data
					vm.groupOptions = data;
				}).catch(function(error){})
				//获取运营商数据
				axios.post('${ctx}/system/operator/getOperatorListForCombobox.action').then(function(response){
					let data = response.data
					vm.operatorOptions = data;
				}).catch(function(error){})

				// #106346 4G,5G,eGW此接口共用
				axios.get("${ctx}/pm/settings/getDataHoldTime.action").then(function(response){
					var data = response.data;

					if(data){
						vm.holdTimeFor15m = data.holdTimeFor15m;
						vm.holdTimeFor60m = data.holdTimeFor60m;
						vm.holdTimeFor24h = data.holdTimeFor24h;
						vm.holdTimeForWeek = data.holdTimeForWeek;
						vm.holdTimeForMonth = data.holdTimeForMonth;
					}
				}).catch(function(error){})
			},
			//获取模板数据
			queryTemplateList(){
				var vm = this, templateTreeUrl = '', start = vm.dateValue[0], end = vm.dateValue[1], num = differ(end,start),
					params = {
						searchText: '',
					};

				if(vm.currentNetworkType == 'enb'){
					templateTreeUrl = '${ctx}/template/grouping/getGroupingTemplateList.action';
					//网元：enb, 粒度：15min,60min，才显示时间轴; ;映射已选时间范围的时间轴
				   	if(vm.periodActive == '15' || vm.periodActive == '60'){
						for(var i = 0; i<=num; i++) {
							var dateStr = dateformatter(addDate(start,i));
							vm.timeFrame.push(dateStr.substr(0,10));
						}
						vm.curTimeFrame = end;
					}

				  	//获取当前是否支持周和月的开关; 只有 enb网元 显示周和月粒度
				   	axios.post('${ctx}/pm/template/getKpiWeekAndMonthSwitch.action').then(function(response){
						var data = response.data;
						vm.periodSwitch = data.periodSwitch;
						//复选框未选中， 不显示 week month
						if( data.periodSwitch == '0'){
						    vm.enableWeekShow = false;
						    vm.enableMonthShow = false;
						}else{
						    vm.enableWeekShow = true;
						    vm.enableMonthShow = true;
						}
					}).catch(function(error){});

				}else if(vm.currentNetworkType == 'gnb'){
					templateTreeUrl = '${ctx}/template/grouping/getGnbGroupingTemplateList.action';
				}else if(vm.currentNetworkType == 'egw'){
					templateTreeUrl = '${ctx}/template/grouping/getWcgGroupingTemplateList.action';
				}
				axios.post(templateTreeUrl, stringify(params)).then(function(response){
					let data = response.data;

					if(data && data.rows.length>0){
						vm.templateTreeData = data.rows;

						vm.treeTemplateLoad(vm.templateTreeData);
					}else{
						vm.templateTreeData = [];
					}
				}).catch(function(error){})
			},
			//左侧模板列表加载成功时,切换分页也会重新调用该方法，所以出现#67810所述问题
			treeTemplateLoad(data){
				var vm = this, defaultRow = '', publicTemplateDefaultRow = '';
				// 循环 data，data 数组层级关系中， is_default 字段： true, 则为默认模板， groupTemplateTree 树则选中该节点
				//默认选中的模板
				if(data && data.length > 0){

					if(vm.templateNameTabs.length>=10) {
						vm.$message({
							message: '<%=rb.getString("XingNengZuiDaDuoKaiTiShi")%>',
							type:'warning'
						})
						return;
					}

					data.map(function(item,index){
						var itemTemplate = item; // Private Template 和 Public Template 都携带 isPublic 等字段的层级
						//0-私有 1-公共
						if(itemTemplate.isPublic == '0'){
							//向 itemTemplate 中增加一个 tempId 的字段，用于树节点的唯一标识
							itemTemplate.tempId = 'private' + itemTemplate.isPublic;
							//私有模板-有分组时，则需将 Private Template  这一父级分组展开
							vm.$nextTick(function(){
								vm.defaultexpandedKeys.push( vm.$refs.groupTemplateTree.getNode(itemTemplate.tempId));
							});
						}

						if(item.children.length > 0){
							//私有模板属于多层级，里面包含不同用户分组， 公共模板层级较单一
							item.children.map(function(childItem,childIndex){
								//childItem：私有模板的用户名称 + 公共模板中的模板名称（admin分组 + basic 层级）
								//私有模板分组：用户分组名称 与 登录用户 名称相同，且该分组下有数据，则展开
								if(childItem.group_name == user_code && itemTemplate.isPublic == '0'){
									if(childItem.children.length > 0){
										//此分组有数据，则展开其父级; 默认展开与登录用户同名的分组
										vm.$nextTick(function(){
											vm.defaultexpandedKeys.push( vm.$refs.groupTemplateTree.getNode(childItem.children[0].tempId).parent);
										});
										//将该分组中is_default为true的节点 ，设置为默认模板
										childItem.children.map(function(grandChildItem,grandChildIndex){
											if(grandChildItem.is_default == 'true'){
												defaultRow = grandChildItem;
											}
										})
									}
								}else{
									//不是登录用户同名的分组，或者是私有模板分组 或者是公共模板时

									//公共模板层级关系中， is_default 字段： true, 则为默认模板
									if(itemTemplate.isPublic == '1'){
										if(childItem.is_default == 'true'){
											defaultRow = childItem;
										}else{
											//公共模板：当前无默认标识，则默认选中第一个
											if(itemTemplate.children.length > 0){
												publicTemplateDefaultRow = itemTemplate.children[0];
											}
										}
									}
								}
							})
							//如果有默认模板，取默认模板
							if(defaultRow && defaultRow.is_default === "true") {
								vm.$nextTick(function(){
									vm.$refs.groupTemplateTree.setCurrentKey(defaultRow.tempId); //默认选中的节点
									vm.defaultexpandedKeys.push( vm.$refs.groupTemplateTree.getNode(defaultRow.tempId).parent);
								});

								vm.commonTemplateSelectRow = defaultRow;
							}else{
								vm.$nextTick(function(){
									vm.$refs.groupTemplateTree.setCurrentKey(publicTemplateDefaultRow.tempId); //默认选中的节点
									vm.defaultexpandedKeys.push(vm.$refs.groupTemplateTree.getNode(publicTemplateDefaultRow.tempId).parent);
								});

								vm.commonTemplateSelectRow = publicTemplateDefaultRow;
							}

							if(vm.commonTemplateSelectRow && vm.templateNameTabs.length == 0){
								//右侧多页签映射数据
								var tabIds = vm.templateNameTabs.map(function(item){
									return item.tempId;
								});
								if(!tabIds.includes(vm.commonTemplateSelectRow.tempId)) {
									vm.templateNameTabs.push({
										title: vm.commonTemplateSelectRow.group_name,
										tempId: vm.commonTemplateSelectRow.tempId,
										content: ' ',
										operatorShow: vm.commonTemplateSelectRow.isAllOperator == '1'
									})

									vm.tplTabForms.push({
										tempId: vm.commonTemplateSelectRow.tempId,
										searchText: '',
										operatorParam: '',
										deviceGroupParam: '',
										dateValue: [startTimeQuery,endTimeQuery],
										periodActive: reportCycle,
										curTimeFrame: endTimeQuery, //年月日
										timeFrame: [],
										selDeviceType: '' //1-设备组 2-设备
									})

									var start = startTimeQuery, end = endTimeQuery, num = differ(end,start);

									//当前24小时，非15 60min，15Min，如果时间范围小于等于7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴；
									for(var i = 0; i<=num; i++) {
										var dateStr = dateformatter(addDate(start,i));
										vm.tplTabForms[vm.tplTabForms.length-1].timeFrame.push(dateStr.substr(0,10));
									}
								}
								vm.selectTemplateName = vm.commonTemplateSelectRow.group_name;
								vm.templateTabsValue = vm.commonTemplateSelectRow.tempId;

								tempId = vm.commonTemplateSelectRow.tempId;

								closeKpiDrillDiv();
								//isAllOperator : local 版，无All Operator 模板； cloud 版 才有  各个网元都显示设备组或运营商
								if(vm.commonTemplateSelectRow.isAllOperator == '1') {
									isAllOperatorTemp = true;
									vm.deviceGroupShow = false;
									vm.deviceGroupParam = '';
									vm.operatorShow = true;
									$('#chart_circle_bt').hide();
									$('#customCharts_').hide();

								}else{
									isAllOperatorTemp = false;
									$('#chart_circle_bt').show();
									$('#customCharts_').show();
									vm.deviceGroupShow = true;
									vm.operatorShow = false;
									vm.operatorParam = '';
								}

								// 搜索条件置空
								device_group_id_query = vm.deviceGroupParam;
								perfOperatorCode = vm.operatorParam;
								newPeriod='';

								$('[_echarts_instance_]').resize();
								// 图表页面加载   4G,5G此接口共用
								if($("#winQueryResult_" + vm.templateTabsValue).is(":visible") && false){
									$('#winChartCondition_'+ vm.templateTabsValue).addClass('loading');
									$("#winChartCondition_"+ vm.templateTabsValue).load('${ctx}/pm/template/viewTemplateChart.action?randomValue='+strRandom,function(data){
										$.parser.parse(this);
										$('#winChartCondition_'+ vm.templateTabsValue).removeClass('loading');
									});
									//new 新图形图表页面
									$('#winCustomChartsCondition_'+ vm.templateTabsValue).addClass('loading');
									$("#winCustomChartsCondition_" + vm.templateTabsValue).load('${ctx}/pm/chart/template/goKpiChartTemplatePage?randomValue='+strRandom,function(data){
										$.parser.parse(this);
										$('#winCustomChartsCondition_'+ vm.templateTabsValue).removeClass('loading');
									});
								}
							}else{
								//左侧模板没数据时
								vm.commonTemplateSelectRow = [];
								//左侧默认模板，加载右侧表格数据
								closeKpiDrillDiv();
								tempId = '';

								// 搜索条件置空
								device_group_id_query = vm.deviceGroupParam;
								perfOperatorCode = vm.operatorParam;
								newPeriod='';

								$('[_echarts_instance_]').resize();
								if($("#winQueryResult_"+ vm.templateTabsValue).is(":visible") && false){
									$('#winChartCondition_'+ vm.templateTabsValue).addClass('loading');
									$("#winChartCondition_"+ vm.templateTabsValue).load('${ctx}/pm/template/viewTemplateChart.action?randomValue='+strRandom,function(data){
										$.parser.parse(this);
										$('#winChartCondition_'+ vm.templateTabsValue).removeClass('loading');
									});
									//new 新图形图表页面
									$('#winCustomChartsCondition_'+ vm.templateTabsValue).addClass('loading');
									$("#winCustomChartsCondition_" + vm.templateTabsValue).load('${ctx}/pm/chart/template/goKpiChartTemplatePage?randomValue='+strRandom,function(data){
										$.parser.parse(this);
										$('#winCustomChartsCondition_'+ vm.templateTabsValue).removeClass('loading');
									});
								}
							}
						}else{
							//左侧模板没数据时
							vm.commonTemplateSelectRow = [];
							//左侧默认模板，加载右侧表格数据
							closeKpiDrillDiv();
							tempId = '';

							// 搜索条件置空
							device_group_id_query = vm.deviceGroupParam;
							perfOperatorCode = vm.operatorParam;
							newPeriod='';

							$('[_echarts_instance_]').resize();
							if($("#winQueryResult_"+ vm.templateTabsValue).is(":visible") && false){
								$('#winChartCondition_'+ vm.templateTabsValue).addClass('loading');
								$("#winChartCondition_"+ vm.templateTabsValue).load('${ctx}/pm/template/viewTemplateChart.action?randomValue='+strRandom,function(data){
									$.parser.parse(this);
									$('#winChartCondition_'+ vm.templateTabsValue).removeClass('loading');
								});

								//new 新图形图表页面
								$('#winCustomChartsCondition_'+ vm.templateTabsValue).addClass('loading');
								$("#winCustomChartsCondition_" + vm.templateTabsValue).load('${ctx}/pm/chart/template/goKpiChartTemplatePage?randomValue='+strRandom,function(data){
									$.parser.parse(this);
									$('#winCustomChartsCondition_'+ vm.templateTabsValue).removeClass('loading');
								});
							}
						}
					})

					immediateQuery();
				}else{
					//左侧模板没数据时
					vm.commonTemplateSelectRow = [];
					//左侧默认模板，加载右侧表格数据
					closeKpiDrillDiv();
					tempId = '';

					// 搜索条件置空
					device_group_id_query = vm.deviceGroupParam;
					perfOperatorCode = vm.operatorParam;
					newPeriod='';
					//根据模板加载右侧表格数据
					immediateQuery();
					// 图表页面加载   4G,5G此接口共用
					$('[_echarts_instance_]').resize();
					if($("#winQueryResult_"+ vm.templateTabsValue).is(":visible") && false){
						$('#winChartCondition_'+ vm.templateTabsValue).addClass('loading');
						$("#winChartCondition_"+ vm.templateTabsValue).load('${ctx}/pm/template/viewTemplateChart.action?randomValue='+strRandom,function(data){
							$.parser.parse(this);
							$('#winChartCondition_'+ vm.templateTabsValue).removeClass('loading');
						});
						//new 新图形图表页面
						$('#winCustomChartsCondition_'+ vm.templateTabsValue).addClass('loading');
						$("#winCustomChartsCondition_" + vm.templateTabsValue).load('${ctx}/pm/chart/template/goKpiChartTemplatePage?randomValue='+strRandom,function(data){
							$.parser.parse(this);
							$('#winCustomChartsCondition_'+ vm.templateTabsValue).removeClass('loading');
						});
					}
				}
			},

			//搜索 只更新 左侧树形结构， 不更新右侧列表
			templateNameQuery(val){
				var vm = this, templateTreeUrl = '', params = {};

				params.searchText = vm.queryGroupSearchText;
				if(vm.currentNetworkType == 'enb'){
					templateTreeUrl = '${ctx}/template/grouping/getGroupingTemplateList.action';
				}else if(vm.currentNetworkType == 'gnb'){
					templateTreeUrl = '${ctx}/template/grouping/getGnbGroupingTemplateList.action';
				}else if(vm.currentNetworkType == 'egw'){
					templateTreeUrl = '${ctx}/template/grouping/getWcgGroupingTemplateList.action';
				}
				axios.post(templateTreeUrl, stringify(params)).then(function(response){
					let data = response.data;

					if(data && data.rows.length > 0){
						vm.templateTreeData = data.rows;

						vm.templateTreeData.map(function(item,index){
							var itemTemplate = item;
							//0-私有 1-公共
							if(itemTemplate.isPublic == '0'){
								//向itemTemplate 中增加一个 tempId 的字段，用于树节点的唯一标识
								itemTemplate.tempId = 'private' + itemTemplate.isPublic;
								//默认展开私有模板这一层级
								vm.$nextTick(function(){
									vm.defaultexpandedKeys.push( vm.$refs.groupTemplateTree.getNode(itemTemplate.tempId));
								});
							}else{
								//默认展开公共模板这一层级
								itemTemplate.tempId = 'public' + itemTemplate.isPublic;
								vm.$nextTick(function(){
									vm.defaultexpandedKeys.push( vm.$refs.groupTemplateTree.getNode(itemTemplate.tempId));
								});
							}
							if(item.children.length > 0){
								item.children.map(function(childItem,childIndex){
									if(childItem.group_name == user_code && itemTemplate.isPublic == '0'){
										if(childItem.children.length > 0){
											//此分组有数据，则展开其父级
											//默认展开与登录用户同名的分组
											vm.$nextTick(function(){
												vm.defaultexpandedKeys.push( vm.$refs.groupTemplateTree.getNode(childItem.children[0].tempId).parent);
											});
										}
									}
								})
							}
						})
						if(vm.commonTemplateSelectRow){
							vm.$nextTick(function(){
								vm.$refs.groupTemplateTree.setCurrentKey(vm.commonTemplateSelectRow.tempId); //默认选中的节点
								vm.defaultexpandedKeys.push(vm.$refs.groupTemplateTree.getNode(vm.commonTemplateSelectRow.tempId).parent);
							});
						}
					}else{
						vm.templateTreeData = [];
					}
				}).catch(function(error){})
			},
			//模板切换后需更新右侧列表数据
			templateNodeClick(node,row,ev){
				var vm = this;

				vm.firstFlag = true; //切换左侧模板时，该值恢复需调用接口，根据接口返回值进场选中设备或设备组
				closeKpiDrillDiv();
				if(vm.templateNameTabs.length > 0){
					var isExist = false;
					vm.templateNameTabs.map(function(item,index){
						if(item.tempId == row.tempId){
							isExist = true;
						}
					})
					if(!isExist){
						if(vm.templateNameTabs.length>=10) {
							vm.$message({
								message: '<%=rb.getString("XingNengZuiDaDuoKaiTiShi")%>',
								type:'warning'
							})
							return;
						}

						vm.templateNameTabs.push({
							title: row.group_name,
							tempId: row.tempId,
							content: ' ',
							operatorShow: row.isAllOperator == '1'
						})

						vm.tplTabForms.push({
							tempId: row.tempId,
							searchText: '',
							operatorParam: '',
							deviceGroupParam: '',
							dateValue: [startTimeQuery,endTimeQuery],
							periodActive: reportCycle,
							curTimeFrame: endTimeQuery,
							timeFrame: [],
							selDeviceType: '' //1-设备组 2-设备
						})

						var start = startTimeQuery, end = endTimeQuery, num = differ(end,start);

						//当前24小时，非15 60min，15Min，如果时间范围小于等于7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴；
						for(var i = 0; i<=num; i++) {
							var dateStr = dateformatter(addDate(start,i));
							vm.tplTabForms[vm.tplTabForms.length-1].timeFrame.push(dateStr.substr(0,10));
						}
					}

				}else{
					vm.templateNameTabs.push({
						title: row.group_name,
						tempId: row.tempId,
						content: ' ',
						operatorShow: row.isAllOperator == '1'
					})

					vm.tplTabForms.push({
						tempId: row.tempId,
						searchText: '',
						operatorParam: '',
						deviceGroupParam: '',
						dateValue: [startTimeQuery,endTimeQuery],
						periodActive: reportCycle,
						curTimeFrame: endTimeQuery,
						timeFrame: [],
						selDeviceType: '' //1-设备组 2-设备
					})

					var start = startTimeQuery, end = endTimeQuery, num = differ(end,start);

					//当前24小时，非15 60min，15Min，如果时间范围小于等于7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴；
					for(var i = 0; i<=num; i++) {
						var dateStr = dateformatter(addDate(start,i));
						vm.tplTabForms[vm.tplTabForms.length-1].timeFrame.push(dateStr.substr(0,10));
					}
				}
				vm.templateTabsValue = row.tempId;
				//表格自适应
				vm.$nextTick(() => {
					try{
						vm.resizeDatagrid()
						//$('table.datagrid-f').datagrid('resize');
						window.dispatchEvent(new Event('resize'));
					}	catch(e){}
				})


				vm.commonTemplateSelectRow = row;
				vm.selectTemplateName = row.group_name;
				// 开始结束时间也需还原为当天时间-近一周的时间范围
				vm.dateValue = [startTimeQuery,endTimeQuery];
				var start = vm.dateValue[0], end = vm.dateValue[1], num = differ(end,start);

				//当前24小时，非15 60min，15Min，如果时间范围小于等于7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴；
				vm.timeFrame = [];
				for(var i = 0; i<=num; i++) {
					var dateStr = dateformatter(addDate(start,i));
					vm.timeFrame.push(dateStr.substr(0,10));
				}
				if(vm.currentNetworkType == 'enb'){
					vm.curTimeFrame = vm.timeFrame[vm.timeFrame.length-1]; // 当前选中最后一个日期
				}else{
					vm.curTimeFrame = '';
				}

				//isAllOperator : local 版，无All Operator 模板； cloud 版 才有
				if(row.isAllOperator == '1') {
					isAllOperatorTemp = true;
					$('#chart_circle_bt').hide();
					$('#customCharts_').hide();
					vm.deviceGroupShow = false;
					vm.deviceGroupParam = '';
					vm.operatorShow = true;
				}else{
					isAllOperatorTemp = false;
					$('#chart_circle_bt').show();
					$('#customCharts_').show();
					vm.deviceGroupShow = true;
					vm.operatorShow = false;
					vm.operatorParam = '';
				}
				if(row){
					tempId = row.tempId;
				}

				device_group_id_query = vm.deviceGroupParam;
				perfOperatorCode = vm.operatorParam;
				newPeriod='';

				//根据模板加载右侧表格数据
				immediateQuery();
				// 图表页面加载   4G,5G此接口共用
				if($("#winQueryResult_"+ vm.templateTabsValue).is(":visible")){
					$('#winChartCondition_'+ vm.templateTabsValue).addClass('loading');
					$("#winChartCondition_"+ vm.templateTabsValue).load('${ctx}/pm/template/viewTemplateChart.action?randomValue='+strRandom,function(data){
						$.parser.parse(this);
						$('#winChartCondition_'+ vm.templateTabsValue).removeClass('loading');
					});

					//new 新图形图表页面
					$('#winCustomChartsCondition_'+ vm.templateTabsValue).addClass('loading');
					$("#winCustomChartsCondition_" + vm.templateTabsValue).load('${ctx}/pm/chart/template/goKpiChartTemplatePage?randomValue='+strRandom,function(data){
						$.parser.parse(this);
						$('#winCustomChartsCondition_'+ vm.templateTabsValue).removeClass('loading');
					});
				}
			},

			//--------------------------------------- 右侧 列表数据
			//设备或设备组切换 1-设备组 2-设备
			deviceOrGroupChange(val){
				var vm = this;

				if(val == '2'){
					if(vm.levelType == 'device'){
						vm.enbPlaceholder = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>';						
					}else{
						vm.enbPlaceholder = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%> / <%=rb.getString("GNBPLMNBiaoShi")%>';
					}
				}else{
					vm.enbPlaceholder = '<%=rb.getString("SheBeiZuMingCheng")%>'
				}
				vm.$nextTick(function(){
					immediateQuery();

					try{
						vm.resizeDatagrid()
						//$('table.datagrid-f').datagrid('resize');
						  window.dispatchEvent(new Event('resize'));
					  }	catch(e){}
				})
			},
			//粒度切换时，查询时间范围不变；全局注意粒度参数
			periodActiveChange(val){
				var vm = this;

				newPeriod = val;
				if(vm.currentNetworkType == 'enb'){
					var curForm = vm.tplTabForms.filter((item) => { return item.tempId == vm.templateTabsValue })[0];

					var start = curForm.dateValue[0], end = curForm.dateValue[1], num = differ(end,start);

					//当前24小时，非15 60min，15Min，如果时间范围小于等于7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴；
					vm.timeFrame = [];
					curForm.timeFrame = [];
					for(var i = 0; i<=num; i++) {
						var dateStr = dateformatter(addDate(start,i));
						vm.timeFrame.push(dateStr.substr(0,10));

						curForm.timeFrame.push(dateStr.substr(0,10));
					}

					if(val == '15' || val == '60'){
						//当前24小时，非15 60min，如果时间范围超出7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴并给出提示；
						if(vm.timeFrame.length > 7){
							vm.timeFrame = vm.timeFrame.slice(-7); //时间轴
							vm.dateValue = [vm.timeFrame[0],vm.timeFrame[vm.timeFrame.length-1]]; // 时间范围选择框需重新赋值；
							vm.curTimeFrame = vm.timeFrame[vm.timeFrame.length-1]; // 当前选中最后一个日期

							curForm.timeFrame = curForm.timeFrame.slice(-7);
							curForm.dateValue = [curForm.timeFrame[0], curForm.timeFrame[curForm.timeFrame.length-1]];
							curForm.curTimeFrame = curForm.timeFrame[curForm.timeFrame.length-1]; // 当前选中最后一个日期

							showMsg('prompt_msg','<%=rb.getString("KPIShiJianFanWeiTiShi")%>');
						}else {
							//当前时间轴只有一天时，则为重新赋值，反之，按当前选中日期查询
							if(vm.timeFrame.length == 1){
								vm.curTimeFrame = vm.timeFrame[0];
							}else{
								vm.curTimeFrame = vm.timeFrame[vm.timeFrame.length-1]; // 当前选中最后一个日期
							}
							//当前时间轴只有一天时，则为重新赋值，反之，按当前选中日期查询
							curForm.curTimeFrame = curForm.timeFrame[curForm.timeFrame.length-1];
						}
					}
				}

				// 45841 问题解决
				var opts = $("#kpiTaskPerfDatagrid_" + vm.templateTabsValue).datagrid('options');
				opts.pageNumber = 1;

				vm.$nextTick(function(){
					immediateQuery();
				})

				try{
					vm.resizeDatagrid()
              		//$('table.datagrid-f').datagrid('resize');
              	  	window.dispatchEvent(new Event('resize'));
          	  	}	catch(e){}
			},
			//运营商改变更新列表
			operatorChange(val){
				var vm = this, curStartTime = '', curEndTime = '', tableParams = {};

				var curForm = vm.tplTabForms.filter((item) => { return item.tempId == vm.templateTabsValue })[0];

				isAllOperatorTemp = true;
				perfOperatorCode = val;

				if(vm.currentNetworkType == 'enb'){
					if(curForm.periodActive == '15' || curForm.periodActive == '60'){
						curStartTime = curForm.curTimeFrame + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(curStartTime),1));
					}else{
						curStartTime = curForm.dateValue[0] + ' 00:00:00';
						var endTime = curForm.dateValue[1] + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(endTime),1));
					}
				}else{
					curStartTime = curForm.dateValue[0] + ' 00:00:00';
					var endTime = curForm.dateValue[1] + ' 00:00:00';
					curEndTime = dateformatter(addDate(new Date(endTime),1));
				}

				//selDeviceType: 1-设备组 2-设备
				if(vm.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
					tableParams = {
						timeZone: timeZone,
						tempId: curForm.tempId,
						searchText: curForm.searchText,
						groupId : '',
						startTime: curStartTime,
						endTime: curEndTime,
						period: curForm.periodActive,
						operatorCode: curForm.operatorParam
					}
				}else{
					tableParams = {
						timeZone :timeZone,
						tempId : curForm.tempId,
						searchText : curForm.searchText,
						groupId : '', // 设备组参数置空
			            serialNumber : '', //version_9.0.0 改版后，该高级查询携带参数无用
			            hostName : '', //version_9.0.0 改版后，该高级查询携带参数无用
			            startTime : curStartTime,
			            endTime : curEndTime,
			            reportPeriod: curForm.periodActive,
			            operatorCode: curForm.operatorParam
					}
				}

				$("#kpiTaskPerfDatagrid_" + vm.templateTabsValue).datagrid({
			        queryParams : tableParams,
			        pageNumber : 1
				});
			},
			//设备组改变更新列表
			deviceGroupChange(val){
				var vm = this, curStartTime = '', curEndTime = '', tableParams = {};

				var curForm = vm.tplTabForms.filter((item) => { return item.tempId == vm.templateTabsValue })[0];

				isAllOperatorTemp = false;
				device_group_id_query = val;
				if(vm.currentNetworkType == 'enb'){
					if(curForm.periodActive == '15' || curForm.periodActive == '60'){
						curStartTime = curForm.curTimeFrame + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(curStartTime),1));
					}else{
						curStartTime = curForm.dateValue[0] + ' 00:00:00';
						var endTime = curForm.dateValue[1] + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(endTime),1));
					}
				}else{
					curStartTime = curForm.dateValue[0] + ' 00:00:00';
					var endTime = curForm.dateValue[1] + ' 00:00:00';
					curEndTime = dateformatter(addDate(new Date(endTime),1));
				}
				//selDeviceType: 1-设备组 2-设备
				if(vm.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
					tableParams = {
						timeZone: timeZone,
						tempId: curForm.tempId,
						searchText: curForm.searchText,
						groupId : curForm.deviceGroupParam,
						startTime: curStartTime,
						endTime: curEndTime,
						period: curForm.periodActive,
						operatorCode: ''
					}
				}else{
					tableParams = {
						timeZone :timeZone,
						tempId : curForm.tempId,
						searchText : curForm.searchText,
						groupId : curForm.deviceGroupParam,
			            serialNumber : '',
			            hostName : '',
			            startTime : curStartTime,
			            endTime : curEndTime,
			            reportPeriod: curForm.periodActive,
			            operatorCode: '' // 运营商参数置空
					}
				}
				$("#kpiTaskPerfDatagrid_" + vm.templateTabsValue).datagrid({
			        queryParams : tableParams,
			        pageNumber : 1
				});
			},
			//时间范围改变时，需更新时间轴，当前的开始时间，结束时间为选择日期的结束时间
			dateChange(val) {
				var vm = this, curStartTime = '', curEndTime = '';

				if(vm.currentNetworkType == 'enb'){
					var curForm = vm.tplTabForms.filter((item) => { return item.tempId == vm.templateTabsValue })[0];

					//开始，结束时间可以保持一致；
					vm.dateValue = val;
					var start = curForm.dateValue[0], end = curForm.dateValue[1], num = differ(end,start);

					//当前24小时，非15 60min，15Min，如果时间范围小于等于7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴；
					vm.timeFrame = [];
					curForm.timeFrame = [];
					for(var i = 0; i<=num; i++) {
						var dateStr = dateformatter(addDate(start,i));
						vm.timeFrame.push(dateStr.substr(0,10));

						curForm.timeFrame.push(dateStr.substr(0,10));
					}

					if(vm.periodActive == '15' || vm.periodActive == '60'){
						//当前24小时，非15 60min，如果时间范围超出7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴并给出提示；
						if(vm.timeFrame.length > 7){
							vm.timeFrame = vm.timeFrame.slice(-7); //时间轴
							vm.dateValue = [vm.timeFrame[0],vm.timeFrame[vm.timeFrame.length-1]]; // 时间范围选择框需重新赋值；

							vm.curTimeFrame = vm.timeFrame[vm.timeFrame.length-1]; // 当前选中最后一个日期

							curForm.timeFrame = curForm.timeFrame.slice(-7);
							curForm.dateValue = [curForm.timeFrame[0], curForm.timeFrame[curForm.timeFrame.length-1]];

							curForm.curTimeFrame = curForm.timeFrame[curForm.timeFrame.length-1]; // 当前选中最后一个日期

							vm.commonReloadTable();

							showMsg('prompt_msg','<%=rb.getString("KPIShiJianFanWeiTiShi")%>');
						}else {
							//当前时间轴只有一天时，则为重新赋值，反之，按当前选中日期查询
							if(vm.timeFrame.length == 1){
								vm.curTimeFrame = vm.timeFrame[0];
							}else{
								vm.curTimeFrame = vm.timeFrame[vm.timeFrame.length-1]; // 当前选中最后一个日期
							}

							//当前时间轴只有一天时，则为重新赋值，反之，按当前选中日期查询
							curForm.curTimeFrame = curForm.timeFrame[curForm.timeFrame.length-1];

							vm.commonReloadTable();
						}
					}else{
						//enb 24hour week month 粒度
						vm.commonReloadTable();
					}
				}else{
					//gnb wcg
					vm.commonReloadTable();
				}
			},
			commonReloadTable(){
				var vm = this,  tableParams = {};
				var curForm = vm.tplTabForms.filter((item) => { return item.tempId == vm.templateTabsValue })[0];

				if(curForm) {
					var curStartTime = curForm.dateValue[0] + ' 00:00:00';
					var endTime = curForm.dateValue[1] + ' 00:00:00';
					var curEndTime = dateformatter(addDate(new Date(endTime),1));

					if(vm.currentNetworkType == 'enb'){
						if(curForm.periodActive == '15' || vm.periodActive == '60') {
							curStartTime = endTime;
						}
					}

					//selDeviceType: 1-设备组 2-设备
					if(vm.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
						tableParams = {
							timeZone: timeZone,
							tempId: curForm.tempId,
							searchText: curForm.searchText,
							groupId : curForm.deviceGroupParam,
							startTime: curStartTime,
							endTime: curEndTime,
							period:  curForm.periodActive,
							operatorCode: curForm.operatorParam
						}
					}else{
						tableParams = {
							timeZone : timeZone,
							tempId : curForm.tempId,
							searchText : curForm.searchText,
							groupId : curForm.deviceGroupParam,
							serialNumber : '',
							hostName : '',
							startTime : curStartTime,
							endTime : curEndTime,
							reportPeriod: curForm.periodActive,
							operatorCode: curForm.operatorParam
						}
					}

					$("#kpiTaskPerfDatagrid_" + vm.templateTabsValue).datagrid({
						queryParams : tableParams,
						pageNumber : 1
					});
				}
			},
			// 时间轴选择改变时更新列表
			curTimeFrameChange(val){
				var vm = this,
					tableParams = {};

				vm.curTimeFrame = val;

				var curForm = vm.tplTabForms.filter((item) => { return item.tempId == vm.templateTabsValue })[0],
					newVal = curForm.curTimeFrame;

				if(newVal != null || newVal != '') {
					// 结束日期在当前选中时间轴基础上加上一天；为下一天的0点： 2023-01-29 00:00:00 - 2023-01-30 00:00:00
					var curEndTime = dateformatter(addDate(new Date(newVal),1));

					if(vm.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
						tableParams = {
							timeZone: timeZone,
							tempId: curForm.tempId,
							searchText: curForm.searchText,
							groupId : curForm.deviceGroupParam,
							startTime : newVal + ' 00:00:00',
							endTime : curEndTime,
							period:  curForm.periodActive,
							operatorCode: curForm.operatorParam
						}
					}else{
						tableParams = {
							timeZone : timeZone,
							tempId : curForm.tempId,
							searchText : curForm.searchText,
							groupId : curForm.deviceGroupParam,
							serialNumber : '',//无用参数
							hostName : '',//无用参数
							startTime :  newVal + ' 00:00:00',
							endTime : curEndTime,
							reportPeriod: curForm.periodActive,
							operatorCode: curForm.operatorParam
						}
					}

					$("#kpiTaskPerfDatagrid_" + vm.templateTabsValue).datagrid({
				        queryParams : tableParams,
				        pageNumber : 1
					});
				}
			},

			//----------------------------------------  左侧模板操作项
			//kpi view: 新建模板
			newTemplateEnbGnbEgw(){
				var vm = this;
				vm.contClass = 'commonBorderSlide';
		    	vm.templateSlideUrl = '${ctx}/cell/perfmgmt/kpitemp/goAddCustomQueryTemplatePage.action';
		    	vm.templateSlideTitle = '<%=rb.getString("XinJianMuBan")%>';
		    	vm.templateSlidePosition = 'left';
		    	vm.templateSliderHeight = '100%';
		    	vm.templateSliderWidth = '100%';
		    	vm.templateFooterShow = false;
		    	vm.templateHeaderShow = true;
		    	vm.$refs.templateSlider.showSlide(function(){
		    		//当前行数据，网元 code，操作类型
		    		eventBus.$emit('action-templateInit','','add');
		    	});
			},
	 	 	//kpi view 关闭侧滑页
	 	    templateCancelSlider(){
	 	    	var vm = this;
	 	    	vm.$refs.templateSlider.hide();
	 	    },
			//表格操作项
			templateOptClick(node,data,ev){
				var vm = this, disableFlag = "", deleteTempFlag = false, defaultFlag = false, setDefaultFlag = false, opdisable = false;
				vm.rowDataTemplate = data;
				vm.rowDataTemplateNode = node;
				var isDefault = data.is_default == "true" ? "true" : "false",
					isAllOperator = data.isAllOperator == '1';
				if(data.isCustomize == '0' || isDefault === "true"){
					deleteTempFlag = false; //0-不可删除
				}else{
					deleteTempFlag = true;
				}
				if(isDefault === "true") {
					defaultFlag = true;
				}else{
					defaultFlag = false;
				}
				//已为默认则置灰； 所有运营商也置灰
				if(isAllOperator) {
					opdisable = true;
				}
				//该用户可查看同运营商其它用户分组的模板; isOneSelf 仅支持复制模版  0-不是自己，只有复制模板权限   1 是自己, 说明有操作权限
				//isAdmin 1-管理员 0-非管理员, 是管理员有所有权限
				if(data.isOneSelf == '0' && node.parent.data.isPublic != '1' && data.isAdmin == '0') {
					vm.menusTemplate = [
						{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",disable:disableFlag,code:'infoOper'},
						{label:'<%=rb.getString("FuZhiMuBan")%>',cls:"el-icon el-icon-operation-copy CODE_PERFORMANCE_VIEW hidden",code:'copyTemplateOper',disable: isAllOperator}
					]
				}else{
					vm.menusTemplate= [
						{label:'<%=rb.getString("YiWeiMoRen")%>',cls:"el-icon el-icon-operation-default CODE_PERFORMANCE_VIEW hidden",code:'defaultOper',show: defaultFlag,disable: opdisable},
						{label:'<%=rb.getString("SheWeiMoRen")%>',cls:"el-icon el-icon-operation-setDefault CODE_PERFORMANCE_VIEW hidden",code:'setDefaultOper',show: !defaultFlag,disable: opdisable},
						{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",disable:disableFlag,code:'infoOper'},
						{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_PERFORMANCE_VIEW hidden",disable:disableFlag,code:'modifyOper'},
						{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_PERFORMANCE_VIEW hidden",code:'delOper',show: deleteTempFlag,disable: opdisable},
						{label:'<%=rb.getString("FuZhiMuBan")%>',cls:"el-icon el-icon-operation-copy CODE_PERFORMANCE_VIEW hidden",code:'copyTemplateOper',disable: isAllOperator},
						{label:'<%=rb.getString("DaoChuMuBan")%>',cls:"el-icon el-icon-operation-export",code:'exportOper'},
						{label:'<%=rb.getString("DingShiBaoBiao")%>',cls:"el-icon el-icon-operation-report CODE_PERFORMANCE_VIEW hidden",code:'reportOper',}
					]
				}

				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menusTemplate.show(ev);
				});
				event.stopPropagation();
			},
			//单点击方法 -- 设备组列表操作
			clickTemplateMenu(ev){
				var vm = this;
				var codes = {
					defaultOper: vm.defaultIconClick,
					setDefaultOper: vm.setDefaultIconClick,
					infoOper: vm.infoIconClick,
					modifyOper: vm.modifyIconClick,
					delOper: vm.delIconClick,
					copyTemplateOper: vm.copyTemplateClick,
					exportOper: vm.exportIconClick,
					reportOper: vm.reportIconClick,
				}
				if(codes[ev.code]){
					codes[ev.code](vm.rowDataTemplate, vm.rowDataTemplateNode)
				}
			},
			//已为默认
			defaultIconClick(row){},
			//设置为默认模板
			setDefaultIconClick(row, node){
				var vm = this, curSetDefaultUrl = '', curUserCode = '',
					params = {
						tempId: row.tempId,
						is_default: true
					};
				//原逻辑 creator 未入库，增加此条件验证 creator 是否为空，为空则取 updator
				if(row.creator == '' || row.creator == null || row.creator == undefined){
					params.creator = row.updator;
				}else{
					params.creator = row.creator;
				}
				if(vm.currentNetworkType == 'enb'){
					curSetDefaultUrl = '${ctx}/pm/template/setDefaultTemplate.action';
				}else if(vm.currentNetworkType == 'gnb'){
					curSetDefaultUrl = '${ctx}/gnb/pm/template/setDefaultTemplate.action';
				}else{
					curSetDefaultUrl = '${ctx}/egw/pm/template/setDefaultTemplate.action';
				}
				//#76662 超级管理员 可将其它用户下的私有模板改为默认模板，
				//如果操作其它用户的私有模板为默认模板，则需提示用户；改变的是其它用户的默认模板， admin 的默认模板不变
				//admin 操作公共模板和 admin下的模板为默认，则不需提示
				//isPublic: 0-私有模板 1-公共模板
				//老版本 没有对 creator 进行数据存储，所以此处判断 creator 为空时，取 updator
				//判断user_code 是否为admin，admin 分组也不可提示
				if(user_code.toLowerCase() == 'admin'){
					curUserCode = 'admin';
				}
				if(vm.isAdmin == true && node.parent.data.isPublic != '1' && node.parent.data.group_name.toLowerCase() != curUserCode){
					vm.$confirm('<%=rb.getString("KPICaoZuoYingXiangQiTaYongHuDeMuBan")%>', '<%=rb.getString("QueRen")%>', {
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type: 'warning'
					}).then(() => {
						axios.post(curSetDefaultUrl, stringify(params)).then(function(response){
							var data = response.data;
							if(data) {
								if(data["success"]){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type:'success'
									});
									vm.queryTemplateList();
								}else{
									vm.$message.error(data["message"])
								}
							}
						}).catch(function(error){})
					}).catch(() => {});
				}else{
					axios.post(curSetDefaultUrl, stringify(params)).then(function(response){
						var data = response.data;
						if(data) {
							if(data["success"]){
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								});
								vm.queryTemplateList();
							}else{
								vm.$message.error(data["message"])
							}
						}
					}).catch(function(error){})
				}
			},
			//详情
			infoIconClick(row){
				var vm = this;

				vm.contClass = 'commonBorderSlide';
		    	vm.templateSlideUrl = '${ctx}/cell/perfmgmt/kpitemp/goAddCustomQueryTemplatePage.action';
		    	vm.templateSlideTitle = '<%=rb.getString("XinXi")%>';
		    	vm.templateSlidePosition = 'left';
		    	vm.templateSliderHeight = '100%';
		    	vm.templateSliderWidth = '100%';
		    	vm.templateFooterShow = false;
		    	vm.templateHeaderShow = true;
		    	vm.$refs.templateSlider.showSlide(function(){
		    		//当前行数据，网元 code，操作类型
		    		eventBus.$emit('action-templateInit', row, 'view');
		    	});
			},
			//修改模板
			modifyIconClick(row){
				var vm = this;

				vm.contClass = 'commonBorderSlide';
		    	vm.templateSlideUrl = '${ctx}/cell/perfmgmt/kpitemp/goAddCustomQueryTemplatePage.action';
		    	vm.templateSlideTitle = '<%=rb.getString("XiuGai")%>';
		    	vm.templateSlidePosition = 'left';
		    	vm.templateSliderHeight = '100%';
		    	vm.templateSliderWidth = '100%';
		    	vm.templateFooterShow = false;
		    	vm.templateHeaderShow = true;
		    	vm.$refs.templateSlider.showSlide(function(){
		    		//当前行数据，网元 code，操作类型
		    		eventBus.$emit('action-templateInit', row, 'modify');
		    	});
			},
			//复制模板
			copyTemplateClick(row){
				var vm = this;

				vm.contClass = 'commonBorderSlide';
		    	vm.templateSlideUrl = '${ctx}/cell/perfmgmt/kpitemp/goAddCustomQueryTemplatePage.action';
		    	vm.templateSlideTitle = '<%=rb.getString("XinJianMuBan")%>';
		    	vm.templateSlidePosition = 'left';
		    	vm.templateSliderHeight = '100%';
		    	vm.templateSliderWidth = '100%';
		    	vm.templateFooterShow = false;
		    	vm.templateHeaderShow = true;
		    	vm.$refs.templateSlider.showSlide(function(){
		    		//当前行数据，网元 code，操作类型
		    		eventBus.$emit('action-templateInit', row, 'copy');
		    	});
			},
			//删除模板
			delIconClick(row){
				var vm = this, curDeleteTemplateUrl = '', params = {};

				params.tempId = row.tempId;
				//原逻辑 creator 未入库，增加此条件验证 creator 是否为空，为空则取 updator
				if(row.creator == '' || row.creator == null || row.creator == undefined){
					params.creator = row.updator;
				}else{
					params.creator = row.creator;
				}

				if(vm.currentNetworkType == 'enb'){
					curDeleteTemplateUrl = '${ctx}/pm/template/delTemplate.action';
				}else if(vm.currentNetworkType == 'gnb'){
					curDeleteTemplateUrl = '${ctx}/gnb/pm/template/delTemplate.action';
				}else if(vm.currentNetworkType == 'egw'){
					curDeleteTemplateUrl = '${ctx}/egw/pm/template/delTemplate.action';
				}

				$.messager.confirm('<%=rb.getString("QueRen")%>', '<%=rb.getString("QueRenShanChuMuBan")%>', function (r) {
			        if (r) {
			           axios.post(curDeleteTemplateUrl, stringify(params)).then(function(response){
							var data = response.data;
							if(data) {
								if(data["success"]){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type:'success'
									});
									//刪除模板时， 如果vm.templateNameTabs 数组中存在该模板，则需要在vm.templateNameTabs 数组中将其删除
									var nameTabs = vm.templateNameTabs, activeName = vm.templateTabsValue;

									if(activeName == row.tempId) {
										nameTabs.forEach((tab, index) => {
											if(tab.tempId == row.tempId) {
												let nextTab = nameTabs[index+1] || nameTabs[index-1];
												if(nextTab) {
													activeName = nextTab.tempId;
												}
											}
										});
									}

									vm.templateTabsValue = activeName;
									vm.templateNameTabs = nameTabs.filter(tab => tab.tempId != row.tempId);
									vm.tplTabForms = vm.tplTabForms.filter(item => item.tempId != row.tempId);
									vm.$refs.groupTemplateTree.setCurrentKey(activeName); //默认选中的节点
									try{
										vm.resizeDatagrid()
										//$('table.datagrid-f').datagrid('resize');
										window.dispatchEvent(new Event('resize'));
									}	catch(e){}
									vm.queryTemplateList();
								}else{
									vm.$message.error(data["message"])
								}
							}
						}).catch(function(error){})
			        }
			    }).addClass('seriousConfirm');
			},
			//导出
			exportIconClick(row){
				var vm = this, curExportTemplateUrl = '', curCreator = '';

				//原逻辑 creator 未入库，增加此条件验证 creator 是否为空，为空则取 updator
				if(row.creator == '' || row.creator == null || row.creator == undefined){
					curCreator = row.updator;
				}else{
					curCreator = row.creator;
				}
				if(vm.currentNetworkType == 'enb'){
					curExportTemplateUrl = '${ctx}/pm/template/exportTemplateIndicators.action';
				}else if(vm.currentNetworkType == 'gnb'){
					curExportTemplateUrl = '${ctx}/gnb/pm/template/exportTemplateIndicators.action';
				}else if(vm.currentNetworkType == 'egw'){
					curExportTemplateUrl = '${ctx}/egw/pm/template/exportTemplateIndicators.action';
				}
				exportByForm(curExportTemplateUrl,{
			    	tempId: row.tempId,
					creator: curCreator
			    });
			},
			//定时报表
			reportIconClick(row){
				var vm = this;
				vm.rowDataTemplate = row;

				//初始化定时报表数据
				vm.initReportForm(row);
				vm.reportTemplateShow = true;
				try{
					vm.resizeDatagrid()
              		//$('table.datagrid-f').datagrid('resize');
              	  	window.dispatchEvent(new Event('resize'));
          	  	}	catch(e){}
			},
			initReportForm(row){
				var vm = this, curReportInfoUrl = '',
					params ={
						tempId: row.tempId,
						timeZone: timeZone
					};

				if(vm.currentNetworkType == 'enb'){
					curReportInfoUrl = '${ctx}/pm/template/getRegularReportInfo.action';
				}else if(vm.currentNetworkType == 'gnb'){
					curReportInfoUrl = '${ctx}/gnb/pm/template/getRegularReportInfo.action';
				}else if(vm.currentNetworkType == 'egw'){
					curReportInfoUrl = '${ctx}/egw/pm/template/getRegularReportInfo.action';
				}

				axios.post(curReportInfoUrl, stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						vm.reportForm.reportStatus = data.reportStatus;
						vm.reportForm.reportTime = data.reportTime || 0;
						vm.reportForm.mailAddress = data.mailAddress || '';
						vm.reportForm.mailStatus = data.mailStatus || '0';
						if(data.reportPeriod == '' || data.reportPeriod == null || data.reportPeriod ==undefined){
							vm.reportForm.reportPeriod = [];
						}else{
							vm.reportForm.reportPeriod = data.reportPeriod.split(',');
						}
						//增加ftp server 支持 2G,4G,5G
						if(vm.currentNetworkType != "egw"){
							vm.reportForm.ftpSwitch = data.ftpSwitch || '0';
							vm.reportForm.ftpUser = data.ftpUser || '';
							vm.reportForm.ftpIp = data.ftpIp || '';
							vm.reportForm.ftpPort = data.ftpPort || '';
							vm.reportForm.ftpPath = data.ftpPath || '';
							vm.reportForm.ftpPassword = data.ftpPassword || '';
							vm.reportForm.ftpProtocol = data.ftpProtocol || 'sftp';
						}
					}
				}).catch(function(error){})
			},
			//定时报表：邮箱开关 change
			mailStatusChange(value){
				var vm = this;

				if(vm.reportForm.mailStatus){
					//判断当前是否填写 setting 中的email 判断是否可以新建
					axios.post("${ctx}/cell/fault/queryHasSettingEmailConfig.action").then(function(response){
						if(response.data.result == '2'){
							vm.$alert('<%=rb.getString("SheZhiYouXiangTipsKPI")%>','<%=rb.getString("TiShi")%>',{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
							})
							return false;
						}

						if(response.data.result == '3'){
							vm.$alert('<%=rb.getString("MeiYouSheZhiYouXiangKPI")%>','<%=rb.getString("TiShi")%>',{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
							})
							return false;
						}
					})
				}else{
					vm.reportForm.mailAddress = '';
				}
			},
			//定时报表 保存
			reportTemplateSubmit(){
				var vm = this, curSaveReportInfoUrl = '',
					params = {
						tempId: vm.rowDataTemplate.tempId,
						reportStatus: vm.reportForm.reportStatus,
						reportTime: vm.reportForm.reportTime,
						mailAddress: vm.reportForm.mailAddress,
						mailStatus: vm.reportForm.mailStatus,
						timeZone: timeZone
					};
				if(vm.reportForm.reportPeriod.length > 0){
					params.reportPeriod = vm.reportForm.reportPeriod.join(',');
				}else{
					params.reportPeriod = '';
				}

				//原逻辑 creator 未入库，增加此条件验证 creator 是否为空，为空则取 updator
				if(vm.rowDataTemplate.creator == '' || vm.rowDataTemplate.creator == null || vm.rowDataTemplate.creator == undefined){
					params.creator = vm.rowDataTemplate.updator;
				}else{
					params.creator = vm.rowDataTemplate.creator;
				}
				//增加ftp server
				if(vm.currentNetworkType != "egw"){
					params.ftpSwitch = vm.reportForm.ftpSwitch;
					params.ftpUser = vm.reportForm.ftpUser;
					params.ftpIp = vm.reportForm.ftpIp;
					params.ftpPort = vm.reportForm.ftpPort;
					params.ftpPath = vm.reportForm.ftpPath;
					params.ftpPassword = vm.reportForm.ftpPassword;
					params.ftpProtocol = vm.reportForm.ftpProtocol;
				}

				if(vm.currentNetworkType == 'enb'){
					curSaveReportInfoUrl = '${ctx}/pm/template/updateRegularReport.action';
				}else if(vm.currentNetworkType == 'gnb'){
					curSaveReportInfoUrl = '${ctx}/gnb/pm/template/updateRegularReport.action';
				}else if(vm.currentNetworkType == 'egw'){
					curSaveReportInfoUrl = '${ctx}/egw/pm/template/updateRegularReport.action';
				}
				vm.$refs.reportForm.validate((valid) => {
	                if (valid){
	                	axios.post(curSaveReportInfoUrl, stringify(params)).then(function(response){
							var data = response.data;
							if(data) {
								if(data["success"]){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type:'success'
									});
									vm.queryTemplateList();
									vm.reportTemplateShow = false;
									try{
										vm.resizeDatagrid()
					              		//$('table.datagrid-f').datagrid('resize');
					              	  	window.dispatchEvent(new Event('resize'));
					          	  	}	catch(e){}
								}else{
									vm.$message.error(data["message"])
								}
							}
						}).catch(function(error){})
	               	}
	           	})
			},
			//定时报表取消
			reportTemplateCancel(){
				var vm = this;

				vm.reportTemplateShow = false;
				try{
					vm.resizeDatagrid()
              		//$('table.datagrid-f').datagrid('resize');
              	  	window.dispatchEvent(new Event('resize'));
          	  	}	catch(e){}
			},
			//点击页面其他地方菜单收起
			handerClose(row){
				this.$refs.menusTemplate.hide();
			},
			//自定义图形图表，图标点击进入指定页面
			openCustomChartsWin(ele, tmpId){
				var vm = this;

				vm.reportTemplateShow = false; //定时报表关闭

				if($("#winQueryResult_" + vm.templateTabsValue).is(":visible")){
					var chartBt = $('#'+ 'customCharts_bt_' + tmpId);
					chartBt.next().html('<%=rb.getString("LieBiao")%>');
					chartBt.removeClass("el-chart_template").addClass("el-icon-table");
					chartBt.parent().css('right','15px');
					vm.customTableChartTip = '<%=rb.getString("LieBiao")%>';
					$('#chart_circle_bt_' + tmpId).addClass('hidden');
					$('#export_bt_' + tmpId).addClass('hidden');

					$("#winQueryResult_" + vm.templateTabsValue).hide();
					$("#winCustomChartsCondition_"+ vm.templateTabsValue).show(function(){
						$('#winCustomChartsCondition_'+ vm.templateTabsValue).addClass('loading');
						var rd = Math.random().toString().replace('0.','');
						$("#winCustomChartsCondition_" + vm.templateTabsValue).load('${ctx}/pm/chart/template/goKpiChartTemplatePage?randomValue='+rd,function(data){
							$.parser.parse(this);
							$('#winCustomChartsCondition_'+ vm.templateTabsValue).removeClass('loading');
						});
					});
				}else{
					var curForm = vm.tplTabForms.filter((item) => { return item.tempId == vm.templateTabsValue })[0];
					//表格切换到图表，再从图表切换到表格时，表格粒度参数（reportPeriod）为当前被选中的粒度对应值  #50956
					var curPeriod = vm.periodActive, curStartTime = '', curEndTime = '';
					if(vm.currentNetworkType == 'enb'){
						if(curForm.periodActive == '15' || curForm.periodActive == '60'){
							curStartTime = curForm.curTimeFrame + ' 00:00:00';
							curEndTime = dateformatter(addDate(new Date(curStartTime),1));
						}else{
							curStartTime = curForm.dateValue[0] + ' 00:00:00';
							var endTime = curForm.dateValue[1] + ' 00:00:00';
							curEndTime = dateformatter(addDate(new Date(endTime),1));
						}
					}else{
						curStartTime = curForm.dateValue[0] + ' 00:00:00';
						var endTime = curForm.dateValue[1] + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(endTime),1));
					}

					var chartBt = $('#'+ 'customCharts_bt_' + tmpId);
					chartBt.next().html('<%=rb.getString("ZiDingYiTuBiao")%>')
					chartBt.removeClass("el-icon-table").addClass("el-chart_template");
					chartBt.parent().css('right','60px');
					vm.customTableChartTip = '<%=rb.getString("ZiDingYiTuBiao")%>';
					$('#export_bt_' + tmpId).removeClass('hidden');
					$('#chart_circle_bt_' + tmpId).removeClass('hidden');
					
					$("#winCustomChartsCondition_"+ vm.templateTabsValue).hide();
					$("#winQueryResult_" + vm.templateTabsValue).show(function(){
						$("#winCustomChartsCondition_"+ vm.templateTabsValue).html("");

						//selDeviceType: 1-设备组 2-设备
						var tableParams = {};
						if(vm.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
							tableParams = {
								timeZone: timeZone,
								tempId: curForm.tempId,
								searchText: curForm.searchText,
								groupId : curForm.deviceGroupParam,
								startTime: curStartTime,
								endTime: curEndTime,
								period: curForm.periodActive,
								operatorCode: curForm.operatorParam
							}
						}else{
							tableParams = {
								timeZone :timeZone,
								tempId : curForm.tempId,
								searchText : curForm.searchText,
								groupId : curForm.deviceGroupParam,
								serialNumber : '', //无用参数
								hostName : '', //无用参数
								startTime : curStartTime,
								endTime : curEndTime,
								reportPeriod: curForm.periodActive,
								operatorCode: curForm.operatorParam 
							}
						}

						$("#kpiTaskPerfDatagrid_" + tmpId).datagrid({ //在图表页切换查询模板，在切换回表格时需要回显对应模板数据
							queryParams : tableParams
						});
					});
				}
			},
			closeCustomChartsWin(ele, tmpId){
				var vm = this;

				var curForm = vm.tplTabForms.filter((item) => { return item.tempId == tmpId })[0];
				//表格切换到图表，再从图表切换到表格时，表格粒度参数（reportPeriod）为当前被选中的粒度对应值  #50956
				var curPeriod = vm.periodActive, curStartTime = '', curEndTime = '';
				if(vm.currentNetworkType == 'enb'){
					if(curForm.periodActive == '15' || curForm.periodActive == '60'){
						curStartTime = curForm.curTimeFrame + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(curStartTime),1));
					}else{
						curStartTime = curForm.dateValue[0] + ' 00:00:00';
						var endTime = curForm.dateValue[1] + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(endTime),1));
					}
				}else{
					curStartTime = curForm.dateValue[0] + ' 00:00:00';
					var endTime = curForm.dateValue[1] + ' 00:00:00';
					curEndTime = dateformatter(addDate(new Date(endTime),1));
				}

				//普通图表图标恢复
				var chartBt = $('#'+ 'chart_bt_' + tmpId);
				chartBt.next().html('<%=rb.getString("TuBiao")%>');
				chartBt.removeClass("el-icon-table").addClass("el-icon-circle-chart");
				chartBt.parent().css('right', (vm.currentNetworkType == 'egw' ? '60px' : '105px'));
				vm.tableChartTip = '<%=rb.getString("TuBiao")%>';
				//自定义图表图标恢复
				var customChartBt = $('#'+ 'customCharts_bt_' + tmpId);
				customChartBt.next().html('<%=rb.getString("ZiDingYiTuBiao")%>')
				customChartBt.removeClass("el-icon-table").addClass("el-chart_template");
				customChartBt.parent().css('right', '60px');
				vm.customTableChartTip = '<%=rb.getString("ZiDingYiTuBiao")%>';
				$('#chart_circle_bt_' + tmpId).removeClass('hidden');
				$('#customCharts_' + tmpId).removeClass('hidden');
				$('#export_bt_' + tmpId).removeClass('hidden');
				$("#winChartCondition_"+ tmpId).hide();
				$("#winCustomChartsCondition_"+ tmpId).hide();
				
				$("#winQueryResult_" + tmpId).show(function(){
					$("#winCustomChartsCondition_"+ tmpId).html("");

					//selDeviceType: 1-设备组 2-设备
					var tableParams = {};
					if(vm.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
						tableParams = {
							timeZone: timeZone,
							tempId: curForm.tempId,
							searchText: curForm.searchText,
							groupId : curForm.deviceGroupParam,
							startTime: curStartTime,
							endTime: curEndTime,
							period: curForm.periodActive,
							operatorCode: curForm.operatorParam
						}
					}else{
						tableParams = {
							timeZone :timeZone,
							tempId : curForm.tempId,
							searchText : curForm.searchText,
							groupId : curForm.deviceGroupParam,
							serialNumber : '', //无用参数
							hostName : '', //无用参数
							startTime : curStartTime,
							endTime : curEndTime,
							reportPeriod: curForm.periodActive,
							operatorCode: curForm.operatorParam 
						}
					}

					$("#kpiTaskPerfDatagrid_" + tmpId).datagrid({ //在图表页切换查询模板，在切换回表格时需要回显对应模板数据
						queryParams : tableParams
					});
				});
			},
			//常规图表图标点击事件
			openChartWin(ele, tmpId){
				var vm = this;

				vm.reportTemplateShow = false; //定时报表关闭

				if($("#winQueryResult_" + vm.templateTabsValue).is(":visible")){
					var chartBt = $('#'+ 'chart_bt_' + tmpId);
					chartBt.next().html('<%=rb.getString("LieBiao")%>');
					chartBt.removeClass("el-icon-circle-chart").addClass("el-icon-table");
					chartBt.parent().css('right','15px');
					$('#export_bt_' + tmpId).addClass('hidden');
					$('#customCharts_' + tmpId).addClass('hidden');
					vm.tableChartTip = '<%=rb.getString("LieBiao")%>';

					$("#winQueryResult_" + vm.templateTabsValue).hide();
					$("#winChartCondition_"+ vm.templateTabsValue).show(function(){
						$('#winChartCondition_'+ vm.templateTabsValue).addClass('loading');
						
						var rd = Math.random().toString().replace('0.','');
						$("#winChartCondition_" + vm.templateTabsValue).load('${ctx}/pm/template/viewTemplateChart.action?randomValue='+rd,function(data){
							$.parser.parse(this);
							$('#winChartCondition_'+ vm.templateTabsValue).removeClass('loading');
						});
					});
				}else{
					var curForm = vm.tplTabForms.filter((item) => { return item.tempId == vm.templateTabsValue })[0];
					//表格切换到图表，再从图表切换到表格时，表格粒度参数（reportPeriod）为当前被选中的粒度对应值  #50956
					var curPeriod = vm.periodActive, curStartTime = '', curEndTime = '';
					if(vm.currentNetworkType == 'enb'){
						if(curForm.periodActive == '15' || curForm.periodActive == '60'){
							curStartTime = curForm.curTimeFrame + ' 00:00:00';
							curEndTime = dateformatter(addDate(new Date(curStartTime),1));
						}else{
							curStartTime = curForm.dateValue[0] + ' 00:00:00';
							var endTime = curForm.dateValue[1] + ' 00:00:00';
							curEndTime = dateformatter(addDate(new Date(endTime),1));
						}
					}else{
						curStartTime = curForm.dateValue[0] + ' 00:00:00';
						var endTime = curForm.dateValue[1] + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(endTime),1));
					}

					var chartBt = $('#'+ 'chart_bt_' + tmpId);
					chartBt.next().html('<%=rb.getString("TuBiao")%>')
					chartBt.removeClass("el-icon-table").addClass("el-icon-circle-chart");
					chartBt.parent().css('right', (vm.currentNetworkType == 'egw' ? '60px' : '105px'));

					$('#export_bt_' + tmpId).removeClass('hidden');
					$('#customCharts_' + tmpId).removeClass('hidden');
					vm.tableChartTip = '<%=rb.getString("TuBiao")%>';

					$("#winChartCondition_"+ vm.templateTabsValue).hide();
					$("#winQueryResult_" + vm.templateTabsValue).show(function(){
						$("#winChartCondition_"+ vm.templateTabsValue).html("");

						//selDeviceType: 1-设备组 2-设备
						var tableParams = {};
						if(vm.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
							tableParams = {
								timeZone: timeZone,
								tempId: curForm.tempId,
								searchText: curForm.searchText,
								groupId : curForm.deviceGroupParam,
								startTime: curStartTime,
								endTime: curEndTime,
								period: curForm.periodActive,
								operatorCode: curForm.operatorParam
							}
						}else{
							tableParams = {
								timeZone :timeZone,
								tempId : curForm.tempId,
								searchText : curForm.searchText,
								groupId : curForm.deviceGroupParam,
								serialNumber : '', //无用参数
								hostName : '', //无用参数
								startTime : curStartTime,
								endTime : curEndTime,
								reportPeriod: curForm.periodActive,
								operatorCode: curForm.operatorParam 
							}
						}

						$("#kpiTaskPerfDatagrid_" + tmpId).datagrid({ //在图表页切换查询模板，在切换回表格时需要回显对应模板数据
							queryParams : tableParams
						});
					});
				}
			},
			exportKpiData(tmpId){
				var curReportCycle = '';

				kpiQueryVue.reportTemplateShow = false; //定时报表关闭
				closeKpiDrillDiv();

				if(reportCycle == '1440'){
					curReportCycle = '24<%=rb.getString("XiaoShi")%>';
				}else if(reportCycle == '10080'){
					curReportCycle ='<%=rb.getString("Zhou")%>';
				}else if(reportCycle == '43200'){
					curReportCycle = '<%=rb.getString("Yue")%>';
				}else{
					curReportCycle = reportCycle + 'Min';
				}
				//跳转导出，显示头部的模板名称及当前选中粒度
				var paramArr = [kpiQueryVue.selectTemplateName, curReportCycle]
				sessionStorage.setItem('exportEnbGnbOrEgw', paramArr);
				$('#exportKpiDiv').addClass('loading');
				$("#exportKpiDiv").slideDown(500,function(){
					$("#exportKpiDiv").load('${ctx}/pm/template/export/goExportPage.action?randomValue='+strRandom,function(data){
						$.parser.parse(this);
						$('#exportKpiDiv').removeClass('loading');
					});
				});
			},
			vagueQueryKPITaskPerfDatagrid(index){
				var vm = this,
					curStartTime = '', 
					curEndTime = '',
					queryForm = vm.tplTabForms[index];

				if(vm.currentNetworkType == 'enb'){
					if(queryForm.periodActive == '15' || queryForm.periodActive == '60'){
						curStartTime = queryForm.curTimeFrame + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(curStartTime),1));
					}else{
						curStartTime = queryForm.dateValue[0] + ' 00:00:00';
						var endTime = queryForm.dateValue[1] + ' 00:00:00';
						curEndTime = dateformatter(addDate(new Date(endTime),1));
					}
					//selDeviceType: 1-设备组 2-设备
					if(queryForm.selDeviceType == '2'){
						//查询数据大于 200条时，接口将给出提示，反之 rows 正常返回表格数据；#62165
						var params = {
							timeZone : timeZone,
							tempId : queryForm.tempId,
							searchText : queryForm.searchText,
							groupId : queryForm.deviceGroupParam,
							serialNumber : '', //无用参数
							hostName : '', //无用参数
							startTime : curStartTime,
							endTime : curEndTime,
							reportPeriod: queryForm.periodActive,
							operatorCode: queryForm.operatorParam
						}
						axios.post('${ctx}/pm/template/getTemplateDataPage.action', stringify(params)).then(function(response){
							var data = response.data;
							if(data) {
								if(data['success'] == false) {
									showMsg('error_msg', '<%=rb.getString("XiangXiChaXunTiaoJian")%>');
								}else {
									$("#kpiTaskPerfDatagrid_"+ vm.templateTabsValue).datagrid({
										queryParams : params
									});
								}
							}
						}).catch(function(error){})		
					}else{
						$("#kpiTaskPerfDatagrid_"+ vm.templateTabsValue).datagrid({
							queryParams : {
								timeZone: timeZone,
								tempId: queryForm.tempId,
								searchText: queryForm.searchText,
								groupId : queryForm.deviceGroupParam,
								startTime: curStartTime,
								endTime: curEndTime,
								period: queryForm.periodActive,
								operatorCode: queryForm.operatorParam
							}
						});
					}
				}else{
					curStartTime = queryForm.dateValue[0] + ' 00:00:00';
					var endTime = queryForm.dateValue[1] + ' 00:00:00';
					curEndTime = dateformatter(addDate(new Date(endTime),1));

					$("#kpiTaskPerfDatagrid_"+ vm.templateTabsValue).datagrid({
						queryParams : {
							timeZone : timeZone,
							tempId : queryForm.tempId,
							searchText : queryForm.searchText,
							groupId : queryForm.deviceGroupParam,
							serialNumber : '',
							hostName : '',
							startTime : curStartTime,
							endTime : curEndTime,
							reportPeriod: queryForm.periodActive,
							operatorCode: queryForm.operatorParam
						}
					});
				}
			}
		 },
		mounted(){
			// 记录初始网元类型
			this.lastNetworkType = this.currentNetworkType;
			
			// 只有在支持的网元下才初始化
			if(this.currentNetworkType === 'enb' || this.currentNetworkType === 'gnb' || this.currentNetworkType === 'egw') {
				this.init();
			}
		}
	})

	$(function () {
	   	// 权限控制
	   	setTimeout(function(){
	   		$('#kpiQueryPanel span[tabtit]:visible').each(function(n,item){
	   			if(n==0) $(item).click();
	   		});
	   	},0);
	});

	function datagridTableLoadSuccess(data) {
    	$(this).datagrid("fixRownumber");
    	$(this).datagrid("enableContextmenuAutoSize");
		var curWinQueryResule = $('#winQueryResult_' + kpiQueryVue.templateTabsValue);
		var curPaginationNum = curWinQueryResule.find('.pagination-num');
		curPaginationNum.attr('disabled',true);
    }

	// 右侧表格数据渲染
	function immediateQuery(){
		var curTemplateRelUrl = '', curTemplateInfoUrl = '', curTableUrl = '', curStartTime = '', curEndTime = '', columnsList = [], frozenColumnsList = [], curCreator = '';

		var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
		if(kpiQueryVue.currentNetworkType == 'enb'){
			//0-4G
			curTemplateRelUrl = '${ctx}/pm/template/getTemplateRelIndicators.action';
			curTemplateInfoUrl = '${ctx}/pm/template/getTemplateInfo.action';
			//curTableUrl = curForm.selDeviceType == '1' ? '${ctx}/pm/template/queryPmDataByPageForGroup.action' : '${ctx}/pm/template/getTemplateDataPage.action';

			/*var periodActive = curForm? curForm.periodActive : kpiQueryVue.periodActive;

			if(periodActive == '15' || periodActive == '60'){
				curStartTime = kpiQueryVue.curTimeFrame + ' 00:00:00';
				curEndTime = dateformatter(addDate(new Date(curStartTime),1));
			}else{
				curStartTime = curForm? curForm.dateValue[0] + ' 00:00:00' : kpiQueryVue.dateValue[0] + ' 00:00:00';
				var endTime = curForm? curForm.dateValue[1] + ' 00:00:00' : kpiQueryVue.dateValue[1] + ' 00:00:00';
				curEndTime = dateformatter(addDate(new Date(endTime),1));
			}*/
				
 	    }else if(kpiQueryVue.currentNetworkType == 'gnb'){
 	    	//1-5G
 		    curTemplateRelUrl = '${ctx}/gnb/pm/template/getTemplateRelIndicators.action';
			curTemplateInfoUrl = '${ctx}/gnb/pm/template/getTemplateInfo.action';
			curTableUrl = '${ctx}/gnb/pm/template/getTemplateDataPage.action';

 	    	curStartTime = curForm? curForm.dateValue[0] + ' 00:00:00' : kpiQueryVue.dateValue[0] + ' 00:00:00';
			var endTime = curForm? curForm.dateValue[1] + ' 00:00:00' : kpiQueryVue.dateValue[1] + ' 00:00:00';
			curEndTime = dateformatter(addDate(new Date(endTime),1));
 	    }else if(kpiQueryVue.currentNetworkType == 'egw'){
 		     //2-eGW
 	    	curTemplateRelUrl = '${ctx}/egw/pm/template/getTemplateRelIndicators.action';
			curTemplateInfoUrl = '${ctx}/egw/pm/template/getTemplateInfo.action';
 	    	curTableUrl = '${ctx}/egw/pm/template/getTemplateDataPage.action';

 	    	curStartTime = curForm? curForm.dateValue[0] + ' 00:00:00' : kpiQueryVue.dateValue[0] + ' 00:00:00';
			var endTime = curForm? curForm.dateValue[1] + ' 00:00:00' : kpiQueryVue.dateValue[1] + ' 00:00:00';
			curEndTime = dateformatter(addDate(new Date(endTime),1));
 	    }
		//原逻辑 creator 未入库，增加此条件验证 creator 是否为空，为空则取 updator
		if(kpiQueryVue.commonTemplateSelectRow.creator == '' || kpiQueryVue.commonTemplateSelectRow.creator == null || kpiQueryVue.commonTemplateSelectRow.creator == undefined){
			curCreator = kpiQueryVue.commonTemplateSelectRow.updator;
		}else{
			curCreator = kpiQueryVue.commonTemplateSelectRow.creator;
		}
		 //curTemplateRelUrl 一直loading  因为模板参数没值 接口没有返回内容；
		// 表格结束时间字段-后面的显示字段
		var formTplId = curForm? curForm.tempId : '';
		if(formTplId == '' || formTplId == null || formTplId == undefined){

			if(kpiQueryVue.currentNetworkType == 'enb'){

        		if(reportCycle == '1440'){
					periodUnit = '(<%=rb.getString("XiaoShi")%>)';
				}else if(reportCycle == '10080'){
					periodUnit = '(<%=rb.getString("Zhou")%>)';
				}else if(reportCycle == '43200'){
					periodUnit = '(<%=rb.getString("Yue")%>)';
				}else{
					periodUnit = '(<%=rb.getString("FenZhongDaXie")%>)';
				}
        		columnsList = [
                    {field: 'id',hidden: true},//唯一标识
                    {field: 'smallCellCode',hidden: true},
                    // {field: 'serialNumber',title: '<%=rb.getString("XiaoZhanBianMa")%>', width: 200},
					// {field: 'hostName', title: '<%=rb.getString("HostName")%>', width: 200},
					{field: 'enodeId', title: '<%=rb.getString("EnodebId")%>', width: 190},
					{field: 'cellId', title: '<%=rb.getString("XIAOQUID")%>',width : 190},
					{field: 'eci', title: 'ECI', width: 200},
					{field: 'groupName', title: '<%=rb.getString("SheBeiZuMingCheng")%>', width: 230},
					{field: 'timeLevel', title: '<%=rb.getString("ChaXunLiDu")%>'+ periodUnit , width: 170,formatter: periodUnitStyle},
					{field: 'startTime', title: '<%=rb.getString("KaiShiShiJian")%>', width: 190},
					{field: 'endTime', title: '<%=rb.getString("JieShuShiJian")%>',width : 190}
               ];
               frozenColumnsList = [
                    {field: 'serialNumber',title: '<%=rb.getString("XiaoZhanBianMa")%>', width: 200},
					{field: 'hostName', title: '<%=rb.getString("HostName")%>', width: 200},
               ];
     	    }
			
			closeLoading();
		}else{
			$.post(curTemplateRelUrl,{tempId : formTplId}, function (data) {
		   		closeLoading();
		        if (data) {
					var levelType = '';
		        	$.ajax({
		        		type: "post",
		        		url: curTemplateInfoUrl,
		        		data: {
		        			tempId : formTplId,
		        			timeZone : timeZone,
							creator: curCreator
		        		},
		        		async: false,
		        		dataType:"json",
		        		success: function(data) {
							//模板关联的层级类型, 控制 plmn id 的显示和隐藏
							levelType = data.indicatorLevel;
							kpiQueryVue.levelType = levelType;

							reportCycle = newPeriod ? newPeriod : data.reportPeriod;
		        			kpiQueryVue.periodActive = reportCycle;
							curForm.periodActive = reportCycle;
							curForm.searchText = '';

							//enb 增加设备组 和 设备的切换
							//判断是否第一次调用getTemplateInfo.action接口
							if(kpiQueryVue.currentNetworkType == 'enb' && kpiQueryVue.firstFlag){

								if(data.isAllOperator == '1'){
									curForm.selDeviceType = '2';
								}else{
									curForm.selDeviceType = data.selDeviceType;
								}

								//curForm.selDeviceType = data.selDeviceType;
								//1-设备组 2-设备
								if(curForm.selDeviceType == '2'){
									if(levelType == 'device'){
										kpiQueryVue.enbPlaceholder = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>';						
									}else{
										kpiQueryVue.enbPlaceholder = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%> / <%=rb.getString("GNBPLMNBiaoShi")%>';
									}
								}else{
									kpiQueryVue.enbPlaceholder = '<%=rb.getString("SheBeiZuMingCheng")%>'
								}
								
								kpiQueryVue.firstFlag = false;
							}

							if(kpiQueryVue.currentNetworkType == 'enb'){
								curTableUrl = curForm.selDeviceType == '1' ? '${ctx}/pm/template/queryPmDataByPageForGroup.action' : '${ctx}/pm/template/getTemplateDataPage.action';
							}
		        		}
		        	});
					
		        	if(kpiQueryVue.currentNetworkType == 'enb'){
		    			//4G
		    			//调整时间轴
		        		var start = curForm.dateValue[0], end = curForm.dateValue[1], num = differ(end,start);

			        	kpiQueryVue.timeFrame = [];
			        	curForm.timeFrame = [];
						for(var i = 0; i<=num; i++) {
							var dateStr = dateformatter(addDate(start,i));
							kpiQueryVue.timeFrame.push(dateStr.substr(0,10));
							curForm.timeFrame.push(dateStr.substr(0,10));
						}
			        	if(curForm.periodActive == '15' || curForm.periodActive == '60'){
							//当前24小时，非15 60min，如果时间范围超出7天，再次切换到15 60min 时，根据最后选中的时间范围进行取值7天的时间轴并给出提示；
							if(curForm.timeFrame.length > 7){
								kpiQueryVue.timeFrame = kpiQueryVue.timeFrame.slice(-7); //时间轴
								kpiQueryVue.dateValue = [kpiQueryVue.timeFrame[0],kpiQueryVue.timeFrame[kpiQueryVue.timeFrame.length-1]]; // 时间范围选择框需重新赋值；
								
								curForm.timeFrame = curForm.timeFrame.slice(-7); //时间轴
								curForm.dateValue = [curForm.timeFrame[0],curForm.timeFrame[curForm.timeFrame.length-1]]; // 时间范围选择框需重新赋值；
								kpiQueryVue.curTimeFrame = kpiQueryVue.timeFrame[kpiQueryVue.timeFrame.length-1]; // 当前选中最后一个日期
								curForm.curTimeFrame = curForm.timeFrame[curForm.timeFrame.length-1]; // 当前选中最后一个日期

								showMsg('prompt_msg','<%=rb.getString("KPIShiJianFanWeiTiShi")%>');
							}else {
								//当前时间轴只有一天时，则为重新赋值，反之，按当前选中日期查询
								if(curForm.timeFrame.length == 1){
									kpiQueryVue.curTimeFrame = kpiQueryVue.timeFrame[0];
									curForm.curTimeFrame = curForm.timeFrame[0];
								}else{
									kpiQueryVue.curTimeFrame = kpiQueryVue.timeFrame[kpiQueryVue.timeFrame.length-1]; // 当前选中最后一个日期
									curForm.curTimeFrame = curForm.timeFrame[curForm.timeFrame.length-1]; // 当前选中最后一个日期
								}
							}
							//curStartTime开始时间为时间轴当前选中的日期
							curStartTime = curForm.curTimeFrame + ' 00:00:00';
							//curEndTime 结束时间为开始时间加1天
							curEndTime = dateformatter(addDate(new Date(curStartTime),1));
						}else{
							curStartTime = curForm? curForm.dateValue[0] + ' 00:00:00' : kpiQueryVue.dateValue[0] + ' 00:00:00';
							var endTime = curForm? curForm.dateValue[1] + ' 00:00:00' : kpiQueryVue.dateValue[1] + ' 00:00:00';
							curEndTime = dateformatter(addDate(new Date(endTime),1));
						}

		        		if(reportCycle == '1440'){
							periodUnit = '(<%=rb.getString("XiaoShi")%>)';
						}else if(reportCycle == '10080'){
							periodUnit = '(<%=rb.getString("Zhou")%>)';
						}else if(reportCycle == '43200'){
							periodUnit = '(<%=rb.getString("Yue")%>)';
						}else{
							periodUnit = '(<%=rb.getString("FenZhongDaXie")%>)';
						}
		        		
					   
						if('${modelType}' == 'S0009'){
                            if(curForm.selDeviceType == '2'){
                                columnsList = [
                                    {field: 'id',hidden: true},//唯一标识
                                    {field: 'smallCellCode',hidden: true},
									{field: 'plmnId', title: '<%=rb.getString("GNBPLMNBiaoShi")%>', width: 200, hidden: levelType == 'plmn' ? false : true}, 
                                    {field: 'subStationName', title: '<%=rb.getString("ZhanZhiMingCheng")%>', width: 200},
                                    {field: 'enodeId', title: '<%=rb.getString("EnodebId")%>', width: 190},
                                    {field: 'cellId', title: '<%=rb.getString("XIAOQUID")%>',width : 190},
                                    {field: 'eci', title: 'ECI', width: 200},
                                    {field: 'groupName', title: '<%=rb.getString("SheBeiZuMingCheng")%>', width: 230},
                                    {field: 'timeLevel', title: '<%=rb.getString("ChaXunLiDu")%>'+ periodUnit , width: 170,formatter: periodUnitStyle},
                                    {field: 'startTime', title: '<%=rb.getString("KaiShiShiJian")%>', width: 190},
                                    {field: 'endTime', title: '<%=rb.getString("JieShuShiJian")%>',width : 190}
                                ];
                                frozenColumnsList = [
                                    {field: 'serialNumber',title: '<%=rb.getString("XiaoZhanBianMa")%>', width: 200},
                                    {field: 'hostName', title: '<%=rb.getString("HostName")%>', width: 200},
                                ];
                            }else{
                                //设备组
                                columnsList = [
                                    {field: 'id',hidden: true},
                                    {field: 'group_name', title: '<%=rb.getString("SheBeiZuMingCheng")%>', width: 460},
                                    {field: 'timeLevel', title: '<%=rb.getString("ChaXunLiDu")%>'+ periodUnit , width: 300,formatter: periodUnitStyle},
                                    {field: 'startTime', title: '<%=rb.getString("KaiShiShiJian")%>', width: 300},
                                    {field: 'endTime', title: '<%=rb.getString("JieShuShiJian")%>',width : 300},
                                ];
                                frozenColumnsList = [];
                            }

						}else{
							//selDeviceType: 1-设备组 2-设备
							if(curForm.selDeviceType == '2'){
								columnsList = [
									{field: 'id',hidden: true},
									{field: 'smallCellCode',hidden: true},
									{field: 'plmnId', title: '<%=rb.getString("GNBPLMNBiaoShi")%>', width: 200, hidden: levelType == 'plmn' ? false : true}, 
									{field: 'enodeId', title: '<%=rb.getString("EnodebId")%>', width: 190},
									{field: 'cellId', title: '<%=rb.getString("XIAOQUID")%>',width : 190},
									{field: 'eci', title: 'ECI', width: 200},
									{field: 'groupName', title: '<%=rb.getString("SheBeiZuMingCheng")%>', width: 230},
									{field: 'timeLevel', title: '<%=rb.getString("ChaXunLiDu")%>'+ periodUnit , width: 170,formatter: periodUnitStyle},
									{field: 'startTime', title: '<%=rb.getString("KaiShiShiJian")%>', width: 190},
									{field: 'endTime', title: '<%=rb.getString("JieShuShiJian")%>',width : 190}
								];
                                frozenColumnsList = [
                                    {field: 'serialNumber',title: '<%=rb.getString("XiaoZhanBianMa")%>', width: 200},
                                    {field: 'hostName', title: '<%=rb.getString("HostName")%>', width: 200},
                                ];
							}else{
								//设备组
								columnsList = [
									{field: 'id',hidden: true},
									{field: 'group_name', title: '<%=rb.getString("SheBeiZuMingCheng")%>', width: 460},
									{field: 'timeLevel', title: '<%=rb.getString("ChaXunLiDu")%>'+ periodUnit , width: 300,formatter: periodUnitStyle},
									{field: 'startTime', title: '<%=rb.getString("KaiShiShiJian")%>', width: 300},
									{field: 'endTime', title: '<%=rb.getString("JieShuShiJian")%>',width : 300},
								];
                                frozenColumnsList = [];
							}
						}
		     	    }else if(kpiQueryVue.currentNetworkType == 'gnb'){
		     	    	//5G 
		     	    	periodUnit = reportCycle == '1440'? '(<%=rb.getString("XiaoShi")%>)':'(<%=rb.getString("FenZhongDaXie")%>)';
						
						columnsList = [
							{field: 'id',hidden: true},
							{field: 'smallCellCode',hidden: true},
							// {field: 'serialNumber',title: '<%=rb.getString("XiaoZhanBianMa")%>', width: 200},
							// {field: 'hostName', title: '<%=rb.getString("GNBMingCheng")%>', width: 200},
							{field: 'plmnId', title: '<%=rb.getString("GNBPLMNBiaoShi")%>', width: 200},
							{field: 'nrCGI', title: 'NrCGI', width: 200},
							{field: 'groupName', title: '<%=rb.getString("SheBeiZuMingCheng")%>', width: 230},
							{field: 'timeLevel', title: '<%=rb.getString("ChaXunLiDu")%>'+ periodUnit , width: 170,formatter: periodUnitStyle},
							{field: 'startTime', title: '<%=rb.getString("KaiShiShiJian")%>', width: 190},
							{field: 'endTime', title: '<%=rb.getString("JieShuShiJian")%>',width : 190},
						];
                        frozenColumnsList = [
                            {field: 'serialNumber',title: '<%=rb.getString("XiaoZhanBianMa")%>', width: 200},
							{field: 'hostName', title: '<%=rb.getString("GNBMingCheng")%>', width: 200},
                        ];
		     	    }else if(kpiQueryVue.currentNetworkType == 'egw'){
		     		     //eGW
		     	    	periodUnit = reportCycle == '1440'? '(<%=rb.getString("XiaoShi")%>)':'(<%=rb.getString("FenZhongDaXie")%>)';
						columnsList = [
	                        {field: 'id',hidden: true},
	                        {field: 'smallCellCode',hidden: true},
	                        // {field: 'serialNumber',title: '<%=rb.getString("eGWBianMa")%>', width: 200},
							// {field: 'hostName', title: '<%=rb.getString("eGWMingCheng")%>', width: 200},
							{field: 'groupName', title: '<%=rb.getString("SheBeiZuMingCheng")%>', width: 230},
							{field: 'timeLevel', title: '<%=rb.getString("ChaXunLiDu")%>'+ periodUnit , width: 170,formatter: periodUnitStyle},
							{field: 'startTime', title: '<%=rb.getString("KaiShiShiJian")%>', width: 190},
							{field: 'endTime', title: '<%=rb.getString("JieShuShiJian")%>',width : 190},
	                   ];
                       frozenColumnsList = [
                            {field: 'serialNumber',title: '<%=rb.getString("eGWBianMa")%>', width: 200},
							{field: 'hostName', title: '<%=rb.getString("eGWMingCheng")%>', width: 200},
                        ];
		     	    }

		            $.each(data,function(index,obj){
			       	    var field = obj.kpiId;
						var title = obj.kpiName + "(" + obj.unit + ")";
						var platformTypeList = obj.platformSupported ? obj.platformSupported.split(',') : [];
						
						//enb 时,选中设备组时，不钻取  1-设备组 2-设备
						if(kpiQueryVue.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
							var columnObj={field: field, title: title, width:260};
						}else{
							if(obj.isCounter != "1" || obj.isBuildIn != "1"){
								var columnObj={field: field, title: title, width:260, formatter: kpiDrillStyle,platform:platformTypeList};
							}else{
								var columnObj={field: field, title: title, width:260};
							}
						}
			           	columnsList.push(columnObj);
					});

					//selDeviceType: 1-设备组 2-设备
					var tableParams = {};
					if(kpiQueryVue.currentNetworkType == 'enb' && curForm.selDeviceType == '1'){
						tableParams = {
							timeZone: timeZone,
							tempId: formTplId,
							searchText: curForm.searchText,
							groupId : isAllOperatorTemp?'':device_group_id_query,
							startTime: curStartTime,
							endTime: curEndTime,
							period: reportCycle,
							operatorCode: isAllOperatorTemp?perfOperatorCode:'' 
						}
					}else{
						tableParams = {
							timeZone: timeZone,
							tempId: formTplId,
							searchText: curForm.searchText,
							groupId : isAllOperatorTemp?'':device_group_id_query,
							serialNumber: '',
							hostName: '',
							startTime: curStartTime,
							endTime: curEndTime,
							reportPeriod: reportCycle,
							operatorCode: isAllOperatorTemp?perfOperatorCode:'' 
						}
					}

					$("#kpiTaskPerfDatagrid_" + kpiQueryVue.templateTabsValue).datagrid({
						url : curTableUrl,
						border:false,
						fit:true,
						toolbar:'#toolbar_kpiTaskPerfDatagrid_'+ kpiQueryVue.templateTabsValue,
						rownumbers:true,
						fitColumns:false,
						pagination:true,
						pagePosition:'bottom',
						striped:true,
						singleSelect:true,
						idField:'id',
						onLoadSuccess:datagridTableLoadSuccess,
						onBeforeLoad:function(param){
							$("#kpiTaskPerfDatagrid_"+ kpiQueryVue.templateTabsValue).datagrid("getPager").pagination({
								layout:['list','prev','manual','next','refresh',]
							});
						},
						queryParams : tableParams,
                        frozenColumns : [frozenColumnsList],
						columns : [columnsList],
						pageNumber : 1
					})
						
					// 强化的 resize 逻辑，修复表头和 body 错位问题 #107735
					setTimeout(function(){
						var $dg = $("#kpiTaskPerfDatagrid_" + kpiQueryVue.templateTabsValue);
						
						try {
							// 第一次 resize
							$dg.datagrid("resize");
							
							// 延迟再次 resize，确保 DOM 完全渲染
							setTimeout(function() {
								try {
									$dg.datagrid("resize");
									
									// 修复冻结列和普通列的表头高度不一致问题
									var panel = $dg.datagrid('getPanel');
									var view = panel.find('div.datagrid-view');
									var view1 = view.find('div.datagrid-view1'); // 冻结列
									var view2 = view.find('div.datagrid-view2'); // 普通列
									
									if(view1.length > 0 && view2.length > 0) {
										var header1 = view1.find('div.datagrid-header');
										var header2 = view2.find('div.datagrid-header');
										
										// 同步表头高度
										var maxHeaderHeight = Math.max(header1.height(), header2.height());
										if(maxHeaderHeight > 0) {
											header1.height(maxHeaderHeight);
											header2.height(maxHeaderHeight);
										}
									}
									
									// 最后再 resize 一次
									$dg.datagrid("resize");
									
									// 重要：在 resize 之后再修复表头宽度（避免被 resize 覆盖）
									setTimeout(function() {
										try {
											// 修复横向滚动时表头和 body 错位问题（关键修复）
											var view2 = panel.find('div.datagrid-view2');
											if(view2.length > 0) {
												var header2 = view2.find('div.datagrid-header');
												var body2 = view2.find('div.datagrid-body');
												
												// 同步表头容器宽度与 body 容器宽度
												if(body2.length > 0 && header2.length > 0) {
													// 获取 body 容器的宽度
													var bodyWidth = body2.width();
													
													// 关键修复：设置 header 容器宽度 = body 容器宽度（保持一致）
													header2.width(bodyWidth);
												}
											}
										} catch(e) {
										}
									}, 50);
									
									// 监听横向滚动事件，动态修复错位
									var body2 = panel.find('div.datagrid-view2 div.datagrid-body');
									if(body2.length > 0) {
										// 移除旧的监听器，避免重复绑定
										body2.off('scroll.headerfix');
										
										// 绑定滚动事件
										body2.on('scroll.headerfix', function() {
											try {
												var scrollLeft = $(this).scrollLeft();
												var header2 = panel.find('div.datagrid-view2 div.datagrid-header');
												
												// 同步表头的滚动位置
												header2.scrollLeft(scrollLeft);
												
												// 同步表头容器宽度（与 body 容器保持一致）
												var bodyWidth = $(this).width();  // 获取 body 容器的宽度
												header2.width(bodyWidth);
												
											} catch(e) {
											}
										});
									}
								} catch(e) {
								}
							}, 100);
						} catch(e) {
						}
					}, 0);

					closeLoading();
				}
			}, "json");
		}
	}

	/**
	* 测量周期显示转换 - 格式化
	* @param value[string] 查询粒度字段值
	* @param rowData[object] 表格行数据
	* @param rowIndex{number}: 选中行的索引
	**/
	function periodUnitStyle(value, rowData, rowIndex){
		if(kpiQueryVue.currentNetworkType == 'enb'){
			if(value == '1440'){
				return '24';
			}else if(value == '10080'){
				return '1';
			}else if(value == '43200'){
				return '1';
			}
		}else{
			if(value == '1440'){
				return '24';
			}
		}
		return value;
	}

	/**
	* 可钻取指标样式呈现 - 格式化
	* @param value[string] 字段值
	* @param rowData[object] 表格行数据
	* @param rowIndex{number}: 选中行的索引
	**/
	function kpiDrillStyle(value, rowData, rowIndex){
		var field = this.field;
		var title = this.title;
		var platformTypeList = this.platform;
		if (value == null || ("" + value) == "") {

		} else {

			//指标值设定
			var drillStartTime = rowData.startTime;
			var drillEndTime = rowData.endTime;
			var drillSn = rowData.smallCellCode;
			var platformType = rowData.platformType;
			var enbBtsId = rowData.bts_id; // enb 中的 btsId 有值时才能钻取
			var enbEci = rowData.eci; // enb 中的 eci 有值时才能钻取
			
			var gnbNrCGI = rowData.nrCGI;//gnb 独有的参数
			var commonPlmnId = rowData.plmnId; //enb， gnb 共有的参数

			//判断是否支持当前平台
			if(platformTypeList.indexOf(platformType) != -1 || platformTypeList.includes('ALL') != -1){
				//支持当前平台
				return "<a style='color:#1DA3FC;text-decoration:underline' href='#' onclick='kpiDrillClick(&quot;"+ drillStartTime +"&quot;,&quot;" +drillEndTime +"&quot;,&quot;" + drillSn + "&quot;,&quot;" + field +"&quot;,&quot;"+ title +"&quot;,&quot;" + gnbNrCGI + "&quot;,&quot;" + commonPlmnId + "&quot;,&quot;" + enbEci + "&quot;,&quot;" + enbBtsId + "&quot; )'>"+value+"</a>";
			}else{
				return value;
			}

		}
		return value;
	}

	/**
	* 格式化指标下钻  - 格式化
	* @param startTime[string] 开始时间
	* @param endTime[string] 结束时间
	* @param smallCellCode{string}: smallCellCode
	* @param field[string] 模板 id
	* @param title{string}: 字段名称
	* @param btsId{string}: btsId
	**/
	function kpiDrillClick(startTime,endTime,smallCellCode,field,title, nrCGI, plmnId, enbEci, enbBtsId){
		$("#kpiDrillDiv").animate({right:'0px'},500,function(){
			$("#kpiDrillDiv").panel({
				 width:900,
				 href:'${ctx}/pm/template/viewStatisDetail.action', // 指标详情页面
				 onLoad:function(){
					var curDataUrl = '', params = {};

					if(kpiQueryVue.currentNetworkType == 'enb'){
						curDataUrl = '${ctx}/pm/template/getKPIDrillDataList.action';
						params = {
							timeZone : timeZone,
						 	startTime: startTime,
						 	endTime: endTime,
						 	kpiId: field,
						 	tempId: tempId,
						 	smallCellCode: smallCellCode
						}
						//enb 下钻需传 btsId 和 plmnId 和 eci
						if(enbEci && enbEci != 'undefined' && enbEci != 'null' && enbEci != ''){
							params.eci = enbEci;
						}
						if(enbBtsId && enbBtsId != 'undefined' && enbBtsId != 'null' && enbBtsId != ''){
							params.btsId = enbBtsId;
						}
						if(plmnId && plmnId != 'undefined' && plmnId != 'null' && plmnId != ''){
							params.plmnId = plmnId;
						}
			 	    }else if(kpiQueryVue.currentNetworkType == 'gnb'){
			 		     //5g
			 		    curDataUrl = '${ctx}/gnb/pm/template/getKPIDrillDataList.action';
						params = {
							timeZone : timeZone,
						 	startTime: startTime,
						 	endTime: endTime,
						 	kpiId: field,
						 	tempId: tempId,
						 	smallCellCode: smallCellCode,
							nrCGI: nrCGI,
							plmnId: plmnId
						}
			 	    }else if(kpiQueryVue.currentNetworkType == 'egw'){
			 	    	 //eGW
			 		    curDataUrl = '${ctx}/egw/pm/template/getKPIDrillDataList.action';
						params = {
							timeZone : timeZone,
						 	startTime: startTime,
						 	endTime: endTime,
						 	kpiId: field,
						 	tempId: tempId,
						 	smallCellCode: smallCellCode
						}
			 	    }

					$("#kpiDrillDatagrid").datagrid({
						url:curDataUrl,
						queryParams : params
					})
				 }
			})
		});
	}

	//关闭指标下钻弹窗
	function closeKpiDrillDiv(){
		$("#kpiDrillDiv").html("").animate({right:'-2000px'},500);
	}
	//关闭导出页面
	function closeKpiExportDiv(){
		$("#exportKpiDiv").slideUp(500,function(){
			$("#exportKpiDiv").html("");
		});
	}
</script>