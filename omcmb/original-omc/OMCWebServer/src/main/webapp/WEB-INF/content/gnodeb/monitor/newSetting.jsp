<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#gnbQuickSettingPage{
		background-color: #FFFFFF;
	}
	#gnbQuickSettingPage .logHeader{
		height: 50px;
	}
	#gnbQuickSettingPage .el-form-item .el-form-item__error{
		padding-top:0px;
		top: unset;
	}
	#gnbQuickSettingPage .headTitleBox{
		height: 50px;
		width: 100%;
		position: relative;
		border-bottom: 1px solid #E9E9E9;
		display: flex;
		align-items: center;
		font-size: 14px;
		font-weight: bold;
		padding-left: 20px;
	}
	#gnbQuickSettingPage .logContent{
		height: calc(100% - 50px) !important;
		padding: 20px 20px 0px 20px;
	}
	#gnbQuickSettingPage .cellTitleCls{
		height: 24px;
		display: flex;
		align-items: center;
	}
	#gnbQuickSettingPage .addCellBoxCls{
		height: 24px;
		width: 80px;
		box-sizing: border-box;
		border: 1px solid var(--main-color);
		background-color: rgba(var(--main-color-rgba1),0.08);
		border-radius: 2px;
		display: flex;
		align-items: center;
		cursor: pointer;
	}
	#gnbQuickSettingPage .cellNumListCls{
		display: flex;
		margin-left: 50px;
		padding-top: 5px;
	}
	#gnbQuickSettingPage .cellItemCls{
		height: 24px;
		width: 60px;
		box-sizing: border-box;
		border: 1px solid var(--main-color);
		background-color: rgba(var(--main-color-rgba1),0.08);
		border-radius: 2px;
		margin-right: 10px;
		display: flex;
		align-items: center;
	}
	#gnbQuickSettingPage .cellItemCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#gnbQuickSettingPage .cellItemCls .el-icon::before{
		font-size: 12px;
	}
	#gnbQuickSettingPage .cellContentBoxCls{
		padding-top: 20px;
		height: calc(100% - 120px) !important;
	}
	#gnbQuickSettingPage .cellContentBoxCls .el-tabs--top{
		height: 100%;
	}
	#gnbQuickSettingPage .cellContentBoxCls .el-tabs__item{
		font-size: 14px;
		font-weight: 500;
	}
	#gnbQuickSettingPage .cellContentBoxCls .el-tabs--card>.el-tabs__header{
		width: 96%;
	}
	#gnbQuickSettingPage .cellItemBoxCls{
		overflow: auto;
		height: 100%;
		display: flex;
	}
	#gnbQuickSettingPage .cellItemLeftCls{
		width: 120px;
		height: 100%;
		background-color: #FFFFFF;
		border-right: 1px solid #E9E9E9;
		box-sizing: border-box;
	}
	#gnbQuickSettingPage .cellItemLeftCls>div{
		height: 30px;
		width: 100%;
		line-height: 30px;
		border-bottom: 1px solid #E9E9E9;
		box-sizing: border-box;
		padding-left: 10px;
		cursor: pointer;
	}
	#gnbQuickSettingPage .cellItemLeftCls .tabsActiveCls{
		background-color: rgba(var(--main-color-rgba1),0.08);
		color:var(--main-color);
	}
	#gnbQuickSettingPage .cellItemRightCls{
		flex: 1;
		overflow: auto;
	}
	#gnbQuickSettingPage .cellItemRightCls>div{
		height: calc(100% - 30px) !important;
		padding: 30px 0px 0px 20px;
	}
	#gnbQuickSettingPage .el-collapse-item__header{
		border-bottom:1px solid #fff;
	}
	#gnbQuickSettingPage .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:0px;
	}
	#gnbQuickSettingPage .el-collapse-item{
		position:relative;
	}
	#gnbQuickSettingPage .el-collapse-item__header .el-icon-arrow-right{
		font-size:16px;
	}
	#gnbQuickSettingPage .el-collapse-item__header .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#gnbQuickSettingPage .el-collapse-item__header .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	#gnbQuickSettingPage .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#gnbQuickSettingPage .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
	}
	#gnbQuickSettingPage .el-collapse-item__wrap{
		border-bottom:1px solid #fff;
	}
	#gnbQuickSettingPage .el-collapse-item__header{
		max-width:600px;
	}
	#gnbQuickSettingPage .rightContentCls .itemTitleCls{
		height: 60px;
		display: flex;
		align-items: center;
	}
	#gnbQuickSettingPage .rightContentCls .itemTitleCls >div:first-child{
		height: 6px;
		width: 6px;
		background-color: #333333;
		border-radius: 50%;
	}
	#gnbQuickSettingPage .rightContentCls .itemTitleCls >div:last-child{
		font-size: 14px;
		font-weight: bold;
		color: #333333;
		margin-left: 10px;
	}
	#gnbQuickSettingPage .rightContentCls .contentTableTitle{
		display: flex;
		justify-content: space-between;
		width: 90%;
		margin-left: 16px;
		
	}
	#gnbQuickSettingPage .rightContentCls .contentTableTitle .el-icon::before{
		font-size: 20px;
	}
	#gnbQuickSettingPage .cellTableBoxCls{
		height: 200px;
		width: 90%;
		margin-left: 16px;
	}
	#gnbQuickSettingPage .labelIconCls .el-icon::before{
		color: #666666;
		font-size: 12px;
	}
	#gnbQuickSettingPage .el-form-item {
		margin-bottom: 20px;
	}
	#gnbQuickSettingPage .el-form-item__content{
		white-space: nowrap;
	}
	#gnbQuickSettingPage .cellTabItem .el-form-item .el-select>.el-input{
		width: 70px;
	}
	#gnbQuickSettingPage .prefix-label {
		display: inline-block;
		background: #F5F5F5;
		border: 1px solid #DCDFE6;
		border-left: none;
		margin-left: -5px;
		width: 40px;
		text-align: center;
		height: 24px;
		line-height: 24px;
	}
	#gnbQuickSettingPage .el-form-item__label{
		font-size: 12px;
	}
	#gnbQuickSettingPage .rightContentCls .el-date-editor .el-range__close-icon{
		line-height: 20px;
	}
	#gnbQuickSettingPage .el-tabs__content{
		width: 96%;
		border: 1px solid #E9E9E9;
		border-top: none;
	}
	#gnbQuickSettingPage .mostNumberCls{
		color: #999999;
		margin-left: 10px;
	}
	#gnbQuickSettingPage .errorBoxCls{
		color:red;
		font-size:10px;
	}
	#gnbQuickSettingPage .itemListBoxCls{
		padding-left: 160px;
	}
	#gnbQuickSettingPage .itemCls{
		height: 24px;
		display: inline-block;
		line-height: 24px;
		border: 1px solid #4D84FF;
		background: #F2F6FF;
		box-sizing: border-box;
		padding: 0px 10px;
		margin-right: 10px;
		margin-bottom: 10px;
		border-radius: 2px;
	}
	#gnbQuickSettingPage .itemListBoxCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#gnbQuickSettingPage .disabledIconBox .el-icon::before{
		color: #e9e9e9;
	}
	#gnbQuickSettingPage .remoteAddressBoxCls .el-input__suffix{
		height: 26px;
		display: flex;
		align-items: center;
		padding-top:7px;
	}
	#gnbQuickSettingPage .syncSettingsInfo{
		display:flex;
		margin-left:16px;
		flex-wrap: wrap;
		width:90%;
		border: 1px solid #E9E9E9;
		padding: 20px;
		margin-bottom: 20px;
	}
	#gnbQuickSettingPage .syncSettingsInfo>div{
		width: 45%;
		white-space: nowrap;
		margin-bottom: 10px;
	}
	#gnbQuickSettingPage .syncSettingsInfo>div span:first-child{
		display: inline-block;
		width: 140px;
	}
	#gnbQuickSettingPage .footer {
		width: 100%;
		background: #FFFFFF;
		border-top: #EEEEEE solid 1px;
		border-right: none;
		border-left: none;
		border-bottom: none;
		position: absolute;
		bottom: 0px;
		left: 0px;
		height: 60px;
		line-height: 60px;
		display: flex;
		align-items: center;
		box-sizing: border-box;
	}
	
	#gnbQuickSettingPage .footer .lnkbuttonGroup {
		margin-left: 48px
	}
	.el-ctable .hidden-row{
		display: none;
	}
	.el-input-group__append{
		border-radius:0px;
		border-right:none;
	}
	.validate-item .el-input__inner{
		width:150px;
	}
	.validate-item .el-input-group__append{
		border:none;
		background:none;
	}
	.validate-item .el-form-item__error{
		display:none;
	}
	.is-error .el-input-group__append{
		color:#FA5555;
	}
	.active_btn {
		animation: rotateBgd 2s linear infinite;
		-moz-animation: rotateBgd 2s linear infinite;
		/* Firefox */
		-webkit-animation: rotateBgd 2s linear infinite;
		/* Safari 和 Chrome */
		-o-animation: rotateBgd 2s linear infinite;
	}
	
	@keyframes rotateBgd {
		from {
			transform: rotate(360deg);
			-moz-transform: rotate(360deg);
			-webkit-transform: rotate(360deg);
			-moz-transform: rotate(360deg);
		}
		to {
			transform: rotate(0deg);
			-moz-transform: rotate(0deg);
			-webkit-transform: rotate(0deg);
			-moz-transform: rotate(0deg);
		}
	}
	#gnbQuickSettingPage .specialItem .el-input__inner{
		width:110px !important;
	}
	#gnbQuickSettingPage .specialItem .el-input-group__append{
		padding-left: 50px!important;
	}
#gnbQuickSettingPage .el-tabs__header {
	width: 100% !important;
	height: 28px !important;
	border-bottom: 1px solid #D5DCEC !important;
}
#gnbQuickSettingPage .el-tabs__content {
	width: 100% !important;
	background: #FFFFFF !important;
}
#gnbQuickSettingPage .el-tabs__active-bar {
	display: unset;
}
#gnbQuickSettingPage .el-tabs__item.is-active {
	background-color: unset;
}
</style>
<div class="panelDefault commonWarp" id="gnbQuickSettingPage" style="overflow:hidden;">
	<div class="headcloseBtn" style="top:6px;right:10px; line-height: 26px;" @click="syncSettings">
		<span class="el-icon el-icon-circle-refresh" style='position: unset; display: block; text-align: center;'></span>
	</div>
	<div class="logContent">
		<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="left" style="height:100%;overflow: auto;">
			<div class="cellBoxCls">
				<div class="cellTitleCls">
					<div style="width:50px">Cell</div>
					<div :class="cellNumList.length == 4? 'addCellBoxCls disabled' : 'addCellBoxCls'" @click="addCell">
						<span v-if="!addCellLoading" class="el-icon el-icon-plus" style="margin:0px 10px;"></span>
						<span v-if="addCellLoading" class="el-icon el-icon-reset active_btn" style="margin:0px 10px;"></span>
						<span><%=rb.getString("TianJia")%></span>
					</div>
					<div style="font-size:12px;color:#999999;margin-left:20px;">No more than 4</div>
				</div>
				<div class="cellNumListCls">
					<div v-for="(item,index) in cellNumList" class="cellItemCls">
						<span style="padding:0px 20px 0px 10px">{{item.value}}</span>
						<span class="el-icon el-icon-close" v-if="item.delFlag == 'true'" @click="delCell(item)"></span>
					</div>
				</div>
			</div>
			<div class="cellContentBoxCls" style="min-width: 600px;">
				<el-tabs v-model="activeName" type="card"  :before-leave="beforeTabLeave">
					<el-tab-pane v-for="(item,index) in cellList" :label="item.label" :name="item.name" class="cellTabItem">
						<div :id="item.name + 'Box'" style="overflow:auto;"></div>
					</el-tab-pane>
					<el-tab-pane label="BTS" name="station">
						<div style="overflow:auto;">
							<el-collapse v-model="stationCollapse" style="padding:30px 0px 40px 20px;">
								<el-collapse-item name="xn">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">Xn</span>
										</p>
									</template>
									<div class="rightContentCls" >
										<!--Xn-->
										<div> 
											<div class="contentTableTitle">
												<div>Xn List</div>
												<div><span class="el-icon el-icon-circle-add" @click="openAddTableListSlide({},'add','Xn')"></span></div>
											</div>
											<div class="cellTableBoxCls">
												<el-ctable
													ref="xnTable" 
													:row-class-name="tableRowClassName"
													:rownumber="true" 
													id="xnTable" 
													:data="ruleForm.XnList" 
													height="100%"
													:pagination="false"
													style="border:1px solid #E9E9E9;"
												>
													<el-table-column label='ID' min-width="40" prop="Xn_idx" show-overflow-tooltip></el-table-column>
													<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
														<template slot-scope="scope">
															<span class="el-icon el-icon-operation-edit" @click="openAddTableListSlide(scope.row,'edit','Xn')" style="margin-right:15px;"></span>
															<span class="el-icon el-icon-operation-delete" @click="delXnList(scope.row,event)" ></span>
														</template>
													</el-table-column>
													<el-table-column label='PLMN ID' min-width="120" prop="Xn_PLMNID" show-overflow-tooltip></el-table-column>
													<el-table-column label='RemoteAddress' min-width="120" prop="Xn_RemoteAddress" show-overflow-tooltip></el-table-column>
													<el-table-column label='XnLinkEnable' min-width="120" prop="Xn_LinkEnable" show-overflow-tooltip>
														<template slot-scope="scope">
															<div v-show="scope.row.Xn_LinkEnable == '0'">OFF</div>
															<div v-show="scope.row.Xn_LinkEnable == '1'">ON</div>
														</template>
													</el-table-column>
													<el-table-column label='XnHoEnable' min-width="120" prop="Xn_HoEnable" show-overflow-tooltip>
														<template slot-scope="scope">
															<div v-show="scope.row.Xn_HoEnable == '0'">OFF</div>
															<div v-show="scope.row.Xn_HoEnable == '1'">ON</div>
														</template>
													</el-table-column>
													<el-table-column label='Status' min-width="120" prop="Xn_Status" show-overflow-tooltip></el-table-column>
													<el-table-column v-if="false" prop="operateType" show-overflow-tooltip></el-table-column>
												</el-ctable>
                                                <el-form-item prop='XnList' style="display:none;" label="" label-width="0px">
                                                    <el-input v-model='ruleForm.XnList'></el-input>
                                                </el-form-item>
											</div>
											<div class="itemTitleCls">
												<div></div>
												<div>Xn Blacklist</div>
											</div>
											<div class="remoteAddressBoxCls" style="margin-left:16px;">
												<el-form-item prop="XnBlacklist" label="RemoteAddress"  label-width="160px" style="margin-bottom:0px;">
													<el-input style='width:200px;padding-top:7px;' v-model="remoteAddress" >
														<i slot="suffix" class="el-icon el-icon-plus" @click="addRemoteAddress" v-show="addRemoteAddressShow"></i>
														<div slot="suffix" class="disabledIconBox">
															<i  class="el-icon el-icon-plus" v-show="!addRemoteAddressShow"></i>
														</div>
													</el-input>
													<span class="mostNumberCls">No more than 8</span>
													<p class="errorBoxCls">{{remoteAddressErrorMessage}}</p>
												</el-form-item>
												<div class="itemListBoxCls">
													<div v-for="(item,index) in ruleForm.XnBlacklist" class="itemCls" v-show="!item.operateType || (item.operateType && item.operateType == 'add')">
														<span>{{item.remoteAddress}}</span>
														<span class="el-icon el-icon-close" style="margin-left:5px;" @click="remoteAddressListDel(item,index)"></span>
													</div>
												</div>
											</div>
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="anr">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">ANR</span>
										</p>
									</template>
									<div class="rightContentCls" >
										<div style="display:flex;margin-left:16px;flex-wrap: wrap">
											<el-form-item prop='anrEnable' style="width:40%;min-width:500px;" label="Enable" label-width="160px">
												<el-switch v-model="ruleForm.anrEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
											<el-form-item prop='interFeqEnable' style="width:40%;min-width:500px;" label="InterFeqEnable" label-width="160px">
												<el-switch v-model="ruleForm.interFeqEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
											<el-form-item prop='EUTRANEnable' style="width:40%;min-width:500px;" label="EUTRANEnable" label-width="160px">
												<el-switch v-model="ruleForm.EUTRANEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
											<el-form-item prop='BiNRCellEnable' style="width:40%;min-width:500px;" label="BiNRCellEnable" label-width="160px">
												<el-switch v-model="ruleForm.BiNRCellEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
											<el-form-item prop='MRTriggerType' style="width:40%;min-width:500px;" label="MRTriggerType" label-width="160px">
												<span slot="label" class="labelIconCls">
													MRTriggerType
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-select v-model='ruleForm.MRTriggerType' style="width:100px;padding-top:5px;">
													<el-option label='Event' value='0'></el-option>
													<el-option label='Period' value='1'></el-option>
												</el-select>
											</el-form-item>
											<el-form-item prop='absoluteThreshold' style="width:40%;min-width:500px;" label="AbsoluteThreshold" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													AbsoluteThreshold
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.absoluteThreshold' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='relativeThreshold' style="width:40%;min-width:500px;" label="RelativeThreshold" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													RelativeThreshold
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.relativeThreshold' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='absEnable' style="width:40%;min-width:500px;" label="AbsEnable" label-width="160px">
												<el-switch v-model="ruleForm.absEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
											<el-form-item prop='kpiPeriod' style="width:40%;min-width:500px;" label="KPIPeriod" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													KPIPeriod
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.kpiPeriod' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='autoAdjustEnable' style="width:40%;min-width:500px;" label="AutoAdjustEnable" label-width="160px">
												<el-switch v-model="ruleForm.autoAdjustEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
											<el-form-item prop='autoRemoveEnable' style="width:40%;min-width:500px;" label="AutoRemoveEnable" label-width="160px">
												<el-switch v-model="ruleForm.autoRemoveEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
											<el-form-item prop='autoRemovePeriod' style="width:40%;min-width:500px;" label="AutoRemovePeriod" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													AutoRemovePeriod
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.autoRemovePeriod' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='autoRemoveMaxCell' style="width:40%;min-width:500px;" label="AutoRemoveMaxCell" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													AutoRemoveMaxCell
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.autoRemoveMaxCell' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：Integer</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='maxHOtimes' style="width:40%;min-width:500px;" label="MaxHOtimes" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													MaxHOtimes
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.maxHOtimes' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='maxHOSuccess' style="width:40%;min-width:500px;" label="MaxHOSuccess" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													MaxHOSuccess
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.maxHOSuccess' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：0~100,Integer</template>
												</el-input>
											</el-form-item>
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="syncSettings">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">Sync Settings</span>
										</p>
									</template>
									<div class="rightContentCls">
										<div style="margin-left:16px;" v-if="productType == 'BaiBNQ'">
											<el-form-item style="width:40%;min-width:500px;margin-bottom:0px;" label="Sync Source" label-width="160px">
												<span slot="label" class="labelIconCls">
													Sync Source
												</span>
												<el-checkbox-group v-model="syncSourceSelectList" style="padding-top:10px;">
													<el-checkbox v-for="(item,index) in syncSourceList" :label="item" :key="index" @change="syncSourceItemChange(item)"></el-checkbox>
												</el-checkbox-group>
											</el-form-item>
											<el-form-item prop='syncSource' label-width="160px">
												<el-input v-model='ruleForm.syncSource' v-show="false"></el-input>
											</el-form-item>
										</div>
										<div style="display:flex;margin-left:16px;flex-wrap: wrap">
											<el-form-item v-if="productType == 'BaiBNX'" prop='syncSource' style="width:40%;min-width:500px;" label="Sync Source" label-width="160px">
												<span slot="label" class="labelIconCls">
													Sync Source
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-select v-model='ruleForm.syncSource' style="width:100px;padding-top:5px;">
													<el-option label='GPS' value='GPS'></el-option>
													<el-option label='BEIDOU' value='BEIDOU'></el-option>
													<el-option label='GPS_BEIDOU' value='GPS_BEIDOU'></el-option>
												</el-select>
											</el-form-item>
											
											<el-form-item prop='SyncMode' style="width:40%;min-width:500px;" label="Sync Mode" label-width="160px">
												<span slot="label" class="labelIconCls">
													Sync Mode
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-select v-model='ruleForm.SyncMode' style="width:100px;padding-top:5px;">
													<el-option label='FREE_Running' value='FREE_OSCILLATION'></el-option>
													<el-option label='GPS' value='GPS_PPS'></el-option>
													<el-option label='LOCAL_CLOCK_HOLDOVER_GPS_PPS' value='LOCAL_CLOCK_HOLDOVER_GPS_PPS'></el-option>
													<el-option label='OCXO_PPS' value='OCXO_PPS'></el-option>
												</el-select>
											</el-form-item>
											<el-form-item v-if="productType == 'BaiBNQ'" prop='ForcedSync' style="width:40%;min-width:500px;" label="Forced Sync" label-width="160px">
												<span slot="label" class="labelIconCls">
													Forced Sync
												</span>
												<el-select v-model='ruleForm.ForcedSync' style="width:100px;padding-top:5px;">
													<el-option label='ON' value='1'></el-option>
													<el-option label='OFF' value='0'></el-option>
												</el-select>
											</el-form-item>
											<el-form-item v-if="productType == 'BaiBNQ'" prop='PPSTimeOffset' style="width:40%;min-width:500px;" label="PPS Time Offset" label-width="160px" class='validate-item specialItem'>
												<span slot="label" class="labelIconCls">
													PPS Time Offset
												</span>
												<el-input v-model.trim='ruleForm.PPSTimeOffset' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：0~5000000,Integer</template>
												</el-input>
												<span class="prefix-label" style="position:relative;left:-226px;top:5px;">
													ns
												</span>
											</el-form-item>
										</div>
										<div class="syncSettingsInfo">
											<div><span>Current Sync Source</span><span>{{ruleForm.CurrentSyncSource}}</span></div>
											<div><span>GPS Sync Status</span><span>{{ruleForm.GPSSyncStatus}}</span></div>
											<div><span>Hardware Version</span><span>{{ruleForm.HardwareVersion}}</span></div>
											<div><span>Software Version</span><span>{{ruleForm.SoftwareVersion}}</span></div>
											<div><span>Sync Percent</span><span>{{ruleForm.SyncPercent}}</span></div>
											<div><span>Longitude</span><span>{{ruleForm.SyncLongitude}}</span></div>
											<div><span>Latitude</span><span>{{ruleForm.SyncLatitude}}</span></div>
											<div v-if="productType == 'BaiBNQ'"><span>Altitude</span><span>{{ruleForm.SyncAltitude}}</span></div>
											<div><span>Number Of Satellite</span><span>{{ruleForm.NumberOfSatellite}}</span></div>
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="HaloB" v-if="productType == 'BaiBNQ' && ruleForm.HalobMode">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">HaloB</span>
											<span v-if="halobChangeTipShow" style="margin-left:20px;font-size:12px;color:#FA5555"><%=rb.getString("ChongQiShengXiao")%></span>
										</p>
									</template>
									<div class="rightContentCls" >
										<div style="display:flex;margin-left:16px;flex-wrap: wrap">
											<el-form-item prop='HalobEnable' style="width:40%;min-width:500px;" label="HaloB Enable" label-width="160px">
												<el-switch v-model="ruleForm.HalobEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
											<el-form-item prop='HalobMode' style="width:40%;min-width:500px;" label="<%=rb.getString("LicenseMoShi")%>" label-width="160px">
												<span slot="label" class="labelIconCls">
													<%=rb.getString("LicenseMoShi")%>
												</span>
												<el-select v-model='ruleForm.HalobMode' style="width:100px;padding-top:5px;">
													<el-option label="--" value="0"></el-option>
													<el-option label="Centralized" value="1"></el-option>
													<el-option label="Single" value="2"></el-option>
												</el-select>
											</el-form-item>
										
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="gnb">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">gNB</span>
										</p>
									</template>
									<div class="rightContentCls" >
										<div style="display:flex;margin-left:16px;flex-wrap: wrap">
											<el-form-item prop='gnbLength' style="width:40%;min-width:500px;" label="gNB Length" label-width="140px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													gNB Length
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.gnbLength' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：22~32,Integer</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='gnbName' style="width:40%;min-width:500px;" label="gNB Name" label-width="140px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													gNB Name
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.gnbName' style="width:150px;padding-top:5px;">
													<template slot="append">Length：0~150</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='gnbId' style="width:40%;min-width:500px;" label="gNB ID" label-width="140px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													gNB ID
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.gnbId' style="width:150px;padding-top:5px;">
													<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,Integer</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='adminState' style="width:40%;min-width:500px;" label="AdminState" label-width="140px">
												<span slot="label" class="labelIconCls">
													AdminState
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-select v-model='ruleForm.adminState' style="width:100px;padding-top:5px;">
													<el-option label='Locked' value='1'></el-option>
													<el-option label='Unlocked' value='2'></el-option>
													<el-option label='ShuttingDown' value='3'></el-option>
												</el-select>
											</el-form-item>
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="plmn">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">PLMN</span>
										</p>
									</template>
									<div class="rightContentCls" >
										<div style="display:flex;margin-left:16px;flex-wrap: wrap">
											<el-form-item prop='multiPlmnEnable' style="width:40%;min-width:500px;" label="Multi Plmn Enable" label-width="160px">
												<el-switch v-model="ruleForm.multiPlmnEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="cu">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">CU</span>
										</p>
									</template>
									<div class="rightContentCls" >
										<div style="display:flex;margin-left:16px;flex-wrap: wrap">
											<el-form-item prop='CU_F1_C_Local_IP' style="width:40%;min-width:500px;" label="F1-C Local IP" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													F1-C Local IP
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.CU_F1_C_Local_IP' style="width:150px;padding-top:5px;">
													<template slot="append">Example：1.1.1.1</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='CU_F1_U_Local_IP' style="width:40%;min-width:500px;" label="F1-U Local IP" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													F1-U Local IP
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.CU_F1_U_Local_IP' style="width:150px;padding-top:5px;">
													<template slot="append">Example：1.1.1.1</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='CU_NG_C_Local_IP' style="width:40%;min-width:500px;" label="NG-C Local IP" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													NG-C Local IP
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.CU_NG_C_Local_IP' style="width:150px;padding-top:5px;">
													<template slot="append">Example：1.1.1.1</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='CU_NG_U_Local_IP' style="width:40%;min-width:500px;" label="NG-U Local IP" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													NG-U Local IP
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.CU_NG_U_Local_IP' style="width:150px;padding-top:5px;">
													<template slot="append">Example：1.1.1.1</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='CU_Xn_Local_IP' style="width:40%;min-width:500px;" label="Xn Local IP" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													Xn Local IP
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.CU_Xn_Local_IP' style="width:150px;padding-top:5px;">
													<template slot="append">Example：1.1.1.1</template>
												</el-input>
											</el-form-item>
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="du">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">DU</span>
										</p>
									</template>
									<div class="rightContentCls" >
										<div style="display:flex;margin-left:16px;flex-wrap: wrap">
											<el-form-item prop='DU_F1_C_Local_IP' style="width:40%;min-width:500px;" label="F1-C Local IP" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													F1-C Local IP
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.DU_F1_C_Local_IP' style="width:150px;padding-top:5px;">
													<template slot="append">Example：1.1.1.1</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='DU_F1_U_Local_IP' style="width:40%;min-width:500px;" label="F1-U Local IP" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													F1-U Local IP
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.DU_F1_U_Local_IP' style="width:150px;padding-top:5px;">
													<template slot="append">Example：1.1.1.1</template>
												</el-input>
											</el-form-item>
											<el-form-item prop='DU_F1_C_Remote_IP' style="width:40%;min-width:500px;" label="F1-C Remote IP" label-width="160px" class='validate-item'>
												<span slot="label" class="labelIconCls">
													F1-C Remote IP
													<!--<el-tooltip placement="bottom">
														<div slot="content">
															123456312123151<br/>
															321354534546
														</div>
														<span class="el-icon-circle-info el-icon"></span>
													</el-tooltip>-->
												</span>
												<el-input v-model.trim='ruleForm.DU_F1_C_Remote_IP' style="width:150px;padding-top:5px;">
													<template slot="append">Example：1.1.1.1</template>
												</el-input>
											</el-form-item>
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="lgw">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">LGW</span>
										</p>
									</template>
									<div class="rightContentCls" >
										<div style="display:flex;margin-left:16px;flex-wrap: wrap">
											<el-form-item prop='lgwEnable' style="width:40%;min-width:500px;" label="LGW Enable" label-width="160px">
												<el-switch v-model="ruleForm.lgwEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
											</el-form-item>
										</div>
									</div>
								</el-collapse-item>
								<el-collapse-item name="amf">
									<template slot='title'>
										<p style="display:inline-block;margin-left:40px;">
											<span class="title-icon" style="vertical-align:sub"></span>
											<span style="font-size:14px;font-weight:bold">AMF</span>
										</p>
									</template>
									<div class="rightContentCls" >
										<!--AMF-->
										<div> 
											<div class="contentTableTitle">
												<div>AMF List</div>
												<div><span class="el-icon el-icon-circle-add" @click="openAddTableListSlide({},'add','AMF')"></span></div>
											</div>
											<div class="cellTableBoxCls" style="margin-bottom: 20px;">
												<el-ctable
													ref="amfTable" 
													:row-class-name="tableRowClassName"
													:rownumber="true" 
													id="amfTable" 
													:data="ruleForm.AMFList" 
													height="100%"
													:pagination="false"
													style="border:1px solid #E9E9E9;"
												>
													<el-table-column label='ID' min-width="40" prop="AMF_idx" show-overflow-tooltip></el-table-column>
													<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
														<template slot-scope="scope">
															<span class="el-icon el-icon-operation-edit" @click="openAddTableListSlide(scope.row,'edit','AMF')" style="margin-right:15px;"></span>
															<span class="el-icon el-icon-operation-delete" @click="delAmfList(scope.row,event)" ></span>
														</template>
													</el-table-column>
													<el-table-column label='AMF IP' min-width="120" prop="AMF_IP" show-overflow-tooltip></el-table-column>
													<el-table-column label='PLMN ID' min-width="120" prop="AMF_PLMNID" show-overflow-tooltip></el-table-column>
													<el-table-column label='Default' min-width="120" prop="AMF_Default" show-overflow-tooltip></el-table-column>
												</el-ctable>
                                                <el-form-item prop='AMFList' style="display:none;" label="" label-width="0px">
                                                    <el-input v-model='ruleForm.AMFList'></el-input>
                                                </el-form-item>
											</div>
										</div>
									</div>
								</el-collapse-item>
							</el-collapse>
						</div>
					</el-tab-pane>
				</el-tabs>
			</div>
			<div class="footer">
				<div class="lnkbuttonGroup" style="padding:10px 0 0;">
					<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</el-form>
	</div>
	<!-- slide -->
	<el-slide  ref="sharingSlide" :url='sharingSlideUrl' :title="sharingSlideTitle" :footer="sharingSlideFooter" :header="sharingSlideHeader" :position="sharingSlidePosition"
		:height="sharingSlideHeight" :modal='modal' :width='sharingSlideWidth'  @cancel="sharingSlideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
</div>

<script type="text/javascript">
	
	var gnbQuickSettingPageVue = new Vue({
		el: '#gnbQuickSettingPage',
		data(){
			var vm = this;
			var validateRange = (rule,value,callback)=>{
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
				validateAutoRemoveMaxCell = (rule,value,callback)=>{
					var mag = rule.mag;
					var reg = /^\d+$/;
					if(value == '' || value == undefined || value == null){
						callback(new Error(mag))
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error(mag))
						}
					}
				},
				validateGnbName = (rule,value,callback)=>{
					var mag = rule.mag;
					if(value == '' || value == undefined || value == null){
						callback()
					}else{
						if(value.length<= 150){
							callback();
						}else{
							callback(new Error(mag))
						}
					}
				},
				validateIPaddress= (rule,value,callback) => {
					var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
					
					if(value === ''){
						callback()
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
						}
					}
				},
				validateIPv4AndIPv6 = (rule,value,callback)=>{

					if(value == '' || value == undefined || value == null){
						callback()
					}else{
						if(vm.isValidIP(value) || vm.isIPv6(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}
			};
			return {
				serialNumber:'',
				cellName:'',
				ruleForm:{
					XnList:[],
					XnBlacklist:[],
					anrEnable:'0',
					interFeqEnable:'0',
					EUTRANEnable:'0',
					BiNRCellEnable:'0',
					MRTriggerType:'0',
					absoluteThreshold:'',
					relativeThreshold:'',
					absEnable:'0',
					kpiPeriod:'',
					autoAdjustEnable:'0',
					autoRemoveEnable:'0',
					autoRemovePeriod:'',
					autoRemoveMaxCell:'',
					maxHOtimes:'',
					maxHOSuccess:'',
					syncSource:'',
					SyncMode:'',
					ForcedSync:'',
					PPSTimeOffset:'',
					CurrentSyncSource:'-',
					GPSSyncStatus:'1',
					HardwareVersion:'',
					SoftwareVersion:'',
					SyncPercent:'',
					SyncLongitude:'',
					SyncLatitude:'',
					SyncAltitude:'',
					NumberOfSatellite:'',
					HalobEnable:'',
					HalobMode:'',

					gnbLength:'',
					gnbName:'',
					gnbId:'',
					adminState:'1',

					multiPlmnEnable:'0',

					CU_F1_C_Local_IP:'',
					CU_F1_U_Local_IP:'',
					CU_NG_C_Local_IP:'',
					CU_NG_U_Local_IP:'',
					CU_Xn_Local_IP:'',

					DU_F1_C_Local_IP:'',
					DU_F1_U_Local_IP:'',
					DU_F1_C_Remote_IP:'',

					AMFList:[],

					lgwEnable:'0',
				},
				rules:{
					absoluteThreshold:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer'}],
					relativeThreshold:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer'}],
					kpiPeriod:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer'}],
					autoRemovePeriod:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer'}],
					autoRemoveMaxCell:[{validator:validateAutoRemoveMaxCell,mag:'<%=rb.getString("FanWei")%>：Integer'}],
					maxHOtimes:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer'}],
					maxHOSuccess:[{validator:validateRange,min:0,max:100,mag:'<%=rb.getString("FanWei")%>：0~100,Integer'}],
					PPSTimeOffset:[{validator:validateRange,min:0,max:5000000,mag:'<%=rb.getString("FanWei")%>：0~5000000,Integer'}],
					gnbLength:[{validator:validateRange,min:22,max:32,mag:'<%=rb.getString("FanWei")%>：22~32,Integer'}],
					gnbName:[{validator:validateGnbName,mag:'Length：0~150'}],
					gnbId:[{validator:validateRange,min:0,max:4294967295,mag:'<%=rb.getString("FanWei")%>0~4294967295,Integer'}],
					CU_F1_C_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
					CU_F1_U_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
					CU_NG_C_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
					CU_NG_U_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
					CU_Xn_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],

					DU_F1_C_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
					DU_F1_U_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
					DU_F1_C_Remote_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
				},
				castsList:{
					'BaiBNX':{
						'6BE6940D344E4215A704E7F5C83690A9':'XnList',
						'BFD051C0A7F9820B2C52FE9CD32A3A02':'Xn_idx',
						'5CA633A471271633EECBFCDCFF02D9F1':'Xn_PLMNID',
						'84ED79E36C73794351FF903957F872C2':'Xn_RemoteAddress',
						'66F7D160C9EFCF551B88A19C44E4432E':'Xn_LinkEnable',
						'288F05566297914DED65ACB247CF4BA0':'Xn_HoEnable',
						'4EE9D25B5B3C5BC2B833DEAA3BDA5AEF':'Xn_Status',

						'999EAC5208400943626160906C500A5E':'XnBlacklist',
						'0C41C9198C0BB434335D9C2C45204EB2':'remoteAddress',
						'943F4BBB1BE70A3C88B871324DCE2C99':'remoteAddress_idx',
						'D7741D897783151220C7D0761929FCF3':'anrEnable',
						'18D95FC55660F147FDBB89549B7BA312':'interFeqEnable',
						'F8F43C01D84156F7E94EEDA1E4D5F381':'EUTRANEnable',
						'1214F4E79FECA5CD18C101BFF6E6BC11':'BiNRCellEnable',
						'DA974A9E0D13DBB7C88DFFE6266F29C7':'MRTriggerType',
						'E54C435C40A04DFDC30D69BD086F5691':'absoluteThreshold',
						'493DD291FC0D8279835A4E6FFC7E56A6':'relativeThreshold',
						'2E4EBE86AC4BD97AF2699D8ADF33F0E4':'absEnable',
						'FDE84C30B792A209AF1591A27E3E8F1B':'kpiPeriod',
						'4BAE90F5FF498B910456B6D5F31A1962':'autoAdjustEnable',
						'088963A022C30B74CA0A6A5596436AA9':'autoRemoveEnable',
						'C289A56B29E687532051A23E47AB654D':'autoRemovePeriod',
						'56F5DBF385B1C8E5C58188403937153D':'autoRemoveMaxCell',
						'CD5E631E01D678DE4265CEC2C6E3D838':'maxHOtimes',
						'C9E91D67F72AC190AC3F0F6865970C65':'maxHOSuccess',
						'E1DA15315F2CE06A4F186C4DB65B5A75':'syncSource',
						'221E4BE98F7BFD986D39B2222E5C1D06':'SyncMode',
						'226520E55096AD94B8E9D730CF0F3761':'ForcedSync',
						'79E17C7D72871DE22FB71714C076F327':'PPSTimeOffset',
						'74FD52E553D04000B2BEA0E3DC9FD157':'CurrentSyncSource',
						'53D7A8A06FE2DAC787FE95FC68E2E6A6':'GPSSyncStatus',
						'83682BA01108782FB773E97D634F8A41':'HardwareVersion',
						'6869DC733F299B33E72C2798D2CB9937':'SoftwareVersion',
						'B5FBC32B4091D2B18A2FE3CE3DBE74AC':'SyncPercent',
						'EEF05E57F5A47AABE05B10875E88575C':'SyncLongitude',
						'1FEBDE1DF41470F4B382796F4A37F9F9':'SyncLatitude',
						'0CDC629D411D4A4032CB7A8BDFAD392B':'SyncAltitude',
						'41318EA31BEE7A044FC63732A17EA23E':'NumberOfSatellite',

						'3E14072D60E96AF83489D7534D270B87':'gnbLength',
						'041A9BC923822E0B9C1F1A437B5E5AC3':'gnbName',
						'D0E8C432350E72E362FFC145BA6EB802':'gnbId',
						'3B995397E62843872955773E84D08164':'adminState',

						'D0EBA5CC71DD66B058CC92FDDC2EB582':'multiPlmnEnable',

						'90F4FBA74F9B5A61FFC5AFF989D6B156':'CU_F1_C_Local_IP',
						'012B4E51E75F5D399982454C845B4823':'CU_F1_U_Local_IP',
						'DF93D693E413C0A9B5AC16C8F522E8DE':'CU_NG_C_Local_IP',
						'88D8386A9AA514F15A6E58EFB22E9910':'CU_NG_U_Local_IP',
						'A626B5BA2475A2F3E9F263602B3211E9':'CU_Xn_Local_IP',

						'95C79E66D356E021C7CEE3CC86622372':'DU_F1_C_Local_IP',
						'020427C5ED1255802F056C6CA6D71604':'DU_F1_U_Local_IP',
						'382675A4724660A2B16658BA39F933F4':'DU_F1_C_Remote_IP',

						'B54E17B4C27DFD51AECEA269C4BC6BFD':'AMFList',
						'7264445E090723EC40C6C64129F3EBDE':'AMF_idx',
						'2F3B65D0690DAEF67BB2C3562FC52FE4':'AMF_IP',
						'C1F123458F26A6BFA10AD6F638A3E61A':'AMF_PLMNID',
						'8C225AF3D7231D2EB7439CEAE444972D':'AMF_Default',

						'FF548107918C0B0DD1082BEFA57DB58E':'lgwEnable',
					},
					'BaiBNQ':{
						'7950162C9A24A70DDDC3118E061E0A27':'XnList',
						'F924527EF1DE6F5A3C1BCA790FADC9F5':'Xn_idx',
						'2CC55133A6849300DA99C935B82C1752':'Xn_PLMNID',
						'F7BCD29DC1DAC25C6112149537F1FDF9':'Xn_RemoteAddress',
						'A61F75C2D2933B58903E3F1AB1B37363':'Xn_LinkEnable',
						'7F2A16BAE7016B1FE8A0E6DC9C2C02FF':'Xn_HoEnable',
						'58547675CFA0A71527B29D9A26438F22':'Xn_Status',

						'4A05D255E8EC01783A1B6E700DDC7CE4':'XnBlacklist',
						'B9A7A2908C1C45DE4B691C83C6F00841':'remoteAddress',
	                    '3D0958ECEAE82263012001EC439348C3':'remoteAddress_idx',
						'82B091C50808E3E6F5D33EFE2760F033':'anrEnable',
						'4B7864F991DF752D3266D010055C4FD2':'interFeqEnable',
						'E02C0EC06DB0F23161D37F636CF051EB':'EUTRANEnable',
						'93936E2D407EBE9B4D20FE4F36BA80EC':'BiNRCellEnable',
						'FB2B5F5BBE059A2A934CFFFF57D7589D':'MRTriggerType',
						'22C51B323E5AB7BD2DDCE44AE6A96574':'absoluteThreshold',
						'E1F8C9EF3E7FEDCED5D2465489C8EC0C':'relativeThreshold',
						'EA1E967554408FA4417F6FC14586F8C1':'absEnable',
						'FA5C715ECED420F841E508BE988D3978':'kpiPeriod',
						'A6F5DE08C3490182866ED03AACDD9AC2':'autoAdjustEnable',
						'41F6A657567A148A560400C25B84CF8D':'autoRemoveEnable',
						'E3E128A2D938BCAA2370BCFDD7CF3F3B':'autoRemovePeriod',
						'D0012607E22676DFBF51D07336379E07':'autoRemoveMaxCell',
						'028E42BB378CD4C349A38955123D2512':'maxHOtimes',
						'C71A32B8C5605A916D59DD76AEB1EF69':'maxHOSuccess',
						'1E105E23E899B2277FE54FE925B2C943':'syncSource',
						'A0E8CB20E1C810EA92B91D36218EAC8D':'SyncMode',

						'A99FFF6D75D09BFB523EC8710447958F':'ForcedSync',
						'F1BE0ADD4E00908B64698F38E3062519':'PPSTimeOffset',

						'4EA357245EAE2CE9F6D341712921DC70':'CurrentSyncSource',
						'64FC363430543E441354605F87B27088':'GPSSyncStatus',
						'E45286B8268F3CB13DF04878D2FF96CA':'HardwareVersion',
						'EC7D1729FD023FF7C66672887BAD0C52':'SoftwareVersion',
						'98777178132532BBBB5B0FC62D5E78F7':'SyncPercent',

						'158C7E2E6E5C6D9353F9CD4D270802CB':'SyncLongitude',
						'624D9BAFED87EA3FBB0C0691B87850BD':'SyncLatitude',
						'C72F802951EE784484B1D14246E1EE47':'SyncAltitude',

						'FB80AFF4FCBB7EF3EA818CFE2B578380':'NumberOfSatellite',
						'808122E4D7A23811A18E64D117B13660':'HalobEnable',
						'DB0E7ADF007CE41C11782D18570F764F':'HalobMode',

						'36747EE98FBA37A491196CCA20E88E8D':'gnbLength',
						'BD0CA05B42E723AD02312D242BC05B87':'gnbName',
						'CF5510F1F52BCB149062D869C481A2EB':'gnbId',
						'F7BBCDB03FB7643B3A16A849128940F4':'adminState',

						'8863A2D15601150146F772A537526E5C':'multiPlmnEnable',

						'4E4B0794F3F3C5F8AACA85D8DAFD0F73':'CU_F1_C_Local_IP',
						'42B352CBDEC5CE9D4769A49B668E8D25':'CU_F1_U_Local_IP',
						'B84A0C2B6DBF02F1310CE12FD46A06CA':'CU_NG_C_Local_IP',
						'2B20F3BC72DABBBD557D2DEF43AC79E8':'CU_NG_U_Local_IP',
						'76B29306617A5D2D9798A3467C7CFC13':'CU_Xn_Local_IP',

						'A64BC450133733601405A8B7160CDD7F':'DU_F1_C_Local_IP',
						'3D86ABF140171F229983EDB6CC6600F2':'DU_F1_U_Local_IP',
						'8F807FCD64D888C3C51FEA2F7A17D315':'DU_F1_C_Remote_IP',

						'B7EF0230C260BFB555FAE99A93662E28':'AMFList',
						'D2EE312531CA9ED69741523F9CA5B151':'AMF_idx',
						'3CE9BF9752F667F6AF50EC00A823E72C':'AMF_IP',
						'F4B3ED9972A6303F1345DC27971AAE8B':'AMF_PLMNID',
						'0E6C432B61CD906596E8CC4B13445F7B':'AMF_Default',

						'34C7445E6DFE01C4E341C380ABF2DDC6':'lgwEnable',

					}
				},
				casts:{},
				sharingSlideUrl:'',
				sharingSlideTitle:'',
				sharingSlideFooter:'',
				sharingSlideHeader:'',
				sharingSlidePosition:'',
				sharingSlideHeight:'',
				sharingSlideWidth:'',

				cellNumList:[],
				activeName:'station',
				leftTabsActive:'1',
				stationCollapse:['xn','anr','syncSettings','gnb','plmn','cu','du','lgw','amf'],
				remoteAddress:'',
				remoteAddressErrorMessage:'',
				cellList:[],
				codeList:[],
				addCellLoading:false,
				productType:'',
				oldRuleForm:{
					HalobEnable:'',
					HalobMode:'',
				},
				syncSourceSelectList:[],
				syncSourceList:['GPS','GLONASS','BEIDOU','GALILEO','QZSS']
			}
		},
		computed: {
           addRemoteAddressShow(){
				var Num = 0;
				if(this.ruleForm.XnBlacklist.length>0){
					var delList=[],
						noDel=[];
					this.ruleForm.XnBlacklist.map((item)=>{
						if(item.operateType && item.operateType == 'remove'){
							delList.push(item)
						}else{
							noDel.push(item)
						}
					})
					Num = noDel.length;
				}

				return Num < 8
		   },
		   halobChangeTipShow(){
			   return this.ruleForm.HalobEnable != this.oldRuleForm.HalobEnable || this.ruleForm.HalobMode != this.oldRuleForm.HalobMode
		   }
		},
		watch: {
			
		},
		methods: {
			// 初始化
			/*init(row){
				var vm = this;
					//row =  gnbTabSettingVue.rowData;
					
				vm.rowData = row;
				vm.smallCellCode = row.small_cell_code;
				vm.serialNumber = row.serial_number;
				vm.cellName = row.host_name;
				vm.productType = row.product;
				vm.casts = vm.castsList[vm.productType];
                var codeList=[];
                 Object.keys(vm.casts).forEach(function(key){
                    codeList.push(key)
                });
                vm.codeList = codeList;
				vm.getParamCellList(vm.smallCellCode,'21000');
				vm.getParamStationData(vm.smallCellCode,'22000');
				
			},*/
			inits(){
				var vm = this,
					row =  gnbTabSettingVue.rowData;
					
				vm.rowData = row;
			
				vm.smallCellCode = row.small_cell_code;
				vm.serialNumber = row.serial_number;
				vm.cellName = row.host_name;
				vm.productType = row.product;
				vm.casts = vm.castsList[vm.productType];
                var codeList=[];
                 Object.keys(vm.casts).forEach(function(key){
                    codeList.push(key)
                });
                vm.codeList = codeList;
				vm.getParamCellList(vm.smallCellCode,'21000');
				vm.getParamStationData(vm.smallCellCode,'22000');
			},
			// 获取当前有那几个Cell
			getParamCellList(code,id){
				var vm = this,
                    url = '${ctx}/cell/quicksettings/getListParamIndex.action',
                    params = {
                        id: id,
                        smallCellCode: code
                    };

                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data.rows;
                    var cellList = [],cellNumArr = [],
						codes = {
							'BaiBNX':'0AC96C7FC6D477CE3DEC796809C0A5AA',
							'BaiBNQ':'8DC2F720296FDFEAC74F35726F4FD95B',
						};
						
                    data.map((item)=>{
						cellList.push({
							label:"Cell"+ item[codes[vm.productType]],
							name:'cell'+ item[codes[vm.productType]],
							value: item[codes[vm.productType]]
						})
						cellNumArr.push(parseInt(item[codes[vm.productType]]))
					});
					vm.cellList = cellList;

					var maxCell = Math.max(...cellNumArr);
					cellList.map((item)=>{
						// if(item.value == (maxCell + '')){
						// 	item.delFlag = 'true'
						// }else{
						// 	item.delFlag = 'false'
						// }
						if(item.value == '1'){
							item.delFlag = 'false'
						}else{
							item.delFlag = 'true'
						}
						
					})
					vm.cellNumList = cellList;
					if(vm.activeName != 'station'){
						var result = null;
						vm.cellList.map((item)=>{
							if(item.name == vm.activeName){
								result = item.name
							}
						})
						if(!result){
							vm.activeName = 'station';
						}
					}
                });
			},
			getParamStationData(code,id) {
				var vm = this,
					codes = [],
					url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
					params = {
						id: id,
						smallCellCode: code
					};
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;
					vm.resetFormData();
					if(data && Array.isArray(data)) {
						data.map(function(item){
							item.groups.map(function(group){
								group.list.map(function(m){
									if(m.type == 'list'){
										vm.initTable(m.url,m.label);
									}else{
										codes.push(m.name);
										// 执行赋值
										vm.setValue(m);
									}
								});
							});
						});
						initForm(vm.$refs.ruleForm);
					}
				});
			},
			// 映射赋值
			setValue(item) {
				var vm = this,
				code = item.name,
				value = item.value;

				// indexs是否含有
					var key = vm.casts[code];
				try{
					if(key){
						if(vm.productType == 'BaiBNQ' && key == 'syncSource'){
							vm.syncSourceSelectList = value.split('_');
						}
						vm.ruleForm[key] = value;
						['HalobEnable','HalobMode'].map((item)=>{
							if(item == key){
								vm.oldRuleForm[key] = value;
							}
						})
					}
				}catch(e){}
			},
			initTable(url,type){
				var vm = this,codes = [];
				var params = {
						smallCellCode : vm.smallCellCode
				}
				axios.post(url,stringify(params)).then(res=>{
					var data = res.data;
					if(data.rows){
						data.rows.map(item=>{
                            var obj = {};
            				for(var key in item){
            					codes.push(key);
            					obj[vm.casts[key]] = item[key]
            				}
                            if(type == 'Xn List'){
                                vm.ruleForm.XnList.push(obj);
                            }else if(type == 'Xn Blacklist'){
                                vm.ruleForm.XnBlacklist.push(obj);
                            }else if(type == 'AMF List'){
								vm.ruleForm.AMFList.push(obj);
							}
						})
                        initForm(vm.$refs.ruleForm);
					}
				})
			},
			// 新增小区
			addCell(){
				var vm = this,
					urls = '${ctx}/gnb/quicksetting/addGnbCell.action',
					params={
						smallCellCode:vm.smallCellCode
					};
				if(vm.cellNumList.length == 4 || vm.addCellLoading == true){
					return
				}
				vm.addCellLoading = true;
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});
						vm.addCellLoading = false;
						vm.getParamCellList(vm.smallCellCode,'21000');
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			// 删除 Cell
			delCell(row){
				var vm = this,
					urls = '${ctx}/gnb/quicksetting/delGnbCell.action',
					params={
                        smallCellCode: vm.smallCellCode,
						index:row.value
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
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
							vm.getParamCellList(vm.smallCellCode,'21000');
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{})
			},
			//  tab 切换前监听事件
			beforeTabLeave(activeName,oldActiveName){
				var vm = this,
                    code={
                        'cell1':'${ctx}/cell/quicksettings/goCell0ParamPage.action',
                        'cell2':'${ctx}/cell/quicksettings/goCell1ParamPage.action',
                        'cell3':'${ctx}/cell/quicksettings/goCell2ParamPage.action',
                        'cell4':'${ctx}/cell/quicksettings/goCell3ParamPage.action',
                    };
				var p = new Promise((resolve,reject)=>{
					if(oldActiveName == 'cell1'){
						var isChanged = isFormChanged(cell1InfoVue.$refs.ruleForm);
						if(isChanged){
							vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
								customClass:'warningConfirm',
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal:false
							}).then(() => {
								if(activeName != 'station'){
									var ids = '#'+activeName+'Box',
										urls = code[activeName];
									$(ids).addClass('loading');
									$(ids).load(urls,function(data){
										$.parser.parse(this);
										var cellIndex = activeName.substr(activeName.length-1,1)
										eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
									});
								}else{
									vm.getParamStationData(vm.smallCellCode,'22000');
								}
								resolve()
							}).catch(() => {
								reject()
							})
						}else{
							if(activeName != 'station'){
								var ids = '#'+activeName+'Box',
									urls = code[activeName];
								$(ids).addClass('loading');
								$(ids).load(urls,function(data){
									$.parser.parse(this);
									var cellIndex = activeName.substr(activeName.length-1,1)
									eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
								});
								
							}else{
								vm.getParamStationData(vm.smallCellCode,'22000');
							}
							resolve() 
						}
					}else if(oldActiveName == 'cell2'){
						var isChanged = isFormChanged(cell2InfoVue.$refs.ruleForm);
						if(isChanged){
							vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
								customClass:'warningConfirm',
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal:false
							}).then(() => {
								if(activeName != 'station'){
									var ids = '#'+activeName+'Box',
										urls = code[activeName];
									$(ids).addClass('loading');
									$(ids).load(urls,function(data){
										$.parser.parse(this);
										var cellIndex = activeName.substr(activeName.length-1,1)
										eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
									});   
								}else{
									vm.getParamStationData(vm.smallCellCode,'22000');
								}
								resolve() 
							}).catch(() => {
								reject()
							})
						}else{
							if(activeName != 'station'){
								var ids = '#'+activeName+'Box',
									urls = code[activeName];
								$(ids).addClass('loading');
								$(ids).load(urls,function(data){
									$.parser.parse(this);
									var cellIndex = activeName.substr(activeName.length-1,1)
									eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
								});   
							}else{
								vm.getParamStationData(vm.smallCellCode,'22000');
							}
							resolve() 
						}
					}else if(oldActiveName == 'cell3'){
						var isChanged = isFormChanged(cell3InfoVue.$refs.ruleForm);
						if(isChanged){
							vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
								customClass:'warningConfirm',
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal:false
							}).then(() => {
								if(activeName != 'station'){
									var ids = '#'+activeName+'Box',
										urls = code[activeName];
									$(ids).addClass('loading');
									$(ids).load(urls,function(data){
										$.parser.parse(this);
										var cellIndex = activeName.substr(activeName.length-1,1)
										eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
									});   
								}else{
									vm.getParamStationData(vm.smallCellCode,'22000');
								}
								resolve() 
							}).catch(() => {
								reject()
							})
						}else{
							if(activeName != 'station'){
								var ids = '#'+activeName+'Box',
									urls = code[activeName];
								$(ids).addClass('loading');
								$(ids).load(urls,function(data){
									$.parser.parse(this);
									var cellIndex = activeName.substr(activeName.length-1,1)
									eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
								});   
							}else{
								vm.getParamStationData(vm.smallCellCode,'22000');
							}
							resolve() 
						}
					}else if(oldActiveName == 'cell4'){
						var isChanged = isFormChanged(cell4InfoVue.$refs.ruleForm);
						if(isChanged){
							vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
								customClass:'warningConfirm',
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal:false
							}).then(() => {
								if(activeName != 'station'){
									var ids = '#'+activeName+'Box',
										urls = code[activeName];
									$(ids).addClass('loading');
									$(ids).load(urls,function(data){
										$.parser.parse(this);
										var cellIndex = activeName.substr(activeName.length-1,1)
										eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
									});   
								}else{
									vm.getParamStationData(vm.smallCellCode,'22000');
								}
								resolve() 
							}).catch(() => {
								reject()
							})
						}else{
							if(activeName != 'station'){
								var ids = '#'+activeName+'Box',
									urls = code[activeName];
								$(ids).addClass('loading');
								$(ids).load(urls,function(data){
									$.parser.parse(this);
									var cellIndex = activeName.substr(activeName.length-1,1)
									eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
								});   
							}else{
								vm.getParamStationData(vm.smallCellCode,'22000');
							}
							resolve()
						}
					}else if(oldActiveName == 'station') {
						var isChanged = isFormChanged(vm.$refs.ruleForm);
						if(isChanged){
							vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
								customClass:'warningConfirm',
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal:false
							}).then(() => {
								if(activeName != 'station'){
									var ids = '#'+activeName+'Box',
										urls = code[activeName];
									$(ids).addClass('loading');
									$(ids).load(urls,function(data){
										$.parser.parse(this);
										var cellIndex = activeName.substr(activeName.length-1,1)
										eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
									});
								}else{
									vm.getParamStationData(vm.smallCellCode,'22000');
								}
								resolve() 
							}).catch(() => {
								reject()
							})
						}else{
							if(activeName != 'station'){
								var ids = '#'+activeName+'Box',
									urls = code[activeName];
								$(ids).addClass('loading');
								$(ids).load(urls,function(data){
									$.parser.parse(this);
									var cellIndex = activeName.substr(activeName.length-1,1)
									eventBus.$emit('cellInfo-init',vm.smallCellCode,cellIndex,vm.productType);
								}); 
							}else{
								vm.getParamStationData(vm.smallCellCode,'22000');
							}
							resolve() 
						}
					}
				})
				return p
				
			},
			
			//打开NrCell新增页面
			openAddNrCellSlide(type,row,plmnType){
				var vm = this;
					
				vm.sharingSlideUrl = '${ctx}/cell/quicksettings/goNrCellParamPage.action';
				vm.sharingSlideHeight = '100%';
				vm.sharingSlideWidth = '100%';
				vm.sharingSlideFooter = false;
				vm.sharingSlidePosition = 'top';
				vm.sharingSlideHeader = false;
				vm.sharingSlideTitle = '';
				vm.$refs.sharingSlide.showSlide(function(){
					eventBus.$emit('addNrCell-init',row,type,vm.productType,plmnType);
				});
			},
			/*
			**打开新增表格参数类型
			*optType：add  edit
			* addType: LTE_N-FREQ LTE_N-CELL  NR_N-FREQ NR_N_CELL Xn
			*/ 
			openAddTableListSlide(row,optType,addType){
				var vm = this;
					
				vm.sharingSlideUrl = "${ctx}/cell/quicksettings/goXnParamPage.action";
				vm.sharingSlideHeight = '100%';
				vm.sharingSlideWidth = '100%';
				vm.sharingSlideFooter = false;
				vm.sharingSlidePosition = 'top';
				vm.sharingSlideHeader = false;
				vm.sharingSlideTitle = '';
				vm.$refs.sharingSlide.showSlide(function(){
					eventBus.$emit('addTable-init',row,optType,addType);
				});
			},
			// 删除 Xn
			delXnList(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
                    var delFlag=false;
                    vm.ruleForm.XnList.map(function(item,index){
                        if(item.Xn_idx == row.Xn_idx){
                            if(item.operateType == 'add'){
                                delFlag = true;
                            }else{
								var params = item;
								params.operateType = 'remove';
								vm.$set(vm.ruleForm.XnList,index,params);
                            }
                            
                        }
                    })
                    if(delFlag){
                        vm.ruleForm.XnList = vm.ruleForm.XnList.filter((items)=>{
                            return items.Xn_idx != row.Xn_idx
                        })
                    }
				})
			},
			// 删除 Amf
			delAmfList(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
                    var delFlag=false;
                    vm.ruleForm.AMFList.map(function(item,index){
                        if(item.AMF_idx == row.AMF_idx){
                            if(item.operateType == 'add'){
                                delFlag = true;
                            }else{
								var params = item;
								params.operateType = 'remove';
								vm.$set(vm.ruleForm.AMFList,index,params);
                            }
                            
                        }
                    })
                    if(delFlag){
                        vm.ruleForm.AMFList = vm.ruleForm.AMFList.filter((items)=>{
                            return items.AMF_idx != row.AMF_idx
                        })
                    }
				})
			},
			// 关闭slide页面
			sharingSlideCancel(){
				gnbQuickSettingPageVue.$refs.sharingSlide.hide();
			},
			// 左侧tab点击事件
			leftTabsClick(val){
				var vm = this;
				vm.leftTabsActive = val;
			},
			// 新增 RemoteAddress
			addRemoteAddress(){
				var vm =this,
					val = vm.remoteAddress;
				if(!val)return
				if(vm.isValidIP(val) || vm.isIPv6(val)) {
                     var obj={
                            remoteAddress:val,
							operateType:'add'
                        };
                    vm.ruleForm.XnBlacklist.push(obj);
                    vm.remoteAddress = '';
                    vm.remoteAddressErrorMessage = '';
				}else {
					vm.remoteAddressErrorMessage = 'Example：1.1.1.1';
				}
			},
			remoteAddressListDel(row,index){
				var vm =this,
					delItem =  vm.ruleForm.XnBlacklist[index]

				if(delItem.operateType && delItem.operateType == 'add'){
					 vm.ruleForm.XnBlacklist.splice(index,1)
				}else{
					vm.$set(vm.ruleForm.XnBlacklist[index],'operateType','remove');
				}
                

			},
			settingsSubmit(){
				var vm = this;
				if(vm.activeName == 'cell1'){
					eventBus.$emit('cell1-submit');
				}else if(vm.activeName == 'cell2'){
					eventBus.$emit('cell2-submit');
				}else if(vm.activeName == 'cell3'){
					eventBus.$emit('cell3-submit');
				}else if(vm.activeName == 'cell4'){
					eventBus.$emit('cell4-submit');
				}else if(vm.activeName == 'station'){
					var params = {},
						isChanged = isFormChanged(vm.$refs.ruleForm),
						isSync = false;

					if(!isChanged){
						showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
						return;
					}

					vm.$refs.ruleForm.fields.map(function(field){
						var key = vm.getNameByProp(field.prop);

						if(Array.isArray(field.fieldValue)){
							var vList = field.fieldValue.map(function(item){return item}),
								oList = (field.reinitialValue||[]).map(function(item){return item}),
								val = JSON.stringify(vList.sort()),
				   				orVal = JSON.stringify(oList.sort());

							if(val != orVal) {
								var editList=[],subList=[];
                                vList.map((items)=>{
                                    if(items.operateType){
										if(items.Xn_idx && items.operateType == 'add'){
											isSync = true
										}
                                        editList.push(items)
                                    }
                                })

                                editList.map((items)=>{
                                    var objs={};
                                    for(var listVal in items){
                                        var listKey = vm.getNameByProp(listVal);
										if(listVal != 'Xn_Status'){
											objs[listKey] = items[listVal]
										}
                                    }
                                    subList.push(objs)
                                })
                                
						        params[key] = subList;
							};
						}else{
							if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
								
							}else if(field.fieldValue != field.reinitialValue) {
                                var editData={
                                    cellIndex:vm.cellIndex,
                                    value:field.fieldValue
                                }
								params[key] = editData;
							};
						}
					});
					vm.$refs.ruleForm.validate(function(valid){
						if(valid) {
							var rowCode = vm.smallCellCode,
								url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
							$('#gnbQuickSettingPage').addClass('loading');
							axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
								var data = res.data;
								if(data["success"]){
									if(isSync){
										vm.syncParams();
									}
									vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
									eventBus.$emit('close-gnb-settingPage');
								}else{
									vm.$message.error(data["message"])
								}
								$('#gnbQuickSettingPage').removeClass('loading');
							})
						}
					});

				}
			},
			// 同步
			syncParams(){
				var vm = this,
					urls='${ctx}/cell/quicksettings/sync.action',
					codes = {
						'BaiBNX':'6BE6940D344E4215A704E7F5C83690A9',
						'BaiBNQ':'7950162C9A24A70DDDC3118E061E0A27',
					},
					params={
						smallCellCode: vm.smallCellCode,
						paramId:codes[vm.productType]
					};
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
				})
			},
			getNameByProp(prop) {
				var vm = this,
					reg = /^\w*\.\d*\.\w*$/
					key = prop;
				
				if(reg.test(prop)) {
					var mReg = /\.(\d*)\./,
						sufReg = /\.(\w*)$/,
						idx = prop.match(mReg)[1],
						sufStr = prop.match(sufReg)[1];

					vm.codeList.map(function(name){
						var index = vm.indexs[name];
						if(vm.casts[name] == sufStr && index == idx) {
							key = name;
						}
					});
				}else {
					vm.codeList.map(function(name){
						if(vm.casts[name] == prop) {
							key = name;
						}
					});
				}

				return key;
			},
			// 同步设置 
			syncSettings(){
				var vm = this,
					activeName = vm.activeName,
					urls='${ctx}/cell/quicksettings/sync.action',
					params={
						 smallCellCode: vm.smallCellCode
					};
					
				if(activeName == 'cell1'){
					var isChanged = isFormChanged(cell1InfoVue.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							vm.syncSubmit(urls,params);
						}).catch(() => {
							
						})
					}else{
						vm.syncSubmit(urls,params);
					}
				}else if(activeName == 'cell2'){
					var isChanged = isFormChanged(cell2InfoVue.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							vm.syncSubmit(urls,params);
						}).catch(() => {
							
						})
					}else{
						vm.syncSubmit(urls,params);
					}
				}else if(activeName == 'cell3'){
					var isChanged = isFormChanged(cell3InfoVue.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							vm.syncSubmit(urls,params);
						}).catch(() => {
							
						})
					}else{
						vm.syncSubmit(urls,params);
					}
				}else if(activeName == 'cell4'){
					var isChanged = isFormChanged(cell4InfoVue.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							vm.syncSubmit(urls,params);
						}).catch(() => {
							
						})
					}else{
						vm.syncSubmit(urls,params);
					}
				}else if(activeName == 'station') {
					var isChanged = isFormChanged(vm.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							vm.syncSubmit(urls,params);
						}).catch(() => {
							
						})
					}else{
						vm.syncSubmit(urls,params);
					}
				}
			},
			syncSubmit(urls,params){
				var vm = this, str = Math.random().toString();
				
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data["success"]){
						gnbTabSettingVue.changeMain('quickSetting_X86');
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			closeSettings(){
				var vm = this,
					activeName = vm.activeName;
					
				if(activeName == 'cell1'){
					var isChanged = isFormChanged(cell1InfoVue.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							eventBus.$emit('close-gnb-settingPage');
						}).catch(() => {
							
						})
					}else{
						eventBus.$emit('close-gnb-settingPage');
					}
				}else if(activeName == 'cell2'){
					var isChanged = isFormChanged(cell2InfoVue.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							eventBus.$emit('close-gnb-settingPage');
						}).catch(() => {
							
						})
					}else{
						eventBus.$emit('close-gnb-settingPage');
					}
				}else if(activeName == 'cell3'){
					var isChanged = isFormChanged(cell3InfoVue.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							eventBus.$emit('close-gnb-settingPage');
						}).catch(() => {
							
						})
					}else{
						eventBus.$emit('close-gnb-settingPage');
					}
				}else if(activeName == 'cell4'){
					var isChanged = isFormChanged(cell4InfoVue.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							eventBus.$emit('close-gnb-settingPage');
						}).catch(() => {
							
						})
					}else{
						eventBus.$emit('close-gnb-settingPage');
					}
				}else if(activeName == 'station') {
					var isChanged = isFormChanged(vm.$refs.ruleForm);
					if(isChanged){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							eventBus.$emit('close-gnb-settingPage');
						}).catch(() => {
							
						})
					}else{
						eventBus.$emit('close-gnb-settingPage');
					}
				}
				
			},
            // 判断是否为空
            isNull(val){
                if(val==undefined || val == null || val =="") return true;
                else return false;
            },
			tableRowClassName({row,rowIndex}){
				if(row.operateType && row.operateType == 'remove'){
					return 'hidden-row'
				}
				return ''
			},
			// 重置form数据
			resetFormData(){
				var vm =this;
					params={
						XnList:[],
						XnBlacklist:[],
						anrEnable:'0',
						interFeqEnable:'0',
						EUTRANEnable:'0',
						BiNRCellEnable:'0',
						MRTriggerType:'0',
						absoluteThreshold:'',
						relativeThreshold:'',
						absEnable:'0',
						kpiPeriod:'',
						autoAdjustEnable:'0',
						autoRemoveEnable:'0',
						autoRemovePeriod:'',
						autoRemoveMaxCell:'',
						maxHOtimes:'',
						maxHOSuccess:'',
						syncSource:'',
						SyncMode:'',
						CurrentSyncSource:'-',
						GPSSyncStatus:'1',
						HardwareVersion:'',
						SoftwareVersion:'',
						SyncPercent:'',
						NumberOfSatellite:'',
						HalobEnable:'',
						HalobMode:'',

						gnbLength:'',
						gnbName:'',
						gnbId:'',
						adminState:'',

						multiPlmnEnable:'0',

						CU_F1_C_Local_IP:'',
						CU_F1_U_Local_IP:'',
						CU_NG_C_Local_IP:'',
						CU_NG_U_Local_IP:'',

						DU_F1_C_Local_IP:'',
						DU_F1_U_Local_IP:'',
						DU_F1_C_Remote_IP:'',

						AMFList:[],

						lgwEnable:'0',
					};
				Object.assign(vm.ruleForm,params);
			},
			// QSS syncSource 点击事件
			syncSourceItemChange(val){
				var vm = this;
					syncSourceSelectList = vm.syncSourceSelectList;
				if(val == 'GLONASS'){
					vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
						return items != 'BEIDOU';
					})
				}else if(val == 'BEIDOU'){
					vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
						return items != 'GLONASS';
					})
				}
				vm.ruleForm.syncSource = vm.syncSourceSelectList.join('_');
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
		},
		created(){},
		mounted(){
			this.inits();
			//eventBus.$off('gnbSetting-init').$on('gnbSetting-init',this.init);
			eventBus.$off('open-addNrCell').$on('open-addNrCell',this.openAddNrCellSlide);
			eventBus.$off('open-addTableList').$on('open-addTableList',this.openAddTableListSlide);
			eventBus.$off('close-sharingSlide').$on('close-sharingSlide',this.sharingSlideCancel);
		}
	});
</script>