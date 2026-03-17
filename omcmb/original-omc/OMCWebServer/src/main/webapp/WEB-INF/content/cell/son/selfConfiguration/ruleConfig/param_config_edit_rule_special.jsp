<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
#editConfigRuleDiv .el-collapse-item__header{
	border-bottom:1px solid #fff;
}
#editConfigRuleDiv .el-collapse-item__arrow{
	position:absolute;
	left:20px;
	top:0px;
}
#editConfigRuleDiv .el-collapse-item{
	position:relative;
}
#editConfigRuleDiv .el-icon-arrow-right{
	font-size:16px;
}
#editConfigRuleDiv .el-icon-arrow-right:before{
	content:"\e639";
	color:#BBB;
}
#editConfigRuleDiv .is-active.el-icon-arrow-right:before{
	content:"\e638";
	color:#BBB;
}
#editConfigRuleDiv .el-collapse-item__arrow.is-active{
	transform:rotate(0deg);
}
#editConfigRuleDiv .el-collapse{
	border-top:1px solid #fff;
	border-bottom:1px solid #fff;
}
#editConfigRuleDiv .el-collapse-item__wrap{
	border-bottom:1px solid #fff;
}
.inputItem .el-input{
	width:300px;
}
#editConfigRuleDiv .el-form-item{
	display:inline-block;
	vertical-align:top;
}
.basicConfig .el-form-item{
	margin-right:200px;
}
#editConfigRuleDiv .btn-next .el-icon-arrow-right:before{
	content:"\e794"
}
.el-input.is-disabled .el-input__inner {
	min-height: 25px;
	max-height: 25px;
}
#editConfigRuleDiv .el-collapse-item__header{
	max-width:200px;
}
</style>
<div id="editConfigRuleDiv" style="margin-top:10px;margin-left:20px;">
	<div style="width:95%;border:1px solid #F3F3F3;overflow:hidden" class="inputItem basicConfig">
		<el-form :disabled="viewDisabled" style="width:120%;" ref="basicForm" :model="basicForm" :rules="basicRules" label-position="top">
			<div style="padding: 15px 0px 0px 65px;">
				<el-form-item label="<%=rb.getString("HaloBKaiGuan")%>" prop="halob_enable">
					<el-select v-model="basicForm.halob_enable" @change="changeHalob">
						<el-option label="<%=rb.getString("KaiQi")%>" value="1"></el-option>
						<el-option label="<%=rb.getString("GuanBi")%>" value="0"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item v-if="showMME" label="MME" style="position:relative;margin-bottom:0px;">
					<el-input v-model="basicForm.mme"></el-input>
					<span v-if="showMMEOper" @click="addMME" class="el-icon el-icon-plus" style="position:absolute;left:275px;top:5px;"></span>
					<div style="margin-top:5px;width:305px;">
						<div class='suffixItem' v-for='(domain,index) in basicForm.mmeGroup' style='line-height:16px;display:inline-block'>
							<div class='form-suffix'>
								<span class='text'>{{domain}}</span>
								<span v-if="showMMEOper" style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeMME(domain)'></span>
							</div>
						</div>
						<p style='color:red;font-size:12px;'>{{errorMsg}}</p>
					</div>
				</el-form-item>
				<el-form-item prop="mmeStr" style="margin-right:0px;">
					<el-input v-model="basicForm.mmeStr" v-show=false></el-input>
				</el-form-item>
				<el-form-item v-show="false" label="<%=rb.getString("ZaiBoLeiXing")%>">
					<el-select v-model="basicForm.carrier_mode">
						<el-option label="<%=rb.getString("DanZaiBo")%>" value="1"></el-option>
						<el-option label="<%=rb.getString("ShuangZaiBo")%>" value="2"></el-option>
					</el-select>
					<el-checkbox v-model="basicForm.carrierAggregationEnable"><%=rb.getString("KaiQiJuHeZaiBo")%></el-checkbox>
				</el-form-item>
			</div>
			<el-collapse v-model="activeNames">
				<!-- Basic -->
				<el-collapse-item name=basic>
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">Cell1</span>
						</p>
					</template>
					<div style="margin-left:65px;">
						<el-form-item label="<%=rb.getString("ZhiChiPinDuan")%>" prop="bands_support">
							<el-input v-model="basicForm.bands_support"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("DaiKuan")%>" prop="band_width" v-show='shownbi'>
							<el-select v-model="basicForm.band_width" :disabled="sasDisabled">
								<el-option label="5MHz" value="n25"></el-option>
								<el-option label="10MHz" value="n50"></el-option>
								<el-option label="15MHz" value="n75"></el-option>
								<el-option label="20MHz" value="n100"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item :label="frequencyLabel" prop="frequency">
							<el-input v-model="basicForm.frequency" @blur="changeToFrequency('dl')" @focus="changeToEarfcn('dl')" :disabled="sasDisabled"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("ZiZhenPeiBi")%>" prop="subframe_assignment">
							<el-select v-model="basicForm.subframe_assignment">
								<el-option label="<%=rb.getString("QingXuanZe")%>" value=""></el-option>
								<el-option label="0(DL:UL = 1:3)" value="0"></el-option>
								<el-option label="1(DL:UL = 2:2)" value="1"></el-option>
								<el-option label="2(DL:UL = 3:1)" value="2"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("TeShuZiZhenPeiBi")%>" prop="special_subframe_patterns">
							<el-select v-model="basicForm.special_subframe_patterns">
								<el-option label="<%=rb.getString("QingXuanZe")%>" value=""></el-option>
								<el-option label="5" value="5"></el-option>
								<el-option label="7" value="7"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("PLMN")%>" prop="plmn_id">
							<el-input v-model="basicForm.plmn_id"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("TAC")%>" prop="tac">
							<el-input v-model="basicForm.tac"></el-input>
						</el-form-item>
						<el-form-item label="ECI (ECI=eNB_ID*256+Cell_ID)" prop="cell_identity">
							<el-input v-model="basicForm.cell_identity"></el-input>
						</el-form-item>
						<el-form-item label="PCI" prop="phycellid">
							<el-input v-model="basicForm.phycellid"></el-input>
						</el-form-item>
						<el-form-item prop="root_sequence_index" label="<%=rb.getString("GenXuLieSuoYin")%>">
							<el-input v-model="basicForm.root_sequence_index"></el-input>
						</el-form-item>
					</div>
				</el-collapse-item>
			</el-collapse>
		</el-form>
	</div>
	<div v-show="false" style="width:95%;border:1px solid #F3F3F3;overflow:hidden" class="inputItem basicConfig">
		<el-form :disabled="viewDisabled" style="width:120%;" ref="cellForm" :model="basicForm" label-position="top">
			<el-collapse v-model="activeNames">
				<!-- Basic -->
				<el-collapse-item name="cell2">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">Cell2</span>
						</p>
					</template>
					<div style="margin-left:65px;">
						<el-form-item label="<%=rb.getString("ZhiChiPinDuan")%>" prop="bands_support_auxiliar">
							<el-input v-model="cellForm.bands_support_auxiliar"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("DaiKuan")%>" prop="band_width_auxiliar" v-show='shownbi'>
							<el-select v-model="cellForm.band_width_auxiliar">
								<el-option label="5MHz" value="n25"></el-option>
								<el-option label="10MHz" value="n50"></el-option>
								<el-option label="15MHz" value="n75"></el-option>
								<el-option label="20MHz" value="n100"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item :label="frequencyLabel" prop="frequency_auxiliar">
							<el-input v-model="basicForm.frequency_auxiliar" @blur="changeToFrequency('dl')" @focus="changeToEarfcn('dl')" disabled></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("ZiZhenPeiBi")%>" prop="subframe_assignment_auxiliar">
							<el-select v-model="cellForm.subframe_assignment_auxiliar">
								<el-option label="<%=rb.getString("QingXuanZe")%>" value=""></el-option>
								<el-option label="0(DL:UL = 1:3)" value="0"></el-option>
								<el-option label="1(DL:UL = 2:2)" value="1"></el-option>
								<el-option label="2(DL:UL = 3:1)" value="2"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("TeShuZiZhenPeiBi")%>" prop="special_subframe_patterns_auxiliar">
							<el-select v-model="cellForm.special_subframe_patterns_auxiliar">
								<el-option label="<%=rb.getString("QingXuanZe")%>" value=""></el-option>
								<el-option label="5" value="5"></el-option>
								<el-option label="7" value="7"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("PLMN")%>" prop="plmn_id_auxiliar">
							<el-input v-model="cellForm.plmn_id_auxiliar"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("TAC")%>" prop="tac_auxiliar">
							<el-input v-model="cellForm.tac_auxiliar"></el-input>
						</el-form-item>
						<el-form-item label="ECI (ECI=eNB_ID*256+Cell_ID)" prop="cell_identity_auxiliar">
							<el-input v-model="cellForm.cell_identity_auxiliar"></el-input>
						</el-form-item>
						<el-form-item label="PCI" prop="phycellid_auxiliar">
							<el-input v-model="cellForm.phycellid_auxiliar"></el-input>
						</el-form-item>
						<el-form-item prop="root_sequence_index_auxiliar" label="<%=rb.getString("GenXuLieSuoYin")%>">
							<el-input v-model="cellForm.root_sequence_index_auxiliar"></el-input>
						</el-form-item>
					</div>
				</el-collapse-item>
			</el-collapse>
		</el-form>
	</div>
	<div style="width:95%;border:1px solid #F3F3F3;margin-top:10px;" v-if="showSetting">
		<el-form ref="settingForm" :disabled="viewDisabled" :model="settingForm" :rules="settingRules" label-position="top">
			<el-collapse>
				<!-- setting -->
				<el-collapse-item>
					<template slot='title'>
						<el-form-item style="display:inline-block;margin-left:40px;margin-top:12px;" prop="ipsec_switch">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold;margin-right:10px;"><%=rb.getString("SheZhi")%></span>
							<el-checkbox v-model="settingForm.ipsec_switch" label=" "></el-checkbox>
						</el-form-item>
					</template>
					<div style="margin-left:65px;">
						<el-form-item label="<%=rb.getString("IpsecKaiGuan")%>" style="margin-right:200px;" prop="ipsec_enable">
							<div style="width:300px;height:25px;padding-top:15px;border:1px solid #DEDFE6">
								<el-radio-group v-model="settingForm.ipsec_enable">
									<el-radio style="margin-left:20px;margin-right:70px;" label="1"><%=rb.getString("QiYong")%></el-radio>
									<el-radio label="0"><%=rb.getString("JinYong")%></el-radio>
								</el-radio-group>
							</div>
						</el-form-item>
						<el-form-item v-if="showItem" label="<%=rb.getString("IKEXieShangMuDiDuanKou")%>" prop="ipsec_rightikeport" class="inputItem" style='margin-right:200px;'>
							<el-select v-model="settingForm.ipsec_rightikeport">
								<el-option label="500" value="500"></el-option>
								<el-option label="4500" value="4500"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item v-if="showItem" label="Left Interface" class="inputItem" prop="left_interface">
							<el-select v-model="settingForm.left_interface">
								<el-option label="none" value="none"></el-option>
								<el-option label="WAN(eth2)" value="WAN"></el-option>
								<el-option label="PPPOE(pppoe-wan)" value="PPPOE"></el-option>
							</el-select>
						</el-form-item>
						<div>
							<div style="display:flex;width:95%">
								<label style="flex:1 1 auto;line-height:30px;">Ipsec Tunnel Table</label>
								<span v-show="viewFlag" @click="addIpsec" class='el-icon el-icon-circle-add' style='font-size:24px;margin-bottom:5px'></span>
							</div>
							<div style="height:300px;max-height:300px;width:95%;border:1px solid #F3F3F3">
								<el-ctable ref="ctableIpsec" :data="settingForm.ipsecList" :pagination="false">
									<el-table-column label="" width="30" prop="" class-name="no-text-tips">
										<template slot-scope="scope">
											<div class="el-icon el-icon-operation-more" @click="optClickIpsec(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
										</template>
									</el-table-column>
									<el-table-column label="Tunnel ID" prop="IPSEC_INDEX"></el-table-column>
									<el-table-column label="Tunnel Enable" prop="TUNNEL_ENABLE" :formatter="tunnelEnableFmt"></el-table-column>
									<el-table-column label="Tunnel Name" prop="TUNNEL_NAME"></el-table-column>
									<el-table-column label="Tunnel Gateway" prop="TUNNEL_GATEWAY"></el-table-column>
								</el-ctable>
								<el-cmenu ref="menu_ipsec" :data="menus_ipsec" @click="clickMenuIpsec"></el-cmenu>
							</div>
							<el-form-item prop="ipsecLength" style="margin-bottom:0px;width:300px;" class="inputItem">
								<el-input v-model="settingForm.ipsecLength" v-show=false></el-input>
							</el-form-item>
						</div>
					</div>
					
				</el-collapse-item>
			</el-collapse>
		</el-form>
	</div>
	<div style="width:95%;border:1px solid #F3F3F3;margin-top:10px;overflow:hidden" class="inputItem basicConfig" v-if="show1588 && false">
		<el-form style="width:120%;" :disabled="viewDisabled" ref="serverForm" :model="serverForm" :rules="serverRules" label-position="top">
			<el-collapse>
				<!-- 1588 -->
				<el-collapse-item>
					<template slot='title'>
						<el-form-item style="display:inline-block;margin-left:40px;margin-top:12px;margin-right:0px;" prop="server_enable">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold;margin-right:10px;">1588</span>
							<el-checkbox v-model="serverForm.server_enable" label=" "></el-checkbox>
						</el-form-item>
					</template>
					<div style="margin-left:65px;">
						<el-form-item label="<%=rb.getString("1588TongBuFangShi")%>" prop="sync_type">
							<el-select v-model="serverForm.sync_type" @change="changeSyncType">
								<el-option label="PTP" value="PTP"></el-option>
								<el-option label="GNSS" value="GNSS"></el-option>
								<el-option label="NL" value="NL"></el-option>
								<el-option label="FREE_RUNNING" value="FREE_RUNNING"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("1588TongBuMoShi")%>" prop="sync_mode">
							<el-select v-model="serverForm.sync_mode">
								<el-option label="TIME" value="TIME"></el-option>
								<el-option label="FREQ" value="FREQ"></el-option>
								<el-option label="PHASE" value="PHASE"></el-option>
							</el-select>
						</el-form-item>
						<div v-if="showPtp">
							<el-form-item label="<%=rb.getString("1588MoShi")%>" prop="mode">
								<el-select v-model="serverForm.mode">
									<el-option label="MODE1" value="MODE1"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="<%=rb.getString("1588JieKou")%>" style="vertical-align:top" prop="ether_interface">
								<el-input disabled="true" v-model="serverForm.ether_interface"></el-input>
								<p>warn:PTP can only be config WAN1</p>
							</el-form-item>
							<el-form-item v-if="showAddress" label="<%=rb.getString("1588DanBoFuWuQiDiZhi")%>" prop="unicast_address">
								<el-input v-model="serverForm.unicast_address"></el-input>
							</el-form-item>
							<el-form-item label="<%=rb.getString("1588DanBoZuBoQieHuan")%>" prop="mode_switch">
								<el-select v-model="serverForm.mode_switch" @change="changeModeSwitch">
									<el-option label="multicast" value="multicast"></el-option>
									<el-option label="unicast" value="unicast"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="<%=rb.getString("1588YuHao")%>" prop="domain">
								<el-input v-model="serverForm.domain"></el-input>
							</el-form-item>
							<el-form-item label="<%=rb.getString("1588TongBuXiaoXiJianGe")%>" prop="sync_interval">
								<el-input v-model="serverForm.sync_interval"></el-input>
							</el-form-item>
							<el-form-item label="<%=rb.getString("1588YanChiXiaoXiJianGe")%>" prop="delay_interval">
								<el-input v-model="serverForm.delay_interval"></el-input>
							</el-form-item>
							<el-form-item label="<%=rb.getString("1588FeiDuiChenShiYan")%>" prop="asymmetry">
								<el-input v-model="serverForm.asymmetry"></el-input>
							</el-form-item>
							<el-form-item label="<%=rb.getString("1588QiDongShiJian")%>" prop="startup_time">
								<el-input v-model="serverForm.startup_time"></el-input>
							</el-form-item>
						</div>
					</div>
				</el-collapse-item>
			</el-collapse>
		</el-form>
	</div>
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition" :force-position="true"
	 :height="slideHeight" :modal='slideModal'  :width="slideWidth" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  @cancel='cancelSlide' @ok='saveSlide'>
	 	
	 </el-slide>
</div>
<script>
	var configRuleVue = new Vue({
		el:"#editConfigRuleDiv",
		data(){
			var vm = this;
			var validateInt = (rule,value,callback) => {
				if(value != "" && value != null){
					var showFlag = value.indexOf("(");
					if(showFlag == -1){
						
					}else{
						value = value.substring(0,showFlag);
					}
				}
				var minVal = rule.min;
				var maxVal = rule.max;
				var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
				var reg = /^-?\d+$/;
				if(value == ""){
					callback(new Error(msgStr));
				}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
					callback();
				}else{
					callback(new Error(msgStr));
				}
			}
			var validatePlmn = (rule,value,callback) => {
				var minVal = rule.min;
				var maxVal = rule.max;
				var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
				var reg = /^-?\d+$/;
				if(value == ""){
					callback(new Error(msgStr));
				}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal)) && value.length<=6){
					callback();
				}else{
					callback(new Error(msgStr));
				}
			}
			var validateRange = (rule,value,callback) => {
				if(vm.platform == 'NBIOT' && rule.type == 'rootIndex'){
					callback();
				}else{
					var reg = /^(\d+\.\.){0,1}(\d+)$/;
					var minVal = rule.min;
					var maxVal = rule.max;
					var msgStr = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
					var showFlag = false;
					if(value == ""){
						showFlag = true;
					}else{
						var showFlag = false;
						if (!reg.test(value)) {
							showFlag = true;
						}else{
							if(value.indexOf("..")>-1){
								value= value.split("..");
								if(value[1]-value[0]<=0){
									showFlag = true;
								}
								if(minVal-value[0]>0){
									showFlag = true;
								}
								if(maxVal-value[1]<0){
									showFlag = true;
								}
							}else{
								if(minVal-value>0){
									showFlag = true;
								}
								if(maxVal-value<0){
									showFlag = true;
								}
							}
						}
					}
					if(showFlag){
						callback(new Error(msgStr));
					}else{
						callback();
					}
				}
			}
			var validateIp = (rule,value,callback) => {
				var regIpv4 = /^((25[0-5]|2[0-4]\d|[0-1]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[0-1]?\d\d?)$/;
				var regIpv6 = /^\s*((([0-9A-Fa-f]{1,4}:){7}(([0-9A-Fa-f]{1,4})|:))|(([0-9A-Fa-f]{1,4}:){6}(:|((25[0-5]|2[0-4]\d|[01]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[01]?\d{1,2})){3})|(:[0-9A-Fa-f]{1,4})))|(([0-9A-Fa-f]{1,4}:){5}((:((25[0-5]|2[0-4]\d|[01]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[01]?\d{1,2})){3})?)|((:[0-9A-Fa-f]{1,4}){1,2})))|(([0-9A-Fa-f]{1,4}:){4}(:[0-9A-Fa-f]{1,4}){0,1}((:((25[0-5]|2[0-4]\d|[01]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[01]?\d{1,2})){3})?)|((:[0-9A-Fa-f]{1,4}){1,2})))|(([0-9A-Fa-f]{1,4}:){3}(:[0-9A-Fa-f]{1,4}){0,2}((:((25[0-5]|2[0-4]\d|[01]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[01]?\d{1,2})){3})?)|((:[0-9A-Fa-f]{1,4}){1,2})))|(([0-9A-Fa-f]{1,4}:){2}(:[0-9A-Fa-f]{1,4}){0,3}((:((25[0-5]|2[0-4]\d|[01]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[01]?\d{1,2})){3})?)|((:[0-9A-Fa-f]{1,4}){1,2})))|(([0-9A-Fa-f]{1,4}:)(:[0-9A-Fa-f]{1,4}){0,4}((:((25[0-5]|2[0-4]\d|[01]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[01]?\d{1,2})){3})?)|((:[0-9A-Fa-f]{1,4}){1,2})))|(:(:[0-9A-Fa-f]{1,4}){0,5}((:((25[0-5]|2[0-4]\d|[01]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[01]?\d{1,2})){3})?)|((:[0-9A-Fa-f]{1,4}){1,2})))|(((25[0-5]|2[0-4]\d|[01]?\d{1,2})(\.(25[0-5]|2[0-4]\d|[01]?\d{1,2})){3})))(%.+)?\s*$/
				if(value == '' || !(regIpv4.test(value) || regIpv6.test(value))){
					callback(new Error("<%=rb.getString("CuoWuGeShi")%>"));
				}else{
					callback();
				}
			}
			var validateNumber = (rule,value,callback) => {
				var minVal = rule.min;
				var maxVal = rule.max;
				var reg = /^-?\d+$/;
				if(value == '' || !reg.test(value) || value-minVal < 0 || value-maxVal > 0 ){
					callback(new Error("<%=rb.getString("FanWei")%> ：" + minVal + "-" + maxVal + " <%=rb.getString("ZhengXing")%>"))
				}else{
					callback();
				}
			}
			var validateIpsec = (rule,value,callback) => {
				if(vm.settingForm.ipsec_switch){
					if(vm.settingForm.ipsecList.length == 0){
						callback(new Error("IPSec Tunnel configure at least one"))
					}else{
						callback();
					}
				}else{
					callback();
				}
			}
			var validateIpsecEnable = (rule,value,callback) => {
				if(vm.settingForm.ipsec_switch){
					if(vm.settingForm.ipsec_enable == "" || vm.settingForm.ipsec_enable == null){
						callback(new Error("<%=rb.getString("QingXuanZe")%>"))
					}else{
						callback();
					}
				}else{
					callback();
				}
			}
			var validateRightikeport = (rule,value,callback) => {
				if(vm.settingForm.ipsec_switch){
					if(vm.settingForm.ipsec_rightikeport == "" || vm.settingForm.ipsec_rightikeport == null){
						callback(new Error("<%=rb.getString("QingXuanZe")%>"))
					}else{
						callback();
					}
				}else{
					callback();
				}
			}
			var validateInterface = (rule,value,callback) => {
				if(vm.settingForm.ipsec_switch){
					if(vm.settingForm.left_interface == "" || vm.settingForm.left_interface == null){
						callback(new Error("<%=rb.getString("QingXuanZe")%>"))
					}else{
						callback();
					}
				}else{
					callback();
				}
			}
			var validate1588Type = (rule,value,callback) => {
				if(vm.serverForm.server_enable){
					if(vm.serverForm.sync_type == "" || vm.serverForm.sync_type == null){
						callback(new Error("<%=rb.getString("QingXuanZe")%>"))
					}else{
						callback();
					}
				}
			}
			var validateSyncMode = (rule,value,callback) => {
				if(vm.serverForm.server_enable){
					if(vm.serverForm.sync_mode == "" || vm.serverForm.sync_mode == null){
						callback(new Error("<%=rb.getString("QingXuanZe")%>"))
					}else{
						callback();
					}
				}
			}
			var validate1588Mode = (rule,value,callback) => {
				if(vm.serverForm.server_enable){
					if(vm.serverForm.mode == "" || vm.serverForm.mode == null){
						callback(new Error("<%=rb.getString("QingXuanZe")%>"))
					}else{
						callback();
					}
				}
			}
			var validateModeSwitch = (rule,value,callback) => {
				if(vm.serverForm.server_enable){
					if(vm.serverForm.mode_switch == "" || vm.serverForm.mode_switch == null){
						callback(new Error("<%=rb.getString("QingXuanZe")%>"))
					}else{
						callback();
					}
				}
			}
			return{
				activeNames:['basic'],
				basicForm:{
					bands_support:"",
					band_width:"",
					frequency:"",
					ul_frequency:"",
					subframe_assignment:"",
					special_subframe_patterns:"",
					plmn_id:"",
					tac:"",
					cell_identity:"",
					phycellid:"",
					root_sequence_index:"",
					mme:"",
					mmeGroup:[],
					mmeStr:"",
					halob_enable:"",
					carrier_mode:"",
					carrierAggregationEnable: false
				},
				basicRules:{
					bands_support:[
						{validator:validateInt,min:1,max:62,trigger:'blur'}
					],
					frequency:[
						{validator:validateInt,min:0,max:65535,trigger:'blur'}
					],
					ul_frequency:[
						{validator:validateInt,min:0,max:65535,trigger:'blur'}
					],
					plmn_id:[
						{validator:validatePlmn,min:00000,max:999999,trigger:'blur'}
					],
					tac:[
						{validator:validateRange,min:0,max:65535,trigger:'blur'}
					],
					cell_identity:[
						{validator:validateRange,min:0,max:268435455,trigger:'blur'}
					],
					phycellid:[
						{validator:validateRange,min:0,max:503,trigger:'blur'}
					],
					root_sequence_index:[
						{validator:validateRange,min:0,max:837,trigger:'blur',type:'rootIndex'}
					]
				},
				cellForm: {
					bands_support_auxiliar:"",
					band_width_auxiliar:"",
					frequency_auxiliar:"",
					subframe_assignment_auxiliar:"",
					special_subframe_patterns_auxiliar:"",
					plmn_id_auxiliar:"",
					tac_auxiliar:"",
					cell_identity_auxiliar:"",
					phycellid_auxiliar:"",
					root_sequence_index_auxiliar:""
				},
				errorMsg:"",
				settingForm:{
					ipsec_switch:false,
					ipsec_enable:"1",
					ipsec_rightikeport:"",
					left_interface:"",
					ipsecList:[],
					defaultIpsecList:[],
					ipsecLength:""
				},
				settingRules:{
					ipsecLength:[
						{validator:validateIpsec}
					],
					ipsec_enable:[
						{validator:validateIpsecEnable}
					],
					ipsec_rightikeport:[
						{validator:validateRightikeport}
					],
					left_interface:[
						{validator:validateInterface}
					],
				},
				serverForm:{
					server_enable:false,
					sync_type:"",
					sync_mode:"",
					mode:"",
					ether_interface:"WAN1",
					unicast_address:"",
					mode_switch:"",
					domain:"",
					sync_interval:"",
					delay_interval:"",
					asymmetry:"",
					startup_time:""
				},
				serverRules:{
					sync_type:[
						{validator:validate1588Type,trigger:'change'}
					],
					sync_mode:[
						{validator:validateSyncMode,trigger:'change'}
					],
					mode:[
						{validator:validate1588Mode,trigger:'change'}
					],
					mode_switch:[
						{validator:validateModeSwitch,trigger:'change'}
					],
					unicast_address:[
						{validator:validateIp,trigger:'blur'}
					],
					domain:[
						{validator:validateNumber,min:0,max:255,trigger:'blur'}
					],
					sync_interval:[
						{validator:validateNumber,min:-7,max:0,trigger:'blur'}
					],
					delay_interval:[
						{validator:validateNumber,min:-7,max:0,trigger:'blur'}
					],
					asymmetry:[
						{validator:validateNumber,min:-65535,max:65535,trigger:'blur'}
					],
					startup_time:[
						{validator:validateNumber,min:0,max:500,trigger:'blur'}
					]
				},
				ipsecData:[],
				paramList:[],
				platform:"",
				slideUrl:"",
				slideTitle:"",
				slideFooter:"",
				slideHeader:"",
				slidePosition:"",
				slideHeight:"",
				slideWidth:"",
				slideModal:"",
				sasDisabled:false,
				showSetting:false,
				show1588:false,
				showBasic:false,
				showMME:false,
				showMode:false,
				showItem:false,
				showPtp:false,
				showAddress:false,
				operTypeIpsec:"",
				menus_ipsec:[],
				rowDataIpsec:[],
				viewDisabled:false,
				viewFlag:true,
				showMMEOper:true,
				shownbi:true,
				frequencyLabel:'<%=rb.getString("PinDian")%>'
			}
		},
		methods:{
			init(){
				var vm = this;
				vm.platform = selfVue.rowDataRule.platform;
				if(vm.platform == "RTD"){
					vm.showItem = true;
					vm.showBasic = true;
					vm.showMode = true;
				}
				if(vm.platform == "RTS"){
					vm.showItem = true;
					vm.showBasic = true;
				}
				if(vm.platform == "QAFA"){
					vm.showMME = true;
					vm.showSetting = true;
					vm.show1588 = true;
				}
				if(vm.platform == "QAFB"){
					vm.showBasic = true;
					vm.showSetting = true;
				}
				if(vm.platform == "QATA"){
					vm.showBasic = true;
					vm.showSetting = true;
					vm.show1588 = true;
					vm.showMME = true;
				}
				if(vm.platform == "NBIOT"){
					vm.shownbi = false;
					vm.showMME = true;
					vm.frequencyLabel = 'DL Frequency'
				}
				var params = {
						platform : selfVue.rowDataRule.platform
				}
				axios.post("${ctx}/SON/SelfConfiguration/getSelfConfigRulesByProduct.action",stringify(params)).then(function(response){
					var data = response.data;
					vm.paramList = data.paramList;
					//如果SAS开关打开，则该参数不可配置
			    	if(SASEnble == "1"){
			    		vm.sasDisabled = true;
			    	}
			    	if(data == "" || data == null){
						
					}else{
						var keys = [
								'bands_support',
								'band_width',
								'frequency',
								'subframe_assignment',
								'special_subframe',
								'plmn_id',
								'tac',
								'cell_identity',
								'phycellid',
								'root_sequence_index'
							];
						
						keys.map(function(code){
							vm.basicForm[code] = data[code];
							vm.cellForm[code+'_auxiliar'] = data[code];
						});
						
						if(data.frequency != null){
							if(translateToFre(data.frequency) == false){
								
							}else{
								vm.basicForm.frequency = translateToFre(data.frequency)
							}
						}
						if(vm.platform == "NBIOT" && data.ul_frequency != null){
							if(translateToFre(data.ul_frequency) == false){
								
							}else{
								vm.basicForm.ul_frequency = translateToFre(data.ul_frequency)
							}
						}
						if(data.subframe_assignment != null){
							vm.basicForm.subframe_assignment = data.subframe_assignment;
						}
						if(data.special_subframe_patterns != null){
							vm.basicForm.special_subframe_patterns = data.special_subframe_patterns;
						}
						
						if(data.mme_ip == "" || data.mme_ip == null){
							
						}else{
							vm.basicForm.mmeGroup = data.mme_ip.split(",")
						}
						vm.basicForm.mmeStr = data.mme_ip;
						vm.basicForm.carrier_mode = data.carrier_mode;
						vm.basicForm.halob_enable = data.halob_enable;
						
						
						vm.settingForm.ipsec_switch = data["ipsec_switch"]==0?false:true;
						vm.settingForm.ipsec_enable = data.ipsec_enable;
						vm.settingForm.ipsec_rightikeport = data.ipsec_rightikeport;
						vm.settingForm.left_interface = data.left_interface;
						vm.settingForm.ipsecList = data.ipsecList;
						if(data.ipsecList.length !=0){
							data.ipsecList.map(function(item){
								vm.settingForm.defaultIpsecList.push(item)
							})
						}
						
						vm.serverForm.server_enable = data["1588_enable"]==0?false:true;
						vm.serverForm.sync_type = data.sync_type;
						vm.serverForm.sync_mode = data.sync_mode;
						vm.serverForm.mode = data.mode;
						vm.serverForm.unicast_address = data.unicast_address;
						vm.serverForm.mode_switch = data.mode_switch;
						vm.serverForm.domain = data.domain;
						vm.serverForm.sync_interval = data.sync_interval;
						vm.serverForm.delay_interval = data.delay_interval;
						vm.serverForm.asymmetry = data.asymmetry;
						vm.serverForm.startup_time = data.startup_time;
						
						if(data.halob_enable == "1"){//打开
							vm.showMME = false;
							vm.showSetting = false;
						}else{//关闭
							vm.showMME = true;
							vm.showSetting = true;
						}
						if(vm.platform == "NBIOT"){
							vm.showSetting = false;
						}
						if(vm.platform == "QAFA" || vm.platform == "QATA"){
							if(data.sync_type == "PTP"){
								vm.showPtp = true;
							}else{
								vm.showPtp = false;
							}
						}
						if(selfVue.operTypeRule == 'view'){
							vm.viewDisabled = true;
							vm.viewFlag = false;
							vm.showMMEOper = false;
						}
						setTimeout(function(){
							initForm(vm.$refs.basicForm);
							if(vm.showSetting){
								initForm(vm.$refs.settingForm);
							}
						},500)
					}
				})
			},
			changeToFrequency(type){
				if(type == 'dl'){
					var value = this.basicForm.frequency;
					if(value == "");
					else this.basicForm.frequency =  translateToFre(this.basicForm.frequency);
				}else if(type == 'ul'){
					var value = this.basicForm.ul_frequency;
					if(value == "");
					else this.basicForm.ul_frequency =  translateToFre(this.basicForm.ul_frequency);
				}
			},
			changeToEarfcn(type){
				if(type == 'dl'){
					var value = this.basicForm.frequency;
					var showFlag = value.indexOf("(");
					if(showFlag == -1);
					else this.basicForm.frequency = this.basicForm.frequency.substring(0,showFlag);
				}else if(type == 'ul'){
					var value = this.basicForm.ul_frequency;
					var showFlag = value.indexOf("(");
					if(showFlag == -1);
					else this.basicForm.ul_frequency = this.basicForm.ul_frequency.substring(0,showFlag);
				}
			},
			addIpsec(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEditRule.action',
				vm.slideTitle = '<%=rb.getString("TianJia")%> Ipsec Tunnel';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '60%';
				vm.operTypeIpsec = 'add';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			saveSlide(){
				eventBus.$emit("save-ipsec");
			},
			cancelSlide(){
				this.$refs.slide.hide();
			},
			addMME(){
				var vm = this,
				msg = "<%=rb.getString("IPDiZhi")%>",
				value = vm.basicForm.mme;
				if(isValidIP(value)){
					vm.basicForm.mmeGroup.push(value);
					vm.basicForm.mme = '';
					vm.errorMsg = '';
				}else{
					vm.errorMsg = msg;
				}
			},
			removeMME(item){
				var vm = this;
				var index = vm.basicForm.mmeGroup.indexOf(item);
				if(index !== -1){
					vm.basicForm.mmeGroup.splice(index,1)
				}
				vm.errorMsg = '';
			},
			changeHalob(val){
				var vm = this;
				if(val == "1"){
					this.showSetting = false;
					this.showMME = false;
				}else{
					this.showSetting = true;
					setTimeout(function(){
						initForm(vm.$refs.settingForm);
					},10)
					this.showMME = true;
				}
			},
			changeSyncType(val){
				if(val == "PTP"){
					this.showPtp = true;
				}else{
					this.showPtp = false;
				}
			},
			changeModeSwitch(val){
				if(val == "unicast"){
					this.showAddress = true;
				}else{
					this.showAddress = false;
				}
			},
			handerClose(){
				this.$refs.menu_ipsec.hide();
			},
			optClickIpsec(row,ev){
				var vm = this;
				vm.rowDataIpsec = row;
				var showView = false;
				var showEdit = false;
				if(selfVue.operTypeRule == "view"){
					showView = true;
				}else if(selfVue.operTypeRule == "edit"){
					showEdit = true
				}
				vm.menus_ipsec = [
					{label:"<%=rb.getString("ChaKan")%>",cls:"el-icon el-icon-operation-info",code:"info",show:showView},
					{label:"<%=rb.getString("XiuGai")%>",cls:"el-icon el-icon-operation-edit",code:"edit",show:showEdit},
					{label:"<%=rb.getString("ShanChu")%>",cls:"el-icon el-icon-operation-delete",code:"del",show:showEdit},
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_ipsec.show(ev);
				})
			},
			clickMenuIpsec(ev){
				var codes = {
						info:this.viewIpsec,
						edit:this.editIpsec,
						del:this.delIpsec
				}
				if(codes[ev.code]){
					codes[ev.code]();
				}
			},
			tunnelEnableFmt(row,column,value,index){
				if(value == "1"){
					return "enable";
				}else{
					return "disable";
				}
			},
			viewIpsec(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEditRule.action',
				vm.slideTitle = '<%=rb.getString("ChaKan")%> Ipsec Tunnel';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '60%';
				vm.operTypeIpsec = 'view';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-ipsec");
	    	    });
			},
			editIpsec(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEditRule.action',
				vm.slideTitle = '<%=rb.getString("XiuGai")%> Ipsec Tunnel';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '60%';
				vm.operTypeIpsec = 'edit';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-ipsec");
	    	    });
			},
			delIpsec(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					var ipsecArr = vm.settingForm.ipsecList.map(function(item){
						return item.IPSEC_INDEX;
					})
					var index = ipsecArr.indexOf(vm.rowDataIpsec.IPSEC_INDEX);
					vm.settingForm.ipsecList.splice(index,1);
					var length = vm.settingForm.ipsecList.length;
					for(let i=0;i<length;i++){
						vm.settingForm.ipsecList[i].IPSEC_INDEX = i+1;
					}
				}).catch(() => {
					
				})
			},
			submit(){
				var vm = this;
				var validFlag = true;
				var params = {};
				params.platform = selfVue.rowDataRule.platform;
				vm.$refs.basicForm.validate(function(valid){
					if(valid){
						params.bands_support = vm.basicForm.bands_support;
						params.band_width = vm.basicForm.band_width;
						params.frequency = vm.basicForm.frequency.substring(0,vm.basicForm.frequency.indexOf("("));
						params.subframe_assignment = vm.basicForm.subframe_assignment;
						params.special_subframe_patterns = vm.basicForm.special_subframe_patterns;
						params.plmn_id = vm.basicForm.plmn_id;
						params.tac = vm.basicForm.tac;
						params.cell_identity = vm.basicForm.cell_identity;
						params.phycellid = vm.basicForm.phycellid;
						params.root_sequence_index = vm.basicForm.root_sequence_index;
						params.mme_ip = vm.basicForm.mmeGroup.toString();
						params.carrier_mode = vm.basicForm.carrier_mode;
						params.halob_enable = vm.basicForm.halob_enable;
						params.carrierAggregationEnable = vm.basicForm.carrierAggregationEnable;
						if(vm.platform == "NBIOT"){
							params.ul_frequency = vm.basicForm.ul_frequency.substring(0,vm.basicForm.ul_frequency.indexOf("("));
						}
						
						//Object.assign(params, vm.cellForm);
					}else{
						validFlag = false
					}
				})
				if(vm.showSetting){
					params.ipsec_switch = vm.settingForm.ipsec_switch?1:0;
					params.ipsec_enable = vm.settingForm.ipsec_enable;
					params.ipsec_rightikeport = vm.settingForm.ipsec_rightikeport;
					params.left_interface = vm.settingForm.left_interface;
					params.ipsecList =JSON.stringify(vm.settingForm.ipsecList);
					//if(vm.settingForm.ipsec_switch){
						vm.$refs.settingForm.validate(function(valid){
							if(valid){
								
							}else{
								validFlag = false
							}
						})
					//}
				}
				
				if(validFlag){
					if(vm.checkParamChange()){//返回true说明改变
						var paramType = sessionStorage.getItem('paramType'),
							urls = {
								special: '${ctx}/SON/SelfConfiguration/updateSelfConfigPlanning.action',
								auto: '${ctx}/SON/SelfConfiguration/saveSelfConfigRulesInfos.action'
							},
							url = urls[paramType];
					
						axios.post("${ctx}/SON/SelfConfiguration/saveSelfConfigRulesInfos.action",stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message:"<%=rb.getString("ChengGong")%>",
		    						type:'success',
		    					})
                                selfVue.$refs.ctableRule.refresh();
                                selfVue.$refs.slide.hide();
							}
						})
					}else{
						vm.$message("<%=rb.getString("WuCanShuBianHua")%>")
					}
				}
			},
			cancel(){
				var vm = this;
				if(vm.checkParamChange()){//返回true说明变化
					vm.$confirm("<%=rb.getString("QueDingLiKaiDangQianYeMian")%>",'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						selfVue.$refs.slide.hide();
					}).catch(() => {
						
					})
				}else{
					selfVue.$refs.slide.hide();
				}
			},
			checkValueChange(oriArr,curArr){//返回true为没有改变，返回false为改变
				var isFlag=true;
				if(oriArr.length == 0 && curArr.length == 0){
					
				}
				if(oriArr.length != curArr.length){
					isFlag = false;
				}else{
					for(let i=0;i<oriArr.length;i++){
						var oriArrItem = {};
						var curArrItem = {};
						Object.keys(oriArr[i]).sort().map(function(key){
							oriArrItem[key] = oriArr[i][key]
						})
						Object.keys(curArr[i]).sort().map(function(key){
							curArrItem[key] = curArr[i][key]
						})
						for(var oriKey in oriArrItem){
							for(var curKey in curArrItem){
								if(oriKey == curKey && oriArrItem[oriKey] != curArrItem[curKey]){
									isFlag =  false;
								}else{
									
								}
							}
						}
					}
				}
				return isFlag
			},
			checkParamChange(){
				var vm = this;
				var changeFlag = true;
				if(isFormChanged(vm.$refs.basicForm)){
					changeFlag = false;
				}
				if(vm.showSetting){
					if(isFormChanged(vm.$refs.settingForm)){
						changeFlag = false;
					}
					if(vm.checkValueChange(vm.settingForm.defaultIpsecList,vm.settingForm.ipsecList) == false){
						changeFlag = false;
					}
				}
				
				if(changeFlag){//没有变化
					return false;
				}else{
					return true;
				}
			}
		},
		watch:{
			"basicForm.mmeGroup":function(){
				this.basicForm.mmeStr = this.basicForm.mmeGroup.toString();
			},
			/* basicForm: {
				handler: function(form) {
					var vm = this,
						keys = [
							'bands_support',
							'band_width',
							'frequency',
							'subframe_assignment',
							'special_subframe',
							'plmn_id',
							'tac',
							'cell_identity',
							'phycellid',
							'root_sequence_index'
						];
					
					for(var key in form) {
						if(keys.includes(key)) {
							vm.cellForm[key+'_auxiliar'] = form[key];
						}
					}
				},
				deep: true
			} */
		},
		mounted(){
			this.init();
			eventBus.$off("save-rule").$on("save-rule",this.submit);
			eventBus.$off("cancel-rule").$on("cancel-rule",this.cancel)
		}
	})
</script>