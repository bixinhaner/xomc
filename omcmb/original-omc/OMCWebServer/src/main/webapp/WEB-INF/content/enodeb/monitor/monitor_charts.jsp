<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
    .minisize .fix-right-slide, .minisize .slide-arrow.left, .expanded .slide-arrow.right {
        display: none;
    }
    .minisize .enodeTable {
        width: 100%;
    }

    .fix-right-slide > div {
        display: flex;
        flex-direction: column;
        flex: 1 auto;
        min-width: 200px;
        height: 130px;
        align-items: stretch;
        max-height: 130px;
        min-height: 120px;
    }
    .fix-right-slide > div:not(:nth-child(5)) {
        border-bottom: 1px solid #e9e9e9;
    }
    .fix-right-slide > div:nth-last-of-type(2) {
        height: 200px;
        max-height: 200px;
    }
    .fix-right-slide > div:nth-last-of-type(3) {
        height: 200px;
        max-height: 200px;
    }
    .expanded .fix-right-slide > div {
        height: calc(50% - 10px);
        max-height: 49%;
        border: 1px solid #E9E9E9;
        margin: 5px;
        background: #fff;
        border-radius: 5px;
    }
</style>

<div style="width: 45%; position: relative;">
    <div id="online_count_rate" style="position: absolute;left: 110px;z-index: 100;font-size: 12px;top: 13px;"></div>
    <div id="online_status_chart" style="width: 100%;height: 100%;"></div>
</div>
<div style="width: 45%; position: relative;">
    <div id="active_count_rate" style="position: absolute;left: 110px;z-index: 100;font-size: 12px;top: 13px;"></div>
    <div id="active_status_chart" style="width: 100%;height: 100%;"></div>
</div>
<div style="width: 45%; position: relative;">
    <div id="mme_count_rate" style="position: absolute;left: 100px;z-index: 100;font-size: 12px;top: 13px;"></div>
    <div id="mme_connected_chart" style="width: 100%;height: 100%;"></div>
</div>
<div style="width: 20%;">
    <div id="enb_product_type_chart" style="width: 100%;height: 100%;"></div>
</div>
<div style="width: 20%;min-height: 220px;">
    <div id="device_time_chart" style="width: 100%;height: 100%;"></div>
</div>

<script>
    var chartCodes = ["online_status_chart","active_status_chart","mme_connected_chart","enb_product_type_chart","device_time_chart"];

   
    initLineChart(false,'');
    
    refresh_cellStatusStatistics();

    var enbLayerTimer = setInterval(function(){
		var ctn = document.querySelector('#cellInfo');

		if(!ctn) {
			clearInterval(enbLayerTimer);
			return;
		}

		var cls = ctn.classList;
        var group_id = '';
        try {
            if(enbvm.queryParams.group_id){
                group_id = enbvm.queryParams.group_id;
            }
        } catch (error) {
            
        }
		if(cls.contains('expanded')) {
			initLineChart(true,group_id);
		}else {
			initLineChart(false,group_id)
		}
		resizeEnbCharts();
	},1000*60*10);
    
    window.removeEventListener('resize',resizeEnbCharts);
    window.addEventListener('resize',resizeEnbCharts);

    function toggleExpand() {
		var ctn = document.querySelector('#cellInfo'),
			cls = ctn.classList;
		var group_id = '';
        try {
            if(enbvm.queryParams.group_id){
                group_id = enbvm.queryParams.group_id;
            }
        } catch (error) {
            
        }
		if(cls.contains('expanded')) {
			cls.remove('expanded');
			initLineChart(false,group_id);
		}else {
			cls.add('expanded');
			initLineChart(true,group_id)
		}

		cls.remove('minisize');
		resizeEnbCharts();
	}

	function minisize() {
		var ctn = document.querySelector('#cellInfo'),
			cls = ctn.classList;

		if(cls.contains('minisize')) {
			cls.remove('minisize');
			$('#enb_export_op').css({right: '220px'});
			$('#addDeviceBtn').css({right: '110px'});
			$('#importDeviceBtn').css({right: '60px'});
			$('#addDeviceCard').css({right: '40px'});
			$('#importDeviceCard').css({right: '40px'});
		}else {
			cls.add('minisize');
			$('#enb_export_op').css({right: '10px'});
			$('#addDeviceBtn').css({right: '110px'});
			$('#importDeviceBtn').css({right: '60px'});
			$('#addDeviceCard').css({right: '40px'});
			$('#importDeviceCard').css({right: '40px'});
		}
	}

    function initLineChart(isExpanded,group_id) {
        var charts = [],
            titles = {
                "online_status_chart": "Online Status",
                "active_status_chart": "Cell Status",
                "mme_connected_chart": "MME Connected Status"
            };

        ["online_status_chart","active_status_chart","mme_connected_chart"].map(function(id){
            var chartDom = document.querySelector('#'+id);
            if(chartDom) {
                var chart = echarts.init(chartDom);

                charts.push(chart);
                // 初始化时间轴切换事件
                chart.off('timelinechanged');
                chart.on('timelinechanged',function(p){
                    proccessEnb(id,p.currentIndex,isExpanded,group_id);
                });
                proccessEnb(id,6,isExpanded,group_id);
            }
        });

        echarts.connect(charts);
        // 初始化产品类型图表
        initEnbProductType(isExpanded);
        // 初始化设备运行时长图表
        initDeviceRunTime(isExpanded);
    }

    function initEnbProductType(expanded) {
        var chartDom = document.querySelector('#enb_product_type_chart');

        if(!chartDom) return;

        var prodChart = echarts.init(chartDom),
            legend = [],
            sdata = [],
            splitNum = 10,
            colorList = ['#69b6fc','#90ec97','#e9a4a4','#f3cc90','#d7a3ef','#ada3ef','#4bdedb','#4bdea3','#e6a46b','#efa3d3','#f5f5f5'];
        $.post('${ctx}/system/device/getEnbProductStatisticsData.action',function(data){
            if(data) {
                var total = 0,
                    others = [];
                data.map(function(item){
                    total += item.device_count*1;
                });
                data.map(function(item,index){
                    var name = item.product_type+ ': ' +item.device_count+ ' (' + (item.device_count*100/total).toFixed(2) + '%)';
                    if(index<splitNum) {
                        legend.push(name);
                        sdata.push({name: name, value: item.device_count*1});
                    }else {
                        others.push({name: name, value: item.device_count*1});
                    }
                });

                if(others.length) {
                    var otherCount = 0;
                    others.map(function(item){
                        otherCount += item.value;
                    });
                    var otherName = '<%=rb.getString("QiTa")%>: '+otherCount+ ' (' + (otherCount*100/total).toFixed(2) + '%)';
                    legend.push(otherName);
                    sdata.push({name: otherName, value: otherCount});
                }
                
                colorList = colorList.slice(0,splitNum);
                colorList.push('#f5f5f5');
            }
            
            prodChart.setOption({
                title: {
                    top: 10,
                    left: 10,
                    text: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',
                    textStyle: {
                        fontSize: 12
                    }
                },
                color: colorList, 
                legend: {
                    itemWidth: expanded?15:10,
                    orient: 'horizontal',
                    right: (expanded?5:null),
                    left: (expanded?20:10),
                    bottom: 5,
                    itemHeight: (expanded?10:4),
                    textStyle: {
                        fontSize: (expanded?12:6)
                    },
                    data: legend,
                    inactiveColor: '#dcdfe6'
                },
                tooltip: {
                    trigger: 'item',
                    formatter: '{b}',
                    confine: true
                },
                grid: {
                    top: 60,
                    left: 35,
                    right: 15,
                    bottom: 90
                },
                series: [
                    {
                        name: '',
                        type: 'pie',
                        radius: '40%',
                        center: ['50%','40%'],
                        data: sdata,
                        emphasis: {
                            itemStyle: {
                                shadowBlur: 10,
                                shadowOffsetX: 0,
                                shadowColor: 'rgba(0,0,0,0.5)'
                            }
                        },
                        label: {
                            normal: {
                                color: '#333',
                                show: expanded,
                                fontSize: 8
                            }
                        },
                        labelLine: {
                            normal: {
                                show: expanded
                            }
                        }
                    }
                ]
            });
        },'json');
    }

    function initDeviceRunTime(expanded) {
        var chartDom = document.querySelector('#device_time_chart');

        if(!chartDom) return;
        
        var devicechart = echarts.init(chartDom),
            legend = ['<10days','10-30days','30-90days','>90days'],
            data = [];

        $.post('${ctx}/system/device/getDeviceEnbUpTimeStatisticsData.action',function(data){
            if(data) {
                data = [
                    data.tenDayDeviceCount,
                    data.thirtyDayDeviceCount,
                    data.ninetyDayDeviceCount,
                    data.bigNinetyDayDeviceCount
                ];
                var total = data[0]+data[1]+data[2]+data[3],
                    lgd1 = '<10days: ' + data[0] + ' (' + (data[0]*100/total).toFixed(2)+'%)',
                    lgd2 = '10-30days: ' + data[1] + ' (' + (data[1]*100/total).toFixed(2)+'%)',
                    lgd3 = '30-90days: ' + data[2] + ' (' + (data[2]*100/total).toFixed(2)+'%)',
                    lgd4 = '>90days: ' + data[3] + ' (' + (data[3]*100/total).toFixed(2)+'%)';

                legend = [
                    lgd1,
                    lgd2,
                    lgd3,
                    lgd4
                ];
                series = [
                    {
                        name: lgd1,
                        type: 'bar',
                        data: []
                    },
                    {
                        name: lgd2,
                        type: 'bar',
                        data: []
                    },
                    {
                        name: 'time',
                        type: 'bar',
                        data: data,
                        barWidth: expanded?30:15,
                        itemStyle: {
                            normal: {
                                color: function(param){
                                    return ['#69b6fc','#90ec97','#e9a4a4','#f3cc90'][param.dataIndex];
                                },
                                label: {
                                    show: true,
                                    position: 'top',
                                    formatter: '{c}'
                                }
                            }
                        }
                    },
                    {
                        name: lgd3,
                        type: 'bar',
                        data: []
                    },
                    {
                        name: lgd4,
                        type: 'bar',
                        data: []
                    }
                ];
            }

			var maxVal = Math.max(  data.tenDayDeviceCount,
				                    data.thirtyDayDeviceCount,
				                    data.ninetyDayDeviceCount,
				                    data.bigNinetyDayDeviceCount),
				maxlength = (maxVal+'').length,
				gridLeft = maxlength>2?(maxlength*11+10):35;
            
            devicechart.setOption({
                title: {
                    top: 10,
                    left: 10,
                    text: '<%=rb.getString("SheBeiYunXingShiChang")%>',
                    textStyle: {
                        fontSize: 12
                    }
                },
                color: ['#69b6fc','#90ec97','#e9a4a4','#f3cc90'],
                grid: {
                    top: 50,
                    left: gridLeft,
                    right: 45,
                    bottom: expanded?120:90
                },
                tooltip: {
                    trigger: 'axis',
                    formatter: function(p) {
                        var ops = p[0];
                        
                        return legend[ops.dataIndex];
                    },
                    confine: true
                },
                legend: {
                    orient: 'vertical',
                    itemHeight: expanded?12:8,
                    left: 30,
                    bottom: 5,
                    data: legend,
                    textStyle: {
                        fontSize: expanded?12:8
                    },
                    icon: 'circle',
                    selectedMode: false,
                    inactiveColor: '#dcdfe6'
                },
                xAxis: {
                    name: '<%=rb.getString("Tian")%>',
                    type: 'category',
                    data: ['<10','10-30','30-90','>90'],
                    axisLine: {
                        show : true,
                        lineStyle:{ color:"#999999" }
                    },
                    axisLabel: {
                        show: expanded
                    },
                    axisTick: {
                        show: expanded,
                        alignWithLabel: true
                    }
                },
                yAxis: {
                    type: 'value',
                    axisLabel : {
                        show:true,
                        textStyle:{ color:"#999999" },
                        lineStyle:{ color:"#999999" }
                    },
                    axisLine:{
                        lineStyle:{ color:"#999999" }
                    },
                    splitLine : {
                        lineStyle:{ color:"#f1f1f4" }
                    }
                },
                series: series
            });
        },'json');
        
    }

    function resizeEnbCharts() {
        chartCodes.map(function(id){
            var dom = document.querySelector('#'+id);

            dom && echarts.getInstanceByDom(dom).resize();
        })
    }

    function proccessEnb(code,index,bool,group_id) {
        var legend = ['Active','Inactive'],
            legendNames = ['<%=rb.getString("ShouYe_HuoYue")%>','<%=rb.getString("ShouYe_BuHuoYue")%>'],
            unitName='<%=rb.getString("GeShu")%>',
            title = 'eNB <%=rb.getString("ShouYe_HuoYueZhuangTai")%>';
            
        end_time = getNowTimeToZoneTimeRange(timeZone, 144).end_time;
        
        if(code=='online_status_chart') {
            legend = ['Onlince','Offline'];
            legendNames = ['<%=rb.getString("ShouYe_ZaiXian")%>','<%=rb.getString("ShouYe_BuZaiXian")%>'];
            title = 'eNB <%=rb.getString("ShouYe_ZaiXianZhuangTai")%>';
        }else if(code=='mme_connected_chart') {
            legend = ['Connected','Disconnected'];
            legendNames = ['<%=rb.getString("ShouYe_LianJie")%>','<%=rb.getString("ShouYe_FeiLianJie")%>'];
            title = '<%=rb.getString("MMEZhuangTai")%>';
        }

        var s_time = getYesterDay(6-index).substring(0,10) + ' 00:00:00',
            e_time = getYesterDay(6-index).substring(0,10) + ' 23:59:59';
        
        if(index==6) {
            e_time = end_time;
        }else{
            e_time = getYesterDay(6-index-1).substring(0,10) + ' 00:00:00';
        }
        var paramsENB = {
                device_type: "enb",
                group_id:group_id,
                time_level: "min",
                timeZone: timeZone,
                start_time: s_time,
                end_time: e_time
            };
        
        $.post("${ctx}/system/device/getDeviceStatisticsDataList.action", paramsENB, function(data){
            var params = {
                    legend: legend,
                    legendNames: legendNames,
                    query: paramsENB,
                    data: data,
                    unit: unitName,
                    pointerCount: 6,
                    index: index,
                    code: code,
                    title: title
                },
                chart = echarts.getInstanceByDom(document.querySelector('#'+code));
            
            chart && chart.setOption(createEnbOption(params,bool));  // 根据请求的数据重新渲染图表
        }, "json")
    }
    // 生成图表的 option
    function createEnbOption(params,expanded) {
        var pointerCount = params.pointerCount, title = params.title,
            finalArr = proccessData(params), startMinDashboard = ' 00:00:00',
            index = params.index, code = params.code, unit =  params.unit,
            legend = params.legend || [], legendNames = params.legendNames,
            type1 = legend[0], type2 = legend[1],
            dates = getDays(),
            option = {
                baseOption: {
                    title: {
                        text: title,
                        textStyle: {
                            fontSize: 12
                        },
                        top: 10,
                        left: expanded?20:10
                    },
                    timeline: {
                        show: expanded,
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
                        data: [getYesterDay(6).substring(5).replace("-","."),
                                getYesterDay(5).substring(5).replace("-","."),
                                getYesterDay(4).substring(5).replace("-","."),
                                getYesterDay(3).substring(5).replace("-","."),
                                getYesterDay(2).substring(5).replace("-","."),
                                getYesterDay(1).substring(5).replace("-","."),
                                getYesterDay(0).substring(5).replace("-",".")],
                        notMerge:true,
                        currentIndex: index,
                        checkpointStyle:{
                            color:'#209FFF',
                            borderColor:'none'
                        }
                    },
                    tooltip : {
                        trigger : 'axis',
                        formatter:function(params){
                            var timeStr = "";
                            var params = JSON.parse(JSON.stringify(params));
                            var str = "";
                            if(code == 'ueCount'){
                                if(params[0]){
                                    str += "<div>" + params[0].name + "</div>";
                                    str += "<div>" + params[0].seriesName + ": " + params[0].data + "</div>" 
                                }
                            }else{
                                for(var i=0;i<params.length;i++){
                                    if (params[0].name.indexOf("/")>=0){
                                            timeStr = params[0].name.split("/");
                                            if(!str){
                                                str += "<div>" + timeStr[0] + " -- </div>";
                                                str += "<div>" + timeStr[1] + "</div>";                     
                                            }
                                        } else {
                                            if(str == ''){
                                                str += "<div>" + params[0].name + "</div>";
                                            }
                                        }
                                    str += "<div>" + params[i].seriesName + ": " + params[i].data + "</div>" 
                                }
                            }
                            return str;
                        },
                        confine: true
                    },
                    grid:{
                        bottom: expanded?80:15,
                        left:  expanded?60:40
                    }, 
                    color: ['#90EC97','#E9A4A4'],
                    legend: {
                        top: expanded?10:30,
                        itemHeight: expanded?12:8,
                        textStyle: {
                            fontSize: expanded?12:8
                        },
                        data: legendNames,
                        selected: {
                            [legendNames[0]]: true,
                            [legendNames[1]]: expanded
                        }
                    },
                    xAxis : [{
                        name : expanded?'<%=rb.getString("XiaoShi")%>':'',
                        type : 'category',
                        boundaryGap : false,
                        axisLine:{
                            show : true,
                            lineStyle:{ color:"#666666" }
                        },
                        axisLabel : {
                            show: expanded,
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
                                    if(startMinDashboard != " 00:00:00"){
                                        degree = 45;
                                    }
                                    return degree;
                            })(),
                        },
                        axisTick: {
                            show: expanded
                        }
                    }],
                    yAxis : [{
                        name: expanded?unit:'',
                        minInterval: (code=='thoughtput' || code=='traffic' )? null:1,
                        type: 'value',
                        axisLabel : {
                            show:true,
                            textStyle:{ color:"#666666" },
                            lineStyle:{ color:"#666666" }
                        },
                        axisLine:{
                            lineStyle:{ color:"#666666" }
                        },
                        splitLine: {
                            lineStyle:{ color:"#f1f1f4",type: 'dotted'}
                        }
                    }],
                    series : [{  
                                name: legendNames[0],
                                type: 'line',
                                symbolSize: 1,
                                itemStyle: {
                                    normal: {
                                        areaStyle: {opacity: 0.2}
                                    }
                                },
                                areaStyle: {opacity: 0.2},
                                showAllSymbol: true,
                                step: false,
                                connectNulls: false
                            },
                            {  
                                name: legendNames[1],
                                type: 'line',
                                symbolSize: 1,
                                itemStyle: {
                                    normal: {
                                        areaStyle: {opacity: 0.2}
                                    }
                                },
                                areaStyle: {opacity: 0.2},
                                showAllSymbol : true,
                                step: false,
                                connectNulls: false
                            }]
                },
                options: []
            };
        
        var subOpts = [];
        for(var idx=0; idx<7; idx++) { // 初始化空数据
            subOpts.push({  
                            xAxis: [{ data: []}],
                            series: [
                                {data: []} , 
                                {data: []}
                            ]  
                        });
        }
        option.options = subOpts;
        // 设置当前选中的日期节点数据
        option.options[index].xAxis[0].data = finalArr[getYesterDay(6-index)]['x'];
        option.options[index].series[0].data = finalArr[getYesterDay(6-index)][type1];
        option.options[index].series[1].data = finalArr[getYesterDay(6-index)][type2];

        var sm1 = option.options[index].series[0].data.some(function(item){
                return item!='-';
            }),
            sm1 = option.options[index].series[1].data.some(function(item){
                return item!='-';
            })
        if(sm1 && sm1) {
            option.options[index].yAxis = [{min: null, max: null}];
            document.querySelector('#'+code).classList.remove('no-data');
        }else {
            option.options[index].yAxis = [{min: 0, max: 3}];
            document.querySelector('#'+code).classList.add('no-data');
        }

        var maxValue = 0;
        finalArr[getYesterDay(6-index)][type1].map(function(item){
            if(!isNaN(item)) {
                maxValue = Math.max(maxValue, item);
            }
        })
        if(expanded) {
            finalArr[getYesterDay(6-index)][type2].map(function(item){
                if(!isNaN(item)) {
                    maxValue = Math.max(maxValue, item);
                }
            })
        }
        var maxlength = (maxValue+'').length;
		option.baseOption.grid.left = maxlength>2?(maxlength*11+10):40;
        
        return option;
    }

    function proccessData(params) {
        var startMinDashboard = ' 00:00:00',
            pointerCount = params.pointerCount,
            chart_data = params.data, chart_id = params.code,
            deviceCount = "", pointerNum = pointerCount*24+1,
            legend_code = params.legend, legend_name = legend_code,
            legend_code_line1 = legend_code[0], legend_code_line2 = legend_code[1],
            type1 = legend_code[0], type2 = legend_code[1];

        dataArr=[];
        $.each(chart_data,function(index,item){
            if( "thoughtput" == chart_id || "traffic" == chart_id ){
                var xDate = item.start_time.split(' ');
            }else{
                var xDate = item.statistics_time.split(' ');
            }
            deviceCount = item.device_count;
            var onlineOrActiveOrConnectCount = "";    
            var offOnlineOrInactiveOrDisconnectCount = "";
            if(chart_id == 'active_status_chart'){
                onlineOrActiveOrConnectCount = item.active_count;
                offOnlineOrInactiveOrDisconnectCount = deviceCount - onlineOrActiveOrConnectCount;
                dataArr[index]=[xDate[0],xDate[1],onlineOrActiveOrConnectCount,offOnlineOrInactiveOrDisconnectCount];
            }else if(chart_id == 'online_status_chart'){
                onlineOrActiveOrConnectCount = item.online_count;
                offOnlineOrInactiveOrDisconnectCount = deviceCount - onlineOrActiveOrConnectCount;
                dataArr[index]=[xDate[0],xDate[1],onlineOrActiveOrConnectCount,offOnlineOrInactiveOrDisconnectCount];
            }else if(chart_id == 'mme_connected_chart'){
                onlineOrActiveOrConnectCount = item.mme_connected_count;
                offOnlineOrInactiveOrDisconnectCount = deviceCount - onlineOrActiveOrConnectCount;
                dataArr[index]=[xDate[0],xDate[1],onlineOrActiveOrConnectCount,offOnlineOrInactiveOrDisconnectCount];
            }
        });
        finalArr = [];//[2017-01-01,00:00:00,0,1,'25' ]
        for(var initIndex=0;initIndex<7;initIndex++ ){
            var arrIndex = getYesterDay(initIndex);
            if(!finalArr[arrIndex]){
                var activeCountArr = [], yaxisArr = [],onlineOrActiveOrConnectCountArr=[],offOnlineOrInactiveOrDisconnectCountArr=[];
                for(var axisIndex=0; axisIndex<pointerNum; axisIndex++){
                    var yaxisStartTime = getYesterDay(initIndex).substring(0,10)+ startMinDashboard,
                        offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*(60/pointerCount));
                    yaxisArr.push(formatDate(offsetTime));
                    onlineOrActiveOrConnectCountArr.push('-');
                    offOnlineOrInactiveOrDisconnectCountArr.push('-');
                }
                finalArr[arrIndex] = [];
                finalArr[arrIndex]['x'] = yaxisArr; //x轴数据
                finalArr[arrIndex][type1] = onlineOrActiveOrConnectCountArr;
                finalArr[arrIndex][type2] = offOnlineOrInactiveOrDisconnectCountArr;
            }
        }
        $.each(dataArr,function(n,m){
            var dateIndex = m[0],yValue=m[1].substring(0,5),onlineOrActiveOrConnectArrValue=m[2],offOnlineOrInactiveOrDisconnectValue=m[3];
            if(finalArr[dateIndex]['x'].includes(dateIndex+' '+m[1])){
                var differTimes = new Date(dateIndex+' '+m[1]).getTime() - new Date(dateIndex+' 00:00:00').getTime();
                var axisDateIndex = Math.round(differTimes/(1000*60*(60/pointerCount)));
                finalArr[dateIndex][type1].splice(axisDateIndex,1,onlineOrActiveOrConnectArrValue); 
                finalArr[dateIndex][type2].splice(axisDateIndex,1,offOnlineOrInactiveOrDisconnectValue); 
            }
        });
        var prevDayData = "";
        for(var key in finalArr ){
            if(prevDayData){
                ['Connected','Disconnected','Active','Inactive','Online','Offline'].map(function(itemCode){
                    if(finalArr[key][itemCode]){
                        finalArr[key][itemCode][finalArr[key][itemCode].length-1] = prevDayData[itemCode][0];
                    }
                });
                prevDayData = finalArr[key]
            }else prevDayData = finalArr[key]
        }
        
        return finalArr;
    }

    function getDays() {
        var lineDays = [],
            start = new Date();
        
        for(var i = 6; i>=0; i--) {
            var dateStr = dateformatter(addDate(start,-i));
            lineDays.push(dateStr.substr(0,10));
        }
        
        return lineDays;
    }
</script>