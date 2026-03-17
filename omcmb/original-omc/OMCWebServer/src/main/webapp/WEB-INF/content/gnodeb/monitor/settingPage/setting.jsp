<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<link rel="stylesheet" href="${ctx}/js/topo/leaflet.css" type="text/css" media="screen" />
<link rel="stylesheet" href="${ctx}/js/topo/dvf.css" type="text/css" media="screen"/>
<link rel="stylesheet" href="${ctx}/js/topo/node.css" type="text/css" media="screen"/>

<style>
	.enbNav {
		width:200px;
		border-right:1px solid #d5dcec;
	}
	.settingMainPage {
		flex:1 ;
		display:flex;
		flex-direction:column;
		overflow: auto;
	}
	.el-card__body {
		flex:1 auto;
		overflow:auto;
	}
	.navItem {
		height:40px;
		line-height:40px;
		border-bottom:1px solid #d5dcec;
		padding:0 15px;
		font-size:14px;
		cursor:pointer;
		color: rgba(0, 0, 0, 0.8);
	}
	.navItem:hover ,.navItem.active  {
		background-color: rgba(var(--main-color-rgba1),0.08);
		color:var(--main-color);
	}
	.navItem .el-icon ,.navItemTitle .el-icon{
		margin-right:10px;
		color:unset;
		font-size: 16px;
	}
	.navItem .el-icon:before {
		color:unset;
	}
	#gnbSetting_main {
		padding:10px;
		background:#f5f5f5;
	}
	.navItemGroup {
		border-bottom:1px solid #d5dcec;
	}
	.navItemTitle {
		color:#999;
		padding: 15px 15px 10px 15px;
		font-size:14px;
	}
	.navItemGroup .navItem {
		border:none;
	}
	.navItemGroup .navItem {
		padding-left:48px;
	}
	.navItemGroup.advanceGroup .navItem {
		padding-left:20px;
	}
	.rightBoxHeader {
		height: 40px; 
		line-height: 40px !important;
		background: #FFFFFF;
		justify-content: space-between; 
		border-bottom: 1px solid #D5DCEC;
		font-size: 14px;
		padding: 0 20px;
		position: relative;
	}
	#gnbSettingPage .el-icon-menu-upgrade:before {
		font-size: 16px;
	}
	.gnbConfigAddDialog .gnbConfigAddMainBoxCls{
		height: 400px;
		width: 100%;
		position: relative;
		overflow-y: auto!important;
		overflow-x:hidden;
	}
	.gnbConfigAddDialog .el-dialog__header .el-icon:before{
		font-size: 16px;
	}
	.gnbConfigAddDialog .el-form{
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		width: 100%;
		position: relative;
	}
	.gnbConfigAddDialog .el-form-item{
		display: inline-block;
		width: 48%;
	}
	.gnbConfigAddDialog .el-form-item .el-form-item__label{
		font-size: 12px;
	}
	.gnbConfigAddDialog .el-form-item__error{
		padding-top: 0px;
		top:30px!important;
	}
	.gnbConfigAddDialog .selectErrcCls .el-form-item__error{
		position: absolute;
		left: 210px;
		top:5px !important;
	}
	#gnbSettingPage .validate-item .el-input__inner,
	.gnbConfigAddDialog  .validate-item .el-input__inner{
		width:200px;
	}
	#gnbSettingPage .validate-item .el-input-group__append,
	.gnbConfigAddDialog .validate-item .el-input-group__append{
		border:none;
		background:none;
		padding: 0px 10px;
	}
	#gnbSettingPage .validate-item .el-form-item__error, 
	.gnbConfigAddDialog .validate-item .el-form-item__error{
		display:none;
	}
	#gnbSettingPage .is-error .el-input-group__append,
	.gnbConfigAddDialog .is-error .el-input-group__append{
		color:#FA5555;
	}

	#gnbSettingPage .el-collapse-item__header,.gnbConfigAddDialog .el-collapse-item__header{
		border-bottom:1px solid #fff;
	}
	#gnbSettingPage .el-collapse-item__arrow,.gnbConfigAddDialog .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:0px;
	}
	#gnbSettingPage .el-collapse-item,.gnbConfigAddDialog .el-collapse-item{
		position:relative;
		border-bottom:1px solid #E9E9E9;
	}
	#gnbSettingPage .el-collapse-item__content,.gnbConfigAddDialog .el-collapse-item__content{
		margin: 0px 40px;
		padding-bottom: unset;
	}
	#gnbSettingPage .el-collapse-item__header .el-icon-arrow-right,.gnbConfigAddDialog .el-collapse-item__header .el-icon-arrow-right{
		font-size:16px;
	}
	#gnbSettingPage .el-collapse-item__header .el-icon-arrow-right:before,.gnbConfigAddDialog .el-collapse-item__header .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#gnbSettingPage .el-collapse-item__header .is-active.el-icon-arrow-right:before,.gnbConfigAddDialog .el-collapse-item__header .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	#gnbSettingPage .el-collapse-item__arrow.is-active,.gnbConfigAddDialog .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#gnbSettingPage .el-collapse,.gnbConfigAddDialog .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
	}
	.gnbConfigAddDialog .el-collapse{
		width: 100%;
	}
	#gnbSettingPage .el-collapse-item__wrap,.gnbConfigAddDialog .el-collapse-item__wrap{
		border-bottom:1px solid #fff;
		padding-left: 0px;
	}
	#gnbSettingPage .el-collapse-item__header,.gnbConfigAddDialog .el-collapse-item__header{
		max-width:800px;
	}
	.gnbConfigAddDialog .PlmnListItemCls{
		border: 1px solid #DFE2EE;
		position: relative;
		margin-bottom: 20px;
		padding: 20px;
	}
	.gnbConfigAddDialog .plmnListDelIcon{
		height: 26px;
		width: 26px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: 1px solid #DFE2EE;
		background-color: #FFFFFF;
		border-radius: 100px;
		position: absolute;
		right:-12px;
		top:-12px;
		z-index: 231
	}
	.gnbConfigAddDialog .sliceListBoxCls .el-collapse-item__content{
		margin: 0px!important;
	}
	.gnbConfigAddDialog .sliceListBoxCls .el-collapse-item{
		border-bottom:none!important;
	}
	.gnbConfigAddDialog .sliceListBoxCls .plmnItemSliceListCls{
		padding: 20px;
		width: 95%;
		border: 1px solid #DFE2EE;
		background-color: #FAFAFD;
		border-radius: 4px;
		position: relative;
		margin-bottom: 20px;
	}
    .gnbConfigAddDialog .el-checkbox.is-bordered.el-checkbox--small{
        height: 28px;
    }
	.el-ctable .hidden-row{
		display: none;
	}
	#gnbSettingPage  .allowMoreInputBoxCls{
		position: relative;
		flex: 1;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls{
		margin-bottom: 5px;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTitleCls{
		font-size: 14px;
		color: rgba(0, 0, 0, 0.8);
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputHeadCls .allowMoreInputTipsCls{
		font-size: 14px;
		color: rgba(0, 0, 0, 0.32);
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputContentCls{
		border: 1px solid #DFE2EE;
		width: 80%;
		min-width: 600px;
		min-height: 78px;
		padding: 10px;
		border-radius: 4px;
		box-sizing: border-box;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls{
		display: flex;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .el-input{
		width: 240px;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls{
		height: 26px;
		width: 56px;
		display: flex;
		align-items: center;
		justify-content: center;
		color:var(--main-color);
		border: 1px solid var(--main-color);
		background:rgba(var(--main-color-rgba1),0.1);
		border-radius: 4px;
		box-sizing: border-box;
		margin-left: 10px;
		cursor: pointer;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddTipCls{
		margin-left: 10px;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls .el-icon::before{
		font-size: 16px;
		color:var(--main-color);
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputFieldCls .allowMoreInputAddBtnCls span:nth-child(2){
		margin: 0px 3px;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputParamsCls{
		display: flex;
		flex-wrap: wrap;
		width: 100%;
		margin-top: 10px;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls{
		height: 26px;
		display: flex;
		align-items: center;
		border: 1px solid #DFE2EE;
		border-radius: 4px;
		box-sizing: border-box;
		padding: 0px 10px;
		margin-right: 10px;
		margin-bottom: 5px;
		background: #F8F8FD;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(1){
		display: inline-block;
		max-width: 520px;
		overflow: hidden;
		text-overflow: ellipsis;
		height: 100%;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls span:nth-child(2){
		margin-left: 10px;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputParamsItemCls .el-icon-close::before{
		font-size: 12px;
		color: #7A7992;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputFootCls{
		height: 18px;
	}
	#gnbSettingPage .allowMoreInputBoxCls .allowMoreInputFootCls .inputErrorBoxCls{
		color:red;
		font-size:10px;
	}
</style>
<div id="gnbSettingPage" style='display:flex;flex-direction:row;flex:1;height:100%;overflow:hidden;'>
	<div class="headcloseBtn" style="top:6px;right:10px;" @click="settingPageClose">
		<span class="el-icon el-icon-close" style='position: unset; display: block; text-align: center;'></span>
	</div>
	<div class="enbNav">
		<div class="navItem" :class="currentItem === 'info' ? 'active':''"  @click="changeMain('info')">
			<span class="el-icon el-icon-overview"></span><%=rb.getString("ZongLan")%>
		</div>
		<div class="navItem" :class="currentItem === 'chart' ? 'active':''" @click="changeMain('chart')">
			<span class="el-icon el-icon-operation-statistics"></span><%=rb.getString("TongJi")%>
		</div>
		<div v-show="isWritableComputed('CODE_ALARM_VIEW','isReadOrWritable')" class="navItem" :class="currentItem === 'alarm' ? 'active':''" @click="changeMain('alarm')">
			<span class="el-icon el-icon-menu-alarm"></span><%=rb.getString("GaoJing")%>
		</div>
		<div v-if="productType != 'BaiBNQ'" class="navItem" :class="currentItem === 'topo' ? 'active':''" @click="changeMain('topo')">
			<span class="el-icon el-icon-operation-topo"></span>TOPO
		</div>
		
		<div class="navItemGroup advanceGroup" v-show="isWritableComputed('CODE_GNB_SETTINGS','isReadOrWritable')">
			<div class="navItemTitle"> 
				<span class='el-icon el-icon-operation-settings' style='color: rgba(0,0,0,0.8); cursor: text;'></span>
				<span class='commonNotes14'><%=rb.getString("SheZhi")%></span>
			</div>
			<div class="navItem" :class="{'active': currentItem === item.code, 'disabled': item.disabled == true}" v-for="item in settingMenus"  @click="changeSettingMain(item.code,item)" style='padding-left: 44px;'>
				{{item.label}}
			</div>
		</div>
		<div v-show="isWritableComputed('CODE_GNB_UPGRADE_IMAGE','isReadOrWritable')" class="navItem" :class="currentItem === 'upgrade' ? 'active':''" @click="changeMain('upgrade')">
			<span class="el-icon el-icon-menu-upgrade"></span><%=rb.getString("ShengJi")%>
		</div>
		<div v-show="isWritableComputed('CODE_GNB_BACKUP_RESTORE','isWritable')" class="navItem" :class="currentItem === 'backup' ? 'active':''" @click="changeMain('backup')">
            <span class="el-icon el-icon-circle-backup"></span><%=rb.getString("beiFenYuHuiFu")%>
        </div>
		<div v-show="isWritableComputed('CODE_GNB_LOGS','isReadOrWritable')" class="navItem" :class="currentItem === 'log' ? 'active':''" @click="changeMain('log')">
			<span class="el-icon el-icon-operation-details"></span><%=rb.getString("RiZhi")%>
		</div>
		<div v-show="isWritableComputed('CODE_GNB_LICENSE','isWritable')" class="navItemGroup advanceGroup">
			<div class="navItemTitle"> 
				<span class='el-icon el-icon-operation-accesscontrol' style='color: rgba(0,0,0,0.8); cursor: text;'></span>
				<span class='commonNotes14'><%=rb.getString("GaoJiSheZhi")%></span>
			</div>
			<div class="navItem" :class="currentItem === 'license' ? 'active':''" @click="changeMain('license')" style='padding-left: 44px;'>
				<span><%=rb.getString("License")%></span>
				<span class="el-icon el-icon-sas-warning" style='margin-left: 10px; color: #000000;' v-if='false'></span>
			</div>			
		</div>
	</div>
	
	<div class="settingMainPage">
		<div class='commonFlex rightBoxHeader'>
			<div style='width: 92%; display:inline-block;overflow:hidden;text-overflow:ellipsis;word-break:break-all;white-space: nowrap;'><span class='commonText14'>{{title}}</span><span class='tableText12'> (SN: {{rowData.serial_number}}, <%=rb.getString("HostName")%>: {{rowData.host_name}})</span></div>
		</div>
		<div class="el-icon-common-refresh el-icon" v-if="currentItem == 'license' && isWritableComputed('CODE_GNB_LICENSE','isWritable')" @click='gnbRefreshLicenseTab' style="position:absolute;right:50px;top: 10px;font-size:20px;"></div>
		<div id="gnbSetting_main" class="el-card__body" style="position: relative;"></div>		
	</div>
	
</div>

<script>
if(window.gnbTabSettingVue) {
	try {
		window.gnbTabSettingVue.$destroy();
	}catch(e){}
}
window.gnbTabSettingVue = new Vue({
	el:'#gnbSettingPage',
	data(){
		return{
			title:'<%=rb.getString("ZongLan")%>',
			delay_avaliable:'',
			currentItem:'info',
			rowData:[],
			settingMenus:[],
			sasEnableStatus:false,
			productType:'',
			pageSource:'', // 页面来源标识
            ranConfigParams:{
                resetParams: {
                    'LTENF':{
                        LTENF_CarrierFreq:'',
                        LTENF_AllowdMeasBandWidth:'',
                        LTENF_PresAntennaPort1:'',
                        LTENF_QOffset:'',
                        LTENF_WideBandRsrqMeas:'',
                        LTENF_CellReselectionPriority:'',
                        LTENF_ThreshXHigh:'',
                        LTENF_ThreshXLow:'',
                        LTENF_QRxLevMin:'',
                        LTENF_QQualMin:'',
                        LTENF_PMaxEUTRA:'',
                    },
					'LTEER':{
						LTEER_EUTRACarrierARFCN:'',
						LTEER_TReselectionEUTRA:'1',
						LTEER_CellReselectionPriority:'7',
						LTEER_ThreshXHigh:'',
						LTEER_ThreshXLow:'',
						LTEER_QRxLevMin:'',
						LTEER_QQualMin:'',
						LTEER_PMaxEUTRA:'',
						LTEER_ThreshXHighQ:'',
						LTEER_AllowdMeasBandWidth:'mbw50',
						LTEER_PresAntennaPort1:'0',
						LTEER_BlackPhysCellIdStart:'',
						LTEER_BlackPhysCellIdRange:'0',
					},
					'LTENC':{
						LTENC_PLMNID:'',
						LTENC_CID:'',
						LTENC_EUTRACarrierARFCN:'',
						LTENC_PhyCellID:'',
						LTENC_QOffset:'',
						LTENC_QRxLevMinOffsetCell:'',
						LTENC_QQualMinOffsetCell:'',
						LTENC_CIO:'',
						LTENC_Blacklisted:'',
						LTENC_TAC:'',
						LTENC_eNBType:'',
						LTENC_eNBID:'',
						LTENC_noRemove:'',
					},
					'NRNF':{
						NRNF_Enable:'false',
						NRNF_SSBFrequency:'',
						NRNF_SubCarrierSpacing:'',
						NRNF_SmtcPeriodicity:'',
						NRNF_SmtcOffset:'',
						NRNF_SmtcDuration:'',
						NRNF_SSBlocksConsolidationRsrp:'',
						NRNF_SSBlocksConsolidationRsrq:'',
						NRNF_SSBlocksConsolidationSinr:'',
						NRNF_NrofSSBlocksToAverage:'',
						NRNF_RsrpOffsetSSB:'',
						NRNF_RsrqOffsetSSB:'',
						NRNF_SinrOffsetSSB:'',
						NRNF_RsrpOffsetCsiRs:'',
						NRNF_RsrqOffsetCsiRs:'',
						NRNF_SinrOffsetCsiRs:'',
						NRNF_BitmapType:'',
						NRNF_Bitmap:'',
						NRNF_DeriveSSBIndexFromCell:'',
						NRNF_FreqBandIndicatorNR:'',
						NRNF_OffsetToPointA:'',
						NRNF_SSBSubCarrierOffset:'',
					},
					'NRIRS':{
						NRIRS_CarrierFreq:'',
						NRIRS_NrofSSBlocksToAverage:'',
						NRIRS_ThresholdRSRP:'',
						NRIRS_ThresholdRSRQ:'',
						NRIRS_ThresholdSINR:'',
						NRIRS_SubCarrierSpacing:'1',
						NRIRS_DeriveSSBIndexFromCell:'0',
						NRIRS_QRxLevMin:'',
						NRIRS_QQualMin:'',
						NRIRS_PMax:'',
						NRIRS_TReselectionNR:'0',
						NRIRS_ThreshXHighP:'',
						NRIRS_ThreshXLowP:'',
						NRIRS_ThreshXHighQ:'',
						NRIRS_ThreshXLowQ:'',
						NRIRS_CellReselectionPriority:'7',
						NRIRS_CellReselectionSubPriority:'0',
						NRIRS_QOffsetFreq:'',
						NRIRS_BlackPhyCellIdStart:'',
						NRIRS_BlackPhyCellIdRange:'0',
					},
					'NRNC':{
						NRNC_PLMNID:'',
						NRNC_CID:'',
						NRNC_NRCarrierARFCN:'',
						NRNC_SSBFrequency:'',
						NRNC_SSBSubcarrierSpacing:'',
						NRNC_PhyCellID:'',
						NRNC_QOffset:'',
						NRNC_QRxLevMinOffsetCell:'',
						NRNC_QQualMinOffsetCell:'',
						NRNC_CIO:'',
						NRNC_Blacklisted:'',
						NRNC_TAC:'',
						NRNC_noRemove:'',
						NRNC_gnbIdLength:''
					},
					'QOS':{
						QOS_Enable:'1',
						QOS_MappingDrbIndex:'',
						QOS_5QI:'',
						QOS_Type:'0',
						QOS_Priority:'',
						QOS_MinBr:'',
						QOS_IsDefault:'0',
						QOS_UeInactivityTimerConfig:'',
						QOS_TReorderingPdcp:'',
						QOS_TReorderingUE:'',
						QOS_DiscardTimer:'0',
						QOS_StatusReportRequired:'1',
						QOS_PdcpSnSizeUL:'1',
						QOS_PdcpSnSizeDL:'1',
						QOS_Dscp:'',
						QOS_RlcMode:'2',
						QOS_SnFieldLengthAmDL:'',
						QOS_SnFieldLengthAmUL:'',
						QOS_SnFieldLengthUmDL:'',
						QOS_SnFieldLengthUmUL:'',
						QOS_ULConfig:'1',
						QOS_EnableRohc:'0',
						QOS_RohcProfile0x0001:'0',
						QOS_RohcProfile0x0002:'0',
						QOS_RohcProfile0x0006:'0',
						QOS_PdcpDuplicationActivated:'0',
						QOS_PrimaryPathDL:'0',
						QOS_PrimaryPath:'0',
						QOS_ULDataSplitThreshold:'0',
						QOS_DLDataSplitThreshold:'0',
						QOS_AllowedIntegrityAlgo:'0',
						QOS_LongDrxCycle:'0',
						QOS_ShortDrxCycle:'5',
						QOS_ShortDrxCycleTimer:'4',
						QOS_DrBlnactivityTimerConfig:''
					},
					'SST':{
						SST_Sst:'',
						SST_SstResourceType:'0',
						SST_MaxResourceReserved:'',
						SST_MinResourceReserved:'',
					},
					'Xn':{
						Xn_PLMNID:'',
						Xn_RemoteAddress:'',
						Xn_LinkEnable:'0',
						Xn_HoEnable:'0',
					},
					'DLBWP':{
						DLBWP_DIBwpID:'0',
						DLBWP_StartPrbPosition:'',
						DLBWP_BandWidth:'5',
						DLBWP_SubcarrierSpacing:'0',
						DLBWP_CyclicPrefix:'normal',
						DLBWP_PMaxULPower:'',
						DLBWP_InitDlMcs:'',
						DLBWP_MaxDlUeToBeScheduleInSlot:'',
					},
					'A1':{
						A1_Enable:'1',
						A1_ThresholdTriggerType:'0',
						A1_ThresholdRSRP:'',
						A1_ThresholdRSRQ:'',
						A1_ThresholdSINR:'',
						A1_ReportOnLeave:'1',
						A1_Hysteresis:'',
						A1_MaxReportCells:'',
						A1_MeasurePurpose:'1',
						A1_ReportAmount:'4',
						A1_MaxNrofRSIndexToReport:'',
						A1_ReportInterval:'5120',
						A1_ReportQuantity:['rsrp','rsrq','sinr'],
						A1_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
						A1_TimeToTrigger:'480',
						A1_RsType:'ssb',
						A1_IncludeBeamMeasurements:'0',
						A1_PLMN:'',
					},
					'A2':{
						A2_Enable:'1',
						A2_ThresholdTriggerType:'0',
						A2_ThresholdRSRP:'',
						A2_ThresholdRSRQ:'',
						A2_ThresholdSINR:'',
						A2_ReportOnLeave:'1',
						A2_Hysteresis:'',
						A2_MaxReportCells:'',
						A2_MeasurePurpose:'1',
						A2_ReportAmount:'4',
						A2_MaxNrofRSIndexToReport:'',
						A2_ReportInterval:'5120',
						A2_ReportQuantity:['rsrp','rsrq','sinr'],
						A2_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
						A2_TimeToTrigger:'480',
						A2_RsType:'ssb',
						A2_IncludeBeamMeasurements:'0',
						A2_PLMN:'',
					},
					'A3':{
						A3_Enable:'1',
						A3_ThresholdTriggerType:'0',
						A3_OffsetRSRP:'',
						A3_OffsetRSRQ:'',
						A3_OffsetSINR:'',
						A3_ReportOnLeave:'1',
						A3_Hysteresis:'',
						A3_MaxReportCells:'',
						A3_MeasurePurpose:'1',
						A3_ReportAmount:'4',
						A3_MaxNrofRSIndexToReport:'',
						A3_ReportInterval:'5120',
						A3_ReportQuantity:['rsrp','rsrq','sinr'],
						A3_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
						A3_TimeToTrigger:'480',
						A3_UseWhiteCellList:'0',
						A3_RsType:'ssb',
						A3_IncludeBeamMeasurements:'0',
						A3_PLMN:'',
					},
					'A4':{
						A4_Enable:'1',
						A4_ThresholdTriggerType:'0',
						A4_ThresholdRSRP:'',
						A4_ThresholdRSRQ:'',
						A4_ThresholdSINR:'',
						A4_ReportOnLeave:'1',
						A4_Hysteresis:'',
						A4_MaxReportCells:'',
						A4_MeasurePurpose:'1',
						A4_ReportAmount:'4',
						A4_MaxNrofRSIndexToReport:'',
						A4_ReportInterval:'5120',
						A4_ReportQuantity:['rsrp','rsrq','sinr'],
						A4_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
						A4_TimeToTrigger:'480',
						A4_UseWhiteCellList:'0',
						A4_RsType:'ssb',
						A4_IncludeBeamMeasurements:'0',
						A4_PLMN:'',
					},
					'A5':{
						A5_Enable:'1',
						A5_ThresholdTriggerType:'0',
						A5_Threshold1RSRP:'',
						A5_Threshold1RSRQ:'',
						A5_Threshold1SINR:'',
						A5_Threshold2TriggerType:'0',
						A5_Threshold2RSRP:'',
						A5_Threshold2RSRQ:'',
						A5_Threshold2SINR:'',
						A5_ReportOnLeave:'1',
						A5_Hysteresis:'',
						A5_MaxReportCells:'',
						A5_MeasurePurpose:'1',
						A5_ReportAmount:'4',
						A5_MaxNrofRSIndexToReport:'',
						A5_ReportInterval:'5120',
						A5_ReportQuantity:['rsrp','rsrq','sinr'],
						A5_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
						A5_TimeToTrigger:'480',
						A5_UseWhiteCellList:'0',
						A5_RsType:'ssb',
						A5_IncludeBeamMeasurements:'0',
						A5_PLMN:'',
					},
					'B1':{
						B1_Enable:'1',
						B1_ThresholdTriggerType:'0',
						B1_Threshold1EUTRARSRP:'',
						B1_Threshold1EUTRARSRQ:'',
						B1_Threshold1EUTRASINR:'',
						B1_Hysteresis:'',
						B1_MaxReportCells:'',
						B1_MeasurePurpose:'1',
						B1_ReportAmount:'4',
						B1_ReportInterval:'5120',
						B1_TimeToTrigger:'480',
						B1_ReportQuantity:['rsrp','rsrq','sinr'],
						B1_ReportOnLeave:'1',
						B1_PLMN:'',
					},
					'B2':{
						B2_Enable:'1',
						B2_ThresholdTriggerType:'0',
						B2_Threshold1RSRP:'',
						B2_Threshold1RSRQ:'',
						B2_Threshold1SINR:'',
						B2_Threshold2TriggerType:'0',
						B2_Threshold2EUTRARSRP:'',
						B2_Threshold2EUTRARSRQ:'',
						B2_Threshold2EUTRASINR:'',
						B2_Hysteresis:'',
						B2_MaxReportCells:'',
						B2_MeasurePurpose:'1',
						B2_ReportAmount:'4',
						B2_ReportInterval:'5120',
						B2_TimeToTrigger:'480',
						B2_ReportQuantity:['rsrp','rsrq','sinr'],
						B2_ReportOnLeave:'1',
						B2_PLMN:'',
					},
                    'PeriodMeasure':{
                        PeriodMeasure_ReportQuantity:['rsrp','rsrq','sinr'],
                        PeriodMeasure_MaxReportCells:'',
                        PeriodMeasure_MeasurePurpose:'1',
                        PeriodMeasure_ReportInterval:'5120',
                        PeriodMeasure_ReportAmount:'4',
			        },
				},
                casts:{
                    'B7EF0230C260BFB555FAE99A93662E28':'AMFList',
                    'D2EE312531CA9ED69741523F9CA5B151':'AMF_idx',
                    '3CE9BF9752F667F6AF50EC00A823E72C':'AMF_IP',
                    'F4B3ED9972A6303F1345DC27971AAE8B':'AMF_PLMNID',
                    '0E6C432B61CD906596E8CC4B13445F7B':'AMF_Default',

                    '2DBF5C7547B020B90FF3A1627FEBD64A':'gnbLength',
                    'B5E7A4393327B6AA1CD7E96B1555422E':'gnbName',
                    '97EFB7C38259BD3EA55AA07AAEC128B5':'gnbId',
                    '63518B64B478B3CC8628FBB209962F21':'adminState',

                    'F3DBE19874A7FCE01A803139AB861D9D':'SSB_SSBAbsoluteFreq',

                    'A1FD39B9572563316780ABE5141231F7':'Mobility_NrToLteMigrateStgy',

                    'ABF99FAD231EC0B203630EA0B5E87CED':'NRCellList',
                    'AB6D58847119715A33B8700151ACD46D':'NRCell_idx',
                    '48CEA6F90FAA75FF3F028C23E76ABCF1':'NRCellIdentity',
                    '132A9910A32BC62AC9E4072C899939DD':'NRCellTAC',
                    '1DD821C2FAA19DBCA04DFA5A26DE5124':'NRCellRanac',
                    'BDD6FE0619FF1D42B7CBDDDB36142252':'multiPlmnEnable',

                    '3C346AF3107FE32BDA36E98B5C9DF660':'PlmnList',
                    '7FB9ED30F77D505D13BF4A484C41D258':'Plmn_idx',
                    '39A3588E3437717AF75CE352615D3344':'PlmnId',
                    'BB7764523CC64492F8E497872F68A594':'Primary',

                    '6670998F724BA7A44582FC9AA5CAD386':'SliceList',
                    'F826ABDD4772DD482CBA0AF2C6D7CF89':'Slice_idx',
                    'E0341A776CDB1E438CEC9E5A38ED15B6':'SNSSAI',
                    'C5D84783D9979D13E8ACC786C6CBC203':'NguIp',

                    '124135E92E936D0EC02DAD1A3CCFC5B9':'band',
                    '7D03B239DAD1E408546209E5058496D0':'NRARFCNDL',
                    '7C6E8C293ACCDBB6D3541B4BDF634978':'NRARFCNUL',
                    'FA5D1ED332D8EA182932B25905D81564':'Pci',
                    'CC2F14DE22FE2FBDC651ED811A1B48A9':'NumOfTxAtenna',
                    '48415C2509759C6EDA5556B171FEDBB9':'NumOfRxAtenna',
                    'B2FB068E79289427A66C1D5810114114':'rftxEnable',
                    '899A5DD3F1B55A955AC62CA0A410908F':'DL_SubCarrierSpacing',
                    '5D835E9298BA427BCE1DBBBADDC7DF55':'DL_CarrierBandWidth',
                    '6B32A58D064AD5AE320FC06D89BA5BF7':'UL_SubCarrierSpacing',
                    '3D3A496D256BB709DDEA524ACE8777E0':'UL_CarrierBandWidth',
                    '43AA98BC4416A76A3B9C992B1D7219A9':'PowerModify',
                    '69D1D27FF521A31E7F70291A6880EF40':'PowerMaxLimit',
                    '27A765CCC3222333ECEF0616EC654596':'PowerMinLimit',
                    '94097AD8F964AEB3657929C154C0D021':'offsetToPointA',
                    'CCE04B7F18E45A81340E8A82DE5229B0':'ssbSubCarrierOffset',
                    // 'BDD6FE0619FF1D42B7CBDDDB36142252':'ssbGSCN',

                    '071F2091F5643F2FD8D2F74A06AC67D9':'LTENFList',
                    '0B7E09E27B31FCFE50D50FB4647D412F':'LTENF_idx',
                    '3F450C6676C2193BB41A4DE79C2106E4':'LTENF_CarrierFreq',
                    'B50D812FB0F214BAEBE06DB7053CD80E':'LTENF_AllowdMeasBandWidth',
                    '3A966E40800FB710C13A1D2F53E75A39':'LTENF_PresAntennaPort1',
                    'CB83C874ED855AC5F7C429AB32623E99':'LTENF_QOffset',
                    '414F01C612A42E9C38227FB7C14DCC35':'LTENF_WideBandRsrqMeas',
                    '0A9A000D3C0A3911E287A96806E3EFCC':'LTENF_CellReselectionPriority',
                    '907871D07DC5D23DD3509482E3D7CA71':'LTENF_ThreshXHigh',
                    '197B0588AEE370A01C5463607EE0B5D8':'LTENF_ThreshXLow',
                    'A839DDBC5F0761D74816D9F40E2D7C66':'LTENF_QRxLevMin',
                    '33780BB7F8668A36B64AC0F306BDB6E2':'LTENF_QQualMin',
                    'EBBAC1EB0DD126528C79B15B6C71A581':'LTENF_PMaxEUTRA',

                    '0A5AC6B6A3B1AF2550332F42BF830115':'LTEERList',
                    'B6DC27B502298D5533C8758DA94A18F1':'LTEER_idx',
                    '123BDB10F9021DBC10D6F715D72E7AA3':'LTEER_EUTRACarrierARFCN',
                    '011AEDC05E4DDB247F736DE265A40B5E':'LTEER_TReselectionEUTRA',
                    'E68EA3C9CF076A5F93C4AEC67496871D':'LTEER_CellReselectionPriority',
                    'A09104FF90470354D231FC381CEB2608':'LTEER_ThreshXHigh',
                    '7B744CC57A5EDACF75951AC491EBBD89':'LTEER_ThreshXLow',
                    'C61D14243949DA8CA7D050C73308DB94':'LTEER_QRxLevMin',
                    'D6E6B4DFAC99BAE5E95F18AF4AF7FA03':'LTEER_QQualMin',
                    '19C7A9161FA418A98F98F421155040D5':'LTEER_PMaxEUTRA',
                    '979062B21163F45E27E8084A9EFFB95C':'LTEER_ThreshXHighQ',
                    '2EB1CFC9F72B0D2FC146E937BAE3A4A8':'LTEER_AllowdMeasBandWidth',
                    '1796E06A7FFF53DEB4798A5101ECAD53':'LTEER_PresAntennaPort1',
                    'B0C93852549D6C55EBDB58A4D583F5E0':'LTEER_BlackPhysCellIdStart',
                    '69D97BBAC7A6EEF0E72E4652A2035092':'LTEER_BlackPhysCellIdRange',

                    '7B7B254A37836B50929913604CBF0B95':'LTENCList',
                    '95DCEE4232482B822AF61B95E11BB328':'LTENC_idx',
                    '3F56A1DA72243C633F5ABEFB054A65DC':'LTENC_PLMNID',
                    '0247AE466667157050B9FE7A6CF65B16':'LTENC_CID',
                    '7300F4B5706BDFB5D541F721389F2658':'LTENC_EUTRACarrierARFCN',
                    '688790C6447761EA49977C0D34FE8B61':'LTENC_PhyCellID',
                    '94249F79ECD5718D8B970D4C5D794E90':'LTENC_QOffset',
                    'D05E321D1D15992C3E12A700F53B9DAC':'LTENC_QRxLevMinOffsetCell',
                    '081827660A86565F9B66EEFC794B4766':'LTENC_QQualMinOffsetCell',
                    '578A93F7F7A48A4A2CAD4BFA2648F445':'LTENC_CIO',
                    'D7EBC619CA34CF28D968A1FCC17AC517':'LTENC_Blacklisted',
                    '5B49AC1E07753DCBAA55EE98E971F38E':'LTENC_TAC',
                    '11734AEA5EE16660C09513EAAD35761B':'LTENC_eNBType',
                    '33D773E7E16DF509D353C6D772336F32':'LTENC_eNBID',
                    '9D2C2F1F2B60F50149958E3A065F1783':'LTENC_noRemove',

                    'FFBD6091E2BBDBF1E2ECFA13A4673E00':'NRNFList',
                    '6C3FE12704E731C385A9508B23318689':'NRNF_idx',
                    'CB9B918DB2FD80F2F3FB22B46FFB7052':'NRNF_Enable',
                    '086F3A4784A0E964C54819F109785AC5':'NRNF_SSBFrequency',
                    'C261FD0E07702C2F1D969303A47C3110':'NRNF_SubCarrierSpacing',
                    'F847642D0EF9CB6E15961D8C21200B10':'NRNF_SmtcPeriodicity',
                    '69727C2AA02C6042793A5114CD81E666':'NRNF_SmtcOffset',
                    'F8F28DDF2FBA8F775569AD611914666A':'NRNF_SmtcDuration',
                    '3CB8EEA23EA29890788451EE4BFE76B6':'NRNF_SSBlocksConsolidationRsrp',
                    'CA7E72D1393D24811E1A14FFD4098C11':'NRNF_SSBlocksConsolidationRsrq',
                    '3FE378CFFE6E4A0167928F3AEA591161':'NRNF_SSBlocksConsolidationSinr',
                    'E97EBDF25595A3030C66CC36D0D62FB9':'NRNF_NrofSSBlocksToAverage',
                    '1C93CE499C64AAC73E125E3387EAF7CB':'NRNF_RsrpOffsetSSB',
                    '2BF88B8AC753A9801DC6176B30BA715B':'NRNF_RsrqOffsetSSB',
                    '38F9862E2A58B9D1EBE9F460B76117FA':'NRNF_SinrOffsetSSB',
                    'BC1CFB2CF45E7FD6CD56220E898D27AB':'NRNF_RsrpOffsetCsiRs',
                    'AD1B752015A69BF98ECB628247E2B714':'NRNF_RsrqOffsetCsiRs',
                    '407BC693EFC35DDE21709F7DE9CFF423':'NRNF_SinrOffsetCsiRs',
                    '0DFBE5074C9E8372DC5DC7EB6935855B':'NRNF_BitmapType',
                    '266976D0E88FC340898915D7B9EB5C47':'NRNF_Bitmap',
                    'DC01FE89B78A1A1F2AE95CF943652904':'NRNF_DeriveSSBIndexFromCell',
                    '5C92E58894F45CEA20B891C2EC54DEBA':'NRNF_FreqBandIndicatorNR',
                    'CA6AB0188BA7B9853E3486496D7D6EFE':'NRNF_OffsetToPointA',
                    'BCFAD98AB5F77886F429D183CEBC9BB2':'NRNF_SSBSubCarrierOffset',
                    
                    '5B0857DDB9E0C9830A4956B59F89BF33':'NRIRSList',
                    '97C4AD27B9E8C45F1AFD5F8373B3F3D0':'NRIRS_idx',
                    '37361244120F92C3846AAF1F4315B20F':'NRIRS_CarrierFreq',
                    '4B8F8686EBF7415DA48963D31F732A10':'NRIRS_NrofSSBlocksToAverage',
                    '5CEA396BAC7EB70167C6C382169520A0':'NRIRS_ThresholdRSRP',
                    '429A4DD14537B07EB92B1E099055CB62':'NRIRS_ThresholdRSRQ',
                    'CAE9274084CB07F2046912F47B843297':'NRIRS_ThresholdSINR',
                    '70CA740B565A5A96CC83EE3D804DD927':'NRIRS_SubCarrierSpacing',
                    '0CEB6717E9E3D7370770085F77DA9476':'NRIRS_DeriveSSBIndexFromCell',
                    '325AA38CED2F92233EA154718ACFE06F':'NRIRS_QRxLevMin',
                    '89271BE6717320E4DABFF94454029378':'NRIRS_QQualMin',
                    'B07FF00FC97A09868612D4B2F8FD2BEB':'NRIRS_PMax',
                    '9E901FAD1F472CAAD2D823C0A479EE88':'NRIRS_TReselectionNR',
                    '17AAD37D8BF013A71CF88C9071A40C2C':'NRIRS_ThreshXHighP',
                    '2CB0ADDE0010723CEC23FAE3E0DFE787':'NRIRS_ThreshXLowP',
                    'EDCB820C823DD4AF1E8EACF1E291E2F2':'NRIRS_ThreshXHighQ',
                    '7F0792B1EAA1727F7B6279246A34B4A5':'NRIRS_ThreshXLowQ',
                    '2DE59538980120FD7F546F6BB2D28D59':'NRIRS_CellReselectionPriority',
                    'B5B49701BA4AF6598E051C965337032A':'NRIRS_CellReselectionSubPriority',
                    '1F5DE40C8933BF46B46F3D9F0D481C6D':'NRIRS_QOffsetFreq',
                    '6AA524F6C64031318A692DDB60D08446':'NRIRS_BlackPhyCellIdStart',
                    'BED7E6C38273182777E576A50A81079F':'NRIRS_BlackPhyCellIdRange',
                    
                    '3993E07C199709BD90BB59C197954ED8':'NRNCList',
                    '6F6158A4AC9A3DAADA272ACEE8965A3E':'NRNC_idx',
                    '0ABC001C934D18B11F43B55DEDCAEF88':'NRNC_PLMNID',
                    'AE87CB0E7C6101FFFAACFB7C728A1AD2':'NRNC_CID',
                    '83DAD05DB20F45E9800B6090512CFC21':'NRNC_NRCarrierARFCN',
                    'F5359EE38287AF5BE2C6B0F24DCBC07B':'NRNC_SSBFrequency',
                    '5715660FCF14D83B1DFEB3AA7291B47E':'NRNC_SSBSubcarrierSpacing',
                    '6B9F12A98D32686CBFC6233991642CEE':'NRNC_PhyCellID',
                    'AA82E5E83A37BC6A6076C075AD5A9EFA':'NRNC_QOffset',
                    '85F4FB44757C5FCC51E6682225C07843':'NRNC_QRxLevMinOffsetCell',
                    '6B1D9444974757EF55B0ABEDD864E3B4':'NRNC_QQualMinOffsetCell',
                    'C354349B6BF8E3111E576B1A22EB7DD1':'NRNC_CIO',
                    '405ABE0B20840AA4787060C6DDFADC25':'NRNC_Blacklisted',
                    '865122707C9C7FE063CC47BA3D779C6E':'NRNC_TAC',
                    'AAD4C0190EDBD517FA29DD625EDEA0AD':'NRNC_noRemove',
                    '3C35A83E6573D575DAC7164154B131CD':'NRNC_gnbIdLength',

                    'A143C4AE4BE1E73F1BC109ED79ABBA23':'QOSList',
                    '2A9270EF3D0184F0675C9BF232B6A01B':'QOS_idx',
                    '5E185B659DAC5BCE84F6E9794E96F75A':'QOS_Enable',
                    '7876B1F3D6B0AC6FD918674D323B0377':'QOS_MappingDrbIndex',
                    'AA3AB1D285160953C65CBD04C2732B9E':'QOS_5QI',
                    'A29483B03CBE80C13DB685026D6F2C2C':'QOS_Type',
                    '2FF2F3185C76E0ABDB9E08A9561A692B':'QOS_Priority',
                    'E5EC9A9D00AD513BB600B988EE4B5D0F':'QOS_MinBr',
                    'D2F83EA3D5491980F9E8070CE3C7401C':'QOS_IsDefault',
                    '787B534BB151122F48D77F40F767581E':'QOS_UeInactivityTimerConfig',
                    'AECE24995B865C5C28C81B4F59253156':'QOS_TReorderingPdcp',
                    '98710A2A6F57E69527FF7AEFF60F4283':'QOS_TReorderingUE',
                    '89556AB93661247FC7FD580DC5AC4EB0':'QOS_DiscardTimer',
                    '1626B6002F3EB537ED931A94A97AC205':'QOS_StatusReportRequired',
                    '7A9C954612EDF9EA7848B0CCCB0DA89B':'QOS_PdcpSnSizeUL',
                    'CB5FB6B13939DAEAAC9CA5170ECB86AB':'QOS_PdcpSnSizeDL',
                    'AC627FF47AB3DB3C2725A304E005679C':'QOS_Dscp',
                    'FA6C8EA67B675439211BFF176D960F1D':'QOS_RlcMode',
                    'E5696CE2108EAE7E309FA0614A65F668':'QOS_SnFieldLengthAmDL',
                    'AE430C5574B938E5B6932DE0C9D873DB':'QOS_SnFieldLengthAmUL',
                    'FCC231317BDD99FD05820D1CB095C4AB':'QOS_SnFieldLengthUmDL',
                    'A980D5A7FCD41617E412C375975741E4':'QOS_SnFieldLengthUmUL',
                    'FC57C81321A9B517AABE2337F0DD03F0':'QOS_ULConfig',
                    '6B7C0CA5C4B8F2DC4FE7499278711BAD':'QOS_EnableRohc',
                    'A583D5B6675B08DC0A2039870BA67B0F':'QOS_RohcProfile0x0001',
                    '48A625A5EBDC92DB9BFC07D23CD6D47A':'QOS_RohcProfile0x0002',
                    '4770A9A40D1357816ACC8DF527302408':'QOS_RohcProfile0x0006',
                    '422041169AF42C6AF368CF0262653D14':'QOS_PdcpDuplicationActivated',
                    '313CAB79508D1511BC2E11F181C2B306':'QOS_PrimaryPathDL',
                    'FF3C423061A2D1656FB240ED78A16093':'QOS_PrimaryPath',
                    '20681F326CFFABE13CBB3441FDF6575F':'QOS_ULDataSplitThreshold',
                    '079054E1ABD35B60F7B939DADFA9D0F7':'QOS_DLDataSplitThreshold',
                    '501983526E9595CEE244F904D72EE44F':'QOS_AllowedIntegrityAlgo',
                    '5DFD78E7351412DA88311A5FF8E2928F':'QOS_LongDrxCycle',
                    '43D147AD44A110FCB52FCBC1A2823671':'QOS_ShortDrxCycle',
                    '6C3B0F0C62B5A00C8CB0EBEFEACA5790':'QOS_ShortDrxCycleTimer',
                    'E0D3C56FD6A50B79543B01D563EC83AB':'QOS_DrBlnactivityTimerConfig',

                    '543527C1BDC7444C43C8D8A088FEE011':'SSTList',
                    '32CC003ACB4196A6C453DD27046347F7':'SST_idx',
                    '7F19EC5578B37DDB9B2885F03EC419DE':'SST_Sst',
                    '06AB0596C8DAE5ED2C82347A273C005D':'SST_SstResourceType',
                    '24976ED95F5B796FD33EA4E484FE35F9':'SST_MaxResourceReserved',
                    '00D2EBE8CF4C9D45FD1C110941566ED3':'SST_MinResourceReserved',

                    '0257A2E30FE31394FB90CDBEB83C2F17':'A1List',
                    'A5ED36FADE0E587E3E80734409563324':'A1_idx',
                    '3CCEA896049BA9C2BD95A08735EF04A7':'A1_Enable',
                    '93D2134C6405C20343D71B93B6FA7D86':'A1_ThresholdTriggerType',
                    '40C823B509F9684ED07AB031675B47C1':'A1_ThresholdRSRP',
                    '135EAE4ED0C586AF56DEBB182FBC3439':'A1_ThresholdRSRQ',
                    '8EF0FF7A91123C99361B13DC3D2315CD':'A1_ThresholdSINR',
                    '4CC9BF15ECDD864F56BDB2E60D77B14E':'A1_ReportOnLeave',
                    'A19D5803A437C434B34F41D4AEB2F5FB':'A1_Hysteresis',
                    '54D90F8FB5D1817443EB2880D75B16E6':'A1_MaxReportCells',
                    '3FC04BAA1616B18EF31120C83E25ED5A':'A1_MeasurePurpose',
                    '27DE29B81F3257F59FBAA44988070367':'A1_ReportAmount',
                    'DBD23EE15743974B1DD917AE72348B9D':'A1_MaxNrofRSIndexToReport',
                    '696CAEEE1C0E6902266E810C3958D5D5':'A1_ReportInterval',
                    '0D5D66749F86708852343879E1723EAF':'A1_ReportQuantity',
                    'F12F0C33CEF5B14B17D98E697B079CF0':'A1_RptQuantityRsIndex',
                    'D04EE74435679A2A1E00D661EA4E5C82':'A1_TimeToTrigger',
                    '8266EE6FB181D8FC212C37DB0CC82EAB':'A1_RsType',
                    'FC176B81A084A5BD0F7EB03BE8EB323B':'A1_IncludeBeamMeasurements',
                    'A56C999C00A87D7DFAA2D0B8839B7C0C':'A1_PLMN',
                    
                    'DAD13756A8F6A351E9C95C2BA9841018':'A2List',
                    'BC0C9645FF163EFFA3DCA2614CAE4835':'A2_idx',
                    'D231B536BA8978EC2B6FB39EA1A09DC8':'A2_Enable',
                    '982A6F7D5436307C3FCD06916F4E3591':'A2_ThresholdTriggerType',
                    '6377B3C3F13CDB349C05A0F13B12A36E':'A2_ThresholdRSRP',
                    'F3A7AFE11DC795D673886F47DA305BC3':'A2_ThresholdRSRQ',
                    'CA0F8150DBD5D073F5C5FBC7C2AA2A88':'A2_ThresholdSINR',
                    '6F59E6B01DC601A858E34E183880B517':'A2_ReportOnLeave',
                    '190DDE718A67000EB34DB6095484689E':'A2_Hysteresis',
                    'A1283090700799926927CD6DE5A26875':'A2_MaxReportCells',
                    '211CCF297C707B95147D9060D156B27C':'A2_MeasurePurpose',
                    '7624A24D2FED16417B490BE67D17D91E':'A2_ReportAmount',
                    '0049B8E72BD398C8CA1E6ABDE452BA10':'A2_MaxNrofRSIndexToReport',
                    '89094108BEB904CF45F01DC37536E913':'A2_ReportInterval',
                    'E018BF2A7223D9C043DCD5014494EC12':'A2_ReportQuantity',
                    'C7918CCE81B482117D7F61513F8A2522':'A2_RptQuantityRsIndex',
                    'EF7E25D685AEDD886C21A0F9130083C6':'A2_TimeToTrigger',
                    'EB5C4CA3F121BAA4B8E8BA89FE52BB4D':'A2_RsType',
                    '8B8EA0C9F19ED07F75B2CCD18824F16B':'A2_IncludeBeamMeasurements',
                    '894BBE731B793A43B4A54138E9DC9680':'A2_PLMN',

                    'E4EC0C6A60E94E17B4AA79A5F3ADF124':'A3List',
                    '0D12D9ACB7281501BE4D6FC542801CF6':'A3_idx',
                    '8E8D5B7479E315F473E8F0B214BEA3A8':'A3_Enable',
                    '578D8C691C44704E203E746C386DF488':'A3_ThresholdTriggerType',
                    'F737CFD1514AA1C1D018CEA328B7896D':'A3_OffsetRSRP',
                    'A7C9952311280CDA12A0433C246554A1':'A3_OffsetRSRQ',
                    '572A4F1C040468FDE63F6F8585535496':'A3_OffsetSINR',
                    '0C96DC0D0601E13EE49F87226047D945':'A3_ReportOnLeave',
                    '547835C743E6919180C5624D99588F33':'A3_Hysteresis',
                    '37BE566C624569B9D94598F9FDF4A2A8':'A3_MaxReportCells',
                    'D4C11DFC9FDC3D0BB41B7D2D5C0892C2':'A3_MeasurePurpose',
                    '11E5A2498249A097F624DB75AFE357FF':'A3_ReportAmount',
                    'CF920588674F8C836AD6FAF6902C47EC':'A3_MaxNrofRSIndexToReport',
                    'A4EC9A0F85B54B7E5093FE4FE13D9026':'A3_ReportInterval',
                    'BFE13F057E0B27A7470AB1DF88F617B7':'A3_ReportQuantity',
                    '6A7D9C6B7B280A6BC7BD70E8905F62F8':'A3_RptQuantityRsIndex',
                    'E999B7233855EB018D13DE831472B926':'A3_TimeToTrigger',
                    'F40688A2D474D1E0AE035FCDDEC2B285':'A3_UseWhiteCellList',
                    '1A5EFF5D0B2273A6709241B5045E2DFE':'A3_RsType',
                    '9B92A0B15D5191F70BB30AF049934909':'A3_IncludeBeamMeasurements',
                    'C22F204DBD72305D6319399F89BB56C5':'A3_PLMN',

                    '07EFC9D63D172ABDCF6E69C1BFD7D520':'A4List',
                    '322DB26B6062F1D6FAD6CBA9E2AD5388':'A4_idx',
                    '354C490F04134AA3A65C0C3DDC6E13C6':'A4_Enable',
                    '10EE2B1A52F73D6F0346680D3929F6EA':'A4_ThresholdTriggerType',
                    'A0FD9C3012796274F7A258FD04425851':'A4_ThresholdRSRP',
                    '26850F0484DC6B6FDE31BCF654287C1A':'A4_ThresholdRSRQ',
                    'CECFD54BCB431463E7554E0907DF2BC9':'A4_ThresholdSINR',
                    '61AEAE17A4C216E5F4C7F22790267035':'A4_ReportOnLeave',
                    '23E7CD35AE4C5AD0DD6EAC5E540257EB':'A4_Hysteresis',
                    '18FB11BEECCE0F351B0FEA57B0E78DEC':'A4_MaxReportCells',
                    '9AB14594834AAD54AA44A1DCBA2E6F3F':'A4_MeasurePurpose',
                    'E652A5D9E99CFF0D00FB97BC4ECD7B59':'A4_ReportAmount',
                    'A12210164EDC2E0F4723FBA045F5BED6':'A4_MaxNrofRSIndexToReport',
                    '8D7053237405C6E3B70B6B7C9FBBE79B':'A4_ReportInterval',
                    '03B51B69D952C448581600237866756D':'A4_ReportQuantity',
                    '51182F0ACA480F5E9C306D133BBC3575':'A4_RptQuantityRsIndex',
                    '267C83264429096CF5650DCA42D6A534':'A4_TimeToTrigger',
                    '71DE723BA89F947A472326367C8BA445':'A4_UseWhiteCellList',
                    'F58B35FB2B4AB7B30E6D84C98D9ECDD7':'A4_RsType',
                    '4627F5F37E4112EF9EA38E09993F1E53':'A4_IncludeBeamMeasurements',
                    '22979A6EC47CF9E6074D01F993CC70D6':'A4_PLMN',

                    'CA5818F33B06552A480C5AAF74AC3341':'A5List',
                    '91B898031E1C975B286FFD9953A93352':'A5_idx',
                    '98A5EDC451234FB5D94B6F5F64329D91':'A5_Enable',
                    'FCB2E18FDFB52A0A99BAAB69380C6B13':'A5_ThresholdTriggerType',
                    '1A09A6CDA4C45FC484D5A210422E9927':'A5_Threshold1RSRP',
                    '2FFA067A3C3C48865CEC9B720BC30258':'A5_Threshold1RSRQ',
                    'F4966FAA47168914887917FA1858B428':'A5_Threshold1SINR',
                    'DC42EBAD75770BAC14E06C84B2227128':'A5_Threshold2TriggerType',
                    '8D21DFEDE224571AE7B78FD293FF90AA':'A5_Threshold2RSRP',
                    '89A20E49D8293FED5A9ADF17FA9BD4A6':'A5_Threshold2RSRQ',
                    'E0351A91F0CC9B878B2A8F27C28ECB80':'A5_Threshold2SINR',
                    '51EA8E7F31D3DA88DDE0FDE7EC2AADA6':'A5_ReportOnLeave',
                    '7AE8312B8F61DE3333646B1A0DCC861F':'A5_Hysteresis',
                    'F57C70EFF093AD4BEEF282CAF14A7352':'A5_MaxReportCells',
                    'FEE002E99BABCCA5AB7AC516725E163C':'A5_MeasurePurpose',
                    'F759D13DCE418BA599C5418B9AF6FBF5':'A5_ReportAmount',
                    '7944D5E11FA09B86E8DE6082D374D90E':'A5_MaxNrofRSIndexToReport',
                    'C4476199A1F9A06BDFDD54181BEB4CC5':'A5_ReportInterval',
                    '5A92052C2E297A048B606444672AF899':'A5_ReportQuantity',
                    'A944C853A9AD1144357F07081996498A':'A5_RptQuantityRsIndex',
                    '1F6535E861DC7BC6E14BFC5F6C7F4E72':'A5_TimeToTrigger',
                    '4100BF47521CE5AB4E966DDEA3D46AB7':'A5_UseWhiteCellList',
                    '6F2AD603AF5FB18F43F9E5E66C88C26F':'A5_RsType',
                    'BDE2B18D8513F48DCDFC4721D24414F5':'A5_IncludeBeamMeasurements',
                    '8285AD15A8461CE9082425A255C0121A':'A5_PLMN',

                    'D0516B7BE65A206C181B247F2E9B5076':'B1List',
                    '0123AAA0D3A2B3F3BB5A333E76347D5E':'B1_idx',
                    'D4B73054274B6B3FD9C442520D359EB6':'B1_Enable',
                    '7EF988590D54ED29B675C1138496D33C':'B1_ThresholdTriggerType',
                    '92C7BDF8EA57DAD74DDDF612651CB8F5':'B1_Threshold1EUTRARSRP',
                    '83CC693174155D83F5C0BF3CD5764123':'B1_Threshold1EUTRARSRQ',
                    '760F0698FE785FBAB479A2CA36F1263D':'B1_Threshold1EUTRASINR',
                    '9DB8C6D158698E06BC1731B33353F566':'B1_Hysteresis',
                    '56158642295310EE21CAD3EE5F8D6F1F':'B1_MaxReportCells',
                    '500D9ACC5BC6CEEF69561099352FB147':'B1_MeasurePurpose',
                    '818558681E84BA4DFC46271CECDA8043':'B1_ReportAmount',
                    '7DB33AA49998428A66D7AEACF6837400':'B1_ReportInterval',
                    'FA72388C3E8DC75B6B26C7AE6ECBF0B4':'B1_TimeToTrigger',
                    '27F6ABB3F147849B6DE835D66D0A32F1':'B1_ReportQuantity',
                    'B4E202B361412C443C3B6736FC9D5923':'B1_ReportOnLeave',
                    'BEE19ACB2711445EEBD64AA90159E130':'B1_PLMN',

                    '384611BB0810231CCD430CEF91938D92':'B2List',
                    '0EDD5DB52CA22D5622D793FB08A34F0E':'B2_idx',
                    'A6E04F13D1F1485EAEC8B6122AF7843A':'B2_Enable',
                    'DAD1BB93F0DA7A245C67F13A77778F18':'B2_ThresholdTriggerType',
                    'D919432A801B47C9FB66DF9E9CFCE3CA':'B2_Threshold1RSRP',
                    '01B46CDFADB733F44A140DC9AE9496B0':'B2_Threshold1RSRQ',
                    'A9599821E863A154CA01E02CDD3C8820':'B2_Threshold1SINR',
                    '3D25B0741C721F6B844203DCA2846E93':'B2_Threshold2TriggerType',
                    '437301E99FE6DDE3228CBCDD2988DCB5':'B2_Threshold2EUTRARSRP',
                    'E955F99D83FE2A7899ADACD4377FD104':'B2_Threshold2EUTRARSRQ',
                    '4DC7AEBFDF8933B97D73FDF1B0365F1B':'B2_Threshold2EUTRASINR',
                    '81D7DB79375196FB5F0C2CE8CBBA6CE8':'B2_Hysteresis',
                    '141EDD14F7E6E937905AC146E14EB87C':'B2_MaxReportCells',
                    'A1B0DC7C3A1E05908A098723C0ACB625':'B2_MeasurePurpose',
                    '20775BA0B789B43CA6A62A513A6CA9C2':'B2_ReportAmount',
                    'B559D38BFDED262FB4BFC20B7A6B6A72':'B2_ReportInterval',
                    'F73D3D5001363A199274C4BC57462827':'B2_TimeToTrigger',
                    'ECC073692780CBDA1071A6F423798690':'B2_ReportQuantity',
                    '9292C3D57433EDE97B5C6C902C07B8A9':'B2_ReportOnLeave',
                    '85C5A254A9C9813C92630385717A7D78':'B2_PLMN',

                    'E53AC49C31855F2B749EDECD2629FDDC':'PeriodMeasureList',
                    '8A0A594A3E699099215DEFDA828782D6':'PeriodMeasure_idx',
                    '5DA600800C506AC90E45AFB50C3EF2C7':'PeriodMeasure_ReportQuantity',
                    '848AC40C3CA80DDFB0B6AFD0FA86FA05':'PeriodMeasure_MaxReportCells',
                    '2EF3BDDD080F516655E870FC939C58BA':'PeriodMeasure_MeasurePurpose',
                    '9FF0DCD50E70730F79D06F46F9494FDD':'PeriodMeasure_ReportInterval',
                    '7F130185D28105DAA71A1161675D53C5':'PeriodMeasure_ReportAmount',

                    '7DBE0DBC347F1E8BF822321802ADB925':'SIB1_QRxLevMinSIB1',
                    'EA997F8DD795917272B763C7E8FCCD76':'SIB1_QQualMinOffset',
                    'BE2B2C98701E99171123021735579D9C':'SIB1_QRxLevMinOffset',
                    '9EF4BEA01B5C3988E5D5593217A4B099':'SIB1_QQualMinSIB1',
                    '226D9876C6DBD35E54EBE9370BF1B6A9':'SIB2_Enable',
                    '30306F33135206A495E3687CFD93DB72':'SIB2_Qhyst',
                    '639B8088378EF2BC91EDD97437EA91F8':'SIB2_QRxLevMinSIB2',
                    '3CCE0D2DCCE81F33B812B6AA2A2CC0F8':'SIB2_SIntraSearchP',
                    'C21F2FC32E60F3D8CEC6481184E2C9E2':'SIB2_TReselectionNR',
                    'BF2457FB803A8DBAFFCBCFAFA1F04CED':'SIB2_CellReselectionPriority',
                    '1667DBF97FA9FB664F6BECB6D3C52BD1':'SIB2_ThreshServingLowP',
                    '855E620C591EBBFD81132A2A60B8513A':'SIB2_DeriveSSBIndexFromCell',
                    '9FC67ED33650216DF15EC752F6BC21FF':'SIB2_SNonIntraSearchP',
                    '4731359982289ECD01A44FE75FF18200':'SIB2_SNonIntraSearchQ',
                    '56B4FCDAE003FAB1BA4462DECF4181A0':'SIB3_Enable',
                    '614C58BA85456F33DBF82B9B50BF054F':'SIB4_Enable',
                    'C684D8C05C28A3BF34F0224479C4DDEB':'SIB5_Enable',
                    '95F44BB29953F20270F1E533DA193341':'SIB9_Enable',
                    'E6130F015C61EC26485D7141C2347D69':'SIB9_DaylightSavingTime',
                    '7FEE53FA2F3A9B3F412EE35BA521FFFE':'SIB9_LeapSecond',

                    '7950162C9A24A70DDDC3118E061E0A27':'XnList',
                    'F924527EF1DE6F5A3C1BCA790FADC9F5':'Xn_idx',
                    '2CC55133A6849300DA99C935B82C1752':'Xn_PLMNID',
                    'F7BCD29DC1DAC25C6112149537F1FDF9':'Xn_RemoteAddress',
                    'A61F75C2D2933B58903E3F1AB1B37363':'Xn_LinkEnable',
                    '7F2A16BAE7016B1FE8A0E6DC9C2C02FF':'Xn_HoEnable',
                    '58547675CFA0A71527B29D9A26438F22':'Xn_Status',

                    '4A05D255E8EC01783A1B6E700DDC7CE4':'XnBlacklist',
                    '3D0958ECEAE82263012001EC439348C3':'remoteAddress_idx',
                    'B9A7A2908C1C45DE4B691C83C6F00841':'remoteAddress',

                    '82B091C50808E3E6F5D33EFE2760F033':'ANR_Enable',
                    '4B7864F991DF752D3266D010055C4FD2':'ANR_InterFeqEnable',
                    'E02C0EC06DB0F23161D37F636CF051EB':'ANR_EUTRANEnable',
                    '93936E2D407EBE9B4D20FE4F36BA80EC':'ANR_BiNRCellEnable',
                    'FB2B5F5BBE059A2A934CFFFF57D7589D':'ANR_MRTriggerType',
                    '22C51B323E5AB7BD2DDCE44AE6A96574':'ANR_AbsoluteThreshold',
                    'E1F8C9EF3E7FEDCED5D2465489C8EC0C':'ANR_RelativeThreshold',
                    'EA1E967554408FA4417F6FC14586F8C1':'ANR_AbsEnable',
                    'FA5C715ECED420F841E508BE988D3978':'ANR_KpiPeriod',
                    'A6F5DE08C3490182866ED03AACDD9AC2':'ANR_AutoAdjustEnable',
                    '41F6A657567A148A560400C25B84CF8D':'ANR_AutoRemoveEnable',
                    'E3E128A2D938BCAA2370BCFDD7CF3F3B':'ANR_AutoRemovePeriod',
                    'D0012607E22676DFBF51D07336379E07':'ANR_AutoRemoveMaxCell',
                    '028E42BB378CD4C349A38955123D2512':'ANR_MaxHOtimes',
                    'C71A32B8C5605A916D59DD76AEB1EF69':'ANR_MaxHOSuccess',

                    'FBC0B717879876C8B21D4AEBBBA4FD2C':'DLBWPList',
                    '48D197CC3BC6E35A20FB1DF6F06A452B':'DLBWP_idx',
                    'AE92515F6B2F655ABB324B97540A25E2':'DLBWP_DIBwpID',
                    'A0D2143497AC31B54844169FB29D1436':'DLBWP_StartPrbPosition',
                    'E710DBA30692A88D4D4418021D1F78F0':'DLBWP_BandWidth',
                    '23B1544338AD7182F674A2B98ADF41D6':'DLBWP_SubcarrierSpacing',
                    '53DD5F6A3A12D222ED5CE57EAF1C70D4':'DLBWP_CyclicPrefix',
                    '7C3AFF393AFF010DBF985974E250CC2C':'DLBWP_PMaxULPower',
                    'C047650F5E19525217415BA683D366D4':'DLBWP_InitDlMcs',
                    '12A320A72CB61644D3A53CDAFDC5D4C8':'DLBWP_MaxDlUeToBeScheduleInSlot',
                },
            }
		}
	},
	computed: {
		isWritableComputed() {
			return (code,type) => { 
                if(type == 'isWritable'){
                    return writableMap[code] == true;
                }else if(type == 'isReadOrWritable'){
                    return writableMap[code] !== undefined;
                }
			}
		},
	},
	methods:{
		// 初始化
		initJumpInit(row, page, source){
			var vm = this,
				status = row.connection_status,
				settingMenus = [];
			
			vm.pageSource = source; // 保存页面来源
			vm.rowData = row;
			vm.productType = row.product ? row.product : '';
			if(page == 'alarm'){
				vm.changeMain(page);	
			}else{
				vm.changeMain('info');	
			}
			var settingMenus = []
			if(vm.rowData.product == 'BaiBNQ'){
				settingMenus = [
					{code:'quickSetting_GT',disabled:false,label:'<%=rb.getString("KuaiSuSheZhi")%>'},
					{code:'network',disabled:false,label:'<%=rb.getString("WangLuoSheZhi")%>'},
					{code:'coreNetwork',disabled:false,label:'<%=rb.getString("CoreNetwork")%>'},
					{code:'ran',disabled:false,label:'RAN'},
					{code:'bts',disabled:false,label:'BTS'},
					{code:'system',disabled:false,label:'System'}
				]
			}else{
				settingMenus = [
					{code:'quickSetting_X86',disabled:false,label:'<%=rb.getString("KuaiSuSheZhi")%>'},
				]
			}
			settingMenus.map((items)=>{
				items.disabled = (status == 'Off' || writableMap.CODE_GNB_SETTINGS != true) ? true : false;
			})
			vm.settingMenus = settingMenus;
			vm.getDeviceSasEnableStatus(row);
			
			// 只有从 gnodeb_monitor.jsp 打开时才启动定时器
			if(vm.pageSource === 'gnodeb_monitor') {
				vm.startMonitorTimer();
			}
		},
		getDeviceSasEnableStatus(row){
            var vm = this,
                sn = row.serial_number
                params={
                    sn:sn
                };
            axios.post('${ctx}/gnb/quicksetting/getSasEnable.action',stringify(params)).then(function(response){
                var data = response.data;
                vm.sasEnableStatus = [1,2,'1','2'].includes(data)? true : false;
            }).catch(function(error){});
        },
		startMonitorTimer(){
			// 启动定时器刷新设备监控列表
			if(typeof gnbMonitor !== 'undefined' && gnbMonitor.$refs && gnbMonitor.$refs.monitor) {
				if(window.updateGnbMonitorTimer) clearInterval(window.updateGnbMonitorTimer);
				window.updateGnbMonitorTimer = setInterval(function(){
					var gnbSettingPageCtn = $("#gnbSettingPage");			
					if(!gnbSettingPageCtn.length) {
						clearInterval(window.updateGnbMonitorTimer);
						return;
					}
					gnbMonitor.$refs.monitor.refresh();
				},6000);
			}
		},
		changeMain(type){
			var vm = this, str = Math.random().toString(),
				urlList = {
					info:'${ctx}/gnb/setting/openOverviewPage.action?randomValue='+str,
					chart:'${ctx}/gnb/setting/openStatisticPage.action?randomValue='+str,
					alarm: '${ctx}/gnb/setting/openAlarmPage.action?randomValue='+str,
					topo: '${ctx}/gnb/gnbMonitor/toGnodebTOPOPage.action?randomValue='+str,
					quickSetting_X86: '${ctx}/cell/quicksettings/goGNBQuickSettingPage.action?randomValue='+str,
					quickSetting_GT: '${ctx}/gnb/setting/openQuickSettingPage.action?randomValue='+str,
					network:'${ctx}/gnb/setting/openNetworkPage.action?randomValue='+str,
					coreNetwork:'${ctx}/gnb/setting/openCoreNetworkPage.action?randomValue='+str,
					ran:'${ctx}/gnb/setting/openRanPage.action?randomValue='+str,
					bts:'${ctx}/gnb/setting/openBtsPage.action?randomValue='+str,
					system:'${ctx}/gnb/setting/openSystemPage.action?randomValue='+str,
					upgrade: '${ctx}/gnb/setting/openUpgradePage.action?randomValue='+str,
					backup: '${ctx}/enb/setting/openGNBBackupAndRestorePage.action?randomValue='+str,
					log: '${ctx}/gnb/setting/openLogPage.action?randomValue='+str,
					license: '${ctx}/gnb/setting/openLicensePage.action?randomValue='+str,
				},
			
				titleList = {
					info:'<%=rb.getString("ZongLan")%>',
					chart:'<%=rb.getString("TongJi")%>',
					alarm:'<%=rb.getString("GaoJing")%>',
					topo:'TOPO',

					quickSetting_X86:'<%=rb.getString("SheZhi")%>',
					quickSetting_GT:'<%=rb.getString("SheZhi")%>',
					network:'<%=rb.getString("SheZhi")%>',
					coreNetwork:'<%=rb.getString("SheZhi")%>',
					ran:'<%=rb.getString("SheZhi")%>',
					bts:'<%=rb.getString("SheZhi")%>',
					system:'<%=rb.getString("SheZhi")%>',

					upgrade:'<%=rb.getString("ShengJi")%>',
					backup:'<%=rb.getString("beiFenYuHuiFu")%>',
					log:'<%=rb.getString("RiZhi")%>',
					license:'<%=rb.getString("License")%>',
				};
			
			$('#gnbSetting_main').html('');
            $('#gnbSetting_main').addClass('loading');
			loadHTML(document.querySelector('#gnbSetting_main'),{
                url:urlList[type] ,
                method:'post',
                success: function() {
                	vm.currentItem = type;
                	vm.title = titleList[type];
                	eventBus.$emit("gnb-data", vm.rowData,vm.sasEnableStatus,vm.ranConfigParams);
					setTimeout(function() {
						$('#gnbSetting_main').removeClass('loading'); 
					}, 500);
                }
            });
		},
		changeSettingMain(type,navItem){
			var vm =this;

			if(navItem.disabled == true) {
				return;
			}
			vm.changeMain(type);	
		},
		settingPageClose(){
			var vm = this;
			// 根据页面来源动态关闭不同页面的slide
			if(vm.pageSource === 'monitor') {
				// 从 gnodeb_monitor.jsp 打开的情况
				eventBus.$emit('close-gnbMonitorsettingSlide');
			} else if(vm.pageSource === 'topo') {
				// 从 eNBTopo_tab.jsp 打开的情况
				eventBus.$emit('cancel-enb-topo-setting');
			}
		},
		//license Tab 右上角刷新
        gnbRefreshLicenseTab(){
            var vm = this,
                params = {
                    smallCellCode: vm.rowData.small_cell_code
                };

            // 发起同步指令- 刷新页面内容,分两种情况
            axios.post("${ctx}/cell/quicksettings/syncLicense.action",stringify(params)).then(function(res){
                var data = res.data;
                if(data["success"]){
					vm.changeMain('license');
				}else{
					vm.$message.error(data["message"])
				}
            })
        },
	},
	mounted(){
		var vm = this;
		// 定时器在 initJumpInit 中根据 pageSource 启动
		eventBus.$off('action-settingPage').$on('action-settingPage',this.initJumpInit);
		// 监听关闭事件
		eventBus.$off('close-gnb-settingPage').$on('close-gnb-settingPage',this.settingPageClose);
	}
})
</script>
