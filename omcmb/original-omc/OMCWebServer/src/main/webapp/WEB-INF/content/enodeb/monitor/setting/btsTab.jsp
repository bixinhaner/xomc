<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<div id="btsPanel" style='height: 100%;overflow: auto;background: #fff;min-width: 1000px;'>
	<el-form ref="btsForm" :model="btsForm" :rules="btsRules" 
	label-position="top" style='width:100%;height:100%;' inline>
		<el-collapse v-model="activeNames">
			<el-collapse-item name="omc">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">OMC</span>
					</p>
				</template>
				<p class='item-title-cls'>Management Server</p>
				<el-form-item label="Management Server" prop="ManagementServer" class="validate-item url-cls" style='width:100%'>
					<el-input disabled v-model="btsForm.ManagementServer">
						<template slot="append">Range:0~120</template>
					</el-input>
				</el-form-item>
				<el-form-item label="Port" prop="Port" class="validate-item">
					<el-input v-model="btsForm.Port">
						<template slot="append">Range:0~65535</template>
					</el-input>
				</el-form-item>
				<el-form-item label="SSL Enable" prop="SSLEnable">
					<el-switch v-model="btsForm.SSLEnable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				<!--  new add -->
				<el-form-item label="TR069 Binding" prop="TR069Binding">
					<el-select v-model="btsForm.TR069Binding">
						<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" v-for="item in bandingList" :label="item.text" :value="item.value"></el-option>
						<el-option v-if="isBaiblx_QRTB || isBaiblx_BLQ" v-for="item in trBandList" :key="item.value" :label="item.text" :value="item.value"></el-option>
					</el-select>
				</el-form-item>
				<p class='item-title-cls'>Cloud</p>
				<el-form-item label="CloudKey" prop="CloudKey" class="validate-item">
					<el-input disabled v-model="btsForm.CloudKey">
						<template slot="append">Range:0~6</template>
					</el-input>
				</el-form-item>
			</el-collapse-item>
			<el-collapse-item name='network'>
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">CoreNetwork</span>
					</p>
				</template>
				<!--  new add -->
				<el-form-item v-show="isBaiblq || isBaiblx_QRTB || isBaiblx_BLQ || isMLQ" label="Mode Selection">
					<el-radio-group v-model="modeSelection" @change="modeSelectionChange">
						<el-radio v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="cloudepc">Cloud EPC</el-radio>
						<el-radio v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="halob">HaloB</el-radio>
						<el-radio label="normal">Normal</el-radio>
						<el-radio v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="halod">HaloD</el-radio>
					</el-radio-group>
				</el-form-item>
				<!-- hidden Mode Prop-->
				<el-form-item v-show="false" label="Cloud EPC" prop="CloudEPC">
					<el-switch v-model="btsForm.CloudEPC" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				<el-form-item v-show="false" label="HaloB Enable" prop="HaloBEnable">
					<el-switch v-model="btsForm.HaloBEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				<!-- HaloB Part -->
				<div v-show="hasKey('HaloBMode') && modeSelection == 'halob'">
					<p class='item-title-cls'>HaloB</p>
					<el-form-item label="HaloB Mode" prop="HaloBMode">
						<el-select v-model="btsForm.HaloBMode">
							<el-option label="Centralized" value="1"></el-option>
							<el-option label="Single" value="2"></el-option>
						</el-select>
					</el-form-item>
				</div>

				<!-- Normal Part -->
				<div v-show="modeSelection == 'normal'">
					<el-form-item v-if="isMLQ" label="Multi S1 Mode" prop="multiS1Mode">
						<el-switch v-model="btsForm.multiS1Mode" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
					</el-form-item>

					<el-form-item label="S1 Connection Mode" prop="S1ConnectionMode">
						<el-select v-model="btsForm.S1ConnectionMode">
							<el-option label='ALL' value='All'></el-option>
							<el-option label='ONE' value='One'></el-option>
						</el-select>
					</el-form-item>
				</div>

				<div v-show="modeSelection == 'normal'">
					<p v-if="hasKey('S1CBinding')" class='item-title-cls'>S1-C Config</p>
					<el-form-item v-if="hasKey('S1CBinding')" label="S1-C Binding" prop="S1CBinding">
						<el-select v-model="btsForm.S1CBinding">
							<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" v-for="item in bandingList" :label="item.text" :value="item.value"></el-option>
							<el-option v-if="isBaiblx_QRTB || isBaiblx_BLQ" v-for="item in scbandingList" :label="item.text" :value="item.value"></el-option>
						</el-select>
					</el-form-item>
					<p v-if="hasKey('SGWSwitch')" class='item-title-cls'>S1-U Config</p>
					<el-form-item v-if="hasKey('SGWSwitch')" label="SGW Switch" prop="SGWSwitch">
						<el-switch v-model="btsForm.SGWSwitch" active-value="0" inactive-value="1" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
					</el-form-item>
					<el-form-item v-if="hasKey('SGWInterfaceBinding')" label="SGW Interface Binding" prop="SGWInterfaceBinding">
						<el-select v-model="btsForm.SGWInterfaceBinding" :disabled="btsForm.SGWSwitch =='1'">
							<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" v-for="item in bandingList" :label="item.text" :value="item.value"></el-option>
							<el-option v-if="isBaiblx_QRTB || isBaiblx_BLQ" v-for="item in SGWInterfaceBindingList" :key="item.value" :label="item.text" :value="item.value"></el-option>
						</el-select>
					</el-form-item>
				</div>
				<!-- HaloD Part -->
				<div v-show="hasKey('HaloDMode') && modeSelection == 'halod'">
					<p class='item-title-cls'>
						HaloD
						<span style='color: #FA5151; margin-left: 15px;font-size: 12px;'><%=rb.getString("ShiLianTiShi")%></span>
					</p>
					<el-form-item label="HaloD Mode" prop="HaloDMode">
						<el-select v-model="btsForm.HaloDMode">
							<el-option label="<%=rb.getString("ZhuZhan")%>" value="0"></el-option>
							<el-option label="<%=rb.getString("CongZhan1")%>" value="1"></el-option>
							<el-option label="<%=rb.getString("CongZhan2")%>" value="2"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item :label="btsForm.HaloDMode=='0'?'HaloD Destination Address':'HaloD Source Address'" prop="HaloDDestinationAddress">
						<el-input v-model="btsForm.HaloDDestinationAddress"></el-input>
					</el-form-item>
					<el-form-item v-show="btsForm.HaloDMode === '0'" label="WCG S1-C IFName" prop="WCGIFName">
						<el-select v-model="btsForm.WCGIFName">
							<el-option v-for="item in bandingList" :label="item.text" :value="item.value"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item v-show="btsForm.HaloDMode === '0'" label="WCG S1-U IFName" prop="WCGUIFName">
						<el-select v-model="btsForm.WCGUIFName">
							<el-option v-for="item in bandingList" :label="item.text" :value="item.value"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item v-show="btsForm.HaloDMode === '0'" label="MME Port" prop="MMEPort" class="validate-item">
						<el-input v-model="btsForm.MMEPort">
							<template slot="append">Range:1024~65535,integer</template>
						</el-input>
					</el-form-item>
					<el-form-item v-show="btsForm.HaloDMode === '0'" label="Slave station 1 IP" prop="slave1IP">
						<el-input v-model="btsForm.slave1IP"></el-input>
					</el-form-item>
					<el-form-item v-show="btsForm.HaloDMode === '0'" label="Slave station 2 IP" prop="slave2IP">
						<el-input v-model="btsForm.slave2IP"></el-input>
					</el-form-item>
				</div>

				<div v-show="modeSelection == 'normal'">
					<p class='item-title-cls'>MME</p>
					<el-form-item label='TAC' prop='TAC' class='validate-item'>
						<el-input v-model="btsForm.TAC">
							<template slot="append">Range:0~65535</template>
						</el-input>
					</el-form-item>
					<el-form-item label='S1 Link Port' prop="S1LinkPort" class='validate-item'>
						<el-input v-model="btsForm.S1LinkPort">
							<template slot="append">Range:0~65535</template>
						</el-input>
					</el-form-item>
					<div class='list-cls'>
						<div class='list-item-cls'>
							<el-form-item label="PLMN ID" style='margin-bottom:0px;position:relative;' class='validate-item' :class="plmnCls">
								<el-input v-model='plmnVal'></el-input>
								<span v-if="plmnGroup.length<6" @click='addPlmn' class='form-bt el-icon el-icon-plus' style='position:absolute;left:165px;top:-5px;'></span>
								<span class='item-tip'>Range:5~6digits,No more than 6</span>
							</el-form-item>
							<div style='overflow:auto;margin-bottom:20px;'>
								<el-form-item class='suffixItem' v-for='(domain,index) in plmnGroup' style="width: auto;">
									<div class='form-suffix'>
										<span class='text'>{{domain}}</span>
										<span style='font-size:14px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click="removePlmn(index)"></span>
									</div>
								</el-form-item>
								<el-form-item prop="PLMNID" style="width: 90%;">
									<el-input class="hide-input" style="border:none;width: 500px;" v-model="btsForm.PLMNID"></el-input>
								</el-form-item>
							</div>
							
							<el-form-item label="MME IP" style='margin-bottom:0px;min-width:625px;' v-show="showMME && btsForm.multiS1Mode != 'true'" :class="mmeCls">
								<el-input v-model='mmeVal'>
									<template slot="append">PLMN</template>
								</el-input>
								<el-select style='vertical-align:bottom;margin-left:-3px;' class='mmeSelect' v-model='mme_plmn'>
									<el-option v-for="item in plmnGroup" :label="item" :value="item"></el-option>
								</el-select>
								<span v-if="mmeGroup.length<16" class='el-icon el-icon-plus' style='vertical-align:middle;margin-left:10px;' @click='addMME("mme_plmn")'></span>
								<span class='item-tip'>Range:0~255,No more than 16 and not repeat</span>
							</el-form-item>
							<div style='overflow:auto;margin-bottom:20px;' v-show="showMME && btsForm.multiS1Mode != 'true'">
								<el-form-item class='suffixItem' v-for='(domain,index) in mmeGroup' style='width:250px;'>
									<div class='form-suffix' style='min-width:235px;'>
										<span class='text' style='min-width:100px;border-right:1px solid #A0C4F9;padding:6px 0px;height:10px;'>{{domain.mme}}</span>
										<span class='text' style='min-width:100px;'>PLMN:<span>{{domain.plmn}}</span></span>
										<span style='font-size:14px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click='removeMME(index)'></span>
									</div>
								</el-form-item>
								<el-form-item prop="MMEIPNEW" style="width: 90%;">
									<el-input class="hide-input" style="border:none;width: 300px;" v-model="btsForm.MMEIPNEW"></el-input>
								</el-form-item>
							</div>
							<el-form-item label="MME IP" style='margin-bottom:0px;position:relative;min-width:625px;' v-show="!showMME && btsForm.multiS1Mode != 'true'" class="validate-item" :class="mmeCls">
								<el-input v-model='mmeVal'></el-input>
								<span v-if="mmeGroup.length<16" class='el-icon el-icon-plus form-bt' style='position:absolute;left:165px;top:-5px;' @click='addMME("mme")'></span>
								<span class='item-tip'>Range:0~255,No more than 16 and not repeat</span>
							</el-form-item>
							<div style='overflow:auto;margin-bottom:20px;' v-show="!showMME && btsForm.multiS1Mode != 'true'">
								<el-form-item class='suffixItem' v-for='(domain,index) in mmeGroup' style='width:200px;'>
									<div class='form-suffix' style="min-width: 160px;">
										<span class='text' style='padding:6px 0px;height:10px;min-width:130px;'>{{domain}}</span>
										<span style='font-size:14px;line-height:22px;' class='form-bt-remove el-icon el-icon-close' @click='removeMME(index)'></span>
									</div>
								</el-form-item>
								<el-form-item prop="MMEIPOld" style="width: 90%;">
									<el-input class="hide-input" style="border:none;width: 300px;" v-model="btsForm.MMEIPOld"></el-input>
								</el-form-item>
							</div>
							
							<div v-show="btsForm.multiS1Mode == 'true'" style='margin-top:5px;width:95%;display:flex'>
								<span style='font-size:12px;font-weight:bold;flex:1'>
									MME IP <span style='color: #999; margin-left: 15px;'>(Add max 16)</span>
                                    <span style='color: #FA5151; margin-left: 15px;'><%=rb.getString("SheZhiHouChongQi")%></span>
								</span>
								<span v-show="btsForm.MMEBindList.length < 16" @click='addMultiMME' class='el-icon el-icon-circle-add' style='font-size:16px;'></span>
							</div>
							<div v-show="btsForm.multiS1Mode == 'true'" style="height:300px;max-height:300px;width:95%;border:1px solid #F3F3F3;margin-bottom:20px;">
								<el-ctable ref="ctableMultiIp" :data="btsForm.MMEBindList" :pagination="false" :rownumber="false">
									<el-table-column label=" " width="40" prop="" class-name="no-text-tips">
										<template slot-scope="scope">
											<span class="el-icon el-icon-operation-delete" @click="delMultiMME(scope.row, scope.$index)"></span>
										</template>
									</el-table-column>
									<el-table-column label="Index" prop="BindIndex" width="70"></el-table-column>
									<el-table-column label="MME IP" prop="MMEIp"></el-table-column>
									<el-table-column label="PLMN" prop="BindPLMNID"></el-table-column>
									<el-table-column label="S1-C IFName" prop="S1CIfname">
										<template slot-scope="scope">
											<span>{{IFNameMap[scope.row.S1CIfname]}}</span>
										</template>
									</el-table-column>
									<el-table-column label="S1-C IP" prop="S1CIp"></el-table-column>
									<el-table-column label="S1-U IFName" prop="S1UIfname">
										<template slot-scope="scope">
											<span>{{IFNameMap[scope.row.S1UIfname]}}</span>
										</template>
									</el-table-column>
									<el-table-column label="S1-U IP" prop="S1UIp"></el-table-column>
									<el-table-column label="Status" prop="MMEstatus">
										<template slot-scope="scope">
											<span v-if="scope.row.MMEstatus == '1'">Active</span>
											<span v-if="scope.row.MMEstatus == '0'">Inactive</span>
										</template>
									</el-table-column>
								</el-ctable>
							</div>
							<el-form-item v-show="false" prop="MMEBindList" class="validate-item">
								<el-input v-model="btsForm.MMEBindList"></el-input>
							</el-form-item>
						</div>
					</div>
				</div>

				
                <p v-if="!(isBaiblx_BLQ || isBaiblx_QRTB)" class='item-title-cls'>LGW</p>
             	<el-form-item v-if="!(isBaiblx_BLQ || isBaiblx_QRTB)" label="Enable" prop="lgwEnable">
					<el-switch v-model="btsForm.lgwEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				</br>
				<template v-if="btsForm.lgwEnable == '1' && !(isBaiblx_BLQ || isBaiblx_QRTB)">
			        <el-form-item label="Mode" prop="lgwMode">
						<el-select v-model="btsForm.lgwMode">
							<el-option label="NAT" value="0"></el-option>
							<el-option label="Router" value="1"></el-option>
							<el-option label="Bridge" value="2"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item label="Interface Binding" prop="InterfaceBinding">
						<el-select v-model="btsForm.InterfaceBinding" disabled>
							<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="WAN" value="eth1"></el-option>
							<el-option v-if="isBaiblx_QRTB || isBaiblx_BLQ" v-for="item in newQuickBandList" :key="item.value" :label="item.text" :value="item.value"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item label="IP POOL" prop="IPPOOL" v-if="btsForm.lgwMode !='2'">
						<el-input v-model="btsForm.IPPOOL" @change="validateStaticIP"></el-input>
					</el-form-item>
					<el-form-item label="IP POOL Netmask" prop="IPPOOLNetmask" v-if="btsForm.lgwMode !='2'">
						<el-input v-model="btsForm.IPPOOLNetmask" @change="validateStaticIP"></el-input>
					</el-form-item>
					<el-form-item v-if="btsForm.lgwMode == '1'" label="LGW Static IP Addr_Enable" prop="LGWStaticIPAddr_Enable" style='width:100%'>
						<el-switch v-model="btsForm.LGWStaticIPAddr_Enable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
					</el-form-item>
					<template v-if="btsForm.lgwMode == '1' && btsForm.LGWStaticIPAddr_Enable == '1'">
						<el-form-item label="LGW First Static IP Addr" prop="LGWFirstStaticIPAddr">
							<el-input v-model="btsForm.LGWFirstStaticIPAddr" @change="validateStaticIP"></el-input>
						</el-form-item>
						<el-form-item prop="IMSItoIPBindingRange" v-show=false>
							<el-input v-model="btsForm.IMSItoIPBindingRange"></el-input>
						</el-form-item>
						<el-form-item label="LGW Last Static IP Addr" prop="LGWLastStaticIPAddr">
							<el-input v-model="btsForm.LGWLastStaticIPAddr" @change="validateStaticIP"></el-input>
						</el-form-item>
						<el-form-item label="IMSI to IP Binding Range" style='margin-bottom:0px;position:relative;width:100%' class="validate-item" :class="bindCls">
							<el-input v-model='bindImsi' style='width:200px;'></el-input>&nbsp;&nbsp;—&nbsp;&nbsp;<el-input v-model='bindIp' style='width:200px;'></el-input>
							<span @click='addBind' class='form-bt el-icon el-icon-plus' style='vertical-align:middle' v-show="btsForm.LGWStaticIPAddr_Enable"></span>
							<span class='item-tip'>IMSI must be number and length of 15,IP must be in Static IP range</span>
						</el-form-item>
						<div style='overflow:auto'>
							<el-form-item class='suffixItem' v-for='(domain,index) in bindGroup' style='line-height:16px;width:auto;'>
								<div class='form-suffix' style='width:auto;'>
									<span class='text' style='width:auto;'>{{domain}}</span>
									<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeBind(index)'></span>
								</div>
							</el-form-item>
						</div>
					</template>
				</template>
			</el-collapse-item>
			<el-collapse-item name="sync">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Sync</span>
					</p>
				</template>

				<el-form-item v-if="hasKey('syncMode')" label="Sync Method" prop="syncMode">
					<el-select v-model="btsForm.syncMode">
						<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="NTP" value="1"></el-option>
						<el-option label="PTP" value="2"></el-option>
						<el-option v-if="!isMLQ" label="GNSS" value="3"></el-option>
						<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="NL" value="4"></el-option>
						<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="EXT_CLK" value="5"></el-option>
						<el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" label="EXT_PPS" value="6"></el-option>
						<el-option label="FREE_RUNNING" value="7"></el-option>
						<el-option v-if="isBaiblq || isMLQ" label="SINGLE_BEIDOU" value="8"></el-option>
						<el-option v-if="isBaiblq || isMLQ" label="GLONASS" value="9"></el-option>
						<el-option v-if="isBaiblq || isMLQ" label="GPS_BEIDOU" value="10"></el-option>
						<el-option v-if="isBaiblq || isMLQ" label="GPS_GLONASS" value="11"></el-option>
                        <el-option v-if="isMLQ" label="SYNC_SINGLE_GPS" value="31"></el-option>
                        <el-option v-if="isMLQ" label="SYNC_SINGLE_GLONASS" value="32"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label="Sync Mode" v-if="btsForm.syncMode == '2'" prop="syncType">
					<el-input v-model="btsForm.syncType" disabled></el-input>
				</el-form-item>
				<el-form-item label="PTP Trigger" v-if="btsForm.syncMode == '2'" prop="syncTrigger">
					<el-input v-model="btsForm.syncTrigger" disabled></el-input>
				</el-form-item>
				<el-form-item label="Interface" v-if="btsForm.syncMode == '2'" prop="syncInterface">
					<el-input v-model="btsForm.syncInterface" disabled></el-input>
				</el-form-item>
				<el-form-item label="Sync Port" v-if="btsForm.syncMode == '2'" prop="syncPort">
					<el-select v-model="btsForm.syncPort">
						<el-option label="Ethernet" value="Ethernet"></el-option>
						<el-option label="UDP" value="UDP"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label="Unicast Address" v-if="btsForm.syncMode == '2'" prop="syncAddress">
					<el-input v-model="btsForm.syncAddress" maxlength="100"></el-input>
				</el-form-item>
				
				<el-form-item label="Profile" v-if="btsForm.syncMode == '2'" prop="profile">
					<el-select v-model="btsForm.profile">
						<el-option label="1588v2" value="1588v2"></el-option>
						<el-option label="G8265" value="G8265"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label="Domain" v-if="btsForm.syncMode == '2'" prop="domain">
					<el-input v-model="btsForm.domain" type="number" max="255"></el-input>
				</el-form-item>

				<p v-show="hasKey('syncList') && btsForm.syncMode == '4'" class='item-title-cls'>NL Sync</p>
				<div v-show="hasKey('syncList') && btsForm.syncMode == '4'" style='margin-top:5px;width:95%;display:flex'>
					<span style='font-size:12px;font-weight:bold;flex:1'>NL Sync List</span>
					<span v-if="false" @click='addSync' class='el-icon el-icon-circle-add' style='font-size:24px;'></span>
				</div>
				<div v-show="hasKey('syncList') && btsForm.syncMode == '4'" style="height:300px;max-height:150px;width:95%;border:1px solid #F3F3F3">
					<el-ctable ref="ctableIpsec" :data="btsForm.syncList" :pagination="false">
						<el-table-column label="Operations" width="100" prop="" class-name="no-text-tips">
							<template slot-scope="scope">
								<span class="el-icon el-icon-operation-edit" @click="editSync(scope.row)" style="cursor: pointer;"></span>
								<span v-if="false" class="el-icon el-icon-operation-delete" @click="delSync(scope.row)" style="margin-left:10px;"></span>
							</template>
						</el-table-column>
						<el-table-column label="Index" prop="Index"></el-table-column>
						<el-table-column label="priority" prop="priority"></el-table-column>
						<el-table-column label="technology" prop="technology"></el-table-column>
						<el-table-column label="Band" prop="Band"></el-table-column>
						<el-table-column label="Channel Number" prop="ChannelNumber"></el-table-column>
						<el-table-column label="PCI" prop="PCI"></el-table-column>
						<el-table-column label="freqUncertaintyThreshold" width="200" prop="freqUncertaintyThreshold"></el-table-column>
						<el-table-column label="syncInterval" prop="syncInterval"></el-table-column>
						<el-table-column label="phaseOffset" prop="phaseOffset"></el-table-column>
					</el-ctable>
				</div>
				<el-form-item v-show="false" prop="syncList" class="validate-item">
					<el-input v-model="btsForm.syncList"></el-input>
				</el-form-item>
			</el-collapse-item>
			<el-collapse-item name="system">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">System</span>
					</p>
				</template>
				<p class='item-title-cls'>NTP</p>
				<el-form-item label="Enable" prop="ntpEnable">
					<el-switch v-model="btsForm.ntpEnable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
				</el-form-item>
				<el-form-item label="timeZone" prop="TimeZone">
					<el-select v-model="btsForm.TimeZone">
						<el-option v-for="item in timeZoneList" :label="item.text" :value="item.value"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item label="Port" prop="Port1" class="validate-item">
					<el-input v-model="btsForm.Port1">
						<template slot="append">Range:1~65535</template>
					</el-input>
				</el-form-item>
				<el-form-item label="Server1" prop="Server1" class="validate-item">
					<el-input v-model="btsForm.Server1">
						<template slot="append">IP Address</template>
					</el-input>
				</el-form-item>
				<el-form-item v-show="false" label="Port2" class="validate-item">
					<el-input v-model="btsForm.Port2">
						<template slot="append">Range:1~65535</template>
					</el-input>
				</el-form-item>
				<el-form-item label="Server2" prop="Server2" class="validate-item">
					<el-input v-model="btsForm.Server2">
						<template slot="append">IP Address</template>
					</el-input>
				</el-form-item>
				<el-form-item v-show="false" label="Port3" class="validate-item">
					<el-input v-model="btsForm.Port3">
						<template slot="append">Range:1~65535</template>
					</el-input>
				</el-form-item>
				<el-form-item label="Server3" prop="Server3" class="validate-item">
					<el-input v-model="btsForm.Server3">
						<template slot="append">IP Address</template>
					</el-input>
				</el-form-item>
			</el-collapse-item>
		</el-collapse>
	</el-form>
	<div class='addSlide' id='syncAddPanel' v-show="showSync">
		<el-form ref="syncForm" :model="syncForm" :rules="syncRules" label-position="left" label-width="160" style='width:100%;height:100%;' inline>
			<div style='height:37px;border-bottom:1px solid #EEE;line-height:37px;font-weight:bold;padding-left:10px;font-size:14px;'>
					Add NL Sync List
					<div class="circleIcon placeholder-bt" style="top: 4px;right:40px;" placeholder="<%=rb.getString("FanHui")%>">		
						<span class="el-icon el-icon-circle-goback" @click='closeSync'></span>
					</div>
			</div>
			<el-collapse v-model="activeNamesSync">
				<!-- Quick Setting -->
				<el-collapse-item name="sync">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">NL Sync Setting</span>
						</p>
					</template>
					<div>
						<el-form-item label="priority" prop="priority" class="validate-item">
							<el-input v-model="syncForm.priority">
								<template slot="append">Range:0~65535</template>
							</el-input>
						</el-form-item>
						<el-form-item label="Band" prop="Band" class="validate-item">
							<el-input v-model="syncForm.Band">
								<template slot="append">Range:0~255</template>
							</el-input>
						</el-form-item>
						<el-form-item label="Channel Number" prop="ChannelNumber" class="validate-item">
							<el-input v-model="syncForm.ChannelNumber">
								<template slot="append">Range:0~65535</template>
							</el-input>
						</el-form-item>
						<el-form-item label="PCI" prop="PCI" class="validate-item">
							<el-input v-model="syncForm.PCI">
								<template slot="append">Range:-1~503</template>
							</el-input>
						</el-form-item>
						<el-form-item label="freUncertainty Threshold" prop="freqUncertaintyThreshold" class="validate-item">
							<el-input v-model="syncForm.freqUncertaintyThreshold">
								<template slot="append">Range:-32768~32767</template>
							</el-input>
						</el-form-item>
						<el-form-item label="syncInterval" prop="syncInterval" class="validate-item">
							<el-input v-model="syncForm.syncInterval">
								<template slot="append">Range:1~60</template>
							</el-input>
						</el-form-item>
						<el-form-item label="phaseOffset" prop="phaseOffset" class="validate-item">
							<el-input v-model="syncForm.phaseOffset">
								<template slot="append">Range:-32768~32767</template>
							</el-input>
						</el-form-item>
					</div>
				</el-collapse-item>
			</el-collapse>
		</el-form>
	</div>

	<el-dialog title="Add" id="multiMMEAddPanel" :visible.sync="multiMMEAddShow" width="600px" :close-on-click-modal="false" :modal-append-to-body="false">
		<el-form ref="multiForm" :model="multiForm" :rules="multiRules" label-position="top" label-width="160" style='width:100%;height:300px;' inline>
			<div v-if="false" style='height:37px;border-bottom:1px solid #EEE;line-height:37px;font-weight:bold;padding-left:10px;font-size:14px;'>
				Add
				<div class="circleIcon placeholder-bt" style="top: 4px;right:40px;" placeholder="<%=rb.getString("FanHui")%>">		
					<span class="el-icon el-icon-circle-goback" @click='closeMultiMME'></span>
				</div>
			</div>

			<el-form-item label="Binding Type" style="width: 90%;">
				<el-radio-group v-model="multiForm.type">
					<el-radio label="1">WAN</el-radio>
					<el-radio label="2">IPSec</el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item label="MME IP" prop="MMEIp" required>
				<el-select v-model="multiForm.MMEIp" @change="mmeplmnChange">
					<el-option v-for="item in mmePlmnList"
						:label="item.name" 
						:value="item.name" 
						:key="item.name + item.plmn + '_p'"
					></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="PLMN" prop="BindPLMNID">
				<el-input v-model="multiForm.BindPLMNID" disabled></el-input>
			</el-form-item>
			<el-form-item label="S1-C IFName" prop="S1CIfname" required>
				<el-select v-model="multiForm.S1CIfname" @change="s1cNameChange">
					<el-option v-for="item in wanIfNameList" v-if="multiForm.type == '1'"
						:label="item.name" 
						:value="item.value" 
						:key="item.name + item.ip + '_c'"
					></el-option>

					<el-option v-for="item in ipsecIfNameList" v-if="multiForm.type == '2'"
						:label="item.name" 
						:value="item.name" 
						:key="item.name + item.ip + '_c'"
					></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="S1-U IFName" prop="S1UIfname" required>
				<el-select v-model="multiForm.S1UIfname" @change="s1uNameChange">
					<el-option v-for="item in wanIfNameList" v-if="multiForm.type == '1'"
						:label="item.name" 
						:value="item.value" 
						:key="item.name + item.ip + '_u'"
					></el-option>

					<el-option v-for="item in ipsecIfNameList" v-if="multiForm.type == '2'"
						:label="item.name" 
						:value="item.name" 
						:key="item.name + item.ip + '_u'"
					></el-option>
				</el-select>
			</el-form-item>
		</el-form>
		<div slot="footer">
			<el-button type="primary" @click="saveMultiMME">OK</el-button>
			<el-button @click="closeMultiMME">Cancel</el-button>
		</div>
	</el-dialog>
</div>
<script>
	var btsVm = new Vue({
		el:'#btsPanel',
		data(){
			var vm = this;
			
			var validateRange = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(value == '' || (reg.test(value) && value >= min && value <= max)){
						callback();
					}else{
						callback(new Error('format error'))
					}
				},
				validSyncAddress = (rule,value,callback)=>{
					if(vm.btsForm.syncMode == '2') {
						if(value.length) {
							if(isValidIP(value)) {
								callback();
							}else {
								callback('Please input valid IP Address');
							}
						}else {
							callback();
						}
					}else {
						callback();
					}
				},
				validDomain = (rule,value,callback)=>{
					var min = 0;
					var max = 255;
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					
					if(vm.btsForm.syncMode == '2') {
						if(!vm.isMLQ && value == '' || (reg.test(value) && value >= min && value <= max)){
							callback();
						}else{
							callback(new Error('Range: 0-255 Interger'))
						}
					}else {
						callback();
					}
				},
				validateIP = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/,
						serverReg = /^[a-zA-Z0-9-_]+(\.[a-zA-Z0-9-_]+)+$/;
					
					if(value == '' || isValidIP(value) || serverReg.test(value)) {
						if(rule.code == 'lgw') {
							vm.isAllBindIpInRange();

							var pool = vm.btsForm.IPPOOL,
								mask = vm.btsForm.IPPOOLNetmask,
								firstIp = vm.btsForm.LGWFirstStaticIPAddr,
								lastIp = vm.btsForm.LGWLastStaticIPAddr;
							
							if(isValidIP(value)) {
								if(pool && isValidIP(pool) && mask) {
									var startIp = getLowAddr(pool, mask),
										endIp = getHighAddr(pool, mask),
										ipRangeTip = '<%=rb.getString("IPBangDingFanWei")%>' + startIp + '-' + endIp;

									if(vm.compareIp(value,startIp,endIp)) {
										if(lastIp && isValidIP(value)) {
											var startNum = vm.changeIpToNum(firstIp),
												endNum = vm.changeIpToNum(lastIp);
											
											if(startNum - endNum >= 0) {
												callback('First IP should less than Last IP');
											}else{
												callback();
											}
										}else {
											callback();
										}
									}else {
										callback(ipRangeTip);
									}
								}else {
									callback();
								}
							}else {
								callback('Please input valid IP Address')
							}
						}else {
							callback();
						}
					}else{
						callback(new Error('Please input valid IP Address'))
					}
				},
				validateListInRange = (rule,value,callback)=>{
					var bool = vm.isAllBindIpInRange();

					if(bool) {
						callback();
					}else {
						callback('IMSI to IP Binding exist out range of Static IP')
					}
				},
				validateDestinationAddr = (rule,value,callback)=>{
					var isHaloD = vm.modeSelection == 'halod' && vm.hasKey('HaloDMode');

					if(isHaloD) {
						if(isValidIP(value)) {
							callback();
						}else {
							callback('Please input valid IP Address');
						}
					}else {
						callback();
					}
				},
				validateHalodIP = (rule,value,callback)=>{
					var isHaloD = vm.modeSelection == 'halod' && vm.hasKey('HaloDMode');

					if(isHaloD && value) {
						if(isValidIP(value)) {
							callback();
						}else {
							callback('Please input valid IP Address');
						}
					}else {
						callback();
					}
				},
				validateMMEPort = (rule,value,callback)=>{
					var isHaloD = vm.modeSelection == 'halod' && vm.hasKey('HaloDMode'),
						isMain = vm.btsForm.HaloDMode === '0';
					//1024~65535
					if(isHaloD && isMain) {
						if(isNaN(value) || value - 1024 < 0 || value - 65535 > 0) {
							callback(' ');
						}else {
							callback();
						}
					}else {
						callback();
					}
				},
				validMultiMMEIP = (rule,value,callback) => {
					var existMMEs = vm.btsForm.MMEBindList.map(function(item){ return item.MMEIp; });
					if(value) {
						if(existMMEs.includes(value)) {
							callback('MME IP existed')
						}else {
							callback()
						}
					}else {
						callback('Please select MME IP')
					}
				};

			return {
				isBaiblx_QRTB: settingVue.selectedRow.platformType == 'BLX' && settingVue.selectedRow.network_model == 'TDDMode',
				isBaiblx_BLQ: settingVue.selectedRow.platformType == 'BLX' && settingVue.selectedRow.network_model == 'FDDMode',

				isBaiblq: settingVue.selectedRow.product == 'BAIBLQ',
				isMLQ: settingVue.selectedRow.product == 'MLQ',

				rebootMap: {},
				delRecord: {
					MMEBindList: [],
				},

				bandingList: [],
				codeTableList: [],
				scbandingList: [],
				SGWInterfaceBindingList: [],
                newQuickBandList: [],
				trBandList: [],
				btsForm:{
                    ManagementServer:'',
                    Port:'',
                    SSLEnable:true,
                    CloudKey:'',
                    CloudEPC:'',
                    PLMNID:'',
                    TAC:'',
                    MMEIP:'',
                    MMEIPOld:'',
					MMEIPNEW:'',
					IMSItoIPBindingRange: '',
					S1LinkPort:'',
					S1ConnectionMode:'',

					'MMEBindList': [],
					'multiS1Mode': 'false',

					HaloBEnable:'',
					HaloBMode:'',
					lgwEnable:'',
					lgwMode:'0',
					InterfaceBinding:'',
					IPPOOL:'',
					IPPOOLNetmask:'',
					LGWStaticIPAddr_Enable:'0',
					LGWFirstStaticIPAddr:'',
					LGWLastStaticIPAddr:'',
					TimeZone:'',
					ntpEnable:'',
					Port1:'',
					Server1:'',
					Port2:'',
					Server2:'',
					Port3:'',
					Server3:'',
					syncList:[],

					syncMode:'',
					syncType:'',
					syncTrigger:'',
					syncInterface:'',
					syncPort:'',
					syncAddress:'',
					profile: '',
					domain: '',

					TR069Binding: '',
					S1CBinding: '',
					SGWSwitch: '',
					SGWInterfaceBinding: '',
					HaloDMode: '',
					HaloDDestinationAddress: '',
					WCGIFName: '',
					WCGUIFName: '',
					MMEPort: '',
					slave1IP: '',
					slave2IP: '',
				},
				btsRules:{
					Port:[{validator:validateRange,min:0,max:65535}],
					TAC:[{validator:validateRange,min:0,max:65535}],
					S1LinkPort:[{validator:validateRange,min:0,max:65535}],
					IPPOOL:[{validator:validateIP}],
					IPPOOLNetmask:[{validator:validateIP}],
					LGWFirstStaticIPAddr:[{validator:validateIP,code: 'lgw'}],
					LGWLastStaticIPAddr:[{validator:validateIP,code: 'lgw'}],
					IMSItoIPBindingRange: [{validator:validateListInRange}],
					Port1:[{validator:validateRange,min:1,max:65535}],
					Server1:[{validator:validateIP,min:0,max:64}],
					//Port2:[{validator:validateRange,min:1,max:65535}],
					Server2:[{validator:validateIP,min:0,max:64}],
					//Port3:[{validator:validateRange,min:1,max:65535}],
					Server3:[{validator:validateIP,min:0,max:64}],

					syncAddress: [{validator:validSyncAddress}],
					domain: [{validator:validDomain}],
					
					HaloDDestinationAddress: [{validator:validateDestinationAddr}],
					MMEPort: [{validator:validateMMEPort}],
					slave1IP: [{validator:validateHalodIP}],
					slave2IP: [{validator:validateHalodIP}],
				},
				modeSelection: '',
				activeNames:['omc','network','sync','system'],
				plmnGroup:[],
				epcSwitchFlag:true,
				plmnVal:'',
				mmeVal:'',
				mme_plmn:'',
				mmeGroup:[],
				bindGroup:[],
				bindImsi:'',
				bindIp:'',

				multiMMEAddShow: false,
				wanIfNameList: [],
				ipsecIfNameList: [],
				mmePlmnList: [],
				multiForm: {
					type: '',
					BindPLMNID: '',
					MMEIp: '',
					S1UIp: '',
					S1UIfname: '',
					S1CIp: '',
					S1CIfname: '',
					MMEstatus: '',
					BindIndex: ''
				},
				multiRules: {
					MMEIp: [{required: true, validator: validMultiMMEIP}],
					S1UIfname: [{required: true, message: 'Please select S1-U IFName'}],
					S1CIfname: [{required: true, message: 'Please select S1-C IFName'}],
				},

				syncData:[],
				showSync:false,
				syncForm:{
					Index:'',
					priority:'',
					technology:'',
					Band:'',
					ChannelNumber:'',
					PCI:'',
					freqUncertaintyThreshold:'',
					syncInterval:'',
					phaseOffset:''
				},
				syncRules:{
					priority:[{validator:validateRange,min:0,max:65535}],
					Band:[{validator:validateRange,min:0,max:255}],
					ChannelNumber:[{validator:validateRange,min:0,max:65535}],
					PCI:[{validator:validateRange,min:-1,max:503}],
					freqUncertaintyThreshold:[{validator:validateRange,min:-32768,max:32767}],
					syncInterval:[{validator:validateRange,min:1,max:60}],
					phaseOffset:[{validator:validateRange,min:-32768,max:32767}],
				},
				activeNamesSync:['sync'],
				smallCellCode:'',
				codeList:[],
				checkedApns:'',
				policyName:'',
				apnList: [],
                policyList: [],
                visible: false,
                showMME:false,
                plmnCls:'',
                mmeCls:'',
                bindCls:'',
                timeZoneList:[{"text":"Africa/Abidjan","value":"Africa/Abidjan"},{"text":"Africa/Accra","value":"Africa/Accra"},{"text":"Africa/Addis_Ababa","value":"Africa/Addis_Ababa"},{"text":"Africa/Algiers","value":"Africa/Algiers"},{"text":"Africa/Asmara","value":"Africa/Asmara"},{"text":"Africa/Bamako","value":"Africa/Bamako"},{"text":"Africa/Bangui","value":"Africa/Bangui"},{"text":"Africa/Banjul","value":"Africa/Banjul"},{"text":"Africa/Bissau","value":"Africa/Bissau"},{"text":"Africa/Blantyre","value":"Africa/Blantyre"},{"text":"Africa/Brazzaville","value":"Africa/Brazzaville"},{"text":"Africa/Bujumbura","value":"Africa/Bujumbura"},{"text":"Africa/Cairo","value":"Africa/Cairo"},{"text":"Africa/Casablanca","value":"Africa/Casablanca"},{"text":"Africa/Ceuta","value":"Africa/Ceuta"},{"text":"Africa/Conakry","value":"Africa/Conakry"},{"text":"Africa/Dakar","value":"Africa/Dakar"},{"text":"Africa/Dar_es_Salaam","value":"Africa/Dar_es_Salaam"},{"text":"Africa/Djibouti","value":"Africa/Djibouti"},{"text":"Africa/Douala","value":"Africa/Douala"},{"text":"Africa/El_Aaiun","value":"Africa/El_Aaiun"},{"text":"Africa/Freetown","value":"Africa/Freetown"},{"text":"Africa/Gaborone","value":"Africa/Gaborone"},{"text":"Africa/Harare","value":"Africa/Harare"},{"text":"Africa/Johannesburg","value":"Africa/Johannesburg"},{"text":"Africa/Juba","value":"Africa/Juba"},{"text":"Africa/Kampala","value":"Africa/Kampala"},{"text":"Africa/Khartoum","value":"Africa/Khartoum"},{"text":"Africa/Kigali","value":"Africa/Kigali"},{"text":"Africa/Kinshasa","value":"Africa/Kinshasa"},{"text":"Africa/Lagos","value":"Africa/Lagos"},{"text":"Africa/Libreville","value":"Africa/Libreville"},{"text":"Africa/Lome","value":"Africa/Lome"},{"text":"Africa/Luanda","value":"Africa/Luanda"},{"text":"Africa/Lubumbashi","value":"Africa/Lubumbashi"},{"text":"Africa/Lusaka","value":"Africa/Lusaka"},{"text":"Africa/Malabo","value":"Africa/Malabo"},{"text":"Africa/Maputo","value":"Africa/Maputo"},{"text":"Africa/Maseru","value":"Africa/Maseru"},{"text":"Africa/Mbabane","value":"Africa/Mbabane"},{"text":"Africa/Mogadishu","value":"Africa/Mogadishu"},{"text":"Africa/Monrovia","value":"Africa/Monrovia"},{"text":"Africa/Nairobi","value":"Africa/Nairobi"},{"text":"Africa/Ndjamena","value":"Africa/Ndjamena"},{"text":"Africa/Niamey","value":"Africa/Niamey"},{"text":"Africa/Nouakchott","value":"Africa/Nouakchott"},{"text":"Africa/Ouagadougou","value":"Africa/Ouagadougou"},{"text":"Africa/Porto-Novo","value":"Africa/Porto-Novo"},{"text":"Africa/Sao_Tome","value":"Africa/Sao_Tome"},{"text":"Africa/Tripoli","value":"Africa/Tripoli"},{"text":"Africa/Tunis","value":"Africa/Tunis"},{"text":"Africa/Windhoek","value":"Africa/Windhoek"},{"text":"America/Adak","value":"America/Adak"},{"text":"America/Anchorage","value":"America/Anchorage"},{"text":"America/Anguilla","value":"America/Anguilla"},{"text":"America/Antigua","value":"America/Antigua"},{"text":"America/Araguaina","value":"America/Araguaina"},{"text":"America/Argentina/Buenos_Aires","value":"America/Argentina/Buenos_Aires"},{"text":"America/Argentina/Catamarca","value":"America/Argentina/Catamarca"},{"text":"America/Argentina/Cordoba","value":"America/Argentina/Cordoba"},{"text":"America/Argentina/Jujuy","value":"America/Argentina/Jujuy"},{"text":"America/Argentina/La_Rioja","value":"America/Argentina/La_Rioja"},{"text":"America/Argentina/Mendoza","value":"America/Argentina/Mendoza"},{"text":"America/Argentina/Rio_Gallegos","value":"America/Argentina/Rio_Gallegos"},{"text":"America/Argentina/Salta","value":"America/Argentina/Salta"},{"text":"America/Argentina/San_Juan","value":"America/Argentina/San_Juan"},{"text":"America/Argentina/San_Luis","value":"America/Argentina/San_Luis"},{"text":"America/Argentina/Tucuman","value":"America/Argentina/Tucuman"},{"text":"America/Argentina/Ushuaia","value":"America/Argentina/Ushuaia"},{"text":"America/Aruba","value":"America/Aruba"},{"text":"America/Asuncion","value":"America/Asuncion"},{"text":"America/Atikokan","value":"America/Atikokan"},{"text":"America/Bahia","value":"America/Bahia"},{"text":"America/Bahia_Banderas","value":"America/Bahia_Banderas"},{"text":"America/Barbados","value":"America/Barbados"},{"text":"America/Belem","value":"America/Belem"},{"text":"America/Belize","value":"America/Belize"},{"text":"America/Blanc-Sablon","value":"America/Blanc-Sablon"},{"text":"America/Boa_Vista","value":"America/Boa_Vista"},{"text":"America/Bogota","value":"America/Bogota"},{"text":"America/Boise","value":"America/Boise"},{"text":"America/Cambridge_Bay","value":"America/Cambridge_Bay"},{"text":"America/Campo_Grande","value":"America/Campo_Grande"},{"text":"America/Cancun","value":"America/Cancun"},{"text":"America/Caracas","value":"America/Caracas"},{"text":"America/Cayenne","value":"America/Cayenne"},{"text":"America/Cayman","value":"America/Cayman"},{"text":"America/Chicago","value":"America/Chicago"},{"text":"America/Chihuahua","value":"America/Chihuahua"},{"text":"America/Costa_Rica","value":"America/Costa_Rica"},{"text":"America/Creston","value":"America/Creston"},{"text":"America/Cuiaba","value":"America/Cuiaba"},{"text":"America/Curacao","value":"America/Curacao"},{"text":"America/Danmarkshavn","value":"America/Danmarkshavn"},{"text":"America/Dawson","value":"America/Dawson"},{"text":"America/Dawson_Creek","value":"America/Dawson_Creek"},{"text":"America/Denver","value":"America/Denver"},{"text":"America/Detroit","value":"America/Detroit"},{"text":"America/Dominica","value":"America/Dominica"},{"text":"America/Edmonton","value":"America/Edmonton"},{"text":"America/Eirunepe","value":"America/Eirunepe"},{"text":"America/El_Salvador","value":"America/El_Salvador"},{"text":"America/Fortaleza","value":"America/Fortaleza"},{"text":"America/Glace_Bay","value":"America/Glace_Bay"},{"text":"America/Godthab","value":"America/Godthab"},{"text":"America/Goose_Bay","value":"America/Goose_Bay"},{"text":"America/Grand_Turk","value":"America/Grand_Turk"},{"text":"America/Grenada","value":"America/Grenada"},{"text":"America/Guadeloupe","value":"America/Guadeloupe"},{"text":"America/Guatemala","value":"America/Guatemala"},{"text":"America/Guayaquil","value":"America/Guayaquil"},{"text":"America/Guyana","value":"America/Guyana"},{"text":"America/Halifax","value":"America/Halifax"},{"text":"America/Havana","value":"America/Havana"},{"text":"America/Hermosillo","value":"America/Hermosillo"},{"text":"America/Indiana/Indianapolis","value":"America/Indiana/Indianapolis"},{"text":"America/Indiana/Knox","value":"America/Indiana/Knox"},{"text":"America/Indiana/Marengo","value":"America/Indiana/Marengo"},{"text":"America/Indiana/Petersburg","value":"America/Indiana/Petersburg"},{"text":"America/Indiana/Tell_City","value":"America/Indiana/Tell_City"},{"text":"America/Indiana/Vevay","value":"America/Indiana/Vevay"},{"text":"America/Indiana/Vincennes","value":"America/Indiana/Vincennes"},{"text":"America/Indiana/Winamac","value":"America/Indiana/Winamac"},{"text":"America/Inuvik","value":"America/Inuvik"},{"text":"America/Iqaluit","value":"America/Iqaluit"},{"text":"America/Jamaica","value":"America/Jamaica"},{"text":"America/Juneau","value":"America/Juneau"},{"text":"America/Kentucky/Louisville","value":"America/Kentucky/Louisville"},{"text":"America/Kentucky/Monticello","value":"America/Kentucky/Monticello"},{"text":"America/Kralendijk","value":"America/Kralendijk"},{"text":"America/La_Paz","value":"America/La_Paz"},{"text":"America/Lima","value":"America/Lima"},{"text":"America/Los_Angeles","value":"America/Los_Angeles"},{"text":"America/Lower_Princes","value":"America/Lower_Princes"},{"text":"America/Maceio","value":"America/Maceio"},{"text":"America/Managua","value":"America/Managua"},{"text":"America/Manaus","value":"America/Manaus"},{"text":"America/Marigot","value":"America/Marigot"},{"text":"America/Martinique","value":"America/Martinique"},{"text":"America/Matamoros","value":"America/Matamoros"},{"text":"America/Mazatlan","value":"America/Mazatlan"},{"text":"America/Menominee","value":"America/Menominee"},{"text":"America/Merida","value":"America/Merida"},{"text":"America/Metlakatla","value":"America/Metlakatla"},{"text":"America/Mexico_City","value":"America/Mexico_City"},{"text":"America/Miquelon","value":"America/Miquelon"},{"text":"America/Moncton","value":"America/Moncton"},{"text":"America/Monterrey","value":"America/Monterrey"},{"text":"America/Montevideo","value":"America/Montevideo"},{"text":"America/Montserrat","value":"America/Montserrat"},{"text":"America/Nassau","value":"America/Nassau"},{"text":"America/New_York","value":"America/New_York"},{"text":"America/Nipigon","value":"America/Nipigon"},{"text":"America/Nome","value":"America/Nome"},{"text":"America/Noronha","value":"America/Noronha"},{"text":"America/North_Dakota/Beulah","value":"America/North_Dakota/Beulah"},{"text":"America/North_Dakota/Center","value":"America/North_Dakota/Center"},{"text":"America/North_Dakota/New_Salem","value":"America/North_Dakota/New_Salem"},{"text":"America/Ojinaga","value":"America/Ojinaga"},{"text":"America/Panama","value":"America/Panama"},{"text":"America/Pangnirtung","value":"America/Pangnirtung"},{"text":"America/Paramaribo","value":"America/Paramaribo"},{"text":"America/Phoenix","value":"America/Phoenix"},{"text":"America/Port_of_Spain","value":"America/Port_of_Spain"},{"text":"America/Port-au-Prince","value":"America/Port-au-Prince"},{"text":"America/Porto_Velho","value":"America/Porto_Velho"},{"text":"America/Puerto_Rico","value":"America/Puerto_Rico"},{"text":"America/Rainy_River","value":"America/Rainy_River"},{"text":"America/Rankin_Inlet","value":"America/Rankin_Inlet"},{"text":"America/Recife","value":"America/Recife"},{"text":"America/Regina","value":"America/Regina"},{"text":"America/Resolute","value":"America/Resolute"},{"text":"America/Rio_Branco","value":"America/Rio_Branco"},{"text":"America/Santa_Isabel","value":"America/Santa_Isabel"},{"text":"America/Santarem","value":"America/Santarem"},{"text":"America/Santiago","value":"America/Santiago"},{"text":"America/Santo_Domingo","value":"America/Santo_Domingo"},{"text":"America/Sao_Paulo","value":"America/Sao_Paulo"},{"text":"America/Scoresbysund","value":"America/Scoresbysund"},{"text":"America/Sitka","value":"America/Sitka"},{"text":"America/St_Barthelemy","value":"America/St_Barthelemy"},{"text":"America/St_Johns","value":"America/St_Johns"},{"text":"America/St_Kitts","value":"America/St_Kitts"},{"text":"America/St_Lucia","value":"America/St_Lucia"},{"text":"America/St_Thomas","value":"America/St_Thomas"},{"text":"America/St_Vincent","value":"America/St_Vincent"},{"text":"America/Swift_Current","value":"America/Swift_Current"},{"text":"America/Tegucigalpa","value":"America/Tegucigalpa"},{"text":"America/Thule","value":"America/Thule"},{"text":"America/Thunder_Bay","value":"America/Thunder_Bay"},{"text":"America/Tijuana","value":"America/Tijuana"},{"text":"America/Toronto","value":"America/Toronto"},{"text":"America/Tortola","value":"America/Tortola"},{"text":"America/Vancouver","value":"America/Vancouver"},{"text":"America/Whitehorse","value":"America/Whitehorse"},{"text":"America/Winnipeg","value":"America/Winnipeg"},{"text":"America/Yakutat","value":"America/Yakutat"},{"text":"America/Yellowknife","value":"America/Yellowknife"},{"text":"Antarctica/Casey","value":"Antarctica/Casey"},{"text":"Antarctica/Davis","value":"Antarctica/Davis"},{"text":"Antarctica/DumontDUrville","value":"Antarctica/DumontDUrville"},{"text":"Antarctica/Macquarie","value":"Antarctica/Macquarie"},{"text":"Antarctica/Mawson","value":"Antarctica/Mawson"},{"text":"Antarctica/McMurdo","value":"Antarctica/McMurdo"},{"text":"Antarctica/Palmer","value":"Antarctica/Palmer"},{"text":"Antarctica/Rothera","value":"Antarctica/Rothera"},{"text":"Antarctica/Syowa","value":"Antarctica/Syowa"},{"text":"Antarctica/Troll","value":"Antarctica/Troll"},{"text":"Antarctica/Troll","value":"Antarctica/Troll"},{"text":"Antarctica/Vostok","value":"Antarctica/Vostok"},{"text":"Arctic/Longyearbyen","value":"Arctic/Longyearbyen"},{"text":"Asia/Aden","value":"Asia/Aden"},{"text":"Asia/Almaty","value":"Asia/Almaty"},{"text":"Asia/Amman","value":"Asia/Amman"},{"text":"Asia/Anadyr","value":"Asia/Anadyr"},{"text":"Asia/Aqtau","value":"Asia/Aqtau"},{"text":"Asia/Aqtobe","value":"Asia/Aqtobe"},{"text":"Asia/Ashgabat","value":"Asia/Ashgabat"},{"text":"Asia/Baghdad","value":"Asia/Baghdad"},{"text":"Asia/Bahrain","value":"Asia/Bahrain"},{"text":"Asia/Baku","value":"Asia/Baku"},{"text":"Asia/Bangkok","value":"Asia/Bangkok"},{"text":"Asia/Beirut","value":"Asia/Beirut"},{"text":"Asia/Bishkek","value":"Asia/Bishkek"},{"text":"Asia/Brunei","value":"Asia/Brunei"},{"text":"Asia/Chita","value":"Asia/Chita"},{"text":"Asia/Choibalsan","value":"Asia/Choibalsan"},{"text":"Asia/Colombo","value":"Asia/Colombo"},{"text":"Asia/Damascus","value":"Asia/Damascus"},{"text":"Asia/Dhaka","value":"Asia/Dhaka"},{"text":"Asia/Dili","value":"Asia/Dili"},{"text":"Asia/Dubai","value":"Asia/Dubai"},{"text":"Asia/Dushanbe","value":"Asia/Dushanbe"},{"text":"Asia/Gaza","value":"Asia/Gaza"},{"text":"Asia/Hebron","value":"Asia/Hebron"},{"text":"Asia/Ho_Chi_Minh","value":"Asia/Ho_Chi_Minh"},{"text":"Asia/Hong_Kong","value":"Asia/Hong_Kong"},{"text":"Asia/Hovd","value":"Asia/Hovd"},{"text":"Asia/Irkutsk","value":"Asia/Irkutsk"},{"text":"Asia/Jakarta","value":"Asia/Jakarta"},{"text":"Asia/Jayapura","value":"Asia/Jayapura"},{"text":"Asia/Jerusalem","value":"Asia/Jerusalem"},{"text":"Asia/Kabul","value":"Asia/Kabul"},{"text":"Asia/Kamchatka","value":"Asia/Kamchatka"},{"text":"Asia/Karachi","value":"Asia/Karachi"},{"text":"Asia/Kathmandu","value":"Asia/Kathmandu"},{"text":"Asia/Khandyga","value":"Asia/Khandyga"},{"text":"Asia/Kolkata","value":"Asia/Kolkata"},{"text":"Asia/Krasnoyarsk","value":"Asia/Krasnoyarsk"},{"text":"Asia/Kuala_Lumpur","value":"Asia/Kuala_Lumpur"},{"text":"Asia/Kuching","value":"Asia/Kuching"},{"text":"Asia/Kuwait","value":"Asia/Kuwait"},{"text":"Asia/Macau","value":"Asia/Macau"},{"text":"Asia/Magadan","value":"Asia/Magadan"},{"text":"Asia/Makassar","value":"Asia/Makassar"},{"text":"Asia/Manila","value":"Asia/Manila"},{"text":"Asia/Muscat","value":"Asia/Muscat"},{"text":"Asia/Nicosia","value":"Asia/Nicosia"},{"text":"Asia/Novokuznetsk","value":"Asia/Novokuznetsk"},{"text":"Asia/Novosibirsk","value":"Asia/Novosibirsk"},{"text":"Asia/Omsk","value":"Asia/Omsk"},{"text":"Asia/Oral","value":"Asia/Oral"},{"text":"Asia/Phnom_Penh","value":"Asia/Phnom_Penh"},{"text":"Asia/Pontianak","value":"Asia/Pontianak"},{"text":"Asia/Pyongyang","value":"Asia/Pyongyang"},{"text":"Asia/Qatar","value":"Asia/Qatar"},{"text":"Asia/Qyzylorda","value":"Asia/Qyzylorda"},{"text":"Asia/Rangoon","value":"Asia/Rangoon"},{"text":"Asia/Riyadh","value":"Asia/Riyadh"},{"text":"Asia/Sakhalin","value":"Asia/Sakhalin"},{"text":"Asia/Samarkand","value":"Asia/Samarkand"},{"text":"Asia/Seoul","value":"Asia/Seoul"},{"text":"Asia/Shanghai","value":"Asia/Shanghai"},{"text":"Asia/Singapore","value":"Asia/Singapore"},{"text":"Asia/Srednekolymsk","value":"Asia/Srednekolymsk"},{"text":"Asia/Taipei","value":"Asia/Taipei"},{"text":"Asia/Tashkent","value":"Asia/Tashkent"},{"text":"Asia/Tbilisi","value":"Asia/Tbilisi"},{"text":"Asia/Thimphu","value":"Asia/Thimphu"},{"text":"Asia/Tokyo","value":"Asia/Tokyo"},{"text":"Asia/Ulaanbaatar","value":"Asia/Ulaanbaatar"},{"text":"Asia/Urumqi","value":"Asia/Urumqi"},{"text":"Asia/Ust-Nera","value":"Asia/Ust-Nera"},{"text":"Asia/Vientiane","value":"Asia/Vientiane"},{"text":"Asia/Vladivostok","value":"Asia/Vladivostok"},{"text":"Asia/Yakutsk","value":"Asia/Yakutsk"},{"text":"Asia/Yekaterinburg","value":"Asia/Yekaterinburg"},{"text":"Asia/Yerevan","value":"Asia/Yerevan"},{"text":"Atlantic/Azores","value":"Atlantic/Azores"},{"text":"Atlantic/Bermuda","value":"Atlantic/Bermuda"},{"text":"Atlantic/Canary","value":"Atlantic/Canary"},{"text":"Atlantic/Cape_Verde","value":"Atlantic/Cape_Verde"},{"text":"Atlantic/Faroe","value":"Atlantic/Faroe"},{"text":"Atlantic/Madeira","value":"Atlantic/Madeira"},{"text":"Atlantic/Reykjavik","value":"Atlantic/Reykjavik"},{"text":"Atlantic/South_Georgia","value":"Atlantic/South_Georgia"},{"text":"Atlantic/St_Helena","value":"Atlantic/St_Helena"},{"text":"Atlantic/Stanley","value":"Atlantic/Stanley"},{"text":"Australia/Adelaide","value":"Australia/Adelaide"},{"text":"Australia/Brisbane","value":"Australia/Brisbane"},{"text":"Australia/Broken_Hill","value":"Australia/Broken_Hill"},{"text":"Australia/Currie","value":"Australia/Currie"},{"text":"Australia/Darwin","value":"Australia/Darwin"},{"text":"Australia/Eucla","value":"Australia/Eucla"},{"text":"Australia/Hobart","value":"Australia/Hobart"},{"text":"Australia/Lindeman","value":"Australia/Lindeman"},{"text":"Australia/Lord_Howe","value":"Australia/Lord_Howe"},{"text":"Australia/Melbourne","value":"Australia/Melbourne"},{"text":"Australia/Perth","value":"Australia/Perth"},{"text":"Australia/Sydney","value":"Australia/Sydney"},{"text":"Europe/Amsterdam","value":"Europe/Amsterdam"},{"text":"Europe/Andorra","value":"Europe/Andorra"},{"text":"Europe/Athens","value":"Europe/Athens"},{"text":"Europe/Athens","value":"Europe/Athens"},{"text":"Europe/Athens","value":"Europe/Athens"},{"text":"Europe/Belgrade","value":"Europe/Belgrade"},{"text":"Europe/Berlin","value":"Europe/Berlin"},{"text":"Europe/Bratislava","value":"Europe/Bratislava"},{"text":"Europe/Brussels","value":"Europe/Brussels"},{"text":"Europe/Bucharest","value":"Europe/Bucharest"},{"text":"Europe/Budapest","value":"Europe/Budapest"},{"text":"Europe/Busingen","value":"Europe/Busingen"},{"text":"Europe/Chisinau","value":"Europe/Chisinau"},{"text":"Europe/Copenhagen","value":"Europe/Copenhagen"},{"text":"Europe/Dublin","value":"Europe/Dublin"},{"text":"Europe/Gibraltar","value":"Europe/Gibraltar"},{"text":"Europe/Guernsey","value":"Europe/Guernsey"},{"text":"Europe/Helsinki","value":"Europe/Helsinki"},{"text":"Europe/Isle_of_Man","value":"Europe/Isle_of_Man"},{"text":"Europe/Istanbul","value":"Europe/Istanbul"},{"text":"Europe/Jersey","value":"Europe/Jersey"},{"text":"Europe/Kaliningrad","value":"Europe/Kaliningrad"},{"text":"Europe/Kiev","value":"Europe/Kiev"},{"text":"Europe/Lisbon","value":"Europe/Lisbon"},{"text":"Europe/Ljubljana","value":"Europe/Ljubljana"},{"text":"Europe/London","value":"Europe/London"},{"text":"Europe/Luxembourg","value":"Europe/Luxembourg"},{"text":"Europe/Madrid","value":"Europe/Madrid"},{"text":"Europe/Malta","value":"Europe/Malta"},{"text":"Europe/Mariehamn","value":"Europe/Mariehamn"},{"text":"Europe/Minsk","value":"Europe/Minsk"},{"text":"Europe/Monaco","value":"Europe/Monaco"},{"text":"Europe/Moscow","value":"Europe/Moscow"},{"text":"Europe/Oslo","value":"Europe/Oslo"},{"text":"Europe/Paris","value":"Europe/Paris"},{"text":"Europe/Podgorica","value":"Europe/Podgorica"},{"text":"Europe/Prague","value":"Europe/Prague"},{"text":"Europe/Riga","value":"Europe/Riga"},{"text":"Europe/Rome","value":"Europe/Rome"},{"text":"Europe/Samara","value":"Europe/Samara"},{"text":"Europe/San_Marino","value":"Europe/San_Marino"},{"text":"Europe/Sarajevo","value":"Europe/Sarajevo"},{"text":"Europe/Simferopol","value":"Europe/Simferopol"},{"text":"Europe/Skopje","value":"Europe/Skopje"},{"text":"Europe/Sofia","value":"Europe/Sofia"},{"text":"Europe/Stockholm","value":"Europe/Stockholm"},{"text":"Europe/Tallinn","value":"Europe/Tallinn"},{"text":"Europe/Tirane","value":"Europe/Tirane"},{"text":"Europe/Uzhgorod","value":"Europe/Uzhgorod"},{"text":"Europe/Vaduz","value":"Europe/Vaduz"},{"text":"Europe/Vatican","value":"Europe/Vatican"},{"text":"Europe/Vienna","value":"Europe/Vienna"},{"text":"Europe/Vilnius","value":"Europe/Vilnius"},{"text":"Europe/Volgograd","value":"Europe/Volgograd"},{"text":"Europe/Warsaw","value":"Europe/Warsaw"},{"text":"Europe/Zagreb","value":"Europe/Zagreb"},{"text":"Europe/Zaporozhye","value":"Europe/Zaporozhye"},{"text":"Europe/Zurich","value":"Europe/Zurich"},{"text":"Indian/Antananarivo","value":"Indian/Antananarivo"},{"text":"Indian/Chagos","value":"Indian/Chagos"},{"text":"Indian/Christmas","value":"Indian/Christmas"},{"text":"Indian/Cocos","value":"Indian/Cocos"},{"text":"Indian/Comoro","value":"Indian/Comoro"},{"text":"Indian/Kerguelen","value":"Indian/Kerguelen"},{"text":"Indian/Mahe","value":"Indian/Mahe"},{"text":"Indian/Maldives","value":"Indian/Maldives"},{"text":"Indian/Mauritius","value":"Indian/Mauritius"},{"text":"Indian/Mayotte","value":"Indian/Mayotte"},{"text":"Indian/Reunion","value":"Indian/Reunion"},{"text":"Pacific/Apia","value":"Pacific/Apia"},{"text":"Pacific/Auckland","value":"Pacific/Auckland"},{"text":"Pacific/Bougainville","value":"Pacific/Bougainville"},{"text":"Pacific/Chatham","value":"Pacific/Chatham"},{"text":"Pacific/Chuuk","value":"Pacific/Chuuk"},{"text":"Pacific/Easter","value":"Pacific/Easter"},{"text":"Pacific/Efate","value":"Pacific/Efate"},{"text":"Pacific/Enderbury","value":"Pacific/Enderbury"},{"text":"Pacific/Fakaofo","value":"Pacific/Fakaofo"},{"text":"Pacific/Fiji","value":"Pacific/Fiji"},{"text":"Pacific/Funafuti","value":"Pacific/Funafuti"},{"text":"Pacific/Galapagos","value":"Pacific/Galapagos"},{"text":"Pacific/Gambier","value":"Pacific/Gambier"},{"text":"Pacific/Guadalcanal","value":"Pacific/Guadalcanal"},{"text":"Pacific/Guam","value":"Pacific/Guam"},{"text":"Pacific/Honolulu","value":"Pacific/Honolulu"},{"text":"Pacific/Johnston","value":"Pacific/Johnston"},{"text":"Pacific/Kiritimati","value":"Pacific/Kiritimati"},{"text":"Pacific/Kosrae","value":"Pacific/Kosrae"},{"text":"Pacific/Kwajalein","value":"Pacific/Kwajalein"},{"text":"Pacific/Majuro","value":"Pacific/Majuro"},{"text":"Pacific/Marquesas","value":"Pacific/Marquesas"},{"text":"Pacific/Midway","value":"Pacific/Midway"},{"text":"Pacific/Nauru","value":"Pacific/Nauru"},{"text":"Pacific/Niue","value":"Pacific/Niue"},{"text":"Pacific/Norfolk","value":"Pacific/Norfolk"},{"text":"Pacific/Noumea","value":"Pacific/Noumea"},{"text":"Pacific/Pago_Pago","value":"Pacific/Pago_Pago"},{"text":"Pacific/Palau","value":"Pacific/Palau"},{"text":"Pacific/Pitcairn","value":"Pacific/Pitcairn"},{"text":"Pacific/Pohnpei","value":"Pacific/Pohnpei"},{"text":"Pacific/Port_Moresby","value":"Pacific/Port_Moresby"},{"text":"Pacific/Rarotonga","value":"Pacific/Rarotonga"},{"text":"Pacific/Saipan","value":"Pacific/Saipan"},{"text":"Pacific/Tahiti","value":"Pacific/Tahiti"},{"text":"Pacific/Tarawa","value":"Pacific/Tarawa"},{"text":"Pacific/Tongatapu","value":"Pacific/Tongatapu"},{"text":"Pacific/Wake","value":"Pacific/Wake"},{"text":"Pacific/Wallis","value":"Pacific/Wallis"}],
                casts:{
                	'ECCADAD408FE6583C0FE4E6F4B11B81C':'ManagementServer',
                	'AC4EE14D3673E5E6BE1BAB7BB4C9C4D6':'Port',
                	'AFAEB0479A4BDE162442BE1EF54AD1C7':'SSLEnable',
                	'E412ABA1481A8E835C32038046800D9A':'CloudKey',
                	'A88739CA691780793CAF2AFF589EA9BC':'CloudEPC',
                	'A840FF9AE6B2A29D9A992DED99019AAB':'PLMNID',
                	'2EEB94E36A9EBF3DD3FD93193F9CAB2A':'TAC',
                	'59F039F3736E27776044BF45C1BA4F7E':'MMEIPOld',
                	'3302861245940B980E261FCB2FDB7332':'MMEIPNEW',
                	'E21B092B2657102DB925F0AE55DFA4C8':'S1LinkPort',
                	'53EC2D3968A5A33AA35C4ABE363CC228':'S1ConnectionMode',
                	'8E3B94ACC719281B41AF1B242E056A90':'HaloBEnable',
                	'F5B323783D32A2D3E3D3EFCE20932054':'HaloBMode',
                	'507E0B6496203811EAAAF21F3DE50317':'L2TunnelEnable',
                	'5C6341923949F5748115E58111BF8688':'L2TunnelIP',
                	'EE9A9F01681B998FDE05E4DC1165DA14':'L2TunnelMode',
                	/* apn1 */
                	'D63F65CF79C93572290E6CE925FF1A87': 'apnName',
                    '770C91E92CA8E01FC7C693B3182AD8BC': 'vxlanApnName',
                    '911B9FA23BEBEED4E7705BFB62A84853': 'defaultApn',
                    '84FD894F974ECE4B72BAE0BEC6D29247': 'greType',
                    'DA26528FA0E637C35B102CED66B1DAA1': 'vlanId',
                    'A7DCF2B1381F66CD8CADA1AB303C9A34': 'vlanId2',
                    /* apn2 */
                    '6988D3BF91AAFBE192B6E0DF78D37D30': 'apnName',
                    '3C992766E5734E9FE46CAF2216D7F4CA': 'vxlanApnName',
                    '1B3D8A865842343E57EC48194B10EF6D': 'defaultApn',
                    'B64BA58CEB25E674153FF360F938A8AD': 'greType',
                    'CBE3FB1BD56FC5AAB7682B1BFDC12414': 'vlanId',
                    '809B51072E9D3E67871656B9418265B8': 'vlanId2',
                    /* apn3 */
                    '8DA189FF34DF9B420F29858A5F5089BB': 'apnName',
                    '74755F527BFFE4FE04450E0A6773D11B': 'vxlanApnName',
                    '4FDAE87E2B8E1F1A5779789D05F9DD4E': 'defaultApn',
                    'B7847CD164E55FA3A6A81BD9F5391CC1': 'greType',
                    'F3CCBE226BB22845E12D7B9FA70B14BB': 'vlanId',
                    '0EC24764287C712877C814F775F87592': 'vlanId2',
                   /*  apn4 */
                    'F60AF8F751142160EDEEDB41331B8156': 'apnName',
                    '6E3E9F3C1022F50BA2614BC9A749A110': 'vxlanApnName',
                    '3E61F41456D50DBAF21987E285A38CCE': 'defaultApn',
                    'C15D12F94ECCBC4DE7FE8213326E2173': 'greType',
                    'D5322DFB943C7525FEDDFF3EB3D776BA': 'vlanId',
                    'DA026FCD9739324226CB51F2E7C95962': 'vlanId2',
                   /*  LGW */
                    '244E430B906E9C7F7E7FD3BFFCFD8D16':'lgwEnable',
                    '98124B268F87F77FAE6BD793DFCB1A27':'lgwMode',
                    '2EA3E870894AD798F5B48B522FD91943':'InterfaceBinding',
                    '1E97C4EA7B37B7AF0B351FB6E77AF6DC':'IPPOOL',
                    '28D05A824491588E6329FE950507BB32':'IPPOOLNetmask',
                    '879333711F0FE7658568983276EBA604':'LGWStaticIPAddr_Enable',
                    '03E2D0F56CA44E3F10FAA222C1EC6243':'LGWFirstStaticIPAddr',
                    '8C383EEDE84A39BB028058EEAB74BC12':'LGWLastStaticIPAddr',
					'078011BAAECEA822C10537443569C6D5':'IMSItoIPBindingRange',
                   /*  ntp */
                   'D06698AAEA84E3CE35D4BAFB3C02EC57':'TimeZone',
                   '14704E8786975361C235270E3F2B8B3A':'ntpEnable',
                   '35CC6BE59F39EBA58CC023042465690D':'Port1',
                   'CB314F892ACC0AF85F97E55CCCA11F74':'Server1',
                   '3DFECB40351ABF4367DFAC7BD1BD4F0F':'Port2',
                   '23961C5E348A03B2F48AC64770F262B7':'Server2',
                   '42EEA4D0C14613C4D4ABAEE30A578E4A':'Port3',
                   '6BE4E8FB3866FB34C610128482D17AB2':'Server3',
                   /* sync */
                   '318F168036707EFAE6B9F14ED15056E8':'syncList',
                   '6D02B91942E773AF882D27C1D47C5A52':'Index',
                   '3016B311861F51BEB5DCB6EDD3C21968':'priority',
                   '683137319A8FA38CFF7E78D98B8DFB68':'technology',
                   '6B080E96D33733D3A7B19587B4BC32C2':'Band',
                   'B5C20E81B364493B126618A6B92688E7':'ChannelNumber',
                   '530D1995AB185ADDA5B96A66250494C5':'PCI',
                   '283FE1C5A08C276D09ADD8385AC55590':'freqUncertaintyThreshold',
                   '0BD93DBBA5D87BF78145AD82D2C8E36D':'syncInterval',
                   'B5B5A353D95630E9801424B455EEB9CB':'phaseOffset',

				   'FA3D6078A3B18B26B9A4F3478E75E795':'syncMode',
				   '49FAF2D896FB1E19352527D1F23A9265':'syncType',
				   'E87FBBFF3FAD968D62786A49E7C99BDE':'syncTrigger',
				   'A71E86E6C90A0B7D3E3E87A59173FD52':'syncInterface',
				   '1E2F5D32F858BA7E05A469995D2696D2':'syncPort',
				   '2F352489D0D1773D5277A3F20EEA6699':'syncAddress',
				   '37F388797734902A41BAC3269F411204': 'profile',
				   'DA391D4BC04780AE0357DB232F676E9A': 'domain',
				   
				    '5BAB3DC7737F3DE79D541F70DCEFCE09':'ManagementServer',
                    '7179C88D4EB44C3A70795E07A65B3410':'Port',
                    '078EB16E564B4CE3F551C1328287CDAA':'SSLEnable',
                    '3F40B1D602EE6A524F706B18FF39089D':'CloudKey',
                    '4317FFD20C9DEC4EE3AA7507C88608CE':'CloudEPC',
                    '5F22A7A192AF7381C3EF2D54FD5BCEA2':'PLMNID',
                    '092E770B3B4E6504BFF9BFA72C0DAC66':'TAC',
                    '0270260D06962A9262770A4CE881DC89':'MMEIPOld',
                    '6D9C3406AD9E3F46A91D26192B92B65C':'MMEIPNEW',
                    'A42FF7B915D116FC85933DC9AFDA8109':'S1LinkPort',
                    '2DB9AF50455FCD81241395DCB1D6D677':'S1ConnectionMode',
                    'A90578CA6D8E7D13CDF9BFB111D27F52':'HaloBEnable',
                    '91689DB996D8E186592045F4367F614A':'HaloBMode',
                    '157799BEA466063F70AA54EA2F731A88':'L2TunnelEnable',
                    'CE237E6B920E610D354CBA2B26F9D45F':'L2TunnelIP',
                    '746D0B95E08303E9BCA734749DA4DF60':'L2TunnelMode',
                    /* apn1 */
                    '01010ED65CB95F1B8DF5678323FE7C00': 'apnName',
                    '770C91E92CA8E01FC7C693B3182AD8BC': 'vxlanApnName',
                    '213685A9160B33A9BD9AE77EB6EC2323': 'defaultApn',
                    'BD39A324569380E098F15679D8171824': 'greType',
                    'E84E6544686CA69395679A1963856D2A': 'vlanId',
                    '55476CBC7725D9B2640FD01E4CB4C281': 'vlanId2',
                    /* apn2 */
                    '768FFC70E65F7E3B9FAAE0A0086FED03': 'apnName',
                    '3C992766E5734E9FE46CAF2216D7F4CA': 'vxlanApnName',
                    'D80FD04D797A6B79638C2907A7166F4C': 'defaultApn',
                    'B8A49D14A39C344BCC2568DA66C2049F': 'greType',
                    'A864CFA1DAE120CECB468C503B88D231': 'vlanId',
                    '9CEA5BF2D8CBD4E5964DD59903B6C508': 'vlanId2',
                    /* apn3 */
                    '23310D65B00726CA4D3780D0124ABFE3': 'apnName',
                    '74755F527BFFE4FE04450E0A6773D11B': 'vxlanApnName',
                    '1046ACF4EB6D45C5EAF1774D819041D8': 'defaultApn',
                    '530CE138AB9624C4E7364F6FA1DB06CE': 'greType',
                    '64E2DA361E721CD344CA094336CAED0F': 'vlanId',
                    'C9323649520C437467073B33CF3F1BCE': 'vlanId2',
                   /*  apn4 */
                    'B41318900504A5FA1976DE3B1B16983A': 'apnName',
                    '6E3E9F3C1022F50BA2614BC9A749A110': 'vxlanApnName',
                    '6AA63425DBC4A5E3763A2EFF1BA8DAD4': 'defaultApn',
                    '625C6E2CFC7BE539EFCE81BA8BDEA425': 'greType',
                    'BB1A5E0393661CC84357080F21ADF439': 'vlanId',
                    '42D857579A42A9AFBE573576DC9A0334': 'vlanId2',
                   /*  LGW */
                    'BE75D426DAB82EFA27D2866D187C1B6F':'lgwEnable',
                    '8096B50627807C8790741A4EECBB53B0':'lgwMode',
                    '916C085A9D59B9B4792F7A2694602D27':'InterfaceBinding',
                    '241E90D858C63727C28602CC735BB002':'IPPOOL',
                    '38C1AFBE2CAD19E9CB6A25E867C4C963':'IPPOOLNetmask',
                    '329F34C3FFC4B2A92C7F0ADA5C413068':'LGWStaticIPAddr_Enable',
                    '389C677AC7638D20C7A2E48EF425DE89':'LGWFirstStaticIPAddr',
                    '015472DC4BE3B49397A4BD511DCA49C5':'LGWLastStaticIPAddr',
					'255A398A150604AB2B7D2424E04BAD20':'IMSItoIPBindingRange',
                   /*  ntp */
                   '01B16B8BD00766A3B10AE2AA37A4A77E':'TimeZone',
                   'D3B04893B615937A5E9F39458FA36321':'ntpEnable',
                   '3D72507346A0018AD7DB9B1F07166D6F':'Port1',
                   'F1EAD0BF357AF4BA4D02429B488241F5':'Server1',
                   '3DFECB40351ABF4367DFAC7BD1BD4F0F':'Port2',
                   '062B0BAC7BC030FDBD6818A83F5D1846':'Server2',
                   '42EEA4D0C14613C4D4ABAEE30A578E4A':'Port3',
                   'D9E53E4D68C0AE206DC4E4D13A2130D3':'Server3',
                   /* sync */
                   '3297DEA47AFACCD8AC595C9AAA70B44A':'syncList',
                   'C134AA63F8601B9A73F1FBB75F2A715C':'Index',
                   'C5BB1BF2B17130D68B2CA6B976A5DF31':'priority',
                   'CFC111538E0DA2D71EDE19768A3FBEFB':'technology',
                   'F491D48D550A5C607154A4755A18EDA4':'Band',
                   '1BCF440C4A6DAB80DBDA9FBE0E57B454':'ChannelNumber',
                   '6DED0BE145C91A7EAA93EFB798219E39':'PCI',
                   '19E74A57CFBF61873F383F5ADA7D7C4D':'freqUncertaintyThreshold',
                   '5BAD13C0BE117FA0EFC9B26B07468EAA':'syncInterval',
                   '8E51E7224860D349507E7782C83D2C1D':'phaseOffset',

				   'C86E924A77744E62468B8F92AF7BC8EF':'syncMode',
				   'E25F3F4A56D8319AA5A46FDBE45F952C':'syncType',
				   '5236CD2B7812A1B51C1194940C82A84C':'syncTrigger',
				   'EA12A57D0FBAB2D9491EB6FEFFD9895A':'syncInterface',
				   'C791C1042A2F34262EB7A1ADEDB83A7A':'syncPort',
				   '5C417E64987A52180EB7822BC424AFB4':'syncAddress',
                   'BBDB1E414AE3968DE3028E3E67EDEEF3': 'profile',
                   'B014F39FFD43F0426FC375C291FDB26D': 'domain',

				   	'53FA74428B673F4F1B9B1C32147CF668':'ManagementServer',
                    'BC8EE4AC0832AE12B5DD853CCD74017B':'Port',
                    'FB20B658A4844D040C507A745C83D028':'SSLEnable',
                    '09CA27D4CA3C50D034D70DFECE405359':'CloudKey',
                    'DA0A8F40C58C38E4F5ACEA400B0BDF79':'CloudEPC',
                    '083D31CEAC8DA0F1138B586843A1C292':'PLMNID',
                    '09BD05CCE3520882A54ED223A39770E6':'TAC',
                    'F1C9614E593F9E1D9FBD9F63E3FF7964':'MMEIPOld',
                    '5B5060AAF5421BB35E1E531202235140':'MMEIPNEW',
                    '49874E6CBE00A49AAAD28C29588920DD':'S1LinkPort',
                    '591AA3B9D87371C11980A7909FED2B98':'S1ConnectionMode',
                    'BDA2883E2911EAAAF56D93FC0E0B07BB':'HaloBEnable',
                    'B8A99B791315C8D8DB90A3B1FFFDB0CE':'HaloBMode',
                    'BBB167F09CB7F4C82220A8AF1CDFB7CA':'L2TunnelEnable',
                    '71B35231FEAC996FF700A4BBD661AC5C':'L2TunnelIP',
                    '828F85566B38D738561D230AA8967A76':'L2TunnelMode',
                    /* apn1 */
                    'DFE0209C782A8B1CCBAB316D80AB9394': 'apnName',
                    '770C91E92CA8E01FC7C693B3182AD8BC': 'vxlanApnName',
                    '4A7644D88899779D3206745F4041E75F': 'defaultApn',
                    '9B5DCB0D85462510E7F1BC1381550911': 'greType',
                    '42061B93DD3BA8C7AA6DB830BFE2E7AF': 'vlanId',
                    '52F463E13D4C2635CBF450FDD133745A': 'vlanId2',
                    /* apn2 */
                    '8938575EDBE58DA1D546D68A61871C39': 'apnName',
                    '3C992766E5734E9FE46CAF2216D7F4CA': 'vxlanApnName',
                    '6D9FC4EB60F9922ED9063C5A6A6556F0': 'defaultApn',
                    '00D1E4AF6C00F65EBE819B0F3E584CB3': 'greType',
                    'E5224B6FC84139FF37F526708F43E76C': 'vlanId',
                    '9B06A170289FB80B568A5EAF48D65459': 'vlanId2',
                    /* apn3 */
                    '0021150D83CC4B755E9410DACB27C524': 'apnName',
                    '74755F527BFFE4FE04450E0A6773D11B': 'vxlanApnName',
                    '0C83B0C72102027427DE2B0762120F34': 'defaultApn',
                    '39595B82E09E82003CC0D244F223E0F1': 'greType',
                    '81D8A6EB426CCE7096C114594E99A7EA': 'vlanId',
                    '8C313BF2C19F3473843E0B6A02E99BCD': 'vlanId2',
                   /*  apn4 */
                    'D49ADFE960695AABE02B12FA0D3BF380': 'apnName',
                    '6E3E9F3C1022F50BA2614BC9A749A110': 'vxlanApnName',
                    '1906879F52BFB3E58655A6A476024F55': 'defaultApn',
                    '581A30D7E7FC5B3D3C93C76D985EB833': 'greType',
                    '9CB077E041F5750465273BE71FFD4C9E': 'vlanId',
                    'B338A53F3481552E2D2A31DF58C6A03D': 'vlanId2',
                   /*  LGW */
                    '4662C1921EB43818A28CBDF7037BB781':'lgwEnable',
                    '777A2A5CF2EE4B8A26082B3D83804E92':'lgwMode',
                    'ACD10EBE9FBF78AC7A15E27F6129C396':'InterfaceBinding',
                    'C6FB6A753B457B65118AD7F38E24CB4D':'IPPOOL',
                    'A94868ABB719C1E70D509CD9DE29396B':'IPPOOLNetmask',
                    '0A6A9AD26198D7E408BBF2F6177C51C0':'LGWStaticIPAddr_Enable',
                    '1EA86791CFC945AE744D60C9B5653822':'LGWFirstStaticIPAddr',
                    '3B64DDFE02286F4A40F6AB5DC5EEA94B':'LGWLastStaticIPAddr',
					'DE7ACA533DEAEE95DCA67049F25AD498':'IMSItoIPBindingRange',
                   /*  ntp */
                   '8A380A37ED8687F134438D21F795C1D0':'TimeZone',
                   '4A3678677217E7D4B4686BD66E93FBEA':'ntpEnable',
                   'F28534D83D11969C705BEAA46E0F0E1E':'Port1',
                   'BD9AFF6BAD323996D66BDCCC5D1CB288':'Server1',
                   '3DFECB40351ABF4367DFAC7BD1BD4F0F':'Port2',
                   '8F952F69CBE7E0B833AF1580D3B9BFB4':'Server2',
                   '42EEA4D0C14613C4D4ABAEE30A578E4A':'Port3',
                   '607C2C00F7BD7413E227DA30D5A80689':'Server3',
                   /* sync */
                   'EF37B39226C1A9C24A1B2AF48C1AC8FE':'syncList',
                   '17480BA0E93467E65BAFADAA19DF2C2C':'Index',
                   '5DB985058250133647114290DF8238B3':'priority',
                   '6FEA8403BDC3465AA276D533504DFF1A':'technology',
                   '3D2E29D000EE203DC9E98CBFA25201D7':'Band',
                   '730497BDF1E3ED01C428463CD71C8C69':'ChannelNumber',
                   'D66F5B5E15C2C3DE869E8B6C6F64349E':'PCI',
                   '85718459E9676B085F67A25D5ADFC283':'freqUncertaintyThreshold',
                   '6DAD5E11BEAC0E949FE212F73F455BBD':'syncInterval',
                   'F47E00EA0990FC0A8523358B39990165':'phaseOffset',

				   '1CFFD352217405952C4BDB6CBA73A913':'syncMode',
				   '20A30A81FD5D109DF5FD31BAF8D540B4':'syncType',
				   'AAFFC7449D2B116E04A58FDF64CB5A6B':'syncTrigger',
				   '75B1C672504A7DC02996218B816B2542':'syncInterface',
				   '0575403E36793A78E02BACA8345A1ED9':'syncPort',
				   'D8044D4A70B3BF6EBEB98C1424CC936C':'syncAddress',
                   '1BA6003080EE8DBE4828EF8D62A9AC7A': 'profile',
                   'EECC01920CC2EE4C0E86A6D3B631E2C2': 'domain',

				   '803E94F9CB7469C2170F6399E7E12ADC': 'TR069Binding',
				   '7A646842BC73F98C4AD7B4CE073D2F04':'HaloBEnable',
				   'FFB74C7C641B574D65B2EC71ADE93ADA':'S1CBinding',
				   'C8BFAD8C5481B980161F20D83D209B25':'SGWSwitch',
				   'EBC35B43E36CDCEE993B35B10309917C':'SGWInterfaceBinding',
				   'D9DF79FD910CB3D6E43EABEA448119F0':'HaloDMode',
				   '72DC89171F2BF3832CAAD1F51895CA3A':'HaloDDestinationAddress',
				   'C42CB9993C7693FAC5ABFB1F3EC74E16':'WCGIFName',
				   '7BDEEF3FBC84630ADFEF7C8B1B8431DD':'S1CBinding',
				   '19FB2984E7E6442018BD78A7DC6BBB09':'WCGUIFName',
				   '7BDEEF3FBC84630ADFEF7C8B1B8431DD':'MMEPort',
				   'FF1931B04264955C3829AB7107D322CE':'slave1IP',
				   'DD37CCC7A03C49A264E416E9D25A57C5':'slave2IP',
				   
					// MLQ
				   '1CB5D2945BFD08F47D7A8E13A88B5231':'HaloDDestinationAddress',
				   '5386FBC9B133A06EDB7B84858819C299':'HaloDMode',

				   // o6
				   '01A037795F64E6084A24A55AD2E4D74F': 'slave1IP',
				   '05B70C57B0C60211A11DF71229C7D695': 'Port1',
				   '076C0B8A86A7F553F4F145CC65D2073B': 'HaloBEnable',
				   '08E3FC593F3108F338DA8E7D2BBCBFAF': 'MMEIPNEW',
				   '0DA5795C2E0ADED819F7D9535259009D': 'HaloDDestinationAddress',
				   '192F2027AC9B9C9A7CCBC82D9CCB1B08': 'SSLEnable',
				   '1D2E0212F1865B69D4D728BA05B4619C': 'TR069Binding',
				   '24A05AB90D694A66F67EDD9A6D49E2F8': 'S1ConnectionMode',
				   '31C56ACBF6152919BB66E080348674A8': 'Server2',
				   '49D2CE644A525875327BDC07F3C5DBDE': 'MMEPort',
				   '4E0A33C3194D2914CBC9DD387900AFDA': 'LGWFirstStaticIPAddr',
				   '4E469102659C08A2A1D2F556D80D9304': 'LGWStaticIPAddr_Enable',
				   '522918424BB2242F6150EDA03D26E3FB': 'TAC',
				   '53E6FCC78B3AE24C3266DE3AB774CCF6': 'CloudKey',
				   '5BF67AF3A0CB3D0B7CC4A25A1923CCE9': 'SGWInterfaceBinding',
				   '67453CFAEE57FCAAE069B29A7D69B10D': 'slave2IP',
				   '68D4D7F75636FC6070A5CF2965C636BE': 'InterfaceBinding',
				   '7192F22C4573C00FF726802266CC0824': 'CloudEPC',
				   '821A98B05487440F47E7A9D157343AAC': 'TimeZone',
				   '829FD22E7B47F71DFAD1E15F3CBA2BCB': 'lgwMode',
				   '8540D977DE69EC9B205A1AD3A648527C': 'lgwEnable',
				   '89FE23775FE2E7F59D30683829E6CE31': 'HaloDMode',
				   '8A2AD6875751B124C0A0D8856E60B1F1': 'IPPOOL',
				   '952682D72730E26226A2490944E12426': 'ManagementServer',
				   '976A8E8A2F425506D29B509CDAE6D97F': 'WCGUIFName',
				   'A056967D33DFDF1150A64C3D96A1C1D5': 'Port',
				   'A0AD93869B5976A6DEF4FA57B1771E25': 'ntpEnable',
				   'A45A54C6EF15C1CC3807D526CDE46B29': 'HaloBMode',
				   'AF554F6AD1A07BFBA94FC8BA4CF55ED9': 'IMSItoIPBindingRange',
				   'C4384D985C357E97AE25E1C4B175BC27': 'HaloBEnable',
				   'C4DDC10E4E3106F8B227F6F9553218B5': 'Satellites', // 待定
				   'CB2B9F2378FAEA2607F81EBEDB66C4E8': 'IPPOOLNetmask',
				   'CEDC6DB5BBB9C600DB1ECAB38A62F2F5': 'PLMNID',
				   'D0B46E183A18390534B016DB84CB657D': 'S1LinkPort',
				   'D808027EEE71E8B26C7A066229F6A616': 'Server3',
				   'D93E34CB2FD9D609D0258F8C41087B82': 'Server1',
				   'D9C62855224200614491EC1507C27C88': 'LGWLastStaticIPAddr',
				   'E459CE45841630B0AD14260AD831EE69': 'SGWSwitch',
				   'EA9F5EE17E3267DB9BFD8D3EC26E33A8': 'WCGIFName',
				   'F157CB9B164E63BDE2B692DB7DE22C6E': 'S1CBinding', // 待定
				   'CCDC657A5E454D671F11E744727EDD02': 'syncMode',

				   '5838E68A971A4EED781C8E98B77B8DC3': 'multiS1Mode',
				   '155A22DB2D757894DBADEE2B3EACBBB5': 'BindPLMNID',
				   '4997E74C2B6499D65F64C5ECD8C76860': 'MMEBindList',
				   '4EDD8C0FF8DADD16A67F9613BBB00F68': 'S1UIp',
				   '60B9A4071A030CE129549DD84F311DFC': 'S1UIfname',
				   '6C30AFD464EADE81CB108FBD442C04D6': 'MMEIp',
				   'E03CFB75AC3D6C9F200CEEC6B37806F1': 'S1CIp',
				   'E0AD3B6990128F2C7DC5BC97C8A512E0': 'S1CIfname',
				   '133036E6CF1898BCB32BE64188AE01C4': 'MMEstatus',
				   'DBB2CB1DB60432E1348209DFC7EBF90F': 'BindIndex',
                },
                indexs: {
                    '06F5E262AF5D8CE03B57F328336C3DC6': 0,
                    'F7DF9764D49942C14CB347C9C7BC113C': 0,
                    '7A3310F1421B8AA8EFA38F2EABD807B3': 0,
                    'F12D8419AC33436AD6D7E39552C9D66D': 0,
                    'D63F65CF79C93572290E6CE925FF1A87': 0,
                    '911B9FA23BEBEED4E7705BFB62A84853': 0,
                    '770C91E92CA8E01FC7C693B3182AD8BC': 0,
                    'DA26528FA0E637C35B102CED66B1DAA1': 0,
                    'FB23F7673381B3DD99B1F8E09C571850': 0,
                    '84FD894F974ECE4B72BAE0BEC6D29247': 0,
                    '19B7E34710EF7DC38F32F989A92AECD3': 0,
                    'A7DCF2B1381F66CD8CADA1AB303C9A34': 0,

                    '19E71941C9591DD8C6737EACF4B4A56C': 1,
                    'A486643C6373FDEFF609C59B8C5D49B7': 1,
                    'C966B74B2983C3AB20D5CDE94AD6473A': 1,
                    '8303553D2DA7EDBE8451BC97913421F8': 1,
                    '6988D3BF91AAFBE192B6E0DF78D37D30': 1,
                    '1B3D8A865842343E57EC48194B10EF6D': 1,
                    '3C992766E5734E9FE46CAF2216D7F4CA': 1,
                    'CBE3FB1BD56FC5AAB7682B1BFDC12414': 1,
                    '450FD510DAECC521296E9DEDD3F98064': 1,
                    'B64BA58CEB25E674153FF360F938A8AD': 1,
                    '02760609C9925C0F6ECD4B8D1A03E595': 1,
                    '809B51072E9D3E67871656B9418265B8': 1,

                    '7EB0CE4CEEFEF97861B472356B72E953': 2,
                    '76D5556ADFE3E86B9F7910268F95C699': 2,
                    '429D90F8CA2C8CC94A97561ECCC6BCDB': 2,
                    'CD5740D4D032059B85373EA25165D464': 2,
                    '8DA189FF34DF9B420F29858A5F5089BB': 2,
                    '4FDAE87E2B8E1F1A5779789D05F9DD4E': 2,
                    '74755F527BFFE4FE04450E0A6773D11B': 2,
                    'F3CCBE226BB22845E12D7B9FA70B14BB': 2,
                    'FBB462CBAF86F92865A0C59AB37F5014': 2,
                    'B7847CD164E55FA3A6A81BD9F5391CC1': 2,
                    '8BFCC9D991C57E661953E57F7846932C': 2,
                    '0EC24764287C712877C814F775F87592': 2,

                    '9AC087BCB14904BC29BE3A981DF745EA': 3,
                    '31629DAE1D2D6CD5262697A503880B47': 3,
                    'B6F679BFF85E536C4FD9D1A96E89181D': 3,
                    '565AA673367B5A4EE8AD1D9485A04B57': 3,
                    'F60AF8F751142160EDEEDB41331B8156': 3,
                    '3E61F41456D50DBAF21987E285A38CCE': 3,
                    '6E3E9F3C1022F50BA2614BC9A749A110': 3,
                    'D5322DFB943C7525FEDDFF3EB3D776BA': 3,
                    '200D90857C0106AB9CA19E943EB3D7D9': 3,
                    'C15D12F94ECCBC4DE7FE8213326E2173': 3,
                    'FB29B8F11A05DC8D8AD5C77938F63C8C': 3,
                    'DA026FCD9739324226CB51F2E7C95962': 3,
					
					
					'06F5E262AF5D8CE03B57F328336C3DC6': 0,
                    'F7DF9764D49942C14CB347C9C7BC113C': 0,
                    '7A3310F1421B8AA8EFA38F2EABD807B3': 0,
                    'F12D8419AC33436AD6D7E39552C9D66D': 0,
                    '01010ED65CB95F1B8DF5678323FE7C00': 0,
                    '911B9FA23BEBEED4E7705BFB62A84853': 0,
                    '213685A9160B33A9BD9AE77EB6EC2323': 0,
                    'E84E6544686CA69395679A1963856D2A': 0,
                    'FB23F7673381B3DD99B1F8E09C571850': 0,
                    'BD39A324569380E098F15679D8171824': 0,
                    '19B7E34710EF7DC38F32F989A92AECD3': 0,
                    '55476CBC7725D9B2640FD01E4CB4C281': 0,

                    '19E71941C9591DD8C6737EACF4B4A56C': 1,
                    'A486643C6373FDEFF609C59B8C5D49B7': 1,
                    'C966B74B2983C3AB20D5CDE94AD6473A': 1,
                    '8303553D2DA7EDBE8451BC97913421F8': 1,
                    '768FFC70E65F7E3B9FAAE0A0086FED03': 1,
                    'D80FD04D797A6B79638C2907A7166F4C': 1,
                    '3C992766E5734E9FE46CAF2216D7F4CA': 1,
                    'A864CFA1DAE120CECB468C503B88D231': 1,
                    '450FD510DAECC521296E9DEDD3F98064': 1,
                    'B8A49D14A39C344BCC2568DA66C2049F': 1,
                    '02760609C9925C0F6ECD4B8D1A03E595': 1,
                    '9CEA5BF2D8CBD4E5964DD59903B6C508': 1,

                    '7EB0CE4CEEFEF97861B472356B72E953': 2,
                    '76D5556ADFE3E86B9F7910268F95C699': 2,
                    '429D90F8CA2C8CC94A97561ECCC6BCDB': 2,
                    'CD5740D4D032059B85373EA25165D464': 2,
                    '23310D65B00726CA4D3780D0124ABFE3': 2,
                    '1046ACF4EB6D45C5EAF1774D819041D8': 2,
                    '74755F527BFFE4FE04450E0A6773D11B': 2,
                    '64E2DA361E721CD344CA094336CAED0F': 2,
                    'FBB462CBAF86F92865A0C59AB37F5014': 2,
                    '530CE138AB9624C4E7364F6FA1DB06CE': 2,
                    '8BFCC9D991C57E661953E57F7846932C': 2,
                    'C9323649520C437467073B33CF3F1BCE': 2,

                    '9AC087BCB14904BC29BE3A981DF745EA': 3,
                    '31629DAE1D2D6CD5262697A503880B47': 3,
                    'B6F679BFF85E536C4FD9D1A96E89181D': 3,
                    '565AA673367B5A4EE8AD1D9485A04B57': 3,
                    'B41318900504A5FA1976DE3B1B16983A': 3,
                    '6AA63425DBC4A5E3763A2EFF1BA8DAD4': 3,
                    '6E3E9F3C1022F50BA2614BC9A749A110': 3,
                    'BB1A5E0393661CC84357080F21ADF439': 3,
                    '200D90857C0106AB9CA19E943EB3D7D9': 3,
                    '625C6E2CFC7BE539EFCE81BA8BDEA425': 3,
                    'FB29B8F11A05DC8D8AD5C77938F63C8C': 3,
                    '42D857579A42A9AFBE573576DC9A0334': 3,
					
					'06F5E262AF5D8CE03B57F328336C3DC6': 0,
                    'F7DF9764D49942C14CB347C9C7BC113C': 0,
                    '7A3310F1421B8AA8EFA38F2EABD807B3': 0,
                    'F12D8419AC33436AD6D7E39552C9D66D': 0,
                    'DFE0209C782A8B1CCBAB316D80AB9394': 0,
                    '4A7644D88899779D3206745F4041E75F': 0,
                    '4A7644D88899779D3206745F4041E75F': 0,
                    '42061B93DD3BA8C7AA6DB830BFE2E7AF': 0,
                    'FB23F7673381B3DD99B1F8E09C571850': 0,
                    '9B5DCB0D85462510E7F1BC1381550911': 0,
                    '19B7E34710EF7DC38F32F989A92AECD3': 0,
                    '52F463E13D4C2635CBF450FDD133745A': 0,

                    '19E71941C9591DD8C6737EACF4B4A56C': 1,
                    'A486643C6373FDEFF609C59B8C5D49B7': 1,
                    'C966B74B2983C3AB20D5CDE94AD6473A': 1,
                    '8303553D2DA7EDBE8451BC97913421F8': 1,
                    '8938575EDBE58DA1D546D68A61871C39': 1,
                    '6D9FC4EB60F9922ED9063C5A6A6556F0': 1,
                    '3C992766E5734E9FE46CAF2216D7F4CA': 1,
                    'E5224B6FC84139FF37F526708F43E76C': 1,
                    '450FD510DAECC521296E9DEDD3F98064': 1,
                    '00D1E4AF6C00F65EBE819B0F3E584CB3': 1,
                    '02760609C9925C0F6ECD4B8D1A03E595': 1,
                    '9B06A170289FB80B568A5EAF48D65459': 1,

                    '7EB0CE4CEEFEF97861B472356B72E953': 2,
                    '76D5556ADFE3E86B9F7910268F95C699': 2,
                    '429D90F8CA2C8CC94A97561ECCC6BCDB': 2,
                    'CD5740D4D032059B85373EA25165D464': 2,
                    '0021150D83CC4B755E9410DACB27C524': 2,
                    '0C83B0C72102027427DE2B0762120F34': 2,
                    '74755F527BFFE4FE04450E0A6773D11B': 2,
                    '81D8A6EB426CCE7096C114594E99A7EA': 2,
                    'FBB462CBAF86F92865A0C59AB37F5014': 2,
                    '39595B82E09E82003CC0D244F223E0F1': 2,
                    '8BFCC9D991C57E661953E57F7846932C': 2,
                    '8C313BF2C19F3473843E0B6A02E99BCD': 2,

                    '9AC087BCB14904BC29BE3A981DF745EA': 3,
                    '31629DAE1D2D6CD5262697A503880B47': 3,
                    'B6F679BFF85E536C4FD9D1A96E89181D': 3,
                    '565AA673367B5A4EE8AD1D9485A04B57': 3,
                    'D49ADFE960695AABE02B12FA0D3BF380': 3,
                    '1906879F52BFB3E58655A6A476024F55': 3,
                    '6E3E9F3C1022F50BA2614BC9A749A110': 3,
                    '9CB077E041F5750465273BE71FFD4C9E': 3,
                    '200D90857C0106AB9CA19E943EB3D7D9': 3,
                    '581A30D7E7FC5B3D3C93C76D985EB833': 3,
                    'FB29B8F11A05DC8D8AD5C77938F63C8C': 3,
                    'B338A53F3481552E2D2A31DF58C6A03D': 3
                },
                operType:''
			}
		},
		computed: {
			IFNameMap() {
				var vm = this,
					nameMap = {};

				vm.wanIfNameList.map(function(item) {
					nameMap[item.value] = item.name;
				})

				vm.ipsecIfNameList.map(function(item) {
					nameMap[item.name] = item.name;
				})

				return nameMap;
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
				vm.getParamNode(code,id);
				// 获取IFName数据
				vm.getMMEBindingSelectParams();
			},
			loadTableData(code) {
				var vm = this,
					url = '${ctx}/cell/quicksettings/getListParamValue.action';

				var vm = this,
					params = {
						parent_id: 10071,
						platform: 'QA_436Q',
						smallCellCode: code
					};

            	axios.post(url,stringify(params)).then(res=>{
            		var data = res.data;

            		if(data.rows){
            			data.rows.map((item, idx)=>{
            				vm.setListValue(item);
            			})
            		}

					initForm(vm.$refs.btsForm);
            	});
			},
            setListValue(item,type) {
            	var vm = this;

	            var obj = {};

				for(key in item) {
					var prop = vm.casts[key];

	            	obj[prop] = item[key];
				}

	            vm.btsForm.syncList.push(obj);
            },
            selectPolicy() {
                var vm = this;

                vm.visible = true;
            },
            setMMEListValue(item,type) {
            	var vm = this;

	            var obj = {};

				for(key in item) {
					var prop = vm.casts[key];

	            	obj[prop] = item[key];
				}

	            vm.btsForm.MMEBindList.push(obj);
            },
			loadMMEListData(url, code) {
				var vm = this,
					params = {
						smallCellCode: code
					};

				axios.post(url,stringify(params)).then(res=>{
            		var data = res.data;

            		if(data.rows){
            			data.rows.map((item, idx)=>{
            				vm.setMMEListValue(item);
            			})
            		}

					initForm(vm.$refs.btsForm);
            	});
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
                            	group.groups.map(function(g){
                                    g.list.map(function(m){
                                        codes.push(m.name);
                                        // 执行赋值
                                        if(m.type != 'list') vm.setValue(m);
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

											vm.rebootMap['MMEIPOld'] = true;
											vm.rebootMap['MMEIPNEW'] = true;
                                        }

										if(['NL List','NL列表'].includes(m.label)) {
											m.list.map(function(field){
												vm.codeTableList.push(field.name);
											});

											vm.loadTableData(code);
										}
										// multi MME List
										if('4997E74C2B6499D65F64C5ECD8C76860' == m.name) {
											m.list.map(function(field){
												vm.codeTableList.push(field.name);
											});

											vm.loadMMEListData(m.url, code);
										}
										
										if(['078011BAAECEA822C10537443569C6D5','255A398A150604AB2B7D2424E04BAD20','DE7ACA533DEAEE95DCA67049F25AD498'].includes(m.name) && m.value) {
											m.value.split(',').map(function(v){
												vm.bindGroup.push(v);
											});
										}

										if('F157CB9B164E63BDE2B692DB7DE22C6E' == m.name) {
											if(m.data) {
												var bandList = JSON.parse(m.data);

												vm.scbandingList = bandList;
											}
										}

										if('803E94F9CB7469C2170F6399E7E12ADC' == m.name) {
											if(m.data) {
												var bandList = JSON.parse(m.data);

												vm.bandingList = bandList;
											}
										}
										
										if('68D4D7F75636FC6070A5CF2965C636BE' == m.name) {
											if(m.data) {
												var bandList = JSON.parse(m.data);

												vm.newQuickBandList = bandList;
											}
										}
										
										if('5BF67AF3A0CB3D0B7CC4A25A1923CCE9' == m.name) {
											if(m.data) {
												var bandList = JSON.parse(m.data);

												vm.SGWInterfaceBindingList = bandList;
											}
										}
										
										if('1D2E0212F1865B69D4D728BA05B4619C' == m.name) {
											if(m.data) {
												var bandList = JSON.parse(m.data);

												vm.trBandList = bandList;
											}
										}
                                    });
									// 初始化重启项关系记录
									vm.initReboot(g.list, vm.rebootMap);
                            	})
                            });
                        });
                        
                     	// init plmnGroup
                        vm.btsForm.PLMNID.split(',').map(function(plmn){
                        	vm.plmnGroup.push(plmn);
                        });
						// 模式判定优先级：CloudEPC -> HaloBEnable -> HaloDMode，三种情况都不满足时，才为normal模式
						if(vm.btsForm.CloudEPC == '1') {// ClouEPC模式判断字段
							vm.modeSelection = 'cloudepc';
						}else if(vm.btsForm.HaloBEnable == '1') {// Halob模式判断字段
							vm.modeSelection = 'halob';
						}else if(vm.btsForm.HaloDMode != '3' && vm.isBaiblq && vm.isMLQ) {// HaloD模式判断字段 + 产品类型：halodMode可能值：0、1、2、3
							vm.modeSelection = 'halod';
						}else {
							vm.modeSelection = 'normal';
						}

						vm.$nextTick(function(){
                            initForm(vm.$refs.btsForm);
                        	initForm(vm.$refs.syncForm);
                            detectReboot(vm.$refs.btsForm, vm.rebootMap);
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
                var idx = vm.indexs[code],
                    key = vm.casts[code],
                    isApnList = [0,1,2,3].includes(idx);

                if(isApnList) {
                    
                }else {
					if(code == 'AFAEB0479A4BDE162442BE1EF54AD1C7') {
						vm.btsForm[key] = [true,'true'].includes(value)?true:false;
					}else {
                    	vm.btsForm[key] = value;
					}
                }
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
                    (vm.codeList.concat(vm.codeTableList)).map(function(name){
                        if(vm.casts[name] == prop) {
                            key = name;
                        }
                    });
                }

                return key;
            },
            hasKey(key) {
                var vm = this,
                    has = false;

                vm.codeList.map(function(name){
                    if(vm.casts[name] == key) has = true;
                });

                return has;
            },
			isNull(val){
                if(val==undefined || val == null || val =="") return true;
                else return false;
            },
            closeDialog() {
                this.visible = false;
            },
            selectPolicy() {
                var vm = this;

                vm.visible = true;
            },
			addPlmn(){
            	var value = this.plmnVal;
            	var reg = /^(\d+\.\.){0,1}(\d+)$/;
            	if(value == '' || !reg.test(value) || value.length < 5 || value.length > 6 || value == undefined || this.plmnGroup.length > 6){
					this.plmnCls = 'is-error';
				}else{
					this.plmnCls = '';
					this.plmnGroup.push(value);
					this.plmnVal = '';
				}
			},
			removePlmn(index){
				this.plmnGroup.splice(index,1);
			},
			addMME(type){
				var value = this.mmeVal;
				this.mmeType = type;
            	if(!isValidIP(value)){
					this.mmeCls = 'is-error';
				}else{
					this.mmeCls = '';
					if(type == 'mme'){
						if(this.mmeGroup.includes(value)) {// 重复校验
							this.mmeCls = 'is-error';
						}else {
							this.mmeGroup.push(value);
						}
					}else{
						var mmePlmnList = this.mmeGroup.map(function(item){
								return item.mme + '_' + item.plmn;
							}),
							mmePlmnStr = this.mmeVal + '_' + this.mme_plmn;

						if(mmePlmnList.includes(mmePlmnStr)) {// 重复校验
							this.mmeCls = 'is-error';
						}else {
							this.mmeGroup.push({mme:this.mmeVal,plmn:this.mme_plmn})
						}
					}
					this.mmeVal = '';
				}
			},
			removeMME(index){
				this.mmeVal = '';
				this.mme_plmn = '';
				this.mmeGroup.splice(index,1);
			},
			compareIp(ipvalue,startip,endip) {
			    var vm = this,
			    	ipNum = vm.changeIpToNum(ipvalue),
			        startNum = vm.changeIpToNum(startip),
			        endNum = vm.changeIpToNum(endip);

			    if(isLessThan(ipNum, endNum) && isLessThan(startNum, ipNum)) {
			        return true;
			    }else {
			        return false;
			    }

			    return true;
			},
			changeIpToNum(ipStr) {
			    var list = (ipStr || '').split('.');

			    list = list.map(function(item){
			        if(item.length < 3) {
			            var dis = 3 - item.length;
			            for(var i = 0; i < dis; i++) item = '0' + item;
			        }

			        return item;
			    });

			    return list.join('');
			},
			isAllBindIpInRange() {
				var vm = this,
					startIp = vm.btsForm.LGWFirstStaticIPAddr,
					endIp = vm.btsForm.LGWLastStaticIPAddr,
					group = vm.bindGroup,
					bool = true;

				group.map(function(item){
					var ip = item.split('+')[1];

					if(!vm.compareIp(ip,startIp,endIp)) {
						bool = false;
					}
				});

				if(bool) {
					vm.bindCls = '';
				}else {
					vm.bindCls = 'is-error';
				}

				return bool;
			},
			isRangeIn() {
				var vm = this,
					ip = vm.bindIp,
					startIp = vm.btsForm.LGWFirstStaticIPAddr,
					endIp = vm.btsForm.LGWLastStaticIPAddr,
					bool = false;

				if(isValidIP(ip) && isValidIP(startIp) && isValidIP(endIp) && vm.compareIp(ip,startIp,endIp)) {
					bool = true;
				}

				return bool;
			},
            addBind(){ 
            	var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/,
					vm = this,
					imsi = vm.bindImsi,
					ip = vm.bindIp,
					str = '';

				if(imsi == '' || imsi == ''){
					this.bindCls = 'is-error';
				}else{
					if(reg.test(imsi) && isValidIP(ip) && imsi.length==15 && vm.isRangeIn()){
						//str = ipStart + "-" + ipEnd;
						vm.bindGroup.push(imsi + '+' + ip);
						vm.bindImsi = '';
						vm.bindIp = '';
						vm.bindCls = '';
					}else{
						vm.bindCls = 'is-error';
					}
				}
            },
            removeBind(index){
            	this.bindGroup.splice(index,1);
            },
            addSync(){
            	var vm = this;
            	vm.showSync = true;
            	vm.operType = 'add';
            	vm.$refs.syncForm.resetFields();
            },
           	editSync(row){
            	var vm = this;
            	vm.showSync = true;
            	vm.operType = 'edit';
            	Object.assign(vm.syncForm,row)
            },
            closeSync(){
            	this.showSync = false;
            },
            delSync(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var syncArr = vm.btsForm.syncList.map(function(item){
						return item.Index;
					})
					var index = syncArr.indexOf(row.Index);
					vm.btsForm.syncList.splice(index,1);
					var length = vm.btsForm.syncList.length;
					for(let i=0;i<length;i++){
						vm.btsForm.syncList[i].Index = i+1;
					}
				})
			},
			addMultiMME() {
				var vm = this;

				vm.multiMMEAddShow = true;
				vm.operType = 'add';

				vm.$nextTick(()=>{
					vm.$refs.multiForm.resetFields();
					
					// 检测第一条数据类型（WAN or IPSec）
					var firsRow = vm.btsForm.MMEBindList[0];
					if(firsRow) {
						// 判断S1-C IFName属于哪个集合里的数据，并设置对应类型
						var wanNameList = vm.wanIfNameList.map(function(item){ return item.name; }),
							ipsecNameList = vm.ipsecIfNameList.map(function(item){ return item.name; }),
							IFName = firsRow.S1CIfname;

						if(wanNameList.includes(IFName)) {
							vm.multiForm.type = '1';
						}
						if(ipsecNameList.includes(IFName)) {
							vm.multiForm.type = '2';
						}
					}else {
						vm.multiForm.type = '2';
					}
				})
			},
			delMultiMME(row, index) {
				var vm = this;

				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr, '<%=rb.getString("QueRen")%>', {
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var multiArr = vm.btsForm.MMEBindList.map(function(item){
							return item.BindIndex;
						});
					//var index = multiArr.indexOf(row.BindIndex);

					vm.btsForm.MMEBindList.splice(index, 1);
					/*
					var length = vm.btsForm.MMEBindList.length;
					for(let i=0; i<length; i++) {
						vm.btsForm.MMEBindList[i].BindIndex = i+1;
					}
					*/
					row.operateType = 'remove';
					vm.delRecord['MMEBindList'].push(Object.assign({operateType: 'remove'},{BindIndex: row.BindIndex}));
				})
			},
			saveMultiMME() {
				var vm = this;

				vm.$refs.multiForm.validate(function(valid){
					if(valid){
						var row = Object.assign({operateType: vm.operType}, vm.multiForm);

						if(vm.operType == 'add'){
							//row.BindIndex = vm.btsForm.MMEBindList.length + 1;

							try{
								delete row.type;
							}catch(e){}

							vm.btsForm.MMEBindList.push(row);
						}else{
							Object.assign(vm.btsForm.MMEBindList[vm.multiForm.BindIndex-1], row)
						}

						vm.closeMultiMME();
					}
				})
			},
			closeMultiMME() {
				var vm = this;

				vm.multiMMEAddShow = false;
			},
			mmeplmnChange(value) {
				var vm = this,
					type = vm.multiForm.type,
					row = vm.mmePlmnList.filter((item)=>{
						return item.name == value;
					})[0];

				if(row) {
					vm.multiForm.BindPLMNID = row.plmn;
				}else {
					vm.multiForm.BindPLMNID = '';
				}
			},
			s1cNameChange(value) {
				var vm = this,
					type = vm.multiForm.type,
					key = type == '1'? 'wanIfNameList':'ipsecIfNameList',
					row = vm[key].filter((item)=>{
						return item.name == value;
					})[0];

				if(row) {
					vm.multiForm.S1CIp = row.ip;
				}else {
					vm.multiForm.S1CIp = '';
				}
			},
			s1uNameChange(value) {
				var vm = this,
					type = vm.multiForm.type,
					key = type == '1'? 'wanIfNameList':'ipsecIfNameList',
					row = vm[key].filter((item)=>{
						return item.name == value;
					})[0];

				if(row) {
					vm.multiForm.S1UIp = row.ip;
				}else {
					vm.multiForm.S1UIp = '';
				}
			},
			getIPSecIFNames() {
				var vm = this,
					url = "${ctx}/cell/quicksettings/getListParamValue.action?parent_id=10014&platform=MLQ",
					params = {
						smallCellCode: vm.smallCellCode
					};

				vm.ipsecIfNameList = [];
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data,
						rows = data.rows || [];

					vm.ipsecIfNameList = rows.map(function(item, index){
						var ifname = '';

						if(index < 2) {
							ifname = 'ipsectunnel' + (index + 1);
						}

						return {
							name: ifname || item['7EB2711EDC49771D26E001655E073DC0'],
							ip: '',
							index: item['36BB6ABD0A241F1B4EF505CAFE672770'] || ''
						}
					})

					axios.post('${ctx}/cell/quicksettings/getIpsecAddress.action',stringify({smallCellCode: vm.smallCellCode})).then(function(res){
						var data = res.data;

						vm.ipsecIfNameList.map(function(item){
							item.ip = data[item.index] || '';
						})
					})
				})
			},
			getWanIFNames() {
				var vm = this,
					url = '${ctx}/cell/quicksettings/getWanAddress.action',
					params = {
						smallCellCode: vm.smallCellCode
					};

				vm.wanIfNameList = [];
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					for(var key in data) {
						vm.wanIfNameList.push({
							name: key,
							ip: data[key]
						})
					}
				})
			},
			getMMEBindingSelectParams() {
				var vm = this,
					url = '${ctx}/cell/quicksettings/getMMEBindingSelectParams.action',
					params = {
						smallCellCode: vm.smallCellCode
					};

				vm.wanIfNameList = [];
				vm.ipsecIfNameList = [];
				vm.mmePlmnList = [];
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data,
						wanData = data.wan,
						ipsecData = data.ipsec,
						mmeData = data.mmeIpPlmn;
					// wan
					var connectType = wanData['wanConnectType'],
						preCode = connectType == 'fiber'? 'eth0':'eth1';
					for(var key in wanData) {
						if(key != 'wanConnectType') {
							var wanDataMap = wanData[key],
								vlanId = wanDataMap.vlan,
								sufStr = vlanId?'.'+vlanId:'';
							vm.wanIfNameList.push({
								name: key,
								ip: wanDataMap.wanIp,
								value: preCode + sufStr + ':' + key.replace('wanConfig','')
							})
						}
					}
					// mme
					for(var key in mmeData) {
						vm.mmePlmnList.push({
							name: key,
							plmn: mmeData[key]
						})
					}
					// ipsec
					axios.post("${ctx}/cell/quicksettings/getListParamValue.action?parent_id=10014&platform=MLQ", stringify(params)).then(function(res){
						var data = res.data,
							rows = data.rows || [];

						vm.ipsecIfNameList = rows.map(function(item, index){
							var ifname = '';

							if(index < 2) {
								ifname = 'ipsectunnel' + (index + 1);
							}

							return {
								name: ifname || item['7EB2711EDC49771D26E001655E073DC0'],
								ip: '',
								index: item['36BB6ABD0A241F1B4EF505CAFE672770'] || ''
							}
						})

						vm.ipsecIfNameList.map(function(item){
							item.ip = ipsecData[item.index] || '';
						})
					});
				})
			},

            save(){
            	var vm = this;
				if(vm.showSync){
					vm.$refs.syncForm.validate(function(valid){
						if(valid){
							var row = Object.assign({operateType: vm.operType}, vm.syncForm);

							if(vm.operType == 'add'){
								vm.syncForm.Index = vm.btsForm.syncList.length + 1;
								vm.btsForm.syncList.push(row);
							}else{
								Object.assign(vm.btsForm.syncList[vm.syncForm.Index-1],row)
							}

							vm.closeSync();
						}
					})
				}else{
					 var params = {},
	                    isChanged = isFormChanged(vm.$refs.btsForm);

	                if(!isChanged){
	                    showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
	                    return;
	                }

	                vm.$refs.btsForm.fields.map(function(field){
	                    var key = vm.getNameByProp(field.prop);

	                    if(Array.isArray(field.fieldValue)){
	                        var vList = field.fieldValue.map(function(item){return item}),
	                            oList = (field.reinitialValue||[]).map(function(item){return item}),
	                            val = JSON.stringify(vList.sort()),
	                            orVal = JSON.stringify(oList.sort());

	                        if(val != orVal) {
	                            params[key] = val;
								if(['syncList'].includes(field.prop)) {
									var nList = [];
									vList.map(function(m){
										var obj = {};

										for(k in m) {
											var prop = vm.getNameByProp(k);

											if(k != 'technology') obj[prop] = m[k];
										}
										nList.push(obj);
									});

									params[key] = nList.filter(function(item){
										return ['edit','add'].includes(item.operateType);
									});
								}

								if(['MMEBindList'].includes(field.prop)) {
									var mList = [];
									vList.map(function(m){
										var obj = {};

										for(k in m) {
											var prop = vm.getNameByProp(k);

											if(k != 'MMEstatus') obj[prop] = m[k];
										}
										mList.push(obj);
									});

									params[key] = mList.filter(function(item){
										if(['add'].includes(item.operateType)) {
											try {
												delete item['DBB2CB1DB60432E1348209DFC7EBF90F'];
											} catch (error) { }
										}

										return ['edit','add'].includes(item.operateType);
									});

									if(vm.delRecord[field.prop].length) {
										vm.delRecord[field.prop].map(function(im){
											var obj = {};

											for(k in im) {
												var prop = vm.getNameByProp(k);

												obj[prop] = im[k];
											}
											
											params[key].push(obj);
										})
									}
								}
	                        };
	                    }else{
	                        if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
	                            
	                        }else if(field.fieldValue != field.reinitialValue) {
	                            params[key] = field.fieldValue;
	                        };
	                    }
	                });
					
					['2EA3E870894AD798F5B48B522FD91943','916C085A9D59B9B4792F7A2694602D27','ACD10EBE9FBF78AC7A15E27F6129C396'].map(function(keyCode) {
						if(params[keyCode]) {
							delete params[keyCode];
						}
					})
	                
	                vm.$refs.btsForm.validate(function(valid){
	                    if(valid) {
							var isNeedReboot = detectReboot(vm.$refs.btsForm, vm.rebootMap);

							if(isNeedReboot) {
								var tipContent = [
										'<%=rb.getString("JiZhanChongQiTiShi")%>',
										'<br/><br/>',
										'<input id="reboot_confirm_status" type="checkbox" />',
										'<label for="reboot_confirm_status" style="font-size: 14px;color: #1DA3FC;cursor: pointer;"><%=rb.getString("SheZhiHouChongQi")%></label>'
									].join(" ");

								var msger = $.messager.confirm('<%=rb.getString("QueRen")%>', tipContent, function (r) {
									if(r) {
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
												vm.$message.error(data["message"])
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
										vm.$message.error(data["message"])
									}

									$('#setting_main').removeClass('loading');
									settingVue.submitDisabled = false;
								})
							}
	                    }
	                });
				}
            },
            cancel(){
            	var vm = this;
				if(isFormChanged(vm.$refs.syncForm)){//返回true为改变
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
            },
			validateStaticIP(val) {
				var vm = this,
					enable = vm.btsForm.LGWStaticIPAddr_Enable == '1';

				if(enable) {
					vm.$refs.btsForm.validateField('LGWFirstStaticIPAddr');
					vm.$refs.btsForm.validateField('LGWLastStaticIPAddr');
				}
			},
			initRestart(list) {
				var vm = this,
					rebootMap = {};

				list.map(function(item){
					if(item.reboot == '1') {
						rebootMap[item.name] = true;
					}
				});
			},
			modeSelectionChange(val) {
				var vm = this;
					params = {
						CloudEPC: val == 'cloudepc'? '1':'0',
						HaloBEnable: val == 'halob'? '1':'0',
						HaloDMode: val == 'halod'? '0':'3',
					};

				if(val == 'halob') { // halob 默认打开lgw开关
					vm.btsForm.lgwEnable = '1';
				}

				Object.assign(vm.btsForm, params);
			}
		},
		watch:{
			plmnGroup:function(){
				this.btsForm.PLMNID = this.plmnGroup.toString();
			},
			mmeGroup:function(){
				if(this.mmeType == 'mme'){
					this.btsForm.MMEIPOld = this.mmeGroup.join(',');
				}else{
					var str = '';
					this.mmeGroup.map(item=>{
						str += item.mme + ',' + item.plmn + ';'
					});
					this.btsForm.MMEIPNEW = str;
				}
			},
			bindGroup:function(){
				this.btsForm.IMSItoIPBindingRange = this.bindGroup.toString();
			},
			btsForm: {
				handler: function(newVal, oldVal) {
					var vm = this,
						form = vm.$refs.btsForm;
					
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
	
	//计算静态IP范围的起始值 
	function getLowAddr(ip, netMask){
	    var lowAddr = "";
	    var ipArray = new Array();
	    var netMaskArray = new Array();
	    
	    if (4 != ip.split(".").length || netMask == ""){
	        return "";
	    }
	    for (var i = 0; i < 4; i++){
	        ipArray[i] = ip.split(".")[i];
	        netMaskArray[i] = netMask.split(".")[i];
	        if ((ipArray[i] > 255) || (ipArray[i] < 0) || (netMaskArray[i] > 255) && (netMaskArray[i] < 0)){
	            return "";
	        }
	        ipArray[i] = ipArray[i] & netMaskArray[i];
	    }
	    
	    for (var i = 0; i < 4; i++){
	        if(i == 3){
	            ipArray[i] = ipArray[i] + 1;
	        }
	        if (lowAddr == ""){
	            lowAddr +=ipArray[i];
	        } else{
	            lowAddr += "." + ipArray[i];
	        }
	    }
	    return lowAddr;
	}

	//计算静态IP范围的终止值 
	function getHighAddr(ip,netMask){
	    var lowAddr = getLowAddr(ip,netMask);
	    var hostNumber = getHostNumber(netMask);
	    if(lowAddr == "" || hostNumber == 0){
	        return "";
	    }
	    
	    var lowAddrArray = new Array();
	    for(var i = 0; i < 4; i++){
	        lowAddrArray[i] = lowAddr.split(".")[i];
	        if(i == 3){
	            lowAddrArray[i] = Number(lowAddrArray[i] - 1);
	        }
	    }
	    lowAddrArray[3] = lowAddrArray[3] + Number(hostNumber - 1);
	   
	    if(lowAddrArray[3] > 255){
	        var k = parseInt(lowAddrArray[3] / 256);       
	        
	        lowAddrArray[3] = lowAddrArray[3] % 256;
	       
	        lowAddrArray[2] = Number(lowAddrArray[2]) + Number(k);       
	       
	        if(lowAddrArray[2] > 255){
	            k = parseInt(lowAddrArray[2] / 256);
	            lowAddrArray[2] = lowAddrArray[2] % 256;
	            lowAddrArray[1] = Number(lowAddrArray[1]) + Number(k);
	            if(lowAddrArray[1] > 255){
	                k = parseInt(lowAddrArray[1] / 256);
	                lowAddrArray[1] = lowAddrArray[1] % 256;
	                lowAddrArray[0] = Number(lowAddrArray[0]) + Number(k);
	            }
	        }
	    }

	    var highAddr = "";
	    for(var i = 0; i < 4; i++){
	        if(i == 3){
	          lowAddrArray[i] = lowAddrArray[i] - 1;
	        }if(highAddr == ""){
	            highAddr = lowAddrArray[i];
	        }else{
	            highAddr += "." + lowAddrArray[i];
	        }
	    }
	    
	    return highAddr;
	}

	function getHostNumber(netMask){
	    var hostNumber = 0;
	    var netMaskArray = new Array();
	    for(var i = 0; i < 4; i++)
	  {
	        netMaskArray[i] = netMask.split(".")[i];
	        if(netMaskArray[i] < 255)
	    {
	            hostNumber = Math.pow(256,3-i) * (256 - netMaskArray[i]);
	            break;
	        }
	    }

	    return hostNumber;
	}
	//计算 static ip 范围 
	function lgwStaticIPRange(){
	    var ip = $("#LTE_LGW_START_UE_ADDR_name").val();
	    var netmask = $("#LTE_LGW_NET_MASK_name").val(); 
	    
	    var ipstart = getLowAddr(ip,netmask);
	    var ipend = getHighAddr(ip,netmask);
	        
	    var ipRangeTip = '<%=rb.getString("IPBangDingFanWei")%>' + ipstart + '-' + ipend;
	    $("#intelTdd_LGW_STATIC_IP_CONFIG_err").css("color","black").text(ipRangeTip).show();
	    
	}
</script>
<style>
	.apn-cls .el-form-item{
		margin-bottom:30px;
	}
	.apn-cls .el-form-item__error{
		left:0px;
		top:100%;
	}
	.url-cls .el-input,.url-cls .el-input__inner{
		width:300px !important;
	}
	.policySelect {
		border:1px solid #4D84FF;
		font-size:14px;
		cursor:pointer;
		border-radius:3px;
		padding:0px 8px;
		display:inline-block;
		margin-left:20px;
	}
</style>