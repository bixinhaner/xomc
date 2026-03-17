<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.el-input__inner[readonly] {
		background-color: #f5f7fa;
	}
	.el-select .el-input__inner[readonly] {
		background-color: #fff;
	}
	.alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	.titleStyML{
		margin-left: 50px;
	}
	.deviceTableBox{
		margin:20px 0px 0px 76px;
		display: flex;
		height:372px;
		background:#FFFFFF;
	}
	.deviceSpecifiedBox{
		width: 200px;
		height: 370px;
		border:1px solid #E9E9E9;
		border-right: none;
	}
	.deviceSpecifiedTitle{
		height: 36px;
		width: 200px;
		font-size: 12px;
		line-height: 36px;
		text-align: center;
		background: #F6F7FB;
		box-sizing: border-box;
		border-bottom:1px solid #E9E9E9;
	}
	.specifiedTypeBox{
		flex: 1;
		padding-top: 30px;
		padding-left: 40px;
	}
	.specifiedTypeBox .el-radio__label{
		font-size: 12px !important;
	}
	.specifiedTypeBox .el-radio+.el-radio{
		margin-left: 0px;
		display: block;
	}
	.productBoxCls .el-input__inner{
		height: 26px !important;
	}
	.changeDeviceTableWarp{
		width: calc(100% - 200px)!important;
		font-size: 12px !important;
	}
	.changeDeviceTableWarp .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	.changeDeviceTableWarp .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	.changeDeviceTableWarp .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	.tableTitles{
		margin-left: 45px;
		padding-top: 20px;
		font-size: 14px;
	}
	.el-radio__input.is-checked+.el-radio__label,
	.el-radio{
		color:#333333;
	}
	.footer{
		width:100%;
		border-top:1px solid #E9E9E9;
		position:absolute;
		bottom:0px;
		height:50px;
		line-height:50px;
		background:#FFFFFF;
		z-index:99;
	}
	.footer div{
		padding-left: 40px;
	}
	.el-form-item__error{
		padding-top: 0px;
	}
	.closeSlideBtn{
		position:absolute;
		top:20px;
		right:20px;
		overflow: hidden;
	}
	.mainWarp{
		overflow:hidden;
	}
	.basicBox{
		padding-top:40px;
	}
	.basicInfo{
		padding: 20px 76px 0;
	}
	.allSelectTable{
		border:1px solid #E9E9E9;
		margin-right:45px;
	}
	.testCpeCode{
		margin-left:76px;
		margin-top:0px;
		margin-bottom: 0;
		
	}
	.executeTypeWarp{
		padding: 20px 60px 80px;
	}		
	/* 参数配置 */
    .nav-flex-item {
		flex: auto;
		overflow: auto;
	}
    .nav-flex-item {
		flex: auto;
		overflow: auto;
	}
	.nav-tabs {
		border-right: 1px solid #eee;
		min-width: 138px;
	}
	.nav-tabs div {
		height: 36px;
		line-height: 36px;
		padding: 0 20px;
		cursor: pointer;
		border-bottom: 1px solid #eee;
		word-break: keep-all;
	}
	.nav-tabs div.active {
		color: #4D84FF;
		background-color: #EDF6FF;
	}
	.nav-title {
		padding: 0 20px;
		height: 40px;
		line-height: 40px;
		color: #363b4e;
		font-weight: bold;
		border-bottom: 1px solid #eee;
	}
	.nav-main {
		display: flex;
		flex-direction: column;
		flex: auto;
		overflow: auto;
	}
	.nav-content {
		padding: 30px 40px;;
	}
	.nav-content .el-form-item__label, .label-flex .el-form-item__label {
		text-align: left;
		line-height: 28px;
	}
	.nav-operations {
		height: 48px;
		line-height: 48px;
		padding-left: 40px;
		border-top: 1px solid #eee;
	}
	.formContentWarp .el-form-item {
		display: flex; 
		margin: 0 100px 22px 0;
	}
	.lteContentWarp {
		display: flex; 
	}
	.formWarp .titleWarp {
		display: flex;
		margin-bottom: 22px;
	}
	.formWarp .titleWarp span{
		font-size: 14px;
		color: #333333;
		font-weight: bold;
	}
	.formWarp .circleTip {
		width: 7px;
		height: 7px;
		background: #333;
		border-radius: 50%;
		margin: 5px 6px 0 0;
	}
	.formWarp .formContentWarp {
		display: flex;
		margin-left: 14px;
	}
	.formWarp .el-form-item .el-switch {
		margin-top: 4px;
	}
	.earfcnPciWarp{
		margin-left: 120px;
		width: 720px;
		height: auto;
		overflow: auto;
		margin-bottom:30px;
		display: flex;
	}

	.ipTable thead tr th:first-child .cell {
		display: none;
	}
	.ipModalWarp .el-dialog__body {
		padding: 30px;
		position: relative;
		min-height: 130px;
	}
	.ipModalWarp .el-dialog__body .el-form .el-form-item__label { 
		line-height: 28px;
	}
	.addErrorTip {
		color: #FA5555;
		font-size: 12px;
		margin-top: 5px;
	}
	.ipModalWarp .editErrorTip {
		color: #FA5555;
		font-size: 12px;
		margin-left: 50px;
		margin-top: 4px;
	}
	.ipModalWarp .el-form-item__error {
		left: 0;
	}
	.ipAddressWarp {
		margin-left: 50px;
		height: auto;
		overflow: auto;
		margin-bottom:40px;
	}
	.ipWarpFotter {
		position: absolute;
		bottom: 25px;
	}
	.ipTip {
		font-size: 15px;
		margin-left: 20px; 
		color: #4D84FF;
	}
	.iplistTitle {
		position: relative;
		height: 40px;
		line-height: 40px;
		margin-left: 14px;
	}
	.systemWarp .el-form-item .el-form-item__label {
		width: 200px;
		margin-right: 0;
		margin-left: 14px;
	}
	.systemWarp .el-form-item .el-form-item__content {
		margin-left: 200px;
	}
	.minWidthWarp .el-form-item .el-form-item__label {
		width: 140px;
	}	
	.minWidthWarp .el-form-item .el-form-item__content {
		margin-left: 140px;
	}
	.lteWidthWarp .el-form-item {
		margin-bottom: 22px;
	}
	.lteWidthWarp .el-form-item .el-form-item__label {
		width: 120px;
	}
	.lteWidthWarp .el-form-item .el-form-item__content {
		display: inline-block;
	}
	.earfcn-pci-line {
		display: flex;
		align-items: center;
	}
	
	.formContentWarp .el-form-item .el-form-item__content {
		margin-left: 30px;
	}
	.lanWarp .el-form-item .el-form-item__label {
		margin-right: 30px;
	}
	.systemWarp .el-form-item,
	.lanWarp .el-form-item {		
		margin-bottom: 22px;
	}
	.systemWarp .el-form-item__error{
		margin-left: 15px;
	}
	.apnWarp .el-form-item__error{
		padding-top: 3px;
	}
	.apnWarp {
		margin-left: 14px;
	}
	.apnWarp .el-form-item {
		margin: 0 30px 22px 0;
	}
	.apnWarp .el-form-item {
		margin: 0 30px 15px 0;
	}
	.apnWarp .el-form-item .el-form-item__label{
		margin-right: 30px;
	}
	.apnWarp .el-form-item .el-form-item__content{
		font-size: 14px;
	}
	.layerTwo .el-form-item {
		display: flex;
		margin: 0 72px 22px 0;
	}
	.form-suffix { 
		border:1px solid #4D84FF; 
		display:inline-block;
		padding:0 10px;
		background:#F2F6FF;	
		width: auto;
	}
	.suffixItem { 		
		margin-right:10px;
		margin-bottom:10px 
	}
	.form-suffix .text { 
		font-size:12px;
		color:#333333;
		width: auto;
	}
	/*导入  */
	.importCard .w400{
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
		margin-left:16px;
	}
	.importCard .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	.importCard .el-dialog__footer{
		padding:20px 0 !important;
		text-align:left;
	}
	.el-dialog__footer .el-button:first-child{
		margin:20px 0 0 30px;
	}
	.importCard .importFooter{
		border-top:1px solid #DCDFE6;	
	}
	.importCard .el-dialog__header .el-icon-close{
		top: 0px;
		font-size: 18px;
	}
	.selectScanMode .el-input__inner{
		height:28px !important;
		width:200px;
	}
	.selectOptions .el-input{
		height:28px !important;
	}
	.selectOptions .el-input__inner{
		height:28px !important;		
	}
	/* apn */
	.apnTitleWarp {
		display: flex;
		padding: 10px 0 20px;
	}
	.apnTitleBox {
		font-size: 14px;
		border: 1px solid #4D84FF;
		cursor: pointer; 
		width: 120px;
		height: 26px;
		line-height: 26px;
		display: inline-block;
		text-align: center;
		border-radius: 3px;
	}
	.apnTitleText {
		font-size: 12px;
		margin: 6px 20px; 
		color: #999999;
	}
	
	.apnConfigBox {
		display: flex;
	}
	 .padding-left-px {
       padding-left: 15px; 
    }
    .padding-left-px .el-form-item__error {
		left: 0 !important;
	}
    .padding-left-px .el-form-item {
        display: inline-block;
        margin-right: 150px;
        margin-bottom: 28px !important;
    }
    .padding-left-px .el-form-item .el-form-item__label {
        line-height: 26px;
    }
    
    .select-80px .el-input__inner,
    .select-height-26  .el-input__inner{
        max-height: 26px;
        height: 26px;
    }
    .select-80px .el-input {
        width: 80px;
    }
    .select-100px .el-input {
        width: 100px;
    }
    .prefix-title {
        display: inline-block;
        height: 30px;
        line-height: 30px;
        font-weight: bold;
        font-size: 14px;
       
     }
     .checkboxFirst .el-form-item__content{
     	margin-left: 0px !important;
     }
     .vlanIdTip{
     	color: #999999;
     	font-size: 12px;
     }
     .slide-position-top .slide-content{
		padding:0 !important;
	 }
	 .el-card__header{
		border-bottom:none !important;
	 }
	 .el-card__body{
	 	border: 1px solid #E9E9E9;
	 }
	 .vlanIdTip{
     	margin-right: 0px !important;
     	margin-bottom: 0px !important;
     	margin-top: -20px !important;
     	color: #999999;
     	font-size: 12px;
     }
</style>
<div class="flex-ctn mainWarp" id="batchConfigParams" style="height:100%;">
	<div id='temp_add_close' class="placeholder-bt closeSlideBtn" placeholder="<%=rb.getString("GuanBi")%>" style="top: 14px; right: 30px;">		
		<span class="el-icon el-icon-circle-close" @click="closePanel"></span>
	</div> 
	<el-form ref="addform" :model="form" :rules="formRules" label-position="left" :hide-required-asterisk='true' style="height:100%;">
		
		<el-form-item class="testCpeCode" prop="cpeCodes"></el-form-item>
		
		<!-- Parameter Configuration -->
		<div style="position: relative; display:flex;">
			<!-- 导入，导出 -->
			<div v-show="AddBtnShow" class="circleIcon placeholder-bt importBtn" placeholder="<%=rb.getString("DaoRu")%>" @click="importFileBtn" style="top: 14px; right: 130px;">		
				<span class="el-icon el-icon-circle-import"></span>
			</div>
			<div v-show="AddBtnShow" class="circleIcon placeholder-bt exportBtn" placeholder="<%=rb.getString("DaoChu")%>" @click="exportFileBtn" style="top: 14px; right: 70px;">			
				<span class="el-icon el-icon-circle-export"></span>
			</div>
		</div>
		
    	<div class="group" style="width:100%; height: 100%;">
            <div style="width: 100%; display: flex; flex-direction: row; flex: auto;height:100%;">              
                <!-- 导航区域 -->
				<div class="nav-tabs">
					<div v-for="(item,index) in tabs"  :class="{active: activeCode == item.code}" @click="tabClick(item)">{{item.text}}</div>
				</div>
				<!-- 右侧详情区域 -->
				<div class="nav-main">
					<!-- 内容载入区域 -->
					<div class="nav-flex-item nav-content">
						<div class="formWarp">
							<!-- Network -->
							<div v-show="activeCode=='network'">
								<!-- WLAN  -->
			 					<div>
				 					<div class="titleWarp">
				 						<div class="circleTip"></div>
				 						<span>WLAN</span>
				 					</div>
				 					<div style="display: none;">
										<el-form-item prop="wifiSsid"></el-form-item>

										<el-form-item prop="wifiEncryption"></el-form-item>
										<el-form-item prop="wifiPassphrase"></el-form-item>
			
										<el-form-item prop="wifi1Enable"></el-form-item>
										<el-form-item prop="wifi1Ssid"></el-form-item>

										<el-form-item prop="wifi1Encryption"></el-form-item>
										<el-form-item prop="wifi1Passphrase"></el-form-item>
			
										<el-form-item prop="wifi2Enable"></el-form-item>
										<el-form-item prop="wifi2Ssid"></el-form-item>

										<el-form-item prop="wifi2Encryption"></el-form-item>
										<el-form-item prop="wifi2Passphrase"></el-form-item>
			
										<el-form-item prop="wifi3Enable"></el-form-item>
										<el-form-item prop="wifi3Ssid"></el-form-item>

										<el-form-item prop="wifi3Encryption"></el-form-item>
										<el-form-item prop="wifi3Passphrase"></el-form-item>
									</div>	 					
				 					<div class="formContentWarp" >
				 						<el-form-item label="WiFi" prop="wlanEnable">
											<el-select v-model="form.wlanEnable" :disabled="readonly" class="selectOptions" @change="wlanEnableChange"> 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="Frequency(Channel)" prop="wifiChannel">
											<el-select v-model="form.wifiChannel" :disabled="readonly" size="mini" :readonly="readonly">
												<el-option label="" value=""></el-option>
												<el-option label="AUTO" value="AUTO"></el-option>
												<el-option label="2.412GHz(Channel 1)" value="1"></el-option>
												<el-option label="2.417GHz(Channel 2)" value="2"></el-option>
												<el-option label="2.422GHz(Channel 3)" value="3"></el-option>
												<el-option label="2.427GHz(Channel 4)" value="4"></el-option>
												<el-option label="2.43GHz(Channel 5)" value="5"></el-option>
												<el-option label="2.437GHz(Channel 6)" value="6"></el-option>
												<el-option label="2.442GHz(Channel 7)" value="7"></el-option>
												<el-option label="2.447GHz(Channel 8)" value="8"></el-option>
												<el-option label="2.452GHz(Channel 9)" value="9"></el-option>
												<el-option label="2.457GHz(Channel 10)" value="10"></el-option>
												<el-option label="2.462GHz(Channel 11)" value="11"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="Channel Bandwidth" prop="wifiBandwidth">
											<el-radio-group v-model="form.wifiBandwidth" style="margin-top: 6px;" :disabled="isWifiClosed && readonly">
												<el-radio label="0">20M</el-radio>
												<el-radio label="1" style="margin-left: 30px;">20/40M</el-radio>
											</el-radio-group>
										</el-form-item>
				 					</div>
				 					
				 					<div style="margin-left: 14px; margin-bottom: 22px;">		 					
				 						<div style="margin-bottom: 10px; font-size: 14px; ">MBSSID</div>
										<el-ctable height="177" ref="taskList" :data="tbData" :pagination="false" style="border: 1px solid #E9E9E9;width:80%; ">
											<el-table-column width="80">
												<template slot-scope="scope">
													<i class="el-icon el-icon-operation-edit" @click="modifyWifi(scope.row)" v-show="!isWifiClosed && editBtnShow"></i>
												</template>
											</el-table-column>
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
										</el-ctable>
				 					</div>
			 					</div>
			 					<!--DMZ  -->	
			 					<div>
				 					<div class="titleWarp">
				 						<div class="circleTip"></div>
				 						<span>DMZ</span>
				 					</div>				 							 					
				 					<div class="formContentWarp">
										<el-form-item label="DMZ Enable" prop="dmzEnable">
											<el-select v-model="form.dmzEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="DMZ Address" prop="dmzHostAddress" >
											<el-input v-model="form.dmzHostAddress" maxlength="100" :readonly="readonly"></el-input>
										</el-form-item>
				 					</div>
			 					</div>	
			 					<!--LAN  -->	
			 					<div>
				 					<div class="titleWarp">
				 						<div class="circleTip"></div>
				 						<span>LAN</span>
				 					</div>			 							 					
				 					<div style='margin-left: 14px;' class="lanWarp">
										<el-form-item label="LAN <%=rb.getString("JieKou")%>" prop="lanEnable">
											<el-select v-model="form.lanEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
				 					</div>
			 					</div> 	
							</div>
							
							<!-- LTE -->
							<div v-show="activeCode=='lte'">
								<!-- PCI Lock  -->
			 					<div>
				 					<div class="titleWarp">
				 						<div class="circleTip"></div>
				 						<span><%=rb.getString("SuoPCI")%></span>
				 					</div>				 							 					
				 					<div style="margin-left: 14px;" class="lteWidthWarp">
				 						<el-form-item label="<%=rb.getString("SaoMiaoFangShi")%>" prop="scanMode">
											<el-select v-model="form.scanMode" :disabled="readonly" class="selectScanMode"> 
												<el-option label="" value=""></el-option>
												<el-option label="Full Band" value="fullband"></el-option>
												<el-option label="Band/Frequency Preferred" value="freqpreferred"></el-option>
												<el-option label="PCI lock" value="pcilock"></el-option>
												<el-option label="PCI Only Lock" value="pcionlylock"></el-option>
											</el-select>
										</el-form-item>	
										<el-form-item v-show="false" prop="pci">
											<el-input v-model="form.pci"></el-input>
										</el-form-item>	
										<!-- earfcn -->
										<div v-if="form.scanMode == 'freqpreferred'">
											<el-form-item class="earfcn-pci-line" label='Earfcn' style="margin-bottom: 3px;">
												<div class="mainWarp">						
													<div class='newAddBtn' >
														<el-input v-model='form.onlyEarfcn'></el-input>
														<i v-show="AddBtnShow" class="el-icon el-icon-plus" @click='addEarfcnBtn' style="margin-left: 10px;"></i>
													</div>							
												</div>
											</el-form-item>
											<div class="earfcnPciWarp">
												<el-form-item class='suffixItem' v-for='(domain,index) in form.earfcnList' style='line-height:16px;'>
													<div class='form-suffix'>
														<span class='text'>Earfcn : {{domain}}</span>
														<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeEarfcn(domain)'></span>
													</div>
												</el-form-item> 
												<p class="addErrorTip">{{earfcnErrorMessage}}</p> 
											</div>											
										</div>
										<!-- earfcn and pci -->
										<div v-if="form.scanMode == 'pcilock'">
											<el-form-item class="earfcn-pci-line" label='Earfcn : PCI' style="margin-bottom: 3px;">
												<div class="mainWarp">						
													<div class='newAddBtn'>
														<el-input v-model='form.earfcnStart' style='width:200px;'></el-input> — <el-input v-model='form.pciEnd' style='width:200px;'></el-input>		 																
														<i v-show="AddBtnShow" class="el-icon el-icon-plus addIpBtn" @click='addPcilockBtn' style="margin-left: 10px;"></i>
													</div>							
												</div>
											</el-form-item>
											<div class="earfcnPciWarp">
												<el-form-item class='suffixItem' v-for='(domain,index) in form.earfcnPciList' style='line-height:16px;'>
													<div class='form-suffix'>
														<div style="display: flex;">
															<span class='text'>Earfcn : {{domain.startValue}}</span>
															<span class='text' style="margin: 0 10px 0 15px;">PCI : {{domain.endValue}}</span>
															<span style='font-size:16px;margin-top:6px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeEarfcnPci(domain)'></span>															
														</div>														 
													</div>
												</el-form-item> 
												<p class="addErrorTip">{{earfcnPciErrorMessage}}</p>
											</div>											
										</div>
										<!--  pci -->
										<div v-if="form.scanMode == 'pcionlylock'">
											<el-form-item class="earfcn-pci-line" label='PCI' style="margin-bottom: 3px;">
												<div class="mainWarp">						
													<div class='newAddBtn'>
														<el-input v-model='form.onlyPci'></el-input>
														<i v-show="AddBtnShow" class="el-icon el-icon-plus" @click='addPciBtn' style="margin-left: 10px;"></i>
													</div>							
												</div>
											</el-form-item>
											<div class="earfcnPciWarp">
												<el-form-item class='suffixItem' v-for='(domain,index) in form.pciList' style='line-height:16px;'>
													<div class='form-suffix'>
														<span class='text'>PCI : {{domain}}</span>
														<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removePci(domain)'></span>
													</div>
												</el-form-item> 
												<p class="addErrorTip">{{pciErrorMessage}}</p>
											</div>
										</div>																	
				 					</div>				 									 					
			 					</div>
								<!-- APN 批量配置参数时，apn 为非必填项 -->
								<div v-show="false">
									<div class="apnTitleWarp">
										<div @click="apnPlicyClick" class="apnTitleBox">
											<span class="el-icon el-icon-readTemplate"></span>
											<span style="color: #666666;">APN Policy</span>
										</div>
										<span class="apnTitleText"><%=rb.getString("APNTiShi")%></span>
									</div>
				 					<div class="titleWarp" style="margin-bottom: 0px;">
				 						<div class="circleTip"></div>
				 						<span>APN List</span>
				 					</div>			 							 					
				 					<div class=" padding-left-px">
				 						<div>								
							                <el-form-item prop="apn1Enable" label-width="30" style="margin-right:22px;" class="checkboxFirst">
							                    <el-checkbox v-model="form.apn1Enable" true-label="1" false-label="0" ></el-checkbox>
							                </el-form-item>
							                
							                <span class="prefix-title">APN1</span>
							                
							                <el-form-item prop="apn1Name" label="APN Name" label-width="100" style="margin: 20px;">
							                    <el-select v-model="form.apn1Name" class="select-height-26" 
							                     :disabled="form.apn1Enable != '1'" >
							                        <el-option v-for="item in apnOptions" :label="item.text" :value="item.value" :disabled="item.value === form.apn2Name || item.value === form.apn3Name || item.value === form.apn4Name"></el-option>
							                    </el-select>
							                </el-form-item>
							                
							                <el-form-item prop="apn1BearType" label="Bear Type" label-width="100" style="margin-left:30px;margin-right: 40px;">
							                    <el-select v-model="form.apn1BearType" class="select-80px" @change="bearTypeEnableChange"
							                        :disabled="form.apn1Enable != '1'">							                       
													<el-option label="MGMT" value="1" :disabled="bearTypeDisabled"></el-option>
                        					 		<el-option label="DATA" value="0"></el-option> 
                        					 		<el-option label="VOIP" value="2"></el-option>      
 												</el-select>							                    
						                	</el-form-item>	
						                	
						                	<el-form-item v-show="form.apn1BearType == '1'" prop="apn1Default" label-width="30" style="margin-right:22px;" class="checkboxFirst">
							                    <el-checkbox v-model="form.apn1Default" disabled true-label="1" false-label="0">Default</el-checkbox>
							                </el-form-item>
							                
							              	<el-form-item prop="apn1VlanId" v-show="form.apn1BearType == '0'" label="VLAN ID" label-width="80" style="margin-right: 10px;">
							                    <el-input v-model="form.apn1VlanId" :disabled="form.apn1Enable == '0'" style="width:210px;"></el-input>
							              	</el-form-item>
							              	 
										</div>		
										<!--apn2  -->
										<div>								
											<el-form-item prop="apn2Enable" label-width="30" style="margin-right:22px;" class="checkboxFirst">
							                    <el-checkbox v-model="form.apn2Enable" true-label="1" false-label="0" ></el-checkbox>
							                </el-form-item>
							                
							                <span class="prefix-title">APN2</span>
							                
							                <el-form-item prop="apn2Name" label="APN Name" label-width="100" style="margin: 0 20px;">
							                    <el-select v-model="form.apn2Name" class="select-height-26" 
							                     :disabled="form.apn2Enable != '1'">
							                        <el-option v-for="item in apnOptions" :label="item.text" :value="item.value" :disabled="item.value === form.apn1Name || item.value === form.apn3Name || item.value === form.apn4Name"></el-option>
							                    </el-select>
							                </el-form-item>
							                <el-form-item prop="apn2BearType" label="Bear Type" label-width="100" style="margin-left:30px;margin-right: 40px;">
							                    <el-select v-model="form.apn2BearType" class="select-80px" @change="bearTypeEnableChange"
							                        :disabled="form.apn2Enable != '1'">	
							                        <el-option label="MGMT" value="1" :disabled="bearTypeDisabled"></el-option>						                        													
                        							<el-option label="DATA" value="0"></el-option>                        							
                        							<el-option label="VOIP" value="2"></el-option>
							                    </el-select>							                    
						                	</el-form-item>	
						                	
							           	 	<el-form-item v-show="form.apn2BearType == '1'" prop="apn2Default" label-width="30" style="margin-right:22px;" class="checkboxFirst">
							                    <el-checkbox v-model="form.apn2Default" disabled true-label="1" false-label="0">Default</el-checkbox>
							                </el-form-item> 							               
							                
							              	<el-form-item prop="apn2VlanId" v-show="form.apn2BearType == '0'" label="VLAN ID" label-width="80" style="margin-right: 10px;">
							                    <el-input v-model="form.apn2VlanId" :disabled="form.apn2Enable == '0'" style="width:210px;"></el-input>
							              	</el-form-item>

										</div>	
										<!--apn3  -->
										<div>								
											<el-form-item prop="apn3Enable" label-width="30" style="margin-right:22px;" class="checkboxFirst">
							                    <el-checkbox v-model="form.apn3Enable" true-label="1" false-label="0" ></el-checkbox>
							                </el-form-item>
							                
							                <span class="prefix-title">APN3</span>
							                
							                <el-form-item prop="apn3Name" label="APN Name" label-width="100" style="margin: 0 20px;">
							                    <el-select v-model="form.apn3Name" class="select-height-26" 
							                     :disabled="form.apn3Enable != '1'">
							                        <el-option v-for="item in apnOptions" :label="item.text" :value="item.value" :disabled="item.value === form.apn1Name || item.value === form.apn2Name || item.value === form.apn4Name"></el-option>
							                    </el-select>
							                </el-form-item>
							                <el-form-item prop="apn3BearType" label="Bear Type" label-width="100" style="margin-left:30px;margin-right: 40px;">
							                    <el-select v-model="form.apn3BearType" class="select-80px" @change="bearTypeEnableChange"
							                        :disabled="form.apn3Enable != '1'">
													<el-option label="MGMT" value="1" :disabled="bearTypeDisabled"></el-option>
                        							<el-option label="DATA" value="0"></el-option> 
                        							<el-option label="VOIP" value="2"></el-option>       
 												</el-select>							                    
						                	</el-form-item>	
						                	
											<el-form-item v-show="form.apn3BearType == '1'" prop="apn3Default" label-width="30" style="margin-right:22px;" class="checkboxFirst">
							                    <el-checkbox v-model="form.apn3Default" disabled true-label="1" false-label="0">Default</el-checkbox>
							                </el-form-item> 
							                
							              	<el-form-item prop="apn3VlanId" v-show="form.apn3BearType == '0'" label="VLAN ID" label-width="80" style="margin-right: 10px;">
							                    <el-input v-model="form.apn3VlanId" :disabled="form.apn3Enable == '0'" style="width:210px;"></el-input>
							              	</el-form-item>

										</div>
										<!--apn4  -->
										<div>								
											<el-form-item prop="apn4Enable" label-width="30" style="margin-right:22px;" class="checkboxFirst">
							                    <el-checkbox v-model="form.apn4Enable" true-label="1" false-label="0" ></el-checkbox>
							                </el-form-item>
							                
							                <span class="prefix-title">APN4</span>
							                
							                <el-form-item prop="apn4Name" label="APN Name" label-width="100" style="margin: 0 20px;">
							                    <el-select v-model="form.apn4Name" class="select-height-26" 
							                     :disabled="form.apn4Enable != '1'">
							                        <el-option v-for="item in apnOptions" :label="item.text" :value="item.value" :disabled="item.value === form.apn1Name || item.value === form.apn2Name || item.value === form.apn3Name"></el-option>
							                    </el-select>
							                </el-form-item>
							                <el-form-item prop="apn4BearType" label="Bear Type" label-width="100" style="margin-left:30px;margin-right: 40px;">
							                    <el-select v-model="form.apn4BearType" class="select-80px" @change="bearTypeEnableChange"
							                        :disabled="form.apn4Enable != '1'">							                        
							                       	<el-option label="MGMT" value="1" :disabled="bearTypeDisabled"></el-option>
                        							<el-option label="DATA" value="0"></el-option>
                        							<el-option label="VOIP" value="2"></el-option>  
							                    </el-select>							                    
						                	</el-form-item>	
						                	
											<el-form-item v-show="form.apn4BearType == '1'" prop="apn4Default" label-width="30" style="margin-right:22px;" class="checkboxFirst">
							                    <el-checkbox v-model="form.apn4Default" disabled true-label="1" false-label="0">Default</el-checkbox>
							                </el-form-item> 							               
							                
							              	<el-form-item prop="apn4VlanId" v-show="form.apn4BearType == '0'" label="VLAN ID" label-width="80" style="margin-right: 10px;">
							                    <el-input v-model="form.apn4VlanId" :disabled="form.apn4Enable == '0'" style="width:210px;"></el-input>
							              	</el-form-item>

										</div>
										<div v-show="bearTypeOnlyShow" style="margin-bottom:22px;color: #FA5555;"><%=rb.getString("XuanZeBearType")%></div>										
										<!-- end -->								
				 					</div>
			 					</div> 
								<!--WAN Config  -->	
			 					<div >
				 					<div class="titleWarp" style="margin-bottom: 22px;">
				 						<div class="circleTip"></div>
				 						<span>WAN Config</span>
				 					</div>																												
				 					<div class="padding-left-px">
							            <el-form-item prop="operMode" label="Mode" label-width="76">
							                <el-select v-model="form.operMode" disabled>
							                    <el-option label="NAT" value="0"></el-option>
							                    <el-option label="Router Mode" value="1"></el-option>
							                    <el-option label="Tunnel Mode" value="2"></el-option>
							                    <el-option label="Bridge Mode" value="3"></el-option>
							                </el-select>
							            </el-form-item>
							           							               
						                <el-form-item prop="apnGreType" label="GRE Type" label-width="80" >
						                    <el-select v-model="form.apnGreType" class="select-80px" 
						                       disabled>
						                        <el-option label="Layer2" value="1"></el-option>
						                    </el-select>
						                </el-form-item> 
							            <el-form-item  prop="greDestIPAddress" label="Destination IP" label-width="120">
							                <el-input v-model="form.greDestIPAddress"></el-input>
						            	</el-form-item>					            								               
						         	</div>	
						         	
						            <div class=" padding-left-px">
						            
						            	<div class="" v-show="form.apn1Enable == '1'">
							            	<span class="prefix-title">APN1</span> 
							            	<el-form-item prop="apn1BearType" label="Bear Type" label-width="100" style="margin-left:30px;margin-right: 150px;">
							                    <el-select v-model="form.apn1BearType" class="select-100px" 
							                        disabled>
							                        <el-option v-for="item in BearTypeOptions" :label="item.text" :value="item.value"></el-option>
							                    </el-select>							                    
						                	</el-form-item>
						                	<el-form-item v-show="form.apn1BearType != '1'" label="VPN Type" label-width="120" style="margin-right: 150px;">
							                    <el-select v-model="form.apn1VpnType" class="select-100px" 
							                       disabled>
							                        <el-option label="GRE" value="1"></el-option>
							                    </el-select>
							                </el-form-item>
							                
							            </div>
							            <div class="" v-show="form.apn2Enable == '1'">
							            	<span class="prefix-title">APN2</span> 
							            	<el-form-item prop="apn2BearType" label="Bear Type" label-width="100" style="margin-left:30px;margin-right: 150px;">
							                    <el-select v-model="form.apn2BearType" class="select-100px" 
							                        disabled>
							                        <el-option v-for="item in BearTypeOptions" :label="item.text" :value="item.value"></el-option>
							                    </el-select>							                    
						                	</el-form-item>
						                	<el-form-item v-show="form.apn2BearType != '1'" label="VPN Type" label-width="120" style="margin-right: 150px;">
							                    <el-select v-model="form.apn2VpnType" class="select-100px" 
							                       disabled>
							                        <el-option label="GRE" value="1"></el-option>
							                    </el-select>
							                </el-form-item>
							                
							            </div>
							            <div class="" v-show="form.apn3Enable == '1'">
							            	<span class="prefix-title">APN3</span> 
							            	<el-form-item prop="apn3BearType" label="Bear Type" label-width="100" style="margin-left:30px;margin-right: 150px;">
							                    <el-select v-model="form.apn3BearType" class="select-100px" 
							                        disabled>
							                        <el-option v-for="item in BearTypeOptions" :label="item.text" :value="item.value"></el-option>
							                    </el-select>							                    
						                	</el-form-item>
						                	<el-form-item v-show="form.apn3BearType != '1'" label="VPN Type" label-width="120" style="margin-right: 150px;">
							                    <el-select v-model="form.apn3VpnType" class="select-100px" 
							                       disabled>
							                        <el-option label="GRE" value="1"></el-option>
							                    </el-select>
							                </el-form-item>
							                
							            </div>
							            <div class="" v-show="form.apn4Enable == '1'">
							            	<span class="prefix-title">APN4</span> 
							            	<el-form-item prop="apn4BearType" label="Bear Type" label-width="100" style="margin-left:30px;margin-right: 150px;">
							                    <el-select v-model="form.apn4BearType" class="select-100px" 
							                        disabled>
							                        <el-option v-for="item in BearTypeOptions" :label="item.text" :value="item.value"></el-option>
							                    </el-select>							                    
						                	</el-form-item>
						                	<el-form-item v-show="form.apn4BearType != '1'" label="VPN Type" label-width="120" style="margin-right: 150px;">
							                    <el-select v-model="form.apn4VpnType" class="select-100px" 
							                       disabled>
							                        <el-option label="GRE" value="1"></el-option>
							                    </el-select>
							                </el-form-item>
							                
							            </div>							            
						            </div>
						            <!-- APN1 -->
			 					</div> 
							</div>
							<!-- System -->
							<div v-show="activeCode=='system'">
								<!-- Web Access  -->
			 					<div >
				 					<div class="titleWarp">
				 						<div class="circleTip"></div>
				 						<span>Web Access</span>
				 					</div>				 							 					
				 					<div class="systemWarp">
				 						<el-form-item label="Password" prop="uiPassword">
											<el-input v-model="form.uiPassword" maxlength="100" :readonly="readonly"></el-input>
										</el-form-item>	
										<el-form-item label="Https<%=rb.getString("SheZhiKaiGuan")%>" prop="httpsEnable">
											<el-select v-model="form.httpsEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="HttpsWan<%=rb.getString("SheZhiKaiGuan")%>" prop="httpsWanEnable">
											<el-select v-model="form.httpsWanEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="Access Control <%=rb.getString("SheZhiKaiGuan")%>" prop="wanAccEnable">
											<el-select v-model="form.wanAccEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item v-show="false" prop="rangeIp">
											<el-input v-model="form.rangeIp"></el-input>
										</el-form-item>	
										<div class="iplistTitle" style="width:81%; margin-top: -10px;">
											<span style="font-size: 14px; color: #333333;"><%=rb.getString("FangWenKongZhiLieBiao")%></span>
											<div class="circleIcon" style="top:0px;" v-show="AddBtnShow">
												<span class="el-icon el-icon-circle-add" @click="systemAddIp" ></span>
												<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
											</div>
										</div>	
										<div style="border:1px solid #E9E9E9;margin-bottom:30px;margin-left: 14px; width:80%;" class="ipTable">
											<el-table ref="ctableIp" :data="ipTableData.slice((currentPage-1)*pageSize,currentPage*pageSize)" height="200" border>
												<el-table-column label="" type="index" min-width="50"></el-table-column>
												<el-table-column label='' width="70" >
													<template slot-scope="scope">
														<div v-show="AddBtnShow" class="el-icon el-icon-operation-edit" title="<%=rb.getString("GengXin") %>" @click="updateIp(scope.row,scope.$index)"></div>
														<div v-show="AddBtnShow" class="el-icon el-icon-operation-delete" title="<%=rb.getString("ShanChu") %>" @click="deleteIp(scope.row,scope.$index)"></div>									
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
												:total="ipTableData.length">
											</el-pagination>
										</div>							
				 					</div>					 								 									 					
			 					</div>
			 					<!-- Ping Watchdog  -->
			 					<div style="padding-bottom: 40px;">
				 					<div class="titleWarp">
				 						<div class="circleTip"></div>
				 						<span>Ping Watchdog</span>
				 					</div>				 							 					
				 					<div class="systemWarp">
				 						<el-form-item label="Ping Watchdog <%=rb.getString("KaiGuan")%>" prop="watchDogEnable">
											<el-select v-model="form.watchDogEnable" :disabled="readonly" class="selectOptions" > 
												<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label="IP Address or URL to Ping" prop="watchDogPingIp" >
											<el-input v-model="form.watchDogPingIp" maxlength="45" placeholder="Length: 0-45" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="Ping Timeout(Seconds)" prop="watchDogPingTimeout">
											<el-input v-model="form.watchDogPingTimeout" maxlength="8" placeholder="Range: 1-65535" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="Ping Count" prop="watchDogPingCount">
											<el-input v-model="form.watchDogPingCount" maxlength="8" placeholder="Range: 1-65535" :readonly="readonly"></el-input>
										</el-form-item>
										<el-form-item label="Failure Count to Reboot" prop="watchDogFailureReboot">
											<el-input v-model="form.watchDogFailureReboot" maxlength="8" placeholder="Range: 1-65535" :readonly="readonly"></el-input>
										</el-form-item>									
				 					</div>				 									 					
			 					</div>
			 					
							</div>		 									
						</div>
					</div>
				</div>             
            </div>
        </div>  		
	</el-form>
	
	<div class='footer'>	
		<div>
			<el-button :disabled="readonly" type="primary" @click="formSubmit" :loading="submitLoading"><%=rb.getString("QueDing")%></el-button>
			<el-button :disabled="readonly" @click="cancel"><%=rb.getString("QuXiao")%></el-button>
		</div>		
	</div>
	
	<!-- 导入 弹窗 -->
	<el-dialog title="<%=rb.getString("DaoRu")%>" width="750px" :visible="showImportCard" class="importCard" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeImportParams">		
		<el-form label-position="left" ref="importRuleForm" :model='importRuleForm' :rules='importRules'>     		     			            
        	<el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="110px" prop="fileName">
               	<el-upload 
             		ref="upload"
             		:before-upload='beforeUpload' 
             		:on-success='checkFile' 
             		:on-change="fileChange"  
             		:show-file-list=false 	                  		
				    :action="importRuleForm.uploadFileUrl" 
				    :data="fileParams" 
				    name="uploadFile" 
				    accept=".xls,.xlsx"
				    :auto-upload="false">
					<el-input :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w400">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
					</el-input>								
					<a slot="trigger" ref="file_up"></a>
					<span class="fileAcceptTip"><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>
				</el-upload>	
         	</el-form-item> 
         	<el-form-item label="">
         		<div style='color:#999999;'>
         			<span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span>
         			<%=rb.getString("DaoRuWenJianTiShi")%>
					<span style="cursor:pointer;margin-left:10px;" @click="exportTemplate">
						<span style='' class='el-icon el-icon-common-download'></span>
						<span style='color:#363B4E;text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
					</span>
				</div> 
         	</el-form-item>      	    
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="uploadParams"><%=rb.getString("QueDing")%></el-button>
              <el-button @click="closeImportParams"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	
	<!-- MBSSID修改窗口 -->
	<el-dialog title="Modify MBSSID" :visible.sync="wifiMDLShow" :modal="true" :append-to-body="true" width="600">
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
				<el-form-item label="Security Mode" prop="wifiEncryption">
					<el-select v-model="mForm.wifiEncryption">
						<el-option label="OPEN" value="OPEN"></el-option>
						<el-option label="WPAPSK" value="WPA"></el-option>
						<el-option label="WPA2PSK" value="WPA2"></el-option>
						<el-option label="WPAPSK/WPA2PSK" value="WPAWPA2"></el-option>
					</el-select>
				</el-form-item>
				<div v-show="mForm.wifiEncryption != 'open'">
					<el-form-item v-show="mForm.wifiEncryption && mForm.wifiEncryption!='open' && false" label="WPA Algorithm">
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
	
    <!-- 新建ip弹窗 -->
	<el-dialog title="<%=rb.getString("TianJia")%>" id="addIpDialog" width='600px' :visible.sync='addIpShow' :append-to-body="true" :close-on-click-modal="false" @close='closeAddIp' class="ipModalWarp">
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
	<el-dialog title="<%=rb.getString("XiuGai")%>" id="editIpDialog" width='600px' :visible.sync='editIpShow' :append-to-body="true" :close-on-click-modal="false" @close='closeEditIp' class="ipModalWarp">
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
	
	<!-- apn policy select-->
	<el-dialog title="<%=rb.getString("QueRen")%>" :visible.sync="apnPolicyShow" :modal="true" :append-to-body="true" width="500" @close='closePolicy'>
		<el-form ref="selectForm" :model="selectForm" :rules="selectRules" class="label-flex" label-width="100" style="padding-left: 20px;">
			<el-form-item label="Apn Policy" prop="policy">
				<el-select v-model="selectForm.policy">
					<el-option v-for="item in policySelections" :key="item.value" :label="item.text" :value="item.value"></el-option>
				</el-select>
			</el-form-item>			
		</el-form>
		<div slot="footer">
			<el-button type="primary" @click="savePolicy"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closePolicy"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>

<script type="text/javascript">
	/**
	*  页面编辑和只读模式通过readonly控制
	*  校验规则也由readonly决定
	**/
	var batchConfigParamVue = new Vue({
		el: '#batchConfigParams',
		data(){
			var vm = this, reg = /^(\d+,?)+$/;
				var watchdogValid = function(rule,value,cb){
					var currVal = value,
						enable = vm.form.watchDogEnable;
					
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
				/* watchdog ip校验 */
				watchdogValidateIP = function(rule,value,cb) {
					var currVal = value,
						enable = vm.form.watchDogEnable;
					
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
					if(vm.mForm.wifiEnable == '1' && vm.mForm.wifiEncryption != 'open') {
						if(value && value.length >= 8 && value.length <= 64) {
							cb();
						}else {
							cb('<%=rb.getString("ZiFuChang")%>: 8 - 64');
						}
					}else {
						cb();
					}
				},
				validateFileName = function(rule,value,callback) {
		        	var value = vm.fileName; 
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else {
						callback();
					} 
				},
				validateApn1Name = function(rule,value,callback) {

					if(vm.form.apn1Enable == '1'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingXuanZe")%> APN Name');
						}else {
							callback();
						}
					}else{
						callback();
					}
										
				},	
				validateApn2Name = function(rule,value,callback) { 
					if(vm.form.apn2Enable == '1'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingXuanZe")%> APN Name');
						}else {
							callback();
						}
					}else{
						callback();
					}					
				},
				validateApn3Name = function(rule,value,callback) { 

					if(vm.form.apn3Enable == '1'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingXuanZe")%> APN Name');
						}else {
							callback();
						}
					}else{
						callback();
					}
										
				},	
				validateApn4Name = function(rule,value,callback) { 
					if(vm.form.apn4Enable == '1'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingXuanZe")%> APN Name');
						}else {
							callback();
						}
					}else{
						callback();
					}					
				},
				validateApn1VlanId = function(rule,value,callback){
					var arrVlan1 = (value || '').split(','), bool = true, repeated1 = false, isRepeated2 = false, isRepeated3 = false, isRepeated4 = false;						
					arrVlan1.map(function(item){if(reg.test(item.trim()) && item >= 0 && item <=4094 && item != 1) {bool = true; }else{ bool = false;}});
					if(arrVlan1.length > 1){
						for(var i = 0; i<arrVlan1.length;i++){
							if(arrVlan1.indexOf(arrVlan1[i]) != i || ( arrVlan1[i] == 1 && arrVlan1[i] < 0 && arrVlan1[i] > 4094)){
								repeated1 = true
							}
						}
					}
					if(vm.form.apn2Enable == '1' && vm.form.apn2BearType == '0'){
						if(vm.form.apn2VlanId === '' || vm.form.apn2VlanId === null || vm.form.apn2VlanId === undefined){
						}else{
							var arr2 = vm.form.apn2VlanId.split(',');
		                    arrVlan1.map(function(m){
	                            if(arr2.includes(m)) {
	                            	isRepeated2 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					
					if(vm.form.apn3Enable == '1' && vm.form.apn3BearType == '0'){
						if(vm.form.apn3VlanId === '' || vm.form.apn3VlanId === null || vm.form.apn3VlanId === undefined){
						}else{
							var arr3 = vm.form.apn3VlanId.split(',');
		                    arrVlan1.map(function(m){
	                            if(arr3.includes(m)) {
	                            	isRepeated3 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					
					if(vm.form.apn4Enable == '1' && vm.form.apn4BearType == '0'){
						if(vm.form.apn4VlanId === '' || vm.form.apn4VlanId === null || vm.form.apn4VlanId === undefined){
						}else{
							var arr4 = vm.form.apn4VlanId.split(',');
		                    arrVlan1.map(function(m){
	                            if(arr4.includes(m)) {
	                            	isRepeated4 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					if(vm.form.apn1Enable == '1' && vm.form.apn1BearType == '0'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingShuRu")%> VLAN ID');
						}else if(!bool || repeated1 || isRepeated2 || isRepeated3 || isRepeated4 || arrVlan1.length > 3 || (value<0 || value>4094 || value == 1)){
							callback('Separated by commas,max 3,range: 0,2~4094 except 1,no repeat');
						}else {								
							callback();	
						}
					}else{
						callback();
					}											
				},
				validateApn2VlanId = function(rule,value,callback){
					var arrVlan2 = (value || '').split(','), bool = true, repeated2 = false, isRepeated1 = false, isRepeated3 = false, isRepeated4 = false;					
					arrVlan2.map(function(item){if(reg.test(item.trim()) && item >= 0 && item <=4094 && item != 1) {bool = true; }else{ bool = false;}});
					
					if(arrVlan2.length > 1){
						for(var i = 0; i<arrVlan2.length;i++){
							if(arrVlan2.indexOf(arrVlan2[i]) != i || ( arrVlan2[i] == 1 && arrVlan2[i] < 0 && arrVlan2[i] > 4094)){
								repeated2 = true
							}
						}
					}
					if(vm.form.apn1Enable == '1'  && vm.form.apn1BearType == '0'){
						if(vm.form.apn1VlanId === '' || vm.form.apn1VlanId === null || vm.form.apn1VlanId === undefined){
						}else{
							var arr1 = vm.form.apn3VlanId.split(',');
		                    arrVlan2.map(function(m){
	                            if(arr1.includes(m)) {
	                            	isRepeated1 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					
					if(vm.form.apn3Enable == '1' && vm.form.apn3BearType == '0'){
						if(vm.form.apn3VlanId === '' || vm.form.apn3VlanId === null || vm.form.apn3VlanId === undefined){
						}else{
							var arr3 = vm.form.apn3VlanId.split(',');
		                    arrVlan2.map(function(m){
	                            if(arr3.includes(m)) {
	                            	isRepeated3 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					
					if(vm.form.apn4Enable == '1' && vm.form.apn4BearType == '0'){
						if(vm.form.apn4VlanId === '' || vm.form.apn4VlanId === null || vm.form.apn4VlanId === undefined){
						}else{
							var arr4 = vm.form.apn4VlanId.split(',');
		                    arrVlan2.map(function(m){
	                            if(arr4.includes(m)) {
	                            	isRepeated4 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					
					if(vm.form.apn2Enable == '1' && vm.form.apn2BearType == '0'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingShuRu")%> VLAN ID');
						}else if(!bool || repeated2 || isRepeated1 || isRepeated3 || isRepeated4 || arrVlan2.length > 3 || (value<0 || value>4094 || value == 1)){
							callback('Separated by commas,max 3,range: 0,2~4094 except 1,no repeat');
						}else {								
							callback();	
						}
					}else{
						callback();
					}					
				},
				validateApn3VlanId = function(rule,value,callback){
					var arrVlan3 = (value || '').split(','), bool = true, repeated3 = false, isRepeated1 = false, isRepeated2 = false, isRepeated4 = false;					
					arrVlan3.map(function(item){if(reg.test(item.trim()) && item >= 0 && item <=4094 && item != 1) {bool = true; }else{ bool = false;}});
					if(arrVlan3.length > 1){
						for(var i = 0; i<arrVlan3.length;i++){
							if(arrVlan3.indexOf(arrVlan3[i]) != i  || ( arrVlan3[i] == 1 && arrVlan3[i] < 0 && arrVlan3[i] > 4094)){
								repeated3 = true
							}
						}
					}
					if(vm.form.apn1Enable == '1'  && vm.form.apn1BearType == '0'){
						if(vm.form.apn1VlanId === '' || vm.form.apn1VlanId === null || vm.form.apn1VlanId === undefined){
						}else{
							var arr1 = vm.form.apn1VlanId.split(',');
		                    arrVlan3.map(function(m){
	                            if(arr1.includes(m)) {
	                            	isRepeated1 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					
					if(vm.form.apn2Enable == '1' && vm.form.apn2BearType == '0'){
						if(vm.form.apn2VlanId === '' || vm.form.apn2VlanId === null || vm.form.apn2VlanId === undefined){
						}else{
							var arr2 = vm.form.apn2VlanId.split(',');
		                    arrVlan3.map(function(m){
	                            if(arr2.includes(m)) {
	                            	isRepeated2 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					
					if(vm.form.apn4Enable == '1' && vm.form.apn4BearType == '0'){
						if(vm.form.apn4VlanId === '' || vm.form.apn4VlanId === null || vm.form.apn4VlanId === undefined){
						}else{
							var arr4 = vm.form.apn4VlanId.split(',');
		                    arrVlan3.map(function(m){
	                            if(arr4.includes(m)) {
	                            	isRepeated4 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					if(vm.form.apn3Enable == '1' && vm.form.apn3BearType == '0'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingShuRu")%> VLAN ID');
						}else if(!bool || repeated3 || isRepeated1 || isRepeated2 || isRepeated4 || arrVlan3.length > 3 || (value<0 || value>4094 || value == 1)){
							callback('Separated by commas,max 3,range: 0,2~4094 except 1,no repeat');
						}else {								
							callback();	
						}
					}else{
						callback();
					}					
				},
				validateApn4VlanId = function(rule,value,callback){
					var arrVlan4 = (value || '').split(','), bool = true, repeated4 = false, isRepeated1 = false, isRepeated2 = false, isRepeated3 = false;					
					arrVlan4.map(function(item){if(reg.test(item.trim()) && item >= 0 && item <=4094 && item != 1) {bool = true; }else{ bool = false;}});
					if(arrVlan4.length > 1){
						for(var i = 0; i<arrVlan4.length;i++){
							if(arrVlan4.indexOf(arrVlan4[i]) != i || ( arrVlan4[i] == 1 && arrVlan4[i] < 0 && arrVlan4[i] > 4094)){
								repeated4 = true
							}
						}
					}
					if(vm.form.apn1Enable == '1' && vm.form.apn1BearType == '0'){
						if(vm.form.apn1VlanId === '' || vm.form.apn1VlanId === null || vm.form.apn1VlanId === undefined){
						}else{
							var arr1 = vm.form.apn1VlanId.split(',');
		                    arrVlan4.map(function(m){
	                            if(arr1.includes(m)) {
	                            	isRepeated1 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					if(vm.form.apn2Enable == '1' && vm.form.apn2BearType == '0'){
						if(vm.form.apn2VlanId === '' || vm.form.apn2VlanId === null || vm.form.apn2VlanId === undefined){
						}else{
							var arr2 = vm.form.apn2VlanId.split(',');
		                    arrVlan4.map(function(m){
	                            if(arr2.includes(m)) {
	                            	isRepeated2 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
					if(vm.form.apn3Enable == '1' && vm.form.apn3BearType == '0'){
						if(vm.form.apn3VlanId === '' || vm.form.apn3VlanId === null || vm.form.apn3VlanId === undefined){
						}else{
							var arr3 = vm.form.apn3VlanId.split(',');
		                    arrVlan4.map(function(m){
	                            if(arr3.includes(m)) {
	                            	isRepeated3 = true;
	                            }
	                        });
						}
					}else{isRepeated = false;}
										
					if(vm.form.apn4Enable == '1' && vm.form.apn4BearType == '0'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingShuRu")%> VLAN ID');
						}else if(!bool || repeated4 || isRepeated1 || isRepeated2 || isRepeated3 || arrVlan4.length > 3 || (value<0 || value>4094 || value == 1)){
							callback('Separated by commas,max 3,range: 0,2~4094 except 1,no repeat');
						}else {								
							callback();	
						}
					}else{
						callback();
					}
				},
				validateIPAddress = function(rule,value,callback){
					var regip = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
					if(vm.form.apn1Enable == '1' || vm.form.apn2Enable == '1' || vm.form.apn3Enable == '1' || vm.form.apn4Enable == '1'){
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("QingShuRu")%> IP');
						}else if(regip.test(value)){
							callback()
						}else {
							callback(new Error('IP <%=rb.getString("GeShiCuoWu")%>'))
						}
					}else{
						callback();
					}						
				},
				validatePolicy = function(rule,value,cb){
					if(value === '' || value === null) {
						cb('<%=rb.getString("QingXuanZe")%>');
					}else {
						cb();
					}
				};
				
			return {	        		       
				taskId:'',
				queryParams: {
					searchText: '',
					timeZone: timeZone
				},
				height:'370px',				
				// 表单数据 
				form: { 
					//必传参数
					timeZone: timeZone,
					taskName: '${addTaskName}',
					cpeCodes: '',
					executeType: 'active',	
					time: '',
					taskId: '',					
					//network wifi
					wlanEnable: '',
					wifiChannel: '',
					wifiBandwidth: '', 					
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: '',
					wifi1Enable: '',
					wifi1Ssid: '',
					wifi1Encryption: '',
					wifi1Passphrase: '',				
					wifi2Enable: '',
					wifi2Ssid: '',
					wifi2Encryption: '',
					wifi2Passphrase: '',				
					wifi3Enable: '',
					wifi3Ssid: '',
					wifi3Encryption: '',
					wifi3Passphrase: '',
					//network dmz
					dmzEnable: '',
					dmzHostAddress: '',	
					//network lan
					lanEnable: '',
					//lte	pci lock				
					scanMode: '',
					pci: '',
					onlyEarfcn: '',
					earfcnStart: '',
					pciEnd: '',
					onlyPci: '',
					earfcnList: [],
					earfcnPciList: [],
					pciList: [],										
					//lte apn 
					apn1Enable: '0',
					apn2Enable: '0',
					apn3Enable: '0',
					apn4Enable: '0',					
					apn1Name: '',
					apn2Name: '',
					apn3Name: '',
					apn4Name: '',
					apn1BearType: '1',
					apn2BearType: '0',
					apn3BearType: '0',
					apn4BearType: '0',					
					apn1Default: '1',
					apn2Default: '0',
					apn3Default: '0',
					apn4Default: '0',					
					apn1VlanId: '',
					apn2VlanId: '',
					apn3VlanId: '',
					apn4VlanId: '',					
					apn1VpnType: '1',//GRE = 1	
					apn2VpnType: '1',
					apn3VpnType: '1',
					apn4VpnType: '1',				
					apnGreType: '1',
					operMode: '2',					
					greDestIPAddress: '',			
					//system web access
					uiPassword: '',
					httpsEnable: '',
					httpsWanEnable: '',
					wanAccEnable: '',
					rangeIp: '',
					//system ping watchdog
					watchDogEnable: '',
					watchDogPingIp: '',
					watchDogPingTimeout: '',
					watchDogPingCount: '',
					watchDogFailureReboot: '', 										 					
				},
				earfcnErrorMessage: '',
				earfcnPciErrorMessage: '',
				pciErrorMessage: '',
				earfcnTip: false,
				editBtnShow: true,
				AddBtnShow: true,
				setDefault: '1', 
				selectAll:'0',
				// 校验规则
				rules: { 
					<%-- taskName:[
						{required: true,message:'<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'},
						{type:'string',max: 100,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100'}
					],
					cpeCodes:[
						{validator: validateDevice}
					],
 --%>
					/* apn1Name:[
						{validator: validateApn1Name}
					],
					apn2Name:[
						{validator: validateApn2Name}
					], 
					apn3Name:[
						{validator: validateApn3Name}
					],
					apn4Name:[
						{validator: validateApn4Name}
					],
					apn1VlanId:[
						{validator: validateApn1VlanId}
					],
					apn2VlanId:[
						{validator: validateApn2VlanId}
					],
					apn3VlanId:[
						{validator: validateApn3VlanId}
					],
					apn4VlanId:[
						{validator: validateApn4VlanId}
					],
					greDestIPAddress:[
						{validator: validateIPAddress}
					], */
					watchDogPingIp:[{validator: watchdogValidateIP}],
					watchDogPingTimeout:[{validator: watchdogValid}],
					watchDogPingCount:[{validator: watchdogValid}],
					watchDogFailureReboot:[{validator: watchdogValid}]
	                
				},
				showPairGrid: true,
		    	leftUrl : '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3',
		    	rightUrl : '',
				deviceTitle:['','<%=rb.getString("YiXuan")%>'],
				groupOptions:[],
				versionOptions:[],
				readonly: false,
				isPswModel: true,
				operateType:'',
				selection:[],							
				//参数配置
                activeCode: 'network',
                selectOptions:[   				
    				{
    					text:'',
    					value:''
    				},
    				{
    					text:'Disable',
    					value:'0'
    				},
    				{
    					text:'Enable',
    					value:'1'
    				}
    			],
                apnOptions:[],
                tbData: [],
                profileListData: [],
                //ip table
                ipTableData: [],
                isLANEnable: true,
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
				//import
				showImportCard:false,
				importRuleForm: {
		            uploadFileUrl: '',
		       	},	         	          
		        fileParams:{},              
		        fileName:'',	            					
				showFileTip:false,
				fileList:[],
				filePath:'',
				importRules: {	           		
					fileName:[
		            	{ validator: validateFileName},
		            ]                   
		        },
				wifiMDLShow: false,
				mForm: {
					id: '',
					wifiEnable: '',
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: '',
					showPassword: false
				},
				mFormRules: {
					wifiEnable: [{validator: validWiFiEnable}],
					wifiSsid: [{validator: validSsid}],
					wifiEncryption: [{validator: validEncryption}],
					wifiPassphrase: [{validator: validPassphrase}]
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
				curCpeCode:'',
				
				
				activeName:'',
				curParamsKey:'',
				//apn
				BearTypeOptions:[
					{
    					text:'VOIP',
    					value:'2'
    				},
					{
    					text:'MGMT',
    					value:'1'
    				},
    				{
    					text:'DATA',
    					value:'0'
    				},
				],								
				bearTypeOnlyShow: false,
				policySelections:[],
				apnPolicyShow: false,
				selectForm:{
					policy:'',
				},
				selectRules: {
					policy : [{validator: validatePolicy}]
				},
                submitLoading:false,
			}	
		},
		computed: {
			formRules() { // 只读模式置空校验
				return this.readonly? []:this.rules;
			},
			isTimeReadonly(){
				return this.readonly || this.form.executeType != 'timing';
			},
			tabs() {
				return [					
					{code: 'network', text: '<%=rb.getString("WangLuoSheZhi")%>',show: true},
					{code: 'lte', text: 'LTE',show: true},
					{code: 'system', text: '<%=rb.getString("XiTong")%>',show: true},
				]
			},
			isWifiClosed() {
				return this.form.wlanEnable != '1';
			},
			bearTypeDisabled() {
                var vm = this,
                    bool = false;
                if(vm.form.apn1Enable == '1' && vm.form.apn1BearType == '1'){ bool = true;}
                if(vm.form.apn2Enable == '1' && vm.form.apn2BearType == '1'){ bool = true;}
                if(vm.form.apn3Enable == '1' && vm.form.apn3BearType == '1'){ bool = true;}
                if(vm.form.apn4Enable == '1' && vm.form.apn4BearType == '1'){ bool = true;}
                return bool;
            }			
		},
		watch: {
			'form.cpeCodes': {
				handler: function(val){
					this.$refs.addform.validateField('cpeCodes');
				},
				deep: true
			},
			'form.apn1Enable': function(newValue,oldValue){
				var vm = this;				
				if(newValue == '1'){
					vm.$refs.addform.validateField('apn1Name');
					if(vm.form.apn1BearType == '0'){
						vm.$refs.addform.validateField('apn1VlanId');
					}
					
				}else{
					vm.form.apn1Enable = '0';
					vm.form.apn1BearType = '1';
					vm.form.apn1VlanId = '';
					vm.form.apn1Default = '1';
					vm.form.apn1Name = '';
				}
			},
			'form.apn2Enable': function(newValue,oldValue){
				var vm = this;
				if(newValue == '1'){
					vm.$refs.addform.validateField('apn2Name');
					if(vm.form.apn2BearType == '0'){
						vm.$refs.addform.validateField('apn2VlanId');
					}					
				}else{
					vm.form.apn2Enable = '0';
					vm.form.apn2BearType = '0';
					vm.form.apn2VlanId = '';
					vm.form.apn2Default = '0';
					vm.form.apn2Name = '';
				}
			},
			'form.apn3Enable': function(newValue,oldValue){
				var vm = this;
				if(newValue == '1'){
					vm.$refs.addform.validateField('apn3Name');
					if(vm.form.apn3BearType == '0'){
						vm.$refs.addform.validateField('apn3VlanId');
					}					
				}else{
					vm.form.apn3Enable = '0';
					vm.form.apn3BearType = '0';
					vm.form.apn3VlanId = '';
					vm.form.apn3Default = '0';
					vm.form.apn3Name = '';
				}
			},
			'form.apn4Enable': function(newValue,oldValue){
				var vm = this;
				if(newValue == '1'){
					vm.$refs.addform.validateField('apn4Name');
					if(vm.form.apn4BearType == '0'){
						vm.$refs.addform.validateField('apn4VlanId');
					}					
				}else{
					vm.form.apn4Enable = '0';
					vm.form.apn4BearType = '0';
					vm.form.apn4VlanId = '';
					vm.form.apn4Default = '0';
					vm.form.apn4Name = '';
				}
			}			
		},
		methods: {
			//选择APN Policy
			apnPlicyClick(){
				var vm = this;
				axios.post("${ctx}/cell/CPE/queryApnPolicyNameList.action").then((response) => {
                    var data = response.data;
                    if(data){
                        vm.policySelections = data;
                    }
                });
				this.apnPolicyShow = true;
			},
			savePolicy(){
				var vm = this,params = {};
				params.apnPolicyName = vm.selectForm.policy;
				vm.$refs.selectForm.validate((valid) => {
	                if (valid) {
	                	axios.post("${ctx}/cell/CPE/queryApnPolicyInfo.action",stringify(params)).then((response) => {
	                		var data = response.data;
	                		if(data['apn1ApplyTo'] == '1' || data['apn1Default'] == '1'){
								vm.form.apn1Default = '1';
							}else if(data['apn2ApplyTo'] == '1' || data['apn2Default'] == '1'){
								vm.form.apn2Default = '1'
							}else if(data['apn3ApplyTo'] == '1' || data['apn3Default'] == '1'){
								vm.form.apn3Default = '1'
							}else if(data['apn4ApplyTo'] == '1' || data['apn4Default'] == '1'){
								vm.form.apn4Default = '1'
							}
	                		Object.assign(vm.form, {
								apn1Enable: data['apn1Enable'],
								apn2Enable: data['apn2Enable'],
								apn3Enable: data['apn3Enable'],
								apn4Enable: data['apn4Enable'],
								
								apn1Name: data['apn1Name'],
								apn2Name: data['apn2Name'],
								apn3Name: data['apn3Name'],
								apn4Name: data['apn4Name'],
								
								apn1BearType: data['apn1ApplyTo'],
								apn2BearType: data['apn2ApplyTo'],
								apn3BearType: data['apn3ApplyTo'],
								apn4BearType: data['apn4ApplyTo'],

								apn1Default: vm.form.apn1Default,
								apn2Default: vm.form.apn2Default,
								apn3Default: vm.form.apn3Default,
								apn4Default: vm.form.apn4Default,
								
								apn1VlanId: data['apn1VlanId'],
								apn2VlanId: data['apn2VlanId'],
								apn3VlanId: data['apn3VlanId'],
								apn4VlanId: data['apn4VlanId'],
								
								operMode: data['apnOperMode'],
								apnGreType: data['apnGreType'],
								greDestIPAddress: data['apnGreIpAddress']
							});
	                    });
	                	vm.closePolicy();
	                }
	            })
			},
			closePolicy(){
				this.$refs.selectForm.resetFields();
				this.selectForm.policy = '';
				this.apnPolicyShow = false;
			},
			
			bearTypeEnableChange(val) {
                var vm = this;
                vm.form.apn1Default = vm.form.apn1BearType == '1'?'1':'0';
                vm.form.apn2Default = vm.form.apn2BearType == '1'?'1':'0';
                vm.form.apn3Default = vm.form.apn3BearType == '1'?'1':'0';
                vm.form.apn4Default = vm.form.apn4BearType == '1'?'1':'0';               
            }, 
						
			//参数配置
        	tabClick(tab) {
				var vm = this;
				if(vm.activeCode != tab.code) {
					vm.activeCode = tab.code;				
				}
			},
			
			// 初始化任务信息
			init(cpeCodes) {
				var vm = this; 				
				vm.curCpeCode = cpeCodes;
				//apn list
				axios.post("${ctx}/cell/CPE/queryApnNameList.action").then((response) => {
                    var data = response.data;
                    if(data){
                        vm.apnOptions = data;                   	
                    }					
                });
				
				vm.reloadTable();					
				// 初始化form原始值
				vm.$nextTick(function(){
					initForm(vm.$refs.addform);					
				});											
			},
			
			//详情页面回显form
			initFormInfo(){
				var vm = this;										
				if(vm.operateType == 'information'){	
					var params = {taskId: vm.taskId, timeZone: timeZone};
					axios.post('${ctx}/cpe/batchconfig/getTaskInfo.action', stringify(params)).then(function(response){
						var data = response.data;		
						if(data){
							vm.form.taskName = data.TASK_NAME;
							vm.form.executeType = data.EXECUTE_TYPE;
							if(data.SELECT_ALL == '0'){
								vm.selectAll = '0';
							}else{
								vm.selectAll = '1';
							}	
							if(data.paramConfig  === '' || data.paramConfig  === null || data.paramConfig === undefined){
								vm.reloadTable();
							}else{
								var curParams = JSON.parse(data.paramConfig);
								for(var key in curParams){
									if(key == 'wifi'){
										if(curParams[key]['wlanEnable']){
											vm.raloadTableData( curParams[key]);
											vm.reloadInfoForm(curParams);		
										}										
										return
									}else{
										vm.tbData = [];
										vm.reloadTable();
										vm.reloadInfoForm(curParams);		
									}
								} 
							}
						}
					}).catch(function(error){})	
				}
			},
			
			// 打开导入弹出框
			importFileBtn(){
				this.showImportCard = true;
			},
			//导入
			checkFile(res,file){  
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
					vm.showImportCard = false;
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
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name;
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
				vm.$refs.upload.clearFiles();
			},
						
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			beforeUpload(file){
				var vm = this, importUrl, fileName = file.name,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};

				fd.append('uploadFile',file); //文件流
				axios.post('${ctx}/cpe/batchconfig/importBatchConfigurationInfos.action',fd,config).then(function(res){
					var data = res.data;
					if(data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');						
						var curParams = data.paramConfig;
						vm.tbData = [];
						vm.$refs.addform.resetFields(); //表单置空重新渲染						
						vm.closeImportParams();						
						for(var key in curParams){
							vm.ipTableData = [];							
							if(key == 'wifi'){
								if(curParams[key] != null){									
									if(curParams[key]['wlanEnable']){
										vm.tbData = [];
										vm.raloadTableData( curParams[key]);
										vm.reloadInfoForm(curParams);										
									}
								}
								return
							}else{
								vm.tbData = [];
								vm.reloadTable();
								vm.reloadInfoForm(curParams);
							}
						} 						
					}else{
						vm.$message.error(data["msg"])
	        			vm.fileList = [];
						vm.fileName = '';
						vm.$refs.importRuleForm.resetFields();
					} 
				})			
				return false;
			},
			
			reloadInfoForm(curParams){
				var vm = this;
				if(curParams != null || curParams != undefined){
					for(var key in curParams){
						if(typeof curParams[key] === 'object'){							
							vm.reloadInfoForm(curParams[key])							
						}else{
							vm.form[key] = curParams[key];
							if(key == 'pci'){															
								var newEarfcnPci = curParams[key].split(';');
								if(vm.form.scanMode == 'freqpreferred'){	
									vm.form.earfcnList = newEarfcnPci.map(function(item){												
										return item
									});
								}else if(vm.form.scanMode == 'pcilock'){
									vm.form.earfcnPciList = newEarfcnPci.map(function(item){
										var getRowData = {};
										if(item.includes(',')){
											getRowData = {     						
												startValue : item.split(',')[0],
												endValue : item.split(',')[1]	    			
					      		    		}
										}
										return getRowData
									});
								}else if (vm.form.scanMode == 'pcionlylock'){
									vm.form.pciList = newEarfcnPci.map(function(item){
										return item
									});
								}														
							}
							if(key == 'rangeIp'){								
								var newIpListData = curParams[key].split(';');	
								vm.ipTableData = newIpListData.map((item,index) => {
									if(item.includes('-')){
										return {     						
											ipStart : item.split('-')[0],
				      						ipEnd : item.split('-')[1]	    			
				      		    		}
									}else{												
										return {     						
											ipStart : item	    			
					      		    	}
									}																			
								});	
							}	
							if(key == 'apn'){
								if(vm.form.apn1BearType === '' || vm.form.apn1BearType === null || vm.form.apn1BearType === undefined){
									vm.form.apn1BearType = '1'
								} 
								if(vm.form.apn2BearType === '' || vm.form.apn2BearType === null || vm.form.apn2BearType === undefined){
									vm.form.apn2BearType = '0'
								}
								if(vm.form.apn3BearType === '' || vm.form.apn3BearType === null || vm.form.apn3BearType === undefined){
									vm.form.apn3BearType = '0'
								}
								if(vm.form.apn4BearType === '' || vm.form.apn4BearType === null || vm.form.apn4BearType === undefined){
									vm.form.apn4BearType = '0'
								}
							}
						}
					}
				} 
			},
		
			reloadTable(){
				var vm = this;
				vm.tbData.push({
					id: 'main',
					wifiEnable: '',
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: ''
				}); 
				// networkForm更新
				Object.assign(vm.form,{
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: ''
				});
				// sub 
				 ['1','2','3'].map(function(item){
					var row = {},pre = 'wifi',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';							
					
					row['id'] = item;
					row['wifiEnable'] = '';
					row['wifiSsid'] = '';
					row['wifiEncryption'] = '';
					row['wifiPassphrase'] = '';

					// networkForm更新
					vm.form[enableKey] = '';
					vm.form[idKey] = '';
					vm.form[encryKey] = '';
					vm.form[phraKey] = '';
					vm.tbData.push(row); 
				}); 
			},
			raloadTableData( data){
				var vm = this;
				vm.tbData.push({
					id: 'main',
					wifiEnable: data["wlanEnable"],
					wifiSsid: data["wifiSsid"],
					wifiEncryption: data["wifiEncryption"],
					wifiPassphrase: data["wifiPassphrase"],
				});
				Object.assign(vm.form,{
					wifiSsid: data["wifiSsid"],
					wifiEncryption: data["wifiEncryption"],
					wifiPassphrase: data["wifiPassphrase"],
				}); 
				// sub 
				['1','2','3'].map(function(item){
					var row = {},pre = 'wifi',suf = 'Old',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',						
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';
					row['id'] = item;
					row['wifiEnable'] = data[enableKey];
					row['wifiSsid'] = data[idKey];
					row['wifiEncryption'] = data[encryKey];
					row['wifiPassphrase'] = data[phraKey];
					vm.form[enableKey] = data[enableKey];
					vm.form[idKey] = data[idKey];
					vm.form[encryKey] = data[encryKey];
					vm.form[phraKey] = data[phraKey];
					vm.tbData.push(row);
				}); 
			},
						
			/*确定导入*/
	        uploadParams() {
				var vm = this;
				vm.$refs.importRuleForm.validate((valid) => {
	                if (valid) {
	                	vm.$refs.upload.submit();                   	
	                }
	            }) 				
			},
			
			// 关闭导入弹出框
			closeImportParams(){
				var vm = this;			
				vm.fileList = [];
				vm.fileName = '';
				vm.$refs.importRuleForm.resetFields();
				vm.showImportCard = false;
			},
			//下载模板
			exportTemplate(){
				exportByForm('${ctx}/cpe/batchconfig/downloadTemplate.action', {});
			},
			
			//导出
			exportFileBtn(){
				var vm = this, newForm = vm.$refs.addform;
				var newParams = vm.formatParams(vm.form),
					pureParams = vm.pureParams(newForm,newParams);

				 exportByForm("${ctx}/cpe/batchconfig/exportBatchConfigurationInfos.action",pureParams);
			},
			// 导入 导出 end
						
			//add  earfcn
			addEarfcnBtn(){
				var vm = this , value = vm.form.onlyEarfcn;
				if(value && isNaN(value)) {
					vm.earfcnErrorMessage = '<%=rb.getString("PinDianGeShiCuoWu")%>';
				}else if(value) {
					if(value - 0 < 0 || value - 65535 > 0) {
						vm.earfcnErrorMessage = '<%=rb.getString("PinDianChaoChuFanWei")%>';
					}else {
						if(vm.form.earfcnList.indexOf(value) == -1){
							vm.form.earfcnList.push(value);
							vm.form.onlyEarfcn = '';
							vm.earfcnErrorMessage = '';
							vm.form.pci = vm.form.earfcnList.map(function(item){
								return item;							
							}).join(';');
						}else{
							vm.earfcnErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}
				}else {
					vm.earfcnErrorMessage = '<%=rb.getString("PinDianWeiKong")%>';
				} 
			},
			//新建 earfcn and pci
			addPcilockBtn(){
				var vm = this , startValue = vm.form.earfcnStart, endValue = vm.form.pciEnd, regNum = /^\d+$/;
				if(startValue == '' || endValue == ''){
					vm.earfcnPciErrorMessage = '<%=rb.getString("QingShuRu")%><%=rb.getString("PinDian")%> , <%=rb.getString("PCI2")%>';
				}else{
					
					if(!regNum.test(startValue) || ( startValue - 0 < 0 || startValue - 65535 > 0)) {
						vm.earfcnPciErrorMessage = 'Earfcn <%=rb.getString("FanWei")%>：0 ~65535';
					}else if(!regNum.test(endValue) || (endValue - 0 < 0 || endValue-503 > 0)){
						vm.earfcnPciErrorMessage = 'PCI <%=rb.getString("FanWei")%>：0~503';
					}else{
						str = startValue + "," + endValue;
						if(vm.form.earfcnPciList.indexOf(str) == -1){
							vm.form.earfcnPciList.push({startValue,endValue});
							vm.form.earfcnStart = '';
							vm.form.pciEnd = '';
							vm.earfcnPciErrorMessage = '';
							vm.form.pci = vm.form.earfcnPciList.map(function(item){
								var curPci = item.startValue +','+ item.endValue;						
								return curPci;
							}).join(';');
						}else{
							vm.earfcnPciErrorMessage = '<%=rb.getString("IPFanWeiYiCunZai")%>';
						} 
					}					
				}				
			},
			// add pci
			addPciBtn(){
				var vm = this , value = vm.form.onlyPci;
				if(value && isNaN(value)) {
					vm.pciErrorMessage = '<%=rb.getString("PCIGeShiCuoWu")%>';
				}else if(value) {
					if(value-0 < 0 || value-503 > 0) {
						vm.pciErrorMessage = '<%=rb.getString("PCIChaoChuFanWei")%>';
					}else {
						if(vm.form.pciList.indexOf(value) == -1){
							vm.form.pciList.push(value);
							vm.form.onlyPci = '';
							vm.pciErrorMessage = '';
							vm.form.pci = vm.form.pciList.map(function(item){
								return item;
							}).join(';');
						}else{
							vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
						}
					}
				}else {
					vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
				} 
			},
			removeEarfcn(item){
				var vm = this, index = vm.form.earfcnList.indexOf(item);
			
				if(index !== -1){
					vm.form.earfcnList.splice(index,1)
				}
				vm.form.pci = vm.form.earfcnList.map(function(item){
					return item;							
				}).join(';');
				vm.earfcnErrorMessage = '';
			},
			removeEarfcnPci(item){
				var vm = this, index = vm.form.earfcnPciList.indexOf(item);
				if(index !== -1){
					vm.form.earfcnPciList.splice(index,1)
				}
				vm.form.pci = vm.form.earfcnPciList.map(function(item){
					var curPci = item.startValue +','+ item.endValue;						
					return curPci;
				}).join(';');
				vm.earfcnPciErrorMessage = '';
			},
			removePci(item){
				var vm = this, index = vm.form.pciList.indexOf(item);
				if(index !== -1){
					vm.form.pciList.splice(index,1)
				}
				vm.form.pci = form.pciList.map(function(item){
					return item;
				}).join(';');
				vm.pciErrorMessage = '';
			},
			
			modifyWifi(row) {
				var vm = this;
				vm.wifiMDLShow = true;
				vm.$nextTick(function(){
					vm.$refs.modifyForm.resetFields();
					Object.assign(vm.mForm, row);
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
			
        	handleSizeChange(val){
				this.pageSize = val;
			},
			handleCurrentChange(val){
				this.currentPage = val;
			},
			//NEW IP
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
								vm.ipTableData.splice(vm.editIndex,1,vm.editIpForm)
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
								vm.ipTableData.splice(vm.editIndex,1,vm.editIpForm)
								vm.closeEditIp();
							}
						}else{
							vm.editErrorMsg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						 }
					 }											
					 vm.form.rangeIp = vm.ipTableData.map((item,index) => {
						if(item.ipEnd ){
							return item.ipStart + '-' + item.ipEnd;
						}else{
							return item.ipStart;
						}								
					}).join(';');
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
					vm.ipTableData.splice(index,1);			    	
			    }
				vm.form.rangeIp =  vm.ipTableData.map((item,index) => {
					if(item.ipEnd ){
						return item.ipStart + '-' + item.ipEnd;
					}else{
						return item.ipStart;
					}								
				}).join(';');
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
						vm.ipTableData.push(getRowData);
					}) 
					vm.oldNewIpTableData();
					vm.closeAddIp();
					vm.form.rangeIp = vm.ipTableData.map((item,index) => {
						if(item.ipEnd ){
							return item.ipStart + '-' + item.ipEnd;
						}else{
							return item.ipStart;
						}								
					}).join(';');
				}				
			},
			
			oldNewIpTableData(){
				var vm = this;
				for(var i = 0; i< vm.ipTableData.length; i++){
					for(var j = i+1; j < vm.ipTableData.length; j++ ){
						//两种情况，1：只有 start ip; 2 start ip, end ip 都有时
						if(vm.ipTableData[i].ipEnd){
							if(vm.ipTableData[i].ipStart == vm.ipTableData[j].ipStart && vm.ipTableData[i].ipEnd == vm.ipTableData[j].ipEnd){
								vm.ipTableData.splice(j,1);								
								j--;
							}
						}else{
							if(vm.ipTableData[i].ipStart == vm.ipTableData[j].ipStart){
								vm.ipTableData.splice(j,1);							
								j--;
							}
						}					
					}					
				}
				return vm.ipTableData;
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
			
			//wifi 开关改变
			wlanEnableChange(val) {
				var vm = this,
				row = vm.tbData[0];

				if(row) {
					row.wifiEnable = val == '1';
				}
			},						
			
			loadSuccessDevice(){
				initForm(this.$refs.addform)
			},
			// 右侧的筛选搜索
			queryDeviceList(val){
				var vm = this;
				vm.queryParams.searchText = val;
			},

			/**
			 * 设备选择变化时，更新选择设备记录
			 * @param value:
			*/
			devicesChange(value) {
				var vm = this;
				vm.$nextTick(function(){
					var rows = vm.$refs.deviceListPairgrid.getData();
					vm.form.cpeCodes = rows.map(function(row){ return row.small_cell_code ;}).sort().join(',');
				})
			},
			
			// 关闭
			closePanel(){
				var vm = this;
				vm.cancel();			
			}, 

			// 提交
			formSubmit(){ 
				var vm = this,		
					params = {},
					newForm = vm.$refs.addform;
					message = '<%=rb.getString("ChengGong")%>';
                // 防止多次提交
                if(vm.submitLoading)return
				//params 必传参数
				params.timeZone = timeZone;
				params.taskName = vm.form.taskName;
				params.time = '';
				params.executeType = 'active';
				params.taskId = '';				
				params.selectAll = '0';
				params.cpeCodes = vm.curCpeCode;

				var flagEarfcn = false;	
				//earfcn, earfcn and pci, pci
				if(vm.form.scanMode == 'freqpreferred'){
					if(vm.form.earfcnList.length == 0){
						flagEarfcn = true;
						vm.earfcnErrorMessage = '<%=rb.getString("PinDianWeiKong")%>';
					}else{
						flagEarfcn = false;
						vm.earfcnErrorMessage = '';						
					} 
				}else if(vm.form.scanMode == 'pcilock'){
					if(vm.form.earfcnPciList.length == 0){
						flagEarfcn = true;
						vm.earfcnPciErrorMessage = '<%=rb.getString("PinDianWeiKong")%> ';
					}else{
						flagEarfcn = false;
						vm.earfcnPciErrorMessage = '';						
					}
				}else if(vm.form.scanMode == 'pcionlylock'){
					if(vm.form.pciList.length == 0){
						flagEarfcn = true;
						vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
					}else{
						flagEarfcn = false;
						vm.pciErrorMessage = '';						
					}
				}		
				if(vm.form.apn1Enable == '1' && vm.form.apn2Enable == '1' && vm.form.apn3Enable == '1' && vm.form.apn4Enable == '1'){					
					if(vm.form.apn1BearType != '1' && vm.form.apn2BearType != '1' && vm.form.apn3BearType != '1' && vm.form.apn4BearType != '1'){
						vm.bearTypeOnlyShow = true; //无 bear type:mgmt 
					}else{
						//有其它 bear type:MGMT
						if((vm.form.apn1BearType == '1' && (vm.form.apn2BearType == '1' || vm.form.apn3BearType == '1' || vm.form.apn4BearType == '1' ))
							||(vm.form.apn2BearType == '1' && (vm.form.apn1BearType == '1' || vm.form.apn3BearType == '1' || vm.form.apn4BearType == '1' ))
							||(vm.form.apn3BearType == '1' && (vm.form.apn1BearType == '1' || vm.form.apn2BearType == '1' || vm.form.apn4BearType == '1' ))
							||(vm.form.apn4BearType == '1' && (vm.form.apn1BearType == '1' || vm.form.apn2BearType == '1' || vm.form.apn3BearType == '1' ))){
							vm.bearTypeOnlyShow = true;
						}else{
							vm.bearTypeOnlyShow = false;
						}
					}
				}else{
					if(vm.form.apn1Enable == '0' && vm.form.apn2Enable == '0' && vm.form.apn3Enable == '0' && vm.form.apn4Enable == '0'){
						vm.bearTypeOnlyShow = false; //可以不配置
					}else{
						
					//勾选三组
					if(vm.form.apn1Enable == '1' && vm.form.apn2Enable == '1' && vm.form.apn3Enable == '1'){
						if((vm.form.apn1BearType == '1' && vm.form.apn2BearType != '1' && vm.form.apn3BearType != '1' ) 																				
							|| (vm.form.apn2BearType == '1' && vm.form.apn1BearType != '1' && vm.form.apn3BearType != '1')											
							|| (vm.form.apn3BearType == '1' && vm.form.apn2BearType != '1' && vm.form.apn1BearType != '1')){									
							vm.bearTypeOnlyShow = false;
						}else{vm.bearTypeOnlyShow = true;}
					}else if(vm.form.apn1Enable == '1' && vm.form.apn2Enable == '1' && vm.form.apn4Enable == '1'){
						if((vm.form.apn1BearType == '1' && vm.form.apn2BearType != '1' && vm.form.apn4BearType != '1' ) 																							
								|| (vm.form.apn2BearType == '1' && vm.form.apn1BearType != '1' && vm.form.apn4BearType != '1')										
								|| (vm.form.apn4BearType == '1' && vm.form.apn2BearType != '1' && vm.form.apn1BearType != '1')){											
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if(vm.form.apn1Enable == '1' && vm.form.apn3Enable == '1' && vm.form.apn4Enable == '1'){
							if((vm.form.apn1BearType == '1' && vm.form.apn3BearType != '1' && vm.form.apn4BearType != '1' ) 																						
								|| (vm.form.apn3BearType == '1' && vm.form.apn1BearType != '1' && vm.form.apn4BearType != '1')											
								|| (vm.form.apn4BearType == '1' && vm.form.apn1BearType != '1' && vm.form.apn3BearType != '1')){											
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if(vm.form.apn2Enable == '1' && vm.form.apn3Enable == '1' && vm.form.apn4Enable == '1'){
							if((vm.form.apn2BearType == '1' && vm.form.apn3BearType != '1' && vm.form.apn4BearType != '1' ) 
								|| (vm.form.apn3BearType == '1' && vm.form.apn2BearType != '1' && vm.form.apn4BearType != '1')
								|| (vm.form.apn4BearType == '1' && vm.form.apn2BearType != '1' && vm.form.apn3BearType != '1')){										
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
							//勾选两组
						}else if(vm.form.apn2Enable == '1' && vm.form.apn3Enable == '1'){									
							if((vm.form.apn2BearType == '1' && vm.form.apn3BearType != '1') || (vm.form.apn2BearType != '1' && vm.form.apn3BearType == '1')){											
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if(vm.form.apn2Enable == '1' && vm.form.apn4Enable == '1'){
							if((vm.form.apn2BearType == '1' && vm.form.apn4BearType != '1') ||(vm.form.apn2BearType != '1' && vm.form.apn4BearType == '1')){
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if(vm.form.apn3Enable == '1' && vm.form.apn4Enable == '1'){
							if((vm.form.apn3BearType == '1' && vm.form.apn4BearType != '1') || (vm.form.apn3BearType != '1' && vm.form.apn4BearType == '1')){
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if (vm.form.apn1Enable == '1' && vm.form.apn2Enable == '1'){
							if((vm.form.apn1BearType == '1' && vm.form.apn2BearType != '1') || (vm.form.apn1BearType != '1' && vm.form.apn2BearType == '1')){
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if(vm.form.apn1Enable == '1' && vm.form.apn3Enable == '1'){
							if((vm.form.apn1BearType == '1' && vm.form.apn3BearType != '1') || (vm.form.apn1BearType != '1' && vm.form.apn3BearType == '1')){
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if (vm.form.apn1Enable == '1' && vm.form.apn4Enable == '1'){
							if((vm.form.apn1BearType == '1' && vm.form.apn4BearType != '1') || (vm.form.apn1BearType != '1' && vm.form.apn4BearType == '1')){										
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if(vm.form.apn1Enable == '1'){
							if(vm.form.apn1BearType == '1'){										
								vm.bearTypeOnlyShow = false;
							}else{vm.bearTypeOnlyShow = true;}
						}else if(vm.form.apn2Enable == '1'){
							if( vm.form.apn2BearType == '1'){
								vm.bearTypeOnlyShow = false;
							}else{
								vm.bearTypeOnlyShow = true;
							}
						}else if(vm.form.apn3Enable == '1'){
							if( vm.form.apn3BearType == '1'){
								vm.bearTypeOnlyShow = false;
							}else{
								vm.bearTypeOnlyShow = true;
							}
						}else if(vm.form.apn4Enable == '1'){
							if( vm.form.apn4BearType == '1'){
								vm.bearTypeOnlyShow = false;
							}else{
								vm.bearTypeOnlyShow = true;
							}
						}
					}
				}
				vm.$refs.addform.validate(function(valid){
					if(valid && flagEarfcn == false){
						
						var newParams = vm.formatParams(vm.form),
							pureParams = vm.pureParams(newForm,newParams);
						Object.assign(params, pureParams);
						if(isFormChanged(vm.$refs.addform)) {
                            vm.submitLoading = true;
							axios.post('${ctx}/cpe/batchconfig/addTask.action',stringify(params)).then(function(response){
								var data = response.data;
			    				if(data["success"]){
			    					vm.$message({
			    						message: message,
			    						type:'success',
			    					})
                                    eventBus.$emit('hide-slide')
			    				}else{
			    					vm.$message.error(data["message"]);
                                    vm.submitLoading = false;
			    				}
							}).catch(function(error){})
						}else{
							vm.$message({
								message: '<%=rb.getString("CanShuZhiMeiYouBianHua")%>',
								type: 'warning'
							})
						}						
					}
				});
			},
			
			pureParams(newForm,params) {
				var vm = this, map = {}, paramConfig = {};
				newForm.fields.map(function(field){
					//根据prop  找对应的组					
					var groups ={							
							wlanEnable: 'wifi',
							wifiChannel: 'wifi',
							wifiBandwidth: 'wifi',							
							wifiSsid: 'wifi',
							wifiEncryption: 'wifi',
							wifiPassphrase: 'wifi',
							wifi1Enable: 'wifi',
							wifi1Ssid: 'wifi',
							wifi1Encryption: 'wifi',
							wifi1Passphrase: 'wifi',						
							wifi2Enable: 'wifi',
							wifi2Ssid: 'wifi',
							wifi2Encryption: 'wifi',
							wifi2Passphrase: 'wifi',							
							wifi3Enable: 'wifi',
							wifi3Ssid: 'wifi',
							wifi3Encryption: 'wifi',
							wifi3Passphrase: 'wifi',							
							dmzEnable:'dmz',
							dmzHostAddress:'dmz',						
							lanEnable: 'lan',							
							scanMode: 'pciLock',
							pci: 'pciLock',							
							apn1Enable: 'apn',
							apn2Enable: 'apn',
							apn3Enable: 'apn',
							apn4Enable: 'apn',							
							apn1Name: 'apn',
							apn2Name: 'apn',
							apn3Name: 'apn',
							apn4Name: 'apn',
							apn1BearType: 'apn',
							apn2BearType: 'apn',
							apn3BearType: 'apn',
							apn4BearType: 'apn',							
							apn1Default: 'apn',
							apn2Default: 'apn',
							apn3Default: 'apn',
							apn4Default: 'apn',							
							apn1VlanId: 'apn',
							apn2VlanId: 'apn',
							apn3VlanId: 'apn',
							apn4VlanId: 'apn',						
							apn1VpnType: 'apn',
							apn2VpnType: 'apn',
							apn3VpnType: 'apn',
							apn4VpnType: 'apn',							
							apnGreType: 'apn',
							operMode: 'apn',		
							greDestIPAddress: 'apn',							
							uiPassword: 'wanAcc',
							httpsEnable: 'wanAcc',
							httpsWanEnable: 'wanAcc',
							wanAccEnable: 'wanAcc',
							rangeIp: 'wanAcc',							
							watchDogEnable: 'watchDog',
							watchDogPingIp: 'watchDog',
							watchDogPingTimeout: 'watchDog',
							watchDogPingCount: 'watchDog',
							watchDogFailureReboot: 'watchDog'						
						},
						prop = field.prop,
						groupName = groups[prop];

					if(groupName === undefined){
						if(field.fieldValue !== field.reinitialValue) map[prop] = field.fieldValue;						
					}
		
					if(field.fieldValue !== field.reinitialValue && groupName !== undefined) {
						if(paramConfig[groupName] === undefined ){
							paramConfig[groupName] = {};						
						}
						paramConfig[groupName][prop] = field.fieldValue;
						if(paramConfig.apn !== undefined){
							if(paramConfig.apn.apn1Enable == '1'){
								//一些固定值也要传入
								Object.assign(paramConfig.apn, {																	
									apn1Default: vm.form.apn1Default,
									apn1BearType: vm.form.apn1BearType,	
									apn1VlanId: vm.form.apn1VlanId,
									operMode: '2',
									greDestIPAddress: vm.form.greDestIPAddress,
									apn1VpnType: '1',
									apnGreType: '1'								
								});
							}
							if(paramConfig.apn.apn2Enable == '1'){
								Object.assign(paramConfig.apn, {
									apn2Default: vm.form.apn2Default,
									apn2BearType: vm.form.apn2BearType,	
									apn2VlanId: vm.form.apn2VlanId,
									operMode: '2',
									greDestIPAddress: vm.form.greDestIPAddress,
									apn2VpnType: '1',
									apnGreType: '1'	
								});
							}
							if(paramConfig.apn.apn3Enable == '1'){
								Object.assign(paramConfig.apn, {
									apn3Default: vm.form.apn3Default,
									apn3BearType: vm.form.apn3BearType,	
									apn3VlanId: vm.form.apn3VlanId,
									operMode: '2',
									greDestIPAddress: vm.form.greDestIPAddress,
									apn3VpnType: '1',
									apnGreType: '1'	
								});
							}
							if(paramConfig.apn.apn4Enable == '1'){
								Object.assign(paramConfig.apn, {
									apn4Default: vm.form.apn4Default,
									apn4BearType: vm.form.apn4BearType,	
									apn4VlanId: vm.form.apn4VlanId,
									operMode: '2',
									greDestIPAddress: vm.form.greDestIPAddress,
									apn4VpnType: '1',
									apnGreType: '1'	
								});
							}
						} 
					}; 
				});
				
				map.paramConfig = JSON.stringify(paramConfig);
				return map;
			},
			
			formatParams(form) {
				var vm = this,
					code = vm.activeCode,
					params = {};
				if(code == 'system'){				
					params.rangeIp = vm.ipTableData.map((item,index) => {
						if(item.ipEnd ){
							return item.ipStart + '-' + item.ipEnd;
						}else{
							return item.ipStart;
						}								
					}).join(';');
				}else if(code == 'lte') {
					var scanMode = form.scanMode;

					params.scanMode = scanMode;
					//earfcn
					if(scanMode == 'freqpreferred') {
						params.pci = form.earfcnList.map(function(item){
							return item;							
						}).join(';');						
					}
					// earfcn and pci
					if(scanMode == 'pcilock') {
						params.pci = form.earfcnPciList.map(function(item){
							var curPci = item.startValue +','+ item.endValue;						
							return curPci;
						}).join(';');
					}
					//pci
					if(scanMode == 'pcionlylock') {
						params.pci = form.pciList.map(function(item){
							return item;
						}).join(';');
					}
					
				}else if(code == 'network') {
					vm.tbData.map(function(row){
						if(['1','2','3'].includes(row.id)) {
							var index = row.id,
								pre = 'wifi',
								enableKey = pre+index+'Enable',
								idKey = pre+index+'Ssid',
								encryKey = pre+index+'Encryption',
								phraKey = pre+index+'Passphrase';							
							form[enableKey] = row['wifiEnable'];
							form[idKey] = row['wifiSsid'];
							form[encryKey] = row['wifiEncryption'];
							form[phraKey] = row['wifiPassphrase'];
						}else {
							Object.assign(form,{
								wifiSsid: row['wifiSsid'],
								wifiEncryption: row['wifiEncryption'],
								wifiPassphrase: row['wifiPassphrase']
							});
						}
					});
				
					if(form.wlanEnable == '1') {
						Object.assign(params, form);
					}else {
						Object.assign(params, {
							wlanEnable: form.wlanEnable
						});
					}					
				}else {
					Object.assign(params, form);
				}
				return params;
			},
			
			// 关闭新建弹窗
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
				if(isFormChanged(vm.$refs.addform)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-slide')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-slide')
				}
			},
					
		},
		mounted(){
			eventBus.$off('modify-task').$on('modify-task',this.init);
		}
	});
</script>