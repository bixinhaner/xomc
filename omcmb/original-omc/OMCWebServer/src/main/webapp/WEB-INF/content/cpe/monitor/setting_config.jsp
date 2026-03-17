<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#cpe_setting_ctn .readonly {
		position: relative;
	}
	#cpe_setting_ctn .readonly::before {
		content: '';
		display: block;
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		left: 0;
		z-index: 10;
	}
	#cpe_setting_ctn .readonly input, 
	#cpe_setting_ctn .readonly .el-switch {
		opacity: 0.7;
	}

	#cpe_setting_ctn .group-setting {
		height: 100%;
		display: flex;
		flex-direction: column;
	}
	#cpe_setting_ctn .group-setting .el-card__body {
		padding: 0px;
		display: flex;
		flex-direction: row;
		flex: auto;
		overflow: auto;
	}
	#cpe_setting_ctn .el-card__body {
		background:#FFF;
	}
	#cpe_setting_ctn .nav-flex-item {
		flex: auto;
		overflow: auto;
	}
	#cpe_setting_ctn .nav-tabs {
		border-right: 1px solid #eee;
		min-width: 138px;
	}
	#cpe_setting_ctn .nav-tabs div {
		height: 36px;
		line-height: 36px;
		padding: 0 20px;
		cursor: pointer;
		border-bottom: 1px solid #eee;
		word-break: keep-all;
	}
	#cpe_setting_ctn .nav-tabs div.active {
		color: #4D84FF;
		background-color: #EDF6FF;
	}
	#cpe_setting_ctn .nav-title {
		padding: 0 20px;
		height: 40px;
		line-height: 40px;
		color: #363b4e;
		font-weight: bold;
		border-bottom: 1px solid #eee;
	}
	#cpe_setting_ctn .nav-main {
		display: flex;
		flex-direction: column;
		flex: auto;
		overflow: auto;
	}
	#cpe_setting_ctn .nav-content {
		padding: 20px;
	}
	#cpe_setting_ctn .nav-content .el-form-item__label, .cpeMonitorSettingBoxDialogCls .label-flex .el-form-item__label {
		text-align: left;
		line-height: 28px;
	}
	#cpe_setting_ctn .nav-operations {
		height: 48px;
		line-height: 48px;
		padding-left: 40px;
		border-top: 1px solid #eee;
	}
	#cpe_setting_ctn .earfcn-pci-line {
		display: flex;
		align-items: center;
		margin-bottom: 20px;
	}
	#cpe_setting_ctn .el-form-item__error, .cpeMonitorSettingBoxDialogCls .el-form-item__error{
		left: 40px;
	}
	#cpe_setting_ctn .suffixItem ,.cpeMonitorSettingBoxDialogCls .suffixItem{
		display: inline-block;
		margin-right:10px;
		margin-bottom:10px !important;
	}
	#cpe_setting_ctn .suffixItem .el-form-item__content,.cpeMonitorSettingBoxDialogCls .suffixItem .el-form-item__content{
		line-height:16px;
	}
	#cpe_setting_ctn .form-suffix,.cpeMonitorSettingBoxDialogCls .form-suffix{
		border:1px solid #4D84FF; 
		display:inline-block;
		padding:0 10px;
		background:#F2F6FF;	
		width: auto;
	}
	#cpe_setting_ctn .form-suffix .text ,.cpeMonitorSettingBoxDialogCls .form-suffix .text{ 
		font-size:12px;
		color:#333333;
		width: auto;
		line-height: 26px;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	#cpe_setting_ctn .ipTable thead tr th:first-child .cell{
		display: none;
	}
	.ipModalWarp .el-dialog__body{
		padding: 30px;
		position: relative;
		min-height: 130px;
	}
	.ipModalWarp .el-dialog__body .el-form .el-form-item__label{ 
		line-height: 28px;
	}
	.ipModalWarp .addErrorTip{
		color: #FA5555;
		font-size: 12px;
	}
	.ipModalWarp .editErrorTip{
		color: #FA5555;
		font-size: 12px;
		margin-left: 50px;
		margin-top: 4px;
	}
	.ipModalWarp .el-form-item__error{
		left: 0;
	}
	.cpeMonitorSettingBoxDialogCls .ipAddressWarp{
		margin-left: 50px;
		height: auto;
		overflow: auto;
		margin-bottom:40px;
	}
	.cpeMonitorSettingBoxDialogCls .ipWarpFotter{
		position: absolute;
		bottom: 25px;
	}
	#cpe_setting_ctn .ipTip{
		font-size: 15px;
		margin-left: 20px; 
		color: #4D84FF;
	}
	#cpe_setting_ctn .iplistTitle{
		position: relative;
		height: 40px;
		line-height: 40px;
	}
	#cpe_setting_ctn .systemFormWarp .el-form-item{
		margin-bottom: 22px;
	}
   	#cpe_setting_ctn .lteFormWarp .el-form-item__error {
		left: 0px;
	}
	#cpe_setting_ctn .lteFormWarp .el-form-item{
		margin-bottom:30px;
	}
	.cpeMonitorSettingBoxDialogCls .el-dialog__body .el-form-item__error {
		left: 0px;
	}
	#cpe_setting_ctn .addErrorTip , .cpeMonitorSettingBoxDialogCls .addErrorTip{
		color: #FA5555;
		font-size: 12px;
		margin-left: 20px;
	 }
	 
	 #cpe_setting_ctn .earfcnPciWarp {
		margin-left: 100px;
		height: auto;
		overflow: auto;
		margin-bottom:30px;
		display: flex;
		flex-wrap: wrap;
	}
	#cpe_setting_ctn .earfcnPciWarp .el-form-item .el-form-item__content {
		margin-left: 0 !important;
	}
	#cpe_setting_ctn .half-item , .cpeMonitorSettingBoxDialogCls .half-item{
		display: flex;
		flex-wrap: wrap;
	}
	#cpe_setting_ctn .half-item .el-form-item , .cpeMonitorSettingBoxDialogCls .half-item .el-form-item{
		flex: 1 1 40%;
		margin-right: 40px;
	}
</style> 
<div id="cpe_setting_ctn" class="flex-ctn container" style="min-width: 1000px;display:flex; flex-direction:column; height:100%;border:1px solid #d5dcec;border-radius:10px;">
	<el-card class="group-setting" :footer="false">
		
		<!-- 导航区域 -->
		<div class="nav-tabs" v-show="false">
			<div v-for="(item,index) in tabs" v-show="item.show" :class="{active: activeCode == item.code, disabled: (!isOnline && index>0)}" @click="tabClick(item)">{{item.text}}</div>
		</div>
		<!-- 右侧详情区域 -->
		<div class="nav-main">
			
			<!-- 内容载入区域 -->
			<div class="nav-flex-item ">
				<!-- Basic 设置 -->
				<el-form ref="basic" v-show="activeCode=='basic'" class="nav-content"
					:model="basicForm" 
					:rules="basicRules">

					<div class="group" label="<%=rb.getString("JiBenXinXi")%>">
						<el-form-item label="<%=rb.getString("CPEName")%>" prop="cpeName">
							<el-input v-model="basicForm.cpeName" placeholder="<%=rb.getString("SheBeiMingChengGuiZe")%>"></el-input>
						</el-form-item>
						<!-- 一个input时会导致 回车提交 -->
						<el-form-item v-show="false">
							<el-input ></el-input>
						</el-form-item>
					</div>
				</el-form>

				<!-- Network 设置 -->
				<el-form ref="network" v-show="activeCode=='network'" label-width="150px" class="nav-content"
					:model="networkForm" 
					:rules="networkRules">
					<!-- 执行修改后，OMC与 CPE交互较慢，导致用户造成错觉，页面提示 -->
					<div v-show="!is43XAP" style='padding: 0 0 16px;' class='commonNotes12'>
						<i class="el-icon el-icon-circle-info" style='margin-right: 6px;'></i><%=rb.getString("CaoZuoKeHuTiShi")%>
					</div>
					<!-- Remote Web -->
					<div v-show="!is43XAP" class="group" label="<%=rb.getString("WanKouDengLuSheZhi")%>" v-if="false">
						<el-form-item label="<%=rb.getString("WanKouDengLu")%>" prop="httpsEnable">
							<el-switch v-model="networkForm.httpsEnable" :disabled="!isOnline"
								active-color="#13ce66"
								active-value="True"
								inactive-value="False"></el-switch>
						</el-form-item>
					</div>
					<!-- LAN -->
					<div v-show="isLANEnable && !isR005" class="group" label="LAN">
						<el-form-item label="LAN <%=rb.getString("JieKou")%>" prop="lanEnable">
							<el-switch v-model="networkForm.lanEnable" 
								active-color="#13ce66"
								active-value="1"
								inactive-value="0"></el-switch>
						</el-form-item>
					</div>
					<!-- DMZ -->
					<div v-show="!isR005 && !is43XAP" class="group" label="DMZ <%=rb.getString("SheZhi")%>">
						<div style="position: absolute; left: 170px;top: 0px;">
							<i @click="refreshDMZ" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
							<i v-if="dmzTips" class="el-icon el-icon-circle-info" style="font-size: 15px;margin-left: 20px; color: #4D84FF;"> {{dmzTips}}</i>
						</div>

						<div :class="{loading: isDmzLoading, readonly: ['NotSupport','NotSync'].includes(networkForm.dmzEnable)}">
							<el-form-item label="DMZ Enable" prop="dmzEnable">
								<el-switch v-model="networkForm.dmzEnable" 
									active-color="#13ce66"
									active-value="1"
									inactive-value="0"></el-switch>
							</el-form-item>

							<el-form-item label="DMZ Address" prop="dmzHostAddress">
								<el-input v-model="networkForm.dmzHostAddress" maxlength="100"></el-input>
							</el-form-item>
						</div>
					</div>

					<!-- LAN DNS -->
					<div v-show="!is43XAP" class="group" label="LAN DNS <%=rb.getString("SheZhi")%>">
						<div style="position: absolute; left: 170px;top: 0px;">
							<i @click="refreshLanDns" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
							<i v-if="lanDnsTips" class="el-icon el-icon-circle-info" style="font-size: 15px;margin-left: 20px; color: #4D84FF;"> {{lanDnsTips}}</i>
						</div>

						<div :class="{loading: isLanDnsLoading, readonly: ['NotSupport','NotSync'].includes(networkForm.lanDnsEnable)}">
							<el-form-item label="DNS Option" prop="lanDnsMode">
								<el-radio-group v-model="networkForm.lanDnsMode">
									<el-radio label="1">Auto</el-radio>
									<el-radio label="0">Manual</el-radio>
								</el-radio-group>
							</el-form-item>

							<el-form-item v-if="networkForm.lanDnsMode == '0'" label="First DNS" prop="lanDns1Address"
								:rules="{
									validator: function(rule, val, cb) {
										if( val && isValidIP(val) ) {
											cb();
										}else if(val){
											cb('Invalid DNS');
										}else {
											if(isLanDnsHas()) {
												cb();
											}else {
												cb('Must have At least one DNS');
											}
										}
									}
								}">
								<el-input v-model="networkForm.lanDns1Address" maxlength="100"></el-input>
							</el-form-item>
							<el-form-item v-if="networkForm.lanDnsMode == '0'" label="Secondary DNS" prop="lanDns2Address"
								:rules="{
									validator: function(rule, val, cb) {
										if( val && isValidIP(val) ) {
											cb();
										}else if(val){
											cb('Invalid DNS');
										}else {
											if(isLanDnsHas()) {
												cb();
											}else {
												cb('Must have At least one DNS');
											}
										}
									}
								}">
								<el-input v-model="networkForm.lanDns2Address" maxlength="100"></el-input>
							</el-form-item>
							<el-form-item v-if="networkForm.lanDnsMode == '0'" label="Thirdly DNS" prop="lanDns3Address"
								:rules="{
									validator: function(rule, val, cb) {
										if( val && isValidIP(val) ) {
											cb();
										}else if(val){
											cb('Invalid DNS');
										}else {
											if(isLanDnsHas()) {
												cb();
											}else {
												cb('Must have At least one DNS');
											}
										}
									}
								}">
								<el-input v-model="networkForm.lanDns3Address" maxlength="100"></el-input>
							</el-form-item>
						</div>
					</div>

					<!-- LAN Host -->
					<div v-show="!is43XAP" class="group" label="LAN Host <%=rb.getString("SheZhi")%>">
						<div style="position: absolute; left: 170px;top: 0px;">
							<i @click="refreshLanHost" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
							<i v-if="lanHostTips" class="el-icon el-icon-circle-info" style="font-size: 15px;margin-left: 20px; color: #4D84FF;"> {{lanHostTips}}</i>
						</div>

						<div :class="{loading: isLanHostLoading, readonly: ['NotSupport','NotSync'].includes(networkForm.lanHostEnable)}">
							<el-form-item label="IP Address" prop="lanIp">
								<el-input v-model="networkForm.lanIp"></el-input>
							</el-form-item>
						</div>
					</div>

					<!-- WAN DNS -->
					<div v-show="!isR005 && !is43XAP" class="group" label="WAN DNS <%=rb.getString("SheZhi")%>">
						<div style="position: absolute; left: 170px;top: 0px;">
							<i @click="refreshWanDns" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
							<i v-if="wanDnsTips" class="el-icon el-icon-circle-info" style="font-size: 15px;margin-left: 20px; color: #4D84FF;"> {{wanDnsTips}}</i>
						</div>

						<div :class="{loading: isWanDnsLoading, readonly: ['NotSupport','NotSync'].includes(networkForm.wanDnsEnable)}">
							<el-form-item label="DNS Mode" prop="wanDnsMode">
								<el-radio-group v-model="networkForm.wanDnsMode">
									<el-radio label="1">Auto</el-radio>
									<el-radio label="0">Manual</el-radio>
								</el-radio-group>
							</el-form-item>

							<el-form-item v-if="networkForm.wanDnsMode == '0'" label="Primary DNS" prop="wanDnsPriAddress"
								:rules="{
									validator: function(rule, val, cb) {
										if( val && isValidIP(val) ) {
											cb();
										}else if(val){
											cb('Invalid DNS');
										}else {
											if(networkForm.wanDnsSecAddress) {
												cb();
											}else {
												cb('Must have At least one DNS');
											}
										}
									}
								}">
								<el-input v-model="networkForm.wanDnsPriAddress" maxlength="100"></el-input>
							</el-form-item>
							<el-form-item v-if="networkForm.wanDnsMode == '0'" label="Secondary DNS" prop="wanDnsSecAddress"
								:rules="{
									validator: function(rule, val, cb) {
										if( val && isValidIP(val) ) {
											cb();
										}else if(val){
											cb('Invalid DNS');
										}else {
											if(networkForm.wanDnsPriAddress) {
												cb();
											}else {
												cb('Must have At least one DNS');
											}
										}
									}
								}">
								<el-input v-model="networkForm.wanDnsSecAddress" maxlength="100"></el-input>
							</el-form-item>
						</div>
					</div>

					<!-- WLAN Network -->
					<div v-show="!isR005 && !is43XAP" class="group" label="WLAN Network">
						<div style="position: absolute; left: 170px;top: 0px;">
							<i @click="refreshWifi" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
						</div>

						<div style="display: none;">
							<el-form-item prop="wifiSsid"></el-form-item>
							<el-form-item prop="wifiHidessid"></el-form-item>
							<el-form-item prop="wifiApisolate"></el-form-item>
							<el-form-item prop="wifiEncryption"></el-form-item>
							<el-form-item prop="wifiPassphrase"></el-form-item>

							<el-form-item prop="wifi1Enable"></el-form-item>
							<el-form-item prop="wifi1Ssid"></el-form-item>
							<el-form-item prop="wifi1Hidessid"></el-form-item>
							<el-form-item prop="wifi1Apisolate"></el-form-item>
							<el-form-item prop="wifi1Encryption"></el-form-item>
							<el-form-item prop="wifi1Passphrase"></el-form-item>

							<el-form-item prop="wifi2Enable"></el-form-item>
							<el-form-item prop="wifi2Ssid"></el-form-item>
							<el-form-item prop="wifi2Hidessid"></el-form-item>
							<el-form-item prop="wifi2Apisolate"></el-form-item>
							<el-form-item prop="wifi2Encryption"></el-form-item>
							<el-form-item prop="wifi2Passphrase"></el-form-item>

							<el-form-item prop="wifi3Enable"></el-form-item>
							<el-form-item prop="wifi3Ssid"></el-form-item>
							<el-form-item prop="wifi3Hidessid"></el-form-item>
							<el-form-item prop="wifi3Apisolate"></el-form-item>
							<el-form-item prop="wifi3Encryption"></el-form-item>
							<el-form-item prop="wifi3Passphrase"></el-form-item>
						</div>

						<div :class="{loading: isWifiLoading, readonly: ['NotSupport','NotSync'].includes(networkForm.wlanEnable)}">
							<el-form-item label="WiFi" prop="wlanEnable">
								<el-switch v-model="networkForm.wlanEnable" :disabled="isWifiNotSupport"
									@change="wlanEnableChange"
									active-color="#13ce66"
									active-value="1"
									inactive-value="0"></el-switch>
									
								<i v-if="wifiTips" class="el-icon el-icon-circle-info" style="font-size: 15px;margin-left: 20px; color: #4D84FF;"> {{wifiTips}}</i>
							</el-form-item>
							<div style="display: flex;">
								<el-form-item v-show="false" label="Network Mode" prop="wifiMode">
									<el-select v-model="networkForm.wifiMode" :disabled="isWifiClosed || isWifiNotSupport">
										<el-option value="11b">11b</el-option>
										<el-option value="11g">11g</el-option>
										<el-option value="11bg">11bg</el-option>
										<el-option value="11bgn">11bgn</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="Frequency(Channel)" prop="wifiChannel">
									<el-select v-model="networkForm.wifiChannel" :disabled="isWifiClosed || isWifiNotSupport" size="mini">
										<el-option label="AUTO" value="AUTO"></el-option>
										<el-option value="1">2.412GHz(Channel 1)</el-option>
										<el-option value="2">2.417GHz(Channel 2)</el-option>
										<el-option value="3">2.422GHz(Channel 3)</el-option>
										<el-option value="4">2.427GHz(Channel 4)</el-option>
										<el-option value="5">2.43GHz(Channel 5)</el-option>
										<el-option value="6">2.437GHz(Channel 6)</el-option>
										<el-option value="7">2.442GHz(Channel 7)</el-option>
										<el-option value="8">2.447GHz(Channel 8)</el-option>
										<el-option value="9">2.452GHz(Channel 9)</el-option>
										<el-option value="10">2.457GHz(Channel 10)</el-option>
										<el-option value="11">2.462GHz(Channel 11)</el-option>
									</el-select>
								</el-form-item>
							</div>
							<div style="display: flex;">
								<el-form-item v-show="false" label="MCS" prop="wifiRate">
									<el-select v-model="networkForm.wifiRate" :disabled="isWifiClosed || isWifiNotSupport">
										<el-option value="auto">AUTO</el-option>
										<el-option value="1">1</el-option>
										<el-option value="2">2</el-option>
										<el-option value="5.5">5.5</el-option>
										<el-option value="6">6</el-option>
										<el-option value="9">9</el-option>
										<el-option value="11">11</el-option>
										<el-option value="12">12</el-option>
										<el-option value="18">18</el-option>
										<el-option value="24">24</el-option>
										<el-option value="36">36</el-option>
										<el-option value="48">48</el-option>
										<el-option value="54">54</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="Channel Bandwidth" prop="wifiBandwidth">
									<el-radio-group v-model="networkForm.wifiBandwidth" style="margin-top: 6px;" :disabled="isWifiClosed || isWifiNotSupport">
										<el-radio label="0">20M</el-radio>
										<el-radio label="1">20/40M</el-radio>
									</el-radio-group>
								</el-form-item>
							</div>
							<div>MBSSID</div>
							<el-ctable height="200" :data="tbData" :pagination="false">
								<el-table-column label="Network Name(SSID)" prop="wifiSsid"></el-table-column>
								<el-table-column label="Security Mode" prop="wifiEncryption">
									<template slot-scope="scope">
										{{modeKeys[scope.row.wifiEncryption]}}
									</template>
								</el-table-column>
								<el-table-column label="Status" prop="wifiEnable">
									<template slot-scope="scope">
										<span v-if="scope.row.wifiEnable=='1'">Enable</span>
										<span v-if="scope.row.wifiEnable=='0'">Disable</span>
									</template>
								</el-table-column>
								<el-table-column width="150">
									<template slot-scope="scope">
										<i class="el-icon el-icon-operation-edit" @click="modifyWifi(scope.row)" v-show="!isWifiClosed&&!isWifiNotSupport"></i>
										<i class="el-icon el-icon-operation-view" @click="viewWifi(scope.row)" style="margin-left: 15px;"></i>
									</template>
								</el-table-column>
							</el-ctable>
						</div>

						<!-- wifi5 -->
						<div style="display: none;">
							<el-form-item prop="wifi5WifiSsid"></el-form-item>
							<el-form-item prop="wifi5WifiHidessid"></el-form-item>
							<el-form-item prop="wifi5WifiApisolate"></el-form-item>
							<el-form-item prop="wifi5WifiEncryption"></el-form-item>
							<el-form-item prop="wifi5WifiPassphrase"></el-form-item>

							<el-form-item prop="wifi5Wifi1Enable"></el-form-item>
							<el-form-item prop="wifi5Wifi1Ssid"></el-form-item>
							<el-form-item prop="wifi5Wifi1Hidessid"></el-form-item>
							<el-form-item prop="wifi5Wifi1Apisolate"></el-form-item>
							<el-form-item prop="wifi5Wifi1Encryption"></el-form-item>
							<el-form-item prop="wifi5Wifi1Passphrase"></el-form-item>

							<el-form-item prop="wifi5Wifi2Enable"></el-form-item>
							<el-form-item prop="wifi5Wifi2Ssid"></el-form-item>
							<el-form-item prop="wifi5Wifi2Hidessid"></el-form-item>
							<el-form-item prop="wifi5Wifi2Apisolate"></el-form-item>
							<el-form-item prop="wifi5Wifi2Encryption"></el-form-item>
							<el-form-item prop="wifi5Wifi2Passphrase"></el-form-item>

							<el-form-item prop="wifi5Wifi3Enable"></el-form-item>
							<el-form-item prop="wifi5Wifi3Ssid"></el-form-item>
							<el-form-item prop="wifi5Wifi3Hidessid"></el-form-item>
							<el-form-item prop="wifi5Wifi3Apisolate"></el-form-item>
							<el-form-item prop="wifi5Wifi3Encryption"></el-form-item>
							<el-form-item prop="wifi5Wifi3Passphrase"></el-form-item>
						</div>

						<div :class="{loading: isWifiLoading, readonly: ['NotSupport','NotSync'].includes(networkForm.wifi5WlanEnable)}">
							<el-form-item label="WiFi5" prop="wifi5WlanEnable">
								<el-switch v-model="networkForm.wifi5WlanEnable"
									:disabled="isWifi5NotSupport"
									@change="wifi5EnableChange"
									active-color="#13ce66"
									active-value="1"
									inactive-value="0"></el-switch>

								<i v-if="wifi5Tips" class="el-icon el-icon-circle-info" style="font-size: 15px;margin-left: 20px; color: #4D84FF;"> {{wifi5Tips}}</i>
							</el-form-item>
							<div style="display: flex;">
								<el-form-item label="Network Mode" prop="wifi5WifiMode" :disabled="isWifi5NotSupport">
									<el-select v-model="networkForm.wifi5WifiMode">
										<el-option label="11a" value="AONLY"></el-option>
										<el-option label="11a/n" value="AN"></el-option>
										<el-option label="11a/n/ac" value="A_AN_AC"></el-option>
										<el-option label="11an/ac" value="AN_AC"></el-option>
										<el-option label="11a/n/ac/ax" value="A_AN_AC_AX"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="Channel" prop="wifi5WifiChannel" style="margin-left: 205px;" :disabled="isWifi5NotSupport">
									<el-select v-model="networkForm.wifi5WifiChannel" size="mini">
										<el-option label="AUTO" value="AUTO"></el-option>
										<el-option v-for="item in wifi5Chanel[networkForm.wifi5WifiSupportChannel]" :label="item" :value="item"></el-option>
									</el-select>
								</el-form-item>
							</div>
							<div style="display: flex;">
								<el-form-item label="Channel Bandwidth" prop="wifi5WifiBandwidth" :disabled="isWifi5NotSupport">
									<el-radio-group v-model="networkForm.wifi5WifiBandwidth" style="margin-top: 6px;">
										<el-radio label="0">20M</el-radio>
										<el-radio label="1">40M</el-radio>
										<el-radio label="2">80M</el-radio>
										<el-radio label="3">160M</el-radio>
									</el-radio-group>
								</el-form-item>
								<el-form-item label="Support Channel" prop="supportChannel" style="margin-left: 100px;" :disabled="isWifi5NotSupport">
									<el-select v-model="networkForm.wifi5WifiSupportChannel" size="mini" @change="function(val){networkForm.wifi5WifiChannel = '';}">
										<el-option value="SC1" label="Ch36~48"></el-option>
										<el-option value="SC2" label="Ch36~64"></el-option>
										<el-option value="SC3" label="Ch52~64"></el-option>
										<el-option value="SC4" label="Ch149~161"></el-option>
										<el-option value="SC5" label="Ch149~165"></el-option>
										<el-option value="SC6" label="Ch36~48,Ch149~161"></el-option>
										<el-option value="SC7" label="Ch36~48,Ch149~165"></el-option>
										<el-option value="SC8" label="Ch36~64,Ch100~140"></el-option>
										<el-option value="SC9" label="Ch36~64,Ch149~161"></el-option>
										<el-option value="SC10" label="Ch52~64,Ch149~161"></el-option>
										<el-option value="SC11" label="Ch52~64,Ch149~165"></el-option>
										<el-option value="SC12" label="Ch36~64,Ch100~120,Ch149~161"></el-option>
										<el-option value="SC13" label="Ch36~64,Ch100~116,Ch132~140"></el-option>
										<el-option value="SC14" label="Ch36~64,Ch100~124,Ch149~161"></el-option>
										<el-option value="SC15" label="Ch36~64,Ch100~140,Ch149~161"></el-option>
										<el-option value="SC16" label="Ch36~64,Ch100~140,Ch149~165"></el-option>
										<el-option value="SC17" label="Ch52~64,Ch100~140,Ch149~161"></el-option>
										<el-option value="SC18" label="Ch56~64,Ch100~140,Ch149~161"></el-option>
										<el-option value="SC19" label="Ch36~64,Ch100~116,Ch132~140,Ch149~165"></el-option>
										<el-option value="SC20" label="Ch36~64,Ch100~116,Ch136~140,Ch149~165"></el-option>
									</el-select>
								</el-form-item>
							</div>
							<div>MBSSID</div>
							<el-ctable height="200" :data="tbDataWifi5" :pagination="false">
								<el-table-column label="Network Name(SSID)" prop="wifi5WifiSsid"></el-table-column>
								<el-table-column label="Security Mode" prop="wifi5WifiEncryption">
									<template slot-scope="scope">
										{{modeKeys[scope.row.wifi5WifiEncryption]}}
									</template>
								</el-table-column>
								<el-table-column label="Status" prop="wifi5WifiEnable">
									<template slot-scope="scope">
										<span v-if="scope.row.wifi5WifiEnable=='1'">Enable</span>
										<span v-if="scope.row.wifi5WifiEnable=='0'">Disable</span>
									</template>
								</el-table-column>
								<el-table-column width="150">
									<template slot-scope="scope">
										<i class="el-icon el-icon-operation-edit" @click="modifyWifi5(scope.row)" v-show="!isWifi5NotSupport"></i>
										<i class="el-icon el-icon-operation-view" @click="viewWifi5(scope.row)" style="margin-left: 15px;"></i>
									</template>
								</el-table-column>
							</el-ctable>
						</div>
					</div>

					<!-- Wi-Fi-->
					<div v-show="is43XAP" class="group" style="height: 40px;">
						<div style="position: absolute; left: 35px;top: 0px;z-index: 100;">
							<i @click="refreshAPWiFi" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
							<i v-if="apwifiTips" class="el-icon el-icon-circle-info" style="font-size: 15px;margin-left: 20px; color: #4D84FF;"> {{apwifiTips}}</i>
						</div>
					</div>
					<div :class="{loading: isAPWifiLoading, readonly: ['NotSupport','NotSync'].includes(networkForm.apWlanEnable)}">
						<div v-show="is43XAP" class="group" label="Master 2.4G SSID">
							<div style="display: flex;flex-wrap:wrap;justify-content: space-between;">
								<el-form-item label="SSID" prop="ap4MasterSsid">
									<el-input v-model="networkForm.ap4MasterSsid" maxlength="32"></el-input>
								</el-form-item>

								<el-form-item label="Encryption" prop="ap4MasterEncryption">
									<el-select v-model="networkForm.ap4MasterEncryption">
										<el-option label="OPEN" value="OPEN"></el-option>
										<el-option label="WPA(AES)-PSK" value="WPA"></el-option>
										<el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
										<el-option label="WPAWPA2(AES)-PSK" value="WPAWPA2"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="PASSWORD" prop="ap4MasterPassPhrase">
									<el-input v-model="networkForm.ap4MasterPassPhrase" maxlength="64"></el-input>
								</el-form-item>
							</div>
						</div>
						<div v-show="is43XAP" class="group" label="Master 5G SSID">
							<div style="display: flex;flex-wrap:wrap;justify-content: space-between;">
								<el-form-item label="SSID" prop="ap5MasterSsid">
									<el-input v-model="networkForm.ap5MasterSsid" maxlength="32"></el-input>
								</el-form-item>

								<el-form-item label="Encryption" prop="ap5MasterEncryption">
									<el-select v-model="networkForm.ap5MasterEncryption">
										<el-option label="WPA(AES)-PSK" value="WPA"></el-option>
										<el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="PASSWORD" prop="ap5MasterPassPhrase">
									<el-input v-model="networkForm.ap5MasterPassPhrase" maxlength="64"></el-input>
								</el-form-item>
							</div>
						</div>
						<div v-show="is43XAP" class="group" label="Master 6G SSID">
							<div style="display: flex;flex-wrap:wrap;justify-content: space-between;">
								<el-form-item label="SSID" prop="ap6MasterSsid">
									<el-input v-model="networkForm.ap6MasterSsid" maxlength="32"></el-input>
								</el-form-item>

								<el-form-item label="Encryption" prop="ap6MasterEncryption">
									<el-select v-model="networkForm.ap6MasterEncryption">
										<el-option label="WPA(AES)-PSK" value="WPA"></el-option>
										<el-option label="WPA2(AES)-PSK" value="WPA2"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="PASSWORD" prop="ap6MasterPassPhrase">
									<el-input v-model="networkForm.ap6MasterPassPhrase" maxlength="64"></el-input>
								</el-form-item>
							</div>
						</div>
						<div v-show="is43XAP" class="group" label="Advance Settings">
							<div style="display: flex;flex-wrap:wrap;justify-content: space-between;">
								<el-form-item label="2.4G Channel" prop="ap4Channel">
									<el-select v-model="networkForm.ap4Channel">
										<el-option v-for="item in channelList2G" :label="item?item:'Auto'" :value="item"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="2.4G Bandwidth" prop="ap4BandWidth">
									<el-select v-model="networkForm.ap4BandWidth">
										<el-option label="20MHz" value="0"></el-option>
										<el-option label="20MHz/40MHz" value="1"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="5G Channel" prop="ap5Channel">
									<el-select v-model="networkForm.ap5Channel">
										<el-option v-for="item in channelList5G" :label="item?item:'Auto'" :value="item"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="5G Bandwidth" prop="ap5BandWidth">
									<el-select v-model="networkForm.ap5BandWidth">
										<el-option label="20MHz" value="0"></el-option>
										<el-option label="20MHz/40MHz" value="1"></el-option>
										<el-option label="20MHz/40MHz/80MHz" value="2"></el-option>
										<el-option label="20MHz/40MHz/80MHz/160MHz" value="3"></el-option>
										<el-option label="20MHz/40MHz/80MHz/160MHz/320MHz" value="4"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="6G Channel" prop="ap6Channel">
									<el-select v-model="networkForm.ap6Channel">
										<el-option v-for="item in channelList6G" :label="item?item:'Auto'" :value="item"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="6G Bandwidth" prop="ap6BandWidth">
									<el-select v-model="networkForm.ap6BandWidth">
										<el-option label="20MHz" value="0"></el-option>
										<el-option label="20MHz/40MHz" value="1"></el-option>
										<el-option label="20MHz/40MHz/80MHz" value="2"></el-option>
										<el-option label="20MHz/40MHz/80MHz/160MHz" value="3"></el-option>
										<el-option label="20MHz/40MHz/80MHz/160MHz/320MHz" value="4"></el-option>
									</el-select>
								</el-form-item>

								<el-form-item label="Country" prop="country">
									<el-select v-model="networkForm.country">
										<el-option value="AE" label="UAE" ></el-option>
										<el-option value="AL" label="Albania" ></el-option>
										<el-option value="AM" label="ARMENIA" ></el-option>
										<el-option value="AR" label="Armenia" ></el-option>
										<el-option value="AT" label="Austria" ></el-option>
										<el-option value="AU" label="Australia" ></el-option>
										<el-option value="AZ" label="Azerbaijan" ></el-option>
										<el-option value="BE" label="Belgium" ></el-option>
										<el-option value="BG" label="Bulgaria" ></el-option>
										<el-option value="BH" label="Bahrain" ></el-option>
										<el-option value="BN" label="BruneiDarussalam" ></el-option>
										<el-option value="BO" label="Bolivia" ></el-option>
										<el-option value="BR" label="Brazil" ></el-option>
										<el-option value="BY" label="Belarus" ></el-option>
										<el-option value="BZ" label="Belize" ></el-option>
										<el-option value="CA" label="Canada" ></el-option>
										<el-option value="CH" label="Switzerland" ></el-option>
										<el-option value="CL" label="Chile" ></el-option>
										<el-option value="CN" label="China" ></el-option>
										<el-option value="CO" label="Colombia" ></el-option>
										<el-option value="CR" label="CostaRica" ></el-option>
										<el-option value="CY" label="Cyprus" ></el-option>
										<el-option value="CZ" label="Czech" ></el-option>
										<el-option value="DE" label="Germany" ></el-option>
										<el-option value="DK" label="Denmark" ></el-option>
										<el-option value="DO" label="DominicanRepublic" ></el-option>
										<el-option value="DZ" label="Algeria" ></el-option>
										<el-option value="EC" label="Ecuador" ></el-option>
										<el-option value="EE" label="Estonia" ></el-option>
										<el-option value="EG" label="Egypt" ></el-option>
										<el-option value="ES" label="Spain" ></el-option>
										<el-option value="FI" label="Finland" ></el-option>
										<el-option value="FO" label="FaroeIslands" ></el-option>
										<el-option value="FR" label="France" ></el-option>
										<el-option value="GB" label="UnitedKingdom" ></el-option>
										<el-option value="GE" label="Georgia" ></el-option>
										<el-option value="GR" label="Greece" ></el-option>
										<el-option value="GT" label="Guatemala" ></el-option>
										<el-option value="HK" label="HongKong" ></el-option>
										<el-option value="HN" label="Honduras" ></el-option>
										<el-option value="HR" label="Croatia" ></el-option>
										<el-option value="HU" label="Hungary" ></el-option>
										<el-option value="ID" label="Indonesia" ></el-option>
										<el-option value="IE" label="Ireland" ></el-option>
										<el-option value="IL" label="Israel" ></el-option>
										<el-option value="IN" label="India" ></el-option>
										<el-option value="IQ" label="Iraq" ></el-option>
										<el-option value="IS" label="Iceland" ></el-option>
										<el-option value="IT" label="Italy" ></el-option>
										<el-option value="JM" label="Jamaica" ></el-option>
										<el-option value="JO" label="Jordan" ></el-option>
										<el-option value="JP" label="Japan" ></el-option>
										<el-option value="KE" label="Kenya" ></el-option>
										<el-option value="KR" label="Korea" ></el-option>
										<el-option value="KW" label="Kuwait" ></el-option>
										<el-option value="KZ" label="Kazakhstan" ></el-option>
										<el-option value="LB" label="Lebanon" ></el-option>
										<el-option value="LI" label="Liechtenstein" ></el-option>
										<el-option value="LT" label="Lithuania" ></el-option>
										<el-option value="LU" label="Luxembourg" ></el-option>
										<el-option value="LV" label="Latvia" ></el-option>
										<el-option value="MA" label="Morocco" ></el-option>
										<el-option value="MC" label="Monaco" ></el-option>
										<el-option value="MK" label="Macedonia" ></el-option>
										<el-option value="MO" label="Macau" ></el-option>
										<el-option value="MX" label="Mexico" ></el-option>
										<el-option value="MY" label="Malaysia" ></el-option>
										<el-option value="NI" label="Nicaragua" ></el-option>
										<el-option value="NL" label="Netherlands" ></el-option>
										<el-option value="NO" label="Norway" ></el-option>
										<el-option value="NZ" label="NewZealand" ></el-option>
										<el-option value="OM" label="Oman" ></el-option>
										<el-option value="PA" label="Panama" ></el-option>
										<el-option value="PE" label="Peru" ></el-option>
										<el-option value="PH" label="Philippines" ></el-option>
										<el-option value="PK" label="Pakistan" ></el-option>
										<el-option value="PL" label="Poland" ></el-option>
										<el-option value="PR" label="PuertoRico" ></el-option>
										<el-option value="PT" label="Portugal" ></el-option>
										<el-option value="PY" label="Paraguay" ></el-option>
										<el-option value="QA" label="Qatar" ></el-option>
										<el-option value="RO" label="Romania" ></el-option>
										<el-option value="RU" label="Russia" ></el-option>
										<el-option value="SA" label="SaudiArabia" ></el-option>
										<el-option value="SE" label="Sweden" ></el-option>
										<el-option value="SG" label="Singapore" ></el-option>
										<el-option value="SI" label="Slovenia" ></el-option>
										<el-option value="SK" label="Slovakia" ></el-option>
										<el-option value="SV" label="ElSalvador" ></el-option>
										<el-option value="TH" label="Thailand" ></el-option>
										<el-option value="TN" label="Tunisia" ></el-option>
										<el-option value="TR" label="Turkey" ></el-option>
										<el-option value="TT" label="RepublicofTrinidadandTobago" ></el-option>
										<el-option value="TW" label="TaiWan" ></el-option>
										<el-option value="UA" label="Ukraine" ></el-option>
										<el-option value="US" label="UnitedStates" ></el-option>
										<el-option value="UY" label="Uruguay" ></el-option>
										<el-option value="UZ" label="Uzbekistan" ></el-option>
										<el-option value="VE" label="Venezuela" ></el-option>
										<el-option value="VN" label="Vietnam" ></el-option>
										<el-option value="YE" label="Yemen" ></el-option>
										<el-option value="ZA" label="SouthAfrica" ></el-option>
										<el-option value="ZW" label="Zimbabwe" ></el-option>
									</el-select>
								</el-form-item>
							</div>
						</div>
					</div>
				</el-form>

				<!-- LTE 设置 -->
				<el-form ref="lte" v-show="activeCode=='lte'" label-width="110px" label-position="left" class="lteFormWarp"
					:model="lteForm" 
					:rules="lteRules">
					
					<div style="padding: 30px 40px 0;">
						<div class="group" label="<%=rb.getString("SuoPin")%>">
							<el-form-item label="<%=rb.getString("SaoMiaoFangShi")%>" prop="scanMode">
								<el-select v-model="lteForm.scanMode" @change="scanModeChange">
									<el-option label="Full Band" value="fullband"></el-option>
									<el-option label="Band/Frequency Preferred" value="freqpreferred"></el-option>
									<el-option label="PCI lock" value="pcilock"></el-option>
									<el-option label="PCI Only Lock" value="pcionlylock"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item v-show="false" prop="CPE_Frequency">
								<el-input v-model="lteForm.CPE_Frequency"></el-input>
							</el-form-item>
							<el-form-item v-show="false" prop="PCI_value">
								<el-input v-model="lteForm.PCI_value"></el-input>
							</el-form-item>
							<!-- earfcn -->
							<div v-if="lteForm.scanMode=='freqpreferred'">
							
								<el-form-item class="" label='Earfcn' style="margin-bottom: 3px;">
									<div class="mainWarp">						
										<div class='newAddBtn' >
											<el-input v-model='lteForm.onlyEarfcn'></el-input>
											<i v-show="AddBtnShow" class="el-icon el-icon-plus" @click='addEarfcnBtn' style="margin-left: 10px;"></i>
										</div>							
									</div>
								</el-form-item>
								<div class="earfcnPciWarp">
									<el-form-item class='suffixItem' v-for='(domain,index) in lteForm.earfcnList' style='line-height:16px;'>
										<div class='form-suffix'>
											<span class='text'>Earfcn : {{domain}}</span>
											<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeEarfcn(domain)'></span>
										</div>
									</el-form-item> 
									<p class="addErrorTip" style="margin-left: 10px;width: 95%;">{{earfcnErrorMessage}}</p> 
								</div>	
							</div>	
							
							<!-- earfcn and pci -->
							<div v-if="lteForm.scanMode=='pcilock'">
								<el-form-item class="" label='Earfcn : PCI' style="margin-bottom: 3px;">
									<div class="mainWarp">						
										<div class='newAddBtn'>
											<el-input v-model='lteForm.earfcnStart' style='width:200px;'></el-input> — <el-input v-model='lteForm.pciEnd' style='width:200px;'></el-input>		 																
											<i v-show="AddBtnShow" class="el-icon el-icon-plus addIpBtn" @click='addPcilockBtn' style="margin-left: 10px;"></i>
										</div>							
									</div>
								</el-form-item>
								<div class="earfcnPciWarp">
									<el-form-item class='suffixItem' v-for='(domain,index) in lteForm.earfcnPciList'>
										<div class='form-suffix'>
											<div style="display: flex;">
												<span class='text'>Earfcn : {{domain.startValue}}</span>
												<span class='text' style="margin: 0 10px 0 15px;">PCI : {{domain.endValue}}</span>
												<span style='font-size:16px;margin-top:6px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeEarfcnPci(domain)'></span>															
											</div>														 
										</div>
									</el-form-item> 
									<p class="addErrorTip" style="margin-left: 10px;width: 95%;">{{earfcnPciErrorMessage}}</p>
								</div>									
							</div>

							<!-- pci -->
							<div v-if="lteForm.scanMode=='pcionlylock'">
								<el-form-item class="" label='PCI' style="margin-bottom: 3px;">
									<div class="mainWarp">						
										<div class='newAddBtn'>
											<el-input v-model='lteForm.onlyPci'></el-input>
											<i v-show="AddBtnShow" class="el-icon el-icon-plus" @click='addPciBtn' style="margin-left: 10px;"></i>
										</div>							
									</div>
								</el-form-item>
								<div class="earfcnPciWarp">
									<el-form-item class='suffixItem' v-for='(domain,index) in lteForm.pciList' style='line-height:16px;'>
										<div class='form-suffix'>
											<span class='text'>PCI : {{domain}}</span>
											<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removePci(domain)'></span>
										</div>
									</el-form-item> 
									<p class="addErrorTip">{{pciErrorMessage}}</p>
								</div>
								
							</div>
						</div>
						<!-- 5G -->
						<div v-if="support5G" class="group" label="5G <%=rb.getString("SuoPin")%>">
							<el-form-item v-show="false" prop="nrLockMode">
								<el-input v-model="lteForm.nrLockMode"></el-input>
							</el-form-item>
							<el-form-item v-show="false" prop="nrPci">
								<el-input v-model="lteForm.nrPci"></el-input>
							</el-form-item>
							<el-form-item v-show="false" prop="nrBand">
								<el-input v-model="lteForm.nrBand"></el-input>
							</el-form-item>

							<el-form-item label="<%=rb.getString("SaoMiaoFangShi")%>" prop="nrLockMode">
								<el-select v-model="lteForm.nrLockMode">
									<el-option label="Full Band" value="fullband"></el-option>
									<el-option label="Cell Lock" value="celllock"></el-option>
									<el-option label="Band Lock" value="bandlock"></el-option>
								</el-select>
							</el-form-item>
							<!-- 5G cell lock -->
							<div v-if="lteForm.nrLockMode=='celllock'">
								<el-form-item label="Cell Lock" style="margin-bottom: 5px;">
									<i class="el-icon el-icon-plus" style="margin-top: 10px;" @click="showCellLock"></i>
								</el-form-item>
								<el-table height="200px" :data="cellLockList">
									<el-table-column label="Index" type="index" width="100"></el-table-column>
									<el-table-column label="Rat" prop="rat">
										<template slot-scope="scope">
											<span v-if="scope.row.rat == '0'">LTE</span>
											<span v-if="scope.row.rat == '1'">NR</span>
										</template>
									</el-table-column>
									<el-table-column label="Band" prop="band"></el-table-column>
									<el-table-column label="Earfcn" prop="earfcn"></el-table-column>
									<el-table-column label="PCI" prop="pci"></el-table-column>
									<el-table-column label="Operation" width="100">
										<template slot-scope="scope">
											<i @click="deleteCellLock(scope.$index)" class="el-icon el-icon-operation-delete"></i>
										</template>
									</el-table-column>
								</el-table>
							</div>

							<!-- 5G Band lock -->
							<div v-if="lteForm.nrLockMode=='bandlock'">
								<el-form-item label="5G Band Lock" style="margin-bottom: 5px;">
									<i class="el-icon el-icon-plus" style="margin-top: 10px;" @click="showBandLock"></i>
								</el-form-item>
								<el-table height="200px" :data="bandLockList">
									<el-table-column label="Index" type="index" width="100"></el-table-column>
									<el-table-column label="Rat" prop="rat">
										<template slot-scope="scope">
											<span v-if="scope.row.rat == '0'">LTE</span>
											<span v-if="scope.row.rat == '1'">NR</span>
										</template>
									</el-table-column>
									<el-table-column label="Band" prop="band">
										<template slot-scope="scope">
											<span v-if="scope.row.band == '0'">Full Band</span>
											<span v-if="scope.row.band != '0'">{{scope.row.band}}</span>
										</template>
									</el-table-column>
									<el-table-column label="Operation" width="100">
										<template slot-scope="scope">
											<i @click="deleteBandLock(scope.$index)" class="el-icon el-icon-operation-delete"></i>
										</template>
									</el-table-column>
								</el-table>
							</div>
						</div>
					</div>					
					<!-- apn 提示 -->
					<div style='margin-left: 72px;'>
						<i class="el-icon el-icon-circle-info commonIcon" style='margin-right: 6px; font-size: 14px;'></i><%=rb.getString("APNPeiZhiTiShi")%>
					</div>
					<div style="padding: 30px 40px 0;">
						<div v-show="isLWAEnable" class="group" label="AP LIST">
							<el-ctable :url="apListUrl" :query-params="{cpeCode: cpeCode}" height="400">
								<el-table-column label="Serial Number" prop="serial_number"></el-table-column>
								<el-table-column label="SSID" prop="ssid"></el-table-column>
								<el-table-column label="Encryption" prop="encryption"></el-table-column>
								<el-table-column label="Key" prop="Key"></el-table-column>
							</el-ctable>
						</div>
					</div>
				</el-form>

				<!-- System 设置 -->
				<el-form ref="system" v-show="activeCode=='system'" label-width="200px" class="systemFormWarp nav-content"
					:model="systemForm" 
					:rules="systemRules">
					<!-- 执行修改后，OMC与 CPE交互较慢，导致用户造成错觉，页面提示 -->
					<div style='padding: 0 0 16px;' class='commonNotes12'>
						<i class="el-icon el-icon-circle-info" style='margin-right: 6px;'></i><%=rb.getString("CaoZuoKeHuTiShi")%>
					</div>
					<div class="group" label="Local Web Access <%=rb.getString("SheZhi")%>">
						<el-form-item label="Https<%=rb.getString("SheZhiKaiGuan")%>" prop="httpsEnable">
							<el-switch v-model="systemForm.httpsEnable"
								@change="function(val){
									systemForm.httpsWanEnable = val;
								}"					
								active-color="#13ce66"
								active-value="1"
								inactive-value="0"></el-switch>
						</el-form-item>									
					</div>
					
					<div class="group" label="WAN Access <%=rb.getString("SheZhi")%>">
						<div style="position: absolute; left: 200px;top: 0px;">
							<i @click="refreshIp" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
							<i v-if="ipTips" class="el-icon el-icon-circle-info ipTip"> {{ipTips}}</i>
						</div>

						<div :class="{loading: isIpLoading, readonly: ['NotSupport','NotSync'].includes(systemForm.wanAccessControlEnable)}">
							<el-form-item label="HttpsWan<%=rb.getString("SheZhiKaiGuan")%>" prop="httpsWanEnable">
								<el-switch v-model="systemForm.httpsWanEnable" 
									@change="function(val){
										systemForm.httpsEnable = val;
									}"
									active-color="#13ce66"
									active-value="1"
									inactive-value="0"></el-switch>
							</el-form-item>
							<el-form-item label="Access Control <%=rb.getString("SheZhiKaiGuan")%>" prop="wanAccessControlEnable">
								<el-switch v-model="systemForm.wanAccessControlEnable" 									
									active-color="#13ce66"
									active-value="1"
									inactive-value="0"></el-switch>
							</el-form-item>
							<el-form-item v-show="false" prop="accessIpList">
								<el-input v-model="systemForm.accessIpList"></el-input>
							</el-form-item>
							<div class="iplistTitle">
								<span><%=rb.getString("FangWenKongZhiLieBiao")%></span>
								<div class="circleIcon" style="top:0px;" v-show="notSupportShow" >
									<span class="el-icon el-icon-circle-add" @click="systemAddIp"></span>
									<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
								</div>
							</div>	
							<div style="border:1px solid #E9E9E9;margin-bottom:30px;" class="ipTable">
								<el-table ref="ctableIp" :data="tableData.slice((currentPage-1)*pageSize,currentPage*pageSize)" height="200" border>
									<el-table-column label="" type="index" min-width="50"></el-table-column>
									<el-table-column label='' width="70" prop="">
										<template slot-scope="scope">
											<div class="el-icon el-icon-operation-edit" title="<%=rb.getString("GengXin") %>" @click="updateIp(scope.row,scope.$index)"></div>
											<div class="el-icon el-icon-operation-delete" title="<%=rb.getString("ShanChu") %>" @click="deleteIp(scope.row,scope.$index)"></div>									
										</template>
									</el-table-column>
									
									<el-table-column label="<%=rb.getString("KaiShiIP") %>" prop="ipStart"></el-table-column>
									<el-table-column label="<%=rb.getString("JieShuIP") %>" prop="ipEnd"></el-table-column>
								</el-table>
								<el-pagination 
									@size-change="handleSizeChange"
									@current-change="handleCurrentChange"
									:current-page="currentPage"
									:page-sizes="[50,100,200]"
									:pageSize="pageSize"
									layout="total,sizes, prev, pager, next, jumper"
									:total="tableData.length">
								</el-pagination>
							</div>												
						</div>						
					</div>
					<!-- end ip -->
					
					<div class="group" label="Ping Watchdog <%=rb.getString("SheZhi")%>">
						<div style="position: absolute; left: 220px;top: 0px;">
							<i @click="refreshWatchDog" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
							<i v-if="watchDogTips" class="el-icon el-icon-circle-info" style="font-size: 15px;margin-left: 20px; color: #4D84FF;"> {{watchDogTips}}</i>
						</div>

						<div :class="{loading: isWatchDogLoading, readonly: ['NotSupport','NotSync'].includes(systemForm.watchDogEnable)}">
							<el-form-item label="Ping Watchdog <%=rb.getString("SheZhiKaiGuan")%>" prop="watchDogEnable">
								<el-switch v-model="systemForm.watchDogEnable" 
									@change="watchdogEnableChange"
									active-color="#13ce66"
									active-value="1"
									inactive-value="0"></el-switch>
							</el-form-item>
							<el-form-item label="IP Address or URL to Ping" prop="watchDogPingIp">
								<el-input v-model="systemForm.watchDogPingIp" maxlength="45" placeholder="Length: 0-45"></el-input>
							</el-form-item>
							<el-form-item label="Ping Timeout(Seconds)" prop="watchDogPingTimeout">
								<el-input v-model="systemForm.watchDogPingTimeout" maxlength="8" placeholder="Range: 1-65535"></el-input>
							</el-form-item>
							<el-form-item label="Ping Count" prop="watchDogPingCount">
								<el-input v-model="systemForm.watchDogPingCount" maxlength="8" placeholder="Range: 1-65535"></el-input>
							</el-form-item>
							<el-form-item label="Failure Count to Reboot" prop="watchDogFailureReboot">
								<el-input v-model="systemForm.watchDogFailureReboot" maxlength="8" placeholder="Range: 1-65535"></el-input>
							</el-form-item>
						</div>
					</div>
					
					<!--snmp start -->
					<div class="group" label="SNMP <%=rb.getString("SheZhi")%>">
						<div style="position: absolute; left: 200px;top: 0px;">
							<i @click="refreshSnmp" class="el-icon el-icon-circle-refresh" style="font-size: 20px;"></i>
							<i v-if="snmpTips" class="el-icon el-icon-circle-info ipTip"> {{snmpTips}}</i>
						</div>

						<div :class="{loading: isSnmpLoading, readonly: ['NotSupport','NotSync'].includes(systemForm.snmpEnable)}">
							<el-form-item label="SNMP <%=rb.getString("SheZhiKaiGuan")%>" prop="snmpEnable">
								<el-switch v-model="systemForm.snmpEnable" 
									active-color="#13ce66"
									active-value="1"
									inactive-value="0"></el-switch>
							</el-form-item>
							<el-form-item label="NMS Address" prop="nmsAddress">
								<el-input v-model="systemForm.nmsAddress" maxlength="" placeholder=""></el-input>
							</el-form-item>
							<el-form-item label="NMS Port" prop="nmsPort">
								<el-input v-model="systemForm.nmsPort" maxlength="5" placeholder=""></el-input>
							</el-form-item>
            				<el-form-item label="Listening Port" prop="listeningPort">
								<el-input v-model="systemForm.listeningPort" maxlength="5" placeholder=""></el-input>
							</el-form-item>
							<el-form-item label="Trap Community" prop="trapCommunity">
								<el-input v-model="systemForm.trapCommunity" maxlength="" placeholder=""></el-input>
							</el-form-item>
							<el-form-item label="Version" prop="version">
						    	<el-select v-model="systemForm.version">
						            <el-option label="V1&V2c" value="v2c"></el-option>
						    		<el-option label="V3" value="v3"></el-option>
						    	</el-select>
						    </el-form-item>
							<div v-if='systemForm.version == "v2c"'>
								<el-form-item label="Read Community" prop="readCommunity">
									<el-input v-model="systemForm.readCommunity" maxlength="" placeholder=""></el-input>
								</el-form-item>
								<el-form-item label="RW Community" prop="rwCommunity">
									<el-input v-model="systemForm.rwCommunity" maxlength="" placeholder=""></el-input>
								</el-form-item>
							</div>
							<div v-if='systemForm.version == "v3"'>
								<el-form-item label="User Name" prop="userName">
									<el-input v-model="systemForm.userName" maxlength="" placeholder=""></el-input>
								</el-form-item>
								<el-form-item label="Authentication Protocol" prop="authenticationProtocol">
						    		<el-select v-model="systemForm.authenticationProtocol">
						            	<el-option label="MD5" value="MD5"></el-option>
						    			<el-option label="SHA" value="SHA"></el-option>
						    		</el-select>
						   	 	</el-form-item>
								<el-form-item label="Authentication Passphrase" prop="authenticationPassphrase">
									<el-input v-model="systemForm.authenticationPassphrase" maxlength="" placeholder=""></el-input>
						    	</el-form-item>
								<el-form-item label="Privacy Protocol" prop="privacyProtocol">
									<el-select v-model="systemForm.privacyProtocol">
						           		<el-option label="DES" value="DES"></el-option>
						    			<el-option label="AES" value="AES"></el-option>
						    		</el-select>
								</el-form-item>
								<el-form-item label="Privacy Passphrase" prop="privacyPassphrase">
									<el-input v-model="systemForm.privacyPassphrase" maxlength="" placeholder=""></el-input>
								</el-form-item>
							</div>											
						</div>						
					</div>
					<!--snmp end -->
				</el-form>

				<!-- apn 设置 -->
				<div class="nav-content">
					<div id="apnSetBox" v-show="activeCode=='apnSet'"></div>
				</div>								
			</div>
			<!-- 操作区域 -->
			<div class="nav-operations">
				<el-button type="primary" @click="saveSetting"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeSetting"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</el-card>
	<!-- MBSSID修改窗口 -->
	<el-dialog title="Modify MBSSID" :visible.sync="wifiMDLShow" :modal="true" :append-to-body="true" width="600" class="cpeMonitorSettingBoxDialogCls">
		<el-form ref="modifyForm" :model="mForm" :rules="mFormRules" class="label-flex" label-width="160" style="padding-left: 20px;">
			<el-form-item v-show="false" prop="id">
				<el-input v-model="mForm.id"></el-input>
			</el-form-item>
			<el-form-item v-show="mForm.id != 'main'" label="Muti-SSID Status" prop="wifiEnable">
				<el-select v-model="mForm.wifiEnable">
					<el-option label="Enable" value="1"></el-option>
					<el-option label="Disable" value="0"></el-option>
				</el-select>
			</el-form-item>
			<div v-show="mForm.wifiEnable == '1'">
				<el-form-item label="Network Name(SSID)" prop="wifiSsid">
					<el-input v-model="mForm.wifiSsid" maxlength="100"></el-input>
				</el-form-item>
				<div v-show="false" style="padding-bottom: 20px;">
					<el-checkbox v-model="mForm.wifiHidessid" true-label="1" false-label="0">Hide SSID</el-checkbox>
					<el-form-item v-show="false" prop="wifiHidessid"></el-form-item>
				</div>
				<div v-show="false" style="padding-bottom: 25px;">
					<el-checkbox v-model="mForm.wifiApisolate" true-label="1" false-label="0">AP Lsolate</el-checkbox>
					<el-form-item v-show="false" prop="wifiApisolate"></el-form-item>
				</div>
				<el-form-item label="Security Mode" prop="wifiEncryption">
					<el-select v-model="mForm.wifiEncryption">
						<el-option label="OPEN" value="OPEN"></el-option>
						<el-option label="WPAPSK" value="WPA"></el-option>
						<el-option label="WPA2PSK" value="WPA2"></el-option>
						<el-option label="WPAPSK/WPA2PSK" value="WPAWPA2"></el-option>
					</el-select>
				</el-form-item>
				<div v-show="mForm.wifiEncryption != 'OPEN'">
					<el-form-item v-show="mForm.wifiEncryption && mForm.wifiEncryption!='OPEN' && false" label="WPA Algorithm">
						{{wpakeys[mForm.wifiEncryption]}}
					</el-form-item>
					<div style="padding-bottom: 25px;">
						<el-checkbox v-model="mForm.showPassword">Display Password</el-checkbox>
						<el-form-item v-show="false" prop="showPassword"></el-form-item>
					</div>
					<el-form-item label="Pass Phrase" prop="wifiPassphrase">
						<el-input :type="mForm.showPassword==true?'text':'password'" v-model="mForm.wifiPassphrase"></el-input>
					</el-form-item>
				</div>
			</div>
		</el-form>
		<div slot="footer">
			<el-button type="primary" @click="saveModify"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="wifiMDLShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<!-- MBSSID查看窗口 -->
	<el-dialog title="WiFi Station List" :visible.sync="viewMDLShow" :modal="true" :append-to-body="true" width="800" class="cpeMonitorSettingBoxDialogCls">
		<el-ctable :data="viewTbData" height="300" :rownumber="true" :pagination="false">
			<el-table-column label="MAC Address" prop="MACaddress"></el-table-column>
			<el-table-column label="Aid" prop="Aid"></el-table-column>
			<el-table-column label="Wireless Mode" prop="wirelessMode"></el-table-column>
			<el-table-column label="BW Sent" prop="sentBandwidth"></el-table-column>
			<el-table-column label="BW Received" prop="ReceivedBandwidth"></el-table-column>
		</el-ctable>
	</el-dialog>
	
	<!-- wifi5 MBSSID修改窗口 -->
	<el-dialog title="Modify MBSSID" :visible.sync="wifi5WifiMDLShow" :modal="true" :append-to-body="true" width="600" class="cpeMonitorSettingBoxDialogCls">
		<el-form ref="modifyWifi5Form" :model="mWifi5Form" :rules="mWifi5FormRules" class="label-flex" label-width="160" style="padding-left: 20px;">
			<el-form-item v-show="false" prop="id">
				<el-input v-model="mForm.id"></el-input>
			</el-form-item>
			<el-form-item v-show="mWifi5Form.id != 'main'" label="Muti-SSID Status" prop="wifi5WifiEnable">
				<el-select v-model="mWifi5Form.wifi5WifiEnable">
					<el-option label="Enable" value="1"></el-option>
					<el-option label="Disable" value="0"></el-option>
				</el-select>
			</el-form-item>
			<div v-show="mWifi5Form.wifi5WifiEnable == '1'">
				<el-form-item label="Network Name(SSID)" prop="wifi5WifiSsid">
					<el-input v-model="mWifi5Form.wifi5WifiSsid" maxlength="100"></el-input>
				</el-form-item>
				<div v-show="false" style="padding-bottom: 20px;">
					<el-checkbox v-model="mWifi5Form.wifi5WifiHidessid" true-label="1" false-label="0">Hide SSID</el-checkbox>
					<el-form-item v-show="false" prop="wifi5WifiHidessid"></el-form-item>
				</div>
				<div v-show="false" style="padding-bottom: 25px;">
					<el-checkbox v-model="mWifi5Form.wifi5WifiApisolate" true-label="1" false-label="0">AP Lsolate</el-checkbox>
					<el-form-item v-show="false" prop="wifi5WifiApisolate"></el-form-item>
				</div>
				<el-form-item label="Security Mode" prop="wifi5WifiEncryption">
					<el-select v-model="mWifi5Form.wifi5WifiEncryption">
						<el-option label="OPEN" value="OPEN"></el-option>
						<el-option label="WPAPSK" value="WPA"></el-option>
						<el-option label="WPA2PSK" value="WPA2"></el-option>
						<el-option label="WPAPSK/WPA2PSK" value="WPAWPA2"></el-option>
					</el-select>
				</el-form-item>
				<div v-show="mWifi5Form.wifiEncryption != 'OPEN'">
					<el-form-item v-show="mWifi5Form.wifi5WifiEncryption && mWifi5Form.wifi5WifiEncryption!='OPEN' && false" label="WPA Algorithm">
						{{wpakeys[mWifi5Form.wifi5WifiEncryption]}}
					</el-form-item>
					<div style="padding-bottom: 25px;">
						<el-checkbox v-model="mWifi5Form.wifi5ShowPassword">Display Password</el-checkbox>
						<el-form-item v-show="false" prop="wifi5ShowPassword"></el-form-item>
					</div>
					<el-form-item label="Pass Phrase" prop="wifi5WifiPassphrase">
						<el-input :type="mWifi5Form.wifi5ShowPassword==true?'text':'password'" v-model="mWifi5Form.wifi5WifiPassphrase"></el-input>
					</el-form-item>
				</div>
			</div>
		</el-form>
		<div slot="footer">
			<el-button type="primary" @click="saveModifyWifi5"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="wifi5WifiMDLShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	<!-- wifi5 MBSSID查看窗口 -->
	<el-dialog title="WiFi Station List" :visible.sync="wifi5ViewMDLShow" :modal="true" :append-to-body="true" width="800" class="cpeMonitorSettingBoxDialogCls">
		<el-ctable :data="viewTbData" height="300" :rownumber="true" :pagination="false">
			<el-table-column label="MAC Address" prop="MACaddress"></el-table-column>
			<el-table-column label="Aid" prop="Aid"></el-table-column>
			<el-table-column label="Wireless Mode" prop="wirelessMode"></el-table-column>
			<el-table-column label="BW Sent" prop="sentBandwidth"></el-table-column>
			<el-table-column label="BW Received" prop="ReceivedBandwidth"></el-table-column>
		</el-ctable>
	</el-dialog>
	
	<!-- 新建ip弹窗 -->
	<el-dialog title="<%=rb.getString("TianJia")%>" id="addIpDialog" width='600px' :visible.sync='addIpShow' :append-to-body="true" :close-on-click-modal="false" @close='closeAddIp' class="cpeMonitorSettingBoxDialogCls ipModalWarp">
		<el-form ref='addIpForm' :model='addIpForm' label-position="left">
			<el-form-item label="IP" label-width="50px"  style='margin-bottom:0px;position:relative'>
				<el-input v-model='addIpForm.ipStart' style='width:200px;'></el-input> — <el-input v-model='addIpForm.ipEnd' style='width:200px;'></el-input>		 		
				<span @click='addIpBtn' class='form-bt el-icon el-icon-plus' style='vertical-align:middle'></span>
			</el-form-item>
			<div class="ipAddressWarp">
				<el-form-item class='suffixItem' v-for='(domain,index) in addIpForm.ipGroup' style='line-height:16px;'>
					<div class='form-suffix'>
						<span class='text'>{{domain}}</span>
						<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeIp(domain)'></span>
					</div>
				</el-form-item> 
				<p class="addErrorTip">{{errorMessage}}</p>
			</div>
			<div class="ipWarpFotter">
				<el-button @click='saveAddIp' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeAddIp'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
	
	<!-- 修改ip弹窗 -->
	<el-dialog title="<%=rb.getString("XiuGai")%>" id="editIpDialog" width='600px' :visible.sync='editIpShow' :append-to-body="true" :close-on-click-modal="false" @close='closeEditIp' class="cpeMonitorSettingBoxDialogCls ipModalWarp">
		<el-form ref='editIpForm' :model='editIpForm' label-position="left">
			<el-form-item label="IP" label-width="50px"  style='margin-bottom:0px;position:relative'>
				<el-input v-model='editIpForm.ipStart' style='width:200px;'></el-input> — <el-input v-model='editIpForm.ipEnd' style='width:200px;'></el-input>		 		
			</el-form-item>
			<p class="editErrorTip">{{editErrorMsg}}</p>			
		</el-form>
		<div  class="ipWarpFotter">
			<el-button @click='saveEditIp' type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click='closeEditIp'><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
		
	<el-dialog title="Add 5G Cell Lock" :visible.sync="cell5gDlShow" :modal="true" :append-to-body="true" class="cpeMonitorSettingBoxDialogCls">
		<el-form ref="cellForm" :model="cellForm" :rules="lockRules" label-width="80" class="label-flex half-item">
			<el-form-item label="Rat" prop="rat">
				<el-select v-model="cellForm.rat">
					<el-option label="LTE" value="0"></el-option>
					<el-option label="NR" value="1"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Band" prop="band">
				<el-select v-model="cellForm.band" filterable>
					<el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Earfcn" prop="earfcn" required>
				<el-input v-model="cellForm.earfcn"></el-input>
			</el-form-item>
			<el-form-item label="PCI" prop="pci" required>
				<el-input v-model="cellForm.pci"></el-input>
			</el-form-item>
		</el-form>
		
		<div slot="footer">
			<el-button type="primary" @click="addCellLock"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="cell5gDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
		
	<el-dialog title="Add 5G Band Lock" :visible.sync="band5gDlShow" :modal="true" :append-to-body="true" class="cpeMonitorSettingBoxDialogCls">
		<el-form ref="bandForm" :model="bandForm" :rules="lockRules" label-width="80" class="label-flex half-item">
			<el-form-item label="Rat" prop="rat">
				<el-select v-model="bandForm.rat">
					<el-option label="LTE" value="0"></el-option>
					<el-option label="NR" value="1"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Band" prop="band">
				<el-select v-model="bandForm.band">
					<el-option v-for="item in bandList" :label="item?item:'Full Band'" :value="item"></el-option>
				</el-select>
			</el-form-item>
		</el-form>
		
		<div slot="footer">
			<el-button type="primary" @click="addBandLock"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="band5gDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>
<script>
	var cpe_configuration = new Vue({
		el: '#cpe_setting_ctn',
		data() {
			var vm = this,
				createChanel = function(chStr) {
					var vm = this,
						list = chStr.split(','),
						arr = [];

					list.map(function(seg){
						var range = seg.split('~');

						for(var i = range[0] - 0; i <= range[1] - 0; i += 4) {
							arr.push(i+'');
						}
					});

					return arr;
				},
				regip = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
				regstr = /^((128|192)|2(24|4[08]|5[245]))(\.(0|(128|192)|2((24)|(4[08])|(5[245])))){3}$/,
				regCode = new RegExp(regstr),
				//reg = /^(\d+,?)+$/,
				reg = /^\d{1,}$/,
				validateCpeName = function(rule,value,callback){
					if(value.length > 45){
						callback(new Error("<%=rb.getString("SheBeiMingChengGuiZe")%>"))
					}else{
						callback();
					}
				},
				/* watchdog相关校验 */
				watchdogValid = function(rule,value,cb){
					var currVal = value,
						enable = vm.systemForm.watchDogEnable;
					
					if(currVal) {
						if(isNaN(currVal) || currVal-1<0 || currVal-65535>0){
							cb('Range: 1-65535');
						}else{
							cb();
						}
					}else {
						if(enable == 1) {
							cb('Range: 1-65535');
						}else {
							cb();
						}
					}
				},
				portValid = function(rule,value,cb){
					var reg = /^(\d+)$/;
					if(vm.systemForm.snmpEnable == '1') {
						if(value && reg.test(value)) {
							cb();
						}else {
							cb('<%=rb.getString("ZhengXing")%>,<%=rb.getString("ZiFuChang")%>: 0-5');
						}
					}else {
						if(value === '' || value === null || value === undefined) {
							cb();
						}else {
							if(reg.test(value)){
								cb();
							}else{
								cb('<%=rb.getString("ZhengXing")%>,<%=rb.getString("ZiFuChang")%>: 0-5');
							}
						}
					}						
				},
				passphraseValid = function(rule,value,cb){
					if(vm.systemForm.snmpEnable == '1') {
						if(value && value.length >= 8 ) {
							cb();
						}else {
							cb('<%=rb.getString("snmpTiShi")%>');
						}
					}else {
						if(value === '' || value === null || value === undefined) {
							cb();
						}else {
							if(value.length >= 8){
								cb();
							}else{
								cb('<%=rb.getString("snmpTiShi")%>');
							}
						}
					}
				};
				/* watchdog ip校验 */
				watchdogValidateIP = function(rule,value,cb) {
					var currVal = value,
						enable = vm.systemForm.watchDogEnable;
					
					if(currVal && currVal.length>45){
						cb('<%=rb.getString("SheBeiMingChengGuiZe")%>');
					}else{
						if(!currVal && enable == 1) {
							cb('<%=rb.getString("SheBeiMingChengGuiZe")%>');
						}else {
							cb();
						}
					}
				},
				validWiFiEnable = function(rule,value,cb){
					if(value == '1' || value == '0') {
						cb();
					}else {
						cb('<%=rb.getString("QingXuanZe")%>');
					}
				},
				validSsid = function(rule,value,cb){
					if(vm.mForm.wifiEnable != '1') {
						cb();
					}else if(value) {
						cb();
					}else {
						cb('<%=rb.getString("ShuRuBiTianXiang")%>');
					}
				},
				validEncryption = function(rule,value,cb){
					if(vm.mForm.wifiEnable != '1') {
						cb();
					}else if(value) {
						cb();
					}else {
						cb('<%=rb.getString("QingXuanZe")%>');
					}
				},
				validPassphrase = function(rule,value,cb){
					if(vm.mForm.wifiEnable == '1' && vm.mForm.wifiEncryption != 'OPEN') {
						if(value && value.length >= 8 && value.length <= 64) {
							cb();
						}else {
							cb('<%=rb.getString("ZiFuChang")%>: 8 - 64');
						}
					}else {
						cb();
					}
				},
				validSsidWifi5 = function(rule,value,cb){
					if(vm.mWifi5Form.wifi5WifiEnable != '1') {
						cb();
					}else if(value) {
						cb();
					}else {
						cb('<%=rb.getString("ShuRuBiTianXiang")%>');
					}
				},
				validEncryptionWifi5 = function(rule,value,cb){
					if(vm.mWifi5Form.wifi5WifiEnable != '1') {
						cb();
					}else if(value) {
						cb();
					}else {
						cb('<%=rb.getString("QingXuanZe")%>');
					}
				},
				validPassphraseWifi5 = function(rule,value,cb){
					if(vm.mWifi5Form.wifi5WifiEnable == '1' && vm.mWifi5Form.wifi5WifiEncryption != 'OPEN') {
						if(value && value.length >= 8 && value.length <= 64) {
							cb();
						}else {
							cb('<%=rb.getString("ZiFuChang")%>: 8 - 64');
						}
					}else {
						cb();
					}
				},
				validateEarfcn = function(rule,value,cb) {
					if(value !== '') {
						if(value - 0 < 0 || value - 3279156 > 0 || isNaN(value)) {
							cb('<%=rb.getString("5GPinDianFanWei")%>');
						}else {
							cb();
						}
					}else {
						cb('Please Input Earfcn');
					}
				},
				validatePCI = function(rule,value,cb) {
					if(value !== '') {
						if(value - 0 < 0 || value - 1007 > 0 || isNaN(value)) {
							cb('<%=rb.getString("5GSpecificPCIFanWei")%>');
						}else {
							cb();
						}
					}else {
						cb('Please Input PCI');
					}
				};
			
			var list6G = ['0'];
			var i = 1;
			while(i<=233) {
				list6G.push(i);
				i = i + 4;
			}

			return {
				support5G: false,
				cellLockList: [],
				bandLockList: [],
				cell5gDlShow: false,
				band5gDlShow: false,
				cellForm: {
					rat: '',
					band: '',
					earfcn: '',
					pci: ''
				},
				bandForm: {
					rat: '',
					band: ''
				},
				lockRules: {
					rat: [{required: true, message: 'Please Select Rat'}],
					band: [{required: true, message: 'Please Select Band'}],
					earfcn: [{validator: validateEarfcn}],
					pci: [{validator: validatePCI}],
				},

				cpeSN: sessionStorage.getItem('cpeSN'),
				isOnline: false,
				cpeCode: '',
				wifi5Chanel: {
					''    : [],
					'SC1' : createChanel('36~48'),
					'SC2' : createChanel('36~64'),
					'SC3' : createChanel('52~64'),
					'SC4' : createChanel('149~161'),
					'SC5' : createChanel('149~165'),
					'SC6' : createChanel('36~48,149~161'),
					'SC7' : createChanel('36~48,149~165'),
					'SC8' : createChanel('36~64,100~140'),
					'SC9' : createChanel('36~64,149~161'),
					'SC10': createChanel('52~64,149~161'),
					'SC11': createChanel('52~64,149~165'),
					'SC12': createChanel('36~64,100~120,149~161'),
					'SC13': createChanel('36~64,100~116,132~140'),
					'SC14': createChanel('36~64,100~124,149~161'),
					'SC15': createChanel('36~64,100~140,149~161'),
					'SC16': createChanel('36~64,100~140,149~165'),
					'SC17': createChanel('52~64,100~140,149~161'),
					'SC18': createChanel('56~64,100~140,149~161'),
					'SC19': createChanel('36~64,100~116,132~140,149~165'),
					'SC20': createChanel('36~64,100~116,136~140,149~165')
				},
				basicForm: {
					cpeName: unescape('${cpeName.replaceAll("\'","%27").replaceAll("\"","%22")}')
				},
				networkForm: {
					httpsEnable: 'False',
					lanEnable: '0',
					dmzEnable: '0',
					dmzHostAddress: '',
					
					lanIp: '',

					lanDnsMode: '',
					lanDns1Address: '',
					lanDns2Address: '',
					lanDns3Address: '',
					wanDnsMode: '',
					wanDnsPriAddress: '',
					wanDnsSecAddress: '',

					wlanEnable: '0',
					wifiMode: '',
					wifiRate: '',
					wifiChannel: '',
					wifiBandwidth: '',
					
					wifiSsid: '',
					wifiHidessid: '',
					wifiApisolate: '',
					wifiEncryption: '',
					wifiPassphrase: '',

					wifi1Enable: '',
					wifi1Ssid: '',
					wifi1Hidessid: '',
					wifi1Apisolate: '',
					wifi1Encryption: '',
					wifi1Passphrase: '',
					
					wifi2Enable: '',
					wifi2Ssid: '',
					wifi2Hidessid: '',
					wifi2Apisolate: '',
					wifi2Encryption: '',
					wifi2Passphrase: '',
					
					wifi3Enable: '',
					wifi3Ssid: '',
					wifi3Hidessid: '',
					wifi3Apisolate: '',
					wifi3Encryption: '',
					wifi3Passphrase: '',

					// wifi5
					wifi5WlanEnable: '0',
					wifi5WifiMode: '',
					wifi5WifiRate: '',
					wifi5WifiChannel: '',
					wifi5WifiBandwidth: '',
					wifi5WifiSupportChannel: '',
					
					wifi5WifiSsid: '',
					wifi5WifiHidessid: '',
					wifi5WifiApisolate: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: '',

					wifi5Wifi1Enable: '',
					wifi5Wifi1Ssid: '',
					wifi5Wifi1Hidessid: '',
					wifi5Wifi1Apisolate: '',
					wifi5Wifi1Encryption: '',
					wifi5Wifi1Passphrase: '',
					
					wifi5Wifi2Enable: '',
					wifi5Wifi2Ssid: '',
					wifi5Wifi2Hidessid: '',
					wifi5Wifi2Apisolate: '',
					wifi5Wifi2Encryption: '',
					wifi5Wifi2Passphrase: '',
					
					wifi5Wifi3Enable: '',
					wifi5Wifi3Ssid: '',
					wifi5Wifi3Hidessid: '',
					wifi5Wifi3Apisolate: '',
					wifi5Wifi3Encryption: '',
					wifi5Wifi3Passphrase: '',

					apWlanEnable: '',
					ap4MasterSsid: '',
					ap4MasterEncryption: '',
					ap4MasterPassPhrase: '',
					ap5MasterSsid: '',
					ap5MasterEncryption: '',
					ap5MasterPassPhrase: '',
					ap6MasterSsid: '',
					ap6MasterEncryption: '',
					ap6MasterPassPhrase: '',
					ap4Channel: '',
					ap4BandWidth: '',
					ap5Channel: '',
					ap5BandWidth: '',
					ap6Channel: '',
					ap6BandWidth: '',
					country: ''
				},
				
                channelList2G: ['0','1','2','3','4','5','6','7','8','9','10','11','12','13','14'],
                channelList5G: ['0','36','40','44','48','52','56','60','64','100','104','108','112','116','120','124','128','132','136','140','144','149','153','157','161','165'],
                channelList6G: list6G,

				lteForm: {
					scanMode: '',
					CPE_Frequency: '',
					PCI_value: '',										
					onlyEarfcn: '',
					earfcnStart: '',
					pciEnd: '',
					onlyPci: '',
					earfcnList: [],
					earfcnPciList: [],
					pciList: [],

					nrLockMode: '',
					nrPci: '',
					nrBand: ''							                  
				},
				//add ip
				addIpShow: false,
				addIpForm:{
					ipStart: '',
					ipEnd: '',
					ipGroup: []
				},
				errorMessage: '',
				//edit ip
				editIpShow: false,
				editIpForm:{
					ipStart: '',
					ipEnd: ''
				},
				editIndex: '',
				pageSize: 50,
				currentPage: 1,
				editErrorMsg: '',
				oldEditIpStart: '',
				oldEditIpEnd: '',
				notSupportShow: false,
				//end ip

				systemForm: {
					httpsEnable: '0',
					httpsWanEnable: '0',
					wanAccessControlEnable: '0',
					accessIpList: '',
					
					watchDogEnable: '0',
					watchDogPingIp: '',
					watchDogPingTimeout: '',
					watchDogPingCount: '',
					watchDogFailureReboot: '',

					snmpEnable: '0',
					nmsAddress: '',
					nmsPort: '',
					listeningPort: '',
					trapCommunity: 'public',
					version: 'v2c',
					readCommunity: 'public',
					rwCommunity: 'private',
					userName: '',
					authenticationProtocol: 'MD5',
					authenticationPassphrase: '',
					privacyProtocol: 'DES',
					privacyPassphrase: ''
				},
				tableData: [],
				basicRules: {
					cpeName : [{validator:validateCpeName}]
				},
				networkRules: [],
				lteRules: [],
				
				systemRules: {
					watchDogPingIp:[{validator: watchdogValidateIP}],
					watchDogPingTimeout:[{validator: watchdogValid}],
					watchDogPingCount:[{validator: watchdogValid}],
					watchDogFailureReboot:[{validator: watchdogValid}],
					nmsPort:[{validator: portValid}],
					listeningPort:[{validator: portValid}],
					authenticationPassphrase:[{validator: passphraseValid}],
					privacyPassphrase:[{validator: passphraseValid}]
				},

				tbData: [],
				tbDataWifi5: [],
				activeCode: '',
				title: '<%=rb.getString("JiBenSheZhi")%>',
				isLANEnable: false,
				apListUrl: isLWAEnable?'${ctx}/cell/ap/queryCPEAPInfos.action':'',

				viewMDLShow: false,
				viewTbData: [],
				wifiMDLShow: false,
				wifi5WifiMDLShow: false,
				wifi5ViewMDLShow: false,
				mForm: {
					id: '',
					wifiEnable: '',
					wifiSsid: '',
					wifiHidessid: '',
					wifiApisolate: '',
					wifiEncryption: '',
					wifiPassphrase: '',
					showPassword: false
				},
				mWifi5Form: {
					id: '',
					wifi5WifiEnable: '',
					wifi5WifiSsid: '',
					wifi5WifiHidessid: '',
					wifi5WifiApisolate: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: '',
					wifi5ShowPassword: false
				},
				mFormRules: {
					wifiEnable: [{validator: validWiFiEnable}],
					wifiSsid: [{validator: validSsid}],
					wifiEncryption: [{validator: validEncryption}],
					wifiPassphrase: [{validator: validPassphrase}]
				},
				mWifi5FormRules: {
					wifi5WifiEnable: [{validator: validWiFiEnable}],
					wifi5WifiSsid: [{validator: validSsidWifi5}],
					wifi5WifiEncryption: [{validator: validEncryptionWifi5}],
					wifi5WifiPassphrase: [{validator: validPassphraseWifi5}]
				},
				wpakeys: {
					'OPEN': '',
					'WPA': 'TKIP',
					'WPA2': 'AES',
					'WPAWPA2': 'TKIP/AES'
				},
				modeKeys: {
					'OPEN': 'OPEN',
					'WPA': 'WPAPSK',
					'WPA2': 'WPA2PSK',
					'WPAWPA2': 'WPAPSK/WPA2PSK'
				},

				isDmzLoading: false,
				isWifiLoading: false,
				isWatchDogLoading: false,
				isSnmpLoading: false,
				isIpLoading:false,
				isLanDnsLoading: false,
				isLanHostLoading: false,
				isWanDnsLoading: false,
				isAPWifiLoading: false,
				dmzTips: '',
				wifiTips: '',
				wifi5Tips: '',
				watchDogTips: '',
				ipTips:'',
				lanDnsTips: '',
				lanHostTips: '',
				wanDnsTips: '',
				snmpTips: '',
				apwifiTips: '',
				bearTypeOnlyShow: false,
			
				earfcnErrorMessage: '',
				earfcnPciErrorMessage: '',
				pciErrorMessage: '',
				earfcnErrorMessage5G: '',
				earfcnPciErrorMessage5G: '',
				pciErrorMessage5G: '',
				earfcnTip: false,
				editBtnShow: true,
				AddBtnShow: true,
			}
		},
		computed: {
			tabs() {

				return [
					{code: 'basic', text: '<%=rb.getString("JiBenSheZhi")%>',show: true},
					{code: 'network', text: '<%=rb.getString("WangLuoSheZhi")%>',show: true},
					{code: 'lte', text: 'LTE',show: !this.wanShow},
					{code: 'system', text: '<%=rb.getString("XiTong")%>',show: true},
					{code: 'apnSet', text: 'APN/L2 <%=rb.getString("SheZhi")%>',show: this.apnSetShow && false}
				]
			},
			isWifiNotSupport() {
				return this.wifiTips == 'Not support' || this.wifiTips == 'Not sync';
			},
			isWifi5NotSupport() {
				return this.wifi5Tips == 'Not support' || this.wifi5Tips == 'Not sync';
			},
			isWifiClosed() {
				return this.networkForm.wlanEnable != '1';
			},
			wanShow() {
				var vm = this,
					reg = /\S*EP3011\S*/,
					product = sessionStorage.getItem('oldProduct');

				return reg.test(product);
			},
			apnSetShow(){
				var vm = this,
					l2flag = false,
					reg = new RegExp("^(IDU\/CN)"),
					regIduEG = new RegExp("^(IDU\/EG)"),
					regOduEG = new RegExp("^(ODU\/EG)"),
					regu4G = new RegExp("^((ODU\/u4G)|(IDU\/u4G))"),
					product = sessionStorage.getItem('oldProduct');

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

				if(l2flag && writableMap["CODE_CPE_APN"] == true){
					return true
				} else {
					return false
				}
			},
			bandList() {
				let list = [];

				for(let i=0;i<=100;i++) {
					list.push(i);
				}

				return list;
			},
			isR005() {
				var product = sessionStorage.getItem('oldProduct');

				return product.indexOf('R005') >= 0;
			},
			is43XAP() {
				var product = sessionStorage.getItem('oldProduct');

				return product.indexOf('Nova430X') >= 0 || product.indexOf('Neutrino430X') >= 0;
			}
		},
		watch: {
			cellLockList: {
				handler: function(rows) {
					var vm = this,
						list = [];

					(rows||[]).map(function(row){
						list.push(row.rat +','+ row.band +','+ row.earfcn +','+ row.pci);
					});

					vm.lteForm.nrPci = list.join(';');
				},
				deep: true
			},
			bandLockList: {
				handler: function(rows) {
					var vm = this,
						list = [];

					(rows||[]).map(function(row){
						list.push(row.rat +','+ row.band);
					});

					vm.lteForm.nrBand = list.join(';');
				},
				deep: true
			}
		},
		methods: {
			init(item,code) {
				var vm = this;
				vm.cpeCode = typeof item == 'string'?item:code;
				vm.isOnline = sessionStorage.getItem('CONNECTION_STATUS') == '1' ||  sessionStorage.getItem('CONNECTION_STATUS') == "On";
				
				$.post("${ctx}/cell/CPE/getSettingParams.action",{cpeCode: vm.cpeCode},function(data) {
					var scanMode = data["lockMode"],
						nrlockMode = data["nrLockMode"],
						earfcns = data["frequency"]?data["frequency"].split(','):[],
						nrPcis = data["nrPci"]?data["nrPci"].split(';'):[],
						nrBands = data["nrBand"]?data["nrBand"].split(';'):[],
						units = data["PCI"]?data["PCI"].split(','):[],
						pcis = data["PCI_value"]?data["PCI_value"].split(','):[];

					// basic
					if(data["cpeName"]) {
						vm.basicForm.cpeName = data["cpeName"];
					}

					// network
					vm.initNetwork(data);
					//vm.isOnline = data["connFlag"] == '1' || true;

					// lte
					vm.lteForm.scanMode = scanMode;
					vm.lteForm.nrLockMode = nrlockMode;
					vm.support5G = data["nrSupport"] == 'support';
					//new pci
					if(vm.lteForm.scanMode == 'freqpreferred'){	
						vm.lteForm.earfcnList = earfcns.map(function(item){	
							return item
						});
					}else if(vm.lteForm.scanMode == 'pcilock'){
						vm.lteForm.earfcnPciList = earfcns.map(function(item,index){
							return {
								startValue: item,
								endValue: units[index]
							}
						});
					}else if (vm.lteForm.scanMode == 'pcionlylock'){
						vm.lteForm.pciList = pcis.map(function(item){
							return item
						});
					}
					// 5G
					if(nrlockMode == 'celllock') {
						nrPcis.map(function(item){
							let arr = item.split(',');

							vm.cellLockList.push({
								rat: arr[0],
								band: arr[1],
								earfcn: arr[2],
								pci: arr[3]
							});
						});
					}else if(nrlockMode == 'bandlock') {
						nrBands.map(function(item){
							let arr = item.split(',');

							vm.bandLockList.push({
								rat: arr[0],
								band: arr[1]
							});
						});
					}
					
					vm.systemForm.httpsEnable = data['wanAccessControlHttpsEnable'];
					if(data["wanAccessControlEnable"] == 'NotSync') {
						vm.systemForm.wanAccessControlEnable = data["wanAccessControlEnable"];	
						vm.notSupportShow = false;
						vm.ipTips = 'Not Sync';
					}else if(data["wanAccessControlEnable"] == 'NotSupport') {
						vm.systemForm.wanAccessControlEnable = data["wanAccessControlEnable"];	
						vm.notSupportShow = false;
						vm.ipTips = 'Not support';
					}else {
						vm.notSupportShow = true;
						vm.ipTips = '';
						Object.assign(vm.systemForm, {
							httpsWanEnable: data['wanAccessControlHttpsWanEnable'],
							wanAccessControlEnable: data['wanAccessControlEnable'],
							accessIpList: data['wanAccessControlIpList'],
						});
					}
					
					if(data["watchDogEnable"] == 'NotSync') {
						vm.systemForm.watchDogEnable = data["watchDogEnable"];	
						vm.watchDogTips = 'Not Sync';
					}else if(data["watchDogEnable"] == 'NotSupport') {
						vm.systemForm.watchDogEnable = data["watchDogEnable"];	
						vm.watchDogTips = 'Not support';
					}else {
						vm.watchDogTips = '';
						Object.assign(vm.systemForm, {
							
							watchDogEnable: data['watchDogEnable'],						
							watchDogPingIp: data['watchDogPingIp'],
							watchDogPingTimeout: data['watchDogPingTimeout'],
							watchDogPingCount: data['watchDogPingCount'],
							watchDogFailureReboot: data['watchDogFailureReboot']
						});
					}
					
					//回显 ip 表格
					if(data["wanAccessControlEnable"] != 'NotSupport') {
						if( data['wanAccessControlIpList'] !== '' || data['wanAccessControlIpList'] !== null || data['wanAccessControlIpList'] !== undefined){
							var ipAccessList = data['wanAccessControlIpList'];
							if(ipAccessList){
								var newIpAccess = ipAccessList.split(';');
								if(newIpAccess[newIpAccess.length-1] == ""){
									newIpAccess.splice(newIpAccess.length-1)
								}

								if(newIpAccess.length >= 1){
									newIpAccess.map((item,index) => {
										var getRowData ={};
										if(item.includes('-')){
											getRowData = {
												ipStart:item.split('-')[0],
					      						ipEnd:item.split('-')[1]
					      		    		}
										}else{
											getRowData = {
												ipStart:item
						      		    	}
										}	
										vm.tableData.push(getRowData);
									})
								}
							}
						}
					}
					//snmp setting
					if(data["snmpEnable"] == 'NotSync') {
						vm.systemForm.snmpEnable = data["snmpEnable"];	
						vm.snmpTips = 'Not sync';
					}else if(data["snmpEnable"] == 'NotSupport') {
						vm.systemForm.snmpEnable = data["snmpEnable"];	
						vm.snmpTips = 'Not support';
					}else {
						vm.snmpTips = '';
						if(data["snmpEnable"]){
							Object.assign(vm.systemForm, {
								snmpEnable: data['snmpEnable'],
								nmsAddress: data['nmsAddress'],
								nmsPort: data['nmsPort'],
								listeningPort: data['listeningPort'],
								trapCommunity: data['trapCommunity'],
								version: data['version'],
								readCommunity: data['readCommunity'],
								rwCommunity: data['rwCommunity'],
								userName: data['userName'],
								authenticationProtocol: data['authenticationProtocol'],
								authenticationPassphrase: data['authenticationPassphrase'],
								privacyProtocol: data['privacyProtocol'],
								privacyPassphrase: data['privacyPassphrase'],
							});
						}
					}
					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.basic);
						initForm(vm.$refs.network);
						initForm(vm.$refs.lte);
						initForm(vm.$refs.system);
						vm.$nextTick(function(){
							vm.tabClick(item);
						});
						
					});
				},"json")
			},
			initWLAN(data) {
				var vm = this;
				// wifi
				Object.assign(vm.networkForm,{
					wlanEnable: data['wlanEnable']||'0',
					wifiMode: data['wifiMode']||'',
					wifiRate: data['wifiRate']||'',
					wifiChannel: data['wifiChannel']||'',
					wifiBandwidth: data['wifiBandwidth']||''
				});
				// main
				var listData = (data['wifiDevicelist']||'').split(';').map(function(line){
									var lineRow = {},
										props = line.split(',');
									
									lineRow['MACaddress'] = props[0];
									lineRow['Aid'] = props[1];
									lineRow['wirelessMode'] = props[2];
									lineRow['sentBandwidth'] = props[3];
									lineRow['ReceivedBandwidth'] = props[4];

									return lineRow;
								});
				vm.tbData = [];
				vm.tbData.push({
					id: 'main',
					wifiEnable: data['wlanEnable']||'',
					wifiSsid: data['wifiSsid']||'',
					wifiHidessid: data['wifiHidessid']||'',
					wifiApisolate: data['wifiApisolate']||'',
					wifiEncryption: data['wifiEncryption']||'',
					wifiPassphrase: data['wifiPassphrase']||'',
					wifiDevicelist: data['wifiDevicelist']?listData:[]
				});
				// networkForm更新
				Object.assign(vm.networkForm,{
					wifiSsid: data['wifiSsid']||'',
					wifiHidessid: data['wifiHidessid']||'',
					wifiApisolate: data['wifiApisolate']||'',
					wifiEncryption: data['wifiEncryption']||'',
					wifiPassphrase: data['wifiPassphrase']||''
				});
				// sub 
				['1','2','3'].map(function(item){
					var row = {},pre = 'wifi',suf = 'Old',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',
						hideKey = pre+item+'Hidessid',
						apKey = pre+item+'Apisolate',
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase',
						deviceKey = pre+item+'Devicelist';

					row['id'] = item;
					row['wifiEnable'] = data[enableKey]||'';
					row['wifiSsid'] = data[idKey]||'';
					row['wifiHidessid'] = data[hideKey]||'';
					row['wifiApisolate'] = data[apKey]||'';
					row['wifiEncryption'] = data[encryKey]||'';
					row['wifiPassphrase'] = data[phraKey]||'';

					var list = (data[deviceKey]||'').split(';').map(function(line){
												var lineRow = {},
													props = line.split(',');
												
												lineRow['MACaddress'] = props[0];
												lineRow['Aid'] = props[1];
												lineRow['wirelessMode'] = props[2];
												lineRow['sentBandwidth'] = props[3];
												lineRow['ReceivedBandwidth'] = props[4];

												return lineRow;
											});
					row['wifiDevicelist'] = data[deviceKey]?list:[];

					// networkForm更新
					vm.networkForm[enableKey] = data[enableKey]||'';
					vm.networkForm[idKey] = data[idKey]||'';
					vm.networkForm[hideKey] = data[hideKey]||'';
					vm.networkForm[apKey] = data[apKey]||'';
					vm.networkForm[encryKey] = data[encryKey]||'';
					vm.networkForm[phraKey] = data[phraKey]||'';

					vm.tbData.push(row);
				});
			},
			initWifi5(data) {
				var vm = this;
				// wifi
				Object.assign(vm.networkForm,{
					wifi5WlanEnable: data['wifi5WlanEnable']||'0',
					wifi5WifiMode: data['wifi5WifiMode']||'',
					wifi5WifiRate: data['wifi5WifiRate']||'',
					wifi5WifiChannel: data['wifi5WifiChannel']||'',
					wifi5WifiBandwidth: data['wifi5WifiBandwidth']||'',
					wifi5WifiSupportChannel: data['wifi5WifiSupportChannel']||''
				});
				// main
				var listData = (data['wifi5WifiDevicelist']||'').split(';').map(function(line){
									var lineRow = {},
										props = line.split(',');
									
									lineRow['MACaddress'] = props[0];
									lineRow['Aid'] = props[1];
									lineRow['wirelessMode'] = props[2];
									lineRow['sentBandwidth'] = props[3];
									lineRow['ReceivedBandwidth'] = props[4];

									return lineRow;
								});
				vm.tbDataWifi5 = [];
				vm.tbDataWifi5.push({
					id: 'main',
					wifi5WifiEnable: data['wifi5WlanEnable']||'',
					wifi5WifiSsid: data['wifi5WifiSsid']||'',
					wifi5WifiHidessid: data['wifi5WifiHidessid']||'',
					wifi5WifiApisolate: data['wifi5WifiApisolate']||'',
					wifi5WifiEncryption: data['wifi5WifiEncryption']||'',
					wifi5WifiPassphrase: data['wifi5WifiPassphrase']||'',
					wifi5WifiDevicelist: data['wifi5WifiDevicelist']?listData:[]
				});
				// networkForm更新
				Object.assign(vm.networkForm,{
					wifi5WifiSsid: data['wifi5WifiSsid']||'',
					wifi5WifiHidessid: data['wifi5WifiHidessid']||'',
					wifi5WifiApisolate: data['wifi5WifiApisolate']||'',
					wifi5WifiEncryption: data['wifi5WifiEncryption']||'',
					wifi5WifiPassphrase: data['wifi5WifiPassphrase']||''
				});
				// sub 
				['1','2','3'].map(function(item){
					var row = {},pre = 'wifi5Wifi',suf = 'Old',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',
						hideKey = pre+item+'Hidessid',
						apKey = pre+item+'Apisolate',
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase',
						deviceKey = pre+item+'Devicelist';

					row['id'] = item;
					row['wifi5WifiEnable'] = data[enableKey]||'';
					row['wifi5WifiSsid'] = data[idKey]||'';
					row['wifi5WifiHidessid'] = data[hideKey]||'';
					row['wifi5WifiApisolate'] = data[apKey]||'';
					row['wifi5WifiEncryption'] = data[encryKey]||'';
					row['wifi5WifiPassphrase'] = data[phraKey]||'';

					var list = (data[deviceKey]||'').split(';').map(function(line){
												var lineRow = {},
													props = line.split(',');
												
												lineRow['MACaddress'] = props[0];
												lineRow['Aid'] = props[1];
												lineRow['wirelessMode'] = props[2];
												lineRow['sentBandwidth'] = props[3];
												lineRow['ReceivedBandwidth'] = props[4];

												return lineRow;
											});
					row['wifi5WifiDevicelist'] = data[deviceKey]?list:[];

					// networkForm更新
					vm.networkForm[enableKey] = data[enableKey]||'';
					vm.networkForm[idKey] = data[idKey]||'';
					vm.networkForm[hideKey] = data[hideKey]||'';
					vm.networkForm[apKey] = data[apKey]||'';
					vm.networkForm[encryKey] = data[encryKey]||'';
					vm.networkForm[phraKey] = data[phraKey]||'';

					vm.tbDataWifi5.push(row);
				});
			},
			initNetwork(data, dmzExcluded) {
				var vm = this,
					httpsEnable = ['1','True'].includes(data["httpsEnable"])?'True':'False';

				// network
				vm.networkForm.httpsEnable = httpsEnable;
				vm.networkForm.lanEnable = data["lanEnable"]=='1'?'1':'0';
				vm.isLANEnable = [0,1,'0','1'].includes(data["lanEnable"]);

				if(!dmzExcluded) {
					if(data["dmzEnable"] == "1" || data["dmzEnable"] == "0") {
						vm.networkForm.dmzEnable = data["dmzEnable"];
						vm.networkForm.dmzHostAddress = data["dmzHostAddress"];
						vm.dmzTips = '';
					}else if(data["dmzEnable"] == "NotSupport") {
						vm.networkForm.dmzEnable = data["dmzEnable"];
						vm.dmzTips = 'Not support';
					}else if(data["dmzEnable"] == "NotSync") {
						vm.networkForm.dmzEnable = data["dmzEnable"];
						vm.dmzTips = 'Not sync';
					}
				}
				// wifi
				vm.initWLAN(data);
				vm.initWifi5(data);

				// DNS
				if(data["lanDnsEnable"] == "1" || data["lanDnsEnable"] == "0") {
					vm.networkForm.lanDnsEnable = data["lanDnsEnable"];
					vm.networkForm.lanDnsMode = data["lanDnsMode"];
					vm.networkForm.lanDns1Address = data["lanDns1Address"];
					vm.networkForm.lanDns2Address = data["lanDns2Address"];
					vm.networkForm.lanDns3Address = data["lanDns3Address"];
					vm.lanDnsTips = '';
				}else if(data["lanDnsEnable"] == "NotSupport") {
					vm.networkForm.lanDnsEnable = data["lanDnsEnable"];
					vm.networkForm.wanDnsEnable = data["wanDnsEnable"];
					vm.lanDnsTips = 'Not support';
				}else if(data["lanDnsEnable"] == "NotSync") {
					vm.networkForm.lanDnsEnable = data["lanDnsEnable"];
					vm.networkForm.wanDnsEnable = data["wanDnsEnable"];
					vm.lanDnsTips = 'Not sync';
				}

				// Host
				if(['NotSupport','NotSync'].includes(data['lanIp'])) {
					vm.networkForm.lanHostEnable = data['lanIp'];
					vm.lanHostTips = data['lanIp'] == 'NotSync'? 'Not sync':'Not support';
				}else {
					vm.networkForm.lanIp = data['lanIp'];
					vm.lanHostTips = '';
				}

				if(data["wanDnsEnable"] == "1" || data["wanDnsEnable"] == "0") {
					vm.networkForm.wanDnsEnable = data["wanDnsEnable"];
					vm.networkForm.wanDnsMode = data["wanDnsMode"];
					vm.networkForm.wanDnsPriAddress = data["wanDnsPriAddress"];
					vm.networkForm.wanDnsSecAddress = data["wanDnsSecAddress"];
					vm.wanDnsTips = '';
				}else if(data["wanDnsEnable"] == "NotSupport") {
					vm.networkForm.wanDnsEnable = data["wanDnsEnable"];
					vm.networkForm.wanDnsEnable = data["wanDnsEnable"];
					vm.wanDnsTips = 'Not support';
				}else if(data["wanDnsEnable"] == "NotSync") {
					vm.networkForm.wanDnsEnable = data["wanDnsEnable"];
					vm.networkForm.wanDnsEnable = data["wanDnsEnable"];
					vm.wanDnsTips = 'Not sync';
				}

				// ap wifi
				if(data["apWlanEnable"] == "1" || data["apWlanEnable"] == "0") {
					[	'apWlanEnable',
						'ap4MasterSsid','ap4MasterEncryption','ap4MasterPassPhrase',
						'ap5MasterSsid','ap5MasterEncryption','ap5MasterPassPhrase',
						'ap6MasterSsid','ap6MasterEncryption','ap6MasterPassPhrase',
						'ap4Channel','ap4BandWidth','ap5Channel','ap5BandWidth','ap6Channel','ap6BandWidth','country'
					].map(function(code){
						vm.networkForm[code] = data[code];
					});
					vm.apwifiTips = '';
				}else if(data["apWlanEnable"] == "NotSupport") {
					vm.networkForm['apWlanEnable'] = data['apWlanEnable'];
					vm.apwifiTips = 'Not support';
				}else if(data["apWlanEnable"] == "NotSync") {
					vm.networkForm['apWlanEnable'] = data['apWlanEnable'];
					vm.apwifiTips = 'Not sync';
				}
			},
			tabClick(tab,active) {
				var vm = this;

				if(!vm.isOnline) {
					return;
				}
				
				if(tab.code == 'apnSet'){
					vm.title = tab.text;
					vm.activeCode = tab.code;
					
					$("#apnSetBox").load("${ctx}/cell/CPE/toCpeAPNDetailParamInfoPage.action?cpeCode=" + vm.cpeCode,function(data){
						$.parser.parse(this);
					});
				}else {
					vm.title = tab.text;
					vm.activeCode = tab.code;
				}
				
				
				<%-- if( active == '') {
					if(tab.code == 'apnSet'){
						vm.title = tab.text;
						
						$("#apnSetBox").load("${ctx}/cell/CPE/toCpeAPNDetailParamInfoPage.action?cpeCode=" + '${cpeCode}',function(data){
							$.parser.parse(this);
						});
					}else {
						vm.title = tab.text;
						vm.activeCode = tab.code;
					}
					
				}else {
					if(isFormChanged(vm.$refs[active])) {
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							confirmButtonText: '<%=rb.getString("QueDing")%>',
							cancelButtonText: '<%=rb.getString("QuXiao")%>',
							type: 'warning',
							closeOnClickModal: false
						}).then(() => {
							vm.title = tab.text;
							vm.activeCode = tab.code;
						}).catch(() => {})
					}else {
						vm.title = tab.text;
						vm.activeCode = tab.code;
					}
				} --%>
			},
			isLanDnsHas() {
				var vm = this,
					dns1 = vm.networkForm.lanDns1Address,
					dns2 = vm.networkForm.lanDns2Address,
					dns3 = vm.networkForm.lanDns3Address;

				if(dns1 || dns2 || dns3) {
					return true;
				}else {
					return false;
				}
			},
						
			//新建 earfcn and pci
			addPcilockBtn(){
				
				var vm = this , startValue = vm.lteForm.earfcnStart.trim(), endValue = vm.lteForm.pciEnd.trim(), regNum = /^\d+$/;
				if(startValue == '' || endValue == ''){
					vm.earfcnPciErrorMessage = '<%=rb.getString("QingShuRu")%><%=rb.getString("PinDian")%> , <%=rb.getString("PCI2")%>';
				}else{
					
					if(!regNum.test(startValue) || ( startValue - 0 < 0 || startValue - 65535 > 0)) {
						vm.earfcnPciErrorMessage = 'Earfcn <%=rb.getString("FanWei")%>：0 ~65535';
					}else if(!regNum.test(endValue) || (endValue - 0 < 0 || endValue-503 > 0)){
						vm.earfcnPciErrorMessage = 'PCI <%=rb.getString("FanWei")%>：0~503';
					}else{
						str = startValue + "," + endValue;
						if( vm.lteForm.earfcnPciList.indexOf(str) == -1){
							vm.lteForm.earfcnPciList.push({startValue,endValue});
							vm.lteForm.earfcnStart = '';
							vm.lteForm.pciEnd = '';
							vm.earfcnPciErrorMessage = '';
							vm.lteForm.CPE_Frequency = vm.lteForm.earfcnPciList.map(function(item){					
								return item.startValue;
							}).join(',');
							vm.lteForm.PCI_value = vm.lteForm.earfcnPciList.map(function(item){					
								return item.endValue;
							}).join(',');
						}else{
							vm.earfcnPciErrorMessage = '<%=rb.getString("IPFanWeiYiCunZai")%>';
						} 
					}					
				}				
			},
			removeEarfcnPci(item){
				var vm = this, index = vm.lteForm.earfcnPciList.indexOf(item);
				if(index !== -1){
					vm.lteForm.earfcnPciList.splice(index,1)
				}

				vm.lteForm.CPE_Frequency = vm.lteForm.earfcnPciList.map(function(item){
					return item.startValue;
				}).join(',');
				vm.lteForm.PCI_value = vm.lteForm.earfcnPciList.map(function(item){
					return item.endValue;
				}).join(',');
				vm.earfcnPciErrorMessage = '';
			},
			//add  earfcn
			addEarfcnBtn(){
				var vm = this , value = vm.lteForm.onlyEarfcn.trim();
				if(value && isNaN(value)) {
					vm.earfcnErrorMessage = '<%=rb.getString("PinDianGeShiCuoWu")%>';
				}else if(value) {
					if(value - 0 < 0 || value - 65535 > 0) {
						vm.earfcnErrorMessage = '<%=rb.getString("PinDianChaoChuFanWei")%>';
					}else {
						if(vm.lteForm.earfcnList.indexOf(value) == -1){
							vm.lteForm.earfcnList.push(value);
							vm.lteForm.onlyEarfcn = '';
							vm.earfcnErrorMessage = '';
							vm.lteForm.CPE_Frequency = vm.lteForm.earfcnList.map(function(item){
								return item;							
							}).join(',');
						}else{
							vm.earfcnErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}
				}else {
					vm.earfcnErrorMessage = '<%=rb.getString("PinDianWeiKong")%>';
				} 
			},
			removeEarfcn(item){
				var vm = this, index = vm.lteForm.earfcnList.indexOf(item);
			
				if(index !== -1){
					vm.lteForm.earfcnList.splice(index,1)
				}
				vm.lteForm.CPE_Frequency = vm.lteForm.earfcnList.map(function(item){
					return item;							
				}).join(',');
				vm.earfcnErrorMessage = '';
			},
			// add pci
			addPciBtn(){
				var vm = this , value = vm.lteForm.onlyPci;
				if(value && isNaN(value)) {
					vm.pciErrorMessage = '<%=rb.getString("PCIGeShiCuoWu")%>';
				}else if(value) {
					if(value-0 < 0 || value-503 > 0) {
						vm.pciErrorMessage = '<%=rb.getString("PCIChaoChuFanWei")%>';
					}else {
						if(vm.lteForm.pciList.indexOf(value) == -1){
							vm.lteForm.pciList.push(value);
							vm.lteForm.onlyPci = '';
							vm.pciErrorMessage = '';
							vm.lteForm.PCI_value = vm.lteForm.pciList.map(function(item){
								return item;
							}).join(',');
						}else{
							vm.pciErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}
				}else {
					vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
				} 
			},
			removePci(item){
				var vm = this, index = vm.lteForm.pciList.indexOf(item);
				if(index !== -1){
					vm.lteForm.pciList.splice(index,1)
				}
				vm.lteForm.PCI_value = vm.lteForm.pciList.map(function(item){
					return item;
				}).join(',');
				vm.pciErrorMessage = '';
			},
			handleSizeChange(val){
				this.pageSize = val;
			},
			handleCurrentChange(val){
				this.currentPage = val;
			},
			systemAddIp(){			
				this.addIpShow = true;
			},
			//update ip
			updateIp(row,index){ 
				var vm = this;
				vm.editIndex = index;
				vm.editIpForm.ipStart = row.ipStart;
				vm.editIpForm.ipEnd = row.ipEnd;
				vm.oldEditIpStart = row.ipStart,
				vm.oldEditIpEnd = row.ipEnd;
				vm.editIpShow = true;
		    },
		    saveEditIp(){
		    	var vm = this,
					reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
					ipStart = vm.editIpForm.ipStart,
					ipEnd = vm.editIpForm.ipEnd;
		    	
				//endIp不是必填项
				if(ipStart === '' || ipStart === null || ipStart === undefined){					
					vm.editErrorMsg = '<%=rb.getString("IPDiZhiBuNengWeiKong")%>';
				}else{
					//如果只有start IP
					 if(ipEnd == '' || ipEnd == null){
						if(reg.test(ipStart)){
							str = ipStart;
							if(ipStart == vm.oldEditIpStart){
								vm.editErrorMsg = '<%=rb.getString("IPYiCunZai")%>';
							}else{
								vm.tableData.splice(vm.editIndex,1,vm.editIpForm)
								vm.closeEditIp();
							}
						}else{
							vm.editErrorMsg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}else{
						//如果star ip ，end ip都有
						if(reg.test(ipStart) && reg.test(ipEnd) && vm.compareIp(ipStart,ipEnd)){
							if(ipStart == vm.oldEditIpStart && ipEnd == vm.oldEditIpEnd){
								vm.editErrorMsg = '<%=rb.getString("IPFanWeiYiCunZai")%>';
							}else{
								vm.tableData.splice(vm.editIndex,1,vm.editIpForm)
								vm.closeEditIp();
							}
						}else{
							vm.editErrorMsg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}										
				} 
		    },
		    closeEditIp(){
		    	var vm = this;
		    	vm.editIpForm = {
					ipStart:'',
					ipEnd:''
				}
		    	vm.editErrorMsg = '';
				vm.$refs.editIpForm.resetFields();
		    	vm.editIpShow = false;
		    },
			//ip list 表格删除
			deleteIp(row,index){ 
				var vm = this;
				if(index >= 0) {
					vm.tableData.splice(index,1);			    	
			    }
		    },
			
			//添加ip
			addIpBtn(){
		    	var vm = this,
					reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
					ipStart = vm.addIpForm.ipStart,
					ipEnd = vm.addIpForm.ipEnd,
					str = '';
				
				//endIp不是必填项
				if(ipStart == ''){					
					vm.errorMessage = '<%=rb.getString("IPDiZhiBuNengWeiKong")%>';
				}else{
					//如果只有start IP
					if(ipEnd == '' || ipEnd == null){
						if(reg.test(ipStart)){
							str = ipStart;
							if(vm.addIpForm.ipGroup.indexOf(str) == -1){
								vm.addIpForm.ipGroup.push(str);
								vm.addIpForm.ipStart = '';
								vm.errorMessage = '';
							}else{
								vm.errorMessage = '<%=rb.getString("IPYiCunZai")%>';
							}
						}else{
							vm.errorMessage = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}else{
						//如果star ip ，end ip都有
						if(reg.test(ipStart) && reg.test(ipEnd) && vm.compareIp(ipStart,ipEnd)){
							str = ipStart + "-" + ipEnd;
							if(vm.addIpForm.ipGroup.indexOf(str) == -1){
								vm.addIpForm.ipGroup.push(str);
								vm.addIpForm.ipStart = '';
								vm.addIpForm.ipEnd = '';
								vm.errorMessage = '';
							}else{
								vm.errorMessage = '<%=rb.getString("IPFanWeiYiCunZai")%>';
							}
						}else{
							vm.errorMessage = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}										
				} 
			},
			/**
			 * 新建ip弹窗中的单个删除ip
			 * @param item {string} 删除项
			*/
			removeIp(item){
				var vm = this, index = vm.addIpForm.ipGroup.indexOf(item);
				
				if(index !== -1){
					vm.addIpForm.ipGroup.splice(index,1)
				}
				vm.errorMessage = '';
			},
			/**
			 * 比较ip大小
			 * @param ipStart {string} 初始ip
			 * @param ipEnd {string} 结束ip
			*/
			compareIp(ipStart,ipEnd){
				var temp1,
					temp2,
					bool = true;
				temp1 = ipStart.split(".");
				temp2 = ipEnd.split(".");
				for(var i=0;i<4;i++){
					if(parseInt(temp1[i])>parseInt(temp2[i])){
						bool = false;
					}
				}

				return bool;
			},
			
			//添加 ip
			saveAddIp(){
				var vm = this;
				if(vm.addIpForm.ipGroup.length == 0){
					vm.errorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
				}else{
					var ipListData = vm.addIpForm.ipGroup.toString().split(",").join(";"),									
						newIpListData = ipListData.split(';'),
						getRowData ={};
					
					newIpListData.map((item,index) => {
						if(item.includes('-')){
							getRowData = {     						
								ipStart:item.split('-')[0],
	      						ipEnd:item.split('-')[1]	    			
	      		    		}
						}else{
							getRowData = {     						
								ipStart:item	    			
		      		    	}
						}
						vm.tableData.push(getRowData);
					}) 
					vm.oldNewTableData();
					vm.closeAddIp();
				}				
			},
			
			oldNewTableData(){
				var vm = this;
				for(var i = 0; i< vm.tableData.length; i++){
					for(var j = i+1; j < vm.tableData.length; j++ ){
						//两种情况，1：只有 start ip; 2 start ip, end ip 都有时
						if(vm.tableData[i].ipEnd){
							if(vm.tableData[i].ipStart == vm.tableData[j].ipStart && vm.tableData[i].ipEnd == vm.tableData[j].ipEnd){
								vm.tableData.splice(j,1);								
								j--;
							}
						}else{
							if(vm.tableData[i].ipStart == vm.tableData[j].ipStart){
								vm.tableData.splice(j,1);							
								j--;
							}
						}					
					}					
				}
				return vm.tableData;
			},
			
			//关闭添加IP弹窗
			closeAddIp(){
				var vm = this;
				vm.addIpForm = {
					ipStart:'',
					ipEnd:'',
					ipGroup:[]
				}
				vm.errorMessage = '';
				vm.$refs.addIpForm.resetFields();
				vm.addIpShow = false;			
			},
			//---------------------------------------ip
			
			wlanEnableChange(val) {
				var vm = this,
					row = vm.tbData[0];

				if(row) {
					row.wifiEnable = val == '1'?'1':'0';
				}
			},
			wifi5EnableChange(val) {
				var vm = this,
					row = vm.tbDataWifi5[0];
				
				if(row) {
					row.wifi5WifiEnable = val == '1'?'1':'0';
				}
			},
			watchdogEnableChange() {
				this.$refs.system.validate(function(){});
			},
			scanModeChange(val){
				var vm = this;
				if(val == 'freqpreferred'){					
					vm.lteForm.earfcnList = vm.lteForm.earfcnPciList.map(function(item){
						return item.startValue;
					});
					vm.lteForm.CPE_Frequency = vm.lteForm.earfcnList;
				}
			},
			closeSetting() {
				eventBus.$emit('close-cpe-setting');
			},
			/* 刷新AP列表 */
			refreshAPList() {
				var vm = this;

				$.post("${ctx}/cell/ap/refreshCPEApList.action", {cpeCode: vm.cpeCode}, function(data){
					if(data["success"]){
						
					}else{
						vm.$message({
							message: data["message"],
							type: 'error'
						});
					}
				}, "json");
			},
			updateRule(index) {
				var vm = this,
					scanMode = vm.lteForm.scanMode,
					codes = {
						freqpreferred: 'freqList',
						pcilock: 'unitList',
						pcionlylock: 'pciList'
					},
					newRows = {
						freqpreferred: {earfcn: ''},
						pcilock: {earfcn: '', pci: ''},
						pcionlylock: {pci: ''}
					},
					list = vm.lteForm[codes[scanMode]];

				if(list) {
					if(index) list.splice(index,1);
					else list.push(newRows[scanMode]);
				}
			},
			earfcnValid(rule,value,cb) {
				var vm = this;

				if(value && isNaN(value)) {
					cb('<%=rb.getString("PinDianGeShiCuoWu")%>');
				}else if(value) {
					if(value-0<0 || value-65535>0) {
						cb('<%=rb.getString("PinDianChaoChuFanWei")%>')
					}else {
						cb();
					}
				}else {
					cb('<%=rb.getString("PinDianWeiKong")%>');
				}
			},
			pciValid(rule,value,cb) {
				var vm = this;

				if(value && isNaN(value)) {
					cb('<%=rb.getString("PCIGeShiCuoWu")%>');
				}else if(value) {
					if(value-0<0 || value-503>0) {
						cb('<%=rb.getString("PCIChaoChuFanWei")%>')
					}else {
						cb();
					}
				}else {
					cb('<%=rb.getString("PCIWeiKong")%>');
				}
			},
			saveSetting() {
				var vm = this,
					form = vm[vm.activeCode+'Form'],
					formEl = vm.$refs[vm.activeCode],
					flagEarfcn = false;	
				
				if(vm.activeCode == 'lte'){
					//earfcn, earfcn and pci, pci					
					if(vm.lteForm.scanMode == 'freqpreferred'){
						if(vm.lteForm.earfcnList.length == 0){
							flagEarfcn = true;
							vm.earfcnErrorMessage = '<%=rb.getString("PinDianWeiKong")%>';
						}else{
							flagEarfcn = false;
							vm.earfcnErrorMessage = '';						
						} 
					}else if(vm.lteForm.scanMode == 'pcilock'){
						if(vm.lteForm.earfcnPciList.length == 0){
							flagEarfcn = true;
							vm.earfcnPciErrorMessage = '<%=rb.getString("PinDianWeiKong")%> ';
						}else{
							flagEarfcn = false;
							vm.earfcnPciErrorMessage = '';						
						}
					}else if(vm.lteForm.scanMode == 'pcionlylock'){
						if(vm.lteForm.pciList.length == 0){
							flagEarfcn = true;
							vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
						}else{
							flagEarfcn = false;
							vm.pciErrorMessage = '';						
						}
					}
					
				}
				
				formEl.validate(function(valid){
					
					if(valid && flagEarfcn == false){
						
						var params = vm.formatParams(form);
						vm.systemForm.accessIpList = params.accessIpList;
						var pureParams = vm.pureParams(formEl, params),
							postParams = {
								cpeCode: vm.cpeCode,
								timeZone: timeZone
							};

						if(isFormChanged(vm.$refs[vm.activeCode])) {
							// 过滤参数
							if(vm.activeCode == 'lte') {													
								
								var pciChanged = false,
									pci5GChanged = false;
								
								['scanMode','CPE_Frequency','PCI_value'].map(function(item){
									if(pureParams[item] != undefined) pciChanged = true;
								});
								['nrLockMode','nrPci', 'nrBand'].map(function(item){
									if(pureParams[item] != undefined) pci5GChanged = true;
								});
								
								Object.assign(postParams, pureParams);
								
								if(pciChanged || pci5GChanged) postParams.pciChanged = 1;

								if(pciChanged == true) {
									if(vm.lteForm.scanMode == 'freqpreferred'){
										['scanMode','CPE_Frequency'].map(function(code){
											postParams[code] = form[code];
										}); 
									}else if(vm.lteForm.scanMode == 'pcilock'){
										['scanMode','CPE_Frequency','PCI_value'].map(function(code){
											postParams[code] = form[code];
										});
									}else if(vm.lteForm.scanMode == 'pcionlylock'){
										['scanMode','PCI_value'].map(function(code){
											postParams[code] = form[code];
										});	
									}else if(vm.lteForm.scanMode == 'fullband'){
										['scanMode'].map(function(code){
											postParams[code] = form[code];
										});
									}
								}

								if(pci5GChanged == true) {
									if(vm.lteForm.nrLockMode == 'celllock') {
										['nrLockMode','nrPci'].map(function(code){
											postParams[code] = form[code];
										});
									}else if(vm.lteForm.nrLockMode == 'bandlock') {
										['nrLockMode','nrBand'].map(function(code){
											postParams[code] = form[code];
										});
									}else {
										['nrLockMode'].map(function(code){
											postParams[code] = form[code];
										});
									}
								}							
																								
							}else if(vm.activeCode == 'basic') {
								Object.assign(postParams, pureParams, {cpeNameChanged: 1});
							}else if(vm.activeCode == 'network') {
								var dmzChanged = false,
									wifiChanged = false,
									wifi5Changed = false,
									lanDnsChanged = false,
									lanHostChanged = false,
									wanDnsChanged = false,
									apWifiChanged = false;
	
								['dmzEnable','dmzHostAddress'].map(function(item){
									if(pureParams[item] != undefined) dmzChanged = true;
								});

								['lanDnsEnable','lanDnsMode','lanDns1Address','lanDns2Address','lanDns3Address'].map(function(item){
									if(pureParams[item] != undefined) lanDnsChanged = true;
								});

								['lanIp'].map(function(item){
									if(pureParams[item] != undefined) lanHostChanged = true;
								});

								['wanDnsEnable','wanDnsMode','wanDnsPriAddress','wanDnsSecAddress'].map(function(item){
									if(pureParams[item] != undefined) wanDnsChanged = true;
								});

								['wlanEnable', 'wifiMode', 'wifiRate', 'wifiChannel', 'wifiBandwidth',
								'wifiSsid', 'wifiHidessid', 'wifiApisolate', 'wifiEncryption', 'wifiPassphrase',
								'wifi1Enable', 'wifi1Ssid', 'wifi1Hidessid', 'wifi1Apisolate', 'wifi1Encryption', 'wifi1Passphrase',
								'wifi2Enable', 'wifi2Ssid', 'wifi2Hidessid', 'wifi2Apisolate', 'wifi2Encryption', 'wifi2Passphrase',
								'wifi3Enable', 'wifi3Ssid', 'wifi3Hidessid', 'wifi3Apisolate', 'wifi3Encryption', 'wifi3Passphrase'].map(function(item){
									if(pureParams[item] != undefined) wifiChanged = true;
								});
								
								['wifi5WlanEnable', 'wifi5WifiMode', 'wifi5WifiRate', 'wifi5WifiChannel', 'wifi5WifiBandwidth','wifi5WifiSupportChannel',
								'wifi5WifiSsid', 'wifi5WifiHidessid', 'wifi5WifiApisolate', 'wifi5WifiEncryption', 'wifi5WifiPassphrase',
								'wifi5Wifi1Enable', 'wifi5Wifi1Ssid', 'wifi5Wifi1Hidessid', 'wifi5Wifi1Apisolate', 'wifi5Wifi1Encryption', 'wifi5Wifi1Passphrase',
								'wifi5Wifi2Enable', 'wifi5Wifi2Ssid', 'wifi5Wifi2Hidessid', 'wifi5Wifi2Apisolate', 'wifi5Wifi2Encryption', 'wifi5Wifi2Passphrase',
								'wifi5Wifi3Enable', 'wifi5Wifi3Ssid', 'wifi5Wifi3Hidessid', 'wifi5Wifi3Apisolate', 'wifi5Wifi3Encryption', 'wifi5Wifi3Passphrase'].map(function(item){
									if(pureParams[item] != undefined) wifi5Changed = true;
								});

								[	'apWlanEnable',
									'ap4MasterSsid','ap4MasterEncryption','ap4MasterPassPhrase',
									'ap5MasterSsid','ap5MasterEncryption','ap5MasterPassPhrase',
									'ap6MasterSsid','ap6MasterEncryption','ap6MasterPassPhrase',
									'ap4Channel','ap4BandWidth','ap5Channel','ap5BandWidth','ap6Channel','ap6BandWidth','country'
								].map(function(item){
									if(pureParams[item] != undefined) apWifiChanged = true;
								});
	
								Object.assign(postParams, pureParams);
								if(dmzChanged) postParams.dmzChanged = 1;
								if(lanDnsChanged) postParams.lanDnsChanged = 1;
								if(lanHostChanged) postParams.lanIpChanged = 1;
								if(wanDnsChanged) postParams.wanDnsChanged = 1;
								if(wifiChanged) postParams.wifiChanged = 1;
								if(wifi5Changed) postParams.wifi5Changed = 1;
								if(apWifiChanged) postParams.apWifiChanged = 1;
								
								if(form['lanDnsMode'] == '1') {// auto模式
									['lanDnsEnable','lanDnsMode'].map(function(code){
										postParams[code] = form[code];
									});
								}else {
									['lanDnsEnable','lanDnsMode','lanDns1Address','lanDns2Address','lanDns3Address'].map(function(code){
										postParams[code] = form[code];
									});
								}
								
								['dmzEnable', 'dmzHostAddress','wanDnsEnable','wanDnsMode','wanDnsPriAddress','wanDnsSecAddress'].map(function(code){
									postParams[code] = form[code];
								});
							}else {
								//system
								var wanAccChanged = false, watchDogChanged = false, httpsEnableChanged = false, snmpChanged = false;
								
								if(pureParams["httpsEnable"] != undefined) httpsEnableChanged = true;
								
								['httpsWanEnable','wanAccessControlEnable','accessIpList'].map(function(item){
									if(pureParams[item] != undefined) wanAccChanged = true;
								});
								['watchDogEnable','watchDogPingIp','watchDogPingTimeout','watchDogPingCount','watchDogFailureReboot'].map(function(item){
									if(pureParams[item] != undefined) watchDogChanged = true;
								});
								['snmpEnable', 'nmsAddress', 'nmsPort', 'listeningPort', 'trapCommunity','version', 'readCommunity', 'rwCommunity', 'userName', 'authenticationProtocol','authenticationPassphrase', 'privacyProtocol', 'nmsPort', 'privacyPassphrase'].map(function(item){
									if(pureParams[item] != undefined) snmpChanged = true;
								});

								Object.assign(postParams, pureParams);
								if(httpsEnableChanged) {
									postParams.httpsEnableChanged = 1;
								}
								
								if(wanAccChanged) {
									postParams.wanAccChanged = 1;
									postParams.httpsWanEnable = vm.systemForm.httpsWanEnable;
									postParams.wanAccessControlEnable = vm.systemForm.wanAccessControlEnable;
									postParams.accessIpList = vm.systemForm.accessIpList;
								}
								if(watchDogChanged) {
									postParams.watchDogChanged = 1;
									['watchDogEnable','watchDogPingIp','watchDogPingTimeout','watchDogPingCount','watchDogFailureReboot'].map(function(code){
										postParams[code] = form[code];
									});
								}
								if(snmpChanged) {
									postParams.snmpChanged = 1;
									//根据版本处理提交参数  v3 时-去除  'readCommunity', 'rwCommunity',参数； v1&v2c 时-去除 'userName', 'authenticationProtocol','authenticationPassphrase', 'privacyProtocol', 'nmsPort', 'privacyPassphrase' 参数
									if(vm.systemForm.version == 'v3'){
										['snmpEnable', 'nmsAddress', 'nmsPort', 'listeningPort', 'trapCommunity','version', 'userName', 'authenticationProtocol','authenticationPassphrase', 'privacyProtocol', 'nmsPort', 'privacyPassphrase'].map(function(code){
											postParams[code] = form[code];
										});
									}else{
										['snmpEnable', 'nmsAddress', 'nmsPort', 'listeningPort', 'trapCommunity','version', 'readCommunity', 'rwCommunity'].map(function(code){
											postParams[code] = form[code];
										});
									}
								}													
							}
							axios.post('${ctx}/cell/CPE/setCpeParams.action',stringify(postParams)).then(function(res){
								var  data = res.data;
	
								if(data["success"]){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type: 'success'
									});
	
									vm.closeSetting();
									$("#cpeSettingOption").removeClass("loading");
									cpevm.refreshList();
								}else{
									vm.$message({
										message: data["message"],
										type: 'error'
									});
								}
							}).catch(function(){});
						}else {
							vm.$message({
								message: '<%=rb.getString("CanShuZhiMeiYouBianHua")%>',
								type: 'warning'
							})
						}
					}
				});
			},
			pureParams(formEl, params) {
				var map = {};

				formEl.fields.map(function(field){
					if(Array.isArray(field.fieldValue)){
						var vList = field.fieldValue.map(function(item){return item});
						var oList = (field.reinitialValue||[]).map(function(item){return item});
						var val = vList.sort().join(' ');
						var orVal = oList.sort().join(' ');
						if(val != orVal) isChanged = true;
					}else{
						if(isNull(field.fieldValue) && isNull(field.reinitialValue)){
							
						}else if(field.fieldValue != field.reinitialValue) {
							var key = field.prop;
							
							map[key] = params[key];
						};
					}
				});

				return map;
				
				function isNull(val){
					if(val==undefined || val == null || val =="") return true;
					else return false;
				}
			},
			formatParams(form) {
				var vm = this,
					code = vm.activeCode,
					params = {};

				if(code == 'lte') {
					var scanMode = form.scanMode;

					params.scanMode = scanMode;
					params.nrLockMode = form.nrLockMode;

					if(scanMode == 'freqpreferred') {
						params.CPE_Frequency = form.earfcnList.map(function(item){
							return item;
						}).join(',');
					}
					if(scanMode == 'pcilock') {
						params.CPE_Frequency = form.earfcnPciList.map(function(item){
							return item.startValue;
						}).join(',');
						params.PCI_value = form.earfcnPciList.map(function(item){
							return item.endValue;
						}).join(',');
					}
					if(scanMode == 'pcionlylock') {
						params.PCI_value = form.pciList.map(function(item){
							return item;
						}).join(',');
					}

					var lockMode = form.nrLockMode;

					if(lockMode == 'celllock') {
						params.nrPci = form.nrPci;
						params.nrBand = '';
					}
					if(lockMode == 'bandlock') {
						params.nrPci = '';
						params.nrBand = form.nrBand;
					}
				}else if(code == 'system'){
					var curIp = vm.tableData.map((item,index) => {
						if(item.ipEnd ){
							return item.ipStart + '-' + item.ipEnd;
						}else{
							return item.ipStart;
						}								
					}).join(';');
					params.httpsEnable = vm.systemForm.httpsEnable;
					params.httpsWanEnable = vm.systemForm.httpsWanEnable;
					params.wanAccessControlEnable = vm.systemForm.wanAccessControlEnable;
					params.accessIpList = curIp;
					Object.assign(params, {
						httpsEnable: form.httpsEnable,
						httpsWanEnable: form.httpsWanEnable,
						wanAccessControlEnable: form.wanAccessControlEnable,
						watchDogEnable: form.watchDogEnable,
						watchDogPingIp: form.watchDogPingIp,
						watchDogPingTimeout: form.watchDogPingTimeout,
						watchDogPingCount: form.watchDogPingCount,
						watchDogFailureReboot: form.watchDogFailureReboot,
						snmpEnable:form.snmpEnable,						
						nmsAddress: form.nmsAddress,
						nmsPort:form.nmsPort,
						listeningPort: form.listeningPort,
						trapCommunity: form.trapCommunity,
						version: form.version,
						readCommunity: form.readCommunity,
						rwCommunity: form.rwCommunity,
						userName: form.userName,
						authenticationProtocol: form.authenticationProtocol,
						authenticationPassphrase:form.authenticationPassphrase,
						privacyProtocol: form.privacyProtocol,
						privacyPassphrase: form.privacyPassphrase
					
					});
				}else if(code == 'network') {

					if(form.wlanEnable == '1') {
						vm.tbData.map(function(row){
							if(['1','2','3'].includes(row.id)) {
								var index = row.id,
									pre = 'wifi',suf = 'Old',
									enableKey = pre+index+'Enable',
									idKey = pre+index+'Ssid',
									hideKey = pre+index+'Hidessid',
									apKey = pre+index+'Apisolate',
									encryKey = pre+index+'Encryption',
									phraKey = pre+index+'Passphrase';
								
								// networkForm更新
								form[enableKey] = row['wifiEnable'];
								form[idKey] = row['wifiSsid'];
								form[hideKey] = row['wifiHidessid'];
								form[apKey] = row['wifiApisolate'];
								form[encryKey] = row['wifiEncryption'];
								form[phraKey] = row['wifiPassphrase'];
							}else {
								Object.assign(form,{
									wifiSsid: row['wifiSsid'],
									wifiHidessid: row['wifiHidessid'],
									wifiApisolate: row['wifiApisolate'],
									wifiEncryption: row['wifiEncryption'],
									wifiPassphrase: row['wifiPassphrase']
								});
							}
						});

						Object.assign(params, form);
					}

					if(form.wifi5WlanEnable == '1') {
						vm.tbDataWifi5.map(function(row){
							if(['1','2','3'].includes(row.id)) {
								var index = row.id,
									pre = 'wifi5Wifi',suf = 'Old',
									enableKey = pre+index+'Enable',
									idKey = pre+index+'Ssid',
									hideKey = pre+index+'Hidessid',
									apKey = pre+index+'Apisolate',
									encryKey = pre+index+'Encryption',
									phraKey = pre+index+'Passphrase';
								
								// networkForm更新
								form[enableKey] = row['wifi5WifiEnable'];
								form[idKey] = row['wifi5WifiSsid'];
								form[hideKey] = row['wifi5WifiHidessid'];
								form[apKey] = row['wifi5WifiApisolate'];
								form[encryKey] = row['wifi5WifiEncryption'];
								form[phraKey] = row['wifi5WifiPassphrase'];
							}else {
								Object.assign(form,{
									wifi5WifiSsid: row['wifi5WifiSsid'],
									wifi5WifiHidessid: row['wifi5WifiHidessid'],
									wifi5WifiApisolate: row['wifi5WifiApisolate'],
									wifi5WifiEncryption: row['wifi5WifiEncryption'],
									wifi5WifiPassphrase: row['wifi5WifiPassphrase']
								});
							}
						});

						Object.assign(params, form);
					}
					
					Object.assign(params, {
						httpsEnable: form.httpsEnable,
						lanEnable: form.lanEnable,
						dmzEnable: form.dmzEnable,
						dmzHostAddress: form.dmzHostAddress,
						wlanEnable: form.wlanEnable,
						wifi5WlanEnable: form.wifi5WlanEnable,

						'lanIp': form.lanIp,
						'lanDnsEnable': form.lanDnsEnable,
						'lanDnsMode': form.lanDnsMode,
						'lanDns1Address': form.lanDns1Address,
						'lanDns2Address': form.lanDns2Address,
						'lanDns3Address': form.lanDns3Address,
						'wanDnsEnable': form.wanDnsEnable,
						'wanDnsMode': form.wanDnsMode,
						'wanDnsPriAddress': form.wanDnsPriAddress,
						'wanDnsSecAddress': form.wanDnsSecAddress,

						apWlanEnable: form.apWlanEnable,
						ap4MasterSsid: form.ap4MasterSsid,
						ap4MasterEncryption: form.ap4MasterEncryption,
						ap4MasterPassPhrase: form.ap4MasterPassPhrase,
						ap5MasterSsid: form.ap5MasterSsid,
						ap5MasterEncryption: form.ap5MasterEncryption,
						ap5MasterPassPhrase: form.ap5MasterPassPhrase,
						ap6MasterSsid: form.ap6MasterSsid,
						ap6MasterEncryption: form.ap6MasterEncryption,
						ap6MasterPassPhrase: form.ap6MasterPassPhrase,
						ap4Channel: form.ap4Channel,
						ap4BandWidth: form.ap4BandWidth,
						ap5Channel: form.ap5Channel,
						ap5BandWidth: form.ap5BandWidth,
						ap6Channel: form.ap6Channel,
						ap6BandWidth: form.ap6BandWidth,
						country: form.country
					});
				}else {
					Object.assign(params, form);
				}

				return params;
			},
			modifyWifi(row) {
				var vm = this;

				vm.wifiMDLShow = true;
				vm.$nextTick(function(){
					vm.$refs.modifyForm.resetFields();
					Object.assign(vm.mForm, row);
				});
			},
			modifyWifi5(row) {
				var vm = this;

				vm.wifi5WifiMDLShow = true;
				vm.$nextTick(function(){
					vm.$refs.modifyWifi5Form.resetFields();
					Object.assign(vm.mWifi5Form, row);
				});
			},
			saveModify() {
				var vm = this;

				vm.$refs.modifyForm.validate(function(valid){
					if(valid) {
						vm.tbData.map(function(item){
							if(item.id == vm.mForm.id) {
								Object.assign(item, vm.mForm);
							}
						});
						vm.wifiMDLShow = false;
					}
				});
			},
			saveModifyWifi5() {
				var vm = this;

				vm.$refs.modifyWifi5Form.validate(function(valid, errors){
					if(valid) {
						vm.tbDataWifi5.map(function(item){
							if(item.id == vm.mWifi5Form.id) {
								Object.assign(item, vm.mWifi5Form);
							}
						});
						vm.wifi5WifiMDLShow = false;
					}
				});
			},
			viewWifi(row) {
				var vm = this,
					deviceList = row.wifiDevicelist;

				vm.viewMDLShow = true;
				vm.$nextTick(function(){
					vm.viewTbData = deviceList;
				});
			},
			viewWifi5(row) {
				var vm = this,
					deviceList = row.wifi5WifiDevicelist;

				vm.wifi5ViewMDLShow = true;
				vm.$nextTick(function(){
					vm.viewTbData = deviceList;
				});
			},

			refreshDMZ() {
				var vm = this,
					url = '${ctx}/cell/CPE/queryDmzInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};
				
				vm.isDmzLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.dmzEnable == 'NotSync') {
						vm.networkForm.dmzEnable = 'NotSync';
						vm.dmzTips = 'Not sync';
					}else if(data.dmzEnable == 'NotSupport') {
						vm.networkForm.dmzEnable = 'NotSupport';
						vm.dmzTips = 'Not support';
					}else {
						if(data["dmzEnable"] == "1" || data["dmzEnable"] == "0") {
							vm.networkForm.dmzEnable = data["dmzEnable"];
							vm.networkForm.dmzHostAddress = data["dmzHostAddress"];
						}
						vm.dmzTips = '';
					}
					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.network, ['dmzEnable','dmzHostAddress']);
					});

					vm.isDmzLoading = false;
				});
			},
			
			//ip 刷新
			refreshIp() {
				var vm = this,
					url = '${ctx}/cell/CPE/queryWanAccInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};

				vm.isIpLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.wanAccessControlEnable == 'NotSync') {
						vm.systemForm.wanAccessControlEnable = 'NotSync';
						vm.ipTips = 'Not sync';
						vm.notSupportShow = false;
					}else if(data.wanAccessControlEnable == 'NotSupport') {
						vm.systemForm.wanAccessControlEnable = 'NotSupport';
						vm.ipTips = 'Not support';
						vm.notSupportShow = false;
					}else {					
						Object.assign(vm.systemForm, {
							httpsWanEnable: data['wanAccessControlHttpsWanEnable'],
							wanAccessControlEnable:data['wanAccessControlEnable'],
							accessIpList: data['wanAccessControlIpList']
						});
						// 后端反馈刷新时状态同httpsWanEnable 2026-1-4
						vm.systemForm.httpsEnable = data['wanAccessControlHttpsWanEnable'];
						vm.ipTips = '';
						vm.notSupportShow = true;
					}
					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.system, ['httpsEnable', 'httpsWanEnable', 'wanAccessControlEnable', 'accessIpList']);
					});
					
					vm.isIpLoading = false;
				});
			},
			refreshWatchDog() {
				var vm = this,
					url = '${ctx}/cell/CPE/queryWatchDogInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};

				vm.isWatchDogLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.watchDogEnable == 'NotSync') {
						vm.watchDogTips = 'Not sync';
					}else if(data.watchDogEnable == 'NotSupport') {
						vm.watchDogTips = 'Not support';
					}else {
						Object.assign(vm.systemForm, {
							watchDogEnable: data['watchDogEnable'],
							watchDogPingIp: data['watchDogPingIp'],
							watchDogPingTimeout: data['watchDogPingTimeout'],
							watchDogPingCount: data['watchDogPingCount'],
							watchDogFailureReboot: data['watchDogFailureReboot']
						});
						vm.watchDogTips = '';
					}

					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.system, ['watchDogEnable', 'watchDogPingIp', 'watchDogPingTimeout', 'watchDogPingCount', 'watchDogFailureReboot']);
					});
					
					vm.isWatchDogLoading = false;
				});
			},
			refreshSnmp(){
				var vm = this,
					url = '${ctx}/cell/CPE/setting/querySnmpInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};

				vm.isSnmpLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.snmpEnable == 'NotSync') {
						vm.snmpTips = 'Not sync';
					}else if(data.snmpEnable == 'NotSupport') {
						vm.snmpTips = 'Not support';
					}else {
						Object.assign(vm.systemForm, {
							snmpEnable: data['snmpEnable'],						
							nmsAddress: data['nmsAddress'],
							nmsPort: data['nmsPort'],
							listeningPort: data['listeningPort'],
							trapCommunity: data['trapCommunity'],
							version: data['version'],
							readCommunity: data['readCommunity'],
							rwCommunity: data['rwCommunity'],
							userName: data['userName'],
							authenticationProtocol: data['authenticationProtocol'],
							authenticationPassphrase: data['authenticationPassphrase'],
							privacyProtocol: data['privacyProtocol'],
							privacyPassphrase: data['privacyPassphrase']
						});
						vm.snmpTips = '';
					}

					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.system, ['snmpEnable', 'nmsAddress', 'nmsPort', 'listeningPort', 'trapCommunity','version', 'readCommunity', 'rwCommunity', 'userName', 'authenticationProtocol','authenticationPassphrase', 'privacyProtocol', 'nmsPort', 'privacyPassphrase']);
					});
					
					vm.isSnmpLoading = false;
				});	
			},
			refreshWifi() {
				var vm = this,
					url = '${ctx}/cell/CPE/queryWifiInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};

				vm.isWifiLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.wlanEnable == 'NotSync') {
						vm.wifiTips = 'Not sync';
					}else if(data.wlanEnable == 'NotSupport') {
						vm.wifiTips = 'Not support';
					}else {
						var dmzExcluded = true;
						vm.initWLAN(data);
						vm.wifiTips = '';
					}

					if(data.wifi5WlanEnable == 'NotSync') {
						vm.wifi5Tips = 'Not sync';
					} else if(data.wifi5WlanEnable == 'NotSupport') {
						vm.wifi5Tips = 'Not support';
					} else {
						vm.initWifi5(data);
						vm.wifi5Tips = '';
					}

					// 初始化form原始值
					vm.$nextTick(function(){
						var props = [
								'wlanEnable', 'wifiMode', 'wifiRate', 'wifiChannel', 'wifiBandwidth', 
								'wifiSsid', 'wifiHidessid', 'wifiApisolate', 'wifiEncryption', 'wifiPassphrase', 
								'wifi1Enable', 'wifi1Ssid', 'wifi1Hidessid', 'wifi1Apisolate', 'wifi1Encryption', 'wifi1Passphrase', 
								'wifi2Enable', 'wifi2Ssid', 'wifi2Hidessid', 'wifi2Apisolate', 'wifi2Encryption', 'wifi2Passphrase', 
								'wifi3Enable', 'wifi3Ssid', 'wifi3Hidessid', 'wifi3Apisolate', 'wifi3Encryption', 'wifi3Passphrase',
								
								'wifi5WlanEnable', 'wifi5WifiMode', 'wifi5WifiRate', 'wifi5WifiChannel', 'wifi5WifiBandwidth','wifi5WifiSupportChannel',
								'wifi5WifiSsid', 'wifi5WifiHidessid', 'wifi5WifiApisolate', 'wifi5WifiEncryption', 'wifi5WifiPassphrase',
								'wifi5Wifi1Enable', 'wifi5Wifi1Ssid', 'wifi5Wifi1Hidessid', 'wifi5Wifi1Apisolate', 'wifi5Wifi1Encryption', 'wifi5Wifi1Passphrase',
								'wifi5Wifi2Enable', 'wifi5Wifi2Ssid', 'wifi5Wifi2Hidessid', 'wifi5Wifi2Apisolate', 'wifi5Wifi2Encryption', 'wifi5Wifi2Passphrase',
								'wifi5Wifi3Enable', 'wifi5Wifi3Ssid', 'wifi5Wifi3Hidessid', 'wifi5Wifi3Apisolate', 'wifi5Wifi3Encryption', 'wifi5Wifi3Passphrase'
							];
					
						initForm(vm.$refs.network, props);
					});

					vm.isWifiLoading = false;
				});
			},
			refreshLanDns() {
				var vm = this,
					url = '${ctx}/cell/CPE/queryLanDnsInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};
				
				vm.isLanDnsLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.lanDnsEnable == 'NotSync') {
						vm.networkForm.lanDnsEnable = 'NotSync';
						vm.lanDnsTips = 'Not sync';
					}else if(data.lanDnsEnable == 'NotSupport') {
						vm.networkForm.lanDnsEnable = 'NotSupport';
						vm.lanDnsTips = 'Not support';
					}else {
						if(data["lanDnsEnable"] == "1" || data["lanDnsEnable"] == "0") {
							vm.networkForm.lanDnsEnable = data["lanDnsEnable"];
							vm.networkForm.lanDnsMode = data["lanDnsMode"];
							vm.networkForm.lanDns1Address = data["lanDns1Address"];
							vm.networkForm.lanDns2Address = data["lanDns2Address"];
							vm.networkForm.lanDns3Address = data["lanDns3Address"];
						}
						vm.lanDnsTips = '';
					}
					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.network, ['lanDnsEnable','lanDnsMode','lanDns1Address','lanDns2Address','lanDns3Address']);
					});

					vm.isLanDnsLoading = false;
				});
			},
			refreshLanHost() {
				var vm = this,
					url = '${ctx}/cell/CPE/setting/queryLanIpInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};
				
				vm.isLanHostLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.lanIp == 'NotSync') {
						vm.networkForm.lanHostEnable = 'NotSync';
						vm.lanHostTips = 'Not sync';
					}else if(data.lanIp == 'NotSupport') {
						vm.networkForm.lanHostEnable = 'NotSupport';
						vm.lanHostTips = 'Not support';
					}else {
						vm.networkForm.lanHostEnable = data["lanIp"];
						vm.networkForm.lanIp = data["lanIp"];
						vm.lanHostTips = '';
					}
					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.network, ['lanIp']);
					});

					vm.isLanHostLoading = false;
				});
			},
			refreshWanDns() {
				var vm = this,
					url = '${ctx}/cell/CPE/queryWanDnsInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};
				
				vm.isWanDnsLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.wanDnsEnable == 'NotSync') {
						vm.networkForm.wanDnsEnable = 'NotSync';
						vm.wanDnsTips = 'Not sync';
					}else if(data.wanDnsEnable == 'NotSupport') {
						vm.networkForm.wanDnsEnable = 'NotSupport';
						vm.wanDnsTips = 'Not support';
					}else {
						if(data["wanDnsEnable"] == "1" || data["wanDnsEnable"] == "0") {
							vm.networkForm.wanDnsEnable = data["wanDnsEnable"];
							vm.networkForm.wanDnsMode = data["wanDnsMode"];
							vm.networkForm.wanDnsPriAddress = data["wanDnsPriAddress"];
							vm.networkForm.wanDnsSecAddress = data["wanDnsSecAddress"];
						}
						vm.wanDnsTips = '';
					}
					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(vm.$refs.network, ['wanDnsEnable','wanDnsMode','wanDnsPriAddress','wanDnsSecAddress']);
					});

					vm.isWanDnsLoading = false;
				});
			},
			refreshAPWiFi() {
				var vm = this,
					url = '${ctx}/cell/CPE/setting/queryApWifiInfo.action',
					params = {
						cpeCode: vm.cpeCode
					};
				
				vm.isAPWifiLoading = true;
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data.wanDnsEnable == 'NotSync') {
						vm.networkForm.apWlanEnable = 'NotSync';
						vm.apwifiTips = 'Not sync';
					}else if(data.wanDnsEnable == 'NotSupport') {
						vm.networkForm.apWlanEnable = 'NotSupport';
						vm.apwifiTips = 'Not support';
					}else {
						if(data["apWlanEnable"] == "1" || data["apWlanEnable"] == "0") {
							[	'apWlanEnable',
								'ap4MasterSsid','ap4MasterEncryption','ap4MasterPassPhrase',
								'ap5MasterSsid','ap5MasterEncryption','ap5MasterPassPhrase',
								'ap6MasterSsid','ap6MasterEncryption','ap6MasterPassPhrase',
								'ap4Channel','ap4BandWidth','ap5Channel','ap5BandWidth','ap6Channel','ap6BandWidth','country'
							].map(function(code){
								vm.networkForm[code] = data[code];
							});
						}
						vm.apwifiTips = '';
					}
					// 初始化form原始值
					vm.$nextTick(function(){
						initForm(
							vm.$refs.network, 
							[	'apWlanEnable',
								'ap4MasterSsid','ap4MasterEncryption','ap4MasterPassPhrase',
								'ap5MasterSsid','ap5MasterEncryption','ap5MasterPassPhrase',
								'ap6MasterSsid','ap6MasterEncryption','ap6MasterPassPhrase',
								'ap4Channel','ap4BandWidth','ap5Channel','ap5BandWidth','ap6Channel','ap6BandWidth','country'
							]
						);
					});

					vm.isAPWifiLoading = false;
				});
			},

			showCellLock() {
				let vm = this;

				vm.resetLockForm();
				vm.cell5gDlShow = true;
				vm.$nextTick(function(){
					vm.$refs.cellForm.clearValidate();
				});
			},
			showBandLock() {
				let vm = this;

				vm.resetLockForm();
				vm.band5gDlShow = true;
				vm.$nextTick(function(){
					vm.$refs.bandForm.clearValidate();
				});
			},
			resetLockForm() {
				var vm = this;

				Object.assign(vm.cellForm,{
					rat: '',
					band: '',
					earfcn: '',
					pci: ''
				});
				Object.assign(vm.bandForm,{
					rat: '',
					band: ''
				});
			},
			addCellLock() {
				var vm = this;

				vm.$refs.cellForm.validate(function(r){
					if(r) {
						vm.cellLockList.push(Object.assign({},vm.cellForm));
						vm.cell5gDlShow = false;
					}
				});
			},
			addBandLock() {
				var vm = this;
				
				vm.$refs.bandForm.validate(function(r){
					if(r) {
						vm.bandLockList.push(Object.assign({},vm.bandForm));
						vm.band5gDlShow = false;
					}
				});
			},
			deleteCellLock(idx) {
				var vm = this;

				vm.cellLockList.splice(idx,1);
			},
			deleteBandLock(idx) {
				var vm = this;

				vm.bandLockList.splice(idx,1);
			}
		},
		mounted() {
			var vm = this;
			eventBus.$off('cpe-data').$on('cpe-data',vm.init)
			
		}
	})
</script>