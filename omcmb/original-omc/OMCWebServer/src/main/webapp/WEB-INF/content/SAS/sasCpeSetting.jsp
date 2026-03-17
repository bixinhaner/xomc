<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>
<style>
	.headerCls{
		border-bottom: 1px solid #e9e9e9;
		position: relative;
	}
	.titleStyML{
		margin-left: 50px;
		margin-top: 40px;
	}
	.stepsCls{
		width: 1000px;
		position: relative;
		margin-bottom: 30px;
	}
	.el-step__head .el-icon::before{
		font-size: 36px!important;
	}
	.el-step__head.is-finish{
		color:#67C23A !important;
		border-color: #67C23A !important;
	}
	.el-step__head.is-process{
		color:#409EFF !important;
		border-color: #409EFF !important;
	}
	.el-step__head.is-wait{
		color:#C0C4CC !important;
		border-color: #C0C4CC !important;
	}
	.el-step__head.is-finish .el-icon::before{
		color:#67C23A !important;
		border-color: #67C23A !important;
	}
	.el-step__head.is-process .el-icon::before{
		color:#409EFF !important;
	}
	.el-step__head.is-wait .el-icon::before{
		color:#C0C4CC !important;
	}
	.el-step__title.is-finish, .el-step__title.is-process,.el-step__title.is-wait{
		color:#444 !important;
	}
	.descriptionRegisteredCls{
		width: 150px;
		background-color: #F9E6E6;
		padding: 10px 10px;
		color: #666666;
	}
	.descriptionGrantedCls{
		width: 180px;
		background-color: #FFF7E0;
		padding: 10px 10px;
		color: #666666;
	}
	.descriptionAuthorizedCls{
		width: 180px;
		background-color: #EBF4E8;
		padding: 10px 10px;
		color: #666666;
	}
	.installParamInfo{
		position: relative;
	}
	.installParamInfo .el-input__inner{
		height: 26px!important;
	}
	.mainBox{
		display: flex;
		flex-wrap: wrap;
		margin: 0px 50px 0px 40px;
	}
	.installParamMain .preferredSettingCls{
		padding-left: 20px;
		min-height: 390px;
		min-width: 600px;
		margin-right: 20px;
		margin-bottom: 22px;
		flex: 2;
	}
	.basicInfoAndAirAndGroupCls{
		display:flex;
		min-width: 900px;
		flex-wrap: wrap;
		flex: 1;
	}
	.basicInfoCls{
		padding-left: 20px;
		min-height: 390px;
		min-width: 400px;
		margin-right: 20px;
		margin-bottom: 22px;
		flex: 1;
	}
	.airAndGroupCls{
		min-height: 390px;
		min-width: 400px;
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	.airIntertaceCls{
		padding-left: 20px;
		min-height: 120px;
		min-width: 400px;
		margin-bottom: 22px;
		flex: 1;
		box-sizing: border-box;
	}
	.groupCls{
		padding-left: 20px;
		min-height: 160px;
		min-width: 400px;
		margin-bottom: 22px;
		flex: 3;
		box-sizing: border-box;
	}
	.groupCls .el-input.is-disabled .el-input__inner{
		overflow:hidden;
		white-space:nowrap;
		text-overflow:ellipsis;
	}
	.installationParameterCls{
		min-height: 340px;
		min-width: 1000px;
		margin: 0px 50px 22px 40px;
	}
	.cpiSignatureDataCls{
		padding-left: 20px;
		padding-right: 110px;
		min-height: 160px;
		min-width: 800px;
		margin: 0px 50px 60px 40px;
	}
	.boxBorderCls{
		border: 1px solid #E9E9E9;
		border-top: 2px solid #4D84FF;
		border-radius: 4px;
	}
	.installParamMain .el-form-item__label{
		font-size: 12px;
	}
	.installParamMain .el-form-item__error{
		top:unset!important;
	}
	.installParamMain .el-form-item__content{
		margin-top: 7px;
	}
	.installParamMain .el-form-item{
		margin-bottom: 13px;
	}
	.btnEditBox{
		width:70px;
		height:24px;
		line-height: 24px;
		border:#DCDFE6 solid 1px;
		float: left;
		text-align: center;
		color:#333333;
		border-radius: 2px;
		cursor:pointer
	}
	.btnEditBox[disabled="disabled"]{
		cursor:not-allowed;
	}
	.installParamMain .btnOver{
		border:#1913BB solid 1px;
		background:#F2F2FC;
		color: #1913BB
	}
	.grouAndIdContainer{
		height:24px;
		min-width:320px;
		border:1px solid #CDE5F7;
		background:#F2F6FF;
		display:table;
		line-height:24px;
		padding-left:5px;
		margin-bottom:20px;
		align-items:center;
		margin-right: 20px;
	}
	.delGroupId{
		display:inline-block;
		width:20px;
		height:20px;
		margin-top: 3px
	}
	.iconFontCls{
		display: inline-block;
		margin-left: 10px;
		margin-top: 5px;
	}
	.iconFontCls .el-icon:before{
		font-size: 18px;
	}
	.pr{
		position:relative
	}
	.antenna{
		font-size: 14px;
		font-weight: bold;
		margin-top: 15px
	}
	.antenna i{
		width: 5px;
		height:5px;
		border-radius: 50%;
		background: #000;
		margin:6px 5px 0px 0px;
		float:left
	}
	.btnBox{
		width:70px;
		height:24px;
		line-height: 24px;
		border:#DCDFE6 solid 1px;
		text-align: center;
		color:#CCCCCC;
		background: #f5f7fa;
		border-radius: 2px;
	}
	.instalInput{
		display:inline-block;
		width: 400px
	}
	.w100{
		width: 100px;
	}
	.antennaCls{
		border-bottom:#E9E9E9 solid 1px;
		padding-left:20px;
	}
	.parameterFromCls{
		margin: 20px;
		text-align: end;
		margin-right: 110px;
	}
	.dilogBox .btnOver{
		border:#1913BB solid 1px;
		background:#F2F2FC;
		color: #1913BB
	}
	.dilogBox .el-form-item__error{
		margin-left:95px
	}
	.parameterBox .el-form-item__label{
		float:left;
		margin-top: 6px;
		margin-right: 10px;
		width:85px
	}
	.modelInstalInput{
		display:inline-block;
		width: 270px;
		vertical-align:middle;
	}
	.el-icon-circle-info:before{
		font-size: 18px;
		position: relative;
		top:2px;
		margin-right: 5px
	}
	.ml30{
		margin-left:50px
	}
	.mt25{
		margin-top: 25px;
	}
	.mt30{
		margin-top: 30px;
	}
	.cpiInfoForm .el-form-item__label{
		float: left;
		margin: 6px 0 10px 0;
		width: 100px
	}
	.importBox-fileSlect{
		display:inline-block;
		height:20px;
		width:20px;
		margin-top:4px;
		cursor:pointer;
	}
	.dilogBox .passwordError .el-form-item__error{
		margin-left: 155px !important;
	}
	.gzstr{
		margin-left:5px;
		color:#CCCCCC
	}
	.footerslide{
		height: 50px;
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: flex-end;
		border: 1px solid #E9E9E9;
		position: absolute;
		bottom:0px;
		box-sizing: border-box;
	}
	.dilogBox .el-form-item__error{
		margin-left:95px
	}
	.wLine{
		width: 98%;
		background: #E9E9E9;
		height: 1px;
		margin:0px auto
	}
	.disabPlus .el-icon-plus:before{
		color:#C9DAFF
	}
	.pageFooter{
		height:55px;
		border-top:1px solid #EEEEEE;
		padding:15px 0 15px 40px;
		box-sizing:border-box;
		background:#ffffff;
		position: fixed;
		bottom:0px;
		width: 100%;
	}
	.dialogFooter{
		height:55px;
		border-top:1px solid #EEEEEE;
		padding:15px 0 15px 40px;
		box-sizing:border-box;
		position: absolute;
		bottom:0px;
		width:540px;
	}
	.dialogFooter .el-button, .footerBtn{
		width:86px;
		height:25px;
		border-radius:unset;
		padding:0px;
	}
	.grantSuspendedCls{
		height: 100px;
		width: 400px;
		display: flex;
		align-items: flex-end;
		position: relative;
		margin-left: 150px;
		overflow: hidden;
		margin-bottom: 30px;
	}
	.left_line{
		width: 200px;
		height: 2px;
		background: #e9e9e9;
		transform: rotate(-30deg)
	}
	.center_line{
		width: 50px;
		height: 2px;
		background: #e9e9e9;
		transform: rotate(90deg)
	}
	.right_line{
		width: 200px;
		height: 2px;
		background: #e9e9e9;
		transform: rotate(30deg)
	}
	.grantSuspendedIconBtn{
		position: absolute;
		left: 180px;
		top:5px;
		display: flex;
		flex-direction: column;
		align-items: center;
		width: 40px;
		text-align: center;
		z-index: 666;
	}
	.grantSuspendedIconBtn .el-icon::before{
		font-size: 36px!important;
		
	}
	.popoverClass{
		padding: 0px!important;
	}
	.severityCls >div{
		height: 28px;
		width: 140px;
		display: flex;
		padding-left: 10px;
		align-items: center;
		cursor: pointer;
	}
	.severityCls >div:hover{
		background-color: #E9E9E9;
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
	.el-popper[x-placement^=bottom]{
		margin-top: 0px;
	}
	.groudIdStr{
		max-width:100px;
		white-space:nowrap;
		text-overflow:ellipsis;
		overflow:hidden;
		margin-right:20px;
	}
	.prompt_one{
		display:inline-block;
		color: #999999
	}
	.prompt_one .el-icon::before{
		color:#CCCCCC;
	}
	.prompt_two{
		padding-left: 20px;
		color:#4D84FF;
	}
	.registrationMethodBoxCls{
		display: inline-block;
		margin: 0px 5px 0px 20px;
	}
	.heightUnitBox .el-select .el-input__inner,.heightUnitBox .el-select .el-input{
		width: 56px;
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
	.sasUserIdDisPopperCls{
		padding: 5px;
	}
	.sasUserIdDisPopperCls .el-button{
		min-width: 60px;
		padding: 0 10px;
		box-sizing: border-box;
	}
	.noUserIdErrBoxCls{
		color: red;
	}
	.noUserIdErrBoxCls a{
		text-decoration: underline;
		padding: 0px 10px;
	}
	.settingBoxForm{
		margin:30px 30px 50px 30px;
	}
	.settingBoxForm .el-form-item__label{
		line-height: 24px;
		width: 150px
	}
	.settingBoxForm .el-input__inner{
		height: 26px !important;
	}
	.GPSInfoBoxCls{
		height: 70px;
		border: 1px solid #E9E9E9;
		padding-top: 20px;
		padding-left: 20px;
		box-sizing: border-box;
		margin-bottom: 20px;
		position: relative;
		margin-right: 20px;
	}
	.GPSInfoBoxCls .el-form-item{
		margin-bottom: 0px;
	}
	.GPSInfoBoxCls .modelInstalInput{
		display:inline-block;
		width: 300px;
		vertical-align:middle;
	}
	.mult-select .el-input{
		width: 80px;
	}
	.preferredPowerItemCls .el-select>.el-input{
		width: 94px!important;
	}
	.sasHeadTitleBox{
		height: 50px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 14px;
		color:#7A7992;
		font-weight: bold;
		padding: 0px 20px;
		border-bottom: 1px solid #E9E9E9;
	}
	.sasSettingMainBox{
		height: 100%;
		width: 100%;
	}
	.sasSettingMainBox >div{
		position: relative;
		height: 100%;
	}
	.w230{
		width: 230px;
	}
	.sasImportCertDialogBox{
		margin: 0px 20px 50px 20px;
		padding: 30px;
		border: 1px solid #E9E9E9;
	}
	.sasImportCertDialogBox .el-form-item{
		width: 42%;
		margin-right:50px; 
		display: inline-block;
	}
	#sasImportCert .el-dialog__body{
		padding-left: 0px;
		padding-right: 0px;
	}
	.queryTimeCls .el-button--text{
		display: none;
	}
</style>
<div class="pageDefault" id='sasInstallParam' v-loading="loading" style="border:none;">
	<div class="sasSettingMainBox">
		<div v-show="!showWindowInfo">
			<div class="container">
				<div class="sasHeadTitleBox">
					<span><%=rb.getString("SheZhi")%></span>
					<span class="el-icon el-icon-circle-close" @click="closeInfoparams"></span>
				</div>
				<div class="headerCls">
					<div class="group-title not-extend titleStyML" >
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("SASGuoChengZhuangTai")%></span>
					</div>
					<div style="display: flex;flex-direction: column;align-items: center;" >
						<div class="grantSuspendedCls" v-if="stepsAction == '4'">
							<div class="grantSuspendedIconBtn" id="statePopover_4">
								<span  :class="setProcedureStatePopover.statePopover_4_icon"></span>
								<el-popover placement='bottom'  trigger='hover' popper-class="popoverClass" v-model="setProcedureStatePopover.statePopover_4">
									<div class="severityCls">
										<div @click="setProcedureState('3','statePopover_4')">Heartbeat req</div>
										<div @click="setProcedureState('4','statePopover_4')">Relinquishment req</div>
										<div @click="setProcedureState('1','statePopover_4')">Deregister req</div>
									</div>
									<span slot="reference" v-if="stepsAction == 4">Grant Suspended</span>
								</el-popover>
								<span v-if="stepsAction != 4">Grant Suspended</span>
							</div>
							<div class="left_line"></div>
							<div class="center_line"></div>
							<div class="right_line"></div>
						</div>
						<div class="stepsCls">
							<el-steps :active="stepsAction">
								<el-step  :icon="setProcedureStatePopover.statePopover_1_icon" id="statePopover_1" style="flex:1">
									<div slot="title">
										<el-popover placement='bottom'  trigger='hover' popper-class="popoverClass" v-model="setProcedureStatePopover.statePopover_1" v-if="stepsAction == 0 && notVirtualCBSD && stepsActive">
											<div class="severityCls">
												<div @click="setProcedureState('0','statePopover_1')">Register req</div>
											</div>
											<span style="cursor: pointer;" slot="reference" v-if="stepsAction == 0 && notVirtualCBSD && stepsActive">Unregistered</span>
										</el-popover>
										<span v-if="stepsAction != 0 || !notVirtualCBSD || !stepsActive">Unregistered</span>
									</div>
								</el-step>
								<el-step  :icon="setProcedureStatePopover.statePopover_2_icon" id="statePopover_2" style="flex:1">
									<div slot="title">
										<el-popover placement='bottom'  trigger='hover' popper-class="popoverClass" v-model="setProcedureStatePopover.statePopover_2" v-if="stepsAction == 1 && notVirtualCBSD && stepsActive">
											<div class="severityCls">
												<div @click="setProcedureState('1','statePopover_2')">Deregister req</div>
												<div @click="setProcedureState('2','statePopover_2')">Grant req</div>
											</div>
											<span style="cursor: pointer;" slot="reference" v-if="stepsAction == 1 && notVirtualCBSD && stepsActive">Registered</span>
										</el-popover>
										<span v-if="stepsAction != 1 || !notVirtualCBSD || !stepsActive">Registered</span>
									</div>
									<div class="descriptionRegisteredCls" slot="description" v-show="false">
										<div>CBSD State</div>
										<div>No EARFCN Configured</div>
									</div>
								</el-step>
								<el-step :icon="setProcedureStatePopover.statePopover_3_icon" id="statePopover_3" style="flex:1">
									<div slot="title">
										<el-popover placement='bottom'  trigger='hover' popper-class="popoverClass" v-model="setProcedureStatePopover.statePopover_3" v-if="stepsAction == 3 && notVirtualCBSD && stepsActive">
											<div class="severityCls">
												<div @click="setProcedureState('3','statePopover_3')">Heartbeat req</div>
												<div @click="setProcedureState('4','statePopover_3')">Relinquishment req</div>
											</div>
											<span style="cursor: pointer;" slot="reference" v-if="stepsAction == 3 && notVirtualCBSD && stepsActive">Granted</span>
										</el-popover>
										<span v-if="stepsAction != 3 || !notVirtualCBSD || !stepsActive">Granted</span>
									</div>
									<div class="descriptionGrantedCls" slot="description" v-show="false">
										<div>CBSD State</div>
										<div>EARFCN Configured & Tx OFF</div>
									</div>
								</el-step>
								<el-step title="" icon="el-icon el-icon-operation-defaultBeta" v-show="false"></el-step>
								<el-step title="" icon="el-icon el-icon-operation-defaultBeta" v-show="false"></el-step>
								<el-step :icon="setProcedureStatePopover.statePopover_5_icon" id="statePopover_5">
									<div slot="title">
										<el-popover placement='bottom'  trigger='hover' popper-class="popoverClass" v-model="setProcedureStatePopover.statePopover_5" v-if="stepsAction == 5 && notVirtualCBSD && stepsActive">
											<div class="severityCls">
												<div @click="setProcedureState('3','statePopover_5')">Heartbeat req</div>
												<div @click="setProcedureState('4','statePopover_5')">Relinquishment req</div>
												<div @click="setProcedureState('1','statePopover_5')">Deregister req</div>
											</div>
											<span style="cursor: pointer;" slot="reference" v-if="stepsAction == 5 && notVirtualCBSD && stepsActive">Authorized</span>
										</el-popover>
										<span v-if="stepsAction != 5 || !notVirtualCBSD || !stepsActive">Authorized</span>
									</div>
									<div class="descriptionAuthorizedCls" slot="description" v-show="false">
										<div>CBSD State</div>
										<div>EARFCN Configured & Tx ON</div>
									</div>
								</el-step>
							</el-steps>
						</div>
					</div>
				</div>
				<div class="installParamInfo">
					<div class="group-title not-extend titleStyML" style="margin-bottom:24px;">
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("AnZhuangCanShu")%></span>
						<div class="prompt_one">
							<span class="el-icon el-icon-circle-info" style="margin-left:20px;"></span>
							<span>The following parameters are mandatory for successful registration with SAS.</span>
						</div>
						
					</div>
					<div v-if="notVirtualCBSD" class="circleIcon placeholder-bt" placeholder="<%=rb.getString("TongBu")%>"  style='margin-right:60px;top:-10px' v-show='isEdit'>		
						<span class="el-icon-circle-refresh el-icon" @click="syncParams"></span>
					</div>
					<div class="installParamMain">
						<el-form :model="ruleForm" :rules="rules" ref="ruleForm" :disabled="disableForm" label-position='left'>
							<div class="mainBox">
								<div class="preferredSettingCls boxBorderCls">
									<div class="group-title not-extend" style="margin:20px 0px;">
										<span class="title-text">Preferred Settings</span>
									</div>
									<div>
										<el-form-item  label="EIRP" required prop="manualMaxEirp" label-width='180px'>
											<el-input v-model="ruleForm.manualMaxEirp" ></el-input>
										</el-form-item>
										<el-form-item label="PreferFreqHigh" prop="manualHighFreq" label-width='180px'>
											<el-input :disabled="isEdit" placeholder="Range:3550-3700(MHz) Integer"  v-model="ruleForm.manualHighFreq"></el-input>
										</el-form-item>
										<el-form-item label="PreferFreqLow" prop="manualLowFreq" label-width='180px'>
											<el-input :disabled="isEdit" placeholder="Range:3550-3700(MHz) Integer" v-model="ruleForm.manualLowFreq" ></el-input>
										</el-form-item>
									</div>
								</div>
								<div class="basicInfoAndAirAndGroupCls">
									<div class="basicInfoCls boxBorderCls">
										<div class="group-title not-extend" style="margin:20px 0px;">
											<span class="title-text">Basic Information</span>
										</div>
										<el-form-item label="CBSD Category" prop="cbsdCategory" label-width='130px'>
											<div v-model='ruleForm.cbsdCategory'>
												<div v-if="disableForm" class="btnEditBox" :disabled="disableForm" :class="{btnOver:item.value == ruleForm.cbsdCategory}"  v-for="(item,index) in cbsdCategoryData">{{item.label}}</div>
												<div v-if="!disableForm" class="btnEditBox" :class="{btnOver:item.value == ruleForm.cbsdCategory}" @click="cbsdChange(index,item)"  v-for="(item,index) in cbsdCategoryData">{{item.label}}</div>
											</div>
										</el-form-item>
										<el-form-item label="User ID" prop="userId" required label-width='130px'>
											<el-input ref="userid" placeholder="length：0 - 256"  v-model="ruleForm.userId"  maxlength='256' class=""></el-input>
											<div class="cellNameClass"  v-if="sasUserIdDisIcon && !disableForm">
												<el-popover placement="top" width="240" trigger="click" v-model="sasUserIdDisVisible" popper-class="sasUserIdDisPopperCls">
													<p style="padding:5px"><%=rb.getString("UserIdBuTongTiShi")%></p>
													<p style="color:#333333;">OMC User ID：{{OMCSasUserId}}</p>
													<div style="text-align:right;margin:0;">
														<el-button size="mini" type="text" @click="sasUserIdDisVisible = false"><%=rb.getString("QuXiao")%></el-button>
														<el-button size="mini" type="primary" @click="sasUserIdReplace"><%=rb.getString("QueDing")%></el-button>
													</div>
													<div v-if="sasUserIdDisIcon" style="cursor: pointer;" slot="reference">
														<span class="el-icon el-icon-circle-warning" v-if="sasUserIdDisIcon"></span> 
													</div>
												</el-popover>
											</div>
										</el-form-item>
										<el-form-item label="Call Sign" prop="callSign" label-width='130px'>
											<el-input placeholder="length：0 - 256"  v-model="ruleForm.callSign" maxlength='256' class=""></el-input>
										</el-form-item>
										<el-form-item label="FCC ID" prop="fccId" required label-width='130px'>
											<el-input ref="fccid" placeholder="length：0 - 19" :disabled="listType == 'CPE' || fccIdIsNoMod"  v-model="ruleForm.fccId" @change="fccIdChange"  class="" maxlength='19'></el-input>
										</el-form-item>
										<el-form-item label="Serial Number" required prop="serialNumber" label-width='130px'>
											<el-input ref="serialnum" v-model="ruleForm.serialNumber" :disabled="isEdit"></el-input>
										</el-form-item>
										<el-form-item label="Antenna Gain" prop="antennaGain" label-width='130px'>
											<el-input ref="antennaGain" v-model="ruleForm.antennaGain" :disabled="hasCode"></el-input>
										</el-form-item>
									</div>
									<div class="airAndGroupCls">
										<div class="airIntertaceCls boxBorderCls">
											<div class="group-title not-extend" style="margin:20px 0px;">
												<span class="title-text">Air Interface</span>
											</div>
											<el-form-item label="Radio Technology" prop="supportSpec" label-width='130px'>
												<el-input  v-model="ruleForm.supportSpec" :disabled="isEdit"></el-input>
											</el-form-item>
										</div>
										<div class="groupCls boxBorderCls">
											<div class="group-title not-extend" style="margin:20px 0px;">
												<span class="title-text">Group</span>
												<span style="color:#4D84FF">(Limit of 5)</span>
											</div>
											<el-form-item  label="Group Type " prop="groupType" label-width='130px'>
												<el-select v-model="groupType">
													<el-option v-for="(item,index) in groupTypeOption" :key="item.value" :value="item.value" :label="item.label"></el-option>
												</el-select>
											</el-form-item>
											<el-form-item  label="Group ID" prop="groupId" label-width='130px'>
												<el-input v-model="OMCSasUserId" style="width:60px;" :disabled="true" :title='OMCSasUserId'></el-input>
												<el-input v-model="ruleForm.groupId" placeholder="number , letter and _" style="width:180px;margin-left:-5px;"></el-input>
												<template v-if="!disableForm">
													<div v-if="ruleForm.groupIdAndGroupType.length>4" class="disabPlus" style="display:inline-block;">
														<span class="el-icon el-icon-plus" style="margin-left:10px;" ></span>
													</div>
													
													<span class="el-icon el-icon-plus" style="margin-left:10px;" v-else @click="addGroupTypeAndId"></span>
												</template>
												<div class="noUserIdErrBoxCls" v-if="noUserIdErrShow">
													<%=rb.getString("WeiSheZhiUserId")%>
												</div>
											</el-form-item>
											
											<div style="display:flex;flex-wrap: wrap">
												<div class="grouAndIdContainer" v-for="(item,index) in ruleForm.groupIdAndGroupType" :key="index">
													<div style="display:flex;">
														<div style="margin:0px 20px 0px 10px"> {{item.groupType}}</div>
														<div class="groudIdStr" :title="item.groupId">{{OMCSasUserId}}_{{item.groupId}}</div>
														<div v-if="!disableForm"><span @click="delGroupTypeAndId(index)" class="el-icon el-icon-operation-delete delGroupId" ></span></div>
													</div>  
												</div>
											</div>
										</div>
									</div>
								</div>
							</div>
							<div class="installationParameterCls boxBorderCls">
								<div class="group-title not-extend" style="margin:20px 0px;padding-left:20px;">
									<span class="title-text">Installation Parameter</span>
								</div>
								<div class="prompt_two" v-if="ruleForm.cbsdCategory == 'A'">
									{{singleOrMultiTip_Category_A}} registration is currently selected.
									<span class="registrationMethodBoxCls" v-show="registrationMethodShow">
										<el-checkbox v-model="ruleForm.plaintextMode" true-label="true" false-label="false" :disabled="registrationMethodDis"></el-checkbox>
									</span>
									<span  v-show="registrationMethodShow">Installation Parameter will be include in RegisterRequest</span>
									<el-tooltip placement="top" content="Configuring Installparameters to CBSD may cause CBSD to recalculate CPI CODE">
										<el-checkbox style="margin:0px 5px 0px 20px;" v-model="ruleForm.saveInstallParamOnlyOnDP" true-label="true" false-label="false"></el-checkbox>
									</el-tooltip>
									<span>Save Installparameters only on DP and not on CBSD</span>
								</div>
								<div class="prompt_two" v-if="ruleForm.cbsdCategory == 'B'">
									{{singleOrMultiTip_Category_B}} registration is currently selected.
									<el-tooltip placement="top" content="Configuring Installparameters to CBSD may cause CBSD to recalculate CPI CODE">
										<el-checkbox style="margin:0px 5px 0px 20px;" v-model="ruleForm.saveInstallParamOnlyOnDP" true-label="true" false-label="false"></el-checkbox>
									</el-tooltip>
									<span>Save Installparameters only on DP and not on CBSD</span>
								</div>
								<div class="pr">
									<div style="margin:10px 0px 0px 20px;">
										<el-form-item label="Deployment" prop="indoor" style="margin-bottom:15px" label-width='100px'>
											<div v-model='ruleForm.indoor'>
												<div v-if="ruleForm.indoor == 'false'" class="btnBox btnOver" :disabled="true" >Outdoor</div>
												<div v-if="ruleForm.indoor == 'true'" class="btnBox btnOver" :disabled="true" >Indoor</div>
											</div>
										</el-form-item>
									</div>
									<div v-for="(item,index) in ruleForm.cellSpecificParams">
										<div class="antennaCls">
											<div class="antenna"> <i></i> Antenna-{{index+1}}</div>
											<div style="display:flex;">
												<el-form-item label="Latitude" :prop="'cellSpecificParams['+index+'].latitude'" :rules="rules.latitude" class="instalInput" label-width='100px'>
													<el-input v-model='item.latitude' :disabled="true" class="w100">{{item.latitude}}</el-input>
												</el-form-item>
												<el-form-item label="Longitude" :prop="'cellSpecificParams['+index+'].longitude'" class="instalInput" label-width='100px'>
													<el-input v-model='item.longitude' :disabled="true" class="w100">{{item.longitude}}</el-input>
												</el-form-item>
												<el-form-item label="Height" :prop="'cellSpecificParams['+index+'].antennaHeight'" class="instalInput heightUnitBox" label-width='100px'>
													<el-input v-model='item.antennaHeight' :disabled="true" class="w100">{{item.antennaHeight}}</el-input>
													<el-select v-model="item.heightUnit" :disabled="true" style="display:inline-block;margin-left:-5px">
														<el-option v-for="(items,index) in heightUnitList" :key="items.value" :value="items.value" :label="items.label"></el-option>
													</el-select>
													<span v-show="item.antennaHeight" v-html="antennaHeightFmt(item)"></span>
												</el-form-item>
												<el-form-item label="HeightType" :prop="'cellSpecificParams['+index+'].antennaHeightType'" class="instalInput tabSelectBox" label-width='100px'>
													<div v-model="item.antennaHeightType" size="small" :disabled="true">
														<div  class="btnBox btnOver" :disabled="true" >{{item.antennaHeightType}}</div>
													</div>
												</el-form-item>
											</div>
											<div style="display:flex;">
												<el-form-item label="Azimuth" :prop="'cellSpecificParams['+index+'].antennaAzimuth'" class="instalInput" label-width='100px'>
													<el-input v-model='item.antennaAzimuth' :disabled="true" class="w100">{{item.antennaAzimuth}}</el-input>
												</el-form-item>
												<el-form-item label="Down Tilt" :prop="'cellSpecificParams['+index+'].antennaDowntilt'" class="instalInput" label-width='100px'>
													<el-input v-model='item.antennaDowntilt' :disabled="true" class="w100">{{item.antennaDowntilt}}</el-input>
												</el-form-item>
												<el-form-item label="Gain" :prop="'cellSpecificParams['+index+'].antennaGain'" class="instalInput" label-width='100px'>
													<el-input v-model='item.antennaGain' :disabled="true" class="w100">{{item.antennaGain}}</el-input>
												</el-form-item>
												<el-form-item label="Beamwidth" :prop="'cellSpecificParams['+index+'].antennaBeamwidth'" class="instalInput" label-width='100px'>
													<el-input v-model='item.antennaBeamwidth' :disabled="true" class="w100">{{item.antennaBeamwidth}}</el-input>
												</el-form-item>
											</div>
										</div>
									</div>
									<div class="group-title not-extend" style="margin:20px 0px;padding-left:20px;">
										<span class="title-text">Professional Installer Data</span>
									</div>
									<div style="padding-left:20px;">
										<el-form-item label="CPI ID" prop="cpiId" class="instalInput" label-width='100px'>
											<el-input v-model="ruleForm.cpiId" :disabled="true" class="w100">{{ruleForm.cpiId}}</el-input>
										</el-form-item>
										<el-form-item label="CPI Name" prop="cpiName" class="instalInput" label-width='100px'>
											<el-input v-model="ruleForm.cpiName" :disabled="true" class="w100">{{ruleForm.cpiName}}</el-input>
										</el-form-item>
										<el-form-item label="Install Cert Time" prop="cpiInstallCertificationTime" class="instalInput"  label-width='120px'>
											<el-input v-model="ruleForm.cpiInstallCertificationTime" :disabled="true" class="">{{ruleForm.cpiInstallCertificationTime}}</el-input>
										</el-form-item>
									</div>	
									<div v-if="!disableForm" class="parameterFromCls">
										<el-button type="primary"  @click='parameterFrom'><%=rb.getString("XiuGai")%></el-button>
									</div>
								</div>
							</div>
							<div class="cpiSignatureDataCls boxBorderCls">
								<div class="group-title not-extend" style="margin:20px 0px;">
									<span class="title-text">CPI Signature Data</span>
								</div>
								<div v-for="(item,index) in ruleForm.cellSpecificParams">
									<el-form-item  :prop="'cellSpecificParams['+index+'].cpiCertCode'" label="">
										<el-input :rows="5" type="textarea" v-model="item.cpiCertCode" :disabled="true" ></el-input>
									</el-form-item>
								</div>
								<div style="margin:0px 0px 20px 0px;text-align:end;">
									<el-button  @click="clearCode"><%=rb.getString("QingKong")%></el-button>
								</div>
							</div>
						</el-form>
					</div>
				</div>
			</div>
			<div v-if="!disableForm" class="pageFooter" style="z-index:999;">
				<el-button type="primary" @click="updateSettings"><%=rb.getString("QueDing")%></el-button>
				<el-button plain @click="closeInfoparams"><%=rb.getString("QuXiao")%></el-button>
				<div style="display:inline-block;margin-left:20px;" v-if="rowData.sasEnable == 'off'">
					<span style="margin-right:10px;"><%=rb.getString("DingShiZhiXing")%></span>
					<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.scheduleTime' popper-class="queryTimeCls" placeholder="NOW" size="mini" type="datetime" @focus='setTime' :picker-options="executePickerOptions"></el-date-picker>	
				</div>
			</div>
		</div>
		<div v-show="showWindowInfo">
			<el-form :model="ruleFormParameter" :rules="rulesParameter" ref="ruleFormParameter" label-position='top' style="padding-bottom:50px;" class="dilogBox parameterBox">
				<div class="sasHeadTitleBox">
					<span>Installation Parameter Modify</span>
					<span class="el-icon el-icon-circle-close" @click="closeSettingInfo"></span>
				</div>
				<div class="mt25" style="position:relative;margin-left:30px;">
					<el-form-item label="Deployment" prop="indoor" style="margin-bottom:30px">
						<div v-model='ruleFormParameter.indoor'>
							<div class="btnEditBox" :class="{btnOver:items.value == ruleFormParameter.indoor}" @click="changeindoor(index,items)" v-for="(items,index) in modeIndoorData">{{items.label}}</div>
						</div>
					</el-form-item>
				</div>
				<div v-for="(item,index) in ruleFormParameter.cellSpecificParams">
					<div class="antenna" style="margin-left:30px;"> <i></i> Antenna-{{index+1}} </div>
					<div class="ml30" style="margin-top:30px;">
						<div class="GPSInfoBoxCls">
							<div style="position:absolute;left:20px;top:-13px;padding:0 10px;background-color: #FFFFFF;">
								<el-button type="primary" @click="syncdeviceGps(index)">Auto</el-button>
								<span style="color:#999999;margin-left:5px;"><%=rb.getString("ZiDongTongBuTianChongTiShi")%></span>
							</div>
							<el-form-item label="Latitude" :prop="'cellSpecificParams['+index+'].latitude'" :rules="rulesParameter.latitude" class="modelInstalInput">
								<el-input v-model.trim="item.latitude" class="w100">{{item.latitude}}</el-input>
							</el-form-item>
							<el-form-item label="Longitude" :prop="'cellSpecificParams['+index+'].longitude'" :rules="rulesParameter.longitude" class="modelInstalInput">
								<el-input v-model.trim="item.longitude" class="w100">{{item.longitude}}</el-input>
							</el-form-item>
							<el-form-item label="Height" :prop="'cellSpecificParams['+index+'].antennaHeight'" :rules="rulesParameter.antennaHeight" class="modelInstalInput heightUnitBox" style="white-space: nowrap">
								<el-input v-model.trim="item.antennaHeight" class="w100">{{item.antennaHeight}}</el-input>
								<el-select v-model="item.heightUnit"  style="display:inline-block;margin-left:-5px">
									<el-option v-for="(items,index) in heightUnitList" :key="items.value" :value="items.value" :label="items.label"></el-option>
								</el-select>
								<span v-show="item.antennaHeight" v-html="antennaHeightFmt(item)"></span>
							</el-form-item>
						</div>
					</div>
					<div class="ml30">
						<el-form-item label="HeightType" :prop="'cellSpecificParams['+index+'].antennaHeightType'" :rules="rulesParameter.antennaHeightType" class="modelInstalInput tabSelectBox">
							<div v-model="item.antennaHeightType" size="small">
								<div class="btnEditBox" :class="{btnOver:items.value==item.antennaHeightType}" @click="changeHeightType(1,index,items)" v-for="(items,indexs) in modeHeightTypeData">{{items.label}}</div>
							</div>
						</el-form-item>
						<el-form-item label="Azimuth" :prop="'cellSpecificParams['+index+'].antennaAzimuth'" :rules="rulesParameter.antennaAzimuth" class="modelInstalInput" >
							<el-input  v-model="item.antennaAzimuth" class="w100">{{item.antennaAzimuth}}</el-input>
						</el-form-item>
						<el-form-item label="Down Tilt" :prop="'cellSpecificParams['+index+'].antennaDowntilt'" :rules="rulesParameter.antennaDowntilt" class="modelInstalInput" >
							<el-input  v-model="item.antennaDowntilt" class="w100">{{item.antennaDowntilt}}</el-input>
						</el-form-item>
					</div>
					<div class="ml30">
						<el-form-item label="Gain" :prop="'cellSpecificParams['+index+'].antennaGain'" :rules="rulesParameter.antennaGain" class="modelInstalInput">
							<el-input  v-model="item.antennaGain" class="w100">{{item.antennaGain}}</el-input>
						</el-form-item>
						<el-form-item label="Beamwidth" :prop="'cellSpecificParams['+index+'].antennaBeamwidth'" :rules="rulesParameter.antennaBeamwidth" class="modelInstalInput">
							<el-input  v-model="item.antennaBeamwidth" class="w100">{{item.antennaBeamwidth}}</el-input>
						</el-form-item>
					</div>
				</div>
			</el-form>
			<div class='pageFooter'>	
				<el-button type="primary" @click="saveInstallParamsModify"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeSettingInfo"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</div>
	<!--证书导入弹出框-->
	<el-dialog title="<%=rb.getString("QueRen")%>" id="sasImportCert" :visible.sync="showSasImportCert" ref="slide" :width="sildeWidth" :append-to-body="true" :height="sildeHeight" 
			:close-on-click-modal="false">
			<div style="margin-bottom:20px;font-size:14px;color:#333333;padding-left:20px;"><%=rb.getString("ZhengShuWeiQianFaTiShi")%></div>
			<div class="sasImportCertDialogBox">
				<div style="font-size:14px;color:#333333;font-weight:bold;margin-bottom:20px;">Verify CPI Identity</div>
				<el-form label-position="top" ref="sasImportCertForm" :model='sasImportCertForm' :rules='sasImportCertFormRules'>
					<el-form-item label="CPI  <%=rb.getString("ZhengShu")%>">
						 <span slot="label" class="labelSlotCls">
							CPI <%=rb.getString("ZhengShu")%>
							<span>( Only .P12/.PEM are supported )</span>
						</span>
						<el-upload :on-success='checkFile' :on-change="fileChangePrivateKey" :show-file-list=false ref="uploadCert" :action="sasImportCertForm.uploadFileURL"
							:auto-upload="false">
							<el-input :readonly="true" :value="sasImportCertForm.fileName" class="w230">
								<a slot="append" class="el-icon el-icon-operation-import greyIcon" @click="fileSelectPrivateKey"></a>
							</el-input>
							<el-checkbox style="margin-left:10px;" v-model="sasImportCertForm.p12" true-label="true" false-label="false" @change="p12CheckChange">P12</el-checkbox>
							<div slot="tip" class="el-form-item__error" v-show="sasImportCertForm.errorFlag">
								{{sasImportCertForm.errorMessage}}
							</div>
							<a slot="trigger" ref="file_up"></a>
						</el-upload>
						
					</el-form-item>
					<el-form-item v-show="sasImportCertForm.p12 == 'true'" label="<%=rb.getString("MiMa")%>" prop="password">
						<el-password v-model="sasImportCertForm.password" size="mini" placeholder="" class="w230" maxlength='50' show-password></el-password>
						<el-input v-model="sasImportCertForm.password" style="display: none;"></el-input>
					</el-form-item>
					<el-form-item label="CPI ID" prop='cpiid'>
                        <el-input v-model="sasImportCertForm.cpiid" class="w230" maxlength='50'></el-input>
                    </el-form-item>
                    <el-form-item label="CPI  <%=rb.getString("MingCheng")%>" prop='cpiname'>
                        <el-input v-model="sasImportCertForm.cpiname" class="w230" maxlength='50'></el-input>
                    </el-form-item>
				</el-form>
			</div>
			<div class='footerslide'>	
				<el-button type="primary" @click="importSasCertSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="importSasCertClose" style="margin-right:20px;"><%=rb.getString("QuXiao")%></el-button>
			</div>
	</el-dialog>
	<!--SAS setting-->
	<el-dialog title='<%=rb.getString("SheZhi")%>' :visible.sync="showSasSettingDialog" ref="sasSettingDialog" 
		:close-on-click-modal="false"  @close='closeSasSettingDialog' width="600" :append-to-body="true">
		<div class='setEnb settingBoxForm'>
			<el-form label-position="left" :model="sasSettingForm" ref="sasSettingForm" >
				<el-form-item  label="<%=rb.getString("SASZiDongZhuCeKaiGuan")%>" prop="sasAble" >
					<el-select v-model="sasSettingForm.sasAble" >
						<el-option v-for="item in selectSasAble" :key="item.value" :value="item.value" :label="item.label"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item  label="<%=rb.getString("SASTiGongShang")%>" prop="sasProvider" >
					<el-select v-model="sasSettingForm.sasProvider"  @change="sasProviderChange">
						<el-option v-for="item in selectSasProvider" :key="item.value" :value="item.value" :label="item.label"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label="User ID" prop="sasUserId">
					<el-input v-model="sasSettingForm.sasUserId"  placeholder='' ></el-input>
				</el-form-item>
			</el-form>
		</div>
		<div class="dialogFooter">
			<el-button  type="primary"  @click="submitSASSetting"><%=rb.getString("QueDing")%></el-button>
			<el-button  @click="closeSasSettingDialog"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>
<script type="text/javascript">
var sasSettingsVue = new Vue({
	el:'#sasInstallParam',
	data(){
		var vm = this;
		var validatemanualMaxEirp = (rule,value,callback) => {
			
			if(this.listType == 'CPE') {
				//校验输入的值是否是整数
				var intTest = /^\+?[0-9]*$/;

				if(intTest.test(value)){
					callback()
				}else{
					return callback(new Error("Please input Integer"))
				}
			}else {
				callback()
			}
		}
		var validatemanualHighFreq = (rule,value,callback) => {
			var numReg = /^\s*\d+\s*$/;
			if(this.listType !== 'CPE'){
				if (value){
					if(numReg.test(value) && (value - 3550 >= 0) && (value - 3700 <= 0)) {
						callback();
					}else {
						callback('Range: 3550-3700(MHz).');
					}
				} else {
					callback();
				}
			}else{
				callback();
			}
		}
		//添加groupId 校验
		var validategroupId = (rule,value,callback) => {
			//groupId 不能为空 超过5条不用校验
			var reg = /^\w+$/;
			if(this.methodType == "saveFlag"){ //save方式不校验
				callback()
			}else{
				if(!(this.ruleForm.groupIdAndGroupType.length > 4)){
					if(value == ""){
						return callback(new Error('please input the group ID'))
					}else if(!reg.test(value)){
						callback('<%=rb.getString("GeShiCuoWu")%>');
					}else{
						callback();
					}
				}else{
					callback(new Error('不能超过5条'))
				}
			}
		}
		var fccIdValidator = (rule,value,callback) => {
			var len = value,lenght
			for(var i =0;i< value,lenght;i++){
				if(value.charCodeAt(i)>255){
					len = len + 2
				}else{
					len = len+1
				}
				return len
			}
			if(!value){
				return callback(new Error('Please input the FccID'))
			}else{
				vm.isShowEdit[2] = true
				callback()
			}
			
		}
		//校验userId
		var validateUserId = (rule,value,callback) => {
			if(!value){
				return callback(new Error('Please input the User ID'))
			}else if(value.length > 256){
				return callback(new Error('User ID cannot exceed 256 digits'))
			}else{
				vm.isShowEdit[1] = true
				callback()
			}
		} 
		// 校验longitude校验
		var longitudeValidator = (rule,value,callback) => {
			let regNum = /^(\-|\+)?(((\d|[1-9]\d|1[0-7]\d|0{1,3})\.\d{0,6})|(\d|[1-9]\d|1[0-7]\d|0{1,3})|180\.0{0,6}|180)$/;
			if(value !== '' && regNum.test(value)){
				callback()
			}else{
				callback('<%=rb.getString("FanWei")%>:-180.000000 ~ 180.000000');
			}
		}
		// 校验latitude校验
		var latitudeValidator = (rule,value,callback) => {
			let regNum = /^(\-|\+)?([0-8]?\d{1}\.\d{0,6}|90\.0{0,6}|[0-8]?\d{1}|90)$/;
			if(value !== '' && regNum.test(value)){
				callback()
			}else{
				callback('<%=rb.getString("FanWei")%>:-90.000000 ~ 90.000000');
			}
		}
		// 校验height校验
		var antennaHeightValidator = (rule,value,callback) => {
			var regNum = /^\d+(\.\d+)?$/; // 验证是否是数字
			var fieldArr = rule.field;
				idx = parseInt(fieldArr.substring(fieldArr.indexOf('[')+1,fieldArr.indexOf(']'))),
				cellData = this.ruleFormParameter.cellSpecificParams[idx];
			
			if(value !== '' && regNum.test(value)){
				if(vm.ruleForm.cbsdCategory == 'A' && (vm.ruleFormParameter.indoor == 'false' || vm.ruleFormParameter.indoor == 'Outdoor')){
					if(cellData.heightUnit=="meter"){
						if(parseFloat(value)<= 6){
							callback()
						}else{
							callback('Maximum height 6m');
						}
					}else{
						if(parseFloat(value) <= 19.47){
							callback()
						}else{
							callback('Maximum height 6m');
						}
					}
				}else{
					callback()
				}
				callback()
			}else{
				callback('<%=rb.getString("QingShuRuShuZi")%>');
			}
		}
		//校验antennaAzimuthValidator
		var antennaAzimuthValidator = (rule,value,callback) => {
			var regNum = /^\d+$/; // 验证是否是数字
			if(value !== '' && regNum.test(value) &&(value>=0 && value<= 359)){
				callback()
			}else{
				callback('<%=rb.getString("FanWei")%>:0 ~ 359');
			}
		}
		var antennaDowntiltValidator = (rule,value,callback) => {
			//var regNum = /^\d+$/; // 验证是否是数字
			var regNum = /^-?\d+$/; //正整数+负整数+0，不可有小数点
			if(value !== '' && regNum.test(value) &&(value>=-90 && value<= 90)){
				callback()
			}else{
				callback('<%=rb.getString("FanWei")%>:-90 ~ 90');
			}
		}
		var basicGainValidator1 = (rule,value,callback) => {
			//var regNum = /^\d+$/; // 验证是否是数字
			var regNum = /^-?\d+$/; //正整数+负整数+0，不可有小数点
			var fieldArr = rule.field,
				index = 0;
			if(fieldArr == 'antennaGain'){
				index = 0;
			}
			if(value !== '' && regNum.test(value) &&(value>=-127 && value<= 128)){
				this.ruleForm.cellSpecificParams[index].antennaGain = value;
				callback();
			}else{
				callback('<%=rb.getString("FanWei")%>:-127 ~ 128');
			}
		}
		var antennaGainValidator = (rule,value,callback) => {
			//var regNum = /^\d+$/; // 验证是否是数字
			var regNum = /^-?\d+$/; //正整数+负整数+0，不可有小数点
			if(value !== '' && regNum.test(value) &&(value>=-127 && value<= 128)){
				callback()
			}else{
				callback('<%=rb.getString("FanWei")%>:-127 ~ 128');
			}
		}
		var antennaBeamwidthValidator = (rule,value,callback) => {
			var regNum = /^\d+$/; // 验证是否是数字
			if(value !== '' && regNum.test(value) &&(value>=0 && value<= 360)){
				callback()
			}else{
				callback('<%=rb.getString("FanWei")%>:0 ~ 360');
			}
		}
		var antennaHeightTypeValidator = (rule,value,callback) => {
			var regNum = /^\d+$/; // 验证是否是数字
			if(value !== '' && value !== null && value !== undefined && value !== 'null'){
				callback()
			}else{
				callback('<%=rb.getString("QingXuanZe")%>');
			}
		}
		var passwordValidator = (rule, value, callback) => {
			if(this.sasImportCertForm.p12 == 'true'){
				if (value === '') {
					callback('<%=rb.getString("QingShuRuMiMa") %>');
				} else {
					callback()
				}
			}else{
				callback()
			}
		}
		return {
			stepsAction: 0,
			numPopover:'',
			stepsActive:true,
			setProcedureStatePopover:{
				statePopover_1:false,
				statePopover_2:false,
				statePopover_3:false,
				statePopover_4:false,
				statePopover_5:false,

				statePopover_1_icon:'el-icon el-icon-sas-success',
				statePopover_2_icon:'el-icon el-icon-sas-success',
				statePopover_3_icon:'el-icon el-icon-sas-success',
				statePopover_4_icon:'el-icon el-icon-sas-warning',
				statePopover_5_icon:'el-icon el-icon-sas-success',
			},
			modeIndoorData:[
				{
					value:'true',
					label:'Indoor'
				},{
					value:'false',
					label:'Outdoor'
				}
			],
			modeHeightTypeData:[
				{
					value:'AGL',
					label:'AGL'
				},{
					value:'AMSL',
					label:'AMSL'
				}
			],
			cbsdCategoryData:[
				{
					value:'A',
					label:'A'
				},{
					value:'B',
					label:'B'
				}
			],
			isEdit:false,
			showWindowInfo:false,
			sildeHeight:'660px',
			sildeWidth:'865px',
			p12:true,
			isAdd:'',
			rowData:'',
			height:'660px',
			slideWidth:'',
			slidePosition:'',
			methodType:'',
			groupTypeOption:[
				{value:'INTERFERENCE_COORDINATION',label:'INTERFERENCE_COORDINATION'},
				{value:'PRINCIPAL_SUBORDINATE_SFG',label:'PRINCIPAL_SUBORDINATE_SFG'},
				{value:'SPECTRUM_REUSE',label:'SPECTRUM_REUSE'}
			],
			groupType:'INTERFERENCE_COORDINATION',
			groupId:'',//form  表单中的groupId and groupType  提交表单提交表单组
			listType:'eNB',
			options:[{value:'',label:''},{value:'A',label:'A'},{value:'B',label:'B'}],
			isShowEdit:[
				false, //Preferred Power
				false, // User ID
				false, // FccID
				false // SN
			],//是否放开修改按钮　只有验证通过才可以
			ruleForm:{
				groupId:'',
				groupIdAndGroupType:[],
				userId:'',
				fccId:'',
				serialNumber:'',
				callSign:'',
				manualMaxEirp:'',
				manualLowFreq:'',
				manualHighFreq:'',
				cbsdCategory:'A',
				supportSpec:'',
				indoor:'false',
				
				antennaGain:'',
				cpiId:'',
				cpiName:'',
				cpiInstallCertificationTime:'',
				plaintextMode:'false',
				saveInstallParamOnlyOnDP:'false',
				executeType:'active',
				scheduleTime:'',
				cellSpecificParams:[
					{
						cellSequenceId:'1',
						antennaHeightType:'AGL',
						antennaHeight:'',
						longitude:'',
						latitude:'',
						antennaAzimuth:'',
						antennaDowntilt:'',
						antennaBeamwidth:'',
						antennaGain:'',
						heightUnit:'meter',
						cpiCertCode:'',
						cpiInstallCertificationTime:''
					}
				]
			},
			rules:{
				manualMaxEirp:[
					{required:true,message:'Please input Integer'},
					{validator:validatemanualMaxEirp}
				],
				manualHighFreq:[ 
					{validator:validatemanualHighFreq,trigger:'blur'}
				],
				groupId:[
					{validator:validategroupId,trigger:'change'}
				],
				fccId:[
					{validator:fccIdValidator}
				],
				userId:[
					{validator:validateUserId}
				],
				antennaGain:[
					{validator:basicGainValidator1,trigger:'change'}
				],
			},
			uploadFileURL: '${ctx}/cell/SAS/generateCpiCode.action',
			fileName:'',
			fileParams:{},  
			ruleFormParameter:{ // 修改页面的表单
				indoor:'',
				cpiId:'',
				cpiName:'',
				password:'',
				cellSpecificParams:[]
			},
			rulesParameter:{ // 修改页面的表单验证
				longitude:[
					{validator:longitudeValidator}
				],
				latitude:[
					{validator:latitudeValidator}
				],
				antennaHeight:[
					{validator:antennaHeightValidator}
				],
				antennaAzimuth:[
					{validator:antennaAzimuthValidator}
				],
				antennaDowntilt:[
					{validator:antennaDowntiltValidator}
				],
				antennaGain:[
					{validator:antennaGainValidator}
				],
				antennaBeamwidth:[
					{validator:antennaBeamwidthValidator}
				],
				antennaHeightType:[
					{validator:antennaHeightTypeValidator}
				],
			},
			loading:false,
			sn:'',
			carrierType:'',
			iscbsd:false,
			notVirtualCBSD:true,
			disableForm:false,
			hasCode:false,
			oldFccId:'',
			heightUnitList:[{value:'meter',label:'m'},{value:'foot',label:'ft'}],
			sasUserIdDisIcon:false,
			sasUserIdDisVisible:false,
			OMCSasUserId:'',
			noUserIdErrShow:false,
			showSasSettingDialog:false,
			selectSasProvider:[
				{value:'0',label:'Federated Wireless'},
				{value:'2',label:'Amdocs'},
				{value:'3',label:'CommScope'},
				{value:'4',label:'Google'},
			],
			selectSasAble:[
				{value:'1',label:'Enable'},
				{value:'0',label:'Disable'}
			],
			sasSettingForm:{ // 设置SAS开关
				sasAble:'',
				sasProvider:'',
				sasUserId:''
			},
			fccIdIsNoMod:true,
			oldRegistrationMethod:'',
			showSasImportCert:false,
			sasImportCertForm:{
				errorFlag:false,
				uploadFileURL:'',
				errorMessage:'Only .P12/.PEM are supported',
				fileName:'',
				certFile:'',
				password:'',
				cpiid:'',
				cpiname:'',
				p12:'true',
				keyData:''
			},
			sasImportCertFormRules:{
				cpiid: [
					{required:true,message:'<%=rb.getString("BuNengWeiKong")%>',trigger:'blur'}
				],
				cpiname: [
					{required:true,message:'<%=rb.getString("BuNengWeiKong")%>',trigger:'blur'}
				],
				password:[
					{ validator:passwordValidator}
				]
			},
			executePickerOptions:{
				disabledDate(time){
					return time.getTime() < Date.getNow()-8.64e7;
				}
			},
		}
	},
	computed: {
		singleOrMultiTip_Category_A(){
			var vm = this,
				result = vm.ruleForm.cellSpecificParams.some(item=>item.cpiCertCode == "null" || item.cpiCertCode == "" || item.cpiCertCode == null);
			if(result){
				return vm.ruleForm.plaintextMode == 'true' ? "Single-Step" :"Multi-Step";
			}else{
				return "Single-Step"
			}
			
		},
		singleOrMultiTip_Category_B(){
			var vm = this,
				result = vm.ruleForm.cellSpecificParams.some(item=>item.cpiCertCode == "null" || item.cpiCertCode == "" || item.cpiCertCode == null);
			
			return result ? "Multi-Step" :"Single-Step";
		},
		registrationMethodDis(){
			var vm = this;
			
			return vm.ruleForm.longitude=="" || vm.ruleForm.latitude == "" || vm.ruleForm.antennaHeight == "" || vm.ruleForm.antennaHeightType == "" || vm.ruleForm.antennaAzimuth == "" || vm.ruleForm.antennaDowntilt == "" || vm.ruleForm.antennaGain == "" || vm.ruleForm.antennaBeamwidth == "" ? true :false;
		},
		registrationMethodShow(){
			var vm = this,
				result = vm.ruleForm.cellSpecificParams.some(item=>item.cpiCertCode == "null" || item.cpiCertCode == "" || item.cpiCertCode == null);
			
			return result
		},
	},
	watch: {
		'ruleForm.scheduleTime':function(newValue,oldValue){
			if(newValue == null){
				this.ruleForm.scheduleTime = '';
			}
		},
	},
	methods:{
		//进程操作事件
		setProcedureState(state,numPopover){
			var vm = this;
			var params = {};
			vm.numPopover = numPopover;
			params.dualCarrierType = vm.carrierType;
			params.serialNumber = vm.sn;
			params.deviceType = vm.listType.toLowerCase();
			params.action = state;

			axios.post("${ctx}/cell/SAS/operate.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
				
				}else{
					vm.$message({
						message:data.message,
						type:'error',
					})
				}
				
			})
			vm.setProcedureStatePopover[numPopover] = false;
			vm.setProcedureStatePopover[numPopover+'_icon'] = 'el-icon el-icon-reset active_btn';
			vm.stepsActive = false;
		},
		// 获取进程进度
		getProcedureState(){
			var procedureCtn = $("#sasInstallParam");
							
			if(!procedureCtn.length || !isVisible(procedureCtn[0])) {
				clearInterval(SASProcressInterval);
			}
			var vm = this,
				getStateUrl='',
				newdeviceType = vm.listType.toLowerCase(),
				indexMap = {
					0: 1,
					1: 2,
					2: 3,
					3: 3,
					4: 4,
					5: 5
				};
			
			if(!vm.iscbsd){
				getStateUrl = "${ctx}/cell/SAS/getState.action?serialNumber=" + vm.sn + '&dualCarrierType=' + vm.carrierType +'&deviceType=' + newdeviceType
			}else{
				getStateUrl = "${ctx}/cell/SAS/getState.action?serialNumber=" + vm.sn + '&dualCarrierType=' + vm.carrierType +'&deviceType=' + newdeviceType +'&virtualCbsd=1'
			}
			axios.post(getStateUrl).then(function(response){
				var data = response.data;
				vm.stepsAction = parseInt(data.state);
				if(vm.stepsAction || vm.stepsAction === 0 || vm.stepsAction === '0'){
					//vm.setProcedureStatePopover[vm.numPopover+'_icon'] = 'el-icon el-icon-operation-defaultBeta';
					Object.assign(vm.setProcedureStatePopover, {
						statePopover_1_icon:'el-icon el-icon-sas-success',
						statePopover_2_icon:'el-icon el-icon-sas-success',
						statePopover_3_icon:'el-icon el-icon-sas-success',
						statePopover_4_icon:'el-icon el-icon-sas-warning',
						statePopover_5_icon:'el-icon el-icon-sas-success'
					});
					
					vm.numPopover = 'statePopover_'+ indexMap[vm.stepsAction];
					if (data["isDoing"]) {
						vm.setProcedureStatePopover[vm.numPopover+'_icon'] = 'el-icon el-icon-reset active_btn';
						vm.stepsActive = false;
					} else {
						vm.setProcedureStatePopover[vm.numPopover+'_icon'] = 'el-icon el-icon-operation-defaultBeta';
						vm.stepsActive = true;
					}
				}
			})
		},
		// 初始化
		init(isAdd,type,row,sn,iscbsd){
			var vm = this;
			vm.isadd = isAdd;
			vm.listType = type;
			vm.rowData = row;
			vm.sn = sn;
			vm.iscbsd = iscbsd // 是否是虚拟CBSD false为真实CBSD
			if(isAdd === 'edit'){
				vm.isEdit = true;
				var obj ={
					serialNumber:sn,
					deviceType:type,
					timeZone:timeZone
				};
				var objQuery ={
						serialNumber:sn,
						smallCellCode:row.small_cell_code
					}
				
				if (vm.iscbsd) {
					obj.isVirtual = "1";
					objQuery.isVirtual = "1";
					vm.notVirtualCBSD = false;
					vm.disableForm = true;
				}
				vm.getProcedureState();
				if(row.connection_status == "Off"){
					vm.disableForm = true;
				}
				if (!writableMap.CODE_ADVANCE_SAS){
					vm.disableForm = true;
					vm.notVirtualCBSD = false;
				}
				vm.getInstallParams(obj,objQuery);
				
				clearInterval(SASProcressInterval);
				SASProcressInterval = setInterval(vm.getProcedureState, 6000);
			}else{
				
			}
		},
		getInstallParams(obj,objQuery) {
			var vm = this;

			axios.post("${ctx}/cell/SAS/getInstallParams.action",stringify(obj)).then(function(response){
				var data = response.data;
				if(data.success){
					var installParamsData = data.installParam;
					['cpiId','cpiName'].map((code)=>{
						if(installParamsData[code] == 'null' || installParamsData[code] == null || installParamsData[code] == undefined){
							installParamsData[code] = '';
						}
					})
					var cellSpecificParams = [],
						cellData = {
							cellSequenceId: '1'
						};
					['antennaHeightType','antennaHeight','antennaAzimuth','antennaBeamwidth','antennaDowntilt','antennaGain','latitude','longitude','cpiCertCode','cpiInstallCertificationTime','heightUnit'].map((item,index)=>{
						if(item == 'heightUnit'){
							cellData[item] = installParamsData[item] ? installParamsData[item] : 'meter';
						}else{
							cellData[item] = installParamsData[item] ? installParamsData[item] : '';
						}
						delete installParamsData[item]
					})
					cellSpecificParams.push(cellData);
					installParamsData.cellSpecificParams = cellSpecificParams;
					installParamsData.cellSpecificParams.map((item,indexs)=>{
						if(indexs === 0){
							installParamsData.cpiInstallCertificationTime = item.cpiInstallCertificationTime;
						}
						if(item.cellSequenceId == '1'){
							installParamsData.antennaGain = item.antennaGain
						}
					})
					vm.$nextTick(function(){
						Object.assign(vm.ruleForm,installParamsData);
						vm.oldFccId = installParamsData.fccId;
						if(vm.ruleForm.fccId == "" || vm.ruleForm.fccId == null || vm.ruleForm.fccId == undefined){
							vm.fccIdIsNoMod = false;
						}
						if(installParamsData.plaintextMode == 'true'){
							vm.ruleForm.plaintextMode = 'true'
						}else{
							vm.ruleForm.plaintextMode = 'false'
						}
						if(installParamsData.saveInstallParamOnlyOnDP == 'false'){
							vm.ruleForm.saveInstallParamOnlyOnDP = 'false'
						}else{
							vm.ruleForm.saveInstallParamOnlyOnDP = 'true'
						}
						if(installParamsData.cbsdCategory == 'A'){
							vm.oldRegistrationMethod = vm.singleOrMultiTip_Category_A ;
						}else if(installParamsData.cbsdCategory == 'B'){
							vm.oldRegistrationMethod = vm.singleOrMultiTip_Category_B ;
						}else {
							vm.oldRegistrationMethod = 'Multi-Step'
						}	
						
						if(vm.ruleForm.userId){
							if(vm.OMCSasUserId && vm.ruleForm.userId != vm.OMCSasUserId){
								vm.sasUserIdDisIcon = true;
							}else{
								vm.sasUserIdDisIcon = false;
							}
						}else{
							vm.ruleForm.userId = vm.OMCSasUserId;
						}
						
						if(vm.ruleForm.scheduleTime == '' || vm.ruleForm.scheduleTime == null || vm.ruleForm.scheduleTime == 'null' || vm.ruleForm.scheduleTime == undefined ){
							vm.ruleForm.scheduleTime = ''
						}
						vm.$nextTick(function(){
							vm.$refs["ruleForm"].clearValidate('groupId')
						})
						
					})
					// 针对模拟table选择的anntenna处理
					var cateGoryIndex= 0;
					if(installParamsData.cbsdCategory == 'A'){
						cateGoryIndex = 0
					}else {
						cateGoryIndex = 1;
						vm.cbsdChange('1',{"label":"B","value":"B"});
					}
					var result = installParamsData.cellSpecificParams.some(item=>item.cpiCertCode == "null" || item.cpiCertCode == "" || item.cpiCertCode == null);
					if(result){
						vm.hasCode = false;
					}else{
						vm.hasCode = true;
					}
				}
				vm.loading = false;
			})
		},
		
		initOMCUserId(isAdd,type,row,sn,iscbsd){
			var vm = this;
			
			axios.post('${ctx}/cell/SAS/getSasSettings.action').then(function(response){
				var data = response.data;

				var providers=[];
				var urls = data.sasURL;
				if(urls && urls.length){
					urls.map((item,index) => {
						providers.push({
							value: item.id,
							label: item.providerName,
							sasUserId:item.sasUserId,
						})
					})
				}
				providers.map((item)=>{
					if(item.value == data.sasProvider){
						vm.OMCSasUserId = item.sasUserId 
					}
				})
				vm.init(isAdd,type,row,sn,iscbsd);
			})
		},
		updateSettings(){// 提交设置函数
			var vm = this;

			vm.methodType = "saveFlag";
			vm.noUserIdErrShow = false;
			vm.$refs["ruleForm"].validate(valid => {
				if(valid ){
					isFormChanged(this.$refs.ruleForm)
					var params = {
							timeZone:timeZone
						};
					var sasCpiName =  sessionStorage.getItem('sasCpiName') ? sessionStorage.getItem('sasCpiName') : '';
					var	sasCpiId = sessionStorage.getItem('sasCpiId') ? sessionStorage.getItem('sasCpiId') : '';
					if(sasCpiName != '' || sasCpiId != ''){
						vm.ruleForm.cpiId = sasCpiId;
						vm.ruleForm.cpiName = sasCpiName;
					}
					params = Object.assign(params,vm.ruleForm);
					var cellData = params.cellSpecificParams[0];
					['antennaHeightType','antennaHeight','antennaAzimuth','antennaBeamwidth','antennaDowntilt','antennaGain','latitude','longitude','cpiCertCode','cpiInstallCertificationTime','heightUnit'].map((item,index)=>{
						params[item] = cellData[item] ? cellData[item] : '';
					})
					delete params.cellSpecificParams;
					delete params.groupId;
					let groupIdAndGroupTypeArr = [];
					groupIdAndGroupTypeArr = params.groupIdAndGroupType.map(function(item){
						let str="";
						str += (item.groupType+ ":" + item.groupId)
						return str;
					})
					
					if(vm.ruleForm.scheduleTime == ''){
						delete params.scheduleTime
					}
					delete params.executeType
					params.groupIdAndGroupTypeArr = groupIdAndGroupTypeArr;
					params.deviceType = vm.listType;
					delete params.groupIdAndGroupType;
					
					var regisMethodStrConfirm = "Saving SAS settings without CPI Signature Data will cause the CBSD registration method to change to multi-step mode.Do you wish to continue?"
					
					if((vm.ruleForm.cbsdCategory == 'A' && vm.oldRegistrationMethod == 'Single-Step' && vm.singleOrMultiTip_Category_A == 'Multi-Step') || (vm.ruleForm.cbsdCategory == 'B' && vm.oldRegistrationMethod == 'Single-Step' && vm.singleOrMultiTip_Category_B == 'Multi-Step')){
						vm.$confirm(regisMethodStrConfirm,'<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							dangerouslyUseHTMLString:true
						}).then(function(){
							vm.loading = true;
							axios.post('${ctx}/cell/SAS/saveCpeCbsdParams.action',stringify(params)).then(function(response){
					    		var data = response.data;
					    		vm.methodType = "";
					    		if(data["success"]){
					    			vm.$message({
				   						message:"Save Success",
				   						type:'success',
				   					})
									vm.loading = false;
					    			vm.closeInfoparams();
					    		}else{
					    			vm.$message({
				   						message:data.message,
				   						type:'error',
				   					})
				   					vm.loading = false;
					    			vm.closeInfoparams();
					    		}
					    	})
						}).catch(function(){});
					}else {
						vm.loading = true;
						axios.post('${ctx}/cell/SAS/saveCpeCbsdParams.action',stringify(params)).then(function(response){
				    		var data = response.data;
				    		vm.methodType = "";
				    		if(data["success"]){
				    			vm.$message({
			   						message:"Save Success",
			   						type:'success',
			   					})
								vm.loading = false;
				    			vm.closeInfoparams();
				    		}else{
				    			vm.$message({
			   						message:data.message,
			   						type:'error',
			   					})
			   					vm.loading = false;
				    			vm.closeInfoparams();
				    		}
				    	})
					}
					
					
				}
			}) 
		},
		clearCode(){
			var vm = this,confirmMsg='<%=rb.getString("QingKongCPICodeTiShi")%>';
			
			vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		vm.ruleForm.cellSpecificParams.map((item)=>{
					item.cpiCertCode = '';
				})
				vm.hasCode = false;
	    	}).catch()
			
			
		},
		parameterFrom(){ // 打开设置页面
			var vm = this,errCount=0 , errIndex = 0 , flag = true;
			
			vm.showWindowInfo=true;
			let newVal = vm.ruleForm;
			let oldVal = vm.ruleFormParameter
			Object.keys(newVal).forEach(key=>{oldVal[key]=newVal[key]}) //修改的信息如果key相同就赋值
			vm.ruleFormParameter.password = '';
		},
		closeSettingInfo(){   //关闭设置页面  
			var vm= this;
			vm.showWindowInfo = false

		},
		// 保存修改信息
		saveInstallParamsModify(){
			var vm = this,
				params={};

			params.deviceType = vm.listType;
			params.indoor = vm.ruleFormParameter.indoor;
			params.fccId = vm.ruleForm.fccId;
			params.serialNumber = vm.ruleForm.serialNumber;
			params.cbsdCategory = vm.ruleForm.cbsdCategory;
			
			var cellData = vm.ruleFormParameter.cellSpecificParams[0];
			['antennaHeightType','antennaHeight','antennaAzimuth','antennaBeamwidth','antennaDowntilt','antennaGain','latitude','longitude','cpiCertCode','cpiInstallCertificationTime','heightUnit'].map((item,index)=>{
				params[item] = cellData[item] ? cellData[item] : '';
			})
			params.cpiId = sessionStorage.getItem('sasCpiId');
			params.cpiName = sessionStorage.getItem('sasCpiName');
			params.keyData = sessionStorage.getItem('keyData');
			params.password = vm.sasImportCertForm.password;
			
			var regisMethodStr = "Saving installation settings without a CPI vertificate will cause the CBSD registration method to change to multi-step mode.Do you wish to continue?"
			vm.$refs["ruleFormParameter"].validate(valid => {
				if(valid){
					
					if(vm.ruleForm.cbsdCategory == 'B'){
						if(params.keyData == '' || params.keyData == null || params.keyData == undefined){
							vm.showSasImportCert = true;
							return
						}else{vm.modifyInstallParamsSubmit(params);
						}
					}else{
						if (vm.ruleFormParameter.indoor == "false" && (params.keyData == '' || params.keyData == null || params.keyData == undefined)){
							vm.showSasImportCert = true;
							return;
						}else{
							if (params.keyData == '' || params.keyData == null || params.keyData == undefined ){
								if(vm.oldRegistrationMethod == 'Single-Step' && vm.ruleForm.plaintextMode == 'false'){
									vm.$confirm(regisMethodStr,'<%=rb.getString("QueRen")%>',{
										customClass:'warningConfirm',
										confirmButtonText:'<%=rb.getString("QueDing")%>',
										cancelButtonText:'<%=rb.getString("QuXiao")%>',
										dangerouslyUseHTMLString:true
									}).then(function(){
										vm.showParams();
										vm.showWindowInfo = false;
									}).catch(function(){});
								}else {
									vm.showParams();
									vm.showWindowInfo = false;
								}
							}else {
								vm.modifyInstallParamsSubmit(params);
							}
						}
					}
				}else{
					return false
				}
			})
		},
		// 保存修改信息 提交
		modifyInstallParamsSubmit(params){
			var vm = this;
			axios.post('${ctx}/cell/SAS/generateCpiCode.action',stringify(params)).then(function(response){
				var data = response.data;
				
				if(data["success"]){
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
					vm.showParams();
					var cpiData = data.cpiMsg;
					// cpiData.map((item,index)=>{
					// 	vm.ruleForm.cellSpecificParams[index].cpiCertCode = item.cpiCertCode;
					// 	vm.ruleForm.cellSpecificParams[index].cpiInstallCertificationTime = item.cpiInstallCertificationTime;
					// })
					vm.ruleForm.cellSpecificParams[0].cpiCertCode = cpiData.cpiCertCode;
					vm.ruleForm.cellSpecificParams[0].cpiInstallCertificationTime = cpiData.cpiInstallCertificationTime;
					vm.closeFileSelect();
					vm.showWindowInfo = false;
				}else{
					vm.$message({
						message:data.message,
						type:'error',
					})
				}
				
			})
		},
		showParams(){
			var vm = this;
			
			let newVal = vm.ruleFormParameter;
			let oldVal = vm.ruleForm;
			Object.keys(newVal).forEach(key=>{oldVal[key]=newVal[key]}) //修改的信息如果key相同就赋值
			
			var result = vm.ruleForm.cellSpecificParams.some(item=>item.cpiCertCode == "null" || item.cpiCertCode == "" || item.cpiCertCode == null);
			if(result){
				vm.hasCode = false;
			}else{
				vm.hasCode = true;
			}
			vm.ruleForm.cellSpecificParams.map((item)=>{
				if(item.cellSequenceId == '1'){
					vm.ruleForm.antennaGain = item.antennaGain
				}
			})
		},
		// 取消
		closeInfoparams(){
			clearInterval(SASProcressInterval);
			eventBus.$emit('close-dialog');
		},
		addGroupTypeAndId(){
			//groupType:[{value:'INTERFERENCE_COORDINATION',label:'INTERFERENCE_COORDINATION'}],
			//groupId:'',//form  表单中的groupId and groupType  提交表单提交表单组
			var vm = this;
			vm.methodType = "";
			let obj = {};
			if(vm.OMCSasUserId){
				vm.noUserIdErrShow = false;
				vm.$refs["ruleForm"].validateField('groupId',(groupIdFlag) => {
					if(!groupIdFlag){ //验证通过
						if(vm.ruleForm.groupId != ""){
							obj.groupId = vm.ruleForm.groupId;
						}else{
							return 
						}
						obj.groupType = vm.groupType;
						vm.ruleForm.groupIdAndGroupType.unshift(obj);
						vm.ruleForm.groupId = ""
						vm.$nextTick(function(){
							vm.$refs["ruleForm"].clearValidate('groupId')
						})
						
					}else{//验证没有通过
						
					}
				})
			}else{
				vm.noUserIdErrShow = true;
			}
			
			
		},
		delGroupTypeAndId(index){ // Group 删除添加项
			var vm = this;
			vm.ruleForm.groupIdAndGroupType.splice(index,1);
		},
		closeFileSelect(){ // 关闭文件选择
			var vm = this;
			var result = vm.ruleForm.cellSpecificParams.some(item=>item.cpiCertCode == "null" || item.cpiCertCode == "" || item.cpiCertCode == null);
			if(result){
				vm.hasCode = false;
			}else{
				vm.hasCode = true;
			}
		},
		changeindoor(index,data){ // indoor 切换选择
			this.ruleFormParameter.indoor = data.value
		},
		changeHeightType(type,index,data){
			this.ruleFormParameter.cellSpecificParams[index].antennaHeightType = data.value;
		},
		cbsdChange(index,data){ // cbsd 切换数据更新
			var vm = this;
			if (this.iscbsd && this.isadd == 'edit') {
				return;
			}
			if(data.value == 'A' && vm.ruleForm.indoor == 'Outdoor'){
				vm.ruleForm.cellSpecificParams.map((item,index)=>{
					if(item.heightUnit == 'meter'){
						if(item.antennaHeight && parseFloat(item.antennaHeight) > 6){
							var message = 'CBSD Category A shall not be deployed or operated outdoors with antennas exceeding a height of 6 meters above average terrain'
							vm.$message({
								message:message,
								type:'error',
							})
							return;
						}
					}else{
						if(item.antennaHeight && parseFloat(item.antennaHeight) > 19.47){
							var message = 'CBSD Category A shall not be deployed or operated outdoors with antennas exceeding a height of 6 meters above average terrain'
							vm.$message({
								message:message,
								type:'error',
							})
							return;
						}
					}
				})
			}
			vm.ruleForm.cbsdCategory = data.value;
		},
		syncParams(){
			var vm = this;
			var obj ={
				deviceType:vm.listType,
				serialNumber:vm.sn
			}
			vm.loading = true;
			axios.post("${ctx}/cell/SAS/syncSasParams.action",stringify(obj)).then(function(response){
				vm.loading = false;
				var data = response.data;
   				
				if(data["success"]){
					vm.$nextTick(function(){
	   	   				Object.assign(vm.ruleForm,response.data.installParam)
	   				})
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
	    		}else{
	    			vm.$message({
   						message:data.message,
   						type:'error',
   					})
	    		}
   			}) 
		},
		// Fcc Id 改变 增加提示信息
		fccIdChange(val){
			var vm = this;

			if(vm.oldFccId != val){
				vm.$message({
					type: 'warning',
					message: '<%=rb.getString("FccIdGaiBianTiShi")%>'
				});
			}
		},
		// 米 英尺单位换算
		unitConversion(val,type){
			var value = '';
			if(!val || isNaN(parseInt(val)))return value
			if(type == 'meter'){ // 类型 米  由英尺转为米
				return (Number(val)*0.3048).toFixed(2)
			}else{ // 类型 英尺  由米转为英尺
				return (Number(val)*3.2808).toFixed(2)
			}
		},
		// 替换确定
		sasUserIdReplace(){
			var vm =this;
			vm.ruleForm.userId = vm.OMCSasUserId;
			vm.sasUserIdDisVisible = false;
			vm.sasUserIdDisIcon = false;
		},
		// 设备 GPS 信息：经度 纬度 高度 同步
		syncdeviceGps(index){
			var vm =this,
				urls='${ctx}/cell/SAS/getDeviceGPSInfo.action',
				index = index, // '0' 第一套参数 点击同步 '1' 第二套参数 点击同步
				params={
					serialNumber:vm.sn,
					deviceType:vm.listType,
				};
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				var str = '<div><%=rb.getString("QueRenZiDongTianChongSheBeiShangBaoZhi")%></div>'+"<div><span>Longitude:"+ data.longitude +"</span><span style='margin-left:20px;'>Latitude:"+ data.latitude +"</span><span style='margin-left:20px;'>Hight:"+data.height+"</span></div>"
   				vm.$confirm(str,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					dangerouslyUseHTMLString:true
				}).then(function(){
					vm.ruleFormParameter.cellSpecificParams[index].latitude = data.latitude;
					vm.ruleFormParameter.cellSpecificParams[index].longitude = data.longitude;
					vm.ruleFormParameter.cellSpecificParams[index].antennaHeight = data.height;
				}).catch(function(){
					
				})
				
   			}) 
		},
		// 打开SAS 设置弹窗
		goSetting(){
			var vm = this;
			axios.post('${ctx}/cell/SAS/getSasSettings.action').then(function(response){
				var data = response.data;
				vm.sasSettingForm.sasAble = data.sasEnable;
				vm.sasSettingForm.sasProvider = data.sasProvider;
				var providers=[];
				var urls = data.sasURL;
				if(urls && urls.length){
					var sasUrlList = {};
					urls.map((item,index) => {
						sasUrlList[item.id] = item.url;
						providers.push({
							value: item.id,
							label: item.providerName,
							sasUserId:item.sasUserId,
						})
					})
				}
				providers.map((item)=>{
					if(item.value == vm.sasSettingForm.sasProvider){
						vm.sasSettingForm.sasUserId = item.sasUserId 
					}
				})
				vm.selectSasProvider = providers;
				vm.showSasSettingDialog = true;
			})
		},
		submitSASSetting(){//sas设置提交事件
			var vm = this, url='${ctx}/cell/SAS/saveSasSettings.action',params={};
			params.sasEnable = vm.sasSettingForm.sasAble;
			params.sasProvider = vm.sasSettingForm.sasProvider;
			params.sasUserId = vm.sasSettingForm.sasUserId;
			
			vm.$refs.sasSettingForm.validate((r)=>{
				if(r){
					axios.post(url,stringify(params)).then(function(response){
						var data = response.data;
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
							vm.cancelViewSlide();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
						// 更新全局变量
						SASEnble = params.sasEnable;
					})
				}
			})
		},
		// 提供商改变
		sasProviderChange(val){
			var vm = this;
			vm.selectSasProvider.map((item)=>{
				if(item.value == val){
					vm.settingsEnbForm.sasUserId = item.sasUserId 
				}
			})
		},
		closeSasSettingDialog(){
			var vm = this;
			vm.showSasSettingDialog = false;
		},
		// 导入证书文件关闭
		importSasCertClose(){
			var vm = this,
				params={
					errorFlag:false,
					uploadFileURL:'',
					fileName:'',
					password:'',
					cpiid:'',
					cpiname:'',
					p12:'true',
					keyData:''
				};
			vm.showSasImportCert = false;
			Object.assign(vm.sasImportCertForm,params);
		},
		 /* PrivateKey 导入监听
		* file:文件
		*fileList：文件列表
		*/
		fileChangePrivateKey(file, fileList) {
			var vm = this;
			var errorFlag = file.name.substr(file.name.lastIndexOf(".")) === '.pem' || file.name.substr(file.name.lastIndexOf(".")) === '.p12'
			vm.sasImportCertForm.errorFlag = !errorFlag;
			if (errorFlag) {
				vm.sasImportCertForm.certFile = file.raw;
				vm.sasImportCertForm.fileName = file.name;
			} else {
				vm.sasImportCertForm.fileName = '';
			}
		},
		checkFile(res, file) {    //发送请求，校验device文件内容 
			var vm = this;
			if (res.success) {
				if (res.suc_count > 0) {
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
				} else {
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("ShiBai")%>'
					});
				}
			} else {
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.uploadCert.uploadFiles;
			fileList.forEach(function (file) {
				file.status = 'ready';
			})
		},
		//PrivateKey 导入文件按钮
		fileSelectPrivateKey() { 
			var vm = this;
			vm.$refs.uploadCert.clearFiles();
			vm.$refs['file_up'].click();
		},
		p12CheckChange(val){
			var vm =this;
			if(val == 'false'){
				vm.sasImportCertForm.password = ''
			}
		},
		// sas 证书导入确定
		importSasCertSubmit(){
			var vm = this;
			 if(vm.sasImportCertForm.fileName == ''){
				vm.sasImportCertForm.errorFlag = true;
				return
			}
			vm.$refs.sasImportCertForm.validate((valid) => {
				if (valid){
					var fileName = vm.sasImportCertForm.fileName;
					var fileType = fileName.substr(fileName.lastIndexOf(".")) === '.p12' ? 'P12' : 'PEM';
					if(fileType == 'P12'){
						if(vm.sasImportCertForm.p12 == 'false'){
							vm.$message({
								message: 'CPI Certificate password is incorrect',
								type:'error',
							});
							return
						}
						vm.parsingP12Cert(vm.sasImportCertForm.certFile);
					}else{
						vm.parsingPemCert(vm.sasImportCertForm.certFile);
					}
					setTimeout(()=>{
						if(vm.sasImportCertForm.keyData == ''){
							vm.$message({
								message: 'CPI Signature Data was not generated successfully',
								type:'error',
							});
							return
						}
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						})
						vm.sasCertStatus = true;
						sessionStorage.setItem('keyData',vm.sasImportCertForm.keyData);
						sessionStorage.setItem('sasCpiId',vm.sasImportCertForm.cpiid);
						sessionStorage.setItem('sasCpiName',vm.sasImportCertForm.cpiname);
						sessionStorage.setItem('userCode',user_code);
						vm.importSasCertClose();
					},200)
				} else {
					return false;
				}
			})
		},
		// 解析P12证书
		parsingP12Cert(file){
			var vm = this;
			var reader = new FileReader();
			reader.onload = function(e){
				var content = e.target.result;
				var pkcs12der = vm.arrBufferToString(content);
				var pkcs12B64 = forge.util.encode64(pkcs12der);

				var pkcs12ders = forge.util.decode64(pkcs12B64);
				try {
					var p12Asn1 = forge.asn1.fromDer(pkcs12ders);
				} catch (error) {
					vm.$message({
						message: 'Invalid CPI Certificate file',
						type:'error',
					});
					return
				}
				
				try {
					var p12 = forge.pkcs12.pkcs12FromAsn1(p12Asn1,vm.sasImportCertForm.password);
				} catch (error) {
					vm.$message({
						message: 'CPI Certificate password is incorrect',
						type:'error',
					});
					return
				}

				var privateKey;
				for(var sci = 0; sci < p12.safeContents.length; sci++){
					var safeContents = p12.safeContents[sci];
					for(var sbi = 0; sbi < safeContents.safeBags.length; sbi++){
						var safeBag = safeContents.safeBags[sbi];

						if(safeBag.type === forge.pki.oids.pkcs8ShroudedKeyBag){
							privateKey = safeBag.key
						}
					}
				};
				
				var pkcs8 = vm.privateKeyToPkcs8(privateKey);
				vm.sasImportCertForm.keyData = pkcs8;
			}
			reader.readAsArrayBuffer(file);
		},
		// 解析Pem证书
		parsingPemCert(file){
			var vm = this;
			var reader = new FileReader();
			reader.onload = function(e){
				var content = e.target.result;
				var pemder = vm.arrBufferToString(content);
				try {
					var privateKey = forge.pki.privateKeyFromPem(pemder);
				} catch (error) {
					vm.$message({
						message: 'Invalid CPI Certificate file',
						type:'error',
					});
					return
				}
				var rsaPrivateKey = forge.pki.privateKeyToAsn1(privateKey);
				var privateKeyInfo = forge.pki.wrapRsaPrivateKey(rsaPrivateKey);
				var privateKeyInfoDer = forge.asn1.toDer(privateKeyInfo).getBytes();
				var p8b64 = forge.util.encode64(privateKeyInfoDer)

				vm.sasImportCertForm.keyData = p8b64;
			}
			reader.readAsArrayBuffer(file);
		},
		// 转换为PKCS8
		privateKeyToPkcs8(privateKey){
			var vm = this;
			var rsaPrivateKey = forge.pki.privateKeyToAsn1(privateKey);
			var privateKeyInfo = forge.pki.wrapRsaPrivateKey(rsaPrivateKey);
			var privateKeyInfoDer = forge.asn1.toDer(privateKeyInfo).getBytes();
			var p8b64 = forge.util.encode64(privateKeyInfoDer)
			return p8b64
		},
		// 数组存储转换成字符串
		arrBufferToString(buffer){
			var binary = '';
			var bytes = new Uint8Array(buffer);
			var len = bytes.byteLength;
			for(var i = 0; i<len; i++){
				binary += String.fromCharCode(bytes[i]);
			}
			return binary
		},
		// 字符串转换为数据存储
		stingToArrayBuffer(data){
			var arrBuff = new ArrayBuffer(data.length);
			var writer = new Uint8Array(arrBuff);
			for(var i = 0,len = data.length; i<len;i++ ){
				writer[i] = data.charCodeAt(i);
			}
			return arrBuff
		},
		//定时执行 时间选择
		setTime(){
			var vm = this;
			
			vm.$nextTick(()=>{
				document.getElementsByClassName('el-button--text')[0].setAttribute('style','display:none');
				if(vm.ruleForm.scheduleTime == '' || vm.ruleForm.scheduleTime == null || vm.ruleForm.scheduleTime == undefined){
					vm.ruleForm.scheduleTime = formatDate(addDate(gloableTime.slice(0,10),1));
				}
			})
		},
		antennaHeightFmt(item){
			var vm = this;
			if(item.heightUnit == 'meter'){
				return '≈' + vm.unitConversion(item.antennaHeight,'foot') + 'ft'
			}else{
				return '≈' + vm.unitConversion(item.antennaHeight,'meter')+ 'm'
			}
		}
	},
	mounted(){
		eventBus.$off('open-dialog').$on('open-dialog', this.initOMCUserId);
	}
	
})

</script> 