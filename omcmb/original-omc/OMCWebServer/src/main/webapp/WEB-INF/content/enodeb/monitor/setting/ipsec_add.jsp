<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp"%>
<div id='ipsecEditPanel'>
	<el-form ref="ipsecForm" :model="ipsecForm" :rules="ipsecRules" 
		label-position="top" label-width="120" style='width:100%;height:100%;' inline>
		
		<div style='height:37px;border-bottom:1px solid #EEE;line-height:37px;font-weight:bold;padding-left:10px;font-size:14px;background-color: #fff;'>
				Add IPSec Tunnel List
				<div class="circleIcon placeholder-bt" style="top: 4px;right:40px;" placeholder="<%=rb.getString("FanHui")%>">		
					<span class="el-icon el-icon-circle-goback" @click='closeAddIpsec'></span>
				</div>
		</div>
		<el-collapse v-model="activeNames">
			<!-- Quick Setting -->
			<el-collapse-item name="configure">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">IPSec Tunnel/Configure</span>
					</p>
				</template>
				<div style="">
					<div class='list-cls'>
						<div class='list-item-cls'>
							<el-form-item label="Enable" prop="configEnable" style="height: 60px;">
								<el-switch v-model="ipsecForm.configEnable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
							</el-form-item>
							<el-form-item label="leftAuth" prop="leftAuth">
								<el-select v-model="ipsecForm.leftAuth">
									<el-option label="psk" value="0"></el-option>
									<el-option label="pubkey" value="1"></el-option>
									<el-option label="eap-aka" value="2"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="Gateway" prop="gateway">
								<el-input v-model="ipsecForm.gateway"></el-input>
							</el-form-item>
							<el-form-item label="leftId" prop="leftId">
								<el-input v-model="ipsecForm.leftId"></el-input>
							</el-form-item>
							<el-form-item label="leftCert" prop="leftCert">
								<el-input v-model="ipsecForm.leftCert"></el-input>
							</el-form-item>
							<el-form-item label="leftSourceIp" prop="leftSourceIp">
								<el-input v-model="ipsecForm.leftSourceIp"></el-input>
							</el-form-item>
						</div>
						<div class='list-item-cls'>
							<el-form-item label="rightAuth" prop="rightAuth">
								<el-select v-model="ipsecForm.rightAuth">
									<el-option label="psk" value="0"></el-option>
									<el-option label="pubkey" value="1"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="Right Subnet" prop="rightSubnet">
								<el-input v-model="ipsecForm.rightSubnet"></el-input>
							</el-form-item>
							<el-form-item label="rightId" prop="rightId">
								<el-input v-model="ipsecForm.rightId"></el-input>
							</el-form-item>
							<el-form-item label="secretKey" prop="secretKey">
								<el-input v-model="ipsecForm.secretKey"></el-input>
							</el-form-item>
							<el-form-item label="leftSubnet" prop="leftSubnet">
								<el-input v-model="ipsecForm.leftSubnet"></el-input>
							</el-form-item>
							<el-form-item label="Left" prop="leftIp">
								<el-input v-model="ipsecForm.leftIp"></el-input>
							</el-form-item>
						</div>
					</div>
				</div>
			</el-collapse-item>
			<el-collapse-item name="advance">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">IPSec Tunnel/Advance</span>
					</p>
				</template>
				<div style="">
					<div class='list-cls'>
						<div class='list-item-cls'>
							<el-form-item label="IKE Encryption" prop="ikeEncryption">
								<el-select v-model="ipsecForm.ikeEncryption">
									<el-option label='aes128' value="0"></el-option>
									<el-option label='aes256' value="1"></el-option>
									<el-option label='3des' value="2"></el-option>
									<el-option label='des' value="3"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="IKE Authentication" prop="ikeAuthentication">
								<el-select v-model="ipsecForm.ikeAuthentication">
									<el-option label='sha1' value="0"></el-option>
									<el-option label='sha1_160' value="1"></el-option>
									<el-option label='sha256_96' value="2"></el-option>
									<el-option label='sha256' value="3"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="ESP DH Group" prop="espDhGroup">
								<el-select v-model="ipsecForm.espDhGroup">
									<el-option label='modp768' value="0"></el-option>
									<el-option label='modp1024' value="1"></el-option>
									<el-option label='modp1536' value="2"></el-option>
									<el-option label='modp2048' value="3"></el-option>
									<el-option label='modp4096' value="4"></el-option>
									<el-option label='none' value="5"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="KeyLife" prop="keylife">
								<el-input v-model="ipsecForm.keylife"></el-input>
							</el-form-item>
							<el-form-item label="RekeyMargin" prop="RekeyMargin">
								<el-input v-model="ipsecForm.RekeyMargin"></el-input>
							</el-form-item>
							<el-form-item label="Dpddelay" prop="Dpddelay">
								<el-input v-model="ipsecForm.Dpddelay"></el-input>
							</el-form-item>
						</div>
						<div class='list-item-cls'>
							<el-form-item label="IKE DH Group" prop="ikeDhGroup">
								<el-select v-model="ipsecForm.ikeDhGroup">
									<el-option label='modp768' value="0"></el-option>
									<el-option label='modp1024' value="1"></el-option>
									<el-option label='modp1536' value="2"></el-option>
									<el-option label='modp2048' value="3"></el-option>
									<el-option label='modp4096' value="4"></el-option>
									<el-option label='none' value="5"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="ESP Encryption" prop="espEncryption">
								<el-select v-model="ipsecForm.espEncryption">
									<el-option label='aes128' value="0"></el-option>
									<el-option label='aes256' value="1"></el-option>
									<el-option label='3des' value="2"></el-option>
									<el-option label='des' value="3"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="ESP Authentication" prop="espAuthentication">
								<el-select v-model="ipsecForm.espAuthentication">
									<el-option label='sha1' value="0"></el-option>
									<el-option label='sha1_160' value="1"></el-option>
									<el-option label='sha256_96' value="2"></el-option>
									<el-option label='sha256' value="3"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="Fragmentation" prop="fragmentation">
								<el-select v-model="ipsecForm.fragmentation">
									<el-option label='Yes' value="0"></el-option>
									<el-option label='Accept' value="1"></el-option>
									<el-option label='Force' value="2"></el-option>
									<el-option label='No' value="3"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="IKELifeTime" prop="IKELifeTime">
								<el-input v-model="ipsecForm.IKELifeTime"></el-input>
							</el-form-item>
							<el-form-item label="Dpdaction" prop="Dpdaction">
								<el-select v-model="ipsecForm.Dpdaction">
									<el-option label='none' value="0"></el-option>
									<el-option label='clear' value="1"></el-option>
									<el-option label='hold' value="2"></el-option>
									<el-option label='restart' value="3"></el-option>
								</el-select>
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
		el:'#ipsecEditPanel',
		data(){
			var vm = this,
				validateTime = (rule,value,callback)=>{
					var reg = /^\d+[mhd]{1}$/,
						inValidMsg = '<%=rb.getString("ShuZiJiaMHD")%>';
					
					if(enbPlatform == '1'||true) {
						reg = /^\d+[smhd]{1}$/;
						inValidMsg = '<%=rb.getString("ShuZiJiaSMHD")%>';
					}
					
					if(value != "" && !reg.test(value)&& value.length<256 && value.length>0){
						callback(inValidMsg);
					}else{
						callback();
					}
				},
				validateKeyLife = (rule,value,callback)=>{
					var reg = /^\d+[s|m|h|d]{1}$/,
						rekeyVal = vm.ipsecForm.RekeyMargin;

					if(reg.test(rekeyVal) && reg.test(value)) {
						var keylifeTime = getSecondsTime(value),
							rekeyTime = getSecondsTime(rekeyVal),
							distance = keylifeTime - rekeyTime*3;

						if(distance >= 0) {
							callback();
						}else {
							callback('KeyLife and RekeyMargin must meet: KeyLife >= RekeyMargin*3');
						}
					}else {
						validateTime(rule,value,callback);
					}
				},
				validateReKeyMargin = (rule,value,callback)=>{
					var reg = /^\d+[s|m|h|d]{1}$/,
						minTypes = {
							'RTS&QRTB': '5m',
							V3: '3m'
						},
						type = getEnbType();

					if(minTypes[type] && reg.test(value)) {
						var prevTime = getSecondsTime(value),
							minTime = getSecondsTime(minTypes[type]),
							distance = prevTime - minTime;

						if(distance >= 0) {
							callback();
						}else {
							callback('RekeyMargin >= ' + minTypes[type]);
						}
					}else {
						validateTime(rule,value,callback);
					}
				},
				validateIkeLife = (rule,value,callback)=>{
					var reg = /^\d+[s|m|h|d]{1}$/,
						tarVal = vm.ipsecForm.keylife;

					if(reg.test(tarVal) && reg.test(value)) {
						var keylifeTime = getSecondsTime(value),
							rekeyTime = getSecondsTime(tarVal),
							distance = keylifeTime - rekeyTime;

						if(distance >= 0) {
							callback();
						}else {
							callback('IKELifeTime and KeyLife must meet: IKELifeTime >= KeyLife');
						}
					}else {
						validateTime(rule,value,callback);
					}
				};
			
			return{
				isBLQBLNMLQ: false,
				
				ipsecForm:{
					configEnable: 'true',
					leftAuth:'',
					rightAuth:'',
					gateway:'',
					rightSubnet:'',
					leftId:'',
					rightId:'',
					leftCert:'',
					secretKey:'',
					leftSourceIp:'',
					leftSubnet:'',
					leftIp: '',
					ikeEncryption:'',
					ikeDhGroup:'',
					ikeAuthentication:'',
					espEncryption:'',
					espDhGroup:'',
					espAuthentication:'',
					fragmentation: '',
					keylife:'',
					IKELifeTime:'',
					RekeyMargin:'',
					Dpdaction:'',
					Dpddelay:'',
					index:''
				},
				ipsecRules:{
					keylife:[{validator: validateKeyLife}],
					IKELifeTime:[{validator: validateIkeLife}],
					RekeyMargin:[{validator: validateReKeyMargin}]
				},
				activeNames:['configure','advance']
			}
		},
		methods:{
			closeAddIpsec(){
				eventBus.$emit('close-ipsec');
			},
			init(){
				if(networkVm.operType == 'edit'){
					this.ipsecForm = networkVm.rowData;
				}
			},
			save(){
				var vm = this;
				vm.$refs.ipsecForm.validate(function(valid){
					if(valid){
						var row = Object.assign({operateType: networkVm.operType},vm.ipsecForm);
						
						if(networkVm.operType == 'add'){
							row.index = networkVm.networkForm.ipsecList.length + 1 + '';
							networkVm.networkForm.ipsecList.push(row);
							networkVm.addIpsecFlag = networkVm.networkForm.ipsecList.length >= 2 ? false : true;
						}else{
							networkVm.networkForm.ipsecList[vm.ipsecForm.index-1] = row
						}
						networkVm.showIpsecAdd = false;
					}
				})
			}
		},
		mounted(){
			this.init();
			eventBus.$off('save-ipsec').$on('save-ipsec',this.save);
		}
	});

	function getIKEEnbType() {
		var type = '',
			product = settingVue.selectedRow.product;
	
		if(['RTS','RTD','QRTB','QRTB-CA','QRTB-DC','QRTB-SC'].includes(product)) {
			type = 'RTS&RTD&QRTB';
		}
		if(['QAFB'].includes(product)) {
			type = 'V3';
		}
	
		return type;
	}
	function getEnbType() {
		var type = '',
			product = settingVue.selectedRow.product;

		if(['RTS','QRTB','QRTB-CA','QRTB-DC','QRTB-SC'].includes(product)) {
			type = 'RTS&QRTB';
		}
		if(['QAFB'].includes(product)) {
			type = 'V3';
		}

		return type;
	}
	function getSecondsTime(time) {
		var units = {
				s: 1,
				m: 60,
				h: 3600,
				d: 86400
			},
			key = time.match(/[smhd]/)[0],
			value = time.match(/\d*/)[0],
			timeUnit = units[key],
			sTime = value*timeUnit;

		return sTime;
	}
</script>