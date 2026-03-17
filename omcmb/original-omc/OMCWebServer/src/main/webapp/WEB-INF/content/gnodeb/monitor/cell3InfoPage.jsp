<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<style>

	#cell4InfoPage {
		height: 100%;
	}
	.el-ctable .hidden-row{
		display: none;
	}
	#cell4InfoPage .subInputCls .el-select>.el-input{
		width: 150px!important;
	}
	#cell4InfoPage .EnergySavingTypeSelectCls .el-select>.el-input{
		width: 110px!important;
	}
	#cell4InfoPage .specialItem .el-input__inner{
		width:110px !important;
	}
	#cell4InfoPage .specialItem .el-input-group__append{
		padding-left: 50px!important;
	}
	
</style>

<div id="cell4InfoPage">
	<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="left">
		<div class="cellItemBoxCls">
			<div class="cellItemLeftCls">
				<div v-for="item in leftTreeData"  :class="leftTabsActive == item.code ? 'tabsActiveCls':''" @click="leftTabsClick(item.code,item.id)">{{item.text}}</div>
			</div>
			<div class="cellItemRightCls">
				<div v-show="leftTabsActive == 'Quick Settings'">
					<el-collapse v-model="quickSettingsCollapse">
						<el-collapse-item name="quickSettings">
							<template slot='title'>
								<p style="display:inline-block;margin-left:40px;">
									<span class="title-icon" style="vertical-align:sub"></span>
									<span style="font-size:14px;font-weight:bold">Quick Settings</span>
								</p>
							</template>
							<div class="rightContentCls">
								<!--PLMN-->
								<div> 
									<div class="itemTitleCls">
										<div></div>
										<div>PLMN</div>
									</div>
									<div class="contentTableTitle">
										<div>NR Cell <span style="font-size:12px;color:#999999;margin-left:10px;">(No more than 6)</span></div>
										<div v-show="ruleForm.NrCellList.length<6"><span class="el-icon el-icon-circle-add" @click="addNrCell('QuickSettings')"></span></div>
									</div>
									<div class="cellTableBoxCls">
										<el-ctable 
											ref="NRCellTable" 
											:rownumber="true" 
											:row-class-name="tableRowClassName"
											id="NRCellTable" 
											:data="ruleForm.NrCellList" 
											height="100%"
											:pagination="false"
											style="border:1px solid #E9E9E9;"
										>
											<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
												<template slot-scope="scope">
													<span class="el-icon el-icon-operation-edit" @click="NRCellEdit(scope.row,'QuickSettings',event)" style="margin-right:15px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="NRCellDel(scope.row,'QuickSettings',event)" ></span>
												</template>
											</el-table-column>
											<el-table-column label='ID' min-width="120" prop="NrCell_idx" show-overflow-tooltip></el-table-column>
											<el-table-column label='NR Cell Identity' min-width="120" prop="NRCellIdentity" show-overflow-tooltip></el-table-column>
											<el-table-column label='TAC' min-width="120" prop="NrCellTAC" show-overflow-tooltip></el-table-column>
											<el-table-column label='Ranac' min-width="120" prop="NrCellRanac" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-form-item prop='NrCellList' style="display:none;" label="" label-width="0px">
											<el-input v-model='ruleForm.NrCellList'></el-input>
										</el-form-item>
									</div>
								</div>
								<!--RF-->
								<div>
									<div class="itemTitleCls">
										<div></div>
										<div>RF</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='band' style="width:40%;min-width:500px;" label="Band" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.band' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：1-1000</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='NRARFCNDL' style="width:40%;min-width:500px;" label="NRARFCNDL" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.NRARFCNDL' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='NRARFCNUL' style="width:40%;min-width:500px;" label="NRARFCNUL" label-width="160px" class='validate-item'>
											<span slot="label" class="labelIconCls">
												NRARFCNUL
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-menu-help el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-input v-model.trim='ruleForm.NRARFCNUL' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='Pci' style="width:40%;min-width:500px;" label="PCI" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.Pci' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~1007,Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='NumOfTxAtenna' style="width:40%;min-width:500px;" label="NumOfTxAtenna" label-width="160px">
											<span slot="label" class="labelIconCls">
												NumOfTxAtenna
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-select v-model='ruleForm.NumOfTxAtenna' style="width:70px;padding-top:5px;">
												<el-option label='1' value='1'></el-option>
												<el-option label='2' value='2'></el-option>
												<el-option label="4" value="4"></el-option>
												<el-option label="8" value="8"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item prop='NumOfRxAtenna' style="width:40%;min-width:500px;" label="NumOfRxAtenna" label-width="160px">
											<span slot="label" class="labelIconCls">
												NumOfRxAtenna
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-select v-model='ruleForm.NumOfRxAtenna' style="width:70px;padding-top:5px;">
												<el-option label='1' value='1'></el-option>
												<el-option label='2' value='2'></el-option>
												<el-option label="4" value="4"></el-option>
												<el-option label="8" value="8"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item prop='rftxEnable' style="width:40%;min-width:500px;" label="RF Enable" label-width="160px">
											<el-switch v-model="ruleForm.rftxEnable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
										</el-form-item>
									</div>
								</div>
								<!--PHY-->
								<div>
									<div class="itemTitleCls">
										<div></div>
										<div>PHY</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='DL_SubCarrierSpacing' style="width:40%;min-width:500px;" label="DL SubCarrierSpacing" class="subInputCls" label-width="160px">
											<span slot="label" class="labelIconCls">
												DL SubCarrierSpacing
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-select v-model='ruleForm.DL_SubCarrierSpacing' @change="DL_SCSChange" style="width:150px;padding-top:5px;">
												<el-option label='0(15kHz)' value='0'></el-option>
												<el-option label='1(30kHz)' value='1'></el-option>
												<el-option label="2(60kHz)" value="2"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item prop='DL_CarrierBandWidth' style="width:40%;min-width:500px;" label="DL CarrierBandWidth" class="subInputCls" label-width="160px">
											<span slot="label" class="labelIconCls">
												DL CarrierBandWidth
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-select v-model='ruleForm.DL_CarrierBandWidth' style="width:150px;padding-top:5px;">
												<el-option v-for="item in DL_CarrierBandWidthData" :label='item.label' :value='item.value'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item prop='UL_SubCarrierSpacing' style="width:40%;min-width:500px;" label="UL SubCarrierSpacing" class="subInputCls" label-width="160px">
											<span slot="label" class="labelIconCls">
												UL SubCarrierSpacing
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-select v-model='ruleForm.UL_SubCarrierSpacing' @change="UL_SCSChange" style="width:150px;padding-top:5px;">
												<el-option label='0(15kHz)' value='0'></el-option>
												<el-option label='1(30kHz)' value='1'></el-option>
												<el-option label="2(60kHz)" value="2"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item prop='UL_CarrierBandWidth' style="width:40%;min-width:500px;" label="UL CarrierBandWidth" class="subInputCls" label-width="160px">
											<span slot="label" class="labelIconCls">
												UL CarrierBandWidth
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-select v-model='ruleForm.UL_CarrierBandWidth' style="width:150px;padding-top:5px;">
												<el-option v-for="item in UL_CarrierBandWidthData" :label='item.label' :value='item.value'></el-option>
											</el-select>
										</el-form-item>
									</div>
								</div>
								<!--Power-->
								<div>
									<div class="itemTitleCls">
										<div></div>
										<div>Power</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='PowerModify' style="width:40%;min-width:500px;" label="Power Modify" label-width="160px" class='validate-item specialItem' >
											<span slot="label" class="labelIconCls">
												Power Modify
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-input v-model.trim='ruleForm.PowerModify' style="width:110px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~43,Integer</template>
											</el-input>
											<span class="prefix-label" style="position:relative;left:-190px;top:5px;">
												dBm
											</span>
										</el-form-item>
									</div>
								</div>
							</div>
						</el-collapse-item>
					</el-collapse>
				</div>
				<div v-show="leftTabsActive == 'RAN'">
					<el-collapse v-model="RAN_Collapse">
						<el-collapse-item name="neighborFreq">
							<template slot='title'>
								<p style="display:inline-block;margin-left:40px;">
									<span class="title-icon" style="vertical-align:sub"></span>
									<span style="font-size:14px;font-weight:bold">Neighbor Freq/Cell</span>
								</p>
							</template>
							<div class="rightContentCls">
								<!--LTE-->
								<div> 
									<div class="itemTitleCls">
										<div></div>
										<div>LTE</div>
									</div>
									<div class="contentTableTitle">
										<div>LTE N-FREQ List</div>
										<div><span class="el-icon el-icon-circle-add" @click="addTableClick({},'add','LF')"></span></div>
									</div>
									<div class="cellTableBoxCls">
										<el-ctable 
											ref="LF_Table" 
											:row-class-name="tableRowClassName"
											:rownumber="false" 
											id="LF_Table" 
											:data="ruleForm.LF_List" 
											height="100%"
											:pagination="true"
                                            :front-pagination="true"
											style="border:1px solid #E9E9E9;"
										>
											<el-table-column label='ID' min-width="40" prop="LF_idx" show-overflow-tooltip></el-table-column>
											<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
												<template slot-scope="scope">
													<span class="el-icon el-icon-operation-edit" @click="addTableClick(scope.row,'edit','LF')" style="margin-right:15px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="delTableClick(scope.row,'LF')" ></span>
												</template>
											</el-table-column>
											<el-table-column label='CarrierFreq' min-width="120" prop="LF_CarrierFreq" show-overflow-tooltip></el-table-column>
											<el-table-column label='QOffset' min-width="120" prop="LF_QOffset" show-overflow-tooltip></el-table-column>
											<el-table-column label='CellReselectionPriority' min-width="120" prop="LF_CellReselectionPriority" show-overflow-tooltip></el-table-column>
											<el-table-column label='QRxLevMin' min-width="120" prop="LF_QRxLevMin" show-overflow-tooltip></el-table-column>
											<el-table-column label='QQualMin' min-width="120" prop="LF_QQualMin" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-form-item prop='LF_List' style="display:none;" label="" label-width="0px">
											<el-input v-model='ruleForm.LF_List'></el-input>
										</el-form-item>
									</div>
									<div class="contentTableTitle" style="padding-top:20px;">
										<div>LTE N-CELL List</div>
										<div><span class="el-icon el-icon-circle-add" @click="addTableClick({},'add','LC')"></span></div>
									</div>
									<div class="cellTableBoxCls">
										<el-ctable 
											ref="LC_Table" 
											:row-class-name="tableRowClassName"
											:rownumber="false" 
											id="LC_Table" 
											:data="ruleForm.LC_List" 
											height="100%"
											:pagination="true"
                                            :front-pagination="true"
											style="border:1px solid #E9E9E9;"
										>
											<el-table-column label='ID' min-width="40" prop="LC_idx" show-overflow-tooltip></el-table-column>
											<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
												<template slot-scope="scope">
													<span class="el-icon el-icon-operation-edit" @click="addTableClick(scope.row,'edit','LC')" style="margin-right:15px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="delTableClick(scope.row,'LC')" ></span>
												</template>
											</el-table-column>
											<el-table-column label='PLMNID' min-width="120" prop="LC_PLMNID" show-overflow-tooltip></el-table-column>
											<el-table-column label='CID' min-width="120" prop="LC_CID" show-overflow-tooltip></el-table-column>
											<el-table-column label='EUTRACarrierARFCN' min-width="120" prop="LC_EUTRACarrierARFCN" show-overflow-tooltip></el-table-column>
											<el-table-column label='PhyCellID' min-width="120" prop="LC_PhyCellID" show-overflow-tooltip></el-table-column>
											<el-table-column label='QOffset' min-width="120" prop="LC_QOffset" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-form-item prop='LC_List' style="display:none;" label="" label-width="0px">
											<el-input v-model='ruleForm.LC_List'></el-input>
										</el-form-item>
									</div>
									<!--NR-->
									<div class="itemTitleCls">
										<div></div>
										<div>NR</div>
									</div>
									<div class="contentTableTitle">
										<div>N FREQ List</div>
										<div><span class="el-icon el-icon-circle-add" @click="addTableClick({},'add','NF')"></span></div>
									</div>
									<div class="cellTableBoxCls">
										<el-ctable 
											ref="N_FREQ_Table" 
											:row-class-name="tableRowClassName"
											:rownumber="false" 
											id="N_FREQ_Table" 
											:data="ruleForm.NF_List" 
											height="100%"
											:pagination="true"
                                            :front-pagination="true"
											style="border:1px solid #E9E9E9;"
										>
											<el-table-column label='ID' min-width="40" prop="NF_idx" show-overflow-tooltip></el-table-column>
											<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
												<template slot-scope="scope">
													<span class="el-icon el-icon-operation-edit" @click="addTableClick(scope.row,'edit','NF')" style="margin-right:15px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="delTableClick(scope.row,'NF')" ></span>
												</template>
											</el-table-column>
											<el-table-column label='Enable' min-width="120" prop="NF_Enable" show-overflow-tooltip></el-table-column>
											<el-table-column label='SSBFrequency' min-width="120" prop="NF_SSBFrequency" show-overflow-tooltip></el-table-column>
											<el-table-column label='SmtcPeriodicity' min-width="120" prop="NF_SmtcPeriodicity" show-overflow-tooltip>
												<template slot-scope="scope">
													<div v-html="smtcPeriodicityFmt(scope.row, scope.row.NF_SmtcPeriodicity, scope.$index)"></div>
												</template>
											</el-table-column>
											<el-table-column label='RsrpOffsetSSB' min-width="120" prop="NF_RsrpOffsetSSB" show-overflow-tooltip></el-table-column>
											<el-table-column label='Bitmap' min-width="120" prop="NF_Bitmap" show-overflow-tooltip></el-table-column>
											<el-table-column label='FreqBandIndicatorNR' min-width="120" prop="NF_FreqBandIndicatorNR" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-form-item prop='NF_List' style="display:none;" label="" label-width="0px">
											<el-input v-model='ruleForm.NF_List'></el-input>
										</el-form-item>
									</div>
									<div class="contentTableTitle" style="padding-top:20px;">
										<div>N CELL List</div>
										<div><span class="el-icon el-icon-circle-add" @click="addTableClick({},'add','NC')"></span></div>
									</div>
									<div class="cellTableBoxCls">
										<el-ctable 
											ref="N_CELL_Table" 
											:row-class-name="tableRowClassName"
											:rownumber="false" 
											id="N_CELL_Table" 
											:data="ruleForm.NC_List" 
											height="100%"
											:pagination="true"
                                            :front-pagination="true"
											style="border:1px solid #E9E9E9;"
										>
											<el-table-column label='ID' min-width="40" prop="NC_idx" show-overflow-tooltip></el-table-column>
											<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100">
												<template slot-scope="scope">
													<span class="el-icon el-icon-operation-edit" @click="addTableClick(scope.row,'edit','NC')" style="margin-right:15px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="delTableClick(scope.row,'NC')" ></span>
												</template>
											</el-table-column>
											<el-table-column label='PLMNID' min-width="120" prop="NC_PLMNID" show-overflow-tooltip></el-table-column>
											<el-table-column label='CID' min-width="120" prop="NC_CID" show-overflow-tooltip></el-table-column>
											<el-table-column label='NRARFCN' min-width="120" prop="NC_NRARFCN" show-overflow-tooltip></el-table-column>
											<el-table-column label='PhyCellID' min-width="120" prop="NC_PhyCellID" show-overflow-tooltip></el-table-column>
											<el-table-column label='QOffset' min-width="120" prop="NC_QOffset" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-form-item prop='NC_List' style="display:none;" label="" label-width="0px">
											<el-input v-model='ruleForm.NC_List'></el-input>
										</el-form-item>
									</div>
								</div>
							</div>
						</el-collapse-item>
						<el-collapse-item name="mobility">
							<template slot='title'>
								<p style="display:inline-block;margin-left:40px;">
									<span class="title-icon" style="vertical-align:sub"></span>
									<span style="font-size:14px;font-weight:bold">Mobility</span>
								</p>
							</template>
							<div class="rightContentCls">
								<!--A1-->
								<div> 
									<div class="itemTitleCls">
										<div></div>
										<div>A1</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='A1ThresholdRSRP' style="width:40%;min-width:500px;" label="A1ThresholdRSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.A1ThresholdRSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
											</el-input>
										</el-form-item>
									</div>
									<div class="itemTitleCls">
										<div></div>
										<div>A2</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='A2ThresholdRSRP' style="width:40%;min-width:500px;" label="A2ThresholdRSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.A2ThresholdRSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
											</el-input>
										</el-form-item>
									</div>
									<div class="itemTitleCls">
										<div></div>
										<div>A3</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='A3OffsetRSRP' style="width:40%;min-width:500px;" label="A3OffsetRSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.A3OffsetRSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：-30~30,Integer</template>
											</el-input>
										</el-form-item>
									</div>
									<div class="itemTitleCls">
										<div></div>
										<div>A4</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='A4ThresholdRSRP' style="width:40%;min-width:500px;" label="A4ThresholdRSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.A4ThresholdRSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
											</el-input>
										</el-form-item>
									</div>
									<div class="itemTitleCls">
										<div></div>
										<div>A5</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='A5Threshold1RSRP' style="width:40%;min-width:500px;" label="A5Threshold1RSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.A5Threshold1RSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='A5Threshold2RSRP' style="width:40%;min-width:500px;" label="A5Threshold2RSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.A5Threshold2RSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
											</el-input>
										</el-form-item>
									</div>
									<div class="itemTitleCls">
										<div></div>
										<div>B1</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='B1ThresholdEUTRARSRP' style="width:40%;min-width:500px;" label="B1ThresholdEUTRARSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.B1ThresholdEUTRARSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~97,Integer</template>
											</el-input>
										</el-form-item>
									</div>
									<div class="itemTitleCls">
										<div></div>
										<div>B2</div>
									</div>
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='B2Threshold1RSRP' style="width:40%;min-width:500px;" label="B2Threshold1RSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.B2Threshold1RSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='B2Threshold2RSRP' style="width:40%;min-width:500px;" label="B2Threshold2RSRP" label-width="160px" class='validate-item'>
											<el-input v-model.trim='ruleForm.B2Threshold2RSRP' style="width:150px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~127,Integer</template>
											</el-input>
										</el-form-item>
									</div>
								</div>
							</div>
						</el-collapse-item>
						<el-collapse-item name="EnergySavingSetting">
							<template slot='title'>
								<p style="display:inline-block;margin-left:40px;">
									<span class="title-icon" style="vertical-align:sub"></span>
									<span style="font-size:14px;font-weight:bold">Energy Saving Setting</span>
								</p>
							</template>
							<div class="rightContentCls" >
								<div style="padding-top:30px;"> 
									<div style="display:flex;margin-left:16px;flex-wrap: wrap">
										<el-form-item prop='EnergySavingType' style="width:40%;min-width:500px;" label="EnergySavingType" class="EnergySavingTypeSelectCls"  label-width="170px">
											<span slot="label" class="labelIconCls">
												EnergySavingType
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-select v-model='ruleForm.EnergySavingType' style="width:110px;padding-top:7px;">
												<el-option label='Not Saving' value='NOT_SAVING'></el-option>
												<el-option label='Deep Saving' value='DEEP_SAVING'></el-option>
												<el-option label='Shallow Saving' value='SHALLOW_SAVING'></el-option>
												<el-option label="Slot Switch" value="SLOT_SWITCH"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item prop='EnergySavingTime' style="width:40%;min-width:500px;" label="EnergySavingTime" label-width="170px">
											<span slot="label" class="labelIconCls">
												EnergySavingTime
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-time-picker
												is-range
												style="width:200px;margin-top:7px;"
												v-model="EnergySavingTimeList"
												@change="EnergySavingTimeListChange"
												range-separator="-"
												value-format="HH:mm:ss"
												start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
												end-placeholder='<%=rb.getString("JieShuShiJian")%>'
											></el-time-picker>
											<span style="margin-left:5px;">For example:01:00:00-06:00:00</span>
										</el-form-item>
										<el-form-item prop='EnergySavingDelayTime' style="width:40%;min-width:500px;" label="EnergySavingDelayTime" label-width="170px" class='validate-item'>
											<span slot="label" class="labelIconCls">
												EnergySavingDelayTime
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-input v-model.trim='ruleForm.EnergySavingDelayTime' style="width:110px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='PrbLowThreshold' style="width:40%;min-width:500px;" label="PrbLowThreshold" label-width="170px" class='validate-item'>
											<span slot="label" class="labelIconCls">
												PrbLowThreshold
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-input v-model.trim='ruleForm.PrbLowThreshold' style="width:110px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~100,Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='PrbReportPeriod' style="width:40%;min-width:500px;" label="PrbReportPeriod" label-width="170px" class='validate-item'>
											<span slot="label" class="labelIconCls">
												PrbReportPeriod
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-input v-model.trim='ruleForm.PrbReportPeriod' style="width:110px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
											</el-input>
										</el-form-item>
										<el-form-item prop='RRCLowThreshold' style="width:40%;min-width:500px;" label="RRCLowThreshold" label-width="170px" class='validate-item'>
											<span slot="label" class="labelIconCls">
												RRCLowThreshold
												<!--<el-tooltip placement="bottom">
													<div slot="content">
														123456312123151<br/>
														321354534546
													</div>
													<span class="el-icon-circle-info el-icon"></span>
												</el-tooltip>-->
											</span>
											<el-input v-model.trim='ruleForm.RRCLowThreshold' style="width:110px;padding-top:5px;">
												<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,Integer</template>
											</el-input>
										</el-form-item>
									</div>
								</div>
							</div>
						</el-collapse-item>
						<!--<el-collapse-item name="ranPlmn">
							<template slot='title'>
								<p style="display:inline-block;margin-left:40px;">
									<span class="title-icon" style="vertical-align:sub"></span>
									<span style="font-size:14px;font-weight:bold">PLMN</span>
								</p>
							</template>
							<div class="rightContentCls">
								<div> 
									<div class="contentTableTitle">
										<div>NR Cell <span style="font-size:12px;color:#999999;margin-left:10px;">(No more than 6)</span></div>
										<div v-show="ruleForm.ranNrCellList.length<6"><span class="el-icon el-icon-circle-add" @click="addNrCell('RAN')"></span></div>
									</div>
									<div class="cellTableBoxCls">
										<el-ctable 
											ref="ranNRCellTable" 
											:rownumber="true" 
											:row-class-name="tableRowClassName"
											id="ranNRCellTable" 
											:data="ruleForm.ranNrCellList" 
											height="100%"
											:pagination="false"
											style="border:1px solid #E9E9E9;"
										>
											<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
												<template slot-scope="scope">
													<span class="el-icon el-icon-operation-edit" @click="NRCellEdit(scope.row,'RAN',event)" style="margin-right:15px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="NRCellDel(scope.row,'RAN',event)" ></span>
												</template>
											</el-table-column>
											<el-table-column label='ID' min-width="120" prop="ranNrCell_idx" show-overflow-tooltip></el-table-column>
											<el-table-column label='NR Cell Identity' min-width="120" prop="ranNRCellIdentity" show-overflow-tooltip></el-table-column>
											<el-table-column label='TAC' min-width="120" prop="ranNrCellTAC" show-overflow-tooltip></el-table-column>
											<el-table-column label='Ranac' min-width="120" prop="ranNrCellRanac" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-form-item prop='ranNrCellList' style="display:none;" label="" label-width="0px">
											<el-input v-model='ruleForm.ranNrCellList'></el-input>
										</el-form-item>
									</div>
								</div>
							</div>
						</el-collapse-item>-->
					</el-collapse>
				</div>
			</div>
		</div>
	</el-form>
</div>
<script type="text/javascript">

var cell4InfoVue =  new Vue({
	el:"#cell4InfoPage",
	data(){
		var vm = this;
		
		var validateBand = (rule,value,callback) => {
				if(vm.leftTabsActive != 'Quick Settings'){
					callback();
				}else{
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 1~1000'))
					}else{
						if(vm.isInteger(value)&& parseInt(value)>=1 && parseInt(value)<=1000){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 1~1000'))
						}
					} 
				}
			},
			validateNRARFCNDL = (rule,value,callback) => {
				if(vm.leftTabsActive != 'Quick Settings'){
					callback();
				}else{
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
					}else{
						if(vm.isInteger(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
						}
					}
				}
			},
			validateNRARFCNUL = (rule,value,callback) => {
				if(vm.leftTabsActive != 'Quick Settings'){
					callback();
				}else{
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
					}else{
						if(vm.isInteger(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%><%=rb.getString("ZhengXing")%>'))
						}
					}
				}
			},
			validatePci = (rule,value,callback) => {
				if(vm.leftTabsActive != 'Quick Settings'){
					callback();
				}else{
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~1007 <%=rb.getString("ZhengXing")%>'))
					}else{
						if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=1007){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~1007 <%=rb.getString("ZhengXing")%>'))
						}
					} 
				}
			},
			validatePowerModify = (rule,value,callback) => {
				if(vm.leftTabsActive != 'Quick Settings'){
					callback();
				}else{
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~43 <%=rb.getString("ZhengXing")%>'))
					}else{
						if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=43){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~43 <%=rb.getString("ZhengXing")%>'))
						}
					} 
				}	
			},
			validateA1ThresholdRSRP = (rule,value,callback) => {
				if(value === '' || value === null || value === undefined){
					callback()
				}else{
					 if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=127){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~127 <%=rb.getString("ZhengXing")%>'))
					}
				}
				
			},
			validateA3OffsetRSRP = (rule,value,callback) => {
				
				if(value === '' || value === null || value === undefined){
					callback()
				}else {
					if(vm.isInteger(value)&& parseInt(value)>=-30 && parseInt(value)<=30){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> -30~30 <%=rb.getString("ZhengXing")%>'))
					}
				} 
				
			},
			validateB1ThresholdEUTRARSRP = (rule,value,callback) => {
				
				if(value === '' || value === null || value === undefined){
					callback()
				}else{
					if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=97){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~97 <%=rb.getString("ZhengXing")%>'))
					}
				} 
				
			},
			validateEnergySavingDelayTime = (rule,value,callback) => {
				if(vm.leftTabsActive != 'RAN'){
					callback();
				}else{
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~15 <%=rb.getString("ZhengXing")%>'))
					}else{
						if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=15){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~15 <%=rb.getString("ZhengXing")%>'))
						}
					} 
				}
			},
			validatePrbLowThreshold = (rule,value,callback) => {
				if(vm.leftTabsActive != 'RAN'){
					callback();
				}else{
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~100 <%=rb.getString("ZhengXing")%>'))
					}else{
						if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=100){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~100 <%=rb.getString("ZhengXing")%>'))
						}
					} 
				}
			},
			validatePrbReportPeriod = (rule,value,callback) => {
				if(vm.leftTabsActive != 'RAN'){
					callback();
				}else{
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~65535 <%=rb.getString("ZhengXing")%>'))
					}else{
						if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=65535){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~65535 <%=rb.getString("ZhengXing")%>'))
						}
					}
				}
			},
			validateRRCLowThreshold = (rule,value,callback) => {
				if(vm.leftTabsActive != 'RAN'){
					callback();
				}else{
					if(value =='' || value == null || value == undefined){
						callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~4294967295 <%=rb.getString("ZhengXing")%>'))
					}else{
						if(vm.isInteger(value)&& parseInt(value)>=0 && parseInt(value)<=4294967295){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> 0~4294967295 <%=rb.getString("ZhengXing")%>'))
						}
					} 
				}
			};
		return{
			ruleForm:{
				NrCellList:[],
				ranNrCellList:[],
				LF_List:[],
				LC_List:[],
				NF_List:[],
				NC_List:[],
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
				A1ThresholdRSRP:'',
				A2ThresholdRSRP:'',
				A3OffsetRSRP:'',
				A4ThresholdRSRP:'',
				A5Threshold1RSRP:'',
				A5Threshold2RSRP:'',
				B1ThresholdEUTRARSRP:'',
				B2Threshold1RSRP:'',
				B2Threshold2RSRP:'',
				EnergySavingType:'1',
				EnergySavingTime:'',
				EnergySavingDelayTime:'',
				PrbLowThreshold:'',
				PrbReportPeriod:'',
				RRCLowThreshold:'',
			},
			rules:{
				band:[
					{validator:validateBand,trigger:'blur'}
				],
				NRARFCNDL:[
					{validator:validateNRARFCNDL,trigger:'blur'}
				],
				NRARFCNUL:[
					{validator:validateNRARFCNUL,trigger:'blur'}
				],
				Pci:[
					{validator:validatePci,trigger:'blur'}
				],
				PowerModify:[
					{validator:validatePowerModify,trigger:'blur'}
				],
				A1ThresholdRSRP:[
					{validator:validateA1ThresholdRSRP,trigger:'blur'}
				],
				A2ThresholdRSRP:[
					{validator:validateA1ThresholdRSRP,trigger:'blur'}
				],
				A3OffsetRSRP:[
					{validator:validateA3OffsetRSRP,trigger:'blur'}
				],
				A4ThresholdRSRP:[
					{validator:validateA1ThresholdRSRP,trigger:'blur'}
				],
				A5Threshold1RSRP:[
					{validator:validateA1ThresholdRSRP,trigger:'blur'}
				],
				A5Threshold2RSRP:[
					{validator:validateA1ThresholdRSRP,trigger:'blur'}
				],
				B1ThresholdEUTRARSRP:[
					{validator:validateB1ThresholdEUTRARSRP,trigger:'blur'}
				],
				B2Threshold1RSRP:[
					{validator:validateA1ThresholdRSRP,trigger:'blur'}
				],
				B2Threshold2RSRP:[
					{validator:validateA1ThresholdRSRP,trigger:'blur'}
				],
				EnergySavingDelayTime:[
					{validator:validateEnergySavingDelayTime,trigger:'blur'}
				],
				PrbLowThreshold:[
					{validator:validatePrbLowThreshold,trigger:'blur'}
				],
				PrbReportPeriod:[
					{validator:validatePrbReportPeriod,trigger:'blur'}
				],
				RRCLowThreshold:[
					{validator:validateRRCLowThreshold,trigger:'blur'}
				],
			},
			castsList:{
				'BaiBNX':{
					//Quick Settings  NR Cell
					'5E33AA12D23A415C0AC4D32E54F26DD7':'NrCellList',
					'FB9C420EC739A1602B2B3364FBEA9DDB':'NrCell_idx',
					'76B2B37BF8DE3A8863CF620EBB3B416F':'NRCellIdentity',
					'5EC66911DB1993EC6D3400BCB2F8A1DB':'NrCellTAC',
					'ED7D77BBD7F452A1F38E0A9231D560DE':'NrCellRanac',
					'7E539755DF6E9ACA483677BC91FF796D':'NrCellPlmnList',
					//RAN  NR Cell
					'A74EE6BCC103DA11F291D53A8A036DF1':'ranNrCellList',
					'9E87BF4FC0E26A927B6E163182452CE5':'ranNrCell_idx',
					'E0B0EBFBF7EB1A37EFEE27DF7998EC27':'ranNRCellIdentity',
					'F28C0E5C0A52047194EA8E6C26117CCF':'ranNrCellTAC',
					'FB956FA566B67408C985F375395EDD0C':'ranNrCellRanac',
					'284C8D3878599989B93BB52FB3E90B74':'ranNrCellPlmnList',
					// LTE_N_FREQ_List
					'D1B98326923B05CBE2ADEFDD09805E8D':'LF_List',
					'F7FBBE2DC85A752B6A563FCF71E67B80':'LF_idx',
					'2ED730C5FEB0C5BA5C9DBC379CA379FC':'LF_CarrierFreq',
					'252C19505A4722C2FB773C4591867865':'LF_QOffset',
					'29D39A3B4A3A9B36522D531131D373F6':'LF_CellReselectionPriority',
					'4E6A8B3015D1D1B74A545AFA17E0A484':'LF_QRxLevMin',
					'A65C246D8EED3671314BC5D5769F6A3C':'LF_QQualMin',
					'1FEA6399BB11EC3ACE15892732E5FA3F':'LF_AllowdMeasBandWidth',
					'3B6F636FDE681F240795918E649F324B':'LF_PresAntennaPort1',
					'31A33F692EE887D58075A33BF6775D43':'LF_WideBandRsrqMeas',
					'C28D457F504D25594E74DE1A0B13A5B7':'LF_ThreshXHigh',
					'877646760C8A907DCF918943685B0512':'LF_ThreshXLow',
					'6D2659E2F351DF7E7B144EAE2A64438A':'LF_PMaxEUTRA',
					// LTE_N_CELL_List
					'3DD242FB1B116709422B0334288DDEE3':'LC_List',
					'09A8A21E8920F94547D8209CBD5D113A':'LC_idx',
					'177E71CA76689857F4AC84CAD84E180D':'LC_PLMNID',
					'6A97AB797ADF27AAF2E7B7ACC965C5E0':'LC_CID',
					'CDE26C9D62B6E233751B5C21EA7EA68C':'LC_EUTRACarrierARFCN',
					'D259AF920225D4F1FCCC19B532B68DD2':'LC_PhyCellID',
					'A8A8371CC8BE1C6D7A9958974E2DDBAA':'LC_QOffset',
					'A1A6E28BEE686B3D9626A01D6F0ED3C0':'LC_QRxLevMinOffsetCell',
					'C7314180D94B5928D8C18177B0490FE5':'LC_QQualMinOffsetCell',
					'528B25A9B81B05C01FB6D4C65E3ADEFD':'LC_CIO',
					'FDEB15A09559708AF190CF3B71AB9473':'LC_Blacklisted',
					'B71691A8062B539D8816D81AFC520722':'LC_TAC',
					'BC1EE04A7AB5A4ED854E89119EE29B4B':'LC_eNBType',
					'C03B0B499C2F813EC5DBA06B6FDB90E2':'LC_ECGI',
					'AAC51E9CA45011EC8B48FA163E7C14DA':'LC_noRemove',
					//NR_N_FREQ_List
					'4F9AF3433BFBE3C8F3EF70ED55BD520F':'NF_List',
					'88DC23206FC16673F0E1A53884003B04':'NF_idx',
					'ABDD20D3F9E630A23A96DEB2E08ED685':'NF_Enable',
					'8F2691F8D6945856F1F24F8186A63709':'NF_SSBFrequency',
					'0DD53B3407F86C6AFAA668D2C56B80C9':'NF_SubCarrierSpacing',
					'A25B955E0BC0EC9D1FD02DC4DA67B64D':'NF_SmtcPeriodicity',
					'28BEBA6C10B57F1A554DA7BF25D2952C':'NF_SmtcOffset',
					'A1B4337413DBEE40A30A17694C06886E':'NF_SmtcDuration',
					'3361BFBC1FBAB53443544134649A149E':'NF_SSBlocksConsolidationRsrp',
					'D0497D20352A53FCB60E0B231C336648':'NF_SSBlocksConsolidationRsrq',
					'C46725636F31E80340077F077773BFA6':'NF_SSBlocksConsolidationSinr',
					'715D1A2EB5E9F682CCBF8F17A386E3D9':'NF_NrofSSBlocksToAverage',
					'5D06326322B28571F74BD6B77A8E5985':'NF_RsrpOffsetSSB',
					'1DFF0E3F66DDF6E3584D75C01C8212CA':'NF_RsrqOffsetSSB',
					'023A875328D8C8BD956BAC3BAD0D29BF':'NF_SinrOffsetSSB',
					'DC37F47D522A4689AA632D689B37F634':'NF_RsrpOffsetCsiRs',
					'B50FCC5A37A428DD6919291537B9D3FE':'NF_RsrqOffsetCsiRs',
					'D6C30DB2EE56D5D917409609354817FA':'NF_SinrOffsetCsiRs',
					'812763322D75F060C543AB56937F6EB2':'NF_BitmapType',
					'0FE7FE282B26D9CEBB4E53A2B79BED17':'NF_Bitmap',
					'3D95048F8CDCBCAAA3E18393A323594C':'NF_DeriveSSBIndexFromCell',
					'37055B9F5196F8833524DD1AB3E35490':'NF_FreqBandIndicatorNR',
					'156B8226521185AAFB399D8C9533C2A9':'NF_OffsetToPointA',
					'E57642F59B8E000DB32CAB494C517070':'NF_SSBSubCarrierOffset',
					//NR_N_Cell_List
					'382727BD2CDE9D84B3B27958BF045838':'NC_List',
					'E549F0AF986E8B42A0A0AB11FCE8F168':'NC_idx',
					'320DCA846D88A1E63CDC00DD1EF9BA38':'NC_PLMNID',
					'70E21C1F409D7890B3F372EC914B8865':'NC_CID',
					'46A97ED71C952B227D17F604D96F99BE':'NC_NRARFCN',
					'F750F6BAFC43BC6AFEB91D447A860695':'NC_SSBFrequency',
					'F65B20F0468891E5E1A6776DB682E992':'NC_SSBSubcarrierSpacing',
					'C6606B3EA0B67AE20F2AD90FDA3020C8':'NC_PhyCellID',
					'FDDB9455AB39EA44AAD82F65EA8805C8':'NC_QOffset',
					'A17749E45F704CEACA3D09FAED97D7C8':'NC_QRxLevMinOffsetCell',
					'B81FE9B9C907EBB7446B234B2AABEE37':'NC_QQualMinOffsetCell',
					'D256C7803FE88962D10E4D5D291B34E3':'NC_CIO',
					'DBCF7D6BC1CD18BD40E35A91F3BB6D27':'NC_Blacklisted',
					'9F3327BBB5FC2642D42B3BCC19DC7120':'NC_TAC',
					'9B06E957A44D11EC9687FA163E754084':'NC_noRemove',
					'4FD556470C766058C17A5A2573D6F792':'NC_gnbIdLength',

					'CAD104C886D50B2888BC29560BE81D57':'band',
					'D0406B02CD2E9B1B35D6582FD48BB22F':'NRARFCNDL',
					'1C285932494845D8B11582F9B19FFDB0':'NRARFCNUL',
					'27346F9D65FAFCAD1F699D85036321D5':'Pci',
					'7A93E00326E39DD7EAD9938B71327AA0':'NumOfTxAtenna',
					'272780E14905A78C5DADCA9A67AFB306':'NumOfRxAtenna',
					'BD722B23713C6D577BE083DA7C809E54':'rftxEnable',
					'1A620455E42093A7FC74479DB795D9BA':'DL_SubCarrierSpacing',
					'EED2FAA8E36F483E436891F4B35AED9D':'DL_CarrierBandWidth',
					'74CF0C2668D3CAA54FED2AC929E49545':'UL_SubCarrierSpacing',
					'DCD73780C8BBB03649921A626DAB4F31':'UL_CarrierBandWidth',
					'F83FE627451478556AB70A59595E2482':'PowerModify',
					'D513FA8F99D7F393D3BBF9C3D9B1AC79':'A1ThresholdRSRP',
					'DC97B2E35CB5D6F37D045C3B8FBD01F7':'A2ThresholdRSRP',
					'F356CB1F0ACA22CB208EC7171A848840':'A3OffsetRSRP',
					'96BB02C3593CB3CE9A9616DA066929C7':'A4ThresholdRSRP',
					'1278986AE00E60968EFF9B0BF5714B56':'A5Threshold1RSRP',
					'0511517C827D7BF14B5011E20B57C429':'A5Threshold2RSRP',
					'1B771361C067721B4BECA30E363244B4':'B1ThresholdEUTRARSRP',
					'9A9C2A9D742E65AD4B6FCE106E3AFB36':'B2Threshold1RSRP',
					'7C21FADBD88D841D4D5784B097C21646':'B2Threshold2RSRP',
					'2EEE6C1A97720E6ED9F4B815FA3959AA':'EnergySavingType',
					'4F516238552A7D88093913473D8F979B':'EnergySavingTime',
					'8B792ED08163A9F26D44085BFF286D9E':'EnergySavingDelayTime',
					'57EBD6C819B1C0C3EEEE1EC077397526':'PrbLowThreshold',
					'CBD8CBF9A14FC4E54759642B1609D3FD':'PrbReportPeriod',
					'4EB1522AD8A610E6941FB28E5891356E':'RRCLowThreshold',
				},
				'BaiBNQ':{
					//Quick Settings  NR Cell
					'B779CD488ABB5033764865947EB012D2':'NrCellList',
					'218AF783286C52ACA1C655C51360E12A':'NrCell_idx',
					'C3BF1E75C209CA6A74D9E688ECBF72F1':'NRCellIdentity',
					'6B708E6294AFCE8EEF4158ABDED5339C':'NrCellTAC',
					'08966670FA93B4E29B484EABBFF1CD99':'NrCellRanac',
					'BD9DBC3DA704A5B8FD894D456A96961E':'NrCellPlmnList',
					//RAN  NR Cell
					'6AB45D901AE3B3D92BF84F4F69634267':'ranNrCellList',
					'C7986CEBEB2FA7F3C356097D40845880':'ranNrCell_idx',
					'469FA48CC794802BEE94415007175BE9':'ranNRCellIdentity',
					'60D701C0A02EFCBAE1D037997AD0B776':'ranNrCellTAC',
					'E4A85FF0B6CF13BA687D77F0BD1F46B3':'ranNrCellRanac',
					'8174D9290BA34529A6318E0D1C210F85':'ranNrCellPlmnList',
					// LTE_N_FREQ_List
					'071F2091F5643F2FD8D2F74A06AC67D9':'LF_List',
					'0B7E09E27B31FCFE50D50FB4647D412F':'LF_idx',
					'3F450C6676C2193BB41A4DE79C2106E4':'LF_CarrierFreq',
					'CB83C874ED855AC5F7C429AB32623E99':'LF_QOffset',
					'0A9A000D3C0A3911E287A96806E3EFCC':'LF_CellReselectionPriority',
					'A839DDBC5F0761D74816D9F40E2D7C66':'LF_QRxLevMin',
					'33780BB7F8668A36B64AC0F306BDB6E2':'LF_QQualMin',
					'B50D812FB0F214BAEBE06DB7053CD80E':'LF_AllowdMeasBandWidth',
					'3A966E40800FB710C13A1D2F53E75A39':'LF_PresAntennaPort1',
					'414F01C612A42E9C38227FB7C14DCC35':'LF_WideBandRsrqMeas',
					'907871D07DC5D23DD3509482E3D7CA71':'LF_ThreshXHigh',
					'197B0588AEE370A01C5463607EE0B5D8':'LF_ThreshXLow',
					'EBBAC1EB0DD126528C79B15B6C71A581':'LF_PMaxEUTRA',
					// LTE_N_CELL_List
					'7B7B254A37836B50929913604CBF0B95':'LC_List',
					'95DCEE4232482B822AF61B95E11BB328':'LC_idx',
					'3F56A1DA72243C633F5ABEFB054A65DC':'LC_PLMNID',
					'0247AE466667157050B9FE7A6CF65B16':'LC_CID',
					'7300F4B5706BDFB5D541F721389F2658':'LC_EUTRACarrierARFCN',
					'688790C6447761EA49977C0D34FE8B61':'LC_PhyCellID',
					'94249F79ECD5718D8B970D4C5D794E90':'LC_QOffset',
					'D05E321D1D15992C3E12A700F53B9DAC':'LC_QRxLevMinOffsetCell',
					'081827660A86565F9B66EEFC794B4766':'LC_QQualMinOffsetCell',
					'578A93F7F7A48A4A2CAD4BFA2648F445':'LC_CIO',
					'D7EBC619CA34CF28D968A1FCC17AC517':'LC_Blacklisted',
					'5B49AC1E07753DCBAA55EE98E971F38E':'LC_TAC',
					'11734AEA5EE16660C09513EAAD35761B':'LC_eNBType',
					'33D773E7E16DF509D353C6D772336F32':'LC_ECGI',
					'9D2C2F1F2B60F50149958E3A065F1783':'LC_noRemove',
					//NR_N_FREQ_List
					'FFBD6091E2BBDBF1E2ECFA13A4673E00':'NF_List',
					'6C3FE12704E731C385A9508B23318689':'NF_idx',
					'CB9B918DB2FD80F2F3FB22B46FFB7052':'NF_Enable',
					'086F3A4784A0E964C54819F109785AC5':'NF_SSBFrequency',
					'C261FD0E07702C2F1D969303A47C3110':'NF_SubCarrierSpacing',
					'F847642D0EF9CB6E15961D8C21200B10':'NF_SmtcPeriodicity',
					'69727C2AA02C6042793A5114CD81E666':'NF_SmtcOffset',
					'F8F28DDF2FBA8F775569AD611914666A':'NF_SmtcDuration',
					'3CB8EEA23EA29890788451EE4BFE76B6':'NF_SSBlocksConsolidationRsrp',
					'CA7E72D1393D24811E1A14FFD4098C11':'NF_SSBlocksConsolidationRsrq',
					'3FE378CFFE6E4A0167928F3AEA591161':'NF_SSBlocksConsolidationSinr',
					'E97EBDF25595A3030C66CC36D0D62FB9':'NF_NrofSSBlocksToAverage',
					'1C93CE499C64AAC73E125E3387EAF7CB':'NF_RsrpOffsetSSB',
					'2BF88B8AC753A9801DC6176B30BA715B':'NF_RsrqOffsetSSB',
					'38F9862E2A58B9D1EBE9F460B76117FA':'NF_SinrOffsetSSB',
					'BC1CFB2CF45E7FD6CD56220E898D27AB':'NF_RsrpOffsetCsiRs',
					'AD1B752015A69BF98ECB628247E2B714':'NF_RsrqOffsetCsiRs',
					'407BC693EFC35DDE21709F7DE9CFF423':'NF_SinrOffsetCsiRs',
					'0DFBE5074C9E8372DC5DC7EB6935855B':'NF_BitmapType',
					'266976D0E88FC340898915D7B9EB5C47':'NF_Bitmap',
					'DC01FE89B78A1A1F2AE95CF943652904':'NF_DeriveSSBIndexFromCell',
					'5C92E58894F45CEA20B891C2EC54DEBA':'NF_FreqBandIndicatorNR',
					'CA6AB0188BA7B9853E3486496D7D6EFE':'NF_OffsetToPointA',
					'BCFAD98AB5F77886F429D183CEBC9BB2':'NF_SSBSubCarrierOffset',
					//NR_N_Cell_List
					'3993E07C199709BD90BB59C197954ED8':'NC_List',
					'6F6158A4AC9A3DAADA272ACEE8965A3E':'NC_idx',
					'0ABC001C934D18B11F43B55DEDCAEF88':'NC_PLMNID',
					'AE87CB0E7C6101FFFAACFB7C728A1AD2':'NC_CID',
					'83DAD05DB20F45E9800B6090512CFC21':'NC_NRARFCN',
					'F5359EE38287AF5BE2C6B0F24DCBC07B':'NC_SSBFrequency',
					'5715660FCF14D83B1DFEB3AA7291B47E':'NC_SSBSubcarrierSpacing',
					'6B9F12A98D32686CBFC6233991642CEE':'NC_PhyCellID',
					'AA82E5E83A37BC6A6076C075AD5A9EFA':'NC_QOffset',
					'85F4FB44757C5FCC51E6682225C07843':'NC_QRxLevMinOffsetCell',
					'6B1D9444974757EF55B0ABEDD864E3B4':'NC_QQualMinOffsetCell',
					'C354349B6BF8E3111E576B1A22EB7DD1':'NC_CIO',
					'405ABE0B20840AA4787060C6DDFADC25':'NC_Blacklisted',
					'865122707C9C7FE063CC47BA3D779C6E':'NC_TAC',
					'AAD4C0190EDBD517FA29DD625EDEA0AD':'NC_noRemove',
					

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
					'6D4AD0057BFFAE8C3018230269E079EF':'A1ThresholdRSRP',
					'30DC49A7AAC3274FBD8BCD7C8350B045':'A2ThresholdRSRP',
					'0020AB2CEFCF76D53BAB430BFCB172D4':'A3OffsetRSRP',
					'AD064861427084C5FDF12351BCB335E9':'A4ThresholdRSRP',
					'674EB6E4CF8DF137D836558A6647F76F':'A5Threshold1RSRP',
					'091D4D073070658CE609410DA6E45A69':'A5Threshold2RSRP',
					'EA63904893494BD2450A4F39C9928359':'B1ThresholdEUTRARSRP',
					'7AE68DD39D2612326B9C5F27683868A2':'B2Threshold1RSRP',
					'390758B58321FE97762DF85F3CF570CD':'B2Threshold2RSRP',
					'12F21ADB0B26C5342F17F42CF99F7880':'EnergySavingType',
					'D2AC053D212F613CE192B5FAF835A29A':'EnergySavingTime',
					'019A448406D26009641A1986F36D5F0E':'EnergySavingDelayTime',
					'EF90F6A721D47D653562F9EADEC603EE':'PrbLowThreshold',
					'213013ED1109189BFE720764FC783EFB':'PrbReportPeriod',
					'CDC098800D7EEB1D9A4DB83B46C93796':'RRCLowThreshold',
				}
			},
			casts:{},
			EnergySavingTimeList:["00:00:00","23:59:59"],
			leftTreeData:[],
			leftTabsActive:'',
			quickSettingsCollapse:'quickSettings',
			RAN_Collapse:['neighborFreq','mobility','EnergySavingSetting','ranPlmn'],
			stationCollapse:'syncSettings',
			
			showNrCellAdd:false,
			operType:'',
			smallCellCode:'',
			codeList:[],
			productType:'',
		}
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
	},
	methods:{
		init(code,cellIndex,productType){
			var vm = this;
			vm.smallCellCode = code;
			vm.cellIndex = cellIndex;
			vm.productType = productType;
			vm.casts = vm.castsList[productType];
			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getLeftTreeData(code);
			$('#cell4Box').removeClass('loading');
		},
		getLeftTreeData(code){
			var vm = this;
				params = {
					title:'Cell',
					groupId:'21000',
					smallCellCode:code
				}
			axios.post("${ctx}/cell/quicksettings/getSettingGroupTree.action",stringify(params)).then(function(res){
				var data = res.data;
				
				vm.leftTreeData = data.filter((items)=>{
					return items.id != '9999'
				})
				vm.leftTabsActive = vm.leftTreeData[0].code;
				var id = vm.leftTreeData[0].id;
				vm.getParamNodeTreeAndData(code,vm.cellIndex,id);
			})
		},
		getParamNodeTreeAndData(code,cellIndex,id) {
			var vm = this,
				url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
				params = {
					id: id,
					cellIndex:cellIndex,
					smallCellCode: code
				};

			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				
				if(data && Array.isArray(data)) {
					data.map(function(item){
						item.groups.map(function(group){
							
							if(group.list &&  Array.isArray(group.list)){
								group.list.map(function(m){
									if(m.type == 'list'){
										vm.initTable(m.url,m.label);
									}else{
										// 执行赋值
										vm.setValue(m);
									}
								});
							}
							if(group.groups && Array.isArray(group.groups)){
								group.groups.map((lastGroup)=>{
									lastGroup.list.map(function(m){
										if(m.type == 'list'){
											vm.initTable(m.url,m.label);
										}else{
											// 执行赋值
											vm.setValue(m);
										}
									});
								})
							}
						});
					});
					initForm(vm.$refs.ruleForm);
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
					vm.ruleForm[key] = value;
					if(key == 'EnergySavingTime'){
						if(vm.ruleForm[key] == ''){
							vm.EnergySavingTimeList = [];
						}else{
							vm.EnergySavingTimeList = vm.ruleForm[key].split('-');
						}
						
					}
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					cellIndex:vm.cellIndex,
					smallCellCode : vm.smallCellCode
			}
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							obj[vm.casts[key]] = item[key]
						}
						if(type == 'NR Cell List'){
							vm.ruleForm.NrCellList.push(obj);
						}else if(type == 'RAN NR Cell List'){
							vm.ruleForm.ranNrCellList.push(obj);
						}else if(type == 'LTE N-FREQ List'){
							vm.ruleForm.LF_List.push(obj);
						}else if(type == 'LTE N-CELL List'){
							vm.ruleForm.LC_List.push(obj);
						}else if(type == 'N FREQ List'){
							vm.ruleForm.NF_List.push(obj);
						}else if(type == 'N Cell List'){
							vm.ruleForm.NC_List.push(obj);
						}
					})
					initForm(vm.$refs.ruleForm);
				}
			})
			
		},
		// NR Cell添加事件
		addNrCell(plmnType){
			var vm = this;
			eventBus.$emit('open-addNrCell','add',{},plmnType);
		},
		NRCellEdit(row,plmnType){
			var vm = this;
			eventBus.$emit('open-addNrCell','edit',row,plmnType);
		},
		NRCellDel(row){
			var vm =this;
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm.NrCellList.map(function(item,index){
					if(item.NrCell_idx == row.NrCell_idx){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm.NrCellList,index,params);
						}
						
					}
				})
				if(delFlag){
					vm.ruleForm.NrCellList = vm.ruleForm.NrCellList.filter((items)=>{
						return items.NrCell_idx != row.NrCell_idx
					})
				}
			})
		},
		addTableClick(row,optType,addType){
			var vm = this;
			eventBus.$emit('open-addTableList',row,optType,addType);
		},
		delTableClick(row,delTable){
			var vm = this;

			if(delTable == 'LF'){
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var delFlag=false;
					vm.ruleForm.LF_List.map(function(item,index){
						if(item.LF_idx == row.LF_idx){
							if(item.operateType == 'add'){
								delFlag = true;
							}else{
								var params = item;
								params.operateType = 'remove';
								vm.$set(vm.ruleForm.LF_List,index,params);
							}
						}
					})
					if(delFlag){
						vm.ruleForm.LF_List = vm.ruleForm.LF_List.filter((items)=>{
							return items.LF_idx != row.LF_idx
						})
					}
				})
			}else if(delTable == 'LC'){
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var delFlag=false;
					vm.ruleForm.LC_List.map(function(item,index){
						if(item.LC_idx == row.LC_idx){
							if(item.operateType == 'add'){
								delFlag = true;
							}else{
								var params = item;
								params.operateType = 'remove';
								vm.$set(vm.ruleForm.LC_List,index,params);
							}
						}
					})
					if(delFlag){
						vm.ruleForm.LC_List = vm.ruleForm.LC_List.filter((items)=>{
							return items.LC_idx != row.LC_idx
						})
					}
				})
			}else if(delTable == 'NF'){
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var delFlag=false;
					vm.ruleForm.NF_List.map(function(item,index){
						if(item.NF_idx == row.NF_idx){
							if(item.operateType == 'add'){
								delFlag = true;
							}else{
								var params = item;
								params.operateType = 'remove';
								vm.$set(vm.ruleForm.NF_List,index,params);
							}
						}
					})
					if(delFlag){
						vm.ruleForm.NF_List = vm.ruleForm.NF_List.filter((items)=>{
							return items.NF_idx != row.NF_idx
						})
					}
				})
			}else if(delTable == 'NC'){
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var delFlag=false;
					vm.ruleForm.NC_List.map(function(item,index){
						if(item.NC_idx == row.NC_idx){
							if(item.operateType == 'add'){
								delFlag = true;
							}else{
								var params = item;
								params.operateType = 'remove';
								vm.$set(vm.ruleForm.NC_List,index,params);
							}
						}
					})
					if(delFlag){
						vm.ruleForm.NC_List = vm.ruleForm.NC_List.filter((items)=>{
							return items.NC_idx != row.NC_idx
						})
					}
				})
			}
		},
		// 左侧tab点击事件
		leftTabsClick(code,id){
			var vm = this;
			var params = {},
				isChanged = isFormChanged(vm.$refs.ruleForm);

			if(isChanged){
				vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					vm.resetFormData();
					vm.leftTabsActive = code;
					vm.getParamNodeTreeAndData(vm.smallCellCode,vm.cellIndex,id);
				}).catch(() => {
					
				})
			}else{
				vm.resetFormData();
				vm.leftTabsActive = code;
				vm.getParamNodeTreeAndData(vm.smallCellCode,vm.cellIndex,id);
			}
			
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					NrCellList:[],
					ranNrCellList:[],
					LF_List:[],
					LC_List:[],
					NF_List:[],
					NC_List:[],
					band: '',
					NRARFCNDL:'',
					NRARFCNUL:'',
					Pci:'',
					NumOfTxAtenna:'',
					NumOfRxAtenna:'',
					rftxEnable:'0',
					DL_SubCarrierSpacing:'',
					DL_CarrierBandWidth:'',
					UL_SubCarrierSpacing:'',
					UL_CarrierBandWidth:'',
					PowerModify:'',
					A1ThresholdRSRP:'',
					A2ThresholdRSRP:'',
					A3OffsetRSRP:'',
					A4ThresholdRSRP:'',
					A5Threshold1RSRP:'',
					A5Threshold2RSRP:'',
					B1ThresholdEUTRARSRP:'',
					B2Threshold1RSRP:'',
					B2Threshold2RSRP:'',
					EnergySavingType:'',
					EnergySavingTime:["00:00:00","23:59:59"],
					EnergySavingDelayTime:'',
					PrbLowThreshold:'',
					PrbReportPeriod:'',
					RRCLowThreshold:'',
				};
			Object.assign(vm.ruleForm,params);
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
		cell4Submit(){
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
								if(vm.leftTabsActive == 'Quick Settings' && items.operateType == 'add'){
									isSync = true;
								}
								editList.push(items)
							}
						})

						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								objs[listKey] = items[listVal]
							}
							objs.cellIndex = vm.cellIndex;
							subList.push(objs)
						})
						
						params[key] = subList;
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						var editData={
							cellIndex:vm.cellIndex,
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
					$('#settingPage').addClass('loading');
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							if(isSync){
								vm.syncParams();
							}
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							eventBus.$emit('close-gnb-settingPage');
						}else{
							vm.$message.error(data["message"])
						}
						$('#settingPage').removeClass('loading');
					})
				}
			});
			
		},
		// 同步
		syncParams(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				codes = {
					'BaiBNX':'5E33AA12D23A415C0AC4D32E54F26DD7',
					'BaiBNQ':'B779CD488ABB5033764865947EB012D2',
				},
				params={
					smallCellCode: vm.smallCellCode,
					paramId:codes[vm.productType],
					cellIndex:vm.cellIndex
				};
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
			})
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
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
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		EnergySavingTimeListChange(time){
			var vm = this;
			if(time == null || time == undefined || time == ''){
				vm.ruleForm.EnergySavingTime = ''
			}else{
				vm.ruleForm.EnergySavingTime = time.join('-');
			}
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
		}
	},
	mounted(){
		eventBus.$off('cellInfo-init').$on('cellInfo-init',this.init);
		eventBus.$off('cell4-submit').$on('cell4-submit',this.cell4Submit);
	}
	
})
</script>