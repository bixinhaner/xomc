<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<div id='quickSettingPanel' style='height: 100%;overflow: auto;background: #fff;'>
	<el-form ref="enbSettingForm" :model="enbSettingForm" :rules="enbSettingRules" 
	label-position="top" style='width:100%;height:100%;' inline
	>
		<el-collapse v-model="activeNames">
			<!-- Quick Setting -->
			<el-collapse-item v-show="(hasKey('halobEnable') || hasKey('sasEnable')) && !isBaiblx_BLQ" name="quickSetting">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Quick Settings</span>
					</p>
				</template>
				<div style="">
					<p v-show="hasKey('sasEnable')" class='item-title-cls'>CBRS</p>
					<el-form-item v-show="hasKey('sasEnable')" label='<%=rb.getString("SASKaiGuan")%>' prop="sasEnable" style="width: 96%;">
						<el-switch v-model="enbSettingForm.sasEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
						<span style='color:#999;font-size:12px;margin-left:20px;'>
							<%=rb.getString("CBSDQianZhiPeiZhiTiShi")%>
							<span @click="goProcedure" style='color:#4D84FF;text-decoration:underline;cursor:pointer'><%=rb.getString("CBSDSheZhi")%> >></span>
						</span>
					</el-form-item>
					<p v-show="hasKey('halobEnable')" class='item-title-cls'>Quick Mode</p>
					<div v-show="hasKey('halobEnable')" class=''>
						<div style="display:flex">
							<div class='list-item-cls'>
								<el-form-item label='' label-width='0' prop="halobEnable" style="width: 100%">
									<el-radio-group v-model='enbSettingForm.halobEnable'>
										<el-radio v-if="!isBaiblx_QRTB" label="1" style='width:160px;'>HaloB</el-radio>
										<el-radio label="0" style='width:160px;margin-left:0px;'>Normal</el-radio>
									</el-radio-group>
								</el-form-item>
								
							</div>
							<div class='list-item-cls' v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)">
								<el-form-item label='' label-width='0' v-show='epcSwitchFlag' prop="epcSwitch" style="width: 100%">
									<el-radio-group v-model='enbSettingForm.epcSwitch'>
										<el-radio v-if="!isBaiblx_QRTB" label="1" style='width:160px;'>CloudEPC</el-radio>
										<el-radio label="0" style='width:160px;margin-left:0px;'>LocalEPC</el-radio>
									</el-radio-group>
								</el-form-item>
								
							</div>
						</div>
						<div  style="display:flex">
							<el-form-item class="list-item-cls" label='<%=rb.getString("JiZhanZhiShi")%>' prop="duplexMode">
								<el-input v-model='enbSettingForm.duplexMode' disabled></el-input>
							</el-form-item>
							
							<el-form-item v-if="!isBaiblx_QRTB" class="list-item-cls" label='<%=rb.getString("ZaiBoLeiXing")%>' prop='carrierMode'>
								<el-select v-model='enbSettingForm.carrierMode' @change="carryModeChange">
									<el-option label="Single Carrier" value='1'></el-option>
									<el-option label='Dual Carrier' value='2'></el-option>
								</el-select>
							</el-form-item>
						</div>
						<el-form-item v-if="!isBaiblx_QRTB" label-width="163" v-show="enbSettingForm.carrierMode != '1'" label='<%=rb.getString("KaiQiJuHeZaiBo")%>' prop="carrierAggEnabled">
							<el-select v-model='enbSettingForm.carrierAggEnabled'>
								<el-option label="True" value='1'></el-option>
								<el-option label='False' value='0'></el-option>
							</el-select>
						</el-form-item>
					</div>
				</div>
			</el-collapse-item>
			<!-- CoreNetwork -->
			<el-collapse-item name="coreNetwork" v-show="hasKey('plmnId')">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">CoreNetwork</span>
					</p>
				</template>
				<div>
					<div v-show="isBaiblx_QRTB">
						<p v-if="hasKey('S1CBinding')" class='item-title-cls'>S1-C Config</p>
						<el-form-item v-if="hasKey('S1CBinding')" label="S1-C Binding" prop="S1CBinding">
							<el-select v-model="btsForm.S1CBinding">
								<el-option v-for="item in bandingList" :label="item.text" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<p v-if="hasKey('SGWSwitch')" class='item-title-cls'>S1-U Config</p>
						<el-form-item v-if="hasKey('SGWSwitch')" label="SGW Switch" prop="SGWSwitch">
							<el-switch v-model="btsForm.SGWSwitch" active-value="0" inactive-value="1" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
						</el-form-item>
						<el-form-item v-if="hasKey('SGWInterfaceBinding')" label="SGW Interface Binding" prop="SGWInterfaceBinding">
							<el-select v-model="btsForm.SGWInterfaceBinding" :disabled="btsForm.SGWSwitch =='1'">
								<el-option v-for="item in bandingList" :label="item.text" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
					</div>

					<p class='item-title-cls'>MME</p>
					<el-form-item label='TAC' prop='tac' class='validate-item'>
						<el-input v-model="enbSettingForm.tac">
							<template slot="append">Range:0~65535</template>
						</el-input>
					</el-form-item>
					<el-form-item label='S1 Link Port' prop="s1LinkPort" class='validate-item'>
						<el-input v-model="enbSettingForm.s1LinkPort">
							<template slot="append">Range:0~65535</template>
						</el-input>
					</el-form-item>
					<div class='list-cls'>
						<div class='list-item-cls'>
							<el-form-item label="PLMN" style='margin-bottom:0px;position:relative' class='validate-item' :class="plmnCls">
								<el-input v-model='plmnVal'>
									<template slot="append">Range:5~6,No more than 6,Not repeat</template>
								</el-input>
								<span v-if="plmnGroup.length<6" @click='addPlmn' class='form-bt el-icon el-icon-plus' style='position:absolute;left:165px;top:-5px;'></span>
							</el-form-item>
							<div style='overflow:auto;margin-bottom:20px;'>
								<el-form-item class='suffixItem' v-for='(domain,index) in plmnGroup' style='width:96px;'>
									<div class='form-suffix'>
										<span class='text'>{{domain}}</span>
										<span style='font-size:14px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click="removePlmn(index)"></span>
									</div>
								</el-form-item>
								<el-form-item prop="plmnId">
									<el-input v-model="enbSettingForm.plmnId" class="hide-input" style="border:none;width: 500px;"></el-input>
								</el-form-item>
							</div>
							<div v-show="showMME && enbSettingForm.halobEnable != '1'">
								<el-form-item label="MME IP" style='margin-bottom:0px;width: 100%;' :class="mmeCls">
									<el-input v-model='mmeVal'>
										<template slot="append">PLMN</template>
									</el-input>
									<el-select style='vertical-align:bottom;margin-left:-3px;' class='mmeSelect' v-model='mme_plmn'>
										<el-option v-for="item in plmnGroup" :label="item" :value="item"></el-option>
									</el-select>
									<span v-if="mmeGroup.length<16" class='el-icon el-icon-plus' style='vertical-align:middle;margin-left:1px;' @click='addMME("mme_plmn")'></span>
									<span class='item-tip'>No more than 16,Not repeat</span>
								</el-form-item>
								<div class="mmeIpListBoxCls">
									<div class='mmeIpItemBoxCls' v-for='(domain,index) in mmeGroup'>
										<div class='mmeItemContent'>
											<span class='mmeItemSpan mmeItemSpanFirst'>{{domain.mme}}</span>
											<span class='mmeItemSpan'>PLMN: {{domain.plmn}}</span>
											<span class='mmeItemRemove el-icon el-icon-close' @click='removeMME(index)'></span>
										</div>
									</div>
									<el-form-item prop="mmeStrNew" style="display:inline-block;">
										<el-input v-model="enbSettingForm.mmeStrNew" class="hide-input" style="border:none;width: 300px;"></el-input>
									</el-form-item>
								</div>
							</div>
							<div v-show="!showMME && enbSettingForm.halobEnable != '1'">
								<el-form-item label="MME IP" style='margin-bottom:0px;position:relative;min-width: 460px;' class="validate-item" :class="mmeCls">
									<el-input v-model='mmeVal'>
										<template slot="append">No more than 16,Not repeat</template>
									</el-input>
									<span v-if="mmeGroup.length<16" class='el-icon el-icon-plus form-bt' style='position:absolute;left:165px;top:-5px;' @click='addMME("mme")'></span>
								</el-form-item>
								<div class="mmeIpListBoxCls">
									<div class='mmeIpItemBoxCls' v-for='(domain,index) in mmeGroup'>
										<div class='mmeItemContent'>
											<span class='mmeItemSpan'>{{domain}}</span>
											<span class='mmeItemRemove el-icon el-icon-close' @click='removeMME(index)'></span>
										</div>
									</div>
									<el-form-item prop="mmeStrOld" style="display:inline-block;">
										<el-input v-model="enbSettingForm.mmeStrOld" class="hide-input" style="border:none;width: 300px;"></el-input>
									</el-form-item>
								</div>
							</div>
							<el-form-item label="S1 Connection Mode" prop="s1ConnectMode" class='validate-item'>
								<el-select v-model="enbSettingForm.s1ConnectMode">
									<el-option label='ALL' value='All'></el-option>
									<el-option label='ONE' value='One'></el-option>
								</el-select>
							</el-form-item>
						</div>
					</div>
				</div>
			</el-collapse-item>
			<!-- Cell -->
			<el-collapse-item name="cell">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Cell</span>
					</p>
				</template>
				<div style="">
					<p class='item-title-cls' v-show="hasKey('earfcn1')">Cell1</p>
					<div class='list-cls' v-show="hasKey('earfcn1')">
						<div class='list-item-cls'>
							<el-form-item label="<%=rb.getString("HostName")%>" prop='cellName1' class='validate-item'>
								<el-input v-model="enbSettingForm.cellName1">
									<template slot="append">Range:0~64</template>
								</el-input>
							</el-form-item>
							<el-form-item label="EARFCN DL" class='validate-item' prop="earfcn1">
								<el-input v-model="enbSettingForm.earfcn1" :disabled="sasDisabled">
									<template slot="append">
										<span style='display:inline-block;width:50px;height:24px;border:1px solid #A0C4F9;background:#F2F6FF;vertical-align:top;border-radius:2px;text-align:center;line-height:24px;color:#333'>
											Band:
											<span>{{enbSettingForm.band1}}</span>
										</span>
										<span style='margin-left:10px;vertical-align:sub;white-space: break-spaces;'>{{earfcn1Tips.freq}} {{earfcn1Tips.tips}}</span>
									</template>
								</el-input>
							</el-form-item>
							<el-form-item v-show="!(isBaiblq || isBaiblx_BLQ)" label="<%=rb.getString("ZiZhenPeiBi")%>" prop="subframe">
								<el-select v-model="enbSettingForm.subframe">
									<el-option label='1(DL:UL = 2:2)' value='1'></el-option>
									<el-option label='2(DL:UL = 3:1)' value='2'></el-option>
									<el-option label='6(DL:UL = 3:5)' value='6'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="PCI" prop="pci1" class='validate-item'>
								<el-input v-model="enbSettingForm.pci1">
									<template slot='append'>Range:0~503</template>
								</el-input>
							</el-form-item>
						</div>
						<div class='list-item-cls'>
							<el-form-item label="ECI" prop="eci1" class="validate-item">
								<el-input v-model="enbSettingForm.eci1">
									<template slot="append">Range:0~268435455</template>
								</el-input>
							</el-form-item>
							<el-form-item label="<%=rb.getString("DaiKuan")%>" prop="bandwidth">
								<el-select v-model="enbSettingForm.bandwidth" :disabled="sasDisabled">
									<el-option label='5M' value='25'></el-option>
									<el-option label='10M' value='50'></el-option>
									<el-option label='15M' value='75'></el-option>
									<el-option label='20M' value='100'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item v-show="!(isBaiblq || isBaiblx_BLQ)" label="<%=rb.getString("TeShuZiZhenPeiBi")%>" prop="specialSubframe">
								<el-select v-model="enbSettingForm.specialSubframe">
									<el-option label='5' value='5'></el-option>
									<el-option label='7' value='7'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="Power Modify" prop="powerModify1" style="width: 100%;">
								<el-select v-model="powerModify1Pre" :disabled="sasDisabled">
									<el-option label="2" value="2"></el-option>
								</el-select>
								<span style='margin:0 10px;'>X</span>
								<el-select v-model="enbSettingForm.powerModify1" class='mmeSelect' :disabled="sasDisabled">
									<el-option v-for="level in powrLevel" :label="level+'dBm'" :value="level+''"></el-option>
								</el-select>
							</el-form-item>
						</div>
					</div>
					<p class='item-title-cls' v-show="hasKey('cellName2') && (enbSettingForm.carrierAggEnabled == '1' || !hasKey('carrierAggEnabled'))">Cell2</p>
					<div class='list-cls' v-show="hasKey('cellName2') && (enbSettingForm.carrierAggEnabled == '1' || !hasKey('carrierAggEnabled'))">
						<div class='list-item-cls'>
							<el-form-item label="<%=rb.getString("HostName")%>" class='validate-item' prop="cellName2">
								<el-input v-model="enbSettingForm.cellName2">
									<template slot="append">Range:0~64</template>
								</el-input>
							</el-form-item>
							<el-form-item label="EARFCN DL" class='validate-item' prop="earfcn2">
								<el-input v-model="enbSettingForm.earfcn2" :disabled="sasDisabled">
									<template slot="append">
										<span style='display:inline-block;width:50px;height:24px;border:1px solid #A0C4F9;background:#F2F6FF;vertical-align:top;border-radius:2px;text-align:center;line-height:24px;color:#333'>
											Band:
											<span>{{enbSettingForm.band2}}</span>
										</span>
										<span style='margin-left:10px;vertical-align:sub;white-space: break-spaces;'>{{earfcn2Tips.freq}} {{earfcn2Tips.tips}}</span>
									</template>
								</el-input>
							</el-form-item>
							<el-form-item label="Power Modify" prop="powerModify2" style="width: 100%;">
								<el-select v-model="powerModify2Pre">
									<el-option label="2" value="2"></el-option>
								</el-select>
								<span style='margin:0 10px;'>X</span>
								<el-select v-model="enbSettingForm.powerModify2" class='mmeSelect'>
									<el-option v-for="level in powrLevel" :label="level+'dBm'" :value="level+''"></el-option>
								</el-select>
							</el-form-item>
						</div>
						<div class='list-item-cls'>
							<el-form-item label="ECI" prop="eci2" class='validate-item'>
								<el-input v-model="enbSettingForm.eci2">
									<template slot="append">Range:0~268435455</template>
								</el-input>
							</el-form-item>
							<el-form-item label="PCI" prop="pci2" class='validate-item'>
								<el-input v-model="enbSettingForm.pci2">
									<template slot="append">Range:0~503</template>
								</el-input>
							</el-form-item>
						</div>
					</div>
				</div>
			</el-collapse-item>
		</el-collapse>
	</el-form>
</div>
<script>
	new Vue({
		el:'#quickSettingPanel',
		data(){
			var vm = this;
				validateRange = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					var reg = /^(\d+\.\.){0,1}(\d+)$/;
					if(value == '' || value == undefined){
						callback();
					}else{
						if(!reg.test(value) || value < min || value > max){
							callback(new Error('format error'))
						}else{
							callback();
						}
					}
				},
				validateLength = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					if(value.length <  min || value.length > max){
						callback(new Error('format error'));
					}else{
						callback();
					}
				},
				validateEarfcn1 = (rule,value,callback)=>{
					var earfcn = vm.enbSettingForm.earfcn1,
						band = vm.enbSettingForm.band1,
						result = checkEarfcnByBand(earfcn, band);
					
					if(result.valid) {
						return callback();
					}else {
						return callback('<%=rb.getString("PinLv")%> <%=rb.getString("FanWei")%>: ' + JSON.stringify(result.range));
					}
				},
				validateEarfcn2 = (rule,value,callback)=>{
					var earfcn = vm.enbSettingForm.earfcn2,
						band = vm.enbSettingForm.band2,
						result = checkEarfcnByBand(earfcn, band);
					
					if(result.valid) {
						return callback();
					}else {
						return callback('<%=rb.getString("PinLv")%> <%=rb.getString("FanWei")%>: ' + JSON.stringify(result.range));
					}
				};

			var powerLevel = [0,1,2,3,4,5,6,7,8,9,10, 11,12,13,14,15,16,17,18,19,20, 21,22,23,24,25,26,27,28,29,30];

			if(settingVue.selectedRow && (settingVue.selectedRow.product == 'MLQ' || settingVue.selectedRow.product == 'BAIBLQ')) {
				powerLevel = [0,1,2,3,4,5,6,7,8,9,10, 11,12,13,14,15,16,17,18,19,20, 21,22,23,24,25,26,27,28,29,30, 31,32,33,34,35,36,37,38,39,40, 41,42,43,44,45,46];
			};
			/*
			if(settingVue.selectedRow && settingVue.selectedRow.product == 'BAIBLQ') {
				powerLevel = [0,1,2,3,4,5,6,7,8,9,10, 11,12,13,14,15,16,17,18,19,20, 21,22,23,24,25,26,27,28,29,30, 31,32,33,34,35,36,37,38,39,40, 41,42,43,44,45,46,47,48,49,50, 51,52];
			};
			*/
				
			return{
				isBaiblx_QRTB: settingVue.selectedRow.platformType == 'BLX' && settingVue.selectedRow.network_model == 'TDDMode',
				isBaiblx_BLQ: settingVue.selectedRow.platformType == 'BLX' && settingVue.selectedRow.network_model == 'FDDMode',
				isBaiblq: settingVue.selectedRow.product == 'BAIBLQ' || settingVue.selectedRow.product == 'MLQ',
				rebootMap: {},

				powrLevel: powerLevel,
				enbSettingForm:{
					sasEnable: '',
					halobEnable:'',
					epcSwitch:'1',
					duplexMode:'TDDMode',
					carrierMode:'',
					carrierAggEnabled: '',
					tac: '',
					plmnId:'',
					mmePort:'',
					mmeStrOld:'',
					mmeStrNew:'',
					s1ConnectMode: '',
					s1LinkPort:'',
					cellName1:'',
					eci1:'',
					earfcn1:'',
					bandwidth:'',
					band1:'',
					pci1:'',
					powerModify1: '',
					subframe:'',
					specialSubframe:'',
					cellName2:'',
					eci2:'',
					pci2:'',
					earfcn2:'',
					band2:'',
					powerModify2: ''
				},
				enbSettingRules:{
					tac:[{validator:validateRange,min:0,max:65535}],
					mmePort:[{validator:validateRange,min:1024,max:4915}],
					s1LinkPort:[{validator:validateRange,min:0,max:65535}],
					cellName1:[{validator:validateLength,min:0,max:64}],
					earfcn1:[{validator: validateEarfcn1,min:0,max:64}],
					eci1:[{validator:validateRange,min:0,max:268435455}],
					pci1:[{validator:validateRange,min:0,max:503}],
					cellName2:[{validator:validateLength,min:0,max:64}],
					eci2:[{validator:validateRange,min:0,max:268435455}],
					earfcn2:[{validator: validateEarfcn2,min:0,max:64}],
					pci2:[{validator:validateRange,min:0,max:503}],
				},
				activeNames:['quickSetting','coreNetwork','cell'],
				plmnGroup:[],
				plmnVal:'',
				mmeVal:'',
				mme_plmn:'',
				mmeGroup:[],
				bandingList: [],
				smallCellCode:'',
				tabId:'',
				casts:{
					'9953B58D9C6567B7516D1E8B9DA6B18F':'sasEnable',
					'73ED21EFA1E1054A54F33F6FE2B15401':'halobEnable',
					'4B392A75B2C3A31DE9DDDEA19AD50C25':'epcSwitch',
					'948074755A03AA2E305EB4F9AC9B140B':'duplexMode',
					'9C10D099A389877A8574183ECF70E474':'carrierMode',
					'3E299D67E9BEEEFDD85D81ADF1CD6D4E':'carrierAggEnabled',
					'9818F8FC383C9962DCB1EB631D1835E7':'plmnId',
					'7459B91FBA36019A9B36D14CE14F1AE3':'tac',
					'0938599C921C48FDF9480C7379157D5E':'mmeStrOld',
					'8E2FEBDF846E8C515B9556DF1FF26720':'mmeStrNew',
					'72FE6678A32C36E37C4F754497C2D043':'s1LinkPort',
					'61DE2B4642F63A0EF2DA4ECAD43B63C3':'s1ConnectMode',
					'737D034F342AB76AADDF28E68EDCD450':'cellName1',
					'41C594323E596A40159B50F0BCD27804':'eci1',
					'99A45DEA9E161AD1889A3CE086F96650':'earfcn1',
					'AA2424739E0E2FF0DFB931532936A9B1':'bandwidth',
					'471F9105CEA45873D68C77D631A100C9':'band1',
					'7FF6D39C78BDB31FA7A2986E0C906851':'subframe',
					'3E2892BDD2CDB99E34BB51DB7FCC3CE7':'specialSubframe',
					'205ACD2630B5A17773BD06CE742FE337':'pci1',
					'32C4F0242AC4FFCBA4EB826CC92AF5A8':'powerModify1',
					'C48E3E2BA83F6CD74D55F52C32E2890A':'cellName2',
					'02FF5BF4BBBBF73C5805E4C8F331E478':'eci2',
					'E339AFBF7E7DD76158F1F6757918DC60':'earfcn2',
					'C3CE3A3C6F422250C33E5C6A4A145A18':'band2',
					'6DEB4263A0675491B4EA34FACA947B69':'pci2',
					'33861BCB5CBDA9E14AA54A82ADFA60E0':'powerModify2',
					
					'75A10414080455D56DD1AF9C31FB408A':'sasEnable',
					'A90578CA6D8E7D13CDF9BFB111D27F52':'halobEnable',
					'4317FFD20C9DEC4EE3AA7507C88608CE':'epcSwitch',
					'51E5C49555C7494F6F92283D864205E2':'duplexMode',
					'E6756975BFF0AEB8CE0950B97CF3D3E7':'carrierMode',
					'4AEFB7C3622A5D7A2F280B7ABF42AFD1':'carrierAggEnabled',
                    'AF72CD9755ABA3EBF0953E09EFFF812D':'plmnId',
                    '86800B3CD4F077D7D78AF99E270F9C68':'tac',
                    '0270260D06962A9262770A4CE881DC89':'mmeStrOld',
                    '6A6C9F605C8BA09DD07FA80F4CBC43D2':'mmeStrNew',
                    '4DC7D7609581E70BE0527614D25390E4':'s1LinkPort',
                    '562CDCB77C3578B50FEAA7CD9A94BB3D':'s1ConnectMode',
					'3F3D7B02ADC71F30802B32AC34C1D9B7':'cellName1',
					'D6A052A7BD50BC036CFE732A7FADBB86':'eci1',
					'44C2CACF5388FBA7F24C1BFD39E859B9':'earfcn1',
					'1584EB9A8D4017B4D0785F22ADD91BF4':'bandwidth',
					'3AB59B04CD038A95D5DCE1227F3C93EC':'band1',
					'9AC22799859D844892D8652B55859D34':'subframe',
					'3C3F13F50B7E94E157EE5A94A0C77585':'specialSubframe',
					'A5EF335C6218179B8730B88B82C7A48C':'pci1',
					'9CF06A1C8C43F2DE2B279DAAEC21F521':'powerModify1',
					'6E915F5DB4E5CC395CD9C38D55CF1EE7':'cellName2',
					'B4909252320BE78FB61A01B4BC54A81A':'eci2',
					'CFF57AF78FEE2BF2C080C5934AA8487F':'earfcn2',
					'BE4FAA9A6B78D996A475BDB6F936FA39':'band2',
					'54465DA60B13352229BB2F52E9C029B2':'pci2',
					'50D3DBC91123C445197674EC61EEDD6F':'powerModify2',
					
					'8505DB34F8FBD5C285C115A139634EFA':'sasEnable',
					'BDA2883E2911EAAAF56D93FC0E0B07BB':'halobEnable',
					'DA0A8F40C58C38E4F5ACEA400B0BDF79':'epcSwitch',
					'BB0159C02B548290604EBFA92D2EE908':'duplexMode',
					'DF6DE44DEDC4F9B6B29156C21975CAF5':'carrierMode',
					'667FAA194DF6ACB4B5911AC81D48AFEA':'carrierAggEnabled',
                    '47C568C2BBB8E1ED6C7B60124760FF86':'plmnId',
                    'D301846DD30EE0273BB84154DBF6DA81':'tac',
                    'F1C9614E593F9E1D9FBD9F63E3FF7964':'mmeStrOld',
                    'A5812C2A72103FCE4957E44B3FCBCA1F':'mmeStrNew',
                    'E6E40A5A7C55B81DEA38D793B815ABBB':'s1LinkPort',
                    'C7C698BB9B4048AC1BF44F4082A429C1':'s1ConnectMode',
					'D9783542D396D0FE8487A1859828F691':'cellName1',
					'F7BE7088B7BBAA853CA6F4DD2883BF2D':'eci1',
					'4B5638319FF2CD0B4B1849528E800CE9':'earfcn1',
					'C363467EFB6BF09F46C44B0A151716E7':'bandwidth',
					'8F8A1DC9E80F3235FCDCF4ED3B47B998':'band1',
					'9AC22799859D844892D8652B55859D34':'subframe',
					'3C3F13F50B7E94E157EE5A94A0C77585':'specialSubframe',
					'9C960682C92E88F9DC2D64300A7BA2AD':'pci1',
					'09A386318D071995656E9F67C8E59DCD':'powerModify1',
					'BB523DA88F6C219077D5696DFE20DFDD':'cellName2',
					'686497B3B63E31C225A7F56A306DB328':'eci2',
					'D4BCEB1E377A528BDEA3A834C071EF9C':'earfcn2',
					'F9F441121DB457DE89A2733AF9FB3055':'band2',
					'70C483DFB535ABC77E3AA735451A11E0':'pci2',
					'961BCD862AB29D32A33D416E02881D0B':'powerModify2',

					'6141CDBD07DF8E58F7C9DD976CF7B258': 'halobEnable',
					'35FEA7BE4C81EF58037EFF2344C2B639': 'epcSwitch',
					'4F1051012D45CB56CBED8FAED9AFAF4E': 'duplexMode',
					'37F7F4B90630B3F48AA0725AD9F82DBB': 'carrierMode',
					'C1F6A1F86E2121C0550D5167DE38E3EF': 'carrierAggEnabled',
					'51E56A7D4DCEABF9C10EDC04550EBAE0': 'plmnId',
					'196D95A950F432F94313FC8A684001DE': 'tac',
					'1CFA728B3A73A1C1019B2E02AA192A93': 'mmeStrNew',
					'13DA4971ADD0AFDD30947136D034D3B3': 's1LinkPort',
					'A866AE1D7A2009ED96770B0A412DB07C': 's1ConnectMode',
					'F3D26E61E4BB02241E3AD8364E9892BF': 'cellName1',
					'136054B0640C44FD647C3D3F0C62D115': 'eci1',
					'E8B5268FA71160424BD0760990CC6258': 'bandwidth',
					'0AEECA6F2B357DD71E8C25FF7F5ADEDA': 'earfcn1',
					'BE31315DEDF1B015DEB3D99D81DA690A': 'band1',
					'57F5E34F003BC528E88DCC2A1F1523AE': 'pci1',
					'AECC60EF6654202E56C418508C6B2FAA': 'powerModify1',
				    'F157CB9B164E63BDE2B692DB7DE22C6E': 'S1CBinding',
				    'E459CE45841630B0AD14260AD831EE69': 'SGWSwitch',
				    '5BF67AF3A0CB3D0B7CC4A25A1923CCE9': 'SGWInterfaceBinding',

					'EB7233193B21C34458BCA52929C8A25D': 'specialSubframe',
					'E21F9FB272E49C8FA7A03BA07EE288B4': 'subframe',
				},
				powerModify1Pre: '2',
				powerModify2Pre: '2',
				codeList:[],
				plmnCls:'',
				mmeType:'',
				mmeCls:'',
				showMME:true,
				ipsecUrl:'',

				cell2Sasenable: true
			}
		},
		computed: {
			sasDisabled() {
				return [1,'1','on'].includes(this.enbSettingForm.sasEnable);
			},
			carrierFlag() {
				return this.enbSettingForm.carrierMode == '2';
			},
			showSasInfo() {
				return this.enbSettingForm.sasEnable == '1';
			},
			epcSwitchFlag() {
				return this.enbSettingForm.halobEnable != '1';
			},
			earfcn1Tips() {
				var vm = this,
					earfcn = vm.enbSettingForm.earfcn1,
					band = vm.enbSettingForm.band1,
					result = checkEarfcnByBand(earfcn, band);
				
				return {
					freq: earfcnFormatter(earfcn),
					tips: '<%=rb.getString("PinLv")%> <%=rb.getString("FanWei")%>: ' + JSON.stringify(result.range)
				};
			},
			earfcn2Tips() {
				var vm = this,
					earfcn = vm.enbSettingForm.earfcn2,
					band = vm.enbSettingForm.band2,
					result = checkEarfcnByBand(earfcn, band);
				
				return {
					freq: earfcnFormatter(earfcn),
					tips: '<%=rb.getString("PinLv")%> <%=rb.getString("FanWei")%>: ' + JSON.stringify(result.range)
				};
			}
		},
		methods:{
			initReboot(list, map) {
				var vm = this;

				list.map(function(item){
					if(item.reboot == '1') {
						var code = item.name,
							key = vm.casts[code];

						map[key] = true;
					}
				});
			},
			init(code,id){
				var vm = this;
				vm.smallCellCode = code;
				vm.tabId = id;
				vm.getParamNode(code,id);
			},
			getParamNode(code,id) {
                var vm = this,
                    codes = [],
                    url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                    params = {
                        id: id,
                        smallCellCode: code
                    };

                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data;
                    
                    if(data && Array.isArray(data)) {
                        data.map(function(item){
                            item.groups.map(function(group){
                                group.list.map(function(m){
                                    codes.push(m.name);
                                    // 执行赋值
                                    vm.setValue(m);
                                    if(m.label == 'MME IP'){
                                    	vm.showMME = m.type == 'bind' ? true : false;
                                    	if(vm.showMME) {
                                    		m.value.split(';').map(function(mmePlmn){
                                    			var mmeArr = mmePlmn.split(',');
                                    			
                                    			vm.mmeGroup.push({mme: mmeArr[0], plmn: mmeArr[1]})
                                    		});
                                    	}else {
											m.value.split(',').map(function(mmeIp){
												vm.mmeGroup.push(mmeIp)
                                    		});
                                    	}
                                    }

									if(m.name == 'E339AFBF7E7DD76158F1F6757918DC60') {
										vm.cell2Sasenable = m.readonly == '1';
									}

									if('803E94F9CB7469C2170F6399E7E12ADC' == m.name) {
										if(m.data) {
											var bandList = JSON.parse(m.data);

											vm.bandingList = bandList;
										}
									}
                                });
								// 初始化重启项关系记录
								vm.initReboot(group.list, vm.rebootMap);
                            });
                        });
                        
                        // init plmnGroup
                        vm.enbSettingForm.plmnId.split(',').map(function(plmn){
                        	vm.plmnGroup.push(plmn);
                        });
						
                        vm.$nextTick(function(){
                            initForm(vm.$refs.enbSettingForm);
                            detectReboot(vm.$refs.enbSettingForm, vm.rebootMap);
							$('#setting_main').removeClass('loading');
                        });

                        vm.codeList = codes;
                    }
                });
            },
            setValue(item) {
            	var vm = this,
                code = item.name,
                value = item.value;

	            // indexs是否含有
	            var key = vm.casts[code];
	            try{
	            	if(key){
		            	vm.enbSettingForm[key] = value;
		            }
	            }catch(e){}
            },
            hasKey(key) {
                var vm = this,
                    has = false;

                vm.codeList.map(function(name){
                    if(vm.casts[name] == key) has = true;
                });

                return has;
            },
            getNameByProp(prop) {
                var vm = this,
                    reg = /^\w*\.\d*\.\w*$/,
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
            goProcedure() {
				var serialNumber = settingVue.selectedRow.serial_number;

				sessionStorage.setItem('submenuid', '1000100');
            	eventAllBus.$emit("gomenupage","10000","",'10000',{},function(){
    				eventBus.$emit('goProcedure',{serialNumber: serialNumber, deviceType: 'eNB'});
    			});
            },
			carryModeChange(val) {
				var vm = this;

				if(val == '1') {
					vm.enbSettingForm.carrierAggEnabled = '0';
				}
			},
			addPlmn(){
            	var value = this.plmnVal;
            	var reg = /^(\d+\.\.){0,1}(\d+)$/,
					existed = false;

				this.plmnGroup.map(function(item){
					if(item == value) existed = true;
				});

            	if(existed || value == '' || !reg.test(value) || value.length < 5 || value.length > 6 || value == undefined || this.plmnGroup.length > 5){
					this.plmnCls = 'is-error';
				}else{
					this.plmnCls = '';
					this.plmnGroup.push(value);
					this.plmnVal = '';
				}
			},
			removePlmn(index){
				this.plmnGroup.splice(index,1);
				this.plmnCls = '';
			},
			isMMEExisted(type) {
				var vm = this,
					mme = vm.mmeVal,
					plmn = vm.mme_plmn,
					existed = false;

				vm.mmeGroup.map(function(item){
					if(type == 'mme'){
						if(item == mme) existed = true;
					}else {
						if(item.mme == mme && item.plmn == plmn) existed = true;
					}
				})

				return existed;
			},
			addMME(type){
				var value = this.mmeVal;
				this.mmeType = type;

            	if(!isValidIP(value) || this.isMMEExisted(type)){
					this.mmeCls = 'is-error';
				}else{
					this.mmeCls = '';
					if(type == 'mme'){
						this.mmeGroup.push(this.mmeVal);
					}else{
						this.mmeGroup.push({mme:this.mmeVal,plmn:this.mme_plmn})
					}
					this.mmeVal = '';
				}
			},
			removeMME(index){
				this.mmeVal = '';
				this.mme_plmn = '';
				this.mmeGroup.splice(index,1);
			},
			isNull(val){
                if(val==undefined || val == null || val =="") return true;
                else return false;
            },
			save() {
                var vm = this;
                var params = {},
                    isChanged = isFormChanged(vm.$refs.enbSettingForm);

                if(!isChanged){
                    showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
                    return;
                }

                vm.$refs.enbSettingForm.fields.map(function(field){
                    var key = vm.getNameByProp(field.prop);

                    if(Array.isArray(field.fieldValue)){
                        var vList = field.fieldValue.map(function(item){return item}),
                            oList = (field.reinitialValue||[]).map(function(item){return item}),
                            val = vList.sort().join(','),
                            orVal = oList.sort().join(',');

                        if(val != orVal) {
                            params[key] = val;
                        };
                    }else{
                        if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
                            
                        }else if(field.fieldValue != field.reinitialValue) {
                            params[key] = field.fieldValue;
                        };
                    }
                });
                
                vm.$refs.enbSettingForm.validate(function(valid){
                    if(valid) {
						var isNeedReboot = detectReboot(vm.$refs.enbSettingForm, vm.rebootMap);

						if(isNeedReboot) {
							var tipContent = [
									'<%=rb.getString("JiZhanChongQiTiShi")%>',
									'<br/><br/>',
									'<input id="reboot_confirm_status" type="checkbox" />',
									'<label for="reboot_confirm_status" style="font-size: 14px;color: #1DA3FC;cursor: pointer;"><%=rb.getString("SheZhiHouChongQi")%></label>'
								].join(" ");

							var msger = $.messager.confirm('<%=rb.getString("QueRen")%>', tipContent, function (r) {
								if (r) {
									/* 重启勾选判断 */
									var needReboot = false,
										rebootCkbox = $('#reboot_confirm_status',msger);
									if(rebootCkbox.length && rebootCkbox.prop('checked')){
										needReboot = true;
									}
									msger = null;
									
									var rowCode = vm.smallCellCode,
										url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;

									$('#setting_main').addClass('loading');
									settingVue.submitDisabled = true;
									axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
										var data = res.data;
										if(data["success"]){
											vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});

											// 勾选重启，下发重启指令
											if(needReboot) {
												$.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
													if (!data["success"]) {
														showMsg('error_msg',data["message"]);
													}
												}, "json");
											}
											closeSettingPanel();
										}else{
											vm.$message.error(data["message"]);
										}

										$('#setting_main').removeClass('loading');
										settingVue.submitDisabled = false;
									})
								}
							}).addClass("seriousConfirm");
						}else {
							var rowCode = vm.smallCellCode,
								url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;

                            $('#setting_main').addClass('loading');
							settingVue.submitDisabled = true;
							axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
								var data = res.data;
								if(data["success"]){
									vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
									closeSettingPanel();
								}else{
									vm.$message.error(data["message"]);
								}

								$('#setting_main').removeClass('loading');
								settingVue.submitDisabled = false;
							})
						}
                    }
                });
            },
            cancel(){
            	var vm = this;
				if(isFormChanged(vm.$refs.enbSettingForm)){//返回true为改变
					vm.$confirm("<%=rb.getString("QueDingLiKaiDangQianYeMian")%>",'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						closeSettingPanel();
					}).catch(() => {})
				}else{
					closeSettingPanel();
				}
            }
		},
		watch:{
			plmnGroup:function(){
				this.enbSettingForm.plmnId = this.plmnGroup.toString();
			},
			mmeGroup:function(){
				if(this.mmeType == 'mme'){
					this.enbSettingForm.mmeStrOld = this.mmeGroup.join(',');
				}else{
					var str = '';
					this.mmeGroup.map(item=>{
						str += item.mme + ',' + item.plmn + ';'
					});
					this.enbSettingForm.mmeStrNew = str;
				}
			},
			enbSettingForm: {
				handler: function(newVal, oldVal) {
					var vm = this,
						form = vm.$refs.enbSettingForm;
					
					detectReboot(form, vm.rebootMap);
				},
				deep: true
			}
		},
		mounted(){
			eventBus.$off('tab-param').$on('tab-param',this.init);
			eventBus.$off('save-set').$on('save-set',this.save);
			eventBus.$off('cancel-set-tab').$on('cancel-set-tab',this.cancel);
		}
	})
	
	function checkEarfcnByBand(earfcn, band) {
		var rels = {
				1: [2110, 2170],      2: [1930, 1990],   3: [1805, 1880],   4: [2110, 2155],      5: [869, 894], 
				6: [875, 885],        7: [2620,2690],    8: [925, 960],     9: [1844.9, 1879.9],  10: [2110, 2170],
				11: [1475.9, 1495.9], 12: [729, 746],    13: [746, 756],    14: [758, 768],       17: [734, 746],
				18: [860, 875],       19: [875, 890],    20: [791, 821],    21: [1495.9, 1510.9], 22: [3510, 3590],
				23: [2180, 2200],     24: [1525, 1559],  25: [1930, 1995],  26: [859, 894],       27: [852, 869],
				28: [758, 803],       29: [717, 728],    30: [2350, 2360],  31: [462.5, 467.5],   32: [1452, 1496],
				33: [1900, 1920],     34: [2010, 2025],  35: [1850, 1910],  36: [1930, 1990],     37: [1880, 1930],
				38: [2570, 2620],     39: [1880, 1920],  40: [2300, 2400],  41: [2496, 2696],     42: [3400, 3600],
				43: [3600, 3800],     44: [703, 803],    45: [1447, 1467],  46: [5150, 5925],     47: [5855, 5925],
				48: [3550, 3700],     49: [3550, 3700],  50: [1432, 1517],  51: [1427, 1432],     52: [3300, 3400],
				53: [2483.5, 2495],   65: [2110, 2200],  66: [2110, 2200],  67: [738, 758],       68: [753, 783],
				69: [2570, 2620],     70: [1995, 2020],  71: [617, 652],    72: [461, 466],       73: [460, 465],
				74: [1475, 1518],     75: [1432, 1517],  76: [1427, 1432],  85: [728, 746],       87: [420, 425],
				88: [422, 427]
			},
			key = (band||'').trim(),
			range = rels[key],
			result = {
				valid: false,
				range: []
			};
		
		if(range) {
			var freqStr = earfcnFormatter(earfcn),
				freq = freqStr?freqStr.match(/[0-9.]*MHz/g)[0].replace('MHz',''):'',
				min = range[0], max = range[1];
			
			if(freq >= min && freq <= max) {
				result.valid = true;
			}
			
			result.range = range;
		}else {
			result.valid = true;
		}
		
		return result;
	}
</script>