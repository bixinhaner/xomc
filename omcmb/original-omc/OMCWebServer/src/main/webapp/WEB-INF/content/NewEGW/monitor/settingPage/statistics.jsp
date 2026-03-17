<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwStatisticPage{
	width: 100%;
	display:flex;
	flex-direction:column;
}
#egwStatisticPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	margin-bottom: 10px;
	border-radius:10px;
	background:#fff;
	height:fit-content;
	overflow: hidden;
}
#egwStatisticPage .itemMainBoxTitle {
	height:36px;
	font-size:14px;
	line-height:36px;
	padding-left:20px;
	font-weight:bold;
    position: relative;
	border-bottom:1px solid #E9E9E9;
}
#egwStatisticPage .chartItemBoxCls {
	display:flex;
	flex-wrap:wrap;
}
#egwStatisticPage .chartItemBoxCls .chartItem{
	flex: 1;
}
#egwStatisticPage .chartBoxCls{
	display: flex;
	flex-wrap: wrap;
}
#egwStatisticPage .flex-col-chart .el-card, #egwStatisticPage .flex-col-chart{
	overflow: unset;
}
#egwStatisticPage .flex-col-chart {
	position: relative;
	flex: 1 auto;
	overflow: hidden;
}
#egwStatisticPage .half-persent {
	flex: 1;
	background-color: #fff;
	min-height: 240px;
	min-width:400px;
}
#egwStatisticPage .chartBoxCls .el-card__body{
	overflow: hidden;
}
#egwStatisticPage .chartBoxCls .el-card__footer{
	display: none;
}
#egwStatisticPage .performanceChartTimeLineBox{
	height: 50px;
	width: 100%;
	position: relative;
	border-bottom:1px solid #E9E9E9;
	display: flex;
	align-items: center;
	justify-content: center;
}
#egwStatisticPage .performanceChartMainBoxCls {
	display:flex;
	height:350px;
}
#egwStatisticPage .chartLeftBtnBoxCls {
	width:400px;
	height:100%;
	border-right:1px solid #d5dcec;
}
#egwStatisticPage .chartLeftBtnBoxCls > div{
	height: 26px;
	padding: 0px 10px;
	box-sizing: border-box;
	border-radius: 2px;
	font-size: 12px;
	font-weight: 500;
	display: flex;
	align-items: center;
	cursor: pointer;
}
#egwStatisticPage .chartLeftBtnBoxCls .buttonDefaultCls{
	border-bottom:1px solid #E9E9E9;
	color: #363B4E;
}
#egwStatisticPage .chartLeftBtnBoxCls .buttonSelectCls{
	border:1px solid var(--main-color)!important;
	color: var(--main-color);
}
#egwStatisticPage .chartRightMainBoxCls {
	flex:1 auto;
	height:100%;
}
</style>
<div id="egwStatisticPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">UE Count</div>
		<div class="chartBoxCls">
            <div :class="chartClass">
                <el-card shadow="hover" class="chart-card">
                    <div id="ueCountChart" style="width: 100%;height: 270px;"></div>
                </el-card>
            </div>
		</div>
	</div>
	<div class="chartItemBoxCls">
		<div class="chartItem itemMainBoxCls" style="margin-right:10px;">
			<div class="itemMainBoxTitle">
				Uplink Traffic
				<div style="position:absolute;right:5px;top:5px;">
					<el-radio-group size="mini" v-model='upLinkUnitType' class="commonRadioButton" @change="upLinkUnitTypeChange">
						<el-radio-button label="MB">MB</el-radio-button>
						<el-radio-button label="GB">GB</el-radio-button>
					</el-radio-group>
				</div>
			</div>
			<div class="chartBoxCls">
				<div :class="chartClass">
					<el-card shadow="hover" class="chart-card" style="position:relative;">
						<div id="upLinkChart" style="width: 100%;height: 270px;" ></div>
					</el-card>
				</div>
			</div>
		</div>
		<div class="chartItem itemMainBoxCls">
			<div class="itemMainBoxTitle">
				Downlink Traffic
				<div style="position:absolute;right:5px;top:5px;">
					<el-radio-group size="mini" v-model='downLinkUnitType' class="commonRadioButton" @change="downLinkUnitTypeChange">
						<el-radio-button label="MB">MB</el-radio-button>
						<el-radio-button label="GB">GB</el-radio-button>
					</el-radio-group>
				</div>
			</div>
			<div class="chartBoxCls">
				<div :class="chartClass">
					<el-card shadow="hover" class="chart-card" style="position:relative;">
						<div id="downLinkChart" style="width: 100%;height: 270px;" ></div>
					</el-card>
				</div>
			</div>
		</div>
	</div>
	<!--<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle"><%=rb.getString("XingNengGuanLi")%></div>
		<div class="chartBoxCls">
			<div class="performanceChartTimeLineBox">
				<div id="time_line_chart" style="width: 70%;min-width:140px;height: 45px;display:inline-block;"></div>
				<div style="position:absolute;right:20px;">
					<el-radio-group v-model='grainType' size="mini" class="commonRadioButton" @change="grainTypeChange">
		                <el-radio-button  label="15">15min</el-radio-button>
		                <el-radio-button  label="60">60min</el-radio-button>
		            </el-radio-group>
				</div>
			</div>
			<div class="performanceChartMainBoxCls">
				<div class="chartLeftBtnBoxCls">
					<div v-for="(item,index) in kpiChartList" :class="kpiChartType == item.id ? 'buttonSelectCls': 'buttonDefaultCls'" @click="kpiChartTypeChange(item.id)">
						{{item.label}}
					</div>
				</div>
				<div class="chartRightMainBoxCls">
					<div v-for="(item,index) in kpiChartList" v-show="kpiChartType == item.id" :id="item.id" style="padding:10px;height:100%;width: 100%;background-color:#20A5F9;"></div>
				</div>
			</div>
		</div>
	</div>-->

</div>

<script>
var timeParam = getNowTimeToZoneTimeRange(timeZone, 144);
var start_time_enb = timeParam.start_time.substring(0,11)+"00:00:00";
var end_time_enb = timeParam.end_time;
var egwStatisticPage = new Vue({
	el: '#egwStatisticPage', 
	data() {
		var vm = this;
		return {
			egwCode:'',
			rowData:{
				egwSn:'',
				egwName:'',
				generation:'',
				egwIp:'',
				egwPort:'',
				enbNumber:'',
				ueCount:'',
				uplink_traffic:'',
				downlink_traffic:'',
				ha_status:'',
				groupName:''
			},
			chartList: ['ueCountChart','upLinkChart','downLinkChart'],
			charts: {},
            currenUpLinkIndex:6,
            currenDownLinkIndex:6,
            upLinkUnitType:'MB',
            downLinkUnitType:'MB',

			grainType:'15',
			currentPMIndex:6,
			kpiChartType:'',
			kpiChartList:[],
		};
	},
	computed: {
		// 图表样式
		chartClass(){
			return {
				'flex-col-chart': true,
				'half-persent': true
			};
		},
	},
	methods: {
		init(row,code,sn,status){
			var vm =this;
			vm.egwCode = code;

			Object.keys(vm.rowData).map((key)=>{
				vm.rowData[key] = row[key]
			});
			vm.chartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
					
					// 初始化时间轴切换事件
                    vm.charts[code].on('timelinechanged',function(p){
                        vm.reloadChart(code,p.currentIndex);
                        if(code == 'upLinkChart'){
                            vm.currenUpLinkIndex = p.currentIndex;
                        }else if(code == 'downLinkChart'){
                            vm.currenDownLinkIndex = p.currentIndex;
                        }
                    });
				}
				
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);

				vm.reloadChart(code,6);
			});
			// vm.initKPI();		
		},
		// 所有图表自适应
		resizeChart(){
			var vm = this;
			vm.chartList.map(function(code){
				if(vm.charts[code]) vm.charts[code].resize();
			});
		},
		// 所有图表自适应
		// resizeChart(){
		// 	var list = ['ueCountChart', 'upLinkChart', 'downLinkChart','time_line_chart','echart1','echart2','echart3','echart4','echart5','echart6','echart7','echart8','echart9'];
			
		// 	list.map(function(id){
		// 		var dom = document.querySelector('#'+id);
		// 		if(dom) {
		// 			var itn = echarts.getInstanceByDom(dom);
		// 			itn && itn.resize();
		// 		}
		// 	})
		// },
		/**
		* 指定图表数据刷新
		* @param code{string} 图表类型
		* @param index{number}  时间轴下标
		*/
		reloadChart(code,index){
			var vm = this;
			
			if(['ueCountChart','upLinkChart','downLinkChart'].includes(code)) {
                vm.proccessChartData(code,index);// 统计数据刷新
            }
		},
		/**
        * 统计数据刷新 UE数 上下行流量
        * @param code{string} 图表类型
        * @param index{number}  时间轴下标
        */
        proccessChartData(code,index){
            var vm = this,
                s_time = getYesterDay(6-index).substring(0,10)+' 00:00:00',
                e_time = getYesterDay(6-index).substring(0,10)+' 23:59:59';
            
            if(index==6) {
                e_time = end_time;
            }else{
                e_time = getYesterDay(6-index-1).substring(0,10)+' 00:00:00';
            }
            var chartData={
                legendNames: [],
                data: [],
                yAxisName: '',
                xAxisData:[],
                color:[],
                pointerCount: 6,
                index: index,
                title: ''
            }
            for(var axisIndex=0; axisIndex < 6*24+1; axisIndex++){
                var yaxisStartTime = getYesterDay(6-index).substring(0,10)+ ' 00:00:00',
                    offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*(60/chartData.pointerCount));
                chartData.data.push('-');
                chartData.xAxisData.push(formatDate(offsetTime));
            }
            var params = {
                start_time : s_time,
                end_time : e_time,
                timeZone : timeZone,
                time_level: "min",
                device_type:'egw',
                device_code:vm.egwCode
            };
            if(code == 'ueCountChart'){
                chartData.legendNames = ['UE'];
                chartData.yAxisName = '('+'<%=rb.getString("ueGe")%>' + ')';
                chartData.title = '';
                chartData.color = ['#69B6FC'];
            }
            if(code == 'upLinkChart'){
                chartData.legendNames = ['UPLink'];
                if(vm.upLinkUnitType == 'MB'){
                    chartData.yAxisName = '(MB)';
                }else{
                    chartData.yAxisName = '(GB)';
                }
                chartData.title = '';
                chartData.color = ['#90EC97'];
            }
            if(code == 'downLinkChart'){
                chartData.legendNames = ['DownLink'];
                if(vm.downLinkUnitType == 'MB'){
                    chartData.yAxisName = '(MB)';
                }else{
                    chartData.yAxisName = '(GB)';
                }
                chartData.title = '';
                chartData.color = ['#90EC97'];
            }
            // 图标数据
            axios.post("${ctx}/system/device/getDeviceStatusDataList.action",stringify(params)).then(function(response){
                var data = response.data;

                if(data && data.length>0){
                    data.map((item)=>{
                        if(chartData.xAxisData.indexOf(item.statistics_time) != -1){
                            var ids = chartData.xAxisData.indexOf(item.statistics_time);
                            if(code == 'ueCountChart'){
                                chartData.data[ids] = item.ueCount;
                            }else if(code == 'upLinkChart'){
                                if(vm.upLinkUnitType == 'MB'){
                                    chartData.data[ids] = item.uplink_traffic;
                                }else{
                                    chartData.data[ids] = Number((item.uplink_traffic/1024).toFixed(1));
                                }
                            }else if(code == 'downLinkChart'){
                                if(vm.downLinkUnitType == 'MB'){
                                    chartData.data[ids] = item.downlink_traffic;
                                }else{
                                    chartData.data[ids] = Number((item.downlink_traffic/1024).toFixed(1));
                                }
                            }
                        }	
                    })
                }
                vm.charts[code].setOption(vm.createOption(chartData));
            }) 
            
            
        },
		// 生成option
		createOption(chartData){
			var vm = this,
				pointerCount = chartData.pointerCount,
				option = {
					baseOption: {
						title: {
							subtext: chartData.title,
							top: -5,
							left:20,
							subtextStyle: {
								color: '#333'
							}
						},
						timeline: {
							axisType: 'category',
							controlPosition: 'none',
							symbolSize:8,
							lineStyle : { color : '#666666',width : 1 },
							itemStyle : {
								normal : { borderColor : '#B0AFBA' },
								emphasis : {
									borderColor : '#1e90ff',
									color : '#1e90ff'
								}
							},
							data: [getYesterDay(6).substring(5).replace("-","."),
								getYesterDay(5).substring(5).replace("-","."),
								getYesterDay(4).substring(5).replace("-","."),
								getYesterDay(3).substring(5).replace("-","."),
								getYesterDay(2).substring(5).replace("-","."),
								getYesterDay(1).substring(5).replace("-","."),
								getYesterDay(0).substring(5).replace("-",".")],
							notMerge:true,
							currentIndex: chartData.index,
							checkpointStyle:{
								color:'#209FFF',
								borderColor:'none'
							}
						},
						tooltip : {
							trigger : 'axis',
						},
						grid:{
							bottom: 70,
							left:10,
							right: 60,
							containLabel:true
						}, 
						color: chartData.color,
						legend: { 
							data: chartData.legendNames,
							top:10
						},
						xAxis : [{
							name : '<%=rb.getString("XiaoShi")%>',
							type : 'category',
							boundaryGap : false,
							axisLine:{
								show : true,
								lineStyle:{ color:"#666666" }
							},
							axisLabel : {
								show:true,
								textStyle:{ color:"#666666" },
								lineStyle:{ color:"#666666" },
								formatter : function(val) {
									var secondTime = val.split(' ')[1];
									clock = secondTime.substring(0,2);
									val = secondTime.substring(0,5);
									if(secondTime.substring(3,5)=='00') return clock;
									return val;
								},
								interval : function(index){
									if(index%pointerCount == 0 && index != pointerCount*24){
										return true;
									}
								},
								rotate : (function(){
										var degree = 0;
										return degree;
								})(),
							}
						}],
						yAxis : [{
							name: chartData.yAxisName,
							minInterval: null,
							type: 'value',
							axisLabel : {
								show:true,
								textStyle:{ color:"#666666" },
								lineStyle:{ color:"#666666" }
							},
							axisLine:{
								lineStyle:{ color:"#666666" }
							},
							splitLine : {
								lineStyle:{ color:"#f1f1f4" }
							}
						}],
						series : [
							{  
								name: chartData.legendNames[0],
								type: 'line',
								symbolSize: 1,
								showAllSymbol: true,
								step: false,
								connectNulls: true,
								itemStyle: {
									normal: {
										areaStyle: {opacity: 0.2}
									}
								},
								areaStyle: {opacity: 0.2},
							}
						]
					},
					options: [
						{  
							xAxis: [{ data: chartData.xAxisData}],
							series: [
								{data:chartData.data}, 
							]  
						},
						{  
							xAxis: [{ data: chartData.xAxisData}],
							series: [
								{data:chartData.data}, 
							]  
						},
						{  
							xAxis: [{ data: chartData.xAxisData}],
							series: [
								{data:chartData.data}, 
							]  
						},
						{  
							xAxis: [{ data: chartData.xAxisData}],
							series: [
								{data:chartData.data}, 
							]  
						},
						{  
							xAxis: [{ data: chartData.xAxisData}],
							series: [
								{data:chartData.data}, 
							]  
						},
						{  
							xAxis: [{ data: chartData.xAxisData}],
							series: [
								{data:chartData.data}, 
							]  
						},
						{  
							xAxis: [{ data: chartData.xAxisData}],
							series: [
								{data:chartData.data}, 
							]  
						},
					]
				};
			return option;
		},
		// 上行流量单位切换
        upLinkUnitTypeChange(){
            var vm = this;
            vm.proccessChartData('upLinkChart',vm.currenUpLinkIndex);
        },
        // 下行流量单位切换
        downLinkUnitTypeChange(){
            var vm = this;
            vm.proccessChartData('downLinkChart',vm.currenDownLinkIndex);
        },
		// KPI粒度切换事件
		grainTypeChange(type){
			var vm = this;
			if(vm.grainType == type)return
			vm.grainType = type;
			vm.reloadPMChart(vm.currentPMIndex);
		},
		// KPI初始化
		initKPI(){ 
			var vm = this,
				timeLine = echarts.init(document.getElementById('time_line_chart'));
			vm.initTimeLine(timeLine);
			vm.reloadPMChart(6);
		},
		// 生成 KPI 时间轴
		initTimeLine(chart) {
			var vm = this;
			var timeData = [getYesterDay(6).substring(5).replace("-","."),
							getYesterDay(5).substring(5).replace("-","."),
							getYesterDay(4).substring(5).replace("-","."),
							getYesterDay(3).substring(5).replace("-","."),
							getYesterDay(2).substring(5).replace("-","."),
							getYesterDay(1).substring(5).replace("-","."),
							getYesterDay(0).substring(5).replace("-",".")];

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
					progress:{
						itemStyle:{
							color:'#fff',
							borderColor : '#B0AFBA'
						},
						lineStyle:{
							color:'#666'
						}
					},
					data: timeData,
					notMerge:true,
					currentIndex: 6,
					checkpointStyle:{
						color:'#209FFF',
						borderColor:'none'
					}
				}
			});

			chart.off('timelinechanged');
			chart.on('timelinechanged',function(param){
				vm.currentPMIndex = param.currentIndex;
				vm.reloadPMChart(param.currentIndex);
			});
			vm.resizeChart();
		},
		// KPI 图表切换事件
		kpiChartTypeChange(type){
			var vm = this;
			if(vm.kpiChartType == type)return
			vm.kpiChartType = type;
			vm.$nextTick(()=>{
				vm.resizeChart();
			})
		},
		// 根据时间轴日期更新性能图表
		reloadPMChart(index) {
			var vm = this,
				prev_startTime = getYesterDay(7-index)+' 00:00:00',
				start_time = formatDate(addDate(new Date(prev_startTime),1));

			vm.creatKpiEchart(start_time);
		},
		// 查询性能图表数据
		creatKpiEchart(queryStartTime){
			var vm = this,
				enb_code = vm.code,
				url = "${ctx}/pm/template/queryStatisKPIChartDataForInformation.action",
				query_start_time_glob = queryStartTime ? queryStartTime : formatDate(new Date(end_time_enb)).substring(0,10)+' 00:00:00',
				dataStartTime = query_start_time_glob,
				dataEndTime = formatDate(addDate(new Date(query_start_time_glob),1));;

			if(queryStartTime == formatDate(new Date(end_time_enb)).substring(0,10)+' 00:00:00'){
				dataEndTime = end_time_enb;
			}
			var params = { 
					timeZone : timeZone,
					statisPeriod : vm.grainType,
					startTime : dataStartTime,
					endTime : dataEndTime,
					enbCode : enb_code
				};
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data,
					chartCodes = [],
					kpiChartList = [];
				if(data && data.length) {// 检测那些图表需要绘制
					for(var key in data[0]) chartCodes.push(key);
				}
				if(chartCodes.includes('UL_Throughput')){
					kpiChartList.push({
										id:'echart1',label:'<%=rb.getString("ZhiBiao1") %>',chart_perf_codes:'UL_Throughput,DL_Throughput',
										chart_show_names:'UL,DL',units:'Mbps'
									})
				}
				if(chartCodes.includes('UL_PRB_utilization_rate')) {
					kpiChartList.push({
										id:'echart2',label:'<%=rb.getString("ZhiBiao2") %>',chart_perf_codes:'UL_PRB_utilization_rate,DL_PRB_utilization_rate',
										chart_show_names:'UL,DL',units:'%'
									})
				}
				if(chartCodes.includes('ERAB_setup_success_rate')) {
					kpiChartList.push({
										id:'echart3',label:'<%=rb.getString("ZhiBiao3") %>',chart_perf_codes:'ERAB_setup_success_rate',
										chart_show_names:'<%=rb.getString("ZhiBiao3")%>',units:'%'
									})
				}
				if(chartCodes.includes('ERAB_drop_rate')) {
					kpiChartList.push({
										id:'echart4',label:'<%=rb.getString("ZhiBiao4") %>',chart_perf_codes:'ERAB_drop_rate',
										chart_show_names:'<%=rb.getString("ZhiBiao4")%>',units:'%'
									})
				}
				if(chartCodes.includes('HO_InterEnbOutSucc_Rate_S1')) {
					kpiChartList.push({
										id:'echart5',label:'<%=rb.getString("ZhiBiao5") %>',chart_perf_codes:'HO_InterEnbOutSucc_Rate_S1',
										chart_show_names:'<%=rb.getString("ZhiBiao5")%>',units:'%'
									})
				}
				if(chartCodes.includes('HO_InterEnbOutSucc_RateX2')) {
					kpiChartList.push({
										id:'echart6',label:'<%=rb.getString("ZhiBiao6") %>',chart_perf_codes:'HO_InterEnbOutSucc_RateX2',
										chart_show_names:'<%=rb.getString("ZhiBiao6")%>',units:'%'
									})
				}
				if(chartCodes.includes('HO_InterEnbOutSucc_Rate')) {
					kpiChartList.push({
										id:'echart7',label:'<%=rb.getString("ZhiBiao7") %>',chart_perf_codes:'HO_InterEnbOutSucc_Rate',
										chart_show_names:'<%=rb.getString("ZhiBiao7")%>',units:'%'
									})
				}
				if(chartCodes.includes('RRC_setup_success_rate')) {
					kpiChartList.push({
										id:'echart8',label:'<%=rb.getString("ZhiBiao8") %>',chart_perf_codes:'RRC_setup_success_rate',
										chart_show_names:'<%=rb.getString("ZhiBiao8")%>',units:'%'
									})
				}
				if(chartCodes.includes('UP_BLER')) {
					kpiChartList.push({
										id:'echart9',label:'<%=rb.getString("ZhiBiao10") %>',chart_perf_codes:'UP_BLER,DOWN_BLER',
										chart_show_names:'UL,DL',units:'%'
									})
				}
				vm.kpiChartList = kpiChartList;
				if(!vm.kpiChartType){
					vm.kpiChartType = vm.kpiChartList.length > 0 ? vm.kpiChartList[0].id : '';
				}
				vm.$nextTick(()=>{
					vm.kpiChartList.map((item,index)=>{
						vm.goSetChartData(item.id, item.chart_perf_codes, item.chart_show_names, params, item.units, data, queryStartTime);
					})
					vm.resizeChart();
				})
			})
		},
		/**
		* 设置图表绘制数据
		* @param chart_id{string}：图表容器Id
		* @param codes{string}：图表code
		* @param show_names{string}：legend数据
		* @param param{object}：过滤参数
		* @param unit{string}：单位
		* @param chart_data{array}：图表数据
		* @param query_start_time{string}：图表统计的开始时间
		**/
		goSetChartData(chart_id, codes, show_names, param, unit, chart_data, query_start_time) {
			var vm = this,
				legend_name = codes.split(","),
				legend_data = [];

			if (show_names != "") {
				legend_name = show_names.split(",");
				for (var i = 0; i < legend_name.length; i++) {
					legend_data.push({
						name : legend_name[i],
						textStyle :{color:(legend_name[i] == "UL" ? "#85b1de" : "#cc6670")} 
					});
				}
			}
				
			var timesDataArr = [];
			var timesCount = 24*(60/vm.grainType)+1; // 6*(60/15) hours
			// 统计返回数据中包含的时间点
			if(vm.getObjLength(chart_data)>0 && vm.grainType == '60'){
				var startMin = " 00:" + chart_data[0].start_time.split(" ")[1].split(":")[1] + ":00";
				startTimeStr = query_start_time.substring(0,10) + startMin;
			}else{
				startTimeStr = query_start_time;
			}
			for(var timesIndex = 0; timesIndex<timesCount; timesIndex++ ){
				timesDataArr.push(formatDate(addTimes(new Date(startTimeStr),timesIndex*vm.grainType)));
			}
			var series_data = [];
			var cellCodeArr = codes.split(",");
			for (var code_index = 0; code_index < cellCodeArr.length; code_index++) {
				var seriesEle_data = [];
				/* new yaxis arr */
				for(var timesIndex = 0; timesIndex<timesCount; timesIndex++ ){
					seriesEle_data.push('-');
				}
				if (chart_data && chart_data != null && chart_data.length > 0) {
					var code_value = cellCodeArr[code_index];
					// 性能指标的值，是按开始时间从大到小排列的
					for (var time_index = 0; time_index < seriesEle_data.length; time_index++) {
						for (var data_index = 0; data_index < chart_data.length; data_index++) {
							var perf_data_obj = chart_data[data_index];
							var start_time = perf_data_obj.start_time;
							var end_time = perf_data_obj.end_time;
							if (timesDataArr[time_index] == start_time ) {
								var perfValue = perf_data_obj[code_value];
								if (perfValue != null && perfValue != 'N/A') {
									var differTimes = new Date(start_time).getTime() - new Date(startTimeStr).getTime();
									var yaxisIndex = Math.round(differTimes/(1000*60*vm.grainType));
									if(chart_id == 'echart1'){
										seriesEle_data.splice(yaxisIndex,1,(Number(perfValue)/1000).toFixed(2));
									}else{
										seriesEle_data.splice(yaxisIndex,1,Number(perfValue));
									}
									break;
								} 
							}
						}
					}
				}
				var seriesEle = {
					name : legend_name[code_index],
					type : 'line',
					data : seriesEle_data,
					symbolSize : 1,
					showAllSymbol:true
				};
				series_data.push(seriesEle);
			}
			
			var kpi_chart_data = {
					legend_data : {
						is_show : legend_data.length>1,
						data : legend_data
					},
					x_data : timesDataArr,
					series_data : series_data
				};
			vm.goCreateChart(chart_id, kpi_chart_data, unit);
		},
		/**
		* 获取对象长度值
		* @param obj{object}：获取长度的对象
		**/
		getObjLength(obj){
			var count=0;
			for(var key in obj){
				if(typeof obj[key] != 'function'){
					count += obj[key].length;
				}
			}
			return count;
		},
		/**
		* 创建KPI图表
		* @param elementId{string}：图表容器Id
		* @param chart_data{array}：图表绘制的数据
		* @param unit{string}：单位
		**/
		goCreateChart(elementId, chart_data, unit) {
			var vm = this,
				kpiChart = echarts.init(document.getElementById(elementId));
				option = {
					tooltip : {
						trigger : 'axis'
					},
					color:['#85b1de','#e9a4a4'],
					legend : {
						show : chart_data.legend_data.is_show,
						data : chart_data.legend_data.data
					},
					grid: {
						left: 20,
						containLabel:true
					},
					xAxis : [{
						name:'<%=rb.getString("XiaoShi") %>',
						type : 'category',
						boundaryGap : false,
						data : chart_data.x_data,
						axisTick : false,
						axisLabel : {
							formatter : function(val) {
								var secondTime = val.split(' ')[1];
								clock = secondTime.substring(0,2);
							if(secondTime.substring(3,5)=='00') return clock;
							return val;
							},
							interval: 'auto',
							rotate : (function(){
								var degree = 0;
								if(startTimeStr.split(" ")[1] != "00:00:00"){
									degree = 45;
								}
								return degree;
								})(),
						},
						axisLine:{
							lineStyle:{ color:"#666666" }
						},
						axisTick:{
							show:true
						}
					}],
					yAxis : [{
						name: '<%=rb.getString("ZuoKuoHao")%>' + unit + '<%=rb.getString("YouKuoHao")%>',
						type : 'value',
						axisLine:{
							lineStyle:{ color:"#666666" },
							show:true
						},
						axisLabel : {
							formatter : '{value}'
						}
					}],
					series : chart_data.series_data
				};
			kpiChart.setOption(option);
		},
	},
	mounted() {
		eventBus.$off("egw-data").$on("egw-data",this.init)
	}
});

</script>
