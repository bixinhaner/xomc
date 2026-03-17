// GoJS Diagram 全局变量声明
var myMapSiteTopoDiagram = null;

/**
* 刷新topo图 -- 节点数据变动
* @param obj{object}：信息有变动的节点
**/
function refreshTopo(obj) {
    let vm = topovm,
        latLonList = vm.allNodes.map(function(node){ return node.lat+'-'+node.lon; }),
        index = -1;
    
    vm.allNodes.map(function(item,idx){
        if(item.cellCode == obj.cellCode) {
            Object.assign(item, obj);
            index = idx;
        }
    });
}
/**
* 判空
* @param val{string}：要判断的值
**/
function isNotNull(val){
    let bool = false;
    if(val !== null && val !== '' && val != undefined) bool = true;
    return bool;
}
/**
* enb数据规范化 -- {code: '',lat: '',lon: '',name: '',type: '',online: '',active: '', ... }为必要属性
* @param nodeList{array}：enb节点原始数据
**/
function transformNode(nodeList){
    let latLons = nodeList.map(function(node){ return node.latitude+'-'+node.longitude; });
    
    nodeList = nodeList.map(function(node){
    let enable = node.mme_enable == 1? true:false,
        mmePool_1 = node.mme_pool_1?node.mme_pool_1:'',
        mmePool_2 = node.mme_pool_2?node.mme_pool_2:'',
        serverList = node.s1siglinkserverlist?node.s1siglinkserverlist:'',
        lat = node.latitude,
        lon = node.longitude,
        height = node.height,
        // 读取扇面参数，如果是测试节点且字段为空，使用默认测试值
        meDowntilt = node.mechanical_downtilt || (node.serial_number && node.serial_number.indexOf('testsn') === 0 ? 3 : ''),
        elDowntilt = node.electronic_downtilt || (node.serial_number && node.serial_number.indexOf('testsn') === 0 ? 6 : ''),
        vertical3dB = node.vertical_3dB_beam_width || (node.serial_number && node.serial_number.indexOf('testsn') === 0 ? 5 : ''),
        horizontal_azimuth = node.horizontal_azimuth,
        radius = 0,
        minRadius = 0,
        ueCount = node.ue_count,
        hasMDT = node.isHasMDTData == true,
        isGSM = node.stationType == 'GSM',
        stationType = node.stationType == 'gNB' ? 'gnb' : (node.stationType == 'GSM' ? 'gsm' : 'enb');

        // 计算半径
        if(height && meDowntilt && elDowntilt && vertical3dB) {
            // 计算总下倾角和半功率波束宽度
            let totalDowntilt = meDowntilt*1 + elDowntilt*1;
            let halfBeamWidth = vertical3dB / 2;
            
            // 外半径：下倾角 - 半功率波束宽度的一半
            let outerAngle = totalDowntilt - halfBeamWidth;
            // 内半径：下倾角 + 半功率波束宽度的一半
            let innerAngle = totalDowntilt + halfBeamWidth;				// 确保角度在合理范围内（避免tan值异常）
            if (outerAngle > 0 && outerAngle < 90) {
                let outerRad = (outerAngle * Math.PI) / 180;
                radius = Math.abs(height / Math.tan(outerRad));
            } else {
                radius = 500; // 默认500米
            }
            
            if (innerAngle > 0 && innerAngle < 90) {
                let innerRad = (innerAngle * Math.PI) / 180;
                minRadius = Math.abs(height / Math.tan(innerRad));
            } else {
                minRadius = 100; // 默认100米
            }
            
            // 限制半径在合理范围内
            if (radius > 10000) radius = 10000; // 最大10km
            if (radius < 50) radius = 50;       // 最小50m
            if (minRadius > radius) minRadius = radius * 0.2; // 内半径不应大于外半径
            if (minRadius < 10) minRadius = 10; // 最小10m
            
            // 四舍五入保留2位小数（转换为数值类型）
            radius = parseFloat(radius.toFixed(2));
            minRadius = parseFloat(minRadius.toFixed(2));
        }
        
        if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) {
            lat = '';
        }
        if(isNaN(node.longitude) || [null,undefined].includes(node.longitude)) {
            lon = '';
        }
        
        // 规范属性
        return {
            code: node.serial_number,
            olat: node.latitude,
            olon: node.longitude,
            lat: lat,
            lon: lon,
            name: node.host_name,
            groupName: node.group_name,
            ip: node.cell_ip,
            type: stationType,
            online: node.connection_status? 'on':'off',
            active: node.op_state? 'yes':'no',
            alarmCount: node.alarm_count,
            alarmLevel: node.alarm_serverity,
            mmeEnable: enable,
            mmeStatus: node.mme_status,
            mmePool1: mmePool_1,
            mmePool2: mmePool_2,
            serverList: serverList,
            cellCode: node.small_cell_code,
            alarm: node.alarm,
            sasEnable: node.sas_enable,
            sasState: node.sas_state,
            relaCpeList: node.relaCpeList,
            NEW_MME_STATUS: node.new_mme_status,
            pci: node.phycellid,
            height: height,
            direct: horizontal_azimuth,
            radius: radius,
            minRadius: minRadius,
            angles: 115,
            meDowntilt: meDowntilt,
            elDowntilt: elDowntilt,
            vertical3dB: vertical3dB,

            mechanical_downtilt: meDowntilt,
            electronic_downtilt: elDowntilt,
            vertical_3dB_beam_width: vertical3dB,
            horizontal_azimuth: horizontal_azimuth,

            ueShow: ueCount - 0 > 0,
            ueCount: ueCount,
            hasMDT: hasMDT,

            isGSM: isGSM,

            throughputDL: node.throughputDL,
            throughputUL: node.throughputUL
        };
    });
    // 处理无经纬度的节点
    //proccessNoLocNodes(nodeList);

    // 过滤无经纬度的节点
    nodeList = nodeList.filter(function(item){
        return item.lat && item.lon;
    });
    return nodeList;
}

function highlightSiteNode(code){
    $('.node-code').each(function(idx,item){
        let $dom = $(item);
        if($dom.text() == code) {
            $dom.addClass('site-selected');
            $dom.parent().addClass('site-selected');
        }else {
            $dom.removeClass('site-selected');
            $dom.parent().removeClass('site-selected');
        }
    });
}

/**
* 跳转到告警菜单
* @param alarm_severity{string}：告警级别Id
* @param sn{string}：基站SN
**/
function jumpToAliveAlarm(alarm_severity,sn) {
    let title = "";
    if (alarm_severity == '31001') {
        title = "Critical Alarms";
    } else if (alarm_severity == '31002') {
        title = "Major Alarms";
    } else if (alarm_severity == '31003') {
        title = "Minor Alarms";
    } else if (alarm_severity == '31004') {
        title = "Warning Alarms";
    }
    
    try{
        let params = {
            alarm_severity: '', 
            unread: '',
            search_text: sn,
        };
        eventAllBus.$emit("gomenupage","8000","","8000",params,function(){
            alarmViewVue.activeName = 'alarmView';
            alarmViewVue.resetQueryParams(params);
        });

        if(alarmViewVue) {
            alarmViewVue.activeName = 'alarmView';
            alarmViewVue.resetQueryParams(params);
        }
    }catch(e){}
}
function showUEcount(dom,event) {
    let sn = event.target.parentNode.getAttribute('code');

    topovm.toUETrace(sn);
    event.stopPropagation();
}
/**
* 是否连接正常
* @param record{object}：地图节点数据
**/
function isConnected(record){
    let bool = false;
    if(record.label == 'EPC'){// EPC连线
        if(record.mmeEnable) {
            bool = record.mmeStatus == 1;
        }else if(record.mmeStatus){
            let poolArr = record.mmeStatus.split(','); // ['mme1=1','mme2=0']格式
            poolArr.map(function(item){
                if(item){
                    let arr = item.split('=');
                    if(arr[0] == record.tar && arr[1] == 1) bool = true;
                }
            })
        }
    }else if(record.label == 'OMC'){// OMC连线
        if(record.online == 'on') bool = true;
    }
    
    return bool;
}
/**
* MME状态-内容处理
* @param value{string}：mme状态值
* @param rowData{object}：节点数据
* @param rowIndex{number}：对应下标
**/
function mmeStatusFormatterTopo(value, rowData, rowIndex){
    if (value == null || value == "") {
        return null;
    }
    
    if (value == "1") {
        value = "<div class='mmeConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='MME'><span class='el-icon el-icon-status-MME'></span></div>"
                + "<div class='mmeDetails MMEDetail'>MME IP : "+rowData.s1siglinkserverlist + "</br>MME Status : <span class='mmeStauts'></span></div>";
    } else if (value == "0") {
        value = "<div class='mmeDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='MME'><span class='el-icon el-icon-status-MME'></span></div>"
        + "<div class='mmeDetails MMEDetail'>MME IP : "+rowData.s1siglinkserverlist + "</br>MME Status : <span class='mmeStauts'></span></div>";
    } else if (value == "2") {
        value = "--";
    } else{
        let mmePools = value.split(",");
        let ret = "";
        for(let i = 0; i < mmePools.length; i++ ){
            let mme = mmePools[i];
            let mmeArr = mme.split("=");
            
            if(mmeArr.length == 2 && "mme1" == mmeArr[0] && mmeArr[1] == "1"){
                ret = ret + "<div class='mmeConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='MME1'><span class='el-icon el-icon-status-MME1'></span></div>"; 
            }else if(mmeArr.length == 2 && "mme1" == mmeArr[0] && mmeArr[1] == "0"){
                ret = ret + "<div class='mmeDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='MME1'><span  class='el-icon el-icon-status-MME1'></span></div>";
            }else if(mmeArr.length == 2 && "mme2" == mmeArr[0] && mmeArr[1] == "1"){
                ret = ret + "<div class='mmeConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' style='margin-left:10px;' type='MME2'><span  class='el-icon el-icon-status-MME2'></span></div>";
            }else if(mmeArr.length == 2 && "mme2" == mmeArr[0] && mmeArr[1] == "0"){
                ret = ret + "<div class='mmeDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' style='margin-left:10px;' type='MME2'><span class='el-icon el-icon-status-MME2'></span></div>"; 
            }
        }
        let lastChar = ret.charAt(ret.length - 1);
        if("," == lastChar){
            ret = ret.substring(0,ret.length - 1);
        }
        value = ret + "<div class='mmeDetails MME1Detail'>MME1 IP : "+rowData.mme_pool_1 + "</br>MME1 Status : <span class='mme1Stauts'></span></div>"+ "<div class='mmeDetails MME2Detail'>MME2 IP : "+rowData.mme_pool_2 + " <br/>MME2 Status : <span class='mme2Stauts'></span></div>";  
    }
    
    return value;
}
function sasStatusFormatter(value, row) {
    let state = row.sasState,
        str = '',
        codes = {
            Unregistered: 'offline',
            Registered: 'registed',
            Granted: 'grated',
            Authorized: 'authed'
        };

    if(value) {
        str = '<i class="el-icon el-icon-topo-enb margin-right-5 ' + codes[value] + '"></i>' + value;
    }

    return str;
}
function alarmPopoverFormatter(value, row) {
    let str = '',
        alarmObj =value;
    
    if(alarmObj) {
        let keys = ['31001','31002','31003','31004'],
            alarmCount = 0,
            levels = {
                31001: 'Critical',
                31002: 'Major',
                31003: 'Minor',
                31004: 'Warning',
            },
            types = [],
            type = '';

        keys.map(function(key){
            if(alarmObj[key]) {
                alarmCount += alarmObj[key]*1;
                types.push(levels[key]);
            }
        });

        type = types[0];

        if(alarmCount) {
            str = '<span class="el-icon el-icon-menu-alarm margin-right-5 '+ type +'"></span>'
                    +type+'<span style="color: blue;cursor: pointer;" onclick="jumpToAliveAlarmEnb(&quot;' + row.code + '&quot;)">('+(alarmCount*1)+')</span>';
        }
    }

    return str;
}
function alarmInfoFormatter(value, row) {
    let str = '',
        alarmObj =value;
    
    if(alarmObj) {
        let keys = ['31001','31002','31003','31004'],
            keycls = [],
            alarmCount = 0,
            levels = {
                31001: 'Critical',
                31002: 'Major',
                31003: 'Minor',
                31004: 'Warning',
            },
            types = [],
            type = '';

        keys.map(function(key){
            if(alarmObj[key]) {
                keycls.push(levels[key]);
                alarmCount += alarmObj[key]*1;
                types.push(levels[key]);
            }
        });

        type = types[0];

        if(alarmCount) {
            str = '<span class="alarm-circle margin-right-5 '+ type +'">' + alarmCount + '</span>'+ type;
        }
    }

    return str;
}
function jumpToAliveAlarmEnb(sn) {
    try{
        let params = {
            alarm_severity: '', 
            unread: '',
            search_text: sn,
        };
        eventAllBus.$emit("gomenupage","8000","","8000",params,function(){
            alarmViewVue.activeName = 'alarmView';
            alarmViewVue.resetQueryParams(params);
        });

        if(alarmViewVue) {
            alarmViewVue.activeName = 'alarmView';
            alarmViewVue.resetQueryParams(params);
        }
    }catch(e){}
}
// 关闭节点信息浮层
function cancelSet() {
    $('.leaflet-popup-close-button')[0].click();
}
/**
* 校验经纬度有效性
* @param gpsValue{object}： gps数据对象
**/
function validPGSValue(gpsValue){
    let bool = false;
    if(gpsValue){
        let arr = gpsValue.split(','),
            lon = arr[0]-0,
            lat = arr[1]-0;
        if(!isNaN(lat) && !isNaN(lon)){
            if(Math.abs(lat) <= 90 && Math.abs(lon) <= 180 && lessThenLength(lat,8) && lessThenLength(lon,9) ) {
                bool = true;
            }
        }
    }
    return bool;
}
/**
* 判断字符是否小于指定长度
* @param str{string}：要判断的字符
* @param num{number}：长度值
**/
function lessThenLength(str,num){
    str += '';
    return str.replace(/\./g,'').length <= num;
}
/* 鼠标移出隐藏mme信息 */
function hideMMEDetail(){
    $(".mmeDetails").fadeOut(300);
}

function showRedAccordingRSRP(value, rowData, rowIndex) {
    let maxValue = highVal,
        minValue = lowVal;
    
    let ret ;
    if ( value ){
        if (value < minValue) {
            ret = "<span class='el-icon el-icon-signal signal-low' style='display:flex;align-items:center;'>" + value + "</span>";
            
        } else if (value > maxValue){
            ret = "<span class='el-icon el-icon-signal signal-high' style='display:flex;align-items:center;'>" + value + "</span>";
        }else {
            ret = "<span class='el-icon el-icon-signal signal-normal' style='display:flex;align-items:center;'>" + value + "</span>";
        }
    }else {
        ret = value || ''
    }
    
    return ret;
}

/**
* enb数据规范化 -- {code: '',lat: '',lon: '',name: '',type: '',online: '',active: '', ... }为必要属性
* @param nodeList{array}：enb节点原始数据
**/
function transformGnbNode(nodeList){
    let latLons = nodeList.map(function(node){ return node.latitude+'-'+node.longitude; });
    
    nodeList = nodeList.map(function(node){
        let enable = node.mme_enable == 1? true:false,
            mmePool_1 = node.mme_pool_1?node.mme_pool_1:'',
            mmePool_2 = node.mme_pool_2?node.mme_pool_2:'',
            serverList = node.s1siglinkserverlist?node.s1siglinkserverlist:'',
            lat = node.latitude,
            lon = node.longitude,
            ueCount = node.ue_count;
        
        if(node.latitude == "null") {
            lat = '';
        }
        if(node.longitude == "null") {
            lon = '';
        }
        
        // 规范属性
        return {
            code: node.serial_number,
            olat: node.latitude,
            olon: node.longitude,
            lat: lat,
            lon: lon,
            name: node.host_name,
            groupName: node.group_name,
            ip: node.cell_ip,
            type: 'gnb',
            online: node.connection_status? 'on':'off',
            active: node.op_state? 'yes':'no',
            alarmCount: node.alarm_count,
            alarmLevel: node.alarm_serverity,
            mmeEnable: enable,
            mmeStatus: node.mme_status,
            mmePool1: mmePool_1,
            mmePool2: mmePool_2,
            serverList: serverList,
            cellCode: node.small_cell_code,
            alarm: node.alarm,
            sasEnable: node.sas_enable,
            sasState: node.sas_state,
            relaCpeList: node.relaCpeList,
            NEW_MME_STATUS: node.new_mme_status,
            height: node.height,
            ueShow: ueCount - 0 > 0,
            ueCount: ueCount
        };
    });
    // 处理无经纬度的节点
    //proccessNoLocNodes(nodeList);

    // 过滤无经纬度的节点
    nodeList = nodeList.filter(function(item){
        return item.lat && item.lon;
    });
    return nodeList;
}
/**
* 刷新分组数据并渲染
* @param param{array}: 分组数据的查询参数
**/
function refreshRenderData2Dom(param) {
    const slider = $('#enbSetting_slide'),
        postData = slider.data('params');
    
    if(slider.length == 0) return;

    // 重新请求分组数据渲染
    if(param.groupId == postData.id && param.smallCellCode == postData.smallCellCode) {
        enbSettingNewPanelVue.changeMenu(param.smallCellCode,param.groupId,false)
    }
}
//画图： 图表由 节点、文字、线组成
//1、stroke: 边框颜色；  2、 margin: 边框间距,   margin: new go.Margin(10,20,30,40) 外边距;   3、fill: 背景颜色；
//1、TextBlock: 创建文本；   2、Shape: 创建图形；  3、 Node:节点（结合文本与图形）；  4、Links 连线
function initMapSiteTopo(params){
     // 清理旧的 Diagram 实例
    if(myMapSiteTopoDiagram) {
        myMapSiteTopoDiagram.clear();
        myMapSiteTopoDiagram.div = null;
        myMapSiteTopoDiagram = null;
    }
    // 创建图表
    let $ = go.GraphObject.make;
    //绑定DOM元素
    myMapSiteTopoDiagram = 
        $(go.Diagram, params.domId,
            {
                isReadOnly: true,					
                // 画布的位置设置：居中显示内容, 不可拖动画布
                contentAlignment: go.Spot.Center,
                // 启用Ctrl-Z和Ctrl-Y撤销重做功能
                'undoManager.isEnabled': false,
                //去掉节点 点击时的边框颜色
                nodeSelectionAdornmentTemplate:
                    $(go.Adornment,'Aiuto',
                        $(go.Shape, 'Rectangle',{fill:'white',stroke: null})		
                    ),
                //树形布局排列方式，从上到下（0，90，180，270） 、每层间距,
                layout: $(go.TreeLayout,
                    {angle: 90, layerSpacing: 66}) 					
            }
        );
    
    //新建节点
    myMapSiteTopoDiagram.nodeTemplate = 
        $(go.Node, 'Auto',
            //突出显示 鼠标滑过、离开
            {
                selectionAdorned: false,
                selectionChanged: allChanged,
                //鼠标滑过显示 设备名称
                mouseEnter: mouseEnter,
                mouseLeave: mouseLeave
            },
            //节点禁止拖动
            {movable: false},
            
            //设置节点形状：圆角矩形
            $(go.Shape, 'RoundedRectangle',
                //设置大小、边框大小、颜色、背景色、鼠标手势
                {width: 120, height: 60, strokeWidth: 2, margin: new go.Margin(0,0,0,20), cursor: 'grab', name: 'SHAPE'},
                //将节点数据nodeDataArray   .color与节点背景色建立联系
                //绑定背景色
                new go.Binding('fill', 'bgColor'),
                //绑定边框色
                new go.Binding('stroke', 'borderColor'),
            ),
            //设置文本节点
            $(go.TextBlock, textStyle(),
                //设置文本样式：大小，是否换行，margin
                { wrap: go.TextBlock.WrapFit, name: 'TEXT'},
                //将TextBlock.text 绑定到 Node.data.text
                new go.Binding('text', 'text'),
                new go.Binding('cursor', 'cursor')),
            //添加 tooltip，显示节点对应的基站编码
            {toolTip:
                $("ToolTip",
                    $(go.TextBlock,{margin: 6},
                        new go.Binding('text','serialnumber', 
                            function(sn){ 
                                return (params.KPISheBei || 'Device') + ': ' + sn;
                            }							
                        ))		
                )					
            } 
        );
    function allChanged(node){
        if(node.isSelected){
            if(node.part.data.outOfContact == 'true'){
                selectionAdornment.adornedObject = node;
                node.addAdornment('Radial',selectionAdornment);
            }else{
                let oldnode = selectionAdornment.adornedPart;
                if(oldnode) oldnode.removeAdornment('Radial');
                selectionAdornment.adornedObject = null;
            } 
        }else{
                let oldnode = selectionAdornment.adornedPart;
                if(oldnode) oldnode.removeAdornment('Radial');
                selectionAdornment.adornedObject = null;
        }
    };
    
    var selectionAdornment = 
        $(go.Adornment, 'Spot',
            $(go.Panel, 'Auto',
                $(go.Shape, {fill: null, stroke:'#DCDFE6', strokeWidth: 0}),
                $(go.Placeholder)
            ), 
            $('Button',
                { alignment: go.Spot.Top, alignmentFocus: go.Spot.Left, 
                    width: 140, cursor: 'pointer', padding: go.Margin.parse('0 0 20 10'), 
                    'ButtonBorder.fill': '#FFFFFF',
                    'ButtonBorder.stroke': '#E9E9E9',
                    '_buttonFillOver': '#EDF6FF',
                    '_buttonStrokeOver': '#E9E9E9',
                    '_buttonFillFocus': '#EDF6FF',
                    '_buttonStrokeFocus': '#E9E9E9',
                    click: function(e, obj){
                        if(obj.part.data.outOfContact == 'true'){
                            let curNodeData = obj.part.data;
                            let delParams = {
                                serialnumber: curNodeData.serialnumber,
                                text: curNodeData.text
                            }
                            delMapSiteTopoNodeClick(delParams,params);
                        }else{
                            return;
                        }
                    }
                },
                
                $(go.TextBlock, params.ShanChu || 'Delete',
                        {margin: 6,stroke: '#333333'},		
                )							
            )	
        )
            
    //虚线 实线	 连接	
    var templmap = new go.Map(), color  = '#FA5555';
    var defaultTemplate = 
        $(go.Link,
            $(go.Shape, { stroke: color, strokeWidth: 2})
        )
    var dashedTemplate = 
        $(go.Link,
            $(go.Shape, { stroke: color, strokeWidth: 2, strokeDashArray: [6, 3]})			
        )
    
    templmap.add('',defaultTemplate);
    templmap.add('dashed',dashedTemplate);
    myMapSiteTopoDiagram.linkTemplateMap = templmap; 
    
    //设置线条，暂无箭头
    myMapSiteTopoDiagram.linkTemplate = 
        $(go.Link,
            //线条连接样式：直线
            {curve: go.Link.Bezier},
            //线的连接形状
            $(go.Shape,
                {strokeWidth: 2, stroke: "#707070"}	
            )
        );
    
        // 定义图形上的文字风格		 
    function textStyle() {
        return {         
            //文本颜色
            stroke: "#333333",
            font: "bold 12px normal"
        }
    };
    //鼠标滑过修改背景颜色
    function mouseEnter(e,obj){			
        let shape = obj.findObject('SHAPE');
            if(obj.data.hoverColor == ''){
                shape.fill = '#F3F3F3';
            }else{
                shape.fill = obj.data.hoverColor;
            }
    };
    //鼠标离开还原色值
    function mouseLeave(e,obj){
        let shape = obj.findObject('SHAPE');
        shape.fill = obj.data.bgColor;
        
    };
    
    //获取TOPO 图数据
    setTimeout(function(){
        var param = { smallCellCode: params.smallCellCode };
        axios.post(webRootPath + '/gnb/gnbMonitor/getTOPOInfo.action',stringify(param)).then(function(response){
            let data = response.data;
            if(data){
                //添加节点数据
                let nodeDataArray = data.nodeDataArray;
                
                //添加连接线数据
                let linkDataArray = data.linkDataArray;

                //新建关系图: 通过节点数据和关系数组完成关系图
                myMapSiteTopoDiagram.model = new go.GraphLinksModel(nodeDataArray,linkDataArray);	
            }
        }).catch(function(error){}) 
    },500);	   
    
}
function delMapSiteTopoNodeClick(delparams,params){
    $.messager.confirm(params.QueRen, params.QueDingShanChuSheBei, function (r) {
        if (r) {
            let param = {
                smallCellCode: params.smallCellCode,
                serialNumber: delparams.serialnumber,
                text: delparams.text
            };
            $.post("${ctx}/gnb/gnbMonitor/delNodeForTOPOInfo.action", param, function (data) {
                if (data["success"]) {
                    showMsg('success_msg',params.ChengGong);
                    initMapSiteTopo(params);								
                }else{
                    showMsg('error_msg',data["message"]);
                }
            }, "json");
        }
    }).addClass("seriousConfirm");
}