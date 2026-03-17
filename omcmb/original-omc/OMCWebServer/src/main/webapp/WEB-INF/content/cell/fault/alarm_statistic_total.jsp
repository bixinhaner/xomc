<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<!-- 告警图表开发页-->
<style>
	.chartConcent{
		position: relative;
	}
	.ChartTopTitle{
		height: 40px;
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
		height: 40px;
		line-height: 40px;
		text-align: center;
		font-weight: 550;
		cursor:pointer; 
	}
	.titleLeftSelect{
		border-bottom: 2px solid #4D84FF;
	}
	.titleRight{
		display: flex;
		flex: 1;
		align-items: center;
		justify-content: center;
	}
	.timeButtonLeft .el-button.is-circle [class^=el-icon-], .timeButtonRight .el-button.is-circle [class^=el-icon-]{
		color: var(--main-color);
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
		cursor: pointer;
		margin-left: 30px;
		border-bottom: 1px solid #FFFFFF;
	}
	.alarmMinor .el-icon:before{
		color: #FFDA41;
		font-size: 20px;
	}
	.alarmMajor .el-icon:before{
		color: #FF973E;
		font-size: 20px;
	}
	.alarmCritical .el-icon:before{
		color: #FC5959;
		font-size: 20px;
	}
	.alarmWarning .el-icon:before{
		color: #60BEFC;
		font-size: 20px;
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
<div id="alarmChartInfo">
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
		</div>
		<div class="ChartTopTitle">
			<div class="titleLeft">
				<div :class="topDataType == 'add'? 'titleLeftSelect': ''" @click="ChartTopTitleClick('add')"><%=rb.getString("XinZeng")%></div>
				<div :class="topDataType == 'clear'? 'titleLeftSelect': ''" @click="ChartTopTitleClick('clear')" style="margin-left:32px;"><%=rb.getString("QingChu")%></div>
			</div>
		</div>
		<div style="display:flex;flex-wrap: wrap;border-bottom: 1px	solid #E9E9E9;">
			<div :class="chartClass" style="min-width:780px;flex:1;border-right:1px solid #D5DCEC;">
				<div id="topBarChart" style="width:100%;height:358px;"></div>
			</div>
		</div>
		<div class="ChartTopTitle">
			<div class="titleLeft">
				<div :class="footDataType == 'active'? 'titleLeftSelect': ''" @click="ChartFootTitleClick('active')"><%=rb.getString("HuoDong")%></div>
				<div :class="footDataType == 'all'? 'titleLeftSelect': ''" @click="ChartFootTitleClick('all')" style="margin-left:32px;"><%=rb.getString("SuoYou")%></div>
			</div>
		</div>
		<div style="display:flex;flex-wrap: wrap;">
			<div :class="chartClass" style="min-width:780px;flex:1;border-right:1px solid #D5DCEC;">
				<div id="footBarChart" style="width:100%;height:358px;"></div>
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
			topDataType:'add', // 头部类型切换  新增-add  所有-all
			footDataType:'active', // 底部类型切换  活动-active  清除-clear
			timeButtonRightDisabled:true, 
			chartList: ['topBarChart','footBarChart'],
			charts: {},
			
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

			vm.proccessTopBarChart('topBarChart');
			vm.proccessFootBarChart('footBarChart');
		},
		chartMonthTimes(val){
			var vm = this;
			if(vm.chartTimesType == 'day')return
			var endDate = new Date(val+'-01');

			if(val == formatDate(Date.getNow()).slice(0,7)){
				vm.timeButtonRightDisabled = true;
			}else{
				vm.timeButtonRightDisabled = false;
			}
			var EndTime = new Date(addMonth(endDate,1)+'-01');
			vm.chartMonthTimesAfter = formatDate(EndTime);

			vm.proccessTopBarChart('topBarChart');
			vm.proccessFootBarChart('footBarChart');
		}
	},
	
	methods:{
		init(){
			var vm = this;
			var endDate = new Date(vm.chartDayTimes);
			vm.chartDayTimesAfter = formatDate(addDate(endDate,1));
			vm.chartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
					if(code == 'topBarChart'){
						vm.charts[code].on('mouseover',(params)=>{
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
						
					}else if(code == 'footBarChart'){
						vm.charts[code].on('mouseover',(params)=>{
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
				}
				
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);

			});
			
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
		// 日  月切换事件
		dayAndMonthClick(type){
			var vm = this;
			vm.chartTimesType = type;
			if(vm.chartTimesType == 'day'){
				vm.chartDayTimes = formatDate(Date.getNow()).slice(0,10);
			}else{
				var endDate = new Date(vm.chartMonthTimes+'-01');
				vm.chartMonthTimes = formatDate(Date.getNow()).slice(0,7);
				vm.timeButtonRightDisabled = true;
			}
			vm.proccessTopBarChart('topBarChart');
			vm.proccessFootBarChart('footBarChart');
		},
		//  新增 所有 切换事件
		ChartTopTitleClick(type){
			var vm = this;
			vm.topDataType = type;
			vm.proccessTopBarChart('topBarChart');
		},
		//  活动 历史 切换事件
		ChartFootTitleClick(type){
			var vm = this;
			vm.footDataType = type;
			vm.proccessFootBarChart('footBarChart');
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
		//判断日期是否是指定月份最后一天
		isLastDay(data){
			var d = new Date(data.replace(/\-/,"/"));
			var nd = new Date(d.getTime()+24*60*60*1000);
			return (d.getMonth() != nd.getMonth())
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

				if(['footBarChart'].includes(code)) {
					vm.proccessFootBarChart(code);// 底部柱状图表统计数据刷新
				}
			}
		},
		// 头部柱状图表统计数据刷新
		proccessTopBarChart(code){
			var vm = this,params = {};
		
			params.templateId = '';
			params.timeZone = timeZone;
			params.statisticObject = 'Serverity';
			if(vm.topDataType == 'add'){
				params.statisticType = 'ActiveIncr';
			}else{
				params.statisticType = 'ClearIncr';
			}
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
		// 底部柱状图表统计数据刷新
		proccessFootBarChart(code){
			var vm = this,params = {};
		
			params.templateId = '';
			params.timeZone = timeZone;
			params.statisticObject = 'Serverity';
			if(vm.footDataType == 'active'){
				params.statisticType = 'ActiveTotal';
			}else{
				params.statisticType = 'Total';
			}
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
		// 生成图表的option 
		createOption(data,code){
			var vm = this,optionData = data;
			var codes = {
				'topBarChart':vm.createTopBarChartOption,
				'footBarChart':vm.createFootBarChartOption,
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
				
				for(var i=0;i<24;i++){
					var times;
					if(i<10){
						if(i!=9){
							times = vm.chartDayTimes +' 0'+ i + ':00:00' +'-'+'0'+ (i+1)+':00:00'
						}else{
							times = vm.chartDayTimes +' 0'+ i + ':00:00' +'-'+ (i+1)+':00:00'
						}
					}else{
						times = vm.chartDayTimes +' '+ i + ':00:00'+'-'+ (i+1)+':00:00'
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
					var times = '',
						xAxisStr = '';
					if(i<10){
						times = vm.chartMonthTimes +'-0'+ i;
					}else{
						times = vm.chartMonthTimes+'-' + i;
					}
					xAxisStr = times + ' 00:00:00' + ' ~ ' + formatDate(addDate(times,1))
					xAxisData.push(xAxisStr);
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
				if(vm.chartTimesType == 'day'){
					var idx = parseInt(item.startTime.split(' ')[1].substring(0,2),10);
				}else{
					var idx = parseInt(item.startTime.split('-')[2],10)-1;
				}
				criticalData[idx] = item.critical;
				majorData[idx] = item.major;
				minorData[idx] = item.minor;
				warningData[idx] = item.warning;
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
						str+=param[0].name+"<br>";
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
								var secondTime = val.split('-')[3],
									clock = '';
								if(secondTime) clock = secondTime.substring(0,2);

								return clock;
							}else{
								var secondTime = val.split('-')[2].slice(0,2);
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
			vm.resizeChart();
		},
		// 创建底部柱状图Option
		createFootBarChartOption(data,code){
			var vm = this,legendName = ['<%=rb.getString("GaoJingShu")%>'],
				lineName = '<%=rb.getString("GaoJingShu")%>',
				xAxisName,xAxisData,optionData = data,
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
				criticalData[idx] = item.critical;
				majorData[idx] = item.major;
				minorData[idx] = item.minor;
				warningData[idx] = item.warning;
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
						str+=param[0].name+"<br>";
						str+= '<span style="display:inline-block;margin-right:5px;border-radius:10px;width:9px;height:9px;background-color:#69B6FC;"></span>'+'<%=rb.getString("ShouYe_ZongShu")%>: '+totalNum+"<br>";
						str+=param[0].marker+param[0].seriesName+":"+param[0].value+"<br>";
						str+=param[1].marker+param[1].seriesName+":"+param[1].value+"<br>";
						str+=param[2].marker+param[2].seriesName+":"+param[2].value+"<br>";
						str+=param[3].marker+param[3].seriesName+":"+param[3].value+"<br>";
						return str
					},
					extraCssText:"z-index:1000"
				},
				legend:{
					data:['Critical','Major','Minor','Warning'],
					bottom:10,
					selectedMode:false,
					itemHeight:8,
					itemWidth:8
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
								var secondTime = val.split(' ')[1],
									clock = '';
								if(secondTime) clock = secondTime.slice(0,2);
									
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
			vm.resizeChart();
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
		this.init();
	},
})

</script>