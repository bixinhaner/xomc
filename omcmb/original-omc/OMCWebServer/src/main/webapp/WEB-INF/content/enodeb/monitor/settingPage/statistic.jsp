<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
.statiscPage {
	background:#f1f1f2;
	height:100%;
	width:100%;
	display:flex;
	flex-wrap:wrap;
}
.chartItem {
	border:1px solid #d5dcec;
	border-radius:10px;
	height:35vh;
	min-height:300px;
	width:49%;
	background:#FFF !important;
	position: relative;
}
.kpiItem {
	flex:1;
	border:1px solid #d5dcec;
	border-radius:10px;
	min-height:350px;
	background:#FFF;
	width:90%;
}
.kpiTitle {
	height:50px;
	line-height:50px;
	padding:0 20px;
	font-size:14px;
	border-bottom:1px solid #d5dcec;
}
.kpiMain {
	display:flex;
	height:350px;
}
.chartLeftBtnBoxCls {
	width:300px;
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
.buttonSelectCls {
	background:rgba(var(--main-color-rgba1),0.1);
	color:var(--main-color);
}
.period-item-cls {
	position: absolute;
	top: 5px;
	right: 15px;
	z-index: 10;
}
</style>
<div id="statiscPage" class="statiscPage">
	<div class="chartItem" style="margin:0 10px 10px 0;">
		<el-radio-group @change="periodOnlineChange" size="mini" v-model="periodOnline" style="margin:0 0px 0px 25px; display: inline-block;" class="period-item-cls commonRadioButton">
			<el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
			<el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
			<el-radio-button label="month"><%=rb.getString("Yue")%></el-radio-button>
		</el-radio-group>
		<div id="echart_enb_online" :style="{height: periodOnline == 'min'?'calc(100% - 50px)':'100%',width: '100%'}"></div>
		<div v-show="periodOnline == 'min'" id="echart_enb_online_timeline" style="height: 50px;width: 100%;"></div>
	</div>
	<div class="chartItem">
		<el-radio-group @change="periodActiveChange" size="mini" v-model="periodActive" style="margin:0 0px 0px 25px; display: inline-block;" class="period-item-cls commonRadioButton">
			<el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
			<el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
			<el-radio-button label="month"><%=rb.getString("Yue")%></el-radio-button>
		</el-radio-group>
		<div id="echart_enb_active" :style="{height: periodActive == 'min'?'calc(100% - 50px)':'100%',width: '100%'}"></div>
		<div v-show="periodActive == 'min'" id="echart_enb_active_timeline" style="height: 50px;width: 100%;"></div>
	</div>
	<div class="chartItem" style="margin:0 10px 10px 0;">
		<el-radio-group @change="periodUeChange" size="mini" v-model="periodUe" style="margin:0 0px 0px 25px; display: inline-block;" class="period-item-cls commonRadioButton">
			<el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
			<el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
			<el-radio-button label="month"><%=rb.getString("Yue")%></el-radio-button>
		</el-radio-group>
		<div id="echart_enb_ue" :style="{height: periodUe == 'min'?'calc(100% - 50px)':'100%',width: '100%'}"></div>
		<div v-show="periodUe == 'min'" id="echart_enb_ue_timeline" style="height: 50px;width: 100%;"></div>
	</div>
	<div class="chartItem">
		<el-radio-group @change="periodEarfcnChange" size="mini" v-model="periodEarfcn" style="margin:0 0px 0px 25px; display: inline-block;" class="period-item-cls commonRadioButton">
			<el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
			<el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
			<el-radio-button label="month"><%=rb.getString("Yue")%></el-radio-button>
		</el-radio-group>
		<div id="echart_enb_earfcn" :style="{height: periodEarfcn == 'min'?'calc(100% - 50px)':'100%',width: '100%'}"></div>
		<div v-show="periodEarfcn == 'min'" id="echart_enb_earfcn_timeline" style="height: 50px;width: 100%;"></div>
	</div>
	
	<div class="kpiItem">
		<div class="kpiTitle">
			<span><%=rb.getString("XingNengGuanLi")%></span>
			<div class="box-titles" style="padding: 0px 20px;float:right;width:80%">
				<div id="time_line_chart" style="width: 70%;min-width:140px;height: 45px;display:inline-block;"></div>
				<div class="herderButtonBoxCls" style="float:right;">
					<el-radio-group v-model='grainType' size="mini" class="commonRadioButton" @change="grainTypeChange" style='margin-top:10px;'>
		                <el-radio-button  label="15">15min</el-radio-button>
		                <el-radio-button  label="60">60min</el-radio-button>
		            </el-radio-group>
					<!-- <div :class="grainType == '15' ? 'buttonSelectCls': 'buttonDefaultCls'" @click="grainTypeChange('15')">15min</div>
					<div :class="grainType == '60' ? 'buttonSelectCls': 'buttonDefaultCls'" style="border-left:1px solid #FFF;" @click="grainTypeChange('60')">60min</div> -->
				</div>
			</div>
		</div>
		<div class="kpiMain">
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
	
</div>

<script>
var timeParam = getNowTimeToZoneTimeRange(timeZone, 144)
var start_time_enb = timeParam.start_time.substring(0,11)+"00:00:00";
var end_time_enb = timeParam.end_time;
var chartPageVue = new Vue({
	el:'#statiscPage',
	data(){
		return{
            enbSelectedRow:{},
			grainType:'15',
			currentPMIndex:6,
			kpiChartType:'',
			kpiChartList:[],
			code:'',
			sn:'',
			
			periodOnline: 'min',
			periodActive: 'min',
			periodUe: 'min',
			periodEarfcn: 'min',
		}
	},
	computed: {
		
	},
	methods:{
		// 初始化
		init(row,code,sn){
			var vm = this;

            vm.enbSelectedRow = row;
			vm.code = code;
			vm.initKPI();
			vm.initHistory();
		},

		periodOnlineChange(period) {
			var vm = this;

			vm.loadAjax('echart_enb_online', 'offline', period);

			if(period == 'min') {
				vm.initStaticTimeLine('echart_enb_online');
			}
		},
		periodActiveChange(period) {
			var vm = this;

			vm.loadAjax('echart_enb_active', 'active', period);

			if(period == 'min') {
				vm.initStaticTimeLine('echart_enb_active');
			}
		},
		periodUeChange(period) {
			var vm = this;

			vm.loadAjax('echart_enb_ue', 'ue', period);

			if(period == 'min') {
				vm.initStaticTimeLine('echart_enb_ue');
			}
		},
		periodEarfcnChange(period) {
			var vm = this;

			vm.loadAjax('echart_enb_earfcn', 'earfcn', period);

			if(period == 'min') {
				vm.initStaticTimeLine('echart_enb_earfcn');
			}
		},

		initHistory(){
			var vm = this;
			vm.loadAjax('echart_enb_active','active','min');
			vm.loadAjax('echart_enb_online','offline','min');
			vm.loadAjax('echart_enb_ue','ue','min');
			vm.loadAjax('echart_enb_earfcn','earfcn','min');

			vm.initStaticTimeLine('echart_enb_active');
			vm.initStaticTimeLine('echart_enb_online');
			vm.initStaticTimeLine('echart_enb_ue');
			vm.initStaticTimeLine('echart_enb_earfcn');
			// 窗口缩放时自适应
			window.removeEventListener('resize',vm.resizeChart);
			window.addEventListener('resize',vm.resizeChart);
		},
		/**
		* 请求后台统计图表数据
		* @param idName{number}：图表容器Id
		* @param type{number}：类型
		* @param fn{number}：回调方法
		**/
		loadAjax(idName,type,period,fn,endTime,dataIndex){
			var vm = this,
				enb_code = vm.code;
				dataArr=[],
				finalArr=[],
				pointerCount = 6*24+1,
				params = {
					device_type: "enb",
					device_code: enb_code,
					time_level: "min",
					timeZone: timeZone,
					//start_time: start_time_enb,
					end_time: endTime || end_time_enb
			};

			if(period == 'min') {
				params.start_time = params.end_time.substring(0,10) + ' 00:00:00';
			}
			if(period == 'week') {
				pointerCount = 7;
				params.time_level = 'week';
			}
			if(period == 'month') {
				pointerCount = 30;
				params.time_level = 'month';
			}

			// 获取统计数据
			axios.post('${ctx}/system/device/getDeviceStatusDataList.action',stringify(params)).then(function(response){
				var data = response.data;
				if(data){
				}else{
					data = [];
				}
				data.map((item,index)=>{
					var xDate=item.statistics_time.split(' ');
					var activeSta=item.active_status;
					var onlineSta=item.online_status;
					var ueSta=item.ue_count;
					var earfcnSta=item.earfcn;
					if(activeSta == 0){
						activeSta = activeSta+1;
					}else if(activeSta == 1){
						activeSta = activeSta+2
					}
					if(onlineSta == 0){
						onlineSta = onlineSta+1;
					}else if(onlineSta == 1){
						onlineSta = onlineSta+2
					}
					dataArr[index]=[xDate[0],xDate[1],activeSta,onlineSta,ueSta,earfcnSta];
				})
				//for(var initIndex=0;initIndex<7;initIndex++ ){
					var initIndex= [undefined,null,''].includes(dataIndex)? 0:(6 - dataIndex);
					var arrIndex = getYesterDay(initIndex);
					if(!finalArr[arrIndex]){
						var timeaxisArr = [], activeaxisArr = [],offlineaxisArr=[],ueaxisArr=[],earfcnaxisArr=[];
						for(var axisIndex=0; axisIndex<pointerCount; axisIndex++){
							var yaxisStartTime = getYesterDay(initIndex).substring(0,10)+' 00:00:00',
								offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*10);

							if(period == 'week') {
								yaxisStartTime = addDate(yaxisStartTime, -6);
								offsetTime = addDate(yaxisStartTime, axisIndex);
							}
							if(period == 'month') {
								yaxisStartTime = addDate(yaxisStartTime, -29);
								offsetTime = addDate(yaxisStartTime, axisIndex);
							}

							timeaxisArr.push(formatDate(offsetTime));
							activeaxisArr.push(undefined);
							offlineaxisArr.push(undefined);
							ueaxisArr.push(undefined);
							earfcnaxisArr.push(undefined);
						}
						finalArr[arrIndex] = [];
						finalArr[arrIndex]['time'] = timeaxisArr;
						finalArr[arrIndex]['active'] = activeaxisArr;
						finalArr[arrIndex]['offline'] = offlineaxisArr;
						finalArr[arrIndex]['ue'] = ueaxisArr;
						finalArr[arrIndex]['earfcn'] = earfcnaxisArr;
					}
				//}
				dataArr.map((item,index)=>{
					var dateIndex = item[0],
						timeValue=item[1].substring(0,5),
						activeValue=item[2],
						offlineValue=item[3],
						ueValue=item[4],
						earfcnValue=item[5];

					if(finalArr[arrIndex]['time'].includes(dateIndex+' '+item[1])){
						var differTimes = new Date(dateIndex+' '+item[1]).getTime() - new Date(dateIndex+' 00:00:00').getTime();
						var axisDateIndex = Math.round(differTimes/(1000*60*10));

						if(['week','month'].includes(period)) {
							axisDateIndex = finalArr[arrIndex]['time'].indexOf(dateIndex+' '+item[1]);
						}

						finalArr[arrIndex]['active'].splice(axisDateIndex,1,activeValue);
						finalArr[arrIndex]['offline'].splice(axisDateIndex,1,offlineValue); 
						finalArr[arrIndex]['ue'].splice(axisDateIndex,1,ueValue); 
						finalArr[arrIndex]['earfcn'].splice(axisDateIndex,1,earfcnValue); 
					}
				})
				var preData = null;
				for(var key in finalArr){
					if(typeof finalArr[key] != 'function'){
						if(preData){
							finalArr[key]['active'][pointerCount-1] = preData['active'][0];
							finalArr[key]['offline'][pointerCount-1] = preData['offline'][0];
							finalArr[key]['ue'][pointerCount-1] = preData['ue'][0];
							finalArr[key]['earfcn'][pointerCount-1] = preData['earfcn'][0];
							preData = finalArr[key]; 
						}else preData = finalArr[key];
					}
				}
				if(fn) try{fn();}catch(e){}
				vm.createEchart(finalArr,idName,type,period,dataIndex);
			}).catch(function(error){})
		},
		/**
		* 绘制统计页面图表
		* @param finalArr{array}：初始化后的图表系列数据
		* @param idName{string}：图表容器Id
		* @param type{string}：类型
		* @param period{string}：粒度类型
		**/
		createEchart(finalArr,idName,type,period,dataIndex){
			var vm = this;
				enodebChart = echarts.init(document.getElementById(idName));
			var titleList = {
				echart_enb_online:'<%=rb.getString("ZaiXianZhuangTai")%>',
				echart_enb_active:'<%=rb.getString("ShiFouJiHuo")%>',
				echart_enb_ue:'<%=rb.getString("ShouYe_UEShu")%>',
				echart_enb_earfcn:'<%=rb.getString("PinDian")%>',
				
			};

			var realIndex = [undefined,null,''].includes(dataIndex)? 0:(6 - dataIndex);
			
			var option = {
					title:{
						text:titleList[idName],
						left:15,
						top:10,
						textStyle:{
							fontSize:14
						}
					},
					
					grid:{
						top: 70,
						bottom: 30,
						left: 60
					},
					legend: {
						show: false,
						data: ['m']
					},
					color: ['#69b6fc','#DCDFE6'],
					tooltip:{
						trigger:"axis",
						padding:[10,5,10,5],
						formatter:function(params){
							if(params.length){
								var ueStr = "";
								var str = "",
									nameStr = params[0].name;

								if(['week','month'].includes(period)) {
									nameStr = nameStr.replace(' 00:00:00','');
								}

								if(idName == "echart_enb_earfcn"){
									ueStr += "<div>" + nameStr + "</div>";
									if(params[0].data == undefined){
										ueStr += "<div>EARFCN:-</div>" 
									}else{
										ueStr += "<div>EARFCN:" + params[0].data + "</div>"
									}
									return ueStr;
								}else if(idName == "echart_enb_ue"){
									ueStr += "<div>" + nameStr + "</div>";
									if(params[0].data == undefined){
										ueStr += "<div>UE:-</div>" 
									}else{
										ueStr += "<div>UE:" + params[0].data + "</div>"
									}
									return ueStr;
								}else{
									str += "<div>" + nameStr + "</div>";
									return str;
								}
							}
						}
					},
					xAxis: [{ 
						name: period == 'min'?'<%=rb.getString("XiaoShi") %>':'<%=rb.getString("Tian") %>',
						type:"category",
						axisLabel:{
							show:true,
							formatter : function(val) {
								if(period == 'min') {
									var secondTime = val.split(' ')[1];
									clock = secondTime.substring(0,2);
									if(secondTime.substring(3,5)=='00') return clock;
									return val;
								}else {
									return val.substring(8,10);
								}
							},
							interval: 'auto'
						},
						axisLine:{
							lineStyle:{ color:"#666666" }
						},
						splitLine:false ,
						boundaryGap:false,
						data: finalArr[getYesterDay(realIndex)]['time']
					}],
					yAxis: [{ 
						minInterval: (idName=='echart_enb_ue')?1:1,
						max: (['echart_enb_ue','echart_enb_earfcn'].includes(idName))?null:4,
						name: ('echart_enb_earfcn' == idName)?'':('echart_enb_ue'==idName?'<%=rb.getString("GeShuUe") %>':'<%=rb.getString("ZhuangTai") %>'), 
						type:"value",
						axisLine:{
							show:true,
							lineStyle:{ color:"#666666" },
							interval:12
						},
						axisTick:{
							show:false
						},
						splitLine:{
							show:idName =='echart_enb_ue'?true:false
						},
						axisLabel:{
							show:true,
							formatter:function(value,index){
								if(idName == 'echart_enb_active'){
									var texts=[];
									if(value == 3){
										texts.push('<%=rb.getString("ShouYe_HuoYue")%>');
									}else if(value == 1){
										texts.push('<%=rb.getString("ShouYe_BuHuoYue")%>');
									}
									return texts;
								}else if(idName == 'echart_enb_online'){
									var texts=[];
									if(value == 3){
										texts.push('<%=rb.getString("ShouYe_ZaiXian")%>');
									}else if(value == 1){
										texts.push('<%=rb.getString("ShouYe_BuZaiXian")%>');
									}
									return texts;
								}else if(idName == 'echart_enb_ue'){
									return value;
								}else if(idName == 'echart_enb_earfcn'){
									return value;
								}
							}
						}
					}],
					series: [{  	
						name: 'm',
						type:'line',
						step:idName=='echart_enb_ue'?false:true,
						markLine: idName=='echart_enb_ue'?{}:{
							data:[{
								name:' ',
								yAxis: 1
							},
							{
								name:' ',
								yAxis: 3
							}],
							animation: false,
							lineStyle:{
								normal:{
									type:'dashed',
									color:'#ccc',
									width:0.5,
									opacity:0.8
								}
							},
							symbolSize:0,
							silent:true,
							label:{
								normal:{
									show: false
								}
							}
						},
						symbolSize: 1,
						showAllSymbol: true,
						data: finalArr[getYesterDay(realIndex)][type]
					}] 
				} 
			
			enodebChart.setOption(option, true);
			vm.resizeChart();
		},
		// 图表自适应
		resizeChart(){
			var list = ['echart_enb_active', 'echart_enb_online', 'echart_enb_ue', 'echart_enb_earfcn','time_line_chart','echart1','echart2','echart3','echart4','echart5','echart6','echart7','echart8','echart9','echart10'];
			
			list.map(function(id){
				var dom = document.querySelector('#'+id);
				if(dom) {
					var itn = echarts.getInstanceByDom(dom);
					itn && itn.resize();
				}
			})
		},
		// KPI粒度切换事件
		grainTypeChange(type){
			var vm = this;
			
			vm.reloadPMChart(vm.currentPMIndex);
		},
		// 生成 统计 时间轴
		initStaticTimeLine(chartId) {
			var vm = this;
			var chart = echarts.init(document.getElementById(chartId + '_timeline'));
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
				var dataIndex = param.currentIndex
					endTime = dataIndex == 6? '':getYesterDay(6-dataIndex) + ' 59:59:59',
					rels = {
						echart_enb_active: 'active',
						echart_enb_online: 'offline',
						echart_enb_ue: 'ue',
						echart_enb_earfcn: 'earfcn'
					},
					relKey = rels[chartId];

				vm.loadAjax(chartId, relKey, 'min',null,endTime,dataIndex);
			});
			vm.resizeChart();
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
				//将ZhiBiao11  ZhiBiao12 合并展示；
                if(chartCodes.includes('MAC_RxTotKbytes')) {
					kpiChartList.push({
                        id:'echart10',label:'<%=rb.getString("HeBingZhiBiao11") %>',chart_perf_codes:'MAC_RxTotKbytes,MAC_TxTotKbytes',
                        chart_show_names:'UL,DL',units:'KByte/s'
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
	mounted(){
		eventBus.$off("enb-data").$on("enb-data",this.init)

	}
})
</script>
