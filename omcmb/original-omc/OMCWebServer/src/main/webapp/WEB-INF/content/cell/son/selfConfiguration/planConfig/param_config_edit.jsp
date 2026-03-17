<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
#editConfigPlanDiv .el-collapse-item__header{
	border-bottom:1px solid #fff;
}
#editConfigPlanDiv .el-collapse-item__arrow{
	position:absolute;
	left:20px;
	top:0px;
}
#editConfigPlanDiv .el-collapse-item{
	position:relative;
}
#editConfigPlanDiv .el-icon-arrow-right{
	font-size:16px;
}
#editConfigPlanDiv .el-icon-arrow-right:before{
	content:"\e639";
	color:#BBB;
}
#editConfigPlanDiv .is-active.el-icon-arrow-right:before{
	content:"\e638";
	color:#BBB;
}
#editConfigPlanDiv .el-collapse-item__arrow.is-active{
	transform:rotate(0deg);
}
#editConfigPlanDiv .el-collapse{
	border-top:1px solid #fff;
	border-bottom:1px solid #fff;
}
#editConfigPlanDiv .el-collapse-item__wrap{
	border-bottom:1px solid #fff;
}
.inputItem .el-input{
	width:300px;
}
#editConfigPlanDiv .el-form-item{
	display:inline-block;
	vertical-align:top;
}
.basicConfig .el-form-item{
	margin-right:200px;
}
#editConfigPlanDiv .btn-next .el-icon-arrow-right:before{
	content:"\e794"
}
.item_label{
	font-size:13px;
	font-weight:400;
	color:#5A7B92;
	width:500;
	display:inline-block;
	margin-bottom:22px;
	white-space:nowrap;
	text-overflow:ellipsis;
	overflow:hidden;
}
.el-input.is-disabled .el-input__inner {
	min-height: 25px;
	max-height: 25px;
}
#editConfigPlanDiv .el-collapse-item__header{
	max-width:200px;
}
</style>
<div id="editConfigPlanDiv" style="margin-top:10px;margin-left:20px;">
	<div style="width:95%;border:1px solid #F3F3F3;overflow:hidden" class="inputItem basicConfig">
		<el-form :disabled="viewDiabled" style="width:120%;" ref="basicForm" :model="basicForm" :rules="basicRules" label-position="top">
			<el-collapse v-model="activeNames">
				<!-- Basic -->
				<el-collapse-item name="basic">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold"><%=rb.getString("JiChuPeiZhi")%></span>
						</p>
					</template>
					<div style="margin-left:65px;">
						<el-form-item label="<%=rb.getString("ZhiChiPinDuan")%>" prop="bands_support">
							<el-input v-model="basicForm.bands_support"></el-input>
						</el-form-item>
						<p class='item_label' v-if=false><%=rb.getString("ZhiChiPinDuan")%> : <span style='color:#000'>{{basicForm.bands_support}}</span></p>
						<el-form-item label="<%=rb.getString("DaiKuan")%>" prop="band_width" v-show='shownbi'>
							<el-select v-model="basicForm.band_width" :disabled="sasDisabled">
								<el-option label="5MHz" value="n25"></el-option>
								<el-option label="10MHz" value="n50"></el-option>
								<el-option label="15MHz" value="n75"></el-option>
								<el-option label="20MHz" value="n100"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="UL Frequency" v-show='false'>
							<el-input v-model="basicForm.ul_frequency" @blur="changeToFrequency('ul')" @focus="changeToEarfcn('ul')"></el-input>
						</el-form-item>
						<el-form-item :label="frequencyLabel" prop="frequency">
							<el-input v-model="basicForm.frequency" @blur="changeToFrequency('dl')" @focus="changeToEarfcn('dl')" :disabled="sasDisabled"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("ZiZhenPeiBi")%>" prop="subframe_assignment" v-show='showBasic'>
							<el-select v-model="basicForm.subframe_assignment">
								<el-option label="<%=rb.getString("QingXuanZe")%>" value=""></el-option>
								<el-option label="0(DL:UL = 1:3)" value="0"></el-option>
								<el-option label="1(DL:UL = 2:2)" value="1"></el-option>
								<el-option label="2(DL:UL = 3:1)" value="2"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("TeShuZiZhenPeiBi")%>" prop="special_subframe_patterns" v-show='showBasic'>
							<el-select v-model="basicForm.special_subframe_patterns">
								<el-option label="<%=rb.getString("QingXuanZe")%>" value=""></el-option>
								<el-option label="5" value="5"></el-option>
								<el-option label="7" value="7"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item v-if="!isQAFA" label="<%=rb.getString("PLMN")%>" prop="plmn_id">
							<el-input v-model="basicForm.plmn_id"></el-input>
						</el-form-item>
						
						<el-form-item v-if="isQAFA" label="<%=rb.getString("PLMN")%>" prop="plmn_id" style="position:relative;margin-bottom:0px;">
							<el-input v-model="plmnStr"></el-input>
							<span v-if="showMMEOper" @click="addPLMN" class="el-icon el-icon-plus" style="position:absolute;left:275px;top:5px;"></span>
							<div style="margin-top:5px;width:305px;">
								<div class='suffixItem' v-for='(domain,index) in plmnGroup' style='line-height:16px;display:inline-block'>
									<div class='form-suffix'>
										<span class='text'>{{index+1}} : {{domain}}</span>
										<span v-if="showMMEOper" style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removePLMN(domain)'></span>
									</div>
								</div>
								<p style='color:red;font-size:12px;'>{{errorPLMNMsg}}</p>
							</div>
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
						<el-form-item prop="root_sequence_index" label="<%=rb.getString("GenXuLieSuoYin")%>" v-show='shownbi'>
							<el-input v-model="basicForm.root_sequence_index"></el-input>
						</el-form-item>
						<el-form-item v-if="showMME" label="MME" style="position:relative;margin-bottom:0px;">
							<el-input v-model="basicForm.mme"></el-input>
							<span v-if="showMMEOper" @click="addMME" class="el-icon el-icon-plus" style="position:absolute;left:275px;top:5px;"></span>
							<div style="margin-top:5px;width:305px;">
								<div class='suffixItem' v-for='(domain,index) in basicForm.mmeGroup' style='line-height:16px;display:inline-block'>
									<div class='form-suffix'>
										<span class='text'>{{index+1}} : {{domain}}</span>
										<span v-if="showMMEOper" style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeMME(domain)'></span>
									</div>
								</div>
								<p style='color:red;font-size:12px;'>{{errorMsg}}</p>
							</div>
						</el-form-item>
						<el-form-item prop="mmeStr" style="margin-right:0px;">
							<el-input v-model="basicForm.mmeStr" v-show=false></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("HaloBKaiGuan")%>" prop="halob_enable" v-if='shownbi'>
							<el-select v-model="basicForm.halob_enable" @change="changeHalob">
								<el-option label="<%=rb.getString("KaiQi")%>" value="1"></el-option>
								<el-option label="<%=rb.getString("GuanBi")%>" value="0"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item v-if="showMode" label="<%=rb.getString("ZaiBoLeiXing")%>" prop="carrier_mode">
							<el-select v-model="basicForm.carrier_mode">
								<el-option label="<%=rb.getString("DanZaiBo")%>" value="1"></el-option>
								<el-option label="<%=rb.getString("ShuangZaiBo")%>" value="2"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item prop="timeZoneUtc" label="<%=rb.getString("ShiQuSheZhi")%>" v-if="mapFlag" key="timeZoneUtc">
							<el-select v-model="basicForm.timeZoneUtc" filterable>
								<el-option v-for="item in timeZoneList" :key="item.value" :label="item.label" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item prop="txPower" label="<%=rb.getString("CPETxPower")%>" v-if="mapFlag" key="txPower">
							<el-select v-model="basicForm.txPower" filterable>
								<el-option v-for="item in powerList" :key="item.value" :label="item.label" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
					</div>
				</el-collapse-item>
			</el-collapse>
		</el-form>
	</div>
	<div style="width:95%;border:1px solid #F3F3F3;margin-top:10px;" v-if="showSetting">
		<el-form ref="settingForm" :disabled="viewDiabled" :model="settingForm" :rules="settingRules" label-position="top">
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
									<el-table-column label="<%=rb.getString("TunnelKaiGuan")%>" prop="TUNNEL_ENABLE" :formatter="tunnelEnableFmt"></el-table-column>
									<el-table-column label="<%=rb.getString("TunnelMingCheng")%>" prop="TUNNEL_NAME"></el-table-column>
									<el-table-column label="<%=rb.getString("TunnelWangGuan")%>" prop="TUNNEL_GATEWAY"></el-table-column>
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
	<div style="width:95%;border:1px solid #F3F3F3;margin-top:10px;overflow:hidden" class="inputItem basicConfig" v-if="show1588">
		<el-form style="width:120%;" :disabled="viewDiabled" ref="serverForm" :model="serverForm" :rules="serverRules" label-position="top">
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
	<div style="width:95%;border:1px solid #F3F3F3;margin-top:10px;overflow:hidden" v-if="showVlan">
		<el-form ref="vlanForm" :disabled="viewDiabled" :model="vlanForm" :rules="vlanRules">
			<el-collapse>
				<!-- vlan -->
				<el-collapse-item>
					<template slot='title'>
						<el-form-item style="display:inline-block;margin-left:40px;margin-top:12px;" prop="vlan_enable">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold;margin-right:10px;">Vlan</span>
							<el-checkbox v-model="vlanForm.vlan_enable" label=" "></el-checkbox>
						</el-form-item>
					</template>
					<div style="margin-left:65px;">
						<div style="display:flex;width:95%">
							<label style="flex:1 1 auto;line-height:30px;">VLAN Table</label>
							<span @click="addVlan" v-show="vlanDisabled" class='el-icon el-icon-circle-add' style='font-size:24px;margin-bottom:5px'></span>
						</div>
						<div style="height:300px;max-height:300px;width:95%;border:1px solid #F3F3F3">
							<el-ctable ref="ctableVlan" :data="vlanForm.vlanList" :pagination="false">
								<el-table-column label="" width="30" prop="" class-name="no-text-tips">
									<template slot-scope="scope">
										<div class="el-icon el-icon-operation-more" @click="optClickVlan(scope.row,event)" v-clickoutside="handerCloseVlan" style="cursor: pointer;"></div>
									</template>
								</el-table-column>
								<el-table-column label="VLAN Index" prop="vlan_index"></el-table-column>
								<el-table-column label="VLAN Name" prop="vlan_name"></el-table-column>
								<el-table-column label="VLAN Id" prop="vlan_id"></el-table-column>
								<el-table-column label="Proto" prop="proto"></el-table-column>
								<el-table-column label="Ipaddr" prop="ipaddr"></el-table-column>
								<el-table-column label="Netmask" prop="netmask"></el-table-column>
								<el-table-column label="Gateway" prop="gateway"></el-table-column>
								<el-table-column label="DNS" prop="dns"></el-table-column>
							</el-ctable>
							<el-cmenu ref="menu_vlan" :data="menus_vlan" @click="clickMenuVlan"></el-cmenu>
						</div>
						<el-form-item prop="vlanLength" style="margin-bottom:0px;width:300px;">
							<el-input v-model="vlanForm.vlanLength" v-show=false></el-input>
						</el-form-item>
						<div style="display:flex;width:95%;margin-top:20px;">
							<label style="flex:1 1 auto;line-height:30px;">Route Table</label>
							<span @click="addRoute" v-show="viewFlag" class='el-icon el-icon-circle-add' style='font-size:24px;margin-bottom:5px'></span>
						</div>
						<div style="height:300px;max-height:300px;width:95%;border:1px solid #F3F3F3">
							<el-ctable ref="ctableRoute" :data="vlanForm.routeList" :pagination="false">
								<el-table-column label="" width="30" prop="" class-name="no-text-tips">
									<template slot-scope="scope">
										<div class="el-icon el-icon-operation-more" @click="optClickRoute(scope.row,event)" v-clickoutside="handerCloseRoute" style="cursor: pointer;"></div>
									</template>
								</el-table-column>
								<el-table-column label="Route Index" prop="route_index"></el-table-column>
								<el-table-column label="Target IP" prop="target"></el-table-column>
								<el-table-column label="Gateway" prop="gateway"></el-table-column>
								<el-table-column label="Netmask" prop="netmask"></el-table-column>
							</el-ctable>
							<el-cmenu ref="menu_route" :data="menus_route" @click="clickMenuRoute"></el-cmenu>
						</div>
						<el-form-item prop="routeLength" style="margin-bottom:0px;width:300px;">
							<el-input v-model="vlanForm.routeLength" v-show=false></el-input>
						</el-form-item>
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
	var configPlanVue = new Vue({
		el:"#editConfigPlanDiv",
		data(){
			var vm = this;
			var validateInt = (rule,value,callback) => {
				if(vm.platform != 'NBIOT' && rule.type == 'ulfrequency'){
					callback();
				}else{
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
			}
			var validatePlmn = (rule,value,callback) => {
				var minVal = rule.min;
				var maxVal = rule.max;
				var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
				var reg = /^-?\d+$/;
				
				if(vm.isQAFA) {
					if(value == "") {
						callback(new Error(msgStr));
					}else {
						callback();
					}
				}else {
					if(value == ""){
						callback(new Error(msgStr));
					}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal)) && value.length<=6){
						callback();
					}else{
						callback(new Error(msgStr));
					}
				}
			}
			var validateRange = (rule,value,callback) => {
				if(vm.platform == 'NBIOT' && rule.type == 'rootIndex'){
					callback();
				}else{
					var reg = /^(\d+\.\.){0,1}(\d+)$/;
					var minVal = rule.min;
					var maxVal = rule.max;
					var tipStr = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
					if(rule.type == 'auto'){
						var msgStr = 'AUTO|'+tipStr;
					}else{
						var msgStr = tipStr;
					}
					var showFlag = false;
					if(value == ""){
						showFlag = true;
					}else{
						var showFlag = false;
						if (!(reg.test(value) || value == 'AUTO')) {
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
			var validateVlan = (rule,value,callback) => {
				if(vm.vlanForm.vlan_enable){
					if(vm.vlanForm.vlanList.length == 0){
						callback(new Error("<%=rb.getString("VLANPeiZhiGeShu")%>"))
					}else{
						callback();
					}
				}
			}
			var validateRoute = (rule,value,callback) => {
				if(vm.vlanForm.vlan_enable){
					if(vm.vlanForm.routeList.length == 0){
						callback(new Error("<%=rb.getString("RoutePeiZhiGeShu")%>"))
					}else{
						callback();
					}
				}
			}
			return{
				plmnStr: '',
				plmnGroup: [],
				errorPLMNMsg: '',
				
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
					timeZoneUtc:'',
					txPower:''
				},
				basicRules:{
					bands_support:[
						{validator:validateInt,min:1,max:62,trigger:'blur'}
					],
					frequency:[
						{validator:validateInt,min:0,max:65535,trigger:'blur'}
					],
					plmn_id:[
						{validator:validatePlmn,min:00000,max:999999,trigger:'blur'}
					],
					tac:[
						{validator:validateRange,min:0,max:65535,trigger:'blur',type:'auto'}
					],
					cell_identity:[
						{validator:validateRange,min:0,max:268435455,trigger:'blur',type:'auto'}
					],
					phycellid:[
						{validator:validateRange,min:0,max:503,trigger:'blur'}
					],
					root_sequence_index:[
						{validator:validateRange,min:0,max:837,trigger:'blur',type:'rootIndex'}
					]
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
						{validator:validateNumber,min:0,max:5000,trigger:'blur'}
					]
				},
				vlanForm:{
					vlanList:[],
					routeList:[],
					vlan_enable:false,
					vlanLength:"",
					routeLength:"",
					defaultVlanList:[],
					defaultRouteList:[]
				},
				vlanRules:{
					vlanLength:[
						{validator:validateVlan}
					],
					routeLength:[
						{validator:validateRoute}
					],
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
				showVlan:false,
				showBasic: true,
				showMME:false,
				showMode:false,
				showItem:false,
				showPtp:false,
				showAddress:false,
				operTypeIpsec:"",
				operTypeVlan:"",
				operTypeRoute:"",
				itemType:"",
				menus_ipsec:[],
				rowDataIpsec:[],
				menus_vlan:[],
				rowDataVlan:[],
				menus_route:[],
				rowDataRoute:[],
				vlanDisabled:true,
				viewDiabled:false,
				viewFlag:true,
				showMMEOper:true,
				shownbi:true,
				frequencyLabel:'<%=rb.getString("PinDian")%>',
				timeZoneList:[],
				powerList:[],
				mapFlag:false
			}
		},
		computed: {
			isQAFA() {

				return this.platform == 'qa';
			}
		},
		methods:{
			init(){
				var vm = this;
				vm.platform = selfVue.rowDataPlan.platform;
				var timeStr = "Africa/Abidjan,Africa/Accra,Africa/Addis_Ababa,Africa/Algiers,Africa/Asmara,Africa/Bamako,Africa/Bangui,Africa/Banjul,Africa/Bissau,Africa/Blantyre,Africa/Brazzaville,Africa/Bujumbura,Africa/Cairo,Africa/Casablanca,Africa/Ceuta,Africa/Conakry,Africa/Dakar,Africa/Dar_es_Salaam,Africa/Djibouti,Africa/Douala,Africa/El_Aaiun,Africa/Freetown,Africa/Gaborone,Africa/Harare,Africa/Johannesburg,Africa/Juba,Africa/Kampala,Africa/Khartoum,Africa/Kigali,Africa/Kinshasa,Africa/Lagos,Africa/Libreville,Africa/Lome,Africa/Luanda,Africa/Lubumbashi,Africa/Lusaka,Africa/Malabo,Africa/Maputo,Africa/Maseru,Africa/Mbabane,Africa/Mogadishu,Africa/Monrovia,Africa/Nairobi,Africa/Ndjamena,Africa/Niamey,Africa/Nouakchott,Africa/Ouagadougou,Africa/Porto-Novo,Africa/Sao_Tome,Africa/Tripoli,Africa/Tunis,Africa/Windhoek,America/Adak,America/Anchorage,America/Anguilla,America/Antigua,America/Araguaina,America/Argentina/Buenos_Aires,America/Argentina/Catamarca,America/Argentina/Cordoba,America/Argentina/Jujuy,America/Argentina/La_Rioja,America/Argentina/Mendoza,America/Argentina/Rio_Gallegos,America/Argentina/Salta,America/Argentina/San_Juan,America/Argentina/San_Luis,America/Argentina/Tucuman,America/Argentina/Ushuaia,America/Aruba,America/Asuncion,America/Atikokan,America/Bahia,America/Bahia_Banderas,America/Barbados,America/Belem,America/Belize,America/Blanc-Sablon,America/Boa_Vista,America/Bogota,America/Boise,America/Cambridge_Bay,America/Campo_Grande,America/Cancun,America/Caracas,America/Cayenne,America/Cayman,America/Chicago,America/Chihuahua,America/Costa_Rica,America/Creston,America/Cuiaba,America/Curacao,America/Danmarkshavn,America/Dawson,America/Dawson_Creek,America/Denver,America/Detroit,America/Dominica,America/Edmonton,America/Eirunepe,America/El_Salvador,America/Fortaleza,America/Glace_Bay,America/Godthab,America/Goose_Bay,America/Grand_Turk,America/Grenada,America/Guadeloupe,America/Guatemala,America/Guayaquil,America/Guyana,America/Halifax,America/Havana,America/Hermosillo,America/Indiana/Indianapolis,America/Indiana/Knox,America/Indiana/Marengo,America/Indiana/Petersburg,America/Indiana/Tell_City,America/Indiana/Vevay,America/Indiana/Vincennes,America/Indiana/Winamac,America/Inuvik,America/Iqaluit,America/Jamaica,America/Juneau,America/Kentucky/Louisville,America/Kentucky/Monticello,America/Kralendijk,America/La_Paz,America/Lima,America/Los_Angeles,America/Lower_Princes,America/Maceio,America/Managua,America/Manaus,America/Marigot,America/Martinique,America/Matamoros,America/Mazatlan,America/Menominee,America/Merida,America/Metlakatla,America/Mexico_City,America/Miquelon,America/Moncton,America/Monterrey,America/Montevideo,America/Montserrat,America/Nassau,America/New_York,America/Nipigon,America/Nome,America/Noronha,America/North_Dakota/Beulah,America/North_Dakota/Center,America/North_Dakota/New_Salem,America/Ojinaga,America/Panama,America/Pangnirtung,America/Paramaribo,America/Phoenix,America/Port_of_Spain,America/Port-au-Prince,America/Porto_Velho,America/Puerto_Rico,America/Rainy_River,America/Rankin_Inlet,America/Recife,America/Regina,America/Resolute,America/Rio_Branco,America/Santa_Isabel,America/Santarem,America/Santiago,America/Santo_Domingo,America/Sao_Paulo,America/Scoresbysund,America/Sitka,America/St_Barthelemy,America/St_Johns,America/St_Kitts,America/St_Lucia,America/St_Thomas,America/St_Vincent,America/Swift_Current,America/Tegucigalpa,America/Thule,America/Thunder_Bay,America/Tijuana,America/Toronto,America/Tortola,America/Vancouver,America/Whitehorse,America/Winnipeg,America/Yakutat,America/Yellowknife,Antarctica/Casey,Antarctica/Davis,Antarctica/DumontDUrville,Antarctica/Macquarie,Antarctica/Mawson,Antarctica/McMurdo,Antarctica/Palmer,Antarctica/Rothera,Antarctica/Syowa,Antarctica/Troll,Antarctica/Vostok,Arctic/Longyearbyen,Asia/Aden,Asia/Almaty,Asia/Amman,Asia/Anadyr,Asia/Aqtau,Asia/Aqtobe,Asia/Ashgabat,Asia/Baghdad,Asia/Bahrain,Asia/Baku,Asia/Bangkok,Asia/Beirut,Asia/Bishkek,Asia/Brunei,Asia/Chita,Asia/Choibalsan,Asia/Colombo,Asia/Damascus,Asia/Dhaka,Asia/Dili,Asia/Dubai,Asia/Dushanbe,Asia/Gaza,Asia/Hebron,Asia/Ho_Chi_Minh,Asia/Hong_Kong,Asia/Hovd,Asia/Irkutsk,Asia/Jakarta,Asia/Jayapura,Asia/Jerusalem,Asia/Kabul,Asia/Kamchatka,Asia/Karachi,Asia/Kathmandu,Asia/Khandyga,Asia/Kolkata,Asia/Krasnoyarsk,Asia/Kuala_Lumpur,Asia/Kuching,Asia/Kuwait,Asia/Macau,Asia/Magadan,Asia/Makassar,Asia/Manila,Asia/Muscat,Asia/Nicosia,Asia/Novokuznetsk,Asia/Novosibirsk,Asia/Omsk,Asia/Oral,Asia/Phnom_Penh,Asia/Pontianak,Asia/Pyongyang,Asia/Qatar,Asia/Qyzylorda,Asia/Rangoon,Asia/Riyadh,Asia/Sakhalin,Asia/Samarkand,Asia/Seoul,Asia/Shanghai,Asia/Singapore,Asia/Srednekolymsk,Asia/Taipei,Asia/Tashkent,Asia/Tbilisi,Asia/Thimphu,Asia/Tokyo,Asia/Ulaanbaatar,Asia/Urumqi,Asia/Ust-Nera,Asia/Vientiane,Asia/Vladivostok,Asia/Yakutsk,Asia/Yekaterinburg,Asia/Yerevan,Atlantic/Azores,Atlantic/Bermuda,Atlantic/Canary,Atlantic/Cape_Verde,Atlantic/Faroe,Atlantic/Madeira,Atlantic/Reykjavik,Atlantic/South_Georgia,Atlantic/St_Helena,Atlantic/Stanley,Australia/Adelaide,Australia/Brisbane,Australia/Broken_Hill,Australia/Currie,Australia/Darwin,Australia/Eucla,Australia/Hobart,Australia/Lindeman,Australia/Lord_Howe,Australia/Melbourne,Australia/Perth,Australia/Sydney,Europe/Amsterdam,Europe/Andorra,Europe/Athens,Europe/Belgrade,Europe/Berlin,Europe/Bratislava,Europe/Brussels,Europe/Bucharest,Europe/Budapest,Europe/Busingen,Europe/Chisinau,Europe/Copenhagen,Europe/Dublin,Europe/Gibraltar,Europe/Guernsey,Europe/Helsinki,Europe/Isle_of_Man,Europe/Istanbul,Europe/Jersey,Europe/Kaliningrad,Europe/Kiev,Europe/Lisbon,Europe/Ljubljana,Europe/London,Europe/Luxembourg,Europe/Madrid,Europe/Malta,Europe/Mariehamn,Europe/Minsk,Europe/Monaco,Europe/Moscow,Europe/Oslo,Europe/Paris,Europe/Podgorica,Europe/Prague,Europe/Riga,Europe/Rome,Europe/Samara,Europe/San_Marino,Europe/Sarajevo,Europe/Simferopol,Europe/Skopje,Europe/Sofia,Europe/Stockholm,Europe/Tallinn,Europe/Tirane,Europe/Uzhgorod,Europe/Vaduz,Europe/Vatican,Europe/Vienna,Europe/Vilnius,Europe/Volgograd,Europe/Warsaw,Europe/Zagreb,Europe/Zaporozhye,Europe/Zurich,Indian/Antananarivo,Indian/Chagos,Indian/Christmas,Indian/Cocos,Indian/Comoro,Indian/Kerguelen,Indian/Mahe,Indian/Maldives,Indian/Mauritius,Indian/Mayotte,Indian/Reunion,Pacific/Apia,Pacific/Auckland,Pacific/Bougainville,Pacific/Chatham,Pacific/Chuuk,Pacific/Easter,Pacific/Efate,Pacific/Enderbury,Pacific/Fakaofo,Pacific/Fiji,Pacific/Funafuti,Pacific/Galapagos,Pacific/Gambier,Pacific/Guadalcanal,Pacific/Guam,Pacific/Honolulu,Pacific/Johnston,Pacific/Kiritimati,Pacific/Kosrae,Pacific/Kwajalein,Pacific/Majuro,Pacific/Marquesas,Pacific/Midway,Pacific/Nauru,Pacific/Niue,Pacific/Norfolk,Pacific/Noumea,Pacific/Pago_Pago,Pacific/Palau,Pacific/Pitcairn,Pacific/Pohnpei,Pacific/Port_Moresby,Pacific/Rarotonga,Pacific/Saipan,Pacific/Tahiti,Pacific/Tarawa,Pacific/Tongatapu,Pacific/Wake,Pacific/Wallis";
				var timeList = timeStr.split(",");
				var powerList = ['-20dBm(0.01mW)','-10dBm(0.1mW)','0dBm(1.0mW)','1dBm(1.3mW)','2dBm(1.6mW)','3dBm(2.0mW)','4dBm(2.5mW)','5dBm(3.2mW)','6dBm(4.0mW)','7dBm(5.0mW)','8dBm(6.3mW)','9dBm(7.9mW)','10dBm(10mW)','11dBm(13mW)','12dBm(16mW)','13dBm(20mW)','14dBm(25mW)','15dBm(32mW)','16dBm(40mW)','17dBm(50mW)','18dBm(63mW)','19dBm(79mW)','20dBm(100mW)','21dBm(126mW)','22dBm(158mW)','23dBm(200mW)','24dBm(251mW)','25dBm(316mW)','26dBm(398mW)','27dBm(501mW)'];
				var powerValList = [-20,-10,0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27]
				timeList.map(item=>{
					vm.timeZoneList.push({label:item,value:item})
				})
				powerList.map((item,index)=>{
					vm.powerList.push({label:item,value:powerValList[index]})
				})
				if(vm.platform == "intel"){
					vm.showMode = true;
					vm.showItem = true;
				}
				if(vm.platform == "qa"){
					vm.show1588 = true;
					vm.mapFlag = true;
				}
				if(vm.platform == 'NBIOT'){
					vm.shownbi = false;
					vm.showMME = true;
					vm.showBasic = false;
					vm.frequencyLabel = 'DL Frequency'
				}
				var params = {
						id : selfVue.rowDataPlan.id
				}
				axios.post("${ctx}/SON/SelfConfiguration/getSelfConfigPlanningInfos.action",stringify(params)).then(function(response){
					var data = response.data;
					vm.paramList = data.paramList;
					//如果SAS开关打开，则该参数不可配置
			    	if(SASEnble == "1"){
			    		vm.sasDisabled = true;
			    	}
			    	if(data == "" || data == null){
						
					}else{
						vm.basicForm.bands_support = data.bands_support;
						vm.basicForm.band_width = data.band_width;
						if(data.frequency != null){
							if(translateToFre(data.frequency) == false){
								
							}else{
								vm.basicForm.frequency = translateToFre(data.frequency)
							}
						}
						if(vm.platform == 'NBIOT' && data.ul_frequency != null){
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
						vm.basicForm.plmn_id = data.plmn_id;
						vm.basicForm.tac = data.tac;
						vm.basicForm.cell_identity = data.cell_identity;
						vm.basicForm.phycellid = data.phycellid;
						vm.basicForm.root_sequence_index = data.root_sequence_index;

						if(data.plmn_id){
							vm.plmnGroup = data.plmn_id.split(",")
						}
						
						if(data.mme_ip == "" || data.mme_ip == null){
							
						}else{
							vm.basicForm.mmeGroup = data.mme_ip.split(",")
						}
						vm.basicForm.mmeStr = data.mme_ip;
						vm.basicForm.carrier_mode = data.carrier_mode;
						vm.basicForm.halob_enable = data.halob_enable;
						vm.basicForm.timeZoneUtc = data.timeZoneUtc;
						if(data.txPower == '' || data.txPower == null){
							vm.basicForm.txPower = ''
						}else{
							vm.basicForm.txPower = parseInt(data.txPower);
						}
						vm.settingForm.ipsec_switch = data["ipsec_switch"]==0?false:true;
						vm.settingForm.ipsec_enable = data.ipsec_enable;
						vm.settingForm.ipsec_rightikeport = data.ipsec_rightikeport;
						vm.settingForm.left_interface = data.left_interface;
						vm.settingForm.ipsecList = data.ipsecList;
						if(data.ipsecList && data.ipsecList.length !=0){
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
						
						vm.vlanForm.vlan_enable = data.vlan_enable==0?false:true;
						vm.vlanForm.vlanList = data.vlanList;
						vm.vlanForm.routeList = data.routeList;
						if(data.vlanList && data.vlanList.length !=0){
							data.vlanList.map(function(item){
								vm.vlanForm.defaultVlanList.push(item)
							})
							data.routeList.map(function(item){
								vm.vlanForm.defaultRouteList.push(item)
							})
						}
						
						if(data.halob_enable == "1"){//打开
							//vm.showVlan = false;
							vm.showMME = false;
							vm.showSetting = false;
						}else{//关闭
							//vm.showVlan = true;
							vm.showMME = true;
							vm.showSetting = true;
						}
						if(vm.platform == 'NBIOT'){
							vm.showSetting = false;
						}
						if(vm.platform == "qa"){
							if(data.sync_type == "PTP"){
								vm.showPtp = true;
							}else{
								vm.showPtp = false;
							}
						}
						if(selfVue.operTypePlan == 'view'){
							
							vm.viewDiabled = true;
							vm.viewFlag = false;
							vm.vlanDisabled = false;
							vm.showMMEOper = false;
						}
						setTimeout(function(){
							initForm(vm.$refs.basicForm);
							if(vm.showSetting){
								initForm(vm.$refs.settingForm);
							}
							if(vm.show1588){
								initForm(vm.$refs.serverForm);
							}
							if(vm.showVlan){
								initForm(vm.$refs.vlanForm);
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
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEdit.action',
				vm.slideTitle = '<%=rb.getString("TianJia")%> Ipsec Tunnel';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '60%';
				vm.operTypeIpsec = 'add';
				vm.itemType = 'ipsec';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			saveSlide(){
				if(this.itemType == 'ipsec'){
					eventBus.$emit("save-ipsec");
				}
				if(this.itemType == "vlan"){
					eventBus.$emit("save-vlan");
				}
				if(this.itemType == "route"){
					eventBus.$emit("save-route");
				}
			},
			cancelSlide(){
				this.$refs.slide.hide();
			},
			addVlan(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goVlanOper.action',
				vm.slideTitle = 'Add Vlan';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypeVlan = 'add';
				vm.itemType = 'vlan';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			addRoute(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goRouteOper.action',
				vm.slideTitle = 'Add Route';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypeRoute = 'add';
				vm.itemType = 'route';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},
			addPLMN() {
				var vm = this,
					msg = 'Length: 5-6',
					value = vm.plmnStr;
				
				if(value.length>=5 && value.length<=6 && value - 10000 >=0 && value - 999999 <=0){
					if(vm.plmnGroup.includes(value)) {
						vm.errorPLMNMsg = 'Existed!';
					}else {
						vm.plmnGroup.push(value);
						vm.plmnStr = '';
						vm.errorPLMNMsg = '';
					}
				}else{
					vm.errorPLMNMsg = msg;
				}
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
			removePLMN(item) {
				var vm = this,
					index = vm.plmnGroup.indexOf(item);
				
				if(index !== -1){
					vm.plmnGroup.splice(index,1)
				}
				
				vm.errorPLMNMsg = '';
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
					//this.showVlan = false;
					this.showSetting = false;
					this.showMME = false;
				}else{
					//this.showVlan = true;
					this.showSetting = true;
					setTimeout(function(){
						initForm(vm.$refs.settingForm);
					},10)
					this.showMME = true;
				}
				
				if(vm.platform == 'NBIOT'){
					vm.showSetting = false;
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
				if(selfVue.operTypePlan == "view"){
					showView = true;
				}else if(selfVue.operTypePlan == "edit"){
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
			editIpsec(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEdit.action',
				vm.slideTitle = '<%=rb.getString("XiuGai")%> Ipsec Tunnel';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '60%';
				vm.operTypeIpsec = 'edit';
				vm.itemType = 'ipsec';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-ipsec");
	    	    });
			},
			viewIpsec(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEdit.action',
				vm.slideTitle = '<%=rb.getString("ChaKan")%> Ipsec Tunnel';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '60%';
				vm.operTypeIpsec = 'view';
				vm.itemType = 'ipsec';
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
			handerCloseVlan(){
				this.$refs.menu_vlan.hide();
			},
			optClickVlan(row,ev){
				var vm = this;
				vm.rowDataVlan = row;
				var showView = false;
				var showEdit = false;
				if(selfVue.operTypePlan == "view"){
					showView = true;
				}else if(selfVue.operTypePlan == "edit"){
					showEdit = true
				}
				vm.menus_vlan = [
					{label:"View",cls:"el-icon el-icon-operation-info",code:"info",show:showView},
					{label:"Modify",cls:"el-icon el-icon-operation-edit",code:"edit",show:showEdit},
					{label:"Delete",cls:"el-icon el-icon-operation-delete",code:"del",show:showEdit},
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_vlan.show(ev);
				})
			},
			clickMenuVlan(ev){
				var codes = {
						info:this.viewVlan,
						edit:this.editVlan,
						del:this.delVlan
				}
				if(codes[ev.code]){
					codes[ev.code]();
				}
			},
			viewVlan(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goVlanOper.action',
				vm.slideTitle = 'View Vlan';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypeVlan = 'view';
				vm.itemType = 'vlan';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-vlan");
	    	    });
			},
			editVlan(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goVlanOper.action',
				vm.slideTitle = 'Modify Vlan';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypeVlan = 'edit';
				vm.itemType = 'vlan';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-vlan");
	    	    });
			},
			delVlan(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					var vlanArr = vm.vlanForm.vlanList.map(function(item){
						return item.vlan_index;
					})
					var index = vlanArr.indexOf(vm.rowDataVlan.vlan_index);
					vm.vlanForm.vlanList.splice(index,1);
					var length = vm.vlanForm.vlanList.length;
					for(let i=0;i<length;i++){
						vm.vlanForm.vlanList[i].vlan_index = i+1;
					}
					vm.vlanDisabled = true;
				}).catch(() => {
					
				})
			},
			handerCloseRoute(){
				this.$refs.menu_route.hide();
			},
			optClickRoute(row,ev){
				var vm = this;
				vm.rowDataRoute = row;
				var showView = false;
				var showEdit = false;
				if(selfVue.operTypePlan == "view"){
					showView = true;
				}else if(selfVue.operTypePlan == "edit"){
					showEdit = true
				}
				vm.menus_route = [
					{label:"View",cls:"el-icon el-icon-operation-info",code:"info",show:showView},
					{label:"Modify",cls:"el-icon el-icon-operation-edit",code:"edit",show:showEdit},
					{label:"Delete",cls:"el-icon el-icon-operation-delete",code:"del",show:showEdit},
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_route.show(ev);
				})
			},
			clickMenuRoute(ev){
				var codes = {
						info:this.viewRoute,
						edit:this.editRoute,
						del:this.delRoute
				}
				if(codes[ev.code]){
					codes[ev.code]();
				}
			},
			viewRoute(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goRouteOper.action',
				vm.slideTitle = 'View Route';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypeRoute = 'view';
				vm.itemType = 'route';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    	eventBus.$emit("edit-route");
	    	    });
			},
			editRoute(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goRouteOper.action',
				vm.slideTitle = 'Modify Route';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.operTypeRoute = 'edit';
				vm.itemType = 'route';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    	eventBus.$emit("edit-route");
	    	    });
			},
			delRoute(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					var routeArr = vm.vlanForm.routeList.map(function(item){
						return item.route_index;
					})
					var index = routeArr.indexOf(vm.rowDataRoute.route_index);
					vm.vlanForm.routeList.splice(index,1);
					var length = vm.vlanForm.routeList.length;
					for(let i=0;i<length;i++){
						vm.vlanForm.routeList[i].route_index = i+1;
					}
				}).catch(() => {
					
				})
			},
			submit(){
				var vm = this;
				var validFlag = true;
				var params = {};
				params.id = selfVue.rowDataPlan.id;
				params.platform = selfVue.rowDataPlan.platform;
				params.serial_number = selfVue.rowDataPlan.serial_number;
				params.host_name = selfVue.rowDataPlan.host_name;
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
						params.timeZoneUtc = vm.basicForm.timeZoneUtc;
						params.txPower = vm.basicForm.txPower;
						if(vm.platform == 'NBIOT'){
							params.ul_frequency = vm.basicForm.ul_frequency.substring(0,vm.basicForm.ul_frequency.indexOf("("));
						}
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
							
				if(vm.show1588){
					params["1588_enable"] = vm.serverForm.server_enable?1:0;
					params.sync_type = vm.serverForm.sync_type;
					params.sync_mode = vm.serverForm.sync_mode;
					params.mode = vm.serverForm.mode;
					params.ether_interface = vm.serverForm.ether_interface;
					params.unicast_address = vm.serverForm.unicast_address;
					params.mode_switch = vm.serverForm.mode_switch;
					params.domain = vm.serverForm.domain;
					params.sync_interval = vm.serverForm.sync_interval;
					params.delay_interval = vm.serverForm.delay_interval;
					params.asymmetry = vm.serverForm.asymmetry;
					params.startup_time = vm.serverForm.startup_time;
					if(vm.serverForm.server_enable){
						vm.$refs.serverForm.validate(function(valid){
							if(valid){
								
							}else{
								validFlag = false
							}
						})
					}
				}
					
				if(vm.showVlan){
					params.vlan_enable = vm.vlanForm.vlan_enable?1:0;
					params.vlanList = JSON.stringify(vm.vlanForm.vlanList);
					params.routeList = JSON.stringify(vm.vlanForm.routeList);
					if(vm.vlanForm.vlan_enable){
						vm.$refs.vlanForm.validate(function(valid){
							if(valid){
								
							}else{
								validFlag = false
							}
						})
					}
				}
					
				if(validFlag){
					if(vm.checkParamChange()){//返回true为改变
						axios.post("${ctx}/SON/SelfConfiguration/updateSelfConfigPlanning.action",stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message:"<%=rb.getString("ChengGong")%>",
		    						type:'success',
		    					})
                                selfVue.$refs.ctablePlan.refresh();
                                selfVue.$refs.slide.hide();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}else{
						vm.$message("<%=rb.getString("WuCanShuBianHua")%>")
					}
				}
			},
			cancel(){
				var vm = this;
				if(vm.checkParamChange()){//返回true为改变
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
				if(vm.show1588 &&　isFormChanged(vm.$refs.serverForm)){
					changeFlag = false;
				}
				if(vm.showVlan){
					if(isFormChanged(vm.$refs.vlanForm)){
						changeFlag = false;
					}
					if(vm.checkValueChange(vm.vlanForm.defaultVlanList,vm.vlanForm.vlanList) == false){
						changeFlag = false;
					}
					if(vm.checkValueChange(vm.vlanForm.defaultRouteList,vm.vlanForm.routeList) == false){
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
			'plmnGroup': function(list) {
				this.basicForm.plmn_id = list.join(',');
			}
		},
		mounted(){
			this.init();
			eventBus.$off("save-plan").$on("save-plan",this.submit);
			eventBus.$off("cancel-plan").$on("cancel-plan",this.cancel)
		}
	})
</script>