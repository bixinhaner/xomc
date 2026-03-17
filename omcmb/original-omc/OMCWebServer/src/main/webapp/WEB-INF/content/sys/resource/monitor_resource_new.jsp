<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#resourceLayout .flex-col {
		position: relative;
		flex: 1 auto;
		overflow: auto;
	}
	#resourceLayout .half-persent {
		height: calc(50% - 10px);
		margin: 5px;
		flex: 1 1 45%;
		background-color: #fff;
		border-radius: 5px;
		min-height: 240px;
		box-sizing:border-box;
		border:1px solid #E9E9E9;
	}
	#resourceLayout .half-persent:nth-of-type(2n+1) {
		margin-left: 0px;
	}
	#resourceLayout .half-persent:nth-of-type(2n) {
		margin-right: 0px;
	}
	#resourceLayout .inner-export {
		position: absolute;
		right: 10px;
		display: inline-block;
		width: 30px;
		height: 40px;
		cursor: pointer;
		z-index: 100;
	}
	
	#resourceLayout .flex-form {
		display: flex;
		flex-wrap: wrap;
	}
	#resourceLayout .flex-form .el-form-item {
		flex: auto;
		width: 45%;
	}
	#resourceLayout .flex-form .el-form-item__content {
		padding-top: 5px;
	}
	#resourceLayout .segment-title {
		display: flex;
		align-items: center;
		padding: 10px 0;
		border-bottom: 1px solid #ddd;
		margin-bottom: 15px;
	}
	#resourceLayout .segment-footer {
		padding: 10px 0;
		border-top: 1px solid #ddd;
		margin-top: 15px;
	}
	#resourceLayout .text-title {
		flex: auto;
		font-size: 14px;
		font-weight: bold;
	}
	#resourceLayout .server-list {
		display: flex; 
		align-items: center;
		padding: 0px 5px;
		margin: 15px 20px;
		border: 1px solid #ddd;
		border-radius: 5px;
	}
	#resourceLayout .server-list > div {
		display: flex;
		align-items: center;
		padding: 10px 20px;
		flex: auto;
		border-right: 1px solid #eee;
		min-height: 50px;
	}
	#resourceLayout .server-list > div:last-child {
		border: none;
	}
	#resourceLayout .server-flex-item {
		flex: auto;
		padding: 5px 10px 5px 5px;
	}
	#resourceLayout .pointer-cls .el-switch__core {
		cursor: pointer !important;
	}
	/*.right-label .el-form-item {
		margin-bottom: 10px;
		border-bottom: 1px solid #eee;
	}
	.right-label .el-form-item__label {
		font-size: 12px;
		text-align: left;
		color: #999;
	}
	.right-label .el-form-item__content {
		padding-top: 10px;
	}
	.word-break .el-form-item__content {
		word-break: break-word;
	}*/

	#resourceLayout .append-query {
		margin: 10px;
	}
	#resourceLayout .append-query .el-input-group__append {
		padding: 0 5px;
		background: transparent;
	}
	#resourceLayout .append-query .el-input__inner {
		border-right: none;
	}
	#resourceLayout .time-step {
		display: flex; 
		align-items: center;
	}
	#resourceLayout .time-step .el-icon {
		float:none;
		position:unset;
	}
	#resourceLayout .time-step .el-icon:before{
		font-size: 28px !important;
	}
	#resourceLayout .alarmHeaderTimeNotIconClass .el-date-editor .el-input__inner{
		cursor: pointer;
		opacity: 0.01;
	}
	#resourceLayout .alarmHeaderTimeNotIconClass .el-date-editor .el-input__prefix ,#dashboard_ctn .alarmHeaderTimeNotIconClass .el-date-editor .el-input__suffix{
		display: none;
	}
	#resourceLayout .el-tabs__header {
		border-bottom: solid 1px #eeeeee;
	}
	.selected-row {
		background-color: rgba(255,70,20,0.1) !important;
		color: #FF4614;
	}
	.selected-row .selectDataIcon {
		border-color: #FF4614;
	}
</style>
<!-- 系统资源管理界面 -->
<div class="overflow-cls">
<div id="resourceLayout" class='panelDefault' style="border:none;min-width: 1100px;">
	<el-tabs style="height: 100%;" @tab-click="tabClick">
		<el-tab-pane label="Resources">
			<div style="height: 100%; display: flex;">
				<div class="flex-ctn" style="width: 240px;min-width: 240px;">
					<div style="padding: 10px 10px 0 10px;color: #333;font-weight: bold;">
						Hostname(IP Address)
					</div>
					<div>
						<el-input v-model="searchText" class="append-query" placeholder="Host name / IP">
							<template slot="append">
								<i class="el-icon el-icon-common-search"></i>
							</template>
						</el-input>
					</div>
					<div style="overflow: auto;flex: auto;height: 100%;">
						<el-table :show-header="false" :data="hostIdList" size="mini" style="height: 100%;">
							<el-table-column prop="name"></el-table-column>
							<el-table-column width="50">
								<template slot-scope="scope">
									<i @click="selectRow(scope.row, event)" class="el-icon el-icon-common-changeOperator selectDataIcon"></i>
								</template>
							</el-table-column>
						</el-table>
					</div>
				</div>

				<div class="flex-ctn" style="position:relative;bottom:0px;top:0px;left:0px;right:0px;background: #f8f8f8;height: 100%;flex:auto;">
					<div style="position: absolute;color:#333;font-size: 14px;font-weight: bold;top: 16px;left: 10px;">
						{{currentRow? currentRow.name : ''}}
					</div>
					<div style="height: 50px;display: flex;min-height: 50px;justify-content: center;">
						<div class="time-step alarmHeaderTimeNotIconClass" style="position: relative;">
							<i class="el-icon el-icon-circle-left" @click="stepTimeChange(-1)"></i>
							<span style="display: inline-block;min-width: 120px;text-align: center;color:#363B4E;">
								<span style="font-size: 14px;font-weight: bold;">{{stepTime}}</span> 
								<el-date-picker ref="stepDate" :clearable="false"
									:picker-options="stepOptions"
									@change="stepDateChange"
									style="width: 110px;height: 25px;position: absolute;left: 36px;top: 12px;z-index: 90;" 
									v-model="stepDate" :type="stepDateType"></el-date-picker>
							</span>
							<i class="el-icon el-icon-circle-right" style="margin-left:0px;" :class="{disabled: isCurrentTime}" @click="stepTimeChange(1)"></i>
						</div>
						<!--
						<div style="width: 300px;padding: 15px;">
							<el-form :model="form" :rules="rules">
								<el-form-item prop="range">
									<el-date-picker v-model="form.range" type="daterange" size="mini" style="width: 260px;" :clearable="false" :editable="false"
									:picker-options="pickerOptions" value-format="yyyy-MM-dd HH:mm:ss" ></el-date-picker>
								</el-form-item>
							</el-form>
						</div>
						
						<div style="flex: 1 auto;text-align: center;">
							<div id="base_time_line" style="width: 99%;height: 98%;"></div>
						</div>
						-->
						<div style="padding: 10px 10px;text-align: right;">
							<div class="circleIcon placeholder-bt" style="top:12px;right:15px;" placeholder="<%=rb.getString("DaoChu")%>" @click="exportChart('All')">		
								<span class="el-icon el-icon-circle-export"></span>
							</div>
							<div class="circleIcon placeholder-bt" style="top:12px;right:55px;" placeholder="<%=rb.getString("ShuaXin")%>" @click="refreshChartsData">		
								<span class="el-icon el-icon-circle-refresh"></span>
							</div>
						</div>
					</div>
					
					<div id="charts_container" style="display: flex;flex-wrap: wrap;padding: 0px 10px 10px 10px;" class="flex-col">
						<div :class="chartClass">
							<span class="el-icon el-icon-operation-export inner-export placeholder-bt last-bt" tip="<%=rb.getString("DaoChu")%>" style="top: 5px;" @click="exportChart('CPU')"></span>
							<div id="cpu_chart" style="width: 98%;height: 98%;"></div>
						</div>
						<div :class="chartClass">
							<span class="el-icon el-icon-operation-export inner-export placeholder-bt last-bt" tip="<%=rb.getString("DaoChu")%>" style="top: 5px;" @click="exportChart('Memory')"></span>
							<div id="memory_chart" style="width: 98%;height: 98%;"></div>
						</div>
						<div :class="chartClass">
							<span class="el-icon el-icon-operation-export inner-export placeholder-bt last-bt" tip="<%=rb.getString("DaoChu")%>" style="top: 5px;" @click="exportChart('Disk')"></span>
							<div id="disk_chart" style="width: 98%;height: 98%;"></div>
						</div>
						<div :class="chartClass">
							<span class="el-icon el-icon-operation-export inner-export placeholder-bt last-bt" tip="<%=rb.getString("DaoChu")%>" style="top: 5px;" @click="exportChart('Database')"></span>
							<div id="dataBase_chart" style="width: 98%;height: 98%;"></div>
						</div>
						<div :class="chartClass">
							<span class="el-icon el-icon-operation-export inner-export placeholder-bt last-bt" tip="<%=rb.getString("DaoChu")%>" style="top: 5px;" @click="exportChart('Network')"></span>
							<div id="network_chart" style="width: 98%;height: 98%;"></div>
						</div>
					</div>
				</div>
			</div>
		</el-tab-pane>
		
		<el-tab-pane label="Remote Server">
			<div id="nms_ctn" style="height: 100%;">
				<div style="display: flex;height: 100%; width: 100%; min-width: 1000px;">
					<div style="display: flex; flex-direction: column;padding: 15px 0px; flex: auto; position: relative;border-radius: 5px;box-shadow: 1px 1px 5px #ddd;overflow: auto; height: 100%; background: #fff;">
						<el-query type="normal" placeholder="Server / IP" @query="query" style="padding-bottom: 5px;"></el-query>
						
						<div style="position: absolute;top: 10px;right: 10px;z-index: 1000;display: flex;">
							<i style="margin-right: 35px; padding-top: 5px;" @click="toSetting">
								<span style="margin-right: 10px;font-weight:bold;font-style:normal;font-size:14px;"><%=rb.getString("DingShiRenWuKaiGuan")%> </span>
								<el-switch v-model="settingForm.enableFlag" :active-value="1" :inactive-value="0" disabled style="opacity: 1;" class="pointer-cls"></el-switch>
							</i>
							<i v-if="false" @click="toSetting" tip="<%=rb.getString("SheZhi")%>" class="el-icon el-icon-circle-setting placeholder-bt" style="margin-right: 15px;"></i>
							<div class="circleIcon" style="right:0px;top:4px;">
								<span class="el-icon el-icon-circle-add" @click="toAdd"></span>
								<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
							</div>
						</div>
						
						<div style="flex: auto; overflow: auto;">
							<div v-for="row in list" class="server-list">
								<div style="max-width: 60px;">
									<i class="el-icon el-icon-operation-edit" style="margin-right: 10px;" @click="toEdit(row)"></i>
									<i class="el-icon el-icon-operation-delete" style="margin-right: 10px;" @click="delTask(row)"></i>
								</div>
								
								<div style="max-width: 40px;">
									<div>
										<%=rb.getString("SheZhiKaiGuan")%>
										<el-switch v-model="row.enableFlag" :active-value="1" :inactive-value="0" @change="changeStatus(row)"></el-switch>
									</div>
								</div>
								
								<div style="max-width: 200px;display: flex;align-items: center;font-size: 16px;min-width: 200px;border-right: none;">
									<i v-if="row.connectStatus == 'ON'" class="el-icon el-icon-status-conn-on" style="font-size: 24px;"></i>
									<i v-if="row.connectStatus == 'OFF'" class="el-icon el-icon-status-conn-off" style="font-size: 24px;"></i>
									<span style="margin-left: 5px;font-weight: bold;word-break: break-word;">{{row.serverName}}</span>
								</div>
								<div> </div>
								<div style="color: #666;">
									<div style="min-width: 75px;">
										<div style="padding-bottom: 5px;text-align: center;"><%=rb.getString("JianKangZhuangTai")%></div>
										<div style="height: 20px;text-align: center;">
											<span @click="fetchAlarm(row)" style="cursor: pointer;">
												<el-tag v-if="row.serverStatus == 'Critical'" type="danger" size="mini">{{row.serverStatus}}</el-tag>
											</span>
											<span v-if="row.serverStatus != 'Critical'">{{row.serverStatus}}</span>
										</div>
									</div>
									<div style="margin-left: 20px; flex: auto;">
										<div style="display: flex;padding: 5px;flex-wrap: wrap;">
											<span class="server-flex-item" style="min-width: 200px;width: 20%;">IP: {{row.serverIp}}</span>
											<span class="server-flex-item" style="min-width: 200px;width: 20%;"><%=rb.getString("YongHuMingCheng")%>: {{row.userName}}</span>
											<span class="server-flex-item" style="min-width: 200px;width: 20%;"><%=rb.getString("ChuangJianShiJian")%>: {{row.createTime}}</span>
											<span class="server-flex-item" style="min-width: 250px;width: 20%;"><%=rb.getString("ZuiXianGengXinShiJian")%>: {{row.updateTime}}</span>
										</div>
										<div style="padding: 5px 10px;">
											<%=rb.getString("EnbBeiZhu")%>: {{row.remark}}
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
					
					<div v-show="segShow" style=" border: 1px solid #DEDFE6;padding: 0 10px;width: 300px;border-radius: 5px;background: #fff;margin-left: 15px;box-shadow: 1px 1px 5px #ddd;display: flex;flex-direction: column;">
						<div class="segment-title">
							<span v-if="type == 'add'" class="text-title"><%=rb.getString("TianJia")%></span>
							<span v-if="type == 'edit'" class="text-title"><%=rb.getString("XiuGai")%></span>
							<span>
								<i @click="closeSlide" class='el-icon el-icon-close'></i>
							</span>
						</div>
						<el-form ref="form" :model="form" label-position="top" style="flex: auto;overflow: auto;">
							<el-form-item label="<%=rb.getString("CeSuFuWuQi")%>">
								<el-input v-model="form.serverName" maxlength="50" placeholder="max length: 50"></el-input>
							</el-form-item>
							
							<el-form-item label="IP">
								<el-input v-model="form.serverIp" maxlength="50" placeholder="max length: 50"></el-input>
							</el-form-item>

							<el-form-item label="<%=rb.getString("XieYi")%>">
								<el-select v-model="form.serverType" disabled>
									<el-option label="Redfish" value="redfish"></el-option>
								</el-select>
							</el-form-item>
							
							<el-form-item label="<%=rb.getString("YongHuMingCheng")%>">
								<el-input v-model="form.userName" maxlength="50" placeholder="max length: 50"></el-input>
							</el-form-item>
							
							<el-form-item label="<%=rb.getString("MiMa")%>">
								<el-input v-model="form.userPwd" type="password" maxlength="20" placeholder="max length: 20"></el-input>
							</el-form-item>
							
							<el-form-item label="<%=rb.getString("EnbBeiZhu")%>">
								<el-input type="textarea" v-model="form.remark" maxlength="200" rows="5" placeholder="max length: 200" :show-word-limit="true"></el-input>
							</el-form-item>
						</el-form>
						
						<div class="segment-footer">
							<el-button @click="save" type="primary"><%=rb.getString("QueDing")%></el-button>
							<el-button @click="closeSlide"><%=rb.getString("QuXiao")%></el-button>
						</div>
					</div>
				</div>
				
				<el-dialog ref="setting" title="<%=rb.getString("QueRen")%>" :visible.sync="settingShow" width="400">
					<el-form ref="settingForm" :model="settingForm" class="flex-form">
						<span style="padding-bottom: 10px;min-width: 200px;">{{settingForm.enableFlag==1?'<%=rb.getString("DingShiFuWuGuanBiTiShi")%>':'<%=rb.getString("DingShiFuWuKaiQiTiShi")%>'}}</span>
						<el-form-item label="<%=rb.getString("FenZhongZhouQi")%>" v-show="settingForm.enableFlag == 0">
							<el-select v-model="settingForm.period">
								<el-option label="5" :value="5"></el-option>
								<el-option label="10" :value="10"></el-option>
							</el-select>
						</el-form-item>
					</el-form>
					<div style="text-align: right;">
						<el-button @click="saveSetting" type="primary"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="closeSettingSlide"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</el-dialog>
			</div>
		</el-tab-pane>
	</el-tabs>

	<el-slide ref="slide" title="<%=rb.getString("GaoJingXiangQing")%>" :footer="false" @cancel="closeInfo">
		<el-table :data="alarmList" style="height: 100%;">
			<el-table-column label="<%=rb.getString("SuoYin")%>" prop="alarmId"></el-table-column>
			<el-table-column label="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>" prop="alarmIdentifier"></el-table-column>
			<el-table-column label="<%=rb.getString("GaoJingJiBie")%>" prop="serverity">
				<template slot-scope="scope">
					<el-tag type="danger" size="mini">{{scope.row.serverity}}</el-tag>
				</template>
			</el-table-column>
			<el-table-column label="<%=rb.getString("KeNengYuanYin")%>" prop="enName">
				<template slot-scope="scope">
					<span>{{isZH? scope.row.cnName : scope.row.enName}}</span>
				</template>
			</el-table-column>
			<el-table-column label="<%=rb.getString("JuTiGuZhang")%>" prop="additionalText"></el-table-column>
			<el-table-column label="<%=rb.getString("GaoJingZhuangTai")%>" prop="alarmType"></el-table-column>
			<el-table-column label="<%=rb.getString("GuZhangShiJian")%>" prop="eventTime"></el-table-column>
			<el-table-column label="<%=rb.getString("GengXinShiJian")%>" prop="updTime"></el-table-column>
		</el-table>
	</el-slide>
</div>
</div>
<script type="text/javascript">
new Vue({
	el: '#resourceLayout',
	data(){
		var vm = this,
			validRange = function(rule,value,callback) { // 校验时间范围
				var num = differ(vm.form.range[1],vm.form.range[0]);
				if(num>7){
					callback('<%=rb.getString("RiQiKuaDuChaoGuoQiTian")%>');
				}else{
					vm.range = vm.form.range;
					callback();
				}
			};
			
		return {
			hostList: [],
			stepTime: '',
			stepDate: '',
			timeType: 'day',
			currentRow: '',
			searchText: '',

			charts: {
				cpu: null,
				memory: null,
				disk: null,
				dataBase: null,
				base: null
			},
			range: [],// 用户图表的数据切换用
			currentIndex: 0,
			form: {
				range: [] // 用于校验日期选择
			},
			rules: {
				range: [
					{validator: validRange}
				]
			},
			chartList: ['CPU','Memory','Disk','Database','Network'],
			splitSize: 10, // 分割粒度
			
			tbUrl: '${ctx}/dell/idrac/config/list',
			type: 'add',
			queryParams: {
				searchTxt: ''
			},
			list: [],
			form: {
				id: '',
				enableFlag: 1,
				serverName: '',
				serverIp: '',
				serverType: 'redfish',
				userName: '',
				userPwd: '',
				remark: ''
			},
			settingForm: {
				enableFlag: 0,
				period: ''
			},
			segShow: false,
			settingShow: false,
			alarmShow: false,
			alarm: {
				alarmId: '',
				alarmIdentifier: '',
				additionalText: '',
				probableCause: '',
				enName: '',
				alarmType: '',
				eventTime: '',
				updTime: '',
				serverity: ''
			},
			alarmList: []
		};
	},
	computed: {
		isZH() {
			return isLocalZH == true;
		},
		// 图表样式
		chartClass(){
			return {
				'flex-col': true,
				'half-persent': true
			};
		},
		// 日期选择限制 当前日期开始向前30天
		pickerOptions() {
			var vm = this;
			return {
				disabledDate(time){
					let _now = Date.now(gloableTime),
						month = 30*24*60*60*1000,
						monthDays = _now - month;
					
					return time.getTime() > _now || time.getTime() < monthDays;
				}
			};
		},
		rangeStr(){
			return this.range.join('-');
		},
		stepOptions() {
			var vm = this;

			return {
				disabledDate: function(time) {
					var now = new Date(getYesterDay(0) + ' 00:00:00');

					return time.getTime() > now.getTime();
				}
			}
		},
		stepDateType() {
			var vm = this,
				type = 'date';

			if(vm.timeType == 'month') {
				type = 'month';
			};

			return type;
		},
		isCurrentTime() {
			var vm = this,
				bool = true,
				stepStr = (vm.stepTime||'').replace(/-/g,''),
				curStr = getYesterDay(0).replace(/-/g,'');

			if(vm.timeType == 'month') {
				stepStr = vm.stepTime.substring(0,7).replace(/-/g,'');
				curStr = getYesterDay(0).substring(0,7).replace(/-/g,'');
			}	
			if(stepStr-curStr<0) bool = false;

			return bool;
		},
		hostIdList() {
			var vm = this,
				searchText = vm.searchText.trim(),
				list = vm.hostList;

			if(searchText) {
				list = list.filter(function(item){
					return item.name.indexOf(searchText) >= 0;
				});
			}

			return list;
		}
	},
	watch:{
		rangeStr(val){
			var vm = this;
			vm.refreshBaseChart();
			vm.loadChartsData(vm.currentIndex);
		}
	},
	methods: {
		// 初始化
		init(){
			var vm = this;
			
			vm.charts.CPU = echarts.init(document.getElementById('cpu_chart'));
			vm.charts.Memory = echarts.init(document.getElementById('memory_chart'));
			vm.charts.Disk = echarts.init(document.getElementById('disk_chart'));
			vm.charts.Database = echarts.init(document.getElementById('dataBase_chart'));
			vm.charts.Network = echarts.init(document.getElementById('network_chart'));
			
			vm.initTimeStep();
			//vm.initTimeLine(); // 初始化时间轴
			
			vm.chartList.map(function(item){
				vm.charts[item].group = 'resource';
			});
			echarts.connect('resource');
			
			// 窗口缩放时自适应
			window.removeEventListener('resize',vm.resizeChart);
			window.addEventListener('resize',vm.resizeChart);
			
			var vm = this,
				url = '${ctx}/dell/idrac/config';
			
			axios.get(url).then(function(res){
				var data = res.data;
				
				if(data) {
					Object.assign(vm.settingForm, data);
				}
			});

			vm.getList();
			vm.getHostList();
		},
		// 初始化时间轴
		initTimeLine(){
			var vm = this,
				now = new Date(gloableTime),
				prevSeven = addDate(now,-7);

			vm.charts.base = echarts.init(document.getElementById('base_time_line'));
			
			vm.range = [dateformatter(prevSeven), dateformatter(now)];
			vm.form.range = [dateformatter(prevSeven), dateformatter(now)];
			vm.refreshBaseChart();// 时间轴option配置
			
			// 初始化时间轴切换事件
			vm.charts.base.on('timelinechanged',function(p){
				vm.currentIndex = p.currentIndex;
				vm.loadChartsData(p.currentIndex);
			});
		},
		tabClick() {
			setTimeout(this.resizeChart,50);
		},

		getHostList() {
			var vm = this;

			axios.post('${ctx}/system/resourceMonitor/getHostIdInfo.action').then((res)=>{
				var data = res.data;

				if(data.hostId) {
					var list = data.hostId.split(',');

					list.map(function(item) {
						vm.hostList.push({
							name: item,
							id: item
						});
					});

					vm.currentRow = vm.hostList[0];

					vm.refreshChartsData();

					setTimeout(function(){
						$('#resourceLayout tbody tr:first').addClass('selected-row');
					},100)
				}
			})
		},
		selectRow(row, evt) {
			var vm = this,
				el = evt.target;

			vm.currentRow = row;
			$('tr.selected-row').removeClass('selected-row');
			$(el).parents('tr').addClass('selected-row');
			vm.refreshChartsData();
		},
		// 初始化时间切换
		stepTimeChange(num) {
			var vm = this,
				type = vm.timeType,
				stepTime = vm.stepTime;
			
			if(num>0 && vm.isCurrentTime) {
				return;
			}
			
			if(type == 'day') {
				vm.stepTime = dateformatter(addDate(stepTime+' 00:00:00',num)).substring(0,10);

			}else if(type == 'month') {
				vm.stepTime = addMonth(stepTime+'-01 00:00:00',num).substring(0,7);
			}

			vm.stepDate = vm.stepTime;
			vm.refreshChartsData();
		},
		stepDateChange(val) {
			var vm = this,
				str = '';
			
			if(val) {
				str = dateformatter(val).substring(0,10);
			}

			if(vm.timeType == 'month') {
				str = str.substring(0,7);
			}
			vm.stepTime = str;
			vm.stepTimeChange(0)
		},
		initTimeStep() {
			var vm = this,
				now = getYesterDay(0);

			vm.stepTime = now;
			if(vm.timeType == 'month') {
				vm.stepTime = now.substring(0,7);
			}
			
			vm.stepDate = vm.stepTime;
		},
		refreshChartsData() {
			var vm = this,
				chartCtn = document.querySelector('#charts_container'),
				row = vm.currentRow || {};
			
			//loading
			chartCtn.classList.add('loading');
			vm.chartList.map(function(item){
				var urls = {
						CPU: '${ctx}/system/resourceMonitor/getCPUInfo',
						Memory: '${ctx}/system/resourceMonitor/getMemoryInfo',
						Disk: '${ctx}/system/resourceMonitor/getFileSystemInfo',
						Database: '${ctx}/system/resourceMonitor/getDatabaseInfo',
						Network: '${ctx}/system/resourceMonitor/getNetThroughInfo'
					},
					startTime = vm.stepTime + ' 00:00:00',
					endTime = dateformatter(addDate(new Date(startTime),1)),
					params = {
						timeZone: timeZone,
						startTime: startTime,
						endTime: endTime,
						hostId: row.id
					};
				
				axios.post(urls[item],stringify(params)).then(function(res){
					var data = res.data;
					var opts = vm.createOpts(item,data);
					vm.charts[item].setOption(opts, true);

					chartCtn.classList.remove('loading');
				}).catch(function(){
					chartCtn.classList.remove('loading');
				});
			});
		},

		// 图表自适应
		resizeChart(){
			var vm = this;
			vm.chartList.map(function(item){
				vm.charts[item].resize();
			});
			//vm.charts.base.resize();
		},
		// 时间轴option配置
		refreshBaseChart(){
			var vm = this;
			vm.currentIndex = vm.getDays().length - 1;
			
			var option = {
					baseOption: {
						timeline: {
							axisType: 'category',
							controlPosition: 'none',
						    symbolSize:8,
						  	lineStyle : {
								color : '#B0AFBA',
							  	width : 1
						  	},
						  	itemStyle : {
							  	normal : {
								  	borderColor : '#B0AFBA'
							  	},
							  	emphasis : {
								 	borderColor : '#4D84FF',
								  	color : '#4D84FF'
							  	}
						  	},
			              	checkpointStyle:{
			            	  	color:'#4D84FF',
			            	  	borderColor:'none'
			              	},
							data: vm.getDays(),
							currentIndex: vm.currentIndex
						}
					}
				};
			
			vm.charts.base.setOption(option);
		},
		/**
		* 重载图表数据
		* @param currentIndex{number}  时间轴下标 
		*/
		loadChartsData(currentIndex){
			var vm = this,
				chartCtn = document.querySelector('#charts_container');
			
			//loading
			chartCtn.classList.add('loading');
			vm.chartList.map(function(item){
				var urls = {
						CPU: '${ctx}/system/resourceMonitor/getCPUInfo',
						Memory: '${ctx}/system/resourceMonitor/getMemoryInfo',
						Disk: '${ctx}/system/resourceMonitor/getFileSystemInfo',
						Database: '${ctx}/system/resourceMonitor/getDatabaseInfo',
						Network: '${ctx}/system/resourceMonitor/getNetThroughInfo'
					},
					startTime = vm.getDays()[currentIndex] + ' 00:00:00',
					endTime = dateformatter(addDate(new Date(startTime),1)),
					params = {
						timeZone: timeZone,
						startTime: startTime,
						endTime: endTime
					};
				
				axios.post(urls[item],stringify(params)).then(function(res){
					var data = res.data;
					var opts = vm.createOpts(item,data);
					vm.charts[item].setOption(opts, true);

					chartCtn.classList.remove('loading');
				}).catch(function(){
					chartCtn.classList.remove('loading');
				});
			});
		},
		/**
		* 初始化图表Series配置
		* @param name{string}  图表名称 
		* @param chartData{Array / object}  CPU、Menory、Database图表数据为Array  Disk图表数据为object
		*/
		initSeries(name,chartData){
			var vm = this,
				hourses = vm.getHourses(),
				pointerNum = 24 * (60/vm.splitSize),
				usedKey = {
					CPU: 'cpu_used',
					Memory: 'memory_used',
					Database: 'database_con_num',
					Network: 'network_rec_total'
				},
				series = [],
				serie = {name: name, type: 'line', smooth: true, data: []},
				maxNum = null;

			for(var i=0; i<pointerNum; i++) serie.data.push({value:'-'});
			
			if(isArray(chartData)) { // CPU、Menory、Database数据处理
				chartData.map(function(row){
					var time = (row.collect_time||'').substr(11,5),
						idx = hourses.indexOf(time),
						max = null,percent = null;
					if(['Memory','Database'].includes(name)) {
						if('Database' == name) {
							max = row['max_connections'];
						}else{
							max = row['memory_total'];
							percent = row['memory_used_percent'];
						}
						maxNum = Math.max(maxNum,max);
					}
					if(['Memory','CPU'].includes(name)) maxNum = 100;

					if(idx>-1) {// 数据格式统一方便取值
						var value = row[usedKey[name]];
						if(['Memory'].includes(name)) value = percent;
							
						serie.data[idx] = {
							value: value,
							name: name,
							max: max,
							percent: percent? percent+'%':'',
							use: row[usedKey[name]],
							total: max
						};
					}
				});
				serie.max = maxNum;
				// init markline
				var markline = vm.initMarkline(serie.data);
				if(!['Memory'].includes(name)) serie.markLine = markline;
				
				series.push(serie);
			}else if(typeof chartData == 'object') {// Disk、Network 数据处理
				if(name == 'Network') {
					for(var key in chartData) {
						var serie = {name: key, type: 'line', smooth: true, data: []};
						for(var i=0; i<pointerNum; i++) serie.data.push({value:'-'});

						chartData[key].map(function(row) {
							var time = (row.collect_time||'').substr(11,5),
								idx = hourses.indexOf(time),
								max = null;

							if(idx >-1) serie.data[idx] = {// 数据格式统一方便取值
								value: row['network_rec_total'],
								name: key,
								max: max,
								percent: '',
								use: '',
								total: ''
							};
						});

						series.push(serie);
					}
				}else {
					var diskUsedkey = {
							system: 'disk_system_used',
							data: 'disk_data_used',
							log: 'disk_log_used'
						}
					for(var key in chartData){
						maxNum = null;
						var serie = {name: key, type: 'line', smooth: true, data: []};
						for(var i=0; i<pointerNum; i++) serie.data.push({value:'-'});
	
						chartData[key].map(function(row){
							var time = (row.collect_time||'').substr(11,5),
								idx = hourses.indexOf(time),
								max = null,
								totalKeys = { // 总量字段映射
									data: 'disk_data_total',
									log: 'disk_log_total',
									system: 'disk_system_total'
								},
								percentKeys = { // 百分比字段映射
									data: 'disk_data_used_percent',
									log: 'disk_log_used_percent',
									system: 'disk_system_used_percent'
								},
								tKey = totalKeys[key],
								pKey = percentKeys[key];
							if(row[pKey]) {
								max = row[pKey];
								maxNum = Math.max(maxNum,row[pKey]);
							}
							if(idx >-1) serie.data[idx] = {// 数据格式统一方便取值
								value: row[percentKeys[key]],
								name: key,
								max: max,
								percent: row[pKey]? (row[pKey]+'%'):"",
								use: row[diskUsedkey[key]],
								total: row[totalKeys[key]]
							};
						});
						serie.max = 100 || maxNum;
						
						series.push(serie);
					}
				}
			}

			return series;
		},
		/**
		* 图表峰值初始化
		* @param data{Array}  节点数据 
		*/
		initMarkline(data){ // init markline
			
			var markline = {data:[]},
				prev = null;
		
			data.map(function(item,idx){
				var first,
					second;
				
				if(idx) {
					if(prev.max != item.max) { // 值出现波动时，绘制之前等值的线
						first = {symbol:'none', xAxis: prev.idx, yAxis: prev.max};
						second = {symbol:'none', xAxis: idx-1, yAxis: prev.max};
						markline.data.push([first,second]);
						
						prev = item;
						prev.idx = idx;
					}
				}else {
					prev = item;
					prev.idx = idx;
				}
				
				if(idx == data.length - 1) {// 最后一个节点时
					first = {symbol:'none', xAxis: prev.idx, yAxis: prev.max};
					second = {symbol:'none', xAxis: idx, yAxis: item.max, value: item.max};
					markline.data.push([first,second]);
				}
			});
			return markline;
		},
		/**
		* 创建图表的option
		* @param name{string}  图表名称 
		* @param data{Array / object}  CPU、Menory、Database图表数据为Array  Disk图表数据为object 
		*/
		createOpts(name,data){
			var vm = this,
				series = vm.initSeries(name,data),
				maxNum = null,
				zoomArr = vm.getHourses(),
				ZongLiang = '<%=rb.getString("ZongLiang")%>',
				BaiFenBi = '<%=rb.getString("BaiFenBi")%>',
				unit = {
					CPU: '(%)',
					Database: '<%=rb.getString("GeShu")%>',
					Memory: '(%)',
					Disk: '(%)',
					Network: '(G)'
				};
			
			var tipNames = {
					CPU: ['<%=rb.getString("cpushiYongLu")%>'],
					Database: ['<%=rb.getString("LianJieShu")%>'],
					Memory: ['<%=rb.getString("ShiYongLiang")%>', ZongLiang, BaiFenBi],
					system: ['<%=rb.getString("xiTongPanShiYongLiang")%>', ZongLiang, BaiFenBi],
					data: ['<%=rb.getString("shuJuPanShiYongLiang")%>', ZongLiang, BaiFenBi],
					log: ['<%=rb.getString("riZhiPanShiYongLiang")%>', ZongLiang, BaiFenBi],
					Network: ['<%=rb.getString("networkJieShouLiang")%>']
				};

			var legend = series.map(function(item){ 
					if(item.max) maxNum = Math.max(maxNum,item.max);// 计算最大容量
					return item.name;
				});

			var option = {
					title: {
						text: name,
						textStyle: {
							fontSize: 14
						},
						left: 20,
						top: 10
					},
					tooltip: {
						trigger: 'axis',
						backgroundColor : 'rgba(205,224,232,0.9)',
						textStyle:{
							color:"#21608a",
							fontSize:12
						},
					},
					color: ['#6498FE','#58D8D8','#2BDC72','#67B83F','#B9DE22','#FFC400','#F38937','#FD6A75','#8B8BF2','#AD60F8'],
					legend: {
						top: 15,
						data: legend,
						formatter: function(name){
							if(['system','data','log'].includes(name)) {
								return tipNames[name][0].replace('G','%');
							}else{
								return tipNames[name]? tipNames[name][0] : name;
							}
						}
					},
					grid: {
						y2: 30
					},
					xAxis: {
						name: 'Time',
						data: zoomArr,
						axisLine: {lineStyle: {color: '#b6b6b6'}}
					},
					yAxis: {
						max: maxNum,
						name: unit[name],
						axisLine: {
							lineStyle: {color: '#b6b6b6'}
						},
						splitLine: {
							lineStyle: {
								color: '#f1f2f5'
							}
						}
					},
					series: series
				};

			option.tooltip.formatter = function(params){// tooltip格式化设置
				var str = "";
				if(params){
					params.map(function(param,idx){
						var data = param.data,
							names = tipNames[param.seriesName] || [param.seriesName];

						if(idx == 0) str += "<div>" + param.name + "</div>";

						if(['Disk','Memory'].includes(name)){
							if('Memory'==name) {
								str += "<div>" + names[1] + "(G): " + (data.total||"-") + "&nbsp;&nbsp;" + names[0] + "(G): " + (data.use||"-") + "&nbsp;&nbsp;" + names[2] + ": " + (data.percent||"-") + " </div>";
							}else {
								str += "<div>" + names[1] + "(G): " + (data.total||"-") + "&nbsp;&nbsp;" + names[0] + ": " + (data.use||"-") + "&nbsp;&nbsp;" +  names[2] + ": " + (data.percent||"-") + " </div>";
							}
						}else {
							if('Network'==name) {
								str += "<div>" + names[0] +" "+ tipNames[name] + ": " + data.value + "</div>";
							}else {
								str += "<div>" + names[0] + ": " + data.value + "</div>";
							}
						}
					});
				}
				
				return str;
			};

			return option;
		},
		// 获取小时粒度的time Data
		getHourses() {
			var vm = this,
				lineDate = [];
			for(var i = 0;i < 24;i++){
				//对24 取余 生成时间点 并格式化 01:00
				var formatterNumber = i,
					splitNum = 60 / vm.splitSize; // 一小时的分割段数
				if(formatterNumber < 10) formatterNumber = "0"+formatterNumber;
					
				for(var n = 0;n < splitNum; n++){
					var xDateStr = formatterNumber + ":" + (n?vm.splitSize*n:'00');
					lineDate.push(xDateStr);
				}
			}
			
			return lineDate;
		},
		// 获取天粒度的 time Data
		getDays() {
			var vm = this,
				lineDays = [],
				start = vm.range[0],
				end = vm.range[1],
				num = differ(end,start);
			
			for(var i = 0; i<=num; i++) {
				var dateStr = dateformatter(addDate(start,i));
				lineDays.push(dateStr.substr(0,10));
			}
			
			return lineDays;
		},
		/**
		* 图表数据导出
		* @param name{string}  图表名称 
		*/
		exportChart(name){
			var vm = this,
				row = vm.currentRow || {},
				urls = {
					CPU: '${ctx}/system/resourceMonitor/exportCPUInfo',
					Memory: '${ctx}/system/resourceMonitor/exportMemoryInfo',
					Disk: '${ctx}/system/resourceMonitor/exportFileSystemInfo',
					Database: '${ctx}/system/resourceMonitor/exportDatabaseInfo',
					Network: '${ctx}/system/resourceMonitor/exportNetThroughInfo',
					All: '${ctx}/system/resourceMonitor/exportAllResourceInfo'
				},
				dayArr = vm.getDays(),
				startTime = vm.stepTime + ' 00:00:00',
				endTime = dateformatter(addDate(new Date(startTime),1)),
				//currentDate = dayArr[vm.currentIndex],
				//range = vm.range[0].substr(0,10)+' - '+vm.range[1].substr(0,10),
				params = {
					timeZone: timeZone,
					startTime: startTime,
					endTime: endTime,
					hostId: row.id
				};
			/*
			if(name == 'All') {
				params.startTime = vm.range[0].substr(0,10) + ' 00:00:00';
				params.endTime = dateformatter(addDate(new Date(vm.range[1]),1)).substr(0,10) + ' 00:00:00';
			}
			*/
			exportByForm(urls[name],params);
		},
		
		query(text) {
			var vm = this;
			
			vm.queryParams.searchTxt = text;
			vm.getList();
		},
		getList() {
			var vm = this,
				url = '${ctx}/dell/idrac/config/list',
				params = {
					page: 0,
					rows: 20,
					timeZone: timeZone
				};
				
			Object.assign(params, vm.queryParams);
			
			axios({method:'get',url:url,params:params}).then(function(res){
				var data = res.data;
				
				vm.list = [];
				if(data) {
					vm.list = data.rows;
				}
			});
		},
		fetchAlarm(row) {
			var vm = this,
				url = '${ctx}/dell/idrac/config/alarm.action',
				params = {
					id: row.id,
					timeZone: timeZone
				};
			
			vm.resetAlarm();
			
			axios({method:'get',url:url,params:params}).then(function(res){
				var data = res.data;
				
				if(data) {
					vm.alarmList = data;
					//Object.assign(vm.alarm, data);
				}
				vm.showAlarmInfo();
			});
		},
		resetAlarm() {
			var vm = this;
			
			vm.alarmList = [];
			Object.assign(vm.alarm, {
				alarmId: '',
				alarmIdentifier: '',
				additionalText: '',
				probableCause: '',
				enName: '',
				alarmType: '',
				eventTime: '',
				updTime: '',
				serverity: ''
			});
		},
		showAlarmInfo() {
			var vm = this;
			
			vm.alarmShow = true;
			vm.segShow = false;
			vm.$refs.slide.showSlide();
		},
		toAdd() {
			var vm = this;
			
			vm.type = 'add';
			vm.resetForm();
			vm.$refs.form.clearValidate();
			vm.segShow = true;
			vm.alarmShow = false;
		},
		toEdit(row) {
			var vm = this;
			
			vm.type = 'edit';
			vm.resetForm();
			
			['id','enableFlag','serverName','serverIp','userName','userPwd','remark'].map(function(code){
				vm.form[code] = row[code];
			});
			
			vm.$refs.form.clearValidate();
			vm.segShow = true;
			vm.alarmShow = false;
		},
		resetForm() {
			var vm = this;
			
			Object.assign(vm.form, {
				id: '',
				enableFlag: 1,
				serverName: '',
				serverIp: '',
				userName: '',
				userPwd: '',
				remark: ''
			});
		},
		delTask(row) {
			var vm = this,
				url = '${ctx}/dell/idrac/config/delete',
				params = {
					id: row.id
				};
			
			vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
	            confirmButtonText: '<%=rb.getString("QueDing")%>',
	            cancalButtonText: '<%=rb.getString("QuXiao")%>',
	            type: 'warning'
            }).then(()=>{
            	axios.post(url,stringify(params)).then(function(res){
					var data = res.data;
					
					if(data.success == true) {
						eventBus.$message({
                            message: '<%=rb.getString("ChengGong")%>',
                            type: 'success'
                        });
						vm.getList();
					}else {
						vm.$message({
                            message: data.message,
                            type: 'error'
                        });
					}
				})
            }).catch(()=>{
            
            })
		},
		changeStatus(row) {
			var vm = this,
				url = '${ctx}/dell/idrac/config/update/enable',
				params = {
					id: row.id,
					enableFlag: row.enableFlag == 1? 1:0
				};
			
			axios.post(url,params).then(function(res){
				var data = res.data;
				
				if(data.success == true) {
					eventBus.$message({
                        message: '<%=rb.getString("ChengGong")%>',
                        type: 'success'
                    });

					vm.getList();
				}else {
					vm.$message({
                        message: data.message,
                        type: 'error'
                    });
				}
			})
		},
		toSetting() {
			var vm = this;
			
			vm.settingShow = true;
		},
		closeInfo() {
			var vm = this;

			vm.$refs.slide.hide();
		},
		closeSlide() {
			var vm = this;
			
			vm.segShow = false;
		},
		closeSettingSlide() {
			var vm = this;
			
			vm.settingShow = false;
		},
		save() {
			var vm = this,
				url = '${ctx}/dell/idrac/config/insert';
				
			if(vm.type == 'edit') {
				url = '${ctx}/dell/idrac/config/update';
			}
			
			vm.$refs.form.validate(function(r){
				if(r) {
					axios.post(url,vm.form).then(function(res){
						var data = res.data;
						
						if(data.message == 'exist') {
							eventBus.$message({
                                message: '<%=rb.getString("IPYiCunZai")%>',
                                type: 'error'
                            });
							
							return;
						}

						if(data.success == true) {
							eventBus.$message({
                                message: '<%=rb.getString("ChengGong")%>',
                                type: 'success'
                            });
							vm.closeSlide();
							vm.getList();
						}else {
							vm.$message({
                                message: data.message,
                                type: 'error'
                            });
						}
					})
				}
			})
		},
		saveSetting() {
			var vm = this,
				url = '${ctx}/dell/idrac/config/enable',
				params = {
					period: vm.settingForm.period,
					enableFlag: vm.settingForm.enableFlag==1?0:1
				};
				
			
			
			vm.$refs.settingForm.validate(function(r){
				if(r) {
					axios.post(url, params).then(function(res){
						var data = res.data;
						
						if(data.success == true) {
							eventBus.$message({
                                message: '<%=rb.getString("ChengGong")%>',
                                type: 'success'
                            });
							Object.assign(vm.settingForm, params);
							vm.closeSettingSlide();
						}else {
							vm.$message({
                                message: data.message,
                                type: 'error'
                            });
						}
					})
				}
			})
		}
	},
	mounted(){
		var vm = this;

		setTimeout(this.init,100);

		if(window.remoteServerInterval) {
			clearInterval(window.remoteServerInterval);
		}

		window.remoteServerInterval = setInterval(function(){
			var ctner = $('#nms_ctn');

			if(ctner.length) {
				vm.getList();
			}else {
				clearInterval(window.remoteServerInterval);
			}
		},6000);
	}
});
</script>