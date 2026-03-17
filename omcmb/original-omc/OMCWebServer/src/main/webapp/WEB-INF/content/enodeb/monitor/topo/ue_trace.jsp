<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
    #ue_trace_ctn {
        position: relative;
        width: 100%;
        height: 100%;
    }
    .left-fix-uelist {
        position: absolute;
        top: 20px;
        left: 20px;
        width: 200px;
        background-color: #fff;
        border-radius: 5px;
        z-index: 100;
        height: 90%;
        overflow: visible;
    }
    .ue-list-200 {
        position: absolute;
        left: 201px;
        top: 30px;
        background-color: #fff;
        border-radius: 5px;
    }
    .time-select-wrap {
        display: flex;
        flex-direction: column;
        height: calc(100% - 60px);
        overflow: auto;
    }
    .time-segment {
        position: relative;
        display: flex;
        align-items: center;
        padding: 5px 10px 5px 45px;
        color: rgba(0,0,0,0.8);
        font-size: 13px;
        cursor: pointer;
        border-radius: 2px;
    }
    .time-segment::before {
        display: block;
        content: '';
        position: absolute;
        top: 0;
        left: 25px;
        bottom: 0;
        width: 1px;
        background-color: #D8D8D8;
    }
    .time-segment::after {
        display: block;
        content: '';
        position: absolute;
        width: 5px;
        height: 5px;
        border-radius: 3px;
        background-color: #D8D8D8;
        left: 23px;
        top: 45%;
    }
    .time-segment:hover,
    .time-segment.selected {
        background-color: rgba(255, 70, 20, 0.1);
        color: rgba(255, 70, 20, 0.8);
    }
    .time-segment.selected::after {
        background-color: rgba(255, 70, 20, 1);
        color: rgba(255, 70, 20, 0.8);
    }
    .segment-op {
        position: absolute;
        right: 10px;
        color: rgba(0,0,0,0.32);
        font-size: 12px;
        z-index: 1;
    }
    .time-segment.selected .segment-op {
        color: #ff4614 !important;
    }
    .ue-total-line {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 10px 15px;
        border-bottom: 1px solid #D8D8D8;
    }

    
	#ue_trace_ctn .step-date-class {
		width: 100px;
		height: 30px;
		position: absolute;
		left: 0px;
		top: 0px;
		z-index: 99;
	}
	#ue_trace_ctn .step-date-class .el-input__inner {
		opacity: 0;
	}
	#ue_trace_ctn .step-date-class .el-input__icon {
		display: none;
	}	
	#ue_trace_ctn .time-step .el-icon:before {
		font-size: 16px !important;
	}
    .ue-circle {
        display: inline-block;
        width: 8px;
        height: 8px;
        border-radius: 4px;
        background-color: #67D972;
    }
    .ue-circle.normal {
        background-color: #F2B354;
    }
    .ue-circle.low {
        background-color: #E88282;
    }
    
	.hidden-ue-name .node-code {
		display: none;
	}
</style>

<div id="ue_trace_ctn" class="overflow-cls hidden-ue-name">
    <div class="left-fix-uelist">
        <div style="display: flex;align-items: center;justify-content: center;border-bottom: 1px solid #D8D8D8;">
            <div class="time-step" style="display: inline-block">
                <i class="el-icon el-icon-circle-left" @click="stepTimeChange(-1)" :class="{disabled: is30Day}"></i>
                <div style="position: relative; display: inline-block; height: 30px; line-height: 30px; width: 110px; text-align: center; cursor: pointer;">
                    <span style=' font-weight: bold; color:#363B4E; font-size: 14px; padding: 0;'>{{stepTime}}</span>
                    <el-date-picker ref="stepDate" :clearable="false" class="step-date-class"
                        :picker-options="pickerOptions"
                        @change="stepDateChange" 
                        v-model="stepDate" 
                        type="date">
                    </el-date-picker>
                </div>
                <i class="el-icon el-icon-circle-right" style="margin-left:0px;font-size: 20px;" :class="{disabled: isCurrentTime}" @click="stepTimeChange(1)"></i>
            </div>
        </div>
        <div class="time-select-wrap" @click="observSlider">
            <div class="time-segment" @click="selectSegment"> 00:00-01:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 01:00-02:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 02:00-03:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 03:00-04:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 04:00-05:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 05:00-06:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 06:00-07:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 07:00-08:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 08:00-09:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 09:00-10:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 10:00-11:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 11:00-12:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 12:00-13:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 13:00-14:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 14:00-15:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 15:00-16:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 16:00-17:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 17:00-18:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 18:00-19:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 19:00-20:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 20:00-21:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 21:00-22:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 22:00-23:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
            <div class="time-segment" @click="selectSegment"> 23:00-00:00 <i class="segment-op el-icon el-icon-arrow-right"></i> </div>
        </div>
        <div v-show="top200Show" class="ue-list-200">
            <div class="ue-total-line">
                <span style="font-size: 13px;">UE Top 200</span>
                <span>Total UE Count: {{total}} </span>
            </div>
            <el-ctable style="margin: 15px;"
                :data="top200List"
                :rownumber="true"
                :pagination="false"
                height="415"
            >
                <el-table-column label="UE ID" prop="ueId"></el-table-column>
                <el-table-column label="Count" prop="samplingPointCount"></el-table-column>
                <el-table-column label="RSRP" prop="avgRsrp"></el-table-column>
            </el-ctable>
        </div>
    </div>
    <div id="ue_map" style="background-color: rgb(161, 212, 224);width: 100%;"></div>
</div>

<script>
var globUEMap;

new Vue({
    el: '#ue_trace_ctn',
    data() {
        //当前时间 年-月-日 时-分-秒
        var endTime = formatDate(new Date(gloableTime));
        //图表的开始查询时间
        var startTime = formatDate(new Date(endTime)).substring(0,10); //年-月-日
        var periodChangeTime = startTime;//年-月-日

        var vm = this;

        return {
            sn: '',

            total: 0,
            top200Show: false,
            top200List: [],

            allNodes: [],
            smallCellCode: '',
            pickerOptions: {
                disabledDate: (time) => {
                    const today = new Date(gloableTime).setHours(0,0,0,0);
                    const thirtyDaysAgo = today - 30*24*3600*1000;
                    // 禁用范围
                    return time.getTime() < thirtyDaysAgo || time.getTime() > Date.now();
                }
            },
            form: {
                startTime: '',
                endTime: ''
            },
            stepDate: '',
            stepTime: startTime,
            startTime: startTime,
            is30Day: false
        };
    },
    computed: {
        //根据当前粒度，置灰 后一天 circle-right 图标
        isCurrentTime() {
            var vm = this, bool = true,
                stepStr = '',
                curStr ='';
            
            stepStr = (vm.stepTime||'').replace(/-/g,''),
            curStr = getYesterDay(0).replace(/-/g,'');
            
            if(stepStr-curStr<0) bool = false;

            return bool;
        },
        stepOptions() {
            var vm = this;

            return {
                disabledDate: function(time) {
                    var now = new Date(getYesterDay(0) + ' 00:00:00'),
                        nowTime = getWeekTime(vm.startTime),
                        endTime = new Date(nowTime.sundayTime + ' 00:00:00'); //周-结束日期

                    if(vm.chartPeriod == '1'){
                        var maxTime = endTime.getTime();

                        return time.getTime() > maxTime
                    }else{
                        return time.getTime() > now.getTime();
                    }
                }
            }
        },
        rsrp1() {
            return uersrp1;
        },
        rsrp2() {
            return uersrp2;
        }
    },
    methods: {
        init(sn) {
            var vm = this;

            vm.sn = sn;
            $('.time-segment:first-child').addClass("selected");
            var timeRange = $('.time-segment.selected').text().split('-');
            if(timeRange.length > 1) {
                vm.form.startTime = vm.stepTime + ' ' + timeRange[0].trim() + ':00';
                vm.form.endTime = vm.stepTime + ' ' + timeRange[1].trim() + ':00';
                vm.queryUE();
            }
        },
        // 初始化前一天 或后一天 时间切换
        stepTimeChange(num) {
            var vm = this,
                endTime = formatDate(new Date(gloableTime));
                stepTime = vm.stepTime; //当前时间点

            // 向后翻页：只检查是否到达当前日期
            if(num > 0 && vm.isCurrentTime) {
                return;
            }
            // 向前翻页：检查是否超过30天限制
            if(num < 0 && vm.is30Day) {
                return;
            }
            
            //后一天
            if(num == 1){
                //当前时间
                var nowTime = formatDate((new Date(endTime))).substring(0,10);
                //时间选择框呈现时间
                var nowDivTime = vm.stepTime;
                //选择时间和周期之后的开始时间
                var nextTime = "";
                
                //天
                nowDivTime += ' 00:00:00';
                nextTime = formatDate(addDate(new Date(nowDivTime),1)).substring(0,10);
                vm.stepTime = nextTime;
                vm.stepDate = nextTime;
                
                // 重新计算是否超过30天限制
                var diffDay = Math.floor( (new Date(endTime.substring(0,10)) - new Date(nextTime)) / (1000*60*60*24) );
                if(diffDay >= 30) {
                    vm.is30Day = true;
                }else {
                    vm.is30Day = false;
                }
            }else {
                //天 前一天点击
                var nowDivTime = vm.stepTime,
                    prevTime = '',
                    showNowTime;
                
                //天
                nowDivTime += ' 00:00:00';
                prevTime = formatDate(addDate(new Date(nowDivTime),-1)).substring(0,10);
                vm.stepTime = prevTime;
                vm.stepDate = prevTime;

                var diffDay = Math.floor( (new Date(endTime.substring(0,10)) - new Date(prevTime)) / (1000*60*60*24) );
                if(diffDay >= 30) {
                    vm.is30Day = true;
                }else {
                    vm.is30Day = false;
                }
            }
            vm.resetTop200();
            // 更新时间
            var selectedItem = $('.time-segment.selected');
            if(selectedItem.length == 0) {
                $('.time-segment:first-child').addClass("selected");
            }
            var timeRange = $('.time-segment.selected').text().split('-');
            if(timeRange.length > 1) {
                vm.form.startTime = vm.stepTime + ' ' + timeRange[0].trim() + ':00';
                vm.form.endTime = vm.stepTime + ' ' + timeRange[1].trim() + ':00';
                vm.queryUE();
            }
        },
        //日期组件中选择时间
        stepDateChange(val) {
            var vm = this, str = '';
            
            //天
            str = dateformatter(val).substring(0,10);
            vm.stepTime = str;
            vm.stepDate = str;
            vm.resetTop200();
            // 更新时间
            var selectedItem = $('.time-segment.selected');
            if(selectedItem.length == 0) {
                $('.time-segment:first-child').addClass("selected");
            }
            var timeRange = $('.time-segment.selected').text().split('-');
            if(timeRange.length > 1) {
                vm.form.startTime = vm.stepTime + ' ' + timeRange[0].trim() + ':00';
                vm.form.endTime = vm.stepTime + ' ' + timeRange[1].trim() + ':00';
                vm.queryUE();
            }

            var endTime = formatDate(new Date(gloableTime)),
                diffDay = Math.floor( (new Date(endTime.substring(0,10)) - new Date(vm.stepTime)) / (1000*60*60*24) );
            if(diffDay == 30) {
                vm.is30Day = true;
            }else {
                vm.is30Day = false;
            }
        },
        queryUE() {
            var vm = this,
                url = '${ctx}/cell/topo/queryUeIdGpsTrack.action',
                params = {
                    timeZone: timeZone,
                    operator_code: operator_code,
                    serialNumber: vm.sn,
                    startTime: vm.form.startTime,
                    endTime: vm.form.endTime
                };

            axios.post(url, stringify(params)).then(function(res) {
                var data = res.data || [];
                vm.allNodes = vm.transformNode(data);
                vm.initMap();
            });
        },
        resize() {
            var vm = this;

            if(globUEMap) {
                globUEMap.updateSize();
            }
        },
        initMap() {
            var vm = this,
                nodes = vm.allNodes;

            // 清除旧地图容器
            var mapContainer = document.getElementById('ue_map');
            if(mapContainer) {
                mapContainer.innerHTML = '';
            }
            
            // 清除旧地图对象
            if(globUEMap) {
                if(typeof globUEMap.setTarget === 'function') {
                    // OpenLayers 地图
                    globUEMap.setTarget(null);
                } else if(typeof globUEMap.remove === 'function') {
                    // Leaflet 地图
                    globUEMap.remove();
                }
                globUEMap = null;
            }
            
            // 处理基站数据格式
            var resObj = vm.proccessNodes(nodes);
            
            // 设置地图图层源
            var tileUrl = offlineMapEnable 
                ? '${ctx}/map/{z}/{x}/{y}.png'
                : 'https://{a-c}.tile.openstreetmap.org/{z}/{x}/{y}.png';
            
            // 创建瓦片图层
            var tileLayer = new ol.layer.Tile({
                source: new ol.source.XYZ({
                    url: tileUrl,
                    maxZoom: offlineMapEnable ? 12 : 18
                })
            });
            
            // 创建地图实例
            var uemap = new ol.Map({
                target: 'ue_map',
                layers: [tileLayer],
                view: new ol.View({
                    center: ol.proj.fromLonLat([resObj.lon, resObj.lat]),
                    zoom: 10,
                    maxZoom: offlineMapEnable ? 12 : 18,
                    minZoom: 4
                })
            });
            
            globUEMap = uemap;
            
            // 绘制 UE Trace 图层
            vm.createUELayer(resObj.ueNodes, uemap);
            
            // 自适应视图到节点边界
            if(resObj.ueNodes.length > 0) {
                var extent = ol.extent.boundingExtent(
                    resObj.ueNodes.map(function(node) {
                        return ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]);
                    })
                );
                uemap.getView().fit(extent, {
                    padding: [50, 50, 50, 50],
                    maxZoom: offlineMapEnable ? 12 : 16
                });
            }

            // 事件绑定
            uemap.on('moveend', function(ev) { // 移动结束后重新计算
                //vm.filterRender(resObj.ueNodes, globUEMap, true);
            });
        },
        proccessNodes(nodes) {
            var cLat = 0, cLon = 0, 
                latMax = -90, lonMax = -180,
                latMin = 90, lonMin = 180;
            
            nodes.map(function(item,index){
                var ilat = parseFloat(item.lat),
                    ilon = parseFloat(item.lon);
                if(ilat>latMax && Math.abs(ilat)<=90) latMax = ilat;
                if(ilon>lonMax && Math.abs(ilon)<=180) lonMax = ilon;
                if(ilat<latMin && Math.abs(ilat)<=90) latMin = ilat;
                if(ilon<lonMin && Math.abs(ilon)<=180) lonMin = ilon;
                
            });
            
            // 计算经纬中心点
            cLat = (latMax + latMin)/2;
            cLon = (lonMax + lonMin)/2;

            return {
                ueNodes: nodes,
                latMax: latMax,
                lat: cLat,
                lon: cLon,
                latdis: latMax - latMin,
                londis: lonMax - lonMin,
                latmax: latMax,
                latmin: latMin,
                lonmax: lonMax,
                lonmin: lonMin
            }
        },
        transformNode(nodeList){
            nodeList = nodeList.map(function(node){
                var lat = node.latitude,
                    lon = node.longitude;

                if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) {
                    lat = '';
                }
                if(isNaN(node.longitude) || [null,undefined].includes(node.longitude)) {
                    lon = '';
                }
                
                // 规范属性
                return {
                    code: node.ueId + '_' + node.latitude + '_' + node.longitude,
                    olat: node.latitude,
                    olon: node.longitude,
                    lat: lat,
                    lon: lon,
                    type: 'enb',
                    rsrp: node.rsrp
                };
            });
            // 过滤无经纬度的节点
            nodeList = nodeList.filter(function(item){
                return item.lat !== '' && item.lon !== '';
            });

            return nodeList;
        },
        createUELayer(nodes, map) {
            var vm = this;

            // 创建要素数组
            var features = nodes.map(function(node) {
                // 创建点要素
                var feature = new ol.Feature({
                    geometry: new ol.geom.Point(ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)])),
                    record: node
                });
                
                // 根据 RSRP 值确定颜色
                var rsrp = node.rsrp,
                    isHigh = rsrp - vm.rsrp2 >= 0,
                    isNormal = rsrp - vm.rsrp2 < 0 && rsrp - vm.rsrp1 >= 0,
                    isLow = rsrp - vm.rsrp1 < 0;
                
                var circleClass = isHigh ? 'ue-circle' : (isNormal ? 'ue-circle normal' : 'ue-circle low');
                
                // 设置样式
                feature.setStyle(new ol.style.Style({
                    image: new ol.style.Icon({
                        anchor: [0.5, 1.5],
                        anchorXUnits: 'fraction',
                        anchorYUnits: 'fraction',
                        src: 'data:image/svg+xml;utf8,' + encodeURIComponent(
                            '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16">' +
                            '<circle cx="8" cy="8" r="4" fill="' + 
                            (isHigh ? '#67D972' : (isNormal ? '#F2B354' : '#E88282')) + 
                            '"/></svg>'
                        )
                    })
                }));
                
                return feature;
            });
            
            // 创建矢量数据源
            var vectorSource = new ol.source.Vector({
                features: features
            });
            
            // 创建矢量图层
            var vectorLayer = new ol.layer.Vector({
                source: vectorSource
            });
            
            // 添加图层到地图
            map.addLayer(vectorLayer);
            
            // 调整窗口大小时更新地图
            $(window).resize();
        },
        selectSegment(evt) {
            var vm = this;

            if(evt.target.classList.contains('time-segment')) {
                $('.time-segment').removeClass('selected');
                evt.target.classList.add('selected');

                var timeRange = evt.target.innerText.split('-');
                Object.assign(vm.form, {
                    startTime: vm.stepTime + ' ' + timeRange[0].trim() + ':00',
                    endTime: vm.stepTime + ' ' + timeRange[1].trim() + ':00'
                });
                vm.queryUE();
                vm.resetTop200();
            }
            if(evt.target.parentNode.classList.contains('time-segment')) {
                var timeNode = evt.target.parentNode;
                $('.time-segment').removeClass('selected');
                timeNode.classList.add('selected');

                var timeRange = timeNode.innerText.split('-');
                Object.assign(vm.form, {
                    startTime: vm.stepTime + ' ' + timeRange[0].trim() + ':00',
                    endTime: vm.stepTime + ' ' + timeRange[1].trim() + ':00'
                });
                vm.queryUE();
            }
        },
        observSlider(ev){// 通过事件代理，将逻辑抽离到父级，避免动态节点时，子节点的复杂绑定逻辑
            var vm = this,
                target = ev.target,
                clsList = target.classList;

            /* 切换当前行箭头状态 */
            if(clsList.contains('el-icon-arrow-right') || clsList.contains('el-icon-arrow-left')) {
                if(clsList.contains('el-icon-arrow-right')) {
                    vm.showDevice(ev,true);
                }else {
                    vm.showDevice(ev,false);
                }

                ['el-icon-arrow-right','el-icon-arrow-left'].map(function(cls){
                    if(clsList.contains(cls)) clsList.remove(cls);
                    else clsList.add(cls);
                });

                /* 重置其他行箭头状态 */
                Array.from(document.querySelectorAll('.segment-op')).map(function(item){
                    if(item != target) {
                        item.classList.remove('el-icon-arrow-left');
                        item.classList.add('el-icon-arrow-right');
                    }
                });
            }

            //ev.stopPropagation();
        },
        resetTop200() {
            var vm = this;

            vm.top200Show = false;
            /* 重置其他行箭头状态 */
            Array.from(document.querySelectorAll('.segment-op')).map(function(item){
                item.classList.remove('el-icon-arrow-left');
                item.classList.add('el-icon-arrow-right');
            });
        },
        showDevice(evt, bool) {
            var vm = this,
                url = '${ctx}/cell/topo/queryHourlyDataBySnAndTimeRange.action',
                timeRange = evt.target.parentNode.innerText.split('-'),
                params = {
                    timeZone: timeZone,
                    serialNumber: vm.sn,
                    startTime: vm.stepTime + ' ' + timeRange[0].trim() + ':00',
                    endTime: vm.stepTime + ' ' + timeRange[1].trim() + ':00'
                };

            vm.top200Show = bool;
            vm.top200List = [];
            axios.post(url, stringify(params)).then(function(res) {
                var data = res.data || [];

                vm.top200List = data;
            });
            // 时段UE总数
            var totalUrl = '${ctx}/cell/topo/countUeIdBySnAndTimeRange.action';
            axios.post(totalUrl, stringify(params)).then(function(res) {
                var data = res.data || 0;

                vm.total = data;
            });
        }
    },
    mounted() {
        var vm = this;

        $('#ue_map').off('click').on('click', function(evt){
            vm.resetTop200();
        });

        eventBus.$off("eu-trace").$on("eu-trace", vm.init);
    }
})
</script>