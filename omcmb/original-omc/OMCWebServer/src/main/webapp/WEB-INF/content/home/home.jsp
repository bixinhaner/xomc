<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>


<style type="text/css">
    #dashboard_ctn{
        position: relative;
        width: 100%;
        height: 100%;
        display: flex;
        min-width: 900px;
        background: unset;
    }
    #dashboard_ctn .dashboardEchartsCustomBoxCls{
        height: 100%;
        display: flex;
        flex-direction: column;
        margin-right: 10px;
        border: 1px solid #DFE2EE;
        border-radius: 4px;
        background-color: #FFFFFF;
        box-sizing: border-box;
    }
    #dashboard_ctn .dashboardEchartsCustomBoxCls .customBoxTitleCls{
        height: 42px;
        min-height: 42px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        border-bottom: 1px solid #DFE2EE;
        box-sizing: border-box;
    }
    .customBoxMainCls{
        height: calc(100% - 42px);
        overflow: auto;
    }
    #dashboard_ctn .dashboardEchartsCustomBoxCls .customBoxItemCls{
        height: 36px;
        min-height: 36px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0px 16px;
        box-sizing: border-box;
    }
    .customBoxItemModuleNameCls > span:nth-child(1){
        font-size: 12px;
        color: rgba(0,0,0,0.32);
    }
    #dashboard_ctn .dashboardEchartsCustomBoxCls .el-collapse-item__header{
        height: 36px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0px 16px;
        background-color: rgba(25,19,187,0.03);
        box-sizing: border-box;
    }
    #dashboard_ctn .dashboardEchartsCustomBoxCls .el-collapse-item__content{
        padding-bottom: 0px;
    }
    #dashboard_ctn .dashboardEchartsMainBoxCls{
        height: 100%;
        display: flex;
        flex-direction: column;
        overflow: auto;
    }
	#dashboard_ctn .board-item {
		position: relative;
		margin-bottom: 16px;
		background-color: #FFFFFF;
		border-radius: 6px;
		border:1px solid #dfe2ee;
	}
    #dashboard_ctn .board-item > .el-card {
		border: 1px solid #ebeef5;
	}
	#dashboard_ctn .el-card__footer{
		 height:unset;
		 line-height: unset;
		 padding-left:unset; 
		 border:none;
	 }
	#dashboard_ctn .inner-export {
		position: absolute;
		right: 10px;
		top:15px;
		display: inline-block;
		width: 30px;
		height: 30px;
		cursor: pointer;
		z-index: 100;
		font-size:18px !important;
	}
	#dashboard_ctn .chart-card {
		border: none;
        position: relative;
		border-bottom: 1px solid #ebeef5;
		border-right: 1px solid #ebeef5;
		border-radius: 0px !important;
	}
	#dashboard_ctn .el-card {
		border-radius: 8px;
	}
	#dashboard_ctn .el-card__header {
		background: #fff;
		border:none;
		border-bottom:1px solid #EEE;
		height: 50px;
		line-height: 50px;
	}
	#dashboard_ctn .el-card__body {
		padding:0;
	}
	#dashboard_ctn .el-date-editor .el-range__icon {
		color: #4D84FF;
	}
    #dashboard_ctn .dashboaedChart{
        position: relative;
    }
	#dashboard_ctn .dashboaedChart .el-card__body {
		padding:15px;
		box-sizing: border-box;
	}
    #dashboard_ctn .kpiNetworkTypeTitleCls{
        width: calc(100% - 20px);
        height: 32px;
        line-height: 32px;
        padding-left: 20px;
        background-color: #F5F5F5;
        margin: 10px;
        font-size: 14px;
        color: rgba(0,0,0,0.64);
        box-sizing: border-box;
    }
	#dashboard_ctn .statisticItemCls{
		min-width: 450px;
		display: flex;
		flex-wrap:wrap;
		width: 100%;
	}
	#dashboard_ctn .statisticItemCls > div{
		box-sizing: border-box;
		flex: 1;
		min-width: 450px;
		position: relative;
	}
    #dashboard_ctn .alarmStatisticMainBoxCls{
        gap: 20px;
        padding: 20px 15px;
        box-sizing: border-box;
    }
    #dashboard_ctn .alarmStatisticMainBoxCls .chart-card{
        border: 1px solid #DFE2EE;
        border-radius: 4px !important;
    }
    #dashboard_ctn .alarmStatisticMainBoxCls .alarmStatisticTitleBoxCls{
        display: flex;
        align-items: center;
    }
    #dashboard_ctn .alarmStatisticMainBoxCls .alarmStatisticTitleBoxCls .el-icon-message::before{
        color: #666666;
    }
    #dashboard_ctn .alarmStatisticMainBoxCls .alarmStatisticValueBoxCls{
        margin-left: 42px;
        padding-top: 10px;
        box-sizing: border-box;
    }
    #dashboard_ctn .alarmStatisticMainBoxCls .alarmStatisticValueBoxCls span{
        color: #FF4614;
        font-size: 40px;
        cursor: pointer;
    }
    #dashboard_ctn .statisticItemHeaderCls{
		height: 45px;
	}
    #dashboard_ctn .statisticItemHeaderAbsoluteCls{
        position: absolute;
        top: 15px;
        left: 15px;
        z-index: 99;
    }
    #dashboard_ctn .statisticItemNameCls{
        font-size: 14px;
        display: inline-block;
        color:rgba(0,0,0,0.8);
        font-weight:550;
        margin-left: 10px;
        box-sizing: border-box;
    }
    #dashboard_ctn .statisticItemHeaderCls .statisticChangeBoxCls{
		display: flex;
        flex-wrap: nowrap;
        gap: 45px;
		align-items: center;
        margin-bottom: 5px;
	}
    #dashboard_ctn .statisticItemHeaderCls .statisticChangeItemCls{
        position: relative;
        display: flex;
        align-items: center;
    }
    #dashboard_ctn .statisticItemHeaderCls .statisticChangeItemCls > span:nth-child(1){
        color: #666666;
        font-size:12px;
	}
    #dashboard_ctn .statisticItemHeaderCls .statisticChangeItemCls > span:nth-child(2){
        color: rgba(0,0,0,0.8);
        font-size:24px;
        margin: 0px 5px;
	}
	#dashboard_ctn .boardItemTitle{
		height: 46px;
		display: flex;
		align-items: center;
		padding: 0px 20px;
		box-sizing: border-box;
		border-bottom: 1px solid rgba(0,0,0,0.06);
		position: relative;
	}
	#dashboard_ctn .boardItemTitle  span:nth-child(2){
		font-size: 15px;
		font-weight: bold;
		color:#333333;
        margin-left: 8px;
        box-sizing: border-box;
	}
	.homeChartPopoverClass{
		padding: 10px;
	}
	#dashboard_ctn .homeAlarmChartHeaderCls{
		background-color: #FFFFFF;
		border: none;
		border-bottom: 1px solid rgba(0,0,0,0.06);
		height: 50px;
		line-height: 50px;
	}
	#dashboard_ctn .alarmHeaderTimeNotIconClass .el-date-editor .el-input__inner{
		cursor: pointer;
		opacity: 0.01;
	}
	#dashboard_ctn .alarmHeaderTimeNotIconClass .el-date-editor .el-input__prefix ,#dashboard_ctn .alarmHeaderTimeNotIconClass .el-date-editor .el-input__suffix{
		display: none;
	}
    #dashboard_ctn .statisticChangeItemCls .upIconCls::before{
        font-size: 16px;
        color: #0ABF5B;
    }
    #dashboard_ctn .statisticChangeItemCls .downIconCls::before{
        font-size: 16px;
        color: #E54545;
    }
    #dashboard_ctn .statisticChangeItemCls .rotate180{
        transform: rotate(180deg);
    }
    #dashboard_ctn .statisticChangeItemTitleCls{
        font-size: 12px;
        color: #666666;
        margin-bottom: 5px;
    }
    #dashboard_ctn .statisticChangeItemCountBoxCls{
        display: flex;
        align-items: center;
    }
    #dashboard_ctn .statisticChangeItemValueCls{
        font-size: 20px;
        margin-right: 5px;
        color: #334155;
        box-sizing: border-box;
    }
    #dashboard_ctn .placeholder-bt[tip]:hover::after {
        right: 0px;
    }
    #dashboard_ctn .periodRadioCls {
        position: absolute;
        top: 10px;
        right: 15px;
        z-index: 10;
        box-sizing: unset;
    }
    #dashboard_ctn .newIconBoxCls-title .el-icon::before{
        color: #FF8F1F;
        font-size: 16px;
    }
    #dashboard_ctn .newIconBoxCls-title {
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        height: 22px;
        width: 22px;
        border-radius: 100px;
        background: rgba(255,143,31,0.1);
        z-index: 99;
        color: #FF8F1F;
    }
    #dashboard_ctn .newIconBoxCls-alarmTitle .el-icon::before{
        color: #FA5151;
        font-size: 18px;
    }
    #dashboard_ctn .newIconBoxCls-alarmTitle {
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        height: 22px;
        width: 22px;
        border-radius: 5px;
        background: rgba(250,81,81,0.1);
        z-index: 99;
    }
    #dashboard_ctn .tabCustomLabelCls .el-icon-menu-eNB::before,#dashboard_ctn .tabCustomLabelCls .el-icon-menu-CPE::before{
        color: unset;
        font-size: 16px !important;
    }
    #dashboard_ctn .tabCustomLabelCls  .el-icon-dash_site::before,#dashboard_ctn .tabCustomLabelCls .el-icon-menu-5G::before , #dashboard_ctn .tabCustomLabelCls .el-icon-GSM::before{
        color: unset;
        font-size: 24px !important;
        position: relative;
        top: 3px;
    }
    #dashboard_ctn .statisticChangeDropdownCls{
        position: absolute;
        top: 60px;
        right: 20px;
        z-index: 66;
    }
    #dashboard_ctn .kpiFeatureHeaderBoxCls{
        height: 45px;
        display: flex;
        align-items: center;
        background-color: #F5F5F5;
        position: relative;
    }
    #dashboard_ctn .kpiFeatureHeaderBoxCls .kpiFeatureIconCls{
        display: inline-block;
        width: 6px;
        height: 18px;
        background-color: #F2B354;
        margin: 0px 10px;
        box-sizing: border-box;
    }
    #dashboard_ctn .kpiFeatureHeaderBoxCls .kpiFeatureTtileCls{
        font-size: 16px;
        font-weight: 500;
        color: rgba(0,0,0,0.8);
    }
    #dashboard_ctn .kpiFeatureBoxCls{
        display: flex;
        flex-wrap: wrap;
        width: 100%;
    }
    #dashboard_ctn .kpiFeatureItemBoxCls{
        flex: 1 1 49%;
        min-width: 450px;
        position: relative;
    }
    #dashboard_ctn .el-icon-operation-export:hover::before{
        color: var(--main-color);
    }
    #dashboard_ctn .newIconBoxCls-alarmTitle .el-icon-dash_site::before{
        font-size: 22px;
    }
    #dashboard_ctn .newTabs .el-tabs__header .el-tabs__item{
        height: 46px;
        color: rgba(0,0,0,0.64);
        line-height: 46px;
    }
    #dashboard_ctn .dashboardDayAndWeekChangeBoxCls{
        position: absolute;
        left: 50%;
        transform: translateX(-50%);
        z-index: 99;
    }
    #dashboard_ctn .dashboardDayAndWeekChangeBoxCls .commonRadioButton{
        display: inline-block;
    }
</style>
<div id="dashboard_ctn">
    <div v-if="false" class="dashboardEchartsCustomBoxCls" :style="{ width: customBoxExpand ? '340px' : '30px'}">
        <div class="customBoxTitleCls" :style="{ padding: customBoxExpand ? '0px 16px' : '0px 5px' }">
            <span v-show="customBoxExpand" style="font-size: 14px; font-weight: bold;">Custom Dashboard</span>
            <span @click="customBoxExpandChange" :class="customBoxExpand ? 'el-icon el-icon-common-left' : 'el-icon el-icon-common-right'"></span>
        </div>
        <div class="customBoxMainCls" v-show="customBoxExpand">
            <div class="customBoxItemCls">
                <span>Alarm</span>
                <el-switch v-model="customData.alarm"></el-switch>
            </div>
            <div class="customBoxItemCls" style="border-top: 1px solid #DFE2EE;">
                <span><%=rb.getString("ZhanDian")%></span>
                <el-switch v-model="customData.site"></el-switch>
            </div>
            <el-collapse>
                <el-collapse-item title="eNB" name="1">
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>Device Status Report</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Total/Online/Active</span>
                        <el-switch v-model="customData.enbOnlineOrActive"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>MME Status</span>
                        <el-switch v-model="customData.enbMmeStatus"></el-switch>
                    </div>
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>Device Report</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>UE Count</span>
                        <el-switch v-model="customData.enbUeCount"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Product Type</span>
                        <el-switch v-model="customData.enbProductType"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Device Running Time</span>
                        <el-switch v-model="customData.enbDeviceRunningTime"></el-switch>
                    </div>
                </el-collapse-item>
                <el-collapse-item title="gNB" name="2">
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>Device Status Report</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Total/Online/Active</span>
                        <el-switch v-model="customData.gnbOnlineOrActive"></el-switch>
                    </div>
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>Device Report</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>UE Count</span>
                        <el-switch v-model="customData.gnbUeCount"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Product Type</span>
                        <el-switch v-model="customData.gnbProductType"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Device Running Time</span>
                        <el-switch v-model="customData.gnbDeviceRunningTime"></el-switch>
                    </div>
                </el-collapse-item>
                <el-collapse-item title="GSM" name="3">
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>Device Status Report</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Total/Online/Active</span>
                        <el-switch v-model="customData.gsmOnlineOrActive"></el-switch>
                    </div>
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>Device Report</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>UE Count</span>
                        <el-switch v-model="customData.gsmUeCount"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Product Type</span>
                        <el-switch v-model="customData.gsmProductType"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Device Running Time</span>
                        <el-switch v-model="customData.gsmDeviceRunningTime"></el-switch>
                    </div>
                </el-collapse-item>
                <el-collapse-item title="CPE" name="3">
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>Device Status Report</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Total/Online</span>
                        <el-switch v-model="customData.cpeOnline"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>CPE_Online Rate</span>
                        <el-switch v-model="customData.cpeOnlineRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>Device Report</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Product Model</span>
                        <el-switch v-model="customData.cpeProductType"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Device Running Time</span>
                        <el-switch v-model="customData.cpeDeviceRunningTime"></el-switch>
                    </div>
                </el-collapse-item>
                <el-collapse-item title="KPI" name="4">
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>eNB</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Wireless Setup Success Rate</span>
                        <el-switch v-model="customData.enbWirelessSetupSuccessRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Total Data Volume DL(GB)</span>
                        <el-switch v-model="customData.enbTotalDataVolumeDL"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Total Data Volume UL(GB)</span>
                        <el-switch v-model="customData.enbTotalDataVolumeUL"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>RRC Setup Success Rate(%)</span>
                        <el-switch v-model="customData.enbRrcSetupSuccessRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>E-RAB Setup Success Rate(%)</span>
                        <el-switch v-model="customData.enbERABSetupSuccessRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>E-RAB Drop Rate(%)</span>
                        <el-switch v-model="customData.enbERABDropRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>CSFB Success Rate(%)</span>
                        <el-switch v-model="customData.enbCsfbSuccessRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>HO IntraEnbOutSucc Rate(%)</span>
                        <el-switch v-model="customData.enbHoIntraEnbOutSuccRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>HO IntraEnbInSucc Rate(%)</span>
                        <el-switch v-model="customData.enbHoIntraEnbInSuccRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>HO InterEnbOutSucc Rate(%)</span>
                        <el-switch v-model="customData.enbHoInterEnbOutSuccRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>HO InterEnbInSucc Rate(%)</span>
                        <el-switch v-model="customData.enbHoInterEnbInSuccRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Downlink PRB utilization Rate(Q)(%)</span>
                        <el-switch v-model="customData.enbDownlinkPRBUtilizationRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>Uplink PRB utilization Rate(Q)(%)</span>
                        <el-switch v-model="customData.enbUplinkPRBUtilizationRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>gNB</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>KPI.PdcpUpOctDL(GB)</span>
                        <el-switch v-model="customData.gnbPdcpUpOctDL"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>KPI.PdcpUpOctUL(GB)</span>
                        <el-switch v-model="customData.gnbPdcpUpOctUL"></el-switch>
                    </div>
                    <div class="customBoxItemCls customBoxItemModuleNameCls" >
                        <span>BTS</span>
                        <span style="height: 1px;background-color: #DFE2EE;flex-grow: 1;margin-left: 10px;"></span>
                    </div>
                    <div class="customBoxItemCls">
                        <span>KPI.CallSetupSuccRate(%)</span>
                        <el-switch v-model="customData.gsmCallSetupSuccRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>KPI.CallDropRate(%)</span>
                        <el-switch v-model="customData.gsmCallDropRate"></el-switch>
                    </div>
                    <div class="customBoxItemCls">
                        <span>KPI.HandoverSuccessRate(%)</span>
                        <el-switch v-model="customData.gsmHandoverSuccessRate"></el-switch>
                    </div>
                </el-collapse-item>
            </el-collapse>
            <div class="customBoxItemCls" style="border-bottom: 1px solid #DFE2EE;">
                <span>SAS</span>
                <el-switch v-model="customData.sas"></el-switch>
            </div>
        </div>
    </div>
    <!-- :style="{ width: customBoxExpand ? 'calc( 100% - 340px )' : 'calc( 100% - 40px )' }" -->
    <div class="dashboardEchartsMainBoxCls" style="width: 100%;">
        <!-- alarm statistic box -->
        <div class="board-item" v-show="visibleCode.alarmStatistic"> 
            <div class="dashboaedChart">
                <div class="boardItemTitle">
                    <div class="newIconBoxCls-title"><span class="el-icon el-icon-menu-alarm" ></span></div>
                    <span><%=rb.getString("GaoJingGuanLi")%></span>
                </div>
                <div class="statisticItemCls alarmStatisticMainBoxCls"> 
                    <el-card shadow="hover" class="chart-card" v-if="visibleCode.siteStatistic">
                        <span @click="alarmStatisticExport('siteOffLineAlarm')" class="inner-export el-icon el-icon-operation-export placeholder-bt" tip="<%=rb.getString("DaoChu")%>"></span>
                        <div class="statisticItemHeaderCls" style="height: 80px;">
                            <div class="alarmStatisticTitleBoxCls">
                                <div class="newIconBoxCls-alarmTitle"><span class="el-icon el-icon-dash_site" ></span></div>
                                <span class="statisticItemNameCls"><%=rb.getString("ZhanDianLiXianGaoJing")%></span>
                            </div>
                            <div class="alarmStatisticValueBoxCls">
                                <span @click="showCurrAliveAlarm('siteOffLineAlarm')">{{alarmStatisticData.site_alarm_count}}</span> 
                            </div>
                        </div>
                    </el-card>
                    <el-card shadow="hover" class="chart-card">
                        <span @click="alarmStatisticExport('cellOfflineAlarm')" class="inner-export el-icon el-icon-operation-export placeholder-bt" tip="<%=rb.getString("DaoChu")%>"></span>
                        <div class="statisticItemHeaderCls" style="height: 80px;">
                            <div class="alarmStatisticTitleBoxCls">
                                <div class="newIconBoxCls-alarmTitle"><span class="el-icon el-icon-dash_offline" ></span></div>
                                <span class="statisticItemNameCls"><%=rb.getString("XiaoQuLiXianGaoJing")%></span>
                                <!-- <span style="margin-left: 5px;" class="el-icon el-icon-message" title="Filtered Correlated Alerts"></span> -->
                            </div>
                            <div class="alarmStatisticValueBoxCls">
                                <span @click="showCurrAliveAlarm('cellOfflineAlarm')">{{alarmStatisticData.offline_alarm_count}}</span> 
                            </div>
                        </div>
                    </el-card>
                    <el-card shadow="hover" class="chart-card">
                        <span @click="alarmStatisticExport('cellInactiveAlarm')" class="inner-export el-icon el-icon-operation-export placeholder-bt" tip="<%=rb.getString("DaoChu")%>"></span>
                        <div class="statisticItemHeaderCls" style="height: 80px;">
                            <div class="alarmStatisticTitleBoxCls">
                                <div class="newIconBoxCls-alarmTitle"><span class="el-icon el-icon-dash_wifi" ></span></div>
                                <span class="statisticItemNameCls"><%=rb.getString("XiaoQuQuJiHuoGaoJing")%></span>
                            </div>
                            <div class="alarmStatisticValueBoxCls" style="color: #FF973E;">
                                <span @click="showCurrAliveAlarm('cellInactiveAlarm')">{{alarmStatisticData.unavailable_alarm_count}}</span> 
                            </div>
                        </div>
                    </el-card>
                </div>
            </div>
        </div>

        <!-- Device Status chart -->
        <div class="board-item"> 
            <div class="dashboaedChart">
                <div class="boardItemTitle">
                    <div class="newIconBoxCls-title"><span class="el-icon el-icon-menu-eNB" ></span></div>
                    <span><%=rb.getString("SheBeiZhuangTai")%></span>
                    <span @click="deviceAndKpiStatisticExport" class="inner-export el-icon el-icon-operation-export placeholder-bt" tip='<%=rb.getString("DaoChuSheBeiZhuangTaiHeKPI")%>'></span>
                </div>
                <el-dropdown @command="enbStatisticChangeClick" trigger="click" class="statisticChangeDropdownCls" v-show="deviceStatusActiveTab==='eNB'">
                    <span style="cursor: pointer;">{{enbStatisticNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                    <el-dropdown-menu slot="dropdown">
                        <el-dropdown-item command="enbOnline"><%=rb.getString("ShouYe_ZaiXian")%></el-dropdown-item>
                        <el-dropdown-item command="enbActive"><%=rb.getString("ShouYe_HuoYue")%></el-dropdown-item>
                        <el-dropdown-item command="enbUeCount"><%=rb.getString("ShouYe_UEShu")%></el-dropdown-item>
                        <el-dropdown-item command="enbMmeStatus"><%=rb.getString("MMEZhuangTai")%></el-dropdown-item>
                        <el-dropdown-item command="enbProductType"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></el-dropdown-item>
                        <el-dropdown-item command="enbDeviceRunningTime"><%=rb.getString("SheBeiYunXingShiChang")%></el-dropdown-item>
                    </el-dropdown-menu>
                </el-dropdown>
                <el-dropdown @command="gnbStatisticChangeClick" trigger="click" class="statisticChangeDropdownCls" v-show="deviceStatusActiveTab==='gNB'">
                    <span style="cursor: pointer;">{{gnbStatisticNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                    <el-dropdown-menu slot="dropdown">
                        <el-dropdown-item command="gnbOnline"><%=rb.getString("ShouYe_ZaiXian")%></el-dropdown-item>
                        <el-dropdown-item command="gnbActive"><%=rb.getString("ShouYe_HuoYue")%></el-dropdown-item>
                        <el-dropdown-item command="gnbUeCount"><%=rb.getString("ShouYe_UEShu")%></el-dropdown-item>
                        <el-dropdown-item command="gnbProductType"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></el-dropdown-item>
                        <el-dropdown-item command="gnbDeviceRunningTime"><%=rb.getString("SheBeiYunXingShiChang")%></el-dropdown-item>
                    </el-dropdown-menu>
                </el-dropdown>
                <el-dropdown @command="gsmStatisticChangeClick" trigger="click" class="statisticChangeDropdownCls" v-show="deviceStatusActiveTab==='GSM'">
                    <span style="cursor: pointer;">{{gsmStatisticNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                    <el-dropdown-menu slot="dropdown">
                        <el-dropdown-item command="gsmOnline"><%=rb.getString("ShouYe_ZaiXian")%></el-dropdown-item>
                        <el-dropdown-item command="gsmActive"><%=rb.getString("ShouYe_HuoYue")%></el-dropdown-item>
                        <el-dropdown-item command="gsmUeCount"><%=rb.getString("ShouYe_UEShu")%></el-dropdown-item>
                        <el-dropdown-item command="gsmProductType"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></el-dropdown-item>
                        <el-dropdown-item command="gsmDeviceRunningTime"><%=rb.getString("SheBeiYunXingShiChang")%></el-dropdown-item>
                    </el-dropdown-menu>
                </el-dropdown>
                <el-dropdown @command="cpeStatisticChangeClick" trigger="click" class="statisticChangeDropdownCls" v-show="deviceStatusActiveTab==='CPE'">
                    <span style="cursor: pointer;">{{cpeStatisticNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                    <el-dropdown-menu slot="dropdown">
                        <el-dropdown-item command="cpeOnline"><%=rb.getString("ShouYe_ZaiXian")%></el-dropdown-item>
                        <el-dropdown-item command="cpeProductType"><%=rb.getString("ChanPinXingHao")%></el-dropdown-item>
                        <el-dropdown-item command="cpeDeviceRunningTime"><%=rb.getString("SheBeiYunXingShiChang")%></el-dropdown-item>
                    </el-dropdown-menu>
                </el-dropdown>
                <el-tabs class="fit newTabs" v-model="deviceStatusActiveTab" style='height:calc(100% - 46px)' @tab-click="deviceStatusTabChange">
                    <el-tab-pane v-if="visibleCode.siteStatistic" label='<%=rb.getString("ZhanDian")%>' name="Site" key="Site">
                        <div slot="label" class="tabCustomLabelCls">
                            <span class="el-icon el-icon-dash_site" ></span>
                            <span><%=rb.getString("ZhanDian")%></span>
                        </div>
                        <div class="statisticItemCls">
                            <el-card shadow="hover" class="chart-card">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZongShu")%></span>
                                            <span>{{lineNowTotalAndStatus('siteOffline','total')}}</span>
                                            <span v-if="lineNowTotalAndStatus('siteOffline','total_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('siteOffline','total_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_BuZaiXian")%></span>
                                            <span>{{lineNowTotalAndStatus('siteOffline','offline_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('siteOffline','offline_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('siteOffline','offline_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="siteOffline" :class="chartLoadingData.siteOffline ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                        </div>
                    </el-tab-pane>
				    <el-tab-pane v-if="visibleCode.enbStatistic" label="<%=rb.getString("PeiZhiGuanLi")%>" name="eNB" key="eNB">
                        <div slot="label" class="tabCustomLabelCls">
                            <span class="el-icon el-icon-menu-eNB" ></span>
                            <span>eNB</span>
                        </div>
                        <div class="statisticItemCls">
                            <el-card shadow="hover" class="chart-card" v-show="enbStatisticSelect =='enbOnline'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZongShu")%></span>
                                            <span>{{lineNowTotalAndStatus('enbOnline','device_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('enbOnline','device_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbOnline','device_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZaiXian")%></span>
                                            <span>{{lineNowTotalAndStatus('enbOnline','online_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('enbOnline','online_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbOnline','online_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ZaiXianLv")%></span>
                                            <span>{{lineNowTotalAndStatus('enbOnline','online_rate')}}%</span>
                                            <span v-if="lineNowTotalAndStatus('enbOnline','online_rate_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbOnline','online_rate_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="enbOnline" :class="chartLoadingData.enbOnline ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="enbStatisticSelect =='enbActive'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZongShu")%></span>
                                            <span>{{lineNowTotalAndStatus('enbActive','device_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('enbActive','device_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbActive','device_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_HuoYue")%></span>
                                            <span>{{lineNowTotalAndStatus('enbActive','active_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('enbActive','active_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbActive','active_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("JiHuoLv")%></span>
                                            <span>{{lineNowTotalAndStatus('enbActive','active_rate')}}%</span>
                                            <span v-if="lineNowTotalAndStatus('enbActive','active_rate_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbActive','active_rate_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="enbActive" :class="chartLoadingData.enbActive ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="enbStatisticSelect =='enbUeCount'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_UEShu")%></span>
                                            <span>{{lineNowTotalAndStatus('enbUeCount','ue_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('enbUeCount','ue_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbUeCount','ue_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="enbUeCount" :class="chartLoadingData.enbUeCount ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="enbStatisticSelect =='enbMmeStatus'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_LianJie")%></span>
                                            <span>{{lineNowTotalAndStatus('enbMmeStatus','mme_connected_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('enbMmeStatus','mme_connected_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbMmeStatus','mme_connected_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_FeiLianJie")%></span>
                                            <span>{{lineNowTotalAndStatus('enbMmeStatus','mme_disconnected_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('enbMmeStatus','mme_disconnected_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('enbMmeStatus','mme_disconnected_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        
                                    </div>
                                </div>
                                <div id="enbMmeStatus" :class="chartLoadingData.enbMmeStatus ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="enbStatisticSelect =='enbProductType'">
                                <div id="enbProductType" :class="chartLoadingData.enbProductType ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="enbStatisticSelect =='enbDeviceRunningTime'">
                                <div id="enbDeviceRunningTime" :class="chartLoadingData.enbDeviceRunningTime ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                        </div>
                    </el-tab-pane>
                    <el-tab-pane v-if="visibleCode.gnbStatistic" label="gNB" name="gNB" key="gNB">
                        <div slot="label" class="tabCustomLabelCls">
                            <span class="el-icon el-icon-menu-5G" ></span>
                            <span>gNB</span>
                        </div>
                        <div class="statisticItemCls">
                            <el-card shadow="hover" class="chart-card" v-show="gnbStatisticSelect =='gnbOnline'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZongShu")%></span>
                                            <span>{{lineNowTotalAndStatus('gnbOnline','device_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gnbOnline','device_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gnbOnline','device_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZaiXian")%></span>
                                            <span>{{lineNowTotalAndStatus('gnbOnline','online_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gnbOnline','online_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gnbOnline','online_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ZaiXianLv")%></span>
                                            <span>{{lineNowTotalAndStatus('gnbOnline','online_rate')}}%</span>
                                            <span v-if="lineNowTotalAndStatus('gnbOnline','online_rate_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gnbOnline','online_rate_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="gnbOnline" :class="chartLoadingData.gnbOnline ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="gnbStatisticSelect =='gnbActive'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZongShu")%></span>
                                            <span>{{lineNowTotalAndStatus('gnbActive','device_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gnbActive','device_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gnbActive','device_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_HuoYue")%></span>
                                            <span>{{lineNowTotalAndStatus('gnbActive','active_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gnbActive','active_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gnbActive','active_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("JiHuoLv")%></span>
                                            <span>{{lineNowTotalAndStatus('gnbActive','active_rate')}}%</span>
                                            <span v-if="lineNowTotalAndStatus('gnbActive','active_rate_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gnbActive','active_rate_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="gnbActive" :class="chartLoadingData.gnbActive ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="gnbStatisticSelect =='gnbUeCount'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_UEShu")%></span>
                                            <span>{{lineNowTotalAndStatus('gnbUeCount','ue_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gnbUeCount','ue_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gnbUeCount','ue_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="gnbUeCount" :class="chartLoadingData.gnbUeCount ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="gnbStatisticSelect =='gnbProductType'">
                                <div id="gnbProductType" :class="chartLoadingData.gnbProductType ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="gnbStatisticSelect =='gnbDeviceRunningTime'">
                                <div id="gnbDeviceRunningTime" :class="chartLoadingData.gnbDeviceRunningTime ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                        </div>
                    </el-tab-pane>
                    <el-tab-pane v-if="visibleCode.gsmStatistic" label="GSM" name="GSM" key="GSM">
                        <div slot="label" class="tabCustomLabelCls">
                            <span class="el-icon el-icon-GSM" ></span>
                            <span>GSM</span>
                        </div>
                        <div class="statisticItemCls">
                            <el-card shadow="hover" class="chart-card" v-show="gsmStatisticSelect =='gsmOnline'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZongShu")%></span>
                                            <span>{{lineNowTotalAndStatus('gsmOnline','device_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gsmOnline','device_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gsmOnline','device_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZaiXian")%></span>
                                            <span>{{lineNowTotalAndStatus('gsmOnline','online_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gsmOnline','online_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gsmOnline','online_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ZaiXianLv")%></span>
                                            <span>{{lineNowTotalAndStatus('gsmOnline','online_rate')}}%</span>
                                            <span v-if="lineNowTotalAndStatus('gsmOnline','online_rate_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gsmOnline','online_rate_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="gsmOnline" :class="chartLoadingData.gsmOnline ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="gsmStatisticSelect =='gsmActive'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZongShu")%></span>
                                            <span>{{lineNowTotalAndStatus('gsmActive','device_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gsmActive','device_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gsmActive','device_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_HuoYue")%></span>
                                            <span>{{lineNowTotalAndStatus('gsmActive','active_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gsmActive','active_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gsmActive','active_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("JiHuoLv")%></span>
                                            <span>{{lineNowTotalAndStatus('gsmActive','active_rate')}}%</span>
                                            <span v-if="lineNowTotalAndStatus('gsmActive','active_rate_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gsmActive','active_rate_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="gsmActive" :class="chartLoadingData.gsmActive ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="gsmStatisticSelect =='gsmUeCount'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_UEShu")%></span>
                                            <span>{{lineNowTotalAndStatus('gsmUeCount','ue_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('gsmUeCount','ue_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('gsmUeCount','ue_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="gsmUeCount" :class="chartLoadingData.gsmUeCount ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="gsmStatisticSelect =='gsmProductType'">
                                <div id="gsmProductType" :class="chartLoadingData.gsmProductType ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="gsmStatisticSelect =='gsmDeviceRunningTime'">
                                <div id="gsmDeviceRunningTime" :class="chartLoadingData.gsmDeviceRunningTime ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                        </div>
                    </el-tab-pane>   
                    <el-tab-pane v-if="visibleCode.cpeStatistic" label="CPE" name="CPE" key="CPE">
                        <div slot="label" class="tabCustomLabelCls">
                            <span class="el-icon el-icon-menu-CPE" ></span>
                            <span>CPE</span>
                        </div>
                        <div class="statisticItemCls">
                            <el-card shadow="hover" class="chart-card" v-show="cpeStatisticSelect =='cpeOnline'">
                                <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls">
                                    <div class="statisticChangeBoxCls">
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZongShu")%></span>
                                            <span>{{lineNowTotalAndStatus('cpeOnline','device_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('cpeOnline','device_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('cpeOnline','device_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ShouYe_ZaiXian")%></span>
                                            <span>{{lineNowTotalAndStatus('cpeOnline','online_count')}}</span>
                                            <span v-if="lineNowTotalAndStatus('cpeOnline','online_count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('cpeOnline','online_count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                        <div class="statisticChangeItemCls">
                                            <span><%=rb.getString("ZaiXianLv")%></span>
                                            <span>{{lineNowTotalAndStatus('cpeOnline','online_rate')}}%</span>
                                            <span v-if="lineNowTotalAndStatus('cpeOnline','online_rate_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                            <span v-if="lineNowTotalAndStatus('cpeOnline','online_rate_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                        </div>
                                    </div>
                                </div>
                                <div id="cpeOnline" :class="chartLoadingData.cpeOnline ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="cpeStatisticSelect =='cpeProductType'">
                                <div id="cpeProductType" :class="chartLoadingData.cpeProductType ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                            <el-card shadow="hover" class="chart-card" v-show="cpeStatisticSelect =='cpeDeviceRunningTime'">
                                <div id="cpeDeviceRunningTime" :class="chartLoadingData.cpeDeviceRunningTime ? 'loading' : ''" style="width: 100%;height: 300px;" ></div>
                            </el-card>
                        </div>
                    </el-tab-pane>     
                </el-tabs>
            </div>
        </div>
        <!-- kpi chart -->
        <div class="board-item" v-show="visibleCode.kpiStatistic">
            <div class="dashboaedChart">
                <div class="boardItemTitle">
                    <div class="newIconBoxCls-title"><span class="el-icon el-icon-KPI-view" ></span></div>
                    <span>KPI</span>
                    <span @click="deviceAndKpiStatisticExport" class="inner-export el-icon el-icon-operation-export placeholder-bt" tip='<%=rb.getString("DaoChuSheBeiZhuangTaiHeKPI")%>'></span>
                </div>
                <el-tabs class="fit newTabs" v-model="kpiStatisticActiveTab" style='height:calc(100% - 46px)' @tab-click="kpiStatisticTabChange">
				    <el-tab-pane v-if="visibleCode.enbStatistic" label="<%=rb.getString("PeiZhiGuanLi")%>" name="eNB" key="eNB">
                        <div slot="label" class="tabCustomLabelCls">
                            <span class="el-icon el-icon-menu-eNB" ></span>
                            <span>eNB</span>
                        </div>
                        <div class="kpiFeatureBoxCls">
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("YeWuLiang")%></span>
                                    <el-dropdown @command="enbKpiTrafficChangeClick" trigger="click" class="statisticChangeDropdownCls" style="top: 15px;right: 20px;">
                                        <span style="cursor: pointer;">{{enbKpiTrafficNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                                        <el-dropdown-menu slot="dropdown">
                                            <el-dropdown-item command="enbTotalDataVolumeDL">Total Data Volume DL</el-dropdown-item>
                                            <el-dropdown-item command="enbTotalDataVolumeUL">Total Data Volume UL</el-dropdown-item>
                                            <el-dropdown-item command="enbThroughputDL">Throughput DL</el-dropdown-item>
                                            <el-dropdown-item command="enbThroughputUL">Throughput UL</el-dropdown-item>
                                        </el-dropdown-menu>
                                    </el-dropdown>
                                </div>
                                <div class="statisticItemCls">
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiTrafficStatisticSelect =='enbTotalDataVolumeDL'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Total Data Volume DL</span>
                                                    <span>{{lineNowTotalAndStatus('enbTotalDataVolumeDL','count')}}GB</span>
                                                    <span v-if="lineNowTotalAndStatus('enbTotalDataVolumeDL','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbTotalDataVolumeDL','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbTotalDataVolumeDL')" size="mini" v-model="periodTypeData.enbTotalDataVolumeDL" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbTotalDataVolumeDL" :class="chartLoadingData.enbTotalDataVolumeDL ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbTotalDataVolumeUL -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiTrafficStatisticSelect =='enbTotalDataVolumeUL'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Total Data Volume UL</span>
                                                    <span>{{lineNowTotalAndStatus('enbTotalDataVolumeUL','count')}}GB</span>
                                                    <span v-if="lineNowTotalAndStatus('enbTotalDataVolumeUL','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbTotalDataVolumeUL','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbTotalDataVolumeUL')" size="mini" v-model="periodTypeData.enbTotalDataVolumeUL" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbTotalDataVolumeUL" :class="chartLoadingData.enbTotalDataVolumeUL ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbThroughputDL -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiTrafficStatisticSelect =='enbThroughputDL'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Throughput DL</span>
                                                    <span>{{lineNowTotalAndStatus('enbThroughputDL','count')}}Mbps</span>
                                                    <span v-if="lineNowTotalAndStatus('enbThroughputDL','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbThroughputDL','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbThroughputDL')" size="mini" v-model="periodTypeData.enbThroughputDL" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbThroughputDL" :class="chartLoadingData.enbThroughputDL ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbThroughputUL -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiTrafficStatisticSelect =='enbThroughputUL'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Throughput UL</span>
                                                    <span>{{lineNowTotalAndStatus('enbThroughputUL','count')}}Mbps</span>
                                                    <span v-if="lineNowTotalAndStatus('enbThroughputUL','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbThroughputUL','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbThroughputUL')" size="mini" v-model="periodTypeData.enbThroughputUL" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbThroughputUL" :class="chartLoadingData.enbThroughputUL ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("KeYongXing")%></span>
                                </div>
                                <div class="statisticItemCls">
                                    <el-card shadow="hover" class="chart-card">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Cell Available</span>
                                                    <span>{{lineNowTotalAndStatus('enbCellAvailable','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbCellAvailable','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbCellAvailable','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbCellAvailable')" size="mini" v-model="periodTypeData.enbCellAvailable" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbCellAvailable" :class="chartLoadingData.enbCellAvailable ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("ShiYongLv")%></span>
                                    <el-dropdown @command="enbKpiUtilizationChangeClick" trigger="click" class="statisticChangeDropdownCls" style="top: 15px;right: 20px;">
                                        <span style="cursor: pointer;">{{enbKpiUtilizationNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                                        <el-dropdown-menu slot="dropdown">
                                            <el-dropdown-item command="enbDownlinkPRBUtilizationRate">Downlink PRB Utilization Rate</el-dropdown-item>
                                            <el-dropdown-item command="enbUplinkPRBUtilizationRate">Uplink PRB Utilization Rate</el-dropdown-item>
                                        </el-dropdown-menu>
                                    </el-dropdown>
                                </div>
                                <div class="statisticItemCls">
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiUtilizationStatisticSelect =='enbDownlinkPRBUtilizationRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Downlink PRB Utilization Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbDownlinkPRBUtilizationRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbDownlinkPRBUtilizationRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbDownlinkPRBUtilizationRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbDownlinkPRBUtilizationRate')" size="mini" v-model="periodTypeData.enbDownlinkPRBUtilizationRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbDownlinkPRBUtilizationRate" :class="chartLoadingData.enbDownlinkPRBUtilizationRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbUplinkPRBUtilizationRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiUtilizationStatisticSelect =='enbUplinkPRBUtilizationRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Uplink PRB Utilization Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbUplinkPRBUtilizationRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbUplinkPRBUtilizationRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbUplinkPRBUtilizationRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbUplinkPRBUtilizationRate')" size="mini" v-model="periodTypeData.enbUplinkPRBUtilizationRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbUplinkPRBUtilizationRate" :class="chartLoadingData.enbUplinkPRBUtilizationRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("JieRuXing")%></span>
                                    <el-dropdown @command="enbKpiAccessibilityChangeClick" trigger="click" class="statisticChangeDropdownCls" style="top: 15px;right: 20px;">
                                        <span style="cursor: pointer;">{{enbKpiAccessibilityNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                                        <el-dropdown-menu slot="dropdown">
                                            <el-dropdown-item command="enbWirelessSetupSuccessRate">Wireless Setup Success Rate</el-dropdown-item>
                                            <el-dropdown-item command="enbRrcSetupSuccessRate">RRC Setup Success Rate</el-dropdown-item>
                                            <el-dropdown-item command="enbERABSetupSuccessRate">E-RAB Setup Success Rate</el-dropdown-item>
                                            <el-dropdown-item command="enbCsfbSuccessRate">CSFB Success Rate</el-dropdown-item>
                                        </el-dropdown-menu>
                                    </el-dropdown>
                                </div>
                                <div class="statisticItemCls">
                                    <!-- enbWirelessSetupSuccessRate-->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiAccessibilityStatisticSelect =='enbWirelessSetupSuccessRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Wireless Setup Success Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbWirelessSetupSuccessRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbWirelessSetupSuccessRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbWirelessSetupSuccessRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbWirelessSetupSuccessRate')" size="mini" v-model="periodTypeData.enbWirelessSetupSuccessRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbWirelessSetupSuccessRate" :class="chartLoadingData.enbWirelessSetupSuccessRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbRrcSetupSuccessRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiAccessibilityStatisticSelect =='enbRrcSetupSuccessRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>RRC Setup Success Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbRrcSetupSuccessRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbRrcSetupSuccessRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbRrcSetupSuccessRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbRrcSetupSuccessRate')" size="mini" v-model="periodTypeData.enbRrcSetupSuccessRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbRrcSetupSuccessRate" :class="chartLoadingData.enbRrcSetupSuccessRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbERABSetupSuccessRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiAccessibilityStatisticSelect =='enbERABSetupSuccessRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>E-RAB Setup Success Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbERABSetupSuccessRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbERABSetupSuccessRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbERABSetupSuccessRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbERABSetupSuccessRate')" size="mini" v-model="periodTypeData.enbERABSetupSuccessRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbERABSetupSuccessRate" :class="chartLoadingData.enbERABSetupSuccessRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbCsfbSuccessRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiAccessibilityStatisticSelect =='enbCsfbSuccessRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>CSFB Success Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbCsfbSuccessRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbCsfbSuccessRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbCsfbSuccessRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbCsfbSuccessRate')" size="mini" v-model="periodTypeData.enbCsfbSuccessRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbCsfbSuccessRate" :class="chartLoadingData.enbCsfbSuccessRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("BaoChiXing")%></span>
                                </div>
                                <div class="statisticItemCls">
                                    <!-- enbERABDropRate -->
                                    <el-card shadow="hover" class="chart-card">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>E-RAB Drop Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbERABDropRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbERABDropRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbERABDropRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbERABDropRate')" size="mini" v-model="periodTypeData.enbERABDropRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbERABDropRate" :class="chartLoadingData.enbERABDropRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("YiDongXing")%></span>
                                    <el-dropdown @command="enbKpiMobilityChangeClick" trigger="click" class="statisticChangeDropdownCls" style="top: 15px;right: 20px;">
                                        <span style="cursor: pointer;">{{enbKpiMobilityNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                                        <el-dropdown-menu slot="dropdown">
                                            <el-dropdown-item command="enbHoIntraEnbOutSuccRate">HO IntraEnbOutSucc Rate</el-dropdown-item>
                                            <el-dropdown-item command="enbHoIntraEnbInSuccRate">HO IntraEnbInSucc Rate</el-dropdown-item>
                                            <el-dropdown-item command="enbHoInterEnbOutSuccRate">HO InterEnbOutSucc Rate</el-dropdown-item>
                                            <el-dropdown-item command="enbHoInterEnbInSuccRate">HO InterEnbInSucc Rate</el-dropdown-item>
                                        </el-dropdown-menu>
                                    </el-dropdown>
                                </div>
                                <div class="statisticItemCls">
                                    <!-- enbHoIntraEnbOutSuccRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiMobilityStatisticSelect =='enbHoIntraEnbOutSuccRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>HO IntraEnbOutSucc Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbHoIntraEnbOutSuccRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbHoIntraEnbOutSuccRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbHoIntraEnbOutSuccRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbHoIntraEnbOutSuccRate')" size="mini" v-model="periodTypeData.enbHoIntraEnbOutSuccRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbHoIntraEnbOutSuccRate" :class="chartLoadingData.enbHoIntraEnbOutSuccRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbHoIntraEnbInSuccRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiMobilityStatisticSelect =='enbHoIntraEnbInSuccRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>HO IntraEnbInSucc Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbHoIntraEnbInSuccRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbHoIntraEnbInSuccRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbHoIntraEnbInSuccRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbHoIntraEnbInSuccRate')" size="mini" v-model="periodTypeData.enbHoIntraEnbInSuccRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbHoIntraEnbInSuccRate" :class="chartLoadingData.enbHoIntraEnbInSuccRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbHoInterEnbOutSuccRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiMobilityStatisticSelect =='enbHoInterEnbOutSuccRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>HO InterEnbOutSucc Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbHoInterEnbOutSuccRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbHoInterEnbOutSuccRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbHoInterEnbOutSuccRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbHoInterEnbOutSuccRate')" size="mini" v-model="periodTypeData.enbHoInterEnbOutSuccRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbHoInterEnbOutSuccRate" :class="chartLoadingData.enbHoInterEnbOutSuccRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- enbHoInterEnbInSuccRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="enbKpiMobilityStatisticSelect =='enbHoInterEnbInSuccRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>HO InterEnbInSucc Rate</span>
                                                    <span>{{lineNowTotalAndStatus('enbHoInterEnbInSuccRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('enbHoInterEnbInSuccRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('enbHoInterEnbInSuccRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('enbHoInterEnbInSuccRate')" size="mini" v-model="periodTypeData.enbHoInterEnbInSuccRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="enbHoInterEnbInSuccRate" :class="chartLoadingData.enbHoInterEnbInSuccRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                        </div>
                    </el-tab-pane>
                    <el-tab-pane v-if="visibleCode.gnbStatistic" label="gNB" name="gNB" key="gNB">
                        <div slot="label" class="tabCustomLabelCls">
                            <span class="el-icon el-icon-menu-5G" ></span>
                            <span>gNB</span>
                        </div>
                        <div class="kpiFeatureBoxCls">
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("YeWuLiang")%></span>
                                    <el-dropdown @command="gnbKpiTrafficChangeClick" trigger="click" class="statisticChangeDropdownCls" style="top: 15px;right: 20px;">
                                        <span style="cursor: pointer;">{{gnbKpiTrafficNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                                        <el-dropdown-menu slot="dropdown">
                                            <!-- gnbPdcpUpOctDL  gnbPdcpUpOctUL gnbThroughputDL gnbThroughputUL -->
                                            <el-dropdown-item command="gnbPdcpUpOctDL">KPI.PdcpUpOctDL</el-dropdown-item>
                                            <el-dropdown-item command="gnbPdcpUpOctUL">KPI.PdcpUpOctUL</el-dropdown-item>
                                            <el-dropdown-item command="gnbThroughputDL">Throughput DL</el-dropdown-item>
                                            <el-dropdown-item command="gnbThroughputUL">Throughput UL</el-dropdown-item>
                                        </el-dropdown-menu>
                                    </el-dropdown>
                                </div>
                                <div class="statisticItemCls">
                                    <!-- gnbPdcpUpOctDL -->
                                    <el-card shadow="hover" class="chart-card" v-show="gnbKpiTrafficStatisticSelect =='gnbPdcpUpOctDL'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>KPI.PdcpUpOctDL</span>
                                                    <span>{{lineNowTotalAndStatus('gnbPdcpUpOctDL','count')}}GB</span>
                                                    <span v-if="lineNowTotalAndStatus('gnbPdcpUpOctDL','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gnbPdcpUpOctDL','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gnbPdcpUpOctDL')" size="mini" v-model="periodTypeData.gnbPdcpUpOctDL" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gnbPdcpUpOctDL" :class="chartLoadingData.gnbPdcpUpOctDL ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- gnbPdcpUpOctUL -->
                                    <el-card shadow="hover" class="chart-card" v-show="gnbKpiTrafficStatisticSelect =='gnbPdcpUpOctUL'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>KPI.PdcpUpOctUL</span>
                                                    <span>{{lineNowTotalAndStatus('gnbPdcpUpOctUL','count')}}GB</span>
                                                    <span v-if="lineNowTotalAndStatus('gnbPdcpUpOctUL','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gnbPdcpUpOctUL','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gnbPdcpUpOctUL')" size="mini" v-model="periodTypeData.gnbPdcpUpOctUL" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gnbPdcpUpOctUL" :class="chartLoadingData.gnbPdcpUpOctUL ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- gnbThroughputDL -->
                                    <el-card shadow="hover" class="chart-card" v-show="gnbKpiTrafficStatisticSelect =='gnbThroughputDL'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Throughput DL</span>
                                                    <span>{{lineNowTotalAndStatus('gnbThroughputDL','count')}}Mbps</span>
                                                    <span v-if="lineNowTotalAndStatus('gnbThroughputDL','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gnbThroughputDL','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gnbThroughputDL')" size="mini" v-model="periodTypeData.gnbThroughputDL" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gnbThroughputDL" :class="chartLoadingData.gnbThroughputDL ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- gnbThroughputUL -->
                                    <el-card shadow="hover" class="chart-card" v-show="gnbKpiTrafficStatisticSelect =='gnbThroughputUL'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Throughput UL</span>
                                                    <span>{{lineNowTotalAndStatus('gnbThroughputUL','count')}}Mbps</span>
                                                    <span v-if="lineNowTotalAndStatus('gnbThroughputUL','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gnbThroughputUL','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gnbThroughputUL')" size="mini" v-model="periodTypeData.gnbThroughputUL" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gnbThroughputUL" :class="chartLoadingData.gnbThroughputUL ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("ShiYongLv")%></span>
                                    <el-dropdown @command="gnbKpiUtilizationChangeClick" trigger="click" class="statisticChangeDropdownCls" style="top: 15px;right: 20px;">
                                        <span style="cursor: pointer;">{{gnbKpiUtilizationNowTitle}}<i class="el-icon-arrow-down el-icon--right"></i></span>
                                        <el-dropdown-menu slot="dropdown">
                                            <el-dropdown-item command="gnbDownlinkPRBUtilizationRate">Downlink PRB Utilization Rate</el-dropdown-item>
                                            <el-dropdown-item command="gnbUplinkPRBUtilizationRate">Uplink PRB Utilization Rate</el-dropdown-item>
                                        </el-dropdown-menu>
                                    </el-dropdown>
                                </div>
                                <div class="statisticItemCls">
                                    <!-- gnbDownlinkPRBUtilizationRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="gnbKpiUtilizationStatisticSelect =='gnbDownlinkPRBUtilizationRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Downlink PRB Utilization Rate</span>
                                                    <span>{{lineNowTotalAndStatus('gnbDownlinkPRBUtilizationRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('gnbDownlinkPRBUtilizationRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gnbDownlinkPRBUtilizationRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gnbDownlinkPRBUtilizationRate')" size="mini" v-model="periodTypeData.gnbDownlinkPRBUtilizationRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gnbDownlinkPRBUtilizationRate" :class="chartLoadingData.gnbDownlinkPRBUtilizationRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                    <!-- gnbUplinkPRBUtilizationRate -->
                                    <el-card shadow="hover" class="chart-card" v-show="gnbKpiUtilizationStatisticSelect =='gnbUplinkPRBUtilizationRate'">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>Uplink PRB Utilization Rate</span>
                                                    <span>{{lineNowTotalAndStatus('gnbUplinkPRBUtilizationRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('gnbUplinkPRBUtilizationRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gnbUplinkPRBUtilizationRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gnbUplinkPRBUtilizationRate')" size="mini" v-model="periodTypeData.gnbUplinkPRBUtilizationRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gnbUplinkPRBUtilizationRate" :class="chartLoadingData.gnbUplinkPRBUtilizationRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                        </div>
                    </el-tab-pane>
                    <el-tab-pane v-if="visibleCode.gsmStatistic" label="GSM" name="GSM" key="GSM">
                        <div slot="label" class="tabCustomLabelCls">
                            <span class="el-icon el-icon-GSM" ></span>
                            <span>GSM</span>
                        </div>
                        <div class="kpiFeatureBoxCls">
                             <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("JieRuXing")%></span>
                                </div>
                                <div class="statisticItemCls">
                                    <!-- gsmCallSetupSuccRate -->
                                    <el-card shadow="hover" class="chart-card">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>KPI.CallSetupSuccRate</span>
                                                    <span>{{lineNowTotalAndStatus('gsmCallSetupSuccRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('gsmCallSetupSuccRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gsmCallSetupSuccRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gsmCallSetupSuccRate')" size="mini" v-model="periodTypeData.gsmCallSetupSuccRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gsmCallSetupSuccRate" :class="chartLoadingData.gsmCallSetupSuccRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("BaoChiXing")%></span>
                                </div>
                                <div class="statisticItemCls">
                                    <!-- gsmCallDropRate -->
                                    <el-card shadow="hover" class="chart-card">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>KPI.CallDropRate</span>
                                                    <span>{{lineNowTotalAndStatus('gsmCallDropRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('gsmCallDropRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gsmCallDropRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gsmCallDropRate')" size="mini" v-model="periodTypeData.gsmCallDropRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gsmCallDropRate" :class="chartLoadingData.gsmCallDropRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                            <div class="kpiFeatureItemBoxCls">
                                <div class="kpiFeatureHeaderBoxCls">
                                    <span class="kpiFeatureIconCls"></span>
                                    <span class="kpiFeatureTtileCls"><%=rb.getString("YiDongXing")%></span>
                                </div>
                                <div class="statisticItemCls">
                                    <!-- gsmHandoverSuccessRate -->
                                    <el-card shadow="hover" class="chart-card">
                                        <div class="statisticItemHeaderCls statisticItemHeaderAbsoluteCls" style="height: 70px;">
                                            <div class="statisticChangeBoxCls">
                                                <div class="statisticChangeItemCls">
                                                    <span>KPI.HandoverSuccessRate</span>
                                                    <span>{{lineNowTotalAndStatus('gsmHandoverSuccessRate','count')}}%</span>
                                                    <span v-if="lineNowTotalAndStatus('gsmHandoverSuccessRate','count_status') == '1'" class="el-icon el-icon-compare_up upIconCls"></span>
                                                    <span v-if="lineNowTotalAndStatus('gsmHandoverSuccessRate','count_status') == '2'" class="el-icon el-icon-compare_up downIconCls rotate180"></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dashboardDayAndWeekChangeBoxCls">
                                            <el-radio-group @change="periodTypeChange('gsmHandoverSuccessRate')" size="mini" v-model="periodTypeData.gsmHandoverSuccessRate" class="commonRadioButton">
                                                <el-radio-button label="min"><%=rb.getString("Tian")%></el-radio-button>
                                                <el-radio-button label="week"><%=rb.getString("Zhou")%></el-radio-button>
                                            </el-radio-group>
                                        </div>
                                        <div id="gsmHandoverSuccessRate" :class="chartLoadingData.gsmHandoverSuccessRate ? 'loading' : ''" style="width: 100%;height: 300px;"></div>
                                    </el-card>
                                </div>
                            </div>
                        </div>
                    </el-tab-pane>  
                </el-tabs>
            </div>
        </div>
        <!-- SAS chart -->
        <div class="board-item" style="margin-bottom: 0px;" v-show="visibleCode.SASStatistic && false">
            <div class="boardItemTitle">
                <span>SAS</span>
                <div class="newIconBoxCls-bt" style="right:10px;top:5px;" @click="goProcedure">
                    <span class="el-icon el-icon-operation-details"></span>
                </div>
            </div>
            <div  style="display: flex;flex-wrap: wrap;" class="dashboaedChart">
                <div class="statisticItemCls">
                    <el-card shadow="hover" class="chart-card" v-show="visibleCode.enbStatistic">
                        <span style="position:absolute;left:20px;top:10px;"><%=rb.getString("PeiZhiGuanLi")%></span>
                        <div id="enbSASStatistic" style="width: 100%;height: 270px;"></div>
                    </el-card>
                    <el-card shadow="hover" class="chart-card" v-show="visibleCode.gnbStatistic">
                        <span style="position:absolute;left:20px;top:10px;"><%=rb.getString("gNB")%></span>
                        <div id="gnbSASStatistic" style="width: 100%;height: 270px;"></div>
                    </el-card>
                    <el-card shadow="hover" class="chart-card" v-show="visibleCode.cpeStatistic">
                        <span style="position:absolute;left:20px;top:10px;">CPE</span>
                        <div id="cpeSASStatistic" style="width: 100%;height: 270px;"></div>
                    </el-card>
                </div>
            </div>
        </div>
    </div>
</div>
<script>
closeLoading();
if(window.homevm) {
	try {
		window.homevm.$destroy();
	}catch(e){}
}
window.homevm = new Vue({
	el: '#dashboard_ctn',
	data(){
		return  {
            deviceStatusActiveTab: 'Site',
            kpiStatisticActiveTab: 'eNB',
            enbStatisticSelect:'enbOnline',
            gnbStatisticSelect:'gnbOnline',
            gsmStatisticSelect:'gsmOnline',
            cpeStatisticSelect:'cpeOnline',
            enbKpiTrafficStatisticSelect:'enbTotalDataVolumeDL',
            enbKpiUtilizationStatisticSelect:'enbDownlinkPRBUtilizationRate',
            enbKpiAccessibilityStatisticSelect:'enbWirelessSetupSuccessRate',
            enbKpiMobilityStatisticSelect:'enbHoIntraEnbOutSuccRate',
            gnbKpiTrafficStatisticSelect:'gnbPdcpUpOctDL',
            gnbKpiUtilizationStatisticSelect:'gnbDownlinkPRBUtilizationRate',
            customBoxExpand: true,
			customData:{
				alarm: true,
				site: true,
				enbOnlineOrActive: true,
				enbMmeStatus: true,
				enbUeCount: true,
				enbProductType: true,
				enbDeviceRunningTime: true,
				gnbOnlineOrActive: true,
				gnbUeCount: true,
				gnbProductType: true,
				gnbDeviceRunningTime: true,
				gsmOnlineOrActive: true,
				gsmUeCount: true,
				gsmProductType: true,
				gsmDeviceRunningTime: true,
				cpeOnline: true,
				cpeOnlineRate: true,
				cpeProductType: true,
				cpeDeviceRunningTime: true,
                enbWirelessSetupSuccessRate: true,
				enbTotalDataVolumeDL: true,
				enbTotalDataVolumeUL: true,
				enbRrcSetupSuccessRate: true,
				enbERABSetupSuccessRate: true,
				enbERABDropRate: true,
				enbCsfbSuccessRate: true,
				enbHoIntraEnbOutSuccRate: true,
				enbHoIntraEnbInSuccRate: true,
				enbHoInterEnbOutSuccRate: true,
				enbHoInterEnbInSuccRate: true,
				gnbDownlinkPRBUtilizationRate: true,
				gnbUplinkPRBUtilizationRate: true,
				gnbPdcpUpOctDL: true,
				gnbPdcpUpOctUL: true,
				gsmCallSetupSuccRate: true,
				gsmCallDropRate: true,
				gsmHandoverSuccessRate: true,
				sas: true
			},
            alarmStatisticData:{
                site_alarm_count: '0',
                offline_alarm_count: '0',
                unavailable_alarm_count: '0',
            },
            lineNowTotalData:{
                siteOffline: {
                    total: '-',
                    total_status: '-',
                    offline_count: '-',
                    offline_count_status: '-',
                },
                enbOnline:{
                    device_count: '-',
                    device_count_status: '-',
                    online_count: '-',
                    online_count_status: '-',
                    online_rate: '-',
                    online_rate_status: '-',
                },
                enbActive:{
                    device_count: '-',
                    device_count_status: '-',
                    active_count: '-',
                    active_count_status: '-',
                    active_rate: '-',
                    active_rate_status: '-',
                },
                enbUeCount:{
                    ue_count: '-',
                    ue_count_status: '-',
                },
                enbMmeStatus:{
                    mme_connected_count: '-',
                    mme_connected_count_status: '-',
                    mme_disconnected_count: '-',
                    mme_disconnected_count_status: '-',
                },
                gnbOnline:{
                    device_count: '-',
                    device_count_status: '-',
                    online_count: '-',
                    online_count_status: '-',
                    online_rate: '-',
                    online_rate_status: '-',
                },
                gnbActive:{
                    device_count: '-',
                    device_count_status: '-',
                    active_count: '-',
                    active_count_status: '-',
                    active_rate: '-',
                    active_rate_status: '-',
                },
                gnbUeCount:{
                    ue_count: '-',
                    ue_count_status: '-',
                },
                gsmOnline:{
                    device_count: '-',
                    device_count_status: '-',
                    online_count: '-',
                    online_count_status: '-',
                    online_rate: '-',
                    online_rate_status: '-',
                },
                gsmActive:{
                    device_count: '-',
                    device_count_status: '-',
                    active_count: '-',
                    active_count_status: '-',
                    active_rate: '-',
                    active_rate_status: '-',
                },
                gsmUeCount:{
                    ue_count: '-',
                    ue_count_status: '-',
                },
                cpeOnline:{
                    device_count: '-',
                    device_count_status: '-',
                    online_count: '-',
                    online_count_status: '-',
                    online_rate: '-',
                    online_rate_status: '-',
                },
                enbThroughputDL:{
                    count: '-',
                    count_status: '-',
                },
                enbThroughputUL:{
                    count: '-',
                    count_status: '-',
                },
                enbWirelessSetupSuccessRate:{
                    count: '-',
                    count_status: '-',
                },
                enbTotalDataVolumeDL:{
                    count: '-',
                    count_status: '-',
                },
                enbTotalDataVolumeUL:{
                    count: '-',
                    count_status: '-',
                },
                enbRrcSetupSuccessRate:{
                    count: '-',
                    count_status: '-',
                },

                enbERABSetupSuccessRate:{
                    count: '-',
                    count_status: '-',
                },
                enbERABDropRate:{
                    count: '-',
                    count_status: '-',
                },
                enbCsfbSuccessRate:{
                    count: '-',
                    count_status: '-',
                },
                enbHoIntraEnbOutSuccRate:{
                    count: '-',
                    count_status: '-',
                },
                enbHoIntraEnbInSuccRate:{
                    count: '-',
                    count_status: '-',
                },
                enbHoInterEnbOutSuccRate:{
                    count: '-',
                    count_status: '-',
                },
                enbHoInterEnbInSuccRate:{
                    count: '-',
                    count_status: '-',
                },
                enbDownlinkPRBUtilizationRate:{
                    count: '-',
                    count_status: '-',
                },
                enbUplinkPRBUtilizationRate:{
                    count: '-',
                    count_status: '-',
                },
                enbCellAvailable:{
                    count: '-',
                    count_status: '-',
                },
                gnbThroughputDL:{
                    count: '-',
                    count_status: '-',
                },
                gnbThroughputUL:{
                    count: '-',
                    count_status: '-',
                },
                gnbDownlinkPRBUtilizationRate:{
                    count: '-',
                    count_status: '-',
                },
                gnbUplinkPRBUtilizationRate:{
                    count: '-',
                    count_status: '-',
                },
                gnbPdcpUpOctDL:{
                    count: '-',
                    count_status: '-',
                },
                gnbPdcpUpOctUL:{
                    count: '-',
                    count_status: '-',
                },
                gsmCallSetupSuccRate:{
                    count: '-',
                    count_status: '-',
                },
                gsmCallDropRate:{
                    count: '-',
                    count_status: '-',
                },
                gsmHandoverSuccessRate:{
                    count: '-',
                    count_status: '-',
                },
            },
			chartList:{
                siteChartList:['siteOffline'],
				enbChartList:['enbOnline','enbActive','enbUeCount','enbMmeStatus','enbProductType','enbDeviceRunningTime'],
				gnbChartList:['gnbOnline','gnbActive','gnbUeCount','gnbProductType','gnbDeviceRunningTime'],
				gsmChartList:['gsmOnline','gsmActive','gsmUeCount','gsmProductType','gsmDeviceRunningTime'],
                cpeChartList:['cpeOnline','cpeProductType','cpeDeviceRunningTime'],
                kpiChartList:['enbThroughputDL','enbThroughputUL','enbWirelessSetupSuccessRate','enbTotalDataVolumeDL','enbTotalDataVolumeUL','enbRrcSetupSuccessRate','enbERABSetupSuccessRate','enbERABDropRate','enbCsfbSuccessRate','enbHoIntraEnbOutSuccRate','enbHoIntraEnbInSuccRate','enbHoInterEnbOutSuccRate','enbHoInterEnbInSuccRate','enbDownlinkPRBUtilizationRate','enbUplinkPRBUtilizationRate','enbCellAvailable','gnbThroughputDL','gnbThroughputUL','gnbDownlinkPRBUtilizationRate','gnbUplinkPRBUtilizationRate','gnbPdcpUpOctDL','gnbPdcpUpOctUL','gsmCallSetupSuccRate','gsmCallDropRate','gsmHandoverSuccessRate'],
                sasChartList:['enbSASStatistic','gnbSASStatistic','cpeSASStatistic']
			},
			charts: {},
			visibleCode: {
                'siteStatistic': supportTopoSite,
                'gsmStatistic': supportGSM,
				'enbStatistic': writableMap['CODE_ENB_MONITOR'] != undefined,
				'cpeStatistic': writableMap['CODE_CPE_MONITOR'] != undefined,
				'gnbStatistic': writableMap['CODE_GNB_MONITOR'] != undefined,
				'SASStatistic': writableMap['CODE_ADVANCE_SAS'] != undefined,
                'alarmStatistic': writableMap['CODE_ALARM_VIEW'] != undefined,
                'kpiStatistic' : writableMap['CODE_PERFORMANCE_VIEW'] != undefined,
			},
            periodTypeData: {
                enbThroughputDL: 'min',
                enbThroughputUL: 'min',
                enbWirelessSetupSuccessRate: 'min',
                enbTotalDataVolumeDL: 'min',
                enbTotalDataVolumeUL: 'min',
                enbRrcSetupSuccessRate: 'min',
                enbERABSetupSuccessRate: 'min',
                enbERABDropRate: 'min',
                enbCsfbSuccessRate: 'min',
                enbHoIntraEnbOutSuccRate: 'min',
                enbHoIntraEnbInSuccRate: 'min',
                enbHoInterEnbOutSuccRate: 'min',
                enbHoInterEnbInSuccRate: 'min',
                enbDownlinkPRBUtilizationRate: 'min',
                enbUplinkPRBUtilizationRate: 'min',
                enbCellAvailable: 'min',
                gnbThroughputDL: 'min',
                gnbThroughputUL: 'min',
                gnbDownlinkPRBUtilizationRate: 'min',
                gnbUplinkPRBUtilizationRate: 'min',
                gnbPdcpUpOctDL: 'min',
                gnbPdcpUpOctUL: 'min',
                gsmCallSetupSuccRate: 'min',
                gsmCallDropRate: 'min',
                gsmHandoverSuccessRate: 'min',
            },
            kpiTitleCodes:{
                'enbThroughputDL':'Throughput DL',
                'enbThroughputUL':'Throughput UL',
                'enbWirelessSetupSuccessRate':'Wireless Setup Success Rate',
                'enbTotalDataVolumeDL':'Total Data Volume DL',
                'enbTotalDataVolumeUL':'Total Data Volume UL',
                'enbRrcSetupSuccessRate':'RRC Setup Success Rate',
                'enbERABSetupSuccessRate':'E-RAB Setup Success Rate',
                'enbERABDropRate':'E-RAB Drop Rate',
                'enbCsfbSuccessRate':'CSFB Success Rate',
                'enbHoIntraEnbOutSuccRate':'HO IntraEnbOutSucc Rate',
                'enbHoIntraEnbInSuccRate':'HO IntraEnbInSucc Rate',
                'enbHoInterEnbOutSuccRate':'HO InterEnbOutSucc Rate',
                'enbHoInterEnbInSuccRate':'HO InterEnbInSucc Rate',
                'enbDownlinkPRBUtilizationRate':'Downlink PRB Utilization Rate',
                'enbUplinkPRBUtilizationRate':'Uplink PRB Utilization Rate',
                'enbCellAvailable':'Cell Available',
                'gnbThroughputDL':'Throughput DL',
                'gnbThroughputUL':'Throughput UL',
                'gnbDownlinkPRBUtilizationRate':'Downlink PRB Utilization Rate',
                'gnbUplinkPRBUtilizationRate':'Uplink PRB Utilization Rate',
                'gnbPdcpUpOctDL':'KPI.PdcpUpOctDL',
                'gnbPdcpUpOctUL':'KPI.PdcpUpOctUL',
                'gsmCallSetupSuccRate':'KPI.CallSetupSuccRate',
                'gsmCallDropRate':'KPI.CallDropRate',
                'gsmHandoverSuccessRate':'KPI.HandoverSuccessRate',
            },
            counterCodes: {
                'enbThroughputDL': '(C000060011+C000060022)*8/#period/1000',
                'enbThroughputUL': '(C000060001+C000060021)*8/#period/1000',
                'enbWirelessSetupSuccessRate': 'K900010006',
                'enbTotalDataVolumeDL': '(C000060011+C000060022)/1000/1000',
                'enbTotalDataVolumeUL': '(C000060001+C000060021)/1000/1000',
                'enbRrcSetupSuccessRate': 'K900010002',
                'enbERABSetupSuccessRate': 'K900010005',
                'enbERABDropRate': 'K900010027',
                'enbCsfbSuccessRate': 'K900010029',
                'enbHoIntraEnbOutSuccRate': 'K900010017',
                'enbHoIntraEnbInSuccRate': 'K900010022',
                'enbHoInterEnbOutSuccRate': 'K900010021',
                'enbHoInterEnbInSuccRate': 'K900010026',
                'enbDownlinkPRBUtilizationRate': 'K900010014',
                'enbUplinkPRBUtilizationRate': 'K900010013',
                'enbCellAvailable':'C000060216/#period*100',
                'gnbThroughputDL': 'C010030006*8/#period/1000',
                'gnbThroughputUL': 'C010030005*8/#period/1000',
                'gnbDownlinkPRBUtilizationRate': 'KGNB0506',
                'gnbUplinkPRBUtilizationRate': 'KGNB0505',
                'gnbPdcpUpOctDL': 'C010030006/1000/1000',
                'gnbPdcpUpOctUL': 'C010030005/1000/1000',
                'gsmCallSetupSuccRate': 'KGSM0102',
                'gsmCallDropRate': 'KGSM0103',
                'gsmHandoverSuccessRate': 'KGSM0101',
            },
            unitCodes: { 
                'enbThroughputDL': '(Mbps)',
                'enbThroughputUL': '(Mbps)',
                'enbWirelessSetupSuccessRate': '(%)',
                'enbTotalDataVolumeDL': '(GB)',
                'enbTotalDataVolumeUL': '(GB)',
                'enbRrcSetupSuccessRate': '(%)',
                'enbERABSetupSuccessRate': '(%)',
                'enbERABDropRate': '(%)',
                'enbCsfbSuccessRate': '(%)',
                'enbHoIntraEnbOutSuccRate': '(%)',
                'enbHoIntraEnbInSuccRate': '(%)',
                'enbHoInterEnbOutSuccRate': '(%)',
                'enbHoInterEnbInSuccRate': '(%)',
                'enbDownlinkPRBUtilizationRate': '(%)',
                'enbUplinkPRBUtilizationRate': '(%)',
                'enbCellAvailable': '(%)',
                'gnbThroughputDL': '(Mbps)',
                'gnbThroughputUL': '(Mbps)',
                'gnbDownlinkPRBUtilizationRate': '(%)',
                'gnbUplinkPRBUtilizationRate': '(%)',
                'gnbPdcpUpOctDL': '(GB)',
                'gnbPdcpUpOctUL': '(GB)',
                'gsmCallSetupSuccRate': '(%)',
                'gsmCallDropRate': '(%)',
                'gsmHandoverSuccessRate': '(%)',
            },
			sevenDays: '<%=rb.getString("Tian")%>' == 'Day'? '7 Days':'7 <%=rb.getString("Tian")%>',
            chartLoadingData:{
                siteOffline: false,
                enbOnline: false,
                enbActive: false,
                enbUeCount: false,
                enbMmeStatus: false,
                enbProductType: false,
                enbDeviceRunningTime: false,
                gnbOnline: false,
                gnbActive: false,
                gnbUeCount: false,
                gnbProductType: false,
                gnbDeviceRunningTime: false,
                gsmOnline: false,
                gsmActive: false,
                gsmUeCount: false,
                gsmProductType: false,
                gsmDeviceRunningTime: false,
                cpeOnline: false,
                cpeProductType: false,
                cpeDeviceRunningTime: false,
                enbThroughputDL: false,
                enbThroughputUL: false,
                enbWirelessSetupSuccessRate: false,
                enbTotalDataVolumeDL: false,
                enbTotalDataVolumeUL: false,
                enbRrcSetupSuccessRate: false,
                enbERABSetupSuccessRate: false,
                enbERABDropRate: false,
                enbCsfbSuccessRate: false,
                enbHoIntraEnbOutSuccRate: false,
                enbHoIntraEnbInSuccRate: false,
                enbHoInterEnbOutSuccRate: false,
                enbHoInterEnbInSuccRate: false,
                enbDownlinkPRBUtilizationRate: false,
                enbUplinkPRBUtilizationRate: false,
                enbCellAvailable: false,
                gnbThroughputDL: false,
                gnbThroughputUL: false,
                gnbDownlinkPRBUtilizationRate: false,
                gnbUplinkPRBUtilizationRate: false,
                gnbPdcpUpOctDL: false,
                gnbPdcpUpOctUL: false,
                gsmCallSetupSuccRate: false,
                gsmCallDropRate: false,
                gsmHandoverSuccessRate: false,
            }
		};
	},
	watch: {},
	computed: {
        enbStatisticNowTitle(){
            var vm = this,
                codes = {
                    'enbOnline':'<%=rb.getString("ShouYe_ZaiXian")%>',
                    'enbActive':'<%=rb.getString("ShouYe_HuoYue")%>',
                    'enbUeCount':'<%=rb.getString("ShouYe_UEShu")%>',
                    'enbMmeStatus': '<%=rb.getString("MMEZhuangTai")%>',
                    'enbProductType':'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',
                    'enbDeviceRunningTime':'<%=rb.getString("SheBeiYunXingShiChang")%>',
                };
            return codes[vm.enbStatisticSelect]
        },
        gnbStatisticNowTitle(){
            var vm = this,
                codes = {
                    'gnbOnline':'<%=rb.getString("ShouYe_ZaiXian")%>',
                    'gnbActive':'<%=rb.getString("ShouYe_HuoYue")%>',
                    'gnbUeCount':'<%=rb.getString("ShouYe_UEShu")%>',
                    'gnbProductType':'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',
                    'gnbDeviceRunningTime':'<%=rb.getString("SheBeiYunXingShiChang")%>',
                };
            return codes[vm.gnbStatisticSelect]
        },
        gsmStatisticNowTitle(){
            var vm = this,
                codes = {
                    'gsmOnline':'<%=rb.getString("ShouYe_ZaiXian")%>',
                    'gsmActive':'<%=rb.getString("ShouYe_HuoYue")%>',
                    'gsmUeCount':'<%=rb.getString("ShouYe_UEShu")%>',
                    'gsmProductType':'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',
                    'gsmDeviceRunningTime':'<%=rb.getString("SheBeiYunXingShiChang")%>',
                };
            return codes[vm.gsmStatisticSelect]
        },
        cpeStatisticNowTitle(){
            var vm = this,
                codes = {
                    'cpeOnline':'<%=rb.getString("ShouYe_ZaiXian")%>',
                    'cpeProductType':'<%=rb.getString("ChanPinXingHao")%>',
                    'cpeDeviceRunningTime':'<%=rb.getString("SheBeiYunXingShiChang")%>',
                };
            return codes[vm.cpeStatisticSelect]
        },
        enbKpiTrafficNowTitle(){
            var vm = this;
            return vm.kpiTitleCodes[vm.enbKpiTrafficStatisticSelect]
        },
        enbKpiUtilizationNowTitle(){
            var vm = this;
            return vm.kpiTitleCodes[vm.enbKpiUtilizationStatisticSelect]
        },
        enbKpiAccessibilityNowTitle(){
            var vm = this;
            return vm.kpiTitleCodes[vm.enbKpiAccessibilityStatisticSelect]
        },
        enbKpiMobilityNowTitle(){
            var vm = this;
            return vm.kpiTitleCodes[vm.enbKpiMobilityStatisticSelect]
        },
        gnbKpiTrafficNowTitle(){
            var vm = this;
            return vm.kpiTitleCodes[vm.gnbKpiTrafficStatisticSelect]
        },
        gnbKpiUtilizationNowTitle(){
            var vm = this;
            return vm.kpiTitleCodes[vm.gnbKpiUtilizationStatisticSelect]
        },
        lineNowTotalAndStatus(){
            return (code , param)=>{
                return this.lineNowTotalData[code][param] != undefined ? this.lineNowTotalData[code][param] : '-';
            }
        }
               
    },
	methods: {
		// 图表初始化
		init(){
			var vm = this,
				codes = [
                    {label:'Site', isPermission: vm.visibleCode['siteStatistic']},
					{label:'eNB', isPermission: vm.visibleCode['enbStatistic']},
					{label:'gNB', isPermission: vm.visibleCode['gnbStatistic']},
                    {label:'GSM', isPermission: vm.visibleCode['gsmStatistic']},
					{label:'CPE', isPermission: vm.visibleCode['cpeStatistic']},
				],
                kpiCodes = [
                    {label:'eNB', isPermission: vm.visibleCode['enbStatistic']},
                    {label:'gNB', isPermission: vm.visibleCode['gnbStatistic']},
                    {label:'GSM', isPermission: vm.visibleCode['gsmStatistic']},
                ],
                codesClick = {
                    'Site': vm.initSiteChart,
                    'eNB': vm.initEnbChart,
                    'gNB': vm.initGnbChart,
                    'GSM': vm.initGsmChart,
                    'CPE': vm.initCpeChart,
                },
				isResult = false,
                isKpiResult = false;
			
			codes.map((item)=>{
				if(item.isPermission){
                    codesClick[item.label]();
                    if(!isResult){
                        vm.deviceStatusActiveTab = item.label;
                    }
                    isResult = true;
				}
			})
            if(vm.deviceStatusActiveTab == 'Site'){
                ['siteOffline'].map(function(code){
                    vm.reloadChart(code,6);
                });
            }else if(vm.deviceStatusActiveTab == 'eNB'){
                ['enbOnline'].map(function(code){
                    vm.reloadChart(code,6);
                });
            }else if(vm.deviceStatusActiveTab == 'gNB'){
                ['gnbOnline'].map(function(code){
                    vm.reloadChart(code,6);
                });
            }else if(vm.deviceStatusActiveTab == 'GSM'){
                ['gsmOnline'].map(function(code){
                    vm.reloadChart(code,6);
                });
            }else if(vm.deviceStatusActiveTab == 'CPE'){
                ['cpeOnline'].map(function(code){
                    vm.reloadChart(code,6);
                });
            }
            if(vm.visibleCode['kpiStatistic']){
                vm.initKpiChart();
                kpiCodes.map((item)=>{
                    if(item.isPermission){
                        if(!isKpiResult){
                            vm.kpiStatisticActiveTab = item.label;
                        }
                        isKpiResult = true;
                    }
                })
                if(vm.kpiStatisticActiveTab == 'eNB'){
                    ['enbTotalDataVolumeDL','enbCellAvailable','enbDownlinkPRBUtilizationRate','enbWirelessSetupSuccessRate','enbERABDropRate','enbHoIntraEnbOutSuccRate'].map(function(code){
                        vm.reloadChart(code,6);
                    });
                }else if(vm.kpiStatisticActiveTab == 'gNB'){
                    ['gnbPdcpUpOctDL','gnbDownlinkPRBUtilizationRate'].map(function(code){
                        vm.reloadChart(code,6);
                    });
                }else if(vm.kpiStatisticActiveTab == 'GSM'){
                    ['gsmCallSetupSuccRate','gsmCallDropRate','gsmHandoverSuccessRate'].map(function(code){
                        vm.reloadChart(code,6);
                    });
                }
            }
            if(vm.visibleCode['alarmStatistic']){
                vm.initAlarmData();
            }
			if(writableMap['CODE_ADVANCE_SAS'] != undefined && false){
				vm.initSasChart();
			}
            vm.setIntervalDashboard();
		},
        // 定时刷新首页统计
        setIntervalDashboard(){ 
            var vm = this;
            if(window.updateDashboardDeviceStatusTimer) clearInterval(window.updateDashboardDeviceStatusTimer);
            window.updateDashboardDeviceStatusTimer = setInterval(function(){
                var dashboardPageCtn = $("#dashboard_ctn");			
                if(!dashboardPageCtn.length) {
                    clearInterval(window.updateDashboardDeviceStatusTimer);
                    return;
                }
                var ctner = document.querySelector('#dashboard_ctn'),
                    visible = isVisible(ctner),
                    isCovered = isOverlapped(ctner);
                
                if(visible && !isCovered) {
                    if(vm.visibleCode['alarmStatistic']){
                        vm.initAlarmData();
                    }
                    vm.refreshDeviceStatus();
                }
            },600000); // 10分钟 600000
            if(window.updateDashboardKpiTimer) clearInterval(window.updateDashboardKpiTimer);
            window.updateDashboardKpiTimer = setInterval(function(){
                var dashboardPageCtn = $("#dashboard_ctn");			
                if(!dashboardPageCtn.length) {
                    clearInterval(window.updateDashboardKpiTimer);
                    return;
                }
                var ctner = document.querySelector('#dashboard_ctn'),
                    visible = isVisible(ctner),
                    isCovered = isOverlapped(ctner);
                
                if(visible && !isCovered) {
                    vm.refreshKpiStatistic('timing');
                }
            },600000); // 10分钟 600000 60分钟 3600000
        },
        // 设备状态 Tab栏切换
        deviceStatusTabChange(tab){
            var vm = this;
            vm.$nextTick(function(){
                vm.resizeChart();
                vm.refreshDeviceStatus();
            })
        },
        // KPI 统计 Tab栏切换
        kpiStatisticTabChange(tab){
            var vm = this;
            vm.$nextTick(function(){
                 vm.resizeChart();
                vm.refreshKpiStatistic('general');
            })
        },
        // 刷新设备状态图表
        refreshDeviceStatus(){
            var vm = this;
            if(vm.deviceStatusActiveTab == 'Site'){
                ['siteOffline'].map(function(code){
                    vm.reloadChart(code,6);
                });
            }else if(vm.deviceStatusActiveTab == 'eNB'){
                vm.reloadChart(vm.enbStatisticSelect,6);
            }else if(vm.deviceStatusActiveTab == 'gNB'){
                vm.reloadChart(vm.gnbStatisticSelect,6);
            }else if(vm.deviceStatusActiveTab == 'GSM'){
                vm.reloadChart(vm.gsmStatisticSelect,6);
            }else if(vm.deviceStatusActiveTab == 'CPE'){
                vm.reloadChart(vm.cpeStatisticSelect,6);
            }
        },
        // 刷新KPI统计图表
        refreshKpiStatistic(type){ 
            var vm = this;
            if(vm.kpiStatisticActiveTab == 'eNB'){
                let arrs = ['enbCellAvailable','enbERABDropRate'];
                arrs.push(vm.enbKpiTrafficStatisticSelect);
                arrs.push(vm.enbKpiUtilizationStatisticSelect);
                arrs.push(vm.enbKpiAccessibilityStatisticSelect);
                arrs.push(vm.enbKpiMobilityStatisticSelect);
                arrs.map((code) =>{
                    if(vm.periodTypeData[code] !== 'week' || type !== 'timing'){
                        vm.reloadChart(code,6);
                    }
                })
            }else if(vm.kpiStatisticActiveTab == 'gNB'){
                let arrs = [];
                arrs.push(vm.gnbKpiTrafficStatisticSelect);
                arrs.push(vm.gnbKpiUtilizationStatisticSelect);
                arrs.map((code) =>{
                    if(vm.periodTypeData[code] !== 'week' || type !== 'timing'){
                        vm.reloadChart(code,6);
                    }
                })
            }else if(vm.kpiStatisticActiveTab == 'GSM'){
                let arrs = ['gsmCallSetupSuccRate','gsmCallDropRate','gsmHandoverSuccessRate'];
                arrs.map((code) =>{
                    if(vm.periodTypeData[code] !== 'week' || type !== 'timing'){
                        vm.reloadChart(code,6);
                    }
                })
            }
        },
        // ENB 统计切换
        enbStatisticChangeClick(code){
            var vm = this;
            vm.enbStatisticSelect = code;
            vm.$nextTick(function(){
                if(code == 'enbProductType'){
                    vm.initProductType('enb');
                }else if(code == 'enbDeviceRunningTime'){
                    vm.initDeviceRunTime('enb');
                }else{
                    vm.reloadChart(code,6);
                }
                vm.resizeChart();
            })
        },
        // GNB 统计切换
        gnbStatisticChangeClick(code){
            var vm = this;
            vm.gnbStatisticSelect = code;
            vm.$nextTick(function(){
                if(code == 'gnbProductType'){
                    vm.initProductType('gnb');
                }else if(code == 'gnbDeviceRunningTime'){
                    vm.initDeviceRunTime('gnb');
                }else{
                    vm.reloadChart(code,6);
                }
                vm.resizeChart();
            })
        },
        // GSM 统计切换
        gsmStatisticChangeClick(code){
            var vm = this;
            vm.gsmStatisticSelect = code;
            vm.$nextTick(function(){
                if(code == 'gsmProductType'){
                    vm.initProductType('gsm');
                }else if(code == 'gsmDeviceRunningTime'){
                    vm.initDeviceRunTime('gsm');
                }else{
                    vm.reloadChart(code,6);
                }
                vm.resizeChart();
            })
        },
        // CPE 统计切换
        cpeStatisticChangeClick(code){
            var vm = this;
            vm.cpeStatisticSelect = code;
            vm.$nextTick(function(){
                if(code == 'cpeProductType'){
                    vm.initProductType('cpe');
                }else if(code == 'cpeDeviceRunningTime'){
                    vm.initDeviceRunTime('cpe');
                }else{
                    vm.reloadChart(code,6);
                }
                vm.resizeChart();
            })
        },
        // ENB KPI Traffic 统计切换
        enbKpiTrafficChangeClick(code){
            var vm = this;
            vm.enbKpiTrafficStatisticSelect = code;
            vm.$nextTick(function(){
                vm.reloadChart(code,6);
                vm.resizeChart();
            })
        },
        // ENB KPI Utilization 统计切换
        enbKpiUtilizationChangeClick(code){
            var vm = this;
            vm.enbKpiUtilizationStatisticSelect = code;
            vm.$nextTick(function(){
                vm.reloadChart(code,6);
                vm.resizeChart();
            })
        },
        // ENB KPI Accessibility 统计切换
        enbKpiAccessibilityChangeClick(code){
            var vm = this;
            vm.enbKpiAccessibilityStatisticSelect = code;
            vm.$nextTick(function(){
                vm.reloadChart(code,6);
                vm.resizeChart();
            })
        },
        // ENB KPI Mobility 统计切换
        enbKpiMobilityChangeClick(code){
            var vm = this;
            vm.enbKpiMobilityStatisticSelect = code;
            vm.$nextTick(function(){
                vm.reloadChart(code,6);
                vm.resizeChart();
            })
        },
        // GNB KPI Traffic 统计切换
        gnbKpiTrafficChangeClick(code){
            var vm = this;
            vm.gnbKpiTrafficStatisticSelect = code;
            vm.$nextTick(function(){
                vm.reloadChart(code,6);
                vm.resizeChart();
            })
        },
        // GNB KPI Utilization 统计切换
        gnbKpiUtilizationChangeClick(code){
            var vm = this;
            vm.gnbKpiUtilizationStatisticSelect = code;
            vm.$nextTick(function(){
                vm.reloadChart(code,6);
                vm.resizeChart();
            })
        },
        // 自定义窗口打开 关闭
        customBoxExpandChange(){
            var vm = this;
            vm.customBoxExpand = !vm.customBoxExpand;
            vm.$nextTick(function(){
                vm.resizeChart();
            })
        },
        // 请求告警数量
        initAlarmData(){ 
            var vm = this,
                params = {
                    timeZone: timeZone,
                    alarmType: 'ACTIVE',
                    queryType: 'View'
                },
                urls = '${ctx}/fault/view/queryDashBoardAlarmInfo.action';
            axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data){
					Object.keys(vm.alarmStatisticData).map(function(key){
                        if(data[key] != undefined){
                            vm.alarmStatisticData[key] = data[key];
                        }
                    })
				}
			})
        },
        // 初始化site图表
        initSiteChart(){
            var vm = this;
			vm.chartList.siteChartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
                    if(['siteOffline'].includes(code)) {
						// 初始化时间轴切换事件
						vm.charts[code].on('timelinechanged',function(p){
							vm.reloadChart(code,p.currentIndex)
						});
					}
				}
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);

			});
        },
        // 初始化ENB图表
		initEnbChart(){
			var vm = this;
			vm.chartList.enbChartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
                    if(['enbOnline','enbActive','enbUeCount','enbMmeStatus'].includes(code)) {
						// 初始化时间轴切换事件
						vm.charts[code].on('timelinechanged',function(p){
							vm.reloadChart(code,p.currentIndex)
						});
					}
				}
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);
			});
		},
        // 初始化CPE图表
		initCpeChart(){
			var vm = this;
			vm.chartList.cpeChartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
					if(['cpeOnline'].includes(code)) {
						// 初始化时间轴切换事件
						vm.charts[code].on('timelinechanged',function(p){
							vm.reloadChart(code,p.currentIndex)
						});
					}
				}
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);
			});
		},
        // 初始化GNB图表
		initGnbChart(){
			var vm = this;
			vm.chartList.gnbChartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
                    if(['gnbOnline','gnbActive','gnbUeCount'].includes(code)) {
						// 初始化时间轴切换事件
						vm.charts[code].on('timelinechanged',function(p){
							vm.reloadChart(code,p.currentIndex)
						});
					}
				}
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);
			});
		},
        // 初始化GSM图表
        initGsmChart(){
			var vm = this;
			vm.chartList.gsmChartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
                    if(['gsmOnline','gsmActive','gsmUeCount'].includes(code)) {
						// 初始化时间轴切换事件
						vm.charts[code].on('timelinechanged',function(p){
							vm.reloadChart(code,p.currentIndex)
						});
					}
				}
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);
			});
		},
        // 初始化KPI图表
        initKpiChart(){
			var vm = this;
			vm.chartList.kpiChartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
					vm.charts[code].on('timelinechanged',function(p){
                        vm.reloadChart(code,p.currentIndex)
                    });
				}
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);

			});
		},
        // 初始化SAS图表
		initSasChart(){
			var vm = this;
			vm.chartList.sasChartList.map(function(code){
				var chart = document.querySelector('#'+code);
				if(chart) {
					vm.charts[code] = echarts.init(chart);
				}
				// 窗口缩放时自适应
				window.removeEventListener('resize',vm.resizeChart);
				window.addEventListener('resize',vm.resizeChart);

				vm.reloadSasChart(code);
			});
		},
		// sas 统计
		reloadSasChart(code){
			var vm = this,
				codeList = {
					'enbSASStatistic': 'eNB',
					'gnbSASStatistic': 'gNB',
					'cpeSASStatistic': 'CPE'
				},
				deviceType = codeList[code],
				params = {
					deviceType: deviceType
				},
				chartParams={
					titleText:'0',
					deviceType:deviceType,
					subText:'Enable',
					chartData:[
						{name:'Authorized',value:0,key:'authorizedCount'},
						{name:'Granted',value:0,key:'grantedCount'},
						{name:'Unregistered',value:0,key:'unregisteredCount'},	
						{name:'Registered',value:0,key:'registeredCount'},			
						{name:'Grant-Suspended',value:0,key:'grantSuspendedCount'},
					]
				};
			axios.post('${ctx}/cell/SAS/queryStatistics.action',stringify(params)).then(function(response){
				var data = response.data
				if(data){
					chartParams.titleText = data.sasEnableCount;
					chartParams.chartData.map((item)=>{
						item.value = data[item.key]
					})
					vm.charts[code].setOption(vm.creatSasChartOption(chartParams));
					vm.resizeChart();
				}
			})
		},
        // 生成SAS图表配置
		creatSasChartOption(chartParams){
			var vm = this,
				option={
					title:{
						text:chartParams.titleText,
						subtext:chartParams.subText,
						subtextStyle:{
							color:'rgba(0,0,0,0.8)',
							fontSize:16
						},
						top:'40%',
						left:'center'
					},
					color: ['#D9EAD3','#FFE198','#F4CCCC','#FAC2C2','#FFF2D1'], 
					series:[
						{
							name:chartParams.deviceType,
							type:'pie',
							radius:['40%','70%'],
							avoidLabelOverlap:true,
							label:{
								show:true,
								normal:{formatter:'{b}:{c}({d}%)',color:'#000'},
							},
							labelLine:{
								show:true,
								fontSize:48,
								length:30
							},
							data:chartParams.chartData,
						}
					]
				};
			return option;
		},
		// 跳转到SAS
		goProcedure() {
			var vm = this;
			sessionStorage.setItem('submenuid', '1000100');
			eventAllBus.$emit("gomenupage","10000","",'10000',{},function(){});
		},
		/**
		* 快捷方式跳转
		* @param item{object}  跳转页面的id地址
		*/
		jumpToPage(item){
			eventAllBus.$emit("gomenupage",item.id,"",item.id);
		
		},
		// 所有图表自适应
		resizeChart(){
			var vm = this,
				codes=[
                    'siteChartList',
                    'gsmChartList',
                    'enbChartList',
                    'gnbChartList',
                    'cpeChartList', 
                    'kpiChartList',
                    'sasChartList'
                ];
            codes.map((codeList)=>{
                vm.chartList[codeList].map(function(code){
                    if(vm.charts[code]) vm.charts[code].resize();
                });
            });
		},
		/**
		* 指定图表数据刷新
		* @param code{string} 图表类型
		* @param index{number}  时间轴下标
		*/
		reloadChart(code,index){
			var vm = this;
			
			if(vm.charts[code]) {
				if(['enbOnline','enbActive','enbUeCount','enbMmeStatus','gnbOnline','gnbActive','gnbUeCount','gsmOnline','gsmActive','gsmUeCount'].includes(code)) {
					vm.proccessEnbOrGnbOrGsm(code,index); // 基站统计数据刷新
				}
				if(['cpeOnline'].includes(code)) {
					vm.proccessCPE(code,index);// CPE统计数据刷新
				}
                if(vm.chartList['kpiChartList'].includes(code)) {
                    vm.proccessTraffic(code,index);// 性能统计数据刷新
                }
                if(['siteOffline'].includes(code)) {
                    vm.proccessSite(code,index);// 站点下线数据刷新
                }
			}
		},
		/**
		* 基站统计数据刷新
		* @param code{string} 图表类型
		* @param index{number}  时间轴下标
		*/
		proccessEnbOrGnbOrGsm(code,index){
		   	var vm = this,
			   	urls = '${ctx}/system/device/getDeviceStatisticsDataList.action',
				color = ['#47D468','#69B6FC'],
			   	legend = ['Active','Onlince'],
                lineLength = '2',
			   	legendNames = ['<%=rb.getString("ShouYe_HuoYue")%>','<%=rb.getString("ShouYe_ZaiXian")%>'],
		   		unitName='<%=rb.getString("GeShu")%>',
				title = '<%=rb.getString("ShouYe_HuoYueZhuangTai")%> / <%=rb.getString("ShouYe_ZaiXianZhuangTai")%>';
            if(['enbOnline','gnbOnline','gsmOnline'].includes(code)){
				color = ['#1F77B4','#0ABF5B'],
				legend = ['Total','Onlince'],
                lineLength = '2';
				legendNames = ['<%=rb.getString("ShouYe_ZongShu")%>','<%=rb.getString("ShouYe_ZaiXian")%>'];
			}
			if(['enbActive','gnbActive','gsmActive'].includes(code)){
				color = ['#1F77B4','#0ABF5B'],
				legend = ['Total','Active'],
                lineLength = '2',
				legendNames = ['<%=rb.getString("ShouYe_ZongShu")%>','<%=rb.getString("ShouYe_HuoYue")%>'];
			}
            if(['enbUeCount','gnbUeCount','gsmUeCount'].includes(code)){
                color = ['#0ABF5B'],
				legend = ['ueCount'],
                lineLength = '1',
				legendNames = ['<%=rb.getString("ShouYe_UEShu")%>'],
				title = '<%=rb.getString("ShouYe_UEShu")%>';
            }
			if(['enbMmeStatus'].includes(code)) {
				color = ['#0ABF5B','#FF5B45'];
				legend = ['Connected','Disconnected'];
                lineLength = '2',
				legendNames = ['<%=rb.getString("ShouYe_LianJie")%>','<%=rb.getString("ShouYe_FeiLianJie")%>'];
				title = '<%=rb.getString("MMEZhuangTai")%>';
			}

			var vm = this,
				s_time = getYesterDay(6-index).substring(0,10) + ' 00:00:00',
				e_time = getYesterDay(6-index).substring(0,10) + ' 23:59:59';
			
			if(index==6) {
				e_time = getNowTimeToZoneTimeRange(timeZone, 144).end_time;
			}else{
				e_time = getYesterDay(6-index-1).substring(0,10) + ' 00:00:00';
			}
			var queryParams = {
					device_type: code.slice(0,3),
					time_level: "min",
					timeZone: timeZone,
					start_time: s_time,
					end_time: e_time
		 		},
				params = {
					legend: legend,
                    lineLength: lineLength,
					legendNames: legendNames,
					data: [],
					unit: unitName,
					pointerCount: 6,
                    period:'min',
					index: index,
					code: code,
					title: title,
					color:color
				};
            if(code.slice(0,3) == 'gnb'){
                urls = '${ctx}/system/device/getGNBDeviceStatisticsDataList.action';
            }else if(code.slice(0,3) == 'gsm'){
                urls = '${ctx}/system/device/getGSMDeviceStatisticsDataList.action';
            }
            vm.chartLoadingData[code] = true;
		   	$.ajax({
				   url: urls, 
				   type: 'post',
				   data: queryParams, 
				   dataType: 'json',
				   success: function(data){
					    if(data && data.length) {
							params.data = data;
                            if(index === 6){
                                vm.createLineNowTotalData(code, data);
                            }
						}
						vm.charts[code].setOption(vm.createOption(params));  // 根据请求的数据重新渲染图表
						vm.resizeChart();
                        vm.chartLoadingData[code] = false;
				   },
				   error: function() {
						vm.charts[code].setOption(vm.createOption(params));
						vm.resizeChart();
                        vm.chartLoadingData[code] = false;
				   }
			})
		},
		/**
		* CPE统计数据刷新
		* @param code{string} 图表类型
		* @param index{number}  时间轴下标
		*/
		proccessCPE(code,index){
			var vm = this,
				s_time = getYesterDay(6-index).substring(0,10)+' 00:00:00',
				e_time = getYesterDay(6-index).substring(0,10)+' 23:59:59',
		   		unitName='<%=rb.getString("GeShu")%>';
			
			if(index==6) {
				e_time = getNowTimeToZoneTimeRange(timeZone, 144).end_time;
			}else{
				e_time = getYesterDay(6-index-1).substring(0,10)+' 00:00:00';
			}
			var queryParams = {
					device_type: "cpe",
					time_level: "min",
					timeZone: timeZone,
					start_time: s_time,
					end_time: e_time
		 		},
				params = {
					legend: ['Total','Onlince'],
                    lineLength: '2',
					legendNames: ['<%=rb.getString("ShouYe_ZongShu")%>','<%=rb.getString("ShouYe_ZaiXian")%>'],
					data: [],
					ueFinal: [],
					unit: unitName,
					pointerCount: 6,
                    period:'min',
					index: index,
					code: code,
					color: ['#1F77B4','#0ABF5B'],
				};
		   	vm.chartLoadingData[code] = true;
		   	$.ajax({
				   url: "${ctx}/system/device/getDeviceStatisticsDataList.action", 
				   data: queryParams, 
				   type: 'post',
				   dataType: 'json',
				   success: function(data){
						if(data && data.length) {
                            params.data = data;
                            if(index === 6){
                                vm.createLineNowTotalData(code, data);
                            }
                        } 
						vm.charts[code].setOption(vm.createOption(params));
                        vm.resizeChart();
                        vm.chartLoadingData[code] = false;
				   },
				   error: function() {
						vm.charts[code].setOption(vm.createOption(params));
                        vm.resizeChart();
                        vm.chartLoadingData[code] = false;
				   }
			});
		},
		/**
		* 性能统计数据刷新
		* @param code{string} 图表类型
		* @param index{number}  时间轴下标
		*/
		proccessTraffic(code,index){
			var vm = this,
                urls = '${ctx}/pm/template/queryStatisKPIChartData.action',
				s_time = getYesterDay(6-index).substring(0,10)+' 00:00:00',
				e_time = getYesterDay(6-index).substring(0,10)+' 23:59:59';
            /**
			if(index==6) {
				e_time = getNowTimeToZoneTimeRange(timeZone, 144).end_time;
			}else{
				e_time = getYesterDay(6-index-1).substring(0,10)+' 00:00:00';
			}
            */
            e_time = getYesterDay(6-index-1).substring(0,10)+' 00:00:00';
		    var queryParams = {
		    		kpiIdList : vm.counterCodes[code],
		    		timeZone : timeZone,
					startTime: s_time,
		            endTime : e_time,
		            statisPeriod : "60"   //3代表小时
		    };

            var params = {
                    legend: vm.counterCodes[code].split(','),
                    lineLength: '2',
                    legendNames: vm.kpiTitleCodes[code].split(','),
                    data: [],
                    unit: vm.unitCodes[code],
                    pointerCount: 1,
                    period:'min',
                    index: index,
                    code: code,
                    title: '',
                    color: ['#1F77B4']
            };
            if(code.slice(0,3) == 'gnb'){
                urls = '${ctx}/gnb/pm/template/queryStatisKPIChartData.action';
            }else if(code.slice(0,3) == 'gsm'){
                urls = '${ctx}/pm/template/queryGSMStatisKPIChartData.action';
            }
           if(vm.periodTypeData[code] && vm.periodTypeData[code] == 'min'){
                params.period = 'min';
                params.legend = [vm.counterCodes[code] + '_yesterday', vm.counterCodes[code]];
                params.legendNames = ['<%=rb.getString("ZuoTian")%>','<%=rb.getString("JinTian")%>'];
                params.color = ['#999999','#1F77B4'];
            }else{
                params.period = 'week';
                queryParams.statisPeriod = '1440';
                params.lineLength =  '1';
                queryParams.startTime = getYesterDay(7).substring(0,10) + ' 00:00:00';
                queryParams.endTime = getYesterDay(0).substring(0,10) + ' 00:00:00';
            }
            vm.chartLoadingData[code] = true;
			$.ajax({
				url: urls,
				type: 'post',
				data: queryParams,
				dataType: 'json',
				success: function(data){
					//上下行吞吐量(Mbps)
					if(data && data.length>0){
						params.data = data;
                        if(index === 6){
                            vm.createLineNowTotalData(code, data);
                        }
					}
					vm.charts[code].setOption(vm.createOption(params)); // 根据请求的数据重新渲染图表
                    vm.resizeChart();
                    vm.chartLoadingData[code] = false;
				},
				error: function(){
					vm.charts[code].setOption(vm.createOption(params));
                    vm.resizeChart();
                    vm.chartLoadingData[code] = false;
				}
			});
		},
		/**
		* Sit 统计数据刷新
		* @param code{string} 图表类型
		* @param index{number}  时间轴下标
		*/
		proccessSite(code,index){
			var vm = this,
				s_time = getYesterDay(6-index).substring(0,10)+' 00:00:00',
				e_time = getYesterDay(6-index).substring(0,10)+' 23:59:59';
			
			if(index==6) {
				e_time = getNowTimeToZoneTimeRange(timeZone, 144).end_time;
			}else{
				e_time = getYesterDay(6-index-1).substring(0,10)+' 00:00:00';
			}
		    var param = {
		    		device_type: 'enb',
					time_level: "min",
					timeZone: timeZone,
					start_time: s_time,
					end_time: e_time
		    	},
				unit = '<%=rb.getString("GeShu")%>',
				title = '';
            var params = {
                legend: ['Total','Offline'],
                lineLength: '2',
                legendNames: ['<%=rb.getString("ShouYe_ZongShu")%>','<%=rb.getString("ShouYe_BuZaiXian")%>'],
                query: param,
                data: [],
                unit: unit,
                pointerCount: 6,
                period:'min',
                index: index,
                code: code,
                title: title,
                color: ['#1F77B4','#FF5B45']
            };
            vm.chartLoadingData[code] = true;
			$.ajax({
				url: '${ctx}/site/getSiteStatisticsInfos.action',
				type: 'post',
				data: param,
				dataType: 'json',
				success: function(data){
					//上下行吞吐量(Mbps)
					if(data && data.length>0){
						params.data = data;
                        if(index === 6){
                            vm.createLineNowTotalData(code, data);
                        }
					}
					vm.charts[code].setOption(vm.createOption(params)); // 根据请求的数据重新渲染图表
                    vm.resizeChart();
                    vm.chartLoadingData[code] = false;
				},
				error: function(){
					vm.charts[code].setOption(vm.createOption(params));
                    vm.resizeChart();
                    vm.chartLoadingData[code] = false;
				}
			});
		},
        createLineNowTotalData(code, data){
            var vm = this;

            if(vm.chartList['kpiChartList'].includes(code)){
                var nowDatas = data[data.length-1];
                nowDatas['count'] = nowDatas['nowVal'] ? nowDatas['nowVal'] : '-';
                nowDatas['count_status'] = nowDatas['nowState'] ? nowDatas['nowState'] : '-';
                Object.keys(vm.lineNowTotalData[code]).map((item)=>{
                    if(nowDatas[item] != undefined){
                        vm.lineNowTotalData[code][item] = nowDatas[item];
                    }else{
                        vm.lineNowTotalData[code][item] = '-';
                    }
                });
            }else{
                var nowDatas = data[data.length-1],
                    yesterDatas = data[data.length-2];

                if(nowDatas){
                    if(code == 'enbMmeStatus'){
                        nowDatas['mme_disconnected_count'] = nowDatas['device_count'] - nowDatas['mme_connected_count'];
                    }
                    Object.keys(vm.lineNowTotalData[code]).map((item)=>{
                        if(nowDatas[item] != undefined){
                            vm.lineNowTotalData[code][item] = nowDatas[item];
                        }else{
                            vm.lineNowTotalData[code][item] = '-';
                        }
                    });
                }
                if(nowDatas&& yesterDatas){
                    if(code == 'enbMmeStatus'){
                        yesterDatas['mme_disconnected_count'] = yesterDatas['device_count'] - yesterDatas['mme_connected_count'];
                    }
                    Object.keys(vm.lineNowTotalData[code]).map((item)=>{
                        if(item.slice(-6) != 'status'){
                            if(nowDatas[item] != undefined && nowDatas[item] != null && nowDatas[item] != '-' && yesterDatas[item] != undefined && yesterDatas[item] != null && yesterDatas[item] != '-'){
                                if(nowDatas[item] - yesterDatas[item] > 0){
                                    vm.lineNowTotalData[code][item + '_status'] = '1';
                                }else if(nowDatas[item] - yesterDatas[item] < 0){
                                    vm.lineNowTotalData[code][item + '_status'] = '2';
                                }else if (nowDatas[item] === yesterDatas[item]){
                                    vm.lineNowTotalData[code][item + '_status'] = '3';
                                }
                            }
                        }
                    });
                }
            }
        },
		/**
		* 生成图表的 option
		* @param params{object} 需渲染图表的请求信息
		*/
		createOption(params){// 生成图表的 option
			var vm = this, pointerCount = params.pointerCount, title = params.title,
				finalArr = vm.proccessData(params), startMinDashboard = ' 00:00:00',
				index = params.index, code = params.code, unit =  params.unit,
				legend = params.legend || [], lineLength = params.lineLength, legendNames = params.legendNames,
				type1 = legend[0], type2 = legend[1],type3 = legend[2],period = params.period,
				dates = vm.getDays(),colors = params.color, showTimeline = (period == 'week') ? false : true;
				option = {
					baseOption: {
						title: {
							subtext: '',
							top: 0,
							subtextStyle: {
								color: '#333',
								fontWeight:'bold'
							}
						},
						timeline: {
                            show: showTimeline,
							axisType: 'category',
							controlPosition: 'none',
						    symbolSize:8,
						  	lineStyle : { color : '#666666',width : 1 },
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
			            	  	color:'#1F77B4',
			            	  	borderColor:'none'
			              	}
						},
			            tooltip : {
			              trigger : 'axis',
			              formatter:function(params){
			              	var timeStr = "";
			              	var params = JSON.parse(JSON.stringify(params));
			  	            var str = "";
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
							  
								str += "<div>" + params[i].seriesName + ": " + params[i].data.value + "</div>";
                                if(code == 'enbOnline' || code == 'gnbOnline' || code == 'gsmOnline' || code == 'cpeOnline') { 
									if(params[i].seriesIndex == 1) {
										str += '<%=rb.getString("ZaiXianLv")%>: ' + (['',null,undefined].includes(params[i].data.cn3)?'-':params[i].data.cn3) + '%</div>';
									}
								}
                                if(code == 'enbActive' || code == 'gnbActive' || code == 'gsmActive') {
                                    if(params[i].seriesIndex == 1) {
										str += '<%=rb.getString("JiHuoLv")%>: ' + (['',null,undefined].includes(params[i].data.cn3)?'-':params[i].data.cn3) + '%</div>';
									}
                                }
							}
			              	return str;
			              },
						  confine: true,
			            },
			          	grid:{
                            top: 75,
			        	  	bottom: showTimeline ? 60 : 10,
							left: 30,
							right: 60,
							containLabel:true
			          	}, 
				        color: colors,
						legend: { 
                            icon: 'circle',
                            itemWidth: 6,   // 圆点宽度
                            itemHeight: 6,  // 圆点高度
							data: legendNames,
							top: 5,
                            right: 5
						},
						xAxis : [{
	                        name : showTimeline ? '<%=rb.getString("XiaoShi")%>' : '<%=rb.getString("Tian")%>',
		                    type : 'category',
		                    boundaryGap : false,
					        axisLine:{
					        	show : true,
				        		lineStyle:{ color:"#666666" }
				        	},
							axisLabel : {
								show:true,
								textStyle:{ color:"#666666" },
								lineStyle:{ color:"#666666" },
								formatter : function(val) {
                                    if(val == null || val == undefined) return '';
                                    var clock = '', month='', day='';
                                    if(period == 'min'){
                                        var secondTime = val.split(' ')[1];
                                        clock = secondTime.substring(0,2);
                                        val = secondTime.substring(0,5);
                                        if(secondTime.substring(3,5)=='00') return clock;
                                    }else if(period == 'week'){
                                        var secondTime = val.split(' ')[0];
                                        month = secondTime.split('-')[1];
                                        day = secondTime.split('-')[2];
                                        val = month+'/'+day
                                    }
                                    return val;
								},
								interval : function(index){
									if(index%pointerCount == 0){//  && index != pointerCount*24
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
							}
						}],
				        yAxis : [{
		                    name: unit,
		            	    minInterval: (vm.chartList['kpiChartList'].includes(code)) ? null:1,
		                    type: 'value',
		                    axisLabel : {
					        	show:true,
					        	textStyle:{ color:"#666666" },
		                    	lineStyle:{ color:"#666666" }
					        },
					        axisLine:{
                                show: true,
				        		lineStyle:{ color:"#666666" }
				        	},
				        	splitLine : {
				        		lineStyle:{ color:"#f1f1f4" ,type:'dashed' }
				        	}
		                }],
		        		series : [{  
				                 	name: legendNames[0],
				                	type: 'line',
                                    symbol: 'circle',
				  					symbolSize: 5,
				  		        	showAllSymbol: true,
				                	step: false,
				                	connectNulls: false,
									itemStyle: {
										normal: {
                                            lineStyle: {
                                                type: (vm.chartList['kpiChartList'].includes(code) && period == 'min') ? 'dashed' : 'solid',
                                                width: (vm.chartList['kpiChartList'].includes(code) && period == 'min') ? 1 : 2
                                            },
											areaStyle: {opacity: 0}
										}
									},
									areaStyle: {opacity: 0}
				                  },
				                  {  
				                	name: legendNames[1],
				                	type: 'line',
                                    symbol: 'circle',
				  					symbolSize: 5,
				  		         	showAllSymbol : true,
				                	step: false,
				                	connectNulls: false,
									itemStyle: {
										normal: {
											areaStyle: {opacity: 0}
										}
									},
									areaStyle: {opacity: 0}
				                  },
				                  {  
				                	name: legendNames[2],
				                	type: 'line',
                                    symbol: 'circle',
				  					symbolSize: 1,
				  		         	showAllSymbol : true,
				                	step: false,
				                	connectNulls: false,
									itemStyle: {
										normal: {
											areaStyle: {opacity: 0}
										}
									},
									areaStyle: {opacity: 0}
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
				                  {data: []},
								  {data: []}
				              ]  
				            });
			}
			option.options = subOpts;
			// 设置当前选中的日期节点数据
			option.options[index].xAxis[0].data = finalArr[getYesterDay(6-index)]['x'];
            if(lineLength == '1'){
                option.options[index].series[0].data = finalArr[getYesterDay(6-index)][type1];
            }else{
                option.options[index].series[0].data = finalArr[getYesterDay(6-index)][type1];
                option.options[index].series[1].data = finalArr[getYesterDay(6-index)][type2];
            }
			if(lineLength == '3') {
				option.options[index].series[2].data = finalArr[getYesterDay(6-index)][type3];
			}

			return option;
		},
		/**
		* 图表信息
		* @param params{object} 需渲染图表的请求信息
		*/
		proccessData(params){
			var vm = this, startMinDashboard = ' 00:00:00',
				pointerCount = params.pointerCount,
                period = params.period,lineLength = params.lineLength,
				chart_data = params.data, chart_id = params.code,
				deviceCount = "", pointerNum = period == 'week'? 7 : pointerCount*24+1,
				legend_code = params.legend, legend_name = legend_code,
				legend_code_line1 = legend_code[0], legend_code_line2 = legend_code[1],
				type1 = legend_code[0], type2 = legend_code[1],type3 = legend_code[2];
			// [yyyy-MM-dd, HH:mm:ss, line1, line2, line3]
		    dataArr=[];
		    $.each(chart_data,function(index,item){
                if(vm.chartList['kpiChartList'].includes(chart_id)){
					var xDate = item.end_time.split(' ');
		    	}else{
					var xDate = item.statistics_time.split(' ');
		    	}
			    var lineOneCount = "";    
				var lineTwoCount = "";
                var lineThreeCount = "";
				if(['siteOffline'].includes(chart_id)){
					lineOneCount = item.total;
					lineTwoCount = item.offline_count;
					
					dataArr[index]=[xDate[0],xDate[1],lineOneCount,lineTwoCount];
				}else if(['enbOnline','gnbOnline','gsmOnline','cpeOnline'].includes(chart_id)){
					lineOneCount = item.device_count;
					lineTwoCount = item.online_count;
                    lineThreeCount = item.online_rate;
					
					dataArr[index]=[xDate[0],xDate[1],lineOneCount,lineTwoCount, lineThreeCount];
				}else if(['enbActive','gnbActive','gsmActive'].includes(chart_id)){
					lineOneCount = item.device_count;
					lineTwoCount = item.active_count;
                    lineThreeCount = item.active_rate;
					dataArr[index]=[xDate[0],xDate[1],lineOneCount,lineTwoCount, lineThreeCount];
				}else if(['enbUeCount','gnbUeCount','gsmUeCount'].includes(chart_id)){
					lineOneCount = item.ue_count;
					lineTwoCount = item.device_count;
					
					dataArr[index]=[xDate[0],xDate[1],lineOneCount,lineTwoCount];
				}else if(vm.chartList['kpiChartList'].includes(chart_id)){ 
					 if (item[legend_code_line1] != null && item[legend_code_line1] != 'N/A' && item[legend_code_line1] != '-') {
		             	lineOneCount = Number(item[legend_code_line1]);
		             } else {
		     			lineOneCount = '-';
		             }
					 if (item[legend_code_line2] != null && item[legend_code_line2] != 'N/A' && item[legend_code_line2] != '-') {
		             	lineTwoCount = Number(item[legend_code_line2]);
		             } else {
		    			lineTwoCount = '-';
		             }
					dataArr[index]=[xDate[0],xDate[1],lineOneCount,lineTwoCount];
					if(dataArr && dataArr[0] && dataArr[0][1]){
						startMinDashboard = " 00:" + dataArr[0][1].split(":")[1] + ":00";
					}
				}else if(['enbMmeStatus'].includes(chart_id)){
					lineOneCount = item.mme_connected_count;
					lineTwoCount = item.device_count - lineOneCount;
					dataArr[index]=[xDate[0],xDate[1],lineOneCount,lineTwoCount];
				}
			});
		    finalArr = [];//[2017-01-01,00:00:00,0,1,'25' ]
			for(var initIndex=0;initIndex<8;initIndex++ ){
				var arrIndex = getYesterDay(initIndex-1);
				if(!finalArr[arrIndex]){
					var activeCountArr = [], yaxisArr = [],lineOneCountArr=[],lineTwoCountArr=[],lineThreeCountArr=[];
					for(var axisIndex=0; axisIndex<pointerNum; axisIndex++){
						var yaxisStartTime = getYesterDay(initIndex-1).substring(0,10)+ startMinDashboard,
							offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*(60/pointerCount));
                        if(period == 'week') {
                            yaxisStartTime = addDate(yaxisStartTime, -6);
                            offsetTime = addDate(yaxisStartTime, axisIndex);
                        }
						yaxisArr.push(formatDate(offsetTime));
                        
						lineOneCountArr.push({value: "-"});
						lineTwoCountArr.push({value: "-"}); 
						lineThreeCountArr.push({value: "-"}); 
					}
					finalArr[arrIndex] = [];
					finalArr[arrIndex]['x'] = yaxisArr; //x轴数据
                    if(lineLength == '1'){
                        finalArr[arrIndex][type1] = lineOneCountArr;
                    }else{
                        finalArr[arrIndex][type1] = lineOneCountArr;
					    finalArr[arrIndex][type2] = lineTwoCountArr;
                    }
                    if(lineLength == '3'){
						finalArr[arrIndex][type3] = lineThreeCountArr;
					}
				}
			}
            if(['week'].includes(period)) {
                var finaDateIndex = getYesterDay(0).substring(0,10);
                $.each(dataArr,function(n,m){
                    var dateIndex = m[0],yValue=m[1].substring(0,5),lineOneCountArrValue=m[2],lineTwoCountArrValue=m[3],lineThreeCountArrValue=m[4];
                    finalArr[finaDateIndex][type1].splice(n,1,{value:lineOneCountArrValue}); 
                });
            }else{
                $.each(dataArr,function(n,m){
                    var dateIndex = m[0],yValue=m[1].substring(0,5),lineOneCountArrValue=m[2],lineTwoCountArrValue=m[3],lineThreeCountArrValue=m[4];
                    try{
                        if(finalArr[dateIndex]['x'].includes(dateIndex+' '+m[1])){
                            var differTimes = new Date(dateIndex+' '+m[1]).getTime() - new Date(dateIndex+' 00:00:00').getTime();
                            var axisDateIndex = Math.round(differTimes/(1000*60*(60/pointerCount)));
                            if(lineLength == '1'){
                                finalArr[dateIndex][type1].splice(axisDateIndex,1,{value:lineOneCountArrValue}); 
                            }else{
                                finalArr[dateIndex][type1].splice(axisDateIndex,1,{value:lineOneCountArrValue,cn3:lineThreeCountArrValue});
                                finalArr[dateIndex][type2].splice(axisDateIndex,1,{value:lineTwoCountArrValue,cn3:lineThreeCountArrValue});
                            }
                            if(lineLength == '3') {
                                finalArr[dateIndex][type3].splice(axisDateIndex,1,{value:lineThreeCountArrValue}); 
                            }
                        }
                    }catch(e){}
                });
            }
            if(['min'].includes(period)) {
                var prevDayData = "";
                for(var key in finalArr ){
                    if(prevDayData){//
                        //['Connected','Disconnected','Active','Inactive','Online','Offline','ueCount','UL','DL'].map(function(itemCode){
                        legend_code.map(function(itemCode){
                            if(finalArr[key][itemCode]){
                                finalArr[key][itemCode][finalArr[key][itemCode].length-1] = prevDayData[itemCode][0];
                            }
                        });
                        prevDayData = finalArr[key]
                    }else prevDayData = finalArr[key]
                }
            }
			return finalArr;
		},
		// 获取天粒度的 time Data
		getDays() {
			var vm = this,
				lineDays = [],
				start = new Date(gloableTime);
			
			for(var i = 6; i>=0; i--) {
				var dateStr = dateformatter(addDate(start,-i));
				lineDays.push(dateStr.substr(0,10));
			}
			
			return lineDays;
		},
		// 初始化产品类型图表
		initProductType(type) {
			var vm = this,chartDom = '',titleText = '',
				urls = '';
			if(type == 'enb'){
				chartDom = document.querySelector('#enbProductType');
				urls = '${ctx}/system/device/getEnbProductStatisticsData.action';
				titleText = '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>';
			}else if(type == 'cpe'){
				chartDom = document.querySelector('#cpeProductType');
				urls = '${ctx}/system/device/getCpeModuleStatisticsData.action';
				titleText = '<%=rb.getString("ChanPinXingHao")%>';
			}else if(type == 'gnb'){
				chartDom = document.querySelector('#gnbProductType');
				urls = '${ctx}/system/device/getGNBProductStatisticsData.action';
				titleText = '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>';
			}else if(type == 'gsm'){
                chartDom = document.querySelector('#gsmProductType');
				urls = '${ctx}/system/device/getGSMProductStatisticsData.action';
				titleText = '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>';
            }
			if(!chartDom) return;
			
			var legend = [],
				sdata = [],
				splitNum = 10,
				colorList = ['#69b6fc','#90ec97','#e9a4a4','#f3cc90','#d7a3ef','#ada3ef','#4bdedb','#4bdea3','#e6a46b','#efa3d3','#f5f5f5'];
			
			$.post(urls,function(data){
				if(data) {
					var total = 0,
						others = [];
					data.map(function(item){
						total += item.device_count*1;
					});
					data.map(function(item,index){
						var name = '';
						if(type == 'enb' || type == 'gnb' || type == 'gsm'){
							name = item.product_type+ ': ' +item.device_count+ ' (' + (item.device_count*100/total).toFixed(2) + '%)';
						}else if(type == 'cpe'){
							name = item.module_name+ ': ' +item.device_count+ ' (' + (item.device_count*100/total).toFixed(2) + '%)';
						}
						
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
					colorList.push('#e9e9e9');
				}
				
				vm.charts[type + 'ProductType'].setOption({
					title: {
						top: 10,
						left: 10,
						text: titleText,
						textStyle: {
							fontSize: 12
						}
					},
					color: colorList, 
					tooltip: {
						trigger: 'item',
						formatter: '{b}',
						confine: true
					},
					grid: {
						top: '20%',
						left: '10%',
						right: '15%',
						bottom: '5%'
					},
					series: [
						{
							name: 'product_type_chart',
							type: 'pie',
							radius: ['30%','60%'],
							avoidLabelOverlap:true,
							center: ['50%','60%'],
							data: sdata,
							itemStyle:{
								borderRadius:10,
								borderColor:'#fff',
								borderWidth:2
							},
							emphasis: {
								label:{
									show:true,
									fontSize:'40',
									fontWeight:'bold'
								}

							},
							label: {
								show: true,
								fontSize: 8,
								position:'center'
							},
							labelLine: {
								show: false
							}
						}
					]
				});
				vm.resizeChart();
			},'json');

		},
		// 初始化enb设备运行时长图表
		initDeviceRunTime(type) {
			var vm = this,chartDom = '',
				urls = '';
			if(type == 'enb'){
				chartDom = document.querySelector('#enbDeviceRunningTime');
				urls = '${ctx}/system/device/getDeviceEnbUpTimeStatisticsData.action';
			}else if(type == 'cpe'){
				chartDom = document.querySelector('#cpeDeviceRunningTime');
				urls = '${ctx}/system/device/getCpeUpTimeRealData.action';
			}else if(type == 'gnb'){
				chartDom = document.querySelector('#gnbDeviceRunningTime');
				urls = '${ctx}/system/device/getDeviceGNBUpTimeStatisticsData.action';
			}else if(type == 'gsm'){
                chartDom = document.querySelector('#gsmDeviceRunningTime');
                urls = '${ctx}/system/device/getDeviceGSMUpTimeStatisticsData.action';
            }
			if(!chartDom) return;
			
			var legend = ['<10days','10-30days','30-90days','>90days'],
                series = [],
				barData = [];
			$.post(urls,function(data){
				if(data) {
					barData = [
						data.tenDayDeviceCount,
						data.thirtyDayDeviceCount,
						data.ninetyDayDeviceCount,
						data.bigNinetyDayDeviceCount
					];
					var total = barData[0]+barData[1]+barData[2]+barData[3],
						lgd1 = '<10days: ' + barData[0] + ' (' + (barData[0]*100/total).toFixed(2)+'%)',
						lgd2 = '10-30days: ' + barData[1] + ' (' + (barData[1]*100/total).toFixed(2)+'%)',
						lgd3 = '30-90days: ' + barData[2] + ' (' + (barData[2]*100/total).toFixed(2)+'%)',
						lgd4 = '>90days: ' + barData[3] + ' (' + (barData[3]*100/total).toFixed(2)+'%)';

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
							data: barData,
							barWidth: 20,
							itemStyle: {
								normal: {
									color: function(param){
										return ['#99CCFF','#99CCFF','#99CCFF','#99CCFF'][param.dataIndex];
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
				
				vm.charts[type + 'DeviceRunningTime'].setOption({
					title: {
						top: 10,
						left: 10,
						text: '<%=rb.getString("SheBeiYunXingShiChang")%>',
						textStyle: {
							fontSize: 12
						}
					},
					color: ['#99CCFF'],
					grid: {
						top: 50,
						left: gridLeft,
						right: 45,
						bottom: 30,
						containLabel:true
					},
					tooltip: {
						trigger: 'axis',
						formatter: function(p) {
							var ops = p[0];
							
							return legend[ops.dataIndex];
						},
						confine: true
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
							show: true
						},
						axisTick: {
							show: true,
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
		},
        // 首页告警数量跳转
        showCurrAliveAlarm(type){
            var vm = this;
            try{
                var params = {
                    alarm_severity: '', 
                    unread: '',
                    search_text: '',
                };
                if(type == 'siteOffLineAlarm'){
                    params.uniqueAlarmIdentifier = '36';
                }else if(type == 'cellOfflineAlarm'){
                    params.uniqueAlarmIdentifier = '7,23,4';
                }else if(type == 'cellInactiveAlarm'){
                    params.uniqueAlarmIdentifier = '11184,50101,60003,11432,10060';
                }
                eventAllBus.$emit("gomenupage","8000","",'8000',params,function() {
                    alarmViewVue.activeName = 'alarmView';
                    alarmViewVue.resetQueryParams(params);
                });
                if(alarmViewVue) {
                    alarmViewVue.activeName = 'alarmView';
                    alarmViewVue.resetQueryParams(params);
                }
            }catch(e){}
        },
        // 首页告警数量导出
        alarmStatisticExport(type){
            var vm = this,
                urls = '${ctx}/fault/view/exportViewPageList.action',
                params = {
                    timeZone: timeZone,
                    alarmType: 'ACTIVE',
                    queryType: 'View',
                    uniqueAlarmIdentifier:'',
                };
            if(type == 'siteOffLineAlarm'){
                params.uniqueAlarmIdentifier = '36';
            }else if(type == 'cellOfflineAlarm'){
                params.uniqueAlarmIdentifier = '7,23,4';
            }else if(type == 'cellInactiveAlarm'){
                params.uniqueAlarmIdentifier = '11184,50101,60003,11432,10060';
            }
            exportByForm(urls,params); // 发送导出请求
        },
        // KPI 天和周粒度切换
        periodTypeChange(code){
            var vm = this;
            vm.reloadChart(code,6);
        },
        // 设备状态统计 KPI统计 导出
        deviceAndKpiStatisticExport(){
            var vm = this,
                urls = '${ctx}/dashboard/exportStatisticsReport.action',
                devicePermsList = [],
                kpiPermsList = [],
                params = {
                    timeZone: timeZone,
                    devicePerms: '',
                    kpiPerms: ''
                };
            
            // 根据 visibleCode 权限生成 devicePerms
            if(vm.visibleCode['siteStatistic']){
                devicePermsList.push('SITE');
            }
            if(vm.visibleCode['enbStatistic']){
                devicePermsList.push('eNB');
            }
            if(vm.visibleCode['gnbStatistic']){
                devicePermsList.push('gNB');
            }
            if(vm.visibleCode['gsmStatistic']){
                devicePermsList.push('GSM');
            }
            params.devicePerms = devicePermsList.join(',');
            
            // 根据 visibleCode 权限生成 kpiPerms
            if(vm.visibleCode['kpiStatistic']){
                if(vm.visibleCode['enbStatistic']){
                    kpiPermsList.push('eNB');
                }
                if(vm.visibleCode['gnbStatistic']){
                    kpiPermsList.push('gNB');
                }
                if(vm.visibleCode['gsmStatistic']){
                    kpiPermsList.push('GSM');
                }
                params.kpiPerms = kpiPermsList.join(',');
            }else{
                params.kpiPerms = '';
            }
            
            exportByForm(urls,params); // 发送导出请求
        }
	},
	mounted(){
		this.init();
		eventBus.$off('dashboard-resize').$on('dashboard-resize',this.resizeChart);
	}
})
</script>