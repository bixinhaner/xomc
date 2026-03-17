<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
.statiscPage {
	background:#f1f1f2;
	height:100%;
	overflow:auto;
	display:flex;
	flex-wrap:wrap;
}
.chartItem {
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:10px;
	height:35vh;
	min-height:300px;
	flex:1;
	min-width:45%;
	background:#FFF !important;
}
.kpiItem {
	flex:1;
	border:1px solid #d5dcec;
	border-radius:10px;
	min-height:350px;
	background:#FFF;
}
.kpiTitle {
	height:40px;
	line-height:40px;
	padding:0 20px;
	font-size:14px;
	border-bottom:1px solid #d5dcec;
}
.kpiMain {
	display:flex;
	height:350px;
}
.chartLeftBtnBoxCls {
	width:200px;
	height:100%;
	border-right:1px solid #d5dcec;
}
.chartRightMainBoxCls {
	flex:1 auto;
	height:100%;
}
.buttonDefaultCls ,.buttonSelectCls{
	height:30px;
	line-height:30px;
	padding:0 20px;
	border-bottom:1px solid #d5dcec;
}
</style>
<div id="cpeStatiscPage" class="statiscPage">
	<div class="kpiItem">
		<div class="kpiTitle">
			<div class="herderButtonBoxCls" style="width:fit-content;margin:2px auto;">
				<el-radio-group v-model='periodType' size="mini" class="commonRadioButton" @change="changeStatis" style='margin-top:10px;'>
	                <el-radio-button  label="day"><%=rb.getString("Tian")%></el-radio-button>
	                <el-radio-button  label="month"><%=rb.getString("Yue")%></el-radio-button>
	            </el-radio-group>
			</div>
		</div>
		<div class="kpiMain">
			<div class="chartLeftBtnBoxCls">
				<div v-for="(item,index) in kpiChartList" :class="historyChartType == item.id ? 'buttonSelectCls': 'buttonDefaultCls'" @click="historyChartTypeChange(item.id)">
					{{item.label}}
				</div> 
			</div>
			<div class="chartRightMainBoxCls">
				<div v-for="(item,index) in kpiChartList" v-show="historyChartType == item.id" :id="item.id" style="padding:10px;height:calc(100% - 50px);width: calc(100% - 100px);"></div>
				<div id="time_line_cht" style="width: calc(100% - 100px);height: 50px;display:inline-block;"></div>
			</div>
		</div>
	</div>
</div>


<script type="text/javascript">
var timeParam = getNowTimeToZoneTimeRange(timeZone, 144)
var lowVal = localStorage.getItem("rsrp1");
var highVal = localStorage.getItem("rsrp2");
var cpeCode = sessionStorage.getItem('CPE_CODE');
var globCpeMap,
	cpeDetailIMSI = sessionStorage.getItem('imsi');
var sn = sessionStorage.getItem('cpeSN');
var product = sessionStorage.getItem('PRODUCT');
var cpeStatiscVue = new Vue({
	el:'#cpeStatiscPage',
	data(){
		var vm = this,
			validDate = function(rule, value, cb) {
				if(vm.isDayStatis) {
					if(value && value.length) {
						cb();
					}else {
						cb('<%=rb.getString("QingXuanZeShiJian")%>');
					}
				}else {
					cb();
				}
			},
			validMonth = function(rule, value, cb) {
				var pre = vm.form.monthpre,
					suf = vm.form.monthsuf;
				
				if(!vm.isDayStatis && pre && suf) {
					if(new Date(pre).getTime() > new Date(suf).getTime()) {
						cb('<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>');
					}else {
						cb();
					}
				}else if(!vm.isDayStatis) {
					cb('<%=rb.getString("QingXuanZeShiJian")%>');
				}else {
					cb();
				}
			};
		return {
			periodType:'day',
			historyChartType:'UL_MCS',
			cpeChartPackType:'expand',
			isDayStatis:true,
			currentMonth: '',
			kpiChartList:[
				{id:'UL_MCS',label:'UL_MCS'},
				{id:'DL_MCS',label:'DL_MCS'},
				{id:'RSRP0',label:'RSRP0'},
				{id:'RSRP1',label:'RSRP1'},
				{id:'CINR0',label:'CINR0'},
				{id:'CINR1',label:'CINR1'},
				{id:'SINR',label:'SINR'},
				{id:'DL_RATE',label:'<%=rb.getString("CPEXiaXingTunTuLiang")%>'},
				{id:'UL_RATE',label:'<%=rb.getString("CPEShangXingTunTuLiang")%>'},
				{id:'PCI',label:'PCI'},
			],
			form: {
				timeRange: [],
				monthpre: '',
				monthsuf: ''
			},
			rules: {
				timeRange: [{validator: validDate}],
				monthpre: [{validator: validMonth}]
			},
			pickerOpts: {}
		}
	},
    computed: {
		
	},
	watch:{},
	methods:{
		// 初始化
		init(){
			var vm = this;
			vm.initHistory();
		},
		// 初始化 历史图表数据
		initHistory(){
			var vm = this;
			vm.getDataByDay();
			var timeLine = echarts.init(document.getElementById('time_line_cht'));
			vm.switchTimeScale(timeLine,'day');
			// 窗口缩放时自适应
			window.removeEventListener('resize',vm.resizeChart);
			window.addEventListener('resize',vm.resizeChart);
		},
		// 图表自适应
		resizeChart(){
			var list = ['UL_MCS', 'DL_MCS', 'RSRP0','RSRP1','CINR0','CINR1','SINR','DL_RATE','UL_RATE','PCI','time_line_cht'];
			
			list.map(function(id){
				var dom = document.querySelector('#'+id);
				if(dom) {
					var itn = echarts.getInstanceByDom(dom);
					itn && itn.resize();
				}
			})
		},
		
		
		// 时间粒度切换事件
		changeStatis(){
			var vm = this;
			
			if(vm.periodType == 'day') {
				vm.getDataByDay();
			}else {
				vm.getDataByMonth();
			}

			var chart = echarts.getInstanceByDom(document.querySelector('#time_line_cht')),
				type = vm.periodType;
			vm.switchTimeScale(chart, type);
		},
		// 生成时间轴
		switchTimeScale(chart, type) {
			var vm = this,
				timeData = [getYesterDay(6).substring(5).replace("-","."),
							getYesterDay(5).substring(5).replace("-","."),
							getYesterDay(4).substring(5).replace("-","."),
							getYesterDay(3).substring(5).replace("-","."),
							getYesterDay(2).substring(5).replace("-","."),
							getYesterDay(1).substring(5).replace("-","."),
							getYesterDay(0).substring(5).replace("-",".")],
				curIndex = 6;

			if(type=='month') {
				timeData = [];
				curIndex = 12;
				var nowDate = getNowZoneTime(timeZone),
					now = new Date(nowDate);
				for(var i=12; i>=0; i--) {
					var cur = new Date(nowDate);
					yearAgo = cur.setMonth(now.getMonth()-i);
					yearAgo = dateformatter(new Date(yearAgo));
					timeData.push(yearAgo.substring(0, 7));
				}
			}

			chart.setOption({
				timeline: {
					top: 0,
					axisType: 'category',
					controlPosition: 'none',
					symbolSize:8,
					lineStyle : { color : '#B0AFBA',width : 1 },
					itemStyle : {
						normal : { borderColor : '#B0AFBA' },
						emphasis : {
							borderColor : '#1e90ff',
							color : '#1e90ff'
						}
					},
					data: timeData,
					notMerge:true,
					currentIndex: curIndex,
					checkpointStyle:{
						color:'#209FFF',
						borderColor:'none'
					}
				}
			});

			chart.off('timelinechanged');
			chart.on('timelinechanged',function(param){
				vm.reloadHistoryChart(param.currentIndex,type);
			});
			vm.resizeChart();
		},
		// 时间线改变请求数据
		reloadHistoryChart(index,type) {
			var vm = this,
				params = {token: 7},
				prev_startTime = getYesterDay(7-index)+' 00:00:00';
				params.cpeCode = sessionStorage.getItem('CPE_CODE');
				params.timeZone = timeZone;
				params.start_time = formatDate(addDate(new Date(prev_startTime),1));
				params.end_time = formatDate(addDate(new Date(params.start_time), 1));
				params.product = product;

			$('#KPIDataGridDayTime').html(prev_startTime.substring(0,10));
				
			if(type == 'month') {
				var nowDate = getNowZoneTime(timeZone).substring(0,8)+'01 00:00:00',
					cur = new Date(nowDate);
					yearAgo = cur.setMonth(cur.getMonth()+index-12);
					yearAgo = dateformatter(new Date(yearAgo));
					currentMonth = yearAgo.substring(0, 7),
					year = currentMonth.split('-')[0]-0,
					month = currentMonth.split('-')[1]-0,
					start_time = currentMonth+'-01 00:00:00',
					prevTime = new Date(start_time),
					end_time = formatDate(new Date(prevTime.setMonth(prevTime.getMonth()+1)));
				params = {
						cpeCode : sessionStorage.getItem('CPE_CODE'),
						timeZone : timeZone,
						start_time : start_time,
						end_time : end_time,
						product : product,
						token: 'm'
				};
			}

			vm.loadCPEHistoryGraphData(params);
		},
		// 月粒度数据
		getDataByMonth(){
			var vm = this,
				year = vm.currentMonth.split('-')[0]-0,
				month = vm.currentMonth.split('-')[1]-0,
				start_time = vm.currentMonth+'-01 00:00:00',
				prevTime = new Date(start_time),
				end_time = formatDate(new Date(prevTime.setMonth(prevTime.getMonth()+1)));
			var params = {
		            cpeCode : sessionStorage.getItem('CPE_CODE'),
		            timeZone : timeZone,
		            start_time : start_time,
		            end_time : end_time,
		            product : product,
		            token: 'm'
		    };
		    vm.loadCPEHistoryGraphData(params);
		},
		// 天粒度数据
		getDataByDay(){
			var vm = this,
				curDay = getNowZoneTime(timeZone).substring(0,10),
				start_time = curDay+' 00:00:00',
				end_time = formatDate(addDate(new Date(start_time), 1));
		
			var params = {
		            cpeCode : sessionStorage.getItem('CPE_CODE'),
		            timeZone : timeZone,
		            start_time : start_time,
		            end_time : end_time,
		            product : product,
		            token: 7
		    };
		    vm.loadCPEHistoryGraphData(params);
		},
		// 请求图表数据
		loadCPEHistoryGraphData(params){
			var vm = this;
			axios.post('${ctx}/cell/CPE/getCPEHistoryGraphData.action',stringify(params)).then(function(response){
				let data = response.data
				if(data) {
					var Xdata = data.bak_time_data,
						token = params.token;
					
					var ulmcsData = vm.parseFloatData(data.ul_mcs_data);
					var dlmcsData = vm.parseFloatData(data.dl_mcs_data);
					var rsrp1Data = vm.parseFloatData(data.rsrp0_data);
					var rsrp2Data = vm.parseFloatData(data.rsrp1_data);
					var cinr1Data = vm.parseFloatData(data.cinr0_data);
					var cinr2Data = vm.parseFloatData(data.cinr1_data);
					var sinrData = vm.parseFloatData(data.cpe_sinr_data);
					var dlrateData = vm.parseFloatData(data.dl_current_datarate_data);
					var ulrateData = vm.parseFloatData(data.ul_current_datarate_data);
					var pciData = vm.parseFloatData(data.pci_data);

					//UL_mcs
					if(data.ul_mcs_flag=="noData"){
						vm.generateCharts("UL_MCS", "UL_MCS", [], [], "",token);
					}else if(data.ul_mcs_flag=="noSupport"){
						vm.generateCharts("UL_MCS", "UL_MCS", [], [], "",token);
					}else{
						vm.generateCharts("UL_MCS", "UL_MCS", Xdata, ulmcsData, "",token);
					}
					//DL_mcs
					if(data.dl_mcs_flag=="noData"){
						vm.generateCharts("DL_MCS", "DL_MCS", [], [], "",token);
					}else if(data.dl_mcs_flag=="noSupport"){
						vm.generateCharts("DL_MCS", "DL_MCS", [], [], "",token);
					}else{
						vm.generateCharts("DL_MCS", "DL_MCS", Xdata, dlmcsData, "",token);
					}
					//RSR1
					if(rsrp1Data.length == 0){
						vm.generateCharts("RSRP0", "RSRP1", [], [], "dBm",token);
					}else{
						vm.generateCharts("RSRP0", "RSRP1", Xdata, rsrp1Data, "dBm",token);
					}
					//RSRP2
					if(rsrp2Data.length == 0){
						vm.generateCharts("RSRP1", "RSRP2", [], [], "dBm",token);
					}else{
						vm.generateCharts("RSRP1", "RSRP2", Xdata, rsrp2Data, "dBm",token);
					}
					//CINR1
					if( cinr1Data.length == 0){
						vm.generateCharts("CINR0", "CINR1", [], [], "dB",token);
					}else{
						vm.generateCharts("CINR0", "CINR1", Xdata, cinr1Data, "dB",token);
					}
					//CINR2
					if(cinr2Data.length == 0){
						vm.generateCharts("CINR1", "CINR2", [], [], "dB",token);
					}else{
						vm.generateCharts("CINR1", "CINR2", Xdata, cinr2Data, "dB",token);
					}
					//SINR
					if(sinrData.length == 0){
						vm.generateCharts("SINR", "SINR", [], [], "dB",token);
					}else{
						vm.generateCharts("SINR", "SINR", Xdata, sinrData, "dB",token);
					}
					//下行速率
					if(dlrateData.length == 0){
						vm.generateCharts("DL_RATE", "<%=rb.getString("CPEXiaXingTunTuLiang")%>", [], [], "Mbps",token);
					}else{
						vm.generateCharts("DL_RATE", "<%=rb.getString("CPEXiaXingTunTuLiang")%>", Xdata, dlrateData, "Mbps",token);
					}
					//上行速率
					if(ulrateData.length == 0){
						vm.generateCharts("UL_RATE", "<%=rb.getString("CPEShangXingTunTuLiang")%>", [], [], "Mbps",token);
					}else{
						vm.generateCharts("UL_RATE", "<%=rb.getString("CPEShangXingTunTuLiang")%>", Xdata, ulrateData, "Mbps",token);
					}
					//PCI
					if(pciData.length == 0){
						vm.generateCharts("PCI", "PCI", [], [], "",token);
					}else{
						vm.generateCharts("PCI", "PCI", Xdata, pciData, "",token);
					}

					if(!vm.historyChartType){
						vm.historyChartType = vm.kpiChartList.length > 0 ? vm.kpiChartList[0].id : '';
					}
				}
			});
		},
		// 图表配置
		generateCharts(elemId, title, Xdata, Ydata, yFormatUnit,token) {
			var vm = this,
				myChart = echarts.init(document.getElementById(elemId)),
				xname = '<%=rb.getString("XiaoShi")%>';
				
			if(token == 'm') xname = '<%=rb.getString("Tian")%>';
				
			var option = {
				title : {
					text : title,
					textStyle : {
						color : '#333333',
						fontSize : 12,
						fontFamily : 'Microsoft YaHei'
					},
					left : 20
				},
				tooltip : {
					trigger : 'axis',
					formatter : function(val) {
						var time = "<div>" + $("#KPIDataGridDayTime").text().substring(0,5) + " "
								+ val[0].name + ":00" + "</div>";
						
						if(token=='m') time = "<div>" + val[0].name + "</div>";
						if (isNaN(val[0].data)) {
							var value = "<div>" + val[0].marker + val[0].seriesName
									+ ":-</div>";
						} else {
							var value = "<div>" + val[0].marker + val[0].seriesName
									+ ":" + val[0].data + "</div>";
						}
						return time + value;
					}
				},
				color : ['#90EC97','#E9A4A4'],
				grid:{
					top:50,
					left:30,
					right:60,
					bottom:20,
					containLabel:true
				},
				legend : {
					left : 350,
					top : 35,
					icon: 'circle',
					itemHeight: 10,
					itemStyle: {
						fontSize: 8
					},
					data : [{
								name : sn,
								textStyle : {
									color : '#85b1de'
								}
							}]
				},
				toolbox : {
					show : true,
					feature : {
						saveAsImage : {
							show : false
						}
					}
				},
				xAxis : [{
							name : xname,
							type : 'category',
							boundaryGap : false,
							axisLine : {
								show : true,
								lineStyle : {
									color : "#b6b6b6"
								}
							},
							axisLabel : {
								show : true,
								textStyle : {
									color : "#b6b6b6"
								},
								lineStyle : {
									color : "#b6b6b6"
								},
								formatter : function(val) {
									if(token=='m') {
										return val.substring(8,10);
									}else {
										if (val.split(" ")[1].substring(3, 5) == '00')
											return val.split(" ")[1].substring(0, 2);
										
									}
									return val;
								},
								interval : function(index) {
									if (index % 6 == 0 && index != 144) {
										return true;
									}
								}
							},
							data : Xdata
						}],
				yAxis : [{
							name: yFormatUnit ,
							type : 'value',
							axisLabel : {
								show : true,
								textStyle : {
									color : "#b6b6b6"
								},
								lineStyle : {
									color : "#b6b6b6"
								},
								formatter : '{value} ' 
							},
							axisLine : {
								lineStyle : {
									color : "#b6b6b6"
								}
							},
							splitLine: {
								show:true,
								lineStyle : {
									type:'dashed'
								}
							}
						}],
				series : [{
					name : sn,
					type : 'line',
					data : Ydata,
					symbolSize : 1,
					itemStyle: {
						normal: {
							areaStyle: {opacity: 0.2}
						}
					},
					areaStyle: {opacity: 0.2},
					showAllSymbol : true
					}]
			};
			// 为echarts对象加载数据
			myChart.setOption(option);
			vm.resizeChart();
		},
		// History 图表切换事件
		historyChartTypeChange(type){
			var vm = this;
			if(vm.historyChartType == type)return
			vm.historyChartType = type;
			vm.$nextTick(()=>{
				vm.resizeChart();
			})
		},
		// 图表展开和收起
		cpeChartPackChange(type){
			var vm = this;
			if(vm.cpeChartPackType == type)return
			vm.cpeChartPackType = type;
		},
		// 图表数据格式化
		parseFloatData(arr) {
			var retArr = [];
			var flag=false;
			if(arr && arr.length>0){
				for (var i = 0; i < arr.length; i++) {
					retArr[i] = parseFloat(arr[i]);
					if( arr[i]!="-" ){
						flag = true;
					}
				}
				if(!flag){
					retArr = [];
				}
			}
			return retArr;
		},
		// 导出时间限制
		initTime() {
			var vm = this,
				start = getYesterDay(0),
				end = getYesterDay(6);
			
			if(!vm.isDayStatis) {
				start = getNowZoneTime(timeZone);
				
				var now = new Date(start),
					yearAgo = now.setMonth(now.getMonth()-12);
				
				end = dateformatter(new Date(yearAgo));
			}
			
			vm.pickerOpts = {disabledDate: function(t){
									return t.getTime() > new Date(start).getTime()
										|| t.getTime() < new Date(end).getTime();
								}
							 }
		},
	
	},
	mounted(){
		var vm = this;
		vm.currentMonth = getNowZoneTime(timeZone).substring(0, 7);
		vm.init();
	}
})
</script> 