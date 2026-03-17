<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
	.chartTemplateContent {
		height: 100%;
		display: flex; 
		justify-content: space-between;
		overflow: hidden;
	}
	.chartTemplateContent .centerChartsTemplateBox {
		flex: 1;
		height: 100%;
		background: #FBFBFB;
		overflow: auto;
	}
	.chartTemplateContent .chartTemplateQueryBox {
		height: 46px; 
		line-height: 46px;
		width: 100%; 
		position: relative;
	}
	.chartTemplateContent .chartTemplateTimeline {
		display: flex; 
		justify-content: center; 
		align-items: center;
	}
	.chartTemplateContent .el-date-editor .el-input__inner {
		cursor: pointer;
	}
	.chartTemplateContent .chartTemplateTimeStep {
		display: inline-block
	}
	.chartTemplateContent .chartTemplateTimeStep .el-icon:before {
		font-size: 16px !important;
	}
	.chartTemplateContent .chartTemplateDataBox {
		position: relative; 
		display: inline-block; 
		height: 30px; 
		line-height: 30px; 
		width: 190px; 
		text-align: center; 
		cursor: pointer;
	}
	.chartTemplateContent .chartTemplateDataBox {
		font-weight: bold; 
		color: #363B4E; 
		font-size: 14px; 
		padding: 0;
	}
	.chartTemplateContent .kpiStepDateClass {
		width: 180px;
		height: 30px;
		position: absolute;
		left: 0px;
		top: 0px;
		z-index: 99;
	}
	.chartTemplateContent .kpiStepDateClass .el-input__inner {
		opacity: 0;
	}
	.chartTemplateContent .kpiStepDateClass .el-input__icon {
		display: none;
	}
	.chartTemplateContent .chartTemplateRightTimeSwitch {
		margin-left: 0px;
		font-size: 20px;
	}
	.chartTemplateContent .chartTemplateRadioButton {
		margin: 0 0px 0px 25px; 
		display: inline-block;
	}
	.chartTemplateContent .chartTemplateAddBtn {
		right: 26px;
		top: 14px;
	}

	.chartTemplateContent .chartTemplateBox {
		height: calc( 100% - 46px); 
		overflow: auto;
	}
	.chartTemplateContent .noKpiDataContentCls {
		padding: 0 0 20px 0; 
		border: 1px solid #D5DCEC;
		margin: 10px 10px 0 10px;
		height: 560px;
		border-radius: 4px;
	}
	.chartTemplateContent .noKpiDataContentCls .noKpiDataTip {
		margin: 170px auto;
		text-align: center;
	}
	.chartTemplateContent .noKpiDataContentCls .noKpiDataPic {
		width: 120px; 
		height: 120px;
		display: inline-block;
		background: url(${ctx}/css/images/chartPic/chartPic.png) no-repeat;
	}
	.chartTemplateContent .noKpiDataContentCls .commonNotes {
		font-size: 12px;
		color: #8E8E8E;
	}
	.chartTemplateContent .noKpiDataContentCls .submitBtn {
		height: 24px;
		padding: 0 20px;
		min-width: 80px;
		line-height: 22px;
		font-size: 12px;
		color: var(--main-color);
		background-color: rgba(var(--main-color-rgba1),0.1);
    	border-color: var(--main-color);
	}
	.chartTemplateContent .noKpiDataContentCls .submitBtn:hover,
	.chartTemplateContent .noKpiDataContentCls .submitBtn:focus,
	.chartTemplateContent .noKpiDataContentCls .submitBtn:active {
		background-color: rgba(var(--main-color-rgba1),0.2);
	}
	.chartTemplateContent .chartTemplateItemCls {
		margin-bottom: 10px;
	}
	.chartTemplateContent .kpiTemplageChartClass {  
		position: relative;
		flex: 1 0 48%;  
		margin: 10px 10px 0 10px; 
		height: 690px; 
		border-radius: 4px;
		border: 1px solid #D5DCEC;  
		background: #FFFFFF; 
	}
	.chartTemplateContent .chartOperCls {
		position: absolute; 
		right: 20px;
		top: 20px;
	}
	.chartTemplateContent .chartOperCls .chartOperBox {
		border: 1px solid #D5DCEC;
		border-radius: 4px;
		padding: 4px 10px;
		font-size: 12px;
		color: rgba(0, 0, 0, 0.8);	
	}
	.chartTemplateContent .chartOperCls .chartOperBox:hover,
	.chartTemplateContent .chartOperCls .chartOperBox:focus,
	.chartTemplateContent .chartOperCls .chartOperBox:active {
		color: var(--main-color);
		border-color: var(--main-color);
	}
	.chartTemplateContent .chartOperCls .chartOperBox .chartOperIcon {
		font-size: 14px;
		margin-top: 2px;
		margin-right: 4px;
	}
	.chartTemplateContent .kpiChartTemplateTitles { 
		margin: 50px 60px 0;
		color: rgba(0, 0, 0, 0.8); 
		font-size: 14px; 
		text-align: center; 
		font-weight: bold;
		white-space: normal;
		word-wrap: break-word;
		overflow-wrap: break-word;
		word-break: break-all;
	}
	.chartTemplateContent .chartTemplateMainBox {
		width: 100%; 
		height: auto; 
		padding: 0 0 20px 0;
	}
	.chartTemplateContent .chartTemplateRightWarp {
		position: relative; 
		flex: 0 1 530px; 
		height: calc(100% - 20px); 
		margin: 10px 10px 10px 0px;
		overflow: hidden;
	}
	.chartTemplateContent .chartTemplateRightForm {
		padding: 10px 20px 20px; 
		position: relative;
	}
	.chartTemplateContent .chartTemplateRightFormRadio {
		position: absolute;
		right: 68px;
		top: 58px;
		height: 26px;
		line-height: 26;
	}
	.chartTemplateContent .chartTemplateSelectDevices {
		padding: 10px 0px 0; 
		justify-content: space-between;
	}
	.chartTemplateContent .chartTemplateRequied {
		color: #FF4614; 
		font-size: 14px;
	}
	.chartTemplateContent .chartTemplateSelectDevicesTable {
		margin: 10px 0 0 0 !important; 
		border: 1px solid #D5DCEC;
	}
	.chartTemplateContent .kpiTemplateChartList {
		width: 100%; 
		overflow: hidden; 
		height: 570px; 
		overflow-x: auto; 
	}
	.chartTemplateContent .kpiFunstionOption,
	.chartTemplateContent .kpiFunstionOption .el-input  {
		width: 122px;
	}
</style>
<!-- 图表界面 -->
<div id="chartTemplatesContent_${randomValue}" class="chartTemplateContent">
	<!--图表-->
	<div class='centerChartsTemplateBox'>
		<div class='chartTemplateQueryBox'>
			<div class='chartTemplateTimeline'>
				<div class='chartTemplateTimeStep'>
					<i class="el-icon el-icon-circle-left" @click="kpiChartTemplateStepTimeChange(-1)"></i>

					<div class='chartTemplateDataBox'>
						<span>{{kpiChartTemplateStepTime}}</span>

						<el-date-picker ref="kpiStepDate" :clearable="false" class="kpiStepDateClass"
							:picker-options="kpiChartStepOptions"
							@change="kpiChartTemplateStepDateChange" 
							v-model="kpiChartTemplateStepDate" :type="kpiChartTemplateStepDateType">
						</el-date-picker>
					</div>
					<i class="el-icon el-icon-circle-right chartTemplateRightTimeSwitch" :class="{disabled: isCurrentTime}" @click="kpiChartTemplateStepTimeChange(1)"></i>
				</div>
				<!--获取图形图表数据的时间参数：day：2024-10-24； week:2024-10-24 00:00:00；  month: 2024-10-->
				<el-radio-group size="mini" v-model='kpiChartTemplatePeriod' class="commonRadioButton chartTemplateRadioButton" @change="kpiChartTemplatePeriodChange">
					<el-radio-button v-show='kpiChartTemplateDayShow == true' label="0">{{tianLiDu}}</el-radio-button>
					<el-radio-button label="1">{{zhouLiDu}}</el-radio-button>
					<el-radio-button label="2">{{yueLiDu}}</el-radio-button>
				</el-radio-group>
			</div> 

			<div v-if="isWritable" class="newIconBoxCls-bt chartTemplateAddBtn" @click="addKpiChartTemplateClick" :tip=tianJia>
				<i class="el-icon el-icon-plus" ></i>
			</div>
		</div>

		<div class="chartTemplateBox">
			<!--<div v-if='kpiGraphicTemplates.length == 0' id="noKpiDataContent" ref="noKpiDataContent" class='noKpiDataContentCls'>-->
			<div v-if='kpiChartemplateDataEmpty' id="noKpiDataContent" ref="noKpiDataContent" class='noKpiDataContentCls'>
				<div class='noKpiDataTip'>
					<div class='noKpiDataPic'></div>
					<p class="commonText14" style="padding: 0 0 10px;">{{zanWuShuJu}}</p>
					<p v-if="isWritable" class="commonNotes">{{tianJiaSheBeiHeZhiBiao}}</p>
					<el-button v-if="isWritable" @click="addKpiChartTemplateClick" class="submitBtn" style="margin:26px 0;">{{tianJia}}</el-button>
				</div>
			</div>

			<!--动态生成多个图表数据-->
			<div v-else class="chartLoading">
				<div v-for="(item,index) in kpiGraphicTemplates" class="chartTemplateItemCls">
					<!--判断 item 中是否含有 (devices or groups) && indicators  字段 -->
					<div class="kpiTemplageChartClass">
						<div class='commonFlex chartOperCls'>
							<div v-if="isWritable" class='commonFlex chartOperBox' style="margin-right: 4px;" @click="modifyKpiChartTemplateClick(item)">
								<i class="el-icon el-icon-operation-edit chartOperIcon"></i>
								<p>{{xiuGai}}</p>
							</div>
							<div v-if="isWritable" class='commonFlex chartOperBox' @click="deleteKpiChartTemplateClick(item)">
								<i class="el-icon el-icon-operation-delete chartOperIcon"></i>
								<p>{{shanChu}}</p>
							</div>
						</div>
						<div v-if="item.indicators" class="kpiChartTemplateTitles">
							 <div v-for="(items, index) in item.indicators" style="display: inline-flex;">
								<div v-if="items.indicator_name">{{items.indicator_id}}({{items.indicator_name}})<i v-if="index < item.indicators.length - 1" style="margin: 0 5px;">/</i></div>
								<div v-else>{{items.indicator_id}}<i v-if="index < item.indicators.length - 1" style="margin: 0 5px;">/</i></div>
							</div>
						</div> 
						<div :id="'chart_main_' + item.id" ref="chartMainDiv" class='chartTemplateMainBox'>		
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>

	<!--右侧添加设备、设备组或指标的窗口-->
	<div v-if='selectDevicesGroupKpiShow' class='rightWarp chartTemplateRightWarp'>
		<div class='rightWarpLayer'>
			<div class='rightBoxHeaderHasTip'>
				<div class='headerText'>
					<span>{{kpiChartTemplateTitle}}</span>
					<span class='closeIconBox' @click='cancelKpiChartTemplate'><i class='el-icon el-icon-close'></i></span>
				</div>
			</div>
			<div class='rightWarpLayerContent'>
				<el-form ref="deviceKpiSelectForm" label-position="top" :model="deviceKpiSelectForm" :rules="deviceKpiSelectRules" class='chartTemplateRightForm'>
					<el-radio-group v-if='templateChartNetType == "enb"' v-model="chartActiveName" size="mini" class='chartTemplateRightFormRadio'>
						<el-radio-button label="device">{{KPISheBei}}</el-radio-button>
						<el-radio-button label="deviceGroup">{{sheBeiZu}}</el-radio-button>
					</el-radio-group>
					<!--设备-->
					<div v-if='chartActiveName == "device"'>
						<div class='commonFlex chartTemplateSelectDevices'>
							<div class="commomFlex">
								<span class='chartTemplateRequied'>*</span>
								<span class='commonText14'>{{sheBeiXuanZe}}</span>
								<span class='commonTextNormal12'>{{zuoKuoHao}}{{zuiDuoXuanZe}} {{kpiChartTemplateDeviceMaxCount}}{{youKuoHao}}</span> 
							</div>
							<i class="el-icon el-icon-operation-clear" @click="clearDeviceOrGroupSelect('device')"></i>
						</div>
						
						<el-ctable id="selectDeviceList" class='chartTemplateSelectDevicesTable'
							ref="devicesTable"
							:row-key="devicesRowKey"
							:default-checked="defaultCheckedDevicesKeys"
							:url="devicesUrl"
							:limit="kpiChartTemplateDeviceMaxCount"
							:height="'290px'"
							:show-pager="false"
							:rownumber="true"
							:query-params="devicesSearchParams"
							@selection-change="devicesChange"
							style="border:1px solid #DFE2EE;">
							<template slot='toolbar'>
								<div class="queryGroup">
									<el-input class='pairgrid-query' v-model="devicesSearchText" @keyup.enter.native="devicesQuery" :placeholder="devicesPlaceholder"></el-input>
									<i @click='devicesQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
							</template>
							<el-table-column label='' type="selection" :reserve-selection="true"></el-table-column>
							<el-table-column v-if="templateChartNetType == 'enb' || templateChartNetType == 'gnb'" :label=xiaoZhanBianMa prop="serialNumber" min-width="120"></el-table-column>
							<el-table-column v-if="templateChartNetType == 'enb'" prop='hostName' :label=hostName></el-table-column>

							<el-table-column v-if="templateChartNetType == 'enb'" prop='cellId' label='<%=rb.getString("XIAOQUID")%>'>
								<template slot-scope="scope">
									<!--判断  4g cellId 字段-->
									<div v-if="scope.row.cellId && scope.row.cellId != '' && scope.row.cellId != null &&scope.row.cellId != undefined">{{scope.row.cellId}}</div>
									<!--判断 2g bts_id-->
									<span v-if="scope.row.bts_id && scope.row.bts_id != '' && scope.row.bts_id != null &&scope.row.bts_id != undefined">{{scope.row.bts_id}}</span>
								</template>
							</el-table-column>

							<!--只是显示，无其他作用-->
							<el-table-column v-if="templateChartNetType == 'enb' && chartLevelType == 'plmn'" prop='plmnId' label='<%=rb.getString("GNBPLMNBiaoShi")%>'></el-table-column>

							<el-table-column v-if="templateChartNetType == 'gnb'" prop='hostName' :label=gNBMingCheng></el-table-column>
							<el-table-column v-if="templateChartNetType == 'gnb'" prop='cellId' label='nrCGI'></el-table-column>
							
							<el-table-column v-if="templateChartNetType == 'egw'" prop='serialNumber' :label=eGWBianMa></el-table-column>
							<el-table-column v-if="templateChartNetType == 'egw'" prop='hostName' :label=eGWMingCheng></el-table-column>
						</el-ctable>
						<el-form-item prop='devices' required style="margin-bottom: 20px;" label-width="0">
							<el-input v-model='deviceKpiSelectForm.devices' v-show="false"></el-input>
						</el-form-item>
					</div>
					<!--设备组 目前只有 enb 网元有-->
					<div v-if='chartActiveName == "deviceGroup" && templateChartNetType == "enb"'>
						<div class='commonFlex chartTemplateSelectDevices'>
							<div class="commomFlex">
								<span class='chartTemplateRequied'>*</span>
								<span class='commonText14'>{{sheBeiZuXuanZe}}</span>
								<span class='commonTextNormal12'>{{zuoKuoHao}}{{zuiDuoXuanZe}} {{kpiChartTemplateDeviceMaxCount}}{{youKuoHao}}</span> 
							</div>
							<i class="el-icon el-icon-operation-clear" @click="clearDeviceOrGroupSelect('group')"></i>
						</div>

						<el-ctable id="selectGroupList" class='chartTemplateSelectDevicesTable'
							ref="groupsTable"
							:row-key="'group_id'"
							:limit="kpiChartTemplateDeviceMaxCount"
							:height="'290px'"
							:default-checked="defaultCheckedGroupsKeys"
							:url="groupUrl"
							:show-pager="false"
							:rownumber="true"
							:query-params="groupsSearchParams"
							@selection-change="groupsChange"
							style="border:1px solid #DFE2EE;">
							<template slot='toolbar'>
								<div class="queryGroup">
									<el-input class='pairgrid-query' v-model="groupsSearchText" @keyup.enter.native="groupsQuery" placeholder='<%=rb.getString("SheBeiZuMingCheng")%>'></el-input>
									<i @click='groupsQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
							</template>
							<el-table-column label='' type="selection" :reserve-selection="true"></el-table-column>
							<el-table-column :label=sheBeiZuMingCheng prop="group_name"></el-table-column>
						</el-ctable>

						<el-form-item prop='groups' required style="margin-bottom: 20px;" label-width="0">
							<el-input v-model='deviceKpiSelectForm.groups' v-show="false"></el-input>
						</el-form-item>
					</div>
					<!--KPI-->
					<div>
						<div class='commonFlex chartTemplateSelectDevices'>
							<div class="commonFlex">
								<span class='chartTemplateRequied'>*</span>
								<span class='commonText14'>{{zhiBiaoXuanZe}}</span>
								<span class='commonTextNormal12'>{{zuoKuoHao}}{{zuiDuoXuanZe}} {{kpiChartTemplateIndicatorMaxCount}}{{youKuoHao}}</span> 
							</div>
							<i class="el-icon el-icon-operation-clear" @click="clearDeviceOrGroupSelect('kpi')"></i>
						</div>
						<el-ctable id="selectKpiList" class='chartTemplateSelectDevicesTable'
							ref="kpisTable"
							:row-key="'kpiId'"
							:default-checked="defaultCheckedKpisKeys"
							:url="kpiUrl"
							:limit="kpiChartTemplateIndicatorMaxCount"
							:height="'290px'"
							:show-pager="false"
							:rownumber="true"
							:query-params="kpisSearchParams"
							@selection-change="kpisChange">
							<template slot='toolbar'>
								<div class='commonFlex commonFlexCenter'>
									<div class="queryGroup">
										<el-input class='pairgrid-query' v-model="kpisSearchtext" @keyup.enter.native="kpisQuery" placeholder='<%=rb.getString("ZhiBiaoID")%>'></el-input>
										<i @click='kpisQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
									</div>
									<el-select v-model="kpiCatagoryId" @change='kpiFunChange' class='kpiFunstionOption'>
										<el-option v-for="item in kpiFunList" :key="item.catagoryId" :label="item.catagoryName" :value="item.catagoryId"></el-option>
									</el-select>
								</div>
							</template>
							<el-table-column label='' type="selection" :reserve-selection="true"></el-table-column>
							<el-table-column prop='kpiId' :label=zhiBiaoID>
								<template slot-scope="scope">
									<span>{{scope.row.kpiId}}{{zuoKuoHao}}{{scope.row.kpiName}}{{youKuoHao}}</span>
								  </template> 
							</el-table-column>
						</el-ctable>

						<el-form-item prop='indicators' required style="margin-bottom: 20px;" label-width="0">
							<el-input v-model='deviceKpiSelectForm.indicators' v-show="false"></el-input>
						</el-form-item>
					</div>
				</el-form>
			</div>
			<div class='commonFlex commonBorderTop commonFormFotter'>
				<div>
					<el-button type="primary" @click="addOrModifyKpiChartTemplateSubmit" :disabled="addOrModifyBtnDisabled">{{queDing}}</el-button>
					<el-button @click="cancelKpiChartTemplate">{{quXiao}}</el-button>
				</div>
			</div>
		</div>
	</div>
</div>

<script type="text/javascript">
	/*5g 设备存在多小区的情况:
		设备会存在相同的 sn, smallCellCode,会将原有的唯一标识(smallcellcode)覆盖; 现增加 uniqueId(smallcellCode + cell Id )字段，用于作为唯一标识;
		现逻辑： 相同 smallcellcode 的数据，含有多个 cell Id 时，每个cell id 按照一条数据来记录;
		已注册设备：
		    已安装的设备， 将会在列表展示
			未安装的设备， 将不会在列表展示
			同时将会修改 模板列表对应的设备列表，将不会展示未安装的设备；包含已选列表中的未安装设备
	*/
	//国际化变量
	var tianLiDu = '<%=rb.getString("Tian")%>',
		zhouLiDu = '<%=rb.getString("Zhou")%>',
		yueLiDu = '<%=rb.getString("Yue")%>',
		xiaoShiLiDu = '<%=rb.getString("XiaoShi")%>',
		tianJia = '<%=rb.getString("TianJia")%>',
		xiuGai = '<%=rb.getString("XiuGai")%>',
		shanChu = '<%=rb.getString("ShanChu")%>',
		zanWuShuJu = '<%=rb.getString("MeiShuJu")%>',
		tianJiaSheBeiHeZhiBiao = '<%=rb.getString("tianJiaSheBeiHeZhiBiao")%>',
		queDing = '<%=rb.getString("QueDing")%>',
		quXiao = '<%=rb.getString("QuXiao")%>',
		queRen = '<%=rb.getString("QueRen")%>',
		queRenShanChu = '<%=rb.getString("QueRenShanChu")%>',
		chengGong = '<%=rb.getString("ChengGong")%>',
		qingDengDai = '<%=rb.getString("QingDengDai")%>',
		jieXiZhong = '<%=rb.getString("JieXiZhong")%>',
		xiaoZhanBianMa = '<%=rb.getString("XiaoZhanBianMa")%>',
		hostName = '<%=rb.getString("HostName")%>',
		gNBMingCheng = '<%=rb.getString("GNBMingCheng")%>',
		KPISheBei  = '<%=rb.getString("KPISheBei")%>',
		sheBeiZu = '<%=rb.getString("kpiZu")%>',
		sheBeiXuanZe = '<%=rb.getString("SheBeiXuanZe")%>',
		sheBeiZuXuanZe = '<%=rb.getString("SheBeiZuXuanZe")%>',
		zuoKuoHao = '<%=rb.getString("ZuoKuoHao")%>',
		youKuoHao = '<%=rb.getString("YouKuoHao")%>',
		sheBeiZuMingCheng= '<%=rb.getString("SheBeiZuMingCheng")%>',
		jiZhanBianMaJiZhanMingCheng = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>';
		eGWBianMa = '<%=rb.getString("eGWBianMa")%>',
		eGWMingCheng = '<%=rb.getString("eGWMingCheng")%>',
		zhiBiaoID = '<%=rb.getString("ZhiBiaoID")%>',
		zhiBiaoMingCheng = '<%=rb.getString("ZhiBiaoMingCheng")%>',
		zhiBiaoXuanZe = '<%=rb.getString("ZhiBiaoXuanZe")%>',
		zuiDuoXuanZe = '<%=rb.getString("ZuiDuoXuanZe")%>',
		qingXuanZeSheBei = '<%=rb.getString("QingXuanZeSheBei")%>',
		qingXuanZeSheBeiZu = '<%=rb.getString("QingXuanZeSheBeiZu")%>',
		qingXuanZeZhiBiao = '<%=rb.getString("QingXuanZeZhiBiao")%>',
		tianJiaShuLiang = '<%=rb.getString("TianJiaShuLiang")%>',
		tuBiaoTianJiaShuLiang = '<%=rb.getString("TuBiaoTianJiaShuLiang")%>',

		//当前时间 年-月-日 时-分-秒
		kpiChartTemplateEndTime = formatDate(new Date(gloableTime)),
		//图表的开始查询时间
		kpiChartTemplateStartTime = formatDate(new Date(kpiChartTemplateEndTime)).substring(0,10), //年-月-日
		//年-月-日
		periodChangeTime = kpiChartTemplateStartTime,
		
		kpiChartTemplate = new Vue({
		el: '#chartTemplatesContent_${randomValue}',
		data(){
			var vm = this,
				validateDevice = function(rule,value,callback) { 
					if(vm.chartActiveName == "device" && value.length == 0) {
						callback(qingXuanZeSheBei);
					}else {
						callback();
					}
				},
				//校验设备组
				validateGroup = function(rule,value,callback) { 
					if(vm.chartActiveName == 'deviceGroup' && value.length == 0) {
						callback(qingXuanZeSheBeiZu);
					}else {
						callback();
					}
				},
				validateKpi = function(rule,value,callback) { 
					if(value.length == 0) {
						callback(qingXuanZeZhiBiao);
					}else {
						callback();
					}
				};
			return {
				kpiChartTemplateEndTime: kpiChartTemplateEndTime,
				kpiChartTemplateStartTime: kpiChartTemplateStartTime,
				periodChangeTime: periodChangeTime,
				kpiChartTemplatePeriod: '0', //图表选择粒度  默认为天
				kpiChartTemplateStepTime: '',//当前显示时间点
				kpiChartTemplateStepDate: '',
				kpiChartTemplateDayShow: true,				

				tempId: '', //模板id
				//--------------------------------------------------------    以下配置满足 eNB, gNB, EGW 等网元
				//可规划多少个图形图表， 默认10个，单现有逻辑可配置动态化数据
				kpiChartTemplateMaxCount: '',
				//可选设备或设备组的数量，默认5个，单现有逻辑可配置动态化数据
				kpiChartTemplateDeviceMaxCount: '', 
				//可选指标的数据，默认3个，单现有逻辑可配置动态化数据
				kpiChartTemplateIndicatorMaxCount: '', 
				//KPI 图形模板数据
				kpiGraphicTemplates: [], 
				kpiChartemplateDataEmpty: false,
				//右侧添加设备、设备组或指标的窗口
				selectDevicesGroupKpiShow: false,
				devicesUrl: '',
				//网元共享数据
				groupUrl: '', 
				kpiUrl: '',

				deviceKpiSelectForm: {
					devices: '',
					groups: '',
					indicators: ''
				},
				deviceKpiSelectRules: {
					devices: [
						{ required: true, validator: validateDevice }
					],
					groups: [
						{ required: true, validator: validateGroup }
					],
					indicators: [
						{ required: true, validator: validateKpi }
					]
				},
				chartActiveName: 'device',

				defaultCheckedDevicesKeys: [],
				defaultCheckedGroupsKeys: [],
				defaultCheckedKpisKeys: [],

				devicesSelection: [],
				groupsSelection: [],
				kpisSelection: [],

				devicesPlaceholder: '',
				devicesSearchParams: {
					searchText: '',
					tempId: kpiQueryVue.templateTabsValue
				},
				devicesSearchText: '',
				
				groupsSearchParams: {
					searchText: '',
					tempId: kpiQueryVue.templateTabsValue
				},
				groupsSearchText: '',
				
				kpisSearchParams: {
					searchText: '',
					catagoryId: '',
					queryType: 'mgmt',
					isEnable: '1',
					tempId: kpiQueryVue.templateTabsValue
				},
				kpisSearchtext: '',
				kpiFunList: [],
				kpiCatagoryId: '',

				//修改或添加时的操作类型	
				operationType: '', 
				//修改时，当前操作的行数据
				kpiChartTemplateRow: {}, 
				addOrModifyBtnDisabled: false,
				chartLevelType: kpiQueryVue.levelType
			}
		},
		computed: {
			templateChartNetType() {
				return kpiQueryVue.currentNetworkType ? kpiQueryVue.currentNetworkType : sysMain.headType;
			},
			//设备列表的 rowKey
			devicesRowKey(){
				var vm = this, 
					rowKey = '';

				if(vm.templateChartNetType == 'egw'){
					rowKey = 'smallCellCode';
				} else {
					rowKey = 'uniqueId';
				}

				return rowKey;
			},
			//根据当前粒度，置灰 后一天，周，月 circle-right 图标
			isCurrentTime() {
				var vm = this, bool = true,
					stepStr = '',
					curStr ='';
				//0-天 1-周 2-月
				if(vm.kpiChartTemplatePeriod == '0'){
					stepStr = (vm.kpiChartTemplateStepTime||'').replace(/-/g,''),
					curStr = getYesterDay(0).replace(/-/g,'');

				}else if(vm.kpiChartTemplatePeriod == '1'){
					//获取当前日期
					stepStr = vm.kpiChartTemplateStepTime.substring(0,10).replace(/-/g,'');
					//获取下一周的日期 curStr
					var nowTime = formatDate((new Date(vm.kpiChartTemplateEndTime))).substring(0,10);
					//时间选择框呈现时间
					var nowDivTime = vm.kpiChartTemplateStepTime;
					curStr = getWeekTime(nowTime).mondayTime.replace(/-/g,'');

				}else if(vm.kpiChartTemplatePeriod == '2') {
					stepStr = vm.kpiChartTemplateStepTime.substring(0,7).replace(/-/g,'');
					curStr = getYesterDay(0).substring(0,7).replace(/-/g,'');
				}

				if(stepStr-curStr<0) bool = false;

				return bool;
			},
			//根据当前粒度，设置时间选择框的类型
			kpiChartTemplateStepDateType() {
				var vm = this,
					type = 'date';

				if(vm.kpiChartTemplatePeriod == '2') {
					type = 'month';
				}

				return type;
			},
			//根据当前粒度，设置时间选择框的选项
			kpiChartStepOptions() {
				var vm = this;

				return {
					disabledDate: function(time) {
						var now = new Date(getYesterDay(0) + ' 00:00:00'),
							nowTime = getWeekTime(vm.kpiChartTemplateStartTime),
							kpiChartTemplateEndTime = new Date(nowTime.sundayTime + ' 00:00:00'); //周-结束日期

						if(vm.kpiChartTemplatePeriod == '1'){
							var maxTime = kpiChartTemplateEndTime.getTime();

							return time.getTime() > maxTime
						}else{
							return time.getTime() > now.getTime();
						}
					}
				}
			},
            isWritable() {
                return writableMap.CODE_PERFORMANCE_VIEW == true;
            }
		},
		watch: {
			devicesSelection(){
				var vm = this;

				vm.$nextTick(function(){
					//区分4g 和 5g, egw
					vm.deviceKpiSelectForm.devices = vm.devicesSelection.map((item) => { 
						if(vm.templateChartNetType == 'enb'){
							//return item.smallCellCode;

							//4g smallCellCode + plmnId; 2g: smallCellCode + btsId 
							return item.uniqueId;
						}else if(vm.templateChartNetType == 'gnb'){
							//5g smallCellCode + cellId
							return item.uniqueId;
						}
					}).join(',');
				})
			},
			groupsSelection(){
				var vm = this;

				vm.$nextTick(function(){
					vm.deviceKpiSelectForm.groups = vm.groupsSelection.map((item) => { 
						return item.group_id
					}).join(',');
				})
			},
			kpisSelection(){
				var vm = this;

				vm.$nextTick(function(){
					vm.deviceKpiSelectForm.indicators = vm.kpisSelection.map((item) => { 
						return item.kpiId 
					}).join(',');
				})
			},
		},
		methods: {
			//初始化
			chartTemplatesinit(){
				var vm = this;
				//初始化时间
				vm.kpiChartTemplateStepTime = vm.kpiChartTemplateStartTime;
				vm.kpiChartTemplateStepDate = vm.kpiChartTemplateStartTime;
				vm.tempId = kpiQueryVue.templateTabsValue;
				//查询配置项
				vm.getKpiConfig();
				//获取模板详情，设置对应的图表周期 如果模板上报周期为"24h"，则取消"天"周期,并获取图表数据
				vm.getKPITempInfoForChart();
				//初始化chart图表面板 5个设备，3个指标的数据
				vm.getDataForKpiChartTemplates();
				$(window).resize();
			},
			//查询配置项
			getKpiConfig(){
				var vm = this;

				//查询 kpi 图形模板配置项： 可新建多少个图形图表、可选几个设备或设备组、可选几个指标
				axios.get("${ctx}/pm/chart/template/getKpiChartTemplateConfigItem").then(function(response){
					var data = response.data;

					if(data){
						vm.kpiChartTemplateMaxCount = Number(data.kpi_chart_template_max_count);
						vm.kpiChartTemplateDeviceMaxCount = Number(data.kpi_chart_template_device_max_count);
						vm.kpiChartTemplateIndicatorMaxCount = Number(data.kpi_chart_template_indicator_max_count);
					}
				});
			},
			
			//获取模板详情，设置对应的图表周期 如果模板上报周期为"24h"，则取消"天"周期,并获取图表数据
			getKPITempInfoForChart(){
				var vm = this, 
					curTemplateInfoUrl = '',
					curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
					params = {
						tempId: curForm.tempId,
						timeZone: timeZone
					};

				if(vm.templateChartNetType == 'enb'){
					curTemplateInfoUrl = '${ctx}/pm/template/getTemplateInfo.action';
				}else if(vm.templateChartNetType == 'gnb'){
					curTemplateInfoUrl = '${ctx}/gnb/pm/template/getTemplateInfo.action';
				}else if(vm.templateChartNetType == 'egw'){
					curTemplateInfoUrl = '${ctx}/egw/pm/template/getTemplateInfo.action';
				}

				axios.post(curTemplateInfoUrl, stringify(params)).then(function(response){
					var data = response.data;
					
					//0-天  1-周  2-月
					if(data){
						
						//当前表格粒度与data.reportPeriod 相同，则将天粒度隐藏，默认选中周粒度
						reportCycle = data.reportPeriod;
						if(reportCycle == "1440"){ //1440: 24h
							vm.kpiChartTemplateDayShow = false; //隐藏天粒度
							vm.kpiChartTemplatePeriod = '1'; //默认选中周
							//触发周事件
							
						}else{
							vm.kpiChartTemplatePeriod = '0'; //默认选中天
						}
					}
				})
			},

			//--------------------------------------------------------------------- charts
			//查询 KPI 图形模板列表数据
			getDataForKpiChartTemplates(){
				var vm = this, 
					getKpiChartTemplateListUrl = '',
					curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
					
				vm.kpiGraphicTemplates = [];

				if(vm.templateChartNetType == 'enb' || vm.templateChartNetType == 'egw'){ 
					getKpiChartTemplateListUrl = '${ctx}/pm/chart/template/getKpiChartTemplateList';
				}else if(vm.templateChartNetType == 'gnb'){
					getKpiChartTemplateListUrl = '${ctx}/pm/chart/template/getKpiChartTemplateList?isGnb=1';
				}
				
				//查询 kpi 图形模板数据，生成图表显示
				axios.get(getKpiChartTemplateListUrl, {
					params:{
						perfTemplateId: curForm.tempId,
						timeZone: timeZone
					}
				}).then(function(response){
					var data = response.data;
					
					vm.kpiGraphicTemplates = data || [];

					if(vm.kpiGraphicTemplates.length > 0){
						vm.kpiChartemplateDataEmpty = false;
						vm.getKPIChartTemplateData(vm.periodChangeTime, vm.kpiChartTemplatePeriod);
					}else{
						vm.kpiChartemplateDataEmpty = true;
					}
				});

				$(window).resize();
			},

			/**
			* 获取图表数据 
			* @param startTimeForParam[string] 开始时间
			* @param range[string] 时间粒度
			**/
			getKPIChartTemplateData(startTimeForParam, range){
				var vm = this, 
					params = {}, 
					curChartListUrl = '',
					curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];

				params.tempId = curForm.tempId;
				params.timeZone = timeZone;
				params.timeRange = range;
				params.startTime = startTimeForParam;

				/*if(vm.templateChartNetType == 'egw'){
					curChartListUrl = '${ctx}/egw/pm/template/getChartDataList';
				}*/
				if(vm.kpiGraphicTemplates && vm.kpiGraphicTemplates.length > 0){
					vm.kpiGraphicTemplates.map((item, index) => {
						var	kpiChartTemplateDeviceData = item.devices || [], //设备数据
							kpiChartTemplateGroupData = item.groups || [], //设备组数据
							kpiChartTemplateKpiData = item.indicators || [],//指标数据
							deviceCodesList = [],
							groupIdsList = [],
						 	gnbParamsList = [],
							gnbUniqueIdList = [],
							enbUniqueIdList = [],
							kpiParamsList = [],
							perf_obj = item;

						if(vm.templateChartNetType == 'enb'){
							kpiChartTemplateDeviceData.map((obj, index) => {
								curChartListUrl = '${ctx}/pm/template/getChartDataList';
								//区分 4g 和 5g 参数
								/*if(obj.small_cell_code){
									deviceCodesList.push(obj.small_cell_code);
									params.enbCodeList = deviceCodesList.join(",");
									params.enbLimitNum = vm.kpiChartTemplateDeviceMaxCount;
									params.kpiLimitNum = vm.kpiChartTemplateIndicatorMaxCount;
									delete params.groupIdList;
								}*/
								//4G: {smallCellCode: XXX, plmnId: XXX}   2G: {smallCellCode: XXX, bts_id: XXX}
								var itemObj = {};
								if(obj.small_cell_code){
									itemObj.smallCellCode = obj.small_cell_code;
								}
								if(obj.plmnId && vm.chartLevelType == 'plmn'){
									itemObj.plmnId = obj.plmnId;
								}
								//2g
								if(obj.bts_id){
									itemObj.bts_id = obj.bts_id;
								}
								deviceCodesList.push(itemObj);
								//唯一标识
								if(obj.uniqueId){
									enbUniqueIdList.push(obj.uniqueId);
								}

								params.enbCodeList = JSON.stringify(deviceCodesList);
								params.enbLimitNum = vm.kpiChartTemplateDeviceMaxCount;
								params.kpiLimitNum = vm.kpiChartTemplateIndicatorMaxCount;
								delete params.groupIdList;
							});
							//设备组
							kpiChartTemplateGroupData.map((obj, index) => {
								curChartListUrl = '${ctx}/pm/template/getChartGroupDataList'
								if(obj.group_id){

									groupIdsList.push(obj.group_id);
									params.groupIdList = groupIdsList.join(",");
									delete params.enbCodeList;
									delete params.enbLimitNum;
									delete params.kpiLimitNum;
								}
							});
						}else if(vm.templateChartNetType == 'gnb'){
							kpiChartTemplateDeviceData.map((obj, index) => {
								curChartListUrl = '${ctx}/gnb/pm/template/getChartDataList';

								//区分 4g 和 5g 参数
								if(obj.cell_id && obj.small_cell_code){
									var itemObj = {};
									itemObj.smallCellCode = obj.small_cell_code;
									itemObj.cellId = obj.cell_id; //注意这里是cellId 还是 cell_id
									gnbParamsList.push(itemObj);									
								}else{
									gnbParamsList.push({ "smallCellCode": obj.small_cell_code});
								}
								if(obj.uniqueId){
									gnbUniqueIdList.push(obj.uniqueId);
								}
								params.enbCodeList = JSON.stringify(gnbParamsList);
								params.enbLimitNum = vm.kpiChartTemplateDeviceMaxCount;
								params.kpiLimitNum = vm.kpiChartTemplateIndicatorMaxCount;
								delete params.groupIdList;
							});
						}
						
						//指标
						kpiChartTemplateKpiData.map((item, index) => {
							if(item.indicator_id){
								kpiParamsList.push(item.indicator_id);
								params.kpiIdList = kpiParamsList.join(",");
							}
							vm.commonKpisArr = kpiParamsList;
						})						

						//$('.chartLoading').addClass('loading');
						if((deviceCodesList.length > 0 || groupIdsList.length > 0 || enbUniqueIdList.length > 0 || gnbUniqueIdList.length > 0) && kpiParamsList.length > 0){ 
							$.post(curChartListUrl, params,function(data){
								
								var	kpi_chart_maindiv = $('#chart_main_'+ item.id),
									chart_id = "kpi_chart_"+item.id,
									chart_div = '<div class="">'
													+'<div id="'+chart_id+'" class="kpiTemplateChartList"><div>'
												+'</div>';

								var charDomCount = $("#"+chart_id).size();
								//将 chart_div 添加到页面中
								if(charDomCount==0) kpi_chart_maindiv.append(chart_div);

								// 区分设备或设备组
								if(vm.templateChartNetType == 'enb'){
									/*if (deviceCodesList.length > 0) {
										vm.goSetKpiChartTemplateData(deviceCodesList, perf_obj, data, vm.periodChangeTime, range);
									}*/
									if(enbUniqueIdList.length > 0){
										vm.goSetKpiChartTemplateData(enbUniqueIdList, perf_obj, data, vm.periodChangeTime, range);
									}
									if (groupIdsList.length > 0) {
										vm.goSetKpiChartTemplateData(groupIdsList, perf_obj, data, vm.periodChangeTime, range);
									}
								}else if (vm.templateChartNetType == 'gnb'){
									if (gnbUniqueIdList.length > 0) {
										vm.goSetKpiChartTemplateData(gnbUniqueIdList, perf_obj, data, vm.periodChangeTime, range);
									}
								}
								//$('.chartLoading').removeClass('loading');
							},"json");
						}else{
							//$('.chartLoading').removeClass('loading');
						}
					})
				}
			},

			//根据查询的KPI指标数据 绘制显示图表
			// range: 0-天  1-周  2-月
			goSetKpiChartTemplateData(codes, perf_obj, chart_data, periodChangeTime, range){
				var vm = this,
					timesDataArr = []; //初始化时间节点数据
			
				if(chart_data){
					if (chart_data.length > 0) {
					    for(var i=0; i<chart_data.length; i++ ){
						    if($.inArray(chart_data[i].startTime,timesDataArr)<0){
							    timesDataArr.push(chart_data[i].startTime);
						    }
					    }
						if(range == '0'){
						    timesDataArr.push(chart_data[chart_data.length-1].endTime);
						}else {
						    timesDataArr.unshift('');
					    }
					}

					//获取图表数据中的每个 id 下的指标数据
					var deviceSmallCellCodeData = perf_obj.devices || [], //设备数据
						groupIdsData = perf_obj.groups || [], //设备组数据
					 	kpiIndicatorsData = perf_obj.indicators || [];

					if(vm.templateChartNetType == 'enb'){
						//设备
						if(deviceSmallCellCodeData.length > 0){
							var snArr = deviceSmallCellCodeData.map ((item, index) => {
								//图表展示头部标题 4g: serial_number（host_name/plmnId）  2g: serial_number（host_name/bts_id)
								var namekeys = '';
								if(item.host_name){
									namekeys = item.host_name;
								}
								if(item.plmnId && vm.chartLevelType == 'plmn'){
									//如果前面有 host_name, 则加上 / 作为分隔符
									if(namekeys){
										namekeys += "/" + '<%=rb.getString("GNBPLMNBiaoShi")%>:' + item.plmnId;
									}else{
										namekeys += '<%=rb.getString("GNBPLMNBiaoShi")%>:' + item.plmnId;
									}
								}
								//2g serialNumber (bts_id)
								if(item.bts_id){
									//如果前面有 host_name, 则加上 / 作为分隔符
									if(namekeys){
										namekeys += "/" + 'BTS ID:' + item.bts_id;
									}else{
										namekeys += 'BTS ID:' + item.bts_id;
									}
								}
								// 判断 namekeys 是否为空
								if(namekeys){
									return item.serial_number + "(" + namekeys + ")";
								}else {
									return item.serial_number;
								}
							})	
						}
						//设备组
						if(groupIdsData.length > 0){
							var snArr = groupIdsData.map ((item, index) => {
								//判断item 是否有group_name
								if(item.group_name){
									return item.group_name 
								}
							})
						}	
					}else if(vm.templateChartNetType == 'gnb'){
						//只有设备
						if(deviceSmallCellCodeData.length > 0){
							var snArr = deviceSmallCellCodeData.map ((item, index) => {
								//图表展示头部标题 serial_number（host_name/cell_id）
								if(item.cell_id && item.serial_number){
									if(item.host_name){
										return item.serial_number + '('+ item.host_name + '/' + item.cell_id +')';
									}else{
										return item.serial_number + '('+ item.cell_id +')';
									}
								}else {
									if(item.host_name){
										return item.serial_number + '('+ item.host_name +')';
									}else{
										return item.serial_number;
									}
								}
							})
						}
					}
					
					//指标
					var kpiid = kpiIndicatorsData.map((item, index) => {
						return item.indicator_name ? item.indicator_id +'('+ item.indicator_name+')' : item.indicator_id;
					})

					var no1 = [], no2=[], no3=[];
					snArr.map((item, index) => {
						//获取item 与 第一个kpi id 进行组合,存入no1
						no1.push(item +'-'+ kpiid[0]);
						no2.push(item +'-'+ kpiid[1]);
						no3.push(item +'-'+ kpiid[2]);
					})

					//这是左侧第一个Y轴的数据
					var perf_code = '', 
						yAxisIndex = 0,
						series_data = [], 
						chartNames = [];
					if(kpiIndicatorsData.length > 0){
						kpiIndicatorsData.map((item, index) => { //index 0 1 2
							yAxisIndex = index;
							perf_code = item.indicator_id;
							
							//动态实现多个Y轴的数据显示
							for(var code_index = 0; code_index < codes.length; code_index++) { 
								var seriesEle_data = [];
								//将所有的点先置为'-'
								for(var i= 0; i<timesDataArr.length; i++ ){
									seriesEle_data.push('-');
								}
								//新逻辑中都要用数据进行填充标题和series 的name, 要根据数据进行组合每组的 y轴数据
								if (chart_data.length > 0) {
									var code_value = codes[code_index];
									for (var time_index = 0; time_index < timesDataArr.length; time_index++) {
										for (var data_index = 0; data_index < chart_data.length; data_index++) {
											var perf_data_obj = chart_data[data_index];
											var time = perf_data_obj.startTime;
											var eNodeB_code = '';
											//区分 4g 和 5g 参数
											if(vm.templateChartNetType == 'gnb'){
												//gnb 应该按照 uniqueId 去匹配 ，按照uniqueId 进行传值
												eNodeB_code = perf_data_obj.uniqueId;
											}else if(vm.templateChartNetType == 'enb'){
												//判断 perf_data_obj 中是否含有 group_id
												perf_data_obj.group_id ? eNodeB_code = perf_data_obj.group_id : eNodeB_code = perf_data_obj.uniqueId;
											}else{
												//egw
												eNodeB_code = perf_data_obj.smallCellCode;
											}

											if(eNodeB_code == code_value){
												if (timesDataArr[time_index] == time ) {
													var perfValue = perf_data_obj[perf_code],  
														dataTimeIndex = range == '0'? (time_index+1):time_index; 

													if (perfValue != null && perfValue != 'N/A' && perfValue != '-') {
														seriesEle_data.splice(dataTimeIndex,1, perfValue.toLocaleString()); //Number(perfValue)
														break;
													}else{
														seriesEle_data.splice(dataTimeIndex,1,'-');
													}
												}
											}	
										}
									}
								}
								var name = [];
								if(yAxisIndex == 0){
									name = no1[ code_index];
								}else if (yAxisIndex == 1) {
									name = no2[ code_index];
								}else if (yAxisIndex == 2) {
									name = no3[ code_index];
								}
								chartNames.push(name);

								//第一个y轴数据正常，5条线， yAxisIndex： 0 正常的
								var seriesEle = { 
										name:  name,
										type : 'line',
										symbol: 'circle',
										symbolSize : range=='0'?4:10,
										showAllSymbol : true,
										yAxisIndex: yAxisIndex,
										data : seriesEle_data // 现在是5条数据
									}
		
								series_data.push(seriesEle);
							}
						})
					}
					//timesDataArr 去重
					timesDataArr = timesDataArr.filter((item, index) => {
						return timesDataArr.indexOf(item) === index;
					})
					var chart_id = "kpi_chart_" + perf_obj.id,
						legend_obj = {
							code : codes, // small_cell_code
							name: chartNames
						},
						chart_data = {
							yAxis_data : perf_obj.indicators, //图表 左侧 Y轴的数据
							legend_data : legend_obj,
							x_data : timesDataArr,
							series_data : series_data
						};
					try{
						vm.goCreateKpiChartTemplate(chart_id, chart_data, range);
					}catch(e){}
			    }
			},
			/**
			* 创建图表
			* @param elementId[string] 图表id
			* @param chart_data[object] 图表数据
			* @param range{string}: 时间范围
			**/
			goCreateKpiChartTemplate(elementId, chart_data, range) {
				var vm = this, 
					kpiChart = echarts.init(document.getElementById(elementId)); //图表对象
				// 绘制一个 多 Y轴的图表
				var chartLegendData = chart_data.legend_data || {},
					legendName = chartLegendData.name,
					yAxisData = chart_data.yAxis_data || [],
					yAxisItem = yAxisData.map((item, index) => {
						var nameData = '';
						
						//kpi id + unit
						nameData = item.indicator_id + '\n' + zuoKuoHao + item.unit + youKuoHao;
						return {
							name: nameData,
							//文字左对齐
							nameTextStyle: {
								fontSize: 12,
								align: 'left'
							},
							position: index == 0 ? 'left' : 'right',
							alignTicks: true,
							type : 'value',
							offset: index > 1 ? 130 : 0,
							axisLabel : {
								formatter: function (value) {
									// 将数字转换为普通数字格式
									return value.toLocaleString(); 
								}
							},
						}
					});
				var option = {
						//legend icon 的颜色
						color: ['#6DA1FF','#E88282','#67D972','#9982FF','#FFC300',  '#92C2F4','#16E8DE','#FF82AC','#1DBE96','#F3CA90', '#ACE380','#B3D66F','#91E7E0','#DE873F','#91AFE2'],
	
						tooltip: {
							trigger: 'axis',
							formatter : function(val) {
								if(!val[0].name) return '';
	
								if(range != "0"){
									var time = "<div>" + val[0].name.substring(0,10) + "</div>";
								}else{
									var time = "<div>" + val[0].name + "</div>";
								}
								var value = "";
								$.each(val,function(index,item){
									if (isNaN(item.data)) {
										value += "<div>" + item.marker + item.seriesName + ":-</div>";
									} else {
										value += "<div>" + item.marker + item.seriesName + ":" + item.data + "</div>";
									}
								})
								return time + value;
							},
							confine: true
						},
						grid: {
							top: '110px',
							bottom: '20px', 
							left: '60px',
							right: '20%',
							containLabel: true
						},
						//图例顶部标题 legend: sn+cell—name - kpi id + kpi name 5g: sn+cell—name+cell—Id - kpi id + kpi name
						legend: {
							data: legendName,
							left: '70px',
							right: '70px',
							top: '30px',
							orient: 'horizontal',
							type: 'scroll',
							tooltip: {
								show: true
							},
							icon:"circle",
							itemHeight: 6,
							itemWidth: 6,
							itemGap: 10
						},
						//X轴
						xAxis: [
							{
								type: 'category',
								boundaryGap : false, //解决 x轴出现空坐标
								data : chart_data.x_data,
								axisLabel : {
									formatter : function(val) {
										if(range == "0"){
											var secondTime = val.split(' ')[1];
												clock = secondTime.substring(0,2);
												val = secondTime.substring(0,5);
											if(secondTime.substring(3,5)=='00') return clock; //小时
											return val;
										}else{
											var secondTime = val.split(' ')[0];
											if(range != "2"){
												if (val.substring(8,10) == "01") return val.substring(5, 10).replace('-','.'); //月
											}
											val = secondTime.substring(8,10); //周
											return val;
										}
									},
									
									//导致 x轴不显示, 是因为它 reportCycle 无值
									interval:function(index,value){
										if(range == "0"){
											if(chart_data.x_data.length-1 != index || true){
												if(index%(60/reportCycle)==0 ){
													return true;
												}
											}
										}else if(range == "3"){
											if(value.substring(8,10) == "01"){
												return true;
											}
										}else{
											return true;
										}
									}
								},

								splitLine : {
									show : false
								},

								name : range == "0"? xiaoShiLiDu : tianLiDu,
							}
						],
						// 多个Y轴 最多三个指标
						yAxis: yAxisItem,
						
						//数据映射 legend 要与这里的值对应，保持一致
						series : chart_data.series_data,
					}
				//绘制图表
				kpiChart.setOption(option, true);
				//图表自适应
				$(window).on('resize',function(){
					setTimeout(function(){
						kpiChart.resize();
					},200);
				})
			},
		
			//-----------------------------------------------------------new -------------			
			//添加 KPI 图形图表
			addKpiChartTemplateClick(){
				var vm = this;

				vm.operationType = 'add';
				vm.chartActiveName = 'device';
				//清空设备、设备组、指标选择
				//设备组
				vm.groupsSelection = []; //清空已选数据
				vm.defaultCheckedGroupsKeys = []; //清空默认勾选数据
				//设备
				vm.devicesSelection = [];
				vm.defaultCheckedDevicesKeys = [];
				
				//指标
				vm.kpisSelection = [];
				vm.defaultCheckedKpisKeys = [];

				vm.kpiChartTemplateTitle = tianJia;
				//查询 kpi 图形模板配置项： 可新建多少个图形图表、可选几个设备或设备组、可选几个指标
				axios.get("${ctx}/pm/chart/template/getKpiChartTemplateConfigItem").then(function(response){
					var data = response.data;
					
					if(data){
						vm.kpiChartTemplateMaxCount = Number(data.kpi_chart_template_max_count);
						vm.kpiChartTemplateDeviceMaxCount = Number(data.kpi_chart_template_device_max_count);
						vm.kpiChartTemplateIndicatorMaxCount = Number(data.kpi_chart_template_indicator_max_count);

						//判断是否可以添加图表
						if(vm.kpiGraphicTemplates.length < vm.kpiChartTemplateMaxCount){

							//设备筛选 针对网元对应的提示和查詢接口
							if(vm.templateChartNetType == 'enb'){
								//设备列表数据
								vm.devicesUrl = '${ctx}/pm/template/getEnbListPageData?isGnb=0';
								vm.groupUrl = "${ctx}/pm/template/getTemplateAssociatedDeviceGroupList?isGnb=0";
								//指标列表数据
								vm.kpiUrl = '${ctx}/pm/indicatormg/getIndicatorListByPage';
								//设备查询提示
								vm.devicesPlaceholder = jiZhanBianMaJiZhanMingCheng;
							}else if(vm.templateChartNetType == 'gnb'){
								vm.devicesUrl = '${ctx}/pm/chart/template/getGnbListPageDataForAddChartTemplate';
								
								vm.kpiUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorListPageData';
								vm.devicesPlaceholder = xiaoZhanBianMa + ' / ' + gNBMingCheng;
							}else if(vm.templateChartNetType == 'egw'){
								vm.devicesUrl = '${ctx}/egw/pm/template/getEgwListPageData';
								vm.kpiUrl = '${ctx}/egw/pm/indicatormg/getIndicatorListPageData';
								vm.devicesPlaceholder = eGWBianMa + ' / ' + eGWMingCheng;
							}
							
							vm.selectDevicesGroupKpiShow = true;
							$(window).resize();
						}else{
							vm.$message.warning(tianJiaShuLiang + vm.kpiChartTemplateMaxCount + tuBiaoTianJiaShuLiang);
						}
					}
				})
			},
			//修改 KPI 图形模板
			modifyKpiChartTemplateClick(row){
				var vm = this;
				vm.operationType = 'modify';
				vm.kpiChartTemplateTitle = xiuGai;

				if(row){
					vm.kpiChartTemplateRow = row;

					//处理当前设备、设备组、指标已勾选数据回显
					if(vm.templateChartNetType == 'enb'){
						//设备列表数据
						vm.devicesUrl = '${ctx}/pm/template/getEnbListPageData?isGnb=0';
						vm.groupUrl = "${ctx}/pm/template/getTemplateAssociatedDeviceGroupList?isGnb=0";
						//指标列表数据
						vm.kpiUrl = '${ctx}/pm/indicatormg/getIndicatorListByPage';
						//设备查询提示
						vm.devicesPlaceholder = jiZhanBianMaJiZhanMingCheng;

						//设备
						if(row.devices && row.devices.length > 0){
							vm.chartActiveName = 'device';
							//vm.defaultCheckedDevicesKeys = row.devices.map((item) => { return item.small_cell_code });
							vm.defaultCheckedDevicesKeys = row.devices.map((item) => { 
								return item.uniqueId 
							});
						}
						//设备组
						if(row.groups && row.groups.length > 0){
							vm.chartActiveName = 'deviceGroup';
							
							vm.defaultCheckedGroupsKeys = row.groups.map((item) => { 
								return item.group_id;
							});
						}
					}else if(vm.templateChartNetType == 'gnb'){
						vm.devicesUrl = '${ctx}/pm/chart/template/getGnbListPageDataForAddChartTemplate';

						vm.kpiUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorListPageData';
						vm.devicesPlaceholder = xiaoZhanBianMa + ' / ' + gNBMingCheng;

						//设备
						if(row.devices && row.devices.length > 0){
							vm.chartActiveName = 'device';
							vm.defaultCheckedDevicesKeys = row.devices.map((item) => { 
								return item.uniqueId 
							});
						}
					}
					/* else if(vm.templateChartNetType == 'egw'){
						vm.devicesUrl = '${ctx}/egw/pm/template/getEgwListPageData';
						vm.kpiUrl = '${ctx}/egw/pm/indicatormg/getIndicatorListPageData';
						vm.devicesPlaceholder = eGWBianMa + ' / ' + eGWMingCheng;
					}*/

					//指标		
					if(row.indicators && row.indicators.length > 0){
						vm.defaultCheckedKpisKeys = row.indicators.map((item) => { return item.indicator_id });
					}

					vm.selectDevicesGroupKpiShow = true;
					//所有图表自适应
					$(window).resize();
				}
			},
			//添加或修改 KPI 图形模板保存
			addOrModifyKpiChartTemplateSubmit(){
				var vm = this,
					params = {},
					addOrModKpiChartTemplateUrl = '',
					curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];

					vm.operationType == 'add' ? params.id = '' : params.id = vm.kpiChartTemplateRow.id;
					

					params.indicators = vm.deviceKpiSelectForm.indicators;
					params.timeZone = timeZone;
					params.perfTemplateId = curForm.tempId; //模板id

				if(vm.templateChartNetType == 'enb'){
					addOrModKpiChartTemplateUrl = '${ctx}/pm/chart/template/addOrModKpiChartTemplate';
					//处理下发参数
					var enbParamsList = [];
					vm.devicesSelection.map((item, index) => {
						//4g: {smallCellCode:xxx, plmnId: xxx}  2g: {smallCellCode:xxx, bts_id:xxx}
						var itemObj = {};
						if(item.smallCellCode){
							itemObj.smallCellCode = item.smallCellCode;
						}
						if(item.plmnId && vm.chartLevelType == 'plmn'){
							itemObj.plmnId = item.plmnId;
						}
						if(item.bts_id){
							itemObj.bts_id = item.bts_id;
						}
						enbParamsList.push(itemObj);
					})

					vm.chartActiveName == "device" ? params.devices = JSON.stringify(enbParamsList) : params.groupIds = vm.deviceKpiSelectForm.groups;	
				}else if(vm.templateChartNetType == 'gnb'){
					addOrModKpiChartTemplateUrl = '${ctx}/pm/chart/template/addOrModKpiChartTemplate?isGnb=1';
					if(vm.devicesSelection.length > 0){
						var gnbParamsList = [];
						vm.devicesSelection.map((item, index) => {
							//注意这是是 cellId 还是 cell_id
							if(item.smallCellCode && item.cellId){
								gnbParamsList.push({
									"smallCellCode": item.smallCellCode,
									"cellId": item.cellId
								})
							}else{
								gnbParamsList.push({
									"smallCellCode": item.smallCellCode
								})
							}
						})
						params.devices = JSON.stringify(gnbParamsList);
					}
				}else {
					//vm.templateChartNetType == 'egw'
					//addOrModKpiChartTemplateUrl = '${ctx}/pm/chart/template/addOrModKpiChartTemplate';
				}

				vm.$refs.deviceKpiSelectForm.validate((valid) => {
					if(valid){
						vm.addOrModifyBtnDisabled = true;
						axios.post(addOrModKpiChartTemplateUrl, stringify(params)).then(function(response){
							var data = response.data;

							if(data["success"]){
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								})
								vm.selectDevicesGroupKpiShow = false;
								vm.getDataForKpiChartTemplates();
							}else{
								if(data.msg == 'tooManyTemplates'){
									vm.$message.error(tianJiaShuLiang + vm.kpiChartTemplateMaxCount + tuBiaoTianJiaShuLiang);
								}
							}

							vm.addOrModifyBtnDisabled = false;
						})
					}
				})
			},
			//取消添加或修改 KPI 图形模板
			cancelKpiChartTemplate(){
				var vm = this;

				vm.selectDevicesGroupKpiShow = false;
				$(window).resize();
			},
			//设备选择变化
			devicesChange(selection){
				var vm = this;

				if(selection){
					vm.devicesSelection = selection;
				}
			},			
			// 设备组列表-当选择项发生变化时触发此事件
			groupsChange(selection) {
				var vm = this;

				if(selection){
					vm.groupsSelection = selection;
				}
			},
			// kpi select change
			kpisChange(selection) {
				var vm = this;
				
				if(selection){
					vm.kpisSelection = selection;
				}	
			},
			//只清除设备或设备组、指标已选数据
			clearDeviceOrGroupSelect(type){
				var vm = this;

				//设备组
				if(type == 'group') {
					vm.groupsSelection = []; //清空已选数据
					vm.defaultCheckedGroupsKeys = []; //清空默认勾选数据
					vm.$refs.groupsTable.clearSelection();
				}
				//设备
				if(type == 'device') {
					vm.devicesSelection = [];
					vm.defaultCheckedDevicesKeys = [];
					vm.$refs.devicesTable.clearSelection();
				}
				
				//指标
				if(type == 'kpi') {
					vm.kpisSelection = [];
					vm.defaultCheckedKpisKeys = [];
					vm.$refs.kpisTable.clearSelection();
				}
			},
			//删除 KPI 图形模板
			deleteKpiChartTemplateClick(row){
				var vm = this,
					params = {
						chartTemplateId : row.id //KPI 图表模板id
					},
					deleteKpiChartTemplateUrl = '';

				if(vm.templateChartNetType == 'enb' || vm.templateChartNetType == 'egw'){
					deleteKpiChartTemplateUrl = '${ctx}/pm/chart/template/deleteKpiChartTemplate';
				}else if(vm.templateChartNetType == 'gnb'){
					deleteKpiChartTemplateUrl = '${ctx}/pm/chart/template/deleteKpiChartTemplate?isGnb=1';
				}
				vm.$confirm(queRenShanChu, queRen,{
					confirmButtonText: queDing,
					cancelButtonText: quXiao,
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					axios.post(deleteKpiChartTemplateUrl, stringify(params)).then(function(response){
						var data = response.data;

						if(data["success"]){
							vm.$message({
	    						message: '<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					})
							
							// 删除对应的图表
							vm.kpiGraphicTemplates = vm.kpiGraphicTemplates.filter((item) => { return item.id != row.id });//删除当前行
							var chart_id = "kpi_chart_" + row.id;//图表id
							$('#'+chart_id).remove();//删除图表
							$('#'+chart_id).removeAttr('_echarts_instance_'); //删除图表实例
							//vm.$nextTick(() => {
								vm.getDataForKpiChartTemplates();// 更新图表数据
							//})
						}else{
							vm.$message.error(data["msg"])
						}
					})
				}).catch(()=>{})
			},
			//获取 KPI 指标功能集下拉列表数据
			getKpisFunList(){
				var vm = this,
					curGroupListUrl = '',
					params = {
						isShowAll: '1',
						noKpiShowGroup: '0'
					};

				if(vm.templateChartNetType == 'enb'){
					curGroupListUrl = '${ctx}/pm/indicatormg/getIndicatorGroupList.action';
				}else if(vm.templateChartNetType == 'gnb'){
					curGroupListUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupList.action';	
				}else if(vm.templateChartNetType == 'egw'){
					curGroupListUrl = '${ctx}/egw/pm/indicatormg/getIndicatorGroupList.action';
				}
				
				axios.post(curGroupListUrl,stringify(params)).then(function(response){
					var data = response.data;
					
					vm.kpiFunList = data.length == 0 ? [] : data;	
				}).catch(function(error){})
			},
			//kpi 指标功能集改变
			kpiFunChange(val){
				var vm = this;

				vm.kpisSearchParams.catagoryId = val;			
			},
			//设备列表查询
			devicesQuery(){
				var vm = this;

				vm.devicesSearchParams.searchText = vm.devicesSearchText;
			},
			//设备组列表查询
			groupsQuery(){
				var vm = this;

				vm.groupsSearchParams.searchText = vm.groupsSearchText;
			},
			//kpi 指标查询
			kpisQuery(){
				var vm = this;

				vm.kpisSearchParams.searchText = vm.kpisSearchtext;
			},
			//------------------------------  old
			//日期组件中选择时间
			kpiChartTemplateStepDateChange(val) {
				var vm = this, str = '';
				if(vm.kpiChartTemplatePeriod == '1'){
					//周
					var curTime = dateformatter(val).substring(0,10);
					var nowTime = getWeekTime(curTime);
					vm.kpiChartTemplateStepTime = nowTime.mondayTime + " - "+ nowTime.sundayTime;
					str = nowTime.mondayTime + ' 00:00:00';
					vm.kpiChartTemplateStepDate = nowTime.mondayTime;
					vm.periodChangeTime = str;
				}else if(vm.kpiChartTemplatePeriod == '2') {
					//月
					var curTime = dateformatter(val).substring(0,10);
					str = curTime.substring(0,7); //2023-09
					vm.kpiChartTemplateStepTime = str;
					vm.kpiChartTemplateStepDate = str;
					vm.periodChangeTime = str;
				}else{
					//天
					str = dateformatter(val).substring(0,10);
					vm.kpiChartTemplateStepTime = str;
					vm.kpiChartTemplateStepDate = str;
					vm.periodChangeTime = str;
				}
				vm.getKPIChartTemplateData(str,vm.kpiChartTemplatePeriod);
			},
			// 初始化前一天，周，月 或后一天，周，月 时间切换
			kpiChartTemplateStepTimeChange(num) {
				var vm = this,
					type = vm.kpiChartTemplatePeriod, //0-天  1-周  2-月
					kpiChartTemplateStepTime = vm.kpiChartTemplateStepTime; //当前时间点

				if(num>0 && vm.isCurrentTime) {
					return;
				}
				//后一天
				if(num == 1){
					//当前时间
					var nowTime = formatDate((new Date(vm.kpiChartTemplateEndTime))).substring(0,10);
					//时间选择框呈现时间
					var nowDivTime = vm.kpiChartTemplateStepTime;
					//选择时间和周期之后的开始时间
					var nextTime = "";
					if(type == '1'){//周
						nowTime = getWeekTime(nowTime).mondayTime;
						nowDivTime = nowDivTime.split(" - ")[0] + ' 00:00:00';
						nextTime = formatDate(addDate(new Date(nowDivTime),7));
						showNowTime = getWeekTime(nextTime);
						vm.kpiChartTemplateStepTime = showNowTime.mondayTime + " - "+ showNowTime.sundayTime;
						vm.kpiChartTemplateStepDate = showNowTime.mondayTime;
						vm.periodChangeTime = showNowTime.mondayTime + ' 00:00:00';
					}else if(type == '2'){//月
						nowDivTime += '-01 00:00:00';
						nowTime = nowTime.substring(0,7);
						nextTime = new Date(nowDivTime);
						nextTime = addMonth(nextTime,1);
						vm.kpiChartTemplateStepTime = nextTime;
						vm.kpiChartTemplateStepDate = nextTime;
						vm.periodChangeTime = nextTime;
					}else{//天
						nowDivTime += ' 00:00:00';
						nextTime = formatDate(addDate(new Date(nowDivTime),1)).substring(0,10);
						vm.kpiChartTemplateStepTime = nextTime;
						vm.kpiChartTemplateStepDate = nextTime;
						vm.periodChangeTime = nextTime;
					}
					 //时间选择赋值
					vm.getKPIChartTemplateData(nextTime, type);
				}else {
					//天，周，月 前一天点击
					var nowDivTime = vm.kpiChartTemplateStepTime,
						prevTime = '',
						showNowTime;
						//周
					if(type == '1') {
						nowDivTime = nowDivTime.split(" - ")[0] + ' 00:00:00';
						prevTime = formatDate(addDate(new Date(nowDivTime),-7));
						showNowTime = getWeekTime(prevTime);
						vm.kpiChartTemplateStepTime = showNowTime.mondayTime + " - "+ showNowTime.sundayTime;
						vm.kpiChartTemplateStepDate = showNowTime.mondayTime;
						vm.periodChangeTime = showNowTime.mondayTime + ' 00:00:00';
					}else if(type == '2') {
						//月
						nowDivTime += '-01 00:00:00',
						prevTime = new Date(nowDivTime),
						prevTime = addMonth(prevTime,-1);
						vm.kpiChartTemplateStepTime = prevTime;
						vm.kpiChartTemplateStepDate = prevTime;
						vm.periodChangeTime = prevTime;
					}else{
						//天
						nowDivTime += ' 00:00:00';
						prevTime = formatDate(addDate(new Date(nowDivTime),-1)).substring(0,10);
						vm.kpiChartTemplateStepTime = prevTime;
						vm.kpiChartTemplateStepDate = prevTime;
						vm.periodChangeTime = prevTime;
					}

					vm.getKPIChartTemplateData(prevTime,type);
				}
			},
			// 天、周，月切换
			kpiChartTemplatePeriodChange(period) {
				var vm = this;
				vm.kpiChartTemplatePeriod = period; //当前选中粒度
				//周
				if(period == '1'){
					var nowTime = getWeekTime(vm.kpiChartTemplateStartTime);
					vm.kpiChartTemplateStepTime = nowTime.mondayTime + " - "+ nowTime.sundayTime; //周一-周日  2023-09-04--2023-09-10
					vm.periodChangeTime = nowTime.mondayTime + ' 00:00:00';
					vm.kpiChartTemplateStepDate = nowTime.mondayTime;
				}else if(period == '2'){
					 //月
					 vm.kpiChartTemplateStepTime = vm.kpiChartTemplateStartTime.substring(0,7);
					 vm.kpiChartTemplateStepDate = vm.kpiChartTemplateStartTime.substring(0,7);
					 vm.periodChangeTime = vm.kpiChartTemplateStartTime.substring(0,7);
				}else{
					//天
					vm.kpiChartTemplateStepTime = formatDate(new Date(vm.kpiChartTemplateEndTime)).substring(0,10);
					vm.kpiChartTemplateStepDate = formatDate(new Date(vm.kpiChartTemplateEndTime)).substring(0,10);
					vm.periodChangeTime = formatDate(new Date(vm.kpiChartTemplateEndTime)).substring(0,10);
				}
				 //时间选择组件赋值
				//periodChangeTime: 年-月-日；period：0-天，1-周，2-月
				vm.getKPIChartTemplateData(vm.periodChangeTime, period);
			}
		},
		mounted(){
			this.chartTemplatesinit();
			//获取指标功能集数据
			this.getKpisFunList();
		}
	})
</script>
