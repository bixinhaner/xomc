<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style type="text/css">
	#settingTables .plr15 {
		padding: 0 25px !important
	}
	
	#settingTables .plr15 .el-form-item__label {
		width: 250px;
		float: left;
		font-size: 12px;
		line-height: 25px
	}
	
	#settingTables .el-form-item {
		margin-bottom: 13px
	}
	
	#settingTables .el-form-item .el-form-item {
		margin-bottom: 6px
	}
	
	#settingTables .modelBox {
		border-bottom: #e9e9e9 solid 1px;
		padding: 10px 0px 10px 35px;
		margin:20px;
	}
	
	#settingTables .modelBox .el-form-item_content {
		font-size: 14px !important
	}
	
	#settingTables .group-title {
		margin-bottom: 20px;
	}
	
	/*.w600 {
		width: 600px
	}*/
	
	#settingTables .timeZone .el-input {
		width: 520px
	}
	
	#settingTables .language .el-input {
		width: 70px
	}
	
	#settingTables .loglanguage .el-input {
		width: 60px
	}
	/*.informSelect .el-input { width: 120px; }*/
	#settingTables .kpiDataStorage .el-input{
		width: 60px
	}
	#settingTables .Yylanguage .el-input {
		width: 90px
	}
	
	#settingTables .w100 {
		width: 100px
	}
	
	/*.w250 {
		width: 250px
	}*/
	
	#settingTables .w50 {
		width: 50px
	}
	
	#settingTables .ml5 {
		margin-left: 5px
	}
	
	#settingTables .mr5 {
		margin-right: 5px
	}
	
	#settingTables .box {
		height: 100%;
		overflow: auto;
		flex:1;
	}
	
	#settingTables .textInput {
		margin-left: 250px
	}
	
	#settingTables a {
		text-decoration: underline;
		color: #4D84FF
	}

	#settingTables .week,
	#settingTables .normal,
	#settingTables .strong {
		display: flex
	}

	#settingTables .week .el-icon-signal:before {
		color: #E88282;
		font-size: 18px;
		top: 3px;
		position: relative;
	}

	#settingTables .normal .el-icon-signal:before {
		color: #F2B354;
		font-size: 18px;
		top: 3px;
		position: relative;
	}
	
	#settingTables .strong .el-icon-signal:before {
		color: #67D972;
		font-size: 18px;
		top: 3px;
		position: relative;
	}
	
	#settingTables .notifyBox .el-form-item__label {
		width: 100px
	}
	
	#settingTables .doubleBox {
		margin-left: 250px;
		margin-top: 7px;
		float: left;
		width: 100%;
	}
	
	/*.mt20 {
		margin-top: 20px
	}*/
	
	#settingTables .mt10 {
		margin-top: 10px
	}
	
	/*.mt5 {
		margin-top: 5px
	}*/
	
	#settingTables .w150 {
		width: 150px
	}
	
	#settingTables .w120 {
		width: 120px
	}
	
	#settingTables .w300 {
		width: 300px
	}
	
	#settingTables .deviceLogContainer {
		height: 300px;
		margin-left: 25px
	}
	
	.el-icon .el-iocn-circle-add {
		font-size: 14px
	}
	
	.footer {
		width: 99.9%;
		overflow: overlay;
		border-top: #E9E9E9 solid 1px;
		height: 48px;
		line-height: 48px;
		background: #FFFFFF
	}
	
	.footer .lnkbuttonGroup {
		margin-left: 48px
	}
	
	#settingTables .ml250 {
		margin-left: 250px
	}
	
	#settingTables .tongzhiBox .el-form-item__error {
		margin-left: 100px
	}
	
	#settingTables .tongzhiLabel .el-form-item__label {
		width: 100px
	}
	
	#settingTables .tongzhiLabel .el-form-item__error {
		margin-left: 45px
	}
	
	#settingTables .jbshezhiBox .el-form-item__error {
		top: 3px;
		left: 460px
	}
	
	/*.logBox .el-form-item__error {
		margin-left: 100px
	}*/
	
	#settingTables .morenmimaBox .el-form-item__error {
		top: 3px;
		left: 320px
	}
	
	#settingTables .error {
		color: #FA5555;
		font-size: 12px
	}
	
	#settingTables .errorBorder {
		border-color: #FA5555 !important;
	}
	
	#settingTables .errorBorder:focus {
		border-color: #FA5555 !important;
	}
	
	#settingTables .tongzhiLi {
		height: 0px
	}
	
	#settingTables .tongzhiLi li {
		line-height: 22px
	}
	
	#settingTables .mimaqiangduBOX .el-form-item__content,
	#settingTables .mimaYouXiaoQi .el-form-item__content {
		line-height: 22px
	}
	
	#settingTables .mb13 {
		margin-bottom: 13px
	}
	#settingTables .mb20 {
		margin-bottom: 20px
	}
	#settingTables .mb10 {
		margin-bottom: 10px
	}
	
	#settingTables .mb1 {
		margin-bottom: 1px
	}
	
	#settingTables .mb7 {
		margin-bottom: 7px
	}
	
	#settingTables .el-select-dropdown__item {
		font-size: 12px
	}
	
	#settingTables .el-icon-circle-add {
		font-size: 20px
	}
	
	#settingTables .el-card__body {
		width: 99.8%
	}
	
	@media screen and (min-width:1400px) {
		.screenWidth {
			width: 20.83333333%
		}
		.screenWidth4 {
			width: 16.6666666%
		}
		.screenWidth7 {
			width: 25%
		}
		.screenWidth1 {
			width: 20%
		}
	}
	
	@media screen and (max-width:1440px) {
		.screenWidth {
			width: 27.83333333%
		}
		.screenWidth .el-form-item__label,
		.screenWidth4 .el-form-item__label,
		.screenWidth7 .el-form-item__label {
			width: 75px
		}
		.screenWidth4 {
			width: 19%
		}
		.screenWidth7 {
			width: 35%
		}
		.screenWidth1 {
			width: 10%
		}
	}
	#settingTables .el-icon-circle-success:before{
		color:#67D972;
		font-size: 16px
	}
	#settingTables .yanzhengsuceess{
		color:#67D972;
		margin-left: 5px
	}
	#settingTables .el-icon-circle-close::before{
		color:#E88282;
		font-size: 16px;
		content:'\e6fb';
	}
	#settingTables .yanzhengerror{
		color:#E88282;
		margin-left: 5px
	}
	#settingTables .ceshiEmailBtn{
		min-width: 0px;
		padding: 7px 15px;
		line-height: 6px;
		font-size: 12px
	}
	
	.readonly .modelBox {
		position: relative;
	}
	.readonly .modelBox::after {
		display: block;
		content: '';
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		left: 0;
		z-index: 100;
	}
	
	.setEmailBox .el-dialog__body{
		height:75px;
	}
	#settingTables .northSet {
		padding-bottom:20px;
	}
	#settingTables .northSet .el-form-item {
		width:35%;
		display:inline-block;
	}
	#settingTables .northSet .el-form-item__label {
		width:150px;
	}
	#settingTables .northSet .el-form-item__error {
		left:150px;
	}
	#settingTables .northUserEnableClickBox{
		height: 20px;
		width: 100%;
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
	#northUserDialog .el-dialog__body .el-form-item__error{
		padding-top:0;
		top: unset
	}
	#northUserDialog .el-form-item{
		margin-bottom: 15px;
	}
	#settingTables .ldapBoxCls{
		width: 100%;
		margin-bottom: 60px;
	}
	#settingTables .ldapBoxCls .el-form-item .el-form-item__error {
		margin-left: 250px;
		padding-top: 0;
	}
	#settingTables .ldapBoxCls .loapSwitchBox .el-form-item__content {
		display: flex;
	}
	#settingTables .ldapBoxCls .el-form-item{
		width: 40%;
		display: inline-block;
	}
	#settingTables .newPasswordCls input[type="password"]::-ms-reveal{
		display: none;
	}
	#settingTables .newPasswordCls .el-input__suffix{
		top:5px;
	}
	#settingTables .errorIconCls::before{
		color: #E88282;
		font-size: 24px;
	}
	#settingTables .successIconCls::before{
		color: #67D972;
		font-size: 24px;
	}
	#settingTables .split-group-title {
		font-size: 14px;
		font-weight: bold;
		margin-left:20px;
		margin-top: 15px;
	}
	#settingTables .split-group {
		padding:20px 20px 20px 50px;
		border-bottom: 1px solid #E9E9E9;
	}
	#settingTables .second-group .split-group:last-child {
		border: none;
	}
	#settingTables .second-group-title {
		font-size: 14px;
		margin: 15px 0;
		font-weight: bold;
		
	}
	#settingTables .second-group-title li {
		list-style-type: disc;
	}
	#settingTables .second-group {
		border: 1px solid #e9e9e9;
	}
	#settingTables .settingsRecycleBinCls .el-form-item__error{
		padding-top: 0px!important;
	}
    #settingTables .settingsRecycleBinCls .el-form-item__label{
		padding-top: 10px!important;
	}
	#settingTables .ldapSSLBoxCls{
		display: inline-block;
		width: 120px!important;
	}
	#settingTables .ldapSSLBoxCls .el-checkbox__label{
		font-size: 12px;
	}
</style>
<div class="overflow-cls">
	<div class='panelDefault commonWarp' id='settingTables' style="min-width: 1100px;width: 100%;">
		<el-tabs v-model="activeName" style='height:100%;' @tab-click='tabClick'>
			<el-tab-pane label="<%=rb.getString("SheZhi")%>" name="sysSettings">
				<div class="panelDefault"  :class="{readonly:!isSetable}" style="border:none;position: relative;height: 100%;display:flex;flex-direction:column;">
					<div class="box">
						<el-form label-position="top" ref="settingsForm" :model='settingsForm' :rules='rules'>
							<!--基本设置-->
							<div class='modelBox' v-show="basicLimit">
								<div class="group-title not-extend">
									<span class="title-icon"></span><span class="title-text"><%=rb.getString("JiBenSheZhi")%></span>
								</div>
								<div class="plr15 jbshezhiBox">
									<el-form-item label='<%=rb.getString("YunYingShangMingCheng")%>' prop="mrVendor" v-if="MrFlag">
										<el-input v-model="settingsForm.mrVendor" placeholder='' class="w290" maxlength='50'></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("OMCMingCheng")%>' prop="mrOMCName" v-if="MrFlag">
										<el-input v-model="settingsForm.mrOMCName" placeholder='' class="w350" maxlength='200'></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ShiQuSheZhi")%>' prop="timezoneCode" v-if='!conpetence.types'>
										<el-select v-model="settingsForm.timezoneCode" placeholder="" class="timeZone" @change="timezoneCodeChange">
											<el-option v-for="item in queryTimeData" :key="item.id" :label="item.text" :value="item.id"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='timeZoneId' style='margin-bottom: 0;'>
										<el-input v-model='settingsForm.timeZoneId' v-show="false"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("YuYanSheZhi")%>' prop="languageCode" v-if='!conpetence.types'>
										<el-select v-model="settingsForm.languageCode" placeholder="" class="Yylanguage">
											<el-option label='<%=rb.getString("ZhongWen")%>' value="zh"></el-option>
											<el-option label='<%=rb.getString("YingWen")%>' value="en"></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
							<!--安全设置-->
							<div class='modelBox' v-if='isAdmin && !conpetence.types'>
								<div class="group-title not-extend">
									<span class="title-icon"></span><span class="title-text"><%=rb.getString("AnQuanShiZhi")%></span>
								</div>
								<div class="plr15 anquan">
									<el-form-item label='<%=rb.getString("MoRenMiMa")%>' class="mb7">
										<el-checkbox v-model='settingsForm.modifyPWD' @change="checkChange('1')"></el-checkbox>
										<span class="ml5"><%=rb.getString("ShouCiDengLuXiuGaiMima")%></span>
										<div class="ml250 morenmimaBox">
											<el-form-item label='' prop="defaultPasswd" class="morenmima">
												<%=rb.getString("Jiang")%>
													<el-input v-model="settingsForm.defaultPasswd" maxlength='50' class="w100 ml5 mr5" :disabled=!settingsForm.modifyPWD @input="inputChange('1')"
														@blur="inputChange('1')"></el-input>
													<%=rb.getString("MiMaChongZhiHouDeMoRenMiMa")%>
														<span class="ml5 error" v-if='showError.morenmima'>{{errorStr.morenmima}}</span>
											</el-form-item>
										</div>
									</el-form-item>
									<el-form-item prop="modifyPWD" style="display:none;" label="" label-width="0px">
										<el-input v-model='settingsForm.modifyPWD'></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("MiMaQiangDu")%>' prop="passwordContent" class="mimaqiangduBOX mb7">
										<el-checkbox v-model='settingsForm.passwordContent' name='type' @change="checkChange('2')"></el-checkbox>
										<span class="ml5">	<%=rb.getString("MiMaBiXuLiangZhongLeiXing")%></span>
										<div class="ml250" style="display: flex;">
											<%=rb.getString("YongHuMiMaChangDu")%>： 
											<span>
												<div style="display: flex;">
													<%=rb.getString("ZuiXiaoZhi")%> 
													<el-form-item style="width: 180px;margin-left: 5px;" prop="pwdMinLength" label-position="left">
														<el-input v-model="settingsForm.pwdMinLength" style="width: 50px;" size="mini"></el-input>
													</el-form-item>
												</div>
												<div style="display: flex;margin-top: 15px;">
													<%=rb.getString("ZuiDaZhi")%> 
													<el-form-item style="width: 300px;margin-left: 5px;" prop="pwdMaxLength">
														<el-input v-model="settingsForm.pwdMaxLength" style="width: 50px;" size="mini"></el-input>
													</el-form-item>
												</div>
											</span>
										</div>
									</el-form-item>

									<el-form-item label='<%=rb.getString("YongHu")%>' class="mb10" prop="checkUserCodeEnable">
										<span>
											<%=rb.getString("YongHuMingCheng")%>：<el-checkbox v-if='false' v-model="settingsForm.checkUserCodeEnable"></el-checkbox> <%=rb.getString("YongHuMingBiHanZiFuTiShi")%>
										</span>
									</el-form-item>

									<el-form-item label='<%=rb.getString("MiMaYouXiaoQi")%>' class="mb10">
										<div class="fl">
											<el-checkbox v-model='settingsForm.expires' name='type' @change="checkChange('3')"></el-checkbox>
											<span class="ml5"><%=rb.getString("YongHuXiuGaiMiMaPinLv")%></span>
										</div>
										<div style="display:flex">
											<el-form-item label='' prop="validPeriod" class="validPeriod">
												<el-input v-model="settingsForm.validPeriod" placeholder='' class="w50 ml5 mr5" :disabled=!settingsForm.expires @input="inputChange('2')"
													@blur="inputChange('2')"></el-input>
											</el-form-item>
											<el-form-item label='' prop="promptBeforeDays" class="promptBeforeDays mimaYouXiaoQi">
												<%=rb.getString("XiuGaiYiCiMiMa")%>,
													<%=rb.getString("XiTongHuiZaiDaoQiQian")%>
														<el-input v-model="settingsForm.promptBeforeDays" class="w50 ml5 mr5" :disabled=!settingsForm.expires @input="inputChange('3')"
															@blur="inputChange('3')" onkeyup=""></el-input>
														<%=rb.getString("TianTiShi")%>
											</el-form-item>
											<span class="ml5 error" v-if='showError.validPeriod'>{{errorStr.validPeriod}}</span>
											<span class="ml5 error" v-if='showError.promptBeforeDays'>{{errorStr.promptBeforeDays}}</span>
										</div>
									</el-form-item>
									<el-form-item prop="expires" style="display:none;" label="" label-width="0px">
										<el-input v-model='settingsForm.expires'></el-input>
									</el-form-item>
									<!-- 最大允许登录错误次数--private String attemptTimes;  最大允许登录错误次数时的锁死时间 - private String unlockMinu;-->
									<el-form-item label='<%=rb.getString("ZuiDaYunXuDengLuCuoWuCiShu")%>' class="mb10">
										<div style="display:flex">
											<el-form-item label='' prop="verifyEnable" class="mb7" style='margin: 6px 10px 0 0;'>
												<el-checkbox v-model='settingsForm.verifyEnable' name='type' @change="checkChange('12')"></el-checkbox>
											</el-form-item>

											<%=rb.getString("YanZhengMaYanZhengShiDeYongHuMingHuoMiMaTiShi")%>
											<!--User login requires CAPTCHA verification. if username or password entered incorrectly-->
											<el-form-item label='' prop="attemptTimes" class='shibaiInput'>
												<el-input v-model="settingsForm.attemptTimes" @input="inputChange('4')" @blur="inputChange('4')" class="w50 ml5 mr5"></el-input>
												<%=rb.getString("SuoDingShiJian")%>  <!--times, will be asked to enter the verification code-->
											</el-form-item>
										</div>

										<div class="ml250 morenmimaBox">
											<div style="display:flex">
												<%=rb.getString("DengLuShiBaiCiShu")%>
												<el-form-item label='' prop="sumTimes" class="validVerifyCode">
													<el-input v-model="settingsForm.sumTimes" placeholder='' class="w50 ml5 mr5"  @input="inputChange('23')"
														@blur="inputChange('23')"></el-input>
													<%=rb.getString("CiHouYongHuSuoDing")%>
												</el-form-item>
												<el-form-item label='' prop="unlockMinu" class='suodingInput'>
													<el-input v-model="settingsForm.unlockMinu" @input="inputChange('5')" @blur="inputChange('5')" class="w50 ml5 mr5"></el-input>
													<%=rb.getString("SettingFenZhong")%>
												</el-form-item>
											</div>
										</div>
										<div style='margin-left: 250px;'>
											<span class="ml5 error" v-if='showError.shibaicishu'>{{errorStr.shibaicishu}}</span>
											<span class="ml5 error" v-if='showError.suodingshijian'>{{errorStr.suodingshijian}}</span>
											<span class="ml5 error" v-if='showError.verifyCodeErrorNum'>{{errorStr.verifyCodeErrorNum}}</span>
										</div>
									</el-form-item>

									<!-- IP 限流 下发3个次数： x分钟内  连续错误 N 次， 加入黑名单， Y 分钟后该 IP 自动释放-->
									<el-form-item label='<%=rb.getString("IPXianLiu")%>' prop="" class="mb10" v-if="isLocalZH == true">
										<div style="display:flex">
											<%=rb.getString("DangZai")%>
											<el-form-item label='' prop="limitMinus" class="validLimitMins">
												<el-input v-model="settingsForm.limitMinus" placeholder='' class="w50 ml5 mr5" @input="inputChange('24')"
													@blur="inputChange('24')"></el-input>
											</el-form-item>
											<%=rb.getString("FenZhongNeiLianXuCuoWu")%>
											<el-form-item label='' prop="limitCount" class="validLimitCount">
												<el-input v-model="settingsForm.limitCount" placeholder='' class="w50 ml5 mr5"  @input="inputChange('25')"
													@blur="inputChange('25')"></el-input>
												<%=rb.getString("IPJiaRuHeiMingDan")%> 
											</el-form-item>
											<el-form-item label='' prop="limitTimes" class="validLimitTimes">
												<el-input v-model="settingsForm.limitTimes" placeholder='' class="w50 ml5 mr5"  @input="inputChange('26')"
													@blur="inputChange('26')"></el-input>
												<%=rb.getString("ZiDongShiFang")%>
											</el-form-item>
										</div>	
										<div style='margin-left: 250px;'>
											<span class="ml5 error" v-if='showError.limitMinusErrorNum'>{{errorStr.limitMinusErrorNum}}</span>
											<span class="ml5 error" v-if='showError.limitCountErrorNum'>{{errorStr.limitCountErrorNum}}</span>
											<span class="ml5 error" v-if='showError.limitTimesErrorNum'>{{errorStr.limitTimesErrorNum}}</span>
										</div>
									</el-form-item>
									<!-- false 英文 -->
									<el-form-item label='<%=rb.getString("IPXianLiu")%>' prop="" class="mb10" v-if="isLocalZH == false">
										<div style="display:flex">
											<%=rb.getString("DangZai")%>
											<el-form-item label='' prop="limitCount" class="validLimitCount">
												<el-input v-model="settingsForm.limitCount" placeholder='' class="w50 ml5 mr5"  @input="inputChange('25')"
													@blur="inputChange('25')"></el-input>
											</el-form-item>
											<%=rb.getString("FenZhongNeiLianXuCuoWu")%>
											<el-form-item label='' prop="limitMinus" class="validLimitMins">
												<el-input v-model="settingsForm.limitMinus" placeholder='' class="w50 ml5 mr5" @input="inputChange('24')"
													@blur="inputChange('24')"></el-input>
												<%=rb.getString("IPJiaRuHeiMingDan")%> 
											</el-form-item>
											<el-form-item label='' prop="limitTimes" class="validLimitTimes">
												<el-input v-model="settingsForm.limitTimes" placeholder='' class="w50 ml5 mr5"  @input="inputChange('26')"
													@blur="inputChange('26')"></el-input>
												<%=rb.getString("ZiDongShiFang")%>
											</el-form-item>
										</div>	
										<div style='margin-left: 250px;'>
											<span class="ml5 error" v-if='showError.limitCountErrorNum'>{{errorStr.limitCountErrorNum}}</span>
											<span class="ml5 error" v-if='showError.limitMinusErrorNum'>{{errorStr.limitMinusErrorNum}}</span>
											<span class="ml5 error" v-if='showError.limitTimesErrorNum'>{{errorStr.limitTimesErrorNum}}</span>
										</div>	
									</el-form-item>
									
									<el-form-item label='<%=rb.getString("SuoPingShiJian")%>' prop="userSessionExpirationMin" class="suoping mb7">
										<%=rb.getString("YongHuFeiHuoDongZhuangTai")%>
										<el-input v-model="settingsForm.userSessionExpirationMin" @input="inputChange('6')" @blur="inputChange('6')" class="w50 ml5 mr5"></el-input>
										<%=rb.getString("QingSuoPing")%>
										<span class="ml5 error" v-if='showError.suoping'>{{errorStr.suoping}}</span>
									</el-form-item>
									<div class="ml250 mb7">
										<el-form-item label='' prop="isBrowserAutoRecordPass">
											<el-checkbox style="margin-right: 5px;" v-model="settingsForm.isBrowserAutoRecordPass" :true-label="true" :false-label="false"></el-checkbox>
											<%=rb.getString("KaiQiLiuLanQiJiLuMiMa")%>
										</el-form-item>
									</div>
									

									<el-form-item label="<%=rb.getString("ZiDongSuoDing")%>" prop="autoLockUserDayEnable">
										<el-checkbox v-model='settingsForm.autoLockUserDayEnable' name='autoLockUserDayEnable' true-label="1" false-label="0"></el-checkbox>
										<span class="ml5"> 
											<%=rb.getString("ZiDongSuoDingTiShiPre")%>
											<el-form-item style="width: 60px;display: inline-block;" prop="autoLockUserDay" label-position="left">
												<el-input v-model="settingsForm.autoLockUserDay" class="w50 ml5 mr5" maxlength="5"></el-input>
											</el-form-item>
											<%=rb.getString("ZiDongSuoDingTiShiSuf")%>
											<span class="ml5 error" v-if='showError.suoding'>{{errorStr.suoding}}</span>
										</span>
									</el-form-item>
									<el-form-item label="<%=rb.getString("ZuiDaHuiHuaXianZhi")%>" prop="isOnlyOneUserLoginEnable">
										<el-checkbox v-model='settingsForm.isOnlyOneUserLoginEnable' name='isOnlyOneUserLoginEnable' true-label="0" false-label="1"></el-checkbox>
										<span class="ml5"> <%=rb.getString("ZuiDaHuiHuaXianZhiTiShi")%></span>
									</el-form-item>

									<el-form-item label='<%=rb.getString("QIYongDengLuHouTiShi")%>' prop="enabledFlag" style="height:100px">
										<el-checkbox v-model='settingsForm.enabledFlag' name='type' @change="checkChange('4')"></el-checkbox>
										<span class="ml5"> <%=rb.getString("TongZhiYongHuXiaoXi")%></span>
										<div class='textInput'>
											<el-form-item label='' prop='msg' class="msg">
												<el-input type='textarea' v-model='settingsForm.msg' @input="inputChange('19')" @blur="inputChange('19')" :disabled=!settingsForm.enabledFlag></el-input>
											</el-form-item>
											<span class="ml5 error" v-if='showError.msg'>{{errorStr.msg}}</span>
										</div>
									</el-form-item>
								</div>
							</div>
							<!--设备设置-->
							<div class='modelBox'>
								<div class="group-title not-extend">
									<span class="title-icon"></span><span class="title-text"><%=rb.getString("SheBeiTongJiSheZhi")%> </span>
								</div>
								<div class="plr15">
									<div class=" " v-if="conpetence.is_super_user">
										<el-form-item label='<%=rb.getString("SheBeiInformZhouQi")%>' class="mb1">
											<!--enb 心跳周期  开关： 控制是否检测Inform 周期并自动调整 只和心跳周期有关；-->
											<el-form-item label='' prop="enbInformPeriod" class="enbPeriod mb7" style='margin-right: 16px;'>
												<el-checkbox v-model='settingsForm.enbInformPeriodAdjustEnable' true-label="1" false-label="0" @change="checkChange('10')"></el-checkbox>
													<%=rb.getString("SheBeiQiDongENBInformZhouQiBuFuHe")%>
												<el-input v-model="settingsForm.enbInformPeriod" :disabled='settingsForm.enbInformPeriodAdjustEnable == "0"'  @input="inputChange('20')" @blur="inputChange('20')" style="width:60px;" class="ml5"></el-input>
													<%=rb.getString("ZiDongTiaoZheng")%>
												<span class="ml5 error" v-if='showError.enbPeriod'>{{errorStr.enbPeriod}}</span>
											</el-form-item>
											<!--enb 超时时间 -->
											<el-form-item label='' prop="enbTimeout" class="wuxiangying mb7" style='margin-left: 250px;'>
													<%=rb.getString("GuiDingShiJianWuXIangYing")%>
												<el-input v-model="settingsForm.enbTimeout" @input="inputChange('7')" @blur="inputChange('7')" style="width:60px;" class="ml5" ></el-input>
													<%=rb.getString("SheBeiJiangHuiGuanJi")%>
													<span class="ml5 error" v-if='showError.wuxiangying'>{{errorStr.wuxiangying}}</span>
												</el-form-item>
										</el-form-item>
										<el-form-item prop='enbInformPeriodAdjustEnable' style="display:none;" label="" label-width="0px">
											<el-input v-model='settingsForm.enbInformPeriodAdjustEnable'></el-input>
										</el-form-item>
										<div style='margin-left: 250px;'>
											<!--CPE 心跳周期 -->
											<el-form-item label='' prop="cpeInformPeriod" class="cpePeriod mb7" style='margin-right: 16px;'>
												<el-checkbox v-model='settingsForm.cpeInformPeriodAdjustEnable' true-label="1" false-label="0" @change="checkChange('11')"></el-checkbox>
												<%=rb.getString("SheBeiQiDongCPEInformZhouQiBuFuHe")%>
												<el-input v-model="settingsForm.cpeInformPeriod" :disabled='settingsForm.cpeInformPeriodAdjustEnable == "0"' @input="inputChange('21')" @blur="inputChange('21')" style="width:60px;" class="ml5" ></el-input>
												<%=rb.getString("ZiDongTiaoZheng")%>
												<span class="ml5 error" v-if='showError.cpePeriod'>{{errorStr.cpePeriod}}</span>
											</el-form-item>
											<el-form-item prop='cpeInformPeriodAdjustEnable' style="display:none;" label="" label-width="0px">
												<el-input v-model='settingsForm.cpeInformPeriodAdjustEnable'></el-input>
											</el-form-item>
											<!--CPE 超时时间 -->
											<el-form-item label='' prop="cpeTimeout" class="cpeOnlineTimeout mb7">
												<%=rb.getString("GuiDingShiJianWuXIangYing")%>
												<el-input v-model="settingsForm.cpeTimeout" @input="inputChange('22')" @blur="inputChange('22')" style="width:60px;" class="ml5" ></el-input>
												<%=rb.getString("SheBeiJiangHuiGuanJi")%>
												
												<span class="ml5 error" v-if='showError.cpeOnlineTimeout'>{{errorStr.cpeOnlineTimeout}}</span>
											</el-form-item>
										</div>
									</div>
		
									<el-form-item label='<%=rb.getString("SheBeiMingChengTongBuSheZhi")%>' prop="nameSettingEnable" class="mb1">
										<el-checkbox v-model='settingsForm.nameSettingEnable' name='type' @change="checkChange('5')"></el-checkbox>
										<span class="ml5"><%=rb.getString("JianChaHeSheZhiXiangTongSheBeiMingCheng")%></span>
										<div>
											<el-checkbox v-model='settingsForm.prompt' name='type' @change="checkChange('6')"></el-checkbox>
											<span class="ml5"><%=rb.getString("TongZhiWoShiFoShouDongTongBu")%></span>
										</div>
									</el-form-item>
									<el-form-item prop='prompt' style="display:none;" label="" label-width="0px">
										<el-input v-model='settingsForm.prompt'></el-input>
									</el-form-item>
									<el-form-item v-show="isAdmin" label='<%=rb.getString("SheBeiFangWenKongZhi")%>' prop="accessContralEnable" class="mb7 CODE_ADVANCE_ACCESS_CONTROL hidden">
										<el-checkbox v-model='settingsForm.accessContralEnable' name='type' @change="checkChange('7')"></el-checkbox>
										<span class="ml5"><%=rb.getString("ZhiYunXuFuHe")%>
											<a href="javascript:;" @click='toGuizeInfo' title='跳转到规则页面'>
												<%=rb.getString("GuiZe")%>
											</a>
											<%=rb.getString("FangWenXiTong")%> .</span>
									</el-form-item>
									<el-form-item label='<%=rb.getString("SheBeiXinHaoQiangDuXianShi")%>' prop="">
										<div class="fl">
											<%=rb.getString("XinHaoQiangDuSheZHi")%>:
										</div>
										<div style="display:flex">
											<div class="week">
												<i class="el-icon el-icon-signal"></i>
												<%=rb.getString("XinHaoRuo")%> <span>< </span>
													<el-form-item label='' prop='rsrpVal0' class='rsrpVal0'>
														<el-input v-model="settingsForm.rsrpVal0" class="w50 ml5 mr5" @input="inputChange('8')" @blur="inputChange('8')">
													</el-form-item>
											</div>
											<div class="normal">
												<i class="el-icon el-icon-signal"></i>
												<%=rb.getString("XinHaoZhengChang")%> <span>< </span>
													<el-form-item label='' prop='rsrpVal1' class='rsrpVal1'>
														<el-input v-model="settingsForm.rsrpVal1" class="w50 ml5 mr5" @input="inputChange('9')" @blur="inputChange('9')">
													</el-form-item>
											</div>
											<div class="strong">
												<i class="el-icon el-icon-signal"></i>
												<%=rb.getString("XinHaoQiang")%>
													<span class="ml5 error" v-if='showError.rsrpVal0'>{{errorStr.rsrpVal0}}</span>
													<span class="ml5 error" v-if='showError.rsrpVal1'>{{errorStr.rsrpVal1}}</span>
											</div>
										</div>
									</el-form-item>
									<!-- UE Signal -->
									<el-form-item label='<%=rb.getString("UESheBeiXinHaoQiangDuXianShi")%>' prop="">
										<div class="fl">
											<%=rb.getString("XinHaoQiangDuSheZHi")%>:
										</div>
										<div style="display:flex">
											<div class="week">
												<i class="el-icon el-icon-signal"></i>
												<%=rb.getString("XinHaoRuo")%> <span>< </span>
													<el-form-item label='' prop='uersrpVal0' class='uersrpVal0'>
														<el-input v-model="settingsForm.uersrpVal0" class="w50 ml5 mr5" @input="inputChange('27')" @blur="inputChange('27')">
													</el-form-item>
											</div>
											<div class="normal">
												<i class="el-icon el-icon-signal"></i>
												<%=rb.getString("XinHaoZhengChang")%> <span>< </span>
													<el-form-item label='' prop='uersrpVal1' class='uersrpVal1'>
														<el-input v-model="settingsForm.uersrpVal1" class="w50 ml5 mr5" @input="inputChange('28')" @blur="inputChange('28')">
													</el-form-item>
											</div>
											<div class="strong">
												<i class="el-icon el-icon-signal"></i>
												<%=rb.getString("XinHaoQiang")%>
													<span class="ml5 error" v-if='showError.uersrpVal0'>{{errorStr.uersrpVal0}}</span>
													<span class="ml5 error" v-if='showError.uersrpVal1'>{{errorStr.uersrpVal1}}</span>
											</div>
										</div>
									</el-form-item>
		
									<el-form-item label='<%=rb.getString("JiZhanWenJianShangChuangXieYi")%>' prop="uploadSelected" class="mb7" v-if="conpetence.is_super_user">
										<el-select v-model="settingsForm.uploadSelected">
											<el-option label="http" value="1"></el-option>
											<el-option label="https" value="2"></el-option>
											<el-option label="<%=rb.getString("BaoChiJiZhanBuBian")%>" value="3"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("HuiShouZhan")%>' class="mb10 settingsRecycleBinCls">
										<div style="display:flex;">
											<el-form-item label='' prop="deviceOfflineEnable" class="mb7" style='margin: 8px 10px 0 0;'>
												<el-checkbox v-model='settingsForm.deviceOfflineEnable' true-label="1" false-label="0"></el-checkbox>
											</el-form-item>

											<%=rb.getString("XiTongSheZhiHuiShouZhanTiShi1")%>
											<el-form-item label='' prop="deviceOfflineSaveDay">
												<el-input v-model.trim="settingsForm.deviceOfflineSaveDay" type="text" class="w50 ml5 mr5"></el-input>
												<%=rb.getString("XiTongSheZhiHuiShouZhanTiShi2")%>
											</el-form-item>
										</div>
										<div style="margin-left:250px ;color:rgba(0, 0, 0,0.32)">
											<%=rb.getString("XiTongSheZhiHuiShouZhanGouXuanTiShi")%>
										</div>
									</el-form-item>
									<!-- GPS Detection Setting -->
									<el-form-item label="<%=rb.getString("JiZhanQuJiHuo")%>" class="mb10 settingsRecycleBinCls">
										<div style="display:flex;">
											<el-form-item label='' prop="locationDetection" class="mb7" style='margin: 8px 10px 0 0;'>
												<el-checkbox v-model='settingsForm.locationDetection' true-label="1" false-label="0"></el-checkbox>
											</el-form-item>

											<%=rb.getString("SheBeiJingWeiDuJianCeTiShi1")%>
											<el-form-item label='' prop="latitudeToleranceRange">
												<el-input v-model.trim="settingsForm.latitudeToleranceRange" type="text" class="w50 ml5 mr5" @change="latToleranceRangeChange"></el-input>
												<%=rb.getString("SheBeiJingWeiDuJianCeTiShi2")%>
											</el-form-item>
										</div>
									</el-form-item>
								</div>
							</div>
							<!--通知设置-->
							<div class='modelBox' v-if='isAdmin'>
								<div class="group-title not-extend">
									<span class="title-icon"></span><span class="title-text"><%=rb.getString("TongZhiSheZhi")%></span>
								</div>
								<div class="plr15 tongzhiBox">
									<el-form-item label='<%=rb.getString("TongZhiFuWuQi")%>'>
										<el-checkbox v-model='settingsForm.emEnabel' name='type' @change="checkChange('8')"></el-checkbox>
										<span class="ml5"><%=rb.getString("TongZhiYouJianFuWuQi")%></span>
										<el-button type="primary" size='mini' @click='showWindowInfo=true' class='ceshiEmailBtn' :disabled='!settingsForm.emEnabel'>
												<%=rb.getString("CeShi")%>
											</el-button>
											<span class="yanzhengsuceess" v-show='verificationMail.success'>
												<i class="el-icon el-icon-circle-success"></i>
												<%=rb.getString("YanZhengTongGuo")%>
											</span>
											<span class="yanzhengerror" v-show='verificationMail.error'>
												<i class="el-icon el-icon-circle-close"></i>
												<%=rb.getString("YouXiangBuShiYouXiaoDe")%>
											</span>
										<div>
											<el-col class="notifyBox screenWidth" style="width: 320px;">
												<el-form-item label='<%=rb.getString("YouXiang")%>' prop="mailUsername" class="mailUsername">
													<el-input v-model="settingsForm.mailUsername" @input="inputChange('10')" @blur="inputChange('10')" :disabled=!settingsForm.emEnabel></el-input>
												</el-form-item>
											</el-col>
											<el-col :span='4' class="notifyBox tongzhiLabel screenWidth4">
												<el-form-item label='<%=rb.getString("MiMa")%>' prop="mailPassword" class="mailPassword">
													<el-password v-model="settingsForm.mailPassword" size="mini" placeholder="" show-password maxlength='50' :disabled="!settingsForm.emEnabel" @input="inputChange('11')" @blur="inputChange('11')" class="w120"></el-password>
													<el-input v-model="settingsForm.mailPassword" style="display: none;"></el-input>
												</el-form-item>
											</el-col>
											<el-col :span='5' class="notifyBox">
												<ul class="tongzhiLi">
													<li><span class="ml5 error" v-if='showError.mailUsername'>{{errorStr.mailUsername}}</span></li>
													<li><span class="ml5 error" v-if='showError.mailPassword'>{{errorStr.mailPassword}}</span></li>
													<li><span class="ml5 error" v-if='showError.mailHost'>{{errorStr.mailHost}}</span></li>
													<li><span class="ml5 error" v-if='showError.mailPort'>{{errorStr.mailPort}}</span></li>
												</ul>
											</el-col>
										</div>
										<div class="doubleBox" style="width: calc(90% - 260px);">
											<el-col :span='5' class="notifyBox screenWidth" style="width: 320px;">
												<el-form-item label='<%=rb.getString("SMTPFuWuQi")%>' prop="mailHost" class="mailHost">
													<el-input v-model="settingsForm.mailHost" @input="inputChange('12')" @blur="inputChange('12')" :disabled=!settingsForm.emEnabel></el-input>
												</el-form-item>
											</el-col>
											<el-col :span='5' class="notifyBox tongzhiLabel screenWidth">
												<el-form-item label='<%=rb.getString("DuanKou")%>' prop="mailPort" class="mailPort">
													<el-input v-model="settingsForm.mailPort" class="w50 ml5" @input="inputChange('13')" @blur="inputChange('13')" :disabled=!settingsForm.emEnabel></el-input>
												</el-form-item>
											</el-col>
										</div>
									</el-form-item>
									<el-form-item prop='emEnabel' style="display:none;" label="" label-width="0px">
										<el-input v-model='settingsForm.emEnabel'></el-input>
									</el-form-item>
								</div>
							</div>
							
							
							<!--存储设置-->
							<div class='modelBox' v-if="isAdmin">
								<div class="group-title not-extend">
									<span class="title-icon"></span><span class="title-text"><%=rb.getString("CunChuSheZhi")%></span>
								</div>
								<div class="plr15">
									<!--日志设置-->
									<ul class="second-group-title"><li><%=rb.getString("KPISheBei")%></li></ul>
									<div class="second-group">
										<!--日志设置-->
										
										<div class="split-group-title"><%=rb.getString("RiZhiSheZhi")%></div>
										<div class="split-group">
											<el-form-item label='<%=rb.getString("RiZhiCunChu")%>' class="mb1">
												<el-form-item prop="logDataSaveDays">
													<!--<el-checkbox v-model='settingsForm.logSheBei' label='' name='type' @change="checkChange('8')"></el-checkbox>-->
													<span><%=rb.getString("SheBeiYuanShiWenJianCunChu")%></span>
													<el-select v-model="settingsForm.logDataSaveDays" placeholder="" class="loglanguage" size='mini'>
														<el-option label='1' value="30"></el-option>
														<el-option label='3' value="90"></el-option>
														<el-option label='6' value="180"></el-option>
													</el-select>
													<%=rb.getString("Yue")%>.
												</el-form-item>
											</el-form-item>
											<el-form-item label='<%=rb.getString("YiChangRiZhiCunChu")%>' class="mb1">
												<el-form-item prop="rebootLogDataSaveDays">
													<span><%=rb.getString("SheBeiYuanShiWenJianCunChu")%></span>
													<el-select v-model="settingsForm.rebootLogDataSaveDays" placeholder="" class="loglanguage" size='mini'>
														<el-option label='1' value="1"></el-option>
														<el-option label='7' value="7"></el-option>
														<el-option label='30' value="30"></el-option>
														<el-option label='60' value="60"></el-option>
														<el-option label='90' value="90"></el-option>
													</el-select>
													<%=rb.getString("Tian")%>.
												</el-form-item>
												<el-form-item prop="rebootLogSaveCount" style="margin-left:250px;">
													<span><%=rb.getString("BaoLiuYiChangRiZhiCiShu")%></span>
													<el-select v-model="settingsForm.rebootLogSaveCount" placeholder="" class="loglanguage" size='mini'>
														<el-option label='1' value="1"></el-option>
														<el-option label='2' value="2"></el-option>
													</el-select>
													<%=rb.getString("GengDuoRiZhiJiangFuGai")%>.
												</el-form-item>
											</el-form-item>
											<!-- 需求单69633 操作日志存储时长设置 -->
											<el-form-item label='<%=rb.getString("CaoZuoRiZhi")%>' class="mb1">
												<el-form-item prop="sysOperateLogDataSaveDays">
													<span><%=rb.getString("CaoZuoRiZhiCunChuShiChang")%></span>
													<el-select v-model="settingsForm.sysOperateLogDataSaveDays" placeholder="" class="loglanguage" size='mini'>
														<el-option label='3' value="90"></el-option>
														<el-option label='6' value="180"></el-option>
														<el-option label='12' value="360"></el-option>
														<el-option label='24' value="720"></el-option>
														<el-option label='36' value="1080"></el-option>
													</el-select>
													<%=rb.getString("Yue")%>.
												</el-form-item>
				
											</el-form-item>
											<el-form-item label='<%=rb.getString("YuanChengCunChu")%>' v-if="isLocal">
												<el-checkbox v-model='settingsForm.logFtpEnable' name='type' @change="checkChange('9')"></el-checkbox>
												<span class="ml5"><%=rb.getString("RiZhiZhuanFaDaoYuanChengDiZhi")%></span>
												<div class="tongzhiBox">
													<el-col class="notifyBox mt10 screenWidth7">
														<el-form-item label='<%=rb.getString("FTPXieYi")%>' prop="logFtpType">
															<el-select v-model="settingsForm.logFtpType" placeholder="" class="language" size='mini' :disabled=!settingsForm.logFtpEnable>
																<el-option label='SFTP' value="sftp"></el-option>
																<el-option label='FTP' value="ftp"></el-option>
															</el-select>
														</el-form-item>
													</el-col>
													<el-col class="notifyBox mt10 screenWidth7">
														<el-form-item label='<%=rb.getString("ShangChuanLuJing")%>' prop="logFtpSavePath" class="logFtpSavePath">
															<el-input v-model="settingsForm.logFtpSavePath" value='/' class="w300 " :disabled=!settingsForm.logFtpEnable @input="inputChange('14')"
																@blur="inputChange('14')"></el-input>
														</el-form-item>
													</el-col>
													<el-col class="notifyBox screenWidth1">
														<ul class="tongzhiLi mt10">
															<li><span class="ml5 error" v-if='showError.logFtpSavePath'>{{errorStr.logFtpSavePath}}</span></li>
															<li><span class="ml5 error" v-if='showError.logFtpIpAddr'>{{errorStr.logFtpIpAddr}}</span></li>
															<li><span class="ml5 error" v-if='showError.logFtpPort'>{{errorStr.logFtpPort}}</span></li>
															<li><span class="ml5 error" v-if='showError.logFtpUser'>{{errorStr.logFtpUser}}</span></li>
															<li><span class="ml5 error" v-if='showError.logFtpPassword'>{{errorStr.logFtpPassword}}</span></li>
														</ul>
													</el-col>
													<div class="doubleBox">
														<el-col class="notifyBox screenWidth7">
															<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop="logFtpIpAddr" class="logFtpIpAddr">
																<el-input v-model="settingsForm.logFtpIpAddr" placeholder='' class="w120" :disabled=!settingsForm.logFtpEnable @input="inputChange('15')"
																	@blur="inputChange('15')"></el-input>
															</el-form-item>
														</el-col>
														<el-col class="notifyBox screenWidth7">
															<el-form-item label='<%=rb.getString("DuanKou")%>' prop="logFtpPort" class="logFtpPort">
																<el-input v-model="settingsForm.logFtpPort" placeholder='' class="w50 " :disabled=!settingsForm.logFtpEnable @input="inputChange('16')"
																	@blur="inputChange('16')"></el-input>
															</el-form-item>
														</el-col>
													</div>
													<div class="doubleBox">
														<el-col class="notifyBox screenWidth7">
															<el-form-item label='<%=rb.getString("YongHuMingCheng")%>' prop="logFtpUser" class="logFtpUser">
																<el-input v-model="settingsForm.logFtpUser" maxlength="60" placeholder='' class="w300" :disabled=!settingsForm.logFtpEnable @input="inputChange('17')"
																	@blur="inputChange('17')"></el-input>
															</el-form-item>
														</el-col>
														<el-col class="notifyBox screenWidth7">
															<el-form-item label='<%=rb.getString("MiMa")%>' prop="logFtpPassword" class="logFtpPassword">
																<el-password v-model="settingsForm.logFtpPassword" size="mini" placeholder="" show-password :disabled="!settingsForm.logFtpEnable" @input="inputChange('18')" @blur="inputChange('18')" class="w120"></el-password>
																<el-input v-model="settingsForm.logFtpPassword" style="display: none;"></el-input>
															</el-form-item>
														</el-col>
													</div>
												</div>
											</el-form-item>
											<el-form-item prop='logFtpEnable' style="display:none;" label="" label-width="0px">
												<el-input v-model='settingsForm.logFtpEnable'></el-input>
											</el-form-item>
										</div>
										<div class="split-group-title"><%=rb.getString("GaoJing")%></div>
										<div class="split-group">
											<el-form-item label='<%=rb.getString("LiShiGaoJingCunChuTianShu")%>' prop='alarmHisMaxHoldTime' class="mb13">
												<%=rb.getString("ShuJuKuShuJuCunChu")%>
													<el-input v-model="settingsForm.alarmHisMaxHoldTime" :disabled=true class="w50"></el-input>
													<%=rb.getString("DanDuTian")%>.
											</el-form-item>
										</div>

										<div class="split-group-title"><%=rb.getString("ZhiBiao")%></div>
										<div class="split-group">
											<el-form-item label='<%=rb.getString("KPIWenJianCunChuTianChu")%>' prop='kpiFilesSaveDays' class="mb13">
												<%=rb.getString("SheBeiBaoGaoCunChu")%>
													<el-input v-model="settingsForm.kpiFilesSaveDays" :disabled=true class="w50"></el-input>
													<%=rb.getString("DanDuTian")%>.
											</el-form-item>
											<el-form-item label='<%=rb.getString("KPIBaoBiaoWenJianCunChuTianShu")%>' prop='kpiReportDataSaveDays' class="mb13">
												<%=rb.getString("KPIBaoBiaoCunChu")%>
													<el-input v-model="settingsForm.kpiReportDataSaveDays" :disabled=true class="w50"></el-input>
													<%=rb.getString("DanDuTian")%>.
											</el-form-item>
											<!--#46114 Local版本系统设置可以修改KPI数据保存天数 -->							
											<el-form-item label='<%=rb.getString("KPIYuanShiShuJuCunChu")%>' prop='kpiStorge15DataDays' class="mb13">
												<%=rb.getString("KPIYuanShiShuJuZaiFuWuQiZuiDuoCunChu")%>
												<el-input v-model="settingsForm.kpiStorge15DataDays" :disabled=true class="w50"></el-input>
												<%=rb.getString("DanDuTian")%>.
											</el-form-item>
											<el-form-item label='<%=rb.getString("KPIXiaoShiShuJuCunChu")%>' prop='kpiStorge60DataDays' class="mb13" v-if="isAdmin && isLocal">
												<%=rb.getString("KPIXiaoShiShuJuZaiFuWuQiZuiDuoCunChu")%>
												<el-select v-model="settingsForm.kpiStorge60DataDays" class="kpiDataStorage" :disabled="!isAdmin || !isLocal"  size='mini'>
													<el-option label='30' value="30"></el-option>
													<el-option label='60' value="60"></el-option>
													<el-option label='90' value="90"></el-option>
												</el-select>
												<%=rb.getString("DanDuTian")%>.
											</el-form-item>	
											<el-form-item label='<%=rb.getString("KPITianShuJuCunChu")%>' prop='kpiStorge1440DataDays' class="mb13">
												<%=rb.getString("KPITianShuJuZaiFuWuQiZuiDuoCunChu")%>
												<el-input v-model="settingsForm.kpiStorge1440DataDays" :disabled=true class="w50"></el-input>
												<%=rb.getString("DanDuTian")%>.
											</el-form-item>	
											<el-form-item label='<%=rb.getString("KPIZhouYueChaXunLiDu")%>' prop='kpiWeekAndMonthSwitch' class="mb13">
												<el-checkbox v-model='settingsForm.kpiWeekAndMonthSwitch' true-label="1" false-label="0"></el-checkbox>	
												<span class="ml5"><%=rb.getString("ZhiChiZhouYueChaXunLiDu")%></span>			
											</el-form-item>	
										</div>

										<div class="split-group-title"><%=rb.getString("MR")%></div>
										<div class="split-group">
											<el-form-item label='<%=rb.getString("MRCunChuTianShu")%>' prop='mrFileSaveDays' class="mb13">
												<%=rb.getString("SheBeiBaoGaoYuanShiWenJianCunChu")%>
													<el-input v-model="settingsForm.mrFileSaveDays" :disabled=true class="w50"></el-input>
													<%=rb.getString("DanDuTian")%>.
											</el-form-item>
										</div>

										<div class="split-group-title"><%=rb.getString("XinLingZhuiZong")%></div>
										<div class="split-group">
											<el-form-item label='<%=rb.getString("XinLingZhuiZongWenJianCunChu")%>' prop='signalingTraceSaveDays' class="mb13" >
												<%=rb.getString("SheBeiBaoGaoCunChu")%>
												<el-input v-model="settingsForm.signalingTraceSaveDays" :disabled=true class="w50"></el-input>
												<%=rb.getString("DanDuTian")%>
											</el-form-item>	
										</div>

									</div>
									<div v-if="rsysLogUIEnableFlag">
										<!-- 网管设置-->
										<ul class="second-group-title"><li>OMC</li></ul>
										<div class="second-group">
											<!-- syslog -->
											<div class="split-group-title"><%=rb.getString("XieYiFangShi")%></div>
											<div class="split-group ldapBoxCls" style="width: 95%;margin-bottom:20px;">
												<el-form-item label='<%=rb.getString("XieYiFangShiKaiGuan")%>' prop='rsysLogEnable' class="mb20 loapSwitchBox">
													<el-switch v-model="settingsForm.rsysLogEnable" @change="rsystemEnableChange" :active-value="'1'" :inactive-value="'0'"></el-switch>
													<el-button style="margin-left:10px;" type="primary" size='mini' @click="syslogTest" class='ceshiEmailBtn' :disabled='settingsForm.rsysLogEnable == "0" || settingsForm.rsysLogEnable == "" || settingsForm.rsysLogEnable == null'>
														<%=rb.getString("CeShi")%>
													</el-button>
													<span v-if="settingsForm.rsysLogTestStatus == 'true'" style="margin-left:10px;"  class="el-icon el-icon-operation-defaultBeta successIconCls"></span>
													<span v-if="settingsForm.rsysLogTestStatus == 'false'" style="margin-left:10px;"  class="el-icon el-icon-deactivate errorIconCls"></span>
												</el-form-item>
												<el-form-item label='IP' class="mb20" prop="rsysLogIp">
													<el-input v-model="settingsForm.rsysLogIp" class="w150"></el-input>
												</el-form-item>
												<el-form-item label='<%=rb.getString("DuanKou")%>' class="mb20" prop="rsysLogPort">
													<el-input v-model="settingsForm.rsysLogPort" class="w150"></el-input>
												</el-form-item>
											</div>
										</div>
									</div>

									<ul class="second-group-title"><li><%=rb.getString("CiPanGaoJing")%></li></ul>
									<div class="second-group" style="padding: 15px 20px;">
										<el-form-item label='<%=rb.getString("CiPanGaoJingTiShiBiaoTi")%>' class="mb1">
											<el-form-item prop="varDiskAlarmThresHold">
												<span style='margin-right: 4px'><%=rb.getString("CiPanGaoJingRiZhiMuLu")%></span>
												<el-select v-model="settingsForm.varDiskAlarmThresHold" placeholder="" class="Yylanguage">
													<el-option v-for="item in diskSpaceSelectOptions" :label="item.text" :value="item.value"></el-option>
												</el-select>
											</el-form-item>
		
											<el-form-item prop="homeDiskAlarmThresHold" style="margin-left:250px;">
												<span style='margin-right: 4px'><%=rb.getString("CiPanGaoJingShuJuMuLu")%></span>
												<el-select v-model="settingsForm.homeDiskAlarmThresHold" placeholder="" class="Yylanguage">
													<el-option v-for="item in diskSpaceSelectOptions" :label="item.text" :value="item.value"></el-option>
												</el-select>
											</el-form-item>
											<el-form-item prop="usrDiskAlarmThresHold" style="margin-left:250px;">
												<span style='margin-right: 4px'><%=rb.getString("CiPanGaoJingYingYongMuLu")%></span>
												<el-select v-model="settingsForm.usrDiskAlarmThresHold" placeholder="" class="Yylanguage">
													<el-option v-for="item in diskSpaceSelectOptions" :label="item.text" :value="item.value"></el-option>
												</el-select>
											</el-form-item>
											<el-form-item prop="rootDiskAlarmThresHold" style="margin-left:250px;">
												<span style='margin-right: 4px'><%=rb.getString("CiPanGaoJingGenMuLu")%></span>
												<el-select v-model="settingsForm.rootDiskAlarmThresHold" placeholder="" class="Yylanguage">
													<el-option v-for="item in diskSpaceSelectOptions" :label="item.text" :value="item.value"></el-option>
												</el-select>
											</el-form-item>
										</el-form-item>
									</div>
								</div>
							</div>
							
							<!--北向接口设置 -->
							<div class='modelBox northSet' v-if="isAdmin || isSubAdmin || hasNorthRole">
								<div class="group-title not-extend">
									<span class="title-icon"></span><span class="title-text"><%=rb.getString("BeiXiangJieKouSheZhi")%></span>
									<span><%=rb.getString("FuWuZhuangTai")%></span><i :class="northStatusCls" style="margin:0 10px 0 20px;"></i><span>{{northStatusText}}</span>
								</div>
								<div class="plr15">
									<el-form-item label='<%=rb.getString("IPDiZhi")%>' class="mb13" prop="northboundIp">
										<el-input v-model="settingsForm.northboundIp" class="w150" :disabled=true></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("DuanKou")%>' class="mb13" prop="northboundPort">
										<el-input v-model="settingsForm.northboundPort" class="w150" :disabled=true></el-input>
									</el-form-item>
									<div style="margin-right:40px;">
										<div style="position:relative;height:24px;">
											<div>
												<div class="newIconBoxCls-bt" v-if="settingsForm.northboundServiceStatus == '1'" style="right:6px;top:-10px;" @click="addNorthUser" tip="<%=rb.getString("TianJia")%>">
													<span class="el-icon el-icon-circle-add" ></span>
												</div>
												<div class="newIconBoxCls-bt" v-if="settingsForm.northboundServiceStatus == '0'" style="right:6px;top:-10px;" @click="addSASList">
													<span class="el-icon el-icon-circle-add disabled" ></span>
												</div>
											</div>
										</div>
										<el-ctable 
											id="northUserTable" 
											ref="northUserTable" 
											:data='northUserTableData' 
											:pagination="false" 
											:height='northUserListTableHight' 
											v-clickoutside="northUserHanderClose"
											style="border:1px solid #E9E9E9"
										>
											<el-table-column label='' width="50" prop="" v-if="settingsForm.northboundServiceStatus == '1'">
												<template slot-scope="scope">
													<div class="el-icon el-icon-operation-more curpo" @click="northUserOptClick(scope.row,event)"></div>
												</template>
											</el-table-column>
											<el-table-column prop="userEnable" width="80" label="<%=rb.getString("ShiFouQiYong")%>">
												<template slot-scope="scope">
													<div style="position:relative;">
														<el-switch
															v-model="scope.row.userEnable"
															active-color="#4D84FF" 
															inactive-color="#CFCFCF" 
															:active-value="'1'" 
															:inactive-value="'0'"
															:disabled="settingsForm.northboundServiceStatus == '0'? true : false"
														></el-switch>
														<div v-if="settingsForm.northboundServiceStatus == '1'" class="northUserEnableClickBox" @click="northUserEnableClick(scope.row)"></div>
														<div v-if="settingsForm.northboundServiceStatus == '0'" class="northUserEnableClickBox disabled"></div>
													</div>
												</template>
											</el-table-column>
											<el-table-column label='<%=rb.getString("YongHuMingCheng")%>' min-width="130" prop="userName"></el-table-column>
											<el-table-column label='<%=rb.getString("MiMa")%>' min-width="210" prop="userPwd"></el-table-column>
											<el-table-column label='<%=rb.getString("ChuangJianShiJian")%>' min-width="130" prop="responseTime" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-cmenu ref="northUserMenu" :data="northUserMenuList" @click="northUserClickMenu"></el-cmenu>
									</div>
								</div>
							</div>
							<!--SAS列表-->
							<div class="modelBox" v-if="isAdmin && isSAS != undefined">
								<div class="functionSet" style="margin-bottom:20px">
									<div class="group-title not-extend" style='margin-bottom:20px'>
										<span class="title-icon"></span><span class="title-text" style="width:94%">SAS</span>
									</div>
									<div style="display: flex;margin:0 0 15px 25px;">
										<label style='margin-right:20px;'><%=rb.getString("SASXinTiaoRiZhi")%></label>
										<div style="display: flex;">
											<el-switch v-model="settingsForm.mainLogHbEnable" active-color="#4d84ff" inactive-color="#bdc1c6"></el-switch>
										</div>
									</div>
									<div class="deviceLogContainer">
										<div style="width:96%;position:relative;">
											<span><%=rb.getString("SASProviderList")%></span>
											<div class="newIconBoxCls-bt" v-if='isSAS' style="right:5px;top:-14px;" @click="addSASList" tip="<%=rb.getString("TianJia")%>">
												<span class="el-icon el-icon-circle-add" ></span>
											</div>
										</div>
										<div style="width:96%;height:300px">
											<el-ctable id="" ref="ctableUpsList" url='${ctx}/cell/SAS/provider/list.action' :pagination="false" :height='height' v-clickoutside="handerClose">
												<el-table-column label='' min-width="30" prop="" v-if="isSAS">
													<template slot-scope="scope">
														<div class="el-icon el-icon-operation-more curpo" @click="optClick(scope.row,event)"></div>
													</template>
												</el-table-column>
												<el-table-column label='<%=rb.getString("SASProviderName")%>' min-width="130" prop="providerName"></el-table-column>
												<el-table-column label='<%=rb.getString("SASServerURL")%>' min-width="210" prop="url"></el-table-column>
												<el-table-column label='<%=rb.getString("TLSCert")%>' min-width="210" prop="device_name" show-overflow-tooltip>
													<template slot-scope="scope">
														<div v-if='scope.row.uploadSuccess === 1'>
															<span class="el-icon el-icon-status-yes"></span>{{scope.row.certName}}({{scope.row.validTimeStr}})
														</div>
														<div v-else>
															<span class="el-icon el-icon-status-close"></span>{{scope.row.certName}}({{scope.row.validTimeStr}})
														</div>
													</template>
												</el-table-column>
												<el-table-column label='<%=rb.getString("GengXinShiJian")%>' min-width="130" prop="updateTimeStr" show-overflow-tooltip></el-table-column>
											</el-ctable>
											<el-cmenu ref="menuActiveFault" :data="menus" @click="clickMenu"></el-cmenu>
										</div>
									</div>
								</div>
								<el-form-item prop='mainLogHbEnable' style="display:none;" label="" label-width="0px">
									<el-input v-model='settingsForm.mainLogHbEnable'></el-input>
								</el-form-item>
							</div>
							<!--LDAP协议 -->
							<div class='modelBox' v-if="isLocal">
								<div class="group-title not-extend">
									<span class="title-icon"></span>
									<span class="title-text">LDAP Protocol</span>
								</div>
								<div class="plr15 ldapBoxCls" style="width: 95%;">
									<el-form-item label='LDAP Enable' class="mb20 loapSwitchBox" prop="ldapEnable">
										<el-switch v-model="settingsForm.ldapEnable" @change="ldapEnableChange" :active-value="true" :inactive-value="false" ></el-switch>
										<el-button style="margin-left:10px;" type="primary" size='mini' @click="ldapTest('test')" class='ceshiEmailBtn' :disabled='!settingsForm.ldapEnable'>
											<%=rb.getString("CeShi")%>
										</el-button>
										<span v-if="settingsForm.ldapResult == 'true'" style="margin-left:10px;"  class="el-icon el-icon-operation-defaultBeta successIconCls"></span>
										<span v-if="settingsForm.ldapResult == 'false'" style="margin-left:10px;"  class="el-icon el-icon-deactivate errorIconCls"></span>
									</el-form-item>
									<el-form-item label='LDAP IP' class="mb20" prop="ldapIp">
										<el-input v-model="settingsForm.ldapIp" class="w150"></el-input>
									</el-form-item>
									<el-form-item label='LDAP Port' class="mb20" prop="ldapPort">
										<el-input v-model="settingsForm.ldapPort" class="w150"></el-input>
										<el-form-item label='' prop="ldapSSL" class="ldapSSLBoxCls">
											<el-checkbox v-model="settingsForm.ldapSSL" :true-label="true" :false-label="false" @change="ldapSSLChange">SSL/TLS</el-checkbox>
										</el-form-item>
									</el-form-item>
									<el-form-item label='LDAP Base' class="mb20" prop="ldapBase">
										<el-input v-model="settingsForm.ldapBase" class="w150"></el-input>
									</el-form-item>
									<el-form-item label='LDAP User' class="mb20" prop="ldapUser">
										<el-input v-model="settingsForm.ldapUser" class="w150"></el-input>
									</el-form-item>
									<el-form-item label='LDAP Password' class="mb20 newPasswordCls" prop="ldapPwd">
										<el-password v-model="settingsForm.ldapPwd" size="mini" placeholder="" show-password class="w150"></el-password>
										<el-input v-model="settingsForm.ldapPwd" style="display: none;"></el-input>
									</el-form-item>
								</div>
							</div>
							
							
						</el-form>
						
						<el-dialog title="<%=rb.getString("YouXiangTanChuKuangBiaoTi")%>" :visible.sync="showWindowInfo" ref="slide" :width="sildeWidth" :append-to-body="true" :height="sildeHeight" class='setEmailBox'
							:close-on-click-modal="false" :url="dialogUrl"  @close="closeSettingInfo">
							<span><%=rb.getString("YouXinagTanChuKuangShuRu")%></span>
							<el-input v-model="receiveMail" @input='emailTest'></el-input>
							<div class="el-form-item__error" v-if='isreceiveMailShow' style='margin-left:210px;position:inherit'>{{errorStr.receiveMail}}</div>
							<div style="position: absolute;bottom:10px;right:20px">
								<el-button type="primary" @click='ceshiEmail'><%=rb.getString("QueDing")%></el-button>
								<el-button @click="showWindowInfo=false"><%=rb.getString("QuXiao")%></el-button>
							</div>
						</el-dialog>
						<el-dialog :title='northUserDialogTitle' id="northUserDialog" :visible.sync="showNorthUserDialog" top="15vh" ref="northUserDialog" width="600px" 
							:close-on-click-modal="false" @close="northUserDialogClose"  append-to-body>
							<el-form  :model="northUserDialogForm" ref="northUserDialogForm" :rules="northUserDialogRules" label-position="left" id="northUserDialogForm" :hide-required-asterisk='true'>
								<el-form-item label='<%=rb.getString("YongHuMingCheng")%>' prop="userName" label-width="140px">
									<el-input v-model="northUserDialogForm.userName" style="padding-top:8px;" class="w150"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("MiMa")%>' prop="userPwd" label-width="140px">
									<el-input v-model="northUserDialogForm.userPwd" style="padding-top:8px;" class="w150"></el-input>
								</el-form-item>
								<el-form-item prop='emailEnable' label="<%=rb.getString("ShiFouQiYong") %>" label-width="140px">
									<el-switch 
										v-model="northUserDialogForm.userEnable"
										active-color="#4D84FF" 
										inactive-color="#CFCFCF" 
										:active-value="'1'" 
										:inactive-value="'0'" 
										style="padding-top:8px;"
									></el-switch>
								</el-form-item>
							</el-form>
							<span slot="footer">
								<div>
									<el-button type="primary" @click="northUserDialogSubmit"><%=rb.getString("QueDing")%></el-button>
									<el-button @click="showNorthUserDialog = false"><%=rb.getString("QuXiao")%></el-button>
								</div>
							</span>
						</el-dialog>
						
					</div>
					<div class="footer CODE_SYSTEM_SETTINGS hidden">
						<div class="lnkbuttonGroup">
							<el-button type="primary" @click="addStting">
								<%=rb.getString("QueDing")%>
							</el-button>
							<el-button @click="close">
								<%=rb.getString("QuXiao")%>
							</el-button>
						</div>
					</div>
				</div>
			</el-tab-pane>
			<el-tab-pane v-if="isAdmin && isLocal" label="<%=rb.getString("UIDingZhiHua")%>" name="UICustom" id="uiCutomerCont">
			</el-tab-pane>
		</el-tabs>
		<el-slide ref="slider" :url="slideUrl" :title="slideTitle" :footer="footerShow" position="top" :header='headerShow' :position="slidePosition"
			:modal="true" :height="sliderHeight" :width="sliderWidth" @cancel='cancelSlide' @success="openDialogSuc">
		</el-slide>
	</div>
</div>
<script type="text/javascript">
	//创建vue实例
	var settingsVue = new Vue({
		el: '#settingTables',
		data() {
			//运营商名称验证
			let mrVendorValidator = (rule, value, cb) => {
				if (value === '') {
					cb('<%=rb.getString("QingShuRu")%><%=rb.getString("ChangShang")%>');
				} else {
					cb()
				}
			};
			// 网管名称
			let mrOMCNameValidator = (rule, value, cb) => {
				if (value === '') {
					cb('<%=rb.getString("QingShuRu")%><%=rb.getString("OMCMingCheng")%>');
				} else {
					cb()
				}
			};
			let northUserNameValidator = (rule, value, cb) => {
				
				var reg = /^[\w+$]{1,50}$/;
				if(value){
					if (reg.test(value)) {
						cb();
					} else {
						cb(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%>1-50'));
					}
				}else{
					cb(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%>1-50'));
				}
			};
			let northPwdValidator = (rule, value, callback) => {
				var vm = this;
				axios.post('${ctx}/system/sysuser/queryPasswordRule.action').then(function(response){
					var data = response.data,
						minLength = 1,
						maxLength = 40,
						passwordContent = data.password_content,
						reg = /^(?![A-Za-z0-9]+$)(?![a-z0-9_!@#$%^&*?]+$)(?![A-Za-z_!@#$%^&*?]+$)(?![A-Z0-9_!@#$%^&*?]+$)[a-zA-Z0-9_!@#$%^&*?]{1,40}$/;
					if (passwordContent == "0") {
						// 密码组成规则有限制
						if (!(value.length >= parseInt(minLength) && value.length <= parseInt(maxLength))) {
							callback(new Error('<%=rb.getString("MiMaChangDuYingGaiWei")%>1-40<%=rb.getString("Ge")%><%=rb.getString("ZiFu")%>'));
						}else if (!reg.test(value)) {
							callback(new Error('<%=rb.getString("MiMaBiXuLiangZhongLeiXing")%>'))
						} else {
							// 密码OK
							callback();
						}
					} else {
						if (!(value.length >= parseInt(minLength) && value.length <= parseInt(maxLength))) {
							callback(new Error('<%=rb.getString("MiMaChangDuYingGaiWei")%>1-40<%=rb.getString("Ge")%><%=rb.getString("ZiFu")%>'));
						}else {
							// 密码OK
							callback();
						}
					}
				})
			};
			let ldapValidator = (rule, value, cb) => {
				if(value){
					cb()
				}else{
					if(this.settingsForm.ldapEnable){
						cb('<%=rb.getString("ShuRuBiTianXiang")%>');
					}else{
						cb()
					}
					
				}
			};
			let rsysLogIPValidator = (rule, value, cb) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				
				if(this.settingsForm.rsysLogEnable == '1'){
					if(value == '' || !(reg.test(value) || reg.test(value))){
						cb(new Error("<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>"));
					}else{
						cb();
					}
				}else{
					cb()
				}
			};
			let rsysLogPortValidator = (rule, value, cb) => {
				var regNum = /^\d+$/;
				if(this.settingsForm.rsysLogEnable == '1'){
					if (value !== '') {
						if (!regNum.test(value) || (value < 0 || value > 65535)) {
							cb(new Error("<%=rb.getString("QingShuRuYouXiaoDuanKou")%>"));
						} else {
							cb();
						}
					} else {
						cb(new Error("<%=rb.getString("QingShuRuYouXiaoDuanKou")%>"));
					}
				} else {
					cb();
				}
			};
			let deviceOfflineSaveDayValidator = (rule, value, cb) => {
				var regNum = /^\d+$/;
				if(this.settingsForm.deviceOfflineEnable == '1'){
					if (value !== '') {
						if (!regNum.test(value) || value < 1 ) {
							cb(new Error('<%=rb.getString("FanWei")%>: <%=rb.getString("DaYu")%> 0,<%=rb.getString("ZhengXing")%>'));
						} else {
							cb();
						}
					} else {
						cb(new Error('<%=rb.getString("FanWei")%>: <%=rb.getString("DaYu")%> 0,<%=rb.getString("ZhengXing")%>'));
					}
				} else {
					cb();
				}
			};
			let latitudeToleranceRangeValidator = (rule, value, cb) => {
				var regNum = /^\d+$/;
				if(this.settingsForm.locationDetection == '1'){
					if (value !== '') {
						if (!regNum.test(value) || value < 1 ) {
							cb(new Error('<%=rb.getString("FanWei")%>: <%=rb.getString("DaYu")%> 0,<%=rb.getString("ZhengXing")%>'));
						} else {
							cb();
						}
					} else {
						cb(new Error('<%=rb.getString("FanWei")%>: <%=rb.getString("DaYu")%> 0,<%=rb.getString("ZhengXing")%>'));
					}
				} else {
					cb();
				}
			};
			let validMinPasswordLength = (rule, value, cb) => {
					var intReg = /^[0-9]{1,}$/;

					if(isNaN(value) || value === '' || !intReg.test(value)) {
						cb('<%=rb.getString("QingShuRuYiGeShuZi")%>, <%=rb.getString("ZhengXing")%>');
					}else {
						if(value - 6 < 0) {
							cb('<%=rb.getString("ZuiXiaoZhi")%>: 6');
						}else {
							cb();
						}
					}
				},
				validMaxPasswordLength = (rule, value, cb) => {
					var minLength = this.settingsForm.pwdMinLength,
						intReg = /^[0-9]{1,}$/;

					if(isNaN(value) || value === '' || !intReg.test(value)) {
						cb('<%=rb.getString("QingShuRuYiGeShuZi")%>, <%=rb.getString("ZhengXing")%>');
					}else if(value - minLength < 0) {
						cb('<%=rb.getString("FanWeiZuiDaZhiDaYuZuiXiaoZhi")%>');
					}else {
						cb();
					}
				},
				validAutoLockUserDay = (rule, value, cb) => {
					var trimVal = value.trim(),
						reg = /^\d*$/;

					if(reg.test(trimVal)) {
						if(this.settingsForm.autoLockUserDayEnable == 1 && (trimVal === '' || trimVal == '0')) {
							cb('<%=rb.getString("ZhengXing")%>: [1, ~]')
						}else {
							cb()
						}
					}else {
						cb('<%=rb.getString("ZhengXing")%>: [1, ~]')
					}
				};

			return {
				activeName:'sysSettings',
				showWindowInfo:false,
				sildeWidth:'500px',
				sildeHeight:'400px',
				dialogUrl:'',
				receiveMail:'',
				isreceiveMailShow:false,
				verificationMail:{
					success:false,
					error:false,
					addEmail:false
				},
				height: '100%',
				width: '90%',
				menus: [],
				rowData: [],
				slideUrl: "",
				slideTitle: '',
				sliderHeight: '100%',
				sliderWidth: '100%',
				footerShow: false,
				headerShow: true,
				slidePosition: 'top',
				deleModel: false,
				isAdmin: true,
				isSAS: false,
				isLocal: isCloud != "true",
				ceshidis:true,
				settingsForm: {
					mrVendor: '',
					mrOMCName: '',
					timezoneCode: '',
					timeZoneId: '',
					languageCode: '',
					modifyPWD: false,
					defaultPasswd: '',
					passwordContent: false,
					expires: false,
					validPeriod: '',
					promptBeforeDays: '',
					unlockMinu: '',
					attemptTimes: '',
					userSessionExpirationMin: '',
					isBrowserAutoRecordPass:false,

					checkUserCodeEnable: 'true',
					pwdMinLength: 6,
					pwdMaxLength: 20,
					
					verifyEnable: false, // 验证码开关
					sumTimes: '', //验证码可尝试次数
					
					limitMinus: '', //ip 限流： Y 分钟内
					limitCount: '', //ip 限流： 密码连续错误 N 次，IP 将被加入黑名单
					limitTimes: '', //ip 限流： Y 分钟后该 IP 将被自动释放
					
					enabledFlag: false,
					msg: '',
					cellTimeout: '',
					enbInformPeriod: '60',
					enbTimeout: '',
					cpeInformPeriod: '60',
					cpeTimeout: '',
					//1-开启， 0-关闭
					enbInformPeriodAdjustEnable: '0',
					cpeInformPeriodAdjustEnable: '0',
					
					nameSettingEnable: false,
					prompt: false,
					accessContralEnable: false,
					rsrpVal0: '',
					rsrpVal1: '',
					uersrpVal0: '',
					uersrpVal1: '',
					uploadSelected: '',
					mailUsername: '',
					mailPassword: '',
					mailHost: '',
					mailPort: '',
					logSheBei: false,
					logDataSaveDays: '',
					sysOperateLogDataSaveDays:'90',
					rebootLogDataSaveDays:'',
					rebootLogSaveCount:'',
					logFtpEnable: false,
					logFtpType: '',
					logFtpSavePath: '/',
					logFtpIpAddr: '',
					logFtpPort: '',
					logFtpUser: '',
					logFtpPassword: '',
					alarmHisMaxHoldTime: '',
					kpiFilesSaveDays: "",
					kpiReportDataSaveDays: '',
					mrFileSaveDays: '',
					//Local版本系统设置可以修改KPI数据保存天数
					kpiStorge15DataDays: '',
					kpiStorge60DataDays: '30',
					kpiStorge1440DataDays: '',
					kpiWeekAndMonthSwitch: '0',
					signalingTraceSaveDays: '7',

					mainLogHbEnable:false,
					emEnabel: false,
					northboundIp:"",
					northboundPort:"",
					northboundName:"",
					northboundPassword:"",
					northboundServiceStatus:"0",

					ldapEnable:false,
					ldapIp:'',
					ldapPort:'',
					ldapBase:'',
					ldapUser:'',
					ldapPwd:'',
					ldapResult:'',
					ldapSSL:false,
					
					rsysLogEnable: '0',
					rsysLogIp: '',
					rsysLogPort: '',
					rsysLogTestStatus: '',

					varDiskAlarmThresHold: '10%',
                    homeDiskAlarmThresHold: '10%',
                    usrDiskAlarmThresHold: '10%',
                    rootDiskAlarmThresHold: '10%',
                    
					deviceOfflineEnable:'0',
					deviceOfflineSaveDay:'',

					locationDetection: '0',
					latitudeToleranceRange: '',
					longitudeToleranceRange: '',

					autoLockUserDay: '90',
					autoLockUserDayEnable: '0',
					isOnlyOneUserLoginEnable: '1'
				},
				northStatus:"",
				northStatusCls:"",
				northStatusText:"",
				rules: {
					mrVendor: [
						{ validator: mrVendorValidator }
					],
					mrOMCName: [
						{ validator: mrOMCNameValidator }
					],
					northboundName:[
						{ validator : northUserNameValidator}
					],
					northboundPassword :[
						{ validator : northPwdValidator }
					],
					ldapIp: [
						{ validator: ldapValidator }
					],
					ldapPort:[
						{ validator : ldapValidator }
					],
					ldapBase:[
						{ validator : ldapValidator }
					],
					ldapUser:[
						{ validator : ldapValidator }
					],
					ldapPwd:[
						{ validator : ldapValidator }
					],
					rsysLogIp: [
						{ validator : rsysLogIPValidator }
					],
					rsysLogPort: [
						{ validator : rsysLogPortValidator }
					],
					deviceOfflineSaveDay: [
						{ validator : deviceOfflineSaveDayValidator }
					],
					latitudeToleranceRange: [
						{ validator : latitudeToleranceRangeValidator }
					],
					pwdMinLength: [{validator : validMinPasswordLength}],
					pwdMaxLength: [{validator : validMaxPasswordLength}],
					autoLockUserDay: [{validator : validAutoLockUserDay}]
				},
				errorStr: { // 验证信息提示
					shibaicishu: '<%=rb.getString("QingShuRu")%><%=rb.getString("ChangShiDengLiCiShu")%>',
					suodingshijian: '<%=rb.getString("QingShuRu")%><%=rb.getString("ZhangHuZiDongJieSuoShiJian")%>',
					suoping: 'eNB: <%=rb.getString("QingShuRuHuiHuaYouXiaoQi")%>',
					wuxiangying: '<%=rb.getString("QingShuRuJiZhanChaoShiShiJian")%>',
					enbPeriod: 'eNB <%=rb.getString("InformZhouQi")%>',
					cpePeriod: 'CPE <%=rb.getString("InformZhouQi")%>',
					cpeOnlineTimeout: 'CPE: <%=rb.getString("QingShuRuHuiHuaYouXiaoQi")%>',
					rsrpVal0: '<%=rb.getString("RuoYUZhengChangXinHaoZhi")%>',
					rsrpVal1: '<%=rb.getString("RuoYUZhengChangXinHaoZhi")%>',
					uersrpVal0: '<%=rb.getString("RuoYUZhengChangXinHaoZhi")%>',
					uersrpVal1: '<%=rb.getString("RuoYUZhengChangXinHaoZhi")%>',
					morenmima: '<%=rb.getString("QingShuRuMiMa")%>',
					validPeriod: '<%=rb.getString("ZhengChangYuQiangXinHaoZhi")%>',
					promptBeforeDays: '<%=rb.getString("QingShuRu")%><%=rb.getString("TiQianTiShiTianShu")%>',
					mailUsername: '<%=rb.getString("QingShuRuShouJianRen")%>',
					mailPassword: '<%=rb.getString("QingShuRuMiMa")%>',
					mailHost: '<%=rb.getString("QingShuRuSMTPHost")%>',
					mailPort: '<%=rb.getString("QingShuRuDuanKou")%>',
					logFtpSavePath: '<%=rb.getString("QingShuRuShangChuanLuJing")%>',
					logFtpIpAddr: '<%=rb.getString("XinIPShuRuTiShi")%>',
					logFtpPort: '<%=rb.getString("QingShuRuDuanKou")%>',
					logFtpUser: '<%=rb.getString("QingShuRuYongHuMing")%>',
					logFtpPassword: '<%=rb.getString("QingShuRuMiMa")%>',
					msg: '<%=rb.getString("QingShuRu")%><%=rb.getString("DengLuHouXiaoXi")%>',
					receiveMail:'<%=rb.getString("YouXiangGeShiCuoWu")%>'
				},
				showError: { // 是否展示验证信息
					shibaicishu: false,
					suodingshijian: false,
					suoping: false,
					wuxiangying: false,
					enbPeriod: false,
					cpePeriod: false,
					cpeOnlineTimeout: false,
					rsrpVal0: false,
					rsrpVal1: false,
					uersrpVal0: false,
					uersrpVal1: false,
					morenmima: false,
					validPeriod: false,
					promptBeforeDays: false,
					mailUsername: false,
					mailPassword: false,
					mailHost: false,
					mailPort: false,
					logFtpSavePath: false,
					logFtpIpAddr: false,
					logFtpPort: false,
					logFtpUser: false,
					logFtpPassword: false,
					msg: false
				},
				queryTimeData: [], // 时区展示
				oldSEMin: '',// 用于后台定时刷新的时间
				oldCTime: '',// 用于后台的初始时间
				oldLanguage: '',
				verification: [ // 用于对表单的验证
					true, //默认密码
					true, // 密码有效期
					true, // 密码提示时间
					false, //登录错误次数
					false, // 账户锁定时间
					false, // 屏幕廀时间
					false, //设备离线时间eNB 
					false, // cpe强度
					false, // cep强度
					true, // 通知 --邮箱
					true, // 通知 --密码
					true, // 通知 --Smtp
					true, // 通知 --端口
					true, // 日志 --上传路径
					true, // 日志 --IP
					true, // 日志 --端口
					true, // 日志 --用户
					true, // 日志 --密码
					true, // 安全 --登录提示
					false, //inform周期-eNB 
					false, //inform周期-cpe
					false, //设备离线时间-cpe 21  
					false, //用户锁定-总失败次数  
					false, //ip 限流-X分钟内 
					false, //ip 限流-可连续错误次数 
					false, //ip 限流-ip 自动解锁时间 
					false, // ue强度
					false, // ue强度
				],// 验证信息的校验
				conpetence: { // 权限控制
					types: false,
					alarmConfirm: false,
					is_super_user: false
				},
				isReset: false, // 是否重置,
				
				defaultEmailName:'',
				defaultEmailPwd:'',
				defaultEmailHost:'',
				defaultEmailPort:'',
				checkEmailFlag:true,

				northUserTableData:[],
				northUserRowData:'',
				northUserMenuList:[],
				showNorthUserDialog:false,
				northUserDialogTitle:'',
				northUserOpType:'',
				northUserDialogForm:{
					userName:'',
					userPwd:'',
					userEnable:'0',
				},
				northUserDialogRules:{
					userName:[
						{ validator : northUserNameValidator}
					],
					userPwd :[
						{required:true,message:'<%=rb.getString("QingShuRuMiMa")%>',trigger:'blur'},
						{validator:northPwdValidator,trigger:'blur'}
					],
				},
                diskSpaceSelectOptions: [
                    { text:'10%', value:'10%'},
                    { text:'20%', value:'20%'},
                    { text:'30%', value:'30%'},
                    { text:'40%', value:'40%'},
                    { text:'50%', value:'50%'},
                    { text:'60%', value:'60%'},
                    { text:'70%', value:'70%'},
                    { text:'80%', value:'80%'},
                    { text:'90%', value:'90%'},
                ],
				rsysLogUIEnableFlag: false,
				defaultSettingParams:{}
			}
		},
		computed: {
			isSetable() {
				return writableMap["CODE_SYSTEM_SETTINGS"] == true;
			},
			MrFlag(){
				return writableMap["CODE_PERFORMANCE_MR"] == true
			},
			basicLimit(){
				return this.isAdmin && (writableMap["CODE_PERFORMANCE_MR"] || !this.conpetence.types)
			},
			northUserListTableHight(){
				var northUserList = this.northUserTableData || [];
				if(northUserList.length == 0){
					return '90px'
				}else if(northUserList.length <= 5){
					var nums = northUserList.length-1
					return (80 + nums*34)+'px'
				}else{
					return '250px'
				}
			},
			isSubAdmin() {

				return is_build_user == 'true';
			},
			hasNorthRole() {

				return writableMap.CODE_SYSTEM_NORTH_INTERFACE == true;
			},
		},
		watch:{
			//enb 0-关， 1- 开, 校验 enb 心跳周期
			'settingsForm.enbInformPeriodAdjustEnable':function(newVal,oldVal){
				var vm = this, value = vm.settingsForm.enbInformPeriod, regNum = /^\d+$/;
				
				if(newVal == '1'){
					if (value !== '') {
						if (!regNum.test(value) || (value < 60 || value > 3600)) {
							vm.errorStr.enbPeriod = '<%=rb.getString("FanWei")%>:60 ~ 3600';
							$('.enbPeriod input').addClass('errorBorder');
							vm.showError.enbPeriod = true;
							vm.verification[19] = false;
						} else {
							vm.showError.enbPeriod = false;
							$('.enbPeriod input').removeClass('errorBorder');
							vm.verification[19] = true;
						}
					} else {
						vm.errorStr.enbPeriod = '<%=rb.getString("QingShuRu")%> eNB <%=rb.getString("InformZhouQi")%>;'
						$('.enbPeriod input').addClass('errorBorder');
						vm.showError.enbPeriod = true;
						vm.verification[19] = false;
					}
				}else{
					vm.showError.enbPeriod = false;
					$('.enbPeriod input').removeClass('errorBorder');
					vm.verification[19] = true;
					
					//此时心跳周期为空，并且开关为关，超时时间为： 默认的心跳周期初始值+10
					if(vm.settingsForm.enbInformPeriod < 60){
						vm.settingsForm.enbTimeout = 70;
					}
				}
			},
			'settingsForm.enbInformPeriod':function(newVal,oldVal){
				var vm = this;
				if(newVal){
					if(newVal < 60){
						vm.settingsForm.enbTimeout = 70;
					}else{
						vm.settingsForm.enbTimeout = Number(newVal)+10;
					}
					vm.showError.wuxiangying = false
					$('.wuxiangying input').removeClass('errorBorder');
					vm.verification[6] = true;
				}else{
					//此时心跳周期为空，并且开关为关，超时时间为： 默认的心跳周期初始值+10
					vm.settingsForm.enbTimeout = 70;
				}
			},
			//enb 0-关， 1- 开, 校验 enb 心跳周期
			'settingsForm.cpeInformPeriodAdjustEnable':function(newVal,oldVal){
				var vm = this, value = vm.settingsForm.cpeInformPeriod, regNum = /^\d+$/;
				
				if(newVal == '1'){
					if (value !== '') {
						if (!regNum.test(value) || (value < 60 || value > 3600)) {
							vm.errorStr.cpePeriod = '<%=rb.getString("FanWei")%>:60 ~ 3600';
							$('.cpePeriod input').addClass('errorBorder');
							vm.showError.cpePeriod = true;
							vm.verification[20] = false;
						} else {
							vm.showError.cpePeriod = false;
							$('.cpePeriod input').removeClass('errorBorder');
							vm.verification[20] = true;
						}
					} else {
						vm.errorStr.cpePeriod = '<%=rb.getString("QingShuRu")%> CPE <%=rb.getString("InformZhouQi")%>;'
						$('.cpePeriod input').addClass('errorBorder');
						vm.showError.cpePeriod = true;
						vm.verification[20] = false;
					}
				}else{
					vm.showError.cpePeriod = false;
					$('.cpePeriod input').removeClass('errorBorder');
					vm.verification[20] = true;
					
					//此时心跳周期为空，并且开关为关，超时时间为： 默认的心跳周期初始值+10
					if(vm.settingsForm.cpeInformPeriod < 60){
						vm.settingsForm.cpeTimeout = 70;
					}
				}
			},
			'settingsForm.cpeInformPeriod':function(newVal,oldVal){
				var vm = this;
				if(newVal){
					if(newVal < 60){
						vm.settingsForm.cpeTimeout = 70;
					}else{
						vm.settingsForm.cpeTimeout = Number(newVal)+10;
					}
					vm.showError.cpeOnlineTimeout = false;
					$('.cpeOnlineTimeout input').removeClass('errorBorder');
					vm.verification[21] = true;
				}else{
					//此时心跳周期为空，并且开关为关，超时时间为： 默认的心跳周期初始值+10
					vm.settingsForm.cpeTimeout = 70;
				}
			},	
			//用户名密码输入错误次数
			'settingsForm.attemptTimes':function(newVal,oldVal){
				var vm = this;

				if(newVal){
					if(Number(newVal) >= Number(vm.settingsForm.sumTimes)){
                        vm.errorStr.verifyCodeErrorNum = '<%=rb.getString("ZongChangShiCiShuFanWei")%>:1~ 90; <%=rb.getString("ZongChangShiCiShuFanWeiTiShi")%>';
                        $('.validVerifyCode input').addClass('errorBorder');
                        vm.showError.verifyCodeErrorNum = true;
                        vm.verification[22] = false
                    }else{
                        vm.showError.verifyCodeErrorNum = false
                        $('.validVerifyCode input').removeClass('errorBorder')
                        vm.verification[22] = true
                    }
				}else{
					vm.showError.verifyCodeErrorNum = false
					$('.validVerifyCode input').removeClass('errorBorder')
					vm.verification[22] = true
				}
				
			}
		},
        created() {
			var vm = this;
			vm.isSAS = writableMap["CODE_ADVANCE_SAS"];
            if(is_super_user === 'false'){
				vm.isAdmin = false
			}else{
				vm.isAdmin = true
			}
		},
		mounted() {
			var vm = this;
			this.queryTime()
		},
		methods: {
			tabClick(tab){
				if (this.activeName == 'UICustom'){
					loadHTML(document.querySelector('#uiCutomerCont'), {
						url: '${ctx}/ui/customization/toCustomizationPage.action'
					});
				}
			},
			/*邮箱测试*/
			async ceshiEmail(){
				var vm = this;
				var obj={
					mailUsername:vm.settingsForm.mailUsername,
					mailHost:vm.settingsForm.mailHost,
					mailPort:vm.settingsForm.mailPort,
					mailPassword:vm.settingsForm.mailPassword,
					receiveMail:vm.receiveMail
				}
				var url = '${ctx}/system/syssettings/verifySMTP.action';
				if(!vm.isreceiveMailShow && vm.receiveMail !== ''){
					var saveParams = JSON.stringify(obj);
					await axios.post(url,saveParams,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function (res) {
						if (res.data) {
							// vm.verificationMail.success = true,
							// vm.verificationMail.error = false,
							vm.verificationMail.addEmail  =true
							vm.$message({
								type: 'success',
								message: '<%=rb.getString("YanZhengTongGuo")%>'
							});
							vm.showWindowInfo = false
						}else{
							// vm.verificationMail.error = true,
							// vm.verificationMail.success = false,
							vm.verificationMail.addEmail = false
							vm.$message({
								type: 'error',
								message: '<%=rb.getString("YouXiangBuShiYouXiaoDe")%>'
							});
							return
						}
					})
				}else{
					vm.isreceiveMailShow =true
				}
				
			},
			// 关闭邮箱弹窗
			closeSettingInfo(){
				this.showWindowInfo = false
			},
			/*
			*input 监听
			* data 为传入参数
			* 1:安全设置--默认密码
			* 2:安全设置--密码有效期
			* 3:安全设置--提示到期时间
			* 4:安全设置--失败次数
			* 5:安全设置--最大登录次数锁屏时间
			* 6:安全设置--锁屏时间
			* 7:设备设置--无响应时间，enb-超时时间
			* 8:设备设置--信号设置0
			* 9:设备设置--信号设置1
			* 10:通知设置--邮箱
			* 11:通知设置--密码
			* 12:通知设置--SMTP
			* 13:通知设置--端口
			* 14:日志设置--路径
			* 15:日志设置--ip
			* 16:日志设置--端口
			* 17:日志设置--用户
			* 18:日志设置--密码
			* 19:安全设置--登录
			* 20:设备设置--enb-Inform 周期
			* 21:设备设置--cpe-Inform 周期
			* 22:设备设置--cpe-超时时间
			
			* 23:安全设置--用户锁定-尝试次数
			* 24:安全设置--IP 限流- X 分钟内
			* 25:安全设置--IP 限流-密码连续错误 N 次
			* 26:安全设置--IP 限流 - IP Y 分钟后自动释放
			*/
			inputChange(data) {
				var vm = this,
					regNum = /^\d+$/; // 验证是否是数字
				regFuNum = /^\-\d+\.?\d*$/; // 只能输入负数
				switch (data) {
					case '1':
						var value = vm.settingsForm.defaultPasswd
						re = /^(?![A-Za-z0-9]+$)(?![a-z0-9_!@#$%^&*?]+$)(?![A-Za-z_!@#$%^&*?]+$)(?![A-Z0-9_!@#$%^&*?]+$)[a-zA-Z0-9_!@#$%^&*?]{6,20}$/; // /(?!^(\d+|[a-zA-Z]+|[~!@#$%^&*?]+)$)^[\w~!@#$%\^&*?]{6,20}$/
						if (vm.settingsForm.passwordContent) {
							if (value.length < 6 || value.length > 20) {
								vm.errorStr.morenmima = '<%=rb.getString("MiMaChangDuYingGaiWei")%>6-20<%=rb.getString("Ge")%><%=rb.getString("ZiFu")%>';
								$('.morenmima input').addClass('errorBorder')
								vm.showError.morenmima = true
								vm.verification[0] = false
							} else if (value !== '' && re.test(value)) {
								vm.showError.morenmima = false
								$('.morenmima input').removeClass('errorBorder')
								vm.verification[0] = true
							} else {
								vm.errorStr.morenmima = '<%=rb.getString("MiMaBiXuLiangZhongLeiXing")%>';
								$('.morenmima input').addClass('errorBorder')
								vm.showError.morenmima = true
								vm.verification[0] = false
							}
						} else {
							if (value.length < 6 || value.length > 20) {
								vm.errorStr.morenmima = '<%=rb.getString("MiMaChangDuYingGaiWei")%>6-20<%=rb.getString("Ge")%><%=rb.getString("ZiFu")%>';
								$('.morenmima input').addClass('errorBorder')
								vm.showError.morenmima = true
								vm.verification[0] = false
							} else if (value !== '') {
								vm.showError.morenmima = false
								$('.morenmima input').removeClass('errorBorder')
								vm.verification[0] = true
							} else {
								vm.errorStr.morenmima = '<%=rb.getString("QingShuRuMiMa")%>';
								$('.morenmima input').addClass('errorBorder')
								vm.showError.morenmima = true
								vm.verification[0] = false
							}
						}


						break;
					case '2':
						var value = vm.settingsForm.validPeriod
						if (value !== '') {
							if (!regNum.test(value) || (value < 1 || value > 90)) {
								vm.errorStr.validPeriod = '<%=rb.getString("MiMaYouXiaoTianShu")%>:1~ 90';
								$('.validPeriod input').addClass('errorBorder');
								vm.showError.validPeriod = true;
								vm.verification[1] = false
							} else {
								vm.showError.validPeriod = false
								$('.validPeriod input').removeClass('errorBorder')
								vm.verification[1] = true
							}
						} else {
							vm.errorStr.validPeriod = '<%=rb.getString("QingShuRu")%><%=rb.getString("MiMaYouXiaoTianShu")%>';
							$('.validPeriod input').addClass('errorBorder')
							vm.showError.validPeriod = true
							vm.verification[1] = false
						}

						break;
					case '3':
						var value = vm.settingsForm.promptBeforeDays
						if (value !== '') {
							if (!regNum.test(value) || (value <= 0 || value > 10)) {
								vm.errorStr.promptBeforeDays = '<%=rb.getString("TiQianTiShiTianShu")%>:1 ~ 10';
								$('.promptBeforeDays input').addClass('errorBorder');
								vm.showError.promptBeforeDays = true;
								vm.verification[2] = false
							} else {
								vm.showError.promptBeforeDays = false
								$('.promptBeforeDays input').removeClass('errorBorder')
								vm.verification[2] = true
							}
						} else {
							vm.errorStr.promptBeforeDays = '<%=rb.getString("QingShuRu")%><%=rb.getString("TiQianTiShiTianShu")%>';
							$('.promptBeforeDays input').addClass('errorBorder')
							vm.showError.promptBeforeDays = true
							vm.verification[2] = false
						}

						break;
					case '4':
						var value = vm.settingsForm.attemptTimes
						if (value !== '') {
						    //限制范围为： 0 ~ 10
							if (!regNum.test(value) || (value < 0 || value > 10)) {
								vm.errorStr.shibaicishu = '<%=rb.getString("ChangShiDengLiCiShu")%>:0 ~ 10';
								$('.shibaiInput input').addClass('errorBorder');
								vm.showError.shibaicishu = true;
								vm.verification[3] = false
							} else {
								vm.showError.shibaicishu = false
								$('.shibaiInput input').removeClass('errorBorder')
								vm.verification[3] = true
							}
						} else {
							vm.errorStr.shibaicishu = '<%=rb.getString("QingShuRu")%><%=rb.getString("ChangShiDengLiCiShu")%>';
							$('.shibaiInput input').addClass('errorBorder')
							vm.showError.shibaicishu = true
							vm.verification[3] = false
						}

						break;
					case '5':
						var value = vm.settingsForm.unlockMinu
						if (value !== '') {
							if (!regNum.test(value) || (value < 1 || value > 60)) {
								vm.errorStr.suodingshijian = '<%=rb.getString("ZhangHuZiDongJieSuoShiJian")%>:1 ~ 60';
								$('.suodingInput input').addClass('errorBorder');
								vm.showError.suodingshijian = true;
								vm.verification[4] = false
							} else {
								vm.showError.suodingshijian = false
								$('.suodingInput input').removeClass('errorBorder')
								vm.verification[4] = true
							}
						} else {
							vm.errorStr.suodingshijian = '<%=rb.getString("QingShuRu")%><%=rb.getString("ZhangHuZiDongJieSuoShiJian")%>';
							$('.suodingInput input').addClass('errorBorder')
							vm.showError.suodingshijian = true
							vm.verification[4] = false
						}
						break;
					case '6':
						var value = vm.settingsForm.userSessionExpirationMin
						if (value !== '') {
							if (!regNum.test(value) || (value < 0 || value > 30)) {
								vm.errorStr.suoping = '<%=rb.getString("HuiHuaYouXiaoQiCuoWu")%>';
								$('.suoping input').addClass('errorBorder');
								vm.showError.suoping = true;
								vm.verification[5] = false
							} else {
								vm.showError.suoping = false
								$('.suoping input').removeClass('errorBorder')
								vm.verification[5] = true
							}
						} else {
							vm.errorStr.suoping = '<%=rb.getString("QingShuRuHuiHuaYouXiaoQi")%>';
							$('.suoping input').addClass('errorBorder')
							vm.showError.suoping = true
							vm.verification[5] = false
						}
						break;
					case '7':	
						var value = Number(vm.settingsForm.enbTimeout);
						if((vm.settingsForm.enbInformPeriod == '' || vm.settingsForm.enbInformPeriod == null || vm.settingsForm.enbInformPeriod == undefined ) && vm.settingsForm.enbInformPeriodAdjustEnable == '0'){
							var minNumber = 60+10;
						}else{
							var minNumber = Number(vm.settingsForm.enbInformPeriod)+10;
						}
						//周期+10
						if (value !== '') {
							if (!regNum.test(value) || (value < minNumber || value > 3610)) {
								vm.errorStr.wuxiangying = '<%=rb.getString("FanWei")%>:'+ minNumber +'~ 3610';
								$('.wuxiangying input').addClass('errorBorder');
								vm.showError.wuxiangying = true;
								vm.verification[6] = false;
							} else {
								vm.showError.wuxiangying = false;
								$('.wuxiangying input').removeClass('errorBorder');
								vm.verification[6] = true;
							}
						} else {
							vm.errorStr.wuxiangying = '<%=rb.getString("QingShuRuJiZhanChaoShiShiJian")%>';
							$('.wuxiangying input').addClass('errorBorder');
							vm.showError.wuxiangying = true;
							vm.verification[6] = false;
						}
						break;
					case '8':
						var value = vm.settingsForm.rsrpVal0,
						    value1 = vm.settingsForm.rsrpVal1;
						
						if (value !== '') {
							if (!regFuNum.test(value) || (value < -150 || value > -80)) {
								vm.errorStr.rsrpVal0 = '<%=rb.getString("FanWei")%>:-150 ~ -80';
								$('.rsrpVal0 input').addClass('errorBorder');
								vm.showError.rsrpVal0 = true;
								vm.verification[7] = false
							} else {
								if(regFuNum.test(value1) && value - value1 >= 0) {
									vm.errorStr.rsrpVal0 = '<%=rb.getString("RuoZhiXiaoYuQiangZhi")%>'
									$('.rsrpVal0 input').addClass('errorBorder')
									vm.showError.rsrpVal0 = true
									vm.verification[7] = false
								}else {
									vm.showError.rsrpVal0 = false
									$('.rsrpVal0 input').removeClass('errorBorder')
									vm.verification[7] = true
								}
							}
						} else {
							vm.errorStr.rsrpVal0 = '<%=rb.getString("RuoYUZhengChangXinHaoZhi")%>'
							$('.rsrpVal0 input').addClass('errorBorder')
							vm.showError.rsrpVal0 = true
							vm.verification[7] = false
						}
						break;
					case '9':
						var value = vm.settingsForm.rsrpVal1
						if (value !== '') {
							if (!regFuNum.test(value) || (value < -150 || value > -80)) {
								vm.errorStr.rsrpVal1 = '<%=rb.getString("FanWei")%>:-150 ~ -80';
								$('.rsrpVal1 input').addClass('errorBorder');
								vm.showError.rsrpVal1 = true;
								vm.verification[8] = false
							} else {
								vm.showError.rsrpVal1 = false
								$('.rsrpVal1 input').removeClass('errorBorder')
								vm.verification[8] = true
								vm.inputChange('8');
							}
						} else {
							vm.errorStr.rsrpVal1 = '<%=rb.getString("ZhengChangYuQiangXinHaoZhi")%>'
							$('.rsrpVal1 input').addClass('errorBorder')
							vm.showError.rsrpVal1 = true
							vm.verification[8] = false
						}
						break;
					case '10':
						var value = vm.settingsForm.mailUsername;
							reg = /^(([a-zA-Z0-9_\.-]+)@([\da-z\.-]+)\.([a-z]{2,6}))$/;
						if (value !== '' && value != null) {
							<%-- if (!reg.test(value)) {
								vm.errorStr.mailUsername = '<%=rb.getString("YouXiangGeShiCuoWu")%>';
								$('.mailUsername input').addClass('errorBorder');
								vm.showError.mailUsername = true;
								vm.verification[9] = false
							} else { --%>
								vm.showError.mailUsername = false
								$('.mailUsername input').removeClass('errorBorder')
								vm.verification[9] = true
							/* } */
						} else {
							vm.errorStr.mailUsername = '<%=rb.getString("QingShuRuShouJianRen")%>'
							$('.mailUsername input').addClass('errorBorder')
							vm.showError.mailUsername = true
							vm.verification[9] = false
						}
						break;
					case '11':
						var value = vm.settingsForm.mailPassword

						if (value !== '') {
							vm.showError.mailPassword = false
							$('.mailPassword input').removeClass('errorBorder')
							vm.verification[10] = true
						}
						break;
					case '12':
						var value = vm.settingsForm.mailHost,
							reg = /[\u4E00-\u9FA5]/g
						if (value !== '') {
							if (reg.test(value)) {
								vm.errorStr.mailHost = '<%=rb.getString("SMTPGeShiCuoWu")%>';
								$('.mailHost input').addClass('errorBorder');
								vm.showError.mailHost = true;
								vm.verification[11] = false
							} else {
								vm.showError.mailHost = false
								$('.mailHost input').removeClass('errorBorder')
								vm.verification[11] = true
							}
						} else {
							vm.errorStr.mailHost = '<%=rb.getString("QingShuRuSMTPHost")%>'
							$('.mailHost input').addClass('errorBorder')
							vm.showError.mailHost = true
							vm.verification[11] = false
						}
						break;
					case '13':
						var value = vm.settingsForm.mailPort;
						if (value !== '') {
							if (!regNum.test(value) || (value < 0 || value > 65535)) {
								vm.errorStr.mailPort = '<%=rb.getString("ChangDuChaoChuFanWei")%>';
								$('.mailPort input').addClass('errorBorder');
								vm.showError.mailPort = true;
								vm.verification[12] = false
							} else {
								vm.showError.mailPort = false
								$('.mailPort input').removeClass('errorBorder')
								vm.verification[12] = true
							}
						} else {
							vm.errorStr.mailPort = '<%=rb.getString("QingShuRuDuanKou")%>'
							$('.mailPort input').addClass('errorBorder')
							vm.showError.mailPort = true
							vm.verification[12] = false
						}
						break;
					case '14':
						var value = vm.settingsForm.logFtpSavePath
						if (value !== '') {
							vm.showError.logFtpSavePath = false
							$('.logFtpSavePath input').removeClass('errorBorder')
							vm.verification[13] = true
						} else {
							vm.errorStr.logFtpSavePath = '<%=rb.getString("QingShuRuShangChuanLuJing")%>';
							$('.logFtpSavePath input').addClass('errorBorder')
							vm.showError.logFtpSavePath = true
							vm.verification[13] = false
						}

						break;
					case '15':
						var value = vm.settingsForm.logFtpIpAddr,
							reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
						if (value !== '') {
							if (!reg.test(value)) {
								vm.errorStr.logFtpIpAddr = '<%=rb.getString("XinIPGeShiCuoWu")%>';
								$('.logFtpIpAddr input').addClass('errorBorder');
								vm.showError.logFtpIpAddr = true;
								vm.verification[14] = false
							} else {
								vm.showError.logFtpIpAddr = false
								$('.logFtpIpAddr input').removeClass('errorBorder')
								vm.verification[14] = true
							}
						} else {
							vm.errorStr.logFtpIpAddr = '<%=rb.getString("XinIPShuRuTiShi")%>';
							$('.logFtpIpAddr input').addClass('errorBorder')
							vm.showError.logFtpIpAddr = true
							vm.verification[14] = false
						}
						break;
					case '16':
						var value = vm.settingsForm.logFtpPort;
						if (value !== '') {
							if (!regNum.test(value) || (value < 0 || value > 65535)) {
								vm.errorStr.logFtpPort = '<%=rb.getString("ChangDuChaoChuFanWei")%>';
								$('.logFtpPort input').addClass('errorBorder');
								vm.showError.logFtpPort = true;
								vm.verification[15] = false
							} else {
								vm.showError.logFtpPort = false
								$('.logFtpPort input').removeClass('errorBorder')
								vm.verification[15] = true
							}
						} else {
							vm.errorStr.logFtpPort = '<%=rb.getString("QingShuRuDuanKou")%>'
							$('.logFtpPort input').addClass('errorBorder')
							vm.showError.logFtpPort = true
							vm.verification[15] = false
						}
						break;
					case '17':
						var value = vm.settingsForm.logFtpUser;
						if (value !== '') {
							vm.showError.logFtpUser = false
							$('.logFtpUser input').removeClass('errorBorder')
							vm.verification[16] = true
						} else {
							vm.errorStr.logFtpUser = '<%=rb.getString("QingShuRuYongHuMing")%>'
							$('.logFtpUser input').addClass('errorBorder')
							vm.showError.logFtpUser = true
							vm.verification[16] = false
						}
						break;
					case '18':
						var value = vm.settingsForm.logFtpPassword
						if (value !== '') {
							vm.showError.logFtpPassword = false
							$('.logFtpPassword input').removeClass('errorBorder')
							vm.verification[17] = true
						} else {
							vm.errorStr.logFtpPassword = '<%=rb.getString("QingShuRuMiMa")%>';
							$('.logFtpPassword input').addClass('errorBorder')
							vm.showError.logFtpPassword = true
							vm.verification[17] = false
						}
						break;
					case '19':
						var value = vm.settingsForm.msg
						if (value !== '') {
							vm.showError.msg = false
							$('.msg textarea').removeClass('errorBorder')
							vm.verification[18] = true
						} else {
							vm.errorStr.msg = '<%=rb.getString("QingShuRu")%><%=rb.getString("DengLuHouXiaoXi")%>';
							$('.msg textarea').addClass('errorBorder')
							vm.showError.msg = true
							vm.verification[18] = false
						}
						break;
					case '20':
						//inform period - enb
						var value = vm.settingsForm.enbInformPeriod;
						if (value !== '') {
							if (!regNum.test(value) || (value < 60 || value > 3600)) {
								vm.errorStr.enbPeriod = '<%=rb.getString("FanWei")%>:60 ~ 3600';
								$('.enbPeriod input').addClass('errorBorder');
								vm.showError.enbPeriod = true;
								vm.verification[19] = false;
							} else {
								vm.showError.enbPeriod = false;
								$('.enbPeriod input').removeClass('errorBorder')
								vm.verification[19] = true;
							}
						} else {
							vm.errorStr.enbPeriod = '<%=rb.getString("QingShuRu")%> eNB <%=rb.getString("InformZhouQi")%>;'
							$('.enbPeriod input').addClass('errorBorder')
							vm.showError.enbPeriod = true;
							vm.verification[19] = false;
						}
						break;
					case '21':
						//inform period - cpe
						var value = vm.settingsForm.cpeInformPeriod;
						if (value !== '') {
							if (!regNum.test(value) || (value < 60 || value > 3600)) {
								vm.errorStr.cpePeriod = '<%=rb.getString("FanWei")%>:60 ~ 3600';
								$('.cpePeriod input').addClass('errorBorder');
								vm.showError.cpePeriod = true;
								vm.verification[20] = false;
							} else {
								vm.showError.cpePeriod = false;
								$('.cpePeriod input').removeClass('errorBorder')
								vm.verification[20] = true;
							}
						} else {
							vm.errorStr.cpePeriod = '<%=rb.getString("QingShuRu")%> CPE <%=rb.getString("InformZhouQi")%>;'
							$('.cpePeriod input').addClass('errorBorder')
							vm.showError.cpePeriod = true;
							vm.verification[20] = false;
						}
						break;
					case '22':
						//device online - cpe
						var value = Number(vm.settingsForm.cpeTimeout);
						//var minNumber = Number(vm.settingsForm.cpeInformPeriod)+10;
						if((vm.settingsForm.cpeInformPeriod == '' || vm.settingsForm.cpeInformPeriod == null || vm.settingsForm.cpeInformPeriod == undefined) && vm.settingsForm.cpeInformPeriodAdjustEnable == '0'){
							var minNumber = 60+10;
						}else{
							var minNumber = Number(vm.settingsForm.cpeInformPeriod)+10;
						}
						if (value !== '') {
							if (!regNum.test(value) || (value < minNumber || value > 3610)) {
								vm.errorStr.cpeOnlineTimeout = '<%=rb.getString("FanWei")%>:'+minNumber+' ~ 3610';
								$('.cpeOnlineTimeout input').addClass('errorBorder');
								vm.showError.cpeOnlineTimeout = true;
								vm.verification[21] = false
							} else {
								vm.showError.cpeOnlineTimeout = false
								$('.cpeOnlineTimeout input').removeClass('errorBorder')
								vm.verification[21] = true
							}
						} else {
							vm.errorStr.cpeOnlineTimeout = '<%=rb.getString("QingShuRuJiZhanChaoShiShiJian")%>'
							$('.cpeOnlineTimeout input').addClass('errorBorder')
							vm.showError.cpeOnlineTimeout = true
							vm.verification[21] = false
						}
						break;
						
					case '23':
						//总尝试次数 要大于 用户名或密码输入的次数
						var value = vm.settingsForm.sumTimes
						if (value !== '') {
							if ((!regNum.test(value) || (value < 1 || value > 90)) || Number(vm.settingsForm.sumTimes) <= Number(vm.settingsForm.attemptTimes)) {
								vm.errorStr.verifyCodeErrorNum = '<%=rb.getString("ZongChangShiCiShuFanWei")%>: 1~ 90; <%=rb.getString("ZongChangShiCiShuFanWeiTiShi")%>';
								$('.validVerifyCode input').addClass('errorBorder');
								vm.showError.verifyCodeErrorNum = true;
								vm.verification[22] = false
							} else {
								vm.showError.verifyCodeErrorNum = false
								$('.validVerifyCode input').removeClass('errorBorder')
								vm.verification[22] = true
							}
						} else {
							vm.errorStr.verifyCodeErrorNum = '<%=rb.getString("ZongChangShiCiShuFanWeiWeiTian")%>;';
							$('.validVerifyCode input').addClass('errorBorder')
							vm.showError.verifyCodeErrorNum = true
							vm.verification[22] = false
						}
						break;
					case '24':
						var value = vm.settingsForm.limitMinus
						if (value !== '') {
							if (!regNum.test(value) || (value < 1 || value > 90)) {
								vm.errorStr.limitMinusErrorNum = '<%=rb.getString("FenZhongXianZhi")%>: 1~ 90;';
								$('.validLimitMins input').addClass('errorBorder');
								vm.showError.limitMinusErrorNum = true;
								vm.verification[23] = false
							} else {
								vm.showError.limitMinusErrorNum = false
								$('.validLimitMins input').removeClass('errorBorder')
								vm.verification[23] = true
							}
						} else {
							vm.errorStr.limitMinusErrorNum = '<%=rb.getString("FenZhongXianZhiTiShi")%>;';
							$('.validLimitMins input').addClass('errorBorder')
							vm.showError.limitMinusErrorNum = true
							vm.verification[23] = false
						}

						break;
					case '25':
						var value = vm.settingsForm.limitCount
						if (value !== '') {
							if (!regNum.test(value) || (value < 1 || value > 90)) {
								vm.errorStr.limitCountErrorNum = '<%=rb.getString("LianXuCuoWuCiShuXianZhi")%>: 1~ 999;';
								$('.validLimitCount input').addClass('errorBorder');
								vm.showError.limitCountErrorNum = true;
								vm.verification[24] = false
							} else {
								vm.showError.limitCountErrorNum = false
								$('.validLimitCount input').removeClass('errorBorder')
								vm.verification[24] = true
							}
						} else {
							vm.errorStr.limitCountErrorNum = '<%=rb.getString("LianXuCuoWuCiShuXianZhiTiShi")%>';
							$('.validLimitCount input').addClass('errorBorder')
							vm.showError.limitCountErrorNum = true
							vm.verification[24] = false
						}

						break;
					case '26':
						var value = vm.settingsForm.limitTimes
						if (value !== '') {
							if (!regNum.test(value) || (value < 60 || value > 9999999)) {
								vm.errorStr.limitTimesErrorNum = '<%=rb.getString("IPZiDongJieSuoShiJianXianZhi")%>: 60~ 9999999;';
								$('.validLimitTimes input').addClass('errorBorder');
								vm.showError.limitTimesErrorNum = true;
								vm.verification[25] = false
							} else {
								vm.showError.limitTimesErrorNum = false
								$('.validLimitTimes input').removeClass('errorBorder')
								vm.verification[25] = true
							}
						} else {
							vm.errorStr.limitTimesErrorNum = '<%=rb.getString("IPZiDongJieSuoShiJianXianZhiTiShi")%>';
							$('.validLimitTimes input').addClass('errorBorder')
							vm.showError.limitTimesErrorNum = true
							vm.verification[25] = false
						}

						break;

					case '27': // ue signal
						var value = vm.settingsForm.uersrpVal0,
						    value1 = vm.settingsForm.uersrpVal1;

						if (value !== '') {
							if (!regFuNum.test(value) || (value < -150 || value > -80)) {
								vm.errorStr.uersrpVal0 = '<%=rb.getString("FanWei")%>:-150 ~ -80';
								$('.uersrpVal0 input').addClass('errorBorder');
								vm.showError.uersrpVal0 = true;
								vm.verification[26] = false
							} else {
								if(regFuNum.test(value1) && value - value1 >= 0) {
									vm.errorStr.uersrpVal0 = '<%=rb.getString("RuoZhiXiaoYuQiangZhi")%>'
									$('.uersrpVal0 input').addClass('errorBorder')
									vm.showError.uersrpVal0 = true
									vm.verification[26] = false
								}else {
									vm.showError.uersrpVal0 = false
									$('.uersrpVal0 input').removeClass('errorBorder')
									vm.verification[26] = true
								}
							}
						} else {
							vm.errorStr.uersrpVal0 = '<%=rb.getString("RuoYUZhengChangXinHaoZhi")%>'
							$('.uersrpVal0 input').addClass('errorBorder')
							vm.showError.uersrpVal0 = true
							vm.verification[26] = false
						}
						break;

					case '28': // ue signal
						var value = vm.settingsForm.uersrpVal1
						if (value !== '') {
							if (!regFuNum.test(value) || (value < -150 || value > -80)) {
								vm.errorStr.uersrpVal1 = '<%=rb.getString("FanWei")%>:-150 ~ -80';
								$('.uersrpVal1 input').addClass('errorBorder');
								vm.showError.uersrpVal1 = true;
								vm.verification[27] = false
							} else {
								vm.showError.uersrpVal1 = false
								$('.uersrpVal1 input').removeClass('errorBorder')
								vm.verification[27] = true
								vm.inputChange('27')
							}
						} else {
							vm.errorStr.uersrpVal1 = '<%=rb.getString("ZhengChangYuQiangXinHaoZhi")%>'
							$('.uersrpVal1 input').addClass('errorBorder')
							vm.showError.uersrpVal1 = true
							vm.verification[27] = false
						}
						break;
				}
			},
			/*
			*check监听
			*data 为传入的参数 
			* 1:安全设置--首次登录修改密码
			* 2:安全设置--密码包含不同类型 
			* 3:安全设置--用户每隔多长时间修改
			* 4:安全设置--登录提示
			* 5:设备设置--检查相同名称LMT
			* 6:设备设置--通知手动同步
			* 7:设备设置--只允许修改符合规则的设备
			* 8:通知设置--通知邮件服务器
			* 9:日志设置--是否转发到远程地址
			* 12:安全设置--用户锁定，验证码开关
			* 其他预留
			*/
			checkChange(data) {
				var vm = this;
				switch (data) {
					case '1':
						if (!vm.settingsForm.modifyPWD) {
							vm.showError.morenmima = false
							$('.morenmima input').removeClass('errorBorder')
						} else {
							vm.inputChange('1')
						}

						break;
					case '2':
						vm.inputChange('1')
						break;
					case '3':
						if (!vm.settingsForm.expires) {
							vm.showError.validPeriod = false
							$('.validPeriod input').removeClass('errorBorder')
							vm.showError.promptBeforeDays = false
							$('.promptBeforeDays input').removeClass('errorBorder')
						} else {
							vm.inputChange('2')
							vm.inputChange('3')
						}
						break;
					case '4':
						if (!vm.settingsForm.enabledFlag) {
							vm.showError.msg = false
							$('.msg textarea').removeClass('errorBorder')
						}
						break;
					case '8':
						if (!vm.settingsForm.emEnabel) {
							vm.ceshidis = true
							vm.verificationMail.success = false
							vm.verificationMail.error = false
							vm.showError.mailUsername = false
							// vm.verificationMail.addEmail = true
							$('.mailUsername input').removeClass('errorBorder')

							vm.showError.mailPassword = false
							$('.mailPassword input').removeClass('errorBorder')

							vm.showError.mailHost = false
							$('.mailHost input').removeClass('errorBorder')

							vm.showError.mailPort = false
							$('.mailPort input').removeClass('errorBorder')
						} else {
							// vm.verificationMail.addEmail = false
							vm.ceshidis = false
							vm.inputChange('10')
							vm.inputChange('11')
							vm.inputChange('12')
							vm.inputChange('13')
						}
						break;
					case '9':
						if (!vm.settingsForm.logFtpEnable) {
							vm.showError.logFtpSavePath = false
							$('.logFtpSavePath input').removeClass('errorBorder')

							vm.showError.logFtpIpAddr = false
							$('.logFtpIpAddr input').removeClass('errorBorder')

							vm.showError.logFtpPort = false
							$('.logFtpPort input').removeClass('errorBorder')

							vm.showError.logFtpUser = false
							$('.logFtpUser input').removeClass('errorBorder')

							vm.showError.logFtpPassword = false
							$('.logFtpPassword input').removeClass('errorBorder')

						} else {
							vm.inputChange('14')
							vm.inputChange('15')
							vm.inputChange('16')
							vm.inputChange('17')
							vm.inputChange('18')
						}
						break;
					/*case '12':
						//用户锁定，验证码开关
						if (!vm.settingsForm.verifyEnable) {
							vm.showError.verifyCodeErrorNum = false
							$('.validVerifyCode input').removeClass('errorBorder')							
						} else {
							vm.inputChange('23')
						}
						break;	*/
				}
			},
			//测试邮箱验证
			emailTest(){
				var vm = this;
				var value = vm.receiveMail,
				reg = /^(([a-zA-Z0-9_\.-]+)@([\da-z\.-]+)\.([a-z]{2,6}))$/;
						if (value !== '') {
							if (!reg.test(value)) {
								vm.isreceiveMailShow = true
								vm.errorStr.receiveMail = '<%=rb.getString("YouXiangGeShiCuoWu")%>';
							}else{
								vm.isreceiveMailShow = false
							}
						}
			},
			// 确定提交
			async addStting() {
				var vm = this, 
					mrVendor = true, ipFlag = true, portFlag = true,
					validateDeviceOfflineSaveDay = true,
					validateLatitudeToleranceRange = true,
					mrOMCName = true;
				//对 自定义验证进行判断
				let formData = vm.settingsForm
				if (!vm.conpetence.types) { // 这里是权限控制 的 提交逻辑
					if (vm.settingsForm.modifyPWD) {
						if (formData.defaultPasswd === '') { // 默认密码
							vm.inputChange('1')
						}
					} else {
						vm.verification[0] = true
					}

					if (vm.settingsForm.expires) {
						if (formData.validPeriod === '') {// 有效期天数
							vm.inputChange('2')
						}
						if (formData.promptBeforeDays === '') { //密码有效期
							vm.inputChange('3')
						}
					} else {
						vm.verification[1] = true
						vm.verification[2] = true
					}
					if (formData.attemptTimes === '') { // 失败锁定时间
						vm.inputChange('4')
					} else {
						vm.verification[3] = true
					}
					if (formData.unlockMinu === '') { // 失败锁定
						vm.inputChange('5')
					} else {
						vm.verification[4] = true
					}
					/*if (formData.verifyEnable === '') { // 用户锁定，失败次数
						if (formData.sumTimes === '') {
							vm.inputChange('23')
						}
					} else {
						vm.verification[22] = true
					}*/
					if (formData.sumTimes === '') { // 用户锁定，失败次数
                        vm.inputChange('23')
                    } else {
                        vm.verification[22] = true
                    }
					// ip限流 限制分钟
					vm.inputChange('24');
					vm.inputChange('25');
					vm.inputChange('26');

					if (formData.userSessionExpirationMin === '') { // 锁屏时间
						vm.inputChange('6')
					} else {
						vm.verification[5] = true
					}
					if (vm.settingsForm.enabledFlag) { // 通知信息
						if (formData.msg === '') {
							vm.inputChange('19')
						}
					} else {
						vm.verification[18] = true
					}
				} else {
					vm.verification[0] = true
					vm.verification[1] = true
					vm.verification[2] = true
					vm.verification[3] = true
					vm.verification[4] = true
					vm.verification[5] = true
					vm.verification[18] = true
					
					vm.verification[22] = true
					vm.verification[23] = true
					vm.verification[24] = true
					vm.verification[25] = true
				}
				/*if (formData.cellTimeout === '') { //设备无响应时间
					vm.inputChange('7')
				} else {
					vm.verification[6] = true
				}*/
				
				if (vm.settingsForm.enbInformPeriodAdjustEnable == '1') {
					if (formData.enbInformPeriod === '') { //enb inform 周期
						vm.inputChange('20')
					}
				} else {
					//vm.verification[19] = true;
					vm.inputChange('20')
				}
				if (formData.enbTimeout === '') { //enb 设备无响应时间
					vm.inputChange('7')
				} else {
					//vm.verification[6] = true
					vm.inputChange('7')
				}
				if (vm.settingsForm.cpeInformPeriodAdjustEnable == '1') {
					if (formData.cpeInformPeriod === '') { //cpe inform 周期
						vm.inputChange('21')
					}
				} else {
					//vm.verification[20] = true;
					vm.inputChange('21')
				}
				if (formData.cpeTimeout === '') { //cpe 设备无响应时间
					vm.inputChange('22')
				} else {
					//vm.verification[21] = true
					vm.inputChange('22')
				}
				
				//增加逻辑
				
				if (formData.rsrpVal0 === '' || vm.showError.rsrpVal0) { // 信号强度设置
					vm.inputChange('8')
				} else {
					vm.verification[7] = true
				}
				if (formData.rsrpVal1 === '' || vm.showError.rsrpVal1) { // 信号强度设置
					vm.inputChange('9')
				} else {
					vm.verification[8] = true
				}

				// ue signal
				if (formData.uersrpVal0 === '' || vm.showError.uersrpVal0) { // 信号强度设置
					vm.inputChange('27')
				} else {
					vm.verification[26] = true
				}
				if (formData.uersrpVal1 === '' || vm.showError.uersrpVal1) { // 信号强度设置
					vm.inputChange('28')
				} else {
					vm.verification[27] = true
				}

				if (vm.settingsForm.emEnabel) {
					if (formData.mailUsername === '') { // 用户设置
						vm.inputChange('10')
					}
					if (formData.mailPassword === '') { //密码
						vm.inputChange('11')
					}
					if (formData.mailHost === '') { // smtp 服务器
						vm.inputChange('12')
					}
					if (formData.mailPort === '') { // 邮箱端口
						vm.inputChange('13')
					}
				} else {
					vm.verification[9] = true
					vm.verification[10] = true
					vm.verification[11] = true
					vm.verification[12] = true
				}

				if (vm.settingsForm.logFtpEnable) { // 日志存储
					if (formData.logFtpSavePath === '') { // 上传路径
						vm.inputChange('14')
					}
					if (formData.logFtpIpAddr === '') { // ip地址
						vm.inputChange('15')
					}
					if (formData.logFtpPort === '') { // 端口
						vm.inputChange('16')
					}
					if (formData.logFtpUser === '') { // 用户名
						vm.inputChange('17')
					}
					if (formData.logFtpUser === '') { // 密码
						vm.inputChange('18')
					}
				} else {
					vm.verification[13] = true
					vm.verification[14] = true
					vm.verification[15] = true
					vm.verification[16] = true
					vm.verification[17] = true
				}

				// 这里执行 自定义验证 应用于 自定义数组的改变
				var isVerification = $.inArray(false, vm.verification)
				// 以下两个参数传给后台 前端不做任何展示
				vm.settingsForm.oldSEMin = vm.oldSEMin
				vm.settingsForm.oldCTime = vm.oldCTime
				// 对运营商名称进行验证
				if(vm.MrFlag){
					vm.$refs.settingsForm.validateField('mrVendor', errorMesage => {
						if (!errorMesage) {
							mrVendor = true
						} else {
							mrVendor = false
						}
					})
					// 对网管名称进行验证
					vm.$refs.settingsForm.validateField('mrOMCName', errorMesage => {
						if (!errorMesage) {
							mrOMCName = true
						} else {
							mrOMCName = false
						}
					})
				}
				vm.$refs.settingsForm.validateField('deviceOfflineSaveDay', errorMesage => {
					if (!errorMesage) {
						validateDeviceOfflineSaveDay = true
					} else {
						validateDeviceOfflineSaveDay = false
					}
				})
				vm.$refs.settingsForm.validateField('latitudeToleranceRange', errorMesage => {
					if (!errorMesage) {
						validateLatitudeToleranceRange = true
					} else {
						validateLatitudeToleranceRange = false
					}
				})
				//邮箱不通过不允许提交
				if(vm.settingsForm.emEnabel && vm.changeEmail()){
					if(vm.verificationMail.addEmail){
						vm.checkEmailFlag = true;
					}else{
						vm.showWindowInfo = true;
						vm.checkEmailFlag = false;
					}
				}
				
				if(vm.settingsForm.rsysLogEnable == '1'){
					if(vm.settingsForm.rsysLogIp == '' || vm.settingsForm.rsysLogIp === null || vm.settingsForm.rsysLogIp === undefined){
						ipFlag = false
					}else{
						ipFlag = true
					}
					if(vm.settingsForm.rsysLogPort === '' || vm.settingsForm.rsysLogPort === null || vm.settingsForm.rsysLogPort === undefined){
						portFlag = false
					}else{
						portFlag = true
					}
				}

				// 校验密码长度合法性
				var validatePasswordLength = true;
				if(vm.isAdmin && !vm.conpetence.types) {
					vm.$refs.settingsForm.validateField(['pwdMinLength','pwdMaxLength'], errorMesage => {
						if (errorMesage) {
							validatePasswordLength = false
						}
					})
				}
				// 自定义的验证通过和两部分验证通过后 执行保存操作
				if (isVerification === -1 && mrVendor && mrOMCName && ipFlag && portFlag && !vm.verificationMail.error && vm.checkEmailFlag && validateDeviceOfflineSaveDay && validateLatitudeToleranceRange && validatePasswordLength) {
					var vm = this;
						url = '${ctx}/system/syssettings/updateSettingsInfo.action',
						isChanged = isFormChanged(vm.$refs.settingsForm),
						saveParams = {
							updateSysSettingParam:{}
						}
						updateParams = {};
					vm.ldapTest('submit');
					if(!isChanged){
						showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
						return;
					}
					vm.$refs.settingsForm.fields.map(function(field){
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
								updateParams[field.prop] = field.fieldValue;
							};
						}
					});
					Object.assign(saveParams,vm.settingsForm);
					
					delete saveParams.northboundIp;
					delete saveParams.northboundPort;
					delete saveParams.northboundServiceStatus;
					delete saveParams.ldapResult;
					//此参数不下发
					delete saveParams.rsysLogTestStatus;
					delete saveParams.rsysLogUIEnable;
					//开关为关，不提交这些参数
					if(vm.rsysLogUIEnableFlag == false){
						delete saveParams.rsysLogEnable;
						delete saveParams.rsysLogIp;
						delete saveParams.rsysLogPort;
					}

					if(saveParams.defaultPasswd){
						saveParams.defaultPasswd = AesEncrypt(saveParams.defaultPasswd);
					}
					
					saveParams.updateSysSettingParam = updateParams;
					saveParams = JSON.stringify(saveParams);
					
					await axios.post(url, saveParams,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function (res) {
						if (res.data.success) {
							vm.$message({
								type: 'success',
								message: '<%=rb.getString("ChengGong")%>'
							});
							// 刷新时间
							setCurrOperator("${SYSSESSIONKEY.operator_id}");
							// 如果修改了语言 就从新刷新页面  
							if (vm.oldLanguage != vm.settingsForm.languageCode) {
								setTimeout(function () {
									window.location.href = "${ctx}/sys/login/reloadAction.action?token=" + omctoken.replaceAll('+','%2B');
								}, 3000)
							}
							rsrp1 = vm.settingsForm.rsrpVal0;
							rsrp2 = vm.settingsForm.rsrpVal1;
							uersrp1 = vm.settingsForm.uersrpVal0;
							uersrp2 = vm.settingsForm.uersrpVal1;
							localStorage.setItem("rsrp1",rsrp1);
							localStorage.setItem("rsrp2",rsrp2);
							localStorage.setItem("uersrp1",uersrp1);
							localStorage.setItem("uersrp2",uersrp2);
							noActionTime = new Date('2100-01-01 00:00:00');
							lastVisitedTime = new Date('2100-01-01 00:00:00');
							vm.init();
						} else {
							vm.$message({
								type: 'error',
								message: res.data.message
							});
						}
					})
				} else {
				}
			},
			// 重置
			close() {
				var vm = this;
				confirmStr = '<%=rb.getString("QueDingChongZhiDangQianYeMian")%>'
				if (isFormChanged(vm.$refs.settingsForm)) {
					vm.$confirm(confirmStr, '<%=rb.getString("QueRen")%>', {
						confirmButtonText: '<%=rb.getString("QueDing")%>',
						cancelButtonText: '<%=rb.getString("QuXiao")%>',
						type: 'warning',
						closeOnClickModal: false
					}).then(() => {
						vm.isReset = true;
						vm.init()
					}).catch(() => {
					})
				}
			},
			// 跳转到规则页面
			toGuizeInfo() {
				eventAllBus.$emit("gomenupage", 60106);
			},
			// 初始化函数
			init() {
				var vm = this;
				url = '${ctx}/system/syssettings/getSettingsInfo.action';
				axios.get(url).then(function (res) {
					vm.defaultEmailName = res.data.mailUsername == null ? '' : res.data.mailUsername;
					vm.defaultEmailPwd = res.data.mailPassword == null ? '' :  res.data.mailPassword;
					vm.defaultEmailHost = res.data.mailHost == null ? '' :  res.data.mailHost;
					vm.defaultEmailPort = res.data.mailPort == null ? '' : res.data.mailPort;
					//系统日志配置是否展示
					if (res.data.rsysLogUIEnable == '0' || res.data.rsysLogUIEnable === '' || res.data.rsysLogUIEnable === null || res.data.rsysLogUIEnable === undefined ){
						vm.rsysLogUIEnableFlag = false;
					}else{
						vm.rsysLogUIEnableFlag = true;
					}
                    // 磁盘告警通知配置
					Object.assign(vm.settingsForm,res.data)
					//映射数据
					var curENBPeriod = Number(vm.settingsForm.enbInformPeriod), curCPEPeriod = Number(vm.settingsForm.cpeInformPeriod);
				
					if (curENBPeriod === '' || curENBPeriod === null || curENBPeriod === undefined || curENBPeriod == 0){
						vm.settingsForm.enbTimeout = '';
					}
					if (curCPEPeriod === '' || curCPEPeriod === null || curCPEPeriod === undefined || curCPEPeriod == 0){
						vm.settingsForm.cpeTimeout = '';
					}							
					if (res.data.signalingTraceSaveDays === '' || res.data.signalingTraceSaveDays === null || res.data.signalingTraceSaveDays === undefined){
						vm.settingsForm.signalingTraceSaveDays = '7';
					}else {
						vm.settingsForm.signalingTraceSaveDays = res.data.signalingTraceSaveDays;
					}
					if(res.data.varDiskAlarmThresHold === '' || res.data.varDiskAlarmThresHold === null || res.data.varDiskAlarmThresHold === undefined){
                        vm.settingsForm.varDiskAlarmThresHold = '10%';
                    }else{
                        vm.settingsForm.varDiskAlarmThresHold = res.data.varDiskAlarmThresHold;
                    }
                    if(res.data.homeDiskAlarmThresHold === '' || res.data.homeDiskAlarmThresHold === null || res.data.homeDiskAlarmThresHold === undefined){
                        vm.settingsForm.homeDiskAlarmThresHold = '10%';
                    }else{
                        vm.settingsForm.homeDiskAlarmThresHold = res.data.homeDiskAlarmThresHold;
                    }
                    if(res.data.usrDiskAlarmThresHold === '' || res.data.usrDiskAlarmThresHold === null || res.data.usrDiskAlarmThresHold === undefined){
                        vm.settingsForm.usrDiskAlarmThresHold = '10%';
                    }else{
                        vm.settingsForm.usrDiskAlarmThresHold = res.data.usrDiskAlarmThresHold;
                    }
                    if(res.data.rootDiskAlarmThresHold === '' || res.data.rootDiskAlarmThresHold === null || res.data.rootDiskAlarmThresHold === undefined){
                        vm.settingsForm.rootDiskAlarmThresHold = '10%';
                    }else{
                        vm.settingsForm.rootDiskAlarmThresHold = res.data.rootDiskAlarmThresHold;
                    }
					if (vm.settingsForm.northboundServiceStatus == '0'){
						vm.northStatusCls = "el-alert__icon el-icon-error redIcon";
						vm.northStatusText = '<%=rb.getString("JinYong")%>';
					}else if (vm.settingsForm.northboundServiceStatus == '1'){
						vm.northStatusCls = "el-alert__icon el-icon-success greenIcon";
						vm.northStatusText = '<%=rb.getString("QiYong")%>';
					}
					
					if (vm.isReset) {
						vm.inputChange('1')
						if (vm.settingsForm.expires) {
							vm.inputChange('2')
							vm.inputChange('3')
						}

						vm.inputChange('4')
						vm.inputChange('5')
						vm.inputChange('6')
						vm.inputChange('7')
						vm.inputChange('8')
						vm.inputChange('9')
						if (vm.settingsForm.emEnabel) {
							vm.inputChange('10')
							vm.inputChange('11')
							vm.inputChange('12')
							vm.inputChange('13')
							
						}else{
							vm.verificationMail.addEmail = true
						}

						if (vm.settingsForm.logFtpEnable) {
							vm.inputChange('14')
							vm.inputChange('15')
							vm.inputChange('16')
							vm.inputChange('17')
							vm.inputChange('18')
						}
						if (vm.settingsForm.enabledFlag) {
							vm.inputChange('19')
						}
						vm.inputChange('22')
						if (vm.settingsForm.enbInformPeriodAdjustEnable == '1') {
							vm.inputChange('20')
						}
						if (vm.settingsForm.cpeInformPeriodAdjustEnable == '1') {
							vm.inputChange('21')
						}
						//用户锁定，总尝试次数
						if (vm.settingsForm.unlockMinu) {
							vm.inputChange('23')
						}
						vm.inputChange('24')
						vm.inputChange('25')
						vm.inputChange('26')
						vm.verificationMail.success = false
						vm.verificationMail.error = false
					}

					
					vm.oldSEMin = res.data.userSessionExpirationMin;
					vm.oldCTime = res.data.cellTimeout;//注意这里的逻辑
					// 权限控制
					vm.conpetence.types = res.data.types;

					vm.oldLanguage = res.data.languageCode;
					
					vm.settingsForm.rebootLogDataSaveDays = res.data.rebootLogDataSaveDays ? res.data.rebootLogDataSaveDays : '7';
					vm.settingsForm.rebootLogSaveCount = res.data.rebootLogSaveCount ? res.data.rebootLogSaveCount : '2';

					if(vm.settingsForm.defaultPasswd) {
						vm.settingsForm.defaultPasswd = AesDecryptSingle(vm.settingsForm.defaultPasswd);
					}
					vm.queryTimeData.map(function(item){
						if(item.id == vm.settingsForm.timezoneCode){
							vm.settingsForm.timeZoneId = item.timeZoneId;
						}
					})
					vm.defaultSettingParams = JSON.parse(JSON.stringify(vm.settingsForm));
					initForm(vm.$refs.settingsForm);
				})
				vm.conpetence.is_super_user = (is_super_user == 'true');
				vm.getNorthUserList();
				
			},
			//时区选择
			queryTime() {
				var vm = this,
					url = '${ctx}/system/syssettings/queryTimezoneList.action';
				axios.post(url).then(function (res) {
					vm.queryTimeData = res.data;
					vm.init();
				})
			},
			// 加载成功后执行此操作
			openDialogSuc() { },
			// 关闭弹窗
			cancelSlide() {
				var vm = this;
				vm.$refs.slider.hide();
				vm.$refs.ctableUpsList.refresh();
			},
			/*新建供应商列表*/
			addSASList() {
				var vm = this;
				vm.slideUrl = '${ctx}/cell/SAS/provider/toEdit.action';
				vm.slideTitle = 'New SAS Provider';
				vm.sliderHeight = '100%';
				vm.headerShow = true;
				vm.$refs.slider.showSlide(function () {
					eventBus.$emit('open-dialog', vm.rowData, 'add');
				});
			},
			clickMenu(ev) { //操作项点击方法 
				var codes = {
					edit: this.editInfo,
					del: this.delInfo,
				}
				if (codes[ev.code]) {
					codes[ev.code](this.$root.rowData)
				}
			},
			handerClose() { //点击页面其他地方菜单收起
				this.$refs.menuActiveFault.hide();
			},
			optClick(row, ev) { // 操作项： 1.修改 2.删除
				var vm = this;

				vm.menus = [
					{ label: '<%=rb.getString("XiuGai")%>', cls: "el-icon el-icon-operation-edit CODE_ADVANCE_SAS hidden", code: 'edit' },
					{ label: '<%=rb.getString("ShanChu")%>', cls: "el-icon  el-icon-operation-delete CODE_ADVANCE_SAS hidden", code: 'del' },
				]
				this.rowData = row
				vm.$nextTick(function () {
					document.body.click();
					vm.$refs.menuActiveFault.show(ev);

				});
			},
			// 修改sas信息
			editInfo() {
				var vm = this;
				vm.slideUrl = '${ctx}/cell/SAS/provider/toEdit.action';
				vm.slideTitle = 'Update SAS Provider';
				vm.sliderHeight = '100%';
				vm.headerShow = true;
				this.$refs.slider.showSlide(function () {
					eventBus.$emit('open-dialog', vm.rowData, 'edit');
				});
			},
			// 删除
			delInfo(code) {
				var vm = this;
				var params = {};
				var message = '<%=rb.getString("QueDingShanChuSheBei")%>';
				var url = "${ctx}/cell/SAS/provider/haveOperator.action"
				params.id = code.id;

				axios.post(url, stringify(params)).then(function (res) {
					var data = res.data;
					message = data.message;
					vm.$confirm(message, '<%=rb.getString("QueRen")%>', {
						customClass: "warningConfirm",
						confirmButtonText: '<%=rb.getString("QueDing")%>',
						cancelButtonText: '<%=rb.getString("QuXiao")%>',
						type: 'warning',
						closeOnClickModal: false
					}).then(() => {
						axios.post('${ctx}/cell/SAS/provider/deleteProvider', stringify(params)).then(function (response) {
							let data = response.data;
							if (data.success) {
								vm.$message({
									message: data.message || '<%=rb.getString("ChengGong")%>',
									type: 'success',
								});
								vm.$refs.ctableUpsList.refresh();
							} else {
								vm.$message.error(data.message)
							}

						}).catch(function (error) { })

					}).catch(() => {

					})
				})

			},
			changeEmail(){
				var vm = this,
					changeFlag,
					mailName = vm.settingsForm.mailUsername == null ? '' : vm.settingsForm.mailUsername,
					pwd = vm.settingsForm.mailPassword == null ? '' : vm.settingsForm.mailPassword,
					host = vm.settingsForm.mailHost == null ? '' : vm.settingsForm.mailHost,
					port = vm.settingsForm.mailPort == null ? '' : vm.settingsForm.mailPort;
				if(mailName != vm.defaultEmailName || pwd != vm.defaultEmailPwd || host != vm.defaultEmailHost || port != vm.defaultEmailPort){
					changeFlag = true;
				}else{
					changeFlag = false;
				}
				return changeFlag;
			},
			// 获取北向用户信息
			getNorthUserList(){
				var vm = this,
					urls = '${ctx}/northboundApi/v1/user/users',
					params={
						timeZone:timeZone,
						operatorCode:operator_code,
					};
				params = JSON.stringify(params);
				axios.post(urls,params,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
					var data = response.data.data;
					vm.northUserTableData = data;
				}) 
			},
			//北向用户菜单操作点击事件
			northUserOptClick(row, ev) { // 操作项： 1.修改 2.删除
				var vm = this;

				vm.northUserMenuList = [
					{ label: '<%=rb.getString("XiuGai")%>', cls: "el-icon el-icon-operation-edit", code: 'edit' },
					{ label: '<%=rb.getString("ShanChu")%>', cls: "el-icon  el-icon-operation-delete", code: 'del' },
				]
				vm.northUserRowData = row
				vm.$nextTick(function () {
					document.body.click();
					vm.$refs.northUserMenu.show(ev);
				});
			},
			// 操作事件
			northUserClickMenu(ev){
				var vm = this,
					codes = {
						edit: this.northUserEdit,
						del: this.northUserDel,
					};
				if (codes[ev.code]) {
					codes[ev.code](vm.northUserRowData)
				}
			},
			// 菜单点击收起
			northUserHanderClose(){
				this.$refs.northUserMenu.hide();
			},
			// 北向用户新增
			addNorthUser(){
				var vm = this;
				vm.northUserDialogTitle = '<%=rb.getString("TianJia")%>';
				vm.northUserOpType = 'add';
				vm.showNorthUserDialog = true;
				vm.$nextTick(()=>{
					vm.$refs.northUserDialogForm.clearValidate();
				})
			},
			// 北向用户  点击修改
			northUserEdit(row){
				var vm = this;
				vm.northUserDialogTitle = '<%=rb.getString("XiuGai")%>';
				vm.northUserOpType = 'edit';
				Object.assign(vm.northUserDialogForm,row)
				vm.showNorthUserDialog = true;
			},
			// 北向用户  点击删除
			northUserDel(row){
				var vm = this,
					urls = '${ctx}/system/sysuser/deleteNorthboundUser.action' || '${ctx}/northboundApi/v1/user/'+row.userId,
					params = {
						id: row.userId
					};
				
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancalButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning'
				}).then(()=>{
					axios.post(urls, stringify(params)).then(function(response){
					//axios.delete(urls).then(function(response){
						let data = response.data;
						if ( data.code == 200 ){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							})
							vm.getNorthUserList();
						}else {
							vm.$message.error(data.message)
						}
						
					}).catch(function(error){})
					
				}).catch(()=>{
					
				})
			},
			// 北向用户 是否启用按钮点击
			northUserEnableClick(row){
				var vm = this;

				if(row.userEnable === '0'){
					vm.northUserEnableSubmit(row,'1')
				}else{
					vm.northUserEnableSubmit(row,'0')
				}
				event.stopPropagation();// 禁止事件穿透
				
			},
			// 北向用户 是否启用提交
			northUserEnableSubmit(row,value){
				var vm = this,
					urls = '${ctx}/system/sysuser/updateNorthboundUser.action' || '${ctx}/northboundApi/v1/user/update',
					params = {
						timeZone: timeZone,
						userId: row.userId,
						userEnable: value,
						operatorCode:operator_code
					};
				params = JSON.stringify(params);
				axios.post(urls,params,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
					var data = response.data;
					if(data.code == 200){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});
						vm.getNorthUserList();
					}else{
						vm.$message({
							message: data.message,
							type:'error',
						});
					}
				})
			},
			// 北向用户 新增 修改提交
			northUserDialogSubmit(){
				var vm = this,
					urls = '${ctx}/system/sysuser/addNorthboundUser.action' || '${ctx}/northboundApi/v1/user',
					params = {
						timeZone:timeZone,
						operatorCode:operator_code,
						userName:vm.northUserDialogForm.userName,
						userPwd:vm.northUserDialogForm.userPwd,
						userEnable:vm.northUserDialogForm.userEnable,
					};
				if(vm.northUserOpType == 'edit'){
					params.userId = vm.northUserRowData.userId;
					urls = '${ctx}/system/sysuser/updateNorthboundUser.action' || '${ctx}/northboundApi/v1/user/update';
				}
				params = JSON.stringify(params);
				vm.$refs["northUserDialogForm"].validate( valid => {
					if(valid){
						if(vm.northUserOpType == 'add'){
							axios.post(urls,params,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
								var data = response.data;
								if(data.code == 200){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type:'success',
									});
									vm.showNorthUserDialog = false;
									vm.getNorthUserList();
								}else{
									if(data.message && data.message == 'USER_EXIST'){
										vm.$message({
											message: '<%=rb.getString("Msg_DengLuYongHuMingYiCunZai")%>',
											type:'error',
										});
									}else{
										vm.$message({
											message: data.message,
											type:'error',
										});
									}
								}
							})
						}else{
							axios.post(urls,params,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
								var data = response.data;
								if(data.code == 200){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type:'success',
									});
									vm.showNorthUserDialog = false;
									vm.getNorthUserList();
								}else{
									if(data.message && data.message == 'USER_EXIST'){
										vm.$message({
											message: '<%=rb.getString("Msg_DengLuYongHuMingYiCunZai")%>',
											type:'error',
										});
									}else{
										vm.$message({
											message: data.message,
											type:'error',
										});
									}
								}
							})
						}
						
					}
				}) 
			},
			// 北向用户 弹窗关闭
			northUserDialogClose(){
				var vm = this,
					params={
						userName:'',
						userPwd:'',
						userEnable:'0',
					};
				Object.assign(vm.northUserDialogForm,params);
				vm.$refs.northUserDialogForm.clearValidate();
			},
			// LDAP 测试
			ldapTest(type){
				var vm = this,
					url = '${ctx}/ldap/test.action',
					params={
						ldapIp:vm.settingsForm.ldapIp,
						ldapPort:vm.settingsForm.ldapPort,
						ldapBase:vm.settingsForm.ldapBase,
						ldapUser:vm.settingsForm.ldapUser,
						ldapPwd:vm.settingsForm.ldapPwd,
						ldapSSL:vm.settingsForm.ldapSSL,
					};
				var saveParams = JSON.stringify(params);
				axios.post(url,saveParams,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function (res) {
					if (res.data) {
						vm.settingsForm.ldapResult = 'true'; 
						if(type == 'test'){
							vm.$message({
								type: 'success',
								message: '<%=rb.getString("YanZhengTongGuo")%>'
							});
						}
					}else{
						vm.settingsForm.ldapResult = 'false';
						if(type == 'test'){
							vm.$message({
								type: 'error',
								message: '<%=rb.getString("LDAPBuShiYouXiaoDe")%>'
							});
						}
					}
				})
			},
			// LDAP 开关改变事件
			ldapEnableChange(val){
				var vm =this;
				['ldapIp','ldapPort','ldapBase','ldapUser','ldapPwd'].map((item)=>{
					vm.$refs.settingsForm.validateField(item);
				})	
			},
			//-------syslog-------------
			syslogTest(type){
				var vm = this, ipPort = '', testIpFlag = true, testPortFlag = true,
					url = '${ctx}/system/syssettings/testServer.action',
					params={};
					
				if(vm.settingsForm.rsysLogEnable == '1'){
					if(vm.settingsForm.rsysLogIp == '' || vm.settingsForm.rsysLogIp === null || vm.settingsForm.rsysLogIp === undefined){
						testIpFlag = false
					}else{
						testIpFlag = true
					}
					if(vm.settingsForm.rsysLogPort === '' || vm.settingsForm.rsysLogPort === null || vm.settingsForm.rsysLogPort === undefined){
						testPortFlag = false
					}else{
						testPortFlag = true
					}
				}

				if (testIpFlag && testPortFlag) {
					ipPort = vm.settingsForm.rsysLogIp + ':' + vm.settingsForm.rsysLogPort;
					params.rsysLogConfig = ipPort;
					
					var saveParams = JSON.stringify(params);
					axios.post(url,saveParams,{headers:{'Content-Type':'application/json;charset=utf-8'},}).then(function (res) {
												
						if (res.data == true) {
							vm.settingsForm.rsysLogTestStatus = 'true'; 
							vm.$message({
								type: 'success',
								message: '<%=rb.getString("YanZhengTongGuo")%>'
							});
						}else{
							vm.settingsForm.rsysLogTestStatus = 'false';
							vm.$message({
								type: 'error',
								message: '<%=rb.getString("LDAPBuShiYouXiaoDe")%>'
							});
						}
					})
				}
			},
			// sysLog开关改变事件
			rsystemEnableChange(val){
				var vm =this;
				['rsysLogIp','rsysLogPort'].map((item)=>{
					vm.$refs.settingsForm.validateField(item);
				})	
			},
			// 判断是否为空
			isNull(val){
				if(val==undefined || val == null || val =="") return true;
				else return false;
			},
			// timezoneCode改变事件
			timezoneCodeChange(val){
				var vm =this;
				vm.queryTimeData.map(function(item){
					if(item.id == val){
						vm.settingsForm.timeZoneId = item.timeZoneId;
					}
				})
			},
			// 选择 ldap SSL证书方式
			ldapSSLChange(val){
				var vm = this;
				if(val){
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("QingLianXiGuanLiYuanShangChuanZhengShu")%>'
					});
				}
			},
			latToleranceRangeChange(val) {
				this.settingsForm.longitudeToleranceRange = val;
			}
		}
	})
</script>