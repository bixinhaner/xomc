<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
 <style>
	.cpeHisChartStyle{
		position: relative;
		margin-left: 5px;
		width: calc(100% - 20px);
		height:380px;
	}
	.cpeHisChartStyle::before {
		position: absolute;
		content: '';
		top: 25px;
		right: 0px;
		bottom: 20px;
		left: 0px;
		border: 1px solid #e9e9e9;
	}
	.ChartCover{
		width:calc(100% - 300px);
		height:160px;
		line-height : 140px;
		position : absolute;
		margin-top : 80px;
		left : 200px;
		text-align : center;
		vertical-align : middle;
		font-size:20px;
		color : #b6b6b6;
	}
	.el-card__body{
		display:flex;
		flex-direction:column;
		flex:1 1 auto;
		height:100%;
		overflow:auto;
	}
	.kpiPeriod {
		display:inline-block;
		vertical-align:middle;
		margin-left:20px;
	}
	.kpiPeriod span{
		display:inline-block;
		border:1px solid #e9e9e9;
		height:30px;
		line-height:30px;
		text-align:center;
		width:45px;
		cursor:pointer;
		margin-left:-5px;
	}
	.kpiPeriod span:first-of-type{
		border-radius: 4px 0 0 4px;
		margin-left:-2px;
	}
	.kpiPeriod span:last-of-type{
		border-radius:0 4px 4px 0;
		margin-left:-2px;
	}
	.kpiPeriod span.active {
		color:#4D84FF;
		border-color:#4D84FF;
		position:relative;
	}
	.type-change-cls {
		display: flex;
	}
	.type-change-cls span {
		height: 20px;
		line-height: 20px;
	}
	.type-change-cls span:first-child {
		border-radius: 3px 0 0 3px;
	}
	.type-change-cls span:last-child {
		border-radius: 0 3px 3px 0;
	}
	.popoverPaddingCls{
		padding: 10px!important;
	}
 </style>

<div class="container" style="display: flex; flex-direction: column; height: 100%;overflow-x: hidden;width: 100%;">
	<div style='width:100%;background:#fff;border:0px solid #EEE;overflow:auto;' id="cpe_history_statistic">
		<div class="operations" style="right: 45px; top: 5px;">
			<span class="el-icon-circle-setting el-icon" style="margin-right: 15px;" onclick="goToCPESetting()"></span>
			
			<el-popover trigger="click" placement="left-start" title="<%=rb.getString("DaoChu")%>"  popper-class="popoverPaddingCls">
		       <div slot="reference" class="placeholder-bt" placeholder="<%=rb.getString("DaoChu")%>" @click="initTime">
		       		<span class="el-icon el-icon-circle-export"></span>
		       </div>
		       <div>
		       		<el-form ref="form" :model="form" :rules="rules" style="padding: 10px 0 30px 0;" label-position="top">
			       		<el-form-item v-show="isDayStatis" prop="timeRange" label="<%=rb.getString("Title_KPI_ShiJianFanWei")%><%=rb.getString("MaoHao")%>">
				       		<el-date-picker size="mini"
				       			v-model="form.timeRange"
				       			:picker-options="pickerOpts"
				       			value-format="yyyy-MM-dd"
						        type="daterange"
						        start-placeholder="<%=rb.getString("KaiShiShiJian")%>"
						        end-placeholder="<%=rb.getString("JieShuShiJian")%>">
				       		</el-date-picker>
			       		</el-form-item>
			       		<el-form-item v-show="!isDayStatis" prop="monthpre" label="<%=rb.getString("Title_KPI_ShiJianFanWei")%><%=rb.getString("MaoHao")%>">
			       			<div style="display: flex;align-items: center;">
					       		<el-date-picker size="mini" style="width: 160px;"
					       			v-model="form.monthpre"
					       			:picker-options="pickerOpts"
					       			value-format="yyyy-MM"
							        type="month">
					       		</el-date-picker>
					       		--
					       		<el-date-picker size="mini" style="width: 160px;"
					       			v-model="form.monthsuf"
					       			:picker-options="pickerOpts"
					       			value-format="yyyy-MM"
							        type="month">
					       		</el-date-picker>
			       			</div>
			       		</el-form-item>
		       		</el-form>
		       		
		       		<div>
			       		<el-button type="primary" @click="exportHis"><%=rb.getString("QueDing")%></el-button>
			       		<el-button @click="document.body.click()"><%=rb.getString("QuXiao")%></el-button>
		       		</div>
		       </div>
		    </el-popover>
	    </div>
		<div style="margin: 0 auto;display: flex;justify-content: space-between;position: relative;align-items: center;">
			<div class="area-title">
				<%=rb.getString("LiShi")%>
			</div>
			<div id="time_line_cht" style="width: 400px;height: 45px;margin-top: 15px;max-width: 70%;"></div>
			<div class="type-change-cls" style="margin-right: 10px;">
				<div class="kpiPeriod" style="margin: 10px;display: flex;">
					<span periodId="15" @click="changeStatis(true)" :class="{active: isDayStatis}"><%=rb.getString("Tian")%></span>
					<span periodId="60" @click="changeStatis(false)" :class="{active: !isDayStatis}"><%=rb.getString("Yue")%></span>
				</div>
			</div>

			<div style="margin:15px;width:170px;"  v-show="isDayStatis&&false">
				<div title="<%=rb.getString("QianYiTian")%>" id="previousDayButton" style="margin-top: -3px;" 
					class="operationDiv el-icon el-icon-left" onclick="previousDayDataGrid()">
				</div>
				<div class="selectedDetail_mana" style="margin-left:10px;">
					<div id="KPIDataGridDayTime" style="display:inline-block;margin-right:15px;margin-left:15px;"></div>
				</div> 
				<div id="afterDayButton" style="margin-top: -3px;" class="operationDiv el-icon el-icon-right disabled" onclick="afterDayDataGrid(this)"></div>
			</div>
			<div style="margin:15px;width:170px"  v-show="!isDayStatis&&false">
				<div title="<%=rb.getString("QianYiTian")%>" id="previousDayButton" style="margin-top: -3px;" 
					class="operationDiv el-icon el-icon-left" :class="{disabled: !monthValid.prev}" @click="previousMonth">
				</div>
				<div class="selectedDetail_mana" style="margin-left:10px;">
					<div style="display:inline-block;margin-right:15px;margin-left:15px;">{{currentMonth}}</div>
				</div> 
				<div id="afterDayButton" style="margin-top: -3px;" @click="nextMonth"
					class="operationDiv el-icon el-icon-right" :class="{disabled: !monthValid.next}"></div>
			</div>
		</div>
		<div class="slidebarDiv" style='top:0px;left:0px;bottom:10px;overflow-x:hidden;position: relative;'>
			<div class="ChartCover" style="" id="UL_MCS_noData"></div>	
			<div id="UL_MCS" class="cpeHisChartStyle"></div> 
			<div class="ChartCover" style="" id="DL_MCS_noData"></div> 
			<div id="DL_MCS" class="cpeHisChartStyle"></div>
			<div class="ChartCover" style="" id="RSRP0_noData"></div> 
			<div id="RSRP0" class="cpeHisChartStyle"></div>  
			<div class="ChartCover" style="" id="RSRP1_noData"></div> 
			<div id="RSRP1" class="cpeHisChartStyle"></div>
			<div class="ChartCover" style="" id="CINR0_noData"></div> 
			<div id="CINR0" class="cpeHisChartStyle"></div>  
			<div class="ChartCover" style="" id="CINR1_noData"></div> 
			<div id="CINR1" class="cpeHisChartStyle"></div> 
			<div class="ChartCover" style="" id="SINR_noData"></div> 
			<div id="SINR" class="cpeHisChartStyle"></div> 
			<div class="ChartCover" style="" id="DL_RATE_noData"></div> 
			<div id="DL_RATE" class="cpeHisChartStyle"></div> 
			<div class="ChartCover" style="" id="UL_RATE_noData"></div> 
			<div id="UL_RATE" class="cpeHisChartStyle"></div> 
		</div>
	</div>
</div>
<script type="text/javascript">
new Vue({
	el: '#cpe_history_statistic',
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
			isDayStatis: true,
			currentMonth: '',
			monthValid: {
				prev: true,
				next: false
			},
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
	methods: {
		changeStatis(bool){
			var vm = this;
			
			vm.isDayStatis = bool;
			if(vm.isDayStatis) {
				vm.getDataByDay();
			}else {
				vm.getDataByMonth();
			}

			var chart = echarts.getInstanceByDom(document.querySelector('#time_line_cht')),
				type = bool?'day':'month';
			switchTimeScale(chart, type);
		},
		previousMonth(){ // 上一个月份
			var vm = this,
				cur = new Date(vm.currentMonth+'-01 00:00:00');
			
			var valid = vm.isValidDate();
			if(valid.prev) {
				vm.changeMonth(cur, -1);
				vm.isValidDate(); // 立即判断是否有效，更新操作是否可用
				
				vm.getDataByMonth();
			}
		},
		nextMonth(){ // 下一个月份
			var vm = this,
				cur = new Date(vm.currentMonth+'-01 00:00:00');
			
			var valid = vm.isValidDate();
			if(valid.next) {
				vm.changeMonth(cur, 1);
				vm.isValidDate();
				
				vm.getDataByMonth();
			}
		},
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
		    loadCPEHistoryGraphData(params);
		},
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
		    loadCPEHistoryGraphData(params);
		},
		isValidDate(){// 检测月份是否在有效范围
			var vm = this,
				nowDate = getNowZoneTime(timeZone),
				cur = new Date(nowDate),
				yearAgo = cur.setMonth(cur.getMonth()-12);
			yearAgo = dateformatter(new Date(yearAgo));
			
			var valid = {
					prev: yearAgo.substring(0, 7).replace('-','') - vm.currentMonth.substring(0, 7).replace('-','')<0,
					next: vm.currentMonth.substring(0, 7).replace('-','') - nowDate.substring(0, 7).replace('-','')<0
				}
			Object.assign(vm.monthValid, valid);
			
			return valid;
		},
		changeMonth(cur,num){
			var vm = this,
				curTime = new Date(cur);
			
			var lastDate = curTime.setMonth(curTime.getMonth()+num);
			lastDate = new Date(lastDate);
			
			vm.currentMonth = dateformatter(lastDate).substring(0, 7);	
		},
		getMonthDay(year, month){
			var days = new Date(year, month, 0).getDate();
			
			return days;
		},
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
		exportHis() {
			var vm = this;
			
			vm.$refs.form.validate(function(valid){
				if(valid) {
					var lastMonth = vm.form.monthsuf||vm.form.timeRange[1],
						lastDay = new Date(lastMonth.substring(0,4),lastMonth.substring(5,7),0).getDate();
					exportByForm('${ctx}/cell/CPE/exportHistoryData.action',{
						type: vm.isDayStatis? 7:'m',
						startTime: vm.isDayStatis?vm.form.timeRange[0]+' 00:00:00':vm.form.monthpre+'-01 00:00:00',
						endTime: vm.isDayStatis?vm.form.timeRange[1]+' 23:59:59':vm.form.monthsuf+'-'+lastDay+' 23:59:59',
						cpeCode: sessionStorage.getItem('CPE_CODE'),
						serialNumber: sn,
						imsi: cpeDetailIMSI,
						timeZone: timeZone,
						product: '${product}'
					});
				}
			});
		}
	},
	mounted(){
		this.currentMonth = getNowZoneTime(timeZone).substring(0, 7);
		initDetailChart();
	}
})

var cpeCode = sessionStorage.getItem('CPE_CODE');
var sn = "${sn}";
var product = "${product}";

function initDetailChart() {
	var nowTime = getNowZoneTime(timeZone);
	var curr_time = nowTime.substring(0, 11);
	$("#KPIDataGridDayTime").html(nowTime.substring(0, 10));
	
	var start_time = nowTime.substring(0, 11) + "00:00:00";
    var end_time = formatDate(addDate(new Date(start_time), 1));
    var params = {
            cpeCode : sessionStorage.getItem('CPE_CODE'),
            timeZone : timeZone,
            start_time : start_time,
            end_time : end_time,
            product : product,
            token: 7
    };
    loadCPEHistoryGraphData(params);
	// 用于导航的时间刻度
	var timeLine = echarts.init(document.getElementById('time_line_cht'));
	switchTimeScale(timeLine,'day');
}

function goToCPESetting() {
	var lan = sessionStorage.getItem('lanFlag'),
		softVersion = sessionStorage.getItem('softVersion'),
		cpeName = sessionStorage.getItem('cpeName');
	
	closeInfoWindow();
	
	cpevm.goSetting({
		CPE_CODE: sessionStorage.getItem('CPE_CODE'), 
		OLDPRODUCT: product, 
		SOFTWARE_VERSION: softVersion, 
		CPE_NAME: cpeName, 
		LAN_INTERFACE: lan
	});
}

function switchTimeScale(chart, type) {
	var timeData = [getYesterDay(6).substring(5).replace("-","."),
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
		reloadHistoryChart(param.currentIndex,type);
	});
}

function reloadHistoryChart(index,type) {
	var params = {token: 7},
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

    loadCPEHistoryGraphData(params);
}

function resizeHisCharts() {
	['UL_MCS','DL_MCS','RSRP0','RSRP1','CINR0','CINR1','SINR','DL_RATE','UL_RATE'].map(function(id){
		echarts.getInstanceByDom(document.querySelector('#'+id)).resize();
	});
}

function loadCPEHistoryGraphData(params){
    $.post("${ctx}/cell/CPE/getCPEHistoryGraphData.action", params, function(data) {
		if (data) {
			var Xdata = data.bak_time_data,
				token = params.token;
			
			var ulmcsData = parseFloatData(data.ul_mcs_data);
			var dlmcsData = parseFloatData(data.dl_mcs_data);
			var rsrp1Data = parseFloatData(data.rsrp0_data);
			var rsrp2Data = parseFloatData(data.rsrp1_data);
			var cinr1Data = parseFloatData(data.cinr0_data);
			var cinr2Data = parseFloatData(data.cinr1_data);
			var sinrData = parseFloatData(data.cpe_sinr_data);
			var dlrateData = parseFloatData(data.dl_current_datarate_data);
			var ulrateData = parseFloatData(data.ul_current_datarate_data);
			
			//UL_mcs
			$("#UL_MCS_noData").html("");
			if(data.ul_mcs_flag=="noData"){
				generateCharts("UL_MCS", "UL_MCS", [], [], "",token);
				$("#UL_MCS_noData").html("<%=rb.getString("MeiShuJu")%>");
			}else if(data.ul_mcs_flag=="noSupport"){
				generateCharts("UL_MCS", "UL_MCS", [], [], "",token);
				$("#UL_MCS_noData").html("<%=rb.getString("DangQianBanBenBuZhiChi")%>");
			}else{
				generateCharts("UL_MCS", "UL_MCS", Xdata, ulmcsData, "",token);
			}
			//DL_mcs
			$("#DL_MCS_noData").html("");
			if(data.dl_mcs_flag=="noData"){
				generateCharts("DL_MCS", "DL_MCS", [], [], "",token);
				$("#DL_MCS_noData").html("<%=rb.getString("MeiShuJu")%>");
			}else if(data.dl_mcs_flag=="noSupport"){
				generateCharts("DL_MCS", "DL_MCS", [], [], "",token);
				$("#DL_MCS_noData").html("<%=rb.getString("DangQianBanBenBuZhiChi")%>");
			}else{
				generateCharts("DL_MCS", "DL_MCS", Xdata, dlmcsData, "",token);
			}
			//RSR1
			$("#RSRP0_noData").html("");
			if(rsrp1Data.length == 0){
				$("#RSRP0_noData").html("<%=rb.getString("MeiShuJu")%>");
				generateCharts("RSRP0", "RSRP1", [], [], "dBm",token);
			}else{
				generateCharts("RSRP0", "RSRP1", Xdata, rsrp1Data, "dBm",token);
			}
			//RSRP2
			$("#RSRP1_noData").html("");
			if(rsrp2Data.length == 0){
				$("#RSRP1_noData").html("<%=rb.getString("MeiShuJu")%>");
				generateCharts("RSRP1", "RSRP2", [], [], "dBm",token);
			}else{
				generateCharts("RSRP1", "RSRP2", Xdata, rsrp2Data, "dBm",token);
			}
			//CINR1
			$("#CINR0_noData").html("");
			if( cinr1Data.length == 0){
				$("#CINR0_noData").html("<%=rb.getString("MeiShuJu")%>");
				generateCharts("CINR0", "CINR1", [], [], "dB",token);
			}else{
				generateCharts("CINR0", "CINR1", Xdata, cinr1Data, "dB",token);
			}
			//CINR2
			$("#CINR1_noData").html("");
			if(cinr2Data.length == 0){
				$("#CINR1_noData").html("<%=rb.getString("MeiShuJu")%>");
				generateCharts("CINR1", "CINR2", [], [], "dB",token);
			}else{
				generateCharts("CINR1", "CINR2", Xdata, cinr2Data, "dB",token);
			}
			//SINR
			$("#SINR_noData").html("");
			if(sinrData.length == 0){
				$("#SINR_noData").html("<%=rb.getString("MeiShuJu")%>");
				generateCharts("SINR", "SINR", [], [], "dB",token);
			}else{
				generateCharts("SINR", "SINR", Xdata, sinrData, "dB",token);
			}
			//下行速率
			$("#DL_RATE_noData").html("");
			if(dlrateData.length == 0){
				$("#DL_RATE_noData").html("<%=rb.getString("MeiShuJu")%>");
				generateCharts("DL_RATE", "<%=rb.getString("CPEXiaXingTunTuLiang")%>", [], [], "Mbps",token);
			}else{
				generateCharts("DL_RATE", "<%=rb.getString("CPEXiaXingTunTuLiang")%>", Xdata, dlrateData, "Mbps",token);
			}
			//上行速率
			$("#UL_RATE_noData").html("");
			if(ulrateData.length == 0){
				$("#UL_RATE_noData").html("<%=rb.getString("MeiShuJu")%>");
				generateCharts("UL_RATE", "<%=rb.getString("CPEShangXingTunTuLiang")%>", [], [], "Mbps",token);
			}else{
				generateCharts("UL_RATE", "<%=rb.getString("CPEShangXingTunTuLiang")%>", Xdata, ulrateData, "Mbps",token);
			}
		}
	}, "json");
}

function generateCharts(elemId, title, Xdata, Ydata, yFormatUnit,token) {
    var myChart = echarts.init(document.getElementById(elemId)),
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
            left : 0
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
}

function parseFloatData(arr) {
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
}

// 前一天
function previousDayDataGrid() {
    $("#afterDayButton").attr("title", "<%=rb.getString("HouYiTian")%>");
    $("#afterDayButton").removeClass("disabled");
    var query_start_time_glob = formatDate(new Date($("#KPIDataGridDayTime").html()+ ' 00:00:00'));
    var prev_startTime = formatDate(addDate(new Date(query_start_time_glob), -1)), 
    	differOneDay = prev_startTime.substring(0, 10), 
    	earliestTime = formatDate(addDate(new Date(gloableTime), -7)).substring(0, 10);
    if (differOneDay == earliestTime) {
        $("#previousDayButton").addClass("disabled");
        $("#previousDayButton").attr("title", "");
    } else {
    	if(differ(differOneDay,earliestTime)==1){
            $("#previousDayButton").addClass("disabled");
            $("#previousDayButton").attr("title", "");
    	}
        $("#KPIDataGridDayTime").html(prev_startTime.substring(0, 10));

        var params = {token: 7};
        params.cpeCode = sessionStorage.getItem('CPE_CODE');
        params.timeZone = timeZone;
        params.start_time = prev_startTime;
        params.end_time = formatDate(addDate(new Date(prev_startTime), 1));
		params.product = product;
        loadCPEHistoryGraphData(params);
    }
}

// 后一天
function afterDayDataGrid(ele) {
    $("#previousDayButton").attr("title", "<%=rb.getString("QianYiTian")%>");
    $("#previousDayButton").removeClass("disabled");
    var query_start_time_glob = formatDate(new Date($("#KPIDataGridDayTime").html() + ' 00:00:00'));
    var next_startTime = formatDate(addDate(new Date(query_start_time_glob), 1));
    var nowDivTime = formatDate((new Date(gloableTime))).substring(0, 10), differOneDay = formatDate(addDate(
            new Date(gloableTime), 1)).substring(0, 10);
    latestTime = query_start_time_glob.substring(0, 10);
    if (differOneDay == latestTime)
        $("#afterDayButton").addClass("disabled");
    if (nowDivTime == latestTime) {
        $("#afterDayButton").addClass("disabled");
        $("#afterDayButton").attr("title", "");
    } else {
    	if(differ(nowDivTime,latestTime)==1){
    		$("#afterDayButton").addClass("disabled");
            $("#afterDayButton").attr("title", "");
    	}
        $("#KPIDataGridDayTime").html(next_startTime.substring(0, 10));

        var params = {token: 7};
        params.cpeCode = sessionStorage.getItem('CPE_CODE');
        params.timeZone = timeZone;
        params.start_time = next_startTime;
        params.end_time = formatDate(addDate(new Date(next_startTime), 1));
        params.product = product;
        loadCPEHistoryGraphData(params)
    }
}
</script>