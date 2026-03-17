<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#egwSettingPage{
		height: 100%;
		width: 100%;
		background-color: #EEF1FB;
	}
	#egwSettingPage div{
		box-sizing: border-box;
	}
	#egwSettingPage .basicConfigBoxCls{
		height: calc(100% - 50px);
		padding:7px 0px 60px 40px;
		overflow: auto;
	}
	#egwSettingPage .basicConfigBoxCls .el-input__suffix{
		height: 26px;
		display: flex;
		align-items: center;
	}
	#egwSettingPage .settingMainBoxCls{
		display: flex;
		height: 100%;
		width: 100%;
		padding: 5px;
	}
	#egwSettingPage .advanceQuery{
		border-radius: 15px;
		margin-left: unset;
		margin-right: 10px;
	}
	#egwSettingPage .opBtnBoxCls{
		height: 28px;
		width: 28px;
		border: 1px solid #D7D7E6;
		border-radius: 5px;
		display: flex;
		align-items: center;
		justify-content: center;
		margin: 0px 5px;
	}
	#egwSettingPage .opBtnBoxCls:hover{
		background-color: #F5F7FA;
	}
	#egwSettingPage .settingLeftContentBoxCls{
		flex: 0 1 200px;
		margin-right: 10px;
		background-color: #FFFFFF;
		box-shadow: 0px 0px 10px 1px #E9EDF9;
		border-radius: 10px;
		border: 1px solid #E9EDF9;
	}
	#egwSettingPage .settingLeftContentBoxCls .tabsItemBoxCls{
		width: 200px;
		height: 40px;
		font-size: 12px;
		display: flex;
		align-items: center;
		border-bottom: 1px solid #E9EDF9;
		cursor: pointer;
	}
	#egwSettingPage .settingLeftContentBoxCls .activeItemCls{
		color: #4D84FF;
	}
	#egwSettingPage .tabsItemBoxCls .iconBoxCls{
		height: 20px;
		width: 20px;
		margin:0px 10px 0px 15px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 10px;
		background-color:#EDF2FF; 
	}
	#egwSettingPage .tabsItemBoxCls .iconBoxCls .el-icon,#egwSettingPage .tabsItemBoxCls .iconBoxCls .el-icon::before{
		font-size: 14px!important;
		color: #7A7992;
	}
	#egwSettingPage .activeItemCls .iconBoxCls .el-icon,#egwSettingPage .activeItemCls .iconBoxCls .el-icon::before{
		font-size: 14px;
		color: #4D84FF!important;
	}
	#egwSettingPage .settingRightContentBoxCls{
		flex: 1;
		width: calc(100% - 210px);
	}
	#egwSettingPage .settingRightContentBoxCls .blackListBoxCls,
	#egwSettingPage .settingRightContentBoxCls .addVerifiedAccountBoxCls,
	#egwSettingPage .settingRightContentBoxCls .certTableBoxCls{
		flex: 0 1 360px;
		margin-left: 10px;
		position: relative;
	}
	#egwSettingPage .settingRightContentBoxCls .blackListBoxCls .el-pagination__sizes,
	#egwSettingPage .settingRightContentBoxCls .certTableBoxCls .el-pagination__sizes{
		display: none;
	}
	#egwSettingPage .addIPsecConfigBoxCls{
		flex: 1;
		position: relative;
	}
	#egwSettingPage	.el-form-item__label{
		font-size: 12px;
	}
	#egwSettingPage .settingContentBoxCls{
		height: 100%;
		position: relative;
		display: flex;
	}
	#egwSettingPage .settingContentBoxCls >div:first-child{
		flex: 1;
		position: relative;
	}
	#egwSettingPage .settingContentBoxCls >div{
		background-color: #FFFFFF;
		box-shadow: 0px 0px 10px 1px #E9EDF9;
		border-radius: 10px;
		border: 1px solid #E9EDF9;
		overflow: hidden;
	}
	#egwSettingPage .settingContentBoxCls .footer{
		width:100%;
		border-top:1px solid #E9E9E9;
		position:absolute;
		bottom:1px;
		height:50px;
		background:#FFFFFF;
		z-index:99;
		display: flex;
		align-items: center;
		border-radius: 0px 0px 10px 10px;
	}
	#egwSettingPage .settingContentBoxCls .footer div{
		padding-left: 40px;
	}
	#egwSettingPage .titleStyML{
		margin-bottom: 10px;
		padding: 15px 0px 0px 40px;
	}
	#egwSettingPage .disabledIconBox .el-icon::before{
		color: #e9e9e9;
	}
	#egwSettingPage .itemListBoxCls{
		padding-left: 25px;
		padding-top: 5px;
	}
	#egwSettingPage .itemCls{
		height: 24px;
		display: inline-block;
		line-height: 24px;
		border: 1px solid #4D84FF;
		box-sizing: border-box;
		padding: 0px 10px;
		margin-right: 10px;
		margin-bottom: 10px;
	}
	#egwSettingPage .itemListBoxCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#egwSettingPage .el-form-item__error{
		padding-top: 0px;
		top:35px;
	}
	#egwSettingPage .mostNumberCls{
		color: #999999;
		margin-left: 10px;
	}
	#egwSettingPage .plmnBoxCls,
	#egwSettingPage .ipAndPortBoxCls,
	#egwSettingPage  .upLinkIpAndDownLinkIpBoxCls,
	#egwSettingPage  .performanceConfigBoxCls{
		display: flex;
		flex-wrap: wrap;
	}
	#egwSettingPage .leftAndRightItemCls{
		width: 50%;
	}
	#egwSettingPage .gnbConfigCls .leftAndRightItemCls{
		width: 50%;
		min-width: 480px;
	}
	#egwSettingPage .bottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#egwSettingPage .enbIdMmeListBoxCls{
		width: 90%;
		height: calc(100% - 100px);
		margin:7px 40px 0px 40px;
		padding-top: 40px;
	}
	#egwSettingPage .enbIdMmeListHeader{
		display: flex;
		justify-content: space-between;
		margin-bottom: 10px;
	}
	#egwSettingPage .enbIdMmeListHeader .el-icon::before{
		font-size: 24px;
	}
	#egwSettingPage .enbIdMmeListTableBoxCls{
		height: 85%;
	}
	#egwSettingPage .errorBoxCls{
		color:red;
		font-size:10px;
	}
	#egwSettingPage .toolbarBoxCls{
		display: flex;
		align-items: center;
		position:relative;
		justify-content:space-between;
	}
	#egwSettingPage .toolbarBoxCls .el-icon-close,#egwSettingPage .rightOutBoxHeadCls .el-icon-close{
		position: relative;
		right: unset;
	}
	#egwSettingPage .verifiedAccountToolbar .el-input.el-input--small{
		width: 330px;
	}
	#egwSettingPage .blackListToolbar {
		padding-top:10px;
	}
	#egwSettingPage .blackListToolbar .el-input.el-input--small{
		width: 212px;
	}
	.tooltipCls.is-dark{
		background : #959595 ;
		color : #FFFFFF ;
	}
	.tooltipCls[x-placement^=top] .popper__arrow ,
	.tooltipCls[x-placement^=top] .popper__arrow::after{
		border-top-color: #959595!important;
	}

	.tooltipCls[x-placement^=bottom] .popper__arrow ,
	.tooltipCls[x-placement^=bottom] .popper__arrow::after {
		border-bottom-color: #959595!important;
	}
	.tooltipCls[x-placement^=right] .popper__arrow ,
	.tooltipCls[x-placement^=right] .popper__arrow::after {
		border-right-color: #959595!important;
	}
	.tooltipCls[x-placement^=left] .popper__arrow ,
	.tooltipCls[x-placement^=left] .popper__arrow::after {
		border-left-color: #959595!important;
	}
	/*导入  */
	.importCard .w320{
		width:320px;
	}
	.importCard .el-form-item{
	 	margin-bottom:16px;
	}
	.importCard .el-dialog__body{
		padding:30px !important;
		background:#FFFFFF;
		border:none;
	}
	.importCard .el-form-item__label{
		line-height:26px;
	}
	.importCard .el-input__suffix{
		top:4px;
	}
	.importCard .fileAcceptTip{
		color:#999999;
		font-size:12px;
	}
	.importCard .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	.importCard .el-dialog__footer{
		padding:20px 0 !important;
		text-align:left;
	}
	.importCard .el-dialog__footer .el-button:first-child{
		margin:15px 0 0 30px;
	}
	.importCard .importFooter{
		border-top:1px solid #DCDFE6;	
	}
	.importCard .el-dialog__header .el-icon-close,.addBlockListDialog .el-dialog__header .el-icon-close{
		top: 0px;
		font-size: 18px;
	}
	.importCard .el-upload__tip{
		color:red;
		margin-top: unset;
	}
	#egwSettingPage .rightOutBoxHeadCls{
		height: 50px;
		display: flex;
		align-items: center;
		font-weight: 600;
		font-size: 14px;
		justify-content: space-between;
		padding: 0px 20px;
		border-bottom: 1px solid #E9EDF9;
	}
	#egwSettingPage .labelSlotCls>span{
		color: #999999;
		font-size: 12px;
		margin-left: 10px;
	}
	.greyIcon::before{
		color: #7A7992;
		font-size: 14px;
	}
	.whirtIcon::before{
		color: #FFFFFF;
		font-size: 14px;
	}
	#egwSettingPage .IPsecConfigEnableClickBox{
		height: 20px;
		width: 50px;
		position: absolute;
		top:0px;
		left: 0px;
		right: 0px;
		bottom: 0px;
		margin: auto;
		z-index: 66;
		cursor: pointer;
		opacity: 0;
	}
	#egwSettingPage .addIPsecConfigHeadCls{
		height: 50px;
		display: flex;
		align-items: center;
		font-size: 14px;
		font-weight: 550;
		border-bottom: 1px solid #E9EDF9;
	}
	#egwSettingPage .addIPsecConfigContentCls{
		height: calc(100% - 50px);
		overflow:auto;
		padding-bottom:50px;
		position: relative;
	}
	#egwSettingPage .addIPsecConfigItemTitleCls{
		display:flex;
		align-items:center;
		margin:30px 50px;
		font-size: 14px;
		font-weight: 500;
	}
	#egwSettingPage .dotCls{
		height: 6px;
		width: 6px;
		background-color: #333333;
		border-radius: 3px;
		margin-right: 10px;
	}
	#egwSettingPage .newErrorCls .el-input-group__append,.gnbConfigAddDialog .el-input-group__append{
		border-radius:0px;
		border-right:none;
	}
	#egwSettingPage .newErrorCls .validate-item .el-input-group__append,.gnbConfigAddDialog .validate-item .el-input-group__append{
		border:none;
		background:none;
	}
	#egwSettingPage .newErrorCls .validate-item .el-form-item__error,.gnbConfigAddDialog .validate-item .el-form-item__error{
		display:none;
	}
	#egwSettingPage .newErrorCls .is-error .el-input-group__append,.gnbConfigAddDialog .is-error .el-input-group__append{
		color:#FA5555;
	}
	#egwSettingPage .newErrorCls .el-form-item__error,.gnbConfigAddDialog .el-form-item__error{
		padding-top: 0px;
	}
	#egwSettingPage .newErrorCls .validate-item .el-input__inner{
		width:230px;
	}
	.gnbConfigAddDialog .validate-item .el-input__inner{
		width:200px;
	}
	.gnbConfigAddDialog .el-dialog__header .el-icon-close{
		top: 0px;
		font-size: 18px;
	}
	#egwSettingPage .newErrorCls .selectErrcCls .el-form-item__error{
		position: absolute;
		left: 120px;
		top:10px !important;
	}
	#egwSettingPage .formItemBoxCls{
		display:flex;
		flex-wrap: wrap;
		margin-left:70px;
	}
	#egwSettingPage .formItemBoxCls .el-form-item{
		width: 40%;
		min-width: 412px;
	}
	#egwSettingPage .IPsecCertContentFormItemCls .el-form-item{
		width: 40%;
		min-width: 360px;
	}
	#egwSettingPage .IPsecCertContentCls{
		height: calc(100% - 50px);
		padding: 25px 50px 80px 50px;
		overflow: auto;
	}
	#egwSettingPage .selectWidthCls .el-select>.el-input{
		width: 190px;
	}
	#egwSettingPage .applyCertBoxCls{
		background-color: #F5F8FF;
		border-radius: 5px;
		border: 1px dashed #4D84FF;
		padding: 20px;
		margin-bottom: 20px;
	}
	#egwSettingPage .applyCertBoxCls .circleIconBoxCls{
		height: 25px;
		width: 25px;
		display: flex;
		justify-content: center;
		align-items: center;
		border-radius: 12px;
		background-color: #E4F1FF;
		margin-right: 5px;
	}
	#egwSettingPage .applyCertBoxCls >div:first-child{
		font-size: 14px;
		color: #000;
		margin-bottom: 20px;
		font-weight: 550;
		display: flex;
		align-items: center;
	}
	#egwSettingPage .certificateStatusItemBoxCls{
		min-width: 300px;
	}
	#egwSettingPage .certificateStatusItemBoxCls >div{
		font-size: 12px;
		color: #333333;
		margin-bottom: 10px;
		
	}
	#egwSettingPage .certificateStatusItemBoxCls >div span{
		display: inline-block;
		width: 140px;
		color:#666666;
	}
	#egwSettingPage .basicConfigItemTitleCls{
		display:flex;
		align-items:center;
		margin:30px 50px 10px 10px;
		font-size: 14px;
		font-weight: 550;
	}
	#egwSettingPage .IPsecCertItemBoxCls{
		display:flex;
		flex-wrap: wrap;
		padding-left: 15px;
	}
	.lastFetchPopoverClass .lastFetchInfoBoxCls{
		padding: 10 15px;
	}
	.lastFetchPopoverClass .lastFetchInfoBoxCls >div{
		margin-bottom: 5px;
		display: flex;
	}
	.lastFetchPopoverClass .lastFetchInfoBoxCls >div:first-child{
		color: #7A7992;
		font-weight: 550;
	}
	.fetchLabelEnCls{
		width:150px; 
	}
	.fetchLabelZhCls{
		width:110px; 
	}
	.FetchSuccessIcon::before{
		color: #67D972;
		font-size: 15px;
	}
	.FetchFailedIcon::before{
		color: #E88282;
		font-size: 15px;
	}
	#egwSettingPage .UnSyncStatus::before{
		font-size: 18px;
		color:#D8C3D9;
	}
	#egwSettingPage .SyncStatus::before{
		font-size: 18px;
		color:#4ED76E;
	}
	#egwSettingPage .beyondEllipsisCls{
		width: 140px;
		overflow:hidden;
		white-space:nowrap;
		text-overflow:ellipsis;
	}
    #egwSettingPage .configTabsBoxCls{
        height:calc(100% - 120px);
        padding:20px 40px 0px 40px;
        position:relative;
    }
    #egwSettingPage .configTabsBoxCls .el-tabs{
		height: 100%;
	}
	#egwSettingPage .configTabsBoxCls .upgradeTaskBox .el-tabs__header{
		border-top: none;
		border-bottom: 1px solid #E9E9E9;
	}
	#egwSettingPage .configTabsBoxCls .el-tabs__item{
		font-size:14px;
	}
	#egwSettingPage .configTabsBoxCls .el-tabs--top{
		border:none;
	}
	#egwSettingPage .configTabsBoxCls .el-tabs__nav-scroll{
		margin-left:10px;
	}
	#egwSettingPage .configTabsBoxCls .el-tabs--card>.el-tabs__header .el-tabs__item{
		border-top:1px solid #E9E9E9;
		border-right:1px solid #E9E9E9;
		height:28px;
		line-height:28px;
		color:#666666;
		margin-right: 10px;
	}
	#egwSettingPage .configTabsBoxCls  .el-tabs--card>.el-tabs__header .el-tabs__item.is-active{
		border-top:2px solid #4D84FF!important;
		color:#4D84FF;
	}
	#egwSettingPage .configTabsBoxCls .el-tabs--card>.el-tabs__header .el-tabs__nav{
		border-right:none;
		border-top:none;
	}
	#egwSettingPage .configTabsBoxCls .el-tabs__content{
		border:1px solid #E9E9E9;
        border-top: none;
	}
	#egwSettingPage .cellItemBoxCls{
		width: calc(100% - 40px);
		display: flex;
		flex-wrap: wrap;
	}
	#egwSettingPage .gnbSectionTableCls,#egwSettingPage .addSectionTableCls{
		height: 100%;
		margin: 40px ;
	}
	.gnbConfigAddDialog .el-form{
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		margin-right: 40px;
	}
	.gnbConfigAddDialog .el-form-item{
		display: inline-block;
		width: 48%;
	}
	.gnbConfigAddDialog .el-form-item__error{
		padding-top: 0px;
		top:30px!important;
	}
	#egwSettingPage .selectAndInputBoxCls{
		margin-left:20px;
		position:relative
	}
	#egwSettingPage .selectAndInputBoxCls .el-input__suffix{
		z-index: 666
	}
	#egwSettingPage .selectAndInput_input {
		padding-right: 30px;
		position:absolute;
		left:1px;
		top: 1px;
	}
	#egwSettingPage .selectAndInput_input .el-input__inner{
		width: 170px;
		height: 22px;
		border: none;
		margin-top: 1px;
	}
	#egwSettingPage .selectAndInput_input  .el-input-group__append{
		padding-left: 50px;
	}
	#egwSettingPage  .alarmThresholdBoxCls{
		display: flex;
		align-items: center;
		margin: 0px 0px 20px 30px;
	}
	#egwSettingPage  .alarmThresholdBoxCls .el-form-item{
		margin-bottom: 0px !important;
	}
	#egwSettingPage  .alarmThresholdBoxCls .el-form-item__error{
		top:30;
		white-space: nowrap;
	}
	#egwSettingPage  .alarmThresholdBoxCls .el-input-group__append{
		padding: 0px 10px!important;
	}
	#egwSettingPage .alarmThresholdHeadCls{
		width: 100px;
	}
	#egwSettingPage .alarmThresholdItemLabelCls{
		padding: 0px 10px 0px 30px;
	}
	#egwSettingPage .alarmThresholdItemRightLabelCls{
		margin-left: 100px;
	}
	#egwSettingPage  .promptCls{
		font-size: 12px;
		color: #BBBBBB;
		position: relative;
		top: 0px;
		margin-left: 10px;
	}
	#egwSettingPage  .promptCls .el-icon:before{
		color: #BBBBBB;
		font-size: 12px;
	}
</style>
<div class="flex-ctn" id="egwSettingPage" style="overflow:hidden">
	<div class="settingMainBoxCls">
		<div class="settingLeftContentBoxCls">
			<div v-for="(item,index) in settingTabsData" @click="tabsClick(item.id)" :class="activeTabs == item.id ? 'tabsItemBoxCls activeItemCls' : 'tabsItemBoxCls'">
				<div class="iconBoxCls"><span :class="item.icon"></span></div>
				<div>{{item.label}}</div>
			</div>
		</div>
		<div class="settingRightContentBoxCls">
			<!--通用配置-->
			<div class="settingContentBoxCls" v-show="activeTabs == '6'">
				<div>
					<div class="rightOutBoxHeadCls">
						<span><%=rb.getString("TongYongPeiZhi")%></span>
					</div>
					<div class="basicConfigBoxCls">
						<el-form ref="commonConfigForm" :model="commonConfigForm" :rules="commonConfigFormRules" label-position="top"  :hide-required-asterisk='true'>
							<div class="basicConfigItemTitleCls">
								<div class="dotCls"></div>
								<div><%=rb.getString("JiBenPeiZhi")%></div>
							</div>
							<div class="leftAndRightItemCls">
								<el-form-item prop="egwName" label="<%=rb.getString("EGWMingCheng")%>"  label-width="160px" style="margin:10px 0px 10px 25px;">
									<el-input style='width:200px;' v-model="commonConfigForm.egwName"></el-input>
								</el-form-item>
							</div>
							<div class="leftAndRightItemCls">
								<el-form-item prop="egwDescription" label="<%=rb.getString("MiaoShu")%>"  label-width="160px" style="margin:20px 0px 0px 25px;">
									<el-input type="textarea" resize="true" :rows="3" maxlength="500" v-model="commonConfigForm.egwDescription" style="width: 560px;"></el-input>
								</el-form-item>
							</div>
							<div class="basicConfigItemTitleCls">
								<div class="dotCls"></div>
								<div><%=rb.getString("KPIShangBaoPeiZhi")%></div>
							</div>
							<div class="performanceConfigBoxCls">
								<div class="leftAndRightItemCls">
									<el-form-item prop="perfSwitch" label="<%=rb.getString("ShangBaoKaiGuan")%>"  label-width="160px" style="margin: 10px 0px 10px 25px;">
										<el-select v-model='commonConfigForm.perfSwitch'>
											<el-option label='Enable' value='ON'></el-option>
											<el-option label='Disable' value='OFF'></el-option>
										</el-select>
									</el-form-item>
								</div>
								<div class="leftAndRightItemCls">
									<el-form-item prop="perfCycle" label="<%=rb.getString("ShangBaoZhouQi")%>"  label-width="160px" style="margin:10px 0px 10px 25px;">
										<el-input style='width:200px;' v-model="commonConfigForm.perfCycle" :disabled="true"></el-input>
									</el-form-item>
								</div>
							</div>
							<div class="performanceConfigBoxCls">
								<div class="leftAndRightItemCls">
									<el-form-item prop="perfUrl" label="<%=rb.getString("ShangBaoURL")%>"  label-width="160px" style="margin:10px 0px 10px 25px;">
										<el-input style='width:400px;' v-model="commonConfigForm.perfUrl"></el-input>
									</el-form-item>
								</div>
							</div>
							<div class="basicConfigItemTitleCls">
								<div class="dotCls"></div>
								<div><%=rb.getString("GaoJingYuZhiPeiZhi")%></div>
								<div class="promptCls"><span class="el-icon-circle-info el-icon" style="margin-right:5px;" ></span><%=rb.getString("GaoJingYuZhiPeiZhiTiShi")%></div>
							</div>
							<div class="alarmThresholdBoxCls">
								<div class="alarmThresholdHeadCls"><%=rb.getString("CPUShiYongLv")%></div>
								<div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
								<el-form-item prop="cpuUseRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.cpuUseRate" @change="commonConfigValid('cpuUseClearRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
								<div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
								<el-form-item prop="cpuUseClearRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.cpuUseClearRate" @change="commonConfigValid('cpuUseRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
							</div>
							<div class="alarmThresholdBoxCls">
								<div class="alarmThresholdHeadCls"><%=rb.getString("NeiCunShiYongLv")%></div>
								<div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
								<el-form-item prop="memoryUseRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.memoryUseRate" @change="commonConfigValid('memoryClearUseRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
								<div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
								<el-form-item prop="memoryClearUseRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.memoryClearUseRate" @change="commonConfigValid('memoryUseRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
							</div>
							<div class="alarmThresholdBoxCls">
								<div class="alarmThresholdHeadCls"><%=rb.getString("CiPanShiYongLv")%></div>
								<div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
								<el-form-item prop="diskUseRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.diskUseRate" @change="commonConfigValid('diskClearUseRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
								<div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
								<el-form-item prop="diskClearUseRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.diskClearUseRate" @change="commonConfigValid('diskUseRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
							</div>
							<div class="alarmThresholdBoxCls">
								<div class="alarmThresholdHeadCls"><%=rb.getString("UEJieRuLv")%></div>
								<div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
								<el-form-item prop="ueMoreLicenseRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.ueMoreLicenseRate" @change="commonConfigValid('ueMoreLicenseClearRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
								<div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
								<el-form-item prop="ueMoreLicenseClearRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.ueMoreLicenseClearRate" @change="commonConfigValid('ueMoreLicenseRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
							</div>
							<div class="alarmThresholdBoxCls">
								<div class="alarmThresholdHeadCls"><%=rb.getString("eNBJieRuLv")%></div>
								<div class="alarmThresholdItemLabelCls"><%=rb.getString("GaoJingZhi")%></div>
								<el-form-item prop="enbMoreLicenseRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.enbMoreLicenseRate" @change="commonConfigValid('enbMoreLicenseClearRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
								<div class="alarmThresholdItemLabelCls alarmThresholdItemRightLabelCls"><%=rb.getString("HuiFuZhi")%></div>
								<el-form-item prop="enbMoreLicenseClearRate" label=""  label-width="160px">
									<el-input style='width:100px;' v-model="commonConfigForm.enbMoreLicenseClearRate" @change="commonConfigValid('enbMoreLicenseRate')">
										<template slot="append">%</template>
									</el-input>
								</el-form-item>
							</div>
						</el-form>
					</div>
					<div class="footer">
                        <div class="lnkbuttonGroup">
                            <el-button type="primary" @click="submitCommonConfig"><%=rb.getString("QueDing")%></el-button>
                            <el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
                        </div>
                    </div>
				</div>
			</div>
			<!--eNB配置-->
			<div class="settingContentBoxCls" v-show="activeTabs == '1'">
				<div>
					<div class="rightOutBoxHeadCls">
						<span><%=rb.getString("JiZhanPeiZhi")%></span>
					</div>
                    <div class="configTabsBoxCls">
                        <el-tabs v-model="enbConfigActive" type="card" :before-leave="enbBeforeTabLeave">
                            <el-tab-pane label="Basic Config" name="basic">
                                <div class="basicConfigBoxCls">
                                    <el-form ref="enbSettingBasicForm" :model="enbSettingBasicForm" :rules="enbBasicFormRules" label-position="top"  :hide-required-asterisk='true'>
                                        <div class="basicConfigItemTitleCls">
                                            <div class="dotCls"></div>
                                            <div><%=rb.getString("JiBenPeiZhi")%></div>
                                        </div>
                                        <div class="plmnBoxCls">
                                            <div class="leftAndRightItemCls">
                                                <el-form-item prop="plmn" label="PLMN"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:200px;' v-model="enbSettingBasicForm.plmn"  placeholder="<%=rb.getString("PLMNFanWei")%>">
                                                        <i slot="suffix" class="el-icon el-icon-plus" @click="addEnbPLMNs" v-show="enbSettingBasicForm.plmn && enbPlmnList.length < 8 "></i>
                                                        <div slot="suffix" class="disabledIconBox">
                                                            <i  class="el-icon el-icon-plus" v-show="enbPlmnList.length > 7 || !enbSettingBasicForm.plmn"></i>
                                                        </div>
                                                    </el-input>
                                                    <span class="mostNumberCls"><%=rb.getString("eGWZuiDuoTianJian8")%></span>
                                                    <p class="errorBoxCls">{{enbPlmnErrorMessage}}</p>
                                                </el-form-item>
                                                <div class="itemListBoxCls">
                                                    <div v-for="item in enbPlmnList" class="itemCls">
                                                        <span>{{item.Hplmn}}</span>
                                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="enbPlmnListDel(item)"></span>
                                                    </div>
                                                </div>
                                            </div>
                                                
                                        </div>
                                        <div class="ipAndPortBoxCls">
                                            <div class="leftAndRightItemCls">
                                                <el-form-item prop="egwIp" label="<%=rb.getString("eGWIP")%>"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:200px;' v-model="enbSettingBasicForm.egwIp" >
                                                        <i slot="suffix" class="el-icon el-icon-plus" @click="addEgwIp" v-show="enbSettingBasicForm.egwIp && enbEgwIpList.length < 4"></i>
                                                        <div slot="suffix" class="disabledIconBox">
                                                            <i  class="el-icon el-icon-plus" v-show="!enbSettingBasicForm.egwIp || enbEgwIpList.length > 3"></i>
                                                        </div>
                                                    </el-input>
                                                    <span class="mostNumberCls"><%=rb.getString("eGWZuiDuoTianJian4")%></span>
                                                    <p class="errorBoxCls">{{ipErrorMessage}}</p>
                                                </el-form-item>
                                                <div class="itemListBoxCls">
                                                    <div v-for="item in enbEgwIpList" class="itemCls">
                                                        <span>{{item.Ip}}</span>
                                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="egwIpListDel(item)"></span>
                                                    </div>
                                                </div>
                                            </div>
                                            <div class="leftAndRightItemCls">
                                                <el-form-item prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:200px;' v-model="enbSettingBasicForm.egwPort" :disabled="true"></el-input>
                                                </el-form-item>
                                            </div>
                                        </div>
                                        <div class="upLinkIpAndDownLinkIpBoxCls">
                                            <div class="leftAndRightItemCls">
                                                <el-form-item prop="upLinkIp" label="eGW data upLink IP"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:200px;' v-model="enbSettingBasicForm.upLinkIp" >
                                                        <i slot="suffix" class="el-icon el-icon-plus" @click="addEnbUpLinkIp" v-show="enbSettingBasicForm.upLinkIp && enbUpLinkIpList.length < 8 "></i>
                                                        <div slot="suffix" class="disabledIconBox">
                                                            <i  class="el-icon el-icon-plus" v-show="!enbSettingBasicForm.upLinkIp || enbUpLinkIpList.length > 7"></i>
                                                        </div>
                                                    </el-input>
                                                    <span class="mostNumberCls"><%=rb.getString("eGWZuiDuoTianJian8")%></span>
                                                    <p class="errorBoxCls">{{enbUpLinkIpErrorMessage}}</p>
                                                </el-form-item>
                                                <div class="itemListBoxCls">
                                                    <div v-for="item in enbUpLinkIpList" class="itemCls">
                                                        <span>{{item.UplinkAddr}}</span>
                                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="enbUpLinkIpListDel(item)"></span>
                                                    </div>
                                                </div>
                                            </div>
                                            <div class="leftAndRightItemCls">
                                                <el-form-item prop="downLinkIp" label="eGW data downLink IP"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:200px;' v-model="enbSettingBasicForm.downLinkIp" >
                                                        <i slot="suffix" class="el-icon el-icon-plus" @click="addEnbDownLinkIp" v-show="enbSettingBasicForm.downLinkIp && enbDownLinkIpList.length < 8 "></i>
                                                        <div slot="suffix" class="disabledIconBox">
                                                            <i  class="el-icon el-icon-plus" v-show="!enbSettingBasicForm.downLinkIp || enbDownLinkIpList.length > 7"></i>
                                                        </div>
                                                    </el-input>
                                                    <span class="mostNumberCls"><%=rb.getString("eGWZuiDuoTianJian8")%></span>
                                                    <p class="errorBoxCls">{{enbDownLinkIpErrorMessage}}</p>
                                                </el-form-item>
                                                <div class="itemListBoxCls">
                                                    <div v-for="item in enbDownLinkIpList" class="itemCls">
                                                        <span>{{item.DownlinkAddr}}</span>
                                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="enbDownLinkIpListDel(item)"></span>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </el-form>
                                </div>
                            </el-tab-pane>
                            <el-tab-pane label="eNB Config" name="enbConfig">
                                <div class="enbIdMmeListBoxCls">
                                    <div class="enbIdMmeListHeader">
                                        <span style="font-size:14px;font-weight:bold;">eNB ID-MME List</span>
                                        <span class="el-icon el-icon-circle-add" @click="linkAdd('enb')"></span>
                                    </div>
                                    <div class="enbIdMmeListTableBoxCls">
                                        <el-ctable 
                                            ref="enbIdMmeListTable" 
                                            :rownumber="true" 
                                            id="enbIdMmeListTable" 
                                            :data="enbIdMmeListTableData" 
                                            height="100%"
                                            :front-pagination="true"
                                            :pagination="true"
                                            style="border:1px solid #E9E9E9;"
                                        >
                                            <el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
                                                <template slot-scope="scope">
                                                    <span class="el-icon el-icon-operation-edit" @click="editEnbIdMme(scope.row,event)"></span>
                                                    <span class="el-icon el-icon-operation-delete" @click="delEnbIdMme(scope.row,event)" style="margin-left:20px;"></span>
                                                </template>
                                            </el-table-column>
                                            <el-table-column label='eNodeB ID' min-width="120" prop="enodebId" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='Link Num' min-width="120" prop="linkNum" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='PLMN' min-width="120" prop="Hplmn" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='TAC' min-width="120" prop="Tac" show-overflow-tooltip></el-table-column>
                                        </el-ctable>
                                    </div>
                                </div>
                            </el-tab-pane>
                        </el-tabs>
                    </div>
                    <div class="footer">
                        <div class="lnkbuttonGroup">
                            <el-button type="primary" @click="submitBasicAndEnbList"><%=rb.getString("QueDing")%></el-button>
                            <el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
                        </div>
                    </div>
				</div>
			</div>
			<!--gNB配置-->
			<div class="settingContentBoxCls" v-show="activeTabs == '2'">
				<div>
					<div class="rightOutBoxHeadCls">
						<span>gNB Config</span>
						<span v-if="gnbConfigActive == 'gnbSectionConfig' || gnbConfigActive == 'gnbPfmConfig'" class="el-icon el-icon-circle-refresh" style="margin-left:10px;" @click="syncSubmit"></span>
					</div>
                     <div class="configTabsBoxCls gnbConfigCls">
                        <el-tabs v-model="gnbConfigActive" type="card" :before-leave="gnbBeforeTabLeave">
                            <el-tab-pane label="Basic Config" name="basic">
                                <div class="basicConfigBoxCls newErrorCls">
                                    <el-form ref="gnbSettingBasicForm" :model="gnbSettingBasicForm" :rules="gnbBasicFormRules" label-position="top"  :hide-required-asterisk='true'>
                                        <div class="basicConfigItemTitleCls">
                                            <div class="dotCls"></div>
                                            <div><%=rb.getString("JiBenPeiZhi")%></div>
                                        </div>
                                        <div class="plmnBoxCls">
                                            <div class="leftAndRightItemCls">
                                                <!--<el-form-item prop="plmn" label="PLMN"  label-width="160px" style="margin:10px 0px 10px 25px;" class='validate-item'>
                                                    <el-input v-model="gnbSettingBasicForm.plmn">
														<template slot="append"><%=rb.getString("PLMNFanWei")%></template>
													</el-input>
                                                </el-form-item>-->
												<el-form-item prop="plmn" label="PLMN"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:230px;' v-model="gnbSettingBasicForm.plmn"  placeholder="<%=rb.getString("PLMNFanWei")%>">
                                                        <i slot="suffix" class="el-icon el-icon-plus" @click="addGnbPLMNs" v-show="gnbSettingBasicForm.plmn && gnbPlmnList.length < 1 "></i>
                                                        <div slot="suffix" class="disabledIconBox">
                                                            <i  class="el-icon el-icon-plus" v-show="gnbPlmnList.length > 0 || !gnbSettingBasicForm.plmn"></i>
                                                        </div>
                                                    </el-input>
                                                    <span class="mostNumberCls">No more than 1</span>
                                                    <p class="errorBoxCls">{{gnbPlmnErrorMessage}}</p>
                                                </el-form-item>
                                                <div class="itemListBoxCls">
                                                    <div v-for="item in gnbPlmnList" class="itemCls">
                                                        <span>{{item.Hplmn}}</span>
                                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbPlmnListDel(item)"></span>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="ipAndPortBoxCls">
                                            <div class="leftAndRightItemCls">
                                                <el-form-item label="NG-AMF IP to gNB"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:230px;' v-model="gnbSettingBasicForm.ngAmfIp">
                                                        <i slot="suffix" class="el-icon el-icon-plus" @click="addNgAmfIp" v-show="gnbSettingBasicForm.ngAmfIp && gnbNgAmfIpList.length < 4"></i>
                                                        <div slot="suffix" class="disabledIconBox">
                                                            <i  class="el-icon el-icon-plus" v-show="!gnbSettingBasicForm.ngAmfIp || gnbNgAmfIpList.length > 3"></i>
                                                        </div>
                                                    </el-input>
                                                    <span class="mostNumberCls">No more than 4,Support configuration of IPV4 or IPV6</span>
                                                    <p class="errorBoxCls">{{ngAmfIpErrorMessage}}</p>
                                                </el-form-item>
												<el-form-item  prop="gnbNgAmfIpList" label="" v-show="false"  label-width="160px" style="margin:10px 0px 10px 25px;">
													 <el-input v-model="gnbNgAmfIpList"></el-input>
												</el-form-item>
                                                <div class="itemListBoxCls">
                                                    <div v-for="item in gnbNgAmfIpList" class="itemCls">
                                                        <span>{{item.Ip}}</span>
                                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbNgAmfIpListDel(item)"></span>
                                                    </div>
                                                </div>
                                            </div>
                                            <div class="leftAndRightItemCls">
                                                <el-form-item label="NG-AMF Port to gNB"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:230px;' v-model="gnbSettingBasicForm.ngAmfPort" :disabled="true"></el-input>
                                                </el-form-item>
                                            </div>
                                        </div>
                                        <div class="upLinkIpAndDownLinkIpBoxCls">
                                            <div class="leftAndRightItemCls">
												<!--<el-form-item prop="plmn" label="PLMN"  label-width="160px" style="margin:10px 0px 10px 25px;" class='validate-item'>
                                                    <el-input v-model="gnbSettingBasicForm.upLinkIp">
														<template slot="append">Support configuration of IPV4 or IPV6</template>
													</el-input>
                                                </el-form-item>-->
												<el-form-item prop="upLinkIp" label="eGW upLink GTPU IP"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:230px;' v-model="gnbSettingBasicForm.upLinkIp" >
                                                        <i slot="suffix" class="el-icon el-icon-plus" @click="addGnbUpLinkIp" v-show="gnbSettingBasicForm.upLinkIp && gnbUpLinkIpList.length < 1 "></i>
                                                        <div slot="suffix" class="disabledIconBox">
                                                            <i  class="el-icon el-icon-plus" v-show="!gnbSettingBasicForm.upLinkIp || gnbUpLinkIpList.length > 0"></i>
                                                        </div>
                                                    </el-input>
                                                    <span class="mostNumberCls">No more than 1,Support configuration of IPV4 or IPV6</span>
                                                    <p class="errorBoxCls">{{gnbUpLinkIpErrorMessage}}</p>
                                                </el-form-item>
                                                <div class="itemListBoxCls">
                                                    <div v-for="item in gnbUpLinkIpList" class="itemCls">
                                                        <span>{{item.N3bAddr}}</span>
                                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbUpLinkIpListDel(item)"></span>
                                                    </div>
                                                </div>
                                            </div>
                                            <div class="leftAndRightItemCls">
												<!--<el-form-item prop="plmn" label="PLMN"  label-width="160px" style="margin:10px 0px 10px 25px;" class='validate-item'>
                                                    <el-input v-model="gnbSettingBasicForm.downLinkIp">
														<template slot="append">Support configuration of IPV4 or IPV6</template>
													</el-input>
                                                </el-form-item>-->
												<el-form-item prop="downLinkIp" label="eGW downLink GTPU IP"  label-width="160px" style="margin:10px 0px 10px 25px;">
                                                    <el-input style='width:230px;' v-model="gnbSettingBasicForm.downLinkIp" >
                                                        <i slot="suffix" class="el-icon el-icon-plus" @click="addGnbDownLinkIp" v-show="gnbSettingBasicForm.downLinkIp && gnbDownLinkIpList.length < 1 "></i>
                                                        <div slot="suffix" class="disabledIconBox">
                                                            <i  class="el-icon el-icon-plus" v-show="!gnbSettingBasicForm.downLinkIp || gnbDownLinkIpList.length > 0"></i>
                                                        </div>
                                                    </el-input>
                                                    <span class="mostNumberCls">No more than 1,Support configuration of IPV4 or IPV6</span>
                                                    <p class="errorBoxCls">{{gnbDownLinkIpErrorMessage}}</p>
                                                </el-form-item>
                                                <div class="itemListBoxCls">
                                                    <div v-for="item in gnbDownLinkIpList" class="itemCls">
                                                        <span>{{item.N3aAddr}}</span>
                                                        <span class="el-icon el-icon-close" style="margin-left:5px;" @click="gnbDownLinkIpListDel(item)"></span>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </el-form>
                                </div>
                            </el-tab-pane>
                            <el-tab-pane label="gNB Config" name="gnbConfig">
                                <div class="enbIdMmeListBoxCls">
									<el-form ref="gnbConfigIdLengthForm" :model="gnbConfigIdLengthForm" label-position="top">
										<div class="cellItemBoxCls">
											<div class="leftAndRightItemCls">
												<el-form-item prop="" label="GNB gNodeB ID Length"  label-width="160px" style="margin: 10px 0px 10px 25px;">
													<el-select v-model='gnbConfigIdLengthForm.GNB_idLength'>
														<el-option v-for="item in GNB_idLengthList" :label='item.label' :value='item.value' :disabled="item.disabled"></el-option>
													</el-select>
												</el-form-item>
											</div>
											<div class="leftAndRightItemCls">
												<el-form-item prop="" label="AMF gNodeB ID Length"  label-width="160px" style="margin: 10px 0px 10px 25px;">
													<el-select v-model='gnbConfigIdLengthForm.AMF_idLength'>
														<el-option v-for="item in AMF_idLengthList" :label='item.label' :value='item.value' :disabled="item.disabled"></el-option>
													</el-select>
												</el-form-item>
											</div>
										</div>
									</el-form>
                                    <div class="enbIdMmeListHeader">
										<div style="display:inline-block;display: flex;align-items: center;">
											<span class="title-icon"></span>
                                        	<span style="font-size:14px;font-weight:bold;">gNB ID-MME List</span>
										</div>
                                        <span class="el-icon el-icon-circle-add" @click="linkAdd('gnb')"></span>
                                    </div>
                                    <div class="enbIdMmeListTableBoxCls">
                                        <el-ctable 
                                            ref="gnbIdMmeListTable" 
                                            :rownumber="true" 
                                            id="gnbIdMmeListTable" 
                                            :data="gnbIdMmeListTableData" 
                                            height="100%"
                                            :front-pagination="true"
                                            :pagination="true"
                                            style="border:1px solid #E9E9E9;margin-left:25px;"
                                        >
                                            <el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
                                                <template slot-scope="scope">
                                                    <span class="el-icon el-icon-operation-edit" @click="editGnbIdMme(scope.row,event)"></span>
                                                    <span class="el-icon el-icon-operation-delete" @click="delGnbIdMme(scope.row,event)" style="margin-left:20px;"></span>
                                                </template>
                                            </el-table-column>
                                            <el-table-column label='gNodeB ID' min-width="120" prop="enodebId" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='Link Num' min-width="120" prop="linkNum" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='PLMN' min-width="120" prop="Hplmn" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='TAC' min-width="120" prop="Tac" show-overflow-tooltip></el-table-column>
                                        </el-ctable>
                                    </div>
                                </div>
                            </el-tab-pane>
							<el-tab-pane label="gNB Section Config" name="gnbSectionConfig">
								<div class="gnbSectionTableCls">
									<div class="enbIdMmeListHeader">
										<div style="display:inline-block;display: flex;align-items: center;">
											<span class="title-icon"></span>
                                        	<span style="font-size:14px;font-weight:bold;">Section Config List</span>
										</div>
                                        <span class="el-icon el-icon-circle-add" @click="sectionAdd"></span>
                                    </div>
                                    <div class="enbIdMmeListTableBoxCls">
                                        <el-ctable 
                                            ref="gnbSectionListTable"
                                            :rownumber="true" 
                                            id="gnbSectionListTable" 
											:url="gnbSectionListTableUrl"
                                            height="100%"
											:time="6"
                                            :pagination="true"
                                            style="border:1px solid #E9E9E9;margin-left:25px;"
                                        >
                                            <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                                <template slot-scope="scope">
                                                    <span class="el-icon el-icon-operation-delete" @click="delSection(scope.row,event)" style="margin-left:20px;"></span>
                                                </template>
                                            </el-table-column>
                                           
                                            <el-table-column label='HPLMN' min-width="120" prop="plmn" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='TAC' min-width="120" prop="tac" show-overflow-tooltip></el-table-column>
											 <el-table-column label='SST' min-width="120" prop="sst" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='SD' min-width="120" prop="sd" show-overflow-tooltip></el-table-column>
                                        </el-ctable>
                                    </div>
								</div>
							</el-tab-pane>
							<el-tab-pane label="gNB PFM Config" name="gnbPfmConfig">
								<div class="gnbSectionTableCls">
									<div class="enbIdMmeListHeader">
										<div style="display:inline-block;display: flex;align-items: center;">
											<span class="title-icon"></span>
                                        	<span style="font-size:14px;font-weight:bold;margin-right:10px;">Set Interface</span>
											<span style="font-size:12px;color:#999999">No more than 2</span>
										</div>
                                        <span class="el-icon el-icon-circle-add" @click="interfaceAdd"></span>
                                    </div>
                                    <div style="margin-bottom:20px;">
                                        <el-ctable 
                                            ref="gnbInterfaceListTable"
                                            :rownumber="true" 
                                            id="gnbInterfaceListTable" 
											:url="gnbInterfaceListTableUrl" 
                                            height="200px"
                                            :pagination="true"
                                            style="border:1px solid #E9E9E9;margin-left:25px;"
                                        >
                                            <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                                <template slot-scope="scope">
                                                    <span class="el-icon el-icon-operation-delete" @click="delInterface(scope.row,event)" style="margin-left:20px;"></span>
                                                </template>
                                            </el-table-column>
                                           
                                            <el-table-column label='Name' min-width="120" prop="name" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='Type' min-width="120" prop="type" show-overflow-tooltip>
												<template slot-scope="scope">
                                                    <span v-if="scope.row.type == 'N3a'">For 5G base stations</span>
													<span v-if="scope.row.type == 'N3b'">Core oriented network</span>
                                                </template>
											</el-table-column>
											<el-table-column label='IP' min-width="120" prop="ipAndMask" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='MUT' min-width="120" prop="mtu" show-overflow-tooltip></el-table-column>
											<el-table-column label='Reassembly' min-width="120" prop="reassemblySwitch" show-overflow-tooltip></el-table-column>
                                        </el-ctable>
                                    </div>
									<div class="enbIdMmeListHeader">
										<div style="display:inline-block;display: flex;align-items: center;">
											<span class="title-icon"></span>
                                        	<span style="font-size:14px;font-weight:bold;margin-right:10px;">Set Route</span>
										</div>
                                        <span class="el-icon el-icon-circle-add" @click="routeAdd"></span>
                                    </div>
									 <div>
                                        <el-ctable 
                                            ref="gnbRouteListTable"
                                            :rownumber="true" 
                                            id="gnbRouteListTable" 
											:url="gnbRouteListTableUrl"
                                            height="300px"
                                            :pagination="true"
                                            style="border:1px solid #E9E9E9;margin-left:25px;"
                                        >
                                            <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                                <template slot-scope="scope">
                                                    <span class="el-icon el-icon-operation-delete" @click="delRoute(scope.row,event)" style="margin-left:20px;"></span>
                                                </template>
                                            </el-table-column>
                                           
                                            <el-table-column label='TargetIP' min-width="120" prop="targetIp" show-overflow-tooltip></el-table-column>
                                            <el-table-column label='NextHopIP' min-width="120" prop="nextHop" show-overflow-tooltip></el-table-column>
											 <el-table-column label='Device' min-width="120" prop="device" show-overflow-tooltip></el-table-column>
                                        </el-ctable>
                                    </div>
								</div>
							</el-tab-pane>
                        </el-tabs>
                    </div>     
					<div class="footer" v-if="gnbConfigActive == 'basic' || gnbConfigActive == 'gnbConfig' ">
						<div class="lnkbuttonGroup">
							<el-button type="primary" @click="submitGnbBasicAndEnbList"><%=rb.getString("QueDing")%></el-button>
							<el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
						</div>
					</div>
				</div>
			</div>
			<!--IPsec配置-->
			<div class="settingContentBoxCls" v-show="activeTabs == '3'">
				<div class="IPsecConfigTableBoxCls" v-show="!addIPsecConfigShow">
					<!--:url="IPsecConfigTableUrl"  :data="IPsecConfigTableData"-->
					<el-ctable
						id="IPsecConfigTable"
						:url="IPsecConfigTableUrl"
						:query-params="queryIPsecConfigParams" 
						ref="IPsecConfigTable" 
						:time="6"
						:height="'100%'" 
						pagination="true">
							<!-- 列表toolbar -->
						<template slot="toolbar">
							<div class="toolbarBoxCls verifiedAccountToolbar">
								<h3 style="padding-left: 20px;"><%=rb.getString("IPsecPeiZhi")%></h3>
								<div style="display: flex;align-items: center;margin-right:25px;">
									<el-query type="normal" @query="queryIPsecConfig" placeholder="Group Name"></el-query>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("TianJia")%>' placement='bottom'>
										<div class="opBtnBoxCls" @click="addIPsecConfigClick"><span class="el-icon el-icon-plus greyIcon"></span></div>
									</el-tooltip>
								</div>
								
							</div>
							
						</template>
							<!-- 列表columns -->
						<el-table-column label="" width="30" class-name="no-text-tips">
							<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
								<div class="el-icon el-icon-operation-more" @click="IPsecConfigOptClick(scope.row,event)" v-clickoutside="hideIPsecConfigMenus"></div>
							</template>
						</el-table-column>
						<el-table-column prop="" show-overflow-tooltip label="Enable" width="100">
							<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
								<div style="position:relative;">
									<el-switch v-model="scope.row.switch_enable" style="height: 18px;margin-left: 10px;"
										active-color="#4D84FF"
										active-value="1"
										inactive-value="0">	
									</el-switch>
									<div class="IPsecConfigEnableClickBox" @click="IPsecConfigEnableChange(scope.row)"></div>
								</div>
							</template>
						</el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="Group Name" min-width="240" ></el-table-column>
					</el-ctable>
					<el-cmenu ref="IPsecConfigMenu" :data="IPsecConfigMenus" @click="IPsecConfigMenuClick"></el-cmenu>
				</div>
				<div class="addIPsecConfigBoxCls newErrorCls" v-show="addIPsecConfigShow">
					<el-form label-position="top" ref="addIPsecConfigForm" :model='addIPsecConfigForm' :rules='addIPsecConfigRules'>     
						<div class="addIPsecConfigHeadCls">
							<span style="margin:0px 10px;" class="el-icon el-icon-goback greyIcon" @click="closeAddIPsecConfig"></span>
							<span v-if="optType == 'edit'"><%=rb.getString("XiuGai")%></span>
							<span v-if="optType == 'add'"><%=rb.getString("TianJia")%></span>
						</div>
						<div class="addIPsecConfigContentCls">
							<div class="addIPsecConfigItemTitleCls">
								<div class="dotCls"></div>
								<div><%=rb.getString("JiBenSheZhi")%></div>
							</div>
							<div class="formItemBoxCls">
								<el-form-item label="Group Name" label-width="110px" prop="group_name" class='validate-item'>
									<el-input type="text" maxlength="64" v-model.trim='addIPsecConfigForm.group_name' style="width:230px;" :disabled="optType == 'edit'">
										<template slot="append">String,Length：1~64</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Enable" label-width="110px" prop="switch_enable">
									<el-select v-model='addIPsecConfigForm.switch_enable'>
										<el-option label='Enable' value='1'></el-option>
										<el-option label='Disable' value='0'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="Secret Key" label-width="110px" prop="secret_key" class='validate-item'>
									<el-input type="text" maxlength="64" v-model.trim='addIPsecConfigForm.secret_key' style="width:230px;">
										<template slot="append">String,Length：0~64</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Fragmentation" label-width="110px" prop="fragmentation">
									<el-select v-model='addIPsecConfigForm.fragmentation'>
										<el-option label='YES' value='yes'></el-option>
										<el-option label='ACCEPT' value='accept'></el-option>
										<el-option label='FORCE' value='force'></el-option>
										<el-option label='NO' value='no'></el-option>
									</el-select>
								</el-form-item>
							</div>
							<div class="bottomLine"></div>
							<div class="formItemBoxCls">
								<el-form-item label="Left" label-width="110px" prop="left_ip" class='validate-item'>
									<el-input v-model.trim='addIPsecConfigForm.left_ip' maxlength="64" style="width:230px;">
										<template slot="append">String,Length：0~64</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Left ID" label-width="110px" prop="left_id" class='validate-item'>
									<el-input type="text" maxlength="64" v-model.trim='addIPsecConfigForm.left_id' style="width:230px;">
										<template slot="append">String,Length：0~64</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Left Auth" label-width="110px" prop="left_auth">
									<el-select v-model='addIPsecConfigForm.left_auth'>
										<el-option label='psk' value='psk'></el-option>
										<el-option label='pubkey' value='pubkey'></el-option>
										<el-option label='eap-aka' value='eap-aka'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="Left Source IP" label-width="110px" prop="left_source" class='validate-item'>
									<el-input v-model.trim='addIPsecConfigForm.left_source' maxlength="64" style="width:230px;">
										<!--<template slot="append"><%=rb.getString("IPMaskGeShiTiShi")%></template>-->
									</el-input>
								</el-form-item>
								<el-form-item label="Left Subnet" label-width="110px" prop="left_subnet" class='validate-item'>
									<el-input v-model.trim='addIPsecConfigForm.left_subnet' maxlength="64"  style="width:230px;">
										<template slot="append">String,Length：0~64</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Left Cert" label-width="110px" prop="left_cert">
									<el-select v-model='addIPsecConfigForm.left_cert'>
										<el-option v-for="item in leftCertData" :label='item.file_name' :value='item.file_name'></el-option>
									</el-select>
								</el-form-item>
							</div>
							<div class="bottomLine"></div>
							<div class="formItemBoxCls">
								<el-form-item label="Right" label-width="110px" prop="right_ip" class='validate-item'>
									<el-input v-model.trim='addIPsecConfigForm.right_ip' maxlength="64" style="width:230px;">
										<template slot="append">String,Length：0~64</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Right ID" label-width="110px" prop="right_id" class='validate-item'>
									<el-input type="text" maxlength="64" v-model.trim='addIPsecConfigForm.right_id' style="width:230px;">
										<template slot="append">String,Length：0~64</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Right Auth" label-width="110px" prop="right_auth">
									<el-select v-model='addIPsecConfigForm.right_auth'>
										<el-option label='psk' value='psk'></el-option>
										<el-option label='pubkey' value='pubkey'></el-option>
										<el-option label='eap-aka' value='eap-aka'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="Right Source IP" label-width="110px" prop="right_source" class='validate-item'>
									<el-input v-model.trim='addIPsecConfigForm.right_source' maxlength="64" style="width:230px;">
										<!--<template slot="append"><%=rb.getString("IPMaskGeShiTiShi")%></template>-->
									</el-input>
								</el-form-item>
								<el-form-item label="Right Subnet" label-width="110px" prop="right_subnet" class='validate-item'>
									<el-input v-model.trim='addIPsecConfigForm.right_subnet' maxlength="64" style="width:230px;">
										<template slot="append">String,Length：0~64</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Right Secret Key" label-width="110px" prop="right_secret_key" class='validate-item'>
									<el-input v-model.trim='addIPsecConfigForm.right_secret_key' maxlength="64" style="width:230px;">
										<template slot="append">String,Length：0~64</template>
									</el-input>
								</el-form-item>
							</div>
							<div class="bottomLine"></div>
							<div class="addIPsecConfigItemTitleCls">
								<div class="dotCls"></div>
								<div><%=rb.getString("GaoJiSheZhi")%></div>
							</div>
							<div class="formItemBoxCls">
								<el-form-item label="IKE Encryption" label-width="110px" prop="ike_encryption">
									<el-select v-model='addIPsecConfigForm.ike_encryption'>
										<el-option label='aes128' value='aes128'></el-option>
										<el-option label='aes256' value='aes256'></el-option>
										<el-option label='3des' value='3des'></el-option>
										<el-option label='des' value='des'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="IKE DH Group" label-width="110px" prop="ike_dh_group">
									<el-select v-model='addIPsecConfigForm.ike_dh_group'>
										<el-option label='modp768' value='modp768'></el-option>
										<el-option label='modp1024' value='modp1024'></el-option>
										<el-option label='modp1536' value='modp1536'></el-option>
										<el-option label='modp2048' value='modp2048'></el-option>
										<el-option label='modp4096' value='modp4096'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="IKE Authentication" label-width="110px" prop="ike_authentication">
									<el-select v-model='addIPsecConfigForm.ike_authentication'>
										<el-option label='sha1' value='sha1'></el-option>
										<el-option label='sha1_160' value='sha1_160'></el-option>
										<el-option label='sha256' value='sha256'></el-option>
										<el-option label='sha256_96' value='sha256_96'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="ESP Encyption" label-width="110px" prop="esp_encryption">
									<el-select v-model='addIPsecConfigForm.esp_encryption'>
										<el-option label='aes128' value='aes128'></el-option>
										<el-option label='aes256' value='aes256'></el-option>
										<el-option label='3des' value='3des'></el-option>
										<el-option label='des' value='des'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="ESP DH Group" label-width="110px" prop="esp_dh_group">
									<el-select v-model='addIPsecConfigForm.esp_dh_group'>
										<el-option label='modp768' value='modp768'></el-option>
										<el-option label='modp1024' value='modp1024'></el-option>
										<el-option label='modp1536' value='modp1536'></el-option>
										<el-option label='modp2048' value='modp2048'></el-option>
										<el-option label='modp4096' value='modp4096'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="ESP Authentication" label-width="110px" prop="esp_authentication">
									<el-select v-model='addIPsecConfigForm.esp_authentication'>
										<el-option label='sha1' value='sha1'></el-option>
										<el-option label='sha1_160' value='sha1_160'></el-option>
										<el-option label='sha256' value='sha256'></el-option>
										<el-option label='sha256_96' value='sha256_96'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="Key Life" label-width="110px" prop="key_life">
									<el-input v-model.trim='addIPsecConfigForm.key_life' style="width:230px;" placeholder="<%=rb.getString("ShiJianShuRuTiShi")%>"></el-input>
								</el-form-item>
								<el-form-item label="IKE Life Time" label-width="110px" prop="ike_life_time">
									<el-input v-model.trim='addIPsecConfigForm.ike_life_time' style="width:230px;" placeholder="<%=rb.getString("ShiJianShuRuTiShi")%>"></el-input>
								</el-form-item>
								<el-form-item label="Rekey Margin" label-width="110px" prop="rekey_margin">
									<el-input v-model.trim='addIPsecConfigForm.rekey_margin' style="width:230px;" placeholder="<%=rb.getString("ShiJianShuRuTiShi")%>"></el-input>
								</el-form-item>
								<el-form-item label="Dpd Action" label-width="110px" prop="dpd_action">
									<el-select v-model='addIPsecConfigForm.dpd_action'>
										<el-option label='none' value='none'></el-option>
										<el-option label='clear' value='clear'></el-option>
										<el-option label='hold' value='hold'></el-option>
										<el-option label='restart' value='restart'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="Dpd Delay" label-width="110px" prop="dpd_delay">
									<el-input v-model.trim='addIPsecConfigForm.dpd_delay' style="width:230px;" placeholder="<%=rb.getString("ShiJianShuRuTiShi")%>"></el-input>
								</el-form-item>
							</div>
						</div>
						<div class="footer">
							<div class="lnkbuttonGroup">
								<el-button type="primary" @click="addIPsecConfigSubmit"><%=rb.getString("QueDing")%></el-button>
								<el-button @click="closeAddIPsecConfig"><%=rb.getString("QuXiao")%></el-button>
							</div>
						</div>
					</el-form>
				</div>
			</div>
			<!--IPsec证书-->
			<div id="IPsecCertBox" class="settingContentBoxCls" v-show="activeTabs == '4'">
				<div class="IPsecCertBoxCls">
					<div class="rightOutBoxHeadCls">
						<span><%=rb.getString("eGWIPsecZhengShu")%></span>
						<div>
							<el-popover 
								trigger="hover" width='320' 
								popper-class="lastFetchPopoverClass"
								>
								<span class="el-icon el-icon-circle-refresh" @click="refreshCertInfo" slot="reference"></span>
								<div class="lastFetchInfoBoxCls">
									<div><%=rb.getString("CongWCGHuoQuZhengShuXinXi")%></div>
									<div>
										<div :class="fetchLabelClass"><%=rb.getString("ShangCiHuoQuShiJian")%><%=rb.getString("MaoHao")%></div>
										<div>{{IPsecCertForm.refresh_time}}</div>
									</div>
									<div>
										<div :class="fetchLabelClass"><%=rb.getString("ShangCiHuoQuZhuangTai")%><%=rb.getString("MaoHao")%></div>
										<div v-if="IPsecCertForm.refresh_status == '1'">
											<span class="el-icon el-icon-circle-success FetchSuccessIcon"></span>
											<span><%=rb.getString("HuoQuChengGong")%></span>
										</div>
										<div v-if="IPsecCertForm.refresh_status == '0'">
											<span class="el-icon el-icon-circle-close FetchFailedIcon"></span>
											<span><%=rb.getString("HuoQuShiBai")%></span>
										</div>
									</div>
								</div>
							</el-popover>
						</div>
					</div>
					<div class="IPsecCertContentCls newErrorCls">
						<el-form label-position="top" ref="IPsecCertForm" :model='IPsecCertForm' :rules='IPsecCertRules'>
							<div class="applyCertBoxCls">
								<div>
									<div class="circleIconBoxCls">
										<span class="el-icon el-icon-circle-cert greyIcon"></span>
									</div>
									Apply Certificate Status
								</div>
								<div>
									<div class="certificateStatusItemBoxCls">
										<div>
											<span>Certificate Name</span>
											{{IPsecCertForm.file_name}}
										</div>
									</div>
									<div class="certificateStatusItemBoxCls">
										<div>
											<span>Certificate Start Time</span>
											{{IPsecCertForm.valid_start_time}}
										</div>
										<div>
											<span>Certificate End Time</span>
											{{IPsecCertForm.valid_end_time}}
										</div>
									</div>
								</div>	
							</div>
							<div style="margin-bottom:20px;">
								<span class="el-icon el-icon-splitGroup greyIcon"></span>
								<span style="margin-left:5px;font-size:14px;color:#7A7992;"><%=rb.getString("ZhengShuGengXin")%></span>
							</div>
							<div class="IPsecCertItemBoxCls IPsecCertContentFormItemCls">
								<el-form-item label="<%=rb.getString("CAZhengShu")%>" label-width="110px" prop="ca_cert" class="selectWidthCls">
									<el-select v-model='IPsecCertForm.ca_cert' style="width:190px;">
										<el-option v-for="item in ca_certList" :label='item.file_name' :value='item.file_name'></el-option>
									</el-select>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("CAZhengShu")%>' placement='bottom'>
										<div style="display:inline-block;">
											<div class="opBtnBoxCls" @click="viewCertClick('CA')"><span class="el-icon el-icon-operation-result greyIcon"></span></div>
										</div>
									</el-tooltip>
								</el-form-item>
							</div>
							<div class="IPsecCertItemBoxCls IPsecCertContentFormItemCls">
								<el-form-item label="IPsec Certificate" label-width="110px" prop="ipsec_cert" class="selectWidthCls">
									<el-select v-model='IPsecCertForm.ipsec_cert' style="width:190px;" @change="ipsecCertChange">
										<el-option v-for="item in ipsec_certList" :label='item.file_name' :value='item.file_name'></el-option>
									</el-select>
									<el-tooltip popper-class="tooltipCls" content='IPsec Certificate' placement='bottom'>
										<div style="display:inline-block;">
											<div class="opBtnBoxCls" @click="viewCertClick('IPsec')"><span class="el-icon el-icon-operation-result greyIcon"></span></div>
										</div>
									</el-tooltip>
								</el-form-item>
								<el-form-item label="IPsec Private Key" label-width="110px" prop="private_key" class="selectWidthCls">
									<el-select v-model='IPsecCertForm.private_key' style="width:190px;">
										<el-option v-for="item in private_keyList" :label='item.file_name' :value='item.file_name'></el-option>
									</el-select>
									<el-tooltip popper-class="tooltipCls" content='IPsec Private Key' placement='bottom'>
										<div style="display:inline-block;">
											<div class="opBtnBoxCls" @click="viewCertClick('Key')"><span class="el-icon el-icon-operation-result greyIcon"></span></div>
										</div>
									</el-tooltip>
								</el-form-item>
							</div>
							<div style="margin-bottom:20px;" class="IPsecCertItemBoxCls IPsecCertContentFormItemCls">
								<el-button type="primary" @click="issueCertFile">Send to WCG</el-button>
								<el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
								<el-button type="primary" @click="updateCertFile" :disabled="issueAndUpdateCertBtnDis"><%=rb.getString("ZhengShuGengXin")%></el-button>
							</div>
							<div class="IPsecCertItemBoxCls IPsecCertContentFormItemCls">
								<el-form-item label="Enable" label-width="110px" prop="cert_manage" class="selectWidthCls">
									<el-select v-model='IPsecCertForm.cert_manage'>
										<el-option label='Enable' value='1'></el-option>
										<el-option label='Disable' value='0'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="CMP Server Address" label-width="110px" prop="cmp_server_address" class='validate-item'>
									<el-input v-model.trim='IPsecCertForm.cmp_server_address' style="width:230px;">
										<template slot="append">String,Length：0~512</template>
									</el-input>
								</el-form-item>
							</div>
							<div class="IPsecCertItemBoxCls IPsecCertContentFormItemCls">
								<el-form-item label="Region" label-width="110px" prop="country">
									<el-input v-model.trim='IPsecCertForm.country' style="width:230px;"></el-input>
								</el-form-item>
							</div>
							<div style="margin-bottom:20px;" class="IPsecCertItemBoxCls IPsecCertContentFormItemCls">
								<el-button type="primary"  @click="saveCertInfo"><%=rb.getString("QueDing")%></el-button>
								<el-button @click="closeLinkSetting"><%=rb.getString("QuXiao")%></el-button>
							</div>
						</el-form>
					</div>
				</div>
				<div class="certTableBoxCls" v-show="certTableBoxShow">
					<div style="position:absolute;width:100%;height:100%;">
						<!--:url="CAcertTableUrl"	:data="CAcertTableData" -->
						<el-ctable
							v-if="certTableType == 'CA'"
							id="CAcertTable"
							:url="CAcertTableUrl"
							:query-params="queryCAcertParams" 
							ref="CAcertTable"
							:time="6"
							:height="'100%'"
							:row-key="'file_name'"
							@selection-change='CAcertSelect'
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div class="toolbarBoxCls">
									<h3 style="padding-left: 20px;"><%=rb.getString("CAZhengShu")%></h3>
									<div style="display: flex;align-items: center;margin-right:25px;">
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("DaoRu")%>' placement='bottom'>
											<div class="opBtnBoxCls" @click="verifiedAccountImportClick('CA')"><span class="el-icon el-icon-operation-import greyIcon"></span></div>
										</el-tooltip>
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("GuanBi")%>' placement='bottom'>
											<span style="margin-left:10px;" class="el-icon el-icon-close greyIcon" @click="certTableBoxClose"></span>
										</el-tooltip>
									</div>
								</div>
								<div class="blackListToolbar" style="display: flex;align-items: center;padding-left: 15px;">
									<el-query type="normal" @query="queryCAcertList" placeholder="Certificate Name"></el-query>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("QingKong")%>' placement='bottom'>
										<div :class="clearCaBtnDis == true ? 'opBtnBoxCls disabled' : 'opBtnBoxCls'" @click="certDel('','clear','CA')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
									</el-tooltip>
								</div>
								
							</template>
								<!-- 列表columns -->
							<el-table-column type="selection" width="45" :reserve-selection="true"></el-table-column>
							<el-table-column prop="" :show-overflow-tooltip="false" label="Certificate Name" min-width="130">
								<template slot-scope="scope">
									<div style="height:54px;display: flex;align-items: center;justify-content: space-between;">
										<div>
											<el-tooltip popper-class="tooltipCls" :content='scope.row.file_name' placement='bottom'>
												<div class="beyondEllipsisCls">{{scope.row.file_name}}</div>
											</el-tooltip>
											<div style="display: flex;align-items: center;">
												<span style="margin-right:10px;">{{scope.row.file_size}}K(Byte)</span>
												<span v-if="scope.row.file_status == '0'" class="el-icon el-icon-operation-synchronize UnSyncStatus"></span>
												<span v-if="scope.row.file_status == '1'" class="el-icon el-icon-operation-synchronize SyncStatus"></span>
											</div>
										</div>
										<div style="display: flex;">
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("HuoQu")%>' placement='bottom'>
												<div v-if="scope.row.file_status == '2'" class="opBtnBoxCls" @click="getCert(scope.row,'CA')"><span class="el-icon el-icon-operation-synchronize greyIcon"></span></div>
											</el-tooltip>
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("XiaZai")%>' placement='bottom'>
												<div v-if="scope.row.file_status != '2'" class="opBtnBoxCls" @click="certDownload(scope.row,'CA')"><span class="el-icon el-icon-operation-download greyIcon"></span></div>
											</el-tooltip>
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("ShanChu")%>' placement='bottom'>
												<div class="opBtnBoxCls" @click="certDel(scope.row,'del','CA')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
											</el-tooltip>
										</div>
									</div>
								</template>
							</el-table-column>
						</el-ctable>
						<!--:url="IPsecCertTableUrl"	:data="IPsecCertTableData" -->
						<el-ctable
							v-if="certTableType == 'IPsec'"
							id="IPsecCertTable"
							:url="IPsecCertTableUrl"
							:query-params="queryIPsecCertParams" 
							ref="IPsecCertTable"
							:height="'100%'" 
							:time="6"
							:row-key="'file_name'"
							@selection-change='IPsecCertSelect'
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div class="toolbarBoxCls">
									<h3 style="padding-left: 20px;"><%=rb.getString("eGWIPsecZhengShu")%></h3>
									<div style="display: flex;align-items: center;margin-right:25px;">
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("DaoRu")%>' placement='bottom'>
											<div class="opBtnBoxCls" @click="verifiedAccountImportClick('IPsec')"><span class="el-icon el-icon-operation-import greyIcon"></span></div>
										</el-tooltip>
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("GuanBi")%>' placement='bottom'>
											<span style="margin-left:10px;" class="el-icon el-icon-close greyIcon" @click="certTableBoxClose"></span>
										</el-tooltip>
									</div>
								</div>
								<div class="blackListToolbar" style="display: flex;align-items: center;padding-left: 15px;">
									<el-query type="normal" @query="queryIPsecCertList" placeholder="Certificate Name"></el-query>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("QingKong")%>' placement='bottom'>
										<div :class="clearIPsecBtnDis == true ? 'opBtnBoxCls disabled' : 'opBtnBoxCls'" @click="certDel('','clear','IPsec')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
									</el-tooltip>
								</div>
								
							</template>
								<!-- 列表columns -->
							<el-table-column type="selection" width="45" :reserve-selection="true"></el-table-column>
							<el-table-column prop="" :show-overflow-tooltip="false" label="Certificate Name" min-width="130">
								<template slot-scope="scope">
									<div style="height:54px;display: flex;align-items: center;justify-content: space-between;">
										<div>
											<el-tooltip popper-class="tooltipCls" :content='scope.row.file_name' placement='bottom'>
												<div class="beyondEllipsisCls">{{scope.row.file_name}}</div>
											</el-tooltip>
											<div style="display: flex;align-items: center;">
												<span style="margin-right:10px;">{{scope.row.file_size}}K(Byte)</span>
												<span v-if="scope.row.file_status == '0'" class="el-icon el-icon-operation-synchronize UnSyncStatus"></span>
												<span v-if="scope.row.file_status == '1'" class="el-icon el-icon-operation-synchronize SyncStatus"></span>
											</div>
										</div>
										<div style="display: flex;">
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("HuoQu")%>' placement='bottom'>
												<div v-if="scope.row.file_status == '2'" class="opBtnBoxCls" @click="getCert(scope.row,'IPsec')"><span class="el-icon el-icon-operation-synchronize greyIcon"></span></div>
											</el-tooltip>
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("XiaZai")%>' placement='bottom'>
												<div v-if="scope.row.file_status != '2'" class="opBtnBoxCls" @click="certDownload(scope.row,'IPsec')"><span class="el-icon el-icon-operation-download greyIcon"></span></div>
											</el-tooltip>
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("ShanChu")%>' placement='bottom'>
												<div class="opBtnBoxCls" @click="certDel(scope.row,'del','IPsec')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
											</el-tooltip>
										</div>
									</div>
								</template>
							</el-table-column>
						</el-ctable>
						<!--:url="IPsecPrivateKeyTableUrl"	:data="IPsecPrivateKeyTableData" -->
						<el-ctable
							v-if="certTableType == 'Key'"
							id="IPsecPrivateKeyTable"
							:url="IPsecPrivateKeyTableUrl"
							:query-params="queryIPsecPrivateKeyParams" 
							ref="IPsecPrivateKeyTable"
							:height="'100%'"
							:time="6"
							:row-key="'file_name'"
							@selection-change='KeyCertSelect'
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div class="toolbarBoxCls">
									<h3 style="padding-left: 20px;">IPsec Private Key</h3>
									<div style="display: flex;align-items: center;margin-right:25px;">
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("DaoRu")%>' placement='bottom'>
											<div class="opBtnBoxCls" @click="verifiedAccountImportClick('Key')"><span class="el-icon el-icon-operation-import greyIcon"></span></div>
										</el-tooltip>
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("GuanBi")%>' placement='bottom'>
											<span style="margin-left:10px;" class="el-icon el-icon-close greyIcon" @click="certTableBoxClose"></span>
										</el-tooltip>
									</div>
								</div>
								<div class="blackListToolbar" style="display: flex;align-items: center;padding-left: 15px;">
									<el-query type="normal" @query="queryIPsecPrivateKeyList" placeholder="Certificate Name"></el-query>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("QingKong")%>' placement='bottom'>
										<div :class="clearKeyBtnDis == true ? 'opBtnBoxCls disabled' : 'opBtnBoxCls'" @click="certDel('','clear','Key')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
									</el-tooltip>
								</div>
								
							</template>
								<!-- 列表columns -->
							<el-table-column type="selection" width="45" :reserve-selection="true"></el-table-column>
							<el-table-column prop="" :show-overflow-tooltip="false" label="IPsec Private Key" min-width="130">
								<template slot-scope="scope">
									<div style="height:54px;display: flex;align-items: center;justify-content: space-between;">
										<div>
											<el-tooltip popper-class="tooltipCls" :content='scope.row.file_name' placement='bottom'>
												<div class="beyondEllipsisCls">{{scope.row.file_name}}</div>
											</el-tooltip>
											<div style="display: flex;align-items: center;">
												<span style="margin-right:10px;">{{scope.row.file_size}}K(Byte)</span>
												<span v-if="scope.row.file_status == '0'" class="el-icon el-icon-operation-synchronize UnSyncStatus"></span>
												<span v-if="scope.row.file_status == '1'" class="el-icon el-icon-operation-synchronize SyncStatus"></span>
											</div>
										</div>
										<div style="display: flex;">
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("HuoQu")%>' placement='bottom'>
												<div v-if="scope.row.file_status == '2'" class="opBtnBoxCls" @click="getCert(scope.row,'Key')"><span class="el-icon el-icon-operation-synchronize greyIcon"></span></div>
											</el-tooltip>
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("XiaZai")%>' placement='bottom'>
												<div v-if="scope.row.file_status != '2'" class="opBtnBoxCls" @click="certDownload(scope.row,'Key')"><span class="el-icon el-icon-operation-download greyIcon"></span></div>
											</el-tooltip>
											<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("ShanChu")%>' placement='bottom'>
												<div class="opBtnBoxCls" @click="certDel(scope.row,'del','Key')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
											</el-tooltip>
										</div>
									</div>
								</template>
							</el-table-column>
						</el-ctable>
					</div>
				</div>
			</div>
			<!--认证用户-->
			<div class="settingContentBoxCls" v-show="activeTabs == '5'">
				<div class="verifiedAccountBoxCls"> 
					<!--:url="verifiedAccountTableUrl"  :data="verifiedAccountTableData"-->
					<el-ctable
						id="verifiedAccountTable"
						:url="verifiedAccountTableUrl"
						:query-params="queryVerifiedAccountParams" 
						ref="verifiedAccountTable" 
						:time="6"
						:height="'100%'" 
						@selection-change='verifiedAccountSelect'
						pagination="true">
							<!-- 列表toolbar -->
						<template slot="toolbar">
							<div class="toolbarBoxCls verifiedAccountToolbar">
								<h3 style="padding-left: 20px;"><%=rb.getString("RenZhengYongHu")%></h3>
								<div style="display: flex;align-items: center;margin-right:25px;">
									<el-query type="normal" @query="queryVerifiedAccount" placeholder="<%=rb.getString("IMSI")%>"></el-query>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("TianJia")%>' placement='bottom'>
										<div class="opBtnBoxCls" @click="addVerifiedAccountClick"><span class="el-icon el-icon-plus greyIcon"></span></div>
									</el-tooltip>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("DaoRu")%>' placement='bottom'>
										<div class="opBtnBoxCls" @click="verifiedAccountImportClick('user')"><span class="el-icon el-icon-operation-import greyIcon"></span></div>
									</el-tooltip>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("DaoChu")%>' placement='bottom'>
										<div class="opBtnBoxCls" @click="exportVerifiedAccount"><span class="el-icon el-icon-operation-export greyIcon"></span></div>
									</el-tooltip>
									<el-tooltip popper-class="tooltipCls" content='Block List' placement='bottom'>
										<div class="opBtnBoxCls" @click="blockListClick"><span class="el-icon el-icon-operation-blacklist greyIcon"></span></div>
									</el-tooltip>
								</div>
								
							</div>
							
						</template>
							<!-- 列表columns -->
						<el-table-column type="selection" width="45"></el-table-column>
						<el-table-column label="" width="30" class-name="no-text-tips">
							<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
								<div class="el-icon el-icon-operation-more" @click="verifiedAccountOptClick(scope.row,event)" v-clickoutside="hideverifiedAccountMenus"></div>
							</template>
						</el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="IMSI" min-width="130"></el-table-column>
						<el-table-column prop="key" show-overflow-tooltip label="KEY" min-width="240" ></el-table-column>
						<el-table-column prop="opc" show-overflow-tooltip label="OPC" min-width="240" ></el-table-column>
					</el-ctable>
					<el-cmenu ref="verifiedAccountMenu" :data="verifiedAccountMenus" @click="verifiedAccountMenuClick"></el-cmenu>
					<!-- 批量操作  -->
					<el-bulk target="verifiedAccountTable" :list="verifiedAccountSelection" :row-key="'imsi'" show-prop="imsi"
						:message="bulkTableMessage">
						<template slot="button">
							<a class="linkbutton" @click="verifiedAccountDel('batch')"><span><%=rb.getString("ShanChu")%></span></a>
							<a class="linkbutton" @click="verifiedAccountMove('batch')"><span><%=rb.getString("YiDongDaoHeiMingDan")%></span></a>
						</template>
					</el-bulk>
				</div>
				<div class="blackListBoxCls" v-show="blockListBoxShow">
					<div style="position:absolute;width:100%;height:100%;">
						<!--:url="blockListTableUrl"	:data="blockListTableData" -->
						<el-ctable
							id="blockListTable"
							:url="blockListTableUrl"
							:query-params="queryBlockListParams" 
							ref="blockListTable" 
							:time="6"
							:height="'100%'" 
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div class="toolbarBoxCls">
									<h3 style="padding-left: 20px;"><%=rb.getString("HeiMingDan")%></h3>
									<div style="display: flex;align-items: center;margin-right:25px;">
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("TianJia")%>' placement='bottom'>
											<div class="opBtnBoxCls" @click="addBlockListClick"><span class="el-icon el-icon-plus greyIcon"></span></div>
										</el-tooltip>
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("DaoRu")%>' placement='bottom'>
											<div class="opBtnBoxCls" @click="verifiedAccountImportClick('block')"><span class="el-icon el-icon-operation-import greyIcon"></span></div>
										</el-tooltip>
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("DaoChu")%>' placement='bottom'>
											<div class="opBtnBoxCls" @click="exportBlockList"><span class="el-icon el-icon-operation-export greyIcon"></span></div>
										</el-tooltip>
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("GuanBi")%>' placement='bottom'>
											<span style="margin-left:10px;" class="el-icon el-icon-close greyIcon" @click="blockListBoxShow = false"></span>
										</el-tooltip>
									</div>
								</div>
								<div class="blackListToolbar" style="display: flex;align-items: center;padding-left: 15px;">
									<el-query type="normal" @query="queryBlockList" placeholder="<%=rb.getString("IMSI")%>"></el-query>
									<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("QingKong")%>' placement='bottom'>
										<div class="opBtnBoxCls" @click="blockListDel('','clear')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
									</el-tooltip>
								</div>
								
							</template>
								<!-- 列表columns -->
							<el-table-column prop="imsi" show-overflow-tooltip label="IMSI" min-width="130">
								<template slot-scope="scope">
									<div style="display: flex;align-items: center;justify-content: space-between;">
										<span>{{scope.row.imsi}}</span>
										<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("ShanChu")%>' placement='bottom'>
											<div class="opBtnBoxCls" @click="blockListDel(scope.row,'del')"><span class="el-icon el-icon-operation-delete greyIcon"></span></div>
										</el-tooltip>
									</div>
								</template>
							</el-table-column>
						</el-ctable>
					</div>
				</div>
				<div class="addVerifiedAccountBoxCls" v-show="addVerifiedAccountBoxShow">
					<div class="rightOutBoxHeadCls">
						<span v-if="optType == 'add'"><%=rb.getString("TianJia")%></span>
						<span v-if="optType == 'edit'"><%=rb.getString("XiuGai")%></span>
						<span class="el-icon el-icon-close greyIcon" @click="addVerifiedAccountClose"></span>
					</div>
					<div style="margin-left:20px;padding-top:28px;">
						<el-form label-position="top" ref="addVerifiedAccountForm" :model='addVerifiedAccountForm' :rules='addVerifiedAccountRules'>
							<el-form-item label="IMSI" label-width="110px" prop="imsi">
								<span slot="label" class="labelSlotCls">
									IMSI
									<span><%=rb.getString("15Wei10JinZhiShu")%></span>
								</span>
								<el-input :disabled="optType == 'edit'" v-model.trim='addVerifiedAccountForm.imsi' style="width:316px;"></el-input>
							</el-form-item>
							<el-form-item label="KEY" label-width="110px" prop="key">
								<span slot="label" class="labelSlotCls">
									KEY
									<span><%=rb.getString("32Wei16JinZhiShu")%></span>
								</span>
								<el-input v-model.trim='addVerifiedAccountForm.key' style="width:316px;"></el-input>
							</el-form-item>
							<el-form-item label="OPC" label-width="110px" prop="opc">
								<span slot="label" class="labelSlotCls">
									OPC
									<span><%=rb.getString("32Wei16JinZhiShu")%></span>
								</span>
								<el-input v-model.trim='addVerifiedAccountForm.opc' style="width:316px;"></el-input>
							</el-form-item>
						</el-form>
					</div>
					<div class="footer">
						<div class="lnkbuttonGroup">
							<el-button type="primary" @click="addVerifiedAccountSubmit"><%=rb.getString("QueDing")%></el-button>
							<el-button @click="addVerifiedAccountClose"><%=rb.getString("QuXiao")%></el-button>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	<!-- 切片设置新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" title="<%=rb.getString("TianJia")%>" width="1100px" :visible="addSectionDialogShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeAddSection">		
		<div style="position:relative;top:-48px;left:50px;font-size:12px;color:#9E9E9E;width:800px;">Tip：Parameter value is set after the completion of the need to restart the gateway of the monitor service</div>
		<el-form label-position="top" ref="addSectionForm" :model='addSectionForm' :rules='addSectionRules' label-position="top">     		     			            
			<el-form-item label="HPLMN" label-width="110px" prop="plmn" style="margin-left:20px;" class='validate-item'>
				<el-input v-model.trim='addSectionForm.plmn' style="width:200px;">
					<template slot="append">Length：5~6 Digit,Integer</template>
				</el-input>
			</el-form-item>
         	<el-form-item label="TAC" label-width="110px" prop="tac" class='validate-item'>
         		<el-input v-model.trim='addSectionForm.tac' style="width:200px;">
					<template slot="append">Range:1-16777215,except 16777214</template>
				</el-input>
         	</el-form-item>   
			 <el-form-item label="SST" label-width="110px" prop="sst" style="" class='validate-item selectAndInputBoxCls'>
				<el-select v-model='addSectionForm.sst' style="width:200px;">
					<el-option label='1--eMBB' value='1--eMBB'></el-option>
					<el-option label='2--URLLC' value='2--URLLC'></el-option>
					<el-option label='3--MIoT' value='3--MIoT'></el-option>
					<el-option label='4--V2X' value='4--V2X'></el-option>
					<el-option label='5--HMTC' value='5--HMTC'></el-option>
				</el-select>
				<el-input class="selectAndInput_input" v-model.trim='addSectionForm.sst' maxlength="10">
					<template slot="append" style="margin-left:30px;">Custom Range：128~255</template>
				</el-input>

         	</el-form-item>   
			<el-form-item label="SD" label-width="110px" prop="sd" class='validate-item'>
         		<el-input v-model.trim='addSectionForm.sd' style="width:200px;">
					<template slot="append">Range：0~16777215</template>
				</el-input>
         	</el-form-item>         	    
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addSectionSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="closeAddSection"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- PMF 连接设置新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" title="<%=rb.getString("TianJia")%>" width="860px" :visible="addInterfaceDialogShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeAddInterface">		
		<el-form label-position="top" ref="addInterfaceForm" :model='addInterfaceForm' :rules='addInterfaceRules' label-position="top">     		     			            
			<el-form-item label="Name" label-width="110px" prop="name" style="margin-left:20px;">
				<el-select v-model='addInterfaceForm.name' style="width:200px;">
					<el-option v-for="(item,index) in CardList" :label='item.NetCardDiscrip' :value='item.NetCardDiscrip'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Type" label-width="110px" prop="type">
				<el-select v-model='addInterfaceForm.type' style="width:200px;">
					<el-option label='For 5G base stations' value='N3a'></el-option>
					<el-option label='Core oriented network' value='N3b'></el-option>
				</el-select>
         	</el-form-item>   
         	<el-form-item label="IP" label-width="110px" prop="ipAndMask" style="margin-left:20px;" class='validate-item'>
         		<el-input v-model.trim='addInterfaceForm.ipAndMask' style="width:200px;">
					 <template slot="append"><%=rb.getString("IPMaskGeShiTiShi")%></template>
				 </el-input>
         	</el-form-item>
			<el-form-item label="MTU" label-width="110px" prop="mtu" class='validate-item'>
         		<el-input v-model.trim='addInterfaceForm.mtu' style="width:200px;">
					<template slot="append"><%=rb.getString("FanWei")%>：64~9220</template>
				</el-input>
         	</el-form-item>       
			 <el-form-item label="Reassembly" label-width="110px" prop="reassemblySwitch" style="margin-left:20px;">
				<el-select v-model='addInterfaceForm.reassemblySwitch' style="width:200px;">
					<el-option label='ON' value='on'></el-option>
					<el-option label='OFF' value='off'></el-option>
				</el-select>
         	</el-form-item>   
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addInterfaceSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="closeAddInterface"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- PMF 路由新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" title="<%=rb.getString("TianJia")%>" width="700px" :visible="addRouteDialogShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeAddRoute">		
		<el-form label-position="top" ref="addRouteForm" :model='addRouteForm' :rules='addRouteRules' label-position="top">     		     			            
			<el-form-item label="TargetIP" label-width="110px" prop="targetIp" style="margin-left:20px;">
				<el-input v-model.trim='addRouteForm.targetIp' placeholder='<%=rb.getString("IPMaskGeShiTiShi")%>' style="width:200px;"></el-input>
			</el-form-item>
			<el-form-item label="NextHopIP" label-width="110px" prop="nextHop">
				<el-input v-model.trim='addRouteForm.nextHop' style="width:200px;"></el-input>
			</el-form-item>
         	<el-form-item label="Device" label-width="110px" prop="device" style="margin-left:20px;">
         		<el-input v-model.trim='addRouteForm.device' style="width:200px;"></el-input>
         	</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addRouteSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="closeAddRoute"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- 认证用户导入 弹窗 -->
	<el-dialog title="<%=rb.getString("DaoRu")%>" width="480px" :visible="verifiedAccountImportShow" class="importCard" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeImportParams">		
		<el-form label-position="top" ref="verifiedAccountImportForm" :model='verifiedAccountImportForm' :rules='verifiedAccountImportRules'>     		     			            
        	<el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="110px" prop="" style="margin-bottom:0px;">
               	<el-upload 
             		ref="verifiedAccountUpload"
             		:before-upload='verifiedAccountBeforeUpload' 
             		:on-success='verifiedAccountCheckFile' 
             		:on-change="verifiedAccountFileChange"  
             		:show-file-list=false 	                  		
				    :action="verifiedAccountImportForm.uploadFileUrl" 
				    :data="verifiedAccountImportFileParams" 
				    name="uploadFile" 
				    :accept="acceptType"
				    :auto-upload="false">
					<el-input :value="verifiedAccountImportFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w320">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="verifiedAccountImportFileSelect"></a>
					</el-input>
					<div slot="tip" :class="verifiedAccountImportForm.errMassageShow? 'el-upload__tip':'fileAcceptTip'">{{verifiedAccountImportForm.errMassage}}</div>
					<a slot="trigger" ref="file_up"></a>
				</el-upload>	
         	</el-form-item> 
         	<el-form-item label="" v-if="importType == 'user' || importType == 'block'">
         		<div style='color:#999999;'>
					<span style="cursor:pointer;" @click="verifiedAccountExportTemplate">
						<span class='el-icon el-icon-common-download'></span>
						<span style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
					</span>
				</div> 
         	</el-form-item>      	    
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="verifiedAccountUploadParams"><%=rb.getString("QueDing")%></el-button>
              <el-button @click="closeImportParams"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- USIM黑名单新增 弹窗 -->
	<el-dialog class="addBlockListDialog" title="<%=rb.getString("TianJia")%>" width="480px" :visible="addBlockListDialogShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeAddBlockList">		
		<el-form label-position="top" ref="addBlockListForm" :model='addBlockListForm' :rules='addBlockListRules'>     		     			            
        	<el-form-item label="IMSI" label-width="110px" prop="imsi" style="margin-left:20px;">
				<span slot="label" class="labelSlotCls">
					IMSI
					<span><%=rb.getString("15Wei10JinZhiShu")%></span>
				</span>
				<el-input v-model.trim='addBlockListForm.imsi' style="width:316px;"></el-input>
			</el-form-item>
         	<el-form-item label="" v-if="false">
         		<el-input style="width:316px;"></el-input>
         	</el-form-item>      	    
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addBlockListSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="closeAddBlockList"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script type="text/javascript">
	var certInfoTimer;
	var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
		regKey = /^[A-Fa-f0-9]{32}$/,
		regNumber = /^[0-9]{15}$/;
	
	var egwSettingVue = new Vue({
		el: '#egwSettingPage',
		data(){
			var vm = this;
			var validateIMSI = function(rule,value,callback) {
		        	
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("15Wei10JinZhiShu")%>');
					}else {
						if(regNumber.test(value)) {
							callback();
						}else{
							callback('<%=rb.getString("15Wei10JinZhiShu")%>');
						}
					}
				},
				validateKEY = function(rule,value,callback) {
		        	
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("32Wei16JinZhiShu")%>');
					}else {
						if(regKey.test(value)) {
							callback();
						}else{
							callback('<%=rb.getString("32Wei16JinZhiShu")%>');
						}
					}
				},
				validateOPC = function(rule,value,callback) {
		        	
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("32Wei16JinZhiShu")%>');
					}else {
						if(regKey.test(value)) {
							callback();
						}else{
							callback('<%=rb.getString("32Wei16JinZhiShu")%>');
						}
					}
				},
				validate_IPOrPort = function(rule,value,callback) {
		        	var regIpOrPort = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\/([0-9]|[12][0-9]|3[012])$/
					if( value === '' || value === null || value === undefined) {
						callback();
					}else {
						if(regIpOrPort.test(value)){
							callback();
						}else{
							callback('error');
						}
						
					}
				},
				validateRange = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					var mag = rule.mag;
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(value == '' || value == undefined || value == null){
						callback(new Error(mag))
					}else{
						if(reg.test(value) && value >= min && value <= max){
							callback();
						}else{
							callback(new Error(mag))
						}
					}
					
				},
				validate_TimeStr = (rule,value,callback)=>{
					
					var reg = /^\+?[1-9][0-9]*$/;
					if(value == '' || value == undefined || value == null){
						callback()
					}else{
						if(value.length < 2){
							callback(new Error('<%=rb.getString("ShiJianShuRuTiShi")%>'))
						}else{
							var lastStr = value.charAt(value.length - 1);
							var delLastStr = value.slice(0,value.length - 1);
							if((lastStr == 'd' || lastStr == 'h' || lastStr == 'm' || lastStr == 's') && reg.test(delLastStr)){
								callback();
							}else{
								callback(new Error('<%=rb.getString("ShiJianShuRuTiShi")%>'))
							}
						}
						
					}
					
				},
				validatePLMN = (rule,value,callback) => {
					var reg = /^[0-9]{5,6}$/

					if(value === ''){
						callback()
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}
					} 
				},
				validateIPV4andIPV6 = (rule,value,callback) => {

					if(value === ''){
						callback()
					}else{
						if(vm.isValidIP(value) || vm.isIPv6(value)){
							callback();
						}else{
							callback(new Error('error'))
						}
					} 
				},
				validateSectionPLMN = (rule,value,callback) => {
					var reg = /^[0-9]{5,6}$/

					if(value === ''){
						callback(new Error('error'))
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}
					} 
				},
				validateSectionTac = (rule,value,callback) => {

					if(value === ''){
						callback(new Error('error'))
					}else{
						if(vm.isNumeric(value)&&parseInt(value)>=1 && parseInt(value)<=16777215){
							if(parseInt(value) == 16777214){
								callback(new Error('error'))
							}else{
								callback();
							}
						}else{
							callback(new Error('error'))
						}
					} 
				},
				validateSectionSst = (rule,value,callback) => {
					var selectFlag = ['1--eMBB','2--URLLC','3--MIoT','4--V2X','5--HMTC'].includes(value); 
					if(selectFlag){
						callback();
					}else{
						if(value === ''){
							callback(new Error('error'))
						}else{
							if(vm.isNumeric(value)&&parseInt(value)>=128 && parseInt(value)<=255){
								callback();
							}else{
								callback(new Error('error'))
							}
						}
						
					}
				},
				validateSectionSd = (rule,value,callback) => {

					if(value === ''){
						callback()
					}else{
						if(vm.isNumeric(value)&&parseInt(value)>=0 && parseInt(value)<=16777215){
							callback();
						}else{
							callback(new Error('error'))
						}
					} 
				},
				validateInterfaceIPorMask = (rule,value,callback) => {
					var regIpOrMask = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\/([0-9]|[12][0-9]|3[012])$/,
						regIPV6OrMask = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^:((:[\da-fA-F]{1,4}){1,6}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){6}:\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$/;
					if( value === '' || value === null || value === undefined) {
						callback(new Error('error'))
					}else {
						if(regIpOrMask.test(value) || regIPV6OrMask.test(value)){
							callback();
						}else{
							callback(new Error('error'))
						}
						
					}
				},
				validateInterfaceMtu = (rule,value,callback) => {

					if(value === ''){
						callback(new Error('error'))
					}else{
						if(vm.isNumeric(value)&&parseInt(value)>=64 && parseInt(value)<=9220){
							callback();
						}else{
							callback(new Error('error'))
						}
					} 
				},
				validateRouteIPorMask = function(rule,value,callback) {
					var regIpOrMask = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\/([0-9]|[12][0-9]|3[012])$/,
					    regIPV6OrMask = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^:((:[\da-fA-F]{1,4}){1,6}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$|^([\da-fA-F]{1,4}:){6}:\/(?:\d|[1-9]\d|1[0-1]\d|12[0-8])$/;
					if( value === '' || value === null || value === undefined) {
						callback(new Error('<%=rb.getString("IPMaskGeShiTiShi")%>'))
					}else {
						if(regIpOrMask.test(value) || regIPV6OrMask.test(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("IPMaskGeShiTiShi")%>'))
						}
						
					}
				},
				validateRouteIP = function(rule,value,callback) {
		        	
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("BiTian")%>');
					}else {
						if(regIp.test(value) || vm.isIPv6(value)){
							callback();
						}else{
							callback('<%=rb.getString("IPGeShiBuDui")%>');
						}
						
					}
				},
				validateAlarmValOrRegainVal = function(rule,value,callback) {
					var contrast = rule.contrast;
		        	var type = rule.type;
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("FanWei")%>1~100,Integer');
					}else {
						if(vm.isNumeric(value)&&parseInt(value)>=1 && parseInt(value)<=100){
							if(vm.commonConfigForm[contrast]){
								if(type == 'max'){
									if(parseInt(value) <= parseInt(vm.commonConfigForm[contrast])){
										callback('<%=rb.getString("GaoJingZhiYingDaYuHuiFuZhiTiShi")%>');
									}else{
										callback();
									}
								}else{
									if(parseInt(value) >= parseInt(vm.commonConfigForm[contrast])){
										callback('<%=rb.getString("HuiFuZhiYingXiaoYuGaoJingZhiTiShi")%>');
									}else{
										callback();
									}
								}
							}else{
								callback();
							}
						}else{
							callback('<%=rb.getString("FanWei")%>1~100,Integer')
						}
					}
				};
			return {
				activeTabs:'',
                enbConfigActive:'basic',
                gnbConfigActive:'basic',
				settingTabsData:[],
				egwCode:'',
				enbSettingBasicForm:{
					plmn:'',
					egwIp:'',
					egwPort:'36412',
					upLinkIp:'',
					downLinkIp:'',
				},
                gnbSettingBasicForm:{
					plmn:'',
					ngAmfIp:'',
					ngAmfPort:'38412',
					upLinkIp:'',
					downLinkIp:'',
				},
                enbBasicFormRules:{},
				gnbBasicFormRules:{
					// plmn:[{validator:validatePLMN,trigger:'blur'}],
					// upLinkIp:[{validator:validateIPV4andIPV6,trigger:'blur'}],
					// downLinkIp:[{validator:validateIPV4andIPV6,trigger:'blur'}],
				},
				readonly:true,
                enbPlmnList:[],
				gnbPlmnList:[],
				enbEgwIpList:[],
				gnbNgAmfIpList:[],
				enbUpLinkIpList:[],
				gnbUpLinkIpList:[],
				enbDownLinkIpList:[],
				gnbDownLinkIpList:[],

				enbIdMmeListTableData:[],
				gnbIdMmeListTableData:[],
				delEnbIdMmeList:[],
				delGnbIdMmeList:[],
				delEnbPlmnList:[],
				delGnbPlmnList:[],
				delIpList:[],
				delNgAmfIpList:[],
				delEnbUplinkAddrList:[],
				delGnbUplinkAddrList:[],
				delEnbDownlinkAddrList:[],
				delGnbDownlinkAddrList:[],
				enbPlmnErrorMessage:'',
				gnbPlmnErrorMessage:'',
				ipErrorMessage:'',
				ngAmfIpErrorMessage:'',
				enbUpLinkIpErrorMessage:'',
				gnbUpLinkIpErrorMessage:'',
				enbDownLinkIpErrorMessage:'',
				gnbDownLinkIpErrorMessage:'',

				oldGnbConfigIdLengthForm:{
					GNB_idLength:32,
					AMF_idLength:22,
				},
				gnbConfigIdLengthForm:{
					GNB_idLength:32,
					AMF_idLength:22,
				},
				idLengthList:[
					{label:'22',value:22,disabled:false},{label:'23',value:23,disabled:false},
					{label:'24',value:24,disabled:false},{label:'25',value:25,disabled:false},
					{label:'26',value:26,disabled:false},{label:'27',value:27,disabled:false},
					{label:'28',value:28,disabled:false},{label:'29',value:29,disabled:false},
					{label:'30',value:30,disabled:false},{label:'31',value:31,disabled:false},
					{label:'32',value:32,disabled:false}
				],
				addSectionDialogShow:false,
				gnbSectionListTableUrl:'',
				addSectionForm:{
					plmn:'',
					tac:'',
					sst:'',
					sd:'',
				},
				addSectionRules:{
					plmn:[{required:true,validator: validateSectionPLMN}],
					tac:[{required:true,validator: validateSectionTac}],
					sst:[{required:true,validator: validateSectionSst}],
					sd:[{validator: validateSectionSd}]
				},
				gnbInterfaceListTableUrl:'',
				addInterfaceDialogShow:false,
				addInterfaceForm:{
					name:'',
					type:'N3a',
					ipAndMask:'',
					mtu:'1500',
					reassemblySwitch:'on'
				},
				addInterfaceRules:{
					name:[{required:true}],
					ipAndMask:[{required:true,validator: validateInterfaceIPorMask}],
					mtu:[{required:true,validator: validateInterfaceMtu}]
				},
				gnbRouteListTableUrl:'',
				addRouteDialogShow:false,
				addRouteForm:{
					targetIp:'',
					nextHop:'',
					device:'',
				},
				addRouteRules:{
					targetIp:[{required:true,validator: validateRouteIPorMask}],
					nextHop:[{required:true,validator: validateRouteIP}]
				},

				verifiedAccountTableUrl:'',
				queryVerifiedAccountParams:{
					searchText:'',
				},
				verifiedAccountSelection:[],
				bulkTableMessage:{title:'<%=rb.getString("YiXuan")%>',subTitle:'IMSI',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'},
				verifiedAccountRowData:{},
				verifiedAccountMenus:[],
				verifiedAccountTableData:[
					{imsi:'460660000100065',key:'fc791d31b58bd1d4bcbf6f708387545b',opc:'fc791d31b58bd1d4bcbf6f708387545b'},
					{imsi:'460660000100066',key:'fc791d31b58bd1d4bcbf6f708387545b',opc:'fc791d31b58bd1d4bcbf6f708387545b'},
				],
				importType:'',
				acceptType:'.csv',
				verifiedAccountImportShow:false,
				verifiedAccountImportForm: {
		            uploadFileUrl: '',
					errMassageShow:false,
					errMassage:'Only .csv is supported'
		       	},	   
				verifiedAccountImportRules: {},
				verifiedAccountImportFileParams:{},              
		        verifiedAccountImportFileName:'',

				addVerifiedAccountBoxShow:false,
				addVerifiedAccountForm:{
					imsi:'',
					key:'',
					opc:'',
				},
				optType:'add',
				addVerifiedAccountRules:{
					imsi:[
                    	{validator: validateIMSI},
                    ],
					key:[
                    	{validator: validateKEY},
                    ],
					opc:[
                    	{validator: validateOPC},
                    ],
				},

				blockListBoxShow:false,
				blockListTableUrl:'',
				blockListTableData:[{imsi:'460660000100065',key:'fc791d31b58bd1d4bcbf6f708387545b',opc:'fc791d31b58bd1d4bcbf6f708387545b'},],
				queryBlockListParams:{
					searchText:''
				},
				addBlockListDialogShow:false,
				addBlockListForm:{
					imsi:''
				},
				addBlockListRules:{
					imsi:[
                    	{validator: validateIMSI},
                    ],
				},
				oldEnbDataList:{
					basicInfo:{},
					enbListInfo:[]
				},
				oldGnbDataList:{
					basicInfo:{},
					enbListInfo:[]
				},
				queryIPsecConfigParams:{
					searchText:'',
				},
				IPsecConfigRowData:{},
				IPsecConfigMenus:[],
				IPsecConfigTableUrl:'',
				IPsecConfigTableData:[
					{group_name:'test12',switch_enable:'1',opc:''},
					{group_name:'test67',switch_enable:'0',opc:''},
				],	
				addIPsecConfigShow:false,	
				addIPsecConfigForm:{
					group_name:'',
					switch_enable:'1',
					secret_key:'',
					fragmentation:'yes',

					left_ip:'',
					left_id:'',
					left_auth:'psk',
					left_source:'',
					left_subnet:'',
					left_cert:'',

					right_ip:'',
					right_id:'',
					right_auth:'psk',
					right_source:'',
					right_subnet:'',
					right_secret_key:'',

					ike_encryption:'aes128',
					ike_dh_group:'modp768',
					ike_authentication:'sha1',
					esp_encryption:'aes128',
					esp_dh_group:'modp768',
					esp_authentication:'sha1',
					key_life:'',
					ike_life_time:'',
					rekey_margin:'',
					dpd_action:'none',
					dpd_delay:''
				},	
				addIPsecConfigRules:{
					group_name:[{required:true,message:'error',trigger:'blur'}],
					// left_source:[{validator: validate_IPOrPort}],
					// right_source:[{validator: validate_IPOrPort}],
					key_life:[{validator: validate_TimeStr}],
					ike_life_time:[{validator: validate_TimeStr}],
					rekey_margin:[{validator: validate_TimeStr}],
					dpd_delay:[{validator: validate_TimeStr}]
				},
				leftCertData:[],

				IPsecCertForm:{
					cert_manage:'Enable',
					cmp_server_address:'',
					country:'',
					ca_cert:'',
					ipsec_cert:'',
					private_key:'',
					file_name:'',
					valid_start_time:'',
					valid_end_time:'',
					refresh_time:'',
					refresh_status:''
				},
				IPsecCertRules:{
					cmp_server_address:[{min:0,max:512,trigger:'change'}]
				},
				certTableBoxShow:false,
				certTableType:'',
				CAcertTableData:[{file_name:'cli.crt',file_size:'928K(Byte)',file_status:'0'},{file_name:'cli2.crt',file_size:'928K(Byte)',file_status:'0'}],
				CAcertTableUrl:'${ctx}/egw/ipsec/config/getCaCertFileList.action',
				queryCAcertParams:{
					searchText:'',
					serial_number:''
				},
				IPsecCertTableData:[{file_name:'cli.crt',file_size:'928K(Byte)',file_status:'1'}],
				IPsecCertTableUrl:'${ctx}/egw/ipsec/config/getCertFileList.action',
				queryIPsecCertParams:{
					searchText:'',
					serial_number:''
				},
				IPsecPrivateKeyTableData:[{file_name:'cli.crt',file_size:'928K(Byte)',file_status:'2'},{file_name:'cli.crt',file_size:'928K(Byte)',file_status:'1'}],
				IPsecPrivateKeyTableUrl:'${ctx}/egw/ipsec/config/getPrivateKeyFileList.action',
				queryIPsecPrivateKeyParams:{
					searchText:'',
					serial_number:''
				},
				tableCode:{
					'CA':'CAcertTable',
					'IPsec':'IPsecCertTable',
					'Key':'IPsecPrivateKeyTable'
				},
				CAcertSelection:[],
				IPsecCertSelection:[],
				KeyCertSelection:[],
				ca_certList:[],
				ipsec_certList:[],
				private_keyList:[],
				CardList:[],

				commonConfigForm:{
					egwName:'',
					egwDescription:'',
					perfSwitch:'OFF',
					perfCycle:'15min',
					perfUrl:'',
					cpuUseRate:'',
					cpuUseClearRate:'',
					memoryUseRate:'',
					memoryClearUseRate:'',
					diskUseRate:'',
					diskClearUseRate:'',
					ueMoreLicenseRate:'',
					ueMoreLicenseClearRate:'',
					enbMoreLicenseRate:'',
					enbMoreLicenseClearRate:'',
				},
				commonConfigFormRules:{
					cpuUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'cpuUseClearRate',type:'max'}],
					cpuUseClearRate:[{validator: validateAlarmValOrRegainVal,contrast:'cpuUseRate',type:'min'}],
					memoryUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'memoryClearUseRate',type:'max'}],
					memoryClearUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'memoryUseRate',type:'min'}],
					diskUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'diskClearUseRate',type:'max'}],
					diskClearUseRate:[{validator: validateAlarmValOrRegainVal,contrast:'diskUseRate',type:'min'}],
					ueMoreLicenseRate:[{validator: validateAlarmValOrRegainVal,contrast:'ueMoreLicenseClearRate',type:'max'}],
					ueMoreLicenseClearRate:[{validator: validateAlarmValOrRegainVal,contrast:'ueMoreLicenseRate',type:'min'}],
					enbMoreLicenseRate:[{validator: validateAlarmValOrRegainVal,contrast:'enbMoreLicenseClearRate',type:'max'}],
					enbMoreLicenseClearRate:[{validator: validateAlarmValOrRegainVal,contrast:'enbMoreLicenseRate',type:'min'}],
				}
			}
		},
		computed: {
			// 获取证书信息 label class
			fetchLabelClass(){
				return language == 'en'? 'fetchLabelEnCls' : 'fetchLabelZhCls';
			},
			clearCaBtnDis(){
				var vm = this;
				var fileList = vm.CAcertSelection;
				return fileList.length == 0 ? true : false ;
			},
			clearIPsecBtnDis(){
				var vm = this;
				var fileList = vm.IPsecCertSelection;
				return fileList.length == 0 ? true : false ;
			},
			clearKeyBtnDis(){
				var vm = this;
				var fileList = vm.KeyCertSelection;
				return fileList.length == 0 ? true : false ;
			},
			issueAndUpdateCertBtnDis(){
				return this.IPsecCertForm.ca_cert == '' || this.IPsecCertForm.ipsec_cert == '' || this.IPsecCertForm.private_key == ''
			},
			GNB_idLengthList(){
				var idLengthList = JSON.parse(JSON.stringify(this.idLengthList)),
					AMF_idLength = this.gnbConfigIdLengthForm.AMF_idLength;
				idLengthList.map((item)=>{
					if(item.value <= AMF_idLength){
						item.disabled = true;
					}
				})
				return idLengthList
			},
			AMF_idLengthList(){
				var idLengthList = JSON.parse(JSON.stringify(this.idLengthList)),
					GNB_idLength = this.gnbConfigIdLengthForm.GNB_idLength;
				idLengthList.map((item)=>{
					if(item.value >= GNB_idLength){
						item.disabled = true;
					}
				})
				return idLengthList
			},
		},
		watch: {},
		methods: {
			// 初始化
			init(row){
				var vm = this,
					code = row.egwCode,
					params={
						egwCode:code
					};
				vm.egwCode = row.egwCode;
				vm.serial_number = row.egwSn;
				vm.verifiedAccountTableUrl = '${ctx}/egw/ipsec/getUserList.action?egw_code='+code;
				vm.blockListTableUrl = '${ctx}/egw/ipsec/getUsimBlackList.action?egw_code='+code;
				vm.IPsecConfigTableUrl = '${ctx}/egw/ipsec/config/getConfigList.action?egw_code='+code;
				vm.queryCAcertParams.serial_number = row.egwSn;
				vm.queryIPsecCertParams.serial_number = row.egwSn;
				vm.queryIPsecPrivateKeyParams.serial_number = row.egwSn;
				var settingTabsData=[
					{id:'6',label:'<%=rb.getString("TongYongPeiZhi")%>',icon:'el-icon el-icon-circle-info1'},
					{id:'1',label:'<%=rb.getString("JiZhanPeiZhi")%>',icon:'el-icon el-icon-menu-eNB'},
					{id:'2',label:'gNB Config',icon:'el-icon el-icon-menu-5G'},
					{id:'3',label:'<%=rb.getString("IPsecPeiZhi")%>',icon:'el-icon el-icon-circle-setting1'},
					{id:'4',label:'<%=rb.getString("eGWIPsecZhengShu")%>',icon:'el-icon el-icon-circle-cert'},
					{id:'5',label:'<%=rb.getString("RenZhengYongHu")%>',icon:'el-icon el-icon-circle-user'},
				];
				var generation = row.generation;
				
				vm.getCommonConfigInfo();
				if(generation == '4G' || !generation){
					vm.settingTabsData = settingTabsData.filter((item)=>{
						return item.id != '2'
					})
					vm.init4GParams();
				}else if(generation == '5G'){
					vm.settingTabsData = settingTabsData.filter((item)=>{
						return item.id != '1'
					})
					vm.init5GParams('5G');
				}else if(generation == '4G/5G'){
					vm.settingTabsData = settingTabsData;
					vm.init4GParams();
					vm.init5GParams('4G/5G');
				}
				vm.activeTabs = vm.settingTabsData[0].id;
				
			},
			init4GParams(){
				var vm = this,
					code = vm.egwCode,
					params={
						egwCode:code
					};
				axios.post('${ctx}/egw/config/getBasicAndENBListConfig.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data){
						var basicInfo = data.basicInfo,
							enbListInfo = data.enbListInfo;
						
						vm.enbIdMmeListTableData = enbListInfo;
						if(basicInfo.HplmnList){
							vm.enbPlmnList = basicInfo.HplmnList;
						}
						if(basicInfo.IpList){
							vm.enbEgwIpList = basicInfo.IpList;
						}
						if(basicInfo.UplinkAddrList){
							vm.enbUpLinkIpList = basicInfo.UplinkAddrList;
						}
						if(basicInfo.DownlinkAddrList){
							vm.enbDownLinkIpList = basicInfo.DownlinkAddrList;
						}
						var initData = {
							HplmnList:basicInfo.HplmnList||[],
							IpList:basicInfo.IpList||[],
							UplinkAddrList:basicInfo.UplinkAddrList||[],
							DownlinkAddrList:basicInfo.DownlinkAddrList||[],
						}
						vm.oldEnbDataList.basicInfo = JSON.parse(JSON.stringify(initData));
						vm.oldEnbDataList.enbListInfo = JSON.parse(JSON.stringify(enbListInfo));
						egwMonitor.settingslideCls = '';
					}else{
						egwMonitor.settingslideCls = '';
					}
				}).catch(function(error){});
			},
			init5GParams(type){
				var vm = this,
					code = vm.egwCode,
					params={
						egwCode:code
					};
				vm.gnbSectionListTableUrl = '${ctx}/egw/config/getGnbSliceList.action?egwCode='+code;
				vm.gnbInterfaceListTableUrl = '${ctx}/egw/config/getGnbInterfaceList.action?egwCode='+code;
				vm.gnbRouteListTableUrl = '${ctx}/egw/config/getGnbRouteList.action?egwCode='+code;
				axios.post('${ctx}/egw/config/getGnbConfig.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data){
						var basicInfo = data.basicInfo,
							enbListInfo = data.enbListInfo;
						
						vm.gnbIdMmeListTableData = enbListInfo;
						vm.CardList = basicInfo.CardList;
						vm.gnbConfigIdLengthForm.GNB_idLength = data.leftSize;
						vm.gnbConfigIdLengthForm.AMF_idLength = data.rightSize;

						vm.oldGnbConfigIdLengthForm.GNB_idLength = data.leftSize;
						vm.oldGnbConfigIdLengthForm.AMF_idLength = data.rightSize;
						if(basicInfo.HplmnList){
							vm.gnbPlmnList = basicInfo.HplmnList;
						}
						if(basicInfo.IpList){
							vm.gnbNgAmfIpList = basicInfo.IpList;
						}
						if(basicInfo.UplinkAddrList){
							vm.gnbUpLinkIpList = basicInfo.UplinkAddrList;
						}
						if(basicInfo.DownlinkAddrList){
							vm.gnbDownLinkIpList = basicInfo.DownlinkAddrList;
						}
						var initData = {
							HplmnList:basicInfo.HplmnList||[],
							IpList:basicInfo.IpList||[],
							UplinkAddrList:basicInfo.UplinkAddrList||[],
							DownlinkAddrList:basicInfo.DownlinkAddrList||[],
						}
						vm.oldGnbDataList.basicInfo = JSON.parse(JSON.stringify(initData));
						vm.oldGnbDataList.enbListInfo = JSON.parse(JSON.stringify(enbListInfo));
						if(type == '5G'){
							egwMonitor.settingslideCls = '';
						}
					}else{
						if(type == '5G'){
							egwMonitor.settingslideCls = '';
						}
					}
				}).catch(function(error){});
			},
			getFileListData(){
				var vm = this;

				axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
					serial_number:vm.serial_number,
					timeZone:timeZone,
					file_type:'cacert'
				})).then(function(response){
					var data = response.data;
					if(data && data.length > 0){
						vm.ca_certList = data;
					}
				});
				axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
					serial_number:vm.serial_number,
					timeZone:timeZone,
					file_type:'ipseccert'
				})).then(function(response){
					var data = response.data;
					if(data && data.length > 0){
						vm.ipsec_certList = data;
					}
				});
				axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
					serial_number:vm.serial_number,
					timeZone:timeZone,
					file_type:'privatekey'
				})).then(function(response){
					var data = response.data;
					if(data && data.length > 0){
						vm.private_keyList = data;
					}
				})
			},
			getCommonConfigInfo(){
				var vm = this,
					code = vm.egwCode,
					params={
						egwCode:code
					};
				axios.post('${ctx}/egw/config/getCommonConfig.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data){
						Object.keys(vm.commonConfigForm).forEach(function(key){
							if(data[key]){
								vm.commonConfigForm[key] = data[key];
							}
						});
						initForm(vm.$refs.commonConfigForm);
					}
				}).catch(function(error){});

			},
			// IPsec证书下拉改变
			ipsecCertChange(val){
				var vm = this;
				vm.ipsec_certList.map((item)=>{
					if(item.file_name == val){
						vm.IPsecCertForm.file_name = item.file_name;
						vm.IPsecCertForm.valid_start_time = item.valid_start_time;
						vm.IPsecCertForm.valid_end_time = item.valid_end_time;
					}
				})
			},
			// 左侧导航切换
			tabsClick(tab){
				var vm =this;
				if(vm.activeTabs == tab)return
				if(vm.activeTabs == '1'){
					vm.activeTabs = tab;
				}else if(vm.activeTabs == '2'){
					vm.activeTabs = tab;
				}else if(vm.activeTabs == '3'){
					if(vm.addIPsecConfigShow == true){
						if(isFormChanged(vm.$refs.addIPsecConfigForm)){
							vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
								customClass:'warningConfirm',
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal:false
							}).then(() => {
								vm.activeTabs = tab;
								vm.addIPsecConfigShow = false;
								vm.resetIPsecConfigForm();
							}).catch(() => {
								
							})
						}else{
							vm.activeTabs = tab;
							vm.addIPsecConfigShow = false;
							vm.resetIPsecConfigForm();
						}
					}else{
						vm.activeTabs = tab;
					}
				}else{
					vm.activeTabs = tab;
				}

				if(vm.activeTabs == '4'){
					vm.getCertInfo();
					vm.getFileListData();
					try{
						clearInterval(certInfoTimer);
					}catch(e){}
					certInfoTimer = setInterval(function() {
						var tb = $("#IPsecCertBox");
						
						if(tb.length) {
							vm.getCertInfo();
						}else {
							clearInterval(certInfoTimer);
						}
					},6000);
					
				}else{
					try{
						clearInterval(certInfoTimer);
					}catch(e){}
				}
			},
			// ENB配置 tab切换前事件
			enbBeforeTabLeave(activeName,oldActiveName){
				var vm = this;
				var p = new Promise((resolve,reject)=>{
					if(oldActiveName == 'basic'){
						var initData = {
							HplmnList:vm.enbPlmnList,
							IpList:vm.enbEgwIpList,
							UplinkAddrList:vm.enbUpLinkIpList,
							DownlinkAddrList:vm.enbDownLinkIpList
						}
						if(vm.isObjectChange(initData,vm.oldEnbDataList['basicInfo'])){
							vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
								customClass:'warningConfirm',
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal:false
							}).then(() => {
								if(vm.oldEnbDataList['basicInfo'].HplmnList){
									vm.enbPlmnList = JSON.parse(JSON.stringify(vm.oldEnbDataList['basicInfo'].HplmnList));
								}
								if(vm.oldEnbDataList['basicInfo'].IpList){
									vm.enbEgwIpList = JSON.parse(JSON.stringify(vm.oldEnbDataList['basicInfo'].IpList));
								}
								if(vm.oldEnbDataList['basicInfo'].UplinkAddrList){
									vm.enbUpLinkIpList =JSON.parse(JSON.stringify(vm.oldEnbDataList['basicInfo'].UplinkAddrList));
								}
								if(vm.oldEnbDataList['basicInfo'].DownlinkAddrList){
									vm.enbDownLinkIpList = JSON.parse(JSON.stringify(vm.oldEnbDataList['basicInfo'].DownlinkAddrList));
								}
								resolve()
							}).catch(() => {
								reject()
							})
						}else{
							resolve() 
						}
					}else{
						resolve()
					}
				})
				return p
			},
			gnbBeforeTabLeave(activeName,oldActiveName){
				var vm = this;
				var p = new Promise((resolve,reject)=>{
					if(oldActiveName == 'basic'){
						var initData = {
							HplmnList:vm.gnbPlmnList,
							IpList:vm.gnbNgAmfIpList,
							UplinkAddrList:vm.gnbUpLinkIpList,
							DownlinkAddrList:vm.gnbDownLinkIpList
						}
						if(vm.isObjectChange(initData,vm.oldGnbDataList['basicInfo'])){
							vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
								customClass:'warningConfirm',
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal:false
							}).then(() => {
								if(vm.oldGnbDataList['basicInfo'].HplmnList){
									vm.gnbPlmnList = JSON.parse(JSON.stringify(vm.oldGnbDataList['basicInfo'].HplmnList));
								}
								if(vm.oldGnbDataList['basicInfo'].IpList){
									vm.gnbNgAmfIpList = JSON.parse(JSON.stringify(vm.oldGnbDataList['basicInfo'].IpList));
								}
								if(vm.oldGnbDataList['basicInfo'].UplinkAddrList){
									vm.gnbUpLinkIpList =JSON.parse(JSON.stringify(vm.oldGnbDataList['basicInfo'].UplinkAddrList));
								}
								if(vm.oldGnbDataList['basicInfo'].DownlinkAddrList){
									vm.gnbDownLinkIpList = JSON.parse(JSON.stringify(vm.oldGnbDataList['basicInfo'].DownlinkAddrList));
								}
								resolve()
							}).catch(() => {
								reject()
							})
						}else{
							resolve() 
						}
					}else{
						resolve()
					}
				})
				return p

			},
			//enb  PLMN 添加事件
			addEnbPLMNs(){
				var vm = this,
					val = vm.enbSettingBasicForm.plmn,
                    reg = /^\d{5,6}$/,
					params={
						Hplmn:vm.enbSettingBasicForm.plmn
					},
                    result = vm.enbPlmnList.some(item=>item.Hplmn == val);
				
				if(val){
					if(reg.test(val)) {
						if(result){
							vm.enbPlmnErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.enbPlmnList.push(params);
							vm.enbPlmnErrorMessage = '';
							vm.enbSettingBasicForm.plmn = '';
						}
					}else{
						vm.enbPlmnErrorMessage = '<%=rb.getString("PLMNFanWei")%>';
					}
				}
				
			},
			// enb  PLMN 删除事件
			enbPlmnListDel(row,type){
				var vm = this;

				if(row.configIndex){
					vm.delEnbPlmnList.push(row.configIndex);
				}
				vm.enbPlmnList = vm.enbPlmnList.filter((items)=>{
					return items.Hplmn != row.Hplmn
				})
			},
			// gnb  PLMN 添加事件
			addGnbPLMNs(){
				var vm = this,
					val = vm.gnbSettingBasicForm.plmn,
                    reg = /^\d{5,6}$/,
					params={
						Hplmn:vm.gnbSettingBasicForm.plmn
					},
                    result = vm.gnbPlmnList.some(item=>item.Hplmn == val);
				
				if(val){
					if(reg.test(val)) {
						if(result){
							vm.gnbPlmnErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.gnbPlmnList.push(params);
							vm.gnbPlmnErrorMessage = '';
							vm.gnbSettingBasicForm.plmn = '';
						}
					}else{
						vm.gnbPlmnErrorMessage = '<%=rb.getString("PLMNFanWei")%>';
					}
				}
				
			},
			// gnb PLMN 删除事件
			gnbPlmnListDel(row,type){
				var vm = this;

				if(row.configIndex){
					vm.delGnbPlmnList.push(row.configIndex);
				}
				vm.gnbPlmnList = vm.gnbPlmnList.filter((items)=>{
					return items.Hplmn != row.Hplmn
				})
			},
			// EGWIP 添加事件
			addEgwIp(){
				var vm = this,
					val = vm.enbSettingBasicForm.egwIp,
					params={
						Ip:vm.enbSettingBasicForm.egwIp
					};
				if(val){
					if(regIp.test(val)) {
						var result = vm.enbEgwIpList.some(item=>item.Ip == val);
						if(result){
							vm.ipErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.enbEgwIpList.push(params);
							vm.enbSettingBasicForm.egwIp = '';
							vm.ipErrorMessage = '';
						}
					}else {
						vm.ipErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
					}
				}
				
			},
			// EGWIP 删除事件
			egwIpListDel(row){
				var vm = this;

				if(row.configIndex){
					vm.delIpList.push(row.configIndex);
				}
				vm.enbEgwIpList = vm.enbEgwIpList.filter((items)=>{
					return items.Ip != row.Ip
				})
			},
			// gnb Ng-Amf Ip 新增事件
			addNgAmfIp(){
				var vm = this,
					val = vm.gnbSettingBasicForm.ngAmfIp,
					params={
						Ip:vm.gnbSettingBasicForm.ngAmfIp
					};
				if(val){
					if(vm.isValidIP(val) || vm.isIPv6(val)) {
						var result = vm.gnbNgAmfIpList.some(item=>item.Ip == val);
						if(result){
							vm.ngAmfIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.gnbNgAmfIpList.push(params);
							vm.gnbSettingBasicForm.ngAmfIp = '';
							vm.ngAmfIpErrorMessage = '';
						}
					}else {
						vm.ngAmfIpErrorMessage = 'Support configuration of IPV4 or IPV6';
					}
				}
			},
			// gnb Ng-Amf Ip 删除事件
			gnbNgAmfIpListDel(row){
				var vm = this;

				if(row.configIndex){
					vm.delNgAmfIpList.push(row.configIndex);
				}
				vm.gnbNgAmfIpList = vm.gnbNgAmfIpList.filter((items)=>{
					return items.Ip != row.Ip
				})
			},
			//enb  UpLink IP添加事件
			addEnbUpLinkIp(){
				var vm = this,
					val = vm.enbSettingBasicForm.upLinkIp,
					params={
						UplinkAddr:vm.enbSettingBasicForm.upLinkIp
					};
				if(val){
					if(regIp.test(val)) {
						var result = vm.enbUpLinkIpList.some(item=>item.UplinkAddr == val);
						if(result){
							vm.enbUpLinkIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.enbUpLinkIpList.push(params);
							vm.enbSettingBasicForm.upLinkIp = '';
							vm.enbUpLinkIpErrorMessage = '';
						}
					}else {
						vm.enbUpLinkIpErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
					}
				}
				
			},
			// enb UpLink IP 删除事件
			enbUpLinkIpListDel(row){
				var vm = this;
				if(row.configIndex){
					vm.delEnbUplinkAddrList.push(row.configIndex);
				}
				vm.enbUpLinkIpList = vm.enbUpLinkIpList.filter((items)=>{
					return items.UplinkAddr != row.UplinkAddr
				})
			},
			//gnb  UpLink IP添加事件
			addGnbUpLinkIp(){
				var vm = this,
					val = vm.gnbSettingBasicForm.upLinkIp,
					params={
						N3bAddr:vm.gnbSettingBasicForm.upLinkIp
					};
				if(val){
					if(vm.isValidIP(val) || vm.isIPv6(val)) {
						var result = vm.gnbUpLinkIpList.some(item=>item.N3bAddr == val);
						if(result){
							vm.gnbUpLinkIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.gnbUpLinkIpList.push(params);
							vm.gnbSettingBasicForm.upLinkIp = '';
							vm.gnbUpLinkIpErrorMessage = '';
						}
					}else {
						vm.gnbUpLinkIpErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
					}
				}
			},
			// gnb UpLink IP 删除事件
			gnbUpLinkIpListDel(row){
				var vm = this;
				if(row.configIndex){
					vm.delGnbUplinkAddrList.push(row.configIndex);
				}
				vm.gnbUpLinkIpList = vm.gnbUpLinkIpList.filter((items)=>{
					return items.N3bAddr != row.N3bAddr
				})
			},
			// enb downLink IP添加事件
			addEnbDownLinkIp(){
				var vm = this,
					val = vm.enbSettingBasicForm.downLinkIp,
					params={
						DownlinkAddr:vm.enbSettingBasicForm.downLinkIp
					};

				if(val){
					if(regIp.test(val)) {
						var result = vm.enbDownLinkIpList.some(item=>item.DownlinkAddr == val);
						if(result){
							vm.enbDownLinkIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.enbDownLinkIpList.push(params);
							vm.enbSettingBasicForm.downLinkIp = '';
							vm.enbDownLinkIpErrorMessage = '';
						}
					}else {
						vm.enbDownLinkIpErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
					}
				}
			},
			
			//  downLink IP 删除事件
			enbDownLinkIpListDel(row){
				var vm = this;
				if(row.configIndex){
					vm.delEnbDownlinkAddrList.push(row.configIndex);
				}
				vm.enbDownLinkIpList = vm.enbDownLinkIpList.filter((items)=>{
					return items.DownlinkAddr != row.DownlinkAddr
				})
			},
			// gnb  downLink IP添加事件
			addGnbDownLinkIp(){
				var vm = this,
					val = vm.gnbSettingBasicForm.downLinkIp,
					params={
						N3aAddr:vm.gnbSettingBasicForm.downLinkIp
					};

				if(val){
					if(vm.isValidIP(val) || vm.isIPv6(val)) {
						var result = vm.gnbDownLinkIpList.some(item=>item.N3aAddr == val);
						if(result){
							vm.gnbDownLinkIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							vm.gnbDownLinkIpList.push(params);
							vm.gnbSettingBasicForm.downLinkIp = '';
							vm.gnbDownLinkIpErrorMessage = '';
						}
					}else {
						vm.gnbDownLinkIpErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
					}
				}
			},
			
			//  downLink IP 删除事件
			gnbDownLinkIpListDel(row){
				var vm = this;
				if(row.configIndex){
					vm.delGnbDownlinkAddrList.push(row.configIndex);
				}
				vm.gnbDownLinkIpList = vm.gnbDownLinkIpList.filter((items)=>{
					return items.N3aAddr != row.N3aAddr
				})
			},
			// 新增链路
			linkAdd(type){
				var vm = this;
				eventBus.$emit("open-linkSetting",'add','',type);
			},
			// 修改 链路总配置
			editEnbIdMme(row,event){
				var vm = this;
				eventBus.$emit("open-linkSetting",'edit',row,'enb');
			},
			// 修改 链路总配置
			editGnbIdMme(row,event){
				var vm = this;
				eventBus.$emit("open-linkSetting",'edit',row,'gnb');
			},
			// 删除 enb 链路总配置
			delEnbIdMme(row,event){
				var vm = this;
				if(row.configIndex){
					vm.delEnbIdMmeList.push(row.configIndex);
				}
				vm.enbIdMmeListTableData = vm.enbIdMmeListTableData.filter((items)=>{
					return (items.enodebId +''+ items.Hplmn) != (row.enodebId +''+ row.Hplmn)
				})
			},
			// 删除 gnb 链路总配置
			delGnbIdMme(row){
				var vm = this;
				if(row.configIndex){
					vm.delGnbIdMmeList.push(row.configIndex);
				}
				vm.gnbIdMmeListTableData = vm.gnbIdMmeListTableData.filter((items)=>{
					return (items.enodebId +''+ items.Hplmn) != (row.enodebId +''+ row.Hplmn)
				})
			},
			// 链路设置保存 更改enbList表格信息
			editEnbList(data,type){
				var vm = this,
					editDataList = data,
					isExist = false;
				if(type == 'enb'){
					vm.enbIdMmeListTableData.forEach((items,index,array)=>{
						if((items.enodebId +''+ items.Hplmn) == (data.enodebId +''+ data.Hplmn)){
							isExist = true;
							array[index].linkNum = data.linkNum;
							array[index].Hplmn = data.Hplmn;
							array[index].Tac = data.Tac;
							array[index].enbLinkListInfo = data.enbLinkListInfo;
							array[index].isEdit = data.isEdit;
							if(data.delEnbLinkList){
								array[index].delEnbLinkList = data.delEnbLinkList;
							}else{
								array[index].delEnbLinkList = '';
							}
						}
					});
					if(!isExist){
						vm.enbIdMmeListTableData.unshift(data);
					}
				}else{
					vm.gnbIdMmeListTableData.forEach((items,index,array)=>{
						if((items.enodebId +''+ items.Hplmn) == (data.enodebId +''+ data.Hplmn)){
							isExist = true;
							array[index].linkNum = data.linkNum;
							array[index].Hplmn = data.Hplmn;
							array[index].Tac = data.Tac;
							array[index].enbLinkListInfo = data.enbLinkListInfo;
							array[index].isEdit = data.isEdit;
							if(data.delEnbLinkList){
								array[index].delEnbLinkList = data.delEnbLinkList;
							}else{
								array[index].delEnbLinkList = '';
							}
						}
					});
					if(!isExist){
						vm.gnbIdMmeListTableData.unshift(data);
					}
				}
			},
			// 同步
			syncSubmit(urls,params){
				var vm = this,
					urls='${ctx}/egw/config/refreshGnbConfig.action',
					params={
						 egwCode: vm.egwCode
					};
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("MingLingYiXiaFa")%>',
							type:'success',
						});
						if(vm.gnbConfigActive == 'gnbSectionConfig'){
							vm.$refs.gnbSectionListTable.refresh();
						}else{
							vm.$refs.gnbInterfaceListTable.refresh();
							vm.$refs.gnbRouteListTable.refresh();
						}
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 切片配置 打开新增弹窗
			sectionAdd(){
				var vm = this;
				vm.addSectionDialogShow = true;
			},
			// 切片配置删除 
			delSection(row){
				var vm = this,
					urls="${ctx}/egw/config/setGnbConfig.action",
					params={
						egwCode:vm.egwCode,
					},
					delOtherConfig ={
						sliceList:row.index
					};
				params.delOtherConfig = JSON.stringify(delOtherConfig);
				var delTips = 'Need to restart after deleting the gateway of the monitor service';
				var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueDingShanChuRenWu")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ delTips +'</div>';
				vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					dangerouslyUseHTMLString:true
				}).then(()=>{
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							vm.$refs.gnbSectionListTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})
			},
			// 切片配置新增提交
			addSectionSubmit(){
				var vm = this,
					urls="${ctx}/egw/config/setGnbConfig.action",
					params={
						egwCode:vm.egwCode,
					},
					otherConfig ={
						sliceList:[]
					};
				var sectionItem = Object.assign({},vm.addSectionForm)
				if(!sectionItem.sd){
					delete sectionItem.sd
				}
				otherConfig.sliceList.push(sectionItem);
				params.otherConfig = JSON.stringify(otherConfig);
				vm.$refs["addSectionForm"].validate( valid => {
					if(valid){
						axios.post(urls,stringify(params)).then(res=>{
							var data = res.data;
							if(data["success"]){
								vm.$message({
									message: '<%=rb.getString("MingLingYiXiaFa")%>',
									type:'success',
								});
								vm.closeAddSection();
								vm.$refs.gnbSectionListTable.refresh();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}else{
						return false
					}
				})
			},
			// 切片配置新增取消
			closeAddSection(){
				var vm = this,
					params={
						plmn:'',
						tac:'',
						sst:'',
						sd:'',
					};
				vm.addSectionDialogShow = false;
				Object.assign(vm.addSectionForm,params);
				vm.$refs.addSectionForm.resetFields();
			},
			// PMF 设备连接 打开新增弹窗
			interfaceAdd(){
				var vm = this;
				vm.addInterfaceDialogShow = true;
			},
			// PMF 设备连接 删除事件
			delInterface(row){
				var vm = this,
					urls="${ctx}/egw/config/setGnbConfig.action",
					params={
						egwCode:vm.egwCode,
					},
					delOtherConfig ={
						interfaceList:row.index
					};
				params.delOtherConfig = JSON.stringify(delOtherConfig);
				vm.$confirm('<%=rb.getString("QueDingShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					dangerouslyUseHTMLString:true
				}).then(()=>{
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							vm.$refs.gnbInterfaceListTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})
			},
			// PMF 设备连接 新增提交
			addInterfaceSubmit(){
				var vm = this,
					urls="${ctx}/egw/config/setGnbConfig.action",
					params={
						egwCode:vm.egwCode,
					},
					otherConfig ={
						interfaceList:[]
					};
				otherConfig.interfaceList.push(vm.addInterfaceForm);
				params.otherConfig = JSON.stringify(otherConfig);
				vm.$refs["addInterfaceForm"].validate( valid => {
					if(valid){
						axios.post(urls,stringify(params)).then(res=>{
							var data = res.data;
							if(data["success"]){
								vm.$message({
									message: '<%=rb.getString("MingLingYiXiaFa")%>',
									type:'success',
								});
								vm.closeAddInterface();
								vm.$refs.gnbInterfaceListTable.refresh();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}else{
						return false
					}
				})
			},
			// PMF 设备连接 新增取消
			closeAddInterface(){
				var vm = this,
					params={
						name:'',
						type:'N3a',
						ipAndMask:'',
						mtu:'',
						reassemblySwitch:'on'
					};
				vm.addInterfaceDialogShow = false;
				Object.assign(vm.addInterfaceForm,params);
				vm.$refs.addInterfaceForm.resetFields();
			},
			// PMF路由新增
			routeAdd(){
				var vm = this;
				vm.addRouteDialogShow = true;
			},
			// PMF路由新增 删除事件
			delRoute(row){
				var vm = this,
					urls="${ctx}/egw/config/setGnbConfig.action",
					params={
						egwCode:vm.egwCode,
					},
					delOtherConfig ={
						routeList:row.index
					};
				params.delOtherConfig = JSON.stringify(delOtherConfig);
				vm.$confirm('<%=rb.getString("QueDingShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					dangerouslyUseHTMLString:true
				}).then(()=>{
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							vm.$refs.gnbRouteListTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})
			},
			// PMF路由新增 新增提交
			addRouteSubmit(){
				var vm = this,
					urls="${ctx}/egw/config/setGnbConfig.action",
					params={
						egwCode:vm.egwCode,
					},
					otherConfig ={
						routeList:[]
					};
				var routeItem = Object.assign({},vm.addRouteForm);
				if(!routeItem.device){
					delete routeItem.device
				}
				otherConfig.routeList.push(routeItem);
				params.otherConfig = JSON.stringify(otherConfig);
				vm.$refs["addRouteForm"].validate( valid => {
					if(valid){
						axios.post(urls,stringify(params)).then(res=>{
							var data = res.data;
							if(data["success"]){
								vm.$message({
									message: '<%=rb.getString("MingLingYiXiaFa")%>',
									type:'success',
								});
								vm.closeAddRoute();
								vm.$refs.gnbRouteListTable.refresh();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}else{
						return false
					}
				})
			},
			// PMF路由新增 新增取消
			closeAddRoute(){
				var vm = this,
					params={
						targetIp:'',
						nextHop:'',
						device:'',
					};
				vm.addRouteDialogShow = false;
				Object.assign(vm.addRouteForm,params);
				vm.$refs.addRouteForm.resetFields();
			},
			// enb 基本配置 基站配置 提交
			submitBasicAndEnbList(){
				var vm = this;

				if(vm.enbConfigActive == 'basic'){
                    var errShow = false;
					if(vm.enbPlmnList.length == 0){
						vm.enbPlmnErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						errShow = true;
					}
					if(vm.enbEgwIpList.length == 0){
						vm.ipErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						errShow = true;
					}
					if(vm.enbUpLinkIpList.length == 0){
						vm.enbUpLinkIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						errShow = true;
					}
					if(vm.enbDownLinkIpList.length == 0){
						vm.enbDownLinkIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						errShow = true;
					}
                    if(errShow) return
				}
				
				let params = {},
					enbListInfo=[],
					addHplmnList = [],
					addIpList = [],
					addUplinkAddrList = [],
					addDownlinkAddrList = [],
					basicInfo={};
				if(vm.enbConfigActive == 'basic'){
					vm.enbPlmnList.map((item,index)=>{
						if(!item.configIndex){
							addHplmnList.push(item)
						}
					})
					vm.enbEgwIpList.map((item,index)=>{
						if(!item.configIndex){
							addIpList.push(item)
						}
					})
					vm.enbUpLinkIpList.map((item,index)=>{
						if(!item.configIndex){
							addUplinkAddrList.push(item)
						}
					})
					vm.enbDownLinkIpList.map((item,index)=>{
						if(!item.configIndex){
							addDownlinkAddrList.push(item)
						}
					})
				}
				
				basicInfo.HplmnList = addHplmnList;
				basicInfo.IpList = addIpList;
				basicInfo.UplinkAddrList = addUplinkAddrList;
				basicInfo.DownlinkAddrList = addDownlinkAddrList;
				params.egwCode = vm.egwCode;
				
				params.basicInfo = JSON.stringify(basicInfo);

				if(vm.enbConfigActive == 'basic'){
					params.delPlmnList = vm.delEnbPlmnList.join(',');
					params.delIpList = vm.delIpList.join(',');
					params.delUplinkAddrList = vm.delEnbUplinkAddrList.join(',');
					params.delDownlinkAddrList = vm.delEnbDownlinkAddrList.join(',');
				}else{
					vm.enbIdMmeListTableData.map((item,index)=>{
						if(item.isEdit && item.isEdit == 'true'){
							enbListInfo.push(item)
						}
					})
					enbListInfo.map((item)=>{
						if(item.enbLinkListInfo && item.enbLinkListInfo.length >0){
							item.enbLinkListInfo = item.enbLinkListInfo.filter((items)=>{
								return !items.configIndex
							})
						}
					})
					params.delEnbList = vm.delEnbIdMmeList.join(',');
				}
				params.enbListInfo = JSON.stringify(enbListInfo);
				axios.post("${ctx}/egw/config/setBasicAndENBListConfig.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
							message:'<%=rb.getString("ChengGong")%>',
							type:'success',
						})
						eventBus.$emit("close-setting");
					}else{
						vm.$message.error('<%=rb.getString("ShiBai")%>') 
					}
				})
				
				
			},
			// gnb 基本配置 基站配置 提交
			submitGnbBasicAndEnbList(){
				var vm = this;

				if(vm.gnbConfigActive == 'basic'){
                    var errShow = false;
					if(vm.gnbPlmnList.length == 0){
						vm.gnbPlmnErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						errShow = true;
					}
					if(vm.gnbNgAmfIpList.length == 0){
						vm.ngAmfIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						errShow = true;
					}
					if(vm.gnbUpLinkIpList.length == 0){
						vm.gnbUpLinkIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						errShow = true;
					}
					if(vm.gnbDownLinkIpList.length == 0){
						vm.gnbDownLinkIpErrorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
						errShow = true;
					}
                    if(errShow) return
				}
				
				let params = {},
					enbListInfo=[],
					addHplmnList = [],
					addIpList = [],
					addUplinkAddrList = [],
					addDownlinkAddrList = [],
					basicInfo={};
				if(vm.gnbConfigActive == 'basic'){
					vm.gnbPlmnList.map((item,index)=>{
						if(!item.configIndex){
							addHplmnList.push(item)
						}
					})
					vm.gnbNgAmfIpList.map((item,index)=>{
						if(!item.configIndex){
							addIpList.push(item)
						}
					})
					vm.gnbUpLinkIpList.map((item,index)=>{
						if(!item.configIndex){
							addUplinkAddrList.push(item)
						}
					})
					vm.gnbDownLinkIpList.map((item,index)=>{
						if(!item.configIndex){
							addDownlinkAddrList.push(item)
						}
					})
				}
				
				basicInfo.HplmnList = addHplmnList;
				basicInfo.IpList = addIpList;
				basicInfo.UplinkAddrList = addUplinkAddrList;
				basicInfo.DownlinkAddrList = addDownlinkAddrList;
				params.egwCode = vm.egwCode;
				params.basicInfo = JSON.stringify(basicInfo);

				if(vm.gnbConfigActive == 'basic'){
					params.delPlmnList = vm.delGnbPlmnList.join(',');
					params.delIpList = vm.delNgAmfIpList.join(',');
					params.delUplinkAddrList = vm.delGnbUplinkAddrList.join(',');
					params.delDownlinkAddrList = vm.delGnbDownlinkAddrList.join(',');
				}else{
					if(vm.gnbConfigIdLengthForm.GNB_idLength != vm.oldGnbConfigIdLengthForm.GNB_idLength){
						params.leftSize = vm.gnbConfigIdLengthForm.GNB_idLength
					}
					if(vm.gnbConfigIdLengthForm.AMF_idLength != vm.oldGnbConfigIdLengthForm.AMF_idLength){
						params.rightSize = vm.gnbConfigIdLengthForm.AMF_idLength
					}
					vm.gnbIdMmeListTableData.map((item,index)=>{
						if(item.isEdit && item.isEdit == 'true'){
							enbListInfo.push(item)
						}
					})
					enbListInfo.map((item)=>{
						if(item.enbLinkListInfo && item.enbLinkListInfo.length >0){
							item.enbLinkListInfo = item.enbLinkListInfo.filter((items)=>{
								return !items.configIndex
							})
						}
					})
					params.delEnbList = vm.delGnbIdMmeList.join(',');
				}
				params.enbListInfo = JSON.stringify(enbListInfo);
				axios.post("${ctx}/egw/config/setGnbConfig.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
							message:'<%=rb.getString("ChengGong")%>',
							type:'success',
						})
						eventBus.$emit("close-setting");
					}else{
						vm.$message.error('<%=rb.getString("ShiBai")%>') 
					}
				})
				
			},
			// 判断是否为空
			isNull(val){
				if(val==undefined || val == null || val =="") return true;
				else return false;
			},
			// 关闭配置页面
			closeLinkSetting(){
				var vm = this;
				eventBus.$emit("close-setting");
			},
			// 认证用户表格 模糊查询
			queryVerifiedAccount(val){
				var vm = this;
				vm.queryVerifiedAccountParams.searchText= val;
			},
			// 认证用户表格选择
			verifiedAccountSelect(selection){
				var vm = this;
				vm.verifiedAccountSelection = selection
			},
			//  打开操作菜单
			verifiedAccountOptClick(row,evt){
				var vm =this;
				vm.verifiedAccountRowData = row;
				vm.verifiedAccountMenus = [
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_EGW hidden",code:'mod'},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_EGW hidden",code:'del'},
					{label:'<%=rb.getString("YiDongDaoHeiMingDan")%>',cls:"el-icon el-icon-move CODE_EGW hidden",code:'move'},
				];
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.verifiedAccountMenu.show(evt)
				})
			},
			/**
			* 菜单点击事件
			* @param ev{object}   行数据
			*/ 
			verifiedAccountMenuClick(evt){
				var vm =this;
				var codes = {
					mod:this.verifiedAccountModify,	// 修改
					del:this.verifiedAccountDel,	// 删除
					move:this.verifiedAccountMove   // 移动到黑名单
				};
				if(codes[evt.code]){
					codes[evt.code]('alone');
				}
			},
			// 菜单关闭
			hideverifiedAccountMenus(){
				this.$refs.verifiedAccountMenu.hide();
			},
			// 认证用户 修改
			verifiedAccountModify(){
				var vm = this;
				vm.optType = 'edit';
				vm.blockListBoxShow = false;
				vm.addVerifiedAccountBoxShow = true;
				Object.assign(vm.addVerifiedAccountForm,vm.verifiedAccountRowData);
			},
			// 认证用户 删除
			verifiedAccountDel(type){
				var vm = this,
					urls = '${ctx}/egw/ipsec/delUserList.action',
					params={
						egwCode:vm.egwCode,
                       	indexList:''
					};
				if(type == 'alone'){
					params.indexList = vm.verifiedAccountRowData.index
				}else{
					var indexList=[];
					vm.verifiedAccountSelection.map((item,index) => {
						indexList.push(item.index)
					})
					params.indexList = indexList.join(',');
				}
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning'
				}).then(()=>{
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							vm.$refs.verifiedAccountTable.refresh();
							vm.$refs.blockListTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})
			},
			// 认证用户 移动到黑名单
			verifiedAccountMove(type){
				var vm = this,
					urls = '${ctx}/egw/ipsec/userMoveToBlackList.action',
					params={
                    	egwCode:vm.egwCode,
                       	indexList:'',
						imsiList:''
					};
				if(type == 'alone'){
					params.imsiList = vm.verifiedAccountRowData.imsi
					params.indexList = vm.verifiedAccountRowData.index
				}else{
					var indexList=[],imsiList=[];
					vm.verifiedAccountSelection.map((item,index) => {
						indexList.push(item.index);
						imsiList.push(item.imsi);
					})
					params.indexList = indexList.join(',');
					params.imsiList = imsiList.join(',');
				}
				vm.$confirm('<%=rb.getString("QueRenJiaRuHeiMingDan")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning'
				}).then(()=>{
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							vm.$refs.verifiedAccountTable.refresh();
							vm.$refs.blockListTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})
				
			},
			verifiedAccountImportClick(type){
				var vm =this,
					code={
						'user' : '.csv',
						'block' : '.csv',
						'CA' : '.crt',
						'IPsec' : '.crt',
						'Key' : '.key'
					};
				vm.importType = type;
				vm.acceptType = code[type];
				if(vm.importType == 'user' || vm.importType == 'block'){
					vm.verifiedAccountImportForm.errMassage = 'Only .csv is supported'
				}else if(vm.importType == 'CA' || vm.importType == 'IPsec'){
					vm.verifiedAccountImportForm.errMassage = 'Only .crt is supported'
				}else if(vm.importType == 'Key'){
					vm.verifiedAccountImportForm.errMassage = 'Only .key is supported'
				}
				vm.verifiedAccountImportShow = true;
			},
			// 选择文件
			verifiedAccountImportFileSelect(){  
				var vm =this;
				vm.$refs.verifiedAccountUpload.clearFiles();
				vm.$refs['file_up'].click();
			},
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			verifiedAccountBeforeUpload(file){
				var vm = this, 
					urls = '', 
					fileName = file.name,
					fileSize = file.size,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					},
					code={
						'user' : 'verifiedAccountTable',
						'block' : 'blockListTable',
						'CA':'CAcertTable',
						'IPsec':'IPsecCertTable',
						'Key':'IPsecPrivateKeyTable'
					};
				if(vm.importType == 'user'){
					urls = '${ctx}/egw/ipsec/uploadUserFile.action';
				}else if(vm.importType == 'block'){
					urls = '${ctx}/egw/ipsec/uploadBlackListFile.action';
				}else if(vm.importType == 'CA'){
					urls = '${ctx}/egw/ipsec/config/uploadCaCertFile.action';
				}else if(vm.importType == 'IPsec'){
					urls = '${ctx}/egw/ipsec/config/uploadCertFile.action';
				}else if(vm.importType == 'Key'){
					urls = '${ctx}/egw/ipsec/config/uploadPrivateKeyFile.action';
				}
				fd.append('egwCode',vm.egwCode); 
				fd.append('uploadFile',file); //文件流
				if(vm.importType != 'user' && vm.importType != 'block'){
					fd.append('serial_number',vm.serial_number); 
					fd.append('file_size',fileSize); 
				}
				axios.post(urls,fd,config).then(function(res){
					var data = res.data;
					if(data["success"]){	
						if(vm.importType != 'user' && vm.importType != 'block'){
							vm.getFileListData();
							vm.$message.success('<%=rb.getString("ChengGong")%>');
						}else{
							vm.$message.success('<%=rb.getString("MingLingYiXiaFa")%>');
						}
						vm.$refs[code[vm.importType]].refresh();
						vm.closeImportParams();						
					}else{
						vm.$message.error(data["message"]);
						vm.closeFileSelect();
					} 
				})			
				return false;
			},
			//导入
			verifiedAccountCheckFile(res,file){  
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}				
					vm.closeFileSelect();
					vm.verifiedAccountImportShow = false;
				}else{
					vm.$message({
						type: 'error',
						message: res.message
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.verifiedAccountUpload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			verifiedAccountFileChange(file,fileList){ 
				var vm = this,typeFlag;
				if(vm.importType == 'user' || vm.importType == 'block'){
					 typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.csv';
				}else if(vm.importType == 'CA' || vm.importType == 'IPsec'){
					 typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.crt';
				}else if(vm.importType == 'Key'){
					 typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.key';
				}
				
				vm.verifiedAccountImportForm.errMassageShow = !typeFlag;
				if(typeFlag){
					vm.verifiedAccountImportFileName = file.name;
					vm.verifiedAccountImportFileParams.FileName = file.name;
				}else {
					vm.verifiedAccountImportFileName = '';
				}
				if(vm.importType == 'user' || vm.importType == 'block'){
					vm.verifiedAccountImportForm.errMassage = 'Only .csv is supported'
				}else if(vm.importType == 'CA' || vm.importType == 'IPsec'){
					vm.verifiedAccountImportForm.errMassage = 'Only .crt is supported'
				}else if(vm.importType == 'Key'){
					vm.verifiedAccountImportForm.errMassage = 'Only .key is supported'
				}
			},
			// 移除导入文件
			closeFileSelect(){
				var vm = this;
				vm.verifiedAccountImportFileName = '';
				vm.verifiedAccountImportFileParams.FileName = '';			
				vm.$refs.verifiedAccountUpload.clearFiles();
			},
			// 关闭导入弹出框
			closeImportParams(){
				var vm = this,
					params = {
						errMassage:'Only .csv is supported',
						errMassageShow:false
					};
				Object.assign(vm.verifiedAccountImportForm,params);		
				vm.verifiedAccountImportFileName = '';
				vm.verifiedAccountImportFileParams.FileName = '';
				vm.$refs.verifiedAccountUpload.clearFiles();
				vm.verifiedAccountImportShow = false;
			},
			// 导出用户列表
			exportVerifiedAccount(){
				var vm = this,
					urls = "${ctx}/egw/ipsec/exportUserList.action",
					params = {
						egw_code: vm.egwCode,
						timeZone:timeZone,
						searchText: vm.queryVerifiedAccountParams.searchText
					};
				exportByForm(urls,params);
			},
			//下载模板
			verifiedAccountExportTemplate(){
				var vm = this,
					urls = '';
				if(vm.importType == 'user'){
					urls = '${ctx}/egw/ipsec/downloadImportWCGAAAUserTemplate.action';
				}else{
					urls = '${ctx}/egw/ipsec/downloadImportWCGUsimBlackListTemplate.action';
				}
				exportByForm(urls, {});
			},
			/*确定导入*/
	        verifiedAccountUploadParams() {
				var vm = this;
				if(vm.verifiedAccountImportFileParams.FileName){
					vm.$refs.verifiedAccountUpload.submit();
				}else{
					vm.verifiedAccountImportForm.errMassageShow = true;
					vm.verifiedAccountImportForm.errMassage = '<%=rb.getString("QingXianXuanZeWenJian")%>'
				}
			},
			// 新增用户认证事件
			addVerifiedAccountClick(){
				var vm = this;
				vm.optType = 'add';
				vm.addVerifiedAccountClose();
				vm.blockListBoxShow = false;
				vm.addVerifiedAccountBoxShow = true
			},
			// 新增用户认证 提交
			addVerifiedAccountSubmit(){
				var vm =this,
					urls = '',
					params={
						egwCode:vm.egwCode,
						imsi:vm.addVerifiedAccountForm.imsi,
						key:vm.addVerifiedAccountForm.key,
						opc:vm.addVerifiedAccountForm.opc
					};
				if(vm.optType == 'add'){
					urls = '${ctx}/egw/ipsec/addUser.action';
				}else{
					params.index = vm.verifiedAccountRowData.index;
					urls = '${ctx}/egw/ipsec/updateUser.action';
				}
				vm.$refs.addVerifiedAccountForm.validate(function(valid){
					if(valid){
						axios.post(urls,stringify(params)).then(function(response){
							var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
		    						message: '<%=rb.getString("MingLingYiXiaFa")%>',
		    						type:'success',
		    					})
								vm.$refs.verifiedAccountTable.refresh();
								vm.addVerifiedAccountClose();
		    				}else{
		    					vm.$message.error(data["message"])
		    				}
						}).catch(function(error){})
					}
				})

			},
			// 关闭新增用户认证
			addVerifiedAccountClose(){
				var vm = this,
					params={
						imsi:'',
						key:'',
						opc:''
					};
				vm.addVerifiedAccountBoxShow = false;
				Object.assign(vm.addVerifiedAccountForm,params);
				vm.$refs.addVerifiedAccountForm.resetFields();
			},
			// 打开黑名单列表
			blockListClick(){
				var vm = this;
				vm.addVerifiedAccountBoxShow = false;
				vm.blockListBoxShow = true;
			},
			// USIM黑名单 模糊查询
			queryBlockList(val){
				var vm = this;
				vm.queryBlockListParams.searchText= val;
			},
			// 新增 USIM黑名单信息
			addBlockListClick(){
				var vm =this;
				vm.addBlockListDialogShow = true;
			},
			// 新增 USIM黑名单提交
			addBlockListSubmit(){
				var vm =this,
					urls = '${ctx}/egw/ipsec/addBlackList.action',
					params={
						egwCode:vm.egwCode,
						imsi:vm.addBlockListForm.imsi,
					};
				vm.$refs.addBlockListForm.validate(function(valid){
					if(valid){
						axios.post(urls,stringify(params)).then(function(response){
							var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
		    						message: '<%=rb.getString("MingLingYiXiaFa")%>',
		    						type:'success',
		    					})
								vm.$refs.blockListTable.refresh();
								vm.closeAddBlockList();
		    				}else{
		    					vm.$message.error(data["message"])
		    				}
						}).catch(function(error){})
					}
				})
			},
			// 关闭新增 USIM黑名单
			closeAddBlockList(){
				var vm = this,
					params={
						imsi:'',
					};
				vm.addBlockListDialogShow = false;
				Object.assign(vm.addBlockListForm,params);
				vm.$refs.addBlockListForm.resetFields();
			},
			// 黑名单删除
			blockListDel(row,type){
				var vm = this,
					urls = '${ctx}/egw/ipsec/delBlackList.action',
					params={
                       egwCode:vm.egwCode,
					   indexList:''
					};
				if(type == 'del'){
					var confirmHint = '<%=rb.getString("QueRenShanChu")%>';
					params.indexList = row.index
				}else{
					var indexList = [],
						confirmHint = '<%=rb.getString("ShanChuDangQianYeShuJu")%>';
					var blockList = vm.$refs.blockListTable.getData();
					blockList.map((item)=>{
						indexList.push(item.index)
					})
					params.indexList = indexList.join(',');
				}
				vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning'
				}).then(()=>{
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})
			},
			// 导出USIM黑名单
			exportBlockList(){
				var vm = this,
					urls = "${ctx}/egw/ipsec/exportUsimBlackList.action",
					params = {
						egw_code: vm.egwCode,
						timeZone:timeZone,
						searchText: vm.queryBlockListParams.searchText
					};
				exportByForm(urls,params);
			},
			// 判断数据有无变化
			isObjectChange(obj1,obj2){
				var str1 = JSON.stringify(obj1);
				var str2 = JSON.stringify(obj2);
				if(str1 != str2){
					return true
				}
				return false
			},
			// IPsec配置 新增
			addIPsecConfigClick(){
				var vm = this;
				vm.optType = 'add';
				axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
					serial_number:vm.serial_number,
					file_type:'ipseccert'
				})).then(function(response){
					var data = response.data;
					if(data && data.length > 0){
						var leftCertData = [];
						data.map((item)=>{
							if(item.file_status != '0'){
								leftCertData.push(item)
							}
						})
						vm.leftCertData = leftCertData;
					}
				});
				vm.addIPsecConfigShow = true;
				
				initForm(vm.$refs.addIPsecConfigForm)
			},
			// IPsec配置表格 模糊查询
			queryIPsecConfig(val){
				var vm = this;
				vm.queryIPsecConfigParams.searchText= val;
			},
			
			//  打开IPsec配置操作菜单
			IPsecConfigOptClick(row,evt){
				var vm =this;
				vm.IPsecConfigRowData = row;
				vm.IPsecConfigMenus = [
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_EGW hidden",code:'mod'},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_EGW hidden",code:'del'},
				];
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.IPsecConfigMenu.show(evt)
				})
			},
			/**
			* IPsec配置 菜单点击事件
			* @param ev{object}   行数据
			*/ 
			IPsecConfigMenuClick(evt){
				var vm =this;
				var codes = {
					mod:this.IPsecConfigModify,	// 修改
					del:this.IPsecConfigDel,	// 删除
				};
				if(codes[evt.code]){
					codes[evt.code](vm.IPsecConfigRowData);
				}
			},
			//IPsec配置  菜单关闭
			hideIPsecConfigMenus(){
				this.$refs.IPsecConfigMenu.hide();
			},
			// IPsec配置 修改
			IPsecConfigModify(row){
				var vm = this,
					params={
						egw_code:vm.egwCode,
						index:row.index
					};
				vm.optType = 'edit';
				axios.post("${ctx}/egw/ipsec/config/getAllCertFile.action",stringify({
					serial_number:vm.serial_number,
					file_type:'ipseccert'
				})).then(function(response){
					var data = response.data;
					if(data && data.length > 0){
						var leftCertData = [];
						data.map((item)=>{
							if(item.file_status != '0'){
								leftCertData.push(item)
							}
						})
						vm.leftCertData = leftCertData;
					}
					axios.post('${ctx}/egw/ipsec/config/getConfigInfo.action',stringify(params)).then(function(response){
					var data = response.data;
						if(data){
							Object.assign(vm.addIPsecConfigForm,data);
							var isExist = false;
							vm.leftCertData.map((item)=>{
								if(item.file_name == vm.addIPsecConfigForm.left_cert){
									isExist = true;
								}
							})
							if(!isExist){
								vm.addIPsecConfigForm.left_cert = '';
							}
							initForm(vm.$refs.addIPsecConfigForm);
							vm.addIPsecConfigShow = true;
						} 
					}) 
				});
				
			},
			// IPsec配置 删除
			IPsecConfigDel(row){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/delConfig.action',
					params={
						egw_code:vm.egwCode,
						index:row.index
					};
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning'
				}).then(()=>{
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})
			},
			// IPsec配置 开关事件
			IPsecConfigEnableChange(row){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/switchEnable.action'
					params={
						egw_code:vm.egwCode,
						index:row.index,
						switch_enable:row.switch_enable == '1' ? '0' : '1'
					};
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("MingLingYiXiaFa")%>',
							type:'success',
						});
						vm.$refs.IPsecConfigTable.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 新增 IPsec配置 提交
			addIPsecConfigSubmit(){
				var vm = this,
					urls = ''
					params={
						egw_code:vm.egwCode
					};
				if(vm.optType == 'edit'){
					params.index = vm.IPsecConfigRowData.index
					urls = '${ctx}/egw/ipsec/config/updateConfig.action';
				}else{
					urls = '${ctx}/egw/ipsec/config/addConfig.action';
				}
				Object.assign(params,vm.addIPsecConfigForm);
				vm.$refs.addIPsecConfigForm.validate(function(valid){
					if(valid){
						axios.post(urls,stringify(params)).then(res=>{
							var data = res.data;
							if(data["success"]){
								vm.$message({
									message: '<%=rb.getString("MingLingYiXiaFa")%>',
									type:'success',
								});
								vm.addIPsecConfigShow = false;
								vm.resetIPsecConfigForm();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}else{
						return false
					}
				})
			},
			// 新增 IPsec配置 取消
			closeAddIPsecConfig(tab){
				var vm = this;

				if(isFormChanged(vm.$refs.addIPsecConfigForm)){
					vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						vm.addIPsecConfigShow = false;
						vm.resetIPsecConfigForm();
					}).catch(() => {
						
					})
				}else{
					vm.addIPsecConfigShow = false;
					vm.resetIPsecConfigForm();
				}
			},
			// 新增 IPsec配置 参数重置
			resetIPsecConfigForm(){
				var vm = this,
					params = {
						group_name:'',
						switch_enable:'1',
						secret_key:'',
						fragmentation:'yes',

						left_ip:'',
						left_id:'',
						left_auth:'psk',
						left_source:'',
						left_subnet:'',
						left_cert:'',

						right_ip:'',
						right_id:'',
						right_auth:'psk',
						right_source:'',
						right_subnet:'',
						right_secret_key:'',

						ike_encryption:'aes128',
						ike_dh_group:'modp768',
						ike_authentication:'sha1',
						esp_encryption:'aes128',
						esp_dh_group:'modp768',
						esp_authentication:'sha1',
						key_life:'',
						ike_life_time:'',
						rekey_margin:'',
						dpd_action:'none',
						dpd_delay:''
					};
				
				Object.assign(vm.addIPsecConfigForm,params)
				vm.$refs.addIPsecConfigForm.clearValidate();
			},
			// IPsec证书导入 打开弹窗
			IPsecCertImportClick(){
				var vm =this;
			},
			// 查看证书表格
			viewCertClick(type){
				var vm = this;
				if(vm.certTableType == type)return
				vm.certTableBoxShow = true;
				if(vm.certTableType){
					vm.$refs[vm.tableCode[vm.certTableType]].clearSelection();
				}
				vm.certTableType = type;
			},
			// CA证书表格 模糊搜索
			queryCAcertList(val){
				var vm = this;
				vm.queryCAcertParams.searchText = val;
			},
			// IPsec证书表格 模糊搜索
			queryIPsecCertList(val){
				var vm = this;
				vm.queryIPsecCertParams.searchText = val;
			},
			// IPsec私钥表格 模糊搜索
			queryIPsecPrivateKeyList(){
				var vm = this;
				vm.queryIPsecPrivateKeyParams.searchText = val;
			},
			// 从WCG获取证书
			getCert(row,type){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/getFile.action'
					params={
						egw_code:vm.egwCode,
						serial_number:vm.serial_number,
					};
				if(type == 'CA'){
					params.ca_cert = row.file_name;
				}else if(type == 'IPsec'){
					params.ipsec_cert = row.file_name;
				}else if(type == 'Key'){
					params.private_key = row.file_name;
				}
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("MingLingYiXiaFa")%>',
							type:'success',
						});
						vm.$refs[vm.tableCode[type]].refresh();
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 证书删除 delType: del-单个删除 clear-删除当前页   fileType :  CA  IPsec Key
			certDel(row,delType,fileType){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/delFile.action',
					code = {
						'CA':'cacert',
						'IPsec':'ipseccert',
						'Key':'privatekey'
					},
					params={
                       egw_code:vm.egwCode,
					   serial_number:vm.serial_number,
					   file_name:'',
					   file_type:code[fileType]
					};
				if(delType == 'del'){
					if(row.file_status == '0'){
						var confirmHint = '<%=rb.getString("QueDingShanChuWenJian")%>';
					}else{
						var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueDingShanChuWenJian")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("WCG_ShanChuWenJianTiShi")%>' +'</div>';
					}
					params.file_name = row.file_name
				}else{
					
					var fileNameList = [];
					var fileList = vm.$refs[vm.tableCode[fileType]].getChecked();
					if(fileList.length == 0)return
					var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueDingShanChuWenJian")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("WCG_PiLiangShanChuWenJianTiShi")%>' +'</div>';
					params.file_name = fileList.join(';');
				}
				vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					dangerouslyUseHTMLString:true
				}).then(()=>{
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type:'success',
							});
							if(delType == 'del'){
								vm.$refs[vm.tableCode[fileType]].toggleRowSelection(row,false);
							}else{
								vm.$refs[vm.tableCode[fileType]].clearSelection();
							}
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})

			},
			// 证书下载
			certDownload(row,type){
				var vm = this,
					urls = '',
					params = {
						serial_number:vm.serial_number,
						egw_code:vm.egwCode,
						file_name:row.file_name
					};
				if(type == 'CA'){
					urls = '${ctx}/egw/ipsec/config/downloadCertFile.action';
				}else if(type == 'IPsec'){
					urls = '${ctx}/egw/ipsec/config/downloadCaCertFile.action';
				}else if(type == 'Key'){
					urls = '${ctx}/egw/ipsec/config/downloadPrivateKeyFile.action';
				}
				exportByForm(urls,params);
			},
			//同WCG获取证书信息
			refreshCertInfo(){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/refreshCertInfo.action',
					params = {
						egw_code:vm.egwCode,
						serial_number:vm.serial_number,
					};
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("MingLingYiXiaFa")%>',
							type:'success',
						});
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 刷新证书信息
			getCertInfo(){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/getCertInfo.action',
					params = {
						egw_code:vm.egwCode,
						timeZone:timeZone
					};
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					['cert_manage','cmp_server_address','country','refresh_time','refresh_status'].map((item)=>{
						if(data[item]){
							vm.IPsecCertForm[item] = data[item];
						}
					})
				})
			},
			// CA证书表格选择
			CAcertSelect(selection){
				var vm = this;

				vm.CAcertSelection = selection;
			},
			// IPsec证书表格选择
			IPsecCertSelect(selection){
				var vm = this;

				vm.IPsecCertSelection = selection;
			},
			//  key 文件表格选择
			KeyCertSelect(selection){
				var vm = this;

				vm.KeyCertSelection = selection;
			},
			// 下发文件
			issueCertFile(){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/sendFile.action',
					params = {
						egw_code:vm.egwCode,
					   	serial_number:vm.serial_number,
						ca_cert: vm.IPsecCertForm.ca_cert,
						ipsec_cert:vm.IPsecCertForm.ipsec_cert,
						private_key:vm.IPsecCertForm.private_key,
					};
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("MingLingYiXiaFa")%>',
							type:'success',
						});
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 更新证书
			updateCertFile(){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/updateCert.action',
					params = {
						egw_code:vm.egwCode,
					   	serial_number:vm.serial_number,
						ca_cert: vm.IPsecCertForm.ca_cert,
						ipsec_cert:vm.IPsecCertForm.ipsec_cert,
						private_key:vm.IPsecCertForm.private_key,
					};
				if(vm.issueAndUpdateCertBtnDis)return
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("MingLingYiXiaFa")%>',
							type:'success',
						});
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 保存证书
			saveCertInfo(){
				var vm = this,
					urls = '${ctx}/egw/ipsec/config/saveCertInfo.action',
					params = {
						egw_code:vm.egwCode,
					   	serial_number:vm.serial_number,
						cert_manage: vm.IPsecCertForm.cert_manage,
						cmp_server_address:vm.IPsecCertForm.cmp_server_address,
						country:vm.IPsecCertForm.country,
					};
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("MingLingYiXiaFa")%>',
							type:'success',
						});
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 左侧文件列表关闭
			certTableBoxClose(){
				var vm = this;

				if(vm.certTableType){
					vm.$refs[vm.tableCode[vm.certTableType]].clearSelection();
				}
				vm.certTableBoxShow = false
				vm.certTableType = '';
			},
			//校验IP
			isValidIP(ip){
				var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
				return reg.test(ip);     
			},
			//Ipv6校验 
			isIPv6(str){ 
				var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
				return reg.test(str);
			},
			// 验证输入的是否是数字
			isNumeric(str) {
				if(str.length==0){
					return false;
				}
				for(var i=0;i<str.length;i++){
					if(str.charAt(i)<"0" || str.charAt(i)>"9"){
						return false;
					}
				}
				return true;  
			},
			// 告警值与恢复值 相互校验
			commonConfigValid(val){
				var vm = this;
				vm.$refs.commonConfigForm.validateField(val);
			},
			submitCommonConfig(){
				var vm = this,
					urls = '${ctx}/egw/config/setCommonConfig.action',
					params = {
						egwCode:vm.egwCode
					},
					isChanged = isFormChanged(vm.$refs.commonConfigForm);
				if(!isChanged){
					showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
					return;
				}
				vm.$refs.commonConfigForm.fields.map(function(field){
					if(Array.isArray(field.fieldValue)){
						
						var vList = field.fieldValue.map(function(item){return item}),
							oList = (field.reinitialValue||[]).map(function(item){return item}),
							val = JSON.stringify(vList.sort()),
							orVal = JSON.stringify(oList.sort());

						if(val != orVal) {
							var editList=[],subList=[];
							if(val != orVal) {
                                editList.push(field.prop);
                            }
						};
					}else{
						if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
	                            
						}else if(field.fieldValue != field.reinitialValue) {
							params[field.prop] = field.fieldValue;
						};
					}
				});
				if(params.egwName){
					params.egwName = encodeURIComponent(params.egwName);
				}
				if(params.egwDescription){
					params.egwDescription = encodeURIComponent(params.egwDescription);
				}
				vm.$refs["commonConfigForm"].validate( valid => {
					if(valid){
						axios.post(urls,stringify(params)).then(res=>{
							var data = res.data;
							if(data["success"]){
								vm.$message({
									message:'<%=rb.getString("ChengGong")%>',
									type:'success',
								})
								eventBus.$emit("close-setting");
							}else{
								vm.$message.error(data["message"])
							}
						})
					}else{
						return false
					}
				}) 
				
			},
		},
		created(){},
		mounted(){
			var vm =this;
			eventBus.$off('setting-init').$on('setting-init',this.init);
			eventBus.$off('edit-enbList').$on('edit-enbList',this.editEnbList);
		}
	});
</script>