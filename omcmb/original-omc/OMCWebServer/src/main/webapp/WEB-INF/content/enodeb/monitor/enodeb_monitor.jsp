<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>eNB Monitor</title>

    <style>
        .charts-list {
            min-width: 200px;
            height: 100%;
            background: #fff;
            overflow-y: auto;
            overflow-x: hidden;
        }
        .monitor-ctner {
            display: flex;
            height: 100%;
            background-color: #fff;
        }
        .flex-item-cls {
            flex: auto;
            overflow: auto;   
        }
        /* tabs css cover */
        .top-active-nav {
            height: 100%;
        }
        .top-active-nav .el-tabs__content {
            border: none;
        }
        .top-active-nav .el-tabs__header {
            background-color: #fff;
        }
        .top-active-nav .el-tabs__active-bar {
            top: 0;
        }

        .monitor-ctner .el-table th>.cell {
            white-space: nowrap;
        }

        .no-padding .slide-content {
            padding: 0px;
        }

        .no-border .el-card__body {
            border: none;
        }


        .cellNameClass{
            display: inline-block;
            height: 18px;
            width: 18px;
            text-align: center;
            line-height: 18px;
            border: 1px solid #DCDFE6;
            border-radius: 2px;
        }
        .cellNameClass .el-icon::before{
            font-size: 16px;
            color: #F2B354;
        }
        .cellNameClass:hover{
            border: 1px solid #4D84FF;
        }
        .mmeDetails {
            top: 30px;
            position:absolute;
            background:white;
            padding:10px 20px;
            box-shadow:4px 4px 19px 0 rgba(0,0,0,0.15);
            border:1px solid #d1ecf5;
            display:none;
            left:0;
            z-index:99999;
        }
        .syncNameInfo div{
            margin-right:20px;
            line-height:30px;
            
        }
        .activeStatusItem .el-icon,.inactiveStatusItem .el-icon{
            font-size:20px;
            vertical-align:bottom;
            margin-right:5px;
        }
        .activeStatusItem .el-icon-status-active:before{
            color:#67D972;
        }
        .inactiveStatusItem .el-icon-status-active:before{
            color:#E88282;
        }
        .mmeConnItem,.mmeDisconnItem,.rfConnItem,.rfDisconnItem {
            display: flex;
            white-space: nowrap;
            align-items: center;
        }
        .mmeConnItem .el-icon-status-MME:before,.mmeConnItem .el-icon-status-MME1:before,.mmeConnItem .el-icon-status-MME2:before{
            color:#67D972;
            font-size:18px;
        }
        .mmeDisconnItem .el-icon-status-MME:before,.mmeDisconnItem .el-icon-status-MME1:before,.mmeDisconnItem .el-icon-status-MME2:before{
            color:#E88282;
            font-size:18px;
        }
        .rfConnItem .el-icon-status-RF:before,.rfConnItem .el-icon-status-RF1:before,.rfConnItem .el-icon-status-RF2:before{
            color:#67D972;
            font-size:20px;
        }
        .rfDisconnItem .el-icon-status-RF:before,.rfDisconnItem .el-icon-status-RF1:before,.rfDisconnItem .el-icon-status-RF2:before{
            color:#E88282;
            font-size:20px;
        }
        
        .cpeLwaKai{
            display: inline-block;
            margin-top:2px;
            width:25px;
            height:25px;
            background:url(${ctx}/css/images/newIcon/statusIcon/TURBOkai.png) no-repeat;
        }
        .cpeLwaGuan{
            display: inline-block;
            margin-top:2px;
            width:25px;
            height:25px;
            background:url(${ctx}/css/images/newIcon/statusIcon/TURBOguan.png) no-repeat;
        }
        .cpeLwaWu{
            display: inline-block;
            margin-top:2px;
            width:25px;
            height:25px;
            background:url(${ctx}/css/images/newIcon/statusIcon/TURBOwu.png) no-repeat;
        }

        .expanded .table-list, .expanded .split-line, .minisize .split-line {
            display: none;
        }
        .expanded .charts-list, .expanded .fix-right-slide, .minisize .table-list {
            width: 100%;
        }
        .expanded .fix-right-slide {
            height: 100%;
            display: flex;
            flex-wrap: wrap;
        }

        .minisize .charts-list {
            width: 0;
            min-width: 0;
        }

        .slide-arrow {
            padding: 3px 0 3px 8px;
            min-width: 18px !important;
            max-width: 18px !important;
            max-height: 18px !important;
            position: fixed;
            top: calc(50% + 10px);
            right: 207px;
            z-index: 100;
            background-color: #4D84FF !important;
            border-radius: 15px 0 0 15px;
            box-shadow: 2px 0 15px rgba(0,0,0,.2);
        }
        .slide-arrow.right {
            right: 181px;
            transform: rotate(180deg);
        }
        .expanded .slide-arrow, .minisize .slide-arrow {
            right: 12px;
            border-radius: 15px 0 0 15px !important;
        }
        .minisize .slide-arrow {
            transform: rotate(0deg);
        }

        .slide-arrow:hover::before {
            position: absolute;
            display: inline-block;
            min-width: 80px;
            content: '<%=rb.getString("ZhanKai")%>';
            bottom: -25px;
            right: -30px;
            font-size: 14px;
            color: #4d84ff;
            background: #fff;
        }
        .slide-arrow.right:hover::before {
            content: '<%=rb.getString("GuanBi")%>';
            right: 0px;
            transform: rotate(180deg);
        }
        .minisize .slide-arrow.right:hover::before {
            content: '<%=rb.getString("ZhanKai")%>';
            right: 0px;
            min-width: 40px;
            transform: rotate(0deg);
        }

        .expanded .slide-arrow:hover::before {
            content: '<%=rb.getString("GuanBi")%>';
        }
        
        .slide-arrow i {
            font-size: 12px;
            transform: rotate(90deg);
        }
        .expanded .slide-arrow i {
            margin-top: 0px;
            transform: rotate(-90deg);
        }
        
        .slide-arrow i::before {
            color: #fff !important;
        }

        
        .showHideItem {
            position:absolute;
            background:white;
            z-index:888;
            padding-top: 10px;
            padding-left: 10px;
            width: 700px;
            top:100px;
            left: 0px;
            display: none;
            box-shadow:5px 10px 23px 0px rgba(0,0,0,0.15);
        }
        .enbMonitorForm{
            margin-left:10px;
        }
        .enbMonitorForm .selectItem .el-input{
            width:127px;
        }
        .enbMonitorForm .selectItem .el-input__inner{
            height:30px;
            border-radius:2px 0px 0px 2px;
        }
        .enbMonitorForm .queryGroup{
            height:28px;
            border-radius:0px 4px 4px 0px;
            margin-left:0px;
        }
        .enbMonitorForm .queryGroup .el-input__inner{
            height:28px;
        }
        .enbMonitorForm .el-form-item{
            display:inline-block;
            margin-bottom:0px;
        }
        .enbMonitorForm .selectContent{
            flex:1
        }
        .enbMonitorForm .selectContent .el-form-item{
            margin-bottom:6px;
        }
        .enbMonitorForm .selectContent .el-input{
            width:100px;
            border-radius:2px;
        }
        .enbMonitorForm .selectContent .el-input__inner{
            height:24px;
        }
        .enbMonitorForm .el-form-item__label{
            line-height:24px;
            text-align:right;
            font-size:12px;
        }
        .enbMonitorForm .el-tag__close{
            display:none;
        }
        .enbMonitorForm .el-select__tags{
            height:24px;
            overflow:hidden;
        }
        .enbMonitorForm .el-tag--small{
            height:16px;
            line-height:16px;
            background:#fff;
        }
        .enbMonitorForm .el-tag--small:nth-of-type(2){
            display:none;
        }
        .el-select-dropdown.is-multiple .el-select-dropdown__item.selected::after{
            right:2px;
        }

        .col-group {
            margin: 5px 10px;
            display: flex;
            flex-wrap: wrap;
        }
        .col-group .el-checkbox {
            min-width: 140px;
        }
        .col-group .el-checkbox__label {
            font-size: 12px;
        }

        .showHideItem input{
            margin-top:-2px;
            margin-bottom:1px;
            vertical-align:middle;
            margin-right:20px;
        }
        .selectAll{
            height:32px;
            width:334px;
            padding:28px 0px 0px 30px;
        }
        
        .showHideItem .select-all-cls {
            padding: 10px 0 0 9px;
            display: flex;
            align-items: center;
        }
        .select-all-cls > i {
            margin-right: 5px;
        }
        .showHideItem .select-all-cls > span {
            font-size: 14px;
            font-weight: bold;
            margin-left: 10px;
        }
        .showHideItem .el-icon-close1::before {
            color: #333;
        }
        
        .left-title-list li{
            height:36px;
            line-height:36px;
            padding:0 20px;
            cursor: pointer;
            border-bottom: 1px solid #EEE;
            word-break: keep-all;
        }
        .left-title-list li.active {
            color: #4D84FF;
            background-color: #EDF6FF;
        }

        /* 列表告警显示样式 */
        .alarmListSty{
            display:inline-block;
            min-width:13px;
            height:23px;
            padding:0 5px;
            line-height:24px;
            border-radius:23px;
            text-align:center;
            color:#FFFFFF;
            -webkit-transform:scale(0.8);
            font-size:12px;
            cursor:pointer;
        }
        .alarmCritical{
            background:#E88282;
        }
        .alarmMajor{
            background:#DCAA5E;
        }
        .alarmMinor{
            background:#CCCC66;
        }
        .alarmWarning{
            background:#9AF0FE;
        }

        .mme-list {
            position: relative;
            display: flex;
            max-width: 825px;
            flex-wrap: wrap;
        }
        .mme-list-item {
            display: flex;
            padding: 10px;
            border: 1px solid #ebeef5;
        }
        .mme-info {
            display: flex;
            flex-direction: column;
        }
    </style>
</head>
<body>
    <div id="cellInfo" class="monitor-ctner">
        <!-- 右上角导出按钮 -->
        <div v-show="activeName=='table'" class="circleIcon placeholder-bt" style="top:10px;right: 220px;" placeholder="<%=rb.getString("DaoChu")%>" id="enb_export_op" @click="showExport">
            <el-popover trigger="click" placement="bottom-end">
                <div class="el-card__header">
                    <%=rb.getString("DaoChu")%>
                    <span style="color: 999;font-weight: normal;margin-left: 5px;">(<%=rb.getString("SuoYouCanShu")%>)</span>
                    <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                </div>
                <div class="export-content" style="width: 920px; max-height: 500px;overflow: auto;"></div>
                <span slot="reference" class="el-icon-circle-export el-icon"></span>
            </el-popover>
        </div>

        <!-- tabs -->
        <div class="flex-item-cls table-list">
            <!-- 新建 设备，导入功能-->
            <div v-show="activeName=='table'" id="addDeviceWarp"></div>

            <el-tabs class="top-active-nav" v-model="activeName" @tab-click="tabClick">
                <!-- list -->
                <el-tab-pane label="Table" name="table">
                    <el-ctable ref="list" height="100%"
                        id="tableHomeCellList"
                        row-key="serial_number"
                        :url="tbURL"
                        :query-params="queryParams"
                        @load-success="loadSuccess"
                        @selection-change="selectionChange">
                        <template slot="toolbar">
                            <div id="toolbar_tableHomeCellList" style="min-height: 78px;"></div>
                        </template>

                        <el-table-column v-if="enableCheckbox" type="selection" :reserve-selection="true"></el-table-column>
                        <el-table-column prop="enb_operation" width="40">
                            <template slot-scope="scope">
                                <div class="el-icon el-icon-operation-more" @click="optClick(scope.row, event)" v-clickoutside="handerClose"></div>
                            </template>
                        </el-table-column>
                        <el-table-column sortable prop="connection_status" width="40">
                            <template slot-scope="scope">
                                <div v-html="connStatusFmt(scope.row, scope.row['connection_status'], scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column sortable label="<%=rb.getString("GaoJingShu")%>" prop="alarm" width="65" :formatter="alarmFmt">
                            <template slot-scope="scope">
                                <div v-html="alarmFmt(scope.row, '', scope.row.alarm, scope.$index)" style="text-align: center;"></div>
                            </template>
                        </el-table-column>

                        <!--<el-table-column v-for="item in showColums" :label="item.label" :prop="item.prop" :width="item.width">
                            <template slot-scope="scope">
                                <div v-if="item.formatter" v-html="item.formatter(scope.row, item, scope.row[item.prop], scope.$index)"></div>
                                <div v-else>{{scope.row[item.prop]}}</div>
                            </template>
                        </el-table-column>-->

                        <el-table-column v-if="showColums.includes('serial_number')" sortable label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number" width="180"></el-table-column>
                        <el-table-column v-if="showColums.includes('host_name')" sortable label="<%=rb.getString("HostName")%>" prop="host_name" width="150">
                            <template slot-scope="scope">
                                <div v-if="scope.row.device_name_tip === '0'">{{scope.row.host_name}}</div>
                                <div v-if="scope.row.host_name && scope.row.device_name_tip !== '0'" class="cellNameClass">
                                    <el-popover trigger="click">
                                        <div slot="reference">
                                            <span class="el-icon el-icon-circle-warning"></span> 
                                            {{scope.row.host_name}}
                                        </div>
                                        <div>
                                            <div> <span class="panel_close" @click="closeSyncName" style="margin-right: 0px;"></span>
                                            <div><%=rb.getString("JiZhanCeMingCheng")%> : {{scope.row.report_host_name}}</div>
                                            <div><%=rb.getString("ShiFouTongBuMingChengDaoOMC")%><div>
                                            <div style="margin-top: 5px;">
                                                <span class="button_simple" @click="syncName(scope.row.small_cell_code, scope.row.report_host_name)">
                                                    <%=rb.getString("QueDing")%>
                                                </span>
                                                <span class="button_simple white" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
                                            </div>
                                        </div>
                                    </el-popover>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('rf_status')" sortable label="<%=rb.getString("ShePinKaiGuanZhuangTai")%>" prop="rf_status" width="100" :show-overflow-tooltip="false">
                            <template slot-scope="scope">
                                <div v-html="rfStatusFmt(scope.row, scope.row.rf_status, scope.$index)" style="display: flex;white-space: nowrap;"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('op_state')" sortable label="<%=rb.getString("ShiFouJiHuo")%>" prop="op_state" width="120">
                            <template slot-scope="scope">
                                <div v-if="scope.row.op_state == '1'" class='activeStatusItem'>
                                    <span class='el-icon el-icon-status-active' style='margin-right: 5px;'></span><%= rb.getString("JiHuo")%>
                                </div>
                                <div v-if="scope.row.op_state == '0'" class='inactiveStatusItem'>
                                    <span class='el-icon el-icon-status-active' style='margin-right: 5px;'></span><%= rb.getString("QuJiHuo")%>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('CELL_IDENTITY')" sortable label="ECI" prop="CELL_IDENTITY" width="70"></el-table-column>
                        <el-table-column v-if="showColums.includes('PHYCELLID')" sortable label="<%=rb.getString("PCI2")%>" prop="PHYCELLID" width="50"></el-table-column>
                        <el-table-column v-if="showColums.includes('mme_status')" label="<%=rb.getString("MMEZhuangTai")%>" prop="mme_status" width="80">
                            <template slot-scope="scope">
                                <div v-if="false" v-html="mmeStatusFmt(scope.row, scope.row.mme_status, scope.$index)" style="display: flex;white-space: nowrap;"></div>
                                <div style="display: flex;align-items: center;">
                                    <span class="el-icon el-icon-status-MME1"></span>
                                    <el-popover title="All MME">
                                        <span style="color:#4d84ff;" slot="reference">
                                            [3<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
                                        </span>
                                        <div class="mme-list">
                                            <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                            <div class="mme-list-item" v-for="item in [1,2,3,4,5]">
                                                <span class="el-icon el-icon-status-MME1"></span> 
                                                <div class="mme-info">
                                                    <span>MME IP: 255.255.255.254</span>
                                                    <span><%=rb.getString("MMEZhuangTai")%>: Disconnected</span>
                                                    <span><%=rb.getString("PLMN")%>: 87654</span>
                                                </div>
                                            </div>
                                        </div>
                                    </el-popover>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('plmnid')||true" label="<%=rb.getString("PLMN")%>" prop="plmnid" width="70">
                            <template slot-scope="scope">
                                <div v-html="plmnFmt(scope.row, scope.row.plmnid, scope.$index)"></div>
                            </template>
                        </el-table-column>

                        <el-table-column v-if="showColums.includes('ue_count')" label="<%=rb.getString("UEShu")%>" prop="ue_count" width="80">
                            <template slot-scope="scope">
                                <div v-html="ueCountFmt(scope.row, scope.row.ue_count, scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('cpe_connect')" label="<%=rb.getString("CPELianJieShu")%>" prop="cpe_connect" width="80">
                            <template slot-scope="scope">
                                <div v-html="cpeCountFmt(scope.row, scope.row.cpe_connect, scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('cell_ip')" sortable label="IP" prop="cell_ip" width="100">
                            <template slot-scope="scope">
                                <div v-html="ipAddrFmt(scope.row, scope.row.cell_ip, scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('mac_address')" sortable label="MAC" prop="mac_address" width="115"></el-table-column>
                        <el-table-column v-if="showColums.includes('product')" sortable label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" prop="product" width="110">
                            <template slot-scope="scope">
                                <div v-html="productFmt(scope.row, scope.row.product, scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('module_type')" sortable label="<%=rb.getString("SheBeiXingHaoMing")%>" prop="module_type" width="95">
                            <template slot-scope="scope">
                                <div v-html="capablityFmt(scope.row, scope.row.module_type, scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('software_version')" sortable label="<%=rb.getString("SoftwareVersion")%>" prop="software_version" width="120"></el-table-column>
                        <el-table-column v-if="showColums.includes('group_name')" sortable label="<%=rb.getString("SheBeiZu")%>" prop="group_name" width="130"></el-table-column>
                        <el-table-column v-if="showColums.includes('IPSEC_ADDR')" label="<%=rb.getString("IPSECDiZhi")%>" prop="IPSEC_ADDR" width="100"></el-table-column>
                        <el-table-column v-if="showColums.includes('mmepool_ipsec_addr')" label="<%=rb.getString("MMEPoolIPSECDiZhi")%>" prop="mmepool_ipsec_addr" width="100"></el-table-column>
                        <el-table-column v-if="showColums.includes('site_id')" sortable :label="siteIdLabel" prop="site_id" width="70"></el-table-column>
                        <el-table-column v-if="showColums.includes('EARFCNDLINUSE')" label="<%=rb.getString("PinDian")%>" prop="EARFCNDLINUSE" width="120">
                            <template slot-scope="scope">
                                <div v-html="earfcnFmt(scope.row, scope.row.EARFCNDLINUSE, scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('synStatus')" sortable label="<%=rb.getString("TongBuZhuangTai")%>" prop="synStatus" width="120">
                            <template slot-scope="scope">
                                <div v-html="syncStatusFmt(scope.row, scope.row.synStatus, scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('pm_report_status')" sortable label="<%=rb.getString("KPIShangBaoZhuangTai")%>" prop="pm_report_status" width="130">
                            <template slot-scope="scope">
                                <div v-html="kpiStatusFmt(scope.row, scope.row.pm_report_status, scope.$index)" style="display: flex; align-items: center;"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('gps_satellite_count')" sortable label="<%=rb.getString("GPSWeiXingShu")%>" prop="gps_satellite_count" width="75"></el-table-column>
                        <el-table-column v-if="showColums.includes('up_time')" sortable label="<%=rb.getString("YunXingShiJian")%>" prop="up_time" width="110"></el-table-column>
                        <el-table-column v-if="showColums.includes('first_online_time')" sortable label="<%=rb.getString("DiYiCiLianJieShiJian")%>" prop="first_online_time" width="125"></el-table-column>
                        <el-table-column v-if="showColums.includes('LASTINFORMTIME')" sortable label="<%=rb.getString("ShangCiLianJieShiJian")%>" prop="LASTINFORMTIME" width="125"></el-table-column>
                        <el-table-column v-if="showColums.includes('network_model')" sortable label="<%=rb.getString("JiZhanZhiShi")%>" prop="network_model" width="90"></el-table-column>
                        <el-table-column v-if="showColums.includes('firmware_version')" sortable label="<%=rb.getString("FirmwareVersion")%>" prop="firmware_version" width="115"></el-table-column>
                        <el-table-column v-if="showColums.includes('gps_version')" label="<%=rb.getString("GPSBanBen")%>" prop="gps_version" width="90"></el-table-column>
                        <el-table-column v-if="showColums.includes('available_rate')" label="<%=rb.getString("XiaoQuKeYongZhanBi")%>" prop="available_rate" width="70">
                            <template slot-scope="scope">
                                <div v-if="scope.row.available_rate=='--'">--</div>
                                <div v-else>
                                    <a style='color:#1DA3FC;text-decoration:underline' href='#' @click='getUnuseTime(scope.row)'>{{scope.row.available_rate}}</a>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('halob_flag')" sortable label="<%=rb.getString("HaloBKaiGuan")%>" prop="halob_flag" width="100">
                            <template slot-scope="scope">
                                <div v-html="halobStatusFmt(scope.row, scope.row.halob_flag, scope.$index)" style="display: flex; align-items: center;"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('tac')" label="<%=rb.getString("TAC")%>" prop="tac" width="50"></el-table-column>
                        <el-table-column v-if="showColums.includes('gps_longitude')" label="<%=rb.getString("GPSJingDu")%>" prop="gps_longitude" width="80">
                            <template slot-scope="scope">
                                <div v-if="scope.row.gps_modify_flag != '1'">{{scope.row.gps_longitude}}</div>

                                <div v-else>
                                    <div v-if="scope.row.gps_longitude === undefined" :key="scope.$index">
                                        {{scope.row.modify_longitude == undefined? scope.row.gps_longitude : scope.row.modify_longitude}}
                                    </div>
                                    
                                    <div v-if="scope.row.gps_longitude !== undefined" class="cellNameClass" :key="scope.$index">
                                        <el-popover trigger="click">
                                            <div slot="reference">
                                                <span class="el-icon el-icon-circle-warning"></span> 
                                                {{scope.row.gps_longitude}}
                                            </div>
                                            <div>
                                                <div> <span class="panel_close" @click="closeSyncName" style="margin-right: 0px;"></span>
                                                <div>
                                                    <%=rb.getString("JingDu")%>: {{scope.row.gps_longitude}}&nbsp;&nbsp;
                                                    <%=rb.getString("WeiDu")%>: {{scope.row.gps_latitude}}&nbsp;&nbsp;
                                                    <%=rb.getString("GaoDu")%>: {{scope.row.gps_height}}
                                                </div>
                                                <div><%=rb.getString("TongBuGPSTiShi")%><div>
                                                <div style="margin-top: 5px;">
                                                    <span class="button_simple" @click="synchronizeGPS(scope.row.small_cell_code)">
                                                        <%=rb.getString("QueDing")%>
                                                    </span>
                                                    <span class="button_simple white" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
                                                </div>
                                            </div>
                                        </el-popover>
                                    </div>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('gps_latitude')" label="<%=rb.getString("GPSWeiDu")%>" prop="gps_latitude" width="70">
                            <template slot-scope="scope">
                                <div v-if="scope.row.gps_modify_flag != '1'">{{scope.row.gps_latitude}}</div>

                                <div v-else>
                                    <div v-if="scope.row.gps_latitude === undefined" :key="scope.$index">
                                        {{scope.row.modify_latitude == undefined? scope.row.gps_latitude : scope.row.modify_latitude}}
                                    </div>
                                    
                                    <div v-if="scope.row.gps_latitude !== undefined" class="cellNameClass" :key="scope.$index">
                                        <el-popover trigger="click">
                                            <div slot="reference">
                                                <span class="el-icon el-icon-circle-warning"></span> 
                                                {{scope.row.gps_latitude}}
                                            </div>
                                            <div>
                                                <div> <span class="panel_close" @click="closeSyncName" style="margin-right: 0px;"></span>
                                                <div>
                                                    <%=rb.getString("JingDu")%>: {{scope.row.gps_longitude}}&nbsp;&nbsp;
                                                    <%=rb.getString("WeiDu")%>: {{scope.row.gps_latitude}}&nbsp;&nbsp;
                                                    <%=rb.getString("GaoDu")%>: {{scope.row.gps_height}}
                                                </div>
                                                <div><%=rb.getString("TongBuGPSTiShi")%><div>
                                                <div style="margin-top: 5px;">
                                                    <span class="button_simple" @click="synchronizeGPS(scope.row.small_cell_code)">
                                                        <%=rb.getString("QueDing")%>
                                                    </span>
                                                    <span class="button_simple white" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
                                                </div>
                                            </div>
                                        </el-popover>
                                    </div>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('gps_height')" label="<%=rb.getString("GPSGaoDu")%>" prop="gps_height" width="70">
                            <template slot-scope="scope">
                                <div v-if="scope.row.gps_modify_flag != '1'">{{scope.row.gps_height}}</div>

                                <div v-else>
                                    <div v-if="scope.row.gps_height === undefined" :key="scope.$index">
                                        {{scope.row.modify_height == undefined? scope.row.gps_height : scope.row.modify_height}}
                                    </div>
                                    
                                    <div v-if="scope.row.gps_height !== undefined" class="cellNameClass" :key="scope.$index">
                                        <el-popover trigger="click">
                                            <div slot="reference">
                                                <span class="el-icon el-icon-circle-warning"></span> 
                                                {{scope.row.gps_height}}
                                            </div>
                                            <div>
                                                <div> <span class="panel_close" @click="closeSyncName" style="margin-right: 0px;"></span>
                                                <div>
                                                    <%=rb.getString("JingDu")%>: {{scope.row.gps_longitude}}&nbsp;&nbsp;
                                                    <%=rb.getString("WeiDu")%>: {{scope.row.gps_latitude}}&nbsp;&nbsp;
                                                    <%=rb.getString("GaoDu")%>: {{scope.row.gps_height}}
                                                </div>
                                                <div><%=rb.getString("TongBuGPSTiShi")%><div>
                                                <div style="margin-top: 5px;">
                                                    <span class="button_simple" @click="synchronizeGPS(scope.row.small_cell_code)">
                                                        <%=rb.getString("QueDing")%>
                                                    </span>
                                                    <span class="button_simple white" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
                                                </div>
                                            </div>
                                        </el-popover>
                                    </div>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('validity')" label="<%=rb.getString("YouXiaoQi")%>" prop="validity" width="125">
                            <template slot-scope="scope">
                                <div v-html="validityFmt(scope.row, scope.row.validity, scope.$index)"></div>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="showColums.includes('lock_status')" label="<%=rb.getString("SuoDingZhuangTai")%>" prop="lock_status" width="90">
                            <template slot-scope="scope">
                                <div v-html="lockStatusFmt(scope.row, scope.row.lock_status, scope.$index)"></div>
                            </template>
                        </el-table-column>
                    </el-ctable>
                    <!-- 批量操作 -->
                    <el-bulk target="tableHomeCellList" 
                        :list="selectedRows" 
                        row-key="serial_number" 
                        :message="{subTitle: '<%=rb.getString("XiaoZhanBianMa")%>'}">
                        <template slot="button">
                            <el-button type="primary" @click="syncList"><%=rb.getString("TongBu")%></el-button>
                            <el-button type="primary" @click="rebootList" style="margin-right: 10px;"><%=rb.getString("ChongQi")%></el-button>
                        </template>
                    </el-bulk>

                    <el-cmenu ref="menu" :data="menus" @click="menuClick"></el-cmenu>
                </el-tab-pane>
                <!-- TOPO -->
                <el-tab-pane label="Map" name="map">
                    <div id="enb_topo_ctner" class="nocontent-loading" style="height: 100%;"></div>
                </el-tab-pane>
            </el-tabs>
        </div>
        <div v-show="activeName=='table'" style="padding: 6px; background-color: #f6f7fb;" class="split-line"></div>
        <!-- charts -->
        <div v-show="activeName=='table'" class="charts-list">
            <div class="fix-right-slide" id="charts_banel"></div>
            
			<div class="slide-arrow left" onclick="toggleExpand()">
				<i class="el-icon el-icon-down"></i>
			</div>
			<div class="slide-arrow right" onclick="minisize()">
				<i class="el-icon el-icon-down"></i>
			</div>
        </div>

        <!-- sliders -->
        <el-slide ref="ueCount" :title="ueslide.title" :footer="false" @cancel="closeUeSlide">
            <el-ctable :data="ueslide.data" :pagination="false">
                <template slot="toolbar">
                    <div style="padding: 20px;"></div>
                </template>

                <el-table-column label="ueId" prop="ue_id" width="100"></el-table-column>
                <el-table-column label="imsi" prop="imsi" width="120"></el-table-column>
                <el-table-column label="vmac" prop="vmac" width="100"></el-table-column>
                <el-table-column label="<%=rb.getString("CPEName")%>" prop="cpe_name" width="100"></el-table-column>
                <el-table-column label="<%=rb.getString("XiaXingTunTuLv")%>" prop="downlink_rate"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingTunTuLv")%>" prop="uplink_rate"></el-table-column>
                <el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="ip" width="100"></el-table-column>
                <el-table-column label="<%=rb.getString("DuanKou")%>" prop="port" width="80"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingSinr")%>" prop="ulsinr" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingCqi")%>" prop="dlcqi" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_Dlcqi" prop="p_dlcqi" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_Dlcqi" prop="s_dlcqi" width="100"></el-table-column>

                <el-table-column label="<%=rb.getString("ShangXingmcs")%>" prop="ulmcs" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingmcs")%>" prop="dlmcs" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_Dlmcs" prop="p_dlmcs" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_Dlmcs" prop="s_dlmcs" width="100"></el-table-column>

                <el-table-column label="<%=rb.getString("FaSongGongLv")%>(dBm)" prop="txpower" width="100"></el-table-column>
                <el-table-column label="<%=rb.getString("ShangXingbler")%>(%)" prop="uplink_bler" width="100"></el-table-column>

                <el-table-column v-if="ueslide.is436q" label="P_TB1_Downlink_BLER(%)" prop="p1_downlink_bler" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="P_TB2_Downlink_BLER(%)" prop="p2_downlink_bler" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_TB1_Downlink_BLER(%)" prop="s1_downlink_bler" width="100"></el-table-column>
                <el-table-column v-if="ueslide.is436q" label="S_TB2_Downlink_BLER(%)" prop="s2_downlink_bler" width="100"></el-table-column>

                <el-table-column v-if="!ueslide.is436q" label="<%=rb.getString("XiaXingbler")%>(%)" prop="downlink_bler" width="100"></el-table-column>

                <el-table-column label="<%=rb.getString("LuJingSunHao")%>(dBm)" prop="pathloss" width="100"></el-table-column>
            </el-ctable>
        </el-slide>

        <el-slide ref="cpeCount" class="no-padding"
            :title="cpeslide.title" 
            :url="cpeslide.url" 
            :footer="false" 
            :header="false">
        </el-slide>

        <el-slide ref="activeRatio" :title="activeslide.title" :footer="false" @cancel="closeActive">
            <div style='display:flex;flex-direction:column;flex:1 1 auto;height:100%;overflow:auto;'>
                <div id="echart_activeRatio" style="min-height: 300px; width: 100%;"></div>
                <div class="list_activeRatio" style="height: 100%;">
                    <el-ctable id="table_activeRatio" :data="activeslide.data" :pagination="false">
                        <el-table-column label='<%=rb.getString("RiQi")%>' prop='days'></el-table-column>
                        <el-table-column label='<%=rb.getString("BuKeYongShiJianDuan")%>' prop='nousedTime'></el-table-column>
                    </el-ctable>
                </div>
            </div>
        </el-slide>

        <el-slide ref="info" class="no-padding no-border slidebarPanel"
            :title="infoslide.title" 
            :url="infoslide.url" 
            :footer="false" 
            :header="false">
        </el-slide>

        <el-slide ref="period" class="no-padding no-border slidebarPanel" 
            :url="periodslide.url" 
            :header="false" 
            :footer="false">
        </el-slide>

        <el-slide ref="distribute" class="no-padding no-border slidebarPanel" 
            :url="distributeslide.url" 
            :header="false" 
            :footer="false">
        </el-slide>
    </div>

    <!-- 快速设置 功能 -->
    <%@ include file="quickSetting.jsp" %>

    <!-- Limitation 功能 -->
    <%@ include file="limitation.jsp" %>

    <script>
        var enbShowCols = '${showCol}',
            columncell = "${cellColumn}",//未选中的列表标识
            allColumn = "${allColumn}",
            eNodeB_column =[],
            enableCheckbox =  (writableMap['CODE_ENB_REBOOT'] == true || writableMap['CODE_ENB_SYNCHRONIZE'] == true),
            is_reboot_batch = false,
            halobSwitchFlag = '';

        var enbvm = new Vue({
            el: '#cellInfo',
            data() {
                var vm = this;

                return {
                    activeName: 'table',
                    selectedRow: '',
                    menus: [],
                    tbURL: '${ctx}/cell/cpeinfos/queryCpeInfosList.action?monitor=1',
                    queryParams: {
                        TimeZone : timeZone,
                        isDual: false,
                        isMonitor: true,
                        search_text: '',
                        like_fields: 'serial_number,host_name,cell_ip',
                        connection_status: [],
						op_state: '',
						product_model: [],
						model_name: [],
						software_version: [],
						firmware_version: [],
						halob_flag: '',
						group_id: []
                    },

                    showProps: enbShowCols.split(','),
                    enableCheckbox: (writableMap['CODE_ENB_REBOOT'] == true || writableMap['CODE_ENB_SYNCHRONIZE'] == true),
                    selectedRows: [],

                    infoslide: {
                        title: '<%=rb.getString("XinXi")%>',
                        url: ''
                    },
                    ueslide: {
                        title: '',
                        data: '',
                        is436q: false
                    },
                    cpeslide: {
                        title: '',
                        url: '',
                    },
                    activeslide: {
                        title: '<%=rb.getString("XiQuKeYongXiangQing")%>',
                        data: ''
                    },
                    periodslide: {
                        title: '',
                        url: ''
                    },
                    distributeslide: {
                        title: '',
                        url: ''
                    }
                    
                };
            },
            computed: {
                showColums() {
                    var vm = this,
                        columns = vm.getAllDefaultCols(),
                        props = columns.map(function(col){
                            return col.prop;
                        });

                    props = props.filter(function(code){
                        return vm.showProps.includes(code);
                    });

                    return props;
                },
                siteIdLabel(){
                    return siteIdLabel
                },
                siteNameLabel(){
                    return siteNameLabel
                },
            },
            methods: {
                refreshList() {
                    this.$refs.list.refresh();
                },
                loadSuccess(data) {
                	refresh_cellStatusStatistics();
                },
                showExport() {
                    addOrImport.showAddDeviceCard = false;
                    addOrImport.showImportCard = false;
                    $('.export-content').html('');
                    $('.export-content').each(function(idx,item){
                        if($(item).is(':visible')) {
                            $(item).load('${ctx}/cell/cpeinfos/toExportConfig.action',function(html) {
                            })
                        }
                    })
                },
                // 获取所有列
                getAllDefaultCols() {
                    var vm = this,
                        columns = [
                            {label: '<%=rb.getString("GaoJingShu")%>', prop: 'alarm', formatter: vm.alarmFmt, width: 65},
                            {label: '<%=rb.getString("XiaoZhanBianMa")%>', prop: 'serial_number', width: 180},
                            {label: '<%=rb.getString("HostName")%>', prop: 'host_name', width: 150, formatter: vm.nameFmt},
                            {label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>', prop: 'rf_status', width: 95},
                            {label: '<%=rb.getString("ShiFouJiHuo")%>', prop: 'op_state', width: 120},
                            {label: 'ECI', prop: 'CELL_IDENTITY', width: 70},
                            {label: '<%=rb.getString("PCI2")%>', prop: 'PHYCELLID', width: 50},
                            {label: '<%=rb.getString("MMEZhuangTai")%>', prop: 'mme_status', width: 80},
                            {label: '<%=rb.getString("UEShu")%>', prop: 'ue_count', width: 80},
                            {label: '<%=rb.getString("CPELianJieShu")%>', prop: 'cpe_connect', width: 80},
                            {label: 'IP', prop: 'cell_ip', width: 100},
                            {label: 'MAC', prop: 'mac_address', width: 115}
                        ];

                    columns.push({label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>', prop: 'product', width: 110});

                    columns.push({label: '<%=rb.getString("SheBeiXingHaoMing")%>', prop: 'module_type', width: 95});
                    columns.push({label: '<%=rb.getString("SoftwareVersion")%>', prop: 'software_version', width: 120});
                    columns.push({label: '<%=rb.getString("SheBeiZu")%>', prop: 'group_name', width: 130});

                    if("${isSuperAdmin}" == '1'){
                        columns.push({label: '<%=rb.getString("IPSECDiZhi")%>', prop: 'IPSEC_ADDR', width: 100});
                        columns.push({label: '<%=rb.getString("MMEPoolIPSECDiZhi")%>', prop: 'mmepool_ipsec_addr', width: 100});
                    }
                    /* 目前只有Amara支持ups，后面后端会调整逻辑 */
                    if(siteIdShow == 'true'){
                        columns.push({label: siteIdLabel, prop: 'site_id', width: 70});
                    }

                    columns.push({label: '<%=rb.getString("PinDian")%>', prop: 'EARFCNDLINUSE', width: 120});
                    columns.push({label: '<%=rb.getString("TongBuZhuangTai")%>', prop: 'synStatus', width: 120});
                    columns.push({label: '<%=rb.getString("KPIShangBaoZhuangTai")%>', prop: 'pm_report_status', width: 130});
                    columns.push({label: '<%=rb.getString("GPSWeiXingShu")%>', prop: 'gps_satellite_count', width: 75});
                    columns.push({label: '<%=rb.getString("YunXingShiJian")%>', prop: 'up_time', width: 110});
                    columns.push({label: '<%=rb.getString("DiYiCiLianJieShiJian")%>', prop: 'first_online_time', width: 125});
                    columns.push({label: '<%=rb.getString("ShangCiLianJieShiJian")%>', prop: 'LASTINFORMTIME', width: 125});
                    columns.push({label: '<%=rb.getString("JiZhanZhiShi")%>', prop: 'network_model', width: 90});
                    columns.push({label: '<%=rb.getString("FirmwareVersion")%>', prop: 'firmware_version', width: 115});
                    columns.push({label: '<%=rb.getString("GPSBanBen")%>', prop: 'gps_version', width: 90});

                    if(isCloud == 'true') {
                        columns.push({label: '<%=rb.getString("XiaoQuKeYongZhanBi")%>', prop: 'available_rate', width: 70});
                    }
                    //cloud版的支持halob 
                    if( isSupportHalob == 'true') {
                        columns.push({label: '<%=rb.getString("HaloBKaiGuan")%>', prop: 'halob_flag', width: 100});
                    }

                    columns.push({label: '<%=rb.getString("PLMN")%>', prop: 'plmnid', width: 60});
                    columns.push({label: '<%=rb.getString("TAC")%>', prop: 'tac', width: 50});
                    columns.push({label: '<%=rb.getString("GPSJingDu")%>', prop: 'gps_longitude', width: 80});
                    columns.push({label: '<%=rb.getString("GPSWeiDu")%>', prop: 'gps_latitude', width: 70});
                    columns.push({label: '<%=rb.getString("GPSGaoDu")%>', prop: 'gps_height', width: 70});

                    if(writableMap["CODE_ENB_EXPIRY_DATE"] != undefined) {
                        columns.push({label: '<%=rb.getString("YouXiaoQi")%>', prop: 'validity', width: 125});
                        columns.push({label: '<%=rb.getString("SuoDingZhuangTai")%>', prop: 'lock_status', width: 90});
                    }

                    return columns;
                },
                selectionChange(s) {
                    this.selectedRows = s||[];
                },
                closeSyncName() {
                    document.body.click();
                },
                clearSelection() {
                	this.$refs.list.clearSelection();
                },
                syncList() {
                	var vm = this,
                		codes = vm.selectedRows.map(function(item){ return item.serial_number}),
                		rows = vm.$refs.list.getData(),
            			cellCodes = '';
                	
                	rows.map(function(row){
                		if(codes.includes(row.serial_number)) {
                			if(row.connection_status != 'Off'){
                				row.connection_status = 'updating';
                			}
                		}
                	});
                	
                	cellCodes = vm.selectedRows.map(function(item){
            			return item.small_cell_code
            		}).join(',');
                	
            		var params = {
            			cellCodes : cellCodes
            		}
            		$.post("${ctx}/cell/quicksettings/batchSyncCell.action",params,function(data){
            			if(!data["success"]){
            				showMsg("error_msg",data["message"]);
            			}else{
            				vm.clearSelection();
            			}
            		},"json")
                },
                rebootList() {
                	var vm = this,
                		checkedRow = vm.selectedRows,
                		cellCodes = '';
                	
            		checkedRow.map(function(item){
            			cellCodes += item.small_cell_code + ',';
            		});
            		
            		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiSheBei")%>", function (r) {
            			if (r) {
            				showMsg('prompt_msg',"<%=rb.getString("MingLingYiXiaFa")%>")
            				var param = {
            					cellCodes: cellCodes
            				};
            				$.post("${ctx}/task/reboot/batchRebootCell.action", param, function (data) {
            					if (data["success"]) {
            						vm.clearSelection();
            					}else{
            						showMsg('error_msg',data["message"]);
            					}
            				}, "json");
            			}
            		}).addClass("seriousConfirm");
                },
                syncName(code, newName) {
                    var vm = this;

                    $.post('${ctx}/cell/cpeinfos/syncCellName.action?smallCellCode='+code, function(data){
                        if(data.success){
                            
                            vm.closeSyncName();
                        }else{
                            showMsg('error_msg',data["message"]);
                        }
                    },'json'); 
                },
                synchronizeGPS(code) {
                    var vm = this;

                    $.ajax({
                        url: '${ctx}/cell/topo/syncGPSInfo.action',
                        type: 'post',
                        data: {cell_code: code},
                        dataType: 'json',
                        success: function(data) {
                            if(data.success){
                                vm.$refs.list.refresh();
                            }else{
                                showMsg('error_msg',data["message"])
                            }
                        }
                    });
                },
                // table formatters
                connStatusFmt(row, value, index) {

                    return connStatusFormatterSyn(value, row, index);
                },
                alarmFmt(rowDatas, col, value, index) {
                    var alarmCount = rowDatas.alarm_count,
                        alarmServerity = rowDatas.alarm_serverity,
                        smallCellCode = rowDatas.small_cell_code,
                        connection_status = rowDatas.connection_status;
                    // 告警级别转义
                    if(alarmServerity == "31001"){
                        value = "<span class='alarmCritical alarmListSty' onclick='goCellDetailAlarmInfoWin(&quot;"+connection_status+"&quot;,&quot;"+smallCellCode+"&quot;)'>"+alarmCount+"</span>"
                    }else if(alarmServerity == "31002"){
                        value = "<span class='alarmMajor alarmListSty' onclick='goCellDetailAlarmInfoWin(&quot;"+connection_status+"&quot;,&quot;"+smallCellCode+"&quot;)'>"+alarmCount+"</span>"
                    }
                    else if(alarmServerity == "31003"){
                        value = "<span class='alarmMinor alarmListSty' onclick='goCellDetailAlarmInfoWin(&quot;"+connection_status+"&quot;,&quot;"+smallCellCode+"&quot;)'>"+alarmCount+"</span>"
                    }else if(alarmServerity == "31004"){
                        value = "<span class='alarmWarning alarmListSty' onclick='goCellDetailAlarmInfoWin(&quot;"+connection_status+"&quot;,&quot;"+smallCellCode+"&quot;)'>"+alarmCount+"</span>"
                    }else{
                        value = "0"
                    }

                    return value;
                },
                rfStatusFmt(row, value, index) {
                    return RFStatusFormatter(value, row, index);
                },
                mmeStatusFmt(row, value, index) {
                    return mmeStatusFormatter(value, row, index);
                },
                plmnFmt(row, value, index) {
                    var vm = this,
                        list = (value||'').split(','),
                        str = list[0];

                    if(list.length>1) {
                        str += '<a style="color: blue;" onclick="showPopLayer({event: event, html: &quot;<di style=\'padding-top: 15px;display: block;\'>'+value+'</div>&quot;})">[' + list.length
                                +'<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]</a>';
                    }

                    return str;
                },
                ueCountFmt(row, value, index) {
                    return ueCountsFormatter(value, row, index);
                },
                cpeCountFmt(row, value, index) {
                    return cpeCountsFormatter(value, row, index);
                },
                ipAddrFmt(row, value, index) {
                    if(value) {
                        value = '<a href="https://' + value + '" target="_blank" style="color: #4d84ff;">'+value+'</a>';
                    }

                    return value;
                },
                productFmt(row, value, index) {
                    if ("${carrierType}" == "1") {
                        return value;
                    }else {
                        if(row.have_connected == 2) return '';
                        if(value == '--') return '--';

                        return value;
                    }
                },
                capablityFmt(rowData,value,rowIndex){
                    var rowDatas = rowData,
                        rowIndexs = rowIndex,
                        capablity = rowData.capablity;

                    if(capablity == 'enable' && isLWAEnable){
                        value = "<span class='cpeLwaKai' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
                    }else if(capablity =='disable' && isLWAEnable){
                        value = "<span class='cpeLwaWu' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
                    }
                    return value;
                },
                earfcnFmt(row, value, index) {
                    if(value) {
                        var resList = [];
                        value.split(',').map(function(item){
                            resList.push(earfcnFormatter(item, row, index));
                        });
                        
                        return resList.join(',');
                    }else {
                        return '';
                    }
                },
                syncStatusFmt(rowData, value, rowIndex){
                    if (value == null) {
                        return null;
                    } else if (value == "--" ){
                        return "--";
                    } else if (value == ("GPS " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="GPS "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
                    }else if (value == ("1588 " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="1588 "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
                    }else if (value == ("REM " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                        val ="REM "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                        value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
                    }else if (value == "GPS "+ "<%= rb.getString("TongBuChengGong")%>" ) {
                        val ="GPS "+ "<%= rb.getString("TongBuChengGong")%>";
                        value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
                    }else if (value == "1588 "+"<%= rb.getString("TongBuChengGong")%>" ) {	
                        val = "<%= rb.getString("TongBuChengGong")%>";
                        val = "1588 " + val;
                        value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
                    }else if (value == "REM "+"<%= rb.getString("TongBuChengGong")%>") {	
                        val = "<%= rb.getString("TongBuChengGong")%>";
                        val = "REM " + val;
                        value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
                    }else if (value == "<%= rb.getString("WeiTongBu")%>") {	
                        val = "<%= rb.getString("WeiTongBu")%>";
                        value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-nosynomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
                    }
                    return '<div style="min-width: 130px;diplay: line-block;">'+value+'</div>';
                },
                kpiStatusFmt(row, value, index){
                    if(value == null){
                        return null;
                    }else if(value == "off"){
                        value = "<span class='el-icon el-icon-status-kpi-off' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"		
                    }else if(value == "normal"){
                        value = "<span class='el-icon el-icon-status-kpi-normal' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"
                    }else if(value == "broken"){
                        value = "<span class='el-icon el-icon-status-kpi-failure' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"
                    }
                    return value;
                },
                halobStatusFmt(rowData, value, rowIndex) {
                    if ("1" == value) {
                        value = "<span class='el-icon el-icon-status-enable' style='margin-right:5px;font-size: 20px;'></span>On";
                    } else if ("0" == value) {
                        value = "<span class='el-icon el-icon-status-disable' style='margin-right:5px;font-size: 20px;'></span>Off";
                    } else {
                        value = "--";
                    }
                    return value;
                },
                validityFmt(rowData, value, rowIndex) {
                    if(value == '--'){
                        return value;
                    }else{
                        if(rowData.lock_status == 0){
                            return "<span>"+value+"</span>"
                        }else if(rowData.lock_status == 1){
                            return "<span style='color:#D2D2D2'>"+value+"</span>"
                        }else{
                            return ''
                        }
                    }
                },
                lockStatusFmt(rowData, value, rowIndex) {
                    if(value == "--"){
                        return value;
                    }else{
                        if(value == 0){
                            return "<div style='display: flex; align-items: center;'><span class='el-icon el-icon-status-unlock' style='margin-right:5px;font-size:20px;'></span><%=rb.getString("YouXiaoQiJieSuo")%></div>";
                        }else{
                            return "<div style='display: flex; align-items: center;'><span class='el-icon el-icon-operation-lock' style='margin-right:5px;font-size:20px;'></span><%=rb.getString("YouXiaoQiSuoDing")%></div>";
                        }
                    }
                },
                // table sliders
                closeUeSlide() {
                    this.$refs.ueCount.hide();
                },
                closeCpeSlide() {
                    this.$refs.cpeCount.hide();
                },
                closeActive() {
                    this.$refs.activeRatio.hide();
                },
                getUnuseTime(row) {
                    getEnbUnuseTimeData(row);
                },

                tabClick(tab) {
                    if(tab.name == 'map') {
                        loadEnbTopo();
                    }
                },
                optClick(row, evt) {
                    var vm = this,
                        rowDatas = row,
                        status = rowDatas.connection_status,
                        idval = rowDatas.small_cell_code+"",
                        flag = rowDatas.platform_flag,
                        isCA = rowDatas.ca_flag,
                        opState = rowDatas.op_state,
                        ip = rowDatas.cell_ip,
                        rfStatus = rowDatas.rf_status||'',
                        lockStatus = rowDatas.lock_status,
                        validitySwitch = rowDatas.validity_switch,
                        CELL_IDENTITY = rowDatas.CELL_IDENTITY,
                        product = rowDatas.product;
                        serialNumber = rowDatas.serial_number;

                    vm.selectedRow = row;
                    
                    var connectStatus = status;
                    var deviceType = flag; // intel -- 0 ,gaotong -- 1
                    var XinXi = '<%=rb.getString("XinXi")%>';
                    var TongBu = '<%=rb.getString("TongBu")%>';
                    var SheZhi = '<%=rb.getString("SheZhi")%>';
                    var MeiYouQuanXian = "<%=rb.getString("MeiYouQuanXian")%>";
                    var ChongQi = '<%=rb.getString("ChongQi")%>';
                    var RiZhi = '<%=rb.getString("RiZhiShouJi")%>';
                    var MiMaChongZhi = '<%=rb.getString("XiuGaiMiMa")%>';
                    var License = '<%=rb.getString("License")%>';
                    var CaoZuo = '<%=rb.getString("CaoZuo")%>';
                    var registerSAS = '<%=rb.getString("SasZhuCe")%>'; 
                    var deregisterSAS = '<%=rb.getString("ZhuXiaoSAS")%>';
                    var GengDuoCaoZuo = '<%=rb.getString("GengDuoCaoZuo")%>';
                    var WeiHuCaoZuo = '<%=rb.getString("Maintenance")%>';
                    var PeiZhiHuiFu = '<%=rb.getString("PeiZhiHuiFu")%>';
                    var RFname = '<%=rb.getString("ShePinCaoZuo")%>';
                    var RFon = '<%=rb.getString("Kai")%>';
                    var RFoff = '<%=rb.getString("Guan")%>';
                    var YouXiaoQi = '<%=rb.getString("YouXiaoQi")%>';
                    var LiuLiangXianZhi = '<%=rb.getString("LiuLiangXianZhi")%>';
                    var FenBuShi = '<%=rb.getString("FenBuShi")%>';
                    var rfText ,rfText1,rfText2,forceRFText,forceRFStatus;
                    var rizhidisableflag;
                    var rfStatusShowfloag=true;
                    var rfStatus1,rfStatus2;
                    var platformType = rowDatas.platformType,
                        dualCarrierType = rowDatas.dual_carrier_type,
                        moduleType = rowDatas.module_type;
                    
                    var showFlagList={
                            CODE_ENB_REBOOT:false,
                            CODE_ENB_RESET_CONFIG:false,
                            CODE_ENB_LOGS:false,
                            CODE_ENB_SYNCHRONIZE:false,
                            CODE_ENB_HALOB_ENABLE:false,
                            CODE_ENB_RF_ENABLE:false,
                            CODE_ENB_ACTIVE:false,
                            CODE_ENB_SAS_RF_ENABLE:false,
                            CODE_ENB_SAS_ENABLE:false,
                            CODE_ENB_LOCK:false,
                            CODE_ENB_CHANGE_PASSWORD:false,
                            CODE_ENB_INFORMATION:false,
                            CODE_ENB_SETTINGS:false,
                            CODE_ENB_EXPIRY_DATE:false,
                            CODE_ENB_EXPIRY_DATE_LOCK:false,
                            CODE_ENB_DISTRIBUTED:false,
                            CODE_ENB_TRAFFIC_LIMITATION:false,
                        },
                        paramsCode={
                            smallCellCode:idval
                        },
                        maintenanceShow = false,
                        actionShow = false;

                    $.ajax({
                        type:'POST',
                        url:'${ctx}/cell/cpeinfos/getENBOperationItem.action',
                        data:paramsCode,
                        async:false,
                        dataType:'json',
                        success:function(data){
                            var operationData = data;
                            if(operationData.Maintenance.length > 0 ){
                                maintenanceShow = true
                                operationData.Maintenance.map((item)=>{
                                    showFlagList[item] = true;
                                });
                            }
                            if(operationData.Actions.length > 0 ){
                                actionShow = true
                                operationData.Actions.map((item)=>{
                                    showFlagList[item] = true;
                                })
                            }
                            if(operationData.others.length > 0 ){
                                operationData.others.map((item)=>{
                                    showFlagList[item] = true;
                                })
                            }
                            
                        },
                    });

                    if(connectStatus != 'Off'){
                        rizhidisableflag = false
                    }else{
                        rizhidisableflag = true;
                    }
                
                    //配置恢复 判断
                    var resetDisableFlag;
                    var tongbuDisableFlag;
                    if(connectStatus != 'Off'){
                        tongbuDisableFlag = false;
                        resetDisableFlag = false;
                    }else{
                        tongbuDisableFlag = true;
                        resetDisableFlag = true;
                    }
                    
                    
                    //判断是否有licence菜单选项
                    var paramLicense={
                            small_cell_code:idval
                        }
                    
                    //判断halob菜单项是否显示  可用
                    //当基站不在线  且不是集中模式   不显示但      非集中模式 且基站在线 菜单可用     非集中模式 且基站不在线 菜单不可用
                    var halobName = "";
                    var disableHalobFlag;
                    var halobSwitch;
                    if(connectStatus != 'Off') disableHalobFlag = false;
                    else disableHalobFlag = true;
                    
                    $.ajax({
                        type:'POST',
                        url:'${ctx}/cell/cpeinfos/checkCellHasLicenseInfo.action',
                        data:paramLicense,
                        async:false,
                        dataType:'json',
                        success:function(data){
                            
                            //控制halob菜单
                            if(data["halobFlag"] == 0){
                                //关闭Halob
                                halobName = '<%=rb.getString("KaiQiHaloB")%>';
                                halobSwitch = 1;
                            }else if(data["halobFlag"] == 1){
                                //开启Halob
                                halobSwitch = 0;
                                halobName = '<%=rb.getString("GuanBiHaloB")%>';
                            }else{
                                //集中模式   集中模式不显示 halob的菜单
                                halobName = '<%=rb.getString("HaloBKaiGuan")%>';
                            }
                        },
                    });
                    //判断重启 flag
                    var chongQidisableFlag;
                    if(connectStatus != 'Off'){
                        chongQidisableFlag = false;
                    }else{
                        chongQidisableFlag = true;
                    }
                    

                    //判断激活状态
                    var opStateName = "";
                    var opStateFlag;
                    var opSwitch = ""
                    if(connectStatus != 'Off'){
                        opStateFlag = false;
                    }else{
                        opStateFlag = true;
                    }
                    if(opState == "1"){
                        opStateName = '<%=rb.getString("goJiHuo")%>';
                        opSwitch = "0";
                    }else{
                        opStateName ='<%=rb.getString("JiHuo")%>';
                        opSwitch = "1";
                    }
                    
                    //基站不在线 RF开关不可操作
                    if(connectStatus != 'Off'){
                        disableRFFlag = false;
                    }else{
                        disableRFFlag = true;
                    }
                    var showCell = false;
                    
                    // 根据RF开关状态，为操作显示的文字赋值 
                    if(rfStatus === '' || rfStatus =='--'){
                        rfStatusShowfloag=false;
                    }else if ( rfStatus == "on" || rfStatus == 1){
                        showCell = false;
                        rfText = RFname + ' '+ RFoff;
                    }else if( rfStatus == "off" || rfStatus === 0){
                        showCell = false;
                        rfText = RFname + ' '+ RFon;
                    }else{
                        showCell = true;
                        rfText = RFname;
                        rfStatus = rfStatus.split(",");
                        rfStatus1 = rfStatus[0];
                        rfStatus2 = rfStatus[1];
                        if(rfStatus1 == "on"){
                            rfText1 = 'Cell1  '+ RFoff;
                        }
                        if(rfStatus1 == "off"){
                            rfText1 = 'Cell1  '+ RFon;
                        }
                        if(rfStatus2 == "on"){
                            rfText2 = 'Cell2  '+ RFoff;
                        }
                        if(rfStatus2 == "off"){
                            rfText2 = 'Cell2  '+ RFon;
                        }
                    }
                    
                    //判断基站锁定状态
                    var lockText;
                    var lockOperFlag = true;
                    if(lockStatus == 0){//未锁定状态
                        lockText = '<%=rb.getString("YouXiaoQiSuoDing")%>';
                    }else if(lockStatus == 1){//锁定状态
                        lockText = '<%=rb.getString("YouXiaoQiJieSuo")%>'
                    }
                    if(lockStatus == "--"){
                        lockOperFlag = true
                    }else{
                        lockOperFlag = false;
                    }
                    var sasSwitch = ("${sasSwitch}" == "1");
                    var rfForceShowFlag = false;
                    if ( rowDatas.sasEnable == "on"){
                        rfForceShowFlag = true;
                        if (rowDatas.autoOrForceRF == "false" ){
                            forceRFText = "<%= rb.getString("QiangZhiGuanBiRF")%>";
                            forceRFStatus = "true"
                        }else if (rowDatas.autoOrForceRF == "true" ){
                            forceRFText = "<%= rb.getString("SASZiDongKongZhiRF")%>"
                            forceRFStatus = "false"
                        }else if(rowDatas.autoOrForceRF == null || rowDatas.autoOrForceRF == undefined){
                            rfForceShowFlag = false;
                        }
                        
                    }
                    var maintenanceOpChild = [
                        {row: row, label:ChongQi,id:36,cls:'CODE_ENB_REBOOT hidden',show:showFlagList.CODE_ENB_REBOOT,disable: chongQidisableFlag,cell_code:idval,product:product},
                        {row: row, label:PeiZhiHuiFu,id:42,cls:'CODE_ENB_RESET_CONFIG hidden',show:showFlagList.CODE_ENB_RESET_CONFIG,disable: resetDisableFlag,cell_code:idval},
                        {row: row, label:RiZhi,id:32,cls:'CODE_ENB_LOGS hidden',show:showFlagList.CODE_ENB_LOGS,disable: rizhidisableflag},
                    ]
                    var moreOpChild = [
                            {row: row, label:TongBu,id:31,cls:'CODE_ENB_SYNCHRONIZE hidden',show:showFlagList.CODE_ENB_SYNCHRONIZE,disable: tongbuDisableFlag,cell_code:idval},//show为false不显示 disable为false显示禁用
                            {row: row, label: registerSAS, id: 35, cls:'', show: sasSwitch},
                            {row: row, label: deregisterSAS, id: 37, cls:'', show: sasSwitch},
                            {row: row, label:opStateName,id:41,cls:'CODE_ENB_ACTIVE hidden',show:showFlagList.CODE_ENB_ACTIVE,disable: opStateFlag,activeStatus:opSwitch,cell_code:idval},
                            {row: row, label:rfText,id:43,cls:'CODE_ENB_RF_ENABLE hidden',cell_code:idval, show:showFlagList.CODE_ENB_RF_ENABLE && rfStatusShowfloag && !showCell,disable:disableRFFlag,rfStatus: rfStatus,cellNumber:''},
                            {row: row, label:rfText,id:431,cls:'CODE_ENB_RF_ENABLE hidden',cell_code:idval, show:showFlagList.CODE_ENB_RF_ENABLE && rfStatusShowfloag && showCell,disable:disableRFFlag,
                                children: [
                                    {row: row, label:rfText1,id:432,cls:' ',rfStatus:rfStatus1,cell_code:idval,show:showCell,cellNumber:1},
                                    {row: row, label:rfText2,id:432,cls:' ',rfStatus:rfStatus2,cell_code:idval,show:showCell,cellNumber:2}
                                ]
                            },
                            {row: row, label:halobName,id:40,cls:'',show:showFlagList.CODE_ENB_HALOB_ENABLE,disable: disableHalobFlag,cell_code:idval,halob_switch:halobSwitch},
                            {row: row, label:forceRFText,id:48,cls:'CODE_ENB_RF_ENABLE hidden',serialNumber:serialNumber, show:showFlagList.CODE_ENB_SAS_RF_ENABLE && rfForceShowFlag,disable:disableRFFlag,rfStatus: forceRFStatus},
                            {row: row, label:lockText,id:44,cls:'CODE_ENB_EXPIRY_DATE hidden',cell_code:idval,lock_status:lockStatus,show:!lockOperFlag}
                        ],
                        isSubDevice = false,
                        isUnusable = false,
                        isSettable = false;
                    
                    if(rowDatas.have_connected == 2) {// 是否真实可用站
                        moreOpChild = [];
                        isSubDevice = true;
                        isUnusable = true;
                        isSettable = true;
                    }
                    var id4param = idval;
                    // 新类型QA_436Q_DC的处理
                    if(['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC','NEU430_DC'].includes(platformType) && dualCarrierType == 2) {// 辅波查看不可操作，更多操作只有射频
                        isSubDevice = true;
                        isUnusable = false;
                        isSettable = false;
                        id4param = id4param.substr(0,id4param.length-2);
                    }
                    // 4860辅站
                    if(['Intel_CR_DC','MLN_DC'].includes(platformType) && dualCarrierType == 2) {
                        moreOpChild = [];
                        isSubDevice = true;
                        isUnusable = true;
                        isSettable = false;
                    }
                    var validityFlag = false;
                    if(validitySwitch == '--'){
                        validityFlag = true
                    }else{
                        validityFlag = false
                    }
                    
                    var limitDisable = true, limitReg = /(LBS|pBSL)\S*/;
                    if(limitReg.test(moduleType)) limitDisable = false
                    
                    
                    vm.menus = [
                            {row: row, label:XinXi,id:1,show:showFlagList.CODE_ENB_INFORMATION, disable: isSubDevice,cls:'el-icon-operation-info el-icon ',small_cell_code : idval,status : status,CELL_IDENTITY : CELL_IDENTITY},
                            {row: row, label:YouXiaoQi,id:5,show:showFlagList.CODE_ENB_EXPIRY_DATE, cls:'el-icon-operation-date el-icon CODE_ENB_EXPIRY_DATE hidden',disable:validityFlag},
                            {row: row, label:LiuLiangXianZhi,id:'limit',show:showFlagList.CODE_ENB_TRAFFIC_LIMITATION, cls:'el-icon-operation-limitation el-icon CODE_ENB_TRAFFIC_LIMITATION hidden',disable: limitDisable},
                            {row: row, label:SheZhi,id:2,cls:'CODE_ENB_SETTINGS hidden el-icon-operation-settings el-icon',show:showFlagList.CODE_ENB_SETTINGS, disable: isSettable},
                            {row: row, label:FenBuShi,id:6,cls:'el-icon-operation-distributed el-icon',small_cell_code : idval,show:showFlagList.CODE_ENB_DISTRIBUTED},
                            {row: row, label:WeiHuCaoZuo,id:4,cls:'el-icon-operation-maintenance el-icon',show:maintenanceShow,
                                child: maintenanceOpChild, disable: isUnusable
                            },
                            {row: row, label:GengDuoCaoZuo,id:3,cls:'el-icon-operation-more-circle el-icon',show:actionShow,
                                child: moreOpChild, disable: isUnusable
                            }
                        ];
                    //判断菜单的位置
                    $.ajax({
                        url: '${ctx}/cell/quicksettings/getSettingGroupTree.action',
                        dataType: 'json',
                        async: false,
                        type: 'post',
                        data: {
                            title:'Settings',
                            smallCellCode: id4param
                        },
                        success: function(json){
                            if(json){
                                // 清空form元素
                                $('#enbSetting_slide_body').html('');
                                Render.tableCollector = {};
                                // 导航容器
                                var navCtn = $('#quick_setting_nav');
                                navCtn.empty();
                                json.map(function(item,idx){
                                    var disabled = false;
                                    if(item.code == "Basic"){
                                        disabled = false;
                                    }else{
                                        if(connectStatus == 'Off') disabled = true;
                                        
                                        if(['QA_436Q_DC','NEU430_DC','Intel_CR_DC','MLN_DC'].includes(platformType) && dualCarrierType == 2) { // 新类型QA_436Q_DC的处理，辅波只有basic可以设置
                                            disabled = true;
                                        }
                                        
                                        if(['QA_436Q_DC'].includes(platformType) && dualCarrierType == 2 && item.text == 'LTE') {
                                            disabled = false;
                                        }
                                    }
                                    var iTitle = item.text;
                                    var mItem = {platform: rowDatas.platform_flag,id: item.code, code: item.id,text: iTitle,disable:disabled, subTitle: '(<%=rb.getString("XiaoZhanBianMa")%><%=rb.getString("MaoHao")%>'+rowDatas.serial_number+'<%=rb.getString("DouHao")%><%=rb.getString("HostName")%><%=rb.getString("MaoHao")%>'+(rowDatas.host_name||'')+')'};
                                    //data[2].children.push(mItem);
                                    // 生成Nav导航，绑定click事件
                                    var navItem = $('<li>'+iTitle+'</li>');
                                    
                                    if(disabled) $(navItem).css({'cursor':'not-allowed',color: 'silver'});
                                    if(!isLWAEnable && mItem.text=="LTE-TURBO")  $(navItem).css({'display':'none'});

                                    navCtn.append(navItem);
                                    navItem.on('click',function(){
                                        if(disabled) {
                                            return false;
                                        }
                                        turnTabs(navItem);
                                        if(mItem.text=="LTE-TURBO"){
                                            window.sessionStorage.setItem('apInfoSn',rowDatas.serial_number)
                                            var url = '${ctx}/cell/ap/toAPInfo.action'
                                            var params = {
                                                smallCellCode:rowDatas.small_cell_code,
                                                enbSerialNumber:rowDatas.serial_number,
                                                connectionStatus:rowDatas.connection_status
                                            }
                                            $('#enbSetting_slide_body').load(url,params,function(){
                                                var ctner = document.querySelector('#enbSetting_slide_body');
                                                var scripts = ctner.querySelectorAll('script');
                                                setTimeout(function() {
                                                    Array.from(scripts).map(function(script) { /* 执行远程的脚本 */
                                                        if (ctner.contains(script)) {
                                                            ctner.removeChild(script);
                                                        }
                                                        var newScript = document.createElement('script');
                                                        newScript.type = 'text/javascript';
                                                        newScript.innerHTML = script.innerHTML;
                                                        ctner.appendChild(newScript);
                                                    });
                                                }, 0);
                                                
                                                setTimeout(function() {
                                                    try{
                                                        $.parser.parse(ctner);
                                                    }catch(e){}
                                                }, 0);
                                            });
                                        }else{
                                            var params = Render.getFormDatas($('#enbSetting_slide_body'));
                                            var edit = !isEmptyJson(params)
                                            var addEdit = false
                                            if(addEdit){
                                                turnTabs(navItem);
                                                var rowCode = $('#enbSetting_slide').data('params').smallCellCode;
                                                settingTabClick(mItem.code,rowCode,mItem.text,mItem.subTitle,mItem.platform);
                                            }else{
                                                if(edit){// 参数有变动时，确认提示
                                                    $.messager.confirm('Confirm','<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',function(r){
                                                        if(r){
                                                            turnTabs(navItem);
                                                            var rowCode = $('#enbSetting_slide').data('params').smallCellCode;
                                                            settingTabClick(mItem.code,rowCode,mItem.text,mItem.subTitle,mItem.platform);
                                                        }
                                                    });
                                                }else{// 无变动直接跳转
                                                    turnTabs(navItem);
                                                    var rowCode = $('#enbSetting_slide').data('params').smallCellCode;
                                                    settingTabClick(mItem.code,rowCode,mItem.text,mItem.subTitle,mItem.platform);
                                                }
                                            }
                                        
                                        }
                                        
                                    })
                                });

                            }else{
                                data[2].disable = true;
                            }
                        }
                    });
                    
                    vm.$nextTick(function(){
                        document.body.click();
                        vm.$refs.menu.show(evt);
                    })
                },
                menuClick(item) {
                    var vm = this,
                        row = item.row;

                    var rowCode = row.small_cell_code;

                    switch(item.id){
                    case 1: //信息
                        goCellDetailParamInfoWin("enbStatistics", row.small_cell_code, row.connection_status, row.CELL_IDENTITY);
                        
                        break;
                    case 2://设置
                        halobSwitchFlag = row.halob_flag;
                        jumpToSetting(rowCode, item.label, row.platform);
                        
                        break;
                    case 3://操作
                        
                        break;
                    case 4://操作
                        
                        break;
                    case 31://同步
                        refreshCell(item.cell_code);
                       
                        break;
                    case 32://日志
                        confirmImmediateCollectLogFile('queryCollectPopdiv');
                        
                        break;
                    case 35://sas注册
                        registerSas();
                        
                        break;
                    case 36://重启
                        cellReboot(item.cell_code,row.product);
                        
                        break;
                    case 37://deregister
                        deregisterSAS();
                        
                        break;
                    case 40://Halob开启关闭
                        var cellCode = item.cell_code;
                        var halobSwitch = item.halob_switch;
                        openCloseHalob(cellCode,halobSwitch);//开启关闭Halob操作
                        
                        break;
                    case 41://激活状态
                        var cellCode = item.activeStatus;
                        var smallcellCode =  item.cell_code;
                        activeOpStatus(cellCode,smallcellCode);//开启关闭Halob操作
                        
                        break;
                    case 42://恢复默认配置
                        var smallcellCode =  item.cell_code
                        configReset(smallcellCode);
                        
                        break;
                    case 43://RF 
                        var smallcellCode =  item.cell_code
                        setRFStatus(smallcellCode, item.rfStatus);
                        
                        break;
                    case 5:
                        var smallcellCode =  item.cell_code
                        setEffectPeriod();
                        
                        break;
                    case 44:
                        var smallcellCode = item.cell_code;
                        var lockStatus = item.lock_status;
                        setLockStatus(smallcellCode,lockStatus);
                        
                        break;
                    case 431:
                        
                        break;
                    case 432://RF 关闭
                        var smallcellCode =  item.cell_code
                        setRFStatus(smallcellCode,item.rfStatus,item.cellNumber)
                        
                        break; 
                    case 48://autoOrForceRF
                        var serialNumber =  row.serial_number;
                        setForceRFStatus(serialNumber, item.rfStatus)
                        
                        break; 
                    case 'limit': // Limitation
                        setLimitation(rowCode);
                        
                        break; 
                    case 6://分布式基站
                        distributeSn("enbDistribute",row.small_cell_code);
                        
                        break;
                    }
                },
                goPeriod() {
                    var vm = this;

                    vm.periodslide.url = '${ctx}/cell/cpeinfos/toCellValidity.action';

                    vm.$refs.period.showSlide();
                },
                goDistribute(code) {
                    var vm = this;
                    var url = '${ctx}/cell/nxp/toCellInfo.action?smallCellCode='+code;

                    vm.distributeslide.url = url;
                    vm.$refs.distribute.showSlide();
                },
                handerClose() {
                    this.$refs.menu.hide();
                },
                closeInfo() {
                    this.$refs.info.hide();
                }
            },
            mounted() {
                //加载 新建设备组及 导入功能
                loadHTML(document.querySelector('#addDeviceWarp'),{
                    url: '${ctx}/cell/cpeinfos/addOrImportDevice.action',
                    success: function() {
                        closeLoading();
                    }
                });

                loadHTML(document.querySelector('#charts_banel'),{
                    url: '${ctx}/cell/cpeinfos/getChartPage.action'
                });

                loadHTML(document.querySelector('#toolbar_tableHomeCellList'),{
                    url: '${ctx}/cell/cpeinfos/toEnbQuery.action',
                    success: function() {
                        loadHTML(document.querySelector('#showOrHideItem'),{
                            url: '${ctx}/cell/cpeinfos/toCellSort.action'
                        });
                    }
                });

                $('#enb_export_form').remove();
            }
        });
    </script>
    <script>
        var enbtopoLoaded = false,
            /* 筛选展示逻辑处理 */
            filterActiveAlarmUrl = '${ctx}/fault/view/queryViewPageList.action',	
            filterHisAlarmUrl = '${ctx}/fault/view/queryViewPageList.action',
            isInfoAlarmVisible = false;
        var rules = [
                {
                    title: '<%=rb.getString("YanZhongChengDu")%>',
                    match: function(code){
                        return code == 'alarm_serverity_value';
                    },
                    action: function(code,e){
                        var key = 'alarmServerity';
                        var isInfoAlarmVisible = $('#alarmTabsHisActive').hasClass('active'); 
                        var params = {};
                        if(isInfoAlarmVisible) params = $('#enbHistoryAlarm').datagrid('options').queryParams;
                        else params = $('#enbActiveAlarm').datagrid('options').queryParams ; 
                        params['source'] = 'enb';
                        /* 配置筛选菜单可选项，可以通过接口获取数据 */
                        var data = [];
                        var url = filterActiveAlarmUrl;
                        if(isInfoAlarmVisible) url = filterHisAlarmUrl;
                        datas = [{text:'Critical',value:31001},{text:'Major',value:31002},{text:'Minor',value:31003},{text:'Warning',value:31004}];
                        datas.map(function(item){
                                    var row = {name:'alarm_serverity',label:item.text,value:item.value};
                                    data.push(row);
                                });
                        
                            data.map(function(item){

                                if(params[key]){
                                    var vals = params[key].split(',');
                                    if(vals.includes(item.value+'')) {
                                        item.checked = true;
                                    }
                                }else{ 
                                    item.checked = true;
                                } 
                            }); 
                            
                            /* 生成筛选菜单 */
                            filterMenu({
                                data: data,
                                fn: function(tips){
                                    tips.css({left:e.x-$('#menuAnimate').width(),top: e.clientY-30});
                                    $('#mainpage').append(tips);
                                },
                                click: function(values){
                                    params[key] = values;
                                    if(isInfoAlarmVisible) $('#enbHistoryAlarm').datagrid('reload');
                                    else $('#enbActiveAlarm').datagrid('reload');
                                }
                            });
                    }
                }
            ];

        /* 菜单隐藏处理 */
        $('#mainpage').mousedown(function(){
            try{
                var target = event.target, list = Array.from(target.classList),
                    plist = Array.from(target.parentNode.classList);
                if(!(list.includes('filter-menu') || list.includes('filter-item') || plist.includes('filter-item'))){
                    $('.filter-menu').hide();
                }
            }catch(e){}
        });

        function loadEnbTopo() {
            if(enbtopoLoaded) return;
            enbtopoLoaded = true;

            $('#enb_topo_ctner').load('${ctx}/cell/topo/toMonitorTopo.action',function(html){
                $.parser.parse(this);
            });
        }

        function RFStatusFormatter(value, rowData, rowIndex){
            if (value == null || value == "") {
                return null;
            }
            if (value == '--') {
                return value;
            }
            
            if (value == "on" || value == 1) {
                value = "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF'><span class='el-icon el-icon-status-RF'></span> ON</div>"
                    + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
            } else if (value == "off" || value == 0) {
                value = "<div class='rfDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF'><span class='el-icon el-icon-status-RF'></span> OFF</div>"
                    + "<div class='mmeDetails MMEDetail'>RF Status : <span class='mmeStauts'></span></div>";
            } else{
                var rfStatus = value.split(",");
                var rfStatusText ='';
                    if(rfStatus[0] == "on"){
                        rfStatusText = rfStatusText + "<div class='rfConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF1'><span class='el-icon el-icon-status-RF1'></span> ON</div>";
                    }
                    if(rfStatus[0] == "off"){
                        rfStatusText = rfStatusText + "<div class='rfDisConnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF1'><span class='el-icon el-icon-status-RF1'></span> OFF</div>";
                    }
                    if(rfStatus[1] == "on"){
                        rfStatusText = rfStatusText + "<div class='rfConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='RF2'><span class='el-icon el-icon-status-RF2'></span> ON</div>";
                    }
                    if(rfStatus[1] == "off"){
                        rfStatusText = rfStatusText + "<div class='rfDisConnItem' style='margin-left:10px;' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='RF2'><span class='el-icon el-icon-status-RF2'></span> OFF</div>";
                    }
                value = rfStatusText + "<div class='mmeDetails MME1Detail'>RF1 Status : <span class='mme1Stauts'></span></div>"+ "<div class='mmeDetails MME2Detail'>RF2 Status : <span class='mme2Stauts'></span></div>";  
            }
            return value;
        }

        function toMMEDetail(ele,index){
            var thisTop = $(ele).offset().top;
            var allHeight = $(document).height();
            var thisLeft = $(ele).offset().left;
            var allWidth = $(document).width();
            
            if((allHeight - thisTop) < 200){
                $(ele).siblings(".mmeDetails").css("top","-55px");
            }
            if((allWidth - thisLeft) < 200){
                $(ele).siblings(".mmeDetails").css("left","-90px");
            }
            var eleClass = $(ele).attr('type'); 
            if(eleClass.includes("RF")){
                var YiLianJie = '<%= rb.getString("KaiQi")%>';
                var WeiLianJie = '<%= rb.getString("GuanBi")%>';
            }
            if(eleClass.includes("MME")){
                var YiLianJie = '<%= rb.getString("MMEYiLianJie")%>';
                var WeiLianJie = '<%= rb.getString("MMEWeiLianJie")%>';
            }
            $(".mmeDetails").hide();
            if(eleClass == 'RF1' || eleClass == 'MME1'){
                $(ele).siblings(".MME1Detail").fadeToggle();
                if(index == 1){
                    $(".mme1Stauts").text(YiLianJie);
                }else if(index == 0){
                    $(".mme1Stauts").text(WeiLianJie);
                }
            }else if(eleClass == 'RF2' || eleClass == 'MME2'){
                $(ele).siblings(".MME2Detail").fadeToggle();
                if(index == 1){
                    $(".mme2Stauts").text(YiLianJie);
                }else if(index == 0){
                    $(".mme2Stauts").text(WeiLianJie);
                }
            }else if(eleClass == 'RF' || eleClass == 'MME'){
                $(ele).siblings(".MMEDetail").fadeToggle();
                if(index == 1){
                    $(".mmeStauts").text(YiLianJie);
                }else if(index == 0){
                    $(".mmeStauts").text(WeiLianJie);
                }
            }	
        }

        function hideMMEDetail() {
            $(".mmeDetails").fadeOut(100);
        }

        function mmeStatusFormatter(value, rowData, rowIndex) {
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
                var mmePools = value.split(",");
                var ret = "";
                for(var i = 0; i < mmePools.length; i++ ){
                    var mme = mmePools[i];
                    var mmeArr = mme.split("=");
                    
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
                var lastChar = ret.charAt(ret.length - 1);
                if("," == lastChar){
                    ret = ret.substring(0,ret.length - 1);
                }
                value = ret + "<div class='mmeDetails MME1Detail'>MME1 IP : "+rowData.mme_pool_1 + "</br>MME1 Status : <span class='mme1Stauts'></span></div>"+ "<div class='mmeDetails MME2Detail'>MME2 IP : "+rowData.mme_pool_2 + " <br/>MME2 Status : <span class='mme2Stauts'></span></div>";  
            }
            
            return value;
        }

        function ueCountsFormatter(value, rowData, rowIndex) {
            if(value == 0 ){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>0</a>"; 
            }else if(value == -1 || value == null){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
            }else{
                if(['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC'].includes(rowData.platformType)) {
                    return "<a style='color:#1DA3FC;text-decoration:underline' href='#' onclick='getueCountsData(&quot;" + rowData.small_cell_code + "&quot;,&quot;" + rowData.host_name + "&quot;,&quot;" + rowData.serial_number + "&quot;,\"enodeb_ueCounts\",true)'>"+value+"</a>"; 
                }else if(rowData.ca_flag == "1" || rowData.platform_flag == "1"){//CA的站不支持UE数详细信息展示 ca_flag==1 表示为CA站
                    return value;
                }else{
                    return "<a style='color:#1DA3FC;text-decoration:underline' href='#' onclick='getueCountsData(&quot;" + rowData.small_cell_code + "&quot;,&quot;" + rowData.host_name + "&quot;,&quot;" + rowData.serial_number + "&quot;,\"enodeb_ueCounts\")'>"+value+"</a>"; 
                }
            }
            
        }

        function getueCountsData(code,cellName,sn,divId,is436q) {
            var enb_code = code,snNumber= sn,cellName = cellName,
                params = {
                    enb_code: enb_code
                },
                ueCounts_title = '<%=rb.getString("UEShu")%>(<%=rb.getString("XiaoZhanBianMa")%>:' + snNumber  + ','+ '<%=rb.getString("HostName")%>:'+cellName + ')';

            enbvm.ueslide.title = ueCounts_title;
            enbvm.ueslide.is436q = is436q;

            // 获取ue数据
            $.post("${ctx}/system/device/enb/uedata/getENBUeStatisticsDataList.action",params,function(data){
                if(Array.isArray(data)){
                    enbvm.$refs.ueCount.showSlide();
                    enbvm.ueslide.data = data? data.rows:[];
                }else{
                    toast("<%=rb.getString("BuZhiChiUEShuZuanQu")%>","#mainpage","",true);
                }
            },"json");
        }

        function cpeCountsFormatter(value, rowData, rowIndex){
            if(value == -1 || value == null){
                return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
            }else if(value === 0 || value === '0') {
                return value;
            }else{
                var is436Q = ['QA_436Q_CA','QA_436Q_SC','QA_436Q_DC'].includes(rowData.platformType);
                
                return "<a style='color:#1DA3FC;text-decoration:underline' href='#' onclick='getueCpeCountsData(&quot;" 
                        + rowData.small_cell_code + "&quot;,&quot;" + rowData.CELL_IDENTITY + "&quot;,&quot;" + rowData.PHYCELLID 
                        + "&quot;,&quot;" + rowData.EARFCNDLINUSE + "&quot;,&quot;" + rowData.serial_number + "&quot,&quot;" 
                        + rowData.host_name + "&quot,\"enodeb_cpe_connect\","+is436Q+")'>"+value+"</a>"; 
            }
        }

        function getueCpeCountsData(code,eci,pci,earfcn,sn,cellname,divId,is436Q) {
            var url = '${ctx}/cell/ap/toUEDetailPage.action',
                obj ={
                    smallCellCode: code,
                    eci: eci,
                    pci: pci,
                    earfcn: earfcn
                };

            enbvm.cpeslide.url = url;
            
            window.sessionStorage.setItem('ueSmallCellCode',code);
            window.sessionStorage.setItem('ueEci',eci);
            window.sessionStorage.setItem('uePci',pci);
            window.sessionStorage.setItem('ueEarfcn',earfcn);
            window.sessionStorage.setItem('snNumber',sn);
            window.sessionStorage.setItem('cellName',cellname);
            window.sessionStorage.setItem('is436Q',is436Q);

            enbvm.$refs.cpeCount.showSlide(obj, function() {
                $("#tabAlarmCli").click();
            });
        }

        function earfcnFormatter(value, rowData, rowIndex){
            if(value == null){
                return "";
            }
            var EARFCN = value;//正常显示的频点值  还需要将此值转换成频率
            var frequency = 0;
            if (EARFCN >= 36000 && EARFCN <= 36199) { //tdd-band 33
                frequency = 1900 + 0.1 * (EARFCN - 36000);
            } else if (EARFCN >= 36200 && EARFCN <= 36349) { //tdd-band 34
                frequency = 2010 + 0.1 * (EARFCN - 36200);
            } else if (EARFCN >= 36350 && EARFCN <= 36949) { //tdd-band 35
                frequency = 1850 + 0.1 * (EARFCN - 36350);
            } else if (EARFCN >= 36950 && EARFCN <= 37549) { //tdd-band 36
                frequency = 1930 + 0.1 * (EARFCN - 36950);
            } else if (EARFCN >= 37550 && EARFCN <= 37749) { //tdd-band 37
                frequency = 1910 + 0.1 * (EARFCN - 37550);
            } else if (EARFCN >= 37750 && EARFCN <= 38249) { //tdd-band 38
                frequency = 2570 + 0.1 * (EARFCN - 37750);
            } else if (EARFCN >= 38250 && EARFCN <= 38649) { //tdd-band 39
                frequency = 1880 + 0.1 * (EARFCN - 38250);
            } else if (EARFCN >= 38650 && EARFCN <= 39649) { //tdd-band 40
                frequency = 2300 + 0.1 * (EARFCN - 38650);
            } else if (EARFCN >= 39650 && EARFCN <= 41589) { //tdd-band 41
                frequency = 2496 + 0.1 * (EARFCN - 39650);
            } else if (EARFCN >= 41590 && EARFCN <= 43589) { //tdd-band 42
                frequency = 3400 + 0.1 * (EARFCN - 41590);
            } else if (EARFCN >= 43590 && EARFCN <= 45589) { //tdd-band 43
                frequency = 3600 + 0.1 * (EARFCN - 43590);
            } else if (EARFCN >= 18000 && EARFCN <= 18599) { //fdd-band 1
                frequency = 1920 + 0.1 * (EARFCN - 18000);
            } else if (EARFCN >= 0 && EARFCN <= 599) {
                frequency = 2110 + 0.1 * (EARFCN - 0);
            } else if (EARFCN >= 18600 && EARFCN <= 19199) { //fdd-band 2
                frequency = 1850 + 0.1 * (EARFCN - 18600);
            } else if (EARFCN >= 600 && EARFCN <= 1199) {
                frequency = 1930 + 0.1 * (EARFCN - 600);
            } else if (EARFCN >= 19200 && EARFCN <= 19949) { //fdd-band 3
                frequency = 1710 + 0.1 * (EARFCN - 19200);
            } else if (EARFCN >= 1200 && EARFCN <= 1949) {
                frequency = 1805 + 0.1 * (EARFCN - 1200);
            } else if (EARFCN >= 19950 && EARFCN <= 20399) { //fdd-band 4
                frequency = 1710 + 0.1 * (EARFCN - 19950);
            } else if (EARFCN >= 1950 && EARFCN <= 2399) {
                frequency = 2110 + 0.1 * (EARFCN - 1950);
            } else if (EARFCN >= 20400 && EARFCN <= 20649) { //fdd-band 5
                frequency = 824 + 0.1 * (EARFCN - 20400);
            } else if (EARFCN >= 2400 && EARFCN <= 2649) {
                frequency = 869 + 0.1 * (EARFCN - 2400);
            } else if (EARFCN >= 20650 && EARFCN <= 20749) { //fdd-band 6
                frequency = 830 + 0.1 * (EARFCN - 20650);
            } else if (EARFCN >= 2650 && EARFCN <= 2749) {
                frequency = 875 + 0.1 * (EARFCN - 2650);
            } else if (EARFCN >= 20750 && EARFCN <= 21449) { //fdd-band 7
                frequency = 2500 + 0.1 * (EARFCN - 20750);
            } else if (EARFCN >= 2750 && EARFCN <= 3449) { 
                frequency = 2620 + 0.1 * (EARFCN - 2750);
            } else if (EARFCN >= 3450 && EARFCN <= 3799) { //fdd-band 8
                frequency = 925 + 0.1 * (EARFCN - 3450);
            } else if (EARFCN >= 5010 && EARFCN <= 5179) { //fdd-band 12
                frequency = 729 + 0.1 * (EARFCN - 5010);
            } else if (EARFCN >= 5180 && EARFCN <= 5279) { //fdd-band 13
                frequency = 746 + 0.1 * (EARFCN - 5180);
            } else if (EARFCN >= 5730 && EARFCN <= 5849) { //fdd-band 17
                frequency = 734 + 0.1 * (EARFCN - 5730);
            } else if (EARFCN >= 6150 && EARFCN <= 6449) { //fdd-band 20     791-821 6150-6449  
                frequency = 791 + 0.1 * (EARFCN - 6150);
            }else if (EARFCN >= 9210 && EARFCN <= 9659) { //fdd-band 28       758-803 9210-9659
                frequency = 758 + 0.1 * (EARFCN - 9210);
            }else if(EARFCN >= 55240 && EARFCN <= 56740){
                frequency = 3550 + 0.1*(EARFCN - 55240);
            }else if(EARFCN >= 46790 && EARFCN <= 54539){
                frequency = 5150 + 0.1*(EARFCN - 46790);
            }else if(EARFCN >= 63000 && EARFCN <= 63999){
                frequency = 5150 + 0.1*(EARFCN - 63000);
            }else if(EARFCN >= 64000 && EARFCN <= 64999){
                frequency = 5725 + 0.1*(EARFCN - 64000);
            }else if(EARFCN >= 46790 && EARFCN <= 54539){
                frequency = 5150 + 0.1*(EARFCN - 46790);
            }else if(EARFCN >= 63000 && EARFCN <= 63999){
                frequency = 5150 + 0.1*(EARFCN - 63000);
            }else if(EARFCN >= 64000 && EARFCN <= 64999){
                frequency = 5725 + 0.1*(EARFCN - 64000);
            } else {
            //throw new Exception("Please Input the right EARFCN!");
            frequency = "--";
            }
            var showStr = EARFCN.toString()+"("+frequency.toString()+"MHZ"+")";
            
            return showStr;
        }

        function getEnbUnuseTimeData(rowData,divId) {
            arrayDate = [];
            arrayDay = [];
            arrayTime = [];

            enbvm.$refs.activeRatio.showSlide(function(){
                try{
                    echarts.getInstanceByDom(document.querySelector('#echart_activeRatio')).resize();
                }catch(e){}
            })
            
            var enb_code = rowData.small_cell_code;
            // 初始化x、y坐标数据
            yArr = [];
            xArr = [];
            
            var param = {
                timeZone:timeZone,
                enb_code:enb_code
            }
            $.post( "${ctx}/system/device/enb/unusetime/getUnuseTimeData.action", param, function(data){
                var availableDayDurationList = data ? (data.availableDayDurationList||[]) : [];
                var unavailableTimeRangeList = (data&&data.unavailableTimeRangList) ? data.unavailableTimeRangList : [];
                var availableDayDurationSort = availableDayDurationList.sort(compare("date"));
                var timeRangeSort = unavailableTimeRangeList.sort(compare("start_time"));
                $.each(availableDayDurationSort, function(index, item) {
                    xArr.push(item.date);
                    yArr.push(item.ailable_duration);
                });
                
                // 处理图表横轴数据，若数据跨月，将月的第一天前显示出月份，并粗体显示
                xArr = xArr.map(function(item) {
                    if (item.substring(8,10) == "01") {
                        return {
                            value: item.substring(5, 10).replace('-','.'),
                            textStyle: {
                                fontWeight: 'bold'
                            }
                        };
                    } else {
                        return {
                            value: item.substring(8, 10),
                            textStyle: {
                                fontWeight:'normal'
                            }
                        };
                    }
                });
                
                // 拼装不可用时间段表格的数据 
                $.each(timeRangeSort, function(index, item) {
                    var startTime = item.start_time;
                    var endTime = item.end_time;
                    var riQiStart = startTime.split(" ")[0];
                    var shiJianStart = startTime.split(" ")[1];
                    var riQiEnd = endTime.split(" ")[0];
                    var shiJianEnd = endTime.split(" ")[1];
                    if (riQiStart != riQiEnd ) {
                        shiJianEnd = "23:59:59";
                    }
                    var timeRangeStr = shiJianStart + "~" + shiJianEnd;
                    var obj = {};
                    obj.days = riQiStart;
                    obj.nousedTime = timeRangeStr;
                    arrayDate.push(obj);
                });
                
                // 创建图表 
                createEchartAvail(availableDayDurationSort);

                enbvm.activeslide.data = arrayDate;
            }, "json");
        }

        function compare(propertyName){
            return function(object1,object2){
                var value1=object1[propertyName];
                var value2=object2[propertyName];
                if(value2<value1) return 1;
                else if(value2>value1) return -1;
                else return 0;
            }
        }

        function createEchartAvail(availableDayDurationSort){
            var enodebChart = echarts.init(document.getElementById("echart_activeRatio"));
            var option = {
                tooltip: {
                    trigger: 'axis',
                    backgroundColor: 'rgba(205,224,232,0.9)',
                    textStyle: {
                        color: "#21608a"
                    },
                    formatter: function(params){
                        var str = "";
                        str += "<div><%=rb.getString("ZiDongBeiFenShiJian")%>:"+availableDayDurationSort[params[0].dataIndex].date+"<br/><%=rb.getString("KeYongFenZhongShu")%>:"+params[0].data+"</div>";
                        return str;
                    }
                },
                xAxis: [{
                    name: "<%=rb.getString("RiQi")%>",
                    type: "category",
                    data: xArr,
                    axisLabel: {
                        show: true,
                        interval: 0
                    },
                    boundaryGap: false
                }],
                yAxis: [{
                    name: "<%=rb.getString("FenZhong")%>",
                    type: "value",
                        axisTick: {
                            show: false
                        }
                }],
                series: [{
                    type: 'line',
                    itemStyle: {
                        normal: {
                            color: '#85B1DE'
                        }
                    },
                    data: yArr,
                    lineStyle: {
                        normal: {
                            color: '#85B1DE'
                        }
                    },
                    showAllSymbol:true
                }],
                grid: { x1: 0, x2: 150, y2: 40, y1: 40 }
            };
            enodebChart.setOption(option);
            // 窗体变化时自适应
            $(window).on('resize',function(){
                try {
                    setTimeout(function(){enodebChart.resize();},200);
                } catch (e) {}
            })
        }

        function gpsFormatter(value, row, rowIndex,field) {
            var isChanged = row.gps_modify_flag == 1,
                str = '',
                lat = row.gps_latitude||'',
                lng = row.gps_longitude||'',
                height = row.gps_height||'';
            
            if(isChanged) {
                value = row['modify_'+field]==undefined? row['gps_'+field]:row['modify_'+field];
            }
            if(value != undefined && isChanged) {
                str = '<div class="cellNameClass"  onclick="showGPSTip(this)" ><span  title="<%=rb.getString("GPSTongBuTipOne")%>&#10;<%=rb.getString("JingDu")%>: '+ lng +'&nbsp;&nbsp;<%=rb.getString("WeiDu")%>: '+lat+'&nbsp;&nbsp;<%=rb.getString("GaoDu")%>: '+height+'"  class="el-icon el-icon-circle-warning"></span></div><span style="margin-left:5px;">'+value+'</span>'
                        +'<div class="syncNameInfo"> <span class="panel_close" onclick="closeSyncName(this)"></span>'
                        +'<div><%=rb.getString("JingDu")%>: '+lng+'&nbsp;&nbsp;<%=rb.getString("WeiDu")%>: '+lat+'&nbsp;&nbsp;<%=rb.getString("GaoDu")%>: '+height+'</div>'
                        +'<div><%=rb.getString("TongBuGPSTiShi")%></span><div>'
                        +'<div><span class="button_simple" onclick="synchronizeGPS(&quot;'+row.small_cell_code+'&quot;)"><%=rb.getString("QueDing")%></span>' 
                        +'<span class="button_simple white" onclick="closeSyncName(this)"><%=rb.getString("QuXiao")%></span>'
                        +'</div></div>';
            }else {
                str = value;
            }
            return str;
        }

        function jumpToSetting(code,title,platform) {
            title += ' <span style="font-size: 12px;"></span>';
            enbPlatform = platform;

            $('#enbSetting_slide .slidebarTitleContainer .default').html(title).data('old',title);
            $('#enbSetting_slide_body').css({overflow: 'auto'}).html('');
            $('#setting_form_cnt').addClass('loading');
            $('.form_bt_refresh').hide();

            $('#enbSetting_slide').slideDown(500,function(){
                var postData = {smallCellCode: code};
                $('#enbSetting_slide').data('params',postData);
                $('#quick_setting_nav li:first').click();
            });
        }
        
        function settingTabClick(id,code,title,subTitle,platform){
            title = title + ' <span style="font-size: 12px;font-weight: normal;">'+subTitle+'</span>';
            enbPlatform = platform;
            $('#enbSetting_slide .slidebarTitleContainer .default').html(title).data('old',title);
            $('#enbSetting_slide_body').css({overflow: 'auto'}).html('');
            $('#setting_form_cnt').addClass('loading');
            $('.form_bt_refresh').hide();

            var postData = {id: id, smallCellCode: code};
            $('#enbSetting_slide').data('params',postData);
            $.ajax({
                url: '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                data: postData,
                type: 'post',
                dataType: 'json',
                success: function(data){
                    renderData2Dom(data);
                    setTimeout(function(){
                        $('.form_bt_refresh').show();
                    },500);
                },
                error: function(data){
                    $('#setting_form_cnt').removeClass('loading');
                }
            });
        }
        /**
        * 把参数组数据渲染成dom节点
        * @param data{array}: 参数组集合
        **/
        function renderData2Dom(data) {
            $('#enbSetting_slide_body').html('');
            Render.tableCollector = {};
            Render.gRender(data,document.querySelector('#enbSetting_slide_body'));
            setTimeout(function(){
                $('#enbSetting_slide_body .group-title:not(:first) .title-text').click();
                
                setTimeout(function(){
                    $('#setting_form_cnt').removeClass('loading');
                },500);
            },0);
        }
        /**
        * 刷新分组数据并渲染
        * @param param{array}: 分组数据的查询参数
        **/
        function refreshRenderData2Dom(param) {
            var slider = $('#enbSetting_slide'),
                postData = slider.data('params');
            
            if(slider.length == 0) return;
            // 重新请求分组数据渲染
            if(param.groupId == postData.id && param.smallCellCode == postData.smallCellCode) {
                slider.addClass('loading');
                $.ajax({
                    url: '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                    data: postData,
                    type: 'post',
                    dataType: 'json',
                    success: function(data){
                        renderData2Dom(data);
                        slider.removeClass('loading');
                    },
                    error: function(data){
                        slider.removeClass('loading');
                    }
                });
            }
        }
        /**
        * 关闭设置浮层面板
        * @param bool{boolean}: 是否检测数据变动
        **/
        function closeSettingPanel(bool){
            var addEdit = false
            var params = Render.getFormDatas($('#enbSetting_slide_body')),toClose=false;
            if(bool==true){
                if(addEdit){ // 保存后变为true 在点击关闭或者取消就会直接关闭
                    $('#enbSetting_slide').slideUp(500);
                }else{ // 没有点击保存 为false 
                    if(!isEmptyJson(params)) {
                    $.messager.confirm(TISHI,'<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',function(r){
                        if(r) $('#enbSetting_slide').slideUp();
                    }).addClass("seriousConfirm");
                    }else{
                        $('#enbSetting_slide').slideUp(500);
                    }
                }
            }else{
                $('#enbSetting_slide').slideUp(500);
            }
        }
        
        function goCellDetailParamInfoWin(divId,small_cell_code,status,CELL_IDENTITY) {
            var url = '${ctx}/cell/param/toCellDetailParamInfoPage.action?smallCellCode='+small_cell_code+'&connection_status=' + status;

            enbvm.infoslide.url = url;
            enbvm.$refs.info.showSlide({timeZone: timeZone}, function(){
                $("#tabAlarmCli").siblings("li").removeClass("active");  
                $("#tabAlarmCli").siblings("li").eq(0).addClass("active");
                var thisPar = $("#tabAlarmCli").closest('.panelDefault');
                var tabClass = $("#tabAlarmCli").siblings("li").eq(0).attr('tabtit');
                $("." + tabClass).show().siblings("div").hide();
                
                if(thisPar.find("div").hasClass('omcTabsPage_second')){
                    $("." + tabClass).find(".omcPageTitleContainer_second > li").first().click();
                }
                try{
                    setTimeout(function() {
                        window.dispatchEvent(new Event('resize'));
                    },200);
                }catch(e){}
            });

            return;
        }

        /**
        * 下发参数查询，刷新小站信息
        * @param cell_code{string}: 基站编码
        **/
        function refreshCell(cell_code) {
            var selCell = enbvm.selectedRow;

            if (!selCell) {
                $.messager.alert(TiShi, "Error.");
                return;
            }
            
            selCell.connection_status = 'updating';

            var params = {smallCellCode: cell_code};
            
            $.post( "${ctx}/cell/param/refreshCellInfo.action", params, function(data){
                if (data["success"]) {
                } else {
                    showMsg('error_msg',data["message"]);
                }
            }, "json");
        }

        function viewLicense(divId) {
            var selected = enbvm.selectedRow;
            var small_cell_code = selected.small_cell_code;
            var param ={
                    small_cell_code:small_cell_code
            }
            // 获取license信息
            $.post("${ctx}/cell/license/getHalobLicenseInfo.action", param, function(data) {
                if(!isEmptyObject(data)){
                    $('#serialNumberInput').val(data.serial_number);
                    $('#versionInput').val(data.version);
                    $('#timeInput').val(data.generate_date);
                    $('#modeInput').val(data.halob_mode);

                    var licenseData= [];
                    if(data.capacity_list){
                        $.each(data.capacity_list,function(index,item){
                            var obj={};
                            obj.Capacity = item.id;
                            obj.ValidPeriod = item.valid_period;
                            var remperiod = item.remaining_period;
                            if (item.valid_period =="0"){
                                obj.RemainingPeriod = "<%=rb.getString("Yongjiu")%>";
                            }
                            else{
                                obj.RemainingPeriod = remperiod;
                            }
                            obj.quantity = item.capa_value;
                            obj.Description = item.description;
                            licenseData.push(obj);
                        })
                    }

                    $("#license_table").datagrid({
                            border:false,
                            fit: true,
                            fitColumns: true,
                            singleSelect:true,
                            striped:true,
                            rownumbers:false,
                            pagination:false,
                            columns:[[
                                { field:'Capacity',title:'<%=rb.getString("TeXingID")%>',width:50},   
                                { field:'Description',title:'<%=rb.getString("MiaoShu")%>',width:120},
                                { field:'quantity',title:'<%=rb.getString("ShuLiang")%>',width:50},   
                                { field:'ValidPeriod',title:'<%=rb.getString("YouXiaoQi")%>',width:60},
                                { field:'RemainingPeriod',title:'<%=rb.getString("ShengYuShiJian")%>',width:60 },
                                            
                            ]],
                            data:licenseData
                        });
                }else{
                    $("#license_table").datagrid({
                            border:false,
                            fit: true,
                            fitColumns: true,
                            singleSelect:true,
                            striped:true,
                            rownumbers:false,
                            pagination:false,
                            columns:[[
                                { field:'Capacity',title:'<%=rb.getString("TeXingID")%>',width:50},  
                                { field:'Description',title:'<%=rb.getString("MiaoShu")%>',width:50},
                                { field:'ValidPeriod',title:'<%=rb.getString("YouXiaoQi")%>',width:100},
                                            
                            ]],
                            data:[]
                        });
                }
            }, "json");
        }

        function ipAddrFormatter(value,row,index) {
            if(value) {
                value = '<a href="https://' + value + '" target="_blank" style="color: #4d84ff;">'+value+'</a>';
            }

            return value;
        }

        function cellStateFormatter(value, rowData, rowIndex){
            if (value == null) {
                return null;
            }
            else if (value == "1") {
                val = "<%= rb.getString("JiHuo")%>";
                value = "<div class='activeStatusItem'><span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span>"+(val)+"</div>"
            }
            else if (value == "0") {
                //状态不一样展示的文字也不一样
                val = "<%= rb.getString("QuJiHuo")%>";
                value = "<div class='inactiveStatusItem'><span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span>"+(val)+"</div>"
            }
            return value;
        }
        /**
        * 判断对象是否为空
        * @param obj{object}: 要判断的对象
        **/
        function isEmptyObject(obj){
            for(var key in obj){
                return false;
            }
            return true;
        }

        function halobStatusFormatter (value, rowData, rowIndex) {
            if ("1" == value) {
                value = "<span class='el-icon el-icon-status-enable' style='margin-right:10px;font-size: 20px;'></span>On";
            } else if ("0" == value) {
                value = "<span class='el-icon el-icon-status-disable' style='margin-right:10px;font-size: 20px;'></span>Off";
            } else {
                value = "--";
            }
            return value;
        }

        function kpiStatusFormatter(value,row,index){
            if(value == null){
                return null;
            }else if(value == "off"){
                value = "<span class='el-icon el-icon-status-kpi-off' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"		
            }else if(value == "normal"){
                value = "<span class='el-icon el-icon-status-kpi-normal' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"
            }else if(value == "broken"){
                value = "<span class='el-icon el-icon-status-kpi-failure' style='float:left;font-size:20px;margin-right:10px'></span>"+"<span style='color:#444;margin-top:4px;display:block;float:left'>"+(value)+"</span>"
            }
            return value;
        }

        function syncStatusFormatter(value, rowData, rowIndex){
            if (value == null) {
                return null;
            } else if (value == "--" ){
                return "--";
            } else if (value == ("GPS " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                val ="GPS "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == ("1588 " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                val ="1588 "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == ("REM " +"<%= rb.getString("ZhengZaiTongBu")%>")) {
                val ="REM "+ "<%= rb.getString("ZhengZaiTongBu")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-syning.gif'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == "GPS "+ "<%= rb.getString("TongBuChengGong")%>" ) {
                val ="GPS "+ "<%= rb.getString("TongBuChengGong")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == "1588 "+"<%= rb.getString("TongBuChengGong")%>" ) {	
                val = "<%= rb.getString("TongBuChengGong")%>";
                val = "1588 " + val;
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == "REM "+"<%= rb.getString("TongBuChengGong")%>") {	
                val = "<%= rb.getString("TongBuChengGong")%>";
                val = "REM " + val;
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/tnsuccessomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }else if (value == "<%= rb.getString("WeiTongBu")%>") {	
                val = "<%= rb.getString("WeiTongBu")%>";
                value = "<img style='margin:0px 10px 0px 0;float:left;width: 18px;' src ='${ctx}/css/images/main/monitor-ico/monitor-nosynomc.png'/>"+"<span style='color:#444;display:block;float:left'>"+(val)+"</span>"
            }
            return '<div style="min-width: 130px;diplay: line-block;">'+value+'</div>';
        }

        function lockStatusFmt(value,rowData,rowIndex){
            if(value == "--"){
                return value;
            }else{
                if(value == 0){
                    return "<div><span class='el-icon el-icon-status-unlock' style='margin-right:10px;font-size:20px;'></span><%=rb.getString("YouXiaoQiJieSuo")%></div>";
                }else{
                    return "<div><span class='el-icon el-icon-operation-lock' style='margin-right:10px;font-size:20px;'></span><%=rb.getString("YouXiaoQiSuoDing")%></div>";
                }
            }
        }

        function confirmImmediateCollectLogFile(divId) {
            var selCell = enbvm.selectedRow,
                cellCode = selCell.small_cell_code,
                serial_number = selCell.serial_number,
                param = {
                    start_time: 'undefined',
                    end_time: 'undefined',
                    execute_type: 'Immediately',
                    reportPeriod: '',
                    isReboot: 'false',
                    serial_number: serial_number, 				    				
                    timeZone: timeZone, 
                    device_type: "eNB",
                    device_code: cellCode
                };
            
            $.post("${ctx}/cell/collect/goImmediateCollectLogFile.action", param, function (data) {
                if (data["success"]) {
                    showMsg('success_msg','<%=rb.getString("RiZhiZhengZaiShouJi")%>')
                } else {
                    showMsg('error_msg',data['message']);
                }
            }, "json");
        }

        // 发起SAS注册
        function registerSas() {
            var selCell = enbvm.selectedRow;

            $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingFaQiSASZhuCe")%>" , function(r) {
                if (r) {
                    var params = {
                        cellCode: selCell["small_cell_code"],
                        serialNumber: selCell["serial_number"]
                    }

                    $.post("${ctx}/cell/SAS/registerCellForSas.action", params, function(data){
                        if (!data["success"]) {
                            showMsg('error_msg',data["message"]);
                        }
                    }, "json");
                }
            });
        }
        // 取消sas注册
        function deregisterSAS(){
            var selCell = enbvm.selectedRow;
            var params = {
                "sn": selCell["serial_number"],
                "enbCode": selCell["small_cell_code"]
            }
            
            $.messager.confirm("<%=rb.getString("QueRen")%>", "Are you sure to deregister?" , function(r) {
                if (r) {
                    $.post("${ctx}/cell/SAS/deregister.action", params, function(data){
                        if (!data["success"]) {
                            showMsg('error_msg',data["message"]);
                        }
                    }, "json");
                }
            });
        }

        function cellReboot(cell_code, product) {
            var selCell = enbvm.selectedRow;

            $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiSheBei")%>", function (r) {
                if (r) {
                    showMsg('prompt_msg',"<%=rb.getString("MingLingYiXiaFa")%>")
                    var param = {
                        cell_code: cell_code
                    };
                    $.post("${ctx}/cell/cpeinfos/cellReboot.action", param, function (data) {
                        if (!data["success"]) {
                            showMsg('error_msg', data["message"]);
                        }
                    }, "json");
                }
            }).addClass("seriousConfirm");
        }

        function openCloseHalob(cellCode,halobSwitch){
            var params = {};
            params.cell_code = cellCode;
            params.halob_switch = halobSwitch;
            $.messager.confirm({
                title: '<%=rb.getString("TiShi")%>',
                msg: '<%=rb.getString("CanShuXiuGaiXuYaoChongQiJiZhan")%>',
                fn:function(r){
                    if(r){
                        $.post("${ctx}/cell/cpeinfos/setCellHalobSwitch.action",params,function(data){
                            if(data["success"]){
                                /* $('#tableHomeCellList').datagrid('reload'); */
                            }
                        },"json")
                    }
                }
            }).addClass("seriousConfirm");
            
        }

        function activeOpStatus(active,smallCellCode){
            var params = {};
            params.op_state = active;
            params.small_cell_code = smallCellCode;
            $.post("${ctx}/cell/cpeinfos/cellModifyActiveStatus.action",params,function(data){
                if(data["success"]){
                    enbvm.refreshList();
                    showMsg('success_msg','<%=rb.getString("XiaFaChengGong")%>');
                }else{
                    showMsg('error_msg',data["message"]);
                }
            },"json")
        }

        function configReset(cellcode){
            $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingHuiFuMoRenPeiZhi")%>", function (r) {
                if (r) {
                    var param = {
                            cellCode: cellcode
                    };
                    $.post("${ctx}/cell/cpeinfos/cellFactoryReset.action", param, function (data) {
                        
                    }, "json");
                }
            }).addClass("seriousConfirm");
        }
        //设置基站有效期
        function setEffectPeriod() {
            enbvm.goPeriod();
        }

        function setLockStatus(smallcellCode, lockStatus){
            var params = {
                    smallCellCode : smallcellCode
            }
            if(lockStatus == 0){
                params.lockStatus = 'lock';
            }else{
                params.lockStatus = 'unlock';
            }
            $.post("${ctx}/cell/cpeinfos/operationLockStatus.action",params,function(data){
                if(data["success"]){
                    enbvm.refreshList();
                }else{
                    showMsg('error_msg',data["message"]);
                }
            },"json")
        }

        function setForceRFStatus(code, status){
            $.ajax({
                url: '${ctx}/cell/SAS/autoOrForceRF.action',
                data: {serialNumber: code, autoOrForceRF: status},
                type: 'post',
                dataType: 'json',
                success: function(data){
                    if(data.success){
                        enbvm.refreshList();
                    }
                    toast(data.message,$('#mainpage'),'success');
                },
                error: function(data){
                    toast('<%=rb.getString("SheZhiShiBai")%>',$('#mainpage'));
                }
            });
        }

        function distributeSn(divId,small_cell_code){
            enbvm.goDistribute(small_cell_code)
        }

        function goCellDetailAlarmInfoWin(status,smallCellCode) {
            goCellDetailParamInfoWin('',smallCellCode,status,'');
        }
        
        function setRFStatus(code,status,cell){
    		var rfStatus = status;
    		// 依据状态设置开、关
    		if(status == 'on') rfStatus = 'off';
    		if(status == 'off') rfStatus = 'on';

    		cell = cell || '';
    		$.ajax({
    			url: '${ctx}/cell/cpeinfos/cellModifyRadioStatus.action',
    			data: {smallCellCode: code, radioStatus: rfStatus,cellNumber:cell},
    			type: 'post',
    			dataType: 'json',
    			success: function(data){
    				if(data.success){
    					enbvm.refreshList();
    				}
    				toast('<%=rb.getString("XiaFaChengGong")%>',$('#mainpage'),'success');
    			},
    			error: function(data){
    				toast('<%=rb.getString("SheZhiShiBai")%>',$('#mainpage'));
    			}
    		});
    	}
      	//刷新基站监控 下 统计信息，填充状态栏
        function refresh_cellStatusStatistics(cb) {
            var params = {},dualStatus = '';
            
            $.extend(params, enbvm.queryParams);

            params["switch_status"] = dualStatus;
            params["isDual"] = true;
            params["isMonitor"] = true;
            
            $.post("${ctx}/cell/cpeinfos/getCellStatusStatistics.action", params, function(data) {
                if(!data["connection_status"]){
                    connectionStatus = "0/0";
                    connectionStatusRef = "0/0"
                }else{
                    connectionStatus = data["connection_status"];
                    connectionStatusRef = data["connection_status_ref"];
                    if($("#onlineStatusNum").prev().hasClass('greenType')){
                        $("#cellInfo #onlineStatusNum").text(data["connection_status"]);
                    }else if($("#onlineStatusNum").prev().hasClass('redType')){
                        $("#cellInfo #onlineStatusNum").text(data["connection_status_ref"]);
                    }
                }
                $('#online_count_rate').text("( "+connectionStatus+" )");
                if(!data["mme_status"]){
                    mmeStatus = "0/0";
                    mmeStatusRef = "0/0";
                    $("#cellInfo #mmeStatusNum").text("0/0");
                }else{
                    mmeStatus = data["mme_status"];
                    mmeStatusRef = data["mmeStatus_ref"];
                    if($("#mmeStatusNum").prev().hasClass('greenType')){
                        $("#cellInfo #mmeStatusNum").text(data["mme_status"]);
                    }else if($("#mmeStatusNum").prev().hasClass('redType')){
                        $("#cellInfo #mmeStatusNum").text(data["mmeStatus_ref"]);
                    }
                }
                $('#mme_count_rate').text("( "+mmeStatus+" )");
                if(!data["op_state"]){
                    opStateStatus = "0/0";
                    opStateStatusRef = "0/0";
                    $("#cellInfo #activeStatusNum").text("0/0");
                }else{
                    opStateStatus = data["op_state"];
                    opStateStatusRef = data["opState_ref"];
                    
                    if($("#activeStatusNum").prev().hasClass('greenType')){
                        $("#cellInfo #activeStatusNum").text(data["op_state"]);
                    }else if($("#activeStatusNum").prev().hasClass('redType')){
                        $("#cellInfo #activeStatusNum").text(data["opState_ref"]);
                    }
                }
                $('#active_count_rate').text("( "+opStateStatus+" )");
                if(cb && typeof cb == 'function') {
                    cb();
                }
            }, "json");
        }
    </script>
</body>
</html>