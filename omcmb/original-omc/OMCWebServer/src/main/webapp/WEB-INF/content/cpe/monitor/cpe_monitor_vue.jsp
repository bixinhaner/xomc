<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<!DOCTYPE html>
<html>
<head>
    <script type="text/javascript" src="${ctx}/js/element/Sortable.min.js?_=${omc_ver}"></script>
    <script type="text/javascript" src="${ctx}/js/element/vuedraggable.umd.min.js?_=${omc_ver}"></script>
    <style>
		.fixed-right-msg {
			display: flex;
			top: -32px;
			right: 100px;
			position: absolute;
            border: 1px solid #DFE2EE;
		    border-radius: 5px;
		}
        .fixed-right-msg .foldBtnCls{
            height: 26px;
            width: 26px;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
			background-color: #fff;
            cursor: pointer;
            transform: rotate(90deg);
        }
        .collect-bt {
            color:#1DA3FC;
            margin-left: 15px;
            text-decoration: underline;
            cursor: pointer;
        }

        .collect-select .el-input {
            width: 120px;
        }
        .list-group > span {
            display: flex;
            flex-direction: column;
            padding-left: 15px; 
            height: 400px;
            overflow: auto;
        }
        .list-group-item {
            display: inline-block;
            position: relative;
            padding: 0px 10px;
            margin: 3px 5px;
            width: 200px;
            border: 0px dashed #ddd;
            cursor: move;
        }
        .no-padding .slide-content {
            padding: 0px;
        }
        .no-border .el-card__body {
            border: none;
        }
        .showHideItem {
            position:absolute;
            width: 950px;
            background:white;
            z-index:888;
            top: 86px;
            left: 0px;
            display:none;
            box-shadow:5px 10px 23px 0px rgba(201,212,231,0.50);
        }
        .showHideItem input{
            margin-top:-2px;
            margin-bottom:1px;
            vertical-align:middle;
            margin-right:20px;
        }
        .col-group {
            margin: 5px 10px;
            display: flex;
            flex-wrap: wrap;
        }
        .col-group .el-checkbox {
            min-width: 170px;
        }
        .col-group .el-checkbox__label {
            font-size: 12px;
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
            margin-left: 10px;
        }
        .showHideItem .el-icon-close1::before {
            color: #333;
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
            text-overflow:ellipsis;
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
        .enbMonitorForm .el-select__input{
            height:24px;
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

        td[field=module_version], td[field=SOFTWARE_VERSION] {
            position: relative;
        }
        .version-details {
            display: none;
            position: absolute;
            right: 10px;
            padding: 20px 20px 10px 15px;
            box-shadow: 2px 2px 5px silver;
            background: #fff;
            border-radius: 3px;
            z-index: 100;
        }
        .version-details.normal {
            display: block;
            position: relative;
            box-shadow: none;
            padding: 0 5px;
        }
        .version-details .item {
            display: block;
            width: auto;
            min-width: 100px;
            padding: 2px 0;
            border-bottom: 1px dashed silver;
        }
        .version-details .panel_close {
            margin-top: -15px;
        }

        .no-padding .slide-content {
            padding: 0px;
        }
        .no-border .el-card__body {
            border: none;
        }
        .flex-body .slide-content {
            display: flex;
            flex-direction: column;
        }

        #cpeInfo .slide-position-top > .el-card > .el-card__header .clearfix span .el-icon-close{
			display:none !important;
		}
		.bold-title {
            display: flex;
            align-items: center;
        }
        .bold-title .el-form-item__label {
            font-weight: bold;
        }
        .confirmCard .el-dialog__body{
			padding:30px !important;
			background:#FFFFFF;
			border:none;
		}
		.confirmCard .el-dialog__body .el-checkbox__input{
			margin-top:-5px;
		}
		.confirmCard .el-dialog__body .el-checkbox__label{
			line-height:28px;
		}

		.commonLeft6 { margin-left: 6px; }
		.commonDialog{ border-radius: 10px; }
		.leftLabel .el-form-item__label{
			width: 90px;
			padding-top: 8px;
		}
		.tipText{
            padding-bottom:20px;
            display:flex;           
            color: rgba(0, 0, 0, 0.32);		
            font-size:12px;
        }
        .tipText .infoTip { margin-top: 2px; }
        .tipText .infoTip:before{
            color: rgba(0, 0, 0, 0.32);		
            font-size:14px;
            margin-right:6px;
        }
        .el-upload__tip{
            margin: -4px 0;
        }
        .uploadFormat{
            margin-left:10px;
            color: rgba(0, 0, 0, 0.32);		
            font-size:12px;
        }
        .importFileItem .el-input__suffix{
            top:4px;
        }
        #cpeInfo .el-ctable-toolbar{
            padding: 0px!important;
        }
        .settingSlide {
	    	width:70% !important;
	    	min-width:800px;
	    	left:auto;
	    }
        .showHideItem #sortAndShowColumnBoxCls{
            height: 510px;
        }
        #showHideItemCpe #sortAndShowColumnBoxCls .el-checkbox__input.is-disabled.is-checked .el-checkbox__inner{
		    opacity: 0.3!important;
        }
        #showHideItemCpe #sortAndShowColumnBoxCls .list-group > span{
            padding-left: 0px;
            height: calc(100% - 52px);
            overflow-x: hidden;
        }
        #showHideItemCpe #sortAndShowColumnBoxCls .list-group .el-checkbox__label{
            padding-left: 0px;
        }
        #showHideItemCpe #sortAndShowColumnBoxCls .list-group .list-group-item {
            width: 220px;
        }
        .width-fix-auto {
            width: 100% !important;
        }
    </style>
</head>
<body>
    <div id="cpeInfo" style="overflow: auto;height: 100%; " class='commonWarp'>
       
        <el-ctable ref="list" id="tableHomeCpeList" style="min-width: 900px;"
            :url="tbURL"
            :limit="limitBatch"
            :query-params="queryParams"
            row-key="CPE_CODE"
            @load-success="loadSuccess"
            @selection-change="selectChange">
            <template slot="toolbar">
                <div class='toolbarHeadBtnBoxCls'>
                    <div v-show="activeName=='table'" class="newIconBoxCls-bt CODE_CPE_DEVICE hidden" @click="addDevice" style="right:56px;top:5px;" tip="<%=rb.getString("TianJia")%>">
                        <span class="el-icon-plus el-icon"></span>
                    </div>	
                    <div id="enb_export_op" @click="showExport" v-show="activeName=='table'">
                        <el-popover trigger="click" placement="bottom-end" popper-class="monitorBtnPopperCls">
                            <div class="el-card__header">
                                    <%=rb.getString("DaoChu")%>
                                    <span style="color: 999;font-weight: normal;margin-left: 5px;">(<%=rb.getString("SuoYouCanShu")%>)</span>
                                    <span style="font-size: 14px;" class="el-icon el-icon-close" onclick="document.body.click();"></span>
                            </div>
                            <div class="export-content" style="width: 920px; max-height: 500px;overflow: auto;"></div>
                            <div slot="reference" class="newIconBoxCls-bt" style="right:20px;top:5px;" tip="<%=rb.getString("DaoChu")%>">
                                <span class="el-icon-operation-export el-icon"></span>
                            </div>	
                        </el-popover>
                    </div>
                    <!-- 已选数据 -->
                    <div class="selectBlukBoxCls">
                        <div class="selectMain">
                            <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                <span class="el-icon-selected el-icon"></span>
                                <span class="bulkSelectNumBoxCls">( {{selectedRows.length}} )</span>
                            </div>
                            <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                                <div class="selectBoxTitle">
                                    <span><%=rb.getString("YiXuan")%></span>
                                    <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                </div>
                                <div class="selectBoxMain">
                                    <div class="tableInfoCls">
                                        <div class="tableInfoHeader">
                                            <div><%=rb.getString("CPEXuLieHao")%></div>
                                            <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
                                        </div>
                                        <el-ctable 
                                            id="bulkSelectTable" 
                                            ref="bulkSelectTable" 
                                            :data="selectedRows" 
                                            :showHeader="false"
                                            :rownumber="false"
                                            :front-pagination="true"
                                            height="300px" pagination="true" >
                                            <el-table-column prop="id" v-if="false"></el-table-column>
                                            <el-table-column width="588">
                                                <template slot-scope="scope" >
                                                    <div class="tableItemCls">
                                                        <span>{{scope.row.SERIAL_NUMBER}}</span>
                                                        <span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
                                                    </div>
                                                </template>
                                            </el-table-column>
                                        </el-ctable>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div v-if="hasRebootRole" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="rebootCpeList">
                        <span class='el-icon el-icon-operation-reboot'></span>
                        <span><%=rb.getString("ChongQi")%></span>
                    </div>
                    <div v-if="hasModifyPwdRole" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="openModifyPwd">
                        <span class='el-icon el-icon-operation-resetPassword'></span>
                        <span><%=rb.getString("XiuGaiMiMa")%></span>
                    </div>
                    <div v-if="turboEnable && isMonitorWritable" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" onclick="ltmkai('1')">
                        <span class='el-icon el-icon-operation-enable1'></span>
                        <span>LTE-TURBO  <%=rb.getString("QiYong")%></span>
                    </div>
                    <div v-if="turboEnable && isMonitorWritable" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" onclick="ltmkai('0')">
                        <span class='el-icon el-icon-operation-disable1'></span>
                        <span>LTE-TURBO <%=rb.getString("JinYong")%></span>
                    </div>
                    <div v-if="isMonitorWritable" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="recycleCells">
                        <span class="el-icon el-a-icon-Recyclebin"></span>
                        <span><%=rb.getString("HuiShouZhan")%></span>
                    </div>
                </div>
                
                <div id="toolbar_tableHomeCpeList" class="toolbarContainer" style="position:relative;padding: 0px 10px !important;"></div>
            </template>

            <el-table-column v-if="isMonitorWritable" type="selection" :reserve-selection="true" fixed></el-table-column>
            <el-table-column prop="cpe_operation" key="cpe_operation" width="70" fixed>
                <template slot-scope="scope">
                	<div class="el-icon el-icon-circle-setting1" @click="openSettingPage(scope.row)" style="font-size:19px;"></div>
                    <div v-if="isMonitorWritable" class="el-icon el-icon-operation-more-circle" @click="optClick(scope.row, event)" v-clickoutside="handerClose"></div>
                </template>
            </el-table-column>
            <el-table-column prop="CONNECTION_STATUS" key="CONNECTION_STATUS" width="40" sortable fixed>
                <template slot-scope="scope">
                    <div v-html="connStatusFmt(scope.row.CONNECTION_STATUS, scope.row, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column sortable label="<%=rb.getString("CPEXuLieHao")%>" prop="SERIAL_NUMBER" key="SERIAL_NUMBER" width="180" fixed></el-table-column>
            <el-table-column sortable label="<%=rb.getString("CPEName")%>" prop="CPE_NAME" key="CPE_NAME" width="120" fixed>
                <template slot-scope="scope">
                    <div v-if="scope.row.device_name_tip === '0'">{{scope.row.CPE_NAME}}</div>
                    <div v-if="scope.row.device_name_tip !== '0'" class="cellNameClass">
                        <el-popover trigger="click">
                            <div slot="reference">
                                <span class="el-icon el-icon-circle-warning"></span> 
                                {{scope.row.CPE_NAME}}
                            </div>
                            <div style='padding: 30px 20px 20px; position:relative;'>
                                <div> <span class="el-icon el-icon-close" @click="closeSyncName" style="top: 10px; position:absolute;"></span>
                                <div style='display: flex;'><span style='color: #333; font-size: 14px;'><%=rb.getString("JiZhanCeMingCheng")%> : </span> <span style='font-size: 14px;margin-left: 6px;color: #333;'>{{scope.row.NICK_NAME}}</span></div>
                                <div style='font-size: 12px; color: #333; margin-top: 6px;'><%=rb.getString("ShiFouTongBuMingChengDaoOMC")%><div>
                                <div style="margin-top: 10px; text-align: right;">
                                    <span class="el-button--primary el-button" @click="syncName(scope.row.CPE_CODE, scope.row.NICK_NAME)">
                                        <%=rb.getString("QueDing")%>
                                    </span>
                                    <span class="white el-button" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
                                </div>
                            </div>
                        </el-popover>
                    </div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('IMSI')" sortable label="IMSI" prop="IMSI" key="IMSI" width="150"></el-table-column>
            <el-table-column v-if="showColums.includes('MACADDRESS')" sortable label="MAC" prop="MACADDRESS" key="MACADDRESS" width="150"></el-table-column>
            <el-table-column v-if="showColums.includes('IPADDRESS')" sortable label="IP" prop="IPADDRESS" key="IPADDRESS" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('cpe_model')" label='<%=rb.getString("SheBeiXingHao")%>' prop="cpe_model" key="cpe_model"  min-width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('MODEL_NAME')" sortable label="<%=rb.getString("ChanPinXingHao")%>" prop="MODEL_NAME" key="MODEL_NAME" width="160">
                <template slot-scope="scope">
                    <div v-html="capablityFmt(scope.row.MODEL_NAME, scope.row, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('SOFTWARE_VERSION')" sortable label="<%=rb.getString("CPEVersion")%>" prop="SOFTWARE_VERSION" key="SOFTWARE_VERSION" width="160">
                <template slot-scope="scope">
                    <div v-html="softwareVersionFmt(scope.row.SOFTWARE_VERSION, scope.row, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('group_name')" sortable label="<%=rb.getString("SheBeiZu")%>" prop="group_name" key="group_name" width="150"></el-table-column>
            <el-table-column v-if="showColums.includes('HOST_NAME')" sortable label="<%=rb.getString("HostName")%>" prop="HOST_NAME" key="HOST_NAME" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('CELL_IDENTITY')" sortable label="ECI" prop="CELL_IDENTITY" key="CELL_IDENTITY" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('SCANMODE')" label="<%=rb.getString("SaoMiaoFangShi")%>" prop="SCANMODE" key="SCANMODE" width="150">
                <template slot-scope="scope">
                    <div v-if="scope.row.SCANMODE == 'fullband'"><%= rb.getString("BuSuoPin")%></div>
                    <div v-if="scope.row.SCANMODE == 'pcilock'"><%= rb.getString("SuoPCI")%></div>
                    <div v-if="scope.row.SCANMODE == 'freqpreferred'"><%= rb.getString("SuoPin")%></div>
                    <div v-if="scope.row.SCANMODE == 'pcionlylock'">PCI only lock</div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('PCI')" sortable label="PCI" prop="PCI" key="PCI" width="80">
                <template slot-scope="scope">
                    <div v-html="pciStatusFmt(scope.row.PCI, scope.row, scope.$index)" style="display: flex;"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('LGW_IP')" label="<%=rb.getString("LgwIPAddress")%>" prop="LGW_IP" key="LGW_IP" width="130">
                <template slot-scope="scope">
                    <div v-if="scope.row.IPADDRESS">
                        <a :href="scope.row.lgw_url" target="_blank" style="color: #4d84ff;">{{scope.row.LGW_IP}}</a>
                    </div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('LGW_MAC')" sortable label="<%=rb.getString("LgwMacAddress")%>" prop="LGW_MAC" key="LGW_MAC" width="170"></el-table-column>
            <el-table-column v-if="showColums.includes('UL_MCS')" sortable label="UL_MCS" prop="UL_MCS" key="UL_MCS" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('DL_MCS')" sortable label="DL_MCS" prop="DL_MCS" key="DL_MCS" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('lte_connection_time')" sortable label="<%=rb.getString("LTEGengXinShiJian")%>" prop="lte_connection_time" key="lte_connection_time" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('DL_BLER')" sortable label="DL BLER" prop="DL_BLER" key="DL_BLER" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('RSRP0')" sortable label="RSRP1" prop="RSRP0" key="RSRP0" width="80">
                <template slot-scope="scope">
                    <div v-html="redAccordingRSRPFmt(scope.row.RSRP0, scope.row, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('RSRP1')" sortable label="RSRP2" prop="RSRP1" key="RSRP1" width="80">
                <template slot-scope="scope">
                    <div v-html="redAccordingRSRPFmt(scope.row.RSRP1, scope.row, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('CINR0')" sortable label="CINR1" prop="CINR0" key="CINR0" width="80"></el-table-column>
            <el-table-column v-if="showColums.includes('CINR1')" sortable label="CINR2" prop="CINR1" key="CINR1" width="80"></el-table-column>
            <el-table-column v-if="showColums.includes('CPE_SINR')" sortable label="SINR" prop="CPE_SINR" prop="key" width="80"></el-table-column>
            <el-table-column v-if="showColums.includes('DL_CURRENT_DATARATE')" sortable label="<%=rb.getString("CPEXiaXingTunTuLiang")%> (Mbps)" prop="DL_CURRENT_DATARATE" key="DL_CURRENT_DATARATE" width="180"></el-table-column>
            <el-table-column v-if="showColums.includes('UL_CURRENT_DATARATE')" sortable label="<%=rb.getString("CPEShangXingTunTuLiang")%> (Mbps)" prop="UL_CURRENT_DATARATE" key="UL_CURRENT_DATARATE" width="180"></el-table-column>
            <el-table-column v-if="showColums.includes('UPTIME')" sortable label="<%=rb.getString("YunXingShiJian")%>" prop="UPTIME" key="UPTIME" width="140"></el-table-column>
            <el-table-column v-if="showColums.includes('first_online_time')" sortable label="<%=rb.getString("DiYiCiLianJieShiJian")%>" prop="first_online_time" key="first_online_time" width="140"></el-table-column>
            <el-table-column v-if="showColums.includes('LASTINFORMTIME')" sortable label="<%=rb.getString("ShangCiLianJieShiJian")%>" prop="LASTINFORMTIME" key="LASTINFORMTIME" width="140"></el-table-column>
            <el-table-column v-if="showColums.includes('PRODUCT')" sortable label="<%=rb.getString("CPELeiXing")%>" prop="PRODUCT" key="PRODUCT" width="120">
                <template slot-scope="scope">
                    <div v-html="CPETypeFmt(scope.row.PRODUCT, scope.row, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('TX_POWER')" sortable label="<%=rb.getString("CPETxPower")%>" prop="TX_POWER" key="TX_POWER" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('DL_EARFCN')" label="<%=rb.getString("PinDian")%>" prop="DL_EARFCN" key="DL_EARFCN" width="60"></el-table-column>
            <el-table-column v-if="showColums.includes('BANDWIDTH')" label="<%=rb.getString("DaiKuan")%>(MHz)" prop="BANDWIDTH" key="BANDWIDTH" width="130"></el-table-column>
            <el-table-column v-if="showColums.includes('MCC')" sortable label="MCC" prop="MCC" key="MCC" width="80"></el-table-column>
            <el-table-column v-if="showColums.includes('MNC')" sortable label="MNC" prop="MNC" key="MNC" width="80"></el-table-column>
            <el-table-column v-if="showColums.includes('longitude')" label="<%=rb.getString("JingDu")%>" prop="longitude" key="longitude" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('latitude')" label="<%=rb.getString("WeiDu")%>" prop="latitude" key="latitude" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('height')" label="<%=rb.getString("GaoDu")%>" prop="height" key="height" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('distance')" label="<%=rb.getString("JuLi")%>" prop="distance" key="distance" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('module_name')" label="<%=rb.getString("MoKuaiMingCheng")%>" prop="module_name" key="module_name" width="150"></el-table-column>
            <el-table-column v-if="showColums.includes('module_version')" label="<%=rb.getString("MoKuaiBanNen")%>" prop="module_version" key="module_version" width="150">
                <template slot-scope="scope">
                    <div v-html="moduleVersionFmt(scope.row.module_version, scope.row, scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column v-if="showColums.includes('MARKET_NAME')" label="<%=rb.getString("ChanPinMingCheng")%>" prop="MARKET_NAME" key="MARKET_NAME" width="150"></el-table-column>
            <el-table-column v-if="showColums.includes('IMEI')" label="IMEI" prop="IMEI" key="IMEI" width="100" sortable></el-table-column>
            
            <el-table-column v-if="showColums.includes('NR_BAND')" label="NR-Band" prop="NR_BAND" key="NR_BAND" width="80"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_BANDWIDTH')" label="NR-<%=rb.getString("DaiKuan")%>(MHz)" prop="NR_BANDWIDTH" key="NR_BANDWIDTH" width="160"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_PCI')" label="NR-PCI" prop="NR_PCI" key="NR_PCI" width="80"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_EARFCN')" label="NR-<%=rb.getString("PinDian")%>" prop="NR_EARFCN" key="NR_EARFCN" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_PLMN')" label="NR-PLMN" prop="NR_PLMN" key="NR_PLMN" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_CELL_ID')" label="NR-Cell ID" prop="NR_CELL_ID" key="NR_CELL_ID" width="120"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_DL_FREQUENCY')" label="NR-DL Frequency" prop="NR_DL_FREQUENCY" key="NR_DL_FREQUENCY" width="150"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_UL_FREQUENCY')" label="NR-UL Frequency" prop="NR_UL_FREQUENCY" key="NR_UL_FREQUENCY" width="150"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_CINR')" label="NR-CINR" prop="NR_CINR" key="NR_CINR" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_SINR')" label="NR-SINR" prop="NR_SINR" key="NR_SINR" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_RSRQ')" label="NR-RSRQ" prop="NR_RSRQ" key="NR_RSRQ" width="90"></el-table-column>
            <el-table-column v-if="showColums.includes('NR_RSRP')" label="NR-RSRP" prop="NR_RSRP" key="NR_RSRP" width="90"></el-table-column>
        </el-ctable>
        <el-cmenu ref="menu" :data="menus" @click="menuClick"></el-cmenu>

		<el-slide ref="cpeSettingPage" class="settingSlide" 
           	:url="settingUrl" width="80%"
            :footer="false" 
            :header="false">
        </el-slide>
		
        <!-- sliders -->
        <el-slide ref="info" id="cpeInformation" class="no-padding flex-body slidebarPanel"
            :url="infoslide.url"
            :title="infoslide.title"
            :footer="false"
            :header="false"
            >
        </el-slide>
        
        <el-slide ref="setting" id="cpeSettingOption" class="no-padding no-border slidebarPanel"
            :url="settingslide.url"
            :title="settingslide.title"
            :footer="false"
            :header="false"
            >
        </el-slide>
        <!-- 新建-->
		<el-slide class='ignore-border' ref="slide" 
			:url="slideUrlAdd" 
			:modal='slideModal' 
			:title="slideTitle" 
			:footer="slideFooter" 
			:header='slideHeader' 
			:position="slidePosition"
		 	:height="slideHeight" 
		 	:width="slideWidth" >	 	
		 </el-slide>
        <!--恢复出厂配置  -->
        <el-dialog title="<%=rb.getString("QueRen")%>" width="450px" :visible="showRestoreFactoryCard" class="confirmCard" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeRestoreFactory">		
			<el-form ref="restoreFactoryForm" :model='restoreFactoryForm'>     		     			                   	
	         	<el-form-item label='<%=rb.getString("QueRenHuiFuChuChangPeiZhiMa")%>' prop="keepConfig" >
					<el-checkbox v-model="restoreFactoryForm.keepConfig" :true-label="'1'" :false-label="'0'" ><%=rb.getString("BaoLiuPeiZhi")%></el-checkbox>					
				</el-form-item>
	        </el-form> 
	       	<div slot="footer" style="padding:10px;">
	          	 <el-button type="primary" @click="confirmRestoreFactory"><%=rb.getString("QueDing")%></el-button>
	             <el-button @click="closeRestoreFactory"><%=rb.getString("QuXiao")%></el-button>
	        </div> 		
		</el-dialog>

        <el-dialog id="modify_password_dialog" title="<%=rb.getString("QueRen")%>" 
            width="450"
            :visible.sync="pwdDlShow">
            <!-- info -->
            <div v-show="!isModified">
                <div style="padding: 0;">
                    <%=rb.getString("QueRenXiuGaiMiMa")%>
                </div>
                <el-form ref="password" :model="pwdForm" :rules="pwdRules" style="flex: auto; padding: 20px 0; position: relative;">
                    <el-form-item class="bold-title" label="<%=rb.getString("XinMiMa")%>" prop="password">
                        <el-password v-model="pwdForm.password" size="mini" placeholder="" show-password></el-password>
                        <el-input v-model="pwdForm.password" style="display: none;"></el-input>
                    </el-form-item>
                </el-form>
                <div style="text-align: right;">
                    <div class="bt-group linkbuttonGroup" style="margin: 0px;">
                        <a class="linkbutton" @click="savePassword"><span><%=rb.getString("QueDing")%></span></a>
                        <a class="linkbutton linkbutton_nowanna" @click="cancelModify"><span><%=rb.getString("QuXiao")%></span></a>
                    </div>
                </div>
            </div>
            <!-- result -->
            <div v-show="isModified">
                <div style="padding: 20px;">
                    <%=rb.getString("RenWuYiJianLi")%><span>{{pwdResultTips}}</span>
                </div>
                <div style="flex: auto;padding:10px 20px;">
                    <%=rb.getString("XiuGaiMiMaChengGongTiShi")%>
                </div>
                <div style="text-align: right;">
                    <div class="bt-group linkbuttonGroup" style="margin: 15px 20px;">
                        <a class="linkbutton" @click="toTaskPage"><span><%=rb.getString("QianWang")%></span></a>
                        <a class="linkbutton linkbutton_nowanna" @click="cancelModify"><span><%=rb.getString("QuXiao")%></span></a>
                    </div>
                </div>
            </div>
        </el-dialog>

        <!-- 收集报文 -->
        <el-dialog title="<%=rb.getString("QueRen")%>" top="30vh" width="550"
            :visible.sync="collectMessageShow" 
            :modal="false"
            :close-on-click-modal="false">
            <el-form :model="collectForm">
                <div>{{confirmTips}}</div>
                <el-form-item label='<%=rb.getString("ChiXuShiChang")%>' style="display: flex;align-items: center;margin: 5px 0px;">
                    <el-select v-model="collectForm.collectInterval" placeholder="Select time" class="collect-select">
                        <el-option label="05" value="05"></el-option>
                        <el-option label="10" value="10"></el-option>
                    </el-select>
                    <div style="display: inline;padding: 5px;margin-left: -4px;border: 1px solid #e9e9e9;background: #F5F7FA;"><%=rb.getString("ANRFenZhong")%></div>
                </el-form-item>
                <span v-if="cllectExisted">
                    <span style="color: #B3B3B3;"><%=rb.getString("ShouJiBaoWenFuGaiTiShi")%> SN={{existedMsgSN}}. </span>
                </span>
            </el-form>

            <div slot="footer" style="text-align: right;">
                <el-button type="primary" @click="sendCollect"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="collectMessageShow = false"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </el-dialog>
        
        <!-- 新建，导入弹窗 -->
        <el-dialog id="addAndImport_dialog" title="<%=rb.getString("TianJiaCPE")%>" width="630" custom-class='commonDialog'
        	:visible="addAndImportShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeAddAndImportDialog">
            <el-form :model='addAndImportForm' :rules="addAndImportRules" ref="addAndImportForm" label-position="top">							
               	<el-form-item label="<%=rb.getString("TianJiaLeiXing") %>" prop="selectType" style='margin-bottom: 20px;' class='leftLabel commonFlex'>
	                <el-radio-group v-model="addAndImportForm.selectType" >
	                    <el-radio label="0" border><%=rb.getString("ShouDongShuRu") %></el-radio>
	                    <el-radio label="1" border style='margin-left: 20px;'><%=rb.getString("PiLiangDaoRu") %></el-radio>
	                </el-radio-group>
	            </el-form-item>
	            <div v-show="addAndImportForm.selectType == '0'">
                    <el-form-item  prop='inputType' label="Input Type" style='margin-bottom:20px;' class='leftLabel commonFlex'>
                        <el-radio-group v-model="addAndImportForm.inputType">
                            <el-radio label="mac" border style='margin-right:20px;'>MAC</el-radio>
                            <el-radio label="sn" border style='margin-bottom:0px;'><%=rb.getString("CPEBianMa")%></el-radio>
                        </el-radio-group>
                    </el-form-item>
		            <el-form-item :label="addAndImportForm.inputType == 'mac' ? 'MAC' :'<%=rb.getString("CPEBianMa")%>'" prop='serialnumber' style='margin-bottom: 18px;'>
	                    <el-input type="textarea" v-model="addAndImportForm.serialnumber"></el-input>
	                </el-form-item>
	                <div class="tipText">
	                    <span class="el-icon el-icon-circle-info infoTip"></span>
	                    <span><%=rb.getString("cpeZhuCeTiShiWenZi")%></span>
	                </div>
	            </div>
                <div v-show="addAndImportForm.selectType == '1'" style="margin-bottom:0px;">
                     <el-form-item  prop='importType' label="Import Type" style='margin-bottom:20px;' class='leftLabel commonFlex'>
                        <el-radio-group v-model="addAndImportForm.importType">
                            <el-radio label="mac" border style='margin-right:20px;'>MAC</el-radio>
                            <el-radio label="sn" border style='margin-bottom:0px;'><%=rb.getString("CPEBianMa")%></el-radio>
                        </el-radio-group>
                    </el-form-item>
					<div style="display:flex;" class="uploadBox">
						<el-form-item label='<%=rb.getString("DaoRuWenJian")%>' prop='' style="margin-bottom: 0px;" class='importFileItem'>
							<el-upload ref="upload"
								:on-success='checkFile' 
								:on-change="fileChange" 
								:show-file-list=false 
								:action="uploadFileURL" 
								:data="fileParams" 
								name="uploadFile" 
								:auto-upload="false"
								accept=".xlsx, .csv">
								<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:340px;">
									<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
								</el-input>
								<span class="uploadFormat"><%=rb.getString("DaoRuWenJianLeiXing")%></span>
								<div slot="tip" class="el-upload__tip" v-show="!typeFlag"><%=rb.getString("ZhiZhiChiXLSXCSVWenJian")%></div>
								<div slot="tip" class="el-upload__tip" v-show="selectFlag"><%=rb.getString("QingXianXuanZeWenJian")%></div>
								<a slot="trigger" ref="file_up"></a>
							</el-upload>	
						</el-form-item>
					</div>
					
					<div class='commonFlex' style='margin-top: 12px;'>
						<div class="tipText">
							<span class="el-icon el-icon-circle-info infoTip"></span>
							<span><%=rb.getString("DaoRuWenJianTiShi")%></span>
						</div>
						<div style="cursor:pointer; margin-left: 16px; margin-top: -2px;" @click="exportTemplate">
							<span class='el-icon el-icon-common-download exportTemplateIcon'></span>
							<span class='exportTemplateText'><%=rb.getString("DaoChuMuBan")%></span>
						</div>
					</div>
				</div>
                <el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupId' class='commonSelect' style='margin-bottom: 0;'>
                    <el-select v-model="addAndImportForm.groupId">
                        <el-option v-for="item in deviceGroupSelections" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
                    </el-select>
                </el-form-item>
            </el-form>
            <div slot='footer'>
                <el-button-group size="mini">
                    <el-button type="primary" size="mini" @click="addAndImportSubmit"><%=rb.getString("QueDing")%></el-button>
                    <el-button size="mini" @click="closeAddAndImportDialog"><%=rb.getString("QuXiao")%></el-button>
                </el-button-group>
			</div>
        </el-dialog>
    </div>

    <script>
        //RSRP
        var lowVal = localStorage.getItem("rsrp1"),
            highVal = localStorage.getItem("rsrp2"),
            
            enableModifyPwd = writableMap['CODE_CPE_CHANGE_PASSWORD'],
            enableReboot = writableMap['CODE_CPE_REBOOT'],
            enableSync = writableMap['CODE_CPE_SYNCHRONIZE'],
            curCPECode = '', // 用户兼容单选模式的行选择记录

            deviceFlag = false,
            cpeShowColumns = '${showCol}',
			showProcessSetFlag = false;
            
        function isBatchable(){
            return enableModifyPwd == true || enableReboot == true || enableSync == true;
        }

        var cpevm = new Vue({
            el: '#cpeInfo',
            data() {
                var vm = this,
                    validatePWD = function(rule, value, cb) {
                        if(value) {
                            if(value.length<5 || value.length>15) { // 长度不符合 5 - 15位
                                cb('<%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%> 5-15 <%=rb.getString("ZiFuFuShu")%>');
                            }else {
                                var reg = /[\u4E00-\u9FA5\uF900-\uFA2D]/; // 中文校验
                                if(reg.test(value)) {
                                    cb('<%=rb.getString("FeiZhongWenZiFu")%>');
                                }else {
                                    cb();
                                }
                            }
                        }else {
                            cb('<%=rb.getString("QingShuRuMiMa")%>');
                        }
                    },
                    validatorNum = (rule,value,callback) => {
                        var serialNumber = value||'',
                        	list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
                                return item.length > 0;
                            }),
                        	temp = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/,
                       		noColTemp = /^([A-Fa-f0-9]{2}){6}$/,
                        	tempSn = /^(\d|[a-zA-Z]|-|\s){1,30}$/;
                        	
                       if(serialNumber != null && serialNumber.length != 0){
                            var nameFlag;
                            if(this.addAndImportForm.inputType == 'mac'){
                                nameFlag = list.every(function(item,index){
                                    return (temp.test(item) ||  noColTemp.test(item)) 
                                })
                            }else{
                                nameFlag = list.every(function(item,index){
                                    return tempSn.test(item)
                                })
                            }
                            if(nameFlag){
                                callback()
                            }else{
                                if(this.addAndImportForm.inputType == 'mac'){
                                    callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
                                }else{
                                    callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
                                }
                            }
                        }else if (serialNumber == null || serialNumber.length == 0) {
                            if(this.addAndImportForm.inputType == 'mac'){
                                callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
                            }else{
                                callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
                            }
                        }else{
                            callback();
                        }
                    };

                return {
                    cllectExisted: false,
                    existedMsgSN: '',
                    collectMessageShow: false,
                    collectForm: {
                        collectInterval: ''
                    },

                    sortColumns: [],
                    columns: [
                        {field: 'SERIAL_NUMBER', label: '<%=rb.getString("CPEXuLieHao")%>',sortable: true ,disabled: true, width: 180},
                        {field: 'CPE_NAME', label: '<%=rb.getString("CPEName")%>',sortable: true ,disabled: true, width: 120},
                        {field: 'IMSI', label: 'IMSI',sortable: true ,disabled: true, width: 150},
                        {field: 'MACADDRESS', label: 'MAC',sortable: true ,disabled: true, width: 150},
                        {field: 'IPADDRESS', label: 'IP',sortable: true ,disabled: true, width: 120},
                        {field: 'MODEL_NAME', label: '<%=rb.getString("ChanPinXingHao")%>',sortable: true ,disabled: true, width: 160},
                        {field: 'SOFTWARE_VERSION', label: '<%=rb.getString("RuanJianBanBen")%>',sortable: true ,disabled: true, width: 160},
                        {field: 'group_name', label: '<%=rb.getString("SheBeiZu")%>',sortable: true ,disabled: true, width: 150},
                        {field: 'HOST_NAME', label: '<%=rb.getString("HostName")%>',sortable: true ,disabled: true, width: 120},
                        {field: 'CELL_IDENTITY', label: 'ECI',sortable: true ,disabled: true, width: 120},
                        {field: 'SCANMODE', label: '<%=rb.getString("SaoMiaoFangShi")%>', width: 150},
                        {field: 'PCI', label: 'PCI',sortable: true ,disabled: true, width: 80},
                        {field: 'LGW_IP', label: '<%=rb.getString("LgwIPAddress")%>', width: 130},
                        {field: 'LGW_MAC', label: '<%=rb.getString("LgwMacAddress")%>',sortable: true, width: 170},
                        {field: 'UL_MCS', label: 'UL_MCS',sortable: true, width: 75},
                        {field: 'DL_MCS', label: 'DL_MCS',sortable: true, width: 75},
                        {field: 'lte_connection_time', label: '<%=rb.getString("LTEGengXinShiJian")%>',sortable: true, width: 100},
                        {field: 'DL_BLER', label: 'DL BLER',sortable: true, width: 75},
                        {field: 'RSRP0', label: 'RSRP1',sortable: true, width: 80},
                        {field: 'RSRP1', label: 'RSRP2',sortable: true, width: 80},
                        {field: 'CINR0', label: 'CINR1',sortable: true, width: 80},
                        {field: 'CINR1', label: 'CINR2',sortable: true, width: 80},
                        {field: 'CPE_SINR', label: 'SINR',sortable: true, width: 80},
                        {field: 'DL_CURRENT_DATARATE', label: '<%=rb.getString("CPEXiaXingTunTuLiang")%> (Mbps)',sortable: true, width: 180},
                        {field: 'UL_CURRENT_DATARATE', label: '<%=rb.getString("CPEShangXingTunTuLiang")%> (Mbps)',sortable: true, width: 180},
                        {field: 'UPTIME', label: '<%=rb.getString("YunXingShiJian")%>',sortable: true, width: 140},
                        {field: 'first_online_time', label: '<%=rb.getString("DiYiCiLianJieShiJian")%>',sortable: true, width: 140},
                        {field: 'LASTINFORMTIME', label: '<%=rb.getString("ShangCiLianJieShiJian")%>',sortable: true, width: 140},
                        {field: 'PRODUCT', label: '<%=rb.getString("ChanPinLeiXing")%>',sortable: true, width: 80},
                        {field: 'TX_POWER', label: '<%=rb.getString("CPETxPower")%>',sortable: true, width: 90},
                        {field: 'DL_EARFCN', label: '<%=rb.getString("PinDian")%>', width: 60},
                        {field: 'BANDWIDTH', label: '<%=rb.getString("DaiKuan")%>(MHz)', width: 130},
                        {field: 'MCC', label: 'MCC',sortable: true, width: 80},
                        {field: 'MNC', label: 'MNC',sortable: true, width: 80},
                        {field: 'longitude', label: '<%=rb.getString("JingDu")%>', width: 90},
                        {field: 'latitude', label: '<%=rb.getString("WeiDu")%>', width: 90},
                        {field: 'height', label: '<%=rb.getString("GaoDu")%>', width: 90},
                        {field: 'distance', label: '<%=rb.getString("JuLi")%>', width: 90},
                        {field: 'module_name', label: '<%=rb.getString("MoKuaiMingCheng")%>', width: 150},
                        {field: 'module_version', label: '<%=rb.getString("MoKuaiBanNen")%>', width: 150},
                        {field: 'MARKET_NAME', label: '<%=rb.getString("ChanPinMingCheng")%>', width: 150},
                        {field: 'IMEI', label: 'IMEI', width: 100,sortable: true},
                        {field: 'cpe_model', label: '<%=rb.getString("SheBeiXingHao")%>', width: 120},

                        {field: 'NR_BAND', label: 'NR-Band', width: 80},
                        {field: 'NR_BANDWIDTH', label: 'NR-<%=rb.getString("DaiKuan")%>(MHz)', width: 160},
                        {field: 'NR_PCI', label: 'NR-PCI', width: 80},
                        {field: 'NR_EARFCN', label: 'NR-<%=rb.getString("PinDian")%>', width: 90},
                        {field: 'NR_PLMN', label: 'NR-PLMN', width: 90},
                        {field: 'NR_CELL_ID', label: 'NR-Cell ID', width: 120},
                        {field: 'NR_DL_FREQUENCY', label: 'NR-DL_Frequency', width: 150},
                        {field: 'NR_UL_FREQUENCY', label: 'NR-UL_Frequency', width: 150},
                        {field: 'NR_CINR', label: 'NR-CINR', width: 90},
                        {field: 'NR_SINR', label: 'NR-SINR', width: 90},
                        {field: 'NR_RSRQ', label: 'NR-RSRQ', width: 90},
                        {field: 'NR_RSRP', label: 'NR-RSRP', width: 90},
                    ],
                    activeName: 'table',
                    showProps: '${showCol}'.split(','),
                    tbURL: '',
                    queryParams: {
                    	isCloudCore: isCloudCore,
                        TimeZone: timeZone,
                        search_text: '',
                        like_fields: '',
                        connection_status: '',
                        cpeMonitorModule: '',
                        product_model: '',
                        software_version: '',
                        group_id: ''
                    },
                    menus: [],
                    selectedRow: '',
                    selectedRows: [],

                    infoslide: {
                        title: '<%=rb.getString("XinXi")%>',
                        url: ''
                    },
                    settingslide: {
                        title: '',
                        url: ''
                    },

                    pwdDlShow: false,
                    isModified: false,
                    pwdForm: {
                        password: '',
                        devices: '',
                        timeZone: timeZone,
                        executeType: 'active',
                        fromCpeMonitor: 'yes'
                    },
                    pwdRules: {
                        password: [
                            {validator: validatePWD}
                        ]
                    },
                    pwdResultTips: '',
                    //恢复出厂配置
                    showRestoreFactoryCard:false,
        			restoreFactoryForm: {
        	        	//0 不保留，1  保留
        	        	keepConfig: '0',       	        	
        	       	},	
        	       	slideUrlAdd:'',
    				slideTitle:'',
    				slideFooter:'',
    				slideHeader:true,
    				slidePosition:'',
    				slideHeight:'',
    				slideWidth:'',
    				slideModal:'',
    				turboEnable:!isLWAEnable ? false : true,
    				//new
    				addAndImportShow: false,
    				addAndImportForm:{
                        serialnumber:'',
                        groupId:'',
                        selectType: '0',
                        inputType:'mac',
                        importType:'mac'
                    },
                    addAndImportRules:{
                        serialnumber:[
                            {validator: validatorNum,trigger:'blur'}
                        ]
                    },
                    deviceGroupSelections:[],
        			defaultGroupId:'',
        			selectFlag:false,        //标识是否选择了文件
        			typeFlag:true,           //校验已选择的文件格式
        			fileName:'',
        			fileParams:{},            //上传文件时自定义的参数   
        		 	uploadFileURL: '${ctx}/cell/CPE/uploadFile.action?importType=append',
        		 	bulkSelectShow:false,
        		 	settingUrl:'',
                }
            },
            computed: {
                limitBatch(){
                    return batchOperation ? '' : 1;
                },
                confirmTips() {
                    var vm = this,
                        sn = vm.selectedRow.SERIAL_NUMBER,
                        msg = '<%=rb.getString("QueRenShouJiPre")%>';

                    return msg.replace('placeholder', sn);
                },
                showColums() {
                    var vm = this,
                        columns = vm.getAllColumns(),
                        props = columns.map(function(col){
                            return col.prop;
                        });

                    props = props.filter(function(code){
                        return vm.showProps.includes(code);
                    });

                    return props;
                },
                hasRebootRole() {
                    return writableMap.CODE_CPE_REBOOT == true;
                },
                hasModifyPwdRole() {
                    return writableMap.CODE_CPE_CHANGE_PASSWORD == true;
                },
                isMonitorWritable() {
                    return writableMap.CODE_CPE_MONITOR == true;
                },
                isCurrentTab() {
                    var vm = this,
                        tabId = vm.$el.parentNode.id.replace('tab_content_', '');

                    return tabId == sysMain.$refs.nav.editableTabsValue;
                }
            },
            methods: {
                showCollectMessage(row) {
                    var vm = this,
                        paramsExist = {
                            type: 'cpe',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
                        var data = res.data;

                        if(data && data.isExist == 'true') {
                            vm.cllectExisted = true;
                            vm.existedMsgSN = data.serialNumber;
                        }else {
                            vm.cllectExisted = false;
                            vm.existedMsgSN = '';
                        }
                    });

                    vm.collectForm.collectInterval = '10';
                    vm.collectMessageShow = true;
                },
                sendCollect() {
                    var vm = this,
                        row = vm.selectedRow || {},
                        time = vm.collectForm.collectInterval+':00',
                        params = {
                            deviceCode: row.CPE_CODE,
                            serialNumber: row.SERIAL_NUMBER,
                            type: 'cpe',
                            operatorCode: operatorCodeGloab,
                            collectInterval: vm.collectForm.collectInterval
                        },
                        paramsExist = {
                            type: 'cpe',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
                        var data = res.data;

                        if(data && data.isExist == 'true') {
							vm.$message.error('SN=' + data.serialNumber + '<%=rb.getString("ZhengZaiShouJi")%>');
                        }else {
                            axios.post('${ctx}/trace/start.action', stringify(params)).then(function(res){
                                var data = res.data;

                                if(data.success == true) {
                                	queryCpeVue.startInterval(time);
                                	queryCpeVue.queryLatestInfo();
                                    vm.collectMessageShow = false;
                                    
                                    vm.$message.success('<%=rb.getString("ChengGong")%>');
                                }else {
                                    vm.$message.error(data.message);
                                }
                            });
						}
                    });
                },

            	refreshList() {
            		this.$refs.list.refresh();
            	},
            	loadSuccess(data) {
                    var ctner = document.querySelector('#cpeInfo');

                    ctner.style.width = '99.9%';
					setTimeout(function(){
						ctner.style.width = '100%';
					},1000);
            		// refresh_cpeStatusStatistics();
            	},
                getAllColumns() {
                    var vm = this,
                        columns = [
                            {prop:'lastsyntime'},
                            {prop:'CONNECTION_STATUS',title:''},
                            {prop:'SERIAL_NUMBER',title:'<%=rb.getString("CPEXuLieHao")%>'},
                            {prop:'CPE_CODE',title:'<%=rb.getString("XiaoZhanBianMa")%>'},
                            {prop:'CPE_NAME',title:'<%=rb.getString("CPEName")%>'},	                  
                            {prop:'IMSI',title:'IMSI'},
                            {prop:'MACADDRESS',title:'MAC'},
                            {prop:'IPADDRESS',title:'IP'},
                            {prop:'cpe_model',title:'<%=rb.getString("SheBeiXingHao")%>'},
                            {prop:'MODEL_NAME',title:'<%=rb.getString("ChanPinXingHao")%>'},
                            {prop:'SOFTWARE_VERSION',title:'<%=rb.getString("CPEVersion")%>'},
                            {prop:'group_name',title:'<%=rb.getString("SheBeiZu")%>'},
                            {prop:'HOST_NAME',title:'<%=rb.getString("HostName")%>'},
                            {prop:'CELL_IDENTITY',title:'ECI'},
                            {prop:'SCANMODE',title:'<%=rb.getString("SaoMiaoFangShi")%>'},
                            {prop:'PCI',title:'PCI'},
                            {prop:'LGW_IP',title:'<%=rb.getString("LgwIPAddress")%>'}, 
                            {prop:'LGW_MAC',title:'<%=rb.getString("LgwMacAddress")%>'}, 
                            {prop:'HISTORY_DATA',title:'<%=rb.getString("LiShiShuJu")%>'},
                            {prop:'UL_MCS',title:'UL_MCS'},
                            {prop:'DL_MCS',title:'DL_MCS'},
                            {prop:'lte_connection_time',title:'<%=rb.getString("LTEGengXinShiJian")%>'},
                            {prop:'DL_BLER',title:'DL BLER'},
                            {prop:'RSRP0',title:'RSRP1'},
                            {prop:'RSRP1',title:'RSRP2'},
                            {prop:'CINR0',title:'CINR1'},
                            {prop:'CINR1',title:'CINR2'},
                            {prop:'CPE_SINR',title:'SINR'},
                            {prop:'DL_CURRENT_DATARATE',title:'<%=rb.getString("CPEXiaXingTunTuLiang")%> (Mbps)'},
                            {prop:'UL_CURRENT_DATARATE',title:'<%=rb.getString("CPEShangXingTunTuLiang")%> (Mbps)'},
                            {prop:'UPTIME',title:'<%=rb.getString("YunXingShiJian")%>'},
                            {prop:'first_online_time',title:'<%=rb.getString("DiYiCiLianJieShiJian")%>'},
                            {prop:'LASTINFORMTIME',title:'<%=rb.getString("ShangCiLianJieShiJian")%>'},    
                            {prop:'PRODUCT',title:'<%=rb.getString("ChanPinLeiXing")%>'},
                            {prop:'TX_POWER',title:'<%=rb.getString("CPETxPower")%>'},
                            {prop:'DL_EARFCN',title:'<%=rb.getString("PinDian")%>'},
                            {prop:'BANDWIDTH',title:'<%=rb.getString("DaiKuan")%>(MHz)'},
                            {prop:'MCC',title:'MCC'},
                            {prop:'MNC',title:'MNC'},
                            {prop:'OLDPRODUCT'},
                            {prop:'longitude',title:'<%=rb.getString("JingDu")%>'},
                            {prop:'latitude',title:'<%=rb.getString("WeiDu")%>'},
                            {prop:'height',title:'<%=rb.getString("GaoDu")%>'},
                            {prop:'distance',title:'<%=rb.getString("JuLi")%>'},
                            {prop:'module_name',title:'<%=rb.getString("MoKuaiMingCheng")%>'},
                            {prop:'module_version',title:'<%=rb.getString("MoKuaiBanNen")%>'},
                            {prop: 'MARKET_NAME', title: '<%=rb.getString("ChanPinMingCheng")%>'},
                            {prop: 'IMEI', title: 'IMEI'},

                            {prop: 'NR_BAND', title: 'NR-Band'},
                            {prop: 'NR_BANDWIDTH', title: 'NR-<%=rb.getString("DaiKuan")%>(MHz)'},
                            {prop: 'NR_PCI', title: 'NR-PCI'},
                            {prop: 'NR_EARFCN', title: 'NR-<%=rb.getString("PinDian")%>'},
                            {prop: 'NR_PLMN', title: 'NR-PLMN'},
                            {prop: 'NR_CELL_ID', title: 'NR-Cell ID'},
                            {prop: 'NR_DL_FREQUENCY', title: 'NR-DL_Frequency'},
                            {prop: 'NR_UL_FREQUENCY', title: 'NR-UL_Frequency'},
                            {prop: 'NR_CINR', title: 'NR-CINR'},
                            {prop: 'NR_SINR', title: 'NR-SINR'},
                            {prop: 'NR_RSRQ', title: 'NR-RSRQ'},
                            {prop: 'NR_RSRP', title: 'NR-RSRP'},
                        ];

                    return columns;
                },
                showExport() {
                    $('.export-content').html('');
                    $('.export-content').each(function(idx,item){
                        if($(item).is(':visible')) {
                            $(item).load('/cell/CPE/toCpeExportConfig.action',function(html){})
                        }
                    });
                },
                selectChange(s) {
                    this.selectedRows = s||[];
                },
                // formatters
                connStatusFmt(value, row, index) {

                    return connStatusFormatter(value, row, index);
                },
                capablityFmt(value,rowData,rowIndex) {
                    var capablity = rowData.CAPABILITY;
                    var lte_turbo_enalbe = rowData.LTE_TURBO_ENABLE;
                    var rowIndexs = rowIndex;
                    if(capablity == '0' && isLWAEnable){
                        value = "<span class='cpeLwaWu' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
                    }else if(capablity =='1' && lte_turbo_enalbe == '1' && isLWAEnable){
                        value = "<span class='cpeLwaKai' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
                    }else if(capablity == '1' && lte_turbo_enalbe == '0' && isLWAEnable){
                        value = "<span class='cpeLwaGuan' style='font-size:22px'>"+ "<span style='font-size:12px;margin-left:25px'>"+(value)+"</span>"+"</span>"
                    }
                    return value;
                },
                softwareVersionFmt(value, row, index) {
                    var tips = '',
                        list = row.software_version_available || [],
                        itemText = '';

                    tips = [
                        '<div class="cellNameClass" style="cursor: pointer;" onclick=showSoftwareVersion('+JSON.stringify(list)+',event) >',
                            '<span class="el-icon el-icon-status-upgrading"></span>',
                        '</div>'
                    ].join('');

                    itemText = list.map(function(item){
                        var sv = item.software_version,
                            mv = item.module_version;
                        
                        if(mv && mv.length) {
                            sv += ' [ ' + mv.join(', ') + ' ]'
                        }
                        
                        return '<span class="item">'+sv+'</span>';
                    }).join('');
                    
                    if(list.length) {
                        return value + tips;
                    }else {
                        return value;
                    }
                },
                pciStatusFmt(value, rowData, rowIndex){
                    var reg = new RegExp("^(IDU\/CN)");
                    if("LTE WiFi VoIP Gateway" == rowData["OLDPRODUCT"] || (reg.test(rowData["OLDPRODUCT"])==true) || value == '--'){
                        return "--";
                    }
                    if(['',null,undefined].includes(value)) return "";
                    var pciValue = rowData.PCI;
                    if(rowData.SCANMODE == 'pcilock' || rowData.SCANMODE == 'pcionlylock' ){
                        var imgL = "<div class='pciClass'><span  title='<%=rb.getString("JieChuSuoDingToolTip")%>' class='el-icon el-icon-operation-lock yellowIcon easyui-tooltip' style='font-size:16px;' onclick=pciLockClick('" + rowData.PCI +"','" + rowData.SCANMODE +"','"+ rowData.CPE_CODE +"','" + rowData.PCI + "') ></span></div><span style='margin-left:5px;'>"+pciValue+"</span>" ;
                        return imgL;
                    }else{
                        var imgL = "<div class='pciClass'><span title='<%=rb.getString("BangDingToolTip")%>' class='el-icon el-icon-status-unlock easyui-tooltip' onclick=pciLockClick('" + rowData.PCI +"','" + rowData.SCANMODE +"','"+ rowData.CPE_CODE+"','" + rowData.PCI + "') ></span></div><span style='margin-left:5px;'>"+pciValue+"</span>" ;
                        return imgL;
                        
                    }
                    
                },
                redAccordingRSRPFmt(value, rowData, rowIndex) {
                    
                    return showRedAccordingRSRP(value, rowData, rowIndex);
                },
                CPETypeFmt(value, rowData, rowIndex) {
                    var reg = new RegExp("^IDU");

                    if (value == "LTE WiFi VoIP Gateway" || (reg.test(value)==true)) {
                        return "IDU";
                    } else {
                        return "ODU";
                    }
                },
                moduleVersionFmt(value, row, index) {
                    var tips = '',
                        list = row.module_version_available || [],
                        itemText = '';

                    tips = [
                        '<div style="color: blue;display: inline;padding: 0px 2px;cursor: pointer;" onclick=showModuleVersion('+JSON.stringify(list)+',event) >',
                            '<span>['+list.length+']</span>',
                        '</div>'
                    ].join('');

                    if(list.length) {
                        return value + tips;
                    }else {
                        return value;
                    }
                },
                openSettingPage(row){
                	var selCpe = row;
                	 sessionStorage.setItem('oldProduct', selCpe.OLDPRODUCT);
                     sessionStorage.setItem('CONNECTION_STATUS',selCpe.CONNECTION_STATUS);
                     sessionStorage.setItem('CPE_CODE',selCpe["CPE_CODE"]);
                     sessionStorage.setItem('PRODUCT',selCpe["PRODUCT"]);
                     sessionStorage.setItem('cpeName',selCpe["CPE_NAME"]||'');
                     
                     sessionStorage.setItem('lanFlag', selCpe.LAN_INTERFACE);
                     sessionStorage.setItem('cpeSN', selCpe.SERIAL_NUMBER);
                     sessionStorage.setItem('softVersion', selCpe.SOFTWARE_VERSION);
                     sessionStorage.setItem('imsi', selCpe.IMSI);
                     sessionStorage.setItem('model_name', selCpe.MODEL_NAME);
                     sessionStorage.setItem('module_name', selCpe.MODEL_NAME);

                    this.selectedRow = row;
                	
                	// 检查 eNBTopo_tab.jsp 是否已打开 setting.jsp 页面，如果打开则先关闭以防止 ID 冲突
                    if(typeof topovm !== 'undefined') {
                        try {
                            // 直接尝试关闭 eNBTopo_tab 的 slide，即使它没有打开也不会报错
                            topovm.$refs.topoSettingSlide.hide();
                        } catch(e) {}
                    }
                	
                	this.settingUrl = '${ctx}/cpe/setting/openSettingPage.action';
                	this.$refs.cpeSettingPage.showSlide(function(){
                		eventBus.$emit("row-data",row.CPE_CODE,row.MACADDRESS,row.SERIAL_NUMBER,row.connection_status,row,'monitor')
                    });
                	
                },
				closeSettingPage(){
                	this.$refs.cpeSettingPage.hide();
                    this.refreshList();
                },
                optClick(row, event) {
                    var vm = this,
                        cpeRowData = row,
                        status = cpeRowData.CONNECTION_STATUS,
                        oldproduct = cpeRowData.OLDPRODUCT,
                        softversion = cpeRowData.SOFTWARE_VERSION,
                        cpe_code = cpeRowData.CPE_CODE,
                        cpe_name = cpeRowData.CPE_NAME,
                        capablity = cpeRowData.CAPABILITY,
                        lte_turbo_enalbe = cpeRowData.LTE_TURBO_ENABLE,
                        lanInterface = cpeRowData.LAN_INTERFACE,
                        connectStatus = status,
                        XinXi = '<%=rb.getString("XinXi")%>',
                        TongBu = '<%=rb.getString("TongBu")%>',
                        SheZhi = '<%=rb.getString("SheZhi")%>',
                        ChongQi = '<%=rb.getString("ChongQi")%>',
                        SuoPin = '<%=rb.getString("SuoPin")%>',
                        GengDuoCaoZuo = '<%=rb.getString("GengDuoCaoZuo")%>',
                        apnSheZhi = 'APN/L2 <%=rb.getString("SheZhi")%>',
                        Kai =  'LTE-TURBO <%= rb.getString("KeYong")%>',
                        Guan = 'LTE-TURBO <%= rb.getString("BuKeYong")%>',
                        CeSu = '<%= rb.getString("CeSu")%>',
                        connectDisableFlag = true,
                        collectShow = is_super_user == 'true';

                    vm.selectedRow = row;
                    
                    //锁频是否可用
                    var suopinDisableFlag;
                    var product = oldproduct;
                    
                    //重启
                    var chongQidisableFlag;
                    chongQiShowFlag = true;
                    if(connectStatus != 'Off'){
                        chongQidisableFlag = false;
                        connectDisableFlag = false;
                    }else{
                        chongQidisableFlag = true;
                    }
                    if(cpe_name == 'null'){
                        cpe_name = '';
                    }
                    /* apn设置是否禁用 */
                    var apnDisableFlag = false;
                    var reg = new RegExp("^(IDU\/CN)");
                    var regIduEG = new RegExp("^(IDU\/EG)");
                    var regOduEG = new RegExp("^(ODU\/EG)");
                    var regu4G = new RegExp("^((ODU\/u4G)|(IDU\/u4G))");
                    var l2flag = false;
                    if (product == "LTE WiFi VoIP Gateway" || (reg.test(product)==true)) {
                        l2flag = false;
                    } else if(regIduEG.test(product)){
                        l2flag = true;
                    } else if(regOduEG.test(product)){
                        l2flag = true;
                    } else if(regu4G.test(product)){
                        l2flag = true;
                    } else {
                        l2flag = false;
                    }
                    if(l2flag){
                        apnDisableFlag = false;
                    } else {
                        apnDisableFlag = true;
                    }
                    var showKai = false ;
                    var showGuan = false;
                    var notUserFlag = false;
                    var capablity = capablity;
                    var lte_turbo_enalbe = lte_turbo_enalbe;
                    if(lte_turbo_enalbe == '0' && capablity!='0'){ // turbo为关闭状态 显示开按钮
                        showKai = true
                    }
                    if(lte_turbo_enalbe == '1' && capablity!='0'){ // turbo为开启状态 显示关按钮
                        showGuan = true
                    }
                    if(!isLWAEnable) {
                        showGuan = false;
                        showKai = false;
                    }

                    var cpeEnable = true;
                    $.ajax({
                        type:'POST',
                        url:'${ctx}/cell/CPE/getCPEOperationItem.action',
                        data:{cpeCode: cpe_code},
                        async:false,
                        dataType:'json',
                        success:function(data){
                            if(data) {
                                cpeEnable = data.cpeFlag == true;
                            }
                        }
                    });

                   <%--  vm.menus = [
                        {label: XinXi, row: row,id:"cp1",cls:'el-icon el-icon-operation-info',show:true},
                        {label: CeSu, row: row,id:"cp9",cls:'el-icon el-icon-operation-diagnostic',show:cpeEnable,cpe_code:cpe_code,disable:connectDisableFlag},
                        {label: SheZhi, row: row,id:"cp2",cls:'el-icon el-icon-operation-settings CODE_CPE_SETTINGS hidden',show:cpeEnable,disable:connectDisableFlag,cpe_code:cpe_code,product:oldproduct,softVersion:softversion,cpe_name:cpe_name,LAN_INTERFACE: lanInterface},
                        {label: GengDuoCaoZuo, row: row,id:'cp',cls:'el-icon el-icon-operation-more-circle CODE_CPE_SYNCHRONIZE CODE_CPE_REBOOT hidden', show: cpeEnable,
                            child: [
                                {label: ChongQi, row: row,id:"cp5",cls:'CODE_CPE_REBOOT hidden',show:chongQiShowFlag,disable: chongQidisableFlag},
                                {label:'<%=rb.getString("ShouJiBaoWen")%>', row: row, id: 'collect',show: collectShow},
                                {label:'<%=rb.getString("RiZhiShouJi")%>',row: row, id:'logs',cls:'CODE_CPE_LOGS hidden',disable: connectDisableFlag},
                            ]
                        },
                        {label: Guan, row: row,id:'cp7',cls:'el-icon el-icon-operation-disable1 CODE_CPE_SETTINGS hidden',show:showGuan && cpeEnable,cpe_code:cpe_code},
                        {label: Kai, row: row,id:'cp8',cls:'el-icon el-icon-operation-enable1 CODE_CPE_SETTINGS hidden',show:showKai && cpeEnable,cpe_code:cpe_code},
                    ]; --%>
				
                    var group3 = [
                        {label: Guan, row: row,id:'cp7',cls:'CODE_CPE_SETTINGS hidden',show:showGuan && cpeEnable,cpe_code:cpe_code},
                        {label: Kai, row: row,id:'cp8',cls:' CODE_CPE_SETTINGS hidden',show:showKai && cpeEnable,cpe_code:cpe_code},
                    ];
                    
                    var group3Flag = false;
                    if(cpeEnable && (showGuan || showKai)){
                        group3Flag = true;
                    }
                   
                    vm.menus = [
                        { row: row,cls:'', show: collectShow,
                            child: [
                                {label:'<%=rb.getString("ShouJiBaoWen")%>', row: row, id: 'collect',show: collectShow},
                            ]
                        },
                        { row: row,cls:'', show: chongQiShowFlag,
                            child: [
                                {label: ChongQi, row: row,id:"cp5",cls:'CODE_CPE_REBOOT hidden',show:chongQiShowFlag,disable: chongQidisableFlag},
                            ]
                        },
                        { row: row,cls:'',show:group3Flag, 
                            child: group3
                        },
                    ];
                    
                    vm.$nextTick(function(){
                        document.body.click();
                        vm.$refs.menu.show(event);
                    });
                },
                handerClose() {
                    this.$refs.menu.hide();
                },
                menuClick(row) {
                    var vm = this,
                        code = row.id,
                        actions = {
                            cp1: vm.goCpeInfo,
                            cp2: vm.goSetting,
                            cp5: vm.rebootCpe,
                            cp7: vm.closeLTE,
                            cp8: vm.openLTE,
                            cp9:vm.goDiagnostic,
                            collect: vm.showCollectMessage,
                            logs: vm.logCollection
                        };

                    if(actions[code]) {
                        actions[code](row.row);
                    }
                },
                goCpeInfo(row) {
                    var vm = this,
                        selCpe = row;
	 
                    if (!selCpe) {
                        showMsg('prompt_msg','<%=rb.getString("QingXuanZeSheBei")%>');
                        return;
                    }
                    
                    sessionStorage.setItem('oldProduct', selCpe.OLDPRODUCT);
                    sessionStorage.setItem('CONNECTION_STATUS',selCpe.CONNECTION_STATUS);
                    sessionStorage.setItem('CPE_CODE',selCpe["CPE_CODE"]);
                    sessionStorage.setItem('PRODUCT',selCpe["PRODUCT"]);
                    sessionStorage.setItem('cpeName',selCpe["CPE_NAME"]||'');

                    vm.infoslide.url = "${ctx}/cell/CPE/toCpeDetailParamInfoPage.action?cpeCode=" + selCpe["CPE_CODE"]+'&timeZone='+timeZone;
                    vm.$refs.info.showSlide(function(){
                        $.parser.parse(document.querySelector("#cpeInformation"));
                    });
                },
                goDiagnostic(row){
                	var vm = this;
		    	    vm.settingslide.url = "${ctx}/cpe/diagnostics/toDiagnosticsPage.action";
                    vm.$refs.setting.showSlide(function(){
                       eventBus.$emit('cpe-test',row.CPE_CODE,row.SERIAL_NUMBER,row.CPE_NAME);
                    });
                },
                goSetting(row) {
                    var vm = this,
                        selCpe = row,
                        cpeName = selCpe.CPE_NAME||'',
                        cpeCode = selCpe.CPE_CODE,
                        softVersion = selCpe.SOFTWARE_VERSION,
                        product = selCpe.OLDPRODUCT;
	 
                    if (!selCpe) {
                        showMsg('prompt_msg','<%=rb.getString("QingXuanZeSheBei")%>');
                        return;
                    }

                    sessionStorage.setItem('lanFlag', selCpe.LAN_INTERFACE);
                    sessionStorage.setItem('oldProduct', product);
                    sessionStorage.setItem('cpeSN', selCpe.SERIAL_NUMBER);
                    sessionStorage.setItem('softVersion', softVersion);
                    sessionStorage.setItem('cpeName', cpeName);

                    vm.settingslide.url = "${ctx}/cell/CPE/toCpeSettingPage.action";

                    vm.$refs.setting.showSlide({cpeCode: cpeCode, product: product, softVersion: softVersion, cpeName: cpeName}, function(){
                        $.parser.parse(document.querySelector("#cpeSettingOption"));
                    });
                },
                // CPE 日志收集
                logCollection(row){
                    var vm = this,
                        urls="${ctx}/cell/collect/goImmediateCollectLogFile.action"
                        params = {
                            timeZone : timeZone,
                            isReboot:false,
                            device_type : 'CPE',
                            device_code : row.CPE_CODE,
                            execute_type: 'Immediately',
                            start_time: '',
                        };
                    axios.post(urls,stringify(params)).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            vm.$message({
                                type: 'success',
                                message: '<%=rb.getString("CPERiZhiZhengZaiShouJi")%>'
                            });
                        }else{
                            vm.$message.error(data["message"])
                        }       			    		
                    })
                },
                //批量配置参数
                paramsConfigBatch(){
                	var vm = this;
                	var checkedRow = vm.selectedRows || [],
                 		cpeCodes = checkedRow.map(function(item){ return item.CPE_CODE; }).join(',');
    				vm.slideHeader = true;
    	    	    vm.slideTitle = '<%=rb.getString("CanShuZiPeiZhi")%>';
    	    	    vm.slideFooter = false;
    	    	    vm.slidePosition = 'top';
    	    	    vm.slideHeight = '100%';
    	    	    vm.slideWidth = '100%';
    	    	    vm.slideUrlAdd = '${ctx}/cpe/batchconfig/toBatchParamsConfigPage.action';			    	    
		    	    vm.$refs.slide.showSlide(function(){	    	    	
						eventBus.$emit("modify-task", cpeCodes);
		    	    });
                },
                //恢复出厂配置
                restoreFactoryBatch(){
                	this.showRestoreFactoryCard = true;
                },
              	//恢复出厂确认
        		confirmRestoreFactory(){
        			var vm = this;
        			vm.$refs.restoreFactoryForm.validate((valid) => {
        				if(valid){	
                            var checkedRow = vm.selectedRows || [],
                             	cpeCodes = checkedRow.map(function(item){ return item.CPE_CODE; }).join(',');
        					var params = {
        						timeZone : timeZone,
        						type : 'monitor',
        						cpeCodes : cpeCodes,
        						executeType : 'active',
        						selectAll : '0',
        						keepConfig : vm.restoreFactoryForm.keepConfig
        					};
        					axios.post('${ctx}/cpe/factoryReset/addTask.action',stringify(params)).then(function(response){
        			    		var data = response.data;
        			    		if(data["success"]){
        			    			vm.$refs.list.refresh();
        			    			vm.selectedRows = [];
        		    				vm.$refs.list.clearSelection();
        			    			vm.closeRestoreFactory();
        			    			vm.$message({
                                        type: 'success',
                                        message: '<%=rb.getString("ChengGong")%>'
                                    });
        						}else{
        							vm.$message.error(data["message"])
        						}       			    		
        			    	})
        				}
        			})
        		},
        		//恢复出厂取消
        		closeRestoreFactory(){
        			var vm = this;
        			vm.restoreFactoryForm.keepConfig = '0';       			
        			vm.showRestoreFactoryCard = false;
        		},
        		
                rebootCpe(row) {
                    var selCell = row;
	 
                    // 单子30936：后端确认直接发起重启请求
                    $.messager.confirm('<%=rb.getString("QueRen")%>', '<%=rb.getString("QueDingChongQiSheBei")%>', function (r) {
                        if (r) {
                            var param = {
                                cpeCodes: selCell["CPE_CODE"]
                            };
                            $.post("${ctx}/cell/CPE/rebootCpe.action", param, function (data) {
                                if (data["success"]) {
                                    showMsg('prompt_msg','<%=rb.getString("MingLingYiXiaFa")%>');
                                }else{
                                    showMsg('error_msg',data["message"]);
                                }
                            }, "json");
                        }
                    }).addClass('seriousConfirm');
                },
                rebootCpeList() {
                    var vm = this,
                        checkedRow = vm.selectedRows || [],
                        cpeCodes = checkedRow.map(function(item){ return item.CPE_CODE; }).join(',');
					
                    if(vm.selectedRows.length == 0) return;
                    
                    var params = {
                            cpeCodes : cpeCodes
                        };

                    $.messager.confirm('<%=rb.getString("QueRen")%>', '<%=rb.getString("QueDingChongQiSheBei")%>', function (r) {
                        if (r) {
                            $.post("${ctx}/cell/CPE/rebootCpe.action", params, function (data) {
                                if (data["success"]) {
                                    showMsg('prompt_msg','<%=rb.getString("MingLingYiXiaFa")%>');
                                    delAllSelectedRecord();
                                }else{
                                    showMsg('error_msg',data["message"]);
                                }
                            }, "json");
                        }
                    }).addClass("seriousConfirm");
                },
                openModifyPwd() {
                    var vm = this;
                    
                    if(vm.selectedRows.length == 0) return;
                    vm.pwdResultTips = '';
                    vm.isModified = false;
                    vm.pwdDlShow = true;
                    vm.$nextTick(function(){
                        vm.$refs.password.resetFields();
                    });
                },
                savePassword() {
                    var vm = this,
                        rows = vm.selectedRows,
                        devices = rows.map(function(row){ return row.CPE_CODE;}).join(','),
                        params = vm.pwdForm;

                    params.devices = devices;

                    $.post("${ctx}/task/cpe/changepwd/addTask.action", params, function(data){
                        if (data["success"]) {
                            vm.isModified = true;
                            vm.$refs.list.clearSelection();
                            vm.pwdResultTips = data['message'];
                            showMsg('prompt_msg','<%=rb.getString("ChengGong")%>');
                        } else {
                            showMsg('error_msg',data["message"]);
                            vm.cancelModify();
                        }
                    }, "json");
                },
                // 批量移入回收站设备
                recycleCells(){
                    var vm = this,
                        params = {},
                        urls= '${ctx}/recycle/moveCpeDeviceToRecycle.action',
                        idsList = [];
                    if(vm.selectedRows.length <= 0)return
                    vm.selectedRows.map((item,index) => {
                        idsList.push(item.CPE_CODE);
                    })
                    params.cpeCodeStr = idsList.join(',');
                    var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueRenJiangSheBeiYiRuHuiShouZhan")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("YiRuHuiShouZhanTiShi")%>' +'</div>';
                    vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
                        customClass:"warningConfirm",
                        confirmButtonText:'<%=rb.getString("QueDing")%>',
                        cancalButtonText:'<%=rb.getString("QuXiao")%>',
                        type:'warning',
                        dangerouslyUseHTMLString:true
                    }).then(()=>{
                        axios.post(urls,stringify(params)).then(function(response){
                            let data = response.data;
                            if ( data.success ){
                                vm.$message({
                                    message: '<%=rb.getString("ChengGong")%>' ,
                                    type:'success',
                                })
                                vm.$refs.list.refresh();
                                vm.$refs.list.clearSelection();
                            }else {
                                vm.$message.error(data.message)
                            }
                        }).catch(function(error){})
                    }).catch(()=>{})
                },
                toTaskPage() {
                	this.cancelModify();
                	
                	try{
                		eventAllBus.$emit("gomenupage","70024","","70024",false);
                	}catch(e){}
                },
                cancelModify() {
                    this.pwdDlShow = false;
                },
                openLTE(row) {
                    this.setLTE(row, '1');
                },
                closeLTE(row) {
                    this.setLTE(row, '0');
                },
                setLTE(row, status) {
                    //1-开启 0-关闭
                    var vm = this,
                        cellCode = row.CPE_CODE,
                        Msg = '';

                    if(status==="0") {
                        Msg = '<%=rb.getString("PiLiangGuanCapacity")%>';
                    }else {
                        Msg = '<%=rb.getString("PiLiangKaiCapacity")%>';
                    }

                    $.messager.confirm('<%=rb.getString("QueRen")%>', Msg, function (r) {
                        if (r) {
                            var params = {
                                    cpeCodes : cellCode,
                                    turboEnable : status
                                };

                            $.post("${ctx}/cell/ap/turboSetting.action",params,function(data){
                                if(data["success"]){
                                    showMsg('success_msg','<%=rb.getString("ChengGong")%>');
                                    vm.$refs.list.refresh();
                                }else{
                                    showMsg('error_msg',data["message"]);
                                }
                            },"json")
                        }
                    }).addClass('normalConfirm');
                },
              //关闭slide,更新表格数据
    			cancelSlide() {
    				var vm = this;								
    				vm.$refs.list.refresh();
	    			vm.selectedRows = [];
    				vm.$refs.list.clearSelection();
    				vm.$refs.slide.hide();
    			},	
    			
    			//新建及导入弹窗
    			addDevice(){
    				var vm = this;
    				vm.typeFlag = true;
                    vm.selectFlag = false;
                    vm.fileName = '';
                    vm.fileParams.FileName = '';
                    vm.fileParams.group_id = vm.defaultGroupId; 
                    
    				vm.addAndImportShow = true;
    			},
    			//获取设备组数据
    			addDeviceInit(){
                    var vm = this;
                    axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then((res) => {
                        var data = res.data;
                        if(data){
                            vm.deviceGroupSelections = data; 						
                            vm.defaultGroupId = data[0].id;
                            vm.addAndImportForm.groupId = vm.defaultGroupId;
                        }					
                    })				
                }, 
    			
                checkFile(res,file){    //发送请求，校验device文件内容 
        			var vm = this;
        			if(res.success){
                        if(res.msg){
                            vm.$message({
                                message: res.msg,
                                type:'success',
                            })
                        }else{
                            vm.$message({
                                message: '<%=rb.getString("ChengGong")%>',
                                type:'success',
                            })
                        }
        				vm.addAndImportShow = false;
        				cpevm.refreshList();//刷新列表
        				vm.closeFileSelect();
        			}else{
        				vm.$message({
        					type: 'error',
        					message: res.msg
        				});
        			}
        			//修改已选择文件状态  
        			var fileList = vm.$refs.upload.uploadFiles;
        			fileList.forEach(function(file){
        				file.status = 'ready';
        			})
        		},
        		/**
        		* 选择文件后，校验格式，并赋值页面显示 
        		* @param file{object}   文件信息
        		* @param fileList{Array}  文件列表
        		*/ 
        		fileChange(file,fileList){ 
        			var vm = this;
        			vm.selectFlag = false;
        			var groudId = '';
        			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.csv';
        			vm.typeFlag = typeFlag;
        			if(typeFlag){
                        vm.fileName = file.name;
                    }else {
                        vm.fileName = '';
                    }
        		},
        		// 选择文件
        		fileSelect(){  
        			var vm =this;
        			vm.$refs.upload.clearFiles();
        			vm.$refs['file_up'].click();
        		},
        		// 移除导入文件
        		closeFileSelect(){
        			var vm = this;
        			vm.fileName = '';
        			vm.typeFlag = true;
        			vm.selectFlag = false;
        			vm.$refs.upload.clearFiles();
        		},

        		addAndImportSubmit(){
        			//0-输入； 1- 导入
    				var param={} , vm = this , url ,id;
                    var message = "<%=rb.getString("TianJiaSheBeiChengGong")%>";
                        id = vm.addAndImportForm.groupId;
                        url = "${ctx}/cell/CPE/addAndAssignCpe.action",
                        mac = vm.addAndImportForm.serialnumber,
                        list = mac.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
	
                    //mac = mac.replace(/[(;\s*)(\r\n)\r\n]+/g,';');
                    
                    if(vm.addAndImportForm.selectType == '1'){
                    	vm.fileParams.FileName = vm.fileName;
        				vm.fileParams.group_id = vm.addAndImportForm.groupId;
                        if(vm.addAndImportForm.importType == 'mac'){
                            vm.uploadFileURL = '${ctx}/cell/CPE/uploadFile.action?importType=append';
                        }else{
                            vm.uploadFileURL = '${ctx}/cell/CPE/uploadFile2.action?importType=append';
                        }
        				if(vm.fileParams.FileName){
                            setTimeout(() => {
                                vm.$refs.upload.submit();
                            }, 400);
        				}else{
        				    vm.typeFlag = true;
        					vm.selectFlag = true;
        				}
                    }else{
                        var params = {
                                group_id:vm.addAndImportForm.groupId,
                                type:vm.addAndImportForm.inputType
                            }
                        if(vm.addAndImportForm.inputType == 'mac'){
                            params.macAddress = list.join(';');
                        }else{
                            params.sns = list.join(';');
                        }
	                    vm.$refs.addAndImportForm.validate((valid) => {
	                        if(valid){
	                            axios.post(url,stringify(params)).then(function(response){
	                                let data = response.data;
	                                if (data.success){
	                                    //刷新表格,关闭新建设备弹窗
	                                    cpevm.$refs.list.refresh();
	                                    vm.closeAddAndImportDialog();
	                                    vm.$message({
	                                        message: message,
	                                        type:'success',
	                                    })
	                                }else {
	                                    vm.$message.error(data.message)
	                                }
	                                
	                            }).catch(function(error){})
	                            
	                        }else{
	                        }
	                    })
                    }
    			},
    			closeAddAndImportDialog(){
    				var vm = this;
    				vm.commonCloseDialog();
                    vm.addAndImportShow = false;
                    document.body.click();
                    vm.typeFlag = true;
        			vm.selectFlag = false;
        			vm.fileName = '';
        			vm.fileParams.FileName = '';
        			vm.fileParams.group_id = vm.defaultGroupId; 
    			},
    			commonCloseDialog(){
                    var vm = this;
                    vm.$refs.addAndImportForm.resetFields();
                    vm.addAndImportForm.groupId = vm.defaultGroupId;
                },
              	// 导出设备模板
        	    exportTemplate(){
        	    	var vm = this, 
                        url = "${ctx}/cell/CPE/downloadImportCpeTemplate.action",
                        params = {
                            group_id:vm.defaultGroupId,
                            type:vm.addAndImportForm.importType,
                            search_text:'',
                            like_fields:'serial_number'
                        };

                	var bool = checkParams(params)
        			if(!bool) return false;
                	exportByForm(url,params)
        	    },
        	 	// 打开已选弹窗
                openBulkSelectTable(){
                    var vm = this;
                    vm.bulkSelectShow = true
                },
                // 关闭已选弹窗
                closeBulkSelectTable(){
                    var vm = this;
                    vm.bulkSelectShow = false;
                },
                // 设备已选表格 清空事件
                clearBulkSelected(){
                    var vm = this;

			        vm.$refs["list"].clearSelection();
                },
                // 设备已选表格 单个删除事件
                delBulkSelected(rows){
                    var vm = this,
                        tabs = 'list',
                        rowKey = 'CPE_CODE';
                    vm.selectedRows = vm.selectedRows.filter((items)=>{
                        return items[rowKey] != rows[rowKey]
                    });
                    var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
                        irow= selection.filter((items)=>{
                            return items[rowKey] == rows[rowKey]
                        })[0];
                    vm.$refs[tabs].toggleRowSelection(irow,false);
                    var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
                    vm.$refs[tabs].ckList.splice(idx,1);
                },
                // 关闭同步名称弹窗
                closeSyncName() {
                    document.body.click();
                },
                syncName(code, newName) {
                    var vm = this,
                    urls = '${ctx}/cell/CPE/updateCpeName.action',
                    params={
                        cpeCode:code,
                        cpeName:newName
                    };
                    axios.post(urls,stringify(params)).then(function(response){
                        let data = response.data;
                        if (data.success){
                            //刷新表格,关闭新建设备弹窗
                            vm.$refs.list.refresh();
                            vm.closeSyncName();
                        }else {
                            vm.$message.error(data.message)
                        }
                    }).catch(function(error){})
                },
            },
            created() {
                var vm = this;

                axios.post('${ctx}/system/column/setting/load/2').then(function(res){
                    var data = res.data.data||{},
                        sortCodes = (data.sortColumn||'').split(','),
                        sortList = vm.columns;

                    vm.showProps = (data.showColumn||'').split(',');
                    vm.sortColumns = sortCodes;

                    vm.columns = sortList.sort(function(n, m) {
                        var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
                            idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

                        return idxn - idxm;
                    });
                    
                    var tbStates = vm.$refs.list.$refs.ctableInner.store.states,
	                    storeCols = tbStates.columns;
	
	                tbStates._columns = storeCols.sort(function(n, m) {
	                    var idxn = sortCodes.indexOf(n.property)==-1?100:sortCodes.indexOf(n.property),
	                        idxm = sortCodes.indexOf(m.property)==-1?100:sortCodes.indexOf(m.property);
	
	                    if( [undefined,'cpe_operation','CONNECTION_STATUS'].includes(n.property) ) {
	                        idxn = 0;
	                    }
	                    if( [undefined,'cpe_operation','CONNECTION_STATUS'].includes(m.property) ) {
	                        idxm = 0;
	                    }
	
	                    return idxn - idxm;
	                });
	
	                vm.$refs.list.$refs.ctableInner.store.updateColumns();
	                
	                loadHTML(document.querySelector('#toolbar_tableHomeCpeList'), {
	                    url: '${ctx}/cell/CPE/toCpeQuery.action'
	                });
	                
	                vm.tbURL = '${ctx}/cell/CPE/queryCpeInfosList.action?type=0';
                });
            },
            mounted() {
                // 排序
                var vm = this,
                    sortCodes = cpeShowColumns.split(','),
                    sortList = vm.columns;
                    
				vm.addDeviceInit();
                eventBus.$off('hide-slide').$on('hide-slide',this.cancelSlide);
                eventBus.$on('cancel-cpe-setting',vm.closeSettingPage);
            }
        });

       

        // ************************************  charts init  ************************************

        function pciLockClick(pciOnlyLock,scanMode,cpeCode,pciValue) {
            var tipText = "",
                params = {
            		pciChanged: 1
            };

            params.cpeCode = cpeCode;
            params.timeZone = timeZone;

            if(scanMode == 'pcilock' || scanMode == 'pcionlylock' ){
                params.scanMode='fullband';
                tipText = '<%=rb.getString("JieChuSuoDingTanChuangTiShi")%>'; 
            }else{
                params.PCI_value = pciOnlyLock;
                params.scanMode='pcionlylock';
                tipText = '<%=rb.getString("BangDingTanChuangTiShiOne")%>'+ pciValue + '<%=rb.getString("BangDingTanChuangTiShiTwo")%>'; 
            }

            $.messager.confirm('<%=rb.getString("QueRen")%>', tipText, function (r) {
                if (r) {
                    $.post("${ctx}/cell/CPE/setCpeParams.action", params, function(data) {
                        if (data["success"]) {
                            showMsg('prompt_msg','<%=rb.getString("SuoPingRenWuJianLiTiShi")%>');
                            cpevm.refreshList();
                        } else {
                            showMsg('error_msg',data["message"]);
                            return;
                        }
                    }, "json");
                }
            }).addClass('seriousConfirm');
        }

        function showRedAccordingRSRP(value, rowData, rowIndex) {
            var maxValue = highVal,
                minValue = lowVal;
            
            var ret ;
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

        function showModuleVersion(list, evt) {
            var tips = '',
                list = list || [],
                itemText = '';

            itemText = list.map(function(item){
                
                return '<span class="item">'+item+'</span>';
            }).join('');
            
            tips += '<div class="version-details normal">' + itemText + '</div>';

            showPopLayer({event: evt, html: tips});
        }
		
        //刷新Cpe监控 下 统计信息，填充状态栏
        function refresh_cpeStatusStatistics(cb) {
        	var params = {};
        	
        	$.extend(params, cpevm.queryParams);

        	$.post("${ctx}/cell/CPE/getCpeStatusStatistics.action", params, function(data) {
        		if(!data["connection_status"]){
        			connectionStatus = "0/0";
        			connectionStautsRef = "0/0";
        			$("#onlineStatusNumCpe").text("0/0");
        			//$("#cpeMonitorId .connStatusStatistics").text("0/0");
        		}else{
        			connectionStatus = data["connection_status"];
        			connectionStatusRef = data["connection_status_ref"];
        			if($("#onlineStatusNumCpe").prev().hasClass('greenType')){
        				$("#onlineStatusNumCpe").text(data["connection_status"]);
        			}else if($("#onlineStatusNumCpe").prev().hasClass('redType')){
        				$("#onlineStatusNumCpe").text(data["connection_status_ref"]);
        			}
        			//$("#cpeMonitorId .connStatusStatistics").text(data["connection_status"]);
        		}
        		$('#cpe_online_count_rate').text('( '+connectionStatus+' )');
        		if(cb && typeof cb == 'function') {
        			cb();
        		}
        	}, "json");
        }
        
        function ltmkai(statu){
        	var checkedRow = cpevm.selectedRows;
        	var cpeCodes = '';
        	
        	if(cpevm.selectedRows.length == 0)return;
        	
        	checkedRow.map(function(item){
        		cpeCodes += item.CPE_CODE + ',';
        	})
        	var params = {
        		turboEnable : statu,
        		cpeCodes : cpeCodes.substring(0,cpeCodes.length-1)
        	}
        	let Msg = ''
        	if(statu=="0"){
        		Msg = "<%=rb.getString("PiLiangGuanCapacity")%>"
        	}else{
        		Msg = "<%=rb.getString("PiLiangKaiCapacity")%>"
        	}
        	$.messager.confirm('<%=rb.getString("QueRen")%>', Msg, function (r) {
       			if (r) {
       				$.post("${ctx}/cell/ap/turboSetting.action",params,function(data){
       					if(data["success"]){
       						showMsg('success_msg','<%=rb.getString("ChengGong")%>');
       						cpevm.refreshList();
       						cpevm.$refs.list.clearSelection();
       					}else{
       						showMsg('error_msg',data["message"]);
       					}
       				},"json")
       			}
       		}).addClass('normalConfirm');
        }
        
        function delAllSelectedRecord() {
        	cpevm.$refs.list.clearSelection();
        }
        
        function showSoftwareVersion(list,evt) {
        	var tips = '',
        		list = list || [],
        		itemText = '';

        	itemText = list.map(function(item){
        		var sv = item.software_version,
        			mv = item.module_version;
        		
        		if(mv && mv.length) {
        			sv += ' [ ' + mv.join(', ') + ' ]'
        		}
        		
        		return '<span class="item">'+sv+'</span>';
        	}).join('');
        	
        	itemText = '<span class="item" style="font-weight: bold;"><%=rb.getString("SoftwareVersion")%> [<%=rb.getString("MoKuaiBanNen")%>]</span>' + itemText;

        	tips += '<div class="version-details normal">' + itemText + '</div>';

        	showPopLayer({event: evt, html: tips});
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
            var showStr = EARFCN.toString()+"("+frequency.toString()+"MHz"+")";
            
            return showStr;
        }
        /*设置提示进度条*/
        function progressDivShow(){
        	$('.setting').css('display','block');
        	$('.set').css('display','none');
        	$('#winSettingProCpe').show();
        }

        /*完成后的进度条*/
        function progressDivHide(){	
        	$('.setting').hide();
        	if(showProcessSetFlag){
        		$('.set').hide();
        	}else{
        		$('.set').fadeIn();
        	}
        	setTimeout(function(){			
        		//$('.newWindow').fadeOut(400);
        		$('#winSettingProCpe').fadeOut(400);
        	},2000)	
        }
    </script>
</body>