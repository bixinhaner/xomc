<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gnbBtsPage{
	height: 100%;
	width: 100%;
}
#gnbBtsPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gnbBtsPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gnbBtsPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gnbBtsPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gnbBtsPage .rightContentCls .contentTableTitle{
	display: flex;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gnbBtsPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gnbBtsPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gnbBtsPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gnbBtsPage .itemListBoxCls{
	padding-top: 5px;
}
#gnbBtsPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gnbBtsPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gnbBtsPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gnbBtsPage .el-form-item{
	margin-bottom: 20px;
}
#gnbBtsPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gnbBtsPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gnbBtsPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
#gnbBtsPage .rightContentCls .el-checkbox.is-bordered.el-checkbox--small{
	padding: 8px 15px 5px 10px;
}
#gnbBtsPage .el-date-editor .el-range-input{
	font-size: 12px;
}
#gnbBtsPage .el-date-editor .el-range__close-icon{
	font-size: 14px!important;
	line-height: 20px;
}
#gnbBtsPage .nlSyncStatusBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
	margin-bottom: 10px;
}
#gnbBtsPage .nlSyncStatusBoxCls >div{
	flex:1;
}
#gnbBtsPage .paramItemLabel{
	color:#7a7992;
}
#gnbBtsPage .paramItemValue{
	height: 18px;
}
#gnbBtsPage .el-select .el-input.is-disabled .el-input__inner,#gnbBtsPage .el-select .el-input .el-input__inner{
    height: 26px !important;
    line-height: 26px;
}
</style>

<div id="gnbBtsPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			BTS
			<!-- 按钮  同步 -->
			<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSettingsClick" tip="<%=rb.getString("TongBu")%>">
				<span class="el-icon el-icon-circle-refresh"></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
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
									<el-input v-model.trim='ruleForm.CU_F1_C_Local_IP' style="width:150px;padding-top:5px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='CU_F1_U_Local_IP' style="width:40%;min-width:400px;" label="F1-U Local IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										F1-U Local IP
									</span>
									<el-input v-model.trim='ruleForm.CU_F1_U_Local_IP' style="width:150px;padding-top:5px;">
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
									<el-input v-model.trim='ruleForm.CU_Xn_Local_IP' style="width:150px;padding-top:5px;">
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
									<el-input v-model.trim='ruleForm.DU_F1_C_Local_IP' style="width:150px;padding-top:5px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_F1_U_Local_IP' style="width:40%;min-width:400px;" label="F1-U Local IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										F1-U Local IP
									</span>
									<el-input v-model.trim='ruleForm.DU_F1_U_Local_IP' style="width:150px;padding-top:5px;">
										<template slot="append">Example：1.1.1.1</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_F1_C_Remote_IP' style="width:40%;min-width:400px;" label="F1-C Remote IP" label-width="160px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										F1-C Remote IP
									</span>
									<el-input v-model.trim='ruleForm.DU_F1_C_Remote_IP' style="width:150px;padding-top:5px;">
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
								<el-form-item prop='DU_PAT2ofDownlinkSlots' style="width:40%;min-width:400px;" label="PAT2 of Downlink Slots" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT2ofDownlinkSlots' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~320,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_PAT2ofDownlinkSymbols' style="width:40%;min-width:400px;" label="PAT2 of Downlink Symbols" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT2ofDownlinkSymbols' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~13,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_PAT2ofUplinkSlots' style="width:40%;min-width:400px;" label="PAT2 of Uplink Slots" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT2ofUplinkSlots' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~320,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='DU_PAT2ofUplinkSymbols' style="width:40%;min-width:400px;" label="PAT2 of Uplink Symbols" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.DU_PAT2ofUplinkSymbols' style="width:150px;">
										<template slot="append"><%=rb.getString("FanWei")%>：0~13,Integer</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="Sync">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Sync Settings</span>
							</p>
						</template>
						<div class="rightContentCls">
							<!--Sync Mode-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">Sync Mode</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='Sync_Mode' style="width:40%;min-width:500px;" label="Mode" label-width="160px">
										<el-select v-model='ruleForm.Sync_Mode'>
											<el-option label='FREE_RUNNING' value='FREE_OSCILLATION'></el-option>
											<el-option label='GNSS' value='GPS_PPS'></el-option>
                                            <el-option label='PTP' value='1588_PPS'></el-option>
                                            <el-option label='GNSS+PTP' value='GPS_AND_PTP'></el-option>
											<el-option v-if="nlModeShow" label='NL_PPS' value='NL_PPS'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
							<!--GNSS Sync-->
							<div v-show="ruleForm.Sync_Mode == 'GPS_PPS' || ruleForm.Sync_Mode == 'GPS_AND_PTP'"> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">GNSS Sync</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='Sync_syncSource' style="width:100%;min-width:400px;" label="Sync Source" label-width="160px">
										<el-checkbox-group v-model="syncSourceSelectList" size="small">
											<el-checkbox v-for="(item,index) in syncSourceList" border :label="item" :key="index" @change="syncSourceItemChange(item)"></el-checkbox>
										</el-checkbox-group>
									</el-form-item>
									<el-form-item v-show="ruleForm.Sync_Mode == 'GPS_PPS'" prop='Sync_ForcedSync' style="width:40%;min-width:400px;" label="Forced Sync" label-width="160px">
										<el-select v-model='ruleForm.Sync_ForcedSync'>
											<el-option label='ON' value='1'></el-option>
											<el-option label='OFF' value='0'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Sync_PPSTimeOffset' style="width:40%;min-width:400px;" label="PPS Time Offset" label-width="160px" class='validate-item'>
										<span slot="label" class="labelIconCls">
											PPS Time Offset(ns)
										</span>
										<el-input v-model.trim='ruleForm.Sync_PPSTimeOffset'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~5000000,Integer</template>
										</el-input>
									</el-form-item>
									<div  style="display:flex;flex-wrap: wrap">
										<el-form-item label='Sync Status' style="width:40%;min-width:400px;" label-width="160px" >
											<el-input v-model='ruleForm.Sync_Status' :disabled="true"></el-input>
										</el-form-item>
										<el-form-item label='Longitude' style="width:40%;min-width:400px;" label-width="160px" >
											<el-input v-model='ruleForm.Sync_Longitude' :disabled="true"></el-input>
										</el-form-item>
										<el-form-item label='Latitude' style="width:40%;min-width:400px;" label-width="160px" >
											<el-input v-model='ruleForm.Sync_Latitude' :disabled="true"></el-input>
										</el-form-item>
										<el-form-item label='Altitude' style="width:40%;min-width:400px;" label-width="160px" >
											<el-input v-model='ruleForm.Sync_Altitude' :disabled="true"></el-input>
										</el-form-item>
									</div>
								</div>
							</div>
                            <!--NL Sync-->
							<div v-show="ruleForm.Sync_Mode == 'NL_PPS'"> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">NL Sync</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<div class="nlSyncStatusBoxCls">
										<div>
											<div class="paramItemLabel">Sync Status</div>
											<div class="paramItemValue" v-if="nlSyncStatusData.syncStatus === '0'">Synchronized</div>
											<div class="paramItemValue" v-if="nlSyncStatusData.syncStatus !== '0'">Synchronized Failure</div>
										</div>
										<div>
											<div class="paramItemLabel">Band</div>
											<div class="paramItemValue">{{nlSyncStatusData.band}}</div>
										</div>
										<div>
											<div class="paramItemLabel">Frequency</div>
											<div class="paramItemValue">{{nlSyncStatusData.frequency}}</div>
										</div>
										<div>
											<div class="paramItemLabel">Cell ID</div>
											<div class="paramItemValue">{{nlSyncStatusData.cellId}}</div>
										</div>
									</div>
									<el-form-item prop='Sync_ForcedSync' style="width:40%;min-width:400px;" label="Forced Sync" label-width="160px">
										<el-select v-model='ruleForm.Sync_ForcedSync'>
											<el-option label='ON' value='1'></el-option>
											<el-option label='OFF' value='0'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Sync_PPSTimeOffset' style="width:40%;min-width:400px;" label="PPS Time Offset" label-width="160px" class='validate-item'>
										<span slot="label" class="labelIconCls">
											PPS Time Offset(ns)
										</span>
										<el-input v-model.trim='ruleForm.Sync_PPSTimeOffset'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~5000000,Integer</template>
										</el-input>
									</el-form-item>
									<div style="display:flex;flex-wrap: wrap">
										<el-form-item prop='Sync_NlLockMode' label='Lock Mode' style="width:40%;min-width:400px;" label-width="160px" >
											<el-select v-model='ruleForm.Sync_NlLockMode'>
												<el-option label='Locked Band' value='1'></el-option>
												<el-option label='Locked Frequency' value='2'></el-option>
												<el-option label='Locked Cell ID' value='3'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item prop='Sync_NlGeneration' label='NL Generation' style="width:40%;min-width:400px;" label-width="160px" >
											<el-select v-model='ruleForm.Sync_NlGeneration'>
												<el-option label='4G' value='4'></el-option>
												<el-option label='5G' value='5'></el-option>
												<el-option label='4G+5G' value='9'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item v-show="ruleForm.Sync_NlLockMode == '1'" prop='Sync_NlLockBand' style="width:40%;min-width:400px;" label="Band" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.Sync_NlLockBand'>
												<template slot="append"><%=rb.getString("FanWei")%>：1~1024,Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item v-show="ruleForm.Sync_NlLockMode == '2' || ruleForm.Sync_NlLockMode == '3'" prop='Sync_NlLockFrequency' style="width:40%;min-width:400px;" label="Frequency" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.Sync_NlLockFrequency'>
												<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item v-show="ruleForm.Sync_NlLockMode == '3'" prop='Sync_NlLockCellId' style="width:40%;min-width:400px;" label="Cell ID" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.Sync_NlLockCellId'>
												<template slot="append"><%=rb.getString("FanWei")%>：0~68719476735,Integer</template>
											</el-input>
										</el-form-item>
									</div>
								</div>
							</div>
                            <!--PTP Sync-->
							<div v-show="ruleForm.Sync_Mode == '1588_PPS' || ruleForm.Sync_Mode == 'GPS_AND_PTP'"> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">PTP Config</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap"> 
                                    <el-form-item prop='Sync_PTP_Profile' style="width:40%;min-width:400px;" label="Profile" label-width="160px">
										<el-select v-model='ruleForm.Sync_PTP_Profile' @change="profileChange">
											<el-option label='1588v2' value='1588v2'></el-option>
											<el-option label='G8275.2' value='G8275.2'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='Sync_PPSTimeOffset' style="width:40%;min-width:400px;" label="PPS Time Offset" label-width="160px" class='validate-item'>
										<span slot="label" class="labelIconCls">
											PPS Time Offset(ns)
										</span>
										<el-input v-model.trim='ruleForm.Sync_PPSTimeOffset'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~5000000,Integer</template>
										</el-input>
									</el-form-item>
                                    <el-form-item prop='Sync_ForcedSync' style="width:40%;min-width:400px;" label="Forced Sync" label-width="160px">
										<el-select v-model='ruleForm.Sync_ForcedSync' :disabled="true">
											<el-option label='ON' value='1'></el-option>
											<el-option label='OFF' value='0'></el-option>
										</el-select>
									</el-form-item>
                                    <el-form-item label='Sync Status' style="width:40%;min-width:400px;" label-width="160px" >
                                        <el-input v-model='ruleForm.Sync_PTP_Status' :disabled="true"></el-input>
                                    </el-form-item>
                                    <el-form-item prop='Sync_PTP_Domain' style="width:40%;min-width:400px;" label="Domain" label-width="160px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.Sync_PTP_Domain'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
                                        </el-input>
                                    </el-form-item>
                                    <el-form-item prop='Sync_PTP_TransferMode' style="width:40%;min-width:400px;" label="Transfer Mode" label-width="160px">
										<el-select v-model='ruleForm.Sync_PTP_TransferMode' :disabled="ruleForm.Sync_PTP_Profile == 'G8275.2'" @change="transferModeChange">
											<el-option label='L2' value='L2'></el-option>
											<el-option label='UDPv4' value='UDPv4'></el-option>
										</el-select>
									</el-form-item>
                                    <el-form-item v-show="ruleForm.Sync_PTP_TransferMode == 'UDPv4'" prop='Sync_PTP_ServerAddress' style="width:40%;min-width:400px;" label="Server Address" label-width="160px">
                                        <el-input v-model.trim='ruleForm.Sync_PTP_ServerAddress'></el-input>
                                    </el-form-item>
                                    <el-form-item prop='Sync_PTP_Interface' style="width:40%;min-width:400px;" label="Interface" label-width="160px">
										<el-select v-model='ruleForm.Sync_PTP_Interface' :disabled="true">
											<el-option v-for="item in interfaceList" :label='item' :value='item'></el-option>
										</el-select>
									</el-form-item>
                                    <el-form-item prop='Sync_PTP_VlanSwitch' style="width:40%;min-width:400px;" label="PTP Vlan Switch" label-width="160px">
										<el-select v-model='ruleForm.Sync_PTP_VlanSwitch' @change='changePTPVlanSwitch'>
											<el-option label='ON' value='1'></el-option>
											<el-option label='OFF' value='0'></el-option>
										</el-select>
									</el-form-item>
                                    <el-form-item v-show="ruleForm.Sync_PTP_VlanSwitch == '1'" prop='Sync_PTP_VLANID' style="width:40%;min-width:400px;" label="PTP VLAN ID" label-width="160px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.Sync_PTP_VLANID'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：1~4094,Integer</template>
                                        </el-input>
                                    </el-form-item>
                                    <el-form-item prop='Sync_PTP_UniCastMode' style="width:40%;min-width:400px;" label="UniCast Mode" label-width="160px">
										<el-select v-model='ruleForm.Sync_PTP_UniCastMode' :disabled="ruleForm.Sync_PTP_Profile == 'G8275.2'">
											<el-option label='Unicast' value='1' v-if="ruleForm.Sync_PTP_Profile !== '1588v2' || ruleForm.Sync_PTP_TransferMode !== 'L2'"></el-option>
											<el-option label='Multicast' value='0'></el-option>
										</el-select>
									</el-form-item>
                                    <el-form-item v-show="ruleForm.Sync_PTP_UniCastMode == '1'" prop='Sync_PTP_IPAddress' style="width:40%;min-width:400px;" label="IP Address" label-width="160px">
                                        <el-input v-model.trim='ruleForm.Sync_PTP_IPAddress'></el-input>
                                    </el-form-item>
                                    <el-form-item prop='Sync_PTP_SyncInterval' style="width:40%;min-width:400px;" label="Sync Interval" label-width="160px">
										<el-select v-model='ruleForm.Sync_PTP_SyncInterval'>
											<el-option v-for="item in intervalList" :label='item' :value='item'></el-option>
										</el-select>
									</el-form-item>
                                    <el-form-item prop='Sync_PTP_DelayInterval' style="width:40%;min-width:400px;" label="Delay Interval" label-width="160px">
										<el-select v-model='ruleForm.Sync_PTP_DelayInterval'>
											<el-option v-for="item in intervalList" :label='item' :value='item'></el-option>
										</el-select>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="Energy">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Energy Saving Setting</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='EnergySavingType' style="width:40%;min-width:400px;" label="EnergySavingType" label-width="160px">
									<el-select v-model='ruleForm.EnergySavingType'>
										<el-option label='Not Saving' value='NOT_SAVING'></el-option>
										<el-option label='Deep Saving' value='DEEP_SAVING'></el-option>
										<el-option label='Shallow Saving' value='SHALLOW_SAVING'></el-option>
										<el-option label="Slot Switch" value="SLOT_SWITCH"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='EnergySavingTime' style="width:40%;min-width:400px;" label="EnergySavingTime" label-width="160px">
									<el-time-picker
										is-range
										style="width:200px;"
										v-model="EnergySavingTimeList"
										@change="EnergySavingTimeListChange"
										range-separator="-"
										value-format="HH:mm:ss"
										start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
										end-placeholder='<%=rb.getString("JieShuShiJian")%>'
									></el-time-picker>
									<span style="margin-left:5px;color:#909399">For example:01:00:00-06:00:00</span>
								</el-form-item>
								<el-form-item prop='EnergySavingDelayTime' style="width:40%;min-width:400px;" label="EnergySavingDelayTime" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.EnergySavingDelayTime'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='PrbLowThreshold' style="width:40%;min-width:400px;" label="PrbLowThreshold" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.PrbLowThreshold'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~100,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='PrbReportPeriod' style="width:40%;min-width:400px;" label="PrbReportPeriod" label-width="170px" class='validate-item'>
									<el-input v-model.trim='ruleForm.PrbReportPeriod'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='RRCLowThreshold' style="width:40%;min-width:400px;" label="RRCLowThreshold" label-width="170px" class='validate-item'>
									<el-input v-model.trim='ruleForm.RRCLowThreshold'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,Integer</template>
									</el-input>
								</el-form-item>
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
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gnbBtsPage = new Vue({
	el: '#gnbBtsPage', 
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
			validateIPaddress= (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				
				if(value === ''){
					callback()
				}else{
					if(vm.isValidIP(value) || vm.isIPv6(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
					}
				}
			},
            validatePPSTimeOffsetRange= (rule,value,callback)=>{
                var min = rule.min;
                var max = rule.max;
                var mag = rule.mag;
                var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

                // 当 Sync_Mode 为 FREE_OSCILLATION 时不校验
                if(vm.ruleForm.Sync_Mode == 'FREE_OSCILLATION'){
                    callback();
                }else{
                    if(value == '' || value == undefined || value == null){
                        callback(new Error(mag))
                    }else{
                        if(reg.test(value) && value >= min && value <= max){
                            callback();
                        }else{
                            callback(new Error(mag))
                        }
                    }
                }
            },
			validateNlLockBandRange= (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

                if(vm.ruleForm.Sync_Mode == 'NL_PPS'){
                    if(value == '' || value == undefined || value == null){
                        if(vm.ruleForm.Sync_NlLockMode == '1'){
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
                }else{
                    callback();
                }
			},
			validateNlLockFrequencyRange= (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
                if(vm.ruleForm.Sync_Mode == 'NL_PPS'){
                    if(value == '' || value == undefined || value == null){
                        if(vm.ruleForm.Sync_NlLockMode == '2' || vm.ruleForm.Sync_NlLockMode == '3'){
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
                }else{
                    callback();
                }
			},
			validateNlLockCellIdRange= (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

                if(vm.ruleForm.Sync_Mode == 'NL_PPS'){
                    if(value == '' || value == undefined || value == null){
                        if(vm.ruleForm.Sync_NlLockMode == '3'){
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
                }else{
                    callback();
                }
			},
            validatePTPRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

                if(vm.ruleForm.Sync_Mode == '1588_PPS' || vm.ruleForm.Sync_Mode == 'GPS_AND_PTP'){
                    if(value == '' || value == undefined || value == null){
                        callback(new Error(mag))
                    }else{
                        if(reg.test(value) && value >= min && value <= max){
                            callback();
                        }else{
                            callback(new Error(mag))
                        }
                    }
                }else{
                    callback();
                }
			},
            validatePTPVlanIdRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

                if((vm.ruleForm.Sync_Mode == '1588_PPS' || vm.ruleForm.Sync_Mode == 'GPS_AND_PTP') && vm.ruleForm.Sync_PTP_VlanSwitch == '1'){
                    if(value == '' || value == undefined || value == null){
                        callback(new Error(mag))
                    }else{
                        if(reg.test(value) && value >= min && value <= max){
                            callback();
                        }else{
                            callback(new Error(mag))
                        }
                    }
                }else{
                    callback();
                }
			},
            validatePTPServerAddressRange = (rule,value,callback)=>{

                if((vm.ruleForm.Sync_Mode == '1588_PPS' || vm.ruleForm.Sync_Mode == 'GPS_AND_PTP') && vm.ruleForm.Sync_PTP_TransferMode == 'UDPv4'){
                    if(value == '' || value == undefined || value == null){
                        callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
                    }else{
                        if(vm.isValidIP(value)){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
                        }
                    }
                }else{
                    callback();
                }
			},
            validatePTPIPAddressRange = (rule,value,callback)=>{

                if((vm.ruleForm.Sync_Mode == '1588_PPS' || vm.ruleForm.Sync_Mode == 'GPS_AND_PTP') && vm.ruleForm.Sync_PTP_UniCastMode == '1'){
                    if(value == '' || value == undefined || value == null){
                        callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
                    }else{
                        if(vm.isValidIP(value)){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
                        }
                    }
                }else{
                    callback();
                }
			};
		return {
			activeCollapse:['CU','DU','Sync','Energy'],
			rowDataInfo: [],
			smallCellCode:'',
			syncSourceSelectList:[],
			syncSourceList:['GPS','GLONASS','BEIDOU','GALILEO','QZSS'],
			ruleForm:{

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

				Sync_Mode:'GPS_PPS',
				Sync_syncSource:'',
				Sync_ForcedSync:'0',
				Sync_PPSTimeOffset:'0',
				Sync_Status:'',
				Sync_Longitude:'',
				Sync_Latitude:'',
				Sync_Altitude:'',
				Sync_NlLockMode:'1',
				Sync_NlGeneration:'4',
				Sync_NlLockBand:'',
				Sync_NlLockFrequency:'',
				Sync_NlLockCellId:'',

                Sync_PTP_Profile:'1588v2',
                Sync_PTP_Status:'',
                Sync_PTP_Domain:'0',
                Sync_PTP_TransferMode:'L2',
                Sync_PTP_ServerAddress:'0.0.0.0',
                Sync_PTP_Interface:'',
                Sync_PTP_VlanSwitch:'0',
                Sync_PTP_VLANID:'0',
                Sync_PTP_UniCastMode:'0',
                Sync_PTP_IPAddress:'',
                Sync_PTP_SyncInterval:'-4',
                Sync_PTP_DelayInterval:'0',

				EnergySavingType:'NOT_SAVING',
				EnergySavingTime:'',
				EnergySavingDelayTime:'',
				PrbLowThreshold:'',
				PrbReportPeriod:'',
				RRCLowThreshold:'',
			},
			rules:{
				CU_F1_C_Local_IP:[{validator:validateIPaddress,trigger:'blur'}],
				CU_F1_U_Local_IP:[{validator:validateIPaddress,trigger:'blur'}],
				CU_Xn_Local_IP:[{validator:validateIPaddress,trigger:'blur'}],

				DU_F1_C_Local_IP:[{validator:validateIPaddress,trigger:'blur'}],
				DU_F1_U_Local_IP:[{validator:validateIPaddress,trigger:'blur'}],
				DU_F1_C_Remote_IP:[{validator:validateIPaddress,trigger:'blur'}],
				DU_PAT1ofDownlinkSlots:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:320,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~320,Integer'}],
				DU_PAT1ofDownlinkSymbols:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~13,Integer'}],
				DU_PAT1ofUplinkSlots:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:320,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~320,Integer'}],
				DU_PAT1ofUplinkSymbols:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~13,Integer'}],
				DU_PAT2ofDownlinkSlots:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:320,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~320,Integer'}],
				DU_PAT2ofDownlinkSymbols:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~13,Integer'}],
				DU_PAT2ofUplinkSlots:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:320,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~320,Integer'}],
				DU_PAT2ofUplinkSymbols:[{required:true,trigger:'blur'},{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~13,Integer'}],

				Sync_PPSTimeOffset:[
					{required:true,trigger:'blur'},
					{validator:validatePPSTimeOffsetRange,min:0,max:5000000,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~5000000,Integer'}
				],
				Sync_NlLockBand:[{validator:validateNlLockBandRange,min:1,max:1024,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~1024,Integer'}],
				Sync_NlLockFrequency:[{validator:validateNlLockFrequencyRange,min:0,max:3279165,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer'}],
				Sync_NlLockCellId:[{validator:validateNlLockCellIdRange,min:0,max:68719476735,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~68719476735,Integer'}],
                Sync_PTP_Domain:[{validator:validatePTPRange,min:0,max:127,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~127,Integer'}],
                Sync_PTP_VLANID:[{validator:validatePTPVlanIdRange,min:1,max:4094,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~4094,Integer'}],
                Sync_PTP_ServerAddress:[{validator:validatePTPServerAddressRange}],
                Sync_PTP_IPAddress:[{validator:validatePTPIPAddressRange}],
				EnergySavingDelayTime:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:15,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~15,Integer'}
				],
				PrbLowThreshold:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:100,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~100,Integer'}
				],
				PrbReportPeriod:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:65535,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer'}
				],
				RRCLowThreshold:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:4294967295,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~4294967295,Integer'}
				],
			},
			CU_NG_C_Local_IPStr:'',
			CU_NG_U_Local_IPStr:'',
			NGC_LocalIPList:[],
			NGC_LocalIPErrorMessage:'',
			NGU_LocalIPList:[],
			NGU_LocalIPErrorMessage:'',
			casts:{
				'2F8C27B7CD28B61F802128616E0B83D7':'CU_F1_C_Local_IP',
				'52B87FA9C8B8D0EF55697F46DE13F1E4':'CU_F1_U_Local_IP',
				'7BF9D5CE06B993E87C6BEEEB4F645038':'CU_NG_C_Local_IP',
				'8247770CBDBC39A2A15C1ACE29936041':'CU_NG_U_Local_IP',
				'67C9777F97E6FA250C43180FD866A489':'CU_Xn_Local_IP',

				'E63AE71C54E4FBD64010641801EE9813':'DU_F1_C_Local_IP',
				'7EC22CF0B057767836E1EEA7D0EB2E11':'DU_F1_U_Local_IP',
				'109AA2F9B8A0DDF75819FCE37F98E200':'DU_F1_C_Remote_IP',
				'3DA1E1CF432ABF502E48EC5A81D61F0D':'DU_PAT1DLULTransPeriodicity',
				'E73892FB17938CBEB90A0B0518068F5E':'DU_PAT1ofDownlinkSlots',
				'68A39AC16EE47F0D2DA04AE9301DA36B':'DU_PAT1ofDownlinkSymbols',
				'AA65F5158161D43CB5F105E885665E09':'DU_PAT1ofUplinkSlots',
				'BDD8C0FB1CE1BED9B7083A2DAE2EB6A5':'DU_PAT1ofUplinkSymbols',
				'FA3BEDEE9C58850B87BA6ABD19E0639B':'DU_PAT2DLULTransPeriodicity',
				'354619C3F17FE51A383BD78B34106F7F':'DU_PAT2ofDownlinkSlots',
				'77D6E39506F55D1DC517C6ACA534804B':'DU_PAT2ofDownlinkSymbols',
				'E3003CBE561C849F591190FDF30DF15F':'DU_PAT2ofUplinkSlots',
				'E0D9F036A5212D730B2ECB4C051BD686':'DU_PAT2ofUplinkSymbols',

				'A0E8CB20E1C810EA92B91D36218EAC8D':'Sync_Mode',
				'1E105E23E899B2277FE54FE925B2C943':'Sync_syncSource',
				'A99FFF6D75D09BFB523EC8710447958F':'Sync_ForcedSync',
				'F1BE0ADD4E00908B64698F38E3062519':'Sync_PPSTimeOffset',
				'64FC363430543E441354605F87B27088':'Sync_Status',
				'C72F802951EE784484B1D14246E1EE47':'Sync_Altitude',
				'158C7E2E6E5C6D9353F9CD4D270802CB':'Sync_Longitude',
				'624D9BAFED87EA3FBB0C0691B87850BD':'Sync_Latitude',
				'0EA97DEE2E00C6ADB5993FE19FF24B2F':'Sync_NlSyncStatus',
				'E192FD9B540F63A004453D1268FF688B':'Sync_NlLockMode',
				'D5FB1227EA558917D2BC5BB14D09B5C3':'Sync_NlGeneration',
				'E44BE6162608A0642BFE3E7F5ACB9D1A':'Sync_NlLockBand',
				'E7EB9FDC7E16E14BDBE0B6A418C58D40':'Sync_NlLockFrequency',
				'283882CF01E3CB8369F2E2DD26209855':'Sync_NlLockCellId',

                '410A5E23889E96430398C8C5EDD9ADA8':'Sync_PTP_Profile',
                'D8B8A77D6BEF63F2C0CB5E7F0589F5D7':'Sync_PTP_Status',
                'BE16851C293DAAE26AAFB8B01CD437A4':'Sync_PTP_Domain',
                '3E04039EEF4A48A9CDF19294AB5A322C':'Sync_PTP_TransferMode',
                'DB0DAB52486A79E009C86518DF0F931D':'Sync_PTP_ServerAddress',
                '55284F331374E1EC41120A18EF6A0E00':'Sync_PTP_Interface',
                '4712E372C8858D3E9E240BC40C3F8312':'Sync_PTP_VLANID',
                '776E5CCD13AE47CF01FCA8FA2B8A8E93':'Sync_PTP_UniCastMode',
                '41B27877C690FD5EBE6D547CB36D253E':'Sync_PTP_IPAddress',
                'EE5C8C324576B935BA3F89BB4EC51D69':'Sync_PTP_SyncInterval',
                'FB6B098280D5DFC2FFCFDAB471F6CBB7':'Sync_PTP_DelayInterval',

				'12F21ADB0B26C5342F17F42CF99F7880':'EnergySavingType',
				'D2AC053D212F613CE192B5FAF835A29A':'EnergySavingTime',
				'019A448406D26009641A1986F36D5F0E':'EnergySavingDelayTime',
				'EF90F6A721D47D653562F9EADEC603EE':'PrbLowThreshold',
				'213013ED1109189BFE720764FC783EFB':'PrbReportPeriod',
				'CDC098800D7EEB1D9A4DB83B46C93796':'RRCLowThreshold',
			},
			optType:'',
			tbType:'',
			EnergySavingTimeList:["00:00:00","23:59:59"],
			nlSyncStatusData:{
				syncStatus:'',
				band:'',
				frequency:'',
				cellId:'',
			},
            interfaceList:[],
            intervalList:['-7','-6','-5','-4','-3','-2','-1','0','1','2','3','4']
		};
	},
	computed: {
		nlModeShow(){
			var vm = this;
			return vm.rowDataInfo.product_name == 'Stellar227' ? true : false;
		}
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
			vm.getParamData(vm.smallCellCode,'23003');

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
					if(key == 'Sync_syncSource'){
						vm.syncSourceSelectList = value.split('_');
					}
					if(key == 'Sync_NlSyncStatus'){
						var data = value.split(';');
						vm.nlSyncStatusData.syncStatus = data[0];
						vm.nlSyncStatusData.band = data[1];
						vm.nlSyncStatusData.frequency = data[2];
						vm.nlSyncStatusData.cellId = data[3];
					}
					vm.ruleForm[key] = value;
					if(key == 'EnergySavingTime'){
						if(vm.ruleForm[key] == ''){
							vm.EnergySavingTimeList = [];
						}else{
							vm.EnergySavingTimeList = vm.ruleForm[key].split('-');
						}
					}
                    if(key == 'Sync_PTP_VLANID'){
                        if(vm.ruleForm[key] == '0'){
                            vm.ruleForm.Sync_PTP_VlanSwitch = '0'
                        }else{
                            vm.ruleForm.Sync_PTP_VlanSwitch = '1'
                        }
                    }
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
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
					})
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
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

					Sync_Mode:'GPS_PPS',
					Sync_syncSource:'',
					Sync_ForcedSync:'0',
					Sync_PPSTimeOffset:'0',
					Sync_Longitude:'',
					Sync_Latitude:'',
					Sync_Altitude:'',
                    Sync_NlLockMode:'1',
                    Sync_NlGeneration:'4',
                    Sync_NlLockBand:'',
                    Sync_NlLockFrequency:'',
                    Sync_NlLockCellId:'',

                    Sync_PTP_Profile:'1588v2',
                    Sync_PTP_Status:'',
                    Sync_PTP_Domain:'0',
                    Sync_PTP_TransferMode:'L2',
                    Sync_PTP_ServerAddress:'0.0.0.0',
                    Sync_PTP_Interface:'',
                    Sync_PTP_VlanSwitch:'0',
                    Sync_PTP_VLANID:'0',
                    Sync_PTP_UniCastMode:'0',
                    Sync_PTP_IPAddress:'',
                    Sync_PTP_SyncInterval:'-4',
                    Sync_PTP_DelayInterval:'0',

					EnergySavingType:'NOT_SAVING',
					EnergySavingTime:'',
					EnergySavingDelayTime:'',
					PrbLowThreshold:'',
					PrbReportPeriod:'',
					RRCLowThreshold:'',
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
								objs[listKey] = items[listVal]
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
                        if(field.prop != 'Sync_PTP_VlanSwitch'){
                            params[key] = editData;
                        }
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
		// Sync_syncSource 点击事件
		syncSourceItemChange(val){
			var vm = this;
				syncSourceSelectList = vm.syncSourceSelectList;
			if(val == 'GLONASS'){
				vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
					return items != 'BEIDOU' && items != 'GALILEO';
				})
			}else if(val == 'BEIDOU'){
				vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
					return items != 'GLONASS' && items != 'GALILEO';
				})
			}else if(val == 'GALILEO'){
				vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
					return items != 'GLONASS' && items != 'BEIDOU';
				})
			}else if(val == 'GPS'){
                let isExist = vm.syncSourceSelectList.some(items=>items == val);
                if(!isExist){
                    vm.syncSourceSelectList = vm.syncSourceSelectList.filter((items)=>{
                        return items != 'QZSS';
                    })
                }
            }else if(val == 'QZSS'){
                let isExist = vm.syncSourceSelectList.some(items=>items == val);
                if(isExist){
                    let isExistGps = vm.syncSourceSelectList.some(items=>items == 'GPS');
                    if(!isExistGps){
                        vm.syncSourceSelectList.push('GPS');
                    }
                }
            }
			vm.ruleForm.Sync_syncSource = vm.syncSourceSelectList.join('_');
		},
		EnergySavingTimeListChange(time){
			var vm = this;
			if(time == null || time == undefined || time == ''){
				vm.ruleForm.EnergySavingTime = ''
			}else{
				vm.ruleForm.EnergySavingTime = time.join('-');
			}
		},
		//同步
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
					gnbTabSettingVue.changeMain('bts');
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
        // 协议类型改变事件
        profileChange(val){
            var vm = this;
           
            if(val == 'G8275.2'){
                vm.ruleForm.Sync_PTP_TransferMode = 'UDPv4';
                vm.ruleForm.Sync_PTP_UniCastMode = '1';
            }
            vm.transferModeChange();
        },
        transferModeChange(){
            var vm = this;
            if(vm.ruleForm.Sync_PTP_Profile == '1588v2' && vm.ruleForm.Sync_PTP_TransferMode == 'L2'){
                vm.ruleForm.Sync_PTP_UniCastMode = '0';
            }
        },
        changePTPVlanSwitch(){
            var vm = this;
            if(vm.ruleForm.Sync_PTP_VlanSwitch == '0'){
                vm.ruleForm.Sync_PTP_VLANID = '0';
            }else{
                if(vm.ruleForm.Sync_PTP_VLANID == '0'){
                    vm.ruleForm.Sync_PTP_VLANID = '1';
                }
            }
        },
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data",this.init)
	}
});

</script>
