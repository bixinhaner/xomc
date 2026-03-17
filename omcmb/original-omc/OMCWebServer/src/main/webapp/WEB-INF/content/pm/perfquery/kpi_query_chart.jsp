<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
	.chart-content .periodSelectStyle { display: inline-block; border: 1px solid #e9e9e9;
    	border-radius: 100px;
    	height: 30px;
    	line-height: 30px; }

	.chart-content .el-tree-node__label {
		font-size: 12px;
	}
	.chart-content .rightDevicesBox,
	.chart-content .leftKpisBox {
		width: 230px;
	}
	.chart-content .centerChartsBox {
		flex: auto;
		background: #FBFBFB;
		border-left:1px solid #e9e9e9;
		border-right: 1px solid #e9e9e9;
		overflow: auto
	}
	.chart-content .kpiChartList { width: 100%; overflow: hidden; height: calc(100% - 80px); overflow-x: auto; }
	.chart-content .kpiChartCellNameClass { border: 1px solid #D5DCEC; margin: 10px 15px 0 15px; height: calc(100% - 80px); border-radius: 4px; background: #FFFFFF; }
	.chart-content .kpiChartTitles { padding: 20px; color: rgba(0, 0, 0, 0.8); font-size: 14px; text-align: center; font-weight: bold;}
	.chart-content .templateNameQuery,
	.chart-content .snTemplateNameQuery {
		padding: 10px 0;
		border-bottom:1px solid #DFE2EE
	}
	.chart-content .templateNameQuery .advanceQuery,
	.chart-content .snTemplateNameQuery .advanceQuery {
		margin: 0 10px;
		padding: 0 10px;
		height: 24px;
	}
	.chart-content .templateNameQuery .advanceQuery .el-input .el-input__inner,
	.chart-content .snTemplateNameQuery .advanceQuery .el-input .el-input__inner {
		height: 24px;
		padding: 0;
		line-height: 24px;
	}
	.chart-content .queryGroup .el-icon,
	.chart-content .templateNameQuery .advanceQuery .el-icon,
	.chart-content .snTemplateNameQuery .advanceQuery .el-icon {
		font-size: 14px;
		margin-left: 0 !important;
		line-height: 23px;
	}
	.chart-content .templateNameQuery .el-input.el-input--small{
		width: 170px;
	}
	.chart-content .snTemplateNameQuery .el-input.el-input--small{
		width: 260px;
	}
	.chart-content .el-tree .el-tree-node .is-leaf + .el-checkbox .el-checkbox__inner {
		display: inline-block;
	}
	.chart-content .el-tree .el-tree-node .el-checkbox .el-checkbox__inner {
		display: none;
	}
	.chart-content .el-date-editor .el-input__inner {
		cursor: pointer;
	}
	.chart-content .kpiChartsBox .el-tree-node__children .el-tree-node__content {
		padding-left: 0px !important;
	}
	
	.chart-content .kpiChartsBox .el-tree .el-tree-node__content {
		width: 100%;
		height: 34px;
		line-height: 34px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.06);
	}

	.chart-content .snChartsBox .ItemLabelCls {
		display: inline-block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		width: 275px;
	}
	.chart-content .kpiChartsBox .ItemLabelCls {
		display: inline-block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		width: 196px;
	}
	.chart-content .kpiStepDateClass {
		width: 180px;
		height: 30px;
		position: absolute;
		left: 0px;
		top: 0px;
		z-index: 99;
	}
	.chart-content .kpiStepDateClass .el-input__inner {
		opacity: 0;
	}
	.chart-content .kpiStepDateClass .el-input__icon {
		display: none;
	}	
	.chart-content .kpiChartTimeStep .el-icon:before {
		font-size: 16px !important;
	}
	.chart-content .snChartsBox .el-tree-node__content {
		height: auto;
		padding: 5px 0;
   	 	line-height: 22px;
		border-bottom: 1px solid rgba(0, 0, 0, 0.06);
	}
	.chart-content .snChartsBox .el-tree-node__content .snCommonItem {
		display: block;
	}
	.chart-content .snChartsBox .el-tree-node__children .el-tree-node__content {
		position: relative;
    	padding-left: 22px !important;
	}
	.chart-content .snChartsBox .el-tree-node__content>.el-checkbox {
		position: absolute;
		top: 10px;
		margin-right: unset;
	}
	.chart-content .snChartsBox .snAllItemLabelCls {
		display: inline-block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		width: 250px;
	}
	.chart-content .enbTabs .el-tabs__nav {
		width: 100%;
		border: 0;
		border-bottom: 1px solid #DFE2EE !important;
	}
	.chart-content .enbTabs .el-tabs__nav-wrap {
		width: 100%;
		max-width: inherit;
	}
    .chart-content .enbTabs .el-tabs__content {
        border-radius: 0;
    }
    .chart-content .snChartsBox .el-table__header-wrapper {
        display: none;
    }
    .chart-content .snChartsBox  .el-table--striped .el-table__body tr.el-table__row--striped td {
        background: #FFFFFF;
    }
    .chart-content .snChartsBox  .el-table--border td {
        border-right: 0;
    }
</style>
<!-- 图表界面 -->
<div id="chartContent_${randomValue}" class="chart-content" style="height:100%;display: flex; justify-content: space-between;overflow: hidden;">
	<!--kpis-->
	<div class='leftKpisBox'>
		<div class='commonFlex' style='padding: 10px 10px 0;'>
			<span class='commonGeneralBold12'><%=rb.getString("KPIXuanZeZhiBiao")%></span>
		</div>
		<div>
			<el-query class='templateNameQuery' @query="queryKpi" type="normal" placeholder='<%=rb.getString("ZhiBiaoID")%>'></el-query>
			<div style='height: calc(100% - 73px); width: 230px; overflow-y: auto;' class="chartGroupTreeBox kpiChartsBox">
				<el-tree
					ref="groupKpiTree"
					:data="kpisTreeData"
					node-key="group_name"
					:props= "{label:'group_name'}"
					:default-expanded-keys="kpidefaultExpandedKeys"
					:highlight-current="true"
					@node-click="kpisNodeChange">
					<div class="treeItemBoxCls" slot-scope="{ node,data }">
						<div v-if="data.children">
							<span class="ItemLabelCls commonGeneral12" :title="node.label">{{node.label}}</span>
						</div>
						<div v-else class="commonFlex ItemLabelCls">
							<span :title="node.label" class="commonGeneral12">{{node.label}}</span>
							<span class="commonGeneral12" v-if='data.kpi_name' :title="data.kpi_name">({{data.kpi_name}})</span>
						</div>
					</div>
				</el-tree>
			</div>
		</div>
	</div>
	<!--图表-->
	<div class='centerChartsBox'>
		<div style="height: 46px; line-height: 46px;width: 100%; ">
			<div class='timeline' style="display:flex; justify-content: center; align-items: center;">
				<div class="kpiChartTimeStep" style="display: inline-block">
					<i class="el-icon el-icon-circle-left" @click="kpiChartStepTimeChange(-1)"></i>

					<div style="position: relative; display: inline-block; height: 30px; line-height: 30px; width: 190px; text-align: center; cursor: pointer;">
						<span style=' font-weight: bold; color:#363B4E; font-size: 14px; padding: 0;'>{{kpiChartStepTime}}</span>
						<el-date-picker ref="kpiStepDate" :clearable="false" class="kpiStepDateClass"
							:picker-options="kpiChartStepOptions"
							@change="kpiChartStepDateChange" 
							v-model="kpiChartStepDate" :type="kpiChartStepDateType">
						</el-date-picker>
					</div>
					<i class="el-icon el-icon-circle-right" style="margin-left:0px;font-size: 20px;" :class="{disabled: isCurrentTime}" @click="kpiChartStepTimeChange(1)"></i>
				</div>

				<el-radio-group size="mini" v-model='chartPeriod' style="margin:0 0px 0px 25px; display: inline-block;" class="commonRadioButton" @change="changeChartPeriod">
					<el-radio-button v-show='dayShow == true' label="0"><%=rb.getString("Tian")%></el-radio-button>
					<el-radio-button label="1"><%=rb.getString("Zhou")%></el-radio-button>
					<el-radio-button label="2"><%=rb.getString("Yue")%></el-radio-button>
				</el-radio-group>
			</div>
		</div>
		<div id="chart_main_${randomValue}" ref="chartMainDiv" style='width: 100%; height: auto; padding: 0 0 20px 0;'></div>
	</div>
	<!--设备-->
	<div>
		<div class='rightDevicesBox' v-if="chartNetType == 'enb'" style="width: 320px;">
			<el-tabs v-model="chartActiveName" @tab-click='chartTabClick' class='enbTabs'>
				<el-tab-pane label='<%=rb.getString("KPISheBei")%>' name="device" key="device">
					<div class='commonFlex' style='padding: 10px 10px 0; justify-content: space-between;'>
						<div>
							<span class='commonGeneralBold12'><%=rb.getString("SheBeiXuanZe")%></span>
							<span class='commonTipSize12'>(<%=rb.getString("ZuiDuoXuanZe5")%>)</span> 
						</div>
						<i class="el-icon el-icon-operation-clear" @click="resetTreeSelet"></i>
					</div>
					<div style=" height: calc(100% - 28px);">
						<el-query class='snTemplateNameQuery' @query="querydevicesSn" type="normal" placeholder='<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>'></el-query>
						<div style='height: calc(100% - 47px); width: 320px; overflow-y: auto;' class="chartGroupTreeBox snChartsBox">
							<el-tree 
								ref="groupDevicesTree"
								:data="devicesTreeData"
								node-key="uniqueId"
								show-checkbox
								check-strictly
								:props= "{label:'group_name'}"
								:default-expanded-keys="devicesDefaultExpandedKeys"
								:default-checked-keys="devicesDefaultCheckedKeys"
								:highlight-current="true"
								@check="devicesCheckChange">
								<div class="treeItemBoxCls" slot-scope="{ node,data }">
									<div v-if="data.children">
										<span class="ItemLabelCls commonGeneral12" :title="node.label">{{node.label}}</span>
									</div>
									<div v-else class="commonFlex ItemLabelCls">
										<span class='snCommonItem commonGeneral12 snAllItemLabelCls' :title="node.label">{{node.label}}</span>
										<span class='snCommonItem commonTextNormal12 snAllItemLabelCls' v-if='data.cell_name' :title="data.cell_name"><%=rb.getString("HostName")%>: {{data.cell_name}} </span>
										<span class='snCommonItem commonTextNormal12 snAllItemLabelCls' v-if='data.sub_station_name && northOperatorScenario == "S0009"' :title="data.sub_station_name"><%=rb.getString("ZhanZhiMingCheng")%>: {{data.sub_station_name}}</span>
										<span class='snCommonItem commonTextNormal12 snAllItemLabelCls' v-if='data.cellId' :title="data.cellId"><%=rb.getString("XIAOQUID")%>: {{data.cellId}}</span>
										<span class='snCommonItem commonTextNormal12 snAllItemLabelCls' v-if='data.bts_id' :title="data.bts_id">BTS ID: {{data.bts_id}}</span>
										<span class='snCommonItem commonTextNormal12 snAllItemLabelCls' v-if='data.plmnId && levelType == "plmn"' :title="data.plmnId"><%=rb.getString("GNBPLMNBiaoShi")%>: {{data.plmnId}}</span>
									</div>
								</div>
							</el-tree>
						</div>
					</div>	
				</el-tab-pane>	
				<el-tab-pane label='<%=rb.getString("SheBeiZu")%>' name="deviceGroup" key="deviceGroup">
					<div class='commonFlex' style='padding: 10px 10px 0; justify-content: space-between;'>
						<div>
							<span class='commonGeneralBold12'><%=rb.getString("SheBeiZuXuanZe")%></span>
							<span class='commonTipSize12'>(<%=rb.getString("ZuiDuoXuanZe5")%>)</span> 
						</div>
						<i class="el-icon el-icon-operation-clear" @click="resetTreeSelet"></i>
					</div>
					<div style=" height: calc(100% - 28px);">
						<el-query class='snTemplateNameQuery' @query="querydeviceGroup" type="normal" placeholder='<%=rb.getString("SheBeiZuMingCheng")%>' style="border-bottom: 0;"></el-query>
						<div style='height: calc(100% - 47px); width: 320px; overflow-y: auto;' class="chartGroupTreeBox snChartsBox">
							<el-ctable id="deviceGroupList"
								ref="deviceGroupTable"
								:row-key="'group_id'"
								:limit="5"
								:default-checked="defaultCheckedGroupKeys"
								:url="deviceGroupUrl"
								height="100%"
								:show-pager="false"
								:rownumber="false" 
								:query-params="queryDeviceGroupParams" 
								@load-success="loadSuccessDeviceGroup"
								@selection-change="deviceGroupChange">
								<el-table-column label='' width='20px' type="selection" :reserve-selection="true"></el-table-column>
								<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' prop="group_name"></el-table-column>
							</el-ctable>
						</div>
					</div>
				</el-tab-pane>	
			</el-tabs>
		</div>
		<div class='rightDevicesBox' style="width: 320px;" v-else>
			<div class='commonFlex' style='padding: 10px 10px 0;'>
				<span class='commonGeneralBold12'><%=rb.getString("SheBeiXuanZe")%></span>
				<span class='commonTipSize12'>(<%=rb.getString("ZuiDuoXuanZe5")%>)</span> 
				<i class="el-icon el-icon-operation-delete" style="margin-left: 10px;" @click="resetTreeSelet"></i>
			</div>
			<div>
				<el-query class='snTemplateNameQuery' @query="querydevicesSn" type="normal" :placeholder="queryPlaceholderName"></el-query>
				<div style='height: calc(100% - 73px); width: 320px; overflow-y: auto;' class="chartGroupTreeBox snChartsBox">
					<el-tree v-if='chartNetType == "gnb"'
						ref="groupDevicesTree"
						:data="devicesTreeData"
						node-key="uniqueId"
						:props= "{label:'group_name'}"
						:default-expanded-keys="devicesDefaultExpandedKeys"
						:default-checked-keys="devicesDefaultCheckedKeys"
						:highlight-current="true"
						:show-checkbox="true"
						check-strictly
						@check="devicesCheckChange">
						<div class="treeItemBoxCls" slot-scope="{ node,data }">
							<div v-if="data.children">
								<span class="ItemLabelCls commonGeneral12" :title="node.label">{{node.label}}</span>
							</div>
							<div v-else class="">
								<span class='snCommonItem commonGeneral12 snAllItemLabelCls' :title="node.label">{{node.label}}</span>
								<span class='snCommonItem commonTextNormal12 snAllItemLabelCls' v-if='data.cell_name' :title="data.cell_name"><%=rb.getString("GNBMingCheng")%>: {{data.cell_name}} </span>
								<span class='snCommonItem commonTextNormal12 snAllItemLabelCls' v-if='data.cellId' :title="data.cellId">nrCGI: {{data.cellId}}</span>
							</div>
						</div>
					</el-tree>
					<el-tree v-else 
						ref="groupDevicesTree"
						:data="devicesTreeData"
						node-key="small_cell_code"
						show-checkbox
						check-strictly
						:props= "{label:'group_name'}"
						:default-expanded-keys="devicesDefaultExpandedKeys"
						:default-checked-keys="devicesDefaultCheckedKeys"
						:highlight-current="true"
						@check="devicesCheckChange">
						<div class="treeItemBoxCls" slot-scope="{ node,data }">
							<div v-if="data.children">
								<span class="ItemLabelCls commonGeneral12" :title="node.label">{{node.label}}</span>
							</div>
							<div v-else class="commonFlex ItemLabelCls">
								<span class='snCommonItem commonGeneral12 snAllItemLabelCls' :title="node.label">{{node.label}}</span>
								<span class='snCommonItem commonTextNormal12 snAllItemLabelCls' v-if='data.cell_name' :title="data.cell_name"><%=rb.getString("eGWMingCheng")%>: {{data.cell_name}} </span>
							</div>
						</div>
					</el-tree>
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

        12.3.1.1:
        4g：smallcellcode-plmnId:  plmnId: 需要根据当前模板对应的等级来判断，如果是 plmn 级别，则取 plmnId
        2g： smallcellcode-bts_id

        4g or 2g 的下发参数 enbCodeList 逻辑梳理：
        1、初始化时： enbCodeList: [];
        2、首次调用 getChartCellsInfoData 接口-返回的 5 个设备的数据中-此时前端传递 enbCodeList: [] 正确；
            拼接 smallcellcode-plmnId 或者 smallcellcode-bts_id 放入 enbSmallCellCodeAndCellIdObj 中；
            也相当于 getDataForSelectedDivToChart 方法中，首次调用时，传递的 enbCodeList: [] 正确；
            3、设备勾选 devicesCheckChange 事件中：
            将选中的 smallcellcode-plmnId 或者 smallcellcode-bts_id 放入 enbSmallCellCodeAndCellIdObj 中；
            4、调用 getDataForSelectedDivToChart 方法中，传递的 enbCodeList:
                 取自 enbSmallCellCodeAndCellIdObj 中的数据，正确；
            5、图表查询接口 getKPIChartData  中，传递的 enbCodeList 正确；
            */
	var kpiChart = new Vue({
		el: '#chartContent_${randomValue}',
		data(){
			var vm = this;
			//当前时间 年-月-日 时-分-秒
			var endTime = formatDate(new Date(gloableTime));
			//图表的开始查询时间
			var chart_start_time = formatDate(new Date(endTime)).substring(0,10); //年-月-日
			var periodChangeTime = chart_start_time;//年-月-日

			return {
				endTime: endTime,
				chart_start_time: chart_start_time,
				periodChangeTime: periodChangeTime,
				enbCodeList: [],
				enbListName: [],
				gnbSmallCellCodeCellIdList: [], // gnb 的 ['smallCellCode - cellId'];
				gnbSmallCellCodeAndCellIdObj: [], //[{"smallCellCode":"xxx", "cellId":"xxx"}]
				gnbCheckedObj: [], //gnb 默认选中的设备对象
				gnbCodeList: [], //gnb 的 sn
				enbSmallCellCodeCellIdList: [], // enb 的 ['smallCellCode - plmnId']; 2g: ['smallCellCode - bts_id']
				enbCheckedObj: [], //enb 默认选中的设备对象
				enbCodeList: [], //enb 的 sn
				enbSmallCellCodeAndCellIdObj: [], //[{"smallCellCode":"xxx", "plmnId":"xxx" }] 2g: [{"smallCellCode":"xxx", "bts_id"}]
				cellNameList: [],
				kpiIdList: [],
				perf_list_all_chart: [],

				enbLimitNum: '5', //设备选择限制
				kpiLimitNum: '1', //指标选择限制
				chartPeriod: '0', //图表选择粒度  默认为天
				kpiChartStepTime: '',//当前显示时间点
				kpiChartStepDate: '',
				dayShow: true,
				queryPlaceholderName:'',
				devicesDefaultExpandedKeys:[],
				devicesDefaultCheckedKeys:[],
				kpidefaultExpandedKeys:[],
				querySNSearchText:'',
				queryKpiText: '',
				checkedSerialNumber:[],
				devicesTreeData:[],
				kpisTreeData: [],

				chartActiveName: 'device',
				groupIdList: [], //设备组的id参数 30,112,114
				groupNameList: [], //设备组名称
				deviceGroupUrl: '',
				//deviceGroupUrl: '${ctx}/pm/template/getTemplateAssociatedDeviceGroupList.action?isGnb=0',
				defaultCheckedGroupKeys: [],
                deviceGroupSelection: [],
				devicesCheckChangeTimer: null, // 设备选择防抖定时器,避免快速选中时多次调用接口
				deviceGroupChangeTimer: null, // 设备组选择防抖定时器,避免快速选中时多次调用接口
				resizeTimer: null, // window resize 防抖定时器
				requestSequence: 0, // 请求序列号,用于识别和丢弃过期的请求响应
				firstFlag: true,
				queryDeviceGroupParams: {
					searchText: ''
				},
				defaultFiveData: [], //默认选中前五条设备组数据
				levelType: kpiQueryVue.levelType 
			}
		},
		computed: {
			chartNetType() {
				return kpiQueryVue.currentNetworkType ? kpiQueryVue.currentNetworkType : sysMain.headType;
			},
			//根据当前粒度，置灰 后一天，周，月 circle-right 图标
			isCurrentTime() {
				var vm = this, bool = true,
					stepStr = '',
					curStr ='';
				//0-天 1-周 2-月
				if(vm.chartPeriod == '0'){
					stepStr = (vm.kpiChartStepTime||'').replace(/-/g,''),
					curStr = getYesterDay(0).replace(/-/g,'');
				}else if(vm.chartPeriod == '1'){
					//获取当前日期
					stepStr = vm.kpiChartStepTime.substring(0,10).replace(/-/g,'');
					//获取下一周的日期 curStr
					var nowTime = formatDate((new Date(vm.endTime))).substring(0,10);
					//时间选择框呈现时间
					var nowDivTime = vm.kpiChartStepTime;
					curStr = getWeekTime(nowTime).mondayTime.replace(/-/g,'');
				}else if(vm.chartPeriod == '2') {
					stepStr = vm.kpiChartStepTime.substring(0,7).replace(/-/g,'');
					curStr = getYesterDay(0).substring(0,7).replace(/-/g,'');
					//stepStr:上一个月 或下一个月的日期 curStr:当前日期
				}
				if(stepStr-curStr<0) bool = false;

				return bool;
			},
			kpiChartStepDateType() {
				var vm = this,
					type = 'date';

				if(vm.chartPeriod == '2') {
					type = 'month';
				}

				return type;
			},

			kpiChartStepOptions() {
				var vm = this;

				return {
					disabledDate: function(time) {
						var now = new Date(getYesterDay(0) + ' 00:00:00'),
							nowTime = getWeekTime(vm.chart_start_time),
							endTime = new Date(nowTime.sundayTime + ' 00:00:00'); //周-结束日期

						if(vm.chartPeriod == '1'){
							var maxTime = endTime.getTime();

							return time.getTime() > maxTime
						}else{
							return time.getTime() > now.getTime();
						}
					}
				}
			},
		},
		watch: {
			deviceGroupSelection(){
				var vm = this;

				vm.groupNameList = [];
				// 使用防抖,避免快速选中设备组时多次调用接口导致图表数据累积
				if(vm.deviceGroupChangeTimer) {
					clearTimeout(vm.deviceGroupChangeTimer);
				}
				vm.deviceGroupChangeTimer = setTimeout(function(){
					vm.$nextTick(function(){
						vm.groupIdList = vm.deviceGroupSelection.map((item) => { 
							//图表展示头部标题： group_name
							vm.groupNameList.push(item.group_name);

							return item.group_id 
						});
						
						vm.getDataForSelectedDivToChart();
						vm.getKPIChartData(vm.periodChangeTime, vm.chartPeriod);
					})
				}, 300); // 延迟300ms执行,如果300ms内再次勾选则重新计时
			},
			//默认选中前五条设备组数据
			//defaultCheckedGroupKeys 有数据后才会触发表格的 selection-change 事件
			defaultFiveData(){
				var vm = this;
				
				if(vm.defaultFiveData.length > 0){
					vm.defaultFiveData.map((item) => {
						//将item.group_id 都存放到 groupIdList 中
						vm.defaultCheckedGroupKeys.push(item.group_id);
						// 将item.group_id 都存放到 groupIdList 中
						vm.groupIdList.push(item.group_id);
						//图表展示头部标题： group_name
						vm.groupNameList.push(item.group_name);
					})
				}
			}
		},
		methods: {
			init(){
				var vm = this;
				//初始化时间
				vm.kpiChartStepTime = vm.chart_start_time;
				vm.kpiChartStepDate = vm.chart_start_time;

				//设备筛选 针对网元对应的提示
				if(vm.chartNetType == 'enb'){
					var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
					//selDeviceType: 1-设备组 2-设备
					vm.chartActiveName = curForm.selDeviceType == '1' ? 'deviceGroup' : 'device';
					//if(vm.chartActiveName == 'deviceGroup'){
						vm.deviceGroupUrl = "${ctx}/pm/template/getTemplateAssociatedDeviceGroupList.action?isGnb=0"+"&tempId="+curForm.tempId
						//vm.deviceGroupUrl = "${ctx}/pm/template/getTempDeviceGroupList.action?isGnb=0"+"&tempId="+curForm.tempId
					//}
					if(northOperatorScenario == 'S0009'){			
						vm.queryPlaceholderName = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>/<%=rb.getString("ZhanZhiMingCheng")%>';
					}else{
						vm.queryPlaceholderName = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>';						
					}
				}else if(vm.chartNetType == 'gnb'){
					vm.queryPlaceholderName = '<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%> / NrCGI';
				}else if(vm.chartNetType == 'egw'){
					vm.queryPlaceholderName = '<%=rb.getString("eGWBianMa")%>/<%=rb.getString("eGWMingCheng")%>';
				}

				//获取指标
				vm.queryKpi(vm.queryKpiText);
				//初始化设备组与设备的数据
				vm.querydevicesSn(vm.querySNSearchText);

				//默认获取前1个指标，前5个设备的图表数据   初始化chart图表面板
				vm.getDataForSelectedDivToChart();
				//获取模板详情，设置对应的图表周期 如果模板上报周期为"24h"，则取消"天"周期,并获取图表数据
				vm.getKPITempInfoForChart();
				$(window).resize();
			},

			//tab 切换                                                                     
			chartTabClick(tab){
				var vm = this;	

				vm.getDataForSelectedDivToChart();
				//获取模板详情，设置对应的图表周期 如果模板上报周期为"24h"，则取消"天"周期,并获取图表数据(getChartDataList)
				vm.getKPIChartData(vm.periodChangeTime, vm.chartPeriod);
			},
			//设备组查询
			querydeviceGroup(val){
				var vm = this;
				
				vm.queryDeviceGroupParams.searchText = val;
			},
			//设备列表查询
			querydevicesSn(val){
				var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
				var vm = this,
					params = {},
					getDeviceListUrl = '';

					params.searchText = val;
					params.tempId = curForm.tempId;
					//params.isGnb = 0;

				if(vm.chartNetType == 'enb'){
					getDeviceListUrl = '${ctx}/pm/template/getTempDeviceList.action?isGnb=0';
				}else if(vm.chartNetType == 'gnb'){
					getDeviceListUrl = '${ctx}/pm/template/getTempDeviceList.action?isGnb=1';
					//params.isGnb = 1;
				}else if(vm.chartNetType == 'egw'){
					getDeviceListUrl = '${ctx}/egw/pm/template/getEgwListData.action';
				}
				axios.post(getDeviceListUrl, stringify(params)).then(function(response){
					var data = response.data;
					if(data && data.rows.length>0){
						vm.devicesTreeData = data.rows;
					}else{
						vm.devicesTreeData = [];
					}
				})
			},
			//kpi列表查询
			queryKpi(val){
				var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
				var vm = this,
					params = {},
					getKpiListUrl = '';

					params.searchText = val;
					params.tempId = curForm.tempId;

				if(vm.chartNetType == 'enb'){
					getKpiListUrl = '${ctx}/pm/indicatormg/getIndicatorListData.action'
				}else if(vm.chartNetType == 'gnb'){
					getKpiListUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorListData.action';
				}else if(vm.chartNetType == 'egw'){
					getKpiListUrl = '${ctx}/egw/pm/indicatormg/getIndicatorListData.action';
				}
				axios.post(getKpiListUrl, stringify(params)).then(function(response){
					var data = response.data;
					if(data && data.rows.length>0){
						vm.kpisTreeData = data.rows;

						//指标当前被选中的节点
						vm.$nextTick(function(){
							vm.$refs.groupKpiTree.setCurrentKey(vm.kpiIdList[0]); //默认选中的节点
							vm.kpidefaultExpandedKeys.push( vm.$refs.groupKpiTree.getNode(vm.kpiIdList[0]).parent);
						});

					}else{
						vm.kpisTreeData = [];
					}
				})
			},

			//获取模板详情，设置对应的图表周期 如果模板上报周期为"24h"，则取消"天"周期,并获取图表数据
			getKPITempInfoForChart(){
				var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
				var vm = this, curTemplateInfoUrl = '',
					params = {
						tempId: curForm.tempId,
						timeZone: timeZone
					};

				if(vm.chartNetType == 'enb'){
					curTemplateInfoUrl = '${ctx}/pm/template/getTemplateInfo.action';
				}else if(vm.chartNetType == 'gnb'){
					curTemplateInfoUrl = '${ctx}/gnb/pm/template/getTemplateInfo.action';
				}else if(vm.chartNetType == 'egw'){
					curTemplateInfoUrl = '${ctx}/egw/pm/template/getTemplateInfo.action';
				}

				axios.post(curTemplateInfoUrl, stringify(params)).then(function(response){
					var data = response.data
					if(data){
						//当前表格粒度与data.reportPeriod 相同，则将天粒度隐藏，默认选中周粒度
						reportCycle = data.reportPeriod;
						
						if(reportCycle == "1440"){
							vm.dayShow = false; //隐藏天粒度
							vm.chartPeriod = '1'; //默认选中周
							//触发周事件
							if(vm.chartNetType == 'enb' && vm.chartActiveName == 'deviceGroup'){

							}else{
								vm.changeChartPeriod(vm.chartPeriod);
							}
						}else{
							//默认获取前1个指标，前5个设备的图表数据 -- 默认为天周期
							if(vm.chartNetType == 'enb' && vm.chartActiveName == 'deviceGroup'){

							}else{
								vm.getKPIChartData(vm.periodChangeTime, vm.chartPeriod);
							}
						}
					}
				})
			},
			//日期组件中选择时间
			kpiChartStepDateChange(val) {
				var vm = this, str = '';
				if(vm.chartPeriod == '1'){
					//周
					var curTime = dateformatter(val).substring(0,10);
					var nowTime = getWeekTime(curTime);
					vm.kpiChartStepTime = nowTime.mondayTime + " - "+ nowTime.sundayTime;
					str = nowTime.mondayTime + ' 00:00:00';
					vm.kpiChartStepDate = nowTime.mondayTime;
					vm.periodChangeTime = str;
				}else if(vm.chartPeriod == '2') {
					//月
					var curTime = dateformatter(val).substring(0,10);
					str = curTime.substring(0,7); //2023-09
					vm.kpiChartStepTime = str;
					vm.kpiChartStepDate = str;
					vm.periodChangeTime = str;
				}else{
					//天
					str = dateformatter(val).substring(0,10);
					vm.kpiChartStepTime = str;
					vm.kpiChartStepDate = str;
					vm.periodChangeTime = str;
				}
				vm.getKPIChartData(str,vm.chartPeriod);
			},

			// 初始化前一天，周，月 或后一天，周，月 时间切换
			kpiChartStepTimeChange(num) {
				var vm = this,
					type = vm.chartPeriod, //0-天  1-周  2-月
					kpiChartStepTime = vm.kpiChartStepTime; //当前时间点

				if(num>0 && vm.isCurrentTime) {
					return;
				}
				//后一天
				if(num == 1){
					//当前时间
					var nowTime = formatDate((new Date(vm.endTime))).substring(0,10);
					//时间选择框呈现时间
					var nowDivTime = vm.kpiChartStepTime;
					//选择时间和周期之后的开始时间
					var nextTime = "";
					if(type == '1'){//周
						nowTime = getWeekTime(nowTime).mondayTime;
						nowDivTime = nowDivTime.split(" - ")[0] + ' 00:00:00';
						nextTime = formatDate(addDate(new Date(nowDivTime),7));
						showNowTime = getWeekTime(nextTime);
						vm.kpiChartStepTime = showNowTime.mondayTime + " - "+ showNowTime.sundayTime;
						vm.kpiChartStepDate = showNowTime.mondayTime;
						vm.periodChangeTime = showNowTime.mondayTime + ' 00:00:00';
					}else if(type == '2'){//月
						nowDivTime += '-01 00:00:00';
						nowTime = nowTime.substring(0,7);
						nextTime = new Date(nowDivTime);
						nextTime = addMonth(nextTime,1);
						vm.kpiChartStepTime = nextTime;
						vm.kpiChartStepDate = nextTime;
						vm.periodChangeTime = nextTime;
					}else{//天
						nowDivTime += ' 00:00:00';
						nextTime = formatDate(addDate(new Date(nowDivTime),1)).substring(0,10);
						vm.kpiChartStepTime = nextTime;
						vm.kpiChartStepDate = nextTime;
						vm.periodChangeTime = nextTime;
					}
					 //时间选择赋值
					vm.getKPIChartData(nextTime, type);
				}else {
					//天，周，月 前一天点击
					var nowDivTime = vm.kpiChartStepTime,
						prevTime = '',
						showNowTime;
						//周
					if(type == '1') {
						nowDivTime = nowDivTime.split(" - ")[0] + ' 00:00:00';
						prevTime = formatDate(addDate(new Date(nowDivTime),-7));
						showNowTime = getWeekTime(prevTime);
						vm.kpiChartStepTime = showNowTime.mondayTime + " - "+ showNowTime.sundayTime;
						vm.kpiChartStepDate = showNowTime.mondayTime;
						vm.periodChangeTime = showNowTime.mondayTime + ' 00:00:00';
					}else if(type == '2') {
						//月
						nowDivTime += '-01 00:00:00',
						prevTime = new Date(nowDivTime),
						prevTime = addMonth(prevTime,-1);
						vm.kpiChartStepTime = prevTime;
						vm.kpiChartStepDate = prevTime;
						vm.periodChangeTime = prevTime;
					}else{
						//天
						nowDivTime += ' 00:00:00';
						prevTime = formatDate(addDate(new Date(nowDivTime),-1)).substring(0,10);
						vm.kpiChartStepTime = prevTime;
						vm.kpiChartStepDate = prevTime;
						vm.periodChangeTime = prevTime;
					}

					vm.getKPIChartData(prevTime,type);

				}
			},

			// 天、周，月切换
			changeChartPeriod(period) {
				var vm = this;
				vm.chartPeriod = period; //当前选中粒度
				//周
				if(period == '1'){
					var nowTime = getWeekTime(vm.chart_start_time);
					vm.kpiChartStepTime = nowTime.mondayTime + " - "+ nowTime.sundayTime; //周一-周日  2023-09-04--2023-09-10
					vm.periodChangeTime = nowTime.mondayTime + ' 00:00:00';
					vm.kpiChartStepDate = nowTime.mondayTime;
				}else if(period == '2'){
					 //月
					 vm.kpiChartStepTime = vm.chart_start_time.substring(0,7);
					 vm.kpiChartStepDate = vm.chart_start_time.substring(0,7);
					 vm.periodChangeTime = vm.chart_start_time.substring(0,7);
				}else{
					//天
					vm.kpiChartStepTime = formatDate(new Date(vm.endTime)).substring(0,10);
					vm.kpiChartStepDate = formatDate(new Date(vm.endTime)).substring(0,10);
					vm.periodChangeTime = formatDate(new Date(vm.endTime)).substring(0,10);
				}
				 //时间选择组件赋值
				//periodChangeTime: 年-月-日；period：0-天，1-周，2-月
				vm.getKPIChartData(vm.periodChangeTime, period);
			},
			//设备勾选
			devicesCheckChange(data, checkedData){
				var vm = this;
				if(vm.chartNetType == "gnb"){
					// 记录历史选择 start -->
					if(checkedData.checkedKeys.includes(data.uniqueId)) {
						if(vm.gnbSmallCellCodeAndCellIdObj.length >= vm.enbLimitNum) {
							vm.$message.error('<%=rb.getString("ZuiDuoXuanZe5")%>');
							vm.$refs.groupDevicesTree.setChecked(data.uniqueId, false)				
							return;
						}

						if(data.cellId){
							vm.gnbSmallCellCodeAndCellIdObj.push({"smallCellCode":data.small_cell_code,"cellId":data.cellId});
						}else{
							vm.gnbSmallCellCodeAndCellIdObj.push({"smallCellCode":data.small_cell_code});
						}
					}else {
						var dataIndex = -1;
						vm.gnbSmallCellCodeAndCellIdObj.map(function(item, index) {
							if(data.cellId) {
								if(item.smallCellCode == data.small_cell_code && item.cellId == data.cellId) dataIndex = index;
							}else {
								if(item.smallCellCode == data.small_cell_code) dataIndex = index;
							}
						});

						if(dataIndex >= 0) {
							vm.gnbSmallCellCodeAndCellIdObj.splice(dataIndex,1);
						}
					}
					var curSelectNodes = vm.gnbSmallCellCodeAndCellIdObj;
					// <-- 记录历史选择 end

					if(curSelectNodes.length == 0){
						vm.$message.error('<%=rb.getString("QingXuanZeSheBei")%>');
						return;
					}else if(curSelectNodes.length > vm.enbLimitNum){
						vm.$message.error('<%=rb.getString("ZuiDuoXuanZe5")%>');
						vm.$refs.groupDevicesTree.setChecked(data.uniqueId, false)				
						return;
					} else{
						// 使用防抖,避免快速选中设备时多次调用接口导致图表数据累积
						if(vm.devicesCheckChangeTimer) {
							clearTimeout(vm.devicesCheckChangeTimer);
						}
						vm.devicesCheckChangeTimer = setTimeout(function(){
							vm.getDataForSelectedDivToChart();
							vm.getKPIChartData(vm.periodChangeTime, vm.chartPeriod);
						}, 300); // 延迟300ms执行,如果300ms内再次勾选则重新计时
					}
				}else if(vm.chartNetType == "enb"){
					// 记录历史选择 start -->
					if(checkedData.checkedKeys.includes(data.uniqueId)) {
						if(vm.enbSmallCellCodeAndCellIdObj.length >= vm.enbLimitNum) {
							vm.$message.error('<%=rb.getString("ZuiDuoXuanZe5")%>');
							vm.$refs.groupDevicesTree.setChecked(data.uniqueId, false)				
							return;
						}

						var itemObj = {};
						if(data.small_cell_code){
							itemObj.smallCellCode = data.small_cell_code;
						}
						//4g:smallCellCode-plmnId
						/*if(data.cellId){
							itemObj.cellId = data.cellId;
						}*/
						if(data.plmnId && vm.levelType == 'plmn'){
							itemObj.plmnId = data.plmnId;
						}
						//2g:smallCellCode-bts_id
						if(data.bts_id){
							itemObj.bts_id = data.bts_id;
						}
						vm.enbSmallCellCodeAndCellIdObj.push(itemObj);

					}else {
						var dataIndex = -1;
						vm.enbSmallCellCodeAndCellIdObj.map(function(item, index) {
							if(item.smallCellCode == data.small_cell_code) dataIndex = index;
						});

						if(dataIndex >= 0) {
							vm.enbSmallCellCodeAndCellIdObj.splice(dataIndex,1);
						}
					}
					var curSelectNodes = vm.enbSmallCellCodeAndCellIdObj;
					// <-- 记录历史选择 end

					if(curSelectNodes.length == 0){
						vm.$message.error('<%=rb.getString("QingXuanZeSheBei")%>');
						return;
					}else if(curSelectNodes.length > vm.enbLimitNum){
						vm.$message.error('<%=rb.getString("ZuiDuoXuanZe5")%>');
						vm.$refs.groupDevicesTree.setChecked(data.uniqueId, false)				
						return;
				} else{
					// 使用防抖,避免快速选中设备时多次调用接口导致图表tip累积
					if(vm.devicesCheckChangeTimer) {
						clearTimeout(vm.devicesCheckChangeTimer);
					}
					vm.devicesCheckChangeTimer = setTimeout(function(){
						vm.getDataForSelectedDivToChart();
						vm.getKPIChartData(vm.periodChangeTime, vm.chartPeriod);
					}, 300); // 延迟300ms执行,如果300ms内再次勾选则重新计时
				}				}else{
					if(checkedData.checkedKeys.includes(data.small_cell_code)) {
						if(vm.enbCodeList.length >= vm.enbLimitNum) {
							vm.$message.error('<%=rb.getString("ZuiDuoXuanZe5")%>');
							vm.$refs.groupDevicesTree.setChecked(data.small_cell_code, false)					
							return;
						}

						if(!vm.enbCodeList.includes(data.small_cell_code)) vm.enbCodeList.push(data.small_cell_code);
					}else {
						vm.enbCodeList.remove(data.small_cell_code);
					}

					if(vm.enbCodeList.length == 0){
						vm.$message.error('<%=rb.getString("QingXuanZeSheBei")%>');
						return;
					}else if(vm.enbCodeList.length > vm.enbLimitNum){
						vm.$message.error('<%=rb.getString("ZuiDuoXuanZe5")%>');
						vm.$refs.groupDevicesTree.setChecked(data.small_cell_code, false)					
						return;
					} else{
						// 使用防抖,避免快速选中设备时多次调用接口导致图表数据累积
						if(vm.devicesCheckChangeTimer) {
							clearTimeout(vm.devicesCheckChangeTimer);
						}
						vm.devicesCheckChangeTimer = setTimeout(function(){
							vm.getDataForSelectedDivToChart();
							vm.getKPIChartData(vm.periodChangeTime, vm.chartPeriod);
						}, 300); // 延迟300ms执行,如果300ms内再次勾选则重新计时
					}
				}
			},
			
			// 设备组列表-当选择项发生变化时触发此事件
			deviceGroupChange(selection) {
				var vm = this;

				if(selection){
					vm.deviceGroupSelection = selection;
				}
			},
			//---------------------------------------------------------------------kpi
			kpisNodeChange(data,node,ev){
				var vm = this;

				if(!data.children){
					vm.kpiIdList = data.group_name.split(',');
					if(data.group_name == '' || data.group_name == null || data.group_name == undefined){
						vm.$message.error('<%=rb.getString("QingXuanZeZhiBiao")%>');
						return;
					} else{
                        if(vm.chartNetType == 'enb' && vm.chartActiveName == 'deviceGroup'){
                            //判断是否选择了设备组
                            
                            // 出现设备组参数是空字符串，调用接口也无实际意义；如果没有选择设备组,则默认选中前五个
                            if(!vm.groupIdList || vm.groupIdList.length === 0){                                
                                if(vm.defaultFiveData && vm.defaultFiveData.length > 0){
                                    // 清空旧数据
                                    vm.defaultCheckedGroupKeys = [];
                                    vm.groupIdList = [];
                                    vm.groupNameList = [];
                                    
                                    vm.defaultFiveData.map((item) => {
                                        //将item.group_id 都存放到 groupIdList 中
                                        vm.defaultCheckedGroupKeys.push(item.group_id);
                                        // 将item.group_id 都存放到 groupIdList 中
                                        vm.groupIdList.push(item.group_id);
                                        //图表展示头部标题： group_name
                                        vm.groupNameList.push(item.group_name);
                                    })
                                } 
                            }
                        }
						vm.getDataForSelectedDivToChart();
						vm.getKPIChartData(vm.periodChangeTime, vm.chartPeriod);
					}
				}
			},

			/**
			* 获取图表数据  kpiIdList
			* @param startTimeForParam[string] 开始时间
			* @param range[string] 时间粒度
			**/
			getKPIChartData(startTimeForParam, range){
				var vm = this, params = {}, curChartListUrl = '';
				
				// 递增请求序列号,用于识别和丢弃过期的请求响应
				vm.requestSequence++;
				var currentRequestId = vm.requestSequence;
				
				// 统一清除防抖定时器，避免定时器在新查询期间触发
				// 无论是用户主动操作（时间切换、周期切换）还是定时器触发，都会调用此方法
				// 在此统一清除可以避免在每个调用点都添加清除逻辑
				// 注意: devicesCheckChangeTimer 只在非防抖触发的情况下清除(由防抖触发时已经自然失效)
				if(vm.deviceGroupChangeTimer) {
					clearTimeout(vm.deviceGroupChangeTimer);
					vm.deviceGroupChangeTimer = null;
				}
				
				var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
				//设备组
				if(vm.chartNetType == 'enb' && vm.chartActiveName == 'deviceGroup'){
					params.tempId = curForm.tempId;
					params.timeZone = timeZone;
					params.timeRange = range;
					params.startTime = startTimeForParam;
					params.kpiIdList = vm.kpiIdList.join(",");
					params.groupIdList = vm.groupIdList.join(",");
					
				}else{
					params.tempId = curForm.tempId;
					params.timeZone = timeZone;
					params.timeRange = range;
					params.startTime = startTimeForParam;
					params.enbLimitNum =  vm.enbLimitNum;
					params.kpiLimitNum =  vm.kpiLimitNum;
					params.kpiIdList = vm.kpiIdList.join(",");
				}
				
				//特殊处理gnb
				if(vm.chartNetType == 'gnb'){
					curChartListUrl = '${ctx}/gnb/pm/template/getChartDataList.action';
					//已选中数据
					if(vm.gnbCheckedObj.length > 0){
						// params.enbCodeList= [{"smallCellCode":"48BF74_120200054322AVB0005", "cellId":"1234"}}]
						var gnbParamsList = [];
						vm.gnbCheckedObj.map(function(item){
							if(item.cellId && item.smallCellCode){
								var itemObj = {};

								if(item.cellId){
									itemObj.smallCellCode = item.smallCellCode;
									itemObj.cellId = item.cellId;
									gnbParamsList.push(itemObj);											
								}
							}else {
								gnbParamsList.push({ "smallCellCode": item.smallCellCode});
							}
						})
						params.enbCodeList = JSON.stringify(gnbParamsList);
						vm.gnbCodeList = JSON.stringify(gnbParamsList);
					}else{
						params.enbCodeList = '';
						//vm.gnbCodeList = [];
					}
				}else if(vm.chartNetType == 'enb'){
					if(vm.chartActiveName == 'deviceGroup'){
						curChartListUrl = '${ctx}/pm/template/getChartGroupDataList.action';
					}else{
						//params.enbCodeList= [{"smallCellCode":"48BF74_120200054322AVB0005", "plmnId":"311480"}] 2g: [{"smallCellCode":"xxx","bts_id":"xxx",}] 
						curChartListUrl = '${ctx}/pm/template/getChartDataList.action';

						if(vm.enbCheckedObj.length > 0){
							var enbParamsList = [];

							vm.enbCheckedObj.map(function(item){
								var itemObj = {};
		
								if(item.smallCellCode){
									itemObj.smallCellCode = item.smallCellCode;
								}
								/*if(item.cellId){
									itemObj.cellId = item.cellId;
								}*/
								if(item.plmnId && vm.levelType == 'plmn'){
									itemObj.plmnId = item.plmnId;
								}
								//2g smallCellCode - bts_id
								if(item.bts_id){
									itemObj.bts_id = item.bts_id;
								}
								enbParamsList.push(itemObj);
							})

							params.enbCodeList = JSON.stringify(enbParamsList);
							vm.enbCodeList = JSON.stringify(enbParamsList);
						}else{
							params.enbCodeList = '';
						}
					}
				}else{
					curChartListUrl = '${ctx}/egw/pm/template/getChartDataList.action';
					if(vm.enbCodeList.length > 0){
						params.enbCodeList = vm.enbCodeList.join(",");
					}else{
						params.enbCodeList = '';
					}
				}
				if($("#winChartCondition").is(":visible")){
					$.messager.progress({
						title : '<%=rb.getString("QingDengDai")%>',
						text : '<%=rb.getString("JieXiZhong")%>'
					});
				}
				/*if(vm.chartNetType == 'enb' && vm.chartActiveName == 'deviceGroup'){
					$.post(curChartListUrl,params,function(data){
						if($("#winChartCondition").is(":visible")){
							$.messager.progress("close");
						}
						for (var perf_index = 0; perf_index < vm.perf_list_all_chart.length; perf_index++) {
							var perf_obj = vm.perf_list_all_chart[perf_index];
							vm.goSetChartData(vm.groupIdList, vm.groupIdList, perf_obj, data, vm.periodChangeTime, range, vm.groupNameList);
						}
					},"json");
				}else{*/
					$.post(curChartListUrl,params,function(data){
						// 检查请求序列号,丢弃过期的响应,防止旧请求的数据污染新选择
						if(vm.requestSequence && vm.requestSequence !== currentRequestId) {
							return; // 丢弃过期响应
						}

						if($("#winChartCondition").is(":visible")){
							 $.messager.progress("close");
						}
						//将5g 与其它网元分开
						if(vm.chartNetType == 'gnb'){
							for (var perf_index = 0; perf_index < vm.perf_list_all_chart.length; perf_index++) {
								var perf_obj = vm.perf_list_all_chart[perf_index];
								vm.goSetChartData(vm.gnbSmallCellCodeCellIdList, vm.enbListName, perf_obj, data, vm.periodChangeTime, range, vm.cellNameList);
							}
						}else if(vm.chartNetType == 'enb'){
							if(vm.chartActiveName == 'deviceGroup'){
								for (var perf_index = 0; perf_index < vm.perf_list_all_chart.length; perf_index++) {
									var perf_obj = vm.perf_list_all_chart[perf_index];
									vm.goSetChartData(vm.groupIdList, vm.groupIdList, perf_obj, data, vm.periodChangeTime, range, vm.groupNameList);
								}
							}else{
								for (var perf_index = 0; perf_index < vm.perf_list_all_chart.length; perf_index++) {
									var perf_obj = vm.perf_list_all_chart[perf_index];
									vm.goSetChartData(vm.enbSmallCellCodeCellIdList, vm.enbListName, perf_obj, data, vm.periodChangeTime, range, vm.cellNameList);
								}
							}
						}else{
							for (var perf_index = 0; perf_index < vm.perf_list_all_chart.length; perf_index++) {
								var perf_obj = vm.perf_list_all_chart[perf_index];
								vm.goSetChartData(vm.enbCodeList, vm.enbListName, perf_obj, data, vm.periodChangeTime, range, vm.cellNameList);
							}
						}

					},"json");
				//}
				
				
			},

			//根据查询的KPI指标数据 绘制显示图表
			goSetChartData(codes, show_names, perf_obj, chart_data, periodChangeTime, range, cellNameList){
				var vm = this;
				if (cellNameList.length == 0) {
					cellNameList = codes;
				}
				//初始化节点数据
				var timesDataArr = [];
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

				    var perf_code = perf_obj.kpiId;
				    var series_data = [];

					for(var code_index = 0; code_index < codes.length; code_index++) {
					    var seriesEle_data = [];
					    //将所有的点先置为'-'
					    for(var i= 0; i<timesDataArr.length; i++ ){
						    seriesEle_data.push('-');
					    }
					    if (chart_data.length > 0) {
						    var code_value = codes[code_index];
							for (var time_index = 0; time_index < timesDataArr.length; time_index++) {
								for (var data_index = 0; data_index < chart_data.length; data_index++) {
									var perf_data_obj = chart_data[data_index];
									var time = perf_data_obj.startTime;
									var eNodeB_code = '';
									//gnb 应该按照 uniqueId 去匹配  //gnb 是否应该按照uniqueId 进行传值
									if(vm.chartNetType == 'gnb'){
										eNodeB_code = perf_data_obj.uniqueId;
									}else if(vm.chartNetType == 'enb'){
										if(vm.chartActiveName == 'deviceGroup'){
											eNodeB_code = perf_data_obj.group_id;
										}else{
											eNodeB_code = perf_data_obj.uniqueId;   
											//看后端接口返回的数据是否有 uniqueId, 如果没有则使用smallCellCode
											/*if(!eNodeB_code){
												eNodeB_code = perf_data_obj.smallCellCode;
											}*/
										}
									}else{
										eNodeB_code = perf_data_obj.smallCellCode;
									}
								
									if(code_value == eNodeB_code){
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

						var seriesEle = {
							name : cellNameList[code_index],
							type : 'line',
							symbol: 'circle',
							symbolSize : range=='0'?4:10,
							showAllSymbol : true,
							data : seriesEle_data
						};
						series_data.push(seriesEle);
					}

					var chart_id = "kpi_chart_" + perf_obj.kpiId + "_${randomValue}";
					var name = perf_obj.kpiName;
					var unit = perf_obj.unit;
					var title_obj = {
						code : perf_obj.kpiId,
						name : name,
						unit : unit,
					};

					var legend_obj = {
						code : codes,
						name : cellNameList
					};

					var chart_data = {
						title_data : title_obj,
						legend_data : legend_obj,
						x_data : timesDataArr,
						series_data : series_data
					};

					try{                        
						vm.goCreateChart(chart_id, chart_data, range);
					}catch(e){}
			    }
			},
			/**
			* 创建图表
			* @param elementId[string] 图表id
			* @param chart_data[object] 图表数据
			* @param range{string}: 时间范围
			**/
			goCreateChart(elementId, chart_data, range) {
				var vm = this;
				var chartDom = document.getElementById(elementId);
				
				// 检查DOM是否存在,防止在组件销毁或切换时出错
				if(!chartDom) {
					return;
				}
				
				// 先销毁已存在的图表实例，避免数据累积
				var existingInstance = echarts.getInstanceByDom(chartDom);
				if(existingInstance) {
					existingInstance.dispose();
				}
				
				// 重新初始化图表
				var kpiChart = echarts.init(chartDom);
				var option = {
					title : {
						codedata : chart_data.title_data.code,
						x : 'left',
						y : -10,
						subtextStyle : {
							fontSize : 12,
							color : '#CAC5D4',
						}
					},

					color : ['#85b1de','#69e7e7','#97dbb9','#e9a4a4','#bf8bf3'],

					tooltip : {
						trigger : 'axis',
						formatter : function(val) {
                            
							if(!val || !val[0] || !val[0].name) return '';

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

					grid : {
						top : '120px',
						bottom : '30px',
						left : '30px',
						right : '60px',
						containLabel: true
					},

					legend : {
					data : (function(){
						// 动态生成 legend，只为实际存在的设备创建图例
						var colors = ['#85b1de','#69e7e7','#97dbb9','#e9a4a4','#bf8bf3'];
						var legendData = [];
						
						// 边界情况处理：检查数据是否存在
						if(!chart_data.legend_data || (!chart_data.legend_data.name && !chart_data.legend_data.code)){
							return legendData; // 返回空数组
						}
						
						var names = chart_data.legend_data.name || [];
						var codes = chart_data.legend_data.code || [];
						var length = Math.max(names.length, codes.length);
						
						for(var i = 0; i < length; i++){
							var legendName = names[i] || codes[i];
							// 只为有效的设备名称添加图例
							if(legendName){
								legendData.push({
									name: legendName,
									textStyle: {
										color: colors[i % colors.length]
									}
								});
							}
						}
						return legendData;
					})(),
					left: '70px',
					orient: 'horizontal',
					textStyle: {
						fontSize: 12
					},
					icon:"circle",
					itemHeight: 6,
					itemWidth: 6,
					itemGap: 10
					},

					xAxis : [{
								type : 'category',
								boundaryGap : false,
								data : chart_data.x_data,
								axisLabel : {
									formatter : function(val) {
										if(range == "0"){
											var secondTime = val.split(' ')[1];
												clock = secondTime.substring(0,2);
												val = secondTime.substring(0,5);
											if(secondTime.substring(3,5)=='00') return clock;
											return val;
										}else{
											var secondTime = val.split(' ')[0];
											if(range != "2"){
												if (val.substring(8,10) == "01") return val.substring(5, 10).replace('-','.');
											}
											val = secondTime.substring(8,10);
											return val;
										}
									},

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

								name : range == "0"?'<%=rb.getString("XiaoShi")%>': '<%=rb.getString("Tian")%>',
						}],

					yAxis : [{
							name: '<%=rb.getString("ZuoKuoHao")%>' + chart_data.title_data.unit + '<%=rb.getString("YouKuoHao")%>',
								type : 'value',
								axisLabel : {
									formatter: function (value) {
										// 将数字转换为普通数字格式
										return value.toLocaleString(); 
									}
									//formatter : '{value}'
								},
							}],

					series : chart_data.series_data
				};

				kpiChart.setOption(option);

				// 使用 echarts 自带的 resize，而不是添加 window resize 监听器
				// echarts 的 dispose 会自动清理相关事件监听器
				// 注意：如果需要响应窗口大小变化，应该在外层统一管理，而不是每个图表都添加监听器
			},
			
			//设备组列表-加载成功  默认获取前1个指标，前5个设备组的图表数据
			loadSuccessDeviceGroup(data){
				var vm = this;

				if(data && data.length > 0){
					vm.defaultFiveData = data.slice(0,5);
				}	
			},
			//默认获取前1个指标，前5个设备的图表数据   初始化chart图表面板
			getDataForSelectedDivToChart(){
				var vm = this,  
					curCodeList = '',
					curGetChartCellsInfoUrl = '', 
					curGetChartIndicatorsInfoUrl = '', 
					kpi_chart_maindiv = $('#chart_main_${randomValue}');

				var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
				
				vm.perf_list_all_chart = [];
				vm.enbListName = [];
				vm.cellNameList = [];
				vm.gnbCheckedObj = [];
				vm.enbCheckedObj = [];
				kpi_chart_maindiv.html("");

				if(vm.chartNetType == 'enb'){
					curGetChartCellsInfoUrl = '${ctx}/pm/template/getChartCellsInfoData.action';
					curGetChartIndicatorsInfoUrl = '${ctx}/pm/template/getChartIndicatorsInfoData.action';
					//复选框发生变化时,该参数为已选中设备的集合;  数组对象 [{"smallCellCode":"xxx",  plmnId":"xxx",}] 2g: smallCellCode[{"smallCellCode":"xxx", "bts_id":"",}]

					if(vm.enbSmallCellCodeAndCellIdObj && vm.enbSmallCellCodeAndCellIdObj.length > 0){
						curCodeList = JSON.stringify(vm.enbSmallCellCodeAndCellIdObj);
					}else{
						if(vm.enbCodeList && vm.enbCodeList.length > 0){
							curCodeList = vm.enbCodeList;
						}else{
							curCodeList = '';
						}
					}
					vm.enbCodeList = curCodeList;

				}else if(vm.chartNetType == 'gnb'){
					curGetChartCellsInfoUrl = '${ctx}/gnb/pm/template/getChartCellsInfoData.action';
					curGetChartIndicatorsInfoUrl = '${ctx}/gnb/pm/template/getChartIndicatorsInfoData.action';
					//首次进入页面,该参数为空字符串
					//复选框发生变化时,该参数为已选中设备的集合;  数组对象 [{"smallCellCode":"xxx", "cellId":"xxx"}]
					if(vm.gnbSmallCellCodeAndCellIdObj && vm.gnbSmallCellCodeAndCellIdObj.length > 0){
						curCodeList = JSON.stringify(vm.gnbSmallCellCodeAndCellIdObj);
					}else{
						if(vm.gnbCodeList && vm.gnbCodeList.length > 0){
							curCodeList = vm.gnbCodeList;
						}else{
							curCodeList = '';
						}
					}
					vm.gnbCodeList = curCodeList;
				}else if(vm.chartNetType == 'egw'){
					curGetChartCellsInfoUrl = '${ctx}/egw/pm/template/getChartCellsInfoData.action';
					curGetChartIndicatorsInfoUrl = '${ctx}/egw/pm/template/getChartIndicatorsInfoData.action';
					curCodeList = vm.enbCodeList.join(",");
				}
				
				if(vm.chartNetType == 'enb' && vm.chartActiveName == 'deviceGroup'){

				}else{
					//获得模板关联的设备详细数据
					$.ajax({
						type: "post",
						url: curGetChartCellsInfoUrl,
						data: {
							tempId: curForm.tempId,
							enbLimitNum: vm.enbLimitNum,
							enbCodeList: curCodeList
						},
						async: false,
						dataType:"json",
						success: function(data) {
                            vm.enbCodeList = [];
							vm.gnbSmallCellCodeAndCellIdObj = [];
							vm.enbSmallCellCodeAndCellIdObj = [];
							vm.devicesDefaultCheckedKeys = [];

							if(data && data.length > 0){
                               
								//获取当前已选设备对应数据信息
								vm.gnbCheckedObj = data; 
								vm.enbCheckedObj = data;
								$.each(data,function(index,obj){
									//5条数据中的 serialNumber
									vm.enbListName.push(obj.serialNumber);
									
									if(vm.chartNetType == "gnb"){
										//图表展示头部标题： serialNumber+(cell name + cell id)
										if(obj.cellId && obj.serialNumber){
											if(obj.hostName){
												vm.cellNameList.push(obj.serialNumber+"(" + obj.hostName + "/" + obj.cellId + ")");
											}else{
												vm.cellNameList.push(obj.serialNumber + "(" + obj.cellId + ")");
											}
										}else{
											if(obj.hostName){
												vm.cellNameList.push(obj.serialNumber+"(" + obj.hostName +")");
											}else{
												vm.cellNameList.push(obj.serialNumber);
											}
										}

										if(obj.cellId){
											vm.gnbSmallCellCodeAndCellIdObj.push({"smallCellCode":obj.smallCellCode,"cellId":obj.cellId});
										}else{
											vm.gnbSmallCellCodeAndCellIdObj.push({"smallCellCode":obj.smallCellCode});
										}

										//vm.$nextTick(function(){
											//将选中的设备存入数组
											if(obj.cellId && obj.smallCellCode){
												vm.devicesDefaultCheckedKeys.push(obj.smallCellCode + "-"  + obj.cellId);
											}else{
												vm.devicesDefaultCheckedKeys.push(obj.smallCellCode);
											}
											//绘制图表需要的唯一标识
											vm.gnbSmallCellCodeCellIdList = vm.devicesDefaultCheckedKeys;
											//默认展开的父节点
											vm.devicesDefaultExpandedKeys = vm.devicesDefaultCheckedKeys;
										//});
									} else if(vm.chartNetType == "enb"){
										//图表展示头部标题： serialNumber+(cell name / plmnId) 2g: serialNumber + (cell name / bts_id)
										var namekeys = '';
										//如果含有 后面拼接 ‘/’
										if(obj.hostName){
											namekeys = obj.hostName;
										}	
										if(obj.plmnId && vm.levelType == 'plmn'){
											//如果前面有 hostName 则拼接 ‘/’
											if(namekeys){
												namekeys += "/" + '<%=rb.getString("GNBPLMNBiaoShi")%>:' + obj.plmnId;
											}else{
												namekeys += '<%=rb.getString("GNBPLMNBiaoShi")%>:' + obj.plmnId;
											}
										}
										//2g serialNumber (bts_id)
										if(obj.bts_id){
											if(namekeys){
												namekeys += "/" + 'BTS ID:' + obj.bts_id;
											}else{
												namekeys += 'BTS ID:' + obj.bts_id;
											}
											
										}
										//判断 nameKeys 是否为空
										if(namekeys){
											vm.cellNameList.push(obj.serialNumber + "(" + namekeys + ")");
										}else{
											vm.cellNameList.push(obj.serialNumber);
										}
										
										var itemObj = {};
										if(obj.smallCellCode){
											itemObj.smallCellCode = obj.smallCellCode;
										}
										/*if(obj.cellId){
											itemObj.cellId = obj.cellId;
										}*/
										if(obj.plmnId && vm.levelType == 'plmn'){
											itemObj.plmnId = obj.plmnId;
										}
										if(obj.bts_id){
											itemObj.bts_id = obj.bts_id;
										}
										vm.enbSmallCellCodeAndCellIdObj.push(itemObj);
										//tab  切换数据加载时间顺序导致不勾选
										//vm.$nextTick(function(){
											//将选中的设备存入数组
											var checkedKey = '';

											if(obj.smallCellCode){
												checkedKey = obj.smallCellCode;
											}
											/*if(obj.cellId){
												checkedKey += "-" + obj.cellId;
											}*/
											if(obj.plmnId && vm.levelType == 'plmn'){
												checkedKey += "-" + obj.plmnId;
											}
											if(obj.bts_id){
												checkedKey += "-" + obj.bts_id;
											}
											//设备已选数据使树形结构杯选中 
											vm.devicesDefaultCheckedKeys.push(checkedKey);
											
											//绘制图表需要的唯一标识
											vm.enbSmallCellCodeCellIdList = vm.devicesDefaultCheckedKeys;
											//默认展开的父节点
											vm.devicesDefaultExpandedKeys = vm.devicesDefaultCheckedKeys;
										//});
									}else{
										//wcg
										if(obj.hostName == '' || obj.hostName == null || obj.hostName == undefined){
											vm.cellNameList.push(obj.serialNumber);
										}else{
											vm.cellNameList.push(obj.serialNumber+"("+ obj.hostName +")");
										}
										vm.enbCodeList.push(obj.smallCellCode);
										//vm.$nextTick(function(){
											vm.devicesDefaultCheckedKeys = vm.enbCodeList;

											vm.devicesDefaultExpandedKeys = vm.enbCodeList;
										//});
									}
								})
							}
						}
					});
				}

				//获得模板关联的指标详细数据
				$.ajax({
					type: "post",
					url: curGetChartIndicatorsInfoUrl,
					data: {
						tempId : curForm.tempId,
						kpiLimitNum : vm.kpiLimitNum,
						kpiIdList : vm.kpiIdList.join(",")
					},
					async: false,
					dataType:"json",
					success: function(data) {
						vm.kpiIdList = [];
						$.each(data,function(index,obj){
							vm.kpiIdList.push(obj.kpiId);
								var title = obj.kpiId + " (" + obj.kpiName + ")";
								var kpiObj ={};
								kpiObj.kpiName = title;
								kpiObj.kpiId = obj.kpiId;
								kpiObj.unit = obj.unit;
								vm.perf_list_all_chart.push(kpiObj);
								var chart_id = "kpi_chart_"+obj.kpiId+"_${randomValue}";
								var chart_div = '<div class="kpiChartCellNameClass">'
												+'<div class="kpiChartTitles">'+ title +'</div>'
												+'<div id="'+chart_id+'" class="kpiChartList"><div>'
												+'</div>';
								var charDomCount = $("#"+chart_id+"_${randomValue}").size();
								if(charDomCount==0) kpi_chart_maindiv.append(chart_div);
						})
					}
				});
			},
		resetTreeSelet() {
			var vm = this;

			// 先清除正在运行的定时器，防止清空后又触发查询
			if(vm.devicesCheckChangeTimer) {
				clearTimeout(vm.devicesCheckChangeTimer);
				vm.devicesCheckChangeTimer = null;
			}
			if(vm.deviceGroupChangeTimer) {
				clearTimeout(vm.deviceGroupChangeTimer);
				vm.deviceGroupChangeTimer = null;
			}

			vm.enbCodeList = [];
			vm.gnbCodeList = [];
			vm.gnbSmallCellCodeAndCellIdObj = [];
			vm.enbSmallCellCodeAndCellIdObj = [];
			vm.devicesDefaultCheckedKeys = [];
			// 清空设备名称列表和已选设备信息
			vm.cellNameList = [];
			vm.enbListName = [];
			vm.gnbCheckedObj = [];
			vm.enbCheckedObj = [];
            if(vm.$refs.groupDevicesTree) {
				vm.$refs.groupDevicesTree.setCheckedKeys([]);

				//将5g 与其它网元分开
				// 清空图表数据,需要先判断图表是否存在
				if(vm.perf_list_all_chart && vm.perf_list_all_chart.length > 0){
					// 遍历所有图表进行清空，而不是只清空第一个
					vm.perf_list_all_chart.forEach(function(perf_obj){
						var chartDom = document.querySelector("#kpi_chart_" + perf_obj.kpiId + "_${randomValue}");
						if(chartDom){
							var charIns = echarts.getInstanceByDom(chartDom);
							if(charIns){
								// 完全清空图表的 series 和 legend
								charIns.setOption({
									legend: {data: []},
									series: []
								});
							}
						}
					});
				}
			}

			if(vm.chartNetType == "enb" && vm.chartActiveName == 'deviceGroup') {
					if(vm.$refs.deviceGroupTable) {
						vm.groupIdList = []; //参数
						vm.deviceGroupSelection = []; //清空已选
						vm.defaultCheckedGroupKeys = []; //清空默认勾选
						vm.$refs.deviceGroupTable.clearSelection();
					}
				}
			}
		},
		mounted(){
			var vm = this;
			vm.init();
			
			// 统一管理窗口 resize 事件，避免每个图表都添加监听器
			vm.resizeHandler = function(){
				// 遍历所有图表实例并调用 resize
				if(vm.perf_list_all_chart && vm.perf_list_all_chart.length > 0){
					vm.perf_list_all_chart.forEach(function(perf_obj){
						var chartDom = document.getElementById("kpi_chart_" + perf_obj.kpiId + "_${randomValue}");
						if(chartDom){
							var chartInstance = echarts.getInstanceByDom(chartDom);
							if(chartInstance){
								chartInstance.resize();
							}
						}
					});
				}
			};
			
			// 添加防抖的 resize 监听器
			$(window).on('resize.kpiChart_${randomValue}', function(){
				if(vm.resizeTimer) clearTimeout(vm.resizeTimer);
				vm.resizeTimer = setTimeout(vm.resizeHandler, 200);
			});
		},
		beforeDestroy(){
			// 清理定时器，防止内存泄漏
			if(this.devicesCheckChangeTimer) {
				clearTimeout(this.devicesCheckChangeTimer);
				this.devicesCheckChangeTimer = null;
			}
			if(this.deviceGroupChangeTimer) {
				clearTimeout(this.deviceGroupChangeTimer);
				this.deviceGroupChangeTimer = null;
			}
			if(this.resizeTimer) {
				clearTimeout(this.resizeTimer);
				this.resizeTimer = null;
			}
			
			// 移除 resize 监听器
			$(window).off('resize.kpiChart_${randomValue}');
		}
	})
</script>
