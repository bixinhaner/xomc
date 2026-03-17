<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#enbDiagnosticPage {
	width: 100%;
	height:100%;
    background: #FFFFFF;
}
#enbDiagnosticPage .enbDiagnosticMainBoxCls{
	display:flex;
	flex-direction:column;
    padding: 20px;
}
#enbDiagnosticPage .enbDiagnosticSelectBoxCls{
    display: flex;
}
#enbDiagnosticPage .enbDiagnosticSelectCls{
    width: 100%;
}
#enbDiagnosticPage .enbDiagnosticStartButton{
	height:40px;
    width: 70px;
	border-radius:0px 4px 4px 0px;
}
#enbDiagnosticPage .enbDiagnosticSelectBoxCls .el-input{
	width: 100%;
	height:40px;
}
#enbDiagnosticPage .enbDiagnosticSelectBoxCls .el-input .el-input__inner{
	width: 100%;
	height:40px;
	font-weight:bold;
	font-size:14px;
	color:#333;
}
#enbDiagnosticPage .enbDiagnosticLimitBoxCls{
    display: flex;
    justify-content: space-around;
    margin: 50px 0px 20px 0px;
}
#enbDiagnosticPage .enbDiagnosticLimitValCls{
    width: 220px;
    display: flex;
    align-items: center;
}
#enbDiagnosticPage .itemIconBoxCls{
    margin-right: 5px;
}
#enbDiagnosticPage .itemIconBoxCls .itemDownIconCls::before{
    color: #19D0F4;
    font-size: 30px!important;
}
#enbDiagnosticPage .itemIconBoxCls .itemUpIconCls::before{
    color: #0099FF;
    font-size: 30px!important;
}
#enbDiagnosticPage .itemIconBoxCls .itemPingTimeIconCls::before{
    color: #FFB64D;
    font-size: 30px!important;
}
#enbDiagnosticPage .itemTitleAndValCls .itemTitleCls{
   color: #777F8E;
   font-size: 14px;
}
#enbDiagnosticPage .itemTitleAndValCls .itemValCls{
    color: rgba(0, 0, 0, 0.8);
    font-size: 14px;
}
#enbDiagnosticPage .enbDiagnosticSpeedTestBoxCls{
    padding: 10px;
    border: 1px solid #E9E9E9;
    height: 400px;
    box-sizing: border-box;
}
#enbDiagnosticPage .enbDiagnosticSpeedTestTitleCls{
    color: #333;
    font-size: 14px;
    font-weight: bold;
}
.enbDiagnosticSpeedTestChartsCls {
    display: flex;
    height: calc(100% - 40px);
}
.enbDiagnosticSpeedTestChartsCls .enbDiagnosticGaugeChartsBoxCls{
    width:40%;
    height:100%;
    position: relative;
}
.enbDiagnosticSpeedTestChartsCls .enbDiagnosticLineChartsBoxCls{
    width:60%;
    height:100%;
}
.enbDiagnosticLineChartsItemCls{
    width:100%;height:10%;
}
</style>

<div id="enbDiagnosticPage">
	<div class="enbDiagnosticMainBoxCls">
        <div class="enbDiagnosticSelectBoxCls">
            <el-select v-model="testVal" class='enbDiagnosticSelectCls' placeholder="<%=rb.getString("ShuRuFuWuQiDiZhi")%>" @change="testValChange">
                <el-option v-for="item in pingList" :key="item.value" :value="item.value" :label="item.text" ></el-option>
            </el-select>
            <el-button type="primary" class='enbDiagnosticStartButton' @click="startTest" :disabled="!testVal" :loading="startTestLoading">{{buttonTitle}}</el-button>
        </div>
        <div class="enbDiagnosticLimitBoxCls">
            <div class="enbDiagnosticLimitValCls">
                <div class="itemIconBoxCls">
                    <span class="el-icon el-icon-down1 itemDownIconCls"></span>
                </div>
                <div class="itemTitleAndValCls">
                    <div class="itemTitleCls">Download</div>
                    <div class="itemValCls">{{pingDownLimitVal}}Mbps</div>
                </div>
            </div>
            <div class="enbDiagnosticLimitValCls">
                <div class="itemIconBoxCls">
                    <span class="el-icon el-icon-up1 itemUpIconCls"></span>
                </div>
                <div class="itemTitleAndValCls">
                    <div class="itemTitleCls">Upload</div>
                    <div class="itemValCls">{{pingUpLimitVal}}Mbps</div>
                </div>
            </div>
            <div class="enbDiagnosticLimitValCls">
                <div class="itemIconBoxCls">
                    <span class="el-icon el-icon-operation-time itemPingTimeIconCls"></span>
                </div>
                <div class="itemTitleAndValCls">
                    <div class="itemTitleCls">Ping</div>
                    <div class="itemValCls">{{pingTimes}}ms</div>
                </div>
            </div>
        </div>
        <div class="enbDiagnosticSpeedTestBoxCls">
            <div class="enbDiagnosticSpeedTestTitleCls">Speed Test</div>
            <div class="enbDiagnosticSpeedTestChartsCls">
                <div class="enbDiagnosticGaugeChartsBoxCls">
                    <div id="enbDiagnosticGaugeCharts" style="width:100%;height:100%;"></div>
                </div>
                <div class="enbDiagnosticLineChartsBoxCls">
                    <div class="enbDiagnosticLineChartsItemCls">
                        <span style="margin-right: 10px;">Download/Mbps</span>
                        <span style="color:#19D0F4;font-size:24px;">{{activeDownLimitVal}}</span>
                        <span style="color:#19D0F4;font-size:14px;">Mbps</span>
                    </div>
                    <div id="enbDiagnosticDownLineCharts" style="width:100%;height:40%;"></div>
                    <div class="enbDiagnosticLineChartsItemCls">
                        <span style="margin-right: 10px;">Upload/Mbps</span>
                        <span style="color:#19D0F4;font-size:24px;">{{activeUpLimitVal}}</span>
                        <span style="color:#19D0F4;font-size:14px;">Mbps</span>
                    </div>
                    <div id="enbDiagnosticUpLineCharts" style="width:100%;height:40%;"></div>
                </div>
            </div>
        </div>
    </div>
</div>

<script> 
if(window.enbDiagnosticPageVue) {
    try {
        window.enbDiagnosticPageVue.$destroy();
    }catch(e){}
}
var updateGaugeChartsTimer = null;
window.enbDiagnosticPageVue = new Vue({
	el: '#enbDiagnosticPage',  
	data() {
		return {
            enbSelectedRow:{},
            smallCellCode:'',
			testVal:'',
            pingList:[], //ping列表
            buttonTitle:'Start',
            pingDownLimitVal:'0',
            pingUpLimitVal:'0',
            pingTimes:'0',
            activeDownLimitVal:'0',
            activeUpLimitVal:'0',
            chartList: ['enbDiagnosticGaugeCharts','enbDiagnosticDownLineCharts','enbDiagnosticUpLineCharts'],
			charts: {},
            startTestLoading:false
		};
	},
	computed: {
		isCloud() {
			return isCloud == 'true';
		},
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		
	},
	watch:{
		
	},
	methods: {
        // 初始化请求数据
		init(row,smallCellCode,sn,status,version,product,licFlag){
            var vm = this;
            vm.smallCellCode = smallCellCode;
            vm.enbSelectedRow = row;
            vm.initCharts();
            vm.getServer();
            // 窗口缩放时自适应
            window.removeEventListener('resize',vm.resizeChart);
            window.addEventListener('resize',vm.resizeChart);

        },
        // 初始化图表
        initCharts(){
            var vm = this;
            vm.pingDownLimitVal = '0';
            vm.pingUpLimitVal = '0';
            vm.pingTimes = '0';
            vm.activeDownLimitVal = '0';
            vm.activeUpLimitVal = '0';
            vm.initGaugeCharts(0);
            vm.initDownLineCharts([0]);
            vm.initUpLineCharts([0]);
        },
        // 获取测试服务器列表
        getServer(){
            var vm = this,
                params = {};

            axios.post("${ctx}/cell/quicksettings/getSpeedTestHostSelectList.action", stringify(params)).then(res=>{
                vm.pingList = res.data?res.data:[];
            })
        },
        // 所有图表自适应
		resizeChart(){
			var vm = this;
			vm.chartList.map(function(code){
				if(vm.charts[code]) vm.charts[code].resize();
			});
		},
        // 测试服务器改变
        testValChange(val){
            var vm = this;
            vm.startTestLoading = false;
        },
        /**
        * 统计数据刷新
        * @param code{string} 图表类型
        * @param index{number}  时间轴下标
        */
        proccessChartData(code){
            var vm = this,
                chartData={
                    downData: [],
                    upData: [],
                },
                params = {
                    speedHost: vm.testVal,
                    smallCellCode: vm.smallCellCode,
                };
            // 图标数据
            axios.post("${ctx}/cell/quicksettings/addVelocimetryTaskCommand.action",stringify(params)).then(function(response){
                var data = response.data;

                if(data && data['success']){
                    vm.startTestLoading = true;
                    vm.initCharts();
                    if(updateGaugeChartsTimer){
                        clearInterval(updateGaugeChartsTimer);
                    }
                    updateGaugeChartsTimer = setInterval(function(){
                        vm.getVelocimetryInfo();
                    },6000);
                    
                }
            }) 
        },
        getVelocimetryInfo(){
            var vm = this,
                params = {
                    smallCellCode: vm.smallCellCode,
                };
            // 图标数据
            axios.post("${ctx}/cell/quicksettings/getVelocimetryInfo.action",stringify(params)).then(function(response){
                var data = response.data;

                if(data && data.downloadSpeed){
                    if(data.downloadSpeed == '--'){
                        clearInterval(updateGaugeChartsTimer);
                        vm.startTestLoading = false;
                    }else{
                        var downData = data.downloadSpeed ? data.downloadSpeed.split(',') : [],
                            upData = data.uploadSpeed ? data.uploadSpeed.split(',') : [];
                        vm.pingTimes = data.delayTime ? data.delayTime : '0';
                        if(downData.length > 0){
                            vm.activeDownLimitVal = downData[downData.length-1];
                            vm.pingDownLimitVal = downData[downData.length-1];
                        }
                        if(upData.length > 0){
                            vm.activeUpLimitVal = upData[upData.length-1];
                            vm.pingUpLimitVal = upData[upData.length-1];
                        }
                        vm.setOptionGauge(downData[downData.length-1]);
                        setTimeout(function(){
                            vm.setOptionGauge(0);
                        },6000);
                        vm.initDownLineCharts(downData);
                        vm.initUpLineCharts(upData);
                        clearInterval(updateGaugeChartsTimer);
                        vm.startTestLoading = false;
                    }
                }
            }) 
        },
        // 仪表盘图表生成
        initGaugeCharts(val){
            var vm = this;
                chartDom = document.getElementById('enbDiagnosticGaugeCharts'),
                color = '';
            vm.charts['enbDiagnosticGaugeCharts'] = echarts.init(chartDom);
            vm.charts['enbDiagnosticGaugeCharts'].clear();
            var option = {
                    series:[{
                        type:'gauge',
                        animationDuration: 6000,
                        animationEasing: "cubicInOut",
                        animationDurationUpdate:6000,
                        animationEasingUpdate: "cubicInOut",
                        max:1000,
                        itemStyle:{
                            normal: {
                                color:'#19D0F4'
                            }
                        },
                        progress:{
                            show:true,
                            width:18,
                        },
                        axisLine:{
                            lineStyle:{
                                width:18,
                                color:[[1,'#ECF0FA']]
                            }
                        },
                        axisTick:{
                            show:false
                        },
                        splitLine:{
                            length:8,
                            distance:5,
                            lineStyle:{
                                width:1,
                                color:'#707070'
                            }
                        },
                        axisLabel:{
                            distance:25,
                            color:'#000',
                            fontSize:12
                        }, 
                        anchor:{
                            show:true,
                            showAbove:true,
                            size:23,
                            itemStyle:{
                                color:'#19D0F4'
                            }
                        },
                        pointer:{
                            length:'50%',
                        },
                        detail:{
                            fontSize:30,
                            formatter:'{value}',
                            offsetCenter:[0,'70%'],
                            color:'#000'
                        },
                        title:{
                            show:true,
                            offsetCenter:[0,'120'],
                            color:'#333',
                            fontSize:18
                        },
                        data:[{
                            value:val,
                            name:'Mbps',
                        }]
                    }]
            }
            option && vm.charts['enbDiagnosticGaugeCharts'].setOption(option);
        },
        // 下行折线图表生成
        initDownLineCharts(dataList){
            var vm = this;
                chartDom = document.getElementById('enbDiagnosticDownLineCharts');
            vm.charts['enbDiagnosticDownLineCharts'] = echarts.init(chartDom);
            vm.charts['enbDiagnosticDownLineCharts'].clear();
            var option = {
                xAxis: {
                  type: 'category',
                  axisLine: { show: false },   // 不显示 x 轴线条
                  splitLine: { show: false },   // 不显示分割线
                  axisTick: { show: false },     // 不显示刻度标记
                  data: []
                },
                grid:{
                    bottom: 0,
                    left:0,
                    right: 0,
                    top: 20
                },
                yAxis: {
                  type: 'value',
                   axisTick: {
                      show: false,  //刻度线
                    },
                    axisLine: {
                      show: false, //隐藏y轴
                    },
                    axisLabel: {
                      show: false, //隐藏刻度值
                    },
                    splitLine: {
                      show: true,
                      lineStyle: {
                        type: 'dashed',
                      },
                    }
                },
                series: [
                  {
                    data: dataList,
                    type: 'line',
                    smooth: true,
                    symbolSize:0,
                    animationDuration: 4000,
                    animationEasing: "linear",
                    itemStyle: {
                        normal: {
                            lineStyle: {
                                // 线条加阴影
                                // 设置阴影颜色
                                width: 3 ,
                                shadowColor: '#19D0F4',
                                shadowOffsetX: 0,
                                // 设置阴影沿y轴偏移量为20
                                shadowOffsetY: 0,
                                // 设置阴影的模糊大小
                                shadowBlur: 30,
                                color:'#19D0F4',
                            },
                            label: {
                                show: true,
                                formatter: function (params) {
                                    return params.value[0];
                                }
                            },
                        }
                    },
                  }
                ]
              };
            option && vm.charts['enbDiagnosticDownLineCharts'].setOption(option);
        },
        // 上行折线图表生成
        initUpLineCharts(dataList){
            var vm = this;
                chartDom = document.getElementById('enbDiagnosticUpLineCharts');
            vm.charts['enbDiagnosticUpLineCharts'] = echarts.init(chartDom);
            vm.charts['enbDiagnosticUpLineCharts'].clear();
            var option = {
                xAxis: {
                  type: 'category',
                  axisLine: { show: false },   // 不显示 x 轴线条
                  splitLine: { show: false },   // 不显示分割线
                  axisTick: { show: false },     // 不显示刻度标记
                  data: []
                },
                grid:{
                    bottom: 0,
                    left:0,
                    right: 0,
                    top: 20
                },
                yAxis: {
                  type: 'value',
                   axisTick: {
                      show: false,  //刻度线
                    },
                    axisLine: {
                      show: false, //隐藏y轴
                    },
                    axisLabel: {
                      show: false, //隐藏刻度值
                    },
                    splitLine: {
                      show: true,
                      lineStyle: {
                        type: 'dashed',
                      },
                    }
                },
                series: [
                  {
                    data: dataList,
                    type: 'line',
                    smooth: true,
                    symbolSize:0,
                    animationDuration: 4000,
                    animationEasing: "linear",
                    itemStyle: {
                        normal: {
                            lineStyle: {
                                // 线条加阴影
                                // 设置阴影颜色
                                width: 3 ,
                                shadowColor: '#0099FF',
                                shadowOffsetX: 0,
                                // 设置阴影沿y轴偏移量为20
                                shadowOffsetY: 0,
                                // 设置阴影的模糊大小
                                shadowBlur: 30,
                                color:'#0099FF',
                            },
                            label: {
                                show: true,
                                formatter: function (params) {
                                    return params.value[0];
                                }
                            },
                        }
                    },
                    endLabel: {
                      show: true,
                      opacity:0,
                      formatter: function (params) {
                        return params.value[0];
                      }
                    },
                    labelLayout: {
                      moveOverlap: 'shiftY'
                    },
                  },
                ]
              };
            option && vm.charts['enbDiagnosticUpLineCharts'].setOption(option);
        },
        // 开始测试
        startTest(){
            var vm = this;
            vm.proccessChartData();
            
        },
        setOptionGauge(val){
            var vm = this;
                option = {
                    series:[{
                        type:'gauge',
                        animationDuration: 6000,
                        animationEasing: "cubicInOut",
                        animationDurationUpdate:6000,
                        animationEasingUpdate: "cubicInOut",
                        max:1000,
                        itemStyle:{
                            normal: {
                                color:'#19D0F4'
                            }
                        },
                        progress:{
                            show:true,
                            width:18,
                        },
                        axisLine:{
                            lineStyle:{
                                width:18,
                                color:[[1,'#ECF0FA']]
                            }
                        },
                        axisTick:{
                            show:false
                        },
                        splitLine:{
                            length:8,
                            distance:5,
                            lineStyle:{
                                width:1,
                                color:'#707070'
                            }
                        },
                        axisLabel:{
                            distance:25,
                            color:'#000',
                            fontSize:12
                        }, 
                        anchor:{
                            show:true,
                            showAbove:true,
                            size:23,
                            itemStyle:{
                                color:'#19D0F4'
                            }
                        },
                        pointer:{
                            length:'50%',
                        },
                        detail:{
                            fontSize:30,
                            formatter:'{value}',
                            offsetCenter:[0,'70%'],
                            color:'#000'
                        },
                        title:{
                            show:true,
                            offsetCenter:[0,'120'],
                            color:'#333',
                            fontSize:18
                        },
                        data:[{
                            value:val,
                            name:'Mbps',
                        }]
                    }]
                }
            option && vm.charts['enbDiagnosticGaugeCharts'].setOption(option);
        }
	},
	mounted() {
		eventBus.$off("enb-data").$on("enb-data",this.init)
	}
});

</script>
