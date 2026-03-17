<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gnbNetworkPage{
	height: 100%;
	width: 100%;
}
#gnbNetworkPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gnbNetworkPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gnbNetworkPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gnbNetworkPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gnbNetworkPage .rightContentCls .contentTableTitle{
	display: flex;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gnbNetworkPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gnbNetworkPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gnbNetworkPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gnbNetworkPage .itemListBoxCls{
	padding-top: 5px;
}
#gnbNetworkPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gnbNetworkPage .itemListBoxCls .el-icon-close ,.gnbConfigAddDialog .closeAddVlanCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gnbNetworkPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gnbNetworkPage .el-form-item{
	margin-bottom: 20px;
}
#gnbNetworkPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gnbNetworkPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gnbNetworkPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
.gnbConfigAddDialog .inputAndSelect{
	position: relative;
}
.gnbConfigAddDialog .inputAndSelect .el-select>.el-input{
	width: 55px!important;
}
.gnbConfigAddDialog .inputAndSelect .el-select>.el-input .el-input__inner{
	width: 55px!important;
}
.gnbConfigAddDialog .inputAndSelect .el-input .el-input__inner{
	width: 145px !important;
}
.gnbConfigAddDialog .inputAndSelect .el-input-group__append{
	padding-left: 65px!important;
}
#gnbNetworkPage .SoftUsimBoxCls .Soft_KeyAndOpcErrorCls .el-input-group__append{
	top:-3px;
}
#gnbNetworkPage .ponMainBoxCls{
    padding: 5px;
}
#gnbNetworkPage .ponIpBoxCls > div{
    margin-bottom: 10px;
    font-size: 12px;
}
#gnbNetworkPage .ponIpBoxCls .el-form-item{
	margin-bottom: 0px;
}
#gnbNetworkPage .ponIpBoxCls .el-form-item .el-form-item__error{
    display: none;
}
#gnbNetworkPage .ponIpBoxCls .PonIpDefaultRangeCls{
    margin-left: 10px;
    font-size: 12px;
    color: rgba(0, 0, 0, 0.32);
}
#gnbNetworkPage .ponIpBoxCls .PonIpErrorRangeCls{
    margin-left: 10px;
    font-size: 12px;
    color: #FA5555;
}
</style>

<div id="gnbNetworkPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			<%=rb.getString("WangLuoSheZhi")%> 
			<!-- 按钮  同步 -->
			<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSettingsClick" tip="<%=rb.getString("TongBu")%>">
				<span class="el-icon el-icon-circle-refresh"></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
					<el-collapse-item name="WAN">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">WAN(VLAN)/LAN</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--WAN-->
							<div> 
								<div class="contentTableTitle">
									<div>WAN(VLAN)</div>
									<div><span class="el-icon el-icon-circle-add" @click="addWANDialogOpen('','add','WAN')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="WANListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="true" 
										id="WANListTable" 
										:data="ruleForm.WANList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
										
										<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
											<template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="addWANDialogOpen(scope.row,'edit','WAN')" style="margin-right:15px;"></span>
												<span class="el-icon el-icon-operation-delete" @click="delWANList(scope.row,'WAN',event)" ></span>
											</template>
										</el-table-column>
										<el-table-column label='IP Type' min-width="100" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-if="scope.row.VlanID">
													<span v-if="scope.row.VLAN_PppoeIPType && scope.row.VLAN_PppoeIPType == 'PPPoE'">{{scope.row.VLAN_PppoeIPType}}</span>
													<span v-if="scope.row.VLAN_IPv4IPType && (scope.row.VLAN_IPv4IPType == 'DHCP' || scope.row.VLAN_IPv4IPType == 'Static')">{{scope.row.VLAN_IPv4IPType}}</span>
													<span v-if="scope.row.VLAN_IPv6IPType && scope.row.VLAN_IPv6IPType == 'DHCPv6'">IPv6 DHCP</span>
													<span v-if="scope.row.VLAN_IPv6IPType && scope.row.VLAN_IPv6IPType == 'Staticv6'">IPv6 Static</span>
												</div>
												<div v-else>
													<span v-if="scope.row.WAN_PppoeIPType && scope.row.WAN_PppoeIPType == 'PPPoE'">{{scope.row.WAN_PppoeIPType}}</span>
													<span v-if="scope.row.WANOrLan_IPv4IPType && (scope.row.WANOrLan_IPv4IPType == 'DHCP' || scope.row.WANOrLan_IPv4IPType == 'Static')">{{scope.row.WANOrLan_IPv4IPType}}</span>
													<span v-if="scope.row.WANOrLan_IPv6IPType && scope.row.WANOrLan_IPv6IPType == 'DHCPv6'">IPv6 DHCP</span>
													<span v-if="scope.row.WANOrLan_IPv6IPType && scope.row.WANOrLan_IPv6IPType == 'Staticv6'">IPv6 Static</span>
												</div>
											</template>
										</el-table-column>
										<el-table-column label='IP' min-width="120" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-if="scope.row.VlanID">
													<span v-if="scope.row.VLAN_PppoeIPType && scope.row.VLAN_PppoeIPType == 'PPPoE'">{{scope.row.VLAN_PppoeIp}}</span>
													<span v-if="scope.row.VLAN_IPv4IPType && (scope.row.VLAN_IPv4IPType == 'DHCP' || scope.row.VLAN_IPv4IPType == 'Static')">{{scope.row.VLAN_IPv4Ip}}</span>
													<span v-if="scope.row.VLAN_IPv6IPType && (scope.row.VLAN_IPv6IPType == 'DHCPv6' || scope.row.VLAN_IPv6IPType == 'Staticv6')">{{scope.row.VLAN_IPv6Ip}}</span>
												</div>
												<div v-else>
													<span v-if="scope.row.WAN_PppoeIPType && scope.row.WAN_PppoeIPType == 'PPPoE'">{{scope.row.WAN_PppoeIp}}</span>
													<span v-if="scope.row.WANOrLan_IPv4IPType && (scope.row.WANOrLan_IPv4IPType == 'DHCP' || scope.row.WANOrLan_IPv4IPType == 'Static')">{{scope.row.WANOrLan_IPv4Ip}}</span>
													<span v-if="scope.row.WANOrLan_IPv6IPType && (scope.row.WANOrLan_IPv6IPType == 'DHCPv6' || scope.row.WANOrLan_IPv6IPType == 'Staticv6')">{{scope.row.WANOrLan_IPv6Ip}}</span>
												</div>
											 </template>
										</el-table-column>
										<el-table-column label='Prefix Length/Subnet Mask' min-width="200" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-if="scope.row.VlanID">
													<span v-if="scope.row.VLAN_PppoeIPType && scope.row.VLAN_PppoeIPType == 'PPPoE'">{{scope.row.VLAN_PppoeSubnetMask}}</span>
													<span v-if="scope.row.VLAN_IPv4IPType && (scope.row.VLAN_IPv4IPType == 'DHCP' || scope.row.VLAN_IPv4IPType == 'Static')">{{scope.row.VLAN_IPv4SubnetMask}}</span>
													<span v-if="scope.row.VLAN_IPv6IPType && (scope.row.VLAN_IPv6IPType == 'DHCPv6' || scope.row.VLAN_IPv6IPType == 'Staticv6')">{{scope.row.VLAN_IPv6SubnetMask}}</span>
												</div>
												<div v-else>
													<span v-if="scope.row.WAN_PppoeIPType && scope.row.WAN_PppoeIPType == 'PPPoE'">{{scope.row.WAN_PppoeSubnetMask}}</span>
													<span v-if="scope.row.WANOrLan_IPv4IPType && (scope.row.WANOrLan_IPv4IPType == 'DHCP' || scope.row.WANOrLan_IPv4IPType == 'Static')">{{scope.row.WANOrLan_IPv4SubnetMask}}</span>
													<span v-if="scope.row.WANOrLan_IPv6IPType && (scope.row.WANOrLan_IPv6IPType == 'DHCPv6' || scope.row.WANOrLan_IPv6IPType == 'Staticv6')">{{scope.row.WANOrLan_IPv6SubnetMask}}</span>
												</div>
											 </template>
										</el-table-column>
                                        <el-table-column label='<%=rb.getString("WangGuan")%>' min-width="120" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-if="scope.row.VlanID">
													<span v-if="scope.row.VLAN_IPv4IPType && (scope.row.VLAN_IPv4IPType == 'DHCP' || scope.row.VLAN_IPv4IPType == 'Static')">{{scope.row.VLAN_IPv4Gateway}}</span>
													<span v-if="scope.row.VLAN_IPv6IPType && (scope.row.VLAN_IPv6IPType == 'DHCPv6' || scope.row.VLAN_IPv6IPType == 'Staticv6')">{{scope.row.VLAN_IPv6Gateway}}</span>
												</div>
												<div v-else>
													<span v-if="scope.row.WANOrLan_IPv4IPType && (scope.row.WANOrLan_IPv4IPType == 'DHCP' || scope.row.WANOrLan_IPv4IPType == 'Static')">{{scope.row.WANOrLan_IPv4Gateway}}</span>
													<span v-if="scope.row.WANOrLan_IPv6IPType && (scope.row.WANOrLan_IPv6IPType == 'DHCPv6' || scope.row.WANOrLan_IPv6IPType == 'Staticv6')">{{scope.row.WANOrLan_IPv6Gateway}}</span>
												</div>
											 </template>
										</el-table-column>
										<el-table-column label='Port Type' min-width="100" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-if="scope.row.VlanID">
													<span v-if="scope.row.VLAN_IPv4IPType && (scope.row.VLAN_IPv4IPType == 'DHCP' || scope.row.VLAN_IPv4IPType == 'Static')">{{scope.row.VLAN_IPv4PortType}}</span>
													<span v-if="scope.row.VLAN_IPv6IPType && (scope.row.VLAN_IPv6IPType == 'DHCPv6' || scope.row.VLAN_IPv6IPType == 'Staticv6')">{{scope.row.VLAN_IPv6PortType}}</span>
												</div>
												<div v-else>
													<span v-if="scope.row.WANOrLan_IPv4IPType && (scope.row.WANOrLan_IPv4IPType == 'DHCP' || scope.row.WANOrLan_IPv4IPType == 'Static')">{{scope.row.WANOrLan_IPv4PortType}}</span>
													<span v-if="scope.row.WANOrLan_IPv6IPType && (scope.row.WANOrLan_IPv6IPType == 'DHCPv6' || scope.row.WANOrLan_IPv6IPType == 'Staticv6')">{{scope.row.WANOrLan_IPv6PortType}}</span>
												</div>
											 </template>
										</el-table-column>
										<el-table-column label='VLAN Name' min-width="100" prop="VlanName" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='VLAN ID' min-width="80" prop="VlanID" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='WANList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.WANList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--LAN List-->
                            <div> 
								<div class="contentTableTitle">
									<div>LAN</div>
									<div><span class="el-icon el-icon-circle-add" @click="addLANDialogOpen('','add','LAN')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="LANListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="true" 
										id="LANListTable" 
										:data="ruleForm.LANList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
											<template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="addLANDialogOpen(scope.row,'edit','LAN')" style="margin-right:15px;"></span>
												<span class="el-icon el-icon-operation-delete" @click="delLANList(scope.row,'LAN',event)" ></span>
											</template>
										</el-table-column>
										<el-table-column label='ID' min-width="120" prop="WANOrLan_IPv4idx" show-overflow-tooltip></el-table-column>
										<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="120" prop="LAN_Status" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='<%=rb.getString("IPDiZhi")%>' min-width="120" prop="WANOrLan_IPv4Ip" show-overflow-tooltip></el-table-column>
										<el-table-column label='Subnet Mask' min-width="120" prop="WANOrLan_IPv4SubnetMask" show-overflow-tooltip></el-table-column>
										<el-table-column label='<%=rb.getString("MACDiZhi")%>' min-width="120" prop="LAN_Mac" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='LANList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.LANList'></el-input>
									</el-form-item>
								</div>
                                <div style="display: flex;align-items: center;margin-bottom:10px;">
                                    <div style=" font-size: 12px;">PON Enable</div>
                                    <el-form-item prop='Pon_Switch' style="width:40%;min-width:400px;margin-bottom:0px" label-width="160px">
                                        <el-switch v-model="ruleForm.Pon_Switch" active-value="1" inactive-value="0" style='margin-left:10px;'></el-switch>
                                    </el-form-item>
                                </div>
                                <div class="ponMainBoxCls" v-if="ruleForm.Pon_Switch == '1'">
                                    <div class="ponIpBoxCls">
                                        <div>IP Address</div>
                                        <div style="display: flex;align-items: center;margin-left:10px;">
                                            <span>192.168.150.</span>
                                            <el-form-item prop='Pon_StartIp' style="width:80px;margin-left:10px;" label-width="160px">
                                                <el-input v-model.trim='ruleForm.Pon_StartIp' style="width:80px"></el-input>
                                            </el-form-item>
                                            <span style="margin: 0px 10px;">-</span>
                                            <span>192.168.150.</span>
                                            <el-form-item prop='Pon_EndIp' style="width:80px;margin-left:10px;" label-width="160px">
                                                <el-input v-model.trim='ruleForm.Pon_EndIp' style="width:80px"></el-input>
                                            </el-form-item>
                                            <div :class="(PonStartIpErrorShow || PonEndIpErrorShow ) ? 'PonIpErrorRangeCls' :'PonIpDefaultRangeCls'"><%=rb.getString("FanWei")%>：1~254,Integer</div>
                                        </div>
                                        <div v-show="ponStartIpNoLessEndIpErrShow" class="PonIpErrorRangeCls">The start ip address must be less than the end ip address</div>
                                    </div>
                                    <el-form-item prop='Pon_SubnetMask' style="min-width:400px;" label="Subnet Mask" label-width="160px">
                                        <el-input v-model.trim='ruleForm.Pon_SubnetMask'></el-input>
                                    </el-form-item>
                                </div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="defaultRoute">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Default Route</span>
							</p>
						</template>
						<div class="rightContentCls" >
                            <div> 
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-form-item prop="defaultRouteVal" label='Default Route' label-width="160px">
										<el-select v-model='ruleForm.defaultRouteVal'>
											<el-option v-for="item in defaultRouteList" :label='item' :value='item'></el-option>
										</el-select>
									</el-form-item>
                                    <el-form-item prop="dnsType" label='DNS Type' label-width="160px">
										<el-select v-model='ruleForm.dnsType'>
											<el-option v-for="item in dnsTypeList" :label='item.label' :value='item.value'></el-option>
										</el-select>
									</el-form-item>
									<div class="allowMoreInputBoxCls" v-show="ruleForm.dnsType === '1'">
										<div class="allowMoreInputHeadCls">
											<span class="allowMoreInputTitleCls">DNS</span>
										</div>
										<div class="allowMoreInputContentCls">
											<div class="allowMoreInputFieldCls">
												<el-input v-model="defaultRoute.Dns"></el-input>
												<div class="allowMoreInputAddBtnCls" @click="addDefaultRouteDns">
													<span class="el-icon el-icon-plus"></span>
													<span>Add</span>
												</div>
											</div>
											<div class="allowMoreInputParamsCls">
												<div v-for="item in defaultRouteDnsList" class="allowMoreInputParamsItemCls">
													<span>{{item}}</span>
													<span class="el-icon el-icon-close" style="margin-left:5px;" @click="defaultRouteDnsDel(item)"></span>
												</div>
											</div>
										</div>
										<div class="allowMoreInputFootCls">
											<p class="inputErrorBoxCls">{{defaultRoute.defaultRouteDnsErrorMessage}}</p>
										</div>
										<el-form-item prop='defaultRouteDnsStr' style="display:none;" label="" label-width="0px">
											<el-input v-model='ruleForm.defaultRouteDnsStr'></el-input>
										</el-form-item>
									</div>
                                    <div class="allowMoreInputBoxCls" v-show="ruleForm.dnsType === '0'">
										<div class="allowMoreInputHeadCls">
											<span class="allowMoreInputTitleCls">DNS</span>
										</div>
										<div class="allowMoreInputContentCls">
											<div class="allowMoreInputParamsCls">
												<div v-for="item in dhcpDnsList" class="allowMoreInputParamsItemCls">
													<span>{{item}}</span>
													<span class="el-icon el-icon-close" style="margin-left:5px;" @click="dhcpDnsDel(item)"></span>
												</div>
											</div>
										</div>
										<el-form-item prop='dhcpDnsStr' style="display:none;" label="" label-width="0px">
											<el-input v-model='ruleForm.dhcpDnsStr'></el-input>
										</el-form-item>
									</div>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="IPSEC">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">IPSEC</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--Interface Binding-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">Interface Binding</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='NGAP_InterfaceBinding' style="width:40%;min-width:400px;" label="NGAP Interface Binding" label-width="140px">
										<span slot="label" class="labelIconCls">
											NGAP Interface Binding
										</span>
										<el-select v-model='ruleForm.NGAP_InterfaceBinding'>
											<el-option v-for="item in tunnelNameList" :label='item' :value='item'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='NGU_InterfaceBinding' style="width:40%;min-width:400px;" label="NGU Interface Binding" label-width="140px">
										<span slot="label" class="labelIconCls">
											NGU Interface Binding
										</span>
										<el-select v-model='ruleForm.NGU_InterfaceBinding'>
											<el-option v-for="item in tunnelNameList" :label='item' :value='item'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
							<!--IPSec Tunnel List-->
							<div> 
								<div class="contentTableTitle">
									<div>IPSec Tunnel List<span style="font-size:12px;color:#999999;margin-left:10px;">(No more than 3)</span></div>
									<div v-if="addIPSecTunnelBtnShow"><span class="el-icon el-icon-circle-add" @click="addIPSecTunnelDialogOpen('','add','IPSec')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="IPSecTunnelListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="true" 
										id="IPSecTunnelListTable" 
										:data="ruleForm.IPSecTunnelList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
											<template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="addIPSecTunnelDialogOpen(scope.row,'edit','IPSec')" style="margin-right:15px;"></span>
												<span class="el-icon el-icon-operation-delete" @click="delIPSecTunnelList(scope.row,'IPSec',event)" ></span>
											</template>
										</el-table-column>
										<el-table-column label='Tunnel Name' min-width="120" prop="IPSec_TunnelName" show-overflow-tooltip></el-table-column>
										<el-table-column label='Enable' min-width="100" prop="IPSec_Switch" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-if="scope.row.IPSec_Switch == '0'">
													<span>OFF</span> 
												</div>
												<div v-if="scope.row.IPSec_Switch == '1'">
													<span>ON</span> 
												</div>
											</template>
										</el-table-column>
										<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="120" prop="IPSec_Status" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-if="scope.row.IPSec_Status != '2'">
													<!-- <span class="el-icon el-icon-status-reject1 redIcon" style="margin-right:5px;"></span>
													<span style="color:red;">Not Connected</span>  -->
													<span>Not Connected</span> 
												</div>
												<div v-if="scope.row.IPSec_Status == '2'">
													<!-- <span class="el-icon el-icon-status-accept greenIcon" style="margin-right:5px;"></span> -->
													<span>Connected</span> 
												</div>
											</template>
										</el-table-column>
                                        <el-table-column label='<%=rb.getString("IPDiZhi")%>' min-width="120" prop="IPSec_IP" show-overflow-tooltip></el-table-column>
										<el-table-column label='<%=rb.getString("WangGuan")%>' min-width="120" prop="IPSec_Gateway" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='IPSecTunnelList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.IPSecTunnelList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--Strong Swan-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">Strong Swan</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='Swan_IKEDebugLevel' style="width:40%;min-width:400px;" label="IKE Debug Level" label-width="140px">
										<el-select v-model='ruleForm.Swan_IKEDebugLevel'>
											<el-option label='-1' value='-1'></el-option>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Swan_ESPDebugLevel' style="width:40%;min-width:400px;" label="ESP Debug Level" label-width="140px">
										<el-select v-model='ruleForm.Swan_ESPDebugLevel'>
											<el-option label='-1' value='-1'></el-option>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Swan_CFGDebugLevel' style="width:40%;min-width:400px;" label="CFG Debug Level" label-width="140px">
										<el-select v-model='ruleForm.Swan_CFGDebugLevel'>
											<el-option label='-1' value='-1'></el-option>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Swan_KNLDebugLevel' style="width:40%;min-width:400px;" label="KNL Debug Level" label-width="140px">
										<el-select v-model='ruleForm.Swan_KNLDebugLevel'>
											<el-option label='-1' value='-1'></el-option>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Swan_MGRDebugLevel' style="width:40%;min-width:400px;" label="MGR Debug Level" label-width="140px">
										<el-select v-model='ruleForm.Swan_MGRDebugLevel'>
											<el-option label='-1' value='-1'></el-option>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Swan_ASNDebugLevel' style="width:40%;min-width:400px;" label="ASN Debug Level" label-width="140px">
										<el-select v-model='ruleForm.Swan_ASNDebugLevel'>
											<el-option label='-1' value='-1'></el-option>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Swan_CHDDebugLevel' style="width:40%;min-width:400px;" label="CHD Debug Level" label-width="140px">
										<el-select v-model='ruleForm.Swan_CHDDebugLevel'>
											<el-option label='-1' value='-1'></el-option>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Swan_LIBDebugLevel' style="width:40%;min-width:400px;" label="LIB Debug Level" label-width="140px">
										<el-select v-model='ruleForm.Swan_LIBDebugLevel'>
											<el-option label='-1' value='-1'></el-option>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Swan_Port' style="width:40%;min-width:400px;" label="Port" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.Swan_Port'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='Swan_PortNATT' style="width:40%;min-width:400px;" label="Port NAT T" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.Swan_PortNATT'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='Swan_RetryInitiateInterval' style="width:40%;min-width:400px;" label="Retry Initiate Interval" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.Swan_RetryInitiateInterval'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='Swan_MTU' style="width:40%;min-width:400px;" label="Ipsec MTU" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.Swan_MTU'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~9600,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='Swan_MSS' style="width:40%;min-width:400px;" label="Ipsec MSS" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.Swan_MSS'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~9600,Integer</template>
										</el-input>
									</el-form-item>
								</div>
							</div>
							<!--Soft Usim-->
							<div class="SoftUsimBoxCls"> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">Soft Usim</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='Soft_Enable' style="width:40%;min-width:400px;" label="Soft Usim" label-width="140px">
										<el-select v-model='ruleForm.Soft_Enable'>
											<el-option label='OFF' value='0'></el-option>
											<el-option label='ON' value='1'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Soft_IMSI' style="width:40%;min-width:400px;" label="IMSI" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.Soft_IMSI' maxlength='1024'>
											<template slot="append">Length:0-1024,String</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='Soft_IpsecUsimAuthenticationEnable' style="width:40%;min-width:400px;" label="IpsecUsimAuthenticationEnable" label-width="160px" class='validate-item'>
										<el-select v-model='ruleForm.Soft_IpsecUsimAuthenticationEnable'>
											<el-option label='unbound' value='0'></el-option>
											<el-option label='SN band' value='1'></el-option>
											<el-option label='MAC band' value='2'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Soft_Key' style="width:80%;min-width:400px;" label="Key" label-width="160px" class='validate-item Soft_KeyAndOpcErrorCls'>
										<el-input v-model='ruleForm.Soft_Key' maxlength='1024'>
											<template slot="append">
												<p>Range:0-1024 Even digit Letters or Numbers</p> 
											</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='Soft_Opc' style="width:80%;min-width:400px;" label="OPC" label-width="160px" class='validate-item Soft_KeyAndOpcErrorCls'>
										<el-input v-model='ruleForm.Soft_Opc' maxlength='1024'>
											<template slot="append">
												<p>Range:0-1024 Even digit Letters or Numbers</p> 
											</template>
										</el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="DSCP">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">DSCP</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--DSCP List-->
                            <div> 
								<div class="contentTableTitle">
									<div>DSCP List</div>
									<div><span class="el-icon el-icon-circle-add" @click="addDSCPDialogOpen('','add','DSCP')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="DSCPListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="true" 
										id="DSCPListTable" 
										:data="ruleForm.DSCPList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
										
										<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
											<template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="addDSCPDialogOpen(scope.row,'edit','DSCP')" style="margin-right:15px;"></span>
												<span class="el-icon el-icon-operation-delete" @click="delDSCPList(scope.row,'DSCP',event)" ></span>
											</template>
										</el-table-column>
										 <el-table-column label='<%=rb.getString("XuLieHao")%>' min-width="120" prop="DSCP_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='DSCP' min-width="120" prop="DSCP_DSCPVal" show-overflow-tooltip></el-table-column>
										<el-table-column label='VLAN Priority' min-width="120" prop="DSCP_VLANPriority" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='DSCPList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.DSCPList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--NGAP DSCP-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">NGAP DSCP</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='NGAP_DSCP' style="width:40%;min-width:400px;" label="DSCP" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.NGAP_DSCP'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~63,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='NGAP_VLANPriority' style="width:40%;min-width:400px;" label="VLAN Priority" label-width="160px">
										<el-select v-model='ruleForm.NGAP_VLANPriority'>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
											<el-option label='5' value='5'></el-option>
											<el-option label='6' value='6'></el-option>
											<el-option label='7' value='7'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
							<!--X2AP DSCP-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">X2AP DSCP</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='X2AP_DSCP' style="width:40%;min-width:400px;" label="DSCP" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.X2AP_DSCP'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~63,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='X2AP_VLANPriority' style="width:40%;min-width:400px;" label="VLAN Priority" label-width="160px">
										<el-select v-model='ruleForm.X2AP_VLANPriority'>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
											<el-option label='5' value='5'></el-option>
											<el-option label='6' value='6'></el-option>
											<el-option label='7' value='7'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
							<!--F1AP DSCP-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">F1AP DSCP</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='F1AP_DSCP' style="width:40%;min-width:400px;" label="DSCP" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.F1AP_DSCP'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~63,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='F1AP_VLANPriority' style="width:40%;min-width:400px;" label="VLAN Priority" label-width="160px">
										<el-select v-model='ruleForm.F1AP_VLANPriority'>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
											<el-option label='5' value='5'></el-option>
											<el-option label='6' value='6'></el-option>
											<el-option label='7' value='7'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
							<!--XNAP DSCP-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">XNAP DSCP</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='XNAP_DSCP' style="width:40%;min-width:400px;" label="DSCP" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.XNAP_DSCP'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~63,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='XNAP_VLANPriority' style="width:40%;min-width:400px;" label="VLAN Priority" label-width="160px">
										<el-select v-model='ruleForm.XNAP_VLANPriority'>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
											<el-option label='5' value='5'></el-option>
											<el-option label='6' value='6'></el-option>
											<el-option label='7' value='7'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
							<!--OAM DSCP-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">OAM DSCP</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='OAM_DSCP' style="width:40%;min-width:400px;" label="DSCP" label-width="160px" class='validate-item'>
										<el-input v-model.trim='ruleForm.OAM_DSCP'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~63,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='OAM_VLANPriority' style="width:40%;min-width:400px;" label="VLAN Priority" label-width="160px">
										<el-select v-model='ruleForm.OAM_VLANPriority'>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
											<el-option label='2' value='2'></el-option>
											<el-option label='3' value='3'></el-option>
											<el-option label='4' value='4'></el-option>
											<el-option label='5' value='5'></el-option>
											<el-option label='6' value='6'></el-option>
											<el-option label='7' value='7'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="Static">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Static Routing</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--Static Routing List-->
                            <div> 
								<div class="contentTableTitle">
									<div>Static Routing List</div>
									<div><span class="el-icon el-icon-circle-add" @click="addStaticDialogOpen('','add','Static')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="StaticListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="true" 
										id="StaticListTable" 
										:data="ruleForm.StaticList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
										
										<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
											<template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="addStaticDialogOpen(scope.row,'edit','Static')" style="margin-right:15px;"></span>
												<span class="el-icon el-icon-operation-delete" @click="delStaticList(scope.row,'Static',event)" ></span>
											</template>
										</el-table-column>
										<el-table-column label='IP Version' min-width="120" prop="Static_IPVersion" show-overflow-tooltip>
											<template slot-scope="scope">
                                                <span v-if="scope.row.Static_IPVersion == '1'">IPv4</span>
												<span v-if="scope.row.Static_IPVersion == '2'">IPv6</span>
											</template>
										</el-table-column>
                                        <el-table-column label='Destination Network' min-width="160" prop="Static_DestinationNetwork" show-overflow-tooltip></el-table-column>
										<el-table-column label='Prefix Length/Subnet Mask' min-width="220" prop="Static_NetmaskPrefixLength" show-overflow-tooltip></el-table-column>
										<el-table-column label='Gateway' min-width="120" prop="Static_Gateway" show-overflow-tooltip></el-table-column>
										<el-table-column label='Interface Name' min-width="120" prop="Static_InterfaceName" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='StaticList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.StaticList'></el-input>
									</el-form-item>
								</div>
							</div>
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
	<!-- WAN 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addWANDialogShow" @close="closeAddWANDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addWANDialogForm" :model='addWANDialogForm' :rules='addWANDialogRules' label-position="top">     		     			            
			<el-form-item prop='IPType' style="min-width:400px;" label="IP Type" label-width="160px" class='validate-item'>
				<el-select v-model='addWANDialogForm.IPType' >
					<el-option v-if="ipv4SeclectShow" label='DHCP' value='DHCP' ></el-option>
                    <el-option v-if="ipv4SeclectShow" label='Static' value='Static'></el-option>
					<el-option v-if="ipv6SeclectShow" label='IPv6 DHCP' value='DHCPv6'></el-option>
                    <el-option v-if="ipv6SeclectShow" label='IPv6 Static' value='Staticv6'></el-option>
					<!-- 2024-11-22 文件和测试沟通要求去掉
					<el-option v-if="optType == 'add' || (optType == 'edit' && addWANDialogForm.IPType == 'PPPoE')" label='PPPOE' value='PPPoE'></el-option>
					-->
				</el-select>
			</el-form-item>
			<el-form-item v-show="!['PPPoE'].includes(addWANDialogForm.IPType)" prop='PortType' style="min-width:400px;" label="Port Type" label-width="160px" class='validate-item'>
				<el-select v-model='addWANDialogForm.PortType' >
					<el-option v-for="item in PortTypeList" :label='item' :value='item'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item v-show="['PPPoE'].includes(addWANDialogForm.IPType)" prop='DiallingMethod' style="min-width:400px;" label="Dialling Method" label-width="160px" class='validate-item'>
				<el-select v-model='addWANDialogForm.DiallingMethod' >
					<el-option label='PAP' value='PAP'></el-option>
                    <el-option label='CHAP' value='CHAP'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item v-show="['PPPoE'].includes(addWANDialogForm.IPType)" prop='UserName' style="min-width:400px;" label="User Name" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addWANDialogForm.UserName' ></el-input>
			</el-form-item>
			<el-form-item v-show="['PPPoE'].includes(addWANDialogForm.IPType)" prop='Password' style="min-width:400px;" label="Password" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addWANDialogForm.Password' ></el-input>
			</el-form-item>
			<el-form-item v-show="['Static','Staticv6'].includes(addWANDialogForm.IPType)" prop='IP' style="min-width:400px;" label="IP" label-width="160px">
				<el-input v-model.trim='addWANDialogForm.IP'></el-input>
			</el-form-item>
			<el-form-item v-show="['Static'].includes(addWANDialogForm.IPType )" prop='SubnetMask' style="min-width:400px;" label="Subnet Mask" label-width="160px">
                <el-input v-model.trim='addWANDialogForm.SubnetMask'></el-input>
			</el-form-item>
            <el-form-item v-show="['Staticv6'].includes(addWANDialogForm.IPType )" prop='SubnetMask' style="min-width:400px;" label="Prefix Length" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addWANDialogForm.SubnetMask'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~128,Integer</template>
                </el-input>
			</el-form-item>
            <el-form-item v-show="['Static','Staticv6'].includes(addWANDialogForm.IPType )" prop='Gateway' style="min-width:400px;" label="Gateway" label-width="160px">
                <el-input v-model.trim='addWANDialogForm.Gateway' ></el-input>
			</el-form-item>
			<el-form-item prop='VlanName' style="min-width:400px;" label="VLAN Name" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addWANDialogForm.VlanName' maxlength='13' :disabled="addWanDisVlanId">
					<template slot="append">Length：1~13,Characters</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='VlanID' style="min-width:400px;" label="VLAN ID" label-width="160px" class='validate-item closeAddVlanCls'>
                <el-input v-model.trim='addWANDialogForm.VlanID' :disabled="addWanDisVlanId">
					<template slot="append">
						<%=rb.getString("FanWei")%>：2~4094,Integer
					</template>
				</el-input>
			</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addWANDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addWANDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- LAN 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addLANDialogShow" @close="closeAddLANDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addLANDialogForm" :model='addLANDialogForm' :rules='addLANDialogRules' label-position="top">     		     			            
			<el-form-item prop='WANOrLan_IPv4Ip' style="min-width:400px;" label="IP" label-width="160px">
				<el-input v-model.trim='addLANDialogForm.WANOrLan_IPv4Ip'></el-input>
			</el-form-item>
			<el-form-item prop='WANOrLan_IPv4SubnetMask' style="min-width:400px;" label="Subnet Mask" label-width="160px">
                <el-input v-model.trim='addLANDialogForm.WANOrLan_IPv4SubnetMask'></el-input>
			</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addLANDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addLANDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- IPSecTunnel 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="54%" :visible.sync="addIPSecTunnelDialogShow" @close="closeAddIPSecTunnelDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addIPSecDialogForm" :model='addIPSecDialogForm' :rules='addIPSecDialogRules' label-position="top">     		     			            
				<el-collapse v-model="activeIPSecTunnelCollapse">
					<el-collapse-item name="Basic">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold"><%=rb.getString("JiBenSheZhi")%></span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='IPSec_Switch' style="width:40%;min-width:400px;" label="Enabled" label-width="160px">
									<el-switch v-model="addIPSecDialogForm.IPSec_Switch" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
								</el-form-item>
								<el-form-item prop='IPSec_TunnelName' style="width:40%;min-width:400px;" label="Tunnel Name" label-width="160px" class='validate-item'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_TunnelName' maxlength="64" :disabled="optType == 'edit'">
										<template slot="append"><%=rb.getString("FanWei")%>：1~64,Digit string</template>
									</el-input>
								</el-form-item>
								<el-form-item prop="IPSec_LeftAuth" style="width:40%;min-width:400px;" label="Left Auth"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_LeftAuth'>
										<el-option label='psk' value='psk'></el-option>
										<el-option label='pubkey' value='pubkey'></el-option>
										<el-option label='eap-aka' value='eap-aka'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_RightAuth" style="width:40%;min-width:400px;" label="Right Auth"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_RightAuth'>
										<el-option label='psk' value='psk'></el-option>
										<el-option label='pubkey' value='pubkey'></el-option>
										<el-option label='eap-aka' value='eap-aka'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='IPSec_Gateway' style="width:40%;min-width:400px;" label="Gateway" label-width="160px" class='validate-item'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_Gateway' maxlength="64">
										<template slot="append"><%=rb.getString("FanWei")%>：1~64,Digit string</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='IPSec_RightSubnet' style="width:40%;min-width:400px;" label="Right Subnet" label-width="160px" class='validate-item'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_RightSubnet' maxlength="128" >
										<template slot="append"><%=rb.getString("FanWei")%>：0~128,Digit string</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='IPSec_RightID' style="width:40%;min-width:400px;" label="Right ID" label-width="160px" class='validate-item'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_RightID' maxlength="64">
										<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
									</el-input>
								</el-form-item>
								<el-form-item prop="IPSec_SecretKey" style="width:40%;min-width:400px;" label="SecretKey" label-width="110px" class='validate-item'>
									<el-input type="text" maxlength="64" v-model.trim='addIPSecDialogForm.IPSec_SecretKey'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="Advance">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold"><%=rb.getString("GaoJiSheZhi")%></span>
							</p>
						</template>
						<div class="rightContentCls">
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='IPSec_LeftID' style="width:40%;min-width:400px;" label="Left ID" label-width="160px" class='validate-item'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_LeftID' maxlength="64">
										<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='IPSec_LeftCert' style="width:40%;min-width:400px;" label="LeftCert" label-width="160px" class='validate-item'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_LeftCert' maxlength="64">
										<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='IPSec_LeftSourceIp' style="width:40%;min-width:400px;" label="LeftSourceIp" label-width="160px" class='validate-item'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_LeftSourceIp' maxlength="64">
										<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='IPSec_LeftSubnet' style="width:40%;min-width:400px;" label="Left Subnet" label-width="160px" class='validate-item'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_LeftSubnet' maxlength="128">
										<template slot="append"><%=rb.getString("FanWei")%>：0~128,Digit string</template>
									</el-input>
								</el-form-item>
								<el-form-item prop="IPSec_Fragmentation" style="width:40%;min-width:400px;" label="Fragmentation"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_Fragmentation'>
										<el-option label='Yes' value='yes'></el-option>
										<el-option label='Accept' value='accept'></el-option>
										<el-option label='Force' value='force'></el-option>
										<el-option label='No' value='no'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_IKEEncryption" style="width:40%;min-width:400px;" label="IKE Encryption"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_IKEEncryption'>
										<el-option label='aes128' value='aes128'></el-option>
										<el-option label='aes256' value='aes256'></el-option>
										<el-option label='3des' value='3des'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_IKEDHGroup" style="width:40%;min-width:400px;" label="IKE DH Group"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_IKEDHGroup'>
										<el-option label='modp768' value='modp768'></el-option>
										<el-option label='modp1024' value='modp1024'></el-option>
										<el-option label='modp1536' value='modp1536'></el-option>
										<el-option label='modp2048' value='modp2048'></el-option>
										<el-option label='modp4096' value='modp4096'></el-option>
										<el-option label='none' value='none'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_IKEAuthentication" style="width:40%;min-width:400px;" label="IKE Authentication"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_IKEAuthentication'>
										<el-option label='sha1' value='sha1'></el-option>
										<el-option label='sha1_160' value='sha1_160'></el-option>
										<el-option label='sha256_96' value='sha256_96'></el-option>
										<el-option label='sha256' value='sha256'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_ESPEncryption" style="width:40%;min-width:400px;" label="ESP Encryption"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_ESPEncryption'>
										<el-option label='aes128' value='aes128'></el-option>
										<el-option label='aes256' value='aes256'></el-option>
										<el-option label='3des' value='3des'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_ESPDHGroup" style="width:40%;min-width:400px;" label="ESP DH Group"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_ESPDHGroup'>
										<el-option label='modp768' value='modp768'></el-option>
										<el-option label='modp1024' value='modp1024'></el-option>
										<el-option label='modp1536' value='modp1536'></el-option>
										<el-option label='modp2048' value='modp2048'></el-option>
										<el-option label='modp4096' value='modp4096'></el-option>
										<el-option label='none' value='none'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_ESPAuthentication" style="width:40%;min-width:400px;" label="ESP Authentication"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_ESPAuthentication'>
										<el-option label='sha1' value='sha1'></el-option>
										<el-option label='sha1_160' value='sha1_160'></el-option>
										<el-option label='sha256_96' value='sha256_96'></el-option>
										<el-option label='sha256' value='sha256'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='IPSec_KeyLeft' style="width:40%;min-width:400px;" label="Key Left" label-width="160px" class='validate-item inputAndSelect'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_KeyLeft' style="width:152px" maxlength="64">
										<template v-if="addIPSecDialogForm.IPSec_KeyLeftType == 's'" slot="append"><%=rb.getString("FanWei")%>：1~31536000,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_KeyLeftType == 'm'" slot="append"><%=rb.getString("FanWei")%>：1~525600,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_KeyLeftType == 'h'" slot="append"><%=rb.getString("FanWei")%>：1~8760,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_KeyLeftType == 'd'" slot="append"><%=rb.getString("FanWei")%>：1~365,Integer</template>
									</el-input>
									<el-select v-model='addIPSecDialogForm.IPSec_KeyLeftType' style="position:absolute;left:144px;top:0px;">
										<el-option label='s' value='s'></el-option>
										<el-option label='m' value='m'></el-option>
										<el-option label='h' value='h'></el-option>
										<el-option label='d' value='d'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='IPSec_IKELeftTime' style="width:40%;min-width:400px;" label="IKELeftTime" label-width="160px" class='validate-item inputAndSelect'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_IKELeftTime' style="width:152px" maxlength="64">
										<template v-if="addIPSecDialogForm.IPSec_IKELeftTimeType == 's'" slot="append"><%=rb.getString("FanWei")%>：1~31536000,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_IKELeftTimeType == 'm'" slot="append"><%=rb.getString("FanWei")%>：1~525600,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_IKELeftTimeType == 'h'" slot="append"><%=rb.getString("FanWei")%>：1~8760,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_IKELeftTimeType == 'd'" slot="append"><%=rb.getString("FanWei")%>：1~365,Integer</template>
									</el-input>
									<el-select v-model='addIPSecDialogForm.IPSec_IKELeftTimeType' style="position:absolute;left:144px;top:0px;">
										<el-option label='s' value='s'></el-option>
										<el-option label='m' value='m'></el-option>
										<el-option label='h' value='h'></el-option>
										<el-option label='d' value='d'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='IPSec_RekeyMargin' style="width:40%;min-width:400px;" label="RekeyMargin" label-width="160px" class='validate-item inputAndSelect'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_RekeyMargin' style="width:152px" maxlength="64">
										<template v-if="addIPSecDialogForm.IPSec_RekeyMarginType == 's'" slot="append"><%=rb.getString("FanWei")%>：1~31536000,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_RekeyMarginType == 'm'" slot="append"><%=rb.getString("FanWei")%>：1~525600,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_RekeyMarginType == 'h'" slot="append"><%=rb.getString("FanWei")%>：1~8760,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_RekeyMarginType == 'd'" slot="append"><%=rb.getString("FanWei")%>：1~365,Integer</template>
									</el-input>
									<el-select v-model='addIPSecDialogForm.IPSec_RekeyMarginType' style="position:absolute;left:144px;top:0px;">
										<el-option label='s' value='s'></el-option>
										<el-option label='m' value='m'></el-option>
										<el-option label='h' value='h'></el-option>
										<el-option label='d' value='d'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_Dpdaction" style="width:40%;min-width:400px;" label="Dpdaction"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_Dpdaction'>
										<el-option label='None' value='none'></el-option>
										<el-option label='Clear' value='clear'></el-option>
										<el-option label='Hold' value='hold'></el-option>
										<el-option label='Restart' value='restart'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='IPSec_Dpddelay' style="width:40%;min-width:400px;" label="Dpddelay" label-width="160px" class='validate-item inputAndSelect'>
									<el-input v-model.trim='addIPSecDialogForm.IPSec_Dpddelay' style="width:152px" maxlength="64">
										<template v-if="addIPSecDialogForm.IPSec_DpddelayType == 's'" slot="append"><%=rb.getString("FanWei")%>：1~31536000,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_DpddelayType == 'm'" slot="append"><%=rb.getString("FanWei")%>：1~525600,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_DpddelayType == 'h'" slot="append"><%=rb.getString("FanWei")%>：1~8760,Integer</template>
										<template v-if="addIPSecDialogForm.IPSec_DpddelayType == 'd'" slot="append"><%=rb.getString("FanWei")%>：1~365,Integer</template>
									</el-input>
									<el-select v-model='addIPSecDialogForm.IPSec_DpddelayType' style="position:absolute;left:144px;top:0px;">
										<el-option label='s' value='s'></el-option>
										<el-option label='m' value='m'></el-option>
										<el-option label='h' value='h'></el-option>
										<el-option label='d' value='d'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop="IPSec_LeftInterface" style="width:40%;min-width:400px;" label="Left Interface"  label-width="160px">
									<el-select v-model='addIPSecDialogForm.IPSec_LeftInterface'>
										<el-option label='None' value=''></el-option>
									</el-select>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
				</el-collapse>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addIPSecTunnelDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addIPSecTunnelDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- DSCP 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addDSCPDialogShow" @close="closeAddDSCPDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addDSCPDialogForm" :model='addDSCPDialogForm' :rules='addDSCPDialogRules' label-position="top">     		     			            
			<el-form-item prop='DSCP_DSCPVal' style="min-width:400px;" label="DSCP" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addDSCPDialogForm.DSCP_DSCPVal'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~63,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='DSCP_VLANPriority' style="min-width:400px;" label="VLAN Priority" label-width="160px">
				<el-select v-model='addDSCPDialogForm.DSCP_VLANPriority'>
					<el-option label='0' value='0'></el-option>
					<el-option label='1' value='1'></el-option>
					<el-option label='2' value='2'></el-option>
					<el-option label='3' value='3'></el-option>
					<el-option label='4' value='4'></el-option>
					<el-option label='5' value='5'></el-option>
					<el-option label='6' value='6'></el-option>
					<el-option label='7' value='7'></el-option>
				</el-select>
			</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addDSCPDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addDSCPDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- Static Routing 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addStaticDialogShow" @close="closeAddStaticDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addStaticDialogForm" :model='addStaticDialogForm' :rules='addStaticDialogRules' label-position="top">
			<el-form-item prop='Static_IPVersion' style="min-width:400px;" label="IP Version" label-width="160px">
				<el-select v-model='addStaticDialogForm.Static_IPVersion'>
					<el-option label='IPv4' value='1'></el-option>
                    <el-option label='IPv6' value='2'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='Static_InterfaceName' style="min-width:400px;" label="Interface Name" label-width="160px">
				<el-select v-model='addStaticDialogForm.Static_InterfaceName' >
					<el-option label='opt' value='opt'></el-option>
					<el-option v-for="item in VlanNameList" :label='item' :value='item'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='Static_DestinationNetwork' style="min-width:400px;" label="Destination Network" label-width="160px">
                <el-input v-model.trim='addStaticDialogForm.Static_DestinationNetwork'></el-input>
			</el-form-item>
			<el-form-item v-show="['1'].includes(addStaticDialogForm.Static_IPVersion)" prop='Static_NetmaskPrefixLength' style="min-width:400px;" label="Subnet Mask" label-width="160px">
                <el-input v-model.trim='addStaticDialogForm.Static_NetmaskPrefixLength'></el-input>
			</el-form-item>
			<el-form-item v-show="['2'].includes(addStaticDialogForm.Static_IPVersion)" prop='Static_NetmaskPrefixLength' style="min-width:400px;" label="Prefix Length" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addStaticDialogForm.Static_NetmaskPrefixLength'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~128,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='Static_Gateway' style="min-width:400px;" label="Gateway" label-width="160px">
                <el-input v-model.trim='addStaticDialogForm.Static_Gateway' ></el-input>
			</el-form-item>
			
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addStaticDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addStaticDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gnbNetworkPage = new Vue({
	el: '#gnbNetworkPage', 
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
			},
            validatePonIpRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
                if(this.ruleForm.Pon_Switch == '1'){
                    if(value == '' || value == undefined || value == null){
                        if(isRequired){
                            if(rule.field == "Pon_StartIp"){
                                this.PonStartIpErrorShow = true;
                            }else{
                                this.PonEndIpErrorShow = true;
                            }
                            callback(new Error(mag))
                        }else{
                            if(rule.field == "Pon_StartIp"){
                                this.PonStartIpErrorShow = false;
                            }else{
                                this.PonEndIpErrorShow = false;
                            }
                            callback();
                        }
                    }else{
                        if(reg.test(value) && value >= min && value <= max){
                            if(rule.field == "Pon_StartIp"){
                                this.PonStartIpErrorShow = false;
                            }else{
                                this.PonEndIpErrorShow = false;
                            }
                            callback();
                        }else{
                            if(rule.field == "Pon_StartIp"){
                                this.PonStartIpErrorShow = true;
                            }else{
                                this.PonEndIpErrorShow = true;
                            }
                            callback(new Error(mag))
                        }
                    }
                }else{
                    callback();
                }
			},
			validateWAN_IP = (rule,value,callback) => {

				if(vm.addWANDialogForm.IPType == 'Static'){
                    if(vm.isValidIP(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
                    }
				}else if(vm.addWANDialogForm.IPType == 'Staticv6'){
                    if(vm.isIPv6(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
                    }
				}else{
                    callback();
                }
			},
			validateWAN_PrefixLengthSubnetMask = (rule,value,callback) => {

				if(vm.addWANDialogForm.IPType == 'Static'){
					if(vm.isMask(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
                    }
				}else if(vm.addWANDialogForm.IPType == 'Staticv6'){
                    if(vm.isNumeric(value)&&parseInt(value)>=0 && parseInt(value)<=128){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("FanWei")%>：0~128,Integer'))
                    }
				}else{
                    callback();
                }
			},
			validateWAN_Gateway = (rule,value,callback) => {

				if(value == '' || value == undefined || value == null){
					callback();
				}else if(vm.addWANDialogForm.IPType == 'Static'){
                    if(vm.isValidIP(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
                    }
				}else if(vm.addWANDialogForm.IPType == 'Staticv6'){
                    if(vm.isIPv6(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>'))
                    }
				}else{
                    callback();
                }
			},
			validateLAN_IP = (rule,value,callback) => {

				if(vm.isValidIP(value)){
					callback();
				}else{
					callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
				}
			},
			validateLAN_SubnetMask = (rule,value,callback) => {

				if(vm.isMask(value)){
					callback();
				}else{
					callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
				}
			},
			validateGateway = (rule,value,callback) => {
				if(value == '' || value == undefined || value == null){
					callback(new Error('<%=rb.getString("FanWei")%>：1~64,Digit string'))
				}else{
					callback();
				}
			},
			validateIPSec_KeyLeft = (rule,value,callback) => {
				var typeVal = vm.addIPSecDialogForm.IPSec_KeyLeftType,
					codes={
						's':31536000,
						'm':525600,
						'h':8760,
						'd':365
					}
					minNum = 1,
					maxNum = codes[typeVal];
				
				if(value == '' || value == undefined || value == null){
					callback('error');
				}else{
					if(vm.isNumeric(value)&&parseInt(value)>=minNum && parseInt(value)<=maxNum){
                        callback();
                    }else{
                        callback(new Error('error'))
                    }
				}
			},
			validateIPSec_IKELeftTime = (rule,value,callback) => {
				var typeVal = vm.addIPSecDialogForm.IPSec_IKELeftTimeType,
					codes={
						's':31536000,
						'm':525600,
						'h':8760,
						'd':365
					}
					minNum = 1,
					maxNum = codes[typeVal];
				
				if(value == '' || value == undefined || value == null){
					callback('error');
				}else{
					if(vm.isNumeric(value)&&parseInt(value)>=minNum && parseInt(value)<=maxNum){
                        callback();
                    }else{
                        callback(new Error('error'))
                    }
				}
			},
			validateIPSec_RekeyMargin = (rule,value,callback) => {
				var typeVal = vm.addIPSecDialogForm.IPSec_RekeyMarginType,
					codes={
						's':31536000,
						'm':525600,
						'h':8760,
						'd':365
					}
					minNum = 1,
					maxNum = codes[typeVal];
				
				if(value == '' || value == undefined || value == null){
					callback('error');
				}else{
					if(vm.isNumeric(value)&&parseInt(value)>=minNum && parseInt(value)<=maxNum){
                        callback();
                    }else{
                        callback(new Error('error'))
                    }
				}
			},
			validateIPSec_Dpddelay = (rule,value,callback) => {
				var typeVal = vm.addIPSecDialogForm.IPSec_DpddelayType,
					codes={
						's':31536000,
						'm':525600,
						'h':8760,
						'd':365
					}
					minNum = 1,
					maxNum = codes[typeVal];
				
				if(value == '' || value == undefined || value == null){
					callback('error');
				}else{
					if(vm.isNumeric(value)&&parseInt(value)>=minNum && parseInt(value)<=maxNum){
                        callback();
                    }else{
                        callback(new Error('error'))
                    }
				}
			},
			validateStatic_DestinationNetwork = (rule,value,callback) => {

				if(vm.addStaticDialogForm.Static_IPVersion == '1'){
					if(vm.isValidIP(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
                    }
				}else if(vm.addStaticDialogForm.Static_IPVersion == '2'){
                    if(vm.isIPv6(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>'))
                    }
				}else{
                    callback();
                }
			},
			validateStatic_NetmaskPrefixLength = (rule,value,callback) => {

				if(vm.addStaticDialogForm.Static_IPVersion == '1'){
					if(vm.isValidSubnetMask(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
					}
				}else if(vm.addStaticDialogForm.Static_IPVersion == '2'){
                    if(vm.isNumeric(value)&&parseInt(value)>=0 && parseInt(value)<=128){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("FanWei")%>：0~128,Integer'))
                    }
				}else{
                    callback();
                }
			},
			validateStatic_Gateway = (rule,value,callback) => {

				if(vm.addStaticDialogForm.Static_IPVersion == '1'){
                    if(vm.isValidIP(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
                    }
				}else if(vm.addStaticDialogForm.Static_IPVersion == '2'){
                    if(vm.isIPv6(value)){
                        callback();
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>'))
                    }
				}else{
					callback();
				}
			},
			validateIPaddress= (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				
				if(value === ''){
					callback()
				}else{
					if(reg.test(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}
				}
			},
			validateNRARFCNDL = (rule,value,callback) => {
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
				}else{
					if(vm.isInteger(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
					}
				}
			},
			validateNRARFCNUL = (rule,value,callback) => {
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
				}else{
					if(vm.isInteger(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
					}
				}
			},
			validateSoft_KeyAndOpc= (rule,value,callback) => {
				//var reg = /^([0-9a-zA-Z]{2}\s[0-9a-zA-Z]{2})(\s([0-9a-zA-Z]{2}\s[0-9a-zA-Z]{2}))*$/;
				var reg = /^[0-9a-zA-Z]{2}([0-9a-zA-Z]{2})*$/;
				
				if(value === '' || value === null || value === undefined){
					callback();
				}else{
					if(reg.test(value)){
						callback();
					}else{
						callback(new Error('error'))
					}
				}
			},
			validateSoft_IMSI= (rule,value,callback) => {
				var reg = /^[0-9a-zA-Z]{2}(\s[0-9a-zA-Z]{2})*$/;
				
				if(value === '' || value === null || value === undefined){
					callback()
				}else{
					callback();
				}
			},
			validateWAN_VlanName= (rule,value,callback) => {
				if(value === '' || value === null || value === undefined){
					if(this.addWANDialogForm.VlanID){
						callback(new Error('error'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateWAN_VlanID= (rule,value,callback) => {
				if(value === '' || value === null || value === undefined){
					if(this.addWANDialogForm.VlanName){
						callback(new Error('error'))
					}else{
						callback();
					}
				}else{
					if(vm.isNumeric(value)&&parseInt(value)>=2 && parseInt(value)<=4094){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%>：2~4094,Integer'))
					}
				}
			},
            validatePonSubnetMask = (rule,value,callback) => {
                if(this.ruleForm.Pon_Switch == '1'){
                    if(value === '' || value === null || value === undefined){
                        callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
                    }else{
                        if(vm.isMask(value)){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
                        }
                    }
                }else{
                    callback();
                }
			};
		return {
			activeCollapse:['WAN','defaultRoute','IPSEC','DSCP','Static'],
			rowDataInfo: [],
			smallCellCode:'',
			ruleForm:{
				WANList:[],
				LANList:[],
				IPSecTunnelList:[],
				DSCPList:[],
				StaticList:[],

                Pon_Switch: '0',
                Pon_StartIp:'',
                Pon_EndIp:'',
                Pon_SubnetMask:'',

				defaultRouteVal:'',
				defaultRouteDnsStr:'',
                dnsType:'1',
                dhcpDnsStr:'',

				NGAP_InterfaceBinding:'',
				NGU_InterfaceBinding:'',

				Swan_IKEDebugLevel:'1',
				Swan_ESPDebugLevel:'1',
				Swan_CFGDebugLevel:'1',
				Swan_KNLDebugLevel:'1',
				Swan_MGRDebugLevel:'1',
				Swan_ASNDebugLevel:'1',
				Swan_CHDDebugLevel:'1',
				Swan_LIBDebugLevel:'1',
				Swan_Port:'',
				Swan_PortNATT:'',
				Swan_RetryInitiateInterval:'',
				Swan_MTU:'',
				Swan_MSS:'',

				Soft_Enable:'0',
				Soft_IMSI:'',
				Soft_IpsecUsimAuthenticationEnable:'0',
				Soft_Key:'',
				Soft_Opc:'',

				NGAP_DSCP:'',
				NGAP_VLANPriority:'',
				X2AP_DSCP:'',
				X2AP_VLANPriority:'',
				F1AP_DSCP:'',
				F1AP_VLANPriority:'',
				XNAP_DSCP:'',
				XNAP_VLANPriority:'',
				OAM_DSCP:'',
				OAM_VLANPriority:'',

			},
			rules:{
                Pon_StartIp:[
                    {validator:validatePonIpRange,min:1,max:254,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~254,Integer'}
                ],
                Pon_EndIp:[
                    {validator:validatePonIpRange,min:1,max:254,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~254,Integer'}
                ],
                Pon_SubnetMask:[
                    {validator:validatePonSubnetMask}
                ],
				Swan_Port:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:65535,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer'}
				],
				Swan_PortNATT:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:65535,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer'}
				],
				Swan_RetryInitiateInterval:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:65535,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer'}
				],
				Swan_MTU:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:9600,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~9600,Integer'}
				],
				Swan_MSS:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:9600,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~9600,Integer'}
				],
				Soft_IMSI:[
					{validator:validateSoft_IMSI}
				],
				Soft_Key:[
					{validator:validateSoft_KeyAndOpc}
				],
				Soft_Opc:[
					{validator:validateSoft_KeyAndOpc}
				],
				NGAP_DSCP:[
					{required:true,message:'<%=rb.getString("FanWei")%>：0~63,Integer',trigger:'blur'},
					{validator:validateRange,min:0,max:63,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~63,Integer'}
				],
				X2AP_DSCP:[
					{required:true,message:'<%=rb.getString("FanWei")%>：0~63,Integer',trigger:'blur'},
					{validator:validateRange,min:0,max:63,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~63,Integer'}
				],
				F1AP_DSCP:[
					{required:true,message:'<%=rb.getString("FanWei")%>：0~63,Integer',trigger:'blur'},
					{validator:validateRange,min:0,max:63,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~63,Integer'}
				],
				XNAP_DSCP:[
					{required:true,message:'<%=rb.getString("FanWei")%>：0~63,Integer',trigger:'blur'},
					{validator:validateRange,min:0,max:63,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~63,Integer'}
				],
				OAM_DSCP:[
					{required:true,message:'<%=rb.getString("FanWei")%>：0~63,Integer',trigger:'blur'},
					{validator:validateRange,min:0,max:63,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~63,Integer'}
				],
			},
			casts:{
				'457A114D1D07C6036188007A37BEAEA1':'WANList',
				'66666666666666666666666666666666':'WANAndVlan_idx',
				'06650412C692D1B302433E406841694E':'InterfaceIndex',
				'8F111142BD809F7CA2B32EBB54CEE70A':'InterfaceType',
				'23B1161BD6DCBD171DA202EDC5D011E6':'VlanInterfaceIndex',
				
				'1DF5350791700C07E781D6C3CDE81535':'WANOrLan_IPv4idx',
				'F420F84CED364E05A86BE443A8418251':'WANOrLan_IPv4IPType',
				'845C37430C11BB52A96753BF482B7DDA':'WANOrLan_IPv4PortType',
				'021427876339CC353AED02B060546ED3':'WANOrLan_IPv4Ip',
				'4F7E22BEDDE41BD91A78FAD46C376719':'WANOrLan_IPv4SubnetMask',
				'5583C3F91A69306F55AA451EE7846C09':'WANOrLan_IPv4Gateway',
				
				'3CBBC67859090B96553284D855512C6E':'VLAN_IPv4idx',
				'C5631F9C39C8DE078EB303374BFA2A4B':'VLAN_IPv4IPType',
				'FAE105CC4D4AFA9A366B08338BF0FE0D':'VLAN_IPv4PortType',
				'EA3441AF5417DB38BF55973EF9A5D203':'VLAN_IPv4Ip',
				'4900F5F83B62B79F584AC596518D6D91':'VLAN_IPv4SubnetMask',
				'973644AC08A413802F39316FC276E23A':'VLAN_IPv4Gateway',

				'BB3013AB03782956694D036AA65D27A2':'WANOrLan_IPv6idx',
				'CCA4B0D542F93AAECC285CBFDFEFE4F5':'WANOrLan_IPv6IPType',
				'E22E7D959AF30593A4646DEA53107B65':'WANOrLan_IPv6PortType',
				'90D878462BAED1573DA27214180E0388':'WANOrLan_IPv6Ip',
				'99AFF0E2045DDEC3DC04682EE6CDDD21':'WANOrLan_IPv6SubnetMask',
				'5AB5A3E6CB1BDB114DAFF4F84FD493B0':'WANOrLan_IPv6Gateway',

				'833BC24E9DA34CEA3146FA6F7D95A05E':'VLAN_IPv6idx',
				'527EE538B2E53F8E6E96C3F8165764F4':'VLAN_IPv6IPType',
				'9BCCAEA76FD707512286A30FF277FAF8':'VLAN_IPv6PortType',
				'C4069601A8037589F1F313882E4F727F':'VLAN_IPv6Ip',
				'923186445E97CDE9F88850CE4DA1243F':'VLAN_IPv6SubnetMask',
				'919B5F1691AE6FE47DB975F90CCEC9E6':'VLAN_IPv6Gateway',

				'831396135055560A860153BEAE8BC7D5':'VlanID',
				'CD97DA72A6C212E49BF9D73748878DDB':'VlanName',

				'B1CEF67B08045AFBD9877D3D52AB5CEC':'WAN_Pppoeidx',
				'290803D1A4DCCCE6E2ABEE6A148149AA':'WAN_PppoeIPType',
				'F3F7B44449630CB5443610F23F1D262A':'WAN_PppoeIp',
				'85AB02B420089903CB7EFDB96F04870B':'WAN_PppoeSubnetMask',
				'13BE361FF34CB19A02D3502D59E4D36E':'WAN_PppoeConnectStatus',
				'F94154DB6F69D86B7B153B3DABAFBE24':'WAN_PppoeDiallingMethod',
				'24524CD7268B248A15E35D390F34226F':'WAN_PppoePassword',
				'71EEAAFD6796F8FC3483F492E9BC603E':'WAN_PppoeUserName',

				'234794964226E6E57FD79F8540752AD5':'VLAN_Pppoeidx',
				'454E6F732F8E72248E2C02A8D7CDA88D':'VLAN_PppoeIPType',
				'D2B57F4CA13EDE674925BD9BBB4359C5':'VLAN_PppoeIp',
				'CD4FC5FDDB6CE2578D9280EAFB693AA3':'VLAN_PppoeSubnetMask',
				'FFC493C035362F7F4A48797DA6A24F3B':'VLAN_PppoeConnectStatus',
				'BAC9AE0366FF11FD6F0045B14C9DC562':'VLAN_PppoeDiallingMethod',
				'BFD7D45E4779299E3B4639E684016E5F':'VLAN_PppoePassword',
				'BBBECFFFA4CB48BBBEA45862AE9A54F5':'VLAN_PppoeUserName',

				'457A114D1D07C6036188007A37BEAEA1':'LANList',
				'E94B26E20AA7A4BB3872509ACCB5051E':'LAN_Status',
				'0E206B5978532EB568A410D5AD876BC8':'LAN_Mac',
				'A80BEBB824986146922DEB11190A2DC9':'defaultRouteVal',
				'46C9AB6746C10540FC9CCAF73DFFBCBF':'defaultRouteDnsStr',
                'C036F1C4491EABCCDE277D08806C14F5':'dnsType',
				'2AC410FA71B3F333A5DECDBF6D71125A':'dhcpDnsStr',

                '75548D7E66D6B11E8175594609C8E240':'Pon_Switch',
                'BFADBD2B2108F71D6813A0D18D4650E5':'Pon_StartIp',
                'BCBD6A522AC74B8C8321723FF44D3AEB':'Pon_EndIp',
                '94AADE6E7878A865506942CE68501628':'Pon_SubnetMask',

				'52AFE54CE3E95BEE52C579028B0559A1':'NGAP_InterfaceBinding',
				'F6AE3D03B7C71D80A95D9EF530AFB5D0':'NGU_InterfaceBinding',

				'8F8761D6C26CC1B2F8E2B752E80E2EC8':'IPSecTunnelList',
				'7D2E02C4BC8C0D70AA98224672591B17':'IPSec_idx',
				'9E3DD4E6FA1978E34E03149D698A22A5':'IPSec_Status',
				'BAA45B2339EB63EBAD0CC37E13D3E0C6':'IPSec_IP',
				'B0FC380E39FE86C0B6F952CD6EEE8BC8':'IPSec_Switch',
				'E8D2015628E1CFD44B279484A83D8EA4':'IPSec_TunnelName',
				'50787853AA08953168D1EC66D8E289E4':'IPSec_LeftAuth',
				'9EEC4808040B506E1FB323EB88BB4FFD':'IPSec_RightAuth',
				'3FBA457EE543252119C79B04F55484BA':'IPSec_Gateway',
				'3A418D4F65FF596A0471A5F3B042B95C':'IPSec_RightSubnet',
				'1D86A78274750EAF0F08511100A81225':'IPSec_RightID',
				'8C7BC77E331D2750B671C0243E29EAC2':'IPSec_SecretKey',
				'944AB0E112D7F24EEA59E9399CEA9A72':'IPSec_LeftID',
				'C6733C5F3B5A21419B7B5AF4CC867417':'IPSec_LeftCert',
				'B1CE025AF9144E06CB00805CD071061C':'IPSec_LeftSourceIp',
				'50C57E75D2D0B095856DEE32B07AE105':'IPSec_LeftSubnet',
				'8CF34626CF9D1D8D274952AF4BA05892':'IPSec_Fragmentation',
				'4B303A948652CCA94FB72BB9CFCB8B9A':'IPSec_IKEEncryption',
				'DC3D854E0DF8E7C3CD43E96C0577BC5F':'IPSec_IKEDHGroup',
				'6CE25DF7E5697E15065BAC435BE260A7':'IPSec_IKEAuthentication',
				'F34A7A8574B810CD813C0E6057E4212D':'IPSec_ESPEncryption',
				'037252587AE6E077AECB7ACD45CE83CB':'IPSec_ESPDHGroup',
				'85807A1BA9A12C3F44431BEE630715B7':'IPSec_ESPAuthentication',
				'2118B01ED6EFE01DCD369A0B6DFFE8A3':'IPSec_KeyLeft',
				'DC4B11F1FCDF28395770DAF41E5EB7E3':'IPSec_IKELeftTime',
				'2FBE47F411114C7FF894449BE5932123':'IPSec_RekeyMargin',
				'AB8BCA5F4DD6A65CB0E301B61CF45790':'IPSec_Dpdaction',
				'2993A539232F031538579407EB61FEBC':'IPSec_Dpddelay',
				'511257C4C8862A352EFBDA9DBEA7B14F':'IPSec_LeftInterface',

				'B12EC17761B1E2089CF2BEB44C65766B':'Swan_IKEDebugLevel',
				'4D351ED5AC7DF26216E4A4310F229BA3':'Swan_ESPDebugLevel',
				'0822FF129F98D5502E883DB08270803E':'Swan_CFGDebugLevel',
				'E2FECCB43FDB772C5338566F8D897BAE':'Swan_KNLDebugLevel',
				'744A5B4484223AC33D9A19C74628DB6C':'Swan_MGRDebugLevel',
				'9FAB72B75A8113BF7642BF7489BD41DF':'Swan_ASNDebugLevel',
				'C11CC1BD4D2B2B1C2E0A5D8438EB640E':'Swan_CHDDebugLevel',
				'F5D3EBE46375F9015E07906425A9D259':'Swan_LIBDebugLevel',
				'0D7D34BD164AA9E16F14191876B2F62F':'Swan_Port',
				'F95F49010D6DEC81D0EE020C3AA75718':'Swan_PortNATT',
				'70FA3C16DFA719BEA40A40A3F0ACAFEF':'Swan_RetryInitiateInterval',
				'E3B40748F02DDE8E8EAC474EE917E066':'Swan_MTU',
				'A38AE04B8350D87CD0D8C687C45659C9':'Swan_MSS',

				'93AEF882E59F86498FF056989DCE3AE4':'Soft_Enable',
				'695C4C8E3F4F30E624ECEE7A40620E11':'Soft_IMSI',
				'6FBD3C88818A5C1DED9AA662EFC302C1':'Soft_IpsecUsimAuthenticationEnable',
				'69EED7DF192B2E7A250D20270C4AE825':'Soft_Key',
				'B77E574C5A4978FC0E9533EFFFC59A79':'Soft_Opc',

				'FDFED36262A1DBEEBA480BD4BEF4C695':'DSCPList',
				'85096C68DB9AC1F415FD3D450E74417C':'DSCP_idx',
				'6937A261AE3683E224BECCF951B6C411':'DSCP_DSCPVal',
				'68948E26A8F15B6990863D534D91ACEA':'DSCP_VLANPriority',

				'BBCB549AF96947C12044CE58CB5C72DB':'NGAP_DSCP',
				'75F5D5D57F408B70DB730EF676D69290':'NGAP_VLANPriority',
				'55D9DA1CBE587DD12814AC1A4C5BAA62':'X2AP_DSCP',
				'8DEEDD97F4B28810EFF13E1965C1EDFD':'X2AP_VLANPriority',
				'0E72EB54795174DBC1E810A1BA4C38CB':'F1AP_DSCP',
				'B1EC8290AA9D3A36783638C0DC1B401B':'F1AP_VLANPriority',
				'C03969B1228E32F54B0DD8EE92645CD2':'XNAP_DSCP',
				'61E62BC0679FCA9B1DDA16D537140AC8':'XNAP_VLANPriority',
				'54061E5AA0CAE0D1755843127D42EA63':'OAM_DSCP',
				'09ADEC6A1600EA4F9305339B92ABF61C':'OAM_VLANPriority',

				'98D653703FA37343F7EC240B37611D7F':'StaticList',
				'B251632364D9CE89B33E5B0D863116EA':'Static_idx',
				'F595867F6998489F55638B392D1EC951':'Static_IPVersion',
				'36450F922D73413549BAED858A9B01F2':'Static_DestinationNetwork',
				'A64F15FD74E4B3128E76A060CF55AD5B':'Static_NetmaskPrefixLength',
				'6B3D0DF02A9899CB93B0E07E66D6ABEA':'Static_Gateway',
				'F0CEB6E203B264658E5A3E52B055DFDF':'Static_InterfaceName',
			},
			defaultRoute:{
				Dns:'',
				defaultRouteDnsErrorMessage:'',
			},
			defaultRouteDnsList:[],
            dhcpDnsList:[],
			optType:'',
			tbType:'',
			wanInterfaceIndex:'',
			lanInterfaceIndex:'',
			addWANDialogShow:false,
			oldWanSelectRow:{},
			wanIPv4MatchAddParams:{
				'IPType':'WANOrLan_IPv4IPType',
				'PortType':'WANOrLan_IPv4PortType',
				'IP':'WANOrLan_IPv4Ip',
				'SubnetMask':'WANOrLan_IPv4SubnetMask',
				'Gateway':'WANOrLan_IPv4Gateway',
				'VlanID':'VlanID',
				'VlanName':'VlanName',
				'operateType':'operateType',
			},
			vlanIPv4MatchAddParams:{
				'IPType':'VLAN_IPv4IPType',
				'PortType':'VLAN_IPv4PortType',
				'IP':'VLAN_IPv4Ip',
				'SubnetMask':'VLAN_IPv4SubnetMask',
				'Gateway':'VLAN_IPv4Gateway',
				'VlanID':'VlanID',
				'VlanName':'VlanName',
			},
			wanIPv6MatchAddParams:{
				'IPType':'WANOrLan_IPv6IPType',
				'PortType':'WANOrLan_IPv6PortType',
				'IP':'WANOrLan_IPv6Ip',
				'SubnetMask':'WANOrLan_IPv6SubnetMask',
				'Gateway':'WANOrLan_IPv6Gateway',
				'VlanID':'VlanID',
				'VlanName':'VlanName',
				'operateType':'operateType',
			},
			vlanIPv6MatchAddParams:{
				'IPType':'VLAN_IPv6IPType',
				'PortType':'VLAN_IPv6PortType',
				'IP':'VLAN_IPv6Ip',
				'SubnetMask':'VLAN_IPv6SubnetMask',
				'Gateway':'VLAN_IPv6Gateway',
				'VlanID':'VlanID',
				'VlanName':'VlanName',
			},
			wanPppoeMatchAddParams:{
				'IPType':'WAN_PppoeIPType',
				'IP':'WAN_PppoeIp',
				'SubnetMask':'WAN_PppoeSubnetMask',
				'ConnectStatus':'WAN_PppoeConnectStatus',
				'DiallingMethod':'WAN_PppoeDiallingMethod',
				'Password':'WAN_PppoePassword',
				'UserName':'WAN_PppoeUserName',
				'VlanID':'VlanID',
				'VlanName':'VlanName',
				'operateType':'operateType',
			},
			vlanPppoeMatchAddParams:{
				'IPType':'VLAN_PppoeIPType',
				'IP':'VLAN_PppoeIp',
				'SubnetMask':'VLAN_PppoeSubnetMask',
				'ConnectStatus':'VLAN_PppoeConnectStatus',
				'DiallingMethod':'VLAN_PppoeDiallingMethod',
				'Password':'VLAN_PppoePassword',
				'UserName':'VLAN_PppoeUserName',
				'VlanID':'VlanID',
				'VlanName':'VlanName',
				'operateType':'operateType',
			},
			addWANDialogForm:{
				IPType:'DHCP',
				PortType:'NG',
                IP:'',
                SubnetMask:'',
                Gateway:'',
                VlanID:'',
				VlanName:'',
				InterfaceType:'wan',
				DiallingMethod:'',
				UserName:'',
				Password:'',
			},
			addWANDialogRules:{
				IP:[{validator:validateWAN_IP,trigger:'blur'}],
				SubnetMask:[{validator:validateWAN_PrefixLengthSubnetMask,trigger:'blur'}],
				Gateway:[{validator:validateWAN_Gateway,trigger:'blur'}],
				VlanName:[{validator:validateWAN_VlanName}],
				VlanID:[
					{validator:validateWAN_VlanID,mag:'<%=rb.getString("FanWei")%>：2~4094,Integer'}
				],
			},
			addLANDialogShow:false,
			addLANDialogForm:{
				WANOrLan_IPv4Ip:'',
				WANOrLan_IPv4SubnetMask:'',
				InterfaceType:'lan',
			},
			addLANDialogRules:{
				WANOrLan_IPv4Ip:[{required:true,message:'<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>',trigger:'blur'},{validator:validateLAN_IP,trigger:'blur'}],
				WANOrLan_IPv4SubnetMask:[
					{required:true,message:'<%=rb.getString("QingShuRuHeFaDeYanMa")%>',trigger:'blur'},
					{validator:validateLAN_SubnetMask,trigger:'blur'}
				],
			},
			addIPSecTunnelDialogShow:false,
			activeIPSecTunnelCollapse:['Basic','Advance'],
			addIPSecDialogForm:{
				IPSec_Switch:'0',
				IPSec_TunnelName:'',
				IPSec_LeftAuth:'psk',
				IPSec_RightAuth:'psk',
				IPSec_Gateway:'10.10.10.10',
				IPSec_RightSubnet:'0.0.0.0/0',
				IPSec_RightID:'C=CH,O=strongSwan,CN=server',
				IPSec_SecretKey:'clientKey.der',
				IPSec_LeftID:'C=CH,O=strongSwan,CN=server',
				IPSec_LeftCert:'',
				IPSec_LeftSourceIp:'%config',
				IPSec_LeftSubnet:'',
				IPSec_Fragmentation:'yes',
				IPSec_IKEEncryption:'aes128',
				IPSec_IKEDHGroup:'modp1024',
				IPSec_IKEAuthentication:'sha256',
				IPSec_ESPEncryption:'aes128',
				IPSec_ESPDHGroup:'modp1024',
				IPSec_ESPAuthentication:'sha256',
				IPSec_KeyLeft:'360',
				IPSec_KeyLeftType:'d',
				IPSec_IKELeftTime:'360',
				IPSec_IKELeftTimeType:'d',
				IPSec_RekeyMargin:'5',
				IPSec_RekeyMarginType:'m',
				IPSec_Dpdaction:'restart',
				IPSec_Dpddelay:'30',
				IPSec_DpddelayType:'s',
				IPSec_LeftInterface:'None',
			},
			addIPSecDialogRules:{
				IPSec_TunnelName:[{required:true,message:'error',trigger:'blur'}],
				IPSec_Gateway:[{validator:validateGateway,trigger:'blur'}],
				IPSec_KeyLeft:[
					{required:true,message:'error',trigger:'blur'},
					{validator:validateIPSec_KeyLeft,trigger:'blur'}
				],
				IPSec_IKELeftTime:[
					{required:true,message:'error',trigger:'blur'},
					{validator:validateIPSec_IKELeftTime,trigger:'blur'}
				],
				IPSec_RekeyMargin:[
					{required:true,message:'error',trigger:'blur'},
					{validator:validateIPSec_RekeyMargin,trigger:'blur'}
				],
				IPSec_Dpddelay:[
					{required:true,message:'error',trigger:'blur'},
					{validator:validateIPSec_Dpddelay,trigger:'blur'}
				],
			},
			addDSCPDialogShow:false,
			addDSCPDialogForm:{
				DSCP_DSCPVal:'',
				DSCP_VLANPriority:''
			},
			addDSCPDialogRules:{
				DSCP_DSCPVal:[
					{required:true,message:'<%=rb.getString("FanWei")%>：0~63,Integer',trigger:'blur'},
					{validator:validateRange,min:0,max:63,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~63,Integer'}
				],
			},
			addStaticDialogShow:false,
			addStaticDialogForm:{
				Static_IPVersion:'1',
				Static_InterfaceName:'opt',
				Static_DestinationNetwork:'',
				Static_NetmaskPrefixLength:'',
				Static_Gateway:'',
				
			},
			addStaticDialogRules:{
				Static_DestinationNetwork:[
					{required:true,message:'<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>',trigger:'blur'},
					{validator:validateStatic_DestinationNetwork,trigger:'blur'}
				],
				Static_NetmaskPrefixLength:[
					{required:true,message:'<%=rb.getString("QingShuRuHeFaDeYanMa")%>',trigger:'blur'},
					{validator:validateStatic_NetmaskPrefixLength,trigger:'blur'}
				],
				Static_Gateway:[
					{required:true,message:'<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>',trigger:'blur'},
					{validator:validateStatic_Gateway,trigger:'blur'}
				]
			},
			PortTypeList:['NG','OAM','NGU','NG/OAM','NG/NGU','OAM/NGU','NG/OAM/NGU','Other'],
            PonStartIpErrorShow:false,
            PonEndIpErrorShow:false,
		};
	},
	computed: {
		gnbConfigAddDialogTitle(){
			return this.optType == 'add' ? '<%=rb.getString("TianJia")%>' : '<%=rb.getString("XiuGai")%>'
		},
		addIPSecTunnelBtnShow(){
			var arr = this.ruleForm.IPSecTunnelList.filter((item)=>{
				return  !item.operateType || (item.operateType &&item.operateType != 'remove')
			})
			return arr.length<3? true : false;
		},
		VlanIdList(){
			var arr = [];
			this.ruleForm.WANList.filter((item)=>{
				if(!item.operateType || (item.operateType &&item.operateType != 'remove')){
					var isExist = arr.some(items =>items == item.VlanID);
					if(!isExist && item.VlanID){
						arr.push(item.VlanID)
					}
				}  
			})
			return arr
		},
		VlanNameList(){
			var arr = [];
			this.ruleForm.WANList.filter((item)=>{
				if(!item.operateType || (item.operateType &&item.operateType != 'remove')){
					var isExist = arr.some(items =>items == item.VlanName);
					if(!isExist && item.VlanName){
						arr.push(item.VlanName)
					}
				}  
			})
			return arr
		},
		WanIpAddressList(){
			var arr = [];
			this.ruleForm.WANList.filter((item)=>{
				if(item.VlanID &&  (!item.operateType || (item.operateType &&item.operateType != 'remove'))){
					var ipStr = '';
					if(item.VLAN_PppoeIPType && item.VLAN_PppoeIPType == 'PPPoE'){
						ipStr = item.VLAN_PppoeIp ? item.VLAN_PppoeIp : '';
					}else if(item.VLAN_IPv4IPType && (item.VLAN_IPv4IPType == 'DHCP' || item.VLAN_IPv4IPType == 'Static')){
						ipStr = item.VLAN_IPv4Ip ? item.VLAN_IPv4Ip : '';
					}else if(item.VLAN_IPv6IPType && (item.VLAN_IPv6IPType == 'DHCPv6' || item.VLAN_IPv6IPType == 'Staticv6')){
						ipStr = item.VLAN_IPv6Ip ? item.VLAN_IPv6Ip : '';
					}
					var isExist = arr.some(items =>items == ipStr);
					if(!isExist && ipStr){
						arr.push(ipStr)
					}
				}  
			})
			return arr
		},
		tunnelNameList(){
			var arr = [];
			this.ruleForm.IPSecTunnelList.filter((item)=>{
				if(!item.operateType || (item.operateType &&item.operateType != 'remove')){
					arr.push(item.IPSec_TunnelName)
				}  
			})
			return arr
		},
		defaultRouteList(){
			var arr = ['eth','opt'];
			this.ruleForm.WANList.filter((item)=>{
				if(item.VlanID &&  (!item.operateType || (item.operateType &&item.operateType != 'remove'))){
					var isExist = arr.some(items =>items == item.VlanName);
					if(!isExist){
						arr.push(item.VlanName)
					}
				}  
			})
			return arr
		},
		addWanDisVlanId(){
			return this.optType == 'edit' &&  (!this.addWANDialogForm.operateType || (this.addWANDialogForm.operateType && this.addWANDialogForm.operateType == 'edit'))
		},
		addVlanIdBtnShow(){
			return this.optType == 'add' ||  (this.optType == 'edit' && this.addWANDialogForm.operateType && this.addWANDialogForm.operateType == 'add')
		},
		ipv4SeclectShow(){
			return this.optType == 'add' || (this.optType == 'edit' && (this.addWANDialogForm.IPType == 'DHCP' ||  this.addWANDialogForm.IPType == 'Static'))
		},
		ipv6SeclectShow(){
			return this.optType == 'add' || (this.optType == 'edit' && (this.addWANDialogForm.IPType == 'DHCPv6' ||  this.addWANDialogForm.IPType == 'Staticv6'))
		},
        ponStartIpNoLessEndIpErrShow(){
            return parseInt(this.ruleForm.Pon_StartIp) >= parseInt(this.ruleForm.Pon_EndIp)
        },
        dnsTypeList(){
            var arr = ['VLAN_IPv4IPType','VLAN_IPv6IPType','WANOrLan_IPv4IPType','WANOrLan_IPv6IPType'],
                dnsTypeList = [{value: '1',label:'Static'}],
                isExistDhcp = false;
            this.ruleForm.WANList.map((item)=>{
				if((!item.operateType || (item.operateType &&item.operateType != 'remove'&& item.operateType != 'add'))){
					arr.map((key)=>{
                        if(item[key] == 'DHCP' || item[key] == 'DHCPv6'){
                            isExistDhcp = true;
                        }
                    })
				}  
			})
            if(isExistDhcp){
                dnsTypeList.push({value: '0',label:'DHCP'})
            }
            return dnsTypeList
        }
	},
	methods: {
		init(row){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = row.small_cell_code;

			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'23001');

		},
		getParamData(code,id) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
				params = {
					id: id,
					cellIndex:'1',
					smallCellCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				vm.resetFormData();
				if(data && Array.isArray(data)) {
					data.map(function(item){
						item.groups.map(function(group){
							group.list.map(function(m){
								if(m.type == 'list'){
									vm.initTable(m.url,m.label);
								}else{
									codes.push(m.name);
									// 执行赋值
									vm.setValue(m);
								}
							});
						});
					});
					if(vm.$refs.ruleForm){
						initForm(vm.$refs.ruleForm);
					}
				}
			});
		},
		// 映射赋值
		setValue(item) {
			var vm = this,
			code = item.name,
			value = item.value;

			// indexs是否含有
				var key = vm.casts[code];
			try{
				if(key){
					if(key == 'defaultRouteDnsStr' && value){
						vm.defaultRouteDnsList = value.split(',');
					}
                    if(key == 'dhcpDnsStr' && value){
						vm.dhcpDnsList = value.split(',');
					}
					if(key == 'Soft_Key' || key == 'Soft_Opc'){
						value = value.replace(/\s*/g,'');
					}
                    if((key == 'Pon_StartIp' || key == 'Pon_EndIp') && value){
                        var arr = value.split('.');
                        value = arr[3];
                    }
					vm.ruleForm[key] = value;
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					smallCellCode : vm.smallCellCode
			}
            if(url== '' || url == null || url == undefined)return
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							codes.push(key);
							obj[vm.casts[key]] = item[key]
						}
						if(type == 'wan/vlan List'){
							var wanObj = {
									InterfaceType:obj.InterfaceType
								},
								lanObj = {
									InterfaceType:obj.InterfaceType
								};
							if(obj.InterfaceType == 'wan'){
								Object.keys(obj).map((key)=>{
									if(key.slice(0,3) !== 'LAN'){
										wanObj[key] = obj[key];
									}
								})
								vm.wanInterfaceIndex = wanObj.InterfaceIndex;
								var IPv4params = {
									InterfaceIndex:wanObj.InterfaceIndex,
									smallCellCode: vm.smallCellCode
								}
								axios.post('${ctx}/cell/quicksettings/getListParamValueForWanAndVlan.action?parent_id=236014,236015,236012,236016,236017,236018,236019&platform=BaiBNQ',stringify(IPv4params)).then(res=>{
									var IPv4Data = res.data;
									if(IPv4Data.rows){
										IPv4Data.rows.map(IPv4Item=>{
											var IPv4Obj = {};
											for(var key in IPv4Item){
												IPv4Obj[vm.casts[key]] = IPv4Item[key]
											}
											Object.keys(wanObj).map((key)=>{
												IPv4Obj[key] = wanObj[key]
											})
											vm.ruleForm.WANList.push(IPv4Obj);
										})
										if(vm.$refs.ruleForm){
											initForm(vm.$refs.ruleForm);
										}
									}
								})
							}else  if(obj.InterfaceType == 'lan'){
								Object.keys(obj).map((key)=>{
									if(key.slice(0,3) !== 'WAN' || key.slice(0,8) == 'WANOrLan'){
										if(key != 'WANOrLan_IPv4IPType' && key != 'WANOrLan_IPv4PortType' && key != 'WANOrLan_IPv4Gateway'){
											lanObj[key] = obj[key];
										}
									}
								})
								vm.lanInterfaceIndex = lanObj.InterfaceIndex;
								var IPv4params = {
									InterfaceIndex:lanObj.InterfaceIndex,
									smallCellCode: vm.smallCellCode
								}
								axios.post('${ctx}/cell/quicksettings/getListParamValue.action?parent_id=236014&platform=BaiBNQ',stringify(IPv4params)).then(res=>{
									var IPv4Data = res.data;
									if(IPv4Data.rows){
										IPv4Data.rows.map(IPv4Item=>{
											var IPv4Obj = {};
											for(var key in IPv4Item){
												IPv4Obj[vm.casts[key]] = IPv4Item[key]
											}
											Object.keys(IPv4Obj).map((key)=>{
												if(key.slice(0,3) == 'LAN' || key.slice(0,8) == 'WANOrLan'){
													if(key != 'WANOrLan_IPv4IPType' && key != 'WANOrLan_IPv4PortType' && key != 'WANOrLan_IPv4Gateway'){
														lanObj[key] = IPv4Obj[key]
													}
													
												}
											})
											var lanObjItem =  JSON.parse(JSON.stringify(lanObj));
											vm.ruleForm.LANList.push(lanObjItem);
										})
										if(vm.$refs.ruleForm){
											initForm(vm.$refs.ruleForm);
										}
									}
								})
							}
						}else if(type == 'IPSec List'){
							vm.ruleForm.IPSecTunnelList.push(obj);
						}else if(type == 'Dscp List'){
							vm.ruleForm.DSCPList.push(obj);
						}else if(type == 'Static Routing List'){
							if(obj.Static_IPVersion == '1'){
								obj.Static_NetmaskPrefixLength = obj.Static_NetmaskPrefixLength ? vm.numberConvertMask(obj.Static_NetmaskPrefixLength) : '';
							}
							vm.ruleForm.StaticList.push(obj);
						}
					})
					if(vm.$refs.ruleForm){
						initForm(vm.$refs.ruleForm);
					}
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					WANList:[],
					LANList:[],
					IPSecTunnelList:[],
					DSCPList:[],
					StaticList:[],

					NGAP_InterfaceBinding:'',
					NGU_InterfaceBinding:'',

					Swan_IKEDebugLevel:'1',
					Swan_ESPDebugLevel:'1',
					Swan_CFGDebugLevel:'1',
					Swan_KNLDebugLevel:'1',
					Swan_MGRDebugLevel:'1',
					Swan_ASNDebugLevel:'1',
					Swan_CHDDebugLevel:'1',
					Swan_LIBDebugLevel:'1',
					Swan_Port:'',
					Swan_PortNATT:'',
					Swan_RetryInitiateInterval:'',
					Swan_MTU:'',
					Swan_MSS:'',

					Soft_Enable:'0',
					Soft_IMSI:'',
					Soft_IpsecUsimAuthenticationEnable:'0',
					Soft_Key:'',
					Soft_Opc:'',

					NGAP_DSCP:'',
					NGAP_VLANPriority:'',
					X2AP_DSCP:'',
					X2AP_VLANPriority:'',
					F1AP_DSCP:'',
					F1AP_VLANPriority:'',
					XNAP_DSCP:'',
					XNAP_VLANPriority:'',
					OAM_DSCP:'',
					OAM_VLANPriority:'',
				};
			Object.assign(vm.ruleForm,params);
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
		// 打开新增 WAN弹窗
		addWANDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				var selectObj = JSON.parse(JSON.stringify(row));
				Object.keys(vm.addWANDialogForm).forEach(function(key){
					if(selectObj.VlanID){
						if(selectObj.VLAN_PppoeIPType && selectObj.VLAN_PppoeIPType == 'PPPoE'){
							if(vm.vlanPppoeMatchAddParams[key]){
								selectObj[key] = selectObj[vm.vlanPppoeMatchAddParams[key]] ? selectObj[vm.vlanPppoeMatchAddParams[key]] :'';
							}
						}else if(selectObj.VLAN_IPv4IPType && (selectObj.VLAN_IPv4IPType == 'DHCP' || selectObj.VLAN_IPv4IPType == 'Static')){
							if(vm.vlanIPv4MatchAddParams[key]){
								selectObj[key] = selectObj[vm.vlanIPv4MatchAddParams[key]] ? selectObj[vm.vlanIPv4MatchAddParams[key]] :'';
							}
						}else if(selectObj.VLAN_IPv6IPType && (selectObj.VLAN_IPv6IPType == 'DHCPv6' || selectObj.VLAN_IPv6IPType == 'Staticv6')){
							if(vm.vlanIPv6MatchAddParams[key]){
								selectObj[key] = selectObj[vm.vlanIPv6MatchAddParams[key]] ? selectObj[vm.vlanIPv6MatchAddParams[key]] :'';
							}
						}
					}else{
						if(selectObj.WAN_PppoeIPType && selectObj.WAN_PppoeIPType == 'PPPoE'){
							if(vm.wanPppoeMatchAddParams[key]){
								selectObj[key] = selectObj[vm.wanPppoeMatchAddParams[key]] ? selectObj[vm.wanPppoeMatchAddParams[key]] :'';
							}
						}else if(selectObj.WANOrLan_IPv4IPType && (selectObj.WANOrLan_IPv4IPType == 'DHCP' || selectObj.WANOrLan_IPv4IPType == 'Static')){
							if(vm.wanIPv4MatchAddParams[key]){
								selectObj[key] = selectObj[vm.wanIPv4MatchAddParams[key]] ? selectObj[vm.wanIPv4MatchAddParams[key]] :'';
							}
						}else if(selectObj.WANOrLan_IPv6IPType && (selectObj.WANOrLan_IPv6IPType == 'DHCPv6' || selectObj.WANOrLan_IPv6IPType == 'Staticv6')){
							if(vm.wanIPv4MatchAddParams[key]){
								selectObj[key] = selectObj[vm.wanIPv6MatchAddParams[key]] ? selectObj[vm.wanIPv6MatchAddParams[key]] :'';
							}
						}
					}
				})
				Object.keys(selectObj).forEach(function(key){
					
					if(key.slice(0,9) == 'WAN_Pppoe' || key.slice(0,10) == 'VLAN_Pppoe' || key.slice(0,12) == 'WANOrLan_IPv' || key.slice(0,8) == 'VLAN_IPv'){
						if(key == 'WAN_Pppoeidx' || key == 'VLAN_Pppoeidx' || key == 'WANOrLan_IPv4idx' || key == 'WANOrLan_IPv6idx' || key == 'VLAN_IPv4idx' || key == 'VLAN_IPv6idx'){

						}else{
							delete selectObj[key]
						}
					}
				})
				Object.assign(vm.addWANDialogForm,selectObj)
				vm.oldWanSelectRow = JSON.parse(JSON.stringify(row));
			}
			vm.addWANDialogShow = true;
		},
		// 新增 WAN提交
		addWANDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'WAN':'WANList',
				},
				idxStr = 'WANAndVlan_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addWANDialogForm).forEach(function(key){

				if(vm.addWANDialogForm.IPType == 'DHCP' || vm.addWANDialogForm.IPType == 'Static'){
					if(vm.addWANDialogForm.VlanID){
						if(vm.vlanIPv4MatchAddParams[key]){
							params[vm.vlanIPv4MatchAddParams[key]] = vm.addWANDialogForm[key];
						}else{
							params[key] = vm.addWANDialogForm[key];
						}
					}else{
						if(vm.wanIPv4MatchAddParams[key]){
							params[vm.wanIPv4MatchAddParams[key]] = vm.addWANDialogForm[key];
						}else{
							params[key] = vm.addWANDialogForm[key];
						}
					}
				}else if(vm.addWANDialogForm.IPType == 'DHCPv6' || vm.addWANDialogForm.IPType == 'Staticv6'){
					if(vm.addWANDialogForm.VlanID){
						if(vm.vlanIPv6MatchAddParams[key]){
							params[vm.vlanIPv6MatchAddParams[key]] = vm.addWANDialogForm[key];
						}else{
							params[key] = vm.addWANDialogForm[key];
						}
					}else{
						if(vm.wanIPv6MatchAddParams[key]){
							params[vm.wanIPv6MatchAddParams[key]] = vm.addWANDialogForm[key];
						}else{
							params[key] = vm.addWANDialogForm[key];
						}
					}
				}else if(vm.addWANDialogForm.IPType == 'PPPoE'){
					if(vm.addWANDialogForm.VlanID){
						if(vm.vlanPppoeMatchAddParams[key]){
							params[vm.vlanPppoeMatchAddParams[key]] = vm.addWANDialogForm[key];
						}else{
							params[key] = vm.addWANDialogForm[key];
						}
					}else{
						if(vm.wanPppoeMatchAddParams[key]){
							params[vm.wanPppoeMatchAddParams[key]] = vm.addWANDialogForm[key];
						}else{
							params[key] = vm.addWANDialogForm[key];
						}
					}
				}
			})
			if(vm.addWANDialogForm.operateType){
				params.operateType = vm.addWANDialogForm.operateType
			}
			vm.$refs.addWANDialogForm.validate(function(valid){
				if(valid){
					var isExist = false;
					if(vm.optType == 'add'){
					    vm.VlanNameList.map((item) =>{
							if(vm.addWANDialogForm.VlanName && item == vm.addWANDialogForm.VlanName){
								isExist = true;
								vm.$message({
									type:'warning',
									message:'Have the same VLAN name conflict!'
								})
								return;
							}
						});
						vm.VlanIdList.map((item) =>{
							if(vm.addWANDialogForm.VlanID && item == vm.addWANDialogForm.VlanID){
								isExist = true;
								vm.$message({
									type:'warning',
									message:'Have the same VLAN ID conflict!'
								})
								return;
							}
						});
						vm.WanIpAddressList.map((item) =>{
							if(item == vm.addWANDialogForm.IP){
								isExist = true;
								vm.$message({
									type:'warning',
									message:'IP of the same network segment already exists for other interfaces,please reconfigure'
								})
								return;
							}
						});
						var vlanDataLength = [];
						vm.ruleForm.WANList.filter((item)=>{
							if(item.VlanID && (!item.operateType || (item.operateType &&item.operateType != 'remove'))){
								vlanDataLength.push(item);
							}
						});
						if(vlanDataLength.length >= 8){
							isExist = true;
							vm.$message({
								type:'warning',
								message:'The maximum number of VLANs is 8'
							})
							return;
						}
						if(isExist)return;
					}else{
						vm.VlanNameList.map((item) =>{
							if(item == vm.addWANDialogForm.VlanName && vm.addWANDialogForm.VlanName != vm.oldWanSelectRow.VlanName){
								isExist = true;
								vm.$message({
									type:'warning',
									message:'Have the same VLAN name conflict!'
								})
								return;
							}
						});
						vm.VlanIdList.map((item) =>{
							if(item == vm.addWANDialogForm.VlanID && vm.addWANDialogForm.VlanID != vm.oldWanSelectRow.VlanID){
								isExist = true;
								vm.$message({
									type:'warning',
									message:'Have the same VLAN ID conflict!'
								})
								return;
							}
						});
						var oldIpAddress = '';
						if(vm.addWANDialogForm.IPType == 'DHCP' || vm.addWANDialogForm.IPType == 'Static'){
							oldIpAddress = vm.addWANDialogForm.VlanID ? vm.oldWanSelectRow.VLAN_IPv4Ip : vm.oldWanSelectRow.WANOrLan_IPv4Ip;
						}else if(vm.addWANDialogForm.IPType == 'DHCPv6' || vm.addWANDialogForm.IPType == 'Staticv6'){
							oldIpAddress = vm.addWANDialogForm.VlanID ? vm.oldWanSelectRow.VLAN_IPv6Ip : vm.oldWanSelectRow.WANOrLan_IPv6Ip;
						}else if(vm.addWANDialogForm.IPType == 'PPPoE'){
							oldIpAddress = vm.addWANDialogForm.VlanID ? vm.oldWanSelectRow.VLAN_PppoeIp : vm.oldWanSelectRow.WAN_PppoeIp;
						}
						vm.WanIpAddressList.map((item) =>{
							if(item == vm.addWANDialogForm.IP && vm.addWANDialogForm.IP != oldIpAddress){
								isExist = true;
								vm.$message({
									type:'warning',
									message:'IP of the same network segment already exists for other interfaces,please reconfigure'
								})
								return;
							}
						});
						if(isExist)return;
					}
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								if(item[idxStr]){
									idList.push(item[idxStr]);
								}
							})
							params[idxStr] = vm.createId(1,idList);
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
							if(params.VlanID && !params[idxStr]){
								var idList=[];
								vm.ruleForm[optTb].map((item)=>{
									if(item[idxStr]){
										idList.push(item[idxStr]);
									}
								})
								params[idxStr] = vm.createId(1,idList); 
							}
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params);
					}
					vm.addWANDialogShow = false;
				}
			})
		},
		// 删除 WAN
		delWANList(row,tbType){
			var vm = this,
				codes = {
					'WAN':'WANList',
				},
				idxStr = 'WANAndVlan_idx',
				optTb = codes[tbType];
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] && item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.ruleForm[optTb].splice(index,1,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return  items[idxStr] != row[idxStr]
					})
				}
			})
		},
		// 新增VLAN 按钮点击
		addVlanClick(){
			var vm = this;
			vm.addWANDialogForm.VlanID = '';
		},
		// 关闭 WAN弹窗
		closeAddWANDialog(){
			var vm = this,
				params = {
					IPType:'DHCP',
					PortType:'NG',
					IP:'',
					SubnetMask:'',
					Gateway:'',
					VlanID:'',
					VlanName:'',
					DiallingMethod:'',
					UserName:'',
					Password:'',
				};
			Object.assign(vm.addWANDialogForm,params);
			vm.$refs.addWANDialogForm.clearValidate();
		},
		// 打开新增 LAN弹窗
		addLANDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addLANDialogForm,row)
			}
			vm.addLANDialogShow = true;
		},
		// 新增 LAN提交
		addLANDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'LAN':'LANList',
				},
				idxStr = 'WANOrLan_IPv4idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addLANDialogForm).forEach(function(key){
				params[key] = vm.addLANDialogForm[key];
			})
			if(vm.addLANDialogForm.operateType){
				params.operateType = vm.addLANDialogForm.operateType
			}
			vm.$refs.addLANDialogForm.validate(function(valid){
				if(valid){
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addLANDialogShow = false;
				}
			})
		},
		// 删除 LAN
		delLANList(row,tbType){
			var vm = this,
				codes = {
					'LAN':'LANList',
				},
				idxStr = 'WANOrLan_IPv4idx',
				optTb = codes[tbType];
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm[optTb],index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return items[idxStr] != row[idxStr]
					})
				}
			})
		},
		// 关闭 LAN弹窗
		closeAddLANDialog(){
			var vm = this,
				params = {
					WANOrLan_IPv4Ip:'',
					WANOrLan_IPv4SubnetMask:''
				};
			Object.assign(vm.addLANDialogForm,params);
			vm.$refs.addLANDialogForm.clearValidate();
		},
		// 添加事件 Default Dns
        addDefaultRouteDns(){
            var vm = this,
                val = vm.defaultRoute.Dns;
            if(val){
                if(vm.isValidIP(val) || vm.isIPv6(val)) {
                    var result = vm.defaultRouteDnsList.some(item=>item == val);
                    if(result){
                        vm.defaultRoute.defaultRouteDnsErrorMessage = '<%=rb.getString("YiCunZai")%>';
                    }else{
                        vm.defaultRouteDnsList.push(val);
                        vm.defaultRoute.Dns = '';
                        vm.defaultRoute.defaultRouteDnsErrorMessage = '';
						vm.ruleForm.defaultRouteDnsStr = vm.defaultRouteDnsList.join(',');
                    }
                }else {
                    vm.defaultRoute.defaultRouteDnsErrorMessage = 'Support configuration of IPV4 or IPV6';
                }
            }
            
        },
        // Default Route Dns 删除事件
        defaultRouteDnsDel(val){
            var vm = this;
            vm.defaultRouteDnsList = vm.defaultRouteDnsList.filter((items)=>{
                return items != val
            })
			vm.ruleForm.defaultRouteDnsStr = vm.defaultRouteDnsList.join(',');
        },
        // DHCP Dns 删除事件
        dhcpDnsDel(val){
            var vm = this;
            vm.dhcpDnsList = vm.dhcpDnsList.filter((items)=>{
                return items != val
            })
			vm.ruleForm.dhcpDnsStr = vm.dhcpDnsList.join(',');
        },
		// 打开新增 IPSecTunnel 弹窗
		addIPSecTunnelDialogOpen(row,optType,tbType){
			var vm = this,
				idxStr = tbType + '_idx';
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.keys(vm.addIPSecDialogForm).map((key)=>{
					var value = row[key],
						endVal = '',
						startVal = '';
					if(key != 'IPSec_KeyLeftType' && key != 'IPSec_IKELeftTimeType' && key != 'IPSec_RekeyMarginType' && key != 'IPSec_DpddelayType'){
						if(key == 'IPSec_KeyLeft'){
							if(value){
								endVal = value.slice(-1);
								startVal = value.slice(0,-1);
								vm.addIPSecDialogForm.IPSec_KeyLeft = startVal;
								vm.addIPSecDialogForm.IPSec_KeyLeftType = endVal;
							}
						}else if(key == 'IPSec_IKELeftTime'){
							if(value){
								endVal = value.slice(-1);
								startVal = value.slice(0,-1);
								vm.addIPSecDialogForm.IPSec_IKELeftTime = startVal;
								vm.addIPSecDialogForm.IPSec_IKELeftTimeType = endVal;
							}
						}else if(key == 'IPSec_RekeyMargin'){
							if(value){
								endVal = value.slice(-1);
								startVal = value.slice(0,-1);
								vm.addIPSecDialogForm.IPSec_RekeyMargin = startVal;
								vm.addIPSecDialogForm.IPSec_RekeyMarginType = endVal;
							}
						}else if(key == 'IPSec_Dpddelay'){
							if(value){
								endVal = value.slice(-1);
								startVal = value.slice(0,-1);
								vm.addIPSecDialogForm.IPSec_Dpddelay = startVal;
								vm.addIPSecDialogForm.IPSec_DpddelayType = endVal;
							}
						}else{
							vm.addIPSecDialogForm[key] = row[key] ? row[key] : '';
						}
					}
				})
				vm.addIPSecDialogForm[idxStr] = row[idxStr];
			}
			vm.addIPSecTunnelDialogShow = true;
		},
		// 新增 IPSecTunnel 提交
		addIPSecTunnelDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'IPSec':'IPSecTunnelList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addIPSecDialogForm).forEach(function(key){
				if(key != 'IPSec_KeyLeftType' && key != 'IPSec_IKELeftTimeType' && key != 'IPSec_RekeyMarginType' && key != 'IPSec_DpddelayType'){
					if(key == 'IPSec_KeyLeft'){
						var value = vm.addIPSecDialogForm[key];
						if(value){
							params[key] = vm.addIPSecDialogForm.IPSec_KeyLeft + '' + vm.addIPSecDialogForm.IPSec_KeyLeftType
						}
					}else if(key == 'IPSec_IKELeftTime'){
						var value = vm.addIPSecDialogForm[key];
						if(value){
							params[key] = vm.addIPSecDialogForm.IPSec_IKELeftTime + '' + vm.addIPSecDialogForm.IPSec_IKELeftTimeType
						}
					}else if(key == 'IPSec_RekeyMargin'){
						var value = vm.addIPSecDialogForm[key];
						if(value){
							params[key] = vm.addIPSecDialogForm.IPSec_RekeyMargin + '' + vm.addIPSecDialogForm.IPSec_RekeyMarginType
						}
					}else if(key == 'IPSec_Dpddelay'){
						var value = vm.addIPSecDialogForm[key];
						if(value){
							params[key] = vm.addIPSecDialogForm.IPSec_Dpddelay + '' + vm.addIPSecDialogForm.IPSec_DpddelayType
						}
					}else{
						params[key] = vm.addIPSecDialogForm[key];
					}
				}
			})
			if(vm.addIPSecDialogForm.operateType){
				params.operateType = vm.addIPSecDialogForm.operateType
			}
			vm.$refs.addIPSecDialogForm.validate(function(valid){
				if(valid){
					var isExist = false;
					
					if(vm.optType == 'add'){
						isExist =  vm.ruleForm[optTb].some(item =>item.IPSec_Gateway == vm.addIPSecDialogForm.IPSec_Gateway);
					}else{
						vm.ruleForm[optTb].map((item)=>{
							if(item.IPSec_Gateway == vm.addIPSecDialogForm.IPSec_Gateway){
								if(item[idxStr] != vm.addIPSecDialogForm[idxStr]){
									isExist = true;
								}
							}
						})
					}
					if(isExist){
						vm.$message.warning('Gateway already exists');
						return;
					}
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addIPSecTunnelDialogShow = false;
				}
			})
		},
		// 删除 IPSec Tunnel
		delIPSecTunnelList(row,tbType){
			var vm = this,
				codes = {
					'IPSec':'IPSecTunnelList',
				},
				idxStr = tbType + '_idx',
				optTb = codes[tbType];
			
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm[optTb],index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return items[idxStr] != row[idxStr]
					})
				}
				vm.$nextTick(function(){
					['NGAP_InterfaceBinding','NGU_InterfaceBinding'].map((key)=>{
						if(vm.ruleForm[key] == row.IPSec_TunnelName){
							if(vm.tunnelNameList.length > 0){
								vm.ruleForm[key] = vm.tunnelNameList[0]
							}else{
								vm.ruleForm[key] = ''
							}
						}
					})
				})
			})
		},
		// 关闭 IPSecTunnel弹窗
		closeAddIPSecTunnelDialog(){
			var vm = this,
				params = {
					IPSec_Switch:'0',
					IPSec_TunnelName:'',
					IPSec_LeftAuth:'psk',
					IPSec_RightAuth:'psk',
					IPSec_Gateway:'10.10.10.10',
					IPSec_RightSubnet:'0.0.0.0/0',
					IPSec_RightID:'C=CH,O=strongSwan,CN=server',
					IPSec_SecretKey:'clientKey.der',
					IPSec_LeftID:'C=CH,O=strongSwan,CN=server',
					IPSec_LeftCert:'',
					IPSec_LeftSourceIp:'%config',
					IPSec_LeftSubnet:'',
					IPSec_Fragmentation:'yes',
					IPSec_IKEEncryption:'aes128',
					IPSec_IKEDHGroup:'modp1024',
					IPSec_IKEAuthentication:'sha256',
					IPSec_ESPEncryption:'aes128',
					IPSec_ESPDHGroup:'modp1024',
					IPSec_ESPAuthentication:'sha256',
					IPSec_KeyLeft:'360',
					IPSec_KeyLeftType:'d',
					IPSec_IKELeftTime:'360',
					IPSec_IKELeftTimeType:'d',
					IPSec_RekeyMargin:'5',
					IPSec_RekeyMarginType:'m',
					IPSec_Dpdaction:'restart',
					IPSec_Dpddelay:'30',
					IPSec_DpddelayType:'s',
					IPSec_LeftInterface:'None',
				};
			Object.assign(vm.addIPSecDialogForm,params);
			vm.$refs.addIPSecDialogForm.clearValidate();
		},
		// 打开新增 DSCP 弹窗
		addDSCPDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addDSCPDialogForm,row)
			}
			vm.addDSCPDialogShow = true;
		},
		// 新增 DSCP 提交
		addDSCPDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'DSCP':'DSCPList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addDSCPDialogForm).map(function(key){
				params[key] = vm.addDSCPDialogForm[key]
			})
			if(vm.addDSCPDialogForm.operateType){
				params.operateType = vm.addDSCPDialogForm.operateType
			}
			vm.$refs.addDSCPDialogForm.validate(function(valid){
				if(valid){
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addDSCPDialogShow = false;
				}
			})
		},
		// 删除 DSCP
		delDSCPList(row,tbType){
			var vm = this,
				codes = {
					'DSCP':'DSCPList',
				},
				idxStr = tbType + '_idx',
				optTb = codes[tbType];
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm[optTb],index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return items[idxStr] != row[idxStr]
					})
				}
			})
		},
		// 关闭 DSCP 弹窗
		closeAddDSCPDialog(){
			var vm = this,
				params = {
					DSCP_DSCPVal:'',
					DSCP_VLANPriority:'',
				};
			Object.assign(vm.addDSCPDialogForm,params);
			vm.$refs.addDSCPDialogForm.clearValidate();
		},
		// 打开新增 Static Routing 弹窗
		addStaticDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addStaticDialogForm,row)
			}
			vm.addStaticDialogShow = true;
		},
		// 新增 Static Routing 提交
		addStaticDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'Static':'StaticList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addStaticDialogForm).forEach(function(key){
				params[key] = vm.addStaticDialogForm[key]
			})
			if(vm.addStaticDialogForm.operateType){
				params.operateType = vm.addStaticDialogForm.operateType
			}
			vm.$refs.addStaticDialogForm.validate(function(valid){
				if(valid){
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addStaticDialogShow = false;
				}
			})
		},
		// 删除 Static Routing
		delStaticList(row,tbType){
			var vm = this,
				codes = {
					'Static':'StaticList',
				},
				idxStr = tbType + '_idx',
				optTb = codes[tbType];
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm[optTb],index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return items[idxStr] != row[idxStr]
					})
				}
			})
		},
		// 关闭 Static Routing 弹窗
		closeAddStaticDialog(){
			var vm = this,
				params = {
					Static_IPVersion:'1',
					Static_InterfaceName:'opt',
					Static_DestinationNetwork:'',
					Static_NetmaskPrefixLength:'',
					Static_Gateway:'',
				};
			Object.assign(vm.addStaticDialogForm,params);
			vm.$refs.addStaticDialogForm.clearValidate();
		},
		closeAddPLMNDialog(){
			var vm = this,
				params = {
					NRCellIdentity:'',
					NRCellTAC:'',
					NRCellRanac:'',
				};
			Object.assign(vm.addPLMNDialogForm,params);
			vm.$refs.addPLMNDialogForm.clearValidate();
		},
		settingsSubmit(){
			var vm = this;
			var params = {},
				isChanged = isFormChanged(vm.$refs.ruleForm),
				isSync = false;

			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}
			vm.$refs.ruleForm.fields.map(function(field){
				var key = vm.getNameByProp(field.prop);

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						var editList=[],subList=[];
						vList.map((items)=>{
							if(items.operateType){
								if(field.prop == 'WANList'){
									var wanSubObj={};
									
									Object.keys(items).map((keyVal)=>{
										wanSubObj[keyVal] = items[keyVal]
									});
									wanSubObj.interfaceIndex = vm.wanInterfaceIndex;
									if(items.operateType !== 'add' && items.VlanID){
										wanSubObj.vlanInterfaceIndex = items.VlanInterfaceIndex
									}
									if(items.operateType == 'add' && items.VlanID && !items.VlanName){
										vm.ruleForm.WANList.map((wanItems)=>{
											if(items.VlanID == wanItems.VlanID){
												wanSubObj.VlanName = wanItems.VlanName;
												wanSubObj.vlanInterfaceIndex = wanItems.VlanInterfaceIndex ? wanItems.VlanInterfaceIndex : wanSubObj.vlanInterfaceIndex
											}
										})
									}
									if(!wanSubObj.VlanID){
										delete wanSubObj.VlanID
										delete wanSubObj.VlanName
									}
									if((wanSubObj.WAN_PppoeIPType && wanSubObj.WAN_PppoeIPType == 'PPPoE') || (wanSubObj.VLAN_PppoeIPType && wanSubObj.VLAN_PppoeIPType == 'PPPoE')){
										delete wanSubObj.PortType
										delete wanSubObj.Gateway
									}else{
										delete wanSubObj.DiallingMethod
										delete wanSubObj.UserName
										delete wanSubObj.Password
									}
									delete wanSubObj.InterfaceType
									delete wanSubObj.InterfaceIndex
									if(wanSubObj.VlanID){
										Object.keys(wanSubObj).forEach(function(key){
											if(key.slice(0,9) == 'WAN_Pppoe' || key.slice(0,12) == 'WANOrLan_IPv'){
												delete wanSubObj[key]
											}
										})

									}
                                    Object.keys(wanSubObj).forEach(function(key){
                                        if(key.slice(-6) == 'IPType' && (wanSubObj[key] == 'DHCP' || wanSubObj[key] == 'DHCPv6' )){
                                            delete wanSubObj['WANOrLan_IPv4Ip']
                                            delete wanSubObj['WANOrLan_IPv4SubnetMask']
                                            delete wanSubObj['WANOrLan_IPv4Gateway']

                                            delete wanSubObj['VLAN_IPv4Ip']
                                            delete wanSubObj['VLAN_IPv4SubnetMask']
                                            delete wanSubObj['VLAN_IPv4Gateway']

                                            delete wanSubObj['WANOrLan_IPv6Ip']
                                            delete wanSubObj['WANOrLan_IPv6SubnetMask']
                                            delete wanSubObj['WANOrLan_IPv6Gateway']

                                            delete wanSubObj['VLAN_IPv6Ip']
                                            delete wanSubObj['VLAN_IPv6SubnetMask']
                                            delete wanSubObj['VLAN_IPv6Gateway']
                                        }
                                    })
									editList.push(wanSubObj)
								}else if(field.prop == 'LANList'){
									var lanSubObj={};
									['WANOrLan_IPv4idx','WANOrLan_IPv4Ip','WANOrLan_IPv4SubnetMask','operateType'].map((keyVal)=>{
										lanSubObj[keyVal] = items[keyVal]
									})
									lanSubObj.interfaceIndex = vm.lanInterfaceIndex;
									
									editList.push(lanSubObj)
								}else{
									editList.push(items)
								}
								
							}
						})
						editList.map((items)=>{
							if(items.operateType == 'add'){
								Object.keys(items).map((key)=>{
									if(key.slice(-3) == 'idx'){
										delete items[key]
									}
									if(key == 'WANOrLan_IPv6IPType' && items[key] == 'Staticv6'){
										items[key] = 'Static';
									}
								})
							}
						})
						if(field.prop == 'StaticList'){
							editList.map((items)=>{
								if(items.Static_IPVersion == '1'){
									items.Static_NetmaskPrefixLength = items.Static_NetmaskPrefixLength ? vm.maskConvertNumber(items.Static_NetmaskPrefixLength) : '';
								}
							})
						}
						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								objs[listKey] = items[listVal]
							}
							objs.cellIndex = '1';
							subList.push(objs)
						})
						
						if(field.prop == 'WANList' || field.prop == 'LANList'){
							if(field.prop == 'WANList'){
								var wanIpv4List = [],
									wanIpv6List = [],
									wanPppoeList = [],
									vlanList = [];
								subList.map((items)=>{
									if(!items['831396135055560A860153BEAE8BC7D5']){
										if(items['290803D1A4DCCCE6E2ABEE6A148149AA'] && items['290803D1A4DCCCE6E2ABEE6A148149AA'] == 'PPPoE'){
											wanPppoeList.push(items);
										}else{
											if(items['F420F84CED364E05A86BE443A8418251'] && (items['F420F84CED364E05A86BE443A8418251'] == 'DHCP' || items['F420F84CED364E05A86BE443A8418251'] == 'Static')){
												wanIpv4List.push(items);
											}else{
												wanIpv6List.push(items);
											}
										}
									}else{
										vlanList.push(items)
									}
								})
								if(wanIpv4List.length > 0) {
									params['1F2A1879F55145E9CB3AEF66D678336E'] = wanIpv4List;
								}
								if(wanIpv6List.length > 0) {
									params['3D32B8E2FA9A0ED0178F573527C0553F'] = wanIpv6List;
								}
								if(wanPppoeList.length > 0) {
									params['5F855B80009EBFBE022194FEB23A3708'] = wanPppoeList;
								}
								if(vlanList.length > 0) {
									params['B85BEE2D9ACE849D267E1D9E71111862'] = vlanList;
								}
							}else{
								if(subList.length > 0) {
									params['1F2A1879F55145E9CB3AEF66D678336E'] = subList;
								}
							}
						}else{
							if(subList.length > 0) {
								params[key] = subList;
							}
						}
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						var editData={
								cellIndex:'1',
								value:field.fieldValue
							}
						if(field.prop == 'Soft_Key' || field.prop == 'Soft_Opc'){
							var valStr =  field.fieldValue;
							if(valStr){
								editData.value = valStr.replaceAll(' ','').split('').reduce(function(n,m){
									if(n.length%3 == 2)n = n + ' ';
									return n + m
								})
							}else{
								editData.value = ''
							}
						}
                        if(field.prop == 'Pon_StartIp' || field.prop == 'Pon_EndIp'){
                            var valStr =  field.fieldValue;
                            if(valStr){
                                editData.value = '192.168.150.' + valStr;
                            }else{
                                editData.value = ''
                            }
                        }
                        if(field.prop == 'Pon_StartIp' || field.prop == 'Pon_EndIp' || field.prop == 'Pon_SubnetMask'){
                            if(vm.ruleForm.Pon_Switch == '1'){
                                params[key] = editData;
                            }
                        }else{
                            params[key] = editData;
                        }
					};
				}
			});
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
                    if(vm.ponStartIpNoLessEndIpErrShow)return;
					var rowCode = vm.smallCellCode,
						url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
					$('#gnbSetting_main').addClass('loading');
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							eventBus.$emit('close-gnb-settingPage');
						}else{
							vm.$message.error(data["message"])
						}
						$('#gnbSetting_main').removeClass('loading');
					})
				}
			});
		},
		getNameByProp(prop) {
			var vm = this,
				reg = /^\w*\.\d*\.\w*$/
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
		createId(idVal,list){
			var vm = this,
				val = idVal + '';
			if(list.includes(val) == true){
				idVal += 1 ;
				return vm.createId(idVal,list);
			}else{
				return  idVal + '';
			}
		},
		closeSettings(){
			eventBus.$emit('close-gnb-settingPage');
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
        //  问题单#103625 添加静态路由 掩码支持4个255
        isValidSubnetMask(mask) {
            // 匹配常见的合法子网掩码
            const regex = /^(?:255|254|252|248|240|224|192|128|0)\.(?:255|254|252|248|240|224|192|128|0)\.(?:255|254|252|248|240|224|192|128|0)\.(?:255|254|252|248|240|224|192|128|0)$/;
            if (!regex.test(mask)) return false;
            // 进一步校验是否为连续的1
            const parts = mask.split('.').map(Number);
            let bin = parts.map(n => n.toString(2).padStart(8, '0')).join('');
            return /^1*0*$/.test(bin);
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
		syncSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				params = {
					smallCellCode:vm.smallCellCode
				},
				str = Math.random().toString();
				
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					gnbTabSettingVue.changeMain('network');
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		// 将整数(string)转换为子网掩码
		numberConvertMask(num){
			var str = '';
			for(var i=0;i<num;i++){
				str += '1';
			}
			for(var i=0;i<32-num;i++){
				str += '0';
			}
			var arr = [];
			for(var i=0;i<str.length;i+=8){
				arr.push(parseInt(str.substr(i,8),2));
			}
			return arr.join('.');
		},
		// 将子网掩码转换为整数(string)
		maskConvertNumber(mask){
			var arr = mask.split('.');
			var str = '';
			for(var i=0;i<arr.length;i++){
				str += parseInt(arr[i]).toString(2);
			}
			return str.split('0').join('').length+'';
		},
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data",this.init)
	}
});

</script>
