<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gnbRanPage{
	height: 100%;
	width: 100%;
}
#gnbRanPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gnbRanPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gnbRanPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gnbRanPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gnbRanPage .rightContentCls .contentTableTitle{
	display: flex;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gnbRanPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gnbRanPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gnbRanPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gnbRanPage .itemListBoxCls{
	padding-top: 5px;
}
#gnbRanPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gnbRanPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gnbRanPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gnbRanPage .el-form-item{
	margin-bottom: 20px;
}
#gnbRanPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gnbRanPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gnbRanPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
#gnbRanPage .el-icon-menu-help::before{
    color: #909399 !important;
    font-size: 14px;
}
#gnbRanPage .labelIconCls{
    position: relative;
}
</style>

<div id="gnbRanPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			RAN
			<!-- 按钮  同步 -->
			<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSettingsClick" tip="<%=rb.getString("TongBu")%>">
				<span class="el-icon el-icon-circle-refresh"></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
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
									<el-input v-model.trim='ruleForm.gnbLength'>
										<template slot="append"><%=rb.getString("FanWei")%>：22~32,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='gnbName' style="width:40%;min-width:400px;" label="gNB Name" label-width="140px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										gNB Name
									</span>
									<el-input v-model.trim='ruleForm.gnbName' maxlength="150">
										<template slot="append">Length：0~150</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='gnbId' style="width:40%;min-width:400px;" label="gNB ID" label-width="140px" class='validate-item'>
									<span slot="label" class="labelIconCls">
										gNB ID
									</span>
									<el-input v-model.trim='ruleForm.gnbId'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='adminState' style="width:40%;min-width:400px;" label="AdminState" label-width="140px">
									<span slot="label" class="labelIconCls">
										AdminState
									</span>
									<el-select v-model='ruleForm.adminState'>
										<el-option label='Locked' value='1'></el-option>
										<el-option label='Unlocked' value='2'></el-option>
										<el-option label='ShuttingDown' value='3'></el-option>
									</el-select>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="SSB">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">SSB</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='SSB_SSBAbsoluteFreq' style="width:40%;min-width:400px;" label="SSB Absolute Freq" label-width="140px" class='validate-item'>
									<el-input v-model.trim='ruleForm.SSB_SSBAbsoluteFreq'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="MobilityStrategy">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Mobility Strategy</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='Mobility_NrToLteMigrateStgy' style="width:40%;min-width:400px;" label="NrToLteMigrateStgy" label-width="140px">
									<el-select v-model='ruleForm.Mobility_NrToLteMigrateStgy'>
										<el-option label='PS_MEAD_RED' value='0'></el-option>
										<el-option label='PS_MEAS_HO' value='1'></el-option>
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
										:rownumber="false" 
										:row-class-name="tableRowClassName"
										id="NRCellTable" 
										:data="ruleForm.NRCellList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
										>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
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
									<el-input v-model.trim='ruleForm.Pci'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~1007,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='NRARFCNDL' style="width:40%;min-width:400px;" label="NRARFCNDL" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.NRARFCNDL' :disabled="sasEnableStatus">
										<template slot="append"><%=rb.getString("FanWei")%>：Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='NRARFCNUL' style="width:40%;min-width:400px;" label="NRARFCNUL" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.NRARFCNUL' :disabled="sasEnableStatus">
										<template slot="append"><%=rb.getString("FanWei")%>：Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='NumOfTxAtenna' style="width:40%;min-width:400px;" label="NumOfTxAtenna" label-width="160px">
									<el-select v-model='ruleForm.NumOfTxAtenna'>
										<el-option label='1' value='1'></el-option>
										<el-option label='2' value='2'></el-option>
										<el-option label="4" value="4"></el-option>
										<el-option label="8" value="8"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='NumOfRxAtenna' style="width:40%;min-width:400px;" label="NumOfRxAtenna" label-width="160px">
									<el-select v-model='ruleForm.NumOfRxAtenna'>
										<el-option label='1' value='1'></el-option>
										<el-option label='2' value='2'></el-option>
										<el-option label="4" value="4"></el-option>
										<el-option label="8" value="8"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='DL_SubCarrierSpacing' style="width:40%;min-width:400px;" label="DL SubCarrierSpacing" label-width="160px">
									<el-select v-model='ruleForm.DL_SubCarrierSpacing' @change="DL_SCSChange">
										<el-option label='0(15kHz)' value='0'></el-option>
										<el-option label='1(30kHz)' value='1'></el-option>
										<el-option label="2(60kHz)" value="2"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='DL_CarrierBandWidth' style="width:40%;min-width:400px;" label="DL CarrierBandWidth" label-width="160px">
									<span slot="label" class="labelIconCls">
										DL CarrierBandWidth
									</span>
									<el-select v-model='ruleForm.DL_CarrierBandWidth' :disabled="sasEnableStatus">
										<el-option v-for="item in DL_CarrierBandWidthData" :label='item.label' :value='item.value'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='UL_SubCarrierSpacing' style="width:40%;min-width:400px;" label="UL SubCarrierSpacing" label-width="160px">
									<el-select v-model='ruleForm.UL_SubCarrierSpacing' @change="UL_SCSChange">
										<el-option label='0(15kHz)' value='0'></el-option>
										<el-option label='1(30kHz)' value='1'></el-option>
										<el-option label="2(60kHz)" value="2"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='UL_CarrierBandWidth' style="width:40%;min-width:400px;" label="UL CarrierBandWidth" label-width="160px">
									<span slot="label" class="labelIconCls">
										UL CarrierBandWidth
									</span>
									<el-select v-model='ruleForm.UL_CarrierBandWidth' :disabled="sasEnableStatus">
										<el-option v-for="item in UL_CarrierBandWidthData" :label='item.label' :value='item.value'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='PowerModify' style="width:40%;min-width:400px;" label="Power Modify" label-width="160px" class='validate-item' >
									<span slot="label" class="labelIconCls">
										Power Modify(dBm)
									</span>
									<el-input v-model.trim='ruleForm.PowerModify' :disabled="sasEnableStatus">
										<template slot="append"><%=rb.getString("FanWei")%>：{{PowerMinLimitValue}}~{{PowerMaxLimitValue}}</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='rftxEnable' style="width:40%;min-width:400px;" label="RF Enable" label-width="160px">
									<el-select v-model='ruleForm.rftxEnable' :disabled="sasEnableStatus">
										<el-option label='ON' value='1'></el-option>
										<el-option label='OFF' value='0'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='offsetToPointA' style="width:40%;min-width:400px;" label="Offset To Point A" label-width="160px" class='validate-item' >
									<el-input v-model.trim='ruleForm.offsetToPointA'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~2199,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='ssbSubCarrierOffset' style="width:40%;min-width:400px;" label="SSB Sub Carrier Offset" label-width="160px" class='validate-item' >
									<el-input v-model.trim='ruleForm.ssbSubCarrierOffset'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
									</el-input>
								</el-form-item>
								<!--<el-form-item prop='ssbGSCN' style="width:40%;min-width:400px;" label="SSB GSCN" label-width="160px" class='validate-item' >
									<el-input v-model.trim='ruleForm.ssbGSCN'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
									</el-input>
								</el-form-item>-->
							</div>
						</div>
					</el-collapse-item>
                    <el-collapse-item name="Neighbor">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Neighbor Freq/Cell</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--LTE N-FREQ List-->
							<div> 
								<div class="contentTableTitle">
									<div>LTE N-FREQ List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','LTENF')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="LTENFListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="LTENFListTable" 
										:data="ruleForm.LTENFList" 
										height="200px"
										:pagination="true"
										:front-pagination="true"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','LTENF')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'LTENF',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="LTENF_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='CarrierFreq' min-width="120" prop="LTENF_CarrierFreq" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='QOffset' min-width="120" prop="LTENF_QOffset" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='CellReselectionPriority' min-width="120" prop="LTENF_CellReselectionPriority" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='QRxLevMin' min-width="120" prop="LTENF_QRxLevMin" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='QQualMin' min-width="120" prop="LTENF_QQualMin" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='LTENFList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.LTENFList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--LTE EutraFREQ Reselection List-->
							<div> 
								<div class="contentTableTitle">
									<div>EutraFREQ Reselection List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','LTEER')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="LTEERListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="LTEERListTable" 
										:data="ruleForm.LTEERList" 
										height="200px"
										:pagination="true"
										:front-pagination="true"
										style="border:1px solid #E9E9E9;"
									>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
											<template slot-scope="scope">
												<span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','LTEER')" style="margin-right:15px;"></span>
												<span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'LTEER',event)"></span>
											</template>
										</el-table-column>
                                        <el-table-column label='Index' min-width="50" prop="LTEER_idx" show-overflow-tooltip></el-table-column>
										<el-table-column label='EUTRACarrierARFCN' min-width="120" prop="LTEER_EUTRACarrierARFCN" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='CellReselectionPriority' min-width="120" prop="LTEER_CellReselectionPriority" show-overflow-tooltip></el-table-column>
										<el-table-column label='ThreshXHigh' min-width="120" prop="LTEER_ThreshXHigh" show-overflow-tooltip></el-table-column>
										<el-table-column label='ThreshXLow' min-width="120" prop="LTEER_ThreshXLow" show-overflow-tooltip></el-table-column>
										<el-table-column label='QRxLevMin' min-width="120" prop="LTEER_QRxLevMin" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='LTEERList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.LTEERList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--LTE N-CELL List-->
							<div> 
								<div class="contentTableTitle">
									<div>LTE N-CELL List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','LTENC')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="LTENCListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="LTENCListTable" 
										:data="ruleForm.LTENCList" 
										height="200px"
										:pagination="true"
										:front-pagination="true"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','LTENC')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'LTENC',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="LTENC_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='PLMNID' min-width="120" prop="LTENC_PLMNID" show-overflow-tooltip></el-table-column>
										<el-table-column label='CID' min-width="120" prop="LTENC_CID" show-overflow-tooltip></el-table-column>
										<el-table-column label='EUTRACarrierARFCN' min-width="120" prop="LTENC_EUTRACarrierARFCN" show-overflow-tooltip></el-table-column>
										<el-table-column label='PhyCellID' min-width="120" prop="LTENC_PhyCellID" show-overflow-tooltip></el-table-column>
										<el-table-column label='QOffset' min-width="120" prop="LTENC_QOffset" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='LTENCList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.LTENCList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--NR N-FREQ List-->
							<div> 
								<div class="contentTableTitle">
									<div>NR N-FREQ List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','NRNF')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="NRNFListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="NRNFListTable" 
										:data="ruleForm.NRNFList" 
										height="200px"
										:pagination="true"
										:front-pagination="true"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','NRNF')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'NRNF',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="NRNF_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='Enable' min-width="120" prop="NRNF_Enable" show-overflow-tooltip></el-table-column>
										<el-table-column label='SSBFrequency' min-width="120" prop="NRNF_SSBFrequency" show-overflow-tooltip></el-table-column>
										<el-table-column label='SmtcPeriodicity' min-width="120" prop="NRNF_SmtcPeriodicity" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-html="smtcPeriodicityFmt(scope.row, scope.row.NRNF_SmtcPeriodicity, scope.$index)"></div>
											</template>
										</el-table-column>
										<el-table-column label='RsrpOffsetSSB' min-width="120" prop="NRNF_RsrpOffsetSSB" show-overflow-tooltip></el-table-column>
										<el-table-column label='Bitmap' min-width="120" prop="NRNF_Bitmap" show-overflow-tooltip></el-table-column>
										<el-table-column label='FreqBandIndicatorNR' min-width="160" prop="NRNF_FreqBandIndicatorNR" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='NRNFList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.NRNFList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--NR InterFREQ Reselection Setting List-->
							<div> 
								<div class="contentTableTitle">
									<div>InterFREQ Reselection Setting</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','NRIRS')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="NRIRSListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="NRIRSListTable" 
										:data="ruleForm.NRIRSList" 
										height="200px"
										:pagination="true"
										:front-pagination="true"
										style="border:1px solid #E9E9E9;"
									>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
											<template slot-scope="scope">
												<span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','NRIRS')" style="margin-right:15px;"></span>
												<span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'NRIRS',event)"></span>
											</template>
										</el-table-column>
                                        <el-table-column label='Index' min-width="50" prop="NRIRS_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='CarrierFreq' min-width="120" prop="NRIRS_CarrierFreq" show-overflow-tooltip></el-table-column>
										<el-table-column label='Subcarrier Spacing' min-width="120" prop="NRIRS_SubCarrierSpacing" show-overflow-tooltip></el-table-column>
										<el-table-column label='QRxLevMin' min-width="120" prop="NRIRS_QRxLevMin" show-overflow-tooltip></el-table-column>
										<el-table-column label='ThreshXHighP' min-width="120" prop="NRIRS_ThreshXHighP" show-overflow-tooltip></el-table-column>
										<el-table-column label='CellReselectionPriority' min-width="160" prop="NRIRS_CellReselectionPriority" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='NRIRSList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.NRIRSList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--NR N-CELL List-->
							<div> 
								<div class="contentTableTitle">
									<div>NR N-CELL List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','NRNC')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="NRNCListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="NRNCListTable" 
										:data="ruleForm.NRNCList" 
										height="200px"
										:pagination="true"
										:front-pagination="true"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','NRNC')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'NRNC',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="NRNC_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='PLMNID' min-width="120" prop="NRNC_PLMNID" show-overflow-tooltip></el-table-column>
										<el-table-column label='CID' min-width="120" prop="NRNC_CID" show-overflow-tooltip></el-table-column>
										<el-table-column label='NRCarrierARFCN' min-width="120" prop="NRNC_NRCarrierARFCN" show-overflow-tooltip></el-table-column>
										<el-table-column label='PhyCellID' min-width="120" prop="NRNC_PhyCellID" show-overflow-tooltip></el-table-column>
										<el-table-column label='QOffset' min-width="120" prop="NRNC_QOffset" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='NRNCList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.NRNCList'></el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="QOS">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">QOS</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--QOS List-->
							<div> 
								<div class="contentTableTitle">
									<div>QOS List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','QOS')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="QOSListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="QOSListTable" 
										:data="ruleForm.QOSList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','QOS')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'QOS',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="QOS_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='QOS' min-width="80" prop="QOS_Enable" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='5QI' min-width="80" prop="QOS_5QI" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='Mapping DrbIndex' min-width="140" prop="QOS_MappingDrbIndex" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='RlcMode' min-width="120" prop="QOS_RlcMode" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='Enable Rohc' min-width="120" prop="QOS_EnableRohc" show-overflow-tooltip></el-table-column>
										<el-table-column label='LongDrxCycle' min-width="120" prop="QOS_LongDrxCycle" show-overflow-tooltip></el-table-column>
										<el-table-column label='ShortDrxCycle' min-width="120" prop="QOS_ShortDrxCycle" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='QOSList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.QOSList'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--SST List-->
							<div> 
								<div class="contentTableTitle">
									<div>SST List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','SST')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="SSTListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="SSTListTable" 
										:data="ruleForm.SSTList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','SST')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'SST',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="SST_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='Sst' min-width="80" prop="SST_Sst" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='SstResourceType' min-width="80" prop="SST_SstResourceType" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='MaxResourceReserved' min-width="140" prop="SST_MaxResourceReserved" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='MinResourceReserved' min-width="120" prop="SST_MinResourceReserved" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='SSTList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.SSTList'></el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="Mobility">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Mobility</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--A1-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">A1 List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','A1')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="A1ListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="A1ListTable" 
										:data="ruleForm.A1List" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','A1')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'A1',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="A1_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='A1 Threshold RSRP' min-width="120" prop="A1_ThresholdRSRP" show-overflow-tooltip></el-table-column>
										<el-table-column label='Hysteresis' min-width="120" prop="A1_Hysteresis" show-overflow-tooltip></el-table-column>
										<el-table-column label='Time To Trigger' min-width="120" prop="A1_TimeToTrigger" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='A1List' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.A1List'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--A2-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">A2 List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','A2')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="A2ListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="A2ListTable" 
										:data="ruleForm.A2List" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','A2')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'A2',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="A2_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='A2 Threshold RSRP' min-width="120" prop="A2_ThresholdRSRP" show-overflow-tooltip></el-table-column>
										<el-table-column label='Hysteresis' min-width="120" prop="A2_Hysteresis" show-overflow-tooltip></el-table-column>
										<el-table-column label='Time To Trigger' min-width="120" prop="A2_TimeToTrigger" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='A2List' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.A2List'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--A3-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">A3 List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','A3')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="A3ListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="A3ListTable" 
										:data="ruleForm.A3List" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','A3')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'A3',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="A3_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='A3 Offset RSRP' min-width="120" prop="A3_OffsetRSRP" show-overflow-tooltip></el-table-column>
										<el-table-column label='Hysteresis' min-width="120" prop="A3_Hysteresis" show-overflow-tooltip></el-table-column>
										<el-table-column label='Time To Trigger' min-width="120" prop="A3_TimeToTrigger" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='A3List' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.A3List'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--A4-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">A4 List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','A4')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="A4ListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="A4ListTable" 
										:data="ruleForm.A4List" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','A4')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'A4',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="A4_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='A4 Threshold RSRP' min-width="120" prop="A4_ThresholdRSRP" show-overflow-tooltip></el-table-column>
										<el-table-column label='Hysteresis' min-width="120" prop="A4_Hysteresis" show-overflow-tooltip></el-table-column>
										<el-table-column label='Time To Trigger' min-width="120" prop="A4_TimeToTrigger" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='A4List' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.A4List'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--A5-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">A5 List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','A5')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="A5ListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="A5ListTable" 
										:data="ruleForm.A5List" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','A5')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'A5',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="A5_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='A5 Threshold RSRP' min-width="120" prop="A5_Threshold1RSRP" show-overflow-tooltip></el-table-column>
										<el-table-column label='Hysteresis' min-width="120" prop="A5_Hysteresis" show-overflow-tooltip></el-table-column>
										<el-table-column label='Time To Trigger' min-width="120" prop="A5_TimeToTrigger" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='A5List' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.A5List'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--B1-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">B1 List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','B1')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="B1ListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="B1ListTable" 
										:data="ruleForm.B1List" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','B1')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'B1',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="B1_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='B1 Threshold1 EUTRA RSRP' min-width="120" prop="B1_Threshold1EUTRARSRP" show-overflow-tooltip></el-table-column>
										<el-table-column label='Hysteresis' min-width="120" prop="B1_Hysteresis" show-overflow-tooltip></el-table-column>
										<el-table-column label='Time To Trigger' min-width="120" prop="B1_TimeToTrigger" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='B1List' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.B1List'></el-input>
									</el-form-item>
								</div>
							</div>
							<!--B2-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">B2 List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','B2')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="B2ListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="B2ListTable" 
										:data="ruleForm.B2List" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','B2')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'B2',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="B2_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='B2 Threshold1 RSRP' min-width="120" prop="B2_Threshold1RSRP" show-overflow-tooltip></el-table-column>
										<el-table-column label='Hysteresis' min-width="120" prop="B2_Hysteresis" show-overflow-tooltip></el-table-column>
										<el-table-column label='Time To Trigger' min-width="120" prop="B2_TimeToTrigger" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='B2List' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.B2List'></el-input>
									</el-form-item>
								</div>
							</div>
                            <!--Period Measure List-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">Period Measure List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','PeriodMeasure')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="PeriodMeasureListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="PeriodMeasureListTable" 
										:data="ruleForm.PeriodMeasureList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','PeriodMeasure')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'PeriodMeasure',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="PeriodMeasure_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='Report Quantity' min-width="120" prop="PeriodMeasure_ReportQuantity" show-overflow-tooltip></el-table-column>
										<el-table-column label='Max Report Cells' min-width="120" prop="PeriodMeasure_MaxReportCells" show-overflow-tooltip></el-table-column>
										<el-table-column label='Measure Purpose' min-width="120" prop="PeriodMeasure_MeasurePurpose" show-overflow-tooltip>
                                            <template slot-scope="scope">
                                                <span v-if="scope.row.PeriodMeasure_MeasurePurpose == '1'">MR</span>
                                                <span v-if="scope.row.PeriodMeasure_MeasurePurpose == '2'">ANR</span>
                                            </template>
                                        </el-table-column>
                                        <el-table-column label='Report Interval' min-width="120" prop="PeriodMeasure_ReportInterval" show-overflow-tooltip></el-table-column>
										<el-table-column label='Report Amount' min-width="120" prop="PeriodMeasure_ReportAmount" show-overflow-tooltip></el-table-column>
                                    </el-ctable>
									<el-form-item prop='PeriodMeasureList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.PeriodMeasureList'></el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
                    <el-collapse-item name="SIB">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">SIB</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--SIB1-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">SIB1</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
                                    <el-form-item prop='SIB1_QRxLevMinSIB1' style="width:40%;min-width:400px;" label="QRxLevMinSIB1" label-width="140px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.SIB1_QRxLevMinSIB1'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：(-70)~(-22),Integer</template>
                                        </el-input>
                                    </el-form-item>
                                    <el-form-item prop='SIB1_QQualMinOffset' style="width:40%;min-width:400px;" label="QQualMinOffset" label-width="160px">
                                        <el-select v-model='ruleForm.SIB1_QQualMinOffset' filterable>
                                            <el-option v-for="item in generateArray(1,8)" :label='item' :value='item'></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item prop='SIB1_QRxLevMinOffset' style="width:40%;min-width:400px;" label="QRxLevMinOffset" label-width="160px">
                                        <el-select v-model='ruleForm.SIB1_QRxLevMinOffset' filterable>
                                            <el-option v-for="item in generateArray(1,8)" :label='item' :value='item'></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item prop='SIB1_QQualMinSIB1' style="width:40%;min-width:400px;" label="QQualMinSIB1" label-width="140px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.SIB1_QQualMinSIB1'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：(-43)~(-12),Integer</template>
                                        </el-input>
                                    </el-form-item>
                                </div>
							</div>
                            <!--SIB2-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">SIB2</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
                                    <el-form-item prop='SIB2_Enable' style="width:40%;min-width:400px;" label="SIB2" label-width="160px">
                                        <el-select v-model='ruleForm.SIB2_Enable'>
                                            <el-option label='ON' value='1'></el-option>
										    <el-option label='OFF' value='0'></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_Qhyst' style="width:40%;min-width:400px;" label="Qhyst" label-width="140px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.SIB2_Qhyst'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
                                        </el-input>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_QRxLevMinSIB2' style="width:40%;min-width:400px;" label="QRxLevMinSIB2" label-width="140px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.SIB2_QRxLevMinSIB2'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：(-70)~(-22),Integer</template>
                                        </el-input>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_SIntraSearchP' style="width:40%;min-width:400px;" label="SIntraSearchP" label-width="140px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.SIB2_SIntraSearchP'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
                                        </el-input>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_TReselectionNR' style="width:40%;min-width:400px;" label="TReselectionNR" label-width="160px">
                                        <el-select v-model='ruleForm.SIB2_TReselectionNR' filterable>
                                            <el-option v-for="item in generateArray(0,7)" :label='item' :value='item'></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_CellReselectionPriority' style="width:40%;min-width:400px;" label="CellReselectionPriority" label-width="160px">
                                        <el-select v-model='ruleForm.SIB2_CellReselectionPriority' filterable>
                                            <el-option v-for="item in generateArray(0,7)" :label='item' :value='item'></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_ThreshServingLowP' style="width:40%;min-width:400px;" label="ThreshServingLowP" label-width="140px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.SIB2_ThreshServingLowP'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
                                        </el-input>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_DeriveSSBIndexFromCell' style="width:40%;min-width:400px;" label="DeriveSSBIndexFromCell" label-width="160px">
                                        <el-select v-model='ruleForm.SIB2_DeriveSSBIndexFromCell'>
                                            <el-option v-for="item in generateArray(0,1)" :label='item' :value='item'></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_SNonIntraSearchP' style="width:40%;min-width:400px;" label="SNonIntraSearchP" label-width="160px">
                                        <el-select v-model='ruleForm.SIB2_SNonIntraSearchP' filterable>
                                            <el-option v-for="item in generateArray(0,31)" :label='item' :value='item'></el-option>
                                            <el-option label="invalid value" value="268435455"></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item prop='SIB2_SNonIntraSearchQ' style="width:40%;min-width:400px;" label="SNonIntraSearchQ" label-width="160px">
                                        <el-select v-model='ruleForm.SIB2_SNonIntraSearchQ' filterable>
                                            <el-option v-for="item in generateArray(0,31)" :label='item' :value='item'></el-option>
                                            <el-option label="invalid value" value="268435455"></el-option>
                                        </el-select>
                                    </el-form-item>
                                </div>
							</div>
                            <!--SIB3-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">SIB3</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
                                    <el-form-item prop='SIB3_Enable' style="width:40%;min-width:400px;" label="SIB3" label-width="160px">
                                        <el-select v-model='ruleForm.SIB3_Enable'>
                                            <el-option label='ON' value='1'></el-option>
										    <el-option label='OFF' value='0'></el-option>
                                        </el-select>
                                    </el-form-item>
                                </div>
							</div>
                            <!--SIB4-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">SIB4</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
                                    <el-form-item prop='SIB4_Enable' style="width:40%;min-width:400px;" label="SIB4" label-width="160px">
                                        <el-select v-model='ruleForm.SIB4_Enable'>
                                            <el-option label='ON' value='1'></el-option>
										    <el-option label='OFF' value='0'></el-option>
                                        </el-select>
                                    </el-form-item>
                                </div>
							</div>
                            <!--SIB5-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">SIB5</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
                                    <el-form-item prop='SIB5_Enable' style="width:40%;min-width:400px;" label="SIB5" label-width="160px">
                                        <el-select v-model='ruleForm.SIB5_Enable'>
                                            <el-option label='ON' value='1'></el-option>
										    <el-option label='OFF' value='0'></el-option>
                                        </el-select>
                                    </el-form-item>
                                </div>
							</div>
                            <!--SIB9-->
							<div> 
								<div class="contentTableTitle">
									<div style="margin-bottom:10px;">SIB9</div>
								</div>
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
                                    <el-form-item prop='SIB9_Enable' style="width:40%;min-width:400px;" label="SIB9" label-width="160px">
                                        <el-select v-model='ruleForm.SIB9_Enable'>
                                            <el-option label='ON' value='1'></el-option>
										    <el-option label='OFF' value='0'></el-option>
                                        </el-select>
                                    </el-form-item>
                                    <el-form-item prop='SIB9_DaylightSavingTime' style="width:40%;min-width:400px;" label="Daylight Saving Time" label-width="140px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.SIB9_DaylightSavingTime'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：0~2,Integer</template>
                                        </el-input>
                                    </el-form-item>
                                    <el-form-item prop='SIB9_LeapSecond' style="width:40%;min-width:400px;" label="Leap Second" label-width="140px" class='validate-item'>
                                        <el-input v-model.trim='ruleForm.SIB9_LeapSecond'>
                                            <template slot="append"><%=rb.getString("FanWei")%>：-127~128,Integer</template>
                                        </el-input>
                                    </el-form-item>
                                </div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="Xn">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Xn</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--Xn List-->
							<div> 
								<div class="contentTableTitle">
									<div>Xn List</div>
									<div><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','Xn')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="XnListTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="XnListTable" 
										:data="ruleForm.XnList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
                                        <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
                                            <template slot-scope="scope">
                                                <span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','Xn')" style="margin-right:15px;"></span>
                                                <span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'Xn',event)"></span>
                                            </template>
                                        </el-table-column>
										<el-table-column label='ID' min-width="40" prop="Xn_idx" show-overflow-tooltip></el-table-column>
                                        <el-table-column label='PLMN ID' min-width="120" prop="Xn_PLMNID" show-overflow-tooltip></el-table-column>
										<el-table-column label='RemoteAddress' min-width="120" prop="Xn_RemoteAddress" show-overflow-tooltip></el-table-column>
										<el-table-column label='XnLinkEnable' min-width="120" prop="Xn_LinkEnable" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-show="scope.row.Xn_LinkEnable == '0'">OFF</div>
												<div v-show="scope.row.Xn_LinkEnable == '1'">ON</div>
											</template>
										</el-table-column>
										<el-table-column label='XnHoEnable' min-width="120" prop="Xn_HoEnable" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-show="scope.row.Xn_HoEnable == '0'">OFF</div>
												<div v-show="scope.row.Xn_HoEnable == '1'">ON</div>
											</template>
										</el-table-column>
										<el-table-column label='Status' min-width="120" prop="Xn_Status" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-show="scope.row.Xn_Status == '0'"><%=rb.getString("MMEWeiLianJie")%></div>
												<div v-show="scope.row.Xn_Status == '1'"><%=rb.getString("MMEYiLianJie")%></div>
											</template>
										</el-table-column>
										<el-table-column v-if="false" prop="operateType" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='XnList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.XnList'></el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="Black">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">Xn BlackList</span>
							</p>
						</template>
						<div class="rightContentCls" style="padding-bottom:20px;">
							<div class="leftAndRightItemCls">
								<el-form-item prop="XnBlacklist" label="RemoteAddress"  label-width="160px" style="margin-bottom:0px;">
									<span slot="label" class="labelIconCls">
										RemoteAddress
										<span style="color: #999999;margin-left: 5px;">No more than 8</span>
									</span>
									<el-input v-model="remoteAddress" >
										<i slot="suffix" class="el-icon el-icon-plus" @click="addRemoteAddress" v-show="addRemoteAddressShow"></i>
										<div slot="suffix" class="disabledIconBox">
											<i  class="el-icon el-icon-plus" v-show="!addRemoteAddressShow"></i>
										</div>
									</el-input>
									<p class="errorBoxCls">{{remoteAddressErrorMessage}}</p>
								</el-form-item>
								<div class="itemListBoxCls">
									<div v-for="(item,index) in ruleForm.XnBlacklist" class="itemCls" v-show="!item.operateType || (item.operateType && item.operateType == 'add')">
										<span>{{item.remoteAddress}}</span>
										<span class="el-icon el-icon-close" style="margin-left:5px;" @click="remoteAddressListDel(item,index)"></span>
									</div>
								</div>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="ANR">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">ANR</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<div style="display:flex;margin-left:16px;flex-wrap: wrap">
								<el-form-item prop='ANR_Enable' style="width:40%;min-width:400px;" label="Enable" label-width="160px">
									<el-switch v-model="ruleForm.ANR_Enable" active-value="1" inactive-value="0"></el-switch>
								</el-form-item>
								<el-form-item prop='ANR_InterFeqEnable' style="width:40%;min-width:400px;" label="InterFeqEnable" label-width="160px">
									<el-switch v-model="ruleForm.ANR_InterFeqEnable" active-value="1" inactive-value="0"></el-switch>
								</el-form-item>
								<el-form-item prop='ANR_EUTRANEnable' style="width:40%;min-width:400px;" label="EUTRANEnable" label-width="160px">
									<el-switch v-model="ruleForm.ANR_EUTRANEnable" active-value="1" inactive-value="0"></el-switch>
								</el-form-item>
								<el-form-item prop='ANR_BiNRCellEnable' style="width:40%;min-width:400px;" label="BiNRCellEnable" label-width="160px">
									<el-switch v-model="ruleForm.ANR_BiNRCellEnable" active-value="1" inactive-value="0"></el-switch>
								</el-form-item>
								<el-form-item prop='ANR_MRTriggerType' style="width:40%;min-width:400px;" label="MRTriggerType" label-width="160px">
									<el-select v-model='ruleForm.ANR_MRTriggerType'>
										<el-option label='Event' value='0'></el-option>
										<el-option label='Period' value='1'></el-option>
									</el-select>
								</el-form-item>
								<el-form-item prop='ANR_AbsoluteThreshold' style="width:40%;min-width:400px;" label="AbsoluteThreshold" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.ANR_AbsoluteThreshold'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='ANR_RelativeThreshold' style="width:40%;min-width:400px;" label="RelativeThreshold" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.ANR_RelativeThreshold'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='ANR_AbsEnable' style="width:40%;min-width:400px;" label="AbsEnable" label-width="160px">
									<el-switch v-model="ruleForm.ANR_AbsEnable" active-value="1" inactive-value="0"></el-switch>
								</el-form-item>
								<el-form-item prop='ANR_KpiPeriod' style="width:40%;min-width:400px;" label="KPIPeriod" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.ANR_KpiPeriod'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='ANR_AutoAdjustEnable' style="width:40%;min-width:400px;" label="AutoAdjustEnable" label-width="160px">
									<el-switch v-model="ruleForm.ANR_AutoAdjustEnable" active-value="1" inactive-value="0"></el-switch>
								</el-form-item>
								<el-form-item prop='ANR_AutoRemoveEnable' style="width:40%;min-width:400px;" label="AutoRemoveEnable" label-width="160px">
									<el-switch v-model="ruleForm.ANR_AutoRemoveEnable" active-value="1" inactive-value="0"></el-switch>
								</el-form-item>
								<el-form-item prop='ANR_AutoRemovePeriod' style="width:40%;min-width:400px;" label="AutoRemovePeriod" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.ANR_AutoRemovePeriod'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='ANR_AutoRemoveMaxCell' style="width:40%;min-width:400px;" label="AutoRemoveMaxCell" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.ANR_AutoRemoveMaxCell'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='ANR_MaxHOtimes' style="width:40%;min-width:400px;" label="MaxHOtimes" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.ANR_MaxHOtimes'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
									</el-input>
								</el-form-item>
								<el-form-item prop='ANR_MaxHOSuccess' style="width:40%;min-width:400px;" label="MaxHOSuccess" label-width="160px" class='validate-item'>
									<el-input v-model.trim='ruleForm.ANR_MaxHOSuccess'>
										<template slot="append"><%=rb.getString("FanWei")%>：0~100,Integer</template>
									</el-input>
								</el-form-item>
							</div>
						</div>
					</el-collapse-item>
					<el-collapse-item name="BWP" style="border-bottom: none;">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">DL BWP Card</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--DL BWP-->
							<div> 
								<div class="contentTableTitle">
									<div>DL BWP<span style="font-size:12px;color:#999999;margin-left:10px;">(No more than 5)</span></div>
									<div v-if="bwpListAddShow"><span class="el-icon el-icon-circle-add" @click="ranTableAddDialogOpen('','add','DLBWP')"></span></div>
								</div>
								<div class="cellTableBoxCls" style="padding-bottom:20px;">
									<el-ctable
										ref="DLBWPTable" 
										:row-class-name="tableRowClassName"
										:rownumber="false" 
										id="DLBWPTable" 
										:data="ruleForm.DLBWPList" 
										height="200px"
										:pagination="false"
										style="border:1px solid #E9E9E9;"
									>
										<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
											<template slot-scope="scope">
												<div style="display: flex;align-items: center;">
													<span v-if="scope.row.operateType !== 'add'" class="el-icon el-icon-operation-details" @click="DLBWPTableDetails(scope.row)"></span>
													<span v-if="scope.row.operateType == 'add'" class="el-icon el-icon-operation-details disabled"></span>
													<span class="el-icon el-icon-operation-edit" @click="ranTableAddDialogOpen(scope.row,'edit','DLBWP')" style="margin: 0px 5px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="ranTableDelList(scope.row,'DLBWP',event)" ></span>
												</div>
											</template>
										</el-table-column>
										<el-table-column label='ID' min-width="40" prop="DLBWP_idx" show-overflow-tooltip></el-table-column>
										<el-table-column label='DIBwp ID' min-width="120" prop="DLBWP_DIBwpID" show-overflow-tooltip></el-table-column>
										<el-table-column label='StartPrbPosition' min-width="120" prop="DLBWP_StartPrbPosition" show-overflow-tooltip></el-table-column>
										<el-table-column label='BandWidth' min-width="120" prop="DLBWP_BandWidth" show-overflow-tooltip></el-table-column>
										<el-table-column label='InitDlMcs' min-width="120" prop="DLBWP_InitDlMcs" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='DLBWPList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.DLBWPList'></el-input>
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
		<!-- slide -->
		<el-slide  ref="sharingSlide" :url='sharingSlideUrl' :title="sharingSlideTitle" :footer="sharingSlideFooter" :header="sharingSlideHeader" :position="sharingSlidePosition"
			:height="sharingSlideHeight" :modal='modal' :width='sharingSlideWidth'  @cancel="sharingSlideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
	</div>
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
									<el-collapse v-model="SliceListCollapseData">
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
    <!-- LTE NF 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addLTENFDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addLTENFDialogForm" :model='addLTENFDialogForm' :rules='addLTENFDialogRules' label-position="top">     		     			            
			<el-form-item prop='LTENF_CarrierFreq' style="width:40%;min-width:400px;" label="CarrierFreq" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTENFDialogForm.LTENF_CarrierFreq' maxlength="11">
                    <template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTENF_AllowdMeasBandWidth' style="width:40%;min-width:400px;" label="AllowMeasBandWidth" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTENFDialogForm.LTENF_AllowdMeasBandWidth'>
                    <el-option label='mbw6' value='mbw6'></el-option>
                    <el-option label='mbw15' value='mbw15'></el-option>
                    <el-option label="mbw25" value="mbw25"></el-option>
                    <el-option label="mbw50" value="mbw50"></el-option>
                    <el-option label="mbw75" value="mbw75"></el-option>
                    <el-option label="mbw100" value="mbw100"></el-option>
                </el-select>
            </el-form-item>
            <el-form-item prop='LTENF_PresAntennaPort1' style="width:40%;min-width:400px;" label="PresAntennaPort1" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTENFDialogForm.LTENF_PresAntennaPort1'>
                    <el-option label='0' value='0'></el-option>
                    <el-option label='1' value='1'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item prop='LTENF_QOffset' style="width:40%;min-width:400px;" label="QOffset" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTENFDialogForm.LTENF_QOffset'>
                    <el-option v-for="item in QOffsetDataList" :label='item' :value='item'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item prop='LTENF_WideBandRsrqMeas' style="width:40%;min-width:400px;" label="WideBandRsrqMeas" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTENFDialogForm.LTENF_WideBandRsrqMeas'>
                    <el-option label='0' value='0'></el-option>
                    <el-option label='1' value='1'></el-option>
                </el-select>
            </el-form-item>
            <el-form-item prop='LTENF_CellReselectionPriority' style="width:40%;min-width:400px;" label="CellReselectionPriority" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTENFDialogForm.LTENF_CellReselectionPriority'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~7,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTENF_ThreshXHigh' style="width:40%;min-width:400px;" label="ThreshXHigh" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTENFDialogForm.LTENF_ThreshXHigh'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTENF_ThreshXLow' style="width:40%;min-width:400px;" label="ThreshXLow" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTENFDialogForm.LTENF_ThreshXLow'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTENF_QRxLevMin' style="width:40%;min-width:400px;" label="QRxLevMin" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTENFDialogForm.LTENF_QRxLevMin'>
                    <template slot="append"><%=rb.getString("FanWei")%>：-70~-22,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTENF_QQualMin' style="width:40%;min-width:400px;" label="QQualMin" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTENFDialogForm.LTENF_QQualMin'>
                    <template slot="append"><%=rb.getString("FanWei")%>：-34~-3,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTENF_PMaxEUTRA' style="width:40%;min-width:400px;" label="PMaxEUTRA" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTENFDialogForm.LTENF_PMaxEUTRA'>
                    <template slot="append"><%=rb.getString("FanWei")%>：-30~33,Integer</template>
                </el-input>
            </el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addLTENFDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- LTE ER 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="60%" :visible.sync="addLTEERDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addLTEERDialogForm" :model='addLTEERDialogForm' :rules='addLTEERDialogRules' label-position="top">     		     			            
			<el-form-item prop='LTEER_EUTRACarrierARFCN' style="width:40%;min-width:400px;" label="EUTRACarrierARFCN" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addLTEERDialogForm.LTEER_EUTRACarrierARFCN'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='LTEER_TReselectionEUTRA' style="width:48%;min-width:530px;" label="TReselectionEUTRA" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTEERDialogForm.LTEER_TReselectionEUTRA'>
                    <el-option label='0' value='0'></el-option>
                    <el-option label='1' value='1'></el-option>
                    <el-option label="2" value="2"></el-option>
                    <el-option label="3" value="3"></el-option>
                    <el-option label="4" value="4"></el-option>
                    <el-option label="5" value="5"></el-option>
					<el-option label="6" value="6"></el-option>
                    <el-option label="7" value="7"></el-option>
                </el-select>
            </el-form-item>
			<el-form-item prop='LTEER_CellReselectionPriority' style="width:40%;min-width:400px;" label="CellReselectionPriority" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTEERDialogForm.LTEER_CellReselectionPriority'>
                    <el-option label='0' value='0'></el-option>
                    <el-option label='1' value='1'></el-option>
                    <el-option label="2" value="2"></el-option>
                    <el-option label="3" value="3"></el-option>
                    <el-option label="4" value="4"></el-option>
                    <el-option label="5" value="5"></el-option>
					<el-option label="6" value="6"></el-option>
                    <el-option label="7" value="7"></el-option>
                </el-select>
            </el-form-item>
			<el-form-item prop='LTEER_ThreshXHigh' style="width:48%;min-width:530px;" label="ThreshXHigh" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTEERDialogForm.LTEER_ThreshXHigh'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTEER_ThreshXLow' style="width:40%;min-width:400px;" label="ThreshXLow" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTEERDialogForm.LTEER_ThreshXLow'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTEER_QRxLevMin' style="width:48%;min-width:530px;" label="QRxLevMin" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTEERDialogForm.LTEER_QRxLevMin'>
                    <template slot="append"><%=rb.getString("FanWei")%>：-70~-22,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTEER_QQualMin' style="width:40%;min-width:400px;" label="QQualMin" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTEERDialogForm.LTEER_QQualMin'>
                    <template slot="append"><%=rb.getString("FanWei")%>：-34~-3,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTEER_PMaxEUTRA' style="width:48%;min-width:530px;" label="PMaxEUTRA" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTEERDialogForm.LTEER_PMaxEUTRA'>
                    <template slot="append"><%=rb.getString("FanWei")%>：-30~33,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='LTEER_ThreshXHighQ' style="width:40%;min-width:400px;" label="ThreshXHighQ" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTEERDialogForm.LTEER_ThreshXHighQ'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='LTEER_AllowdMeasBandWidth' style="width:48%;min-width:530px;" label="AllowMeasBandWidth" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTEERDialogForm.LTEER_AllowdMeasBandWidth'>
                    <el-option label='mbw6' value='mbw6'></el-option>
                    <el-option label='mbw15' value='mbw15'></el-option>
                    <el-option label="mbw25" value="mbw25"></el-option>
                    <el-option label="mbw50" value="mbw50"></el-option>
                    <el-option label="mbw75" value="mbw75"></el-option>
                    <el-option label="mbw100" value="mbw100"></el-option>
                </el-select>
            </el-form-item>
			<el-form-item prop='LTEER_PresAntennaPort1' style="width:40%;min-width:400px;" label="PresAntennaPort1" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTEERDialogForm.LTEER_PresAntennaPort1'>
                    <el-option label='0' value='0'></el-option>
                    <el-option label='1' value='1'></el-option>
                </el-select>
            </el-form-item>
			<el-form-item prop='LTEER_BlackPhysCellIdStart' style="width:48%;min-width:530px;" label="BlackPhysCellIdStart" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addLTEERDialogForm.LTEER_BlackPhysCellIdStart' maxlength="11">
                    <template slot="append"><%=rb.getString("FanWei")%>：0~503 or 268435455(268435455:OFF),Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='LTEER_BlackPhysCellIdRange' style="width:40%;min-width:400px;" label="BlackPhysCellIdRange" label-width="160px" class="selectErrcCls">
                <el-select v-model='addLTEERDialogForm.LTEER_BlackPhysCellIdRange'>
                    <el-option v-for="item in lteBlackPhysCellIdRangeList" :label='item.label' :value='item.value'></el-option>
                </el-select>
            </el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addLTEERDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- LTE NC 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="60%" :visible.sync="addLTENCDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addLTENCDialogForm" :model='addLTENCDialogForm' :rules='addLTENCDialogRules' label-position="top">     		     			            
			<el-form-item prop='LTENC_PLMNID' style="width:40%;min-width:400px;" label="PLMNID" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addLTENCDialogForm.LTENC_PLMNID'>
					<template slot="append">Length：5~6 Digit,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='LTENC_CID' style="width:40%;min-width:400px;" label="CID" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addLTENCDialogForm.LTENC_CID'>
					<template slot="append"><%=rb.getString("FanWei")%>：1~268435455,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='LTENC_EUTRACarrierARFCN' style="width:40%;min-width:400px;" label="EUTRACarrierARFCN" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addLTENCDialogForm.LTENC_EUTRACarrierARFCN'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='LTENC_PhyCellID' style="width:40%;min-width:400px;" label="PhyCellID" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addLTENCDialogForm.LTENC_PhyCellID'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~503,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='LTENC_QOffset' style="width:40%;min-width:400px;" label="QOffset" label-width="160px" class="selectErrcCls">
				<el-select v-model='addLTENCDialogForm.LTENC_QOffset'>
					<el-option v-for="item in QOffsetDataList" :label='item' :value='item'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='LTENC_QRxLevMinOffsetCell' style="width:40%;min-width:400px;" label="QRxLevMinOffsetCell" label-width="160px" class="selectErrcCls">
				<el-select v-model='addLTENCDialogForm.LTENC_QRxLevMinOffsetCell'>
					<el-option label='1' value='1'></el-option>
					<el-option label="2" value="2"></el-option>
					<el-option label="3" value="3"></el-option>
					<el-option label='4' value='4'></el-option>
					<el-option label="5" value="5"></el-option>
					<el-option label="6" value="6"></el-option>
					<el-option label='7' value='7'></el-option>
					<el-option label="8" value="8"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='LTENC_QQualMinOffsetCell' style="width:40%;min-width:400px;" label="QQualMinOffsetCell" label-width="160px" class="selectErrcCls">
				<el-select v-model='addLTENCDialogForm.LTENC_QQualMinOffsetCell'>
					<el-option label='1' value='1'></el-option>
					<el-option label="2" value="2"></el-option>
					<el-option label="3" value="3"></el-option>
					<el-option label='4' value='4'></el-option>
					<el-option label="5" value="5"></el-option>
					<el-option label="6" value="6"></el-option>
					<el-option label='7' value='7'></el-option>
					<el-option label="8" value="8"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='LTENC_CIO' style="width:40%;min-width:400px;" label="CIO" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addLTENCDialogForm.LTENC_CIO'>
					<template slot="append"><%=rb.getString("FanWei")%>：-24~24,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='LTENC_Blacklisted' style="width:40%;min-width:400px;" label="Blacklisted" label-width="160px" class="selectErrcCls">
				<el-select v-model='addLTENCDialogForm.LTENC_Blacklisted'>
					<el-option label='false' value='0'></el-option>
					<el-option label='true' value='1'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='LTENC_TAC' style="width:40%;min-width:400px;" label="TAC" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addLTENCDialogForm.LTENC_TAC'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='LTENC_eNBType' style="width:40%;min-width:400px;" label="eNB Type" label-width="160px" class="selectErrcCls">
				<el-select v-model='addLTENCDialogForm.LTENC_eNBType'>
					<el-option label='Macro' value='0'></el-option>
					<el-option label='Home' value='1'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='LTENC_eNBID' style="width:40%;min-width:400px;" label="eNB ID" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addLTENCDialogForm.LTENC_eNBID'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~1048575,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='LTENC_noRemove' style="width:40%;min-width:400px;" label="No Remove" label-width="160px" class="selectErrcCls">
				<el-select v-model='addLTENCDialogForm.LTENC_noRemove'>
					<el-option label='false' value='0'></el-option>
					<el-option label='true' value='1'></el-option>
				</el-select>
			</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addLTENCDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- NR NF 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="60%" :visible.sync="addNRNFDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addNRNFDialogForm" :model='addNRNFDialogForm' :rules='addNRNFDialogRules' label-position="top">     		     			            
				<el-form-item prop='NRNF_Enable' style="width:40%;min-width:460px;" label="Enable" label-width="160px">
					<el-switch v-model="addNRNFDialogForm.NRNF_Enable" active-value="true" inactive-value="false"></el-switch>
				</el-form-item>
				<el-form-item prop='NRNF_SSBFrequency' style="width:40%;min-width:460px;" label="SSBFrequency" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_SSBFrequency'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_SubCarrierSpacing' style="width:40%;min-width:460px;" label="SubCarrierSpacing" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNFDialogForm.NRNF_SubCarrierSpacing'>
						<el-option label='kHz15' value='0'></el-option>
						<el-option label="kHz30" value="1"></el-option>
						<el-option label="kHz60" value="2"></el-option>
						<el-option label='kHz120' value='3'></el-option>
						<el-option label='kHz240' value='4'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNF_SmtcPeriodicity' style="width:40%;min-width:460px;" label="SmtcPeriodicity" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNFDialogForm.NRNF_SmtcPeriodicity'>
						<el-option label='sf5' value='0'></el-option>
						<el-option label='sf10' value='1'></el-option>
						<el-option label="sf20" value="2"></el-option>
						<el-option label="sf40" value="3"></el-option>
						<el-option label="sf80" value="4"></el-option>
						<el-option label="sf160" value="5"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNF_SmtcOffset' style="width:40%;min-width:460px;" label="SmtcOffset" label-width="160px" class='validate-item' >
					<el-input v-model.trim='addNRNFDialogForm.NRNF_SmtcOffset'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~159,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_SmtcDuration' style="width:40%;min-width:460px;" label="SmtcDuration" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNFDialogForm.NRNF_SmtcDuration'>
						<el-option label='sf1' value='0'></el-option>
						<el-option label='sf2' value='1'></el-option>
						<el-option label="sf3" value="2"></el-option>
						<el-option label="sf4" value="3"></el-option>
						<el-option label="sf5" value="4"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNF_SSBlocksConsolidationRsrp' style="width:40%;min-width:460px;" label="SSBlocksConsolidationRsrp" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_SSBlocksConsolidationRsrp'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_SSBlocksConsolidationRsrq' style="width:40%;min-width:460px;" label="SSBlocksConsolidationRsrq" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_SSBlocksConsolidationRsrq'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_SSBlocksConsolidationSinr' style="width:40%;min-width:460px;" label="SSBlocksConsolidationSinr" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_SSBlocksConsolidationSinr'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_NrofSSBlocksToAverage' style="width:40%;min-width:460px;" label="NrofSSBlocksToAverage" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_NrofSSBlocksToAverage'>
						<template slot="append"><%=rb.getString("FanWei")%>：2~16,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_RsrpOffsetSSB' style="width:40%;min-width:460px;" label="RsrpOffsetSSB" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_RsrpOffsetSSB'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_RsrqOffsetSSB' style="width:40%;min-width:460px;" label="RsrqOffsetSSB" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_RsrqOffsetSSB'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_SinrOffsetSSB' style="width:40%;min-width:460px;" label="SinrOffsetSSB" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_SinrOffsetSSB'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_RsrpOffsetCsiRs' style="width:40%;min-width:460px;" label="RsrpOffsetCsiRs" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_RsrpOffsetCsiRs'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_RsrqOffsetCsiRs' style="width:40%;min-width:460px;" label="RsrqOffsetCsiRs" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_RsrqOffsetCsiRs'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_SinrOffsetCsiRs' style="width:40%;min-width:460px;" label="SinrOffsetCsiRs" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_SinrOffsetCsiRs'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_BitmapType' style="width:40%;min-width:460px;" label="BitmapType" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNFDialogForm.NRNF_BitmapType'>
						<el-option label='short' value='0'></el-option>
						<el-option label='medium' value='1'></el-option>
						<el-option label="long" value="2"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNF_Bitmap' style="width:40%;min-width:460px;" label="Bitmap" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_Bitmap'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~18446744073709551615,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_DeriveSSBIndexFromCell' style="width:40%;min-width:460px;" label="DeriveSSBIndexFromCell" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNFDialogForm.NRNF_DeriveSSBIndexFromCell'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNF_FreqBandIndicatorNR' style="width:40%;min-width:460px;" label="FreqBandIndicatorNR" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_FreqBandIndicatorNR'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~1024,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_OffsetToPointA' style="width:40%;min-width:460px;" label="Offset To Point A" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_OffsetToPointA'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~2199,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNF_SSBSubCarrierOffset' style="width:40%;min-width:460px;" label="SSB Sub Carrier Offset" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNFDialogForm.NRNF_SSBSubCarrierOffset'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addNRNFDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- NR interFREQ Reselection setting 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="60%" :visible.sync="addNRIRSDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addNRIRSDialogForm" :model='addNRIRSDialogForm' :rules='addNRIRSDialogRules' label-position="top">     		     			            
				<el-form-item prop='NRIRS_CarrierFreq' style="width:40%;min-width:460px;" label="CarrierFreq" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_CarrierFreq'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_NrofSSBlocksToAverage' style="width:40%;min-width:460px;" label="NrofSSBlocksToAverage" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_NrofSSBlocksToAverage'>
						<template slot="append"><%=rb.getString("FanWei")%>：2~16,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_ThresholdRSRP' style="width:40%;min-width:460px;" label="ThresholdRSRP" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_ThresholdRSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_ThresholdRSRQ' style="width:40%;min-width:460px;" label="ThresholdRSRQ" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_ThresholdRSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_ThresholdSINR' style="width:40%;min-width:460px;" label="ThresholdSINR" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_ThresholdSINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_SubCarrierSpacing' style="width:40%;min-width:460px;" label="SubCarrierSpacing" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRIRSDialogForm.NRIRS_SubCarrierSpacing'>
						<el-option label='kHz15' value='0'></el-option>
						<el-option label="kHz30" value="1"></el-option>
						<el-option label="kHz60" value="2"></el-option>
						<el-option label='kHz120' value='3'></el-option>
						<el-option label='kHz240' value='4'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRIRS_DeriveSSBIndexFromCell' style="width:40%;min-width:460px;" label="DeriveSSBIndexFromCell" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRIRSDialogForm.NRIRS_DeriveSSBIndexFromCell'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRIRS_QRxLevMin' style="width:40%;min-width:460px;" label="QRxLevMin" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_QRxLevMin'>
						<template slot="append"><%=rb.getString("FanWei")%>：-70~-22,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_QQualMin' style="width:40%;min-width:460px;" label="QQualMin" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_QQualMin'>
						<template slot="append"><%=rb.getString("FanWei")%>：-43~-12,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_PMax' style="width:40%;min-width:460px;" label="PMax" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_PMax'>
						<template slot="append"><%=rb.getString("FanWei")%>：-30~33,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_TReselectionNR' style="width:40%;min-width:460px;" label="TReselectionNR" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRIRSDialogForm.NRIRS_TReselectionNR'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
						<el-option label="2" value="2"></el-option>
						<el-option label="3" value="3"></el-option>
						<el-option label="4" value="4"></el-option>
						<el-option label="5" value="5"></el-option>
						<el-option label="6" value="6"></el-option>
						<el-option label="7" value="7"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRIRS_ThreshXHighP' style="width:40%;min-width:460px;" label="ThreshXHighP" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_ThreshXHighP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_ThreshXLowP' style="width:40%;min-width:460px;" label="ThreshXLowP" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_ThreshXLowP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_ThreshXHighQ' style="width:40%;min-width:460px;" label="ThreshXHighQ" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_ThreshXHighQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_ThreshXLowQ' style="width:40%;min-width:460px;" label="ThreshXLowQ" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_ThreshXLowQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~31,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_CellReselectionPriority' style="width:40%;min-width:460px;" label="CellReselectionPriority" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRIRSDialogForm.NRIRS_CellReselectionPriority'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
						<el-option label="2" value="2"></el-option>
						<el-option label="3" value="3"></el-option>
						<el-option label="4" value="4"></el-option>
						<el-option label="5" value="5"></el-option>
						<el-option label="6" value="6"></el-option>
						<el-option label="7" value="7"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRIRS_CellReselectionSubPriority' style="width:40%;min-width:460px;" label="CellReselectionSubPriority" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRIRSDialogForm.NRIRS_CellReselectionSubPriority'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
						<el-option label="2" value="2"></el-option>
						<el-option label="3" value="3"></el-option>
						<!-- <el-option label="4" value="4"></el-option> --> <!-- 10.2.1版本确认 设备侧不支持 -->
					</el-select>
				</el-form-item>
				<el-form-item prop='NRIRS_QOffsetFreq' style="width:40%;min-width:460px;" label="QOffsetFreq" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_QOffsetFreq'>
						<template slot="append"><%=rb.getString("FanWei")%>：-24~24,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_BlackPhyCellIdStart' style="width:40%;min-width:460px;" label="BlackPhyCellIdStart" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRIRSDialogForm.NRIRS_BlackPhyCellIdStart' maxlength="11">
						<template slot="append"><%=rb.getString("FanWei")%>：0~1007 or 268435455(268435455:OFF),Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRIRS_BlackPhyCellIdRange' style="width:40%;min-width:460px;" label="BlackPhysCellIdRange" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRIRSDialogForm.NRIRS_BlackPhyCellIdRange'>
						<el-option v-for="item in nrBlackPhysCellIdRangeList" :label='item.label' :value='item.value'></el-option>
					</el-select>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addNRIRSDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- NR NC 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="60%" :visible.sync="addNRNCDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addNRNCDialogForm" :model='addNRNCDialogForm' :rules='addNRNCDialogRules' label-position="top">     		     			            
				<el-form-item prop='NRNC_PLMNID' style="width:40%;min-width:400px;" label="PLMNID" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNCDialogForm.NRNC_PLMNID'>
						<template slot="append">Length：5~6 Digit,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNC_CID' style="width:40%;min-width:400px;" label="CID" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNCDialogForm.NRNC_CID'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~68719476735,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNC_NRCarrierARFCN' style="width:40%;min-width:400px;" label="NRCarrierARFCN" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNCDialogForm.NRNC_NRCarrierARFCN'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNC_SSBFrequency' style="width:40%;min-width:400px;" label="ssbFrequency" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNCDialogForm.NRNC_SSBFrequency'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNC_SSBSubcarrierSpacing' style="width:40%;min-width:400px;" label="ssbSubcarrierSpacing" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNCDialogForm.NRNC_SSBSubcarrierSpacing'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
						<el-option label="2" value="2"></el-option>
						<el-option label="3" value="3"></el-option>
						<el-option label="4" value="4"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNC_PhyCellID' style="width:40%;min-width:400px;" label="PhyCellID" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNCDialogForm.NRNC_PhyCellID'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~1007,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNC_QOffset' style="width:40%;min-width:400px;" label="QOffset" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNCDialogForm.NRNC_QOffset'>
							<el-option v-for="item in QOffsetDataList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNC_QRxLevMinOffsetCell' style="width:40%;min-width:400px;" label="QRxLevMinOffsetCell" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNCDialogForm.NRNC_QRxLevMinOffsetCell'>
						<el-option label='1' value='1'></el-option>
						<el-option label="2" value="2"></el-option>
						<el-option label="3" value="3"></el-option>
						<el-option label='4' value='4'></el-option>
						<el-option label="5" value="5"></el-option>
						<el-option label="6" value="6"></el-option>
						<el-option label='7' value='7'></el-option>
						<el-option label="8" value="8"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNC_QQualMinOffsetCell' style="width:40%;min-width:400px;" label="QQualMinOffsetCell" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNCDialogForm.NRNC_QQualMinOffsetCell'>
						<el-option label='1' value='1'></el-option>
						<el-option label="2" value="2"></el-option>
						<el-option label="3" value="3"></el-option>
						<el-option label='4' value='4'></el-option>
						<el-option label="5" value="5"></el-option>
						<el-option label="6" value="6"></el-option>
						<el-option label='7' value='7'></el-option>
						<el-option label="8" value="8"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNC_CIO' style="width:40%;min-width:400px;" label="CIO" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNCDialogForm.NRNC_CIO'>
							<el-option v-for="item in CIODataList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNC_Blacklisted' style="width:40%;min-width:400px;" label="Blacklisted" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNCDialogForm.NRNC_Blacklisted'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNC_TAC' style="width:40%;min-width:400px;" label="TAC" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNCDialogForm.NRNC_TAC'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~16777215,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='NRNC_noRemove' style="width:40%;min-width:400px;" label="No Remove" label-width="160px" class="selectErrcCls">
					<el-select v-model='addNRNCDialogForm.NRNC_noRemove'>
						<el-option label='false' value='0'></el-option>
						<el-option label='true' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='NRNC_gnbIdLength' style="width:40%;min-width:400px;" label="gNB ID Length" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addNRNCDialogForm.NRNC_gnbIdLength'>
						<template slot="append"><%=rb.getString("FanWei")%>：22~32,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addNRNCDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- QOS List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="60%" :visible.sync="addQOSDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addQOSDialogForm" :model='addQOSDialogForm' :rules='addQOSDialogRules' label-position="top">     		     			            
				<el-form-item prop='QOS_Enable' style="width:40%;min-width:460px;" label="QOS" label-width="160px">
					<el-switch v-model="addQOSDialogForm.QOS_Enable" active-value="1" inactive-value="0"></el-switch>
				</el-form-item>
				<el-form-item prop='QOS_MappingDrbIndex' style="width:40%;min-width:400px;" label="Mapping DrbIndex" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addQOSDialogForm.QOS_MappingDrbIndex'>
						<template slot="append"><%=rb.getString("FanWei")%>：5~32,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_5QI' style="width:40%;min-width:400px;" label="5QI" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addQOSDialogForm.QOS_5QI'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~255,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_Type' style="width:40%;min-width:400px;" label="Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_Type'>
						<el-option label='GBR' value='0'></el-option>
						<el-option label='NON-GBR' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_Priority' style="width:40%;min-width:400px;" label="Priority" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addQOSDialogForm.QOS_Priority'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~16,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_MinBr' style="width:40%;min-width:400px;" label="MinBr" label-width="160px" class='validate-item'>
					<el-input v-model.trim='addQOSDialogForm.QOS_MinBr'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~3999999999,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_IsDefault' style="width:40%;min-width:400px;" label="IsDefault" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_IsDefault'>
						<el-option label='ON' value='1'></el-option>
						<el-option label='OFF' value='0'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_UeInactivityTimerConfig' style="width:40%;min-width:400px;" label="UeInactivityTimerConfig" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_UeInactivityTimerConfig'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_TReorderingPdcp' style="width:40%;min-width:400px;" label="TReorderingPdcp" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_TReorderingPdcp'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~35,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_TReorderingUE' style="width:40%;min-width:400px;" label="TReorderingUE" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_TReorderingUE'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~35,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_DiscardTimer' style="width:40%;min-width:400px;" label="DiscardTimer" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_DiscardTimer'>
						<el-option v-for="item in DiscardTimerList" :label='item.label' :value='item.value'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_StatusReportRequired' style="width:40%;min-width:400px;" label="StatusReportRequired" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_StatusReportRequired'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_PdcpSnSizeUL' style="width:40%;min-width:400px;" label="PdcpSnSizeUL" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_PdcpSnSizeUL'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_PdcpSnSizeDL' style="width:40%;min-width:400px;" label="PdcpSnSizeDL" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_PdcpSnSizeDL'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_Dscp' style="width:40%;min-width:400px;" label="Dscp" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_Dscp'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~63,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_RlcMode' style="width:40%;min-width:400px;" label="RlcMode" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_RlcMode'>
						<el-option label='AM' value='1'></el-option>
						<el-option label='UM' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_SnFieldLengthAmDL' style="width:40%;min-width:400px;" label="SnFieldLengthAmDL" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_SnFieldLengthAmDL'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~255,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_SnFieldLengthAmUL' style="width:40%;min-width:400px;" label="SnFieldLengthAmUL" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_SnFieldLengthAmUL'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~255,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_SnFieldLengthUmDL' style="width:40%;min-width:400px;" label="SnFieldLengthUmDL" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_SnFieldLengthUmDL'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~255,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_SnFieldLengthUmUL' style="width:40%;min-width:400px;" label="SnFieldLengthUmUL" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_SnFieldLengthUmUL'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~255,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='QOS_ULConfig' style="width:40%;min-width:400px;" label="ULConfig" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_ULConfig'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
						<el-option label='2' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_EnableRohc' style="width:40%;min-width:400px;" label="EnableRohc" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_EnableRohc'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_RohcProfile0x0001' style="width:40%;min-width:400px;" label="RohcProfile0x0001" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_RohcProfile0x0001'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_RohcProfile0x0002' style="width:40%;min-width:400px;" label="RohcProfile0x0002" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_RohcProfile0x0002'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_RohcProfile0x0006' style="width:40%;min-width:400px;" label="RohcProfile0x0006" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_RohcProfile0x0006'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_PdcpDuplicationActivated' style="width:40%;min-width:400px;" label="PdcpDuplicationActivated" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_PdcpDuplicationActivated'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_PrimaryPathDL' style="width:40%;min-width:400px;" label="PrimaryPathDL" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_PrimaryPathDL'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_PrimaryPath' style="width:40%;min-width:400px;" label="PrimaryPath" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_PrimaryPath'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_ULDataSplitThreshold' style="width:40%;min-width:400px;" label="ULDataSplitThreshold" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_ULDataSplitThreshold'>
						<el-option v-for="item in DataSplitThresholdList" :label='item.label' :value='item.value'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_DLDataSplitThreshold' style="width:40%;min-width:400px;" label="DLDataSplitThreshold" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_DLDataSplitThreshold'>
						<el-option v-for="item in DataSplitThresholdList" :label='item.label' :value='item.value'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_AllowedIntegrityAlgo' style="width:40%;min-width:400px;" label="AllowedIntegrityAlgo" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_AllowedIntegrityAlgo'>
						<el-option label='0' value='0'></el-option>
						<el-option label='1' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_LongDrxCycle' style="width:40%;min-width:400px;" label="LongDrxCycle" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_LongDrxCycle' filterable>
						<el-option v-for="item in generateArray(0,19)" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_ShortDrxCycle' style="width:40%;min-width:400px;" label="ShortDrxCycle" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_ShortDrxCycle' filterable>
						<el-option v-for="item in generateArray(0,22)" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_ShortDrxCycleTimer' style="width:40%;min-width:400px;" label="ShortDrxCycleTimer" label-width="160px" class="selectErrcCls">
					<el-select v-model='addQOSDialogForm.QOS_ShortDrxCycleTimer' filterable>
						<el-option v-for="item in generateArray(1,16)" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='QOS_DrBlnactivityTimerConfig' style="width:40%;min-width:400px;" label="DrBlnactivityTimerConfig" label-width="160px" class="validate-item">
					<el-input v-model.trim='addQOSDialogForm.QOS_DrBlnactivityTimerConfig'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addQOSDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- SST List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addSSTDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addSSTDialogForm" :model='addSSTDialogForm' :rules='addSSTDialogRules' label-position="top">     		     			            
			<el-form-item prop='SST_Sst' style="width:40%;min-width:400px;" label="Sst" label-width="160px" class="validate-item">
                <span slot="label" class="labelIconCls">
                    Sst
                    <span class="el-icon-menu-help el-icon" 
                        :title="`1( eMBB: Slice suitable for the handing of 5G enhanced Mobile Broadband. )\n2( URLLC: Slice suitable for the handing of ultra- reliable low latency communications. )\n3( MIoT: Slice suitable for the handing of massive IoT. )`"
                    ></span>
                </span>
                <el-input v-model.trim='addSSTDialogForm.SST_Sst'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~255,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='SST_SstResourceType' style="width:40%;min-width:400px;" label="SstResourceType" label-width="160px" class="selectErrcCls">
				<el-select v-model='addSSTDialogForm.SST_SstResourceType'>
					<el-option label='0' value='0'></el-option>
					<el-option label='1' value='1'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='SST_MaxResourceReserved' style="width:40%;min-width:400px;" label="MaxResourceReserved" label-width="160px" class="validate-item">
				<el-input v-model.trim='addSSTDialogForm.SST_MaxResourceReserved'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~273,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='SST_MinResourceReserved' style="width:40%;min-width:400px;" label="MinResourceReserved" label-width="160px" class="validate-item">
				<el-input v-model.trim='addSSTDialogForm.SST_MinResourceReserved'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~273,Integer</template>
				</el-input>
			</el-form-item>
		</el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addSSTDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- A1 List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addA1DialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addA1DialogForm" :model='addA1DialogForm' :rules='addA1DialogRules' label-position="top">     		     			            
				<el-form-item prop='A1_Enable' style="width:40%;min-width:400px;" label="A1" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA1DialogForm.A1_Enable'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_ThresholdTriggerType' style="width:40%;min-width:400px;" label="Threshold Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA1DialogForm.A1_ThresholdTriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_ThresholdRSRP' style="width:40%;min-width:400px;" label="A1 Threshold RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA1DialogForm.A1_ThresholdRSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A1_ThresholdRSRQ' style="width:40%;min-width:400px;" label="A1 Threshold RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA1DialogForm.A1_ThresholdRSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A1_ThresholdSINR' style="width:40%;min-width:400px;" label="A1 Threshold SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA1DialogForm.A1_ThresholdSINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A1_ReportOnLeave' style="width:40%;min-width:400px;" label="Report On Leave" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA1DialogForm.A1_ReportOnLeave'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_Hysteresis' style="width:40%;min-width:400px;" label="Hysteresis" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA1DialogForm.A1_Hysteresis'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A1_MaxReportCells' style="width:40%;min-width:400px;" label="Max Report Cells" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA1DialogForm.A1_MaxReportCells'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A1_MeasurePurpose' style="width:40%;min-width:400px;" label="Measure Purpose" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA1DialogForm.A1_MeasurePurpose'>
						<el-option label='Inter Frequency Measure' value='1'></el-option>
						<el-option label='Inter RAT EUTRA Measure' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_ReportAmount' style="width:40%;min-width:400px;" label="Report Amount" label-width="160px" class="validate-item">
					<el-select v-model='addA1DialogForm.A1_ReportAmount'>
						<el-option v-for="item in ReportAmountList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_MaxNrofRSIndexToReport' style="width:40%;min-width:400px;" label="Max Nr of RS Index To Report" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA1DialogForm.A1_MaxNrofRSIndexToReport'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~32,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A1_ReportInterval' style="width:40%;min-width:400px;" label="Report Interval" label-width="160px" class="validate-item">
					<el-select v-model='addA1DialogForm.A1_ReportInterval'>
						<el-option v-for="item in ReportIntervalList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_ReportQuantity' style="width:40%;min-width:400px;" label="Report Quantity" label-width="160px">
                    <el-checkbox-group v-model="addA1DialogForm.A1_ReportQuantity" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A1_RptQuantityRsIndex' style="width:40%;min-width:400px;" label="Rpt Quantity Rs Index" label-width="160px">
                    <el-checkbox-group v-model="addA1DialogForm.A1_RptQuantityRsIndex" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A1_TimeToTrigger' style="width:40%;min-width:400px;" label="Time To Trigger" label-width="160px" class="validate-item">
					<el-select v-model='addA1DialogForm.A1_TimeToTrigger'>
						<el-option v-for="item in TimeToTriggerList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_RsType' style="width:40%;min-width:400px;" label="Rs Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA1DialogForm.A1_RsType'>
						<el-option label='ssb' value='ssb'></el-option>
						<el-option label='csi-rs' value='csi-rs'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_IncludeBeamMeasurements' style="width:40%;min-width:400px;" label="Include Beam Measurements" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA1DialogForm.A1_IncludeBeamMeasurements'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A1_PLMN' style="width:40%;min-width:400px;" label="PLMN" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA1DialogForm.A1_PLMN'>
						<template slot="append">Length：5~6 Digit,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addA1DialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- A2 List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addA2DialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addA2DialogForm" :model='addA2DialogForm' :rules='addA2DialogRules' label-position="top">     		     			            
				<el-form-item prop='A2_Enable' style="width:40%;min-width:400px;" label="A2" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA2DialogForm.A2_Enable'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_ThresholdTriggerType' style="width:40%;min-width:400px;" label="Threshold Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA2DialogForm.A2_ThresholdTriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_ThresholdRSRP' style="width:40%;min-width:400px;" label="A2 Threshold RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA2DialogForm.A2_ThresholdRSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A2_ThresholdRSRQ' style="width:40%;min-width:400px;" label="A2 Threshold RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA2DialogForm.A2_ThresholdRSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A2_ThresholdSINR' style="width:40%;min-width:400px;" label="A2 Threshold SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA2DialogForm.A2_ThresholdSINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A2_ReportOnLeave' style="width:40%;min-width:400px;" label="Report On Leave" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA2DialogForm.A2_ReportOnLeave'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_Hysteresis' style="width:40%;min-width:400px;" label="Hysteresis" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA2DialogForm.A2_Hysteresis'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A2_MaxReportCells' style="width:40%;min-width:400px;" label="Max Report Cells" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA2DialogForm.A2_MaxReportCells'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A2_MeasurePurpose' style="width:40%;min-width:400px;" label="Measure Purpose" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA2DialogForm.A2_MeasurePurpose'>
						<el-option label='Inter Frequency Measure' value='1'></el-option>
						<el-option label='Inter RAT EUTRA Data Measure' value='2'></el-option>
                        <el-option label='Inter RAT EUTRA Voice Measure' value='3'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_ReportAmount' style="width:40%;min-width:400px;" label="Report Amount" label-width="160px" class="validate-item">
					<el-select v-model='addA2DialogForm.A2_ReportAmount'>
						<el-option v-for="item in ReportAmountList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_MaxNrofRSIndexToReport' style="width:40%;min-width:400px;" label="Max Nr of RS Index To Report" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA2DialogForm.A2_MaxNrofRSIndexToReport'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~32,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A2_ReportInterval' style="width:40%;min-width:400px;" label="Report Interval" label-width="160px" class="validate-item">
					<el-select v-model='addA2DialogForm.A2_ReportInterval'>
						<el-option v-for="item in ReportIntervalList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_ReportQuantity' style="width:40%;min-width:400px;" label="Report Quantity" label-width="160px">
                    <el-checkbox-group v-model="addA2DialogForm.A2_ReportQuantity" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A2_RptQuantityRsIndex' style="width:40%;min-width:400px;" label="Rpt Quantity Rs Index" label-width="160px">
                    <el-checkbox-group v-model="addA2DialogForm.A2_RptQuantityRsIndex" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A2_TimeToTrigger' style="width:40%;min-width:400px;" label="Time To Trigger" label-width="160px" class="validate-item">
					<el-select v-model='addA2DialogForm.A2_TimeToTrigger'>
						<el-option v-for="item in TimeToTriggerList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_RsType' style="width:40%;min-width:400px;" label="Rs Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA2DialogForm.A2_RsType'>
						<el-option label='ssb' value='ssb'></el-option>
						<el-option label='csi-rs' value='csi-rs'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_IncludeBeamMeasurements' style="width:40%;min-width:400px;" label="Include Beam Measurements" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA2DialogForm.A2_IncludeBeamMeasurements'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A2_PLMN' style="width:40%;min-width:400px;" label="PLMN" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA2DialogForm.A2_PLMN'>
						<template slot="append">Length：5~6 Digit,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addA2DialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- A3 List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addA3DialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addA3DialogForm" :model='addA3DialogForm' :rules='addA3DialogRules' label-position="top">     		     			            
				<el-form-item prop='A3_Enable' style="width:40%;min-width:400px;" label="A3" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA3DialogForm.A3_Enable'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_ThresholdTriggerType' style="width:40%;min-width:400px;" label="Threshold Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA3DialogForm.A3_ThresholdTriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_OffsetRSRP' style="width:40%;min-width:400px;" label="A3 Offset RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA3DialogForm.A3_OffsetRSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：-30~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A3_OffsetRSRQ' style="width:40%;min-width:400px;" label="A3 Offset RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA3DialogForm.A3_OffsetRSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：-30~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A3_OffsetSINR' style="width:40%;min-width:400px;" label="A3 Offset SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA3DialogForm.A3_OffsetSINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：-30~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A3_Hysteresis' style="width:40%;min-width:400px;" label="Hysteresis" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA3DialogForm.A3_Hysteresis'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A3_MaxReportCells' style="width:40%;min-width:400px;" label="Max Report Cells" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA3DialogForm.A3_MaxReportCells'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A3_MeasurePurpose' style="width:40%;min-width:400px;" label="Measure Purpose" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA3DialogForm.A3_MeasurePurpose'>
						<el-option label='Intra Frequency Measure' value='1'></el-option>
						<el-option label='Inter Frequency Measure' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_ReportAmount' style="width:40%;min-width:400px;" label="Report Amount" label-width="160px" class="validate-item">
					<el-select v-model='addA3DialogForm.A3_ReportAmount'>
						<el-option v-for="item in ReportAmountList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_MaxNrofRSIndexToReport' style="width:40%;min-width:400px;" label="Max Nr of RS Index To Report" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA3DialogForm.A3_MaxNrofRSIndexToReport'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~32,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A3_ReportOnLeave' style="width:40%;min-width:400px;" label="Report On Leave" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA3DialogForm.A3_ReportOnLeave'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_ReportInterval' style="width:40%;min-width:400px;" label="Report Interval" label-width="160px" class="validate-item">
					<el-select v-model='addA3DialogForm.A3_ReportInterval'>
						<el-option v-for="item in ReportIntervalList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_ReportQuantity' style="width:40%;min-width:400px;" label="Report Quantity" label-width="160px">
                    <el-checkbox-group v-model="addA3DialogForm.A3_ReportQuantity" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A3_RptQuantityRsIndex' style="width:40%;min-width:400px;" label="Rpt Quantity Rs Index" label-width="160px">
                    <el-checkbox-group v-model="addA3DialogForm.A3_RptQuantityRsIndex" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A3_TimeToTrigger' style="width:40%;min-width:400px;" label="Time To Trigger" label-width="160px" class="validate-item">
					<el-select v-model='addA3DialogForm.A3_TimeToTrigger'>
						<el-option v-for="item in TimeToTriggerList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_UseWhiteCellList' style="width:40%;min-width:400px;" label="UseWhiteCellList" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA3DialogForm.A3_UseWhiteCellList'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_RsType' style="width:40%;min-width:400px;" label="Rs Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA3DialogForm.A3_RsType'>
						<el-option label='ssb' value='ssb'></el-option>
						<el-option label='csi-rs' value='csi-rs'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_IncludeBeamMeasurements' style="width:40%;min-width:400px;" label="Include Beam Measurements" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA3DialogForm.A3_IncludeBeamMeasurements'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A3_PLMN' style="width:40%;min-width:400px;" label="PLMN" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA3DialogForm.A3_PLMN'>
						<template slot="append">Length：5~6 Digit,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addA3DialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- A4 List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addA4DialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addA4DialogForm" :model='addA4DialogForm' :rules='addA4DialogRules' label-position="top">     		     			            
				<el-form-item prop='A4_Enable' style="width:40%;min-width:400px;" label="A4" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA4DialogForm.A4_Enable'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_ThresholdTriggerType' style="width:40%;min-width:400px;" label="Threshold Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA4DialogForm.A4_ThresholdTriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_ThresholdRSRP' style="width:40%;min-width:400px;" label="A4 Threshold RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA4DialogForm.A4_ThresholdRSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A4_ThresholdRSRQ' style="width:40%;min-width:400px;" label="A4 Threshold RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA4DialogForm.A4_ThresholdRSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A4_ThresholdSINR' style="width:40%;min-width:400px;" label="A4 Threshold SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA4DialogForm.A4_ThresholdSINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A4_Hysteresis' style="width:40%;min-width:400px;" label="Hysteresis" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA4DialogForm.A4_Hysteresis'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A4_MaxReportCells' style="width:40%;min-width:400px;" label="Max Report Cells" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA4DialogForm.A4_MaxReportCells'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A4_MeasurePurpose' style="width:40%;min-width:400px;" label="Measure Purpose" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA4DialogForm.A4_MeasurePurpose'>
						<el-option label='Inter Frequency Measure' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_ReportAmount' style="width:40%;min-width:400px;" label="Report Amount" label-width="160px" class="validate-item">
					<el-select v-model='addA4DialogForm.A4_ReportAmount'>
						<el-option v-for="item in ReportAmountList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_MaxNrofRSIndexToReport' style="width:40%;min-width:400px;" label="Max Nr of RS Index To Report" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA4DialogForm.A4_MaxNrofRSIndexToReport'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~32,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A4_ReportOnLeave' style="width:40%;min-width:400px;" label="Report On Leave" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA4DialogForm.A4_ReportOnLeave'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_ReportInterval' style="width:40%;min-width:400px;" label="Report Interval" label-width="160px" class="validate-item">
					<el-select v-model='addA4DialogForm.A4_ReportInterval'>
						<el-option v-for="item in ReportIntervalList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_ReportQuantity' style="width:40%;min-width:400px;" label="Report Quantity" label-width="160px">
                    <el-checkbox-group v-model="addA4DialogForm.A4_ReportQuantity" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A4_RptQuantityRsIndex' style="width:40%;min-width:400px;" label="Rpt Quantity Rs Index" label-width="160px">
                    <el-checkbox-group v-model="addA4DialogForm.A4_RptQuantityRsIndex" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A4_TimeToTrigger' style="width:40%;min-width:400px;" label="Time To Trigger" label-width="160px" class="validate-item">
					<el-select v-model='addA4DialogForm.A4_TimeToTrigger'>
						<el-option v-for="item in TimeToTriggerList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_UseWhiteCellList' style="width:40%;min-width:400px;" label="UseWhiteCellList" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA4DialogForm.A4_UseWhiteCellList'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_RsType' style="width:40%;min-width:400px;" label="Rs Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA4DialogForm.A4_RsType'>
						<el-option label='ssb' value='ssb'></el-option>
						<el-option label='csi-rs' value='csi-rs'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_IncludeBeamMeasurements' style="width:40%;min-width:400px;" label="Include Beam Measurements" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA4DialogForm.A4_IncludeBeamMeasurements'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A4_PLMN' style="width:40%;min-width:400px;" label="PLMN" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA4DialogForm.A4_PLMN'>
						<template slot="append">Length：5~6 Digit,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addA4DialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- A5 List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addA5DialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addA5DialogForm" :model='addA5DialogForm' :rules='addA5DialogRules' label-position="top">     		     			            
				<el-form-item prop='A5_Enable' style="width:40%;min-width:400px;" label="A5" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA5DialogForm.A5_Enable'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_ThresholdTriggerType' style="width:40%;min-width:400px;" label="Threshold Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA5DialogForm.A5_ThresholdTriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_Threshold1RSRP' style="width:40%;min-width:400px;" label="A5 Threshold1 RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_Threshold1RSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_Threshold1RSRQ' style="width:40%;min-width:400px;" label="A5 Threshold1 RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_Threshold1RSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_Threshold1SINR' style="width:40%;min-width:400px;" label="A5 Threshold1 SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_Threshold1SINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_Threshold2TriggerType' style="width:40%;min-width:400px;" label="Threshold2 Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA5DialogForm.A5_Threshold2TriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_Threshold2RSRP' style="width:40%;min-width:400px;" label="A5 Threshold2 RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_Threshold2RSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_Threshold2RSRQ' style="width:40%;min-width:400px;" label="A5 Threshold2 RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_Threshold2RSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_Threshold2SINR' style="width:40%;min-width:400px;" label="A5 Threshold2 SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_Threshold2SINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_ReportOnLeave' style="width:40%;min-width:400px;" label="Report On Leave" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA5DialogForm.A5_ReportOnLeave'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_Hysteresis' style="width:40%;min-width:400px;" label="Hysteresis" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_Hysteresis'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_MaxReportCells' style="width:40%;min-width:400px;" label="Max Report Cells" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_MaxReportCells'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_MeasurePurpose' style="width:40%;min-width:400px;" label="Measure Purpose" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA5DialogForm.A5_MeasurePurpose'>
						<el-option label='Inter Frequency Measure' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_ReportAmount' style="width:40%;min-width:400px;" label="Report Amount" label-width="160px" class="validate-item">
					<el-select v-model='addA5DialogForm.A5_ReportAmount'>
						<el-option v-for="item in ReportAmountList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_MaxNrofRSIndexToReport' style="width:40%;min-width:400px;" label="Max Nr of RS Index To Report" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_MaxNrofRSIndexToReport'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~32,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='A5_ReportInterval' style="width:40%;min-width:400px;" label="Report Interval" label-width="160px" class="validate-item">
					<el-select v-model='addA5DialogForm.A5_ReportInterval'>
						<el-option v-for="item in ReportIntervalList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_ReportQuantity' style="width:40%;min-width:400px;" label="Report Quantity" label-width="160px">
                    <el-checkbox-group v-model="addA5DialogForm.A5_ReportQuantity" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A5_RptQuantityRsIndex' style="width:40%;min-width:400px;" label="Rpt Quantity Rs Index" label-width="160px">
                    <el-checkbox-group v-model="addA5DialogForm.A5_RptQuantityRsIndex" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='A5_TimeToTrigger' style="width:40%;min-width:400px;" label="Time To Trigger" label-width="160px" class="validate-item">
					<el-select v-model='addA5DialogForm.A5_TimeToTrigger'>
						<el-option v-for="item in TimeToTriggerList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_UseWhiteCellList' style="width:40%;min-width:400px;" label="UseWhiteCellList" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA5DialogForm.A5_UseWhiteCellList'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_RsType' style="width:40%;min-width:400px;" label="Rs Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA5DialogForm.A5_RsType'>
						<el-option label='ssb' value='ssb'></el-option>
						<el-option label='csi-rs' value='csi-rs'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_IncludeBeamMeasurements' style="width:40%;min-width:400px;" label="Include Beam Measurements" label-width="160px" class="selectErrcCls">
					<el-select v-model='addA5DialogForm.A5_IncludeBeamMeasurements'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='A5_PLMN' style="width:40%;min-width:400px;" label="PLMN" label-width="160px" class="validate-item">
					<el-input v-model.trim='addA5DialogForm.A5_PLMN'>
						<template slot="append">Length：5~6 Digit,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addA5DialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- B1 List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addB1DialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addB1DialogForm" :model='addB1DialogForm' :rules='addB1DialogRules' label-position="top">     		     			            
				<el-form-item prop='B1_Enable' style="width:40%;min-width:400px;" label="B1" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB1DialogForm.B1_Enable'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B1_ThresholdTriggerType' style="width:40%;min-width:400px;" label="Threshold Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB1DialogForm.B1_ThresholdTriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B1_Threshold1EUTRARSRP' style="width:40%;min-width:400px;" label="B1 Threshold1 EUTRA RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB1DialogForm.B1_Threshold1EUTRARSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~97,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B1_Threshold1EUTRARSRQ' style="width:40%;min-width:400px;" label="B1 Threshold1 EUTRA RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB1DialogForm.B1_Threshold1EUTRARSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~34,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B1_Threshold1EUTRASINR' style="width:40%;min-width:400px;" label="B1 Threshold1 EUTRA SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB1DialogForm.B1_Threshold1EUTRASINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B1_Hysteresis' style="width:40%;min-width:400px;" label="Hysteresis" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB1DialogForm.B1_Hysteresis'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B1_MaxReportCells' style="width:40%;min-width:400px;" label="Max Report Cells" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB1DialogForm.B1_MaxReportCells'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B1_MeasurePurpose' style="width:40%;min-width:400px;" label="Measure Purpose" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB1DialogForm.B1_MeasurePurpose'>
						<el-option label='Inter RAT EUTRA Data Measure' value='1'></el-option>
                        <el-option label='Inter RAT EUTRA Voice Measure' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B1_ReportAmount' style="width:40%;min-width:400px;" label="Report Amount" label-width="160px" class="validate-item">
					<el-select v-model='addB1DialogForm.B1_ReportAmount'>
						<el-option v-for="item in ReportAmountList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B1_ReportInterval' style="width:40%;min-width:400px;" label="Report Interval" label-width="160px" class="validate-item">
					<el-select v-model='addB1DialogForm.B1_ReportInterval'>
						<el-option v-for="item in ReportIntervalList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B1_TimeToTrigger' style="width:40%;min-width:400px;" label="Time To Trigger" label-width="160px" class="validate-item">
					<el-select v-model='addB1DialogForm.B1_TimeToTrigger'>
						<el-option v-for="item in TimeToTriggerList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B1_ReportQuantity' style="width:40%;min-width:400px;" label="Report Quantity" label-width="160px">
                    <el-checkbox-group v-model="addB1DialogForm.B1_ReportQuantity" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='B1_ReportOnLeave' style="width:40%;min-width:400px;" label="Report On Leave" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB1DialogForm.B1_ReportOnLeave'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B1_PLMN' style="width:40%;min-width:400px;" label="PLMN" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB1DialogForm.B1_PLMN'>
						<template slot="append">Length：5~6 Digit,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addB1DialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- B2 List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addB2DialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls">
			<el-form ref="addB2DialogForm" :model='addB2DialogForm' :rules='addB2DialogRules' label-position="top">     		     			            
				<el-form-item prop='B2_Enable' style="width:40%;min-width:400px;" label="B2" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB2DialogForm.B2_Enable'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B2_ThresholdTriggerType' style="width:40%;min-width:400px;" label="Threshold Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB2DialogForm.B2_ThresholdTriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B2_Threshold1RSRP' style="width:40%;min-width:400px;" label="B2 Threshold1 RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_Threshold1RSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B2_Threshold1RSRQ' style="width:40%;min-width:400px;" label="B2 Threshold1 RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_Threshold1RSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B2_Threshold1SINR' style="width:40%;min-width:400px;" label="B2 Threshold1 SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_Threshold1SINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B2_Threshold2TriggerType' style="width:40%;min-width:400px;" label="Threshold2 Trigger Type" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB2DialogForm.B2_Threshold2TriggerType'>
						<el-option label='rsrp' value='0'></el-option>
						<el-option label='rsrq' value='1'></el-option>
						<el-option label='sinr' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B2_Threshold2EUTRARSRP' style="width:40%;min-width:400px;" label="B2 Threshold2 EUTRA RSRP" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_Threshold2EUTRARSRP'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~97,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B2_Threshold2EUTRARSRQ' style="width:40%;min-width:400px;" label="B2 Threshold2 EUTRA RSRQ" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_Threshold2EUTRARSRQ'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~34,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B2_Threshold2EUTRASINR' style="width:40%;min-width:400px;" label="B2 Threshold2 EUTRA SINR" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_Threshold2EUTRASINR'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B2_Hysteresis' style="width:40%;min-width:400px;" label="Hysteresis" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_Hysteresis'>
						<template slot="append"><%=rb.getString("FanWei")%>：0~30,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B2_MaxReportCells' style="width:40%;min-width:400px;" label="Max Report Cells" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_MaxReportCells'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
					</el-input>
				</el-form-item>
				<el-form-item prop='B2_MeasurePurpose' style="width:40%;min-width:400px;" label="Measure Purpose" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB2DialogForm.B2_MeasurePurpose'>
						<el-option label='Inter RAT EUTRA Data Measure' value='1'></el-option>
                        <el-option label='Inter RAT EUTRA Voice Measure' value='2'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B2_ReportAmount' style="width:40%;min-width:400px;" label="Report Amount" label-width="160px" class="validate-item">
					<el-select v-model='addB2DialogForm.B2_ReportAmount'>
						<el-option v-for="item in ReportAmountList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B2_ReportInterval' style="width:40%;min-width:400px;" label="Report Interval" label-width="160px" class="validate-item">
					<el-select v-model='addB2DialogForm.B2_ReportInterval'>
						<el-option v-for="item in ReportIntervalList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B2_TimeToTrigger' style="width:40%;min-width:400px;" label="Time To Trigger" label-width="160px" class="validate-item">
					<el-select v-model='addB2DialogForm.B2_TimeToTrigger'>
						<el-option v-for="item in TimeToTriggerList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B2_ReportQuantity' style="width:40%;min-width:400px;" label="Report Quantity" label-width="160px">
                    <el-checkbox-group v-model="addB2DialogForm.B2_ReportQuantity" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
				<el-form-item prop='B2_ReportOnLeave' style="width:40%;min-width:400px;" label="Report On Leave" label-width="160px" class="selectErrcCls">
					<el-select v-model='addB2DialogForm.B2_ReportOnLeave'>
						<el-option label='OFF' value='0'></el-option>
						<el-option label='ON' value='1'></el-option>
					</el-select>
				</el-form-item>
				<el-form-item prop='B2_PLMN' style="width:40%;min-width:400px;" label="PLMN" label-width="160px" class="validate-item">
					<el-input v-model.trim='addB2DialogForm.B2_PLMN'>
						<template slot="append">Length：5~6 Digit,Integer</template>
					</el-input>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addB2DialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
    <!-- PeriodMeasure List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addPeriodMeasureDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<div class="gnbConfigAddMainBoxCls" style="height: 240px;">
			<el-form ref="addPeriodMeasureDialogForm" :model='addPeriodMeasureDialogForm' :rules='addPeriodMeasureDialogRules' label-position="top">     		     			            
				<el-form-item prop='PeriodMeasure_ReportQuantity' style="width:40%;min-width:400px;" label="Report Quantity" label-width="160px">
                    <el-checkbox-group v-model="addPeriodMeasureDialogForm.PeriodMeasure_ReportQuantity" size="small">
                        <el-checkbox label="rsrp" border>rsrp</el-checkbox>
                        <el-checkbox label="rsrq" border style='margin-left: 20px;'>rsrq</el-checkbox>
                        <el-checkbox label="sinr" border style='margin-left: 20px;'>sinr</el-checkbox>
                    </el-checkbox-group>
				</el-form-item>
                <el-form-item prop='PeriodMeasure_MaxReportCells' style="width:40%;min-width:400px;" label="Max Report Cells" label-width="160px" class="validate-item">
					<el-input v-model.trim='addPeriodMeasureDialogForm.PeriodMeasure_MaxReportCells'>
						<template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
					</el-input>
				</el-form-item>
                <el-form-item prop='PeriodMeasure_MeasurePurpose' style="width:40%;min-width:400px;" label="Measure Purpose" label-width="160px" class="selectErrcCls">
					<el-select v-model='addPeriodMeasureDialogForm.PeriodMeasure_MeasurePurpose'>
						<el-option label='MR' value='1'></el-option>
						<el-option label='ANR' value='2'></el-option>
					</el-select>
				</el-form-item>
                <el-form-item prop='PeriodMeasure_ReportInterval' style="width:40%;min-width:400px;" label="Report Interval" label-width="160px" class="validate-item">
					<el-select v-model='addPeriodMeasureDialogForm.PeriodMeasure_ReportInterval'>
						<el-option v-for="item in ReportIntervalList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
                <el-form-item prop='PeriodMeasure_ReportAmount' style="width:40%;min-width:400px;" label="Report Amount" label-width="160px" class="validate-item">
					<el-select v-model='addPeriodMeasureDialogForm.PeriodMeasure_ReportAmount'>
						<el-option v-for="item in ReportAmountList" :label='item' :value='item'></el-option>
					</el-select>
				</el-form-item>
			</el-form> 
		</div>
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addPeriodMeasureDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- Xn List 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addXnDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addXnDialogForm" :model='addXnDialogForm' :rules='addXnDialogRules' label-position="top">     		     			            
			<el-form-item prop='Xn_PLMNID' style="width:40%;min-width:400px;" label="PLMNID" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addXnDialogForm.Xn_PLMNID'>
					<template slot="append">Length：5~6 Digit,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='Xn_RemoteAddress' style="width:40%;min-width:400px;" label="RemoteAddress" label-width="160px" class='validate-item'>
				<el-input v-model.trim='addXnDialogForm.Xn_RemoteAddress'>
					<template slot="append">Example：1.1.1.1</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='Xn_LinkEnable' style="width:40%;min-width:400px;" label="XnLinkEnable" label-width="160px">
				<el-switch v-model="addXnDialogForm.Xn_LinkEnable" active-value="1" inactive-value="0"></el-switch>
			</el-form-item>
			<el-form-item prop='Xn_HoEnable' style="width:40%;min-width:400px;" label="XnHoEnable" label-width="160px">
				<el-switch v-model="addXnDialogForm.Xn_HoEnable" active-value="1" inactive-value="0"></el-switch>
			</el-form-item>
		</el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addXnDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- DLBWP新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addDLBWPDialogShow" @close="ranTablecloseAddDialog" :close-on-click-modal="false" append-to-body>		
		<el-form ref="addDLBWPDialogForm" :model='addDLBWPDialogForm' :rules='addDLBWPDialogRules' label-position="top">   
            <el-form-item prop='DLBWP_DIBwpID' style="min-width:400px;" label="DIBwp ID" label-width="160px">
				<el-select v-model='addDLBWPDialogForm.DLBWP_DIBwpID' style="width:60px;padding-top:5px;">
					<el-option label='0' value='0'></el-option>
					<el-option label='1' value='1'></el-option>
                    <el-option label='2' value='2'></el-option>
					<el-option label='3' value='3'></el-option>
				</el-select>
			</el-form-item>  
            <el-form-item prop='DLBWP_StartPrbPosition' style="min-width:400px;" label="StartPrbPosition" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addDLBWPDialogForm.DLBWP_StartPrbPosition'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~273,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='DLBWP_BandWidth' style="min-width:400px;" label="BandWidth" label-width="160px">
				<el-select v-model='addDLBWPDialogForm.DLBWP_BandWidth' style="width:60px;padding-top:5px;">
					<el-option v-for="item in BandWidthList" :label='item' :value='item'></el-option>
				</el-select>
			</el-form-item>
            <el-form-item prop='DLBWP_SubcarrierSpacing' style="min-width:400px;" label="SubcarrierSpacing" label-width="160px">
				<el-select v-model='addDLBWPDialogForm.DLBWP_SubcarrierSpacing'>
					<el-option label='30KHz' value='1'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='DLBWP_CyclicPrefix' style="min-width:400px;" label="CyclicPrefix" label-width="160px">
				<el-select v-model='addDLBWPDialogForm.DLBWP_CyclicPrefix'>
					<el-option label='normal' value='normal'></el-option>
					<el-option label='extended' value='extended'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='DLBWP_PMaxULPower' style="min-width:400px;" label="PMaxULPower" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addDLBWPDialogForm.DLBWP_PMaxULPower'>
                    <template slot="append"><%=rb.getString("FanWei")%>：-30~30,Integer</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='DLBWP_InitDlMcs' style="min-width:400px;" label="InitDIMcs" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addDLBWPDialogForm.DLBWP_InitDlMcs'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~28,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='DLBWP_MaxDlUeToBeScheduleInSlot' style="min-width:400px;" label="MaxDlUeToBeScheduleInSlot" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addDLBWPDialogForm.DLBWP_MaxDlUeToBeScheduleInSlot'>
                    <template slot="append"><%=rb.getString("FanWei")%>：1~8,Integer</template>
                </el-input>
            </el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="ranTableAddDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addDLBWPDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gnbRanPage = new Vue({
	el: '#gnbRanPage', 
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
						if(reg.test(value)){
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
				if(value === ''){
					callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
				}else{
					if(reg.test(value)){
						callback();
					}else{
						callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
					}
				} 
			},
			validateIPaddress= (rule,value,callback) => {
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
				
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
				}else{
					if(vm.isValidIP(value) || vm.isIPv6(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
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
			validateBitmapRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
				if(value == '' || value == undefined || value == null){
					callback(new Error(mag))
				}else{
					if(reg.test(value) && this.isLessThan(value,max) && this.isLessThan(min,value)){
						callback()
					}else{
						callback(new Error(mag))
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
			validateBlackPhysCellIdStart = (rule,value,callback)=>{
				var min = 0,
					max = 503,
					mag = '<%=rb.getString("FanWei")%>：0~503 or 268435455(268435455:OFF),Integer',
					reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
				if(this.tbType == 'NRIRS'){
					max = 1007;
					mag = '<%=rb.getString("FanWei")%>：0~1007 or 268435455(268435455:OFF),Integer'
				}
				if(value == '' || value == undefined || value == null){
					callback(new Error(mag))
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						if(value == 268435455){
							callback();
						}else{
							callback(new Error(mag))
						}
					}
				}
			},
			validateQuantity = (rule,value,callback)=>{
				if(value.length === 0){
					callback('<%=rb.getString("QingXuanZe")%>')
				}else{
					callback();
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
			activeCollapse:['GNB','SSB','MobilityStrategy','SIB','PLMN','CELL','Neighbor','QOS','Mobility','Xn','Black','ANR','BWP'],
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
				NRCellList:[],
                LTENFList:[],
				LTEERList:[],
				LTENCList:[],
				NRNFList:[],
				NRIRSList:[],
				NRNCList:[],
				QOSList:[],
				SSTList:[],
				A1List:[],
				A2List:[],
				A3List:[],
				A4List:[],
				A5List:[],
				B1List:[],
				B2List:[],
                PeriodMeasureList:[],
				XnList:[],
				XnBlacklist:[],
				DLBWPList:[],

				multiPlmnEnable:'0',

				gnbLength:'',
				gnbName:'',
				gnbId:'',
				adminState:'1',

				SSB_SSBAbsoluteFreq:'',
				Mobility_NrToLteMigrateStgy:'0',

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
				PowerMaxLimit:40,
                PowerMinLimit:0,
				offsetToPointA:'',
				ssbSubCarrierOffset:'',
				// ssbGSCN:'',

				ANR_Enable:'0',
				ANR_InterFeqEnable:'0',
				ANR_EUTRANEnable:'0',
				ANR_BiNRCellEnable:'0',
				ANR_MRTriggerType:'0',
				ANR_AbsoluteThreshold:'',
				ANR_RelativeThreshold:'',
				ANR_AbsEnable:'0',
				ANR_KpiPeriod:'',
				ANR_AutoAdjustEnable:'0',
				ANR_AutoRemoveEnable:'0',
				ANR_AutoRemovePeriod:'',
				ANR_AutoRemoveMaxCell:'',
				ANR_MaxHOtimes:'',
				ANR_MaxHOSuccess:'',

                SIB1_QRxLevMinSIB1:'',
                SIB1_QQualMinOffset:'1',
                SIB1_QRxLevMinOffset:'1',
                SIB1_QQualMinSIB1:'',
                SIB2_Enable:'0',
                SIB2_Qhyst:'',
                SIB2_QRxLevMinSIB2:'',
                SIB2_SIntraSearchP:'',
                SIB2_TReselectionNR:'0',
                SIB2_CellReselectionPriority:'0',
                SIB2_ThreshServingLowP:'',
                SIB2_DeriveSSBIndexFromCell:'0',
                SIB2_SNonIntraSearchP:'0',
                SIB2_SNonIntraSearchQ:'0',
                SIB3_Enable:'0',
                SIB4_Enable:'0',
                SIB5_Enable:'0',
                SIB9_Enable:'0',
                SIB9_DaylightSavingTime:'',
                SIB9_LeapSecond:''
			},
			rules:{
				gnbLength:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:22,max:32,isRequired:true,mag:'<%=rb.getString("FanWei")%>：22~32,Integer'}
				],
				gnbId:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:4294967295,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~4294967295,Integer'}
				],
				SSB_SSBAbsoluteFreq:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:3279165,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~3279165,Integer'}
				],
				band:[
					{validator:validateBandRange,min:1,max:1000,isRequired:false,mag:'<%=rb.getString("FanWei")%>：1~1000,Integer'}
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
				// ssbGSCN:[
				// 	{validator:validateRange,min:0,max:31,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~31,Integer'}
				// ],

				ANR_AbsoluteThreshold:[{validator:validateRange,min:0,max:127,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~127,Integer'}],
				ANR_RelativeThreshold:[{validator:validateRange,min:0,max:127,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~127,Integer'}],
				ANR_KpiPeriod:[{validator:validateRange,min:0,max:3279165,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer'}],
				ANR_AutoRemovePeriod:[{validator:validateRange,min:0,max:3279165,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer'}],
				ANR_AutoRemoveMaxCell:[{validator:validateRange,min:0,max:65535,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer'}],
				ANR_MaxHOtimes:[{validator:validateRange,min:0,max:3279165,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer'}],
				ANR_MaxHOSuccess:[{validator:validateRange,min:0,max:100,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~100,Integer'}],

                SIB1_QRxLevMinSIB1:[{validator:validateRange,min:-70,max:-22,isRequired:true,mag:'<%=rb.getString("FanWei")%>：(-70)~(-22),Integer'}],
                SIB1_QQualMinSIB1:[{validator:validateRange,min:-43,max:-12,isRequired:true,mag:'<%=rb.getString("FanWei")%>：(-43)~(-12),Integer'}],
                SIB2_Qhyst:[{validator:validateRange,min:0,max:15,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~15,Integer'}],
                SIB2_QRxLevMinSIB2:[{validator:validateRange,min:-70,max:-22,isRequired:true,mag:'<%=rb.getString("FanWei")%>：(-70)~(-22),Integer'}],
                SIB2_SIntraSearchP:[{validator:validateRange,min:0,max:31,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~31,Integer'}],
                SIB2_ThreshServingLowP:[{validator:validateRange,min:0,max:31,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~31,Integer'}],
                SIB9_DaylightSavingTime:[{validator:validateRange,min:0,max:2,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~2,Integer'}],
                SIB9_LeapSecond:[{validator:validateRange,min:-127,max:128,isRequired:true,mag:'<%=rb.getString("FanWei")%>：-127~128,Integer'}],
			},
			casts:{},
			optType:'',
			tbType:'',
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
            addLTENFDialogShow:false,
			addLTENFDialogForm:{
				LTENF_CarrierFreq:'',
                LTENF_AllowdMeasBandWidth:'',
                LTENF_PresAntennaPort1:'',
                LTENF_QOffset:'',
                LTENF_WideBandRsrqMeas:'',
                LTENF_CellReselectionPriority:'',
                LTENF_ThreshXHigh:'',
                LTENF_ThreshXLow:'',
                LTENF_QRxLevMin:'',
                LTENF_QQualMin:'',
                LTENF_PMaxEUTRA:'',
			},
			addLTENFDialogRules:{
				LTENF_CarrierFreq:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',isRequired:true}],
                LTENF_CellReselectionPriority:[{validator:validateRange,min:0,max:7,mag:'<%=rb.getString("FanWei")%>：0~7,Integer',isRequired:true}],
                LTENF_ThreshXHigh:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
                LTENF_ThreshXLow:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
                LTENF_QRxLevMin:[{validator:validateRange,min:-70,max:-22,mag:'<%=rb.getString("FanWei")%>：-70~-22,Integer',isRequired:true}],
                LTENF_QQualMin:[{validator:validateRange,min:-34,max:-3,mag:'<%=rb.getString("FanWei")%>：-34~-3,Integer',isRequired:true}],
                LTENF_PMaxEUTRA:[{validator:validateRange,min:-30,max:33,mag:'<%=rb.getString("FanWei")%>：-30~33,Integer',isRequired:true}],
                LTENF_AllowdMeasBandWidth:[{required:true,message:'required',trigger:'change'}],
                LTENF_PresAntennaPort1:[{required:true,message:'required',trigger:'change'}],
                LTENF_QOffset:[{required:true,message:'required',trigger:'change'}],
                LTENF_WideBandRsrqMeas:[{required:true,message:'required',trigger:'change'}],
			},
			addLTEERDialogShow:false,
			addLTEERDialogForm:{
				LTEER_EUTRACarrierARFCN:'',
				LTEER_TReselectionEUTRA:'1',
				LTEER_CellReselectionPriority:'7',
				LTEER_ThreshXHigh:'',
				LTEER_ThreshXLow:'',
				LTEER_QRxLevMin:'',
                LTEER_QQualMin:'',
                LTEER_PMaxEUTRA:'',
				LTEER_ThreshXHighQ:'',
                LTEER_AllowdMeasBandWidth:'mbw50',
                LTEER_PresAntennaPort1:'0',
				LTEER_BlackPhysCellIdStart:'',
				LTEER_BlackPhysCellIdRange:'0',
			},
			addLTEERDialogRules:{
				LTEER_EUTRACarrierARFCN:[{validator:validateRange,min:0,max:65535,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer',isRequired:true}],
				LTEER_ThreshXHigh:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
                LTEER_ThreshXLow:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
                LTEER_QRxLevMin:[{validator:validateRange,min:-70,max:-22,mag:'<%=rb.getString("FanWei")%>：-70~-22,Integer',isRequired:true}],
                LTEER_QQualMin:[{validator:validateRange,min:-34,max:-3,mag:'<%=rb.getString("FanWei")%>：-34~-3,Integer',isRequired:true}],
                LTEER_PMaxEUTRA:[{validator:validateRange,min:-30,max:33,mag:'<%=rb.getString("FanWei")%>：-30~33,Integer',isRequired:true}],
				LTEER_ThreshXHighQ:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
				LTEER_BlackPhysCellIdStart:[{required:true,trigger:'blur'},{validator:validateBlackPhysCellIdStart,trigger:'blur'}],
			},
			addLTENCDialogShow:false,
			addLTENCDialogForm:{
				LTENC_PLMNID:'',
				LTENC_CID:'',
				LTENC_EUTRACarrierARFCN:'',
				LTENC_PhyCellID:'',
				LTENC_QOffset:'',
				LTENC_QRxLevMinOffsetCell:'',
				LTENC_QQualMinOffsetCell:'',
				LTENC_CIO:'',
				LTENC_Blacklisted:'',
				LTENC_TAC:'',
				LTENC_eNBType:'',
				LTENC_eNBID:'',
				LTENC_noRemove:'',
			},
			addLTENCDialogRules:{
				LTENC_PLMNID:[{required:true,trigger:'blur'},{validator:validatePLMNID,trigger:'blur'}],
				LTENC_CID:[{validator:validateRange,min:1,max:268435455,mag:'<%=rb.getString("FanWei")%>：1~268435455,Integer',isRequired:true}],
				LTENC_EUTRACarrierARFCN:[{validator:validateRange,min:0,max:65535,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer',isRequired:true}],
				LTENC_PhyCellID:[{validator:validateRange,min:0,max:503,mag:'<%=rb.getString("FanWei")%>：0~503,Integer',isRequired:true}],
				LTENC_CIO:[{validator:validateRange,min:-24,max:24,mag:'<%=rb.getString("FanWei")%>：-24~24,Integer',isRequired:true}],
				LTENC_TAC:[{validator:validateRange,min:0,max:65535,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer',isRequired:true}],
				LTENC_eNBID:[{validator:validateRange,min:0,max:1048575,mag:'<%=rb.getString("FanWei")%>：0~1048575,Integer',isRequired:true}],
				LTENC_QRxLevMinOffsetCell:[{required:true,message:'required',trigger:'change'}],
				LTENC_QQualMinOffsetCell:[{required:true,message:'required',trigger:'change'}],
				LTENC_eNBType:[{required:true,message:'required',trigger:'change'}],
				LTENC_Blacklisted:[{required:true,message:'required',trigger:'change'}],
				LTENC_QOffset:[{required:true,message:'required',trigger:'change'}],
				LTENC_noRemove:[{required:true,message:'required',trigger:'change'}],
			},
			addNRNFDialogShow:false,
			addNRNFDialogForm:{
				NRNF_Enable:'false',
				NRNF_SSBFrequency:'',
				NRNF_SubCarrierSpacing:'',
				NRNF_SmtcPeriodicity:'',
				NRNF_SmtcOffset:'',
				NRNF_SmtcDuration:'',
				NRNF_SSBlocksConsolidationRsrp:'',
				NRNF_SSBlocksConsolidationRsrq:'',
				NRNF_SSBlocksConsolidationSinr:'',
				NRNF_NrofSSBlocksToAverage:'',
				NRNF_RsrpOffsetSSB:'',
				NRNF_RsrqOffsetSSB:'',
				NRNF_SinrOffsetSSB:'',
				NRNF_RsrpOffsetCsiRs:'',
				NRNF_RsrqOffsetCsiRs:'',
				NRNF_SinrOffsetCsiRs:'',
				NRNF_BitmapType:'',
				NRNF_Bitmap:'',
				NRNF_DeriveSSBIndexFromCell:'',
				NRNF_FreqBandIndicatorNR:'',
				NRNF_OffsetToPointA:'',
				NRNF_SSBSubCarrierOffset:'',
			},
			addNRNFDialogRules:{
				NRNF_SSBFrequency:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',isRequired:true}],
				NRNF_SmtcOffset:[{validator:validateRange,min:0,max:159,mag:'<%=rb.getString("FanWei")%>：0~159,Integer',isRequired:true}],
				NRNF_SSBlocksConsolidationRsrp:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				NRNF_SSBlocksConsolidationRsrq:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				NRNF_SSBlocksConsolidationSinr:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				NRNF_NrofSSBlocksToAverage:[{validator:validateRange,min:2,max:16,mag:'<%=rb.getString("FanWei")%>：2~16,Integer',isRequired:true}],
				NRNF_RsrpOffsetSSB:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				NRNF_RsrqOffsetSSB:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				NRNF_SinrOffsetSSB:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				NRNF_RsrpOffsetCsiRs:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				NRNF_RsrqOffsetCsiRs:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				NRNF_SinrOffsetCsiRs:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				NRNF_Bitmap:[{validator:validateBitmapRange,min:'0',max:'18446744073709551615',mag:'<%=rb.getString("FanWei")%>：0~18446744073709551615,Integer',isRequired:true}],
				NRNF_FreqBandIndicatorNR:[{validator:validateRange,min:1,max:1024,mag:'<%=rb.getString("FanWei")%>：1~1024,Integer',isRequired:true}],
				NRNF_OffsetToPointA:[{validator:validateRange,min:0,max:2199,mag:'<%=rb.getString("FanWei")%>：0~2199,Integer',isRequired:true}],
				NRNF_SSBSubCarrierOffset:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
				NRNF_SubCarrierSpacing:[{required:true,message:'required',trigger:'change'}],
				NRNF_SmtcPeriodicity:[{required:true,message:'required',trigger:'change'}],
				NRNF_SmtcDuration:[{required:true,message:'required',trigger:'change'}],
				NRNF_BitmapType:[{required:true,message:'required',trigger:'change'}],
				NRNF_DeriveSSBIndexFromCell:[{required:true,message:'required',trigger:'change'}],
			},
			addNRIRSDialogShow:false,
			addNRIRSDialogForm:{
				NRIRS_CarrierFreq:'',
				NRIRS_NrofSSBlocksToAverage:'',
				NRIRS_ThresholdRSRP:'',
				NRIRS_ThresholdRSRQ:'',
				NRIRS_ThresholdSINR:'',
				NRIRS_SubCarrierSpacing:'1',
				NRIRS_DeriveSSBIndexFromCell:'0',
				NRIRS_QRxLevMin:'',
				NRIRS_QQualMin:'',
				NRIRS_PMax:'',
				NRIRS_TReselectionNR:'0',
				NRIRS_ThreshXHighP:'',
				NRIRS_ThreshXLowP:'',
				NRIRS_ThreshXHighQ:'',
				NRIRS_ThreshXLowQ:'',
				NRIRS_CellReselectionPriority:'7',
				NRIRS_CellReselectionSubPriority:'0',
				NRIRS_QOffsetFreq:'',
				NRIRS_BlackPhyCellIdStart:'',
				NRIRS_BlackPhyCellIdRange:'0',
			},
			addNRIRSDialogRules:{
				NRIRS_CarrierFreq:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',isRequired:true}],
				NRIRS_NrofSSBlocksToAverage:[{validator:validateRange,min:2,max:16,mag:'<%=rb.getString("FanWei")%>：2~16,Integer',isRequired:false}],
				NRIRS_ThresholdRSRP:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				NRIRS_ThresholdRSRQ:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				NRIRS_ThresholdSINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				NRIRS_SubCarrierSpacing:[{required:true,message:'required',trigger:'change'}],
				NRIRS_DeriveSSBIndexFromCell:[{required:true,message:'required',trigger:'change'}],
				NRIRS_QRxLevMin:[{validator:validateRange,min:-70,max:-22,mag:'<%=rb.getString("FanWei")%>：-70~-22,Integer',isRequired:true}],
				NRIRS_QQualMin:[{validator:validateRange,min:-43,max:-12,mag:'<%=rb.getString("FanWei")%>：-43~-12,Integer',isRequired:true}],
				NRIRS_PMax:[{validator:validateRange,min:-30,max:33,mag:'<%=rb.getString("FanWei")%>：-30~33,Integer',isRequired:true}],
				NRIRS_ThreshXHighP:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
				NRIRS_ThreshXLowP:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
				NRIRS_ThreshXHighQ:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
				NRIRS_ThreshXLowQ:[{validator:validateRange,min:0,max:31,mag:'<%=rb.getString("FanWei")%>：0~31,Integer',isRequired:true}],
				NRIRS_QOffsetFreq:[{validator:validateRange,min:-24,max:24,mag:'<%=rb.getString("FanWei")%>：-24~24,Integer',isRequired:true}],
				NRIRS_BlackPhyCellIdStart:[{required:true,trigger:'blur'},{validator:validateBlackPhysCellIdStart,trigger:'blur'}],
			},
			addNRNCDialogShow:false,
			addNRNCDialogForm:{
				NRNC_PLMNID:'',
				NRNC_CID:'',
				NRNC_NRCarrierARFCN:'',
				NRNC_SSBFrequency:'',
				NRNC_SSBSubcarrierSpacing:'',
				NRNC_PhyCellID:'',
				NRNC_QOffset:'',
				NRNC_QRxLevMinOffsetCell:'',
				NRNC_QQualMinOffsetCell:'',
				NRNC_CIO:'',
				NRNC_Blacklisted:'',
				NRNC_TAC:'',
				NRNC_noRemove:'',
				NRNC_gnbIdLength:''
			},
			addNRNCDialogRules:{
				NRNC_PLMNID:[{validator:validatePLMNID,trigger:'blur'}],
				NRNC_CID:[{validator:validateRange,min:1,max:68719476735,mag:'<%=rb.getString("FanWei")%>：1~68719476735,Integer',isRequired:true}],
				NRNC_NRCarrierARFCN:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',isRequired:true}],
				NRNC_SSBFrequency:[{validator:validateRange,min:0,max:3279165,mag:'<%=rb.getString("FanWei")%>：0~3279165,Integer',isRequired:true}],
				NRNC_PhyCellID:[{validator:validateRange,min:0,max:1007,mag:'<%=rb.getString("FanWei")%>：0~1007,Integer',isRequired:true}],
				NRNC_TAC:[{validator:validateRange,min:0,max:16777215,mag:'<%=rb.getString("FanWei")%>：0~16777215,Integer',isRequired:true}],
				NRNC_CIO:[{required:true,message:'required',trigger:'change'}],
				NRNC_SSBSubcarrierSpacing:[{required:true,message:'required',trigger:'change'}],
				NRNC_QOffset:[{required:true,message:'required',trigger:'change'}],
				NRNC_QRxLevMinOffsetCell:[{required:true,message:'required',trigger:'change'}],
				NRNC_QQualMinOffsetCell:[{required:true,message:'required',trigger:'change'}],
				NRNC_Blacklisted:[{required:true,message:'required',trigger:'change'}],
				NRNC_noRemove:[{required:true,message:'required',trigger:'change'}],
				NRNC_gnbIdLength:[{validator:validateRange,min:22,max:32,mag:'<%=rb.getString("FanWei")%>：0~16777215,Integer',isRequired:true}],
			},
			addQOSDialogShow:false,
			addQOSDialogForm:{
				QOS_Enable:'1',
				QOS_MappingDrbIndex:'',
				QOS_5QI:'',
				QOS_Type:'0',
				QOS_Priority:'',
				QOS_MinBr:'',
				QOS_IsDefault:'0',
				QOS_UeInactivityTimerConfig:'',
				QOS_TReorderingPdcp:'',
				QOS_TReorderingUE:'',
				QOS_DiscardTimer:'0',
				QOS_StatusReportRequired:'1',
				QOS_PdcpSnSizeUL:'1',
				QOS_PdcpSnSizeDL:'1',
				QOS_Dscp:'',
				QOS_RlcMode:'2',
				QOS_SnFieldLengthAmDL:'',
				QOS_SnFieldLengthAmUL:'',
				QOS_SnFieldLengthUmDL:'',
				QOS_SnFieldLengthUmUL:'',
				QOS_ULConfig:'1',
				QOS_EnableRohc:'0',
				QOS_RohcProfile0x0001:'0',
				QOS_RohcProfile0x0002:'0',
				QOS_RohcProfile0x0006:'0',
				QOS_PdcpDuplicationActivated:'0',
				QOS_PrimaryPathDL:'0',
				QOS_PrimaryPath:'0',
				QOS_ULDataSplitThreshold:'0',
				QOS_DLDataSplitThreshold:'0',
				QOS_AllowedIntegrityAlgo:'0',
				QOS_LongDrxCycle:'0',
				QOS_ShortDrxCycle:'5',
				QOS_ShortDrxCycleTimer:'4',
				QOS_DrBlnactivityTimerConfig:''
			},
			addQOSDialogRules:{
				QOS_MappingDrbIndex:[{validator:validateRange,min:5,max:32,mag:'<%=rb.getString("FanWei")%>：5~32,Integer',isRequired:true}],
				QOS_5QI:[{validator:validateRange,min:1,max:255,mag:'<%=rb.getString("FanWei")%>：1~255,Integer',isRequired:true}],
				QOS_Priority:[{validator:validateRange,min:1,max:16,mag:'<%=rb.getString("FanWei")%>：1~16,Integer',isRequired:true}],
				QOS_MinBr:[{validator:validateRange,min:0,max:3999999999,mag:'<%=rb.getString("FanWei")%>：0~3999999999,Integer',isRequired:true}],
				QOS_UeInactivityTimerConfig:[{validator:validateRange,min:0,max:4294967295,mag:'<%=rb.getString("FanWei")%>：0~4294967295,Integer',isRequired:true}],
				QOS_TReorderingPdcp:[{validator:validateRange,min:0,max:35,mag:'<%=rb.getString("FanWei")%>：0~35,Integer',isRequired:true}],
				QOS_TReorderingUE:[{validator:validateRange,min:0,max:35,mag:'<%=rb.getString("FanWei")%>：0~35,Integer',isRequired:true}],
				QOS_Dscp:[{validator:validateRange,min:0,max:63,mag:'<%=rb.getString("FanWei")%>：0~63,Integer',isRequired:true}],
				QOS_SnFieldLengthAmDL:[{validator:validateRange,min:0,max:255,mag:'<%=rb.getString("FanWei")%>：0~255,Integer',isRequired:true}],
				QOS_SnFieldLengthAmUL:[{validator:validateRange,min:0,max:255,mag:'<%=rb.getString("FanWei")%>：0~255,Integer',isRequired:true}],
				QOS_SnFieldLengthUmDL:[{validator:validateRange,min:0,max:255,mag:'<%=rb.getString("FanWei")%>：0~255,Integer',isRequired:true}],
				QOS_SnFieldLengthUmUL:[{validator:validateRange,min:0,max:255,mag:'<%=rb.getString("FanWei")%>：0~255,Integer',isRequired:true}],
				QOS_DrBlnactivityTimerConfig:[{validator:validateRange,min:0,max:4294967295,mag:'<%=rb.getString("FanWei")%>：0~4294967295,Integer',isRequired:true}],
			},
			addSSTDialogShow:false,
			addSSTDialogForm:{
				SST_Sst:'',
				SST_SstResourceType:'0',
				SST_MaxResourceReserved:'',
				SST_MinResourceReserved:'',
			},
			addSSTDialogRules:{
                SST_Sst:[{validator:validateRange,min:0,max:255,mag:'<%=rb.getString("FanWei")%>：0~255,Integer',isRequired:true}],
				SST_MaxResourceReserved:[{validator:validateRange,min:0,max:273,mag:'<%=rb.getString("FanWei")%>：0~273,Integer',isRequired:true}],
				SST_MinResourceReserved:[{validator:validateRange,min:0,max:273,mag:'<%=rb.getString("FanWei")%>：0~273,Integer',isRequired:true}],
			},
			addA1DialogShow:false,
			addA1DialogForm:{
				A1_Enable:'1',
				A1_ThresholdTriggerType:'0',
				A1_ThresholdRSRP:'',
				A1_ThresholdRSRQ:'',
				A1_ThresholdSINR:'',
				A1_ReportOnLeave:'1',
				A1_Hysteresis:'',
				A1_MaxReportCells:'',
				A1_MeasurePurpose:'1',
				A1_ReportAmount:'4',
				A1_MaxNrofRSIndexToReport:'',
				A1_ReportInterval:'5120',
				A1_ReportQuantity:['rsrp','rsrq','sinr'],
				A1_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
				A1_TimeToTrigger:'480',
				A1_RsType:'ssb',
				A1_IncludeBeamMeasurements:'0',
				A1_PLMN:'',
			},
			addA1DialogRules:{
				A1_ThresholdRSRP:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A1_ThresholdRSRQ:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A1_ThresholdSINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A1_Hysteresis:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				A1_MaxReportCells:[{validator:validateRange,min:1,max:8,mag:'<%=rb.getString("FanWei")%>：1~8,Integer',isRequired:true}],
				A1_MaxNrofRSIndexToReport:[{validator:validateRange,min:1,max:32,mag:'<%=rb.getString("FanWei")%>：1~32,Integer',isRequired:true}],
				A1_PLMN:[{required:true,trigger:'blur'},{validator:validatePLMNID,trigger:'blur'}],
				A1_ReportQuantity:[{validator:validateQuantity}],
				A1_RptQuantityRsIndex:[{validator:validateQuantity}],
			},
			addA2DialogShow:false,
			addA2DialogForm:{
				A2_Enable:'1',
				A2_ThresholdTriggerType:'0',
				A2_ThresholdRSRP:'',
				A2_ThresholdRSRQ:'',
				A2_ThresholdSINR:'',
				A2_ReportOnLeave:'1',
				A2_Hysteresis:'',
				A2_MaxReportCells:'',
				A2_MeasurePurpose:'1',
				A2_ReportAmount:'4',
				A2_MaxNrofRSIndexToReport:'',
				A2_ReportInterval:'5120',
				A2_ReportQuantity:['rsrp','rsrq','sinr'],
				A2_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
				A2_TimeToTrigger:'480',
				A2_RsType:'ssb',
				A2_IncludeBeamMeasurements:'0',
				A2_PLMN:'',
			},
			addA2DialogRules:{
				A2_ThresholdRSRP:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A2_ThresholdRSRQ:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A2_ThresholdSINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A2_Hysteresis:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				A2_MaxReportCells:[{validator:validateRange,min:1,max:8,mag:'<%=rb.getString("FanWei")%>：1~8,Integer',isRequired:true}],
				A2_MaxNrofRSIndexToReport:[{validator:validateRange,min:1,max:32,mag:'<%=rb.getString("FanWei")%>：1~32,Integer',isRequired:true}],
				A2_PLMN:[{required:true,trigger:'blur'},{validator:validatePLMNID,trigger:'blur'}],
				A2_ReportQuantity:[{validator:validateQuantity}],
				A2_RptQuantityRsIndex:[{validator:validateQuantity}],
			},
			addA3DialogShow:false,
			addA3DialogForm:{
				A3_Enable:'1',
				A3_ThresholdTriggerType:'0',
				A3_OffsetRSRP:'',
				A3_OffsetRSRQ:'',
				A3_OffsetSINR:'',
				A3_ReportOnLeave:'1',
				A3_Hysteresis:'',
				A3_MaxReportCells:'',
				A3_MeasurePurpose:'1',
				A3_ReportAmount:'4',
				A3_MaxNrofRSIndexToReport:'',
				A3_ReportInterval:'5120',
				A3_ReportQuantity:['rsrp','rsrq','sinr'],
				A3_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
				A3_TimeToTrigger:'480',
				A3_UseWhiteCellList:'0',
				A3_RsType:'ssb',
				A3_IncludeBeamMeasurements:'0',
				A3_PLMN:'',
			},
			addA3DialogRules:{
				A3_OffsetRSRP:[{validator:validateRange,min:-30,max:30,mag:'<%=rb.getString("FanWei")%>：-30~30,Integer',isRequired:true}],
				A3_OffsetRSRQ:[{validator:validateRange,min:-30,max:30,mag:'<%=rb.getString("FanWei")%>：-30~30,Integer',isRequired:true}],
				A3_OffsetSINR:[{validator:validateRange,min:-30,max:30,mag:'<%=rb.getString("FanWei")%>：-30~30,Integer',isRequired:true}],
				A3_Hysteresis:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				A3_MaxReportCells:[{validator:validateRange,min:1,max:8,mag:'<%=rb.getString("FanWei")%>：1~8,Integer',isRequired:true}],
				A3_MaxNrofRSIndexToReport:[{validator:validateRange,min:1,max:32,mag:'<%=rb.getString("FanWei")%>：1~32,Integer',isRequired:true}],
				A3_PLMN:[{required:true,trigger:'blur'},{validator:validatePLMNID,trigger:'blur'}],
				A3_ReportQuantity:[{validator:validateQuantity}],
				A3_RptQuantityRsIndex:[{validator:validateQuantity}],
			},
			addA4DialogShow:false,
			addA4DialogForm:{
				A4_Enable:'1',
				A4_ThresholdTriggerType:'0',
				A4_ThresholdRSRP:'',
				A4_ThresholdRSRQ:'',
				A4_ThresholdSINR:'',
				A4_ReportOnLeave:'1',
				A4_Hysteresis:'',
				A4_MaxReportCells:'',
				A4_MeasurePurpose:'1',
				A4_ReportAmount:'4',
				A4_MaxNrofRSIndexToReport:'',
				A4_ReportInterval:'5120',
				A4_ReportQuantity:['rsrp','rsrq','sinr'],
				A4_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
				A4_TimeToTrigger:'480',
				A4_UseWhiteCellList:'0',
				A4_RsType:'ssb',
				A4_IncludeBeamMeasurements:'0',
				A4_PLMN:'',
			},
			addA4DialogRules:{
				A4_ThresholdRSRP:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A4_ThresholdRSRQ:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A4_ThresholdSINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A4_Hysteresis:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				A4_MaxReportCells:[{validator:validateRange,min:1,max:8,mag:'<%=rb.getString("FanWei")%>：1~8,Integer',isRequired:true}],
				A4_MaxNrofRSIndexToReport:[{validator:validateRange,min:1,max:32,mag:'<%=rb.getString("FanWei")%>：1~32,Integer',isRequired:true}],
				A4_PLMN:[{required:true,trigger:'blur'},{validator:validatePLMNID,trigger:'blur'}],
				A4_ReportQuantity:[{validator:validateQuantity}],
				A4_RptQuantityRsIndex:[{validator:validateQuantity}],
			},
			addA5DialogShow:false,
			addA5DialogForm:{
				A5_Enable:'1',
				A5_ThresholdTriggerType:'0',
				A5_Threshold1RSRP:'',
				A5_Threshold1RSRQ:'',
				A5_Threshold1SINR:'',
				A5_Threshold2TriggerType:'0',
				A5_Threshold2RSRP:'',
				A5_Threshold2RSRQ:'',
				A5_Threshold2SINR:'',
				A5_ReportOnLeave:'1',
				A5_Hysteresis:'',
				A5_MaxReportCells:'',
				A5_MeasurePurpose:'1',
				A5_ReportAmount:'4',
				A5_MaxNrofRSIndexToReport:'',
				A5_ReportInterval:'5120',
				A5_ReportQuantity:['rsrp','rsrq','sinr'],
				A5_RptQuantityRsIndex:['rsrp','rsrq','sinr'],
				A5_TimeToTrigger:'480',
				A5_UseWhiteCellList:'0',
				A5_RsType:'ssb',
				A5_IncludeBeamMeasurements:'0',
				A5_PLMN:'',
			},
			addA5DialogRules:{
				A5_Threshold1RSRP:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A5_Threshold1RSRQ:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A5_Threshold1SINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A5_Threshold2RSRP:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A5_Threshold2RSRQ:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A5_Threshold2SINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				A5_Hysteresis:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				A5_MaxReportCells:[{validator:validateRange,min:1,max:8,mag:'<%=rb.getString("FanWei")%>：1~8,Integer',isRequired:true}],
				A5_MaxNrofRSIndexToReport:[{validator:validateRange,min:1,max:32,mag:'<%=rb.getString("FanWei")%>：1~32,Integer',isRequired:true}],
				A5_PLMN:[{required:true,trigger:'blur'},{validator:validatePLMNID,trigger:'blur'}],
				A5_ReportQuantity:[{validator:validateQuantity}],
				A5_RptQuantityRsIndex:[{validator:validateQuantity}],
			},
			addB1DialogShow:false,
			addB1DialogForm:{
				B1_Enable:'1',
				B1_ThresholdTriggerType:'0',
				B1_Threshold1EUTRARSRP:'',
				B1_Threshold1EUTRARSRQ:'',
				B1_Threshold1EUTRASINR:'',
				B1_Hysteresis:'',
				B1_MaxReportCells:'',
				B1_MeasurePurpose:'1',
				B1_ReportAmount:'4',
				B1_ReportInterval:'5120',
				B1_TimeToTrigger:'480',
				B1_ReportQuantity:['rsrp','rsrq','sinr'],
				B1_ReportOnLeave:'1',
				B1_PLMN:'',
			},
			addB1DialogRules:{
				B1_Threshold1EUTRARSRP:[{validator:validateRange,min:0,max:97,mag:'<%=rb.getString("FanWei")%>：0~97,Integer',isRequired:true}],
				B1_Threshold1EUTRARSRQ:[{validator:validateRange,min:0,max:34,mag:'<%=rb.getString("FanWei")%>：0~34,Integer',isRequired:true}],
				B1_Threshold1EUTRASINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				B1_Hysteresis:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				B1_MaxReportCells:[{validator:validateRange,min:1,max:8,mag:'<%=rb.getString("FanWei")%>：1~8,Integer',isRequired:true}],
				B1_PLMN:[{required:true,trigger:'blur'},{validator:validatePLMNID,trigger:'blur'}],
				B1_ReportQuantity:[{validator:validateQuantity}],
			},
			addB2DialogShow:false,
			addB2DialogForm:{
				B2_Enable:'1',
				B2_ThresholdTriggerType:'0',
				B2_Threshold1RSRP:'',
				B2_Threshold1RSRQ:'',
				B2_Threshold1SINR:'',
				B2_Threshold2TriggerType:'0',
				B2_Threshold2EUTRARSRP:'',
				B2_Threshold2EUTRARSRQ:'',
				B2_Threshold2EUTRASINR:'',
				B2_Hysteresis:'',
				B2_MaxReportCells:'',
				B2_MeasurePurpose:'1',
				B2_ReportAmount:'4',
				B2_ReportInterval:'5120',
				B2_TimeToTrigger:'480',
				B2_ReportQuantity:['rsrp','rsrq','sinr'],
				B2_ReportOnLeave:'1',
				B2_PLMN:'',
			},
			addB2DialogRules:{
				B2_Threshold1RSRP:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				B2_Threshold1RSRQ:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				B2_Threshold1SINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				B2_Threshold2EUTRARSRP:[{validator:validateRange,min:0,max:97,mag:'<%=rb.getString("FanWei")%>：0~97,Integer',isRequired:true}],
				B2_Threshold2EUTRARSRQ:[{validator:validateRange,min:0,max:34,mag:'<%=rb.getString("FanWei")%>：0~34,Integer',isRequired:true}],
				B2_Threshold2EUTRASINR:[{validator:validateRange,min:0,max:127,mag:'<%=rb.getString("FanWei")%>：0~127,Integer',isRequired:true}],
				B2_Hysteresis:[{validator:validateRange,min:0,max:30,mag:'<%=rb.getString("FanWei")%>：0~30,Integer',isRequired:true}],
				B2_MaxReportCells:[{validator:validateRange,min:1,max:8,mag:'<%=rb.getString("FanWei")%>：1~8,Integer',isRequired:true}],
				B2_PLMN:[{required:true,trigger:'blur'},{validator:validatePLMNID,trigger:'blur'}],
				B2_ReportQuantity:[{validator:validateQuantity}],
			},
            addPeriodMeasureDialogShow:false,
			addPeriodMeasureDialogForm:{
				PeriodMeasure_ReportQuantity:['rsrp','rsrq','sinr'],
                PeriodMeasure_MaxReportCells:'',
                PeriodMeasure_MeasurePurpose:'1',
                PeriodMeasure_ReportInterval:'5120',
                PeriodMeasure_ReportAmount:'4',
			},
			addPeriodMeasureDialogRules:{
                PeriodMeasure_ReportQuantity:[{validator:validateQuantity}],
				PeriodMeasure_MaxReportCells:[{validator:validateRange,min:1,max:8,mag:'<%=rb.getString("FanWei")%>：1~8,Integer',isRequired:true}],
			},
			addXnDialogShow:false,
			addXnDialogForm:{
				Xn_PLMNID:'',
				Xn_RemoteAddress:'',
				Xn_LinkEnable:'0',
				Xn_HoEnable:'0',
			},
			addXnDialogRules:{
				Xn_PLMNID:[{validator:validatePLMNID,trigger:'blur'}],
				Xn_RemoteAddress:[{validator:validateIPaddress,trigger:'blur'}],
			},
			addDLBWPDialogShow:false,
			addDLBWPDialogForm:{
				DLBWP_DIBwpID:'0',
				DLBWP_StartPrbPosition:'',
				DLBWP_BandWidth:'5',
                DLBWP_SubcarrierSpacing:'1',
                DLBWP_CyclicPrefix:'normal',
				DLBWP_PMaxULPower:'',
				DLBWP_InitDlMcs:'',
				DLBWP_MaxDlUeToBeScheduleInSlot:'',
			},
			addDLBWPDialogRules:{
				DLBWP_StartPrbPosition:[{validator:validateRange,min:0,max:273,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~273,Integer'}],
				DLBWP_PMaxULPower:[{validator:validateRange,min:-30,max:30,isRequired:true,mag:'<%=rb.getString("FanWei")%>：-30~30,Integer'}],
				DLBWP_InitDlMcs:[{validator:validateRange,min:0,max:28,isRequired:true,mag:'<%=rb.getString("FanWei")%>：0~28,Integer'}],
				DLBWP_MaxDlUeToBeScheduleInSlot:[{validator:validateRange,min:1,max:8,isRequired:true,mag:'<%=rb.getString("FanWei")%>：1~8,Integer'}],
			},
            BandWidthList:['5','10','15','20','25','30','40','50','60','70','80','90','100'],

			sharingSlideUrl:'',
			sharingSlideTitle:'',
			sharingSlideFooter:'',
			sharingSlideHeader:'',
			sharingSlidePosition:'',
			sharingSlideHeight:'',
			sharingSlideWidth:'',
			remoteAddress:'',
			remoteAddressErrorMessage:'',
            QOffsetDataList:['-24','-22','-20','-18','-16','-14','-12','-10','-8','-6','-5','-4','-3','-2','-1',
            '0','1','2','3','4','5','6','8','10','12','14','16','18','20','22','24'],
            CIODataList:['-24','-22','-20','-18','-16','-14','-12','-10','-8','-6','-5','-4','-3','-2','-1',
            '0','1','2','3','4','5','6','8','10','12','14','16','18','20','22','24'],
			sasEnableStatus:false,
			ReportAmountList:['0','2','4','8','16','32','64'],
			ReportIntervalList:['120','240','480','640','1024','2048','5120','10240','60000','360000','720000','1800000'],
			TimeToTriggerList:['0','40','64','80','100','128','160','256','320','480','512','640','1024','1280','2560','5120'],
			addFormCodes: {
				'LTENF': 'addLTENFDialogForm',
				'LTEER': 'addLTEERDialogForm',
				'LTENC': 'addLTENCDialogForm',
				'NRNF': 'addNRNFDialogForm',
				'NRIRS': 'addNRIRSDialogForm',
				'NRNC': 'addNRNCDialogForm',
				'QOS': 'addQOSDialogForm',
				'SST': 'addSSTDialogForm',
				'A1': 'addA1DialogForm',
				'A2': 'addA2DialogForm',
				'A3': 'addA3DialogForm',
				'A4': 'addA4DialogForm',
				'A5': 'addA5DialogForm',
				'B1': 'addB1DialogForm',
				'B2': 'addB2DialogForm',
                'PeriodMeasure': 'addPeriodMeasureDialogForm',
				'Xn': 'addXnDialogForm',
				'DLBWP': 'addDLBWPDialogForm',
			},
			addTableCodes:{
				'LTENF': 'LTENFList',
				'LTEER': 'LTEERList',
				'LTENC': 'LTENCList',
				'NRNF': 'NRNFList',
				'NRIRS': 'NRIRSList',
				'NRNC': 'NRNCList',
				'QOS': 'QOSList',
				'SST': 'SSTList',
				'A1': 'A1List',
				'A2': 'A2List',
				'A3': 'A3List',
				'A4': 'A4List',
				'A5': 'A5List',
				'B1': 'B1List',
				'B2': 'B2List',
                'PeriodMeasure': 'PeriodMeasureList',
				'Xn': 'XnList',
				'DLBWP': 'DLBWPList',
			},
			addShowCodes:{
				'LTENF': 'addLTENFDialogShow',
				'LTEER': 'addLTEERDialogShow',
				'LTENC': 'addLTENCDialogShow',
				'NRNF': 'addNRNFDialogShow',
				'NRIRS': 'addNRIRSDialogShow',
				'NRNC': 'addNRNCDialogShow',
				'QOS': 'addQOSDialogShow',
				'SST': 'addSSTDialogShow',
				'A1': 'addA1DialogShow',
				'A2': 'addA2DialogShow',
				'A3': 'addA3DialogShow',
				'A4': 'addA4DialogShow',
				'A5': 'addA5DialogShow',
				'B1': 'addB1DialogShow',
				'B2': 'addB2DialogShow',
                'PeriodMeasure': 'addPeriodMeasureDialogShow',
				'Xn': 'addXnDialogShow',
				'DLBWP': 'addDLBWPDialogShow',
			},
			bandAndPowerMaxList:[],
			isSharingBaseStation:false,
            resetParams:{}
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
		addRemoteAddressShow(){
			var Num = 0;
			if(this.ruleForm.XnBlacklist.length>0){
				var delList=[],
					noDel=[];
				this.ruleForm.XnBlacklist.map((item)=>{
					if(item.operateType && item.operateType == 'remove'){
						delList.push(item)
					}else{
						noDel.push(item)
					}
				})
				Num = noDel.length;
			}

			return Num < 8
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
		DataSplitThresholdList(){
			var list = ['B0','B100','B200','B400','B800','B1600','B3200','B6400','B12800','B25600','B51200',
			'B102400','B204800','B409600','B818200','B1228800','B1638400','B2457600','B3276800','B4096000','B4915200','B5734400','B6553600'],
				fotList = [];
			list.map((item,index)=>{
				var obj = {};
				obj.label = item;
				obj.value = index+'';
				fotList.push(obj);
			});
			return fotList
		},
		DiscardTimerList(){
			var list = ['ms10','ms20','ms30','ms40','ms50','ms60','ms75','ms100','ms150','ms200','ms250',
			'ms300','ms500','ms750','ms1500','infinity'],
				fotList = [];
			list.map((item,index)=>{
				var obj = {};
				obj.label = item;
				obj.value = index+'';
				fotList.push(obj);
			});
			return fotList
		},
		gnbConfigAddDialogTitle(){
			return this.optType == 'add' ? '<%=rb.getString("TianJia")%>' : '<%=rb.getString("XiuGai")%>'
		},
		bwpListAddShow(){
			var arr = this.ruleForm.DLBWPList.filter((item)=>{
				return  !item.operateType || (item.operateType &&item.operateType != 'remove')
			})
			return arr.length<5? true : false;
		},
		lteBlackPhysCellIdRangeList(){
			var list = ['n4','n8','n12','n16','n24','n32','n48','n64','n84','n96','n128',
			'n168','n252','n504'],
				fotList = [];
			list.map((item,index)=>{
				var obj = {};
				obj.label = item;
				obj.value = index+'';
				fotList.push(obj);
			});
			fotList.push({label:'OFF',value:'0x0FFFFFFF'})
			return fotList
		},
		nrBlackPhysCellIdRangeList(){
			var list = ['n4','n8','n12','n16','n24','n32','n48','n64','n84','n96','n128',
			'n168','n252','n504','n1008'],
				fotList = [];
			list.map((item,index)=>{
				var obj = {};
				obj.label = item;
				obj.value = index+'';
				fotList.push(obj);
			});
			fotList.push({label:'OFF',value:'0x0FFFFFFF'})
			return fotList
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
		init(row,sasEnableStatus,ranConfigParams){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = row.small_cell_code;
			vm.sasEnableStatus = sasEnableStatus;
            vm.casts = ranConfigParams.casts;
            vm.resetParams = ranConfigParams.resetParams;
			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'23004');
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
						if(type == 'NR Cell List'){
							vm.ruleForm.NRCellList.push(obj);
						}else if(type == 'LTE N-FREQ List'){
							vm.ruleForm.LTENFList.push(obj);
						}else if(type == 'EUTRACarrierList'){
							vm.ruleForm.LTEERList.push(obj);
						}else if(type == 'LTE N-CELL List'){
							vm.ruleForm.LTENCList.push(obj);
						}else if(type == 'N FREQ List'){
							vm.ruleForm.NRNFList.push(obj);
						}else if(type == 'InterFreqCarrierList'){
							vm.ruleForm.NRIRSList.push(obj);
						}else if(type == 'N Cell List'){
							vm.ruleForm.NRNCList.push(obj);
						}else if(type == 'QOS List'){
							vm.ruleForm.QOSList.push(obj);
						}else if(type == 'SST List'){
							vm.ruleForm.SSTList.push(obj);
						}else if(type == 'A1MeasureCtrlList'){
							vm.ruleForm.A1List.push(obj);
						}else if(type == 'A2MeasureCtrlList'){
							vm.ruleForm.A2List.push(obj);
						}else if(type == 'A3MeasureCtrlList'){
							vm.ruleForm.A3List.push(obj);
						}else if(type == 'A4MeasureCtrlList'){
							vm.ruleForm.A4List.push(obj);
						}else if(type == 'A5MeasureCtrlList'){
							vm.ruleForm.A5List.push(obj);
						}else if(type == 'B1MeasureCtrlList'){
							vm.ruleForm.B1List.push(obj);
						}else if(type == 'B2MeasureCtrlList'){
							vm.ruleForm.B2List.push(obj);
						}else if(type == 'PeriodMeasCtrlList'){
							vm.ruleForm.PeriodMeasureList.push(obj);
						}else if(type == 'Xn List'){
							vm.ruleForm.XnList.push(obj);
						}else if(type == 'Xn Blacklist'){
							vm.ruleForm.XnBlacklist.push(obj);
						}else if(type == 'DL BWP List'){
							vm.ruleForm.DLBWPList.push(obj);
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
					LTENFList:[],
					LTEERList:[],
					LTENCList:[],
					NRNFList:[],
					NRIRSList:[],
					NRNCList:[],
					QOSList:[],
					SSTList:[],
					XnList:[],
					XnBlacklist:[],
					DLBWPList:[],

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
					// ssbGSCN:'',

					A1ThresholdRSRP:'',
					A2ThresholdRSRP:'',
					A3OffsetRSRP:'',
					A4ThresholdRSRP:'',
					A5Threshold1RSRP:'',
					A5Threshold2RSRP:'',
					B1ThresholdEUTRARSRP:'',
					B2Threshold1RSRP:'',
					B2Threshold2RSRP:'',

					ANR_Enable:'0',
					ANR_InterFeqEnable:'0',
					ANR_EUTRANEnable:'0',
					ANR_BiNRCellEnable:'0',
					ANR_MRTriggerType:'0',
					ANR_AbsoluteThreshold:'',
					ANR_RelativeThreshold:'',
					ANR_AbsEnable:'0',
					ANR_KpiPeriod:'',
					ANR_AutoAdjustEnable:'0',
					ANR_AutoRemoveEnable:'0',
					ANR_AutoRemovePeriod:'',
					ANR_AutoRemoveMaxCell:'',
					ANR_MaxHOtimes:'',
					ANR_MaxHOSuccess:'',
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
				urls='${ctx}/cell/quicksettings/getListParamValue.action?parent_id=233051&platform=BaiBNQ';

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
								sliceUrl = '${ctx}/cell/quicksettings/getListParamValue.action?parent_id=2330511&platform=BaiBNQ';
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
		// 新增 RemoteAddress
		addRemoteAddress(){
			var vm =this,
				regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
				val = vm.remoteAddress;
			if(!val)return
			if(vm.isValidIP(val) || vm.isIPv6(val)) {
					var obj={
						remoteAddress:val,
						operateType:'add'
					};
				vm.ruleForm.XnBlacklist.push(obj);
				vm.remoteAddress = '';
				vm.remoteAddressErrorMessage = '';
			}else {
				vm.remoteAddressErrorMessage = 'Example：1.1.1.1';
			}
		},
		remoteAddressListDel(row,index){
			var vm =this,
				delItem =  vm.ruleForm.XnBlacklist[index];

			if(delItem.operateType && delItem.operateType == 'add'){
					vm.ruleForm.XnBlacklist.splice(index,1)
			}else{
				vm.$set(vm.ruleForm.XnBlacklist[index],'operateType','remove');
			}
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
								if(listVal != 'Xn_Status'){
									objs[listKey] = items[listVal]
								}
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
		smtcPeriodicityFmt(row, value, index){
			var statusObj = {
				'0':"sf5",	
				'1':"sf10",
				'2':"sf20",	
				'3':"sf40",	
				'4':"sf80",	
				'5':"sf160",
				'':"",	
			}
			return statusObj[value];
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
					gnbTabSettingVue.changeMain('ran');
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		isLessThan(val,target){
			var max = target + '',
				num = val + '',
				bool = true,
				list = [];
			if(num.length > max.length || isNaN(num)){
				bool = false;
			}
			if(num.length == max.length){
				for(var i = 0;i<max.length;i++){
					var isBig = max[i] - num[i] >= 0;
					list.push(max[i] - num[i] > 0);
					if(!isBig){
						var some = list.filter(function(item){return item == true;});
						if(some.length == 0){
							bool = false;
							break;
						}

					}
				}
			}
			return bool;
		},
		// 打开BWP详情
		DLBWPTableDetails(row){
			var vm = this;
				
			vm.sharingSlideUrl = '${ctx}/gnb/setting/openBwpDetailsPage.action';
			vm.sharingSlideHeight = '100%';
			vm.sharingSlideWidth = '100%';
			vm.sharingSlideFooter = false;
			vm.sharingSlidePosition = 'top';
			vm.sharingSlideHeader = false;
			vm.sharingSlideTitle = '';
			vm.$refs.sharingSlide.showSlide(function(){
				eventBus.$emit('bwpDetails-init',row,vm.smallCellCode);
			});
		},
		// 关闭slide页面
		sharingSlideCancel(){
			gnbRanPage.$refs.sharingSlide.hide();
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
		// RAN 打开表格新增、修改
		ranTableAddDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
                let rowData = JSON.parse(JSON.stringify(row));
                Object.keys(rowData).forEach(function(key){
                    if(['A1_ReportQuantity','A1_RptQuantityRsIndex','A2_ReportQuantity','A2_RptQuantityRsIndex','A3_ReportQuantity','A3_RptQuantityRsIndex','A4_ReportQuantity','A4_RptQuantityRsIndex','A5_ReportQuantity','A5_RptQuantityRsIndex','B1_ReportQuantity','B2_ReportQuantity','PeriodMeasure_ReportQuantity'].includes(key)){
                        rowData[key] = rowData[key] ? rowData[key].split(',') : [];
                    }
                });
				Object.assign(vm[vm.addFormCodes[tbType]],rowData);
			}
			vm[vm.addShowCodes[tbType]] = true;
			vm.$nextTick(function(){
				vm.$refs[vm.addFormCodes[tbType]].clearValidate();
			});
		},
		// RAN 打开表格新增、修改 提交
		ranTableAddDialogSubmit(){
			var vm = this,
				params = {},
				idxStr = vm.tbType + '_idx',
				optTb = vm.addTableCodes[vm.tbType];

			Object.keys(vm[vm.addFormCodes[vm.tbType]]).forEach(function(key){
                if(['A1_ReportQuantity','A1_RptQuantityRsIndex','A2_ReportQuantity','A2_RptQuantityRsIndex','A3_ReportQuantity','A3_RptQuantityRsIndex','A4_ReportQuantity','A4_RptQuantityRsIndex','A5_ReportQuantity','A5_RptQuantityRsIndex','B1_ReportQuantity','B2_ReportQuantity','PeriodMeasure_ReportQuantity'].includes(key)){
                    params[key] = vm[vm.addFormCodes[vm.tbType]][key].join(',');
                }else{
                    params[key] = vm[vm.addFormCodes[vm.tbType]][key];
                }
			})
			if(vm[vm.addFormCodes[vm.tbType]].operateType){
				params.operateType = vm[vm.addFormCodes[vm.tbType]].operateType
			}
			vm.$refs[vm.addFormCodes[vm.tbType]].validate(function(valid){
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
					vm[vm.addShowCodes[vm.tbType]] = false;
				}
			})
		},
		// RAN 表格删除
		ranTableDelList(row,tbType){
			var vm = this,
				idxStr = tbType + '_idx',
				optTb = vm.addTableCodes[tbType];
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
		// 关闭 RAN 表格 新增,修改弹窗
		ranTablecloseAddDialog(){
			var vm = this,
				tbType = vm.tbType,
				params = vm.resetParams;
			Object.assign(vm[vm.addFormCodes[tbType]],params[tbType]);
			vm.$nextTick(function(){
				vm.$refs[vm.addFormCodes[tbType]].clearValidate();
			});
		},
		// band 改变事件
		bandSelectChange(){
			var vm = this;
			vm.$refs.ruleForm.validateField('PowerModify');
		},
        // 生成固定范围的数组
        generateArray(min, max) {
            if (min > max) {
                return [];
            }
            
            let result = [];
            for (let i = min; i <= max; i++) {
                result.push(i+'');
            }
            return result;
        }
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data",this.init)
	}
});
</script>
