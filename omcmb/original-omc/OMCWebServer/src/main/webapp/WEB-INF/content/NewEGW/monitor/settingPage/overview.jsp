<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwOverviewPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	padding:0px 20px;
	height:fit-content;
}
#egwOverviewPage .itemMainBoxTitle {
	height:36px;
	line-height: 36px;
	font-size:14px;
	font-weight:bold;
}
#egwOverviewPage .paramItemBoxCls {
	display:flex;
	flex-wrap:wrap;
}
#egwOverviewPage .paramItemBoxCls>div {
	width:33%;
	min-width:220px;
	min-height:55px;
}
#egwOverviewPage .paramItemLabel{
	color:#7a7992;
	margin-bottom:5px;
}
#egwOverviewPage .paramItemValue{
	height: 18px;
}
#egwOverviewPage .chartBoxCls{
	display: flex;
	flex-wrap: wrap;
}
#egwOverviewPage .flex-col-chart .el-card, #egwOverviewPage .flex-col-chart{
	overflow: unset;
}
#egwOverviewPage .flex-col-chart {
	position: relative;
}
#egwOverviewPage .half-persent {
	margin: 0px;
	flex: 1;
	background-color: #fff;
	min-height: 240px;
	min-width:400px;
	border:1px solid #E9E9E9;
}
#egwOverviewPage .chartBoxCls .el-card__body{
	overflow: hidden;
}
#egwOverviewPage .chartBoxCls .el-card__footer{
	display: none;
}
#egwOverviewPage .querySigGWStatisticBox{
	margin:20px 0px;
	display: flex;
	align-items: center;
}
#egwOverviewPage .querySigGWStatisticBox .el-input.el-input--small{
	width: 200px;
}
#egwOverviewPage .sigGWStatisticMainBox{
	display: flex;
	flex-wrap: wrap;
	margin-bottom: 20px;
}
#egwOverviewPage .sigGWStatisticMainBox >div{
	flex:1;
}
#egwOverviewPage .bottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 10px; 
}
</style>

<div id="egwOverviewPage" style='display:flex;flex-direction:column;width:100%;'>
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle"><%=rb.getString("SASSheBeiXinXi")%></div>
		<div class="paramItemBoxCls">
			<div>
				<div class="paramItemLabel"><%=rb.getString("eGWBianMa")%></div>
				<div class="paramItemValue">{{rowData.egwSn}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("EGWMingCheng")%></div>
				<div class="paramItemValue">{{rowData.egwName}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("SheBeiXingHao")%></div>
				<div class="paramItemValue">
					<!--<div v-if="rowData.generation == '4G' || !rowData.generation" class="egwModelBoxCls">
						<span class='enbModelCls' style='font-size:22px'></span>
						<span>4G</span>
					</div>
					<div v-if="rowData.generation == '5G'" class="egwModelBoxCls">
						<span class='gnbModelCls' style='font-size:22px'></span>
						<span>5G</span>
					</div>
					<div v-if="rowData.generation == '4G/5G'" class="egwModelBoxCls">
						<span class='enbModelCls' style='font-size:22px'></span>
						<span class='gnbModelCls' style='font-size:22px'></span>
						<span>4G/5G</span>
					</div>-->
					{{rowData.generation}}
				</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("EGWIP")%></div>
				<div class="paramItemValue">{{rowData.egwIp}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("EGWDuanKou")%></div>
				<div class="paramItemValue">{{rowData.egwPort}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("WangGuanJiZhanShu")%>(<%=rb.getString("ShiFouZaiXian")%>)</div>
				<div class="paramItemValue">{{rowData.enbNumber}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("UEShu")%>(<%=rb.getString("ShiFouZaiXian")%>)</div>
				<div class="paramItemValue">{{rowData.ueCount}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("EGWShangXingLiuLiang")%></div>
				<div class="paramItemValue">{{rowData.uplink_traffic}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("EGWXiaXingLiuLiang")%></div>
				<div class="paramItemValue">{{rowData.downlink_traffic}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("ZhuBeiZhuangTai")%></div>
				<div class="paramItemValue">{{rowData.ha_status}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("BanBen")%></div>
				<div class="paramItemValue">{{rowData.softwareVersion}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("HADuiDuanIP")%></div>
				<div class="paramItemValue">{{rowData.peerIp}}</div>
			</div>
			<div>
				<div class="paramItemLabel"><%=rb.getString("SheBeiZu")%></div>
				<div class="paramItemValue">{{rowData.groupName}}</div>
			</div>
		</div>
	</div>
	<div class="itemMainBoxCls" style="padding:0px;">
		<div class="itemMainBoxTitle" style="padding-left:20px;">System Status</div>
		<div class="chartBoxCls">
			<div :class="chartClass">
				<el-card shadow="hover" class="chart-card">
					<div id="cpuChart" style="width: 100%;height: 270px;"></div>
				</el-card>
			</div>
			<div :class="chartClass">
				<el-card shadow="hover" class="chart-card">
					<div id="memoryChart" style="width: 100%;height: 270px;"></div>
				</el-card>
			</div>
			<div :class="chartClass">
				<el-card shadow="hover" class="chart-card">
					<div id="diskChart" style="width: 100%;height: 270px;"></div>
				</el-card>
			</div>
		</div>
	</div>
	
</div>

<script>
var timeParam = getNowTimeToZoneTimeRange(timeZone, 144);
var start_time = timeParam.start_time.substring(0,11)+"00:00:00";
var end_time = timeParam.end_time;
var egwOverviewPage = new Vue({
	el: '#egwOverviewPage', 
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
				peerIp:'',
				softwareVersion:'',
				groupName:''
			},
			chartList: ['cpuChart','memoryChart','diskChart'],
			charts: {},
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
					});
				}
				
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);

				vm.reloadChart(code,6);
			});
		},
		// 所有图表自适应
		resizeChart(){
			var vm = this;
			vm.chartList.map(function(code){
				if(vm.charts[code]) vm.charts[code].resize();
			});
		},
		/**
		* 指定图表数据刷新
		* @param code{string} 图表类型
		* @param index{number}  时间轴下标
		*/
		reloadChart(code,index){
			var vm = this;
			
			if(vm.charts[code]) {
				if(['cpuChart','memoryChart','diskChart'].includes(code)) {
					vm.getDeviceChartData(code,index);// 统计数据刷新
				}
			}
		},
		/**
		* 统计数据刷新  CPU 硬盘 内存
		* @param code{string} 图表类型
		* @param index{number}  时间轴下标
		*/
		getDeviceChartData(code,index){
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
				pointerCount: 4,
				index: index,
				title: ''
			}
			for(var axisIndex=0; axisIndex < 4*24+1; axisIndex++){
				var yaxisStartTime = getYesterDay(6-index).substring(0,10)+ ' 00:00:00',
					offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*(60/chartData.pointerCount));
				chartData.data.push('-');
				chartData.xAxisData.push(formatDate(offsetTime));
			}
			var params = {
				startTime : s_time,
				endTime : e_time,
				timeZone : timeZone,
				egwCode:vm.egwCode
			};
			if(code == 'cpuChart'){
				chartData.legendNames = ['CPU'];
				chartData.yAxisName = '(%)';
				chartData.title = '<%=rb.getString("CPUShiYongLv")%>';
				chartData.color = ['#69B6FC'];
			}
			if(code == 'memoryChart'){
				chartData.legendNames = ['<%=rb.getString("NeiCun")%>'];
				chartData.yAxisName = '(%)';
				chartData.title = '<%=rb.getString("NeiCunShiYongLv")%>';
				chartData.color = ['#90EC97'];
			}
			if(code == 'diskChart'){
				chartData.legendNames = ['<%=rb.getString("CiPan")%>'];
				chartData.yAxisName = '(%)';
				chartData.title = '<%=rb.getString("CiPanShiYongLv")%>';
				chartData.color = ['#90EC97'];
			}
			// 图表数据
			axios.post("${ctx}/system/device/getEgwStatisticsList.action",stringify(params)).then(function(response){
				var data = response.data;

				if(data && data.length>0){
					data.map((item)=>{
						if(chartData.xAxisData.indexOf(item.startTime) != -1){
							var ids = chartData.xAxisData.indexOf(item.startTime);
							if(code == 'cpuChart'){
								chartData.data[ids] = item.cpu;
							}else if(code == 'memoryChart'){
								chartData.data[ids] = item.memory;
							}else if(code == 'diskChart'){
								chartData.data[ids] = item.disk;
							}
						}	
					})
				}
				vm.charts[code].setOption(vm.createOption(chartData));
				setTimeout(()=>{
					vm.charts[code].resize();
				},200)
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
		
	},
	mounted() {
		eventBus.$off("egw-data").$on("egw-data",this.init)
	}
});

</script>
