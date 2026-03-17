<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<!-- 告警图表开发页-->
<style>
	.chartConcent{
		position: relative;
	}
	.ChartTopTitle{
		height: 46px;
		display: flex;
		border-bottom: 1px	solid #E9E9E9;
	}
	.ChartTopTitle .titleLeft{
		display: flex;
		font-size: 16px;
		margin-left: 20px;
	}
	.titleLeft div{
		padding: 0 5px;
		height: 45px;
		line-height: 45px;
		text-align: center;
		font-weight: 550;
	}
	.titleRight{
		display: flex;
		flex: 1;
		align-items: center;
		justify-content: center;
	}
	.timeButtonLeft .el-button.is-circle [class^=el-icon-], .timeButtonRight .el-button.is-circle [class^=el-icon-]{
		color: #4D84FF;
		font-size: 12px;
	}
	.timeButtonRightBan .el-button.is-circle [class^=el-icon-]{
		color: #C3D5FD;
		font-size: 12px;
	}
	.chartTimeStyle{
		width: 120px;
	}
	.chartTimeStyle .el-input__inner{
		font-size: 16px;
		font-weight: 550;
		width: 120px;
		text-align: center;
		border: none;
		height: 30px;
		line-height: 30px;
	}
	.chartTimeStyle  .el-input__prefix{
		display: none;
	}
	.chartTimeStyle .el-input--prefix .el-input__inner{
		padding-left: unset;
		padding-right: unset;
	}
	.timeButtonLeft .el-button.is-circle,.timeButtonRight .el-button.is-circle,.timeButtonRightBan .el-button.is-circle{
		height: 18px;
		width: 18px;
	}
	.dayAndMonth{
		margin-left: 40px;
	}
	.pieButtonStyle{
		height: 50px;
		padding: 0px 20px;
		z-index: 2000;
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.pieButtonStyle div{
		cursor: pointer;
	}
	.pieButtonTitleCls{
		font-size: 14px;
		font-weight: bold;
	}
	.deviceGroupSty{
		height: 26px;
		border: 1px solid #DEDFE6;
		border-radius:13px 0px 0px 13px;
		line-height: 26px;
		padding: 0 10px;
	}
	.deviceGroupStySelect{
		height: 26px;
		color:#4D84FF;
		border: 1px solid #4D84FF;
		border-radius:13px 0px 0px 13px;
		line-height: 26px;
		padding: 0 10px;
	}
	.deviceSty{
		height: 26px;
		border: 1px solid #DEDFE6;
		border-left: 1px solid transparent;
		border-right: 1px solid transparent;
		line-height: 26px;
		padding: 0 10px;
	}
	.deviceStySelect{
		height: 26px;
		color:#4D84FF;
		border: 1px solid #4D84FF;
		line-height: 26px;
		padding: 0 10px;
	}
	.alarmIdSty{
		height: 26px;
		border: 1px solid #DEDFE6;
		border-radius:0px 13px 13px 0px;
		line-height: 26px;
		padding: 0 10px;
	}
	.alarmIdStySelect{
		height: 26px;
		color:#4D84FF;
		border: 1px solid #4D84FF;
		border-radius:0px 13px 13px 0px;
		line-height: 26px;
		padding: 0 10px;
	}
	.topTimeSty{
		margin-left: 15px;
		height: 26px;
		width: 206px;
	}
	.topTimeSty .el-range-editor.el-input__inner{
		width: 206px;
	}
	.topTimeSty .el-icon-time:before,.topTimeSty .el-icon-date:before{
		color:#4D84FF;
	}
	.topTimeSty .el-date-editor .el-range__close-icon{
		display: none;
	}
	#specialLook{
		pointer-events: all;
	}
	.footAlarmListSty{
		position: relative;
		height: 358px;
	}
	.footButtonStyle{
		position: absolute;
		top: 15px;
		left: 500px;
		z-index: 9999;
		display: flex;
	}
	.footButtonStyle div{
		cursor: pointer;
	}
	#specialLook{
		cursor: pointer;
		margin-left: 30px;
		border-bottom: 1px solid #FFFFFF;
	}
	.flex-col-chart {
		position: relative;
		overflow:visible;
	}
	.half-persent {
		margin: 0px;
		background-color: #fff;
		min-height: 240px;
	}
	.alarmSourceClass{
		display: flex;
	}
	#alarmChartInfo .footDataTop10ItemCls{
		height: 30px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0px 10px;
		margin: 0px 20px;
	}
	#alarmChartInfo .footDataTop10ItemCls .itemIndexCls{
		height: 18px;
		width: 18px;
		border-radius: 50%;
		margin-right: 10px;
		line-height: 18px;
		text-align: center;
	}
	#alarmChartInfo .footDataTop10ItemCls .top3ItemIndexCls{
		background: #EF5958;
		color: #FFFFFF;
	}
	#alarmChartInfo .footDataTop10ItemCls .top10ItemIndexCls{
		background: rgba(0, 0, 0, 0.1);
		color: rgba(0, 0, 0, 0.6);
	}
	#alarmChartInfo .concentPageHeader{
		height: 50px;
		display: flex;
		align-items: center;
		position: relative;
		background-color: #FFF;
		border-bottom: 1px solid #E9E9E9;
	}
	#alarmChartInfo .headTitleBox{
		width: 120px;
		font-size: 14px;
		font-weight: bold;
		padding-left: 20px;
	}
</style> 
<div id="alarmChartInfo" class="panelDefault">
	<div class="chartConcent">
		<div class="concentPageHeader">
			<div class="headTitleBox">
				<%=rb.getString("GaoJingTuBiao")%>
			</div>
			<div class="titleRight">
				<div class="timeButtonLeft">
					<el-button type="primary" size="mini" icon="el-icon-arrow-left" circle @click="timeReduce"></el-button>
				</div>
				<div class="chartTimeStyle">
					<el-date-picker 
						v-model="chartDayTimes"
						:picker-options="pickerOption"
						v-show="chartTimesType == 'day'"
						type="date"
						:editable="false"
						:clearable="false"
						value-format="yyyy-MM-dd"
						style="width:120px"
						@change="chartDayTimesChange"
					></el-date-picker>
					<el-date-picker 
						v-model="chartMonthTimes"
						:picker-options="pickerOption"
						v-show="chartTimesType == 'month'"
						type="month"
						:editable="false"
						:clearable="false"
						value-format="yyyy-MM"
						style="width:120px"
						@change="chartMonthTimesChange"
					></el-date-picker>
				</div>
				<div :class="timeButtonRightDisabled ?'timeButtonRightBan':'timeButtonRight'">
					<el-button type="primary" size="mini" icon="el-icon-arrow-right" circle @click="timeAdd" :disabled="timeButtonRightDisabled"></el-button>
				</div>
				<div class="dayAndMonth">
					<el-radio-group size="mini" v-model='chartTimesType ' class="commonRadioButton" @change="dayAndMonthClick">
						<el-radio-button label="day"><%=rb.getString("Tian")%></el-radio-button>
						<el-radio-button  label="month"><%=rb.getString("Yue")%></el-radio-button>
					</el-radio-group>
				</div>
			</div>
			<div style="width:120px"></div>
			<div class="newIconBoxCls-bt" style="right:20px;top:14px;" @click="closeAlarmChartInfo" tip="<%=rb.getString("GuanBi")%>">
				<span class="el-icon-close el-icon"></span>
			</div>	
			<div class="newIconBoxCls-bt" style="right:60px;top:14px;" @click="openExportChart" tip="<%=rb.getString("DaoChu")%>">		
				<span class="el-icon-operation-export el-icon"></span>
			</div>
		</div>
		<div class="ChartTopTitle">
			<div class="titleLeft">
				<div :class="topDataType == 'active'? 'titleLeftSelect': ''"><%=rb.getString("HuoDongGaoJing")%></div>
			</div>
		</div>
		<div style="display:flex;flex-wrap: wrap;border-bottom: 1px solid #E9E9E9;" >
			<div :class="chartClass" style="min-width:780px;flex:4;border-right:1px solid #D5DCEC;">
				<div id="topBarChart" style="width:98%;height:358px;min-width:800px"></div>
			</div>
			<div :class="chartClass" v-show="showEnbTOPN" style="flex:2">
				<!--<div id="topPieChart" style="width:98%;height:358px;border-left: 1px solid #E9E9E9;min-width:800px"></div>-->
				<div class="pieButtonStyle">
					<div class="pieButtonTitleCls">Top 10</div>
					<el-radio-group size="mini" v-model='topPieBtnType ' class="commonRadioButton" @change="topPieBtnClick">
						<el-radio-button label="Device"><%=rb.getString("SheBeiTongJi")%></el-radio-button>
						<el-radio-button  label="AlarmId"><%=rb.getString("GaoJingId")%></el-radio-button>
					</el-radio-group>
				</div>
				<div  :class="headTop10Loading ? 'loading' : ''" style="height:calc(100% - 50px);position:relative;">
					<div v-for="(item,index) in topDataTop10" class="footDataTop10ItemCls">
						<div style="display: flex;">
							<div :class=" index <= 2 ? 'itemIndexCls top3ItemIndexCls' : 'itemIndexCls top10ItemIndexCls'">{{index+1}}</div>
							<div class="alalrmTop10ItemNameBox">
								<span v-show="topPieBtnType == 'Device'" class="alalrmTop10NameCls" :title="item.deviceName">{{item.deviceName}}</span>
								<span v-show="topPieBtnType == 'AlarmId'" class="alalrmTop10NameCls" :title="item.probableCause">{{item.probableCause}}</span>
								({{item.statisticObject}})
							</div>
						</div>
						<div>
							<%=rb.getString("GaoJingShu")%>:
							<span style="color:#4D84FF;">{{item.alarmCount}}</span>
						</div>
					</div>
				</div>
			</div>
		</div>
		<div class="ChartTopTitle">
			<div class="titleLeft">
				<div class="titleLeftSelect" ><%=rb.getString("GaoJingLieBiao")%></div>
			</div>
		</div>
		<div class="footAlarmListSty">
			<div class="footButtonStyle">
				<div v-if="alarmSourceList.length == 1">
					<div v-for="(item,index) in alarmSourceList">
						<div :class="footDeviceType == item ?'deviceStySelect':'deviceSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
					</div>
				</div>
				<div v-if="alarmSourceList.length == 2" class="alarmSourceClass">
					<div v-for="(item,index) in alarmSourceList" >
						<div v-if="index == 0" :class="footDeviceType == item ?'deviceGroupStySelect':'deviceGroupSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 1" :class="footDeviceType == item ?'alarmIdStySelect':'alarmIdSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
					</div>
				</div>
				<div v-if="alarmSourceList.length == 3" class="alarmSourceClass">
					<div v-for="(item,index) in alarmSourceList">
						<div v-if="index == 0" :class="footDeviceType == item ?'deviceGroupStySelect':'deviceGroupSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 1" :class="footDeviceType == item ?'deviceStySelect':'deviceSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 2" :class="footDeviceType == item ?'alarmIdStySelect':'alarmIdSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
					</div>
				</div>
				<div v-if="alarmSourceList.length == 4" class="alarmSourceClass">
					<div v-for="(item,index) in alarmSourceList" >
						<div v-if="index == 0" :class="footDeviceType == item ?'deviceGroupStySelect':'deviceGroupSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 1" :class="footDeviceType == item ?'deviceStySelect':'deviceSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 2" :class="footDeviceType == item ?'deviceStySelect':'deviceSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 3" :class="footDeviceType == item ?'alarmIdStySelect':'alarmIdSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
					</div>
				</div>
				<div v-if="alarmSourceList.length == 5" class="alarmSourceClass">
					<div v-for="(item,index) in alarmSourceList" >
						<div v-if="index == 0" :class="footDeviceType == item ?'deviceGroupStySelect':'deviceGroupSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 1" :class="footDeviceType == item ?'deviceStySelect':'deviceSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 2" :class="footDeviceType == item ?'deviceStySelect':'deviceSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 3" :class="footDeviceType == item ?'deviceStySelect':'deviceSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
						<div v-if="index == 4" :class="footDeviceType == item ?'alarmIdStySelect':'alarmIdSty'" @click="footDeviceTypeClick(item)">{{item}}</div>
					</div>
				</div>
				<div style="width:30px;"></div>
				<div :class="footPieBtnType == 'DeviceGroup'?'deviceGroupStySelect':'deviceGroupSty'" :style="{'background-color':(footDeviceType == 'OMC' ? '#F5FAF7' : ''), 'color':(footDeviceType == 'OMC' ? '#999999' : '')}" @click="footPieBtnClick('DeviceGroup')"><%=rb.getString("SheBeiZu")%></div>
				<div :class="footPieBtnType == 'Device'?'deviceStySelect':'deviceSty'" :style="{'background-color':(footDeviceType == 'OMC' ? '#F5FAF7' : ''), 'color':(footDeviceType == 'OMC' ? '#999999' : '')}" @click="footPieBtnClick('Device')"><%=rb.getString("SheBeiTongJi")%></div>
				<div :class="footPieBtnType == 'AlarmId'?'alarmIdStySelect':'alarmIdSty'" @click="footPieBtnClick('AlarmId')"><%=rb.getString("GaoJingId")%></div>

			</div>
			<div style="height:100%;">
				<el-ctable 
					ref="alarmList" 
					:url="alarmListUrl" 
					:query-params="params" 
					:height="height" 
					:page-size="pageSize" 
					pagination="true"
					v-loading="loading"
					@load-success="alarmListLoadSuccess" 
				 >
					<template slot="toolbar">
						<div class="queryGroup">
							<el-input placeholder="<%=rb.getString("SheBeiZuMingCheng")%>" v-model="queryParams.searchText" @keyup.enter.native="searchResult" v-if="footPieBtnType == 'DeviceGroup'"></el-input>
							<el-input placeholder="<%=rb.getString("SheBeiWeiYiBiaoZhi")%>" v-model="queryParams.searchText" @keyup.enter.native="searchResult" v-if="footPieBtnType == 'Device'"></el-input>
							<el-input placeholder="<%=rb.getString("AlarmId")%>" v-model="queryParams.searchText" @keyup.enter.native="searchResult" v-if="footPieBtnType == 'AlarmId'"></el-input>
							<i class="el-icon el-icon-common-search" @click="searchResult"></i>
						</div>
					</template>
					
					<el-table-column prop="statisticObject" label="<%=rb.getString("SheBeiZuMingCheng")%>" v-if="footPieBtnType == 'DeviceGroup'"></el-table-column>
					<el-table-column prop="statisticObject" label="<%=rb.getString("SheBeiWeiYiBiaoZhi")%>" v-if="footPieBtnType == 'Device'"></el-table-column>
					<el-table-column prop="statisticObject" label="<%=rb.getString("AlarmId")%>" v-if="footPieBtnType == 'AlarmId'"></el-table-column>
					<el-table-column prop="probableCause" label="<%=rb.getString("KeNengYuanYin")%>" v-if="footPieBtnType == 'AlarmId'"></el-table-column>
					<el-table-column prop="alarmCount" label="<%=rb.getString("GaoJingShu")%>"></el-table-column>
					<el-table-column prop="endTime" label="<%=rb.getString("JieZhiShiJian")%>" ></el-table-column>
				</el-ctable>
			</div>
		</div>
	</div>
</div>
<!-- 添加模板结束 -->
<script type="text/javascript">
var dobj = Vue.getInstance('#alarmChartInfo');
dobj && document.querySelector('#alarmChartInfo').remove();

new Vue({
	el:'#alarmChartInfo',
	data(){
		var vm = this;
		return{
			pickerOption:{
				disabledDate(time){
					return time.getTime() >Date.getNow()-8.64e6
				}
			},
			topDatePickerOption:{
				disabledDate(time){
					var firstDate = new Date(vm.chartMonthTimes+'-01');
					firstDate.setDate(1);
					var endDate = new Date(vm.chartMonthTimes+'-01');
					endDate.setMonth(endDate.getMonth()+1);
					endDate.setDate(0);
					return  firstDate.getTime() > time.getTime() || time.getTime() > endDate.getTime() || time.getTime() >Date.getNow()-8.64e6
				}
			},
			chartDayTimes:formatDate(Date.getNow()).slice(0,10), // 日期时间
			chartMonthTimes:formatDate(Date.getNow()).slice(0,7), // 月
			chartDayTimesAfter:'', // 后一天
			chartMonthTimesAfter:'', // 月的最后一天
			chartTimesType:'day', //  日期类型  日-day  月-month
			topDataType:'active', // 头部类型切换  新增-add  所有-all
			topTimeValue:["00:00:00","23:59:59"], // 新增，所有 饼状图  时间-时分秒
			topDateValue:[], // 新增，所有 饼状图  时间-日
			topPieBtnType:'Device',  //  新增，所有 饼状图数据类型  1-设备组  2-设备 3-告警ID
			footPieBtnType:'Device',  // 活动，清除 饼状图数据类型  1-设备组  2-设备 3-告警ID
			footDeviceType:'',
			timeButtonRightDisabled:true,
			chartList: ['topBarChart','topPieChart'],
			charts: {},
			height:'100%',
			pageSize:50,
			alarmListUrl:'',
			params:{
				searchText:'', // 搜索 value
				templateId : '',
				timeZone : timeZone,
				statisticObject : 'Device',
				statisticType : 'ActiveTotal',
				queryStartTime : '',
				queryEndTime : '',
				statisticTime : 'Day',
				deviceType:'ENB',
			},
			queryParams:{
				searchText:''
			},
			templateId:'', // 模板id
			alarmSourceList:[],
			showEnbTOPN:true,
			loading:false,
			topDataTop10:[],
			headTop10Loading:false,
		}
	},
	watch:{
		chartDayTimes(val){
			var vm = this;
			if(vm.chartTimesType == 'month')return
			if(val == formatDate(Date.getNow()).slice(0,10)){
				vm.timeButtonRightDisabled = true;
			}else{
				vm.timeButtonRightDisabled = false;
			}
			var endDate = new Date(val);
			vm.chartDayTimesAfter = formatDate(addDate(endDate,1));
			vm.topTimeValue=["00:00:00","23:59:59"];
			vm.params.queryStartTime = val+' 00:00:00';
			vm.params.queryEndTime = formatDate(addDate(endDate,1));
			if(vm.showEnbTOPN == true){
				vm.proccessTopPie('topPieChart');
			}
			vm.proccessTopBarChart('topBarChart');
		},
		chartMonthTimes(val){
			var vm = this;
			if(vm.chartTimesType == 'day')return
			if(val == formatDate(Date.getNow()).slice(0,7)){
				vm.timeButtonRightDisabled = true;
			}else{
				vm.timeButtonRightDisabled = false;
			}
			var endDate = new Date(val+'-01');
			vm.topDateValue= [];
			vm.topDateValue[0] = formatDate(endDate).slice(0,10);
			vm.topDateValue[1] = formatDate(addDate(addMonth(endDate,1)+'-01',-1)).slice(0,10);
			vm.chartMonthTimesAfter = formatDate(addDate(addMonth(endDate,1)+'-01',-1));
			vm.params.queryStartTime = formatDate(endDate);
			var EndTime = new Date(addMonth(endDate,1)+'-01');
			vm.params.queryEndTime = formatDate(EndTime);
			if(vm.showEnbTOPN == true){
				vm.proccessTopPie('topPieChart');
			}
			vm.proccessTopBarChart('topBarChart');
		},
		footPieBtnType(val){
			var vm = this;
			vm.params.statisticObject = val;
		}
	},
	beforeUpdate(){
		this.$nextTick(()=>{
			this.$refs.alarmList.doLayout();
		})
	},
	methods:{
		init(templateId){
			var vm = this;
			vm.templateId = templateId;
			vm.params.templateId = templateId;
			vm.params.queryStartTime = vm.chartDayTimes +' 00:00:00';
			var endDate = new Date(vm.chartDayTimes);
			vm.chartDayTimesAfter = formatDate(addDate(endDate,1));
			vm.params.queryEndTime = vm.chartDayTimesAfter;

			axios.post("${ctx}/fault/viewConfig/queryViewInfoById.action",stringify({templateId:templateId,timeZone:timeZone})).then(function(response){
				var data = response.data;
				vm.alarmSourceList = data.deviceType ? data.deviceType.split(',') : '';
				vm.footDeviceType = vm.alarmSourceList[0];
				vm.footDeviceTypeClick(vm.footDeviceType);
				if(vm.alarmSourceList.indexOf('ENB') == -1){
					vm.showEnbTOPN = false;
					vm.chartList =  ['topBarChart'];
				}else{
					vm.showEnbTOPN = true;
					vm.chartList =  ['topBarChart','topPieChart'];
				}
				
				vm.chartList.map(function(code){
					var chart = document.querySelector('#'+code);
					if(chart) {
						vm.charts[code] = echarts.init(chart);
						if(code == 'topBarChart'){
							vm.charts[code].on('mouseover',(params)=>{
								var ev=params.event
								vm.charts[code].dispatchAction({
									type:'showTip',
									x:params.event.offsetX,
									y:params.event.offsetY,
									position:[params.event.offsetX,params.event.offsetY]
								})
							})
							vm.charts[code].on('globalout',(params)=>{
								vm.charts[code].dispatchAction({
									type:'hideTip',
								})
								vm.charts[code].dispatchAction({
									type:'updateAxisPointer',
									currTrigger:'leave'
								})
							})
						}
						// 根据请求的数据重新渲染图表
						vm.reloadChart(code);
					}else{
						vm.reloadChart(code);
					}
					// 窗口缩放时自适应
					window.removeEventListener('resize',vm.resizeChart);
					window.addEventListener('resize',vm.resizeChart);
				});
			})
			
		},
		// 图表自适应
		resizeChart(){
			var vm = this;
			vm.chartList.map(function(code){
				if(vm.charts[code]) vm.charts[code].resize();
			});
		},
		chartDayTimesChange(val){
			var vm = this;
			vm.chartMonthTimes = val;
		},
		chartMonthTimesChange(val){
			var vm = this;
			vm.chartDayTimes = val +vm.chartDayTimes.slice(-3);

		},
		// 天 月切换事件
		dayAndMonthClick(type){
			var vm = this;
			
			vm.chartTimesType = type;
			if(vm.chartTimesType == 'day'){
				
				vm.chartDayTimes = formatDate(Date.getNow()).slice(0,10);
				vm.topTimeValue=["00:00:00","23:59:59"];
				vm.params.queryStartTime = vm.chartDayTimes+' 00:00:00';
				var endDate = new Date(vm.chartDayTimes);
				vm.params.queryEndTime = formatDate(addDate(endDate,1));
				vm.params.statisticTime = 'Day';
				vm.timeButtonRightDisabled = true;
			}else{
				var endDate = new Date(vm.chartMonthTimes+'-01');
				vm.chartMonthTimes = formatDate(Date.getNow()).slice(0,7);
				vm.timeButtonRightDisabled = true;
				vm.topDateValue[0] = vm.chartMonthTimes+'-01'
				vm.topDateValue[1] = formatDate(addDate(addMonth(endDate,1)+'-01',-1)).slice(0,10);
				vm.params.statisticTime = 'Month';
				vm.params.queryStartTime = formatDate(endDate);
				var EndTime = new Date(addMonth(endDate,1)+'-01');
				vm.params.queryEndTime = formatDate(EndTime);
			}
			if(vm.showEnbTOPN == true){
				vm.proccessTopPie('topPieChart');
			}
			vm.proccessTopBarChart('topBarChart');
		},

		// 时间事件 递减
		timeReduce(){
			var vm = this;
			if(vm.chartTimesType == 'day'){
				var data = new Date(vm.chartDayTimes);
				data.setDate(data.getDate()-1);
				vm.chartDayTimes = formatDate(data).slice(0,10);
			}else{
				var dataMonth = new Date(vm.chartMonthTimes+'-01');
				dataMonth.setMonth(dataMonth.getMonth()-1);
				vm.chartMonthTimes = formatDate(dataMonth).slice(0,7);
			}
		},
		// 时间 递加
		timeAdd(){
			var vm = this;
			if(vm.chartTimesType == 'day'){
				var data = new Date(vm.chartDayTimes);
				data.setDate(data.getDate()+1);
				vm.chartDayTimes = formatDate(data).slice(0,10);
			}else{
				var dataMonth = new Date(vm.chartMonthTimes+'-01');
				dataMonth.setMonth(dataMonth.getMonth()+1);
				vm.chartMonthTimes = formatDate(dataMonth).slice(0,7);
			}
		},
		// 新增，所有饼图 类型切换
		topPieBtnClick(type){
			var vm = this;
			vm.topPieBtnType = type;
			vm.headTop10Loading = true;
			vm.topDataTop10 = [];
			vm.proccessTopPie('topPieChart');
		},
		// 表格 设备组 设备 告警ID 类型切换
		footPieBtnClick(type){
			var vm = this;
			if(vm.footDeviceType == 'OMC')return
			vm.footPieBtnType = type;
			vm.loading = true;
			
		},
		footDeviceTypeClick(type){
			var vm = this;
			vm.footDeviceType = type;
			vm.params.deviceType = type;
			
			if(vm.footDeviceType == 'OMC'){
				vm.footPieBtnType = 'AlarmId';
				vm.loading = true;
			}
			vm.alarmListUrl = '${ctx}/cell/fault/queryAlarmStatisticResult.action';
		},
		// 新增，所有饼图 时分秒 时间控件改变事件
		topTimeHourChange(val){
			var vm = this;
			vm.proccessTopPie('topPieChart');
		},
		// 新增，所有饼图 日 时间控件改变事件
		topTimeDateChange(val){
			var vm = this;
			vm.proccessTopPie('topPieChart');
		},
		// 图表导出
		openExportChart(){
			var vm = this,
				params={timeZone: timeZone},
				urls ='${ctx}/cell/fault/exportAlarmStatisticResult.action';
			params.templateId = vm.templateId;
			params.statisticType = 'ActiveTotal,ActiveTotalTable';
			params.deviceType = vm.params.deviceType;
			if(vm.showEnbTOPN == true){
				params.topNActiveTotalDeviceType = 'ENB';
			}
			params.activeTotalTableStatisticObject = vm.footPieBtnType;
			params.activeTotalTableDeviceType = vm.footDeviceType;
			if(vm.chartTimesType == 'day'){
				var endDate = new Date(vm.chartDayTimes);
				vm.chartDayTimesAfter = formatDate(addDate(endDate,1));
				params.queryStartTime = vm.chartDayTimes +' 00:00:00';
				params.queryEndTime = vm.chartDayTimesAfter;
				params.statisticTime = 'Day';
				if(vm.showEnbTOPN == true){
					params.topNActiveTotalStatisticObject = vm.topPieBtnType;
					if(vm.topTimeValue[1] == '23:59:59'){
						params.topNActiveTotalEndTime = vm.chartDayTimesAfter;
					}else{
						params.topNActiveTotalEndTime = vm.chartDayTimes +' '+vm.topTimeValue[1];
					}
					params.topNActiveTotalStartTime = vm.chartDayTimes +' '+vm.topTimeValue[0];
				}
			}else{
				var endDate = new Date(vm.chartMonthTimes+'-01');
				vm.chartMonthTimesAfter = formatDate(addDate(addMonth(endDate,1)+'-01',0));
				params.queryStartTime = vm.chartMonthTimes+'-01' +' 00:00:00';
				params.queryEndTime = vm.chartMonthTimesAfter;
				params.statisticTime = 'Month';
				var topMonthNextDate = new Date(vm.topDateValue[1]);
				if(vm.showEnbTOPN == true){
					params.topNActiveTotalStatisticObject = vm.topPieBtnType;
					params.topNActiveTotalStartTime = vm.topDateValue[0] +' 00:00:00';
					params.topNActiveTotalEndTime = formatDate(addDate(topMonthNextDate,1));
				}
			}
			params.statisticObject = 'Serverity';
			exportByForm(urls,params);
		},
		// 告警图表搜索点击事件
		searchResult(){ 
			var vm = this;
			vm.params.searchText = vm.queryParams.searchText;
		},
		/**
		* 指定图表数据刷新
		* @param code{string} 图表类型
		*/
		reloadChart(code){
			var vm = this;

			if(vm.charts[code]) {
				if(['topBarChart'].includes(code)) {
					vm.proccessTopBarChart(code); // 头部柱状图表统计数据刷新
				}
			}else{
				if(['topPieChart'].includes(code)) {
					vm.proccessTopPie(code);// 头部饼图统计数据刷新
				}
			}
		},
		// 头部柱状图表统计数据刷新
		proccessTopBarChart(code){
			var vm = this,params = {};
		
			params.templateId = vm.templateId;
			params.timeZone = timeZone;
			params.statisticObject = 'Serverity';
			params.statisticType = 'ActiveTotal';
			
			if(vm.chartTimesType == 'day'){
				var endDate = new Date(vm.chartDayTimes);
				vm.chartDayTimesAfter = formatDate(addDate(endDate,1));
				params.queryStartTime = vm.chartDayTimes +' 00:00:00';
				params.queryEndTime = vm.chartDayTimesAfter;
				params.statisticTime = 'Day'
			}else{
				var endDate = new Date(vm.chartMonthTimes+'-01');
				var EndTime = new Date(addMonth(endDate,1)+'-01');
				vm.chartMonthTimesAfter = formatDate(EndTime);
				params.queryStartTime = vm.chartMonthTimes+'-01' +' 00:00:00';
				params.queryEndTime = vm.chartMonthTimesAfter;
				params.statisticTime = 'Month'
			}
			
			axios.post('${ctx}/cell/fault/queryAlarmStatisticResult.action',stringify(params)).then(function(response){
					var data = '';
					data = response.data.rows;
					vm.createOption(data,code)
			}) 
		},
		// 头部饼图统计数据刷新
		proccessTopPie(code){
			var vm = this,params = {};
		
			params.templateId = vm.templateId;
			params.timeZone = timeZone;
			params.deviceType = 'ENB';
			params.statisticType = 'ActiveTotal';
			params.topNActiveTotalDeviceType = 'ENB';
			params.statisticObject = vm.topPieBtnType;
			if(vm.chartTimesType == 'day'){
				if(vm.topTimeValue[1] == '23:59:59'){
					params.queryEndTime = vm.chartDayTimesAfter;
				}else{
					params.queryEndTime = vm.chartDayTimes +' '+vm.topTimeValue[1];
				}
				params.queryStartTime = vm.chartDayTimes +' '+vm.topTimeValue[0];
				params.statisticTime = 'Day'
			}else{
				var endDate = new Date(vm.topDateValue[1]);
				params.queryStartTime = vm.topDateValue[0] +' 00:00:00';
				params.queryEndTime = formatDate(addDate(endDate,1));
				params.statisticTime = 'Month'
			}
			axios.post('${ctx}/cell/fault/queryAlarmStatisticTopN.action',stringify(params)).then(function(response){
				var data = '';
				data = response.data.rows;
				vm.topDataTop10 = data.sort((a,b)=>{return b.alarmCount - a.alarmCount});
				vm.headTop10Loading = false;
				// vm.topDataTop10 = vm.topDataTop10.sort((a,b)=>{return b.alarmCount - a.alarmCount});
				// vm.createOption(data,code)
			}) 
		},
		// 生成图表的option 
		createOption(data,code){
			var vm = this,
				optionData = data;
			var codes = {
				'topBarChart':vm.createTopBarChartOption,
				'topPieChart':vm.createTopPieChartOption,
			}
			codes[code](optionData,code);	
		},
		// 创建头部柱状图Option
		createTopBarChartOption(data,code){
			var vm = this,legendName = ['<%=rb.getString("GaoJingShu")%>'],lineName = '<%=rb.getString("GaoJingShu")%>',xAxisName,xAxisData,optionData = data,
				monthDayNum,criticalData=[],majorData=[],minorData=[],warningData=[],lineData=[];

			if(vm.chartTimesType == 'day'){
				xAxisName = '<%=rb.getString("XiaoShi")%>'
				xAxisData = [];
				for(var i=1;i<25;i++){
					var times;
					if(i<10){
						times = vm.chartDayTimes +' 0'+ i + ':00:00'
					}else{
						times = vm.chartDayTimes +' '+ i + ':00:00'
					}
					xAxisData.push(times)
				}
			}else{
				xAxisName = '<%=rb.getString("Tian")%>';
				xAxisData = [];
				var selectDate = new Date(vm.chartMonthTimes+'-01');
				var selectMonth = selectDate.getMonth()+1;
				selectDate.setMonth(selectMonth);
				selectDate.setDate(0);
				monthDayNum = selectDate.getDate();
				
				for(var i=1;i<=monthDayNum;i++){
					var times;
					if(i<10){
						times = vm.chartMonthTimes +'-0'+ i
					}else{
						times = vm.chartMonthTimes+'-' + i;
					}
					xAxisData.push(times)
				}
			}
			for(var i=0;i< xAxisData.length;i++){
				criticalData.push(0);
				majorData.push(0);
				minorData.push(0);
				warningData.push(0);
				lineData.push(0);
			}
			optionData.map((item)=>{
				var idx;
				if(vm.chartTimesType == 'day'){
					if(item.endTime.split('-')[2].substring(0,2) == vm.chartDayTimes.split('-')[2]){
						idx = parseInt(item.endTime.split(' ')[1].substring(0,2),10)-1;
					}else{
						idx = 23;
					}
				}else{
					if(item.endTime.split('-')[1] == vm.chartMonthTimes.split('-')[1]){
						idx = parseInt(item.endTime.split('-')[2],10)-2;
					}else{
						idx = monthDayNum - 1;
					}
				}
				if(item.statisticObject == 31001){
					criticalData[idx] = item.alarmCount;
				}else if(item.statisticObject == 31002){
					majorData[idx] = item.alarmCount;
				}else if(item.statisticObject == 31003){
					minorData[idx] = item.alarmCount;
				}else if(item.statisticObject == 31004){
					warningData[idx] = item.alarmCount;
				}
			})
			for(var i=0;i<lineData.length;i++){
				lineData[i] =parseInt(criticalData[i])+parseInt(majorData[i])+parseInt(minorData[i])+parseInt(warningData[i]);
				if(lineData[i] == 0){
					lineData[i] = '';
				}
			}
			
			var option = {
				tooltip : {
					trigger : 'axis',
					triggerOn:'none',
					enterable:true,
					axisPointer:{
						type:'line',
					},
					textStyle:{
						fontSize:12
					},
					formatter:function(param){
						var str='',
							totalNum = 0;
							arrData = param;
						arrData.map((item)=>{
							if(item.value){
								totalNum += item.value
							}
						})
						str+=param[0].name+'<span class="lookSty" id="specialLook" onclick="lookVideGo(&quot;'+param[0].name+'&quot;,&quot;'+vm.topDataType+'&quot;)"><%=rb.getString("ChaKan")%>>></span>'+"<br>";
						str+= '<span style="display:inline-block;margin-right:5px;border-radius:10px;width:9px;height:9px;background-color:#69B6FC;"></span>'+'<%=rb.getString("ShouYe_ZongShu")%>: '+totalNum+"<br>";
						str+=param[0].marker+param[0].seriesName+": "+param[0].value+"<br>";
						str+=param[1].marker+param[1].seriesName+": "+param[1].value+"<br>";
						str+=param[2].marker+param[2].seriesName+": "+param[2].value+"<br>";
						str+=param[3].marker+param[3].seriesName+": "+param[3].value+"<br>";
						return str
					},
					extraCssText:"z-index:1000"
				},
				legend:{
					data:['Critical','Major','Minor','Warning'],
					bottom:10,
					selectedMode:false,
					itemHeight:8,
					itemWidth:8,
					itemGap:50
				},
				grid:{
					top:'12%',
					left:'3%',
					right:'6%',
					bottom:'10%',
					containLabel:true
				},
				xAxis:[{
					name : xAxisName,
					type: 'category',
					data:xAxisData,
					axisLabel : {
						show:true,
						textStyle:{ color:"#333333" },
						lineStyle:{ color:"#b6b6b6" },
						formatter : function(val) {
							
							if(vm.chartTimesType == 'day'){
								var secondTime = val.split(' ')[1];
								clock = secondTime.substring(0,2);
								return clock;
							}else{
								var secondTime = val.split('-')[2];
								return secondTime;
							}
							
						},
					},
					splitLine:{
						show:true,
						lineStyle:{
							type:'dashed',
							color:['#E9E9E9'],
						}
					}
				}],
				yAxis:[{
					name : 'Alarm Count',
					type: 'value',
					splitLine:{
						show:true,
						lineStyle:{
							type:'dashed',
							color:['#E9E9E9'],
						}
					}
				}],
				series:[
					// {
					// 	name:lineName,
					// 	type:'line',
					// 	itemStyle:{
					// 		normal:{
					// 			color:'#69B6FC',
					// 			areaStyle:{
					// 				color:'#69B6FC',
					// 				opacity:0.1
					// 			},
					// 			lineStyle:{
					// 				color:'#69B6FC'
					// 			}
					// 		},
					// 	},
					// 	connectNulls:true,
					// 	data:lineData,
					// },
					{
						name:'Critical',
						type:'bar',
						stack:'程度',
						barWidth:20,
						itemStyle:{
							normal:{
								color:'#FC5959',
							},
						},
						data:criticalData,
					},
					{
						name:'Major',
						type:'bar',
						stack:'程度',
						barWidth:20,
						itemStyle:{
							normal:{
								color:'#FF973E',
							},
						},
						data:majorData,
					},
					{
						name:'Minor',
						type:'bar',
						stack:'程度',
						barWidth:20,
						itemStyle:{
							normal:{
								color:'#FFDA41',
							},
						},
						data:minorData,
					},
					{
						name:'Warning',
						type:'bar',
						stack:'程度',
						barWidth:20,
						itemStyle:{
							normal:{
								color:'#67DFF8',
							},
						},
						data:warningData,
					},
				]
			};
			vm.charts[code].setOption(option);
			vm.charts[code].resize();
		},
		// 创建头部饼状图Option
		createTopPieChartOption(data,code){
			var vm = this,titleText ='<%=rb.getString("BingZhuangTuBiaoTi")%>',optionData = data,
				pieName,legendData=[],pieData=[];
			if(vm.chartTimesType == 'day'){
				pieName = vm.chartDayTimes;
			}else{
				pieName = vm.chartMonthTimes;
			}
			
			if(optionData.length == 0){
				pieData = [{
					value:0,
					name:'<%=rb.getString("MeiShuJu")%>'
				}];
				legendData=['<%=rb.getString("MeiShuJu")%>']
			}else{
				optionData.map((item)=>{
					var param={};
					param.value = item.alarmCount;
					param.name = item.statisticObject;
					param.cause = item.probableCause;
					pieData.push(param);
					legendData.push(item.statisticObject)
				})
			}

			var	option = {
					title:{
						text:titleText,
						left:'left',
						textStyle:{
							fontSize:12
						}
					},
					tooltip : {
						trigger: 'item',
						formatter:function(param){
							var text = param.seriesName+"<br>"+param.marker+param.name+":"+param.value+"<br>";
							
							if(param.data.cause) {
								text = param.seriesName+"<br>("+param.data.cause+")<br>"+param.marker+param.name+":"+param.value+"<br>";
							}
							
							return text;
						},
						extraCssText:"z-index:3000"
					},
					legend:{
						type:'plain',
						orient:'horizontal',
						top:20,
						left:0,
						selectedMode:false,
						itemHeight:8,
						icon:"circle",
						formatter:function(name){
							if(name.length >16){
								return name.slice(0,16)+'...'
							}else{
								return name
							}
							
						},
						data:legendData,
					},
					series:[
						{
							name:pieName,
							type:'pie',
							radius:'55%',
							hoverAnimation:true,
							center:['50%','60%'],
							data:pieData,
							
							itemStyle:{
								emphasis:{
									itemStyle:{
										shadowBlur:10,
										shadowOffserX:0,
										shadowColor:'rgba(0,0,0,0.5)'
									}
								},
								normal:{
									color:function(params){
										if(optionData.length == 0){
											var colorList = ['#E6E6E6'];
											return colorList[params.dataIndex]
										}else{
											var colorList = ['#69B6FC','#90EC97','#E9A4A4','#F3CC90','#D7A3EF','#ADA3EF','#4BDEDB','#4BDEA3','#E6A46B','#EFA3D3','#E9E9E9'];
											return colorList[params.dataIndex]
										}
									}
								},
							},
						},
					]
			};
			vm.charts[code].setOption(option);
			vm.charts[code].resize();
		},
		alarmListLoadSuccess(data){
			var vm = this;
			vm.loading = false;
		},
		closeAlarmChartInfo(){
			alarmViewVue.$refs.sharingSlide.hide();
		}
	},
	computed:{
		// 图表样式
		chartClass(){
			return {
				'flex-col-chart': true,
				'half-persent': true
			};
		},
	},
	mounted(){
		var vm = this;
		eventBus.$off('custom-chart-task').$on('custom-chart-task',this.init);
		window.lookVideGo=function(val,tabType){
			var params={};
			if(tabType == 'clear'){
				params.alarmType = 'HISTORY';
			}else{
				params.alarmType = 'ALL';
			}
			params.timeType = vm.chartTimesType;
			params.timeVal = val;
			params.templateId = vm.templateId;
			params.tabType = tabType;
			eventBus.$emit("open-chartDetail",params)
		}
		
	},
})

</script>