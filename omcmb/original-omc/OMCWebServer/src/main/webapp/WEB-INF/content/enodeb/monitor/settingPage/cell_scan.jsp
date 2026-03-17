<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<div id="enb_cell_scan" :class="{'loading': isPageLoading}" style='display:flex;flex-direction:column;height: 100%;background-color: #fff;border-radius: 3px;'>
    <el-form class="flex-form" style="padding: 20px 10px 10px;display: flex;align-items: center;">
        <el-form-item label="<%=rb.getString("ShangBaoKaiGuan")%>" style="width: 300px;">
            <el-select v-model="scanEnable">
                <el-option label="OFF" value="0"></el-option>
                <el-option label="No-load Mode" value="1"></el-option>
                <el-option label="Regular Mode" value="2"></el-option>
                <el-option v-if="isBase" label="Full band" value="3"></el-option>
            </el-select>
        </el-form-item>

        <span v-if="scanEnable == '3'" style="font-size: 12px;color: #bbb;padding-top: 5px;">
            <i class="el-icon el-icon-circle-info"></i>
            <%=rb.getString("SaoMiaoDengDaiTiShi")%>
        </span>
    </el-form>

    <div style="padding: 0px 0px 20px 40px;">
        <el-button @click="setScanEnable" type="primary" size="mini"><%=rb.getString("QueDing")%></el-button>
    </div>

    <div v-show="scanForm.mode != '0'" id="scanchart" style="height: 500px;width: 100%;"></div>
</div>

<script>
    new Vue({
        el: '#enb_cell_scan',
        data() {

            return {
                selectedRow: settingVue.selectedRow,
                smallCellCode: settingVue.small_cell_code,
                scanChart: null,
                scanEnable: '0',
                scanForm: {
                    mode: '0'
                },
                isPageLoading: true
            }
        },
        computed: {
            isBase() {

                return this.selectedRow.dual_carrier_type != 2;
            },
            bandMap() {
                return {
                    0: [824, 849],
                    1: [1920, 1980],
                    2: [1850, 1910],
                    3: [1710, 1785],
                    4: [1710, 1755],
                    5: [824, 849],
                    6: [830, 840],
                    7: [2500, 2570],
                    8: [880, 915],
                    9: [1749.9, 1784.9],
                    10: [1710, 1770 ],
                    11: [1427.9, 1452.9],
                    12: [699, 716],
                    13: [777, 787],
                    14: [788, 798],
                    17: [704, 716],
                    18: [815, 830],
                    19: [830, 845],
                    20: [832, 862],
                    21: [1447.9, 1462.9],
                    22: [3410, 3490],
                    23: [2000, 2020],
                    24: [1626.5, 1660.5],
                    25: [1850, 1915],
                    26: [814, 849],
                    27: [807, 824],
                    28: [703, 748],
                    30: [2305, 2315],
                    31: [452.5, 457.5],
                    33: [1900, 1920],
                    34: [2010, 2025],
                    35: [1850, 1915],
                    36: [1930, 1995],
                    37: [1910, 1930],
                    38: [2570, 2620],
                    39: [1880, 1920],
                    40: [2300, 2400],
                    41: [2496, 2690],
                    42: [3400, 3600],
                    43: [3600, 3800],
                    44: [703, 803],
                    45: [1447.9, 1462.9],
                    46: [5150, 5925],
                    47: [5855, 5925],
                    48: [3550, 3700],
                    49: [4400, 4500],
                    50: [1432, 1517],
                    51: [1427.9, 1447.9],
                    52: [3300, 3800],
                    53: [2483.5, 2495.5],
                    65: [1920, 2010],
                    66: [1710, 1780],
                    67: [738, 758],
                    68: [698, 716],
                    70: [1695, 1710],
                    71: [663, 698 ],
                    72: [5450, 5725], 
                    73: [5735, 5850],
                    74: [1427.9, 1470.9],
                    75: [1432, 1517],
                    76: [1427.9, 1447.9],
                    77: [3300, 4200],
                    78: [3300, 3800],
                };
            }
        },
        methods: {
            init() {
                var vm = this;

                vm.scanChart = echarts.init(document.querySelector('#scanchart'));
                // 窗口缩放时自适应
                window.removeEventListener('resize',vm.resizeChart);
                window.addEventListener('resize',vm.resizeChart);
                vm.getScanInfo();
            },
            // 图表自适应
            resizeChart(){
                var list = ['scanchart'];
                
                list.map(function(id){
                    var dom = document.querySelector('#'+id);
                    if(dom) {
                        var itn = echarts.getInstanceByDom(dom);
                        itn && itn.resize();
                    }
                })
            },
            getScanInfo() {
                var vm = this,
                    url = '${ctx}/cell/scan/queryScanEnable.action',
                    params = {
                        smallCellCode: vm.smallCellCode
                    };
                
                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data;

                    vm.scanForm.mode = data.scanEnable;
                    vm.scanEnable = data.scanEnable;
                    if(data.scanEnable !== '0') {
                        // 等待 v-show 切换完成后再获取数据
                        vm.$nextTick(function() {
                            setTimeout(function() {
                                vm.getScanChartData();
                            }, 100);
                        });
                    }
                    vm.isPageLoading = false;
                });
            },
            setScanEnable() {
                var vm = this;

                var url = '${ctx}/cell/scan/setScanAbility.action',
                    params = {
                        smallCellCode: vm.smallCellCode,
                        switchEnable: vm.scanEnable
                    };
                
                vm.isPageLoading = true;

                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data;

                    if(data == true) {
                        vm.$message({
                            message: '<%=rb.getString("XiaFaChengGongZhuangTai")%>',
                            type:'success',
                        });

                        if(vm.scanForm.mode != vm.scanEnable) {
                            vm.scanForm.mode = vm.scanEnable;
                            // 等待 v-show 切换完成后再获取数据和调整图表
                            vm.$nextTick(function() {
                                setTimeout(function() {
                                    vm.resizeChart();
                                    vm.getScanChartData();
                                }, 100);
                            });
                        }else {
                            vm.isPageLoading = false;
                        }
                    }else {
                        vm.$message({
                            message: '<%=rb.getString("ShiBai")%>',
                            type:'error',
                        });
                        vm.isPageLoading = false;
                    }
                });
            },
            getScanChartData() {
                var vm = this,
                    mode = vm.scanForm.mode,
                    url = '${ctx}/cell/scan/queryScanData.action',
                    params = {
                        smallCellCode: vm.selectedRow.small_cell_code
                    };
                if(mode == '0') {
                    vm.isPageLoading = false;
                    return;
                }
                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data,
                        map = {
                            band: data.band,
                            xLabel: data.xData ? data.xData.split(',') : [],
                            data0: data.ant0Data ? data.ant0Data.split(',').map(function(item){ return item - 0;}) : [],
                            data1: data.ant1Data ? data.ant1Data.split(',').map(function(item){ return item - 0;}) : [],
                            base: []
                        };

                    for(let i=0; i< map.xLabel.length; i++) {
                        map.base.push(-118);
                    }

                    var options = vm.createScanOps(map);
                    
                    // 等待 DOM 更新后再设置图表和调整大小
                    vm.$nextTick(function() {
                        vm.scanChart.setOption(options);
                        vm.resizeChart();
                        // 额外延迟一次 resize 确保容器完全展开
                        setTimeout(function() {
                            vm.resizeChart();
                        }, 100);
                    });
                    
                    vm.isPageLoading = false;
                });
            },
            createScanOps(params) {
                var vm = this,
                    mode = vm.scanForm.mode, // 0--off 3--Full Band, 1-- 2--
                    row = vm.selectedRow,
                    band = params.band,
                    earfcn = (row.EARFCNDLINUSE||'').split(',')[0],
                    bandwidth = (row.bandwidth||'').replace('MHz',''),
                    freqStr = earfcnFormatter(earfcn, row),
                    m = freqStr.match(/\d+\(([0-9\.]+)MHz\)/),
                    freq = m?m[1]:'',
                    begin = '',
                    end = '';

                if(mode == '3') {
                    let range = vm.bandMap[band];

                    if(range) {
                        begin = range[0] + 'MHz';
                        end = range[1] + 'MHz';
                    }
                    /*
                    begin = freq -75 + 'MHz';
                    end = freq-0 + 75 + 'MHz';
                    */
                }else if(mode == '1' || mode == '2') {
                    begin = freq - bandwidth/2 + 'MHz';
                    end = freq-0 + bandwidth/2 + 'MHz';
                }

                var vm = this,
                    options = {
                        title: {
                            text: 'UL PRB RSSI',
                            subtext: '       ' + begin,
                            subtextStyle: {
                                color: '#666'
                            },
                            left: 20,
                            textStyle: {
                                fontSize: 14
                            }
                        },
                        grid: {
                            left: 80,
                            right: 80
                        },
                        color: ['#fc5959','#ff973e','#ffda41'],
                        tooltip: {
                            trigger: 'axis'
                        },
                        legend: {
                            data: ['Antenna0','Antenna1','BaseLine']
                        },
                        xAxis: {
                            type: 'category',
                            data: params.xLabel,
                            name: end,
                            nameTextStyle: {
                                padding: [0,0,50,0],
                                verticalAlign: 'bottom',
                            },
                            position: 'top',
                            boundaryGap: false,
                            axisLine:{
                                show : true,
                                lineStyle:{ color:"#666666" }
                            },
                        },
                        yAxis: {
                            type: 'value',
                            name: 'dBm/PRB',
                            nameLocation: 'middle',
                            nameGap: 60,
                            nameRotate: 90,
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
                        },
                        series: [
                            {
                                name: 'Antenna0',
                                type: 'line',
                                smooth: true,
                                data: params.data0
                            },
                            {
                                name: 'Antenna1',
                                type: 'line',
                                smooth: true,
                                data: params.data1
                            },
                            {
                                name: 'BaseLine',
                                type: 'line',
                                smooth: true,
                                data: params.base.length?params.base:[-118]
                            }
                        ]
                    };

                return options;
            },
        },
        mounted() {
            var vm = this;
            vm.init();
            if(window.enbCellScanTimer) {
                clearInterval(window.enbCellScanTimer)
            }
            window.enbCellScanTimer = setInterval(function(){
                var enbCellScanPageCtn = $("#enb_cell_scan");			
                if(!enbCellScanPageCtn.length) {
                    clearInterval(window.enbCellScanTimer);
                    return;
                }
                var ctner = document.querySelector('#enb_cell_scan'),
                    visible = isVisible(ctner),
                    isCovered = isOverlapped(ctner);
                
                if(visible && !isCovered) {
                    vm.getScanChartData();
                }
            },6000);
        }
    })
</script>