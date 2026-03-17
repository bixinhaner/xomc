<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>
<style type="text/css">
.pageContainer{
	display:flex;
	flex-direction:column;
	height:100%;
	flex: 1 auto;
	overflow: auto;
}
.selectButtonGroup{
	width:250px;
	align-self:center;
}
.selectButtonGroup span{
	display:inline-block;
	width:123px;
	height:29px;
	line-height:29px;
	border:1px solid #DEDFE6;
	text-align:center;
}
.selectSpan{
	background-color:#4D84FF;
	color:#FFF;
}
.viewTitle{
	font-weight:700;
	font-style:'normal';
	font-size: 16px;
	color:#5A7B92;
}
.echartContainer {
	position: relative;
	width: 95%;
	border: 1px solid #DEDFE6;
	flex: 1 1 100%;
	margin:auto;
	margin-top: 20px;
	display: flex;
	flex-direction:column;
	padding: 35px 0;
	overflow: auto;
}
.echartContainer .el-input__prefix{
	top:unset!
}
.iconTemp{
	vertical-align:middle;
	display:inline-block;
	width:30px;
	height:30px;
	cursor:pointer;
}
.titleTemp{
	position:absolute;
	top:60px;
	left:41px;
	z-index:10;
}
</style>
<div class="panelDefault" id="echartStatisticPage">
	<div class="circleIcon" style="top:10px;">
		<span class="el-icon el-icon-circle-table" @click="openChartPage"></span>
		<div class="titleButtonText"><%=rb.getString("LieBiao")%></div>
	</div>

	<div class="pageContainer">
		<div style="position: absolute;left:15px;top:10px;">
			<span class="el-icon el-icon-templateSetting" @click="showSlider" style="font-size:24px;"></span>
			<span class="titleButtonText titleTemp" style="top: 30px;word-break: keep-all;"><%=rb.getString("SheZhi")%></span>
		</div>
		<div class="selectButtonGroup" style="margin-top:10px;">
			<span style="border-right:none" :class="{selectSpan:hasSelect}" @click="hasSelect = true;resizeChart()"><%=rb.getString("ZhuXingTu")%></span><span style="border-left:none" :class="{selectSpan:!hasSelect}" @click="hasSelect = false;resizeChart();"><%=rb.getString("ZheXianTu")%></span>
		</div>
		<div class="echartContainer">
			<div style="align-self:center;flex-direction:column;margin-bottom:20px;" v-show="timeShow">
				<el-date-picker v-model="dataRange" size="mini" type="date" value-format="yyyy-MM-dd HH:mm:ss" :picker-options=pickerOptions
					@change="timeSelect" :editable="false" :clearable="false" placeholder="<%=rb.getString("RiQi")%>"></el-date-picker>
			</div>
			<div style="height:470px;" v-show="hasSelect">
				<div class="echartBox" id="echarBox" style="height:500px;width: 100%;" ref="mychart"></div>
			</div>
			
			<div style="height:470px;" v-show="!hasSelect">
				<div class="echartBox" id="lineEcharBox" style="height:500px;width: 100%;" ref="myLinechart"></div>
			</div>
		</div>
	</div>
	<el-slide ref="statisticSlider" :title="echartSlideTitle" position="top" @cancel="closeSlider" @ok="selectedOk">
		<el-ctable ref="device_select_list" height="99%" :url="listURL" :row-key="rowKey" :pagination="true">
			<el-table-column type="selection"></el-table-column>
			<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' prop="group_name" v-if="deviceType==1"></el-table-column>
			<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" v-if="deviceType===0"></el-table-column>
			<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" v-if="deviceType===0"></el-table-column>
			<el-table-column label='<%=rb.getString("AlarmReason")%>' prop="ALARM_NAME" v-if="deviceType==2"></el-table-column>
			<el-table-column label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' prop="ALARM_IDENTIFIER" v-if="deviceType==2"></el-table-column>
		</el-ctable>
	</el-slide>
</div>
<script type="text/javascript">
new Vue({
	el:"#echartStatisticPage",
	data(){
		return {
			dataRange:'',
			hasSelect:true,
			mychart:null,
			mylinechart:null,
			initTimeType:'',
			initStartTime:'',
			initEndTime:'',
			statisticId:'',
			statisticType: 'hourse',
			currentIndex: 0, // 当前选择的时间轴节点下标
			timeArr: [], // 时间轴,
			zoomArr: [], // 
			echartSlideTitle: '',
			tburl: '',
			deviceType: '',
			statisticObjects: ''
		}
	},
	computed: {
		timeShow() {
			var vm = this,
				isShow = false;
			if(vm.hasSelect) {
				if(vm.timeArr.length>0) isShow = true;
			}else if(vm.statisticType == 'hourse') {
				isShow = true;
			}
			return isShow;
		},
		rowKey(){
			var vm = this,
				rowkey = '';
			if(vm.deviceType == 1) rowkey = 'id';
			if(vm.deviceType === 0) rowkey = 'serial_number';
			if(vm.deviceType == 2) rowkey = 'ALARM_IDENTIFIER';

			return rowkey;
		},
		listURL(){
			return this.tburl;
		},
		pickerOptions() {
			var vm = this;
			return {
				disabledDate(time){
					let _now = Date.now(),
						seven = 7*24*60*60*1000,
						sevenDays = _now - seven,
						prevBool = false,
						sufBool = false;
					if(vm.initStartTime) {
						var prev = new Date(vm.initStartTime);
						prevBool = time.getTime() < prev;
					}
					if(vm.initEndTime) {
						var suf = new Date(vm.initEndTime);
						sufBool = time.getTime() > suf;
					}
					return time.getTime() > _now || prevBool || sufBool;
				}
			};
		}
	},
	methods:{
		// 关闭图表页
		openChartPage(){
			eventBus.$emit('hide-chartPage');
		},
		// 打开设置模板
		showSlider(){
			this.$refs.statisticSlider.showSlide();
		},
		// 关闭设置模板
		closeSlider(){
			this.$refs.statisticSlider.hide();
		},
		// 设置模板确定
		selectedOk(){
			var vm = this,
				ids = [];

			var selected = vm.$refs['device_select_list'].getChecked();

			vm.statisticObjects = selected.join(',');

			vm.reloadChart();
			vm.closeSlider();
		},
		/**
		* 获取图表数据
		* @param index{number}  时间轴下标 
		*/
		getChartData(index) {
			var vm = this,
				ctnObj = document.querySelector('.echartContainer'),
				prev = vm.addHourse(vm.dataRange,index-1),
				suf = vm.addHourse(vm.dataRange,index),
				params = {
					statisticId: vm.statisticId,
					statisticObjects: vm.statisticObjects,
					queryStartTime: prev,
					queryEndTime: suf,
					timeZone: timeZone,
					type: 'bar'
				};
		
			if(vm.statisticType == 'day') {
				params.queryStartTime = dateformatter(addDate(vm.initStartTime,index));
				params.queryEndTime = dateformatter(addDate(vm.initStartTime,index+1));
				vm.dataRange = dateformatter(addDate(vm.initStartTime,index));
			}
		
			ctnObj.classList.add('loading');
			vm.currentIndex = index;
			
			// 请求柱图图表数据
			axios.post("${ctx}/cell/fault/queryAlarmStatisticResult.action",stringify(params)).then(function(response){
				var rows = response.data.rows;
				// 设置柱图
				var ops = vm.createOption(rows);
				vm.mychart.setOption(ops);
				
				ctnObj.classList.remove('loading');
			}).catch(function(){
				ctnObj.classList.remove('loading');
			});
		},
		// 图表初始化
		initEchart(){
			var vm = this;
			if(vm.mychart) {
				// 重置一些参数
				vm.currentIndex = 0;
				vm.timeArr = [];
			}else {
				vm.mychart = echarts.init(document.getElementById('echarBox'));
				// 初始化时间轴切换事件
				vm.mychart.on('timelinechanged',function(p){
					vm.getChartData(p.currentIndex);
				});
			}
			
			// init line chart
			if(vm.mylinechart) {
				// 重置一些参数
				vm.zoomArr = [];
			}else {
				vm.mylinechart = echarts.init(document.getElementById('lineEcharBox'));
				// 初始化时间轴切换事件
				/* vm.mylinechart.on('timelinechanged',function(p){
					
				}); */
			}
			
			// 窗口缩放时自适应
			window.removeEventListener('resize',vm.resizeChart);
			window.addEventListener('resize',vm.resizeChart);
			
			vm.reloadChart();
		},
		// 图表自适应
		resizeChart(){
			var vm = this;
			vm.$nextTick(function(){
				vm.mychart.resize();
				vm.mylinechart.resize();
			});
		},
		/**
		* 获取图表数据
		* @param datestr{string}  点击事件所传时间字符串 
		* @param hourse{number}   
		*/
		addHourse(datestr,hourse){
			var date = new Date(datestr);
			date = date.valueOf();
			date += hourse*60*60*1000;
			return dateformatter(new Date(date));
		},
		// 重新加载图表数据
		reloadChart(){
			var vm = this,
				reduceOneDay = dateformatter(addDate(vm.dataRange,-1)),
				reduceOneHourse = vm.addHourse(vm.dataRange,-1),
				plusOneDay = dateformatter(addDate(vm.dataRange,1));
				params = {
					statisticId: vm.statisticId,
					statisticObjects: vm.statisticObjects,
					queryStartTime: reduceOneHourse,
					queryEndTime: vm.dataRange,
					timeZone: timeZone,
					type: 'bar'
				},
				ctnObj = document.querySelector('.echartContainer');
			// loading效果
			ctnObj.classList.add('loading');
			
			if(vm.statisticType == 'day') {
				params.queryStartTime = vm.dataRange;
				params.queryEndTime = plusOneDay;
			}else if('week' == vm.statisticType) {
				var endwStr = vm.initEndTime.substr(0,10) + ' 00:00:00';
				params.queryStartTime = vm.initStartTime;
				params.queryEndTime = dateformatter(addDate(endwStr,1));
			}else{
				params.queryStartTime = vm.addHourse(vm.dataRange, vm.currentIndex-1);
				params.queryEndTime = vm.addHourse(vm.dataRange, vm.currentIndex);
			}
			
			// 请求柱图图表数据
			axios.post("${ctx}/cell/fault/queryAlarmStatisticResult.action",stringify(params)).then(function(response){
				var rows = response.data.rows;
				// 设置柱图
				var ops = vm.createOption(rows);
				vm.mychart.setOption(ops);
				
				ctnObj.classList.remove('loading');
			}).catch(function(){
				ctnObj.classList.remove('loading');
			});
			
			if(vm.statisticType == 'hourse') {
				params.queryStartTime = vm.dataRange;
				params.queryEndTime = plusOneDay;
			}else if(['day','week'].includes(vm.statisticType)) {
				var endstr = vm.initEndTime.substr(0,10) + ' 00:00:00';
				params.queryStartTime = vm.initStartTime;
				params.queryEndTime = dateformatter(addDate(endstr,1));
			}
			params.type = 'line';
			// 请求折线图表数据
			axios.post("${ctx}/cell/fault/queryAlarmStatisticResult.action",stringify(params)).then(function(response){
				var rows = response.data.rows;
				// 设置折线图
				var lineops = vm.createLineOption(rows);
				vm.mylinechart.clear();
				vm.mylinechart.setOption(lineops);
				
				ctnObj.classList.remove('loading');
			}).catch(function(){
				ctnObj.classList.remove('loading');
			});
		},
		// 日期改变事件
		timeSelect() {
			var vm = this;
			vm.$nextTick(function(){
				if(vm.statisticType == 'day') vm.currentIndex = vm.timeArr.indexOf(vm.dataRange.substring(0,10))
				vm.reloadChart();
			});
		},
		/**
		* 生成series
		* @param chartData{Array}  图表数据 
		*/
		proccessLineData(chartData) {
			var vm = this;
			
			var chartsData = chartData || [];
			var codeMap = {};
			chartsData.map(function(item){
				var key = item.statisticObject,
					time = item.endTime,
					count = item.alarmCount,
					timeData = {time: time, count: count};
				if(codeMap[key]) {
					codeMap[key].push(timeData);
				}else {
					codeMap[key] = [timeData];
				}
			});
			// 判断统计粒度生成对应粒度数据
			var series = [];
			if(chartsData.length){
				// 迭代各统计对象，生成24小时数据，并组装为series对象
				for(var key in codeMap) {
					var dList = [],
						timeList = codeMap[key],
						serie = {name: key, type: 'line', smooth: true};
					
					for(var i=0; i<vm.zoomArr.length; i++) dList.push(null);
					
					timeList.map(function(item){
						var timeStr = item.time,
							tIdx = '';

						if(vm.statisticType == 'hourse') {// 小时粒度数据处理
							tIdx = timeStr.substr(11,2) - 0;
						}else if(vm.statisticType == 'day'){// 天粒度
							tIdx = vm.zoomArr.indexOf(timeStr.substr(0,10));
						}else {
							tIdx = 0;
						}
						
						if(tIdx !== '' && tIdx >= 0) dList[tIdx] = item.count;
					});
					
					serie['data'] = dList;
					series.push(serie);
				}
			}else{
				vm.statisticObjects.split(',').map(function(item){
					var serie = {name: item, type: 'line', smooth: true, data: []};
					series.push(serie);
				});
			}
			
			return series;
		},
		/**
		* 生成折线图的option
		* @param chartsData{Array}  图表数据 
		*/
		createLineOption(chartsData) {
			var vm = this;
			// 生成dataZoom data
			vm.initTimeLineData(vm.statisticType);
			
			// 生成series
			var series = vm.proccessLineData(chartsData);
			
			var legend = series.map(function(item){ return item.name ;});
			
			// 主要设置 xAxist的data 和 series数组
			var options = {
					tooltip: {
						trigger: 'axis'
					},
					grid: {
						y: 80
					},
					legend: {
						data: legend,
						width: '80%'
					},
					color: ['#6498FE','#58D8D8','#2BDC72','#67B83F','#B9DE22','#FFC400','#F38937','#FD6A75','#8B8BF2','#AD60F8'],
					xAxis: {
						name: 'Time',
						data: vm.zoomArr,
						axisLine: {lineStyle: {color: '#888'}}
					},
					yAxis: {
						name: 'Alarm Count',
						spliteLine: {
							show: true
						},
						axisLine: {lineStyle: {color: '#888'}}
					},
					dataZoom: [{
						show: vm.statisticType != 'week'
					},{
						type: 'inside'
					}],
					series: series
				};
			
			return options;
		},
		/**
		* 生成柱图的option
		* @param dataMap{Array}  图表数据 
		*/
		createOption(dataMap){
			var vm = this,
				keys = [], // 存所选设备编码
				subOps = [], // 存各个时间点数据
				seriesData = [],
				chartsData = dataMap || [];
			// 生成timeline data
			vm.initTimeLineData(vm.statisticType);
			
			
			if(chartsData.length){
				chartsData.map(function(item) {
					keys.push(item.statisticObject);
					seriesData.push(item.alarmCount);
				});
			}else{
				vm.statisticObjects.split(',').map(function(item){
					keys.push(item);
					seriesData.push(null);
				});
			}
			
			if(vm.timeArr.length) { // 显示timeline轴时
				// 迭代生成各系列options
				vm.timeArr.map(function(key,idx){
					var op = {};
					op.series = [{data: seriesData}];
					
					subOps.push(op);
				});
			}else { // 不显示timeline轴时（week粒度）
				subOps.push({series: [{data: seriesData}]});
			}
			
			// 主体options
			var chartOp = {
					baseOption: {
						timeline: { // 时间轴设置
							show: vm.timeArr.length > 0,
							data: vm.timeArr,
							label: {
								formatter: function(s){
									return s;
								}
							},
							currentIndex: vm.currentIndex,
							axisType: 'category',
							controlPosition: 'none',
							lineStyle: {color: '#aaa'}
						},
						color: ['#6498FE','#58D8D8','#2BDC72','#67B83F','#B9DE22','#FFC400','#F38937','#FD6A75','#8B8BF2','#AD60F8'],
						tooltip: {},
						grid: {bottom: 100,top: 80},
						xAxis: [{
							type: 'category', name: 'SN', axisLabel: {interval: 0,rotate: 354,fontSize: 10}, data: keys,
							axisLine: {lineStyle: {color: '#888'}}
						}],
						yAxis: [{
							type: 'value', name: 'Alarm Count',
							axisLine: {lineStyle: {color: '#888'}}
						}],
						itemStyle: { // 柱体圆角和颜色
							normal: {
								color: ['#7FC9FB'],
								barBorderRadius: [5,5,0,0]
							}
						},
						series: [{name: 'alarm',type: 'bar',barMaxWidth: 40}]
					},
					options: subOps
				};
			
			return chartOp;
		},
		/**
		* 生成时间data
		* @param type{string}  粒度类型 
		*/
		initTimeLineData(type){
			var vm = this;
			
			if(type == 'hourse') { // 小时粒度时，timeline轴值
				vm.getHourses();
			}
			else if(type == 'day') { // 天粒度时，timeline轴值
				vm.getDays();
			}else { // 周粒度时，timeline不显示
				vm.timeArr = [];
				var strdate = vm.initStartTime.substr(0,10) + '  -  ' + vm.initEndTime.substr(0,10);
				vm.zoomArr = [strdate];
			}
		},
		// 获取小时粒度的time Data
		getHourses() {
			var lineDate = [];
			for(var i = 0;i < 24;i++){
				//对24 取余 生成时间点 并格式化 01:00
				var formatterNumber = i%24;
				if(formatterNumber < 10){
					formatterNumber = "0"+formatterNumber+":00"
				}else{
					formatterNumber = formatterNumber+":00"
				}
				
				lineDate.push(formatterNumber);
			}
			this.timeArr = lineDate;
			this.zoomArr = lineDate;
		},
		// 获取天粒度的 time Data
		getDays() {
			var lineDays = [],
				start = this.initStartTime,
				end = this.initEndTime.substr(0,10)+' 00:00:00',
				num = differ(end,start);
			
			for(var i = 0; i<=num; i++) {
				var dateStr = dateformatter(addDate(start,i));
				lineDays.push(dateStr.substr(0,10));
			}
			
			this.timeArr = lineDays;
			this.zoomArr = lineDays;
		},
		/**
		* 查看图表时，依据模板参数初始化
		* @param type{number}  统计粒度 
		* @param startTime{string}   统计开始时间
		* @param endTime{string}   统计结束时间
		* @param code{number}   统计模板id
		* @param deviceType{number}   统计类型 
		*/
		initStatisticTime(type,startTime,endTime,code,deviceType) {
			var vm = this;
			vm.initStartTime = startTime;
			vm.initEndTime = endTime;  
			vm.statisticId = code;
			vm.dataRange = startTime;
			vm.deviceType = deviceType;
			
			if(type === 0) vm.statisticType = 'hourse';
			if(type == 1) vm.statisticType = 'day';
			if(type == 2) vm.statisticType = 'week';

			if(deviceType == 1) {
				vm.tburl = '${ctx}/cell/fault/queryDeviceGroupInfoById.action?statisticId=' + code;
				vm.echartSlideTitle = '<%=rb.getString("YiXuanSheBeiZu")%>';
			}
			if(deviceType === 0) {
				vm.tburl = '${ctx}/cell/fault/queryDeviceInfoById.action?statisticId=' + code + '&deviceType=' + deviceType;
				vm.echartSlideTitle = '<%=rb.getString("YiXuanSheBei")%>';
			}
			if(deviceType == 2) {
				vm.tburl = '${ctx}/cell/fault/queryIdentifierInfoById.action?statisticId=' + code;
				vm.echartSlideTitle = '<%=rb.getString("YiXuanGaoJing")%>';
			}
			
			vm.$nextTick(function(){
				vm.initEchart();
			})
		}
	},
	mounted(){
		eventBus.$off("open-echart").$on("open-echart",this.initStatisticTime);
	}
})
</script>
