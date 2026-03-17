<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gnbQuickSettingPage{
	height: 100%;
	width: 100%;
}
#gnbQuickSettingPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gnbQuickSettingPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #DFE2EE;
}
#gnbQuickSettingPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gnbQuickSettingPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #DFE2EE;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gnbQuickSettingPage .rightContentCls .contentTableTitle{
	display: flex;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gnbQuickSettingPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gnbQuickSettingPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gnbQuickSettingPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gnbQuickSettingPage .itemListBoxCls{
	padding-top: 5px;
}
#gnbQuickSettingPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gnbQuickSettingPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gnbQuickSettingPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gnbQuickSettingPage .el-form-item{
	margin-bottom: 20px;
}
#gnbQuickSettingPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gnbQuickSettingPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gnbQuickSettingPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
</style>

<div id="gnbQuickSettingPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			<%=rb.getString("KuaiSuSheZhi")%>
			<!-- 按钮  同步 -->
			<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSettingsClick" tip="<%=rb.getString("TongBu")%>">
				<span class="el-icon el-icon-circle-refresh"></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
					<el-collapse-item name="AMF">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">AMF</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--AMF-->
							<div> 
								<div class="contentTableTitle">
									<div>AMF List</div>
									<div><span class="el-icon el-icon-circle-add" @click="addAMFDialogOpen('','add','AMF')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="amfTable" 
										:row-class-name="tableRowClassName"
										:rownumber="true" 
										id="amfTable" 
										:data="ruleForm.AMFList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
											<template slot-scope="scope">
												<span class="el-icon el-icon-operation-delete" @click="delAMFList(scope.row,event)" ></span>
											</template>
										</el-table-column>
										<el-table-column label='ID' min-width="40" prop="AMF_idx" show-overflow-tooltip></el-table-column>
										<el-table-column label='AMF IP' min-width="120" prop="AMF_IP" show-overflow-tooltip></el-table-column>
										<el-table-column label='PLMN ID' min-width="120" prop="AMF_PLMNID" show-overflow-tooltip></el-table-column>
										<el-table-column label='Default' min-width="120" prop="AMF_Default" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='AMFList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.AMFList'></el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="CU">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">CU</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='CU_F1_C_Local_IP' style="width:40%;min-width:400px;" label="F1-C Local IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										F1-C Local IP
									</span>
									<el-input v-model.trim='ruleForm.CU_F1_C_Local_IP' style="width:150px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='CU_F1_U_Local_IP' style="width:40%;min-width:400px;" label="F1-U Local IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										F1-U Local IP
									</span>
									<el-input v-model.trim='ruleForm.CU_F1_U_Local_IP' style="width:150px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
								<div class="moreIpItemBoxCls">
									<div class="leftAndRightItemCls">
										<el-form-item label="NG-C Local IP"  label-width="160px" style="margin-bottom:0px;">
											<el-input style='width:200px;' v-model="CU_NG_C_Local_IPStr">
												<i slot="suffix" class="el-icon el-icon-plus" @click="addNGCLocalIP"></i>
											</el-input>
											<p class="errorBoxCls">{{NGC_LocalIPErrorMessage}}</p>
										</el-form-item>
										<div class="itemListBoxCls">
											<div v-for="item in NGC_LocalIPList" class="itemCls">
												<span>{{item}}</span>
												<span class="el-icon el-icon-close" style="margin-left:5px;" @click="NGCLocalIPListDel(item)"></span>
											</div>
										</div>
										<el-form-item prop="CU_NG_C_Local_IP" style="display:none;" label-width="160px">
											<el-input style='width:200px;' v-model="ruleForm.CU_NG_C_Local_IP" ></el-input>
										</el-form-item>
									</div>
									<div class="leftAndRightItemCls">
										<el-form-item label="NG-U Local IP"  label-width="160px" style="margin-bottom:0px;">
											<el-input style='width:200px;' v-model="CU_NG_U_Local_IPStr" >
												<i slot="suffix" class="el-icon el-icon-plus" @click="addNGULocalIP"></i>
											</el-input>
											<p class="errorBoxCls">{{NGU_LocalIPErrorMessage}}</p>
										</el-form-item>
										<div class="itemListBoxCls">
											<div v-for="item in NGU_LocalIPList" class="itemCls">
												<span>{{item}}</span>
												<span class="el-icon el-icon-close" style="margin-left:5px;" @click="NGULocalIPListDel(item)"></span>
											</div>
										</div>
										<el-form-item prop="CU_NG_U_Local_IP" style="display:none;" label-width="160px">
											<el-input style='width:200px;' v-model="ruleForm.CU_NG_U_Local_IP" ></el-input>
										</el-form-item>
									</div>
								</div>
								<el-form-item prop='CU_Xn_Local_IP' style="width:40%;min-width:400px;" label="Xn Local IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										Xn Local IP
									</span>
									<el-input v-model.trim='ruleForm.CU_Xn_Local_IP' style="width:150px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="DU">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">DU</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='DU_F1_C_Local_IP' style="width:40%;min-width:400px;" label="F1-C Local IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										F1-C Local IP
									</span>
									<el-input v-model.trim='ruleForm.DU_F1_C_Local_IP' style="width:150px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_F1_U_Local_IP' style="width:40%;min-width:400px;" label="F1-U Local IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										F1-U Local IP
									</span>
									<el-input v-model.trim='ruleForm.DU_F1_U_Local_IP' style="width:150px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_F1_C_Remote_IP' style="width:40%;min-width:400px;" label="F1-C Remote IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										F1-C Remote IP
									</span>
									<el-input v-model.trim='ruleForm.DU_F1_C_Remote_IP' style="width:150px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_PAT1DLULTransPeriodicity' style="width:40%;min-width:400px;" label="PAT1 DL UL Trans Periodicity" label-width="160px" class='validate-item'>
									<el-select v-model='ruleForm.DU_PAT1DLULTransPeriodicity'>
										<el-option label='ms0p5' value='0'></el-option>
										<el-option label='ms0p625' value='1'></el-option>
										<el-option label='ms1' value='2'></el-option>
										<el-option label='ms1p25' value='3'></el-option>
										<el-option label='ms2' value='4'></el-option>
										<el-option label='ms2p5' value='5'></el-option>
										<el-option label='ms3' value='6'></el-option>
										<el-option label='ms4' value='7'></el-option>
										<el-option label='ms5' value='8'></el-option>
										<el-option label='ms10' value='9'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='DU_PAT1ofDownlinkSlots' style="width:40%;min-width:400px;" label="PAT1 of Downlink Slots" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT1ofDownlinkSlots' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~320,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_PAT1ofDownlinkSymbols' style="width:40%;min-width:400px;" label="PAT1 of Downlink Symbols" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT1ofDownlinkSymbols' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~13,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_PAT1ofUplinkSlots' style="width:40%;min-width:400px;" label="PAT1 of Uplink Slots" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT1ofUplinkSlots' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~320,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_PAT1ofUplinkSymbols' style="width:40%;min-width:400px;" label="PAT1 of Uplink Symbols" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT1ofUplinkSymbols' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~13,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_PAT2DLULTransPeriodicity' style="width:40%;min-width:400px;" label="PAT2 DL UL Trans Periodicity" label-width="160px" class='validate-item'>
									<el-select v-model='ruleForm.DU_PAT2DLULTransPeriodicity'>
										<el-option label='OFF' value='0x00000FFF'></el-option>
										<el-option label='ms0p5' value='0'></el-option>
										<el-option label='ms0p625' value='1'></el-option>
										<el-option label='ms1' value='2'></el-option>
										<el-option label='ms1p25' value='3'></el-option>
										<el-option label='ms2' value='4'></el-option>
										<el-option label='ms2p5' value='5'></el-option>
										<el-option label='ms3' value='6'></el-option>
										<el-option label='ms4' value='7'></el-option>
										<el-option label='ms5' value='8'></el-option>
										<el-option label='ms10' value='9'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item v-if="ruleForm.DU_PAT2DLULTransPeriodicity != '0x00000FFF'" prop='DU_PAT2ofDownlinkSlots' style="width:40%;min-width:400px;" label="PAT2 of Downlink Slots" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT2ofDownlinkSlots' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~320,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item v-if="ruleForm.DU_PAT2DLULTransPeriodicity != '0x00000FFF'" prop='DU_PAT2ofDownlinkSymbols' style="width:40%;min-width:400px;" label="PAT2 of Downlink Symbols" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT2ofDownlinkSymbols' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~13,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item v-if="ruleForm.DU_PAT2DLULTransPeriodicity != '0x00000FFF'" prop='DU_PAT2ofUplinkSlots' style="width:40%;min-width:400px;" label="PAT2 of Uplink Slots" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT2ofUplinkSlots' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~320,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item v-if="ruleForm.DU_PAT2DLULTransPeriodicity != '0x00000FFF'" prop='DU_PAT2ofUplinkSymbols' style="width:40%;min-width:400px;" label="PAT2 of Uplink Symbols" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT2ofUplinkSymbols' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~13,Integer</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="GNB">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">gNB</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='gnbLength' style="width:40%;min-width:400px;" label="gNB Length" label-width="140px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										gNB Length
									</span>
									<el-input v-model.trim='ruleForm.gnbLength' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：22~32,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='gnbName' style="width:40%;min-width:400px;" label="gNB Name" label-width="140px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										gNB Name
									</span>
									<el-input v-model.trim='ruleForm.gnbName' maxlength="150" style="width:150px;">
										<template slot="append">Length：0~150 Digit</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='gnbId' style="width:40%;min-width:400px;" label="gNB ID" label-width="140px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										gNB ID
									</span>
									<el-input v-model.trim='ruleForm.gnbId' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='adminState' style="width:40%;min-width:400px;" label="AdminState" label-width="140px">
									<span slot="label" class="labelIconCls">
										AdminState
									</span>
									<el-select v-model='ruleForm.adminState' style="width:100px;">
										<el-option label='Locked' value='1'></el-option>
										<el-option label='Unlocked' value='2'></el-option>
										<el-option label='ShuttingDown' value='3'></el-option>
									</el-select>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="PLMN">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">PLMN</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--PLMN-->
							<div class="multiPlmnEnableBoxCls">
								<el-form-item prop='multiPlmnEnable' style="width:100%;min-width:400px;display:flex;" label="Multi Plmn Enable" label-width="180px">
									<el-switch v-model="ruleForm.multiPlmnEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
								</el-form-item>
							</div>
							<div> 
								<div class="contentTableTitle">
									<div>NR Cell <span style="font-size:12px;color:#999999;margin-left:10px;">(No more than 6)</div>
									<div v-if="NRCellAddShow"><span class="el-icon el-icon-circle-add" @click="addPLMNDialogOpen('','add','NRCell')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable 
										ref="NRCellTable" 
										:rownumber="true" 
										:row-class-name="tableRowClassName"
										id="NRCellTable" 
										:data="ruleForm.NRCellList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
										>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
											<template slot-scope="scope">
												<span class="el-icon el-icon-operation-edit" @click="addPLMNDialogOpen(scope.row,'edit','NRCell')" style="margin-right:15px;"></span>
												<span class="el-icon el-icon-operation-delete" @click="delPLMNList(scope.row,event)" ></span>
											</template>
										</el-table-column>
										<el-table-column label='ID' min-width="120" prop="NRCell_idx" show-overflow-tooltip></el-table-column>
										<el-table-column label='NR Cell Identity' min-width="120" prop="NRCellIdentity" show-overflow-tooltip></el-table-column>
										<el-table-column label='TAC' min-width="120" prop="NRCellTAC" show-overflow-tooltip></el-table-column>
										<el-table-column label='Ranac' min-width="120" prop="NRCellRanac" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='NRCellList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.NRCellList'></el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="CELL">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">CELL</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='band' style="width:40%;min-width:400px;" label="Band" label-width="160px" class='validate-item'>
									<el-select v-if="isSharingBaseStation" v-model='ruleForm.band' @change="bandSelectChange" :disabled="sasEnableStatus">
										<el-option v-for="item in bandAndPowerMaxList" :label='item.label' :value='item.label'></el-option>
									</el-select>
									<el-input v-if="!isSharingBaseStation" v-model.trim='ruleForm.band' :disabled="sasEnableStatus">
										<template slot="append"><%=rb.getString("FanWei")%>：1-1000</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='Pci' style="width:40%;min-width:400px;" label="PCI" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.Pci' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~1007,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='NRARFCNDL' style="width:40%;min-width:400px;" label="NRARFCNDL" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.NRARFCNDL' style="width:150px;" :disabled="sasEnableStatus">
										<template slot="append"><%=rb.getString("FanWei")%>：Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='NRARFCNUL' style="width:40%;min-width:400px;" label="NRARFCNUL" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.NRARFCNUL' style="width:150px;" :disabled="sasEnableStatus">
										<template slot="append"><%=rb.getString("FanWei")%>：Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='NumOfTxAtenna' style="width:40%;min-width:400px;" label="NumOfTxAtenna" label-width="160px">
									<el-select v-model='ruleForm.NumOfTxAtenna' style="width:70px;">
										<el-option label='1' value='1'></el-option>
										<el-option label='2' value='2'></el-option>
										<el-option label="4" value="4"></el-option>
										<el-option label="8" value="8"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='NumOfRxAtenna' style="width:40%;min-width:400px;" label="NumOfRxAtenna" label-width="160px">
									<el-select v-model='ruleForm.NumOfRxAtenna' style="width:70px;">
										<el-option label='1' value='1'></el-option>
										<el-option label='2' value='2'></el-option>
										<el-option label="4" value="4"></el-option>
										<el-option label="8" value="8"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='DL_SubCarrierSpacing' style="width:40%;min-width:400px;" label="DL SubCarrierSpacing" label-width="160px">
									<el-select v-model='ruleForm.DL_SubCarrierSpacing' @change="DL_SCSChange" style="width:100px;">
										<el-option label='0(15kHz)' value='0'></el-option>
										<el-option label='1(30kHz)' value='1'></el-option>
										<el-option label="2(60kHz)" value="2"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='DL_CarrierBandWidth' style="width:40%;min-width:400px;" label="DL CarrierBandWidth" label-width="160px">
									<el-select v-model='ruleForm.DL_CarrierBandWidth' style="width:70px;" :disabled="sasEnableStatus">
										<el-option v-for="item in DL_CarrierBandWidthData" :label='item.label' :value='item.value'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='UL_SubCarrierSpacing' style="width:40%;min-width:400px;" label="UL SubCarrierSpacing" label-width="160px">
									<el-select v-model='ruleForm.UL_SubCarrierSpacing' @change="UL_SCSChange" style="width:100px;">
										<el-option label='0(15kHz)' value='0'></el-option>
										<el-option label='1(30kHz)' value='1'></el-option>
										<el-option label="2(60kHz)" value="2"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='UL_CarrierBandWidth' style="width:40%;min-width:400px;" label="UL CarrierBandWidth" label-width="160px">
									<el-select v-model='ruleForm.UL_CarrierBandWidth' style="width:70px;" :disabled="sasEnableStatus">
										<el-option v-for="item in UL_CarrierBandWidthData" :label='item.label' :value='item.value'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='PowerModify' style="width:40%;min-width:400px;" label="Power Modify(dBm)" label-width="160px" class='validate-item' >
									<el-input v-model.trim='ruleForm.PowerModify' style="width:110px;" :disabled="sasEnableStatus">
										<template slot="append"><%=rb.getString("FanWei")%>：{{PowerMinLimitValue}}~{{PowerMaxLimitValue}}</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='rftxEnable' style="width:40%;min-width:400px;" label="RF Enable" label-width="160px">
									<el-select v-model='ruleForm.rftxEnable' style="width:70px;" :disabled="sasEnableStatus">
										<el-option label='ON' value='1'></el-option>
										<el-option label='OFF' value='0'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='offsetToPointA' style="width:40%;min-width:400px;" label="Offset To Point A" label-width="160px" class='validate-item' >
									<el-input v-model.trim='ruleForm.offsetToPointA' style="width:110px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~2199,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='ssbSubCarrierOffset' style="width:40%;min-width:400px;" label="SSB Sub Carrier Offset" label-width="160px" class='validate-item' >
									<el-input v-model.trim='ruleForm.ssbSubCarrierOffset' style="width:110px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
									</el-input>
								</el-form-item>
								<!--<el-form-item prop='ssbGSCN' style="width:40%;min-width:400px;" label="SSB GSCN" label-width="160px" class='validate-item' >
									<el-input v-model.trim='ruleForm.ssbGSCN' style="width:110px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
									</el-input>
								</el-form-item>-->
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
	<!-- AMF新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addAMFDialogShow" @close="closeAddAMFDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addAMFDialogForm" :model='addAMFDialogForm' :rules='addAMFDialogRules' label-position="top">     		     			            
			<el-form-item prop='AMF_IP' style="min-width:400px;" label="AMF IP" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addAMFDialogForm.AMF_IP' style="width:150px;">
					<template slot="append">Example：1.1.1.1</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='AMF_PLMNID' style="min-width:400px;" label="PLMN ID" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addAMFDialogForm.AMF_PLMNID' style="width:150px;">
					<template slot="append">Length：5~6 Digit,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='AMF_Default' style="min-width:400px;" label="Default" label-width="160px">
				<el-select v-model='addAMFDialogForm.AMF_Default' style="width:60px;">
					<el-option label='0' value='0'></el-option>
					<el-option label='1' value='1'></el-option>
				</el-select>
			</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addAMFDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addAMFDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- PLMN新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addPLMNDialogShow" @close="closeAddPLMNDialog" :close-on-click-modal="false" append-to-body>		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addPLMNDialogForm" :model='addPLMNDialogForm' :rules='addPLMNDialogRules' label-position="top">     		     			            
				<el-collapse v-model="nrCellCollapse">
					<el-collapse-item name="Cell">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span class="title-icon" style="vertical-align:sub"></span>
								<span style="font-size:14px;font-weight:bold">NR Cell Setting</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;flex-wrap: wrap">
								<el-form-item prop='NRCellIdentity' style="width:40%;min-width:400px;" label="NR Cell Identity" label-width="140px" class='validate-item'>
									<el-input v-model.trim='addPLMNDialogForm.NRCellIdentity'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~68719476735,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='NRCellTAC' style="width:40%;min-width:400px;" label="TAC" label-width="140px" class='validate-item'>
									<el-input v-model.trim='addPLMNDialogForm.NRCellTAC'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~16777215,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='NRCellRanac' style="width:40%;min-width:400px;" label="Ranac" label-width="140px" class='validate-item'>
									<el-input v-model.trim='addPLMNDialogForm.NRCellRanac'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~255,Integer</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="Plmn" v-show="plmnSettingShow">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span class="title-icon" style="vertical-align:sub"></span>
								<span style="font-size:14px;font-weight:bold">PLMN Setting</span>
								<span style="font-size:12px;color:#999999;margin-right:10px;">（No more than 6）</span>
								<span class="el-icon el-icon-circle-add" @click="addNrPlmnAddClick" style="position:relative;top:2px;"></span>
							</p>
						</template>
						<div style="padding:20px;">
							<div v-for="(item,index) in addPLMNDialogForm.PlmnList" class="PlmnListItemCls" v-show="!item.operateType || item.operateType && item.operateType != 'remove'">
								<div style="display:flex;flex-wrap: wrap;font-size:12px">
									<el-form-item :prop="'PlmnList['+index+'].PlmnId'" :rules="addPLMNDialogRules.PlmnId" label="PLMN ID"  label-width="160px" style="width:40%;min-width:400px;" class='validate-item'> 
										<el-input v-model.trim='item.PlmnId'>
											<template slot="append"><%=rb.getString("FanWei")%>：5~6 Digit,Integer</template>
										</el-input>
									</el-form-item>
									<el-form-item :prop="'PlmnList['+index+'].Primary'" label="Primary"  label-width="160px" style="width:40%;min-width:300px;">
										<el-select v-model='item.Primary'>
											<el-option label='0' value='0'></el-option>
											<el-option label='1' value='1'></el-option>
										</el-select>
									</el-form-item>
								</div>
								<div class="plmnListDelIcon">
									<span class="el-icon el-icon-circle-close" @click="addNrPlmnListDel(item)"></span>
								</div>
								<div class="sliceListBoxCls" v-show="!item.operateType || item.operateType && item.operateType != 'add'">
									<el-collapse v-model="SliceListCollapseData[index]">
										<el-collapse-item name="Slice">
											<template slot='title'>
												<p style="display:inline-block;margin-left:40px;">
													<span style="font-size:14px;font-weight:bold">Slice List</span>
													<span style="font-size:12px;color:#999999;margin-right:10px;">（No more than 6）</span>
													<span class="el-icon el-icon-circle-add" @click="addSliceListClick(index)" v-show="sliceListAddShow(item.sliceList)" style="position:relative;top:2px;"></span>
												</p>
											</template>
											<div style="padding:15px 20px 15px 15px;">
												<div v-for="(items,sliceIndex) in (item.sliceList)" class="plmnItemSliceListCls" v-show="!items.operateType || (items.operateType && items.operateType != 'remove')">
													<div style="display:flex;flex-wrap: wrap;font-size:12px">
														<el-form-item :prop="'PlmnList['+index+'].sliceList['+sliceIndex+'].NguIp'" label="NGU IP"  label-width="160px" style="width:40%;min-width:210px;">
															<el-select v-model='items.NguIp'>
																<el-option v-for="(its,index) in nguIpList" :label='its.name' :value='its.value'></el-option>
															</el-select>
														</el-form-item>
														<el-form-item :prop="'PlmnList['+index+'].sliceList['+sliceIndex+'].SNSSAI'" :rules="addPLMNDialogRules.SNSSAI" label="SNSSAI"  label-width="160px" style="width:40%;min-width:360px;" class='validate-item'> 
															<el-input v-model.trim='items.SNSSAI' style="width:160px;">
																<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,Integer</template>
															</el-input>
														</el-form-item>
													</div>
													<div class="plmnListDelIcon">
														<span class="el-icon el-icon-circle-close" @click="sliceListDel(index,sliceIndex,items)"></span>
													</div>
												</div>
											</div>
										</el-collapse-item>
									</el-collapse>
								</div>
							</div>
						</div>
						<el-form-item prop='PlmnList' style="display:none;" label-width="0px">
							<el-input v-model='addPLMNDialogForm.PlmnList'></el-input>
						</el-form-item>
					</el-collapse-item>
				</el-collapse>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addPLMNDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addPLMNDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gnbQuickSettingPage = new Vue({
	el: '#gnbQuickSettingPage', 
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
			validateAMF_IPAddress= (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				if(vm.tbType !== 'AMF'){
					callback()
				}else{
					if(value === ''){
						callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
					}else{
						if(vm.isValidIP(value) || vm.isIPv6(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
						}
					}
				}
			},
			validateAMF_PLMNID = (rule,value,callback) => {
				var reg = /^[0-9]{5,6}$/
				if(vm.tbType !== 'AMF'){
					callback()
				}else{
					if(value === ''){
						callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}
					}
				}
			},
			validatePLMNID = (rule,value,callback) => {
				var reg = /^[0-9]{5,6}$/
				if(reg.test(value)){
					callback();
				}else{
					callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
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
						callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
					}
				}
			},
			validateIPv4AndIPv6 = (rule,value,callback)=>{

				if(value == '' || value == undefined || value == null){
					callback()
				}else{
					if(vm.isValidIP(value) || vm.isIPv6(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}
				}
			},
			validateNRARFCNDL = (rule,value,callback) => {
				if(value === '' || value === null || value === undefined){
					if(this.sasEnableStatus){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
					}
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
					if(this.sasEnableStatus){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
					}
				}else{
					if(vm.isInteger(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
					}
				}
			},
			validatePowerModifyRange = (rule,value,callback)=>{
				var min = this.PowerMinLimitValue,
					max = this.PowerMaxLimitValue,
					mag = '<%=rb.getString("FanWei")%>： ' + min + '~' + max,
					reg = /^[-]?\d*\.?\d*$/;

				if(value == '' || value == undefined || value == null){
					if(this.sasEnableStatus){
						callback();
					}else{
						callback(new Error(mag))
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			},
			validateBandRange = (rule,value,callback)=>{
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
					if(this.isSharingBaseStation){
						callback();
					}else{
						if(reg.test(value) && value >= min && value <= max){
							callback();
						}else{
							callback(new Error(mag))
						}
					}
				}
			};
		return {
			activeCollapse:['AMF','CU','DU','GNB','PLMN','CELL'],
			rowDataInfo: [],
			smallCellCode:'',
			activeNrRows:{},
			nrCellCollapse:['Cell','Plmn'],
			SliceListCollapseData:{
				'0':['Slice'],
				'1':['Slice'],
				'2':['Slice'],
				'3':['Slice'],
				'4':['Slice'],
				'5':['Slice'],
			},
			nguIpList:[],
			ruleForm:{
				AMFList:[],

				CU_F1_C_Local_IP:'',
				CU_F1_U_Local_IP:'',
				CU_NG_C_Local_IP:'',
				CU_NG_U_Local_IP:'',
				CU_Xn_Local_IP:'',

				DU_F1_C_Local_IP:'',
				DU_F1_U_Local_IP:'',
				DU_F1_C_Remote_IP:'',
				DU_PAT1DLULTransPeriodicity:'',
				DU_PAT1ofDownlinkSlots:'',
				DU_PAT1ofDownlinkSymbols:'',
				DU_PAT1ofUplinkSlots:'',
				DU_PAT1ofUplinkSymbols:'',
				DU_PAT2DLULTransPeriodicity:'',
				DU_PAT2ofDownlinkSlots:'',
				DU_PAT2ofDownlinkSymbols:'',
				DU_PAT2ofUplinkSlots:'',
				DU_PAT2ofUplinkSymbols:'',

				gnbLength:'',
				gnbName:'',
				gnbId:'',
				adminState:'1',

				multiPlmnEnable:'0',
				NRCellList:[],

				band: '',
				NRARFCNDL:'',
				NRARFCNUL:'',
				Pci:'',
				NumOfTxAtenna:'1',
				NumOfRxAtenna:'1',
				rftxEnable:'0',
				DL_SubCarrierSpacing:'1',
				DL_CarrierBandWidth:'',
				UL_SubCarrierSpacing:'1',
				UL_CarrierBandWidth:'',
				PowerMaxLimit:40,
                PowerMinLimit:0,
				PowerModify:'',
				offsetToPointA:'',
				ssbSubCarrierOffset:'',
				// ssbGSCN:''
			},
			rules:{
				CU_F1_C_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
				CU_F1_U_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
				CU_Xn_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],

				DU_F1_C_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
				DU_F1_U_Local_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
				DU_F1_C_Remote_IP:[{validator:validateIPv4AndIPv6,trigger:'blur'}],
				DU_PAT1ofDownlinkSlots:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:320,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~320,Integer'}],
				DU_PAT1ofDownlinkSymbols:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~13,Integer'}],
				DU_PAT1ofUplinkSlots:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:320,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~320,Integer'}],
				DU_PAT1ofUplinkSymbols:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~13,Integer'}],
				DU_PAT2ofDownlinkSlots:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:320,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~320,Integer'}],
				DU_PAT2ofDownlinkSymbols:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~13,Integer'}],
				DU_PAT2ofUplinkSlots:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:320,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~320,Integer'}],
				DU_PAT2ofUplinkSymbols:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~13,Integer'}],

				gnbLength:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:22,max:32,isRequired:true,mag:'<%=rb.getString("FanWei")%>：22~32,Integer'}
				],
				gnbId:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:4294967295,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~4294967295,Integer'}
				],
				NRARFCNDL:[
					{validator:validateNRARFCNDL,trigger:'blur'}
				],
				NRARFCNUL:[
					{validator:validateNRARFCNUL,trigger:'blur'}
				],
				Pci:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:1007,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~1007,Integer'}
				],
				PowerModify:[
					{required:true,trigger:'blur'},
					{validator:validatePowerModifyRange}
				],
				offsetToPointA:[
					{validator:validateRange,min:0,max:2199,isRequired:false,mag:'<%=rb.getString("FanWei")%>：0~2199,Integer'}
				],
				ssbSubCarrierOffset:[
					{validator:validateRange,min:0,max:31,isRequired:false,mag:'<%=rb.getString("FanWei")%>：0~31,Integer'}
				],
				band:[
					{validator:validateBandRange,min:1,max:1000,isRequired:false,mag:'<%=rb.getString("FanWei")%>：1~1000,Integer'}
				],
				// ssbGSCN:[
				// 	{validator:validateRange,min:0,max:31,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~31,Integer'}
				// ],
			},
			
			CU_NG_C_Local_IPStr:'',
			CU_NG_U_Local_IPStr:'',
			NGC_LocalIPList:[],
			NGC_LocalIPErrorMessage:'',
			NGU_LocalIPList:[],
			NGU_LocalIPErrorMessage:'',
			casts:{
				'B7EF0230C260BFB555FAE99A93662E28':'AMFList',
				'D2EE312531CA9ED69741523F9CA5B151':'AMF_idx',
				'3CE9BF9752F667F6AF50EC00A823E72C':'AMF_IP',
				'F4B3ED9972A6303F1345DC27971AAE8B':'AMF_PLMNID',
				'0E6C432B61CD906596E8CC4B13445F7B':'AMF_Default',

				'4E4B0794F3F3C5F8AACA85D8DAFD0F73':'CU_F1_C_Local_IP',
				'42B352CBDEC5CE9D4769A49B668E8D25':'CU_F1_U_Local_IP',
				'B84A0C2B6DBF02F1310CE12FD46A06CA':'CU_NG_C_Local_IP',
				'2B20F3BC72DABBBD557D2DEF43AC79E8':'CU_NG_U_Local_IP',
				'76B29306617A5D2D9798A3467C7CFC13':'CU_Xn_Local_IP',

				'A64BC450133733601405A8B7160CDD7F':'DU_F1_C_Local_IP',
				'3D86ABF140171F229983EDB6CC6600F2':'DU_F1_U_Local_IP',
				'8F807FCD64D888C3C51FEA2F7A17D315':'DU_F1_C_Remote_IP',
				'E2A226C74C0F4A9BD5F6F6185625A579':'DU_PAT1DLULTransPeriodicity',
				'B814822900B071870F81C425BE44A02A':'DU_PAT1ofDownlinkSlots',
				'26F28EF787AC887E9FDDD4931AE3BA4B':'DU_PAT1ofDownlinkSymbols',
				'48FD3B2AF2CAD1A866888BADDE0EC493':'DU_PAT1ofUplinkSlots',
				'175363B510FA96F6FF739235A95210C8':'DU_PAT1ofUplinkSymbols',
				'5D952A79117282150B57E73A54BAB8F0':'DU_PAT2DLULTransPeriodicity',
				'834E878B1E39039638493CF1B51301F6':'DU_PAT2ofDownlinkSlots',
				'916B56313EF83E5B4B3BEEA0C9AD2008':'DU_PAT2ofDownlinkSymbols',
				'161570C3CF7B6CB9312E54CCA96A7288':'DU_PAT2ofUplinkSlots',
				'A3AFAA704B54082127B8F20AC66F68AE':'DU_PAT2ofUplinkSymbols',

				'36747EE98FBA37A491196CCA20E88E8D':'gnbLength',
				'BD0CA05B42E723AD02312D242BC05B87':'gnbName',
				'CF5510F1F52BCB149062D869C481A2EB':'gnbId',
				'F7BBCDB03FB7643B3A16A849128940F4':'adminState',

				'51599B5261D829AC73FA8D43D440C793':'band',
				'38F7A00CF211E9547C206D0FCA8562BA':'NRARFCNDL',
				'113AC93D4C61F0E8109AABF94B0634D2':'NRARFCNUL',
				'FC17821A355F4DBA8A760DFE88A7F26E':'Pci',
				'C1EC39F74E7EC6D061BCEDD21E313B99':'NumOfTxAtenna',
				'8F69D5A3687CE1A38B860E60572620E9':'NumOfRxAtenna',
				'B42C4A4528CB6254DF778FC33906ED9A':'rftxEnable',
				'BF669A31B39B13037DCF2464BE69FDDB':'DL_SubCarrierSpacing',
				'08C4EC2D3C5C9728412FA3969CCD31B0':'DL_CarrierBandWidth',
				'8FE5D8F3DFC96BA79D5C291076F69009':'UL_SubCarrierSpacing',
				'A7F9FF6212AB5A36A22DD83CC3F0DD3B':'UL_CarrierBandWidth',
				'0820982DBA2D510432398691643A0524':'PowerModify',
				'1BD33991CDEA88370D1FF04A46749C57':'PowerMaxLimit',
                'BD1E9319755EE6713EC8C391478B82FB':'PowerMinLimit',
				'BB41CC3F46B267F02A84318A745D6598':'offsetToPointA',
				'8CB0E033165A63A8376007555710CAE5':'ssbSubCarrierOffset',
				// 'CF5510F1F52BCB149062D869C481A2EB':'ssbGSCN',

				'8863A2D15601150146F772A537526E5C':'multiPlmnEnable',
				'B779CD488ABB5033764865947EB012D2':'NRCellList',
				'218AF783286C52ACA1C655C51360E12A':'NRCell_idx',
				'C3BF1E75C209CA6A74D9E688ECBF72F1':'NRCellIdentity',
				'6B708E6294AFCE8EEF4158ABDED5339C':'NRCellTAC',
				'08966670FA93B4E29B484EABBFF1CD99':'NRCellRanac',


				'BD9DBC3DA704A5B8FD894D456A96961E':'PlmnList',
				'FDC50DC1194F473C4EBD38BB6A2C76FC':'Plmn_idx',
				'4F7CCEF2044731557F5B9F568FA4D7B9':'PlmnId',
				'9669A2A5A94A97B78AC014FD15DB9C16':'Primary',

				'0CE37FD8EB99D752FD5E48E5D0A7AC6B':'SliceList',
				'338E7490AC59BBD1DFE3590BCBE4A9E6':'Slice_idx',
				'7BFD701CCE9C0D16F8F7610BA6559AD1':'SNSSAI',
				'A27A21EE511324D3DC749B80D2A3AA93':'NguIp',
			},
			codeList:[],
			optType:'',
			tbType:'',
			addAMFDialogShow:false,
			addAMFDialogForm:{
				AMF_IP:'',
				AMF_PLMNID:'',
				AMF_Default:'0',
			},
			addAMFDialogRules:{
				AMF_IP:[{required:true,trigger:'blur'},{validator:validateAMF_IPAddress,trigger:'blur'}],
				AMF_PLMNID:[{required:true,trigger:'blur'},{validator:validateAMF_PLMNID,trigger:'blur'}],
			},
			addPLMNDialogShow:false,
			addPLMNDialogForm:{
				NRCellIdentity:'',
				NRCellTAC:'',
				NRCellRanac:'',
				PlmnList:[],
			},
			addPLMNDialogRules:{
				NRCellIdentity:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:68719476735,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~68719476735,Integer'}
				],
				NRCellTAC:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:16777215,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~16777215,Integer'}
				],
				NRCellRanac:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:255,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0-255,Integer'}
				],
				PlmnId:[
					{required:true,trigger:'blur'},
					{validator:validatePLMNID,trigger:'blur'}
				],
				SNSSAI:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:4294967295,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0-4294967295,Integer'}
				],
			},
			oldPlmnList:[],
			sasEnableStatus:false,
			bandAndPowerMaxList:[],
			isSharingBaseStation:false,
		};
	},
	computed: {
		DL_CarrierBandWidthData() {
			var code={
				'0':[
					{label:'5MHz(25)',value:'25'},
					{label:'10MHz(52)',value:'52'},
					{label:'15MHz(79)',value:'79'},
					{label:'20MHz(106)',value:'106'},
					{label:'25MHz(133)',value:'133'},
					{label:'30MHz(160)',value:'160'},
					{label:'40MHz(216)',value:'216'},
					{label:'50MHz(270)',value:'270'}
				],
				'1':[
					{label:'5MHz(11)',value:'11'},
					{label:'10MHz(24)',value:'24'},
					{label:'15MHz(38)',value:'38'},
					{label:'20MHz(51)',value:'51'},
					{label:'25MHz(65)',value:'65'},
					{label:'30MHz(78)',value:'78'},
					{label:'40MHz(106)',value:'106'},
					{label:'50MHz(133)',value:'133'},
					{label:'60MHz(162)',value:'162'},
					{label:'70MHz(189)',value:'189'},
					{label:'80MHz(217)',value:'217'},
					{label:'90MHz(245)',value:'245'},
					{label:'100MHz(273)',value:'273'}
				],
				'2':[
					{label:'10MHz(11)',value:'11'},
					{label:'15MHz(18)',value:'18'},
					{label:'20MHz(24)',value:'24'},
					{label:'25MHz(31)',value:'31'},
					{label:'30MHz(38)',value:'38'},
					{label:'40MHz(51)',value:'51'},
					{label:'50MHz(65)',value:'65'},
					{label:'60MHz(79)',value:'79'},
					{label:'70MHz(93)',value:'93'},
					{label:'80MHz(107)',value:'107'},
					{label:'90MHz(121)',value:'121'},
					{label:'100MHz(135)',value:'135'}
				]
			}
			
			return code[this.ruleForm.DL_SubCarrierSpacing]
		},
		UL_CarrierBandWidthData() {
			var code={
				'0':[
					{label:'5MHz(25)',value:'25'},
					{label:'10MHz(52)',value:'52'},
					{label:'15MHz(79)',value:'79'},
					{label:'20MHz(106)',value:'106'},
					{label:'25MHz(133)',value:'133'},
					{label:'30MHz(160)',value:'160'},
					{label:'40MHz(216)',value:'216'},
					{label:'50MHz(270)',value:'270'}
				],
				'1':[
					{label:'5MHz(11)',value:'11'},
					{label:'10MHz(24)',value:'24'},
					{label:'15MHz(38)',value:'38'},
					{label:'20MHz(51)',value:'51'},
					{label:'25MHz(65)',value:'65'},
					{label:'30MHz(78)',value:'78'},
					{label:'40MHz(106)',value:'106'},
					{label:'50MHz(133)',value:'133'},
					{label:'60MHz(162)',value:'162'},
					{label:'70MHz(189)',value:'189'},
					{label:'80MHz(217)',value:'217'},
					{label:'90MHz(245)',value:'245'},
					{label:'100MHz(273)',value:'273'}
				],
				'2':[
					{label:'10MHz(11)',value:'11'},
					{label:'15MHz(18)',value:'18'},
					{label:'20MHz(24)',value:'24'},
					{label:'25MHz(31)',value:'31'},
					{label:'30MHz(38)',value:'38'},
					{label:'40MHz(51)',value:'51'},
					{label:'50MHz(65)',value:'65'},
					{label:'60MHz(79)',value:'79'},
					{label:'70MHz(93)',value:'93'},
					{label:'80MHz(107)',value:'107'},
					{label:'90MHz(121)',value:'121'},
					{label:'100MHz(135)',value:'135'}
				]
			}
			return code[this.ruleForm.UL_SubCarrierSpacing]
		},
		NRCellAddShow(){
			var arr = this.ruleForm.NRCellList.filter((item)=>{
				return  !item.operateType || (item.operateType &&item.operateType != 'remove')
			})
			return arr.length<6? true : false;
		},
		plmnSettingShow(){
			var rowData = this.activeNrRows;
			return this.optType == 'edit' && (!rowData.operateType || (rowData.operateType &&rowData.operateType != 'add'))
		},
		sliceListAddShow(){
			return (items)=>{
				if(items){
					var arr = items.filter((item)=>{
						return  !item.operateType || (item.operateType &&item.operateType != 'remove')
					})
					return arr.length<6? true : false;
				}else{
					return true
				}
				
			}
		},
		gnbConfigAddDialogTitle(){
			return this.optType == 'add' ? '<%=rb.getString("TianJia")%>' : '<%=rb.getString("XiuGai")%>'
		},
		PowerMaxLimitValue(){
			var arr = this.bandAndPowerMaxList,
				maxValue = 40;
			if(arr.length == 0){
				if(this.ruleForm.PowerMaxLimit){
					maxValue = this.ruleForm.PowerMaxLimit;
				}else{
					maxValue = 40;
				}
			}else{
				arr.map((item)=>{
					if(item.label == this.ruleForm.band){
						maxValue = item.value;
					}
				})
			}
			return maxValue;
		},
	        PowerMinLimitValue(){
	            var minValue = 0;
	            if(this.ruleForm.PowerMinLimit || this.ruleForm.PowerMinLimit === 0){
	                minValue = this.ruleForm.PowerMinLimit;
	            }else{
	                minValue = 0;
	            }
	            return minValue;
	        },
	},
	watch: {
		// 
		NGC_LocalIPList(){
			var data = this.NGC_LocalIPList;
			this.ruleForm.CU_NG_C_Local_IP = this.NGC_LocalIPList.join(',');
		},	
		// 
		NGU_LocalIPList(){
			var data = this.NGU_LocalIPList;
			this.ruleForm.CU_NG_U_Local_IP = this.NGU_LocalIPList.join(',');
		},
		oldPlmnList(){
			var data = this.oldPlmnList;
			if(data != null && data != undefined && data != ''){
				var dataVal = JSON.stringify(data);
				this.addPLMNDialogForm.PlmnList =JSON.parse(dataVal);
				initForm(this.$refs.addPLMNDialogForm);
			}
		},
	},
	methods: {
		init(row,sasEnableStatus){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = row.small_cell_code;
			vm.sasEnableStatus = sasEnableStatus;
			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'23000');
			var params={
					smallCellCode: vm.smallCellCode
				},
				urls='${ctx}/cell/quicksettings/queryNguIpIndex.action';
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				vm.nguIpList = data;
			})
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
					initForm(vm.$refs.ruleForm);
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
					if(key == 'CU_NG_C_Local_IP' && value){
						vm.NGC_LocalIPList = value.split(',');
					}
					if(key == 'CU_NG_U_Local_IP' && value){
						vm.NGU_LocalIPList = value.split(',');
					}
					if(key == 'PowerMaxLimit'){
						if(value == '' || value == null || value == undefined){
							value = 40;
							vm.bandAndPowerMaxList = [];
							vm.isSharingBaseStation = false;
						}else{
							if(value.indexOf('=') != -1){
								var bandAndPowerMaxArr = value.split(',');
								bandAndPowerMaxArr.map((items)=>{
									var bandAndPower = items.split('=');
									vm.bandAndPowerMaxList.push({label:bandAndPower[0],value:(bandAndPower[1])*1});
								})
								vm.isSharingBaseStation = true;
							}else{
								value = value*1;
								vm.bandAndPowerMaxList = [];
								vm.isSharingBaseStation = false;
							}
						}
					}
		                    if(key == 'PowerMinLimit'){
		                        if(value == '' || value == null || value == undefined){
		                            value = 0;
		                        }else{
		                            value = value*1;
		                        }
		                    }
					vm.ruleForm[key] = value;
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					cellIndex:'1',
					smallCellCode : vm.smallCellCode
			}
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							codes.push(key);
							obj[vm.casts[key]] = item[key]
						}
						if(type == 'AMF List'){
							vm.ruleForm.AMFList.push(obj);
						}else if(type == 'NR Cell List'){
							vm.ruleForm.NRCellList.push(obj);
						}
					})
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					AMFList:[],

				CU_F1_C_Local_IP:'',
				CU_F1_U_Local_IP:'',
				CU_NG_C_Local_IP:'',
				CU_NG_U_Local_IP:'',
				CU_Xn_Local_IP:'',

				DU_F1_C_Local_IP:'',
				DU_F1_U_Local_IP:'',
				DU_F1_C_Remote_IP:'',
				DU_PAT1DLULTransPeriodicity:'',
				DU_PAT1ofDownlinkSlots:'',
				DU_PAT1ofDownlinkSymbols:'',
				DU_PAT1ofUplinkSlots:'',
				DU_PAT1ofUplinkSymbols:'',
				DU_PAT2DLULTransPeriodicity:'',
				DU_PAT2ofDownlinkSlots:'',
				DU_PAT2ofDownlinkSymbols:'',
				DU_PAT2ofUplinkSlots:'',
				DU_PAT2ofUplinkSymbols:'',

				gnbLength:'',
				gnbName:'',
				gnbId:'',
				adminState:'1',

				multiPlmnEnable:'0',
				NRCellList:[],

				band: '',
				NRARFCNDL:'',
				NRARFCNUL:'',
				Pci:'',
				NumOfTxAtenna:'1',
				NumOfRxAtenna:'1',
				rftxEnable:'0',
				DL_SubCarrierSpacing:'1',
				DL_CarrierBandWidth:'',
				UL_SubCarrierSpacing:'1',
				UL_CarrierBandWidth:'',
				PowerModify:'',
				offsetToPointA:'',
				ssbSubCarrierOffset:'',
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
		// 打开新增 AMF弹窗
		addAMFDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addAMFDialogForm,row)
			}
			vm.addAMFDialogShow = true;
		},
		// 新增 AMF提交
		addAMFDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'AMF':'AMFList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addAMFDialogForm).forEach(function(key){
				if(key.substring(0,3) == vm.tbType){
					params[key] = vm.addAMFDialogForm[key]
				}
			})
			if(vm.addAMFDialogForm.operateType){
				params.operateType = vm.addAMFDialogForm.operateType
			}
			vm.$refs.addAMFDialogForm.validate(function(valid){
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
					vm.addAMFDialogShow = false;
				}
			})
		},
		// 删除 Amf
		delAMFList(row){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm.AMFList.map(function(item,index){
					if(item.AMF_idx == row.AMF_idx){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm.AMFList,index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm.AMFList = vm.ruleForm.AMFList.filter((items)=>{
						return items.AMF_idx != row.AMF_idx
					})
				}
			})
		},
		closeAddAMFDialog(){
			var vm = this,
				params = {
					AMF_IP:'',
					AMF_PLMNID:'',
					AMF_Default:'0',
				};
			Object.assign(vm.addAMFDialogForm,params);
			vm.$refs.addAMFDialogForm.clearValidate();
		},
		// 打开新增 PLMN弹窗
		addPLMNDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			
			if(vm.optType == 'edit'){
				vm.activeNrRows = row;
				Object.assign(vm.addPLMNDialogForm,row);
				if(!row.operateType || (row.operateType &&row.operateType != 'add')){
					vm.initPlmnList();
				}
			}
			vm.addPLMNDialogShow = true;
		},
		// PLMN List Slice List回显
		initPlmnList(){
			var vm = this
				params = {
					cellIndex:'1',
					taIndex:vm.activeNrRows.NRCell_idx,
					smallCellCode: vm.smallCellCode
				},
				urls='${ctx}/cell/quicksettings/getListParamValue.action?parent_id=231051&platform=BaiBNQ';

			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					var PlmnList=[];
					data.rows.map(item=>{
						var plmn = {};
						for(var key in item){
							plmn[vm.casts[key]] = item[key]
						}
						PlmnList.push(plmn);
					})
					if(PlmnList.length>0){
						PlmnList.map(item=>{
							var sliceList=[],
								sliceParams = {
									cellIndex: '1',
									taIndex: vm.activeNrRows.NRCell_idx,
									plmnIndex: item.Plmn_idx,
									smallCellCode: vm.smallCellCode
								},
								sliceUrl = '${ctx}/cell/quicksettings/getListParamValue.action?parent_id=2310511&platform=BaiBNQ';
							axios.post(sliceUrl,stringify(sliceParams)).then(res=>{
								var sliceData = res.data;
								if(sliceData.rows){
									sliceData.rows.map(sliceItem=>{
										var slice = {};
										for(var key in sliceItem){
											slice[vm.casts[key]] = sliceItem[key]
										}
										sliceList.push(slice);
									})
									vm.$set(item,'sliceList',sliceList);
								}
							})
						})
					}
					vm.oldPlmnList = PlmnList;
					initForm(vm.$refs.addPLMNDialogForm);
				}
			})
		},
		// 新增 PLMN提交
		addPLMNDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'NRCell':'NRCellList'
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addPLMNDialogForm).forEach(function(key){
				if(key != 'PlmnList'){
					params[key] = vm.addPLMNDialogForm[key]
				}
			})
			if(vm.addPLMNDialogForm.operateType){
				params.operateType = vm.addPLMNDialogForm.operateType
			}
			vm.$refs.addPLMNDialogForm.validate(function(valid){
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
						vm.addPLMNDialogShow = false;
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
						Object.assign(vm.ruleForm[optTb][idx],params);
						if(params.operateType = 'edit'){
							vm.PlmnListAndSliceListSubmit();
						}else{
							vm.addPLMNDialogShow = false;
						}
					}
				}
			})
		},
		// PLMN List Slice List提交
		PlmnListAndSliceListSubmit(){
			var vm=this,
				params={},
				changePlmnList=[],
				changeSliceList=[]
				isSync = false;
			vm.$refs.addPLMNDialogForm.fields.map(function(field){

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						newList = vList.sort(),
						oldList = oList.sort();
					for(i=0;i<oldList.length; i++){
						if(!newList[i].operateType){
							if(newList[i].PlmnId != oldList[i].PlmnId || newList[i].Primary != oldList[i].Primary){
								newList[i].operateType = 'edit'
							}
							if(newList[i].sliceList&&newList[i].sliceList.length>0){
								newList[i].sliceList.map((item,index)=>{
									if(!item.operateType){
										var sliceVal = JSON.stringify(item);
										var oldSliceItem = oldList[i].sliceList.find((items)=>{
											return items.Slice_idx == item.Slice_idx
										})
										var sliceOrVal = JSON.stringify(oldSliceItem);
										if(sliceVal != sliceOrVal) item.operateType = 'edit';
									}
									
								})
							}
						}
					}
					
					newList.map((item,index)=>{
						if(item.operateType){
							var plmn={
									PlmnId:item.PlmnId,
									Primary:item.Primary,
									cellIndex:'1',
									taIndex:vm.activeNrRows.NRCell_idx,
									operateType:item.operateType
								}
							if(item.Plmn_idx){
								plmn.Plmn_idx = item.Plmn_idx;
							}
							changePlmnList.push(plmn)
							if(item.operateType == 'add'){
								isSync = true;
							}
						}
						if(item.sliceList&&item.sliceList.length>0){
							item.sliceList.map((items,indexs)=>{
								if(items.operateType){
									var slice={
											SNSSAI:items.SNSSAI,
											NguIp:items.NguIp,
											cellIndex:'1',
											taIndex:vm.activeNrRows.NRCell_idx,
											plmnIndex:item.Plmn_idx,
											operateType:items.operateType
										}
									if(items.Slice_idx){
										slice.Slice_idx = items.Slice_idx;
									}
									changeSliceList.push(slice)
									if(item.operateType == 'add'){
										isSync = true;
									}
								}
							})
						}
					})
					if(changePlmnList.length>0){
						var subPlmnList = [];
						changePlmnList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								objs[listKey] = items[listVal]
							}
							subPlmnList.push(objs)
						})
						params['BD9DBC3DA704A5B8FD894D456A96961E'] = subPlmnList;

					}
					if(changeSliceList.length>0){
						var subSliceList = [];
						changeSliceList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								objs[listKey] = items[listVal]
							}
							subSliceList.push(objs)
						})
						params['0CE37FD8EB99D752FD5E48E5D0A7AC6B'] = subSliceList;
					}
					
				}
				
			});
			if(changePlmnList.length>0 || changeSliceList.length>0){
				var rowCode = vm.smallCellCode,
					url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
				var confirmStr = '<%=rb.getString("5GPLMNSheZhiTiJiaoTiShi")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							if(isSync){
								vm.syncParams();
							}
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							vm.addPLMNDialogShow = false;
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(()=>{
					
				})
			}else{
				vm.addPLMNDialogShow = false;
			}
			
		},
		// 同步
		syncParams(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				codes = {
					'BaiBNQ':'B779CD488ABB5033764865947EB012D2',
				},
				params={
					smallCellCode: vm.smallCellCode,
					paramId:codes['BaiBNQ'],
					cellIndex:'1'
				};
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
			})
		},
		// 删除 pLMN
		delPLMNList(row){
			var vm =this;
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm.NRCellList.map(function(item,index){
					if(item.NRCell_idx == row.NRCell_idx){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm.NRCellList,index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm.NRCellList = vm.ruleForm.NRCellList.filter((items)=>{
						return items.NRCell_idx != row.NRCell_idx
					})
				}
			})
		},
		closeAddPLMNDialog(){
			var vm = this,
				params = {
					NRCellIdentity:'',
					NRCellTAC:'',
					NRCellRanac:'',
					PlmnList:[]
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
								editList.push(items)
							}
						})
						editList.map((items)=>{
							if(items.operateType == 'add'){
								Object.keys(items).map((key)=>{
									if(key.slice(-3) == 'idx'){
										delete items[key]
									}
								})
							}
						})
						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								objs[listKey] = items[listVal];
							}
							objs.cellIndex = '1';
							subList.push(objs)
						})
						
						params[key] = subList;
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						var editData={
								cellIndex:'1',
								value:field.fieldValue
							}
						params[key] = editData;
					};
				}
			});
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
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
		// NG C Local IP新增
		addNGCLocalIP(){
			var vm = this,
				val = vm.CU_NG_C_Local_IPStr;
			if(val){
				if(vm.isValidIP(val) || vm.isIPv6(val)) {
					var result = vm.NGC_LocalIPList.some(item=>item == val);
					if(result){
						vm.NGC_LocalIPErrorMessage = '<%=rb.getString("YiCunZai")%>';
					}else{
						vm.NGC_LocalIPList.push(val);
						vm.CU_NG_C_Local_IPStr = '';
						vm.NGC_LocalIPErrorMessage = '';
					}
				}else {
					vm.NGC_LocalIPErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
				}
			}
		},
		// NG C Local IP 删除
		NGCLocalIPListDel(item){
			var vm = this;
			
			vm.NGC_LocalIPList = vm.NGC_LocalIPList.filter((items)=>{
				return items != item
			})
		},
		// NG U Local IP新增
		addNGULocalIP(){
			var vm = this,
				val = vm.CU_NG_U_Local_IPStr;
			if(val){
				if(vm.isValidIP(val) || vm.isIPv6(val)) {
					var result = vm.NGU_LocalIPList.some(item=>item == val);
					if(result){
						vm.NGU_LocalIPErrorMessage = '<%=rb.getString("YiCunZai")%>';
					}else{
						vm.NGU_LocalIPList.push(val);
						vm.CU_NG_U_Local_IPStr = '';
						vm.NGU_LocalIPErrorMessage = '';
					}
				}else {
					vm.NGU_LocalIPErrorMessage = '<%=rb.getString("IPGeShiBuDui")%>';
				}
			}
		},
		// NG U Local IP 删除
		NGULocalIPListDel(item){
			var vm = this;

			vm.NGU_LocalIPList = vm.NGU_LocalIPList.filter((items)=>{
				return items != item
			})
		},
		DL_SCSChange(val){
			var vm = this;
			if(val != '' && val != undefined && val != null){
				setTimeout(()=>{
					vm.ruleForm.DL_CarrierBandWidth = vm.DL_CarrierBandWidthData[0].value;
				},100)
			}
		},
		UL_SCSChange(val){
			var vm = this;
			if(val != '' && val != undefined && val != null){
				setTimeout(()=>{
					vm.ruleForm.UL_CarrierBandWidth = vm.UL_CarrierBandWidthData[0].value;
				},100)
			}
		},
		// 修改NR CELL弹窗中 plmnList 新增
		addNrPlmnAddClick(){
			var vm =this,
				params={
					PlmnId:'',
					Primary:'0',
					sliceList:[],
					operateType:'add',
					taIndex:vm.activeNrRows.NRCell_idx,
				};
			var idList=[];
			vm.addPLMNDialogForm.PlmnList.map((item)=>{
				idList.push(item.Plmn_idx);
			})
			params.Plmn_idx = vm.createId(1,idList); 
			var delNum = 0;
			vm.addPLMNDialogForm.PlmnList.map((item)=>{
				if(item.operateType&&item.operateType == 'remove'){
					delNum+=1;
				}
			})
			if((vm.addPLMNDialogForm.PlmnList.length - delNum) < 6){
				vm.addPLMNDialogForm.PlmnList.push(params)
			}else{
				vm.$message({
					message: 'No more than 6',
					type:'error'
				});
			}
			event.stopPropagation();
		},
		addNrPlmnListDel(row){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.addPLMNDialogForm.PlmnList.map(function(item,index){
					if(item.Plmn_idx == row.Plmn_idx){
						if(item.operateType && item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.addPLMNDialogForm.PlmnList,index,params);
						}
						
					}
				})
				if(delFlag){
					vm.addPLMNDialogForm.PlmnList = vm.addPLMNDialogForm.PlmnList.filter((items)=>{
						return items.Plmn_idx != row.Plmn_idx
					})
				}
			}).catch(()=>{
				
			})
		},
		// 修改NR CELL弹窗中 sliceList 新增
		addSliceListClick(index){
			var vm =this,
				params={
					SNSSAI:'',
					NguIp:'',
					operateType:'add'
				};
			var idList=[];
			vm.addPLMNDialogForm.PlmnList[index].sliceList.map((item)=>{
				idList.push(item.Slice_idx);
			})
			params.Slice_idx = vm.createId(1,idList); 
			var delNum = 0;
			vm.addPLMNDialogForm.PlmnList[index].sliceList.map((item)=>{
				if(item.operateType&&item.operateType == 'remove'){
					delNum+=1;
				}
			})
			if((vm.addPLMNDialogForm.PlmnList[index].sliceList.length - delNum) < 6){
				vm.addPLMNDialogForm.PlmnList[index].sliceList.push(params)
			}else{
				vm.$message({
					message: 'No more than 6',
					type:'error'
				});
			}
			event.stopPropagation();
		},
		// slice 删除
		sliceListDel(index,sliceIndex,row){
			var vm = this;
			
			if(row.Slice_idx){
					vm.$set(vm.addPLMNDialogForm.PlmnList[index].sliceList[sliceIndex],'operateType','remove');
			}else{
				vm.addPLMNDialogForm.PlmnList[index].sliceList.splice(sliceIndex,1);
			}
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
					gnbTabSettingVue.changeMain('quickSetting_GT');
				}else{
					vm.$message.error(data["message"])
				}
			})
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
		// band 改变事件
		bandSelectChange(){
			var vm = this;
			vm.$refs.ruleForm.validateField('PowerModify');
		},
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data",this.init)
	}
});

</script>
