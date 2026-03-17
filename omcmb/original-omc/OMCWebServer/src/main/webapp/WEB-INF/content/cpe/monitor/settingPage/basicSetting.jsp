<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#cpeBasicSettingPage{
	height: 100%;
	width: 100%;
}
#cpeBasicSettingPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#cpeBasicSettingPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#cpeBasicSettingPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#cpeBasicSettingPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#cpeBasicSettingPage .el-form-item{
	margin-bottom: 20px;
}
#cpeBasicSettingPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
#cpeBasicSettingPage .paramsItemBoxCls{
	display: flex;
	align-items: center;
	margin-bottom: 20px;
	min-width: 740px;
}
#cpeBasicSettingPage .paramsItemBoxCls .el-form-item{
	margin-bottom: 0px;
}
#cpeBasicSettingPage .paramsItemLabelCls{
	width: 140px;
}
#cpeBasicSettingPage .exportBtnBoxCls{
	height: 28px;
	width: 130px;
	margin-left: 140px;
	border : 1px solid #E9E9E9;
	border-radius: 4px;
	font-size: 12px;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	margin-bottom: 20px;
}
#cpeBasicSettingPage .exportTemplateBoxCls{
	display: flex;
	align-items: center;
}
#cpeBasicSettingPage .exportTemplateTipCls{
	color: rgba(0,0,0,0.32);
	margin-right: 20px;
}

#cpeBasicSettingPage .validate-item .el-input__inner{
	width:200px;
}
#cpeBasicSettingPage .validate-item .el-input-group__append{
	border:none;
	background:none;
	padding: 0px 10px;
}
#cpeBasicSettingPage .validate-item .el-form-item__error{
	display:none;
}
#cpeBasicSettingPage .is-error .el-input-group__append{
	color:#FA5555;
}
#cpeBasicSettingPage .notSupportTipBoxCls{
	color: rgba(0,0,0,0.32);
	font-size: 14px;
	margin-left: 20px;
}
</style>

<div id="cpeBasicSettingPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxCenter">
				<el-collapse v-model="activeCollapse">
					<el-collapse-item name="Basic">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Basic Config</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
								<div style="display:flex;flex-wrap: wrap">
									<el-form-item prop='cpeName' style="width:100%;min-width:400px;" label="<%=rb.getString("CPEName")%>" class='validate-item'>
										<el-input v-model.trim='ruleForm.cpeName' maxlength="45">
											<template slot="append"><%=rb.getString("FanWei")%>：0~45 Digit,String</template>
										</el-input>
									</el-form-item>
									<!-- 一个input时会导致 回车提交 -->
									<el-form-item v-show="false">
										<el-input ></el-input>
									</el-form-item>
								</div>
							</el-form>
						</div>
					</el-collapse-item>
					<el-collapse-item v-show="!isR005 && !is43XAP" name="Mac">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">MAC Filter</span>
							</p>
							<span class="notSupportTipBoxCls" v-show="['NotSupport'].includes(macRuleForm.macFilterEnable)">(Not support)</span>
							<span class="notSupportTipBoxCls" v-show="['NotSync'].includes(macRuleForm.macFilterEnable)">(Not sync)</span>
							<div class="newIconBoxCls-bt" style="right:30px;top:10px;" @click="syncMacSettingsClick" tip="<%=rb.getString("TongBu")%>">
								<span class="el-icon el-icon-circle-refresh"></span>
							</div>
						</template>
						<div :class="isMacLoading ? 'rightContentCls loading' : 'rightContentCls'">
							<el-form :model='macRuleForm' ref="macRuleForm" :disabled="['NotSupport','NotSync'].includes(macRuleForm.macFilterEnable)" :rules="rules" label-position="top">
								<div class="paramsItemBoxCls">
									<div class="paramsItemLabelCls">MAC Filter Enable</div>
									<el-form-item prop='macFilterEnable' style="width:40%;min-width:400px;" label="">
										<el-switch v-model="macRuleForm.macFilterEnable" active-value="1" inactive-value="0"></el-switch>
									</el-form-item>
								</div>
								<div class="paramsItemBoxCls">
									<div class="paramsItemLabelCls">MAC Filter Mode</div>
									<el-form-item prop='macFilterMode' style="width:40%;min-width:400px;" label="">
										<el-radio-group v-model="macRuleForm.macFilterMode" >
											<el-radio label="1" border size="small">Whitelist</el-radio>
											<el-radio label="0" border size="small">Blocklist</el-radio>
										</el-radio-group>
									</el-form-item>
								</div>
								<div class="paramsItemBoxCls">
									<div class="paramsItemLabelCls">MAC Add Mode</div>
									<el-form-item prop='macAddMode' style="width:40%;min-width:400px;" label="">
										<el-radio-group v-model="macRuleForm.macAddMode">
											<el-radio label="0" border size="small">Manual Input</el-radio>
											<el-radio label="1" border size="small">Batch Import</el-radio>
										</el-radio-group>
									</el-form-item>
								</div>
								<div class="paramsItemBoxCls" style="margin-bottom: 5px;">
									<div class="paramsItemLabelCls">MAC Address</div>
									<div class="allowMoreInputBoxCls">
										<div class="allowMoreInputContentCls">
											<div class="allowMoreInputAddedCls">
												Added (
												<span v-if="macRuleForm.macFilterMode == '0'" style="color: #4D84FF;">{{macRuleForm.macFilterBlackList.length}} / 128</span>
												<span v-if="macRuleForm.macFilterMode == '1'" style="color: #4D84FF;">{{macRuleForm.macFilterWhiteList.length}} / 128</span> )
											</div>
											<div class="allowMoreInputFieldCls">
												<el-input v-show="macRuleForm.macAddMode == '0'" v-model="macRuleForm.macAddress"></el-input>
												<el-upload ref="macAddressAddUpload"
													v-show="macRuleForm.macAddMode == '1'"
													:before-upload='macAddressAddBeforeUpload' 
													:on-success='macAddressAddCheckFile' 
													:on-change="macAddressAddFileChange" 
													:show-file-list="false" 
													:action="macAddressAddUploadFileURL" 
													:data="macAddressAddFileParams" 
													name="macAddressUploadFile" 
													:auto-upload="false"
													accept=".xls,.xlsx">
													<el-input :readonly="true" :value="macAddressAddFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:300px;">
														<a slot="append" class="el-icon el-icon-operation-import grayIcon" @click="macAddressAddFileSelect"></a>
													</el-input>
													<a slot="trigger" ref="macAddressAddFile_up"></a>
												</el-upload>
												<div v-show="macRuleForm.macAddMode == '0'" class="allowMoreInputAddBtnCls" @click="macAddressAdd">
													<span class="el-icon el-icon-plus"></span>
													<span>Add</span>
												</div>
												<div class="allowMoreInputAddTipCls">
													<span style="color:red" v-if="macAddressErrorMessage">{{macAddressErrorMessage}}</span>
													<span style="color:rgba(0,0,0,0.32)" v-if="!macAddressErrorMessage && macRuleForm.macAddMode == '0'">Format：xx:xx:xx:xx:xx:xx,no more than 128</span>
													<span style="color:rgba(0,0,0,0.32)" v-if="!macAddressErrorMessage && macRuleForm.macAddMode == '1'"><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>
												</div>
											</div>
											<div v-show="macRuleForm.macAddMode == '1'" class="exportTemplateBoxCls">
												<div class="exportTemplateTipCls"><%=rb.getString("ShiYongMuBanDaoRuTiShi")%></div>
												<div @click="exportAddTemplate('MAC')" style="cursor:pointer;">
													<span class='el-icon el-icon-common-download exportTemplateIcon'></span>
													<span  class='exportTemplateText'><%=rb.getString("DaoChuMuBan")%></span>
												</div>
											</div>
											<div class="allowMoreInputParamsCls" v-show="macRuleForm.macFilterMode == '1'">
												<div v-for="item in macRuleForm.macFilterWhiteList" class="allowMoreInputParamsItemCls">
													<span>{{item.macAddress}}</span>
													<span class="el-icon el-icon-close" @click="macAddressListDel(item)"></span>
												</div>
											</div>
											<div class="allowMoreInputParamsCls" v-show="macRuleForm.macFilterMode == '0'">
												<div v-for="item in macRuleForm.macFilterBlackList" class="allowMoreInputParamsItemCls">
													<span>{{item.macAddress}}</span>
													<span class="el-icon el-icon-close" @click="macAddressListDel(item)"></span>
												</div>
											</div>
											<el-form-item prop='macFilterWhiteList' style="display:none;" label="" label-width="0px">
												<el-input v-model='macRuleForm.macFilterWhiteList'></el-input>
											</el-form-item>
											<el-form-item prop='macFilterBlackList' style="display:none;" label="" label-width="0px">
												<el-input v-model='macRuleForm.macFilterBlackList'></el-input>
											</el-form-item>
											<el-form-item prop='macFilterWhiteListRemoved' style="display:none;" label="" label-width="0px">
												<el-input v-model='macRuleForm.macFilterWhiteListRemoved'></el-input>
											</el-form-item>
											<el-form-item prop='macFilterBlackListRemoved' style="display:none;" label="" label-width="0px">
												<el-input v-model='macRuleForm.macFilterBlackListRemoved'></el-input>
											</el-form-item>
										</div>
									</div>
								</div>
								<div v-show="!['NotSupport','NotSync'].includes(macRuleForm.macFilterEnable)" class="exportBtnBoxCls" @click="macAddressListExportClick">
									<span class="el-icon el-icon-operation-export"></span>
									<span style="margin-left: 5px;">Export MAC List</span>
								</div>
							</el-form>
						</div>
					</el-collapse-item>
					<el-collapse-item v-show="!isR005 && !is43XAP" name="IP">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">IP Filter</span>
							</p>
							<span class="notSupportTipBoxCls" v-show="['NotSupport'].includes(ipRuleForm.ipFilterEnable)">(Not support)</span>
							<span class="notSupportTipBoxCls" v-show="['NotSync'].includes(ipRuleForm.ipFilterEnable)">(Not sync)</span>
							<div class="newIconBoxCls-bt" style="right:30px;top:10px;" @click="syncIpSettingsClick" tip="<%=rb.getString("TongBu")%>">
								<span class="el-icon el-icon-circle-refresh"></span>
							</div>
						</template>
						<div :class="isIpLoading ? 'rightContentCls loading' : 'rightContentCls'">
							<el-form :model='ipRuleForm' ref="ipRuleForm" :disabled="['NotSupport','NotSync'].includes(ipRuleForm.ipFilterEnable)"  :rules="rules" label-position="top">
								<div class="paramsItemBoxCls">
									<div class="paramsItemLabelCls">IP Filter Enable</div>
									<el-form-item prop='ipFilterEnable' style="width:40%;min-width:400px;" label="">
										<el-switch v-model="ipRuleForm.ipFilterEnable" active-value="1" inactive-value="0"></el-switch>
									</el-form-item>
								</div>
								<div class="paramsItemBoxCls">
									<div class="paramsItemLabelCls">IP Filter Mode</div>
									<el-form-item prop='ipFilterMode' style="width:40%;min-width:400px;" label="">
										<el-radio-group v-model="ipRuleForm.ipFilterMode" >
											<el-radio label="1" border size="small">Whitelist</el-radio>
											<el-radio label="0" border size="small">Blocklist</el-radio>
										</el-radio-group>
									</el-form-item>
								</div>
								<div class="paramsItemBoxCls">
									<div class="paramsItemLabelCls">IP Add Mode</div>
									<el-form-item prop='ipAddMode' style="width:40%;min-width:400px;" label="">
										<el-radio-group v-model="ipRuleForm.ipAddMode">
											<el-radio label="0" border size="small">Manual Input</el-radio>
											<el-radio label="1" border size="small">Batch Import</el-radio>
										</el-radio-group>
									</el-form-item>
								</div>
								<div class="paramsItemBoxCls" style="margin-bottom: 5px;">
									<div class="paramsItemLabelCls">Source IP Address</div>
									<div class="allowMoreInputBoxCls">
										<div class="allowMoreInputContentCls">
											<div class="allowMoreInputAddedCls">
												Added (
												<span v-if="ipRuleForm.ipFilterMode == '0'" style="color: #4D84FF;">{{ipRuleForm.ipFilterBlackList.length}} / 128</span>
												<span v-if="ipRuleForm.ipFilterMode == '1'" style="color: #4D84FF;">{{ipRuleForm.ipFilterWhiteList.length}} / 128</span> )
											</div>
											<div class="allowMoreInputFieldCls">
												<el-input v-show="ipRuleForm.ipAddMode == '0'" v-model="ipRuleForm.sourceIp"></el-input>
												<el-upload ref="sourceIpAddUpload"
													v-show="ipRuleForm.ipAddMode == '1'"
													:before-upload='sourceIpAddBeforeUpload' 
													:on-success='sourceIpAddCheckFile' 
													:on-change="sourceIpAddFileChange" 
													:show-file-list="false" 
													:action="sourceIpAddUploadFileURL" 
													:data="sourceIpAddFileParams" 
													name="sourceIpUploadFile" 
													:auto-upload="false"
													accept=".xlsx, .csv">
													<el-input :readonly="true" :value="sourceIpAddFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:300px;">
														<a slot="append" class="el-icon el-icon-operation-import grayIcon" @click="sourceIpAddFileSelect"></a>
													</el-input>
													<a slot="trigger" ref="sourceIpAddFile_up"></a>
												</el-upload>
												<div v-show="ipRuleForm.ipAddMode == '0'" class="allowMoreInputAddBtnCls" @click="sourceIpAdd">
													<span class="el-icon el-icon-plus"></span>
													<span>Add</span>
												</div>
												<div class="allowMoreInputAddTipCls">
													<span style="color:red" v-if="sourceIpErrorMessage">{{sourceIpErrorMessage}}</span>
													<span style="color:rgba(0,0,0,0.32)" v-if="!sourceIpErrorMessage && ipRuleForm.ipAddMode == '0'">Support IPv4 or IPv6,no more than 128</span>
													<span style="color:rgba(0,0,0,0.32)" v-if="!sourceIpErrorMessage && ipRuleForm.ipAddMode == '1'"><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>
												</div>
											</div>
											<div v-show="ipRuleForm.ipAddMode == '1'" class="exportTemplateBoxCls">
												<div class="exportTemplateTipCls"><%=rb.getString("ShiYongMuBanDaoRuTiShi")%></div>
												<div @click="exportAddTemplate('IP')" style="cursor:pointer;">
													<span class='el-icon el-icon-common-download exportTemplateIcon'></span>
													<span  class='exportTemplateText'><%=rb.getString("DaoChuMuBan")%></span>
												</div>
											</div>
											<div class="allowMoreInputParamsCls" v-show="ipRuleForm.ipFilterMode == '1'">
												<div v-for="item in ipRuleForm.ipFilterWhiteList" class="allowMoreInputParamsItemCls">
													<span>{{item.sourceIp}}</span>
													<span class="el-icon el-icon-close" @click="sourceIpListDel(item)"></span>
												</div>
											</div>
											<div class="allowMoreInputParamsCls" v-show="ipRuleForm.ipFilterMode == '0'">
												<div v-for="item in ipRuleForm.ipFilterBlackList" class="allowMoreInputParamsItemCls">
													<span>{{item.sourceIp}}</span>
													<span class="el-icon el-icon-close" @click="sourceIpListDel(item)"></span>
												</div>
											</div>
											<el-form-item prop='ipFilterWhiteList' style="display:none;" label="" label-width="0px">
												<el-input v-model='ipRuleForm.ipFilterWhiteList'></el-input>
											</el-form-item>
											<el-form-item prop='ipFilterBlackList' style="display:none;" label="" label-width="0px">
												<el-input v-model='ipRuleForm.ipFilterBlackList'></el-input>
											</el-form-item>
											<el-form-item prop='ipFilterWhiteListRemoved' style="display:none;" label="" label-width="0px">
												<el-input v-model='ipRuleForm.ipFilterWhiteListRemoved'></el-input>
											</el-form-item>
											<el-form-item prop='ipFilterBlackListRemoved' style="display:none;" label="" label-width="0px">
												<el-input v-model='ipRuleForm.ipFilterBlackListRemoved'></el-input>
											</el-form-item>
										</div>
									</div>
								</div>
								<div v-show="!['NotSupport','NotSync'].includes(ipRuleForm.ipFilterEnable)" class="exportBtnBoxCls" @click="sourceIpListExportClick">
									<span class="el-icon el-icon-operation-export"></span>
									<span style="margin-left: 5px;">Export IP List</span>
								</div>
							</el-form>
						</div>
					</el-collapse-item>
					<el-collapse-item v-show="!isR005 && !is43XAP" name="URL">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">URL Blocklist Filter</span>
							</p>
							<span class="notSupportTipBoxCls" v-show="['NotSupport'].includes(urlRuleForm.urlFilterEnable)">(Not support)</span>
							<span class="notSupportTipBoxCls" v-show="['NotSync'].includes(urlRuleForm.urlFilterEnable)">(Not sync)</span>
							<div class="newIconBoxCls-bt" style="right:30px;top:10px;" @click="syncUrlSettingsClick" tip="<%=rb.getString("TongBu")%>">
								<span class="el-icon el-icon-circle-refresh"></span>
							</div>
						</template>
						<div :class="isUrlLoading ? 'rightContentCls loading' : 'rightContentCls'">
							<el-form :model='urlRuleForm' ref="urlRuleForm" :disabled="['NotSupport','NotSync'].includes(urlRuleForm.urlFilterEnable)" :rules="rules" label-position="top">
								<div class="paramsItemBoxCls">
									<div class="paramsItemLabelCls">URL Filter Enable</div>
									<el-form-item prop='urlFilterEnable' style="width:40%;min-width:400px;" label="">
										<el-switch v-model="urlRuleForm.urlFilterEnable" active-value="1" inactive-value="0"></el-switch>
									</el-form-item>
								</div>
								<div class="paramsItemBoxCls">
									<div class="paramsItemLabelCls">URL Add Mode</div>
									<el-form-item prop='urlAddMode' style="width:40%;min-width:400px;" label="">
										<el-radio-group v-model="urlRuleForm.urlAddMode">
											<el-radio label="0" border size="small">Manual Input</el-radio>
											<el-radio label="1" border size="small">Batch Import</el-radio>
										</el-radio-group>
									</el-form-item>
								</div>
								<div class="paramsItemBoxCls" style="margin-bottom: 5px;">
									<div class="paramsItemLabelCls">URL</div>
									<div class="allowMoreInputBoxCls">
										<div class="allowMoreInputContentCls">
											<div class="allowMoreInputAddedCls">
												Added (
												<span style="color: #4D84FF;">{{urlRuleForm.urlFilterBlackList.length}} / 64</span> )
											</div>
											<div class="allowMoreInputFieldCls">
												<el-input v-show="urlRuleForm.urlAddMode == '0'" v-model="urlRuleForm.url"></el-input>
												<el-upload ref="urlAddUpload"
													v-show="urlRuleForm.urlAddMode == '1'"
													:before-upload='urlAddBeforeUpload' 
													:on-success='urlAddCheckFile' 
													:on-change="urlAddFileChange" 
													:show-file-list="false" 
													:action="urlAddUploadFileURL" 
													:data="urlAddFileParams" 
													name="urlUploadFile" 
													:auto-upload="false"
													accept=".xlsx, .xls">
													<el-input :readonly="true" :value="urlAddFileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:300px;">
														<a slot="append" class="el-icon el-icon-operation-import grayIcon" @click="urlAddFileSelect"></a>
													</el-input>
													<a slot="trigger" ref="urlAddFile_up"></a>
												</el-upload>
												<div v-show="urlRuleForm.urlAddMode == '0'" class="allowMoreInputAddBtnCls" @click="urlAdd">
													<span class="el-icon el-icon-plus"></span>
													<span>Add</span>
												</div>
												<div class="allowMoreInputAddTipCls">
													<span style="color:red" v-if="urlErrorMessage">{{urlErrorMessage}}</span>
													<span style="color:rgba(0,0,0,0.32)" v-if="!urlErrorMessage && urlRuleForm.urlAddMode == '0'">no more than 64</span>
													<span style="color:rgba(0,0,0,0.32)" v-if="!urlErrorMessage && urlRuleForm.urlAddMode == '1'"><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>
												</div>
											</div>
											<div v-show="urlRuleForm.urlAddMode == '1'" class="exportTemplateBoxCls">
												<div class="exportTemplateTipCls"><%=rb.getString("ShiYongMuBanDaoRuTiShi")%></div>
												<div @click="exportAddTemplate('URL')" style="cursor:pointer;">
													<span class='el-icon el-icon-common-download exportTemplateIcon'></span>
													<span  class='exportTemplateText'><%=rb.getString("DaoChuMuBan")%></span>
												</div>
											</div>
											<div class="allowMoreInputParamsCls">
												<div v-for="item in urlRuleForm.urlFilterBlackList" class="allowMoreInputParamsItemCls">
													<span>{{item.url}}</span>
													<span class="el-icon el-icon-close" @click="urlListDel(item)"></span>
												</div>
											</div>
											<el-form-item prop='urlFilterBlackList' style="display:none;" label="" label-width="0px">
												<el-input v-model='urlRuleForm.urlFilterBlackList'></el-input>
											</el-form-item>
											<el-form-item prop='urlFilterBlackListRemoved' style="display:none;" label="" label-width="0px">
												<el-input v-model='urlRuleForm.urlFilterBlackListRemoved'></el-input>
											</el-form-item>
										</div>
									</div>
								</div>
								<div v-show="!['NotSupport','NotSync'].includes(urlRuleForm.urlFilterEnable)" class="exportBtnBoxCls" @click="urlListExportClick">
									<span class="el-icon el-icon-operation-export"></span>
									<span style="margin-left: 5px;">Export URL List</span>
								</div>
							</el-form>
						</div>
					</el-collapse-item>
				</el-collapse>
			</el-form>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var cpeBasicSettingPage = new Vue({
	el: '#cpeBasicSettingPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			};
		return {
			activeCollapse:['Basic','Mac','IP','URL'],
			cpeCode:'',
			ruleForm:{
				cpeName:'',
			},
			macRuleForm:{
				macFilterEnable:'0',
				macFilterMode:'1',
				macAddMode:'0',
				macAddress:'',
				macFilterWhiteList:[],
				macFilterBlackList:[],
				macFilterWhiteListRemoved:[],
				macFilterBlackListRemoved:[],
			},
			ipRuleForm:{
				ipFilterEnable:'0',
				ipFilterMode:'1',
				ipAddMode:'0',
				sourceIp:'',
				ipFilterWhiteList:[],
				ipFilterBlackList:[],
				ipFilterWhiteListRemoved:[],
				ipFilterBlackListRemoved:[],
			},
			urlRuleForm:{
				urlFilterEnable:'0',
				urlFilterMode:'0',
				urlAddMode:'0',
				url:'',
				urlFilterBlackList:[],
				urlFilterBlackListRemoved:[],
			},

			rules:{},
			
			macAddressErrorMessage:'',
			macAddressAddUploadFileURL:'',
			macAddressAddFileParams:{},
			macAddressAddFileName:'',

			sourceIpErrorMessage:'',
			sourceIpAddUploadFileURL:'',
			sourceIpAddFileParams:{},
			sourceIpAddFileName:'',

			urlErrorMessage:'',
			urlAddUploadFileURL:'',
			urlAddFileParams:{},
			urlAddFileName:'',

			isMacLoading:false,
			isIpLoading:false,
			isUrlLoading:false,
			
		};
	},
	computed: {
		isR005() {
			var product = sessionStorage.getItem('oldProduct');

			return product.indexOf('R005') >= 0;
		},
		is43XAP() {
			var product = sessionStorage.getItem('oldProduct');

			return product.indexOf('Nova430X') >= 0 || product.indexOf('Neutrino430X') >= 0;
		}
	},
	methods: {
		init(itemParam,code){
			var vm = this;
			vm.cpeCode = code;
			
			vm.getParamData(code);
		},
		getParamData(code) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/CPE/getSettingParams.action',
				params = {
					cpeCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				['cpeName','macFilterEnable','macFilterMode','ipFilterEnable','ipFilterMode','urlFilterEnable'].map((key,index)=>{
					if(['cpeName'].includes(key)){
						vm.ruleForm[key] = data[key] ? data[key] : vm.ruleForm[key];
					}
					if(['macFilterEnable','macFilterMode'].includes(key)){
						vm.macRuleForm[key] = data[key] ? data[key] : vm.macRuleForm[key];
					}
					if(['ipFilterEnable','ipFilterMode'].includes(key)){
						vm.ipRuleForm[key] = data[key] ? data[key] : vm.ipRuleForm[key];
					}
					if(['urlFilterEnable'].includes(key)){
						vm.urlRuleForm[key] = data[key] ? data[key] : vm.urlRuleForm[key];
					}
					
				});

				['macFilterWhiteList','macFilterBlackList','ipFilterWhiteList','ipFilterBlackList','urlFilterBlackList'].map((key,index)=>{
					if(['macFilterWhiteList','macFilterBlackList'].includes(key)){
						var macList = data[key] ? data[key].split(';') : [];
						macList.map((item,index)=>{
							vm.macRuleForm[key].push({
								macAddress:item,
								paramsType:'default'
							});
						});
					};
					if(['ipFilterWhiteList','ipFilterBlackList'].includes(key)){
						var ipList = data[key] ? data[key].split(';') : [];
						ipList.map((item,index)=>{
							vm.ipRuleForm[key].push({
								sourceIp:item,
								paramsType:'default'
							});
						});
					};
					if(['urlFilterBlackList'].includes(key)){
						var urlList = data[key] ? data[key].split(';') : [];
						urlList.map((item,index)=>{
							vm.urlRuleForm[key].push({
								url:item,
								paramsType:'default'
							});
						});
					};
				});
				initForm(vm.$refs.ruleForm);
				initForm(vm.$refs.macRuleForm);
				initForm(vm.$refs.ipRuleForm);
				initForm(vm.$refs.urlRuleForm);
			});
		},
		// Mac Address添加事件
        macAddressAdd(){
            var vm = this,
                val = vm.macRuleForm.macAddress,
				addType = vm.macRuleForm.macAddMode,
				filterType = vm.macRuleForm.macFilterMode,
				listCode = {
					'1':'macFilterWhiteList',
					'0':'macFilterBlackList'
				},
				delListCode = {
					'1':'macFilterWhiteListRemoved',
					'0':'macFilterBlackListRemoved'
				},
                params={
                    macAddress:vm.macRuleForm.macAddress,
                };
			if(addType == '0'){
				if(vm.macRuleForm[listCode[filterType]].length >= 128){
					vm.$message.warning('No more than 128');
					return;
				}

				if(val){
					if(vm.isValidMacAddress(val)) {
						var result = vm.macRuleForm[listCode[filterType]].some(item=>item.macAddress == val);
						if(result){
							vm.macAddressErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							if(vm.macRuleForm[delListCode[filterType]].includes(val)){
								vm.macRuleForm[delListCode[filterType]].splice(vm.macRuleForm[delListCode[filterType]].indexOf(val),1);
								params.paramsType = 'default';
							}
							vm.macRuleForm[listCode[filterType]].push(params);
							vm.macRuleForm.macAddress = '';
							vm.macAddressErrorMessage = '';
						}
					}else {
						vm.macAddressErrorMessage = 'Format：xx:xx:xx:xx:xx:xx,no more than 128';
					}
				}
			}
		},
        // Mac Address  删除事件
        macAddressListDel(row){
            var vm = this
				filterType = vm.macRuleForm.macFilterMode,
				listCode = {
					'1':'macFilterWhiteList',
					'0':'macFilterBlackList'
				},
				delListCode = {
					'1':'macFilterWhiteListRemoved',
					'0':'macFilterBlackListRemoved'
				};
            if(row.paramsType && row.paramsType == 'default'){
                vm.macRuleForm[delListCode[filterType]].push(row.macAddress);
            }
            vm.macRuleForm[listCode[filterType]] = vm.macRuleForm[listCode[filterType]].filter((items)=>{
                return items.macAddress != row.macAddress
            })
        },
		macAddressAddBeforeUpload(file){
			var vm = this, 
				urls = '${ctx}/cell/CPE/setting/importMacFilterInfo.action',
				FileName = file.name,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
			fd.append('uploadFile',file); //文件流
			fd.append('FileName',FileName);//文件名
			axios.post(urls,fd,config).then(function(res){
				let data = res.data;
				if(data.success){
					if(data.macList){
						var macList = data.macList.split(';'),
							filterType = vm.macRuleForm.macFilterMode,
							listCode = {
								'1':'macFilterWhiteList',
								'0':'macFilterBlackList'
							};
						try{
							macList.forEach(item=>{
								let isExist = vm.macRuleForm[listCode[filterType]].some(items=>items.macAddress == item);
								
								if(!isExist){
									if(vm.macRuleForm[listCode[filterType]].length >= 128){
										vm.$message.warning('<%=rb.getString("PiLiangDaoRuChaoXianTiShi")%>');
										throw Error();
									}else{
										vm.macRuleForm[listCode[filterType]].push({
											macAddress:item
										})
									}
								}
							})
						}catch(e){}
					}else{
						vm.$message.warning('No valid data in the file');
					}
					vm.macAddressCloseAddFileSelect();
				}else{
					vm.$message.error(data["msg"])
				}
			})
			return false;
		},
		macAddressAddCheckFile(res,file){    //发送请求，校验device文件内容 
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
				vm.macAddressCloseAddFileSelect();
			}else{
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.macAddressAddUpload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		// 移除导入文件
		macAddressCloseAddFileSelect(){
			var vm = this;
			vm.macAddressAddFileName = '';
			vm.$refs.macAddressAddUpload.clearFiles();
		},
		// 选择文件
		macAddressAddFileSelect(){  
			var vm =this;
			vm.$refs.macAddressAddUpload.clearFiles();
			vm.$refs['macAddressAddFile_up'].click();
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		macAddressAddFileChange(file,fileList){ 
			var vm = this;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.xls'
			
			if(typeFlag){
				vm.macAddressAddFileName = file.name;
				vm.macAddressErrorMessage = '';	
				if(vm.macAddressAddFileName){
					vm.$refs.macAddressAddUpload.submit();
				}		
			}else {
				vm.macAddressAddFileName = '';
				vm.macAddressErrorMessage = '<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'
			}
		},
		// 导出 Mac List
		macAddressListExportClick(){ 
	    	var vm = this,
				urls = '${ctx}/cell/CPE/setting/exportMacFilterInfo.action',
				filterType = vm.macRuleForm.macFilterMode,
				listCode = {
					'1':'macFilterWhiteList',
					'0':'macFilterBlackList'
				},
    	    	params = {
					macs:''
				};
			params.macs = vm.macRuleForm[listCode[filterType]].map(item=>item.macAddress).join(';');
        	var bool = checkParams(params);
			if(!bool) return false;
        	exportByForm(urls,params)
	    },
		// source Ip添加事件
        sourceIpAdd(){
            var vm = this,
                val = vm.ipRuleForm.sourceIp,
				addType = vm.ipRuleForm.ipAddMode,
				filterType = vm.ipRuleForm.ipFilterMode,
				listCode = {
					'1':'ipFilterWhiteList',
					'0':'ipFilterBlackList'
				},
				delListCode = {
					'1':'ipFilterWhiteListRemoved',
					'0':'ipFilterBlackListRemoved'
				},
                params={
                    sourceIp:vm.ipRuleForm.sourceIp,
                };
			if(addType == '0'){
				if(vm.ipRuleForm[listCode[filterType]].length >= 128){
					vm.$message.warning('No more than 128');
					return;
				}
				if(val){
					if(vm.isValidIP(val) || vm.isIPv6(val)) {
						var result = vm.ipRuleForm[listCode[filterType]].some(item=>item.sourceIp == val);
						if(result){
							vm.sourceIpErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}else{
							if(vm.ipRuleForm[delListCode[filterType]].includes(val)){
								vm.ipRuleForm[delListCode[filterType]].splice(vm.ipRuleForm[delListCode[filterType]].indexOf(val),1);
								params.paramsType = 'default';
							}
							vm.ipRuleForm[listCode[filterType]].push(params);
							vm.ipRuleForm.sourceIp = '';
							vm.sourceIpErrorMessage = '';
						}
					}else {
						vm.sourceIpErrorMessage = 'Support IPv4 or IPv6,no more than 128';
					}
				}
			}
        },
        // source Ip  删除事件
        sourceIpListDel(row){
            var vm = this
				filterType = vm.ipRuleForm.ipFilterMode,
				listCode = {
					'1':'ipFilterWhiteList',
					'0':'ipFilterBlackList'
				}
				delListCode = {
					'1':'ipFilterWhiteListRemoved',
					'0':'ipFilterBlackListRemoved'
				};
            if(row.paramsType && row.paramsType == 'default'){
                vm.ipRuleForm[delListCode[filterType]].push(row.sourceIp);
            }
            vm.ipRuleForm[listCode[filterType]] = vm.ipRuleForm[listCode[filterType]].filter((items)=>{
                return items.sourceIp != row.sourceIp
            })
        },
		sourceIpAddBeforeUpload(file){
			var vm = this, 
				urls = '${ctx}/cell/CPE/setting/importIPFilterInfo.action',
				FileName = file.name,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
			fd.append('uploadFile',file); //文件流
			fd.append('FileName',FileName);//文件名
			axios.post(urls,fd,config).then(function(res){
				let data = res.data;

				if(data.success){
					if(data.ipList){
						var ipList = data.ipList.split(';'),
							filterType = vm.ipRuleForm.ipFilterMode,
							listCode = {
								'1':'ipFilterWhiteList',
								'0':'ipFilterBlackList'
							};
						try{
							ipList.forEach(item=>{
								let isExist = vm.ipRuleForm[listCode[filterType]].some(items=>items.sourceIp == item);
								
								if(!isExist){
									if(vm.ipRuleForm[listCode[filterType]].length >= 128){
										vm.$message.warning('<%=rb.getString("PiLiangDaoRuChaoXianTiShi")%>');
										throw Error();
									}else{
										vm.ipRuleForm[listCode[filterType]].push({
											sourceIp:item
										})
									}
								}
							})
						}catch(e){}
					}else{
						vm.$message.warning('No valid data in the file');
					}
					vm.sourceIpCloseAddFileSelect();
				}else{
					vm.$message.error(data["msg"])
				}
			})
			return false;
		},
		sourceIpAddCheckFile(res,file){    //发送请求，校验device文件内容 
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
				vm.sourceIpCloseAddFileSelect();
			}else{
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.sourceIpAddUpload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		// 移除导入文件
		sourceIpCloseAddFileSelect(){
			var vm = this;
			vm.sourceIpAddFileName = '';
			vm.$refs.sourceIpAddUpload.clearFiles();
		},
		// 选择文件
		sourceIpAddFileSelect(){  
			var vm =this;
			vm.$refs.sourceIpAddUpload.clearFiles();
			vm.$refs['sourceIpAddFile_up'].click();
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		sourceIpAddFileChange(file,fileList){ 
			var vm = this;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.xls'
			
			if(typeFlag){
				vm.sourceIpAddFileName = file.name;
				vm.sourceIpErrorMessage = '';	
				if(vm.sourceIpAddFileName){
					vm.$refs.sourceIpAddUpload.submit();
				}			
			}else {
				vm.sourceIpAddFileName = '';
				vm.sourceIpErrorMessage = '<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'
			}
		},
		// 导出 source Ip List
		sourceIpListExportClick(){ 
	    	var vm = this,
				urls = '${ctx}/cell/CPE/setting/exportIPFilterInfo.action',
				filterType = vm.ipRuleForm.ipFilterMode,
				listCode = {
					'1':'ipFilterWhiteList',
					'0':'ipFilterBlackList'
				},
    	    	params = {
					ips:''
				};
			params.ips = vm.ipRuleForm[listCode[filterType]].map(item=>item.sourceIp).join(';');
        	var bool = checkParams(params);
			if(!bool) return false;
        	exportByForm(urls,params)
	    },
		// URL 添加事件
        urlAdd(){
			var vm = this,
                val = vm.urlRuleForm.url,
				addType = vm.urlRuleForm.urlAddMode,
				filterType = vm.urlRuleForm.urlFilterMode,
				listCode = {
					'1':'urlFilterWhiteList',
					'0':'urlFilterBlackList'
				},
				delListCode = {
					'1':'urlFilterWhiteListRemoved',
					'0':'urlFilterBlackListRemoved'
				},
                params={
                    url:vm.urlRuleForm.url,
                };
			if(addType == '0'){
				
				if(vm.urlRuleForm[listCode[filterType]].length >= 64){
					vm.$message.warning('No more than 64');
					return;
				}
				if(val){
					var result = vm.urlRuleForm[listCode[filterType]].some(item=>item.url == val);
					if(result){
						vm.urlErrorMessage = '<%=rb.getString("YiCunZai")%>';
					}else{
						if(vm.urlRuleForm[delListCode[filterType]].includes(val)){
							vm.urlRuleForm[delListCode[filterType]].splice(vm.urlRuleForm[delListCode[filterType]].indexOf(val),1);
							params.paramsType = 'default';
						}
						vm.urlRuleForm[listCode[filterType]].push(params);
						vm.urlRuleForm.url = '';
						vm.urlErrorMessage = '';
					}
				}
			}
        },
        // URL 删除事件
        urlListDel(row){
			var vm = this
				filterType = vm.urlRuleForm.urlFilterMode,
				listCode = {
					'1':'urlFilterWhiteList',
					'0':'urlFilterBlackList'
				}
				delListCode = {
					'1':'urlFilterWhiteListRemoved',
					'0':'urlFilterBlackListRemoved'
				};
            if(row.paramsType && row.paramsType == 'default'){
                vm.urlRuleForm[delListCode[filterType]].push(row.url);
            }
            vm.urlRuleForm[listCode[filterType]] = vm.urlRuleForm[listCode[filterType]].filter((items)=>{
                return items.url != row.url
            })
        },
		urlAddBeforeUpload(file){
			var vm = this, 
				urls = '${ctx}/cell/CPE/setting/importUrlFilterInfo.action',
				FileName = file.name,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
			fd.append('uploadFile',file); //文件流
			fd.append('FileName',FileName);//文件名
			axios.post(urls,fd,config).then(function(res){
				let data = res.data;
				if(data.success){
					if(data.urlList){
						var urlList = data.urlList.split(';'),
							filterType = vm.urlRuleForm.urlFilterMode,
							listCode = {
								'1':'urlFilterWhiteList',
								'0':'urlFilterBlackList'
							};
						
						try{
							urlList.forEach(item=>{
								let isExist = vm.urlRuleForm[listCode[filterType]].some(items=>items.url == item);
								
								if(!isExist){
									if(vm.urlRuleForm[listCode[filterType]].length >= 64){
										vm.$message.warning('<%=rb.getString("PiLiangDaoRuChaoXianTiShi")%>');
										throw Error();
									}else{
										vm.urlRuleForm[listCode[filterType]].push({
											url:item
										})
									}
								}
							})
						}catch(e){}
					}else{
						vm.$message.warning('No valid data in the file')
					}
					vm.urlCloseAddFileSelect();
				}else{
					vm.$message.error(data["msg"])
				}
			})
			return false;
		},
		urlAddCheckFile(res,file){    //发送请求，校验device文件内容 
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
				vm.urlCloseAddFileSelect();
			}else{
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.urlAddUpload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		// 移除导入文件
		urlCloseAddFileSelect(){
			var vm = this;
			vm.urlAddFileName = '';
			vm.$refs.urlAddUpload.clearFiles();
		},
		// 选择文件
		urlAddFileSelect(){  
			var vm =this;
			vm.$refs.urlAddUpload.clearFiles();
			vm.$refs['urlAddFile_up'].click();
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		urlAddFileChange(file,fileList){ 
			var vm = this;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' || file.name.substr(file.name.lastIndexOf("."))  === '.xls'
			
			if(typeFlag){
				vm.urlAddFileName = file.name;
				vm.urlErrorMessage = '';
				if(vm.urlAddFileName){
					vm.$refs.urlAddUpload.submit();
				}			
			}else {
				vm.urlAddFileName = '';
				vm.urlErrorMessage = '<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'
			}
		},
		// 导出 URL List
		urlListExportClick(){ 
			var vm = this,
				urls = '${ctx}/cell/CPE/setting/exportUrlFilterInfo.action',
				filterType = vm.urlRuleForm.urlFilterMode,
				listCode = {
					'1':'urlFilterWhiteList',
					'0':'urlFilterBlackList'
				},
    	    	params = {
					urls:''
				};
			params.urls = vm.urlRuleForm[listCode[filterType]].map(item=>item.url).join(';');
        	var bool = checkParams(params);
			if(!bool) return false;
        	exportByForm(urls,params)
	    },
		// 导出模板
		exportAddTemplate(type){
			var vm = this,
				urlCodes={
					'MAC':'${ctx}/cell/CPE/setting/exportMacFilterTemplateInfo.action',
					'IP':'${ctx}/cell/CPE/setting/exportIPFilterTemplateInfo.action',
					'URL':'${ctx}/cell/CPE/setting/exportUrlFilterTemplateInfo.action'
				}
				urls = urlCodes[type],
    	    	params = {};
			
        	var bool = checkParams(params);
			if(!bool) return false;
        	exportByForm(urls,params)
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
		},
		// 验证输入的是否是整数
		isInteger(str) {
			if(str.length==0){
				return false;
			}
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(!reg.test(str)){
				return false;
			}
			return true;  
		},
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		settingsSubmit(){
			var vm = this;
			var params = {
					cpeCode:vm.cpeCode,
				},
				changeKeys = [],
				isBasicChanged = isFormChanged(vm.$refs.ruleForm),
				isMacChange = false,
				isIpChange = false,
				isUrlChange = false;
			vm.$refs.ruleForm.fields.map(function(field){

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						changeKeys.push(field.prop);
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
                        var key = field.prop;
						changeKeys.push(field.prop);
					};
				}
			});
			vm.$refs.macRuleForm.fields.map(function(field){

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						changeKeys.push(field.prop);
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
                        var key = field.prop;
						changeKeys.push(field.prop);
					};
				}
			});
			vm.$refs.ipRuleForm.fields.map(function(field){

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						changeKeys.push(field.prop);
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
                        var key = field.prop;
						changeKeys.push(field.prop);
					};
				}
			});
			vm.$refs.urlRuleForm.fields.map(function(field){

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						changeKeys.push(field.prop);
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
                        var key = field.prop;
						changeKeys.push(field.prop);
					};
				}
			});
			if(changeKeys.indexOf('cpeName') != -1){
				params.cpeName = vm.ruleForm.cpeName;
				params.cpeNameChanged =  '1';
			}
			['macFilterEnable','macFilterMode','macFilterWhiteList','macFilterBlackList','macFilterWhiteListRemoved','macFilterBlackListRemoved'].map(function(key){
				if(changeKeys.indexOf(key) != -1){
					isMacChange = true;
				}
			});
			['ipFilterEnable','ipFilterMode','ipFilterWhiteList','ipFilterBlackList','ipFilterWhiteListRemoved','ipFilterBlackListRemoved'].map(function(key){
				if(changeKeys.indexOf(key) != -1){
					isIpChange = true;
				}
			});
			['urlFilterEnable','urlFilterMode','urlFilterBlackList','urlFilterBlackListRemoved'].map(function(key){
				if(changeKeys.indexOf(key) != -1){
					isUrlChange = true;
				}
			});
			if(!isBasicChanged && !isMacChange && !isIpChange && !isUrlChange){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}
			if(isMacChange){
				var macFilterWhiteList = vm.macRuleForm.macFilterWhiteList.map(function(item){return item.macAddress}),
					macFilterBlackList = vm.macRuleForm.macFilterBlackList.map(function(item){return item.macAddress}),
					macFilterWhiteListRemoved = vm.macRuleForm.macFilterWhiteListRemoved,
					macFilterBlackListRemoved = vm.macRuleForm.macFilterBlackListRemoved;
				params.macFilterChanged = '1';
				['macFilterEnable','macFilterMode'].map((key,index)=>{
					params[key] = vm.macRuleForm[key];
				});
				if(vm.macRuleForm.macFilterMode == '1'){
					params.macFilterWhiteList = macFilterWhiteList.join(';');
					params.macFilterWhiteListRemoved = macFilterWhiteListRemoved.join(';');
				}else{
					params.macFilterBlackList = macFilterBlackList.join(';');
					params.macFilterBlackListRemoved = macFilterBlackListRemoved.join(';');
				}
			}
			if(isIpChange){
				var ipFilterWhiteList = vm.ipRuleForm.ipFilterWhiteList.map(function(item){return item.sourceIp}),
					ipFilterBlackList = vm.ipRuleForm.ipFilterBlackList.map(function(item){return item.sourceIp}),
					ipFilterWhiteListRemoved = vm.ipRuleForm.ipFilterWhiteListRemoved,
					ipFilterBlackListRemoved = vm.ipRuleForm.ipFilterBlackListRemoved;
				params.ipFilterChanged = '1';
				['ipFilterEnable','ipFilterMode'].map((key,index)=>{
					params[key] = vm.ipRuleForm[key];
				});
				if(vm.ipRuleForm.ipFilterMode == '1'){
					params.ipFilterWhiteList = ipFilterWhiteList.join(';');
					params.ipFilterWhiteListRemoved = ipFilterWhiteListRemoved.join(';');
				}else{
					params.ipFilterBlackList = ipFilterBlackList.join(';');
					params.ipFilterBlackListRemoved = ipFilterBlackListRemoved.join(';');
				}
			}
			if(isUrlChange){
				var urlFilterBlackList = vm.urlRuleForm.urlFilterBlackList.map(function(item){return item.url}),
					urlFilterBlackListRemoved = vm.urlRuleForm.urlFilterBlackListRemoved;
				params.urlFilterChanged = '1';
				['urlFilterEnable','urlFilterMode'].map((key,index)=>{
					params[key] = vm.urlRuleForm[key];
				});
				params.urlFilterBlackList = urlFilterBlackList.join(';');
				params.urlFilterBlackListRemoved = urlFilterBlackListRemoved.join(';');
			}
			var rowCode = vm.cpeCode,
				url = '${ctx}/cell/CPE/setCpeParams.action?cpeCode='+rowCode;
			$('#setting_main_cpe').addClass('loading');
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type: 'success'
					});
					vm.closeSettings();
					$("#setting_main_cpe").removeClass("loading");
				}else{
					vm.$message({
						message: data["message"],
						type: 'error'
					});
				}
				$('#setting_main_cpe').removeClass('loading');
			});
		},
		closeSettings(){
			eventBus.$emit('close-cpe-setting');
		},
		//校验MAC
		isValidMacAddress(mac){
			var reg = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/;
			return reg.test(mac); 
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
        //校验子网掩码
        isMask(str){
            var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
            return exp.test(str); 		
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
		// 同步刷新 mac Filter
		syncMacSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/CPE/setting/queryMacFilterInfo.action',
				params = {
					cpeCode:vm.cpeCode
				},
				str = Math.random().toString();
			vm.isMacLoading = true;	
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data){
					['macFilterWhiteList','macFilterBlackList','macFilterEnable','macFilterMode','macFilterWhiteListRemoved','macFilterBlackListRemoved'].map((key,index)=>{
						if(['macFilterWhiteList','macFilterBlackList'].includes(key)){
							var macList = data[key] ? data[key].split(';') : [],
								newMacList=[];
							macList.map((item,index)=>{
								newMacList.push({
									macAddress:item,
									paramsType:'default'
								});
							});
							vm.macRuleForm[key] = newMacList;
						};
						if(['macFilterEnable','macFilterMode'].includes(key)){
							vm.macRuleForm[key] = data[key] ? data[key] : vm.macRuleForm[key];
						};
						if(['macFilterWhiteListRemoved','macFilterBlackListRemoved'].includes(key)){
							vm.macRuleForm[key] = [];
						};
					});
					initForm(vm.$refs.macRuleForm);
				}else{
					vm.$message.error(data["message"])
				}
				vm.isMacLoading = false;
			});
			event.stopPropagation();
		},
		// 同步刷新 ip Filter
		syncIpSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/CPE/setting/queryIPFilterInfo.action',
				params = {
					cpeCode:vm.cpeCode
				},
				str = Math.random().toString();
			vm.isIpLoading = true;
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data){
					['ipFilterWhiteList','ipFilterBlackList','ipFilterEnable','ipFilterMode','ipFilterWhiteListRemoved','ipFilterBlackListRemoved'].map((key,index)=>{
						if(['ipFilterWhiteList','ipFilterBlackList'].includes(key)){
							var ipList = data[key] ? data[key].split(';') : [],
								newIpList = [];
							ipList.map((item,index)=>{
								newIpList.push({
									sourceIp:item,
									paramsType:'default'
								});
							});
							vm.ipRuleForm[key] = newIpList;
						};
						if(['ipFilterEnable','ipFilterMode'].includes(key)){
							vm.ipRuleForm[key] = data[key] ? data[key] : vm.ipRuleForm[key];
						};
						if(['ipFilterWhiteListRemoved','ipFilterBlackListRemoved'].includes(key)){
							vm.ipRuleForm[key] = [];
						};
					});
					initForm(vm.$refs.ipRuleForm);
				}else{
					vm.$message.error(data["message"])
				}
				vm.isIpLoading = false;
			});
			event.stopPropagation();
		},
		// 同步刷新 url Filter
		syncUrlSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/CPE/setting/queryUrlFilterInfo.action',
				params = {
					cpeCode:vm.cpeCode
				},
				str = Math.random().toString();
			vm.isUrlLoading = true;
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data){
					['urlFilterEnable','urlFilterBlackList','urlFilterBlackListRemoved'].map((key,index)=>{
						if(['urlFilterBlackList'].includes(key)){
							var urlList = data[key] ? data[key].split(';') : []
								newUrlList = [];
							urlList.map((item,index)=>{
								newUrlList.push({
									url:item,
									paramsType:'default'
								});
							});
							vm.urlRuleForm[key] = newUrlList;
						};
						if(['urlFilterEnable'].includes(key)){
							vm.urlRuleForm[key] = data[key] ? data[key] : vm.urlRuleForm[key];
						};
						if(['urlFilterBlackListRemoved'].includes(key)){
							vm.urlRuleForm[key] = [];
						};
					});
					initForm(vm.$refs.urlRuleForm);
				}else{
					vm.$message.error(data["message"])
				}
				vm.isUrlLoading = false;
			});
			event.stopPropagation();
		},
	},
	mounted() {
		eventBus.$off("cpe-data").$on("cpe-data",this.init)
	}
});

</script>
