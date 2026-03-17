<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
	#imsi_ctn .el-tabs {
        width: 100%;
    }
    .npn-config .slide-content {
    	padding: 0px;
    }
    .normal-color::before {
    	color: #67D972;
    }
    .unusual-color::before {
    	color: #E88282;
    }
    .failure-color::before {
    	color: #F2B354;
    }
    .delete-color::before{
    	color:#CFCFCF;
    }
    .no-padding .slide-content {
        padding: 0px;
    }
    .apn-list {
        padding: 10px;
    }
    .apn-list div {
        margin-top: 5px;
    }
    .apn-title {
        font-weight: bold;
    }
    .imsi-info {
        display: flex;
        padding: 20px 20px;
    }
    .imsi-info .el-form-item  {
        margin-bottom: 0px;
    }
    .imsi-info .el-form-item__label  {
        font-size: 12px;
        line-height: 30px;
    }
    .imsi-info-item {
        margin-left: 20px;
        padding: 15px 20px;
        border: 1px solid #CFCFCF;
        border-top: 2px solid #4d84ff;
    }
    .active-status-div {
        display: flex;
        align-items: center
    }
    .active-status-div > i {
        margin-right: 5px;
        font-size: 20px;
    }
    .active-status-div .success-status::before {
        color: #67D972;
    }

    .circle-cls {
        background-color: #4d84ff;
        border-radius: 20px;
        padding: 5px;
    }
    .circle-cls::before {
        color: #fff;
        font-size: 15px;
    }

    .switch-ctn {
        display: inline-block;
        position: relative;
    }
    .switch-ctn::before {
        content: '';
        position: absolute;
        top: 0;
        right: 0;
        bottom: 0;
        left: 0;
        z-index: 100;
    }

    .mini-icon::before {
        font-size: 14px;
        margin-right: 5px;
    }
    .el-button--primary .mini-icon::before {
        color: #fff;
    }

    .flex-form {
        display: flex;
        flex-wrap: wrap;
    }
    .flex-form .el-form-item__label {
        line-height: 30px;
    }
    .flex-form.half .el-form-item {
        width: 45%;
    }
    .flex-form.enb {
    	padding-top: 20px;
    	border-bottom: 1px solid #e8e8e8;
    }
    .auto-width .el-input {
        width: auto;
    }
    .group-bg-color {
        /*background-color: #F6F7FB;*/
        padding-top: 20px;
        border: 1px solid #E8E8E8;
    }

    .apn-default {
        height: 14px;
        padding: 1px 5px 2px 5px;
        border: 1px solid #F4BF6D;
        color: #F4BF6D;
        border-radius: 3px;
        margin-left: 35px;
        margin-top: 3px;
    }

    .readonly-cls input[readonly] {
        background-color: #F8F8F8 !important;
    }

    .split-title {
        padding: 15px 0 0 0;
        position: relative;
        display: flex;
        align-items: center;
        font-weight: bold;
    }
    .split-title i {
        padding: 0px 10px;
    }
    .split-title::after {
        padding: 0px;
        content: '';
        display: inline-block;
        height: 1px;
        flex: auto;
        width: 99%;
        background-color: #E8E8E8 !important;
        overflow: hidden;
    }
    .view-pane {
        top: -20px;
        padding: 10px 20px;
        position: absolute;
        width: 600px;
        background: #fff;
        z-index: 1;
        box-shadow: 2px 5px 10px #CFCFCF;
        border-radius: 3px;
        border: 1px solid #E8E8E8;
        border-top: 2px solid #4d84ff;
    }
    .view-title {
        font-weight: bold;
        font-size: 14px;
    }
    .view-item {
        display: flex;
        padding-left: 30px;
    }
    .view-item > div {
        flex: 1 auto;
        width: 45%;
    }

    .apn-cls {
        padding: 1px 2px;
        margin-right: 10px;
        position: relative;
        display: inline-block;
        border-radius: 15px;
        background-color: #4d84ff;
    }
    .apn-cls::before {
        content: 'APN';
        display: inline-block;
        font-size: 12px;
        font-weight: normal;
        color: #fff;
        height: 25px;
    }
    .el-tooltip__popper.is-dark { margin: 0 30px 0 80px; }
    .tips-table td {
        padding: 5px 0;
    }
	#imsi_ctn .halobTableBox .el-ctable-toolbar{
		padding: 0px!important;
	}
    #imsi_ctn .toolbarHeadBtnCls{
        position: absolute;
        right: 20px;
        top: 5px;
        display: flex;
    }
    #imsi_ctn .toolbarHeadBtnCls > div{
        position: relative;
        margin-left: 10px;
    }
</style>
<div class="overflow-cls">
<div id="imsi_ctn" style="height: 100%;display: flex;min-width: 1000px;position: relative;width: 100%;">
    <div style="height: 100%; width: 100%;" class="halobTableBox">
        <el-ctable v-show="tabStatus == 'eNB'" ref="halob" id="halob_maintain_list"
            :url="halobURL"
            :time="6"
            :query-params="halobParams">
            <template slot="toolbar">
                <div style="position:relative;">
                    <div class="toolbarHeadBtnBoxCls" style="height: 36px;">
                        <h3 style="padding-left: 10px;">HaloB</h3>
                        <div class="toolbarHeadBtnCls">
                            <div class="newIconBoxCls-bt" @click="toIMEIConfig" tip="IMEI">
                                <span class="el-icon el-icon-circle-setting" ></span>
                            </div>
                            <div class="newIconBoxCls-bt" @click="toIMSIAllocated" tip='IMSI&APN'>
                                <span class="el-icon el-icon-operation-synchronize"></span>
                            </div>
                            <div v-if="isWritable && halobEnable" class="newIconBoxCls-bt" @click="apnSetting" tip='<%=rb.getString("APNPeiZhi") %>'>
                                <span class="el-icon el-icon-operation-settings"></span>
                            </div>
                            <div class="newIconBoxCls-bt" @click="exportHalob" tip='<%=rb.getString("DaoChu") %>'>
                                <span class="el-icon el-icon-operation-export"></span>
                            </div>
                        </div>
                    </div>
                    <div id="tableHeadQuery" class="tableHeadQueryBoxCls">
                        <el-radio-group size="mini" v-model='tabStatus' class="commonRadioButton" @change="tabStatusChange">
                            <el-radio-button v-if="netType == 'eNB'" label="eNB">eNB</el-radio-button>
                            <el-radio-button v-if="netType == 'gNB'" label="gNB">gNB</el-radio-button>
                            <el-radio-button v-if="netType == 'CPE'" label="CPE">CPE</el-radio-button>
                        </el-radio-group>
                        <div class="queryGroup" style="margin:0px 10px;">
                            <el-input v-model="enb_search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
                            <i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
                        </div>
                        <div v-for="(item,index) in advancedQueryItemList">
                            <div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
                                <el-popfilter
                                    :label='item.label'
                                    v-model="item.checkedItemList"
                                    :list="item.options"
                                    :visible.sync="item.isShow"
                                    @check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
                                </el-popfilter>
                            </div>
                            <div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
                                <el-popfilter
                                    type="single"
                                    :label='item.label'
                                    v-model="item.selectVal"
                                    :list="item.options"
                                    :visible.sync="item.isShow"
                                    @check-change="advanceQuery(item.type,item.value,item.selectVal)">
                                </el-popfilter>
                            </div>
                        </div>
                        <div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
                            <%=rb.getString("QingKongShaiXuan")%>
                        </div>
                    </div>
                </div>
            </template>

            <el-table-column v-if="isWritable" width="40" key="enbOp">
                <template slot-scope="scope">
                    <div class="el-icon el-icon-operation-more" @click="halobOptClick(scope.row,event)" v-clickoutside="halobHanderClose"></div>
                </template>
            </el-table-column>
            <el-table-column prop="connectionStatus" width="40">
                <template slot-scope="scope">
                    <div v-html="connStatusFmt(scope.row, scope.row['connectionStatus'], scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column label="<%=rb.getString("XiaoZhanXuLieHao")%>" prop="serialNumber"></el-table-column>
            <el-table-column label="<%=rb.getString("HostName")%>" prop="hostName">
            	<template slot-scope="scope">
            		<div v-if="scope.row.deviceNameTip !== '1'">{{scope.row.hostName}}</div>
            		<div v-if="scope.row.deviceNameTip == '1'">{{scope.row.reportHostName}}</div>
            	</template>
            </el-table-column>
            <el-table-column label="ECI" prop="cellIdentity"></el-table-column>
            <el-table-column label="<%=rb.getString("HaloBKaiGuan")%>" prop="halobEnable">
                <template slot-scope="scope">
                    <div v-if="['0','1'].includes(scope.row.halobEnable) && scope.row.connectionStatus == 1 && isWritable" class="switch-ctn"
                         @click="halobEnableChange(scope.row.halobEnable, scope.row, event)">
                        <el-switch v-model="scope.row.halobEnable" active-value="1" inactive-value="0" style="zoom: 0.8" :disabled="!imsiEnable"></el-switch>
                    </div>
                    
                    <div v-if="['0','1'].includes(scope.row.halobEnable) && scope.row.connectionStatus == 1 && !isWritable" class="switch-ctn">
                        <el-switch v-model="scope.row.halobEnable" active-value="1" inactive-value="0" style="zoom: 0.8" disabled></el-switch>
                    </div>
                    
                    <div v-if="['0','1'].includes(scope.row.halobEnable) && scope.row.connectionStatus != 1" class="switch-ctn">
                        <el-switch v-model="scope.row.halobEnable" active-value="1" inactive-value="0" style="zoom: 0.8" disabled></el-switch>
                    </div>
                    <div v-if="!['0','1'].includes(scope.row.halobEnable)">--</div>
                </template>
            </el-table-column>
            <el-table-column label="IMSI" prop="imsiInfos">
                <template slot-scope="scope">
                    <el-popover v-if="scope.row.imsiInfos && scope.row.imsiInfos.length">
                       <span slot="reference">
                           {{scope.row.imsiInfos[0].imsi}}
                           <span v-if="scope.row.imsiInfos.length>=1" @click="toImsiList(scope.row)">
                               [ <span style="color:#4d84ff;">{{scope.row.imsiInfos.length}}</span>
                               <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;color:#4d84ff;"></i>]
                           </span>
                       </span>
                    </el-popover>
                </template>
            </el-table-column>
            <el-table-column label="APN" prop="apnInfos">
                <template slot-scope="scope">
                    <el-popover v-if="scope.row.apnInfos && scope.row.apnInfos.length">
                        <span slot="reference">
                            {{scope.row.apnInfos[0].apnName}}
                            <span v-if="scope.row.apnInfos.length>1">
                                [ <span style="color:#4d84ff;">{{scope.row.apnInfos.length}}</span>
                                <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;color:#4d84ff;"></i>]
                            </span>
                        </span>
                        <div class="apn-list">
                            <div v-for="(item,index) in scope.row.apnInfos">
                                APN{{item.apnOrder}} APN Name:  {{item.apnName}}
                            </div>
                        </div>
                    </el-popover>
                </template>
            </el-table-column>
            <el-table-column label="Configuration Mode" prop="configMode">
                <template slot-scope="scope">
                    <div v-if="scope.row.configMode != '1'">Auto</div>
                    <div v-if="scope.row.configMode == '1'">Manual</div>
                </template>
            </el-table-column>
            <el-table-column label="APN Status" prop="status">
                <template slot-scope="scope">
                    <div v-if="scope.row.status == '1'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/monitor-syning.gif" style="margin-right: 10px;"/> <%=rb.getString("JinXingZhong")%>
                    </div>
                    <div v-if="scope.row.status == '2' && scope.row.result == '1'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/tnsuccessomc.png" style="margin-right: 10px;"/> <%=rb.getString("ChengGong")%>
                    </div>
                    <div v-if="scope.row.status == '2' && scope.row.result == '0'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/monitor-nosynomc.png" style="margin-right: 10px;"/> <%=rb.getString("ShiBai")%>
                    </div>
                </template>
            </el-table-column>
            <el-table-column label="APN Sync Time" prop="operationTime"></el-table-column>
            <el-table-column label="Failure Reason" prop="failureReason" width="150"></el-table-column>
        </el-ctable>

        <el-ctable v-show="tabStatus == 'gNB'" ref="gnbHalob" id="halob_maintain_gnb_list"
            :url="gnbAPNURL"
            :time="6"
            :query-params="gnbParams">
            <template slot="toolbar">
                <div style="position:relative;">
                    <div class="toolbarHeadBtnBoxCls" style="height: 36px;">
                        <h3 style="padding-left: 10px;">HaloB</h3>
                        <div class="toolbarHeadBtnCls">
                            <div class="newIconBoxCls-bt" @click="toIMEIConfig" tip="IMEI">
                                <span class="el-icon el-icon-circle-setting" ></span>
                            </div>
                            <div class="newIconBoxCls-bt" @click="toIMSIAllocated" tip='IMSI&APN'>
                                <span class="el-icon el-icon-operation-synchronize"></span>
                            </div>
                            <div v-if="isWritable && halobEnable" class="newIconBoxCls-bt" @click="apnSetting" tip='<%=rb.getString("APNPeiZhi") %>'>
                                <span class="el-icon el-icon-operation-settings"></span>
                            </div>
                            <div class="newIconBoxCls-bt" @click="exportHalob" tip='<%=rb.getString("DaoChu") %>'>
                                <span class="el-icon el-icon-operation-export"></span>
                            </div>
                        </div>
                    </div>
                    <div id="tableHeadQuery" class="tableHeadQueryBoxCls">
                        <el-radio-group size="mini" v-model='tabStatus' class="commonRadioButton" @change="tabStatusChange">
                            <el-radio-button v-if="netType == 'eNB'" label="eNB">eNB</el-radio-button>
                            <el-radio-button v-if="netType == 'gNB'" label="gNB">gNB</el-radio-button>
                            <el-radio-button v-if="netType == 'CPE'" label="CPE">CPE</el-radio-button>
                        </el-radio-group>
                        <div class="queryGroup" style="margin:0px 10px;">
                            <el-input v-model="gnb_search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
                            <i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
                        </div>
                        <div v-for="(item,index) in advancedQueryItemList">
                            <div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
                                <el-popfilter
                                    :label='item.label'
                                    v-model="item.checkedItemList"
                                    :list="item.options"
                                    :visible.sync="item.isShow"
                                    @check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
                                </el-popfilter>
                            </div>
                            <div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
                                <el-popfilter
                                    type="single"
                                    :label='item.label'
                                    v-model="item.selectVal"
                                    :list="item.options"
                                    :visible.sync="item.isShow"
                                    @check-change="advanceQuery(item.type,item.value,item.selectVal)">
                                </el-popfilter>
                            </div>
                        </div>
                        <div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
                            <%=rb.getString("QingKongShaiXuan")%>
                        </div>
                    </div>
                </div>
            </template>

            <el-table-column v-if="isWritable" width="40" key="enbOp">
                <template slot-scope="scope">
                    <div class="el-icon el-icon-operation-more" @click="halobOptClick(scope.row,event)" v-clickoutside="halobHanderClose"></div>
                </template>
            </el-table-column>
            <el-table-column prop="connectionStatus" width="40">
                <template slot-scope="scope">
                    <div v-html="connStatusFmt(scope.row, scope.row['connectionStatus'], scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column label="<%=rb.getString("XiaoZhanXuLieHao")%>" prop="serialNumber"></el-table-column>
            <el-table-column label="<%=rb.getString("HostName")%>" prop="hostName">
            	<template slot-scope="scope">
            		<div v-if="scope.row.deviceNameTip !== '1'">{{scope.row.hostName}}</div>
            		<div v-if="scope.row.deviceNameTip == '1'">{{scope.row.reportHostName}}</div>
            	</template>
            </el-table-column>
            <el-table-column label="ECI" prop="cellIdentity"></el-table-column>
            <el-table-column label="<%=rb.getString("HaloBKaiGuan")%>" prop="halobEnable">
                <template slot-scope="scope">
                    <div v-if="['0','1'].includes(scope.row.halobEnable) && scope.row.connectionStatus == 1 && isWritable" class="switch-ctn"
                         @click="halobEnableChange(scope.row.halobEnable, scope.row, event)">
                        <el-switch v-model="scope.row.halobEnable" active-value="1" inactive-value="0" style="zoom: 0.8" :disabled="!imsiEnable"></el-switch>
                    </div>
                    
                    <div v-if="['0','1'].includes(scope.row.halobEnable) && scope.row.connectionStatus == 1 && !isWritable" class="switch-ctn">
                        <el-switch v-model="scope.row.halobEnable" active-value="1" inactive-value="0" style="zoom: 0.8" disabled></el-switch>
                    </div>
                    
                    <div v-if="['0','1'].includes(scope.row.halobEnable) && scope.row.connectionStatus != 1" class="switch-ctn">
                        <el-switch v-model="scope.row.halobEnable" active-value="1" inactive-value="0" style="zoom: 0.8" disabled></el-switch>
                    </div>
                    <div v-if="!['0','1'].includes(scope.row.halobEnable)">--</div>
                </template>
            </el-table-column>
            <el-table-column label="IMSI" prop="imsiInfos">
                <template slot-scope="scope">
                    <el-popover v-if="scope.row.imsiInfos && scope.row.imsiInfos.length">
                       <span slot="reference">
                           {{scope.row.imsiInfos[0].imsi}}
                           <span v-if="scope.row.imsiInfos.length>=1" @click="toImsiList(scope.row)">
                               [ <span style="color:#4d84ff;">{{scope.row.imsiInfos.length}}</span>
                               <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;color:#4d84ff;"></i>]
                           </span>
                       </span>
                    </el-popover>
                </template>
            </el-table-column>
            <el-table-column label="APN" prop="apnInfos">
                <template slot-scope="scope">
                    <el-popover v-if="scope.row.apnInfos && scope.row.apnInfos.length">
                        <span slot="reference">
                            {{scope.row.apnInfos[0].apnName}}
                            <span v-if="scope.row.apnInfos.length>1">
                                [ <span style="color:#4d84ff;">{{scope.row.apnInfos.length}}</span>
                                <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;color:#4d84ff;"></i>]
                            </span>
                        </span>
                        <div class="apn-list">
                            <div v-for="(item,index) in scope.row.apnInfos">
                                APN{{item.apnOrder}} APN Name:  {{item.apnName}}
                            </div>
                        </div>
                    </el-popover>
                </template>
            </el-table-column>
            <el-table-column label="Configuration Mode" prop="configMode">
                <template slot-scope="scope">
                    <div v-if="scope.row.configMode != '1'">Auto</div>
                    <div v-if="scope.row.configMode == '1'">Manual</div>
                </template>
            </el-table-column>
            <el-table-column label="APN Status" prop="status">
                <template slot-scope="scope">
                    <div v-if="scope.row.status == '1'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/monitor-syning.gif" style="margin-right: 10px;"/> <%=rb.getString("JinXingZhong")%>
                    </div>
                    <div v-if="scope.row.status == '2' && scope.row.result == '1'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/tnsuccessomc.png" style="margin-right: 10px;"/> <%=rb.getString("ChengGong")%>
                    </div>
                    <div v-if="scope.row.status == '2' && scope.row.result == '0'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/monitor-nosynomc.png" style="margin-right: 10px;"/> <%=rb.getString("ShiBai")%>
                    </div>
                </template>
            </el-table-column>
            <el-table-column label="APN Sync Time" prop="operationTime"></el-table-column>
            <el-table-column label="Failure Reason" prop="failureReason" width="150"></el-table-column>
        </el-ctable>

        <el-ctable 
            v-show="tabStatus == 'CPE'" ref="cpeHalob" id="halob_maintain_cpe_list"
            :time="6"
            :url="cpeApnUrl"
            :query-params="cpeApnParams"
        >
            <template slot="toolbar">
				 <div style="position:relative;">
                    <div class="toolbarHeadBtnBoxCls" style="height: 36px;">
                        <h3 style="padding-left: 10px;">HaloB Maintain</h3>
                        <div class="toolbarHeadBtnCls">
                            <div class="newIconBoxCls-bt" @click="toIMSIAllocated" tip='IMSI&APN'>
                                <span class="el-icon el-icon-operation-synchronize"></span>
                            </div>
                            <div v-if="isWritable && halobEnable" class="newIconBoxCls-bt" @click="apnSetting" tip='<%=rb.getString("APNPeiZhi") %>'>
                                <span class="el-icon el-icon-operation-settings"></span>
                            </div>
                            <div class="newIconBoxCls-bt" @click="exportHalob" tip='<%=rb.getString("DaoChu") %>'>
                                <span class="el-icon el-icon-operation-export"></span>
                            </div>
                        </div>
                    </div>
                    <div id="tableHeadQuery" class="tableHeadQueryBoxCls">
                        <el-radio-group size="mini" v-model='tabStatus' class="commonRadioButton" @change="tabStatusChange">
                            <el-radio-button v-if="netType == 'eNB'" label="eNB">eNB</el-radio-button>
                            <el-radio-button v-if="netType == 'gNB'" label="gNB">gNB</el-radio-button>
                            <el-radio-button v-if="netType == 'CPE'" label="CPE">CPE</el-radio-button>
                        </el-radio-group>
                        <div class="queryGroup" style="margin:0px 10px;">
                            <el-input v-model="cpe_search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
                            <i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
                        </div>
                        <div v-for="(item,index) in advancedQueryItemList">
                            <div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
                                <el-popfilter
                                    :label='item.label'
                                    v-model="item.checkedItemList"
                                    :list="item.options"
                                    :visible.sync="item.isShow"
                                    @check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
                                </el-popfilter>
                            </div>
                            <div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
                                <el-popfilter
                                    type="single"
                                    :label='item.label'
                                    v-model="item.selectVal"
                                    :list="item.options"
                                    :visible.sync="item.isShow"
                                    @check-change="advanceQuery(item.type,item.value,item.selectVal)">
                                </el-popfilter>
                            </div>
                        </div>
                        <div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
                            <%=rb.getString("QingKongShaiXuan")%>
                        </div>
                    </div>
                </div>
            </template>

            <el-table-column v-if="isWritable && halobEnable" prop="op" width="40" key="cpeOP">
                <template slot-scope="scope">
                    <i v-if="scope.row.write == 1" class="el-icon el-icon-operation-edit" @click="showCpeSync(scope.row)"></i>
                    <i v-if="scope.row.write != 1" class="el-icon el-icon-operation-edit disabled"></i>
                </template>
            </el-table-column>
            <el-table-column prop="connectionStatus" key="connectionStatus" width="40">
                <template slot-scope="scope">
                    <div v-html="connStatusFmt(scope.row, scope.row['connectionStatus'], scope.$index)"></div>
                </template>
            </el-table-column>
            <el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="serialNumber"></el-table-column>
            <el-table-column label="<%=rb.getString("CPEName")%>" prop="cpeName"></el-table-column>
            <el-table-column label="MAC" prop="mac"></el-table-column>
            <el-table-column label="IMSI" prop="imsi"></el-table-column>
            <el-table-column label="APN" prop="apnConfig">
                <template slot-scope="scope">
                    <el-popover v-if="scope.row.apnConfig && scope.row.apnConfig.length">
                        <span slot="reference">
                            {{scope.row.apnConfig[0].apn_name}}
                            <span v-if="scope.row.apnConfig.length>1">
                                [ <span style="color:#4d84ff;">{{scope.row.apnConfig.length}}</span>
                                <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;color:#4d84ff;"></i>]
                            </span>
                        </span>
                        <div class="apn-list">
                            <div v-for="(item,index) in scope.row.apnConfig">
                                APN{{index+1}} APN Name:  {{item.apn_name}}
                            </div>
                        </div>
                    </el-popover>
                </template>
            </el-table-column>
            <el-table-column label="Configuration Mode" prop="configMode">
                <template slot-scope="scope">
                    <div v-if="scope.row.configMode != '1'">Auto</div>
                    <div v-if="scope.row.configMode == '1'">Manual</div>
                </template>
            </el-table-column>
            <el-table-column label="APN Status" prop="apnSyncStatus">
                <template slot-scope="scope">
                    <div v-if="scope.row.apnSyncStatus == '2'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/monitor-syning.gif" style="margin-right: 10px;"/> <%=rb.getString("JinXingZhong")%>
                    </div>
                    <div v-if="scope.row.apnSyncStatus == '1'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/tnsuccessomc.png" style="margin-right: 10px;"/> <%=rb.getString("ChengGong")%>
                    </div>
                    <div v-if="scope.row.apnSyncStatus == '0'" class="active-status-div">
                        <img src="${ctx}/css/images/main/monitor-ico/monitor-nosynomc.png" style="margin-right: 10px;"/> <%=rb.getString("ShiBai")%>
                    </div>
                </template>
            </el-table-column>
            <el-table-column label="APN Sync Time" prop="apnSyncTime"></el-table-column>
            <el-table-column label="Failure Reason" prop="failureReason" width="150"></el-table-column>
        </el-ctable>

        <el-cmenu ref="halobmenu" :data="halobmenus" @click="halobClickEvent"></el-cmenu>
    </div>

    <el-slide ref="apn" class="no-padding"
        method="get"
        :title="apnslide.title"
        :url="apnslide.url"
        @ok="saveApn"
        @cancel="closeApn">
    </el-slide>
    <el-slide ref="imsilist" class="no-padding"
        method="get"
        :header="imsislide.header"
        :footer="false"
        :title="imsislide.title"
        :url="imsislide.url"
        @cancel="closeIMSI">
    </el-slide>

    <el-slide ref="imeilist" class="no-padding"
        method="get"
        :header="imeislide.header"
        :footer="false"
        :title="imeislide.title"
        :url="imeislide.url"
        @cancel="closeIMEI">
    </el-slide>

    <el-dialog title="<%=rb.getString("QueRen")%>" width="500"
        :visible.sync="dialogVisible">
        <p v-if="enableFlag" style="margin-bottom: 15px;"><%=rb.getString("QueRenGuanBiHaloB")%></p>
        <p v-else style="margin-bottom: 15px;"><%=rb.getString("QueRenKaiQiHaloB")%></p>
        <el-checkbox v-model="reboot" true-label="true" false-label="false"></el-checkbox> <%=rb.getString("SheZhiHouChongQi")%>

        <span slot="footer">
            <el-button type="primary" @click="halobConfirm"><%=rb.getString("Shi")%></el-button>
            <el-button @click="halobCancel"><%=rb.getString("Fou")%></el-button>
        </span>
    </el-dialog>

    <el-dialog title="<%=rb.getString("XiuGai")%>" width="960" :visible.sync="syncDLVisible">
        <el-form ref="enbApnForm" :model="syncForm" :rules="enbRules" label-width="100" style="max-height: 400px;overflow: auto;">
            <div class="group-bg-color flex-form half" style="border: none;">
                <el-form-item label="Configuration mode" prop="configMode" label-width="150">
                    <el-radio-group v-model="syncForm.configMode" style="padding-top: 8px;" @change="enbModeChange">
                        <el-radio label="0">Automatic</el-radio>
                        <el-radio label="1">Manual</el-radio>
                    </el-radio-group>
                </el-form-item>
	            <el-form-item label="LGW Mode">
	                <el-select v-model="syncForm.lgwMode"  :disabled="syncForm.configMode == '0'">
	                    <el-option label="NAT" value="0" :disabled="isBridge"></el-option>
	                    <el-option label="Bridge" value="2"></el-option>
	                </el-select>
	            </el-form-item>
	        	<el-form-item label="MTU" label-width="150" prop="mtu">
	        		<el-input v-model="syncForm.mtu" :disabled="isEnbAuto" placeholder="Range: 700 - 1600"></el-input>
	        	</el-form-item>
            </div>
            <div class="group-bg-color" style="padding-top: 0px;width: 99%;border-bottom: none;">
                <div v-for="(row, idx) in syncForm.enbApnConfig" class="flex-form enb">
                    <span v-if="false" style="padding-left: 15px;padding-top: 6px;font-weight: bold;">APN{{row.apnOrder}}</span>

                    <el-form-item label="APN Name">
                        <div style="display: flex;" class="readonly-cls">
                            <el-input v-model="row.apnName" style="width: 150px;" readonly></el-input>
                            <el-button v-if="false" @click="viewInfo(row,idx)" style="min-width: 40px;padding: 0 5px;margin-left: 10px;border:1px solid #4d84ff;">
                                <i class="el-icon el-icon-operation-result mini-icon"></i>View
                            </el-button>

                            <div v-if="viewIndex == idx && viewShow" class="view-pane">
                                <div style="position: absolute;right: 10px;top: 5px;">
                                    <i @click="viewShow = false" class="el-icon el-icon-close"></i>
                                </div>
                                <div class="view-title"><i class="apn-cls"></i> APN Name: {{row.apnName}}</div>
                                <div v-if="false" class="view-item">
                                    <div>Index: APN{{idx+1}}</div>
                                    <div>IPv4: 
                                    	<el-popover v-if="viewForm.IPPOOL_INFO && viewForm.IPPOOL_INFO.length">
					                        <span slot="reference">
					                            {{viewForm.IPPOOL_INFO[0].START_SERVED_PARTY_IPV4_ADDRESS}} -- {{viewForm.IPPOOL_INFO[0].END_SERVED_PARTY_IPV4_ADDRESS}}
					                            <span v-if="viewForm.IPPOOL_INFO.length>1">
					                                [ <span style="color:#4d84ff;">{{viewForm.IPPOOL_INFO.length}}</span>
					                                <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;color:#4d84ff;"></i>]
					                            </span>
					                        </span>
					                        <div class="apn-list">
					                            <div v-for="(item,index) in viewForm.IPPOOL_INFO">
					                                {{item.START_SERVED_PARTY_IPV4_ADDRESS}} -- {{item.END_SERVED_PARTY_IPV4_ADDRESS}}
					                            </div>
					                        </div>
					                    </el-popover>
                                    </div>
                                </div>
                                <div class="view-item">
                                    <div>APN Downlink Limit: {{viewForm.APN_AMBR_DL}}{{viewForm.APN_AMBR_DL?'Mbps':''}}</div>
                                    <div>APN Uplink Limit: {{viewForm.APN_AMBR_UL}}{{viewForm.APN_AMBR_DL?'Mbps':''}}</div>
                                </div>
                                <div class="view-item">
                                    <div>Primary DNS IP: {{viewForm.PRIMARY_DNS_IPADDR}}</div>
                                    <div>Secondary DNS IP: {{viewForm.SECONDARY_DNS_IPADDR}}</div>
                                </div>
                                <div class="view-item">
                                    <div>QCI: {{viewForm.QCI}}</div>
                                    <div>ARP Priority Level: {{viewForm.ARP_PRIORITYLEVEL}}</div>
                                </div>
                                <div class="view-item">
                                    <div>ARP PCI: {{viewForm.ARP_PCI == '1'?'Open':'Close'}}</div>
                                    <div>ARP PVI: {{viewForm.ARP_PVI == '1'?'Open':'Close'}}</div>
                                </div>
                            </div>
                        </div>
                    </el-form-item>

                    <el-form-item label="APN Type" label-width="140">
                        <el-select v-model="row.apnTypeEnb" :disabled="isEnbAuto" class="auto-width" style="width: 130px;" 
                        	@change="function(value){
                                   if(value == 'tunnel') {
                                        row.apnType = '2';
                                        if(row.vlanId) {
                                            //var list = row.vlanId.split(',');

                                           // row.untaggedVlan = list[0];
                                        }
                                        
                                        row.untaggedVlan = '';
                                        row.vlanId = row.taggedVlan;
                                   }else if(row.apnType == '2'){
                                   	    row.apnType = '1';
                                   }
                                   linkTunnel();
                               }">
                            <el-option label="Normal" value="normal"></el-option>
                            <el-option label="Tunnel" value="tunnel"></el-option>
                        </el-select>
                    </el-form-item>
                    
                    <el-form-item v-show="row.apnType != '2'" :label="['0','1','3'].includes(row.apnType)?'VLAN ID':'VLAN List'" label-width="140"
                        :prop="'enbApnConfig.'+idx+'.vlanId'"
                        :key="row.apnOrder+'_vlanId'"
                        :rules="{
                            validator: function(rule, val, cb){
                                var reg = /^\d{1,}$/,
                                    ids = (val||'').split(','),
                                    bool = true;

                                ids.map(function(item){
                                    var trimVal = item.trim();

                                    if(!reg.test(trimVal)) {
                                        bool = false;
                                    }

                                    if(trimVal<0 || trimVal>4094) {
                                        bool = false;
                                    }
                                });
								
                                if(row.apnType == '2') {// tunnel mode, vlan List hide
                                    cb();
                                }else {
                                    if(val || [0,'0'].includes(val)) {
                                        if(['2'].includes(row.apnType) && (!bool || ids.length>3)) {
                                            if(val) cb('Numbers separated by commas,max 3,range: 0-4094');
                                            else cb();
                                        }else if(['0','1','3'].includes(row.apnType) && (!bool || ids.length>1)) {
                                            if(val) {
                                                cb('Numbers,range: 0-4094');
                                            }else {
                                                var isL3Empty = false,
                                                    l3LanId = syncForm.enbApnConfig.filter(function(m){
                                                        return ['0','1','3'].includes(m.apnType)
                                                    }),
                                                    l3EmptyLanId = syncForm.enbApnConfig.filter(function(m){
                                                        return ['0','1','3'].includes(m.apnType) && m.vlanId.trim() == '';
                                                    });

                                                if(['0','1','3'].includes(row.apnType) && l3EmptyLanId.length == l3LanId.length) {
                                                    cb('VLAN ID of L3 should`t all empty');
                                                }else {
                                                    cb();
                                                }
                                            }
                                        }else {
                                            var enableVlans = syncForm.enbApnConfig.filter(function(m){
                                                    return m.apnOrder != row.apnOrder && m.apnType == '2'; // 是否和tunnel的存在id重复
                                                }),
                                                otherVlans = [],
                                                selfVlans = [],
                                                isRepeated = false;

                                            enableVlans.map(function(o){
                                                o.vlanId.split(',').map(function(m){
                                                    otherVlans.push(m||'0');
                                                });
                                            });

                                            ids.map(function(m){
                                                if(!selfVlans.includes(m)) {
                                                    selfVlans.push(m||'0');
                                                }
                                            });

                                            ids.map(function(m){
                                                if(otherVlans.includes(m||'0')) isRepeated = true;
                                            });

                                            if(ids.length > selfVlans.length) isRepeated = true;

                                            if(isRepeated) {
                                                cb('Vlanid repeated');
                                            }else {
                                                cb();
                                            }
                                        }
                                    }else {
                                        cb();
                                    }
                                }
                            }
                        }">
                        <el-input v-model="row.vlanId" :disabled="isEnbAuto" style="width: 220px;"></el-input>
                    </el-form-item>

                    <div style="margin-left: 100px;width: 98%; display: flex;flex-wrap: wrap;border: 1px solid #E8E8E8;padding-top: 19px;background-color: #f9f9f9;margin-bottom: 10px;">
                       	<el-form-item label="CPE Default Setting" label-width="148"></el-form-item>

                        <el-form-item label="Bearer Type" label-width="140">
                            <el-select v-model="row.apnType" :disabled="isEnbAuto" class="auto-width" style="width: 130px;" 
                            	@change="function(value){
                                       linkTunnel(value);

                                       if(row.apnType != '2' && row.apnType != '3') {
                                       	   row.untaggedVlan = '';
                                       	   row.taggedVlan = '';
                                       }
                                   }">
                                <el-option label="L3-MGMT" value="0" :disabled="row.apnTypeEnb == 'tunnel'"></el-option>
                                <el-option label="L3-NAT" value="1" :disabled="row.apnTypeEnb == 'tunnel'"></el-option>
                                <el-option label="L2-TUNNEL" value="2" :disabled="row.apnTypeEnb != 'tunnel'"></el-option>
                                <el-option label="L2-BRIDGE" value="3" :disabled="row.apnTypeEnb == 'tunnel'"></el-option>
                            </el-select>
                        </el-form-item>
                       	
                       	<el-form-item v-show="row.apnType == '2'" label="Untagged VLAN" label-width="140"
                       		:prop="'enbApnConfig.'+idx+'.untaggedVlan'"
                            :key="row.apnOrder+'_untaggedVlan'"
                            :rules="{
                                validator: function(rule, val, cb){
                                	var enbVlans = (row.vlanId||'').split(','),
                                        enableVlans = syncForm.enbApnConfig.filter(function(m){
                                            return m.apnOrder != row.apnOrder && ['2'].includes(m.apnType);
                                        }),
                                        otherVlans = [],
                                        isRepeated = false;

									enableVlans.map(function(o){
                                        if(o.untaggedVlan || ['0'].includes(o.untaggedVlan)){
                                            otherVlans.push(o.untaggedVlan);
                                        }
                                    })

                                    if(otherVlans.includes(val)) isRepeated = true;

                                    if(row.apnType == '2') {
                                        if(!isNaN(val) && (val-0>=0 && val-4094<=0) || ['','0'].includes(val)) {
                                            if(isRepeated) {
                                                cb('The untagged Vlans in total APNs of CPE cannot be overlap');
                                            }else {
                                                var list = (row.taggedVlan||'').split(',');
                                                
                                                if(val && list.includes(val) && !['0'].includes(val)) {
                                                    cb('Untagged VLAN and tagged VLANs cannot be overlap');
                                                }else {
                                                    cb();
                                                }
                                            }
                                        }else {
                                            cb('Single number is from 0 to 4094');
                                        }
                                    }else {
                                        cb();
                                    }
                                }
                             }">
                       		<el-input v-model="row.untaggedVlan" :disabled="isEnbAuto" style="width: 220px;" @change="function(val){
                                    $refs.enbApnForm.validateField('enbApnConfig.'+idx+'.taggedVlan');
                                    
                                    if(row.apnType == '2') {
                                        var list = [];

                                        if(![null,undefined,''].includes(row.untaggedVlan)) {
                                            row.untaggedVlan.split(',').map(function(m){
                                                if(!list.includes(m)) list.push(m);
                                            });
                                        }
                                        if(![null,undefined,''].includes(row.taggedVlan)) {
                                            row.taggedVlan.split(',').map(function(m){
                                                if(!list.includes(m)) list.push(m);
                                            });
                                        }
                                        row.vlanId = list.join(',');
                                    }
                                }"></el-input>
                       	</el-form-item>
                       	
                       	<el-form-item v-show="row.apnType == '2'" label-width="418"></el-form-item>
                       	
                       	<el-form-item v-show="row.apnType != '0' && row.apnType != '1'" :label="row.apnType == '2'?'Tagged VLANs':'Tagged VLAN'" label-width="140"
                       		:prop="'enbApnConfig.'+idx+'.taggedVlan'"
                            :key="row.apnOrder+'_taggedVlan'"
                            :rules="{
                                validator: function(rule, val, cb){
                                	var reg = /^\d{1,}$/,
                                           enableVlans = syncForm.enbApnConfig.filter(function(m){
                                               return m.apnOrder != row.apnOrder && ['2','3'].includes(m.apnType);
                                           }),
                                           ids = (val||'').split(','),
                                           otherVlans = [],
                                           selfVlans = [],
                                           isRepeated = false,
                                           bool = true;
                                           
                                       if(row.apnType == '0' || row.apnType == '1') {
                                       	cb();
                                       	return;
                                       }

                                       
                                       var hasNat = syncForm.enbApnConfig.filter(function(m){
                                                return m.apnOrder != row.apnOrder && ['1'].includes(m.apnType);
                                           }).length > 0;
	

                                       ids.map(function(item){
                                        var trimVal = item.trim();
                                        
                                        if(trimVal.length && !reg.test(trimVal)) {
                                            bool = false;
                                        }
                                        
                                        if(trimVal<1 || trimVal>4094) {
                                            bool = false;
                                        }
                                        
                                        if(['2','3'].includes(row.apnType) && ['','0'].includes(trimVal)) {
                                            bool = true;

                                            if(row.untaggedVlan - 1 >= 0 && ['0'].includes(trimVal)) {
                                                bool = false;
                                            }
                                            //if(row.untaggedVlan == '' && trimVal == '') {
                                            //    bool = false;
                                            //}
                                            // 包含L3-NAT，则tunnel的untagged vlan和tagged vlan不能同时为空
                                            //if(hasNat && row.apnType == '2' && row.untaggedVlan == '' && (trimVal == '0' || trimVal == '')) {
                                            //    bool = false;
                                            //}
                                        }
                                    });

                                    enableVlans.map(function(o){
                                        (o.taggedVlan||'').split(',').map(function(m){
                                            otherVlans.push(m||'0');
                                        });
                                    });

                                    ids.map(function(m){
                                        if(!selfVlans.includes(m||'0')) {
                                            selfVlans.push(m||'0');
                                        }
                                    });

                                    ids.map(function(m){
                                        if(otherVlans.includes(m||'0') && !['','0'].includes(val)) isRepeated = true;
                                    });

                                    if(ids.length > selfVlans.length && !['','0'].includes(val)) isRepeated = true;

									if(bool) {
                                        if(isRepeated) {
                                            if(row.apnType == '3') {// bridge
                                                cb('The Tagged Vlan in total APNs of CPE cannot be overlap');
                                            }else {
                                                cb('The Tagged Vlans in total APNs of CPE cannot be overlap');
                                            }
                                        }else {
                                               var list = (row.taggedVlan||'').split(',');

                                               if(val && list.includes(row.untaggedVlan) && row.untaggedVlan !== '0') {
                                                   cb('Untagged VLAN and tagged VLANs cannot be overlap except 0');
                                               }else {
                                                   if(row.apnType == '3' && ids.length > 1) {
                                                        cb('Single number is from 0 to 4094');
                                                   //}else if(row.apnType == '2' & ids.length > 2) {
                                                   //     cb('Allows up to 2 comma-separated numbers,range of each number is from 0 to 4094');
                                                   }else {
                                                        if(row.apnType == '2') {
                                                            let tunnelVlans = syncForm.enbApnConfig.filter(function(m){
                                                                    return m.apnOrder != row.apnOrder && ['2'].includes(m.apnType);
                                                                }),
                                                                tunnelIds = [],
                                                                tunnelVlanIdRepeated = false;

                                                            tunnelVlans.map(function(o){
                                                                if(o.vlanId.length) {
                                                                    o.vlanId.split(',').map(function(m){
                                                                        tunnelIds.push(m||'0');
                                                                    });
                                                                }
                                                            });

                                                            list.map(function(m){
                                                                if(tunnelIds.includes(m)) tunnelVlanIdRepeated = true;
                                                            })

                                                            if(tunnelVlanIdRepeated) {
                                                                cb('Tunnel mode Vlan ID cannot be overlap');
                                                            }else {
                                                                cb();
                                                            }
                                                        }else {
                                                            cb();
                                                        }
                                                   }
                                               }
                                        }
									}else {
                                        if(row.apnType == '3') {
                                            if([null,''].includes(val)) {
                                                cb();
                                            }else {
                                                cb('Single number is from 0 to 4094');
                                            }
                                        }else {
                                            if(row.untaggedVlan - 1 >= 0) {
                                                //cb('Allows up to 2 comma-separated numbers,range of each number is from 1 to 4094');
                                                cb('Allows comma-separated numbers,range of each number is from 1 to 4094');
                                            }else {
                                                //cb('Allows up to 2 comma-separated numbers,range of each number is from 0 to 4094');
                                                cb('Allows comma-separated numbers,range of each number is from 0 to 4094');
                                            }
                                        }
									}
                                }
                             }">
                       		<el-input v-model="row.taggedVlan" :disabled="isEnbAuto" style="width: 220px;" @change="function(val){
                                    $refs.enbApnForm.validateField('enbApnConfig.'+idx+'.untaggedVlan');
                                    
                                    if(row.apnType == '2') {
                                        var list = [];

                                        if(![null,undefined,''].includes(row.untaggedVlan)) {
                                            row.untaggedVlan.split(',').map(function(m){
                                                if(!list.includes(m)) list.push(m);
                                            });
                                        }
                                        if(![null,undefined,''].includes(row.taggedVlan)) {
                                            row.taggedVlan.split(',').map(function(m){
                                                if(!list.includes(m)) list.push(m);
                                            });
                                        }
                                        row.vlanId = list.join(',');
                                    }
                                }"></el-input>
                       	</el-form-item>
                    </div>
                </div>
            </div>
            <div class="split-title" v-show="syncForm.l2TunnelEnable == 'true'">
                Advance <i :class="arrowCls" @click="isDown = !isDown"></i>
            </div>
            <div v-show="!isDown && syncForm.l2TunnelEnable == 'true'" class="group-bg-color flex-form half" style="border: none;">
	            <el-form-item v-show="false" label="L2 Tunnel Enable" class="third" label-width="150">
	                <el-switch v-model="syncForm.l2TunnelEnable" :disabled="isEnbAuto" active-value="true" inactive-value="false" style="zoom: 0.8"></el-switch>
	            </el-form-item>
                <el-form-item label="L2 Server IP" prop="l2ServerIp">
                    <el-input v-model="syncForm.l2ServerIp" :disabled="isEnbAuto"></el-input>
                </el-form-item>
            </div>
        </el-form>

        <span slot="footer">
            <el-button type="primary" @click="showSaveDialog('enb')"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="syncDLVisible=false"><%=rb.getString("QuXiao")%></el-button>
        </span>
    </el-dialog>

    <el-dialog title="<%=rb.getString("XiuGai")%>" width="1400" :visible.sync="syncCpeDLVisible">
        <el-form ref="cpeApnForm" :rules="cpeRules" :model="syncCpeForm" label-width="100">
            <div class="group-bg-color flex-form half" style="border: none;flex-wrap: wrap;">
                <el-form-item label="Operation Mode" label-width="130">
                    <el-select v-model="syncCpeForm.cpeL2TunnelMode" :disabled="isCpeAuto" @change="function(value){
                            if(value == '0') {
                                syncCpeForm.cpeApnConfig[2].bearType = '3';
                                syncCpeForm.cpeApnConfig[3].bearType = '3';
                            }
                        }">
                        <el-option label="NAT" value="0"></el-option>
                        <el-option label="Tunnel" value="2"></el-option>
                        <el-option label="Bridge" value="3"></el-option>
                    </el-select>
                </el-form-item>

                <el-form-item v-show="false" label="GRE Type" label-width="120">
                    <el-select v-model="syncCpeForm.greType" :disabled="isCpeAuto" class="auto-width" style="width: 90px;">
                        <el-option label="Layer2" value="1"></el-option>
                    </el-select>
                </el-form-item>

                <el-form-item label="Configuration mode" prop="configMode" label-width="150">
                    <el-radio-group v-model="syncCpeForm.configMode" style="padding-top: 8px;" @change="cpeModeChange">
                        <el-radio label="0">Automatic</el-radio>
                        <el-radio label="1">Manual</el-radio>
                    </el-radio-group>
                </el-form-item>
	        	<el-form-item label="MTU" label-width="130" prop="mtu">
	        		<el-input v-model="syncCpeForm.mtu" :disabled="isCpeAuto" placeholder="Range: 700 - 1600"></el-input>
	        	</el-form-item>
            </div>
            <div class="group-bg-color">
                <div v-for="(row, idx) in syncCpeForm.cpeApnConfig" class="flex-form" v-show="!(row.apnName == '' && syncCpeForm.configMode=='0')">
                    <span v-if="false" style="padding-left: 15px;padding-top: 6px;font-weight: bold;">APN{{row.apnOrder}}</span>

                    <el-form-item label="APN Name">
                        <div style="display: flex;" class="readonly-cls">
                            <el-input v-model="row.apnName" style="width: 120px;" readonly></el-input>
                            <el-button v-if="false" @click="viewInfo(row,idx)" style="min-width: 40px;padding: 0 5px;margin-left: 10px;border:1px solid #4d84ff;">
                                <i class="el-icon el-icon-operation-result mini-icon"></i>View
                            </el-button>

                            <div v-if="viewIndex == idx && viewShow" class="view-pane">
                                <div style="position: absolute;right: 10px;top: 5px;">
                                    <i @click="viewShow = false" class="el-icon el-icon-close"></i>
                                </div>
                                <div class="view-title"><i class="apn-cls"></i> APN Name: {{row.apnName}}</div>
                                <div v-show="false" class="view-item">
                                    <div>Index: APN{{idx+1}}</div>
                                    <div>IPv4: {{viewForm.GW_IP_ADDRESS}}</div>
                                </div>
                                <div class="view-item">
                                    <div>APN Downlink Limit: {{viewForm.APN_AMBR_DL}}{{viewForm.APN_AMBR_DL?'Mbps':''}}</div>
                                    <div>APN Uplink Limit: {{viewForm.APN_AMBR_UL}}{{viewForm.APN_AMBR_UL?'Mbps':''}}</div>
                                </div>
                                <div class="view-item">
                                    <div>Primary DNS IP: {{viewForm.PRIMARY_DNS_IPADDR}}</div>
                                    <div>Secondary DNS IP: {{viewForm.SECONDARY_DNS_IPADDR}}</div>
                                </div>
                                <div class="view-item">
                                    <div>QCI: {{viewForm.QCI}}</div>
                                    <div>ARP Priority Level: {{viewForm.ARP_PRIORITYLEVEL}}</div>
                                </div>
                                <div class="view-item">
                                    <div>ARP PCI: {{viewForm.ARP_PCI}}</div>
                                    <div>ARP PVI: {{viewForm.ARP_PVI}}</div>
                                </div>
                            </div>
                        </div>
                    </el-form-item>

                    <el-form-item label="Bearer Type"
                    	:prop="'cpeApnConfig.'+idx+'.bearType'"
                        :key="row.apnOrder+'_bearType'"
                        :rules="{
                            validator: function(rule, val, cb){
                                if(idx == 0) {
                                    if(row.bearType == '1') {
                                        var list = syncCpeForm.cpeApnConfig.filter(function(item){
                                                return ![undefined,null,''].includes(item.apnName);
                                            });
                                        // 非bridge 少于2个apn时，第一个可以是Data类型
                                        if(list.length<=1 && syncCpeForm.cpeL2TunnelMode != '3') {
                                            cb(' ');
                                        }else {
                                            cb();
                                        }
                                    }else {
                                        var list = syncCpeForm.cpeApnConfig.filter(function(item){
                                                return ![undefined,null,''].includes(item.apnName);
                                            });
                                        // 非bridge 少于2个apn时，第一个可以是Data类型
                                        if(list.length<=1 && row.bearType == '0' && syncCpeForm.cpeL2TunnelMode != '3') {
                                            cb();
                                        }else {
                                            cb(' ');
                                        }
                                    }
                                }else if(idx == 1) {
                                    if(row.bearType == '0' || [undefined,null,''].includes(row.apnName)) {
                                        cb();
                                    }else {
                                        cb(' ');
                                    }
                                }else {
                                    if(syncCpeForm.cpeL2TunnelMode == '0') {// NAT
                                        if(row.bearType == '3' || [undefined,null,''].includes(row.apnName)) {
                                            cb();
                                        }else {
                                            cb(' ');
                                        }
                                    }else {
                                        cb();
                                    }
                                }
                            }
                        }">
                        <el-select v-model="row.bearType" :disabled="isCpeAuto" class="auto-width" style="width: 120px;" @change="function(value){
                                if(!(!['0'].includes(syncCpeForm.cpeL2TunnelMode) && value == '0')) {
                                    row.taggedVlan = '';
                                }
                                if(!(syncCpeForm.cpeL2TunnelMode == '2' && value != '1')) {
                                	row.untaggedVlan = '';
                                }
                            }">
                            <el-option label="MGMT" value="1" :disabled="mgmtDisabled"></el-option>
                            <el-option label="DATA" value="0" :disabled="[2,3].includes(idx) && syncCpeForm.cpeL2TunnelMode=='0'"></el-option>
                            <el-option label="RESERVED" value="3"></el-option>
                        </el-select>
                    </el-form-item>

                    <el-form-item v-show="false" :label="getLayer(syncCpeForm.cpeL2TunnelMode, row.bearType)" label-width="40" style="width: 20px;"></el-form-item>
                    
                    <el-form-item v-show="syncCpeForm.cpeL2TunnelMode == '2' && row.bearType == '0'" label="Untagged VLAN" label-width="140"
                    	:prop="'cpeApnConfig.'+idx+'.untaggedVlan'"
                        :key="row.apnOrder+'_untaggedVlan'"
                        :rules="{
                            validator: function(rule, val, cb){
                            	var enableVlans = syncCpeForm.cpeApnConfig.filter(function(m){
                                        return !['',null,undefined].includes(m.untaggedVlan) && m.apnOrder != row.apnOrder && !['',null,undefined].includes(m.apnName) && row.bearType == '0';
                                    }),
                                    ids = (val||'').split(','),
                                    otherVlans = [],
                                    selfVlans = [],
                                    isRepeated = false;
                                    
                                enableVlans.map(function(o){
                                    o.untaggedVlan.split(',').map(function(m){
                                    	otherVlans.push(m);
                                    });
                                });

                                ids.map(function(m){
                                    if(!selfVlans.includes(m)) {
                                        selfVlans.push(m);
                                    }
                                });

                                ids.map(function(m){
                                    if(otherVlans.includes(m)) isRepeated = true;
                                });

                                if(ids.length > selfVlans.length) isRepeated = true;
								
                                if(syncCpeForm.cpeL2TunnelMode == '0') {
                                    cb();
                                }else if(val-0>=1 && val-0<=4094 || [0,'0'].includes(val)) {
                                    if(!['',null,undefined].includes(row.apnName)) {// apnName not empoty
                                        if(enableVlans.length) {
                                            cb('One untagged vlan or blank is available in Tunnel Mode');
                                        }else {
                                            var list = (row.taggedVlan||'').split(',');

                                            if(val && list.includes(val) 
                                                && !['','0'].includes(val)
                                                && !['',null,undefined].includes(row.apnName) 
                                                && syncCpeForm.cpeL2TunnelMode == '2' && row.bearType == '0') {
                                                cb('Untagged VLAN and tagged VLANs cannot be overlap');
                                            }else {
                                                var vlanList = row.vlanList? row.vlanList.split(','):[];

                                                if(vlanList.length && !vlanList.includes(val) && ![0,'0'].includes(val)) {
                                                    cb('Only values in VLAN list(' + row.vlanList + ') or 0 is available');
                                                }else {
                                                    cb();
                                                }
                                            }
                                        }
                                    }else {
                                        cb();
                                    }
								}else if(val !== ''){
									cb('Number is from 0 to 4094');
								}else {
									//if((row.taggedVlan||'').length == 0 && syncCpeForm.cpeL2TunnelMode == '2' && !['',null,undefined].includes(row.apnName) && row.bearType == '0') {
									//	cb('Untagged VLAN and tagged VLANs cannot be all blank');
									//}else {
										cb();
									//}
								}
                            }
                        }">
                    	<el-input v-model="row.untaggedVlan" :disabled="isCpeAuto" style="width: 300px;"></el-input>
                    </el-form-item>

                    <el-form-item v-show="!['0'].includes(syncCpeForm.cpeL2TunnelMode) && row.bearType == '0'" :label="syncCpeForm.cpeL2TunnelMode == '3'?'Tagged VLAN':'Tagged VLANs'" label-width="140"
                        :prop="'cpeApnConfig.'+idx+'.taggedVlan'"
                        :key="row.apnOrder+'_taggedVlan'"
                        :rules="{
                            validator: function(rule, val, cb){
                                var reg = /^\d{1,}$/,
                                    ids = (val||'').split(','),
                                    bool = true;

                                ids.map(function(item){
                                    var trimVal = item.trim();

                                    if(trimVal.length && !reg.test(trimVal)) {
                                        bool = false;
                                    }

                                    if((trimVal<1 && !['','0'].includes(trimVal)) || trimVal>4094) {
                                        bool = false;
                                    }
                                    
                                    //if(syncCpeForm.cpeL2TunnelMode == '2' && ['','0'].includes(trimVal)) {
                                    //	bool = false;
                                    //}
                                });
                                
                                if(row.bearType == '1'||['0'].includes(syncCpeForm.cpeL2TunnelMode)) { // MGMT 不校验taggedVlan
                                    cb();
                                }else if([undefined,null,''].includes(row.apnName)) {
                                    if(val && row.bearType == '0' && !bool) {
                                        if(syncCpeForm.cpeL2TunnelMode == '3') {
                                            cb('Single number is from 0 to 4094');
                                        }else if(syncCpeForm.cpeL2TunnelMode == '2'){
                                            //cb('Allows up to 2 comma-separated numbers,range of each number is from 1 to 4094');
                                            cb('Allows comma-separated numbers,range of each number is from 1 to 4094');
                                        }else {
                                            cb();
                                        }
                                    }else {
                                        cb();
                                    }
                                }else {
                                    if(val || ['','0'].includes(val)) {
                                        // 原始：!bool || ids.length>3
                                        if(!bool) {
                                            var list = (row.taggedVlan||'').split(',');

                                            if(syncCpeForm.cpeL2TunnelMode == '3') {
                                                if(list.length>1 || !bool) {
                                                    cb('Number is from 0 to 4094');
                                                }else {
                                                    cb();
                                                }
                                            }else if(val) {
                                            	if(syncCpeForm.cpeL2TunnelMode == '2') {
                                            		//cb('Allows up to 3 comma-separated numbers,range of each number is from 0 to 4094');
                                            		cb('Allows comma-separated numbers,range of each number is from 0 to 4094');
                                            	}
                                            }else {
                                            	if(syncCpeForm.cpeL2TunnelMode == '2' && '' == val) {
                                            		//cb('Allows up to 2 comma-separated numbers,range of each number is from 1 to 4094');
                                            		cb('Allows comma-separated numbers,range of each number is from 1 to 4094');
                                            	}else {
                                            		cb();
                                            	}
                                            }
                                        }else {
                                            var enableVlans = syncCpeForm.cpeApnConfig.filter(function(m){
                                                    return ![null,undefined].includes(m.taggedVlan) && m.apnOrder != row.apnOrder && m.bearType == '0' && !['',null,undefined].includes(m.apnName);
                                                }),
                                                otherVlans = [],
                                                selfVlans = [],
                                                isRepeated = false;

                                            if(row.bearType != '0') {
                                                cb();
                                                return;
                                            }

                                            enableVlans.map(function(o){
                                                o.taggedVlan.split(',').map(function(m){
                                                    otherVlans.push(m||'0');
                                                });
                                            });

                                            ids.map(function(m){
                                                if(!selfVlans.includes(m||'0') || ['','0'].includes(m)) {
                                                    selfVlans.push(m||'0');
                                                }
                                            });
											
                                            ids.map(function(m){
                                                if(otherVlans.includes(m||'0') && !['','0'].includes(m)) isRepeated = true;
                                            });

                                            if(ids.length > selfVlans.length) isRepeated = true;

                                            if(isRepeated) {
                                                cb('The taggedVlans in total APNs of CPE cannot be overlap');
                                            }else {
                                                var list = (row.taggedVlan||'').split(',');

                                                if(val && list.includes(row.untaggedVlan)
                                                    && !['','0'].includes(row.untaggedVlan)
                                                    && !['',null,undefined].includes(row.apnName) 
                                                    && syncCpeForm.cpeL2TunnelMode == '2' && row.bearType == '0') {
                                                    cb('Untagged VLAN and tagged VLANs cannot be overlap');
                                                }else {
                                                    if(syncCpeForm.cpeL2TunnelMode == '3' && list.length>1 ) {
                                                        cb('Single number is from 0 to 4094');
                                                    }else {
                                                        if(syncCpeForm.cpeL2TunnelMode == '2' && ids.length>2) {
                                                            //cb('Allows up to 2 comma-separated numbers,range of each number is from 0 to 4094');
                                                            cb();
                                                        }else {
                                                            cb();
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }else {
                                        if(!['0'].includes(syncCpeForm.cpeL2TunnelMode) && row.bearType == '0' && !['',null,undefined].includes(row.apnName)) {
                                        	if(syncCpeForm.cpeL2TunnelMode == '2') {
                                        		if((row.untaggedVlan||'').length == 0 && ![0,'0'].includes(row.untaggedVlan)) {
													cb('Untagged VLAN and tagged VLANs cannot be all blank');
                                        		}else {
                                        			cb();
                                        		}
											}else {
                                            	cb();
											}
                                        }else {
                                            cb();
                                        }
                                    }
                                }
                            }
                        }">
                        <el-input v-model="row.taggedVlan" :disabled="isCpeAuto" style="width: 300px;"></el-input>
                    </el-form-item>
                </div>
                <i v-if="false" @click="ruleTipShow = !ruleTipShow" class="el-icon el-icon-menu-help" style="font-size: 20px;margin: 5px 15px;"></i>
                <el-form-item label-width="0" v-if="ruleTipShow">
                    <el-input v-show="false" v-model="syncCpeForm.cpeApnConfig"></el-input>
                    <div style="font-size: 14px;font-weight: bold;padding: 0 15px;">Rules:</div>
                    <table style="font-size: 12px;padding: 0 15px;color: #b95050;" class="tips-table">
                        <tr>
                            <td style="font-weight: bold;padding-right: 5px;">APN1:</td>
                            <td>APN TYPE must be L3(BearType=MGMT)</td>
                        </tr>
                        <tr>
                            <td style="font-weight: bold;padding-right: 5px;">APN2:</td>
                            <td>APN TYPE Can be set as L3-NAT,L2-BRIDGE or L2-TUNNEL</td>
                        </tr>
                        <tr>
                            <td style="font-weight: bold;padding-right: 5px;">APN3:</td>
                            <td>
                                IF APN2=L3-NAT,then APN3 should be unselected or APN TYPE=L3-NAT and BearType=RESERVED<br/>
                                IF APN2=L2-BRIDGE or L2-TUNNEL,then APN TYPE Must be equal with APN2 or APN3 be Disabled
                            </td>
                        </tr>
                        <tr>
                            <td style="font-weight: bold;padding-right: 5px;">APN4:</td>
                            <td>Logic is the same as APN3</td>
                        </tr>
                        <tr>
                            <td>IF Operation</td>
                            <td> mode = BRIDGE,you need to configure two APNs(MGMT + DATA) at least</td>
                        </tr>
                        <tr>
                            <td>IF Operation</td>
                            <td> mode = NAT or TUNNEL,you can only configure one APN(DATA) to CPE</td>
                        </tr>
                    </table>
                </el-form-item>
            </div>
            <div class="split-title">
                Advance <i :class="arrowCls" @click="isDown = !isDown"></i>
            </div>
            <div v-show="!isDown" class="group-bg-color flex-form half" style="border: none;">
                <el-form-item label="L2 Destination IP" prop="destinationIp" label-width="130">
                    <el-input v-model="syncCpeForm.destinationIp" :disabled="isCpeAuto"></el-input>
                </el-form-item>
            </div>
        </el-form>

        <span slot="footer">
            <el-button type="primary" @click="showSaveDialog('cpe')"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="syncCpeDLVisible=false"><%=rb.getString("QuXiao")%></el-button>
        </span>
    </el-dialog>
    <!-- 保存重启提示 -->
    <el-dialog ref="dl" title="<%=rb.getString("QueRen")%>" :visible.sync="dlShow" :append-to-body="false" width="500">
        <div v-if="saveFlag == 'enb'">
            <el-checkbox v-model="needReboot"><%=rb.getString("SheZhiHouChongQi")%></el-checkbox>
        </div>
        <div v-if="saveFlag == 'cpe'">
            <%=rb.getString("SheBeiChongQiTiShi")%>
        </div>
        <span slot="footer">
            <el-button type="primary" @click="confirmReboot"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="dlShow=false"><%=rb.getString("QuXiao")%></el-button>
        </span>
    </el-dialog>
</div>
</div>
<script>
if(window.halobMainPageVue) {
    try {
        window.halobMainPageVue.$destroy();
    }catch(e){}
}
window.halobMainPageVue = new Vue({
    el: '#imsi_ctn',
    data() {
        var vm = this,
            validateL2ServerIP  = function(rule,value,cb) {
                if(vm.syncForm.l2TunnelEnable == 'true') {
                    if(value && isValidIP(value)) {
                        cb();
                    }else if(value) {
                        cb('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>')
                    }else {
                        cb();
                    }
                }else {
                    cb();
                }
            },
            validateIP = function(rule,value,cb) {
                if(value && isValidIP(value)) {
                    cb();
                }else if(value) {
                    cb('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>')
                }else {
                    cb();
                }
            },
            validConfig = function(rule,value,cb) {
                var enableItems = value||[],
                    isBearTypeRight = false,
                    bearTypes = enableItems.map(function(item){
                        return item.bearType;
                    });
                
                
                if(bearTypes.includes('1')) isBearTypeRight = true;

                if(isBearTypeRight) {
                    cb();
                }else {
                    cb('<%=rb.getString("XuanZeBearType")%>');
                }
            },
            validateMtu = function(rule,val,cb) {
            	if(!['',undefined,null].includes(val) && (val -700 < 0 || val - 1600 > 0 || isNaN(val))) {
            		cb('Range: 700 - 1600');
            	}else {
            		cb();
            	}
            };
            
        var codes = {
                enb: 'eNB',
                cpe: 'CPE',
                gnb: 'gNB'
            },
            netType = codes[sysMain.headType],
            defaultNetType = ['eNB','gNB','CPE'].includes(netType)? netType : 'eNB';
        
        return {
            dlShow: false,
            needReboot: true,
            saveFlag: '',

        	halobEnable: '${halobEnable}' != 'false',
        	ruleTipShow: false,
            apnTips: 'Non compliance with safety verification rules',
            tabStatus: defaultNetType,

            imsiURL: '${ctx}/cell/imsi/queryImsiPageList.action',

            syncForm: {
            	mtu: '',
                configMode: '0',
            	lgwMode: '',
                smallCellCode: '',
                l2TunnelEnable: '',
                l2ServerIp: '',
                enbApnConfig: []
            },
            syncCpeForm: {
            	mtu: '',
                configMode: '0',
                cpeCode: '',
                destinationIp: '',
                greType: '',
                cpeL2TunnelMode: '0',
                cpeApnConfig: [
                    {apnName: '',apnOrder:'1',bearType:'1',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'2',bearType:'0',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'3',bearType:'2',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'4',bearType:'2',taggedVlan: '',untaggedVlan: ''}
                ]
            },
            defaultForm: {
                configMode: '0',
            	lgwMode: '',
                smallCellCode: '',
                l2TunnelEnable: '',
                l2ServerIp: '',
                enbApnConfig: []
            },
            originCpeForm: {
                cpeCode: '',
                destinationIp: '',
                greType: '',
                cpeL2TunnelMode: '0',
                cpeApnConfig: [
                    {apnName: '',apnOrder:'1',bearType:'1',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'2',bearType:'0',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'3',bearType:'2',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'4',bearType:'2',taggedVlan: '',untaggedVlan: ''}
                ]
            },
            defaultCpeForm: {
                configMode: '0',
                cpeCode: '',
                destinationIp: '',
                greType: '',
                cpeL2TunnelMode: '0',
                cpeApnConfig: [
                    {apnName: '',apnOrder:'1',bearType:'1',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'2',bearType:'0',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'3',bearType:'2',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'4',bearType:'2',taggedVlan: '',untaggedVlan: ''}
                ]
            },

            enbRules: {
                l2ServerIp: [{validator: validateL2ServerIP}],
                mtu: [{validator: validateMtu}]
            },
            cpeRules: {
                destinationIp: [{validator: validateIP}],
                cpeApnConfig: [{validator: validConfig}],
                mtu: [{validator: validateMtu}]
            },

            halobURL: '${ctx}/cell/imsi/queryEnbPageList.action',
            halobParams: {
                timeZone: timeZone,
                searchText: '',
                connectionStatus: '',
                halobEnable: '',
                result: '',
                status: ''
            },
            enbQueryForm: {
                connectionStatus: '',
                halobEnable: '',
                result: '',
                status: ''
            },

            gnbAPNURL: '${ctx}/cell/imsi/queryGNBPageList.action',
            gnbParams: {
                timeZone: timeZone,
                searchText: '',
                connectionStatus: '',
                halobEnable: '',
                result: '',
                status: ''
            },

            cpeApnUrl: '${ctx}/epc/apnl2config/queryApnL2ResultCpe.action',
            cpeApnParams: {
                timeZone: timeZone,
                searchText: '',
                connectionStatus: '',
                apnSyncStatus: '',
                apnName: ''
            },
            cpeQueryForm: {
                connectionStatus: '',
                apnSyncStatus: '',
                apnName: ''
            },
            halobmenus: [],

            apnslide: {
                title: '<%=rb.getString("SheZhi")%>',
                url: ''
            },
            imsislide: {
                header: true,
                title: '',
                url: ''
            },
            imeislide: {
                header: true,
                title: '',
                url: ''
            },

            switchParams: {
                halob_switch: '',
                cell_code: '',
                reboot: false
            },
            reboot: false,
            dialogVisible: false,
            enableFlag:false,
            syncDLVisible: false,
            syncCpeDLVisible: false,

            isDown: true,
            viewShow: false,
            viewIndex: '',
            viewForm: {
                APN_AMBR_DL: '',
                APN_AMBR_UL: '',
                GW_IP_ADDRESS: '',
                PRIMARY_DNS_IPADDR: '',
                SECONDARY_DNS_IPADDR: '',
                QCI: '',
                ARP_PRIORITYLEVEL: '',
                ARP_PCI: '',
                ARP_PVI: '',
                IPPOOL_INFO: []
            },
            imsiEnable: true,

            enb_search_text:'',
            gnb_search_text:'',
			cpe_search_text:'',
			placeholderText:'<%=rb.getString("QingShuRu")%>',
            advancedQueryItemList:[
				{
                    tabStatus:'eNB',
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("UPSLianJieZhuangTai") %>',
					options:[
						{value:"",label:'<%=rb.getString("QuanBu")%>'},
                        {value:"1",label:'<%=rb.getString("LianJieZhengChang")%>'},
                        {value:"0",label:'<%=rb.getString("LianJieDuanKai")%>'},
                        {value:"3",label:'<%=rb.getString("TongBuZhong")%>'},
                        {value:"2",label:'<%=rb.getString("TongBuShiBai")%>'}
					],
					value:'connectionStatus',
				},
				{
                    tabStatus:'eNB',
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("HaloBKaiGuan") %>',
					options:[
                        {value:'',label:'<%=rb.getString("QuanBu") %>'},
						{value:'1',label:'<%=rb.getString("HalobKaiQi") %>'},
						{value:'0',label:'<%=rb.getString("HalobGuanBi") %>'}
					],
					value:'halobEnable',
				},
                {
                    tabStatus:'eNB',
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'APN Status',
					options:[
                        {value:'',label:'<%=rb.getString("QuanBu") %>'},
                        {value:'2',label:'<%=rb.getString("JinXingZhong") %>'},
						{value:'1',label:'<%=rb.getString("ChengGong") %>'},
						{value:'0',label:'<%=rb.getString("ShiBai") %>'}
					],
					value:'result',
				},
				{
                    tabStatus:'gNB',
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("UPSLianJieZhuangTai") %>',
					options:[
						{value:"",label:'<%=rb.getString("QuanBu")%>'},
                        {value:"1",label:'<%=rb.getString("LianJieZhengChang")%>'},
                        {value:"0",label:'<%=rb.getString("LianJieDuanKai")%>'},
                        {value:"3",label:'<%=rb.getString("TongBuZhong")%>'},
                        {value:"2",label:'<%=rb.getString("TongBuShiBai")%>'}
					],
					value:'connectionStatus',
				},
				{
                    tabStatus:'gNB',
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("HaloBKaiGuan") %>',
					options:[
                        {value:'',label:'<%=rb.getString("QuanBu") %>'},
						{value:'1',label:'<%=rb.getString("HalobKaiQi") %>'},
						{value:'0',label:'<%=rb.getString("HalobGuanBi") %>'}
					],
					value:'halobEnable',
				},
                {
                    tabStatus:'gNB',
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'APN Status',
					options:[
                        {value:'',label:'<%=rb.getString("QuanBu") %>'},
                        {value:'2',label:'<%=rb.getString("JinXingZhong") %>'},
						{value:'1',label:'<%=rb.getString("ChengGong") %>'},
						{value:'0',label:'<%=rb.getString("ShiBai") %>'}
					],
					value:'result',
				},
                {
                    tabStatus:'CPE',
					type:'select',
					isShow:false,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("UPSLianJieZhuangTai") %>',
					options:[
						{value:"",label:'<%=rb.getString("QuanBu")%>',},
                        {value:"On",label:'<%=rb.getString("LianJieZhengChang")%>'},
                        {value:"Off",label:'<%=rb.getString("LianJieDuanKai")%>'},
                        {value:"updating",label:'<%=rb.getString("TongBuZhong")%>'},
                        {value:"Exception",label:'<%=rb.getString("TongBuShiBai")%>'}
					],
					value:'connectionStatus',
				},
				{
                    tabStatus:'CPE',
					type:'select',
					isShow:false,
					popoverShow:false,
					selectVal:'',
					label:'APN Status',
					options:[
						{value:"",label:'<%=rb.getString("QuanBu")%>',},
                        {value:"2",label:'<%=rb.getString("JinXingZhong")%>'},
                        {value:"1",label:'<%=rb.getString("ChengGong")%>'},
                        {value:"0",label:'<%=rb.getString("ShiBai")%>'},
					],
					value:'apnSyncStatus',
				},
				{
                    tabStatus:'CPE',
					type:'select',
					isShow:false,
					popoverShow:false,
					selectVal:'',
					label:'Apn Name',
					options:[],
					value:'apnName',
				},
			],
        };
    },
    computed: {
        isBridge() {
            var vm = this,
                bool = false;
            
            if(vm.syncForm.enbApnConfig) {
                vm.syncForm.enbApnConfig.map(function(item) {
                    if(item.apnType == '2' || item.apnType == '3') {
                        bool = true;
                    }
                });
            }
            
            return bool;
        },
        isEnbAuto() {

            return this.syncForm.configMode === '0';
        },
        isCpeAuto() {
            return this.syncCpeForm.configMode === '0';
        },
    	isEnb() {
    		return this.tabStatus == 'eNB';
    	},
        mgmtDisabled() {
            var vm = this,
                bool = false;

            vm.syncCpeForm.cpeApnConfig.map(function(item){
                if(item.bearType == '1') bool = true;
            });

            return bool;
        },
        arrowCls() {
            var vm = this;

            return {
                'el-icon': true,
                'el-icon-down': vm.isDown,
                'el-icon-up': !vm.isDown
            }
        },
        isWritable() {
        	
        	return writableMap.CODE_ENB_DEVICE_HALOB == true;
        },
        netType() {
            var vm = this,
                neType = sysMain.headType,
                codes = {
                    enb: 'eNB',
                    cpe: 'CPE',
                    gnb: 'gNB'
                };

            return codes[neType];
        }
    },
    watch: {
        netType(type) {
            var vm = this,
                types = ['eNB','gNB','CPE'];

            if(types.includes(type)) {
                vm.tabStatus = type;
                vm.tabStatusChange(type);
            }

            vm.syncDLVisible = false;
            vm.syncCpeDLVisible = false;
        }
    },
    methods: {
        linkTunnel() {
            var vm = this,
                hasEnable = false,
                isBridge = false;

            vm.syncForm.enbApnConfig.map(function(item){
                if(item.apnType == '2') hasEnable = true;
                
                if(item.apnType == '2' || item.apnType == '3') {
                	isBridge = true;
                }
            });
            
            if(isBridge) {
            	vm.syncForm.lgwMode = '2';
            }else {
            	vm.syncForm.lgwMode = '0';
            }
            
            var newVal = hasEnable?'true':'false';
            if(newVal != vm.syncForm.l2TunnelEnable) {
                vm.syncForm.l2TunnelEnable = newVal;
                vm.$message({
                    message: 'eNB L2 Tunnel will be changed to ' + (newVal=='true'?'enable':'disable'),
                    type: 'warning'
                })
            }
        },
        getApnNameList() {
            var vm = this;

            axios.post('${ctx}/epc/apnconfig/queryApnNameList.action').then(function(res){
                var data = res.data||[];

                if(data) {
					var arr = [{label:'<%=rb.getString("QuanBu")%>',value:''}];
					data.map(function(item){
						if(item){
							arr.push({label:item,value:item})
						}
					})
                    vm.advancedQueryItemList.map((items)=>{
						if('apnName' == items.value){
							items.options = arr
						}
					})
                }
            });
        },
        getLayer(mode, apnType) {
            if(mode == '0') {
                return 'L3';
            }else if(['2','3'].includes(mode)){
                if(apnType == '1') {
                    return 'L3'
                }else if(['0'].includes(apnType)){
                    return 'L2'
                }else {
                    return '';
                }
            }else {
                return '';
            }
            
        },
        viewInfo(row,idx) {
            var vm = this,
                url = '${ctx}/epc/apnconfig/queryApnInfoByName.action',
                params = {
                    apnName: row.apnName
                };

            vm.viewIndex = idx;
            vm.viewShow = false;

            Object.assign(vm.viewForm, {
                APN_AMBR_DL: '',
                APN_AMBR_UL: '',
                GW_IP_ADDRESS: '',
                PRIMARY_DNS_IPADDR: '',
                SECONDARY_DNS_IPADDR: '',
                QCI: '',
                ARP_PRIORITYLEVEL: '',
                ARP_PCI: '',
                ARP_PVI: ''
            });

            axios.post(url, stringify(params)).then(function(res){
                var data = res.data || {};

                Object.assign(vm.viewForm, data);
                vm.viewShow = true;
            });
        },
        halobConfirm() {
            var vm = this,
                url = '${ctx}/cell/cpeinfos/setCellHalobSwitch.action';

            vm.switchParams.reboot = vm.reboot;
            vm.switchParams.halob_switch = vm.switchParams.halob_switch=='1'?'0':'1';

            axios.post(url, stringify(vm.switchParams)).then(function(res){
                var data = res.data;

                if(data.success == true) {
                    vm.$message({
                        type: 'success',
                        message: '<%=rb.getString("XiaFaChengGong")%>'
                    });
                    vm.halobCancel();
                }else {
                    vm.$message({
                        type: 'error',
                        message: data.message
                    });
                    row.halobEnable = row.halobEnable == '1'?'0':'1';
                }
            });
        },
        halobCancel() {
            var vm = this;

            Object.assign(vm.halobParams, {
                halob_switch: '',
                cell_code: ''
            });
            vm.dialogVisible = false;
            vm.reboot = false;
        },
        // table formatters
        connStatusFmt(row, value, index) {
            return connStatusFormatterSyn(value, row, index);
        },
        fmtImsiInfo(list) {
            var html = '';

            if(list) {
                var str = list.map(function(item){
                    return item.apnName;
                }).join(' , ');

                html = list.length + ' ( ' + str + ' )';
            }

            return html;
        },
        fmtHalob(list) {
            var html = '';

            if(list) {
                list.map(function(item){
                    var hostName = item.hostName? '(' + item.hostName + ')':'';

                    html += '<div>' + item.serialNumber + hostName + '</div>';
                });
            }

            return html;
        },
        fmtAssignTime(list) {
            var html = '';

            if(list && list[0]) {
                html = list[0].assignTime;
            }

            return html;
        },
        halobEnableChange(val, row, evt) {
            var vm = this,
                params = {
                	halob_switch: val,
                    cell_code: row.smallCellCode
                };

            evt.stopPropagation();
            if ( val == "1"){
            	vm.enableFlag = true
            }
            vm.dialogVisible = true;
            vm.reboot = false;
            Object.assign(vm.switchParams, params);
        },
        toImsiList(row) {
            var vm = this;

            vm.imsislide.header = true;
            vm.imsislide.title = 'All IMSI';
            vm.imsislide.url = '${ctx}/cell/imsi/goRequestImsi.action';
            vm.$refs.imsilist.showSlide(function(){
                eventBus.$emit('init-allimsi', row);
            });
        },
        toIMSIAllocated() {
            var vm = this;

            vm.imsislide.header = false;
            vm.imsislide.url = '${ctx}/cell/imsi/goImsiList.action';
            vm.$refs.imsilist.showSlide();
        },
        toIMEIConfig(){
            var vm = this;

            vm.imeislide.header = false;
            vm.imeislide.title = 'IMEI Config';
            vm.imeislide.url = '${ctx}/cell/imei/goIMEIPage.action';
            vm.$refs.imeilist.showSlide();
        },
        exportHalob() {
            var vm = this;
            if(vm.isEnb) {
                exportByForm('${ctx}/cell/imsi/exportEnbList.action',vm.halobParams);
            }else if(vm.tabStatus == 'gNB') {// 导出接口待确认。。。
                exportByForm('${ctx}/cell/imsi/exportEnbList.action', Object.assign({deviceType: 'gNB'}, vm.gnbParams));
            }else {
                exportByForm('${ctx}/epc/apnl2config/exportApnL2ResultCpeToExcel.action',vm.cpeApnParams);
            }
        },
        syncHalob(row) {
            var vm = this,
                url = '${ctx}/cell/imsi/sendImsiToEnb.action',
                params = {
                    cmdType: 'imsi',
                    smallCellCode: []
                };

            if(row.smallCellCode) {
                params.smallCellCode.push(row.smallCellCode);
            }

            params.smallCellCode = JSON.stringify(params.smallCellCode);

            vm.$confirm('<%=rb.getString("QueRenTongBuIMSIDaoJiZhan")%>','Confirm').then(function(r){
                if(r) {
                    axios.post(url, stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success) {
                            vm.$message({
                                type: 'success',
                                message: '<%=rb.getString("ChengGong")%>'
                            });
                        }else {
                            vm.$message({
                                type: 'error',
                                message: data.message
                            });
                        }
                    });
                }
            }).catch(function(){});
        },
        clearIMSI(row) {
            var vm = this,
                url = '${ctx}/cell/imsi/sendImsiToEnb.action',
                params = {
                    cmdType: 'clear',
                    smallCellCode: []
                };

            if(row.smallCellCode) {
                params.smallCellCode.push(row.smallCellCode);
            }

            params.smallCellCode = JSON.stringify(params.smallCellCode);

            vm.$confirm('<%=rb.getString("QueRenQingChuIMSI")%>','Confirm').then(function(r){
                if(r) {
                    axios.post(url, stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success) {
                            vm.$message({
                                type: 'success',
                                message: '<%=rb.getString("ChengGong")%>'
                            });
                            
                            vm.$refs.halob.refresh();
                        }else {
                            vm.$message({
                                type: 'error',
                                message: data.message
                            });
                        }
                    });
                }
            }).catch(function(){});
        },
        resetApnForm() {
            var vm = this;

            Object.assign(vm.syncForm, {
                configMode: '0',
                smallCellCode: '',
                l2TunnelEnable: '',
                l2ServerIp: '',
                enbApnConfig: []
            });

            Object.assign(vm.syncCpeForm, {
                configMode: '0',
                cpeCode: '',
                destinationIp: '',
                greType: '',
                cpeL2TunnelMode: '0',
                cpeApnConfig: [
                    {apnName: '',apnOrder:'1',bearType:'1',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'2',bearType:'0',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'3',bearType:'3',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'4',bearType:'3',taggedVlan: '',untaggedVlan: ''}
                ]
            });
            Object.assign(vm.defaultCpeForm, {
                configMode: '0',
                cpeCode: '',
                destinationIp: '',
                greType: '',
                cpeL2TunnelMode: '0',
                cpeApnConfig: [
                    {apnName: '',apnOrder:'1',bearType:'1',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'2',bearType:'0',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'3',bearType:'3',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'4',bearType:'3',taggedVlan: '',untaggedVlan: ''}
                ]
            });
            Object.assign(vm.originCpeForm, {
                configMode: '0',
                cpeCode: '',
                destinationIp: '',
                greType: '',
                cpeL2TunnelMode: '0',
                cpeApnConfig: [
                    {apnName: '',apnOrder:'1',bearType:'1',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'2',bearType:'0',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'3',bearType:'3',taggedVlan: '',untaggedVlan: ''},
                    {apnName: '',apnOrder:'4',bearType:'3',taggedVlan: '',untaggedVlan: ''}
                ]
            });
        },
        syncApn(row) {
            var vm = this,
                url = '${ctx}/epc/apnl2config/queryApnL2ConfigEnb.action',
                params = {
                    smallCellCode: row.smallCellCode
                };

            vm.isDown = true;
            vm.resetApnForm();
            vm.syncForm.smallCellCode = row.smallCellCode;

            axios.post(url, stringify(params)).then(function(res){
                var data = res.data;

                if(data['enbApnConfig']) {
                    data['enbApnConfig'].map(function(itm){
                        if(itm.defaultApn == '1' && itm.vlanId == '0') itm.vlanId = '';
                    });
                }
                
                data.configMode = data.configMode == '1'?'1':'0';

                Object.assign(vm.syncForm, data);
                vm.syncForm.smallCellCode = row.smallCellCode;
                vm.syncDLVisible = true;

                Object.assign( vm.defaultForm, JSON.parse(JSON.stringify(vm.syncForm)), {configMode: '1'});
                
                vm.$nextTick(function(){
                    vm.$refs.enbApnForm.clearValidate();
                });
            });
        },
        showCpeSync(row) {
            var vm = this,
                url = '${ctx}/epc/apnl2config/queryApnL2ConfigCpe.action',
                params = {
                    cpeCode: row.cpeCode
                };

            vm.isDown = true;
            vm.resetApnForm();
            vm.syncCpeForm.cpeCode = row.cpeCode;

            axios.post(url, stringify(params)).then(function(res){
                var data = res.data;
                
                data.configMode = data.configMode == '1'?'1':'0';

                ['mtu','configMode','destinationIp','greType','cpeL2TunnelMode'].map(function(code){
                    if(data[code] != undefined) {
                        vm.syncCpeForm[code] = data[code];
                    }else{
                        vm.syncCpeForm[code] = data[code+'Default'];
                    }
                });
                // 初始化cpeApnConfig -- 固定四个赋值
                vm.syncCpeForm['cpeApnConfig'].map(function(item,idx){
                    var itemCode = '';

                    if(data['cpeApnConfig'] != undefined) {
                        itemCode = 'cpeApnConfig';
                    }else {
                        itemCode = 'cpeApnConfigDefault';
                    }

                    var dataList = data[itemCode]||[],
                        dataItem = dataList[idx];

                    if(dataItem) {
                        Object.assign(item, dataItem);
                    }
                });

                // if(vm.syncCpeForm.cpeL2TunnelMode == '2') {
                //     vm.syncCpeForm['cpeApnConfig'][2].bearType = '0';
                //     vm.syncCpeForm['cpeApnConfig'][3].bearType = '0';
                // }

                vm.syncCpeForm.cpeCode = row.cpeCode;
                vm.syncCpeDLVisible = true;

                ['mtu','destinationIp','greType','cpeL2TunnelMode'].map(function(code){
                    vm.originCpeForm[code] = data[code+'Default'];
                });
                //vm.originCpeForm.cpeApnConfig = JSON.parse(JSON.stringify(data['cpeApnConfigDefault']));
                
                vm.originCpeForm['cpeApnConfig'].map(function(item,idx){
                    var itemCode = 'cpeApnConfigDefault',
                        dataList = data[itemCode]||[],
                        dataItem = dataList[idx];

                    if(dataItem) {
                        Object.assign(item, dataItem);
                    }
                });

                // if(vm.originCpeForm.cpeL2TunnelMode == '2') {
                //     vm.originCpeForm['cpeApnConfig'][2].bearType = '0';
                //     vm.originCpeForm['cpeApnConfig'][3].bearType = '0';
                // }

                vm.originCpeForm.cpeCode = row.cpeCode;

                Object.assign( vm.defaultCpeForm, JSON.parse(JSON.stringify(vm.syncCpeForm)), {configMode: '1'});
                if(data.configMode != '1') {
                    Object.assign(vm.syncCpeForm, vm.originCpeForm);
                    vm.syncCpeForm.cpeApnConfig = JSON.parse(JSON.stringify(vm.originCpeForm.cpeApnConfig));
                }

                vm.$nextTick(function(){
                    vm.$refs.cpeApnForm.clearValidate();
                });
            });
        },
        enbModeChange(value) {
            var vm = this;

            if(value === '0') {
                axios.post('${ctx}/epc/apnl2config/queryApnL2ConfigEnb.action').then(function(res){
        			var data = res.data || {};
        			
                    if(data['enbApnConfig']) {
                        data['enbApnConfig'].map(function(itm){
                            if(itm.defaultApn == '1' && itm.vlanId == '0') itm.vlanId = '';
                        });
                    }

        			['lgwMode','mtu','l2ServerIp', 'enbApnConfig', 'l2TunnelEnable'].map(function(code){
        				vm.syncForm[code] = data[code];
        			});
        		})
            }else {
                Object.assign( vm.syncForm, JSON.parse(JSON.stringify(vm.defaultForm)) );
            }
        },
        cpeModeChange(value) {
            var vm = this;

            if(value === '0') {
                Object.assign(vm.syncCpeForm, vm.originCpeForm);
                vm.syncCpeForm.cpeApnConfig = JSON.parse(JSON.stringify(vm.originCpeForm.cpeApnConfig));
            }else {
                Object.assign(vm.syncCpeForm, vm.defaultCpeForm);
                vm.syncCpeForm.cpeApnConfig = JSON.parse(JSON.stringify(vm.defaultCpeForm.cpeApnConfig));
            }
        },
        halobOptClick(row,ev) {
            var vm = this;

            var imsiEnable = true;
			$.ajax({
				type:'POST',
				url:'${ctx}/cell/imsi/getIMSIOperationItem.action',
				data:{smallCellCode: row.smallCellCode},
				async:false,
				dataType:'json',
				success:function(data){
					if(data) {
						imsiEnable = data.imsiFlag == true;
                        vm.imsiEnable = data.imsiFlag == true;
					}
				}
			});

            vm.halobmenus= [
                {label:'<%=rb.getString("QingChuIMSI")%>',cls:"el-icon-operation-clear el-icon",code:'clear', row: row,disable: !imsiEnable},
                {label:'<%=rb.getString("XiaFaDaoJiZhan")%>',cls:"el-icon-operation-synchronize el-icon" ,code:'dispatch', row: row,disable: !imsiEnable},
                {label:'Modify APN',cls:"el-icon-operation-edit el-icon" ,code:'modifyApn', row: row,disable: !imsiEnable, show: vm.halobEnable}
            ];

            if(vm.tabStatus == 'gNB') {
                vm.halobmenus= [
                    {label:'<%=rb.getString("QingChuIMSI")%>',cls:"el-icon-operation-clear el-icon",code:'clear', row: row,disable: !imsiEnable},
                    {label:'<%=rb.getString("XiaFaDaoJiZhan")%>',cls:"el-icon-operation-synchronize el-icon" ,code:'dispatch', row: row,disable: !imsiEnable}
                ];
            }
            
            vm.$nextTick(function() {
                document.body.click();
                vm.$refs.halobmenu.show(ev);
            });
        },
        halobClickEvent(row) {
            var vm = this,
                code = row.code,
                actions = {
                    clear: vm.clearIMSI,
                    dispatch: vm.syncHalob,
                    modifyApn: vm.syncApn
                };

            if(actions[code]) {
                actions[code](row.row);
            }
        },
        handerClose() {
            this.$refs.menu.hide();
        },
        halobHanderClose() {
            this.$refs.halobmenu.hide();
        },
        apnSetting() {
            var vm = this;

            vm.apnslide.url = '${ctx}/epc/apnl2config/goApnL2Config.action';
            //vm.apnslide.url = '${ctx}/cell/imsi/goAPNSetting.action';
            vm.$refs.apn.showSlide();
        },
        closeApn() {
            this.$refs.apn.hide();
        },
        closeIMSI() {
            this.$refs.imsilist.hide();
        },
        closeIMEI() {
            this.$refs.imeilist.hide();
        },
        saveApn() {
            eventBus.$emit('save-apn');
        },
        saveEnbApn() {
            var vm = this,
                url = '${ctx}/epc/apnl2config/sendApnL2ConfigEnb.action',
                params = JSON.parse(JSON.stringify(vm.syncForm));

            params.enbApnConfig.map(function(itm){
                if(itm.defaultApn == '1' && itm.vlanId === '') itm.vlanId = '0';
                
                if(['2','3'].includes(itm.apnType) && itm.untaggedVlan == '' && itm.taggedVlan == '') {
                    itm.untaggedVlan = '0';
                    itm.taggedVlan = '0';
                }
            });

            params.enbApnConfig = JSON.stringify(params.enbApnConfig);
            if(params.configMode != '1') {
            	params.enbApnConfig = '[]';
            }

            vm.$refs.enbApnForm.validate(function(r){
                if(r) {
                    params.isReboot = vm.needReboot? '1' : '0';
                    
                    axios.post(url, stringify(params)).then(function(res){
                        var data = res.data;

                        if(data['success'] ) {
                            eventBus.$message({
                                message: '<%=rb.getString("ChengGong")%>',
                                type: 'success'
                            });

                            vm.syncDLVisible = false;
                        }else {
                            vm.$message({
                                message: data.message,
                                type: 'error'
                            });
                        }
                    });
                }
            });
        },
        saveCpeApn() {
            var vm = this,
                url = '${ctx}/epc/apnl2config/sendApnL2ConfigCpe.action',
                params = JSON.parse(JSON.stringify(vm.syncCpeForm));

            params.cpeApnConfig.map(function(item){
                if(item.bearType == '1') item.taggedVlan = '';

                if(item.bearType == '0' && ['3'].includes(vm.syncCpeForm.cpeL2TunnelMode)) {// Bridge -- Data
                    item.untaggedVlan = '0';
                }
            });

            params.cpeApnConfig = JSON.stringify(params.cpeApnConfig);

            if(params.configMode != '1') {
                params.cpeApnConfig = '[]';
            }

            vm.$refs.cpeApnForm.validate(function(r){
                if(r) {
                    params.isReboot = '0';

                    axios.post(url, stringify(params)).then(function(res){
                        var data = res.data;

                        if(data['success'] ) {
                            eventBus.$message({
                                message: '<%=rb.getString("ChengGong")%>',
                                type: 'success'
                            });

                            vm.syncCpeDLVisible = false;
                        }else {
                            vm.$message({
                                message: data.message,
                                type: 'error'
                            });
                        }
                    });
                }
            });
        },
        confirmReboot() {
            var vm = this;

            if(vm.saveFlag == 'enb') {
                vm.saveEnbApn();
            }else if(vm.saveFlag == 'cpe'){
                vm.saveCpeApn();
            }
            
            vm.dlShow = false;
        },
        showSaveDialog(flag) {
            var vm = this;

            vm.saveFlag = flag;

            vm.dlShow = true;
            vm.needReboot = true;
        },
        // 表格切换事件
        tabStatusChange(val){
            var vm =this;
            vm.tabStatus = val;

			vm.advancedQueryItemList.map((items)=>{
				if(vm.tabStatus == items.tabStatus){
					items.isShow = true;
				}else{
					items.isShow = false;
				}
			})	
        },
        // 清除筛选
		clearFilterClick(){
			var vm = this,
				params = {};
			vm.advancedQueryItemList.map((items)=>{
				if(items.isShow && items.isShow== true ){
					if(items.type == 'checkbox'){
						items.checkedItemList = [];
						items.oldCheckedItemList = [];
						items.checkAll = false;
						items.isIndeterminate = false;
					}else if(items.type == 'select'){
						items.selectVal = '';
					}
					if(items.type == 'checkbox' || items.type == 'select'){
						params[items.value] = '';
                        if(vm.isEnb){
                            params.status = '';
                        }
					}
				}
			})

			if(vm.tabStatus == 'eNB'){
				Object.assign(vm.halobParams, params);
			}else if(vm.tabStatus == 'gNB'){
                Object.assign(vm.gnbParams, params);
            }else{
				Object.assign(vm.cpeApnParams, params);
			}
			document.body.click();
		},
        // 模糊搜索
		query(){
			var vm = this;
			
			if(vm.tabStatus == 'eNB'){
				vm.halobParams.searchText = this.enb_search_text;
			}else if(vm.tabStatus == 'gNB'){
                vm.gnbParams.searchText = this.gnb_search_text;
            }else{
				vm.cpeApnParams.searchText = this.cpe_search_text;
			}
			
		},
		// 搜索域聚焦事件
		queryInputFocus(){
			var vm = this;

			if(vm.tabStatus == 'CPE'){
				vm.placeholderText = 'IMSI/CPE SN';
			}else{
				vm.placeholderText = 'IMSI/<%=rb.getString("XiaoZhanXuLieHao")%>';
			}
			
		},
		// 搜索域失焦事件
		queryInputBlur(){
			var vm = this;
			vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
		},
        // 高级查询 确定事件
        advanceQuery(type,paramsItem,value){
            var vm = this,
                params ={};
            if(type == 'select'){
                if(vm.isEnb){
                    if(paramsItem == 'result' && value == '2'){
                        params.status = '1';
                        params[paramsItem] = '';
                    }else{
                        params[paramsItem] = value;
                        params.status = '';
                    }
                }else{
                    params[paramsItem] = value;
                }
            }else{
                params[paramsItem] = value.join(',');
            }
            if(vm.tabStatus == 'eNB'){
				Object.assign(vm.halobParams, params);
			}else if(vm.tabStatus == 'gNB'){
				Object.assign(vm.gnbParams, params);
            }else{
				Object.assign(vm.cpeApnParams, params);
			}
			document.body.click();
        },
    },
    mounted() {
        this.getApnNameList();
        this.tabStatusChange(this.tabStatus);

    	eventBus.$off('hide-apn').$on('hide-apn', this.closeApn);
    	eventBus.$off('hide-imsi').$on('hide-imsi', this.closeIMSI);
        eventBus.$off('hide-imei').$on('hide-imei', this.closeIMEI);
    }
});
</script>
